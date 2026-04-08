## 1. 인프라 준비

- [x] 1.1 `docker-compose.yml`에 Redis 7-alpine 서비스 추가 (포트 6379)
- [x] 1.2 `config.toml`에 `[redis]` 섹션 추가 (`addr`, `password`, `db`)
- [x] 1.3 `config/config.go`에 `RedisConfig` 구조체 추가
- [x] 1.4 `go.mod`에 `go-redis/redis/v9`, `github.com/google/uuid` 의존성 추가
- [x] 1.5 `internal/repository/redis.go` — Redis 클라이언트 초기화 함수 (`NewRedisClient`) 작성
- [x] 1.6 `cmd/server/main.go` 서버 시작 시 `uuid.New().String()`으로 instanceID 생성, 이후 LeaderLock/Hub에 전달

## 2. DB 마이그레이션

- [x] 2.1 `migrations/000003_create_game_states.up.sql` — `game_states` 테이블 (room_id PK, phase, round, players_json JSONB, night_kills JSONB, updated_at)
- [x] 2.2 `migrations/000003_create_game_states.down.sql`
- [x] 2.3 `migrations/000004_create_ai_histories.up.sql` — `ai_histories` 테이블 (room_id + player_id PK, history_json JSONB, updated_at)
- [x] 2.4 `migrations/000004_create_ai_histories.down.sql`

## 3. Repository 계층 추가

- [x] 3.1 `internal/repository/game_state.go` — `GameStateRepository`: `Save(ctx, state)`, `GetByRoomID(ctx, roomID)`, `Delete(ctx, roomID)` 구현
- [x] 3.2 `internal/repository/ai_history.go` — `AIHistoryRepository`: `Save(ctx, roomID, playerID, history)`, `GetByRoom(ctx, roomID)` 구현
- [x] 3.3 `internal/repository/room.go`에 `GetByID(ctx, roomID)`, `ListPlaying(ctx)`, `ListPublic(ctx)` 조회 메서드 추가

## 4. RoomService DB 기반으로 교체

- [x] 4.1 `internal/platform/room.go`의 `RoomService`에 DB pool(`*pgxpool.Pool`)과 `*zap.Logger` 필드 추가
- [x] 4.2 `NewRoomService()` 생성자에 `pool *pgxpool.Pool`, `logger *zap.Logger` 파라미터 추가
- [x] 4.3 `Create` — 메모리 저장과 동시에 DB upsert (현재 `_ = roomRepo`로 연결 안 된 부분 연결)
- [x] 4.4 `GetByID` — 메모리 먼저, 없으면 DB 조회로 fallback
- [x] 4.5 `ListPublic` — DB에서 직접 조회로 교체
- [x] 4.6 `Join`, `JoinByCode` — DB 조회 결과를 메모리 캐시에도 반영
- [x] 4.7 `main.go`에서 `RoomService` 생성 시 DB pool + logger 주입

## 5. 게임 상태 체크포인트

- [x] 5.1 `internal/games/mafia/phases.go`의 `PhaseManager`에 `onSave func(ctx context.Context, state GameState) error` 콜백 추가
- [x] 5.2 `PhaseManager.RunDayDiscussion`, `RunDayVote`, `RunNight` 각 시작 직후 `onSave` 호출 (실패해도 게임 계속)
- [x] 5.3 `cmd/server/main.go`의 `gameManager.start()`에서 `onSave` 콜백을 `GameStateRepository.Save`로 연결
- [x] 5.4 게임 종료 시 `GameStateRepository.Delete(roomID)` 호출

## 6. AI 히스토리 지속성

- [x] 6.1 `internal/ai/manager.go`의 `Manager`에 `aiHistoryRepo *repository.AIHistoryRepository` 필드 추가
- [x] 6.2 `Manager`에 `SaveHistories(ctx, roomID)` 메서드 추가 — 모든 에이전트의 히스토리를 DB에 저장
- [x] 6.3 `cmd/server/main.go`의 이벤트 포워딩 goroutine에서 페이즈 전환 이벤트 시 `aiManager.SaveHistories()` 호출
- [x] 6.4 `Manager.SpawnAgents()`에 초기 히스토리 주입 지원 추가 — `preloadedHistories map[string][]anthropic.MessageParam` 파라미터 추가 (nil이면 빈 히스토리로 시작)

