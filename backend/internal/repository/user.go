package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// GetOrCreate returns the player_id for auth_id. On first call it creates a new
// user with a fresh UUID. On subsequent calls it updates display_name and returns
// the existing player_id (PostgreSQL UPSERT returns the existing row's player_id).
func (r *UserRepository) GetOrCreate(ctx context.Context, authID, displayName string) (string, error) {
	playerID := uuid.NewString()
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (auth_id, player_id, display_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (auth_id) DO UPDATE SET display_name = EXCLUDED.display_name
		RETURNING player_id
	`, authID, playerID, displayName).Scan(&playerID)
	return playerID, err
}

// GetByAuthID returns the player_id for an existing user.
// Returns empty string (no error) if the user does not exist.
func (r *UserRepository) GetByAuthID(ctx context.Context, authID string) (string, error) {
	var playerID string
	err := r.db.QueryRow(ctx,
		`SELECT player_id FROM users WHERE auth_id = $1`, authID,
	).Scan(&playerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return playerID, err
}
