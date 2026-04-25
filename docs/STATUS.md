# STATUS.md

현재 어디까지 구현됐는가를 한 눈에 본다.
**마지막 업데이트:** 2026-04-24 (Phase A 구현 완료, 검증 대기)

> **업데이트 규칙 (MANDATORY)**
> 기능 구현을 완료하면 이 파일을 반드시 갱신한다. 업데이트 없이는 완료로 간주하지 않는다.
> 1. Phase 체크리스트의 해당 항목을 ✅로 이동
> 2. "최근 변경 이력" 맨 위에 한 줄 추가 (`YYYY-MM-DD · 간결한 요약 · 참조 커밋/PR`)
> 3. 맨 위의 "마지막 업데이트" 날짜 갱신
> 4. "알려진 결함"에서 해결된 항목 제거, 새로 발견한 건 추가

---

## Phase 체크리스트

### Phase 1 — 핵심 게임 루프
- ✅ 방 생성/입장/퇴장 (RoomService, in-memory + DB fallback)
- ✅ WebSocket 허브 · 클라이언트 연결 관리
- ✅ 마피아 6인 고정 룰 (마피아 2, 경찰 1, 시민 3)
- ✅ PhaseManager (day_discussion / day_vote / night)
- ✅ 역할 배정 · 밤 행동 · 투표
- ✅ 게임 결과 기록 (`game_results` 테이블)
- ✅ AI 플레이어 자동 참여 (인원 부족 시 보충)
- ✅ 채팅 payload 계약 `{ chat: { message } }` 정립

### Phase 2 — 인증 · 프로필
- ✅ Supabase Google OAuth 연동
- ✅ ES256 JWT 검증 (Supabase JWK)
- ✅ `users` 테이블 + display_name 유지
- ✅ 방 입장 시 닉네임 입력 제거, 프로필 기반 자동 해결
- ✅ `/profile` 페이지 (통계·역할별 승률·최근 게임)
- ✅ ProfilePage error / loading / UTC 날짜 핸들링

### Phase 3 — 테스트 · 품질
- ✅ `internal/games/mafia` 유닛 테스트 (25개)
- ✅ `internal/platform` 핸들러 테스트 (48개) — mockUserStore, nil userRepo, 401/403/404 경계 포함
- ❌ `internal/repository` 통합 테스트
- ❌ `internal/ai` 테스트
- ❌ Frontend 컴포넌트·E2E 테스트
- ❌ CI 자동 테스트 실행 (GitHub Actions)

### Phase 4 — 운영 배포
- ✅ 로컬 `docker-compose.yml` (postgres + redis)
- ✅ 루트 `.gitignore` (secret · 빌드 · OS noise · *.zip 차단)
- ✅ 루트 `.dockerignore` (이미지 컨텍스트 최소화)
- ✅ `backend/config.example.toml` · `frontend/.env.example` 템플릿
- ✅ `backend/config.toml` · `frontend/.env.development` · `.env.production` git 트래킹 해제
- ❌ `Dockerfile` (backend · frontend)
- ❌ Kubernetes 매니페스트 (로컬·운영 공용)
- ❌ Doppler secret 주입 설정
- ❌ Railway 프로젝트 연결
- ❌ GitHub Actions (main push → 자동 배포)
- ❌ 배포 자동화 (로컬에서도 단일 커맨드)

### Phase 5 — 성장·SEO·수익화
- ❌ SEO 메타·sitemap
- ❌ 리얼타임 presence 표시 (계획만 존재)
- ❌ UX 애니메이션 (계획만 존재)

### Phase A — Unit Economics Foundation (2026-04-24 구현 완료, 검증 대기)
스펙: `docs/superpowers/specs/2026-04-24-phase-a-unit-economics-foundation-design.md` · 플랜: `docs/superpowers/plans/2026-04-24-phase-a-unit-economics-foundation.md`
- ✅ `game_metrics` 테이블 + migration 000007
- ✅ `GameMetricsRepository` (Create / Finalize / AddAIUsage / IncrementAdImpression / RecordQuickMatch) + nil-pool 안전
- ✅ **Game lifecycle hooks**: `GameManager.start` → `Create`, `EventGameOver` → `Finalize`. `game_results.id` ↔ `game_metrics.game_id` UUID 통일 (T21 · `503e9ea`)
- ✅ AI Cost Optimizer: max_tokens chat(160)/decision(20) 분리, Anthropic prompt cache, stop_reason 관측 훅
- ✅ Quick Match: `POST /api/rooms/quick` join-or-create + latency metric + 프론트 `빠른 참가` 버튼
- ✅ Ad Integration: `POST /api/metrics/ad` + Redis-backed rate limiter (30 req/min/IP) + `AdBanner` IntersectionObserver + Lobby/Waiting/Result 3-surface 배치
- ✅ Fail-safe regression lock: 모든 AI 기권해도 게임이 다음 phase 로 진행 (`phases_test.go`)
- 🟠 **검증 대기**: `_workspace/phase-a-verification.md` 의 6개 정량 기준을 로컬에서 실측 후 pass 확인 필요

