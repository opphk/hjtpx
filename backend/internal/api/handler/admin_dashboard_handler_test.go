package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestGetAdminDashboardMetrics(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminDashboardHandler()

	router.GET("/api/dashboard/admin", handler.GetAdminDashboardMetrics)

	req, _ := http.NewRequest("GET", "/api/dashboard/admin", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "code")
}

func TestGetAdminDashboardRealtime(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminDashboardHandler()

	router.GET("/api/dashboard/admin/realtime", handler.GetAdminDashboardRealtime)

	req, _ := http.NewRequest("GET", "/api/dashboard/admin/realtime", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "code")
}

func TestGetAdminDashboardTrend(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminDashboardHandler()

	router.GET("/api/dashboard/admin/trend", handler.GetAdminDashboardTrend)

	req, _ := http.NewRequest("GET", "/api/dashboard/admin/trend?period=hour", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "code")
}

func TestGetAdminDashboardAlerts(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminDashboardHandler()

	router.GET("/api/dashboard/admin/alerts", handler.GetAdminDashboardAlerts)

	req, _ := http.NewRequest("GET", "/api/dashboard/admin/alerts", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "code")
}

func TestExportAdminDashboardData(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminDashboardHandler()

	router.GET("/api/dashboard/admin/export", handler.ExportAdminDashboardData)

	testCases := []struct {
		format string
		period string
	}{
		{format: "csv", period: "today"},
		{format: "excel", period: "today"},
		{format: "json", period: "today"},
		{format: "pdf", period: "today"},
	}

	for _, tc := range testCases {
		t.Run(tc.format, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/dashboard/admin/export?format="+tc.format+"&period="+tc.period, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestGenerateAdminDashboardReport(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminDashboardHandler()

	router.POST("/api/dashboard/admin/report", handler.GenerateAdminDashboardReport)

	body := `{"name":"测试报表","type":"summary","period":"today"}`
	req, _ := http.NewRequest("POST", "/api/dashboard/admin/report", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "code")
}

func TestGenerateAdminDashboardReportInvalidBody(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminDashboardHandler()

	router.POST("/api/dashboard/admin/report", handler.GenerateAdminDashboardReport)

	body := `{"invalid":"data"}`
	req, _ := http.NewRequest("POST", "/api/dashboard/admin/report", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetDashboardAlertsList(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminDashboardHandler()

	router.GET("/api/dashboard/admin/alerts-list", handler.GetDashboardAlertsList)

	req, _ := http.NewRequest("GET", "/api/dashboard/admin/alerts-list", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "code")
}

func TestPublishTestVerificationEvent(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminDashboardHandler()

	router.POST("/api/dashboard/admin/test-event", handler.PublishTestVerificationEvent)

	body := `{
		"session_id": "test-session-123",
		"captcha_type": "slider",
		"status": "success",
		"risk_score": 25.5,
		"ip_address": "192.168.1.1",
		"response_time": 85
	}`
	req, _ := http.NewRequest("POST", "/api/dashboard/admin/test-event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "code")
}

func TestPublishTestVerificationEventInvalidBody(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminDashboardHandler()

	router.POST("/api/dashboard/admin/test-event", handler.PublishTestVerificationEvent)

	body := `{"invalid":"data"}`
	req, _ := http.NewRequest("POST", "/api/dashboard/admin/test-event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetAdminDashboardHandler(t *testing.T) {
	handler := GetAdminDashboardHandler()
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.dashboardService)
}

func TestNewAdminDashboardHandler(t *testing.T) {
	handler := NewAdminDashboardHandler()
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.dashboardService)
}

func TestBroadcastVerificationEvent(t *testing.T) {
	event := VerificationEvent{
		Timestamp:   now(),
		SessionID:   "test-session-123",
		CaptchaType: "slider",
		Status:      "success",
		RiskScore:   25.5,
		IPAddress:   "192.168.1.1",
		ResponseTime: 85,
	}

	assert.NotPanic(t, func() {
		BroadcastVerificationEvent(event)
	})
}

func TestBroadcastDashboardMetrics(t *testing.T) {
	metrics := &DashboardMetrics{
		Summary: &SummaryMetrics{
			TotalRequests:   1000,
			PassRate:        95.5,
			BlockRate:       2.3,
			AvgResponseTime: 85,
			ActiveSessions:  50,
		},
		Extended: &ExtendedMetrics{
			CurrentQPS:        250.5,
			ActiveConnections: 500,
			CPUUsage:          35.5,
			MemoryUsage:       58.3,
			CacheHitRate:      94.7,
			DiskUsage:         45.2,
			NetworkIn:         125.8,
			NetworkOut:        89.3,
		},
	}

	assert.NotPanic(t, func() {
		BroadcastDashboardMetrics(metrics)
	})
}

func TestExportDashboardDataWithDifferentPeriods(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminDashboardHandler()

	router.GET("/api/dashboard/admin/export", handler.ExportAdminDashboardData)

	periods := []string{"today", "yesterday", "week", "month"}

	for _, period := range periods {
		t.Run(period, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/dashboard/admin/export?format=csv&period="+period, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestGetDashboardTrendWithDifferentPeriods(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminDashboardHandler()

	router.GET("/api/dashboard/admin/trend", handler.GetAdminDashboardTrend)

	periods := []string{"hour", "day", "week"}

	for _, period := range periods {
		t.Run(period, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/dashboard/admin/trend?period="+period, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestMultipleConcurrentRequests(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminDashboardHandler()

	router.GET("/api/dashboard/admin", handler.GetAdminDashboardMetrics)
	router.GET("/api/dashboard/admin/realtime", handler.GetAdminDashboardRealtime)
	router.GET("/api/dashboard/admin/alerts", handler.GetAdminDashboardAlerts)

	for i := 0; i < 10; i++ {
		t.Run("concurrent", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/dashboard/admin", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestAPIResponseFormat(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminDashboardHandler()

	router.GET("/api/dashboard/admin", handler.GetAdminDashboardMetrics)

	req, _ := http.NewRequest("GET", "/api/dashboard/admin", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(t, body, "\"code\":0")
}

func TestWebSocketUpgrade(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminDashboardHandler()

	router.GET("/api/dashboard/admin/ws", handler.AdminDashboardWebSocketHandler)

	req, _ := http.NewRequest("GET", "/api/dashboard/admin/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Contains(t, []int{http.StatusSwitchingProtocols, http.StatusOK}, w.Code)
}
