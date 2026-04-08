# CLAUDE.md (Project: ai_side)

## gstack

For all web browsing tasks, use the `/browse` skill from gstack.
Never use `mcp__claude-in-chrome__*` tools directly.

### Available gstack skills

`/office-hours`, `/plan-ceo-review`, `/plan-eng-review`, `/plan-design-review`,
`/design-consultation`, `/design-shotgun`, `/design-html`, `/review`, `/ship`,
`/land-and-deploy`, `/canary`, `/benchmark`, `/browse`, `/connect-chrome`,
`/qa`, `/qa-only`, `/design-review`, `/setup-browser-cookies`, `/setup-deploy`,
`/retro`, `/investigate`, `/document-release`, `/codex`, `/cso`, `/autoplan`,
`/careful`, `/freeze`, `/guard`, `/unfreeze`, `/gstack-upgrade`, `/learn`

If gstack skills aren't working, rebuild the binary and re-register:

```bash
cd .claude/skills/gstack && ./setup
```

## Skill routing

When the user's request matches an available skill, ALWAYS invoke it using the Skill
tool as your FIRST action. Do NOT answer directly, do NOT use other tools first.
The skill has specialized workflows that produce better results than ad-hoc answers.

Key routing rules:
- Product ideas, "is this worth building", brainstorming → invoke office-hours
- Bugs, errors, "why is this broken", 500 errors → invoke investigate
- Ship, deploy, push, create PR → invoke ship
- QA, test the site, find bugs → invoke qa
- Code review, check my diff → invoke review
- Update docs after shipping → invoke document-release
- Weekly retro → invoke retro
- Design system, brand → invoke design-consultation
- Visual audit, design polish → invoke design-review
- Architecture review → invoke plan-eng-review

## Design System
Always read DESIGN.md before making any visual or UI decisions.
All font choices, colors, spacing, and aesthetic direction are defined there.
Do not deviate without explicit user approval.
In QA mode, flag any code that doesn't match DESIGN.md.

---

## 하네스: AI 마피아 게임 플랫폼

**목표:** Backend(Go) · Frontend(React) · QA 에이전트가 협업하여 기능 구현, 버그 수정, 테스트를 수행한다.

**에이전트 팀:**

| 에이전트 | 역할 |
|---------|------|
| `backend` | Go + Fiber 백엔드 구현·수정·테스트 |
| `frontend` | React + TypeScript 프론트엔드 구현·수정 |
| `qa` | 경계면 검증, 테스트 작성·실행, 버그 탐지 |

**스킬:**

| 스킬 | 용도 | 사용 에이전트 |
|------|------|-------------|
| `mafia-orchestrator` | 팀 전체 조율 (기본 진입점) | 리더 |
| `backend-dev` | Go 백엔드 작업 가이드 | backend |
| `frontend-dev` | React 프론트엔드 작업 가이드 | frontend |
| `qa-test` | QA 검증 및 테스트 작성 가이드 | qa |

**실행 규칙:**
- 기능 구현, 버그 수정, 테스트 작성 등 개발 관련 작업 → `mafia-orchestrator` 스킬로 에이전트 팀 처리
- QA/테스트만 필요한 경우 → `qa-test` 스킬 직접 사용 가능
- 백엔드만, 프론트만 필요한 경우 → `backend-dev` / `frontend-dev` 스킬 직접 사용 가능
- 모든 에이전트는 `model: "opus"` 사용
- 중간 산출물: `_workspace/` 디렉토리

**디렉토리 구조:**
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

**변경 이력:**

| 날짜 | 변경 내용 | 대상 | 사유 |
|------|----------|------|------|
| 2026-04-07 | 초기 구성 | 전체 | AI 마피아 게임 플랫폼 하네스 신규 구축 |
