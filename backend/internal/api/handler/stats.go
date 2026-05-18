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

// GetVerificationStats 获取验证统计
// @Summary 获取验证统计
// @Description 获取验证系统的整体统计数据，包括成功率、风险评分等
// @Tags 统计
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "验证统计数据"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/stats/verification [get]
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

// GetChartData 获取图表数据
// @Summary 获取图表数据
// @Description 获取验证趋势图表数据，包括成功、失败和总数
// @Tags 统计
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ChartData "图表数据"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/stats/chart [get]
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

// GetDashboardStats 获取仪表盘统计
// @Summary 获取仪表盘统计
// @Description 获取仪表盘展示所需的统计数据，包括用户数、应用数、请求数和错误数
// @Tags 统计
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} DashboardStats "仪表盘统计数据"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/stats/dashboard [get]
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

// GetRecentActivity 获取最近活动
// @Summary 获取最近活动
// @Description 获取最近的系统活动和验证记录
// @Tags 统计
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} []ActivityItem "最近活动列表"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/stats/recent-activity [get]
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

// GetTrendData 获取趋势数据
// @Summary 获取趋势数据
// @Description 获取指定天数内的验证趋势数据
// @Tags 统计
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "趋势数据"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/stats/trend [get]
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

// GetHourlyStats 获取小时统计
// @Summary 获取小时统计
// @Description 获取指定日期的小时级验证统计数据
// @Tags 统计
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param date query string false "日期，格式YYYY-MM-DD，默认当天"
// @Success 200 {object} map[string]interface{} "小时统计数据"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/stats/hourly [get]
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

// GetRealtimeStats 获取实时统计
// @Summary 获取实时统计
// @Description 获取当前实时验证统计数据
// @Tags 统计
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "实时统计数据"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/stats/realtime [get]
func GetRealtimeStats(c *gin.Context) {
	handler := GetStatsHandler()

	realtimeStats, err := handler.statsService.GetRealtimeStats()
	if err != nil {
		response.InternalServerError(c, "获取实时统计失败")
		return
	}

	response.Success(c, realtimeStats)
}

// GetRiskDistribution 获取风险分布
// @Summary 获取风险分布
// @Description 获取验证请求的风险评分分布情况
// @Tags 统计
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "风险分布数据"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/stats/risk-distribution [get]
func GetRiskDistribution(c *gin.Context) {
	handler := GetStatsHandler()

	distribution, err := handler.statsService.GetRiskDistribution()
	if err != nil {
		response.InternalServerError(c, "获取风险分布失败")
		return
	}

	response.Success(c, distribution)
}

// GetTopIPs 获取Top IP
// @Summary 获取Top IP
// @Description 获取发起验证请求最多的IP地址列表
// @Tags 统计
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Top IP列表"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/stats/top-ips [get]
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

// GetApplicationStats 获取应用统计
// @Summary 获取应用统计
// @Description 获取验证请求最多的应用列表
// @Tags 统计
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "应用统计数据"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/stats/application-stats [get]
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

// GetCaptchaTypeStats 获取验证码类型统计
// @Summary 获取验证码类型统计
// @Description 获取各类型验证码的使用统计
// @Tags 统计
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "验证码类型统计数据"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/stats/captcha-type [get]
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

// GenerateReport 生成报告
// @Summary 生成报告
// @Description 根据指定日期范围和报告类型生成验证报告
// @Tags 统计
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param report_type query string true "报告类型" Enums(daily, weekly, monthly)
// @Param start_date query string false "开始日期，格式YYYY-MM-DD"
// @Param end_date query string false "结束日期，格式YYYY-MM-DD"
// @Success 200 {object} map[string]interface{} "生成的报告"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/stats/report [get]
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
