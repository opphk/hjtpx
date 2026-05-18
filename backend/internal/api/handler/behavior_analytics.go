package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type BehaviorAnalyticsHandler struct {
	behaviorService *service.BehaviorAnalyticsService
}

func NewBehaviorAnalyticsHandler() *BehaviorAnalyticsHandler {
	return &BehaviorAnalyticsHandler{
		behaviorService: service.NewBehaviorAnalyticsService(),
	}
}

func GetBehaviorAnalyticsHandler() *BehaviorAnalyticsHandler {
	return NewBehaviorAnalyticsHandler()
}

type BehaviorHeatmapPoint struct {
	X         int     `json:"x"`
	Y         int     `json:"y"`
	Count     int     `json:"count"`
	Intensity float64 `json:"intensity"`
}

type BehaviorTrajectory struct {
	UserID    string              `json:"userId"`
	SessionID string              `json:"sessionId"`
	Points    []BehaviorDataPoint `json:"points"`
	StartTime string              `json:"startTime"`
	EndTime   string              `json:"endTime"`
	Duration  int64               `json:"duration"`
}

type BehaviorDataPoint struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Timestamp int64  `json:"timestamp"`
	Event     string `json:"event"`
}

type AnomalyPattern struct {
	PatternID     string `json:"patternId"`
	Type          string `json:"type"`
	Description   string `json:"description"`
	Severity      string `json:"severity"`
	Count         int    `json:"count"`
	AffectedUsers int    `json:"affectedUsers"`
	FirstSeen     string `json:"firstSeen"`
	LastSeen      string `json:"lastSeen"`
}

type RiskScoreDistribution struct {
	Range      string  `json:"range"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type BehaviorSummary struct {
	TotalSessions      int64   `json:"totalSessions"`
	TotalInteractions  int64   `json:"totalInteractions"`
	AvgSessionDuration float64 `json:"avgSessionDuration"`
	AvgMouseSpeed      float64 `json:"avgMouseSpeed"`
	ClickCount         int64   `json:"clickCount"`
	KeyboardEventCount int64   `json:"keyboardEventCount"`
	AnomalyCount       int64   `json:"anomalyCount"`
	HighRiskUsers      int64   `json:"highRiskUsers"`
}

func GetBehaviorAnalytics(c *gin.Context) {
	handler := GetBehaviorAnalyticsHandler()
	period := c.DefaultQuery("period", "7d")

	summary, err := handler.behaviorService.GetBehaviorSummary(period)
	if err != nil {
		response.InternalServerError(c, "获取行为分析概览失败")
		return
	}

	heatmapData, err := handler.behaviorService.GetHeatmapData(period)
	if err != nil {
		response.InternalServerError(c, "获取热力图数据失败")
		return
	}

	trajectories, err := handler.behaviorService.GetRecentTrajectories(20)
	if err != nil {
		response.InternalServerError(c, "获取轨迹数据失败")
		return
	}

	anomalies, err := handler.behaviorService.GetAnomalyPatterns(period)
	if err != nil {
		response.InternalServerError(c, "获取异常模式失败")
		return
	}

	riskDistribution, err := handler.behaviorService.GetRiskDistribution(period)
	if err != nil {
		response.InternalServerError(c, "获取风险分布失败")
		return
	}

	sankeyData, err := handler.behaviorService.GetSankeyData(period)
	if err != nil {
		response.InternalServerError(c, "获取桑基图数据失败")
		return
	}

	radarData, err := handler.behaviorService.GetRadarData(period)
	if err != nil {
		response.InternalServerError(c, "获取雷达图数据失败")
		return
	}

	response.Success(c, gin.H{
		"summary":          summary,
		"heatmap":          heatmapData,
		"trajectories":     trajectories,
		"anomalies":        anomalies,
		"riskDistribution": riskDistribution,
		"sankeyData":       sankeyData,
		"radarData":        radarData,
	})
}

func GetBehaviorHeatmap(c *gin.Context) {
	handler := GetBehaviorAnalyticsHandler()
	period := c.DefaultQuery("period", "7d")

	heatmapData, err := handler.behaviorService.GetHeatmapData(period)
	if err != nil {
		response.InternalServerError(c, "获取热力图数据失败")
		return
	}

	response.Success(c, heatmapData)
}

func GetBehaviorTrajectories(c *gin.Context) {
	handler := GetBehaviorAnalyticsHandler()
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := time.ParseDuration(l); err == nil {
			limit = int(parsed)
		}
	}

	trajectories, err := handler.behaviorService.GetRecentTrajectories(limit)
	if err != nil {
		response.InternalServerError(c, "获取轨迹数据失败")
		return
	}

	response.Success(c, trajectories)
}

func GetAnomalyPatterns(c *gin.Context) {
	handler := GetBehaviorAnalyticsHandler()
	period := c.DefaultQuery("period", "7d")

	anomalies, err := handler.behaviorService.GetAnomalyPatterns(period)
	if err != nil {
		response.InternalServerError(c, "获取异常模式失败")
		return
	}

	response.Success(c, anomalies)
}

func GetRiskScoreDistribution(c *gin.Context) {
	handler := GetBehaviorAnalyticsHandler()
	period := c.DefaultQuery("period", "7d")

	riskDistribution, err := handler.behaviorService.GetRiskDistribution(period)
	if err != nil {
		response.InternalServerError(c, "获取风险分布失败")
		return
	}

	response.Success(c, riskDistribution)
}

func ExportBehaviorData(c *gin.Context) {
	handler := GetBehaviorAnalyticsHandler()
	format := c.DefaultQuery("format", "csv")
	period := c.DefaultQuery("period", "7d")

	data, err := handler.behaviorService.ExportBehaviorData(format, period)
	if err != nil {
		response.InternalServerError(c, "导出数据失败")
		return
	}

	filename := "behavior_analytics"
	switch format {
	case "csv":
		filename += ".csv"
		c.Header("Content-Type", "text/csv")
	case "json":
		filename += ".json"
		c.Header("Content-Type", "application/json")
	default:
		filename += ".csv"
		c.Header("Content-Type", "text/csv")
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(200, "text/csv", data)
}

func (h *BehaviorAnalyticsHandler) getRecentBehaviorLogs(limit int) ([]models.BehaviorData, error) {
	var logs []models.BehaviorData
	err := database.DB.Order("created_at DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

// Top-level functions for router
func GetUserTrajectories(c *gin.Context) {
	GetBehaviorTrajectories(c)
}

func GetBehaviorAnomalies(c *gin.Context) {
	GetAnomalyPatterns(c)
}

func ReplayTrajectory(c *gin.Context) {
	// Temporary placeholder implementation
	response.Success(c, gin.H{"message": "Replay functionality not implemented yet"})
}
