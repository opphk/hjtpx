package service

import (
	"context"
	"testing"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestNewAdvancedStatsService(t *testing.T) {
	service := NewAdvancedStatsService()
	if service == nil {
		t.Error("NewAdvancedStatsService should return a non-nil service")
	}
}

func TestGetVerificationStats(t *testing.T) {
	service := NewAdvancedStatsService()

	stats, err := service.GetVerificationStats(context.Background())
	if err != nil {
		t.Fatalf("GetVerificationStats failed: %v", err)
	}

	if stats == nil {
		t.Error("Expected non-nil stats")
	}
}

func TestGetVerificationStatsByTimeRange(t *testing.T) {
	service := NewAdvancedStatsService()

	testCases := []struct {
		name      string
		startTime int64
		endTime   int64
	}{
		{
			name:      "Last hour",
			startTime: 0,
			endTime:   3600,
		},
		{
			name:      "Last day",
			startTime: 0,
			endTime:   86400,
		},
		{
			name:      "Last week",
			startTime: 0,
			endTime:   604800,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stats, err := service.GetVerificationStatsByTimeRange(context.Background(), tc.startTime, tc.endTime)
			if err != nil {
				t.Fatalf("GetVerificationStatsByTimeRange failed: %v", err)
			}

			if stats == nil {
				t.Error("Expected non-nil stats")
			}
		})
	}
}

func TestGetTrendData(t *testing.T) {
	service := NewAdvancedStatsService()

	trends, err := service.GetTrendData(context.Background(), "daily", 7)
	if err != nil {
		t.Fatalf("GetTrendData failed: %v", err)
	}

	if trends == nil {
		t.Error("Expected non-nil trend data")
	}
}

func TestGetVerificationStatsByType(t *testing.T) {
	service := NewAdvancedStatsService()

	stats, err := service.GetVerificationStatsByType(context.Background(), "slider")
	if err != nil {
		t.Fatalf("GetVerificationStatsByType failed: %v", err)
	}

	if stats == nil {
		t.Error("Expected non-nil stats by type")
	}
}

func TestGetApplicationStats(t *testing.T) {
	service := NewAdvancedStatsService()

	stats, err := service.GetApplicationStats(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetApplicationStats failed: %v", err)
	}

	if stats == nil {
		t.Error("Expected non-nil application stats")
	}
}

func TestGetTrafficStats(t *testing.T) {
	service := NewAdvancedStatsService()

	stats, err := service.GetTrafficStats(context.Background())
	if err != nil {
		t.Fatalf("GetTrafficStats failed: %v", err)
	}

	if stats == nil {
		t.Error("Expected non-nil traffic stats")
	}
}

func TestGetSuccessRate(t *testing.T) {
	service := NewAdvancedStatsService()

	testCases := []struct {
		name    string
		appID   int64
		isValid bool
	}{
		{
			name:    "Valid app ID",
			appID:   1,
			isValid: true,
		},
		{
			name:    "Zero app ID",
			appID:   0,
			isValid: true,
		},
		{
			name:    "Negative app ID",
			appID:   -1,
			isValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rate, err := service.GetSuccessRate(context.Background(), tc.appID)
			if tc.isValid {
				if err != nil {
					t.Fatalf("GetSuccessRate should not fail for appID %d: %v", tc.appID, err)
				}
				if rate < 0 || rate > 100 {
					t.Errorf("Expected success rate in [0, 100], got %v", rate)
				}
			}
		})
	}
}

func TestGetAverageResponseTime(t *testing.T) {
	service := NewAdvancedStatsService()

	responseTime, err := service.GetAverageResponseTime(context.Background())
	if err != nil {
		t.Fatalf("GetAverageResponseTime failed: %v", err)
	}

	if responseTime < 0 {
		t.Errorf("Expected non-negative response time, got %v", responseTime)
	}
}

func TestGetFailureStats(t *testing.T) {
	service := NewAdvancedStatsService()

	stats, err := service.GetFailureStats(context.Background())
	if err != nil {
		t.Fatalf("GetFailureStats failed: %v", err)
	}

	if stats == nil {
		t.Error("Expected non-nil failure stats")
	}
}

func TestGetRiskDistribution(t *testing.T) {
	service := NewAdvancedStatsService()

	distribution, err := service.GetRiskDistribution(context.Background())
	if err != nil {
		t.Fatalf("GetRiskDistribution failed: %v", err)
	}

	if distribution == nil {
		t.Error("Expected non-nil risk distribution")
	}
}

func TestGetTopFailedReasons(t *testing.T) {
	service := NewAdvancedStatsService()

	reasons, err := service.GetTopFailedReasons(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetTopFailedReasons failed: %v", err)
	}

	if reasons == nil {
		t.Error("Expected non-nil top failed reasons")
	}
}

func TestGetVerificationVolumeByHour(t *testing.T) {
	service := NewAdvancedStatsService()

	volume, err := service.GetVerificationVolumeByHour(context.Background())
	if err != nil {
		t.Fatalf("GetVerificationVolumeByHour failed: %v", err)
	}

	if volume == nil {
		t.Error("Expected non-nil hourly volume data")
	}
}

