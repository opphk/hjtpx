package service

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type Task func() error

type PoolStats struct {
	ActiveWorkers  int32
	TotalTasks     int64
	CompletedTasks int64
	FailedTasks    int64
	PendingTasks   int32
	QueueCapacity  int
}

type GoroutinePool struct {
	workers    int
	taskQueue  chan Task
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	running    atomic.Bool
	mu         sync.RWMutex
	stats      PoolStats
	spawnCond  *sync.Cond
	shouldStop atomic.Bool
	autoScale  atomic.Bool
	minWorkers int
	maxWorkers int
}

func NewGoroutinePool(workers, queueSize int) *GoroutinePool {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if queueSize <= 0 {
		queueSize = workers * 10
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &GoroutinePool{
		workers:   workers,
		taskQueue: make(chan Task, queueSize),
		ctx:       ctx,
		cancel:    cancel,
		minWorkers: workers,
		maxWorkers: workers * 4,
		stats: PoolStats{
			QueueCapacity: queueSize,
		},
	}
	pool.spawnCond = sync.NewCond(&pool.mu)

	return pool
}

func (p *GoroutinePool) Start() {
	if p.running.Load() {
		return
	}
	p.running.Store(true)
	p.shouldStop.Store(false)

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

func (p *GoroutinePool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.taskQueue:
			if !ok {
				return
			}
			p.executeTask(task)
		}
	}
}

func (p *GoroutinePool) executeTask(task Task) {
	atomic.AddInt32(&p.stats.ActiveWorkers, 1)
	atomic.AddInt64(&p.stats.TotalTasks, 1)
	defer atomic.AddInt32(&p.stats.ActiveWorkers, -1)

	if err := task(); err != nil {
		atomic.AddInt64(&p.stats.FailedTasks, 1)
	} else {
		atomic.AddInt64(&p.stats.CompletedTasks, 1)
	}
}

func (p *GoroutinePool) Submit(task Task) bool {
	if p.shouldStop.Load() {
		return false
	}

	select {
	case p.taskQueue <- task:
		atomic.AddInt32(&p.stats.PendingTasks, 1)
		return true
	default:
		return false
	}
}

func (p *GoroutinePool) SubmitBlocking(task Task) error {
	if p.shouldStop.Load() {
		return fmt.Errorf("pool is stopped")
	}

	select {
	case p.taskQueue <- task:
		atomic.AddInt32(&p.stats.PendingTasks, 1)
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	}
}

func (p *GoroutinePool) SubmitWithTimeout(task Task, timeout time.Duration) bool {
	if p.shouldStop.Load() {
		return false
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case p.taskQueue <- task:
		atomic.AddInt32(&p.stats.PendingTasks, 1)
		return true
	case <-timer.C:
		return false
	}
}

func (p *GoroutinePool) SubmitAndWait(task Task) error {
	if p.shouldStop.Load() {
		return fmt.Errorf("pool is stopped")
	}

	done := make(chan error, 1)

	select {
	case p.taskQueue <- func() error {
		err := task()
		done <- err
		return nil
	}:
		atomic.AddInt32(&p.stats.PendingTasks, 1)
		return <-done
	case <-p.ctx.Done():
		return p.ctx.Err()
	}
}

func (p *GoroutinePool) Scale(workers int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if workers < p.minWorkers {
		workers = p.minWorkers
	}
	if workers > p.maxWorkers {
		workers = p.maxWorkers
	}

	current := p.workers
	p.workers = workers

	diff := workers - current
	if diff > 0 {
		for i := 0; i < diff; i++ {
			p.wg.Add(1)
			go p.worker(current + i)
		}
	}
}

func (p *GoroutinePool) AutoScale(enabled bool, minWorkers, maxWorkers int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.autoScale.Store(enabled)
	if minWorkers > 0 {
		p.minWorkers = minWorkers
	}
	if maxWorkers > 0 {
		p.maxWorkers = maxWorkers
	}
}

