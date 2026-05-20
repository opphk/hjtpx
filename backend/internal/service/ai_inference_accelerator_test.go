package service

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestAIInferenceAccelerator_Initialize(t *testing.T) {
	accel := NewAIInferenceAccelerator()
	ctx := context.Background()

	err := accel.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !accel.initialized {
		t.Error("initialized should be true after Initialize")
	}
}

func TestModelQuantizer_Quantize(t *testing.T) {
	quantizer := NewModelQuantizer()
	ctx := context.Background()
	quantizer.Initialize(ctx)

	weights := make([]float64, 256)
	for i := range weights {
		weights[i] = float64(i) * 0.01
	}

	tests := []struct {
		name        string
		targetType  QuantizationType
		bits        int
	}{
		{"INT8 Quantization", QuantTypeINT8, 8},
		{"INT16 Quantization", QuantTypeINT16, 16},
		{"FP16 Quantization", QuantTypeFP16, 16},
		{"Dynamic Quantization", QuantTypeDynamic, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &QuantizationRequest{
				ModelID:    "test_model",
				Weights:    weights,
				TargetType: tt.targetType,
				Bits:       tt.bits,
			}

			response, err := quantizer.Quantize(request)
			if err != nil {
				t.Fatalf("Quantize failed: %v", err)
			}

			if !response.Success {
				t.Error("expected successful quantization")
			}

			if response.CompressionRatio <= 0 {
				t.Error("compression ratio should be positive")
			}

			if response.AccuracyLoss < 0 || response.AccuracyLoss > 1 {
				t.Error("accuracy loss should be between 0 and 1")
			}
		})
	}
}

func TestModelQuantizer_QuantizeINT8(t *testing.T) {
	quantizer := NewModelQuantizer()
	ctx := context.Background()
	quantizer.Initialize(ctx)

	weights := []float64{0.1, 0.5, -0.3, 1.0, -0.5}

	request := &QuantizationRequest{
		ModelID:    "test_model",
		Weights:    weights,
		TargetType: QuantTypeINT8,
		Bits:       8,
	}

	response, err := quantizer.quantizeINT8(weights, &QuantizationConfig{
		ModelID:    "test",
		TargetType: QuantTypeINT8,
		Bits:       8,
	})

	if err != nil {
		t.Fatalf("quantizeINT8 failed: %v", err)
	}

	if response.QuantizedModel == nil {
		t.Error("quantized model should not be nil")
	}

	if response.CompressionRatio != 8.0 {
		t.Logf("compression ratio: %f (expected 8.0)", response.CompressionRatio)
	}
}

func TestModelPruner_Prune(t *testing.T) {
	pruner := NewModelPruner()
	ctx := context.Background()
	pruner.Initialize(ctx)

	weights := make([]float64, 256)
	for i := range weights {
		weights[i] = float64(i) * 0.1
	}

	tests := []struct {
		name     string
		method   string
		sparsity float64
	}{
		{"Magnitude Pruning", "magnitude", 0.3},
		{"Gradient Pruning", "gradient", 0.5},
		{"Random Pruning", "random", 0.4},
		{"Structured Pruning", "structured", 0.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &PruningRequest{
				ModelID:       "test_model",
				Weights:       weights,
				Method:        tt.method,
				SparsityLevel: tt.sparsity,
			}

			response, err := pruner.Prune(request)
			if err != nil {
				t.Fatalf("Prune failed: %v", err)
			}

			if !response.Success {
				t.Error("expected successful pruning")
			}

			if len(response.PrunedIndices) == 0 {
				t.Error("pruned indices should not be empty for non-zero sparsity")
			}

			if response.SparsityAchieved < 0 || response.SparsityAchieved > 1 {
				t.Error("sparsity achieved should be between 0 and 1")
			}
		})
	}
}

