package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetAdminDashboardService(t *testing.T) {
	service := GetAdminDashboardService()
	assert.NotNil(t, service)
}

func TestGetDashboardMetrics(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	metrics, err := service.GetDashboardMetrics(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.Summary)
	assert.NotNil(t, metrics.Extended)
	assert.NotNil(t, metrics.Trend)
	assert.NotNil(t, metrics.RiskDistribution)
	assert.NotNil(t, metrics.CaptchaType)
}

func TestGetSummaryMetrics(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	summary := service.getSummaryMetrics(ctx)

	assert.NotNil(t, summary)
	assert.GreaterOrEqual(t, summary.TotalRequests, int64(0))
	assert.GreaterOrEqual(t, summary.PassRate, float64(0))
	assert.LessOrEqual(t, summary.PassRate, float64(100))
	assert.GreaterOrEqual(t, summary.BlockRate, float64(0))
	assert.LessOrEqual(t, summary.BlockRate, float64(100))
	assert.GreaterOrEqual(t, summary.AvgResponseTime, int64(0))
}

func TestGetExtendedMetrics(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	extended := service.getExtendedMetrics(ctx)

	assert.NotNil(t, extended)
	assert.GreaterOrEqual(t, extended.CurrentQPS, float64(0))
	assert.GreaterOrEqual(t, extended.ActiveConnections, 0)
	assert.GreaterOrEqual(t, extended.CPUUsage, float64(0))
	assert.LessOrEqual(t, extended.CPUUsage, float64(100))
	assert.GreaterOrEqual(t, extended.MemoryUsage, float64(0))
	assert.LessOrEqual(t, extended.MemoryUsage, float64(100))
	assert.GreaterOrEqual(t, extended.CacheHitRate, float64(0))
	assert.LessOrEqual(t, extended.CacheHitRate, float64(100))
}

func TestGetTrendMetrics(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	testCases := []struct {
		period string
	}{
		{period: "hour"},
		{period: "day"},
		{period: "week"},
	}

	for _, tc := range testCases {
		t.Run(tc.period, func(t *testing.T) {
			trend := service.GetTrendMetrics(ctx, tc.period)
			assert.NotNil(t, trend)
			if tc.period == "hour" {
				assert.Len(t, trend, 24)
			} else if tc.period == "day" {
				assert.Len(t, trend, 7)
			} else if tc.period == "week" {
				assert.Len(t, trend, 7)
			}

			for _, m := range trend {
				assert.NotEmpty(t, m.Time)
				assert.GreaterOrEqual(t, m.Requests, int64(0))
				assert.GreaterOrEqual(t, m.Success, int64(0))
				assert.GreaterOrEqual(t, m.Failed, int64(0))
			}
		})
	}
}

func TestGetRiskDistributionMetrics(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	distribution := service.getRiskDistributionMetrics(ctx)

	assert.NotNil(t, distribution)
	assert.GreaterOrEqual(t, distribution.Low, int64(0))
	assert.GreaterOrEqual(t, distribution.Medium, int64(0))
	assert.GreaterOrEqual(t, distribution.High, int64(0))
	assert.GreaterOrEqual(t, distribution.Critical, int64(0))

	total := distribution.Low + distribution.Medium + distribution.High + distribution.Critical
	assert.Greater(t, total, int64(0))
}

func TestGetCaptchaTypeMetrics(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	captchaTypes := service.getCaptchaTypeMetrics(ctx)

	assert.NotNil(t, captchaTypes)
	assert.NotEmpty(t, captchaTypes)

	for _, ct := range captchaTypes {
		assert.NotEmpty(t, ct.Type)
		assert.GreaterOrEqual(t, ct.Count, int64(0))
	}
}

func TestGetGeoDistributionMetrics(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	geoMetrics := service.getGeoDistributionMetrics(ctx)

	assert.NotNil(t, geoMetrics)
	assert.NotEmpty(t, geoMetrics)

	for _, geo := range geoMetrics {
		assert.NotEmpty(t, geo.Region)
		assert.GreaterOrEqual(t, geo.Count, int64(0))
	}
}

