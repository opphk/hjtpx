package ai

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
)

const (
	Epsilon        = 1e-8
	 XavierGain    = 1.0
)

type Layer struct {
	Weights        [][]float64
	Biases         []float64
	Inputs         []float64
	Outputs        []float64
	GradientsW     [][]float64
	GradientsB     []float64
	ActivationFunc string
}

func NewLayer(inputSize, outputSize int, activationFunc string) *Layer {
	layer := &Layer{
		Weights:        make([][]float64, outputSize),
		Biases:         make([]float64, outputSize),
		Inputs:         make([]float64, inputSize),
		Outputs:        make([]float64, outputSize),
		GradientsW:     make([][]float64, outputSize),
		GradientsB:     make([]float64, outputSize),
		ActivationFunc: activationFunc,
	}

	scale := math.Sqrt(2.0 / float64(inputSize+outputSize))
	for i := 0; i < outputSize; i++ {
		layer.Weights[i] = make([]float64, inputSize)
		layer.GradientsW[i] = make([]float64, inputSize)
		for j := 0; j < inputSize; j++ {
			layer.Weights[i][j] = (rand.Float64()*2 - 1) * scale
		}
		layer.Biases[i] = 0
	}

	return layer
}

func (l *Layer) Forward(inputs []float64) []float64 {
	if len(inputs) != len(l.Inputs) {
		l.Inputs = make([]float64, len(inputs))
	}
	copy(l.Inputs, inputs)

	if len(l.Outputs) != len(l.Biases) {
		l.Outputs = make([]float64, len(l.Biases))
	}

	for i := 0; i < len(l.Biases); i++ {
		sum := l.Biases[i]
		for j := 0; j < len(inputs); j++ {
			sum += l.Weights[i][j] * inputs[j]
		}
		l.Outputs[i] = sum
	}

	switch l.ActivationFunc {
	case "relu":
		l.applyReLU()
	case "sigmoid":
		l.applySigmoid()
	case "tanh":
		l.applyTanh()
	case "softmax":
		l.applySoftmax()
	case "leaky_relu":
		l.applyLeakyReLU()
	default:
	}

	return l.Outputs
}

func (l *Layer) applyReLU() {
	for i := 0; i < len(l.Outputs); i++ {
		if l.Outputs[i] < 0 {
			l.Outputs[i] = 0
		}
	}
}

func (l *Layer) applyLeakyReLU() {
	for i := 0; i < len(l.Outputs); i++ {
		if l.Outputs[i] < 0 {
			l.Outputs[i] *= 0.01
		}
	}
}

func (l *Layer) applySigmoid() {
	for i := 0; i < len(l.Outputs); i++ {
		l.Outputs[i] = sigmoid(l.Outputs[i])
	}
}

func (l *Layer) applyTanh() {
	for i := 0; i < len(l.Outputs); i++ {
		l.Outputs[i] = math.Tanh(l.Outputs[i])
	}
}

func (l *Layer) applySoftmax() {
	maxVal := l.Outputs[0]
	for i := 1; i < len(l.Outputs); i++ {
		if l.Outputs[i] > maxVal {
			maxVal = l.Outputs[i]
		}
	}

	sum := 0.0
	for i := 0; i < len(l.Outputs); i++ {
		l.Outputs[i] = math.Exp(l.Outputs[i] - maxVal)
		sum += l.Outputs[i]
	}

	for i := 0; i < len(l.Outputs); i++ {
		l.Outputs[i] /= sum
	}
}

