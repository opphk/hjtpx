package handler

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var seamlessV15Service *service.SeamlessV15Service

func GetSeamlessV15Service() *service.SeamlessV15Service {
	if seamlessV15Service == nil {
		seamlessV15Service = service.NewSeamlessV15Service()
	}
	return seamlessV15Service
}

type SeamlessV15UpdateRequest struct {
	UserID       string                  `json:"user_id" binding:"required"`
	Fingerprint  string                  `json:"fingerprint" binding:"required"`
	SessionID    string                  `json:"session_id"`
	Timestamp   int64                   `json:"timestamp"`
	Duration     int64                   `json:"duration"`
	MouseMoves   int                     `json:"mouse_moves"`
	KeyStrokes   int                     `json:"key_strokes"`
	Clicks       int                     `json:"clicks"`
	ScrollEvents int                     `json:"scroll_events"`
	AverageSpeed float64                 `json:"average_speed"`
	RiskScore    float64                 `json:"risk_score"`
	Success      bool                    `json:"success"`
	IPAddress    string                  `json:"ip_address"`
	UserAgent    string                  `json:"user_agent"`
	BehaviorHash string                  `json:"behavior_hash"`
	FingerprintComponents map[string]string `json:"fingerprint_components"`
}

type SeamlessV15VerifyRequest struct {
	UserID       string                 `json:"user_id" binding:"required"`
	Fingerprint  string                 `json:"fingerprint" binding:"required"`
	SessionID    string                 `json:"session_id"`
	RiskScore    float64                `json:"risk_score"`
	BehaviorData []map[string]interface{} `json:"behavior_data"`
}

type SeamlessV15ReportRequest struct {
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
}

type SeamlessV15UpdateResponse struct {
	Success       bool    `json:"success"`
	Message       string  `json:"message"`
	TrustScore    float64 `json:"trust_score,omitempty"`
	ModelAccuracy float64 `json:"model_accuracy,omitempty"`
}

type SeamlessV15VerifyResponse struct {
	Success           bool     `json:"success"`
	VerificationType  string   `json:"verification_type"`
	TrustScore        float64  `json:"trust_score"`
	RiskScore         float64  `json:"risk_score"`
	Confidence        float64  `json:"confidence"`
	Reasons           []string `json:"reasons"`
	ProgressiveLevel  int      `json:"progressive_level"`
	Token             string   `json:"token,omitempty"`
	SkipVerification  bool     `json:"skip_verification"`
	Message           string   `json:"message,omitempty"`
}

type SeamlessV15TrustScoreResponse struct {
	Success           bool    `json:"success"`
	UserID            string  `json:"user_id"`
	Fingerprint       string  `json:"fingerprint"`
	TrustScore        float64 `json:"trust_score"`
	RiskScore         float64 `json:"risk_score"`
	DeviceStable      bool    `json:"device_stable"`
	DeviceConfidence  float64 `json:"device_confidence"`
	DeviceUsageCount  int     `json:"device_usage_count"`
	BehaviorKnown     bool    `json:"behavior_known"`
	BehaviorConfidence float64 `json:"behavior_confidence"`
	SessionCount      int     `json:"session_count"`
	Recommendation    string  `json:"recommendation"`
}

type SeamlessV15ReportResponse struct {
	Success   bool                    `json:"success"`
	ReportID  string                  `json:"report_id"`
	Summary   *ReportSummaryData      `json:"summary"`
	DeviceAnalysis  *DeviceAnalysisData  `json:"device_analysis"`
	BehaviorAnalysis *BehaviorAnalysisData `json:"behavior_analysis"`
	TrustAnalysis   *TrustAnalysisData   `json:"trust_analysis"`
	SwitchAnalysis  *SwitchAnalysisData  `json:"switch_analysis"`
	Recommendations []string           `json:"recommendations"`
	GeneratedAt     time.Time          `json:"generated_at"`
}

type ReportSummaryData struct {
	TotalVerifications   int     `json:"total_verifications"`
	SeamlessVerifications int     `json:"seamless_verifications"`
	StrongVerifications   int     `json:"strong_verifications"`
	BlockedVerifications  int     `json:"blocked_verifications"`
	SeamlessRate         float64 `json:"seamless_rate"`
	BlockRate            float64 `json:"block_rate"`
	AverageTrustScore    float64 `json:"average_trust_score"`
	AverageRiskScore     float64 `json:"average_risk_score"`
}

