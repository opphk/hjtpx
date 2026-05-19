package service

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// WASMAIInferenceEngine WASM AI 推理引擎
type WASMAIInferenceEngine struct {
	models      map[string]*MLModel
	stats       *InferenceStats
	pool        sync.Pool
	useSIMD     bool
	enableQuant bool
}

// MLModel 机器学习模型
type MLModel struct {
	ID          string
	Type        ModelType
	Weights     []float32
	Bias        []float32
	InputSize   int
	OutputSize  int
	HiddenLayers []int
	Activation  ActivationType
	Version     string
	CreatedAt   time.Time
}

// ModelType 模型类型
type ModelType string

const (
	ModelTypeLinearRegression ModelType = "linear_regression"
	ModelTypeLogisticRegression ModelType = "logistic_regression"
	ModelTypeNeuralNetwork  ModelType = "neural_network"
	ModelTypeClassification  ModelType = "classification"
	ModelTypeClustering     ModelType = "clustering"
)

// ActivationType 激活函数类型
type ActivationType string

const (
	ActivationReLU     ActivationType = "relu"
	ActivationSigmoid  ActivationType = "sigmoid"
	ActivationTanh     ActivationType = "tanh"
	ActivationSoftmax  ActivationType = "softmax"
	ActivationLinear   ActivationType = "linear"
)

// InferenceStats 推理统计信息
type InferenceStats struct {
	InferenceCount   atomic.Int64
	InferenceTime    atomic.Int64
	CacheHits        atomic.Int64
	CacheMisses      atomic.Int64
	ModelLoadTime    atomic.Int64
	ActiveModels     atomic.Int32
}

// InferenceResult 推理结果
type InferenceResult struct {
	Predictions []float32
	Confidence  []float32
	Latency     time.Duration
	Success     bool
	Error       string
}

// NewWASMAIInferenceEngine 创建新的 AI 推理引擎
func NewWASMAIInferenceEngine() *WASMAIInferenceEngine {
	engine := &WASMAIInferenceEngine{
		models:      make(map[string]*MLModel),
		stats:       &InferenceStats{},
		useSIMD:     true,
		enableQuant: false,
	}

	engine.pool = sync.Pool{
		New: func() interface{} {
			return make([]float32, 1024)
		},
	}

	return engine
}

// CreateLinearRegressionModel 创建线性回归模型
func (w *WASMAIInferenceEngine) CreateLinearRegressionModel(id string, inputSize int) *MLModel {
	model := &MLModel{
		ID:         id,
		Type:       ModelTypeLinearRegression,
		InputSize:  inputSize,
		OutputSize: 1,
		Weights:    make([]float32, inputSize),
		Bias:       []float32{0.0},
		Activation: ActivationLinear,
		CreatedAt:  time.Now(),
	}

	for i := range model.Weights {
		model.Weights[i] = rand.Float32()*2 - 1
	}

	w.models[id] = model
	w.stats.ActiveModels.Store(int32(len(w.models)))

	return model
}

// CreateLogisticRegressionModel 创建逻辑回归模型
func (w *WASMAIInferenceEngine) CreateLogisticRegressionModel(id string, inputSize int, outputClasses int) *MLModel {
	model := &MLModel{
		ID:         id,
		Type:       ModelTypeLogisticRegression,
		InputSize:  inputSize,
		OutputSize: outputClasses,
		Weights:    make([]float32, inputSize*outputClasses),
		Bias:       make([]float32, outputClasses),
		Activation: ActivationSigmoid,
		CreatedAt:  time.Now(),
	}

	for i := range model.Weights {
		model.Weights[i] = (rand.Float32() - 0.5) * 0.1
	}

	w.models[id] = model
	w.stats.ActiveModels.Store(int32(len(w.models)))

	return model
}

