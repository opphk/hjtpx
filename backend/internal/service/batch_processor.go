package service

import (
	"context"
	"sync"
)

type BatchItem[T any] struct {
	Index int
	Data  T
}

type BatchResult[T any] struct {
	Index int
	Data  T
	Error error
}

type BatchProcessor[T any] struct {
	workerCount int
	maxRetries int
}

func NewBatchProcessor[T any](workerCount int) *BatchProcessor[T] {
	if workerCount <= 0 {
		workerCount = 4
	}
	return &BatchProcessor[T]{
		workerCount: workerCount,
		maxRetries: 3,
	}
}

func (p *BatchProcessor[T]) SetMaxRetries(maxRetries int) *BatchProcessor[T] {
	p.maxRetries = maxRetries
	return p
}

type ProcessFunc[T any, R any] func(ctx context.Context, item T) (R, error)

func ProcessBatch[T any, R any](ctx context.Context, items []T, workerCount int, fn ProcessFunc[T, R]) []R {
	results := make([]R, len(items))
	if len(items) == 0 {
		return results
	}

	if workerCount <= 0 {
		workerCount = 4
	}

	if len(items) < workerCount*4 {
		for i, item := range items {
			result, _ := fn(ctx, item)
			results[i] = result
		}
		return results
	}

	itemChan := make(chan BatchItem[T], len(items))
	resultChan := make(chan BatchResult[R], len(items))
	var wg sync.WaitGroup

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range itemChan {
				result, err := fn(ctx, item.Data)
				resultChan <- BatchResult[R]{
					Index: item.Index,
					Data:  result,
					Error: err,
				}
			}
		}()
	}

	for i, item := range items {
		itemChan <- BatchItem[T]{Index: i, Data: item}
	}
	close(itemChan)

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		if result.Error == nil {
			results[result.Index] = result.Data
		}
	}

	return results
}

type ProcessFuncSimple[T any] func(ctx context.Context, item T) error

func ProcessBatchSimple[T any](ctx context.Context, items []T, workerCount int, fn ProcessFuncSimple[T]) []error {
	errors := make([]error, len(items))
	if len(items) == 0 {
		return errors
	}

	if workerCount <= 0 {
		workerCount = 4
	}

	if len(items) < workerCount*4 {
		for i, item := range items {
			errors[i] = fn(ctx, item)
		}
		return errors
	}

	itemChan := make(chan BatchItem[T], len(items))
	resultChan := make(chan BatchResult[struct{}], len(items))
	var wg sync.WaitGroup

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range itemChan {
				err := fn(ctx, item.Data)
				resultChan <- BatchResult[struct{}]{
					Index: item.Index,
					Error: err,
				}
			}
		}()
	}

	for i, item := range items {
		itemChan <- BatchItem[T]{Index: i, Data: item}
	}
	close(itemChan)

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		errors[result.Index] = result.Error
	}

	return errors
}

type ChunkProcessor[T any, R any] struct {
	chunkSize   int
	workerCount int
}

func NewChunkProcessor[T any, R any](chunkSize, workerCount int) *ChunkProcessor[T, R] {
	if chunkSize <= 0 {
		chunkSize = 100
	}
	if workerCount <= 0 {
		workerCount = 4
	}
	return &ChunkProcessor[T, R]{
		chunkSize:   chunkSize,
		workerCount: workerCount,
	}
}

func (cp *ChunkProcessor[T, R]) ProcessChunks(ctx context.Context, items []T, fn func(ctx context.Context, chunk []T) ([]R, error)) ([]R, error) {
	if len(items) == 0 {
		return []R{}, nil
	}

	chunks := cp.splitIntoChunks(items)
	resultChan := make(chan []R, len(chunks))
	errorChan := make(chan error, cp.workerCount)
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, cp.workerCount)

	for _, chunk := range chunks {
		wg.Add(1)
		go func(c []T) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			results, err := fn(ctx, c)
			if err != nil {
				errorChan <- err
				return
			}
			resultChan <- results
		}(chunk)
	}

	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	var allResults []R
	for results := range resultChan {
		allResults = append(allResults, results...)
	}

	for err := range errorChan {
		if err != nil {
			return allResults, err
		}
	}

	return allResults, nil
}

func (cp *ChunkProcessor[T, R]) splitIntoChunks(items []T) [][]T {
	var chunks [][]T
	for i := 0; i < len(items); i += cp.chunkSize {
		end := i + cp.chunkSize
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}
	return chunks
}

type WorkerPool[T any] struct {
	tasks   chan T
	workers int
	handler func(ctx context.Context, task T)
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mu      sync.RWMutex
}

func NewWorkerPool[T any](workers int, bufferSize int, handler func(ctx context.Context, task T)) *WorkerPool[T] {
	if workers <= 0 {
		workers = 4
	}
	if bufferSize <= 0 {
		bufferSize = 1000
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool[T]{
		tasks:   make(chan T, bufferSize),
		workers: workers,
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (wp *WorkerPool[T]) Start() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.running {
		return
	}
	wp.running = true

	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go func() {
			defer wp.wg.Done()
			for {
				select {
				case <-wp.ctx.Done():
					return
				case task, ok := <-wp.tasks:
					if !ok {
						return
					}
					wp.handler(wp.ctx, task)
				}
			}
		}()
	}
}

func (wp *WorkerPool[T]) Submit(task T) bool {
	wp.mu.RLock()
	running := wp.running
	wp.mu.RUnlock()

	if !running {
		return false
	}

	select {
	case wp.tasks <- task:
		return true
	default:
		return false
	}
}

func (wp *WorkerPool[T]) Stop() {
	wp.mu.Lock()
	wp.running = false
	wp.mu.Unlock()

	wp.cancel()
	close(wp.tasks)
	wp.wg.Wait()
}

func (wp *WorkerPool[T]) Wait() {
	wp.wg.Wait()
}
