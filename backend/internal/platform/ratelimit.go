package platform

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
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
	if errors.Is(err, redis.Nil) {
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

// Reset scans and deletes all keys under prefix. Fiber rarely calls this;
// a best-effort SCAN is acceptable for the middleware/limiter use-case.
func (s *RedisStorage) Reset() error {
	ctx := context.Background()
	iter := s.client.Scan(ctx, 0, s.prefix+":*", 100).Iterator()
	for iter.Next(ctx) {
		if err := s.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// Close is a no-op. The underlying redis.Client is owned by main.go — closing
// it here would bring down all other Redis users (SessionRepository,
// LeaderLock, pub/sub relay).
func (s *RedisStorage) Close() error {
	return nil
}

// Compile-time check that RedisStorage implements fiber.Storage.
var _ fiber.Storage = (*RedisStorage)(nil)
