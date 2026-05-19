package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestTransformerXLConfig(t *testing.T) {
	config := &TransformerXLConfig{
		ModelDim:     256,
		NumHeads:     4,
		NumLayers:    3,
		MemoryLen:    128,
		MaxSeqLen:    512,
		FFDim:       1024,
		Dropout:     0.1,
		LearningRate: 0.0001,
	}

	txl := NewTransformerXL(config)
	if txl == nil {
		t.Fatal("TransformerXL should not be nil")
	}

	if txl.config.ModelDim != 256 {
		t.Errorf("Expected ModelDim 256, got %d", txl.config.ModelDim)
	}
	if txl.config.NumHeads != 4 {
		t.Errorf("Expected NumHeads 4, got %d", txl.config.NumHeads)
	}
	if txl.config.NumLayers != 3 {
		t.Errorf("Expected NumLayers 3, got %d", txl.config.NumLayers)
	}
}

func TestTransformerXLInitialize(t *testing.T) {
	txl := NewTransformerXL(nil)
	ctx := context.Background()

	err := txl.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}

	if !txl.initialized {
		t.Error("TransformerXL should be initialized")
	}
}

func TestTransformerXLForward(t *testing.T) {
	txl := NewTransformerXL(nil)

	tokenEmbeddings := make([]float64, TransformerXLModelDim)
	for j := range tokenEmbeddings {
		tokenEmbeddings[j] = 0.1
	}

	input := &TransformerXLInput{
		TokenEmbeddings: tokenEmbeddings,
		PositionIDs:     []int{0, 1, 2, 3, 4},
		SegmentIDs:      []int{0, 0, 0, 0, 0},
		Mask:            make([][]bool, 5),
	}

	for i := range input.Mask {
		input.Mask[i] = make([]bool, 5)
		for j := range input.Mask[i] {
			input.Mask[i][j] = true
		}
	}

	output := txl.Forward(input, nil)

	if output == nil {
		t.Fatal("Forward output should not be nil")
	}

	if len(output.Logits) != TransformerXLModelDim {
		t.Errorf("Expected logits length %d, got %d", TransformerXLModelDim, len(output.Logits))
	}

	if output.PredictiveScore < 0 || output.PredictiveScore > 1 {
		t.Errorf("PredictiveScore should be between 0 and 1, got %f", output.PredictiveScore)
	}

	if output.Confidence < 0 || output.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", output.Confidence)
	}
}

func TestTransformerXLMemory(t *testing.T) {
	txl := NewTransformerXL(nil)

	hiddenStates := make([][]float64, 3)
	for i := range hiddenStates {
		hiddenStates[i] = make([]float64, TransformerXLModelDim)
		for j := range hiddenStates[i] {
			hiddenStates[i][j] = float64(i+1) * 0.5
		}
	}

	txl.UpdateMemory(hiddenStates)

	memory := txl.GetMemory()
	if memory == nil {
		t.Fatal("GetMemory should not return nil")
	}

	if memory.Length != 3 {
		t.Errorf("Expected memory length 3, got %d", memory.Length)
	}

	if memory.Capacity != TransformerXLMemoryLen {
		t.Errorf("Expected memory capacity %d, got %d", TransformerXLMemoryLen, memory.Capacity)
	}
}

func TestTransformerXLMemoryOverflow(t *testing.T) {
	txl := NewTransformerXL(&TransformerXLConfig{
		MemoryLen: 10,
		MaxSeqLen: 20,
	})

	hiddenStates := make([][]float64, 15)
	for i := range hiddenStates {
		hiddenStates[i] = make([]float64, TransformerXLModelDim)
		for j := range hiddenStates[i] {
			hiddenStates[i][j] = float64(i) * 0.1
		}
	}

	txl.UpdateMemory(hiddenStates)

	memory := txl.GetMemory()
	if memory.Length != 10 {
		t.Errorf("Expected memory length 10 (capped), got %d", memory.Length)
	}
}

func TestTransformerXLPredictBehavior(t *testing.T) {
	txl := NewTransformerXL(nil)
	ctx := context.Background()

	sequence := &BehaviorSequence{
		UserID:     "test_user",
		SessionID:  "test_session",
		SequenceID: "test_seq_1",
		BehaviorVecs: make([][]float64, 3),
	}

	for i := range sequence.BehaviorVecs {
		sequence.BehaviorVecs[i] = make([]float64, TransformerXLModelDim)
		for j := range sequence.BehaviorVecs[i] {
			sequence.BehaviorVecs[i][j] = 0.1 * float64(i+1)
		}
	}

	output, err := txl.PredictBehavior(ctx, sequence)
	if err != nil {
		t.Errorf("PredictBehavior failed: %v", err)
	}

	if output == nil {
		t.Fatal("PredictBehavior output should not be nil")
	}

	if output.PredictiveScore < 0 || output.PredictiveScore > 1 {
		t.Errorf("PredictiveScore out of range: %f", output.PredictiveScore)
	}
}

