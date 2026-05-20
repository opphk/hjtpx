package service

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type WASMAIInference struct {
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	isRunning     bool
	models        map[string]*WASMModel
	modelPool     *sync.Pool
	stats         *WASMInferenceStats
	quantizer     *ModelQuantizer
	accelerator   *AIAccelerator
	enableSIMD    bool
	maxBatchSize  int
}

type WASMModel struct {
	ID            string
	Name          string
	Version       string
	Weights       []float32
	Architecture  string
	InputShape    []int
	OutputShape   []int
	Quantized     bool
	LoadedAt      time.Time
	LastUsed      time.Time
	UseCount      int64
	MemorySize    int64
}

type WASMInferenceStats struct {
	TotalInferences  atomic.Int64
	CacheHits        atomic.Int64
	CacheMisses      atomic.Int64
	BatchHits        atomic.Int64
	BatchMisses      atomic.Int64
	AvgInferenceTime atomic.Int64
	TotalMemory      atomic.Int64
	ActiveModels     atomic.Int64
	LastUpdate       atomic.Value
}

type ModelQuantizer struct {
	mu        sync.RWMutex
	quantized map[string][]byte
}

type AIAccelerator struct {
	mu            sync.RWMutex
	gpuAvailable  bool
	simdEnabled   bool
	threadCount   int
	offloadQueue  chan InferenceTask
}

type InferenceTask struct {
	ModelID   string
	Input     []float32
	Output    chan []float32
	Error     chan error
}

type Tensor struct {
	Shape []int
	Data  []float32
}

func NewWASMAIInference() *WASMAIInference {
	ctx, cancel := context.WithCancel(context.Background())

	return &WASMAIInference{
		ctx:          ctx,
		cancel:       cancel,
		models:       make(map[string]*WASMModel),
		modelPool:    newModelPool(),
		stats:        &WASMInferenceStats{},
		quantizer:    NewModelQuantizer(),
		accelerator:  NewAIAccelerator(),
		enableSIMD:   true,
		maxBatchSize: 64,
	}
}

func newModelPool() *sync.Pool {
	return &sync.Pool{
		New: func() interface{} {
			return &WASMInferenceContext{
				buffer: make([]float32, 65536),
			}
		},
	}
}

func NewModelQuantizer() *ModelQuantizer {
	return &ModelQuantizer{
		quantized: make(map[string][]byte),
	}
}

func NewAIAccelerator() *AIAccelerator {
	return &AIAccelerator{
		gpuAvailable: false,
		simdEnabled:  true,
		threadCount:  4,
		offloadQueue: make(chan InferenceTask, 1000),
	}
}

type WASMInferenceContext struct {
	buffer []float32
}

func (w *WASMAIInference) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isRunning {
		return nil
	}

	w.isRunning = true
	go w.cleanupModels()
	go w.processOffloadQueue()

	fmt.Println("[WASMAIInference] Started successfully")
	return nil
}

func (w *WASMAIInference) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isRunning {
		return
	}

	w.cancel()
	w.isRunning = false
	fmt.Println("[WASMAIInference] Stopped")
}

func (w *WASMAIInference) LoadModel(id, name, version string, weights []float32, inputShape, outputShape []int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.models[id]; exists {
		return nil
	}

	architecture := w.inferArchitecture(len(inputShape), len(outputShape))
	model := &WASMModel{
		ID:           id,
		Name:         name,
		Version:      version,
		Weights:      weights,
		Architecture: architecture,
		InputShape:   inputShape,
		OutputShape:  outputShape,
		Quantized:    false,
		LoadedAt:    time.Now(),
		LastUsed:     time.Now(),
		MemorySize:   int64(len(weights) * 4),
	}

	w.models[id] = model
	w.stats.ActiveModels.Store(int64(len(w.models)))
	w.stats.TotalMemory.Add(model.MemorySize)

	return nil
}

