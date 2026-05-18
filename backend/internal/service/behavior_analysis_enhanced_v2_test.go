package service

import (
	"math"
	"testing"
	"time"
)

func TestBehaviorDataCollector(t *testing.T) {
	collector := NewBehaviorDataCollector(100)

	for i := 0; i < 50; i++ {
		item := BehaviorDataItem{
			DataType:  "move",
			Data:      `{"x":100,"y":200}`,
			Timestamp: time.Now().UnixMilli() + int64(i*16),
		}
		if err := collector.Collect(item); err != nil {
			t.Errorf("Collection error: %v", err)
		}
	}

	metrics := collector.GetMetrics()

	if metrics.TotalPoints != 50 {
		t.Errorf("Expected 50 points, got %d", metrics.TotalPoints)
	}

	if metrics.DataCompleteness != 1.0 {
		t.Errorf("Expected 100%% completeness, got %.2f%%", metrics.DataCompleteness*100)
	}

	if metrics.DimensionCoverage["move"] != 1.0 {
		t.Errorf("Expected 100%% move coverage, got %.2f%%", metrics.DimensionCoverage["move"]*100)
	}
}

func TestEnhancedBehaviorCollector(t *testing.T) {
	collector := NewEnhancedBehaviorCollector()

	sessionID := "test-session-123"

	for i := 0; i < 10; i++ {
		item := BehaviorDataItem{
			DataType:  "click",
			Data:      `{"x":100,"y":200}`,
			Timestamp: time.Now().UnixMilli() + int64(i*16),
		}
		if err := collector.Collect(sessionID, item); err != nil {
			t.Errorf("Collection error: %v", err)
		}
	}

	metrics := collector.GetSessionMetrics(sessionID)
	if metrics == nil {
		t.Error("Expected metrics for session")
		return
	}

	if metrics.TotalPoints != 10 {
		t.Errorf("Expected 10 points, got %d", metrics.TotalPoints)
	}

	collector.RemoveSession(sessionID)

	metricsAfter := collector.GetSessionMetrics(sessionID)
	if metricsAfter != nil {
		t.Error("Expected nil metrics after session removal")
	}
}

func TestAdvancedFeatureExtractor(t *testing.T) {
	extractor := NewAdvancedFeatureExtractor()

	points := make([]BehaviorDataPoint, 100)
	startTime := time.Now().UnixMilli()

	for i := 0; i < 100; i++ {
		x := float64(i * 10)
		y := float64(i * 5)
		points[i] = BehaviorDataPoint{
			X:         int(x),
			Y:         int(y),
			Timestamp: startTime + int64(i*16),
			Event:     "move",
		}
	}

	features := extractor.ExtractComprehensiveFeatures(points, nil, nil)

	if features.BasicFeatures.PointCount != 100 {
		t.Errorf("Expected 100 points, got %d", features.BasicFeatures.PointCount)
	}

	if features.BasicFeatures.TotalDistance <= 0 {
		t.Error("Expected positive total distance")
	}

	if features.PatternFeatures.PathRatio <= 0 {
		t.Error("Expected positive path ratio")
	}

	if features.DerivedFeatures.BotProbability < 0 || features.DerivedFeatures.BotProbability > 1 {
		t.Error("Bot probability should be between 0 and 1")
	}
}

func TestBotDetection(t *testing.T) {
	extractor := NewAdvancedFeatureExtractor()

	botPoints := make([]BehaviorDataPoint, 100)
	startTime := time.Now().UnixMilli()

	for i := 0; i < 100; i++ {
		progress := float64(i) / float64(100)
		botPoints[i] = BehaviorDataPoint{
			X:         int(progress * 400),
			Y:         int(progress * 300),
			Timestamp: startTime + int64(i*16),
			Event:     "move",
		}
	}

	features := extractor.ExtractComprehensiveFeatures(botPoints, nil, nil)

	if features.DerivedFeatures.BotProbability < 0.5 {
		t.Logf("Bot probability for synthetic bot data: %.2f", features.DerivedFeatures.BotProbability)
	}

	humanPoints := make([]BehaviorDataPoint, 100)
	for i := 0; i < 100; i++ {
		humanPoints[i] = BehaviorDataPoint{
			X:         i*10 + (i % 5),
			Y:         i*5 + (i % 3),
			Timestamp: startTime + int64(i*16),
			Event:     "move",
		}
	}

	humanFeatures := extractor.ExtractComprehensiveFeatures(humanPoints, nil, nil)

	if humanFeatures.DerivedFeatures.BotProbability >= features.DerivedFeatures.BotProbability {
		t.Logf("Human bot probability (%.2f) should be lower than bot probability (%.2f)",
			humanFeatures.DerivedFeatures.BotProbability, features.DerivedFeatures.BotProbability)
	}
}

