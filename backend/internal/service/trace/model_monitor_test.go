package trace

import (
	"errors"
	"testing"
	"time"
)

func TestEnhancedModelMonitorInitialization(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	if monitor == nil {
		t.Fatal("EnhancedModelMonitor should not be nil")
	}
}

func TestEnhancedModelMonitorStartStop(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	monitor.Start()
	
	if !monitor.IsRunning() {
		t.Error("Monitor should be running after Start()")
	}
	
	monitor.Stop()
	
	if monitor.IsRunning() {
		t.Error("Monitor should be stopped after Stop()")
	}
}

func TestEnhancedModelMonitorRecordPerformance(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	monitor.RecordPerformance("yolo", 0.95, 0.92, 0.94, 0.93, 50.0, 100.0)
	
	metrics := monitor.GetMetrics()
	if len(metrics) == 0 {
		t.Error("Expected metrics after recording performance")
	}
}

func TestEnhancedModelMonitorGetAlerts(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	monitor.triggerAlert("test_model", "accuracy", 50.0, 85.0, AlertLevelCritical)
	
	alerts := monitor.GetAlerts(AlertLevelCritical)
	if len(alerts) == 0 {
		t.Error("Expected critical alerts")
	}
}

func TestEnhancedModelMonitorGetAllAlerts(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	monitor.triggerAlert("model1", "accuracy", 40.0, 85.0, AlertLevelCritical)
	monitor.triggerAlert("model2", "accuracy", 30.0, 85.0, AlertLevelCritical)
	
	allAlerts := monitor.GetAlerts("")
	if len(allAlerts) == 0 {
		t.Error("Expected alerts")
	}
}

func TestEnhancedModelMonitorSetAlertThreshold(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	monitor.RecordPerformance("test_model", 0.9, 0.9, 0.9, 0.9, 50.0, 100.0)
	
	err := monitor.SetAlertThreshold("test_model", "accuracy", 85.0, AlertLevelWarning)
	if err != nil {
		t.Errorf("SetAlertThreshold should not return error: %v", err)
	}
}

func TestEnhancedModelMonitorGetStats(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	stats := monitor.GetStats()
	
	if stats.TotalPredictions != 0 {
		t.Errorf("Expected TotalPredictions 0, got %d", stats.TotalPredictions)
	}
}

func TestEnhancedModelMonitorTriggerAlert(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	monitor.triggerAlert("test", "metric", 100.0, 50.0, AlertLevelCritical)
	
	alerts := monitor.GetAlerts(AlertLevelCritical)
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}
}

func TestEnhancedModelMonitorResolveAlert(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	monitor.triggerAlert("test", "metric", 100.0, 50.0, AlertLevelCritical)
	alerts := monitor.GetAlerts(AlertLevelCritical)
	if len(alerts) != 1 {
		t.Fatal("Expected 1 alert")
	}
	
	err := monitor.ResolveAlert(alerts[0].AlertID)
	if err != nil {
		t.Errorf("ResolveAlert should not return error: %v", err)
	}
	
	alerts = monitor.GetAlerts(AlertLevelCritical)
	if len(alerts) != 0 {
		t.Errorf("Expected 0 unresolved alerts, got %d", len(alerts))
	}
}

func TestEnhancedModelMonitorCleanupOldData(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	for i := 0; i < 5; i++ {
		monitor.RecordPerformance("test", 0.9, 0.9, 0.9, 0.9, 50.0, 100.0)
		time.Sleep(time.Millisecond)
	}
	
	monitor.cleanupOldData()
	
	history := monitor.GetPerformanceHistory("test", time.Now().Add(-1*time.Hour), time.Now())
	if len(history) == 0 {
		t.Error("Expected performance history after cleanup")
	}
}

func TestEnhancedModelMonitorUpdateMetric(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	monitor.mu.Lock()
	monitor.metrics["test_metric"] = &MonitorMetric{
		Name:  "test_metric",
		Value: 0,
		Min:   0,
		Max:   0,
		Avg:   0,
		Count: 0,
	}
	monitor.mu.Unlock()
	
	monitor.updateMetric("test_metric", 10.0)
	monitor.updateMetric("test_metric", 20.0)
	monitor.updateMetric("test_metric", 30.0)
	
	metrics := monitor.GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(metrics))
	}
	
	var metric *MonitorMetric
	for _, m := range metrics {
		metric = m
		break
	}
	
	if metric.Avg != 20.0 {
		t.Errorf("Expected avg 20.0, got %f", metric.Avg)
	}
}

