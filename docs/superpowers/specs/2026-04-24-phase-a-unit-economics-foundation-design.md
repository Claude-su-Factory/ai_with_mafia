# Phase A — Unit Economics Foundation

**Status:** Design approved 2026-04-24
**Related:**
- Product Principle: [`CLAUDE.md`](../../../CLAUDE.md) "Unit Economics 렌즈" 섹션
- Architecture Decision: [`docs/ARCHITECTURE.md`](../../ARCHITECTURE.md) §4.12
- ROADMAP backing: [`docs/ROADMAP.md`](../../ROADMAP.md) "수익화 · 리텐션 레버 4축"

---

## 1. Context & Goal

이 플랫폼은 Claude API 비용을 광고 수익으로 감당하는 구조다. Phase A 는
**unit economics 를 실측 가능한 상태로 만드는 기반 공사**다.

- **비용 축(①)** — Claude API 호출을 prompt cache 로 최적화
- **수익 축(②)** — 광고 슬롯 3곳(Lobby / WaitingRoom / Result) 가동
- **리텐션·밀도 축(③④)** — `빠른 참가` 버튼으로 진입 마찰 제거, 인간 밀도↑
- **측정 축** — 위 세 변화를 실측할 `game_metrics` 테이블과 로그 태깅

Phase A 종료 시점에 **B/C/D 의 ROI 를 실측 데이터 기반**으로 평가할 수 있어야 한다.

### Non-goals (Phase A 에서 **하지 않음**)

- 전면 광고(interstitial), 보상형 비디오 — Phase B
- 새 역할(의사·기자·스파이), 방 사이즈 가변 — Phase C
- 랭킹·시즌·도전과제 — Phase D
- ELO·큐·친구 매칭 — Phase D
- 광고 제공자 변경 (AdSense 유지)
- 게임 엔진 규칙 변경 (6인 고정 유지)
- 관리자 대시보드·시각화 UI — SQL 질의만으로 충분

---

## 2. Components Overview

| # | 이름 | 위치 | 규모 |
|---|------|------|------|
| A | **AI Cost Optimizer** | `backend/internal/ai/` | prompt cache + config 정리 |
| B | **Ad Integration** | `frontend/src/components/AdBanner.tsx` + 3 페이지 + `/api/metrics/ad` | 컴포넌트 배치 + 측정 엔드포인트 |
| C | **Quick Match (축소판)** | `backend/internal/platform/handler.go` + 신규 `/api/rooms/quick` + `LobbyPage` | join-or-create 한 개 엔드포인트 |
| D | **Metrics Foundation** | Migration 000007 + `GameMetricsRepository` + upsert hook 여러 곳 | 측정 기반 |

### 의존성

```
[A. AI Cost Optimizer] ─┐
                        ├─► emit → [D. Metrics]  (게임 단위 upsert)
[B. Ad Integration]    ─┤
                        │
[C. Quick Match]       ─┘
```

- A / B / C 는 서로 독립적 — 병렬 구현 가능
- D 는 세 컴포넌트가 쏘는 이벤트를 **수동적으로** 집계
- **구현 순서 권장:** D 의 스키마와 Repo 인터페이스 먼저 확정 → A / B / C 병렬 착수

---

## 3. Component Detail

### 3-A. AI Cost Optimizer

#### 변경 대상
- `backend/internal/ai/manager.go` — Anthropic 호출 지점
- `backend/internal/ai/persona_pool.go` (또는 동등) — persona prompt 구성
- `backend/config.toml` / `backend/config.example.toml` — 신규 `[ai]` 필드
- `backend/internal/ai/manager_test.go` (신규)

#### 설계
- **Prompt caching** — Anthropic `messages.create` 요청의 **system** 블록과 **persona block** 에 `cache_control: { type: "ephemeral" }` 를 부착
- **모델 분기 정리** — 일반 대화는 `model_default` (Haiku 4.5). "투표 직전 reasoning 타이밍" 만 `model_reasoning` (Sonnet 4.6) 사용. 현재 분기 로직을 명시적 함수(`selectModel(turnContext)`)로 추출
- **토큰·발화 파라미터** — `config.toml.[ai]` 에 다음 추가:
  - `max_tokens_per_turn` (int, 기본 160) — sdk `MaxTokens` 로 직접 전달
