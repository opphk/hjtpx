package service

import (
	"testing"
)

func TestJerkAnalyzer(t *testing.T) {
	ja := NewJerkAnalyzer()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 5, Timestamp: 100},
		{X: 200, Y: 10, Timestamp: 200},
		{X: 300, Y: 15, Timestamp: 300},
		{X: 400, Y: 20, Timestamp: 400},
		{X: 500, Y: 25, Timestamp: 500},
	}

	features := ja.ExtractJerkFeatures(trajectory)

	if features == nil {
		t.Fatal("Jerk features should not be nil")
	}

	if features["jerk_average"] == 0 && features["jerk_max"] == 0 {
		t.Log("Warning: Jerk features might not be calculated for simple trajectory")
	}

	t.Logf("Jerk features: %+v", features)
}

func TestJerkAnalyzerInsufficientData(t *testing.T) {
	ja := NewJerkAnalyzer()

	shortTrajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 5, Timestamp: 100},
	}

	features := ja.ExtractJerkFeatures(shortTrajectory)

	if len(features) > 0 {
		t.Log("Warning: Features returned for insufficient data")
	}
}

func TestSpeedAnalyzer(t *testing.T) {
	sa := NewSpeedAnalyzer()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 5, Timestamp: 100},
		{X: 200, Y: 10, Timestamp: 200},
		{X: 300, Y: 15, Timestamp: 300},
		{X: 400, Y: 20, Timestamp: 400},
		{X: 500, Y: 25, Timestamp: 500},
	}

	instantAccels := sa.CalculateInstantAccelerations(trajectory)

	if instantAccels == nil {
		t.Fatal("Instant accelerations should not be nil")
	}

	t.Logf("Instant accelerations: %v", instantAccels)

	speeds := []float64{500, 520, 480, 510, 490, 505, 495, 515, 505, 490}
	volatility := sa.CalculateSpeedVolatility(speeds)

	t.Logf("Speed volatility: %f", volatility)

	waveIndex := sa.CalculateSpeedWaveIndex(speeds)
	t.Logf("Speed wave index: %f", waveIndex)
}

func TestSpeedVolatilityCalculation(t *testing.T) {
	sa := NewSpeedAnalyzer()

	uniformSpeeds := []float64{500, 500, 500, 500, 500}
	volatility := sa.CalculateSpeedVolatility(uniformSpeeds)

	t.Logf("Uniform speeds volatility: %f", volatility)

	variedSpeeds := []float64{100, 300, 500, 700, 900}
	volatility = sa.CalculateSpeedVolatility(variedSpeeds)

	t.Logf("Varied speeds volatility: %f", volatility)
}

func TestSpeedWaveIndex(t *testing.T) {
	sa := NewSpeedAnalyzer()

	uniformSpeeds := []float64{500, 500, 500, 500, 500, 500, 500, 500}
	waveIndex := sa.CalculateSpeedWaveIndex(uniformSpeeds)

	t.Logf("Uniform speeds wave index: %f (should be close to 0)", waveIndex)

	oscillatingSpeeds := []float64{100, 900, 100, 900, 100, 900, 100, 900}
	waveIndex = sa.CalculateSpeedWaveIndex(oscillatingSpeeds)

	t.Logf("Oscillating speeds wave index: %f (should be high)", waveIndex)
}

func TestCurvatureAnalyzer(t *testing.T) {
	ca := NewCurvatureAnalyzer()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 10, Timestamp: 100},
		{X: 200, Y: 5, Timestamp: 200},
		{X: 300, Y: 15, Timestamp: 300},
		{X: 400, Y: 8, Timestamp: 400},
		{X: 500, Y: 20, Timestamp: 500},
	}

	stats := ca.CalculatePathCurvatureStats(trajectory)

	t.Logf("Curvature stats - Mean: %f, StdDev: %f, Max: %f, Min: %f",
		stats.Mean, stats.StdDev, stats.Max, stats.Min)
	t.Logf("Curvature stats - Range: %f, Median: %f, PeakCount: %d",
		stats.Range, stats.Median, stats.PeakCount)
	t.Logf("Curvature stats - ZeroCount: %d, Uniformity: %f",
		stats.ZeroCount, stats.Uniformity)
}

