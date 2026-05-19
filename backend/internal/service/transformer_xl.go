package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

var mathRand = rand.New(rand.NewSource(time.Now().UnixNano()))

const (
	TransformerXLModelDim     = 512
	TransformerXLNumHeads    = 8
	TransformerXLNumLayers  = 6
	TransformerXLMemoryLen   = 512
	TransformerXLMaxSeqLen   = 2048
	TransformerXLFFDim       = 2048
	TransformerXLDropout     = 0.1
)

type TransformerXLConfig struct {
	ModelDim      int
	NumHeads      int
	NumLayers     int
	MemoryLen     int
	MaxSeqLen     int
	FFDim         int
	Dropout       float64
	LearningRate  float64
}

type TransformerXL struct {
	config            TransformerXLConfig
	queryWeights      [][][]float64
	keyWeights        [][][]float64
	valueWeights      [][][]float64
	outputWeights     [][][]float64
	ffWeights         [][][]float64
	ffBiases          [][]float64
	relativePosBias   [][]float64
	memory            [][]float64
	posEmbedding      []float64
	segmentEmbedding  []float64
	layerNorms        [][]float64
	mu                sync.RWMutex
	initialized       bool
	sequenceLength    int
	memoryLength      int
}

type TransformerXLInput struct {
	TokenEmbeddings []float64
	PositionIDs     []int
	SegmentIDs      []int
	Mask            [][]bool
}

type TransformerXLOutput struct {
	Logits          []float64
	AttentionScores [][][]float64
	HiddenStates    [][]float64
	MemoryUsage     int
	PredictiveScore float64
	Confidence      float64
}

type BehaviorSequence struct {
	UserID         string
	SessionID      string
	SequenceID     string
	Timesteps      []int64
	BehaviorVecs   [][]float64
	Labels         []bool
	SegmentIDs     []int
	Metadata       map[string]interface{}
}

type XLMemory struct {
	HiddenStates [][]float64
	KeyStates    [][]float64
	ValueStates  [][]float64
	SegmentIDs   []int
	Timestamps   []int64
	Length       int
	Capacity     int
}

func NewTransformerXL(config *TransformerXLConfig) *TransformerXL {
	if config == nil {
		config = &TransformerXLConfig{
			ModelDim:     TransformerXLModelDim,
			NumHeads:     TransformerXLNumHeads,
			NumLayers:    TransformerXLNumLayers,
			MemoryLen:    TransformerXLMemoryLen,
			MaxSeqLen:    TransformerXLMaxSeqLen,
			FFDim:        TransformerXLFFDim,
			Dropout:      TransformerXLDropout,
			LearningRate: 0.0001,
		}
	}

	txl := &TransformerXL{
		config:            *config,
		queryWeights:      make([][][]float64, config.NumLayers),
		keyWeights:        make([][][]float64, config.NumLayers),
		valueWeights:      make([][][]float64, config.NumLayers),
		outputWeights:     make([][][]float64, config.NumLayers),
		ffWeights:         make([][][]float64, config.NumLayers),
		ffBiases:          make([][]float64, config.NumLayers),
		relativePosBias:   make([][]float64, config.NumLayers),
		memory:            make([][]float64, config.MemoryLen),
		posEmbedding:      make([]float64, config.ModelDim),
		segmentEmbedding:  make([]float64, config.ModelDim),
		layerNorms:        make([][]float64, config.NumLayers+1),
		sequenceLength:     0,
		memoryLength:       0,
	}

	for i := 0; i < config.NumLayers; i++ {
		txl.queryWeights[i] = createRandomMatrix(config.ModelDim, config.ModelDim, 0.02)
		txl.keyWeights[i] = createRandomMatrix(config.ModelDim, config.ModelDim, 0.02)
		txl.valueWeights[i] = createRandomMatrix(config.ModelDim, config.ModelDim, 0.02)
		txl.outputWeights[i] = createRandomMatrix(config.ModelDim, config.ModelDim, 0.02)
		txl.ffWeights[i] = createRandomMatrix(config.FFDim, config.ModelDim, 0.02)
		txl.ffBiases[i] = createRandomVector(config.FFDim, 0.02)
		txl.relativePosBias[i] = createRandomVector(config.MemoryLen+config.MaxSeqLen, 0.02)
	}

	for i := 0; i <= config.NumLayers; i++ {
		txl.layerNorms[i] = createLayerNormParams(config.ModelDim)
	}

	for i := 0; i < config.MemoryLen; i++ {
		txl.memory[i] = make([]float64, config.ModelDim)
	}

	for i := range txl.posEmbedding {
		txl.posEmbedding[i] = math.Sin(float64(i)/100.0) * 0.1
	}

	txl.initialized = true
	return txl
}

