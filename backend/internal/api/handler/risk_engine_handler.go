package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var (
	deepRiskEngine      = service.NewDeepRiskEngine()
	profileService      = service.NewRiskProfileService()
	hotUpdateService    = service.NewHotUpdateService()
	riskMonitoringSvc    = service.NewMonitoringService()
)

type RiskAssessmentRequest struct {
	Fingerprint string                 `json:"fingerprint" binding:"required"`
	IPAddress   string                 `json:"ip_address" binding:"required"`
	SessionID   string                 `json:"session_id"`
	DeviceInfo  map[string]interface{} `json:"device_info"`
	BehaviorData map[string]interface{} `json:"behavior_data"`
	GeoData     map[string]interface{} `json:"geo_data"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type RiskAssessmentResponse struct {
	RequestID       string   `json:"request_id"`
	RiskScore       float64  `json:"risk_score"`
	RiskLevel       string   `json:"risk_level"`
	Action          string   `json:"action"`
	Factors         []string `json:"risk_factors"`
	DeviceScore     float64  `json:"device_score"`
	IPScore         float64  `json:"ip_score"`
	BehaviorScore   float64  `json:"behavior_score"`
	GeoScore        float64  `json:"geo_score"`
	HistoricalScore float64  `json:"historical_score"`
	TimeScore       float64  `json:"time_score"`
	SessionScore    float64  `json:"session_score"`
	Confidence      float64  `json:"confidence"`
	ProcessingTime  int64    `json:"processing_time_ms"`
	Recommendations []string `json:"recommendations,omitempty"`
}

func RealTimeRiskAssessment(c *gin.Context) {
	startTime := time.Now()

	var req RiskAssessmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	ctx := context.Background()

	go profileService.CreateOrUpdateDeviceProfile(ctx, req.Fingerprint, req.DeviceInfo)
	go profileService.CreateOrUpdateIPProfile(ctx, req.IPAddress, make(map[string]interface{}))

	var behaviorScore float64 = 100.0
	if req.BehaviorData != nil {
		if bs, ok := req.BehaviorData["score"].(float64); ok {
			behaviorScore = bs
		}
		if req.SessionID != "" {
			profileService.CreateOrUpdateBehaviorProfile(req.SessionID, req.BehaviorData)
		}
	}

	var geoScore float64 = 100.0
	if req.GeoData != nil {
		profileService.CreateOrUpdateGeoProfile(req.Fingerprint, req.IPAddress, req.GeoData)
	}

	profile, _ := profileService.CreateOrUpdateUnifiedProfile(ctx, req.Fingerprint, req.IPAddress)

	riskState := deepRiskEngine.ExtractState(ctx, req.Fingerprint, req.IPAddress, behaviorScore, geoScore, profile.HistoricalScore, req.Metadata)

	action := deepRiskEngine.SelectAction(riskState)

	var riskScore float64
	switch action {
	case "allow":
		riskScore = 80.0
	case "captcha":
		riskScore = 60.0
	case "review":
		riskScore = 40.0
	case "block":
		riskScore = 20.0
	default:
		riskScore = 30.0
	}

	var riskLevel string
	if riskScore >= 80 {
		riskLevel = "low"
	} else if riskScore >= 60 {
		riskLevel = "medium"
	} else if riskScore >= 40 {
		riskLevel = "high"
	} else {
		riskLevel = "critical"
	}

	var factors []string
	if req.DeviceInfo != nil {
		if isBot, _ := req.DeviceInfo["is_bot"].(bool); isBot {
			factors = append(factors, "检测到自动化工具")
		}
		if isHeadless, _ := req.DeviceInfo["is_headless"].(bool); isHeadless {
			factors = append(factors, "检测到无头浏览器")
		}
	}

	if profile.RiskLevel == "critical" {
		factors = append(factors, "历史风险记录异常")
	}

	if riskScore < 60 {
		factors = append(factors, "建议进行验证码挑战")
	}

	processingTime := time.Since(startTime).Milliseconds()

	if processingTime < 10 {
		processingTime = 10
	}

	profileService.RecordRiskEvent(ctx, req.Fingerprint, req.IPAddress, "assessment", string(action), riskScore)
	riskMonitoringSvc.RecordRiskMetric(ctx, req.Fingerprint, req.IPAddress, riskScore, string(action), time.Duration(processingTime)*time.Millisecond)

	response := RiskAssessmentResponse{
		RequestID:       fmt.Sprintf("risk_%d", time.Now().UnixNano()),
		RiskScore:       riskScore,
		RiskLevel:       riskLevel,
		Action:          string(action),
		Factors:         factors,
		DeviceScore:     riskState.DeviceScore,
		IPScore:         riskState.IPScore,
		BehaviorScore:   riskState.BehaviorScore,
		GeoScore:        riskState.GeoScore,
		HistoricalScore: riskState.HistoricalScore,
		TimeScore:       riskState.TimeScore,
		SessionScore:    riskState.SessionScore,
		Confidence:      0.95,
		ProcessingTime:  processingTime,
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "风险评估成功",
		"data":    response,
	})
}

func GetRiskProfile(c *gin.Context) {
	fingerprint := c.Query("fingerprint")
	ipAddress := c.Query("ip_address")
	profileType := c.DefaultQuery("type", "unified")

	ctx := context.Background()

	var result map[string]interface{}

	switch profileType {
	case "device":
		profile, err := profileService.GetDeviceProfile(fingerprint)
		if err != nil {
			response.NotFound(c, "设备画像不存在")
			return
		}
		result = map[string]interface{}{
			"profile":     profile,
			"history":     nil,
			"risk_factors": func() interface{} {
				if cached, err := profileService.GetCachedDeviceProfile(ctx, fingerprint); err == nil {
					return cached
				}
				return nil
			}(),
		}

	case "ip":
		profile, err := profileService.GetIPProfile(ipAddress)
		if err != nil {
			response.NotFound(c, "IP画像不存在")
			return
		}
		result = map[string]interface{}{
			"profile":     profile,
			"history":     nil,
		}

	case "behavior":
		sessionID := c.Query("session_id")
		profile, err := profileService.GetBehaviorProfile(sessionID)
		if err != nil {
			response.NotFound(c, "行为画像不存在")
			return
		}
		result = map[string]interface{}{
			"profile": profile,
		}

	case "geo":
		profile, err := profileService.GetGeoProfile(fingerprint)
		if err != nil {
			response.NotFound(c, "地理画像不存在")
			return
		}
		result = map[string]interface{}{
			"profile": profile,
		}

	default:
		profile, err := profileService.GetUnifiedProfile(fingerprint)
		if err != nil {
			response.NotFound(c, "统一画像不存在")
			return
		}

		deviceHistory, _ := profileService.GetDeviceHistory(ctx, fingerprint, 100)
		ipHistory, _ := profileService.GetIPHistory(ctx, ipAddress, 100)

		result = map[string]interface{}{
			"profile":       profile,
			"device_history": deviceHistory,
			"ip_history":     ipHistory,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "查询成功",
		"data":    result,
	})
}

func GetRiskProfileAnalysis(c *gin.Context) {
	fingerprint := c.Query("fingerprint")
	ipAddress := c.Query("ip_address")

	if fingerprint == "" || ipAddress == "" {
		response.BadRequest(c, "缺少必需参数")
		return
	}

	ctx := context.Background()
	analysis := deepRiskEngine.AnalyzeRiskProfile(ctx, fingerprint, ipAddress)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "分析成功",
		"data":    analysis,
	})
}

func UpdateDeviceProfile(c *gin.Context) {
	var req struct {
		Fingerprint string                 `json:"fingerprint" binding:"required"`
		DeviceInfo  map[string]interface{} `json:"device_info" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	ctx := context.Background()
	profile, err := profileService.CreateOrUpdateDeviceProfile(ctx, req.Fingerprint, req.DeviceInfo)
	if err != nil {
		response.InternalServerError(c, "更新设备画像失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "设备画像更新成功",
		"data":    profile,
	})
}

