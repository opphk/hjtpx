package middleware

import (
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var distributedRateLimitService *service.DistributedRateLimitService
var distributedOnce sync.Once

func initDistributedRateLimitService() {
	distributedRateLimitService = service.NewDistributedRateLimitService()
}

type DistributedRateLimitOptions struct {
	Type            service.DistributedRateLimitType
	MaxRequests     int
	WindowSecs      int
	KeyPrefix       string
	ConsistencyMode bool
}

func DistributedRateLimitMiddleware(options *DistributedRateLimitOptions) gin.HandlerFunc {
	distributedOnce.Do(initDistributedRateLimitService)

	if options != nil {
		config := service.DistributedRateLimitConfig{
			Type:            options.Type,
			MaxRequests:     options.MaxRequests,
			WindowSecs:      options.WindowSecs,
			RedisKeyPrefix:  options.KeyPrefix,
			ConsistencyMode: options.ConsistencyMode,
		}
		distributedRateLimitService.UpdateConfig(config)
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = c.GetHeader("X-Forwarded-For")
			if ip == "" {
				ip = c.GetHeader("X-Real-IP")
			}
		}

		result, err := distributedRateLimitService.CheckIPRateLimit(c.Request.Context(), ip)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(result.Remaining+int(result.TotalCount)))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))
		c.Header("X-RateLimit-Node", result.NodeID)
		c.Header("X-RateLimit-Global", strconv.FormatInt(result.GlobalCount, 10))

		if !result.Allowed {
			c.Header("Retry-After", strconv.FormatInt(int64(result.ResetAt.Unix()-c.GetTime("request_time").Unix()), 10))
			response.TooManyRequests(c, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

func DistributedUserRateLimitMiddleware(options *DistributedRateLimitOptions) gin.HandlerFunc {
	distributedOnce.Do(initDistributedRateLimitService)

	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			c.Next()
			return
		}

		result, err := distributedRateLimitService.CheckUserRateLimit(c.Request.Context(), userID)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(result.Remaining+int(result.TotalCount)))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))
		c.Header("X-RateLimit-Node", result.NodeID)

		if !result.Allowed {
			c.Header("Retry-After", strconv.FormatInt(int64(result.ResetAt.Unix()-c.GetTime("request_time").Unix()), 10))
			response.TooManyRequests(c, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

func DistributedAppRateLimitMiddleware(options *DistributedRateLimitOptions) gin.HandlerFunc {
	distributedOnce.Do(initDistributedRateLimitService)

	return func(c *gin.Context) {
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

		result, err := distributedRateLimitService.CheckAppRateLimit(c.Request.Context(), uint(appID))
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(result.Remaining+int(result.TotalCount)))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))
		c.Header("X-RateLimit-Node", result.NodeID)

		if !result.Allowed {
			c.Header("Retry-After", strconv.FormatInt(int64(result.ResetAt.Unix()-c.GetTime("request_time").Unix()), 10))
			response.TooManyRequests(c, "应用请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

func GetDistributedRateLimitService() *service.DistributedRateLimitService {
	distributedOnce.Do(initDistributedRateLimitService)
	return distributedRateLimitService
}
