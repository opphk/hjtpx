package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/hjtpx/hjtpx/internal/service"
)

var smartRateLimitService *service.SmartRateLimitService

func init() {
	smartRateLimitService = service.NewSmartRateLimitService()
}

type SmartRateLimitConfig struct {
	Enabled        bool
	KeyFunc        func(c *gin.Context) string
	ExcludePaths   []string
	RiskScoreFunc  func(c *gin.Context) float64
}

var defaultSmartRateLimitConfig = SmartRateLimitConfig{
	Enabled: true,
	KeyFunc: func(c *gin.Context) string {
		return c.ClientIP()
	},
	RiskScoreFunc: func(c *gin.Context) float64 {
		if score, exists := c.Get("risk_score"); exists {
			return score.(float64)
		}
		return 0
	},
}

func SmartRateLimitMiddleware(config ...SmartRateLimitConfig) gin.HandlerFunc {
	cfg := defaultSmartRateLimitConfig
	if len(config) > 0 {
		cfg = config[0]
		if cfg.KeyFunc == nil {
			cfg.KeyFunc = defaultSmartRateLimitConfig.KeyFunc
		}
		if cfg.RiskScoreFunc == nil {
			cfg.RiskScoreFunc = defaultSmartRateLimitConfig.RiskScoreFunc
		}
	}

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		for _, excluded := range cfg.ExcludePaths {
			if path == excluded || pathHasPrefix(path, excluded+"/") {
				c.Next()
				return
			}
		}

		clientID := cfg.KeyFunc(c)
		riskScore := cfg.RiskScoreFunc(c)

		result := smartRateLimitService.CheckRateLimit(clientID, riskScore)

		c.Set("rate_limit_result", result)
		c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Limit-result.CurrentCount))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))
		c.Header("X-RateLimit-Tier", result.Tier)
		c.Header("X-Risk-Score", strconv.FormatFloat(result.RiskScore, 'f', 2, 64))

		if !result.Allowed {
			c.Header("Retry-After", strconv.Itoa(result.RetryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"message":     "Too many requests",
				"limit":       result.Limit,
				"tier":        result.Tier,
				"retry_after": result.RetryAfter,
			})
			return
		}

		c.Next()
	}
}

func GetSmartRateLimitService() *service.SmartRateLimitService {
	return smartRateLimitService
}

func GetRateLimitResult(c *gin.Context) *service.SmartRateLimitResult {
	if result, exists := c.Get("rate_limit_result"); exists {
		return result.(*service.SmartRateLimitResult)
	}
	return nil
}
