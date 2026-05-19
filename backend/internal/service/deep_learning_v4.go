package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
)

// ============================================
// Transformer-XL 实现
// ============================================

type TransformerXL struct {
	mu          sync.RWMutex
	initialized bool
	dModel      int
	nLayers     int
	nHeads      int
	dFF         int
	dropout     float64
	layers      []*TransformerXLLayer
	memories    [][]float64
	memoryLen   int
	maxSeqLen   int
}

type TransformerXLLayer struct {
	multiHeadAttention *MultiHeadAttention
	feedForward        *FeedForwardNetwork
	layerNorm1         *LayerNorm
	layerNorm2         *LayerNorm
}

type MultiHeadAttention struct {
	dModel int
	nHeads int
	dKey   int
	wQ     []float64
	wK     []float64
	wV     []float64
	wO     []float64
}

type FeedForwardNetwork struct {
	dModel int
	dFF    int
	w1     []float64
	b1     []float64
	w2     []float64
	b2     []float64
}

type LayerNorm struct {
	dModel int
	gamma  []float64
	beta   []float64
	eps    float64
}

func NewTransformerXL(dModel, nLayers, nHeads, dFF, memoryLen int) *TransformerXL {
	layers := make([]*TransformerXLLayer, nLayers)
	for i := 0; i < nLayers; i++ {
		layers[i] = NewTransformerXLLayer(dModel, nHeads, dFF)
	}
	memories := make([][]float64, nLayers)
	for i := 0; i < nLayers; i++ {
		memories[i] = make([]float64, 0, memoryLen*dModel)
	}
	return &TransformerXL{
		dModel:    dModel,
		nLayers:   nLayers,
		nHeads:    nHeads,
		dFF:       dFF,
		memoryLen: memoryLen,
		maxSeqLen: 1024,
		layers:    layers,
		memories:  memories,
	}
}

func NewTransformerXLLayer(dModel, nHeads, dFF int) *TransformerXLLayer {
	return &TransformerXLLayer{
		multiHeadAttention: NewMultiHeadAttention(dModel, nHeads),
		feedForward:        NewFeedForwardNetwork(dModel, dFF),
		layerNorm1:         NewLayerNorm(dModel),
		layerNorm2:         NewLayerNorm(dModel),
	}
}

func NewMultiHeadAttention(dModel, nHeads int) *MultiHeadAttention {
	dKey := dModel / nHeads
	size := dModel * dKey
	return &MultiHeadAttention{
		dModel: dModel,
		nHeads: nHeads,
		dKey:   dKey,
		wQ:     randomMatrix(size),
		wK:     randomMatrix(size),
		wV:     randomMatrix(size),
		wO:     randomMatrix(size),
	}
}

func NewFeedForwardNetwork(dModel, dFF int) *FeedForwardNetwork {
	return &FeedForwardNetwork{
		dModel: dModel,
		dFF:    dFF,
		w1:     randomMatrix(dModel * dFF),
		b1:     randomVector(dFF),
		w2:     randomMatrix(dFF * dModel),
		b2:     randomVector(dModel),
	}
}

func NewLayerNorm(dModel int) *LayerNorm {
	return &LayerNorm{
		dModel: dModel,
		gamma:  make([]float64, dModel),
		beta:   make([]float64, dModel),
		eps:    1e-6,
	}
}

func randomMatrix(size int) []float64 {
	mat := make([]float64, size)
	scale := math.Sqrt(2.0 / float64(size))
	for i := range mat {
		mat[i] = (rand.Float64() - 0.5) * scale
	}
	return mat
}

func randomVector(size int) []float64 {
	vec := make([]float64, size)
	for i := range vec {
		vec[i] = rand.Float64()*0.2 - 0.1
	}
	return vec
}

func (t *TransformerXL) Initialize(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.initialized {
		return nil
	}
	t.initialized = true
	return nil
}

