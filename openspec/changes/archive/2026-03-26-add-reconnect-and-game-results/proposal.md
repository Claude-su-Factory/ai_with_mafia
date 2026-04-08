## Why

WS 연결이 끊기면 즉시 플레이어가 게임에서 제거되어 네트워크 순간 끊김이나 새로고침만으로도 게임 슬롯이 날아가는 UX 문제가 있다. 또한 게임 결과 저장 인프라(`GameResultRepository`, DB 스키마)가 이미 존재하지만 실제로 저장하는 코드가 없어 게임 기록이 전혀 쌓이지 않는다.

## What Changes

- WS 연결 끊김 시 즉시 제거 대신 grace period 타이머 도입 — N초 내 재연결 시 세션 복원
- 크로스 인스턴스 reconnect 이벤트: 기존 Redis `room:{id}` Pub/Sub 채널에 `player_reconnected` / `player_removed` 이벤트 타입 추가
- 게임 종료 시 `gameResultRepo.Save()` 호출 — winner, round, duration, 플레이어별 역할/생존 여부 저장

## Capabilities

### New Capabilities

- `player-reconnect`: WS 재연결 grace period — N초 내 동일 playerID 재연결 시 게임 슬롯 및 역할 세션 복원
- `game-results`: 게임 종료 시 결과(승자 팀, 라운드 수, 소요 시간, 플레이어별 역할/생존) DB 저장

### Modified Capabilities

없음

## Impact

- `internal/platform/ws/hub.go` — grace period 타이머 로직, Pub/Sub reconnect 이벤트 처리
- `internal/platform/ws/pubsub.go` — `player_reconnected` / `player_removed` 이벤트 타입 추가 (add-ha-and-observability에서 생성)
- `config/config.go`, `config.toml` — `reconnect_grace_sec` 설정 추가
- `internal/games/mafia/game.go` — `startedAt` 필드, `endGame()` payload 확장
- `cmd/server/main.go` — `EventGameOver` 수신 시 `gameResultRepo.Save()` 연결
- 의존: `add-ha-and-observability` 먼저 적용 필요 (Redis Pub/Sub 인프라)
