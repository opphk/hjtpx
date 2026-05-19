package service

import (
	"context"
	"testing"
	"time"
)

func TestNewLogAnomalyDetector(t *testing.T) {
	detector := NewLogAnomalyDetector()

	if detector == nil {
		t.Fatal("NewLogAnomalyDetector returned nil")
	}

	if len(detector.anomalyPatterns) == 0 {
		t.Error("No anomaly patterns initialized")
	}

	if len(detector.baselineMetrics) == 0 {
		t.Error("No baseline metrics initialized")
	}
}

func TestLogAnomalyDetector_DetectAnomalies(t *testing.T) {
	detector := NewLogAnomalyDetector()
	ctx := context.Background()

	metrics := OperationalMetrics{
		CPUUsage:     75,
		MemoryUsage:  85,
		ErrorRate:   8,
	}

	anomalies, err := detector.DetectAnomalies(ctx, metrics)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	if anomalies == nil {
		t.Error("DetectAnomalies returned nil")
	}
}

func TestLogAnomalyDetector_MatchPattern(t *testing.T) {
	detector := NewLogAnomalyDetector()

	pattern := &AnomalyPattern{
		Regex:    `(?i)(error|exception)`,
		Keywords: []string{"error", "exception"},
	}

	logs := []LogEntry{
		{Message: "An error occurred"},
		{Message: "Exception thrown"},
		{Message: "Success"},
		{Message: "No error here"},
	}

	matched := detector.matchPattern(logs, pattern)

	if len(matched) != 3 {
		t.Errorf("Expected 3 matched logs, got %d", len(matched))
	}
}

func TestLogAnomalyDetector_ContainsKeywords(t *testing.T) {
	detector := NewLogAnomalyDetector()

	tests := []struct {
		message  string
		keywords []string
		expected bool
	}{
		{"This is an error message", []string{"error"}, true},
		{"Success operation", []string{"error"}, false},
		{"Database connection exception", []string{"database", "exception"}, true},
	}

	for _, tt := range tests {
		result := detector.containsKeywords(tt.message, tt.keywords)
		if result != tt.expected {
			t.Errorf("containsKeywords(%s, %v) = %v, want %v", tt.message, tt.keywords, result, tt.expected)
		}
	}
}

func TestLogAnomalyDetector_GetBaseline(t *testing.T) {
	detector := NewLogAnomalyDetector()
	ctx := context.Background()

	baseline, err := detector.GetBaseline(ctx, "error_count")
	if err != nil {
		t.Fatalf("GetBaseline failed: %v", err)
	}

	if baseline == nil {
		t.Fatal("GetBaseline returned nil")
	}

	if baseline.MetricName != "error_count" {
		t.Errorf("Metric name mismatch: got %s, want error_count", baseline.MetricName)
	}

	_, err = detector.GetBaseline(ctx, "non_existent")
	if err == nil {
		t.Error("Expected error for non-existent metric")
	}
}

func TestLogAnomalyDetector_GetAllBaselines(t *testing.T) {
	detector := NewLogAnomalyDetector()
	ctx := context.Background()

	baselines, err := detector.GetAllBaselines(ctx)
	if err != nil {
		t.Fatalf("GetAllBaselines failed: %v", err)
	}

	if len(baselines) == 0 {
		t.Error("GetAllBaselines returned empty map")
	}
}

func TestLogAnomalyDetector_GetPatterns(t *testing.T) {
	detector := NewLogAnomalyDetector()
	ctx := context.Background()

	patterns, err := detector.GetPatterns(ctx)
	if err != nil {
		t.Fatalf("GetPatterns failed: %v", err)
	}

	if len(patterns) == 0 {
		t.Error("GetPatterns returned empty")
	}
}

func TestLogAnomalyDetector_AnalyzeLogEntry(t *testing.T) {
	detector := NewLogAnomalyDetector()
	ctx := context.Background()

	errorEntry := LogEntry{
		Timestamp: time.Now(),
		Level:     "error",
		Message:   "Database connection error",
		Source:    "api",
	}

	anomaly, err := detector.AnalyzeLogEntry(ctx, errorEntry)
	if err != nil {
		t.Fatalf("AnalyzeLogEntry failed: %v", err)
	}

	if anomaly == nil {
		t.Error("Expected anomaly for error log entry")
	}

	normalEntry := LogEntry{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   "Everything is fine",
		Source:    "api",
	}

	anomaly, err = detector.AnalyzeLogEntry(ctx, normalEntry)
	if err != nil {
		t.Fatalf("AnalyzeLogEntry failed: %v", err)
	}

	if anomaly != nil {
		t.Error("Did not expect anomaly for normal log entry")
	}
}

func TestLogAnomalyDetector_GetAnomalyHistory(t *testing.T) {
	detector := NewLogAnomalyDetector()
	ctx := context.Background()

	history, err := detector.GetAnomalyHistory(ctx, 10)
	if err != nil {
		t.Fatalf("GetAnomalyHistory failed: %v", err)
	}

	if history == nil {
		t.Error("GetAnomalyHistory returned nil")
	}
}

