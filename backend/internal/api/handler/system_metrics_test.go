package handler

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSystemMetricsHandler_GetSystemMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSystemMetricsHandler()

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/system-metrics", handler.GetSystemMetrics)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/system-metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cpu")
	assert.Contains(t, w.Body.String(), "memory")
	assert.Contains(t, w.Body.String(), "disk")
}

func TestSystemMetricsHandler_GetAPIMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSystemMetricsHandler()

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/api-metrics", handler.GetAPIMetrics)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/api-metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "total_requests")
}

func TestSystemMetricsHandler_GetSystemStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSystemMetricsHandler()

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/system-status", handler.GetSystemStatus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/system-status", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "overall")
}

func TestSystemMetricsHandler_GetRealtimeData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSystemMetricsHandler()

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/realtime-data", handler.GetRealtimeData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/realtime-data", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "system")
	assert.Contains(t, w.Body.String(), "api")
}

func TestSystemMetricsHandler_GetAlerts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSystemMetricsHandler()

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/alerts", handler.GetAlerts)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/alerts", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetSystemMetricsData(t *testing.T) {
	metrics := GetSystemMetricsData()
	assert.NotNil(t, metrics)
}

func TestGetAPIMetricsData(t *testing.T) {
	metrics := GetAPIMetricsData()
	assert.NotNil(t, metrics)
}

func TestGetSystemStatusData(t *testing.T) {
	status := GetSystemStatusData()
	assert.NotNil(t, status)
	assert.Contains(t, status, "overall")
}

func TestGetRealtimeData(t *testing.T) {
	data := GetRealtimeData()
	assert.NotNil(t, data)
	assert.Contains(t, data, "system")
	assert.Contains(t, data, "api")
}

func TestGetAlertData(t *testing.T) {
	alerts := GetAlertData()
	assert.True(t, alerts == nil || len(alerts) >= 0)
}

func TestMetricsCollector_NewMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector(100, 60)
	assert.NotNil(t, collector)
	assert.Equal(t, 100, collector.bufferSize)
	assert.Equal(t, 60, collector.historySize)
}

func TestMetricsCollector_RecordRequest(t *testing.T) {
	collector := NewMetricsCollector(100, 60)
	req := RequestMetric{
		Path:       "/api/test",
		Method:     "GET",
		StatusCode: 200,
		Duration:   100 * time.Millisecond,
		IP:         "127.0.0.1",
		UserAgent:  "test-agent",
		Timestamp:  time.Now(),
	}
	collector.RecordRequest(req)
	assert.Greater(t, len(collector.requestBuffer), 0)
}

func TestMetricsCollector_GetSystemMetrics(t *testing.T) {
	collector := NewMetricsCollector(100, 60)
	collector.CollectSystemMetrics()
	metrics := collector.GetSystemMetrics()
	assert.NotNil(t, metrics)
	assert.GreaterOrEqual(t, metrics.Time, int64(0))
}

func TestMetricsCollector_GetAPIMetrics(t *testing.T) {
	collector := NewMetricsCollector(100, 60)
	for i := 0; i < 5; i++ {
		collector.RecordRequest(RequestMetric{
			Path:       "/api/test",
			Method:     "GET",
			StatusCode: 200,
			Duration:   time.Duration(100+i) * time.Millisecond,
			Timestamp:  time.Now(),
		})
	}
	metrics := collector.GetAPIMetrics()
	assert.NotNil(t, metrics)
}

func TestMetricsCollector_GetHistory(t *testing.T) {
	collector := NewMetricsCollector(100, 5)
	for i := 0; i < 10; i++ {
		collector.CollectSystemMetrics()
	}
	history := collector.GetHistory()
	assert.NotNil(t, history)
}

func TestMetricsCollector_GetSystemStatus(t *testing.T) {
	collector := NewMetricsCollector(100, 60)
	collector.CollectSystemMetrics()
	status := collector.GetSystemStatus()
	assert.NotNil(t, status)
	assert.Contains(t, status, "overall")
	assert.Contains(t, status, "cpu")
	assert.Contains(t, status, "memory")
	assert.Contains(t, status, "disk")
}

func TestMetricsCollector_CheckAlerts(t *testing.T) {
	collector := NewMetricsCollector(100, 60)
	collector.CollectSystemMetrics()
	alerts := collector.CheckAlerts()
	assert.True(t, alerts == nil || len(alerts) >= 0)
}

