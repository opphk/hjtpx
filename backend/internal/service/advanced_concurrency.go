package service

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type AdaptiveWorkerPool struct {
	tasks       chan func() error
	workers     int
	maxWorkers  int
	minWorkers  int
	activeCount atomic.Int64
	queueSize   int
	running     atomic.Bool
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	metrics     *workerPoolMetrics
	autotune    bool
	mu          sync.RWMutex
}

type workerPoolMetrics struct {
	TasksSubmitted   atomic.Int64
	TasksCompleted   atomic.Int64
	TasksFailed      atomic.Int64
	TotalLatency     atomic.Int64
	AvgLatency       atomic.Int64
	MaxQueueSize     atomic.Int64
	CurrentQueueSize atomic.Int64
	WorkerCount      atomic.Int64
}

type WorkerPoolMetrics struct {
	TasksSubmitted   int64
	TasksCompleted   int64
	TasksFailed      int64
	TotalLatency     int64
	AvgLatency       int64
	MaxQueueSize     int64
	CurrentQueueSize int64
	WorkerCount      int64
}

func NewAdaptiveWorkerPool(minWorkers, maxWorkers, queueSize int) *AdaptiveWorkerPool {
	if minWorkers <= 0 {
		minWorkers = runtime.NumCPU()
	}
	if maxWorkers <= 0 {
		maxWorkers = minWorkers * 2
	}
	if queueSize <= 0 {
		queueSize = 1000
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &AdaptiveWorkerPool{
		tasks:      make(chan func() error, queueSize),
		workers:    minWorkers,
		maxWorkers: maxWorkers,
		minWorkers: minWorkers,
		queueSize:  queueSize,
		ctx:        ctx,
		cancel:     cancel,
		metrics:    &workerPoolMetrics{},
		autotune:   true,
	}

	pool.metrics.WorkerCount.Store(int64(minWorkers))

	return pool
}

func (p *AdaptiveWorkerPool) Start() {
	if !p.running.CompareAndSwap(false, true) {
		return
	}

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}

	if p.autotune {
		go p.autoTune()
	}
}

func (p *AdaptiveWorkerPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.tasks:
			if !ok {
				return
			}

			p.activeCount.Add(1)
			start := time.Now()

			if err := task(); err != nil {
				p.metrics.TasksFailed.Add(1)
			} else {
				p.metrics.TasksCompleted.Add(1)
			}

			latency := time.Since(start).Nanoseconds()
			p.metrics.TotalLatency.Add(latency)
			p.metrics.AvgLatency.Store(p.metrics.TotalLatency.Load() / p.metrics.TasksCompleted.Load())

			p.activeCount.Add(-1)
		}
	}
}

func (p *AdaptiveWorkerPool) Submit(task func() error) bool {
	if !p.running.Load() {
		return false
	}

	queueSize := int(p.metrics.CurrentQueueSize.Load())
	if int64(queueSize) > p.metrics.MaxQueueSize.Load() {
		p.metrics.MaxQueueSize.Store(int64(queueSize))
	}

	select {
	case p.tasks <- task:
		p.metrics.TasksSubmitted.Add(1)
		p.metrics.CurrentQueueSize.Add(1)
		return true
	default:
		return false
	}
}

func (p *AdaptiveWorkerPool) SubmitAndWait(task func() error) error {
	if !p.running.Load() {
		return fmt.Errorf("worker pool not running")
	}

	done := make(chan error, 1)

	select {
	case p.tasks <- func() error {
		err := task()
		done <- err
		p.metrics.CurrentQueueSize.Add(-1)
		return err
	}:
		p.metrics.TasksSubmitted.Add(1)
		return <-done
	default:
		return fmt.Errorf("worker pool queue full")
	}
}

func (p *AdaptiveWorkerPool) autoTune() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.analyzeAndAdjust()
		}
	}
}

