package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestLSTMFeatureExtractor_ExtractFeatures(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	points := make([]model.TracePoint, 10)
	for i := 0; i < 10; i++ {
		points[i] = model.TracePoint{
			Timestamp: int64(i * 100),
			X:         float64(i * 10),
			Y:         float64(i * 10),
			Event:     "move",
		}
	}
	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 900,
	}

	features, err := extractor.ExtractFeatures(context.Background(), traceData)
	if err != nil {
		t.Fatalf("ExtractFeatures failed: %v", err)
	}

	if len(features) != AIFeatureVectorSize {
		t.Errorf("Expected feature vector size %d, got %d", AIFeatureVectorSize, len(features))
	}

	for i, f := range features {
		if math.IsNaN(f) || math.IsInf(f, 0) {
			t.Errorf("Feature %d has invalid value: %v", i, f)
		}
	}
}

func TestTransformerPredictor_Predict(t *testing.T) {
	predictor := NewTransformerPredictor()
	err := predictor.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	features := make([]float64, AIFeatureVectorSize)
	for i := range features {
		features[i] = float64(i) / float64(AIFeatureVectorSize)
	}

	score, err := predictor.Predict(context.Background(), features)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if score < 0 || score > 1 {
		t.Errorf("Expected score in [0,1], got %v", score)
	}
}

func TestOptimizedDTW_ComputeDistance(t *testing.T) {
	dtw := NewOptimizedDTW()

	path1 := []model.TracePoint{
		{X: 0, Y: 0},
		{X: 1, Y: 1},
		{X: 2, Y: 2},
	}
	path2 := []model.TracePoint{
		{X: 0, Y: 0},
		{X: 1, Y: 1},
		{X: 2, Y: 2},
	}

	distance := dtw.ComputeDistance(path1, path2)
	if distance > 0.01 {
		t.Errorf("Expected distance close to 0 for identical paths, got %v", distance)
	}

	path3 := []model.TracePoint{
		{X: 0, Y: 0},
		{X: 100, Y: 100},
		{X: 200, Y: 200},
	}
	distance = dtw.ComputeDistance(path1, path3)
	if distance < 100 {
		t.Errorf("Expected large distance for different paths, got %v", distance)
	}
}

func TestAIAnomalyDetector_Detect(t *testing.T) {
	detector := NewAIAnomalyDetector()

	points := make([]model.TracePoint, 20)
	for i := 0; i < 20; i++ {
		points[i] = model.TracePoint{
			Timestamp: int64(i * 50),
			X:         float64(i * 5),
			Y:         float64(i * 5),
			Event:     "move",
		}
	}
	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 950,
	}

	patterns := detector.Detect(context.Background(), traceData)
	if len(patterns) == 0 {
		t.Error("Expected to detect anomaly patterns, got none")
	}
}

func TestRealTimeBehaviorAnalysisService_PredictRiskFromData(t *testing.T) {
	service := NewRealTimeBehaviorAnalysisService()
	err := service.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	points := make([]model.TracePoint, 50)
	for i := 0; i < 50; i++ {
		points[i] = model.TracePoint{
			Timestamp: int64(i * 100),
			X:         float64(i*10) + float64(i%3),
			Y:         float64(i*10) + float64(i%5),
			Event:     "move",
		}
	}
	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 4900,
	}

	result, err := service.PredictRiskFromData(context.Background(), traceData)
	if err != nil {
		t.Fatalf("PredictRiskFromData failed: %v", err)
	}

	if result.ProcessingTime > 50*time.Millisecond {
		t.Errorf("Expected processing time < 50ms, got %v", result.ProcessingTime)
	}

	if result.CombinedScore < 0 || result.CombinedScore > 1 {
		t.Errorf("Expected CombinedScore in [0,1], got %v", result.CombinedScore)
	}
}

func TestRealTimeBehaviorAnalysisService_Loaded(t *testing.T) {
	service := NewRealTimeBehaviorAnalysisService()

	if service.IsLoaded() {
		t.Error("Expected service to be unloaded initially")
	}

	err := service.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !service.IsLoaded() {
		t.Error("Expected service to be loaded after initialization")
	}
}
