## Context

3개 이상의 인스턴스가 LB 뒤에서 동작한다. 인스턴스 크래시 시 게임 소멸을 방지하는 Level 1 HA를 구현한다. 타이머 리셋은 허용한다.

## Goals / Non-Goals

**Goals:**
- 인스턴스 간 WS 메시지 동기화 (Redis Pub/Sub)
- 방 상태를 DB 기반으로 관리
- 게임 상태 및 AI 히스토리를 DB에 체크포인트
- 크래시 후 복구 (타이머 리셋 허용)
- 에러 로깅 커버리지 확보

**Non-Goals:**
- 타이머 연속성 보장 (Level 2 이상)
- 끊김 없는 WS 연속성 (클라이언트 재접속 필요)
- 게임 상태 실시간 동기화 (페이즈 전환 체크포인트만)
- Redis 캐시 레이어 (DB 직접 접근으로 충분)

## Decisions

### 1. Redis 역할 범위

Redis를 캐시 목적으로 쓰지 않는다. 두 가지 용도만 사용한다:

- **Pub/Sub**: 방별 채널(`room:{id}`)로 WS 메시지 relay
- **Leader lock**: `SETNX game:{id}:leader {instance-uuid} EX 30` + heartbeat goroutine

방 목록/상태는 PostgreSQL이 single source of truth. 성능이 문제가 되면 그때 Redis 캐시를 고려한다.

### 2. RoomService — DB + 메모리 이중 구조 유지

현재 `platform/RoomService`는 메모리 맵이 유일한 소스다. DB를 authoritative source로 추가하되, 메모리 캐시는 유지한다.

메모리 캐시를 유지하는 이유: 게임 진행 중 `Room` 객체는 AI 플레이어 추가, 역할 배정 등으로 직접 변경된다. goroutine들이 동일한 포인터를 참조하므로 메모리에서 완전히 제거할 수 없다.

역할 분담:
- **메모리**: in-game 상태 (빠른 접근, 게임 중 변경)
- **DB**: 영속성 (재시작 후 복구, 다른 인스턴스 조회)

조회 패턴:
- `GetByID`: 메모리 먼저, 없으면 DB 조회 후 메모리에 로드
- `ListPublic`: DB 직접 조회 (다른 인스턴스의 방도 포함해야 하므로)
- `Create`, `Join`: DB 저장 + 메모리 동기화

`RoomService`에 `*zap.Logger` 필드 추가: DB 에러 로깅에 필요하다. `NewRoomService()` 생성자에 주입한다.

### 3. 게임 상태 체크포인트 시점

매 액션마다 저장하지 않는다. **페이즈 전환 시점**에만 저장한다.

```
day_discussion 시작 → game_states upsert
day_vote 시작       → game_states upsert
night 시작          → game_states upsert
결과 처리 후        → game_states upsert (또는 삭제)
```

크래시 복구 시 마지막 체크포인트 페이즈부터 타이머 리셋으로 재시작한다.

저장 내용:
```sql
game_states (
    room_id TEXT PRIMARY KEY,
    phase TEXT,
    round INT,
    players_json JSONB,      -- []Player {id, name, role, is_alive, is_ai} — 역할 포함 전체 목록
    night_kills JSONB,       -- map[string]string (killerID→targetID)
    updated_at TIMESTAMPTZ
)
```

`alive_player_ids`가 아닌 `players_json`을 저장한다. 역할(role)과 AI 여부가 없으면 게임 복구 시 GameState를 재구성할 수 없다. 생존 여부(is_alive)도 포함하여 페이즈 처리 상태를 복원한다.

### 4. AI 히스토리 직렬화

`[]anthropic.MessageParam`을 JSON으로 직렬화하여 저장한다. `anthropic.MessageParam`은 JSON 직렬화 가능한 구조체이므로 별도 변환 불필요.

저장 시점: 페이즈 전환마다. (게임 상태와 동일 타이밍)

```sql
ai_histories (
    room_id TEXT,
    player_id TEXT,
    history_json JSONB,
    updated_at TIMESTAMPTZ,
    PRIMARY KEY (room_id, player_id)
)
```

`Agent`에 `onPhaseSave(ctx) error` 콜백을 추가하거나, `Manager`가 페이즈 전환 시점에 모든 에이전트 히스토리를 수집해서 저장하는 방식을 택한다. 후자가 더 단순하다.

