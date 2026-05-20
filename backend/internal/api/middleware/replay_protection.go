package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hjtpx/hjtpx/internal/service"
)

var replayProtectionService *service.ReplayProtectionService

func init() {
	replayProtectionService = service.NewReplayProtectionService()
}

type ReplayProtectionConfig struct {
	Enabled        bool
	SecretKey      string
	ExcludePaths   []string
	ErrorOnFailure bool
}

var defaultReplayProtectionConfig = ReplayProtectionConfig{
	Enabled:        true,
	SecretKey:      "default-secret-key-change-in-production",
	ErrorOnFailure: true,
}

func ReplayProtectionMiddleware(config ...ReplayProtectionConfig) gin.HandlerFunc {
	cfg := defaultReplayProtectionConfig
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

		result := replayProtectionService.VerifyRequest(c.Request, cfg.SecretKey)

		c.Set("replay_verified", result.Valid)
		c.Set("replay_result", result)

		if !result.Valid && cfg.ErrorOnFailure {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "replay_protection_failed",
				"message": result.Reason,
			})
			return
		}

		c.Next()
	}
}

func pathHasPrefix(path, prefix string) bool {
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}

func GetReplayProtectionService() *service.ReplayProtectionService {
	return replayProtectionService
}

func GenerateReplayNonce() (string, error) {
	return replayProtectionService.GenerateNonce()
}

func IsReplayVerified(c *gin.Context) bool {
	if verified, exists := c.Get("replay_verified"); exists {
		return verified.(bool)
	}
	return false
}

func GetReplayVerificationResult(c *gin.Context) *service.ReplayVerificationResult {
	if result, exists := c.Get("replay_result"); exists {
		return result.(*service.ReplayVerificationResult)
	}
	return nil
}
