package trace

import (
	"math"
	"testing"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestExtractFeaturesWithNewFields(t *testing.T) {
	extractor := NewTraceExtractor()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 25, Event: "move"},
			{Timestamp: 1300, X: 30, Y: 40, Event: "move"},
			{Timestamp: 1400, X: 40, Y: 60, Event: "move"},
			{Timestamp: 1500, X: 50, Y: 85, Event: "move"},
			{Timestamp: 1600, X: 60, Y: 100, Event: "end"},
		},
		TotalTime: 600,
		StartX:    0,
		StartY:    0,
		EndX:      60,
		EndY:      100,
	}

	features, err := extractor.ExtractFeatures(traceData)
	if err != nil {
		t.Fatalf("ExtractFeatures failed: %v", err)
	}

	if features == nil {
		t.Fatal("Features should not be nil")
	}

	if features.AvgAcceleration <= 0 {
		t.Errorf("AvgAcceleration should be positive, got %f", features.AvgAcceleration)
	}

	if features.AccelVariance < 0 {
		t.Errorf("AccelVariance should not be negative, got %f", features.AccelVariance)
	}

	if features.AvgCurvature < 0 {
		t.Errorf("AvgCurvature should not be negative, got %f", features.AvgCurvature)
	}

	if features.MaxCurvature < 0 {
		t.Errorf("MaxCurvature should not be negative, got %f", features.MaxCurvature)
	}

	if features.JitterFrequency < 0 {
		t.Errorf("JitterFrequency should not be negative, got %f", features.JitterFrequency)
	}

	if features.JitterAmplitude < 0 {
		t.Errorf("JitterAmplitude should not be negative, got %f", features.JitterAmplitude)
	}

	if features.SpeedChangeRate < 0 {
		t.Errorf("SpeedChangeRate should not be negative, got %f", features.SpeedChangeRate)
	}

	if features.DirectionChange < 0 {
		t.Errorf("DirectionChange should not be negative, got %f", features.DirectionChange)
	}

	t.Logf("New features - AvgAccel: %f, AccelVariance: %f, AvgCurvature: %f, MaxCurvature: %f",
		features.AvgAcceleration, features.AccelVariance, features.AvgCurvature, features.MaxCurvature)
	t.Logf("JitterFrequency: %f, JitterAmplitude: %f, SpeedChangeRate: %f, DirectionChange: %f",
		features.JitterFrequency, features.JitterAmplitude, features.SpeedChangeRate, features.DirectionChange)
}

func TestDTWDistanceCalculation(t *testing.T) {
	dtw := NewDTWMatcher()

	trace1 := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 20, Event: "move"},
			{Timestamp: 1300, X: 30, Y: 30, Event: "end"},
		},
		TotalTime: 300,
	}

	trace2 := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1150, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1300, X: 20, Y: 20, Event: "end"},
		},
		TotalTime: 300,
	}

	distance := dtw.CalculateDTWDistance(trace1, trace2)
	if distance <= 0 {
		t.Errorf("DTW distance should be positive, got %f", distance)
	}

	similarity := dtw.CalculateDTWSimilarity(trace1, trace2)
	if similarity < 0 || similarity > 1 {
		t.Errorf("Similarity should be between 0 and 1, got %f", similarity)
	}

	t.Logf("DTW Distance: %f, Similarity: %f", distance, similarity)
}

func TestDTWSimilarityWithIdenticalTraces(t *testing.T) {
	dtw := NewDTWMatcher()

	trace1 := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 20, Event: "end"},
		},
		TotalTime: 200,
	}

	trace2 := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 20, Event: "end"},
		},
		TotalTime: 200,
	}

	similarity := dtw.CalculateDTWSimilarity(trace1, trace2)
	if similarity < 0.9 {
		t.Errorf("Identical traces should have high similarity, got %f", similarity)
	}
}

func TestDTWMatchTraces(t *testing.T) {
	dtw := NewDTWMatcher()

	trace1 := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 20, Event: "end"},
		},
		TotalTime: 200,
	}

	trace2 := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 5, Y: 5, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 20, Event: "end"},
		},
		TotalTime: 200,
	}

	distance, level := dtw.MatchTraces(trace1, trace2)
	if distance <= 0 {
		t.Error("Distance should be positive")
	}

	if level != "high" && level != "medium" {
		t.Errorf("Expected high or medium match level, got %s", level)
	}
}