type DeviceAnalysisData struct {
	TotalDevices       int              `json:"total_devices"`
	TrustedDevices     int              `json:"trusted_devices"`
	NewDevices         int              `json:"new_devices"`
	SuspiciousDevices  int              `json:"suspicious_devices"`
	DevicesByStability map[string]int   `json:"devices_by_stability"`
	FingerprintAccuracy float64          `json:"fingerprint_accuracy"`
}

type BehaviorAnalysisData struct {
	TotalUsers          int      `json:"total_users"`
	ActiveModels        int      `json:"active_models"`
	ModelAccuracy       float64  `json:"model_accuracy"`
	AnomalyDetectionRate float64 `json:"anomaly_detection_rate"`
	BehavioralEntropyAvg float64 `json:"behavioral_entropy_avg"`
	HabitStrengthAvg     float64 `json:"habit_strength_avg"`
	CommonPatterns       []string `json:"common_patterns"`
}

type TrustAnalysisData struct {
	TrustDistribution   map[string]int `json:"trust_distribution"`
	AverageBaseTrust    float64        `json:"average_base_trust"`
	AverageAdjustedTrust float64       `json:"average_adjusted_trust"`
	TrustScoreVariance  float64        `json:"trust_score_variance"`
	UsersAboveThreshold int            `json:"users_above_threshold"`
	UsersBelowThreshold int            `json:"users_below_threshold"`
}

type SwitchAnalysisData struct {
	TotalSwitches      int              `json:"total_switches"`
	SeamlessToStrong   int              `json:"seamless_to_strong"`
	StrongToSeamless   int              `json:"strong_to_seamless"`
	SwitchReasons      map[string]int   `json:"switch_reasons"`
	AverageSwitchLatency float64        `json:"average_switch_latency"`
	SwitchSuccessRate  float64          `json:"switch_success_rate"`
}

func SeamlessV15Update(c *gin.Context) {
	var req SeamlessV15UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	if req.Timestamp == 0 {
		req.Timestamp = time.Now().UnixMilli()
	}

	svc := GetSeamlessV15Service()

	behaviorData := &service.BehaviorUpdateData{
		SessionID:     req.SessionID,
		Timestamp:    time.UnixMilli(req.Timestamp),
		Duration:     req.Duration,
		MouseMoves:   req.MouseMoves,
		KeyboardEvents: req.KeyStrokes,
		Clicks:       req.Clicks,
		ScrollEvents: req.ScrollEvents,
		AverageSpeed: req.AverageSpeed,
		RiskScore:    req.RiskScore,
		Success:      req.Success,
		BehaviorHash: req.BehaviorHash,
		Fingerprint:  req.Fingerprint,
		FingerprintComponents: req.FingerprintComponents,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
	}

	if err := svc.UpdateBehaviorData(req.UserID, behaviorData); err != nil {
		response.Error(c, http.StatusInternalServerError, "更新行为数据失败: "+err.Error())
		return
	}

	trustResult := svc.GetTrustScore(req.UserID, req.Fingerprint)

	seamlessLog := models.SeamlessVerification{
		SessionID:         req.SessionID,
		DeviceFingerprint: req.Fingerprint,
		Decision:          "update",
		RiskScore:         req.RiskScore,
		IPAddress:         req.IPAddress,
		UserAgent:         req.UserAgent,
		Duration:          req.Duration,
	}
	database.DB.Create(&seamlessLog)

	response.Success(c, SeamlessV15UpdateResponse{
		Success:       true,
		Message:       "行为数据更新成功",
		TrustScore:    trustResult.TrustScore,
		ModelAccuracy: trustResult.BehaviorConfidence,
	})
}

