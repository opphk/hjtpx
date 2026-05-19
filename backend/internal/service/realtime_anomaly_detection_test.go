package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestNewRealTimeAnomalyDetector(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()
	if detector == nil {
		t.Fatal("NewRealTimeAnomalyDetector should not return nil")
	}
	if detector.thresholds == nil {
		t.Error("thresholds map should be initialized")
	}
	if detector.patterns == nil {
		t.Error("patterns slice should be initialized")
	}
	if detector.baselineStats == nil {
		t.Error("baselineStats should be initialized")
	}
}

func TestRealTimeAnomalyDetectorInitialize(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()
	ctx := context.Background()

	err := detector.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}

	if !detector.initialized {
		t.Error("detector should be initialized")
	}
}

func TestRealTimeAnomalyDetectorDetectNilTrace(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()
	ctx := context.Background()
	detector.Initialize(ctx)

	result, err := detector.Detect(ctx, nil, "user1", "session1")
	if err != nil {
		t.Errorf("Detect failed: %v", err)
	}

	if result.IsAnomaly {
		t.Error("Nil trace should not be anomaly")
	}
}

func TestRealTimeAnomalyDetectorDetectEmptyTrace(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()
	ctx := context.Background()
	detector.Initialize(ctx)

	trace := &model.TraceData{
		Points:    []model.TracePoint{},
		TotalTime: 0,
	}

	result, err := detector.Detect(ctx, trace, "user1", "session1")
	if err != nil {
		t.Errorf("Detect failed: %v", err)
	}

	if result.IsAnomaly {
		t.Error("Empty trace should not be anomaly")
	}
}