func TestTransformerXLProcessLongSequence(t *testing.T) {
	txl := NewTransformerXL(&TransformerXLConfig{
		MaxSeqLen: 10,
	})
	ctx := context.Background()

	totalChunks := 50
	sequence := &BehaviorSequence{
		UserID:       "test_user",
		SessionID:    "test_session",
		SequenceID:   "long_seq",
		BehaviorVecs: make([][]float64, totalChunks*10),
	}

	for i := range sequence.BehaviorVecs {
		sequence.BehaviorVecs[i] = make([]float64, TransformerXLModelDim)
		for j := range sequence.BehaviorVecs[i] {
			sequence.BehaviorVecs[i][j] = float64(i % 100)
		}
	}

	outputs, err := txl.ProcessLongSequence(ctx, sequence, 10)
	if err != nil {
		t.Errorf("ProcessLongSequence failed: %v", err)
	}

	if outputs == nil {
		t.Fatal("ProcessLongSequence outputs should not be nil")
	}

	if len(outputs) != totalChunks {
		t.Errorf("Expected %d output chunks, got %d", totalChunks, len(outputs))
	}
}

func TestTransformerXLAnomalyDetection(t *testing.T) {
	txl := NewTransformerXL(nil)

	testCases := []struct {
		name     string
		output   *TransformerXLOutput
		expected bool
	}{
		{
			name: "low_score_should_be_anomaly",
			output: &TransformerXLOutput{
				PredictiveScore: 0.1,
				AttentionScores: make([][][]float64, 1),
			},
			expected: true,
		},
		{
			name: "high_score_should_not_be_anomaly",
			output: &TransformerXLOutput{
				PredictiveScore: 0.8,
				AttentionScores: make([][][]float64, 1),
			},
			expected: false,
		},
		{
			name:     "nil_output_should_not_be_anomaly",
			output:   nil,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := txl.DetectAnomaly(tc.output)
			if result != tc.expected {
				t.Errorf("Expected anomaly=%v, got %v", tc.expected, result)
			}
		})
	}
}

func TestTransformerXLService(t *testing.T) {
	service := NewTransformerXLService(nil)
	if service == nil {
		t.Fatal("TransformerXLService should not be nil")
	}

	ctx := context.Background()
	err := service.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}
}

func TestTransformerXLServiceAnalyzeBehavior(t *testing.T) {
	service := NewTransformerXLService(nil)
	ctx := context.Background()
	service.Initialize(ctx)

	traces := make([]*model.TraceData, 3)
	for i := range traces {
		points := make([]model.TracePoint, 10)
		for j := range points {
			points[j] = model.TracePoint{
				Timestamp: int64(j * 100),
				X:         float64(j * 10),
				Y:         float64(j * 5),
				Event:     "move",
			}
		}
		traces[i] = &model.TraceData{
			Points:    points,
			TotalTime: 1000,
		}
	}

	result, err := service.AnalyzeBehavior(ctx, "test_user", traces)
	if err != nil {
		t.Errorf("AnalyzeBehavior failed: %v", err)
	}

	if result == nil {
		t.Fatal("AnalyzeBehavior result should not be nil")
	}

	if result.UserID != "test_user" {
		t.Errorf("Expected UserID 'test_user', got '%s'", result.UserID)
	}

	if result.SequenceLength != 3 {
		t.Errorf("Expected SequenceLength 3, got %d", result.SequenceLength)
	}

	if result.PredictiveScore < 0 || result.PredictiveScore > 1 {
		t.Errorf("PredictiveScore out of range: %f", result.PredictiveScore)
	}
}

func TestTransformerXLServiceSequenceHistory(t *testing.T) {
	service := NewTransformerXLService(nil)
	ctx := context.Background()
	service.Initialize(ctx)

	traces := make([]*model.TraceData, 2)
	for i := range traces {
		points := make([]model.TracePoint, 5)
		for j := range points {
			points[j] = model.TracePoint{
				Timestamp: int64(j * 100),
				X:         float64(j),
				Y:         float64(j),
				Event:     "move",
			}
		}
		traces[i] = &model.TraceData{Points: points, TotalTime: 500}
	}

	service.AnalyzeBehavior(ctx, "user1", traces)
	service.AnalyzeBehavior(ctx, "user1", traces)

	history := service.GetSequenceHistory("user1")
	if len(history) != 2 {
		t.Errorf("Expected 2 sequences in history, got %d", len(history))
	}
}