func SeamlessV15Verify(c *gin.Context) {
	var req SeamlessV15VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	if req.SessionID == "" {
		req.SessionID = uuid.New().String()
	}

	svc := GetSeamlessV15Service()

	if req.RiskScore < 0 {
		req.RiskScore = 50.0
	}

	result := svc.PerformSeamlessVerification(req.UserID, req.Fingerprint, req.RiskScore)

	seamlessLog := models.SeamlessVerification{
		SessionID:         req.SessionID,
		DeviceFingerprint: req.Fingerprint,
		Decision:         result.VerificationType,
		RiskScore:        result.RiskScore,
		Reason:           fmt.Sprintf("trust:%.2f,conf:%.2f", result.TrustScore, result.Confidence),
		IPAddress:         c.ClientIP(),
		UserAgent:        c.GetHeader("User-Agent"),
	}
	database.DB.Create(&seamlessLog)

	skipVerification := result.VerificationType == "seamless" && result.Confidence > 0.7

	response.Success(c, SeamlessV15VerifyResponse{
		Success:          result.Success,
		VerificationType: result.VerificationType,
		TrustScore:       result.TrustScore,
		RiskScore:        result.RiskScore,
		Confidence:       result.Confidence,
		Reasons:          result.Reasons,
		ProgressiveLevel: result.ProgressiveLevel,
		Token:            result.Token,
		SkipVerification: skipVerification,
		Message:          getVerificationMessage(result),
	})
}

func SeamlessV15TrustScore(c *gin.Context) {
	userID := c.Query("user_id")
	fingerprint := c.Query("fingerprint")

	if userID == "" || fingerprint == "" {
		response.BadRequest(c, "缺少必要参数: user_id 或 fingerprint")
		return
	}

	svc := GetSeamlessV15Service()
	result := svc.GetTrustScore(userID, fingerprint)

	recommendation := "normal"
	if result.TrustScore > 0.8 {
		recommendation = "highly_trusted"
	} else if result.TrustScore < 0.3 {
		recommendation = "require_strong_verification"
	} else if result.DeviceConfidence < 0.5 || result.BehaviorConfidence < 0.5 {
		recommendation = "continue_learning"
	}

	response.Success(c, SeamlessV15TrustScoreResponse{
		Success:            true,
		UserID:             userID,
		Fingerprint:        fingerprint,
		TrustScore:         result.TrustScore,
		RiskScore:          result.RiskScore,
		DeviceStable:       result.DeviceStable,
		DeviceConfidence:   result.DeviceConfidence,
		DeviceUsageCount:   result.DeviceUsageCount,
		BehaviorKnown:      result.BehaviorKnown,
		BehaviorConfidence: result.BehaviorConfidence,
		SessionCount:       result.SessionCount,
		Recommendation:     recommendation,
	})
}

func SeamlessV15Report(c *gin.Context) {
	var req SeamlessV15ReportRequest
	c.ShouldBindQuery(&req)

	periodStart := time.Now().AddDate(0, 0, -7)
	periodEnd := time.Now()

	if req.PeriodStart != "" {
		if parsed, err := time.Parse("2006-01-02", req.PeriodStart); err == nil {
			periodStart = parsed
		}
	}

	if req.PeriodEnd != "" {
		if parsed, err := time.Parse("2006-01-02", req.PeriodEnd); err == nil {
			periodEnd = parsed.Add(24*time.Hour - time.Second)
		}
	}

	svc := GetSeamlessV15Service()
	report := svc.GenerateReport(periodStart, periodEnd)

	responseData := SeamlessV15ReportResponse{
		Success:  true,
		ReportID: report.ReportID,
		Summary: &ReportSummaryData{
			TotalVerifications:    report.Summary.TotalVerifications,
			SeamlessVerifications: report.Summary.SeamlessVerifications,
			StrongVerifications:   report.Summary.StrongVerifications,
			BlockedVerifications:  report.Summary.BlockedVerifications,
			SeamlessRate:         report.Summary.SeamlessRate,
			BlockRate:            report.Summary.BlockRate,
			AverageTrustScore:    report.Summary.AverageTrustScore,
			AverageRiskScore:     report.Summary.AverageRiskScore,
		},
		DeviceAnalysis: &DeviceAnalysisData{
			TotalDevices:       report.DeviceAnalysis.TotalDevices,
			TrustedDevices:     report.DeviceAnalysis.TrustedDevices,
			NewDevices:        report.DeviceAnalysis.NewDevices,
			SuspiciousDevices: report.DeviceAnalysis.SuspiciousDevices,
			DevicesByStability: report.DeviceAnalysis.DevicesByStability,
			FingerprintAccuracy: report.DeviceAnalysis.FingerprintAccuracy,
		},
		BehaviorAnalysis: &BehaviorAnalysisData{
			TotalUsers:            report.BehaviorAnalysis.TotalUsers,
			ActiveModels:          report.BehaviorAnalysis.ActiveModels,
			ModelAccuracy:         report.BehaviorAnalysis.ModelAccuracy,
			AnomalyDetectionRate:  report.BehaviorAnalysis.AnomalyDetectionRate,
			BehavioralEntropyAvg:  report.BehaviorAnalysis.BehavioralEntropyAvg,
			HabitStrengthAvg:      report.BehaviorAnalysis.HabitStrengthAvg,
			CommonPatterns:        report.BehaviorAnalysis.CommonPatterns,
		},
		TrustAnalysis: &TrustAnalysisData{
			TrustDistribution:    report.TrustAnalysis.TrustDistribution,
			AverageBaseTrust:     report.TrustAnalysis.AverageBaseTrust,
			AverageAdjustedTrust: report.TrustAnalysis.AverageAdjustedTrust,
			TrustScoreVariance:   report.TrustAnalysis.TrustScoreVariance,
			UsersAboveThreshold:  report.TrustAnalysis.UsersAboveThreshold,
			UsersBelowThreshold:  report.TrustAnalysis.UsersBelowThreshold,
		},
		SwitchAnalysis: &SwitchAnalysisData{
			TotalSwitches:        report.SwitchAnalysis.TotalSwitches,
			SeamlessToStrong:     report.SwitchAnalysis.SeamlessToStrong,
			StrongToSeamless:     report.SwitchAnalysis.StrongToSeamless,
			SwitchReasons:        report.SwitchAnalysis.SwitchReasons,
			AverageSwitchLatency: report.SwitchAnalysis.AverageSwitchLatency,
			SwitchSuccessRate:    report.SwitchAnalysis.SwitchSuccessRate,
		},
		Recommendations: report.Recommendations,
		GeneratedAt:    report.GeneratedAt,
	}

	response.Success(c, responseData)
}

