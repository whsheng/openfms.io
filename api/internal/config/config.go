package config

import (
	"os"
	"strconv"
	"time"

	"openfms/api/internal/middleware"
)

// RateLimitRule 限流规则配置
type RateLimitRule struct {
	// 路径匹配（支持前缀匹配）
	Path string
	// 请求限制数
	Limit int
	// 窗口大小
	Window time.Duration
	// 限流算法
	Algorithm middleware.RateLimitAlgorithm
	// 限流类型
	Type middleware.RateLimitType
}

// RateLimitConfig 限流总配置
type RateLimitConfig struct {
	// 是否启用限流
	Enabled bool
	// 默认限流配置
	DefaultRule RateLimitRule
	// 特定路径规则
	SpecificRules []RateLimitRule
}

// Config holds all configuration for the API server
type Config struct {
	APIPort     int
	DatabaseURL string
	RedisURL    string
	NATSURL     string
	JWTSecret   string
	// 限流配置
	RateLimit RateLimitConfig
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		APIPort:     getEnvAsInt("API_PORT", 3000),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://openfms:openfms_secret@localhost:5432/openfms?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		NATSURL:     getEnv("NATS_URL", "nats://localhost:4222"),
		JWTSecret:   getEnv("JWT_SECRET", "openfms-secret-key-change-in-production"),
		RateLimit:   loadRateLimitConfig(),
	}
}

// loadRateLimitConfig 加载限流配置
func loadRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled: getEnvAsBool("RATE_LIMIT_ENABLED", true),
		DefaultRule: RateLimitRule{
			Path:      "*",
			Limit:     getEnvAsInt("RATE_LIMIT_DEFAULT_LIMIT", 100),
			Window:    time.Duration(getEnvAsInt("RATE_LIMIT_DEFAULT_WINDOW", 60)) * time.Second,
			Algorithm: middleware.RateLimitAlgorithm(getEnv("RATE_LIMIT_DEFAULT_ALGORITHM", "token_bucket")),
			Type:      middleware.RateLimitType(getEnv("RATE_LIMIT_DEFAULT_TYPE", "ip")),
		},
		SpecificRules: []RateLimitRule{
			// 登录接口限流：5次/分钟，基于IP
			{
				Path:      "/api/v1/auth/login",
				Limit:     getEnvAsInt("RATE_LIMIT_LOGIN_LIMIT", 5),
				Window:    time.Duration(getEnvAsInt("RATE_LIMIT_LOGIN_WINDOW", 60)) * time.Second,
				Algorithm: middleware.RateLimitAlgorithm(getEnv("RATE_LIMIT_LOGIN_ALGORITHM", "fixed_window")),
				Type:      middleware.RateLimitType(getEnv("RATE_LIMIT_LOGIN_TYPE", "ip")),
			},
			// 指令下发限流：10次/分钟，基于用户
			{
				Path:      "/api/v1/devices/",
				Limit:     getEnvAsInt("RATE_LIMIT_COMMAND_LIMIT", 10),
				Window:    time.Duration(getEnvAsInt("RATE_LIMIT_COMMAND_WINDOW", 60)) * time.Second,
				Algorithm: middleware.RateLimitAlgorithm(getEnv("RATE_LIMIT_COMMAND_ALGORITHM", "token_bucket")),
				Type:      middleware.RateLimitType(getEnv("RATE_LIMIT_COMMAND_TYPE", "user")),
			},
			// Webhook接口限流：60次/分钟，基于IP
			{
				Path:      "/api/v1/webhooks",
				Limit:     getEnvAsInt("RATE_LIMIT_WEBHOOK_LIMIT", 60),
				Window:    time.Duration(getEnvAsInt("RATE_LIMIT_WEBHOOK_WINDOW", 60)) * time.Second,
				Algorithm: middleware.RateLimitAlgorithm(getEnv("RATE_LIMIT_WEBHOOK_ALGORITHM", "token_bucket")),
				Type:      middleware.RateLimitType(getEnv("RATE_LIMIT_WEBHOOK_TYPE", "ip")),
			},
		},
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

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// GetRateLimitRuleForPath 获取指定路径的限流规则
func (c *Config) GetRateLimitRuleForPath(path string) RateLimitRule {
	// 检查特定路径规则
	for _, rule := range c.RateLimit.SpecificRules {
		if len(rule.Path) > 0 && len(path) >= len(rule.Path) && path[:len(rule.Path)] == rule.Path {
			return rule
		}
	}
	// 返回默认规则
	return c.RateLimit.DefaultRule
}

// ToMiddlewareConfig 转换为中间件配置
func (r *RateLimitRule) ToMiddlewareConfig() *middleware.RateLimitConfig {
	return &middleware.RateLimitConfig{
		Limit:     r.Limit,
		Window:    int(r.Window.Seconds()),
		Algorithm: r.Algorithm,
		Type:      r.Type,
	}
}
