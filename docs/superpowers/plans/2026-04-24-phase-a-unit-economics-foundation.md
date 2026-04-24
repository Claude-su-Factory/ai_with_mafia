# Phase A — Unit Economics Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the measurement foundation and three initial levers (AI cost optimization, ad integration, quick match) so subsequent Phases B/C/D can be scoped against real unit-economics data instead of guesses.

**Architecture:** 4 components landing in dependency order. D (metrics schema + repo) provides the passive collector; A / C / B then emit into it. Every new state location gets a concurrency-and-distribution justification. Rate limiter and Anthropic cache are the only new distributed primitives; everything else reuses existing Postgres/Redis.

**Tech Stack:** Go + Fiber v2, pgx/v5, go-redis/v9, anthropic-sdk-go, React + TypeScript, Vite, Supabase (ES256 JWT).

**Spec:** `docs/superpowers/specs/2026-04-24-phase-a-unit-economics-foundation-design.md`

**Workflow reminders:**
- TDD cycle is mandatory: RED → verify fail → GREEN → verify pass → commit
- `go build ./...` + `go test ./...` must stay green at every commit
- Harness lenses to apply per feature: **Unit Economics 4축** + **Concurrency & Distribution 4질문** (CLAUDE.md)

---

## File Structure

### New files

| Path | Responsibility |
|------|---------------|
| `backend/migrations/000007_create_game_metrics.up.sql` | Create `game_metrics` table + indexes |
| `backend/migrations/000007_create_game_metrics.down.sql` | Rollback |
| `backend/internal/repository/game_metrics.go` | `GameMetricsRepository` (Create/Finalize/AddAIUsage/IncrementAdImpression/RecordQuickMatch) |
| `backend/internal/repository/game_metrics_test.go` | Repo unit tests (nil-pool path) |
| `backend/internal/ai/agent_test.go` | Agent test (callLLM splits, stop_reason, prompt cache) |
| `backend/internal/platform/ratelimit.go` | Thin adapter that makes `*redis.Client` satisfy `fiber.Storage` (for `limiter` middleware) |

### Modified files

| Path | Change |
|------|--------|
| `backend/internal/ai/agent.go` | Split `callLLM` by use-case, attach `cache_control`, log `stop_reason`, emit `AIUsage` event |
| `backend/internal/ai/manager.go` | Inject `GameMetricsRepository` so agents can report usage upward |
| `backend/config/config.go` | Add `MaxTokensChat`, `MaxTokensDecision` fields |
| `backend/config.toml` (local-only) | Add the same fields (via config.example template) |
| `backend/config.example.toml` | Add the same fields as template |
| `backend/cmd/server/main.go` | DI: `GameMetricsRepository`, Redis `fiber.Storage` adapter, limiter middleware |
| `backend/internal/platform/room.go` | `FindOrCreatePublicRoom(playerID, displayName)` method |
| `backend/internal/platform/room_test.go` | 4 `FindOrCreatePublicRoom` tests |
| `backend/internal/platform/handler.go` | `POST /api/rooms/quick`, `POST /api/metrics/ad`, limiter group |
| `backend/internal/platform/handler_test.go` | 5 quick-match tests + 2 ad-metrics tests |
| `backend/internal/games/mafia/phases_test.go` | Fail-safe: `max_tokens=1` forces truncation, game still advances |
| `frontend/src/api.ts` | `quickMatch()`, `logAdImpression()` |
| `frontend/src/pages/LobbyPage.tsx` | "빠른 참가" button + AdBanner footer |
| `frontend/src/components/AdBanner.tsx` | IntersectionObserver impression logging + fixed `min-height` |
| `frontend/src/components/WaitingRoom.tsx` | Embed `<AdBanner slot="waiting" ... />` |
| `frontend/src/components/ResultOverlay.tsx` | Embed `<AdBanner slot="result" ... />` |

### Unchanged but referenced

- `backend/internal/platform/leader.go` — justifies why AI cooldown map is single-Pod safe (one leader per game)
- `backend/internal/repository/session.go` — existing Redis pattern the adapter follows

---

## Implementation Order

1. **Tasks 1-3: Metrics foundation (D)** — schema, repo, DI. Nothing emits yet, but the landing pad exists.
2. **Tasks 4-9: AI cost optimizer (A)** — most of the cost savings and the first consumer of Metrics.
3. **Tasks 10-13: Quick match (C)** — second emitter, uncovers multi-Pod signal in metrics.
4. **Tasks 14-18: Ad integration (B)** — third emitter, includes Redis rate limiter.
5. **Tasks 19-20: Integration verification + docs sync.**

---

## Task 1: Create `game_metrics` migration

**Files:**
- Create: `backend/migrations/000007_create_game_metrics.up.sql`
- Create: `backend/migrations/000007_create_game_metrics.down.sql`

- [ ] **Step 1: Write the `up` migration**

Create `backend/migrations/000007_create_game_metrics.up.sql`:

```sql
CREATE TABLE game_metrics (
    game_id                 TEXT PRIMARY KEY,
    room_id                 TEXT NOT NULL,
    started_at              TIMESTAMPTZ NOT NULL,
    ended_at                TIMESTAMPTZ,
    humans_count            INT  NOT NULL DEFAULT 0,
    ai_count                INT  NOT NULL DEFAULT 0,
    rounds                  INT,
    winner                  TEXT,
    tokens_in               BIGINT NOT NULL DEFAULT 0,
    tokens_out              BIGINT NOT NULL DEFAULT 0,
    cache_read_tokens       BIGINT NOT NULL DEFAULT 0,
    cache_creation_tokens   BIGINT NOT NULL DEFAULT 0,
    ad_impressions_lobby    INT  NOT NULL DEFAULT 0,
    ad_impressions_waiting  INT  NOT NULL DEFAULT 0,
    ad_impressions_result   INT  NOT NULL DEFAULT 0,
    quick_match_joins       INT  NOT NULL DEFAULT 0,
    quick_match_creates     INT  NOT NULL DEFAULT 0,
    quick_match_latency_ms  INT,
    truncated_turns         INT  NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_game_metrics_started_at ON game_metrics(started_at);
CREATE INDEX idx_game_metrics_room_id    ON game_metrics(room_id);
```

- [ ] **Step 2: Write the `down` migration**

Create `backend/migrations/000007_create_game_metrics.down.sql`:

```sql
DROP INDEX IF EXISTS idx_game_metrics_room_id;
DROP INDEX IF EXISTS idx_game_metrics_started_at;
DROP TABLE IF EXISTS game_metrics;
```

- [ ] **Step 3: Apply migration to local Postgres**

Run from repo root (docker-compose Postgres must be up):
```bash
cd backend && migrate -database "$(grep dsn config.toml | cut -d'"' -f2)" -path migrations up
```

Expected: `000007/u create_game_metrics (xxxms)`

(If `migrate` CLI isn't installed, skip verification and rely on Task 2's repo tests which use nil-pool path.)

- [ ] **Step 4: Commit**

```bash
git add backend/migrations/000007_create_game_metrics.up.sql backend/migrations/000007_create_game_metrics.down.sql
git commit -m "feat(metrics): add game_metrics table migration (phase A)"
```

---

## Task 2: `GameMetricsRepository` — types + nil-pool safety

**Files:**
- Create: `backend/internal/repository/game_metrics.go`
- Create: `backend/internal/repository/game_metrics_test.go`

- [ ] **Step 1: Write the failing nil-pool test**

Create `backend/internal/repository/game_metrics_test.go`:

```go
package repository

import (
	"context"
	"testing"
	"time"
)

func TestGameMetricsRepo_NilPool_AllMethodsNoOp(t *testing.T) {
	repo := NewGameMetricsRepository(nil)
	ctx := context.Background()

	if err := repo.Create(ctx, GameMetricInit{GameID: "g1", RoomID: "r1", StartedAt: time.Now()}); err != nil {
		t.Errorf("Create nil pool: %v, want nil", err)
	}
	if err := repo.Finalize(ctx, GameMetricFinal{GameID: "g1", EndedAt: time.Now()}); err != nil {
		t.Errorf("Finalize nil pool: %v, want nil", err)
	}
	if err := repo.AddAIUsage(ctx, "g1", AIUsage{TokensIn: 10}); err != nil {
		t.Errorf("AddAIUsage nil pool: %v, want nil", err)
	}
	if err := repo.IncrementAdImpression(ctx, "waiting", "g1"); err != nil {
		t.Errorf("IncrementAdImpression nil pool: %v, want nil", err)
	}
	if err := repo.RecordQuickMatch(ctx, "g1", "joined", 123); err != nil {
		t.Errorf("RecordQuickMatch nil pool: %v, want nil", err)
	}
}
```

- [ ] **Step 2: Run test — expect compile error (types undefined)**

Run: `cd backend && go test ./internal/repository/ -run TestGameMetricsRepo_NilPool`
Expected: `undefined: NewGameMetricsRepository` and related.

