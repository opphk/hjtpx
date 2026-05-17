package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

var (
	ddosEnhancedService *service.DDoSEnhancedProtectionService
	ddosEnhancedOnce    = &sync.Once{}
)

func initDDoSEnhanced() {
	ddosEnhancedOnce.Do(func() {
		ddosEnhancedService = service.NewDDoSEnhancedProtectionService(nil)
	})
}

type DDoSEnhancedConfig struct {
	Enabled               bool
	ExcludePaths          []string
	EnableTrafficCleaning  bool
	EnableIPReputation    bool
	EnableAdaptiveLimits  bool
	EnableBandwidthThrottle bool
	MaxBandwidthGbps     float64
	RequestsPerSecond     int
	BurstSize            int
	BlockDurationSeconds  int
}

var DefaultDDoSEnhancedConfig = DDoSEnhancedConfig{
	Enabled:               true,
	EnableTrafficCleaning: true,
	EnableIPReputation:    true,
	EnableAdaptiveLimits:  true,
	EnableBandwidthThrottle: true,
	MaxBandwidthGbps:     10.0,
	RequestsPerSecond:    100,
	BurstSize:           200,
	BlockDurationSeconds: 300,
}

func DDoSEnhancedProtectionMiddleware(config ...DDoSEnhancedConfig) gin.HandlerFunc {
	initDDoSEnhanced()

	cfg := DefaultDDoSEnhancedConfig
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

		result := ddosEnhancedService.CheckRequest(c.Request)
		c.Set("ddos_result", result)

		if !result.Allowed {
			c.Header("X-RateLimit-Reason", result.Reason)

			if result.RetryAfter > 0 {
				c.Header("Retry-After", formatRetryAfter(result.RetryAfter))
			}

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Request blocked by DDoS protection",
				"code":        http.StatusTooManyRequests,
				"reason":      result.Reason,
				"retry_after": result.RetryAfter,
			})
			return
		}

		if cfg.EnableTrafficCleaning {
			cleanResult := ddosEnhancedService.CleanTraffic(c.ClientIP(), c.Request)
			c.Set("traffic_clean_result", cleanResult)

			if cleanResult.Cleaned {
				c.Header("X-Traffic-Cleaned", "true")
				c.Header("X-Clean-Reason", cleanResult.DropReason)
			}
		}

		reputation := ddosEnhancedService.GetIPReputation(c.ClientIP())
		c.Set("ip_reputation", reputation)
		c.Header("X-IP-Trust-Level", formatInt(reputation.TrustLevel))

		c.Next()
	}
}

type DDoSStatsHandler struct{}

func (h *DDoSStatsHandler) GetStats(c *gin.Context) {
	stats := ddosEnhancedService.GetProtectionStats()

	c.JSON(http.StatusOK, gin.H{
		"total_requests":     stats.TotalRequests,
		"cleaned_requests":   stats.CleanedRequests,
		"dropped_bytes":      stats.DroppedBytes,
		"peak_bandwidth":     stats.PeakBandwidth,
		"active_ips":         stats.ActiveIPs,
		"blacklisted_ips":    stats.BlacklistedIPs,
		"tracked_reputations": stats.TrackedReputations,
		"clean_rate":         stats.CleanRate,
		"timestamp":          time.Now().UTC(),
	})
}

func (h *DDoSStatsHandler) GetIPReputation(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		ip = c.ClientIP()
	}

	reputation := ddosEnhancedService.GetIPReputation(ip)

	c.JSON(http.StatusOK, gin.H{
		"ip":               reputation.IP,
		"score":            reputation.Score,
		"tier":             reputation.Tier,
		"trust_level":      reputation.TrustLevel,
		"is_whitelisted":   reputation.IsWhitelisted,
		"is_blacklisted":   reputation.IsBlacklisted,
		"request_count":    reputation.RequestCount,
		"block_count":      reputation.BlockCount,
		"last_updated":     reputation.LastUpdated,
	})
}

func (h *DDoSStatsHandler) SetIPWhitelist(c *gin.Context) {
	var req struct {
		IP         string `json:"ip" binding:"required"`
		Whitelisted bool   `json:"whitelisted"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ddosEnhancedService.SetIPWhitelist(req.IP, req.Whitelisted)

	c.JSON(http.StatusOK, gin.H{
		"ip":          req.IP,
		"whitelisted": req.Whitelisted,
		"message":     "IP whitelist status updated",
	})
}

func (h *DDoSStatsHandler) SetIPBlacklist(c *gin.Context) {
	var req struct {
		IP          string `json:"ip" binding:"required"`
		Blacklisted bool   `json:"blacklisted"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ddosEnhancedService.SetIPBlacklist(req.IP, req.Blacklisted)

	c.JSON(http.StatusOK, gin.H{
		"ip":          req.IP,
		"blacklisted": req.Blacklisted,
		"message":     "IP blacklist status updated",
	})
}

func (h *DDoSStatsHandler) GetRateLimitStrategy(c *gin.Context) {
	strategy := ddosEnhancedService.GetRateLimitStrategy()

	c.JSON(http.StatusOK, gin.H{
		"name":            strategy.Name,
		"requests_per_sec": strategy.RequestsPerSec,
		"burst_size":       strategy.BurstSize,
		"window_size":      strategy.WindowSize.String(),
		"adaptive":         strategy.Adaptive,
		"block_duration":   strategy.BlockDuration.String(),
	})
}

func (h *DDoSStatsHandler) UpdateRateLimitStrategy(c *gin.Context) {
	var req struct {
		Name            string `json:"name"`
		RequestsPerSec  int    `json:"requests_per_sec"`
		BurstSize      int    `json:"burst_size"`
		Adaptive       bool   `json:"adaptive"`
		BlockDurationSecs int `json:"block_duration_secs"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	strategy := &service.RateLimitStrategy{
		Name:           req.Name,
		RequestsPerSec: req.RequestsPerSec,
		BurstSize:     req.BurstSize,
		Adaptive:      req.Adaptive,
		BlockDuration: time.Duration(req.BlockDurationSecs) * time.Second,
	}

	ddosEnhancedService.SetRateLimitStrategy(strategy)

	c.JSON(http.StatusOK, gin.H{
		"message": "Rate limit strategy updated",
		"strategy": gin.H{
			"name":             strategy.Name,
			"requests_per_sec": strategy.RequestsPerSec,
			"burst_size":       strategy.BurstSize,
			"adaptive":         strategy.Adaptive,
			"block_duration":   strategy.BlockDuration.String(),
		},
	})
}

func GetDDoSEnhancedService() *service.DDoSEnhancedProtectionService {
	initDDoSEnhanced()
	return ddosEnhancedService
}

func formatRetryAfter(seconds int) string {
	return (time.Duration(seconds) * time.Second).String()
}

func formatInt(n int) string {
	return string(rune(n + '0'))
}
