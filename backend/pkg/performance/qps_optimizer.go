package performance

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const (
	TargetQPSv2 = 20000
	BatchSize   = 100
)

type QPSOptimizer struct {
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	isRunning bool

	// 请求批处理
	batchProcessor *BatchProcessor

	// 工作池
	workerPool *HighPerformanceWorkerPool

	// 算法优化
	algorithmOptimizer *AlgorithmOptimizer

	// 统计信息
	stats *QPSStats
}

type BatchProcessor struct {
	mu           sync.Mutex
	batchQueue   []BatchRequest
	batchSize    int
	flushTimeout time.Duration
	flushChan    chan struct{}
	processor    func([]BatchRequest) error
}

type BatchRequest struct {
	ID      string
	Payload interface{}
	Result  chan interface{}
}

type HighPerformanceWorkerPool struct {
	workers     []*WorkerV2
	taskQueue   chan TaskV2
	wg          sync.WaitGroup
	workerCount int
}

type WorkerV2 struct {
	id       int
	pool     *HighPerformanceWorkerPool
	taskChan chan TaskV2
	ctx      context.Context
	cancel   context.CancelFunc
}

type TaskV2 struct {
	fn       func() error
	priority int
}

type AlgorithmOptimizer struct {
	mu           sync.RWMutex
	cache        map[string]interface{}
	accessCount  map[string]int64
	lruList      []string
	maxCacheSize int
}

type QPSStats struct {
	CurrentQPS      atomic.Int64
	PeakQPS         atomic.Int64
	TotalRequests   atomic.Int64
	BatchProcessed  atomic.Int64
	WorkerUtilization atomic.Int64
	CacheHits       atomic.Int64
	CacheMisses     atomic.Int64
	LastUpdate      atomic.Value
}

func NewQPSOptimizer() *QPSOptimizer {
	ctx, cancel := context.WithCancel(context.Background())
	return &QPSOptimizer{
		ctx:                ctx,
		cancel:             cancel,
		batchProcessor:     NewBatchProcessor(BatchSize, 10*time.Millisecond),
		workerPool:         NewHighPerformanceWorkerPool(500),
		algorithmOptimizer: NewAlgorithmOptimizer(10000),
		stats:              &QPSStats{},
	}
}

func NewBatchProcessor(batchSize int, flushTimeout time.Duration) *BatchProcessor {
	return &BatchProcessor{
		batchQueue:   make([]BatchRequest, 0, batchSize),
		batchSize:    batchSize,
		flushTimeout: flushTimeout,
		flushChan:    make(chan struct{}, 1),
	}
}

func NewHighPerformanceWorkerPool(workerCount int) *HighPerformanceWorkerPool {
	pool := &HighPerformanceWorkerPool{
		taskQueue:   make(chan TaskV2, 100000),
		workerCount: workerCount,
		workers:     make([]*WorkerV2, 0, workerCount),
	}
	return pool
}

func NewAlgorithmOptimizer(maxCacheSize int) *AlgorithmOptimizer {
	return &AlgorithmOptimizer{
		cache:        make(map[string]interface{}, maxCacheSize),
		accessCount:  make(map[string]int64, maxCacheSize),
		lruList:      make([]string, 0, maxCacheSize),
		maxCacheSize: maxCacheSize,
	}
}

func (q *QPSOptimizer) Start(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.isRunning {
		return nil
	}
	q.isRunning = true

	q.workerPool.Start(q.ctx)
	go q.batchProcessor.run(q.ctx)
	go q.monitorQPS()
	go q.optimizeWorkerPool()

	log.Println("[QPSOptimizer] Started successfully with target QPS:", TargetQPSv2)
	return nil
}

func (q *QPSOptimizer) Stop() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.isRunning {
		return
	}
	q.isRunning = false
	q.cancel()
	q.workerPool.Stop()

	log.Println("[QPSOptimizer] Stopped")
}

func (q *QPSOptimizer) SubmitRequest(req BatchRequest) {
	q.stats.TotalRequests.Add(1)
	q.batchProcessor.submit(req)
}

func (q *QPSOptimizer) SubmitTask(task func() error, priority int) bool {
	select {
	case q.workerPool.taskQueue <- TaskV2{fn: task, priority: priority}:
		return true
	default:
		return false
	}
}

func (q *QPSOptimizer) CacheResult(key string, value interface{}) {
	q.algorithmOptimizer.set(key, value)
}

func (q *QPSOptimizer) GetCachedResult(key string) (interface{}, bool) {
	value, ok := q.algorithmOptimizer.get(key)
	if ok {
		q.stats.CacheHits.Add(1)
	} else {
		q.stats.CacheMisses.Add(1)
	}
	return value, ok
}