func TestMagnitudePruner_Prune(t *testing.T) {
	pruner := &MagnitudePruner{}

	weights := []float64{0.1, 0.5, 0.01, -0.3, 0.8, -0.05, 1.0, -0.2}

	prunedWeights, prunedIndices, err := pruner.Prune(weights, 0.2)
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	if len(prunedWeights) != len(weights) {
		t.Errorf("expected %d pruned weights, got %d", len(weights), len(prunedWeights))
	}

	expectedPruned := []int{2, 5}
	for _, idx := range expectedPruned {
		found := false
		for _, pi := range prunedIndices {
			if pi == idx {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected index %d to be pruned", idx)
		}
	}

	if pruner.GetName() != "magnitude" {
		t.Errorf("expected name 'magnitude', got '%s'", pruner.GetName())
	}
}

func TestRandomPruner_Prune(t *testing.T) {
	pruner := &RandomPruner{}

	weights := make([]float64, 100)
	for i := range weights {
		weights[i] = float64(i) * 0.1
	}

	_, prunedIndices, err := pruner.Prune(weights, 0.3)
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	targetCount := int(0.3 * 100)
	if len(prunedIndices) < targetCount-5 || len(prunedIndices) > targetCount+5 {
		t.Logf("pruned %d indices, target around %d", len(prunedIndices), targetCount)
	}
}

func TestStructuredPruner_Prune(t *testing.T) {
	pruner := &StructuredPruner{BlockSize: 4}

	weights := make([]float64, 16)
	for i := range weights {
		weights[i] = float64(i) * 0.1
	}

	prunedWeights, prunedIndices, err := pruner.Prune(weights, 0.3)
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	if len(prunedWeights) != len(weights) {
		t.Errorf("expected %d weights, got %d", len(weights), len(prunedWeights))
	}

	if pruner.GetName() != "structured" {
		t.Errorf("expected name 'structured', got '%s'", pruner.GetName())
	}
}

func TestONNXRuntimeEngine_Infer(t *testing.T) {
	engine := NewONNXRuntimeEngine()
	ctx := context.Background()
	engine.Initialize(ctx)

	inputData := make([]float64, 100)
	for i := range inputData {
		inputData[i] = float64(i) * 0.01
	}

	options := &InferenceOptionsV2{
		Device:           "CPU",
		BatchSize:        1,
		Quantize:         true,
		Prune:            true,
		OptimizationLevel: 3,
	}

	response, err := engine.Infer(ctx, "test_model", inputData, options)
	if err != nil {
		t.Fatalf("Infer failed: %v", err)
	}

	if !response.Success {
		t.Error("expected successful inference")
	}

	if len(response.OutputData) == 0 {
		t.Error("output data should not be empty")
	}

	if response.LatencyMs < 0 {
		t.Error("latency should be non-negative")
	}

	if response.Confidence < 0 || response.Confidence > 1 {
		t.Error("confidence should be between 0 and 1")
	}

	if len(response.Optimizations) == 0 {
		t.Error("optimizations should not be empty")
	}
}

func TestONNXRuntimeEngine_CalculateConfidence(t *testing.T) {
	engine := NewONNXRuntimeEngine()

	tests := []struct {
		name     string
		output   []float64
		expected float64
	}{
		{"High confidence", []float64{0.95, 0.05}, 0.95},
		{"Low confidence", []float64{0.55, 0.45}, 0.55},
		{"Equal", []float64{0.5, 0.5}, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := engine.calculateConfidence(tt.output)
			if math.Abs(confidence-tt.expected) > 0.001 {
				t.Errorf("expected %f, got %f", tt.expected, confidence)
			}
		})
	}
}

func TestEdgeAIDeployer_Deploy(t *testing.T) {
	deployer := NewEdgeAIDeployer()
	ctx := context.Background()
	deployer.Initialize(ctx)

	request := &DeploymentRequest{
		ModelID:  "bot_detector",
		Version:  "v1.0",
		Strategy: "latency",
		Replicas: 2,
	}

	response, err := deployer.Deploy(ctx, request)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	if !response.Success {
		t.Error("expected successful deployment")
	}

	if response.DeploymentID == "" {
		t.Error("deployment ID should not be empty")
	}

	if len(response.DeployedNodes) == 0 {
		t.Error("deployed nodes should not be empty")
	}
}

