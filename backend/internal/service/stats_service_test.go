package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewStatsService(t *testing.T) {
	service := NewStatsService()
	assert.NotNil(t, service)
}

func TestOverviewStats_Structure(t *testing.T) {
	stats := OverviewStats{
		TotalVerifications: 1000,
		SuccessCount:       950,
		FailedCount:        50,
		PendingCount:       0,
		SuccessRate:        95.0,
		AvgRiskScore:       25.5,
		TotalApplications:  10,
		TotalUsers:         100,
	}

	assert.Equal(t, int64(1000), stats.TotalVerifications)
	assert.Equal(t, int64(950), stats.SuccessCount)
	assert.Equal(t, int64(50), stats.FailedCount)
	assert.Equal(t, float64(95.0), stats.SuccessRate)
	assert.Equal(t, float64(25.5), stats.AvgRiskScore)
}

func TestCaptchaTypeStats_Structure(t *testing.T) {
	stats := CaptchaTypeStats{
		CaptchaType:  "slider",
		TotalCount:  500,
		SuccessCount: 475,
		FailedCount: 25,
		SuccessRate: 95.0,
		AvgRiskScore: 20.0,
		AvgDuration: 150,
	}

	assert.Equal(t, "slider", stats.CaptchaType)
	assert.Equal(t, int64(500), stats.TotalCount)
	assert.Equal(t, int64(475), stats.SuccessCount)
	assert.Equal(t, int64(25), stats.FailedCount)
	assert.Equal(t, float64(95.0), stats.SuccessRate)
}

func TestApplicationStats_Structure(t *testing.T) {
	stats := ApplicationStats{
		ApplicationID:   1,
		ApplicationName: "Test App",
		TotalCount:      300,
		SuccessCount:    285,
		FailedCount:     15,
		SuccessRate:     95.0,
		AvgRiskScore:    22.5,
	}

	assert.Equal(t, uint(1), stats.ApplicationID)
	assert.Equal(t, "Test App", stats.ApplicationName)
	assert.Equal(t, int64(300), stats.TotalCount)
}

func TestTrendDataPoint_Structure(t *testing.T) {
	data := TrendDataPoint{
		Date:         "2026-01-01",
		TotalCount:   200,
		SuccessCount: 190,
		FailedCount:  10,
		SuccessRate:  95.0,
		AvgRiskScore: 20.0,
	}

	assert.Equal(t, "2026-01-01", data.Date)
	assert.Equal(t, int64(200), data.TotalCount)
	assert.Equal(t, float64(95.0), data.SuccessRate)
}

func TestHourlyStats_Structure(t *testing.T) {
	stats := HourlyStats{
		Hour:         12,
		TotalCount:   100,
		SuccessCount: 95,
		FailedCount: 5,
	}

	assert.Equal(t, 12, stats.Hour)
	assert.Equal(t, int64(100), stats.TotalCount)
	assert.Equal(t, int64(95), stats.SuccessCount)
	assert.Equal(t, int64(5), stats.FailedCount)
}

func TestRealtimeStats_Structure(t *testing.T) {
	stats := RealtimeStats{
		CurrentMinute:   50,
		LastMinute:     45,
		CurrentHour:    3000,
		SuccessRate:    95.5,
		AvgResponseTime: 125.5,
		ActiveSessions: 25,
	}

	assert.Equal(t, int64(50), stats.CurrentMinute)
	assert.Equal(t, int64(45), stats.LastMinute)
	assert.Equal(t, int64(3000), stats.CurrentHour)
	assert.Equal(t, float64(95.5), stats.SuccessRate)
}

func TestRiskDistribution_Structure(t *testing.T) {
	distribution := RiskDistribution{
		RiskLevel:  "Low (0-30)",
		MinScore:   0,
		MaxScore:   30,
		Count:      500,
		Percentage: 50.0,
	}

	assert.Equal(t, "Low (0-30)", distribution.RiskLevel)
	assert.Equal(t, float64(0), distribution.MinScore)
	assert.Equal(t, float64(30), distribution.MaxScore)
	assert.Equal(t, int64(500), distribution.Count)
	assert.Equal(t, float64(50.0), distribution.Percentage)
}