func (p *AdaptiveWorkerPool) analyzeAndAdjust() {
	queueLen := int(p.metrics.CurrentQueueSize.Load())
	activeWorkers := p.activeCount.Load()
	currentWorkers := int64(p.workers)

	avgLatency := p.metrics.AvgLatency.Load()
	completionRate := p.metrics.TasksCompleted.Load()

	if completionRate == 0 {
		return
	}

	latencyPerTask := avgLatency / completionRate

	if queueLen > p.queueSize/2 && activeWorkers >= currentWorkers && currentWorkers < int64(p.maxWorkers) {
		newWorkers := int(currentWorkers + 1)
		if newWorkers > p.maxWorkers {
			newWorkers = p.maxWorkers
		}
		for i := currentWorkers; i < int64(newWorkers); i++ {
			p.wg.Add(1)
			go p.worker()
		}
		p.mu.Lock()
		p.workers = newWorkers
		p.mu.Unlock()
		p.metrics.WorkerCount.Store(int64(newWorkers))
	}

	if queueLen < p.queueSize/4 && activeWorkers <= int64(p.workers)/2 && currentWorkers > int64(p.minWorkers) {
		newWorkers := int(currentWorkers - 1)
		if newWorkers < p.minWorkers {
			newWorkers = p.minWorkers
		}
		p.mu.Lock()
		p.workers = newWorkers
		p.mu.Unlock()
		p.metrics.WorkerCount.Store(int64(newWorkers))
	}

	if latencyPerTask > 100*1e6 {
		runtime.GC()
	}
}

func (p *AdaptiveWorkerPool) Stop() {
	if !p.running.CompareAndSwap(true, false) {
		return
	}

	p.cancel()
	close(p.tasks)
	p.wg.Wait()
}

func (p *AdaptiveWorkerPool) GetMetrics() *WorkerPoolMetrics {
	return &WorkerPoolMetrics{
		TasksSubmitted:   p.metrics.TasksSubmitted.Load(),
		TasksCompleted:   p.metrics.TasksCompleted.Load(),
		TasksFailed:      p.metrics.TasksFailed.Load(),
		AvgLatency:       p.metrics.AvgLatency.Load(),
		MaxQueueSize:     p.metrics.MaxQueueSize.Load(),
		CurrentQueueSize: p.metrics.CurrentQueueSize.Load(),
		WorkerCount:      p.metrics.WorkerCount.Load(),
	}
}

func (p *AdaptiveWorkerPool) SetWorkerCount(count int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if count < p.minWorkers {
		count = p.minWorkers
	}
	if count > p.maxWorkers {
		count = p.maxWorkers
	}

	delta := count - p.workers
	if delta > 0 {
		for i := 0; i < delta; i++ {
			p.wg.Add(1)
			go p.worker()
		}
	}

	p.workers = count
	p.metrics.WorkerCount.Store(int64(count))
}

type ConcurrencyLimiter struct {
	mu          sync.Mutex
	semaphore   chan struct{}
	maxParallel int
	current     int
	waiting     int
	queue       chan struct{}
}

func NewConcurrencyLimiter(maxParallel int) *ConcurrencyLimiter {
	if maxParallel <= 0 {
		maxParallel = runtime.NumCPU()
	}

	return &ConcurrencyLimiter{
		semaphore:   make(chan struct{}, maxParallel),
		maxParallel: maxParallel,
		queue:       make(chan struct{}, maxParallel*2),
	}
}

func (l *ConcurrencyLimiter) Acquire(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case l.semaphore <- struct{}{}:
		l.mu.Lock()
		l.current++
		l.mu.Unlock()
		return nil
	default:
		l.mu.Lock()
		l.waiting++
		l.mu.Unlock()

		select {
		case <-ctx.Done():
			l.mu.Lock()
			l.waiting--
			l.mu.Unlock()
			return ctx.Err()
		case l.queue <- struct{}{}:
			l.mu.Lock()
			l.waiting--
			l.current++
			l.mu.Unlock()
			<-l.queue
			l.semaphore <- struct{}{}
			return nil
		}
	}
}

func (l *ConcurrencyLimiter) Release() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.current > 0 {
		l.current--
	}
	<-l.semaphore
}

func (l *ConcurrencyLimiter) GetStats() (current, waiting, maxParallel int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.current, l.waiting, l.maxParallel
}

type SemaphorePool struct {
	sem   chan struct{}
	mu    sync.Mutex
	stats map[string]*SemaphoreStats
}

type SemaphoreStats struct {
	AcquiredCount atomic.Int64
	ReleasedCount atomic.Int64
	WaitCount     atomic.Int64
	TotalWaitTime atomic.Int64
}