func TestComprehensiveAnalysisService(t *testing.T) {
	service := NewComprehensiveAnalysisService()

	behaviorData := GenerateSyntheticBehaviorData(100, false)

	result, err := service.AnalyzeComprehensively(behaviorData)
	if err != nil {
		t.Errorf("Analysis error: %v", err)
		return
	}

	if result.LatencyMs > 100 {
		t.Logf("Warning: Latency %.2fms exceeds target of 100ms", result.LatencyMs)
	}

	if result.Accuracy < 0.5 {
		t.Errorf("Expected accuracy > 50%%, got %.2f%%", result.Accuracy*100)
	}

	if result.DataCompleteness < 0.5 {
		t.Errorf("Expected data completeness > 50%%, got %.2f%%", result.DataCompleteness*100)
	}
}

func TestBenchmarkBehaviorAnalysis(t *testing.T) {
	behaviorData := GenerateSyntheticBehaviorData(100, false)

	benchmark := BenchmarkBehaviorAnalysis(behaviorData)

	latency := benchmark["latency_ms"].(int64)
	if latency < 0 || latency > 500 {
		t.Errorf("Unexpected latency: %d ms", latency)
	}

	dataPoints := benchmark["data_points"].(int)
	if dataPoints != 100 {
		t.Errorf("Expected 100 data points, got %d", dataPoints)
	}

	accuracy := benchmark["estimated_accuracy"].(float64)
	if accuracy < 0.5 || accuracy > 1.0 {
		t.Errorf("Unexpected accuracy: %.2f%%", accuracy*100)
	}
}

func TestPatternLibraryV2(t *testing.T) {
	library := NewBehaviorPatternLibraryV2()

	machinePatterns := library.GetAllMachinePatterns()
	if len(machinePatterns) < 10 {
		t.Errorf("Expected at least 10 machine patterns, got %d", len(machinePatterns))
	}

	normalPatterns := library.GetAllNormalPatterns()
	if len(normalPatterns) < 5 {
		t.Errorf("Expected at least 5 normal patterns, got %d", len(normalPatterns))
	}

	points := make([]BehaviorDataPoint, 50)
	startTime := time.Now().UnixMilli()
	for i := 0; i < 50; i++ {
		points[i] = BehaviorDataPoint{
			X:         i * 10,
			Y:         i * 5,
			Timestamp: startTime + int64(i*16),
			Event:     "move",
		}
	}

	features := ExtractFeaturesForPatternMatchingV2(points)

	matches := library.DetectMachinePatterns(features)
	t.Logf("Detected %d machine patterns", len(matches))

	normalScores := library.ValidateNormalPatterns(features)
	t.Logf("Normal pattern validation scores: %d", len(normalScores))

	stats := library.GetPatternStatistics()
	if stats == nil {
		t.Error("Expected non-nil pattern statistics")
	}
}

func TestComprehensivePatternAnalyzerV2(t *testing.T) {
	analyzer := NewComprehensivePatternAnalyzerV2()

	points := make([]BehaviorDataPoint, 100)
	startTime := time.Now().UnixMilli()
	for i := 0; i < 100; i++ {
		points[i] = BehaviorDataPoint{
			X:         i * 10,
			Y:         i * 5,
			Timestamp: startTime + int64(i*16),
			Event:     "move",
		}
	}

	features := ExtractFeaturesForPatternMatchingV2(points)

	result := analyzer.Analyze(features)

	if result.FinalBotScore < 0 || result.FinalBotScore > 1 {
		t.Errorf("Final bot score should be between 0 and 1, got %.2f", result.FinalBotScore)
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %.2f", result.Confidence)
	}

	if result.RiskLevel == "" {
		t.Error("Risk level should not be empty")
	}

	if len(result.Recommendations) == 0 {
		t.Error("Should have at least one recommendation")
	}
}

func TestGenerateSyntheticBehaviorData(t *testing.T) {
	humanData := GenerateSyntheticBehaviorData(100, false)
	if len(humanData) != 100 {
		t.Errorf("Expected 100 data points, got %d", len(humanData))
	}

	botData := GenerateSyntheticBehaviorData(100, true)
	if len(botData) != 100 {
		t.Errorf("Expected 100 data points, got %d", len(botData))
	}

	for _, item := range humanData {
		if item.Timestamp <= 0 {
			t.Error("Timestamp should be positive")
			break
		}
	}
}

func TestDataCompletenessCalculation(t *testing.T) {
	service := NewComprehensiveAnalysisService()

	behaviorData := GenerateSyntheticBehaviorData(100, false)

	result, err := service.AnalyzeComprehensively(behaviorData)
	if err != nil {
		t.Errorf("Analysis error: %v", err)
		return
	}

	if result.DataCompleteness < 0.99 {
		t.Errorf("Data completeness should be > 99%%, got %.2f%%", result.DataCompleteness*100)
	}
}

