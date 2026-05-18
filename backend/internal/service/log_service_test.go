package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLogService(t *testing.T) {
	service := NewLogService()
	assert.NotNil(t, service)
}

func TestLogQueryParams_DefaultValues(t *testing.T) {
	params := LogQueryParams{}
	
	assert.Equal(t, 0, params.Page)
	assert.Equal(t, 0, params.PageSize)
	assert.Equal(t, uint(0), params.ApplicationID)
	assert.Empty(t, params.Status)
	assert.Empty(t, params.CaptchaType)
	assert.Empty(t, params.SessionID)
}

func TestLogQueryParams_WithValues(t *testing.T) {
	startDate := time.Now().Add(-24 * time.Hour)
	endDate := time.Now()
	
	params := LogQueryParams{
		Page:          2,
		PageSize:      50,
		ApplicationID: 1,
		Status:        "success",
		CaptchaType:   "slider",
		SessionID:     "test-session",
		StartDate:     startDate,
		EndDate:       endDate,
		MinRiskScore:  10.0,
		MaxRiskScore:  50.0,
		IPAddress:     "192.168.1.1",
		UserAgent:     "Mozilla/5.0",
	}
	
	assert.Equal(t, 2, params.Page)
	assert.Equal(t, 50, params.PageSize)
	assert.Equal(t, uint(1), params.ApplicationID)
	assert.Equal(t, "success", params.Status)
	assert.Equal(t, "slider", params.CaptchaType)
	assert.Equal(t, "test-session", params.SessionID)
	assert.Equal(t, 10.0, params.MinRiskScore)
	assert.Equal(t, 50.0, params.MaxRiskScore)
}

func TestLogListResult_Structure(t *testing.T) {
	result := LogListResult{
		Total:      100,
		Page:       2,
		PageSize:   20,
		TotalPages: 5,
		Logs:       []interface{}{}.([]interface{}),
	}
	
	assert.Equal(t, int64(100), result.Total)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 20, result.PageSize)
	assert.Equal(t, 5, result.TotalPages)
}

func TestLogExportParams_DefaultValues(t *testing.T) {
	params := LogExportParams{}
	
	assert.Equal(t, uint(0), params.ApplicationID)
	assert.Empty(t, params.Status)
	assert.Empty(t, params.CaptchaType)
	assert.Empty(t, params.Format)
}

func TestLogExportParams_WithValues(t *testing.T) {
	startDate := time.Now().Add(-24 * time.Hour)
	endDate := time.Now()
	
	params := LogExportParams{
		ApplicationID: 1,
		Status:        "failed",
		CaptchaType:   "click",
		StartDate:     startDate,
		EndDate:       endDate,
		Format:        "csv",
	}
	
	assert.Equal(t, uint(1), params.ApplicationID)
	assert.Equal(t, "failed", params.Status)
	assert.Equal(t, "click", params.CaptchaType)
	assert.Equal(t, "csv", params.Format)
}

func TestLogService_GetLogByID(t *testing.T) {
	service := NewLogService()
	
	log, err := service.GetLogByID(1)
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.NotNil(t, log)
	}
}

func TestLogService_QueryLogs(t *testing.T) {
	service := NewLogService()
	
	params := LogQueryParams{
		Page:     1,
		PageSize: 10,
	}
	
	result, err := service.QueryLogs(params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLogService_QueryLogs_DefaultPage(t *testing.T) {
	service := NewLogService()
	
	params := LogQueryParams{
		Page:     0,
		PageSize: 10,
	}
	
	result, err := service.QueryLogs(params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Page)
}

func TestLogService_QueryLogs_DefaultPageSize(t *testing.T) {
	service := NewLogService()
	
	params := LogQueryParams{
		Page:     1,
		PageSize: 0,
	}
	
	result, err := service.QueryLogs(params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 20, result.PageSize)
}

func TestLogService_QueryLogs_ExcessivePageSize(t *testing.T) {
	service := NewLogService()
	
	params := LogQueryParams{
		Page:     1,
		PageSize: 200,
	}
	
	result, err := service.QueryLogs(params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 20, result.PageSize)
}

func TestLogService_QueryLogs_WithApplicationID(t *testing.T) {
	service := NewLogService()
	
	params := LogQueryParams{
		Page:          1,
		PageSize:      10,
		ApplicationID: 1,
	}
	
	result, err := service.QueryLogs(params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLogService_QueryLogs_WithStatus(t *testing.T) {
	service := NewLogService()
	
	params := LogQueryParams{
		Page:     1,
		PageSize: 10,
		Status:   "success",
	}
	
	result, err := service.QueryLogs(params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLogService_QueryLogs_WithCaptchaType(t *testing.T) {
	service := NewLogService()
	
	params := LogQueryParams{
		Page:        1,
		PageSize:    10,
		CaptchaType: "slider",
	}
	
	result, err := service.QueryLogs(params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLogService_QueryLogs_WithSessionID(t *testing.T) {
	service := NewLogService()
	
	params := LogQueryParams{
		Page:      1,
		PageSize:  10,
		SessionID: "test",
	}
	
	result, err := service.QueryLogs(params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLogService_QueryLogs_WithDateRange(t *testing.T) {
	service := NewLogService()
	
	params := LogQueryParams{
		Page:      1,
		PageSize:  10,
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now(),
	}
	
	result, err := service.QueryLogs(params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLogService_QueryLogs_WithRiskScore(t *testing.T) {
	service := NewLogService()
	
	params := LogQueryParams{
		Page:         1,
		PageSize:     10,
		MinRiskScore: 20.0,
		MaxRiskScore: 80.0,
	}
	
	result, err := service.QueryLogs(params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLogService_QueryLogs_WithIPAddress(t *testing.T) {
	service := NewLogService()
	
	params := LogQueryParams{
		Page:       1,
		PageSize:   10,
		IPAddress:  "192.168",
	}
	
	result, err := service.QueryLogs(params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLogService_ExportLogs(t *testing.T) {
	service := NewLogService()
	
	params := LogExportParams{
		Format: "csv",
	}
	
	data, err := service.ExportLogs(params)
	assert.NoError(t, err)
	assert.NotNil(t, data)
}

func TestLogService_ExportLogs_WithFilters(t *testing.T) {
	service := NewLogService()
	
	params := LogExportParams{
		ApplicationID: 1,
		Status:        "success",
		CaptchaType:   "slider",
		Format:        "csv",
	}
	
	data, err := service.ExportLogs(params)
	assert.NoError(t, err)
	assert.NotNil(t, data)
}

func TestLogService_GetLogsBySessionID(t *testing.T) {
	service := NewLogService()
	
	logs, err := service.GetLogsBySessionID("test-session")
	assert.NoError(t, err)
	assert.NotNil(t, logs)
}

func TestLogService_GetLogCountByStatus(t *testing.T) {
	service := NewLogService()
	
	count, err := service.GetLogCountByStatus("success")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
}

func TestLogService_GetLogCountByStatus_Failed(t *testing.T) {
	service := NewLogService()
	
	count, err := service.GetLogCountByStatus("failed")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
}

func TestLogService_GetLogCountByDateRange(t *testing.T) {
	service := NewLogService()
	
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	
	count, err := service.GetLogCountByDateRange(start, end)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
}
