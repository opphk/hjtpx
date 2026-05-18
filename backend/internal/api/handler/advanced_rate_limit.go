package handler

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

var (
	tokenBucketService     *service.TokenBucketRateLimitService
	quotaManagementService *service.QuotaManagementService
	advancedRateLimitOnce  sync.Once
)

func initAdvancedRateLimitServices() {
	tokenBucketService = service.NewTokenBucketRateLimitService()
	quotaManagementService = service.NewQuotaManagementService()
}

// AdvancedRateLimitHandler 高级限流处理器结构
type AdvancedRateLimitHandler struct{}

// NewAdvancedRateLimitHandler 创建新的高级限流处理器
func NewAdvancedRateLimitHandler() *AdvancedRateLimitHandler {
	advancedRateLimitOnce.Do(initAdvancedRateLimitServices)
	return &AdvancedRateLimitHandler{}
}

// ---------------------------- 令牌桶限流相关 API ----------------------------

// CheckTokenBucketRequest 检查令牌桶请求
type CheckTokenBucketRequest struct {
	Key    string                  `json:"key" binding:"required"`
	Config *service.TokenBucketConfig `json:"config"`
}

// CheckTokenBucketResponse 检查令牌桶响应
type CheckTokenBucketResponse struct {
	Allowed    bool          `json:"allowed"`
	Tokens     float64       `json:"tokens"`
	Capacity   float64       `json:"capacity"`
	RetryAfter float64       `json:"retry_after_seconds"`
	WaitTime   float64       `json:"wait_time_seconds"`
	IsBurst    bool          `json:"is_burst"`
}

// CheckTokenBucket 检查令牌桶限流
// @Summary 检查令牌桶限流
// @Description 检查指定键的令牌桶限流状态
// @Tags 高级限流
// @Accept json
// @Produce json
// @Param request body CheckTokenBucketRequest true "请求参数"
// @Success 200 {object} CheckTokenBucketResponse
// @Router /api/v1/advanced-rate-limit/token-bucket/check [post]
func (h *AdvancedRateLimitHandler) CheckTokenBucket(c *gin.Context) {
	var req CheckTokenBucketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := tokenBucketService.CheckTokenBucketRateLimitRedis(c.Request.Context(), req.Key, req.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, CheckTokenBucketResponse{
		Allowed:    result.Allowed,
		Tokens:     result.Tokens,
		Capacity:   result.Capacity,
		RetryAfter: result.RetryAfter.Seconds(),
		WaitTime:   result.WaitTime.Seconds(),
		IsBurst:    result.IsBurst,
	})
}

// ResetTokenBucketRequest 重置令牌桶请求
type ResetTokenBucketRequest struct {
	Key string `json:"key" binding:"required"`
}

// ResetTokenBucket 重置令牌桶
// @Summary 重置令牌桶
// @Description 重置指定键的令牌桶
// @Tags 高级限流
// @Accept json
// @Produce json
// @Param request body ResetTokenBucketRequest true "请求参数"
// @Success 200 {object} map[string]string
// @Router /api/v1/advanced-rate-limit/token-bucket/reset [post]
func (h *AdvancedRateLimitHandler) ResetTokenBucket(c *gin.Context) {
	var req ResetTokenBucketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := tokenBucketService.ResetBucket(c.Request.Context(), req.Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "令牌桶已重置"})
}

// GetBucketStatsResponse 获取桶统计响应
type GetBucketStatsResponse struct {
	Key         string    `json:"key"`
	Tokens      float64   `json:"tokens"`
	Capacity    float64   `json:"capacity"`
	Rate        float64   `json:"rate"`
	BurstSize   float64   `json:"burst_size"`
	LastRefill  string    `json:"last_refill"`
}

// GetBucketStats 获取桶统计
// @Summary 获取令牌桶统计
// @Description 获取指定键的令牌桶统计信息
// @Tags 高级限流
// @Accept json
// @Produce json
// @Param key query string true "令牌桶键"
// @Success 200 {object} GetBucketStatsResponse
// @Router /api/v1/advanced-rate-limit/token-bucket/stats [get]
func (h *AdvancedRateLimitHandler) GetBucketStats(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key不能为空"})
		return
	}

	stats := tokenBucketService.GetBucketStats(key)
	if stats == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "令牌桶不存在"})
		return
	}

	c.JSON(http.StatusOK, GetBucketStatsResponse{
		Key:         stats["key"].(string),
		Tokens:      stats["tokens"].(float64),
		Capacity:    stats["capacity"].(float64),
		Rate:        stats["rate"].(float64),
		BurstSize:   stats["burst_size"].(float64),
		LastRefill:  stats["last_refill"].(string),
	})
}

// ---------------------------- 配额管理相关 API ----------------------------

// CreateQuotaRequest 创建配额请求
type CreateQuotaRequest struct {
	Key      string            `json:"key" binding:"required"`
	Config   *service.QuotaConfig `json:"config" binding:"required"`
}