func (l *Layer) Backward(outputGradients []float64, learningRate float64) []float64 {
	var inputGradients []float64

	switch l.ActivationFunc {
	case "relu":
		inputGradients = l.backpropReLU(outputGradients)
	case "sigmoid":
		inputGradients = l.backpropSigmoid(outputGradients)
	case "tanh":
		inputGradients = l.backpropTanh(outputGradients)
	case "leaky_relu":
		inputGradients = l.backpropLeakyReLU(outputGradients)
	default:
		inputGradients = make([]float64, len(l.Inputs))
		for i := 0; i < len(l.Inputs); i++ {
			inputGradients[i] = 0
			for j := 0; j < len(outputGradients); j++ {
				inputGradients[i] += l.Weights[j][i] * outputGradients[j]
			}
		}
	}

	for i := 0; i < len(l.Biases); i++ {
		l.Biases[i] -= learningRate * outputGradients[i]
		for j := 0; j < len(l.Inputs); j++ {
			l.Weights[i][j] -= learningRate * outputGradients[i] * l.Inputs[j]
		}
	}

	return inputGradients
}

func (l *Layer) backpropReLU(gradients []float64) []float64 {
	inputGradients := make([]float64, len(l.Inputs))
	for i := 0; i < len(l.Inputs); i++ {
		inputGradients[i] = 0
		for j := 0; j < len(gradients); j++ {
			deriv := 0.0
			if l.Inputs[i] > 0 {
				deriv = 1.0
			}
			inputGradients[i] += l.Weights[j][i] * gradients[j] * deriv
		}
	}
	return inputGradients
}

func (l *Layer) backpropLeakyReLU(gradients []float64) []float64 {
	inputGradients := make([]float64, len(l.Inputs))
	for i := 0; i < len(l.Inputs); i++ {
		inputGradients[i] = 0
		for j := 0; j < len(gradients); j++ {
			deriv := 1.0
			if l.Inputs[i] < 0 {
				deriv = 0.01
			}
			inputGradients[i] += l.Weights[j][i] * gradients[j] * deriv
		}
	}
	return inputGradients
}

func (l *Layer) backpropSigmoid(gradients []float64) []float64 {
	inputGradients := make([]float64, len(l.Inputs))
	for i := 0; i < len(l.Inputs); i++ {
		inputGradients[i] = 0
		for j := 0; j < len(gradients); j++ {
			sigmoidDeriv := l.Outputs[j] * (1 - l.Outputs[j])
			inputGradients[i] += l.Weights[j][i] * gradients[j] * sigmoidDeriv
		}
	}
	return inputGradients
}

func (l *Layer) backpropTanh(gradients []float64) []float64 {
	inputGradients := make([]float64, len(l.Inputs))
	for i := 0; i < len(l.Inputs); i++ {
		inputGradients[i] = 0
		for j := 0; j < len(gradients); j++ {
			tanhDeriv := 1 - l.Outputs[j]*l.Outputs[j]
			inputGradients[i] += l.Weights[j][i] * gradients[j] * tanhDeriv
		}
	}
	return inputGradients
}

type NeuralNetwork struct {
	layers      []*Layer
	inputDim    int
	outputDim   int
	mu          sync.RWMutex
	isTraining  bool
	dropoutRate float64
}

func NewNeuralNetwork(inputDim int, hiddenDims []int, outputDim int) *NeuralNetwork {
	nn := &NeuralNetwork{
		inputDim:   inputDim,
		outputDim:  outputDim,
		dropoutRate: 0.0,
	}

	currentDim := inputDim
	for i, hiddenDim := range hiddenDims {
		activation := "relu"
		if i == len(hiddenDims)-1 {
			activation = "relu"
		}
		nn.layers = append(nn.layers, NewLayer(currentDim, hiddenDim, activation))
		currentDim = hiddenDim
	}

	nn.layers = append(nn.layers, NewLayer(currentDim, outputDim, "sigmoid"))

	return nn
}

func (nn *NeuralNetwork) Forward(inputs []float64) []float64 {
	nn.mu.Lock()
	defer nn.mu.Unlock()

	current := inputs
	for _, layer := range nn.layers {
		current = layer.Forward(current)
	}
	return current
}

func (nn *NeuralNetwork) Predict(inputs []float64) float64 {
	outputs := nn.Forward(inputs)
	if len(outputs) > 0 {
		return outputs[0]
	}
	return 0.0
}

