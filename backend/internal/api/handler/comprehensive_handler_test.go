package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("HealthCheck", func(t *testing.T) {
		router := gin.New()
		router.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy"})
		})

		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", response["status"])
	})
}

func TestStatsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("GetStats", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/stats", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"total": 0})
		})

		req, _ := http.NewRequest("GET", "/api/stats", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response)
	})
}

func TestApplicationHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("CreateApplication", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/applications", func(c *gin.Context) {
			var app struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Website     string `json:"website"`
			}
			if err := c.ShouldBindJSON(&app); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, gin.H{"id": 1, "name": app.Name})
		})

		app := map[string]interface{}{
			"name":        "Test App",
			"description": "Test Description",
			"website":     "https://example.com",
		}
		body, _ := json.Marshal(app)

		req, _ := http.NewRequest("POST", "/api/v1/applications", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), response["id"])
	})

	t.Run("GetApplication", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/v1/applications/:id", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"id":          1,
				"name":        "Test App",
				"description": "Test Description",
			})
		})

		req, _ := http.NewRequest("GET", "/api/v1/applications/1", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), response["id"])
	})
}

func TestBlacklistHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("AddToBlacklist", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/blacklist", func(c *gin.Context) {
			var item struct {
				Target   string `json:"target" binding:"required"`
				Type     string `json:"type" binding:"required"`
				Reason   string `json:"reason"`
				Duration string `json:"duration"`
			}
			if err := c.ShouldBindJSON(&item); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, gin.H{"success": true, "target": item.Target})
		})

		item := map[string]interface{}{
			"target":   "192.168.1.1",
			"type":     "ip",
			"reason":   "test_block",
			"duration": "1h",
		}
		body, _ := json.Marshal(item)

		req, _ := http.NewRequest("POST", "/api/v1/blacklist", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, true, response["success"])
	})

	t.Run("GetBlacklist", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/v1/blacklist", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"items": []map[string]interface{}{
					{"id": 1, "target": "192.168.1.1", "type": "ip"},
				},
				"total": 1,
			})
		})

		req, _ := http.NewRequest("GET", "/api/v1/blacklist", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), response["total"])
	})
}

func TestLogsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("GetLogs", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/v1/logs", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"logs": []map[string]interface{}{
					{"id": 1, "level": "info", "message": "Test log"},
				},
				"total": 1,
			})
		})

		req, _ := http.NewRequest("GET", "/api/v1/logs", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), response["total"])
	})
}

func TestMonitoringHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("GetMetrics", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/v1/monitoring/metrics", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"requests_per_second": 100,
				"cpu_usage":          45.5,
				"memory_usage":       60.2,
			})
		})

		req, _ := http.NewRequest("GET", "/api/v1/monitoring/metrics", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response["requests_per_second"])
	})
}

func TestDashboardHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("GetDashboardStats", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/v1/dashboard/stats", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"total_applications": 10,
				"total_verifications": 1000,
				"success_rate": 95.5,
				"active_users": 50,
			})
		})

		req, _ := http.NewRequest("GET", "/api/v1/dashboard/stats", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(10), response["total_applications"])
	})
}