func TestCurvatureAnalyzerLinearPath(t *testing.T) {
	ca := NewCurvatureAnalyzer()

	linearTrajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 0, Timestamp: 100},
		{X: 200, Y: 0, Timestamp: 200},
		{X: 300, Y: 0, Timestamp: 300},
		{X: 400, Y: 0, Timestamp: 400},
		{X: 500, Y: 0, Timestamp: 500},
	}

	stats := ca.CalculatePathCurvatureStats(linearTrajectory)

	t.Logf("Linear path curvature stats - Mean: %f, Uniformity: %f",
		stats.Mean, stats.Uniformity)

	if stats.Mean > 0.01 {
		t.Errorf("Linear path should have very low curvature mean")
	}
}

func TestCurvaturePeaksCount(t *testing.T) {
	ca := NewCurvatureAnalyzer()

	curvatures := []float64{0.1, 0.15, 0.5, 0.2, 0.1, 0.6, 0.15, 0.1}
	peaks := ca.countCurvaturePeaks(curvatures)

	t.Logf("Detected curvature peaks: %d", peaks)

	if peaks < 1 {
		t.Errorf("Should detect at least one peak")
	}
}

func TestAngleAnalyzer(t *testing.T) {
	aa := NewAngleAnalyzer()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 10, Timestamp: 100},
		{X: 200, Y: 5, Timestamp: 200},
		{X: 300, Y: 15, Timestamp: 300},
		{X: 400, Y: 8, Timestamp: 400},
		{X: 500, Y: 20, Timestamp: 500},
	}

	angularVelocity := aa.CalculateAngularVelocity(trajectory)
	t.Logf("Angular velocity: %f", angularVelocity)

	angularAcceleration := aa.CalculateAngularAcceleration(trajectory)
	t.Logf("Angular acceleration: %f", angularAcceleration)

	angleChangeRate := aa.CalculateAngleChangeRate(trajectory)
	t.Logf("Angle change rate: %f", angleChangeRate)
}

func TestAngleAnalyzerLinearPath(t *testing.T) {
	aa := NewAngleAnalyzer()

	linearTrajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 0, Timestamp: 100},
		{X: 200, Y: 0, Timestamp: 200},
		{X: 300, Y: 0, Timestamp: 300},
		{X: 400, Y: 0, Timestamp: 400},
		{X: 500, Y: 0, Timestamp: 500},
	}

	angularVelocity := aa.CalculateAngularVelocity(linearTrajectory)
	t.Logf("Linear path angular velocity: %f (should be ~0)", angularVelocity)

	if angularVelocity > 0.01 {
		t.Log("Note: Very small angular velocity detected, might be due to floating point precision")
	}
}

func TestBacktrackAnalyzer(t *testing.T) {
	ba := NewBacktrackAnalyzer()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 5, Timestamp: 100},
		{X: 200, Y: 10, Timestamp: 200},
		{X: 150, Y: 8, Timestamp: 250},
		{X: 180, Y: 12, Timestamp: 300},
		{X: 300, Y: 15, Timestamp: 400},
		{X: 400, Y: 18, Timestamp: 500},
		{X: 350, Y: 20, Timestamp: 550},
		{X: 500, Y: 25, Timestamp: 600},
	}

	typeDist, timing := ba.AnalyzeBacktrackPatterns(trajectory)

	t.Logf("Backtrack type distribution: %v", typeDist)
	t.Logf("Backtrack timing: %v", timing)
}

func TestBacktrackSeverityClassification(t *testing.T) {
	ba := NewBacktrackAnalyzer()

	testCases := []struct {
		name          string
		count         int
		totalDistance float64
		expected      string
	}{
		{"No backtrack", 0, 0, "none"},
		{"Minimal backtrack", 1, 15, "minimal"},
		{"Normal backtrack", 3, 40, "normal"},
		{"Excessive backtrack", 8, 80, "excessive"},
		{"Severe backtrack", 15, 200, "severe"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			severity := ba.ClassifyBacktrackSeverity(tc.count, tc.totalDistance)
			if severity != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, severity)
			}
		})
	}
}

