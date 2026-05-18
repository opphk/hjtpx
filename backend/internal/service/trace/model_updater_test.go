package trace

import (
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestEnhancedModelUpdaterInitialization(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	
	if updater == nil {
		t.Fatal("EnhancedModelUpdater should not be nil")
	}
	
	if updater.GetStatus() != UpdateStatusIdle {
		t.Errorf("Expected status 'idle', got %v", updater.GetStatus())
	}
}

func TestEnhancedModelUpdaterStartStop(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	
	updater.Start()
	
	if updater.GetStatus() != UpdateStatusCollecting {
		t.Errorf("Expected status 'collecting', got %v", updater.GetStatus())
	}
	
	updater.Stop()
	
	if updater.GetStatus() != UpdateStatusIdle {
		t.Errorf("Expected status 'idle', got %v", updater.GetStatus())
	}
}

func TestEnhancedModelUpdaterSetConfig(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	
	config := UpdateConfig{
		Strategy:             UpdateStrategyOnline,
		UpdateInterval:       10 * time.Minute,
		MinSamplesForUpdate:  20,
		ConfidenceThreshold:  0.8,
		PerformanceThreshold: 0.1,
		MaxUpdatesPerHour:    5,
		EnableRollback:       true,
		RollbackThreshold:    0.15,
	}
	
	err := updater.SetConfig(config)
	if err != nil {
		t.Errorf("SetConfig should not return error: %v", err)
	}
	
	retrievedConfig := updater.GetConfig()
	if retrievedConfig.MinSamplesForUpdate != 20 {
		t.Errorf("Expected MinSamplesForUpdate 20, got %d", retrievedConfig.MinSamplesForUpdate)
	}
}

func TestEnhancedModelUpdaterSetConfigValidation(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	
	err := updater.SetConfig(UpdateConfig{MinSamplesForUpdate: 0})
	if err == nil {
		t.Error("SetConfig should return error for MinSamplesForUpdate < 1")
	}
	
	err = updater.SetConfig(UpdateConfig{ConfidenceThreshold: 1.5})
	if err == nil {
		t.Error("SetConfig should return error for ConfidenceThreshold > 1")
	}
	
	err = updater.SetConfig(UpdateConfig{ConfidenceThreshold: -0.1})
	if err == nil {
		t.Error("SetConfig should return error for ConfidenceThreshold < 0")
	}
}

func TestEnhancedModelUpdaterQueueSample(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	
	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
		},
	}
	
	err := updater.QueueSample(traceData, false, 0.9)
	if err != nil {
		t.Errorf("QueueSample should not return error: %v", err)
	}
	
	count := updater.GetSampleCount()
	if count != 1 {
		t.Errorf("Expected sample count 1, got %d", count)
	}
}

func TestEnhancedModelUpdaterQueueSampleLowConfidence(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	
	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
		},
	}
	
	err := updater.QueueSample(traceData, false, 0.5)
	if err != nil {
		t.Errorf("QueueSample should not return error: %v", err)
	}
	
	count := updater.GetSampleCount()
	if count != 0 {
		t.Errorf("Expected sample count 0 (low confidence), got %d", count)
	}
}

func TestEnhancedModelUpdaterClearSamples(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	
	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
		},
	}
	
	for i := 0; i < 5; i++ {
		_ = updater.QueueSample(traceData, false, 0.9)
	}
	
	updater.ClearSamples()
	
	count := updater.GetSampleCount()
	if count != 0 {
		t.Errorf("Expected sample count 0 after clear, got %d", count)
	}
}

func TestEnhancedModelUpdaterGetStats(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	
	stats := updater.GetStats()
	if stats == nil {
		t.Error("Stats should not be nil")
	}
	
	if stats["current_version"] != "v1.0.0" {
		t.Errorf("Expected version 'v1.0.0', got %v", stats["current_version"])
	}
}

func TestEnhancedModelUpdaterGetCurrentVersion(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	
	version := updater.GetCurrentVersion()
	if version != "v1.0.0" {
		t.Errorf("Expected version 'v1.0.0', got %s", version)
	}
}

func TestEnhancedModelUpdaterResetUpdateCount(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	
	updater.ResetUpdateCount()
	
	stats := updater.GetStats()
	if stats["update_count"] != 0 {
		t.Errorf("Expected update_count 0, got %v", stats["update_count"])
	}
}

func TestEnhancedModelUpdaterBatchUpdate(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	
	samples := make([]TrajectorySample, 3)
	for i := 0; i < 3; i++ {
		samples[i] = TrajectorySample{
			TraceData: &model.TraceData{
				Points: []model.TracePoint{
					{X: 0, Y: 0, Timestamp: 0},
					{X: float64(10*i), Y: float64(10*i), Timestamp: 100},
				},
			},
			IsBot:     false,
			Confidence: 0.9,
		}
	}
	
	request := BatchUpdateRequest{
		Samples: samples,
	}
	
	response, err := updater.BatchUpdate(request)
	if err != nil {
		t.Fatalf("BatchUpdate should not return error: %v", err)
	}
	
	if response == nil {
		t.Error("Response should not be nil")
	}
	
	if !response.Success {
		t.Error("Response should be successful")
	}
	
	if response.SamplesProcessed != 3 {
		t.Errorf("Expected SamplesProcessed 3, got %d", response.SamplesProcessed)
	}
}

func TestEnhancedModelUpdaterCalculateSampleDiversity(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	
	samples := []TrajectorySample{
		{FeatureVec: []float64{1.0, 2.0, 3.0}},
		{FeatureVec: []float64{4.0, 5.0, 6.0}},
		{FeatureVec: []float64{7.0, 8.0, 9.0}},
	}
	
	diversity := updater.CalculateSampleDiversity(samples)
	
	if diversity <= 0 {
		t.Errorf("Expected diversity > 0, got %f", diversity)
	}
}

func TestEnhancedModelUpdaterCalculateSampleDistance(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	
	s1 := TrajectorySample{FeatureVec: []float64{0.0, 0.0}}
	s2 := TrajectorySample{FeatureVec: []float64{3.0, 4.0}}
	
	distance := updater.calculateSampleDistance(s1, s2)
	
	if distance != 5.0 {
		t.Errorf("Expected distance 5.0, got %f", distance)
	}
}

func TestEnhancedModelUpdaterForceUpdate(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	updater.Start()
	
	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
		},
	}
	
	for i := 0; i < 10; i++ {
		_ = updater.QueueSample(traceData, false, 0.9)
	}
	
	result, err := updater.ForceUpdate()
	if err != nil {
		t.Fatalf("ForceUpdate should not return error: %v", err)
	}
	
	if result == nil {
		t.Error("Result should not be nil")
	}
	
	updater.Stop()
}

func TestEnhancedModelUpdaterForceUpdateNotEnoughSamples(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	updater.Start()
	
	_, err := updater.ForceUpdate()
	if err == nil {
		t.Error("ForceUpdate should return error when not enough samples")
	}
	
	updater.Stop()
}

func TestEnhancedModelUpdaterGetUpdateHistory(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	
	history := updater.GetUpdateHistory()
	
	_ = history
}

func TestEnhancedModelUpdaterGetModelVersions(t *testing.T) {
	traceService := NewTraceService()
	updater := NewEnhancedModelUpdater(traceService)
	
	versions := updater.GetModelVersions()
	
	_ = versions
}