func TestDTWFindNearestNeighbor(t *testing.T) {
	dtw := NewDTWMatcher()

	traces := []*model.TraceData{
		{
			Points: []model.TracePoint{
				{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
				{Timestamp: 1100, X: 5, Y: 5, Event: "move"},
				{Timestamp: 1200, X: 10, Y: 10, Event: "end"},
			},
			TotalTime: 200,
		},
		{
			Points: []model.TracePoint{
				{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
				{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
				{Timestamp: 1200, X: 20, Y: 20, Event: "end"},
			},
			TotalTime: 200,
		},
		{
			Points: []model.TracePoint{
				{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
				{Timestamp: 1100, X: 100, Y: 100, Event: "move"},
				{Timestamp: 1200, X: 200, Y: 200, Event: "end"},
			},
			TotalTime: 200,
		},
	}

	target := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 20, Event: "end"},
		},
		TotalTime: 200,
	}

	idx, similarity := dtw.FindNearestNeighbor(traces, target)
	if idx != 1 {
		t.Errorf("Expected nearest neighbor index 1, got %d", idx)
	}
	if similarity < 0.9 {
		t.Errorf("Expected high similarity, got %f", similarity)
	}
}

func TestAnomalyDetectionPerfectLine(t *testing.T) {
	detector := NewAnomalyDetector()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 50, Event: "start"},
			{Timestamp: 1100, X: 50, Y: 50, Event: "move"},
			{Timestamp: 1200, X: 100, Y: 50, Event: "move"},
			{Timestamp: 1300, X: 150, Y: 50, Event: "move"},
			{Timestamp: 1400, X: 200, Y: 50, Event: "end"},
		},
		TotalTime: 400,
	}

	anomalies, err := detector.DetectAnomalies(traceData)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	found := false
	for _, anomaly := range anomalies {
		if anomaly.Type == PatternPerfectLine {
			found = true
			if anomaly.Confidence < 0.5 {
				t.Errorf("Expected high confidence for perfect line, got %f", anomaly.Confidence)
			}
			break
		}
	}

	if !found {
		t.Error("Expected to detect perfect line anomaly")
	}
}

func TestAnomalyDetectionConstantSpeed(t *testing.T) {
	detector := NewAnomalyDetector()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1200, X: 100, Y: 5, Event: "move"},
			{Timestamp: 1400, X: 200, Y: 3, Event: "move"},
			{Timestamp: 1600, X: 300, Y: 5, Event: "move"},
			{Timestamp: 1800, X: 400, Y: 4, Event: "move"},
			{Timestamp: 2000, X: 500, Y: 5, Event: "move"},
			{Timestamp: 2200, X: 600, Y: 3, Event: "move"},
			{Timestamp: 2400, X: 700, Y: 5, Event: "end"},
		},
		TotalTime: 1400,
	}

	features, _ := detector.extractor.ExtractFeatures(traceData)
	t.Logf("Features: %+v", features)

	anomalies, err := detector.DetectAnomalies(traceData)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	t.Logf("Anomalies: %+v", anomalies)
}

func TestAnomalyDetectionInstantJump(t *testing.T) {
	detector := NewAnomalyDetector()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1001, X: 100, Y: 100, Event: "move"},
			{Timestamp: 1100, X: 150, Y: 150, Event: "end"},
		},
		TotalTime: 100,
	}

	anomalies, err := detector.DetectAnomalies(traceData)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	found := false
	for _, anomaly := range anomalies {
		if anomaly.Type == PatternInstantJump {
			found = true
			if anomaly.Confidence < 0.5 {
				t.Errorf("Expected high confidence for instant jump, got %f", anomaly.Confidence)
			}
			break
		}
	}

	if !found {
		t.Error("Expected to detect instant jump anomaly")
	}
}

func TestAnomalyDetectionSquareWave(t *testing.T) {
	detector := NewAnomalyDetector()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1020, X: 0, Y: 20, Event: "move"},
			{Timestamp: 1040, X: 0, Y: 40, Event: "move"},
			{Timestamp: 1060, X: 20, Y: 40, Event: "move"},
			{Timestamp: 1080, X: 40, Y: 40, Event: "move"},
			{Timestamp: 1100, X: 40, Y: 60, Event: "move"},
			{Timestamp: 1120, X: 40, Y: 80, Event: "move"},
			{Timestamp: 1140, X: 60, Y: 80, Event: "move"},
			{Timestamp: 1160, X: 80, Y: 80, Event: "move"},
			{Timestamp: 1180, X: 80, Y: 100, Event: "move"},
			{Timestamp: 1200, X: 80, Y: 120, Event: "end"},
		},
		TotalTime: 200,
	}

	anomalies, err := detector.DetectAnomalies(traceData)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	found := false
	for _, anomaly := range anomalies {
		if anomaly.Type == PatternSquareWave {
			found = true
			break
		}
	}

	if !found {
		t.Logf("Anomalies: %+v", anomalies)
		t.Error("Expected to detect square wave anomaly")
	}
}

