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
	TokensIn, TokensOut                  int
	CacheReadTokens, CacheCreationTokens int
	Truncated                            bool
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
		VALUES ($1, '', NOW(), $2, $3, $4, $5, $6)
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
	roomID := ""
	if gameID == "" {
		// Lobby impression (no game) → use daily sentinel row
		gameID = "lobby-" + time.Now().UTC().Format("2006-01-02")
		roomID = "lobby"
	}
	// Build query manually because column name varies
	q := `
		INSERT INTO game_metrics (game_id, room_id, started_at, ` + column + `)
		VALUES ($1, $2, NOW(), 1)
		ON CONFLICT (game_id) DO UPDATE
		  SET ` + column + ` = game_metrics.` + column + ` + 1
	`
	_, err := r.pool.Exec(ctx, q, gameID, roomID)
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
		VALUES ($1, '', NOW(), $2, $3, $4)
		ON CONFLICT (game_id) DO UPDATE
		  SET quick_match_joins      = game_metrics.quick_match_joins      + EXCLUDED.quick_match_joins,
		      quick_match_creates    = game_metrics.quick_match_creates    + EXCLUDED.quick_match_creates,
		      quick_match_latency_ms = EXCLUDED.quick_match_latency_ms
	`
	_, err := r.pool.Exec(ctx, q, gameID, joinInc, createInc, latencyMs)
	return err
}
