package middleware

import (
	"log"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var rateLimitService = service.NewRateLimitService()

var (
	tokenBucketRateLimitService   *service.TokenBucketRateLimitService
	quotaManagementService        *service.QuotaManagementService
	advancedRateLimitServicesOnce sync.Once
)

func initAdvancedRateLimitServices() {
	tokenBucketRateLimitService = service.NewTokenBucketRateLimitService()
	quotaManagementService = service.NewQuotaManagementService()
}

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

// TokenBucketOptions 令牌桶配置
type TokenBucketOptions struct {
	Rate          float64
	Capacity      float64
	BurstSize     float64
	InitialTokens float64
}

// TokenBucketRateLimitMiddleware 令牌桶限流中间件
func TokenBucketRateLimitMiddleware(options *TokenBucketOptions) gin.HandlerFunc {
	advancedRateLimitServicesOnce.Do(initAdvancedRateLimitServices)

	return func(c *gin.Context) {
		if options == nil {
			options = &TokenBucketOptions{
				Rate:          10,
				Capacity:      100,
				BurstSize:     50,
				InitialTokens: 100,
			}
		}

		ip := c.ClientIP()
		if ip == "" {
			ip = c.GetHeader("X-Forwarded-For")
			if ip == "" {
				ip = c.GetHeader("X-Real-IP")
			}
		}

		config := &service.TokenBucketConfig{
			Capacity:     int64(options.Capacity),
			RefillRate:  options.Rate,
			RefillPerSec: options.Rate,
		}

		result, err := tokenBucketRateLimitService.CheckIPTokenBucketLimit(c.Request.Context(), ip, config)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-TokenBucket-Limit", strconv.FormatInt(config.Capacity, 10))
		c.Header("X-TokenBucket-Remaining", strconv.Itoa(result.Remaining))
		if result.RetryAfter > 0 {
			c.Header("X-TokenBucket-RetryAfter", strconv.Itoa(result.RetryAfter))
		}

		if !result.Allowed {
			if result.RetryAfter > 0 {
				c.Header("Retry-After", strconv.Itoa(result.RetryAfter))
			}
			response.TooManyRequests(c, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

// QuotaOptions 配额配置
type QuotaOptions struct {
	Type      service.QuotaType
	Limit     int64
	HardLimit bool
}

// QuotaMiddleware 配额中间件
func QuotaMiddleware(options *QuotaOptions) gin.HandlerFunc {
	advancedRateLimitServicesOnce.Do(initAdvancedRateLimitServices)

	return func(c *gin.Context) {
		if options == nil {
			options = &QuotaOptions{
				Type:      service.QuotaTypeDaily,
				Limit:     10000,
				HardLimit: true,
			}
		}

		var key string
		userID := GetUserID(c)
		if userID > 0 {
			key = service.UserQuotaKey(userID, "api", options.Type)
		} else {
			appIDStr := c.GetHeader("X-App-ID")
			if appIDStr != "" {
				if appID, err := strconv.ParseUint(appIDStr, 10, 64); err == nil {
					key = service.AppQuotaKey(uint(appID), "api", options.Type)
				}
			}
			if key == "" {
				ip := c.ClientIP()
				if ip == "" {
					ip = c.GetHeader("X-Forwarded-For")
					if ip == "" {
						ip = c.GetHeader("X-Real-IP")
					}
				}
				key = "ip:" + ip + ":" + string(options.Type)
			}
		}

		// 确保配额存在
		_, err := quotaManagementService.GetQuota(c.Request.Context(), key)
		if err == nil {
			config := &service.QuotaConfig{
				Type:             options.Type,
				Limit:            options.Limit,
				WarningThreshold: 80,
				HardLimit:        options.HardLimit,
			}
			if err := quotaManagementService.CreateOrUpdateQuota(c.Request.Context(), key, config); err != nil {
				log.Printf("创建或更新配额失败: %v", err)
			}
		}

		status, allowed, err := quotaManagementService.ConsumeQuota(c.Request.Context(), key, 1)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-Quota-Limit", strconv.FormatInt(status.Limit, 10))
		c.Header("X-Quota-Remaining", strconv.FormatInt(status.Remaining, 10))
		c.Header("X-Quota-ResetAt", strconv.FormatInt(status.ResetAt.Unix(), 10))

		if !allowed {
			response.TooManyRequests(c, "配额已用尽，请稍后再试或升级套餐")
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdvancedCombinedMiddleware 高级组合限流中间件
func AdvancedCombinedMiddleware(tbOptions *TokenBucketOptions, quotaOptions *QuotaOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		if tbOptions != nil {
			tbHandler := TokenBucketRateLimitMiddleware(tbOptions)
			tbHandler(c)
			if c.IsAborted() {
				return
			}
		}

		if quotaOptions != nil && !c.IsAborted() {
			quotaHandler := QuotaMiddleware(quotaOptions)
			quotaHandler(c)
			if c.IsAborted() {
				return
			}
		}

		if !c.IsAborted() {
			c.Next()
		}
	}
}

// GetTokenBucketRateLimitService 获取令牌桶服务
func GetTokenBucketRateLimitService() *service.TokenBucketRateLimitService {
	advancedRateLimitServicesOnce.Do(initAdvancedRateLimitServices)
	return tokenBucketRateLimitService
}

// GetQuotaManagementService 获取配额管理服务
func GetQuotaManagementService() *service.QuotaManagementService {
	advancedRateLimitServicesOnce.Do(initAdvancedRateLimitServices)
	return quotaManagementService
}
