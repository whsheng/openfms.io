package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the API server
type Config struct {
	APIPort     int
	DatabaseURL string
	RedisURL    string
	NATSURL     string
	JWTSecret   string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		APIPort:     getEnvAsInt("API_PORT", 3000),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://openfms:openfms_secret@localhost:5432/openfms?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		NATSURL:     getEnv("NATS_URL", "nats://localhost:4222"),
		JWTSecret:   getEnv("JWT_SECRET", "openfms-secret-key-change-in-production"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
