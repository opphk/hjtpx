package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

const (
	FusionEmbeddingDim   = 256
	FusionNumHeads       = 8
	FusionNumLayers      = 4
	FusionFeatureDim     = 128
)

type ModalType string

const (
	ModalTypeBehavioral ModalType = "behavioral"
	ModalTypeText       ModalType = "text"
	ModalTypeImage       ModalType = "image"
	ModalTypeAudio      ModalType = "audio"
	ModalTypeSensor     ModalType = "sensor"
)

type MultimodalConfig struct {
	EmbeddingDim   int
	NumHeads       int
	NumLayers      int
	FeatureDim     int
	AttentionDim   int
	Dropout        float64
}

type MultimodalInput struct {
	Behavioral *BehavioralFeatures
	Text       *TextFeatures
	Image      *ImageFeatures
	Audio      *AudioFeatures
	Sensor     *SensorFeatures
}

type BehavioralFeatures struct {
	Trajectory  [][]float64
	Speed       []float64
	Acceleration []float64
	Curvature   []float64
	ClickData   []ClickData
	ScrollData  []ScrollData
	DeviceInfo  map[string]interface{}
	SessionInfo map[string]interface{}
}

type TextFeatures struct {
	Tokens      []int
	TokenEmbeddings []float64
	AttentionMask []bool
	TextLength int
	Language    string
}

type ImageFeatures struct {
	ImageData   []float64
	Regions     []ImageRegion
	ObjectDetection []ObjectInfo
	SceneType   string
}

type ImageRegion struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
	Label  string
	Score  float64
}

type ObjectInfo struct {
	Class     string
	Confidence float64
	BoundingBox ImageRegion
}

type AudioFeatures struct {
	Waveform   []float64
	MFCC       []float64
	Spectrogram []float64
	Duration   float64
	SampleRate int
}

type SensorFeatures struct {
	Accelerometer []float64
	Gyroscope     []float64
	TouchPressure []float64
	Orientation   []float64
}

type ClickData struct {
	X         float64
	Y         float64
	Timestamp int64
	Pressure  float64
	HoldTime  int64
}

type ScrollData struct {
	Direction string
	Velocity  float64
	Timestamp int64
}

type FeatureEncoder struct {
	Weights     [][]float64
	Bias        []float64
	ModalType   ModalType
	OutputDim   int
}

type CrossModalAttention struct {
	QueryWeights  [][]float64
	KeyWeights    [][]float64
	ValueWeights  [][]float64
	OutputWeights [][]float64
	NumHeads      int
	HeadDim      int
}

type MultimodalAttention struct {
	QueryWeights  [][]float64
	KeyWeights    [][]float64
	ValueWeights  [][]float64
	OutputWeights [][]float64
	Layers        []*CrossModalAttention
}

type FusionLayer struct {
	CrossAttention *CrossModalAttention
	SelfAttention  *CrossModalAttention
	FFN            *FeedForward
	LayerNorm1    []float64
	LayerNorm2    []float64
}

type FeedForward struct {
	Weights1 [][]float64
	Weights2 [][]float64
	Bias1    []float64
	Bias2    []float64
}

type MultimodalFusion struct {
	config         MultimodalConfig
	encoders       map[ModalType]*FeatureEncoder
	crossAttention *MultimodalAttention
	fusion_layers  []*FusionLayer
	alignment_matrices map[ModalType][][]float64
	dynamic_weights map[ModalType]float64
	mu             sync.RWMutex
	initialized    bool
}

type FusionResult struct {
	FusedEmbedding []float64
	AttentionMaps  map[ModalType][][]float64
	ModalScores    map[ModalType]float64
	Contribution   map[ModalType]float64
	Confidence     float64
	ProcessedAt    time.Time
}

type AlignmentModule struct {
	TransformMatrix [][]float64
	ProjectionDim   int
	SourceDim      int
	TargetDim      int
}

type DynamicWeightCalculator struct {
	attention_weights [][][]float64
	confidence_weights []float64
	history_weights   map[ModalType][]float64
	update_interval   time.Duration
	last_update       time.Time
}

