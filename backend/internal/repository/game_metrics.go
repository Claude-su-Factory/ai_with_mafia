// Package repository hosts the GameMetricsRepository Postgres gateway for
// Phase A Unit Economics measurement.
//
// Column split: `game_metrics` has two kinds of columns:
//
//   - IDENTITY columns (game_id PK, room_id, started_at, humans_count, ai_count)
//     are set by Create and generally not touched by later methods. Counter
//     methods that insert-then-upsert may write placeholder values (empty
//     room_id, NOW()) on first-writer-wins, which Create repairs on a later
//     call. **Contract: Create MUST be invoked before any counter method for a
//     given game_id**, otherwise the row's identity columns are meaningless
//     (though counter values remain correct).
//   - COUNTER columns (token_*, cache_*_tokens, truncated_turns, ad_impressions_*,
//     quick_match_joins, quick_match_creates) use `col = game_metrics.col +
//     EXCLUDED.col` for idempotent multi-writer safety (spec §4a).
//   - FINAL-STATE columns (ended_at, rounds, winner, quick_match_latency_ms)
//     use EXCLUDED (overwrite). These are last-write-wins per design.
//
// Nil-pool fail-open: if constructed with nil *pgxpool.Pool all methods return
// nil without attempting I/O. This keeps gameplay unaffected by metrics infra
// outages (spec §5.2 "fail-open" policy for observability paths).
//
// Lobby sentinel: IncrementAdImpression with empty gameID writes into a daily
// row keyed by "lobby-YYYY-MM-DD" (UTC) so sentinel rollover is deterministic
// across Pods and timezone-agnostic for analytics.
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

// AIUsage carries per-turn Anthropic usage counters plus a Truncated flag.
// Truncated=true increments `truncated_turns` — spec §3-A observability hook
// for stop_reason == "max_tokens" surfacing (not originally in spec §3-D
// AIUsage struct; added via plan-level §5 "Phase A 개선 1").
type AIUsage struct {
	TokensIn, TokensOut                  int
	CacheReadTokens, CacheCreationTokens int
	Truncated                            bool
}

// GameMetricsRepository persists Phase A unit-economics counters.
// Use db as the field name to match project convention (UserRepository,
// GameResultRepository, etc.).
type GameMetricsRepository struct {
	db *pgxpool.Pool
}

func NewGameMetricsRepository(pool *pgxpool.Pool) *GameMetricsRepository {
	return &GameMetricsRepository{db: pool}
}

// Create records the initial row for a game. Should run first in the game
// lifecycle. Overwrites identity columns on conflict to repair placeholder
// rows written by counter methods that may have raced ahead.
func (r *GameMetricsRepository) Create(ctx context.Context, init GameMetricInit) error {
	if r.db == nil {
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
	_, err := r.db.Exec(ctx, q, init.GameID, init.RoomID, init.StartedAt, init.Humans, init.AIs)
	return err
}

// Finalize records end-of-game state. Caller should ensure Create has landed;
// if not, identity columns (room_id, started_at) remain placeholder until a
// later Create repairs them.
func (r *GameMetricsRepository) Finalize(ctx context.Context, f GameMetricFinal) error {
	if r.db == nil {
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
	_, err := r.db.Exec(ctx, q, f.GameID, f.EndedAt, f.Rounds, f.Winner)
	return err
}

// AddAIUsage accumulates Claude API usage counters for a game. Idempotent-additive
// via EXCLUDED.col summation — safe for concurrent writers across Pods.
func (r *GameMetricsRepository) AddAIUsage(ctx context.Context, gameID string, u AIUsage) error {
	if r.db == nil {
		return nil
	}
	truncInc := 0
	if u.Truncated {
		truncInc = 1
	}
	const q = `
		INSERT INTO game_metrics (game_id, room_id, started_at, tokens_in, tokens_out,
		                          cache_read_tokens, cache_creation_tokens, truncated_turns)
		VALUES ($1, '', NOW(), $2, $3, $4, $5, $6)
		ON CONFLICT (game_id) DO UPDATE
		  SET tokens_in             = game_metrics.tokens_in             + EXCLUDED.tokens_in,
		      tokens_out            = game_metrics.tokens_out            + EXCLUDED.tokens_out,
		      cache_read_tokens     = game_metrics.cache_read_tokens     + EXCLUDED.cache_read_tokens,
		      cache_creation_tokens = game_metrics.cache_creation_tokens + EXCLUDED.cache_creation_tokens,
		      truncated_turns       = game_metrics.truncated_turns       + EXCLUDED.truncated_turns
	`
	_, err := r.db.Exec(ctx, q, gameID, u.TokensIn, u.TokensOut, u.CacheReadTokens, u.CacheCreationTokens, truncInc)
	return err
}

// IncrementAdImpression bumps one of the three per-slot counters.
// The slot → column lookup below is an ALLOWLIST and the ONLY reason
// string-concatenating `column` into the query is safe — any unknown slot
// hits the early return and never reaches the concat. Do not relax this
// without introducing a new injection surface.
func (r *GameMetricsRepository) IncrementAdImpression(ctx context.Context, slot, gameID string) error {
	if r.db == nil {
		return nil
	}
	column := map[string]string{
		"lobby":   "ad_impressions_lobby",
		"waiting": "ad_impressions_waiting",
		"result":  "ad_impressions_result",
	}[slot]
	if column == "" {
		return nil // unknown slot silently ignored (see M4 follow-up: add observability)
	}
	roomID := ""
	if gameID == "" {
		// Lobby impression (no game) → daily sentinel row.
		// UTC so rollover is deterministic across Pods.
		gameID = "lobby-" + time.Now().UTC().Format("2006-01-02")
		roomID = "lobby"
	}
	// Column name is from the allowlist above; safe to interpolate.
	q := `
		INSERT INTO game_metrics (game_id, room_id, started_at, ` + column + `)
		VALUES ($1, $2, NOW(), 1)
		ON CONFLICT (game_id) DO UPDATE
		  SET ` + column + ` = game_metrics.` + column + ` + 1
	`
	_, err := r.db.Exec(ctx, q, gameID, roomID)
	return err
}

// RecordQuickMatch logs one quick-match attempt. Joins and creates are
// accumulated across calls; latency is last-write-wins (per-event, not
// aggregated — quick match has exactly one event per game_id in practice).
func (r *GameMetricsRepository) RecordQuickMatch(ctx context.Context, gameID, result string, latencyMs int) error {
	if r.db == nil {
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
		VALUES ($1, '', NOW(), $2, $3, $4)
		ON CONFLICT (game_id) DO UPDATE
		  SET quick_match_joins      = game_metrics.quick_match_joins      + EXCLUDED.quick_match_joins,
		      quick_match_creates    = game_metrics.quick_match_creates    + EXCLUDED.quick_match_creates,
		      quick_match_latency_ms = EXCLUDED.quick_match_latency_ms
	`
	_, err := r.db.Exec(ctx, q, gameID, joinInc, createInc, latencyMs)
	return err
}