func TestRealTimeAnomalyDetectorDetectHighSpeed(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()
	ctx := context.Background()
	detector.Initialize(ctx)

	points := make([]model.TracePoint, 10)
	for i := range points {
		points[i] = model.TracePoint{
			Timestamp: int64(i * 10),
			X:         float64(i) * 100,
			Y:         float64(i) * 50,
		}
	}

	trace := &model.TraceData{
		Points:    points,
		TotalTime: 90,
	}

	result, err := detector.Detect(ctx, trace, "user1", "session1")
	if err != nil {
		t.Errorf("Detect failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.AnomalyScore == nil {
		t.Fatal("AnomalyScore should not be nil")
	}
}

func TestRealTimeAnomalyDetectorDetectStraightLine(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()
	ctx := context.Background()
	detector.Initialize(ctx)

	points := make([]model.TracePoint, 10)
	for i := range points {
		points[i] = model.TracePoint{
			Timestamp: int64(i * 100),
			X:         float64(i),
			Y:         float64(i),
		}
	}

	trace := &model.TraceData{
		Points:    points,
		TotalTime: 900,
	}

	result, err := detector.Detect(ctx, trace, "user1", "session1")
	if err != nil {
		t.Errorf("Detect failed: %v", err)
	}

	foundStraightLine := false
	for _, t := range result.AnomalyType {
		if t == AnomalyTypeStraightLine {
			foundStraightLine = true
			break
		}
	}
	if foundStraightLine {
		t.Log("Detected straight line anomaly (expected for linear path)")
	}
}

func TestRealTimeAnomalyDetectorCalculateSpeeds(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()

	points := []model.TracePoint{
		{Timestamp: 0, X: 0, Y: 0},
		{Timestamp: 100, X: 100, Y: 0},
		{Timestamp: 200, X: 200, Y: 0},
	}

	speeds := detector.calculateSpeeds(points)

	if len(speeds) != 2 {
		t.Errorf("Expected 2 speeds, got %d", len(speeds))
	}

	if speeds[0] != 1.0 {
		t.Errorf("Expected first speed 1.0, got %f", speeds[0])
	}
}

func TestRealTimeAnomalyDetectorCalculateTimings(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()

	points := []model.TracePoint{
		{Timestamp: 0, X: 0, Y: 0},
		{Timestamp: 100, X: 10, Y: 0},
		{Timestamp: 200, X: 20, Y: 0},
	}

	timings := detector.calculateTimings(points)

	if len(timings) != 2 {
		t.Errorf("Expected 2 timings, got %d", len(timings))
	}

	if timings[0] != 100.0 {
		t.Errorf("Expected first timing 100.0, got %f", timings[0])
	}
}

func TestRealTimeAnomalyDetectorCalculateDirections(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()

	points := []model.TracePoint{
		{Timestamp: 0, X: 0, Y: 0},
		{Timestamp: 100, X: 10, Y: 0},
		{Timestamp: 200, X: 10, Y: 10},
	}

	directions := detector.calculateDirections(points)

	if len(directions) != 2 {
		t.Errorf("Expected 2 directions, got %d", len(directions))
	}
}

func TestRealTimeAnomalyDetectorCalculateCurvatures(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()

	points := []model.TracePoint{
		{Timestamp: 0, X: 0, Y: 0},
		{Timestamp: 100, X: 10, Y: 5},
		{Timestamp: 200, X: 20, Y: 0},
	}

	curvatures := detector.calculateCurvatures(points)

	if len(curvatures) != 1 {
		t.Errorf("Expected 1 curvature, got %d", len(curvatures))
	}
}

func TestRealTimeAnomalyDetectorCalculateSpeedAnomalyScore(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()

	speeds := []float64{1.0, 1.5, 2.0, 15.0, 1.0}
	score := detector.calculateSpeedAnomalyScore(speeds)

	if score < 0 {
		t.Errorf("Score should be non-negative, got %f", score)
	}
	if score > 1 {
		t.Errorf("Score should be <= 1, got %f", score)
	}
}

func TestRealTimeAnomalyDetectorCalculatePatternAnomalyScore(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()

	points := []model.TracePoint{
		{Timestamp: 0, X: 0, Y: 0},
		{Timestamp: 100, X: 10, Y: 0},
		{Timestamp: 200, X: 20, Y: 0},
		{Timestamp: 300, X: 30, Y: 0},
		{Timestamp: 400, X: 40, Y: 0},
	}

	score := detector.calculatePatternAnomalyScore(points)

	t.Logf("Pattern anomaly score: %f", score)
}

func TestRealTimeAnomalyDetectorDetermineRiskLevel(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()

	testCases := []struct {
		score    float64
		expected string
	}{
		{0.9, "critical"},
		{0.7, "high"},
		{0.5, "medium"},
		{0.3, "low"},
		{0.1, "none"},
	}

	for _, tc := range testCases {
		level := detector.determineRiskLevel(tc.score)
		if level != tc.expected {
			t.Errorf("Score %f: expected %s, got %s", tc.score, tc.expected, level)
		}
	}
}

func TestRealTimeAnomalyDetectorGetEffectiveThreshold(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()

	types := []AnomalyType{AnomalyTypeHighSpeed, AnomalyTypeRapidClicks}
	threshold := detector.getEffectiveThreshold(types)

	if threshold <= 0 {
		t.Error("Threshold should be positive")
	}
}

func TestRealTimeAnomalyDetectorGetEffectiveThresholdEmpty(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()

	threshold := detector.getEffectiveThreshold([]AnomalyType{})
	if threshold != 0.5 {
		t.Errorf("Expected default threshold 0.5, got %f", threshold)
	}
}

func TestCalculateMean(t *testing.T) {
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	mean := calculateMean(values)

	if mean != 3.0 {
		t.Errorf("Expected mean 3.0, got %f", mean)
	}
}

func TestCalculateMeanEmpty(t *testing.T) {
	mean := calculateMean([]float64{})
	if mean != 0.0 {
		t.Errorf("Expected mean 0.0 for empty slice, got %f", mean)
	}
}

func TestCalculateVariance(t *testing.T) {
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	variance := calculateVariance(values)

	if variance < 1.9 || variance > 2.1 {
		t.Errorf("Expected variance ~2.0, got %f", variance)
	}
}

func TestCalculateVarianceSingleValue(t *testing.T) {
	variance := calculateVariance([]float64{5.0})
	if variance != 0.0 {
		t.Errorf("Expected variance 0.0 for single value, got %f", variance)
	}
}

func TestCalculateMax(t *testing.T) {
	values := []float64{1.0, 5.0, 3.0, 2.0, 4.0}
	max := calculateMax(values)

	if max != 5.0 {
		t.Errorf("Expected max 5.0, got %f", max)
	}
}

func TestCalculateMaxEmpty(t *testing.T) {
	max := calculateMax([]float64{})
	if max != 0.0 {
		t.Errorf("Expected max 0.0 for empty slice, got %f", max)
	}
}

func TestComputeCurvature(t *testing.T) {
	p1 := model.TracePoint{X: 0, Y: 0}
	p2 := model.TracePoint{X: 1, Y: 1}
	p3 := model.TracePoint{X: 2, Y: 0}

	curvature := computeCurvature(p1, p2, p3)

	if curvature < 0 || curvature > math.Pi {
		t.Errorf("Curvature should be between 0 and Pi, got %f", curvature)
	}
}

func TestComputeCurvatureStraightLine(t *testing.T) {
	p1 := model.TracePoint{X: 0, Y: 0}
	p2 := model.TracePoint{X: 1, Y: 0}
	p3 := model.TracePoint{X: 2, Y: 0}

	curvature := computeCurvature(p1, p2, p3)

	if curvature > 0.001 {
		t.Errorf("Expected curvature ~0 for straight line, got %f", curvature)
	}
}

func TestRealTimeAnomalyDetectorExtractFeatures(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()

	trace := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 0, Y: 0},
			{Timestamp: 100, X: 10, Y: 0},
			{Timestamp: 200, X: 20, Y: 0},
		},
		TotalTime: 200,
	}

	features := detector.extractFeatures(trace)

	if features["point_count"] != 3.0 {
		t.Errorf("Expected point_count 3, got %f", features["point_count"])
	}
	if features["total_time"] != 200.0 {
		t.Errorf("Expected total_time 200, got %f", features["total_time"])
	}
}