func NewMultimodalFusion(config *MultimodalConfig) *MultimodalFusion {
	if config == nil {
		config = &MultimodalConfig{
			EmbeddingDim:   FusionEmbeddingDim,
			NumHeads:        FusionNumHeads,
			NumLayers:       FusionNumLayers,
			FeatureDim:      FusionFeatureDim,
			AttentionDim:    64,
			Dropout:         0.1,
		}
	}

	fusion := &MultimodalFusion{
		config:            *config,
		encoders:          make(map[ModalType]*FeatureEncoder),
		fusion_layers:     make([]*FusionLayer, config.NumLayers),
		alignment_matrices: make(map[ModalType][][]float64),
		dynamic_weights:   make(map[ModalType]float64),
		initialized:       false,
	}

	fusion.initializeEncoders()
	fusion.initializeCrossAttention()
	fusion.initializeFusionLayers()
	fusion.initializeAlignment()
	fusion.initializeDynamicWeights()

	return fusion
}

func (m *MultimodalFusion) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		for modal := range m.encoders {
			m.dynamic_weights[modal] = 1.0 / float64(len(m.encoders))
		}
	}

	m.initialized = true
	return nil
}

func (m *MultimodalFusion) initializeEncoders() {
	modalTypes := []ModalType{ModalTypeBehavioral, ModalTypeText, ModalTypeImage, ModalTypeAudio, ModalTypeSensor}

	for _, modal := range modalTypes {
		encoder := &FeatureEncoder{
			Weights:   createRandomMatrix(FusionFeatureDim, m.config.EmbeddingDim, 0.02),
			Bias:      createRandomVector(m.config.EmbeddingDim, 0.02),
			ModalType: modal,
			OutputDim: m.config.EmbeddingDim,
		}
		m.encoders[modal] = encoder
		m.dynamic_weights[modal] = 1.0 / float64(len(m.encoders))
	}
}

func (m *MultimodalFusion) initializeCrossAttention() {
	m.crossAttention = &MultimodalAttention{
		QueryWeights:  createRandomMatrix(m.config.EmbeddingDim, m.config.EmbeddingDim, 0.02),
		KeyWeights:    createRandomMatrix(m.config.EmbeddingDim, m.config.EmbeddingDim, 0.02),
		ValueWeights:  createRandomMatrix(m.config.EmbeddingDim, m.config.EmbeddingDim, 0.02),
		OutputWeights: createRandomMatrix(m.config.EmbeddingDim, m.config.EmbeddingDim, 0.02),
		Layers:        make([]*CrossModalAttention, m.config.NumLayers),
	}

	for i := 0; i < m.config.NumLayers; i++ {
		headDim := m.config.EmbeddingDim / m.config.NumHeads
		layer := &CrossModalAttention{
			QueryWeights:  createRandomMatrix(m.config.EmbeddingDim, m.config.EmbeddingDim, 0.02),
			KeyWeights:    createRandomMatrix(m.config.EmbeddingDim, m.config.EmbeddingDim, 0.02),
			ValueWeights:  createRandomMatrix(m.config.EmbeddingDim, m.config.EmbeddingDim, 0.02),
			OutputWeights: createRandomMatrix(m.config.EmbeddingDim, m.config.EmbeddingDim, 0.02),
			NumHeads:      m.config.NumHeads,
			HeadDim:      headDim,
		}
		m.crossAttention.Layers[i] = layer
	}
}

func (m *MultimodalFusion) initializeFusionLayers() {
	for i := 0; i < m.config.NumLayers; i++ {
		layer := &FusionLayer{
			CrossAttention: m.crossAttention.Layers[i],
			SelfAttention: &CrossModalAttention{
				QueryWeights:  createRandomMatrix(m.config.EmbeddingDim, m.config.EmbeddingDim, 0.02),
				KeyWeights:    createRandomMatrix(m.config.EmbeddingDim, m.config.EmbeddingDim, 0.02),
				ValueWeights:  createRandomMatrix(m.config.EmbeddingDim, m.config.EmbeddingDim, 0.02),
				OutputWeights: createRandomMatrix(m.config.EmbeddingDim, m.config.EmbeddingDim, 0.02),
				NumHeads:      m.config.NumHeads,
				HeadDim:      m.config.EmbeddingDim / m.config.NumHeads,
			},
			FFN: &FeedForward{
				Weights1: createRandomMatrix(m.config.EmbeddingDim, m.config.EmbeddingDim*4, 0.02),
				Weights2: createRandomMatrix(m.config.EmbeddingDim*4, m.config.EmbeddingDim, 0.02),
				Bias1:    createRandomVector(m.config.EmbeddingDim*4, 0.02),
				Bias2:    createRandomVector(m.config.EmbeddingDim, 0.02),
			},
			LayerNorm1: createLayerNormVec(m.config.EmbeddingDim),
			LayerNorm2: createLayerNormVec(m.config.EmbeddingDim),
		}
		m.fusion_layers[i] = layer
	}
}

