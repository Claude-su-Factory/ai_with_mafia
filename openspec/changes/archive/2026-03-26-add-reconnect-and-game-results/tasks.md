## 1. Config 확장

- [x] 1.1 `config/config.go`의 `ServerConfig`에 `ReconnectGraceSec int` 필드 추가
- [x] 1.2 `config.toml`에 `reconnect_grace_sec = 30` 추가
- [x] 1.3 `main.go`에서 `NewHub()` 호출 시 grace 설정값 전달

## 2. Hub Grace Period

- [x] 2.1 `internal/platform/ws/hub.go`에 `pendingDisconnect` 구조체 추가 (`timer *time.Timer`, `roomID string`, `role entity.Role`)
- [x] 2.2 `Hub`에 `pendingDisconnects map[string]*pendingDisconnect`와 전용 mutex 추가
- [x] 2.3 `Hub`에 `graceSec int` 필드 추가, `NewHub()` 파라미터로 주입
- [x] 2.4 `Unregister()` 수정 — `graceSec > 0`이면 grace timer 시작, 만료 시 `doRemove()` 호출
- [x] 2.5 `doRemove()` 내부 메서드 추출 — `RemovePlayer()` 호출 + `player_removed` Pub/Sub publish
- [x] 2.6 `ServeWS()` 진입 시 `pendingDisconnects` 확인 — 존재하면 timer 취소 후 제거, `player_reconnected` Pub/Sub publish

## 3. 크로스 인스턴스 Reconnect 이벤트

- [x] 3.1 `internal/platform/ws/pubsub.go`의 payload 구조체에 `player_reconnected` / `player_removed` 이벤트 타입 추가
- [x] 3.2 `hub.startSubscriber()` goroutine에서 위 두 타입 처리 — `player_reconnected` 수신 시 해당 playerID의 grace timer 취소, `player_removed` 수신 시 무시(idempotent)

## 4. 게임 결과 저장

- [x] 4.1 `internal/games/mafia/game.go`의 `MafiaGame`에 `startedAt time.Time` 필드 추가
- [x] 4.2 `MafiaGame.Start()` 시작 시 `g.startedAt = time.Now()` 기록
- [x] 4.3 `endGame()` payload 확장 — `round`, `duration_sec`, `players`(id/name/role/is_ai/survived) 포함
- [x] 4.4 `cmd/server/main.go` 이벤트 포워딩 goroutine에서 `EventGameOver` 수신 시 payload 파싱 후 `gameResultRepo.Save()` 호출
- [x] 4.5 `gameResultRepo.Save()` 에러 시 `logger.Error()`로 기록, 게임 흐름 계속
- [x] 4.6 `main.go`에서 `_ = gameResultRepo` 제거, 실제 사용으로 교체