func (p *GoroutinePool) AdjustScale() {
	if !p.autoScale.Load() {
		return
	}

	queueLen := int32(len(p.taskQueue))
	activeWorkers := atomic.LoadInt32(&p.stats.ActiveWorkers)

	p.mu.Lock()
	defer p.mu.Unlock()

	if queueLen > int32(p.workers) && p.workers < p.maxWorkers {
		newWorkers := min(p.workers*2, p.maxWorkers)
		for i := p.workers; i < newWorkers; i++ {
			p.wg.Add(1)
			go p.worker(i)
		}
		p.workers = newWorkers
	}

	if queueLen == 0 && activeWorkers == 0 && p.workers > p.minWorkers {
		reduceCount := min(p.workers/2, p.workers-p.minWorkers)
		p.workers -= reduceCount
	}
}

func (p *GoroutinePool) Stop() {
	if !p.running.Load() {
		return
	}

	p.shouldStop.Store(true)
	p.running.Store(false)
	p.cancel()

	close(p.taskQueue)
	p.wg.Wait()
}

func (p *GoroutinePool) Await() {
	p.wg.Wait()
}

func (p *GoroutinePool) AwaitWithTimeout(timeout time.Duration) bool {
	done := make(chan struct{})

	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (p *GoroutinePool) Stats() PoolStats {
	stats := p.stats
	stats.ActiveWorkers = atomic.LoadInt32(&p.stats.ActiveWorkers)
	stats.TotalTasks = atomic.LoadInt64(&p.stats.TotalTasks)
	stats.CompletedTasks = atomic.LoadInt64(&p.stats.CompletedTasks)
	stats.FailedTasks = atomic.LoadInt64(&p.stats.FailedTasks)
	stats.PendingTasks = atomic.LoadInt32(&p.stats.PendingTasks)
	stats.QueueCapacity = p.stats.QueueCapacity
	return stats
}

func (p *GoroutinePool) IsRunning() bool {
	return p.running.Load()
}

func (p *GoroutinePool) QueueLength() int {
	return len(p.taskQueue)
}

func (p *GoroutinePool) QueueCapacity() int {
	return p.stats.QueueCapacity
}

func (p *GoroutinePool) WorkerCount() int {
	return p.workers
}

func (p *GoroutinePool) Utilization() float64 {
	active := atomic.LoadInt32(&p.stats.ActiveWorkers)
	return float64(active) / float64(p.workers)
}

func (p *GoroutinePool) TryDecrementPending() {
	pending := atomic.LoadInt32(&p.stats.PendingTasks)
	if pending > 0 {
		atomic.AddInt32(&p.stats.PendingTasks, -1)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type Semaphore struct {
	mu       sync.Mutex
	cond     *sync.Cond
	value    int
	maxValue int
}

func NewSemaphore(maxValue int) *Semaphore {
	if maxValue <= 0 {
		maxValue = 1
	}
	s := &Semaphore{
		value:    maxValue,
		maxValue: maxValue,
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *Semaphore) Acquire() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for s.value <= 0 {
		s.cond.Wait()
	}
	s.value--
}

func (s *Semaphore) TryAcquire() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.value <= 0 {
		return false
	}
	s.value--
	return true
}

func (s *Semaphore) TryAcquireWithTimeout(timeout time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.value <= 0 {
		return false
	}
	s.value--
	return true
}

func (s *Semaphore) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.value < s.maxValue {
		s.value++
		s.cond.Signal()
	}
}

func (s *Semaphore) WithAcquire(fn func()) {
	s.Acquire()
	defer s.Release()
	fn()
}

func (s *Semaphore) Available() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.value
}

type OnceRunner struct {
	mu       sync.Mutex
	ran      bool
	panicked bool
}

func NewOnceRunner() *OnceRunner {
	return &OnceRunner{}
}

func (o *OnceRunner) Run(fn func()) (ran bool, err error) {
	o.mu.Lock()
	if o.ran {
		o.mu.Unlock()
		return false, nil
	}
	o.ran = true
	o.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			o.mu.Lock()
			o.ran = false
			o.mu.Unlock()
			o.panicked = true
			panic(r)
		}
	}()

	fn()
	return true, nil
}

