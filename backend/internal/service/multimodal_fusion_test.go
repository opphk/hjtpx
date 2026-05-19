package service

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestNewMultimodalFusion(t *testing.T) {
	config := &MultimodalConfig{
		EmbeddingDim: 128,
		NumHeads:     4,
		NumLayers:    2,
		FeatureDim:   64,
	}

	fusion := NewMultimodalFusion(config)
	if fusion == nil {
		t.Fatal("NewMultimodalFusion should not return nil")
	}

	if fusion.config.EmbeddingDim != 128 {
		t.Errorf("Expected EmbeddingDim 128, got %d", fusion.config.EmbeddingDim)
	}

	if len(fusion.encoders) == 0 {
		t.Error("Encoders should be initialized")
	}
}

func TestNewMultimodalFusionDefaultConfig(t *testing.T) {
	fusion := NewMultimodalFusion(nil)
	if fusion == nil {
		t.Fatal("NewMultimodalFusion with nil config should not return nil")
	}
	if fusion.config.EmbeddingDim != FusionEmbeddingDim {
		t.Errorf("Expected default EmbeddingDim %d, got %d", FusionEmbeddingDim, fusion.config.EmbeddingDim)
	}
}

func TestMultimodalFusionInitialize(t *testing.T) {
	fusion := NewMultimodalFusion(nil)
	ctx := context.Background()

	err := fusion.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}

	if !fusion.initialized {
		t.Error("Fusion should be initialized")
	}
}

func TestMultimodalFusionEncodeBehavioral(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	features := &BehavioralFeatures{
		Trajectory: [][]float64{
			{1.0, 2.0},
			{3.0, 4.0},
			{5.0, 6.0},
		},
		Speed: []float64{1.0, 1.5, 2.0},
		ClickData: []ClickData{
			{X: 10.0, Y: 20.0, Timestamp: 1000},
		},
	}

	encoding := fusion.encodeBehavioral(features)

	if len(encoding) != FusionFeatureDim {
		t.Errorf("Expected encoding length %d, got %d", FusionFeatureDim, len(encoding))
	}
}

func TestMultimodalFusionEncodeText(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	features := &TextFeatures{
		Tokens:         []int{1, 2, 3, 4, 5},
		TokenEmbeddings: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		TextLength:    5,
		Language:      "en",
	}

	encoding := fusion.encodeText(features)

	if len(encoding) != FusionFeatureDim {
		t.Errorf("Expected encoding length %d, got %d", FusionFeatureDim, len(encoding))
	}
}

func TestMultimodalFusionEncodeImage(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	features := &ImageFeatures{
		ImageData: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		Regions: []ImageRegion{
			{X: 0, Y: 0, Width: 100, Height: 100, Label: "object", Score: 0.9},
		},
		SceneType: "indoor",
	}

	encoding := fusion.encodeImage(features)

	if len(encoding) != FusionFeatureDim {
		t.Errorf("Expected encoding length %d, got %d", FusionFeatureDim, len(encoding))
	}
}

func TestMultimodalFusionEncodeAudio(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	features := &AudioFeatures{
		Waveform:   []float64{0.1, 0.2, 0.3},
		MFCC:       []float64{0.5, 0.6, 0.7},
		Duration:   2.5,
		SampleRate: 44100,
	}

	encoding := fusion.encodeAudio(features)

	if len(encoding) != FusionFeatureDim {
		t.Errorf("Expected encoding length %d, got %d", FusionFeatureDim, len(encoding))
	}
}

func TestMultimodalFusionEncodeSensor(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	features := &SensorFeatures{
		Accelerometer: []float64{0.1, 0.2, 0.3},
		Gyroscope:     []float64{0.4, 0.5, 0.6},
		TouchPressure: []float64{0.7},
		Orientation:   []float64{0.8, 0.9},
	}

	encoding := fusion.encodeSensor(features)

	if len(encoding) != FusionFeatureDim {
		t.Errorf("Expected encoding length %d, got %d", FusionFeatureDim, len(encoding))
	}
}

