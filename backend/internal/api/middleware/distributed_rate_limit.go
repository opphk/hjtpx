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
	Type        service.DistributedRateLimitType
	MaxRequests int
	WindowSecs  int
}

func DistributedRateLimitMiddleware(options *DistributedRateLimitOptions) gin.HandlerFunc {
	distributedOnce.Do(initDistributedRateLimitService)

	if options != nil {
		config := service.DistributedRateLimitConfig{
			Type:           options.Type,
			MaxRequests:    options.MaxRequests,
			WindowSecs:      options.WindowSecs,
			RedisKeyPrefix:  "dist:ratelimit:",
			ConsistencyMode: false,
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

		if !result.Allowed {
			c.Header("Retry-After", strconv.FormatInt(int64(result.ResetAt.Unix()-c.GetTime("request_time").Unix()), 10))
			response.TooManyRequests(c, "请求过于频繁，请稍后再试")
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