func TestTopIPs_Structure(t *testing.T) {
	topIP := TopIPs{
		IPAddress:    "192.168.1.100",
		RequestCount: 1000,
		SuccessRate:  98.5,
	}

	assert.Equal(t, "192.168.1.100", topIP.IPAddress)
	assert.Equal(t, int64(1000), topIP.RequestCount)
	assert.Equal(t, float64(98.5), topIP.SuccessRate)
}

func TestStatsService_GetOverviewStats(t *testing.T) {
	service := NewStatsService()
	stats, err := service.GetOverviewStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
}

func TestStatsService_GetCaptchaTypeStats(t *testing.T) {
	service := NewStatsService()
	stats, err := service.GetCaptchaTypeStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
}

func TestStatsService_GetApplicationStats(t *testing.T) {
	service := NewStatsService()
	stats, err := service.GetApplicationStats(5)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
}

func TestStatsService_GetApplicationStats_DefaultLimit(t *testing.T) {
	service := NewStatsService()
	stats, err := service.GetApplicationStats(0)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
}

func TestStatsService_GetTrendData(t *testing.T) {
	service := NewStatsService()
	data, err := service.GetTrendData(7)
	assert.NoError(t, err)
	assert.NotNil(t, data)
}

func TestStatsService_GetTrendData_DefaultDays(t *testing.T) {
	service := NewStatsService()
	data, err := service.GetTrendData(0)
	assert.NoError(t, err)
	assert.NotNil(t, data)
}

func TestStatsService_GetHourlyStats(t *testing.T) {
	service := NewStatsService()
	date := time.Now().Format("2006-01-02")
	stats, err := service.GetHourlyStats(date)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Len(t, stats, 24)
}

func TestStatsService_GetHourlyStats_InvalidDate(t *testing.T) {
	service := NewStatsService()
	stats, err := service.GetHourlyStats("invalid-date")
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Len(t, stats, 24)
}

func TestStatsService_GetRealtimeStats(t *testing.T) {
	service := NewStatsService()
	stats, err := service.GetRealtimeStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
}

func TestStatsService_GetRiskDistribution(t *testing.T) {
	service := NewStatsService()
	distribution, err := service.GetRiskDistribution()
	assert.NoError(t, err)
	assert.NotNil(t, distribution)
	assert.Len(t, distribution, 4)
}

func TestStatsService_GetTopIPs(t *testing.T) {
	service := NewStatsService()
	ips, err := service.GetTopIPs(5)
	assert.NoError(t, err)
	assert.NotNil(t, ips)
}

func TestStatsService_GetTopIPs_DefaultLimit(t *testing.T) {
	service := NewStatsService()
	ips, err := service.GetTopIPs(0)
	assert.NoError(t, err)
	assert.NotNil(t, ips)
}

func TestStatsService_GetLogStatistics(t *testing.T) {
	service := NewStatsService()
	stats, err := service.GetLogStatistics()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "total_count")
	assert.Contains(t, stats, "success_count")
	assert.Contains(t, stats, "failed_count")
}

func TestStatsService_GenerateReport(t *testing.T) {
	service := NewStatsService()
	
	report, err := service.GenerateReport("daily", time.Now(), time.Now())
	assert.NoError(t, err)
	assert.Contains(t, report, "Daily Report")
	
	report, err = service.GenerateReport("weekly", time.Now(), time.Now())
	assert.NoError(t, err)
	assert.Contains(t, report, "Weekly Report")
	
	report, err = service.GenerateReport("monthly", time.Now(), time.Now())
	assert.NoError(t, err)
	assert.Contains(t, report, "Monthly Report")
}

func TestStatsService_GenerateReport_UnsupportedType(t *testing.T) {
	service := NewStatsService()
	report, err := service.GenerateReport("yearly", time.Now(), time.Now())
	assert.Error(t, err)
	assert.Empty(t, report)
}
