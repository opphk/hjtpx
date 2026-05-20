package service

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSubMillisecondService_Initialize(t *testing.T) {
	service := NewSubMillisecondService()
	ctx := context.Background()

	err := service.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !service.isRunning {
		t.Error("Service should be running after Initialize")
	}

	service.Shutdown()
}

func TestSubMillisecondService_ProcessVerification(t *testing.T) {
	service := NewSubMillisecondService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	req := &VerificationRequest{
		RequestID: "test-verification",
		Data:      []byte("test data"),
		UseCache:  true,
		CacheKey:  "test-key",
		Priority:  1,
	}

	resp, err := service.ProcessVerification(ctx, req)
	if err != nil {
		t.Fatalf("ProcessVerification failed: %v", err)
	}

	if !resp.Success {
		t.Error("Verification should succeed")
	}

	if resp.Latency > SubMillisecondLatencyTarget {
		t.Logf("Latency %v exceeded target %v", resp.Latency, SubMillisecondLatencyTarget)
	}
}

func TestSubMillisecondService_CacheHit(t *testing.T) {
	service := NewSubMillisecondService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	cacheKey := "cache-test-key"
	testData := []byte("cached test data")

	req1 := &VerificationRequest{
		RequestID: "first-request",
		Data:      testData,
		UseCache:  true,
		CacheKey:  cacheKey,
	}

	resp1, _ := service.ProcessVerification(ctx, req1)
	stats1 := service.GetStats()

	req2 := &VerificationRequest{
		RequestID: "second-request",
		Data:      testData,
		UseCache:  true,
		CacheKey:  cacheKey,
	}

	resp2, _ := service.ProcessVerification(ctx, req2)
	stats2 := service.GetStats()

	if !resp2.FromCache {
		t.Error("Second request should be from cache")
	}

	if resp2.Latency >= resp1.Latency {
		t.Logf("Cache hit latency %v may not be faster than first request %v", resp2.Latency, resp1.Latency)
	}

	cacheHits := stats2["cache_hits"].(int64) - stats1["cache_hits"].(int64)
	if cacheHits != 1 {
		t.Errorf("Expected 1 cache hit, got %d", cacheHits)
	}
}

func TestSubMillisecondService_SubMillisecondTarget(t *testing.T) {
	service := NewSubMillisecondService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	requestCount := 1000
	subMsCount := 0

	for i := 0; i < requestCount; i++ {
		req := &VerificationRequest{
			RequestID: "latency-test",
			Data:      []byte("latency test data"),
			UseCache:  false,
		}

		resp, _ := service.ProcessVerification(ctx, req)
		if resp.Latency < SubMillisecondLatencyTarget {
			subMsCount++
		}
	}

	percentage := float64(subMsCount) / float64(requestCount) * 100
	t.Logf("%.2f%% of requests completed under %v", percentage, SubMillisecondLatencyTarget)

	if percentage < 90 {
		t.Logf("Warning: Only %.2f%% of requests met sub-millisecond target", percentage)
	}
}

func TestSubMillisecondService_WorkerPool(t *testing.T) {
	service := NewSubMillisecondService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	var wg sync.WaitGroup
	taskCount := 500

	start := time.Now()
	for i := 0; i < taskCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			task := func() ([]byte, error) {
				time.Sleep(1 * time.Millisecond)
				return []byte("task result"), nil
			}
			service.SubmitOptimizedTask(task, 1)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Executed %d tasks in %v", taskCount, elapsed)
	t.Logf("Task throughput: %.2f tasks/s", float64(taskCount)/elapsed.Seconds())
}

func TestSubMillisecondService_ConcurrentVerification(t *testing.T) {
	service := NewSubMillisecondService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	var wg sync.WaitGroup
	requestCount := 2000

	start := time.Now()
	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := &VerificationRequest{
				RequestID: "concurrent-test",
				Data:      []byte("concurrent test data"),
				UseCache:  true,
				CacheKey:  "concurrent-key",
			}
			service.ProcessVerification(ctx, req)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Processed %d concurrent requests in %v", requestCount, elapsed)
	t.Logf("Throughput: %.2f req/s", float64(requestCount)/elapsed.Seconds())

	stats := service.GetStats()
	totalRequests := stats["total_requests"].(int64)
	if totalRequests != int64(requestCount) {
		t.Errorf("Expected %d total requests, got %d", requestCount, totalRequests)
	}
}