func TestMultimodalFusionEncodeModalFeatures(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	input := &MultimodalInput{
		Behavioral: &BehavioralFeatures{
			Trajectory: [][]float64{{1.0, 2.0}},
			Speed:      []float64{1.0},
		},
		Text: &TextFeatures{
			Tokens:   []int{1, 2},
			TextLength: 2,
		},
	}

	encoded, err := fusion.EncodeModalFeatures(input)
	if err != nil {
		t.Errorf("EncodeModalFeatures failed: %v", err)
	}

	if len(encoded) != 2 {
		t.Errorf("Expected 2 encoded modalities, got %d", len(encoded))
	}

	if _, ok := encoded[ModalTypeBehavioral]; !ok {
		t.Error("Should have Behavioral encoding")
	}

	if _, ok := encoded[ModalTypeText]; !ok {
		t.Error("Should have Text encoding")
	}
}

func TestMultimodalFusionAlignFeatures(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	embeddings := map[ModalType][]float64{
		ModalTypeBehavioral: make([]float64, FusionFeatureDim),
		ModalTypeText:       make([]float64, FusionFeatureDim),
	}

	for i := range embeddings[ModalTypeBehavioral] {
		embeddings[ModalTypeBehavioral][i] = 0.1 * float64(i)
	}

	aligned := fusion.AlignFeatures(embeddings)

	if len(aligned) != len(embeddings) {
		t.Errorf("Expected %d aligned embeddings, got %d", len(embeddings), len(aligned))
	}
}

func TestMultimodalFusionCalculateDynamicWeights(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	embeddings := map[ModalType][]float64{
		ModalTypeBehavioral: make([]float64, FusionFeatureDim),
		ModalTypeText:       make([]float64, FusionFeatureDim),
	}

	for i := range embeddings[ModalTypeBehavioral] {
		embeddings[ModalTypeBehavioral][i] = 0.1
		embeddings[ModalTypeText][i] = 0.2
	}

	confidences := map[ModalType]float64{
		ModalTypeBehavioral: 0.9,
		ModalTypeText:       0.8,
	}

	weights := fusion.CalculateDynamicWeights(embeddings, confidences)

	if len(weights) != len(embeddings) {
		t.Errorf("Expected %d weights, got %d", len(embeddings), len(weights))
	}

	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}

	if totalWeight < 0.99 || totalWeight > 1.01 {
		t.Errorf("Weights should sum to ~1.0, got %f", totalWeight)
	}
}

func TestMultimodalFusionCalculateEmbeddingQuality(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	embedding := make([]float64, 10)
	for i := range embedding {
		embedding[i] = float64(i)
	}

	quality := fusion.calculateEmbeddingQuality(embedding)

	if quality < 0 || quality > 1 {
		t.Errorf("Quality should be between 0 and 1, got %f", quality)
	}
}

func TestMultimodalFusionCalculateEmbeddingQualityEmpty(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	quality := fusion.calculateEmbeddingQuality([]float64{})

	if quality != 0.0 {
		t.Errorf("Expected quality 0.0 for empty embedding, got %f", quality)
	}
}

func TestMultimodalFusionCalculateAttentionScore(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	embedding := make([]float64, 20)
	for i := range embedding {
		embedding[i] = math.Sin(float64(i))
	}

	score := fusion.calculateAttentionScore(embedding)

	if score < 0 || score > 1 {
		t.Errorf("Attention score should be between 0 and 1, got %f", score)
	}
}

func TestMultimodalFusionFuse(t *testing.T) {
	fusion := NewMultimodalFusion(nil)
	fusion.initialized = true

	embeddings := map[ModalType][]float64{
		ModalTypeBehavioral: make([]float64, FusionEmbeddingDim),
		ModalTypeText:       make([]float64, FusionEmbeddingDim),
	}

	for i := range embeddings[ModalTypeBehavioral] {
		embeddings[ModalTypeBehavioral][i] = 1.0
		embeddings[ModalTypeText][i] = 2.0
	}

	weights := map[ModalType]float64{
		ModalTypeBehavioral: 0.6,
		ModalTypeText:       0.4,
	}

	fused := fusion.Fuse(embeddings, weights)

	if len(fused) != FusionEmbeddingDim {
		t.Errorf("Expected fused length %d, got %d", FusionEmbeddingDim, len(fused))
	}
}