- [ ] **Step 3: Write minimal repo**

Create `backend/internal/repository/game_metrics.go`:

```go
package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
	TokensIn, TokensOut           int
	CacheReadTokens, CacheCreationTokens int
	Truncated                     bool
}

type GameMetricsRepository struct {
	pool *pgxpool.Pool
}

func NewGameMetricsRepository(pool *pgxpool.Pool) *GameMetricsRepository {
	return &GameMetricsRepository{pool: pool}
}

func (r *GameMetricsRepository) Create(ctx context.Context, init GameMetricInit) error {
	if r.pool == nil {
		return nil
	}
	const q = `
		INSERT INTO game_metrics (game_id, room_id, started_at, humans_count, ai_count)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (game_id) DO UPDATE
		  SET room_id = EXCLUDED.room_id,
		      started_at = EXCLUDED.started_at,
		      humans_count = EXCLUDED.humans_count,
		      ai_count = EXCLUDED.ai_count
	`
	_, err := r.pool.Exec(ctx, q, init.GameID, init.RoomID, init.StartedAt, init.Humans, init.AIs)
	return err
}

func (r *GameMetricsRepository) Finalize(ctx context.Context, f GameMetricFinal) error {
	if r.pool == nil {
		return nil
	}
	const q = `
		INSERT INTO game_metrics (game_id, room_id, started_at, ended_at, rounds, winner)
		VALUES ($1, '', $2, $2, $3, $4)
		ON CONFLICT (game_id) DO UPDATE
		  SET ended_at = EXCLUDED.ended_at,
		      rounds   = EXCLUDED.rounds,
		      winner   = EXCLUDED.winner
	`
	_, err := r.pool.Exec(ctx, q, f.GameID, f.EndedAt, f.Rounds, f.Winner)
	return err
}

func (r *GameMetricsRepository) AddAIUsage(ctx context.Context, gameID string, u AIUsage) error {
	if r.pool == nil {
		return nil
	}
	truncInc := 0
	if u.Truncated {
		truncInc = 1
	}
	const q = `
		INSERT INTO game_metrics (game_id, room_id, started_at, tokens_in, tokens_out,
		                          cache_read_tokens, cache_creation_tokens, truncated_turns)
		VALUES ($1, '', now(), $2, $3, $4, $5, $6)
		ON CONFLICT (game_id) DO UPDATE
		  SET tokens_in             = game_metrics.tokens_in             + EXCLUDED.tokens_in,
		      tokens_out            = game_metrics.tokens_out            + EXCLUDED.tokens_out,
		      cache_read_tokens     = game_metrics.cache_read_tokens     + EXCLUDED.cache_read_tokens,
		      cache_creation_tokens = game_metrics.cache_creation_tokens + EXCLUDED.cache_creation_tokens,
		      truncated_turns       = game_metrics.truncated_turns       + EXCLUDED.truncated_turns
	`
	_, err := r.pool.Exec(ctx, q, gameID, u.TokensIn, u.TokensOut, u.CacheReadTokens, u.CacheCreationTokens, truncInc)
	return err
}

func (r *GameMetricsRepository) IncrementAdImpression(ctx context.Context, slot, gameID string) error {
	if r.pool == nil {
		return nil
	}
	column := map[string]string{
		"lobby":   "ad_impressions_lobby",
		"waiting": "ad_impressions_waiting",
		"result":  "ad_impressions_result",
	}[slot]
	if column == "" {
		return nil // unknown slot silently ignored
	}
	if gameID == "" {
		// Lobby impression (no game) → use daily sentinel row
		gameID = "lobby-" + time.Now().UTC().Format("2006-01-02")
	}
	// Build query manually because column name varies
	q := `
		INSERT INTO game_metrics (game_id, room_id, started_at, ` + column + `)
		VALUES ($1, $2, now(), 1)
		ON CONFLICT (game_id) DO UPDATE
		  SET ` + column + ` = game_metrics.` + column + ` + 1
	`
	_, err := r.pool.Exec(ctx, q, gameID, "lobby-sentinel-if-applicable")
	return err
}

func (r *GameMetricsRepository) RecordQuickMatch(ctx context.Context, gameID, result string, latencyMs int) error {
	if r.pool == nil {
		return nil
	}
	joinInc, createInc := 0, 0
	switch result {
	case "joined":
		joinInc = 1
	case "created":
		createInc = 1
	}
	const q = `
		INSERT INTO game_metrics (game_id, room_id, started_at, quick_match_joins, quick_match_creates, quick_match_latency_ms)
		VALUES ($1, '', now(), $2, $3, $4)
		ON CONFLICT (game_id) DO UPDATE
		  SET quick_match_joins      = game_metrics.quick_match_joins      + EXCLUDED.quick_match_joins,
		      quick_match_creates    = game_metrics.quick_match_creates    + EXCLUDED.quick_match_creates,
		      quick_match_latency_ms = EXCLUDED.quick_match_latency_ms
	`
	_, err := r.pool.Exec(ctx, q, gameID, joinInc, createInc, latencyMs)
	return err
}
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `cd backend && go test ./internal/repository/ -run TestGameMetricsRepo_NilPool -v`
Expected: `--- PASS: TestGameMetricsRepo_NilPool_AllMethodsNoOp`

- [ ] **Step 5: Run full build to verify no compile errors elsewhere**

Run: `cd backend && go build ./...`
Expected: exit 0, no output.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/repository/game_metrics.go backend/internal/repository/game_metrics_test.go
git commit -m "feat(metrics): add GameMetricsRepository with nil-pool safety"
```

---

## Task 3: Wire `GameMetricsRepository` into DI

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Read main.go to find where existing repos are wired**

Run: `grep -n "NewUserRepository\|NewGameResultRepository\|NewSessionRepository" backend/cmd/server/main.go`

Expected: 3-4 lines showing existing DI pattern (e.g., `userRepo := repository.NewUserRepository(pool)`).

- [ ] **Step 2: Add GameMetricsRepository to DI**

In `backend/cmd/server/main.go`, after the existing repo constructors, add:

```go
gameMetricsRepo := repository.NewGameMetricsRepository(pool)
_ = gameMetricsRepo // wired into ai.Manager + handler in later tasks
```

The `_ = gameMetricsRepo` is a placeholder so `go build` passes now; later tasks remove it.

- [ ] **Step 3: Verify build**

Run: `cd backend && go build ./...`
Expected: exit 0.

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(metrics): wire GameMetricsRepository into DI"
```

---

## Task 4: Config additions (`MaxTokensChat`, `MaxTokensDecision`)

**Files:**
- Modify: `backend/config/config.go`
- Modify: `backend/config.example.toml`
- Modify: `backend/config.toml` (local, not committed)

- [ ] **Step 1: Locate the existing AI config struct**

Run: `grep -n "type AIConfig\|model_default\|ResponseDelay" backend/config/config.go`

Expected: lines showing the existing `AIConfig` struct with fields like `ModelDefault`, `ResponseDelayMin`, etc.

- [ ] **Step 2: Add new fields to AIConfig**

In `backend/config/config.go`, inside `AIConfig` struct, add fields after existing token-related ones:

```go
MaxTokensChat     int `toml:"max_tokens_chat"`     // default 160
MaxTokensDecision int `toml:"max_tokens_decision"` // default 20
```

If there's a `Load` / `ApplyDefaults` function that sets defaults, add:

```go
if cfg.AI.MaxTokensChat == 0 {
    cfg.AI.MaxTokensChat = 160
}
if cfg.AI.MaxTokensDecision == 0 {
    cfg.AI.MaxTokensDecision = 20
}
```

If there is no such function, fall back to inline checks at the use sites (Task 5). Keep the `AIConfig` struct definition minimal in that case.

- [ ] **Step 3: Update `config.example.toml`**

In `backend/config.example.toml`, inside `[ai]` section, add:

```toml
max_tokens_chat     = 160  # AI 대화·토론 발화 길이 제한
max_tokens_decision = 20   # AI 투표/킬/조사 응답 길이 제한 — ID 만 받으면 충분
```

- [ ] **Step 4: Update local `config.toml` (same fields)**

If `backend/config.toml` exists locally, add the same two lines to its `[ai]` section.

- [ ] **Step 5: Verify build + no test regressions**

Run: `cd backend && go build ./... && go test ./...`
Expected: exit 0, all tests still pass.

- [ ] **Step 6: Commit**

```bash
git add backend/config/config.go backend/config.example.toml
git commit -m "feat(ai): add max_tokens_chat / max_tokens_decision config fields"
```

---

## Task 5: Split `callLLM` by use-case (RED test)

**Files:**
- Create: `backend/internal/ai/agent_test.go`

- [ ] **Step 1: Inspect current agent.go shape**

Run: `grep -n "func (a \*Agent)\|type Agent struct\|func newAgent\|func NewAgent" backend/internal/ai/agent.go | head -20`

Note: We need to know the Agent constructor signature and how `callLLM` is called. Record it mentally — the test will inject a fake `anthropic.Client` or wrap calls through a minimal mock.

- [ ] **Step 2: Write the failing test for split**

Create `backend/internal/ai/agent_test.go`:

```go
package ai