func NewSemaphorePool(size int) *SemaphorePool {
	if size <= 0 {
		size = runtime.NumCPU()
	}

	return &SemaphorePool{
		sem:   make(chan struct{}, size),
		stats: make(map[string]*SemaphoreStats),
	}
}

func (sp *SemaphorePool) Acquire(ctx context.Context, key string) error {
	if _, exists := sp.stats[key]; !exists {
		sp.mu.Lock()
		sp.stats[key] = &SemaphoreStats{}
		sp.mu.Unlock()
	}

	start := time.Now()

	select {
	case <-ctx.Done():
		sp.stats[key].WaitCount.Add(1)
		sp.stats[key].TotalWaitTime.Add(int64(time.Since(start)))
		return ctx.Err()
	case sp.sem <- struct{}{}:
		sp.stats[key].AcquiredCount.Add(1)
		sp.stats[key].TotalWaitTime.Add(int64(time.Since(start)))
		return nil
	}
}

func (sp *SemaphorePool) Release(key string) {
	sp.stats[key].ReleasedCount.Add(1)
	<-sp.sem
}

func (sp *SemaphorePool) GetStats(key string) *SemaphoreStats {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.stats[key]
}

type RateLimitedExecutor struct {
	limiter      *ConcurrencyLimiter
	rateLimiter  *AdaptiveWorkerPool
	queueSize    int
	timeout      time.Duration
}

func NewRateLimitedExecutor(maxParallel int, queueSize int, timeout time.Duration) *RateLimitedExecutor {
	return &RateLimitedExecutor{
		limiter:     NewConcurrencyLimiter(maxParallel),
		rateLimiter: NewAdaptiveWorkerPool(maxParallel/2, maxParallel, queueSize),
		queueSize:   queueSize,
		timeout:     timeout,
	}
}

func (r *RateLimitedExecutor) Execute(ctx context.Context, fn func() error) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	if err := r.limiter.Acquire(ctx); err != nil {
		return err
	}
	defer r.limiter.Release()

	return fn()
}

func (r *RateLimitedExecutor) ExecuteBatch(ctx context.Context, tasks []func() error) []error {
	results := make([]error, len(tasks))
	if len(tasks) == 0 {
		return results
	}

	r.rateLimiter.Start()
	defer r.rateLimiter.Stop()

	var wg sync.WaitGroup
	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t func() error) {
			defer wg.Done()
			results[idx] = r.Execute(ctx, t)
		}(i, task)
	}

	wg.Wait()
	return results
}

type AdaptiveBatchProcessor[T any] struct {
	pool          *AdaptiveWorkerPool
	batchSize     int
	workers       int
	enableParallel bool
	metrics       *BatchMetrics
}

type BatchMetrics struct {
	BatchesProcessed atomic.Int64
	ItemsProcessed   atomic.Int64
	Errors           atomic.Int64
	AvgBatchTime     atomic.Int64
}

func NewAdaptiveBatchProcessor[T any](batchSize, workers int) *AdaptiveBatchProcessor[T] {
	if batchSize <= 0 {
		batchSize = 100
	}
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	return &AdaptiveBatchProcessor[T]{
		batchSize:     batchSize,
		workers:       workers,
		enableParallel: true,
		metrics:       &BatchMetrics{},
	}
}

func (bp *AdaptiveBatchProcessor[T]) Process(ctx context.Context, items []T, fn func(context.Context, T) error) []error {
	if len(items) == 0 {
		return nil
	}

	results := make([]error, len(items))

	if !bp.enableParallel || len(items) < bp.batchSize {
		for i, item := range items {
			results[i] = fn(ctx, item)
		}
		return results
	}

	batches := bp.createBatches(items)
	batchResults := make([][]error, len(batches))

	pool := NewAdaptiveWorkerPool(bp.workers, bp.workers*2, len(batches))
	pool.Start()
	defer pool.Stop()

	var wg sync.WaitGroup

	for i, batch := range batches {
		batch := batch
		batchIdx := i
		wg.Add(1)

		pool.Submit(func() error {
			defer wg.Done()

			start := time.Now()
			batchResults[batchIdx] = make([]error, len(batch))

			for j, item := range batch {
				if err := fn(ctx, item); err != nil {
					batchResults[batchIdx][j] = err
					bp.metrics.Errors.Add(1)
				}
				bp.metrics.ItemsProcessed.Add(1)
			}

			bp.metrics.BatchesProcessed.Add(1)
			bp.metrics.AvgBatchTime.Store(time.Since(start).Nanoseconds() / int64(len(batch)))

			return nil
		})
	}

	wg.Wait()

	idx := 0
	for _, batchResult := range batchResults {
		for _, err := range batchResult {
			results[idx] = err
			idx++
		}
	}

	return results
}

