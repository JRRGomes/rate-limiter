package limiter

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RateLimiter struct {
	storage        LimiterStorage
	rateLimitIP    int
	rateLimitToken int
	blockDuration  int
}

type RedisLimiterStorage struct {
	client *redis.Client
}

func (r *RedisLimiterStorage) GetClient() *redis.Client {
	return r.client
}

func NewRateLimiter(storage LimiterStorage, rateLimitIP, rateLimitToken, blockDuration int) *RateLimiter {
	return &RateLimiter{
		storage:        storage,
		rateLimitIP:    rateLimitIP,
		rateLimitToken: rateLimitToken,
		blockDuration:  blockDuration,
	}
}

func NewRedisLimiterStorage(client *redis.Client) *RedisLimiterStorage {
	return &RedisLimiterStorage{client: client}
}

func (r *RateLimiter) Allow(ctx context.Context, key string, limit int) (bool, error) {
	// Check if the key is blocked
	blocked, err := r.storage.IsBlocked(ctx, key)
	if err != nil {
		return false, err
	}
	if blocked {
		return false, nil
	}

	// Increment and check if the key exceeds the limit
	count, err := r.storage.Increment(ctx, key, 1)
	if err != nil {
		return false, err
	}

	// If the count exceeds the limit, block the key
	if int(count) > limit {
		err := r.storage.Block(ctx, key, r.blockDuration)
		if err != nil {
			return false, err
		}
		// Reset the request count after blocking
		err = r.storage.Reset(ctx, key)
		if err != nil {
			return false, err
		}
		return false, nil
	}

	return true, nil
}

func (r *RedisLimiterStorage) IsBlocked(ctx context.Context, key string) (bool, error) {
	blockKey := "blocked:" + key
	blocked, err := r.client.Get(ctx, blockKey).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	return blocked != "", nil
}

func (r *RedisLimiterStorage) Increment(ctx context.Context, key string, value int) (int64, error) {
	// Increment the value for the key in Redis
	count, err := r.client.IncrBy(ctx, key, int64(value)).Result()
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *RedisLimiterStorage) Block(ctx context.Context, key string, duration int) error {
	blockKey := "blocked:" + key
	_, err := r.client.Set(ctx, blockKey, "1", time.Duration(duration)*time.Second).Result()
	if err != nil {
		return err
	}
	return nil
}

func (r *RateLimiter) GetStorage() LimiterStorage {
	return r.storage
}

func (r *RedisLimiterStorage) Reset(ctx context.Context, key string) error {
	_, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return err
	}
	return nil
}