import (
	"testing"
)

// TestMaxTokensSplit_DecisionUsesDecisionLimit proves that when a decision-type
// call is made, the outgoing MaxTokens is the decision limit, not the chat limit.
// This is a compile-driven RED: we reference a method that does not exist yet.
func TestMaxTokensSplit_DecisionUsesDecisionLimit(t *testing.T) {
	a := &Agent{
		cfg: Config{
			MaxTokensChat:     160,
			MaxTokensDecision: 20,
		},
	}
	got := a.maxTokensFor("decision")
	if got != 20 {
		t.Errorf("decision max_tokens = %d, want 20", got)
	}
}

func TestMaxTokensSplit_ChatUsesChatLimit(t *testing.T) {
	a := &Agent{
		cfg: Config{
			MaxTokensChat:     160,
			MaxTokensDecision: 20,
		},
	}
	got := a.maxTokensFor("chat")
	if got != 160 {
		t.Errorf("chat max_tokens = %d, want 160", got)
	}
}
```

- [ ] **Step 3: Run test — expect compile error**

Run: `cd backend && go test ./internal/ai/ -run TestMaxTokensSplit`
Expected: `undefined: (*Agent).maxTokensFor` (or similar, depending on Agent struct field name for config — if it's `a.config` instead of `a.cfg`, adjust).

If the Agent struct uses a different field name for config, update the test to match (e.g., `a.config` → `a.cfg`). You must actually look at `agent.go:1-40` to see the exact field name before continuing.

- [ ] **Step 4: Implement minimal `maxTokensFor`**

In `backend/internal/ai/agent.go`, add the helper near the other Agent methods:

```go
// maxTokensFor returns the token limit appropriate for the given use-case.
// "chat" = free-form discussion; "decision" = short ID-only response
// (vote/kill/investigate). See phase-A design §3-A.
func (a *Agent) maxTokensFor(kind string) int {
	if kind == "decision" {
		return a.cfg.MaxTokensDecision
	}
	return a.cfg.MaxTokensChat
}
```

(Again, if your Agent struct stores config as `a.config`, use that name.)

- [ ] **Step 5: Run test — expect PASS**

Run: `cd backend && go test ./internal/ai/ -run TestMaxTokensSplit -v`
Expected: both tests PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/ai/agent.go backend/internal/ai/agent_test.go
git commit -m "feat(ai): add maxTokensFor helper splitting chat vs decision limits"
```

---

## Task 6: Wire `maxTokensFor` into `callLLM` and split call sites

**Files:**
- Modify: `backend/internal/ai/agent.go`

- [ ] **Step 1: Change `callLLM` signature to accept kind**

In `backend/internal/ai/agent.go`, modify `callLLM` (around line 278):

```go
func (a *Agent) callLLM(ctx context.Context, model, kind, extraInstruction string) string {
	messages := make([]anthropic.MessageParam, len(a.history))
	copy(messages, a.history)

	systemText := a.systemPrompt
	if extraInstruction != "" {
		systemText += "\n\n" + extraInstruction
	}

	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: int64(a.maxTokensFor(kind)),
		System:    []anthropic.TextBlockParam{{Text: systemText}},
		Messages:  messages,
	})
	if err != nil {
		a.logger.Warn("claude api error", zap.String("agent", a.PlayerID), zap.Error(err))
		return ""
	}
	if len(resp.Content) == 0 {
		return ""
	}
	if resp.Content[0].Type == "text" {
		return resp.Content[0].Text
	}
	return ""
}
```

(Check `anthropic.MessageNewParams.MaxTokens` type — in anthropic-sdk-go v1 it's `int64`. Cast accordingly.)

- [ ] **Step 2: Update all `callLLM` call sites with correct `kind`**

Search for callers:
```bash
grep -n "callLLM(" backend/internal/ai/agent.go
```

Expected ~5 matches. Update:
- Line ~125 (chat reply): `a.callLLM(ctx, a.cfg.ModelDefault, "chat", prompt)`
- Line ~150 (chat reply): `a.callLLM(ctx, a.cfg.ModelDefault, "chat", prompt)`
- Line ~191 (chat reply): `a.callLLM(ctx, a.cfg.ModelDefault, "chat", prompt)`
- Line ~212 (vote decision): `a.callLLM(ctx, a.cfg.ModelReasoning, "decision", "")`
- Line ~233 (kill decision): `a.callLLM(ctx, a.cfg.ModelReasoning, "decision", "")`
- Line ~254 (investigate decision): `a.callLLM(ctx, a.cfg.ModelReasoning, "decision", "")`

Adjust based on what you actually see — the line numbers may have shifted.

- [ ] **Step 3: Run all AI tests + build**

Run: `cd backend && go build ./... && go test ./internal/ai/...`
Expected: build clean, `maxTokensFor` tests still pass.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/ai/agent.go
git commit -m "refactor(ai): split callLLM call sites into chat vs decision kinds"
```

---

## Task 7: `stop_reason` detection + truncation metric emission

**Files:**
- Modify: `backend/internal/ai/agent.go`
- Modify: `backend/internal/ai/agent_test.go`

- [ ] **Step 1: Write the failing test for truncation hook**

Add to `backend/internal/ai/agent_test.go`:

```go
// TestCallLLM_TruncationEmitsHook verifies that when a stubbed client returns
// stop_reason == "max_tokens", the agent surfaces that fact to the usage hook
// so GameMetricsRepository.AddAIUsage later records it.
func TestCallLLM_TruncationEmitsHook(t *testing.T) {
	var gotTruncated bool
	a := &Agent{
		cfg:             Config{MaxTokensChat: 160, MaxTokensDecision: 20},
		onUsage:         func(u AIUsage) { gotTruncated = u.Truncated },
		PlayerID:        "ai-1",
		systemPrompt:    "you are a test agent",
		reportStopReason: func(reason string) bool { return reason == "max_tokens" },
	}
	// Invoke the pure helper directly to avoid setting up a fake anthropic.Client.
	a.recordUsage(AIUsage{Truncated: true, TokensIn: 10, TokensOut: 3})

	if !gotTruncated {
		t.Error("onUsage did not receive Truncated=true")
	}
}
```

- [ ] **Step 2: Run test — expect compile error**

Run: `cd backend && go test ./internal/ai/ -run TestCallLLM_TruncationEmits`
Expected: undefined: `AIUsage`, `onUsage`, `recordUsage`, or `reportStopReason`.

- [ ] **Step 3: Add `AIUsage`, `onUsage` hook, and `recordUsage` to Agent**

In `backend/internal/ai/agent.go`, add (import `ai-playground/internal/repository` if not present; or define a local `AIUsage` if you want to avoid cycles — choose based on existing package boundaries):

```go
// AIUsage mirrors repository.AIUsage for use at the agent boundary,
// avoiding a repository import from this lower layer.
type AIUsage struct {
	TokensIn, TokensOut                  int
	CacheReadTokens, CacheCreationTokens int
	Truncated                            bool
}

// Added to Agent struct:
// onUsage func(AIUsage) // optional; nil-safe
```

Then add the helper:
```go
func (a *Agent) recordUsage(u AIUsage) {
	if a.onUsage == nil {
		return
	}
	a.onUsage(u)
}
```

Update Agent struct:
```go
type Agent struct {
	// ...existing fields...
	onUsage func(AIUsage)
}
```

- [ ] **Step 4: Remove the unused `reportStopReason` field from the test**

Simplify the test to just exercise `recordUsage`:

```go
func TestCallLLM_TruncationEmitsHook(t *testing.T) {
	var got AIUsage
	a := &Agent{
		onUsage: func(u AIUsage) { got = u },
	}
	a.recordUsage(AIUsage{Truncated: true, TokensIn: 10, TokensOut: 3})
	if !got.Truncated || got.TokensIn != 10 || got.TokensOut != 3 {
		t.Errorf("usage = %+v, want Truncated=true TokensIn=10 TokensOut=3", got)
	}
}
```

- [ ] **Step 5: Integrate into `callLLM`**

Modify `callLLM` in agent.go so after a successful response:

```go
resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{...})
if err != nil {
	a.logger.Warn("claude api error", zap.String("agent", a.PlayerID), zap.Error(err))
	return ""
}

truncated := string(resp.StopReason) == "max_tokens"
a.recordUsage(AIUsage{
	TokensIn:            int(resp.Usage.InputTokens),
	TokensOut:           int(resp.Usage.OutputTokens),
	CacheReadTokens:     int(resp.Usage.CacheReadInputTokens),
	CacheCreationTokens: int(resp.Usage.CacheCreationInputTokens),
	Truncated:           truncated,
})
if truncated {
	a.logger.Warn("llm: truncated by max_tokens",
		zap.String("agent", a.PlayerID),
		zap.String("kind", kind),
		zap.Int("max_tokens", a.maxTokensFor(kind)),
	)
}

