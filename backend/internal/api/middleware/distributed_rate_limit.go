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
	MaxRequests int
	WindowSecs  int
	KeyPrefix   string
}

func DistributedRateLimitMiddleware(options *DistributedRateLimitOptions) gin.HandlerFunc {
	distributedOnce.Do(initDistributedRateLimitService)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = c.GetHeader("X-Forwarded-For")
			if ip == "" {
				ip = c.GetHeader("X-Real-IP")
			}
		}

		key := "ip:" + ip
		maxRequests := int64(60)
		if options != nil && options.MaxRequests > 0 {
			maxRequests = int64(options.MaxRequests)
		}

		result, err := distributedRateLimitService.Check(c.Request.Context(), key, maxRequests)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(result.Remaining+int(maxRequests)))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

		if !result.Allowed {
			c.Header("Retry-After", strconv.FormatInt(int64(result.RetryAfter), 10))
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

		key := "user:" + strconv.FormatUint(uint64(userID), 10)
		maxRequests := int64(100)
		if options != nil && options.MaxRequests > 0 {
			maxRequests = int64(options.MaxRequests)
		}

		result, err := distributedRateLimitService.Check(c.Request.Context(), key, maxRequests)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(result.Remaining+int(maxRequests)))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

		if !result.Allowed {
			c.Header("Retry-After", strconv.FormatInt(int64(result.RetryAfter), 10))
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

		key := "app:" + strconv.FormatUint(appID, 10)
		maxRequests := int64(200)
		if options != nil && options.MaxRequests > 0 {
			maxRequests = int64(options.MaxRequests)
		}

		result, err := distributedRateLimitService.Check(c.Request.Context(), key, maxRequests)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(result.Remaining+int(maxRequests)))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

		if !result.Allowed {
			c.Header("Retry-After", strconv.FormatInt(int64(result.RetryAfter), 10))
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
