package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type RateLimitConfig struct {
	MaxRequests    int
	WindowSeconds  int
	KeyPrefix      string
	BlockDuration  int
	EnableBurst    bool
	BurstSize      int
}

var DefaultRateLimitConfig = &RateLimitConfig{
	MaxRequests:    100,
	WindowSeconds:  60,
	KeyPrefix:      "ratelimit:",
	BlockDuration:  300,
	EnableBurst:    false,
	BurstSize:      0,
}

var CaptchaRateLimitConfig = &RateLimitConfig{
	MaxRequests:    60,
	WindowSeconds:  60,
	KeyPrefix:      "ratelimit:captcha:",
	BlockDuration:  300,
	EnableBurst:    false,
	BurstSize:      10,
}

var AuthRateLimitConfig = &RateLimitConfig{
	MaxRequests:    10,
	WindowSeconds:  60,
	KeyPrefix:      "ratelimit:auth:",
	BlockDuration:  600,
	EnableBurst:    false,
	BurstSize:      5,
}

var AdminRateLimitConfig = &RateLimitConfig{
	MaxRequests:    200,
	WindowSeconds:  60,
	KeyPrefix:      "ratelimit:admin:",
	BlockDuration:  300,
	EnableBurst:    false,
	BurstSize:      20,
}

var APIRateLimitConfig = &RateLimitConfig{
	MaxRequests:    1000,
	WindowSeconds:  60,
	KeyPrefix:      "ratelimit:api:",
	BlockDuration:  300,
	EnableBurst:    true,
	BurstSize:      50,
}

func GetRateLimitConfig(level string) *RateLimitConfig {
	switch level {
	case "high":
		return &RateLimitConfig{
			MaxRequests:    10,
			WindowSeconds:  60,
			BlockDuration:  600,
			EnableBurst:    false,
			BurstSize:      0,
		}
	case "medium":
		return &RateLimitConfig{
			MaxRequests:    100,
			WindowSeconds:  60,
			BlockDuration:  300,
			EnableBurst:    false,
			BurstSize:      20,
		}
	case "low":
		return &RateLimitConfig{
			MaxRequests:    1000,
			WindowSeconds:  60,
			BlockDuration:  60,
			EnableBurst:    true,
			BurstSize:      100,
		}
	default:
		return DefaultRateLimitConfig
	}
}

func CreateRateLimitMiddleware(config *RateLimitConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultRateLimitConfig
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = c.GetHeader("X-Forwarded-For")
			if ip == "" {
				ip = c.GetHeader("X-Real-IP")
			}
		}

		key := config.KeyPrefix + ip

		remaining := config.MaxRequests - 1

		c.Header("X-RateLimit-Limit", strconv.Itoa(config.MaxRequests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(int64(config.WindowSeconds), 10))

		if config.BlockDuration > 0 {
			c.Header("X-RateLimit-Window", strconv.Itoa(config.WindowSeconds))
		}

		c.Next()
	}
}

func CaptchaRateLimitMiddleware() gin.HandlerFunc {
	return CreateRateLimitMiddleware(CaptchaRateLimitConfig)
}

func AuthRateLimitMiddleware() gin.HandlerFunc {
	return CreateRateLimitMiddleware(AuthRateLimitConfig)
}

func AdminRateLimitMiddleware() gin.HandlerFunc {
	return CreateRateLimitMiddleware(AdminRateLimitConfig)
}

func APIRateLimitMiddleware() gin.HandlerFunc {
	return CreateRateLimitMiddleware(APIRateLimitConfig)
}

type RateLimitStatus struct {
	Allowed   bool
	Remaining int
	Reset     int64
	Limit     int
}

func GetRateLimitStatus(c *gin.Context) *RateLimitStatus {
	limit := c.GetHeader("X-RateLimit-Limit")
	remaining := c.GetHeader("X-RateLimit-Remaining")
	reset := c.GetHeader("X-RateLimit-Reset")

	limitInt, _ := strconv.Atoi(limit)
	remainingInt, _ := strconv.Atoi(remaining)
	resetInt, _ := strconv.ParseInt(reset, 10, 64)

	return &RateLimitStatus{
		Allowed:   remainingInt > 0,
		Remaining: remainingInt,
		Reset:     resetInt,
		Limit:     limitInt,
	}
}

func CheckRateLimit(c *gin.Context) bool {
	status := GetRateLimitStatus(c)
	if !status.Allowed {
		response.TooManyRequests(c, "rate limit exceeded")
		return false
	}
	return true
}
