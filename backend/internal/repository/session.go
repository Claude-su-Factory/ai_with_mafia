package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const sessionTTL = 24 * time.Hour

type SessionRepository struct {
	rdb *redis.Client
}

func NewSessionRepository(rdb *redis.Client) *SessionRepository {
	return &SessionRepository{rdb: rdb}
}

// Set stores player → room mapping with 24-hour TTL.
func (r *SessionRepository) Set(ctx context.Context, playerID, roomID string) error {
	return r.rdb.Set(ctx, "user_session:"+playerID, roomID, sessionTTL).Err()
}

// Get returns the room_id for a player. Returns empty string (no error) if not found.
func (r *SessionRepository) Get(ctx context.Context, playerID string) (string, error) {
	roomID, err := r.rdb.Get(ctx, "user_session:"+playerID).Result()
	if err == redis.Nil {
		return "", nil
	}
	return roomID, err
}

// Delete removes the session entry for a player.
func (r *SessionRepository) Delete(ctx context.Context, playerID string) error {
	return r.rdb.Del(ctx, "user_session:"+playerID).Err()
}