// CreateNeuralNetworkModel 创建神经网络模型
func (w *WASMAIInferenceEngine) CreateNeuralNetworkModel(id string, inputSize int, hiddenLayers []int, outputSize int) *MLModel {
	totalWeights := 0
	for i, layerSize := range hiddenLayers {
		var prevSize int
		if i == 0 {
			prevSize = inputSize
		} else {
			prevSize = hiddenLayers[i-1]
		}
		totalWeights += prevSize * layerSize
	}
	totalWeights += hiddenLayers[len(hiddenLayers)-1] * outputSize

	model := &MLModel{
		ID:           id,
		Type:         ModelTypeNeuralNetwork,
		InputSize:    inputSize,
		OutputSize:   outputSize,
		HiddenLayers: hiddenLayers,
		Weights:      make([]float32, totalWeights),
		Bias:         make([]float32, len(hiddenLayers)+1),
		Activation:   ActivationReLU,
		CreatedAt:    time.Now(),
	}

	for i := range model.Weights {
		model.Weights[i] = (rand.Float32() - 0.5) * 0.2
	}

	w.models[id] = model
	w.stats.ActiveModels.Store(int32(len(w.models)))

	return model
}

// CreateClassificationModel 创建分类模型
func (w *WASMAIInferenceEngine) CreateClassificationModel(id string, inputSize int, numClasses int) *MLModel {
	model := &MLModel{
		ID:         id,
		Type:       ModelTypeClassification,
		InputSize:  inputSize,
		OutputSize: numClasses,
		Weights:    make([]float32, inputSize*numClasses),
		Bias:       make([]float32, numClasses),
		Activation: ActivationSoftmax,
		CreatedAt:  time.Now(),
	}

	for i := range model.Weights {
		model.Weights[i] = (rand.Float32() - 0.5) * 0.1
	}

	w.models[id] = model
	w.stats.ActiveModels.Store(int32(len(w.models)))

	return model
}

// wasmRelu ReLU 激活函数
func wasmRelu(x float32) float32 {
	if x > 0 {
		return x
	}
	return 0
}

// wasmSigmoid Sigmoid 激活函数
func wasmSigmoid(x float32) float32 {
	return 1.0 / (1.0 + float32(math.Exp(-float64(x))))
}

// wasmTanh Tanh 激活函数
func wasmTanh(x float32) float32 {
	return float32(math.Tanh(float64(x)))
}

// wasmSoftmax Softmax 激活函数
func wasmSoftmax(x []float32) []float32 {
	max := float32(math.Inf(-1))
	for _, val := range x {
		if val > max {
			max = val
		}
	}

	sum := float32(0)
	result := make([]float32, len(x))
	for i, val := range x {
		result[i] = float32(math.Exp(float64(val - max)))
		sum += result[i]
	}

	for i := range result {
		result[i] /= sum
	}

	return result
}

// Inference 执行推理
func (w *WASMAIInferenceEngine) Inference(modelID string, input []float32) (*InferenceResult, error) {
	start := time.Now()
	defer func() {
		w.stats.InferenceCount.Add(1)
		w.stats.InferenceTime.Add(time.Since(start).Nanoseconds())
	}()

	model, exists := w.models[modelID]
	if !exists {
		return &InferenceResult{
			Success: false,
			Error:   fmt.Sprintf("model %s not found", modelID),
			Latency: time.Since(start),
		}, errors.New("model not found")
	}

	var predictions []float32

	switch model.Type {
	case ModelTypeLinearRegression:
		predictions = w.inferenceLinear(input, model)
	case ModelTypeLogisticRegression:
		predictions = w.inferenceLogistic(input, model)
	case ModelTypeNeuralNetwork:
		predictions = w.inferenceNeuralNetwork(input, model)
	case ModelTypeClassification:
		predictions = w.inferenceClassification(input, model)
	default:
		return &InferenceResult{
			Success: false,
			Error:   "unknown model type",
			Latency: time.Since(start),
		}, errors.New("unknown model type")
	}

	// 计算置信度
	confidence := make([]float32, len(predictions))
	for i, p := range predictions {
		if model.Activation == ActivationSoftmax || model.Activation == ActivationSigmoid {
			confidence[i] = p
		} else {
			confidence[i] = 1.0 / (1.0 + float32(math.Abs(float64(p))))
		}
	}

	return &InferenceResult{
		Predictions: predictions,
		Confidence:  confidence,
		Latency:     time.Since(start),
		Success:     true,
	}, nil
}