func TestTransformerXLServiceStats(t *testing.T) {
	service := NewTransformerXLService(nil)

	stats := service.GetModelStats()
	if stats == nil {
		t.Fatal("GetModelStats should not return nil")
	}

	if stats["model_dim"] != TransformerXLModelDim {
		t.Errorf("Expected model_dim %d, got %v", TransformerXLModelDim, stats["model_dim"])
	}
}

func TestTransformerXLServiceSetAnomalyThreshold(t *testing.T) {
	service := NewTransformerXLService(nil)

	service.SetAnomalyThreshold(0.7)
}

func TestDotProduct(t *testing.T) {
	a := []float64{1.0, 2.0, 3.0}
	b := []float64{4.0, 5.0, 6.0}

	result := dotProduct(a, b)
	expected := 1.0*4.0 + 2.0*5.0 + 3.0*6.0

	if result != expected {
		t.Errorf("Expected %f, got %f", expected, result)
	}
}

func TestSoftmax(t *testing.T) {
	values := []float64{1.0, 2.0, 3.0}

	result := softmax(values)

	sum := 0.0
	for _, v := range result {
		sum += v
	}

	if sum < 0.999 || sum > 1.001 {
		t.Errorf("Softmax sum should be ~1.0, got %f", sum)
	}

	for _, v := range result {
		if v < 0 || v > 1 {
			t.Errorf("Softmax values should be in [0,1], got %f", v)
		}
	}
}

func TestRelu(t *testing.T) {
	values := []float64{-1.0, 0.0, 1.0, -5.0, 3.0}

	result := relu(values)

	expected := []float64{0.0, 0.0, 1.0, 0.0, 3.0}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Expected %f at index %d, got %f", expected[i], i, v)
		}
	}
}

func TestExtractFeaturesFromTrace(t *testing.T) {
	trace := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 10.0, Y: 20.0},
			{Timestamp: 100, X: 15.0, Y: 25.0},
			{Timestamp: 200, X: 20.0, Y: 30.0},
		},
		TotalTime: 200,
	}

	vec := extractFeaturesFromTrace(trace)

	if len(vec) != TransformerXLModelDim {
		t.Errorf("Expected vector length %d, got %d", TransformerXLModelDim, len(vec))
	}

	if vec[0] != 3.0 {
		t.Errorf("Expected first feature (point count) 3, got %f", vec[0])
	}
}

func TestExtractFeaturesFromNilTrace(t *testing.T) {
	vec := extractFeaturesFromTrace(nil)

	if len(vec) != TransformerXLModelDim {
		t.Errorf("Expected vector length %d, got %d", TransformerXLModelDim, len(vec))
	}

	for i := 0; i < TransformerXLModelDim; i++ {
		if vec[i] != 0.0 {
			t.Errorf("Expected all zeros for nil trace, got %f at index %d", vec[i], i)
		}
	}
}

func TestExtractFeaturesFromEmptyTrace(t *testing.T) {
	trace := &model.TraceData{
		Points:    []model.TracePoint{},
		TotalTime: 0,
	}

	vec := extractFeaturesFromTrace(trace)

	if len(vec) != TransformerXLModelDim {
		t.Errorf("Expected vector length %d, got %d", TransformerXLModelDim, len(vec))
	}

	if vec[0] != 0.0 {
		t.Errorf("Expected first feature 0 for empty trace, got %f", vec[0])
	}
}

func TestBuildBehaviorSequence(t *testing.T) {
	service := NewTransformerXLService(nil)
	ctx := context.Background()
	service.Initialize(ctx)

	traces := make([]*model.TraceData, 2)
	for i := range traces {
		traces[i] = &model.TraceData{
			Points: []model.TracePoint{
				{Timestamp: int64(i * 100), X: float64(i), Y: float64(i)},
			},
			TotalTime: 100,
		}
	}

	sequence := service.buildBehaviorSequence("test_user", traces)

	if sequence.UserID != "test_user" {
		t.Errorf("Expected UserID 'test_user', got '%s'", sequence.UserID)
	}

	if len(sequence.BehaviorVecs) != 2 {
		t.Errorf("Expected 2 behavior vectors, got %d", len(sequence.BehaviorVecs))
	}

	for i, vec := range sequence.BehaviorVecs {
		if len(vec) != TransformerXLModelDim {
			t.Errorf("Expected vector length %d at index %d, got %d", TransformerXLModelDim, i, len(vec))
		}
	}
}

