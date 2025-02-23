package main

import (
	"log"
	"net/http"
	"os"

	"github.com/JRRGomes/rate-limiter/config"
	"github.com/JRRGomes/rate-limiter/limiter"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		if err := godotenv.Load("../.env"); err != nil {
			log.Println("No .env file found, using environment variables")
		}
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	redisHost := cfg.RedisHost
	if redisHost == "" {
		redisHost = os.Getenv("REDIS_HOST")
		if redisHost == "" {
			redisHost = "localhost"
		}
	}

	redisPort := cfg.RedisPort
	if redisPort == "" {
		redisPort = os.Getenv("REDIS_PORT")
		if redisPort == "" {
			redisPort = "6379"
		}
	}

	// Initialize Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost + ":" + cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	// Initialize RedisStorage and RateLimiter
	storage := limiter.NewRedisLimiterStorage(client)
	rateLimiter := limiter.NewRateLimiter(storage, cfg)

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
