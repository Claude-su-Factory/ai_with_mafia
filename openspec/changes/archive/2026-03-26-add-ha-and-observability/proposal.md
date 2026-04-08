## Why

서버는 최소 3개 인스턴스로 운영되며 LB 뒤에 배치된다. 현재 모든 상태(방 목록, 게임 진행, AI 히스토리, WS 연결)가 프로세스 메모리에만 존재하므로 인스턴스가 죽으면 진행 중인 게임이 소멸된다. 또한 서로 다른 인스턴스에 연결된 같은 방의 플레이어끼리 메시지 교환이 불가능하다.

목표는 Level 1 HA다: 인스턴스 재시작 후 게임 상태와 AI 맥락이 복구된다. 페이즈 타이머는 리셋되는 것을 허용한다.

추가로 코드 전반의 에러 로깅 누락을 정리하여 운영 가시성을 확보한다.

## What Changes

### HA 관련

- `RoomService`를 메모리 맵 기반에서 PostgreSQL 기반으로 교체 (rooms 테이블이 이미 있으나 실제로 연결되지 않음)
- `game_states` 테이블 추가 + 페이즈 전환마다 게임 상태 저장
- `ai_histories` 테이블 추가 + AI 에이전트 히스토리 직렬화 저장
- Redis Pub/Sub 추가: 인스턴스 간 WS 메시지 relay
- Redis SETNX leader lock: 방당 하나의 인스턴스만 게임 루프를 담당하도록 제어
- 게임 복구 로직: 서버 시작 시 playing 상태이나 leader 없는 방을 감지하여 게임 루프 재시작
- `docker-compose.yml`에 Redis 서비스 추가

### 에러 로깅 관련

- goroutine 내부 에러 (WS write, HandleAction, AI 호출 실패) 로깅 추가
- `_ =` 로 버려지는 중요 에러 처리

## Capabilities

### New Capabilities

없음 — 외부 동작 명세 변경 없음

### Modified Capabilities

- **방 정보 조회**: DB 기반으로 변경되어 인스턴스 재시작 후에도 방 목록 유지
- **게임 진행**: 인스턴스 크래시 후 다른 인스턴스가 복구, 타이머 리셋으로 재개
- **AI 에이전트**: 크래시 후 DB에서 히스토리 복원하여 대화 맥락 유지

## Impact

- `docker-compose.yml` — Redis 서비스 추가
- `config.toml` + `config/config.go` — Redis DSN 설정 추가
- `internal/repository/db.go` — Redis 클라이언트 초기화
- `internal/repository/room.go` — RoomRepository를 RoomService 역할까지 확장 (또는 RoomService를 DB 기반으로 교체)
- `migrations/` — game_states, ai_histories 테이블 마이그레이션
- `internal/repository/game_state.go` — GameState 저장/조회
- `internal/repository/ai_history.go` — AI 히스토리 저장/조회
- `internal/platform/ws/hub.go` — Redis Pub/Sub relay 추가
- `internal/platform/room.go` — DB 기반으로 교체
- `cmd/server/main.go` — Redis 초기화, 복구 로직, 에러 로깅
- `internal/ai/agent.go` — 에러 로깅, 히스토리 저장 훅
- `internal/games/mafia/phases.go` — 페이즈 전환 시 game_state 저장
