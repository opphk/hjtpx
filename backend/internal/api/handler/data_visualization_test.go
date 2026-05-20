package handler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDataVisualizationHandler_ChartDataGeneration(t *testing.T) {
	handler := NewDataVisualizationHandler()

	t.Run("Test Generate Requests Chart Data", func(t *testing.T) {
		data := handler.generateRequestsChartData("hour")

		assert.NotEmpty(t, data.Labels, "Labels should not be empty")
		assert.NotEmpty(t, data.Datasets, "Datasets should not be empty")
		assert.Equal(t, 24, len(data.Labels), "Hour chart should have 24 labels")
		assert.Equal(t, 24, len(data.Datasets[0].Data), "Hour chart should have 24 data points")
	})

	t.Run("Test Generate Requests Chart Data for Day", func(t *testing.T) {
		data := handler.generateRequestsChartData("day")

		assert.NotEmpty(t, data.Labels, "Labels should not be empty")
		assert.Equal(t, 7, len(data.Labels), "Day chart should have 7 labels")
	})

	t.Run("Test Generate Requests Chart Data for Week", func(t *testing.T) {
		data := handler.generateRequestsChartData("week")

		assert.NotEmpty(t, data.Labels, "Labels should not be empty")
		assert.Equal(t, 7, len(data.Labels), "Week chart should have 7 labels")
	})

	t.Run("Test Generate Requests Chart Data for Month", func(t *testing.T) {
		data := handler.generateRequestsChartData("month")

		assert.NotEmpty(t, data.Labels, "Labels should not be empty")
		assert.Equal(t, 30, len(data.Labels), "Month chart should have 30 labels")
	})
}

func TestDataVisualizationHandler_VerificationChartData(t *testing.T) {
	handler := NewDataVisualizationHandler()

	t.Run("Test Generate Verification Chart Data", func(t *testing.T) {
		data := handler.generateVerificationChartData("day")

		assert.NotEmpty(t, data.Labels, "Labels should not be empty")
		assert.Equal(t, 4, len(data.Labels), "Verification chart should have 4 categories")
		assert.Contains(t, data.Labels, "成功", "Should contain 成功")
		assert.Contains(t, data.Labels, "失败", "Should contain 失败")
	})
}

func TestDataVisualizationHandler_PerformanceChartData(t *testing.T) {
	handler := NewDataVisualizationHandler()

	t.Run("Test Generate Performance Chart Data", func(t *testing.T) {
		data := handler.generatePerformanceChartData("hour")

		assert.NotEmpty(t, data.Labels, "Labels should not be empty")
		assert.NotEmpty(t, data.Datasets, "Datasets should not be empty")
		assert.Equal(t, 24, len(data.Labels), "Performance chart should have 24 labels")
	})
}

func TestDataVisualizationHandler_UsersChartData(t *testing.T) {
	handler := NewDataVisualizationHandler()

	t.Run("Test Generate Users Chart Data", func(t *testing.T) {
		data := handler.generateUsersChartData("week")

		assert.NotEmpty(t, data.Labels, "Labels should not be empty")
		assert.Equal(t, 2, len(data.Datasets), "Users chart should have 2 datasets")
		assert.Equal(t, 7, len(data.Datasets[0].Data), "Users chart should have 7 data points")
	})
}

func TestDataVisualizationHandler_RiskChartData(t *testing.T) {
	handler := NewDataVisualizationHandler()

	t.Run("Test Generate Risk Chart Data", func(t *testing.T) {
		data := handler.generateRiskChartData("day")

		assert.NotEmpty(t, data.Labels, "Labels should not be empty")
		assert.Equal(t, 4, len(data.Labels), "Risk chart should have 4 risk levels")
		assert.Contains(t, data.Labels, "低风险", "Should contain 低风险")
		assert.Contains(t, data.Labels, "高风险", "Should contain 高风险")
	})
}

func TestDataVisualizationHandler_RevenueChartData(t *testing.T) {
	handler := NewDataVisualizationHandler()

	t.Run("Test Generate Revenue Chart Data", func(t *testing.T) {
		data := handler.generateRevenueChartData("month")

		assert.NotEmpty(t, data.Labels, "Labels should not be empty")
		assert.Equal(t, 12, len(data.Labels), "Revenue chart should have 12 labels")
	})
}

func TestDataVisualizationHandler_DefaultChartData(t *testing.T) {
	handler := NewDataVisualizationHandler()

	t.Run("Test Generate Default Chart Data", func(t *testing.T) {
		data := handler.generateDefaultChartData("week")

		assert.NotEmpty(t, data.Labels, "Labels should not be empty")
		assert.NotEmpty(t, data.Datasets, "Datasets should not be empty")
	})
}

func TestDataVisualizationHandler_CalculateMetadata(t *testing.T) {
	handler := NewDataVisualizationHandler()

	t.Run("Test Calculate Metadata with Data", func(t *testing.T) {
		data := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
		metadata := handler.calculateMetadata(data, "test")

		assert.Equal(t, 150.0, metadata.Total, "Total should be 150")
		assert.Equal(t, 30.0, metadata.Average, "Average should be 30")
		assert.Equal(t, 10.0, metadata.Min, "Min should be 10")
		assert.Equal(t, 50.0, metadata.Max, "Max should be 50")
		assert.Equal(t, "test", metadata.Period, "Period should be test")
	})

	t.Run("Test Calculate Metadata with Empty Data", func(t *testing.T) {
		data := []float64{}
		metadata := handler.calculateMetadata(data, "empty")

		assert.Equal(t, 0.0, metadata.Total, "Total should be 0")
		assert.Equal(t, "empty", metadata.Period, "Period should be empty")
	})

	t.Run("Test Calculate Metadata with Single Value", func(t *testing.T) {
		data := []float64{100.0}
		metadata := handler.calculateMetadata(data, "single")

		assert.Equal(t, 100.0, metadata.Total, "Total should be 100")
		assert.Equal(t, 100.0, metadata.Average, "Average should be 100")
		assert.Equal(t, 0.0, metadata.ChangeRate, "Change rate should be 0")
	})
}

