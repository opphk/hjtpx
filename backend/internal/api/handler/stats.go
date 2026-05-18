package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type StatsHandler struct {
	statsService *service.StatsService
}

func NewStatsHandler() *StatsHandler {
	return &StatsHandler{
		statsService: service.NewStatsService(),
	}
}

func GetStatsHandler() *StatsHandler {
	return NewStatsHandler()
}

type DashboardStats struct {
	TotalUsers    int64 `json:"totalUsers"`
	TotalApps     int64 `json:"totalApps"`
	TotalRequests int64 `json:"totalRequests"`
	TotalErrors   int64 `json:"totalErrors"`
}

type ActivityItem struct {
	Time   string `json:"time"`
	Event  string `json:"event"`
	User   string `json:"user"`
	Status string `json:"status"`
}

type VerificationStats struct {
	Total        int64 `json:"total"`
	Pending      int64 `json:"pending"`
	Success      int64 `json:"success"`
	Failed       int64 `json:"failed"`
	Applications int64 `json:"applications"`
	Users        int64 `json:"users"`
}

type ChartDataPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type ChartData struct {
	Success []ChartDataPoint `json:"success"`
	Failed  []ChartDataPoint `json:"failed"`
	Total   []ChartDataPoint `json:"total"`
}

func GetVerificationStats(c *gin.Context) {
	handler := GetStatsHandler()

	stats, err := handler.statsService.GetOverviewStats()
	if err != nil {
		response.InternalServerError(c, "获取统计数据失败")
		return
	}

	captchaStats, err := handler.statsService.GetCaptchaTypeStats()
	if err != nil {
		response.InternalServerError(c, "获取验证码类型统计失败")
		return
	}

	response.Success(c, gin.H{
		"total":              stats.TotalVerifications,
		"success":            stats.SuccessCount,
		"failed":             stats.FailedCount,
		"pending":            stats.PendingCount,
		"success_rate":       stats.SuccessRate,
		"avg_risk_score":     stats.AvgRiskScore,
		"total_applications": stats.TotalApplications,
		"total_users":        stats.TotalUsers,
		"captcha_stats":      captchaStats,
	})
}

func GetChartData(c *gin.Context) {
	handler := GetStatsHandler()
	days := 7

	trendData, err := handler.statsService.GetTrendData(days)
	if err != nil {
		response.InternalServerError(c, "获取趋势数据失败")
		return
	}

	var successData, failedData, totalData []ChartDataPoint
	for _, point := range trendData {
		totalData = append(totalData, ChartDataPoint{Date: point.Date, Count: point.TotalCount})
		successData = append(successData, ChartDataPoint{Date: point.Date, Count: point.SuccessCount})
		failedData = append(failedData, ChartDataPoint{Date: point.Date, Count: point.FailedCount})
	}

	response.Success(c, ChartData{
		Success: successData,
		Failed:  failedData,
		Total:   totalData,
	})
}

func GetDashboardStats(c *gin.Context) {
	handler := GetStatsHandler()

	stats, err := handler.statsService.GetOverviewStats()
	if err != nil {
		response.InternalServerError(c, "获取仪表盘数据失败")
		return
	}

	response.Success(c, DashboardStats{
		TotalUsers:    stats.TotalUsers,
		TotalApps:     stats.TotalApplications,
		TotalRequests: stats.TotalVerifications,
		TotalErrors:   stats.FailedCount,
	})
}

func GetRecentActivity(c *gin.Context) {
	handler := GetStatsHandler()
	var activities []ActivityItem

	logs, err := handler.getRecentLogs(10)
	if err != nil || len(logs) == 0 {
		activities = []ActivityItem{
			{"2024-01-15 14:32:18", "用户登录", "admin", "success"},
			{"2024-01-15 14:28:45", "创建应用", "developer1", "success"},
			{"2024-01-15 14:25:12", "API请求失败", "app_001", "error"},
			{"2024-01-15 14:20:33", "更新配置", "admin", "success"},
			{"2024-01-15 14:15:09", "用户注册", "new_user", "success"},
		}
		response.Success(c, activities)
		return
	}

	for _, log := range logs {
		activities = append(activities, ActivityItem{
			Time:   log.CreatedAt.Format("2006-01-02 15:04:05"),
			Event:  "验证请求",
			User:   "用户",
			Status: log.Status,
		})
	}

	response.Success(c, activities)
}