- **발화 쿨다운** — 같은 AI 가 짧은 시간 내 연속 발화하지 않도록 한다. 구현 단계에서 `manager.go` 의 기존 `response_delay_*` 로직을 먼저 확인하고, 이미 동일 AI 필터가 있으면 재사용. 없으면 `lastSpokeAt map[playerID]time.Time` 을 더해 `response_delay_min` 이내 재발화 차단. **구현 시작 시 실제 코드 확인 필수.**

#### 측정 hook
매 `messages.create` 응답마다 아래 4개 값을 emit:

```go
type AIUsageEvent struct {
    GameID              string
    TokensIn            int
    TokensOut           int
    CacheReadTokens     int  // response.usage.cache_read_input_tokens
    CacheCreationTokens int  // response.usage.cache_creation_input_tokens
}
```

`Manager` 는 이 이벤트를 `GameMetricsRepository.AddAIUsage(ctx, event)` 로 upsert. 누적 합산.

#### 4축 영향
- ① 비용 ⬇⬇ (cache hit 70% 이상 목표)
- ② 수익 — (간접)
- ③ 리텐션 — (AI 응답 품질 유지 필수)
- ④ 인간 밀도 — (불변)

#### 테스트 (`manager_test.go`)
- `TestPromptCacheControl_AttachedToSystemAndPersona` — 실제 Anthropic 요청 payload 구성을 검사 (sdk 호출 전 단계 mock)
- `TestSelectModel_ReasoningPathUsesSonnet` — 투표 직전 타이밍 플래그가 Sonnet 선택
- `TestSelectModel_DefaultPathUsesHaiku` — 일반 대화는 Haiku
- `TestCooldown_SameAIBlocked` — 같은 AI 연속 발화 거절
- (optional) `TestTokenLimit_Respected` — `max_tokens_per_turn` 이 sdk 옵션으로 전달됨

---

### 3-B. Ad Integration

#### 변경 대상
- `frontend/src/components/AdBanner.tsx` — 배너 본체 (이미 placeholder 존재)
- `frontend/src/pages/LobbyPage.tsx` — 하단 배너
- `frontend/src/components/WaitingRoom.tsx` — 사이드 또는 하단 배너
- `frontend/src/components/ResultOverlay.tsx` — 결과창 하단 배너
- `backend/internal/platform/handler.go` — `POST /api/metrics/ad` 엔드포인트 추가
- `backend/internal/platform/handler_test.go` — 엔드포인트 테스트

#### 설계
- **AdBanner 계약**
  - props: `{ slot: 'lobby' | 'waiting' | 'result'; gameID?: string }`
  - `gameID` 규칙: `slot === 'lobby'` 이면 undefined. `slot === 'waiting'` 또는 `'result'` 이면 반드시 게임/방 식별자 전달 (타입상 여전히 optional 이나 런타임 계약)
  - `VITE_ADSENSE_CLIENT` / `VITE_ADSENSE_SLOT_*` env 없으면 **dev**에서 placeholder 박스(`[AD:{slot}]`), **prod**에서 완전 no-op (`null`)
  - **레이아웃 shift 방지**: wrapper 에 고정 `min-height` (desktop 90px, mobile 50px). 광고 미표시 상태도 공간 예약
- **Impression 로깅**
  - AdBanner 가 mount 되고 DOM 에 실제로 그려지면(`IntersectionObserver` 50% visible) `POST /api/metrics/ad` 1회 호출
  - body: `{ slot: string, game_id?: string }`
  - 세션당 같은 slot 은 쿨다운 30초 (중복 호출 방지)
- **Backend 엔드포인트**
  - `POST /api/metrics/ad` — 인증 불필요 (공개 trigger)
  - Rate limit: IP당 30 req/min. `github.com/gofiber/fiber/v2/middleware/limiter` 를 이 엔드포인트 그룹에만 적용. **Storage 에 Redis 주입** (`Storage: redislimiter.New(...)` 또는 자체 어댑터) — 멀티 Pod 환경에서 IP 허용량이 Pod 수만큼 증폭되지 않도록 함. 이미 `go-redis` 클라이언트가 DI 로 주입되어 있으므로 추가 인프라 비용 없음
  - `GameMetricsRepository.IncrementAdImpression(ctx, gameID, slot)` 호출. `gameID` 없으면 (lobby) `room_id = NULL` 인 row 를 slot 별로 별도 집계 (세부는 3-D 참조)

