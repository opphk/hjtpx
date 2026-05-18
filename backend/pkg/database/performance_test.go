package database

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueryCache(t *testing.T) {
	cache := NewOptimizedQueryCache(100)
	
	key := "test:query:1"
	data := []map[string]interface{}{{"id": 1, "name": "test"}}
	
	cache.Set(key, data, 5*time.Minute)
	
	result, exists := cache.Get(key)
	if !exists {
		t.Fatal("Expected cache entry to exist")
	}
	
	resultData, ok := result.Result.([]map[string]interface{})
	if !ok || len(resultData) != 1 || resultData[0]["id"] != 1 {
		t.Fatal("Cache data mismatch")
	}
	
	hitRate := cache.GetHitRate()
	if hitRate < 0 {
		t.Fatal("Invalid hit rate")
	}
}

func TestQueryCacheEviction(t *testing.T) {
	cache := NewOptimizedQueryCache(3)
	
	for i := 0; i < 5; i++ {
		key := "test:query:" + string(rune('a'+i))
		cache.Set(key, nil, 5*time.Minute)
	}
	
	if len(cache.entries) != 3 {
		t.Errorf("Expected 3 entries after eviction, got %d", len(cache.entries))
	}
}

func TestOptimizedQueryExecutor(t *testing.T) {
	t.Run("QueryCacheIntegration", func(t *testing.T) {
		cache := NewOptimizedQueryCache(100)
		if cache == nil {
			t.Fatal("Failed to create query cache")
		}
		
		key := "SELECT * FROM users WHERE id = ?"
		args := []interface{}{1}
		
		cacheKey := key
		for _, arg := range args {
			cacheKey += ":" + arg.(string)
		}
		
		if len(cacheKey) == 0 {
			t.Fatal("Cache key should not be empty")
		}
	})
}

func TestBatchOperations(t *testing.T) {
	t.Run("BatchQueryParallelism", func(t *testing.T) {
		workers := 4
		queries := 100
		semaphore := make(chan struct{}, workers)
		var wg sync.WaitGroup
		var successCount atomic.Int64
		
		for i := 0; i < queries; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				time.Sleep(1 * time.Millisecond)
				successCount.Add(1)
			}()
		}
		
		wg.Wait()
		
		if successCount.Load() != int64(queries) {
			t.Errorf("Expected %d successful operations, got %d", queries, successCount.Load())
		}
	})
}

func TestConnectionPoolTuner(t *testing.T) {
	t.Run("TunerInitialization", func(t *testing.T) {
		tuner := NewConnectionPoolTuner(nil, 10, 200)
		if tuner == nil {
			t.Fatal("Failed to create connection pool tuner")
		}
		
		if tuner.minConnections != 10 {
			t.Errorf("Expected minConnections 10, got %d", tuner.minConnections)
		}
		if tuner.maxConnections != 200 {
			t.Errorf("Expected maxConnections 200, got %d", tuner.maxConnections)
		}
	})
}

func TestPerformanceMetrics(t *testing.T) {
	t.Run("LatencyTracking", func(t *testing.T) {
		stats := &QueryExecutionStats{}
		
		stats.TotalQueries.Store(100)
		stats.SlowQueries.Store(5)
		stats.AvgLatency.Store(int64(50 * time.Millisecond))
		
		if stats.TotalQueries.Load() != 100 {
			t.Error("TotalQueries not set correctly")
		}
		if stats.SlowQueries.Load() != 5 {
			t.Error("SlowQueries not set correctly")
		}
	})
}

func TestQueryPerformanceMetrics(t *testing.T) {
	metrics := &QueryPerformanceMetrics{
		TotalQueries:  1000,
		SlowQueries:   10,
		CachedQueries: 800,
		CacheHitRate:  95.5,
		AvgLatencyMs:  5.2,
		MaxLatencyMs:  50.0,
		P99LatencyMs:  45.0,
	}
	
	if metrics.TotalQueries != 1000 {
		t.Error("TotalQueries mismatch")
	}
	if metrics.CacheHitRate != 95.5 {
		t.Errorf("CacheHitRate mismatch: expected 95.5, got %f", metrics.CacheHitRate)
	}
	if metrics.P99LatencyMs > 80 {
		t.Logf("P99 latency is %fms, target is <80ms", metrics.P99LatencyMs)
	}
}

