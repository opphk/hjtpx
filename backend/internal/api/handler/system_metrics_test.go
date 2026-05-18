package handler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector(1000, 60)
	assert.NotNil(t, collector)
	assert.Equal(t, 1000, collector.bufferSize)
	assert.Equal(t, 60, collector.historySize)
	assert.NotNil(t, collector.apiMetrics.StatusCodes)
	assert.NotNil(t, collector.apiMetrics.TopEndpoints)
}

func TestNewSystemMetricsHandler(t *testing.T) {
	handler := NewSystemMetricsHandler()
	assert.NotNil(t, handler)
}

func TestMetricsCollector_GetSystemMetrics(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	metrics := collector.GetSystemMetrics()
	assert.NotNil(t, metrics)
}

func TestMetricsCollector_GetAPIMetrics(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	metrics := collector.GetAPIMetrics()
	assert.NotNil(t, metrics)
}

func TestMetricsCollector_GetHistory(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	history := collector.GetHistory()
	assert.NotNil(t, history)
	assert.Empty(t, history)
}

func TestMetricsCollector_GetRealtimeData(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	data := collector.GetRealtimeData()
	assert.NotNil(t, data)
	assert.Contains(t, data, "system")
	assert.Contains(t, data, "api")
	assert.Contains(t, data, "history")
}

func TestMetricsCollector_GetSystemStatus(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	status := collector.GetSystemStatus()
	assert.NotNil(t, status)
	assert.Contains(t, status, "overall")
	assert.Contains(t, status, "cpu")
	assert.Contains(t, status, "memory")
	assert.Contains(t, status, "disk")
}

func TestMetricsCollector_RecordRequest(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
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
	assert.Len(t, collector.requestBuffer, 1)
}

func TestMetricsCollector_RecordAPIRequest(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	collector.RecordAPIRequest("/api/test", "GET", 200, 100*time.Millisecond, "127.0.0.1", "test-agent")
	assert.Greater(t, collector.apiMetrics.TotalRequests, uint64(0))
}

func TestMetricsCollector_IncrementRequestCount(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	collector.IncrementRequestCount()
	assert.Equal(t, uint64(1), collector.apiMetrics.TotalRequests)
}

func TestMetricsCollector_IncrementSuccessCount(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	collector.IncrementSuccessCount()
	assert.Equal(t, uint64(1), collector.apiMetrics.SuccessRequests)
}

func TestMetricsCollector_IncrementFailureCount(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	collector.IncrementFailureCount()
	assert.Equal(t, uint64(1), collector.apiMetrics.FailedRequests)
}

func TestMetricsCollector_ResetMetrics(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	collector.IncrementRequestCount()
	collector.IncrementSuccessCount()
	collector.IncrementFailureCount()
	collector.ResetMetrics()
	assert.Equal(t, uint64(0), collector.apiMetrics.TotalRequests)
	assert.Equal(t, uint64(0), collector.apiMetrics.SuccessRequests)
	assert.Equal(t, uint64(0), collector.apiMetrics.FailedRequests)
}

func TestMetricsCollector_CalculatePercentile(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	for i := 0; i < 5; i++ {
		collector.RecordRequest(RequestMetric{
			Path:       "/api/test",
			Method:     "GET",
			StatusCode: 200,
			Duration:   time.Duration(i+1) * 100 * time.Millisecond,
		})
	}

	percentile := collector.CalculatePercentile(50)
	assert.Greater(t, percentile, 0.0)
}

func TestMetricsCollector_CalculatePercentile_Empty(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	percentile := collector.CalculatePercentile(50)
	assert.Equal(t, 0.0, percentile)
}

func TestFormatStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected string
	}{
		{"2xx success", 200, "2xx"},
		{"2xx created", 201, "2xx"},
		{"3xx redirect", 301, "3xx"},
		{"4xx bad request", 400, "4xx"},
		{"5xx server error", 500, "5xx"},
		{"5xx internal error", 503, "5xx"},
		{"unknown", 100, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatusCode(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortEndpointMetrics(t *testing.T) {
	metrics := []EndpointMetric{
		{Path: "/api/a", Requests: 100},
		{Path: "/api/b", Requests: 300},
		{Path: "/api/c", Requests: 200},
	}
	sortEndpointMetrics(metrics)
	assert.Equal(t, "/api/b", metrics[0].Path)
	assert.Equal(t, "/api/c", metrics[1].Path)
	assert.Equal(t, "/api/a", metrics[2].Path)
}

func TestGenerateAlertID(t *testing.T) {
	id1 := generateAlertID()
	id2 := generateAlertID()
	assert.GreaterOrEqual(t, id1, 0)
	assert.GreaterOrEqual(t, id2, 0)
}

func TestSystemMetrics_Structure(t *testing.T) {
	metrics := SystemMetrics{}
	metrics.CPU.Usage = 50.0
	metrics.CPU.Count = 4
	metrics.Memory.Total = 8589934592
	metrics.Memory.Used = 4294967296
	metrics.Memory.UsagePercent = 50.0
	metrics.Disk.Total = 500000000000
	metrics.Disk.Used = 250000000000
	metrics.Disk.UsagePercent = 50.0
	metrics.Uptime = 3600 * time.Second
	metrics.Time = time.Now().Unix()

	assert.Equal(t, 50.0, metrics.CPU.Usage)
	assert.Equal(t, 4, metrics.CPU.Count)
	assert.Equal(t, uint64(8589934592), metrics.Memory.Total)
	assert.Equal(t, uint64(500000000000), metrics.Disk.Total)
	assert.Equal(t, time.Duration(3600*time.Second), metrics.Uptime)
}

func TestAPIMetrics_Structure(t *testing.T) {
	metrics := APIMetrics{
		TotalRequests:   1000,
		SuccessRequests: 950,
		FailedRequests: 50,
		RequestsPerSec: 100.5,
		AvgResponseTime: 125.5,
		MinResponseTime: 10.0,
		MaxResponseTime: 500.0,
		ErrorRate:       5.0,
		SuccessRate:     95.0,
		StatusCodes:     map[string]uint64{"2xx": 950, "4xx": 30, "5xx": 20},
	}

	assert.Equal(t, uint64(1000), metrics.TotalRequests)
	assert.Equal(t, uint64(950), metrics.SuccessRequests)
	assert.Equal(t, uint64(50), metrics.FailedRequests)
	assert.Equal(t, 100.5, metrics.RequestsPerSec)
	assert.Equal(t, 5.0, metrics.ErrorRate)
	assert.Equal(t, 95.0, metrics.SuccessRate)
}

func TestEndpointMetric_Structure(t *testing.T) {
	metric := EndpointMetric{
		Path:           "/api/v1/captcha",
		Method:         "POST",
		Requests:       500,
		AvgTime:        150.5,
		MaxTime:        800.0,
		ErrorRate:      2.0,
		RequestsPerSec: 50.0,
	}

	assert.Equal(t, "/api/v1/captcha", metric.Path)
	assert.Equal(t, "POST", metric.Method)
	assert.Equal(t, uint64(500), metric.Requests)
	assert.Equal(t, 150.5, metric.AvgTime)
}

func TestRequestMetric_Structure(t *testing.T) {
	now := time.Now()
	metric := RequestMetric{
		Path:       "/api/v1/captcha",
		Method:     "POST",
		StatusCode: 200,
		Duration:   150 * time.Millisecond,
		IP:         "192.168.1.100",
		UserAgent:  "Mozilla/5.0",
		Timestamp:  now,
	}

	assert.Equal(t, "/api/v1/captcha", metric.Path)
	assert.Equal(t, "POST", metric.Method)
	assert.Equal(t, 200, metric.StatusCode)
	assert.Equal(t, 150*time.Millisecond, metric.Duration)
	assert.Equal(t, "192.168.1.100", metric.IP)
	assert.Equal(t, "Mozilla/5.0", metric.UserAgent)
	assert.Equal(t, now, metric.Timestamp)
}

func TestAlertInfo_Structure(t *testing.T) {
	alert := AlertInfo{
		ID:           1,
		Type:         "high_cpu",
		Severity:     "warning",
		Message:      "CPU使用率超过75%",
		Timestamp:    1234567890,
		Acknowledged: false,
		Icon:         "cpu",
	}

	assert.Equal(t, 1, alert.ID)
	assert.Equal(t, "high_cpu", alert.Type)
	assert.Equal(t, "warning", alert.Severity)
	assert.Equal(t, "CPU使用率超过75%", alert.Message)
	assert.False(t, alert.Acknowledged)
	assert.Equal(t, "cpu", alert.Icon)
}

func TestSystemMetricsHandler_GetSystemMetrics(t *testing.T) {
	handler := NewSystemMetricsHandler()
	assert.NotNil(t, handler)
}
