package service

import (
	"math"
	"testing"
)

func TestNewEnhancedSliderAnalyzer(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()
	if analyzer == nil {
		t.Fatal("EnhancedSliderAnalyzer should not be nil")
	}
	if analyzer.optimizer == nil {
		t.Fatal("Optimizer should not be nil")
	}
	if analyzer.dtwOptimizer == nil {
		t.Fatal("DTW optimizer should not be nil")
	}
}

func TestEnhancedSliderAnalyzer_AnalyzeTrajectory(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 210, Timestamp: 1100},
		{X: 200, Y: 205, Timestamp: 1200},
		{X: 250, Y: 215, Timestamp: 1300},
		{X: 300, Y: 210, Timestamp: 1400},
	}

	features := analyzer.AnalyzeTrajectory(trajectory, 500)

	if features.BotScore < 0 || features.BotScore > 1 {
		t.Errorf("BotScore should be between 0 and 1, got %f", features.BotScore)
	}

	if features.HumanScore < 0 || features.HumanScore > 1 {
		t.Errorf("HumanScore should be between 0 and 1, got %f", features.HumanScore)
	}

	if features.Confidence < 0 || features.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", features.Confidence)
	}

	t.Logf("BotScore: %.4f, HumanScore: %.4f, Confidence: %.4f, RiskLevel: %s",
		features.BotScore, features.HumanScore, features.Confidence, features.RiskLevel)
}

func TestEnhancedSliderAnalyzer_AnalyzeTrajectory_InsufficientPoints(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 210, Timestamp: 1100},
	}

	features := analyzer.AnalyzeTrajectory(trajectory, 500)

	if features.BotScore != 1.0 {
		t.Errorf("Insufficient points should result in BotScore 1.0, got %f", features.BotScore)
	}

	if len(features.Indicators) == 0 {
		t.Error("Should have at least one indicator for insufficient points")
	}

	t.Logf("Insufficient points BotScore: %.4f", features.BotScore)
}

func TestEnhancedSliderAnalyzer_AnalyzeTrajectory_BotLike(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	trajectory := make([]SliderPoint, 50)
	for i := 0; i < 50; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200,
			Timestamp: int64(1000 + i*50),
		}
	}

	features := analyzer.AnalyzeTrajectory(trajectory, 500)

	t.Logf("Straight line trajectory - BotScore: %.4f, RiskLevel: %s",
		features.BotScore, features.RiskLevel)

	if features.BotScore < 0.3 {
		t.Logf("Expected higher BotScore for straight line, got %.4f", features.BotScore)
	}
}

func TestEnhancedSliderAnalyzer_AnalyzeTrajectory_HumanLike(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	trajectory := make([]SliderPoint, 50)
	for i := 0; i < 50; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200 + int(math.Sin(float64(i)/5.0)*20),
			Timestamp: int64(1000 + i*50 + i%10*5),
		}
	}

	features := analyzer.AnalyzeTrajectory(trajectory, 500)

	t.Logf("Human-like trajectory - BotScore: %.4f, RiskLevel: %s",
		features.BotScore, features.RiskLevel)
}

func TestEnhancedSliderAnalyzer_DetectBot(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 210, Timestamp: 1100},
		{X: 200, Y: 205, Timestamp: 1200},
		{X: 250, Y: 215, Timestamp: 1300},
		{X: 300, Y: 210, Timestamp: 1400},
	}

	result := analyzer.DetectBot(trajectory, 500)

	if result.BotScore < 0 || result.BotScore > 1 {
		t.Errorf("BotScore should be between 0 and 1, got %f", result.BotScore)
	}

	t.Logf("DetectBot - BotScore: %.4f, IsBot: %v, Confidence: %.4f",
		result.BotScore, result.IsBot, result.Confidence)
}

func TestEnhancedSliderAnalyzer_DetectBot_Empty(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	result := analyzer.DetectBot([]SliderPoint{}, 500)

	if !result.IsBot {
		t.Error("Empty trajectory should be detected as bot")
	}

	if result.BotScore != 1.0 {
		t.Errorf("Empty trajectory should have BotScore 1.0, got %f", result.BotScore)
	}
}

func TestEnhancedSliderAnalyzer_DetectBot_BotLike(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	trajectory := make([]SliderPoint, 50)
	for i := 0; i < 50; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200,
			Timestamp: int64(1000 + i*50),
		}
	}

	result := analyzer.DetectBot(trajectory, 500)

	t.Logf("Straight line - SpeedAnomaly: %v, CurvatureAnomaly: %v, BacktrackAnomaly: %v, SmoothnessAnomaly: %v",
		result.SpeedAnomaly, result.CurvatureAnomaly, result.BacktrackAnomaly, result.SmoothnessAnomaly)

	if len(result.Indicators) == 0 {
		t.Error("Should detect at least one anomaly indicator for bot-like trajectory")
	}
}

