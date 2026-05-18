package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/internal/service"
)

var riskScoringService *service.RiskScoringService

func init() {
	riskScoringService = service.NewRiskScoringService()
}

func GetRiskScoringService() *service.RiskScoringService {
	return riskScoringService
}

type RiskScoringHandler struct {
	service *service.RiskScoringService
}

func NewRiskScoringHandler() *RiskScoringHandler {
	return &RiskScoringHandler{
		service: service.NewRiskScoringService(),
	}
}

func GetRiskScoringHandler() *RiskScoringHandler {
	return NewRiskScoringHandler()
}

func (h *RiskScoringHandler) GetConfig(c *gin.Context) {
	config := h.service.GetConfig()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": config,
	})
}

func (h *RiskScoringHandler) UpdateConfig(c *gin.Context) {
	var config model.RiskScoringConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "invalid config: " + err.Error(),
		})
		return
	}

	if err := h.service.UpdateConfig(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "config updated successfully",
	})
}

type UpdateWeightsRequest struct {
	TraceWeight    float64 `json:"trace_weight"`
	EnvWeight      float64 `json:"env_weight"`
	BehaviorWeight float64 `json:"behavior_weight"`
	DeviceWeight   float64 `json:"device_weight"`
	HistoryWeight  float64 `json:"history_weight"`
}

func (h *RiskScoringHandler) UpdateWeights(c *gin.Context) {
	var req UpdateWeightsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	weights := &model.RiskScoringWeights{
		TraceWeight:    req.TraceWeight,
		EnvWeight:      req.EnvWeight,
		BehaviorWeight: req.BehaviorWeight,
		DeviceWeight:   req.DeviceWeight,
		HistoryWeight:  req.HistoryWeight,
	}

	if err := h.service.UpdateWeights(weights); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "weights updated successfully",
		"data":    weights,
	})
}

func (h *RiskScoringHandler) GetWeights(c *gin.Context) {
	weights := h.service.GetWeights()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": weights,
	})
}

type UpdateThresholdsRequest struct {
	LowMax      float64 `json:"low_max"`
	MediumMax   float64 `json:"medium_max"`
	HighMax     float64 `json:"high_max"`
	CriticalMax float64 `json:"critical_max"`
	VerifyMin   float64 `json:"verify_min"`
	BlockMin    float64 `json:"block_min"`
}

func (h *RiskScoringHandler) UpdateThresholds(c *gin.Context) {
	var req UpdateThresholdsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	thresholds := &model.RiskThresholds{
		LowMax:      req.LowMax,
		MediumMax:   req.MediumMax,
		HighMax:     req.HighMax,
		CriticalMax: req.CriticalMax,
		VerifyMin:   req.VerifyMin,
		BlockMin:    req.BlockMin,
	}

	if err := h.service.UpdateThresholds(thresholds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "thresholds updated successfully",
		"data":    thresholds,
	})
}

func (h *RiskScoringHandler) GetThresholds(c *gin.Context) {
	thresholds := h.service.GetThresholds()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": thresholds,
	})
}

type EvaluateRequest struct {
	SessionID         string            `json:"session_id"`
	IPAddress         string            `json:"ip_address"`
	UserAgent         string            `json:"user_agent"`
	Fingerprint       string            `json:"fingerprint"`
	DeviceInfo        map[string]string `json:"device_info"`
	PositionDiff      int               `json:"position_diff"`
	TraceData         []model.TracePoint `json:"trace_data"`
	VerificationCount int               `json:"verification_count"`
	FailureCount      int               `json:"failure_count"`
	TimeFromStart     int64             `json:"time_from_start"`
	MouseSpeed        float64           `json:"mouse_speed"`
	HasTouchDevice    bool              `json:"has_touch_device"`
	BrowserPlugins    []string          `json:"browser_plugins"`
	Language          string            `json:"language"`
	Timezone          string            `json:"timezone"`
	ScreenRes         string            `json:"screen_res"`
	Referer           string            `json:"referer"`
	IsProxy           bool              `json:"is_proxy"`
	IsVPN             bool              `json:"is_vpn"`
	IsTor             bool              `json:"is_tor"`
	IsHosting         bool              `json:"is_hosting"`
	IPReputation      string            `json:"ip_reputation"`
}