func TestMultimodalFusionFuseEmpty(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	fused := fusion.Fuse(map[ModalType][]float64{}, map[ModalType]float64{})

	if len(fused) != FusionEmbeddingDim {
		t.Errorf("Expected fused length %d, got %d", FusionEmbeddingDim, len(fused))
	}
}

func TestMultimodalFusionProcess(t *testing.T) {
	fusion := NewMultimodalFusion(nil)
	ctx := context.Background()
	fusion.Initialize(ctx)

	input := &MultimodalInput{
		Behavioral: &BehavioralFeatures{
			Trajectory: [][]float64{{1.0, 2.0}},
			Speed:      []float64{1.0},
		},
	}

	result, err := fusion.Process(ctx, input)
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if len(result.FusedEmbedding) != FusionEmbeddingDim {
		t.Errorf("Expected fused embedding length %d, got %d", FusionEmbeddingDim, len(result.FusedEmbedding))
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", result.Confidence)
	}
}

func TestMultimodalFusionProcessNotInitialized(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	input := &MultimodalInput{
		Behavioral: &BehavioralFeatures{},
	}

	_, err := fusion.Process(context.Background(), input)
	if err == nil {
		t.Error("Expected error for uninitialized fusion")
	}
}

func TestMultimodalFusionCrossModalAttention(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	query := make([]float64, FusionEmbeddingDim)
	key := make([]float64, FusionEmbeddingDim)
	value := make([]float64, FusionEmbeddingDim)

	for i := range query {
		query[i] = 0.1
		key[i] = 0.2
		value[i] = 0.3
	}

	output := fusion.CrossModalAttention(query, key, value, ModalTypeBehavioral, ModalTypeText)

	if len(output) != FusionEmbeddingDim {
		t.Errorf("Expected output length %d, got %d", FusionEmbeddingDim, len(output))
	}
}

func TestMultimodalFusionTransform(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	input := make([]float64, 10)
	for i := range input {
		input[i] = float64(i)
	}

	weights := createRandomMatrix(FusionEmbeddingDim, 10, 0.02)

	output := fusion.transform(input, weights)

	if len(output) != FusionEmbeddingDim {
		t.Errorf("Expected output length %d, got %d", FusionEmbeddingDim, len(output))
	}
}

func TestMultimodalFusionEstimateModalConfidences(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	input := &MultimodalInput{
		Behavioral: &BehavioralFeatures{},
		Text:       &TextFeatures{},
		Image:      &ImageFeatures{},
		Audio:      &AudioFeatures{},
		Sensor:     &SensorFeatures{},
	}

	confidences := fusion.estimateModalConfidences(input)

	if confidences[ModalTypeBehavioral] != 0.9 {
		t.Errorf("Expected Behavioral confidence 0.9, got %f", confidences[ModalTypeBehavioral])
	}

	if confidences[ModalTypeText] != 0.8 {
		t.Errorf("Expected Text confidence 0.8, got %f", confidences[ModalTypeText])
	}
}

func TestMultimodalFusionComputeAttentionMaps(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	embeddings := map[ModalType][]float64{
		ModalTypeBehavioral: make([]float64, FusionEmbeddingDim),
		ModalTypeText:       make([]float64, FusionEmbeddingDim),
	}

	for i := range embeddings[ModalTypeBehavioral] {
		embeddings[ModalTypeBehavioral][i] = 0.1 * float64(i)
		embeddings[ModalTypeText][i] = 0.2 * float64(i)
	}

	maps := fusion.computeAttentionMaps(embeddings)

	if len(maps) != len(embeddings) {
		t.Errorf("Expected %d attention maps, got %d", len(embeddings), len(maps))
	}

	for modal, attention := range maps {
		if len(attention) != fusion.config.NumHeads {
			t.Errorf("Expected %d heads for %s, got %d", fusion.config.NumHeads, modal, len(attention))
		}
	}
}

