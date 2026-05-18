package trace

import (
	"testing"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestEnhancedFeatureExtraction(t *testing.T) {
	extractor := NewTraceExtractor()

	trace := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 30, Timestamp: 200},
			{X: 30, Y: 60, Timestamp: 300},
			{X: 40, Y: 100, Timestamp: 400},
			{X: 50, Y: 150, Timestamp: 500},
			{X: 60, Y: 210, Timestamp: 600},
			{X: 70, Y: 280, Timestamp: 700},
			{X: 80, Y: 360, Timestamp: 800},
			{X: 90, Y: 450, Timestamp: 900},
		},
	}

	features, err := extractor.ExtractFeatures(trace)
	if err != nil {
		t.Fatalf("ExtractFeatures failed: %v", err)
	}

	if features.AvgSpeed <= 0 {
		t.Errorf("Expected AvgSpeed > 0, got %f", features.AvgSpeed)
	}

	if features.MaxSpeed <= 0 {
		t.Errorf("Expected MaxSpeed > 0, got %f", features.MaxSpeed)
	}

	if features.SpeedVariance < 0 {
		t.Errorf("Expected SpeedVariance >= 0, got %f", features.SpeedVariance)
	}

	if features.Smoothness < 0 || features.Smoothness > 1 {
		t.Errorf("Expected Smoothness in [0, 1], got %f", features.Smoothness)
	}

	if features.PathRatio <= 1 {
		t.Errorf("Expected PathRatio >= 1, got %f", features.PathRatio)
	}
}

func TestAdvancedFeatures(t *testing.T) {
	extractor := NewTraceExtractor()

	trace := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 5, Timestamp: 100},
			{X: 20, Y: 15, Timestamp: 200},
			{X: 30, Y: 30, Timestamp: 300},
			{X: 40, Y: 50, Timestamp: 400},
			{X: 50, Y: 75, Timestamp: 500},
			{X: 60, Y: 105, Timestamp: 600},
			{X: 70, Y: 140, Timestamp: 700},
			{X: 80, Y: 180, Timestamp: 800},
			{X: 90, Y: 225, Timestamp: 900},
			{X: 100, Y: 275, Timestamp: 1000},
			{X: 110, Y: 330, Timestamp: 1100},
			{X: 120, Y: 390, Timestamp: 1200},
			{X: 130, Y: 455, Timestamp: 1300},
			{X: 140, Y: 525, Timestamp: 1400},
		},
	}

	advanced, err := extractor.ExtractAdvancedFeatures(trace)
	if err != nil {
		t.Fatalf("ExtractAdvancedFeatures failed: %v", err)
	}

	if advanced.HurstExponent < 0 || advanced.HurstExponent > 1 {
		t.Errorf("Expected HurstExponent in [0, 1], got %f", advanced.HurstExponent)
	}

	if advanced.FractalDimension <= 0 {
		t.Errorf("Expected FractalDimension > 0, got %f", advanced.FractalDimension)
	}

	if advanced.SpectralEntropy < 0 || advanced.SpectralEntropy > 1 {
		t.Errorf("Expected SpectralEntropy in [0, 1], got %f", advanced.SpectralEntropy)
	}

	if advanced.PermutationEntropy < 0 || advanced.PermutationEntropy > 1 {
		t.Errorf("Expected PermutationEntropy in [0, 1], got %f", advanced.PermutationEntropy)
	}

	if advanced.ApproximateEntropy < 0 {
		t.Errorf("Expected ApproximateEntropy >= 0, got %f", advanced.ApproximateEntropy)
	}

	if advanced.SampleEntropy < 0 {
		t.Errorf("Expected SampleEntropy >= 0, got %f", advanced.SampleEntropy)
	}
}