type OnceRunnerWithResult struct {
	mu       sync.Mutex
	ran      bool
	panicked bool
}

func NewOnceRunnerWithResult() *OnceRunnerWithResult {
	return &OnceRunnerWithResult{}
}

func (o *OnceRunnerWithResult) Run(fn func() error) (ran bool, err error) {
	o.mu.Lock()
	if o.ran {
		o.mu.Unlock()
		return false, nil
	}
	o.ran = true
	o.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			o.mu.Lock()
			o.ran = false
			o.mu.Unlock()
			o.panicked = true
		}
	}()

	err = fn()
	if err != nil {
		o.mu.Lock()
		o.ran = false
		o.mu.Unlock()
	}
	return true, err
}

type BatchExecutor struct {
	pool      *GoroutinePool
	batchSize int
	semaphore *Semaphore
}

func NewBatchExecutor(workers, batchSize int) *BatchExecutor {
	if batchSize <= 0 {
		batchSize = 100
	}

	pool := NewGoroutinePool(workers, batchSize*2)
	pool.Start()

	return &BatchExecutor{
		pool:      pool,
		batchSize: batchSize,
		semaphore: NewSemaphore(batchSize),
	}
}

func (be *BatchExecutor) ExecuteBatch(items []interface{}, handler func(item interface{}) error) []error {
	if len(items) == 0 {
		return nil
	}

	results := make([]error, len(items))
	itemChan := make(chan interface{}, len(items))
	errorChan := make(chan struct {
		index int
		err   error
	}, len(items))

	for _, item := range items {
		itemChan <- item
	}
	close(itemChan)

	var wg sync.WaitGroup

	for i, item := range items {
		wg.Add(1)
		idx := i
		it := item

		be.pool.Submit(func() error {
			defer wg.Done()

			be.semaphore.Acquire()
			defer be.semaphore.Release()

			err := handler(it)
			if err != nil {
				errorChan <- struct {
					index int
					err   error
				}{index: idx, err: err}
			}
			return nil
		})
	}

	go func() {
		wg.Wait()
		close(errorChan)
	}()

	for e := range errorChan {
		results[e.index] = e.err
	}

	return results
}

func (be *BatchExecutor) ExecuteBatchAsync(items []interface{}, handler func(item interface{}) error, resultChan chan<- []error) {
	if len(items) == 0 {
		resultChan <- nil
		return
	}

	go func() {
		results := be.ExecuteBatch(items, handler)
		resultChan <- results
	}()
}

func (be *BatchExecutor) Stop() {
	be.pool.Stop()
}

func (be *BatchExecutor) Stats() PoolStats {
	return be.pool.Stats()
}

type PoolMetrics struct {
	TotalTasks      int64
	CompletedTasks int64
	FailedTasks     int64
	ActiveWorkers  int32
	QueueLength    int
	Utilization    float64
}

func (p *GoroutinePool) Metrics() PoolMetrics {
	return PoolMetrics{
		TotalTasks:      atomic.LoadInt64(&p.stats.TotalTasks),
		CompletedTasks:  atomic.LoadInt64(&p.stats.CompletedTasks),
		FailedTasks:     atomic.LoadInt64(&p.stats.FailedTasks),
		ActiveWorkers:   atomic.LoadInt32(&p.stats.ActiveWorkers),
		QueueLength:     len(p.taskQueue),
		Utilization:     p.Utilization(),
	}
}

var defaultPool *GoroutinePool
var defaultPoolOnce sync.Once

func GetDefaultPool() *GoroutinePool {
	defaultPoolOnce.Do(func() {
		defaultPool = NewGoroutinePool(runtime.NumCPU()*2, 10000)
		defaultPool.Start()
	})
	return defaultPool
}

func SubmitToDefault(task Task) bool {
	return GetDefaultPool().Submit(task)
}

func SubmitAndWaitDefault(task Task) error {
	return GetDefaultPool().SubmitAndWait(task)
}
