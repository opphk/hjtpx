package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type AdvancedRateLimitHandler struct {
	adaptiveService     *service.AdaptiveRateLimitService
	distributedService *service.DistributedRateLimitService
	tokenBucketService *service.TokenBucketRateLimitService
	smartRateService   *service.SmartRateLimitService
}

func NewAdvancedRateLimitHandler() *AdvancedRateLimitHandler {
	return &AdvancedRateLimitHandler{
		adaptiveService:     service.NewAdaptiveRateLimitService(),
		distributedService: service.NewDistributedRateLimitService(),
		tokenBucketService: service.NewTokenBucketRateLimitService(),
		smartRateService:   service.NewSmartRateLimitService(),
	}
}

type AdaptiveRateLimitConfigRequest struct {
	BaseLimit      int64 `json:"base_limit"`
	PeakLimit      int64 `json:"peak_limit"`
	OffPeakLimit   int64 `json:"off_peak_limit"`
	OffPeakStart   int   `json:"off_peak_start"`
	OffPeakEnd     int   `json:"off_peak_end"`
	EnableDynamic  bool  `json:"enable_dynamic"`
	CooldownPeriod int   `json:"cooldown_period"`
}

func (h *AdvancedRateLimitHandler) GetAdaptiveConfig(c *gin.Context) {
	stats := h.adaptiveService.GetStats()
	
	config := map[string]interface{}{
		"base_rate":                stats["base_rate"],
		"base_capacity":            stats["base_capacity"],
		"load_level":               stats["load_level"],
		"load_factor":              stats["load_factor"],
		"current_load":             stats["current_load"],
		"bucket_count":             stats["bucket_count"],
		"total_tokens":             stats["total_tokens"],
	}

	response.Success(c, config)
}

func (h *AdvancedRateLimitHandler) UpdateAdaptiveConfig(c *gin.Context) {
	var req AdaptiveRateLimitConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	config := service.AdaptiveRateLimitConfig{
		BaseLimit:      req.BaseLimit,
		PeakLimit:      req.PeakLimit,
		OffPeakLimit:   req.OffPeakLimit,
		OffPeakStart:   req.OffPeakStart,
		OffPeakEnd:     req.OffPeakEnd,
		EnableDynamic:  req.EnableDynamic,
		CooldownPeriod: time.Duration(req.CooldownPeriod) * time.Second,
	}

	h.adaptiveService.UpdateConfig(config)
	response.Success(c, gin.H{"message": "adaptive config updated"})
}

func (h *AdvancedRateLimitHandler) GetAdaptiveStats(c *gin.Context) {
	stats := h.adaptiveService.GetStats()
	response.Success(c, stats)
}

type DistributedRateLimitConfigRequest struct {
	RedisEnabled     bool   `json:"redis_enabled"`
	ConsistencyLevel string `json:"consistency_level"`
	Nodes            []string `json:"nodes"`
	SyncInterval     int    `json:"sync_interval"`
}

func (h *AdvancedRateLimitHandler) GetDistributedConfig(c *gin.Context) {
	stats := h.distributedService.GetStats()
	
	config := map[string]interface{}{
		"type":              stats["type"],
		"max_requests":      stats["max_requests"],
		"window_secs":       stats["window_secs"],
		"node_id":           stats["node_id"],
		"redis_enabled":     stats["redis_enabled"],
		"sync_interval":     stats["sync_interval"],
		"consistency_mode":  stats["consistency_mode"],
	}

	response.Success(c, config)
}

func (h *AdvancedRateLimitHandler) UpdateDistributedConfig(c *gin.Context) {
	var req DistributedRateLimitConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	syncInterval := 5 * time.Second
	if req.SyncInterval > 0 {
		syncInterval = time.Duration(req.SyncInterval) * time.Second
	}

	config := service.DistributedRateLimitConfig{
		RedisEnabled:     req.RedisEnabled,
		ConsistencyLevel: req.ConsistencyLevel,
		Nodes:            req.Nodes,
		SyncInterval:    syncInterval,
	}

	h.distributedService.UpdateConfig(config)
	response.Success(c, gin.H{"message": "distributed config updated"})
}

