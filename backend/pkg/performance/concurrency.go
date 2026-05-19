package performance

import (
	"context"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type ConcurrencyManager struct {
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.RWMutex
	isRunning      bool
	workerPool     *WorkerPool
	taskQueue      chan Task
	workers        int
	maxWorkers     int
	minWorkers     int
	stats          *ConcurrencyStats
	rateLimiter    *RateLimiter
	semaphore      *Semaphore
}

type Task struct {
	fn       func() error
	priority int
}

type ConcurrencyStats struct {
	TotalTasks       atomic.Int64
	CompletedTasks   atomic.Int64
	FailedTasks      atomic.Int64
	ActiveWorkers    atomic.Int64
	QueuedTasks      atomic.Int64
	AverageLatency   atomic.Int64
	Throughput       atomic.Int64
	LastUpdate       atomic.Value
}

type WorkerPool struct {
	tasks    chan Task
	workers  []*Worker
	wg       sync.WaitGroup
}

type Worker struct {
	id        int
	pool      *WorkerPool
	taskChan  chan Task
	ctx       context.Context
	cancel    context.CancelFunc
}

type RateLimiter struct {
	mu           sync.RWMutex
	tokens       int
	maxTokens    int
	refillRate   time.Duration
	lastRefill   time.Time
}

type Semaphore struct {
	permits chan struct{}
}

func NewConcurrencyManager() *ConcurrencyManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	numCPU := runtime.NumCPU()
	minWorkers := numCPU * 2
	maxWorkers := numCPU * 20
	
	return &ConcurrencyManager{
		ctx:         ctx,
		cancel:      cancel,
		workerPool:  NewWorkerPool(minWorkers, maxWorkers),
		taskQueue:   make(chan Task, 10000),
		workers:     minWorkers,
		maxWorkers:  maxWorkers,
		minWorkers:  minWorkers,
		stats:       &ConcurrencyStats{},
		rateLimiter: NewRateLimiter(10000, 100*time.Millisecond),
		semaphore:   NewSemaphore(1000),
	}
}

func NewWorkerPool(min, max int) *WorkerPool {
	return &WorkerPool{
		tasks:   make(chan Task, 10000),
		workers: make([]*Worker, 0, max),
	}
}

func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:       maxTokens,
		maxTokens:    maxTokens,
		refillRate:   refillRate,
		lastRefill:   time.Now(),
	}
}

func NewSemaphore(size int) *Semaphore {
	return &Semaphore{
		permits: make(chan struct{}, size),
	}
}

func (c *ConcurrencyManager) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning {
		return nil
	}

	c.isRunning = true

	for i := 0; i < c.workers; i++ {
		c.workerPool.AddWorker()
	}

	go c.processTasks()
	go c.monitorWorkers()
	go c.adaptiveScaling()

	log.Println("[ConcurrencyManager] Started successfully")
	return nil
}

func (c *ConcurrencyManager) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning {
		return
	}

	c.cancel()
	c.isRunning = false
	c.workerPool.Stop()

	log.Println("[ConcurrencyManager] Stopped")
}

func (c *ConcurrencyManager) Submit(task func() error) bool {
	return c.SubmitWithPriority(task, 0)
}

func (c *ConcurrencyManager) SubmitWithPriority(task func() error, priority int) bool {
	c.stats.TotalTasks.Add(1)
	c.stats.QueuedTasks.Add(1)

	select {
	case c.taskQueue <- Task{fn: task, priority: priority}:
		return true
	default:
		return false
	}
}

func (c *ConcurrencyManager) processTasks() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case task := <-c.taskQueue:
			c.stats.QueuedTasks.Add(-1)
			
			if !c.rateLimiter.Acquire() {
				time.Sleep(10 * time.Millisecond)
				c.taskQueue <- task
				continue
			}

			if !c.semaphore.Acquire() {
				c.rateLimiter.Release()
				c.taskQueue <- task
				continue
			}

			go func(t Task) {
				defer c.semaphore.Release()
				defer c.rateLimiter.Release()

				start := time.Now()
				if err := t.fn(); err != nil {
					c.stats.FailedTasks.Add(1)
				} else {
					c.stats.CompletedTasks.Add(1)
				}
				
				latency := time.Since(start).Nanoseconds()
				c.updateLatency(latency)
			}(task)
		}
	}
}