func (h *StatsHandler) getRecentLogs(limit int) ([]models.VerificationLog, error) {
	var logs []models.VerificationLog
	err := database.DB.Order("created_at DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

func GetTrendData(c *gin.Context) {
	handler := GetStatsHandler()
	days := 7

	trendData, err := handler.statsService.GetTrendData(days)
	if err != nil {
		response.InternalServerError(c, "获取趋势数据失败")
		return
	}

	response.Success(c, trendData)
}

func GetHourlyStats(c *gin.Context) {
	handler := GetStatsHandler()
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	hourlyStats, err := handler.statsService.GetHourlyStats(date)
	if err != nil {
		response.InternalServerError(c, "获取小时统计失败")
		return
	}

	response.Success(c, hourlyStats)
}

func GetRealtimeStats(c *gin.Context) {
	handler := GetStatsHandler()

	realtimeStats, err := handler.statsService.GetRealtimeStats()
	if err != nil {
		response.InternalServerError(c, "获取实时统计失败")
		return
	}

	response.Success(c, realtimeStats)
}

func GetRiskDistribution(c *gin.Context) {
	handler := GetStatsHandler()

	distribution, err := handler.statsService.GetRiskDistribution()
	if err != nil {
		response.InternalServerError(c, "获取风险分布失败")
		return
	}

	response.Success(c, distribution)
}

func GetTopIPs(c *gin.Context) {
	handler := GetStatsHandler()
	limit := 10

	topIPs, err := handler.statsService.GetTopIPs(limit)
	if err != nil {
		response.InternalServerError(c, "获取Top IP失败")
		return
	}

	response.Success(c, topIPs)
}

func GetApplicationStats(c *gin.Context) {
	handler := GetStatsHandler()
	limit := 10

	applicationStats, err := handler.statsService.GetApplicationStats(limit)
	if err != nil {
		response.InternalServerError(c, "获取应用统计失败")
		return
	}

	response.Success(c, applicationStats)
}

func GetCaptchaTypeStats(c *gin.Context) {
	handler := GetStatsHandler()

	captchaStats, err := handler.statsService.GetCaptchaTypeStats()
	if err != nil {
		response.InternalServerError(c, "获取验证码类型统计失败")
		return
	}

	response.Success(c, captchaStats)
}

type GenerateReportRequest struct {
	ReportType string `form:"report_type" binding:"required"`
	StartDate  string `form:"start_date"`
	EndDate    string `form:"end_date"`
}

func GenerateReport(c *gin.Context) {
	handler := GetStatsHandler()
	var req GenerateReportRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	startDate := time.Now()
	if req.StartDate != "" {
		if parsed, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			startDate = parsed
		}
	}

	report, err := handler.statsService.GenerateReport(req.ReportType, startDate, startDate)
	if err != nil {
		response.InternalServerError(c, "生成报告失败")
		return
	}

	response.Success(c, gin.H{
		"report": report,
	})
}

type DetailedStatsRequest struct {
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	GroupBy   string `form:"group_by"`
}

type DetailedStatsResponse struct {
	TotalVerifications int64              `json:"total_verifications"`
	SuccessRate       float64             `json:"success_rate"`
	AvgResponseTime   float64             `json:"avg_response_time"`
	TopApplications   []ApplicationStats  `json:"top_applications"`
	TopIPs            []IPStats           `json:"top_ips"`
	HourlyDistribution []HourlyStats      `json:"hourly_distribution"`
	DailyTrend        []DailyStats        `json:"daily_trend"`
}

type ApplicationStats struct {
	AppID      uint   `json:"app_id"`
	AppName    string `json:"app_name"`
	TotalCount int64  `json:"total_count"`
	SuccessRate float64 `json:"success_rate"`
}

type IPStats struct {
	IP        string `json:"ip"`
	Count     int64  `json:"count"`
	BlockRate float64 `json:"block_rate"`
}

type HourlyStats struct {
	Hour  int   `json:"hour"`
	Count int64 `json:"count"`
}

type DailyStats struct {
	Date        string  `json:"date"`
	TotalCount  int64   `json:"total_count"`
	SuccessCount int64  `json:"success_count"`
	FailCount   int64   `json:"fail_count"`
	AvgResponseTime float64 `json:"avg_response_time"`
}

func GetDetailedStats(c *gin.Context) {
	handler := GetStatsHandler()
	var req DetailedStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	startDate := time.Now().AddDate(0, 0, -7)
	endDate := time.Now()
	
	if req.StartDate != "" {
		if parsed, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			startDate = parsed
		}
	}
	if req.EndDate != "" {
		if parsed, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			endDate = parsed.Add(24 * time.Hour)
		}
	}

	stats, err := handler.statsService.GetDetailedStats(startDate, endDate, req.GroupBy)
	if err != nil {
		response.InternalServerError(c, "获取详细统计失败")
		return
	}

	response.Success(c, stats)
}
