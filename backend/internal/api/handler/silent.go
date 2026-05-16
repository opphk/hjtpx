package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

type SilentVerifyRequest struct {
	DeviceFingerprint string                  `json:"device_fingerprint"`
	SessionID        string                  `json:"session_id"`
	BehaviorData     []BehaviorDataPointRequest `json:"behavior_data"`
	Timestamp        int64                   `json:"timestamp"`
	UserID           uint                     `json:"user_id"`
	ApplicationID    uint                     `json:"application_id"`
}

type BehaviorDataPointRequest struct {
	X         int     `json:"x"`
	Y         int     `json:"y"`
	Timestamp int64   `json:"timestamp"`
	Event     string  `json:"event"`
}

type SilentVerifyResponse struct {
	Pass        bool                             `json:"pass"`
	RiskLevel   string                           `json:"risk_level"`
	NeedCaptcha bool                             `json:"need_captcha"`
	CaptchaType string                           `json:"captcha_type"`
	Token       string                           `json:"token"`
	Strategy    *service.VerificationStrategy   `json:"strategy,omitempty"`
	WaitTime    int                              `json:"wait_time"`
	Message     string                           `json:"message"`
}

type VerificationStatusResponse struct {
	Status  string                      `json:"status"`
	Result  *service.VerificationCache `json:"result,omitempty"`
	Message string                      `json:"message"`
}

type ConfigUpdateRequest struct {
	Enabled              *bool    `json:"enabled"`
	RiskThreshold        *float64 `json:"risk_threshold"`
	MinBehaviorDataPoints *int    `json:"min_behavior_data_points"`
	MaxVerifyDuration    *int64   `json:"max_verify_duration"`
	EnableDeviceCheck    *bool    `json:"enable_device_check"`
	EnableBehaviorCheck  *bool    `json:"enable_behavior_check"`
	EnableHistoryCheck   *bool    `json:"enable_history_check"`
	CacheTTL             *int64   `json:"cache_ttl"`
}

type StrategyRuleUpdateRequest struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Priority    int                     `json:"priority"`
	Conditions  []RuleConditionRequest `json:"conditions"`
	Action      RuleActionRequest       `json:"action"`
	Enabled     bool                    `json:"enabled"`
}

type RuleConditionRequest struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type RuleActionRequest struct {
	Level       string  `json:"level"`
	NeedCaptcha bool    `json:"need_captcha"`
	CaptchaType string  `json:"captcha_type"`
	WaitTime    int     `json:"wait_time"`
	Score       float64 `json:"score"`
}

var silentService = service.NewSilentVerificationService()

func SilentVerify(c *gin.Context) {
	startTime := time.Now()

	var req SilentVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的请求参数",
			"error":   err.Error(),
		})
		return
	}

	allowed, err := silentService.CheckRateLimit(c.ClientIP(), req.UserID)
	if err != nil {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"success": false,
			"message": "请求过于频繁，请稍后重试",
			"error":   err.Error(),
		})
		return
	}
	if !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"success": false,
			"message": "请求被限制",
		})
		return
	}

	config := silentService.GetConfig()
	if !config.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "无感验证服务已禁用",
		})
		return
	}

	serviceReq := &service.SilentVerifyRequest{
		DeviceFingerprint: req.DeviceFingerprint,
		SessionID:        req.SessionID,
		Timestamp:        req.Timestamp,
		UserID:           req.UserID,
		IPAddress:        c.ClientIP(),
		UserAgent:        c.GetHeader("User-Agent"),
	}

	for _, bd := range req.BehaviorData {
		serviceReq.BehaviorData = append(serviceReq.BehaviorData, service.BehaviorDataPoint{
			X:         bd.X,
			Y:         bd.Y,
			Timestamp: bd.Timestamp,
			Event:     bd.Event,
		})
	}

	response, err := silentService.ProcessVerification(serviceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "验证处理失败",
			"error":   err.Error(),
		})
		return
	}

	duration := time.Since(startTime).Milliseconds()

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"pass":         response.Pass,
		"risk_level":   response.RiskLevel,
		"need_captcha":  response.NeedCaptcha,
		"captcha_type": response.CaptchaType,
		"token":        response.Token,
		"strategy":     response.Strategy,
		"wait_time":    response.WaitTime,
		"message":      response.Message,
		"duration_ms":  duration,
	})
}

func GetSilentVerifyStatus(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "缺少token参数",
		})
		return
	}

	cache, err := silentService.GetVerificationStatus(token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "验证记录不存在",
			"error":   err.Error(),
		})
		return
	}

	status := "pending"
	if cache.Status == "completed" {
		status = "completed"
	} else if cache.Status == "failed" {
		status = "failed"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  status,
		"result":  cache,
		"message": "获取验证状态成功",
	})
}

func GetSilentConfig(c *gin.Context) {
	config := silentService.GetConfig()
	rules := silentService.GetStrategyRules()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"config":  config,
		"rules":   rules,
	})
}

func UpdateSilentConfig(c *gin.Context) {
	var req ConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的请求参数",
			"error":   err.Error(),
		})
		return
	}

	config := silentService.GetConfig()

	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if req.RiskThreshold != nil {
		config.RiskThreshold = *req.RiskThreshold
	}
	if req.MinBehaviorDataPoints != nil {
		config.MinBehaviorDataPoints = *req.MinBehaviorDataPoints
	}
	if req.MaxVerifyDuration != nil {
		config.MaxVerifyDuration = *req.MaxVerifyDuration
	}
	if req.EnableDeviceCheck != nil {
		config.EnableDeviceCheck = *req.EnableDeviceCheck
	}
	if req.EnableBehaviorCheck != nil {
		config.EnableBehaviorCheck = *req.EnableBehaviorCheck
	}
	if req.EnableHistoryCheck != nil {
		config.EnableHistoryCheck = *req.EnableHistoryCheck
	}
	if req.CacheTTL != nil {
		config.CacheTTL = *req.CacheTTL
	}

	silentService.UpdateConfig(config)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置更新成功",
		"config":  config,
	})
}

func UpdateStrategyRule(c *gin.Context) {
	ruleID := c.Param("id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "缺少规则ID",
		})
		return
	}

	var req StrategyRuleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的请求参数",
			"error":   err.Error(),
		})
		return
	}

	rule := service.StrategyRule{
		ID:       req.ID,
		Name:     req.Name,
		Priority: req.Priority,
		Enabled:  req.Enabled,
	}

	for _, cond := range req.Conditions {
		rule.Conditions = append(rule.Conditions, service.RuleCondition{
			Field:    cond.Field,
			Operator: cond.Operator,
			Value:    cond.Value,
		})
	}

	rule.Action = service.StrategyAction{
		Level:       req.Action.Level,
		NeedCaptcha: req.Action.NeedCaptcha,
		CaptchaType: req.Action.CaptchaType,
		WaitTime:    req.Action.WaitTime,
		Score:       req.Action.Score,
	}

	err := silentService.UpdateStrategyRule(ruleID, rule)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "规则不存在",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "策略规则更新成功",
	})
}

func GetSilentStats(c *gin.Context) {
	stats, err := silentService.GetVerificationStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取统计数据失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"stats":   stats,
	})
}

func ResetSilentRateLimit(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "限流重置成功",
	})
}
