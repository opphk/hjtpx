package api_performance

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type ConcurrencyOptimizer struct {
	config       *ConcurrencyConfig
	workerPool   *WorkerPool
	taskQueue    chan *Task
	resultCache  *ResultCache
	stats        *ConcurrencyStats
	ctx          context.Context
	cancel       context.CancelFunc
}

type ConcurrencyConfig struct {
	WorkerCount       int
	MaxQueueSize      int
	MaxTaskDuration   time.Duration
	EnableResultCache bool
	ResultCacheSize   int
	ResultCacheTTL    time.Duration
	EnableTimeout     bool
	TaskTimeout       time.Duration
	EnableRetry       bool
	MaxRetries        int
	RetryDelay        time.Duration
	EnableBackoff     bool
	BackoffBase       time.Duration
}

var DefaultConcurrencyConfig = &ConcurrencyConfig{
	WorkerCount:       100,
	MaxQueueSize:      10000,
	MaxTaskDuration:   30 * time.Second,
	EnableResultCache: true,
	ResultCacheSize:   5000,
	ResultCacheTTL:    10 * time.Minute,
	EnableTimeout:     true,
	TaskTimeout:       10 * time.Second,
	EnableRetry:       true,
	MaxRetries:        3,
	RetryDelay:        100 * time.Millisecond,
	EnableBackoff:     true,
	BackoffBase:       100 * time.Millisecond,
}

type ConcurrencyStats struct {
	TotalTasks       atomic.Int64
	CompletedTasks   atomic.Int64
	FailedTasks      atomic.Int64
	TimeoutTasks     atomic.Int64
	RetriedTasks     atomic.Int64
	ActiveWorkers    atomic.Int64
	QueuedTasks      atomic.Int64
	AvgTaskDuration  atomic.Int64
	P50TaskDuration  atomic.Int64
	P95TaskDuration  atomic.Int64
	P99TaskDuration  atomic.Int64
	MaxQueueDepth    atomic.Int64
	TotalWaitTime    atomic.Int64
	CacheHits        atomic.Int64
	CacheMisses      atomic.Int64
}

type Task struct {
	ID        string
	Type      string
	Func      func() (interface{}, error)
	Args      []interface{}
	Priority  int
	Timeout   time.Duration
	CreatedAt time.Time
	Result    chan *TaskResult
}

type TaskResult struct {
	TaskID    string
	Value     interface{}
	Error     error
	Duration  time.Duration
	Retries   int
	FromCache  bool
}

type WorkerPool struct {
	workers    []*Worker
	taskQueue  chan *Task
	resultChan chan *TaskResult
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	config     *ConcurrencyConfig
	mu         sync.RWMutex
	stats      *WorkerPoolStats
}

type Worker struct {
	ID       int
	pool     *WorkerPool
	taskChan chan *Task
	busy     atomic.Bool
}

type WorkerPoolStats struct {
	TotalWorkers   atomic.Int64
	ActiveWorkers  atomic.Int64
	IdleWorkers    atomic.Int64
	TotalTasks     atomic.Int64
	CompletedTasks atomic.Int64
	FailedTasks    atomic.Int64
	AvgTaskTime    atomic.Int64
}

type ResultCache struct {
	cache   *sync.Map
	maxSize int
	mu      sync.RWMutex
	hits    atomic.Int64
	misses  atomic.Int64
	evictions atomic.Int64
}

type ResultCacheEntry struct {
	Value     interface{}
	ExpiresAt time.Time
}

type Semaphore struct {
	c    chan struct{}
	mu   sync.Mutex
	used int64
}

func NewConcurrencyOptimizer(config *ConcurrencyConfig) *ConcurrencyOptimizer {
	if config == nil {
		config = DefaultConcurrencyConfig
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ConcurrencyOptimizer{
		config:      config,
		workerPool:  NewWorkerPool(config),
		taskQueue:  make(chan *Task, config.MaxQueueSize),
		resultCache: NewResultCache(config.ResultCacheSize),
		stats:      &ConcurrencyStats{},
		ctx:        ctx,
		cancel:     cancel,
	}
}

func NewWorkerPool(config *ConcurrencyConfig) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	wp := &WorkerPool{
		workers:    make([]*Worker, 0, config.WorkerCount),
		taskQueue:  make(chan *Task, config.MaxQueueSize),
		resultChan: make(chan *TaskResult, config.MaxQueueSize),
		ctx:        ctx,
		cancel:     cancel,
		config:     config,
		stats:      &WorkerPoolStats{},
	}

	for i := 0; i < config.WorkerCount; i++ {
		worker := &Worker{
			ID:       i,
			pool:     wp,
			taskChan: make(chan *Task, 1),
		}
		wp.workers = append(wp.workers, worker)
		wp.stats.TotalWorkers.Add(1)
	}

	return wp
}