func TestMultimodalFusionComputeModalScores(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	embeddings := map[ModalType][]float64{
		ModalTypeBehavioral: make([]float64, FusionEmbeddingDim),
	}

	weights := map[ModalType]float64{
		ModalTypeBehavioral: 1.0,
	}

	scores := fusion.computeModalScores(embeddings, weights)

	if len(scores) != len(embeddings) {
		t.Errorf("Expected %d scores, got %d", len(embeddings), len(scores))
	}
}

func TestMultimodalFusionCalculateOverallConfidence(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	scores := map[ModalType]float64{
		ModalTypeBehavioral: 0.8,
		ModalTypeText:       0.6,
	}

	weights := map[ModalType]float64{
		ModalTypeBehavioral: 0.6,
		ModalTypeText:       0.4,
	}

	confidence := fusion.calculateOverallConfidence(scores, weights)

	if confidence < 0 || confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", confidence)
	}
}

func TestMultimodalFusionCalculateOverallConfidenceEmpty(t *testing.T) {
	fusion := NewMultimodalFusion(nil)

	confidence := fusion.calculateOverallConfidence(map[ModalType]float64{}, map[ModalType]float64{})

	if confidence != 0.0 {
		t.Errorf("Expected confidence 0.0 for empty scores, got %f", confidence)
	}
}

func TestNewMultimodalFusionService(t *testing.T) {
	service := NewMultimodalFusionService(nil)
	if service == nil {
		t.Fatal("NewMultimodalFusionService should not return nil")
	}
	if service.fusion == nil {
		t.Error("Service should have fusion")
	}
}

func TestMultimodalFusionServiceInitialize(t *testing.T) {
	service := NewMultimodalFusionService(nil)
	ctx := context.Background()

	err := service.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}
}

