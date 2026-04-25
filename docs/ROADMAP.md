# ROADMAP.md

다음 할 일을 **우선순위 tier**로 분류한다.
**마지막 업데이트:** 2026-04-24 (Phase A 구현 완료 · 검증 대기)

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

> Phase A (Unit Economics Foundation) 가 방금 구현 완료. 다음은 실측 데이터 수집 + Phase B 브레인스토밍.

1. **Phase A 검증 runbook 실행** — `_workspace/phase-a-verification.md` 6개 정량 기준 로컬 실측
   - Prompt cache hit rate ≥ 70%
   - Ad impressions 3 슬롯 모두 관측
   - Quick match p95 ≤ 3s, 성공률 ≥ 95%
   - Metric coverage: `missing = 0`
   - 2-Pod rate limiter 429 검증
2. **Phase B 브레인스토밍** — 검증 데이터를 바탕으로 B/C/D 중 다음 대상 확정. 후보: 보상형 광고 + 초대 링크, 또는 새 역할 (Phase C)
3. **Tier 1 · 원격 푸시 전략 결정** (보류 가능 — 외부 노출 0)
4. **Tier 2 · backend/frontend Dockerfile 작성** (multi-stage, scratch/distroless)
5. **Tier 2 · GitHub Actions CI** (`go test ./...`, `vite build`, `tsc --noEmit`)

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

### T2-0. Phase A — Unit Economics Foundation (2026-04-24 구현 완료)
- [x] D. Metrics schema + Repository (`abef790`, `0e83386`, `315e1ec`, `4679a0e`)
- [x] A. AI Cost Optimizer: max_tokens split, prompt cache, stop_reason hook (`2e98956`, `4d3f980`, `f9efbfe`, `cb80c26`, `e8ceda8`, `6aa9ee4`)
- [x] C. Quick Match (축소판): `/api/rooms/quick` + 프론트 버튼 (`c5a3a1d`, `dc9baf8`)
- [x] B. Ad Integration: `/api/metrics/ad` + Redis-backed limiter + 3-surface AdBanner (`4a24106`, `d4b9874`, `1babfd8`, `2ed76c1`)
- [x] Integration wiring: `1896bcb`
- [ ] **로컬 검증 runbook 실행** — 6개 정량 기준 pass 확인
- Next Phases (별도 브레인스토밍):
  - Phase B — 보상형 광고 + 초대 링크 (Phase A cache hit 및 impression 데이터 필요)
  - Phase C — 새 역할 or 방 사이즈 가변 (후 `ai_count` 분포 보고 결정)
  - Phase D — 랭킹 + 시즌 (C 확정 후)

### T2-0c. Data retention 작업의 사전 조건 — cascade 삭제 가이드 (2026-04-25 spec self-review 발견)
- **배경**: FK 금지 정책(ARCHITECTURE §4.14) 으로 `game_results` 가 DELETE 될 때 `game_result_players` 가 자동 cascade 되지 않는다
- [ ] retention 정책 도입 전에 `GameResultRepository` 에 `DeleteOlderThan(t time.Time)` 같은 메서드를 추가하고, 자식(`game_result_players`) 을 **부모보다 먼저** 단일 트랜잭션에서 삭제하도록 명시
- [ ] 동일 패턴이 필요한 향후 테이블 (예: `game_metrics` 가 `game_results` 와 의미적으로 1:1 이지만 hard cascade 불필요) 도 retention 도입 시 점검
- **출처**: `docs/superpowers/specs/2026-04-25-db-schema-policy-design.md` D-1
- **우선순위**: retention 도입 직전. 단독으로는 가치 없음 (현재 DELETE 사용처 0건)

### T2-0b. Phase A follow-up: `game_id` vs `room_id` 분리 (✅ T21 완료 · 2026-04-24 · `503e9ea`)
- [x] `GameMetricsRepository.Create` / `Finalize` 를 게임 생명주기에 훅
- [x] `game_results.id` ↔ `game_metrics.game_id` 를 동일 UUID 로 통일 (GameManager.start 에서 pre-generate)
- [x] `ai.Manager.SpawnAgents` / `AddAgent` 에 gameID 파라미터 추가, T12 `gameID := roomID` 플레이스홀더 제거
- [x] Runbook §5 쿼리 수정 (`ended_at` → `created_at`, `game_id` join)

