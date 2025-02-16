package config

import (
	"os"
	"strconv"
)

type Config struct {
	RateLimitIP    int
	RateLimitToken int
	BlockDuration  int
	RedisHost      string
	RedisPort      string
	RedisPassword  string
}

func LoadConfig() (*Config, error) {
	rateLimitIP, _ := strconv.Atoi(getEnv("RATE_LIMIT_IP", "10"))
	rateLimitToken, _ := strconv.Atoi(getEnv("RATE_LIMIT_TOKEN", "11"))
	blockDuration, _ := strconv.Atoi(getEnv("BLOCK_DURATION", "60"))

	return &Config{
		RateLimitIP:    rateLimitIP,
		RateLimitToken: rateLimitToken,
		BlockDuration:  blockDuration,
		RedisHost:      getEnv("REDIS_HOST", "localhost"),
		RedisPort:      getEnv("REDIS_PORT", "6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
	}, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
