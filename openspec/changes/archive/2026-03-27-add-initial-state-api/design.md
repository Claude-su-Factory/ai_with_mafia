## Context

현재 `hub.ServeWS()`는 WS 연결 직후 아래 메시지를 전송한다.

```json
{ "type": "room_state", "payload": { "room": "roomID" } }
```

roomID만 포함되어 있어 클라이언트가 현재 상태를 알 수 없다. 또한 `PhaseManager`는 페이즈 시작 시각을 기록하지 않아 타이머 잔여 시간을 계산할 수 없으며, `hub`의 `GameManager` 인터페이스에 게임 상태 조회 메서드가 없어 hub이 game 상태에 접근하는 경로 자체가 없다.

## Goals / Non-Goals

**Goals:**
- WS 연결/재연결 직후 클라이언트가 UI를 완전히 복원할 수 있는 `initial_state` 이벤트 제공
- 페이즈별 타이머 잔여 시간을 서버가 계산하여 포함
- 플레이어 본인의 역할(role)을 초기 상태에 포함 (본인에게만)

**Non-Goals:**
- HTTP polling 방식의 상태 조회 API 추가
- 게임 이벤트 히스토리 재전송 (재연결 시 이전 채팅 내역 복원 등)
- 죽은 플레이어의 역할 공개

## Decisions

### 1. 타이머 잔여 시간: GameState에서 계산

`GameState` 구조체에 `phaseStartedAt time.Time`과 `phaseDuration time.Duration` 필드를 추가한다. 기존 `state.mu (sync.RWMutex)`가 이미 GameState의 모든 필드를 보호하고 있으므로, 두 필드도 동일한 뮤텍스로 일관되게 보호된다. `PhaseManager`의 각 `Run*` 메서드에서 `state.mu.Lock()` 범위 내에 기록하고, `State()`의 `state.mu.RLock()` 범위 내에서 `remaining = phaseDuration - time.Since(phaseStartedAt)`로 계산하여 `TimerRemainingSeconds`를 반환한다.

잔여 시간이 음수(페이즈 종료 후 짧은 지연)면 0으로 클리핑한다.

### 2. hub → gameManager 상태 조회: 인터페이스 확장

`hub.go`의 `GameManager` 인터페이스에 `GetSnapshot(roomID string) *GameSnapshot` 추가. `GameSnapshot`은 hub 패키지에서 정의한 별도 구조체로, hub이 직접 사용할 필드만 담는다.

```go
// hub.go 내 정의
type GameSnapshot struct {
    Phase              string
    Round              int
    TimerRemainingSec  int
    AlivePlayerIDs     []string
    Votes              map[string]string // voterID → targetID, nil if not vote phase
}
```

`main.go`의 `gameManager`가 이를 구현한다. `activeGames[roomID]`가 없으면 `nil` 반환 (게임 미진행 상태).

### 3. initial_state 페이로드 설계

```json
{
  "type": "initial_state",
  "payload": {
    "room": {
      "id": "...",
      "name": "...",
      "status": "playing",
      "host_id": "...",
      "visibility": "private",
      "join_code": "ABC123",
      "players": [
        { "id": "...", "name": "...", "is_alive": true, "is_ai": false }
      ]
    },
    "game": {
      "phase": "day_vote",
      "round": 2,
      "timer_remaining_sec": 47,
      "alive_player_ids": ["p1", "p2", "p3"],
      "votes": { "p1": "p3" }
    },
    "my_role": "mafia"
  }
}
```

게임이 진행 중이 아니면 `game` 필드는 `null`. `my_role`은 게임 시작 전이면 `""`.

`votes`는 day_vote 페이즈가 아닐 때 빈 map으로 전송한다 (null 대신 빈 map이 프론트 처리에 편리).

### 4. 게임 시작 시 역할 전달: role_assigned 이벤트

`initial_state`는 재연결 시 역할을 복원한다. 그런데 대기실에서 이미 연결된 채로 게임이 시작되는 경우, 클라이언트는 `phase_change` 이벤트는 받지만 역할을 받지 못한다.

`main.go`의 `updateRoleFunc`는 현재 `hub.UpdateClientRole()`만 호출해 hub 내부 Client.Role만 갱신한다. WS 메시지를 보내지 않는다.

해결: hub에 `SendToPlayer(roomID, playerID string, payload any)` 메서드를 추가하고, `updateRoleFunc`에서 `UpdateClientRole` 호출 후 해당 플레이어에게 `role_assigned` 이벤트를 개별 전송한다.

```json
{ "type": "role_assigned", "payload": { "role": "mafia" } }
```

이 이벤트는 broadcast가 아닌 단일 플레이어 전송이므로 역할 정보 유출이 없다.

### 5. initial_state의 my_role 출처: room.PlayerByID

`hub.ServeWS`에서 이미 `player := room.PlayerByID(playerID)`를 호출해 `client.Role`을 설정한다. 이 값을 그대로 `my_role`에 사용한다. 게임 시작 전(role 미배정)이면 빈 문자열.

## Risks / Trade-offs

- **시간 drift**: 서버 시간 기준으로 계산된 잔여 시간이 클라이언트 렌더링 시점에 약간 오차가 생긴다. 게임 UX 수준에서 허용 범위 내이므로 무시.
- **race condition — phaseStartedAt vs State() 호출**: `phaseStartedAt` 기록과 `State()` 호출 사이에 페이즈 전환이 일어날 수 있다. `phaseStartedAt`/`phaseDuration`을 `GameState` 안에 두고 기존 `state.mu`로 보호하므로 자연스럽게 해결된다.
- **기존 클라이언트 호환**: `room_state` 타입이 `initial_state`로 바뀐다. 현재 프론트가 없으므로 breaking 변경 무해.