func (t *TransformerXL) Forward(ctx context.Context, x []float64, seqLen int) ([]float64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.initialized {
		return nil, fmt.Errorf("transformer-xl not initialized")
	}
	if len(x) != seqLen*t.dModel {
		return nil, fmt.Errorf("input size mismatch")
	}

	h := make([]float64, len(x))
	copy(h, x)

	for i, layer := range t.layers {
		var memory []float64
		if len(t.memories[i]) > 0 {
			memory = t.memories[i]
		}
		h = layer.Forward(h, memory, seqLen, t.dModel)
		t.memories[i] = t.updateMemory(t.memories[i], h, seqLen)
	}
	return h, nil
}

func (t *TransformerXL) updateMemory(oldMemory, newHidden []float64, seqLen int) []float64 {
	totalLen := len(oldMemory)/t.dModel + seqLen
	if totalLen > t.memoryLen {
		keepLen := t.memoryLen - seqLen
		if keepLen <= 0 {
			return newHidden
		}
		startIdx := (len(oldMemory)/t.dModel - keepLen) * t.dModel
		oldMemory = oldMemory[startIdx:]
	}
	newMemory := make([]float64, len(oldMemory)+len(newHidden))
	copy(newMemory, oldMemory)
	copy(newMemory[len(oldMemory):], newHidden)
	return newMemory
}

func (l *TransformerXLLayer) Forward(x, memory []float64, seqLen, dModel int) []float64 {
	residual := make([]float64, len(x))
	copy(residual, x)

	normalized := l.layerNorm1.Forward(x)
	attnOutput := l.multiHeadAttention.Forward(normalized, memory, seqLen, dModel)

	for i := range x {
		x[i] = residual[i] + attnOutput[i]
	}

	residual = make([]float64, len(x))
	copy(residual, x)

	normalized = l.layerNorm2.Forward(x)
	ffOutput := l.feedForward.Forward(normalized)

	for i := range x {
		x[i] = residual[i] + ffOutput[i]
	}
	return x
}

func (mha *MultiHeadAttention) Forward(x, memory []float64, seqLen, dModel int) []float64 {
	totalLen := seqLen
	if len(memory) > 0 {
		totalLen += len(memory) / dModel
	}

	q := mha.linear(x, mha.wQ, dModel, mha.dKey*mha.nHeads)
	var k, v []float64
	if len(memory) > 0 {
		combined := make([]float64, len(memory)+len(x))
		copy(combined, memory)
		copy(combined[len(memory):], x)
		k = mha.linear(combined, mha.wK, dModel, mha.dKey*mha.nHeads)
		v = mha.linear(combined, mha.wV, dModel, mha.dKey*mha.nHeads)
	} else {
		k = mha.linear(x, mha.wK, dModel, mha.dKey*mha.nHeads)
		v = mha.linear(x, mha.wV, dModel, mha.dKey*mha.nHeads)
	}

	headsQ := mha.splitHeads(q, seqLen)
	headsK := mha.splitHeads(k, totalLen)
	headsV := mha.splitHeads(v, totalLen)

	attnScores := mha.scaledDotProductAttention(headsQ, headsK, seqLen, totalLen)
	weightedValues := mha.weightedSum(headsV, attnScores, seqLen, totalLen)
	concatenated := mha.concatenateHeads(weightedValues, seqLen)

	return mha.linear(concatenated, mha.wO, mha.nHeads*mha.dKey, dModel)
}

func (mha *MultiHeadAttention) linear(x, weights []float64, inDim, outDim int) []float64 {
	seqLen := len(x) / inDim
	result := make([]float64, seqLen*outDim)
	for i := 0; i < seqLen; i++ {
		for j := 0; j < outDim; j++ {
			sum := 0.0
			for k := 0; k < inDim; k++ {
				sum += x[i*inDim+k] * weights[k*outDim+j]
			}
			result[i*outDim+j] = sum
		}
	}
	return result
}

func (mha *MultiHeadAttention) splitHeads(x []float64, seqLen int) [][]float64 {
	heads := make([][]float64, mha.nHeads)
	dKey := mha.dKey
	for h := 0; h < mha.nHeads; h++ {
		heads[h] = make([]float64, seqLen*dKey)
		for i := 0; i < seqLen; i++ {
			for j := 0; j < dKey; j++ {
				heads[h][i*dKey+j] = x[i*(mha.nHeads*dKey)+h*dKey+j]
			}
		}
	}
	return heads
}