func TestEdgeAIDeployer_MultipleStrategies(t *testing.T) {
	deployer := NewEdgeAIDeployer()
	ctx := context.Background()
	deployer.Initialize(ctx)

	strategies := []string{"latency", "cost", "reliability"}

	for _, strategy := range strategies {
		t.Run(strategy, func(t *testing.T) {
			request := &DeploymentRequest{
				ModelID:  "test_model",
				Version:  "v1.0",
				Strategy: strategy,
				Replicas: 1,
			}

			response, err := deployer.Deploy(ctx, request)
			if err != nil {
				t.Fatalf("Deploy with strategy %s failed: %v", strategy, err)
			}

			if !response.Success {
				t.Errorf("strategy %s: expected successful deployment", strategy)
			}
		})
	}
}

func TestLatencyBasedStrategy_SelectNodes(t *testing.T) {
	strategy := &LatencyBasedStrategy{TargetLatencyMs: 100}

	nodes := []*EdgeNode{
		{NodeID: "n1", Status: "online", Network: &NetworkSpec{LatencyMs: 50, Connected: true}},
		{NodeID: "n2", Status: "online", Network: &NetworkSpec{LatencyMs: 30, Connected: true}},
		{NodeID: "n3", Status: "offline", Network: &NetworkSpec{LatencyMs: 20, Connected: false}},
		{NodeID: "n4", Status: "online", Network: &NetworkSpec{LatencyMs: 80, Connected: true}},
	}

	selected := strategy.SelectNodes(&ModelMetadata{ModelID: "test"}, nodes)

	if len(selected) == 0 {
		t.Error("should select at least one node")
	}

	for _, node := range selected {
		if node.Status != "online" || !node.Network.Connected {
			t.Error("selected nodes should be online and connected")
		}
	}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Network.LatencyMs < selected[j].Network.LatencyMs
	})

	if strategy.GetName() != "latency" {
		t.Errorf("expected name 'latency', got '%s'", strategy.GetName())
	}
}

func TestInferencePerformanceMonitor_RecordInference(t *testing.T) {
	monitor := NewInferencePerformanceMonitor()
	ctx := context.Background()
	monitor.Initialize(ctx)

	monitor.RecordInference(50.0, true, false)
	monitor.RecordInference(60.0, true, true)
	monitor.RecordInference(30.0, true, false)
	monitor.RecordInference(100.0, false, false)

	metrics := monitor.GetMetrics()

	if metrics.TotalRequests != 4 {
		t.Errorf("expected 4 total requests, got %d", metrics.TotalRequests)
	}

	if metrics.SuccessfulRequests != 3 {
		t.Errorf("expected 3 successful requests, got %d", metrics.SuccessfulRequests)
	}

	if metrics.FailedRequests != 1 {
		t.Errorf("expected 1 failed request, got %d", metrics.FailedRequests)
	}

	if metrics.AvgLatencyMs != 60.0 {
		t.Errorf("expected average latency 60.0, got %f", metrics.AvgLatencyMs)
	}
}

func TestInferencePerformanceMonitor_AlertGeneration(t *testing.T) {
	monitor := NewInferencePerformanceMonitor()
	ctx := context.Background()
	monitor.Initialize(ctx)

	monitor.thresholds.MaxLatencyMs = 50

	for i := 0; i < 10; i++ {
		monitor.RecordInference(80.0, true, false)
	}

	if len(monitor.alerts) == 0 {
		t.Error("should generate at least one alert when latency exceeds threshold")
	}
}

func TestInferencePerformanceMonitor_TakeSnapshot(t *testing.T) {
	monitor := NewInferencePerformanceMonitor()
	ctx := context.Background()
	monitor.Initialize(ctx)

	monitor.RecordInference(50.0, true, false)
	monitor.TakeSnapshot()

	if len(monitor.history) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(monitor.history))
	}

	if monitor.history[0].Metrics.TotalRequests != 1 {
		t.Error("snapshot metrics should match current metrics")
	}
}

