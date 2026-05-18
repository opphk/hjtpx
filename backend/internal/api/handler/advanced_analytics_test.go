package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetUserBehaviorAnalysis(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - returns user behavior analysis",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/analytics/user-behavior", GetUserBehaviorAnalysis)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/analytics/user-behavior", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Contains(t, resp, "totalVerifications")
			assert.Contains(t, resp, "successRate")
			assert.Contains(t, resp, "avgVerificationTime")
		})
	}
}

func TestGetAttackTrendAnalysis(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - returns attack trend analysis",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/analytics/attack-trend", GetAttackTrendAnalysis)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/analytics/attack-trend", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Contains(t, resp, "detectionRateTrend")
			assert.Contains(t, resp, "attackTypeDistribution")
		})
	}
}

func TestGenerateRiskReport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "success - daily report",
			requestBody: map[string]interface{}{
				"report_type": "daily",
				"start_date":  "2025-05-01",
				"end_date":    "2025-05-15",
				"format":      "pdf",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "success - weekly report",
			requestBody: map[string]interface{}{
				"report_type": "weekly",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "success - monthly report",
			requestBody: map[string]interface{}{
				"report_type": "monthly",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - missing report type",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/analytics/risk-report", GenerateRiskReport)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/analytics/risk-report", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestListReportConfigs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - returns report configs",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/analytics/report-configs", ListReportConfigs)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/analytics/report-configs", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Contains(t, resp, "list")
			assert.Contains(t, resp, "total")
		})
	}
}

func TestCreateReportConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "success - create report config",
			requestBody: map[string]interface{}{
				"name":          "Test Report",
				"description":   "Test description",
				"metrics":       []string{"totalRequests", "successRate"},
				"visualization": "dashboard",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - missing required fields",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/analytics/report-configs", CreateReportConfig)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/analytics/report-configs", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetReportConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		configID       string
		expectedStatus int
	}{
		{
			name:           "success - get existing config",
			configID:       "config-1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found - non-existent config",
			configID:       "non-existent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/analytics/report-configs/:id", GetReportConfig)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/analytics/report-configs/"+tt.configID, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestUpdateReportConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		configID       string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name:    "success - update existing config",
			configID: "config-1",
			requestBody: map[string]interface{}{
				"name":        "Updated Report",
				"description": "Updated description",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found - non-existent config",
			configID:       "non-existent",
			requestBody: map[string]interface{}{
				"name": "Test",
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.PUT("/api/v1/analytics/report-configs/:id", UpdateReportConfig)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/analytics/report-configs/"+tt.configID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestDeleteReportConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		configID       string
		expectedStatus int
	}{
		{
			name:           "success - delete existing config",
			configID:       "config-1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found - non-existent config",
			configID:       "non-existent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.DELETE("/api/v1/analytics/report-configs/:id", DeleteReportConfig)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/api/v1/analytics/report-configs/"+tt.configID, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetVisualizationData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - returns visualization data",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/analytics/visualization", GetVisualizationData)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/analytics/visualization", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Contains(t, resp, "pieChart")
			assert.Contains(t, resp, "barChart")
			assert.Contains(t, resp, "lineChart")
		})
	}
}

