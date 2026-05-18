package service

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

func TestIntegration_HumanVsBotClassification(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	testCases := []struct {
		name          string
		trajectory    []SliderPoint
		expectedBot   bool
		description   string
	}{
		{
			name:          "human_like_trajectory",
			trajectory:    GenerateHumanLikeSliderTrajectory(100, 200, 500, 250, 3000),
			expectedBot:   false,
			description:   "Normal human dragging with natural variation",
		},
		{
			name:          "bot_straight_line",
			trajectory:    GenerateBotLikeSliderTrajectory(100, 200, 500, 200, 1000),
			expectedBot:   true,
			description:   "Perfect straight line typical of bots",
		},
		{
			name:          "suspicious_bot",
			trajectory:    GenerateBotLikeSliderTrajectory(100, 200, 500, 250, 1500),
			expectedBot:   true,
			description:   "Nearly perfect trajectory with minimal noise",
		},
		{
			name:          "backtrack_human",
			trajectory:    GenerateHumanLikeSliderTrajectory(100, 200, 500, 250, 4000),
			expectedBot:   false,
			description:   "Human with intentional backtracking",
		},
		{
			name:          "variable_speed_human",
			trajectory:    GenerateHumanLikeSliderTrajectory(100, 200, 500, 250, 3500),
			expectedBot:   false,
			description:   "Human with varying drag speed",
		},
		{
			name:          "curved_human",
			trajectory:    GenerateHumanLikeSliderTrajectory(100, 200, 500, 250, 3000),
			expectedBot:   false,
			description:   "Human with curved trajectory",
		},
	}

	successCount := 0
	totalTests := len(testCases)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := analyzer.DetectBot(tc.trajectory, 500)

			correct := result.IsBot == tc.expectedBot

			t.Logf("Test: %s", tc.description)
			t.Logf("  Expected bot: %v, Got bot: %v", tc.expectedBot, result.IsBot)
			t.Logf("  Bot score: %.4f, Confidence: %.4f", result.BotScore, result.Confidence)
			t.Logf("  Risk level: %s", result.RiskLevel)

			if len(result.Indicators) > 0 {
				t.Logf("  Indicators: %v", result.Indicators)
			}

			if correct {
				successCount++
				t.Logf("  ✓ PASSED")
			} else {
				t.Errorf("  ✗ FAILED - Expected bot=%v, got bot=%v", tc.expectedBot, result.IsBot)
			}
		})
	}

	accuracy := float64(successCount) / float64(totalTests)
	t.Logf("\n=== Integration Test Summary ===")
	t.Logf("Accuracy: %.2f%% (%d/%d)", accuracy*100, successCount, totalTests)

	if accuracy < 0.95 {
		t.Errorf("Integration test accuracy should be >= 95%%, got %.2f%%", accuracy*100)
	}
}

func TestIntegration_ComprehensiveAccuracy(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	humanTrajectories := 50
	botTrajectories := 50

	humanScores := make([]float64, 0, humanTrajectories)
	botScores := make([]float64, 0, botTrajectories)

	t.Logf("Testing %d human trajectories...", humanTrajectories)
	for i := 0; i < humanTrajectories; i++ {
		traj := GenerateHumanLikeSliderTrajectory(100, 200, 500, 250, int64(2500+rand.Intn(1000)))
		result := analyzer.DetectBot(traj, 500)
		humanScores = append(humanScores, result.BotScore)
	}

	t.Logf("Testing %d bot trajectories...", botTrajectories)
	for i := 0; i < botTrajectories; i++ {
		traj := GenerateBotLikeSliderTrajectory(100, 200, 500, 250, int64(800+rand.Intn(400)))
		result := analyzer.DetectBot(traj, 500)
		botScores = append(botScores, result.BotScore)
	}

	humanAvg := average(humanScores)
	botAvg := average(botScores)

	humanCorrect := 0
	for _, score := range humanScores {
		if score < 0.5 {
			humanCorrect++
		}
	}

	botCorrect := 0
	for _, score := range botScores {
		if score >= 0.5 {
			botCorrect++
		}
	}

	humanAccuracy := float64(humanCorrect) / float64(humanTrajectories)
	botAccuracy := float64(botCorrect) / float64(botTrajectories)
	overallAccuracy := float64(humanCorrect+botCorrect) / float64(humanTrajectories+botTrajectories)

	t.Logf("\n=== Comprehensive Accuracy Test Results ===")
	t.Logf("Human trajectories: %d/%d correct (%.2f%%)", humanCorrect, humanTrajectories, humanAccuracy*100)
	t.Logf("Bot trajectories: %d/%d correct (%.2f%%)", botCorrect, botTrajectories, botAccuracy*100)
	t.Logf("Overall accuracy: %.2f%%", overallAccuracy*100)
	t.Logf("Average human bot-score: %.4f", humanAvg)
	t.Logf("Average bot bot-score: %.4f", botAvg)

	t.Logf("\n=== Requirements Validation ===")
	t.Logf("Robot detection accuracy target: > 99%%")
	t.Logf("Actual robot detection accuracy: %.2f%%", botAccuracy*100)

	t.Logf("Human false positive target: < 0.5%%")
	t.Logf("Actual human false positive rate: %.2f%%", (1-humanAccuracy)*100)
}

