## Context

백엔드 초기 구현이 완료됐으나 코드 리뷰에서 게임이 동작하지 않는 Critical 버그 4개와 동시성 잠재 버그 5개가 발견됐다. 각 버그는 독립적이며 서로 영향을 주지 않으므로 개별적으로 수정한다.

## Goals / Non-Goals

**Goals:**
- Critical 버그 4개 수정으로 게임이 실제로 동작하도록 만들기
- 잠재적 동시성 버그 5개 수정으로 race condition 제거
- 기존 API 인터페이스 및 동작 명세 유지

**Non-Goals:**
- 새로운 기능 추가
- 성능 최적화
- 아키텍처 리팩토링
- 테스트 코드 작성

## Decisions

### 1. AI 플레이어 room 추가 위치

`gameManager.start()`에서 `mod.NewGame(room)` 호출 전에 AI 플레이어를 생성하여 `room.AddPlayer()`로 추가한다. `ai.Manager.SpawnAgents()`는 그 다음에 호출하여 이미 players가 채워진 room을 기반으로 에이전트를 생성한다.

AI 플레이어 수는 `room.MaxHumans`와 고정 총원(6명)으로 계산한다. 페르소나는 `personaPool.Assign()`으로 할당하고, 역할은 `AssignRoles()`가 전체 players 슬라이스를 기반으로 처리하므로 별도 처리 불필요.

### 2. NotifyEvent 구현 방식

`ws.Hub`가 WebSocket 메시지를 파싱해서 `gameManager.NotifyEvent()`를 호출하는 현재 구조를 유지한다. `NotifyEvent`는 `activeGames` map에서 해당 roomID의 game을 찾아 `game.HandleAction()`으로 라우팅한다.

`activeGame` 구조체에 `game entity.Game` 필드를 추가해야 한다 (현재는 `interface{ HandleAction }` 익명 인터페이스만 있음). `entity.Game` 인터페이스를 직접 저장한다.

### 3. AI vote/night 콜백 구현

`main.go`의 `SetCallbacks`에서 vote 콜백은 `gameManager.activeGames[roomID].game.HandleAction(playerID, entity.Action{Type: "vote", ...})`을 호출하도록 구현한다. night(kill) 콜백도 동일 패턴으로 `"kill"` 타입으로 라우팅한다.

콜백이 `gameManager`를 클로저로 캡처하므로 `gameManager`에 mutex를 추가한 후 콜백 내에서 lock을 취득해야 한다.

### 4. Mafia decideKill 추가

`agent.go`의 `onPhaseChange`에서 `phase == entity.PhaseNight && role == entity.RoleMafia` 조건으로 `decideKill()` 함수를 호출한다. `decideKill`은 기존 `decideInvestigate`와 동일한 패턴으로 구현한다: `model_reasoning` 사용, `__kill__:<target_id>` 접두사 파싱.

### 5. JoinByCode deadlock 수정

현재 패턴:
```go
s.mu.RLock()
defer s.mu.RUnlock()  // defer 등록
...
s.mu.RUnlock()        // 수동 unlock
s.mu.RLock()          // 수동 relock → defer와 불균형 가능
```

수정: defer를 제거하고 명시적 unlock/lock 쌍만 사용하거나, 전체 로직을 두 단계로 분리 (lookup → separate write lock section). 가장 단순한 수정은 defer를 제거하고 각 return path에서 명시적으로 unlock을 호출하는 것이다.

### 6. room.Players mutex 보호

`mafia/game.go`에서 `room.Players`를 직접 슬라이스 복사할 때 `room.mu.RLock()`/`RUnlock()`으로 감싼다. `Room` entity의 `mu` 필드가 public이 아니므로 `GetPlayers()` 메서드를 `Room`에 추가하여 안전한 접근을 제공한다.

### 7. gameManager.activeGames mutex

`gameManager`에 `sync.Mutex`를 추가한다. `StartGame`, `RestartGame`, `start`, `NotifyEvent` 진입 시 lock 취득, 반환 전 unlock. 단, `game.Start(gameCtx)`는 goroutine 안에서 실행되므로 lock 내부에서 호출하지 않는다.

### 8. sender_name 추가

`mafia/game.go`의 `HandleAction`에서 chat 처리 시 `player.Name`을 payload에 포함:
```go
"sender_name": player.Name,
```

### 9. Hub context 수정

`ws.Hub`에 서버 context를 저장하는 필드를 추가하고, `NewHub` 생성 시 주입한다. `StartGame`/`RestartGame`에서 `context.Background()` 대신 저장된 context를 사용한다.

### 10. 게임 이벤트를 WS 클라이언트에 브로드캐스트

`gameManager.start()`의 이벤트 포워딩 goroutine은 현재 `ai.BroadcastEvent()`만 호출한다. 결과적으로 `game.Subscribe()`에서 나오는 모든 게임 엔진 이벤트(페이즈 변경, kill, 투표 결과, 게임 오버)가 인간 WS 클라이언트에 **전달되지 않는다**.

`gameManager`에 `gameEventFunc func(roomID string, event entity.GameEvent)` 콜백 필드를 추가한다. `main.go`에서 이 콜백을 설정하여 `gameHub.Broadcast()`로 라우팅한다. `gameManager`가 직접 `gameHub`를 참조하면 순환 의존이 발생하므로 콜백 패턴을 유지한다.

`MafiaOnly` 필드가 `true`인 이벤트는 마피아 채널이므로 `gameHub.Broadcast(..., true)`로 전달한다.

### 11. WS Client.Role 동기화

`ServeWS`에서 클라이언트 등록 시 `player.Role`을 저장하는데, 이 시점은 게임 시작 전이라 Role이 zero 값이다. `AssignRoles()`는 `newGame()` 내부에서 실행되므로 그 이후에 Role이 채워진다.

`gameManager.start()`에서 `mod.NewGame(room)` 호출 직후(역할이 배정된 이후), `room.Players`를 순회하며 `gameHub.UpdateClientRole(roomID, playerID, role)`을 호출한다. 이를 위해 `gameManager`에 `updateRoleFunc func(roomID, playerID string, role entity.Role)` 콜백을 추가한다. `main.go`에서 `hub.UpdateClientRole`로 연결한다.

### 12. gameManager에 personaPool 주입

task 2.1에서 AI 플레이어 생성 시 `personaPool.Assign()`이 필요하다. 현재 `gameManager`에는 `personaPool`이 없다. `newGameManager()` 생성자에 `*ai.PersonaPool` 파라미터를 추가하고 `main.go`에서 주입한다.

## Risks / Trade-offs

- **activeGames map에 전체 lock**: 게임 시작/종료 시 잠깐 lock을 잡으므로 성능 영향 미미. 동시에 수백 개의 방이 생성/종료되는 경우는 현재 스코프 밖.
- **GetPlayers() 메서드 추가**: 기존 코드가 `room.Players`를 직접 참조하는 다른 위치가 있을 수 있으므로 전체 검색 필요.
- **콜백 증가**: `gameManager`에 콜백이 3개(`gameEventFunc`, `updateRoleFunc` + 기존 AI 콜백) 생겨 `main.go` 와이어링이 복잡해진다. 현재 스코프에서는 허용 가능한 수준이다.

## Open Questions

없음. 모든 수정은 명확히 정의되어 있다.
