package trace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type ModelUpdateService struct {
	mu sync.RWMutex

	modelRegistry     map[string]*ModelVersion
	currentVersions   map[string]string
	versionHistory    map[string][]*ModelVersion
	performanceMetrics map[string]*PerformanceMetrics
	healthCheckResults map[string]*HealthCheckResult

	updateCallbacks   map[string]ModelUpdateCallback
	rollbackCallbacks map[string]ModelRollbackCallback

	config        *UpdateConfig
	lastCheckTime time.Time
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

type ModelVersion struct {
	VersionID     string                 `json:"version_id"`
	ModelType     string                 `json:"model_type"`
	ModelPath     string                 `json:"model_path"`
	Checksum      string                 `json:"checksum"`
	CreatedAt     time.Time              `json:"created_at"`
	CreatedBy     string                 `json:"created_by"`
	Status       ModelStatus            `json:"status"`
	Metadata     map[string]interface{} `json:"metadata"`
	ParentVersion string                 `json:"parent_version,omitempty"`
	RollbackFrom  string                 `json:"rollback_from,omitempty"`
	RollbackCount int                    `json:"rollback_count"`
}

type ModelStatus string

const (
	ModelStatusActive        ModelStatus = "active"
	ModelStatusStaging       ModelStatus = "staging"
	ModelStatusStable        ModelStatus = "stable"
	ModelStatusDeprecated    ModelStatus = "deprecated"
	ModelStatusRollbacking   ModelStatus = "rollbacking"
	ModelStatusRollbackFailed ModelStatus = "rollback_failed"
	ModelStatusFailed        ModelStatus = "failed"
)

type PerformanceMetrics struct {
	ModelType       string           `json:"model_type"`
	VersionID       string           `json:"version_id"`
	Accuracy        float64          `json:"accuracy"`
	Precision       float64          `json:"precision"`
	Recall          float64          `json:"recall"`
	F1Score         float64          `json:"f1_score"`
	LatencyP50      float64          `json:"latency_p50"`
	LatencyP95      float64          `json:"latency_p95"`
	LatencyP99      float64          `json:"latency_p99"`
	Throughput      float64          `json:"throughput"`
	ErrorRate       float64          `json:"error_rate"`
	SampleCount     int64            `json:"sample_count"`
	TruePositive    int64            `json:"true_positive"`
	FalsePositive   int64            `json:"false_positive"`
	TrueNegative    int64            `json:"true_negative"`
	FalseNegative   int64            `json:"false_negative"`
	LastUpdated     time.Time        `json:"last_updated"`
	MetricsHistory  []MetricSnapshot `json:"metrics_history"`
	mu              sync.RWMutex
}

type MetricSnapshot struct {
	Timestamp    time.Time `json:"timestamp"`
	Accuracy     float64   `json:"accuracy"`
	LatencyP95   float64   `json:"latency_p95"`
	ErrorRate    float64   `json:"error_rate"`
	SampleCount  int64     `json:"sample_count"`
}

type HealthCheckResult struct {
	ModelType    string         `json:"model_type"`
	VersionID    string         `json:"version_id"`
	IsHealthy   bool           `json:"is_healthy"`
	Score        float64        `json:"score"`
	CheckTime    time.Time      `json:"check_time"`
	Issues       []HealthIssue  `json:"issues"`
	Recommendations []string    `json:"recommendations"`
}

type HealthIssue struct {
	Severity  string `json:"severity"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Detail    string `json:"detail,omitempty"`
}

type UpdateConfig struct {
	AutoRollbackEnabled      bool          `json:"auto_rollback_enabled"`
	RollbackThresholdAccuracy float64       `json:"rollback_threshold_accuracy"`
	RollbackThresholdLatency float64       `json:"rollback_threshold_latency"`
	RollbackThresholdErrorRate float64      `json:"rollback_threshold_error_rate"`
	HealthCheckInterval      time.Duration `json:"health_check_interval"`
	MetricsWindowSize        int           `json:"metrics_window_size"`
	MinSamplesForHealthCheck int           `json:"min_samples_for_health_check"`
	StagingDuration          time.Duration `json:"staging_duration"`
	MaxRollbackAttempts      int           `json:"max_rollback_attempts"`
	EnableMetricsHistory     bool          `json:"enable_metrics_history"`
}

type ModelUpdateCallback func(modelType string, oldVersion, newVersion *ModelVersion) error
type ModelRollbackCallback func(modelType string, currentVersion, rollbackVersion *ModelVersion) error

type ModelRegistration struct {
	ModelType string                 `json:"model_type"`
	VersionID  string                 `json:"version_id"`
	ModelPath string                 `json:"model_path"`
	Checksum  string                 `json:"checksum"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedBy string                 `json:"created_by"`
}

type UpdateResult struct {
	Success         bool          `json:"success"`
	OldVersionID    string        `json:"old_version_id,omitempty"`
	NewVersionID    string        `json:"new_version_id,omitempty"`
	Message         string        `json:"message"`
	Error           error         `json:"error,omitempty"`
	RollbackTriggered bool        `json:"rollback_triggered"`
	RollbackVersion  string       `json:"rollback_version,omitempty"`
}

type VersionQuery struct {
	ModelType    string
	VersionID    string
	Status       ModelStatus
	Limit        int
	Offset       int
	StartTime    time.Time
	EndTime      time.Time
	IncludeStats bool
}

type ModelMetricsUpdate struct {
	ModelType   string  `json:"model_type"`
	VersionID   string  `json:"version_id"`
	Prediction  float64 `json:"prediction"`
	Actual      float64 `json:"actual"`
	LatencyMs   float64 `json:"latency_ms"`
	IsError     bool    `json:"is_error"`
}

func NewModelUpdateService() *ModelUpdateService {
	return &ModelUpdateService{
		modelRegistry:      make(map[string]*ModelVersion),
		currentVersions:   make(map[string]string),
		versionHistory:    make(map[string][]*ModelVersion),
		performanceMetrics: make(map[string]*PerformanceMetrics),
		healthCheckResults: make(map[string]*HealthCheckResult),
		updateCallbacks:   make(map[string]ModelUpdateCallback),
		rollbackCallbacks: make(map[string]ModelRollbackCallback),
		config: &UpdateConfig{
			AutoRollbackEnabled:       true,
			RollbackThresholdAccuracy: 0.05,
			RollbackThresholdLatency: 2.0,
			RollbackThresholdErrorRate: 0.02,
			HealthCheckInterval:       5 * time.Minute,
			MetricsWindowSize:         1000,
			MinSamplesForHealthCheck:   100,
			StagingDuration:           30 * time.Minute,
			MaxRollbackAttempts:        3,
			EnableMetricsHistory:       true,
		},
		stopChan: make(chan struct{}),
	}
}

func (s *ModelUpdateService) Start(ctx context.Context) {
	s.wg.Add(1)
	go s.healthCheckLoop(ctx)
}

func (s *ModelUpdateService) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

func (s *ModelUpdateService) RegisterModel(ctx context.Context, reg ModelRegistration) (*ModelVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if reg.ModelType == "" {
		return nil, errors.New("model type is required")
	}
	if reg.VersionID == "" {
		return nil, errors.New("version ID is required")
	}
	if reg.ModelPath == "" {
		return nil, errors.New("model path is required")
	}

	key := s.modelKey(reg.ModelType, reg.VersionID)
	if _, exists := s.modelRegistry[key]; exists {
		return nil, fmt.Errorf("model version already exists: %s", reg.VersionID)
	}

	parentVersion := ""
	if currentV, ok := s.currentVersions[reg.ModelType]; ok {
		parentVersion = currentV
	}

	version := &ModelVersion{
		VersionID:     reg.VersionID,
		ModelType:     reg.ModelType,
		ModelPath:     reg.ModelPath,
		Checksum:      reg.Checksum,
		CreatedAt:     time.Now(),
		CreatedBy:     reg.CreatedBy,
		Status:        ModelStatusStaging,
		Metadata:      reg.Metadata,
		ParentVersion: parentVersion,
		RollbackCount: 0,
	}

	s.modelRegistry[key] = version
	s.versionHistory[reg.ModelType] = append(s.versionHistory[reg.ModelType], version)

	if s.performanceMetrics[reg.ModelType] == nil {
		s.performanceMetrics[reg.ModelType] = &PerformanceMetrics{
			ModelType: reg.ModelType,
			MetricsHistory: make([]MetricSnapshot, 0),
		}
	}
	s.performanceMetrics[reg.ModelType].VersionID = reg.VersionID

	return version, nil
}

func (s *ModelUpdateService) ActivateVersion(ctx context.Context, modelType, versionID string) (*UpdateResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.activateVersionInternal(ctx, modelType, versionID)
}

func (s *ModelUpdateService) activateVersionInternal(ctx context.Context, modelType, versionID string) (*UpdateResult, error) {
	key := s.modelKey(modelType, versionID)
	version, exists := s.modelRegistry[key]
	if !exists {
		return &UpdateResult{
			Success: false,
			Message: fmt.Sprintf("model version not found: %s", versionID),
		}, fmt.Errorf("model version not found: %s", versionID)
	}

	oldVersionID := s.currentVersions[modelType]
	var oldVersion *ModelVersion
	if oldVersionID != "" {
		oldKey := s.modelKey(modelType, oldVersionID)
		oldVersion = s.modelRegistry[oldKey]
		if oldVersion != nil && oldVersion.Status == ModelStatusActive {
			oldVersion.Status = ModelStatusDeprecated
		}
	}

	version.Status = ModelStatusActive
	s.currentVersions[modelType] = versionID

	if callback, ok := s.updateCallbacks[modelType]; ok {
		s.mu.Unlock()
		err := callback(modelType, oldVersion, version)
		s.mu.Lock()
		if err != nil {
			version.Status = ModelStatusFailed
			s.currentVersions[modelType] = oldVersionID
			if oldVersion != nil {
				oldVersion.Status = ModelStatusActive
			}
			return &UpdateResult{
				Success: false,
				Message: fmt.Sprintf("update callback failed: %v", err),
				Error:   err,
			}, err
		}
	}

	return &UpdateResult{
		Success:      true,
		OldVersionID: oldVersionID,
		NewVersionID: versionID,
		Message:      "model version activated successfully",
	}, nil
}

func (s *ModelUpdateService) GetVersion(modelType, versionID string) (*ModelVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.modelKey(modelType, versionID)
	version, exists := s.modelRegistry[key]
	if !exists {
		return nil, fmt.Errorf("model version not found: %s", versionID)
	}

	result := *version
	return &result, nil
}

func (s *ModelUpdateService) GetCurrentVersion(modelType string) (*ModelVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	versionID, exists := s.currentVersions[modelType]
	if !exists {
		return nil, fmt.Errorf("no active version for model type: %s", modelType)
	}

	key := s.modelKey(modelType, versionID)
	version, exists := s.modelRegistry[key]
	if !exists {
		return nil, fmt.Errorf("current version not found: %s", versionID)
	}

	result := *version
	return &result, nil
}

func (s *ModelUpdateService) ListVersions(query VersionQuery) ([]*ModelVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var versions []*ModelVersion
	for _, v := range s.versionHistory[query.ModelType] {
		if query.Status != "" && v.Status != query.Status {
			continue
		}
		if !query.StartTime.IsZero() && v.CreatedAt.Before(query.StartTime) {
			continue
		}
		if !query.EndTime.IsZero() && v.CreatedAt.After(query.EndTime) {
			continue
		}
		if query.VersionID != "" && v.VersionID != query.VersionID {
			continue
		}
		versions = append(versions, v)
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].CreatedAt.After(versions[j].CreatedAt)
	})

	if query.Limit > 0 {
		start := query.Offset
		if start > len(versions) {
			start = len(versions)
		}
		end := start + query.Limit
		if end > len(versions) {
			end = len(versions)
		}
		versions = versions[start:end]
	}

	result := make([]*ModelVersion, len(versions))
	for i, v := range versions {
		item := *v
		result[i] = &item
	}

	return result, nil
}

