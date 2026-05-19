package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
)

type EnhancedAttention struct {
	mu          sync.RWMutex
	initialized bool
	dModel      int
	nHeads      int
	attentionTypes []string
	gateLayers   []*FeedForwardNetwork
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

type SelfSupervisedPretrainer struct {
	mu          sync.RWMutex
	initialized bool
	dModel      int
	encoder     *TransformerXL
	predictor   *FeedForwardNetwork
	maskingRate float64
}

func NewSelfSupervisedPretrainer(dModel int) *SelfSupervisedPretrainer {
	return &SelfSupervisedPretrainer{
		dModel:      dModel,
		encoder:     NewTransformerXL(dModel, 4, 8, dModel*4, 64),
		predictor:   NewFeedForwardNetwork(dModel, dModel*2),
		maskingRate: 0.15,
	}
}

func (ssp *SelfSupervisedPretrainer) Initialize(ctx context.Context) error {
	ssp.mu.Lock()
	defer ssp.mu.Unlock()
	if ssp.initialized {
		return nil
	}
	if err := ssp.encoder.Initialize(ctx); err != nil {
		return err
	}
	ssp.initialized = true
	return nil
}

func (ssp *SelfSupervisedPretrainer) MaskSequence(sequence []float64, seqLen int) ([]float64, []bool) {
	masked := make([]float64, len(sequence))
	copy(masked, sequence)
	mask := make([]bool, seqLen)

	for i := 0; i < seqLen; i++ {
		if rand.Float64() < ssp.maskingRate {
			mask[i] = true
			for j := 0; j < ssp.dModel; j++ {
				masked[i*ssp.dModel+j] = 0.0
			}
		}
	}
	return masked, mask
}

func (ssp *SelfSupervisedPretrainer) PretrainStep(ctx context.Context, sequence []float64, seqLen int) ([]float64, float64, error) {
	ssp.mu.Lock()
	defer ssp.mu.Unlock()
	if !ssp.initialized {
		return nil, 0.0, fmt.Errorf("self-supervised pretrainer not initialized")
	}

	maskedSequence, mask := ssp.MaskSequence(sequence, seqLen)

	encoded, err := ssp.encoder.Forward(ctx, maskedSequence, seqLen)
	if err != nil {
		return nil, 0.0, err
	}

	predicted := ssp.predictor.Forward(encoded)

	loss := 0.0
	count := 0
	for i := 0; i < seqLen; i++ {
		if mask[i] {
			for j := 0; j < ssp.dModel; j++ {
				idx := i*ssp.dModel + j
				diff := predicted[idx] - sequence[idx]
				loss += diff * diff
				count++
			}
		}
	}
	if count > 0 {
		loss /= float64(count)
	}

	return predicted, loss, nil
}

func (ssp *SelfSupervisedPretrainer) ContrastiveLearning(ctx context.Context, sequences [][]float64, seqLen int, temperature float64) (float64, error) {
	ssp.mu.Lock()
	defer ssp.mu.Unlock()
	if !ssp.initialized {
		return 0.0, fmt.Errorf("self-supervised pretrainer not initialized")
	}

	embeddings := make([][]float64, len(sequences))
	for i, seq := range sequences {
		encoded, err := ssp.encoder.Forward(ctx, seq, seqLen)
		if err != nil {
			return 0.0, err
		}
		embeddings[i] = encoded
	}

	similarities := make([][]float64, len(embeddings))
	for i := range similarities {
		similarities[i] = make([]float64, len(embeddings))
		for j := range embeddings {
			sim := 0.0
			normI := 0.0
			normJ := 0.0
			for k := range embeddings[i] {
				sim += embeddings[i][k] * embeddings[j][k]
				normI += embeddings[i][k] * embeddings[i][k]
				normJ += embeddings[j][k] * embeddings[j][k]
			}
			similarities[i][j] = sim / (math.Sqrt(normI*normJ) + 1e-8)
		}
	}

	loss := 0.0
	for i := range embeddings {
		expSum := 0.0
		for j := range embeddings {
			if i != j {
				expSum += math.Exp(similarities[i][j] / temperature)
			}
		}
		if expSum > 0 {
			loss -= math.Log(math.Exp(similarities[i][i]/temperature) / expSum)
		}
	}
	loss /= float64(len(embeddings))

	return loss, nil
}

type AIModelV4Service struct {
	mu                    sync.RWMutex
	initialized           bool
	transformerXL         *TransformerXL
	gnn                *GraphNeuralNetwork
	enhancedAttention *EnhancedAttention
	pretrainer         *SelfSupervisedPretrainer
}

func NewAIModelV4Service() *AIModelV4Service {
	return &AIModelV4Service{
		transformerXL:     NewTransformerXL(512, 6, 8, 2048, 128),
		gnn:                NewGraphNeuralNetwork(64, 128, 64, 4),
		enhancedAttention: NewEnhancedAttention(512, 8),
		pretrainer:         NewSelfSupervisedPretrainer(512),
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
	if err := s.pretrainer.Initialize(ctx); err != nil {
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

func (s *AIModelV4Service) PretrainStep(ctx context.Context, sequence []float64, seqLen int) ([]float64, float64, error) {
	return s.pretrainer.PretrainStep(ctx, sequence, seqLen)
}

func (s *AIModelV4Service) ContrastiveLearning(ctx context.Context, sequences [][]float64, seqLen int, temperature float64) (float64, error) {
	return s.pretrainer.ContrastiveLearning(ctx, sequences, seqLen, temperature)
}
