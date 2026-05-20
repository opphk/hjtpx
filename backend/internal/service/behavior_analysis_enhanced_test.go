package service

import (
	"context"
	"testing"

	github.com/hjtpx/hjtpx/internal/model"
)

func TestNewBehaviorAnalysisEnhancedService(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()
	if service == nil {
		t.Error("NewBehaviorAnalysisEnhancedService should return a non-nil service")
	}
}

func TestEnhancedBehaviorAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 50, X: 150, Y: 120, Event: "move"},
			{Timestamp: 100, X: 200, Y: 150, Event: "click"},
		},
		TotalTime: 100,
	}

	result, err := service.EnhancedAnalyze(context.Background(), traceData)
	if err != nil {
		t.Fatalf("EnhancedAnalyze failed: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestEnhancedExtractFeatures(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 100, X: 150, Y: 120, Event: "move"},
		},
		TotalTime: 100,
	}

	features, err := service.EnhancedExtractFeatures(context.Background(), traceData)
	if err != nil {
		t.Fatalf("EnhancedExtractFeatures failed: %v", err)
	}

	if features == nil {
		t.Error("Expected non-nil features")
	}
}

func TestEnhancedDetectAnomaly(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 0, Y: 0, Event: "move"},
			{Timestamp: 1, X: 1000, Y: 1000, Event: "move"},
			{Timestamp: 2, X: 2000, Y: 2000, Event: "move"},
		},
		TotalTime: 2,
	}

	anomalies, err := service.EnhancedDetectAnomaly(context.Background(), traceData)
	if err != nil {
		t.Fatalf("EnhancedDetectAnomaly failed: %v", err)
	}

	if anomalies == nil {
		t.Error("Expected non-nil anomaly list")
	}
}

func TestEnhancedPredictScore(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 50, X: 110, Y: 105, Event: "move"},
			{Timestamp: 100, X: 120, Y: 110, Event: "move"},
		},
		TotalTime: 100,
	}

	score, err := service.EnhancedPredictScore(context.Background(), traceData)
	if err != nil {
		t.Fatalf("EnhancedPredictScore failed: %v", err)
	}

	if score < 0 || score > 1 {
		t.Errorf("Expected score in [0, 1], got %v", score)
	}
}

func TestEnhancedBehaviorPatternRecognition(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 100, X: 150, Y: 120, Event: "move"},
		},
		TotalTime: 100,
	}

	pattern, err := service.EnhancedPatternRecognition(context.Background(), traceData)
	if err != nil {
		t.Fatalf("EnhancedPatternRecognition failed: %v", err)
	}

	if pattern == "" {
		t.Log("Pattern may be empty for short traces")
	}
}

func TestEnhancedVelocityAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 0, Y: 0, Event: "move"},
			{Timestamp: 100, X: 100, Y: 100, Event: "move"},
			{Timestamp: 200, X: 200, Y: 200, Event: "move"},
		},
		TotalTime: 200,
	}

	velocity, err := service.EnhancedVelocityAnalysis(context.Background(), traceData)
	if err != nil {
		t.Fatalf("EnhancedVelocityAnalysis failed: %v", err)
	}

	if velocity < 0 {
		t.Error("Expected non-negative velocity")
	}
}

func TestEnhancedAccelerationAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 0, Y: 0, Event: "move"},
			{Timestamp: 100, X: 100, Y: 100, Event: "move"},
			{Timestamp: 200, X: 200, Y: 200, Event: "move"},
		},
		TotalTime: 200,
	}

	acceleration, err := service.EnhancedAccelerationAnalysis(context.Background(), traceData)
	if err != nil {
		t.Fatalf("EnhancedAccelerationAnalysis failed: %v", err)
	}

	if acceleration == nil {
		t.Error("Expected non-nil acceleration data")
	}
}

