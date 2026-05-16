package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var rateLimitService = service.NewRateLimitService()

type RateLimitOptions struct {
	MaxRequests int
	WindowSecs  int
}

func IPRateLimitMiddleware(options *RateLimitOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		if options == nil {
			options = &RateLimitOptions{
				MaxRequests: 100,
				WindowSecs:  60,
			}
		}

		ip := c.ClientIP()
		if ip == "" {
			ip = c.GetHeader("X-Forwarded-For")
			if ip == "" {
				ip = c.GetHeader("X-Real-IP")
			}
		}

		config := &service.RateLimitConfig{
			MaxRequests: options.MaxRequests,
			WindowSecs:  options.WindowSecs,
		}

		result, err := rateLimitService.CheckIPRateLimit(c.Request.Context(), ip, config)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(options.MaxRequests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

		if !result.Allowed {
			c.Header("Retry-After", strconv.Itoa(options.WindowSecs))
			response.TooManyRequests(c, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

func UserRateLimitMiddleware(options *RateLimitOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		if options == nil {
			options = &RateLimitOptions{
				MaxRequests: 200,
				WindowSecs:  60,
			}
		}

		userID := GetUserID(c)
		if userID == 0 {
			c.Next()
			return
		}

		config := &service.RateLimitConfig{
			MaxRequests: options.MaxRequests,
			WindowSecs:  options.WindowSecs,
		}

		result, err := rateLimitService.CheckUserRateLimit(c.Request.Context(), userID, config)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(options.MaxRequests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

		if !result.Allowed {
			c.Header("Retry-After", strconv.Itoa(options.WindowSecs))
			response.TooManyRequests(c, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

func AppRateLimitMiddleware(options *RateLimitOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		if options == nil {
			options = &RateLimitOptions{
				MaxRequests: 500,
				WindowSecs:  60,
			}
		}

		appIDStr := c.GetHeader("X-App-ID")
		if appIDStr == "" {
			c.Next()
			return
		}

		appID, err := strconv.ParseUint(appIDStr, 10, 64)
		if err != nil {
			c.Next()
			return
		}

		config := &service.RateLimitConfig{
			MaxRequests: options.MaxRequests,
			WindowSecs:  options.WindowSecs,
		}

		result, err := rateLimitService.CheckAppRateLimit(c.Request.Context(), uint(appID), config)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(options.MaxRequests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

		if !result.Allowed {
			c.Header("Retry-After", strconv.Itoa(options.WindowSecs))
			response.TooManyRequests(c, "应用请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

func CombinedRateLimitMiddleware(ipOptions, userOptions, appOptions *RateLimitOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = c.GetHeader("X-Forwarded-For")
			if ip == "" {
				ip = c.GetHeader("X-Real-IP")
			}
		}

		if ipOptions != nil {
			ipConfig := &service.RateLimitConfig{
				MaxRequests: ipOptions.MaxRequests,
				WindowSecs:  ipOptions.WindowSecs,
			}
			result, err := rateLimitService.CheckIPRateLimit(c.Request.Context(), ip, ipConfig)
			if err == nil && !result.Allowed {
				c.Header("Retry-After", strconv.Itoa(ipOptions.WindowSecs))
				response.TooManyRequests(c, "请求过于频繁，请稍后再试")
				c.Abort()
				return
			}
		}

		userID := GetUserID(c)
		if userID > 0 && userOptions != nil {
			userConfig := &service.RateLimitConfig{
				MaxRequests: userOptions.MaxRequests,
				WindowSecs:  userOptions.WindowSecs,
			}
			result, err := rateLimitService.CheckUserRateLimit(c.Request.Context(), userID, userConfig)
			if err == nil && !result.Allowed {
				c.Header("Retry-After", strconv.Itoa(userOptions.WindowSecs))
				response.TooManyRequests(c, "请求过于频繁，请稍后再试")
				c.Abort()
				return
			}
		}

		appIDStr := c.GetHeader("X-App-ID")
		if appIDStr != "" && appOptions != nil {
			if appID, err := strconv.ParseUint(appIDStr, 10, 64); err == nil {
				appConfig := &service.RateLimitConfig{
					MaxRequests: appOptions.MaxRequests,
					WindowSecs:  appOptions.WindowSecs,
				}
				result, err := rateLimitService.CheckAppRateLimit(c.Request.Context(), uint(appID), appConfig)
				if err == nil && !result.Allowed {
					c.Header("Retry-After", strconv.Itoa(appOptions.WindowSecs))
					response.TooManyRequests(c, "应用请求过于频繁，请稍后再试")
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}

func RecordViolationMiddleware(violationType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := c.ClientIP()
		if identifier == "" {
			identifier = c.GetHeader("X-Forwarded-For")
			if identifier == "" {
				identifier = c.GetHeader("X-Real-IP")
			}
		}

		rateLimitService.RecordViolation(c.Request.Context(), identifier, violationType)
		c.Next()
	}
}

func RecordFailedAttemptMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := c.ClientIP()
		if identifier == "" {
			identifier = c.GetHeader("X-Forwarded-For")
			if identifier == "" {
				identifier = c.GetHeader("X-Real-IP")
			}
		}

		failedCount, _ := rateLimitService.RecordFailedAttempt(c.Request.Context(), identifier)
		if failedCount >= 5 {
			_, _ = rateLimitService.RecordViolation(c.Request.Context(), identifier, "failed_attempts")
		}

		c.Next()
	}
}

func ClearFailedAttemptsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := c.ClientIP()
		if identifier == "" {
			identifier = c.GetHeader("X-Forwarded-For")
			if identifier == "" {
				identifier = c.GetHeader("X-Real-IP")
			}
		}

		rateLimitService.ClearFailedAttempts(c.Request.Context(), identifier)
		c.Next()
	}
}

func GetRateLimitService() *service.RateLimitService {
	return rateLimitService
}