func (m *MultimodalFusion) initializeAlignment() {
	modalTypes := []ModalType{ModalTypeBehavioral, ModalTypeText, ModalTypeImage, ModalTypeAudio, ModalTypeSensor}

	for _, modal := range modalTypes {
		m.alignment_matrices[modal] = createRandomMatrix(FusionFeatureDim, m.config.EmbeddingDim, 0.02)
	}
}

func (m *MultimodalFusion) initializeDynamicWeights() {
	modalTypes := []ModalType{ModalTypeBehavioral, ModalTypeText, ModalTypeImage, ModalTypeAudio, ModalTypeSensor}

	for _, modal := range modalTypes {
		m.dynamic_weights[modal] = 1.0 / float64(len(modalTypes))
	}
}

func createLayerNormVec(dim int) []float64 {
	vec := make([]float64, dim*2)
	for i := 0; i < dim; i++ {
		vec[i] = 1.0
	}
	return vec
}

func (m *MultimodalFusion) EncodeModalFeatures(input *MultimodalInput) (map[ModalType][]float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	encoded := make(map[ModalType][]float64)

	if input.Behavioral != nil {
		encoded[ModalTypeBehavioral] = m.encodeBehavioral(input.Behavioral)
	}

	if input.Text != nil {
		encoded[ModalTypeText] = m.encodeText(input.Text)
	}

	if input.Image != nil {
		encoded[ModalTypeImage] = m.encodeImage(input.Image)
	}

	if input.Audio != nil {
		encoded[ModalTypeAudio] = m.encodeAudio(input.Audio)
	}

	if input.Sensor != nil {
		encoded[ModalTypeSensor] = m.encodeSensor(input.Sensor)
	}

	return encoded, nil
}

func (m *MultimodalFusion) encodeBehavioral(features *BehavioralFeatures) []float64 {
	embedding := make([]float64, FusionFeatureDim)

	if len(features.Trajectory) > 0 {
		for i, point := range features.Trajectory {
			if i < FusionFeatureDim {
				embedding[i] = point[0]
			}
		}
	}

	if len(features.Speed) > 0 {
		speedSum := 0.0
		for _, s := range features.Speed {
			speedSum += s
		}
		avgSpeed := speedSum / float64(len(features.Speed))
		if FusionFeatureDim < len(embedding) {
			embedding[FusionFeatureDim-3] = avgSpeed
		}
	}

	featureIdx := 0
	for _, click := range features.ClickData {
		if featureIdx < FusionFeatureDim-3 {
			embedding[featureIdx] = click.X
			featureIdx++
		}
		if featureIdx < FusionFeatureDim-3 {
			embedding[featureIdx] = click.Y
			featureIdx++
		}
		if featureIdx < FusionFeatureDim-3 {
			embedding[featureIdx] = float64(click.Timestamp)
			featureIdx++
		}
	}

	return embedding
}

func (m *MultimodalFusion) encodeText(features *TextFeatures) []float64 {
	embedding := make([]float64, FusionFeatureDim)

	if len(features.TokenEmbeddings) > 0 {
		for i, val := range features.TokenEmbeddings {
			if i < FusionFeatureDim {
				embedding[i] = val
			}
		}
	}

	if FusionFeatureDim > 0 && len(embedding) > 3 {
		embedding[FusionFeatureDim-2] = float64(features.TextLength)
	}

	return embedding
}

