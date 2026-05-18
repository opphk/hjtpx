package trace

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type ModelUpdateStrategy string

const (
	UpdateStrategyOnline       ModelUpdateStrategy = "online"
	UpdateStrategyPeriodic     ModelUpdateStrategy = "periodic"
	UpdateStrategyThreshold    ModelUpdateStrategy = "threshold"
	UpdateStrategyManual       ModelUpdateStrategy = "manual"
)

type ModelUpdateStatus string

const (
	UpdateStatusIdle        ModelUpdateStatus = "idle"
	UpdateStatusCollecting  ModelUpdateStatus = "collecting"
	UpdateStatusUpdating    ModelUpdateStatus = "updating"
	UpdateStatusCompleted   ModelUpdateStatus = "completed"
	UpdateStatusFailed      ModelUpdateStatus = "failed"
)

type UpdateConfig struct {
	Strategy              ModelUpdateStrategy
	UpdateInterval        time.Duration
	MinSamplesForUpdate   int
	ConfidenceThreshold   float64
	PerformanceThreshold  float64
	MaxUpdatesPerHour     int
	EnableRollback        bool
	RollbackThreshold     float64
}

type ModelUpdateRecord struct {
	UpdateID        string
	Timestamp       time.Time
	Status          ModelUpdateStatus
	Duration        time.Duration
	SamplesUsed     int
	PerformanceGain float64
	Error           string
	ModelVersion    string
}

type ModelVersionInfo struct {
	Version        string
	CreatedAt      time.Time
	UpdateRecord   *ModelUpdateRecord
	Performance    ModelPerformanceMetrics
	IsActive       bool
}

type EnhancedModelUpdater struct {
	mu                      sync.RWMutex
	traceService            *TraceService
	sampleQueue             []TrajectorySample
	isRunning               bool
	status                  ModelUpdateStatus
	stopChan                chan struct{}
	updateInterval          time.Duration
	minSamplesForUpdate     int
	updateCount             int
	lastUpdateTime          time.Time
	updateRecords           []ModelUpdateRecord
	modelVersions           []ModelVersionInfo
	currentVersion          string
	updateConfig            UpdateConfig
	rollbackHistory         []ModelUpdateRecord
	performanceHistory      []ModelPerformanceMetrics
}

func NewEnhancedModelUpdater(traceService *TraceService) *EnhancedModelUpdater {
	return &EnhancedModelUpdater{
		traceService:        traceService,
		sampleQueue:         make([]TrajectorySample, 0, 1000),
		stopChan:            make(chan struct{}),
		updateInterval:      5 * time.Minute,
		minSamplesForUpdate: 10,
		status:              UpdateStatusIdle,
		currentVersion:      "v1.0.0",
		updateConfig: UpdateConfig{
			Strategy:             UpdateStrategyOnline,
			UpdateInterval:       5 * time.Minute,
			MinSamplesForUpdate:  10,
			ConfidenceThreshold:  0.7,
			PerformanceThreshold: 0.05,
			MaxUpdatesPerHour:    6,
			EnableRollback:       true,
			RollbackThreshold:    0.1,
		},
		updateRecords:      make([]ModelUpdateRecord, 0, 100),
		modelVersions:      make([]ModelVersionInfo, 0, 10),
		rollbackHistory:    make([]ModelUpdateRecord, 0, 20),
		performanceHistory: make([]ModelPerformanceMetrics, 0, 100),
	}
}

func (u *EnhancedModelUpdater) SetConfig(config UpdateConfig) error {
	if config.MinSamplesForUpdate < 1 {
		return errors.New("minimum samples for update must be at least 1")
	}
	if config.ConfidenceThreshold < 0 || config.ConfidenceThreshold > 1 {
		return errors.New("confidence threshold must be between 0 and 1")
	}
	if config.PerformanceThreshold < 0 || config.PerformanceThreshold > 1 {
		return errors.New("performance threshold must be between 0 and 1")
	}

	u.mu.Lock()
	u.updateConfig = config
	u.updateInterval = config.UpdateInterval
	u.minSamplesForUpdate = config.MinSamplesForUpdate
	u.mu.Unlock()

	return nil
}

func (u *EnhancedModelUpdater) GetConfig() UpdateConfig {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.updateConfig
}

