package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/security"
	"github.com/redis/go-redis/v9"
)

// RateLimiterType 定义限流算法类型
type RateLimiterType int

const (
	FixedWindow RateLimiterType = iota
	SlidingWindow
	TokenBucket
	LeakyBucket
)

// LimitConfig 限流配置
type LimitConfig struct {
	MaxRequests    int           // 最大请求数
	WindowDuration time.Duration // 时间窗口
	BurstSize      int           // 突发容量
}

// RequestRecord 请求记录
type RequestRecord struct {
	Count     int
	FirstTime time.Time
	LastTime  time.Time
	Tokens    float64
	Mu        sync.Mutex
}

// IPLevelLimit IP级别限流配置
type IPLevelLimit struct {
	Default    *LimitConfig
	Endpoints map[string]*LimitConfig
	Mu         sync.RWMutex
}

// UserLevelLimit 用户级别限流配置
type UserLevelLimit struct {
	Default    *LimitConfig
	Endpoints map[string]*LimitConfig
	Mu         sync.RWMutex
}

// APILevelLimit API级别限流配置
type APILevelLimit struct {
	Default    *LimitConfig
	Endpoints map[string]*LimitConfig
	Mu         sync.RWMutex
}

// RateLimiter 限流器
type RateLimiter struct {
	redis            *redis.Client
	requests         map[string]*RequestRecord
	ipLimits         *IPLevelLimit
	userLimits       *UserLevelLimit
	apiLimits        *APILevelLimit
	globalLimit      *LimitConfig
	limiterType      RateLimiterType
	mu               sync.RWMutex
	cleanupInterval  time.Duration
	lastCleanup      time.Time
	whitelist        map[string]bool
	blacklist        map[string]bool
	enableWhitelist  bool
	enableBlacklist  bool
	tokenRefillRate  float64
}

// RateLimiterConfig 限流器配置
type RateLimiterConfig struct {
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	LimiterType   RateLimiterType
	GlobalLimit   *LimitConfig
	IPLimit       *LimitConfig
	UserLimit     *LimitConfig
	APILimit      *LimitConfig
	TokenRefillRate float64
	EnableWhitelist bool
	EnableBlacklist  bool
	Whitelist     []string
	Blacklist     []string
}

// NewRateLimiter 创建限流器
func NewRateLimiter(config *RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{
		requests:        make(map[string]*RequestRecord),
		limiterType:    config.LimiterType,
		cleanupInterval: 5 * time.Minute,
		lastCleanup:    time.Now(),
		tokenRefillRate: config.TokenRefillRate,
		enableWhitelist: config.EnableWhitelist,
		enableBlacklist: config.EnableBlacklist,
		whitelist:      make(map[string]bool),
		blacklist:      make(map[string]bool),
	}

	for _, ip := range config.Whitelist {
		rl.whitelist[strings.TrimSpace(ip)] = true
	}

	for _, ip := range config.Blacklist {
		rl.blacklist[strings.TrimSpace(ip)] = true
	}

	rl.ipLimits = &IPLevelLimit{
		Default: config.IPLimit,
		Endpoints: make(map[string]*LimitConfig),
	}

	rl.userLimits = &UserLevelLimit{
		Default: config.UserLimit,
		Endpoints: make(map[string]*LimitConfig),
	}

	rl.apiLimits = &APILevelLimit{
		Default: config.APILimit,
		Endpoints: make(map[string]*LimitConfig),
	}

	rl.globalLimit = config.GlobalLimit

	if config.RedisHost != "" {
		rl.redis = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
			Password: config.RedisPassword,
			DB:       config.RedisDB,
		})
	}

	return rl
}

// SetIPEndpointLimit 设置特定IP的限流配置
func (rl *RateLimiter) SetIPEndpointLimit(ip, endpoint string, config *LimitConfig) {
	rl.ipLimits.Mu.Lock()
	defer rl.ipLimits.Mu.Unlock()
	key := fmt.Sprintf("%s:%s", ip, endpoint)
	rl.ipLimits.Endpoints[key] = config
}

// SetUserEndpointLimit 设置特定用户的限流配置
func (rl *RateLimiter) SetUserEndpointLimit(userID, endpoint string, config *LimitConfig) {
	rl.userLimits.Mu.Lock()
	defer rl.userLimits.Mu.Unlock()
	key := fmt.Sprintf("%s:%s", userID, endpoint)
	rl.userLimits.Endpoints[key] = config
}

// SetAPIEndpointLimit 设置特定API的限流配置
func (rl *RateLimiter) SetAPIEndpointLimit(api, endpoint string, config *LimitConfig) {
	rl.apiLimits.Mu.Lock()
	defer rl.apiLimits.Mu.Unlock()
	key := fmt.Sprintf("%s:%s", api, endpoint)
	rl.apiLimits.Endpoints[key] = config
}