func SeamlessV15Stats(c *gin.Context) {
	svc := GetSeamlessV15Service()
	stats := svc.GetGlobalStats()

	var totalVerifications int64
	var seamlessCount int64
	var blockCount int64

	database.DB.Model(&models.SeamlessVerification{}).Count(&totalVerifications)
	database.DB.Model(&models.SeamlessVerification{}).Where("decision = ?", "seamless").Count(&seamlessCount)
	database.DB.Model(&models.SeamlessVerification{}).Where("decision = ?", "block").Count(&blockCount)

	stats["db_total_verifications"] = totalVerifications
	stats["db_seamless_count"] = seamlessCount
	stats["db_block_count"] = blockCount

	if totalVerifications > 0 {
		stats["seamless_rate"] = float64(seamlessCount) / float64(totalVerifications) * 100
		stats["block_rate"] = float64(blockCount) / float64(totalVerifications) * 100
	}

	response.Success(c, gin.H{
		"success": true,
		"stats":   stats,
	})
}

func SeamlessV15Export(c *gin.Context) {
	svc := GetSeamlessV15Service()
	jsonData, err := svc.ExportModelData()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "导出模型数据失败: "+err.Error())
		return
	}

	filename := fmt.Sprintf("seamless_v15_model_%s.json", time.Now().Format("20060102150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/json")
	c.Data(200, "application/json", jsonData)
}

func SeamlessV15Import(c *gin.Context) {
	var req struct {
		Data string `json:"data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	svc := GetSeamlessV15Service()
	if err := svc.ImportModelData([]byte(req.Data)); err != nil {
		response.Error(c, http.StatusInternalServerError, "导入模型数据失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"success": true,
		"message": "模型数据导入成功",
	})
}

func SeamlessV15Cleanup(c *gin.Context) {
	var req struct {
		RetentionDays int `json:"retention_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.RetentionDays <= 0 {
		req.RetentionDays = 90
	}

	svc := GetSeamlessV15Service()
	cleaned := svc.CleanupOldData(req.RetentionDays)

	response.Success(c, gin.H{
		"success":       true,
		"cleaned_count": cleaned,
		"message":       fmt.Sprintf("已清理 %d 条过期数据", cleaned),
	})
}