func createRandomMatrix(rows, cols int, std float64) [][]float64 {
	matrix := make([][]float64, rows)
	for i := range matrix {
		matrix[i] = make([]float64, cols)
		for j := range matrix[i] {
			matrix[i][j] = gaussianRandom(0, std)
		}
	}
	return matrix
}

func createRandomVector(size int, std float64) []float64 {
	vec := make([]float64, size)
	for i := range vec {
		vec[i] = gaussianRandom(0, std)
	}
	return vec
}

func createLayerNormParams(dim int) []float64 {
	params := make([]float64, dim*2)
	for i := 0; i < dim; i++ {
		params[i] = 1.0
	}
	for i := dim; i < dim*2; i++ {
		params[i] = 0.0
	}
	return params
}

func gaussianRandom(mean, std float64) float64 {
	u1 := 1.0 - (float64)(mathRand.Intn(1000000))/1000000.0
	u2 := 1.0 - (float64)(mathRand.Intn(1000000))/1000000.0
	normal := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
	return mean + std*normal
}

func (t *TransformerXL) Initialize(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.initialized {
		return fmt.Errorf("transformer XL initialization failed")
	}
	t.initialized = true
	return nil
}

func (t *TransformerXL) Forward(input *TransformerXLInput, memory *XLMemory) *TransformerXLOutput {
	t.mu.RLock()
	defer t.mu.RUnlock()

	seqLen := len(input.TokenEmbeddings)
	if seqLen == 0 {
		return &TransformerXLOutput{
			Logits:          make([]float64, 0),
			AttentionScores: make([][][]float64, 0),
			HiddenStates:    make([][]float64, 0),
			MemoryUsage:     0,
			PredictiveScore: 0.0,
			Confidence:      0.0,
		}
	}

	hiddenStates := make([][]float64, seqLen)
	for i := range hiddenStates {
		hiddenStates[i] = make([]float64, t.config.ModelDim)
		if i < len(input.TokenEmbeddings) && len(input.TokenEmbeddings) == t.config.ModelDim {
			copy(hiddenStates[i], input.TokenEmbeddings)
		} else if len(input.TokenEmbeddings) > i {
			for j := 0; j < t.config.ModelDim && j < len(input.TokenEmbeddings); j++ {
				hiddenStates[i][j] = input.TokenEmbeddings[j]
			}
		}
	}

	var memLen int
	var memKeyStates, memValueStates [][]float64
	if memory != nil && memory.Length > 0 {
		memLen = memory.Length
		memKeyStates = memory.KeyStates
		memValueStates = memory.ValueStates
	} else {
		memLen = t.memoryLength
		if memLen > 0 {
			memKeyStates = make([][]float64, memLen)
			memValueStates = make([][]float64, memLen)
			for i := 0; i < memLen; i++ {
				memKeyStates[i] = make([]float64, t.config.ModelDim)
				memValueStates[i] = make([]float64, t.config.ModelDim)
				copy(memKeyStates[i], t.memory[i])
				copy(memValueStates[i], t.memory[i])
			}
		}
	}

	totalLen := memLen + seqLen
	attentionScores := make([][][]float64, t.config.NumLayers)
	for layer := 0; layer < t.config.NumLayers; layer++ {
		attentionScores[layer] = make([][]float64, t.config.NumHeads)
		for head := 0; head < t.config.NumHeads; head++ {
			attentionScores[layer][head] = make([]float64, totalLen*totalLen)
		}
	}

	for layer := 0; layer < t.config.NumLayers; layer++ {
		layerInput := make([][]float64, seqLen)
		for i := range layerInput {
			layerInput[i] = make([]float64, t.config.ModelDim)
			copy(layerInput[i], hiddenStates[i])
		}

		if memLen > 0 {
			layerInput = append(memValueStates, layerInput...)
		}

		headDim := t.config.ModelDim / t.config.NumHeads
		layerAttnOutput := make([][]float64, seqLen)
		for i := range layerAttnOutput {
			layerAttnOutput[i] = make([]float64, t.config.ModelDim)
		}

		for head := 0; head < t.config.NumHeads; head++ {
			headQuery := t.computeLinear(layerInput, t.queryWeights[layer], head*headDim, headDim)
			headKey := t.computeLinear(layerInput, t.keyWeights[layer], head*headDim, headDim)
			headValue := t.computeLinear(layerInput, t.valueWeights[layer], head*headDim, headDim)

		for i := 0; i < seqLen; i++ {
			qi := i + memLen
			attnScores := make([]float64, totalLen)
			for j := 0; j < totalLen; j++ {
				contentScore := dotProduct(headQuery[i], headKey[j])
				relPosScore := 0.0
				if layer < len(t.relativePosBias) && qi < len(t.relativePosBias[layer]) {
					relPosScore = t.relativePosBias[layer][qi] * t.relativePosBias[layer][j%len(t.relativePosBias[layer])]
				}
				attnScores[j] = contentScore + relPosScore
			}

				attnScores = softmax(attnScores)

				for k := 0; k < len(attnScores); k++ {
					for d := 0; d < headDim; d++ {
						layerAttnOutput[i][head*headDim+d] += attnScores[k] * headValue[k][d]
					}
				}
			}
		}

		layerAttnOutput = t.layerNorm(layerAttnOutput, t.layerNorms[layer][:t.config.ModelDim], t.layerNorms[layer][t.config.ModelDim:])

		for i := range hiddenStates {
			for j := range hiddenStates[i] {
				hiddenStates[i][j] += layerAttnOutput[i][j]
			}
		}

		ffOutput := t.feedForward(hiddenStates, layer)
		for i := range hiddenStates {
			for j := range hiddenStates[i] {
				hiddenStates[i][j] += ffOutput[i][j]
			}
		}
	}

	logits := t.computeLogits(hiddenStates)

	score := 0.0
	for i := 0; i < len(logits) && i < t.config.ModelDim; i++ {
		score += math.Abs(logits[i])
	}
	if len(logits) > 0 {
		score /= float64(len(logits))
	}

	memoryUsage := 0
	if memLen+seqLen <= t.config.MemoryLen {
		memoryUsage = memLen + seqLen
	} else {
		memoryUsage = t.config.MemoryLen
	}

	return &TransformerXLOutput{
		Logits:          logits,
		AttentionScores: attentionScores,
		HiddenStates:    hiddenStates,
		MemoryUsage:     memoryUsage,
		PredictiveScore: score,
		Confidence:      math.Min(1.0, score*2),
	}
}

