package service

import (
	"context"
	"fmt"
	"math"
	"sync"
)

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