func (w *WASMAIInference) inferArchitecture(inputDim, outputDim int) string {
	if inputDim == 1 && outputDim == 1 {
		return "dense"
	}
	if inputDim == 2 {
		return "conv2d"
	}
	if inputDim == 3 {
		return "conv3d"
	}
	return "mlp"
}

func (w *WASMAIInference) RunInference(modelID string, input []float32) ([]float32, error) {
	start := time.Now()
	w.stats.TotalInferences.Add(1)

	w.mu.RLock()
	model, exists := w.models[modelID]
	w.mu.RUnlock()

	if !exists {
		w.stats.CacheMisses.Add(1)
		return nil, fmt.Errorf("model %s not found", modelID)
	}

	w.stats.CacheHits.Add(1)

	// Validate input shape
	if len(input) != product(model.InputShape) {
		return nil, fmt.Errorf("invalid input size: expected %d, got %d", product(model.InputShape), len(input))
	}

	// Get context from pool
	ctx := w.modelPool.Get().(*WASMInferenceContext)
	defer w.modelPool.Put(ctx)

	// Perform inference based on architecture
	var output []float32
	switch model.Architecture {
	case "dense":
		output = w.runDenseInference(input, model.Weights, model.OutputShape)
	case "conv2d":
		output = w.runConv2DInference(input, model.Weights, model.OutputShape)
	case "mlp":
		output = w.runMLPInference(input, model.Weights, model.OutputShape)
	default:
		output = w.runDenseInference(input, model.Weights, model.OutputShape)
	}

	// Update statistics
	elapsed := time.Since(start).Nanoseconds()
	oldAvg := w.stats.AvgInferenceTime.Load()
	count := w.stats.TotalInferences.Load()
	newAvg := (oldAvg*(count-1) + elapsed) / count
	w.stats.AvgInferenceTime.Store(newAvg)
	w.stats.LastUpdate.Store(time.Now())

	// Update model usage
	w.mu.Lock()
	model.LastUsed = time.Now()
	model.UseCount++
	w.mu.Unlock()

	return output, nil
}

func (w *WASMAIInference) RunBatchInference(modelID string, inputs [][]float32) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, errors.New("no inputs provided")
	}

	if len(inputs) > w.maxBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds maximum %d", len(inputs), w.maxBatchSize)
	}

	results := make([][]float32, len(inputs))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, input := range inputs {
		wg.Add(1)
		go func(idx int, in []float32) {
			defer wg.Done()
			output, err := w.RunInference(modelID, in)
			mu.Lock()
			results[idx] = output
			if firstErr == nil && err != nil {
				firstErr = err
			}
			mu.Unlock()
		}(i, input)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	w.stats.BatchHits.Add(1)
	return results, nil
}

func (w *WASMAIInference) runDenseInference(input, weights []float32, outputShape []int) []float32 {
	outputSize := product(outputShape)
	output := make([]float32, outputSize)

	// Simplified dense layer: y = Wx + b
	for i := 0; i < outputSize && i*len(input) < len(weights); i++ {
		sum := float32(0)
		for j := 0; j < len(input) && i*len(input)+j < len(weights); j++ {
			sum += input[j] * weights[i*len(input)+j]
		}
		output[i] = sigmoid(sum)
	}

	return output
}

func (w *WASMAIInference) runConv2DInference(input, weights []float32, outputShape []int) []float32 {
	outputSize := product(outputShape)
	output := make([]float32, outputSize)

	// Simplified 2D convolution
	kernelSize := int(math.Sqrt(float64(len(weights) / outputSize)))
	inputSize := int(math.Sqrt(float64(len(input))))

	for i := 0; i < outputSize && i*kernelSize*kernelSize < len(weights); i++ {
		sum := float32(0)
		for k := 0; k < kernelSize*kernelSize && i*kernelSize*kernelSize+k < len(weights); k++ {
			pos := k / kernelSize * inputSize % inputSize * inputSize
			pos += k % kernelSize
			if pos < len(input) {
				sum += input[pos] * weights[i*kernelSize*kernelSize+k]
			}
		}
		output[i] = relu(sum)
	}

	return output
}