func (mha *MultiHeadAttention) scaledDotProductAttention(headsQ, headsK [][]float64, qLen, kLen int) [][]float64 {
	dKey := mha.dKey
	attnScores := make([][]float64, mha.nHeads)
	for h := 0; h < mha.nHeads; h++ {
		attnScores[h] = make([]float64, qLen*kLen)
		for i := 0; i < qLen; i++ {
			for j := 0; j < kLen; j++ {
				sum := 0.0
				for k := 0; k < dKey; k++ {
					sum += headsQ[h][i*dKey+k] * headsK[h][j*dKey+k]
				}
				attnScores[h][i*kLen+j] = sum / math.Sqrt(float64(dKey))
			}
			attnScores[h] = softmax(attnScores[h], i*kLen, kLen)
		}
	}
	return attnScores
}

func softmax(x []float64, start, length int) []float64 {
	maxVal := x[start]
	for i := start + 1; i < start+length; i++ {
		if x[i] > maxVal {
			maxVal = x[i]
		}
	}
	sum := 0.0
	for i := start; i < start+length; i++ {
		x[i] = math.Exp(x[i] - maxVal)
		sum += x[i]
	}
	for i := start; i < start+length; i++ {
		x[i] /= sum
	}
	return x
}

func (mha *MultiHeadAttention) weightedSum(headsV, attnScores [][]float64, qLen, kLen int) [][]float64 {
	dKey := mha.dKey
	output := make([][]float64, mha.nHeads)
	for h := 0; h < mha.nHeads; h++ {
		output[h] = make([]float64, qLen*dKey)
		for i := 0; i < qLen; i++ {
			for j := 0; j < dKey; j++ {
				sum := 0.0
				for k := 0; k < kLen; k++ {
					sum += attnScores[h][i*kLen+k] * headsV[h][k*dKey+j]
				}
				output[h][i*dKey+j] = sum
			}
		}
	}
	return output
}

func (mha *MultiHeadAttention) concatenateHeads(heads [][]float64, seqLen int) []float64 {
	dKey := mha.dKey
	result := make([]float64, seqLen*mha.nHeads*dKey)
	for i := 0; i < seqLen; i++ {
		for h := 0; h < mha.nHeads; h++ {
			for j := 0; j < dKey; j++ {
				result[i*(mha.nHeads*dKey)+h*dKey+j] = heads[h][i*dKey+j]
			}
		}
	}
	return result
}

func (ff *FeedForwardNetwork) Forward(x []float64) []float64 {
	seqLen := len(x) / ff.dModel
	hidden := make([]float64, seqLen*ff.dFF)
	for i := 0; i < seqLen; i++ {
		for j := 0; j < ff.dFF; j++ {
			sum := 0.0
			for k := 0; k < ff.dModel; k++ {
				sum += x[i*ff.dModel+k] * ff.w1[k*ff.dFF+j]
			}
			hidden[i*ff.dFF+j] = math.Max(0, sum+ff.b1[j])
		}
	}
	output := make([]float64, len(x))
	for i := 0; i < seqLen; i++ {
		for j := 0; j < ff.dModel; j++ {
			sum := 0.0
			for k := 0; k < ff.dFF; k++ {
				sum += hidden[i*ff.dFF+k] * ff.w2[k*ff.dModel+j]
			}
			output[i*ff.dModel+j] = sum + ff.b2[j]
		}
	}
	return output
}

func (ln *LayerNorm) Forward(x []float64) []float64 {
	seqLen := len(x) / ln.dModel
	result := make([]float64, len(x))
	for i := 0; i < seqLen; i++ {
		mean := 0.0
		for j := 0; j < ln.dModel; j++ {
			mean += x[i*ln.dModel+j]
		}
		mean /= float64(ln.dModel)
		variance := 0.0
		for j := 0; j < ln.dModel; j++ {
			diff := x[i*ln.dModel+j] - mean
			variance += diff * diff
		}
		variance /= float64(ln.dModel)
		for j := 0; j < ln.dModel; j++ {
			normalized := (x[i*ln.dModel+j] - mean) / math.Sqrt(variance+ln.eps)
			result[i*ln.dModel+j] = ln.gamma[j]*normalized + ln.beta[j]
		}
	}
	return result
}