func TestRealTimeAnomalyDetectorUpdateBaseline(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()

	features := map[string]float64{
		"mean_speed":      1.5,
		"speed_variance":  0.3,
		"mean_timing":     100.0,
	}

	detector.UpdateBaseline(features)

	if detector.baselineStats.SampleCount != 1 {
		t.Errorf("Expected sample count 1, got %d", detector.baselineStats.SampleCount)
	}
}

func TestRealTimeAnomalyDetectorRecordFeedback(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()

	detector.recentAnomalies = []*AnomalyRecord{
		{TraceID: "trace1", Score: 0.8},
		{TraceID: "trace2", Score: 0.6},
	}

	detector.RecordFeedback("trace1", false)

	if !detector.recentAnomalies[0].FalsePositive {
		t.Error("trace1 should be marked as false positive")
	}
}

func TestRealTimeAnomalyDetectorGetThresholds(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()

	thresholds := detector.GetThresholds()

	if len(thresholds) == 0 {
		t.Error("Should have some thresholds")
	}

	if _, ok := thresholds[AnomalyTypeHighSpeed]; !ok {
		t.Error("Should have HighSpeed threshold")
	}
}

func TestRealTimeAnomalyDetectorGetBaselineStats(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()
	detector.baselineStats.SampleCount = 10

	stats := detector.GetBaselineStats()

	if stats == nil {
		t.Fatal("Stats should not be nil")
	}
	if stats.SampleCount != 10 {
		t.Errorf("Expected sample count 10, got %d", stats.SampleCount)
	}
}

func TestRealTimeAnomalyDetectorGetRecentAnomalies(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()

	for i := 0; i < 15; i++ {
		detector.recentAnomalies = append(detector.recentAnomalies, &AnomalyRecord{
			TraceID: "trace" + string(rune('0'+i)),
			Score:   float64(i) * 0.1,
		})
	}

	anomalies := detector.GetRecentAnomalies(5)
	if len(anomalies) != 5 {
		t.Errorf("Expected 5 anomalies, got %d", len(anomalies))
	}
}