func (nn *NeuralNetwork) PredictBatch(inputs [][]float64) []float64 {
	outputs := make([]float64, len(inputs))
	for i, input := range inputs {
		outputs[i] = nn.Predict(input)
	}
	return outputs
}

func (nn *NeuralNetwork) Train(inputs [][]float64, labels []float64, config *ModelConfig) *TrainingResult {
	nn.mu.Lock()
	nn.isTraining = true
	nn.mu.Unlock()

	if config.DropoutRate > 0 {
		nn.dropoutRate = config.DropoutRate
	}

	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 32
	}

	epochs := config.Epochs
	if epochs <= 0 {
		epochs = 100
	}

	bestLoss := math.MaxFloat64
	bestWeights := nn.cloneWeights()

	for epoch := 0; epoch < epochs; epoch++ {
		indices := rand.Perm(len(inputs))
		epochLoss := 0.0
		numBatches := 0

		for i := 0; i < len(inputs); i += batchSize {
			end := i + batchSize
			if end > len(inputs) {
				end = len(inputs)
			}

			for _, idx := range indices[i:end] {
				input := inputs[idx]
				label := labels[idx]

				outputs := nn.Forward(input)
				if len(outputs) > 0 {
					loss := nn.calculateLoss(outputs[0], label)
					epochLoss += loss

					gradients := nn.calculateOutputGradient(outputs[0], label)
					for i := len(nn.layers) - 1; i >= 0; i-- {
						gradients = nn.layers[i].Backward(gradients, config.LearningRate)
					}
				}
			}
			numBatches++
		}

		avgLoss := epochLoss / float64(len(inputs))

		if config.WeightDecay > 0 {
			nn.applyWeightDecay(config.WeightDecay)
		}

		if avgLoss < bestLoss {
			bestLoss = avgLoss
			bestWeights = nn.cloneWeights()
		}

		if epoch%10 == 0 {
			fmt.Printf("Epoch %d, Loss: %.6f, Best Loss: %.6f\n", epoch, avgLoss, bestLoss)
		}
	}

	nn.mu.Lock()
	nn.restoreWeights(bestWeights)
	nn.isTraining = false
	nn.mu.Unlock()

	return &TrainingResult{
		Epoch:          epochs,
		TrainLoss:      bestLoss,
		ValidationLoss: bestLoss,
	}
}

func (nn *NeuralNetwork) calculateLoss(predicted, actual float64) float64 {
	eps := Epsilon
	predicted = math.Max(eps, math.Min(1-eps, predicted))
	return -actual*math.Log(predicted) - (1-actual)*math.Log(1-predicted)
}

func (nn *NeuralNetwork) calculateOutputGradient(predicted, actual float64) []float64 {
	gradients := make([]float64, len(nn.layers[len(nn.layers)-1].Outputs))
	for i := range gradients {
		gradients[i] = predicted - actual
	}
	return gradients
}

func (nn *NeuralNetwork) applyWeightDecay(decay float64) {
	for _, layer := range nn.layers {
		for i := range layer.Weights {
			for j := range layer.Weights[i] {
				layer.Weights[i][j] *= (1 - decay)
			}
		}
	}
}

func (nn *NeuralNetwork) cloneWeights() [][][]float64 {
	weights := make([][][]float64, len(nn.layers))
	for i, layer := range nn.layers {
		weights[i] = make([][]float64, len(layer.Weights))
		for j := range layer.Weights {
			weights[i][j] = make([]float64, len(layer.Weights[j]))
			copy(weights[i][j], layer.Weights[j])
		}
	}
	return weights
}

func (nn *NeuralNetwork) restoreWeights(weights [][][]float64) {
	for i, layerWeights := range weights {
		if i < len(nn.layers) {
			for j := range layerWeights {
				if j < len(nn.layers[i].Weights) {
					copy(nn.layers[i].Weights[j], layerWeights[j])
				}
			}
		}
	}
}