func (q *QPSOptimizer) SetBatchProcessor(processor func([]BatchRequest) error) {
	q.batchProcessor.processor = processor
}

func (q *QPSOptimizer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"current_qps":          q.stats.CurrentQPS.Load(),
		"peak_qps":             q.stats.PeakQPS.Load(),
		"target_qps":           TargetQPSv2,
		"total_requests":       q.stats.TotalRequests.Load(),
		"batch_processed":      q.stats.BatchProcessed.Load(),
		"worker_utilization":   q.stats.WorkerUtilization.Load(),
		"cache_hits":           q.stats.CacheHits.Load(),
		"cache_misses":         q.stats.CacheMisses.Load(),
		"last_update":          q.stats.LastUpdate.Load(),
	}
}

func (q *QPSOptimizer) monitorQPS() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	requestWindow := make([]int64, 0, 5)
	lastRequestCount := int64(0)

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			currentRequests := q.stats.TotalRequests.Load()
			qps := currentRequests - lastRequestCount
			lastRequestCount = currentRequests

			requestWindow = append(requestWindow, qps)
			if len(requestWindow) > 5 {
				requestWindow = requestWindow[1:]
			}

			var avgQPS int64
			for _, r := range requestWindow {
				avgQPS += r
			}
			avgQPS /= int64(len(requestWindow))

			q.stats.CurrentQPS.Store(avgQPS)
			if avgQPS > q.stats.PeakQPS.Load() {
				q.stats.PeakQPS.Store(avgQPS)
			}
			q.stats.LastUpdate.Store(time.Now())

			if avgQPS > TargetQPSv2 {
				log.Printf("[QPSOptimizer] QPS target achieved: %d", avgQPS)
			}
		}
	}
}

func (q *QPSOptimizer) optimizeWorkerPool() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			queueSize := len(q.workerPool.taskQueue)
			q.stats.WorkerUtilization.Store(int64(queueSize) * 100 / int64(cap(q.workerPool.taskQueue)))
		}
	}
}

func (bp *BatchProcessor) submit(req BatchRequest) {
	bp.mu.Lock()
	bp.batchQueue = append(bp.batchQueue, req)
	if len(bp.batchQueue) >= bp.batchSize {
		bp.mu.Unlock()
		select {
		case bp.flushChan <- struct{}{}:
		default:
		}
		return
	}
	bp.mu.Unlock()
}

func (bp *BatchProcessor) run(ctx context.Context) {
	ticker := time.NewTicker(bp.flushTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-bp.flushChan:
			bp.flush()
		case <-ticker.C:
			bp.flush()
		}
	}
}

func (bp *BatchProcessor) flush() {
	bp.mu.Lock()
	if len(bp.batchQueue) == 0 {
		bp.mu.Unlock()
		return
	}

	batch := make([]BatchRequest, len(bp.batchQueue))
	copy(batch, bp.batchQueue)
	bp.batchQueue = bp.batchQueue[:0]
	bp.mu.Unlock()

	if bp.processor != nil {
		bp.processor(batch)
	}
}

func (wp *HighPerformanceWorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.workerCount; i++ {
		workerCtx, cancel := context.WithCancel(ctx)
		worker := &WorkerV2{
			id:       i,
			pool:     wp,
			taskChan: wp.taskQueue,
			ctx:      workerCtx,
			cancel:   cancel,
		}
		wp.workers = append(wp.workers, worker)
		wp.wg.Add(1)
		go worker.run()
	}
}

func (wp *HighPerformanceWorkerPool) Stop() {
	for _, worker := range wp.workers {
		worker.cancel()
	}
	wp.wg.Wait()
}

func (w *WorkerV2) run() {
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

func (ao *AlgorithmOptimizer) get(key string) (interface{}, bool) {
	ao.mu.RLock()
	value, ok := ao.cache[key]
	if ok {
		ao.accessCount[key]++
	}
	ao.mu.RUnlock()
	return value, ok
}

func (ao *AlgorithmOptimizer) set(key string, value interface{}) {
	ao.mu.Lock()
	defer ao.mu.Unlock()

	if _, exists := ao.cache[key]; exists {
		ao.cache[key] = value
		return
	}

	if len(ao.cache) >= ao.maxCacheSize {
		ao.evictLRU()
	}

	ao.cache[key] = value
	ao.accessCount[key] = 1
	ao.lruList = append(ao.lruList, key)
}

func (ao *AlgorithmOptimizer) evictLRU() {
	if len(ao.lruList) == 0 {
		return
	}

	key := ao.lruList[0]
	ao.lruList = ao.lruList[1:]
	delete(ao.cache, key)
	delete(ao.accessCount, key)
}