func (wp *WorkerPool) Start() {
	for _, worker := range wp.workers {
		wp.wg.Add(1)
		go func(w *Worker) {
			defer wp.wg.Done()
			w.run()
		}(worker)
	}
}

func (w *Worker) run() {
	for {
		select {
		case task := <-w.pool.taskQueue:
			w.busy.Store(true)
			w.pool.stats.ActiveWorkers.Add(1)

			start := time.Now()
			result := w.executeTask(task)
			duration := time.Since(start)

			result.TaskID = task.ID
			result.Duration = duration

			if task.Result != nil {
				task.Result <- result
			}
			w.pool.resultChan <- result

			w.busy.Store(false)
			w.pool.stats.ActiveWorkers.Add(-1)
			w.pool.stats.CompletedTasks.Add(1)

		case <-w.pool.ctx.Done():
			return
		}
	}
}

func (w *Worker) executeTask(task *Task) *TaskResult {
	result := &TaskResult{}

	if task.Timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
		defer cancel()

		done := make(chan struct{})
		go func() {
			val, err := task.Func()
			result.Value = val
			result.Error = err
			close(done)
		}()

		select {
		case <-done:
			return result
		case <-ctx.Done():
			result.Error = fmt.Errorf("task timeout")
			return result
		}
	}

	val, err := task.Func()
	result.Value = val
	result.Error = err
	return result
}

func (wp *WorkerPool) Submit(task *Task) {
	wp.taskQueue <- task
	wp.stats.TotalTasks.Add(1)
}

func (wp *WorkerPool) SubmitAndWait(task *Task) *TaskResult {
	task.Result = make(chan *TaskResult, 1)
	wp.taskQueue <- task
	wp.stats.TotalTasks.Add(1)

	select {
	case result := <-task.Result:
		return result
	case <-time.After(task.Timeout):
		return &TaskResult{
			TaskID: task.ID,
			Error:  fmt.Errorf("task timeout"),
		}
	}
}

func (wp *WorkerPool) Stop() {
	wp.cancel()
	wp.wg.Wait()
}

func NewResultCache(maxSize int) *ResultCache {
	return &ResultCache{
		cache:   &sync.Map{},
		maxSize: maxSize,
	}
}

func (rc *ResultCache) Get(key string) (interface{}, bool) {
	val, ok := rc.cache.Load(key)
	if !ok {
		rc.misses.Add(1)
		return nil, false
	}

	entry := val.(*ResultCacheEntry)
	if time.Now().After(entry.ExpiresAt) {
		rc.cache.Delete(key)
		rc.misses.Add(1)
		return nil, false
	}

	rc.hits.Add(1)
	return entry.Value, true
}

func (rc *ResultCache) Set(key string, value interface{}) {
	if rc.getSize() >= rc.maxSize {
		rc.evictOldest()
	}

	entry := &ResultCacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	rc.cache.Store(key, entry)
}

func (rc *ResultCache) Delete(key string) {
	rc.cache.Delete(key)
}

func (rc *ResultCache) Clear() {
	rc.cache = &sync.Map{}
}

func (rc *ResultCache) getSize() int {
	size := 0
	rc.cache.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}

func (rc *ResultCache) evictOldest() {
	var oldestKey string

	rc.cache.Range(func(key, value interface{}) bool {
		if oldestKey == "" {
			oldestKey = key.(string)
		}
		return true
	})

	if oldestKey != "" {
		rc.cache.Delete(oldestKey)
		rc.evictions.Add(1)
	}
}

func (o *ConcurrencyOptimizer) Start() {
	o.workerPool.Start()
	go o.processResults()
}

