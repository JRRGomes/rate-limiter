package config

import (
	"os"
	"strconv"
)

type RateLimitConfig struct {
	Limit         int
	BlockDuration int
}

type Config struct {
	RateLimitIP        int
	BlockDurationIP    int
	RateLimitToken     int
	BlockDurationToken int
	RedisHost          string
	RedisPort          string
	RedisPassword      string
	TokenLimits        map[string]RateLimitConfig
}

func LoadConfig() (*Config, error) {
	rateLimitIP, _ := strconv.Atoi(getEnv("RATE_LIMIT_IP", "10"))
	blockDurationIP, _ := strconv.Atoi(getEnv("BLOCK_DURATION_IP", "60"))
	rateLimitToken, _ := strconv.Atoi(getEnv("RATE_LIMIT_TOKEN", "11"))
	blockDurationToken, _ := strconv.Atoi(getEnv("BLOCK_DURATION_TOKEN", "60"))

	tokenLimits := map[string]RateLimitConfig{
		"public": {
			Limit:         getEnvInt("RATE_LIMIT_PUBLIC", 10),
			BlockDuration: getEnvInt("BLOCK_DURATION_PUBLIC", 60),
		},
		"premium": {
			Limit:         getEnvInt("RATE_LIMIT_PREMIUM", 100),
			BlockDuration: getEnvInt("BLOCK_DURATION_PREMIUM", 30),
		},
		"admin": {
			Limit:         getEnvInt("RATE_LIMIT_ADMIN", 1000),
			BlockDuration: getEnvInt("BLOCK_DURATION_ADMIN", 10),
		},
	}

	return &Config{
		RateLimitIP:        rateLimitIP,
		BlockDurationIP:    blockDurationIP,
		RateLimitToken:     rateLimitToken,
		BlockDurationToken: blockDurationToken,
		RedisHost:          getEnv("REDIS_HOST", "localhost"),
		RedisPort:          getEnv("REDIS_PORT", "6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		TokenLimits:        tokenLimits,
	}, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