func (m *MultimodalFusion) encodeImage(features *ImageFeatures) []float64 {
	embedding := make([]float64, FusionFeatureDim)

	if len(features.ImageData) > 0 {
		for i, val := range features.ImageData {
			if i < FusionFeatureDim {
				embedding[i] = val
			}
		}
	}

	regionIdx := 0
	for _, region := range features.Regions {
		if regionIdx < 5 && regionIdx*4+3 < FusionFeatureDim {
			embedding[regionIdx*4] = region.X
			embedding[regionIdx*4+1] = region.Y
			embedding[regionIdx*4+2] = region.Width
			embedding[regionIdx*4+3] = region.Height
			regionIdx++
		}
	}

	return embedding
}

func (m *MultimodalFusion) encodeAudio(features *AudioFeatures) []float64 {
	embedding := make([]float64, FusionFeatureDim)

	if len(features.MFCC) > 0 {
		for i, val := range features.MFCC {
			if i < FusionFeatureDim {
				embedding[i] = val
			}
		}
	}

	if FusionFeatureDim > 0 && len(embedding) > 1 {
		embedding[FusionFeatureDim-1] = features.Duration
	}

	return embedding
}

func (m *MultimodalFusion) encodeSensor(features *SensorFeatures) []float64 {
	embedding := make([]float64, FusionFeatureDim)

	if len(features.Accelerometer) > 0 {
		for i, val := range features.Accelerometer {
			if i < FusionFeatureDim {
				embedding[i] = val
			}
		}
	}

	offset := len(features.Accelerometer)
	for i, val := range features.Gyroscope {
		if offset+i < FusionFeatureDim {
			embedding[offset+i] = val
		}
	}

	return embedding
}

func (m *MultimodalFusion) AlignFeatures(modalEmbeddings map[ModalType][]float64) map[ModalType][]float64 {
	aligned := make(map[ModalType][]float64)

	for modal, embedding := range modalEmbeddings {
		alignedEmbedding := m.alignSingleModal(embedding, modal)
		aligned[modal] = alignedEmbedding
	}

	return aligned
}

func (m *MultimodalFusion) alignSingleModal(embedding []float64, modal ModalType) []float64 {
	aligned := make([]float64, m.config.EmbeddingDim)
	transformMatrix, exists := m.alignment_matrices[modal]

	if !exists || len(transformMatrix) == 0 {
		if len(embedding) >= m.config.EmbeddingDim {
			copy(aligned, embedding[:m.config.EmbeddingDim])
		}
		return aligned
	}

	for i := range aligned {
		for j := 0; j < len(embedding); j++ {
			if i < len(transformMatrix) && j < len(transformMatrix[i]) {
				aligned[i] += embedding[j] * transformMatrix[i][j]
			}
		}
	}

	return aligned
}

func (m *MultimodalFusion) CalculateDynamicWeights(modalEmbeddings map[ModalType][]float64, confidences map[ModalType]float64) map[ModalType]float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	weights := make(map[ModalType]float64)
	var totalScore float64

	for modal, embedding := range modalEmbeddings {
		embeddingScore := m.calculateEmbeddingQuality(embedding)

		confidenceScore := 0.5
		if conf, ok := confidences[modal]; ok {
			confidenceScore = conf
		}

		attentionScore := m.calculateAttentionScore(embedding)

		score := (embeddingScore + confidenceScore + attentionScore) / 3.0
		weights[modal] = score
		totalScore += score
	}

	if totalScore > 0 {
		for modal := range weights {
			weights[modal] /= totalScore
		}
	}

	for modal, weight := range weights {
		m.dynamic_weights[modal] = m.dynamic_weights[modal]*0.9 + weight*0.1
	}

	return weights
}

func (m *MultimodalFusion) calculateEmbeddingQuality(embedding []float64) float64 {
	if len(embedding) == 0 {
		return 0.0
	}

	variance := 0.0
	mean := 0.0
	for _, val := range embedding {
		mean += val
	}
	mean /= float64(len(embedding))

	for _, val := range embedding {
		diff := val - mean
		variance += diff * diff
	}
	variance /= float64(len(embedding))

	nonZeroCount := 0
	for _, val := range embedding {
		if math.Abs(val) > 0.001 {
			nonZeroCount++
		}
	}
	nonZeroRatio := float64(nonZeroCount) / float64(len(embedding))

	return (math.Min(1.0, math.Sqrt(variance)*10) + nonZeroRatio) / 2.0
}

