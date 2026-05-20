package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBatchProcessor_NewProcessor(t *testing.T) {
	processor := NewBatchProcessor[string](4)

	assert.NotNil(t, processor)
	assert.Equal(t, 4, processor.workerCount)
	assert.Equal(t, 3, processor.maxRetries)
}

func TestBatchProcessor_NewProcessor_DefaultWorkerCount(t *testing.T) {
	processor := NewBatchProcessor[string](0)

	assert.Equal(t, 4, processor.workerCount)
}

func TestBatchProcessor_NewProcessor_NegativeWorkerCount(t *testing.T) {
	processor := NewBatchProcessor[string](-5)

	assert.Equal(t, 4, processor.workerCount)
}

func TestBatchProcessor_SetMaxRetries(t *testing.T) {
	processor := NewBatchProcessor[string](4)

	result := processor.SetMaxRetries(10)

	assert.Equal(t, 10, result.maxRetries)
	assert.Same(t, processor, result)
}

func TestProcessBatch_EmptyItems(t *testing.T) {
	ctx := context.Background()
	items := []string{}

	results := ProcessBatch(ctx, items, 4, func(ctx context.Context, item string) (string, error) {
		return item + "_processed", nil
	})

	assert.Empty(t, results)
}

func TestProcessBatch_SingleItem(t *testing.T) {
	ctx := context.Background()
	items := []string{"item1"}

	results := ProcessBatch(ctx, items, 4, func(ctx context.Context, item string) (string, error) {
		time.Sleep(10 * time.Millisecond)
		return item + "_processed", nil
	})

	assert.Len(t, results, 1)
	assert.Equal(t, "item1_processed", results[0])
}

func TestProcessBatch_MultipleItems(t *testing.T) {
	ctx := context.Background()
	items := []string{"item1", "item2", "item3", "item4", "item5"}

	results := ProcessBatch(ctx, items, 2, func(ctx context.Context, item string) (string, error) {
		return item + "_processed", nil
	})

	assert.Len(t, results, 5)
	for i, item := range items {
		assert.Equal(t, item+"_processed", results[i])
	}
}

func TestProcessBatch_WithErrors(t *testing.T) {
	ctx := context.Background()
	items := []string{"ok1", "error", "ok2"}

	results := ProcessBatch(ctx, items, 2, func(ctx context.Context, item string) (string, error) {
		if item == "error" {
			return "", errors.New("test error")
		}
		return item + "_processed", nil
	})

	assert.Len(t, results, 3)
	assert.Equal(t, "ok1_processed", results[0])
	assert.Equal(t, "", results[1])
	assert.Equal(t, "ok2_processed", results[2])
}

func TestProcessBatch_DefaultWorkerCount(t *testing.T) {
	ctx := context.Background()
	items := []string{"item1", "item2", "item3"}

	results := ProcessBatch(ctx, items, 0, func(ctx context.Context, item string) (string, error) {
		return item + "_processed", nil
	})

	assert.Len(t, results, 3)
}

func TestProcessBatch_SmallDataset(t *testing.T) {
	ctx := context.Background()
	items := []string{"item1", "item2"}

	results := ProcessBatch(ctx, items, 8, func(ctx context.Context, item string) (string, error) {
		return item + "_processed", nil
	})

	assert.Len(t, results, 2)
}

func TestProcessBatch_LargeDataset(t *testing.T) {
	ctx := context.Background()
	items := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = "item" + string(rune(i))
	}

	results := ProcessBatch(ctx, items, 8, func(ctx context.Context, item string) (string, error) {
		return item + "_processed", nil
	})

	assert.Len(t, results, 1000)
}

func TestProcessBatch_ConcurrentProcessing(t *testing.T) {
	ctx := context.Background()
	items := []string{"item1", "item2", "item3", "item4"}
	var processedCount int32

	results := ProcessBatch(ctx, items, 4, func(ctx context.Context, item string) (string, error) {
		atomic.AddInt32(&processedCount, 1)
		time.Sleep(10 * time.Millisecond)
		return item + "_processed", nil
	})

	assert.Len(t, results, 4)
	assert.Equal(t, int32(4), processedCount)
}