func (s *ModelUpdateService) UpdateMetrics(ctx context.Context, update ModelMetricsUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metrics, exists := s.performanceMetrics[update.ModelType]
	if !exists {
		metrics = &PerformanceMetrics{
			ModelType:      update.ModelType,
			VersionID:      update.VersionID,
			MetricsHistory: make([]MetricSnapshot, 0),
		}
		s.performanceMetrics[update.ModelType] = metrics
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	metrics.SampleCount++
	if update.LatencyMs > 0 {
		metrics.LatencyP50 = s.updatePercentile(metrics.LatencyP50, update.LatencyMs, 50, metrics.SampleCount)
		metrics.LatencyP95 = s.updatePercentile(metrics.LatencyP95, update.LatencyMs, 95, metrics.SampleCount)
		metrics.LatencyP99 = s.updatePercentile(metrics.LatencyP99, update.LatencyMs, 99, metrics.SampleCount)
	}

	if update.IsError {
		metrics.ErrorRate = (float64(metrics.SampleCount-metrics.TruePositive-metrics.FalsePositive-metrics.TrueNegative-metrics.FalseNegative) + 1) / float64(metrics.SampleCount)
	}

	if update.Prediction == update.Actual {
		if update.Prediction > 0.5 {
			metrics.TruePositive++
		} else {
			metrics.TrueNegative++
		}
	} else {
		if update.Prediction > 0.5 {
			metrics.FalsePositive++
		} else {
			metrics.FalseNegative++
		}
	}

	if metrics.SampleCount > 0 {
		metrics.Accuracy = float64(metrics.TruePositive+metrics.TrueNegative) / float64(metrics.SampleCount)
	}
	if metrics.TruePositive+metrics.FalsePositive > 0 {
		metrics.Precision = float64(metrics.TruePositive) / float64(metrics.TruePositive+metrics.FalsePositive)
	}
	if metrics.TruePositive+metrics.FalseNegative > 0 {
		metrics.Recall = float64(metrics.TruePositive) / float64(metrics.TruePositive+metrics.FalseNegative)
	}
	if metrics.Precision+metrics.Recall > 0 {
		metrics.F1Score = 2 * (metrics.Precision * metrics.Recall) / (metrics.Precision + metrics.Recall)
	}

	metrics.Throughput = float64(metrics.SampleCount) / time.Since(metrics.LastUpdated).Seconds()
	metrics.LastUpdated = time.Now()

	if s.config.EnableMetricsHistory && metrics.SampleCount%100 == 0 {
		snapshot := MetricSnapshot{
			Timestamp:   metrics.LastUpdated,
			Accuracy:    metrics.Accuracy,
			LatencyP95:  metrics.LatencyP95,
			ErrorRate:   metrics.ErrorRate,
			SampleCount: metrics.SampleCount,
		}
		metrics.MetricsHistory = append(metrics.MetricsHistory, snapshot)

		if len(metrics.MetricsHistory) > s.config.MetricsWindowSize {
			metrics.MetricsHistory = metrics.MetricsHistory[len(metrics.MetricsHistory)-s.config.MetricsWindowSize:]
		}
	}

	return nil
}

func (s *ModelUpdateService) updatePercentile(currentValue, newValue float64, percentile, count int64) float64 {
	alpha := 1.0 / float64(math.Min(float64(count), 100.0))
	return currentValue*(1-alpha) + newValue*alpha
}

func (s *ModelUpdateService) GetPerformanceMetrics(modelType string) (*PerformanceMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics, exists := s.performanceMetrics[modelType]
	if !exists {
		return nil, fmt.Errorf("metrics not found for model type: %s", modelType)
	}

	metrics.mu.RLock()
	defer metrics.mu.RUnlock()

	result := &PerformanceMetrics{
		ModelType:     metrics.ModelType,
		VersionID:     metrics.VersionID,
		Accuracy:      metrics.Accuracy,
		Precision:     metrics.Precision,
		Recall:        metrics.Recall,
		F1Score:       metrics.F1Score,
		LatencyP50:    metrics.LatencyP50,
		LatencyP95:    metrics.LatencyP95,
		LatencyP99:    metrics.LatencyP99,
		Throughput:    metrics.Throughput,
		ErrorRate:     metrics.ErrorRate,
		SampleCount:   metrics.SampleCount,
		TruePositive:  metrics.TruePositive,
		FalsePositive: metrics.FalsePositive,
		TrueNegative:  metrics.TrueNegative,
		FalseNegative: metrics.FalseNegative,
		LastUpdated:   metrics.LastUpdated,
		MetricsHistory: make([]MetricSnapshot, len(metrics.MetricsHistory)),
	}
	copy(result.MetricsHistory, metrics.MetricsHistory)

	return result, nil
}

func (s *ModelUpdateService) PerformHealthCheck(ctx context.Context, modelType string) (*HealthCheckResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	versionID, exists := s.currentVersions[modelType]
	if !exists {
		return nil, fmt.Errorf("no active version for model type: %s", modelType)
	}

	metrics, metricsExists := s.performanceMetrics[modelType]
	result := &HealthCheckResult{
		ModelType:       modelType,
		VersionID:      versionID,
		CheckTime:      time.Now(),
		Issues:         make([]HealthIssue, 0),
		Recommendations: make([]string, 0),
	}

	if !metricsExists || metrics.SampleCount < int64(s.config.MinSamplesForHealthCheck) {
		result.IsHealthy = true
		result.Score = 0.5
		result.Issues = append(result.Issues, HealthIssue{
			Severity: "info",
			Code:     "INSUFFICIENT_SAMPLES",
			Message:  "Not enough samples for comprehensive health check",
			Detail:   fmt.Sprintf("Have %d samples, need %d", metrics.SampleCount, s.config.MinSamplesForHealthCheck),
		})
		result.Recommendations = append(result.Recommendations, "Collect more samples before making health assessment")
		s.healthCheckResults[modelType] = result
		return result, nil
	}

	metrics.mu.RLock()
	defer metrics.mu.RUnlock()

	score := 1.0

	if metrics.Accuracy < 0.5 {
		result.Issues = append(result.Issues, HealthIssue{
			Severity: "critical",
			Code:     "LOW_ACCURACY",
			Message:  fmt.Sprintf("Model accuracy is critically low: %.2f%%", metrics.Accuracy*100),
		})
		score -= 0.4
		result.Recommendations = append(result.Recommendations, "Consider rolling back to previous stable version")
	} else if metrics.Accuracy < 0.7 {
		result.Issues = append(result.Issues, HealthIssue{
			Severity: "warning",
			Code:     "MODERATE_ACCURACY",
			Message:  fmt.Sprintf("Model accuracy is below target: %.2f%%", metrics.Accuracy*100),
		})
		score -= 0.2
		result.Recommendations = append(result.Recommendations, "Monitor closely, may need update or rollback")
	}

	if metrics.LatencyP95 > s.config.RollbackThresholdLatency*100 {
		result.Issues = append(result.Issues, HealthIssue{
			Severity: "warning",
			Code:     "HIGH_LATENCY",
			Message:  fmt.Sprintf("Model latency is high: %.2fms (P95)", metrics.LatencyP95),
		})
		score -= 0.2
		result.Recommendations = append(result.Recommendations, "Consider model optimization or hardware upgrade")
	}

	if metrics.ErrorRate > s.config.RollbackThresholdErrorRate {
		result.Issues = append(result.Issues, HealthIssue{
			Severity: "critical",
			Code:     "HIGH_ERROR_RATE",
			Message:  fmt.Sprintf("Error rate is critically high: %.2f%%", metrics.ErrorRate*100),
		})
		score -= 0.3
		result.Recommendations = append(result.Recommendations, "Immediate rollback recommended")
	}

	if metrics.F1Score < 0.5 {
		result.Issues = append(result.Issues, HealthIssue{
			Severity: "warning",
			Code:     "LOW_F1_SCORE",
			Message:  fmt.Sprintf("F1 score is below target: %.2f", metrics.F1Score),
		})
		score -= 0.15
	}

	result.Score = math.Max(0, score)
	result.IsHealthy = result.Score >= 0.6 && len(result.Issues) == 0

	for _, issue := range result.Issues {
		if issue.Severity == "critical" {
			result.IsHealthy = false
			break
		}
	}

	s.healthCheckResults[modelType] = result

	return result, nil
}

func (s *ModelUpdateService) GetHealthCheckResult(modelType string) (*HealthCheckResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result, exists := s.healthCheckResults[modelType]
	if !exists {
		return nil, fmt.Errorf("no health check result for model type: %s", modelType)
	}

	res := *result
	res.Issues = make([]HealthIssue, len(result.Issues))
	copy(res.Issues, result.Issues)
	res.Recommendations = make([]string, len(result.Recommendations))
	copy(res.Recommendations, result.Recommendations)

	return &res, nil
}

func (s *ModelUpdateService) TriggerRollback(ctx context.Context, modelType string, reason string) (*UpdateResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	versionID, exists := s.currentVersions[modelType]
	if !exists {
		return &UpdateResult{
			Success: false,
			Message: "no active version to rollback from",
		}, errors.New("no active version to rollback from")
	}

	key := s.modelKey(modelType, versionID)
	currentVersion := s.modelRegistry[key]
	if currentVersion == nil {
		return &UpdateResult{
			Success: false,
			Message: "current version not found in registry",
		}, errors.New("current version not found in registry")
	}

	if currentVersion.RollbackCount >= s.config.MaxRollbackAttempts {
		return &UpdateResult{
			Success: false,
			Message: fmt.Sprintf("maximum rollback attempts (%d) reached", s.config.MaxRollbackAttempts),
		}, fmt.Errorf("maximum rollback attempts reached")
	}

	var rollbackVersion *ModelVersion
	var historyKey int = -1

	history := s.versionHistory[modelType]
	for i := len(history) - 2; i >= 0; i-- {
		v := history[i]
		if v.Status == ModelStatusStable || v.Status == ModelStatusActive {
			rollbackVersion = v
			historyKey = i
			break
		}
	}

	if rollbackVersion == nil {
		for i := len(history) - 2; i >= 0; i-- {
			v := history[i]
			if v.Status == ModelStatusDeprecated && v.VersionID != versionID {
				rollbackVersion = v
				historyKey = i
				break
			}
		}
	}

	if rollbackVersion == nil {
		return &UpdateResult{
			Success: false,
			Message: "no suitable version found for rollback",
		}, errors.New("no suitable rollback version found")
	}

	currentVersion.Status = ModelStatusRollbacking
	currentVersion.RollbackCount++

	if callback, ok := s.rollbackCallbacks[modelType]; ok {
		s.mu.Unlock()
		err := callback(modelType, currentVersion, rollbackVersion)
		s.mu.Lock()
		if err != nil {
			currentVersion.Status = ModelStatusRollbackFailed
			return &UpdateResult{
				Success:          false,
				Message:          fmt.Sprintf("rollback callback failed: %v", err),
				Error:            err,
				RollbackTriggered: true,
			}, err
		}
	}

	currentVersion.Status = ModelStatusDeprecated
	rollbackVersion.Status = ModelStatusActive
	rollbackVersion.RollbackFrom = versionID
	s.currentVersions[modelType] = rollbackVersion.VersionID

	return &UpdateResult{
		Success:           true,
		OldVersionID:      versionID,
		NewVersionID:      rollbackVersion.VersionID,
		Message:           fmt.Sprintf("rollback successful: %s", reason),
		RollbackTriggered: true,
		RollbackVersion:   rollbackVersion.VersionID,
	}, nil
}

func (s *ModelUpdateService) AutoRollbackIfNeeded(ctx context.Context, modelType string) (*UpdateResult, error) {
	if !s.config.AutoRollbackEnabled {
		return nil, nil
	}

	s.mu.RLock()
	metrics := s.performanceMetrics[modelType]
	s.mu.RUnlock()

	if metrics == nil || metrics.SampleCount < int64(s.config.MinSamplesForHealthCheck) {
		return nil, nil
	}

	metrics.mu.RLock()
	shouldRollback := metrics.Accuracy < s.config.RollbackThresholdAccuracy ||
		metrics.LatencyP95 > s.config.RollbackThresholdLatency*100 ||
		metrics.ErrorRate > s.config.RollbackThresholdErrorRate
	metrics.mu.RUnlock()

	if shouldRollback {
		reason := fmt.Sprintf("auto-rollback triggered: accuracy=%.2f (threshold=%.2f), latency_p95=%.2f (threshold=%.2f), error_rate=%.2f (threshold=%.2f)",
			metrics.Accuracy, s.config.RollbackThresholdAccuracy,
			metrics.LatencyP95, s.config.RollbackThresholdLatency*100,
			metrics.ErrorRate, s.config.RollbackThresholdErrorRate)
		return s.TriggerRollback(ctx, modelType, reason)
	}

	return nil, nil
}

func (s *ModelUpdateService) MarkVersionStable(ctx context.Context, modelType, versionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.modelKey(modelType, versionID)
	version, exists := s.modelRegistry[key]
	if !exists {
		return fmt.Errorf("model version not found: %s", versionID)
	}

	version.Status = ModelStatusStable
	return nil
}

func (s *ModelUpdateService) DeprecateVersion(ctx context.Context, modelType, versionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.modelKey(modelType, versionID)
	version, exists := s.modelRegistry[key]
	if !exists {
		return fmt.Errorf("model version not found: %s", versionID)
	}

	if version.Status == ModelStatusActive {
		return errors.New("cannot deprecate active version, use rollback instead")
	}

	version.Status = ModelStatusDeprecated
	return nil
}

func (s *ModelUpdateService) RegisterUpdateCallback(modelType string, callback ModelUpdateCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateCallbacks[modelType] = callback
}

func (s *ModelUpdateService) RegisterRollbackCallback(modelType string, callback ModelRollbackCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rollbackCallbacks[modelType] = callback
}

func (s *ModelUpdateService) UpdateConfig(cfg *UpdateConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cfg.AutoRollbackEnabled {
		s.config.AutoRollbackEnabled = cfg.AutoRollbackEnabled
	}
	if cfg.RollbackThresholdAccuracy > 0 {
		s.config.RollbackThresholdAccuracy = cfg.RollbackThresholdAccuracy
	}
	if cfg.RollbackThresholdLatency > 0 {
		s.config.RollbackThresholdLatency = cfg.RollbackThresholdLatency
	}
	if cfg.RollbackThresholdErrorRate > 0 {
		s.config.RollbackThresholdErrorRate = cfg.RollbackThresholdErrorRate
	}
	if cfg.HealthCheckInterval > 0 {
		s.config.HealthCheckInterval = cfg.HealthCheckInterval
	}
	if cfg.MetricsWindowSize > 0 {
		s.config.MetricsWindowSize = cfg.MetricsWindowSize
	}
	if cfg.MinSamplesForHealthCheck > 0 {
		s.config.MinSamplesForHealthCheck = cfg.MinSamplesForHealthCheck
	}
	if cfg.StagingDuration > 0 {
		s.config.StagingDuration = cfg.StagingDuration
	}
	if cfg.MaxRollbackAttempts > 0 {
		s.config.MaxRollbackAttempts = cfg.MaxRollbackAttempts
	}
	s.config.EnableMetricsHistory = cfg.EnableMetricsHistory
}

func (s *ModelUpdateService) GetConfig() *UpdateConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg := *s.config
	return &cfg
}

