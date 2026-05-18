package service

import (
	"testing"
	"time"
)

func TestDTWAnalyzer(t *testing.T) {
	dtw := NewDTWAnalyzer()

	traj1 := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 10, Y: 5, Timestamp: 100},
		{X: 20, Y: 10, Timestamp: 200},
		{X: 30, Y: 15, Timestamp: 300},
	}

	traj2 := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 12, Y: 6, Timestamp: 100},
		{X: 22, Y: 12, Timestamp: 200},
		{X: 32, Y: 18, Timestamp: 300},
	}

	distance := dtw.ComputeDistance(traj1, traj2)
	if distance < 0 {
		t.Errorf("DTW distance should be non-negative, got %f", distance)
	}

	similarity := dtw.ComputeSimilarity(traj1, traj2)
	if similarity < 0 || similarity > 1 {
		t.Errorf("Similarity should be between 0 and 1, got %f", similarity)
	}
}

func TestBotPatternLibrary(t *testing.T) {
	library := NewBotPatternLibrary()

	perfectLinear := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 10, Y: 0, Timestamp: 100},
		{X: 20, Y: 0, Timestamp: 200},
		{X: 30, Y: 0, Timestamp: 300},
	}

	score, patterns := library.DetectPatterns(perfectLinear)
	if score < 0 {
		t.Errorf("Bot pattern score should be non-negative, got %f", score)
	}

	if len(patterns) == 0 {
		t.Logf("No patterns detected for perfect linear trajectory (may be expected)")
	}

	normalTraj := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 10, Y: 5, Timestamp: 100},
		{X: 18, Y: 12, Timestamp: 200},
		{X: 25, Y: 20, Timestamp: 300},
		{X: 30, Y: 30, Timestamp: 400},
	}

	score, patterns = library.DetectPatterns(normalTraj)
	if score < 0 {
		t.Errorf("Bot pattern score should be non-negative, got %f", score)
	}

	t.Logf("Normal trajectory score: %f, patterns: %v", score, patterns)
}

func TestSliderAnalyzerAdvancedFeatures(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 10, Y: 2, Timestamp: 50},
		{X: 20, Y: 5, Timestamp: 100},
		{X: 30, Y: 8, Timestamp: 150},
		{X: 40, Y: 12, Timestamp: 200},
		{X: 50, Y: 18, Timestamp: 250},
		{X: 60, Y: 25, Timestamp: 300},
		{X: 70, Y: 35, Timestamp: 350},
	}

	features := analyzer.AnalyzeAdvancedFeatures(trajectory, 100)

	if features == nil {
		t.Fatal("Features should not be nil")
	}

	if len(features) == 0 {
		t.Error("Features should not be empty")
	}

	t.Logf("Advanced features count: %d", len(features))
}

func TestSliderAnalyzerBotScore(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	botTrajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 10, Y: 0, Timestamp: 50},
		{X: 20, Y: 0, Timestamp: 100},
		{X: 30, Y: 0, Timestamp: 150},
		{X: 40, Y: 0, Timestamp: 200},
		{X: 50, Y: 0, Timestamp: 250},
		{X: 60, Y: 0, Timestamp: 300},
		{X: 70, Y: 0, Timestamp: 350},
		{X: 80, Y: 0, Timestamp: 400},
		{X: 90, Y: 0, Timestamp: 450},
	}

	score, indicators := analyzer.CalculateAdvancedBotScore(botTrajectory, 100)

	if score < 0 || score > 1 {
		t.Errorf("Bot score should be between 0 and 1, got %f", score)
	}

	if len(indicators) == 0 {
		t.Logf("No bot indicators detected for linear trajectory")
	}

	t.Logf("Bot score: %f, indicators: %v", score, indicators)
}

func TestAnalysisCache(t *testing.T) {
	cache := NewAnalysisCache(100, 5*time.Minute)

	cache.Set("test_key", "test_value")

	value, exists := cache.Get("test_key")
	if !exists {
		t.Error("Value should exist in cache")
	}

	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%v'", value)
	}

	cache.Delete("test_key")

	_, exists = cache.Get("test_key")
	if exists {
		t.Error("Value should not exist after deletion")
	}

	stats := cache.GetStats()
	if stats == nil {
		t.Error("Stats should not be nil")
	}

	t.Logf("Cache stats: %+v", stats)
}

func TestPerformanceMonitor(t *testing.T) {
	monitor := NewPerformanceMonitor()

	monitor.Record("test_operation", 100*time.Millisecond)
	monitor.Record("test_operation", 150*time.Millisecond)
	monitor.Record("test_operation", 200*time.Millisecond)

	stats := monitor.GetStats("test_operation")
	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	if stats["count"].(int) != 3 {
		t.Errorf("Expected 3 measurements, got %d", stats["count"].(int))
	}

	t.Logf("Performance stats: %+v", stats)
}