func TestProcessBatch_IntegerTypes(t *testing.T) {
	ctx := context.Background()
	items := []int{1, 2, 3, 4, 5}

	results := ProcessBatch(ctx, items, 2, func(ctx context.Context, item int) (int, error) {
		return item * 2, nil
	})

	assert.Len(t, results, 5)
	expected := []int{2, 4, 6, 8, 10}
	for i, exp := range expected {
		assert.Equal(t, exp, results[i])
	}
}

func TestProcessBatch_IntegerToString(t *testing.T) {
	ctx := context.Background()
	items := []int{10, 20, 30}

	results := ProcessBatch(ctx, items, 2, func(ctx context.Context, item int) (string, error) {
		return string(rune(item)), nil
	})

	assert.Len(t, results, 3)
}

func TestProcessBatch_StructTypes(t *testing.T) {
	ctx := context.Background()
	type TestItem struct {
		ID   int
		Name string
	}
	items := []TestItem{
		{ID: 1, Name: "first"},
		{ID: 2, Name: "second"},
	}

	results := ProcessBatch(ctx, items, 2, func(ctx context.Context, item TestItem) (TestItem, error) {
		return TestItem{ID: item.ID * 10, Name: item.Name + "_processed"}, nil
	})

	assert.Len(t, results, 2)
	assert.Equal(t, 10, results[0].ID)
	assert.Equal(t, "first_processed", results[0].Name)
}

func TestProcessBatchSimple_EmptyItems(t *testing.T) {
	ctx := context.Background()
	items := []string{}

	errors := ProcessBatchSimple(ctx, items, 4, func(ctx context.Context, item string) error {
		return nil
	})

	assert.Empty(t, errors)
}

func TestProcessBatchSimple_Success(t *testing.T) {
	ctx := context.Background()
	items := []string{"item1", "item2", "item3"}
	var processedCount int32

	errors := ProcessBatchSimple(ctx, items, 2, func(ctx context.Context, item string) error {
		atomic.AddInt32(&processedCount, 1)
		return nil
	})

	assert.Len(t, errors, 3)
	assert.Equal(t, int32(3), processedCount)
	for _, err := range errors {
		assert.NoError(t, err)
	}
}

func TestProcessBatchSimple_WithErrors(t *testing.T) {
	ctx := context.Background()
	items := []string{"ok1", "error1", "error2", "ok2"}

	errors := ProcessBatchSimple(ctx, items, 2, func(ctx context.Context, item string) error {
		if item == "error1" || item == "error2" {
			return errors.New("processing error")
		}
		return nil
	})

	assert.Len(t, errors, 4)
	assert.NoError(t, errors[0])
	assert.Error(t, errors[1])
	assert.Error(t, errors[2])
	assert.NoError(t, errors[3])
}

func TestProcessBatchSimple_LargeDataset(t *testing.T) {
	ctx := context.Background()
	items := make([]string, 500)
	for i := 0; i < 500; i++ {
		items[i] = "item" + string(rune(i))
	}

	errors := ProcessBatchSimple(ctx, items, 8, func(ctx context.Context, item string) error {
		return nil
	})

	assert.Len(t, errors, 500)
}

func TestChunkProcessor_NewProcessor(t *testing.T) {
	processor := NewChunkProcessor[string, string](100, 4)

	assert.NotNil(t, processor)
	assert.Equal(t, 100, processor.chunkSize)
	assert.Equal(t, 4, processor.workerCount)
}

func TestChunkProcessor_NewProcessor_Defaults(t *testing.T) {
	processor := NewChunkProcessor[string, string](0, 0)

	assert.Equal(t, 100, processor.chunkSize)
	assert.Equal(t, 4, processor.workerCount)
}