func (t *TransformerXL) computeLinear(input [][]float64, weights [][]float64, offset, size int) [][]float64 {
	output := make([][]float64, len(input))
	for i := range output {
		output[i] = make([]float64, size)
		for j := 0; j < size; j++ {
			for k := range input[i] {
				if offset+j < len(weights) && k < len(weights[offset+j]) {
					output[i][j] += input[i][k] * weights[offset+j][k]
				}
			}
		}
	}
	return output
}

func (t *TransformerXL) layerNorm(input [][]float64, gamma, beta []float64) [][]float64 {
	output := make([][]float64, len(input))
	mean := make([]float64, len(input[0]))
	variance := make([]float64, len(input[0]))

	for j := range input[0] {
		sum := 0.0
		for i := range input {
			sum += input[i][j]
		}
		mean[j] = sum / float64(len(input))
	}

	for j := range input[0] {
		sum := 0.0
		for i := range input {
			diff := input[i][j] - mean[j]
			sum += diff * diff
		}
		variance[j] = sum / float64(len(input))
	}

	for i := range output {
		output[i] = make([]float64, len(input[0]))
		for j := range output[i] {
			if variance[j] > 1e-8 {
				output[i][j] = gamma[j]*(input[i][j]-mean[j])/math.Sqrt(variance[j]) + beta[j]
			} else {
				output[i][j] = input[i][j]
			}
		}
	}
	return output
}