#### 4축 영향
- ① 비용 거의 없음
- ② 수익 ⬆⬆
- ③ 리텐션 — (레이아웃 안정 필수, negative 위험 있음)
- ④ 인간 밀도 — 없음

#### 테스트
- Backend: `TestAdMetrics_IncrementsRow`, `TestAdMetrics_RateLimited` (handler_test)
- Frontend: Vitest 부재 → tsc + 수동 시각 확인. 이번 Phase 의 프론트 테스트 부재는 **ROADMAP T2-5 (Vitest 도입)** 의존 이슈로 분리

---

### 3-C. Quick Match (축소판)

#### 변경 대상
- `backend/internal/platform/handler.go` — `POST /api/rooms/quick` 추가
- `backend/internal/platform/room.go` — `RoomService.FindOrCreatePublicRoom(player, displayName)` 메서드
- `backend/internal/platform/handler_test.go` — 5개 테스트
- `frontend/src/pages/LobbyPage.tsx` — `빠른 참가` 버튼
- `frontend/src/api.ts` — `quickMatch()` 함수

#### 설계
- **API**: `POST /api/rooms/quick`
  - 요청 body: 없음 (Authorization header 만 사용)
  - 응답: `{ room_id: string, player_id: string, created: bool }`
- **RoomService.FindOrCreatePublicRoom 로직**:
  1. 인메모리 `rooms` 순회, `visibility=public` `status=waiting` `HumanCount < MaxHumans` 인 방 후보 수집
  2. 후보가 있으면 그중 **`HumanCount` 가 가장 큰 방**(= 채워지기 직전) 에 Join. `HumanCount` 동률 시: `entity.Room` 에 생성 시각 필드가 없으면 `room.ID` 사전식 가장 작은 것을 선택 (결정적이고 추가 스키마 불필요)
  3. 후보 없으면 `CreateRoom({Name: "빠른 게임", Visibility: "public", MaxHumans: 6}, player, displayName)` 후 Join
  4. 메트릭 emit: `quick_match_result: "joined" | "created"`, latency 측정
- **프론트**: 버튼 클릭 → `quickMatch()` → 응답의 `room_id` 로 navigate. 에러(네트워크/401)는 기존 에러 토스트와 동일 처리

#### 동시성 주의
- `FindOrCreatePublicRoom` 는 `rooms` 맵을 스캔하면서 Join 후보를 고르므로 `sync.RWMutex` 를 **쓰기 락**으로 유지한 채 후보 선택 + 조인까지 원자 처리
- 이 락을 놓치면 두 플레이어가 동시에 같은 "마지막 자리" 를 차지하려다 overfill 발생 가능

#### 4축 영향
- ① 비용 small negative (구현 시간)
- ② 수익 + (도달률 개선이 장기 impression 증가)
- ③ 리텐션 ⬆
- ④ 인간 밀도 ⬆⬆

#### 테스트 (`handler_test.go`, 5개)
- `TestQuickMatch_NoPublicRoom_CreatesNew` — 공개 방 0개 → 새 방 + created=true
- `TestQuickMatch_PublicRoomFull_CreatesNew` — 공개 방 1개 full → 새 방 + created=true
- `TestQuickMatch_PublicRoomAvailable_Joins` — 공개 방 1개 빈자리 → joined + created=false
- `TestQuickMatch_IgnoresPrivateRoom` — private 방만 있으면 새 방 생성 (private join 금지)
- `TestQuickMatch_Unauthorized` — JWT 없으면 401

---

### 3-D. Metrics Foundation

#### 변경 대상
- `backend/migrations/000007_create_game_metrics.up.sql` / `.down.sql` (신규)
- `backend/internal/repository/game_metrics.go` (신규) — `GameMetricsRepository`
- `backend/internal/repository/game_metrics_test.go` (신규)
- `backend/cmd/server/main.go` — DI 에 GameMetricsRepository 주입

#### 스키마 (migration 000007 up)