func TestAdvancedAnalyticsDataStructures(t *testing.T) {
	t.Run("UserBehaviorAnalysis marshaling", func(t *testing.T) {
		analysis := UserBehaviorAnalysis{
			TotalVerifications:  854321,
			SuccessRate:        94.7,
			AvgVerificationTime: 3.2,
			CompletionRateTrend: []TrendPoint{
				{Date: "2025-05-16", Value: 95.5},
				{Date: "2025-05-17", Value: 94.2},
			},
			VerificationTimeStats: TimeStats{
				Min:     0.8,
				Max:     15.6,
				Average: 3.2,
				Median:  2.8,
				P95:     8.5,
			},
			CaptchaTypePreference: []TypeCount{
				{Type: "滑块验证", Count: 425632},
				{Type: "点选验证", Count: 215678},
			},
		}

		data, err := json.Marshal(analysis)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("AttackTrendAnalysis marshaling", func(t *testing.T) {
		analysis := AttackTrendAnalysis{
			DetectionRateTrend: []TrendPoint{
				{Date: "2025-05-16", Value: 95.5},
			},
			AttackTypeDistribution: []TypeCount{
				{Type: "暴力破解", Count: 4523},
			},
			GeoDistribution: []GeoCount{
				{Region: "北京", Count: 2845},
			},
		}

		data, err := json.Marshal(analysis)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("ReportResponse marshaling", func(t *testing.T) {
		report := ReportResponse{
			ReportID:    "REPORT-123",
			ReportType:  "daily",
			GeneratedAt: time.Now(),
			Summary: ReportSummary{
				TotalRequests:  854321,
				SuccessRate:   94.7,
				AttackDetected: 12345,
			},
			KeyMetrics: []MetricItem{
				{Name: "验证成功率", Value: 94.7, Unit: "%"},
			},
			Recommendations: []string{"建议1", "建议2"},
		}

		data, err := json.Marshal(report)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("ReportConfig marshaling", func(t *testing.T) {
		config := ReportConfig{
			ID:          "config-1",
			Name:        "Test Config",
			Description: "Test description",
			Metrics:     []string{"totalRequests", "successRate"},
			TimeRange: TimeRangeConfig{
				Type: "daily",
			},
			Filters: map[string]interface{}{
				"severity": "high",
			},
			Visualization: "dashboard",
			Schedule: ScheduleConfig{
				Enabled:   true,
				Frequency: "daily",
				Email:     "admin@example.com",
			},
		}

		data, err := json.Marshal(config)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("VisualizationData marshaling", func(t *testing.T) {
		data := VisualizationData{
			PieChart: PieChartData{
				Labels: []string{"滑块验证", "点选验证"},
				Data:   []int64{425632, 215678},
			},
			BarChart: BarChartData{
				Labels: []string{"周一", "周二"},
				Datasets: []Dataset{
					{Label: "成功", Data: []int64{12000, 15000}},
				},
			},
		}

		jsonData, err := json.Marshal(data)
		assert.NoError(t, err)
		assert.NotEmpty(t, jsonData)
	})
}

func TestAdvancedAnalyticsHelperFunctions(t *testing.T) {
	t.Run("generateCompletionRateTrend", func(t *testing.T) {
		trend := generateCompletionRateTrend()
		assert.Len(t, trend, 30)
		for _, point := range trend {
			assert.NotEmpty(t, point.Date)
			assert.Greater(t, point.Value, float64(0))
		}
	})

	t.Run("generateHeatmap", func(t *testing.T) {
		heatmap := generateHeatmap()
		assert.Len(t, heatmap, 7)
		for _, day := range heatmap {
			assert.Len(t, day, 24)
		}
	})

	t.Run("generateHourlyDistribution", func(t *testing.T) {
		distribution := generateHourlyDistribution()
		assert.Len(t, distribution, 24)
		for i, hour := range distribution {
			assert.Equal(t, i, hour.Hour)
			assert.GreaterOrEqual(t, hour.Count, int64(0))
		}
	})

	t.Run("generateDetectionRateTrend", func(t *testing.T) {
		trend := generateDetectionRateTrend()
		assert.Len(t, trend, 30)
		for _, point := range trend {
			assert.NotEmpty(t, point.Date)
		}
	})

	t.Run("generateRecentAttacks", func(t *testing.T) {
		attacks := generateRecentAttacks()
		assert.Len(t, attacks, 10)
		for _, attack := range attacks {
			assert.NotEmpty(t, attack.ID)
			assert.NotEmpty(t, attack.IP)
			assert.NotEmpty(t, attack.Type)
		}
	})

	t.Run("generateAlerts", func(t *testing.T) {
		alerts := generateAlerts()
		assert.Len(t, alerts, 5)
		for _, alert := range alerts {
			assert.NotEmpty(t, alert.ID)
			assert.NotEmpty(t, alert.Severity)
			assert.NotEmpty(t, alert.Message)
		}
	})

	t.Run("generateLast30DaysLabels", func(t *testing.T) {
		labels := generateLast30DaysLabels()
		assert.Len(t, labels, 30)
		for _, label := range labels {
			assert.NotEmpty(t, label)
		}
	})

	t.Run("generateRandomFloats", func(t *testing.T) {
		floats := generateRandomFloats(10, 0, 100)
		assert.Len(t, floats, 10)
		for _, f := range floats {
			assert.GreaterOrEqual(t, f, float64(0))
			assert.LessOrEqual(t, f, float64(100))
		}
	})

	t.Run("generateScatterPoints", func(t *testing.T) {
		points := generateScatterPoints()
		assert.Len(t, points, 50)
		for _, point := range points {
			assert.False(t, point.X != point.X) // Check for NaN
			assert.False(t, point.Y != point.Y) // Check for NaN
		}
	})
}