### 5. Leader Lock 패턴

```
인스턴스 시작 시:
  1. DB에서 playing 상태 방 목록 조회
  2. 각 방에 대해 Redis SET game:{id}:leader {uuid} NX EX 30 시도
  3. 성공한 방 → 게임 루프 복구 시작
  4. 실패한 방 → 다른 인스턴스가 담당 중, 스킵

게임 루프 실행 중:
  - 30초마다 EXPIRE game:{id}:leader 30 으로 갱신 (heartbeat)

인스턴스 크래시:
  - 30초 후 TTL 만료
  - 다른 인스턴스가 다음 DB 폴링 또는 WS 연결 이벤트 시 감지
```

새 WS 클라이언트가 playing 방에 연결될 때도 leader lock을 확인하는 트리거로 사용한다.

### 6. Redis Pub/Sub WS Relay

현재 `hub.Broadcast()`는 인스턴스 로컬 클라이언트에만 전달한다. Redis relay를 추가한다.

```
발신 경로:
  hub.Broadcast(roomID, msg, mafiaOnly)
    → Redis PUBLISH room:{id} {payload}
    → (자신의 로컬 클라이언트는 즉시 전달)

수신 경로 (별도 goroutine):
  Redis SUBSCRIBE room:{id}
    → hub.broadcastLocal(roomID, msg, mafiaOnly)
```

mafiaOnly 필터 정보는 payload에 포함해서 직렬화한다.

중복 전달 방지: 자신이 publish한 메시지는 subscribe에서도 받는다. `origin` 필드를 payload에 추가하여 자신이 보낸 메시지는 skip한다.

### 7. 에러 로깅 원칙

프로젝트 전체에서 `go.uber.org/zap`을 사용한다. 표준 `log` 패키지나 `fmt.Println` 사용 금지.

로그 레벨 기준:
- `zap.Error()`: goroutine 내부 에러, DB 저장 실패, 게임 로직 에러 — 프로그램 종료 안 함
- `zap.Warn()`: WS write 에러, 채널 드롭, 연결 종료 실패 — 복구 가능한 이벤트
- `zap.Info()`: 게임 시작/종료, 복구 완료 — 운영 추적용

구조화 필드 사용: `zap.String("room_id", roomID)`, `zap.Error(err)` 등 key-value 형태로 기록.

현재 zap 미보유 컴포넌트:
- `platform/RoomService`: `*zap.Logger` 추가 필요
- `platform/Handler`: 에러 처리는 Fiber error handler에 위임하므로 별도 로거 불필요

DB 저장 에러는 게임 중단 사유가 되면 안 된다: 체크포인트 실패 → 로그만 남기고 게임 계속 진행.

### 8. Instance UUID

각 서버 인스턴스는 시작 시 `uuid.New().String()`으로 고유 ID를 생성한다. Leader lock의 value로 사용된다. `go-uuid` 또는 `google/uuid` 라이브러리 사용. 환경변수로 외부 주입도 가능하나, 자동 생성으로 충분하다.

## Risks / Trade-offs

- **30초 leader TTL**: 크래시 후 최대 30초간 해당 방 게임이 정지될 수 있음. 클라이언트는 재접속 후 대기 화면이 필요하나, 현재 스코프 밖.
- **페이즈 내 크래시**: 체크포인트가 없으면 해당 페이즈 처음부터 재시작. 수십 초 되돌아갈 수 있음. Level 1에서 허용.
- **RoomService 시그니처 변경**: DB pool + logger 주입으로 `NewRoomService()` 파라미터가 변경되므로 handler, main.go 등 의존부 수정 필요.
- **`anthropic.MessageParam` JSON 직렬화**: SDK의 `param.Field` 패턴이 표준 JSON round-trip을 보장하는지 구현 전 검증 필요. 문제가 있으면 자체 직렬화 구조체로 변환하는 어댑터를 작성한다.

## Open Questions

- Redis 클라이언트 라이브러리: `go-redis/redis` v9 사용 (가장 널리 쓰이는 Go Redis 클라이언트)
- DB 폴링 주기 (leader 감지): WS 연결 이벤트 기반으로 처리하면 폴링 불필요
- Instance UUID 라이브러리: `github.com/google/uuid` 추가 예정