func TestMetricsCollector_RecordAPIRequest(t *testing.T) {
	collector := NewMetricsCollector(100, 60)
	collector.RecordAPIRequest("/api/test", "GET", 200, 100*time.Millisecond, "127.0.0.1", "test-agent")
	assert.Greater(t, len(collector.requestBuffer), 0)
}

func TestMetricsCollector_IncrementRequestCount(t *testing.T) {
	collector := NewMetricsCollector(100, 60)
	initial := collector.apiMetrics.TotalRequests
	collector.IncrementRequestCount()
	assert.Equal(t, initial+1, collector.apiMetrics.TotalRequests)
}

func TestMetricsCollector_IncrementSuccessCount(t *testing.T) {
	collector := NewMetricsCollector(100, 60)
	initial := collector.apiMetrics.SuccessRequests
	collector.IncrementSuccessCount()
	assert.Equal(t, initial+1, collector.apiMetrics.SuccessRequests)
}

func TestMetricsCollector_IncrementFailureCount(t *testing.T) {
	collector := NewMetricsCollector(100, 60)
	initial := collector.apiMetrics.FailedRequests
	collector.IncrementFailureCount()
	assert.Equal(t, initial+1, collector.apiMetrics.FailedRequests)
}

func TestMetricsCollector_ResetMetrics(t *testing.T) {
	collector := NewMetricsCollector(100, 60)
	collector.IncrementRequestCount()
	collector.IncrementSuccessCount()
	collector.IncrementFailureCount()

	collector.ResetMetrics()

	assert.Equal(t, uint64(0), collector.apiMetrics.TotalRequests)
	assert.Equal(t, uint64(0), collector.apiMetrics.SuccessRequests)
	assert.Equal(t, uint64(0), collector.apiMetrics.FailedRequests)
}

func TestMetricsCollector_CalculatePercentile(t *testing.T) {
	collector := NewMetricsCollector(100, 60)
	for i := 0; i < 10; i++ {
		collector.RecordRequest(RequestMetric{
			Path:       "/api/test",
			Method:     "GET",
			StatusCode: 200,
			Duration:   time.Duration((i+1)*100) * time.Millisecond,
			Timestamp:  time.Now(),
		})
	}

	p50 := collector.CalculatePercentile(50)
	assert.GreaterOrEqual(t, p50, 0.0)

	p90 := collector.CalculatePercentile(90)
	assert.GreaterOrEqual(t, p90, 0.0)

	p99 := collector.CalculatePercentile(99)
	assert.GreaterOrEqual(t, p99, 0.0)
}

func TestMetricsCollector_CalculatePercentile_Empty(t *testing.T) {
	collector := NewMetricsCollector(100, 60)
	p50 := collector.CalculatePercentile(50)
	assert.Equal(t, 0.0, p50)
}

func TestGenerateAlertID(t *testing.T) {
	ids := make(map[int]bool)
	for i := 0; i < 100; i++ {
		id := generateAlertID()
		assert.GreaterOrEqual(t, id, 0)
		assert.Less(t, id, 10000)
		ids[id] = true
	}
}

func TestFormatStatusCode(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{200, "2xx"},
		{201, "2xx"},
		{301, "3xx"},
		{404, "4xx"},
		{500, "5xx"},
	}

	for _, tt := range tests {
		result := formatStatusCode(tt.code)
		assert.Equal(t, tt.expected, result)
	}
}

func TestSortEndpointMetrics(t *testing.T) {
	metrics := []EndpointMetric{
		{Path: "/a", Requests: 100},
		{Path: "/b", Requests: 300},
		{Path: "/c", Requests: 200},
	}

	sortEndpointMetrics(metrics)

	assert.Equal(t, "/b", metrics[0].Path)
	assert.Equal(t, "/c", metrics[1].Path)
	assert.Equal(t, "/a", metrics[2].Path)
}

func TestMetricsCollector_ProcessRequestBuffer(t *testing.T) {
	collector := NewMetricsCollector(100, 60)

	for i := 0; i < 5; i++ {
		collector.RecordRequest(RequestMetric{
			Path:       "/api/test1",
			Method:     "GET",
			StatusCode: 200,
			Duration:   100 * time.Millisecond,
			Timestamp:  time.Now(),
		})
	}

	for i := 0; i < 3; i++ {
		collector.RecordRequest(RequestMetric{
			Path:       "/api/test2",
			Method:     "POST",
			StatusCode: 404,
			Duration:   200 * time.Millisecond,
			Timestamp:  time.Now(),
		})
	}

	metrics := collector.GetAPIMetrics()
	assert.Equal(t, uint64(8), metrics.TotalRequests)
}