func (nn *NeuralNetwork) Save(path string) error {
	nn.mu.RLock()
	defer nn.mu.RUnlock()

	data := make(map[string]interface{})
	data["inputDim"] = nn.inputDim
	data["outputDim"] = nn.outputDim
	data["dropoutRate"] = nn.dropoutRate

	layers := make([]map[string]interface{}, len(nn.layers))
	for i, layer := range nn.layers {
		layerData := map[string]interface{}{
			"weights":        layer.Weights,
			"biases":         layer.Biases,
			"activationFunc": layer.ActivationFunc,
		}
		layers[i] = layerData
	}
	data["layers"] = layers

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal neural network data: %w", err)
	}

	return os.WriteFile(path, jsonData, 0644)
}

func (nn *NeuralNetwork) Load(path string) error {
	nn.mu.Lock()
	defer nn.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read neural network file: %w", err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("failed to unmarshal neural network data: %w", err)
	}

	nn.inputDim = int(jsonData["inputDim"].(float64))
	nn.outputDim = int(jsonData["outputDim"].(float64))
	nn.dropoutRate = jsonData["dropoutRate"].(float64)

	layersData := jsonData["layers"].([]interface{})
	nn.layers = make([]*Layer, len(layersData))

	for i, layerInterface := range layersData {
		layerMap := layerInterface.(map[string]interface{})

		weightsInterface := layerMap["weights"].([]interface{})
		weights := make([][]float64, len(weightsInterface))
		for j, weightRowInterface := range weightsInterface {
			weightRow := weightRowInterface.([]interface{})
			weights[j] = make([]float64, len(weightRow))
			for k, w := range weightRow {
				weights[j][k] = w.(float64)
			}
		}

		biasesInterface := layerMap["biases"].([]interface{})
		biases := make([]float64, len(biasesInterface))
		for j, b := range biasesInterface {
			biases[j] = b.(float64)
		}

		activationFunc := layerMap["activationFunc"].(string)

		layer := &Layer{
			Weights:        weights,
			Biases:         biases,
			Inputs:         make([]float64, len(weights[0])),
			Outputs:        make([]float64, len(biases)),
			GradientsW:     make([][]float64, len(weights)),
			GradientsB:     make([]float64, len(biases)),
			ActivationFunc: activationFunc,
		}

		for j := range layer.GradientsW {
			layer.GradientsW[j] = make([]float64, len(weights[0]))
		}

		nn.layers[i] = layer
	}

	return nil
}

func (nn *NeuralNetwork) GetInputDimension() int {
	return nn.inputDim
}

func (nn *NeuralNetwork) GetOutputDimension() int {
	return nn.outputDim
}

func (nn *NeuralNetwork) IsTraining() bool {
	nn.mu.RLock()
	defer nn.mu.RUnlock()
	return nn.isTraining
}

type MLPModel struct {
	nn     *NeuralNetwork
	config *ModelConfig
	ready  bool
	mu     sync.RWMutex
}

func NewMLPModel(config *ModelConfig) *MLPModel {
	hiddenDims := config.HiddenDims
	if len(hiddenDims) == 0 {
		hiddenDims = []int{64, 32, 16}
	}

	nn := NewNeuralNetwork(config.InputDim, hiddenDims, config.OutputDim)

	return &MLPModel{
		nn:     nn,
		config: config,
		ready:  true,
	}
}

func (m *MLPModel) Predict(ctx context.Context, features []float64) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.ready {
		return 0.5, fmt.Errorf("model is not ready")
	}

	if len(features) != m.nn.inputDim {
		return 0.5, fmt.Errorf("invalid feature dimension: expected %d, got %d", m.nn.inputDim, len(features))
	}

	normalizedFeatures := m.normalizeFeatures(features)
	score := m.nn.Predict(normalizedFeatures)

	return score, nil
}