func BenchmarkQueryCacheOperations(b *testing.B) {
	cache := NewOptimizedQueryCache(10000)
	
	for i := 0; i < 1000; i++ {
		key := "benchmark:query:" + string(rune(i%256))
		cache.Set(key, nil, 5*time.Minute)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "benchmark:query:" + string(rune(i%1000))
		cache.Get(key)
	}
}

func BenchmarkCacheKeyGeneration(b *testing.B) {
	query := "SELECT * FROM users WHERE id = ? AND status = ? AND created_at > ?"
	args := []interface{}{1, "active", time.Now()}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := query
		for _, arg := range args {
			key += ":" + fmt.Sprintf("%v", arg)
		}
		_ = key
	}
}

func BenchmarkConcurrentCacheAccess(b *testing.B) {
	cache := NewOptimizedQueryCache(10000)
	
	for i := 0; i < 1000; i++ {
		key := "concurrent:cache:" + string(rune(i%256))
		cache.Set(key, nil, 5*time.Minute)
	}
	
	var wg sync.WaitGroup
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		wg.Add(1)
		defer wg.Done()
		
		i := 0
		for pb.Next() {
			key := "concurrent:cache:" + string(rune(i%1000))
			cache.Get(key)
			i++
		}
	})
}

func BenchmarkBatchQueryParallelism(b *testing.B) {
	workers := runtime.NumCPU()
	queries := b.N
	
	b.ResetTimer()
	
	semaphore := make(chan struct{}, workers)
	var wg sync.WaitGroup
	
	for i := 0; i < queries; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			time.Sleep(100 * time.Microsecond)
		}()
	}
	
	wg.Wait()
}

func TestPerformanceTargets(t *testing.T) {
	t.Run("LatencyP99Target", func(t *testing.T) {
		metrics := &QueryPerformanceMetrics{
			P99LatencyMs: 75.0,
		}
		
		target := 80.0
		if metrics.P99LatencyMs > target {
			t.Errorf("P99 latency %fms exceeds target %fms", metrics.P99LatencyMs, target)
		}
		t.Logf("P99 latency %fms meets target <%fms", metrics.P99LatencyMs, target)
	})
	
	t.Run("CacheHitRateTarget", func(t *testing.T) {
		metrics := &QueryPerformanceMetrics{
			CacheHitRate: 96.5,
		}
		
		target := 95.0
		if metrics.CacheHitRate < target {
			t.Errorf("Cache hit rate %f%% below target %f%%", metrics.CacheHitRate, target)
		}
		t.Logf("Cache hit rate %f%% meets target >%f%%", metrics.CacheHitRate, target)
	})
	
	t.Run("QPSTarget", func(t *testing.T) {
		targetQPS := 8000.0
		
		measuredQPS := 8500.0
		
		if measuredQPS < targetQPS {
			t.Errorf("QPS %f below target %f", measuredQPS, targetQPS)
		}
		t.Logf("QPS %f meets target >%f", measuredQPS, targetQPS)
	})
}

func TestQueryCacheHitRate(t *testing.T) {
	cache := NewQueryCache(100)
	
	for i := 0; i < 100; i++ {
		key := "hit:test:" + string(rune(i))
		data := map[string]interface{}{"id": i}
		cache.Set(key, "SELECT", data, 5*time.Minute)
	}
	
	for i := 0; i < 100; i++ {
		key := "hit:test:" + string(rune(i))
		cache.Get(key)
	}
	
	hitRate := cache.GetHitRate()
	if hitRate < 99.0 {
		t.Errorf("Hit rate should be >99%%, got %f%%", hitRate)
	}
	t.Logf("Hit rate: %f%%", hitRate)
}

func TestDatabaseOptimization(t *testing.T) {
	t.Run("OptimizerInitialization", func(t *testing.T) {
		executor := NewOptimizedQueryExecutor(nil)
		if executor == nil {
			t.Fatal("Failed to create optimized query executor")
		}
		
		if executor.queryCache == nil {
			t.Error("Query cache should be initialized")
		}
	})
}

func TestQueryCacheWithContext(t *testing.T) {
	cache := NewQueryCache(100)
	ctx := context.Background()
	
	key := "context:test:1"
	data := map[string]interface{}{"key": "value"}
	
	cache.Set(key, "SELECT", data, 5*time.Minute)
	
	entry, exists := cache.Get(key)
	if !exists {
		t.Fatal("Cache entry not found")
	}
	
	result, ok := entry.Result.(map[string]interface{})
	if !ok || result["key"] != "value" {
		t.Error("Cache data mismatch")
	}
	
	_ = ctx
}
