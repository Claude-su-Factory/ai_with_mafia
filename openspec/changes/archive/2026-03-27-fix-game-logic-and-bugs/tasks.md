## 1. 동시성 안전성 기반 작업

- [x] 1.1 `entity/room.go`에 `GetPlayers() []*Player` 메서드 추가 (RLock 보호)
- [x] 1.2 `gameManager`에 `sync.Mutex` 필드 추가 및 `activeGame` 구조체에 `game entity.Game` 필드 추가
- [x] 1.3 `gameManager.StartGame`, `RestartGame`, `start` 진입부에 mutex lock/unlock 추가
- [x] 1.4 `platform/room.go`의 `JoinByCode` — defer 제거, 각 return path에 명시적 `RUnlock` 추가

## 2. 의존성 주입 — gameManager 확장

- [x] 2.1 `gameManager` 구조체에 `personaPool *ai.PersonaPool` 필드 추가, `newGameManager()` 파라미터에 추가
- [x] 2.2 `gameManager` 구조체에 `gameEventFunc func(roomID string, event entity.GameEvent)` 콜백 필드 추가
- [x] 2.3 `gameManager` 구조체에 `updateRoleFunc func(roomID, playerID string, role entity.Role)` 콜백 필드 추가
- [x] 2.4 `main.go`에서 `newGameManager(registry, aiManager, personaPool, logger)` 호출로 변경
- [x] 2.5 `main.go`에서 `gameManager.gameEventFunc` 설정: `gameHub.Broadcast(roomID, event_dto, event.MafiaOnly)`로 라우팅
- [x] 2.6 `main.go`에서 `gameManager.updateRoleFunc` 설정: `gameHub.UpdateClientRole(roomID, playerID, role)`로 연결

## 3. Critical 버그 수정 — 게임 동작

- [x] 3.1 `gameManager.start()`에서 `mod.NewGame()` 전에 AI 플레이어 생성 및 `room.AddPlayer()` 호출 (`personaPool.Assign` 사용)
- [x] 3.2 `gameManager.start()`에서 `mod.NewGame()` 직후 `room.GetPlayers()`로 순회하며 `updateRoleFunc` 호출
- [x] 3.3 `gameManager.start()`의 이벤트 포워딩 goroutine에서 `ai.BroadcastEvent` 다음에 `gameEventFunc` 호출 추가
- [x] 3.4 `gameManager.NotifyEvent()`에 activeGames 조회 후 `game.HandleAction()` 라우팅 구현 (payload에서 action 타입/필드 추출)
- [x] 3.5 `main.go` vote 콜백에서 `gameManager` mutex 취득 후 `game.HandleAction(playerID, entity.Action{Type:"vote", Payload:{"target_id":targetID}})` 호출
- [x] 3.6 `main.go` night(kill) 콜백에서 `gameManager` mutex 취득 후 `game.HandleAction(playerID, entity.Action{Type:actionType, Payload:{"target_id":targetID}})` 호출

## 4. AI 마피아 Kill 로직

- [x] 4.1 `internal/ai/agent.go`에 `decideKill(ctx)` 함수 구현 (`model_reasoning` 사용, `__kill__:<target_id>` 포맷 출력)
- [x] 4.2 `agent.go`의 `onPhaseChange`에서 `PhaseNight && role == RoleMafia` 조건으로 `decideKill` 호출 추가

## 5. 이벤트 Payload 수정

- [x] 5.1 `mafia/game.go`의 chat `HandleAction`에서 payload에 `sender_name: player.Name` 추가
- [x] 5.2 `mafia/game.go`의 `newGame()`에서 `room.Players` 직접 접근을 `room.GetPlayers()` 호출로 교체

## 6. Server Context 전파

- [x] 6.1 `ws.Hub`에 `serverCtx context.Context` 필드 추가 및 `NewHub` 파라미터에 context 추가
- [x] 6.2 `hub.StartGame`/`RestartGame`에서 `context.Background()` 대신 `h.serverCtx` 사용
- [x] 6.3 `main.go`의 `ws.NewHub(...)` 호출에 server context(`ctx`) 전달
