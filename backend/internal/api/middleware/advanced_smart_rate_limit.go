package middleware

import (
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var adaptiveRateLimitService *service.AdaptiveRateLimitService
var adaptiveOnce sync.Once

func initAdaptiveRateLimitService() {
	adaptiveRateLimitService = service.NewAdaptiveRateLimitService()
}

func AdaptiveRateLimitMiddleware(options *service.AdaptiveRateLimitConfig) gin.HandlerFunc {
	adaptiveOnce.Do(initAdaptiveRateLimitService)

	if options != nil {
		adaptiveRateLimitService.UpdateConfig(*options)
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = c.GetHeader("X-Forwarded-For")
			if ip == "" {
				ip = c.GetHeader("X-Real-IP")
			}
		}

		key := "adaptive:ip:" + ip

		allowed, err := adaptiveRateLimitService.Allow(c.Request.Context(), key)
		if err != nil {
			c.Next()
			return
		}

		stats := adaptiveRateLimitService.GetStats()
		c.Header("X-RateLimit-Adaptive", "enabled")

		if tierStats, ok := stats["sliding_window"].(map[string]interface{}); ok {
			if config, ok := tierStats["config"].(map[string]interface{}); ok {
				if maxReqs, ok := config["max_requests"].(int64); ok {
					c.Header("X-RateLimit-Limit", strconv.FormatInt(maxReqs, 10))
				}
			}
		}

		if !allowed {
			c.Header("Retry-After", "60")
			response.TooManyRequests(c, "系统负载较高，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

func GetAdaptiveRateLimitService() *service.AdaptiveRateLimitService {
	adaptiveOnce.Do(initAdaptiveRateLimitService)
	return adaptiveRateLimitService
}