func (t *TransformerXL) feedForward(input [][]float64, layer int) [][]float64 {
	hidden := make([][]float64, len(input))
	for i := range hidden {
		hidden[i] = make([]float64, t.config.FFDim)
		for j := 0; j < t.config.FFDim; j++ {
			for k := 0; k < len(input[i]) && k < len(t.ffWeights[layer][j]); k++ {
				hidden[i][j] += input[i][k] * t.ffWeights[layer][j][k]
			}
			if j < len(t.ffBiases[layer]) {
				hidden[i][j] += t.ffBiases[layer][j]
			}
		}
		hidden[i] = relu(hidden[i])
	}

	output := make([][]float64, len(input))
	for i := range output {
		output[i] = make([]float64, t.config.ModelDim)
		for j := 0; j < t.config.ModelDim; j++ {
			output[i][j] = hidden[i][j%len(hidden[i])]
		}
	}
	return output
}

func (t *TransformerXL) computeLogits(hidden [][]float64) []float64 {
	logits := make([]float64, t.config.ModelDim)
	if len(hidden) == 0 {
		return logits
	}
	for j := range logits {
		sum := 0.0
		for i := range hidden {
			sum += hidden[i][j]
		}
		logits[j] = sum / float64(len(hidden))
	}
	return logits
}

func dotProduct(a, b []float64) float64 {
	result := 0.0
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	for i := 0; i < minLen; i++ {
		result += a[i] * b[i]
	}
	return result
}

func softmax(values []float64) []float64 {
	maxVal := values[0]
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}

	result := make([]float64, len(values))
	sum := 0.0
	for i, v := range values {
		result[i] = math.Exp(v - maxVal)
		sum += result[i]
	}

	if sum > 0 {
		for i := range result {
			result[i] /= sum
		}
	}
	return result
}

func relu(values []float64) []float64 {
	result := make([]float64, len(values))
	for i, v := range values {
		if v > 0 {
			result[i] = v
		}
	}
	return result
}

func (t *TransformerXL) UpdateMemory(newHiddenStates [][]float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	newLen := len(newHiddenStates)
	totalLen := t.memoryLength + newLen

	if totalLen <= t.config.MemoryLen {
		for i := 0; i < newLen; i++ {
			copy(t.memory[t.memoryLength+i], newHiddenStates[i])
		}
		t.memoryLength = totalLen
	} else {
		shift := totalLen - t.config.MemoryLen
		for i := 0; i < t.memoryLength && i < t.config.MemoryLen; i++ {
			srcIdx := i + shift
			if srcIdx < t.memoryLength && srcIdx < t.config.MemoryLen {
				copy(t.memory[i], t.memory[srcIdx])
			}
		}
		for i := 0; i < newLen; i++ {
			destIdx := t.config.MemoryLen - newLen + i
			if destIdx >= 0 && destIdx < t.config.MemoryLen {
				copy(t.memory[destIdx], newHiddenStates[i])
			}
		}
		t.memoryLength = t.config.MemoryLen
	}
}

