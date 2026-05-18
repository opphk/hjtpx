package handler

import (
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
	BaseRate              float64 `json:"base_rate"`
	BaseCapacity          float64 `json:"base_capacity"`
	MinCapacity           float64 `json:"min_capacity"`
	MaxCapacity           float64 `json:"max_capacity"`
	LoadCheckInterval     int     `json:"load_check_interval"`
	AdjustmentInterval    int     `json:"adjustment_interval"`
	HighLoadThreshold     float64 `json:"high_load_threshold"`
	CriticalLoadThreshold float64 `json:"critical_load_threshold"`
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
		BaseRate:               req.BaseRate,
		BaseCapacity:           req.BaseCapacity,
		MinCapacity:            req.MinCapacity,
		MaxCapacity:           req.MaxCapacity,
		HighLoadThreshold:      req.HighLoadThreshold,
		CriticalLoadThreshold: req.CriticalLoadThreshold,
	}

	if req.LoadCheckInterval > 0 {
		config.LoadCheckInterval = service.ParseDuration(req.LoadCheckInterval)
	}
	if req.AdjustmentInterval > 0 {
		config.AdjustmentInterval = service.ParseDuration(req.AdjustmentInterval)
	}

	h.adaptiveService.UpdateConfig(config)
	response.Success(c, gin.H{"message": "adaptive config updated"})
}

func (h *AdvancedRateLimitHandler) GetAdaptiveStats(c *gin.Context) {
	stats := h.adaptiveService.GetStats()
	response.Success(c, stats)
}

type DistributedRateLimitConfigRequest struct {
	Type            string `json:"type"`
	MaxRequests     int    `json:"max_requests"`
	WindowSecs      int    `json:"window_secs"`
	SyncInterval    int    `json:"sync_interval"`
	ConsistencyMode bool   `json:"consistency_mode"`
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

	var limitType service.DistributedRateLimitType
	switch req.Type {
	case "fixed_window":
		limitType = service.DistributedFixedWindow
	case "sliding_window":
		limitType = service.DistributedSlidingWindow
	case "token_bucket":
		limitType = service.DistributedTokenBucket
	case "leaky_bucket":
		limitType = service.DistributedLeakyBucket
	default:
		limitType = service.DistributedTokenBucket
	}

	syncInterval := 5
	if req.SyncInterval > 0 {
		syncInterval = req.SyncInterval
	}

	config := service.DistributedRateLimitConfig{
		Type:            limitType,
		MaxRequests:     req.MaxRequests,
		WindowSecs:      req.WindowSecs,
		SyncInterval:    service.ParseDuration(syncInterval),
		ConsistencyMode: req.ConsistencyMode,
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
	Rate          float64 `json:"rate"`
	Capacity      float64 `json:"capacity"`
	BurstSize     float64 `json:"burst_size"`
	InitialTokens float64 `json:"initial_tokens"`
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
	Key      string  `json:"key" binding:"required"`
	Rate     float64 `json:"rate"`
	Capacity float64 `json:"capacity"`
	BurstSize float64 `json:"burst_size"`
}

func (h *AdvancedRateLimitHandler) UpdateTokenBucketConfig(c *gin.Context) {
	var req TokenBucketUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	config := &service.TokenBucketConfig{
		Rate:     req.Rate,
		Capacity: req.Capacity,
		BurstSize: req.BurstSize,
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
		var adaptiveResult *service.AdaptiveRateLimitResult
		adaptiveResult, err = h.adaptiveService.CheckRateLimitWithTokens(ctx, req.Key, float64(req.Count))
		if adaptiveResult != nil {
			result = adaptiveResult
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
		var tbResult *service.TokenBucketResult
		tbResult, err = h.tokenBucketService.CheckTokenBucketRateLimit(ctx, req.Key, &service.TokenBucketConfig{
			Rate:     10,
			Capacity: 100,
		})
		if tbResult != nil {
			result = tbResult
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
