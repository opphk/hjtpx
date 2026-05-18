package trace

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type TrajectoryNNService struct {
	lstmExtractor          *LSTMFeatureExtractor
	transformerPredictor   *TransformerPredictor
	modelUpdateService     *ModelUpdateService
	mu                     sync.RWMutex
	isInitialized          bool
	lastInferenceTime      time.Time
	inferenceCount         int64
	enableUpdateMonitoring bool
}

type NNInferenceResult struct {
	RiskScore        float64           `json:"risk_score"`
	BotProbability   float64           `json:"bot_probability"`
	HumanProbability float64           `json:"human_probability"`
	Confidence       float64           `json:"confidence"`
	ModelVersion     string            `json:"model_version"`
	LatencyMs        float64           `json:"latency_ms"`
	NNFeatures       map[string]float64 `json:"nn_features"`
}

type ModelInfo struct {
	ModelType    string    `json:"model_type"`
	VersionID    string    `json:"version_id"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	IsActive     bool      `json:"is_active"`
}

func NewTrajectoryNNService() *TrajectoryNNService {
	return &TrajectoryNNService{
		lstmExtractor:          NewLSTMFeatureExtractor(),
		transformerPredictor:   NewTransformerPredictor(),
		modelUpdateService:     NewModelUpdateService(),
		isInitialized:          true,
		enableUpdateMonitoring: true,
	}
}

func (s *TrajectoryNNService) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.modelUpdateService != nil {
		s.modelUpdateService.Start(ctx)
	}

	s.registerModelUpdateCallbacks()

	s.isInitialized = true
	return nil
}

func (s *TrajectoryNNService) registerModelUpdateCallbacks() {
	if s.modelUpdateService == nil {
		return
	}

	s.modelUpdateService.RegisterUpdateCallback("lstm", func(modelType string, oldVersion, newVersion *ModelVersion) error {
		if oldVersion != nil {
			fmt.Printf("[ModelUpdate] LSTM model updated from %s to %s\n", oldVersion.VersionID, newVersion.VersionID)
		}
		return nil
	})

	s.modelUpdateService.RegisterUpdateCallback("transformer", func(modelType string, oldVersion, newVersion *ModelVersion) error {
		if oldVersion != nil {
			fmt.Printf("[ModelUpdate] Transformer model updated from %s to %s\n", oldVersion.VersionID, newVersion.VersionID)
		}
		return nil
	})

	s.modelUpdateService.RegisterRollbackCallback("lstm", func(modelType string, currentVersion, rollbackVersion *ModelVersion) error {
		fmt.Printf("[ModelRollback] Rolling back LSTM model from %s to %s\n", currentVersion.VersionID, rollbackVersion.VersionID)
		return nil
	})

	s.modelUpdateService.RegisterRollbackCallback("transformer", func(modelType string, currentVersion, rollbackVersion *ModelVersion) error {
		fmt.Printf("[ModelRollback] Rolling back Transformer model from %s to %s\n", currentVersion.VersionID, rollbackVersion.VersionID)
		return nil
	})
}

func (s *TrajectoryNNService) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.modelUpdateService != nil {
		s.modelUpdateService.Stop()
	}

	s.isInitialized = false
}

func (s *TrajectoryNNService) Predict(ctx context.Context, traceData *model.TraceData) (*NNInferenceResult, error) {
	if !s.isInitialized {
		return nil, fmt.Errorf("service not initialized")
	}

	if traceData == nil || len(traceData.Points) < 2 {
		return nil, fmt.Errorf("invalid trace data")
	}

	startTime := time.Now()

	lstmFeatures, err := s.extractLSTMFeatures(traceData)
	if err != nil {
		return nil, fmt.Errorf("LSTM feature extraction failed: %w", err)
	}

	transformerResult, err := s.transformerPredictor.PredictTrajectory(traceData)
	if err != nil {
		return nil, fmt.Errorf("transformer prediction failed: %w", err)
	}

	combinedResult := s.combinePredictions(lstmFeatures, transformerResult)

	latencyMs := float64(time.Since(startTime).Microseconds()) / 1000.0
	combinedResult.LatencyMs = latencyMs

	s.updateInferenceMetrics(ctx, combinedResult)

	s.mu.Lock()
	s.lastInferenceTime = time.Now()
	s.inferenceCount++
	s.mu.Unlock()

	return combinedResult, nil
}

func (s *TrajectoryNNService) extractLSTMFeatures(traceData *model.TraceData) (map[string]float64, error) {
	if s.lstmExtractor == nil {
		return nil, fmt.Errorf("LSTM extractor not initialized")
	}

	return s.lstmExtractor.ExtractRiskFeatures(traceData)
}

func (s *TrajectoryNNService) combinePredictions(lstmFeatures map[string]float64, transformerResult *TransformerRiskPrediction) *NNInferenceResult {
	result := &NNInferenceResult{
		NNFeatures: make(map[string]float64),
		ModelVersion: "combined-v1",
	}

	if transformerResult != nil {
		result.RiskScore = transformerResult.RiskScore
		result.BotProbability = transformerResult.BotProbability
		result.HumanProbability = transformerResult.HumanProbability
		result.Confidence = transformerResult.Confidence

		for k, v := range transformerResult.FeatureImportance {
			result.NNFeatures["transformer_"+k] = v
		}
	}

	if lstmFeatures != nil {
		for k, v := range lstmFeatures {
			result.NNFeatures["lstm_"+k] = v
		}

		lstmRiskScore := s.calculateLSTMRiskScore(lstmFeatures)
		if result.RiskScore > 0 {
			result.RiskScore = 0.6*result.RiskScore + 0.4*lstmRiskScore
		} else {
			result.RiskScore = lstmRiskScore
		}
	}

	if result.RiskScore > 1.0 {
		result.RiskScore = 1.0
	}
	if result.RiskScore < 0.0 {
		result.RiskScore = 0.0
	}

	if result.BotProbability > 1.0 {
		result.BotProbability = 1.0
	}
	if result.HumanProbability < 0.0 {
		result.HumanProbability = 0.0
	}

	return result
}

func (s *TrajectoryNNService) calculateLSTMRiskScore(features map[string]float64) float64 {
	var totalScore float64
	var weightSum float64

	weights := map[string]float64{
		"velocity_mean":        0.15,
		"velocity_variance":    0.10,
		"acceleration_mean":    0.15,
		"acceleration_max":     0.10,
		"direction_change_rate": 0.20,
		"curvature_mean":       0.15,
		"temporal_interval_mean": 0.05,
		"temporal_interval_variance": 0.05,
		"pause_ratio":          0.05,
	}

	for key, weight := range weights {
		if val, ok := features[key]; ok {
			normalizedVal := s.normalizeValue(val, key)
			totalScore += normalizedVal * weight
			weightSum += weight
		}
	}

	if weightSum > 0 {
		return totalScore / weightSum
	}

	return 0.5
}

func (s *TrajectoryNNService) normalizeValue(val float64, featureName string) float64 {
	thresholds := map[string]struct {
		Min, Max float64
	}{
		"velocity_mean":            {0, 100},
		"velocity_variance":        {0, 50},
		"acceleration_mean":        {0, 10},
		"acceleration_max":         {0, 50},
		"direction_change_rate":   {0, 1},
		"curvature_mean":           {0, 3.14},
		"temporal_interval_mean":  {0, 500},
		"temporal_interval_variance": {0, 100},
		"pause_ratio":              {0, 1},
	}

	if threshold, ok := thresholds[featureName]; ok {
		normalized := (val - threshold.Min) / (threshold.Max - threshold.Min)
		if normalized < 0 {
			normalized = 0
		}
		if normalized > 1 {
			normalized = 1
		}
		return normalized
	}

	return val
}

func (s *TrajectoryNNService) updateInferenceMetrics(ctx context.Context, result *NNInferenceResult) {
	if s.modelUpdateService == nil || !s.enableUpdateMonitoring {
		return
	}

	modelTypes := []string{"lstm", "transformer"}
	for _, modelType := range modelTypes {
		actual := result.RiskScore
		if actual > 0.5 {
			actual = 1.0
		} else {
			actual = 0.0
		}

		update := ModelMetricsUpdate{
			ModelType:  modelType,
			VersionID:  result.ModelVersion,
			Prediction: result.BotProbability,
			Actual:     actual,
			LatencyMs:  result.LatencyMs,
			IsError:    false,
		}

		_ = s.modelUpdateService.UpdateMetrics(ctx, update)
	}
}

func (s *TrajectoryNNService) RegisterNewModel(ctx context.Context, modelType, versionID, modelPath, checksum, createdBy string, metadata map[string]interface{}) error {
	if s.modelUpdateService == nil {
		return fmt.Errorf("model update service not initialized")
	}

	reg := ModelRegistration{
		ModelType: modelType,
		VersionID: versionID,
		ModelPath: modelPath,
		Checksum:  checksum,
		CreatedBy: createdBy,
		Metadata:  metadata,
	}

	_, err := s.modelUpdateService.RegisterModel(ctx, reg)
	return err
}

func (s *TrajectoryNNService) ActivateModelVersion(ctx context.Context, modelType, versionID string) error {
	if s.modelUpdateService == nil {
		return fmt.Errorf("model update service not initialized")
	}

	result, err := s.modelUpdateService.ActivateVersion(ctx, modelType, versionID)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("failed to activate model: %s", result.Message)
	}

	return nil
}

func (s *TrajectoryNNService) GetModelVersion(ctx context.Context, modelType string) (*ModelVersion, error) {
	if s.modelUpdateService == nil {
		return nil, fmt.Errorf("model update service not initialized")
	}

	return s.modelUpdateService.GetCurrentVersion(modelType)
}

func (s *TrajectoryNNService) ListModelVersions(ctx context.Context, modelType string, limit int) ([]*ModelVersion, error) {
	if s.modelUpdateService == nil {
		return nil, fmt.Errorf("model update service not initialized")
	}

	query := VersionQuery{
		ModelType: modelType,
		Limit:     limit,
	}

	return s.modelUpdateService.ListVersions(query)
}

func (s *TrajectoryNNService) GetModelMetrics(ctx context.Context, modelType string) (*PerformanceMetrics, error) {
	if s.modelUpdateService == nil {
		return nil, fmt.Errorf("model update service not initialized")
	}

	return s.modelUpdateService.GetPerformanceMetrics(modelType)
}

func (s *TrajectoryNNService) PerformHealthCheck(ctx context.Context, modelType string) (*HealthCheckResult, error) {
	if s.modelUpdateService == nil {
		return nil, fmt.Errorf("model update service not initialized")
	}

	return s.modelUpdateService.PerformHealthCheck(ctx, modelType)
}

func (s *TrajectoryNNService) TriggerModelRollback(ctx context.Context, modelType, reason string) error {
	if s.modelUpdateService == nil {
		return fmt.Errorf("model update service not initialized")
	}

	result, err := s.modelUpdateService.TriggerRollback(ctx, modelType, reason)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("rollback failed: %s", result.Message)
	}

	return nil
}

func (s *TrajectoryNNService) MarkModelStable(ctx context.Context, modelType, versionID string) error {
	if s.modelUpdateService == nil {
		return fmt.Errorf("model update service not initialized")
	}

	return s.modelUpdateService.MarkVersionStable(ctx, modelType, versionID)
}

func (s *TrajectoryNNService) GetModelUpdateService() *ModelUpdateService {
	return s.modelUpdateService
}

func (s *TrajectoryNNService) SetUpdateMonitoring(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enableUpdateMonitoring = enabled
}

func (s *TrajectoryNNService) GetServiceStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["is_initialized"] = s.isInitialized
	stats["inference_count"] = s.inferenceCount
	stats["enable_update_monitoring"] = s.enableUpdateMonitoring

	if !s.lastInferenceTime.IsZero() {
		stats["last_inference_time"] = s.lastInferenceTime
		stats["time_since_last_inference"] = time.Since(s.lastInferenceTime).String()
	}

	if s.modelUpdateService != nil {
		s.mu.RUnlock()
		s.mu.RLock()
		cfg := s.modelUpdateService.GetConfig()
		stats["auto_rollback_enabled"] = cfg.AutoRollbackEnabled
		stats["health_check_interval"] = cfg.HealthCheckInterval.String()
	}

	return stats
}

func (s *TrajectoryNNService) GetAllModelInfo(ctx context.Context) ([]ModelInfo, error) {
	modelTypes := []string{"lstm", "transformer"}
	infoList := make([]ModelInfo, 0, len(modelTypes))

	for _, modelType := range modelTypes {
		version, err := s.modelUpdateService.GetCurrentVersion(modelType)
		if err != nil {
			continue
		}

		info := ModelInfo{
			ModelType:  modelType,
			VersionID:  version.VersionID,
			Status:     string(version.Status),
			CreatedAt:  version.CreatedAt,
			IsActive:   version.Status == ModelStatusActive,
		}
		infoList = append(infoList, info)
	}

	return infoList, nil
}

func (s *TrajectoryNNService) CompareModelVersions(ctx context.Context, modelType, versionID1, versionID2 string) (map[string]interface{}, error) {
	if s.modelUpdateService == nil {
		return nil, fmt.Errorf("model update service not initialized")
	}

	return s.modelUpdateService.CompareVersions(modelType, versionID1, versionID2)
}

func (s *TrajectoryNNService) ExportVersionHistory(ctx context.Context, modelType string) ([]byte, error) {
	if s.modelUpdateService == nil {
		return nil, fmt.Errorf("model update service not initialized")
	}

	return s.modelUpdateService.ExportVersionHistory(modelType)
}
