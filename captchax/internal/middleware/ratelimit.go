package middleware

import (
	"captchax/internal/service"
	"captchax/pkg/response"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimitConfig struct {
	RequestsPerMinute int
	BurstSize         int
	Enabled           bool
}

var defaultRateLimitConfig = &RateLimitConfig{
	RequestsPerMinute: 60,
	BurstSize:         10,
	Enabled:           true,
}

type ipRateLimiter struct {
	requests map[string][]time.Time
	lastCleanup time.Time
}

var rateLimiters = make(map[string]*ipRateLimiter)

func getRateLimiter(key string) *ipRateLimiter {
	if limiter, exists := rateLimiters[key]; exists {
		return limiter
	}
	limiter := &ipRateLimiter{
		requests:    make(map[string][]time.Time),
		lastCleanup: time.Now(),
	}
	rateLimiters[key] = limiter
	return limiter
}

func (rl *ipRateLimiter) isAllowed(ip string, limit int, window time.Duration) bool {
	now := time.Now()
	windowStart := now.Add(-window)

	requests := rl.requests[ip]
	var validRequests []time.Time
	for _, t := range requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	if len(validRequests) >= limit {
		rl.requests[ip] = validRequests
		return false
	}

	rl.requests[ip] = append(validRequests, now)

	if now.Sub(rl.lastCleanup) > 5*time.Minute {
		rl.cleanup(window)
		rl.lastCleanup = now
	}

	return true
}

func (rl *ipRateLimiter) cleanup(window time.Duration) {
	now := time.Now()
	windowStart := now.Add(-window)

	for ip, requests := range rl.requests {
		var validRequests []time.Time
		for _, t := range requests {
			if t.After(windowStart) {
				validRequests = append(validRequests, t)
			}
		}
		if len(validRequests) == 0 {
			delete(rl.requests, ip)
		} else {
			rl.requests[ip] = validRequests
		}
	}
}

func RateLimit(captchaService *service.CaptchaService) gin.HandlerFunc {
	return RateLimitWithConfig(captchaService, defaultRateLimitConfig)
}

func RateLimitWithConfig(captchaService *service.CaptchaService, config *RateLimitConfig) gin.HandlerFunc {
	if !config.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	limit := config.RequestsPerMinute
	window := time.Minute

	return func(c *gin.Context) {
		clientIP := getClientIP(c)
		rateLimitKey := fmt.Sprintf("ratelimit:%s", clientIP)

		if captchaService != nil {
			ctx := context.Background()
			allowed, remaining, resetAt, err := captchaService.CheckRateLimit(ctx, clientIP, limit)
			if err == nil && !allowed {
				c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
				c.Header("X-RateLimit-Remaining", "0")
				c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))
				c.Header("Retry-After", fmt.Sprintf("%d", int(time.Until(resetAt).Seconds())))

				response.ErrorWithStatus(c, http.StatusTooManyRequests, 429,
					fmt.Sprintf("rate limit exceeded, retry after %d seconds", int(time.Until(resetAt).Seconds())))
				c.Abort()
				return
			}

			if err == nil {
				c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
				c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
				c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))
			}
		} else {
			limiter := getRateLimiter(rateLimitKey)
			if !limiter.isAllowed(clientIP, limit, window) {
				c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
				c.Header("X-RateLimit-Remaining", "0")
				c.Header("Retry-After", "60")

				response.ErrorWithStatus(c, http.StatusTooManyRequests, 429,
					"rate limit exceeded, retry after 60 seconds")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

func getClientIP(c *gin.Context) string {
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	xri := c.GetHeader("X-Real-IP")
	if xri != "" {
		return xri
	}

	return c.ClientIP()
}