func TestEnhancedSliderAnalyzer_DetectBot_HumanLike(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	trajectory := make([]SliderPoint, 100)
	for i := 0; i < 100; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*5 + i%10 - 5,
			Y:         200 + int(math.Sin(float64(i)/8.0)*30) + i%5 - 2,
			Timestamp: int64(1000 + i*40 + i%20*10),
		}
	}

	result := analyzer.DetectBot(trajectory, 500)

	t.Logf("Human-like - BotScore: %.4f, IsBot: %v", result.BotScore, result.IsBot)
}

func TestEnhancedSliderAnalyzer_ClassifyRiskLevel(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	tests := []struct {
		score       float64
		expected    string
	}{
		{0.9, "critical"},
		{0.7, "high"},
		{0.5, "medium"},
		{0.3, "low"},
		{0.1, "minimal"},
	}

	for _, tt := range tests {
		level := analyzer.classifyRiskLevel(tt.score)
		if level != tt.expected {
			t.Errorf("Score %.2f should be %s, got %s", tt.score, tt.expected, level)
		}
	}
}

func TestEnhancedSliderAnalyzer_CompareWithTemplate(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	template := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	candidate := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	similarity := analyzer.CompareWithTemplate(template, candidate)

	if similarity < 0.9 {
		t.Errorf("Identical trajectories should have similarity > 0.9, got %f", similarity)
	}

	t.Logf("Similarity between identical trajectories: %f", similarity)
}

func TestEnhancedSliderAnalyzer_MultiScaleSimilarity(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	traj1 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
		{X: 400, Y: 200, Timestamp: 1300},
		{X: 500, Y: 200, Timestamp: 1400},
		{X: 600, Y: 200, Timestamp: 1500},
	}

	traj2 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
		{X: 400, Y: 200, Timestamp: 1300},
		{X: 500, Y: 200, Timestamp: 1400},
		{X: 600, Y: 200, Timestamp: 1500},
	}

	similarity := analyzer.ComputeMultiScaleSimilarity(traj1, traj2)

	t.Logf("Multi-scale similarity: %f", similarity)

	if similarity < 0.9 {
		t.Errorf("Identical trajectories should have high similarity, got %f", similarity)
	}
}

func TestEnhancedSliderAnalyzer_AddHumanTemplate(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	analyzer.AddHumanTemplate("test_human", trajectory)

	name, similarity := analyzer.ClassifyTrajectory(trajectory)

	t.Logf("Classified as: %s, Similarity: %f", name, similarity)
}

func TestEnhancedSliderAnalyzer_ValidateTrajectory(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	tests := []struct {
		name        string
		trajectory  []SliderPoint
		expectValid bool
	}{
		{
			name:        "valid trajectory",
			trajectory:  generateValidTrajectory(),
			expectValid: true,
		},
		{
			name: "insufficient points",
			trajectory: []SliderPoint{
				{X: 100, Y: 200, Timestamp: 1000},
				{X: 200, Y: 200, Timestamp: 1100},
			},
			expectValid: false,
		},
		{
			name:        "empty trajectory",
			trajectory:  []SliderPoint{},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, _ := analyzer.ValidateTrajectory(tt.trajectory)
			if valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectValid, valid)
			}
		})
	}
}

func TestEnhancedSliderAnalyzer_ValidateTrajectory_Duration(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	trajectory := make([]SliderPoint, 15)
	for i := 0; i < 15; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200,
			Timestamp: int64(1000 + i*5),
		}
	}

	valid, msg := analyzer.ValidateTrajectory(trajectory)

	t.Logf("Duration validation - Valid: %v, Message: %s", valid, msg)
}

func TestEnhancedSliderAnalyzer_ValidateTrajectory_TooFast(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	trajectory := make([]SliderPoint, 15)
	for i := 0; i < 15; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*100,
			Y:         200,
			Timestamp: int64(1000 + i*10),
		}
	}

	valid, msg := analyzer.ValidateTrajectory(trajectory)

	t.Logf("Speed validation - Valid: %v, Message: %s", valid, msg)
}

func TestEnhancedSliderAnalyzer_GenerateComprehensiveReport(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 210, Timestamp: 1100},
		{X: 200, Y: 205, Timestamp: 1200},
		{X: 250, Y: 215, Timestamp: 1300},
		{X: 300, Y: 210, Timestamp: 1400},
	}

	report := analyzer.GenerateComprehensiveReport(trajectory, 500)

	if len(report) == 0 {
		t.Error("Report should not be empty")
	}

	t.Logf("Report length: %d characters", len(report))
}

