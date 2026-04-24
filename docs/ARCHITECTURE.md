# ARCHITECTURE.md

AI 마피아 게임 플랫폼의 시스템 구성과 **핵심 설계 결정(Why 포함)** 을 기록한다.
새 세션이 이 문서만 읽고도 "왜 이 구조인가"를 이해할 수 있어야 한다.

> **업데이트 규칙**
> 아키텍처에 영향을 주는 변경(새 컴포넌트 추가, 의존성 교체, 설계 결정 번복)이 있을 때만 갱신한다.
> 단순 기능 구현·버그 수정은 STATUS.md / ROADMAP.md 로 간다.
> 결정을 바꿀 때는 "핵심 설계 결정" 섹션에 기존 항목을 삭제하지 말고 상태를 `변경됨(YYYY-MM-DD)` 으로 남겨 이력을 보존한다.

---

## 1. 한 눈에 보는 구성

```
[Client: React + Vite]                [Supabase Auth]
        │                                  │
        │ Google OAuth / ES256 JWT ────────┘
        │
        │  HTTP /api/*  +  WS /ws
        ▼
[Backend: Go + Fiber v2]
  ├── HTTP handler (Fiber ctx)
  ├── WebSocket hub (gofiber/websocket/v2)
  ├── RoomService  ─ in-memory + DB fallback
  ├── GameManager  ─ 마피아 게임 생명주기
  ├── PhaseManager ─ day/vote/night 전이
  └── AI Manager   ─ Anthropic Claude API
        │
        ▼
[PostgreSQL (pgx/v5)]     [Redis (go-redis/v9)]
  rooms                     presence · rate limit
  game_results
  game_states
  ai_histories
  users
```

### 기술 스택

| 계층 | 선택 | 대안 |
|------|------|------|
| Backend | Go 1.x + Fiber v2 | — |
| WebSocket | `github.com/gofiber/websocket/v2` | gorilla/websocket |
| DB | PostgreSQL 16 + pgx/v5 | — |
| Cache/Presence | Redis 7 | — |
| AI | Anthropic Claude (`anthropic-sdk-go`) | OpenAI (미채택) |
| Frontend | React 18 + TypeScript + Vite | — |
| 상태관리 | Zustand | Redux (과함) |
| Auth | Supabase (Google OAuth, ES256 JWK) | 자체 구현 (보류) |

### 포트

| 서비스 | 포트 | 비고 |
|--------|------|------|
| Backend | **8080** | `3000`은 `kubectl port-forward` 선점 회피 (`3ffc55c`) |
| Vite Dev | 5173 | `/api`·`/ws` 프록시 → `:8080` (`0909d3f`) |
| Postgres | 5432 | docker-compose |
| Redis | 6379 | docker-compose |

---

## 2. 백엔드 레이어

```
cmd/server/main.go          진입점, 의존성 주입
config/                     TOML 로더
internal/
  domain/
    entity/                 Room, Player, Game (sync.RWMutex)
    dto/                    HTTP 요청/응답 (JSON snake_case)
  games/mafia/              PhaseManager, GameState, roles
  platform/
    room.go                 RoomService (in-mem + DB fallback)
    handler.go              HTTP 핸들러 (Fiber)
    game_manager.go         게임 생명주기
    ws/hub.go               WebSocket 허브
  ai/                       PersonaPool, AI Manager
  repository/               pgx repo (rooms, game_states, ai_histories, users)
migrations/                 000001..000006 (golang-migrate 포맷)
```

### HTTP ↔ DTO 계약
- 요청/응답 JSON 키는 **모두 snake_case** (예: `host_id`, `is_alive`)
- 프론트 `types.ts` 타입과 1:1 대응 (경계면 drift 감시 대상)
- Handler 필드는 concrete가 아니라 **interface**(UserStore, GameResultStore) — 테스트에서 nil 또는 mock 주입 가능

### 동시성
- `entity.Room` 은 `sync.RWMutex` 로 보호. 외부에서 필드 직접 접근 금지, getter 사용
- `RoomService` 는 `db == nil` 일 때 **인메모리만** 사용 → 테스트가 외부 의존성 없이 가능
- WS 메시지 rate limit: **200ms** (이전 500ms에서 조정)

---

## 3. 프론트엔드 레이어

```
src/
  api.ts              HTTP 클라이언트 (fetch)
  types.ts            백엔드 DTO 1:1 대응 타입
  store/
    gameStore.ts      Zustand, WS 연결·게임 상태
    authStore.ts      Supabase 세션, displayName
  pages/              LandingPage, LobbyPage, RoomPage, ProfilePage
  components/         WaitingRoom, GameRoom, ChatInput, PhaseHeader …
```

