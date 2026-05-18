package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type RiskControlHandler struct {
	riskService *service.RiskControlService
}

func NewRiskControlHandler() *RiskControlHandler {
	return &RiskControlHandler{
		riskService: service.GetRiskControlService(),
	}
}

func GetRiskControlHandler() *RiskControlHandler {
	return NewRiskControlHandler()
}

type EvaluateRiskRequest struct {
	SessionID     string            `json:"session_id" binding:"required"`
	IPAddress     string            `json:"ip_address"`
	UserAgent     string            `json:"user_agent"`
	Fingerprint   string            `json:"fingerprint"`
	DeviceInfo    map[string]string `json:"device_info"`
	PositionDiff  int               `json:"position_diff"`
	TraceData     []model.TracePoint `json:"trace_data"`
	EnvInfo       *model.EnvInfo    `json:"env_info"`
	VerificationCount int            `json:"verification_count"`
	FailureCount  int               `json:"failure_count"`
	TimeFromStart int64             `json:"time_from_start"`
	MouseSpeed    float64           `json:"mouse_speed"`
	HasTouchDevice bool             `json:"has_touch_device"`
	BrowserPlugins []string         `json:"browser_plugins"`
	Language      string            `json:"language"`
	Timezone      string            `json:"timezone"`
	ScreenRes     string            `json:"screen_res"`
	Referer       string            `json:"referer"`
	IsProxy       bool              `json:"is_proxy"`
	IsVPN         bool              `json:"is_vpn"`
	IsTor         bool              `json:"is_tor"`
	IsHosting     bool              `json:"is_hosting"`
	IPReputation  string            `json:"ip_reputation"`
	Country       string            `json:"country"`
	ASNumber      int               `json:"as_number"`
}

type EvaluateRiskResponse struct {
	RiskScore       float64          `json:"risk_score"`
	RiskLevel       string           `json:"risk_level"`
	PositionScore   float64          `json:"position_score"`
	BehaviorScore   float64          `json:"behavior_score"`
	EnvScore        float64          `json:"env_score"`
	DeviceScore     float64          `json:"device_score"`
	IPScore         float64          `json:"ip_score"`
	Action          string           `json:"action"`
	RiskFactors     []string         `json:"risk_factors"`
	Details         map[string]float64 `json:"details"`
	RecommendVerify bool             `json:"recommend_verify"`
	ProcessingTime  int64            `json:"processing_time_ms"`
}

func EvaluateRisk(c *gin.Context) {
	handler := GetRiskControlHandler()

	var req EvaluateRiskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	ctx := &model.RiskContext{
		SessionID:     req.SessionID,
		IPAddress:     req.IPAddress,
		UserAgent:     req.UserAgent,
		Fingerprint:   req.Fingerprint,
		DeviceInfo:    req.DeviceInfo,
		PositionDiff: req.PositionDiff,
		TraceData:     req.TraceData,
		EnvInfo:       req.EnvInfo,
		VerificationCount: req.VerificationCount,
		FailureCount:  req.FailureCount,
		TimeFromStart: req.TimeFromStart,
		MouseSpeed:    req.MouseSpeed,
		HasTouchDevice: req.HasTouchDevice,
		BrowserPlugins: req.BrowserPlugins,
		Language:      req.Language,
		Timezone:      req.Timezone,
		ScreenRes:     req.ScreenRes,
		Referer:       req.Referer,
		IsProxy:       req.IsProxy,
		IsVPN:         req.IsVPN,
		IsTor:         req.IsTor,
		IsHosting:     req.IsHosting,
		IPReputation:  req.IPReputation,
		Country:       req.Country,
		ASNumber:      req.ASNumber,
	}

	result, err := handler.riskService.EvaluateRisk(c.Request.Context(), ctx)
	if err != nil {
		response.InternalServerError(c, "风险评估失败: "+err.Error())
		return
	}

	resp := EvaluateRiskResponse{
		RiskScore:       result.RiskScore,
		RiskLevel:       string(result.RiskLevel),
		PositionScore:   result.PositionScore,
		BehaviorScore:   result.BehaviorScore,
		EnvScore:        result.EnvScore,
		DeviceScore:     result.DeviceScore,
		IPScore:         result.IPScore,
		Action:          result.Action,
		RiskFactors:     result.RiskFactors,
		Details:         result.Details,
		RecommendVerify: result.RecommendVerify,
		ProcessingTime:  result.ProcessingTime,
	}

	response.Success(c, resp)
}