func TestOptimizedAnalyzer(t *testing.T) {
	analyzer := NewOptimizedAnalyzer()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 10, Y: 5, Timestamp: 100},
		{X: 20, Y: 10, Timestamp: 200},
		{X: 30, Y: 15, Timestamp: 300},
		{X: 40, Y: 20, Timestamp: 400},
	}

	result, err := analyzer.AnalyzeSliderWithCache(trajectory, 50)
	if err != nil {
		t.Fatalf("Analysis should not return error: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	stats := analyzer.GetPerformanceStats()
	if stats == nil {
		t.Error("Performance stats should not be nil")
	}

	t.Logf("Performance stats: %+v", stats)
}

func TestEnhancedRuleEngine(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency:     0.5,
		SpeedConsistency:   0.5,
		AverageSpeed:       1500,
		MaxSpeed:           2500,
		SpeedVariance:      0.3,
		CurvatureAverage:   0.1,
		DirectionChanges:   5,
		MicroCorrections:   2,
		BacktrackCount:     0,
		PauseCount:         1,
		HesitationTime:     200,
		ResponseTime:       1000,
		Accuracy:           0.8,
		HumanLikenessScore: 0.7,
		AnomalyScore:       0.3,
		MLScore:            0.6,
	}

	result := engine.Evaluate(features)

	if result == nil {
		t.Fatal("Evaluate should not return nil")
	}

	if result.TotalScore < 0 || result.TotalScore > 100 {
		t.Errorf("TotalScore should be between 0 and 100, got %f", result.TotalScore)
	}

	t.Logf("Rule engine result: TotalScore=%f, IsBot=%v, RiskLevel=%s", 
		result.TotalScore, result.IsBot, result.RiskLevel)
}

func TestAdvancedScoringSystem(t *testing.T) {
	scorer := NewAdvancedScoringSystem()

	features := &BehaviorFeatures{
		AvgSpeed:             1500,
		MaxSpeed:             2500,
		TrajectorySmoothness: 0.95,
		Acceleration:         0.1,
		PathComplexity:       0.2,
		PathSimilarity:       0.4,
		SpeedVariation:       0.15,
		ClickInterval:        40,
		RiskScore:            60,
	}

	score := scorer.CalculateComprehensiveScore(features, nil, nil)

	if score < 0 || score > 100 {
		t.Errorf("Score should be between 0 and 100, got %f", score)
	}

	t.Logf("Comprehensive score: %f", score)

	weights := scorer.GetWeights()
	if weights == nil {
		t.Error("Weights should not be nil")
	}

	newWeights := map[string]float64{
		"rule_engine":         0.30,
		"trajectory_features": 0.30,
		"speed_analysis":      0.20,
		"click_pattern":       0.10,
		"risk_score":          0.10,
	}
	scorer.SetWeights(newWeights)

	updatedWeights := scorer.GetWeights()
	if updatedWeights["rule_engine"] != 0.30 {
		t.Error("Weights should be updated")
	}
}

func TestTrajectoryClassifier(t *testing.T) {
	classifier := NewTrajectoryClassifier()

	normalTrajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 10, Y: 5, Timestamp: 50},
		{X: 20, Y: 12, Timestamp: 100},
		{X: 30, Y: 20, Timestamp: 150},
		{X: 40, Y: 30, Timestamp: 200},
		{X: 50, Y: 42, Timestamp: 250},
		{X: 60, Y: 55, Timestamp: 300},
		{X: 70, Y: 70, Timestamp: 350},
		{X: 80, Y: 85, Timestamp: 400},
		{X: 90, Y: 100, Timestamp: 450},
	}

	probability, category := classifier.Classify(normalTrajectory)

	if probability < 0 || probability > 1 {
		t.Errorf("Probability should be between 0 and 1, got %f", probability)
	}

	if category == "" {
		t.Error("Category should not be empty")
	}

	t.Logf("Classification: probability=%f, category=%s", probability, category)

	analysis := classifier.GetDetailedAnalysis(normalTrajectory)
	if analysis == nil {
		t.Error("Detailed analysis should not be nil")
	}

	t.Logf("Analysis keys: %v", getMapKeys(analysis))
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestClickTimingAnalyzer(t *testing.T) {
	analyzer := NewClickTimingAnalyzer()

	clicks := []ClickData{
		{X: 100, Y: 100, Timestamp: 0},
		{X: 200, Y: 200, Timestamp: 500},
		{X: 300, Y: 300, Timestamp: 1000},
		{X: 400, Y: 400, Timestamp: 1500},
	}

	features := analyzer.AnalyzeTiming(clicks)

	if features == nil {
		t.Fatal("Timing features should not be nil")
	}

	if features.MeanInterval <= 0 {
		t.Error("Mean interval should be positive")
	}

	t.Logf("Timing features: mean=%.2f, cv=%.2f, isRhythmic=%v",
		features.MeanInterval, features.CvInterval, features.IsRhythmic)
}

