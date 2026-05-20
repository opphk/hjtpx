package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewAdaptiveWorkerPool(t *testing.T) {
	pool := NewAdaptiveWorkerPool(2, 4, 100)
	if pool == nil {
		t.Error("NewAdaptiveWorkerPool returned nil")
	}
	if pool.minWorkers != 2 {
		t.Errorf("minWorkers should be 2, got %d", pool.minWorkers)
	}
	if pool.maxWorkers != 4 {
		t.Errorf("maxWorkers should be 4, got %d", pool.maxWorkers)
	}
	if pool.queueSize != 100 {
		t.Errorf("queueSize should be 100, got %d", pool.queueSize)
	}
}

func TestNewAdaptiveWorkerPool_DefaultValues(t *testing.T) {
	pool := NewAdaptiveWorkerPool(0, 0, 0)
	if pool.minWorkers <= 0 {
		t.Error("minWorkers should have default value")
	}
	if pool.maxWorkers <= 0 {
		t.Error("maxWorkers should have default value")
	}
	if pool.queueSize <= 0 {
		t.Error("queueSize should have default value")
	}
	if pool.maxWorkers < pool.minWorkers {
		t.Error("maxWorkers should be >= minWorkers with defaults")
	}
}

func TestAdaptiveWorkerPool_StartStop(t *testing.T) {
	pool := NewAdaptiveWorkerPool(2, 4, 100)

	pool.Start()
	if !pool.running.Load() {
		t.Error("Pool should be running after Start()")
	}

	pool.Start()

	pool.Stop()
	if pool.running.Load() {
		t.Error("Pool should not be running after Stop()")
	}
}

func TestAdaptiveWorkerPool_Submit(t *testing.T) {
	pool := NewAdaptiveWorkerPool(2, 4, 100)
	pool.Start()
	defer pool.Stop()

	var counter int64
	for i := 0; i < 10; i++ {
		success := pool.Submit(func() error {
			atomic.AddInt64(&counter, 1)
			return nil
		})
		if !success {
			t.Errorf("Submit failed at iteration %d", i)
		}
	}

	time.Sleep(500 * time.Millisecond)

	if counter != 10 {
		t.Errorf("Expected 10 tasks completed, got %d", counter)
	}
}

func TestAdaptiveWorkerPool_SubmitToStoppedPool(t *testing.T) {
	pool := NewAdaptiveWorkerPool(2, 4, 100)

	success := pool.Submit(func() error {
		return nil
	})
	if success {
		t.Error("Submit to stopped pool should return false")
	}
}

func TestAdaptiveWorkerPool_SubmitAndWait(t *testing.T) {
	pool := NewAdaptiveWorkerPool(2, 4, 100)
	pool.Start()
	defer pool.Stop()

	err := pool.SubmitAndWait(func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Errorf("SubmitAndWait failed: %v", err)
	}
}

func TestAdaptiveWorkerPool_SubmitAndWaitToStoppedPool(t *testing.T) {
	pool := NewAdaptiveWorkerPool(2, 4, 100)

	err := pool.SubmitAndWait(func() error {
		return nil
	})
	if err == nil {
		t.Error("SubmitAndWait to stopped pool should return error")
	}
}

func TestAdaptiveWorkerPool_GetMetrics(t *testing.T) {
	pool := NewAdaptiveWorkerPool(2, 4, 100)
	pool.Start()
	defer pool.Stop()

	for i := 0; i < 5; i++ {
		pool.Submit(func() error {
			return nil
		})
	}

	time.Sleep(300 * time.Millisecond)

	metrics := pool.GetMetrics()
	if metrics.TasksSubmitted < 5 {
		t.Errorf("Expected at least 5 tasks submitted, got %d", metrics.TasksSubmitted)
	}
	if metrics.WorkerCount <= 0 {
		t.Error("WorkerCount should be greater than 0")
	}
}