func TestEnhancedSliderAnalyzer_DetectSpeedAnomaly(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	sp := SpeedProfile{
		SpeedCV:         0.03,
		MaxSpeed:        3000,
		AccelerationMax: 6000,
		JerkAvg:         0.05,
	}

	indicators := make([]string, 0)
	isAnomaly := analyzer.detectSpeedAnomaly(sp, &indicators)

	if !isAnomaly {
		t.Error("Should detect speed anomaly")
	}

	t.Logf("Speed anomaly indicators: %v", indicators)
}

func TestEnhancedSliderAnalyzer_DetectCurvatureAnomaly(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	cu := CurvatureAnalysis{
		AverageCurvature:  0.005,
		PeakCount:         0,
		SharpTurnCount:    0,
		CurvatureVariance: 0.00005,
		CurvatureEntropy:  1.0,
	}

	indicators := make([]string, 0)
	isAnomaly := analyzer.detectCurvatureAnomaly(cu, &indicators)

	if !isAnomaly {
		t.Error("Should detect curvature anomaly")
	}

	t.Logf("Curvature anomaly indicators: %v", indicators)
}

func TestEnhancedSliderAnalyzer_DetectBacktrackAnomaly(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	bt := BacktrackAnalysis{
		Count:            0,
		MaxBacktrackDist: 0,
	}

	indicators := make([]string, 0)
	isAnomaly := analyzer.detectBacktrackAnomaly(bt, &indicators)

	if !isAnomaly {
		t.Error("Should detect backtrack anomaly (no backtrack)")
	}

	t.Logf("Backtrack anomaly indicators: %v", indicators)
}

func TestEnhancedSliderAnalyzer_DetectSmoothnessAnomaly(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	sm := SmoothnessAnalysis{
		OverallScore:       0.98,
		JitterScore:        0.005,
		DirectionStability: 0.99,
		PathRegularity:     0.98,
	}

	indicators := make([]string, 0)
	isAnomaly := analyzer.detectSmoothnessAnomaly(sm, &indicators)

	if !isAnomaly {
		t.Error("Should detect smoothness anomaly")
	}

	t.Logf("Smoothness anomaly indicators: %v", indicators)
}

func TestEnhancedSliderAnalyzer_HighAccuracy(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	successCount := 0
	totalTests := 100

	for i := 0; i < totalTests; i++ {
		isHuman := i%2 == 0
		var trajectory []SliderPoint

		if isHuman {
			trajectory = GenerateHumanLikeSliderTrajectory(100, 200, 500, 200, 3000)
		} else {
			trajectory = GenerateBotLikeSliderTrajectory(100, 200, 500, 200, 1000)
		}

		result := analyzer.DetectBot(trajectory, 500)

		correctlyIdentified := (isHuman && !result.IsBot) || (!isHuman && result.IsBot)
		if correctlyIdentified {
			successCount++
		}
	}

	accuracy := float64(successCount) / float64(totalTests)
	t.Logf("Enhanced analyzer accuracy: %.2f%% (%d/%d)", accuracy*100, successCount, totalTests)

	if accuracy < 0.90 {
		t.Errorf("Accuracy should be > 90%%, got %.2f%%", accuracy*100)
	}
}

func generateValidTrajectory() []SliderPoint {
	trajectory := make([]SliderPoint, 20)
	for i := 0; i < 20; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200 + i%5 - 2,
			Timestamp: int64(1000 + i*100),
		}
	}
	return trajectory
}

func BenchmarkEnhancedSliderAnalyzer_AnalyzeTrajectory(b *testing.B) {
	analyzer := NewEnhancedSliderAnalyzer()

	trajectory := make([]SliderPoint, 100)
	for i := 0; i < 100; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*5 + i%10 - 5,
			Y:         200 + int(math.Sin(float64(i)/8.0)*30),
			Timestamp: int64(1000 + i*40),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.AnalyzeTrajectory(trajectory, 500)
	}
}

func BenchmarkEnhancedSliderAnalyzer_DetectBot(b *testing.B) {
	analyzer := NewEnhancedSliderAnalyzer()

	trajectory := make([]SliderPoint, 100)
	for i := 0; i < 100; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*5 + i%10 - 5,
			Y:         200 + int(math.Sin(float64(i)/8.0)*30),
			Timestamp: int64(1000 + i*40),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.DetectBot(trajectory, 500)
	}
}