### WS payload 계약 (프론트 → 백엔드)
액션 전송 시 **중첩 구조** 사용:
```ts
sendAction('chat',  { chat:  { message } })
sendAction('vote',  { vote:  { target_id } })
sendAction('kill',  { night: { action_type: 'kill', target_id } })
```
플랫한 `{ message }` 로 보내면 백엔드 `dto.ActionRequest.Chat.Message` 로 파싱되지 않아 조용히 버려진다 (회귀 방지 필수).

---

## 4. 핵심 설계 결정 (Decision Log)

각 항목은 **결정 / Why / How**를 명시한다. 번복 시 `변경됨(YYYY-MM-DD)` 라벨을 붙이고 이유를 남긴다.

### 4.1 Backend 포트 3000 → 8080 (2026-04-22 ~)
- **결정:** 서버 포트를 8080으로 고정
- **Why:** 로컬 `kubectl port-forward` 가 3000을 선점하는 일이 잦아 개발 흐름이 끊김
- **How:** `config.toml.[server].port = 8080`, Vite `/api`·`/ws` 프록시도 8080으로 정렬
- **참고 커밋:** `3ffc55c`, `0909d3f`

### 4.2 JWT 검증: HS256 → ES256 (Supabase JWK) (2026-04-23 ~)
- **결정:** Supabase 발급 JWT 는 ES256 공개키로 검증
- **Why:** Supabase 신규 프로젝트는 대칭키(HS256)가 아닌 비대칭(ES256) 로 서명. HS256 검증 유지 시 로그인 실패
- **How:** `config.toml` 에 JWK `x`, `y` 좌표 저장 → ECDSA 퍼블릭 키 복원
- **참고 커밋:** `73b69de`

### 4.3 RoomService: in-memory + DB fallback
- **결정:** 방 상태는 인메모리가 primary, DB는 recovery용
- **Why:** 실시간 게임은 지연 민감. DB를 primary로 두면 모든 이벤트가 왕복 쿼리
- **How:** `RoomService.ListPublic()` 은 인메모리만 반환 → stale DB record 노출 방지. 프로세스 재시작 시 DB에서 복구
- **트레이드오프:** 단일 인스턴스 가정. 수평 확장 시 Redis pub/sub 또는 sticky routing 필요 → ROADMAP Tier 3

### 4.4 Handler 의존성 = interface (concrete 금지)
- **결정:** Handler 는 `UserStore`, `GameResultStore` **인터페이스** 를 주입
- **Why:** 테스트에서 실제 DB 없이 mock 주입 → `handler_test.go` 가 가볍고 빠름
- **How:** `internal/platform/handler.go` 상단에 인터페이스 정의, `main.go` 에서 concrete repo 주입, 테스트는 `mockUserStore` 사용
- **남은 과제:** `GameResultStore.GetStatsByPlayerID` 가 `repository.PlayerStats` 를 반환 → handler 가 repository 패키지에 여전히 의존. `domain/dto` 로 옮기면 완전 분리

### 4.5 닉네임: 방 입장 시 입력 제거, 프로필 기반 자동 resolution
- **결정:** 사용자는 프로필의 display_name 을 씀, 방 입장 때 닉네임 입력 UI 없음
- **Why:** 매 방마다 닉네임을 적는 건 마찰. 재로그인 시에도 동일 display_name 유지
- **How:** `resolvePlayerFull` 이 user row에서 display_name 로드, 없으면 자동 생성 (`a17e38d`, `4d70600`)

### 4.6 AI 모델 분리 (default / reasoning)
- **결정:** Haiku 4.5 기본, Sonnet 4.6 은 추론 집약 타이밍만
- **Why:** 비용/지연 균형. 일반 대화는 Haiku 충분, 추리·투표 결정은 Sonnet
- **How:** `config.toml.[ai].model_default` / `model_reasoning` 분리

### 4.7 Fiber Context + sync.Pool 최적화 (방향성)
- **결정:** 핸들러는 `*fiber.Ctx` 만 통해 I/O, 반복 할당 구조체는 `sync.Pool` 로 재사용
- **Why:** Fiber 의 제로-얼로케이션 장점을 살리고 GC 부담 감소
- **How:** 현 상태에서는 아직 모든 경로에 적용되지 않음 → ROADMAP Tier 2 로 점진 적용
- **상태:** 부분 적용

### 4.8 운영 배포 스택 (설계 목표 · 2026-04-24 기준 미구현)
- **결정:** Railway + Doppler + k8s 조합. GitHub main push → 자동 배포
- **Why:**
  - Railway: 관리형 배포로 인프라 오버헤드 최소화
  - Doppler: secret 주입 (키를 코드/yaml 에 박지 않음)
  - k8s: 로컬/운영 매니페스트 공유, 재현 가능한 런타임
  - GitHub 자동 배포: 배포 절차 수동화 제거
- **How (예정):** `.github/workflows/` + `infra/k8s/*.yaml` + Doppler CLI 로 env 주입, 환경변수는 **파일 읽기** 방식 (key-by-key YAML 수정 불필요)
- **상태:** ❌ 아직 없음. 현재는 로컬 `docker-compose.yml` (postgres + redis) 만 존재