func TestMultimodalFusionServiceFuseMultimodalInput(t *testing.T) {
	service := NewMultimodalFusionService(nil)
	ctx := context.Background()
	service.Initialize(ctx)

	input := &MultimodalInput{
		Behavioral: &BehavioralFeatures{
			Trajectory: [][]float64{{1.0, 2.0}},
			Speed:      []float64{1.0},
		},
	}

	result, err := service.FuseMultimodalInput(ctx, input)
	if err != nil {
		t.Errorf("FuseMultimodalInput failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}
}

func TestMultimodalFusionServiceGetHistory(t *testing.T) {
	service := NewMultimodalFusionService(nil)
	ctx := context.Background()
	service.Initialize(ctx)

	input := &MultimodalInput{
		Behavioral: &BehavioralFeatures{},
	}

	for i := 0; i < 5; i++ {
		service.FuseMultimodalInput(ctx, input)
	}

	history := service.GetHistory(3)
	if len(history) != 3 {
		t.Errorf("Expected 3 history items, got %d", len(history))
	}
}

func TestMultimodalFusionServiceGetHistoryNoLimit(t *testing.T) {
	service := NewMultimodalFusionService(nil)

	history := service.GetHistory(0)
	if len(history) != 0 {
		t.Errorf("Expected 0 history items initially, got %d", len(history))
	}
}

func TestMultimodalFusionServiceGetStatistics(t *testing.T) {
	service := NewMultimodalFusionService(nil)

	stats := service.GetStatistics()
	if stats == nil {
		t.Fatal("Statistics should not be nil")
	}

	if _, ok := stats["history_size"]; !ok {
		t.Error("Should have history_size stat")
	}
}

func TestMultimodalFusionServiceGetCurrentWeights(t *testing.T) {
	service := NewMultimodalFusionService(nil)

	weights := service.GetCurrentWeights()
	if weights == nil {
		t.Fatal("Weights should not be nil")
	}
}

func TestMultimodalFusionServiceResetWeights(t *testing.T) {
	service := NewMultimodalFusionService(nil)

	service.ResetWeights()

	weights := service.GetCurrentWeights()
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}

	if totalWeight < 0.99 || totalWeight > 1.01 {
		t.Errorf("Weights should sum to ~1.0 after reset, got %f", totalWeight)
	}
}

func TestFusionResultStructure(t *testing.T) {
	result := &FusionResult{
		FusedEmbedding: make([]float64, FusionEmbeddingDim),
		AttentionMaps: map[ModalType][][]float64{},
		ModalScores:   map[ModalType]float64{},
		Contribution:  map[ModalType]float64{},
		Confidence:    0.85,
		ProcessedAt:  time.Now(),
	}

	if len(result.FusedEmbedding) != FusionEmbeddingDim {
		t.Errorf("Expected embedding length %d, got %d", FusionEmbeddingDim, len(result.FusedEmbedding))
	}

	if result.Confidence != 0.85 {
		t.Errorf("Expected confidence 0.85, got %f", result.Confidence)
	}
}

func TestMultimodalInputStructure(t *testing.T) {
	input := &MultimodalInput{
		Behavioral: &BehavioralFeatures{
			Trajectory: [][]float64{{1.0, 2.0}},
		},
		Text: &TextFeatures{
			Tokens: []int{1, 2, 3},
		},
		Image: &ImageFeatures{
			ImageData: []float64{0.1, 0.2},
		},
	}

	if input.Behavioral == nil {
		t.Error("Behavioral should not be nil")
	}
	if input.Text == nil {
		t.Error("Text should not be nil")
	}
	if input.Image == nil {
		t.Error("Image should not be nil")
	}
}

func TestBehavioralFeaturesStructure(t *testing.T) {
	features := &BehavioralFeatures{
		Trajectory:   [][]float64{{1.0, 2.0}, {3.0, 4.0}},
		Speed:       []float64{1.0, 1.5, 2.0},
		ClickData:   []ClickData{{X: 10, Y: 20, Timestamp: 1000}},
		DeviceInfo:  map[string]interface{}{"type": "mobile"},
		SessionInfo: map[string]interface{}{"id": "session123"},
	}

	if len(features.Trajectory) != 2 {
		t.Errorf("Expected 2 trajectory points, got %d", len(features.Trajectory))
	}

	if len(features.ClickData) != 1 {
		t.Errorf("Expected 1 click data, got %d", len(features.ClickData))
	}
}

func TestTextFeaturesStructure(t *testing.T) {
	features := &TextFeatures{
		Tokens:         []int{1, 2, 3, 4, 5},
		TokenEmbeddings: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		AttentionMask: []bool{true, true, true, false, false},
		TextLength:     5,
		Language:      "en",
	}

	if features.TextLength != 5 {
		t.Errorf("Expected TextLength 5, got %d", features.TextLength)
	}
}

func TestImageFeaturesStructure(t *testing.T) {
	features := &ImageFeatures{
		ImageData: []float64{0.1, 0.2, 0.3},
		Regions: []ImageRegion{
			{X: 0, Y: 0, Width: 100, Height: 100, Label: "object", Score: 0.9},
		},
		ObjectDetection: []ObjectInfo{
			{Class: "person", Confidence: 0.95},
		},
		SceneType: "indoor",
	}

	if len(features.Regions) != 1 {
		t.Errorf("Expected 1 region, got %d", len(features.Regions))
	}

	if len(features.ObjectDetection) != 1 {
		t.Errorf("Expected 1 object, got %d", len(features.ObjectDetection))
	}
}

func TestAudioFeaturesStructure(t *testing.T) {
	features := &AudioFeatures{
		Waveform:   []float64{0.1, 0.2, 0.3},
		MFCC:       []float64{0.5, 0.6, 0.7},
		Spectrogram: []float64{0.8, 0.9},
		Duration:   2.5,
		SampleRate: 44100,
	}

	if features.Duration != 2.5 {
		t.Errorf("Expected Duration 2.5, got %f", features.Duration)
	}

	if features.SampleRate != 44100 {
		t.Errorf("Expected SampleRate 44100, got %d", features.SampleRate)
	}
}

func TestSensorFeaturesStructure(t *testing.T) {
	features := &SensorFeatures{
		Accelerometer: []float64{0.1, 0.2, 0.3},
		Gyroscope:     []float64{0.4, 0.5, 0.6},
		TouchPressure: []float64{0.7, 0.8},
		Orientation:   []float64{0.9, 1.0, 1.1},
	}

	if len(features.Accelerometer) != 3 {
		t.Errorf("Expected 3 accelerometer values, got %d", len(features.Accelerometer))
	}
}

func TestFeatureEncoderStructure(t *testing.T) {
	encoder := &FeatureEncoder{
		Weights:   createRandomMatrix(128, 256, 0.02),
		Bias:      createRandomVector(256, 0.02),
		ModalType: ModalTypeBehavioral,
		OutputDim: 256,
	}

	if encoder.ModalType != ModalTypeBehavioral {
		t.Errorf("Expected ModalType Behavioral, got %s", encoder.ModalType)
	}

	if encoder.OutputDim != 256 {
		t.Errorf("Expected OutputDim 256, got %d", encoder.OutputDim)
	}
}

func TestCrossModalAttentionStructure(t *testing.T) {
	attn := &CrossModalAttention{
		QueryWeights:  createRandomMatrix(256, 256, 0.02),
		KeyWeights:    createRandomMatrix(256, 256, 0.02),
		ValueWeights:  createRandomMatrix(256, 256, 0.02),
		OutputWeights: createRandomMatrix(256, 256, 0.02),
		NumHeads:      8,
		HeadDim:      32,
	}

	if attn.NumHeads != 8 {
		t.Errorf("Expected NumHeads 8, got %d", attn.NumHeads)
	}
}

func TestFusionLayerStructure(t *testing.T) {
	layer := &FusionLayer{
		CrossAttention: &CrossModalAttention{},
		SelfAttention:  &CrossModalAttention{},
		FFN:            &FeedForward{},
		LayerNorm1:    createLayerNormVec(256),
		LayerNorm2:    createLayerNormVec(256),
	}

	if layer.CrossAttention == nil {
		t.Error("CrossAttention should not be nil")
	}
	if layer.SelfAttention == nil {
		t.Error("SelfAttention should not be nil")
	}
	if layer.FFN == nil {
		t.Error("FFN should not be nil")
	}
}

func TestFeedForwardStructure(t *testing.T) {
	ffn := &FeedForward{
		Weights1: createRandomMatrix(256, 1024, 0.02),
		Weights2: createRandomMatrix(1024, 256, 0.02),
		Bias1:    createRandomVector(1024, 0.02),
		Bias2:    createRandomVector(256, 0.02),
	}

	if len(ffn.Bias1) != 1024 {
		t.Errorf("Expected Bias1 length 1024, got %d", len(ffn.Bias1))
	}
}

func BenchmarkMultimodalFusionEncodeModalFeatures(b *testing.B) {
	fusion := NewMultimodalFusion(nil)

	input := &MultimodalInput{
		Behavioral: &BehavioralFeatures{
			Trajectory: make([][]float64, 100),
			Speed:      make([]float64, 100),
		},
		Text:  &TextFeatures{},
		Image: &ImageFeatures{},
	}

	for i := 0; i < b.N; i++ {
		fusion.EncodeModalFeatures(input)
	}
}

func BenchmarkMultimodalFusionFuse(b *testing.B) {
	fusion := NewMultimodalFusion(nil)
	fusion.initialized = true

	embeddings := map[ModalType][]float64{
		ModalTypeBehavioral: make([]float64, FusionEmbeddingDim),
		ModalTypeText:       make([]float64, FusionEmbeddingDim),
		ModalTypeImage:      make([]float64, FusionEmbeddingDim),
	}

	weights := map[ModalType]float64{
		ModalTypeBehavioral: 0.5,
		ModalTypeText:       0.3,
		ModalTypeImage:      0.2,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fusion.Fuse(embeddings, weights)
	}
}

func BenchmarkMultimodalFusionCalculateDynamicWeights(b *testing.B) {
	fusion := NewMultimodalFusion(nil)

	embeddings := map[ModalType][]float64{
		ModalTypeBehavioral: make([]float64, FusionFeatureDim),
		ModalTypeText:       make([]float64, FusionFeatureDim),
	}

	confidences := map[ModalType]float64{
		ModalTypeBehavioral: 0.9,
		ModalTypeText:       0.8,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fusion.CalculateDynamicWeights(embeddings, confidences)
	}
}
