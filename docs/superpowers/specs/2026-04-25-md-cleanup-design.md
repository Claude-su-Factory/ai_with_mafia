# 불필요한 .md 파일 정리

**Status:** Approved 2026-04-25 (brainstorming 사이클 정상 통과)

**Related:**
- 사용자 안건: "불필요한 .md 파일 좀 삭제해줘"
- ROADMAP.md (TODO 이전 대상)
- CLAUDE.md "스펙 작성 규칙 (MANDATORY)"

---

## 1. Goal & Scope

프로젝트에 산재한 옛 워크플로우 도구 산출물(`openspec/`), 사용처 없는 디자인 레퍼런스(`ui/`), ROADMAP 과 중복되는 메모(`backend/TODOS.md`) 를 제거하여 repo 가시성을 높인다.

**삭제 대상 (총 104 tracked 파일):**

| 경로 | 파일 수 | 정당성 |
|------|---------|--------|
| `openspec/` 디렉토리 전체 | 99 | 옛 워크플로우 도구의 산출물. `.claude/settings.local.json` 의 bash 권한 외 코드 참조 0건. `archive/` 서브디렉토리가 자체로 outdated 명시 |
| `ui/` 디렉토리 전체 (DESIGN.md + stitch/) | 4 | `openspec/changes/glacier-ui-redesign` 만 참조 (둘 다 삭제 대상). frontend 코드 import 0건. 옛 디자인 레퍼런스 |
| `backend/TODOS.md` | 1 | ROADMAP.md 와 중복 의도. 단, 미해결 TODO 2건을 ROADMAP 으로 이전 후 삭제 |

**보존 (절대 삭제 X):**
- 루트 `README.md`, `CLAUDE.md`, `DESIGN.md`
- `docs/{ARCHITECTURE,STATUS,ROADMAP}.md`
- `backend/README.md`, `frontend/README.md`
- `docs/superpowers/{specs,plans}/*` 전체 (Phase A 직전 시기 포함)

### Out of scope

- `_workspace/*.md` (gitignored, 영향 없음)
- `docs/superpowers/specs/2026-04-08~04-13` 옛 specs/plans (미래 참조 가치 있음, 정리는 별도 결정)
- `.claude/settings.local.json` 의 `openspec` bash 권한 라인 — 권한이라 삭제해도 무해하지만 단일 책임 원칙상 별도 작업

---

## 2. TODOS.md 의 2개 TODO ROADMAP 이전

단순 삭제하면 정보 손실. ROADMAP 에 명시 등록한 뒤 삭제.

### TODO-1 → ROADMAP T2-8 (Tier 2 — 게임 UX 보강)

**원문 요약:**
- `processMafiaKill` 가 합의 실패 시 silently 종료 → 플레이어가 "왜 아무도 안 죽었지?" 로 혼란
- `internal/games/mafia/phases.go` 의 fall-through 경로에 `EventNightAction` 또는 `EventKill` reason `"no_consensus"` 추가 필요

**ROADMAP 등록 형태:**
```markdown
### T2-8. 게임 UX: 마피아 합의 실패 시 `night_result` 이벤트
- 현재 `processMafiaKill` 가 합의 실패 시 조용히 종료 → 플레이어 혼란
- `phases.go` 의 fall-through 경로에 `EventNightAction` (reason: "no_consensus") emit 추가
- 프론트 `gameStore.ts` 의 `night_action` 케이스에서 시스템 메시지로 표시
- 4축 영향: 리텐션 ⬆ (UX 명확성), 비용/수익/밀도 ―
- 출처: `backend/TODOS.md` TODO-1 (2026-04-25 정리 시 이전)
```

### TODO-2 → ROADMAP T2-7b (Tier 2 — 분산 안정성)

**원문 요약:**
- `internal/platform/ws/hub.go.startSubscriber` 가 Redis pub/sub 채널 close 시 silent exit
- Redis 재시작/네트워크 blip 후 cross-instance WS 이벤트 중단 → 멀티 Pod 환경 game 깨짐

**ROADMAP 등록 형태:**
```markdown
### T2-7b. 분산 안정성: Redis pub/sub 재연결 회복 로직
- `startSubscriber` 의 outer loop 에 reconnect-with-backoff (max ~30s) 추가
- `ctx.Done()` 만 진정한 종료 신호. 채널 close 는 재시도
- 4축 영향: 멀티 Pod 환경 안정성 ⬆ (분산·동시성 렌즈 §4.13 참조). 단일 Pod 단계에서는 N/A
- 출처: `backend/TODOS.md` TODO-2 (2026-04-25 정리 시 이전)
```

---

## 3. Implementation Order