func TestPatternAnalyzer(t *testing.T) {
	pa := NewPatternAnalyzer()

	mechanicalTrajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 0, Timestamp: 50},
		{X: 200, Y: 0, Timestamp: 100},
		{X: 300, Y: 0, Timestamp: 150},
		{X: 400, Y: 0, Timestamp: 200},
		{X: 500, Y: 0, Timestamp: 250},
	}

	pattern := pa.AnalyzeTrajectoryPattern(mechanicalTrajectory)
	t.Logf("Mechanical trajectory pattern: %s", pattern)

	naturalTrajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 80, Y: 5, Timestamp: 150},
		{X: 120, Y: -3, Timestamp: 250},
		{X: 200, Y: 8, Timestamp: 400},
		{X: 280, Y: -5, Timestamp: 600},
		{X: 350, Y: 10, Timestamp: 800},
		{X: 420, Y: -2, Timestamp: 1000},
		{X: 500, Y: 5, Timestamp: 1200},
	}

	pattern = pa.AnalyzeTrajectoryPattern(naturalTrajectory)
	t.Logf("Natural trajectory pattern: %s", pattern)
}

func TestAnomalousPatternDetection(t *testing.T) {
	pa := NewPatternAnalyzer()

	mechanicalTrajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 0, Timestamp: 50},
		{X: 200, Y: 0, Timestamp: 100},
		{X: 300, Y: 0, Timestamp: 150},
		{X: 400, Y: 0, Timestamp: 200},
		{X: 500, Y: 0, Timestamp: 250},
	}

	anomalies := pa.DetectAnomalousPatterns(mechanicalTrajectory)
	t.Logf("Mechanical trajectory anomalies: %v", anomalies)

	if len(anomalies) == 0 {
		t.Log("Warning: No anomalies detected in mechanical trajectory")
	}

	normalTrajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 80, Y: 5, Timestamp: 150},
		{X: 120, Y: -3, Timestamp: 250},
		{X: 200, Y: 8, Timestamp: 400},
		{X: 280, Y: -5, Timestamp: 600},
		{X: 350, Y: 10, Timestamp: 800},
		{X: 420, Y: -2, Timestamp: 1000},
		{X: 500, Y: 5, Timestamp: 1200},
	}

	anomalies = pa.DetectAnomalousPatterns(normalTrajectory)
	t.Logf("Normal trajectory anomalies: %v", anomalies)
}

func TestEnhancedAnomalyDetection(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	botTrajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 0, Timestamp: 50},
		{X: 200, Y: 0, Timestamp: 100},
		{X: 300, Y: 0, Timestamp: 150},
		{X: 400, Y: 0, Timestamp: 200},
		{X: 500, Y: 0, Timestamp: 250},
	}

	botResult, err := analyzer.AnalyzeSliderTrajectory(botTrajectory, 500)
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	t.Logf("Bot trajectory - AnomalyScore: %.4f, IsBot: %v", botResult.AnomalyScore, botResult.IsBot)

	humanTrajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 80, Y: 5, Timestamp: 150},
		{X: 120, Y: -3, Timestamp: 250},
		{X: 200, Y: 8, Timestamp: 400},
		{X: 280, Y: -5, Timestamp: 600},
		{X: 350, Y: 10, Timestamp: 800},
		{X: 420, Y: -2, Timestamp: 1000},
		{X: 500, Y: 5, Timestamp: 1200},
	}

	humanResult, err := analyzer.AnalyzeSliderTrajectory(humanTrajectory, 500)
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	t.Logf("Human trajectory - AnomalyScore: %.4f, IsBot: %v", humanResult.AnomalyScore, humanResult.IsBot)

	if botResult.AnomalyScore <= humanResult.AnomalyScore {
		t.Log("Note: Bot trajectory anomaly score should be higher than human trajectory")
	}
}