func (t *TransformerXL) GetMemory() *XLMemory {
	t.mu.RLock()
	defer t.mu.RUnlock()

	memory := &XLMemory{
		HiddenStates: make([][]float64, t.memoryLength),
		KeyStates:    make([][]float64, t.memoryLength),
		ValueStates:  make([][]float64, t.memoryLength),
		SegmentIDs:   make([]int, t.memoryLength),
		Timestamps:   make([]int64, t.memoryLength),
		Length:       t.memoryLength,
		Capacity:     t.config.MemoryLen,
	}

	for i := 0; i < t.memoryLength; i++ {
		memory.HiddenStates[i] = make([]float64, t.config.ModelDim)
		memory.KeyStates[i] = make([]float64, t.config.ModelDim)
		memory.ValueStates[i] = make([]float64, t.config.ModelDim)
		copy(memory.HiddenStates[i], t.memory[i])
		copy(memory.KeyStates[i], t.memory[i])
		copy(memory.ValueStates[i], t.memory[i])
	}

	return memory
}

func (t *TransformerXL) PredictBehavior(ctx context.Context, sequence *BehaviorSequence) (*TransformerXLOutput, error) {
	if sequence == nil || len(sequence.BehaviorVecs) == 0 {
		return nil, fmt.Errorf("empty behavior sequence")
	}

	segmentLen := len(sequence.BehaviorVecs)
	if segmentLen > t.config.MaxSeqLen {
		segmentLen = t.config.MaxSeqLen
	}

	var tokenEmbedding []float64
	if len(sequence.BehaviorVecs) > 0 && len(sequence.BehaviorVecs[0]) > 0 {
		tokenEmbedding = make([]float64, t.config.ModelDim)
		copy(tokenEmbedding, sequence.BehaviorVecs[0])
	} else {
		tokenEmbedding = make([]float64, t.config.ModelDim)
	}

	input := &TransformerXLInput{
		TokenEmbeddings: tokenEmbedding,
		PositionIDs:     make([]int, segmentLen),
		SegmentIDs:      make([]int, segmentLen),
		Mask:            make([][]bool, segmentLen),
	}

	for i := 0; i < segmentLen; i++ {
		input.PositionIDs[i] = i
		if i < len(sequence.SegmentIDs) {
			input.SegmentIDs[i] = sequence.SegmentIDs[i]
		}
		input.Mask[i] = make([]bool, segmentLen)
		for j := 0; j < segmentLen; j++ {
			input.Mask[i][j] = true
		}
	}

	memory := t.GetMemory()

	output := t.Forward(input, memory)

	if output.HiddenStates != nil && len(output.HiddenStates) > 0 {
		t.UpdateMemory(output.HiddenStates)
	}

	return output, nil
}

func (t *TransformerXL) ProcessLongSequence(ctx context.Context, sequence *BehaviorSequence, chunkSize int) ([]*TransformerXLOutput, error) {
	if sequence == nil || len(sequence.BehaviorVecs) == 0 {
		return nil, fmt.Errorf("empty behavior sequence")
	}

	if chunkSize <= 0 {
		chunkSize = t.config.MaxSeqLen
	}

	var outputs []*TransformerXLOutput
	totalLen := len(sequence.BehaviorVecs)

	for start := 0; start < totalLen; start += chunkSize {
		end := start + chunkSize
		if end > totalLen {
			end = totalLen
		}

		chunkVecs := sequence.BehaviorVecs[start:end]
		chunkSegmentIDs := make([]int, len(chunkVecs))
		if len(sequence.SegmentIDs) >= end {
			copy(chunkSegmentIDs, sequence.SegmentIDs[start:end])
		}

		chunkSeq := &BehaviorSequence{
			UserID:       sequence.UserID,
			SessionID:    sequence.SessionID,
			SequenceID:   fmt.Sprintf("%s_chunk_%d", sequence.SequenceID, start/chunkSize),
			BehaviorVecs: chunkVecs,
			SegmentIDs:   chunkSegmentIDs,
			Timesteps:    nil,
			Labels:       nil,
			Metadata:     sequence.Metadata,
		}

		output, err := t.PredictBehavior(ctx, chunkSeq)
		if err != nil {
			return outputs, err
		}
		outputs = append(outputs, output)
	}

	return outputs, nil
}

