package api_performance

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAPIOptimizer(t *testing.T) {
	config := &APIOptimizationConfig{
		TargetLatency:    TargetLow,
		EnablePrefetch:   true,
		EnableBatch:      true,
		BatchSize:        10,
		BatchTimeout:     5 * time.Millisecond,
		CacheEnabled:     true,
		CacheTTL:         1 * time.Minute,
		CompressionLevel: 5,
		MaxRetries:       3,
		RetryDelay:       100 * time.Millisecond,
	}

	optimizer := NewAPIOptimizer(config)
	defer optimizer.Stop()

	t.Run("BasicGet", func(t *testing.T) {
		ctx := context.Background()
		key := "test-key-1"

		callback := func() ([]byte, error) {
			return []byte("test-value"), nil
		}

		result, err := optimizer.Get(ctx, key, callback)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if string(result) != "test-value" {
			t.Fatalf("Expected 'test-value', got '%s'", string(result))
		}
	})

	t.Run("CacheHit", func(t *testing.T) {
		ctx := context.Background()
		key := "test-key-2"

		callbackCalled := 0
		callback := func() ([]byte, error) {
			callbackCalled++
			return []byte("test-value-2"), nil
		}

		_, err := optimizer.Get(ctx, key, callback)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		_, err = optimizer.Get(ctx, key, callback)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if callbackCalled != 1 {
			t.Fatalf("Expected callback to be called 1 time, got %d", callbackCalled)
		}
	})

	t.Run("BatchGet", func(t *testing.T) {
		key := "batch-key-1"

		callback := func() ([]byte, error) {
			return []byte("batch-value"), nil
		}

		resultChan := optimizer.BatchGet(context.Background(), key, callback)

		select {
		case result := <-resultChan:
			if string(result) != "batch-value" {
				t.Fatalf("Expected 'batch-value', got '%s'", string(result))
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Batch request timeout")
		}
	})

	t.Run("Prefetch", func(t *testing.T) {
		key := "prefetch-key-1"

		callback := func() ([]byte, error) {
			return []byte("prefetch-value"), nil
		}

		optimizer.Prefetch(key, callback)

		time.Sleep(10 * time.Millisecond)

		stats := optimizer.GetStats()
		if stats["prefetch_hits"].(int64) < 1 {
			t.Fatal("Prefetch should have recorded a hit")
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		stats := optimizer.GetStats()

		if stats["total_requests"] == nil {
			t.Fatal("Expected total_requests in stats")
		}

		if stats["cache_hit_rate"] == nil {
			t.Fatal("Expected cache_hit_rate in stats")
		}
	})
}

func TestQueryCache(t *testing.T) {
	cache := NewAPIQueryCache(100)

	t.Run("SetAndGet", func(t *testing.T) {
		key := "test-key"
		value := []byte("test-value")

		cache.Set(key, value, 1*time.Minute)

		result, ok := cache.Get(key)
		if !ok {
			t.Fatal("Expected cache hit")
		}

		if string(result) != string(value) {
			t.Fatalf("Expected '%s', got '%s'", string(value), string(result))
		}
	})

	t.Run("CacheMiss", func(t *testing.T) {
		result, ok := cache.Get("non-existent-key")
		if ok || result != nil {
			t.Fatal("Expected cache miss")
		}
	})

	t.Run("Eviction", func(t *testing.T) {
		smallCache := NewAPIQueryCache(2)

		smallCache.Set("key1", []byte("value1"), 1*time.Minute)
		smallCache.Set("key2", []byte("value2"), 1*time.Minute)
		smallCache.Set("key3", []byte("value3"), 1*time.Minute)

		_, ok1 := smallCache.Get("key1")
		_, ok2 := smallCache.Get("key2")

		if ok1 && ok2 {
			t.Fatal("At least one key should have been evicted")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		cache.Set("key1", []byte("value1"), 1*time.Minute)
		cache.Set("key2", []byte("value2"), 1*time.Minute)

		cache.Clear()

		_, ok1 := cache.Get("key1")
		_, ok2 := cache.Get("key2")

		if ok1 || ok2 {
			t.Fatal("Cache should be empty after clear")
		}
	})
}

func TestQueryOptimizer(t *testing.T) {
	config := &QueryOptimizerConfig{
		EnableQueryCache:    true,
		EnableStmtCache:     true,
		EnableIndexAnalysis: true,
		MaxQueryCacheSize:   100,
		MaxStmtCacheSize:    50,
		QueryTimeout:        5 * time.Second,
		EnableSlowQueryLog:  true,
		SlowQueryThreshold:  100 * time.Millisecond,
		RetryAttempts:       3,
		RetryDelay:          50 * time.Millisecond,
	}

	optimizer := NewQueryOptimizer(config)

	t.Run("QueryCacheStats", func(t *testing.T) {
		stats := optimizer.GetStats()

		if stats["total_queries"] == nil {
			t.Fatal("Expected total_queries in stats")
		}

		if stats["cache_hits"] == nil {
			t.Fatal("Expected cache_hits in stats")
		}
	})

	t.Run("ClearCaches", func(t *testing.T) {
		optimizer.ClearCaches()

		stats := optimizer.GetStats()
		if stats["query_cache_size"] == nil {
			t.Fatal("Expected query cache stats")
		}
	})
}

func TestMiddlewareChainOptimizer(t *testing.T) {
	config := &ChainOptimizerConfig{
		EnableBatching:      true,
		EnableCaching:        true,
		BatchWindow:         5 * time.Millisecond,
		CacheEnabled:        true,
		CacheTTL:            30 * time.Second,
		MaxConcurrency:      1000,
		EnableEarlyExit:     true,
		EarlyExitThreshold:  10 * time.Millisecond,
		MonitorEnabled:      true,
	}

	optimizer := NewMiddlewareChainOptimizer(config)

	t.Run("AddMiddleware", func(t *testing.T) {
		handler := func(c *gin.Context) {
			c.Next()
		}

		optimizer.AddMiddleware("test-middleware", handler, 1)

		stats := optimizer.GetStats()
		if stats == nil {
			t.Fatal("Expected stats to be available")
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		stats := optimizer.GetStats()

		if stats["total_requests"] == nil {
			t.Fatal("Expected total_requests in stats")
		}
	})

	t.Run("ClearCaches", func(t *testing.T) {
		optimizer.ClearCaches()
	})
}

func TestConcurrencyOptimizer(t *testing.T) {
	config := &ConcurrencyConfig{
		WorkerCount:       10,
		MaxQueueSize:      1000,
		MaxTaskDuration:   30 * time.Second,
		EnableResultCache: true,
		ResultCacheSize:   500,
		ResultCacheTTL:    10 * time.Minute,
		EnableTimeout:     true,
		TaskTimeout:       5 * time.Second,
		EnableRetry:       true,
		MaxRetries:        3,
		RetryDelay:        100 * time.Millisecond,
	}

	optimizer := NewConcurrencyOptimizer(config)
	optimizer.Start()
	defer optimizer.Stop()

	t.Run("ExecuteTask", func(t *testing.T) {
		task := &Task{
			ID: "test-task-1",
			Func: func() (interface{}, error) {
				return "test-result", nil
			},
			Timeout: 5 * time.Second,
		}

		result := optimizer.Execute(task)

		if result.Error != nil {
			t.Fatalf("Expected no error, got %v", result.Error)
		}

		if result.Value != "test-result" {
			t.Fatalf("Expected 'test-result', got '%v'", result.Value)
		}
	})

	t.Run("CacheHit", func(t *testing.T) {
		task := &Task{
			ID: "test-task-2",
			Func: func() (interface{}, error) {
				return "cached-result", nil
			},
			Timeout: 5 * time.Second,
		}

		result1 := optimizer.Execute(task)
		result2 := optimizer.Execute(task)

		if result1.FromCache || !result2.FromCache {
			t.Fatal("Second execution should be from cache")
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		stats := optimizer.GetStats()

		if stats["total_tasks"] == nil {
			t.Fatal("Expected total_tasks in stats")
		}
	})
}

func TestWorkerPool(t *testing.T) {
	config := &ConcurrencyConfig{
		WorkerCount:  5,
		MaxQueueSize: 100,
	}

	pool := NewWorkerPool(config)
	pool.Start()
	defer pool.Stop()

	t.Run("SubmitAndWait", func(t *testing.T) {
		task := &Task{
			ID: "worker-task-1",
			Func: func() (interface{}, error) {
				time.Sleep(5 * time.Millisecond)
				return "worker-result", nil
			},
			Timeout: 5 * time.Second,
		}

		result := pool.SubmitAndWait(task)

		if result.Error != nil {
			t.Fatalf("Expected no error, got %v", result.Error)
		}

		if result.Value != "worker-result" {
			t.Fatalf("Expected 'worker-result', got '%v'", result.Value)
		}
	})
}

func TestMetricsCollector(t *testing.T) {
	config := &MetricsConfig{
		EnableCounters:    true,
		EnableGauges:      true,
		EnableHistograms:  true,
		EnableTimers:      true,
		EnablePercentiles: true,
		FlushInterval:     10 * time.Second,
	}

	collector := NewMetricsCollector(config)
	collector.Start()
	defer collector.Stop()

	t.Run("IncrementCounter", func(t *testing.T) {
		collector.IncrementCounter("test-counter", 5)

		stats := collector.GetStats()
		if stats["counters"] == nil {
			t.Fatal("Expected counters in stats")
		}
	})

	t.Run("SetGauge", func(t *testing.T) {
		collector.SetGauge("test-gauge", 100)

		stats := collector.GetStats()
		if stats["gauges"] == nil {
			t.Fatal("Expected gauges in stats")
		}
	})

	t.Run("RecordHistogram", func(t *testing.T) {
		collector.RecordHistogram("test-histogram", 50.5)

		stats := collector.GetStats()
		if stats["histograms"] == nil {
			t.Fatal("Expected histograms in stats")
		}
	})

	t.Run("RecordTimer", func(t *testing.T) {
		collector.RecordTimer("test-timer", 100*time.Millisecond)

		stats := collector.GetStats()
		if stats["timers"] == nil {
			t.Fatal("Expected timers in stats")
		}
	})

	t.Run("RecordRequest", func(t *testing.T) {
		collector.RecordRequest(true, 50*time.Millisecond)

		snapshot := collector.GetSnapshot()
		if snapshot.TotalRequests == 0 {
			t.Fatal("Expected at least one request")
		}
	})

	t.Run("GetSnapshot", func(t *testing.T) {
		snapshot := collector.GetSnapshot()

		if snapshot.Timestamp.IsZero() {
			t.Fatal("Expected non-zero timestamp")
		}

		if snapshot.TotalRequests == 0 {
			t.Fatal("Expected total requests > 0")
		}
	})
}

func TestSemaphoreLimiter(t *testing.T) {
	sem := NewSemaphoreLimiter(2)

	t.Run("AcquireAndRelease", func(t *testing.T) {
		sem.Acquire()
		if sem.GetUsed() != 1 {
			t.Fatal("Expected 1 used semaphore")
		}

		sem.Release()
		if sem.GetUsed() != 0 {
			t.Fatal("Expected 0 used semaphore")
		}
	})

	t.Run("TryAcquire", func(t *testing.T) {
		if !sem.TryAcquire() {
			t.Fatal("Should be able to acquire")
		}

		sem.Release()
	})
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(10, 10)

	t.Run("Allow", func(t *testing.T) {
		if !limiter.Allow() {
			t.Fatal("Should allow request")
		}
	})

	t.Run("AllowN", func(t *testing.T) {
		if !limiter.AllowN(5) {
			t.Fatal("Should allow 5 requests")
		}
	})

	t.Run("GetTokens", func(t *testing.T) {
		tokens := limiter.GetTokens()
		if tokens < 0 {
			t.Fatal("Tokens should be non-negative")
		}
	})
}

func TestAdaptivePool(t *testing.T) {
	pool := NewAdaptivePool(2, 10)
	pool.Start()
	defer pool.Stop()

	t.Run("SubmitTask", func(t *testing.T) {
		completed := false
		pool.Submit(func() interface{} {
			time.Sleep(10 * time.Millisecond)
			completed = true
			return "adaptive-result"
		})

		time.Sleep(50 * time.Millisecond)

		if !completed {
			t.Fatal("Task should have completed")
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		stats := pool.GetStats()

		if stats["current_workers"] == nil {
			t.Fatal("Expected current_workers in stats")
		}
	})
}

func TestPerformanceProfiler(t *testing.T) {
	profiler := NewPerformanceProfiler(1000)

	t.Run("RecordSample", func(t *testing.T) {
		profiler.Record("test-operation", 100*time.Millisecond, nil)

		samples := profiler.GetSamples()
		if len(samples) == 0 {
			t.Fatal("Expected at least one sample")
		}
	})

	t.Run("GetSlowOperations", func(t *testing.T) {
		profiler.Record("slow-op", 200*time.Millisecond, nil)
		profiler.Record("fast-op", 50*time.Millisecond, nil)

		slowSamples := profiler.GetSlowOperations(150 * time.Millisecond)
		if len(slowSamples) != 1 {
			t.Fatal("Expected exactly one slow operation")
		}
	})

	t.Run("Reset", func(t *testing.T) {
		profiler.Reset()

		samples := profiler.GetSamples()
		if len(samples) != 0 {
			t.Fatal("Expected no samples after reset")
		}
	})
}

func TestHealthChecker(t *testing.T) {
	checker := NewHealthChecker(1 * time.Second)
	checker.Start()
	defer checker.Stop()

	t.Run("RegisterCheck", func(t *testing.T) {
		checker.Register("test-check", &HealthCheck{
			Name: "test-check",
			CheckFunc: func() *HealthCheckResult {
				return &HealthCheckResult{
					Status:  "healthy",
					Message: "OK",
				}
			},
		})
	})

	t.Run("RunChecks", func(t *testing.T) {
		results := checker.RunChecks()

		if results["test-check"] == nil {
			t.Fatal("Expected test-check result")
		}
	})
}

func TestQueryBuilder(t *testing.T) {
	t.Run("BasicQuery", func(t *testing.T) {
		qb := NewQueryBuilder("users")
		query, args := qb.Where("id = ?", 1).Build()

		if query != "SELECT * FROM users WHERE id = ?" {
			t.Fatalf("Unexpected query: %s", query)
		}

		if len(args) != 1 || args[0].(int) != 1 {
			t.Fatal("Unexpected args")
		}
	})

	t.Run("QueryWithOrderBy", func(t *testing.T) {
		qb := NewQueryBuilder("users")
		query, _ := qb.Where("status = ?", "active").OrderBy("created_at", true).Limit(10).Build()

		expectedQuery := "SELECT * FROM users WHERE status = ? ORDER BY created_at DESC LIMIT 10"
		if query != expectedQuery {
			t.Fatalf("Expected query: %s, got: %s", expectedQuery, query)
		}
	})

	t.Run("QueryWithOffset", func(t *testing.T) {
		qb := NewQueryBuilder("users")
		query, _ := qb.Limit(10).Offset(20).Build()

		expectedQuery := "SELECT * FROM users LIMIT 10 OFFSET 20"
		if query != expectedQuery {
			t.Fatalf("Expected query: %s, got: %s", expectedQuery, query)
		}
	})

	t.Run("BuildCount", func(t *testing.T) {
		qb := NewQueryBuilder("users")
		query, _ := qb.Where("status = ?", "active").BuildCount()

		expectedQuery := "SELECT COUNT(*) FROM users WHERE status = ?"
		if query != expectedQuery {
			t.Fatalf("Expected query: %s, got: %s", expectedQuery, query)
		}
	})
}

func BenchmarkAPIOptimizer(b *testing.B) {
	optimizer := NewAPIOptimizer(nil)
	defer optimizer.Stop()

	ctx := context.Background()
	key := "benchmark-key"

	callback := func() ([]byte, error) {
		return []byte("benchmark-value"), nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.Get(ctx, key, callback)
	}
}

func BenchmarkQueryCache(b *testing.B) {
	cache := NewAPIQueryCache(10000)

	key := "benchmark-key"
	value := []byte("benchmark-value")
	cache.Set(key, value, 1*time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(key)
	}
}

func BenchmarkConcurrencyOptimizer(b *testing.B) {
	config := &ConcurrencyConfig{
		WorkerCount: 50,
	}

	optimizer := NewConcurrencyOptimizer(config)
	optimizer.Start()
	defer optimizer.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task := &Task{
			ID: fmt.Sprintf("bench-task-%d", i),
			Func: func() (interface{}, error) {
				return "result", nil
			},
			Timeout: 5 * time.Second,
		}
		optimizer.Execute(task)
	}
}

func BenchmarkMetricsCollector(b *testing.B) {
	collector := NewMetricsCollector(nil)
	collector.Start()
	defer collector.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.IncrementCounter("bench-counter", 1)
		collector.SetGauge("bench-gauge", int64(i))
		collector.RecordHistogram("bench-histogram", float64(i))
		collector.RecordTimer("bench-timer", time.Duration(i)*time.Millisecond)
	}
}

func TestIntegration(t *testing.T) {
	optimizer := NewAPIOptimizer(nil)
	defer optimizer.Stop()

	collector := NewMetricsCollector(nil)
	collector.Start()
	defer collector.Stop()

	ctx := context.Background()

	t.Run("EndToEnd", func(t *testing.T) {
		key := "integration-key"

		callback := func() ([]byte, error) {
			time.Sleep(10 * time.Millisecond)
			return []byte("integration-value"), nil
		}

		start := time.Now()
		result, err := optimizer.Get(ctx, key, callback)
		latency := time.Since(start)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if string(result) != "integration-value" {
			t.Fatalf("Expected 'integration-value', got '%s'", string(result))
		}

		collector.RecordRequest(true, latency)

		optStats := optimizer.GetStats()
		if optStats["total_requests"].(int64) == 0 {
			t.Fatal("Expected total_requests > 0")
		}

		metricsStats := collector.GetStats()
		if metricsStats["total_requests"] == nil {
			t.Fatal("Expected total_requests in metrics")
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	optimizer := NewAPIOptimizer(nil)
	defer optimizer.Stop()

	var wg sync.WaitGroup
	numGoroutines := 100

	t.Run("ConcurrentGet", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				key := fmt.Sprintf("concurrent-key-%d", id)
				callback := func() ([]byte, error) {
					return []byte(fmt.Sprintf("value-%d", id)), nil
				}

				result, err := optimizer.Get(context.Background(), key, callback)
				if err != nil {
					t.Errorf("Error in goroutine %d: %v", id, err)
				}

				expected := fmt.Sprintf("value-%d", id)
				if string(result) != expected {
					t.Errorf("Expected '%s', got '%s'", expected, string(result))
				}
			}(i)
		}

		wg.Wait()
	})
}
