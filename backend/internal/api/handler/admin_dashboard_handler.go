package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var (
	adminDashboardUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	adminDashboardClients   = make(map[*websocket.Conn]bool)
	adminDashboardClientsMu sync.RWMutex

	verificationEventChannel chan service.VerificationEvent
	metricsBroadcastChannel  chan *service.DashboardMetrics
)

type AdminDashboardHandler struct {
	dashboardService *service.AdminDashboardService
}

func NewAdminDashboardHandler() *AdminDashboardHandler {
	verificationEventChannel = make(chan service.VerificationEvent, 1000)
	metricsBroadcastChannel = make(chan *service.DashboardMetrics, 100)

	return &AdminDashboardHandler{
		dashboardService: service.GetAdminDashboardService(),
	}
}

func GetAdminDashboardHandler() *AdminDashboardHandler {
	return NewAdminDashboardHandler()
}

func GetAdminDashboardMetrics(c *gin.Context) {
	handler := GetAdminDashboardHandler()
	ctx := context.Background()

	metrics, err := handler.dashboardService.GetDashboardMetrics(ctx)
	if err != nil {
		response.InternalServerError(c, "获取仪表盘数据失败")
		return
	}

	response.Success(c, metrics)
}

func GetAdminDashboardRealtime(c *gin.Context) {
	handler := GetAdminDashboardHandler()
	ctx := context.Background()

	metrics, err := handler.dashboardService.GetRealtimeMetrics(ctx)
	if err != nil {
		response.InternalServerError(c, "获取实时数据失败")
		return
	}

	response.Success(c, metrics)
}

func GetAdminDashboardTrend(c *gin.Context) {
	handler := GetAdminDashboardHandler()
	ctx := context.Background()

	period := c.DefaultQuery("period", "hour")
	trend := handler.dashboardService.GetTrendMetrics(ctx, period)

	response.Success(c, trend)
}

func GetAdminDashboardAlerts(c *gin.Context) {
	handler := GetAdminDashboardHandler()
	ctx := context.Background()

	alerts := handler.dashboardService.CheckAlerts(ctx)

	response.Success(c, gin.H{
		"alerts": alerts,
	})
}

func ExportAdminDashboardData(c *gin.Context) {
	handler := GetAdminDashboardHandler()
	ctx := context.Background()

	format := c.DefaultQuery("format", "csv")
	period := c.DefaultQuery("period", "today")

	data, filename, err := handler.dashboardService.ExportData(ctx, format, period)
	if err != nil {
		response.InternalServerError(c, "导出数据失败")
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	switch format {
	case "csv":
		c.Header("Content-Type", "text/csv; charset=utf-8")
	case "excel":
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	case "pdf":
		c.Header("Content-Type", "application/pdf")
	case "json":
		c.Header("Content-Type", "application/json")
	default:
		c.Header("Content-Type", "application/octet-stream")
	}

	c.Data(http.StatusOK, c.GetHeader("Content-Type"), data)
}

func GenerateAdminDashboardReport(c *gin.Context) {
	handler := GetAdminDashboardHandler()
	ctx := context.Background()

	var req struct {
		Name   string `json:"name" binding:"required"`
		Type   string `json:"type" binding:"required"`
		Period string `json:"period" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	data, filename, err := handler.dashboardService.GenerateReport(ctx, req.Name, req.Type, req.Period)
	if err != nil {
		response.InternalServerError(c, "生成报表失败")
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Data(http.StatusOK, c.GetHeader("Content-Type"), data)
}

func AdminDashboardWebSocketHandler(c *gin.Context) {
	conn, err := adminDashboardUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	adminDashboardClientsMu.Lock()
	adminDashboardClients[conn] = true
	adminDashboardClientsMu.Unlock()

	go handleAdminDashboardWebSocket(conn)
}

func handleAdminDashboardWebSocket(conn *websocket.Conn) {
	defer func() {
		adminDashboardClientsMu.Lock()
		delete(adminDashboardClients, conn)
		adminDashboardClientsMu.Unlock()
		conn.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go broadcastMetricsToClient(ctx, conn)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-verificationEventChannel:
			data := map[string]interface{}{
				"type":      "verification",
				"timestamp": event.Timestamp.Unix(),
				"payload": map[string]interface{}{
					"session_id":    event.SessionID,
					"captcha_type":  event.CaptchaType,
					"status":        event.Status,
					"risk_score":    event.RiskScore,
					"ip_address":    event.IPAddress,
					"response_time": event.ResponseTime,
				},
			}
			msg, _ := json.Marshal(data)
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case metrics := <-metricsBroadcastChannel:
			data := map[string]interface{}{
				"type":      "metrics",
				"timestamp": time.Now().Unix(),
				"payload": map[string]interface{}{
					"qps":               metrics.Extended.CurrentQPS,
					"active_connections": metrics.Extended.ActiveConnections,
					"cpu_usage":         metrics.Extended.CPUUsage,
					"memory_usage":      metrics.Extended.MemoryUsage,
					"cache_hit_rate":    metrics.Extended.CacheHitRate,
				},
			}
			msg, _ := json.Marshal(data)
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

func broadcastMetricsToClient(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			handler := GetAdminDashboardHandler()
			metrics, err := handler.dashboardService.GetRealtimeMetrics(ctx)
			if err != nil {
				continue
			}

			data := map[string]interface{}{
				"type":      "metrics",
				"timestamp": time.Now().Unix(),
				"payload": map[string]interface{}{
					"qps":               metrics.QPS,
					"active_connections": metrics.ActiveConnections,
					"cpu_usage":         metrics.CPUUsage,
					"memory_usage":      metrics.MemoryUsage,
					"cache_hit_rate":    metrics.CacheHitRate,
					"total_requests":    metrics.QPS * 100,
				},
			}
			msg, _ := json.Marshal(data)
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}
}

func BroadcastVerificationEvent(event service.VerificationEvent) {
	select {
	case verificationEventChannel <- event:
	default:
	}
}

func BroadcastDashboardMetrics(metrics *service.DashboardMetrics) {
	select {
	case metricsBroadcastChannel <- metrics:
	default:
	}
}

func GetDashboardAlertsList(c *gin.Context) {
	handler := GetAdminDashboardHandler()
	ctx := context.Background()

	alerts := handler.dashboardService.CheckAlerts(ctx)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"alerts": alerts,
			"count":  len(alerts),
		},
	})
}

func PublishTestVerificationEvent(c *gin.Context) {
	var req struct {
		SessionID   string  `json:"session_id"`
		CaptchaType string  `json:"captcha_type"`
		Status      string  `json:"status"`
		RiskScore   float64 `json:"risk_score"`
		IPAddress   string  `json:"ip_address"`
		ResponseTime int64  `json:"response_time"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	event := service.VerificationEvent{
		Timestamp:   time.Now(),
		SessionID:   req.SessionID,
		CaptchaType: req.CaptchaType,
		Status:      req.Status,
		RiskScore:   req.RiskScore,
		IPAddress:   req.IPAddress,
		ResponseTime: req.ResponseTime,
	}

	BroadcastVerificationEvent(event)

	response.Success(c, gin.H{
		"message": "事件已发布",
	})
}
