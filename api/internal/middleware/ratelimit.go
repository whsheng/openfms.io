package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitAlgorithm 限流算法类型
type RateLimitAlgorithm string

const (
	// TokenBucket 令牌桶算法
	TokenBucket RateLimitAlgorithm = "token_bucket"
	// LeakyBucket 漏桶算法
	LeakyBucket RateLimitAlgorithm = "leaky_bucket"
	// FixedWindow 固定窗口算法
	FixedWindow RateLimitAlgorithm = "fixed_window"
)

// RateLimitType 限流类型
type RateLimitType string

const (
	// RateLimitByIP 基于IP限流
	RateLimitByIP RateLimitType = "ip"
	// RateLimitByUser 基于用户限流
	RateLimitByUser RateLimitType = "user"
	// RateLimitByEndpoint 基于接口限流
	RateLimitByEndpoint RateLimitType = "endpoint"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// 请求限制数
	Limit int
	// 窗口大小（秒）
	Window int
	// 限流算法
	Algorithm RateLimitAlgorithm
	// 限流类型
	Type RateLimitType
	// 自定义Key生成函数（可选）
	KeyFunc func(*gin.Context) string
}

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(ctx context.Context, key string, config *RateLimitConfig) (*RateLimitResult, error)
}

// RateLimitResult 限流结果
type RateLimitResult struct {
	// 是否允许通过
	Allowed bool
	// 剩余请求数
	Remaining int
	// 重置时间（Unix时间戳）
	ResetAt int64
	// 总限制数
	Limit int
}

// RedisRateLimiter 基于Redis的限流器
type RedisRateLimiter struct {
	redis *redis.Client
}

// NewRedisRateLimiter 创建Redis限流器
func NewRedisRateLimiter(redis *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{redis: redis}
}

// Allow 检查是否允许请求通过
func (r *RedisRateLimiter) Allow(ctx context.Context, key string, config *RateLimitConfig) (*RateLimitResult, error) {
	switch config.Algorithm {
	case TokenBucket:
		return r.tokenBucket(ctx, key, config)
	case LeakyBucket:
		return r.leakyBucket(ctx, key, config)
	case FixedWindow:
		return r.fixedWindow(ctx, key, config)
	default:
		return r.tokenBucket(ctx, key, config)
	}
}

// tokenBucket 令牌桶算法实现
func (r *RedisRateLimiter) tokenBucket(ctx context.Context, key string, config *RateLimitConfig) (*RateLimitResult, error) {
	now := time.Now().Unix()
	bucketKey := fmt.Sprintf("ratelimit:token:%s", key)
	
	// Lua脚本实现令牌桶
	script := `
		local bucket = redis.call('HMGET', KEYS[1], 'tokens', 'last_update')
		local capacity = tonumber(ARGV[1])
		local rate = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local requested = tonumber(ARGV[4])
		
		local tokens = tonumber(bucket[1]) or capacity
		local last_update = tonumber(bucket[2]) or now
		
		-- 计算新增令牌
		local elapsed = now - last_update
		local new_tokens = math.min(capacity, tokens + elapsed * rate)
		
		-- 检查是否有足够令牌
		local allowed = new_tokens >= requested
		local remaining = 0
		
		if allowed then
			new_tokens = new_tokens - requested
			remaining = math.floor(new_tokens)
		end
		
		-- 更新桶状态
		redis.call('HMSET', KEYS[1], 'tokens', new_tokens, 'last_update', now)
		redis.call('EXPIRE', KEYS[1], math.ceil(capacity / rate) + 1)
		
		return {allowed and 1 or 0, remaining, capacity}
	`
	
	// 每秒产生令牌数
	ratePerSecond := float64(config.Limit) / float64(config.Window)
	
	result, err := r.redis.Eval(ctx, script, []string{bucketKey}, 
		config.Limit,       // 桶容量
		ratePerSecond,      // 每秒产生令牌数
		now,                // 当前时间
		1,                  // 请求1个令牌
	).Result()
	
	if err != nil {
		return nil, err
	}
	
	values := result.([]interface{})
	allowed := values[0].(int64) == 1
	remaining := int(values[1].(int64))
	limit := int(values[2].(int64))
	
	// 计算重置时间
	resetAt := now + int64(config.Window)
	
	return &RateLimitResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   resetAt,
		Limit:     limit,
	}, nil
}

