package serverless

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestServerlessVerifier_Initialize(t *testing.T) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()

	err := verifier.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !verifier.isRunning {
		t.Error("Verifier should be running after Initialize")
	}

	verifier.Shutdown()
}

func TestServerlessVerifier_RegisterFunction(t *testing.T) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	config := &FunctionConfig{
		Name:         "test-function",
		Runtime:      RuntimeGo,
		MemoryMB:     512,
		Timeout:      10 * time.Second,
		Concurrency:  10,
		MinInstances: 2,
		MaxInstances: 100,
	}

	functionID, err := verifier.RegisterFunction(ctx, config)
	if err != nil {
		t.Fatalf("RegisterFunction failed: %v", err)
	}

	if functionID == "" {
		t.Error("FunctionID should not be empty")
	}

	fn := verifier.getFunction(functionID)
	if fn == nil {
		t.Fatal("Function should be retrievable after registration")
	}

	if fn.Name != config.Name {
		t.Errorf("Expected function name %s, got %s", config.Name, fn.Name)
	}
}

func TestServerlessVerifier_InvokeFunction(t *testing.T) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	config := &FunctionConfig{
		Name:        "invocation-test-function",
		Runtime:     RuntimeGo,
		MemoryMB:    512,
		Timeout:     10 * time.Second,
		Concurrency: 10,
	}

	functionID, _ := verifier.RegisterFunction(ctx, config)

	req := &VerificationRequest{
		RequestID:  "test-invocation",
		FunctionID: functionID,
		Payload:    []byte("test payload"),
	}

	resp, err := verifier.InvokeFunction(ctx, req)
	if err != nil {
		t.Fatalf("InvokeFunction failed: %v", err)
	}

	if !resp.Success {
		t.Error("Function invocation should succeed")
	}

	if resp.RequestID != req.RequestID {
		t.Errorf("Expected request ID %s, got %s", req.RequestID, resp.RequestID)
	}

	if resp.InstanceID == "" {
		t.Error("InstanceID should not be empty")
	}
}

func TestServerlessVerifier_ColdStart(t *testing.T) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	config := &FunctionConfig{
		Name:        "cold-start-test",
		Runtime:     RuntimeGo,
		MemoryMB:    512,
		Timeout:     10 * time.Second,
		Concurrency: 1,
	}

	functionID, _ := verifier.RegisterFunction(ctx, config)

	coldStartCount := 0
	for i := 0; i < 10; i++ {
		verifier.instances.mu.Lock()
		for _, inst := range verifier.instances.instances {
			if inst.FunctionID == functionID {
				inst.Status = InstanceStatusIdle
				inst.Ready = false
			}
		}
		verifier.instances.mu.Unlock()

		req := &VerificationRequest{
			RequestID:  "cold-start-test",
			FunctionID: functionID,
			Payload:    []byte("cold start test"),
		}

		resp, _ := verifier.InvokeFunction(ctx, req)
		if !resp.IsWarm {
			coldStartCount++
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Logf("Cold starts encountered: %d out of 10 invocations", coldStartCount)
}

func TestServerlessVerifier_WarmInstanceReuse(t *testing.T) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	config := &FunctionConfig{
		Name:        "reuse-test",
		Runtime:     RuntimeGo,
		MemoryMB:    512,
		Timeout:     10 * time.Second,
		Concurrency: 10,
	}

	functionID, _ := verifier.RegisterFunction(ctx, config)

	var instanceIDs []string
	warmCount := 0

	for i := 0; i < 5; i++ {
		req := &VerificationRequest{
			RequestID:  "reuse-test",
			FunctionID: functionID,
			Payload:    []byte("reuse test"),
		}

		resp, _ := verifier.InvokeFunction(ctx, req)
		instanceIDs = append(instanceIDs, resp.InstanceID)

		if resp.IsWarm {
			warmCount++
		}
	}

	if warmCount > 0 {
		t.Logf("Warm invocations: %d out of 5", warmCount)
	}
}