// AddToWhitelist 添加IP到白名单
func (rl *RateLimiter) AddToWhitelist(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.whitelist[ip] = true
}

// RemoveFromWhitelist 从白名单移除IP
func (rl *RateLimiter) RemoveFromWhitelist(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.whitelist, ip)
}

// AddToBlacklist 添加IP到黑名单
func (rl *RateLimiter) AddToBlacklist(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.blacklist[ip] = true
}

// RemoveFromBlacklist 从黑名单移除IP
func (rl *RateLimiter) RemoveFromBlacklist(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.blacklist, ip)
}

// IsWhitelisted 检查IP是否在白名单中
func (rl *RateLimiter) IsWhitelisted(ip string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.whitelist[ip]
}

// IsBlacklisted 检查IP是否在黑名单中
func (rl *RateLimiter) IsBlacklisted(ip string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.blacklist[ip]
}

// getIP 从请求中获取真实IP
func (rl *RateLimiter) getIP(c *gin.Context) string {
	realIP := c.GetHeader("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	forwardedFor := c.GetHeader("X-Forwarded-For")
	if forwardedFor != "" {
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	return c.ClientIP()
}

// getUserID 从请求中获取用户ID
func (rl *RateLimiter) getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		return fmt.Sprintf("%v", userID)
	}
	return ""
}

// getAPIKey 从请求中获取API Key
func (rl *RateLimiter) getAPIKey(c *gin.Context) string {
	apiKey := c.GetHeader("X-API-Key")
	if apiKey != "" {
		return apiKey
	}

	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return "bearer"
	}

	return "anonymous"
}

// getLimitConfig 获取限流配置
func (rl *RateLimiter) getLimitConfig(c *gin.Context) *LimitConfig {
	ip := rl.getIP(c)
	userID := rl.getUserID(c)
	apiKey := rl.getAPIKey(c)
	endpoint := c.FullPath()

	// 优先级：API级别 > 用户级别 > IP级别 > 全局限流
	if apiKey != "" && apiKey != "anonymous" {
		rl.apiLimits.Mu.RLock()
		if config, ok := rl.apiLimits.Endpoints[fmt.Sprintf("%s:%s", apiKey, endpoint)]; ok {
			rl.apiLimits.Mu.RUnlock()
			return config
		}
		if config := rl.apiLimits.Default; config != nil {
			rl.apiLimits.Mu.RUnlock()
			return config
		}
		rl.apiLimits.Mu.RUnlock()
	}

	if userID != "" {
		rl.userLimits.Mu.RLock()
		if config, ok := rl.userLimits.Endpoints[fmt.Sprintf("%s:%s", userID, endpoint)]; ok {
			rl.userLimits.Mu.RUnlock()
			return config
		}
		if config := rl.userLimits.Default; config != nil {
			rl.userLimits.Mu.RUnlock()
			return config
		}
		rl.userLimits.Mu.RUnlock()
	}

	rl.ipLimits.Mu.RLock()
	if config, ok := rl.ipLimits.Endpoints[fmt.Sprintf("%s:%s", ip, endpoint)]; ok {
		rl.ipLimits.Mu.RUnlock()
		return config
	}
	if config := rl.ipLimits.Default; config != nil {
		rl.ipLimits.Mu.RUnlock()
		return config
	}
	rl.ipLimits.Mu.RUnlock()

	return rl.globalLimit
}

// generateKey 生成限流键
func (rl *RateLimiter) generateKey(c *gin.Context, limitType string) string {
	ip := rl.getIP(c)
	userID := rl.getUserID(c)
	endpoint := c.FullPath()

	if userID != "" {
		return fmt.Sprintf("ratelimit:%s:user:%s:%s", limitType, userID, endpoint)
	}

	return fmt.Sprintf("ratelimit:%s:ip:%s:%s", limitType, ip, endpoint)
}