```sql
CREATE TABLE game_metrics (
    game_id                 TEXT PRIMARY KEY,
    room_id                 TEXT NOT NULL,
    started_at              TIMESTAMPTZ NOT NULL,
    ended_at                TIMESTAMPTZ,
    humans_count            INT NOT NULL DEFAULT 0,
    ai_count                INT NOT NULL DEFAULT 0,
    rounds                  INT,
    winner                  TEXT,
    tokens_in               BIGINT NOT NULL DEFAULT 0,
    tokens_out              BIGINT NOT NULL DEFAULT 0,
    cache_read_tokens       BIGINT NOT NULL DEFAULT 0,
    cache_creation_tokens   BIGINT NOT NULL DEFAULT 0,
    ad_impressions_lobby    INT NOT NULL DEFAULT 0,
    ad_impressions_waiting  INT NOT NULL DEFAULT 0,
    ad_impressions_result   INT NOT NULL DEFAULT 0,
    quick_match_joins       INT NOT NULL DEFAULT 0,
    quick_match_creates     INT NOT NULL DEFAULT 0,
    quick_match_latency_ms  INT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_game_metrics_started_at ON game_metrics(started_at);
CREATE INDEX idx_game_metrics_room_id   ON game_metrics(room_id);
```

**Lobby impression은 game_id 없이 발생**. 이 경우는 별도 저장 — 옵션 2개 중 선택:

- **옵션 1 (권장):** `game_metrics` 의 `game_id` 를 `lobby-YYYY-MM-DD` 같은 일자 집계 row 로 저장. Lobby impression 카운터를 그 row 에 누적. `room_id = 'lobby'` sentinel.
- 옵션 2: 별도 `lobby_metrics` 테이블.

**결정: 옵션 1** — 스키마 단순화 우선. `game_id = 'lobby-2026-04-24'` 같은 식. 분석 시 WHERE 조건으로 구분.

#### Repository 인터페이스

```go
type GameMetricsRepository interface {
    // 게임 시작 시
    Create(ctx context.Context, init GameMetricInit) error

    // 게임 종료 시
    Finalize(ctx context.Context, game GameMetricFinal) error

    // AI API 호출마다
    AddAIUsage(ctx context.Context, gameID string, usage AIUsage) error

    // 광고 impression
    IncrementAdImpression(ctx context.Context, slot, gameID string) error

    // 빠른 참가
    RecordQuickMatch(ctx context.Context, gameID string, result string, latencyMs int) error
}

type GameMetricInit struct {
    GameID, RoomID string
    StartedAt      time.Time
    Humans, AIs    int
}
type GameMetricFinal struct {
    GameID  string
    EndedAt time.Time
    Rounds  int
    Winner  string
}
type AIUsage struct {
    TokensIn, TokensOut, CacheRead, CacheCreation int
}
```

모든 메서드는 SQL `ON CONFLICT (game_id) DO UPDATE SET <counter> = <counter> + EXCLUDED.<counter>` 패턴. 동시성 안전.

#### DI 실패 안전
`pgxpool` 이 nil 이면 Repository 는 no-op (로그만). 기존 `UserStore` / `GameResultStore` 와 동일 패턴. 테스트에서 nil DB 상태로도 돌아감.

#### 4축 영향
- ① 비용 small (DB I/O 몇 건 추가)
- ②③④ 장기 positive (측정 있어야 의사결정 가능)

#### 테스트 (`game_metrics_test.go`, 3개)
- `TestGameMetrics_UpsertMerges` — Create 후 AddAIUsage 2회, counter 정확히 합산
- `TestGameMetrics_NilPool_NoOp` — nil DB 에서 모든 메서드 에러 없음 (log만)
- `TestGameMetrics_ConcurrentIncrements` — 동일 game_id 에 대해 goroutine 10개가 동시 ad impression 증분, 총합 == 10

---

## 4. Data Flow

```
┌──────────────────────────────────────────────────────────────┐
│ Game lifecycle                                               │
│                                                              │
│  RoomService.Start(game_id, room_id, humans, ais)            │
│      └──► GameMetricsRepo.Create(init)                       │
│                                                              │
│  [AI turn] Manager.messages.create(...)                      │
│      └──► GameMetricsRepo.AddAIUsage(game_id, usage)         │
│                                                              │
│  PhaseManager.endGame(winner, rounds)                        │
│      └──► GameMetricsRepo.Finalize(final)                    │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│ Ad flow                                                      │
│                                                              │
│  AdBanner.useEffect (IntersectionObserver 50%)               │
│      └──► fetch POST /api/metrics/ad { slot, game_id? }      │
│            └──► handler ──► GameMetricsRepo                  │
│                               .IncrementAdImpression         │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│ Quick match flow                                             │
│                                                              │
│  LobbyPage click "빠른 참가"                                  │
│      └──► fetch POST /api/rooms/quick                        │
│            └──► handler ──► RoomService.FindOrCreatePublic   │
│                   ├─► join existing OR                       │
│                   ├─► create new                             │
│                   └─► GameMetricsRepo.RecordQuickMatch       │
└──────────────────────────────────────────────────────────────┘
```

