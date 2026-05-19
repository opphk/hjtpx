package service

import (
	"context"
	"math"
	"testing"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestNewTrajectoryEnhancedAnalyzer(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()
	if analyzer == nil {
		t.Fatal("Expected analyzer to be created, got nil")
	}
	if analyzer.jitterThreshold != DefaultJitterThreshold {
		t.Errorf("Expected jitter threshold %v, got %v", DefaultJitterThreshold, analyzer.jitterThreshold)
	}
	if analyzer.jitterWindowSize != DefaultJitterWindowSize {
		t.Errorf("Expected jitter window size %v, got %v", DefaultJitterWindowSize, analyzer.jitterWindowSize)
	}
	if analyzer.speedFitDegree != SpeedFitDefaultDegree {
		t.Errorf("Expected speed fit degree %v, got %v", SpeedFitDefaultDegree, analyzer.speedFitDegree)
	}
}

func TestDetectJitterWithNilData(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()
	result := analyzer.DetectJitter(nil)

	if result.JitterCount != 0 {
		t.Errorf("Expected jitter count 0, got %d", result.JitterCount)
	}
	if result.JitterRatio != 0 {
		t.Errorf("Expected jitter ratio 0, got %v", result.JitterRatio)
	}
	if result.IsJittery != false {
		t.Error("Expected IsJittery to be false")
	}
	if len(result.JitterPositions) != 0 {
		t.Errorf("Expected no jitter positions, got %d", len(result.JitterPositions))
	}
}

func TestDetectJitterWithTooFewPoints(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()
	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 0, Y: 0},
			{Timestamp: 100, X: 10, Y: 10},
		},
		TotalTime: 100,
	}

	result := analyzer.DetectJitter(traceData)

	if result.JitterCount != 0 {
		t.Errorf("Expected jitter count 0, got %d", result.JitterCount)
	}
}

func TestDetectJitterWithSmoothTrajectory(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	points := []model.TracePoint{}
	for i := 0; i < 20; i++ {
		points = append(points, model.TracePoint{
			Timestamp: int64(i * 50),
			X:         float64(i) * 10,
			Y:         float64(i) * 10,
		})
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 1000,
	}

	result := analyzer.DetectJitter(traceData)

	if result.IsJittery == true {
		t.Error("Expected smooth trajectory not to be jittery")
	}
}

func TestDetectJitterWithJitteryTrajectory(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	points := []model.TracePoint{}
	for i := 0; i < 30; i++ {
		x := float64(i) * 10
		y := float64(i) * 10
		if i%3 == 0 {
			x += 1
			y += 1
		}
		if i%5 == 0 {
			x -= 1
			y -= 1
		}
		points = append(points, model.TracePoint{
			Timestamp: int64(i * 50),
			X:         x,
			Y:         y,
		})
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 1500,
	}

	result := analyzer.DetectJitter(traceData)

	t.Logf("Jitter count: %d, ratio: %v, isJittery: %v", result.JitterCount, result.JitterRatio, result.IsJittery)
}

func TestDetectJitterAmplitudeCalculation(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	points := []model.TracePoint{
		{Timestamp: 0, X: 0, Y: 0},
		{Timestamp: 10, X: 1, Y: 0},
		{Timestamp: 20, X: 100, Y: 0},
		{Timestamp: 30, X: 101, Y: 0},
		{Timestamp: 40, X: 200, Y: 0},
		{Timestamp: 50, X: 201, Y: 0},
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 50,
	}

	result := analyzer.DetectJitter(traceData)

	t.Logf("Max jitter amplitude: %v", result.MaxJitterAmplitude)
}

func TestAnalyzeCurvatureWithNilData(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()
	result := analyzer.AnalyzeCurvature(nil)

	if result.AvgCurvature != 0 {
		t.Errorf("Expected avg curvature 0, got %v", result.AvgCurvature)
	}
	if result.CurvatureScore != 1.0 {
		t.Errorf("Expected curvature score 1.0, got %v", result.CurvatureScore)
	}
}

func TestAnalyzeCurvatureWithStraightLine(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	points := []model.TracePoint{}
	for i := 0; i < 10; i++ {
		points = append(points, model.TracePoint{
			Timestamp: int64(i * 50),
			X:         float64(i) * 10,
			Y:         0,
		})
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 500,
	}

	result := analyzer.AnalyzeCurvature(traceData)

	if result.AvgCurvature > 0.1 {
		t.Errorf("Expected near-zero curvature for straight line, got %v", result.AvgCurvature)
	}
	if result.DirectionChanges > 0 {
		t.Errorf("Expected no direction changes for straight line, got %d", result.DirectionChanges)
	}
}