// checkFixedWindow 固定窗口限流算法
func (rl *RateLimiter) checkFixedWindow(ctx context.Context, key string, config *LimitConfig) (bool, int, error) {
	if rl.redis != nil {
		redisKey := fmt.Sprintf("ratelimit:fixed:%s", key)
		count, err := rl.redis.Incr(ctx, redisKey).Result()
		if err != nil {
			return true, 0, err
		}

		if count == 1 {
			rl.redis.Expire(ctx, redisKey, config.WindowDuration)
		}

		remaining := config.MaxRequests - int(count)

		if count > int64(config.MaxRequests) {
			return false, int(remaining), nil
		}

		return true, int(remaining), nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	record, exists := rl.requests[key]
	now := time.Now()

	if !exists || now.Sub(record.FirstTime) > config.WindowDuration {
		rl.requests[key] = &RequestRecord{
			Count:     1,
			FirstTime: now,
			LastTime:  now,
		}
		return true, config.MaxRequests - 1, nil
	}

	record.Count++
	record.LastTime = now

	remaining := config.MaxRequests - record.Count
	if record.Count > config.MaxRequests {
		return false, 0, nil
	}

	return true, remaining, nil
}

// checkSlidingWindow 滑动窗口限流算法
func (rl *RateLimiter) checkSlidingWindow(ctx context.Context, key string, config *LimitConfig) (bool, int, error) {
	if rl.redis != nil {
		redisKey := fmt.Sprintf("ratelimit:sliding:%s", key)
		now := time.Now().UnixMilli()
		windowStart := now - config.WindowDuration.Milliseconds()

		pipe := rl.redis.Pipeline()
		pipe.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%d", windowStart))
		pipe.ZAdd(ctx, redisKey, redis.Z{Score: float64(now), Member: now})
		pipe.ZCard(ctx, redisKey)
		pipe.Expire(ctx, redisKey, config.WindowDuration)

		results, err := pipe.Exec(ctx)
		if err != nil {
			return true, 0, err
		}

		count := results[2].(*redis.IntCmd).Val()
		remaining := config.MaxRequests - int(count)

		if count >= int64(config.MaxRequests) {
			return false, 0, nil
		}

		return true, remaining, nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	record, exists := rl.requests[key]
	now := time.Now()

	if !exists {
		rl.requests[key] = &RequestRecord{
			Count:     1,
			FirstTime: now,
			LastTime:  now,
		}
		return true, config.MaxRequests - 1, nil
	}

	record.Count++

	remaining := config.MaxRequests - record.Count
	if record.Count > config.MaxRequests {
		return false, 0, nil
	}

	record.LastTime = now
	return true, remaining, nil
}

// checkTokenBucket 令牌桶限流算法
func (rl *RateLimiter) checkTokenBucket(ctx context.Context, key string, config *LimitConfig) (bool, int, error) {
	if rl.redis != nil {
		redisKey := fmt.Sprintf("ratelimit:token:%s", key)
		now := time.Now().UnixNano()
		luaScript := `
			local key = KEYS[1]
			local max_tokens = tonumber(ARGV[1])
			local refill_rate = tonumber(ARGV[2])
			local now = tonumber(ARGV[3])
			local requested = tonumber(ARGV[4])
			
			local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
			local tokens = tonumber(bucket[1])
			local last_refill = tonumber(bucket[2])
			
			if tokens == nil then
				tokens = max_tokens
				last_refill = now
			end
			
			local elapsed = (now - last_refill) / 1000000000
			local new_tokens = math.min(max_tokens, tokens + (elapsed * refill_rate))
			
			if new_tokens >= requested then
				new_tokens = new_tokens - requested
				redis.call('HMSET', key, 'tokens', new_tokens, 'last_refill', now)
				redis.call('EXPIRE', key, 3600)
				return {1, math.floor(new_tokens)}
			else
				return {0, math.floor(new_tokens)}
			end
		`

		result, err := rl.redis.Eval(ctx, luaScript, []string{redisKey},
			config.MaxRequests, rl.tokenRefillRate, now, 1).Result()
		if err != nil {
			return true, 0, err
		}

		values := result.([]interface{})
		allowed := values[0].(int64) == 1
		remaining := int(values[1].(int64))

		return allowed, remaining, nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	record, exists := rl.requests[key]
	now := time.Now()

	if !exists {
		record = &RequestRecord{
			Tokens:    float64(config.MaxRequests),
			FirstTime: now,
			LastTime:  now,
		}
		rl.requests[key] = record
	}

	record.Mu.Lock()
	defer record.Mu.Unlock()

	elapsed := now.Sub(record.LastTime).Seconds()
	record.Tokens = mathMin(float64(config.MaxRequests), record.Tokens+(elapsed*rl.tokenRefillRate))

	remaining := int(record.Tokens)
	if record.Tokens >= 1 {
		record.Tokens--
		record.LastTime = now
		return true, remaining - 1, nil
	}

	return false, 0, nil
}

// checkLeakyBucket 漏桶限流算法
func (rl *RateLimiter) checkLeakyBucket(ctx context.Context, key string, config *LimitConfig) (bool, int, error) {
	if rl.redis != nil {
		redisKey := fmt.Sprintf("ratelimit:leaky:%s", key)
		now := time.Now().UnixNano()
		leakRate := float64(config.MaxRequests) / config.WindowDuration.Seconds()

		luaScript := `
			local key = KEYS[1]
			local max_burst = tonumber(ARGV[1])
			local leak_rate = tonumber(ARGV[2])
			local now = tonumber(ARGV[3])
			
			local bucket = redis.call('HMGET', key, 'water', 'last_leak')
			local water = tonumber(bucket[1])
			local last_leak = tonumber(bucket[2])
			
			if water == nil then
				water = 0
				last_leak = now
			end
			
			local elapsed = (now - last_leak) / 1000000000
			local leaked = elapsed * leak_rate
			water = math.max(0, water - leaked)
			
			if water + 1 <= max_burst then
				water = water + 1
				redis.call('HMSET', key, 'water', water, 'last_leak', now)
				redis.call('EXPIRE', key, 3600)
				return {1, math.floor(max_burst - water)}
			else
				return {0, 0}
			end
		`

		result, err := rl.redis.Eval(ctx, luaScript, []string{redisKey},
			config.BurstSize, leakRate, now).Result()
		if err != nil {
			return true, 0, err
		}

		values := result.([]interface{})
		allowed := values[0].(int64) == 1
		remaining := int(values[1].(int64))

		return allowed, remaining, nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	record, exists := rl.requests[key]
	now := time.Now()

	if !exists {
		record = &RequestRecord{
			Tokens:    0,
			FirstTime: now,
			LastTime:  now,
		}
		rl.requests[key] = record
	}

	record.Mu.Lock()
	defer record.Mu.Unlock()

	elapsed := now.Sub(record.LastTime).Seconds()
	leakRate := float64(config.MaxRequests) / config.WindowDuration.Seconds()
	record.Tokens = mathMax(0, record.Tokens-(elapsed*leakRate))

	remaining := config.BurstSize - int(record.Tokens)
	if record.Tokens < float64(config.BurstSize) {
		record.Tokens++
		record.LastTime = now
		return true, remaining - 1, nil
	}

	return false, 0, nil
}

// checkLimit 检查限流
func (rl *RateLimiter) checkLimit(c *gin.Context, config *LimitConfig) (bool, int, error) {
	key := rl.generateKey(c, "")
	ctx := context.Background()

	switch rl.limiterType {
	case FixedWindow:
		return rl.checkFixedWindow(ctx, key, config)
	case SlidingWindow:
		return rl.checkSlidingWindow(ctx, key, config)
	case TokenBucket:
		return rl.checkTokenBucket(ctx, key, config)
	case LeakyBucket:
		return rl.checkLeakyBucket(ctx, key, config)
	default:
		return rl.checkFixedWindow(ctx, key, config)
	}
}

// cleanup 清理过期记录
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if time.Since(rl.lastCleanup) < rl.cleanupInterval {
		return
	}

	now := time.Now()
	for key, record := range rl.requests {
		if now.Sub(record.LastTime) > rl.cleanupInterval*2 {
			delete(rl.requests, key)
		}
	}

	rl.lastCleanup = now
}

// Middleware 返回限流中间件
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := rl.getIP(c)

		// 检查白名单
		if rl.enableWhitelist && rl.IsWhitelisted(ip) {
			c.Next()
			return
		}

		// 检查黑名单
		if rl.enableBlacklist && rl.IsBlacklisted(ip) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "访问被拒绝",
				"error":   "IP已被禁止访问",
			})
			return
		}

		config := rl.getLimitConfig(c)
		if config == nil {
			c.Next()
			return
		}

		allowed, remaining, err := rl.checkLimit(c, config)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.MaxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "请求过于频繁，请稍后再试",
				"error":   "rate limit exceeded",
				"retry_after": config.WindowDuration.Seconds(),
			})
			return
		}

		rl.cleanup()
		c.Next()
	}
}

// GlobalRateLimiter 全局限流器实例
var GlobalRateLimiter *RateLimiter

// InitGlobalRateLimiter 初始化全局限流器
func InitGlobalRateLimiter(config *RateLimiterConfig) {
	GlobalRateLimiter = NewRateLimiter(config)
}

// mathMin 最小值
func mathMin(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// mathMax 最大值
func mathMax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// RateLimitMiddleware 限流中间件包装器
func RateLimitMiddleware(limiterType RateLimiterType, globalLimit, ipLimit, userLimit, apiLimit *LimitConfig) gin.HandlerFunc {
	config := &RateLimiterConfig{
		LimiterType:  limiterType,
		GlobalLimit:  globalLimit,
		IPLimit:      ipLimit,
		UserLimit:    userLimit,
		APILimit:     apiLimit,
	}

	limiter := NewRateLimiter(config)
	return limiter.Middleware()
}

// ApplySecurityHeaders 应用安全响应头
func ApplySecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

// securityLogger 安全日志记录
func securityLogger(event string, ip string, details map[string]interface{}) {
	security.LogSecurityEvent(event, ip, details)
}