func TestRealTimeAnomalyDetectorGetRecentAnomaliesNoLimit(t *testing.T) {
	detector := NewRealTimeAnomalyDetector()

	for i := 0; i < 5; i++ {
		detector.recentAnomalies = append(detector.recentAnomalies, &AnomalyRecord{
			TraceID: "trace" + string(rune('0'+i)),
		})
	}

	anomalies := detector.GetRecentAnomalies(0)
	if len(anomalies) != 5 {
		t.Errorf("Expected all 5 anomalies, got %d", len(anomalies))
	}
}

func TestNewRealTimeAnomalyService(t *testing.T) {
	service := NewRealTimeAnomalyService()
	if service == nil {
		t.Fatal("NewRealTimeAnomalyService should not return nil")
	}
	if service.detector == nil {
		t.Error("Service should have detector")
	}
	if service.alertConfig == nil {
		t.Error("Service should have alertConfig")
	}
}

func TestRealTimeAnomalyServiceInitialize(t *testing.T) {
	service := NewRealTimeAnomalyService()
	ctx := context.Background()

	err := service.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}
}

func TestRealTimeAnomalyServiceDetectAnomaly(t *testing.T) {
	service := NewRealTimeAnomalyService()
	ctx := context.Background()
	service.Initialize(ctx)

	points := make([]model.TracePoint, 10)
	for i := range points {
		points[i] = model.TracePoint{
			Timestamp: int64(i * 100),
			X:         float64(i) * 10,
			Y:         float64(i) * 5,
		}
	}

	trace := &model.TraceData{
		Points:    points,
		TotalTime: 900,
	}

	result, err := service.DetectAnomaly(ctx, trace, "user1", "session1")
	if err != nil {
		t.Errorf("DetectAnomaly failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}
}

func TestRealTimeAnomalyServiceGetStatistics(t *testing.T) {
	service := NewRealTimeAnomalyService()

	stats := service.GetStatistics()
	if stats == nil {
		t.Fatal("Statistics should not be nil")
	}

	if _, ok := stats["total_alerts"]; !ok {
		t.Error("Should have total_alerts stat")
	}
	if _, ok := stats["recent_anomalies"]; !ok {
		t.Error("Should have recent_anomalies stat")
	}
}

func TestRealTimeAnomalyServiceEnableAlerts(t *testing.T) {
	service := NewRealTimeAnomalyService()

	service.EnableAlerts(false)
	if service.alertConfig.Enabled {
		t.Error("Alerts should be disabled")
	}

	service.EnableAlerts(true)
	if !service.alertConfig.Enabled {
		t.Error("Alerts should be enabled")
	}
}

func TestRealTimeAnomalyServiceSetAlertThreshold(t *testing.T) {
	service := NewRealTimeAnomalyService()

	service.SetAlertThreshold(0.8)
	if service.alertConfig.MinSeverity != 0.8 {
		t.Errorf("Expected min severity 0.8, got %f", service.alertConfig.MinSeverity)
	}
}

func TestRealTimeAnomalyServiceSetCooldownPeriod(t *testing.T) {
	service := NewRealTimeAnomalyService()

	period := 5 * time.Minute
	service.SetCooldownPeriod(period)
	if service.alertConfig.CooldownPeriod != period {
		t.Errorf("Expected cooldown period %v, got %v", period, service.alertConfig.CooldownPeriod)
	}
}

func TestRealTimeAnomalyServiceRecordFeedback(t *testing.T) {
	service := NewRealTimeAnomalyService()

	service.RecordFeedback("trace1", true)
}

func TestAnomalyDetectionResultStructure(t *testing.T) {
	result := &AnomalyDetectionResult{
		IsAnomaly:         true,
		AnomalyType:       []AnomalyType{AnomalyTypeHighSpeed},
		AnomalyScore:      &AnomalyScore{OverallScore: 0.8},
		Confidence:        0.9,
		RiskLevel:         "high",
		Severity:          0.8,
		Threshold:         0.5,
		ContributingFactors: []string{"异常速度"},
		Recommendations:   []string{"增加验证"},
		DetectedAt:        time.Now(),
	}

	if !result.IsAnomaly {
		t.Error("IsAnomaly should be true")
	}
	if len(result.AnomalyType) != 1 {
		t.Error("Should have 1 anomaly type")
	}
	if result.Confidence != 0.9 {
		t.Errorf("Expected confidence 0.9, got %f", result.Confidence)
	}
}

func TestAnomalyScoreStructure(t *testing.T) {
	score := &AnomalyScore{
		OverallScore:   0.7,
		SpeedScore:     0.8,
		TimingScore:    0.6,
		PatternScore:   0.5,
		DirectionScore: 0.4,
		PressureScore:  0.3,
		SessionScore:   0.2,
	}

	if score.OverallScore != 0.7 {
		t.Errorf("Expected overall score 0.7, got %f", score.OverallScore)
	}
}

func TestAdaptiveThresholdStructure(t *testing.T) {
	threshold := &RealtimeAdaptiveThreshold{
		BaseThreshold:    0.5,
		CurrentThreshold: 0.6,
		MinThreshold:     0.3,
		MaxThreshold:     0.9,
		AdaptationRate:  0.05,
		LastUpdated:     time.Now(),
	}

	if threshold.CurrentThreshold < threshold.MinThreshold {
		t.Error("CurrentThreshold should be >= MinThreshold")
	}
	if threshold.CurrentThreshold > threshold.MaxThreshold {
		t.Error("CurrentThreshold should be <= MaxThreshold")
	}
}

func TestAnomalyRecordStructure(t *testing.T) {
	record := &AnomalyRecord{
		Type:          AnomalyTypeHighSpeed,
		Score:         0.8,
		TraceID:       "trace123",
		UserID:        "user1",
		SessionID:     "session1",
		DetectedAt:    time.Now(),
		FalsePositive: false,
	}

	if record.Type != AnomalyTypeHighSpeed {
		t.Errorf("Expected type HighSpeed, got %s", record.Type)
	}
}

func TestBaselineStatisticsStructure(t *testing.T) {
	stats := &BaselineStatistics{
		MeanSpeed:   1.5,
		StdSpeed:    0.3,
		MeanTiming:  100.0,
		StdTiming:   20.0,
		SampleCount: 100,
		LastUpdated: time.Now(),
	}

	if stats.SampleCount != 100 {
		t.Errorf("Expected sample count 100, got %d", stats.SampleCount)
	}
}

func TestAlertConfigStructure(t *testing.T) {
	config := &AlertConfig{
		Enabled:        true,
		MinSeverity:    0.7,
		CooldownPeriod: 1 * time.Minute,
		LastAlertTime:  time.Now(),
		AlertCallbacks: []func(*AnomalyAlert){},
	}

	if !config.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestAnomalyAlertStructure(t *testing.T) {
	alert := &AnomalyAlert{
		AlertID:     "alert123",
		UserID:      "user1",
		SessionID:  "session1",
		AnomalyType: AnomalyTypeHighSpeed,
		Severity:   0.9,
		Message:    "Test alert",
		Timestamp:  time.Now(),
		Metadata:   map[string]interface{}{"key": "value"},
	}

	if alert.AlertID != "alert123" {
		t.Errorf("Expected AlertID 'alert123', got '%s'", alert.AlertID)
	}
}

func BenchmarkAnomalyDetectorDetect(b *testing.B) {
	detector := NewRealTimeAnomalyDetector()
	ctx := context.Background()
	detector.Initialize(ctx)

	points := make([]model.TracePoint, 50)
	for i := range points {
		points[i] = model.TracePoint{
			Timestamp: int64(i * 100),
			X:         float64(i) * 10,
			Y:         float64(i) * 5,
		}
	}

	trace := &model.TraceData{
		Points:    points,
		TotalTime: 4900,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(ctx, trace, "user1", "session1")
	}
}

func BenchmarkCalculateSpeeds(b *testing.B) {
	detector := NewRealTimeAnomalyDetector()

	points := make([]model.TracePoint, 100)
	for i := range points {
		points[i] = model.TracePoint{
			Timestamp: int64(i * 100),
			X:         float64(i) * 10,
			Y:         float64(i) * 5,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.calculateSpeeds(points)
	}
}

func BenchmarkCalculateVariance(b *testing.B) {
	values := make([]float64, 100)
	for i := range values {
		values[i] = float64(i) * 0.1
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateVariance(values)
	}
}
