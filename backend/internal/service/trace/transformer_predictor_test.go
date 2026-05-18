package trace

import (
	"testing"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestTransformerPredictorInitialization(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	if predictor == nil {
		t.Fatal("TransformerPredictor should not be nil")
	}
	
	if !predictor.isInitialized {
		t.Error("TransformerPredictor should be initialized after creation")
	}
}

func TestTransformerPredictorPredict(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	seq := &TrajectorySequence{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
		},
		VelocitySeq:     []float64{1.0, 1.0},
		AccelerationSeq: []float64{0.0},
		DirectionSeq:    []float64{0.785, 0.785},
	}
	
	result, err := predictor.Predict(seq)
	if err != nil {
		t.Fatalf("Predict should not return error: %v", err)
	}
	
	if result == nil {
		t.Error("Result should not be nil")
	}
	
	if result.RiskScore < 0 || result.RiskScore > 1 {
		t.Errorf("RiskScore should be between 0 and 1, got %f", result.RiskScore)
	}
}

func TestTransformerPredictorPredictTrajectory(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
		},
	}
	
	result, err := predictor.PredictTrajectory(traceData)
	if err != nil {
		t.Fatalf("PredictTrajectory should not return error: %v", err)
	}
	
	if result == nil {
		t.Error("Result should not be nil")
	}
}

func TestTransformerPredictorClassifyBehavior(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	seq := &TrajectorySequence{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
			{X: 30, Y: 30, Timestamp: 300},
			{X: 40, Y: 40, Timestamp: 400},
		},
		VelocitySeq:     []float64{1.0, 1.0, 1.0, 1.0},
		AccelerationSeq: []float64{0.0, 0.0, 0.0},
		DirectionSeq:    []float64{0.785, 0.785, 0.785, 0.785},
	}
	
	result := predictor.ClassifyBehavior(seq)
	
	if result == nil {
		t.Error("Result should not be nil")
	}
	
	if result.PatternType == "" {
		t.Error("PatternType should not be empty")
	}
	
	if result.ComplexityScore < 0 || result.ComplexityScore > 1 {
		t.Errorf("ComplexityScore should be between 0 and 1, got %f", result.ComplexityScore)
	}
	
	if result.ConsistencyScore < 0 || result.ConsistencyScore > 1 {
		t.Errorf("ConsistencyScore should be between 0 and 1, got %f", result.ConsistencyScore)
	}
}

func TestTransformerPredictorPredictIntent(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	seq := &TrajectorySequence{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 100, Y: 100, Timestamp: 50},
			{X: 200, Y: 200, Timestamp: 100},
		},
		VelocitySeq: []float64{282.84, 282.84},
	}
	
	result := predictor.PredictIntent(seq)
	
	if result == nil {
		t.Error("Result should not be nil")
	}
	
	if result.IntentType == "" {
		t.Error("IntentType should not be empty")
	}
	
	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", result.Confidence)
	}
}

func TestTransformerPredictorDetectAnomalies(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	seq := &TrajectorySequence{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 1},
			{X: 20, Y: 20, Timestamp: 2},
		},
		VelocitySeq:     []float64{14142.0, 14142.0},
		AccelerationSeq: []float64{0.0},
	}
	
	anomalies := predictor.DetectAnomalies(seq)
	
	if anomalies == nil {
		t.Error("Anomalies should not be nil")
	}
	
	if len(anomalies) == 0 {
		t.Error("Expected anomalies to be detected for extreme velocity")
	}
}

func TestTransformerPredictorAnalyzeComprehensiveBehavior(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	seq := &TrajectorySequence{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
		},
		VelocitySeq:     []float64{1.0, 1.0},
		AccelerationSeq: []float64{0.0},
	}
	
	result := predictor.AnalyzeComprehensiveBehavior(seq)
	
	if result == nil {
		t.Error("Result should not be nil")
	}
	
	if result.RiskPrediction == nil {
		t.Error("RiskPrediction should not be nil")
	}
	
	if result.BehaviorClassification == nil {
		t.Error("BehaviorClassification should not be nil")
	}
	
	if result.IntentPrediction == nil {
		t.Error("IntentPrediction should not be nil")
	}
}