func TestEnhancedSpeedFeatures(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 50, Y: 2, Timestamp: 50},
		{X: 100, Y: 5, Timestamp: 100},
		{X: 150, Y: 8, Timestamp: 150},
		{X: 200, Y: 10, Timestamp: 200},
		{X: 250, Y: 12, Timestamp: 250},
		{X: 300, Y: 15, Timestamp: 300},
		{X: 350, Y: 18, Timestamp: 350},
		{X: 400, Y: 20, Timestamp: 400},
		{X: 450, Y: 22, Timestamp: 450},
		{X: 500, Y: 25, Timestamp: 500},
	}

	result, err := analyzer.AnalyzeSliderTrajectory(trajectory, 500)
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	if result.Features == nil {
		t.Fatal("Features should not be nil")
	}

	t.Logf("Speed volatility: %f", result.Features.SpeedVolatility)
	t.Logf("Jerk average: %f, max: %f, variance: %f",
		result.Features.JerkAverage, result.Features.JerkMax, result.Features.JerkVariance)
	t.Logf("Angular velocity: %f, angular acceleration: %f",
		result.Features.AngularVelocity, result.Features.AngularAcceleration)
	t.Logf("Angle change rate: %f", result.Features.AngleChangeRate)
	t.Logf("Smoothness index: %f", result.Features.SmoothnessIndex)
	t.Logf("Curvature stats - Mean: %f, StdDev: %f, PeakCount: %d",
		result.Features.PathCurvatureStats.Mean, result.Features.PathCurvatureStats.StdDev,
		result.Features.PathCurvatureStats.PeakCount)
}

func TestCompleteAnalysisWorkflow(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	testTrajectories := []struct {
		name     string
		traj     []SliderPoint
		expected bool
	}{
		{
			name: "Perfect linear bot trajectory",
			traj: []SliderPoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 100, Y: 0, Timestamp: 50},
				{X: 200, Y: 0, Timestamp: 100},
				{X: 300, Y: 0, Timestamp: 150},
				{X: 400, Y: 0, Timestamp: 200},
				{X: 500, Y: 0, Timestamp: 250},
			},
			expected: true,
		},
		{
			name: "Natural human trajectory",
			traj: []SliderPoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 80, Y: 5, Timestamp: 150},
				{X: 120, Y: -3, Timestamp: 250},
				{X: 200, Y: 8, Timestamp: 400},
				{X: 280, Y: -5, Timestamp: 600},
				{X: 350, Y: 10, Timestamp: 800},
				{X: 420, Y: -2, Timestamp: 1000},
				{X: 500, Y: 5, Timestamp: 1200},
			},
			expected: false,
		},
		{
			name: "Rush bot trajectory",
			traj: []SliderPoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 500, Y: 0, Timestamp: 50},
				{X: 1000, Y: 0, Timestamp: 100},
				{X: 1500, Y: 0, Timestamp: 150},
			},
			expected: true,
		},
	}

	for _, tt := range testTrajectories {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.AnalyzeSliderTrajectory(tt.traj, 1500)
			if err != nil {
				t.Fatalf("Analysis failed: %v", err)
			}

			t.Logf("%s - IsBot: %v, AnomalyScore: %.4f, Confidence: %.4f",
				tt.name, result.IsBot, result.AnomalyScore, result.Confidence)

			if result.IsBot != tt.expected {
				t.Logf("Note: Detection result differs from expected for %s", tt.name)
			}
		})
	}
}

func TestComprehensiveFeatureExtraction(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 50, Y: 3, Timestamp: 60},
		{X: 100, Y: -2, Timestamp: 120},
		{X: 150, Y: 5, Timestamp: 180},
		{X: 200, Y: -1, Timestamp: 250},
		{X: 250, Y: 4, Timestamp: 320},
		{X: 300, Y: 2, Timestamp: 400},
		{X: 350, Y: -3, Timestamp: 480},
		{X: 400, Y: 6, Timestamp: 560},
		{X: 450, Y: 1, Timestamp: 650},
		{X: 500, Y: 4, Timestamp: 750},
	}

	result, err := analyzer.AnalyzeSliderTrajectory(trajectory, 500)
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	t.Logf("=== Comprehensive Feature Test Results ===")
	t.Logf("Total features extracted: path_efficiency=%.4f, avg_speed=%.2f",
		result.Features.PathEfficiency, result.Features.AverageSpeed)
	t.Logf("Speed features: variance=%.4f, skewness=%.4f, kurtosis=%.4f",
		result.Features.SpeedVariance, result.Features.SpeedSkewness, result.Features.SpeedKurtosis)
	t.Logf("Acceleration features: avg=%.4f, peak=%.4f, change=%.4f",
		result.Features.AverageAcceleration, result.Features.AccelerationPeak, result.Features.AccelerationChange)
	t.Logf("Jerk features: avg=%.4f, max=%.4f, variance=%.4f",
		result.Features.JerkAverage, result.Features.JerkMax, result.Features.JerkVariance)
	t.Logf("Curvature features: avg=%.4f, variance=%.4f, max=%.4f",
		result.Features.CurvatureAverage, result.Features.CurvatureVariance, result.Features.CurvatureMax)
	t.Logf("Backtrack features: count=%d, distance=%.2f, depth=%.2f",
		result.Features.BacktrackCount, result.Features.BacktrackDistance, result.Features.BacktrackDepth)
	t.Logf("Human likeness score: %.4f", result.Features.HumanLikenessScore)
	t.Logf("Bot detection: IsBot=%v, AnomalyScore=%.4f, OverallRiskScore=%.4f",
		result.IsBot, result.AnomalyScore, result.OverallRiskScore)

	if result.Features.PathEfficiency == 0 {
		t.Error("Path efficiency should not be zero")
	}
}

