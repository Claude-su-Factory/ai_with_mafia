# STATUS.md

현재 어디까지 구현됐는가를 한 눈에 본다.
**마지막 업데이트:** 2026-04-24

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
- ❌ `Dockerfile` (backend · frontend)
- ❌ `.dockerignore`
- ❌ 루트 `.gitignore`
- ❌ Kubernetes 매니페스트 (로컬·운영 공용)
- ❌ Doppler secret 주입 설정
- ❌ Railway 프로젝트 연결
- ❌ GitHub Actions (main push → 자동 배포)
- ❌ 배포 자동화 (로컬에서도 단일 커맨드)

### Phase 5 — 성장·SEO·수익화
- ❌ SEO 메타·sitemap
- ❌ 광고 (계획만 존재: `docs/superpowers/plans/2026-04-09-ad-revenue.md`)
- ❌ 리얼타임 presence 표시 (계획만 존재)
- ❌ UX 애니메이션 (계획만 존재)

---

## 서비스 상태 (로컬 기준)

| 항목 | 상태 | 비고 |
|------|------|------|
| Backend 빌드 | ✅ `go build ./...` OK | |
| Backend 테스트 | ✅ `go test ./...` 전부 통과 | 73 tests |
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

- **2026-04-24** · 하네스 엔지니어링: `docs/STATUS.md`, `docs/ROADMAP.md`, `docs/ARCHITECTURE.md` 신설, CLAUDE.md에 문서/배포/스킬 라우팅 규칙 추가
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

### 🔴 치명적 (즉시 조치 필요)

1. **`backend/config.toml`에 API 키가 평문으로 git 트래킹됨**
   - Anthropic API key, Supabase JWK가 커밋 히스토리에 노출
   - 루트 `.gitignore` 부재 → 새 secret 파일도 자동 커밋 위험
   - **조치:** 키 rotation → secret 파일 분리 → `.gitignore` 작성 → 히스토리에서 제거(`git filter-repo`) 고려
   - **ROADMAP Tier 1** 에 대응 항목

### 🟡 중요 (작업 중·다음 릴리즈)

2. **미커밋 변경사항(2026-04-17 기준)** — `handler.go` 인터페이스 추출 + 신규 테스트 9개. 커밋 필요
3. **`GameResultStore` 타입 결합도** — handler가 `repository.PlayerStats` / `repository.PlayerGameRecord` 를 여전히 참조. `domain/dto` 로 이동 시 완전 분리
4. **Frontend 테스트 0건** — 회귀 방어선 부재
5. **DB 통합 테스트 부재** — `internal/repository` migration + 쿼리 계약 검증 없음
6. **AI 매니저 테스트 부재** — `internal/ai` PersonaPool·Manager 동작 미검증

### 🟢 개선 여지

7. **Fiber sync.Pool 적용 범위** — 모든 handler/DTO 경로에 적용 안 됨. 점진 확대
8. **SEO 메타** — 랜딩/공개 페이지 메타 태그·sitemap 없음

---

## 참고

- 다음 할 일 우선순위: [`docs/ROADMAP.md`](./ROADMAP.md)
- 아키텍처 설계 결정: [`docs/ARCHITECTURE.md`](./ARCHITECTURE.md)
- 기능별 스펙 / 계획: [`docs/superpowers/specs/`](./superpowers/specs/), [`docs/superpowers/plans/`](./superpowers/plans/)
