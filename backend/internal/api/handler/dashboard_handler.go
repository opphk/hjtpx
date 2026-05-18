package handler

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var (
	dashboardUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	dashboardClients   = make(map[*websocket.Conn]bool)
	dashboardClientsMu sync.RWMutex
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

// GetDashboardData 获取仪表盘数据
// @Summary 获取仪表盘数据
// @Description 获取仪表盘统计数据，包括验证次数、成功率、风险评分等
// @Tags 仪表盘
// @Accept json
// @Produce json
// @Param period query string false "时间周期：hour, day, week, month，默认hour"
// @Success 200 {object} response.Response{data=service.DashboardData} "获取成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/dashboard [get]
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

// ExportDashboardData 导出仪表盘数据
// @Summary 导出仪表盘数据
// @Description 导出仪表盘数据，支持CSV、JSON、Excel格式
// @Tags 仪表盘
// @Accept json
// @Produce json/csv
// @Param format query string false "导出格式：csv, json, excel，默认csv"
// @Param period query string false "时间周期：hour, day, week, month，默认month"
// @Success 200 {file} file "导出文件"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/dashboard/export [get]
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

// GetRecentVerifications 获取最近验证记录
// @Summary 获取最近验证记录
// @Description 获取最近的验证记录列表
// @Tags 仪表盘
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "获取成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/dashboard/recent [get]
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
		"slider":  "滑动验证",
		"click":   "点选验证",
		"image":   "图片验证",
		"text":    "文字验证",
		"gesture": "手势验证",
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

// GetDashboardAlerts 获取仪表盘告警
// @Summary 获取仪表盘告警
// @Description 获取仪表盘显示的告警信息
// @Tags 仪表盘
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/admin/dashboard/alerts [get]
func GetDashboardAlerts(c *gin.Context) {
	handler := GetDashboardHandler()

	alerts := handler.dashboardService.CheckAlerts()

	response.Success(c, gin.H{
		"alerts": alerts,
	})
}

// GetExtendedDashboardStats 获取扩展统计数据
// @Summary 获取扩展统计数据
// @Description 获取扩展的仪表盘统计数据
// @Tags 仪表盘
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "获取成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/dashboard/extended-stats [get]
func GetExtendedDashboardStats(c *gin.Context) {
	handler := GetDashboardHandler()

	stats, err := handler.dashboardService.GetExtendedStats()
	if err != nil {
		response.InternalServerError(c, "获取扩展统计失败")
		return
	}

	response.Success(c, stats)
}

// GetAttackTypeDistribution 获取攻击类型分布
// @Summary 获取攻击类型分布
// @Description 获取各类攻击类型的分布统计
// @Tags 仪表盘
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "获取成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/dashboard/attack-distribution [get]
func GetAttackTypeDistribution(c *gin.Context) {
	handler := GetDashboardHandler()

	distribution, err := handler.dashboardService.GetAttackTypeDistribution()
	if err != nil {
		response.InternalServerError(c, "获取攻击类型分布失败")
		return
	}

	response.Success(c, distribution)
}

// GetDashboardRiskScoreDistribution 获取风险评分分布
// @Summary 获取风险评分分布
// @Description 获取风险评分的分布统计
// @Tags 仪表盘
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "获取成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/dashboard/risk-distribution [get]
func GetDashboardRiskScoreDistribution(c *gin.Context) {
	handler := GetDashboardHandler()

	distribution, err := handler.dashboardService.GetRiskScoreDistribution()
	if err != nil {
		response.InternalServerError(c, "获取风险评分分布失败")
		return
	}

	response.Success(c, distribution)
}

// DashboardWebSocketHandler 仪表盘WebSocket连接
// @Summary 仪表盘WebSocket连接
// @Description 建立WebSocket连接，实时接收验证事件通知
// @Tags 仪表盘
// @Accept json
// @Produce json
// @Success 101 {string} string "WebSocket连接建立成功"
// @Router /api/v1/admin/dashboard/ws [get]
func DashboardWebSocketHandler(c *gin.Context) {
	conn, err := dashboardUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	dashboardClientsMu.Lock()
	dashboardClients[conn] = true
	dashboardClientsMu.Unlock()

	defer func() {
		dashboardClientsMu.Lock()
		delete(dashboardClients, conn)
		dashboardClientsMu.Unlock()
		conn.Close()
	}()

	go func() {
		for {
			select {
			case event := <-service.SubscribeToVerificationEvents():
				data, _ := json.Marshal(map[string]interface{}{
					"type":         "verification",
					"timestamp":    event.Timestamp.Unix(),
					"session_id":   event.SessionID,
					"captcha_type": event.CaptchaType,
					"status":       event.Status,
					"risk_score":   event.RiskScore,
					"ip_address":   event.IPAddress,
				})
				conn.WriteMessage(websocket.TextMessage, data)
			case <-time.After(5 * time.Second):
				conn.WriteMessage(websocket.PingMessage, nil)
			}
		}
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func BroadcastDashboardUpdate(data interface{}) {
	dashboardClientsMu.RLock()
	defer dashboardClientsMu.RUnlock()

	msg, _ := json.Marshal(map[string]interface{}{
		"type":      "update",
		"data":      data,
		"timestamp": time.Now().Unix(),
	})

	for client := range dashboardClients {
		client.WriteMessage(websocket.TextMessage, msg)
	}
}
