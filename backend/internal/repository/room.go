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

	id := newResultID()
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