func TestGenerateHeatmapData(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	heatmap := service.generateHeatmapData(ctx)

	assert.NotNil(t, heatmap)
	assert.Len(t, heatmap, 7)

	for _, row := range heatmap {
		assert.Len(t, row, 24)
		for _, value := range row {
			assert.GreaterOrEqual(t, value, 0)
		}
	}
}

func TestGetRealtimeMetrics(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	metrics, err := service.GetRealtimeMetrics(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.GreaterOrEqual(t, metrics.QPS, float64(0))
	assert.GreaterOrEqual(t, metrics.ActiveConnections, 0)
	assert.GreaterOrEqual(t, metrics.CPUUsage, float64(0))
	assert.LessOrEqual(t, metrics.CPUUsage, float64(100))
	assert.GreaterOrEqual(t, metrics.MemoryUsage, float64(0))
	assert.LessOrEqual(t, metrics.MemoryUsage, float64(100))
	assert.Greater(t, metrics.Timestamp, int64(0))
}

func TestCheckAlerts(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	alerts := service.CheckAlerts(ctx)

	assert.NotNil(t, alerts)
}

func TestExportToCSV(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	metrics, err := service.aggregateMetrics(ctx)
	assert.NoError(t, err)

	data, filename, err := service.exportToCSV(metrics, "today")

	assert.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, filename, ".csv")
	assert.Contains(t, filename, "dashboard_export")
}

func TestExportToExcel(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	metrics, err := service.aggregateMetrics(ctx)
	assert.NoError(t, err)

	data, filename, err := service.exportToExcel(metrics, "today")

	assert.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, filename, ".xlsx")
	assert.Contains(t, filename, "dashboard_export")
}

func TestExportToJSON(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	metrics, err := service.aggregateMetrics(ctx)
	assert.NoError(t, err)

	data, filename, err := service.exportToJSON(metrics)

	assert.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, filename, ".json")
	assert.Contains(t, filename, "dashboard_export")
}

func TestExportData(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	testCases := []struct {
		format string
		period string
	}{
		{format: "csv", period: "today"},
		{format: "excel", period: "today"},
		{format: "pdf", period: "today"},
		{format: "json", period: "today"},
	}

	for _, tc := range testCases {
		t.Run(tc.format, func(t *testing.T) {
			data, filename, err := service.ExportData(ctx, tc.format, tc.period)

			assert.NoError(t, err)
			assert.NotEmpty(t, data)
			assert.NotEmpty(t, filename)
			assert.Contains(t, filename, tc.format)
		})
	}
}

func TestGenerateReport(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	data, filename, err := service.GenerateReport(ctx, "测试报表", "summary", "today")

	assert.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.NotEmpty(t, filename)
	assert.Contains(t, filename, "测试报表")
	assert.Contains(t, filename, ".xlsx")
}

func TestPublishVerificationEvent(t *testing.T) {
	service := GetAdminDashboardService()

	event := VerificationEvent{
		Timestamp:   time.Now(),
		SessionID:   "test-session-123",
		CaptchaType: "slider",
		Status:      "success",
		RiskScore:   25.5,
		IPAddress:   "192.168.1.1",
		ResponseTime: 85,
	}

	service.PublishVerificationEvent(event)
}

func TestSubscribeToVerificationEvents(t *testing.T) {
	ch := SubscribeToVerificationEvents()
	assert.NotNil(t, ch)
}

func TestDashboardMetricsStructure(t *testing.T) {
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
			ActiveConnections:  500,
			CPUUsage:          35.5,
			MemoryUsage:       58.3,
			CacheHitRate:      94.7,
			DiskUsage:         45.2,
			NetworkIn:         125.8,
			NetworkOut:        89.3,
		},
		Trend: []TrendMetrics{
			{Time: "10:00", Requests: 100, Success: 95, Failed: 5},
		},
		RiskDistribution: &RiskDistribution{
			Low:      750,
			Medium:   150,
			High:     70,
			Critical: 30,
		},
		CaptchaType: []CaptchaTypeMetrics{
			{Type: "滑动验证", Count: 500},
		},
	}

	assert.NotNil(t, metrics)
	assert.Equal(t, int64(1000), metrics.Summary.TotalRequests)
	assert.Equal(t, float64(95.5), metrics.Summary.PassRate)
	assert.Equal(t, float64(250.5), metrics.Extended.CurrentQPS)
	assert.Equal(t, int64(750), metrics.RiskDistribution.Low)
}