---

## 4a. Concurrency & Distribution Analysis

**Mandatory section per CLAUDE.md "동시성·분산 안전성" rule.** 각 컴포넌트의 상태가 멀티 Pod 환경에서 어떻게 동작하는지, 단일 Pod 전제라면 왜 허용되는지, 이관 경로가 무엇인지를 명시한다.

### 컴포넌트별 상태 분포

| 컴포넌트 | 상태 위치 | 현재(단일 Pod) 정확성 | 멀티 Pod 확장 시 거동 | 허용 여부 / 이관 |
|---------|---------|:------:|---------|--------|
| A. AI 발화 쿨다운 (`lastSpokeAt map`) | 프로세스 로컬 | ✅ | 같은 게임은 **LeaderLock** 으로 단일 Pod 가 관리 → 해당 Pod 에만 쿨다운 유지하면 충분 | ✅ 허용 (리더 failover 시 쿨다운 리셋 — AI가 약간 빨리 말할 수 있음, non-critical) |
| A. Anthropic prompt cache | Anthropic 서버 측 | ✅ | 분산 무관 (서버 측 캐시는 API 키/프로젝트 단위) | ✅ 분산 안전 |
| B. Ad impression 쿨다운 (세션당 30초) | **클라이언트 로컬** | ✅ | 분산 무관 (브라우저 메모리) | ✅ 분산 안전 |
| B. Rate limiter (`/api/metrics/ad`) | **Redis 기반**으로 구현 | ✅ | Redis 단일 진실. Pod 수에 관계없이 IP 당 30 req/min | ✅ 분산 안전 — **Phase A 에서 Redis storage 선택이 이 때문** |
| B. Ad impression 카운터 | Postgres (`game_metrics.ad_impressions_*`) | ✅ | `ON CONFLICT DO UPDATE SET col = col + EXCLUDED.col` — row-level 락 + atomic | ✅ 분산 안전 |
| C. `RoomService.rooms` 맵 | **프로세스 로컬** | ✅ | ⚠️ 각 Pod 가 자기 방만 조회. Quick Match 가 다른 Pod 의 빈자리 방을 못 찾음 → 밀도↓, 생성 증가 | ⚠️ 단일 Pod 전제. **ROADMAP T3-2 이관 대상** |
| C. Quick Match 후보 선택 + Join 원자성 | 같은 Pod 내 `sync.RWMutex` | ✅ | 같은 Pod 내는 안전. **Pod 간 경쟁은 자연히 분리** (다른 Pod 의 방은 안 보이므로 race 대상 자체 없음) | ✅ 단일 Pod 기준 정확 |
| D. `game_metrics` upsert | Postgres | ✅ | `ON CONFLICT DO UPDATE` atomic. 동시 upsert 안전 | ✅ 분산 안전 |
| D. Game lifecycle hook (Create/Finalize) | 리더 Pod 1곳 | ✅ | LeaderLock 으로 단일 리더만 발행 | ✅ 분산 안전 |

### 단일 Pod 전제 — RoomService 상세 분석

**현재 단일 Pod 기준 정확성:**
- 한 Pod 내에서 `FindOrCreatePublicRoom` 은 `sync.Mutex` 로 후보 선택 + Join 을 원자화 → **overfill 방지**
- 모든 사용자가 같은 Pod 에 연결되므로 Quick Match 가 찾는 "빈자리 방" 은 실제로 모든 공개 방을 본다

**멀티 Pod 전환 시 깨지는 것:**
- Pod A 에 생성된 방이 Pod B 의 Quick Match 에 보이지 않음
- 결과: **새 방 생성 비율 증가** → 인간 밀도 감소 → AI 비용 증가 (즉 Unit Economics ① / ④ 축에 직접 타격)
- overfill 은 발생하지 않음 (각 Pod 의 방은 여전히 로컬에서 원자적 join)

