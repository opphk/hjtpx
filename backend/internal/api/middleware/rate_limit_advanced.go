package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type RateLimitConfig struct {
	MaxRequests       int
	WindowSecs        int
	EnableRedis       bool
	EnableDistributed bool
	KeyPrefix         string
	BlockDuration     time.Duration
}

type rateLimitEntry struct {
	count     int
	resetTime time.Time
}

type advancedRateLimiter struct {
	entries     map[string]*rateLimitEntry
	mu          sync.RWMutex
	config      RateLimitConfig
	redisKeyTTL time.Duration
}

var defaultAdvancedRateLimitConfig = RateLimitConfig{
	MaxRequests:       100,
	WindowSecs:        60,
	EnableRedis:       false,
	EnableDistributed: false,
	KeyPrefix:         "ratelimit:",
	BlockDuration:     5 * time.Minute,
}

var globalAdvancedRateLimiter = &advancedRateLimiter{
	entries:     make(map[string]*rateLimitEntry),
	config:      defaultAdvancedRateLimitConfig,
	redisKeyTTL: time.Minute,
}

func init() {
	go globalAdvancedRateLimiter.cleanupLoop()
}

func (r *advancedRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		r.cleanup()
	}
}

func (r *advancedRateLimiter) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for key, entry := range r.entries {
		if now.After(entry.resetTime) {
			delete(r.entries, key)
		}
	}
}

func (r *advancedRateLimiter) checkLimit(key string, maxRequests int, windowSecs int) (bool, int, time.Time) {
	now := time.Now()
	windowDuration := time.Duration(windowSecs) * time.Second

	if r.config.EnableRedis && redis.Client != nil {
		return r.checkRedisLimit(key, maxRequests, windowDuration)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.entries[key]
	if !exists || now.After(entry.resetTime) {
		r.entries[key] = &rateLimitEntry{
			count:     1,
			resetTime: now.Add(windowDuration),
		}
		return true, maxRequests - 1, now.Add(windowDuration)
	}

	entry.count++
	remaining := maxRequests - entry.count

	if entry.count > maxRequests {
		return false, 0, entry.resetTime
	}

	return true, remaining, entry.resetTime
}

func (r *advancedRateLimiter) checkRedisLimit(key string, maxRequests int, windowDuration time.Duration) (bool, int, time.Time) {
	ctx := context.Background()
	redisKey := r.config.KeyPrefix + key

	pipe := redis.Client.Pipeline()
	incrCmd := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, windowDuration)
	_, err := pipe.Exec(ctx)

	if err != nil {
		return true, maxRequests - 1, time.Now().Add(windowDuration)
	}

	count, _ := incrCmd.Result()
	remaining := maxRequests - int(count)
	resetTime := time.Now().Add(windowDuration)

	if count > int64(maxRequests) {
		return false, 0, resetTime
	}

	return true, remaining, resetTime
}

func (r *advancedRateLimiter) isBlocked(key string) bool {
	if !r.config.EnableRedis || redis.Client == nil {
		return false
	}

	ctx := context.Background()
	blockKey := r.config.KeyPrefix + "blocked:" + key
	exists, err := redis.Client.Exists(ctx, blockKey).Result()
	if err != nil {
		return false
	}

	return exists > 0
}

func (r *advancedRateLimiter) block(key string, duration time.Duration) {
	if !r.config.EnableRedis || redis.Client == nil {
		return
	}

	ctx := context.Background()
	blockKey := r.config.KeyPrefix + "blocked:" + key
	redis.Client.Set(ctx, blockKey, "1", duration)
}

func AdvancedRateLimit(config ...RateLimitConfig) gin.HandlerFunc {
	cfg := defaultAdvancedRateLimitConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	limiter := &advancedRateLimiter{
		entries:     make(map[string]*rateLimitEntry),
		config:      cfg,
		redisKeyTTL: time.Duration(cfg.WindowSecs) * time.Second,
	}
	go limiter.cleanupLoop()

	return func(c *gin.Context) {
		key := c.ClientIP()

		if appID := c.GetHeader("X-App-ID"); appID != "" {
			key = "app:" + appID
		}

		if userID, exists := c.Get("user_id"); exists {
			key = fmt.Sprintf("user:%d", userID)
		}

		allowed, remaining, resetTime := limiter.checkLimit(key, cfg.MaxRequests, cfg.WindowSecs)

		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.MaxRequests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if !allowed {
			limiter.block(key, cfg.BlockDuration)

			c.Header("Retry-After", strconv.Itoa(cfg.WindowSecs))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"message":     "Too many requests",
				"retry_after": cfg.WindowSecs,
			})
			return
		}

		c.Next()
	}
}

type TokenBucketConfig struct {
	Capacity     int64
	RefillRate   float64
	RefillPerSec float64
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
	capacity   int64
	refillRate float64
	mu         sync.Mutex
}

var globalTokenBuckets = &sync.Map{}

func newTokenBucket(config TokenBucketConfig) *tokenBucket {
	return &tokenBucket{
		tokens:     float64(config.Capacity),
		lastRefill: time.Now(),
		capacity:   config.Capacity,
		refillRate: config.RefillRate,
	}
}

func (tb *tokenBucket) Allow() (bool, float64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate

	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}

	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true, tb.tokens
	}

	return false, tb.tokens
}

func TokenBucketLimit(config ...TokenBucketConfig) gin.HandlerFunc {
	cfg := TokenBucketConfig{
		Capacity:     100,
		RefillRate:   10,
		RefillPerSec: 10,
	}
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		key := c.ClientIP()

		bucketInterface, _ := globalTokenBuckets.LoadOrStore(key, newTokenBucket(cfg))
		bucket := bucketInterface.(*tokenBucket)

		allowed, remaining := bucket.Allow()

		c.Header("X-TokenBucket-Limit", strconv.FormatInt(cfg.Capacity, 10))
		c.Header("X-TokenBucket-Remaining", strconv.FormatFloat(remaining, 'f', 2, 64))

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests",
			})
			return
		}

		c.Next()
	}
}

func GetRateLimitStatus(c *gin.Context) {
	key := c.ClientIP()

	entry, exists := globalAdvancedRateLimiter.entries[key]

	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"key":       key,
			"count":     0,
			"limit":     defaultAdvancedRateLimitConfig.MaxRequests,
			"remaining": defaultAdvancedRateLimitConfig.MaxRequests,
			"reset":     time.Now().Add(time.Duration(defaultAdvancedRateLimitConfig.WindowSecs) * time.Second).Unix(),
		})
		return
	}

	remaining := defaultAdvancedRateLimitConfig.MaxRequests - entry.count
	if remaining < 0 {
		remaining = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"key":       key,
		"count":     entry.count,
		"limit":     defaultAdvancedRateLimitConfig.MaxRequests,
		"remaining": remaining,
		"reset":     entry.resetTime.Unix(),
	})
}