func (w *WASMAIInference) runMLPInference(input, weights []float32, outputShape []int) []float32 {
	outputSize := product(outputShape)
	output := make([]float32, outputSize)
	
	// Multi-layer perceptron with 2 hidden layers
	hiddenSize := len(input) / 2
	if hiddenSize < 4 {
		hiddenSize = 4
	}

	layer1 := make([]float32, hiddenSize)
	for i := 0; i < hiddenSize && i*len(input) < len(weights); i++ {
		sum := float32(0)
		for j := 0; j < len(input) && i*len(input)+j < len(weights); j++ {
			sum += input[j] * weights[i*len(input)+j]
		}
		layer1[i] = relu(sum)
	}

	layer2 := make([]float32, hiddenSize)
	offset := len(input) * hiddenSize
	for i := 0; i < hiddenSize && offset+i*hiddenSize < len(weights); i++ {
		sum := float32(0)
		for j := 0; j < hiddenSize && offset+i*hiddenSize+j < len(weights); j++ {
			sum += layer1[j] * weights[offset+i*hiddenSize+j]
		}
		layer2[i] = relu(sum)
	}

	outputOffset := offset + hiddenSize*hiddenSize
	for i := 0; i < outputSize && outputOffset+i*hiddenSize < len(weights); i++ {
		sum := float32(0)
		for j := 0; j < hiddenSize && outputOffset+i*hiddenSize+j < len(weights); j++ {
			sum += layer2[j] * weights[outputOffset+i*hiddenSize+j]
		}
		output[i] = softmax(sum)
	}

	return output
}

func (w *WASMAIInference) QuantizeModel(modelID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	model, exists := w.models[modelID]
	if !exists {
		return fmt.Errorf("model %s not found", modelID)
	}

	if model.Quantized {
		return nil
	}

	// Quantize weights from float32 to int8
	quantized := w.quantizeWeights(model.Weights)
	model.Weights = nil
	model.Weights = quantized
	model.Quantized = true
	model.MemorySize = int64(len(quantized))

	w.quantizer.mu.Lock()
	w.quantizer.quantized[modelID] = quantized
	w.quantizer.mu.Unlock()

	return nil
}

func (w *WASMAIInference) quantizeWeights(weights []float32) []float32 {
	if len(weights) == 0 {
		return weights
	}

	// Find min and max for quantization
	minVal := weights[0]
	maxVal := weights[0]
	for _, w := range weights {
		if w < minVal {
			minVal = w
		}
		if w > maxVal {
			maxVal = w
		}
	}

	// Quantize to int8 range
	scale := (maxVal - minVal) / 255.0
	quantized := make([]float32, len(weights))
	for i, w := range weights {
		quantized[i] = float32(int((w-minVal)/scale)) + minVal
	}

	return quantized
}

func (w *WASMAIInference) DequantizeWeights(quantizedWeights []float32) []float32 {
	if len(quantizedWeights) == 0 {
		return quantizedWeights
	}

	// Simple dequantization (in real impl, would need scale factor)
	dequantized := make([]float32, len(quantizedWeights))
	for i, w := range quantizedWeights {
		dequantized[i] = w * 1.0
	}

	return dequantized
}

func (w *WASMAIInference) processOffloadQueue() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case task := <-w.accelerator.offloadQueue:
			go func(t InferenceTask) {
				output, err := w.RunInference(t.ModelID, t.Input)
				if err != nil {
					t.Error <- err
				} else {
					t.Output <- output
				}
			}(task)
		}
	}
}

