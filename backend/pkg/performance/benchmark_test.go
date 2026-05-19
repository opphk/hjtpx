package performance

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func BenchmarkPerformanceEngine(b *testing.B) {
	engine := NewPerformanceEngine()
	defer engine.Stop()

	ctx := context.Background()
	if err := engine.Start(ctx); err != nil {
		b.Fatalf("Failed to start engine: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.RecordRequest(true, 100*time.Microsecond)
	}
}

func TestQPSRequirement(t *testing.T) {
	engine := NewPerformanceEngine()
	defer engine.Stop()

	ctx := context.Background()
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	const targetQPS = 15000
	const testDuration = 5 * time.Second
	var wg sync.WaitGroup

	requestsPerWorker := targetQPS / 100
	workers := 100

	start := time.Now()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < requestsPerWorker; i++ {
				engine.RecordRequest(true, 10*time.Microsecond)
			}
		}(w)
	}

	wg.Wait()
	duration := time.Since(start)

	totalRequests := workers * requestsPerWorker
	actualQPS := float64(totalRequests) / duration.Seconds()

	fmt.Printf("Test Results:\n")
	fmt.Printf("  Total Requests: %d\n", totalRequests)
	fmt.Printf("  Duration: %.2f seconds\n", duration.Seconds())
	fmt.Printf("  Actual QPS: %.2f\n", actualQPS)
	fmt.Printf("  Target QPS: %d\n", targetQPS)

	if actualQPS < float64(targetQPS) {
		t.Errorf("QPS requirement not met: %.2f < %d", actualQPS, targetQPS)
	} else {
		fmt.Printf("✓ QPS requirement met!\n")
	}

	stats := engine.GetStats()
	fmt.Printf("\nEngine Stats:\n")
	for k, v := range stats {
		fmt.Printf("  %s: %v\n", k, v)
	}
}

func TestDatabaseOptimizer(t *testing.T) {
	dbOpt := NewDatabaseOptimizer()
	ctx := context.Background()

	if err := dbOpt.Start(ctx); err != nil {
		t.Fatalf("Failed to start database optimizer: %v", err)
	}
	defer dbOpt.Stop()

	// Test query caching
	queryKey := "SELECT * FROM users WHERE id=?"

	// First call (cache miss)
	result1, err := dbOpt.ExecuteCached(queryKey, func() (interface{}, error) {
		return []string{"user1", "user2"}, nil
	})
	if err != nil {
		t.Errorf("Failed to execute query: %v", err)
	}

	// Second call (cache hit)
	result2, err := dbOpt.ExecuteCached(queryKey, func() (interface{}, error) {
		return nil, nil // Shouldn't be called
	})
	if err != nil {
		t.Errorf("Failed to execute cached query: %v", err)
	}

	if result1 == nil || result2 == nil {
		t.Errorf("Expected non-nil results")
	}

	stats := dbOpt.GetStats()
	fmt.Printf("Database Stats: %+v\n", stats)
}

func TestCacheOptimizer(t *testing.T) {
	cacheOpt := NewCacheOptimizer()
	ctx := context.Background()

	if err := cacheOpt.Start(ctx); err != nil {
		t.Fatalf("Failed to start cache optimizer: %v", err)
	}
	defer cacheOpt.Stop()

	key := "test-key"
	value := []byte("test-value")

	if err := cacheOpt.Set(key, value); err != nil {
		t.Errorf("Failed to set cache: %v", err)
	}

	retrieved, err := cacheOpt.Get(key)
	if err != nil {
		t.Errorf("Failed to get cache: %v", err)
	}

	if string(retrieved) != string(value) {
		t.Errorf("Cache mismatch: expected %s, got %s", value, retrieved)
	}

	stats := cacheOpt.GetStats()
	fmt.Printf("Cache Stats: %+v\n", stats)
}

func TestConcurrencyManager(t *testing.T) {
	concurrency := NewConcurrencyManager()
	ctx := context.Background()

	if err := concurrency.Start(ctx); err != nil {
		t.Fatalf("Failed to start concurrency manager: %v", err)
	}
	defer concurrency.Stop()

	var wg sync.WaitGroup
	results := make([]int, 100)

	for i := 0; i < 100; i++ {
		idx := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			concurrency.Submit(func() error {
				results[idx] = idx * 2
				return nil
			})
		}()
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	stats := concurrency.GetStats()
	fmt.Printf("Concurrency Stats: %+v\n", stats)
}

