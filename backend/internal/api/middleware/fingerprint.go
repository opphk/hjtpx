package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hjtpx/hjtpx/internal/service"
)

var fingerprintService = service.NewFingerprintService()

type FingerprintConfig struct {
	Enabled        bool
	CheckBlacklist bool
	BlockAnomaly    bool
	AnomalyThreshold float64
}

var defaultFingerprintConfig = FingerprintConfig{
	Enabled:        true,
	CheckBlacklist: true,
	BlockAnomaly:    false,
	AnomalyThreshold: 50.0,
}

func FingerprintMiddleware(config ...FingerprintConfig) gin.HandlerFunc {
	cfg := defaultFingerprintConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		additionalData := make(map[string]string)
		additionalData["screen_info"] = c.GetHeader("X-Screen-Info")
		additionalData["timezone"] = c.GetHeader("X-Timezone")
		additionalData["canvas_hash"] = c.GetHeader("X-Canvas-Hash")
		additionalData["webgl_hash"] = c.GetHeader("X-WebGL-Hash")

		result := fingerprintService.AnalyzeFingerprint(c.Request, additionalData)

		c.Set("fingerprint", result.Fingerprint)
		c.Set("fingerprint_id", result.Fingerprint.FingerprintID)
		c.Set("fingerprint_result", result)
		c.Set("risk_score", result.RiskScore)

		if cfg.CheckBlacklist {
			isBlacklisted, reason := fingerprintService.IsBlacklisted(result.Fingerprint.FingerprintID)
			if isBlacklisted {
				c.Set("is_blacklisted", true)
				c.Set("blacklist_reason", reason)
				c.Header("X-Blacklisted", "true")
				c.Header("X-Blacklist-Reason", reason)
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":   "access_denied",
					"message": "Your device has been blocked",
					"reason":  reason,
				})
				return
			}
		}

		if cfg.BlockAnomaly && result.IsAnomaly && result.RiskScore >= cfg.AnomalyThreshold {
			c.Set("is_anomaly", true)
			c.Set("anomaly_reason", result.AnomalyReason)
			c.Header("X-Anomaly-Detected", "true")
			c.Header("X-Anomaly-Reason", result.AnomalyReason)
			c.Header("X-Risk-Score", string(rune(result.RiskScore)))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "suspicious_activity",
				"message": "Suspicious activity detected",
				"reason":  result.AnomalyReason,
			})
			return
		}

		c.Next()
	}
}

func GetFingerprintService() *service.FingerprintService {
	return fingerprintService
}

func ExtractFingerprintFromContext(c *gin.Context) (*service.FingerprintData, bool) {
	fp, exists := c.Get("fingerprint")
	if !exists {
		return nil, false
	}
	return fp.(*service.FingerprintData), true
}

func GetFingerprintID(c *gin.Context) string {
	if id, exists := c.Get("fingerprint_id"); exists {
		return id.(string)
	}
	return ""
}

func GetRiskScore(c *gin.Context) float64 {
	if score, exists := c.Get("risk_score"); exists {
		return score.(float64)
	}
	return 0
}

func IsBlacklisted(c *gin.Context) bool {
	if blacklisted, exists := c.Get("is_blacklisted"); exists {
		return blacklisted.(bool)
	}
	return false
}

func AddFingerprintToBlacklist(c *gin.Context, reason string) error {
	fingerprintID := GetFingerprintID(c)
	if fingerprintID == "" {
		return nil
	}
	fingerprintService.AddToBlacklist(fingerprintID, reason)
	return nil
}

func RemoveFingerprintFromBlacklist(fingerprintID string) {
	fingerprintService.RemoveFromBlacklist(fingerprintID)
}