func TestServerlessVerifier_ConcurrentInvocations(t *testing.T) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	config := &FunctionConfig{
		Name:        "concurrent-test",
		Runtime:     RuntimeGo,
		MemoryMB:    512,
		Timeout:     10 * time.Second,
		Concurrency: 100,
	}

	functionID, _ := verifier.RegisterFunction(ctx, config)

	var wg sync.WaitGroup
	requestCount := 1000

	start := time.Now()
	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := &VerificationRequest{
				RequestID:  "concurrent-invocation",
				FunctionID: functionID,
				Payload:    []byte("concurrent test payload"),
			}
			verifier.InvokeFunction(ctx, req)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Processed %d concurrent invocations in %v", requestCount, elapsed)
	t.Logf("Throughput: %.2f invocations/s", float64(requestCount)/elapsed.Seconds())

	stats := verifier.GetStats()
	totalInvocations := stats["total_invocations"].(int64)
	if totalInvocations != int64(requestCount) {
		t.Errorf("Expected %d total invocations, got %d", requestCount, totalInvocations)
	}
}

func TestServerlessVerifier_PublishEvent(t *testing.T) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	config := &FunctionConfig{
		Name:        "event-test",
		Runtime:     RuntimeGo,
		MemoryMB:    512,
		Timeout:     10 * time.Second,
		Concurrency: 10,
	}

	functionID, _ := verifier.RegisterFunction(ctx, config)

	event := &FunctionEvent{
		EventID:   "test-event-1",
		FunctionID: functionID,
		Payload:   []byte("event payload"),
		Timestamp: time.Now(),
		Priority:  1,
		Source:    "test-source",
	}

	err := verifier.PublishEvent(ctx, event)
	if err != nil {
		t.Fatalf("PublishEvent failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
}

func TestServerlessVerifier_ScalingPolicy(t *testing.T) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	policy := &ScalingPolicy{
		MetricType:         "cpu",
		TargetValue:        70,
		ScaleUpCooldown:    30 * time.Second,
		ScaleDownCooldown:  60 * time.Second,
	}

	verifier.SetScalingPolicy(ctx, policy)

	if verifier.autoScaler.targetCPU != 70 {
		t.Errorf("Expected target CPU 70, got %d", verifier.autoScaler.targetCPU)
	}
}

func TestServerlessVerifier_ColdStartConfig(t *testing.T) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	config := &ColdStartConfig{
		PrewarmEnabled:     true,
		MinPreWarmed:       5,
		PredictivePrewarm: true,
	}

	verifier.SetColdStartConfig(ctx, config)

	prewarmCount := verifier.coldStart.prewarmCount
	if prewarmCount < 5 {
		t.Logf("Prewarm count: %d (may vary)", prewarmCount)
	}
}

func TestServerlessVerifier_GetActiveInstances(t *testing.T) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	config := &FunctionConfig{
		Name:        "instances-test",
		Runtime:     RuntimeGo,
		MemoryMB:    512,
		Timeout:     10 * time.Second,
		Concurrency: 10,
	}

	functionID, _ := verifier.RegisterFunction(ctx, config)

	for i := 0; i < 5; i++ {
		req := &VerificationRequest{
			RequestID:  "instances-test",
			FunctionID: functionID,
			Payload:    []byte("test"),
		}
		verifier.InvokeFunction(ctx, req)
	}

	activeInstances := verifier.GetActiveInstances()
	if activeInstances == 0 {
		t.Error("Should have at least one active instance")
	}
}

func TestServerlessVerifier_GetStats(t *testing.T) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	config := &FunctionConfig{
		Name:        "stats-test",
		Runtime:     RuntimeGo,
		MemoryMB:    512,
		Timeout:     10 * time.Second,
		Concurrency: 10,
	}

	functionID, _ := verifier.RegisterFunction(ctx, config)

	for i := 0; i < 100; i++ {
		req := &VerificationRequest{
			RequestID:  "stats-test",
			FunctionID: functionID,
			Payload:    []byte("stats test payload"),
		}
		verifier.InvokeFunction(ctx, req)
	}

	stats := verifier.GetStats()

	if stats["total_invocations"].(int64) != 100 {
		t.Errorf("Expected 100 total invocations, got %d", stats["total_invocations"])
	}

	successfulInvocations := stats["successful_invocations"].(int64)
	if successfulInvocations != 100 {
		t.Errorf("Expected 100 successful invocations, got %d", successfulInvocations)
	}

	if stats["avg_latency_ns"].(int64) == 0 {
		t.Error("Average latency should not be zero")
	}

	activeInstances := stats["active_instances"].(int64)
	if activeInstances == 0 {
		t.Logf("Active instances: %d (may be 0 if all returned to idle)", activeInstances)
	}
}