func TestEnhancedModelMonitorFloatToString(t *testing.T) {
	result := floatToString(3.14159)
	if result != "3.14" {
		t.Errorf("Expected '3.14', got '%s'", result)
	}
}

func TestMonitorAlertJSON(t *testing.T) {
	alert := MonitorAlert{
		AlertID:      "test-123",
		Level:        AlertLevelWarning,
		ModelType:    "test_model",
		MetricName:   "accuracy",
		Message:      "Test alert",
		Threshold:    85.0,
		CurrentValue: 75.0,
		Timestamp:    time.Now(),
		Resolved:     false,
	}
	
	if alert.AlertID == "" {
		t.Error("AlertID should not be empty")
	}
	
	if alert.Level != AlertLevelWarning {
		t.Error("Level should be warning")
	}
}

func TestMonitorMetricInitialization(t *testing.T) {
	metric := MonitorMetric{
		Name:      "test",
		ModelType: "model",
		Value:     100.0,
	}
	
	if metric.Name != "test" {
		t.Error("Name should be 'test'")
	}
	
	if metric.Value != 100.0 {
		t.Error("Value should be 100.0")
	}
}

func TestMonitorStatsInitialization(t *testing.T) {
	stats := MonitorStats{
		TotalPredictions: 100,
		TotalErrors:      5,
	}
	
	if stats.TotalPredictions != 100 {
		t.Error("TotalPredictions should be 100")
	}
	
	if stats.TotalErrors != 5 {
		t.Error("TotalErrors should be 5")
	}
}

func TestModelPerformanceHistoryInitialization(t *testing.T) {
	history := ModelPerformanceHistory{
		Timestamp:         time.Now(),
		ModelType:         "test",
		Accuracy:          0.9,
		Precision:         0.85,
		Recall:            0.88,
		F1Score:           0.86,
		AvgResponseTimeMs: 50.0,
		Throughput:        100.0,
	}
	
	if history.ModelType != "test" {
		t.Error("ModelType should be 'test'")
	}
	
	if history.Accuracy != 0.9 {
		t.Error("Accuracy should be 0.9")
	}
}

func TestMinFloat(t *testing.T) {
	result := minFloat(10.0, 5.0)
	if result != 5.0 {
		t.Errorf("Expected 5.0, got %f", result)
	}
}

func TestMaxFloat(t *testing.T) {
	result := maxFloat(10.0, 5.0)
	if result != 10.0 {
		t.Errorf("Expected 10.0, got %f", result)
	}
}

func TestEnhancedModelMonitorGetHealthStatus(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	monitor.RecordPerformance("test_model", 0.95, 0.92, 0.94, 0.93, 50.0, 100.0)
	
	status := monitor.GetHealthStatus("test_model")
	if status == nil {
		t.Error("Health status should not be nil")
	}
}

func TestEnhancedModelMonitorGetComprehensiveReport(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	monitor.RecordPerformance("test", 0.9, 0.9, 0.9, 0.9, 50.0, 100.0)
	
	report := monitor.GetComprehensiveReport()
	if report == nil {
		t.Error("Report should not be nil")
	}
}

func TestEnhancedModelMonitorResetStats(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	monitor.RecordPerformance("test", 0.9, 0.9, 0.9, 0.9, 50.0, 100.0)
	monitor.RecordError("test", errors.New("test error"))
	
	monitor.ResetStats()
	
	stats := monitor.GetStats()
	if stats.TotalPredictions != 0 {
		t.Errorf("Expected TotalPredictions 0 after reset, got %d", stats.TotalPredictions)
	}
}

func TestEnhancedModelMonitorGetAlertCount(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	monitor.triggerAlert("test", "metric", 100.0, 50.0, AlertLevelCritical)
	monitor.triggerAlert("test", "metric2", 90.0, 50.0, AlertLevelWarning)
	
	count := monitor.GetAlertCount()
	if count != 2 {
		t.Errorf("Expected 2 alerts, got %d", count)
	}
}

func TestEnhancedModelMonitorGetActiveAlertCount(t *testing.T) {
	monitor := NewEnhancedModelMonitor()
	
	monitor.triggerAlert("test", "metric", 100.0, 50.0, AlertLevelCritical)
	alerts := monitor.GetAlerts(AlertLevelCritical)
	_ = monitor.ResolveAlert(alerts[0].AlertID)
	
	count := monitor.GetActiveAlertCount()
	if count != 0 {
		t.Errorf("Expected 0 active alerts, got %d", count)
	}
}