func (t *TransformerXL) ResetMemory() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := range t.memories {
		t.memories[i] = make([]float64, 0, t.memoryLen*t.dModel)
	}
}

// ============================================
// 图神经网络 (GNN) 实现
// ============================================

type GraphNeuralNetwork struct {
	mu          sync.RWMutex
	initialized bool
	inputDim    int
	hiddenDim   int
	outputDim   int
	nLayers     int
	layers      []*GNNLayer
	activation  func(float64) float64
}

type GNNLayer struct {
	inputDim  int
	outputDim int
	w         []float64
	a         []float64
	b         []float64
}

func NewGraphNeuralNetwork(inputDim, hiddenDim, outputDim, nLayers int) *GraphNeuralNetwork {
	layers := make([]*GNNLayer, nLayers)
	layers[0] = NewGNNLayer(inputDim, hiddenDim)
	for i := 1; i < nLayers-1; i++ {
		layers[i] = NewGNNLayer(hiddenDim, hiddenDim)
	}
	layers[nLayers-1] = NewGNNLayer(hiddenDim, outputDim)
	return &GraphNeuralNetwork{
		inputDim:  inputDim,
		hiddenDim: hiddenDim,
		outputDim: outputDim,
		nLayers:   nLayers,
		layers:    layers,
		activation: func(x float64) float64 {
			return math.Max(0, x)
		},
	}
}

func NewGNNLayer(inputDim, outputDim int) *GNNLayer {
	return &GNNLayer{
		inputDim:  inputDim,
		outputDim: outputDim,
		w:         randomMatrix(inputDim * outputDim),
		a:         randomMatrix(inputDim * outputDim),
		b:         randomVector(outputDim),
	}
}

func (gnn *GraphNeuralNetwork) Initialize(ctx context.Context) error {
	gnn.mu.Lock()
	defer gnn.mu.Unlock()
	if gnn.initialized {
		return nil
	}
	gnn.initialized = true
	return nil
}

func (gnn *GraphNeuralNetwork) Forward(ctx context.Context, nodeFeatures [][]float64, adjacencyMatrix [][]float64) ([][]float64, error) {
	gnn.mu.RLock()
	defer gnn.mu.RUnlock()
	if !gnn.initialized {
		return nil, fmt.Errorf("gnn not initialized")
	}

	h := make([][]float64, len(nodeFeatures))
	for i := range nodeFeatures {
		h[i] = make([]float64, len(nodeFeatures[i]))
		copy(h[i], nodeFeatures[i])
	}

	for l, layer := range gnn.layers {
		h = layer.Forward(h, adjacencyMatrix, gnn.activation)
		if l < gnn.nLayers-1 {
			for i := range h {
				for j := range h[i] {
					h[i][j] = gnn.activation(h[i][j])
				}
			}
		}
	}
	return h, nil
}

func (layer *GNNLayer) Forward(nodeFeatures [][]float64, adjacencyMatrix [][]float64, activation func(float64) float64) [][]float64 {
	nNodes := len(nodeFeatures)
	output := make([][]float64, nNodes)
	for i := 0; i < nNodes; i++ {
		output[i] = make([]float64, layer.outputDim)
	}

	for i := 0; i < nNodes; i++ {
		neighborSum := make([]float64, layer.inputDim)
		degree := 0.0
		for j := 0; j < nNodes; j++ {
			if adjacencyMatrix[i][j] > 0 {
				for k := 0; k < layer.inputDim; k++ {
					neighborSum[k] += nodeFeatures[j][k] * adjacencyMatrix[i][j]
				}
				degree += adjacencyMatrix[i][j]
			}
		}
		if degree > 0 {
			for k := range neighborSum {
				neighborSum[k] /= math.Sqrt(degree + 1)
			}
		}
		for j := 0; j < layer.outputDim; j++ {
			selfTerm := 0.0
			for k := 0; k < layer.inputDim; k++ {
				selfTerm += nodeFeatures[i][k] * layer.w[k*layer.outputDim+j]
			}
			neighborTerm := 0.0
			for k := 0; k < layer.inputDim; k++ {
				neighborTerm += neighborSum[k] * layer.a[k*layer.outputDim+j]
			}
			output[i][j] = selfTerm + neighborTerm + layer.b[j]
		}
	}
	return output
}