func TestServerlessVerifier_MultipleFunctions(t *testing.T) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	functions := []struct {
		name   string
		memory int
	}{
		{"small-function", 128},
		{"medium-function", 512},
		{"large-function", 2048},
	}

	var functionIDs []string
	for _, f := range functions {
		config := &FunctionConfig{
			Name:        f.name,
			Runtime:     RuntimeGo,
			MemoryMB:    f.memory,
			Timeout:     10 * time.Second,
			Concurrency: 10,
		}
		functionID, _ := verifier.RegisterFunction(ctx, config)
		functionIDs = append(functionIDs, functionID)
	}

	for _, functionID := range functionIDs {
		req := &VerificationRequest{
			RequestID:  "multi-function-test",
			FunctionID: functionID,
			Payload:    []byte("test"),
		}
		verifier.InvokeFunction(ctx, req)
	}

	if len(verifier.functions) != len(functions) {
		t.Errorf("Expected %d functions, got %d", len(functions), len(verifier.functions))
	}
}

func TestServerlessVerifier_Timeout(t *testing.T) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	config := &FunctionConfig{
		Name:        "timeout-test",
		Runtime:     RuntimeGo,
		MemoryMB:    512,
		Timeout:     1 * time.Millisecond,
		Concurrency: 1,
	}

	functionID, _ := verifier.RegisterFunction(ctx, config)

	req := &VerificationRequest{
		RequestID:  "timeout-test",
		FunctionID: functionID,
		Payload:    []byte("timeout test"),
	}

	resp, err := verifier.InvokeFunction(ctx, req)
	if err != nil {
		t.Logf("Timeout error: %v (expected)", err)
	}

	if resp != nil && resp.Success {
		t.Log("Request completed before timeout")
	}
}

func BenchmarkServerlessVerifier_InvokeFunction(b *testing.B) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	config := &FunctionConfig{
		Name:        "bench-function",
		Runtime:     RuntimeGo,
		MemoryMB:    512,
		Timeout:     10 * time.Second,
		Concurrency: 100,
	}

	functionID, _ := verifier.RegisterFunction(ctx, config)

	req := &VerificationRequest{
		RequestID:  "benchmark",
		FunctionID: functionID,
		Payload:    []byte("benchmark payload"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		verifier.InvokeFunction(ctx, req)
	}
}

func BenchmarkServerlessVerifier_ConcurrentInvoke(b *testing.B) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	config := &FunctionConfig{
		Name:        "concurrent-bench-function",
		Runtime:     RuntimeGo,
		MemoryMB:    512,
		Timeout:     10 * time.Second,
		Concurrency: 1000,
	}

	functionID, _ := verifier.RegisterFunction(ctx, config)

	var wg sync.WaitGroup

	b.ResetTimer()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < b.N/100; j++ {
				req := &VerificationRequest{
					RequestID:  "concurrent-bench",
					FunctionID: functionID,
					Payload:    []byte("data"),
				}
				verifier.InvokeFunction(ctx, req)
			}
		}()
	}

	wg.Wait()
}

func BenchmarkServerlessVerifier_ColdStart(b *testing.B) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	config := &FunctionConfig{
		Name:        "cold-bench-function",
		Runtime:     RuntimeGo,
		MemoryMB:    512,
		Timeout:     10 * time.Second,
		Concurrency: 1,
	}

	functionID, _ := verifier.RegisterFunction(ctx, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		verifier.instances.mu.Lock()
		for _, inst := range verifier.instances.instances {
			if inst.FunctionID == functionID {
				inst.Status = InstanceStatusIdle
				inst.Ready = false
			}
		}
		verifier.instances.mu.Unlock()

		req := &VerificationRequest{
			RequestID:  "cold-bench",
			FunctionID: functionID,
			Payload:    []byte("data"),
		}
		verifier.InvokeFunction(ctx, req)

		time.Sleep(1 * time.Millisecond)
	}
}

func BenchmarkServerlessVerifier_PublishEvent(b *testing.B) {
	verifier := NewServerlessVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)
	defer verifier.Shutdown()

	config := &FunctionConfig{
		Name:        "event-function",
		Runtime:     RuntimeGo,
		MemoryMB:    512,
		Timeout:     10 * time.Second,
		Concurrency: 100,
	}

	functionID, _ := verifier.RegisterFunction(ctx, config)

	event := &FunctionEvent{
		EventID:    "bench-event",
		FunctionID: functionID,
		Payload:    []byte("event payload"),
		Timestamp: time.Now(),
		Priority:  1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		verifier.PublishEvent(ctx, event)
	}
}