1. **ROADMAP.md 갱신** — T2-7b, T2-8 두 항목 추가 (정보 보존이 우선)
2. **삭제 실행** — `git rm -r ui openspec` + `git rm backend/TODOS.md`
3. **STATUS.md 변경 이력** — 한 줄 추가
4. **빌드/테스트 회귀 검증** — `go build ./... && go test ./...` 영향 없을 것
5. **단일 commit** — `chore: remove unused openspec/, ui/, backend/TODOS.md (TODOs migrated to ROADMAP)`
6. **push** — origin master

---

## 4. Risk & Failure Modes

### R1. 다른 docs 가 삭제 대상 경로를 링크

- **검증:** `grep -rn "openspec\|ui/DESIGN\|backend/TODOS" docs/ README.md CLAUDE.md` 로 cross-link 확인
- 발견 시: 링크 제거 또는 인라인 대체

### R2. `.claude/settings.local.json` 의 openspec 권한 라인

- 권한 정의는 무해. 명령어 자체가 사라진 것뿐
- 본 spec 범위 외. 사용자가 `.claude/settings.local.json` 정리 원하면 별도 작업

### R3. archive 가치 손실

- git history 에 남으므로 `git show <sha>:openspec/...` 으로 회수 가능
- 추가 안전망: 본 commit 직전 state 의 SHA 를 ROADMAP/STATUS 에 기록해서 회수 진입점 제공

### R4. ROADMAP 이전 누락

- TODOS.md 안에 다른 잠재 TODO 가 있는지 self-review 에서 재확인 (현 시점: 2개만 식별)

---

## 5. Concurrency & Distribution Analysis

CLAUDE.md "동시성·분산 안전성" 렌즈:

| 질문 | 답 |
|------|-----|
| 상태 위치 | 파일 시스템 (git tracked). 분산 무관 |
| Cross-Pod 일관성 | 모든 Pod 가 동일 git revision 에서 빌드 → 자동 일관 |
| Eventual consistency 경계 | 없음 |
| 멀티 Pod 이관 시 영향 | 없음 (런타임 로직과 무관한 docs/메타파일 정리) |

**runtime 영향 0건.** 빌드 산출물·바이너리 변화 없음.

---

## 6. Testing Strategy

자동 검증:
- `go build ./...` exit 0
- `go test ./...` 전체 green (cached 포함)
- `cd frontend && npx tsc --noEmit` 0 errors

수동 검증:
- 삭제 후 `find . -name "*.md" -not -path "*/.git/*" -not -path "*/node_modules/*" -not -path "*/.claude/*" -not -path "*/.superpowers/*" -not -path "*/_workspace/*" | wc -l` 결과 ~30개 수준 (이전 ~90개에서 감소)
- ROADMAP.md 에 T2-7b, T2-8 항목 visible

---

## 7. Success Criteria

- [ ] 104 tracked 파일이 git 에서 제거됨
- [ ] ROADMAP.md 에 T2-7b, T2-8 두 항목 추가
- [ ] STATUS.md "최근 변경 이력" 에 한 줄 항목
- [ ] go build/test/tsc 모두 green
- [ ] GitHub push 성공
- [ ] commit 메시지에 회수 진입점 SHA 기록

---

## 8. 자체 검토 사이클 (Spec Self-Review)

### 8.1 Placeholder scan

- TBD/TODO: 0건
- §2 의 ROADMAP 등록 텍스트는 literal — placeholder 아님

### 8.2 Internal consistency

- §1 (삭제 대상) ↔ §3 (실행 순서) ↔ §6 (검증) 정합 ✅
- §2 의 TODO 이전 형식 ↔ ROADMAP.md 기존 항목 형식 비교: T2-0~T2-7 의 표기와 일관 (`### T2-X. 제목`)

### 8.3 Scope check

- 단일 spec, 단일 plan 으로 처리 가능. decomposition 불필요
- Out of scope 명시됨 — `_workspace/*.md`, 옛 superpowers 문서, `.claude/settings.local.json`

### 8.4 Ambiguity

| 잠재 ambiguity | 해소 |
|---|------|
| "ROADMAP 이전" 시 우선순위가 불분명 | T2-7b, T2-8 둘 다 Tier 2 (현재 추천 다음 작업 아님). 명시됨 |
| `ui/stitch/screen.png` 의 비-md 파일도 삭제? | YES. `git rm -r ui/` 가 디렉토리 전체 (HTML/PNG 포함) 처리 |
| 삭제 후 archived 형태로 어딘가 보관? | NO. git history 만으로 충분 (디자인 §4 R3 참조) |

### 8.5 발견된 결함 (이 검토에서)

**없음.** 작업 scope 가 명확하고 destructive 영역이 잘 격리됨.

---

## 9. 검토 이력

| 날짜 | 이벤트 | 결과 |
|-----|-------|------|
| 2026-04-25 | brainstorming → design 합의 → spec 작성 + self-review 1회 | 결함 0건. 진행 가능 |