func (m *MultimodalFusion) calculateAttentionScore(embedding []float64) float64 {
	if len(embedding) < 10 {
		return 0.5
	}

	maxAbs := 0.0
	for _, val := range embedding {
		if math.Abs(val) > maxAbs {
			maxAbs = math.Abs(val)
		}
	}

	if maxAbs < 0.001 {
		return 0.1
	}

	normalized := make([]float64, len(embedding))
	for i, val := range embedding {
		normalized[i] = val / maxAbs
	}

	attention := softmax(normalized)
	entropy := 0.0
	for _, a := range attention {
		if a > 0 {
			entropy -= a * math.Log(a+1e-10)
		}
	}

	maxEntropy := math.Log(float64(len(embedding)))
	if maxEntropy > 0 {
		entropy /= maxEntropy
	}

	return entropy
}

func (m *MultimodalFusion) CrossModalAttention(query, key, value []float64, modalFrom, modalTo ModalType) []float64 {
	queryTransformed := m.transform(query, m.crossAttention.QueryWeights)
	keyTransformed := m.transform(key, m.crossAttention.KeyWeights)
	valueTransformed := m.transform(value, m.crossAttention.ValueWeights)

	attentionScores := make([]float64, len(queryTransformed))
	for i := range attentionScores {
		attentionScores[i] = dotProduct(queryTransformed, keyTransformed) / math.Sqrt(float64(m.config.EmbeddingDim))
	}

	attentionWeights := softmax(attentionScores)

	output := make([]float64, m.config.EmbeddingDim)
	for i, weight := range attentionWeights {
		for j := range output {
			output[j] += weight * valueTransformed[i] * (float64(len(valueTransformed)) / float64(len(value)))
		}
	}

	output = m.transform(output, m.crossAttention.OutputWeights)

	return output
}

func (m *MultimodalFusion) transform(input []float64, weights [][]float64) []float64 {
	output := make([]float64, len(weights))
	for i := range output {
		for j := range input {
			if j < len(weights[i]) {
				output[i] += input[j] * weights[i][j]
			}
		}
	}
	return output
}

func (m *MultimodalFusion) Fuse(modalEmbeddings map[ModalType][]float64, weights map[ModalType]float64) []float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(modalEmbeddings) == 0 {
		return make([]float64, m.config.EmbeddingDim)
	}

	fused := make([]float64, m.config.EmbeddingDim)
	var totalWeight float64

	for modal, embedding := range modalEmbeddings {
		weight := 0.2
		if w, ok := weights[modal]; ok {
			weight = w
		}
		if dw, ok := m.dynamic_weights[modal]; ok {
			weight = (weight + dw) / 2.0
		}

		for i := 0; i < m.config.EmbeddingDim && i < len(embedding); i++ {
			fused[i] += embedding[i] * weight
		}
		totalWeight += weight
	}

	if totalWeight > 0 {
		for i := range fused {
			fused[i] /= totalWeight
		}
	}

	return fused
}

func (m *MultimodalFusion) Process(ctx context.Context, input *MultimodalInput) (*FusionResult, error) {
	if !m.initialized {
		return nil, fmt.Errorf("multimodal fusion not initialized")
	}

	encoded, err := m.EncodeModalFeatures(input)
	if err != nil {
		return nil, err
	}

	aligned := m.AlignFeatures(encoded)

	confidences := m.estimateModalConfidences(input)
	weights := m.CalculateDynamicWeights(aligned, confidences)

	for i := 0; i < m.config.NumLayers; i++ {
		aligned = m.applyFusionLayer(aligned, i)
	}

	fused := m.Fuse(aligned, weights)

	attentionMaps := m.computeAttentionMaps(aligned)
	modalScores := m.computeModalScores(aligned, weights)

	result := &FusionResult{
		FusedEmbedding: fused,
		AttentionMaps:  attentionMaps,
		ModalScores:    modalScores,
		Contribution:   weights,
		Confidence:    m.calculateOverallConfidence(modalScores, weights),
		ProcessedAt:   time.Now(),
	}

	return result, nil
}