func TestAnomalyDetectionNoPauses(t *testing.T) {
	detector := NewAnomalyDetector()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 50, Event: "start"},
			{Timestamp: 1100, X: 30, Y: 50, Event: "move"},
			{Timestamp: 1200, X: 60, Y: 50, Event: "move"},
			{Timestamp: 1300, X: 90, Y: 50, Event: "move"},
			{Timestamp: 1400, X: 120, Y: 50, Event: "move"},
			{Timestamp: 1500, X: 150, Y: 50, Event: "move"},
			{Timestamp: 1600, X: 180, Y: 50, Event: "move"},
			{Timestamp: 1700, X: 210, Y: 50, Event: "move"},
			{Timestamp: 1800, X: 240, Y: 50, Event: "move"},
			{Timestamp: 1900, X: 270, Y: 50, Event: "move"},
			{Timestamp: 2000, X: 300, Y: 50, Event: "end"},
		},
		TotalTime: 1000,
	}

	features, _ := detector.extractor.ExtractFeatures(traceData)
	
	t.Logf("Features: %+v", features)

	anomalies, err := detector.DetectAnomalies(traceData)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	t.Logf("Anomalies: %+v", anomalies)
}

func TestAnomalyDetectionHumanLikeTrace(t *testing.T) {
	detector := NewAnomalyDetector()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 50, Event: "start"},
			{Timestamp: 1100, X: 20, Y: 48, Event: "move"},
			{Timestamp: 1200, X: 45, Y: 52, Event: "move"},
			{Timestamp: 1350, X: 70, Y: 49, Event: "move"},
			{Timestamp: 1500, X: 95, Y: 51, Event: "move"},
			{Timestamp: 1650, X: 120, Y: 48, Event: "move"},
			{Timestamp: 1800, X: 145, Y: 52, Event: "move"},
			{Timestamp: 1950, X: 170, Y: 49, Event: "move"},
			{Timestamp: 2100, X: 200, Y: 50, Event: "end"},
		},
		TotalTime: 1100,
	}

	anomalies, err := detector.DetectAnomalies(traceData)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	criticalCount := 0
	highCount := 0
	for _, anomaly := range anomalies {
		if anomaly.Severity == "critical" {
			criticalCount++
		}
		if anomaly.Severity == "high" {
			highCount++
		}
	}

	t.Logf("Critical anomalies: %d, High anomalies: %d", criticalCount, highCount)
	t.Logf("Total anomalies: %d", len(anomalies))

	if criticalCount > 1 {
		t.Errorf("Human-like trace should have at most 1 critical anomaly, got %d", criticalCount)
	}

	if highCount > 2 {
		t.Errorf("Human-like trace should have few high anomalies, got %d", highCount)
	}
}

func TestAnomalySummary(t *testing.T) {
	detector := NewAnomalyDetector()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1001, X: 100, Y: 100, Event: "move"},
			{Timestamp: 1100, X: 200, Y: 200, Event: "end"},
		},
		TotalTime: 100,
	}

	anomalies, err := detector.DetectAnomalies(traceData)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	summary := detector.GetAnomalySummary(anomalies)

	if summary["total_anomalies"] != len(anomalies) {
		t.Errorf("Expected total_anomalies to be %d, got %v", len(anomalies), summary["total_anomalies"])
	}

	if len(anomalies) > 0 {
		suspicious := summary["is_suspicious"].(bool)
		if !suspicious {
			t.Error("Expected is_suspicious to be true when anomalies are detected")
		}
	}
}

func TestDTWWithEmptyTraces(t *testing.T) {
	dtw := NewDTWMatcher()

	distance := dtw.CalculateDTWDistance(nil, nil)
	if distance != math.MaxFloat64 {
		t.Errorf("Expected MaxFloat64 for nil traces, got %f", distance)
	}

	trace1 := &model.TraceData{Points: []model.TracePoint{{Timestamp: 1000, X: 0, Y: 0, Event: "start"}}}
	distance = dtw.CalculateDTWDistance(trace1, trace1)
	if distance != math.MaxFloat64 {
		t.Errorf("Expected MaxFloat64 for single point traces, got %f", distance)
	}
}

func TestFeatureBasedDistance(t *testing.T) {
	dtw := NewDTWMatcher()

	trace1 := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 20, Event: "end"},
		},
		TotalTime: 200,
	}

	trace2 := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 20, Event: "end"},
		},
		TotalTime: 200,
	}

	distance, err := dtw.CalculateDTWDistanceWithFeatures(trace1, trace2)
	if err != nil {
		t.Fatalf("CalculateDTWDistanceWithFeatures failed: %v", err)
	}

	if distance >= 1.0 {
		t.Errorf("Identical traces should have low feature distance, got %f", distance)
	}
}

func TestBatchCompare(t *testing.T) {
	dtw := NewDTWMatcher()

	traces := []*model.TraceData{
		{
			Points: []model.TracePoint{
				{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
				{Timestamp: 1100, X: 5, Y: 5, Event: "end"},
			},
			TotalTime: 100,
		},
		{
			Points: []model.TracePoint{
				{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
				{Timestamp: 1100, X: 10, Y: 10, Event: "end"},
			},
			TotalTime: 100,
		},
	}

	target := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "end"},
		},
		TotalTime: 100,
	}

	results := dtw.BatchCompare(traces, target)
	if len(results) != len(traces) {
		t.Errorf("Expected %d results, got %d", len(traces), len(results))
	}

	if results[1] <= results[0] {
		t.Error("Second trace should have higher similarity to target")
	}
}
