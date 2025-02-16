package limiter

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisStorage struct {
	client *redis.Client
}

// NewRedisStorage initializes a Redis-based LimiterStorage.
func NewRedisStorage(client *redis.Client) *RedisStorage {
	return &RedisStorage{client: client}
}

// Increment increments the request counter for the given key.
func (r *RedisStorage) Increment(ctx context.Context, key string, expirationSeconds int) (int64, error) {
	pipe := r.client.TxPipeline()

	// Increment the counter
	count := pipe.Incr(ctx, key)

	// Set expiration time for the key
	pipe.Expire(ctx, key, time.Duration(expirationSeconds)*time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return count.Val(), nil
}

// IsBlocked checks if a key is marked as blocked.
func (r *RedisStorage) IsBlocked(ctx context.Context, key string) (bool, error) {
	val, err := r.client.Get(ctx, key+":blocked").Result()
	if err == redis.Nil {
		return false, nil // Not blocked
	} else if err != nil {
		return false, err // Redis error
	}
	return val == "1", nil
}

// Block marks a key as blocked for a specific duration.
func (r *RedisStorage) Block(ctx context.Context, key string, durationSeconds int) error {
	return r.client.Set(ctx, key+":blocked", "1", time.Duration(durationSeconds)*time.Second).Err()
}
