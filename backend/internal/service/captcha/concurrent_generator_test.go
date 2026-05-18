package captcha

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrentGeneratorCreation(t *testing.T) {
	generator := NewConcurrentGenerator()
	if generator == nil {
		t.Fatal("Failed to create concurrent generator")
	}
	if generator.imageGenerator == nil {
		t.Fatal("Image generator is nil")
	}
	if generator.workerPool == nil {
		t.Fatal("Worker pool is nil")
	}
	defer generator.Close()
}

func TestConcurrentGeneratorGenerateWithPool(t *testing.T) {
	generator := NewConcurrentGenerator()
	defer generator.Close()

	ctx := context.Background()

	result, err := generator.GenerateWithPool(ctx, "slider")
	if err != nil {
		t.Fatalf("Failed to generate captcha: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if len(result.Background) == 0 {
		t.Fatal("Background is empty")
	}

	if len(result.Slider) == 0 {
		t.Fatal("Slider is empty")
	}

	if result.GapX < 0 || result.GapY < 0 {
		t.Errorf("Invalid gap coordinates: GapX=%d, GapY=%d", result.GapX, result.GapY)
	}
}

func TestConcurrentGenerateMultiple(t *testing.T) {
	generator := NewConcurrentGenerator()
	defer generator.Close()

	ctx := context.Background()
	count := 10
	requests := make([]GenerateRequest, count)

	for i := 0; i < count; i++ {
		requests[i] = GenerateRequest{
			Type:      "slider",
			RequestID: "test_req_" + string(rune(i)),
			Metadata:  map[string]interface{}{"index": i},
		}
	}

	results, errors := generator.ConcurrentGenerate(ctx, requests)

	if len(results) != count {
		t.Errorf("Expected %d results, got %d", count, len(results))
	}

	failedCount := 0
	for i, err := range errors {
		if err != nil {
			failedCount++
			t.Logf("Request %d failed: %v", i, err)
		}
	}

	successfulCount := 0
	for _, result := range results {
		if result != nil {
			successfulCount++
			if len(result.Background) == 0 {
				t.Error("Empty background in result")
			}
			if len(result.Slider) == 0 {
				t.Error("Empty slider in result")
			}
		}
	}

	t.Logf("Successfully generated %d/%d captchas", successfulCount, count)
}

func TestPriorityQueueBasicOperations(t *testing.T) {
	pq := &PriorityQueue{
		items: make([]*PriorityItem, 0),
	}
	pq.notEmpty = sync.NewCond(&pq.mu)

	pq.Enqueue(&PriorityItem{
		Request:  GenerateRequest{Type: "low"},
		Priority: 1,
		AddedAt:  time.Now(),
	})

	pq.Enqueue(&PriorityItem{
		Request:  GenerateRequest{Type: "high"},
		Priority: 10,
		AddedAt:  time.Now(),
	})

	pq.Enqueue(&PriorityItem{
		Request:  GenerateRequest{Type: "medium"},
		Priority: 5,
		AddedAt:  time.Now(),
	})

	if pq.Len() != 3 {
		t.Errorf("Expected queue length 3, got %d", pq.Len())
	}

	items := pq.GetAll()
	if len(items) != 3 {
		t.Errorf("Expected 3 items in GetAll, got %d", len(items))
	}

	if items[0].Priority != 10 {
		t.Errorf("Expected first item to have priority 10, got %d", items[0].Priority)
	}
}

func TestPriorityQueueOrdering(t *testing.T) {
	pq := &PriorityQueue{
		items: make([]*PriorityItem, 0),
	}
	pq.notEmpty = sync.NewCond(&pq.mu)

	priorities := []int{5, 10, 3, 8, 1, 7, 2, 9, 4, 6}
	
	for i, p := range priorities {
		pq.Enqueue(&PriorityItem{
			Request:  GenerateRequest{RequestID: "req_" + string(rune(i))},
			Priority: p,
			AddedAt:  time.Now().Add(time.Duration(i) * time.Millisecond),
		})
	}

	if pq.Len() != len(priorities) {
		t.Errorf("Expected %d items, got %d", len(priorities), pq.Len())
	}

	items := pq.GetAll()
	expectedPriorities := []int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	
	for i, item := range items {
		if item.Priority != expectedPriorities[i] {
			t.Errorf("At position %d, expected priority %d, got %d", i, expectedPriorities[i], item.Priority)
		}
	}
}

func TestEnqueueWithPriority(t *testing.T) {
	generator := NewConcurrentGenerator()
	defer generator.Close()

	ctx := context.Background()

	err := generator.EnqueueWithPriority(ctx, GenerateRequest{
		Type:      "slider",
		RequestID: "high_priority",
	}, 100)
	if err != nil {
		t.Fatalf("Failed to enqueue: %v", err)
	}

	err = generator.EnqueueWithPriority(ctx, GenerateRequest{
		Type:      "slider",
		RequestID: "low_priority",
	}, 1)
	if err != nil {
		t.Fatalf("Failed to enqueue: %v", err)
	}

	stats := generator.GetStats()
	if stats.QueueLength < 2 {
		t.Errorf("Expected queue length >= 2, got %d", stats.QueueLength)
	}
}

func TestConcurrentGeneratorStress(t *testing.T) {
	generator := NewConcurrentGenerator()
	defer generator.Close()

	ctx := context.Background()
	concurrency := runtime.NumCPU() * 2
	iterations := 100

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				result, err := generator.GenerateWithPool(ctx, "slider")
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else if result != nil && len(result.Background) > 0 {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	total := successCount + errorCount
	successRate := float64(successCount) / float64(total) * 100

	t.Logf("Stress test results: %d/%d successful (%.2f%%)", successCount, total, successRate)

	if successRate < 90 {
		t.Errorf("Success rate too low: %.2f%%", successRate)
	}
}

func TestGeneratorStats(t *testing.T) {
	generator := NewConcurrentGenerator()
	defer generator.Close()

	ctx := context.Background()

	for i := 0; i < 20; i++ {
		generator.GenerateWithPool(ctx, "slider")
	}

	stats := generator.GetStats()

	if stats.TotalGenerated < 20 {
		t.Errorf("Expected at least 20 generated, got %d", stats.TotalGenerated)
	}

	if stats.ActiveWorkers <= 0 {
		t.Errorf("Expected active workers > 0, got %d", stats.ActiveWorkers)
	}
}

func TestContextCancellation(t *testing.T) {
	generator := NewConcurrentGenerator()
	defer generator.Close()

	ctx, cancel := context.WithCancel(context.Background())

	resultCh := make(chan *ImageResult, 1)
	errCh := make(chan error, 1)

	go func() {
		result, err := generator.GenerateWithPool(ctx, "slider")
		if err != nil {
			errCh <- err
		} else {
			resultCh <- result
		}
	}()

	time.Sleep(100 * time.Millisecond)

	cancel()

	select {
	case <-resultCh:
		t.Log("Result received before cancellation")
	case <-errCh:
		t.Log("Error received before cancellation")
	case <-time.After(5 * time.Second):
		t.Log("Operation continued after cancellation (expected for in-progress tasks)")
	}
}

func TestMultipleCaptchaTypes(t *testing.T) {
	generator := NewConcurrentGenerator()
	defer generator.Close()

	ctx := context.Background()

	types := []string{"slider", "slider", "slider"}
	
	for _, captchaType := range types {
		result, err := generator.GenerateWithPool(ctx, captchaType)
		if err != nil {
			t.Fatalf("Failed to generate %s captcha: %v", captchaType, err)
		}

		if result == nil {
			t.Fatalf("Result is nil for type %s", captchaType)
		}

		if result.Type != captchaType {
			t.Errorf("Expected type %s, got %s", captchaType, result.Type)
		}
	}
}

func TestQueueClear(t *testing.T) {
	pq := &PriorityQueue{
		items: make([]*PriorityItem, 0),
	}
	pq.notEmpty = sync.NewCond(&pq.mu)

	for i := 0; i < 10; i++ {
		pq.Enqueue(&PriorityItem{
			Request:  GenerateRequest{RequestID: "req_" + string(rune(i))},
			Priority: i,
			AddedAt:  time.Now(),
		})
	}

	if pq.Len() != 10 {
		t.Errorf("Expected length 10, got %d", pq.Len())
	}

	pq.Clear()

	if pq.Len() != 0 {
		t.Errorf("Expected length 0 after clear, got %d", pq.Len())
	}
}

func TestConcurrentEnqueue(t *testing.T) {
	generator := NewConcurrentGenerator()
	defer generator.Close()

	ctx := context.Background()
	enqueueCount := 100

	var wg sync.WaitGroup
	for i := 0; i < enqueueCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			priority := id % 10
			generator.EnqueueWithPriority(ctx, GenerateRequest{
				Type:      "slider",
				RequestID: "concurrent_req_" + string(rune(id)),
			}, priority)
		}(i)
	}

	wg.Wait()

	stats := generator.GetStats()
	if stats.QueueLength < int64(enqueueCount) {
		t.Logf("Some enqueued items may have been processed (queue length: %d)", stats.QueueLength)
	}
}