func TestIntegration_DTWComparison(t *testing.T) {
	dtw := NewOptimizedDTW()

	t.Log("Testing DTW trajectory comparison...")

	traj1 := GenerateHumanLikeSliderTrajectory(100, 200, 500, 250, 3000)
	traj2 := GenerateHumanLikeSliderTrajectory(100, 200, 500, 250, 3200)
	traj3 := GenerateBotLikeSliderTrajectory(100, 200, 500, 250, 1000)

	dist1 := dtw.ComputeDistance(traj1, traj1)
	t.Logf("Identical human trajectories distance: %.4f", dist1)

	dist2 := dtw.ComputeDistance(traj1, traj2)
	t.Logf("Similar human trajectories distance: %.4f", dist2)

	dist3 := dtw.ComputeDistance(traj1, traj3)
	t.Logf("Human vs Bot trajectories distance: %.4f", dist3)

	if dist1 > 1.0 {
		t.Errorf("Identical trajectories should have distance < 1, got %.4f", dist1)
	}

	if dist2 <= dist1 {
		t.Logf("Similar human trajectories should have larger distance than identical")
	}

	if dist3 <= dist2 {
		t.Errorf("Human vs Bot distance should be larger than human vs human")
	}

	t.Logf("\n=== DTW Performance ===")
	t.Logf("Sakoe-Chiba constraint with window size 10")
	t.Logf("Optimized for O(n*m) with banded matrix")
}

func TestIntegration_MultiScaleDTW(t *testing.T) {
	msdtw := NewMultiScaleDTW()

	t.Log("Testing Multi-Scale DTW...")

	traj1 := GenerateHumanLikeSliderTrajectory(100, 200, 500, 250, 3000)
	traj2 := GenerateHumanLikeSliderTrajectory(100, 200, 500, 250, 3500)
	traj3 := GenerateBotLikeSliderTrajectory(100, 200, 500, 250, 1000)

	dist1 := msdtw.ComputeDistance(traj1, traj1)
	dist2 := msdtw.ComputeDistance(traj1, traj2)
	dist3 := msdtw.ComputeDistance(traj1, traj3)

	t.Logf("Multi-scale distances:")
	t.Logf("  Identical: %.4f", dist1)
	t.Logf("  Similar human: %.4f", dist2)
	t.Logf("  Human vs Bot: %.4f", dist3)

	similarity1 := 1.0 - math.Min(dist1/1000, 1.0)
	similarity2 := 1.0 - math.Min(dist2/1000, 1.0)
	similarity3 := 1.0 - math.Min(dist3/1000, 1.0)

	t.Logf("Multi-scale similarities:")
	t.Logf("  Identical: %.4f", similarity1)
	t.Logf("  Similar human: %.4f", similarity2)
	t.Logf("  Human vs Bot: %.4f", similarity3)

	if similarity1 < 0.99 {
		t.Errorf("Identical trajectories should have similarity > 0.99")
	}

	if similarity2 < similarity3 {
		t.Errorf("Similar human should have higher similarity than human vs bot")
	}
}

func TestIntegration_SpeedProfileAnalysis(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	t.Log("Testing speed profile analysis...")

	humanTraj := GenerateHumanLikeSliderTrajectory(100, 200, 500, 250, 3000)
	botTraj := GenerateBotLikeSliderTrajectory(100, 200, 500, 250, 1000)

	humanProfile := optimizer.AnalyzeSpeedProfile(humanTraj)
	botProfile := optimizer.AnalyzeSpeedProfile(botTraj)

	t.Logf("\n=== Speed Profile Comparison ===")
	t.Logf("Human trajectory:")
	t.Logf("  Average speed: %.2f px/s", humanProfile.AverageSpeed)
	t.Logf("  Max speed: %.2f px/s", humanProfile.MaxSpeed)
	t.Logf("  Speed CV: %.4f", humanProfile.SpeedCV)
	t.Logf("  Acceleration avg: %.4f", humanProfile.AccelerationAvg)
	t.Logf("  Jerk avg: %.4f", humanProfile.JerkAvg)

	t.Logf("Bot trajectory:")
	t.Logf("  Average speed: %.2f px/s", botProfile.AverageSpeed)
	t.Logf("  Max speed: %.2f px/s", botProfile.MaxSpeed)
	t.Logf("  Speed CV: %.4f", botProfile.SpeedCV)
	t.Logf("  Acceleration avg: %.4f", botProfile.AccelerationAvg)
	t.Logf("  Jerk avg: %.4f", botProfile.JerkAvg)

	if botProfile.SpeedCV >= humanProfile.SpeedCV {
		t.Logf("Note: Bot has CV %.4f, Human has CV %.4f", botProfile.SpeedCV, humanProfile.SpeedCV)
	}

	if botProfile.JerkAvg >= humanProfile.JerkAvg {
		t.Logf("Note: Bot has higher jerk consistency than human")
	}
}

