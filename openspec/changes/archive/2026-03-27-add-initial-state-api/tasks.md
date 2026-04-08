## 1. GameState 타이머 필드 추가

- [x] 1.1 `internal/domain/entity/game.go`의 `GameState`에 `TimerRemainingSeconds int` 필드 추가

## 2. PhaseManager 타이머 추적

- [x] 2.1 `internal/games/mafia/phases.go`의 `GameState` 구조체에 `phaseStartedAt time.Time`, `phaseDuration time.Duration` 필드 추가 — `PhaseManager`가 아닌 `GameState` 안에 두어 기존 `state.mu`(RWMutex)로 일관되게 보호
- [x] 2.2 `RunDayDiscussion` 시작 시 `state.mu.Lock()` 내에서 `state.phaseStartedAt = time.Now()`, `state.phaseDuration = timers.DayDiscussion` 기록
- [x] 2.3 `RunDayVote` 시작 시 `state.mu.Lock()` 내에서 `state.phaseStartedAt = time.Now()`, `state.phaseDuration = timers.DayVote` 기록
- [x] 2.4 `RunNight` 시작 시 `state.mu.Lock()` 내에서 `state.phaseStartedAt = time.Now()`, `state.phaseDuration = timers.Night` 기록
- [x] 2.5 `State()` 메서드 내 `state.mu.RLock()` 범위에서 `remaining = state.phaseDuration - time.Since(state.phaseStartedAt)` 계산, 음수면 0으로 클리핑, `TimerRemainingSeconds`에 설정

## 3. GameManager 인터페이스 + GameSnapshot 타입 + SendToPlayer

- [x] 3.1 `internal/platform/ws/hub.go`에 `GameSnapshot` 구조체 정의 (`Phase string`, `Round int`, `TimerRemainingSec int`, `AlivePlayerIDs []string`, `Votes map[string]string`)
- [x] 3.2 `GameManager` 인터페이스에 `GetSnapshot(roomID string) *GameSnapshot` 메서드 추가
- [x] 3.3 `Hub`에 `SendToPlayer(roomID, playerID string, payload any)` 메서드 추가 — 해당 playerID의 send 채널에만 메시지 전달, 채널 full 시 `logger.Warn()`

## 4. gameManager GetSnapshot 구현 + role_assigned 이벤트 연결

- [x] 4.1 `cmd/server/main.go`의 `gameManager`에 `GetSnapshot(roomID string) *ws.GameSnapshot` 구현 — `activeGames[roomID]` 없으면 nil 반환, 있으면 `game.State()`로 `GameSnapshot` 변환하여 반환
- [x] 4.2 `main.go`의 `updateRoleFunc` 콜백에서 `hub.UpdateClientRole()` 호출 직후 `gameHub.SendToPlayer(roomID, playerID, map[string]any{"type": "role_assigned", "payload": map[string]any{"role": string(role)}})` 호출

## 5. hub.ServeWS initial_state 전송

- [x] 5.1 `hub.ServeWS`에서 기존 `room_state` 메시지 제거
- [x] 5.2 `h.gameManager.GetSnapshot(roomID)` 호출하여 게임 스냅샷 획득
- [x] 5.3 `initial_state` 타입 이벤트 구성 — payload에 `room`(id/name/status/host_id/visibility/join_code/players[]), `game`(nil 또는 스냅샷), `my_role`(player.Role) 포함
- [x] 5.4 `initial_state` 이벤트를 JSON 직렬화하여 WS에 전송, 실패 시 `logger.Warn()` 후 return