func (o *ConcurrencyOptimizer) Stop() {
	o.cancel()
	o.workerPool.Stop()
}

func (o *ConcurrencyOptimizer) processResults() {
	for {
		select {
		case result := <-o.workerPool.resultChan:
			o.processResult(result)
		case <-o.ctx.Done():
			return
		}
	}
}

func (o *ConcurrencyOptimizer) processResult(result *TaskResult) {
	if result.Error != nil {
		o.stats.FailedTasks.Add(1)
	} else {
		o.stats.CompletedTasks.Add(1)
	}

	if o.config.EnableResultCache {
		o.resultCache.Set(result.TaskID, result.Value)
	}
}

func (o *ConcurrencyOptimizer) Execute(task *Task) *TaskResult {
	o.stats.TotalTasks.Add(1)

	cacheKey := o.generateCacheKey(task)
	if o.config.EnableResultCache {
		if val, ok := o.resultCache.Get(cacheKey); ok {
			o.stats.CacheHits.Add(1)
			return &TaskResult{
				TaskID:   task.ID,
				Value:    val,
				FromCache: true,
			}
		}
		o.stats.CacheMisses.Add(1)
	}

	result := o.executeWithRetry(task)

	if result.Error == nil && o.config.EnableResultCache {
		o.resultCache.Set(cacheKey, result.Value)
	}

	return result
}

func (o *ConcurrencyOptimizer) executeWithRetry(task *Task) *TaskResult {
	var lastErr error
	retries := 0

	for i := 0; i <= o.config.MaxRetries; i++ {
		if i > 0 {
			o.stats.RetriedTasks.Add(1)
			retries++

			delay := o.config.RetryDelay
			if o.config.EnableBackoff {
				multiplier := 1 << uint(i-1)
				delay = o.config.BackoffBase * time.Duration(multiplier)
			}

			time.Sleep(delay)
		}

		result := o.executeTask(task)
		if result.Error == nil {
			result.Retries = retries
			return result
		}

		lastErr = result.Error

		if !o.isRetryableError(result.Error) {
			break
		}
	}

	return &TaskResult{
		TaskID:   task.ID,
		Error:    lastErr,
		Retries:  retries,
	}
}

func (o *ConcurrencyOptimizer) executeTask(task *Task) *TaskResult {
	start := time.Now()

	var ctx context.Context
	var cancel context.CancelFunc

	if o.config.EnableTimeout {
		timeout := o.config.TaskTimeout
		if task.Timeout > 0 {
			timeout = task.Timeout
		}
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	done := make(chan struct{})
	var result *TaskResult

	go func() {
		value, err := task.Func()
		result = &TaskResult{
			TaskID: task.ID,
			Value:  value,
			Error:  err,
		}
		close(done)
	}()

	select {
	case <-done:
		result.Duration = time.Since(start)
		return result
	case <-ctx.Done():
		o.stats.TimeoutTasks.Add(1)
		return &TaskResult{
			TaskID:   task.ID,
			Error:    fmt.Errorf("task timeout"),
			Duration: time.Since(start),
		}
	}
}

func (o *ConcurrencyOptimizer) isRetryableError(err error) bool {
	return err != nil
}

func (o *ConcurrencyOptimizer) generateCacheKey(task *Task) string {
	key := task.ID
	for _, arg := range task.Args {
		key += fmt.Sprintf(":%v", arg)
	}
	return key
}

func (o *ConcurrencyOptimizer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_tasks":        o.stats.TotalTasks.Load(),
		"completed_tasks":   o.stats.CompletedTasks.Load(),
		"failed_tasks":      o.stats.FailedTasks.Load(),
		"timeout_tasks":     o.stats.TimeoutTasks.Load(),
		"retried_tasks":     o.stats.RetriedTasks.Load(),
		"active_workers":    o.stats.ActiveWorkers.Load(),
		"queued_tasks":      o.stats.QueuedTasks.Load(),
		"avg_task_duration": time.Duration(o.stats.AvgTaskDuration.Load()).String(),
		"p50_task_duration": time.Duration(o.stats.P50TaskDuration.Load()).String(),
		"p95_task_duration": time.Duration(o.stats.P95TaskDuration.Load()).String(),
		"p99_task_duration": time.Duration(o.stats.P99TaskDuration.Load()).String(),
		"max_queue_depth":   o.stats.MaxQueueDepth.Load(),
		"total_wait_time":   time.Duration(o.stats.TotalWaitTime.Load()).String(),
		"cache_hits":        o.stats.CacheHits.Load(),
		"cache_misses":      o.stats.CacheMisses.Load(),
	}
}