func (u *EnhancedModelUpdater) QueueSample(traceData *model.TraceData, isBot bool, confidence float64) error {
	if confidence < u.updateConfig.ConfidenceThreshold {
		return nil
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	if len(u.sampleQueue) >= 10000 {
		u.sampleQueue = u.sampleQueue[len(u.sampleQueue)-5000:]
	}

	u.sampleQueue = append(u.sampleQueue, TrajectorySample{
		TraceData:  traceData,
		IsBot:      isBot,
		Confidence: confidence,
		Timestamp:  time.Now(),
	})

	return nil
}

func (u *EnhancedModelUpdater) Start() {
	u.mu.Lock()
	if u.isRunning {
		u.mu.Unlock()
		return
	}
	u.isRunning = true
	u.status = UpdateStatusCollecting
	u.mu.Unlock()

	go u.runUpdateLoop()
}

func (u *EnhancedModelUpdater) Stop() {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.isRunning {
		close(u.stopChan)
		u.isRunning = false
		u.status = UpdateStatusIdle
	}
}

func (u *EnhancedModelUpdater) runUpdateLoop() {
	ticker := time.NewTicker(u.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			u.checkAndUpdate()
		case <-u.stopChan:
			return
		}
	}
}

func (u *EnhancedModelUpdater) checkAndUpdate() {
	u.mu.Lock()
	now := time.Now()
	samplesReady := len(u.sampleQueue) >= u.minSamplesForUpdate
	withinRateLimit := u.updateCount < u.updateConfig.MaxUpdatesPerHour
	enoughTimePassed := now.Sub(u.lastUpdateTime) >= time.Hour/time.Duration(u.updateConfig.MaxUpdatesPerHour)
	
	if !samplesReady || !withinRateLimit || !enoughTimePassed {
		u.mu.Unlock()
		return
	}

	samples := make([]TrajectorySample, len(u.sampleQueue))
	copy(samples, u.sampleQueue)
	u.sampleQueue = u.sampleQueue[:0]
	u.status = UpdateStatusUpdating
	u.mu.Unlock()

	u.performUpdate(samples)
}

func (u *EnhancedModelUpdater) performUpdate(samples []TrajectorySample) {
	updateID := generateUpdateID()
	startTime := time.Now()

	record := ModelUpdateRecord{
		UpdateID:  updateID,
		Timestamp: startTime,
		Status:    UpdateStatusUpdating,
	}

	defer func() {
		u.mu.Lock()
		record.Duration = time.Since(startTime)
		record.SamplesUsed = len(samples)
		u.updateRecords = append(u.updateRecords, record)
		u.status = record.Status
		u.lastUpdateTime = startTime
		u.updateCount++
		
		if len(u.updateRecords) > 100 {
			u.updateRecords = u.updateRecords[len(u.updateRecords)-100:]
		}
		u.mu.Unlock()
	}()

	initialPerformance := u.traceService.GetModelPerformanceReport()

	for _, sample := range samples {
		u.updateModelWithSample(sample)
	}

	finalPerformance := u.traceService.GetModelPerformanceReport()
	performanceGain := u.calculatePerformanceGain(initialPerformance, finalPerformance)

	if u.updateConfig.EnableRollback && performanceGain < -u.updateConfig.RollbackThreshold {
		u.rollback()
		record.Status = UpdateStatusFailed
		record.Error = "Performance degraded, rolled back"
		record.PerformanceGain = performanceGain
		return
	}

	record.Status = UpdateStatusCompleted
	record.PerformanceGain = performanceGain
	record.ModelVersion = u.currentVersion

	u.mu.Lock()
	u.currentVersion = u.generateNextVersion()
	u.modelVersions = append(u.modelVersions, ModelVersionInfo{
		Version:      u.currentVersion,
		CreatedAt:    startTime,
		UpdateRecord: &record,
		IsActive:     true,
	})
	
	if len(u.modelVersions) > 10 {
		u.modelVersions = u.modelVersions[len(u.modelVersions)-10:]
	}
	u.mu.Unlock()
}

func (u *EnhancedModelUpdater) updateModelWithSample(sample TrajectorySample) {
	if u.traceService.transformerPredictor == nil {
		return
	}

	featureVec := make([]float64, 0)
	if sample.FeatureVec != nil && len(sample.FeatureVec) > 0 {
		featureVec = sample.FeatureVec
	} else {
		featureMap, err := u.traceService.ExtractNNFeatures(nil, sample.TraceData)
		if err == nil && len(featureMap) > 0 {
			for _, v := range featureMap {
				featureVec = append(featureVec, v)
			}
		}
	}

	if len(featureVec) == 0 {
		return
	}

	prediction, err := u.traceService.transformerPredictor.PredictWithFeatures(featureVec)
	if err != nil {
		return
	}

	predictedIsBot := prediction.BotProbability > 0.5
	actualIsBot := sample.IsBot

	u.traceService.RecordPrediction("transformer", predictedIsBot, actualIsBot, 0)

	learningRate := 0.01
	errorSignal := 0.0
	if actualIsBot && !predictedIsBot {
		errorSignal = 1.0
	} else if !actualIsBot && predictedIsBot {
		errorSignal = -1.0
	}

	if errorSignal != 0 {
		u.adjustPredictionHead(featureVec, errorSignal, learningRate)
	}
}

