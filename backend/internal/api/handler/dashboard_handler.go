package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type DashboardHandler struct {
	dashboardService *service.DashboardService
}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{
		dashboardService: service.NewDashboardService(),
	}
}

func GetDashboardHandler() *DashboardHandler {
	return NewDashboardHandler()
}

func GetDashboardData(c *gin.Context) {
	handler := GetDashboardHandler()
	period := c.DefaultQuery("period", "hour")

	data, err := handler.dashboardService.GetDashboardData(period)
	if err != nil {
		response.InternalServerError(c, "获取仪表盘数据失败")
		return
	}

	response.Success(c, data)
}

func ExportDashboardData(c *gin.Context) {
	handler := GetDashboardHandler()

	format := c.DefaultQuery("format", "csv")
	period := c.DefaultQuery("period", "month")

	data, err := handler.dashboardService.ExportData(format, period)
	if err != nil {
		response.InternalServerError(c, "导出数据失败")
		return
	}

	filename := "dashboard_export"
	switch format {
	case "csv":
		filename += ".csv"
		c.Header("Content-Type", "text/csv")
	case "json":
		filename += ".json"
		c.Header("Content-Type", "application/json")
	case "excel":
		filename += ".xlsx"
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	default:
		filename += ".json"
		c.Header("Content-Type", "application/json")
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "text/csv", data)
}

func GetRecentVerifications(c *gin.Context) {
	handler := GetDashboardHandler()

	verifications, err := handler.getRecentVerifications(10)
	if err != nil {
		response.InternalServerError(c, "获取最近验证记录失败")
		return
	}

	response.Success(c, verifications)
}

func (h *DashboardHandler) getRecentVerifications(limit int) ([]map[string]interface{}, error) {
	verifications := make([]map[string]interface{}, 0)

	rows, err := database.DB.Table("verifications").
		Select("verifications.created_at, applications.name as app_name, verifications.captcha_type, verifications.status").
		Joins("LEFT JOIN applications ON verifications.application_id = applications.id").
		Order("verifications.created_at DESC").
		Limit(limit).
		Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	typeMap := map[string]string{
		"slider":   "滑动验证",
		"click":    "点选验证",
		"image":    "图片验证",
		"text":     "文字验证",
		"gesture":  "手势验证",
	}

	for rows.Next() {
		var createdAt string
		var appName, captchaType, status string

		if err := rows.Scan(&createdAt, &appName, &captchaType, &status); err != nil {
			continue
		}

		if mapped, ok := typeMap[captchaType]; ok {
			captchaType = mapped
		}

		verifications = append(verifications, map[string]interface{}{
			"time":   createdAt,
			"app":    appName,
			"type":   captchaType,
			"status": status,
		})
	}

	return verifications, nil
}

func GetDashboardAlerts(c *gin.Context) {
	handler := GetDashboardHandler()

	alerts := handler.dashboardService.CheckAlerts()

	response.Success(c, gin.H{
		"alerts": alerts,
	})
}