func (m *MultimodalFusion) applyFusionLayer(modalEmbeddings map[ModalType][]float64, layerIdx int) map[ModalType][]float64 {
	if layerIdx >= len(m.fusion_layers) {
		return modalEmbeddings
	}

	layer := m.fusion_layers[layerIdx]
	updated := make(map[ModalType][]float64)

	for modal1, emb1 := range modalEmbeddings {
		crossAttnOutput := emb1

		for modal2, emb2 := range modalEmbeddings {
			if modal1 != modal2 {
				crossAttnOutput = m.crossModalAttentionSingle(crossAttnOutput, emb2, layer.CrossAttention)
			}
		}

		selfAttnOutput := m.selfAttention(crossAttnOutput, layer.SelfAttention)
		ffnOutput := m.feedForward(selfAttnOutput, layer.FFN)

		updated[modal1] = ffnOutput
	}

	return updated
}

func (m *MultimodalFusion) crossModalAttentionSingle(query, context []float64, attn *CrossModalAttention) []float64 {
	queryTrans := make([]float64, m.config.EmbeddingDim)
	keyTrans := make([]float64, m.config.EmbeddingDim)
	valTrans := make([]float64, m.config.EmbeddingDim)

	for i := range queryTrans {
		for j := 0; j < len(query) && j < len(attn.QueryWeights); j++ {
			if i < len(attn.QueryWeights[j]) {
				queryTrans[i] += query[j] * attn.QueryWeights[j][i]
				keyTrans[i] += context[j] * attn.KeyWeights[j][i]
				valTrans[i] += context[j] * attn.ValueWeights[j][i]
			}
		}
	}

	score := dotProduct(queryTrans, keyTrans) / math.Sqrt(float64(m.config.EmbeddingDim))
	attnWeight := math.Tanh(score)

	output := make([]float64, m.config.EmbeddingDim)
	for i := range output {
		output[i] = valTrans[i] * attnWeight
	}

	return output
}

func (m *MultimodalFusion) selfAttention(input []float64, attn *CrossModalAttention) []float64 {
	query := make([]float64, m.config.EmbeddingDim)
	key := make([]float64, m.config.EmbeddingDim)
	value := make([]float64, m.config.EmbeddingDim)

	for i := range query {
		for j := 0; j < len(input) && j < len(attn.QueryWeights); j++ {
			if i < len(attn.QueryWeights[j]) {
				query[i] += input[j] * attn.QueryWeights[j][i]
				key[i] += input[j] * attn.KeyWeights[j][i]
				value[i] += input[j] * attn.ValueWeights[j][i]
			}
		}
	}

	score := dotProduct(query, key) / math.Sqrt(float64(m.config.EmbeddingDim))
	attnWeight := softmax([]float64{score})[0]

	output := make([]float64, m.config.EmbeddingDim)
	for i := range output {
		output[i] = value[i] * attnWeight
	}

	return output
}

func (m *MultimodalFusion) feedForward(input []float64, ffn *FeedForward) []float64 {
	hidden := make([]float64, len(ffn.Bias1))
	for i := range hidden {
		for j := 0; j < len(input) && j < len(ffn.Weights1); j++ {
			if i < len(ffn.Weights1[j]) {
				hidden[i] += input[j] * ffn.Weights1[j][i]
			}
		}
		hidden[i] += ffn.Bias1[i]
	}
	hidden = relu(hidden)

	output := make([]float64, len(ffn.Bias2))
	for i := range output {
		for j := 0; j < len(hidden) && j < len(ffn.Weights2); j++ {
			if i < len(ffn.Weights2[j]) {
				output[i] += hidden[j] * ffn.Weights2[j][i]
			}
		}
		output[i] += ffn.Bias2[i]
	}

	return output
}

func (m *MultimodalFusion) estimateModalConfidences(input *MultimodalInput) map[ModalType]float64 {
	confidences := make(map[ModalType]float64)

	if input.Behavioral != nil {
		confidences[ModalTypeBehavioral] = 0.9
	}

	if input.Text != nil {
		confidences[ModalTypeText] = 0.8
	}

	if input.Image != nil {
		confidences[ModalTypeImage] = 0.85
	}

	if input.Audio != nil {
		confidences[ModalTypeAudio] = 0.75
	}

	if input.Sensor != nil {
		confidences[ModalTypeSensor] = 0.7
	}

	return confidences
}