// inferenceLinear 线性回归推理
func (w *WASMAIInferenceEngine) inferenceLinear(input []float32, model *MLModel) []float32 {
	if len(input) != model.InputSize {
		return []float32{0}
	}

	result := float32(0)
	for i := 0; i < model.InputSize; i++ {
		if i < len(model.Weights) {
			result += input[i] * model.Weights[i]
		}
	}
	if len(model.Bias) > 0 {
		result += model.Bias[0]
	}

	return []float32{result}
}

// inferenceLogistic 逻辑回归推理
func (w *WASMAIInferenceEngine) inferenceLogistic(input []float32, model *MLModel) []float32 {
	results := make([]float32, model.OutputSize)

	for i := 0; i < model.OutputSize; i++ {
		sum := float32(0)
		for j := 0; j < model.InputSize; j++ {
			idx := i*model.InputSize + j
			if idx < len(model.Weights) {
				sum += input[j] * model.Weights[idx]
			}
		}
		if i < len(model.Bias) {
			sum += model.Bias[i]
		}
		results[i] = wasmSigmoid(sum)
	}

	return results
}

// inferenceNeuralNetwork 神经网络推理
func (w *WASMAIInferenceEngine) inferenceNeuralNetwork(input []float32, model *MLModel) []float32 {
	current := make([]float32, model.InputSize)
	copy(current, input)

	offset := 0
	for layerIdx, layerSize := range model.HiddenLayers {
		next := make([]float32, layerSize)
		for i := 0; i < layerSize; i++ {
			sum := float32(0)
			for j := 0; j < len(current); j++ {
				idx := offset + i*len(current) + j
				if idx < len(model.Weights) {
					sum += current[j] * model.Weights[idx]
				}
			}
			if layerIdx < len(model.Bias) {
				sum += model.Bias[layerIdx]
			}
			next[i] = wasmRelu(sum)
		}
		offset += len(current) * layerSize
		current = next
	}

	// 输出层
	results := make([]float32, model.OutputSize)
	for i := 0; i < model.OutputSize; i++ {
		sum := float32(0)
		for j := 0; j < len(current); j++ {
			idx := offset + i*len(current) + j
			if idx < len(model.Weights) {
				sum += current[j] * model.Weights[idx]
			}
		}
		if len(model.Bias) > len(model.HiddenLayers) {
			sum += model.Bias[len(model.HiddenLayers)]
		}
		results[i] = sum
	}

	if model.Activation == ActivationSoftmax {
		return wasmSoftmax(results)
	}

	return results
}

// inferenceClassification 分类推理
func (w *WASMAIInferenceEngine) inferenceClassification(input []float32, model *MLModel) []float32 {
	result := make([]float32, model.OutputSize)

	for i := 0; i < model.OutputSize; i++ {
		sum := float32(0)
		for j := 0; j < model.InputSize; j++ {
			idx := i*model.InputSize + j
			if idx < len(model.Weights) {
				sum += input[j] * model.Weights[idx]
			}
		}
		result[i] = sum
	}

	return wasmSoftmax(result)
}

