package service

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkGoroutinePoolSubmit(b *testing.B) {
	pool := NewGoroutinePool(runtime.NumCPU(), 10000)
	pool.Start()
	defer pool.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.Submit(func() error {
				return nil
			})
		}
	})
}

func BenchmarkGoroutinePoolExecute(b *testing.B) {
	pool := NewGoroutinePool(runtime.NumCPU()*2, 10000)
	pool.Start()
	defer pool.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(func() error {
			time.Sleep(time.Microsecond)
			return nil
		})
	}
	pool.Await()
}

func BenchmarkSemaphoreAcquire(b *testing.B) {
	sem := NewSemaphore(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sem.Acquire()
			sem.Release()
		}
	})
}

func BenchmarkGoroutinePoolThroughput(b *testing.B) {
	pool := NewGoroutinePool(runtime.NumCPU()*4, 50000)
	pool.Start()
	defer pool.Stop()

	var counter int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.Submit(func() error {
				atomic.AddInt64(&counter, 1)
				return nil
			})
		}
	})
	b.ReportMetric(float64(counter)/b.Elapsed().Seconds(), "ops/sec")
}

func BenchmarkBatchExecutor(b *testing.B) {
	executor := NewBatchExecutor(runtime.NumCPU()*2, 1000)
	defer executor.Stop()

	items := make([]interface{}, 10000)
	for i := range items {
		items[i] = i
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		results := executor.ExecuteBatch(items, func(item interface{}) error {
			time.Sleep(time.Microsecond)
			return nil
		})
		_ = results
	})
}

func TestGoroutinePoolMetrics(t *testing.T) {
	pool := NewGoroutinePool(4, 100)
	pool.Start()
	defer pool.Stop()

	for i := 0; i < 50; i++ {
		pool.SubmitAndWait(func() error {
			return nil
		})
	}

	stats := pool.Stats()
	if stats.TotalTasks == 0 {
		t.Error("expected TotalTasks > 0")
	}

	metrics := pool.Metrics()
	if metrics.Utilization < 0 || metrics.Utilization > 1 {
		t.Errorf("unexpected utilization: %f", metrics.Utilization)
	}
}

func TestGoroutinePoolAutoScale(t *testing.T) {
	pool := NewGoroutinePool(2, 1000)
	pool.AutoScale(true, 2, 16)
	pool.Start()
	defer pool.Stop()

	for i := 0; i < 500; i++ {
		pool.Submit(func() error {
			time.Sleep(time.Millisecond)
			return nil
		})
	}

	pool.AdjustScale()

	workers := pool.WorkerCount()
	if workers < 2 || workers > 16 {
		t.Errorf("unexpected worker count: %d", workers)
	}
}

func TestGoroutinePoolSubmitAndWait(t *testing.T) {
	pool := NewGoroutinePool(2, 100)
	pool.Start()
	defer pool.Stop()

	err := pool.SubmitAndWait(func() error {
		time.Sleep(time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSemaphoreAvailable(t *testing.T) {
	sem := NewSemaphore(5)

	if sem.Available() != 5 {
		t.Errorf("expected 5 available, got %d", sem.Available())
	}

	sem.Acquire()
	if sem.Available() != 4 {
		t.Errorf("expected 4 available, got %d", sem.Available())
	}

	sem.Release()
	if sem.Available() != 5 {
		t.Errorf("expected 5 available, got %d", sem.Available())
	}
}

func TestBatchExecutor(t *testing.T) {
	executor := NewBatchExecutor(4, 100)
	defer executor.Stop()

	items := make([]interface{}, 100)
	for i := range items {
		items[i] = i
	}

	results := executor.ExecuteBatch(items, func(item interface{}) error {
		return nil
	})

	if len(results) != len(items) {
		t.Errorf("expected %d results, got %d", len(items), len(results))
	}
}

func TestOnceRunner(t *testing.T) {
	runner := NewOnceRunner()
	count := 0

	for i := 0; i < 10; i++ {
		ran, _ := runner.Run(func() {
			count++
		})
		if ran && count != 1 {
			t.Errorf("expected count to be 1 after first run, got %d", count)
		}
		if !ran && count != 1 {
			t.Errorf("expected count to remain 1 after subsequent runs, got %d", count)
		}
	}
}

func BenchmarkPoolStats(b *testing.B) {
	pool := NewGoroutinePool(8, 1000)
	pool.Start()
	defer pool.Stop()

	for i := 0; i < 100; i++ {
		pool.Submit(func() error { return nil })
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pool.Stats()
	}
}

func BenchmarkPoolMetrics(b *testing.B) {
	pool := NewGoroutinePool(8, 1000)
	pool.Start()
	defer pool.Stop()

	for i := 0; i < 100; i++ {
		pool.Submit(func() error { return nil })
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pool.Metrics()
	}
}

func BenchmarkAtomicOperations(b *testing.B) {
	var counter int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			atomic.AddInt64(&counter, 1)
		}
	})
}

func TestGoroutinePoolConcurrency(t *testing.T) {
	pool := NewGoroutinePool(10, 10000)
	pool.Start()
	defer pool.Stop()

	var wg sync.WaitGroup
	concurrency := 100

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				pool.SubmitAndWait(func() error {
					return nil
				})
			}
		}()
	}

	wg.Wait()

	stats := pool.Stats()
	if stats.TotalTasks < int64(concurrency*100) {
		t.Errorf("expected at least %d tasks, got %d", concurrency*100, stats.TotalTasks)
	}
}
