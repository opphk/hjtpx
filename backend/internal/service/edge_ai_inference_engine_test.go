package service

import (
	"context"
	"testing"
	"time"
)

func TestEdgeAIInferenceEngine(t *testing.T) {
	engine := NewEdgeAIInferenceEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	if !engine.initialized {
		t.Error("Engine should be initialized")
	}
}

func TestEdgeModelManager(t *testing.T) {
	manager := NewEdgeModelManager()
	ctx := context.Background()

	if err := manager.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	models := manager.ListModels()
	if len(models) == 0 {
		t.Error("Should have at least one model")
	}

	model, err := manager.GetModel("edge_bot_detector_v1")
	if err != nil {
		t.Fatalf("Failed to get model: %v", err)
	}

	if model == nil {
		t.Error("Model should not be nil")
	}

	if err := manager.SetActiveModel("edge_behavior_analyzer_v1"); err != nil {
		t.Fatalf("Failed to set active model: %v", err)
	}

	if manager.activeModel != "edge_behavior_analyzer_v1" {
		t.Errorf("Expected active model to be edge_behavior_analyzer_v1, got %s", manager.activeModel)
	}
}

func TestLocalInferenceEngine(t *testing.T) {
	engine := NewLocalInferenceEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	model := &EdgeModel{
		ModelID:      "test_model",
		Name:         "Test Model",
		Version:      "1.0.0",
		Architecture: "lightweight_cnn",
		Weights:      []float64{0.1, 0.2, 0.3, 0.4, 0.5},
	}

	input := []float64{0.1, 0.2, 0.3, 0.4, 0.5}

	result, err := engine.Infer(ctx, model, input, nil)
	if err != nil {
		t.Fatalf("Failed to infer: %v", err)
	}

	if result == nil {
		t.Fatal("Inference result should not be nil")
	}

	if result.OutputData == nil {
		t.Error("Output data should not be nil")
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", result.Confidence)
	}
}

func TestOfflineValidator(t *testing.T) {
	validator := NewOfflineValidator()
	ctx := context.Background()

	if err := validator.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize validator: %v", err)
	}

	request := &OfflineValidationRequest{
		DataType: "behavior",
		Data:     map[string]interface{}{"click_count": 10, "scroll_speed": 0.5},
		Rules:    []string{"basic_check", "pattern_match"},
	}

	result, err := validator.Validate(ctx, request)
	if err != nil {
		t.Fatalf("Failed to validate: %v", err)
	}

	if result == nil {
		t.Fatal("Validation result should not be nil")
	}

	if !result.Success {
		t.Error("Validation should succeed")
	}
}

func TestOfflineValidationCaching(t *testing.T) {
	validator := NewOfflineValidator()
	ctx := context.Background()

	if err := validator.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize validator: %v", err)
	}

	request := &OfflineValidationRequest{
		DataType: "behavior",
		Data:     map[string]interface{}{"test": "data"},
		Rules:    []string{"basic_check"},
	}

	result1, err := validator.Validate(ctx, request)
	if err != nil {
		t.Fatalf("Failed to validate: %v", err)
	}

	result2, err := validator.Validate(ctx, request)
	if err != nil {
		t.Fatalf("Failed to validate again: %v", err)
	}

	if !result2.Cached {
		t.Error("Second validation should be cached")
	}
}

func TestDataMinimizer(t *testing.T) {
	minimizer := NewDataMinimizer()
	ctx := context.Background()

	if err := minimizer.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize minimizer: %v", err)
	}

	data := map[string]interface{}{
		"username":    "test_user",
		"password":    "secret123",
		"email":       "test@example.com",
		"ip_address":  "192.168.1.1",
		"timestamp":   time.Now(),
	}

	result, err := minimizer.Minimize(data, "field_removal")
	if err != nil {
		t.Fatalf("Failed to minimize data: %v", err)
	}

	if result["username"] == nil {
		t.Error("Username should be kept")
	}

	if result["password"] != nil {
		t.Error("Password should be removed")
	}
}

func TestPowerOptimizer(t *testing.T) {
	optimizer := NewPowerOptimizer()
	ctx := context.Background()

	if err := optimizer.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize optimizer: %v", err)
	}

	profile := optimizer.AdjustForPower(0.15)
	if profile.ProfileID != "low_power" {
		t.Errorf("Expected low_power profile for battery 0.15, got %s", profile.ProfileID)
	}

	profile = optimizer.AdjustForPower(0.35)
	if profile.ProfileID != "power_save" {
		t.Errorf("Expected power_save profile for battery 0.35, got %s", profile.ProfileID)
	}

	profile = optimizer.AdjustForPower(0.65)
	if profile.ProfileID != "balanced" {
		t.Errorf("Expected balanced profile for battery 0.65, got %s", profile.ProfileID)
	}

	profile = optimizer.AdjustForPower(0.95)
	if profile.ProfileID != "performance" {
		t.Errorf("Expected performance profile for battery 0.95, got %s", profile.ProfileID)
	}
}

func TestPerformInference(t *testing.T) {
	engine := NewEdgeAIInferenceEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	request := &InferenceRequest{
		ModelID:   "edge_bot_detector_v1",
		InputData: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		Options: &InferenceOptions{
			BatchSize:    1,
			Device:      "cpu",
			Quantization: false,
		},
	}

	result, err := engine.PerformInference(ctx, request)
	if err != nil {
		t.Fatalf("Failed to perform inference: %v", err)
	}

	if result == nil {
		t.Fatal("Inference result should not be nil")
	}

	if !result.Success {
		t.Error("Inference should succeed")
	}
}

func TestQuantizeWeights(t *testing.T) {
	engine := NewEdgeAIInferenceEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	weights := []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}
	quantized := engine.quantizeWeights(weights, 8)

	if len(quantized) != len(weights) {
		t.Errorf("Expected %d weights, got %d", len(weights), len(quantized))
	}
}

func TestOfflineValidation(t *testing.T) {
	engine := NewEdgeAIInferenceEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	request := &OfflineValidationRequest{
		DataType: "behavior",
		Data:     map[string]interface{}{"score": 85},
		Rules:    []string{"basic_check"},
	}

	result, err := engine.ValidateOffline(ctx, request)
	if err != nil {
		t.Fatalf("Failed to validate offline: %v", err)
	}

	if result == nil {
		t.Fatal("Validation result should not be nil")
	}
}

func TestMinimizeData(t *testing.T) {
	engine := NewEdgeAIInferenceEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	data := map[string]interface{}{
		"user_id": "123",
		"token":   "secret_token",
	}

	result, err := engine.MinimizeData(ctx, data, "field_removal")
	if err != nil {
		t.Fatalf("Failed to minimize data: %v", err)
	}

	if result["user_id"] == nil {
		t.Error("user_id should be kept")
	}

	if result["token"] != nil {
		t.Error("token should be removed")
	}
}

func TestAdjustPowerProfile(t *testing.T) {
	engine := NewEdgeAIInferenceEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	profile, err := engine.AdjustPowerProfile(ctx, 0.2)
	if err != nil {
		t.Fatalf("Failed to adjust power profile: %v", err)
	}

	if profile == nil {
		t.Fatal("Power profile should not be nil")
	}
}

func TestGetStats(t *testing.T) {
	engine := NewEdgeAIInferenceEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	stats, err := engine.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	if stats.TotalModels == 0 {
		t.Error("Should have models")
	}

	if !stats.OfflineMode {
		t.Error("Should be in offline mode")
	}
}