func (gnn *GraphNeuralNetwork) GraphPooling(nodeFeatures [][]float64, poolType string) []float64 {
	if len(nodeFeatures) == 0 {
		return []float64{}
	}
	dim := len(nodeFeatures[0])
	result := make([]float64, dim)
	switch poolType {
	case "sum":
		for i := range nodeFeatures {
			for j := range nodeFeatures[i] {
				result[j] += nodeFeatures[i][j]
			}
		}
	case "mean":
		for i := range nodeFeatures {
			for j := range nodeFeatures[i] {
				result[j] += nodeFeatures[i][j]
			}
		}
		for j := range result {
			result[j] /= float64(len(nodeFeatures))
		}
	case "max":
		for j := range result {
			result[j] = nodeFeatures[0][j]
			for i := 1; i < len(nodeFeatures); i++ {
				if nodeFeatures[i][j] > result[j] {
					result[j] = nodeFeatures[i][j]
				}
			}
		}
	}
	return result
}

// ============================================
// 增强注意力机制
// ============================================

type EnhancedAttention struct {
	mu                  sync.RWMutex
	initialized         bool
	dModel              int
	nHeads              int
	attentionTypes      []string
	gateLayers          []*FeedForwardNetwork
}

func NewEnhancedAttention(dModel, nHeads int) *EnhancedAttention {
	attentionTypes := []string{"self", "cross", "local", "global"}
	gateLayers := make([]*FeedForwardNetwork, len(attentionTypes))
	for i := range gateLayers {
		gateLayers[i] = NewFeedForwardNetwork(dModel, dModel*2)
	}
	return &EnhancedAttention{
		dModel:         dModel,
		nHeads:         nHeads,
		attentionTypes: attentionTypes,
		gateLayers:     gateLayers,
	}
}

func (ea *EnhancedAttention) Initialize(ctx context.Context) error {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	if ea.initialized {
		return nil
	}
	ea.initialized = true
	return nil
}

func (ea *EnhancedAttention) MultiTypeAttention(ctx context.Context, q, k, v []float64, seqLen int, attnType string) ([]float64, error) {
	ea.mu.RLock()
	defer ea.mu.RUnlock()
	if !ea.initialized {
		return nil, fmt.Errorf("enhanced attention not initialized")
	}

	var output []float64
	switch attnType {
	case "self":
		mha := NewMultiHeadAttention(ea.dModel, ea.nHeads)
		output = mha.Forward(q, nil, seqLen, ea.dModel)
	case "cross":
		output = ea.crossAttention(q, k, v, seqLen)
	case "local":
		output = ea.localAttention(q, k, v, seqLen, 32)
	case "global":
		output = ea.globalAttention(q, k, v, seqLen)
	default:
		return nil, fmt.Errorf("unknown attention type: %s", attnType)
	}
	return output, nil
}

func (ea *EnhancedAttention) crossAttention(q, k, v []float64, seqLen int) []float64 {
	mha := NewMultiHeadAttention(ea.dModel, ea.nHeads)
	return mha.Forward(q, append(k, v...), seqLen, ea.dModel)
}

func (ea *EnhancedAttention) localAttention(q, k, v []float64, seqLen, windowSize int) []float64 {
	output := make([]float64, len(q))
	dModel := ea.dModel
	for i := 0; i < seqLen; i++ {
		start := max(0, i-windowSize)
		end := min(seqLen-1, i+windowSize)
		localK := make([]float64, 0, (end-start+1)*dModel)
		localV := make([]float64, 0, (end-start+1)*dModel)
		for j := start; j <= end; j++ {
			localK = append(localK, k[j*dModel:(j+1)*dModel]...)
			localV = append(localV, v[j*dModel:(j+1)*dModel]...)
		}
		mha := NewMultiHeadAttention(dModel, ea.nHeads)
		singleQ := q[i*dModel : (i+1)*dModel]
		result := mha.Forward(singleQ, append(localK, localV...), 1, dModel)
		copy(output[i*dModel:(i+1)*dModel], result)
	}
	return output
}

