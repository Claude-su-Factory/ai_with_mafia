package platform

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const leaderTTL = 30 * time.Second

type LeaderLock struct {
	rdb *redis.Client
}

func NewLeaderLock(rdb *redis.Client) *LeaderLock {
	return &LeaderLock{rdb: rdb}
}

// Acquire attempts to acquire the leader lock for a room.
// Returns true if this instance is now the leader.
func (l *LeaderLock) Acquire(ctx context.Context, roomID, instanceID string) bool {
	key := leaderKey(roomID)
	result, err := l.rdb.SetArgs(ctx, key, instanceID, redis.SetArgs{
		TTL:  leaderTTL,
		Mode: "NX",
	}).Result()
	if err != nil {
		return false
	}
	return result == "OK"
}

// Release releases the leader lock for a room.
func (l *LeaderLock) Release(ctx context.Context, roomID string) {
	l.rdb.Del(ctx, leaderKey(roomID))
}

// Heartbeat refreshes the TTL of the leader lock.
func (l *LeaderLock) Heartbeat(ctx context.Context, roomID string) {
	l.rdb.Expire(ctx, leaderKey(roomID), leaderTTL)
}

// HasLeader checks whether any instance holds the lock for a room.
func (l *LeaderLock) HasLeader(ctx context.Context, roomID string) bool {
	val, err := l.rdb.Get(ctx, leaderKey(roomID)).Result()
	return err == nil && val != ""
}

func leaderKey(roomID string) string {
	return "game:" + roomID + ":leader"
}