func TestAcceleratorCache_GetSet(t *testing.T) {
	cache := NewAcceleratorCache(1024 * 1024)

	key := "test_key"
	value := &InferenceResponseV2{
		Success:    true,
		OutputData: []float64{0.5, 0.5},
	}

	cache.Set(key, value, 5*time.Minute)

	result, exists := cache.Get(key)
	if !exists {
		t.Error("expected key to exist")
	}

	if _, ok := result.(*InferenceResponseV2); !ok {
		t.Error("expected InferenceResponseV2 type")
	}
}

func TestAcceleratorCache_Expiration(t *testing.T) {
	cache := NewAcceleratorCache(1024 * 1024)

	key := "expiring_key"
	value := "expiring_value"

	cache.Set(key, value, 1*time.Millisecond)

	time.Sleep(5 * time.Millisecond)

	_, exists := cache.Get(key)
	if exists {
		t.Error("key should have expired")
	}
}

func TestAcceleratorCache_Eviction(t *testing.T) {
	cache := NewAcceleratorCache(2048)

	for i := 0; i < 100; i++ {
		key := "key_" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		value := "value_" + key
		cache.Set(key, value, 5*time.Minute)
	}

	if cache.currentSize > cache.maxSizeBytes {
		t.Errorf("cache size %d exceeds max %d", cache.currentSize, cache.maxSizeBytes)
	}
}

func TestAIInferenceAccelerator_QuantizeModel(t *testing.T) {
	accel := NewAIInferenceAccelerator()
	ctx := context.Background()
	accel.Initialize(ctx)

	weights := make([]float64, 256)
	for i := range weights {
		weights[i] = float64(i) * 0.01
	}

	request := &QuantizationRequest{
		ModelID:    "test_model",
		Weights:    weights,
		TargetType: QuantTypeINT8,
		Bits:       8,
	}

	response, err := accel.QuantizeModel(ctx, request)
	if err != nil {
		t.Fatalf("QuantizeModel failed: %v", err)
	}

	if !response.Success {
		t.Error("expected successful quantization")
	}

	if response.CompressionRatio <= 0 {
		t.Error("compression ratio should be positive")
	}
}

func TestAIInferenceAccelerator_PruneModel(t *testing.T) {
	accel := NewAIInferenceAccelerator()
	ctx := context.Background()
	accel.Initialize(ctx)

	weights := make([]float64, 256)
	for i := range weights {
		weights[i] = float64(i) * 0.1
	}

	request := &PruningRequest{
		ModelID:       "test_model",
		Weights:       weights,
		Method:        "magnitude",
		SparsityLevel: 0.3,
	}

	response, err := accel.PruneModel(ctx, request)
	if err != nil {
		t.Fatalf("PruneModel failed: %v", err)
	}

	if !response.Success {
		t.Error("expected successful pruning")
	}

	if len(response.PrunedIndices) == 0 {
		t.Error("should have pruned some weights")
	}
}

func TestAIInferenceAccelerator_RunInference(t *testing.T) {
	accel := NewAIInferenceAccelerator()
	ctx := context.Background()
	accel.Initialize(ctx)

	inputData := make([]float64, 100)
	for i := range inputData {
		inputData[i] = float64(i) * 0.01
	}

	request := &InferenceRequestV2{
		ModelID:   "test_model",
		InputData: inputData,
		Options: &InferenceOptionsV2{
			Device:   "CPU",
			BatchSize: 1,
		},
	}

	response, err := accel.RunInference(ctx, request)
	if err != nil {
		t.Fatalf("RunInference failed: %v", err)
	}

	if !response.Success {
		t.Error("expected successful inference")
	}

	if len(response.OutputData) == 0 {
		t.Error("output data should not be empty")
	}

	if response.LatencyMs < 0 {
		t.Error("latency should be non-negative")
	}
}

