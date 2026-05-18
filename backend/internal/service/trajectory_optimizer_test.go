package service

import (
	"math"
	"testing"
)

func TestNewTrajectoryOptimizer(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()
	if optimizer == nil {
		t.Fatal("TrajectoryOptimizer should not be nil")
	}
}

func TestTrajectoryOptimizer_AnalyzeSpeedProfile(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 210, Timestamp: 1100},
		{X: 200, Y: 205, Timestamp: 1200},
		{X: 250, Y: 215, Timestamp: 1300},
		{X: 300, Y: 210, Timestamp: 1400},
		{X: 350, Y: 220, Timestamp: 1500},
	}

	profile := optimizer.AnalyzeSpeedProfile(trajectory)

	if profile.AverageSpeed <= 0 {
		t.Errorf("AverageSpeed should be positive, got %f", profile.AverageSpeed)
	}

	if profile.MaxSpeed <= 0 {
		t.Errorf("MaxSpeed should be positive, got %f", profile.MaxSpeed)
	}

	if profile.MinSpeed <= 0 {
		t.Errorf("MinSpeed should be positive, got %f", profile.MinSpeed)
	}

	t.Logf("Speed Profile - Avg: %.2f, Max: %.2f, Min: %.2f, Variance: %.6f",
		profile.AverageSpeed, profile.MaxSpeed, profile.MinSpeed, profile.SpeedVariance)
}

func TestTrajectoryOptimizer_AnalyzeSpeedProfile_EmptyTrajectory(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()
	profile := optimizer.AnalyzeSpeedProfile([]SliderPoint{})

	if profile.AverageSpeed != 0 {
		t.Errorf("Empty trajectory should have zero AverageSpeed, got %f", profile.AverageSpeed)
	}
}

func TestTrajectoryOptimizer_AnalyzeSpeedProfile_SinglePoint(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()
	profile := optimizer.AnalyzeSpeedProfile([]SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
	})

	if profile.AverageSpeed != 0 {
		t.Errorf("Single point trajectory should have zero AverageSpeed, got %f", profile.AverageSpeed)
	}
}

func TestTrajectoryOptimizer_AnalyzeSpeedProfile_HighConsistency(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	trajectory := make([]SliderPoint, 50)
	for i := 0; i < 50; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200,
			Timestamp: int64(1000 + i*100),
		}
	}

	profile := optimizer.AnalyzeSpeedProfile(trajectory)

	if profile.SpeedCV > 0.1 {
		t.Logf("Constant speed trajectory has CV: %.6f (expected low for bot-like)", profile.SpeedCV)
	}

	t.Logf("Speed CV: %.6f, IsConsistent: %v", profile.SpeedCV, profile.IsSpeedConsistent)
}

func TestTrajectoryOptimizer_AnalyzeCurvature(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 250, Timestamp: 1100},
		{X: 200, Y: 200, Timestamp: 1200},
		{X: 250, Y: 150, Timestamp: 1300},
		{X: 300, Y: 200, Timestamp: 1400},
	}

	curvature := optimizer.AnalyzeCurvature(trajectory)

	if curvature.AverageCurvature < 0 {
		t.Errorf("AverageCurvature should be non-negative, got %f", curvature.AverageCurvature)
	}

	t.Logf("Curvature - Avg: %.6f, Max: %.6f, Peaks: %d, Entropy: %.4f",
		curvature.AverageCurvature, curvature.MaxCurvature, curvature.PeakCount, curvature.CurvatureEntropy)
}

func TestTrajectoryOptimizer_AnalyzeCurvature_EmptyTrajectory(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()
	curvature := optimizer.AnalyzeCurvature([]SliderPoint{})

	if curvature.AverageCurvature != 0 {
		t.Errorf("Empty trajectory should have zero AverageCurvature, got %f", curvature.AverageCurvature)
	}
}

func TestTrajectoryOptimizer_AnalyzeCurvature_StraightLine(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	trajectory := make([]SliderPoint, 20)
	for i := 0; i < 20; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200,
			Timestamp: int64(1000 + i*100),
		}
	}

	curvature := optimizer.AnalyzeCurvature(trajectory)

	if curvature.PeakCount != 0 {
		t.Logf("Straight line has %d peaks (expected 0 for bot-like)", curvature.PeakCount)
	}

	t.Logf("Straight line curvature - Avg: %.6f, Peaks: %d, SharpTurns: %d",
		curvature.AverageCurvature, curvature.PeakCount, curvature.SharpTurnCount)
}

func TestTrajectoryOptimizer_AnalyzeBacktrack(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
		{X: 250, Y: 200, Timestamp: 1300},
		{X: 350, Y: 200, Timestamp: 1400},
	}

	backtrack := optimizer.AnalyzeBacktrack(trajectory)

	if backtrack.Count < 1 {
		t.Errorf("Should detect at least 1 backtrack, got %d", backtrack.Count)
	}

	t.Logf("Backtrack - Count: %d, TotalDist: %.2f, MaxDist: %.2f",
		backtrack.Count, backtrack.TotalDistance, backtrack.MaxBacktrackDist)
}

func TestTrajectoryOptimizer_AnalyzeBacktrack_NoBacktrack(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	trajectory := make([]SliderPoint, 20)
	for i := 0; i < 20; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200,
			Timestamp: int64(1000 + i*100),
		}
	}

	backtrack := optimizer.AnalyzeBacktrack(trajectory)

	if backtrack.Count != 0 {
		t.Logf("No backtrack trajectory has %d backtracks", backtrack.Count)
	}

	t.Logf("No backtrack result - Count: %d, TotalDist: %.2f", backtrack.Count, backtrack.TotalDistance)
}

