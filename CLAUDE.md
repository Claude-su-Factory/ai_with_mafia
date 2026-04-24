# CLAUDE.md — Project: ai_side (AI 마피아 게임 플랫폼)

이 문서는 모든 세션(리더, backend / frontend / qa 서브에이전트 포함)의 **단일 진입점**이다.
규칙은 파일에 박아 둔다. 메모리·대화 맥락에만 남기는 것은 허용되지 않는다.
새 세션이 이 파일 + `docs/` 3종(STATUS · ROADMAP · ARCHITECTURE) 만 읽어도 프로젝트 이해도를 확보해야 한다.

---

## 빠른 네비게이션

| 목적 | 파일 |
|------|------|
| 현재 구현 상태 · Phase 체크리스트 · 알려진 결함 | [`docs/STATUS.md`](docs/STATUS.md) |
| 다음 작업 · Tier 1/2/3 우선순위 · JD 매핑 | [`docs/ROADMAP.md`](docs/ROADMAP.md) |
| 시스템 구성 · 설계 결정(Why 포함) · 결정 로그 | [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) |
| 기능별 상세 설계 | [`docs/superpowers/specs/`](docs/superpowers/specs/) |
| 기능별 구현 계획 | [`docs/superpowers/plans/`](docs/superpowers/plans/) |
| 디자인 시스템 | [`DESIGN.md`](DESIGN.md) |

---

## 개발 워크플로우 (MANDATORY)

모든 세션은 아래 순서를 **예외 없이** 따른다.

1. **계획 작성** — 코드를 한 줄도 쓰기 전에 `superpowers:writing-plans` 스킬 호출. 저장 위치: `docs/superpowers/plans/YYYY-MM-DD-<feature>.md`
2. **구현** — 계획이 준비되면 `superpowers:subagent-driven-development` 스킬로 실행
3. **코드 리뷰** — 구현 완료 후 `superpowers:requesting-code-review` 스킬을 **반드시** 호출 (사용자가 명시적 생략 요청하지 않는 한 예외 없음). 리뷰 이슈는 수정 후 재검토

---

## Skill routing (MANDATORY)

사용자 요청이 스킬과 매칭되면 **다른 도구보다 먼저** `Skill` 도구로 해당 스킬을 호출한다. 직접 답하거나 다른 도구를 먼저 쓰지 않는다.

### 상황별 필수 스킬

| 상황 | 사용 스킬 |
|------|----------|
| 새 아이디어·안건 제시, "어떻게 하면 좋을까", 접근법이 열린 질문 | `superpowers:brainstorming` |
| 새 기능·버그 수정·리팩터링 등 모든 구현 작업 | `superpowers:writing-plans` → `superpowers:subagent-driven-development` → `superpowers:requesting-code-review` |
| 버그·에러·"왜 안 되지"·500 에러·예기치 않은 동작 | `superpowers:investigate` |
| 코드 작성 (기능 구현·버그 수정) | `superpowers:test-driven-development` |
| 완료 직전 리뷰 요청 | `superpowers:requesting-code-review` |
| 디자인 플랜 리뷰 · UI/UX 포함 계획 | `plan-design-review` |
| 기능·버그·QA·테스트 (backend+frontend+qa 협업 필요) | `mafia-orchestrator` |
| 백엔드 전용 작업 | `backend-dev` |
| 프론트엔드 전용 작업 | `frontend-dev` |
| QA·테스트 작성만 | `qa-test` |

### gstack 계열 보조 스킬

| 상황 | 스킬 |
|------|------|
| 웹 브라우징·스크린샷·사이트 QA | `/browse` (절대 `mcp__claude-in-chrome__*` 직접 호출 금지) |
| 배포·PR·푸시 | `/ship`, `/land-and-deploy` |
| PR 전 diff 리뷰 | `/review` |
| 디자인 시스템 구축 | `/design-consultation` |
| 라이브 사이트 디자인 감사 | `/design-review` |
| 아키텍처 리뷰 | `/plan-eng-review` |

---

## 문서 관리 (MANDATORY)

### 스펙 작성 규칙

스펙(`docs/superpowers/specs/*.md`)을 작성하면 사용자에게 보여주기 전에 **반드시 자체 검토** 사이클을 거친다.

1. 작성 직후 스스로 검토하여 이슈 식별
   - Critical: 명세 그대로 구현 시 동작 안 함 (race, chunk 경계, 잘못된 API)
   - Important: 리소스 누수, 비효율 패턴, 에러 핸들링 누락
   - Minor: 명확성, 비범위 명시, 예시 코드 helper 누락
2. 우선순위별로 사용자에게 보고
3. 스펙 파일을 직접 패치
4. 스펙 하단에 "검토 이력" 섹션 추가/업데이트

