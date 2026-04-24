# ROADMAP.md

다음 할 일을 **우선순위 tier**로 분류한다.
**마지막 업데이트:** 2026-04-24 (Tier 1 secret 분리 기본선 완료)

> **업데이트 규칙 (MANDATORY)**
> - 완료된 항목은 이 파일에서 제거하고, STATUS.md 체크리스트에 ✅ 로 옮긴다
> - 새 요구사항은 tier 판단을 달아 맨 위에 추가한다
> - "현재 추천 다음 작업"은 사용자가 "뭐부터 할까?"라고 물을 때 즉답 가능하도록 **항상 최신 상태** 로 유지한다

## Tier 정의

| Tier | 의미 | 예 |
|------|------|-----|
| **Tier 1** | 차단성 · 보안 · 사용자 경험 치명 | 키 유출, 크래시, 결제 오류 |
| **Tier 2** | 운영/품질 기반 · 다음 릴리즈 | 배포 파이프라인, 관측성, 커버리지 |
| **Tier 3** | 성장·최적화·장기 투자 | SEO, 수평 확장, 실험 기능 |

---

## 현재 추천 다음 작업

> 프로젝트는 아직 GitHub 등 원격에 푸시된 적이 없다. 외부 노출 리스크는 0이므로,
> 키 rotation 보다 "푸시 전에 로컬 히스토리만 깨끗이 정리" 가 자연스럽다.

1. **Tier 1 · 원격 푸시 전략 결정** — 아래 중 하나로 확정
   - A. `git filter-repo` 로 과거 커밋의 `backend/config.toml`, `frontend/.env.development`, `frontend/.env.production` 경로 제거 후 remote 생성
   - B. 기존 히스토리를 버리고 `git init` 으로 새 저장소 시작
   - C. 원격 없이 당분간 로컬 개발만 지속 (푸시 예정 시점에 다시 판단)
2. **Tier 2 · backend/frontend Dockerfile 작성** (multi-stage, scratch/distroless)
3. **Tier 2 · GitHub Actions CI** (`go test ./...`, `vite build`, `tsc --noEmit`)
4. **Tier 2 · Railway 프로젝트 설정 + Doppler 연결**

---

## Tier 1 · 차단성

### T1-1. Secret 관리 정립
- [x] 루트 `.gitignore` 작성 (`abc86c2`)
- [x] `.dockerignore` 작성 (`abc86c2`)
- [x] `backend/config.toml` · `frontend/.env.development` · `.env.production` git 트래킹 해제
- [x] `backend/config.example.toml` · `frontend/.env.example` 템플릿 제공
- [ ] **푸시 전 히스토리 정리** — 아직 원격에 올린 적 없으므로 rotation 보다 히스토리 정리가 저렴
      - A안: `git filter-repo --path backend/config.toml --path frontend/.env.development --path frontend/.env.production --invert-paths`
      - B안: 새 저장소로 `git init` 후 현 상태만 커밋
- [ ] 필요 시 키 rotation (옵션 C 즉 "그대로 푸시" 선택 시 필수, A/B 선택 시 생략 가능)
- [ ] 운영 환경 secret 주입 경로 확정 (Doppler secret file mount, ARCHITECTURE 4.9)
- **근거:** ARCHITECTURE 4.9 · 사용자 규칙 "키가 절대 노출 안 됨"

### T1-2. 미커밋 변경사항 정리
- [x] `handler.go` 인터페이스 추출 + 테스트 9개 커밋 (`231e57f`)

### T1-3. WaitingRoom 최소 인원 룰 재확인
- [ ] 현재 `canStart = players.length >= 1` (1인 시작 가능). 의도된 디버그 조건인지 검토, 프로덕션은 6인 전체 또는 최소 2 사람 요구가 자연스러움
- [ ] 룰이 바뀌면 회귀 테스트 추가

---

## Tier 2 · 운영/품질 기반

### T2-1. Docker 빌드
- [ ] `backend/Dockerfile` — multi-stage, `CGO_ENABLED=0`, `scratch` 또는 `distroless` 베이스
- [ ] `frontend/Dockerfile` — vite build → nginx static serve (또는 Railway 정적 호스팅)
- [ ] `.dockerignore` 로 빌드 컨텍스트 최소화
- [ ] 이미지 사이즈 목표: backend < 25MB, frontend < 30MB

### T2-2. Kubernetes 매니페스트 (로컬·운영 공용)
- [ ] `infra/k8s/base/` — deployment, service, ingress, configmap, secret(from file)
- [ ] `infra/k8s/overlays/local/` vs `overlays/prod/` (kustomize)
- [ ] 로컬 배포 자동화 스크립트 (`make deploy-local`)