func (s *ModelUpdateService) healthCheckLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.performPeriodicHealthCheck(ctx)
		}
	}
}

func (s *ModelUpdateService) performPeriodicHealthCheck(ctx context.Context) {
	s.mu.RLock()
	modelTypes := make([]string, 0, len(s.currentVersions))
	for modelType := range s.currentVersions {
		modelTypes = append(modelTypes, modelType)
	}
	s.mu.RUnlock()

	for _, modelType := range modelTypes {
		_, err := s.PerformHealthCheck(ctx, modelType)
		if err != nil {
			continue
		}

		result, _ := s.AutoRollbackIfNeeded(ctx, modelType)
		if result != nil && result.Success {
			continue
		}
	}
}

func (s *ModelUpdateService) modelKey(modelType, versionID string) string {
	return fmt.Sprintf("%s:%s", modelType, versionID)
}

func (s *ModelUpdateService) ExportVersionHistory(modelType string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := s.versionHistory[modelType]
	data := make([]*ModelVersion, len(history))
	for i, v := range history {
		item := *v
		data[i] = &item
	}

	return json.MarshalIndent(data, "", "  ")
}

func (s *ModelUpdateService) GetVersionStatistics(modelType string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := s.versionHistory[modelType]
	if len(history) == 0 {
		return nil, fmt.Errorf("no version history for model type: %s", modelType)
	}

	stats := make(map[string]interface{})

	statusCounts := make(map[string]int)
	for _, v := range history {
		statusCounts[string(v.Status)]++
	}
	stats["status_counts"] = statusCounts
	stats["total_versions"] = len(history)

	var totalRollbacks int
	for _, v := range history {
		totalRollbacks += v.RollbackCount
	}
	stats["total_rollbacks"] = totalRollbacks

	if len(history) > 0 {
		stats["first_version"] = history[0].VersionID
		stats["latest_version"] = history[len(history)-1].VersionID
	}

	if metrics := s.performanceMetrics[modelType]; metrics != nil {
		metrics.mu.RLock()
		stats["current_accuracy"] = metrics.Accuracy
		stats["current_latency_p95"] = metrics.LatencyP95
		stats["current_error_rate"] = metrics.ErrorRate
		stats["total_samples"] = metrics.SampleCount
		metrics.mu.RUnlock()
	}

	return stats, nil
}

func (s *ModelUpdateService) CompareVersions(modelType, versionID1, versionID2 string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key1 := s.modelKey(modelType, versionID1)
	key2 := s.modelKey(modelType, versionID2)

	v1, exists1 := s.modelRegistry[key1]
	v2, exists2 := s.modelRegistry[key2]

	if !exists1 {
		return nil, fmt.Errorf("version not found: %s", versionID1)
	}
	if !exists2 {
		return nil, fmt.Errorf("version not found: %s", versionID2)
	}

	comparison := make(map[string]interface{})

	comparison["version1"] = map[string]interface{}{
		"version_id":  v1.VersionID,
		"status":      v1.Status,
		"created_at":  v1.CreatedAt,
		"rollback_count": v1.RollbackCount,
	}

	comparison["version2"] = map[string]interface{}{
		"version_id":  v2.VersionID,
		"status":      v2.Status,
		"created_at":  v2.CreatedAt,
		"rollback_count": v2.RollbackCount,
	}

	comparison["age_difference_hours"] = v2.CreatedAt.Sub(v1.CreatedAt).Hours()

	return comparison, nil
}