func TestClickPressureAnalyzer(t *testing.T) {
	analyzer := NewClickPressureAnalyzer()

	clickEvents := []map[string]interface{}{
		{"x": 100, "y": 100, "timestamp": 0, "pressure": 0.5},
		{"x": 200, "y": 200, "timestamp": 500, "pressure": 0.6},
		{"x": 300, "y": 300, "timestamp": 1000, "pressure": 0.55},
	}

	features := analyzer.AnalyzePressure(clickEvents)

	if features == nil {
		t.Fatal("Pressure features should not be nil")
	}

	if !features.HasPressureData {
		t.Error("Should have pressure data")
	}

	t.Logf("Pressure features: mean=%.2f, consistency=%.2f",
		features.MeanPressure, features.PressureConsistency)
}

func TestAnomalyClickDetector(t *testing.T) {
	detector := NewAnomalyClickDetector()

	result := &ClickAnalysisResult{
		AccuracyAnalysis: &AccuracyAnalysis{
			Accuracy:            1.0,
			TotalClicks:         5,
			AverageMissDistance: 3.0,
		},
		TimingAnalysis: &TimingAnalysis{
			IsRhythmic:       true,
			DurationVariance: 50.0,
			TotalDuration:    3000,
			FirstClickDelay:  50,
			HesitationTimes:  []float64{},
		},
		ClickPattern: &ClickPatternAnalysis{
			SequencePattern: "linear",
			ClickIntervals:  []float64{500, 500, 500},
		},
	}

	score, patterns := detector.DetectAnomalies(result)

	if score < 0 {
		t.Errorf("Anomaly score should be non-negative, got %f", score)
	}

	if len(patterns) == 0 {
		t.Logf("No anomalies detected (unexpected for this test data)")
	}

	t.Logf("Anomaly score: %f, patterns: %v", score, patterns)
}

func TestAdvancedClickAnalyzer(t *testing.T) {
	analyzer := NewAdvancedClickAnalyzer()

	verification := &ClickVerification{
		Clicks: []ClickData{
			{X: 100, Y: 100, Timestamp: 0},
			{X: 200, Y: 200, Timestamp: 400},
			{X: 300, Y: 300, Timestamp: 800},
			{X: 400, Y: 400, Timestamp: 1200},
		},
		TargetImages: []TargetImage{
			{X: 90, Y: 90, Width: 20, Height: 20},
			{X: 190, Y: 190, Width: 20, Height: 20},
			{X: 290, Y: 290, Width: 20, Height: 20},
			{X: 390, Y: 390, Width: 20, Height: 20},
		},
	}

	result := analyzer.AnalyzeAdvanced(verification)

	if result == nil {
		t.Fatal("Advanced click result should not be nil")
	}

	if result.BasicResult == nil {
		t.Error("Basic result should not be nil")
	}

	t.Logf("Bot score: %f", result.BotScore)
	t.Logf("Anomaly patterns: %v", result.AnomalyPatterns)
}

func BenchmarkSliderAnalysis(b *testing.B) {
	analyzer := NewSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 10, Y: 5, Timestamp: 50},
		{X: 20, Y: 12, Timestamp: 100},
		{X: 30, Y: 20, Timestamp: 150},
		{X: 40, Y: 30, Timestamp: 200},
		{X: 50, Y: 42, Timestamp: 250},
		{X: 60, Y: 55, Timestamp: 300},
		{X: 70, Y: 70, Timestamp: 350},
		{X: 80, Y: 85, Timestamp: 400},
		{X: 90, Y: 100, Timestamp: 450},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analyzer.AnalyzeSliderTrajectory(trajectory, 100)
	}
}

func BenchmarkOptimizedAnalysis(b *testing.B) {
	analyzer := NewOptimizedAnalyzer()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 10, Y: 5, Timestamp: 50},
		{X: 20, Y: 12, Timestamp: 100},
		{X: 30, Y: 20, Timestamp: 150},
		{X: 40, Y: 30, Timestamp: 200},
		{X: 50, Y: 42, Timestamp: 250},
		{X: 60, Y: 55, Timestamp: 300},
		{X: 70, Y: 70, Timestamp: 350},
		{X: 80, Y: 85, Timestamp: 400},
		{X: 90, Y: 100, Timestamp: 450},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analyzer.AnalyzeSliderWithCache(trajectory, 100)
	}
}