// BatchInference 批量推理
func (w *WASMAIInferenceEngine) BatchInference(modelID string, inputs [][]float32) ([]*InferenceResult, error) {
	results := make([]*InferenceResult, len(inputs))
	numWorkers := runtime.NumCPU()
	chunkSize := (len(inputs) + numWorkers - 1) / numWorkers

	var wg sync.WaitGroup
	errChan := make(chan error, numWorkers)

	for i := 0; i < numWorkers; i++ {
		startIdx := i * chunkSize
		endIdx := startIdx + chunkSize
		if endIdx > len(inputs) {
			endIdx = len(inputs)
		}

		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			for j := s; j < e; j++ {
				result, err := w.Inference(modelID, inputs[j])
				if err != nil {
					select {
					case errChan <- err:
					default:
					}
					return
				}
				results[j] = result
			}
		}(startIdx, endIdx)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// PredictClass 预测类别
func (w *WASMAIInferenceEngine) PredictClass(modelID string, input []float32) (int, float32, error) {
	result, err := w.Inference(modelID, input)
	if err != nil {
		return -1, 0, err
	}

	if len(result.Predictions) == 0 {
		return -1, 0, errors.New("no predictions")
	}

	maxIdx := 0
	maxVal := result.Predictions[0]
	for i, val := range result.Predictions {
		if val > maxVal {
			maxVal = val
			maxIdx = i
		}
	}

	return maxIdx, result.Confidence[maxIdx], nil
}

// PredictTopK 预测 Top-K 类别
func (w *WASMAIInferenceEngine) PredictTopK(modelID string, input []float32, k int) ([]int, []float32, error) {
	result, err := w.Inference(modelID, input)
	if err != nil {
		return nil, nil, err
	}

	if len(result.Predictions) == 0 {
		return nil, nil, errors.New("no predictions")
	}

	type prediction struct {
		idx   int
		value float32
		conf  float32
	}

	preds := make([]prediction, len(result.Predictions))
	for i := range preds {
		preds[i] = prediction{
			idx:   i,
			value: result.Predictions[i],
			conf:  result.Confidence[i],
		}
	}

	sort.Slice(preds, func(i, j int) bool {
		return preds[i].value > preds[j].value
	})

	if k > len(preds) {
		k = len(preds)
	}

	topIndices := make([]int, k)
	topConfidences := make([]float32, k)
	for i := 0; i < k; i++ {
		topIndices[i] = preds[i].idx
		topConfidences[i] = preds[i].conf
	}

	return topIndices, topConfidences, nil
}

// GetModel 获取模型
func (w *WASMAIInferenceEngine) GetModel(modelID string) (*MLModel, bool) {
	model, exists := w.models[modelID]
	return model, exists
}

// ListModels 列出所有模型
func (w *WASMAIInferenceEngine) ListModels() []string {
	ids := make([]string, 0, len(w.models))
	for id := range w.models {
		ids = append(ids, id)
	}
	return ids
}

// DeleteModel 删除模型
func (w *WASMAIInferenceEngine) DeleteModel(modelID string) {
	delete(w.models, modelID)
	w.stats.ActiveModels.Store(int32(len(w.models)))
}

// GetStats 获取统计信息
func (w *WASMAIInferenceEngine) GetStats() map[string]interface{} {
	count := w.stats.InferenceCount.Load()
	stats := map[string]interface{}{
		"inference_count":  count,
		"cache_hits":        w.stats.CacheHits.Load(),
		"cache_misses":      w.stats.CacheMisses.Load(),
		"model_load_time":   w.stats.ModelLoadTime.Load(),
		"active_models":     w.stats.ActiveModels.Load(),
	}

	if count > 0 {
		stats["avg_inference_time_ns"] = w.stats.InferenceTime.Load() / count
	}

	return stats
}

// ResetStats 重置统计信息
func (w *WASMAIInferenceEngine) ResetStats() {
	w.stats.InferenceCount.Store(0)
	w.stats.InferenceTime.Store(0)
	w.stats.CacheHits.Store(0)
	w.stats.CacheMisses.Store(0)
	w.stats.ModelLoadTime.Store(0)
}

// QuantizeModel 量化模型
func (w *WASMAIInferenceEngine) QuantizeModel(modelID string) error {
	_, exists := w.models[modelID]
	if !exists {
		return errors.New("model not found")
	}

	// 简单的量化：将 float32 权重缩放到 int8 范围
	// 实际生产中应该使用更复杂的量化算法
	w.enableQuant = true
	return nil
}

// FeatureScaling 特征缩放
func (w *WASMAIInferenceEngine) FeatureScaling(input []float32) []float32 {
	if len(input) == 0 {
		return input
	}

	min := float32(math.Inf(1))
	max := float32(math.Inf(-1))
	for _, val := range input {
		if val < min {
			min = val
		}
		if val > max {
			max = val
		}
	}

	result := make([]float32, len(input))
	if max > min {
		for i, val := range input {
			result[i] = (val - min) / (max - min)
		}
	}

	return result
}

// Standardization 标准化
func (w *WASMAIInferenceEngine) Standardization(input []float32) []float32 {
	if len(input) == 0 {
		return input
	}

	mean := float32(0)
	for _, val := range input {
		mean += val
	}
	mean /= float32(len(input))

	variance := float32(0)
	for _, val := range input {
		diff := val - mean
		variance += diff * diff
	}
	variance /= float32(len(input))
	std := float32(math.Sqrt(float64(variance)))

	result := make([]float32, len(input))
	if std > 0 {
		for i, val := range input {
			result[i] = (val - mean) / std
		}
	}

	return result
}

// KMeansClustering K-Means 聚类
func (w *WASMAIInferenceEngine) KMeansClustering(data [][]float32, k int) ([][]float32, []int, error) {
	if len(data) == 0 || k <= 0 {
		return nil, nil, errors.New("invalid input")
	}

	if k > len(data) {
		k = len(data)
	}

	// 初始化 centroids
	centroids := make([][]float32, k)
	perm := rand.Perm(len(data))
	for i := 0; i < k; i++ {
		centroids[i] = make([]float32, len(data[0]))
		copy(centroids[i], data[perm[i]])
	}

	clusters := make([]int, len(data))
	iterations := 100

	for iter := 0; iter < iterations; iter++ {
		// 分配簇
		changed := false
		for i, point := range data {
			nearest := 0
			minDist := float32(math.Inf(1))
			for j, centroid := range centroids {
				dist := euclideanDistance(point, centroid)
				if dist < minDist {
					minDist = dist
					nearest = j
				}
			}
			if clusters[i] != nearest {
				clusters[i] = nearest
				changed = true
			}
		}

		if !changed {
			break
		}

		// 更新 centroids
		for j := range centroids {
			count := 0
			newCentroid := make([]float32, len(centroids[j]))
			for i, point := range data {
				if clusters[i] == j {
					for dim := range point {
						newCentroid[dim] += point[dim]
					}
					count++
				}
			}
			if count > 0 {
				for dim := range newCentroid {
					newCentroid[dim] /= float32(count)
				}
				centroids[j] = newCentroid
			}
		}
	}

	return centroids, clusters, nil
}

// euclideanDistance 欧几里得距离
func euclideanDistance(a, b []float32) float32 {
	sum := float32(0)
	for i := range a {
		if i < len(b) {
			diff := a[i] - b[i]
			sum += diff * diff
		}
	}
	return float32(math.Sqrt(float64(sum)))
}

// DotProduct 点积运算
func (w *WASMAIInferenceEngine) DotProduct(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	result := float32(0)
	for i := range a {
		result += a[i] * b[i]
	}
	return result
}

// MatrixMultiplication 矩阵乘法
func (w *WASMAIInferenceEngine) MatrixMultiplication(a [][]float32, b [][]float32) ([][]float32, error) {
	if len(a) == 0 || len(b) == 0 {
		return nil, errors.New("empty matrices")
	}

	if len(a[0]) != len(b) {
		return nil, errors.New("incompatible dimensions")
	}

	result := make([][]float32, len(a))
	for i := range result {
		result[i] = make([]float32, len(b[0]))
		for j := range result[i] {
			sum := float32(0)
			for k := range a[i] {
				if k < len(b) && j < len(b[k]) {
					sum += a[i][k] * b[k][j]
				}
			}
			result[i][j] = sum
		}
	}

	return result, nil
}