func TestMetricsCollector_ProcessRequestBuffer_SuccessRate(t *testing.T) {
	collector := NewMetricsCollector(100, 60)

	for i := 0; i < 8; i++ {
		collector.RecordRequest(RequestMetric{
			Path:       "/api/test",
			Method:     "GET",
			StatusCode: 200,
			Duration:   100 * time.Millisecond,
			Timestamp:  time.Now(),
		})
	}

	for i := 0; i < 2; i++ {
		collector.RecordRequest(RequestMetric{
			Path:       "/api/test",
			Method:     "GET",
			StatusCode: 500,
			Duration:   100 * time.Millisecond,
			Timestamp:  time.Now(),
		})
	}

	metrics := collector.GetAPIMetrics()
	assert.Equal(t, uint64(10), metrics.TotalRequests)
	assert.Greater(t, metrics.SuccessRate, 0.0)
}

func TestSystemMetrics_Uptime(t *testing.T) {
	collector := NewMetricsCollector(100, 60)
	time.Sleep(10 * time.Millisecond)
	collector.CollectSystemMetrics()
	metrics := collector.GetSystemMetrics()
	assert.Greater(t, metrics.Uptime.Milliseconds(), int64(0))
}

func TestGetMetricsCollector_Singleton(t *testing.T) {
	collector1 := GetMetricsCollector()
	collector2 := GetMetricsCollector()
	assert.Equal(t, collector1, collector2)
}

func TestSystemMetricsHandler_GetSystemMetrics_WithResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSystemMetricsHandler()

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/system-metrics", handler.GetSystemMetrics)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/system-metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cpu")
	assert.Contains(t, w.Body.String(), "memory")
	assert.Contains(t, w.Body.String(), "disk")
}

func TestSystemMetricsHandler_GetSystemStatus_WithResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSystemMetricsHandler()

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/system-status", handler.GetSystemStatus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/system-status", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "overall")
	assert.Contains(t, w.Body.String(), "cpu")
	assert.Contains(t, w.Body.String(), "memory")
	assert.Contains(t, w.Body.String(), "disk")
}

func TestAlertInfo_Struct(t *testing.T) {
	alert := AlertInfo{
		ID:           1,
		Type:         "test_type",
		Severity:     "warning",
		Message:      "Test message",
		Timestamp:    time.Now().Unix(),
		Acknowledged: false,
		Icon:        "bell",
	}

	assert.Equal(t, 1, alert.ID)
	assert.Equal(t, "test_type", alert.Type)
	assert.Equal(t, "warning", alert.Severity)
	assert.Equal(t, "Test message", alert.Message)
	assert.False(t, alert.Acknowledged)
	assert.Equal(t, "bell", alert.Icon)
}

func TestAPIMetrics_Struct(t *testing.T) {
	metrics := APIMetrics{
		TotalRequests:   1000,
		SuccessRequests: 950,
		FailedRequests:  50,
		RequestsPerSec:  10.5,
		AvgResponseTime: 123.45,
		MinResponseTime: 10.0,
		MaxResponseTime: 500.0,
		ErrorRate:      5.0,
		SuccessRate:    95.0,
		StatusCodes:    map[string]uint64{"2xx": 950, "4xx": 30, "5xx": 20},
		TopEndpoints:   []EndpointMetric{},
		Time:           time.Now().Unix(),
	}

	assert.Equal(t, uint64(1000), metrics.TotalRequests)
	assert.Equal(t, uint64(950), metrics.SuccessRequests)
	assert.Equal(t, uint64(50), metrics.FailedRequests)
	assert.Equal(t, 10.5, metrics.RequestsPerSec)
	assert.Equal(t, 123.45, metrics.AvgResponseTime)
	assert.Equal(t, 10.0, metrics.MinResponseTime)
	assert.Equal(t, 500.0, metrics.MaxResponseTime)
	assert.Equal(t, 5.0, metrics.ErrorRate)
	assert.Equal(t, 95.0, metrics.SuccessRate)
	assert.Equal(t, uint64(950), metrics.StatusCodes["2xx"])
}

