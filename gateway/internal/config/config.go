package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the gateway
type Config struct {
	GatewayID   string
	GatewayPort int
	HTTPPort    int
	RedisURL    string
	NATSURL     string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		GatewayID:   getEnv("GATEWAY_ID", "node-01"),
		GatewayPort: getEnvAsInt("GATEWAY_PORT", 8080),
		HTTPPort:    getEnvAsInt("HTTP_PORT", 8081),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		NATSURL:     getEnv("NATS_URL", "nats://localhost:4222"),
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