func TestAnalyzeCurvatureWithCurvedPath(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	points := []model.TracePoint{}
	for i := 0; i < 20; i++ {
		angle := float64(i) * 0.2
		x := 100 + 50*math.Cos(angle)
		y := 100 + 50*math.Sin(angle)
		points = append(points, model.TracePoint{
			Timestamp: int64(i * 50),
			X:         x,
			Y:         y,
		})
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 1000,
	}

	result := analyzer.AnalyzeCurvature(traceData)

	if result.AvgCurvature == 0 {
		t.Error("Expected non-zero curvature for curved path")
	}
	t.Logf("Avg curvature: %v, direction changes: %d", result.AvgCurvature, result.DirectionChanges)
}

func TestAnalyzeCurvatureWithSharpTurns(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	points := []model.TracePoint{
		{Timestamp: 0, X: 0, Y: 0},
		{Timestamp: 50, X: 100, Y: 0},
		{Timestamp: 100, X: 100, Y: 100},
		{Timestamp: 150, X: 200, Y: 100},
		{Timestamp: 200, X: 200, Y: 200},
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 200,
	}

	result := analyzer.AnalyzeCurvature(traceData)

	if result.SharpTurnCount == 0 {
		t.Error("Expected sharp turns to be detected")
	}
	if result.DirectionChanges == 0 {
		t.Error("Expected direction changes to be detected")
	}
	t.Logf("Sharp turns: %d, smooth turns: %d", result.SharpTurnCount, result.SmoothTurnCount)
}

func TestFitSpeedCurveWithNilData(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()
	result, err := analyzer.FitSpeedCurve(nil)

	if err == nil {
		t.Error("Expected error for nil data")
	}
	if result != nil && result.AccelerationPattern == "insufficient_data" {
		t.Log("Correctly returned insufficient data pattern")
	}
}

func TestFitSpeedCurveWithInsufficientPoints(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	points := []model.TracePoint{
		{Timestamp: 0, X: 0, Y: 0},
		{Timestamp: 50, X: 10, Y: 10},
		{Timestamp: 100, X: 20, Y: 20},
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 100,
	}

	result, err := analyzer.FitSpeedCurve(traceData)

	if err == nil {
		t.Error("Expected error for insufficient points")
	}
	if result.AccelerationPattern != "insufficient_data" {
		t.Errorf("Expected insufficient_data pattern, got %s", result.AccelerationPattern)
	}
}

func TestFitSpeedCurveWithNormalMovement(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	points := []model.TracePoint{}
	for i := 0; i < 20; i++ {
		speed := 1.0 + float64(i)*0.1
		x := float64(i) * 10 * speed
		y := float64(i) * 10 * speed
		points = append(points, model.TracePoint{
			Timestamp: int64(i * 50),
			X:         x,
			Y:         y,
		})
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 1000,
	}

	result, err := analyzer.FitSpeedCurve(traceData)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.Coefficients) == 0 {
		t.Error("Expected coefficients to be non-empty")
	}
	if result.Degree != SpeedFitDefaultDegree {
		t.Errorf("Expected degree %d, got %d", SpeedFitDefaultDegree, result.Degree)
	}
	if result.AccelerationPattern == "" {
		t.Error("Expected acceleration pattern to be set")
	}

	t.Logf("R2 score: %v, RMSE: %v, pattern: %s", result.R2Score, result.RMSE, result.AccelerationPattern)
}

func TestFitSpeedCurveWithConstantSpeed(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	points := []model.TracePoint{}
	for i := 0; i < 20; i++ {
		points = append(points, model.TracePoint{
			Timestamp: int64(i * 50),
			X:         float64(i) * 10,
			Y:         float64(i) * 10,
		})
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 1000,
	}

	result, err := analyzer.FitSpeedCurve(traceData)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.AccelerationPattern != "constant" {
		t.Logf("Expected constant pattern, got %s", result.AccelerationPattern)
	}
	if result.SpeedFluctuation > 0.1 {
		t.Logf("Expected low speed fluctuation for constant speed, got %v", result.SpeedFluctuation)
	}
}

func TestEvaluateSmoothnessWithNilData(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()
	result := analyzer.EvaluateSmoothness(nil)

	if result.SmoothnessScore != 1.0 {
		t.Errorf("Expected smoothness score 1.0 for nil data, got %v", result.SmoothnessScore)
	}
	if result.PathEfficiency != 1.0 {
		t.Errorf("Expected path efficiency 1.0 for nil data, got %v", result.PathEfficiency)
	}
}