func TestLogAnomalyDetector_SetThreshold(t *testing.T) {
	detector := NewLogAnomalyDetector()
	ctx := context.Background()

	err := detector.SetThreshold(ctx, 2.5)
	if err != nil {
		t.Errorf("SetThreshold failed: %v", err)
	}

	err = detector.SetThreshold(ctx, -1)
	if err == nil {
		t.Error("Expected error for negative threshold")
	}
}

func TestLogAnomalyDetector_SetWindowSize(t *testing.T) {
	detector := NewLogAnomalyDetector()
	ctx := context.Background()

	err := detector.SetWindowSize(ctx, 48*time.Hour)
	if err != nil {
		t.Errorf("SetWindowSize failed: %v", err)
	}

	err = detector.SetWindowSize(ctx, -1*time.Hour)
	if err == nil {
		t.Error("Expected error for negative window size")
	}
}

func TestLogAnomalyDetector_ExportAnomalies(t *testing.T) {
	detector := NewLogAnomalyDetector()
	ctx := context.Background()

	data, err := detector.ExportAnomalies(ctx, "json")
	if err != nil {
		t.Fatalf("ExportAnomalies (json) failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("ExportAnomalies returned empty data")
	}

	csvData, err := detector.ExportAnomalies(ctx, "csv")
	if err != nil {
		t.Fatalf("ExportAnomalies (csv) failed: %v", err)
	}

	if len(csvData) == 0 {
		t.Error("ExportAnomalies (csv) returned empty data")
	}

	_, err = detector.ExportAnomalies(ctx, "invalid")
	if err == nil {
		t.Error("Expected error for invalid format")
	}
}

func TestDetectPatternAnomalies(t *testing.T) {
	detector := NewLogAnomalyDetector()

	errorLogs := make([]LogEntry, 20)
	for i := 0; i < 20; i++ {
		errorLogs[i] = LogEntry{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "Error message",
		}
	}

	anomalies := detector.detectPatternAnomalies(errorLogs)

	if len(anomalies) == 0 {
		t.Error("Expected anomalies from error logs")
	}
}

func TestDetectMetricAnomalies(t *testing.T) {
	detector := NewLogAnomalyDetector()

	metrics := OperationalMetrics{
		CPUUsage:          95,
		MemoryUsage:       92,
		ErrorRate:         15,
		AvgResponseTime:  600,
		CacheHitRate:     45,
	}

	anomalies := detector.detectMetricAnomalies(metrics)

	if len(anomalies) == 0 {
		t.Error("Expected anomalies from abnormal metrics")
	}
}

func TestDetectStatisticalAnomalies(t *testing.T) {
	detector := NewLogAnomalyDetector()
	ctx := context.Background()

	metrics := OperationalMetrics{
		ErrorRate:        15,
		AvgResponseTime:  300,
		RequestThroughput: 2000,
	}

	anomalies := detector.detectStatisticalAnomalies(ctx, metrics)

	if anomalies == nil {
		t.Error("DetectStatisticalAnomalies returned nil")
	}
}

func TestDetectSequenceAnomalies(t *testing.T) {
	detector := NewLogAnomalyDetector()

	logs := []LogEntry{
		{Timestamp: time.Now(), Level: "info"},
		{Timestamp: time.Now().Add(-1 * time.Minute), Level: "error"},
		{Timestamp: time.Now().Add(-2 * time.Minute), Level: "error"},
		{Timestamp: time.Now().Add(-3 * time.Minute), Level: "error"},
		{Timestamp: time.Now().Add(-4 * time.Minute), Level: "error"},
		{Timestamp: time.Now().Add(-5 * time.Minute), Level: "info"},
	}

	anomalies := detector.detectSequenceAnomalies(logs)

	found := false
	for _, anomaly := range anomalies {
		if anomaly.Type == "error_sequence" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected error_sequence anomaly")
	}
}

func TestCalculateMean(t *testing.T) {
	detector := NewLogAnomalyDetector()

	tests := []struct {
		values   []float64
		expected float64
	}{
		{[]float64{1, 2, 3, 4, 5}, 3.0},
		{[]float64{10, 20, 30}, 20.0},
		{[]float64{}, 0.0},
	}

	for _, tt := range tests {
		result := detector.calculateMean(tt.values)
		if result != tt.expected {
			t.Errorf("calculateMean(%v) = %f, want %f", tt.values, result, tt.expected)
		}
	}
}

func TestCalculateStdDev(t *testing.T) {
	detector := NewLogAnomalyDetector()

	values := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	mean := 5.0

	stdDev := detector.calculateStdDev(values, mean)

	if stdDev <= 0 {
		t.Errorf("calculateStdDev returned invalid value: %f", stdDev)
	}
}

func TestUpdateBaseline(t *testing.T) {
	detector := NewLogAnomalyDetector()

	metrics := OperationalMetrics{
		CPUUsage:          65,
		MemoryUsage:      70,
		ErrorRate:        3,
		AvgResponseTime:  150,
	}

	detector.updateBaseline(metrics)

	detector.mu.RLock()
	defer detector.mu.RUnlock()

	baseline := detector.baselineMetrics["error_count"]
	if baseline == nil {
		t.Error("Baseline not updated")
	}

	if baseline.SampleCount < 1 {
		t.Error("Sample count not incremented")
	}
}