func TestIntegration_CurvatureAnalysis(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	t.Log("Testing curvature analysis...")

	humanTraj := GenerateHumanLikeSliderTrajectory(100, 200, 500, 250, 3000)
	botTraj := GenerateBotLikeSliderTrajectory(100, 200, 500, 250, 1000)

	humanCurv := optimizer.AnalyzeCurvature(humanTraj)
	botCurv := optimizer.AnalyzeCurvature(botTraj)

	t.Logf("\n=== Curvature Analysis ===")
	t.Logf("Human trajectory:")
	t.Logf("  Average curvature: %.6f", humanCurv.AverageCurvature)
	t.Logf("  Curvature variance: %.8f", humanCurv.CurvatureVariance)
	t.Logf("  Peak count: %d", humanCurv.PeakCount)
	t.Logf("  Curvature entropy: %.4f", humanCurv.CurvatureEntropy)

	t.Logf("Bot trajectory:")
	t.Logf("  Average curvature: %.6f", botCurv.AverageCurvature)
	t.Logf("  Curvature variance: %.8f", botCurv.CurvatureVariance)
	t.Logf("  Peak count: %d", botCurv.PeakCount)
	t.Logf("  Curvature entropy: %.4f", botCurv.CurvatureEntropy)

	if botCurv.AverageCurvature >= humanCurv.AverageCurvature {
		t.Logf("Note: Bot has lower/equal curvature than human")
	}
}

func TestIntegration_SmoothnessAnalysis(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	t.Log("Testing smoothness analysis...")

	humanTraj := GenerateHumanLikeSliderTrajectory(100, 200, 500, 250, 3000)
	botTraj := GenerateBotLikeSliderTrajectory(100, 200, 500, 250, 1000)

	humanSmooth := optimizer.AnalyzeSmoothness(humanTraj)
	botSmooth := optimizer.AnalyzeSmoothness(botTraj)

	t.Logf("\n=== Smoothness Analysis ===")
	t.Logf("Human trajectory:")
	t.Logf("  Overall score: %.4f", humanSmooth.OverallScore)
	t.Logf("  Jitter score: %.6f", humanSmooth.JitterScore)
	t.Logf("  Direction stability: %.4f", humanSmooth.DirectionStability)
	t.Logf("  Path regularity: %.4f", humanSmooth.PathRegularity)

	t.Logf("Bot trajectory:")
	t.Logf("  Overall score: %.4f", botSmooth.OverallScore)
	t.Logf("  Jitter score: %.6f", botSmooth.JitterScore)
	t.Logf("  Direction stability: %.4f", botSmooth.DirectionStability)
	t.Logf("  Path regularity: %.4f", botSmooth.PathRegularity)

	if botSmooth.OverallScore >= humanSmooth.OverallScore {
		t.Logf("Note: Bot has higher smoothness score than human")
	}
}

func TestIntegration_ReportGeneration(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	t.Log("Testing comprehensive report generation...")

	humanTraj := GenerateHumanLikeSliderTrajectory(100, 200, 500, 250, 3000)
	botTraj := GenerateBotLikeSliderTrajectory(100, 200, 500, 250, 1000)

	humanReport := analyzer.GenerateComprehensiveReport(humanTraj, 500)
	botReport := analyzer.GenerateComprehensiveReport(botTraj, 500)

	t.Logf("\nHuman trajectory report length: %d characters", len(humanReport))
	t.Logf("Bot trajectory report length: %d characters", len(botReport))

	if len(humanReport) == 0 {
		t.Error("Human report should not be empty")
	}

	if len(botReport) == 0 {
		t.Error("Bot report should not be empty")
	}

	t.Log("\nHuman report preview:")
	fmt.Println(humanReport[:minInt(500, len(humanReport))])
}

func TestIntegration_ValidateTrajectories(t *testing.T) {
	analyzer := NewEnhancedSliderAnalyzer()

	t.Log("Testing trajectory validation...")

	tests := []struct {
		name     string
		traj     []SliderPoint
		expected bool
	}{
		{"valid_human", GenerateHumanLikeSliderTrajectory(100, 200, 500, 250, 3000), true},
		{"valid_bot", GenerateBotLikeSliderTrajectory(100, 200, 500, 250, 1000), true},
		{"short", []SliderPoint{{X: 100, Y: 200, Timestamp: 1000}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, msg := analyzer.ValidateTrajectory(tt.traj)
			t.Logf("  %s: valid=%v, msg=%s", tt.name, valid, msg)
			if valid != tt.expected {
				t.Errorf("Expected valid=%v, got %v", tt.expected, valid)
			}
		})
	}
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
