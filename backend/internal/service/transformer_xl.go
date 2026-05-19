package service

import (
	"context"
	"fmt"
	"math"
	"sync"
)

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
