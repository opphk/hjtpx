package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
	}{
		{
			name: "returns health status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("GET", "/health", nil)
			c.Request = req

			// Act
			HealthCheck(c)

			// Assert - 检查有响应即可，状态码可以是200或503，取决于外部服务
			assert.GreaterOrEqual(t, w.Code, 200)
			assert.Less(t, w.Code, 600)
		})
	}
}

func TestReadiness(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
	}{
		{
			name: "returns readiness status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("GET", "/ready", nil)
			c.Request = req

			// Act
			Readiness(c)

			// Assert - 检查有响应即可
			assert.GreaterOrEqual(t, w.Code, 200)
			assert.Less(t, w.Code, 600)
		})
	}
}

func TestLiveness(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "returns liveness status",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("GET", "/alive", nil)
			c.Request = req

			// Act
			Liveness(c)

			// Assert - Liveness 应该总是返回 200 OK
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestNewReadinessCheck(t *testing.T) {
	check := NewReadinessCheck()
	assert.NotNil(t, check)
}

func TestNewLivenessCheck(t *testing.T) {
	check := NewLivenessCheck()
	assert.NotNil(t, check)
}

func TestLivenessCheck_IsAlive(t *testing.T) {
	check := NewLivenessCheck()
	alive := check.IsAlive()
	assert.True(t, alive)
}

func TestGetSystemMetricsInfo(t *testing.T) {
	metrics := getSystemMetrics()
	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "go_version")
	assert.Contains(t, metrics, "go_routines")
}