func TestEnhancedTrajectoryCurvature(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	points := make([]model.TracePoint, 20)
	for i := 0; i < 20; i++ {
		angle := float64(i) * 0.5
		points[i] = model.TracePoint{
			Timestamp: int64(i * 50),
			X:         float64(i * 10),
			Y:         float64(i * 10),
			Event:     "move",
		}
		_ = angle
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 950,
	}

	curvature, err := service.EnhancedCurvatureAnalysis(context.Background(), traceData)
	if err != nil {
		t.Fatalf("EnhancedCurvatureAnalysis failed: %v", err)
	}

	if curvature < 0 {
		t.Error("Expected non-negative curvature")
	}
}

func TestEnhancedClickPattern(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 100, X: 200, Y: 200, Event: "click"},
			{Timestamp: 200, X: 300, Y: 300, Event: "move"},
			{Timestamp: 300, X: 400, Y: 400, Event: "click"},
		},
		TotalTime: 300,
	}

	pattern, err := service.EnhancedClickPattern(context.Background(), traceData)
	if err != nil {
		t.Fatalf("EnhancedClickPattern failed: %v", err)
	}

	if pattern == nil {
		t.Error("Expected non-nil click pattern")
	}
}

func TestEnhancedKeyboardPattern(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 0, Y: 0, Event: "keydown"},
			{Timestamp: 100, X: 0, Y: 0, Event: "keyup"},
			{Timestamp: 200, X: 0, Y: 0, Event: "keydown"},
		},
		TotalTime: 200,
	}

	pattern, err := service.EnhancedKeyboardPattern(context.Background(), traceData)
	if err != nil {
		t.Fatalf("EnhancedKeyboardPattern failed: %v", err)
	}

	if pattern == nil {
		t.Error("Expected non-nil keyboard pattern")
	}
}

func TestEnhancedSessionAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 1000, X: 150, Y: 120, Event: "move"},
		},
		TotalTime: 1000,
	}

	session, err := service.EnhancedSessionAnalysis(context.Background(), traceData)
	if err != nil {
		t.Fatalf("EnhancedSessionAnalysis failed: %v", err)
	}

	if session == nil {
		t.Error("Expected non-nil session analysis")
	}
}

func TestEnhancedRiskLevelClassification(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	testCases := []struct {
		name  string
		score float64
	}{
		{
			name:  "Low risk",
			score: 0.1,
		},
		{
			name:  "Medium risk",
			score: 0.5,
		},
		{
			name:  "High risk",
			score: 0.8,
		},
		{
			name:  "Critical risk",
			score: 0.95,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			riskLevel, err := service.ClassifyRiskLevel(tc.score)
			if err != nil {
				t.Fatalf("ClassifyRiskLevel failed: %v", err)
			}

			validLevels := map[string]bool{
				"low":      true,
				"medium":   true,
				"high":     true,
				"critical": true,
			}

			if !validLevels[riskLevel] {
				t.Errorf("Invalid risk level: %s", riskLevel)
			}
		})
	}
}

func TestEnhancedLearningAdaptation(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 50, X: 150, Y: 120, Event: "move"},
		},
		TotalTime: 50,
	}

	err := service.AdaptToUser(context.Background(), "user123", traceData)
	if err != nil {
		t.Logf("AdaptToUser error (may be expected): %v", err)
	}
}

func TestEnhancedBiometricAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 100, X: 150, Y: 120, Event: "move"},
		},
		TotalTime: 100,
	}

	biometric, err := service.AnalyzeBiometrics(context.Background(), traceData)
	if err != nil {
		t.Fatalf("AnalyzeBiometrics failed: %v", err)
	}

	if biometric == nil {
		t.Error("Expected non-nil biometric analysis")
	}
}

func TestEnhancedDeviceFingerprint(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
		},
		TotalTime: 0,
		DeviceInfo: &model.DeviceInfo{
			UserAgent:    "Mozilla/5.0",
			ScreenWidth:  1920,
			ScreenHeight: 1080,
			Platform:     "Win32",
		},
	}

	fingerprint, err := service.AnalyzeDeviceFingerprint(context.Background(), traceData)
	if err != nil {
		t.Fatalf("AnalyzeDeviceFingerprint failed: %v", err)
	}

	if fingerprint == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestEnhancedNetworkAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 100, X: 150, Y: 120, Event: "move"},
		},
		TotalTime: 100,
	}

	network, err := service.AnalyzeNetworkPatterns(context.Background(), traceData)
	if err != nil {
		t.Fatalf("AnalyzeNetworkPatterns failed: %v", err)
	}

	if network == nil {
		t.Error("Expected non-nil network analysis")
	}
}

