package trace

import (
	"testing"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestBehaviorSequenceAnalyzer_Init(t *testing.T) {
	analyzer := NewBehaviorSequenceAnalyzer()
	if analyzer == nil {
		t.Fatal("Failed to create BehaviorSequenceAnalyzer")
	}

	if len(analyzer.GetAllPatterns()) == 0 {
		t.Error("Expected default patterns to be initialized")
	}
}

func TestBehaviorSequenceAnalyzer_AddPattern(t *testing.T) {
	analyzer := NewBehaviorSequenceAnalyzer()

	pattern := &SequencePattern{
		ID:          "test_pattern",
		Name:        "Test Pattern",
		Pattern:     []string{"event1", "event2", "event3"},
		Weight:      0.8,
		Description: "Test pattern description",
		RiskLevel:   "medium",
	}

	err := analyzer.AddPattern(pattern)
	if err != nil {
		t.Fatalf("Failed to add pattern: %v", err)
	}

	retrieved, exists := analyzer.GetPattern("test_pattern")
	if !exists {
		t.Error("Pattern should exist")
	}
	if retrieved.Name != "Test Pattern" {
		t.Errorf("Expected pattern name 'Test Pattern', got '%s'", retrieved.Name)
	}
}

func TestBehaviorSequenceAnalyzer_ExtractSequenceFeatures(t *testing.T) {
	analyzer := NewBehaviorSequenceAnalyzer()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
			{X: 30, Y: 30, Timestamp: 300},
			{X: 40, Y: 40, Timestamp: 400},
		},
	}

	features, err := analyzer.ExtractSequenceFeatures(traceData)
	if err != nil {
		t.Fatalf("Failed to extract features: %v", err)
	}

	if features.TotalPoints != 5 {
		t.Errorf("Expected TotalPoints=5, got %d", features.TotalPoints)
	}

	if features.TimeDurationMs != 400 {
		t.Errorf("Expected TimeDurationMs=400, got %d", features.TimeDurationMs)
	}

	if features.VelocityFeatures == nil {
		t.Error("VelocityFeatures should not be nil")
	}
}

func TestBehaviorSequenceAnalyzer_AnalyzeSequence(t *testing.T) {
	analyzer := NewBehaviorSequenceAnalyzer()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 50, Y: 50, Timestamp: 100},
			{X: 100, Y: 100, Timestamp: 200},
			{X: 150, Y: 150, Timestamp: 300},
			{X: 200, Y: 200, Timestamp: 400},
			{X: 250, Y: 250, Timestamp: 500},
			{X: 300, Y: 300, Timestamp: 600},
		},
	}

	result, err := analyzer.AnalyzeSequence(traceData)
	if err != nil {
		t.Fatalf("Failed to analyze sequence: %v", err)
	}

	if result.Features == nil {
		t.Error("Features should not be nil")
	}

	if result.RiskScore < 0 || result.RiskScore > 100 {
		t.Errorf("RiskScore should be between 0 and 100, got %f", result.RiskScore)
	}
}

func TestBehaviorSequenceAnalyzer_DetectAnomalousTransitions(t *testing.T) {
	analyzer := NewBehaviorSequenceAnalyzer()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
		},
	}

	analyzer.UpdateTransitionMatrix(traceData)

	anomalies := analyzer.DetectAnomalousTransitions(traceData)
	if anomalies == nil {
		t.Error("Anomalies should not be nil")
	}
}

func TestBehaviorSequenceAnalyzer_TrainOnSequence(t *testing.T) {
	analyzer := NewBehaviorSequenceAnalyzer()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
		},
	}

	analyzer.TrainOnSequence(traceData, true)

	pattern, exists := analyzer.GetPattern("rapid_login_attempts")
	if !exists {
		t.Error("Pattern should exist")
	}

	if pattern.Weight < 0.85 {
		t.Errorf("Expected weight >= 0.85 after training, got %f", pattern.Weight)
	}
}

func TestBehaviorSequenceAnalyzer_ExportImportPatterns(t *testing.T) {
	analyzer := NewBehaviorSequenceAnalyzer()

	data, err := analyzer.ExportPatterns()
	if err != nil {
		t.Fatalf("Failed to export patterns: %v", err)
	}

	if len(data) == 0 {
		t.Error("Exported data should not be empty")
	}

	newAnalyzer := NewBehaviorSequenceAnalyzer()
	err = newAnalyzer.ImportPatterns(data)
	if err != nil {
		t.Fatalf("Failed to import patterns: %v", err)
	}

	if len(newAnalyzer.GetAllPatterns()) == 0 {
		t.Error("Patterns should be imported")
	}
}

func TestBehaviorSequenceAnalyzer_RemovePattern(t *testing.T) {
	analyzer := NewBehaviorSequenceAnalyzer()

	err := analyzer.RemovePattern("rapid_login_attempts")
	if err != nil {
		t.Fatalf("Failed to remove pattern: %v", err)
	}

	_, exists := analyzer.GetPattern("rapid_login_attempts")
	if exists {
		t.Error("Pattern should be removed")
	}
}

func TestCalculateDuration(t *testing.T) {
	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
		},
	}

	duration := calculateDuration(traceData)
	if duration != 200 {
		t.Errorf("Expected duration 200, got %d", duration)
	}
}

func TestCalculateEventDensity(t *testing.T) {
	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
		},
	}

	density := calculateEventDensity(traceData)
	expected := float64(3) / float64(200) * 1000
	if density != expected {
		t.Errorf("Expected density %f, got %f", expected, density)
	}
}

func TestCalculateSequenceEntropy(t *testing.T) {
	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 0, Timestamp: 100},
			{X: 20, Y: 0, Timestamp: 200},
			{X: 30, Y: 0, Timestamp: 300},
		},
	}

	entropy := calculateSequenceEntropy(traceData)
	if entropy < 0 || entropy > 1 {
		t.Errorf("Entropy should be between 0 and 1, got %f", entropy)
	}
}

func TestDetectPeriodicity(t *testing.T) {
	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
			{X: 30, Y: 30, Timestamp: 300},
			{X: 40, Y: 40, Timestamp: 400},
			{X: 50, Y: 50, Timestamp: 500},
			{X: 60, Y: 60, Timestamp: 600},
			{X: 70, Y: 70, Timestamp: 700},
			{X: 80, Y: 80, Timestamp: 800},
			{X: 90, Y: 90, Timestamp: 900},
			{X: 100, Y: 100, Timestamp: 1000},
		},
	}

	periodicity := detectPeriodicity(traceData)
	if periodicity < 0 || periodicity > 1 {
		t.Errorf("Periodicity should be between 0 and 1, got %f", periodicity)
	}
}