func TestDTWDistance(t *testing.T) {
	matcher := NewDTWMatcher()

	trace1 := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
			{X: 30, Y: 30, Timestamp: 300},
			{X: 40, Y: 40, Timestamp: 400},
		},
	}

	trace2 := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
			{X: 30, Y: 30, Timestamp: 300},
			{X: 40, Y: 40, Timestamp: 400},
		},
	}

	distance := matcher.CalculateDTWDistance(trace1, trace2)
	if distance < 0 {
		t.Errorf("Expected distance >= 0, got %f", distance)
	}

	trace3 := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 100, Y: 100, Timestamp: 100},
			{X: 200, Y: 200, Timestamp: 200},
			{X: 300, Y: 300, Timestamp: 300},
			{X: 400, Y: 400, Timestamp: 400},
		},
	}

	distance2 := matcher.CalculateDTWDistance(trace1, trace3)
	if distance2 <= distance {
		t.Errorf("Expected larger distance between dissimilar traces, got %f vs %f", distance2, distance)
	}
}

func TestDTWWithConstraints(t *testing.T) {
	matcher := NewDTWMatcher()

	trace1 := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
			{X: 30, Y: 30, Timestamp: 300},
			{X: 40, Y: 40, Timestamp: 400},
		},
	}

	trace2 := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 25, Timestamp: 200},
			{X: 30, Y: 35, Timestamp: 300},
			{X: 40, Y: 45, Timestamp: 400},
		},
	}

	_ = matcher.CalculateDTWDistance(trace1, trace2)
	constrained := matcher.CalculateDTWDistanceWithSakoeChiba(trace1, trace2, 2)

	if constrained < 0 {
		t.Errorf("Expected constrained distance >= 0, got %f", constrained)
	}

	itakura := matcher.CalculateDTWDistanceWithItakura(trace1, trace2)
	if itakura < 0 {
		t.Errorf("Expected itakura distance >= 0, got %f", itakura)
	}
}

func TestFastDTW(t *testing.T) {
	matcher := NewDTWMatcher()

	trace1 := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 5, Y: 5, Timestamp: 50},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 15, Y: 15, Timestamp: 150},
			{X: 20, Y: 20, Timestamp: 200},
			{X: 25, Y: 25, Timestamp: 250},
			{X: 30, Y: 30, Timestamp: 300},
			{X: 35, Y: 35, Timestamp: 350},
			{X: 40, Y: 40, Timestamp: 400},
			{X: 45, Y: 45, Timestamp: 450},
		},
	}

	trace2 := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 5, Y: 6, Timestamp: 50},
			{X: 10, Y: 12, Timestamp: 100},
			{X: 15, Y: 18, Timestamp: 150},
			{X: 20, Y: 24, Timestamp: 200},
			{X: 25, Y: 30, Timestamp: 250},
			{X: 30, Y: 36, Timestamp: 300},
			{X: 35, Y: 42, Timestamp: 350},
			{X: 40, Y: 48, Timestamp: 400},
			{X: 45, Y: 54, Timestamp: 450},
		},
	}

	fastDistance := matcher.CalculateFastDTWDistance(trace1, trace2)
	if fastDistance < 0 {
		t.Errorf("Expected fastDTW distance >= 0, got %f", fastDistance)
	}

	classicDistance := matcher.CalculateDTWDistance(trace1, trace2)
	if fastDistance < classicDistance*0.9 || fastDistance > classicDistance*1.1 {
		t.Errorf("FastDTW distance should be close to classic DTW. Fast: %f, Classic: %f", fastDistance, classicDistance)
	}
}

