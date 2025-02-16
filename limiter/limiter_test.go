package limiter_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JRRGomes/rate-limiter/limiter"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func setupRateLimiter() (*limiter.RateLimiter, *redis.Client) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}

	redisStorage := limiter.NewRedisLimiterStorage(client)
	return limiter.NewRateLimiter(redisStorage, 10, 11, 2), client
}

func cleanupRedis(client *redis.Client) {
	client.FlushDB(context.Background())
}

func TestRateLimitByIP(t *testing.T) {
	rateLimiter, client := setupRateLimiter()
	defer cleanupRedis(client)

	ctx := context.Background()
	ipKey := "ip:192.168.1.1"

	// The first request should be allowed
	allowed, err := rateLimiter.Allow(ctx, ipKey, 10)
	assert.NoError(t, err)
	assert.True(t, allowed, "First request should be allowed")

	// Simulate 9 more requests (total of 10 requests)
	for i := 0; i < 9; i++ {
		allowed, err := rateLimiter.Allow(ctx, ipKey, 10)
		assert.NoError(t, err)
		assert.True(t, allowed, fmt.Sprintf("Request %d should be allowed", i+2))
	}

	// The 11th request should be blocked
	allowed, err = rateLimiter.Allow(ctx, ipKey, 10)
	assert.NoError(t, err)
	assert.False(t, allowed, "11th request should be blocked")

	// Wait for the block to expire
	fmt.Println("Waiting for block to expire...")
	time.Sleep(5 * time.Second)

	// Verify that the block key has expired
	blockKey := "blocked:" + ipKey
	ttl, err := rateLimiter.GetStorage().(*limiter.RedisLimiterStorage).GetClient().TTL(ctx, blockKey).Result()
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(-2), ttl, "Block key should have expired")

	rateLimiter.GetStorage().(*limiter.RedisLimiterStorage).GetClient().Del(ctx, ipKey)

	// After waiting, the IP should be allowed again
	allowed, err = rateLimiter.Allow(ctx, ipKey, 10)
	fmt.Printf("After waiting: Allowed = %v, Error = %v\n", allowed, err)
	assert.NoError(t, err)
	assert.True(t, allowed, "Request should be allowed after block expires")
}

func TestRateLimitByToken(t *testing.T) {
	rateLimiter, client := setupRateLimiter()
	defer cleanupRedis(client)

	token := "abc123"
	for i := 0; i < 11; i++ {
		allowed, err := rateLimiter.Allow(context.Background(), "token:"+token, 11)
		assert.NoError(t, err)
		assert.True(t, allowed, "Request should be allowed")
	}

	allowed, err := rateLimiter.Allow(context.Background(), "token:"+token, 11)
	assert.NoError(t, err)
	assert.False(t, allowed, "Request should be blocked after limit is exceeded")
}

func TestBlockingAndRecovery(t *testing.T) {
	rateLimiter, client := setupRateLimiter()
	defer cleanupRedis(client)

	ip := "192.168.1.2"
	for i := 0; i < 10; i++ {
		allowed, err := rateLimiter.Allow(context.Background(), "ip:"+ip, 10)
		assert.NoError(t, err)
		assert.True(t, allowed, "Request should be allowed")
	}

	allowed, err := rateLimiter.Allow(context.Background(), "ip:"+ip, 10)
	assert.NoError(t, err)
	assert.False(t, allowed, "Request should be blocked")

	time.Sleep(5 * time.Second)

	allowed, err = rateLimiter.Allow(context.Background(), "ip:"+ip, 10)
	assert.NoError(t, err)
	assert.True(t, allowed, "Request should be allowed after block duration expires")
}

func TestMiddlewareRateLimit(t *testing.T) {
	rateLimiter, client := setupRateLimiter()
	defer cleanupRedis(client)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome!"))
	})

	mux := http.NewServeMux()
	mux.Handle("/", limiter.Middleware(rateLimiter)(handler))

	server := httptest.NewServer(mux)
	defer server.Close()

	token := "abc123"
	for i := 0; i < 10; i++ {
		resp, err := http.Get(server.URL + "?API_KEY=" + token)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Request should succeed")
	}

	resp, err := http.Get(server.URL + "?API_KEY=" + token)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "Request should be blocked after limit")

	time.Sleep(5 * time.Second)

	resp, err = http.Get(server.URL + "?API_KEY=" + token)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Request should succeed after block expires")
}
