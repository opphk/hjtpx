package service

import (
	"context"
	"testing"
	"time"
)

func TestK8sAutoscaler(t *testing.T) {
	config := &AutoscalerConfig{
		MinReplicas: 1,
		MaxReplicas: 10,
	}

	autoscaler := NewK8sAutoscaler(config)
	defer autoscaler.Stop()

	if err := autoscaler.Start(); err != nil {
		t.Errorf("Start() failed: %v", err)
	}

	stats := autoscaler.GetStats()
	if stats == nil {
		t.Error("GetStats() returned nil")
	}
}

func TestK8sAutoscalerScaling(t *testing.T) {
	config := DefaultAutoscalerConfig()
	autoscaler := NewK8sAutoscaler(config)
	defer autoscaler.Stop()

	autoscaler.Start()

	autoscaler.metricsCollector.Record("cpu", 85.0)
	autoscaler.metricsCollector.Record("memory", 80.0)

	time.Sleep(2 * time.Second)

	stats := autoscaler.GetStats()
	if stats["current_replicas"].(int64) < 1 {
		t.Errorf("Expected at least 1 replica, got %d", stats["current_replicas"])
	}
}

func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector(5 * time.Minute)

	collector.Record("cpu", 50.0)
	collector.Record("cpu", 60.0)
	collector.Record("cpu", 70.0)

	collector.Record("memory", 40.0)
	collector.Record("memory", 50.0)

	averages := collector.GetAverages()
	if averages["cpu"] < 50.0 || averages["cpu"] > 70.0 {
		t.Errorf("CPU average out of range: %f", averages["cpu"])
	}
}

func TestResourcePredictor(t *testing.T) {
	predictor := NewResourcePredictor(10 * time.Minute)

	points := make([]TimeSeriesPoint, 20)
	for i := 0; i < 20; i++ {
		points[i] = TimeSeriesPoint{
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Value:     float64(i * 10),
		}
	}

	prediction := predictor.Predict(points)
	if prediction < 200 {
		t.Errorf("Expected prediction >= 200, got %f", prediction)
	}
}

func TestResourcePredictorV2(t *testing.T) {
	ctx := context.Background()
	predictor := NewResourcePredictorV2(ctx)
	defer predictor.Stop()

	predictor.AddModel("test-cpu", ModelLinear, ModelParams{
		WindowSize:     10,
		ForecastHorizon: 60,
	})

	for i := 0; i < 15; i++ {
		predictor.AddDataPoint("test-cpu", TimeSeriesPoint{
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Value:     float64(i * 10),
		})
	}

	predictions, err := predictor.Predict("test-cpu", 30*time.Minute)
	if err != nil {
		t.Errorf("Predict failed: %v", err)
	}

	if len(predictions) == 0 {
		t.Error("No predictions returned")
	}
}

func TestLinearRegressionPredictor(t *testing.T) {
	predictor := &LinearRegressionPredictor{}

	points := make([]TimeSeriesPoint, 10)
	for i := 0; i < 10; i++ {
		points[i] = TimeSeriesPoint{
			Timestamp: time.Now(),
			Value:     float64(i * 2),
		}
	}

	prediction, confidence := predictor.Predict(points)
	if prediction < 18 {
		t.Errorf("Expected prediction >= 18, got %f", prediction)
	}
	if confidence < 0 || confidence > 1 {
		t.Errorf("Confidence out of range: %f", confidence)
	}
}

func TestCostOptimizer(t *testing.T) {
	config := &CostConfig{
		BudgetLimit:    1000,
		BudgetPeriod:   30 * 24 * time.Hour,
		AlertThreshold: 0.8,
	}

	optimizer := NewCostOptimizer(config)
	defer optimizer.Stop()

	optimizer.Start()

	optimizer.RecordCost(CostRecord{
		Timestamp:    time.Now(),
		Cost:        100,
		ResourceType: "compute",
		Provider:    "aws",
		Quantity:    1,
	})

	stats := optimizer.GetStats()
	if stats["total_cost"].(float64) < 100 {
		t.Errorf("Expected total_cost >= 100, got %f", stats["total_cost"])
	}
}

func TestCostOptimizerProjectedCost(t *testing.T) {
	optimizer := NewCostOptimizer(nil)
	defer optimizer.Stop()

	optimizer.Start()

	optimizer.RecordCost(CostRecord{
		Timestamp:    time.Now().Add(-24 * time.Hour),
		Cost:        200,
		ResourceType: "compute",
		Provider:    "aws",
		Quantity:    1,
	})

	optimizer.RecordCost(CostRecord{
		Timestamp:    time.Now(),
		Cost:        200,
		ResourceType: "compute",
		Provider:    "aws",
		Quantity:    1,
	})

	projected := optimizer.CalculateProjectedCost()
	if projected < 200 {
		t.Errorf("Expected projected cost >= 200, got %f", projected)
	}
}

func TestCostRecommendations(t *testing.T) {
	optimizer := NewCostOptimizer(nil)
	defer optimizer.Stop()

	optimizer.Start()

	optimizer.RecordCost(CostRecord{
		Timestamp:    time.Now(),
		Cost:        900,
		ResourceType: "compute",
		Provider:    "aws",
		Quantity:    1,
	})

	optimizer.CalculateProjectedCost()
	optimizer.updateRecommendations()

	recs := optimizer.GetRecommendations()
	if len(recs) == 0 {
		t.Error("Expected recommendations, got none")
	}
}

