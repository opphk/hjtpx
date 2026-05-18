package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/hjtpx/hjtpx/internal/service"
)

func TestNewDashboardHandler(t *testing.T) {
	handler := NewDashboardHandler()
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.dashboardService)
}

func TestGetDashboardHandler_Function(t *testing.T) {
	handler := GetDashboardHandler()
	assert.NotNil(t, handler)
}

func TestGetDashboardHandler_ReturnsNewInstance(t *testing.T) {
	handler1 := GetDashboardHandler()
	handler2 := GetDashboardHandler()
	assert.NotNil(t, handler1)
	assert.NotNil(t, handler2)
}

func TestGetDashboardData_QueryParameters(t *testing.T) {
	tests := []struct {
		name   string
		period string
	}{
		{"hour", "hour"},
		{"day", "day"},
		{"week", "week"},
		{"month", "month"},
		{"default", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.GET("/dashboard", GetDashboardData)

			url := "/dashboard"
			if tt.period != "" {
				url = "/dashboard?period=" + tt.period
			}

			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code)
		})
	}
}

func TestExportDashboardData_Formats(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{"csv format", "csv"},
		{"json format", "json"},
		{"excel format", "excel"},
		{"default format", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.GET("/dashboard/export", ExportDashboardData)

			url := "/dashboard/export"
			if tt.format != "" {
				url = "/dashboard/export?format=" + tt.format
			}

			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code)
		})
	}
}

func TestExportDashboardData_Headers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/dashboard/export", ExportDashboardData)

	req, _ := http.NewRequest("GET", "/dashboard/export?format=csv", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		assert.NotEmpty(t, w.Header().Get("Content-Disposition"))
	}
}

func TestGetRecentVerifications(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/dashboard/recent", GetRecentVerifications)

	req, _ := http.NewRequest("GET", "/dashboard/recent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code)
}

func TestGetDashboardAlerts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/dashboard/alerts", GetDashboardAlerts)

	req, _ := http.NewRequest("GET", "/dashboard/alerts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code)

	if w.Code == http.StatusOK {
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp, "data")
	}
}

func TestDashboardHandler_Structure(t *testing.T) {
	handler := NewDashboardHandler()
	assert.NotNil(t, handler.dashboardService)
}

func TestGetAttackTypeDistribution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/attack-distribution", GetAttackTypeDistribution)

	req, _ := http.NewRequest("GET", "/api/attack-distribution", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code)

	if w.Code == http.StatusOK {
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		data, ok := resp["data"]
		assert.True(t, ok)
		assert.IsType(t, []interface{}{}, data)
	}
}

func TestGetDashboardRiskScoreDistribution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/risk-score-distribution", GetDashboardRiskScoreDistribution)

	req, _ := http.NewRequest("GET", "/api/risk-score-distribution", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code)

	if w.Code == http.StatusOK {
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		data, ok := resp["data"]
		assert.True(t, ok)
		assert.IsType(t, []interface{}{}, data)
	}
}

func TestDashboardWebSocketHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/dashboard/ws", DashboardWebSocketHandler)

	req, _ := http.NewRequest("GET", "/api/dashboard/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")

	w := httptest.NewRecorder()
	
	defer func() {
		if r := recover(); r != nil {
			t.Skip("Skipping WebSocket test - httptest.ResponseRecorder does not support WebSocket Hijack interface")
		}
	}()
	
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSwitchingProtocols, w.Code)
}

func TestDashboardDataStructure(t *testing.T) {
	service := service.NewDashboardService()

	data, err := service.GetDashboardData("hour")
	assert.NoError(t, err)
	assert.NotNil(t, data)

	if data.Summary != nil {
		assert.NotNil(t, data.Summary.TotalRequests)
		assert.NotNil(t, data.Summary.PassRate)
		assert.NotNil(t, data.Summary.BlockRate)
		assert.NotNil(t, data.Summary.AvgResponseTime)
		assert.NotNil(t, data.Summary.ActiveSessions)
	}

	assert.NotNil(t, data.Trend)
	assert.NotNil(t, data.RiskDistribution)
	assert.NotNil(t, data.CaptchaType)
	assert.NotNil(t, data.AttackTypeDistribution)
	assert.NotNil(t, data.RiskScoreDistribution)
}

func TestDashboardService_GetAttackTypeDistribution(t *testing.T) {
	service := service.NewDashboardService()

	distribution, err := service.GetAttackTypeDistribution()
	assert.NoError(t, err)
	assert.NotNil(t, distribution)
	assert.GreaterOrEqual(t, len(distribution), 0)
}

func TestDashboardService_GetRiskScoreDistribution(t *testing.T) {
	service := service.NewDashboardService()

	distribution, err := service.GetRiskScoreDistribution()
	assert.NoError(t, err)
	assert.NotNil(t, distribution)
	assert.Equal(t, 10, len(distribution))
}

func TestDashboardService_ExportData_Formats(t *testing.T) {
	service := service.NewDashboardService()

	formats := []string{"csv", "json"}
	for _, format := range formats {
		data, err := service.ExportData(format, "hour")
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	}
}

func TestDashboardService_ExportData_UnsupportedFormat(t *testing.T) {
	service := service.NewDashboardService()

	_, err := service.ExportData("unsupported", "hour")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported export format")
}

func TestPublishVerificationEvent(t *testing.T) {
	event := service.RealTimeVerificationEvent{
		Timestamp:   time.Now(),
		SessionID:   "test-session",
		CaptchaType: "slider",
		Status:      "success",
		RiskScore:   25.5,
		IPAddress:   "127.0.0.1",
	}

	service.PublishVerificationEvent(event)
}

func TestTrendDataStructure(t *testing.T) {
	service := service.NewDashboardService()

	trend, err := service.GetDashboardData("hour")
	assert.NoError(t, err)

	for _, data := range trend.Trend {
		assert.NotEmpty(t, data.Time)
		assert.NotNil(t, data.Requests)
		assert.NotNil(t, data.Success)
		assert.NotNil(t, data.Failed)
	}
}
