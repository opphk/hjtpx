package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

var (
	comboVerificationService *service.ComboVerificationService
	smartCaptchaSelector     *service.SmartCaptchaSelector
	dynamicDifficultyService *service.EnhancedDynamicDifficultyService
)

func init() {
	comboVerificationService = service.NewComboVerificationService(
		service.NewEnhancedAdaptiveDifficultyService(),
	)
	smartCaptchaSelector = service.NewSmartCaptchaSelector()
	dynamicDifficultyService = service.NewEnhancedDynamicDifficultyService()
}

// CreateComboVerificationFlow 创建组合验证流程
func CreateComboVerificationFlow(c *gin.Context) {
	var req service.CreateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	resp, err := comboVerificationService.CreateFlow(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetComboVerificationFlow 获取验证流程信息
func GetComboVerificationFlow(c *gin.Context) {
	flowID := c.Param("flow_id")
	if flowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flow_id is required"})
		return
	}

	ctx := context.Background()
	flow, err := comboVerificationService.GetFlow(ctx, flowID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, flow)
}

// VerifyComboVerificationStep 验证单个步骤
func VerifyComboVerificationStep(c *gin.Context) {
	var req service.VerifyStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	resp, err := comboVerificationService.VerifyStep(ctx, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新动态难度
	if req.BehaviorData != nil && len(req.BehaviorData) > 0 {
		for _, bd := range req.BehaviorData {
			behavior := &service.RealTimeBehavior{
				UserID:    req.FlowID,
				ActionType: "verification",
				Timestamp:  bd.Timestamp,
				Accuracy:   0.0,
				Deviation:  0,
				Velocity:   0,
				ErrorRate:  0,
			}
			if resp.Success {
				behavior.Accuracy = 1.0
				behavior.ErrorRate = 0
			} else {
				behavior.Accuracy = 0.0
				behavior.ErrorRate = 1
			}
			dynamicDifficultyService.UpdateBehavior(ctx, behavior)
		}
	}

	c.JSON(http.StatusOK, resp)
}

// VerifyComboVerificationAll 验证所有步骤
func VerifyComboVerificationAll(c *gin.Context) {
	var req service.VerifyAllRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	resp, err := comboVerificationService.VerifyAll(ctx, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteComboVerificationFlow 删除验证流程
func DeleteComboVerificationFlow(c *gin.Context) {
	flowID := c.Param("flow_id")
	if flowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flow_id is required"})
		return
	}

	ctx := context.Background()
	err := comboVerificationService.DeleteFlow(ctx, flowID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "flow deleted"})
}

// SelectSmartCaptcha 智能选择验证码
func SelectSmartCaptcha(c *gin.Context) {
	var ctxData struct {
		UserID             string  `json:"user_id"`
		RiskScore          float64 `json:"risk_score"`
		DeviceType         string  `json:"device_type"`
		Platform           string  `json:"platform"`
		NetworkQuality     float64 `json:"network_quality"`
		AccessibilityRequired bool `json:"accessibility_required"`
		PreviousAttempts   int     `json:"previous_attempts"`
		TimeConstraints    bool    `json:"time_constraints"`
		GeoLocation        string  `json:"geo_location"`
	}

	if err := c.ShouldBindJSON(&ctxData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	selectionCtx := &service.SelectionContext{
		UserID:              ctxData.UserID,
		RiskScore:           ctxData.RiskScore,
		DeviceType:          ctxData.DeviceType,
		Platform:            ctxData.Platform,
		NetworkQuality:      ctxData.NetworkQuality,
		AccessibilityRequired: ctxData.AccessibilityRequired,
		PreviousAttempts:    ctxData.PreviousAttempts,
		TimeConstraints:     ctxData.TimeConstraints,
		GeoLocation:         ctxData.GeoLocation,
	}

	result, err := smartCaptchaSelector.SelectCaptcha(ctx, selectionCtx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// SelectMultipleCaptchas 选择多个验证码
func SelectMultipleCaptchas(c *gin.Context) {
	var req struct {
		Count int `json:"count"`
		service.SelectionContext
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	results, err := smartCaptchaSelector.SelectMultipleCaptchas(ctx, &req.SelectionContext, req.Count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}

// UpdateBehaviorAndAdjustDifficulty 更新行为数据并调整难度
func UpdateBehaviorAndAdjustDifficulty(c *gin.Context) {
	var behavior service.RealTimeBehavior
	if err := c.ShouldBindJSON(&behavior); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	adjustment, err := dynamicDifficultyService.UpdateBehavior(ctx, &behavior)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if adjustment != nil {
		c.JSON(http.StatusOK, adjustment)
	} else {
		c.JSON(http.StatusOK, gin.H{"status": "no adjustment needed"})
	}
}

// GetDynamicUserDifficulty 获取用户当前动态难度
func GetDynamicUserDifficulty(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	difficulty := dynamicDifficultyService.GetDifficulty(userID)
	c.JSON(http.StatusOK, gin.H{"user_id": userID, "difficulty": string(difficulty)})
}

// SetRiskScore 设置用户风险评分
func SetRiskScore(c *gin.Context) {
	var req struct {
		UserID    string  `json:"user_id"`
		RiskScore float64 `json:"risk_score"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dynamicDifficultyService.SetRiskScore(req.UserID, req.RiskScore)
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "risk score updated"})
}

// GetDifficultyAnalysisReport 获取难度分析报告
func GetDifficultyAnalysisReport(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	report := dynamicDifficultyService.GetAnalysisReport(userID)
	if report == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no report found for user"})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetGlobalDifficultyStats 获取全局难度统计
func GetGlobalDifficultyStats(c *gin.Context) {
	stats := dynamicDifficultyService.GetGlobalStats()
	c.JSON(http.StatusOK, stats)
}

// BatchUpdateBehavior 批量更新行为数据
func BatchUpdateBehavior(c *gin.Context) {
	var behaviors []*service.RealTimeBehavior
	if err := c.ShouldBindJSON(&behaviors); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	adjustments, err := dynamicDifficultyService.BatchUpdateBehavior(ctx, behaviors)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "success",
		"adjustments": adjustments,
		"count":       len(adjustments),
	})
}

// GetAllCaptchaCapabilities 获取所有验证码能力信息
func GetAllCaptchaCapabilities(c *gin.Context) {
	capabilities := smartCaptchaSelector.GetAllCapabilities()
	c.JSON(http.StatusOK, capabilities)
}

// GetCaptchaCapability 获取单个验证码能力信息
func GetCaptchaCapability(c *gin.Context) {
	captchaType := c.Param("captcha_type")
	if captchaType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "captcha_type is required"})
		return
	}

	capability, err := smartCaptchaSelector.GetCapability(service.CaptchaType(captchaType))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, capability)
}

// AnalyzeUserCaptchaHistory 分析用户验证码历史
func AnalyzeUserCaptchaHistory(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	result := smartCaptchaSelector.AnalyzeUserHistory(userID)
	if result == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no history found for user"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// RefreshDifficultySession 重置用户难度会话
func RefreshDifficultySession(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	dynamicDifficultyService.ResetSession(userID)
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "session refreshed"})
}

// GetAdjustmentHistory 获取难度调整历史
func GetAdjustmentHistory(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	history := dynamicDifficultyService.GetAdjustmentHistory(userID)
	c.JSON(http.StatusOK, gin.H{"user_id": userID, "history": history})
}
