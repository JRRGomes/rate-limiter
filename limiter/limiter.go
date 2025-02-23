package limiter

import (
	"context"
	"time"

	"github.com/JRRGomes/rate-limiter/config"
	"github.com/go-redis/redis/v8"
)

type RateLimiter struct {
	storage            LimiterStorage
	rateLimitIP        int
	blockDurationIP    int
	rateLimitToken     int
	blockDurationToken int
	tokenLimits        map[string]config.RateLimitConfig
}

type RedisLimiterStorage struct {
	client *redis.Client
}

func (r *RedisLimiterStorage) GetClient() *redis.Client {
	return r.client
}

func NewRateLimiter(storage LimiterStorage, cfg *config.Config) *RateLimiter {
	return &RateLimiter{
		storage:            storage,
		rateLimitIP:        cfg.RateLimitIP,
		blockDurationIP:    cfg.BlockDurationIP,
		rateLimitToken:     cfg.RateLimitToken,
		blockDurationToken: cfg.BlockDurationToken,
		tokenLimits:        cfg.TokenLimits,
	}
}

func NewRedisLimiterStorage(client *redis.Client) *RedisLimiterStorage {
	return &RedisLimiterStorage{client: client}
}

func (r *RateLimiter) Allow(ctx context.Context, key string, tokenType string) (bool, error) {
	blocked, err := r.storage.IsBlocked(ctx, key)
	if err != nil {
		return false, err
	}
	if blocked {
		return false, nil
	}

	var limit int
	var blockDuration int

	if tokenType == "" {
		limit = r.rateLimitIP
		blockDuration = r.blockDurationIP
	} else {
		if config, exists := r.tokenLimits[tokenType]; exists {
			limit = config.Limit
			blockDuration = config.BlockDuration
		} else {
			limit = r.rateLimitToken
			blockDuration = r.blockDurationToken
		}
	}

	count, err := r.storage.Increment(ctx, key, 1)
	if err != nil {
		return false, err
	}

	if int(count) > limit {
		_ = r.storage.Block(ctx, key, blockDuration)
		_ = r.storage.Reset(ctx, key)
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
	count, err := r.client.IncrBy(ctx, key, int64(value)).Result()
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *RedisLimiterStorage) Block(ctx context.Context, key string, duration int) error {
	blockKey := "blocked:" + key
	_, err := r.client.Set(ctx, blockKey, "1", time.Duration(duration)*time.Second).Result()
	return err
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