func (u *EnhancedModelUpdater) adjustPredictionHead(features []float64, errorSignal, learningRate float64) {
	if u.traceService.transformerPredictor == nil {
		return
	}

	predictor := u.traceService.transformerPredictor
	predictor.mu.Lock()
	defer predictor.mu.Unlock()

	for i := range features {
		if i < len(predictor.predictionHead) {
			predictor.predictionHead[i] += errorSignal * features[i] * learningRate
		}
	}

	predictor.predictionBias += errorSignal * learningRate
}

func (u *EnhancedModelUpdater) calculatePerformanceGain(initial, final map[string]interface{}) float64 {
	initialAccuracy := 0.0
	finalAccuracy := 0.0

	if transformer, ok := initial["transformer"].(map[string]interface{}); ok {
		if acc, ok := transformer["accuracy"].(float64); ok {
			initialAccuracy = acc
		}
	}

	if transformer, ok := final["transformer"].(map[string]interface{}); ok {
		if acc, ok := transformer["accuracy"].(float64); ok {
			finalAccuracy = acc
		}
	}

	return finalAccuracy - initialAccuracy
}

func (u *EnhancedModelUpdater) rollback() {
	u.mu.Lock()
	defer u.mu.Unlock()

	if len(u.modelVersions) >= 2 {
		u.currentVersion = u.modelVersions[len(u.modelVersions)-2].Version
	}

	if u.traceService.transformerPredictor != nil {
		u.traceService.transformerPredictor.initializeWeights()
	}
}

func (u *EnhancedModelUpdater) ForceUpdate() (*ModelUpdateRecord, error) {
	u.mu.Lock()
	if !u.isRunning {
		u.mu.Unlock()
		return nil, errors.New("updater is not running")
	}
	
	if len(u.sampleQueue) < u.minSamplesForUpdate {
		u.mu.Unlock()
		return nil, errors.New("not enough samples collected")
	}

	samples := make([]TrajectorySample, len(u.sampleQueue))
	copy(samples, u.sampleQueue)
	u.sampleQueue = u.sampleQueue[:0]
	u.status = UpdateStatusUpdating
	updateID := generateUpdateID()
	startTime := time.Now()
	u.mu.Unlock()

	record := ModelUpdateRecord{
		UpdateID:  updateID,
		Timestamp: startTime,
		Status:    UpdateStatusUpdating,
	}

	initialPerformance := u.traceService.GetModelPerformanceReport()

	for _, sample := range samples {
		u.updateModelWithSample(sample)
	}

	finalPerformance := u.traceService.GetModelPerformanceReport()
	performanceGain := u.calculatePerformanceGain(initialPerformance, finalPerformance)

	if u.updateConfig.EnableRollback && performanceGain < -u.updateConfig.RollbackThreshold {
		u.rollback()
		record.Status = UpdateStatusFailed
		record.Error = "Performance degraded, rolled back"
	} else {
		record.Status = UpdateStatusCompleted
		record.ModelVersion = u.currentVersion
	}

	record.Duration = time.Since(startTime)
	record.SamplesUsed = len(samples)
	record.PerformanceGain = performanceGain

	u.mu.Lock()
	u.updateRecords = append(u.updateRecords, record)
	u.status = record.Status
	u.lastUpdateTime = startTime
	u.updateCount++
	u.mu.Unlock()

	return &record, nil
}

func (u *EnhancedModelUpdater) generateNextVersion() string {
	return "v" + time.Now().Format("2.06.0102")
}

func generateUpdateID() string {
	return "update_" + time.Now().Format("20060102150405") + "_" + randString(8)
}

func (u *EnhancedModelUpdater) GetStatus() ModelUpdateStatus {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.status
}

func (u *EnhancedModelUpdater) GetUpdateHistory() []ModelUpdateRecord {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return append([]ModelUpdateRecord(nil), u.updateRecords...)
}

func (u *EnhancedModelUpdater) GetModelVersions() []ModelVersionInfo {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return append([]ModelVersionInfo(nil), u.modelVersions...)
}