### 4.9 환경변수 주입 정책 (MANDATORY)
- **결정:** secret 은 **파일 읽기** 로 주입한다. 키 이름을 yaml/파이프라인에 하드코딩하지 않는다
- **Why:** 환경변수 추가마다 yaml/파이프라인을 바꾸고 싶지 않다. 또한 secret 이 코드/이미지에 섞이는 사고를 원천 차단
- **How:** Doppler → 런타임이 secret file 을 mount → 앱이 파일을 읽어 config 구성. `backend/config.toml` 같은 평문 키 파일은 **절대 커밋 금지**

### 4.10 Docker 이미지 빌드 최적화 (기준)
- **결정:** multi-stage 빌드, 베이스는 `scratch` 또는 `distroless`
- **Why:** 불필요한 빌드 단계·도구체인이 이미지에 남으면 배포 시간과 공격면이 동시에 증가
- **How:** Go 바이너리는 `CGO_ENABLED=0` 정적 빌드 → scratch 단계에 바이너리만 복사. `.dockerignore` 로 `node_modules`, `.git`, `config.toml` 제외

### 4.11 검색 최적화 (SEO, 방향성)
- **결정:** 랜딩/공개 페이지는 메타태그·구조화 데이터·sitemap 를 갖춘다
- **Why:** 방 목록·플레이 유입을 검색에서 확보
- **How:** `react-helmet-async` 또는 Vite 빌드 시 정적 메타 주입. ROADMAP Tier 2 로 점진 추가

### 4.12 제품 원칙: Unit Economics 렌즈 (2026-04-24 ~)
- **결정:** 모든 기능 결정은 **비용 · 수익 · 리텐션 · 인간 밀도** 4축으로 판단한다
- **Why:** 이 플랫폼은 Claude API 비용을 광고 수익으로 감당하는 구조다. 비용이 수익을 초과하면 성장 자체가 손실을 확대한다. 재미가 부족하면 유입은 있어도 리텐션이 0이고, 광고 노출 기회가 사라져 수익 구조가 무너진다. 따라서 "재미"와 "수익"은 같은 제약 안에서 동시에 최적화되어야 하며, 어느 한쪽만 고려한 결정은 지속 불가능하다
- **How:**
  - 기능 제안 시 scorecard 작성: Impact / Effort / ROI + 각 축 영향 한 줄씩
  - ROADMAP 의 Tier 배치에도 동일 렌즈 적용 (Tier 1 = 4축 중 둘 이상에 즉각 positive)
  - 구체 기능 spec 은 `docs/superpowers/specs/*.md` 에 축별 영향을 섹션으로 기록
  - Phase A 가 측정 기반(`game_metrics` 테이블 + 로그 태깅) 을 구축하여 이후 Phase의 ROI 를 실측 가능하게 만든다
- **연결 문서:** CLAUDE.md "Product Principle: Unit Economics 렌즈" 섹션

---

## 5. 데이터 모델 요약

| 테이블 | 용도 |
|--------|------|
| `rooms` | 방 상태 스냅샷 (프로세스 재시작 recovery 용) |
| `game_states` | 게임 단위 상태 직렬화 |
| `game_results` | 승패·역할 이력 (통계 소스) |
| `ai_histories` | 페르소나·대화 맥락 (리콜 용) |
| `users` | Supabase uid ↔ display_name |

외래키 제약은 `000005_remove_fk_constraints` 로 제거됨 — 분산 환경에서 재삽입 순서에 유연성 부여.

---

## 6. 알려진 구조적 제약

1. **단일 백엔드 인스턴스 가정** — RoomService 인메모리 상태는 수평 확장 불가. Redis pub/sub 또는 Consistent hashing 필요
2. **Frontend 테스트 0건** — React Testing Library·Playwright 미도입
3. **`internal/repository` 통합 테스트 없음** — DB 계약 검증 부재
4. **`internal/ai` 테스트 없음** — PersonaPool/Manager 유닛 테스트 부재
5. **Secret 관리** — `config.toml` 평문 트래킹 상태. 즉시 secrets 분리 필요 (STATUS 의 "알려진 결함" 참조)

이들은 ROADMAP 의 Tier 2/3 에 대응 항목이 있다.

---

## 7. 참고 문서

- 현 구현 상태 · 체크리스트: [`docs/STATUS.md`](./STATUS.md)
- 다음 작업 우선순위: [`docs/ROADMAP.md`](./ROADMAP.md)
- 기능별 스펙: [`docs/superpowers/specs/`](./superpowers/specs/)
- 기능별 계획: [`docs/superpowers/plans/`](./superpowers/plans/)

---

## 변경 이력

| 날짜 | 변경 내용 | 사유 |
|------|----------|------|
| 2026-04-24 | 초기 작성 | 하네스 엔지니어링 — 아키텍처 결정을 파일로 고정 |