type GetRulesResponse struct {
	Rules       []service.RiskRuleConfig `json:"rules"`
	Total       int                      `json:"total"`
	LastUpdated string                   `json:"last_updated"`
}

func GetRiskRules(c *gin.Context) {
	handler := GetRiskControlHandler()

	rules := handler.riskService.GetRules()

	resp := GetRulesResponse{
		Rules: rules,
		Total: len(rules),
	}

	response.Success(c, resp)
}

type UpdateRulesRequest struct {
	Rules []service.RiskRuleConfig `json:"rules" binding:"required,min=1"`
}

type UpdateRulesResponse struct {
	Updated   int      `json:"updated"`
	Rules     []service.RiskRuleConfig `json:"rules"`
	Message   string  `json:"message"`
}

func UpdateRiskRules(c *gin.Context) {
	handler := GetRiskControlHandler()

	var req UpdateRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	if len(req.Rules) == 0 {
		response.BadRequest(c, "规则列表不能为空")
		return
	}

	err := handler.riskService.UpdateRules(req.Rules)
	if err != nil {
		response.BadRequest(c, "更新规则失败: "+err.Error())
		return
	}

	resp := UpdateRulesResponse{
		Updated: len(req.Rules),
		Rules:   handler.riskService.GetRules(),
		Message: "规则更新成功",
	}

	response.Success(c, resp)
}

type UpdateWeightsRequest struct {
	DeviceWeight   float64 `json:"device_weight"`
	IPWeight       float64 `json:"ip_weight"`
	BehaviorWeight float64 `json:"behavior_weight"`
	EnvWeight      float64 `json:"env_weight"`
}

type UpdateWeightsResponse struct {
	Weights service.RiskWeights `json:"weights"`
	Message string              `json:"message"`
}

func UpdateRiskWeights(c *gin.Context) {
	handler := GetRiskControlHandler()

	var req UpdateWeightsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	weights := service.RiskWeights{
		DeviceWeight:   req.DeviceWeight,
		IPWeight:       req.IPWeight,
		BehaviorWeight: req.BehaviorWeight,
		EnvWeight:      req.EnvWeight,
	}

	err := handler.riskService.UpdateWeights(weights)
	if err != nil {
		response.BadRequest(c, "更新权重失败: "+err.Error())
		return
	}

	resp := UpdateWeightsResponse{
		Weights: weights,
		Message: "权重更新成功",
	}

	response.Success(c, resp)
}

type GetStatisticsResponse struct {
	TotalCount     int64            `json:"total_count"`
	PassCount      int64            `json:"pass_count"`
	ReviewCount    int64            `json:"review_count"`
	BlockCount     int64            `json:"block_count"`
	AvgRiskScore   float64          `json:"avg_risk_score"`
	RiskLevelStats map[string]int64 `json:"risk_level_stats"`
	TopRiskFactors []service.RiskFactorStat `json:"top_risk_factors"`
}

func GetRiskStatistics(c *gin.Context) {
	handler := GetRiskControlHandler()

	stats, err := handler.riskService.GetStatistics()
	if err != nil {
		response.InternalServerError(c, "获取统计数据失败: "+err.Error())
		return
	}

	resp := GetStatisticsResponse{
		TotalCount:     stats.TotalCount,
		PassCount:      stats.PassCount,
		ReviewCount:    stats.ReviewCount,
		BlockCount:     stats.BlockCount,
		AvgRiskScore:   stats.AvgRiskScore,
		RiskLevelStats: stats.RiskLevelStats,
		TopRiskFactors: stats.TopRiskFactors,
	}

	response.Success(c, resp)
}

