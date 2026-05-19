package service

import (
	"testing"
)

func TestNewLogService(t *testing.T) {
	logService := NewLogService()
	if logService == nil {
		t.Error("NewLogService 返回了 nil")
	}
}

func TestLogInfo(t *testing.T) {
	logService := NewLogService()
	
	err := logService.Info("test info message")
	if err != nil {
		t.Errorf("记录 info 日志失败: %v", err)
	}
}

func TestLogError(t *testing.T) {
	logService := NewLogService()
	
	err := logService.Error("test error message")
	if err != nil {
		t.Errorf("记录 error 日志失败: %v", err)
	}
}

func TestLogWarn(t *testing.T) {
	logService := NewLogService()
	
	err := logService.Warn("test warning message")
	if err != nil {
		t.Errorf("记录 warning 日志失败: %v", err)
	}
}

func TestLogDebug(t *testing.T) {
	logService := NewLogService()
	
	err := logService.Debug("test debug message")
	if err != nil {
		t.Errorf("记录 debug 日志失败: %v", err)
	}
}

func TestLogWithFields(t *testing.T) {
	logService := NewLogService()
	
	fields := map[string]interface{}{
		"user_id": 123,
		"action":  "test",
	}
	
	err := logService.LogWithFields("info", "test message with fields", fields)
	if err != nil {
		t.Errorf("记录带字段的日志失败: %v", err)
	}
}

func TestGetLogs(t *testing.T) {
	logService := NewLogService()
	
	err := logService.Info("test message for get")
	if err != nil {
		t.Skipf("无法记录日志，跳过测试: %v", err)
	}
	
	logs, err := logService.GetLogs(0, 10)
	if err != nil {
		t.Errorf("获取日志失败: %v", err)
	}
	if logs == nil {
		t.Error("日志列表不应为 nil")
	}
}

func TestSearchLogs(t *testing.T) {
	logService := NewLogService()
	
	err := logService.Info("unique_test_message_12345")
	if err != nil {
		t.Skipf("无法记录日志，跳过测试: %v", err)
	}
	
	logs, err := logService.SearchLogs("unique_test_message_12345", 0, 10)
	if err != nil {
		t.Errorf("搜索日志失败: %v", err)
	}
	if logs == nil {
		t.Error("搜索结果不应为 nil")
	}
}

func TestClearLogs(t *testing.T) {
	logService := NewLogService()
	
	err := logService.ClearLogs()
	if err != nil {
		t.Errorf("清空日志失败: %v", err)
	}
}

func TestExportLogs(t *testing.T) {
	logService := NewLogService()
	
	export, err := logService.ExportLogsForTest("2024-01-01", "2024-12-31")
	if err != nil {
		t.Errorf("导出日志失败: %v", err)
	}
	if export == "" {
		t.Error("导出的日志不应为空")
	}
}

func TestSetLogLevel(t *testing.T) {
	logService := NewLogService()
	
	err := logService.SetLogLevel("debug")
	if err != nil {
		t.Errorf("设置日志级别失败: %v", err)
	}
	
	err = logService.Info("test after level change")
	if err != nil {
		t.Errorf("设置级别后记录日志失败: %v", err)
	}
}

func TestGetLogStats(t *testing.T) {
	logService := NewLogService()
	
	stats, err := logService.GetLogStats()
	if err != nil {
		t.Errorf("获取日志统计失败: %v", err)
	}
	if stats == nil {
		t.Error("日志统计不应为 nil")
	}
}