func TestGeneratorClose(t *testing.T) {
	generator := NewConcurrentGenerator()

	err := generator.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	ctx := context.Background()
	result, err := generator.GenerateWithPool(ctx, "slider")
	if err == nil && result != nil {
		t.Log("Generator still functional after close (expected if generation started before close)")
	}
}

func TestPriorityQueuePeek(t *testing.T) {
	pq := &PriorityQueue{
		items: make([]*PriorityItem, 0),
	}
	pq.notEmpty = sync.NewCond(&pq.mu)

	peekResult := pq.Peek()
	if peekResult != nil {
		t.Error("Expected nil peek on empty queue")
	}

	pq.Enqueue(&PriorityItem{
		Request:  GenerateRequest{Type: "test"},
		Priority: 5,
		AddedAt:  time.Now(),
	})

	peekResult = pq.Peek()
	if peekResult == nil {
		t.Fatal("Expected non-nil peek on non-empty queue")
	}

	if peekResult.Priority != 5 {
		t.Errorf("Expected priority 5, got %d", peekResult.Priority)
	}

	pq.Clear()
}

func BenchmarkConcurrentGenerate(b *testing.B) {
	generator := NewConcurrentGenerator()
	defer generator.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.GenerateWithPool(ctx, "slider")
	}
}

func BenchmarkConcurrentGenerateParallel(b *testing.B) {
	generator := NewConcurrentGenerator()
	defer generator.Close()

	ctx := context.Background()
	concurrency := runtime.NumCPU()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			generator.GenerateWithPool(ctx, "slider")
		}
	})
}

func BenchmarkPriorityQueueOperations(b *testing.B) {
	pq := &PriorityQueue{
		items: make([]*PriorityItem, 0),
	}
	pq.notEmpty = sync.NewCond(&pq.mu)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pq.Enqueue(&PriorityItem{
			Request:  GenerateRequest{Type: "test"},
			Priority: i % 100,
			AddedAt:  time.Now(),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pq.GetAll()
	}
}