### T2-5b. 경계면 drift 정리 (2026-04-24 QA 발견 → 당일 해결)
- [x] **D1 (Critical)** `game_over` all_humans_left path — `buildAbortedGameOverPayload()` 헬퍼로 full-shape `{winner: "aborted", round, duration_sec, players: [], reason}` 전송 (TDD RED→GREEN). 프론트 `GameOverResult.winner` 에 `'aborted'` 추가, ResultOverlay 에 "게임 중단" 분기 추가
- [x] **D2** `max_humans` drift — `buildInitialStateRoomPayload()` 헬퍼 추출(dto.RoomResponse 정책 미러). hub.go initial_state 가 이를 재사용하여 max_humans 포함. 프론트 Room 타입에 `max_humans: number` 추가
- [x] **D3** `join_code` 타입 vs omitempty — 프론트 Room `join_code?: string` 로 optional 전환 (공개방 런타임 동작과 정합)
- 근거: `_workspace/qa_report.md`

### T2-6. Fiber 성능 패턴 적용
- [ ] 반복 할당 구조체 `sync.Pool` 도입 (WS 브로드캐스트 payload, DTO 빌더)
- [ ] Context 전파 정리 (`c.Context()` 일관 사용, 핸들러 밖 고루틴 cancel 연결)

### T2-7. 관측성 기본선
- [ ] 구조화 로그(zap) 필드 표준화 (`room_id`, `player_id`, `event`)
- [ ] HTTP 요청 로깅 미들웨어
- [ ] `/healthz`, `/readyz` 엔드포인트

### T2-7b. 분산 안정성: Redis pub/sub 재연결 회복 로직
- 현재 `internal/platform/ws/hub.go` 의 `startSubscriber` 가 채널 close (`!ok` branch) 또는 `ctx.Done()` 시 silent exit. 재연결 루프 없음
- Redis 가 일시 단절(네트워크 blip / Redis 재시작)되면 모든 인스턴스가 cross-instance WS 이벤트 릴레이를 멈춤 → 다른 인스턴스 플레이어들이 게임 이벤트를 못 보게 됨
- **수정**: outer loop 에 reconnect-with-exponential-backoff (최대 ~30s) 추가. 진정한 종료 신호는 `ctx.Done()` 만; 채널 close 는 재시도. `PSubscribe(ctx, "room:*")` 다시 호출
- **4축 영향**: 비용 ―, 수익 ―, 리텐션 ⬆ (멀티 인스턴스 환경 게임 안정성), 인간 밀도 ―
- **현재 우선순위**: 단일 Pod 단계에서는 N/A. 멀티 Pod 이전 시 필수 (ARCHITECTURE §4.3, §4.13 참조)
- **출처**: `backend/TODOS.md` TODO-2 (2026-04-25 정리 시 이전)

### T2-8. 게임 UX: 마피아 합의 실패 시 `night_result` 이벤트
- 현재 `internal/games/mafia/phases.go` 의 `processMafiaKill` 가 마피아 투표 합의 실패 시 silently 종료 → 플레이어가 "왜 아무도 안 죽었지?" 로 혼란. 특히 첫 라운드 신규 플레이어에게 합의 메커닉이 가려짐
- **수정**: `processMafiaKill` 의 fall-through 경로에 `EventNightAction` (또는 `EventKill`) emit 추가, payload `reason: "no_consensus"`. 프론트 `gameStore.ts` 의 `night_action` 케이스에서 시스템 메시지 ("마피아가 합의에 실패하여 밤 사이 아무 일도 일어나지 않았습니다") 표시
- **트레이드오프**: 시민에게 마피아가 분열했다는 약한 정보가 새지만(누가 누구에게 투표했는지는 알 수 없음), UX 명확성 가치가 더 큼
- **4축 영향**: 비용 ―, 수익 ―, 리텐션 ⬆ (UX 명확성), 인간 밀도 ―
- **출처**: `backend/TODOS.md` TODO-1 (2026-04-25 정리 시 이전)

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