### T2-3. 배포 자동화 (Railway + Doppler + GitHub Actions)
- [ ] Railway 프로젝트 생성, `railway.json` 혹은 Nixpacks 설정
- [ ] Doppler 프로젝트·env 설정, `doppler secrets substitute` 또는 secret file mount 방식 결정
- [ ] `.github/workflows/deploy.yml` — main push → test → build → railway deploy
- [ ] 환경변수는 **파일 읽기** 로 주입 (yaml 에 키 이름 하드코딩 금지, ARCHITECTURE 4.9)

### T2-4. CI 테스트 게이트
- [ ] `.github/workflows/ci.yml` — `go test ./...`, `vite build`, `tsc --noEmit`
- [ ] PR 에 필수 체크로 연결

### T2-5. 테스트 커버리지 확장
- [ ] `internal/repository` 통합 테스트 (testcontainers 또는 pg_tmp)
- [ ] `internal/ai` 유닛 테스트 (Anthropic client mock)
- [ ] Frontend: Vitest + React Testing Library 도입 · LandingPage/LobbyPage 스모크부터

### T2-5b. 경계면 drift 정리 (2026-04-24 QA 발견)
- [ ] **D1 (Critical)** `game_over` 의 `all_humans_left` path(hub.go:156)가 `{reason}` 만 전송 → 프론트 `GameOverResult` 필드 전부 undefined. 같은 shape 으로 채우거나 별도 이벤트 타입(`game_aborted`)로 분리
- [ ] **D2** `max_humans` 프론트 Room 타입에 없음 + initial_state.room 에도 누락. types.ts / hub.go 양쪽 보강
- [ ] **D3** HTTP `RoomResponse.join_code` omitempty vs 프론트 Room `join_code: string` 필수. 프론트를 optional로 낮추거나 백엔드 omitempty 제거
- 근거: `_workspace/qa_report.md`

### T2-6. Fiber 성능 패턴 적용
- [ ] 반복 할당 구조체 `sync.Pool` 도입 (WS 브로드캐스트 payload, DTO 빌더)
- [ ] Context 전파 정리 (`c.Context()` 일관 사용, 핸들러 밖 고루틴 cancel 연결)

### T2-7. 관측성 기본선
- [ ] 구조화 로그(zap) 필드 표준화 (`room_id`, `player_id`, `event`)
- [ ] HTTP 요청 로깅 미들웨어
- [ ] `/healthz`, `/readyz` 엔드포인트

---

## Tier 3 · 성장·최적화

### T3-1. SEO · 유입
- [ ] 랜딩 메타 태그(`og:*`, `twitter:*`), `index.html` 정적 주입
- [ ] `sitemap.xml`, `robots.txt`
- [ ] 구조화 데이터(JSON-LD: `WebSite`, `Game`)
- [ ] 초기 렌더 속도 예산(Vite chunk split, 폰트 preload)

### T3-2. 수평 확장 준비
- [ ] Redis pub/sub 으로 방 상태 브로드캐스트 분산
- [ ] RoomService 상태 → Redis 또는 sticky routing 설계
- [ ] ARCHITECTURE 4.3 트레이드오프 해소

### T3-3. 실시간 presence
- [ ] `docs/superpowers/plans/2026-04-09-realtime-player-presence.md` 구현

### T3-4. 광고/수익화
- [ ] `docs/superpowers/plans/2026-04-09-ad-revenue.md` 구현

### T3-5. UX 애니메이션
- [ ] `docs/superpowers/plans/2026-04-09-ux-animation.md` 구현

### T3-6. DTO 의존성 분리
- [ ] `repository.PlayerStats`, `PlayerGameRecord` → `domain/dto` 이동
- [ ] handler 가 repository 패키지 미참조 상태 달성

---

## JD 매핑 (선택)

> 이 프로젝트가 어떤 JD 요구사항에 매핑되는지 기록. 포트폴리오 프레이밍용.

| 요구사항 | 대응 작업 |
|---------|----------|
| Go 백엔드 · 동시성 · 실시간 | Phase 1 · T2-6 · T3-2 |
| React + TypeScript | Phase 1~2 · T2-5 |
| 인증·인가 | Phase 2 Supabase ES256 |
| 테스트 작성 | Phase 3 · T2-4 · T2-5 |
| Docker / k8s / 배포 자동화 | T2-1 · T2-2 · T2-3 |
| AI 통합 | AI Manager · T2-5 |
| 관측성 | T2-7 |

---

## 참고

- 현 구현 상태: [`docs/STATUS.md`](./STATUS.md)
- 아키텍처 설계 결정: [`docs/ARCHITECTURE.md`](./ARCHITECTURE.md)
- 기능별 스펙 / 계획: [`docs/superpowers/specs/`](./superpowers/specs/), [`docs/superpowers/plans/`](./superpowers/plans/)