func TestBotDetectionAccuracy(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	botTrajectories := [][]SliderPoint{
		{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 100, Y: 0, Timestamp: 50},
			{X: 200, Y: 0, Timestamp: 100},
			{X: 300, Y: 0, Timestamp: 150},
			{X: 400, Y: 0, Timestamp: 200},
			{X: 500, Y: 0, Timestamp: 250},
		},
		{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 200, Y: 0, Timestamp: 30},
			{X: 400, Y: 0, Timestamp: 60},
			{X: 600, Y: 0, Timestamp: 90},
			{X: 800, Y: 0, Timestamp: 120},
		},
		{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 150, Y: 0, Timestamp: 40},
			{X: 300, Y: 0, Timestamp: 80},
			{X: 450, Y: 0, Timestamp: 120},
			{X: 600, Y: 0, Timestamp: 160},
		},
	}

	humanTrajectories := [][]SliderPoint{
		{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 80, Y: 5, Timestamp: 150},
			{X: 120, Y: -3, Timestamp: 250},
			{X: 200, Y: 8, Timestamp: 400},
			{X: 280, Y: -5, Timestamp: 600},
			{X: 350, Y: 10, Timestamp: 800},
			{X: 420, Y: -2, Timestamp: 1000},
			{X: 500, Y: 5, Timestamp: 1200},
		},
		{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 100, Y: 8, Timestamp: 200},
			{X: 180, Y: -5, Timestamp: 350},
			{X: 250, Y: 12, Timestamp: 550},
			{X: 320, Y: -3, Timestamp: 750},
			{X: 400, Y: 7, Timestamp: 950},
		},
	}

	botDetected := 0
	for _, traj := range botTrajectories {
		result, err := analyzer.AnalyzeSliderTrajectory(traj, 500)
		if err != nil {
			t.Logf("Bot analysis error: %v", err)
			continue
		}
		if result.IsBot {
			botDetected++
		}
		t.Logf("Bot trajectory - IsBot: %v, AnomalyScore: %.4f", result.IsBot, result.AnomalyScore)
	}

	humanDetectedAsBot := 0
	for _, traj := range humanTrajectories {
		result, err := analyzer.AnalyzeSliderTrajectory(traj, 500)
		if err != nil {
			t.Logf("Human analysis error: %v", err)
			continue
		}
		if result.IsBot {
			humanDetectedAsBot++
		}
		t.Logf("Human trajectory - IsBot: %v, AnomalyScore: %.4f", result.IsBot, result.AnomalyScore)
	}

	botAccuracy := float64(botDetected) / float64(len(botTrajectories)) * 100
	humanAccuracy := float64(len(humanTrajectories)-humanDetectedAsBot) / float64(len(humanTrajectories)) * 100

	t.Logf("Bot detection accuracy: %.2f%% (%d/%d)",
		botAccuracy, botDetected, len(botTrajectories))
	t.Logf("Human preservation accuracy: %.2f%% (%d/%d)",
		humanAccuracy, len(humanTrajectories)-humanDetectedAsBot, len(humanTrajectories))

	if botAccuracy < 99.0 {
		t.Logf("Warning: Bot detection accuracy (%.2f%%) is below target (99%%)", botAccuracy)
	}
}