func TestDataVisualizationHandler_GetLastNDays(t *testing.T) {
	handler := NewDataVisualizationHandler()

	t.Run("Test Get Last 7 Days", func(t *testing.T) {
		days := handler.getLastNDays(7)

		assert.Equal(t, 7, len(days), "Should return 7 days")
		assert.NotEmpty(t, days[0], "First day should not be empty")
	})

	t.Run("Test Get Last 30 Days", func(t *testing.T) {
		days := handler.getLastNDays(30)

		assert.Equal(t, 30, len(days), "Should return 30 days")
	})
}

func TestDataVisualizationHandler_MultiDimensionAnalysis(t *testing.T) {
	handler := NewDataVisualizationHandler()

	t.Run("Test Get Geographic Data", func(t *testing.T) {
		data := handler.getGeographicData()

		assert.NotEmpty(t, data, "Geographic data should not be empty")
		assert.Greater(t, len(data), 0, "Should have geographic entries")
		for _, entry := range data {
			assert.NotEmpty(t, entry.Country, "Country should not be empty")
			assert.Greater(t, entry.Count, 0, "Count should be positive")
			assert.Greater(t, entry.Rate, 0.0, "Rate should be positive")
		}
	})

	t.Run("Test Get Device Type Data", func(t *testing.T) {
		data := handler.getDeviceTypeData()

		assert.NotEmpty(t, data, "Device type data should not be empty")
		assert.Greater(t, len(data), 0, "Should have device type entries")
	})

	t.Run("Test Get Browser Type Data", func(t *testing.T) {
		data := handler.getBrowserTypeData()

		assert.NotEmpty(t, data, "Browser type data should not be empty")
		assert.Greater(t, len(data), 0, "Should have browser type entries")
	})

	t.Run("Test Get Behavior Pattern Data", func(t *testing.T) {
		data := handler.getBehaviorPatternData()

		assert.NotEmpty(t, data, "Behavior pattern data should not be empty")
		assert.Greater(t, len(data), 0, "Should have behavior pattern entries")
	})
}

func TestDataVisualizationHandler_TrendsResponse(t *testing.T) {
	t.Run("Test Trends Response Structure", func(t *testing.T) {
		trends := TrendsResponse{
			OverallTrend: "增长",
			Metrics: map[string]TrendData{
				"requests": {
					Current:    15000,
					Previous:   12000,
					Change:     3000,
					ChangeRate: 25.0,
					Trend:      "up",
				},
			},
		}

		assert.Equal(t, "增长", trends.OverallTrend)
		assert.NotNil(t, trends.Metrics)
		assert.Contains(t, trends.Metrics, "requests")
	})
}

func TestDataVisualizationHandler_DistributionResponse(t *testing.T) {
	t.Run("Test Distribution Response Structure", func(t *testing.T) {
		distribution := DistributionResponse{
			Type: "verification",
			Data: []PieData{
				{Label: "成功", Value: 8500, Rate: 85.0},
				{Label: "失败", Value: 1200, Rate: 12.0},
			},
			Summary: DistributionSummary{
				Total:    10000,
				Distinct: 2,
				TopItem:  "成功",
				Entropy:  0.56,
			},
		}

		assert.Equal(t, "verification", distribution.Type)
		assert.Equal(t, 2, len(distribution.Data))
		assert.Equal(t, 10000, distribution.Summary.Total)
	})
}

func TestDataVisualizationHandler_RealTimeMetrics(t *testing.T) {
	t.Run("Test RealTime Metrics Response", func(t *testing.T) {
		metrics := RealTimeMetricsResponse{
			Current: RealTimeData{
				Timestamp: time.Now().Format("15:04:05"),
				QPS:       1250.5,
				Latency:   85.3,
				Errors:    5,
				Users:     850,
			},
			History: []RealTimeData{},
			Status: SystemStatus{
				Health:     "healthy",
				Uptime:     360000,
				CPU:        35,
				Memory:     55,
				Disk:       62,
				NetworkIn:  150000,
				NetworkOut: 250000,
			},
			Alerts: []Alert{},
		}

		assert.Greater(t, metrics.Current.QPS, 0.0, "QPS should be positive")
		assert.Greater(t, metrics.Current.Users, 0, "Users should be positive")
		assert.Equal(t, "healthy", metrics.Status.Health)
	})
}

func TestChartMetadata(t *testing.T) {
	t.Run("Test Chart Metadata Generation", func(t *testing.T) {
		metadata := ChartMetadata{
			Total:       1000,
			Average:     100,
			Min:         10,
			Max:         200,
			ChangeRate:  15.5,
			Period:      "day",
			GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		}

		assert.Equal(t, float64(1000), metadata.Total)
		assert.Equal(t, float64(100), metadata.Average)
		assert.Equal(t, "day", metadata.Period)
	})
}
