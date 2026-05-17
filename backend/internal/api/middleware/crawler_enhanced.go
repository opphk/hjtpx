package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

var (
	crawlerEnhancedService *service.CrawlerEnhancedDetectionService
	crawlerEnhancedOnce   = &sync.Once{}
)

func initCrawlerEnhanced() {
	crawlerEnhancedOnce.Do(func() {
		crawlerEnhancedService = service.NewCrawlerEnhancedDetectionService()
	})
}

type CrawlerEnhancedConfig struct {
	Enabled         bool
	ExcludePaths    []string
	BlockThreshold  float64
	ChallengeMode   bool
	LogAllDetections bool
}

var DefaultCrawlerEnhancedConfig = CrawlerEnhancedConfig{
	Enabled:          true,
	BlockThreshold:   0.7,
	ChallengeMode:    true,
	LogAllDetections: true,
}

func CrawlerEnhancedDetectionMiddleware(config ...CrawlerEnhancedConfig) gin.HandlerFunc {
	initCrawlerEnhanced()

	cfg := DefaultCrawlerEnhancedConfig
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
		additionalData["X-Webdriver"] = c.GetHeader("Webdriver")
		additionalData["X-Driver"] = c.GetHeader("Driver")

		result := crawlerEnhancedService.DetectCrawler(c.Request, additionalData)

		c.Set("crawler_detection", result)
		c.Set("is_crawler", result.IsCrawler)
		c.Set("crawler_type", result.CrawlerType)
		c.Set("crawler_confidence", result.Confidence)
		c.Set("crawler_risk_level", result.RiskLevel)

		c.Header("X-Crawler-Risk-Level", result.RiskLevel)
		c.Header("X-Crawler-Confidence", formatFloat(result.Confidence))

		if result.IsCrawler && result.RiskLevel == "high" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":          "Crawler detected",
				"code":           http.StatusForbidden,
				"message":        "Automated access detected and blocked",
				"crawler_type":   result.CrawlerType,
				"confidence":     result.Confidence,
				"risk_level":     result.RiskLevel,
				"reasons":        result.Reasons,
				"recommended_action": result.RecommendedAction,
			})
			return
		}

		if cfg.ChallengeMode && result.Confidence > cfg.BlockThreshold {
			c.Set("require_challenge", true)
			c.Set("challenge_type", result.ChallengeType)
			c.Header("X-Require-Challenge", "true")
			c.Header("X-Challenge-Type", result.ChallengeType)
		}

		if cfg.LogAllDetections && result.IsCrawler {
			logCrawlerDetection(c, result)
		}

		c.Next()
	}
}

func logCrawlerDetection(c *gin.Context, result *service.CrawlerDetectionResult) {
	c.Set("crawler_log", map[string]interface{}{
		"timestamp":    c.GetHeader("Date"),
		"client_ip":    c.ClientIP(),
		"path":        c.Request.URL.Path,
		"user_agent":  c.Request.UserAgent(),
		"is_crawler":   result.IsCrawler,
		"crawler_type": result.CrawlerType,
		"confidence":   result.Confidence,
		"risk_level":   result.RiskLevel,
		"reasons":      result.Reasons,
	})
}

type CrawlerStatsHandler struct{}

func (h *CrawlerStatsHandler) GetStats(c *gin.Context) {
	stats := crawlerEnhancedService.GetCrawlerStats()

	c.JSON(http.StatusOK, gin.H{
		"tracked_ips":         stats["tracked_ips"],
		"total_records":      stats["total_records"],
		"known_bot_types":    stats["known_bot_types"],
		"behavior_cache_size": stats["behavior_cache_size"],
	})
}

func (h *CrawlerStatsHandler) GetIPHistory(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		ip = c.ClientIP()
	}

	history := crawlerEnhancedService.GetRequestHistory(ip)

	c.JSON(http.StatusOK, gin.H{
		"ip":      ip,
		"history": history,
	})
}

func (h *CrawlerStatsHandler) ClearIPHistory(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		ip = c.ClientIP()
	}

	crawlerEnhancedService.ClearRequestHistory(ip)

	c.JSON(http.StatusOK, gin.H{
		"ip":      ip,
		"message": "Request history cleared",
	})
}

func (h *CrawlerStatsHandler) AddKnownBot(c *gin.Context) {
	var req struct {
		BotName     string  `json:"bot_name" binding:"required"`
		BotType     string  `json:"bot_type"`
		AllowSearch bool    `json:"allow_search"`
		AllowSocial bool    `json:"allow_social"`
		AllowAPI    bool    `json:"allow_api"`
		IsMalicious bool    `json:"is_malicious"`
		Confidence  float64 `json:"confidence"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	botType := service.CrawlerTypeUnknown
	if req.BotType != "" {
		botType = service.CrawlerType(req.BotType)
	}

	if req.Confidence == 0 {
		req.Confidence = 0.7
	}

	signature := &service.CrawlerSignature{
		Type:         botType,
		Name:         req.BotName,
		Confidence:   req.Confidence,
		AllowSearch:  req.AllowSearch,
		AllowSocial:  req.AllowSocial,
		AllowAPI:     req.AllowAPI,
		IsMalicious:  req.IsMalicious,
	}

	crawlerEnhancedService.AddKnownBotSignature(req.BotName, signature)

	c.JSON(http.StatusOK, gin.H{
		"bot_name": req.BotName,
		"message":  "Bot signature added",
	})
}

func (h *CrawlerStatsHandler) GetKnownBots(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"bots": crawlerEnhancedService.GetCrawlerStats()["known_bot_types"],
	})
}

func GetCrawlerEnhancedService() *service.CrawlerEnhancedDetectionService {
	initCrawlerEnhanced()
	return crawlerEnhancedService
}

func formatFloat(f float64) string {
	return string([]byte{
		byte('0') + byte(int(f*100)/100),
		'.',
		byte('0') + byte((int(f*100)%100)/10),
		byte('0') + byte(int(f*100)%10),
	})
}