func (m *MLPModel) PredictBatch(ctx context.Context, featuresBatch [][]float64) ([]float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.ready {
		return nil, fmt.Errorf("model is not ready")
	}

	scores := make([]float64, len(featuresBatch))
	for i, features := range featuresBatch {
		if len(features) != m.nn.inputDim {
			scores[i] = 0.5
			continue
		}
		normalizedFeatures := m.normalizeFeatures(features)
		scores[i] = m.nn.Predict(normalizedFeatures)
	}

	return scores, nil
}

func (m *MLPModel) normalizeFeatures(features []float64) []float64 {
	normalized := make([]float64, len(features))
	copy(normalized, features)

	mean := 0.0
	for _, f := range normalized {
		mean += f
	}
	mean /= float64(len(normalized))

	std := 0.0
	for _, f := range normalized {
		std += (f - mean) * (f - mean)
	}
	std = math.Sqrt(std / float64(len(normalized)))

	if std > Epsilon {
		for i := range normalized {
			normalized[i] = (normalized[i] - mean) / std
		}
	}

	return normalized
}

func (m *MLPModel) LoadWeights(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.nn.Load(path); err != nil {
		return err
	}

	m.ready = true
	return nil
}

func (m *MLPModel) SaveWeights(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.nn.Save(path)
}

func (m *MLPModel) GetModelType() ModelType {
	return ModelTypeMLP
}

func (m *MLPModel) GetInputDimension() int {
	return m.nn.inputDim
}

func (m *MLPModel) GetOutputDimension() int {
	return m.nn.outputDim
}

func (m *MLPModel) IsReady() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ready
}

func (m *MLPModel) GetNeuralNetwork() *NeuralNetwork {
	return m.nn
}

func (m *MLPModel) SetReady(ready bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ready = ready
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func sigmoidDerivative(x float64) float64 {
	s := sigmoid(x)
	return s * (1 - s)
}

func ExportWeightsAsBinary(nn *NeuralNetwork, path string) error {
	nn.mu.RLock()
	defer nn.mu.RUnlock()

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := binary.Write(file, binary.LittleEndian, int32(nn.inputDim)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, int32(nn.outputDim)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, int32(len(nn.layers))); err != nil {
		return err
	}

	for _, layer := range nn.layers {
		if err := binary.Write(file, binary.LittleEndian, int32(len(layer.Weights))); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, int32(len(layer.Weights[0]))); err != nil {
			return err
		}

		for i := range layer.Weights {
			for j := range layer.Weights[i] {
				if err := binary.Write(file, binary.LittleEndian, layer.Weights[i][j]); err != nil {
					return err
				}
			}
		}

		for i := range layer.Biases {
			if err := binary.Write(file, binary.LittleEndian, layer.Biases[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func ImportWeightsFromBinary(nn *NeuralNetwork, path string) error {
	nn.mu.Lock()
	defer nn.mu.Unlock()

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var inputDim, outputDim int32
	if err := binary.Read(file, binary.LittleEndian, &inputDim); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &outputDim); err != nil {
		return err
	}

	var numLayers int32
	if err := binary.Read(file, binary.LittleEndian, &numLayers); err != nil {
		return err
	}

	for l := 0; l < int(numLayers); l++ {
		var rows, cols int32
		if err := binary.Read(file, binary.LittleEndian, &rows); err != nil {
			return err
		}
		if err := binary.Read(file, binary.LittleEndian, &cols); err != nil {
			return err
		}

		for i := int32(0); i < rows; i++ {
			for j := int32(0); j < cols; j++ {
				if err := binary.Read(file, binary.LittleEndian, &nn.layers[l].Weights[i][j]); err != nil {
					return err
				}
			}
		}

		for i := int32(0); i < rows; i++ {
			if err := binary.Read(file, binary.LittleEndian, &nn.layers[l].Biases[i]); err != nil {
				return err
			}
		}
	}

	return nil
}