func TestTrajectoryOptimizer_AnalyzeSmoothness(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 110, Y: 210, Timestamp: 1100},
		{X: 120, Y: 205, Timestamp: 1200},
		{X: 130, Y: 215, Timestamp: 1300},
		{X: 140, Y: 210, Timestamp: 1400},
	}

	smoothness := optimizer.AnalyzeSmoothness(trajectory)

	if smoothness.OverallScore < 0 || smoothness.OverallScore > 1 {
		t.Errorf("OverallScore should be between 0 and 1, got %f", smoothness.OverallScore)
	}

	t.Logf("Smoothness - Overall: %.4f, Jitter: %.6f, DirectionStability: %.4f",
		smoothness.OverallScore, smoothness.JitterScore, smoothness.DirectionStability)
}

func TestTrajectoryOptimizer_AnalyzeSmoothness_PerfectStraight(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	trajectory := make([]SliderPoint, 30)
	for i := 0; i < 30; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200,
			Timestamp: int64(1000 + i*50),
		}
	}

	smoothness := optimizer.AnalyzeSmoothness(trajectory)

	if smoothness.OverallScore < 0.9 {
		t.Logf("Perfect straight line has smoothness: %.4f (expected high for bot-like)", smoothness.OverallScore)
	}

	t.Logf("Perfect straight smoothness - Overall: %.4f, Jitter: %.6f",
		smoothness.OverallScore, smoothness.JitterScore)
}

func TestTrajectoryOptimizer_ExtractAngularChanges(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 300, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
		{X: 400, Y: 300, Timestamp: 1300},
	}

	changes := optimizer.extractAngularChanges(trajectory)

	if len(changes) == 0 {
		t.Errorf("Should detect angular changes")
	}

	t.Logf("Angular changes count: %d", len(changes))
}

func TestTrajectoryOptimizer_CalculateSkewness(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	skewness := optimizer.calculateSkewness(values)

	t.Logf("Skewness of [1,2,3,4,5]: %.4f", skewness)
}

func TestTrajectoryOptimizer_CalculateKurtosis(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	kurtosis := optimizer.calculateKurtosis(values)

	t.Logf("Kurtosis of [1,2,3,4,5]: %.4f", kurtosis)
}

func TestTrajectoryOptimizer_MeanVariance(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	mean := optimizer.mean(values)
	if math.Abs(mean-3.0) > 0.001 {
		t.Errorf("Mean should be 3.0, got %f", mean)
	}

	variance := optimizer.variance(values)
	if variance <= 0 {
		t.Errorf("Variance should be positive, got %f", variance)
	}
}

func TestTrajectoryOptimizer_Median(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	values := []float64{5.0, 2.0, 3.0, 1.0, 4.0}
	median := optimizer.median(values)

	if median != 3.0 {
		t.Errorf("Median should be 3.0, got %f", median)
	}
}

func TestTrajectoryOptimizer_MaxMin(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	values := []float64{5.0, 2.0, 3.0, 1.0, 4.0}

	max := optimizer.max(values)
	min := optimizer.min(values)

	if max != 5.0 {
		t.Errorf("Max should be 5.0, got %f", max)
	}

	if min != 1.0 {
		t.Errorf("Min should be 1.0, got %f", min)
	}
}

func TestTrajectoryOptimizer_MaxAbs(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	values := []float64{-5.0, 2.0, -3.0, 1.0, 4.0}

	maxAbs := optimizer.maxAbs(values)

	if maxAbs != 5.0 {
		t.Errorf("MaxAbs should be 5.0, got %f", maxAbs)
	}
}

func TestTrajectoryOptimizer_SmoothTrajectory(t *testing.T) {
	optimizer := NewTrajectoryOptimizer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 110, Y: 210, Timestamp: 1100},
		{X: 120, Y: 220, Timestamp: 1200},
		{X: 130, Y: 230, Timestamp: 1300},
	}

	smoothed := optimizer.smoothTrajectory(trajectory, 3)

	if len(smoothed) != len(trajectory) {
		t.Errorf("Smoothed trajectory should have same length as original")
	}

	for i, p := range smoothed {
		if p.Timestamp != trajectory[i].Timestamp {
			t.Errorf("Timestamp should be preserved at index %d", i)
		}
	}
}

func BenchmarkTrajectoryOptimizer_AnalyzeSpeedProfile(b *testing.B) {
	optimizer := NewTrajectoryOptimizer()

	trajectory := make([]SliderPoint, 100)
	for i := 0; i < 100; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200 + i%20 - 10,
			Timestamp: int64(1000 + i*50),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.AnalyzeSpeedProfile(trajectory)
	}
}

func BenchmarkTrajectoryOptimizer_AnalyzeCurvature(b *testing.B) {
	optimizer := NewTrajectoryOptimizer()

	trajectory := make([]SliderPoint, 100)
	for i := 0; i < 100; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200 + i%20 - 10,
			Timestamp: int64(1000 + i*50),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.AnalyzeCurvature(trajectory)
	}
}

func BenchmarkTrajectoryOptimizer_AnalyzeSmoothness(b *testing.B) {
	optimizer := NewTrajectoryOptimizer()

	trajectory := make([]SliderPoint, 100)
	for i := 0; i < 100; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200 + i%20 - 10,
			Timestamp: int64(1000 + i*50),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.AnalyzeSmoothness(trajectory)
	}
}