func (bp *AdaptiveBatchProcessor[T]) createBatches(items []T) [][]T {
	var batches [][]T

	for i := 0; i < len(items); i += bp.batchSize {
		end := i + bp.batchSize
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, items[i:end])
	}

	return batches
}

func (bp *AdaptiveBatchProcessor[T]) SetBatchSize(size int) {
	if size > 0 {
		bp.batchSize = size
	}
}

func (bp *AdaptiveBatchProcessor[T]) SetWorkers(count int) {
	if count > 0 {
		bp.workers = count
	}
}

func (bp *AdaptiveBatchProcessor[T]) EnableParallel(enable bool) {
	bp.enableParallel = enable
}

func (bp *AdaptiveBatchProcessor[T]) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"batches_processed": bp.metrics.BatchesProcessed.Load(),
		"items_processed":   bp.metrics.ItemsProcessed.Load(),
		"errors":            bp.metrics.Errors.Load(),
		"avg_batch_time_ns": bp.metrics.AvgBatchTime.Load(),
	}
}

type PriorityTaskExecutor struct {
	highPriority chan func() error
	normalPriority chan func() error
	lowPriority chan func() error
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	running     atomic.Bool
	highWorkers int
	normalWorkers int
	lowWorkers int
}

func NewPriorityTaskExecutor(highWorkers, normalWorkers, lowWorkers int) *PriorityTaskExecutor {
	if highWorkers <= 0 {
		highWorkers = 2
	}
	if normalWorkers <= 0 {
		normalWorkers = runtime.NumCPU()
	}
	if lowWorkers <= 0 {
		lowWorkers = 1
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &PriorityTaskExecutor{
		highPriority:   make(chan func() error, highWorkers*10),
		normalPriority: make(chan func() error, normalWorkers*10),
		lowPriority:    make(chan func() error, lowWorkers*10),
		ctx:            ctx,
		cancel:         cancel,
		highWorkers:    highWorkers,
		normalWorkers:  normalWorkers,
		lowWorkers:     lowWorkers,
	}
}

func (p *PriorityTaskExecutor) Start() {
	if !p.running.CompareAndSwap(false, true) {
		return
	}

	for i := 0; i < p.highWorkers; i++ {
		p.wg.Add(1)
		go p.worker(p.highPriority, "high")
	}

	for i := 0; i < p.normalWorkers; i++ {
		p.wg.Add(1)
		go p.worker(p.normalPriority, "normal")
	}

	for i := 0; i < p.lowWorkers; i++ {
		p.wg.Add(1)
		go p.lowPriorityWorker()
	}
}

func (p *PriorityTaskExecutor) worker(queue chan func() error, priority string) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-queue:
			if !ok {
				return
			}
			task()
		}
	}
}

func (p *PriorityTaskExecutor) lowPriorityWorker() {
	defer p.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			select {
			case task, ok := <-p.lowPriority:
				if !ok {
					return
				}
				task()
			default:
			}
		}
	}
}

func (p *PriorityTaskExecutor) SubmitHigh(task func() error) bool {
	if !p.running.Load() {
		return false
	}

	select {
	case p.highPriority <- task:
		return true
	default:
		return false
	}
}

func (p *PriorityTaskExecutor) SubmitNormal(task func() error) bool {
	if !p.running.Load() {
		return false
	}

	select {
	case p.normalPriority <- task:
		return true
	default:
		return false
	}
}

func (p *PriorityTaskExecutor) SubmitLow(task func() error) bool {
	if !p.running.Load() {
		return false
	}

	select {
	case p.lowPriority <- task:
		return true
	default:
		return false
	}
}

func (p *PriorityTaskExecutor) Stop() {
	if !p.running.CompareAndSwap(true, false) {
		return
	}

	p.cancel()
	close(p.highPriority)
	close(p.normalPriority)
	close(p.lowPriority)
	p.wg.Wait()
}