// CreateQuota 创建配额
// @Summary 创建配额
// @Description 创建新的配额
// @Tags 高级限流-配额
// @Accept json
// @Produce json
// @Param request body CreateQuotaRequest true "请求参数"
// @Success 200 {object} map[string]string
// @Router /api/v1/advanced-rate-limit/quota/create [post]
func (h *AdvancedRateLimitHandler) CreateQuota(c *gin.Context) {
	var req CreateQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := quotaManagementService.CreateOrUpdateQuota(c.Request.Context(), req.Key, req.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "配额创建成功"})
}

// GetQuotaStatusResponse 获取配额状态响应
type GetQuotaStatusResponse struct {
	Used       int64   `json:"used"`
	Limit      int64   `json:"limit"`
	Remaining  int64   `json:"remaining"`
	ResetAt    string  `json:"reset_at"`
	Type       string  `json:"type"`
	Percentage float64 `json:"percentage"`
}

// GetQuotaStatus 获取配额状态
// @Summary 获取配额状态
// @Description 获取指定键的配额状态
// @Tags 高级限流-配额
// @Accept json
// @Produce json
// @Param key query string true "配额键"
// @Success 200 {object} GetQuotaStatusResponse
// @Router /api/v1/advanced-rate-limit/quota/status [get]
func (h *AdvancedRateLimitHandler) GetQuotaStatus(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key不能为空"})
		return
	}

	status, err := quotaManagementService.GetQuotaStatus(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, GetQuotaStatusResponse{
		Used:       status.Used,
		Limit:      status.Limit,
		Remaining:  status.Remaining,
		ResetAt:    status.ResetAt.Format("2006-01-02 15:04:05"),
		Type:       string(status.Type),
		Percentage: status.Percentage,
	})
}

// ConsumeQuotaRequest 消费配额请求
type ConsumeQuotaRequest struct {
	Key    string `json:"key" binding:"required"`
	Amount int64  `json:"amount"`
}

// ConsumeQuotaResponse 消费配额响应
type ConsumeQuotaResponse struct {
	Allowed bool                 `json:"allowed"`
	Status  *GetQuotaStatusResponse `json:"status"`
}

// ConsumeQuota 消费配额
// @Summary 消费配额
// @Description 消费指定键的配额
// @Tags 高级限流-配额
// @Accept json
// @Produce json
// @Param request body ConsumeQuotaRequest true "请求参数"
// @Success 200 {object} ConsumeQuotaResponse
// @Router /api/v1/advanced-rate-limit/quota/consume [post]
func (h *AdvancedRateLimitHandler) ConsumeQuota(c *gin.Context) {
	var req ConsumeQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Amount <= 0 {
		req.Amount = 1
	}

	status, allowed, err := quotaManagementService.ConsumeQuota(c.Request.Context(), req.Key, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	statusResp := GetQuotaStatusResponse{
		Used:       status.Used,
		Limit:      status.Limit,
		Remaining:  status.Remaining,
		ResetAt:    status.ResetAt.Format("2006-01-02 15:04:05"),
		Type:       string(status.Type),
		Percentage: status.Percentage,
	}

	c.JSON(http.StatusOK, ConsumeQuotaResponse{
		Allowed: allowed,
		Status:  &statusResp,
	})
}

// ResetQuotaRequest 重置配额请求
type ResetQuotaRequest struct {
	Key string `json:"key" binding:"required"`
}

// ResetQuota 重置配额
// @Summary 重置配额
// @Description 重置指定键的配额
// @Tags 高级限流-配额
// @Accept json
// @Produce json
// @Param request body ResetQuotaRequest true "请求参数"
// @Success 200 {object} map[string]string
// @Router /api/v1/advanced-rate-limit/quota/reset [post]
func (h *AdvancedRateLimitHandler) ResetQuota(c *gin.Context) {
	var req ResetQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := quotaManagementService.ResetQuota(c.Request.Context(), req.Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "配额已重置"})
}

// DeleteQuotaRequest 删除配额请求
type DeleteQuotaRequest struct {
	Key string `json:"key" binding:"required"`
}

// DeleteQuota 删除配额
// @Summary 删除配额
// @Description 删除指定键的配额
// @Tags 高级限流-配额
// @Accept json
// @Produce json
// @Param request body DeleteQuotaRequest true "请求参数"
// @Success 200 {object} map[string]string
// @Router /api/v1/advanced-rate-limit/quota/delete [delete]
func (h *AdvancedRateLimitHandler) DeleteQuota(c *gin.Context) {
	var req DeleteQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := quotaManagementService.DeleteQuota(c.Request.Context(), req.Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "配额已删除"})
}

// ListQuotasResponse 列出配额响应
type ListQuotasResponse struct {
	Quotas []*QuotaInfo `json:"quotas"`
}

// QuotaInfo 配额信息
type QuotaInfo struct {
	Key          string  `json:"key"`
	Type         string  `json:"type"`
	Limit        int64   `json:"limit"`
	Used         int64   `json:"used"`
	Remaining    int64   `json:"remaining"`
	ResetAt      string  `json:"reset_at"`
	Percentage   float64 `json:"percentage"`
}

