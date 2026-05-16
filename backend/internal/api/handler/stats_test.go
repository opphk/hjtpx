package handler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDashboardStats_JSON(t *testing.T) {
	stats := DashboardStats{
		TotalUsers:    1000,
		TotalApps:     50,
		TotalRequests: 100000,
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
}

func TestActivityItem_JSON(t *testing.T) {
	item := ActivityItem{
		Time:   "2025-05-16 10:00:00",
		Event:  "用户登录",
		User:   "admin",
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
}

func TestVerificationStats_JSON(t *testing.T) {
	stats := VerificationStats{
		Total:        10000,
		Pending:      100,
		Success:      9500,
		Failed:       400,
		Applications: 50,
		Users:        1000,
	}

	data, err := json.Marshal(stats)
	assert.NoError(t, err)

	var unmarshaled VerificationStats
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, stats.Total, unmarshaled.Total)
	assert.Equal(t, stats.Pending, unmarshaled.Pending)
	assert.Equal(t, stats.Success, unmarshaled.Success)
	assert.Equal(t, stats.Failed, unmarshaled.Failed)
	assert.Equal(t, stats.Applications, unmarshaled.Applications)
	assert.Equal(t, stats.Users, unmarshaled.Users)
}

func TestChartDataPoint_JSON(t *testing.T) {
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
}

func TestChartData_JSON(t *testing.T) {
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

	jsonData, err := json.Marshal(data)
	assert.NoError(t, err)

	var unmarshaled ChartData
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Len(t, unmarshaled.Success, 3)
	assert.Len(t, unmarshaled.Failed, 3)
	assert.Len(t, unmarshaled.Total, 3)

	assert.Equal(t, "2025-05-16", unmarshaled.Success[2].Date)
	assert.Equal(t, int64(1500), unmarshaled.Success[2].Count)
}

func TestStatsHandler_NewStatsHandler(t *testing.T) {
	handler := NewStatsHandler()
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.statsService)
}

func TestStatsHandler_GetStatsHandler(t *testing.T) {
	handler1 := GetStatsHandler()
	handler2 := GetStatsHandler()

	assert.NotNil(t, handler1)
	assert.NotNil(t, handler2)
	assert.Equal(t, handler1, handler2)
}

func TestGenerateReportRequest_JSON(t *testing.T) {
	req := GenerateReportRequest{
		ReportType: "daily",
		StartDate:  "2025-05-01",
		EndDate:    "2025-05-15",
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)

	var unmarshaled GenerateReportRequest
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, req.ReportType, unmarshaled.ReportType)
	assert.Equal(t, req.StartDate, unmarshaled.StartDate)
	assert.Equal(t, req.EndDate, unmarshaled.EndDate)
}
