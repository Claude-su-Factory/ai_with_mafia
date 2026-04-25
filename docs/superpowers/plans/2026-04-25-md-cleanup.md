# Unused .md Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove 104 stale tracked `.md` files (`openspec/`, `ui/`, `backend/TODOS.md`) after migrating two pending TODOs into ROADMAP, in a single atomic commit.

**Architecture:** ROADMAP/STATUS updates land FIRST so information is preserved before any deletion. All file removals + doc updates go into one commit so the repo never observes a half-cleaned intermediate state. Build/test sanity check confirms no runtime impact (none expected — all deleted files are docs/meta).

**Tech Stack:** git, markdown.

**Spec:** `docs/superpowers/specs/2026-04-25-md-cleanup-design.md`

---

## File Structure

### Modified
| Path | Change |
|------|--------|
| `docs/ROADMAP.md` | Add T2-7b (Redis pub/sub reconnect) + T2-8 (mafia no-consensus event) sections, sourced from `backend/TODOS.md` |
| `docs/STATUS.md` | Append change-log line referencing this cleanup |

### Deleted
| Path | Reason |
|------|--------|
| `openspec/` (entire directory, 99 files) | Old workflow tool artifacts; superpowers replaced it |
| `ui/` (entire directory, 4 files: DESIGN.md, stitch/{DESIGN.md, code.html, screen.png}) | Only referenced by deleted openspec change; no code imports |
| `backend/TODOS.md` | Two TODOs migrated to ROADMAP; rest of file is a duplicate of ROADMAP intent |

### Unchanged
- `README.md`, `CLAUDE.md`, `DESIGN.md`
- `docs/ARCHITECTURE.md`
- `backend/README.md`, `frontend/README.md`
- `docs/superpowers/{specs,plans}/**`
- All source code

---

## Task 1: Migrate TODO-2 (Redis pub/sub reconnect) into ROADMAP T2-7b

**Files:**
- Modify: `docs/ROADMAP.md`

- [ ] **Step 1: Locate the right insertion point**

Run:
```bash
grep -n "^### T2-7\b\|^### T2-8\b\|^### T3" /Users/yuhojin/Desktop/ai_side/docs/ROADMAP.md
```

Expected: lines showing the existing `### T2-7. 관측성 기본선` (or similar) section, and the boundary to Tier 3. Insert T2-7b immediately after T2-7.

- [ ] **Step 2: Insert T2-7b section**

In `docs/ROADMAP.md`, immediately after the closing line of `### T2-7. 관측성 기본선` and before the next `### T2-X` or `### T3` header, add this block (literal text, do not paraphrase):

```markdown
### T2-7b. 분산 안정성: Redis pub/sub 재연결 회복 로직
- 현재 `internal/platform/ws/hub.go` 의 `startSubscriber` 가 채널 close (`!ok` branch) 또는 `ctx.Done()` 시 silent exit. 재연결 루프 없음
- Redis 가 일시 단절(네트워크 blip / Redis 재시작)되면 모든 인스턴스가 cross-instance WS 이벤트 릴레이를 멈춤 → 다른 인스턴스 플레이어들이 게임 이벤트를 못 보게 됨
- **수정**: outer loop 에 reconnect-with-exponential-backoff (최대 ~30s) 추가. 진정한 종료 신호는 `ctx.Done()` 만; 채널 close 는 재시도. `PSubscribe(ctx, "room:*")` 다시 호출
- **4축 영향**: 비용 ―, 수익 ―, 리텐션 ⬆ (멀티 인스턴스 환경 게임 안정성), 인간 밀도 ―
- **현재 우선순위**: 단일 Pod 단계에서는 N/A. 멀티 Pod 이전 시 필수 (ARCHITECTURE §4.3, §4.13 참조)
- **출처**: `backend/TODOS.md` TODO-2 (2026-04-25 정리 시 이전)
```

- [ ] **Step 3: Verify insertion**

Run:
```bash
grep -A 1 "^### T2-7b\b" /Users/yuhojin/Desktop/ai_side/docs/ROADMAP.md | head -3
```

Expected: header line + first body line visible, no duplicate `### T2-7b` elsewhere.

- [ ] **Step 4: Don't commit yet** — Task 4 will batch-commit ROADMAP + STATUS + deletions together.

---

## Task 2: Migrate TODO-1 (mafia no-consensus event) into ROADMAP T2-8

**Files:**
- Modify: `docs/ROADMAP.md`

- [ ] **Step 1: Insert T2-8 section**

Immediately after the T2-7b block (just inserted) and before the `### T3-1` header, add (literal):