func (t *TransformerXL) DetectAnomaly(output *TransformerXLOutput) bool {
	if output == nil {
		return false
	}

	if output.PredictiveScore < 0.3 {
		return true
	}

	attentionEntropy := t.calculateAttentionEntropy(output.AttentionScores)
	if attentionEntropy < 0.1 {
		return true
	}

	return false
}

func (t *TransformerXL) calculateAttentionEntropy(scores [][][]float64) float64 {
	if len(scores) == 0 {
		return 0.0
	}

	totalEntropy := 0.0
	count := 0

	for _, layer := range scores {
		for _, head := range layer {
			if len(head) == 0 {
				continue
			}
			probs := softmax(head)
			entropy := 0.0
			for _, p := range probs {
				if p > 0 {
					entropy -= p * math.Log(p+1e-10)
				}
			}
			totalEntropy += entropy
			count++
		}
	}

	if count > 0 {
		return totalEntropy / float64(count)
	}
	return 0.0
}

type TransformerXLService struct {
	model           *TransformerXL
	sequenceHistory map[string][]*BehaviorSequence
	anomalyThreshold float64
	mu              sync.RWMutex
}

func NewTransformerXLService(config *TransformerXLConfig) *TransformerXLService {
	return &TransformerXLService{
		model:            NewTransformerXL(config),
		sequenceHistory:  make(map[string][]*BehaviorSequence),
		anomalyThreshold: 0.5,
	}
}

func (s *TransformerXLService) Initialize(ctx context.Context) error {
	return s.model.Initialize(ctx)
}

func (s *TransformerXLService) AnalyzeBehavior(ctx context.Context, userID string, traces []*model.TraceData) (*BehaviorAnalysisResult, error) {
	if len(traces) == 0 {
		return nil, fmt.Errorf("no traces provided")
	}

	sequence := s.buildBehaviorSequence(userID, traces)

	output, err := s.model.PredictBehavior(ctx, sequence)
	if err != nil {
		return nil, err
	}

	isAnomaly := s.model.DetectAnomaly(output)

	result := &BehaviorAnalysisResult{
		UserID:           userID,
		PredictiveScore:  output.PredictiveScore,
		Confidence:       output.Confidence,
		IsAnomaly:        isAnomaly,
		AnomalyReasons:   s.getAnomalyReasons(output),
		AttentionPattern: s.analyzeAttentionPattern(output.AttentionScores),
		SequenceLength:   len(traces),
		MemoryUsage:      output.MemoryUsage,
		ProcessedAt:      time.Now(),
	}

	s.addToHistory(userID, sequence)

	return result, nil
}

func (s *TransformerXLService) buildBehaviorSequence(userID string, traces []*model.TraceData) *BehaviorSequence {
	sequenceID := fmt.Sprintf("seq_%s_%d", userID, time.Now().UnixNano())
	vecs := make([][]float64, 0, len(traces))

	for _, trace := range traces {
		vec := extractFeaturesFromTrace(trace)
		vecs = append(vecs, vec)
	}

	return &BehaviorSequence{
		UserID:       userID,
		SequenceID:   sequenceID,
		BehaviorVecs: vecs,
		Timesteps:    make([]int64, len(traces)),
		Labels:       make([]bool, len(traces)),
		SegmentIDs:   make([]int, len(traces)),
		Metadata:     make(map[string]interface{}),
	}
}