---

## 서비스 상태 (로컬 기준)

| 항목 | 상태 | 비고 |
|------|------|------|
| Backend 빌드 | ✅ `go build ./...` OK | |
| Backend 테스트 | ✅ `go test ./...` 전부 통과 | ~95+ tests (Phase A 후) |
| Frontend 빌드 | ✅ Vite 빌드 OK | `dist/` 생성 확인 |
| Frontend 테스트 | ❌ 테스트 파일 0 | |
| Postgres · Redis | ✅ docker-compose 실행 가능 | `docker compose up` |
| 운영 배포 URL | ❌ 없음 | 미구축 |

### 포트

| 서비스 | 포트 |
|--------|------|
| Backend | 8080 |
| Vite Dev | 5173 (프록시 `/api`, `/ws` → 8080) |
| Postgres | 5432 |
| Redis | 6379 |

---

## 최근 변경 이력 (최신순)

- **2026-04-25** · 불필요한 `.md` 정리: `openspec/` (99 파일, 옛 워크플로우 도구 산출물), `ui/` (4 파일, frontend 와 별도인 옛 디자인 레퍼런스), `backend/TODOS.md` 삭제. TODOS.md 의 미해결 항목 2건은 ROADMAP T2-7b (Redis pub/sub 재연결) / T2-8 (마피아 합의 실패 UX) 으로 이전. 코드 영향 0건
- **2026-04-25** · DB 스키마 관리 정책 변경: `golang-migrate` 폐기, `backend/migrations/` 디렉토리 삭제, 자동 migration 코드 제거. 전체 DDL 을 루트 `README.md` "DB Schema" 섹션에 통합. **FK 금지** 규칙을 CLAUDE.md / ARCHITECTURE §4.14 에 영구 등록 (운영 편의 + 분산 환경 무결성 비용 회피). 사람이 `psql` 로 직접 스키마 적용
- **2026-04-24** · `503e9ea` Phase A final-review 대응 (T21): game lifecycle `Create`/`Finalize` 훅 연결, `game_results.id` ↔ `game_metrics.game_id` UUID 통일, runbook §5 쿼리 수정(ended_at→created_at, game_id join), I1 (`maxTokensFor` nil-guard) / I2 (`errors.Is(redis.Nil)`) minor fix
- **2026-04-24** · **Phase A — Unit Economics Foundation 구현 완료** (20 TDD tasks, commits `0e83386`~`2ed76c1`): game_metrics 스키마/Repo, AI prompt cache + max_tokens split + stop_reason 훅, Quick Match join-or-create, Redis-backed ad rate limiter, 3-surface AdBanner. 6개 정량 기준 검증은 로컬 runbook (`_workspace/phase-a-verification.md`) 로 대기
- **2026-04-24** · 경계면 drift D1~D3 TDD 해결: `buildAbortedGameOverPayload` + `buildInitialStateRoomPayload` 헬퍼 추출(+ 유닛 테스트 10건), hub.go가 이를 사용, 프론트 GameOverResult/Room 타입 동기화, ResultOverlay aborted 분기 추가
- **2026-04-24** · 코드 리뷰 피드백 반영: `respondPlayerErr` DB 에러 → 500 분류 수정, `resolvePlayerFull` 500 경로 테스트 추가 (+3 tests), `backend/server` 바이너리 untrack, `.dockerignore` markdown negate 단순화, `.gitignore` 중복 제거, CLAUDE.md 현재 상태 갱신
- **2026-04-24** · `3c0db5f` docs: 원격 없음 반영 — secret을 "치명적"에서 "푸시 전 필수"로 재분류
- **2026-04-24** · `09260d1` docs: STATUS/ROADMAP을 secret 분리 후 상태로 동기화
- **2026-04-24** · `abc86c2` secret 분리: `.gitignore`/`.dockerignore` 작성, `config.toml`·`.env.development`·`.env.production` 트래킹 해제, `config.example.toml`·`.env.example` 추가
- **2026-04-24** · `231e57f` Handler에 UserStore/GameResultStore 인터페이스 추출 + 테스트 9개 (2026-04-17 기준 미커밋 작업 정리)
- **2026-04-24** · `139d97b` 하네스 엔지니어링: `docs/STATUS.md`, `docs/ROADMAP.md`, `docs/ARCHITECTURE.md` 신설, CLAUDE.md에 문서/배포/스킬 라우팅 규칙 추가
- **2026-04-22** · `0909d3f` Vite proxy target 8080 정렬
- **2026-04-22** · `3ffc55c` backend port 3000 → 8080
- **2026-04-23** · `73b69de` JWT HS256 → ES256 (Supabase JWK)
- **2026-04-21** · `d237aff` ProfilePage re-fetch 전 fetchError 리셋
- **2026-04-21** · `fd0e4eb` ProfilePage error state · UTC 날짜 · 서버 응답 기반 저장
- **2026-04-20** · `cd50e03` ProfilePage 구현 (통계 · 역할별 분석 · 게임 히스토리)
- **2026-04-20** · `511e2cf` `/profile` 라우트 추가
- **2026-04-20** · `54f7cb6` 방 입장 닉네임 입력 제거 · 프로필 버튼 추가
- **2026-04-19** · `fe493e4` authStore 경쟁 상태 · stale state · Content-Type · 204 핸들링
- **2026-04-19** · `a17e38d` handler_test 새 시그니처 반영 · 프로필 401 테스트 추가
- **2026-04-19** · `f453850` 프로필 엔드포인트 · `resolvePlayerFull` (자동 닉네임)
- **2026-04-19** · `d2b1457` `GameResultRepository.GetStatsByPlayerID` · `GetRecentGamesByPlayerID`
- **2026-04-18** · `4d70600` 재로그인 시 display_name 유지

