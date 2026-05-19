package service

import (
	"testing"
)

func TestNewWASMAIInferenceEngine(t *testing.T) {
	engine := NewWASMAIInferenceEngine()
	if engine == nil {
		t.Fatal("Expected engine to be non-nil")
	}
}

func TestCreateLinearRegressionModel(t *testing.T) {
	engine := NewWASMAIInferenceEngine()
	model := engine.CreateLinearRegressionModel("test-linear", 5)
	
	if model == nil {
		t.Fatal("Expected model to be non-nil")
	}
	if model.ID != "test-linear" {
		t.Errorf("Expected ID 'test-linear', got %s", model.ID)
	}
	if model.Type != ModelTypeLinearRegression {
		t.Error("Expected LinearRegression model type")
	}
}

func TestCreateLogisticRegressionModel(t *testing.T) {
	engine := NewWASMAIInferenceEngine()
	model := engine.CreateLogisticRegressionModel("test-logistic", 10, 3)
	
	if model == nil {
		t.Fatal("Expected model to be non-nil")
	}
	if model.OutputSize != 3 {
		t.Errorf("Expected output size 3, got %d", model.OutputSize)
	}
}

func TestInference(t *testing.T) {
	engine := NewWASMAIInferenceEngine()
	engine.CreateLinearRegressionModel("test-infer", 3)
	
	input := []float32{1.0, 2.0, 3.0}
	result, err := engine.Inference("test-infer", input)
	
	if err != nil {
		t.Fatalf("Inference failed: %v", err)
	}
	if !result.Success {
		t.Error("Expected inference to succeed")
	}
	if len(result.Predictions) == 0 {
		t.Error("Expected predictions to be non-empty")
	}
}

func TestBatchInference(t *testing.T) {
	engine := NewWASMAIInferenceEngine()
	engine.CreateClassificationModel("test-batch", 4, 2)
	
	inputs := [][]float32{
		{1.0, 0.0, 0.0, 0.0},
		{0.0, 1.0, 0.0, 0.0},
		{0.0, 0.0, 1.0, 0.0},
	}
	
	results, err := engine.BatchInference("test-batch", inputs)
	if err != nil {
		t.Fatalf("Batch inference failed: %v", err)
	}
	if len(results) != len(inputs) {
		t.Errorf("Expected %d results, got %d", len(inputs), len(results))
	}
}

func TestPredictClass(t *testing.T) {
	engine := NewWASMAIInferenceEngine()
	engine.CreateClassificationModel("test-class", 5, 4)
	
	input := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	class, confidence, err := engine.PredictClass("test-class", input)
	
	if err != nil {
		t.Fatalf("PredictClass failed: %v", err)
	}
	if class < 0 || class >= 4 {
		t.Errorf("Expected class between 0-3, got %d", class)
	}
	if confidence < 0 || confidence > 1 {
		t.Errorf("Expected confidence between 0-1, got %f", confidence)
	}
}

func TestWASMAIInferenceGetStats(t *testing.T) {
	engine := NewWASMAIInferenceEngine()
	engine.CreateLinearRegressionModel("stats-test", 2)
	
	// 做一些推理
	input := []float32{0.5, 1.5}
	for i := 0; i < 10; i++ {
		engine.Inference("stats-test", input)
	}
	
	stats := engine.GetStats()
	if stats["inference_count"] != int64(10) {
		t.Error("Expected 10 inferences")
	}
}

func TestWASMAIInferenceResetStats(t *testing.T) {
	engine := NewWASMAIInferenceEngine()
	engine.CreateLinearRegressionModel("reset-test", 1)
	
	engine.Inference("reset-test", []float32{1.0})
	engine.ResetStats()
	
	stats := engine.GetStats()
	if stats["inference_count"] != int64(0) {
		t.Error("Expected inference count to be 0 after reset")
	}
}

func TestDeleteModel(t *testing.T) {
	engine := NewWASMAIInferenceEngine()
	engine.CreateLinearRegressionModel("delete-me", 3)
	
	// 检查模型存在
	models := engine.ListModels()
	if len(models) != 1 {
		t.Error("Expected 1 model")
	}
	
	engine.DeleteModel("delete-me")
	
	// 检查模型删除
	models = engine.ListModels()
	if len(models) != 0 {
		t.Error("Expected 0 models after deletion")
	}
}

func TestFeatureScaling(t *testing.T) {
	engine := NewWASMAIInferenceEngine()
	input := []float32{1.0, 3.0, 5.0, 7.0, 9.0}
	
	scaled := engine.FeatureScaling(input)
	for _, val := range scaled {
		if val < 0 || val > 1 {
			t.Error("Expected scaled values between 0 and 1")
		}
	}
}

func TestStandardization(t *testing.T) {
	engine := NewWASMAIInferenceEngine()
	input := []float32{2.0, 4.0, 6.0, 8.0, 10.0}
	
	standardized := engine.Standardization(input)
	mean := float32(0)
	for _, val := range standardized {
		mean += val
	}
	mean /= float32(len(standardized))
	
	// 均值应该接近 0
	if mean < -0.1 || mean > 0.1 {
		t.Error("Expected mean close to 0 after standardization")
	}
}