func (w *WASMAIInference) OffloadInference(modelID string, input []float32) (<-chan []float32, <-chan error) {
	output := make(chan []float32, 1)
	errChan := make(chan error, 1)

	select {
	case w.accelerator.offloadQueue <- InferenceTask{
		ModelID: modelID,
		Input:   input,
		Output:  output,
		Error:   errChan,
	}:
	default:
		errChan <- errors.New("offload queue full")
	}

	return output, errChan
}

func (w *WASMAIInference) cleanupModels() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.mu.Lock()
			now := time.Now()
			for id, model := range w.models {
				if now.Sub(model.LastUsed) > 30*time.Minute && model.UseCount == 0 {
					w.stats.TotalMemory.Add(-model.MemorySize)
					delete(w.models, id)
					fmt.Printf("[WASMAIInference] Cleaned up model %s\n", id)
				}
			}
			w.stats.ActiveModels.Store(int64(len(w.models)))
			w.mu.Unlock()
		}
	}
}

func (w *WASMAIInference) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_inferences":  w.stats.TotalInferences.Load(),
		"cache_hits":        w.stats.CacheHits.Load(),
		"cache_misses":      w.stats.CacheMisses.Load(),
		"batch_hits":        w.stats.BatchHits.Load(),
		"avg_inference_ns":  w.stats.AvgInferenceTime.Load(),
		"total_memory":      w.stats.TotalMemory.Load(),
		"active_models":     w.stats.ActiveModels.Load(),
		"simd_enabled":      w.enableSIMD,
		"max_batch_size":    w.maxBatchSize,
		"last_update":       w.stats.LastUpdate.Load(),
	}
}

func (w *WASMAIInference) GetModelInfo(modelID string) (map[string]interface{}, error) {
	w.mu.RLock()
	model, exists := w.models[modelID]
	w.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("model %s not found", modelID)
	}

	return map[string]interface{}{
		"id":          model.ID,
		"name":        model.Name,
		"version":     model.Version,
		"architecture": model.Architecture,
		"input_shape":  model.InputShape,
		"output_shape": model.OutputShape,
		"quantized":    model.Quantized,
		"memory_size":  model.MemorySize,
		"use_count":    model.UseCount,
		"loaded_at":    model.LoadedAt,
		"last_used":    model.LastUsed,
	}, nil
}

func (w *WASMAIInference) PruneModel(modelID string, threshold float32) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	model, exists := w.models[modelID]
	if !exists {
		return fmt.Errorf("model %s not found", modelID)
	}

	// Prune small weights
	pruned := make([]float32, 0, len(model.Weights))
	for _, w := range model.Weights {
		if math.Abs(float64(w)) > float64(threshold) {
			pruned = append(pruned, w)
		}
	}

	model.Weights = pruned
	model.MemorySize = int64(len(pruned) * 4)

	return nil
}

func (w *WASMAIInference) Benchmark(modelID string, inputSize int) map[string]interface{} {
	testInput := make([]float32, inputSize)
	for i := range testInput {
		testInput[i] = rand.Float32()
	}

	const iterations = 1000

	start := time.Now()
	for i := 0; i < iterations; i++ {
		w.RunInference(modelID, testInput)
	}
	duration := time.Since(start)

	return map[string]interface{}{
		"iterations":          iterations,
		"input_size":          inputSize,
		"total_time_ms":       duration.Milliseconds(),
		"avg_time_us":         duration.Microseconds() / iterations,
		"inferences_per_sec":  float64(iterations) / duration.Seconds(),
	}
}

func product(shape []int) int {
	result := 1
	for _, dim := range shape {
		result *= dim
	}
	return result
}

func sigmoid(x float32) float32 {
	return float32(1.0 / (1.0 + math.Exp(-float64(x))))
}

func relu(x float32) float32 {
	if x > 0 {
		return x
	}
	return 0
}

func softmax(x float32) float32 {
	exp := math.Exp(float64(x))
	return float32(exp / (exp + 1.0))
}