// leakyBucket 漏桶算法实现
func (r *RedisRateLimiter) leakyBucket(ctx context.Context, key string, config *RateLimitConfig) (*RateLimitResult, error) {
	now := time.Now().UnixMilli()
	bucketKey := fmt.Sprintf("ratelimit:leaky:%s", key)
	
	// Lua脚本实现漏桶
	script := `
		local queue = redis.call('LRANGE', KEYS[1], 0, -1)
		local capacity = tonumber(ARGV[1])
		local leak_rate = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local window = tonumber(ARGV[4]) * 1000
		
		-- 计算应该漏出的请求数
		local last_leak = tonumber(redis.call('GET', KEYS[1] .. ':last_leak') or now)
		local elapsed = now - last_leak
		local leaked = math.floor(elapsed * leak_rate / 1000)
		
		-- 移除已漏出的请求
		if leaked > 0 then
			redis.call('LTRIM', KEYS[1], leaked, -1)
			redis.call('SET', KEYS[1] .. ':last_leak', now)
		end
		
		-- 获取当前队列长度
		local current_size = redis.call('LLEN', KEYS[1])
		local allowed = current_size < capacity
		local remaining = capacity - current_size - 1
		
		if allowed then
			redis.call('RPUSH', KEYS[1], now)
			remaining = capacity - current_size - 1
		end
		
		redis.call('EXPIRE', KEYS[1], math.ceil(window / 1000) + 1)
		redis.call('EXPIRE', KEYS[1] .. ':last_leak', math.ceil(window / 1000) + 1)
		
		return {allowed and 1 or 0, remaining, capacity}
	`
	
	// 每秒处理请求数（漏出速率）
	leakRatePerSecond := float64(config.Limit) * 1000 / float64(config.Window)
	
	result, err := r.redis.Eval(ctx, script, []string{bucketKey},
		config.Limit,       // 桶容量
		leakRatePerSecond,  // 漏出速率（每毫秒）
		now,                // 当前时间（毫秒）
		config.Window,      // 窗口大小（秒）
	).Result()
	
	if err != nil {
		return nil, err
	}
	
	values := result.([]interface{})
	allowed := values[0].(int64) == 1
	remaining := int(values[1].(int64))
	limit := int(values[2].(int64))
	
	// 计算重置时间
	resetAt := now/1000 + int64(config.Window)
	
	return &RateLimitResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   resetAt,
		Limit:     limit,
	}, nil
}

// fixedWindow 固定窗口算法实现
func (r *RedisRateLimiter) fixedWindow(ctx context.Context, key string, config *RateLimitConfig) (*RateLimitResult, error) {
	now := time.Now().Unix()
	// 计算当前窗口
	window := now / int64(config.Window)
	windowKey := fmt.Sprintf("ratelimit:fixed:%s:%d", key, window)
	
	// Lua脚本实现固定窗口
	script := `
		local current = tonumber(redis.call('GET', KEYS[1]) or 0)
		local limit = tonumber(ARGV[1])
		local ttl = tonumber(ARGV[2])
		
		local allowed = current < limit
		local remaining = limit - current - 1
		
		if allowed then
			redis.call('INCR', KEYS[1])
			if current == 0 then
				redis.call('EXPIRE', KEYS[1], ttl)
			end
		else
			remaining = -1
		end
		
		return {allowed and 1 or 0, remaining, limit}
	`
	
	result, err := r.redis.Eval(ctx, script, []string{windowKey},
		config.Limit,       // 限制数
		config.Window + 1,  // TTL（稍微多一点时间）
	).Result()
	
	if err != nil {
		return nil, err
	}
	
	values := result.([]interface{})
	allowed := values[0].(int64) == 1
	remaining := int(values[1].(int64))
	limit := int(values[2].(int64))
	
	// 计算重置时间（下一个窗口开始）
	resetAt := (window + 1) * int64(config.Window)
	
	return &RateLimitResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   resetAt,
		Limit:     limit,
	}, nil
}

// RateLimitMiddleware 限流中间件
type RateLimitMiddleware struct {
	limiter RateLimiter
	config  *RateLimitConfig
}

// NewRateLimitMiddleware 创建限流中间件
func NewRateLimitMiddleware(limiter RateLimiter, config *RateLimitConfig) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
		config:  config,
	}
}

// Middleware 返回Gin中间件函数
func (m *RateLimitMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成限流Key
		key := m.generateKey(c)
		
		// 检查限流
		result, err := m.limiter.Allow(c.Request.Context(), key, m.config)
		if err != nil {
			// Redis错误时，允许请求通过（降级策略）
			c.Next()
			return
		}
		
		// 设置响应头
		c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt, 10))
		
		if !result.Allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
				"retry_after": result.ResetAt - time.Now().Unix(),
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// generateKey 生成限流Key
func (m *RateLimitMiddleware) generateKey(c *gin.Context) string {
	// 如果配置了自定义Key函数，使用自定义函数
	if m.config.KeyFunc != nil {
		return m.config.KeyFunc(c)
	}
	
	switch m.config.Type {
	case RateLimitByIP:
		return m.getClientIP(c)
	case RateLimitByUser:
		userID, exists := c.Get("user_id")
		if !exists {
			// 未登录用户使用IP
			return "ip:" + m.getClientIP(c)
		}
		return fmt.Sprintf("user:%v", userID)
	case RateLimitByEndpoint:
		return fmt.Sprintf("endpoint:%s:%s", c.Request.Method, c.Request.URL.Path)
	default:
		return m.getClientIP(c)
	}
}