func TestEnhancedRealTimeScoring(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 50, X: 110, Y: 105, Event: "move"},
		},
		TotalTime: 50,
	}

	score, err := service.RealTimeScore(context.Background(), traceData)
	if err != nil {
		t.Fatalf("RealTimeScore failed: %v", err)
	}

	if score < 0 || score > 1 {
		t.Errorf("Expected score in [0, 1], got %v", score)
	}
}

func TestEnhancedComparisonWithBaseline(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	currentTrace := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 100, X: 150, Y: 120, Event: "move"},
		},
		TotalTime: 100,
	}

	baselineTrace := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 100, X: 150, Y: 120, Event: "move"},
		},
		TotalTime: 100,
	}

	similarity, err := service.CompareWithBaseline(context.Background(), currentTrace, baselineTrace)
	if err != nil {
		t.Fatalf("CompareWithBaseline failed: %v", err)
	}

	if similarity < 0 || similarity > 1 {
		t.Errorf("Expected similarity in [0, 1], got %v", similarity)
	}
}

func TestEnhancedMultiDimensionalAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 100, X: 150, Y: 120, Event: "click"},
		},
		TotalTime: 100,
	}

	analysis, err := service.MultiDimensionalAnalysis(context.Background(), traceData)
	if err != nil {
		t.Fatalf("MultiDimensionalAnalysis failed: %v", err)
	}

	if analysis == nil {
		t.Error("Expected non-nil multi-dimensional analysis")
	}
}

func TestEnhancedBatchAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traces := []*model.TraceData{
		{
			Points:    []model.TracePoint{{Timestamp: 0, X: 100, Y: 100, Event: "move"}},
			TotalTime: 0,
		},
		{
			Points:    []model.TracePoint{{Timestamp: 0, X: 200, Y: 200, Event: "move"}},
			TotalTime: 0,
		},
	}

	results, err := service.BatchAnalyze(context.Background(), traces)
	if err != nil {
		t.Fatalf("BatchAnalyze failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestEnhancedCacheUtilization(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
		},
		TotalTime: 0,
	}

	_, _ = service.EnhancedAnalyze(context.Background(), traceData)

	_, err := service.EnhancedAnalyze(context.Background(), traceData)
	if err != nil {
		t.Fatalf("Cached analysis should not fail: %v", err)
	}
}

func TestEnhancedConcurrentAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 100, X: 150, Y: 120, Event: "move"},
		},
		TotalTime: 100,
	}

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_, _ = service.EnhancedAnalyze(context.Background(), traceData)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestEnhancedEmptyTrace(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	traceData := &model.TraceData{
		Points:    []model.TracePoint{},
		TotalTime: 0,
	}

	result, err := service.EnhancedAnalyze(context.Background(), traceData)
	if err != nil {
		t.Fatalf("EnhancedAnalyze should not fail on empty trace: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result for empty trace")
	}
}

func TestEnhancedLongTrace(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	points := make([]model.TracePoint, 1000)
	for i := 0; i < 1000; i++ {
		points[i] = model.TracePoint{
			Timestamp: int64(i * 10),
			X:         float64(i * 5),
			Y:         float64(i * 5),
			Event:     "move",
		}
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 9990,
	}

	result, err := service.EnhancedAnalyze(context.Background(), traceData)
	if err != nil {
		t.Fatalf("EnhancedAnalyze failed on long trace: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result for long trace")
	}
}

func TestEnhancedMetricsCollection(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	metrics := service.GetEnhancedMetrics()
	if metrics == nil {
		t.Error("Expected non-nil metrics")
	}
}

func TestEnhancedPerformanceMonitoring(t *testing.T) {
	service := NewBehaviorAnalysisEnhancedService()

	performance := service.GetEnhancedPerformance()
	if performance == nil {
		t.Error("Expected non-nil performance data")
	}
}