func extractFeaturesFromTrace(trace *model.TraceData) []float64 {
	vec := make([]float64, TransformerXLModelDim)
	if trace == nil || len(trace.Points) == 0 {
		return vec
	}

	vec[0] = float64(len(trace.Points))
	vec[1] = float64(trace.TotalTime) / 1000.0

	totalDist := 0.0
	speedSum := 0.0
	speedCount := 0
	for i := 1; i < len(trace.Points); i++ {
		dx := trace.Points[i].X - trace.Points[i-1].X
		dy := trace.Points[i].Y - trace.Points[i-1].Y
		dist := math.Sqrt(dx*dx + dy*dy)
		totalDist += dist
		dt := float64(trace.Points[i].Timestamp - trace.Points[i-1].Timestamp)
		if dt > 0 {
			speed := dist / dt
			speedSum += speed
			speedCount++
		}
	}
	vec[2] = totalDist
	if speedCount > 0 {
		vec[3] = speedSum / float64(speedCount)
	}

	baseIdx := 10
	for i := 0; i < TransformerXLModelDim-baseIdx && i < len(trace.Points); i++ {
		vec[baseIdx+i] = trace.Points[i].X / 1000.0
		vec[baseIdx+i+TransformerXLModelDim/2] = trace.Points[i].Y / 1000.0
	}

	return vec
}

func (s *TransformerXLService) getAnomalyReasons(output *TransformerXLOutput) []string {
	var reasons []string

	if output.PredictiveScore < 0.3 {
		reasons = append(reasons, "预测分数过低")
	}

	if output.Confidence < 0.4 {
		reasons = append(reasons, "预测置信度不足")
	}

	entropy := s.model.calculateAttentionEntropy(output.AttentionScores)
	if entropy < 0.15 {
		reasons = append(reasons, "注意力模式异常")
	}

	return reasons
}

func (s *TransformerXLService) analyzeAttentionPattern(scores [][][]float64) map[string]interface{} {
	pattern := make(map[string]interface{})

	if len(scores) > 0 {
		pattern["num_layers"] = len(scores)
	}
	if len(scores) > 0 && len(scores[0]) > 0 {
		pattern["num_heads"] = len(scores[0])
	}

	avgEntropy := s.model.calculateAttentionEntropy(scores)
	pattern["attention_entropy"] = avgEntropy

	if avgEntropy > 0.5 {
		pattern["pattern_type"] = "diverse"
	} else if avgEntropy > 0.2 {
		pattern["pattern_type"] = "normal"
	} else {
		pattern["pattern_type"] = "focused"
	}

	return pattern
}

func (s *TransformerXLService) addToHistory(userID string, sequence *BehaviorSequence) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := userID
	s.sequenceHistory[key] = append(s.sequenceHistory[key], sequence)

	if len(s.sequenceHistory[key]) > 100 {
		s.sequenceHistory[key] = s.sequenceHistory[key][len(s.sequenceHistory[key])-100:]
	}
}

func (s *TransformerXLService) GetSequenceHistory(userID string) []*BehaviorSequence {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if seqs, ok := s.sequenceHistory[userID]; ok {
		result := make([]*BehaviorSequence, len(seqs))
		copy(result, seqs)
		return result
	}
	return []*BehaviorSequence{}
}

type BehaviorAnalysisResult struct {
	UserID            string
	PredictiveScore   float64
	Confidence        float64
	IsAnomaly         bool
	AnomalyReasons    []string
	AttentionPattern  map[string]interface{}
	SequenceLength    int
	MemoryUsage       int
	ProcessedAt       time.Time
}

func (s *TransformerXLService) SetAnomalyThreshold(threshold float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.anomalyThreshold = threshold
}

func (s *TransformerXLService) GetModelStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"model_dim":      s.model.config.ModelDim,
		"num_heads":       s.model.config.NumHeads,
		"num_layers":     s.model.config.NumLayers,
		"memory_len":      s.model.config.MemoryLen,
		"max_seq_len":     s.model.config.MaxSeqLen,
		"current_memory": s.model.memoryLength,
		"history_size":   len(s.sequenceHistory),
	}

	return stats
}