func (m *MultimodalFusion) computeAttentionMaps(modalEmbeddings map[ModalType][]float64) map[ModalType][][]float64 {
	maps := make(map[ModalType][][]float64)

	for modal, embedding := range modalEmbeddings {
		attention := make([][]float64, m.config.NumHeads)
		for h := 0; h < m.config.NumHeads; h++ {
			attention[h] = make([]float64, len(embedding))
			norm := 0.0
			for i, val := range embedding {
				attention[h][i] = math.Abs(val)
				norm += attention[h][i]
			}
			if norm > 0 {
				for i := range attention[h] {
					attention[h][i] /= norm
				}
			}
		}
		maps[modal] = attention
	}

	return maps
}

func (m *MultimodalFusion) computeModalScores(modalEmbeddings map[ModalType][]float64, weights map[ModalType]float64) map[ModalType]float64 {
	scores := make(map[ModalType]float64)

	for modal, embedding := range modalEmbeddings {
		quality := m.calculateEmbeddingQuality(embedding)
		weight := 0.2
		if w, ok := weights[modal]; ok {
			weight = w
		}
		scores[modal] = quality * weight
	}

	return scores
}

func (m *MultimodalFusion) calculateOverallConfidence(modalScores map[ModalType]float64, weights map[ModalType]float64) float64 {
	if len(modalScores) == 0 {
		return 0.0
	}

	totalScore := 0.0
	totalWeight := 0.0

	for modal, score := range modalScores {
		weight := 0.2
		if w, ok := weights[modal]; ok {
			weight = w
		}
		totalScore += score * weight
		totalWeight += weight
	}

	if totalWeight > 0 {
		return totalScore / totalWeight
	}
	return 0.5
}

type MultimodalFusionService struct {
	fusion       *MultimodalFusion
	history      []*FusionResult
	mu           sync.RWMutex
}

func NewMultimodalFusionService(config *MultimodalConfig) *MultimodalFusionService {
	return &MultimodalFusionService{
		fusion:  NewMultimodalFusion(config),
		history: make([]*FusionResult, 0),
	}
}

func (s *MultimodalFusionService) Initialize(ctx context.Context) error {
	return s.fusion.Initialize(ctx)
}

func (s *MultimodalFusionService) FuseMultimodalInput(ctx context.Context, input *MultimodalInput) (*FusionResult, error) {
	result, err := s.fusion.Process(ctx, input)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.history = append(s.history, result)
	if len(s.history) > 1000 {
		s.history = s.history[len(s.history)-1000:]
	}
	s.mu.Unlock()

	return result, nil
}

func (s *MultimodalFusionService) GetHistory(limit int) []*FusionResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit >= len(s.history) {
		result := make([]*FusionResult, len(s.history))
		copy(result, s.history)
		return result
	}

	result := make([]*FusionResult, limit)
	copy(result, s.history[len(s.history)-limit:])
	return result
}

func (s *MultimodalFusionService) GetStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"history_size": len(s.history),
	}

	if len(s.history) > 0 {
		totalConfidence := 0.0
		for _, result := range s.history {
			totalConfidence += result.Confidence
		}
		stats["avg_confidence"] = totalConfidence / float64(len(s.history))

		modalUsage := make(map[ModalType]int)
		for _, result := range s.history {
			for modal := range result.Contribution {
				modalUsage[modal]++
			}
		}
		stats["modal_usage"] = modalUsage
	}

	return stats
}

func (s *MultimodalFusionService) GetCurrentWeights() map[ModalType]float64 {
	return s.fusion.dynamic_weights
}

func (s *MultimodalFusionService) ResetWeights() {
	s.fusion.mu.Lock()
	defer s.fusion.mu.Unlock()

	for modal := range s.fusion.dynamic_weights {
		s.fusion.dynamic_weights[modal] = 1.0 / float64(len(s.fusion.dynamic_weights))
	}
}
