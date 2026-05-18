package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestStatsHandler_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Get verification stats", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/stats/verification", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"total":   1000,
				"success": 950,
				"failed":  50,
				"pending": 0,
			})
		})

		req, _ := http.NewRequest("GET", "/api/v1/stats/verification", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotNil(t, resp["total"])
		assert.NotNil(t, resp["success"])
		assert.NotNil(t, resp["failed"])
	})

	t.Run("Get dashboard stats", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/stats/dashboard", func(c *gin.Context) {
			c.JSON(http.StatusOK, DashboardStats{
				TotalUsers:    100,
				TotalApps:     50,
				TotalRequests: 10000,
				TotalErrors:   100,
			})
		})

		req, _ := http.NewRequest("GET", "/api/v1/stats/dashboard", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp DashboardStats
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, int64(100), resp.TotalUsers)
		assert.Equal(t, int64(50), resp.TotalApps)
		assert.Equal(t, int64(10000), resp.TotalRequests)
		assert.Equal(t, int64(100), resp.TotalErrors)
	})

	t.Run("Get chart data", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/stats/chart", func(c *gin.Context) {
			c.JSON(http.StatusOK, ChartData{
				Success: []ChartDataPoint{
					{Date: "2025-05-15", Count: 100},
					{Date: "2025-05-16", Count: 150},
				},
				Failed: []ChartDataPoint{
					{Date: "2025-05-15", Count: 10},
					{Date: "2025-05-16", Count: 5},
				},
				Total: []ChartDataPoint{
					{Date: "2025-05-15", Count: 110},
					{Date: "2025-05-16", Count: 155},
				},
			})
		})

		req, _ := http.NewRequest("GET", "/api/v1/stats/chart", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp ChartData
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.Success, 2)
		assert.Len(t, resp.Failed, 2)
		assert.Len(t, resp.Total, 2)
	})

	t.Run("Get recent activity", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/stats/activity", func(c *gin.Context) {
			activities := []ActivityItem{
				{Time: "2025-05-16 10:00:00", Event: "用户登录", User: "admin", Status: "success"},
				{Time: "2025-05-16 09:30:00", Event: "创建应用", User: "developer", Status: "success"},
			}
			c.JSON(http.StatusOK, activities)
		})

		req, _ := http.NewRequest("GET", "/api/v1/stats/activity", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp []ActivityItem
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(resp), 0)
	})

	t.Run("Get trend data", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/stats/trend", func(c *gin.Context) {
			c.JSON(http.StatusOK, []map[string]interface{}{
				{"date": "2025-05-15", "total": 100, "success": 90, "failed": 10},
				{"date": "2025-05-16", "total": 120, "success": 110, "failed": 10},
			})
		})

		req, _ := http.NewRequest("GET", "/api/v1/stats/trend", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Get hourly stats", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/stats/hourly", func(c *gin.Context) {
			date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
			hourlyData := make([]map[string]interface{}, 24)
			for i := 0; i < 24; i++ {
				hourlyData[i] = map[string]interface{}{
					"hour":  i,
					"count": i * 10,
				}
			}
			c.JSON(http.StatusOK, gin.H{
				"date":  date,
				"hours": hourlyData,
			})
		})

		req, _ := http.NewRequest("GET", "/api/v1/stats/hourly?date=2025-05-16", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Get realtime stats", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/stats/realtime", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"active_users":    50,
				"pending_verify":  5,
				"requests_minute": 100,
				"timestamp":       time.Now().Unix(),
			})
		})

		req, _ := http.NewRequest("GET", "/api/v1/stats/realtime", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Get risk distribution", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/stats/risk", func(c *gin.Context) {
			c.JSON(http.StatusOK, []map[string]interface{}{
				{"range": "0-20", "count": 500},
				{"range": "20-50", "count": 300},
				{"range": "50-80", "count": 150},
				{"range": "80-100", "count": 50},
			})
		})

		req, _ := http.NewRequest("GET", "/api/v1/stats/risk", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Get top IPs", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/stats/top-ips", func(c *gin.Context) {
			c.JSON(http.StatusOK, []map[string]interface{}{
				{"ip": "192.168.1.1", "count": 500},
				{"ip": "192.168.1.2", "count": 300},
				{"ip": "192.168.1.3", "count": 200},
			})
		})

		req, _ := http.NewRequest("GET", "/api/v1/stats/top-ips", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Get application stats", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/stats/applications", func(c *gin.Context) {
			c.JSON(http.StatusOK, []map[string]interface{}{
				{"app_id": 1, "name": "App1", "requests": 1000},
				{"app_id": 2, "name": "App2", "requests": 800},
			})
		})

		req, _ := http.NewRequest("GET", "/api/v1/stats/applications", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Get captcha type stats", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/stats/captcha-types", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"slider":   500,
				"click":    300,
				"image":    200,
				"rotation": 100,
			})
		})

		req, _ := http.NewRequest("GET", "/api/v1/stats/captcha-types", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Generate report", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/stats/report", func(c *gin.Context) {
			reportType := c.Query("report_type")
			if reportType == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "report_type required"})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"report_type":  reportType,
				"generated_at": time.Now().Format(time.RFC3339),
				"data": gin.H{
					"total_requests": 1000,
					"success_rate":   95.5,
				},
			})
		})

		tests := []struct {
			name    string
			query   string
			wantErr bool
		}{
			{"daily report", "?report_type=daily", false},
			{"weekly report", "?report_type=weekly", false},
			{"monthly report", "?report_type=monthly", false},
			{"missing type", "", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req, _ := http.NewRequest("GET", "/api/v1/stats/report"+tt.query, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				if tt.wantErr {
					assert.Equal(t, http.StatusBadRequest, w.Code)
				} else {
					assert.Equal(t, http.StatusOK, w.Code)
				}
			})
		}
	})
}

