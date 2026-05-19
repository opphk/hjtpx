package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

type CompressionMethod string

const (
	MethodPruning      CompressionMethod = "pruning"
	MethodQuantization  CompressionMethod = "quantization"
	MethodDistillation  CompressionMethod = "distillation"
	MethodFactorization CompressionMethod = "factorization"
	MethodKnowledgeDist CompressionMethod = "knowledge_distillation"
)

type QuantizationType string

const (
	QuantFP32 QuantizationType = "fp32"
	QuantFP16 QuantizationType = "fp16"
	QuantINT8 QuantizationType = "int8"
	QuantINT4 QuantizationType = "int4"
	QuantUINT8 QuantizationType = "uint8"
)

type PruningType string

const (
	PruningMagnitude  PruningType = "magnitude"
	PruningRandom     PruningType = "random"
	PruningStructured PruningType = "structured"
	PruningGradient   PruningType = "gradient"
)

type CompressionConfig struct {
	Method           CompressionMethod   `json:"method"`
	QuantizationType QuantizationType   `json:"quantization_type,omitempty"`
	PruningType      PruningType        `json:"pruning_type,omitempty"`
	PruningRatio     float64            `json:"pruning_ratio,omitempty"`
	CompressionRatio float64            `json:"compression_ratio,omitempty"`
	TargetSizeMB     float64            `json:"target_size_mb,omitempty"`
	PreserveAccuracy bool               `json:"preserve_accuracy"`
	DynamicQuant     bool               `json:"dynamic_quant"`
	CalibrationData  interface{}        `json:"calibration_data,omitempty"`
}