func SeamlessV15Config(c *gin.Context) {
	svc := GetSeamlessV15Service()

	config := gin.H{
		"device_learning": gin.H{
			"initial_trust_score":   0.5,
			"learning_rate":          0.1,
			"decay_rate":             0.05,
			"min_confidence_samples": 5,
			"similarity_threshold":    0.85,
		},
		"behavior_modeling": gin.H{
			"window_size":           "720h",
			"feature_update_rate":    0.15,
			"anomaly_threshold":     2.5,
			"min_samples_for_model": 10,
		},
		"trust_scoring": gin.H{
			"device_weight":      0.30,
			"behavior_weight":    0.25,
			"time_weight":        0.15,
			"location_weight":    0.15,
			"history_weight":     0.10,
			"anomaly_weight":     0.05,
		},
		"switch_control": gin.H{
			"seamless_threshold":  0.7,
			"strong_threshold":    0.3,
			"high_risk_threshold": 0.8,
			"progressive_enabled": true,
			"cooldown_period":      "5m",
		},
	}

	svc.GetGlobalStats()

	response.Success(c, gin.H{
		"success": true,
		"config": config,
	})
}

func SeamlessV15HealthCheck(c *gin.Context) {
	svc := GetSeamlessV15Service()
	stats := svc.GetGlobalStats()

	health := gin.H{
		"status":             "healthy",
		"service":            "seamless_v15",
		"version":            "15.0.0",
		"total_devices":      stats["total_devices"],
		"total_users":        stats["total_users"],
		"total_trust_cache":  stats["total_cached_trust_scores"],
		"total_switches":     stats["total_switch_records"],
	}

	if stats["total_devices"] == 0 && stats["total_users"] == 0 {
		health["status"] = "initializing"
		health["message"] = "服务已初始化，等待数据积累"
	}

	response.Success(c, health)
}

func getVerificationMessage(result *service.VerificationResult) string {
	switch result.VerificationType {
	case "seamless":
		if result.Confidence > 0.8 {
			return "高置信度无缝验证通过"
		}
		return "无缝验证通过"
	case "strong":
		return "建议进行强验证"
	case "progressive":
		return fmt.Sprintf("建议进行渐进式验证（级别 %d）", result.ProgressiveLevel)
	case "block":
		return "请求被阻止，请稍后重试"
	default:
		return "验证完成"
	}
}

