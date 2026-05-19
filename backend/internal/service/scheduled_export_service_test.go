package service

import (
	"testing"
	"time"
)

func TestNewScheduledExportService(t *testing.T) {
	service := NewScheduledExportService(nil)
	if service == nil {
		t.Error("NewScheduledExportService returned nil")
	}
}

func TestNewScheduledExportServiceWithConfig(t *testing.T) {
	config := &ScheduledExportConfig{
		MaxConcurrentExports: 10,
		DefaultTimeout:        15 * time.Minute,
		RetryCount:            5,
		RetryDelay:            2 * time.Minute,
		EnableMetrics:         true,
		MetricsInterval:       2 * time.Minute,
		EnableNotifications:   true,
		DefaultExportFormat:   "csv",
		DefaultExportPath:     "/exports",
	}
	
	service := NewScheduledExportService(config)
	if service == nil {
		t.Error("NewScheduledExportService returned nil")
	}
	
	if service.config.MaxConcurrentExports != 10 {
		t.Errorf("Expected MaxConcurrentExports 10, got %d", service.config.MaxConcurrentExports)
	}
}

func TestScheduledExportConfig(t *testing.T) {
	config := DefaultScheduledExportConfig
	
	if config.MaxConcurrentExports != 5 {
		t.Errorf("Expected MaxConcurrentExports 5, got %d", config.MaxConcurrentExports)
	}
	
	if config.DefaultTimeout != 10*time.Minute {
		t.Errorf("Expected DefaultTimeout 10 minutes, got %v", config.DefaultTimeout)
	}
	
	if config.RetryCount != 3 {
		t.Errorf("Expected RetryCount 3, got %d", config.RetryCount)
	}
	
	if config.EnableMetrics != true {
		t.Error("Expected EnableMetrics to be true")
	}
	
	if config.EnableNotifications != true {
		t.Error("Expected EnableNotifications to be true")
	}
}

func TestTaskExecutor(t *testing.T) {
	config := &ScheduledExportConfig{
		MaxConcurrentExports: 3,
		DefaultTimeout:       5 * time.Second,
	}
	
	executor := NewTaskExecutor(config)
	if executor == nil {
		t.Error("NewTaskExecutor returned nil")
	}
	
	if executor.maxConcurrent != 3 {
		t.Errorf("Expected maxConcurrent 3, got %d", executor.maxConcurrent)
	}
	
	if executor.timeout != 5*time.Second {
		t.Errorf("Expected timeout 5 seconds, got %v", executor.timeout)
	}
}

func TestTaskExecutor_Execute(t *testing.T) {
	executor := NewTaskExecutor(nil)
	
	executed := false
	err := executor.Execute(func() error {
		executed = true
		return nil
	})
	
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	
	if !executed {
		t.Error("Task was not executed")
	}
}

func TestTaskExecutor_ExecuteWithError(t *testing.T) {
	executor := NewTaskExecutor(&ScheduledExportConfig{
		DefaultTimeout: 1 * time.Second,
	})
	
	testErr := &testError{message: "test error"}
	err := executor.Execute(func() error {
		return testErr
	})
	
	if err == nil {
		t.Error("Execute should return error")
	}
}

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

func TestScheduledExportMetrics(t *testing.T) {
	metrics := NewScheduledExportMetrics()
	
	if metrics.TotalExecutions != 0 {
		t.Errorf("Expected TotalExecutions 0, got %d", metrics.TotalExecutions)
	}
	
	if metrics.SuccessfulExecutions != 0 {
		t.Errorf("Expected SuccessfulExecutions 0, got %d", metrics.SuccessfulExecutions)
	}
	
	if metrics.FailedExecutions != 0 {
		t.Errorf("Expected FailedExecutions 0, got %d", metrics.FailedExecutions)
	}
	
	if metrics.AverageExecTime != 0 {
		t.Errorf("Expected AverageExecTime 0, got %v", metrics.AverageExecTime)
	}
}

func TestParseCronExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"* * * * *", 5},
		{"0 * * * *", 5},
		{"0 0 * * *", 5},
		{"0 0 1 * *", 5},
		{"*/5 * * * *", 5},
	}
	
	for _, test := range tests {
		parts := parseCronExpression(test.input)
		if len(parts) != test.expected {
			t.Errorf("Expected %d parts for '%s', got %d", test.expected, test.input, len(parts))
		}
	}
}

func TestScheduledExportService_ConvertToCronExpression(t *testing.T) {
	service := NewScheduledExportService(nil)
	
	validCases := []struct {
		input string
	}{
		{"* * * * *"},
		{"0 * * * *"},
		{"0 0 * * *"},
	}
	
	for _, test := range validCases {
		result := service.convertToCronExpression(test.input)
		if result == "" {
			t.Errorf("convertToCronExpression returned empty for input '%s'", test.input)
		}
	}
}

func TestReportTemplateService(t *testing.T) {
	service := NewReportTemplateService()
	if service == nil {
		t.Error("NewReportTemplateService returned nil")
	}
}

func TestExportHistoryService(t *testing.T) {
	service := NewExportHistoryService()
	if service == nil {
		t.Error("NewExportHistoryService returned nil")
	}
}

func TestNewTaskExecutorWithNilConfig(t *testing.T) {
	executor := NewTaskExecutor(nil)
	if executor == nil {
		t.Error("NewTaskExecutor returned nil for nil config")
	}
}

func TestScheduledExportService_ValidateCronExpression(t *testing.T) {
	service := NewScheduledExportService(nil)
	
	validExprs := []string{
		"* * * * *",
		"0 * * * *",
		"0 0 * * *",
		"0 0 1 * *",
		"*/5 * * * *",
	}
	
	for _, expr := range validExprs {
		err := service.ValidateCronExpression(expr)
		if err != nil {
			t.Errorf("Valid cron expression '%s' returned error: %v", expr, err)
		}
	}
}