func (ea *EnhancedAttention) globalAttention(q, k, v []float64, seqLen int) []float64 {
	mha := NewMultiHeadAttention(ea.dModel, ea.nHeads)
	return mha.Forward(q, k, seqLen, ea.dModel)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (ea *EnhancedAttention) GatedAttentionFusion(ctx context.Context, attentionOutputs [][]float64, seqLen int) ([]float64, error) {
	ea.mu.RLock()
	defer ea.mu.RUnlock()
	if !ea.initialized {
		return nil, fmt.Errorf("enhanced attention not initialized")
	}
	if len(attentionOutputs) == 0 {
		return nil, fmt.Errorf("no attention outputs to fuse")
	}

	fused := make([]float64, len(attentionOutputs[0]))
	gates := make([][]float64, len(attentionOutputs))
	for i, output := range attentionOutputs {
		gated := ea.gateLayers[i].Forward(output)
		gates[i] = make([]float64, seqLen)
		for j := 0; j < seqLen; j++ {
			gates[i][j] = sigmoid(gated[j*ea.dModel])
		}
	}

	for i := 0; i < seqLen; i++ {
		sumGate := 0.0
		for _, g := range gates {
			sumGate += g[i]
		}
		for j := 0; j < ea.dModel; j++ {
			for idx, output := range attentionOutputs {
				weight := gates[idx][i] / (sumGate + 1e-8)
				fused[i*ea.dModel+j] += weight * output[i*ea.dModel+j]
			}
		}
	}
	return fused, nil
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// ============================================
// AI模型v4 主服务
// ============================================

type AIModelV4Service struct {
	mu                 sync.RWMutex
	initialized        bool
	transformerXL      *TransformerXL
	gnn                *GraphNeuralNetwork
	enhancedAttention  *EnhancedAttention
}

func NewAIModelV4Service() *AIModelV4Service {
	return &AIModelV4Service{
		transformerXL:     NewTransformerXL(512, 6, 8, 2048, 128),
		gnn:               NewGraphNeuralNetwork(64, 128, 64, 4),
		enhancedAttention: NewEnhancedAttention(512, 8),
	}
}

func (s *AIModelV4Service) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.initialized {
		return nil
	}
	if err := s.transformerXL.Initialize(ctx); err != nil {
		return err
	}
	if err := s.gnn.Initialize(ctx); err != nil {
		return err
	}
	if err := s.enhancedAttention.Initialize(ctx); err != nil {
		return err
	}
	s.initialized = true
	return nil
}

func (s *AIModelV4Service) ProcessSequence(ctx context.Context, sequence []float64, seqLen int) ([]float64, error) {
	return s.transformerXL.Forward(ctx, sequence, seqLen)
}

func (s *AIModelV4Service) ProcessGraph(ctx context.Context, nodeFeatures [][]float64, adjacencyMatrix [][]float64) ([][]float64, error) {
	return s.gnn.Forward(ctx, nodeFeatures, adjacencyMatrix)
}

func (s *AIModelV4Service) MultiAttention(ctx context.Context, q, k, v []float64, seqLen int) ([]float64, error) {
	outputs := make([][]float64, 4)
	var err error
	for i, attnType := range []string{"self", "cross", "local", "global"} {
		outputs[i], err = s.enhancedAttention.MultiTypeAttention(ctx, q, k, v, seqLen, attnType)
		if err != nil {
			return nil, err
		}
	}
	return s.enhancedAttention.GatedAttentionFusion(ctx, outputs, seqLen)
}

func (s *AIModelV4Service) ResetTransformerMemory() {
	s.transformerXL.ResetMemory()
}