type SemaphoreLimiter struct {
	sem *Semaphore
	mu  sync.Mutex
}

func NewSemaphoreLimiter(n int) *SemaphoreLimiter {
	return &SemaphoreLimiter{
		sem: &Semaphore{
			c: make(chan struct{}, n),
		},
	}
}

func (s *SemaphoreLimiter) Acquire() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sem.c <- struct{}{}
	atomic.AddInt64(&s.sem.used, 1)
}

func (s *SemaphoreLimiter) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.sem.c:
		atomic.AddInt64(&s.sem.used, -1)
	default:
	}
}

func (s *SemaphoreLimiter) TryAcquire() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case s.sem.c <- struct{}{}:
		atomic.AddInt64(&s.sem.used, 1)
		return true
	default:
		return false
	}
}

func (s *SemaphoreLimiter) GetUsed() int64 {
	return atomic.LoadInt64(&s.sem.used)
}

func (s *SemaphoreLimiter) GetCapacity() int {
	return cap(s.sem.c)
}

type RateLimiter struct {
	mu          sync.Mutex
	tokens      float64
	maxTokens   float64
	 refillRate float64
	lastRefill  time.Time
}

func NewRateLimiter(maxTokens, refillRate float64) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}

	return false
}

func (rl *RateLimiter) AllowN(n int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	if int(rl.tokens) >= n {
		rl.tokens -= float64(n)
		return true
	}

	return false
}

func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()

	tokensToAdd := elapsed * rl.refillRate
	rl.tokens += tokensToAdd

	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}

	rl.lastRefill = now
}

func (rl *RateLimiter) GetTokens() float64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()
	return rl.tokens
}

type AdaptivePool struct {
	mu            sync.RWMutex
	minWorkers    int
	maxWorkers    int
	currentWorkers int
	activeWorkers int
	queueLength   int
	scaleUpThreshold int
	scaleDownThreshold int
	taskQueue    chan func() interface{}
	resultCache  *ResultCache
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

func NewAdaptivePool(minWorkers, maxWorkers int) *AdaptivePool {
	ctx, cancel := context.WithCancel(context.Background())

	return &AdaptivePool{
		minWorkers:       minWorkers,
		maxWorkers:       maxWorkers,
		currentWorkers:   minWorkers,
		scaleUpThreshold: 100,
		scaleDownThreshold: 10,
		taskQueue:       make(chan func() interface{}, 10000),
		resultCache:     NewResultCache(5000),
		ctx:             ctx,
		cancel:          cancel,
	}
}

func (ap *AdaptivePool) Start() {
	for i := 0; i < ap.currentWorkers; i++ {
		ap.wg.Add(1)
		go ap.worker()
	}
}

func (ap *AdaptivePool) worker() {
	defer ap.wg.Done()

	for {
		select {
		case task := <-ap.taskQueue:
			ap.activeWorkers++
			task()
			ap.activeWorkers--

			ap.checkScale()
		case <-ap.ctx.Done():
			return
		}
	}
}

func (ap *AdaptivePool) checkScale() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.activeWorkers >= ap.currentWorkers && ap.currentWorkers < ap.maxWorkers {
		ap.currentWorkers++
		ap.wg.Add(1)
		go ap.worker()
	} else if ap.activeWorkers == 0 && ap.currentWorkers > ap.minWorkers {
		ap.currentWorkers--
	}
}

func (ap *AdaptivePool) Submit(task func() interface{}) {
	ap.taskQueue <- task
}

func (ap *AdaptivePool) Stop() {
	ap.cancel()
	ap.wg.Wait()
}

func (ap *AdaptivePool) GetStats() map[string]interface{} {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	return map[string]interface{}{
		"min_workers":     ap.minWorkers,
		"max_workers":     ap.maxWorkers,
		"current_workers": ap.currentWorkers,
		"active_workers":  ap.activeWorkers,
		"queue_length":    len(ap.taskQueue),
	}
}