func (c *ConcurrencyManager) monitorWorkers() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.stats.ActiveWorkers.Store(int64(len(c.workerPool.workers)))
			c.stats.LastUpdate.Store(time.Now())
		}
	}
}

func (c *ConcurrencyManager) adaptiveScaling() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.scaleWorkers()
		}
	}
}

func (c *ConcurrencyManager) scaleWorkers() {
	c.mu.Lock()
	defer c.mu.Unlock()

	queueSize := len(c.taskQueue)
	currentWorkers := len(c.workerPool.workers)

	if queueSize > 1000 && currentWorkers < c.maxWorkers {
		newWorkers := min(currentWorkers*2, c.maxWorkers)
		for i := currentWorkers; i < newWorkers; i++ {
			c.workerPool.AddWorker()
		}
		log.Printf("[ConcurrencyManager] Scaled up to %d workers", newWorkers)
	} else if queueSize < 100 && currentWorkers > c.minWorkers {
		newWorkers := max(currentWorkers/2, c.minWorkers)
		for i := currentWorkers; i > newWorkers; i-- {
			c.workerPool.RemoveWorker()
		}
		log.Printf("[ConcurrencyManager] Scaled down to %d workers", newWorkers)
	}
}

func (c *ConcurrencyManager) updateLatency(latency int64) {
	oldLatency := c.stats.AverageLatency.Load()
	count := c.stats.CompletedTasks.Load()
	if count > 0 {
		newLatency := (oldLatency*(count-1) + latency) / count
		c.stats.AverageLatency.Store(newLatency)
	}
}

func (c *ConcurrencyManager) LimitGoroutines(limit int) {
	c.semaphore = NewSemaphore(limit)
}

func (c *ConcurrencyManager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_tasks":       c.stats.TotalTasks.Load(),
		"completed_tasks":   c.stats.CompletedTasks.Load(),
		"failed_tasks":      c.stats.FailedTasks.Load(),
		"active_workers":    c.stats.ActiveWorkers.Load(),
		"queued_tasks":      c.stats.QueuedTasks.Load(),
		"average_latency":   c.stats.AverageLatency.Load(),
		"workers":           len(c.workerPool.workers),
	}
}

func (w *WorkerPool) AddWorker() {
	ctx, cancel := context.WithCancel(context.Background())
	worker := &Worker{
		id:       len(w.workers),
		pool:     w,
		taskChan: w.tasks,
		ctx:      ctx,
		cancel:   cancel,
	}
	
	w.workers = append(w.workers, worker)
	w.wg.Add(1)
	
	go worker.Run()
}

func (w *WorkerPool) RemoveWorker() {
	if len(w.workers) > 0 {
		worker := w.workers[len(w.workers)-1]
		worker.cancel()
		w.workers = w.workers[:len(w.workers)-1]
	}
}

func (w *WorkerPool) Stop() {
	for _, worker := range w.workers {
		worker.cancel()
	}
	w.wg.Wait()
}

func (w *Worker) Run() {
	defer w.pool.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		case task := <-w.taskChan:
			if task.fn != nil {
				task.fn()
			}
		}
	}
}

func (r *RateLimiter) Acquire() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()

	if r.tokens > 0 {
		r.tokens--
		return true
	}
	return false
}

func (r *RateLimiter) Release() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.tokens < r.maxTokens {
		r.tokens++
	}
}

func (r *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	tokensToAdd := int(elapsed / r.refillRate)
	
	if tokensToAdd > 0 {
		r.tokens = min(r.tokens+tokensToAdd, r.maxTokens)
		r.lastRefill = now
	}
}

func (s *Semaphore) Acquire() bool {
	select {
	case s.permits <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *Semaphore) Release() {
	select {
	case <-s.permits:
	default:
	}
}

func (s *Semaphore) AcquireWithContext(ctx context.Context) error {
	select {
	case s.permits <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