// ListQuotas 列出所有配额
// @Summary 列出所有配额
// @Description 列出所有配额
// @Tags 高级限流-配额
// @Accept json
// @Produce json
// @Success 200 {object} ListQuotasResponse
// @Router /api/v1/advanced-rate-limit/quota/list [get]
func (h *AdvancedRateLimitHandler) ListQuotas(c *gin.Context) {
	quotas, err := quotaManagementService.ListQuotas(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	quotaInfos := make([]*QuotaInfo, 0, len(quotas))
	for _, q := range quotas {
		remaining := q.Limit - q.Used
		if remaining < 0 {
			remaining = 0
		}
		percentage := 0.0
		if q.Limit > 0 {
			percentage = (float64(q.Used) / float64(q.Limit)) * 100
		}
		quotaInfos = append(quotaInfos, &QuotaInfo{
			Key:        q.Key,
			Type:       string(q.Type),
			Limit:      q.Limit,
			Used:       q.Used,
			Remaining:  remaining,
			ResetAt:    q.ResetAt.Format("2006-01-02 15:04:05"),
			Percentage: percentage,
		})
	}

	c.JSON(http.StatusOK, ListQuotasResponse{Quotas: quotaInfos})
}

// ---------------------------- 综合限流 API ----------------------------

// CombinedCheckRequest 综合检查请求
type CombinedCheckRequest struct {
	IP           string                  `json:"ip"`
	UserID       uint                    `json:"user_id"`
	AppID        uint                    `json:"app_id"`
	TokenBucketKey string                 `json:"token_bucket_key"`
	QuotaKey     string                  `json:"quota_key"`
	TokenBucketConfig *service.TokenBucketConfig `json:"token_bucket_config"`
}

// CombinedCheckResponse 综合检查响应
type CombinedCheckResponse struct {
	Allowed           bool                  `json:"allowed"`
	TokenBucketResult *CheckTokenBucketResponse `json:"token_bucket_result"`
	QuotaResult       *ConsumeQuotaResponse  `json:"quota_result"`
	Reason            string                `json:"reason"`
}

// CombinedCheck 综合限流检查
// @Summary 综合限流检查
// @Description 同时检查令牌桶和配额
// @Tags 高级限流
// @Accept json
// @Produce json
// @Param request body CombinedCheckRequest true "请求参数"
// @Success 200 {object} CombinedCheckResponse
// @Router /api/v1/advanced-rate-limit/combined-check [post]
func (h *AdvancedRateLimitHandler) CombinedCheck(c *gin.Context) {
	var req CombinedCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	allowed := true
	var reason string
	var tokenBucketResult *CheckTokenBucketResponse
	var quotaResult *ConsumeQuotaResponse

	// 检查令牌桶
	if req.TokenBucketKey != "" {
		var tbKey string
		if req.IP != "" {
			tbKey = "ip:" + req.IP
		} else if req.UserID > 0 {
			tbKey = "user:" + strconv.FormatUint(uint64(req.UserID), 10)
		} else if req.AppID > 0 {
			tbKey = "app:" + strconv.FormatUint(uint64(req.AppID), 10)
		} else {
			tbKey = req.TokenBucketKey
		}

		tbResult, err := tokenBucketService.CheckTokenBucketRateLimitRedis(c.Request.Context(), tbKey, req.TokenBucketConfig)
		if err == nil {
			tokenBucketResult = &CheckTokenBucketResponse{
				Allowed:    tbResult.Allowed,
				Tokens:     tbResult.Tokens,
				Capacity:   tbResult.Capacity,
				RetryAfter: tbResult.RetryAfter.Seconds(),
				WaitTime:   tbResult.WaitTime.Seconds(),
				IsBurst:    tbResult.IsBurst,
			}
			if !tbResult.Allowed {
				allowed = false
				reason = "令牌桶限流"
			}
		}
	}

	// 检查配额
	if allowed && req.QuotaKey != "" {
		var qKey string
		if req.UserID > 0 {
			qKey = service.UserQuotaKey(req.UserID, "api", service.QuotaTypeDaily)
		} else if req.AppID > 0 {
			qKey = service.AppQuotaKey(req.AppID, "api", service.QuotaTypeDaily)
		} else {
			qKey = req.QuotaKey
		}

		status, qAllowed, err := quotaManagementService.ConsumeQuota(c.Request.Context(), qKey, 1)
		if err == nil {
			statusResp := GetQuotaStatusResponse{
				Used:       status.Used,
				Limit:      status.Limit,
				Remaining:  status.Remaining,
				ResetAt:    status.ResetAt.Format("2006-01-02 15:04:05"),
				Type:       string(status.Type),
				Percentage: status.Percentage,
			}
			quotaResult = &ConsumeQuotaResponse{
				Allowed: qAllowed,
				Status:  &statusResp,
			}
			if !qAllowed {
				allowed = false
				reason = "配额不足"
			}
		}
	}

	c.JSON(http.StatusOK, CombinedCheckResponse{
		Allowed:           allowed,
		TokenBucketResult: tokenBucketResult,
		QuotaResult:       quotaResult,
		Reason:            reason,
	})
}