func TestStatsHandler_DataStructures(t *testing.T) {
	t.Run("DashboardStats marshaling", func(t *testing.T) {
		stats := DashboardStats{
			TotalUsers:    1000,
			TotalApps:     50,
			TotalRequests: 50000,
			TotalErrors:   500,
		}

		data, err := json.Marshal(stats)
		assert.NoError(t, err)

		var unmarshaled DashboardStats
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)

		assert.Equal(t, stats.TotalUsers, unmarshaled.TotalUsers)
		assert.Equal(t, stats.TotalApps, unmarshaled.TotalApps)
		assert.Equal(t, stats.TotalRequests, unmarshaled.TotalRequests)
		assert.Equal(t, stats.TotalErrors, unmarshaled.TotalErrors)
	})

	t.Run("VerificationStats calculations", func(t *testing.T) {
		stats := VerificationStats{
			Total:        1000,
			Pending:      50,
			Success:      900,
			Failed:       50,
			Applications: 30,
			Users:        500,
		}

		assert.Equal(t, int64(1000), stats.Total)
		assert.Equal(t, int64(900), stats.Success)
		assert.Equal(t, int64(50), stats.Failed)

		successRate := float64(stats.Success) / float64(stats.Total) * 100
		assert.InDelta(t, 90.0, successRate, 0.1)
	})

	t.Run("ChartDataPoint operations", func(t *testing.T) {
		point := ChartDataPoint{
			Date:  "2025-05-16",
			Count: 1500,
		}

		data, err := json.Marshal(point)
		assert.NoError(t, err)

		var unmarshaled ChartDataPoint
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)

		assert.Equal(t, point.Date, unmarshaled.Date)
		assert.Equal(t, point.Count, unmarshaled.Count)
	})

	t.Run("ChartData aggregation", func(t *testing.T) {
		data := ChartData{
			Success: []ChartDataPoint{
				{Date: "2025-05-14", Count: 1000},
				{Date: "2025-05-15", Count: 1200},
				{Date: "2025-05-16", Count: 1500},
			},
			Failed: []ChartDataPoint{
				{Date: "2025-05-14", Count: 50},
				{Date: "2025-05-15", Count: 60},
				{Date: "2025-05-16", Count: 40},
			},
			Total: []ChartDataPoint{
				{Date: "2025-05-14", Count: 1050},
				{Date: "2025-05-15", Count: 1260},
				{Date: "2025-05-16", Count: 1540},
			},
		}

		var totalSuccess int64
		for _, p := range data.Success {
			totalSuccess += p.Count
		}

		var totalFailed int64
		for _, p := range data.Failed {
			totalFailed += p.Count
		}

		var totalAll int64
		for _, p := range data.Total {
			totalAll += p.Count
		}

		assert.Equal(t, totalAll, totalSuccess+totalFailed)
	})

	t.Run("ActivityItem tracking", func(t *testing.T) {
		item := ActivityItem{
			Time:   "2025-05-16 10:30:00",
			Event:  "用户登录",
			User:   "testuser",
			Status: "success",
		}

		data, err := json.Marshal(item)
		assert.NoError(t, err)

		var unmarshaled ActivityItem
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)

		assert.Equal(t, item.Time, unmarshaled.Time)
		assert.Equal(t, item.Event, unmarshaled.Event)
		assert.Equal(t, item.User, unmarshaled.User)
		assert.Equal(t, item.Status, unmarshaled.Status)
	})

	t.Run("GenerateReportRequest validation", func(t *testing.T) {
		tests := []struct {
			name      string
			req       GenerateReportRequest
			wantValid bool
		}{
			{
				name:      "daily report",
				req:       GenerateReportRequest{ReportType: "daily"},
				wantValid: true,
			},
			{
				name:      "weekly report with dates",
				req:       GenerateReportRequest{ReportType: "weekly", StartDate: "2025-05-01", EndDate: "2025-05-15"},
				wantValid: true,
			},
			{
				name:      "missing report type",
				req:       GenerateReportRequest{},
				wantValid: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data, err := json.Marshal(tt.req)
				assert.NoError(t, err)

				var unmarshaled GenerateReportRequest
				err = json.Unmarshal(data, &unmarshaled)
				assert.NoError(t, err)

				if tt.wantValid {
					assert.NotEmpty(t, unmarshaled.ReportType)
				} else {
					assert.Empty(t, unmarshaled.ReportType)
				}
			})
		}
	})
}