func TestTransformerPredictorGetModelArchitecture(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	info := predictor.GetModelArchitecture()
	
	if info == nil {
		t.Error("Model architecture should not be nil")
	}
	
	if info["model_type"] != "Transformer" {
		t.Errorf("Expected model_type 'Transformer', got %v", info["model_type"])
	}
	
	if info["embedding_dim"] != TransformerDim {
		t.Errorf("Expected embedding_dim %d, got %v", TransformerDim, info["embedding_dim"])
	}
}

func TestTransformerPredictorGetPredictionStats(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	stats := predictor.GetPredictionStats()
	
	if stats == nil {
		t.Error("Stats should not be nil")
	}
	
	if stats["embedding_dimension"] != float64(TransformerDim) {
		t.Errorf("Expected embedding_dimension %d, got %v", TransformerDim, stats["embedding_dimension"])
	}
}

func TestTransformerPredictorGetMemoryUsageBytes(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	usage := predictor.GetMemoryUsageBytes()
	
	if usage <= 0 {
		t.Errorf("Memory usage should be positive, got %d", usage)
	}
}

func TestTransformerPredictorEnableQuantization(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	predictor.EnableQuantization(true)
	
	if !predictor.quantizationEnabled {
		t.Error("Quantization should be enabled")
	}
	
	predictor.EnableQuantization(false)
	
	if predictor.quantizationEnabled {
		t.Error("Quantization should be disabled")
	}
}

func TestTransformerPredictorAnalyzeAttentionPatterns(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	embeddings := make([][]float64, 5)
	for i := range embeddings {
		embeddings[i] = make([]float64, TransformerDim)
		for j := range embeddings[i] {
			embeddings[i][j] = float64(i*10 + j)
		}
	}
	
	patterns, err := predictor.AnalyzeAttentionPatterns(embeddings)
	if err != nil {
		t.Fatalf("AnalyzeAttentionPatterns should not return error: %v", err)
	}
	
	if patterns == nil {
		t.Error("Patterns should not be nil")
	}
}

func TestTransformerPredictorComputeAttentionEntropy(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	attentionMaps := [][][]float64{
		{
			{0.5, 0.5},
			{0.3, 0.7},
		},
	}
	
	entropy := predictor.ComputeAttentionEntropy(attentionMaps)
	
	if entropy < 0 {
		t.Errorf("Entropy should be non-negative, got %f", entropy)
	}
}

func TestTransformerPredictorComputeSequenceComplexity(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	seq := &TrajectorySequence{
		VelocitySeq:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
		DirectionSeq:  []float64{0.0, 0.5, 1.0, 1.5, 2.0},
		AccelerationSeq: []float64{1.0, 1.0, 1.0, 1.0},
		CurvatureSeq:   []float64{0.1, 0.2, 0.3, 0.4, 0.5},
	}
	
	complexity := predictor.computeSequenceComplexity(seq)
	
	if complexity < 0 || complexity > 1 {
		t.Errorf("Complexity should be between 0 and 1, got %f", complexity)
	}
}

func TestTransformerPredictorComputeSequenceConsistency(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	seq := &TrajectorySequence{
		VelocitySeq: []float64{1.0, 1.1, 0.9, 1.0, 1.0},
	}
	
	consistency := predictor.computeSequenceConsistency(seq)
	
	if consistency < 0 || consistency > 1 {
		t.Errorf("Consistency should be between 0 and 1, got %f", consistency)
	}
}

func TestTransformerPredictorAnalyzeVelocityPattern(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	velocities := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
	
	result := predictor.analyzeVelocityPattern(velocities)
	
	if result == nil {
		t.Error("Result should not be nil")
	}
	
	if result["mean"] != 30.0 {
		t.Errorf("Expected mean 30.0, got %f", result["mean"])
	}
	
	if result["min"] != 10.0 {
		t.Errorf("Expected min 10.0, got %f", result["min"])
	}
	
	if result["max"] != 50.0 {
		t.Errorf("Expected max 50.0, got %f", result["max"])
	}
}

func TestTransformerPredictorAnalyzeDirectionPattern(t *testing.T) {
	predictor := NewTransformerPredictor()
	
	directions := []float64{0.0, 0.5, 1.0, 1.5, 2.0}
	
	result := predictor.analyzeDirectionPattern(directions)
	
	if result == nil {
		t.Error("Result should not be nil")
	}
}