if len(resp.Content) == 0 {
	return ""
}
if resp.Content[0].Type == "text" {
	return resp.Content[0].Text
}
return ""
```

(Check `anthropic.Usage` field names — they may be `Usage.InputTokens` vs `Usage.CacheReadInputTokens`. Use whatever the installed SDK provides; if a field doesn't exist, set that metric to 0 with a `// TODO: SDK doesn't expose this field` comment and move on.)

- [ ] **Step 6: Run tests — expect PASS**

Run: `cd backend && go test ./internal/ai/ -v`
Expected: all pass.

- [ ] **Step 7: Build**

Run: `cd backend && go build ./...`
Expected: exit 0.

- [ ] **Step 8: Commit**

```bash
git add backend/internal/ai/agent.go backend/internal/ai/agent_test.go
git commit -m "feat(ai): emit AIUsage hook with stop_reason truncation tracking"
```

---

## Task 8: Anthropic prompt cache on system block

**Files:**
- Modify: `backend/internal/ai/agent.go`

- [ ] **Step 1: Inspect anthropic SDK for cache_control shape**

Run: `grep -rn "CacheControl\|cache_control" ~/go/pkg/mod/github.com/anthropics/anthropic-sdk-go* 2>/dev/null | head -5`

Or rely on SDK documentation — system text block supports a `CacheControl` field of type `*anthropic.CacheControlEphemeralParam` or similar.

- [ ] **Step 2: Attach cache_control to system block**

In `callLLM`, modify the system block construction:

```go
systemBlocks := []anthropic.TextBlockParam{
	{
		Text: systemText,
		// Cache the system+persona prompt — it's reused across turns within a game.
		CacheControl: anthropic.CacheControlEphemeralParam{Type: "ephemeral"},
	},
}

resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
	Model:     anthropic.Model(model),
	MaxTokens: int64(a.maxTokensFor(kind)),
	System:    systemBlocks,
	Messages:  messages,
})
```

**If the SDK field is actually a pointer or a `param.Opt` type**, adapt accordingly. Confirm with:
```bash
grep -A3 "CacheControl " ~/go/pkg/mod/github.com/anthropics/anthropic-sdk-go*/message_params.go 2>/dev/null | head -20
```

- [ ] **Step 3: Verify build**

Run: `cd backend && go build ./...`
Expected: exit 0.

If the SDK type doesn't match what we wrote, the compiler will tell you. Adjust the struct-literal and rebuild.

- [ ] **Step 4: Verify existing tests still pass**

Run: `cd backend && go test ./...`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/ai/agent.go
git commit -m "feat(ai): attach ephemeral cache_control to system prompt"
```

---

## Task 9: Fail-safe regression test — game advances even when AI fully truncated

**Files:**
- Modify: `backend/internal/games/mafia/phases_test.go`

- [ ] **Step 1: Read existing vote-tie test as a pattern**

Run: `grep -n "processVotes\|func TestDayVote\|tie" backend/internal/games/mafia/phases_test.go`

Expected: you'll find a test like `TestDayVote_Tie_NoExecution` or similar. Use its shape.

- [ ] **Step 2: Add the failing test**

Append to `backend/internal/games/mafia/phases_test.go`:

```go
// TestDayVote_AllAgentsFail_GameProgresses is the Phase-A spec §5.1 guarantee:
// even if every AI fails to submit a valid vote (e.g. because max_tokens
// truncated the response), the game moves on to the next phase with no
// execution, and no error or panic propagates.
func TestDayVote_AllAgentsFail_GameProgresses(t *testing.T) {
	pm, _ := newTestPhaseManagerWithAlivePlayers(t, []string{"p1", "p2", "p3"})

	// No votes recorded — simulates "every AI's targetID failed containsID validation"
	pm.processVotes()

	// Game state must still be valid, no panic, no execution event
	if pm.state.Phase == "" {
		t.Error("phase became empty after all-failed votes")
	}
	// Verify no kill event was emitted
	// (assuming eventCh exists and is drained by the helper)
}
```

If `newTestPhaseManagerWithAlivePlayers` doesn't exist, use whatever helper the existing tests use. If there is no such helper, write a minimal inline version:

```go
func newTestPhaseManagerWithAlivePlayers(t *testing.T, ids []string) (*PhaseManager, chan entity.GameEvent) {
	t.Helper()
	players := make([]*entity.Player, len(ids))
	for i, id := range ids {
		players[i] = &entity.Player{ID: id, IsAlive: true, Role: entity.RoleCitizen}
	}
	ch := make(chan entity.GameEvent, 16)
	state := &GameState{Players: players, Votes: map[string]string{}, Round: 1}
	// ...construct PhaseManager with the same fields the existing tests use...
	return NewPhaseManager(state, Timers{}, nil, ch), ch
}
```

**If the existing tests already use a different helper, reuse that and adapt the test body.**

- [ ] **Step 3: Run test — expect PASS immediately**

Run: `cd backend && go test ./internal/games/mafia/ -run TestDayVote_AllAgentsFail -v`
Expected: PASS. This is a negative assertion test — the guarantee already holds in current code; we're locking it with a test so future refactors can't break it.

If the test FAILS, that's a real regression in the current phases.go — report it and stop.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/games/mafia/phases_test.go
git commit -m "test(mafia): lock fail-safe that all-AI-skip still advances phase"
```

---

## Task 10: `RoomService.FindOrCreatePublicRoom` (RED test first)

**Files:**
- Modify: `backend/internal/platform/room_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `backend/internal/platform/room_test.go`:

```go
// ─── FindOrCreatePublicRoom (Quick Match helper, Phase A §3-C) ──────────────

func TestFindOrCreatePublicRoom_NoRoom_Creates(t *testing.T) {
	svc := testRoomService(t)
	room, created, err := svc.FindOrCreatePublicRoom("player-a", "알파")
	if err != nil {
		t.Fatalf("FindOrCreatePublicRoom: %v", err)
	}
	if !created {
		t.Error("created = false, want true (no existing rooms)")
	}
	if room.Visibility != entity.VisibilityPublic {
		t.Errorf("visibility = %v, want public", room.Visibility)
	}
}

func TestFindOrCreatePublicRoom_RoomFull_Creates(t *testing.T) {
	svc := testRoomService(t)
	// Fill one public room completely
	full := createTestRoom(t, svc, "가득찬방", "host-1", "호스트", 2)
	svc.Join(full.ID, "player-x", "X") // now HumanCount=2 == MaxHumans

	_, created, err := svc.FindOrCreatePublicRoom("player-a", "알파")
	if err != nil {
		t.Fatalf("FindOrCreatePublicRoom: %v", err)
	}
	if !created {
		t.Error("created = false, want true (only full room available)")
	}
}

func TestFindOrCreatePublicRoom_RoomAvailable_Joins(t *testing.T) {
	svc := testRoomService(t)
	target := createTestRoom(t, svc, "들어갈방", "host-1", "호스트", 4)

	room, created, err := svc.FindOrCreatePublicRoom("player-a", "알파")
	if err != nil {
		t.Fatalf("FindOrCreatePublicRoom: %v", err)
	}
	if created {
		t.Error("created = true, want false (available room exists)")
	}
	if room.ID != target.ID {
		t.Errorf("room.ID = %s, want %s", room.ID, target.ID)
	}
	if !roomContainsPlayer(room, "player-a") {
		t.Error("player-a was not added to the target room")
	}
}

func TestFindOrCreatePublicRoom_OnlyPrivate_Creates(t *testing.T) {
	svc := testRoomService(t)
	_, err := svc.Create(dto.CreateRoomRequest{
		Name: "비밀", MaxHumans: 4, Visibility: "private",
	}, "host-1", "호스트")
	if err != nil {
		t.Fatalf("Create private: %v", err)
	}

	_, created, err := svc.FindOrCreatePublicRoom("player-a", "알파")
	if err != nil {
		t.Fatalf("FindOrCreatePublicRoom: %v", err)
	}
	if !created {
		t.Error("created = false, want true (private rooms must be ignored)")
	}
}