```markdown
### T2-8. 게임 UX: 마피아 합의 실패 시 `night_result` 이벤트
- 현재 `internal/games/mafia/phases.go` 의 `processMafiaKill` 가 마피아 투표 합의 실패 시 silently 종료 → 플레이어가 "왜 아무도 안 죽었지?" 로 혼란. 특히 첫 라운드 신규 플레이어에게 합의 메커닉이 가려짐
- **수정**: `processMafiaKill` 의 fall-through 경로에 `EventNightAction` (또는 `EventKill`) emit 추가, payload `reason: "no_consensus"`. 프론트 `gameStore.ts` 의 `night_action` 케이스에서 시스템 메시지 ("마피아가 합의에 실패하여 밤 사이 아무 일도 일어나지 않았습니다") 표시
- **트레이드오프**: 시민에게 마피아가 분열했다는 약한 정보가 새지만(누가 누구에게 투표했는지는 알 수 없음), UX 명확성 가치가 더 큼
- **4축 영향**: 비용 ―, 수익 ―, 리텐션 ⬆ (UX 명확성), 인간 밀도 ―
- **출처**: `backend/TODOS.md` TODO-1 (2026-04-25 정리 시 이전)
```

- [ ] **Step 2: Verify**

Run:
```bash
grep -A 1 "^### T2-8\b" /Users/yuhojin/Desktop/ai_side/docs/ROADMAP.md | head -3
```

Expected: header + first body line, single occurrence.

- [ ] **Step 3: No commit yet.**

---

## Task 3: Append STATUS.md change-log entry

**Files:**
- Modify: `docs/STATUS.md`

- [ ] **Step 1: Locate the change-log section**

Run:
```bash
grep -n "## 최근 변경 이력" /Users/yuhojin/Desktop/ai_side/docs/STATUS.md
```

Expected: a single `## 최근 변경 이력 (최신순)` heading. The change-log entries follow it, newest at the top.

- [ ] **Step 2: Prepend new entry**

Immediately after the `## 최근 변경 이력 (최신순)` heading line and the blank line below it, insert this single bullet at the top of the existing list:

```markdown
- **2026-04-25** · 불필요한 `.md` 정리: `openspec/` (99 파일, 옛 워크플로우 도구 산출물), `ui/` (4 파일, frontend 와 별도인 옛 디자인 레퍼런스), `backend/TODOS.md` 삭제. TODOS.md 의 미해결 항목 2건은 ROADMAP T2-7b (Redis pub/sub 재연결) / T2-8 (마피아 합의 실패 UX) 으로 이전. 코드 영향 0건
```

- [ ] **Step 3: Verify exactly one new bullet**

Run:
```bash
grep -c "^- \*\*2026-04-25\*\* · 불필요한" /Users/yuhojin/Desktop/ai_side/docs/STATUS.md
```

Expected: `1`.

- [ ] **Step 4: No commit yet.**

---

## Task 4: Cross-link audit before deletion

**Files:**
- Read-only audit. No changes.

- [ ] **Step 1: Verify deletion targets are not linked from preserved docs**

Run from repo root:
```bash
cd /Users/yuhojin/Desktop/ai_side
echo "=== docs/ + root .md cross-references to soon-to-be-deleted paths ==="
grep -rn "openspec/\|ui/DESIGN\|ui/stitch\|backend/TODOS" \
  README.md CLAUDE.md DESIGN.md \
  docs/ARCHITECTURE.md docs/STATUS.md docs/ROADMAP.md \
  docs/superpowers/specs/ docs/superpowers/plans/ \
  backend/README.md frontend/README.md \
  2>&1 | grep -v "^docs/superpowers/specs/2026-04-25-md-cleanup\|^docs/superpowers/plans/2026-04-25-md-cleanup"
```