func (h *AdvancedRateLimitHandler) GetDistributedStats(c *gin.Context) {
	stats := h.distributedService.GetStats()
	response.Success(c, stats)
}

func (h *AdvancedRateLimitHandler) ResetDistributedKey(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		response.BadRequest(c, "key is required")
		return
	}

	err := h.distributedService.ResetKey(c.Request.Context(), key)
	if err != nil {
		response.InternalServerError(c, "Failed to reset key")
		return
	}

	response.Success(c, gin.H{"message": "key reset successfully"})
}

type TokenBucketConfigRequest struct {
	RefillRate   float64 `json:"refill_rate"`
	Capacity     int64   `json:"capacity"`
	RefillPerSec float64 `json:"refill_per_sec"`
}

func (h *AdvancedRateLimitHandler) GetTokenBucketStats(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		stats := h.tokenBucketService.GetGlobalStats()
		bucketList := h.tokenBucketService.GetBucketList()
		response.Success(c, gin.H{
			"global_stats": stats,
			"buckets":      bucketList,
		})
		return
	}

	stats := h.tokenBucketService.GetBucketStats(key)
	response.Success(c, gin.H{"key": key, "stats": stats})
}

type TokenBucketUpdateRequest struct {
	Key         string  `json:"key" binding:"required"`
	RefillRate  float64 `json:"refill_rate"`
	Capacity    int64   `json:"capacity"`
	RefillPerSec float64 `json:"refill_per_sec"`
}