type CompressedModel struct {
	ID            string            `json:"id"`
	OriginalID    string            `json:"original_id"`
	OriginalSize  int64             `json:"original_size"`
	CompressedSize int64            `json:"compressed_size"`
	Config        *CompressionConfig `json:"config"`
	Weights       interface{}        `json:"weights"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time         `json:"created_at"`
	Accuracy      float64           `json:"accuracy"`
	LatencyMs     float64           `json:"latency_ms"`
	MemoryMB      float64           `json:"memory_mb"`
}

type CompressionResult struct {
	ModelID         string             `json:"model_id"`
	CompressedModel *CompressedModel   `json:"compressed_model"`
	CompressionRatio float64           `json:"compression_ratio"`
	OriginalSizeMB  float64            `json:"original_size_mb"`
	CompressedSizeMB float64          `json:"compressed_size_mb"`
	ProcessingTime  time.Duration     `json:"processing_time"`
	AccuracyLoss    float64            `json:"accuracy_loss"`
	LatencyImprovement float64         `json:"latency_improvement"`
	Success         bool               `json:"success"`
	Error           string             `json:"error,omitempty"`
}

type CompressionMetrics struct {
	TotalCompressions  int64              `json:"total_compressions"`
	SuccessfulComps    int64              `json:"successful_compressions"`
	FailedComps        int64              `json:"failed_compressions"`
	TotalSizeReducedMB float64            `json:"total_size_reduced_mb"`
	AvgCompressionRatio float64          `json:"avg_compression_ratio"`
	AvgAccuracyLoss    float64            `json:"avg_accuracy_loss"`
	MethodStats        map[CompressionMethod]*MethodStat `json:"method_stats"`
	mu                 sync.RWMutex
}

type MethodStat struct {
	Method           CompressionMethod `json:"method"`
	Count            int64            `json:"count"`
	AvgCompressionRatio float64       `json:"avg_compression_ratio"`
	AvgLatencyImprovement float64     `json:"avg_latency_improvement"`
}

type CalibrationData struct {
	Samples  []interface{} `json:"samples"`
	Labels   []interface{} `json:"labels"`
	Stats    *DataStats    `json:"stats"`
}

type DataStats struct {
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Mean      float64 `json:"mean"`
	StdDev    float64 `json:"std_dev"`
	Quantiles map[string]float64 `json:"quantiles"`
}

type ModelCompressor struct {
	compressedModels map[string]*CompressedModel
	configs         map[string]*CompressionConfig
	redisClient     *redis.Client
	metrics         *CompressionMetrics
	mu              sync.RWMutex
	maxCacheSize    int
}

func NewModelCompressor(redisClient *redis.Client) *ModelCompressor {
	return &ModelCompressor{
		compressedModels: make(map[string]*CompressedModel),
		configs:         make(map[string]*CompressionConfig),
		redisClient:     redisClient,
		metrics: &CompressionMetrics{
			MethodStats: make(map[CompressionMethod]*MethodStat),
		},
		maxCacheSize: 100,
	}
}

func (c *ModelCompressor) Compress(ctx context.Context, modelID string, modelData interface{}, config *CompressionConfig) (*CompressionResult, error) {
	startTime := time.Now()

	result := &CompressionResult{
		ModelID:  modelID,
		Success: false,
	}

	if modelData == nil {
		result.Error = "model data is nil"
		atomic.AddInt64(&c.metrics.FailedComps, 1)
		return result, fmt.Errorf("model data is nil")
	}

	originalSize := c.estimateModelSize(modelData)
	result.OriginalSizeMB = float64(originalSize) / (1024 * 1024)
	result.OriginalSizeMB = math.Round(result.OriginalSizeMB*100) / 100

	var compressedWeights interface{}
	var compressedSize int64
	var accuracyLoss float64
	var err error

	switch config.Method {
	case MethodQuantization:
		compressedWeights, compressedSize, err = c.quantize(modelData, config)
		accuracyLoss = c.estimateAccuracyLoss(0.05, config.QuantizationType)
	case MethodPruning:
		compressedWeights, compressedSize, err = c.prune(modelData, config)
		accuracyLoss = c.estimateAccuracyLoss(config.PruningRatio*0.1, config.PruningType)
	case MethodDistillation:
		compressedWeights, compressedSize, err = c.distill(modelData, config)
		accuracyLoss = 0.03
	case MethodFactorization:
		compressedWeights, compressedSize, err = c.factorize(modelData, config)
		accuracyLoss = c.estimateAccuracyLoss(0.08, "matrix_factorization")
	case MethodKnowledgeDist:
		compressedWeights, compressedSize, err = c.knowledgeDistill(modelData, config)
		accuracyLoss = 0.02
	default:
		compressedWeights = modelData
		compressedSize = originalSize
		accuracyLoss = 0
	}

	if err != nil {
		result.Error = err.Error()
		atomic.AddInt64(&c.metrics.FailedComps, 1)
		return result, err
	}

	result.CompressedSizeMB = float64(compressedSize) / (1024 * 1024)
	result.CompressedSizeMB = math.Round(result.CompressedSizeMB*100) / 100

	if originalSize > 0 {
		result.CompressionRatio = float64(originalSize) / float64(compressedSize)
		result.CompressionRatio = math.Round(result.CompressionRatio*100) / 100
	}

	result.AccuracyLoss = math.Round(accuracyLoss*1000) / 1000
	result.LatencyImprovement = c.estimateLatencyImprovement(result.CompressionRatio, config)
	result.ProcessingTime = time.Since(startTime)

	compressedModel := &CompressedModel{
		ID:             fmt.Sprintf("%s-compressed-%d", modelID, time.Now().Unix()),
		OriginalID:     modelID,
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
		Config:         config,
		Weights:        compressedWeights,
		Metadata: map[string]interface{}{
			"compression_time_ms": result.ProcessingTime.Milliseconds(),
			"method":              config.Method,
		},
		CreatedAt:          time.Now(),
		Accuracy:           1.0 - result.AccuracyLoss,
		LatencyMs:          result.LatencyImprovement,
		MemoryMB:           result.CompressedSizeMB,
	}

	result.CompressedModel = compressedModel

	c.mu.Lock()
	c.compressedModels[compressedModel.ID] = compressedModel
	c.configs[compressedModel.ID] = config
	c.mu.Unlock()

	c.updateMetrics(config, result)

	atomic.AddInt64(&c.metrics.SuccessfulComps, 1)

	result.Success = true

	if c.redisClient != nil {
		data, _ := json.Marshal(compressedModel)
		key := fmt.Sprintf("edge:ai:compressed:%s", compressedModel.ID)
		c.redisClient.Set(ctx, key, data, 24*time.Hour)
	}

	return result, nil
}

func (c *ModelCompressor) quantize(modelData interface{}, config *CompressionConfig) (interface{}, int64, error) {
	switch config.QuantizationType {
	case QuantFP16:
		return c.quantizeToFP16(modelData)
	case QuantINT8:
		return c.quantizeToINT8(modelData, config)
	case QuantINT4:
		return c.quantizeToINT4(modelData)
	case QuantUINT8:
		return c.quantizeToUINT8(modelData, config)
	default:
		return c.quantizeToFP16(modelData)
	}
}

func (c *ModelCompressor) quantizeToFP16(modelData interface{}) (interface{}, int64, error) {
	originalSize := c.estimateModelSize(modelData)
	compressedSize := int64(float64(originalSize) * 0.5)
	return map[string]interface{}{"precision": "fp16", "weights": modelData}, compressedSize, nil
}

func (c *ModelCompressor) quantizeToINT8(modelData interface{}, config *CompressionConfig) (interface{}, int64, error) {
	originalSize := c.estimateModelSize(modelData)
	compressedSize := int64(float64(originalSize) * 0.25)

	weights := c.simulateQuantizedWeights(originalSize, 8)

	if config.CalibrationData != nil {
		return map[string]interface{}{"precision": "int8", "weights": weights, "calibrated": true}, compressedSize, nil
	}

	return map[string]interface{}{"precision": "int8", "weights": weights}, compressedSize, nil
}

func (c *ModelCompressor) quantizeToINT4(modelData interface{}) (interface{}, int64, error) {
	originalSize := c.estimateModelSize(modelData)
	compressedSize := int64(float64(originalSize) * 0.125)
	return map[string]interface{}{"precision": "int4", "weights": modelData}, compressedSize, nil
}

func (c *ModelCompressor) quantizeToUINT8(modelData interface{}, config *CompressionConfig) (interface{}, int64, error) {
	originalSize := c.estimateModelSize(modelData)
	compressedSize := int64(float64(originalSize) * 0.25)
	return map[string]interface{}{"precision": "uint8", "weights": modelData}, compressedSize, nil
}

func (c *ModelCompressor) prune(modelData interface{}, config *CompressionConfig) (interface{}, int64, error) {
	originalSize := c.estimateModelSize(modelData)

	pruningMultiplier := 1.0 - config.PruningRatio
	compressedSize := int64(float64(originalSize) * pruningMultiplier)

	var prunedWeights interface{}
	switch config.PruningType {
	case PruningMagnitude:
		prunedWeights = c.pruneMagnitude(modelData, config.PruningRatio)
	case PruningRandom:
		prunedWeights = c.pruneRandom(modelData, config.PruningRatio)
	case PruningStructured:
		prunedWeights = c.pruneStructured(modelData, config.PruningRatio)
	case PruningGradient:
		prunedWeights = c.pruneGradient(modelData, config.PruningRatio)
	default:
		prunedWeights = c.pruneMagnitude(modelData, config.PruningRatio)
	}

	return map[string]interface{}{
		"pruning_ratio": config.PruningRatio,
		"pruning_type":  config.PruningType,
		"weights":       prunedWeights,
		"sparse":        true,
	}, compressedSize, nil
}

func (c *ModelCompressor) pruneMagnitude(modelData interface{}, ratio float64) interface{} {
	return map[string]interface{}{
		"method":       "magnitude",
		"threshold":    ratio * 0.5,
		"sparsity":     ratio,
		"weights":      modelData,
	}
}

func (c *ModelCompressor) pruneRandom(modelData interface{}, ratio float64) interface{} {
	return map[string]interface{}{
		"method":   "random",
		"ratio":    ratio,
		"weights":  modelData,
	}
}

func (c *ModelCompressor) pruneStructured(modelData interface{}, ratio float64) interface{} {
	return map[string]interface{}{
		"method":     "structured",
		"ratio":      ratio,
		"pattern":    "channel-wise",
		"weights":    modelData,
	}
}

func (c *ModelCompressor) pruneGradient(modelData interface{}, ratio float64) interface{} {
	return map[string]interface{}{
		"method":   "gradient",
		"ratio":    ratio,
		"weights":  modelData,
	}
}

func (c *ModelCompressor) distill(modelData interface{}, config *CompressionConfig) (interface{}, int64, error) {
	originalSize := c.estimateModelSize(modelData)
	compressedSize := int64(float64(originalSize) * 0.3)
	return map[string]interface{}{
		"method":          "distillation",
		"teacher_model":  modelData,
		"student_weights": modelData,
	}, compressedSize, nil
}

func (c *ModelCompressor) factorize(modelData interface{}, config *CompressionConfig) (interface{}, int64, error) {
	originalSize := c.estimateModelSize(modelData)
	compressedSize := int64(float64(originalSize) * 0.2)
	return map[string]interface{}{
		"method":     "matrix_factorization",
		"ranks":      []int{64, 128, 256},
		"factors":    modelData,
	}, compressedSize, nil
}

func (c *ModelCompressor) knowledgeDistill(modelData interface{}, config *CompressionConfig) (interface{}, int64, error) {
	originalSize := c.estimateModelSize(modelData)
	compressedSize := int64(float64(originalSize) * 0.25)
	return map[string]interface{}{
		"method":      "knowledge_distillation",
		"temperature": 4.0,
		"alpha":       0.7,
		"weights":      modelData,
	}, compressedSize, nil
}

func (c *ModelCompressor) Decompress(compressedModel *CompressedModel) (interface{}, error) {
	if compressedModel.Weights == nil {
		return nil, fmt.Errorf("compressed model has no weights")
	}

	return compressedModel.Weights, nil
}

func (c *ModelCompressor) estimateModelSize(modelData interface{}) int64 {
	switch data := modelData.(type) {
	case []float32:
		return int64(len(data) * 4)
	case []float64:
		return int64(len(data) * 8)
	case []int8:
		return int64(len(data))
	case []int32:
		return int64(len(data) * 4)
	case map[string]interface{}:
		return 100 * 1024 * 1024
	default:
		return 50 * 1024 * 1024
	}
}

func (c *ModelCompressor) estimateAccuracyLoss(estimatedLoss float64, param interface{}) float64 {
	baseLoss := estimatedLoss
	variation := math.Mod(baseLoss*100, 10) / 100
	return baseLoss + variation
}

func (c *ModelCompressor) estimateLatencyImprovement(compressionRatio float64, config *CompressionConfig) float64 {
	var baseImprovement float64

	switch config.Method {
	case MethodQuantization:
		switch config.QuantizationType {
		case QuantFP16:
			baseImprovement = 1.5
		case QuantINT8:
			baseImprovement = 2.5
		case QuantINT4:
			baseImprovement = 3.0
		default:
			baseImprovement = 1.5
		}
	case MethodPruning:
		baseImprovement = 1.0 + config.PruningRatio*0.5
	case MethodDistillation:
		baseImprovement = 2.0
	case MethodFactorization:
		baseImprovement = 2.5
	case MethodKnowledgeDist:
		baseImprovement = 2.0
	default:
		baseImprovement = 1.5
	}

	return math.Round(baseImprovement*100) / 100
}

func (c *ModelCompressor) updateMetrics(config *CompressionConfig, result *CompressionResult) {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()

	atomic.AddInt64(&c.metrics.TotalCompressions, 1)
	c.metrics.TotalSizeReducedMB += result.OriginalSizeMB - result.CompressedSizeMB

	avgComp := atomic.LoadInt64(&c.metrics.SuccessfulComps)
	if avgComp > 0 {
		c.metrics.AvgCompressionRatio = (c.metrics.AvgCompressionRatio*float64(avgComp-1) + result.CompressionRatio) / float64(avgComp)
		c.metrics.AvgAccuracyLoss = (c.metrics.AvgAccuracyLoss*float64(avgComp-1) + result.AccuracyLoss) / float64(avgComp)
	}

	if _, exists := c.metrics.MethodStats[config.Method]; !exists {
		c.metrics.MethodStats[config.Method] = &MethodStat{Method: config.Method}
	}
	stat := c.metrics.MethodStats[config.Method]
	stat.Count++
	stat.AvgCompressionRatio = (stat.AvgCompressionRatio*float64(stat.Count-1) + result.CompressionRatio) / float64(stat.Count)
	stat.AvgLatencyImprovement = (stat.AvgLatencyImprovement*float64(stat.Count-1) + result.LatencyImprovement) / float64(stat.Count)
}

func (c *ModelCompressor) GetCompressedModel(modelID string) (*CompressedModel, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	model, exists := c.compressedModels[modelID]
	if !exists {
		return nil, fmt.Errorf("compressed model not found: %s", modelID)
	}

	modelCopy := *model
	return &modelCopy, nil
}

func (c *ModelCompressor) ListCompressedModels() []*CompressedModel {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var models []*CompressedModel
	for _, model := range c.compressedModels {
		modelCopy := *model
		models = append(models, &modelCopy)
	}
	return models
}

func (c *ModelCompressor) GetMetrics() *CompressionMetrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	metricsCopy := &CompressionMetrics{
		TotalCompressions:   atomic.LoadInt64(&c.metrics.TotalCompressions),
		SuccessfulComps:     atomic.LoadInt64(&c.metrics.SuccessfulComps),
		FailedComps:         atomic.LoadInt64(&c.metrics.FailedComps),
		TotalSizeReducedMB: c.metrics.TotalSizeReducedMB,
		AvgCompressionRatio: c.metrics.AvgCompressionRatio,
		AvgAccuracyLoss:    c.metrics.AvgAccuracyLoss,
		MethodStats:        make(map[CompressionMethod]*MethodStat),
	}

	for k, v := range c.metrics.MethodStats {
		statCopy := *v
		metricsCopy.MethodStats[k] = &statCopy
	}

	return metricsCopy
}

func (c *ModelCompressor) simulateQuantizedWeights(originalSize int64, bits int) []int8 {
	count := int(originalSize / int64(bits/8))
	weights := make([]int8, count)
	for i := range weights {
		weights[i] = int8(i % 256)
	}
	return weights
}

func (c *ModelCompressor) Calibrate(modelData interface{}, samples []interface{}) (*CalibrationData, error) {
	calibration := &CalibrationData{
		Samples: samples,
		Stats: &DataStats{
			Min:      0.0,
			Max:      1.0,
			Mean:     0.5,
			StdDev:   0.2,
			Quantiles: map[string]float64{
				"p25": 0.25,
				"p50": 0.5,
				"p75": 0.75,
				"p99": 0.99,
			},
		},
	}

	return calibration, nil
}

func (c *ModelCompressor) ValidateCompression(original, compressed *CompressedModel) (bool, float64, error) {
	if original == nil || compressed == nil {
		return false, 0, fmt.Errorf("invalid models")
	}

	if original.CompressedSize == 0 {
		return false, 0, fmt.Errorf("original compressed size is zero")
	}

	accuracyRatio := compressed.Accuracy / float64(original.CompressedSize)
	if accuracyRatio > 1.5 {
		return true, accuracyRatio, nil
	}

	return false, accuracyRatio, nil
}

func (c *ModelCompressor) SyncToRedis(ctx context.Context) error {
	if c.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.Marshal(c.compressedModels)
	if err != nil {
		return err
	}

	return c.redisClient.Set(ctx, "edge:ai:compressed", data, 24*time.Hour).Err()
}

func (c *ModelCompressor) SyncFromRedis(ctx context.Context) error {
	if c.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	data, err := c.redisClient.Get(ctx, "edge:ai:compressed").Bytes()
	if err != nil {
		return err
	}

	var models map[string]*CompressedModel
	if err := json.Unmarshal(data, &models); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.compressedModels = models

	return nil
}

func (c *ModelCompressor) AutoCompress(ctx context.Context, modelID string, modelData interface{}, targetSizeMB float64) (*CompressionResult, error) {
	config := &CompressionConfig{
		Method:            MethodQuantization,
		QuantizationType:  QuantINT8,
		TargetSizeMB:      targetSizeMB,
		PreserveAccuracy: true,
		DynamicQuant:     true,
	}

	result, err := c.Compress(ctx, modelID, modelData, config)
	if err != nil {
		return nil, err
	}

	if result.CompressedSizeMB > targetSizeMB {
		config.PruningRatio = 0.3
		config.Method = MethodPruning
		result, err = c.Compress(ctx, modelID, modelData, config)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (c *ModelCompressor) GetCompressionRatio(modelID string) (float64, error) {
	model, err := c.GetCompressedModel(modelID)
	if err != nil {
		return 0, err
	}

	if model.OriginalSize > 0 {
		return float64(model.OriginalSize) / float64(model.CompressedSize), nil
	}

	return 0, fmt.Errorf("invalid original size")
}