func TestEvaluateSmoothnessWithSmoothTrajectory(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	points := []model.TracePoint{}
	for i := 0; i < 30; i++ {
		angle := float64(i) * 0.1
		x := float64(i) * 10 * math.Cos(angle)
		y := float64(i) * 10 * math.Sin(angle)
		points = append(points, model.TracePoint{
			Timestamp: int64(i * 50),
			X:         x,
			Y:         y,
		})
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 1500,
	}

	result := analyzer.EvaluateSmoothness(traceData)

	if result.SmoothnessScore < 0.5 {
		t.Errorf("Expected smooth trajectory to have high smoothness score, got %v", result.SmoothnessScore)
	}
	if result.SmoothRatio < 0.3 {
		t.Logf("Expected some smooth segments, got ratio %v", result.SmoothRatio)
	}
	t.Logf("Smoothness score: %v, smooth ratio: %v", result.SmoothnessScore, result.SmoothRatio)
}

func TestEvaluateSmoothnessWithRaggedTrajectory(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	points := []model.TracePoint{}
	for i := 0; i < 30; i++ {
		x := float64(i) * 10
		y := float64(i) * 10
		if i%2 == 0 {
			x += 5
			y -= 5
		} else {
			x -= 5
			y += 5
		}
		points = append(points, model.TracePoint{
			Timestamp: int64(i * 50),
			X:         x,
			Y:         y,
		})
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 1500,
	}

	result := analyzer.EvaluateSmoothness(traceData)

	if result.RaggedRatio == 0 {
		t.Error("Expected ragged trajectory to have non-zero ragged ratio")
	}
	t.Logf("Ragged ratio: %v, smoothness score: %v", result.RaggedRatio, result.SmoothnessScore)
}

func TestEvaluateSmoothnessPathEfficiency(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	directPoints := []model.TracePoint{
		{Timestamp: 0, X: 0, Y: 0},
		{Timestamp: 100, X: 100, Y: 100},
	}

	indirectPoints := []model.TracePoint{
		{Timestamp: 0, X: 0, Y: 0},
		{Timestamp: 50, X: 50, Y: 0},
		{Timestamp: 100, X: 50, Y: 50},
		{Timestamp: 150, X: 100, Y: 50},
		{Timestamp: 200, X: 100, Y: 100},
	}

	directTrace := &model.TraceData{Points: directPoints, TotalTime: 100}
	indirectTrace := &model.TraceData{Points: indirectPoints, TotalTime: 200}

	directResult := analyzer.EvaluateSmoothness(directTrace)
	indirectResult := analyzer.EvaluateSmoothness(indirectTrace)

	if directResult.PathEfficiency <= indirectResult.PathEfficiency {
		t.Error("Direct path should have higher efficiency than indirect path")
	}
	if directResult.PathEfficiency != 1.0 {
		t.Logf("Direct path efficiency: %v", directResult.PathEfficiency)
	}

	t.Logf("Direct efficiency: %v, indirect efficiency: %v", directResult.PathEfficiency, indirectResult.PathEfficiency)
}

func TestAnalyzeTrajectoryComplete(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	points := []model.TracePoint{}
	for i := 0; i < 25; i++ {
		speed := 1.0 + float64(i)*0.05
		x := float64(i) * 8 * speed
		y := float64(i) * 8 * math.Sin(float64(i)*0.2) * speed
		points = append(points, model.TracePoint{
			Timestamp: int64(i * 40),
			X:         x,
			Y:         y,
		})
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 1000,
	}

	ctx := context.Background()
	result, err := analyzer.AnalyzeTrajectory(ctx, traceData)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result.JitterResult == nil {
		t.Error("Expected jitter result to be non-nil")
	}
	if result.CurvatureResult == nil {
		t.Error("Expected curvature result to be non-nil")
	}
	if result.SpeedFitResult == nil {
		t.Error("Expected speed fit result to be non-nil")
	}
	if result.SmoothnessResult == nil {
		t.Error("Expected smoothness result to be non-nil")
	}

	if result.OverallScore < 0 || result.OverallScore > 1 {
		t.Errorf("Expected overall score between 0 and 1, got %v", result.OverallScore)
	}
	if result.ConfidenceLevel < 0 || result.ConfidenceLevel > 1 {
		t.Errorf("Expected confidence level between 0 and 1, got %v", result.ConfidenceLevel)
	}

	t.Logf("Overall score: %v, confidence: %v, anomalies: %v",
		result.OverallScore, result.ConfidenceLevel, len(result.AnomalyIndicators))
}