**이관 경로 (ROADMAP T3-2):**
- Redis `sorted set` 으로 공개 방 인덱스 유지 (score = HumanCount, member = room_id)
- `RoomService.Create / AddPlayer / RemovePlayer` 가 Redis 동기 업데이트
- Quick Match 는 `ZRANGEBYSCORE public_rooms 0 5 WITHSCORES` 로 자리 있는 방 조회
- **방 실제 상태 자체는 어느 Pod 가 보유해야 하는가?** — WS 연결이 붙은 Pod 가 보유 (현재 구조 유지) + 다른 Pod 가 "이 방에 Join 하려면 나한테 다시 연결해" 라고 redirect. sticky routing + Redis registry 조합

**Phase A 에서 할 일:**
- 실제 해결은 T3-2 로 연기
- **관측:** `game_metrics` 에 `quick_match_creates` / `quick_match_joins` 비율 기록 → 멀티 Pod 전환 후 `create` 비율 상승이 보이면 이관 타이밍으로 판단

### 새 컴포넌트에서 도입하는 Redis 사용

Phase A 가 **Redis 를 새로 사용하는 유일한 지점**: Ad 엔드포인트 rate limiter.

- 선택 이유: 멀티 Pod 에서 IP 당 허용량 보존이 UX(악의적 광고 조회 방지 + 수익 왜곡 방지)에 직접 영향
- 구현 비용: 기존 `go-redis` 클라이언트를 limiter middleware storage 로 어댑터 작성 (~30 LOC)
- 대안 검토:
  - **in-memory limiter**: Pod N 개 → 실제 허용량 30N req/min. 프로덕션 Scale 시 무력화. ❌
  - **DB 기반 counter**: 왕복 비용 큼, Redis 가 있는데 굳이. ❌

### 위험 목록 (단일 Pod → 멀티 Pod 전환 시)

| 위험 | 영향 | 완화 |
|-----|------|------|
| RoomService `rooms` 분할 | 인간 밀도 ↓, AI 비용 ↑ | T3-2 이관 (Redis sorted set) |
| WS 클라이언트 연결 고정 | 사용자는 한 Pod 에 붙어 있음 → 그 Pod 재시작 시 재연결 | 기존 WS reconnect 로직 유지. 세션 재참가는 이미 Redis 기반으로 복구됨 |
| LeaderLock 전환 | 게임 소유 Pod 이 바뀔 때 state 직렬화 필요 | 이미 Postgres checkpointing 있음 (game_states 테이블) |
| AI 쿨다운 리셋 | 리더 Pod failover 시 같은 AI 가 빨리 다시 말할 수 있음 | 허용 (UX 영향 미미) |

---

## 5. Error Handling

| 장애 | 대응 |
|-----|------|
| Anthropic API 실패 | 해당 AI 발화 스킵. 게임 진행 계속. metric 에는 `tokens_in/out = 0` row |
| `GameMetricsRepo` 쓰기 실패 | `logger.Warn("metrics: upsert failed", ...)` 만. **게임 흐름을 절대 블로킹하지 않는다** — 경제 지표 누락 ≠ 게임 중단 |
| `POST /api/metrics/ad` 실패 | 프론트는 silent retry 1회 후 포기. 사용자 UX 영향 없음 |
| `POST /api/rooms/quick` 실패 | 기존 에러 토스트로 표시. 사용자는 수동 로비 진입 가능 (fallback UX) |
| Rate limit 초과 | 429. 프론트는 silent (impression 은 fire-and-forget) |
| `rooms` 맵 쓰기 락 경합 (Quick Match) | Go `sync.RWMutex` 순서 보장. 데드락 위험 회피 위해 RoomService 내부 락 재진입 금지 — `FindOrCreatePublicRoom` 가 락 획득 후 `Join` 호출 시 내부 `Join` 은 이미 락 보유 상태로 동작하도록 private helper 분리 |

---

## 6. Testing Strategy

| 레이어 | 도구 | 신규 테스트 수 |
|-------|------|---------------|
| AI manager | `go test` + mock anthropic client | ~5 |
| Quick Match handler | `go test` + mock UserStore | 5 |
| Ad Metrics handler | `go test` | 2 |
| GameMetrics repo | `go test` (DB nil path) | 3 |
| 통합 (optional) | `testcontainers` | 0 (ROADMAP T2-5 로 분리) |
| Frontend | tsc + 수동 시각 | - |

**모든 신규 Go 테스트는 `test-driven-development` 스킬로 RED → GREEN 사이클 준수.**

---

## 7. Success Criteria (Phase A 완료 정의)

