package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"ai-playground/internal/domain/entity"
)

type RoomRepository struct {
	db *pgxpool.Pool
}

func NewRoomRepository(db *pgxpool.Pool) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) Save(ctx context.Context, room *entity.Room) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO rooms (id, name, game_type, visibility, join_code, host_id, max_humans, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE
		SET name=$2, status=$8, host_id=$6, updated_at=NOW()
	`,
		room.ID, room.Name, "mafia",
		string(room.Visibility), nullString(room.JoinCode),
		room.HostID, room.MaxHumans, string(room.GetStatus()),
	)
	return err
}

func (r *RoomRepository) Delete(ctx context.Context, roomID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM rooms WHERE id=$1`, roomID)
	return err
}

func (r *RoomRepository) GetByID(ctx context.Context, roomID string) (*entity.Room, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, game_type, visibility, COALESCE(join_code,''), host_id, max_humans, status
		FROM rooms WHERE id=$1
	`, roomID)
	return scanRoom(row)
}

func (r *RoomRepository) ListPublic(ctx context.Context) ([]*entity.Room, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, game_type, visibility, COALESCE(join_code,''), host_id, max_humans, status
		FROM rooms WHERE visibility='public' AND status != 'finished'
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRooms(rows)
}

func (r *RoomRepository) ListPlaying(ctx context.Context) ([]*entity.Room, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, game_type, visibility, COALESCE(join_code,''), host_id, max_humans, status
		FROM rooms WHERE status='playing'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRooms(rows)
}

type roomScanner interface {
	Scan(dest ...any) error
}

func scanRoom(row roomScanner) (*entity.Room, error) {
	var room entity.Room
	var status, visibility, gameType string
	if err := row.Scan(
		&room.ID, &room.Name, &gameType, &visibility,
		&room.JoinCode, &room.HostID, &room.MaxHumans, &status,
	); err != nil {
		return nil, err
	}
	room.Visibility = entity.Visibility(visibility)
	room.Status = entity.RoomStatus(status)
	return &room, nil
}

func scanRooms(rows interface{ Next() bool; Scan(...any) error; Err() error }) ([]*entity.Room, error) {
	var rooms []*entity.Room
	for rows.Next() {
		var room entity.Room
		var status, visibility, gameType string
		if err := rows.Scan(
			&room.ID, &room.Name, &gameType, &visibility,
			&room.JoinCode, &room.HostID, &room.MaxHumans, &status,
		); err != nil {
			return nil, err
		}
		room.Visibility = entity.Visibility(visibility)
		room.Status = entity.RoomStatus(status)
		rooms = append(rooms, &room)
	}
	return rooms, rows.Err()
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type GameResultRepository struct {
	db *pgxpool.Pool
}

func NewGameResultRepository(db *pgxpool.Pool) *GameResultRepository {
	return &GameResultRepository{db: db}
}

type GameResult struct {
	ID          string
	RoomID      string
	WinnerTeam  string
	RoundCount  int
	DurationSec int
	Players     []GameResultPlayer
}

type GameResultPlayer struct {
	ID           string
	GameResultID string
	PlayerID     string
	PlayerName   string
	Role         string
	IsAI         bool
	Survived     bool
}

func (r *GameResultRepository) Save(ctx context.Context, result GameResult) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Honor caller-provided ID so game_results.id unifies with game_metrics.game_id (T21).
	// Legacy callers that don't set ID still get a fresh UUID, preserving back-compat.
	id := result.ID
	if id == "" {
		id = newResultID()
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO game_results (id, room_id, winner_team, round_count, duration_sec)
		VALUES ($1, $2, $3, $4, $5)
	`, id, result.RoomID, result.WinnerTeam, result.RoundCount, result.DurationSec)
	if err != nil {
		return err
	}

	for _, p := range result.Players {
		_, err = tx.Exec(ctx, `
			INSERT INTO game_result_players (id, game_result_id, player_id, player_name, role, is_ai, survived)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, newResultID(), id, p.PlayerID, p.PlayerName, p.Role, p.IsAI, p.Survived)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func newResultID() string {
	return uuid.NewString()
}

// ─── Stats ───────────────────────────────────────────────────────────────────

type RoleStats struct {
	Games   int
	Wins    int
	WinRate float64
}

type PlayerStats struct {
	TotalGames int
	Wins       int
	Losses     int
	WinRate    float64
	ByRole     map[string]RoleStats
}

type PlayerGameRecord struct {
	GameID      string
	PlayedAt    string // RFC3339
	Role        string
	Survived    bool
	Won         bool
	RoundCount  int
	DurationSec int
}

// GetStatsByPlayerID aggregates win/loss stats for a human player.
func (r *GameResultRepository) GetStatsByPlayerID(ctx context.Context, playerID string) (PlayerStats, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			p.role,
			COUNT(*) AS games,
			COUNT(*) FILTER (WHERE
				(p.role = 'mafia' AND gr.winner_team = 'mafia') OR
				(p.role != 'mafia' AND gr.winner_team = 'citizen')
			) AS wins
		FROM game_result_players p
		JOIN game_results gr ON p.game_result_id = gr.id
		WHERE p.player_id = $1 AND p.is_ai = false
		GROUP BY p.role
	`, playerID)
	if err != nil {
		return PlayerStats{}, err
	}
	defer rows.Close()

	byRole := make(map[string]RoleStats)
	for rows.Next() {
		var role string
		var games, wins int
		if err := rows.Scan(&role, &games, &wins); err != nil {
			return PlayerStats{}, err
		}
		wr := 0.0
		if games > 0 {
			wr = float64(wins) / float64(games)
		}
		byRole[role] = RoleStats{Games: games, Wins: wins, WinRate: wr}
	}
	if err := rows.Err(); err != nil {
		return PlayerStats{}, err
	}

	var total, wins int
	for _, rs := range byRole {
		total += rs.Games
		wins += rs.Wins
	}
	wr := 0.0
	if total > 0 {
		wr = float64(wins) / float64(total)
	}
	return PlayerStats{
		TotalGames: total,
		Wins:       wins,
		Losses:     total - wins,
		WinRate:    wr,
		ByRole:     byRole,
	}, nil
}

// GetRecentGamesByPlayerID returns the most recent game records for a player.
func (r *GameResultRepository) GetRecentGamesByPlayerID(ctx context.Context, playerID string, limit int) ([]PlayerGameRecord, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := r.db.Query(ctx, `
		SELECT
			gr.id,
			gr.created_at,
			p.role,
			p.survived,
			(p.role = 'mafia' AND gr.winner_team = 'mafia') OR
			(p.role != 'mafia' AND gr.winner_team = 'citizen') AS won,
			gr.round_count,
			gr.duration_sec
		FROM game_result_players p
		JOIN game_results gr ON p.game_result_id = gr.id
		WHERE p.player_id = $1 AND p.is_ai = false
		ORDER BY gr.created_at DESC
		LIMIT $2
	`, playerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []PlayerGameRecord
	for rows.Next() {
		var rec PlayerGameRecord
		var playedAt interface{ String() string }
		if err := rows.Scan(
			&rec.GameID, &playedAt, &rec.Role,
			&rec.Survived, &rec.Won,
			&rec.RoundCount, &rec.DurationSec,
		); err != nil {
			return nil, err
		}
		rec.PlayedAt = playedAt.String()
		records = append(records, rec)
	}
	return records, rows.Err()
}