별도 요청이 없어도 **작성 → 검토 → 패치 → 보고**가 한 사이클이다.

### 기능 구현 완료 시 문서 업데이트

다음 파일은 기능 구현 완료 시 반드시 갱신한다. 업데이트 없이는 완료로 간주하지 않는다.

1. **`docs/STATUS.md`** — 해당 항목 ✅로 이동 · "최근 변경 이력" 맨 위에 한 줄 추가 · "마지막 업데이트" 날짜 갱신 · 해결된 "알려진 결함" 제거
2. **`docs/ROADMAP.md`** — 완료된 항목 제거 · 필요 시 "현재 추천 다음 작업" 재설정
3. **`docs/ARCHITECTURE.md`** — 아키텍처에 영향을 준 변경에만 반영 (새 컴포넌트, 설계 결정 번복 등). 설계 결정을 바꿀 때는 기존 항목 삭제 금지, `변경됨(YYYY-MM-DD)` 라벨로 이력 보존

### 하네스 엔지니어링 규칙

작업 중 발견한 **규칙·판단 기준·프로젝트 결정**은 반드시 프로젝트 문서에 기록한다. 메모리·대화 맥락에만 남기는 것은 허용되지 않는다.

| 성격 | 기록 위치 |
|------|----------|
| 프로젝트 전반 작업 규칙 | 이 `CLAUDE.md` |
| 아키텍처 설계 결정 (Why / How) | `docs/ARCHITECTURE.md` "핵심 설계 결정" |
| 작업 흐름 / 문서 관리 규칙 | 해당 문서의 "업데이트 규칙" 섹션 |
| 기능별 상세 규칙·트레이드오프 | `docs/superpowers/specs/<feature>-design.md` |
| 알려진 결함·미구현 이슈 | `docs/STATUS.md` "알려진 결함" + `docs/ROADMAP.md` |

기록 흐름:
1. "이 결정/규칙은 다음 세션에도 유효하다"고 판단되면 위 표에서 위치 선택
2. 임시 메모가 아니라 **명시적 섹션**으로 추가
3. 같은 커밋에 포함 (문서 변경과 코드 변경을 묶는다)
4. 필요하면 이 CLAUDE.md 의 "빠른 네비게이션"에 새 경로도 추가

---

## 하네스: AI 마피아 게임 플랫폼

**목표:** Backend(Go) · Frontend(React) · QA 에이전트가 협업하여 기능 구현, 버그 수정, 테스트를 수행한다.

### 에이전트 팀

| 에이전트 | 역할 |
|---------|------|
| `backend` | Go + Fiber 백엔드 구현·수정·테스트 |
| `frontend` | React + TypeScript 프론트엔드 구현·수정 |
| `qa` | 경계면 검증, 테스트 작성·실행, 버그 탐지 |

### 스킬

| 스킬 | 용도 | 사용 에이전트 |
|------|------|-------------|
| `mafia-orchestrator` | 팀 전체 조율 (기본 진입점) | 리더 |
| `backend-dev` | Go 백엔드 작업 가이드 | backend |
| `frontend-dev` | React 프론트엔드 작업 가이드 | frontend |
| `qa-test` | QA 검증 및 테스트 작성 가이드 | qa |

### 실행 규칙

- 기능·버그·QA 전반 → `mafia-orchestrator` 스킬로 에이전트 팀 처리
- QA/테스트만 → `qa-test` 스킬 직접 사용 가능
- 백엔드만 / 프론트만 → `backend-dev` / `frontend-dev` 스킬 직접 사용 가능
- 모든 에이전트는 `model: "opus"` 사용
- 중간 산출물: `_workspace/` 디렉토리 (gitignored)

### 디렉토리 구조

```
.claude/
├── agents/
│   ├── backend.md
│   ├── frontend.md
│   └── qa.md
└── skills/
    ├── mafia-orchestrator/SKILL.md
    ├── backend-dev/SKILL.md
    ├── frontend-dev/SKILL.md
    └── qa-test/SKILL.md
```

---

## 작업 참고사항

### 언어 · 프레임워크

- **Go (Backend)** — Fiber v2 API. `*fiber.Ctx` 로 I/O, Context 전파로 자원 수명 관리. 반복 할당 구조체는 `sync.Pool` 로 재사용하여 GC 부담 감소
- **TypeScript (API 서버가 필요한 경우)** — Express 사용. 유지보수 친화적 디자인 패턴(계층 분리, DI) 적용
- **React + TypeScript (Frontend)** — Zustand 로 상태 관리. 컴포넌트는 얇게, 로직은 훅·스토어로

### 인증·인가

- **Supabase** 기반. Google OAuth + ES256 JWT (JWK 공개키). HS256 방식은 쓰지 않는다. ARCHITECTURE 4.2 참조