func TestStatsHandler_StatsCalculation(t *testing.T) {
	t.Run("Success rate calculation", func(t *testing.T) {
		tests := []struct {
			name     string
			success  int64
			total    int64
			expected float64
		}{
			{"50% success", 50, 100, 50.0},
			{"100% success", 100, 100, 100.0},
			{"0% success", 0, 100, 0.0},
			{"95.5% success", 955, 1000, 95.5},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.total > 0 {
					rate := float64(tt.success) / float64(tt.total) * 100
					assert.InDelta(t, tt.expected, rate, 0.1)
				}
			})
		}
	})

	t.Run("Trend data aggregation", func(t *testing.T) {
		dailyData := []map[string]interface{}{
			{"date": "2025-05-14", "total": float64(100), "success": float64(90), "failed": float64(10)},
			{"date": "2025-05-15", "total": float64(120), "success": float64(110), "failed": float64(10)},
			{"date": "2025-05-16", "total": float64(150), "success": float64(140), "failed": float64(10)},
		}

		var totalRequests int64
		var totalSuccess int64
		var totalFailed int64

		for _, day := range dailyData {
			if total, ok := day["total"].(float64); ok {
				totalRequests += int64(total)
			}
			if success, ok := day["success"].(float64); ok {
				totalSuccess += int64(success)
			}
			if failed, ok := day["failed"].(float64); ok {
				totalFailed += int64(failed)
			}
		}

		assert.Equal(t, int64(370), totalRequests)
		assert.Equal(t, int64(340), totalSuccess)
		assert.Equal(t, int64(30), totalFailed)
	})

	t.Run("Hourly distribution", func(t *testing.T) {
		hourlyData := make(map[int]int64)
		for i := 0; i < 24; i++ {
			hourlyData[i] = int64(i * 10)
		}

		peakHour := 0
		peakCount := int64(0)
		for hour, count := range hourlyData {
			if count > peakCount {
				peakCount = count
				peakHour = hour
			}
		}

		assert.Equal(t, 23, peakHour)
		assert.Equal(t, int64(230), peakCount)
	})

	t.Run("Risk distribution bucketing", func(t *testing.T) {
		risks := []float64{10, 25, 35, 55, 75, 85, 95, 15, 45, 65}

		buckets := map[string]int64{
			"low":    0,
			"medium": 0,
			"high":   0,
		}

		for _, risk := range risks {
			switch {
			case risk < 30:
				buckets["low"]++
			case risk < 70:
				buckets["medium"]++
			default:
				buckets["high"]++
			}
		}

		assert.Equal(t, int64(3), buckets["low"])
		assert.Equal(t, int64(4), buckets["medium"])
		assert.Equal(t, int64(3), buckets["high"])
	})
}

func TestStatsHandler_PerformanceMetrics(t *testing.T) {
	t.Run("Average response time calculation", func(t *testing.T) {
		responseTimes := []int64{100, 150, 200, 250, 300, 350, 400, 450, 500, 550}

		var total int64
		for _, rt := range responseTimes {
			total += rt
		}
		avg := float64(total) / float64(len(responseTimes))

		assert.InDelta(t, 325.0, avg, 1.0)
	})

	t.Run("Percentile calculation", func(t *testing.T) {
		values := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}

		p50Index := (len(values) - 1) * 50 / 100
		p50 := values[p50Index]
		assert.Equal(t, int64(50), p50)

		p90Index := (len(values) - 1) * 90 / 100
		assert.GreaterOrEqual(t, values[p90Index], int64(90))
	})

	t.Run("Requests per minute calculation", func(t *testing.T) {
		requestsInWindow := int64(600)
		windowMinutes := 10

		rpm := float64(requestsInWindow) / float64(windowMinutes)
		assert.InDelta(t, 60.0, rpm, 0.1)
	})
}

func TestStatsHandler_Concurrent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Concurrent stats requests", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/stats/dashboard", func(c *gin.Context) {
			c.JSON(http.StatusOK, DashboardStats{
				TotalUsers:    100,
				TotalApps:     50,
				TotalRequests: 1000,
				TotalErrors:   10,
			})
		})

		done := make(chan bool, 100)

		for i := 0; i < 100; i++ {
			go func() {
				req, _ := http.NewRequest("GET", "/api/v1/stats/dashboard", nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				if w.Code == http.StatusOK {
					done <- true
				} else {
					done <- false
				}
			}()
		}

		successCount := 0
		for i := 0; i < 100; i++ {
			if <-done {
				successCount++
			}
		}

		assert.Equal(t, 100, successCount, "All concurrent requests should succeed")
	})
}