func TestServiceMeshManager(t *testing.T) {
	mesh := NewServiceMeshManager()
	defer mesh.Stop()

	if err := mesh.Start(); err != nil {
		t.Errorf("Start() failed: %v", err)
	}

	route := &Route{
		Name:    "test-route",
		Service: "test-service",
		Weight:  100,
	}

	if err := mesh.CreateRoute(route); err != nil {
		t.Errorf("CreateRoute failed: %v", err)
	}

	retrieved, err := mesh.GetRoute("test-route")
	if err != nil {
		t.Errorf("GetRoute failed: %v", err)
	}

	if retrieved.Name != "test-route" {
		t.Errorf("Expected route name 'test-route', got '%s'", retrieved.Name)
	}

	stats := mesh.GetStats()
	if stats == nil {
		t.Error("GetStats() returned nil")
	}
}

func TestCanaryDeployment(t *testing.T) {
	mesh := NewServiceMeshManager()
	defer mesh.Stop()

	mesh.Start()

	err := mesh.CreateCanaryDeployment("test-canary", "service-a", "v2", 10.0)
	if err != nil {
		t.Errorf("CreateCanaryDeployment failed: %v", err)
	}

	err = mesh.UpdateCanaryTraffic("service-a-v2", 50.0)
	if err != nil {
		t.Errorf("UpdateCanaryTraffic failed: %v", err)
	}

	err = mesh.PromoteCanary("service-a-v2")
	if err != nil {
		t.Errorf("PromoteCanary failed: %v", err)
	}
}

func TestCircuitBreaker(t *testing.T) {
	mesh := NewServiceMeshManager()
	defer mesh.Stop()

	mesh.Start()

	thresholds := &CircuitThresholds{
		MaxConnections:     100,
		MaxPendingRequests: 100,
		MaxRetries:        3,
		SleepWindow:        30 * time.Second,
		RequestVolume:      100,
		ErrorRate:          0.5,
	}

	err := mesh.RegisterCircuitBreaker("test-cb", thresholds)
	if err != nil {
		t.Errorf("RegisterCircuitBreaker failed: %v", err)
	}

	state, err := mesh.GetCircuitBreakerState("test-cb")
	if err != nil {
		t.Errorf("GetCircuitBreakerState failed: %v", err)
	}

	if state != CircuitClosed {
		t.Errorf("Expected CircuitClosed, got %v", state)
	}
}

func TestLoadBalancingPolicy(t *testing.T) {
	mesh := NewServiceMeshManager()
	defer mesh.Stop()

	mesh.Start()

	policy := LoadBalancingPolicy{
		Name:      "round-robin",
		Algorithm: LB_ROUND_ROBIN,
	}

	err := mesh.SetLoadBalancingPolicy("test-service", policy)
	if err != nil {
		t.Errorf("SetLoadBalancingPolicy failed: %v", err)
	}

	retrieved, err := mesh.GetLoadBalancingPolicy("test-service")
	if err != nil {
		t.Errorf("GetLoadBalancingPolicy failed: %v", err)
	}

	if retrieved.Algorithm != LB_ROUND_ROBIN {
		t.Errorf("Expected LB_ROUND_ROBIN, got %v", retrieved.Algorithm)
	}
}

func TestResourceUsagePredictor(t *testing.T) {
	ctx := context.Background()
	predictor := NewResourceUsagePredictor(ctx)
	defer predictor.Stop()

	predictor.Start()

	usage := &ResourceUsage{
		CPUPercent:    50.0,
		MemoryPercent: 60.0,
		DiskPercent:   30.0,
		NetworkBytes:  1024,
	}

	err := predictor.RecordResourceUsage(usage)
	if err != nil {
		t.Errorf("RecordResourceUsage failed: %v", err)
	}

	forecast, err := predictor.Forecast(10 * time.Minute)
	if err != nil {
		t.Errorf("Forecast failed: %v", err)
	}

	if forecast == nil {
		t.Error("Forecast returned nil")
	}
}

func TestScalingPolicyEngine(t *testing.T) {
	engine := NewScalingPolicyEngine()

	metrics := map[string]float64{
		"cpu": 85.0,
	}

	percent, action := engine.Evaluate(metrics)
	if action != "increase" {
		t.Errorf("Expected action 'increase', got '%s'", action)
	}
	if percent == 0 {
		t.Error("Expected non-zero percent")
	}
}

func BenchmarkResourcePredictor(b *testing.B) {
	predictor := NewResourcePredictor(10 * time.Minute)

	points := make([]TimeSeriesPoint, 100)
	for i := 0; i < 100; i++ {
		points[i] = TimeSeriesPoint{
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Value:     float64(i * 10),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		predictor.Predict(points)
	}
}

func BenchmarkCostCalculation(b *testing.B) {
	optimizer := NewCostOptimizer(nil)

	for i := 0; i < 1000; i++ {
		optimizer.RecordCost(CostRecord{
			Timestamp:    time.Now().Add(time.Duration(i) * time.Hour),
			Cost:        float64(i % 100),
			ResourceType: "compute",
			Provider:    "aws",
			Quantity:    1,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.CalculateProjectedCost()
	}
}
