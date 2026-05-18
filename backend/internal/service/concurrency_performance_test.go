package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkAdaptiveWorkerPoolSubmit(b *testing.B) {
	pool := NewAdaptiveWorkerPool(10, 20, 1000)
	pool.Start()
	defer pool.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(func() error {
			return nil
		})
	}
}

func BenchmarkAdaptiveWorkerPoolProcess(b *testing.B) {
	pool := NewAdaptiveWorkerPool(10, 20, 1000)
	pool.Start()
	defer pool.Stop()

	var counter int64

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(func() error {
			atomic.AddInt64(&counter, 1)
			time.Sleep(1 * time.Microsecond)
			return nil
		})
	}
}

func TestAdaptiveWorkerPool(t *testing.T) {
	pool := NewAdaptiveWorkerPool(5, 10, 100)
	pool.Start()
	defer pool.Stop()

	var counter int64

	for i := 0; i < 100; i++ {
		ok := pool.Submit(func() error {
			atomic.AddInt64(&counter, 1)
			return nil
		})
		if !ok {
			t.Error("Submit should succeed")
		}
	}

	time.Sleep(500 * time.Millisecond)

	metrics := pool.GetMetrics()

	if metrics.TasksSubmitted != 100 {
		t.Errorf("Expected 100 submitted tasks, got %d", metrics.TasksSubmitted)
	}

	if metrics.TasksCompleted != 100 {
		t.Errorf("Expected 100 completed tasks, got %d", metrics.TasksCompleted)
	}

	if counter != 100 {
		t.Errorf("Expected counter to be 100, got %d", counter)
	}
}

func TestAdaptiveWorkerPoolSubmitAndWait(t *testing.T) {
	pool := NewAdaptiveWorkerPool(5, 10, 100)
	pool.Start()
	defer pool.Stop()

	err := pool.SubmitAndWait(func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestAdaptiveWorkerPoolSubmitWithTimeout(t *testing.T) {
	pool := NewAdaptiveWorkerPool(1, 1, 1)
	pool.Start()
	defer pool.Stop()

	err := pool.SubmitAndWait(func() error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestConcurrencyLimiter(t *testing.T) {
	limiter := NewConcurrencyLimiter(5)
	ctx := context.Background()

	var counter int64
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := limiter.Acquire(ctx)
			if err != nil {
				t.Errorf("Acquire failed: %v", err)
				return
			}
			atomic.AddInt64(&counter, 1)
			time.Sleep(10 * time.Millisecond)
			limiter.Release()
		}()
	}

	wg.Wait()

	if counter != 10 {
		t.Errorf("Expected counter to be 10, got %d", counter)
	}
}

func TestConcurrencyLimiterConcurrent(t *testing.T) {
	limiter := NewConcurrencyLimiter(5)
	ctx := context.Background()

	var counter int64
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := limiter.Acquire(ctx)
			if err == nil {
				atomic.AddInt64(&counter, 1)
				limiter.Release()
			}
		}()
	}

	wg.Wait()

	if counter != 20 {
		t.Errorf("Expected counter to be 20, got %d", counter)
	}
}

func BenchmarkConcurrencyLimiter(b *testing.B) {
	limiter := NewConcurrencyLimiter(100)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Acquire(ctx)
		limiter.Release()
	}
}

func TestSemaphorePool(t *testing.T) {
	pool := NewSemaphorePool(5)
	ctx := context.Background()

	var counter int64

	for i := 0; i < 10; i++ {
		err := pool.Acquire(ctx, "test_key")
		if err != nil {
			t.Errorf("Acquire failed: %v", err)
			continue
		}
		atomic.AddInt64(&counter, 1)
		pool.Release("test_key")
	}

	if counter != 10 {
		t.Errorf("Expected counter to be 10, got %d", counter)
	}
}

func TestRateLimitedExecutor(t *testing.T) {
	executor := NewRateLimitedExecutor(5, 100, 10*time.Second)
	ctx := context.Background()

	var counter int64

	for i := 0; i < 10; i++ {
		err := executor.Execute(ctx, func() error {
			atomic.AddInt64(&counter, 1)
			return nil
		})
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}
	}

	if counter != 10 {
		t.Errorf("Expected counter to be 10, got %d", counter)
	}
}

func TestAdaptiveBatchProcessor(t *testing.T) {
	processor := NewAdaptiveBatchProcessor[int](10, 4)
	ctx := context.Background()

	items := make([]int, 100)
	for i := 0; i < 100; i++ {
		items[i] = i
	}

	var counter int64

	results := processor.Process(ctx, items, func(ctx context.Context, item int) error {
		atomic.AddInt64(&counter, 1)
		return nil
	})

	if len(results) != 100 {
		t.Errorf("Expected 100 results, got %d", len(results))
	}

	if counter != 100 {
		t.Errorf("Expected counter to be 100, got %d", counter)
	}
}

func BenchmarkAdaptiveBatchProcessor(b *testing.B) {
	processor := NewAdaptiveBatchProcessor[int](100, 4)
	ctx := context.Background()

	items := make([]int, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Process(ctx, items, func(ctx context.Context, item int) error {
			time.Sleep(1 * time.Microsecond)
			return nil
		})
	}
}

func TestPriorityTaskExecutor(t *testing.T) {
	executor := NewPriorityTaskExecutor(2, 2, 1)
	executor.Start()
	defer executor.Stop()

	var counter int64

	executor.SubmitHigh(func() error {
		atomic.AddInt64(&counter, 1)
		return nil
	})

	executor.SubmitNormal(func() error {
		atomic.AddInt64(&counter, 1)
		return nil
	})

	executor.SubmitLow(func() error {
		atomic.AddInt64(&counter, 1)
		return nil
	})

	time.Sleep(500 * time.Millisecond)

	if counter != 3 {
		t.Errorf("Expected counter to be 3, got %d", counter)
	}
}

func TestWorkerPoolMetrics(t *testing.T) {
	pool := NewAdaptiveWorkerPool(5, 10, 100)
	pool.Start()
	defer pool.Stop()

	metrics := pool.GetMetrics()

	if metrics.WorkerCount != 5 {
		t.Errorf("Expected 5 workers, got %d", metrics.WorkerCount)
	}

	if metrics.CurrentQueueSize != 0 {
		t.Errorf("Expected 0 queue size, got %d", metrics.CurrentQueueSize)
	}
}

func TestConcurrencyLimiterStats(t *testing.T) {
	limiter := NewConcurrencyLimiter(10)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		limiter.Acquire(ctx)
		limiter.Release()
	}

	current, waiting, maxParallel := limiter.GetStats()

	if current < 0 || current > 10 {
		t.Errorf("Current should be between 0 and 10, got %d", current)
	}
	if waiting < 0 {
		t.Errorf("Waiting should be non-negative, got %d", waiting)
	}
	if maxParallel != 10 {
		t.Errorf("MaxParallel should be 10, got %d", maxParallel)
	}
}

func TestBatchProcessorMetrics(t *testing.T) {
	processor := NewAdaptiveBatchProcessor[int](10, 4)
	ctx := context.Background()

	items := make([]int, 100)
	for i := 0; i < 100; i++ {
		items[i] = i
	}

	processor.Process(ctx, items, func(ctx context.Context, item int) error {
		return nil
	})

	t.Log("BatchProcessor metrics test completed")
}