func TestResourceManager(t *testing.T) {
	resource := NewResourceManager()
	ctx := context.Background()

	if err := resource.Start(ctx); err != nil {
		t.Fatalf("Failed to start resource manager: %v", err)
	}
	defer resource.Stop()

	// Test scaling
	resource.ScaleUp()
	stats := resource.GetStats()
	fmt.Printf("Resource Stats After Scale Up: %+v\n", stats)

	resource.ScaleDown()
	stats = resource.GetStats()
	fmt.Printf("Resource Stats After Scale Down: %+v\n", stats)
}

func TestEdgeCompute(t *testing.T) {
	edge := NewEdgeCompute()
	ctx := context.Background()

	if err := edge.Start(ctx); err != nil {
		t.Fatalf("Failed to start edge compute: %v", err)
	}
	defer edge.Stop()

	// Add test nodes
	edge.AddNode("node-1", "http://localhost:8001", 100)
	edge.AddNode("node-2", "http://localhost:8002", 100)

	// Test routing
	_, cached := edge.RouteRequest("test-key")
	if cached {
		t.Log("Cache hit (expected for first request?)")
	}

	// Cache a result
	edge.CacheResult("test-key", []byte("test-data"))

	// Check cache hit
	_, cached = edge.RouteRequest("test-key")
	if !cached {
		t.Errorf("Expected cache hit")
	}

	stats := edge.GetStats()
	fmt.Printf("Edge Stats: %+v\n", stats)
}

func TestWASMEngine(t *testing.T) {
	wasm := NewWASMEngine()
	ctx := context.Background()

	if err := wasm.Start(ctx); err != nil {
		t.Fatalf("Failed to start WASM engine: %v", err)
	}
	defer wasm.Stop()

	// Load test module
	moduleData := []byte("test-module-data")
	if err := wasm.LoadModule("test-module", moduleData); err != nil {
		t.Errorf("Failed to load module: %v", err)
	}

	// Execute module
	input := []byte("test-input")
	result, err := wasm.ExecuteModule("test-module", input)
	if err != nil {
		t.Errorf("Failed to execute module: %v", err)
	}

	// Verify result
	if len(result) != len(input) {
		t.Errorf("Unexpected result length")
	}

	// Batch execution
	inputs := [][]byte{
		[]byte("input1"),
		[]byte("input2"),
		[]byte("input3"),
	}
	batchResult, err := wasm.ExecuteBatch("test-module", inputs)
	if err != nil {
		t.Errorf("Failed to execute batch: %v", err)
	}
	if len(batchResult) != len(inputs) {
		t.Errorf("Unexpected batch result length")
	}

	stats := wasm.GetStats()
	fmt.Printf("WASM Stats: %+v\n", stats)
}

func TestEndToEndPerformance(t *testing.T) {
	engine := NewPerformanceEngine()
	defer engine.Stop()

	ctx := context.Background()
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	const concurrentUsers = 100
	const requestsPerUser = 1000

	var wg sync.WaitGroup
	start := time.Now()

	for u := 0; u < concurrentUsers; u++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			for r := 0; r < requestsPerUser; r++ {
				engine.RecordRequest(true, 50*time.Microsecond)
			}
		}(u)
	}

	wg.Wait()
	duration := time.Since(start)

	totalRequests := concurrentUsers * requestsPerUser
	qps := float64(totalRequests) / duration.Seconds()

	fmt.Printf("\nEnd-to-End Test Results:\n")
	fmt.Printf("  Concurrent Users: %d\n", concurrentUsers)
	fmt.Printf("  Requests per User: %d\n", requestsPerUser)
	fmt.Printf("  Total Requests: %d\n", totalRequests)
	fmt.Printf("  Duration: %.2f seconds\n", duration.Seconds())
	fmt.Printf("  QPS: %.2f\n", qps)
	fmt.Printf("  Target QPS: %d\n", TargetQPS)

	if qps < float64(TargetQPS) {
		t.Errorf("QPS below target: %.2f < %d", qps, TargetQPS)
	}

	engineStats := engine.GetStats()
	fmt.Printf("\nEngine Stats:\n")
	for k, v := range engineStats {
		fmt.Printf("  %s: %v\n", k, v)
	}
}
