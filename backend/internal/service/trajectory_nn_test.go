package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestNNService(t *testing.T) {
	service := NewTrajectoryNNService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 25, Y: 25, Event: "move"},
			{Timestamp: 1300, X: 40, Y: 40, Event: "end"},
		},
		TotalTime: 300,
	}

	t.Run("Initialize", func(t *testing.T) {
		ctx := context.Background()
		err := service.Initialize(ctx)
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		if !service.IsLoaded() {
			t.Error("Service should be loaded after Initialize")
		}
	})

	t.Run("Predict Risk From Data", func(t *testing.T) {
		ctx := context.Background()
		result, err := service.PredictRiskFromData(ctx, traceData)
		if err != nil {
			t.Fatalf("PredictRiskFromData failed: %v", err)
		}

		if result == nil {
			t.Fatal("Result should not be nil")
		}

		if result.CombinedScore < 0 || result.CombinedScore > 1 {
			t.Errorf("Combined score should be between 0 and 1, got %f", result.CombinedScore)
		}

		t.Logf("Combined score: %f, Risk level: %s, IsBot: %v",
			result.CombinedScore, result.RiskLevel, result.IsBot)
	})

	t.Run("Predict Risk From JSON", func(t *testing.T) {
		ctx := context.Background()
		traceDataJSON, _ := json.Marshal(traceData)

		result, err := service.PredictRisk(ctx, traceDataJSON)
		if err != nil {
			t.Fatalf("PredictRisk failed: %v", err)
		}

		if result == nil {
			t.Fatal("Result should not be nil")
		}

		t.Logf("JSON-based prediction successful")
	})

	t.Run("Get Config", func(t *testing.T) {
		config := service.GetConfig()
		if config == nil {
			t.Fatal("Config should not be nil")
		}

		if !config.EnableLSTM {
			t.Error("LSTM should be enabled by default")
		}

		if !config.EnableTransformer {
			t.Error("Transformer should be enabled by default")
		}
	})

	t.Run("Get Model Info", func(t *testing.T) {
		info := service.GetModelInfo()
		if info == nil {
			t.Fatal("Model info should not be nil")
		}

		t.Logf("Model info: %+v", info)
	})
}

func TestNNServiceBatchPredict(t *testing.T) {
	service := NewTrajectoryNNService()

	traces := [][]byte{
		[]byte(`{"points":[{"t":1000,"x":0,"y":0,"e":"start"},{"t":1100,"x":10,"y":10,"e":"move"},{"t":1200,"x":20,"y":20,"e":"end"}],"total_time":200}`),
		[]byte(`{"points":[{"t":1000,"x":0,"y":0,"e":"start"},{"t":1100,"x":15,"y":15,"e":"move"},{"t":1200,"x":30,"y":30,"e":"end"}],"total_time":200}`),
	}

	ctx := context.Background()
	results, err := service.BatchPredict(ctx, traces)
	if err != nil {
		t.Fatalf("BatchPredict failed: %v", err)
	}

	if len(results) != len(traces) {
		t.Errorf("Expected %d results, got %d", len(traces), len(results))
	}

	for i, result := range results {
		t.Logf("Batch result %d: score=%f, risk=%s", i, result.CombinedScore, result.RiskLevel)
	}
}

func TestNNServiceDisabled(t *testing.T) {
	config := &ModelConfig{
		EnableLSTM:        false,
		EnableTransformer: false,
	}

	service := NewTrajectoryNNService()
	ctx := context.Background()
	
	if err := service.UpdateConfig(config); err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 20, Event: "end"},
		},
	}

	result, err := service.PredictRiskFromData(ctx, traceData)
	if err != nil {
		t.Fatalf("PredictRiskFromData failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil even when models are disabled")
	}

	if result.CombinedScore == 0 {
		t.Logf("Models disabled, using default score")
	}
}

func TestNNServiceLoadWeights(t *testing.T) {
	service := NewTrajectoryNNService()

	config := &ModelConfig{
		LSTMWeightsPath:        "/tmp/lstm_weights.bin",
		TransformerWeightsPath: "/tmp/transformer_weights.bin",
		EnableLSTM:             true,
		EnableTransformer:      true,
	}

	ctx := context.Background()
	err := service.LoadModelWeights(ctx, config)
	if err != nil {
		t.Logf("LoadModelWeights returned error (expected for non-existent files): %v", err)
	}

	info := service.GetModelInfo()
	if info == nil {
		t.Fatal("Model info should not be nil")
	}

	if !info["is_loaded"].(bool) {
		t.Error("Service should be loaded")
	}
}

func TestNNServiceExtractFeatures(t *testing.T) {
	service := NewTrajectoryNNService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 25, Y: 25, Event: "move"},
			{Timestamp: 1300, X: 40, Y: 40, Event: "end"},
		},
		TotalTime: 300,
	}

	ctx := context.Background()
	features, err := service.ExtractFeatures(ctx, traceData)
	if err != nil {
		t.Fatalf("ExtractFeatures failed: %v", err)
	}

	if features == nil {
		t.Fatal("Features should not be nil")
	}

	if len(features) == 0 {
		t.Error("Features should not be empty")
	}

	t.Logf("Extracted %d features", len(features))
}

func TestNNServiceAnalyzeTrajectory(t *testing.T) {
	service := NewTrajectoryNNService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 25, Y: 25, Event: "move"},
			{Timestamp: 1300, X: 40, Y: 40, Event: "end"},
		},
		TotalTime: 300,
	}

	ctx := context.Background()
	result, err := service.AnalyzeTrajectory(ctx, traceData)
	if err != nil {
		t.Fatalf("AnalyzeTrajectory failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.TotalRiskScore < 0 || result.TotalRiskScore > 1 {
		t.Errorf("Total risk score should be between 0 and 1, got %f", result.TotalRiskScore)
	}

	t.Logf("Total risk score: %f, Risk level: %s", result.TotalRiskScore, result.RiskLevel)
}

func TestNNServiceUpdateConfig(t *testing.T) {
	service := NewTrajectoryNNService()

	config := &ModelConfig{
		EnableLSTM:          true,
		EnableTransformer:   false,
		ConfidenceThreshold: 0.8,
		RiskThreshold:       0.6,
	}

	err := service.UpdateConfig(config)
	if err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	retrievedConfig := service.GetConfig()
	if retrievedConfig.EnableTransformer {
		t.Error("Transformer should be disabled after UpdateConfig")
	}

	if retrievedConfig.ConfidenceThreshold != 0.8 {
		t.Errorf("ConfidenceThreshold should be 0.8, got %f", retrievedConfig.ConfidenceThreshold)
	}
}

func TestNNServiceShutdown(t *testing.T) {
	service := NewTrajectoryNNService()

	ctx := context.Background()
	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !service.IsLoaded() {
		t.Error("Service should be loaded")
	}

	if err := service.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	if service.IsLoaded() {
		t.Error("Service should not be loaded after Shutdown")
	}
}
