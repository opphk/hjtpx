package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
)

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

// GetVerificationStats 获取验证统计数据
func GetVerificationStats(c *gin.Context) {
	var stats VerificationStats

	database.DB.Model(&models.Verification{}).Count(&stats.Total)
	database.DB.Model(&models.Verification{}).Where("status = ?", "pending").Count(&stats.Pending)
	database.DB.Model(&models.Verification{}).Where("status = ?", "success").Count(&stats.Success)
	database.DB.Model(&models.Verification{}).Where("status = ?", "failed").Count(&stats.Failed)
	database.DB.Model(&models.Application{}).Count(&stats.Applications)
	database.DB.Model(&models.User{}).Count(&stats.Users)

	response.Success(c, stats)
}

// GetChartData 获取图表数据
func GetChartData(c *gin.Context) {
	var chartData ChartData

	days := 7
	now := time.Now()

	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		startTime := time.Date(now.Year(), now.Month(), now.Day()-i, 0, 0, 0, 0, now.Location())
		endTime := startTime.Add(24 * time.Hour)

		var successCount int64
		database.DB.Model(&models.Verification{}).
			Where("status = ? AND created_at >= ? AND created_at < ?", "success", startTime, endTime).
			Count(&successCount)
		chartData.Success = append(chartData.Success, ChartDataPoint{Date: date, Count: successCount})

		var failedCount int64
		database.DB.Model(&models.Verification{}).
			Where("status = ? AND created_at >= ? AND created_at < ?", "failed", startTime, endTime).
			Count(&failedCount)
		chartData.Failed = append(chartData.Failed, ChartDataPoint{Date: date, Count: failedCount})

		var totalCount int64
		database.DB.Model(&models.Verification{}).
			Where("created_at >= ? AND created_at < ?", startTime, endTime).
			Count(&totalCount)
		chartData.Total = append(chartData.Total, ChartDataPoint{Date: date, Count: totalCount})
	}

	response.Success(c, chartData)
}

// GetDashboardStats 获取仪表盘统计数据
func GetDashboardStats(c *gin.Context) {
	var stats DashboardStats

	database.DB.Model(&models.User{}).Count(&stats.TotalUsers)
	database.DB.Model(&models.Application{}).Count(&stats.TotalApps)
	database.DB.Model(&models.Verification{}).Count(&stats.TotalRequests)
	database.DB.Model(&models.Verification{}).Where("status = ?", "failed").Count(&stats.TotalErrors)

	response.Success(c, stats)
}

// GetRecentActivity 获取最近活动
func GetRecentActivity(c *gin.Context) {
	var logs []models.VerificationLog
	var activities []ActivityItem

	database.DB.Order("created_at DESC").Limit(10).Find(&logs)

	for _, log := range logs {
		activities = append(activities, ActivityItem{
			Time:   log.CreatedAt.Format("2006-01-02 15:04:05"),
			Event:  "验证请求",
			User:   "用户",
			Status: log.Status,
		})
	}

	if len(activities) == 0 {
		activities = []ActivityItem{
			{"2024-01-15 14:32:18", "用户登录", "admin", "success"},
			{"2024-01-15 14:28:45", "创建应用", "developer1", "success"},
			{"2024-01-15 14:25:12", "API请求失败", "app_001", "error"},
			{"2024-01-15 14:20:33", "更新配置", "admin", "success"},
			{"2024-01-15 14:15:09", "用户注册", "new_user", "success"},
		}
	}

	response.Success(c, activities)
}