func TestAnalyzeTrajectoryWithNilData(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()
	ctx := context.Background()

	result, err := analyzer.AnalyzeTrajectory(ctx, nil)

	if err == nil {
		t.Error("Expected error for nil data")
	}
	if result != nil {
		t.Error("Expected nil result for nil data")
	}
}

func TestAnalyzeTrajectoryWithTooFewPoints(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()
	ctx := context.Background()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 0, Y: 0},
		},
		TotalTime: 0,
	}

	result, err := analyzer.AnalyzeTrajectory(ctx, traceData)

	if err == nil {
		t.Error("Expected error for insufficient points")
	}
	if result != nil {
		t.Error("Expected nil result for insufficient points")
	}
}

func TestCalculateConfidenceLevel(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	testCases := []struct {
		name         string
		pointCount   int
		totalTime    int64
		minExpected  float64
		maxExpected  float64
	}{
		{"few points", 5, 500, 0.1, 0.6},
		{"many points", 50, 2500, 0.6, 1.0},
		{"normal", 30, 1500, 0.3, 0.9},
	}

	for _, tc := range testCases {
		points := make([]model.TracePoint, tc.pointCount)
		for i := range points {
			points[i] = model.TracePoint{
				Timestamp: int64(i * 50),
				X:         float64(i) * 10,
				Y:         float64(i) * 10,
			}
		}

		traceData := &model.TraceData{
			Points:    points,
			TotalTime: tc.totalTime,
		}

		confidence := analyzer.calculateConfidenceLevel(traceData)

		if confidence < tc.minExpected || confidence > tc.maxExpected {
			t.Errorf("Case %s: expected confidence between %v and %v, got %v",
				tc.name, tc.minExpected, tc.maxExpected, confidence)
		}
	}
}

func TestDetectAnomalyIndicators(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	testCases := []struct {
		name            string
		jittery         bool
		jitterRatio     float64
		sharpTurns      int
		smoothTurns     int
		pathEfficiency  float64
		pattern         string
		speedFluctuation float64
		expectedCount   int
	}{
		{"normal", false, 0.05, 2, 5, 0.9, "variable", 0.1, 0},
		{"jittery", true, 0.2, 2, 5, 0.9, "variable", 0.1, 2},
		{"high jitter ratio", false, 0.2, 2, 5, 0.9, "variable", 0.1, 1},
		{"mechanical", false, 0.05, 2, 5, 0.9, "constant", 0.03, 1},
		{"inefficient", false, 0.05, 2, 5, 0.5, "variable", 0.1, 1},
	}

	for _, tc := range testCases {
		jitter := &model.JitterDetectionResult{
			IsJittery:    tc.jittery,
			JitterRatio:  tc.jitterRatio,
		}
		curvature := &model.TrajectoryCurvatureResult{
			SharpTurnCount: tc.sharpTurns,
			SmoothTurnCount: tc.smoothTurns,
		}
		speedFit := &model.SpeedCurveFitResult{
			AccelerationPattern: tc.pattern,
			SpeedFluctuation:     tc.speedFluctuation,
			R2Score:              0.5,
		}
		smoothness := &model.TrajectorySmoothnessResult{
			PathEfficiency: tc.pathEfficiency,
			RaggedRatio:    0.3,
		}

		indicators := analyzer.detectAnomalyIndicators(jitter, curvature, speedFit, smoothness)

		if len(indicators) != tc.expectedCount {
			t.Errorf("Case %s: expected %d indicators, got %d: %v",
				tc.name, tc.expectedCount, len(indicators), indicators)
		}
	}
}

func TestCalculateAngularChangesFull(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	straightPoints := []model.TracePoint{
		{X: 0, Y: 0},
		{X: 10, Y: 0},
		{X: 20, Y: 0},
		{X: 30, Y: 0},
	}

	straightChanges := analyzer.calculateAngularChangesFull(straightPoints)
	for i, change := range straightChanges {
		if change > 0.01 {
			t.Errorf("Expected near-zero angular change for straight line at index %d, got %v", i, change)
		}
	}

	curvedPoints := []model.TracePoint{}
	for i := 0; i < 10; i++ {
		angle := float64(i) * 0.3
		curvedPoints = append(curvedPoints, model.TracePoint{
			X: math.Cos(angle) * float64(i),
			Y: math.Sin(angle) * float64(i),
		})
	}

	curvedChanges := analyzer.calculateAngularChangesFull(curvedPoints)
	hasNonZero := false
	for _, change := range curvedChanges {
		if change > 0.1 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("Expected non-zero angular changes for curved path")
	}
}