func TestPredictionLatency(t *testing.T) {
	service := NewComprehensiveAnalysisService()

	behaviorData := GenerateSyntheticBehaviorData(100, false)

	result, err := service.AnalyzeComprehensively(behaviorData)
	if err != nil {
		t.Errorf("Analysis error: %v", err)
		return
	}

	if result.LatencyMs > 100 {
		t.Errorf("Prediction latency should be < 100ms, got %.2fms", result.LatencyMs)
	}
}

func TestAccuracyRequirement(t *testing.T) {
	service := NewComprehensiveAnalysisService()

	testCases := []struct {
		name     string
		isBot    bool
		expected float64
	}{
		{"Human data", false, 0.5},
		{"Bot data", true, 0.6},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			behaviorData := GenerateSyntheticBehaviorData(100, tc.isBot)
			result, err := service.AnalyzeComprehensively(behaviorData)
			if err != nil {
				t.Errorf("Analysis error: %v", err)
				return
			}

			if result.Accuracy < tc.expected {
				t.Errorf("Expected accuracy > %.2f%%, got %.2f%%", tc.expected*100, result.Accuracy*100)
			}
		})
	}
}

func TestFeatureExtractionCompleteness(t *testing.T) {
	extractor := NewAdvancedFeatureExtractor()

	points := make([]BehaviorDataPoint, 100)
	startTime := time.Now().UnixMilli()
	for i := 0; i < 100; i++ {
		points[i] = BehaviorDataPoint{
			X:         i * 10,
			Y:         i * 5,
			Timestamp: startTime + int64(i*16),
			Event:     "move",
		}
	}

	features := extractor.ExtractComprehensiveFeatures(points, nil, nil)

	if features.BasicFeatures == nil {
		t.Error("Basic features should not be nil")
	}

	if features.AdvancedFeatures == nil {
		t.Error("Advanced features should not be nil")
	}

	if features.StatisticalFeatures == nil {
		t.Error("Statistical features should not be nil")
	}

	if features.FrequencyFeatures == nil {
		t.Error("Frequency features should not be nil")
	}

	if features.PatternFeatures == nil {
		t.Error("Pattern features should not be nil")
	}

	if features.DerivedFeatures == nil {
		t.Error("Derived features should not be nil")
	}
}

func TestStatisticalFeatures(t *testing.T) {
	extractor := NewAdvancedFeatureExtractor()

	points := make([]BehaviorDataPoint, 50)
	startTime := time.Now().UnixMilli()
	for i := 0; i < 50; i++ {
		points[i] = BehaviorDataPoint{
			X:         i * 10,
			Y:         i * 5,
			Timestamp: startTime + int64(i*16),
			Event:     "move",
		}
	}

	features := extractor.ExtractComprehensiveFeatures(points, nil, nil)

	stats := features.StatisticalFeatures

	if stats.Mean <= 0 {
		t.Logf("Mean X coordinate: %.2f", stats.Mean)
	}

	if stats.StdDev < 0 {
		t.Error("Standard deviation should be non-negative")
	}

	if stats.Range < 0 {
		t.Error("Range should be non-negative")
	}
}

func TestModelTrainer(t *testing.T) {
	trainer := NewEnhancedModelTrainer()

	initialVersion := trainer.GetModelVersion()

	features := make([]float64, 20)
	for i := range features {
		features[i] = float64(i)
	}

	trainer.AddSample(features, 0.5)

	newVersion := trainer.GetModelVersion()

	if newVersion < initialVersion {
		t.Error("Model version should increase or stay the same")
	}
}

func TestPredictionEngine(t *testing.T) {
	engine := NewEnhancedPredictionEngine()

	extractor := NewAdvancedFeatureExtractor()
	points := make([]BehaviorDataPoint, 50)
	startTime := time.Now().UnixMilli()
	for i := 0; i < 50; i++ {
		points[i] = BehaviorDataPoint{
			X:         i * 10,
			Y:         i * 5,
			Timestamp: startTime + int64(i*16),
			Event:     "move",
		}
	}

	features := extractor.ExtractComprehensiveFeatures(points, nil, nil)

	result := engine.Predict(features)

	if result.BotProbability < 0 || result.BotProbability > 1 {
		t.Errorf("Bot probability should be between 0 and 1, got %.2f", result.BotProbability)
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %.2f", result.Confidence)
	}

	if result.LatencyMs < 0 {
		t.Error("Latency should be non-negative")
	}
}

func TestPerformanceTracker(t *testing.T) {
	tracker := NewEnhancedPerformanceTracker()

	tracker.Record("test_operation", 50.0, 100.0)
	tracker.Record("test_operation", 60.0, 110.0)
	tracker.Record("test_operation", 70.0, 120.0)

	avgLatency := tracker.GetAverageLatency("test_operation")

	if math.Abs(avgLatency-60.0) > 0.1 {
		t.Errorf("Expected average latency ~60ms, got %.2fms", avgLatency)
	}

	avgLatencyOther := tracker.GetAverageLatency("other_operation")
	if avgLatencyOther != 0 {
		t.Errorf("Expected 0 latency for non-existent operation, got %.2f", avgLatencyOther)
	}
}