Expected: empty output (the spec + plan we just wrote are excluded from the grep so they don't false-positive).

- [ ] **Step 2: If any matches appear**

If matches are found in preserved docs, **STOP** and resolve before deletion. Likely action: replace the link with a brief inline summary or remove the dangling reference. Then re-run Step 1.

- [ ] **Step 3: No commit yet.**

---

## Task 5: Delete the three targets

**Files:**
- Delete: `openspec/` (recursively)
- Delete: `ui/` (recursively)
- Delete: `backend/TODOS.md`

- [ ] **Step 1: Capture pre-delete SHA for recovery reference**

Run:
```bash
PRE_DELETE_SHA=$(git -C /Users/yuhojin/Desktop/ai_side rev-parse HEAD)
echo "Pre-delete HEAD: $PRE_DELETE_SHA"
```

Record the SHA — it will appear in the final commit message so future readers can recover deleted content via `git show $SHA:openspec/...`.

- [ ] **Step 2: Execute the deletions**

```bash
cd /Users/yuhojin/Desktop/ai_side
git rm -r openspec
git rm -r ui
git rm backend/TODOS.md
```

Expected: each command prints `rm '...'` lines for every removed file. No errors.

- [ ] **Step 3: Verify staged state**

```bash
git -C /Users/yuhojin/Desktop/ai_side status --short | head -10
git -C /Users/yuhojin/Desktop/ai_side status --short | grep -c "^D " | awk '{print $0" files staged for deletion"}'
```

Expected: ≥104 lines starting with `D ` (99 openspec + 4 ui + 1 TODOS = 104). Exact count may vary by 1-2 if openspec had files we didn't enumerate; ≥104 is the floor.

- [ ] **Step 4: No commit yet — Task 6 batches everything.**

---

## Task 6: Build & test sanity check

**Files:**
- Read-only validation.

- [ ] **Step 1: Backend build**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && /Users/yuhojin/.gvm/gos/go1.23/bin/go build ./...
echo "build exit: $?"
```

Expected: `exit: 0` (no output before `exit:`).

- [ ] **Step 2: Backend tests**

```bash
cd /Users/yuhojin/Desktop/ai_side/backend && /Users/yuhojin/.gvm/gos/go1.23/bin/go test ./...
```

Expected: all packages `ok` or `[no test files]`. No `FAIL` line.

- [ ] **Step 3: Frontend type check**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npx tsc --noEmit
echo "tsc exit: $?"
```

Expected: `exit: 0`, no error lines.

- [ ] **Step 4: If any of Steps 1-3 fail**

STOP. The spec assumed runtime impact is 0; a failure here invalidates that assumption. Investigate which deleted file was actually load-bearing, restore it via `git restore --staged --worktree <path>`, and update the spec/plan before proceeding.

- [ ] **Step 5: No commit yet — Task 7 commits.**

---

## Task 7: Single atomic commit + push

**Files:**
- Already staged: ROADMAP, STATUS, deletions (104 files)

- [ ] **Step 1: Stage doc edits**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add docs/ROADMAP.md docs/STATUS.md
git status --short | head -10
```

Expected: ROADMAP.md and STATUS.md show `M`; the 104 deletions show `D`.

- [ ] **Step 2: Atomic commit**

Use the captured pre-delete SHA from Task 5 Step 1. If lost, recompute: `PRE_DELETE_SHA=$(git log --oneline | head -1 | awk '{print $1}')` (this is the current HEAD, which is the pre-delete state because nothing has been committed yet).

```bash
cd /Users/yuhojin/Desktop/ai_side
PRE_DELETE_SHA=$(git rev-parse HEAD)
git commit -m "$(cat <<EOF
chore: remove unused openspec/, ui/, backend/TODOS.md

Outdated workflow-tool artifacts and a duplicate TODO list. The two
unresolved TODOs in backend/TODOS.md are migrated to ROADMAP so no
information is lost:

- TODO-1 (mafia no-consensus UX feedback) -> ROADMAP T2-8
- TODO-2 (Redis pub/sub reconnect resilience) -> ROADMAP T2-7b

Pre-delete HEAD for recovery: ${PRE_DELETE_SHA}
Use 'git show ${PRE_DELETE_SHA}:openspec/...' to retrieve any specific
file from before the deletion.

Spec: docs/superpowers/specs/2026-04-25-md-cleanup-design.md
Plan: docs/superpowers/plans/2026-04-25-md-cleanup.md

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

Expected: commit summary line shows ~106 files changed (104 deletions + 2 modifications).

- [ ] **Step 3: Verify commit**

```bash
git -C /Users/yuhojin/Desktop/ai_side log -1 --stat | tail -10
```

Expected: stat showing many `D` lines + 2 `M` lines for the doc edits.

- [ ] **Step 4: Push to GitHub**

```bash
git -C /Users/yuhojin/Desktop/ai_side push origin master
```

Expected: `<old-sha>..<new-sha>  master -> master`. No errors.

- [ ] **Step 5: Final verification**

```bash
echo "=== .md count after cleanup ==="
find /Users/yuhojin/Desktop/ai_side -name "*.md" \
  -not -path "*/.git/*" \
  -not -path "*/node_modules/*" \
  -not -path "*/.claude/*" \
  -not -path "*/.superpowers/*" \
  -not -path "*/_workspace/*" \
  | wc -l

echo "=== ROADMAP has T2-7b + T2-8 ==="
grep -c "^### T2-7b\|^### T2-8" /Users/yuhojin/Desktop/ai_side/docs/ROADMAP.md
```

Expected:
- `.md` count drops from ~90 to ~30
- ROADMAP grep returns `2`

---

## Plan Self-Review

**Spec coverage:**
- §1 Scope (104 deletions, preserved list) → Tasks 4 (audit), 5 (deletions), 7 (verify count)
- §2 TODO migration (T2-7b, T2-8) → Tasks 1, 2
- §3 Implementation Order → Plan order matches: ROADMAP/STATUS → audit → delete → build/test → commit
- §4 Risk R1 (cross-link audit) → Task 4
- §4 Risk R3 (recovery via git history) → Task 7 Step 2 captures SHA in commit message
- §6 Testing strategy → Task 6
- §7 Success criteria 6/6 → Task 7 Step 5 verification

No spec gaps.

**Placeholder scan:** No "TBD", "implement later", "similar to". Every step has a literal command or block.

**Type consistency:** N/A (no code types). Path strings checked: `openspec`, `ui`, `backend/TODOS.md` consistently spelled.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-25-md-cleanup.md`.

Per CLAUDE.md "Plan 실행 모드 기본값 (MANDATORY)" rule, I'll proceed with **subagent-driven execution** without re-asking. Trivial scope, but the rule applies regardless.
