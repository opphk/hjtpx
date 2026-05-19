package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerlessManager_RegisterFunction(t *testing.T) {
	manager := NewServerlessManager()

	config := &FunctionConfig{
		FunctionName: "test-function",
		Runtime:     RuntimeGo120,
		Memory:      Memory256MB,
		Timeout:     Timeout30s,
		Handler:     "main.Handle",
	}

	err := manager.RegisterFunction(config)
	require.NoError(t, err)

	function, err := manager.GetFunction("test-function")
	require.NoError(t, err)
	assert.Equal(t, "test-function", function.FunctionName)
	assert.Equal(t, RuntimeGo120, function.Runtime)
	assert.Equal(t, Memory256MB, function.Memory)
}

func TestServerlessManager_RegisterFunction_Duplicate(t *testing.T) {
	manager := NewServerlessManager()

	config := &FunctionConfig{
		FunctionName: "test-function",
		Runtime:      RuntimeGo120,
		Handler:      "main.Handle",
	}

	err := manager.RegisterFunction(config)
	require.NoError(t, err)

	err = manager.RegisterFunction(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestServerlessManager_RegisterFunction_Invalid(t *testing.T) {
	manager := NewServerlessManager()

	tests := []struct {
		name   string
		config *FunctionConfig
		errMsg string
	}{
		{
			name:   "nil config",
			config: nil,
			errMsg: "cannot be nil",
		},
		{
			name: "empty name",
			config: &FunctionConfig{
				Runtime:  RuntimeGo120,
				Handler:  "main.Handle",
			},
			errMsg: "name is required",
		},
		{
			name: "empty runtime",
			config: &FunctionConfig{
				FunctionName: "test",
				Handler:      "main.Handle",
			},
			errMsg: "runtime is required",
		},
		{
			name: "empty handler",
			config: &FunctionConfig{
				FunctionName: "test",
				Runtime:      RuntimeGo120,
			},
			errMsg: "handler is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.RegisterFunction(tt.config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestServerlessManager_GetFunction_NotFound(t *testing.T) {
	manager := NewServerlessManager()

	_, err := manager.GetFunction("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestServerlessManager_ListFunctions(t *testing.T) {
	manager := NewServerlessManager()

	configs := []*FunctionConfig{
		{FunctionName: "func1", Runtime: RuntimeGo120, Handler: "main.Handle"},
		{FunctionName: "func2", Runtime: RuntimeGo120, Handler: "main.Handle"},
		{FunctionName: "func3", Runtime: RuntimeGo120, Handler: "main.Handle"},
	}

	for _, config := range configs {
		err := manager.RegisterFunction(config)
		require.NoError(t, err)
	}

	functions := manager.ListFunctions()
	assert.Len(t, functions, 3)
}

func TestServerlessManager_UpdateFunction(t *testing.T) {
	manager := NewServerlessManager()

	config := &FunctionConfig{
		FunctionName: "test-function",
		Runtime:      RuntimeGo120,
		Memory:       Memory256MB,
		Timeout:      Timeout30s,
		Handler:      "main.Handle",
	}

	err := manager.RegisterFunction(config)
	require.NoError(t, err)

	newConfig := &FunctionConfig{
		FunctionName: "test-function",
		Runtime:      RuntimeGo122,
		Memory:       Memory512MB,
		Timeout:      Timeout60s,
		Handler:      "main.Handle",
	}

	err = manager.UpdateFunction("test-function", newConfig)
	require.NoError(t, err)

	function, err := manager.GetFunction("test-function")
	require.NoError(t, err)
	assert.Equal(t, RuntimeGo122, function.Runtime)
	assert.Equal(t, Memory512MB, function.Memory)
}

func TestServerlessManager_DeleteFunction(t *testing.T) {
	manager := NewServerlessManager()

	config := &FunctionConfig{
		FunctionName: "test-function",
		Runtime:      RuntimeGo120,
		Handler:      "main.Handle",
	}

	err := manager.RegisterFunction(config)
	require.NoError(t, err)

	err = manager.DeleteFunction("test-function")
	require.NoError(t, err)

	_, err = manager.GetFunction("test-function")
	assert.Error(t, err)
}

func TestServerlessManager_SetFunctionState(t *testing.T) {
	manager := NewServerlessManager()

	config := &FunctionConfig{
		FunctionName: "test-function",
		Runtime:      RuntimeGo120,
		Handler:      "main.Handle",
	}

	err := manager.RegisterFunction(config)
	require.NoError(t, err)

	err = manager.SetFunctionState("test-function", FunctionStateRunning)
	require.NoError(t, err)

	state, err := manager.GetFunctionState("test-function")
	require.NoError(t, err)
	assert.Equal(t, FunctionStateRunning, state)
}

func TestServerlessManager_RecordInvocation(t *testing.T) {
	manager := NewServerlessManager()

	config := &FunctionConfig{
		FunctionName: "test-function",
		Runtime:      RuntimeGo120,
		Handler:      "main.Handle",
	}

	err := manager.RegisterFunction(config)
	require.NoError(t, err)

	record := &InvocationRecord{
		Timestamp:     time.Now(),
		Duration:      100 * 1e6,
		MemoryUsed:    50 * 1024 * 1024,
		BilledDuration: 100 * 1e6,
		StatusCode:    200,
		RequestID:     "req-123",
	}

	err = manager.RecordInvocation("test-function", record)
	require.NoError(t, err)

	function, err := manager.GetFunction("test-function")
	require.NoError(t, err)
	assert.Equal(t, int64(1), function.InvokeCount.Load())
}

func TestServerlessManager_GetMetrics(t *testing.T) {
	manager := NewServerlessManager()

	metrics := manager.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "total_functions")
	assert.Contains(t, metrics, "total_invocations")
}

func TestServerlessManager_CalculateCost(t *testing.T) {
	manager := NewServerlessManager()

	config := &FunctionConfig{
		FunctionName: "test-function",
		Runtime:      RuntimeGo120,
		Memory:       Memory256MB,
		Handler:      "main.Handle",
	}

	err := manager.RegisterFunction(config)
	require.NoError(t, err)

	record := &InvocationRecord{
		Timestamp:     time.Now(),
		Duration:      100 * 1e6,
		MemoryUsed:    50 * 1024 * 1024,
		BilledDuration: 100 * 1e6,
		StatusCode:    200,
		RequestID:     "req-123",
	}

	err = manager.RecordInvocation("test-function", record)
	require.NoError(t, err)

	cost, err := manager.CalculateCost("test-function")
	require.NoError(t, err)
	assert.Greater(t, cost, 0.0)
}

func TestServerlessManager_ExportConfig(t *testing.T) {
	manager := NewServerlessManager()

	config := &FunctionConfig{
		FunctionName: "test-function",
		Runtime:      RuntimeGo120,
		Memory:       Memory256MB,
		Timeout:      Timeout30s,
		Handler:      "main.Handle",
	}

	err := manager.RegisterFunction(config)
	require.NoError(t, err)

	configJSON, err := manager.ExportConfig("test-function")
	require.NoError(t, err)
	assert.Contains(t, configJSON, "test-function")
}

func TestFunctionState_String(t *testing.T) {
	tests := []struct {
		state   FunctionState
		visible string
	}{
		{FunctionStatePending, "pending"},
		{FunctionStateDeploying, "deploying"},
		{FunctionStateRunning, "running"},
		{FunctionStateScaling, "scaling"},
		{FunctionStateError, "error"},
		{FunctionStateStopped, "stopped"},
		{FunctionStateUpdating, "updating"},
		{FunctionState(100), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.visible, func(t *testing.T) {
			assert.Equal(t, tt.visible, tt.state.String())
		})
	}
}

func TestGetDefaultRuntime(t *testing.T) {
	runtime := GetDefaultRuntime()
	assert.NotEmpty(t, runtime)
}

func TestColdStartOptimizer_Configure(t *testing.T) {
	manager := NewServerlessManager()
	optimizer := NewColdStartOptimizer(manager)

	config := &OptimizationConfig{
		Strategy:          StrategyPreWarming,
		PreWarmingEnabled: true,
		PreWarmingInterval: 5 * time.Minute,
		PreWarmingCount:   2,
	}

	err := optimizer.Configure("test-function", config)
	require.NoError(t, err)

	optConfig, err := optimizer.GetOptimizationConfig("test-function")
	require.NoError(t, err)
	assert.Equal(t, StrategyPreWarming, optConfig.Strategy)
}

func TestColdStartOptimizer_OptimizeColdStart(t *testing.T) {
	manager := NewServerlessManager()
	optimizer := NewColdStartOptimizer(manager)

	config := &FunctionConfig{
		FunctionName: "test-function",
		Runtime:      RuntimeGo120,
		Handler:      "main.Handle",
	}

	err := manager.RegisterFunction(config)
	require.NoError(t, err)

	result, err := optimizer.OptimizeColdStart("test-function")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-function", result.FunctionName)
}

func TestAutoScaler_CreateScalingPolicy(t *testing.T) {
	manager := NewServerlessManager()
	scaler := NewAutoScaler(manager)

	policy := CreateTargetTrackingPolicy("test-function", "test-policy", MetricCPUUtilization, 70.0)

	err := scaler.CreateScalingPolicy("test-function", policy)
	require.NoError(t, err)

	retrieved, err := scaler.GetScalingPolicy("test-policy")
	require.NoError(t, err)
	assert.Equal(t, "test-function", retrieved.FunctionName)
	assert.Equal(t, 70.0, retrieved.TargetValue)
}

func TestAutoScaler_Scale(t *testing.T) {
	manager := NewServerlessManager()
	scaler := NewAutoScaler(manager)

	config := &FunctionConfig{
		FunctionName: "test-function",
		Runtime:      RuntimeGo120,
		Handler:      "main.Handle",
		MaxInstances: 10,
		MinInstances: 1,
	}

	err := manager.RegisterFunction(config)
	require.NoError(t, err)

	policy := CreateTargetTrackingPolicy("test-function", "test-policy", MetricCPUUtilization, 70.0)
	err = scaler.CreateScalingPolicy("test-function", policy)
	require.NoError(t, err)

	result, err := scaler.Scale(context.Background(), "test-function")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-function", result.FunctionName)
}

func TestCostOptimizer_Configure(t *testing.T) {
	manager := NewServerlessManager()
	optimizer := NewCostOptimizer(manager)

	config := &CostOptimizationConfig{
		PricingModel:       PricingOnDemand,
		MemoryOptimization: true,
		BudgetLimit:        100.0,
		AlertThreshold:     80.0,
	}

	err := optimizer.Configure("test-function", config)
	require.NoError(t, err)
}

func TestCostOptimizer_GetCostAllocation(t *testing.T) {
	manager := NewServerlessManager()
	optimizer := NewCostOptimizer(manager)

	config := &FunctionConfig{
		FunctionName: "test-function",
		Runtime:      RuntimeGo120,
		Memory:       Memory256MB,
		Handler:      "main.Handle",
	}

	err := manager.RegisterFunction(config)
	require.NoError(t, err)

	allocation, err := optimizer.GetCostAllocation("test-function")
	require.NoError(t, err)
	assert.Equal(t, "test-function", allocation.FunctionName)
	assert.GreaterOrEqual(t, allocation.TotalCost, 0.0)
}

func TestCostOptimizer_GenerateReport(t *testing.T) {
	manager := NewServerlessManager()
	optimizer := NewCostOptimizer(manager)

	startDate := time.Now().Add(-24 * time.Hour)
	endDate := time.Now()

	report, err := optimizer.GenerateReport(context.Background(), startDate, endDate)
	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, "report-", report.ReportID[:7])
}

func TestCostOptimizer_SetBudgetAlert(t *testing.T) {
	manager := NewServerlessManager()
	optimizer := NewCostOptimizer(manager)

	err := optimizer.SetBudgetAlert("test-function", 100.0)
	require.NoError(t, err)

	alerts := optimizer.CheckBudgetAlerts()
	assert.NotNil(t, alerts)
}

func TestServerlessRuntime_Initialize(t *testing.T) {
	runtime := NewServerlessRuntime("test-function", RuntimeGo120)

	config := &RuntimeConfig{
		FunctionName: "test-function",
		Runtime:      RuntimeGo120,
		Handler:      "main.Handle",
		Memory:       Memory256MB,
		Timeout:      Timeout30s,
	}

	err := runtime.Initialize(config)
	require.NoError(t, err)
	assert.Equal(t, RuntimeStateReady, runtime.GetState())
}

func TestServerlessRuntime_Invoke(t *testing.T) {
	runtime := NewServerlessRuntime("test-function", RuntimeGo120)

	config := &RuntimeConfig{
		FunctionName: "test-function",
		Runtime:      RuntimeGo120,
		Handler:      "main.Handle",
		Memory:       Memory256MB,
		Timeout:      Timeout30s,
	}

	err := runtime.Initialize(config)
	require.NoError(t, err)

	err = runtime.Start()
	require.NoError(t, err)

	req := &InvocationRequest{
		RequestID:    "req-123",
		FunctionName: "test-function",
		Payload:      []byte(`{"name":"test"}`),
	}

	resp, err := runtime.Invoke(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	runtime.Stop()
}

func TestServerlessRuntime_GetMetrics(t *testing.T) {
	runtime := NewServerlessRuntime("test-function", RuntimeGo120)

	config := &RuntimeConfig{
		FunctionName: "test-function",
		Runtime:      RuntimeGo120,
		Handler:      "main.Handle",
		Memory:       Memory256MB,
		Timeout:      Timeout30s,
	}

	err := runtime.Initialize(config)
	require.NoError(t, err)

	metrics := runtime.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "total_invocations")
}

func TestTriggerManager_CreateTrigger(t *testing.T) {
	manager := NewTriggerManager()

	config := &HTTPTriggerConfig{
		Path:   "/api/test",
		Method: []string{"GET", "POST"},
	}

	trigger, err := manager.CreateTrigger("test-function", "test-trigger", TriggerHTTP, config)
	require.NoError(t, err)
	assert.Equal(t, "test-function", trigger.FunctionName)
	assert.Equal(t, TriggerHTTP, trigger.TriggerType)
}

func TestTriggerManager_EnableTrigger(t *testing.T) {
	manager := NewTriggerManager()

	config := &HTTPTriggerConfig{
		Path:   "/api/test",
		Method: []string{"GET"},
	}

	trigger, err := manager.CreateTrigger("test-function", "test-trigger", TriggerHTTP, config)
	require.NoError(t, err)

	err = manager.EnableTrigger(trigger.TriggerID)
	require.NoError(t, err)

	updated, err := manager.GetTrigger(trigger.TriggerID)
	require.NoError(t, err)
	assert.Equal(t, TriggerStateActive, updated.State)
}

func TestTriggerManager_ValidateConfig(t *testing.T) {
	manager := NewTriggerManager()

	tests := []struct {
		name        string
		triggerType TriggerType
		config      interface{}
		expectErr   bool
	}{
		{
			name:        "valid http config",
			triggerType: TriggerHTTP,
			config: &HTTPTriggerConfig{
				Path:   "/api/test",
				Method: []string{"GET"},
			},
			expectErr: false,
		},
		{
			name:        "http config missing path",
			triggerType: TriggerHTTP,
			config: &HTTPTriggerConfig{
				Path:   "",
				Method: []string{"GET"},
			},
			expectErr: true,
		},
		{
			name:        "valid timer config",
			triggerType: TriggerTimer,
			config: &TimerTriggerConfig{
				Expression:  "0 */5 * * * *",
				CronEnabled:  true,
			},
			expectErr: false,
		},
		{
			name:        "timer config missing expression",
			triggerType: TriggerTimer,
			config: &TimerTriggerConfig{
				Expression: "",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ValidateConfig(tt.triggerType, tt.config)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFunctionDeployer_Deploy(t *testing.T) {
	manager := NewServerlessManager()
	deployer := NewFunctionDeployer(manager)

	config := &FunctionConfig{
		FunctionName: "test-function",
		Runtime:      RuntimeGo120,
		Handler:      "main.Handle",
		Memory:       Memory256MB,
		Timeout:      Timeout30s,
	}

	err := manager.RegisterFunction(config)
	require.NoError(t, err)

	sourceCode := []byte(`package main

func main() {
    println("Hello, World!")
}
`)

	result, err := deployer.Deploy(context.Background(), "test-function", sourceCode, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-function", result.FunctionName)
}

func TestFunctionDeployer_CreateDeploymentPackage(t *testing.T) {
	manager := NewServerlessManager()
	deployer := NewFunctionDeployer(manager)

	sourceCode := []byte(`package main

func main() {
    println("Hello, World!")
}
`)

	pkg, err := deployer.CreateDeploymentPackage("test-function", RuntimeGo120, sourceCode)
	require.NoError(t, err)
	assert.NotEmpty(t, pkg)
}

func TestCreateHTTPTrigger(t *testing.T) {
	trigger, err := CreateHTTPTrigger("test-function", "test-trigger", "/api/test", []string{"GET", "POST"})
	require.NoError(t, err)
	assert.Equal(t, TriggerHTTP, trigger.TriggerType)
}

func TestCreateTimerTrigger(t *testing.T) {
	trigger, err := CreateTimerTrigger("test-function", "test-trigger", "0 */5 * * * *")
	require.NoError(t, err)
	assert.Equal(t, TriggerTimer, trigger.TriggerType)
}

func TestCreateQueueTrigger(t *testing.T) {
	trigger, err := CreateQueueTrigger("test-function", "test-trigger", "my-queue", 10)
	require.NoError(t, err)
	assert.Equal(t, TriggerQueue, trigger.TriggerType)
}

func TestEstimateInitTime(t *testing.T) {
	duration := EstimateInitTime(1*1024*1024, 10)
	assert.Greater(t, duration, 0*time.Millisecond)
	assert.LessOrEqual(t, duration, 500*time.Millisecond)
}

func TestCreateSnapshot(t *testing.T) {
	snapshot := CreateSnapshot()
	assert.NotNil(t, snapshot)
	assert.Equal(t, int64(0), snapshot.AvgColdStartTime)
}

func TestAnalyzeDependencyGraph(t *testing.T) {
	imports := []string{"fmt", "os", "time"}
	graph := AnalyzeDependencyGraph(imports)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Nodes, 3)
}

func TestCreateTargetTrackingPolicy(t *testing.T) {
	policy := CreateTargetTrackingPolicy("test-function", "test-policy", MetricCPUUtilization, 70.0)
	assert.Equal(t, PolicyTypeTargetTracking, policy.PolicyType)
	assert.Equal(t, MetricCPUUtilization, policy.Metric)
	assert.Equal(t, 70.0, policy.TargetValue)
}

func TestCreateStepScalingPolicy(t *testing.T) {
	adjustments := []StepAdjustment{
		{LowerBound: 0, UpperBound: 100, Adjustment: 1},
		{LowerBound: 100, UpperBound: 200, Adjustment: 2},
	}

	policy := CreateStepScalingPolicy("test-function", "test-policy", MetricRequestCount, adjustments)
	assert.Equal(t, PolicyTypeStepScaling, policy.PolicyType)
	assert.Len(t, policy.StepAdjustments, 2)
}

func TestCreateScheduledScalingPolicy(t *testing.T) {
	policy := CreateScheduledScalingPolicy("test-function", "test-policy", "0 9 * * *", 2, 10)
	assert.Equal(t, PolicyTypeScheduled, policy.PolicyType)
	assert.Equal(t, 2, policy.ScheduledConfig.MinCapacity)
	assert.Equal(t, 10, policy.ScheduledConfig.MaxCapacity)
}

func TestCalculateBilledDuration(t *testing.T) {
	tests := []struct {
		latency  time.Duration
		expected time.Duration
	}{
		{50 * time.Millisecond, 100 * time.Millisecond},
		{150 * time.Millisecond, 200 * time.Millisecond},
		{300 * time.Millisecond, 300 * time.Millisecond},
	}

	for _, tt := range tests {
		result := calculateBilledDuration(tt.latency)
		assert.Equal(t, tt.expected, result)
	}
}

func TestInstancePool(t *testing.T) {
	pool := NewInstancePool("test-function", 2, 5)

	stats := pool.GetStats()
	assert.Equal(t, 2, int(stats["min_instances"].(int)))
	assert.Equal(t, 5, int(stats["max_instances"].(int)))
	assert.Equal(t, 2, int(stats["current_instances"].(int)))

	instance, err := pool.Acquire(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, instance)

	pool.Release(instance)

	stats = pool.GetStats()
	assert.Equal(t, 2, int(stats["available_instances"].(int)))
}

func TestColdStartSnapshot(t *testing.T) {
	snapshot := &ColdStartSnapshot{
		AvgColdStartTime: 100 * 1e6,
		MaxColdStartTime: 200 * 1e6,
		MinColdStartTime: 50 * 1e6,
		ColdStartCount:   10,
		WarmStartCount:   100,
		Timestamp:        time.Now(),
	}

	assert.Equal(t, int64(100*1e6), snapshot.AvgColdStartTime)
	assert.Equal(t, int64(200*1e6), snapshot.MaxColdStartTime)
	assert.Equal(t, int64(50*1e6), snapshot.MinColdStartTime)
}

func BenchmarkServerlessManager_RegisterFunction(b *testing.B) {
	manager := NewServerlessManager()

	config := &FunctionConfig{
		FunctionName: "benchmark-function",
		Runtime:      RuntimeGo120,
		Handler:      "main.Handle",
		Memory:       Memory256MB,
		Timeout:      Timeout30s,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.FunctionName = fmt.Sprintf("bench-func-%d", i)
		manager.RegisterFunction(config)
	}
}

func BenchmarkColdStartOptimizer_OptimizeColdStart(b *testing.B) {
	manager := NewServerlessManager()
	optimizer := NewColdStartOptimizer(manager)

	config := &FunctionConfig{
		FunctionName: "benchmark-function",
		Runtime:      RuntimeGo120,
		Handler:      "main.Handle",
	}
	manager.RegisterFunction(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.OptimizeColdStart("benchmark-function")
	}
}