func (h *RiskScoringHandler) Evaluate(c *gin.Context) {
	var req EvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	ctx := &model.RiskContext{
		SessionID:         req.SessionID,
		IPAddress:         req.IPAddress,
		UserAgent:         req.UserAgent,
		Fingerprint:       req.Fingerprint,
		DeviceInfo:        req.DeviceInfo,
		PositionDiff:      req.PositionDiff,
		TraceData:         req.TraceData,
		VerificationCount: req.VerificationCount,
		FailureCount:      req.FailureCount,
		TimeFromStart:     req.TimeFromStart,
		MouseSpeed:        req.MouseSpeed,
		HasTouchDevice:    req.HasTouchDevice,
		BrowserPlugins:    req.BrowserPlugins,
		Language:          req.Language,
		Timezone:          req.Timezone,
		ScreenRes:         req.ScreenRes,
		Referer:           req.Referer,
		IsProxy:           req.IsProxy,
		IsVPN:             req.IsVPN,
		IsTor:             req.IsTor,
		IsHosting:         req.IsHosting,
		IPReputation:      req.IPReputation,
	}

	result, err := h.service.EvaluateWithVerification(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}

func (h *RiskScoringHandler) GetScoreBreakdown(c *gin.Context) {
	var req EvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	ctx := &model.RiskContext{
		SessionID:         req.SessionID,
		IPAddress:         req.IPAddress,
		UserAgent:         req.UserAgent,
		Fingerprint:       req.Fingerprint,
		DeviceInfo:        req.DeviceInfo,
		PositionDiff:      req.PositionDiff,
		TraceData:         req.TraceData,
		VerificationCount: req.VerificationCount,
		FailureCount:      req.FailureCount,
		TimeFromStart:     req.TimeFromStart,
		MouseSpeed:        req.MouseSpeed,
		HasTouchDevice:    req.HasTouchDevice,
		BrowserPlugins:    req.BrowserPlugins,
		Language:          req.Language,
		Timezone:          req.Timezone,
		ScreenRes:         req.ScreenRes,
		Referer:           req.Referer,
		IsProxy:           req.IsProxy,
		IsVPN:             req.IsVPN,
		IsTor:             req.IsTor,
		IsHosting:         req.IsHosting,
		IPReputation:      req.IPReputation,
	}

	breakdown := h.service.GetScoreBreakdown(ctx)
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": breakdown,
	})
}

type RecordHistoryRequest struct {
	SessionID  string `json:"session_id"`
	Verified   bool   `json:"verified"`
	Success    bool   `json:"success"`
	RiskScore  float64 `json:"risk_score"`
}

func (h *RiskScoringHandler) RecordHistory(c *gin.Context) {
	var req RecordHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	ctx := &model.RiskContext{
		SessionID: req.SessionID,
	}

	score := &model.MultiDimensionalScore{
		TotalScore: req.RiskScore,
		TraceScore: 50,
		EnvScore:   50,
		RiskLevel:  model.DetermineRiskLevel(req.RiskScore),
	}

	var action string
	if req.RiskScore >= 80 {
		action = "block"
	} else if req.RiskScore >= 40 {
		action = "verify"
	} else {
		action = "allow"
	}

	if err := h.service.RecordHistory(ctx, score, action, req.Verified, req.Success); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "history recorded successfully",
	})
}

func (h *RiskScoringHandler) GetHistory(c *gin.Context) {
	sessionID := c.Query("session_id")

	history, err := h.service.GetHistory(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": history,
	})
}

func (h *RiskScoringHandler) GetDistribution(c *gin.Context) {
	distribution, err := h.service.GetDistribution()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": distribution,
	})
}

func (h *RiskScoringHandler) AdjustThresholds(c *gin.Context) {
	if err := h.service.AdjustThresholds(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "thresholds adjusted successfully",
		"data":    h.service.GetThresholds(),
	})
}

