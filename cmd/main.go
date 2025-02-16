package main

import (
	"log"
	"net/http"

	"github.com/JRRGomes/rate-limiter/config"
	"github.com/JRRGomes/rate-limiter/limiter"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load("../.env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost + ":" + cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	// Initialize RedisStorage and RateLimiter
	storage := limiter.NewRedisLimiterStorage(client)
	rateLimiter := limiter.NewRateLimiter(storage, cfg.RateLimitIP, cfg.RateLimitToken, cfg.BlockDuration)

	// Create HTTP server with rate limiting middleware
	mux := http.NewServeMux()
	mux.Handle("/", limiter.Middleware(rateLimiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to the rate-limited server!"))
	})))

	log.Println("Server running on port 8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
