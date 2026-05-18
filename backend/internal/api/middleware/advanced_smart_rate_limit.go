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

		result, err := adaptiveRateLimitService.CheckIPRateLimit(c.Request.Context(), ip)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.FormatFloat(result.Capacity, 'f', 2, 64))
		c.Header("X-RateLimit-Remaining", strconv.FormatFloat(result.Tokens, 'f', 2, 64))
		c.Header("X-RateLimit-Rate", strconv.FormatFloat(result.CurrentRate, 'f', 2, 64))
		c.Header("X-RateLimit-LoadLevel", result.LoadLevel.String())
		c.Header("X-RateLimit-LoadFactor", strconv.FormatFloat(result.LoadFactor, 'f', 2, 64))

		if !result.Allowed {
			if result.RetryAfter > 0 {
				c.Header("Retry-After", strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
			}
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