func TestAdaptiveWorkerPool_SetWorkerCount(t *testing.T) {
	pool := NewAdaptiveWorkerPool(2, 8, 100)
	pool.Start()
	defer pool.Stop()

	initialWorkers := pool.workers

	pool.SetWorkerCount(4)
	if pool.workers != 4 {
		t.Errorf("Worker count should be 4, got %d", pool.workers)
	}

	pool.SetWorkerCount(1)
	if pool.workers != initialWorkers {
		t.Error("Worker count should not go below minWorkers")
	}

	pool.SetWorkerCount(100)
	if pool.workers != 8 {
		t.Error("Worker count should not exceed maxWorkers")
	}
}

func TestAdaptiveWorkerPool_ConcurrentSubmit(t *testing.T) {
	pool := NewAdaptiveWorkerPool(4, 8, 100)
	pool.Start()
	defer pool.Stop()

	var wg sync.WaitGroup
	var successCount int64

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if pool.Submit(func() error {
				time.Sleep(10 * time.Millisecond)
				return nil
			}) {
				atomic.AddInt64(&successCount, 1)
			}
		}()
	}

	wg.Wait()

	if successCount == 0 {
		t.Error("At least some submissions should succeed")
	}
}

func TestConcurrencyLimiter_NewConcurrencyLimiter(t *testing.T) {
	limiter := NewConcurrencyLimiter(5)
	if limiter == nil {
		t.Error("NewConcurrencyLimiter returned nil")
	}
	if limiter.maxParallel != 5 {
		t.Errorf("maxParallel should be 5, got %d", limiter.maxParallel)
	}
}

func TestConcurrencyLimiter_DefaultMaxParallel(t *testing.T) {
	limiter := NewConcurrencyLimiter(0)
	if limiter.maxParallel <= 0 {
		t.Error("maxParallel should have default value")
	}
}

func TestConcurrencyLimiter_AcquireRelease(t *testing.T) {
	limiter := NewConcurrencyLimiter(2)

	ctx := context.Background()

	err := limiter.Acquire(ctx)
	if err != nil {
		t.Errorf("First Acquire failed: %v", err)
	}

	err = limiter.Acquire(ctx)
	if err != nil {
		t.Errorf("Second Acquire failed: %v", err)
	}

	limiter.Release()
	limiter.Release()

	current, waiting, max := limiter.GetStats()
	if current != 0 {
		t.Errorf("Current should be 0 after releases, got %d", current)
	}
	if max != 2 {
		t.Errorf("Max should be 2, got %d", max)
	}
}