## 7. Leader Lock (Redis SETNX)

- [x] 7.1 `internal/platform/leader.go` — `LeaderLock` 구조체: `Acquire(ctx, roomID, instanceID) bool`, `Release(ctx, roomID)`, `Heartbeat(ctx, roomID)` (30초 TTL)
- [x] 7.2 `gameManager.start()`에서 `LeaderLock.Acquire()` 호출, 실패 시 return (이미 다른 인스턴스 담당)
- [x] 7.3 게임 루프 goroutine 내에서 `Heartbeat` goroutine 실행 (10초마다 갱신)
- [x] 7.4 게임 종료 시 `LeaderLock.Release()` 호출

## 8. 게임 복구 로직

- [x] 8.1 `cmd/server/main.go` 서버 시작 시 `recoverOrphanGames(ctx)` 함수 호출
- [x] 8.2 `recoverOrphanGames`: `GameStateRepository`에서 game_states 있는 방 목록 조회 → 각 방에 대해 `LeaderLock.Acquire()` 시도 → 성공 시 게임 복구
- [x] 8.3 게임 복구 시 DB에서 로드한 Room 객체를 `RoomService` 메모리 캐시에 추가 (GetByID fallback을 통해 자동 처리되도록 유도 가능)
- [x] 8.4 게임 복구 시 `AIHistoryRepository.GetByRoom()`으로 히스토리 로드 → `SpawnAgents`에 주입
- [x] 8.5 WS 클라이언트가 playing 상태 방에 연결될 때 leader 없으면 복구 트리거

## 9. Redis Pub/Sub WS Relay

- [x] 9.1 `internal/platform/ws/pubsub.go` — Redis Pub/Sub 구독/발행 헬퍼 작성
- [x] 9.2 `Hub`에 Redis 클라이언트와 instanceID 필드 추가, `NewHub` 파라미터에 추가
- [x] 9.3 `hub.Broadcast()`에서 로컬 전달 후 Redis `PUBLISH room:{id}` 호출 (origin 필드 포함)
- [x] 9.4 `hub.startSubscriber(ctx)` goroutine 추가 — `SUBSCRIBE room:*` 패턴 구독, 자신이 보낸 것(origin == instanceID) 제외하고 `broadcastLocal()` 호출
- [x] 9.5 `main.go`에서 `hub.startSubscriber(ctx)` 시작

## 10. 에러 로깅 수정

- [x] 10.1 `cmd/server/main.go` vote/night 콜백의 `_ = ag.game.HandleAction(...)` → `logger.Error()` 추가
- [x] 10.2 `cmd/server/main.go` `NotifyEvent`의 `_ = ag.game.HandleAction(...)` → `logger.Error()` 추가
- [x] 10.3 `cmd/server/main.go` game goroutine 내 `game.Start()` 이후 room 상태 전환 시 logger 호출 추가
- [x] 10.4 `internal/platform/ws/hub.go` Broadcast의 JSON marshal 실패 → `logger.Error()` 추가 (중복 위치 모두 처리)
- [x] 10.5 `internal/platform/ws/hub.go` 초기 상태 메시지 WS write 실패 → `logger.Warn()` 추가
- [x] 10.6 `internal/platform/ws/hub.go` 클라이언트 send 채널 drop → `logger.Warn(playerID, roomID)` 추가
- [x] 10.7 `internal/platform/ws/hub.go` `c.Close()` 에러 → `logger.Warn()` 추가
- [x] 10.8 `internal/ai/agent.go` eventCh drop → `logger.Warn(eventType, playerID)` 추가
- [x] 10.9 `internal/games/mafia/game.go` eventCh drop → `logger.Warn(eventType, roomID)` 추가
- [x] 10.10 `internal/platform/room.go` `newID()`, `generateCode()` — `rand.Read()` 에러를 반환하도록 수정, 호출부 에러 처리
- [x] 10.11 `cmd/server/main.go` WS playerID 누락 시 `logger.Warn()` 추가