func (h *RiskScoringHandler) GetVisualization(c *gin.Context) {
	data, err := h.service.GetVisualizationData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": data,
	})
}

func (h *RiskScoringHandler) ExportConfig(c *gin.Context) {
	configJSON, err := h.service.ExportConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": map[string]string{
			"config": configJSON,
		},
	})
}

func (h *RiskScoringHandler) ImportConfig(c *gin.Context) {
	var req struct {
		Config string `json:"config" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	if err := h.service.ImportConfig(req.Config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "config imported successfully",
	})
}

func (h *RiskScoringHandler) ResetConfig(c *gin.Context) {
	h.service.ResetToDefault()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "config reset to default",
	})
}

func (h *RiskScoringHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": stats,
	})
}

func (h *RiskScoringHandler) GetScoreBands(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": model.DefaultScoreBands,
	})
}

func (h *RiskScoringHandler) RegisterRoutes(api *gin.RouterGroup) {
	scoring := api.Group("/scoring")
	{
		scoring.GET("/config", h.GetConfig)
		scoring.PUT("/config", h.UpdateConfig)
		scoring.GET("/weights", h.GetWeights)
		scoring.PUT("/weights", h.UpdateWeights)
		scoring.GET("/thresholds", h.GetThresholds)
		scoring.PUT("/thresholds", h.UpdateThresholds)
		scoring.POST("/evaluate", h.Evaluate)
		scoring.POST("/breakdown", h.GetScoreBreakdown)
		scoring.POST("/history", h.RecordHistory)
		scoring.GET("/history", h.GetHistory)
		scoring.GET("/distribution", h.GetDistribution)
		scoring.POST("/thresholds/adjust", h.AdjustThresholds)
		scoring.GET("/visualization", h.GetVisualization)
		scoring.GET("/export", h.ExportConfig)
		scoring.POST("/import", h.ImportConfig)
		scoring.POST("/reset", h.ResetConfig)
		scoring.GET("/stats", h.GetStats)
		scoring.GET("/bands", h.GetScoreBands)
	}
}

func RegisterRiskScoringRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	handler := NewRiskScoringHandler()
	handler.RegisterRoutes(api)
}

func GetRiskScoringConfig(c *gin.Context) {
	handler := GetRiskScoringHandler()
	handler.GetConfig(c)
}

func UpdateRiskScoringConfig(c *gin.Context) {
	handler := GetRiskScoringHandler()
	handler.UpdateConfig(c)
}

func GetRiskScoringWeights(c *gin.Context) {
	handler := GetRiskScoringHandler()
	handler.GetWeights(c)
}

func UpdateRiskScoringWeights(c *gin.Context) {
	handler := GetRiskScoringHandler()
	handler.UpdateWeights(c)
}

func GetRiskScoringThresholds(c *gin.Context) {
	handler := GetRiskScoringHandler()
	handler.GetThresholds(c)
}

func UpdateRiskScoringThresholds(c *gin.Context) {
	handler := GetRiskScoringHandler()
	handler.UpdateThresholds(c)
}

func EvaluateRisk(c *gin.Context) {
	handler := GetRiskScoringHandler()
	handler.Evaluate(c)
}

func GetRiskScoreBreakdown(c *gin.Context) {
	handler := GetRiskScoringHandler()
	handler.GetScoreBreakdown(c)
}

func RecordRiskScoringHistory(c *gin.Context) {
	handler := GetRiskScoringHandler()
	handler.RecordHistory(c)
}

func GetRiskScoringHistory(c *gin.Context) {
	sessionID := c.Query("session_id")
	handler := GetRiskScoringHandler()
	
	var history interface{}
	var err error
	if sessionID != "" {
		history, err = handler.service.GetHistory(sessionID)
	} else {
		history, err = handler.service.GetHistory("")
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": history,
	})
}

func GetRiskScoringDistribution(c *gin.Context) {
	handler := GetRiskScoringHandler()
	handler.GetDistribution(c)
}

func AdjustRiskScoringThresholds(c *gin.Context) {
	handler := GetRiskScoringHandler()
	handler.AdjustThresholds(c)
}

func GetRiskScoringVisualization(c *gin.Context) {
	handler := GetRiskScoringHandler()
	handler.GetVisualization(c)
}

func GetRiskScoringStats(c *gin.Context) {
	handler := GetRiskScoringHandler()
	handler.GetStats(c)
}

func (h *RiskScoringHandler) GetRiskScoringHistory(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "session_id is required",
		})
		return
	}

	history, err := h.service.GetHistory(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": history,
	})
}

func GetRiskScoringBySessionID(sessionID string) ([]*model.RiskScoringHistory, error) {
	handler := GetRiskScoringHandler()
	return handler.service.GetHistory(sessionID)
}

func GetRiskScoringDistributionData() (*model.RiskScoreDistribution, error) {
	handler := GetRiskScoringHandler()
	return handler.service.GetDistribution()
}

func GetRiskScoringVisualizationData() (map[string]interface{}, error) {
	handler := GetRiskScoringHandler()
	return handler.service.GetVisualizationData()
}

func RecordRiskScore(ctx *model.RiskContext, score *model.MultiDimensionalScore, action string, verified, success bool) error {
	handler := GetRiskScoringHandler()
	return handler.service.RecordHistory(ctx, score, action, verified, success)
}

func GetRiskScoringStatsData() (map[string]interface{}, error) {
	handler := GetRiskScoringHandler()
	return handler.service.GetStats()
}

func AdjustRiskThresholds() error {
	handler := GetRiskScoringHandler()
	return handler.service.AdjustThresholds()
}

func ResetRiskScoringConfig() {
	handler := GetRiskScoringHandler()
	handler.service.ResetToDefault()
}

func GetRiskScoringScoreBands() []model.ScoreBand {
	return model.DefaultScoreBands
}

func GetRiskScoringCurrentConfig() *model.RiskScoringConfig {
	handler := GetRiskScoringHandler()
	return handler.service.GetConfig()
}

func UpdateRiskScoringWeightsConfig(weights *model.RiskScoringWeights) error {
	handler := GetRiskScoringHandler()
	return handler.service.UpdateWeights(weights)
}

func UpdateRiskScoringThresholdsConfig(thresholds *model.RiskThresholds) error {
	handler := GetRiskScoringHandler()
	return handler.service.UpdateThresholds(thresholds)
}

func CalculateRiskScoreMultiDimensional(ctx *model.RiskContext) *model.MultiDimensionalScore {
	handler := GetRiskScoringHandler()
	return handler.service.CalculateScore(ctx)
}

func GetRiskScoreBreakdownData(ctx *model.RiskContext) map[string]interface{} {
	handler := GetRiskScoringHandler()
	return handler.service.GetScoreBreakdown(ctx)
}

func ExportRiskScoringConfigJSON() (string, error) {
	handler := GetRiskScoringHandler()
	return handler.service.ExportConfig()
}

func ImportRiskScoringConfigJSON(configJSON string) error {
	handler := GetRiskScoringHandler()
	return handler.service.ImportConfig(configJSON)
}

func GetRiskScoringServiceInstance() *service.RiskScoringService {
	return GetRiskScoringHandler().service
}

type RiskScoringHandlerWithParams struct {
	SessionID string `form:"session_id"`
	Limit    int    `form:"limit,default=100"`
	Offset   int    `form:"offset,default=0"`
}

func (h *RiskScoringHandler) GetHistoryList(c *gin.Context) {
	var params RiskScoringHandlerWithParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	sessionID := c.Query("session_id")
	limitStr := c.DefaultQuery("limit", "100")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	history, err := h.service.GetHistory(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	if offset >= len(history) {
		history = []*model.RiskScoringHistory{}
	} else {
		end := offset + limit
		if end > len(history) {
			end = len(history)
		}
		history = history[offset:end]
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": map[string]interface{}{
			"items": history,
			"total": len(history),
			"limit": limit,
			"offset": offset,
		},
	})
}

func GetRiskScoringHistoryList(c *gin.Context) {
	handler := GetRiskScoringHandler()
	handler.GetHistoryList(c)
}