func TestConcurrencyLimiter_AcquireWithTimeout(t *testing.T) {
	limiter := NewConcurrencyLimiter(1)

	ctx := context.Background()
	err := limiter.Acquire(ctx)
	if err != nil {
		t.Errorf("First Acquire failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	err = limiter.Acquire(ctx)
	if err == nil {
		t.Error("Acquire should timeout when limit reached")
	}

	limiter.Release()
}

func TestConcurrencyLimiter_AcquireContextCanceled(t *testing.T) {
	limiter := NewConcurrencyLimiter(1)

	ctx, cancel := context.WithCancel(context.Background())
	err := limiter.Acquire(ctx)
	if err != nil {
		t.Errorf("First Acquire failed: %v", err)
	}

	cancel()

	err = limiter.Acquire(ctx)
	if err == nil {
		t.Error("Acquire should fail when context is canceled")
	}
}

func TestConcurrencyLimiter_GetStats(t *testing.T) {
	limiter := NewConcurrencyLimiter(3)

	limiter.Acquire(context.Background())
	limiter.Acquire(context.Background())

	current, waiting, max := limiter.GetStats()
	if current != 2 {
		t.Errorf("Current should be 2, got %d", current)
	}
	if max != 3 {
		t.Errorf("Max should be 3, got %d", max)
	}
}

func TestSemaphorePool_NewSemaphorePool(t *testing.T) {
	pool := NewSemaphorePool(10)
	if pool == nil {
		t.Error("NewSemaphorePool returned nil")
	}
	if cap(pool.sem) != 10 {
		t.Errorf("Semaphore capacity should be 10, got %d", cap(pool.sem))
	}
}

func TestSemaphorePool_DefaultSize(t *testing.T) {
	pool := NewSemaphorePool(0)
	if cap(pool.sem) <= 0 {
		t.Error("Semaphore should have default size")
	}
}

func TestSemaphorePool_AcquireRelease(t *testing.T) {
	pool := NewSemaphorePool(2)

	ctx := context.Background()

	err := pool.Acquire(ctx, "key1")
	if err != nil {
		t.Errorf("Acquire failed: %v", err)
	}

	pool.Release("key1")

	stats := pool.GetStats("key1")
	if stats == nil {
		t.Error("GetStats should return stats for acquired key")
	}
	if stats.AcquiredCount != 1 {
		t.Errorf("AcquiredCount should be 1, got %d", stats.AcquiredCount)
	}
	if stats.ReleasedCount != 1 {
		t.Errorf("ReleasedCount should be 1, got %d", stats.ReleasedCount)
	}
}

func TestSemaphorePool_AcquireContextCanceled(t *testing.T) {
	pool := NewSemaphorePool(1)

	ctx := context.Background()
	pool.Acquire(ctx, "key1")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := pool.Acquire(ctx, "key1")
	if err == nil {
		t.Error("Acquire should fail when context is canceled")
	}
}

func TestSemaphorePool_MultipleKeys(t *testing.T) {
	pool := NewSemaphorePool(10)

	ctx := context.Background()

	pool.Acquire(ctx, "key1")
	pool.Acquire(ctx, "key2")
	pool.Acquire(ctx, "key3")

	pool.Release("key1")
	pool.Release("key2")
	pool.Release("key3")

	stats1 := pool.GetStats("key1")
	stats2 := pool.GetStats("key2")

	if stats1 == nil || stats2 == nil {
		t.Error("Should have stats for all keys")
	}
}

func TestRateLimitedExecutor_NewRateLimitedExecutor(t *testing.T) {
	executor := NewRateLimitedExecutor(5, 100, 10*time.Second)
	if executor == nil {
		t.Error("NewRateLimitedExecutor returned nil")
	}
	if executor.limiter == nil {
		t.Error("limiter should be initialized")
	}
	if executor.rateLimiter == nil {
		t.Error("rateLimiter should be initialized")
	}
}

func TestRateLimitedExecutor_Execute(t *testing.T) {
	executor := NewRateLimitedExecutor(2, 100, 10*time.Second)

	ctx := context.Background()

	err := executor.Execute(ctx, func() error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
}

func TestRateLimitedExecutor_ExecuteBatch(t *testing.T) {
	executor := NewRateLimitedExecutor(4, 100, 10*time.Second)

	ctx := context.Background()
	tasks := make([]func() error, 10)
	for i := range tasks {
		tasks[i] = func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		}
	}

	results := executor.ExecuteBatch(ctx, tasks)
	for i, err := range results {
		if err != nil {
			t.Errorf("Task %d failed: %v", i, err)
		}
	}
}

func TestRateLimitedExecutor_ExecuteBatchEmpty(t *testing.T) {
	executor := NewRateLimitedExecutor(2, 100, 10*time.Second)

	ctx := context.Background()
	results := executor.ExecuteBatch(ctx, []func() error{})

	if len(results) != 0 {
		t.Errorf("Empty batch should return empty results, got %d", len(results))
	}
}

func TestAdaptiveBatchProcessor_Process(t *testing.T) {
	processor := NewAdaptiveBatchProcessor[int](10, 2)

	items := []int{1, 2, 3, 4, 5}
	ctx := context.Background()

	results := processor.Process(ctx, items, func(ctx context.Context, item int) error {
		return nil
	})

	if len(results) != len(items) {
		t.Errorf("Results length mismatch: expected %d, got %d", len(items), len(results))
	}
}

func TestAdaptiveBatchProcessor_ProcessEmpty(t *testing.T) {
	processor := NewAdaptiveBatchProcessor[int](10, 2)

	ctx := context.Background()
	results := processor.Process(ctx, []int{}, func(ctx context.Context, item int) error {
		return nil
	})

	if len(results) != 0 {
		t.Errorf("Empty items should return empty results, got %d", len(results))
	}
}

func TestAdaptiveBatchProcessor_ProcessWithErrors(t *testing.T) {
	processor := NewAdaptiveBatchProcessor[int](10, 2)

	items := []int{1, 2, 3, 4, 5}
	ctx := context.Background()

	results := processor.Process(ctx, items, func(ctx context.Context, item int) error {
		if item == 3 {
			return context.DeadlineExceeded
		}
		return nil
	})

	errorCount := 0
	for _, err := range results {
		if err != nil {
			errorCount++
		}
	}

	if errorCount != 1 {
		t.Errorf("Expected 1 error, got %d", errorCount)
	}
}

func TestAdaptiveBatchProcessor_SetBatchSize(t *testing.T) {
	processor := NewAdaptiveBatchProcessor[int](10, 2)

	processor.SetBatchSize(50)
	if processor.batchSize != 50 {
		t.Errorf("BatchSize should be 50, got %d", processor.batchSize)
	}

	processor.SetBatchSize(0)
	if processor.batchSize != 50 {
		t.Error("SetBatchSize with 0 should not change value")
	}
}

func TestAdaptiveBatchProcessor_SetWorkers(t *testing.T) {
	processor := NewAdaptiveBatchProcessor[int](10, 2)

	processor.SetWorkers(8)
	if processor.workers != 8 {
		t.Errorf("Workers should be 8, got %d", processor.workers)
	}

	processor.SetWorkers(0)
	if processor.workers != 8 {
		t.Error("SetWorkers with 0 should not change value")
	}
}

func TestAdaptiveBatchProcessor_EnableParallel(t *testing.T) {
	processor := NewAdaptiveBatchProcessor[int](10, 2)

	processor.EnableParallel(false)
	if processor.enableParallel {
		t.Error("enableParallel should be false")
	}

	processor.EnableParallel(true)
	if !processor.enableParallel {
		t.Error("enableParallel should be true")
	}
}

func TestAdaptiveBatchProcessor_GetMetrics(t *testing.T) {
	processor := NewAdaptiveBatchProcessor[int](10, 2)

	items := []int{1, 2, 3, 4, 5}
	ctx := context.Background()
	processor.Process(ctx, items, func(ctx context.Context, item int) error {
		return nil
	})

	metrics := processor.GetMetrics()
	if metrics == nil {
		t.Error("GetMetrics should return metrics")
	}
}

func TestPriorityTaskExecutor_NewPriorityTaskExecutor(t *testing.T) {
	executor := NewPriorityTaskExecutor(2, 4, 1)
	if executor == nil {
		t.Error("NewPriorityTaskExecutor returned nil")
	}
	if executor.highWorkers != 2 {
		t.Errorf("highWorkers should be 2, got %d", executor.highWorkers)
	}
	if executor.normalWorkers != 4 {
		t.Errorf("normalWorkers should be 4, got %d", executor.normalWorkers)
	}
	if executor.lowWorkers != 1 {
		t.Errorf("lowWorkers should be 1, got %d", executor.lowWorkers)
	}
}

func TestPriorityTaskExecutor_DefaultWorkers(t *testing.T) {
	executor := NewPriorityTaskExecutor(0, 0, 0)
	if executor.highWorkers != 2 {
		t.Errorf("Default highWorkers should be 2, got %d", executor.highWorkers)
	}
	if executor.normalWorkers <= 0 {
		t.Error("Default normalWorkers should be positive")
	}
	if executor.lowWorkers != 1 {
		t.Errorf("Default lowWorkers should be 1, got %d", executor.lowWorkers)
	}
}

func TestPriorityTaskExecutor_StartStop(t *testing.T) {
	executor := NewPriorityTaskExecutor(1, 1, 1)

	executor.Start()
	if !executor.running.Load() {
		t.Error("Executor should be running after Start()")
	}

	executor.Start()

	executor.Stop()
	if executor.running.Load() {
		t.Error("Executor should not be running after Stop()")
	}
}

func TestPriorityTaskExecutor_SubmitHigh(t *testing.T) {
	executor := NewPriorityTaskExecutor(1, 1, 1)
	executor.Start()
	defer executor.Stop()

	success := executor.SubmitHigh(func() error {
		return nil
	})
	if !success {
		t.Error("SubmitHigh should succeed")
	}
}

func TestPriorityTaskExecutor_SubmitNormal(t *testing.T) {
	executor := NewPriorityTaskExecutor(1, 1, 1)
	executor.Start()
	defer executor.Stop()

	success := executor.SubmitNormal(func() error {
		return nil
	})
	if !success {
		t.Error("SubmitNormal should succeed")
	}
}

func TestPriorityTaskExecutor_SubmitLow(t *testing.T) {
	executor := NewPriorityTaskExecutor(1, 1, 1)
	executor.Start()
	defer executor.Stop()

	success := executor.SubmitLow(func() error {
		return nil
	})
	if !success {
		t.Error("SubmitLow should succeed")
	}
}

func TestPriorityTaskExecutor_SubmitToStoppedExecutor(t *testing.T) {
	executor := NewPriorityTaskExecutor(1, 1, 1)

	if executor.SubmitHigh(func() error { return nil }) {
		t.Error("SubmitHigh to stopped executor should return false")
	}
	if executor.SubmitNormal(func() error { return nil }) {
		t.Error("SubmitNormal to stopped executor should return false")
	}
	if executor.SubmitLow(func() error { return nil }) {
		t.Error("SubmitLow to stopped executor should return false")
	}
}

func TestAdaptiveBatchProcessor_CreateBatches(t *testing.T) {
	processor := NewAdaptiveBatchProcessor[int](3, 2)

	items := []int{1, 2, 3, 4, 5, 6, 7}
	batches := processor.createBatches(items)

	if len(batches) != 3 {
		t.Errorf("Expected 3 batches, got %d", len(batches))
	}

	if len(batches[0]) != 3 {
		t.Errorf("First batch should have 3 items, got %d", len(batches[0]))
	}
	if len(batches[1]) != 3 {
		t.Errorf("Second batch should have 3 items, got %d", len(batches[1]))
	}
	if len(batches[2]) != 1 {
		t.Errorf("Third batch should have 1 item, got %d", len(batches[2]))
	}
}

func TestAdaptiveBatchProcessor_ProcessSequential(t *testing.T) {
	processor := NewAdaptiveBatchProcessor[int](10, 1)
	processor.EnableParallel(false)

	items := []int{1, 2, 3, 4, 5}
	ctx := context.Background()

	var processedOrder []int
	var mu sync.Mutex

	processor.Process(ctx, items, func(ctx context.Context, item int) error {
		mu.Lock()
		processedOrder = append(processedOrder, item)
		mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if len(processedOrder) != len(items) {
		t.Error("All items should be processed")
	}
}

func TestConcurrencyStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	pool := NewAdaptiveWorkerPool(10, 20, 1000)
	pool.Start()
	defer pool.Stop()

	var wg sync.WaitGroup
	var completedCount int64

	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if pool.Submit(func() error {
				time.Sleep(5 * time.Millisecond)
				atomic.AddInt64(&completedCount, 1)
				return nil
			}) {
			}
		}()
	}

	wg.Wait()
	time.Sleep(1 * time.Second)

	if completedCount < 400 {
		t.Errorf("Expected at least 400 completions under stress, got %d", completedCount)
	}
}