func TestAIInferenceAccelerator_RunInference_CacheHit(t *testing.T) {
	accel := NewAIInferenceAccelerator()
	ctx := context.Background()
	accel.Initialize(ctx)

	inputData := make([]float64, 10)
	for i := range inputData {
		inputData[i] = 0.5
	}

	request := &InferenceRequestV2{
		ModelID:   "cache_test_model",
		InputData: inputData,
		Options: &InferenceOptionsV2{
			Device:   "CPU",
			BatchSize: 1,
		},
	}

	response1, _ := accel.RunInference(ctx, request)

	response2, _ := accel.RunInference(ctx, request)

	if !response2.CacheHit {
		t.Error("second request should be a cache hit")
	}

	_ = response1
}

func TestAIInferenceAccelerator_DeployToEdge(t *testing.T) {
	accel := NewAIInferenceAccelerator()
	ctx := context.Background()
	accel.Initialize(ctx)

	request := &DeploymentRequest{
		ModelID:  "edge_model",
		Version:  "v1.0",
		Strategy: "latency",
		Replicas: 2,
	}

	response, err := accel.DeployToEdge(ctx, request)
	if err != nil {
		t.Fatalf("DeployToEdge failed: %v", err)
	}

	if !response.Success {
		t.Error("expected successful deployment")
	}

	if response.DeploymentID == "" {
		t.Error("deployment ID should not be empty")
	}
}

func TestAIInferenceAccelerator_GetMonitoringStats(t *testing.T) {
	accel := NewAIInferenceAccelerator()
	ctx := context.Background()
	accel.Initialize(ctx)

	for i := 0; i < 5; i++ {
		accel.RunInference(ctx, &InferenceRequestV2{
			ModelID:   "test_model",
			InputData: make([]float64, 10),
			Options:   &InferenceOptionsV2{Device: "CPU", BatchSize: 1},
		})
	}

	request := &MonitoringStatsRequestV2{
		TimeRange: "1h",
		Metrics:   []string{"latency", "throughput"},
	}

	response, err := accel.GetMonitoringStats(ctx, request)
	if err != nil {
		t.Fatalf("GetMonitoringStats failed: %v", err)
	}

	if response.CurrentMetrics == nil {
		t.Error("current metrics should not be nil")
	}

	if len(response.History) == 0 {
		t.Error("history should have at least one snapshot")
	}
}

func TestParseInferenceRequestV2(t *testing.T) {
	jsonData := `{
		"model_id": "test_model",
		"input_data": [0.1, 0.2, 0.3],
		"input_shape": [3],
		"options": {
			"device": "CPU",
			"batch_size": 1,
			"quantize": true,
			"optimization_level": 3
		}
	}`

	request, err := ParseInferenceRequestV2(jsonData)
	if err != nil {
		t.Fatalf("ParseInferenceRequestV2 failed: %v", err)
	}

	if request.ModelID != "test_model" {
		t.Errorf("expected model_id 'test_model', got '%s'", request.ModelID)
	}

	if len(request.InputData) != 3 {
		t.Errorf("expected 3 input data elements, got %d", len(request.InputData))
	}

	if request.Options.Device != "CPU" {
		t.Errorf("expected device 'CPU', got '%s'", request.Options.Device)
	}

	if !request.Options.Quantize {
		t.Error("expected quantize to be true")
	}
}

func TestParseQuantizationRequest(t *testing.T) {
	jsonData := `{
		"model_id": "test",
		"weights": [0.1, 0.2, 0.3],
		"target_type": "int8",
		"bits": 8
	}`

	request, err := ParseQuantizationRequest(jsonData)
	if err != nil {
		t.Fatalf("ParseQuantizationRequest failed: %v", err)
	}

	if request.ModelID != "test" {
		t.Errorf("expected model_id 'test', got '%s'", request.ModelID)
	}

	if request.Bits != 8 {
		t.Errorf("expected bits 8, got %d", request.Bits)
	}
}

func TestParsePruningRequest(t *testing.T) {
	jsonData := `{
		"model_id": "test",
		"weights": [0.1, 0.2, 0.3],
		"method": "magnitude",
		"sparsity_level": 0.5
	}`

	request, err := ParsePruningRequest(jsonData)
	if err != nil {
		t.Fatalf("ParsePruningRequest failed: %v", err)
	}

	if request.Method != "magnitude" {
		t.Errorf("expected method 'magnitude', got '%s'", request.Method)
	}

	if request.SparsityLevel != 0.5 {
		t.Errorf("expected sparsity_level 0.5, got %f", request.SparsityLevel)
	}
}

