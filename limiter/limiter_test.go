package limiter_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JRRGomes/rate-limiter/config"
	"github.com/JRRGomes/rate-limiter/limiter"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func setupRateLimiter() (*limiter.RateLimiter, *redis.Client) {
	if err := godotenv.Load("../.env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}

	redisStorage := limiter.NewRedisLimiterStorage(client)
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	return limiter.NewRateLimiter(redisStorage, cfg), client
}

func cleanupRedis(client *redis.Client) {
	client.FlushDB(context.Background())
}

func TestRateLimitByIP(t *testing.T) {
	rateLimiter, client := setupRateLimiter()
	defer cleanupRedis(client)

	ctx := context.Background()
	ipKey := "ip:192.168.1.1"

	// Reset any existing keys
	redisClient := rateLimiter.GetStorage().(*limiter.RedisLimiterStorage).GetClient()
	redisClient.Del(ctx, ipKey)
	redisClient.Del(ctx, "blocked:"+ipKey)

	// Make requests until we hit the limit
	var i int
	var allowed bool
	var err error
	for i = 0; i < 30; i++ {
		allowed, err = rateLimiter.Allow(ctx, ipKey, "")
		assert.NoError(t, err)
		if !allowed {
			break
		}
	}

	// Verify that we got the expected number of allowed requests
	assert.Equal(t, 20, i, "Should allow exactly 20 requests before blocking")
	assert.False(t, allowed, "Request after limit should be blocked")

	// Wait for the block to expire (15 seconds as per .env)
	fmt.Println("Waiting for block to expire...")
	time.Sleep(16 * time.Second)

	// Clear the key and block key
	redisClient.Del(ctx, ipKey)
	redisClient.Del(ctx, "blocked:"+ipKey)

	// After waiting and clearing keys, the IP should be allowed again
	allowed, err = rateLimiter.Allow(ctx, ipKey, "")
	fmt.Printf("After waiting: Allowed = %v, Error = %v\n", allowed, err)
	assert.NoError(t, err)
	assert.True(t, allowed, "Request should be allowed after block expires and keys are cleared")
}

func TestRateLimitByTokenTypes(t *testing.T) {
	rateLimiter, client := setupRateLimiter()
	defer cleanupRedis(client)

	ctx := context.Background()
	redisClient := rateLimiter.GetStorage().(*limiter.RedisLimiterStorage).GetClient()

	// Test public token (25 requests/second with 15-second blocking)
	publicToken := "token:public123"
	redisClient.Del(ctx, publicToken)
	redisClient.Del(ctx, "blocked:"+publicToken)

	var i int
	var allowed bool
	var err error
	for i = 0; i < 30; i++ {
		allowed, err = rateLimiter.Allow(ctx, publicToken, "public")
		assert.NoError(t, err)
		if !allowed {
			break
		}
	}
	assert.Equal(t, 25, i, "Should allow exactly 25 public token requests before blocking")
	assert.False(t, allowed, "Public token should be blocked after 25 requests")

	// Test premium token (30 requests/second with 10-second blocking)
	premiumToken := "token:premium456"
	redisClient.Del(ctx, premiumToken)
	redisClient.Del(ctx, "blocked:"+premiumToken)

	for i = 0; i < 35; i++ {
		allowed, err = rateLimiter.Allow(ctx, premiumToken, "premium")
		assert.NoError(t, err)
		if !allowed {
			break
		}
	}
	assert.Equal(t, 30, i, "Should allow exactly 30 premium token requests before blocking")
	assert.False(t, allowed, "Premium token should be blocked after 30 requests")

	// Test admin token (40 requests/second with 5-second blocking)
	adminToken := "token:admin789"
	redisClient.Del(ctx, adminToken)
	redisClient.Del(ctx, "blocked:"+adminToken)

	for i = 0; i < 45; i++ {
		allowed, err = rateLimiter.Allow(ctx, adminToken, "admin")
		assert.NoError(t, err)
		if !allowed {
			break
		}
	}
	assert.Equal(t, 40, i, "Should allow exactly 40 admin token requests before blocking")
	assert.False(t, allowed, "Admin token should be blocked after 40 requests")

	// Test recovery after block duration for admin (shortest duration)
	time.Sleep(6 * time.Second)
	redisClient.Del(ctx, adminToken)
	redisClient.Del(ctx, "blocked:"+adminToken)
	allowed, err = rateLimiter.Allow(ctx, adminToken, "admin")
	assert.NoError(t, err)
	assert.True(t, allowed, "Admin token should be allowed after block expires and keys are cleared")
}

func TestBlockingAndRecovery(t *testing.T) {
	rateLimiter, client := setupRateLimiter()
	defer cleanupRedis(client)

	ip := "192.168.1.2"
	ipKey := "ip:" + ip
	ctx := context.Background()
	redisClient := rateLimiter.GetStorage().(*limiter.RedisLimiterStorage).GetClient()

	// Clear any existing keys
	redisClient.Del(ctx, ipKey)
	redisClient.Del(ctx, "blocked:"+ipKey)

	// Make requests until limit
	var i int
	var allowed bool
	var err error
	for i = 0; i < 25; i++ {
		allowed, err = rateLimiter.Allow(ctx, ipKey, "")
		assert.NoError(t, err)
		if !allowed {
			break
		}
	}
	assert.Equal(t, 20, i, "Should allow exactly 20 IP requests before blocking")

	// Verify blocked
	allowed, err = rateLimiter.Allow(ctx, ipKey, "")
	assert.NoError(t, err)
	assert.False(t, allowed, "Request should be blocked after limit")

	time.Sleep(16 * time.Second)
	redisClient.Del(ctx, ipKey)
	redisClient.Del(ctx, "blocked:"+ipKey)

	allowed, err = rateLimiter.Allow(ctx, ipKey, "")
	assert.NoError(t, err)
	assert.True(t, allowed, "Request should be allowed after block duration expires and keys are cleared")
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

	ctx := context.Background()
	redisClient := rateLimiter.GetStorage().(*limiter.RedisLimiterStorage).GetClient()
	redisClient.FlushAll(ctx)

	tests := []struct {
		name          string
		tokenType     string
		expectedLimit int
		blockDuration int
	}{
		{
			name:          "Public Token",
			tokenType:     "public",
			expectedLimit: 25,
			blockDuration: 15,
		},
		{
			name:          "Premium Token",
			tokenType:     "premium",
			expectedLimit: 30,
			blockDuration: 10,
		},
		{
			name:          "Admin Token",
			tokenType:     "admin",
			expectedLimit: 40,
			blockDuration: 5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear any existing keys
			tokenKey := fmt.Sprintf("token:%s-token", tc.tokenType)
			blockKey := "blocked:" + tokenKey
			redisClient.Del(ctx, tokenKey)
			redisClient.Del(ctx, blockKey)

			// Count successful requests
			successCount := 0
			apiKey := fmt.Sprintf("%s-token", tc.tokenType)

			// Make requests until we hit the limit
			for i := 0; i < tc.expectedLimit+5; i++ {
				req, err := http.NewRequest("GET", server.URL, nil)
				assert.NoError(t, err)

				// Add both token and token type to headers
				req.Header.Set("API_KEY", apiKey)
				req.Header.Set("TOKEN_TYPE", tc.tokenType)

				resp, err := http.DefaultClient.Do(req)
				assert.NoError(t, err)

				if resp.StatusCode == http.StatusOK {
					successCount++
				} else {
					resp.Body.Close()
					break
				}
				resp.Body.Close()
			}

			// Verify we got the expected number of successful requests
			assert.Equal(t, tc.expectedLimit, successCount,
				"Should allow exactly %d requests for %s token type",
				tc.expectedLimit, tc.tokenType)

			// Verify next request is blocked
			req, err := http.NewRequest("GET", server.URL, nil)
			assert.NoError(t, err)
			req.Header.Set("API_KEY", apiKey)
			req.Header.Set("TOKEN_TYPE", tc.tokenType)

			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode,
				"%s token should be blocked after limit", tc.name)
			resp.Body.Close()

			// Wait for block to expire
			time.Sleep(time.Duration(tc.blockDuration+1) * time.Second)

			// Clear the keys
			redisClient.Del(ctx, tokenKey)
			redisClient.Del(ctx, blockKey)

			// Try again after block expires
			req, err = http.NewRequest("GET", server.URL, nil)
			assert.NoError(t, err)
			req.Header.Set("API_KEY", apiKey)
			req.Header.Set("TOKEN_TYPE", tc.tokenType)

			resp, err = http.DefaultClient.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode,
				"%s token should be allowed after block expires", tc.name)
			resp.Body.Close()
		})
	}
}