// getClientIP 获取客户端IP
func (m *RateLimitMiddleware) getClientIP(c *gin.Context) string {
	// 优先从X-Forwarded-For获取
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// 从X-Real-IP获取
	xri := c.GetHeader("X-Real-Ip")
	if xri != "" {
		return xri
	}
	
	// 从RemoteAddr获取
	ip := c.ClientIP()
	if ip != "" {
		return ip
	}
	
	return "unknown"
}

// RateLimitGroup 限流组配置
type RateLimitGroup struct {
	limiter         RateLimiter
	defaultConfig   *RateLimitConfig
	specificConfigs map[string]*RateLimitConfig
}

// NewRateLimitGroup 创建限流组
func NewRateLimitGroup(limiter RateLimiter, defaultConfig *RateLimitConfig) *RateLimitGroup {
	return &RateLimitGroup{
		limiter:         limiter,
		defaultConfig:   defaultConfig,
		specificConfigs: make(map[string]*RateLimitConfig),
	}
}

// AddSpecificConfig 添加特定路径配置
func (g *RateLimitGroup) AddSpecificConfig(path string, config *RateLimitConfig) {
	g.specificConfigs[path] = config
}

// Middleware 返回Gin中间件函数（支持不同路径不同配置）
func (g *RateLimitGroup) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 查找特定路径配置
		config := g.defaultConfig
		path := c.Request.URL.Path
		if specificConfig, exists := g.specificConfigs[path]; exists {
			config = specificConfig
		}
		
		// 生成限流Key
		key := g.generateKey(c, config)
		
		// 检查限流
		result, err := g.limiter.Allow(c.Request.Context(), key, config)
		if err != nil {
			// Redis错误时，允许请求通过（降级策略）
			c.Next()
			return
		}
		
		// 设置响应头
		c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt, 10))
		
		if !result.Allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": result.ResetAt - time.Now().Unix(),
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// generateKey 生成限流Key
func (g *RateLimitGroup) generateKey(c *gin.Context, config *RateLimitConfig) string {
	// 如果配置了自定义Key函数，使用自定义函数
	if config.KeyFunc != nil {
		return config.KeyFunc(c)
	}
	
	switch config.Type {
	case RateLimitByIP:
		return g.getClientIP(c)
	case RateLimitByUser:
		userID, exists := c.Get("user_id")
		if !exists {
			// 未登录用户使用IP
			return "ip:" + g.getClientIP(c)
		}
		return fmt.Sprintf("user:%v", userID)
	case RateLimitByEndpoint:
		return fmt.Sprintf("endpoint:%s:%s", c.Request.Method, c.Request.URL.Path)
	default:
		return g.getClientIP(c)
	}
}

// getClientIP 获取客户端IP
func (g *RateLimitGroup) getClientIP(c *gin.Context) string {
	// 优先从X-Forwarded-For获取
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// 从X-Real-IP获取
	xri := c.GetHeader("X-Real-Ip")
	if xri != "" {
		return xri
	}
	
	// 从RemoteAddr获取
	ip := c.ClientIP()
	if ip != "" {
		return ip
	}
	
	return "unknown"
}

// SkipSuccessfulRequests 跳过成功请求的限流计数（用于登录等场景）
func SkipSuccessfulRequests(limiter RateLimiter, config *RateLimitConfig) gin.HandlerFunc {
	middleware := NewRateLimitMiddleware(limiter, config)
	
	return func(c *gin.Context) {
		// 先生成Key
		key := middleware.generateKey(c)
		
		// 检查限流但不计数
		result, err := limiter.Allow(c.Request.Context(), key, &RateLimitConfig{
			Limit:     config.Limit,
			Window:    config.Window,
			Algorithm: config.Algorithm,
			Type:      config.Type,
			KeyFunc: func(c *gin.Context) string {
				return key + ":check"
			},
		})
		
		if err != nil {
			c.Next()
			return
		}
		
		// 设置响应头
		c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt, 10))
		
		if !result.Allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": result.ResetAt - time.Now().Unix(),
			})
			c.Abort()
			return
		}
		
		c.Next()
		
		// 只有请求失败时才计数（防止暴力破解）
		if c.Writer.Status() >= 400 {
			// 消耗一个令牌
			limiter.Allow(c.Request.Context(), key, config)
		}
	}
}