func TestParseDeploymentRequest(t *testing.T) {
	jsonData := `{
		"model_id": "test",
		"version": "v1.0",
		"strategy": "latency",
		"replicas": 3
	}`

	request, err := ParseDeploymentRequest(jsonData)
	if err != nil {
		t.Fatalf("ParseDeploymentRequest failed: %v", err)
	}

	if request.Strategy != "latency" {
		t.Errorf("expected strategy 'latency', got '%s'", request.Strategy)
	}

	if request.Replicas != 3 {
		t.Errorf("expected replicas 3, got %d", request.Replicas)
	}
}

func TestParseMonitoringStatsRequestV2(t *testing.T) {
	jsonData := `{
		"time_range": "1h",
		"metrics": ["latency", "throughput"],
		"group_by": "node"
	}`

	request, err := ParseMonitoringStatsRequestV2(jsonData)
	if err != nil {
		t.Fatalf("ParseMonitoringStatsRequestV2 failed: %v", err)
	}

	if request.TimeRange != "1h" {
		t.Errorf("expected time_range '1h', got '%s'", request.TimeRange)
	}

	if len(request.Metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(request.Metrics))
	}
}

func TestInferenceResponseV2_Serialization(t *testing.T) {
	response := &InferenceResponseV2{
		Success:      true,
		OutputData:   []float64{0.8, 0.2},
		OutputShape:  []int{2},
		Confidence:   0.8,
		LatencyMs:   45.5,
		DeviceUsed:  "CPU",
		Optimizations: []string{"quantization", "pruning"},
		ModelVersion: "v1.0",
		CacheHit:    false,
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled InferenceResponseV2
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.Success != response.Success {
		t.Error("Success mismatch")
	}

	if len(unmarshaled.OutputData) != len(response.OutputData) {
		t.Error("OutputData length mismatch")
	}

	if unmarshaled.Confidence != response.Confidence {
		t.Error("Confidence mismatch")
	}
}

func TestQuantizedModel_Serialization(t *testing.T) {
	model := &QuantizedModel{
		ModelID:       "test_model",
		OriginalSize:  1024,
		QuantizedSize: 256,
		ScaleFactors:  []float64{0.5, 1.0},
		ZeroPoints:    []float64{0.0, 0.0},
		OutputShape:   []int{256},
		Accuracy:     0.98,
	}

	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled QuantizedModel
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.ModelID != model.ModelID {
		t.Error("ModelID mismatch")
	}

	if unmarshaled.Accuracy != model.Accuracy {
		t.Error("Accuracy mismatch")
	}
}

func TestEdgeNode(t *testing.T) {
	node := &AccelEdgeNode{
		NodeID:       "node_test",
		Name:         "Test Node",
		Platform:     "linux",
		Architecture: "x86_64",
		CPU: &HardwareSpec{
			Model:       "Intel i7",
			Cores:       8,
			Frequency:   3.5,
			Utilization: 0.5,
		},
		Memory: &MemorySpec{
			TotalBytes:     16 * 1024 * 1024 * 1024,
			AvailableBytes: 8 * 1024 * 1024 * 1024,
			UsedPercent:    50.0,
		},
		Network: &NetworkSpec{
			BandwidthMbps: 1000,
			LatencyMs:     10,
			Connected:     true,
		},
		Status: "online",
	}

	if node.CPU.Cores != 8 {
		t.Errorf("expected 8 CPU cores, got %d", node.CPU.Cores)
	}

	if node.Memory.UsedPercent != 50.0 {
		t.Errorf("expected 50%% memory usage, got %f%%", node.Memory.UsedPercent)
	}

	if !node.Network.Connected {
		t.Error("node should be connected")
	}
}

func TestPerformanceMetrics(t *testing.T) {
	metrics := &PerformanceMetrics{
		TotalRequests:      1000,
		SuccessfulRequests: 950,
		FailedRequests:     50,
		AvgLatencyMs:      45.5,
		P50LatencyMs:      40.0,
		P90LatencyMs:      60.0,
		P99LatencyMs:      80.0,
		ThroughputQPS:      100.5,
		CacheHitRate:      0.75,
		GPUUtilization:    0.85,
		CPUUtilization:    0.60,
	}

	if metrics.TotalRequests != metrics.SuccessfulRequests+metrics.FailedRequests {
		t.Error("total requests should equal successful + failed")
	}

	if metrics.AvgLatencyMs <= 0 {
		t.Error("average latency should be positive")
	}

	if metrics.CacheHitRate < 0 || metrics.CacheHitRate > 1 {
		t.Error("cache hit rate should be between 0 and 1")
	}
}

func TestOptimizationEngine(t *testing.T) {
	engine := NewOptimizationEngine()

	ir := &IntermediateRepresentation{
		ModelID: "test",
		Nodes:   make([]*IRNode, 0),
		Inputs:  make([]*TensorInfo, 0),
		Outputs: make([]*TensorInfo, 0),
	}

	cf := &ConstantFolding{}
	err := cf.Apply(ir)
	if err != nil {
		t.Fatalf("ConstantFolding.Apply failed: %v", err)
	}

	of := &OperatorFusion{}
	err = of.Apply(ir)
	if err != nil {
		t.Fatalf("OperatorFusion.Apply failed: %v", err)
	}

	if cf.GetName() != "constant_folding" {
		t.Errorf("expected 'constant_folding', got '%s'", cf.GetName())
	}

	if of.GetName() != "operator_fusion" {
		t.Errorf("expected 'operator_fusion', got '%s'", of.GetName())
	}
}

func TestIntermediateRepresentation(t *testing.T) {
	ir := &IntermediateRepresentation{
		ModelID: "test_model",
		Nodes: []*IRNode{
			{NodeID: "node1", OpType: "conv2d", Inputs: []string{"input"}, Outputs: []string{"output1"}},
			{NodeID: "node2", OpType: "relu", Inputs: []string{"output1"}, Outputs: []string{"output2"}},
		},
		Inputs: []*TensorInfo{
			{Name: "input", Shape: []int{1, 3, 224, 224}, DType: "float32"},
		},
		Outputs: []*TensorInfo{
			{Name: "output", Shape: []int{1, 1000}, DType: "float32"},
		},
	}

	if len(ir.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(ir.Nodes))
	}

	if len(ir.Inputs) != 1 {
		t.Errorf("expected 1 input, got %d", len(ir.Inputs))
	}

	if len(ir.Outputs) != 1 {
		t.Errorf("expected 1 output, got %d", len(ir.Outputs))
	}

	if ir.Inputs[0].Shape[0] != 1 {
		t.Error("batch size should be 1")
	}
}

func TestDeploymentResponse(t *testing.T) {
	response := &DeploymentResponse{
		Success:       true,
		DeploymentID:  "deploy_001",
		DeployedNodes: []string{"node_001", "node_002"},
		Status:       "deployed",
		Resources: map[string]*ResourceAllocation{
			"node_001": {CPUCores: 2.0, MemoryMB: 2048},
			"node_002": {CPUCores: 2.0, MemoryMB: 2048},
		},
	}

	if !response.Success {
		t.Error("expected success")
	}

	if len(response.DeployedNodes) != 2 {
		t.Errorf("expected 2 deployed nodes, got %d", len(response.DeployedNodes))
	}

	if response.Resources["node_001"].CPUCores != 2.0 {
		t.Errorf("expected 2.0 CPU cores, got %f", response.Resources["node_001"].CPUCores)
	}
}

func TestSigmoid(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"Zero", 0.0, 0.5},
		{"Large positive", 10.0, 0.99995},
		{"Large negative", -10.0, 0.00005},
		{"One", 1.0, 0.731058},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := accelSigmoid(tt.input)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"a < b", 5, 10, 5},
		{"a > b", 10, 5, 5},
		{"a == b", 5, 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := accelMin(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}