---

## 알려진 결함 (Known Issues)

### 🟠 푸시 전 필수 (현재는 로컬 한정)

1. **로컬 `.git` 히스토리에 Anthropic API key · Supabase anon key 가 남아 있음**
   - 원격(GitHub) 에 아직 올린 적 없음 → **외부 노출은 현재 0**
   - 현재 시점 트래킹은 해제 완료 (`abc86c2`)
   - **푸시 직전에 반드시 처리**:
     - 옵션 A (권장): `git filter-repo` 또는 BFG 로 과거 커밋의 secret 경로 제거 후 remote 생성
     - 옵션 B: 현 저장소를 버리고 `git init` 으로 새로 시작 (커밋 이력 포기)
     - 옵션 C: 그대로 푸시하고 즉시 키 rotation (비추)
   - 혼자 개발 중이고 아직 clone 공유 안 했다면 옵션 A 또는 B 가 가장 저렴
   - **ROADMAP Tier 1** 참조

### 🟡 중요 (작업 중·다음 릴리즈)

2. **`GameResultStore` 타입 결합도** — handler가 `repository.PlayerStats` / `repository.PlayerGameRecord` 를 여전히 참조. `domain/dto` 로 이동 시 완전 분리
3. **Frontend 테스트 0건** — 회귀 방어선 부재
4. **DB 통합 테스트 부재** — `internal/repository` migration + 쿼리 계약 검증 없음
5. **AI 매니저 테스트 부재** — `internal/ai` PersonaPool·Manager 동작 미검증

### 🟢 개선 여지

7. **Fiber sync.Pool 적용 범위** — 모든 handler/DTO 경로에 적용 안 됨. 점진 확대
8. **SEO 메타** — 랜딩/공개 페이지 메타 태그·sitemap 없음

---

## 참고

- 다음 할 일 우선순위: [`docs/ROADMAP.md`](./ROADMAP.md)
- 아키텍처 설계 결정: [`docs/ARCHITECTURE.md`](./ARCHITECTURE.md)
- 기능별 스펙 / 계획: [`docs/superpowers/specs/`](./superpowers/specs/), [`docs/superpowers/plans/`](./superpowers/plans/)
