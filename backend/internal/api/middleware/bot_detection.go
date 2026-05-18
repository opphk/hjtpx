package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

var (
	botDetectionService *service.BotDetectionService
	botDetectionOnce    = &sync.Once{}
)

func initBotDetectionService() {
	botDetectionOnce.Do(func() {
		botDetectionService = service.NewBotDetectionService()
	})
}

type BotDetectionMiddlewareConfig struct {
	Enabled        bool
	ExcludePaths   []string
	BlockThreshold float64
	ChallengeMode  bool
}

var DefaultBotDetectionConfig = BotDetectionMiddlewareConfig{
	Enabled:        true,
	BlockThreshold: 0.7,
	ChallengeMode:  true,
}

func BotDetectionMiddleware(config ...BotDetectionMiddlewareConfig) gin.HandlerFunc {
	initBotDetectionService()

	cfg := DefaultBotDetectionConfig
	if len(config) > 0 {
		cfg = config[0]
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

		additionalData := make(map[string]string)
		additionalData["X-Screen-Info"] = c.GetHeader("X-Screen-Info")
		additionalData["X-Timezone"] = c.GetHeader("X-Timezone")
		additionalData["X-Canvas-Hash"] = c.GetHeader("X-Canvas-Hash")
		additionalData["X-WebGL-Hash"] = c.GetHeader("X-WebGL-Hash")

		result := botDetectionService.DetectBot(c.Request, additionalData)

		c.Set("bot_detection", result)
		c.Set("is_bot", result.IsBot)
		c.Set("bot_risk_score", result.RiskScore)

		if result.IsBot && result.ShouldBlock {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "Bot detected",
				"code":    http.StatusForbidden,
				"message": "Access denied - suspicious activity detected",
				"reasons": result.Reasons,
			})
			return
		}

		if cfg.ChallengeMode && result.RiskScore > cfg.BlockThreshold {
			c.Set("require_challenge", true)
			c.Set("challenge_type", result.ChallengeType)
		}

		c.Next()
	}
}

func GetBotDetectionService() *service.BotDetectionService {
	initBotDetectionService()
	return botDetectionService
}