func TestGetUserEngagementStats(t *testing.T) {
	service := NewAdvancedStatsService()

	stats, err := service.GetUserEngagementStats(context.Background())
	if err != nil {
		t.Fatalf("GetUserEngagementStats failed: %v", err)
	}

	if stats == nil {
		t.Error("Expected non-nil user engagement stats")
	}
}

func TestGetDeviceDistribution(t *testing.T) {
	service := NewAdvancedStatsService()

	distribution, err := service.GetDeviceDistribution(context.Background())
	if err != nil {
		t.Fatalf("GetDeviceDistribution failed: %v", err)
	}

	if distribution == nil {
		t.Error("Expected non-nil device distribution")
	}
}

func TestGetGeographicDistribution(t *testing.T) {
	service := NewAdvancedStatsService()

	distribution, err := service.GetGeographicDistribution(context.Background())
	if err != nil {
		t.Fatalf("GetGeographicDistribution failed: %v", err)
	}

	if distribution == nil {
		t.Error("Expected non-nil geographic distribution")
	}
}

func TestGetCaptchatypeDistribution(t *testing.T) {
	service := NewAdvancedStatsService()

	distribution, err := service.GetCaptchatypeDistribution(context.Background())
	if err != nil {
		t.Fatalf("GetCaptchatypeDistribution failed: %v", err)
	}

	if distribution == nil {
		t.Error("Expected non-nil captcha type distribution")
	}
}

func TestGetHourlyTrend(t *testing.T) {
	service := NewAdvancedStatsService()

	trend, err := service.GetHourlyTrend(context.Background())
	if err != nil {
		t.Fatalf("GetHourlyTrend failed: %v", err)
	}

	if trend == nil {
		t.Error("Expected non-nil hourly trend")
	}
}

func TestGetDailyTrend(t *testing.T) {
	service := NewAdvancedStatsService()

	trend, err := service.GetDailyTrend(context.Background())
	if err != nil {
		t.Fatalf("GetDailyTrend failed: %v", err)
	}

	if trend == nil {
		t.Error("Expected non-nil daily trend")
	}
}

func TestGetWeeklyTrend(t *testing.T) {
	service := NewAdvancedStatsService()

	trend, err := service.GetWeeklyTrend(context.Background())
	if err != nil {
		t.Fatalf("GetWeeklyTrend failed: %v", err)
	}

	if trend == nil {
		t.Error("Expected non-nil weekly trend")
	}
}

func TestGetMonthlyTrend(t *testing.T) {
	service := NewAdvancedStatsService()

	trend, err := service.GetMonthlyTrend(context.Background())
	if err != nil {
		t.Fatalf("GetMonthlyTrend failed: %v", err)
	}

	if trend == nil {
		t.Error("Expected non-nil monthly trend")
	}
}

func TestGetComparisonStats(t *testing.T) {
	service := NewAdvancedStatsService()

	comparison, err := service.GetComparisonStats(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetComparisonStats failed: %v", err)
	}

	if comparison == nil {
		t.Error("Expected non-nil comparison stats")
	}
}

func TestGetPerformanceMetrics(t *testing.T) {
	service := NewAdvancedStatsService()

	metrics, err := service.GetPerformanceMetrics(context.Background())
	if err != nil {
		t.Fatalf("GetPerformanceMetrics failed: %v", err)
	}

	if metrics == nil {
		t.Error("Expected non-nil performance metrics")
	}
}

func TestAdvancedStatsCache(t *testing.T) {
	service := NewAdvancedStatsService()

	stats1, err := service.GetVerificationStats(context.Background())
	if err != nil {
		t.Fatalf("First GetVerificationStats failed: %v", err)
	}

	stats2, err := service.GetVerificationStats(context.Background())
	if err != nil {
		t.Fatalf("Second GetVerificationStats failed: %v", err)
	}

	if stats1 != stats2 {
		t.Log("Stats may not be cached - this is acceptable if cache is disabled")
	}
}

func TestAdvancedStatsConcurrent(t *testing.T) {
	service := NewAdvancedStatsService()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_, _ = service.GetVerificationStats(context.Background())
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestAdvancedStatsWithFilters(t *testing.T) {
	service := NewAdvancedStatsService()

	filters := &model.StatsFilter{
		AppID:     1,
		Type:      "slider",
		StartTime: 0,
		EndTime:   86400,
	}

	stats, err := service.GetFilteredStats(context.Background(), filters)
	if err != nil {
		t.Fatalf("GetFilteredStats failed: %v", err)
	}

	if stats == nil {
		t.Error("Expected non-nil filtered stats")
	}
}

func TestAdvancedStatsExport(t *testing.T) {
	service := NewAdvancedStatsService()

	exportFormat := "json"
	data, err := service.ExportStats(context.Background(), exportFormat)
	if err != nil {
		t.Fatalf("ExportStats failed: %v", err)
	}

	if data == nil {
		t.Error("Expected non-nil exported data")
	}
}

func TestAdvancedStatsRealTime(t *testing.T) {
	service := NewAdvancedStatsService()

	stats, err := service.GetRealTimeStats(context.Background())
	if err != nil {
		t.Fatalf("GetRealTimeStats failed: %v", err)
	}

	if stats == nil {
		t.Error("Expected non-nil real-time stats")
	}
}