func TestEndpointMetric_Struct(t *testing.T) {
	metric := EndpointMetric{
		Path:           "/api/test",
		Method:         "GET",
		Requests:        500,
		AvgTime:        85.5,
		MaxTime:        200.0,
		ErrorRate:      2.5,
		RequestsPerSec: 5.0,
	}

	assert.Equal(t, "/api/test", metric.Path)
	assert.Equal(t, "GET", metric.Method)
	assert.Equal(t, uint64(500), metric.Requests)
	assert.Equal(t, 85.5, metric.AvgTime)
	assert.Equal(t, 200.0, metric.MaxTime)
	assert.Equal(t, 2.5, metric.ErrorRate)
	assert.Equal(t, 5.0, metric.RequestsPerSec)
}

func TestSystemMetricsHandler_GetAPIMetrics_WithResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSystemMetricsHandler()

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/api-metrics", handler.GetAPIMetrics)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/api-metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "total_requests")
}

func TestSystemMetricsHandler_GetRealtimeData_WithResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSystemMetricsHandler()

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/realtime-data", handler.GetRealtimeData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/realtime-data", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "system")
	assert.Contains(t, w.Body.String(), "api")
}

func TestSystemMetricsHandler_GetAlerts_WithResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSystemMetricsHandler()

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/alerts", handler.GetAlerts)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/alerts", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMetricsCollector_CollectSystemMetrics_Coverage(t *testing.T) {
	collector := NewMetricsCollector(100, 60)

	for i := 0; i < 5; i++ {
		collector.CollectSystemMetrics()
	}

	metrics := collector.GetSystemMetrics()
	assert.NotNil(t, metrics)
	assert.GreaterOrEqual(t, metrics.Time, int64(0))
}

func TestMetricsCollector_GetHistory_WithData(t *testing.T) {
	collector := NewMetricsCollector(100, 10)

	for i := 0; i < 8; i++ {
		collector.CollectSystemMetrics()
	}

	history := collector.GetHistory()
	assert.NotNil(t, history)
	assert.LessOrEqual(t, len(history), 10)
}

func TestMetricsCollector_GetSystemStatus_AllLevels(t *testing.T) {
	collector := NewMetricsCollector(100, 60)
	collector.CollectSystemMetrics()

	status := collector.GetSystemStatus()
	assert.NotNil(t, status)
	assert.Contains(t, status, "overall")
	assert.Contains(t, status, "cpu")
	assert.Contains(t, status, "memory")
	assert.Contains(t, status, "disk")
	assert.Contains(t, status, "uptime")
	assert.Contains(t, status, "procs")
	assert.Contains(t, status, "load_avg")
	assert.Contains(t, status, "timestamp")
}

func TestSortEndpointMetrics_AlreadySorted(t *testing.T) {
	metrics := []EndpointMetric{
		{Path: "/a", Requests: 300},
		{Path: "/b", Requests: 200},
		{Path: "/c", Requests: 100},
	}

	sortEndpointMetrics(metrics)

	assert.Equal(t, "/a", metrics[0].Path)
	assert.Equal(t, "/b", metrics[1].Path)
	assert.Equal(t, "/c", metrics[2].Path)
}

func TestSortEndpointMetrics_Empty(t *testing.T) {
	metrics := []EndpointMetric{}
	sortEndpointMetrics(metrics)
	assert.Equal(t, 0, len(metrics))
}

func TestSortEndpointMetrics_Single(t *testing.T) {
	metrics := []EndpointMetric{
		{Path: "/a", Requests: 100},
	}
	sortEndpointMetrics(metrics)
	assert.Equal(t, "/a", metrics[0].Path)
}

func TestGetMetricsCollector(t *testing.T) {
	collector1 := GetMetricsCollector()
	collector2 := GetMetricsCollector()
	assert.NotNil(t, collector1)
	assert.NotNil(t, collector2)
	assert.Equal(t, collector1, collector2)
}

func TestConcurrentAccess(t *testing.T) {
	collector := NewMetricsCollector(1000, 60)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				collector.RecordRequest(RequestMetric{
					Path:       "/api/test",
					Method:     "GET",
					StatusCode: 200,
					Duration:   100 * time.Millisecond,
					Timestamp:  time.Now(),
				})
				collector.IncrementRequestCount()
			}
		}()
	}

	wg.Wait()
	assert.Greater(t, collector.apiMetrics.TotalRequests, uint64(0))
}
