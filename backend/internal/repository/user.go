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
// user with a fresh UUID and sets display_name from Google JWT.
// On subsequent calls it does NOT overwrite display_name (preserving custom nicknames).
func (r *UserRepository) GetOrCreate(ctx context.Context, authID, displayName string) (string, error) {
	newPlayerID := uuid.NewString()
	var playerID string
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (auth_id, player_id, display_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (auth_id) DO UPDATE SET player_id = users.player_id
		RETURNING player_id
	`, authID, newPlayerID, displayName).Scan(&playerID)
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

// GetDisplayName returns the display_name for a player_id.
// Returns empty string (no error) if the user does not exist.
func (r *UserRepository) GetDisplayName(ctx context.Context, playerID string) (string, error) {
	var name string
	err := r.db.QueryRow(ctx,
		`SELECT display_name FROM users WHERE player_id = $1`, playerID,
	).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return name, err
}

// UpdateDisplayName sets a new display_name for the given player_id.
func (r *UserRepository) UpdateDisplayName(ctx context.Context, playerID, displayName string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET display_name = $1 WHERE player_id = $2`,
		displayName, playerID,
	)
	return err
}