func TestRealtimeMetricsStructure(t *testing.T) {
	metrics := &RealtimeMetrics{
		QPS:               250.5,
		ActiveConnections: 500,
		CPUUsage:          35.5,
		MemoryUsage:       58.3,
		CacheHitRate:      94.7,
		Timestamp:         time.Now().Unix(),
		RequestsPerSecond: []RequestDataPoint{
			{Time: "10:00:00", Value: 250.5},
		},
	}

	assert.NotNil(t, metrics)
	assert.Equal(t, float64(250.5), metrics.QPS)
	assert.Equal(t, 500, metrics.ActiveConnections)
	assert.Len(t, metrics.RequestsPerSecond, 1)
}

func TestDashboardAlertStructure(t *testing.T) {
	alert := DashboardAlert{
		Type:      "high_block_rate",
		Level:     "warning",
		Message:   "拦截率异常",
		Timestamp: time.Now().Unix(),
		Score:     25.5,
	}

	assert.NotNil(t, alert)
	assert.Equal(t, "high_block_rate", alert.Type)
	assert.Equal(t, "warning", alert.Level)
	assert.NotEmpty(t, alert.Message)
	assert.Greater(t, alert.Timestamp, int64(0))
}

func TestCalculateQPS(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	qps := service.calculateQPS(ctx)

	assert.GreaterOrEqual(t, qps, float64(0))
}

func TestGetActiveConnections(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	connections := service.getActiveConnections(ctx)

	assert.GreaterOrEqual(t, connections, 0)
}

func TestGetCPUUsage(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	cpuUsage := service.getCPUUsage(ctx)

	assert.GreaterOrEqual(t, cpuUsage, float64(0))
	assert.LessOrEqual(t, cpuUsage, float64(100))
}

func TestGetMemoryUsage(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	memUsage := service.getMemoryUsage(ctx)

	assert.GreaterOrEqual(t, memUsage, float64(0))
	assert.LessOrEqual(t, memUsage, float64(100))
}

func TestGetCacheHitRate(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	cacheHitRate := service.getCacheHitRate(ctx)

	assert.GreaterOrEqual(t, cacheHitRate, float64(0))
	assert.LessOrEqual(t, cacheHitRate, float64(100))
}

func TestGetDiskUsage(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	diskUsage := service.getDiskUsage(ctx)

	assert.GreaterOrEqual(t, diskUsage, float64(0))
	assert.LessOrEqual(t, diskUsage, float64(100))
}

func TestContainsFunction(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}

	assert.True(t, contains(slice, 3))
	assert.True(t, contains(slice, 1))
	assert.True(t, contains(slice, 5))
	assert.False(t, contains(slice, 6))
	assert.False(t, contains(slice, 0))
}

func TestAggregateMetrics(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	metrics, err := service.aggregateMetrics(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.Summary)
	assert.NotNil(t, metrics.Extended)
	assert.NotNil(t, metrics.Trend)
	assert.NotNil(t, metrics.RiskDistribution)
	assert.NotNil(t, metrics.CaptchaType)
	assert.NotNil(t, metrics.GeoDistribution)
	assert.NotNil(t, metrics.HeatmapData)
}

func TestAlertRules(t *testing.T) {
	assert.NotEmpty(t, alertRules)

	for _, rule := range alertRules {
		assert.NotEmpty(t, rule.Name)
		assert.NotNil(t, rule.Condition)
		assert.NotEmpty(t, rule.Level)
		assert.NotEmpty(t, rule.Message)
	}
}

func TestMultipleExportFormats(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	metrics, _ := service.aggregateMetrics(ctx)

	formats := []string{"csv", "excel", "json"}
	for _, format := range formats {
		var data []byte
		var err error

		switch format {
		case "csv":
			data, _, err = service.exportToCSV(metrics, "today")
		case "excel":
			data, _, err = service.exportToExcel(metrics, "today")
		case "json":
			data, _, err = service.exportToJSON(metrics)
		}

		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	}
}

func TestConcurrentAccess(t *testing.T) {
	service := GetAdminDashboardService()
	ctx := context.Background()

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			metrics, err := service.GetDashboardMetrics(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, metrics)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