func TestChunkProcessor_ProcessChunks_EmptyItems(t *testing.T) {
	ctx := context.Background()
	processor := NewChunkProcessor[string, string](10, 2)

	results, err := processor.ProcessChunks(ctx, []string{}, func(ctx context.Context, chunk []string) ([]string, error) {
		return nil, nil
	})

	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestChunkProcessor_ProcessChunks_SingleChunk(t *testing.T) {
	ctx := context.Background()
	processor := NewChunkProcessor[string, string](100, 2)
	items := []string{"a", "b", "c"}

	results, err := processor.ProcessChunks(ctx, items, func(ctx context.Context, chunk []string) ([]string, error) {
		processed := make([]string, len(chunk))
		for i, item := range chunk {
			processed[i] = item + "_processed"
		}
		return processed, nil
	})

	assert.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestChunkProcessor_ProcessChunks_MultipleChunks(t *testing.T) {
	ctx := context.Background()
	processor := NewChunkProcessor[string, string](10, 2)
	items := make([]string, 25)
	for i := 0; i < 25; i++ {
		items[i] = "item"
	}

	results, err := processor.ProcessChunks(ctx, items, func(ctx context.Context, chunk []string) ([]string, error) {
		processed := make([]string, len(chunk))
		for i, item := range chunk {
			processed[i] = item + "_chunk"
		}
		return processed, nil
	})

	assert.NoError(t, err)
	assert.Len(t, results, 25)
}

func TestChunkProcessor_ProcessChunks_WithError(t *testing.T) {
	ctx := context.Background()
	processor := NewChunkProcessor[string, string](10, 2)
	items := []string{"ok", "error", "ok"}

	results, err := processor.ProcessChunks(ctx, items, func(ctx context.Context, chunk []string) ([]string, error) {
		for _, item := range chunk {
			if item == "error" {
				return nil, errors.New("chunk error")
			}
		}
		return chunk, nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chunk error")
}

func TestChunkProcessor_ProcessChunks_Concurrent(t *testing.T) {
	ctx := context.Background()
	processor := NewChunkProcessor[string, string](5, 4)
	items := make([]string, 20)
	for i := 0; i < 20; i++ {
		items[i] = "item"
	}

	var processedCount int32
	results, err := processor.ProcessChunks(ctx, items, func(ctx context.Context, chunk []string) ([]string, error) {
		atomic.AddInt32(&processedCount, int32(len(chunk)))
		time.Sleep(5 * time.Millisecond)
		return chunk, nil
	})

	assert.NoError(t, err)
	assert.Len(t, results, 20)
	assert.Equal(t, int32(20), processedCount)
}

func TestChunkProcessor_SplitIntoChunks(t *testing.T) {
	processor := NewChunkProcessor[string, string](10, 2)

	items := make([]string, 25)
	chunks := processor.splitIntoChunks(items)

	assert.Len(t, chunks, 3)
	assert.Len(t, chunks[0], 10)
	assert.Len(t, chunks[1], 10)
	assert.Len(t, chunks[2], 5)
}

func TestChunkProcessor_SplitIntoChunks_ExactDivision(t *testing.T) {
	processor := NewChunkProcessor[string, string](10, 2)

	items := make([]string, 30)
	chunks := processor.splitIntoChunks(items)

	assert.Len(t, chunks, 3)
	for _, chunk := range chunks {
		assert.Len(t, chunk, 10)
	}
}

func TestChunkProcessor_SplitIntoChunks_LessThanChunkSize(t *testing.T) {
	processor := NewChunkProcessor[string, string](10, 2)

	items := make([]string, 5)
	chunks := processor.splitIntoChunks(items)

	assert.Len(t, chunks, 1)
	assert.Len(t, chunks[0], 5)
}

func TestChunkProcessor_SplitIntoChunks_Empty(t *testing.T) {
	processor := NewChunkProcessor[string, string](10, 2)

	chunks := processor.splitIntoChunks([]string{})

	assert.Empty(t, chunks)
}

func TestWorkerPool_NewPool(t *testing.T) {
	handler := func(ctx context.Context, task string) {}

	pool := NewWorkerPool[string](4, 100, handler)

	assert.NotNil(t, pool)
	assert.Equal(t, 4, pool.workers)
	assert.Equal(t, 100, cap(pool.tasks))
	assert.NotNil(t, pool.ctx)
	assert.NotNil(t, pool.cancel)
}

func TestWorkerPool_NewPool_Defaults(t *testing.T) {
	handler := func(ctx context.Context, task string) {}

	pool := NewWorkerPool[string](0, 0, handler)

	assert.Equal(t, 4, pool.workers)
	assert.Equal(t, 1000, cap(pool.tasks))
}

func TestWorkerPool_Start(t *testing.T) {
	handler := func(ctx context.Context, task string) {}
	pool := NewWorkerPool[string](2, 10, handler)

	pool.Start()

	assert.True(t, pool.IsRunning())

	pool.Stop()
}

func TestWorkerPool_Submit(t *testing.T) {
	var mu sync.Mutex
	processed := make([]string, 0)

	handler := func(ctx context.Context, task string) {
		mu.Lock()
		processed = append(processed, task)
		mu.Unlock()
	}

	pool := NewWorkerPool[string](2, 10, handler)
	pool.Start()

	success := pool.Submit("task1")
	assert.True(t, success)

	success = pool.Submit("task2")
	assert.True(t, success)

	pool.Wait()
	pool.Stop()

	mu.Lock()
	assert.Len(t, processed, 2)
	assert.Contains(t, processed, "task1")
	assert.Contains(t, processed, "task2")
	mu.Unlock()
}

func TestWorkerPool_Submit_NotRunning(t *testing.T) {
	handler := func(ctx context.Context, task string) {}
	pool := NewWorkerPool[string](2, 10, handler)

	success := pool.Submit("task")

	assert.False(t, success)
}

func TestWorkerPool_Stop(t *testing.T) {
	handler := func(ctx context.Context, task string) {
		time.Sleep(10 * time.Millisecond)
	}
	pool := NewWorkerPool[string](2, 10, handler)

	pool.Start()
	pool.Submit("task1")
	pool.Submit("task2")

	time.Sleep(5 * time.Millisecond)
	pool.Stop()

	assert.False(t, pool.IsRunning())
}

func TestWorkerPool_Wait(t *testing.T) {
	var mu sync.Mutex
	processed := 0

	handler := func(ctx context.Context, task string) {
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		processed++
		mu.Unlock()
	}

	pool := NewWorkerPool[string](2, 10, handler)
	pool.Start()

	pool.Submit("task1")
	pool.Submit("task2")

	pool.Wait()

	mu.Lock()
	assert.Equal(t, 2, processed)
	mu.Unlock()

	pool.Stop()
}

func TestWorkerPool_BufferFull(t *testing.T) {
	handler := func(ctx context.Context, task string) {
		time.Sleep(100 * time.Millisecond)
	}
	pool := NewWorkerPool[string](1, 2, handler)
	pool.Start()

	for i := 0; i < 2; i++ {
		pool.Submit("task")
	}

	success := pool.Submit("task")

	assert.False(t, success)

	pool.Stop()
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	var cancelCalled bool
	ctx, cancel := context.WithCancel(context.Background())

	handler := func(ctx context.Context, task string) {
		<-ctx.Done()
		cancelCalled = true
	}

	pool := NewWorkerPoolWithContext[string](ctx, 2, 10, handler)
	pool.Start()

	cancel()
	time.Sleep(10 * time.Millisecond)
	pool.Stop()

	assert.True(t, cancelCalled)
}

func TestWorkerPoolWithContext_NewPool(t *testing.T) {
	ctx := context.Background()
	handler := func(ctx context.Context, task string) {}

	pool := NewWorkerPoolWithContext[string](ctx, 4, 100, handler)

	assert.NotNil(t, pool)
	assert.Equal(t, 4, pool.workers)
}

func TestBatchItem_Struct(t *testing.T) {
	item := BatchItem[string]{
		Index: 5,
		Data:  "test_data",
	}

	assert.Equal(t, 5, item.Index)
	assert.Equal(t, "test_data", item.Data)
}

func TestBatchResult_Struct(t *testing.T) {
	result := BatchResult[string]{
		Index: 3,
		Data:  "result_data",
		Error: nil,
	}

	assert.Equal(t, 3, result.Index)
	assert.Equal(t, "result_data", result.Data)
	assert.Nil(t, result.Error)
}

func TestBatchResult_WithError(t *testing.T) {
	testErr := errors.New("test error")

	result := BatchResult[string]{
		Index: 2,
		Data:  "",
		Error: testErr,
	}

	assert.Equal(t, 2, result.Index)
	assert.Equal(t, "", result.Data)
	assert.Error(t, result.Error)
	assert.Equal(t, "test error", result.Error.Error())
}

func TestProcessFunc_Type(t *testing.T) {
	var fn ProcessFunc[string, int]

	fn = func(ctx context.Context, item string) (int, error) {
		return len(item), nil
	}

	result, err := fn(context.Background(), "hello")
	assert.NoError(t, err)
	assert.Equal(t, 5, result)
}

func TestProcessFuncSimple_Type(t *testing.T) {
	var fn ProcessFuncSimple[string]

	fn = func(ctx context.Context, item string) error {
		return nil
	}

	err := fn(context.Background(), "hello")
	assert.NoError(t, err)
}

func TestBatchProcessor_ConcurrentStress(t *testing.T) {
	ctx := context.Background()
	items := make([]int, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = i
	}

	start := time.Now()
	results := ProcessBatch(ctx, items, 16, func(ctx context.Context, item int) (int, error) {
		return item * 2, nil
	})
	duration := time.Since(start)

	assert.Len(t, results, 1000)
	assert.Less(t, duration, 5*time.Second)
}

func TestChunkProcessor_ConcurrentStress(t *testing.T) {
	ctx := context.Background()
	items := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = "item"
	}

	processor := NewChunkProcessor[string, string](50, 16)

	start := time.Now()
	results, err := processor.ProcessChunks(ctx, items, func(ctx context.Context, chunk []string) ([]string, error) {
		processed := make([]string, len(chunk))
		for i, item := range chunk {
			processed[i] = item + "_processed"
		}
		return processed, nil
	})
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Len(t, results, 1000)
	assert.Less(t, duration, 5*time.Second)
}

func TestWorkerPool_ConcurrentStress(t *testing.T) {
	var processedCount int32
	var mu sync.Mutex

	handler := func(ctx context.Context, task string) {
		time.Sleep(1 * time.Millisecond)
		atomic.AddInt32(&processedCount, 1)
	}

	pool := NewWorkerPool[string](16, 1000, handler)
	pool.Start()

	for i := 0; i < 500; i++ {
		pool.Submit("task")
	}

	pool.Wait()
	pool.Stop()

	assert.Equal(t, int32(500), processedCount)
}

func TestBatchProcessor_TypeConversion(t *testing.T) {
	ctx := context.Background()
	type Input struct {
		ID   int
		Name string
	}
	type Output struct {
		ID     int
		Name   string
		Result string
	}

	items := []Input{
		{ID: 1, Name: "first"},
		{ID: 2, Name: "second"},
	}

	results := ProcessBatch(ctx, items, 2, func(ctx context.Context, item Input) (Output, error) {
		return Output{
			ID:     item.ID,
			Name:   item.Name,
			Result: "processed",
		}, nil
	})

	assert.Len(t, results, 2)
	assert.Equal(t, 1, results[0].ID)
	assert.Equal(t, "first", results[0].Name)
	assert.Equal(t, "processed", results[0].Result)
}

func TestBatchProcessor_NilError(t *testing.T) {
	ctx := context.Background()
	items := []string{"a", "b", "c"}

	results := ProcessBatch(ctx, items, 2, func(ctx context.Context, item string) (string, error) {
		return item, nil
	})

	assert.Len(t, results, 3)
	for _, r := range results {
		assert.NotEmpty(t, r)
	}
}

func TestChunkProcessor_StructTransformation(t *testing.T) {
	ctx := context.Background()
	type Input struct{ Value int }
	type Output struct{ Doubled int }

	processor := NewChunkProcessor[Input, Output](10, 2)
	items := []Input{{Value: 1}, {Value: 2}, {Value: 3}}

	results, err := processor.ProcessChunks(ctx, items, func(ctx context.Context, chunk []Input) ([]Output, error) {
		output := make([]Output, len(chunk))
		for i, inp := range chunk {
			output[i] = Output{Doubled: inp.Value * 2}
		}
		return output, nil
	})

	assert.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, 2, results[0].Doubled)
	assert.Equal(t, 4, results[1].Doubled)
	assert.Equal(t, 6, results[2].Doubled)
}
