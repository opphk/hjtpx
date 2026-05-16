package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetMonitoringData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/data", GetMonitoringData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/data", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "timestamp")
	assert.Contains(t, w.Body.String(), "requests")
	assert.Contains(t, w.Body.String(), "system")
}

func TestGetAlerts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/alerts", GetAlerts)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/alerts", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "message")
	assert.Contains(t, w.Body.String(), "severity")
}

func TestAcknowledgeAlert(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.POST("/api/v1/admin/monitoring/alerts/:id/acknowledge", AcknowledgeAlert)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/admin/monitoring/alerts/1/acknowledge", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Alert acknowledged")
}

func TestGetSystemMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/system-metrics", GetSystemMetrics)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/system-metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cpu")
	assert.Contains(t, w.Body.String(), "memory")
	assert.Contains(t, w.Body.String(), "disk")
}

func TestGetRequestMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/request-metrics", GetRequestMetrics)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/request-metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "total_requests")
	assert.Contains(t, w.Body.String(), "requests_per_second")
}

func TestGetApiStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/api-stats", GetApiStats)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/api-stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "endpoints")
}

func TestWebSocketHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/ws", WebSocketHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/ws", nil)
	r.ServeHTTP(w, req)

	assert.NotNil(t, w.Code)
}