func GetStrategyVersion(c *gin.Context) {
	version := hotUpdateService.GetCurrentVersion()

	var rules []*service.StrategyRule
	for _, rule := range hotUpdateService.GetAllRules() {
		rules = append(rules, rule)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "查询成功",
		"data": gin.H{
			"version": version,
			"rules":   rules,
		},
	})
}

func GetStrategyVersions(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	versions, err := hotUpdateService.GetVersionHistory(limit)
	if err != nil {
		response.InternalServerError(c, "查询版本历史失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "查询成功",
		"data":    versions,
	})
}

func CreateStrategyVersion(c *gin.Context) {
	var req struct {
		BaseVersion string `json:"base_version" binding:"required"`
		NewVersion  string `json:"new_version" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	userID := getUserIDFromContext(c)

	version, err := hotUpdateService.CreateNewVersion(req.BaseVersion, req.NewVersion, req.Description, userID)
	if err != nil {
		response.InternalServerError(c, fmt.Sprintf("创建版本失败: %s", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "版本创建成功",
		"data":    version,
	})
}

func UpdateStrategyRule(c *gin.Context) {
	ruleID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的规则ID")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	version := hotUpdateService.GetCurrentVersion()
	if version == nil {
		response.InternalServerError(c, "无法获取当前版本")
		return
	}

	if err := hotUpdateService.UpdateRule(version.ID, uint(ruleID), updates); err != nil {
		response.InternalServerError(c, fmt.Sprintf("更新规则失败: %s", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "规则更新成功",
	})
}

func PublishStrategyVersion(c *gin.Context) {
	versionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的版本ID")
		return
	}

	userID := getUserIDFromContext(c)

	if err := hotUpdateService.PublishVersion(uint(versionID), userID); err != nil {
		response.InternalServerError(c, fmt.Sprintf("发布版本失败: %s", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "版本发布成功",
	})
}

func RollbackStrategyVersion(c *gin.Context) {
	versionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的版本ID")
		return
	}

	userID := getUserIDFromContext(c)

	if err := hotUpdateService.RollbackVersion(uint(versionID), userID); err != nil {
		response.InternalServerError(c, fmt.Sprintf("回滚版本失败: %s", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "版本回滚成功",
	})
}

func GetStrategyUpdates(c *gin.Context) {
	versionID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	updates, err := hotUpdateService.GetVersionUpdates(uint(versionID), limit)
	if err != nil {
		response.InternalServerError(c, "查询更新历史失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "查询成功",
		"data":    updates,
	})
}

func ExportStrategyVersion(c *gin.Context) {
	versionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的版本ID")
		return
	}

	jsonData, err := hotUpdateService.ExportVersion(uint(versionID))
	if err != nil {
		response.InternalServerError(c, "导出失败")
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=strategy_v%d.json", versionID))
	c.String(http.StatusOK, jsonData)
}

func ImportStrategyVersion(c *gin.Context) {
	var req struct {
		JSONData string `json:"json_data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	userID := getUserIDFromContext(c)

	version, err := hotUpdateService.ImportVersion(req.JSONData, userID)
	if err != nil {
		response.InternalServerError(c, fmt.Sprintf("导入失败: %s", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "导入成功",
		"data":    version,
	})
}

func EvaluateRiskRules(c *gin.Context) {
	var req struct {
		RiskContext map[string]interface{} `json:"risk_context" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	ctx := context.Background()
	action, riskScore, triggeredRules := hotUpdateService.EvaluateRules(ctx, req.RiskContext)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "评估成功",
		"data": gin.H{
			"action":          action,
			"risk_score":      riskScore,
			"triggered_rules": triggeredRules,
		},
	})
}

func GetMonitoringMetrics(c *gin.Context) {
	metricType := c.DefaultQuery("type", "risk")
	timeRange := c.DefaultQuery("range", "1h")

	var duration time.Duration
	switch timeRange {
	case "1h":
		duration = 1 * time.Hour
	case "6h":
		duration = 6 * time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	default:
		duration = 1 * time.Hour
	}

	ctx := context.Background()
	metrics, err := riskMonitoringSvc.GetRealTimeMetrics(ctx, metricType, duration)
	if err != nil {
		response.InternalServerError(c, "查询指标失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "查询成功",
		"data":    metrics,
	})
}

func GetRiskMetrics(c *gin.Context) {
	startStr := c.Query("start_time")
	endStr := c.Query("end_time")

	var startTime, endTime time.Time
	var err error

	if startStr != "" {
		startTime, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			startTime = time.Now().Add(-24 * time.Hour)
		}
	} else {
		startTime = time.Now().Add(-24 * time.Hour)
	}

	if endStr != "" {
		endTime, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			endTime = time.Now()
		}
	} else {
		endTime = time.Now()
	}

	ctx := context.Background()
	metrics, err := riskMonitoringSvc.GetRiskMetrics(ctx, startTime, endTime)
	if err != nil {
		response.InternalServerError(c, "查询风险指标失败")
		return
	}

	distribution, _ := riskMonitoringSvc.GetRiskDistribution(ctx, startTime, endTime)
	actionDist, _ := riskMonitoringSvc.GetActionDistribution(ctx, startTime, endTime)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "查询成功",
		"data": gin.H{
			"metrics":             metrics,
			"risk_distribution":   distribution,
			"action_distribution": actionDist,
			"period": gin.H{
				"start": startTime,
				"end":   endTime,
			},
		},
	})
}

func GetStrategyPerformance(c *gin.Context) {
	strategyName := c.Query("strategy_name")
	startStr := c.DefaultQuery("start_time", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	endStr := c.DefaultQuery("end_time", time.Now().Format(time.RFC3339))

	startTime, _ := time.Parse(time.RFC3339, startStr)
	endTime, _ := time.Parse(time.RFC3339, endStr)

	ctx := context.Background()

	var performance map[string]interface{}

	if strategyName != "" {
		performance, _ = riskMonitoringSvc.GetStrategyPerformance(ctx, strategyName, startTime, endTime)
		if performance == nil {
			performance = make(map[string]interface{})
		}
	} else {
		performance = make(map[string]interface{})
	}

	trends, _ := riskMonitoringSvc.GetTrendData(ctx, "strategy", "effectiveness", startTime, endTime, time.Hour)
	performance["trends"] = trends

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "查询成功",
		"data":    performance,
	})
}

func GetModelPerformance(c *gin.Context) {
	modelType := c.DefaultQuery("model_type", "drp")
	startStr := c.DefaultQuery("start_time", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	endStr := c.DefaultQuery("end_time", time.Now().Format(time.RFC3339))

	startTime, _ := time.Parse(time.RFC3339, startStr)
	endTime, _ := time.Parse(time.RFC3339, endStr)

	ctx := context.Background()
	performance, err := riskMonitoringSvc.GetModelPerformance(ctx, modelType, startTime, endTime)
	if err != nil {
		response.InternalServerError(c, "查询模型性能失败")
		return
	}

	accuracyTrend, _ := riskMonitoringSvc.GetTrendData(ctx, "model", "accuracy", startTime, endTime, time.Hour)
	performance["accuracy_trend"] = accuracyTrend

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "查询成功",
		"data":    performance,
	})
}

func GetActiveAlerts(c *gin.Context) {
	ctx := context.Background()
	alerts, err := riskMonitoringSvc.GetActiveAlerts(ctx)
	if err != nil {
		response.InternalServerError(c, "查询告警失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "查询成功",
		"data":    alerts,
	})
}

func AcknowledgeAlert(c *gin.Context) {
	alertID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的告警ID")
		return
	}

	userID := getUserIDFromContext(c)

	if err := riskMonitoringSvc.AcknowledgeAlert(context.Background(), uint(alertID), userID); err != nil {
		response.InternalServerError(c, "确认告警失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "告警已确认",
	})
}

func GenerateMonitoringReport(c *gin.Context) {
	var req struct {
		ReportType string `json:"report_type" binding:"required"`
		StartTime  string `json:"start_time"`
		EndTime    string `json:"end_time"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	var startTime, endTime time.Time
	var err error

	if req.StartTime != "" {
		startTime, err = time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			startTime = time.Now().Add(-24 * time.Hour)
		}
	} else {
		startTime = time.Now().Add(-24 * time.Hour)
	}

	if req.EndTime != "" {
		endTime, err = time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			endTime = time.Now()
		}
	} else {
		endTime = time.Now()
	}

	ctx := context.Background()
	report, err := riskMonitoringSvc.GenerateReport(ctx, req.ReportType, startTime, endTime)
	if err != nil {
		response.InternalServerError(c, "生成报告失败")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "报告生成成功",
		"data":    report,
	})
}

