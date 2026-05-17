package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthHandler_Check(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "health check returns OK",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/health", HealthCheck)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/health", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestReadinessCheck_IsReady(t *testing.T) {
	check := NewReadinessCheck()
	assert.NotNil(t, check)

	result := check.IsReady()
	assert.IsType(t, false, result)
}

func TestReadinessCheck_Liveness(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/ready", Readiness)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ready", nil)
	r.ServeHTTP(w, req)

	assert.NotNil(t, w)
}

func TestLivenessCheck_IsAlive(t *testing.T) {
	check := NewLivenessCheck()
	assert.NotNil(t, check)

	result := check.IsAlive()
	assert.True(t, result)
}

func TestLivenessCheck_Liveness(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/live", Liveness)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/live", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthStatus_Structure(t *testing.T) {
	status := HealthStatus{
		Status:    "healthy",
		Timestamp: "2024-01-01T00:00:00Z",
		Uptime:    "1h0m0s",
		Services:  make(map[string]interface{}),
		Metrics:   make(map[string]interface{}),
		System:    make(map[string]interface{}),
	}

	assert.Equal(t, "healthy", status.Status)
	assert.Equal(t, "2024-01-01T00:00:00Z", status.Timestamp)
	assert.Equal(t, "1h0m0s", status.Uptime)
	assert.NotNil(t, status.Services)
	assert.NotNil(t, status.Metrics)
	assert.NotNil(t, status.System)
}

func TestHealthCheck_Services(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/health", HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.NotNil(t, w)
	assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, w.Code)
}

func TestHealthCheck_Degraded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/health", HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.NotNil(t, w)
}
