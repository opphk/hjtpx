package edge

import (
	"context"
	"testing"
)

func TestEdgeAIEngine_NewEdgeAIEngine(t *testing.T) {
	engine := NewEdgeAIEngine(nil, nil, nil)

	if engine == nil {
		t.Fatal("Expected engine to not be nil")
	}

	if len(engine.models) != 0 {
		t.Errorf("Expected 0 models initially, got %d", len(engine.models))
	}
}

func TestEdgeAIEngine_RegisterModel(t *testing.T) {
	engine := NewEdgeAIEngine(nil, nil, nil)
	ctx := context.Background()

	model := &ModelInfo{
		ID:         "test-model",
		Name:       "Test Model",
		Version:    "1.0.0",
		Type:       ModelTypeONNX,
		TaskType:   TaskTypeClassification,
		InputShape: []int{1, 100},
		Labels:     []string{"class1", "class2", "class3"},
	}

	err := engine.RegisterModel(ctx, model)
	if err != nil {
		t.Fatalf("RegisterModel failed: %v", err)
	}

	retrieved, err := engine.GetModel("test-model")
	if err != nil {
		t.Fatalf("GetModel failed: %v", err)
	}

	if retrieved.Name != model.Name {
		t.Errorf("Expected name %s, got %s", model.Name, retrieved.Name)
	}
}

func TestEdgeAIEngine_UnregisterModel(t *testing.T) {
	engine := NewEdgeAIEngine(nil, nil, nil)
	ctx := context.Background()

	model := &ModelInfo{
		ID:       "test-model-2",
		Name:     "Test Model 2",
		Version:  "1.0.0",
		Type:     ModelTypePyTorch,
		TaskType: TaskTypeDetection,
	}

	engine.RegisterModel(ctx, model)

	err := engine.UnregisterModel(ctx, "test-model-2")
	if err != nil {
		t.Fatalf("UnregisterModel failed: %v", err)
	}

	_, err = engine.GetModel("test-model-2")
	if err == nil {
		t.Error("Expected error for unregistered model")
	}
}

func TestEdgeAIEngine_ListModels(t *testing.T) {
	engine := NewEdgeAIEngine(nil, nil, nil)
	ctx := context.Background()

	models := []*ModelInfo{
		{
			ID:       "model-1",
			Name:     "Model 1",
			Version:  "1.0.0",
			Type:     ModelTypeONNX,
			TaskType: TaskTypeClassification,
		},
		{
			ID:       "model-2",
			Name:     "Model 2",
			Version:  "1.0.0",
			Type:     ModelTypeTensorFlow,
			TaskType: TaskTypeDetection,
		},
	}

	for _, model := range models {
		engine.RegisterModel(ctx, model)
	}

	retrieved := engine.ListModels()
	if len(retrieved) != 2 {
		t.Errorf("Expected 2 models, got %d", len(retrieved))
	}
}

func TestEdgeAIEngine_LoadModel(t *testing.T) {
	engine := NewEdgeAIEngine(nil, nil, nil)
	ctx := context.Background()

	model := &ModelInfo{
		ID:       "test-model-3",
		Name:     "Test Model 3",
		Version:  "1.0.0",
		Type:     ModelTypeONNX,
		TaskType: TaskTypeClassification,
		FileSize: 1024 * 1024,
	}

	engine.RegisterModel(ctx, model)

	err := engine.LoadModel(ctx, "test-model-3")
	if err != nil {
		t.Fatalf("LoadModel failed: %v", err)
	}

	loadedModels := engine.GetLoadedModels()
	if len(loadedModels) != 1 {
		t.Errorf("Expected 1 loaded model, got %d", len(loadedModels))
	}
}

func TestEdgeAIEngine_Infer(t *testing.T) {
	engine := NewEdgeAIEngine(nil, nil, nil)
	ctx := context.Background()

	model := &ModelInfo{
		ID:         "test-model-4",
		Name:       "Test Model 4",
		Version:    "1.0.0",
		Type:       ModelTypeONNX,
		TaskType:   TaskTypeClassification,
		InputShape: []int{1, 100},
		Labels:     []string{"class1", "class2"},
		FileSize:   1024 * 1024,
	}

	engine.RegisterModel(ctx, model)

	req := &InferenceRequest{
		ModelID:   "test-model-4",
		InputData: []float32{1.0, 2.0},
		Options: &InferenceOptions{
			UseCache: true,
		},
	}

	response, err := engine.Infer(ctx, req)
	if err != nil {
		t.Fatalf("Infer failed: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response to not be nil")
	}

	if response.RequestID == "" {
		t.Error("Expected request ID to be set")
	}

	if response.LatencyMs < 0 {
		t.Error("Expected non-negative latency")
	}
}

func TestEdgeAIEngine_GetMetrics(t *testing.T) {
	engine := NewEdgeAIEngine(nil, nil, nil)

	metrics := engine.GetMetrics()

	if metrics == nil {
		t.Fatal("Expected metrics to not be nil")
	}

	if metrics.TotalRequests != 0 {
		t.Errorf("Expected 0 total requests, got %d", metrics.TotalRequests)
	}
}

func TestEdgeAIEngine_GetCacheStats(t *testing.T) {
	engine := NewEdgeAIEngine(nil, nil, nil)

	hits, misses, size := engine.GetCacheStats()

	_ = hits
	_ = misses
	_ = size
}

func TestInferenceCache_NewInferenceCache(t *testing.T) {
	cache := NewInferenceCache(100)

	if cache == nil {
		t.Fatal("Expected cache to not be nil")
	}

	if cache.maxSize != 100 {
		t.Errorf("Expected max size 100, got %d", cache.maxSize)
	}
}

func TestInferenceCache_GetSet(t *testing.T) {
	cache := NewInferenceCache(100)

	response := &InferenceResponse{
		RequestID: "test-request",
		ModelID:   "test-model",
		Predictions: map[string]interface{}{
			"class": "test",
		},
	}

	cache.Set("test-key", response)

	retrieved, exists := cache.Get("test-key")
	if !exists {
		t.Fatal("Expected entry to exist")
	}

	if retrieved.RequestID != response.RequestID {
		t.Errorf("Expected request ID %s, got %s", response.RequestID, retrieved.RequestID)
	}
}