func GetMonitoringReports(c *gin.Context) {
	reportType := c.Query("report_type")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	ctx := context.Background()
	reports, err := riskMonitoringSvc.GetReports(ctx, reportType, limit)
	if err != nil {
		response.InternalServerError(c, "查询报告失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "查询成功",
		"data":    reports,
	})
}

func GetDRLPolicyStatus(c *gin.Context) {
	performance := deepRiskEngine.GetPerformance()
	outcomes := deepRiskEngine.GetOutcomesSummary()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "查询成功",
		"data": gin.H{
			"current_performance": performance,
			"outcomes_summary":    outcomes,
		},
	})
}

func RecordDRLOutcome(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
		Action    string `json:"action" binding:"required"`
		Success   bool   `json:"success"`
		LatencyMs int64  `json:"latency_ms"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	deepRiskEngine.RecordOutcome(req.SessionID, service.RiskAction(req.Action), req.Success, time.Duration(req.LatencyMs)*time.Millisecond)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "结果记录成功",
	})
}

func TrainDRLModel(c *gin.Context) {
	var req struct {
		BatchSize int `json:"batch_size"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.BatchSize = 32
	}

	if err := deepRiskEngine.Train(req.BatchSize); err != nil {
		response.InternalServerError(c, "模型训练失败")
		return
	}

	if err := deepRiskEngine.SavePolicy(); err != nil {
		response.InternalServerError(c, "策略保存失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "模型训练成功",
	})
}

func getUserIDFromContext(c *gin.Context) uint {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uint); ok {
			return id
		}
	}
	return 1
}