| 기준 | 목표 | 측정 방법 |
|-----|------|----------|
| Prompt cache hit rate (system + persona) | ≥ 70% | `cache_read_tokens / (cache_read_tokens + tokens_in)` 평균 |
| Rate limiter Redis storage 동작 | 2 Pod 시뮬레이션에서 IP당 30 req/min 유지 | 통합 테스트 또는 수동 — 2개 프로세스 기동 후 동일 IP에서 총 60 req/min 시도 시 절반 거부 |
| 광고 슬롯 3곳 impression 로그 관측 | 3곳 전부 ≥ 1 | `SELECT SUM(ad_impressions_*) FROM game_metrics` |
| `빠른 참가` latency | ≤ 3s (95th percentile) | `quick_match_latency_ms` |
| `빠른 참가` 성공률 | ≥ 95% | `created + joined` / 총 클릭 수 |
| `game_metrics` row 커버리지 | 100% (누락 0) | 기간 내 `game_results` row 수 == `game_metrics` row 수 |
| 기존 + 신규 테스트 | all green | `go test ./...` + `go test -race ./...` |

---

## 8. 4-축 영향 요약 (Product Principle 연결)

| 컴포넌트 | ① 비용 | ② 수익 | ③ 리텐션 | ④ 밀도 | 2축 이상 positive? |
|---------|:----:|:----:|:----:|:----:|:-------:|
| A. AI Cost Optimizer | ⬇⬇ | - | - | - | OK (비용만으로도 강한 단독 positive) |
| B. Ad Integration | - | ⬆⬆ | ⚠️ | - | OK (수익 + 레이아웃 안정이라는 리텐션 전제 준수) |
| C. Quick Match | small ⬇ | + | ⬆ | ⬆⬆ | OK (3축 positive) |
| D. Metrics | small ⬇ | + | + | + | OK (3축 간접 positive) |

**Phase A 전체:** 4축 전부에 긍정적. 특히 D 가 B/C/D 의 ROI 를 실측 가능하게 만들어 **Phase B/C/D 의 설계 근거를 수치화**한다.

---

## 9. Open Questions (사용자 확인 필요)

1. **Lobby impression의 `game_id` 방식** — "옵션 1: `lobby-YYYY-MM-DD` sentinel row" 로 잠정 결정. 대체안(별도 `lobby_metrics` 테이블)을 선호하면 말씀 주세요
2. **Quick match "가장 채워진 방" vs "가장 빈 방" 선택** — 잠정 결정은 **"가장 채워진 방"** (빨리 채워져서 게임 시작). 대안은 "가장 빈 방" (플레이어 1명 방에 합류해 둘이 기다리기). 인간 밀도 관점에선 채워진 쪽이 유리
3. **`max_tokens_per_turn` 기본값 160** — 기존 AI 발화 평균 길이를 실측 안 한 값. 너무 짧으면 발화 잘림. Phase A 구현 중 첫 시운전 때 조정 가능하다는 전제

---

## 10. Implementation Order

1. **Step 1 — Metrics schema & repo** (D) — 30% 분량. 다른 컴포넌트의 emit 대상 확정
2. **Step 2 — AI Cost Optimizer** (A) 와 **Quick Match** (C) 병렬 — 40%
3. **Step 3 — Ad Integration** (B) — 20%
4. **Step 4 — Integration test + 6개 정량 성공 기준 확인** — 10%

---

## 11. 검토 이력

| 날짜 | 이벤트 | 결과 |
|-----|-------|------|
| 2026-04-24 | 초기 작성 | brainstorming 세션에서 owner-level 결정 반영 |
| 2026-04-24 | self-review pass 1 | 수정 4건: (a) YAGNI 제거 — `system_cache_ttl_sec` 삭제, (b) AdBanner `gameID` 런타임 계약 명시, (c) rate limit 구현 방법 고정(`fiber/middleware/limiter`), (d) Quick Match tie-break 을 구현 가능한 형태(`room.ID` 사전식) 로 확정 + 발화 쿨다운 ambiguity 완화 |
| 2026-04-24 | pass 2 — 동시성·분산 반영 | 신규 §4a "Concurrency & Distribution Analysis" 추가. 컴포넌트별 상태 위치·허용 여부·이관 경로 테이블화. Ad rate limiter 를 **Redis storage 기반**으로 고정(멀티 Pod 에서 IP 할당 증폭 방지). RoomService 분산 이관은 ROADMAP T3-2 로 명시 연기. Success Criteria 에 2-Pod 시뮬레이션 검증 추가 |
