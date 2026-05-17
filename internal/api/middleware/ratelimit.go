package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hjtpx/internal/config"
	"hjtpx/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type RateLimiter struct {
	client    *redis.Client
	cfg       config.RateLimitConfig
	window    time.Duration
	luaScript *redis.Script
}

func NewRateLimiter(client *redis.Client, cfg config.RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		client: client,
		cfg:    cfg,
		window: time.Minute,
		luaScript: redis.NewScript(`
			local key = KEYS[1]
			local now = tonumber(ARGV[1])
			local window = tonumber(ARGV[2])
			local limit = tonumber(ARGV[3])
			local burst = tonumber(ARGV[4])
			
			redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)
			
			local count = redis.call('ZCARD', key)
			
			if count >= limit then
				return {0, count, limit}
			end
			
			redis.call('ZADD', key, now, now .. ':' .. math.random())
			redis.call('EXPIRE', key, window)
			
			return {1, count + 1, limit}
		`),
	}

	return rl
}

func (rl *RateLimiter) Allow(ctx context.Context, key string) (bool, int64, int64, error) {
	now := time.Now().UnixMilli()
	windowMs := int64(rl.window.Milliseconds())
	limit := int64(rl.cfg.RequestsPerMinute)
	burst := int64(rl.cfg.BurstSize)

	result, err := rl.luaScript.Run(ctx, rl.client, []string{key}, now, windowMs, limit, burst).Slice()
	if err != nil {
		return false, 0, 0, fmt.Errorf("rate limiter script error: %w", err)
	}

	allowed := result[0].(int64) == 1
	currentCount := result[1].(int64)
	maxLimit := result[2].(int64)

	return allowed, currentCount, maxLimit, nil
}

func RateLimitWithRedis(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.cfg.Enabled {
			c.Next()
			return
		}

		ctx := context.Background()
		key := fmt.Sprintf("ratelimit:%s:%s", c.ClientIP(), c.FullPath())

		allowed, current, limit, err := rl.Allow(ctx, key)
		if err != nil {
			utils.Error("Rate limit check error: %v", err)
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", limit-current))

		if !allowed {
			utils.TooManyRequests(c, "Rate limit exceeded")
			c.Abort()
			return
		}

		c.Next()
	}
}

func RateLimitByUser(jwtManager *utils.JWTManager, rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.cfg.Enabled {
			c.Next()
			return
		}

		ctx := context.Background()
		var key string

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 {
				claims, err := jwtManager.ValidateToken(parts[1])
				if err == nil {
					key = fmt.Sprintf("ratelimit:user:%d:%s", claims.UserID, c.FullPath())
				}
			}
		}

		if key == "" {
			key = fmt.Sprintf("ratelimit:ip:%s:%s", c.ClientIP(), c.FullPath())
		}

		allowed, current, limit, err := rl.Allow(ctx, key)
		if err != nil {
			utils.Error("Rate limit check error: %v", err)
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", limit-current))

		if !allowed {
			utils.TooManyRequests(c, "Rate limit exceeded")
			c.Abort()
			return
		}

		c.Next()
	}
}