func TestTransformerXLServiceAnalyzeEmptyTraces(t *testing.T) {
	service := NewTransformerXLService(nil)
	ctx := context.Background()
	service.Initialize(ctx)

	_, err := service.AnalyzeBehavior(ctx, "test_user", []*model.TraceData{})
	if err == nil {
		t.Error("Expected error for empty traces")
	}
}

func TestTransformerXLServiceAnalyzeNilTraces(t *testing.T) {
	service := NewTransformerXLService(nil)
	ctx := context.Background()
	service.Initialize(ctx)

	_, err := service.AnalyzeBehavior(ctx, "test_user", nil)
	if err == nil {
		t.Error("Expected error for nil traces")
	}
}

func TestAttentionEntropyCalculation(t *testing.T) {
	txl := NewTransformerXL(nil)

	attentionScores := make([][][]float64, 1)
	attentionScores[0] = make([][]float64, 1)
	attentionScores[0][0] = []float64{0.25, 0.25, 0.25, 0.25}

	entropy := txl.calculateAttentionEntropy(attentionScores)

	expectedEntropy := -4 * 0.25 * math.Log(0.25)
	if entropy < expectedEntropy-0.1 || entropy > expectedEntropy+0.1 {
		t.Errorf("Expected entropy around %f, got %f", expectedEntropy, entropy)
	}
}

func TestAttentionEntropyEmptyScores(t *testing.T) {
	txl := NewTransformerXL(nil)

	entropy := txl.calculateAttentionEntropy([][][]float64{})

	if entropy != 0.0 {
		t.Errorf("Expected entropy 0.0 for empty scores, got %f", entropy)
	}
}

func BenchmarkTransformerXLForward(b *testing.B) {
	txl := NewTransformerXL(nil)

	tokenEmbeddings := make([]float64, TransformerXLModelDim)
	for j := range tokenEmbeddings {
		tokenEmbeddings[j] = 0.1
	}

	input := &TransformerXLInput{
		TokenEmbeddings: tokenEmbeddings,
		PositionIDs:     make([]int, 50),
		SegmentIDs:      make([]int, 50),
		Mask:            make([][]bool, 50),
	}

	for i := range input.Mask {
		input.Mask[i] = make([]bool, 50)
		for j := range input.Mask[i] {
			input.Mask[i][j] = true
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txl.Forward(input, nil)
	}
}

func BenchmarkTransformerXLMemoryUpdate(b *testing.B) {
	txl := NewTransformerXL(nil)

	hiddenStates := make([][]float64, 10)
	for i := range hiddenStates {
		hiddenStates[i] = make([]float64, TransformerXLModelDim)
		for j := range hiddenStates[i] {
			hiddenStates[i][j] = 0.1
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txl.UpdateMemory(hiddenStates)
	}
}

func TestBehaviorAnalysisResultStructure(t *testing.T) {
	result := &BehaviorAnalysisResult{
		UserID:            "user123",
		PredictiveScore:   0.75,
		Confidence:        0.85,
		IsAnomaly:         false,
		AnomalyReasons:    []string{},
		AttentionPattern:  map[string]interface{}{"pattern_type": "normal"},
		SequenceLength:    10,
		MemoryUsage:       512,
		ProcessedAt:      time.Now(),
	}

	if result.UserID != "user123" {
		t.Errorf("Expected UserID 'user123', got '%s'", result.UserID)
	}

	if result.PredictiveScore != 0.75 {
		t.Errorf("Expected PredictiveScore 0.75, got %f", result.PredictiveScore)
	}

	if result.IsAnomaly != false {
		t.Error("Expected IsAnomaly to be false")
	}
}

func TestXLMemoryStructure(t *testing.T) {
	memory := &XLMemory{
		HiddenStates: make([][]float64, 5),
		KeyStates:    make([][]float64, 5),
		ValueStates:  make([][]float64, 5),
		SegmentIDs:   make([]int, 5),
		Timestamps:  make([]int64, 5),
		Length:      5,
		Capacity:   10,
	}

	if memory.Length != 5 {
		t.Errorf("Expected Length 5, got %d", memory.Length)
	}

	if memory.Capacity != 10 {
		t.Errorf("Expected Capacity 10, got %d", memory.Capacity)
	}
}

func TestTransformerXLRelativePositionBias(t *testing.T) {
	txl := NewTransformerXL(nil)

	if len(txl.relativePosBias) != TransformerXLNumLayers {
		t.Errorf("Expected %d layers of relative position bias, got %d",
			TransformerXLNumLayers, len(txl.relativePosBias))
	}

	for layer := range txl.relativePosBias {
		if len(txl.relativePosBias[layer]) != TransformerXLNumHeads {
			t.Errorf("Expected %d heads in layer %d, got %d",
				TransformerXLNumHeads, layer, len(txl.relativePosBias[layer]))
		}
	}
}