type TensorProcessor struct {
	mu     sync.RWMutex
	tensors map[string]*Tensor
}

func NewTensorProcessor() *TensorProcessor {
	return &TensorProcessor{
		tensors: make(map[string]*Tensor),
	}
}

func (tp *TensorProcessor) CreateTensor(id string, shape []int, data []float32) *Tensor {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	tensor := &Tensor{
		Shape: shape,
		Data:  data,
	}
	tp.tensors[id] = tensor
	return tensor
}

func (tp *TensorProcessor) GetTensor(id string) (*Tensor, bool) {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	t, ok := tp.tensors[id]
	return t, ok
}

func (tp *TensorProcessor) Reshape(id string, newShape []int) error {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	tensor, ok := tp.tensors[id]
	if !ok {
		return fmt.Errorf("tensor %s not found", id)
	}

	if product(newShape) != product(tensor.Shape) {
		return fmt.Errorf("cannot reshape: size mismatch")
	}

	tensor.Shape = newShape
	return nil
}

func (tp *TensorProcessor) Transpose(id string) error {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	tensor, ok := tp.tensors[id]
	if !ok {
		return fmt.Errorf("tensor %s not found", id)
	}

	if len(tensor.Shape) != 2 {
		return fmt.Errorf("only 2D tensors can be transposed")
	}

	rows, cols := tensor.Shape[0], tensor.Shape[1]
	transposed := make([]float32, len(tensor.Data))

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			transposed[j*rows+i] = tensor.Data[i*cols+j]
		}
	}

	tensor.Shape = []int{cols, rows}
	tensor.Data = transposed

	return nil
}

func (tp *TensorProcessor) Concat(tensors []*Tensor, axis int) (*Tensor, error) {
	if len(tensors) == 0 {
		return nil, errors.New("no tensors to concatenate")
	}

	for _, t := range tensors {
		if len(t.Shape) != len(tensors[0].Shape) {
			return nil, errors.New("all tensors must have same dimensions")
		}
	}

	for i, dim := range tensors[0].Shape {
		if i == axis {
			continue
		}
		for _, t := range tensors[1:] {
			if t.Shape[i] != dim {
				return nil, errors.New("non-concat dimensions must match")
			}
		}
	}

	totalSize := 0
	for _, t := range tensors {
		totalSize += len(t.Data)
	}

	result := make([]float32, 0, totalSize)
	resultShape := make([]int, len(tensors[0].Shape))
	copy(resultShape, tensors[0].Shape)
	resultShape[axis] = 0

	for _, t := range tensors {
		result = append(result, t.Data...)
		resultShape[axis] += t.Shape[axis]
	}

	return &Tensor{Shape: resultShape, Data: result}, nil
}

func EncodeTensor(t *Tensor) []byte {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(len(t.Shape)))
	for _, dim := range t.Shape {
		dimBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(dimBytes, uint32(dim))
		data = append(data, dimBytes...)
	}
	for _, v := range t.Data {
		vBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(vBytes, math.Float32bits(v))
		data = append(data, vBytes...)
	}
	return data
}

func DecodeTensor(data []byte) (*Tensor, error) {
	if len(data) < 4 {
		return nil, errors.New("invalid tensor data")
	}

	shapeLen := int(binary.LittleEndian.Uint32(data[:4]))
	pos := 4

	shape := make([]int, shapeLen)
	for i := 0; i < shapeLen; i++ {
		if pos+4 > len(data) {
			return nil, errors.New("invalid tensor data: missing shape dimension")
		}
		shape[i] = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
		pos += 4
	}

	dataLen := product(shape)
	floatData := make([]float32, dataLen)
	for i := 0; i < dataLen; i++ {
		if pos+4 > len(data) {
			return nil, errors.New("invalid tensor data: missing float data")
		}
		floatData[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4]))
		pos += 4
	}

	return &Tensor{Shape: shape, Data: floatData}, nil
}