func (h *AdvancedRateLimitHandler) UpdateTokenBucketConfig(c *gin.Context) {
	var req TokenBucketUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	config := &service.TokenBucketConfig{
		RefillRate:   req.RefillRate,
		Capacity:     req.Capacity,
		RefillPerSec: req.RefillPerSec,
	}

	if err := h.tokenBucketService.UpdateBucketConfig(req.Key, config); err != nil {
		response.InternalServerError(c, "Failed to update bucket config: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "bucket config updated successfully"})
}

func (h *AdvancedRateLimitHandler) GetTokenBucketList(c *gin.Context) {
	bucketList := h.tokenBucketService.GetBucketList()
	response.Success(c, gin.H{
		"count":   len(bucketList),
		"buckets": bucketList,
	})
}

func (h *AdvancedRateLimitHandler) ResetTokenBucket(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		response.BadRequest(c, "key is required")
		return
	}

	err := h.tokenBucketService.ResetBucket(c.Request.Context(), key)
	if err != nil {
		response.InternalServerError(c, "Failed to reset bucket")
		return
	}

	response.Success(c, gin.H{"message": "bucket reset successfully"})
}

type SmartRateLimitConfigRequest struct {
	DefaultRequestsPerMin int      `json:"default_requests_per_min"`
	DefaultBurstLimit    int      `json:"default_burst_limit"`
	EnableAdaptiveLimit  bool     `json:"enable_adaptive_limit"`
	EnableRiskBasedLimit bool     `json:"enable_risk_based_limit"`
	Tiers                []string `json:"tiers"`
}

func (h *AdvancedRateLimitHandler) GetSmartRateLimitStats(c *gin.Context) {
	stats := h.smartRateService.GetStats()
	response.Success(c, stats)
}

func (h *AdvancedRateLimitHandler) GetClientStats(c *gin.Context) {
	clientID := c.Query("client_id")
	if clientID == "" {
		response.BadRequest(c, "client_id is required")
		return
	}

	stats := h.smartRateService.GetClientStats(clientID)
	if stats == nil {
		response.NotFound(c, "client not found")
		return
	}

	response.Success(c, stats)
}

func (h *AdvancedRateLimitHandler) SetClientTier(c *gin.Context) {
	var req struct {
		ClientID string `json:"client_id" binding:"required"`
		Tier     string `json:"tier" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	h.smartRateService.SetClientTier(req.ClientID, req.Tier)
	response.Success(c, gin.H{"message": "client tier updated"})
}

type RateLimitCheckRequest struct {
	Type  string `json:"type" binding:"required"`
	Key   string `json:"key" binding:"required"`
	Count int    `json:"count"`
}

func (h *AdvancedRateLimitHandler) CheckRateLimit(c *gin.Context) {
	var req RateLimitCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	if req.Count <= 0 {
		req.Count = 1
	}

	ctx := c.Request.Context()
	var result interface{}
	var err error

	switch req.Type {
	case "adaptive":
		var adaptiveResult *service.AdvancedRateLimitResult
		adaptiveResult, err = h.adaptiveService.CheckRateLimitWithTokens(ctx, req.Key, float64(req.Count))
		if adaptiveResult != nil {
			result = gin.H{
				"allowed":     adaptiveResult.Allowed,
				"remaining":   adaptiveResult.Remaining,
				"reset_at":    adaptiveResult.ResetAt,
				"retry_after": adaptiveResult.RetryAfter,
			}
		}
	case "distributed":
		var distResult *service.DistributedRateLimitResult
		distResult, err = h.distributedService.CheckRateLimitWithCount(ctx, req.Key, req.Count)
		if distResult != nil {
			result = gin.H{
				"allowed":     distResult.Allowed,
				"remaining":   distResult.Remaining,
				"reset_at":    distResult.ResetAt,
				"total_count": distResult.TotalCount,
				"node_id":     distResult.NodeID,
			}
		}
	case "token_bucket":
		var tbResult *service.AdvancedRateLimitResult
		tbResult, err = h.tokenBucketService.CheckTokenBucketRateLimit(ctx, req.Key, &service.TokenBucketConfig{
			Capacity:   100,
			RefillRate: 10,
		})
		if tbResult != nil {
			result = gin.H{
				"allowed":     tbResult.Allowed,
				"remaining":   tbResult.Remaining,
				"reset_at":    tbResult.ResetAt,
			}
		}
	default:
		response.BadRequest(c, "invalid type, must be: adaptive, distributed, or token_bucket")
		return
	}

	if err != nil {
		response.InternalServerError(c, "Failed to check rate limit")
		return
	}

	response.Success(c, result)
}

type OverallRateLimitStatus struct {
	Adaptive     map[string]interface{} `json:"adaptive"`
	Distributed  map[string]interface{} `json:"distributed"`
	Smart        map[string]interface{} `json:"smart"`
	LoadLevel    string                 `json:"load_level"`
}

func (h *AdvancedRateLimitHandler) GetOverallStatus(c *gin.Context) {
	status := OverallRateLimitStatus{
		Adaptive:    h.adaptiveService.GetStats(),
		Distributed: h.distributedService.GetStats(),
		Smart:       h.smartRateService.GetStats(),
		LoadLevel:   h.adaptiveService.GetLoadLevel().String(),
	}

	response.Success(c, status)
}

func (h *AdvancedRateLimitHandler) RegisterRoutes(r *gin.RouterGroup) {
	rateLimit := r.Group("/rate-limit")
	{
		rateLimit.GET("/status", h.GetOverallStatus)
		rateLimit.POST("/check", h.CheckRateLimit)

		rateLimit.GET("/adaptive/config", h.GetAdaptiveConfig)
		rateLimit.PUT("/adaptive/config", h.UpdateAdaptiveConfig)
		rateLimit.GET("/adaptive/stats", h.GetAdaptiveStats)

		rateLimit.GET("/distributed/config", h.GetDistributedConfig)
		rateLimit.PUT("/distributed/config", h.UpdateDistributedConfig)
		rateLimit.GET("/distributed/stats", h.GetDistributedStats)
		rateLimit.POST("/distributed/reset", h.ResetDistributedKey)

		rateLimit.GET("/token-bucket/stats", h.GetTokenBucketStats)
		rateLimit.GET("/token-bucket/list", h.GetTokenBucketList)
		rateLimit.POST("/token-bucket/update", h.UpdateTokenBucketConfig)
		rateLimit.POST("/token-bucket/reset", h.ResetTokenBucket)

		rateLimit.GET("/smart/stats", h.GetSmartRateLimitStats)
		rateLimit.GET("/smart/client", h.GetClientStats)
		rateLimit.PUT("/smart/client/tier", h.SetClientTier)
	}
}

func GetAdvancedRateLimitHandler() *AdvancedRateLimitHandler {
	return NewAdvancedRateLimitHandler()
}