func TestCalculateLinearDeviation(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	directPoints := []model.TracePoint{
		{X: 0, Y: 0},
		{X: 50, Y: 50},
		{X: 100, Y: 100},
	}
	directDeviation := analyzer.calculateLinearDeviation(directPoints)
	if directDeviation != 0 {
		t.Errorf("Expected zero deviation for direct path, got %v", directDeviation)
	}

	indirectPoints := []model.TracePoint{
		{X: 0, Y: 0},
		{X: 0, Y: 50},
		{X: 50, Y: 50},
		{X: 50, Y: 100},
		{X: 100, Y: 100},
	}
	indirectDeviation := analyzer.calculateLinearDeviation(indirectPoints)
	if indirectDeviation <= 0 {
		t.Error("Expected positive deviation for indirect path")
	}

	t.Logf("Direct deviation: %v, indirect deviation: %v", directDeviation, indirectDeviation)
}

func TestCalculateMovementContinuity(t *testing.T) {
	analyzer := NewTrajectoryEnhancedAnalyzer()

	continuousPoints := []model.TracePoint{}
	for i := 0; i < 10; i++ {
		continuousPoints = append(continuousPoints, model.TracePoint{
			Timestamp: int64(i * 50),
			X:         float64(i) * 10,
			Y:         float64(i) * 10,
		})
	}
	continuousScore := analyzer.calculateMovementContinuity(continuousPoints)
	if continuousScore < 0.9 {
		t.Errorf("Expected high continuity for regular intervals, got %v", continuousScore)
	}

	irregularPoints := []model.TracePoint{
		{Timestamp: 0, X: 0, Y: 0},
		{Timestamp: 50, X: 10, Y: 10},
		{Timestamp: 200, X: 20, Y: 20},
		{Timestamp: 250, X: 30, Y: 30},
		{Timestamp: 300, X: 40, Y: 40},
	}
	irregularScore := analyzer.calculateMovementContinuity(irregularPoints)
	if irregularScore >= continuousScore {
		t.Logf("Irregular continuity %v should be lower than continuous %v", irregularScore, continuousScore)
	}
}

func BenchmarkDetectJitter(b *testing.B) {
	analyzer := NewTrajectoryEnhancedAnalyzer()
	points := make([]model.TracePoint, 100)
	for i := range points {
		points[i] = model.TracePoint{
			Timestamp: int64(i * 50),
			X:         float64(i) * 10,
			Y:         float64(i) * 10,
		}
	}
	traceData := &model.TraceData{Points: points, TotalTime: 5000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.DetectJitter(traceData)
	}
}

func BenchmarkAnalyzeCurvature(b *testing.B) {
	analyzer := NewTrajectoryEnhancedAnalyzer()
	points := make([]model.TracePoint, 100)
	for i := range points {
		points[i] = model.TracePoint{
			Timestamp: int64(i * 50),
			X:         float64(i) * 10,
			Y:         float64(i) * 10 * math.Sin(float64(i)*0.1),
		}
	}
	traceData := &model.TraceData{Points: points, TotalTime: 5000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.AnalyzeCurvature(traceData)
	}
}

func BenchmarkFitSpeedCurve(b *testing.B) {
	analyzer := NewTrajectoryEnhancedAnalyzer()
	points := make([]model.TracePoint, 50)
	for i := range points {
		speed := 1.0 + float64(i)*0.05
		points[i] = model.TracePoint{
			Timestamp: int64(i * 50),
			X:         float64(i) * 10 * speed,
			Y:         float64(i) * 10 * speed,
		}
	}
	traceData := &model.TraceData{Points: points, TotalTime: 2500}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.FitSpeedCurve(traceData)
	}
}

func BenchmarkEvaluateSmoothness(b *testing.B) {
	analyzer := NewTrajectoryEnhancedAnalyzer()
	points := make([]model.TracePoint, 100)
	for i := range points {
		points[i] = model.TracePoint{
			Timestamp: int64(i * 50),
			X:         float64(i) * 10,
			Y:         float64(i) * 10,
		}
	}
	traceData := &model.TraceData{Points: points, TotalTime: 5000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.EvaluateSmoothness(traceData)
	}
}

func BenchmarkAnalyzeTrajectory(b *testing.B) {
	analyzer := NewTrajectoryEnhancedAnalyzer()
	points := make([]model.TracePoint, 50)
	for i := range points {
		speed := 1.0 + float64(i)*0.05
		points[i] = model.TracePoint{
			Timestamp: int64(i * 40),
			X:         float64(i) * 8 * speed,
			Y:         float64(i) * 8 * math.Sin(float64(i)*0.2) * speed,
		}
	}
	traceData := &model.TraceData{Points: points, TotalTime: 2000}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.AnalyzeTrajectory(ctx, traceData)
	}
}