### Docker 빌드

- 항상 **multi-stage + 최소 베이스(scratch / distroless)**
- 불필요한 빌드 단계를 넣지 않아 빌드 시간·이미지 사이즈를 최적화
- `.dockerignore` 로 `node_modules`, `.git`, `config.toml`, `_workspace*/`, `dist/` 제외
- 이미지 사이즈 목표: backend < 25MB, frontend < 30MB

### 검색(SEO)

- 공개 페이지는 메타 태그(`og:*`, `twitter:*`), 구조화 데이터(JSON-LD), `sitemap.xml`, `robots.txt` 를 갖춘다
- 유입 최적화를 매 작업에서 고려 (랜딩 copy, URL 설계, 초기 렌더 속도)

---

## 운영 배포 (MANDATORY 규칙)

### 스택

- **Railway** — 운영 호스팅
- **Doppler** — secret 관리·주입
- **Kubernetes** — 로컬·운영 공용 매니페스트
- **GitHub Actions** — `main` 브랜치 push 시 **자동 운영 배포**

### 규칙

1. **secret 은 코드·yaml·파이프라인에 하드코딩 금지.** 키 이름을 yaml 에 박지 않는다. 환경변수가 추가될 때마다 yaml / 파이프라인을 수정하고 싶지 않기 때문
2. **secret 주입은 파일 읽기 방식.** Doppler 가 secret file 을 mount → 앱이 파일을 읽어 config 구성. `backend/config.toml` 같은 평문 키 파일은 **절대 커밋 금지**
3. **로컬에서도 자동화된 배포 흐름을 쓴다.** 수동 kubectl 커맨드 나열이 아니라 스크립트 한 줄(`make deploy-local` 등)로 재현
4. **`.gitignore` / `.dockerignore` 를 엄격히 작성.** GitHub 에 올라가는 소스코드에 키·토큰·세션 파일이 섞이면 안 된다
5. 운영 배포는 `main` 머지 → CI 테스트 통과 → Railway 배포 파이프라인으로 자동화

### 현재 상태

- ✅ 로컬 `docker-compose.yml`(postgres + redis)
- ✅ `.gitignore` · `.dockerignore` · `backend/config.example.toml` · `frontend/.env.example` 정비 (`abc86c2`)
- ✅ `backend/config.toml` · `frontend/.env.development` · `.env.production` git 트래킹 해제 (`abc86c2`)
- ❌ `Dockerfile` (backend·frontend), k8s 매니페스트, CI/CD 파이프라인 미구현 (ROADMAP T2-1~T2-3)
- 🟠 과거 커밋에는 secret이 잔존 — **remote push 전에 `git filter-repo` 또는 새 `git init` 로 정리 필요** (현재 원격 없으므로 외부 노출 0)

---

## gstack

웹 브라우징은 반드시 `/browse` 스킬을 사용한다. `mcp__claude-in-chrome__*` 도구는 절대 사용하지 않는다.

사용 가능한 gstack 스킬 일람:
`/office-hours`, `/plan-ceo-review`, `/plan-eng-review`, `/plan-design-review`,
`/design-consultation`, `/design-shotgun`, `/design-html`, `/review`, `/ship`,
`/land-and-deploy`, `/canary`, `/benchmark`, `/browse`, `/connect-chrome`,
`/qa`, `/qa-only`, `/design-review`, `/setup-browser-cookies`, `/setup-deploy`,
`/retro`, `/investigate`, `/document-release`, `/codex`, `/cso`, `/autoplan`,
`/careful`, `/freeze`, `/guard`, `/unfreeze`, `/gstack-upgrade`, `/learn`

gstack 스킬이 동작하지 않으면 재빌드:
```bash
cd .claude/skills/gstack && ./setup
```

---

## Design System

UI 작업 전 항상 `DESIGN.md` 를 먼저 읽는다. 폰트·색·간격·미학 방향은 거기에 정의되어 있다. 명시적 승인 없이 벗어나지 않는다. QA 모드에서는 `DESIGN.md` 와 일치하지 않는 코드를 플래그로 표시한다.

---

## 변경 이력

| 날짜 | 변경 내용 | 대상 | 사유 |
|------|----------|------|------|
| 2026-04-07 | 초기 하네스 구성 | 전체 | AI 마피아 게임 플랫폼 하네스 신규 구축 |
| 2026-04-24 | 문서 3종(STATUS/ROADMAP/ARCHITECTURE) 신설 · 스킬 라우팅·문서·배포·작업 참고 규칙 CLAUDE.md에 고정 · frontend.md 포트 drift 수정 | 전체 | 다른 세션에서도 프로젝트 규칙을 즉시 파악하도록 단일 진입점 정립 |