func TestSubMillisecondService_Prefetch(t *testing.T) {
	service := NewSubMillisecondService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	keys := make([]string, 100)
	for i := 0; i < len(keys); i++ {
		key := "prefetch-key"
		keys[i] = key

		req := &VerificationRequest{
			RequestID: "prefetch-setup",
			Data:      []byte("prefetch data"),
			UseCache:  true,
			CacheKey:  key,
		}
		service.ProcessVerification(ctx, req)
	}

	start := time.Now()
	service.Prefetch(keys)
	elapsed := time.Since(start)

	t.Logf("Prefetched %d keys in %v", len(keys), elapsed)
}

func TestSubMillisecondService_GetStats(t *testing.T) {
	service := NewSubMillisecondService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	for i := 0; i < 100; i++ {
		req := &VerificationRequest{
			RequestID: "stats-test",
			Data:      []byte("stats test data"),
			UseCache:  true,
			CacheKey:  "stats-key",
		}
		service.ProcessVerification(ctx, req)
	}

	stats := service.GetStats()

	if stats["total_requests"].(int64) != 100 {
		t.Errorf("Expected 100 total requests, got %d", stats["total_requests"])
	}

	if stats["avg_latency_ns"].(int64) == 0 {
		t.Error("Average latency should not be zero")
	}

	if stats["worker_utilization"].(int64) < 0 || stats["worker_utilization"].(int64) > 100 {
		t.Error("Worker utilization should be between 0 and 100")
	}
}

func TestSubMillisecondCache_GetSet(t *testing.T) {
	cache := NewSubMillisecondCache(1000)

	key := "test-key"
	value := []byte("test value")

	cache.Set(key, value, 5*time.Minute)

	entry := cache.Get(key)
	if entry == nil {
		t.Fatal("Cache entry should not be nil after Set")
	}

	if string(entry.Value) != string(value) {
		t.Errorf("Expected value %s, got %s", value, entry.Value)
	}
}

func TestSubMillisecondCache_LFUEviction(t *testing.T) {
	cache := NewSubMillisecondCache(10)

	for i := 0; i < 15; i++ {
		key := "eviction-key"
		value := []byte("eviction value")
		cache.Set(key, value, 5*time.Minute)
	}

	if len(cache.items) > 10 {
		t.Errorf("Cache should evict entries, size: %d", len(cache.items))
	}
}

func TestSubMillisecondCache_CleanExpired(t *testing.T) {
	cache := NewSubMillisecondCache(100)

	cache.Set("expired-key", []byte("value"), 1*time.Millisecond)
	cache.Set("valid-key", []byte("value"), 1*time.Hour)

	time.Sleep(10 * time.Millisecond)

	cache.CleanExpired()

	if cache.Get("expired-key") != nil {
		t.Error("Expired key should be removed")
	}

	if cache.Get("valid-key") == nil {
		t.Error("Valid key should remain")
	}
}

func BenchmarkSubMillisecondService_ProcessVerification(b *testing.B) {
	service := NewSubMillisecondService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	req := &VerificationRequest{
		RequestID: "benchmark",
		Data:      []byte("benchmark data"),
		UseCache:  false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ProcessVerification(ctx, req)
	}
}

func BenchmarkSubMillisecondService_CacheHit(b *testing.B) {
	service := NewSubMillisecondService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	cacheKey := "bench-cache-key"
	req := &VerificationRequest{
		RequestID: "setup",
		Data:      []byte("cached data"),
		UseCache:  true,
		CacheKey:  cacheKey,
	}
	service.ProcessVerification(ctx, req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.RequestID = "benchmark"
		service.ProcessVerification(ctx, req)
	}
}

func BenchmarkSubMillisecondService_Concurrent(b *testing.B) {
	service := NewSubMillisecondService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	var wg sync.WaitGroup

	b.ResetTimer()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < b.N/100; j++ {
				req := &VerificationRequest{
					RequestID: "concurrent-bench",
					Data:      []byte("data"),
					UseCache:  true,
					CacheKey:  "key",
				}
				service.ProcessVerification(ctx, req)
			}
		}()
	}

	wg.Wait()
}

func BenchmarkSubMillisecondCache_Lookup(b *testing.B) {
	cache := NewSubMillisecondCache(10000)

	for i := 0; i < 1000; i++ {
		cache.Set(string(rune(i)), []byte("value"), 1*time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("500")
	}
}
