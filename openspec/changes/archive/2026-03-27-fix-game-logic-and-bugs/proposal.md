## Why

코드 리뷰에서 발견된 Critical 버그 4개로 인해 게임이 현재 전혀 동작하지 않으며, 동시성 관련 잠재적 버그 5개가 추가로 존재한다. 서비스를 실제로 사용하기 전에 반드시 수정해야 한다.

## What Changes

- `gameManager.start()`에서 게임 시작 전 AI 플레이어를 `room.Players`에 추가 (`gameManager`에 `personaPool` 참조 주입 필요)
- `gameManager.NotifyEvent()`를 실제 구현으로 채워 사람 플레이어의 채팅/투표/밤 행동이 게임 로직에 전달되도록 수정
- `agent.go`의 `onPhaseChange`에 마피아 AI의 `decideKill` 로직 추가
- `main.go`의 AI vote/night 콜백을 실제 게임 액션으로 라우팅
- `RoomService.JoinByCode()`의 defer + 수동 unlock 혼용으로 인한 deadlock 수정
- `mafia/game.go`에서 `room.Players` 접근 시 mutex 보호 추가
- `gameManager.activeGames` map에 `sync.Mutex` 추가
- mafia chat 이벤트 payload에 `sender_name` 필드 추가
- `ws.Hub`의 `StartGame`/`RestartGame`이 서버 context를 사용하도록 수정
- `gameManager.start()`의 이벤트 포워딩 goroutine에서 `game.Subscribe()` 이벤트를 WS 클라이언트에도 브로드캐스트 (`gameManager`에 게임 이벤트 브로드캐스트 콜백 추가)
- 게임 시작 후 역할 배정 완료 시 `hub.UpdateClientRole()`을 호출하여 WS Client.Role 동기화

## Capabilities

### New Capabilities

없음 — 이 변경은 기존 동작을 명세대로 동작하도록 수정하는 것이며 새로운 기능을 추가하지 않는다.

### Modified Capabilities

없음 — 외부에서 관찰 가능한 스펙 수준의 동작 변경은 없다. 내부 버그 수정만 포함한다.

## Impact

- `cmd/server/main.go` — `gameManager` 생성자(personaPool 주입), `NotifyEvent`, AI 콜백, 게임 이벤트 브로드캐스트 콜백 연결
- `internal/ai/agent.go` — 마피아 `decideKill` 추가
- `internal/games/mafia/game.go` — `sender_name` payload, `room.Players` mutex 접근
- `internal/platform/room.go` — `JoinByCode` lock 패턴 수정
- `internal/platform/ws/hub.go` — server context 전파, `UpdateClientRole` 호출 시점 보장
