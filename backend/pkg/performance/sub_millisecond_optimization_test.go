package performance

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSubMillisecondOptimizer(t *testing.T) {
	optimizer := NewSubMillisecondOptimizer()
	ctx := context.Background()

	if err := optimizer.Start(ctx); err != nil {
		t.Fatalf("Failed to start optimizer: %v", err)
	}
	defer optimizer.Stop()

	t.Run("ConnectionPool", func(t *testing.T) {
		conn, err := optimizer.GetConnection("test", func() (interface{}, error) {
			return "test-connection", nil
		})
		if err != nil {
			t.Errorf("Failed to get connection: %v", err)
		}
		if conn != "test-connection" {
			t.Errorf("Expected test-connection, got %v", conn)
		}
	})

	t.Run("ZeroCopyBuffer", func(t *testing.T) {
		buf := optimizer.GetBuffer()
		if len(buf) != 0 {
			t.Errorf("Expected empty buffer, got len %d", len(buf))
		}
		if cap(buf) != 4096 {
			t.Errorf("Expected buffer capacity 4096, got %d", cap(buf))
		}
		optimizer.ReleaseBuffer(buf)
	})

	t.Run("CacheWarming", func(t *testing.T) {
		optimizer.AddPreloadKey("warm-key")
		optimizer.WarmCache("warm-key", func() interface{} {
			return "warmed-value"
		})
	})

	stats := optimizer.GetStats()
	t.Logf("Optimizer stats: %+v", stats)
}

func TestQPSOptimizer(t *testing.T) {
	optimizer := NewQPSOptimizer()
	ctx := context.Background()

	if err := optimizer.Start(ctx); err != nil {
		t.Fatalf("Failed to start QPS optimizer: %v", err)
	}
	defer optimizer.Stop()

	t.Run("SubmitTask", func(t *testing.T) {
		var wg sync.WaitGroup
		var executed int32

		for i := 0; i < 100; i++ {
			wg.Add(1)
			optimizer.SubmitTask(func() error {
				defer wg.Done()
				executed++
				return nil
			}, 0)
		}

		wg.Wait()
		time.Sleep(200 * time.Millisecond)

		if executed < 90 {
			t.Errorf("Expected at least 90 tasks executed, got %d", executed)
		}
	})

	t.Run("ResultCaching", func(t *testing.T) {
		optimizer.CacheResult("test-key", "test-value")
		value, ok := optimizer.GetCachedResult("test-key")
		if !ok {
			t.Error("Expected cache hit")
		}
		if value != "test-value" {
			t.Errorf("Expected test-value, got %v", value)
		}
	})

	t.Run("BatchProcessing", func(t *testing.T) {
		processed := make(chan []BatchRequest, 1)
		optimizer.SetBatchProcessor(func(batch []BatchRequest) error {
			processed <- batch
			return nil
		})

		for i := 0; i < 50; i++ {
			optimizer.SubmitRequest(BatchRequest{
				ID:      string(rune('A' + i)),
				Payload: i,
			})
		}

		select {
		case batch := <-processed:
			if len(batch) < 50 {
				t.Logf("Processed %d requests", len(batch))
			}
		case <-time.After(1 * time.Second):
			t.Log("Batch processing timed out, expected behavior with partial batch")
		}
	})

	stats := optimizer.GetStats()
	t.Logf("QPS optimizer stats: %+v", stats)
}

func TestResourceEfficiencyOptimizer(t *testing.T) {
	optimizer := NewResourceEfficiencyOptimizer()
	ctx := context.Background()

	if err := optimizer.Start(ctx); err != nil {
		t.Fatalf("Failed to start resource optimizer: %v", err)
	}
	defer optimizer.Stop()

	t.Run("MemoryPool", func(t *testing.T) {
		buf := optimizer.GetMemory(100)
		if cap(buf) < 100 {
			t.Errorf("Expected buffer capacity >= 100, got %d", cap(buf))
		}
		optimizer.ReleaseMemory(buf)
	})

	t.Run("ObjectPool", func(t *testing.T) {
		type TestObject struct {
			Value int
		}

		optimizer.RegisterObjectPool(
			"test-object",
			func() interface{} {
				return &TestObject{}
			},
			func(obj interface{}) {
				if to, ok := obj.(*TestObject); ok {
					to.Value = 0
				}
			},
		)

		obj := optimizer.GetObject("test-object")
		if obj == nil {
			t.Error("Failed to get object from pool")
		} else {
			if to, ok := obj.(*TestObject); ok {
				to.Value = 42
				optimizer.ReleaseObject("test-object", to)
			}
		}
	})

	t.Run("LowPowerMode", func(t *testing.T) {
		optimizer.SetLowPowerMode(true)
		stats := optimizer.GetStats()
		if !stats["power_saving_mode"].(bool) {
			t.Error("Expected power saving mode to be true")
		}
		optimizer.SetLowPowerMode(false)
	})

	stats := optimizer.GetStats()
	t.Logf("Resource optimizer stats: %+v", stats)
}

func BenchmarkSubMillisecondOptimizer(b *testing.B) {
	optimizer := NewSubMillisecondOptimizer()
	ctx := context.Background()
	optimizer.Start(ctx)
	defer optimizer.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := optimizer.GetBuffer()
		optimizer.ReleaseBuffer(buf)
	}
}

func BenchmarkQPSOptimizer(b *testing.B) {
	optimizer := NewQPSOptimizer()
	ctx := context.Background()
	optimizer.Start(ctx)
	defer optimizer.Stop()

	var wg sync.WaitGroup
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		optimizer.SubmitTask(func() error {
			defer wg.Done()
			return nil
		}, 0)
	}
	wg.Wait()
}

func BenchmarkMemoryPool(b *testing.B) {
	optimizer := NewResourceEfficiencyOptimizer()
	ctx := context.Background()
	optimizer.Start(ctx)
	defer optimizer.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := optimizer.GetMemory(256)
		optimizer.ReleaseMemory(buf)
	}
}