func SeamlessV15Decision(c *gin.Context) {
	var req struct {
		UserID      string  `json:"user_id" binding:"required"`
		Fingerprint string  `json:"fingerprint" binding:"required"`
		RiskScore   float64 `json:"risk_score"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	if req.RiskScore < 0 {
		req.RiskScore = 50.0
	}

	svc := GetSeamlessV15Service()
	decision := svc.DetermineVerificationType(req.UserID, req.Fingerprint, req.RiskScore)

	captchaType := "none"
	switch decision.RecommendedType {
	case "seamless":
		captchaType = "none"
	case "strong":
		captchaType = "slider"
	case "progressive":
		captchaType = fmt.Sprintf("progressive_level_%d", decision.ProgressiveLevel)
	case "block":
		captchaType = "blocked"
	}

	response.Success(c, gin.H{
		"success":           true,
		"verification_type": decision.RecommendedType,
		"captcha_type":      captchaType,
		"trust_score":       decision.TrustScore,
		"risk_score":        decision.RiskScore,
		"confidence":        decision.Confidence,
		"reasons":           decision.Reasons,
		"skip_captcha":      decision.RecommendedType == "seamless" && decision.Confidence > 0.7,
	})
}

type SeamlessV15LearningStatusResponse struct {
	Success         bool    `json:"success"`
	UserID          string  `json:"user_id"`
	Fingerprint     string  `json:"fingerprint"`
	LearningPhase   string  `json:"learning_phase"`
	Progress        float64 `json:"progress"`
	EstimatedTime   string  `json:"estimated_time"`
	RequiredActions []string `json:"required_actions"`
}

func SeamlessV15LearningStatus(c *gin.Context) {
	userID := c.Query("user_id")
	fingerprint := c.Query("fingerprint")

	if userID == "" && fingerprint == "" {
		response.BadRequest(c, "至少需要提供 user_id 或 fingerprint")
		return
	}

	svc := GetSeamlessV15Service()

	responseData := SeamlessV15LearningStatusResponse{
		Success:   true,
		UserID:    userID,
		Fingerprint: fingerprint,
	}

	requiredActions := make([]string, 0)

	if fingerprint != "" {
		trustResult := svc.GetTrustScore(userID, fingerprint)

		if trustResult.DeviceConfidence < 0.3 {
			responseData.LearningPhase = "device_initial"
			responseData.Progress = trustResult.DeviceConfidence * 100
			responseData.EstimatedTime = "约 5-10 次验证后完成"
			requiredActions = append(requiredActions, "继续使用当前设备进行验证")
		} else if trustResult.DeviceConfidence < 0.7 {
			responseData.LearningPhase = "device_learning"
			responseData.Progress = trustResult.DeviceConfidence * 100
			responseData.EstimatedTime = "约 3-5 次验证后完成"
			requiredActions = append(requiredActions, "保持使用习惯的一致性")
		} else {
			responseData.LearningPhase = "device_trained"
			responseData.Progress = 100
			responseData.EstimatedTime = "已完成"
		}
	}

	if userID != "" {
		trustResult := svc.GetTrustScore(userID, fingerprint)

		if trustResult.BehaviorConfidence < 0.3 {
			responseData.LearningPhase = "behavior_initial"
			responseData.Progress = math.Min(responseData.Progress, trustResult.BehaviorConfidence*100)
			if len(requiredActions) == 0 {
				responseData.EstimatedTime = "约 10-20 次验证后完成"
				requiredActions = append(requiredActions, "保持自然的行为习惯")
			}
		} else if trustResult.BehaviorConfidence < 0.7 {
			responseData.LearningPhase = "behavior_learning"
			if responseData.Progress == 0 {
				responseData.Progress = trustResult.BehaviorConfidence * 100
			} else {
				responseData.Progress = (responseData.Progress + trustResult.BehaviorConfidence*100) / 2
			}
			requiredActions = append(requiredActions, "保持使用时间和行为模式的一致性")
		}
	}

	if len(requiredActions) == 0 {
		requiredActions = append(requiredActions, "无需额外操作")
	}
	responseData.RequiredActions = requiredActions

	response.Success(c, responseData)
}

func CalculateSeamlessMetrics(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	periodStart := time.Now().AddDate(0, 0, -7)
	periodEnd := time.Now()

	if startDate != "" {
		if parsed, err := time.Parse("2006-01-02", startDate); err == nil {
			periodStart = parsed
		}
	}

	if endDate != "" {
		if parsed, err := time.Parse("2006-01-02", endDate); err == nil {
			periodEnd = parsed.Add(24*time.Hour - time.Second)
		}
	}

	var logs []models.SeamlessVerification
	database.DB.Where("created_at BETWEEN ? AND ?", periodStart, periodEnd).Find(&logs)

	total := len(logs)
	seamless := 0
	strong := 0
	block := 0
	totalRisk := 0.0
	totalTrust := 0.0

	decisionCounts := make(map[string]int)
	for _, log := range logs {
		decisionCounts[log.Decision]++
		switch log.Decision {
		case "seamless", "allow":
			seamless++
		case "strong", "challenge", "progressive":
			strong++
		case "block":
			block++
		}
		totalRisk += log.RiskScore
		totalTrust += (100 - log.RiskScore)
	}

	metrics := gin.H{
		"success":            true,
		"period_start":       periodStart,
		"period_end":         periodEnd,
		"total_verifications": total,
		"seamless_count":      seamless,
		"strong_count":       strong,
		"block_count":        block,
		"seamless_rate":       0.0,
		"block_rate":         0.0,
		"average_risk_score": 0.0,
		"average_trust_score": 0.0,
		"decision_distribution": decisionCounts,
	}

	if total > 0 {
		metrics["seamless_rate"] = float64(seamless) / float64(total) * 100
		metrics["block_rate"] = float64(block) / float64(total) * 100
		metrics["average_risk_score"] = totalRisk / float64(total)
		metrics["average_trust_score"] = totalTrust / float64(total)
	}

	response.Success(c, metrics)
}

func ExportSeamlessData(c *gin.Context) {
	format := c.DefaultQuery("format", "json")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	periodStart := time.Now().AddDate(0, 0, -30)
	periodEnd := time.Now()

	if startDate != "" {
		if parsed, err := time.Parse("2006-01-02", startDate); err == nil {
			periodStart = parsed
		}
	}

	if endDate != "" {
		if parsed, err := time.Parse("2006-01-02", endDate); err == nil {
			periodEnd = parsed.Add(24*time.Hour - time.Second)
		}
	}

	var logs []models.SeamlessVerification
	database.DB.Where("created_at BETWEEN ? AND ?", periodStart, periodEnd).
		Order("created_at DESC").
		Find(&logs)

	switch format {
	case "csv":
		csv := "ID,SessionID,Decision,RiskScore,Reason,IPAddress,UserAgent,Duration,CreatedAt\n"
		for _, log := range logs {
			csv += fmt.Sprintf("%d,%s,%s,%.2f,%s,%s,%s,%d,%s\n",
				log.ID,
				log.SessionID,
				log.Decision,
				log.RiskScore,
				log.Reason,
				log.IPAddress,
				log.UserAgent,
				log.Duration,
				log.CreatedAt.Format("2006-01-02 15:04:05"),
			)
		}
		filename := fmt.Sprintf("seamless_data_%s.csv", time.Now().Format("20060102150405"))
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.Header("Content-Type", "text/csv")
		c.Data(200, "text/csv", []byte(csv))

	default:
		response.Success(c, gin.H{
			"success":    true,
			"count":      len(logs),
			"data":       logs,
			"period":     gin.H{"start": periodStart, "end": periodEnd},
		})
	}
}

type AnalyticsDataPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	Value       float64   `json:"value"`
	Label       string    `json:"label"`
	Category    string    `json:"category"`
}

func GetSeamlessAnalytics(c *gin.Context) {
	metricType := c.DefaultQuery("type", "seamless_rate")
	granularity := c.DefaultQuery("granularity", "hour")
	days := 7

	if daysArg := c.Query("days"); daysArg != "" {
		if parsed, err := strconv.Atoi(daysArg); err == nil {
			days = parsed
		}
	}

	periodStart := time.Now().AddDate(0, 0, -days)

	var logs []models.SeamlessVerification
	database.DB.Where("created_at >= ?", periodStart).Find(&logs)

	dataPoints := make([]AnalyticsDataPoint, 0)

	switch metricType {
	case "seamless_rate":
		hourlyData := make(map[string]struct {
			total, seamless int
		})
		for _, log := range logs {
			key := log.CreatedAt.Format("2006-01-02 15:00")
			hourlyData[key] = struct {
				total, seamless int
			}{
				total:    hourlyData[key].total + 1,
				seamless: hourlyData[key].seamless + boolToInt(log.Decision == "seamless" || log.Decision == "allow"),
			}
		}
		for timeStr, data := range hourlyData {
			if parsedTime, err := time.Parse("2006-01-02 15:00", timeStr); err == nil {
				rate := 0.0
				if data.total > 0 {
					rate = float64(data.seamless) / float64(data.total) * 100
				}
				dataPoints = append(dataPoints, AnalyticsDataPoint{
					Timestamp: parsedTime,
					Value:     rate,
					Label:     timeStr,
					Category: "seamless_rate",
				})
			}
		}

	case "risk_distribution":
		riskBuckets := map[string]int{
			"0-20":   0,
			"20-40":  0,
			"40-60":  0,
			"60-80":  0,
			"80-100": 0,
		}
		for _, log := range logs {
			switch {
			case log.RiskScore < 20:
				riskBuckets["0-20"]++
			case log.RiskScore < 40:
				riskBuckets["20-40"]++
			case log.RiskScore < 60:
				riskBuckets["40-60"]++
			case log.RiskScore < 80:
				riskBuckets["60-80"]++
			default:
				riskBuckets["80-100"]++
			}
		}
		for label, value := range riskBuckets {
			dataPoints = append(dataPoints, AnalyticsDataPoint{
				Timestamp: time.Now(),
				Value:     float64(value),
				Label:     label,
				Category:  "risk_distribution",
			})
		}
	}

	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Timestamp.Before(dataPoints[j].Timestamp)
	})

	response.Success(c, gin.H{
		"success":      true,
		"metric_type":  metricType,
		"granularity":  granularity,
		"period_start": periodStart,
		"period_end":   time.Now(),
		"data_points":  dataPoints,
		"total_points": len(dataPoints),
	})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