type GetDeviceProfileResponse struct {
	DeviceID      string   `json:"device_id"`
	RequestCount  int      `json:"request_count"`
	SuccessCount  int      `json:"success_count"`
	FailureCount  int      `json:"failure_count"`
	BlockCount    int      `json:"block_count"`
	AvgResponseMs float64  `json:"avg_response_ms"`
	RiskScore     float64  `json:"risk_score"`
	IsSuspicious  bool     `json:"is_suspicious"`
	FirstSeen     string   `json:"first_seen"`
	LastSeen      string   `json:"last_seen"`
}

func GetDeviceRiskProfile(c *gin.Context) {
	handler := GetRiskControlHandler()
	deviceID := c.Param("device_id")

	if deviceID == "" {
		response.BadRequest(c, "设备ID不能为空")
		return
	}

	profile, err := handler.riskService.GetDeviceProfile(deviceID)
	if err != nil {
		response.NotFound(c, "设备档案不存在")
		return
	}

	resp := GetDeviceProfileResponse{
		DeviceID:      profile.DeviceID,
		RequestCount:  profile.RequestCount,
		SuccessCount:  profile.SuccessCount,
		FailureCount: profile.FailureCount,
		BlockCount:   profile.BlockCount,
		AvgResponseMs: profile.AvgResponseMs,
		RiskScore:    profile.RiskScore,
		IsSuspicious: profile.IsSuspicious,
		FirstSeen:    profile.FirstSeen.Format("2006-01-02 15:04:05"),
		LastSeen:     profile.LastRequest.Format("2006-01-02 15:04:05"),
	}

	response.Success(c, resp)
}

type GetIPProfileResponse struct {
	IPAddress     string   `json:"ip_address"`
	RequestCount  int      `json:"request_count"`
	SuccessCount  int      `json:"success_count"`
	FailureCount  int      `json:"failure_count"`
	BlockCount    int      `json:"block_count"`
	IsProxy       bool     `json:"is_proxy"`
	IsVPN         bool     `json:"is_vpn"`
	IsTor         bool     `json:"is_tor"`
	IsHosting     bool     `json:"is_hosting"`
	Country       string   `json:"country"`
	RiskScore     float64  `json:"risk_score"`
	GeoVelocity   float64  `json:"geo_velocity"`
	UniqueDevices int      `json:"unique_devices"`
	FirstSeen     string   `json:"first_seen"`
	LastSeen      string   `json:"last_seen"`
}

func GetIPRiskProfile(c *gin.Context) {
	handler := GetRiskControlHandler()
	ipAddress := c.Param("ip_address")

	if ipAddress == "" {
		response.BadRequest(c, "IP地址不能为空")
		return
	}

	profile, err := handler.riskService.GetIPProfile(ipAddress)
	if err != nil {
		response.NotFound(c, "IP档案不存在")
		return
	}

	resp := GetIPProfileResponse{
		IPAddress:     profile.IPAddress,
		RequestCount:  profile.RequestCount,
		SuccessCount:  profile.SuccessCount,
		FailureCount: profile.FailureCount,
		BlockCount:   profile.BlockCount,
		IsProxy:      profile.IsProxy,
		IsVPN:        profile.IsVPN,
		IsTor:        profile.IsTor,
		IsHosting:    profile.IsHosting,
		Country:      profile.Country,
		RiskScore:    profile.RiskScore,
		GeoVelocity:  profile.GeoVelocity,
		UniqueDevices: profile.UniqueDevices,
		FirstSeen:    profile.FirstSeen.Format("2006-01-02 15:04:05"),
		LastSeen:     profile.LastRequest.Format("2006-01-02 15:04:05"),
	}

	response.Success(c, resp)
}

type ResetRiskProfileRequest struct {
	Type string `json:"type" binding:"required"`
	ID   string `json:"id" binding:"required"`
}

func ResetRiskProfile(c *gin.Context) {
	handler := GetRiskControlHandler()

	var req ResetRiskProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	var err error
	switch req.Type {
	case "device":
		err = handler.riskService.ResetDeviceRisk(req.ID)
	case "ip":
		err = handler.riskService.ResetIPRisk(req.ID)
	default:
		response.BadRequest(c, "无效的类型，支持: device, ip")
		return
	}

	if err != nil {
		response.InternalServerError(c, "重置失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"message": "风险档案已重置",
		"type":    req.Type,
		"id":      req.ID,
	})
}