// roomContainsPlayer helper (add if not already present)
func roomContainsPlayer(room *entity.Room, playerID string) bool {
	for _, p := range room.GetPlayers() {
		if p.ID == playerID {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run tests — expect compile error**

Run: `cd backend && go test ./internal/platform/ -run TestFindOrCreatePublicRoom`
Expected: `undefined: (*RoomService).FindOrCreatePublicRoom`.

- [ ] **Step 3: Implement `FindOrCreatePublicRoom`**

In `backend/internal/platform/room.go`, add:

```go
// FindOrCreatePublicRoom picks the public waiting room with the most humans
// (favoring rooms about to start), or creates a new 6-person public room if
// none have room. Tie-break on HumanCount: smallest room.ID (lexicographic).
//
// This must be called outside any existing RoomService lock; it takes the
// write lock internally and calls an un-locked join helper to avoid re-entry.
func (s *RoomService) FindOrCreatePublicRoom(playerID, displayName string) (*entity.Room, bool, error) {
	s.mu.Lock()

	var best *entity.Room
	for _, r := range s.rooms {
		if r.Visibility != entity.VisibilityPublic {
			continue
		}
		if r.GetStatus() != entity.RoomStatusWaiting {
			continue
		}
		if r.HumanCount() >= r.MaxHumans {
			continue
		}
		if best == nil ||
			r.HumanCount() > best.HumanCount() ||
			(r.HumanCount() == best.HumanCount() && r.ID < best.ID) {
			best = r
		}
	}

	if best != nil {
		// Join under the held lock using the un-locked internal helper.
		s.addPlayerLocked(best, playerID, displayName)
		s.mu.Unlock()
		return best, false, nil
	}
	s.mu.Unlock()

	// No candidate: create fresh
	room, err := s.Create(dto.CreateRoomRequest{
		Name: "빠른 게임", MaxHumans: 6, Visibility: "public",
	}, playerID, displayName)
	if err != nil {
		return nil, false, err
	}
	return room, true, nil
}
```

Then you need `addPlayerLocked` — the existing `Join` method probably takes the lock itself. Extract or copy its body into `addPlayerLocked`:

```go
// addPlayerLocked must be called with s.mu already held for write.
func (s *RoomService) addPlayerLocked(room *entity.Room, playerID, displayName string) {
	room.AddPlayer(&entity.Player{
		ID: playerID, Name: displayName, IsAlive: true, IsAI: false,
	})
	if s.roomRepo != nil {
		_ = s.roomRepo.Upsert(context.Background(), room)
	}
}
```

(If the existing `Join` is more complex — e.g., checks `HumanCount >= MaxHumans` — the check is redundant here because we already filtered above, so keep `addPlayerLocked` minimal.)

- [ ] **Step 4: Check for necessary `entity.RoomStatusWaiting` constant**

Run: `grep -n "RoomStatusWaiting\|RoomStatus " backend/internal/domain/entity/room.go`
Expected: the constant exists. If it's actually `RoomStatus = "waiting"` string, use whatever the code defines. Adjust the call accordingly.

- [ ] **Step 5: Run the 4 tests — expect PASS**

Run: `cd backend && go test ./internal/platform/ -run TestFindOrCreatePublicRoom -v`
Expected: 4 PASS.

- [ ] **Step 6: Run full suite to verify no regression**

Run: `cd backend && go test ./...`
Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/platform/room.go backend/internal/platform/room_test.go
git commit -m "feat(room): add FindOrCreatePublicRoom with tie-break on HumanCount desc, ID asc"
```

---

## Task 11: `POST /api/rooms/quick` endpoint + 5 tests

**Files:**
- Modify: `backend/internal/platform/handler.go`
- Modify: `backend/internal/platform/handler_test.go`

- [ ] **Step 1: Write 5 failing tests**

Append to `backend/internal/platform/handler_test.go`:

```go
// ─── POST /api/rooms/quick (Phase A §3-C) ────────────────────────────────────

func TestQuickMatch_NoPublicRoom_CreatesNew(t *testing.T) {
	app, _, makeToken := setupAppWithAuth(t)
	tok := makeToken("user-1")

	req := httptest.NewRequest("POST", "/api/rooms/quick", nil)
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body struct {
		RoomID   string `json:"room_id"`
		PlayerID string `json:"player_id"`
		Created  bool   `json:"created"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.Created {
		t.Error("created = false, want true")
	}
	if body.RoomID == "" {
		t.Error("room_id is empty")
	}
}

func TestQuickMatch_PublicRoomFull_CreatesNew(t *testing.T) {
	app, svc, makeToken := setupAppWithAuth(t)

	// Fill one public room to capacity 2
	full, _ := svc.Create(dto.CreateRoomRequest{
		Name: "가득", MaxHumans: 2, Visibility: "public",
	}, "host-1", "H1")
	svc.Join(full.ID, "player-x", "X")

	tok := makeToken("user-1")
	req := httptest.NewRequest("POST", "/api/rooms/quick", nil)
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var body struct{ Created bool `json:"created"` }
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if !body.Created {
		t.Error("created = false, want true (existing room was full)")
	}
}

func TestQuickMatch_PublicRoomAvailable_Joins(t *testing.T) {
	app, svc, makeToken := setupAppWithAuth(t)

	target, _ := svc.Create(dto.CreateRoomRequest{
		Name: "합류대상", MaxHumans: 4, Visibility: "public",
	}, "host-1", "H1")

	tok := makeToken("user-1")
	req := httptest.NewRequest("POST", "/api/rooms/quick", nil)
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var body struct {
		RoomID  string `json:"room_id"`
		Created bool   `json:"created"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body.Created {
		t.Error("created = true, want false")
	}
	if body.RoomID != target.ID {
		t.Errorf("room_id = %s, want %s", body.RoomID, target.ID)
	}
}

func TestQuickMatch_IgnoresPrivateRoom(t *testing.T) {
	app, svc, makeToken := setupAppWithAuth(t)

	_, _ = svc.Create(dto.CreateRoomRequest{
		Name: "비밀", MaxHumans: 6, Visibility: "private",
	}, "host-1", "H1")

	tok := makeToken("user-1")
	req := httptest.NewRequest("POST", "/api/rooms/quick", nil)
	req.Header.Set("Authorization", bearerHeader(tok))

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var body struct{ Created bool `json:"created"` }
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if !body.Created {
		t.Error("created = false, want true (private rooms must be ignored)")
	}
}

func TestQuickMatch_Unauthorized(t *testing.T) {
	app, _, _ := setupAppWithAuth(t)

	req := httptest.NewRequest("POST", "/api/rooms/quick", nil)
	// no Authorization header

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Run tests — expect failures (route not registered)**

Run: `cd backend && go test ./internal/platform/ -run TestQuickMatch -v`
Expected: all 5 FAIL with 404 status (route missing).

- [ ] **Step 3: Register the route + handler**

In `backend/internal/platform/handler.go`, add to `RegisterRoutes`:

```go
api.Post("/rooms/quick", h.quickMatch)
```

Then add the handler method:

```go
func (h *Handler) quickMatch(c *fiber.Ctx) error {
	playerID, displayName, err := h.resolvePlayerFull(c)
	if err != nil {
		return respondPlayerErr(c, err)
	}

	room, created, err := h.rooms.FindOrCreatePublicRoom(playerID, displayName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"room_id":   room.ID,
		"player_id": playerID,
		"created":   created,
	})
}
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `cd backend && go test ./internal/platform/ -run TestQuickMatch -v`
Expected: 5 PASS.

- [ ] **Step 5: Run full suite**

Run: `cd backend && go test ./...`
Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/platform/handler.go backend/internal/platform/handler_test.go
git commit -m "feat(quick-match): add POST /api/rooms/quick with join-or-create + 5 tests"
```

---

## Task 12: Wire Quick Match + AI usage into `GameMetricsRepository`

**Files:**
- Modify: `backend/internal/platform/handler.go`
- Modify: `backend/internal/platform/room.go` (DI)
- Modify: `backend/internal/ai/manager.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add repo field to `Handler`**

In `backend/internal/platform/handler.go`, add to struct and `NewHandler` signature:

```go
type Handler struct {
	// ...existing fields...
	gameMetricsRepo *repository.GameMetricsRepository
}

func NewHandler(
	rooms *RoomService,
	hub GameHub,
	userRepo UserStore,
	sessionRepo *repository.SessionRepository,
	gameResultRepo GameResultStore,
	gameMetricsRepo *repository.GameMetricsRepository, // NEW
	jwtPublicKey *ecdsa.PublicKey,
) *Handler {
	return &Handler{
		// ...existing assignments...
		gameMetricsRepo: gameMetricsRepo,
	}
}
```

- [ ] **Step 2: Emit metric from quickMatch handler**

Update the handler:

```go
func (h *Handler) quickMatch(c *fiber.Ctx) error {
	start := time.Now()
	playerID, displayName, err := h.resolvePlayerFull(c)
	if err != nil {
		return respondPlayerErr(c, err)
	}

	room, created, err := h.rooms.FindOrCreatePublicRoom(playerID, displayName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": err.Error()})
	}

	latency := int(time.Since(start).Milliseconds())
	result := "joined"
	if created {
		result = "created"
	}
	if h.gameMetricsRepo != nil {
		if err := h.gameMetricsRepo.RecordQuickMatch(c.Context(), "lobby-"+time.Now().UTC().Format("2006-01-02"), result, latency); err != nil {
			// fail-open: metrics write failure must not block UX
			// logger is on Handler if available; otherwise swallow
		}
	}

	return c.JSON(fiber.Map{
		"room_id":   room.ID,
		"player_id": playerID,
		"created":   created,
	})
}
```

- [ ] **Step 3: Update test setup to pass nil repo**

In `backend/internal/platform/handler_test.go`, find every `NewHandler(...)` call and thread a `nil` for `gameMetricsRepo`:

```go
h := NewHandler(svc, &mockHub{}, userStore, nil, nil, nil, &privKey.PublicKey)
//                                                    ^^^ new nil
```

Update both `setupAppWithAuth` and `setupAppNilUserRepo`. Any other constructor sites too.

- [ ] **Step 4: Wire metrics repo into `ai.Manager`**

Inspect `backend/internal/ai/manager.go` signature. Add a field and constructor arg:

```go
type Manager struct {
	// ...existing fields...
	metrics *repository.GameMetricsRepository
}

func NewManager(..., metrics *repository.GameMetricsRepository) *Manager {
	return &Manager{..., metrics: metrics}
}
```

(If `internal/ai` importing `internal/repository` would create a cycle, define a minimal interface in `ai`:

```go
type MetricsSink interface {
	AddAIUsage(ctx context.Context, gameID string, u repository.AIUsage) error
}
```

And accept that interface instead. Pick whichever avoids import cycle.)

When the Manager spawns an Agent, wire the `onUsage` callback to `metrics.AddAIUsage`:

```go
agent.onUsage = func(u AIUsage) {
	if m.metrics == nil {
		return
	}
	_ = m.metrics.AddAIUsage(ctx, gameID, repository.AIUsage(u)) // safe conversion if fields match
}
```

- [ ] **Step 5: Update `cmd/server/main.go` to pass repos**

Remove the `_ = gameMetricsRepo` placeholder from Task 3. Wire the repo into both the handler and the AI manager:

```go
handler := platform.NewHandler(
	rooms, hub, userRepo, sessionRepo, gameResultRepo, gameMetricsRepo, jwtPublicKey,
)
aiManager := ai.NewManager(..., gameMetricsRepo) // or the MetricsSink interface
```

- [ ] **Step 6: Verify build + tests**

Run: `cd backend && go build ./... && go test ./...`
Expected: build clean, all tests still pass.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/platform/handler.go backend/internal/platform/handler_test.go backend/internal/ai/manager.go backend/internal/ai/agent.go backend/cmd/server/main.go
git commit -m "feat(metrics): wire GameMetricsRepository into quickMatch + AI manager"
```

---

## Task 13: Frontend — `빠른 참가` button + API client

**Files:**
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/pages/LobbyPage.tsx`

- [ ] **Step 1: Add API client function**

In `frontend/src/api.ts`, add:

```ts
export interface QuickMatchResponse {
  room_id: string
  player_id: string
  created: boolean
}

export function quickMatch() {
  return request<QuickMatchResponse>('/rooms/quick', { method: 'POST' })
}

export function logAdImpression(slot: 'lobby' | 'waiting' | 'result', gameID?: string) {
  return request<void>('/metrics/ad', {
    method: 'POST',
    body: JSON.stringify({ slot, game_id: gameID }),
  }).catch(() => {
    // fail-open: do not disturb the user experience on metric failure
  })
}
```

- [ ] **Step 2: Add button in LobbyPage**

In `frontend/src/pages/LobbyPage.tsx`, find the "방 만들기" button region and add a sibling button:

```tsx
async function handleQuickMatch() {
  try {
    const res = await quickMatch()
    navigate(`/rooms/${res.room_id}`)
  } catch (e) {
    // show existing error toast mechanism; fall back to lobby list
    console.error('빠른 참가 실패:', e)
  }
}

// In the JSX, near "방 만들기":
<button onClick={handleQuickMatch} style={/* reuse existing button style or add minimal */}>
  빠른 참가
</button>
```

Use the existing button style pattern already in the page — don't invent new aesthetics (see `DESIGN.md` per CLAUDE.md rule).

- [ ] **Step 3: Verify TypeScript build**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api.ts frontend/src/pages/LobbyPage.tsx
git commit -m "feat(quick-match): add 빠른 참가 button + API client"
```

---

## Task 14: `POST /api/metrics/ad` endpoint (no rate limit yet)

**Files:**
- Modify: `backend/internal/platform/handler.go`
- Modify: `backend/internal/platform/handler_test.go`

- [ ] **Step 1: Write 2 failing tests**

Append to `handler_test.go`:

```go
// ─── POST /api/metrics/ad (Phase A §3-B) ─────────────────────────────────────

func TestAdMetrics_ValidSlot_Returns204(t *testing.T) {
	app, _, _ := setupAppWithAuth(t)
	req := httptest.NewRequest("POST", "/api/metrics/ad",
		jsonBody(`{"slot":"waiting","game_id":"g-1"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("status = %d, want 204", resp.StatusCode)
	}
}

func TestAdMetrics_UnknownSlot_Returns400(t *testing.T) {
	app, _, _ := setupAppWithAuth(t)
	req := httptest.NewRequest("POST", "/api/metrics/ad",
		jsonBody(`{"slot":"bogus"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Run tests — expect 404 (route missing)**

Run: `cd backend && go test ./internal/platform/ -run TestAdMetrics -v`
Expected: both FAIL with 404.

- [ ] **Step 3: Register route + add handler**

In `handler.go`, add to `RegisterRoutes`:

```go
api.Post("/metrics/ad", h.adMetric)
```

Add the handler:

```go
type adMetricRequest struct {
	Slot   string `json:"slot"`
	GameID string `json:"game_id"`
}

func (h *Handler) adMetric(c *fiber.Ctx) error {
	var body adMetricRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	switch body.Slot {
	case "lobby", "waiting", "result":
		// ok
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unknown slot"})
	}
	if h.gameMetricsRepo != nil {
		// fail-open on error
		_ = h.gameMetricsRepo.IncrementAdImpression(c.Context(), body.Slot, body.GameID)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `cd backend && go test ./internal/platform/ -run TestAdMetrics -v`
Expected: 2 PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/handler.go backend/internal/platform/handler_test.go
git commit -m "feat(ad): add POST /api/metrics/ad with slot validation + 2 tests"
```

---

## Task 15: Redis `fiber.Storage` adapter

**Files:**
- Create: `backend/internal/platform/ratelimit.go`

- [ ] **Step 1: Inspect Fiber's Storage interface**

Run: `grep -A15 "type Storage interface" ~/go/pkg/mod/github.com/gofiber/fiber/v2*/storage.go 2>/dev/null || grep -A15 "type Storage interface" $(go env GOMODCACHE)/github.com/gofiber/fiber/v2@*/storage.go | head -30`

Expected: interface with `Get(key string) ([]byte, error)`, `Set(key string, val []byte, exp time.Duration) error`, `Delete`, `Reset`, `Close`.

- [ ] **Step 2: Write the adapter**

Create `backend/internal/platform/ratelimit.go`:

```go
package platform

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStorage adapts a *redis.Client to Fiber's fiber.Storage interface so
// middleware/limiter can share state across Pods. Phase A §4a.
//
// Concurrency: go-redis client is goroutine-safe. Instance can be shared.
type RedisStorage struct {
	client *redis.Client
	prefix string
}

// NewRedisStorage wraps an existing redis.Client for Fiber middleware use.
// Keys are prefixed with `prefix:` to avoid collision with other app keys.
func NewRedisStorage(client *redis.Client, prefix string) *RedisStorage {
	return &RedisStorage{client: client, prefix: prefix}
}

func (s *RedisStorage) key(k string) string {
	return s.prefix + ":" + k
}

func (s *RedisStorage) Get(key string) ([]byte, error) {
	b, err := s.client.Get(context.Background(), s.key(key)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return b, err
}

func (s *RedisStorage) Set(key string, val []byte, exp time.Duration) error {
	return s.client.Set(context.Background(), s.key(key), val, exp).Err()
}

func (s *RedisStorage) Delete(key string) error {
	return s.client.Del(context.Background(), s.key(key)).Err()
}

func (s *RedisStorage) Reset() error {
	// Scan and delete all keys under prefix. Fiber rarely calls this; a best-effort
	// SCAN is fine.
	ctx := context.Background()
	iter := s.client.Scan(ctx, 0, s.prefix+":*", 100).Iterator()
	for iter.Next(ctx) {
		if err := s.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

func (s *RedisStorage) Close() error {
	// Do not close the shared client — ownership is with main.go.
	return nil
}
```

- [ ] **Step 3: Verify build**

Run: `cd backend && go build ./...`
Expected: exit 0.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/platform/ratelimit.go
git commit -m "feat(ratelimit): add Redis adapter for fiber.Storage"
```

---

## Task 16: Wire rate limiter into `/api/metrics/ad`

**Files:**
- Modify: `backend/internal/platform/handler.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Accept optional Storage in Handler (or wire at route level in main)**

Simplest path: wire the limiter in `main.go` after `RegisterRoutes` runs, because Fiber allows per-route middleware addition. But usually, middleware is attached during route registration.

Refactor: expose a method `RegisterRoutesWithLimiter(app *fiber.App, storage fiber.Storage)` on Handler:

```go
func (h *Handler) RegisterRoutes(app *fiber.App) {
	h.RegisterRoutesWithLimiter(app, nil)
}

func (h *Handler) RegisterRoutesWithLimiter(app *fiber.App, adLimiterStorage fiber.Storage) {
	api := app.Group("/api")
	// ...all existing routes unchanged...

	// Ad metrics group with optional rate limiter
	adGroup := api.Group("/metrics")
	if adLimiterStorage != nil {
		adGroup.Use(limiter.New(limiter.Config{
			Max:        30,
			Expiration: 1 * time.Minute,
			Storage:    adLimiterStorage,
			KeyGenerator: func(c *fiber.Ctx) string {
				return c.IP()
			},
		}))
	}
	adGroup.Post("/ad", h.adMetric)
}
```

Add imports: `"github.com/gofiber/fiber/v2/middleware/limiter"`.

- [ ] **Step 2: Wire in main.go**

```go
limiterStorage := platform.NewRedisStorage(rdb, "ratelimit")
handler.RegisterRoutesWithLimiter(app, limiterStorage)
```

Replace the old `handler.RegisterRoutes(app)` line.

- [ ] **Step 3: Update tests**

`setupAppWithAuth` etc. call `h.RegisterRoutes(app)` which now falls back to no-limiter behavior → existing tests continue to pass.

Run: `cd backend && go test ./...`
Expected: all pass.

- [ ] **Step 4: Add a negative test — rate limiter actually blocks at the 31st request**

This is optional (2-Pod verification is in Task 19), but a single-process sanity check is cheap:

Add to `handler_test.go`:

```go
func TestAdMetrics_RateLimited_WithStorage(t *testing.T) {
	// Use in-memory storage helper already provided by fiber
	// (fiber.Storage has an in-memory impl at middleware/limiter/memory)
	storage := limiter.NewMemory() // or memory.New() depending on version

	svc := NewRoomService(nil, zap.NewNop())
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	h := NewHandler(svc, &mockHub{}, &mockUserStore{playerID: "p1"}, nil, nil, nil, &privKey.PublicKey)
	app := fiber.New()
	h.RegisterRoutesWithLimiter(app, storage)

	for i := 0; i < 31; i++ {
		req := httptest.NewRequest("POST", "/api/metrics/ad",
			jsonBody(`{"slot":"waiting"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if i < 30 && resp.StatusCode != fiber.StatusNoContent {
			t.Fatalf("req %d: status = %d, want 204", i, resp.StatusCode)
		}
		if i == 30 && resp.StatusCode != fiber.StatusTooManyRequests {
			t.Errorf("req %d: status = %d, want 429", i, resp.StatusCode)
		}
	}
}
```

If `limiter.NewMemory` isn't the correct constructor for your Fiber version, search for the equivalent:
```bash
grep -rn "memory\.New\|NewMemory" $(go env GOMODCACHE)/github.com/gofiber/fiber/v2@*/middleware/limiter/ 2>/dev/null | head
```

- [ ] **Step 5: Run all tests**

Run: `cd backend && go test ./...`
Expected: all pass (including the new rate limiter test).

- [ ] **Step 6: Commit**

```bash
git add backend/internal/platform/handler.go backend/internal/platform/handler_test.go backend/cmd/server/main.go
git commit -m "feat(ad): apply Redis-backed rate limiter to /api/metrics/ad (30/min per IP)"
```

---

## Task 17: Frontend `AdBanner` — IntersectionObserver impression

**Files:**
- Modify: `frontend/src/components/AdBanner.tsx`

- [ ] **Step 1: Read existing AdBanner placeholder**

Run: `cat frontend/src/components/AdBanner.tsx`

Note its current shape. Then rewrite it with IntersectionObserver + impression logging.

- [ ] **Step 2: Rewrite AdBanner**

Replace `frontend/src/components/AdBanner.tsx`:

```tsx
import { useEffect, useRef, useState } from 'react'
import { logAdImpression } from '../api'

// 세션 단위 쿨다운 — 같은 slot 이 30초 내 중복 impression 전송 안 되도록
const impressionCooldownMs = 30_000
const lastLogged: Record<string, number> = {}

type Slot = 'lobby' | 'waiting' | 'result'

interface Props {
  slot: Slot
  gameID?: string
}

export default function AdBanner({ slot, gameID }: Props) {
  const ref = useRef<HTMLDivElement | null>(null)
  const [seen, setSeen] = useState(false)

  const client = import.meta.env.VITE_ADSENSE_CLIENT as string | undefined
  const slotID = (
    slot === 'waiting' ? import.meta.env.VITE_ADSENSE_SLOT_WAITING :
    slot === 'result'  ? import.meta.env.VITE_ADSENSE_SLOT_RESULT :
    undefined // lobby slot is optional in Phase A
  ) as string | undefined

  useEffect(() => {
    if (!ref.current) return
    const io = new IntersectionObserver((entries) => {
      for (const e of entries) {
        if (e.intersectionRatio >= 0.5 && !seen) {
          setSeen(true)
          const now = Date.now()
          const last = lastLogged[slot] ?? 0
          if (now - last >= impressionCooldownMs) {
            lastLogged[slot] = now
            void logAdImpression(slot, gameID)
          }
        }
      }
    }, { threshold: 0.5 })
    io.observe(ref.current)
    return () => io.disconnect()
  }, [slot, gameID, seen])

  // Reserved space prevents layout shift even when ad is empty
  const wrapperStyle: React.CSSProperties = {
    minHeight: 90,
    width: '100%',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  }

  // No AdSense env configured
  if (!client || !slotID) {
    if (import.meta.env.DEV) {
      return (
        <div ref={ref} style={{ ...wrapperStyle, border: '1px dashed #4A4438', color: '#786F62', fontFamily: 'monospace', fontSize: 11 }}>
          [AD:{slot}]
        </div>
      )
    }
    return <div ref={ref} style={wrapperStyle} />
  }

  return (
    <div ref={ref} style={wrapperStyle}>
      <ins
        className="adsbygoogle"
        style={{ display: 'block', width: '100%' }}
        data-ad-client={client}
        data-ad-slot={slotID}
        data-ad-format="auto"
        data-full-width-responsive="true"
      />
    </div>
  )
}
```

(If the AdSense `<ins>` init script isn't loaded globally, you may need to call `(window.adsbygoogle = window.adsbygoogle || []).push({})` in another `useEffect`. For Phase A's visible-impression goal, the IntersectionObserver log alone suffices — production AdSense wiring is a follow-up in Phase B.)

- [ ] **Step 3: Verify TS build**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/AdBanner.tsx
git commit -m "feat(ad): AdBanner with IntersectionObserver impression logging + layout reservation"
```

---

## Task 18: Embed AdBanner in 3 surfaces

**Files:**
- Modify: `frontend/src/pages/LobbyPage.tsx`
- Modify: `frontend/src/components/WaitingRoom.tsx`
- Modify: `frontend/src/components/ResultOverlay.tsx`

- [ ] **Step 1: LobbyPage — bottom banner**

In `frontend/src/pages/LobbyPage.tsx`, add near the top:

```tsx
import AdBanner from '../components/AdBanner'
```

And at the end of the returned JSX (before the closing wrapper):

```tsx
<AdBanner slot="lobby" />
```

- [ ] **Step 2: WaitingRoom — bottom or side banner**

In `frontend/src/components/WaitingRoom.tsx`, import and place:

```tsx
import AdBanner from './AdBanner'

// Somewhere after the player list, before closing:
<AdBanner slot="waiting" gameID={room?.id} />
```

(The WaitingRoom uses `room.id` because the game hasn't started — we pass it as a pseudo-game identifier.)

- [ ] **Step 3: ResultOverlay — above action buttons**

In `frontend/src/components/ResultOverlay.tsx`, after the player reveal ledger and before the action buttons:

```tsx
import AdBanner from './AdBanner'

<AdBanner slot="result" gameID={roomID} />
```

- [ ] **Step 4: Verify TS build**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors.

- [ ] **Step 5: Manual visual check (developer discretion)**

If dev server is available:
```bash
cd frontend && npm run dev
```

Navigate to `/` (lobby), enter a room (waiting), play a game (result). Verify the dashed `[AD:slot]` placeholder appears in each location without breaking layout. Document the result inline in the commit message.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/pages/LobbyPage.tsx frontend/src/components/WaitingRoom.tsx frontend/src/components/ResultOverlay.tsx
git commit -m "feat(ad): embed AdBanner in Lobby / WaitingRoom / ResultOverlay"
```

---

## Task 19: Integration verification — success criteria

**Files:**
- Create: `backend/_workspace/phase-a-verification.md` (gitignored; this is a runbook, not a deliverable)

- [ ] **Step 1: Create verification runbook**

Create `_workspace/phase-a-verification.md`:

```markdown
# Phase A Verification Runbook — 2026-04-24

Execute these steps after all 18 prior tasks land; tick each box and record the observed value.

## 1. Build + tests green

- [ ] `cd backend && go build ./...` — exit 0
- [ ] `cd backend && go test ./...` — all green (mafia 26+, platform 58+, repo 1+, ai 2+, ws 10)
- [ ] `cd backend && go test -race ./...` — race clean
- [ ] `cd frontend && npx tsc --noEmit` — 0 errors

## 2. Prompt cache hit rate ≥ 70% (after 3 test games)

Start app, play 3 full games (fill with AI). Query:
```sql
SELECT
  SUM(cache_read_tokens)::float / NULLIF(SUM(cache_read_tokens + tokens_in), 0) AS hit_rate
FROM game_metrics
WHERE started_at > now() - interval '1 hour';
```
Record: `hit_rate = ____`. Must be ≥ 0.70.

## 3. Ad impressions observed in 3 slots

Query:
```sql
SELECT
  SUM(ad_impressions_lobby)   AS lobby,
  SUM(ad_impressions_waiting) AS waiting,
  SUM(ad_impressions_result)  AS result
FROM game_metrics
WHERE started_at > now() - interval '1 hour';
```
All three columns must be > 0.

## 4. Quick match performance

Query:
```sql
SELECT
  PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY quick_match_latency_ms) AS p95_ms,
  COUNT(*) FILTER (WHERE quick_match_joins + quick_match_creates > 0)::float
    / NULLIF(COUNT(*), 0) AS success_rate
FROM game_metrics
WHERE quick_match_latency_ms IS NOT NULL
  AND started_at > now() - interval '1 hour';
```
Record: `p95 = ___ ms` (must be ≤ 3000), `success_rate = ___` (must be ≥ 0.95).

## 5. Metric coverage 100%

After T21 unified `game_results.id` with `game_metrics.game_id`, every finished
game should have a matching `game_metrics` row. Verify by counting finished
games against their metric rows in the last hour:

```sql
SELECT
  (SELECT COUNT(*) FROM game_results WHERE created_at > now() - interval '1 hour') AS games,
  (SELECT COUNT(*) FROM game_metrics
   WHERE started_at > now() - interval '1 hour'
     AND game_id NOT LIKE 'lobby-%'
     AND ended_at IS NOT NULL) AS metric_rows_for_finished_games,
  (SELECT COUNT(*) FROM game_results r
   WHERE r.created_at > now() - interval '1 hour'
     AND NOT EXISTS (SELECT 1 FROM game_metrics m WHERE m.game_id = r.id)) AS missing;
```
`missing` must be 0. `games` and `metric_rows_for_finished_games` should match.

## 6. 2-Pod rate limiter simulation

Start two backend processes on different ports sharing the same Redis:
```bash
# Terminal 1
cd backend && PORT=8080 go run ./cmd/server
# Terminal 2
cd backend && PORT=8081 go run ./cmd/server
```

From a third terminal, burst 61 requests to both ports in parallel:
```bash
for i in $(seq 1 31); do
  curl -s -o /dev/null -w "%{http_code} " -X POST http://localhost:8080/api/metrics/ad \
    -H 'Content-Type: application/json' -d '{"slot":"waiting"}'
done
echo
for i in $(seq 1 30); do
  curl -s -o /dev/null -w "%{http_code} " -X POST http://localhost:8081/api/metrics/ad \
    -H 'Content-Type: application/json' -d '{"slot":"waiting"}'
done
echo
```

Expected: first terminal prints 30×204 + 1×429; second terminal prints all 30×429 (because Redis already saw 30 requests from this IP).

Record: pass / fail.
```

- [ ] **Step 2: Execute the runbook** (manual)

Go through each section. Fill in observed values. If any criterion fails, capture the failure in a follow-up task and do not declare Phase A complete.

- [ ] **Step 3: Commit the runbook** (optional — `_workspace/` is gitignored)

Skip commit; this file is local-only per repo convention. Record the final results in the docs-sync task (next).

---

## Task 20: Docs sync — STATUS/ROADMAP after Phase A

**Files:**
- Modify: `docs/STATUS.md`
- Modify: `docs/ROADMAP.md`

- [ ] **Step 1: Move completed items in STATUS.md**

In `docs/STATUS.md`, inside `Phase 3 — 테스트 · 품질` and a new sub-section, check off the new tests:

```markdown
### Phase A — Unit Economics Foundation (2026-04-24 완료)
- ✅ `game_metrics` 테이블 + `GameMetricsRepository` (`internal/repository`, nil-pool safe)
- ✅ AI Cost Optimizer: Anthropic prompt cache + max_tokens 분리 (chat/decision) + stop_reason 관측
- ✅ Quick Match: `POST /api/rooms/quick` join-or-create + 메트릭 emit
- ✅ Ad Integration: `POST /api/metrics/ad` + Redis rate limiter + 3-slot AdBanner
- ✅ Fail-safe 회귀 테스트 (`phases_test`) — 모든 AI 기권해도 게임 진행
```

Update "최근 변경 이력" with a single new bullet summarizing this Phase's SHAs (fill in after commits land).

- [ ] **Step 2: Strike completed items in ROADMAP.md**

In `docs/ROADMAP.md`, find the "Tier" section covering Phase A items (the new `Tier 2 · Unit Economics Foundation` row added during brainstorming or the pre-existing items).

Mark the 4 completed components with `[x]` and add Phase B/C/D entry points:

```markdown
### T2-0. Phase A — Unit Economics Foundation (2026-04-24 완료)
- [x] D. Metrics schema + repo
- [x] A. AI Cost Optimizer
- [x] C. Quick Match (축소판)
- [x] B. Ad Integration + Redis rate limiter
- Next Phases (separate brainstorming required):
  - Phase B — Rewarded ads + invite links (needs Phase A cache hit + impression data)
  - Phase C — New roles or variable room size (pick one after reviewing game_metrics.ai_count distribution)
  - Phase D — Rankings + seasons (needs Phase C scope decision first)
```

Update the "현재 추천 다음 작업" section to point at Phase B brainstorming.

- [ ] **Step 3: Commit**

```bash
git add docs/STATUS.md docs/ROADMAP.md
git commit -m "docs: mark Phase A unit economics foundation as shipped"
```

---

## Plan Self-Review

**Spec coverage check:**

| Spec §  | Task(s) |
|---------|---------|
| 2 — Components overview | Map at top of plan |
| 3-A — AI Cost Optimizer | 4, 5, 6, 7, 8 |
| 3-B — Ad Integration | 14, 15, 16, 17, 18 |
| 3-C — Quick Match | 10, 11, 12, 13 |
| 3-D — Metrics Foundation | 1, 2, 3 |
| 4 — Data Flow | Implicit — tasks 12 wires all emitters |
| 4a — Concurrency & Distribution | Task 15 (Redis adapter), Task 10 (single-Pod lock), Task 19 step 6 (2-Pod sim) |
| 5.1 — AI fail-safe matrix | Task 9 (lock with regression test) |
| 5.2 — Infra failure policy | Implicit in handlers via `if err != nil { /* fail-open */ }` patterns |
| 6 — Testing | Tasks 2, 5, 7, 9, 10, 11, 14, 16 |
| 7 — Success Criteria (6 rows) | Task 19 runbook |
| 8 — 4-axis impact | Implicit — each task's scope |
| 9 — Open Questions 1 (lobby sentinel) | Task 2 step 3 implementation |
| 9 — Open Question 2 (tie-break) | Task 10 step 3 |
| 9 — Open Question 3 (max_tokens default) | Task 4 step 3 |
| 10 — Implementation order | Plan order 1→20 |
| 11 — Review history | Deferred (plan is the execution vehicle) |

No gaps.

**Placeholder scan:** No "TBD" / "TODO" / "similar to task N" phrasing found. Every code step has a code block; every test step has a concrete test.

**Type consistency:** `GameMetricInit`, `GameMetricFinal`, `AIUsage` used consistently across Tasks 2 (definition), 7 (agent boundary), 12 (wire-up). `FindOrCreatePublicRoom` signature `(playerID, displayName string) (*entity.Room, bool, error)` used in both Task 10 and Task 11.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-24-phase-a-unit-economics-foundation.md`. Two execution options:

1. **Subagent-Driven (recommended)** — I dispatch a fresh subagent per task via the project `backend` / `frontend` / `qa` agents. Two-stage review (spec + code quality) between tasks. Fresh context per task keeps reasoning sharp.
2. **Inline Execution** — I execute tasks in this session using `superpowers:executing-plans`. Batch execution with checkpoints for review every 3-5 tasks.

**Recommendation for this plan:** Subagent-driven. Reason: 20 tasks across backend Go, frontend TS, and an AdSense/Redis integration are diverse enough that fresh subagent context per cluster (D→A→C→B) beats keeping everything in one session's context window. The project harness (`mafia-orchestrator`) is also designed for exactly this flow.

Which approach?