func TestMultiFeatureDTW(t *testing.T) {
	matcher := NewDTWMatcher()

	trace1 := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
			{X: 30, Y: 30, Timestamp: 300},
			{X: 40, Y: 40, Timestamp: 400},
		},
	}

	trace2 := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
			{X: 30, Y: 30, Timestamp: 300},
			{X: 40, Y: 40, Timestamp: 400},
		},
	}

	trace3 := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 5, Y: 5, Timestamp: 50},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 15, Y: 15, Timestamp: 150},
			{X: 20, Y: 20, Timestamp: 200},
		},
	}

	distance1, err := matcher.CalculateMultiFeatureDTW(trace1, trace2)
	if err != nil {
		t.Fatalf("CalculateMultiFeatureDTW failed: %v", err)
	}

	distance2, err := matcher.CalculateMultiFeatureDTW(trace1, trace3)
	if err != nil {
		t.Fatalf("CalculateMultiFeatureDTW failed: %v", err)
	}

	if distance1 < 0 || distance1 > 1 {
		t.Errorf("Expected distance1 in [0, 1], got %f", distance1)
	}

	if distance2 < 0 || distance2 > 1 {
		t.Errorf("Expected distance2 in [0, 1], got %f", distance2)
	}
}

func TestIsolationForest(t *testing.T) {
	detector := NewAnomalyDetector()

	traces := make([]*model.TraceData, 20)
	for i := 0; i < 20; i++ {
		traces[i] = &model.TraceData{
			Points: []model.TracePoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: float64(i*10), Y: float64(i*10), Timestamp: int64(i * 100)},
				{X: float64(i*20), Y: float64(i*20), Timestamp: int64(i * 200)},
				{X: float64(i*30), Y: float64(i*30), Timestamp: int64(i * 300)},
			},
		}
	}

	forest, err := detector.TrainIsolationForest(traces, 10, 10)
	if err != nil {
		t.Fatalf("TrainIsolationForest failed: %v", err)
	}

	normalTrace := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 100, Y: 100, Timestamp: 100},
			{X: 200, Y: 200, Timestamp: 200},
			{X: 300, Y: 300, Timestamp: 300},
		},
	}

	score, err := detector.PredictAnomalyScore(forest, normalTrace)
	if err != nil {
		t.Fatalf("PredictAnomalyScore failed: %v", err)
	}

	if score < 0 || score > 1 {
		t.Errorf("Expected score in [0, 1], got %f", score)
	}

	anomalyTrace := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 1000, Y: 1000, Timestamp: 100},
			{X: 2000, Y: 2000, Timestamp: 200},
			{X: 3000, Y: 3000, Timestamp: 300},
		},
	}

	anomalyScore, err := detector.PredictAnomalyScore(forest, anomalyTrace)
	if err != nil {
		t.Fatalf("PredictAnomalyScore failed: %v", err)
	}

	if anomalyScore < score {
		t.Errorf("Expected anomaly score higher than normal score. Anomaly: %f, Normal: %f", anomalyScore, score)
	}
}

func TestAutoencoder(t *testing.T) {
	detector := NewAnomalyDetector()

	traces := make([]*model.TraceData, 30)
	for i := 0; i < 30; i++ {
		traces[i] = &model.TraceData{
			Points: []model.TracePoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: float64(i*5), Y: float64(i*5), Timestamp: int64(i * 50)},
				{X: float64(i*10), Y: float64(i*10), Timestamp: int64(i * 100)},
				{X: float64(i*15), Y: float64(i*15), Timestamp: int64(i * 150)},
			},
		}
	}

	ae := detector.NewAutoencoder(10, 5)
	err := detector.TrainAutoencoder(ae, traces)
	if err != nil {
		t.Fatalf("TrainAutoencoder failed: %v", err)
	}

	normalTrace := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 50, Y: 50, Timestamp: 50},
			{X: 100, Y: 100, Timestamp: 100},
			{X: 150, Y: 150, Timestamp: 150},
		},
	}

	normalError, err := detector.PredictAutoencoderAnomaly(ae, normalTrace)
	if err != nil {
		t.Fatalf("PredictAutoencoderAnomaly failed: %v", err)
	}

	if normalError < 0 {
		t.Errorf("Expected reconstruction error >= 0, got %f", normalError)
	}

	anomalyTrace := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 500, Y: 500, Timestamp: 50},
			{X: 1000, Y: 1000, Timestamp: 100},
			{X: 1500, Y: 1500, Timestamp: 150},
		},
	}

	anomalyError, err := detector.PredictAutoencoderAnomaly(ae, anomalyTrace)
	if err != nil {
		t.Fatalf("PredictAutoencoderAnomaly failed: %v", err)
	}

	if anomalyError <= normalError {
		t.Errorf("Expected anomaly error higher than normal error. Anomaly: %f, Normal: %f", anomalyError, normalError)
	}
}