func (u *EnhancedModelUpdater) GetCurrentVersion() string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.currentVersion
}

func (u *EnhancedModelUpdater) GetSampleCount() int {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return len(u.sampleQueue)
}

func (u *EnhancedModelUpdater) GetStats() map[string]interface{} {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return map[string]interface{}{
		"status":             u.status,
		"current_version":    u.currentVersion,
		"update_count":       u.updateCount,
		"pending_samples":    len(u.sampleQueue),
		"last_update_time":   u.lastUpdateTime,
		"total_updates":      len(u.updateRecords),
		"strategy":          u.updateConfig.Strategy,
		"update_interval":   u.updateInterval,
		"min_samples":       u.minSamplesForUpdate,
		"rollback_enabled":  u.updateConfig.EnableRollback,
	}
}

func (u *EnhancedModelUpdater) ResetUpdateCount() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.updateCount = 0
}

func (u *EnhancedModelUpdater) ClearSamples() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.sampleQueue = make([]TrajectorySample, 0, 1000)
}

func (u *EnhancedModelUpdater) UpdateByThreshold() bool {
	u.mu.Lock()
	samplesReady := len(u.sampleQueue) >= u.minSamplesForUpdate
	u.mu.Unlock()

	if !samplesReady {
		return false
	}

	performance := u.traceService.GetModelPerformanceReport()
	accuracy := 0.0

	if transformer, ok := performance["transformer"].(map[string]interface{}); ok {
		if acc, ok := transformer["accuracy"].(float64); ok {
			accuracy = acc
		}
	}

	if accuracy < u.updateConfig.PerformanceThreshold {
		_, _ = u.ForceUpdate()
		return true
	}

	return false
}

type BatchUpdateRequest struct {
	Samples []TrajectorySample
	UpdateStrategy ModelUpdateStrategy
}

type BatchUpdateResponse struct {
	Success        bool
	UpdateID       string
	SamplesProcessed int
	PerformanceGain float64
	Error          string
}

func (u *EnhancedModelUpdater) BatchUpdate(request BatchUpdateRequest) (*BatchUpdateResponse, error) {
	if len(request.Samples) == 0 {
		return nil, errors.New("no samples provided")
	}

	updateID := generateUpdateID()
	startTime := time.Now()

	initialPerformance := u.traceService.GetModelPerformanceReport()

	for _, sample := range request.Samples {
		u.updateModelWithSample(sample)
	}

	finalPerformance := u.traceService.GetModelPerformanceReport()
	performanceGain := u.calculatePerformanceGain(initialPerformance, finalPerformance)

	if u.updateConfig.EnableRollback && performanceGain < -u.updateConfig.RollbackThreshold {
		u.rollback()
		return &BatchUpdateResponse{
			Success:        false,
			UpdateID:       updateID,
			SamplesProcessed: len(request.Samples),
			PerformanceGain: performanceGain,
			Error:          "Performance degraded, rolled back",
		}, nil
	}

	u.mu.Lock()
	u.currentVersion = u.generateNextVersion()
	u.updateRecords = append(u.updateRecords, ModelUpdateRecord{
		UpdateID:        updateID,
		Timestamp:       startTime,
		Status:          UpdateStatusCompleted,
		Duration:        time.Since(startTime),
		SamplesUsed:     len(request.Samples),
		PerformanceGain: performanceGain,
		ModelVersion:    u.currentVersion,
	})
	u.mu.Unlock()

	return &BatchUpdateResponse{
		Success:        true,
		UpdateID:       updateID,
		SamplesProcessed: len(request.Samples),
		PerformanceGain: performanceGain,
	}, nil
}

func (u *EnhancedModelUpdater) CalculateSampleDiversity(samples []TrajectorySample) float64 {
	if len(samples) < 2 {
		return 0.0
	}

	var totalDistance float64
	count := 0

	for i := 0; i < len(samples); i++ {
		for j := i + 1; j < len(samples); j++ {
			dist := u.calculateSampleDistance(samples[i], samples[j])
			totalDistance += dist
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return totalDistance / float64(count)
}

func (u *EnhancedModelUpdater) calculateSampleDistance(s1, s2 TrajectorySample) float64 {
	if s1.FeatureVec == nil || s2.FeatureVec == nil {
		return 0.0
	}

	minLen := int(math.Min(float64(len(s1.FeatureVec)), float64(len(s2.FeatureVec))))
	var distance float64

	for i := 0; i < minLen; i++ {
		diff := s1.FeatureVec[i] - s2.FeatureVec[i]
		distance += diff * diff
	}

	return math.Sqrt(distance)
}