func TestHybridAnomalyDetection(t *testing.T) {
	detector := NewAnomalyDetector()

	traces := make([]*model.TraceData, 20)
	for i := 0; i < 20; i++ {
		traces[i] = &model.TraceData{
			Points: []model.TracePoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: float64(i*10), Y: float64(i*10), Timestamp: int64(i * 100)},
				{X: float64(i*20), Y: float64(i*20), Timestamp: int64(i * 200)},
				{X: float64(i*30), Y: float64(i*30), Timestamp: int64(i * 300)},
			},
		}
	}

	forest, err := detector.TrainIsolationForest(traces, 10, 10)
	if err != nil {
		t.Fatalf("TrainIsolationForest failed: %v", err)
	}

	ae := detector.NewAutoencoder(10, 5)
	err = detector.TrainAutoencoder(ae, traces)
	if err != nil {
		t.Fatalf("TrainAutoencoder failed: %v", err)
	}

	normalTrace := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 100, Y: 100, Timestamp: 100},
			{X: 200, Y: 200, Timestamp: 200},
			{X: 300, Y: 300, Timestamp: 300},
		},
	}

	result, err := detector.HybridAnomalyDetection(normalTrace, forest, ae, 0.5, 0.5)
	if err != nil {
		t.Fatalf("HybridAnomalyDetection failed: %v", err)
	}

	if result.Score < 0 || result.Score > 1 {
		t.Errorf("Expected hybrid score in [0, 1], got %f", result.Score)
	}

	if result.Method != "hybrid" {
		t.Errorf("Expected method 'hybrid', got '%s'", result.Method)
	}
}

func TestDTWAlignment(t *testing.T) {
	matcher := NewDTWMatcher()

	trace1 := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
			{X: 30, Y: 30, Timestamp: 300},
		},
	}

	trace2 := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
			{X: 30, Y: 30, Timestamp: 300},
		},
	}

	alignment, err := matcher.GetDTWAlignment(trace1, trace2)
	if err != nil {
		t.Fatalf("GetDTWAlignment failed: %v", err)
	}

	if alignment.Distance < 0 {
		t.Errorf("Expected distance >= 0, got %f", alignment.Distance)
	}

	if alignment.Similarity < 0 || alignment.Similarity > 1 {
		t.Errorf("Expected similarity in [0, 1], got %f", alignment.Similarity)
	}

	if len(alignment.Path) == 0 {
		t.Error("Expected non-empty alignment path")
	}
}

func TestNearestNeighbor(t *testing.T) {
	matcher := NewDTWMatcher()

	traces := []*model.TraceData{
		{
			Points: []model.TracePoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 10, Y: 10, Timestamp: 100},
				{X: 20, Y: 20, Timestamp: 200},
			},
		},
		{
			Points: []model.TracePoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 100, Y: 100, Timestamp: 100},
				{X: 200, Y: 200, Timestamp: 200},
			},
		},
		{
			Points: []model.TracePoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 5, Y: 5, Timestamp: 100},
				{X: 10, Y: 10, Timestamp: 200},
			},
		},
	}

	target := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 12, Y: 12, Timestamp: 100},
			{X: 22, Y: 22, Timestamp: 200},
		},
	}

	idx, similarity := matcher.FindNearestNeighbor(traces, target)

	if idx != 0 {
		t.Errorf("Expected nearest neighbor to be index 0, got %d", idx)
	}

	if similarity < 0 || similarity > 1 {
		t.Errorf("Expected similarity in [0, 1], got %f", similarity)
	}
}
