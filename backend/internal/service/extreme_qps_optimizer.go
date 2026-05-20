package service

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type ExtremeQPSOptimizer struct {
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	isRunning        bool
	config          *QPSConfig
	stats           *QPSStats
	handler         RequestHandler
	middleware      []Middleware
	workerPool      *WorkerPoolV2
	batchProcessor  *BatchProcessor
	cache           *QPSCache
	circuitBreaker  *CircuitBreaker
	rateLimiter     *RateLimiterV2
	loadBalancer    *LoadBalancer
}

type QPSConfig struct {
	TargetQPS           int64
	MaxQPS              int64
	WorkerCount         int
	BatchSize           int
	BatchTimeout        time.Duration
	EnableBatching      bool
	EnableCompression   bool
	EnablePipeline      bool
	EnableHTTP2         bool
	ReadBufferSize      int
	WriteBufferSize     int
	KeepAliveTimeout    time.Duration
	MaxIdleConns        int
	MaxIdleConnsPerHost int
}

type QPSStats struct {
	TotalRequests      atomic.Int64
	SuccessfulRequests atomic.Int64
	FailedRequests     atomic.Int64
	RejectedRequests   atomic.Int64
	CurrentQPS         atomic.Int64
	PeakQPS            atomic.Int64
	AvgLatency         atomic.Int64
	P50Latency         atomic.Int64
	P90Latency         atomic.Int64
	P95Latency         atomic.Int64
	P99Latency         atomic.Int64
	MaxLatency         atomic.Int64
	MinLatency         atomic.Int64
	CacheHits          atomic.Int64
	CacheMisses        atomic.Int64
	BatchCount         atomic.Int64
	BatchesProcessed   atomic.Int64
	Errors             atomic.Int64
	LastUpdate         atomic.Value
}

type RequestHandler func(req *Request) *Response

type Request struct {
	ID        string
	Path      string
	Method    string
	Headers   map[string]string
	Body      []byte
	Timestamp time.Time
}

type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	Error      error
}

type Middleware func(RequestHandler) RequestHandler

type WorkerPoolV2 struct {
	workers    []*WorkerV2
	taskQueue  chan *Task
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
	active     int
	minWorkers int
	maxWorkers int
}

type WorkerV2 struct {
	id      int
	tasks   chan *Task
	ctx     context.Context
	cancel  context.CancelFunc
}

type Task struct {
	Request  *Request
	Response chan *Response
}

type BatchProcessor struct {
	mu         sync.RWMutex
	batchQueue chan *BatchRequest
	batchSize  int
	timeout    time.Duration
	processing atomic.Bool
}

type BatchRequest struct {
	Requests  []*Request
	Response  chan []*Response
	CreatedAt time.Time
}

type QPSCache struct {
	mu       sync.RWMutex
	items    map[string]*CacheItem
	maxSize  int
	hits     atomic.Int64
	misses   atomic.Int64
	evictions atomic.Int64
}

type CacheItem struct {
	Key        string
	Value      []byte
	Expiration time.Time
	Hits       int64
}

type CircuitBreaker struct {
	mu            sync.RWMutex
	state         CircuitState
	failureCount  atomic.Int64
	successCount  atomic.Int64
	threshold     int64
	timeout       time.Duration
	lastFailure   time.Time
	halfOpenMax   int64
	halfOpenCount atomic.Int64
}

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

type RateLimiterV2 struct {
	mu         sync.RWMutex
	tokens     int64
	maxTokens  int64
	refillRate time.Duration
	lastRefill time.Time
}

type LoadBalancer struct {
	mu         sync.RWMutex
	nodes      []*Node
	algorithm  LoadBalancingAlgorithm
	currentIdx atomic.Int64
}

type Node struct {
	ID        string
	Address   string
	Weight    int
	Active    atomic.Bool
	Requests  atomic.Int64
	Failures  atomic.Int64
}

type LoadBalancingAlgorithm int

const (
	RoundRobin LoadBalancingAlgorithm = iota
	WeightedRoundRobin
	LeastConnections
	IPHash
)

const (
	TargetQPS = 15000
)

func NewExtremeQPSOptimizer(config *QPSConfig) *ExtremeQPSOptimizer {
	if config == nil {
		config = DefaultQPSConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ExtremeQPSOptimizer{
		ctx:            ctx,
		cancel:         cancel,
		config:         config,
		stats:          &QPSStats{},
		workerPool:     NewWorkerPoolV2(config.WorkerCount),
		batchProcessor: NewBatchProcessor(config.BatchSize, config.BatchTimeout),
		cache:          NewQPSCache(10000),
		circuitBreaker: NewCircuitBreaker(10, 30*time.Second),
		rateLimiter:    NewRateLimiterV2(config.TargetQPS),
		loadBalancer:   NewLoadBalancer(),
		middleware:     make([]Middleware, 0),
	}
}

func DefaultQPSConfig() *QPSConfig {
	return &QPSConfig{
		TargetQPS:           TargetQPS,
		MaxQPS:              TargetQPS * 2,
		WorkerCount:         runtime.NumCPU() * 4,
		BatchSize:           100,
		BatchTimeout:        10 * time.Millisecond,
		EnableBatching:      true,
		EnableCompression:   true,
		EnablePipeline:      true,
		EnableHTTP2:         true,
		ReadBufferSize:      32 * 1024,
		WriteBufferSize:     32 * 1024,
		KeepAliveTimeout:    30 * time.Second,
		MaxIdleConns:        10000,
		MaxIdleConnsPerHost: 1000,
	}
}

func (e *ExtremeQPSOptimizer) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.isRunning {
		return nil
	}

	e.isRunning = true

	if err := e.workerPool.Start(); err != nil {
		return err
	}

	go e.batchProcessor.Start(e.ctx)
	go e.monitor()
	go e.adaptiveOptimization()

	log.Printf("[ExtremeQPSOptimizer] Started with target QPS: %d", e.config.TargetQPS)
	return nil
}

func (e *ExtremeQPSOptimizer) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning {
		return
	}

	e.cancel()
	e.isRunning = false
	e.workerPool.Stop()

	log.Println("[ExtremeQPSOptimizer] Stopped")
}

func (e *ExtremeQPSOptimizer) Handle(req *Request) *Response {
	start := time.Now()
	e.stats.TotalRequests.Add(1)

	if e.config.EnableBatching && e.shouldBatch(req) {
		return e.handleBatched(req)
	}

	if e.circuitBreaker.IsOpen() {
		e.stats.RejectedRequests.Add(1)
		return &Response{
			StatusCode: 503,
			Error:      fmt.Errorf("circuit breaker open"),
		}
	}

	cacheKey := e.getCacheKey(req)
	if cached := e.cache.Get(cacheKey); cached != nil {
		e.stats.CacheHits.Add(1)
		return &Response{
			StatusCode: 200,
			Body:       cached,
		}
	}
	e.stats.CacheMisses.Add(1)

	response := e.processRequest(req)

	latency := time.Since(start)
	e.recordLatency(latency)

	if response.Error == nil {
		e.stats.SuccessfulRequests.Add(1)
		e.circuitBreaker.RecordSuccess()
		if req.Path != "" && response.Body != nil {
			e.cache.Set(cacheKey, response.Body)
		}
	} else {
		e.stats.FailedRequests.Add(1)
		e.circuitBreaker.RecordFailure()
		e.stats.Errors.Add(1)
	}

	e.stats.LastUpdate.Store(time.Now())
	return response
}

func (e *ExtremeQPSOptimizer) processRequest(req *Request) *Response {
	var handler RequestHandler = e.handler
	for i := len(e.middleware) - 1; i >= 0; i-- {
		handler = e.middleware[i](handler)
	}
	return handler(req)
}

func (e *ExtremeQPSOptimizer) shouldBatch(req *Request) bool {
	return req.Path == "/api/v1/batch" || req.Method == "POST"
}

func (e *ExtremeQPSOptimizer) handleBatched(req *Request) *Response {
	batchReq := &BatchRequest{
		Requests:  []*Request{req},
		Response:  make(chan []*Response, 1),
		CreatedAt: time.Now(),
	}

	select {
	case e.batchProcessor.batchQueue <- batchReq:
		e.stats.BatchCount.Add(1)
		responses := <-batchReq.Response
		e.stats.BatchesProcessed.Add(1)
		if len(responses) > 0 {
			return responses[0]
		}
	default:
		e.stats.RejectedRequests.Add(1)
	}

	return &Response{
		StatusCode: 429,
		Error:      fmt.Errorf("batch queue full"),
	}
}

func (e *ExtremeQPSOptimizer) recordLatency(latency time.Duration) {
	latencyNs := latency.Nanoseconds()

	oldAvg := e.stats.AvgLatency.Load()
	count := e.stats.TotalRequests.Load()
	if count > 0 {
		e.stats.AvgLatency.Store((oldAvg*(count-1) + latencyNs) / count)
	}

	minLatency := e.stats.MinLatency.Load()
	if minLatency == 0 || latencyNs < minLatency {
		e.stats.MinLatency.Store(latencyNs)
	}

	maxLatency := e.stats.MaxLatency.Load()
	if latencyNs > maxLatency {
		e.stats.MaxLatency.Store(latencyNs)
	}
}

func (e *ExtremeQPSOptimizer) getCacheKey(req *Request) string {
	return fmt.Sprintf("%s:%s:%s", req.Method, req.Path, req.ID)
}

func (e *ExtremeQPSOptimizer) monitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastCount int64
	latencies := make([]int64, 0, 1000)

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			current := e.stats.TotalRequests.Load()
			qps := current - lastCount
			lastCount = current

			e.stats.CurrentQPS.Store(qps)
			if qps > e.stats.PeakQPS.Load() {
				e.stats.PeakQPS.Store(qps)
			}

			if qps > e.config.TargetQPS*90/100 {
				log.Printf("[ExtremeQPSOptimizer] High QPS: %d (target: %d)", qps, e.config.TargetQPS)
			}
		}
	}
}

func (e *ExtremeQPSOptimizer) adaptiveOptimization() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			currentQPS := e.stats.CurrentQPS.Load()

			if currentQPS > e.config.TargetQPS*80/100 {
				e.workerPool.ScaleUp()
			} else if currentQPS < e.config.TargetQPS*30/100 {
				e.workerPool.ScaleDown()
			}
		}
	}
}

func (e *ExtremeQPSOptimizer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_requests":      e.stats.TotalRequests.Load(),
		"successful_requests":  e.stats.SuccessfulRequests.Load(),
		"failed_requests":      e.stats.FailedRequests.Load(),
		"rejected_requests":    e.stats.RejectedRequests.Load(),
		"current_qps":          e.stats.CurrentQPS.Load(),
		"peak_qps":             e.stats.PeakQPS.Load(),
		"target_qps":           e.config.TargetQPS,
		"avg_latency_ns":       e.stats.AvgLatency.Load(),
		"p50_latency_ns":       e.stats.P50Latency.Load(),
		"p90_latency_ns":       e.stats.P90Latency.Load(),
		"p99_latency_ns":       e.stats.P99Latency.Load(),
		"min_latency_ns":       e.stats.MinLatency.Load(),
		"max_latency_ns":       e.stats.MaxLatency.Load(),
		"cache_hits":           e.stats.CacheHits.Load(),
		"cache_misses":         e.stats.CacheMisses.Load(),
		"batch_count":          e.stats.BatchCount.Load(),
		"batches_processed":    e.stats.BatchesProcessed.Load(),
		"errors":               e.stats.Errors.Load(),
		"circuit_breaker":      e.circuitBreaker.GetState(),
		"worker_pool":          e.workerPool.GetStats(),
		"last_update":          e.stats.LastUpdate.Load(),
	}
}

func (e *ExtremeQPSOptimizer) SetHandler(handler RequestHandler) {
	e.handler = handler
}

func (e *ExtremeQPSOptimizer) AddMiddleware(middleware Middleware) {
	e.middleware = append(e.middleware, middleware)
}

func NewWorkerPoolV2(workerCount int) *WorkerPoolV2 {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPoolV2{
		tasks:      make(chan *Task, 10000),
		ctx:        ctx,
		cancel:     cancel,
		minWorkers: workerCount / 2,
		maxWorkers: workerCount * 2,
	}
}

func (wp *WorkerPoolV2) Start() error {
	for i := 0; i < wp.minWorkers; i++ {
		wp.addWorker(i)
	}
	return nil
}

func (wp *WorkerPoolV2) Stop() {
	wp.cancel()
	wp.wg.Wait()
}

func (wp *WorkerPoolV2) addWorker(id int) {
	workerCtx, workerCancel := context.WithCancel(wp.ctx)

	worker := &WorkerV2{
		id:     id,
		tasks:  wp.tasks,
		ctx:    workerCtx,
		cancel: workerCancel,
	}

	wp.workers = append(wp.workers, worker)
	wp.wg.Add(1)

	go worker.run(&wp.wg)
}

func (w *WorkerV2) run(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		case task := <-w.tasks:
			response := &Response{
				StatusCode: 200,
				Body:       []byte("processed"),
			}
			task.Response <- response
		}
	}
}

func (wp *WorkerPoolV2) Submit(req *Request) *Response {
	task := &Task{
		Request:  req,
		Response: make(chan *Response, 1),
	}

	select {
	case wp.tasks <- task:
		return <-task.Response
	default:
		return &Response{
			StatusCode: 429,
			Error:      fmt.Errorf("worker pool queue full"),
		}
	}
}

func (wp *WorkerPoolV2) ScaleUp() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if len(wp.workers) < wp.maxWorkers {
		wp.addWorker(len(wp.workers))
		log.Printf("[WorkerPoolV2] Scaled up to %d workers", len(wp.workers))
	}
}

func (wp *WorkerPoolV2) ScaleDown() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if len(wp.workers) > wp.minWorkers {
		worker := wp.workers[len(wp.workers)-1]
		worker.cancel()
		wp.workers = wp.workers[:len(wp.workers)-1]
		log.Printf("[WorkerPoolV2] Scaled down to %d workers", len(wp.workers))
	}
}

func (wp *WorkerPoolV2) GetStats() map[string]interface{} {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	return map[string]interface{}{
		"workers":      len(wp.workers),
		"min_workers":  wp.minWorkers,
		"max_workers":  wp.maxWorkers,
		"queue_length": len(wp.tasks),
	}
}

func NewBatchProcessor(batchSize int, timeout time.Duration) *BatchProcessor {
	return &BatchProcessor{
		batchSize: batchSize,
		timeout:   timeout,
		batchQueue: make(chan *BatchRequest, 100),
	}
}

func (bp *BatchProcessor) Start(ctx context.Context) {
	go bp.processBatches(ctx)
}

func (bp *BatchProcessor) processBatches(ctx context.Context) {
	batch := make([]*BatchRequest, 0, bp.batchSize)

	ticker := time.NewTicker(bp.timeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case req := <-bp.batchQueue:
			batch = append(batch, req)
			if len(batch) >= bp.batchSize {
				bp.processBatch(batch)
				batch = make([]*BatchRequest, 0, bp.batchSize)
			}
		case <-ticker.C:
			if len(batch) > 0 {
				bp.processBatch(batch)
				batch = make([]*BatchRequest, 0, bp.batchSize)
			}
		}
	}
}

func (bp *BatchProcessor) processBatch(batch []*BatchRequest) {
	responses := make([][]*Response, len(batch))

	for i, req := range batch {
		resp := make([]*Response, len(req.Requests))
		for j, r := range req.Requests {
			resp[j] = &Response{
				StatusCode: 200,
				Body:       []byte("batch processed"),
			}
		}
		responses[i] = resp
	}

	for i, req := range batch {
		req.Response <- responses[i]
	}
}

func NewQPSCache(maxSize int) *QPSCache {
	return &QPSCache{
		items:   make(map[string]*CacheItem),
		maxSize: maxSize,
	}
}

func (c *QPSCache) Get(key string) []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		c.misses.Add(1)
		return nil
	}

	if time.Now().After(item.Expiration) {
		delete(c.items, key)
		c.misses.Add(1)
		return nil
	}

	atomic.AddInt64(&item.Hits, 1)
	c.hits.Add(1)
	return item.Value
}

func (c *QPSCache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.items) >= c.maxSize {
		c.evict()
	}

	c.items[key] = &CacheItem{
		Key:        key,
		Value:      value,
		Expiration: time.Now().Add(5 * time.Minute),
	}
}

func (c *QPSCache) evict() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range c.items {
		if oldestTime.IsZero() || item.Expiration.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.Expiration
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
		c.evictions.Add(1)
	}
}

func NewCircuitBreaker(threshold int64, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold:   threshold,
		timeout:     timeout,
		halfOpenMax: 3,
		state:       CircuitClosed,
	}
}

func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitOpen:
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = CircuitHalfOpen
			cb.halfOpenCount.Store(0)
			return false
		}
		return true
	case CircuitHalfOpen:
		return cb.halfOpenCount.Load() >= cb.halfOpenMax
	default:
		return false
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.failureCount.Store(0)
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitHalfOpen {
		cb.halfOpenCount.Add(1)
		if cb.halfOpenCount.Load() >= cb.halfOpenMax {
			cb.state = CircuitClosed
			cb.successCount.Store(0)
		}
	} else {
		cb.successCount.Add(1)
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.lastFailure = time.Now()
	cb.failureCount.Add(1)
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.failureCount.Load() >= cb.threshold {
		cb.state = CircuitOpen
	}
}

func (cb *CircuitBreaker) GetState() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

func NewRateLimiterV2(qps int64) *RateLimiterV2 {
	return &RateLimiterV2{
		maxTokens:  qps,
		tokens:     qps,
		refillRate: time.Second / time.Duration(qps),
		lastRefill: time.Now(),
	}
}

func (r *RateLimiterV2) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()

	if r.tokens > 0 {
		r.tokens--
		return true
	}
	return false
}

func (r *RateLimiterV2) refill() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	tokensToAdd := int64(elapsed / r.refillRate)

	if tokensToAdd > 0 {
		r.tokens = minInt64(r.tokens+tokensToAdd, r.maxTokens)
		r.lastRefill = now
	}
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		nodes:     make([]*Node, 0),
		algorithm: RoundRobin,
	}
}

func (lb *LoadBalancer) AddNode(id, address string, weight int) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.nodes = append(lb.nodes, &Node{
		ID:      id,
		Address: address,
		Weight:  weight,
		Active:  atomic.Bool{},
	})
	lb.nodes[len(lb.nodes)-1].Active.Store(true)
}

func (lb *LoadBalancer) GetNextNode() *Node {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.nodes) == 0 {
		return nil
	}

	switch lb.algorithm {
	case RoundRobin:
		idx := int(lb.currentIdx.Add(1) % int64(len(lb.nodes)))
		return lb.nodes[idx]
	case LeastConnections:
		return lb.getLeastConnectionsNode()
	default:
		return lb.nodes[0]
	}
}

func (lb *LoadBalancer) getLeastConnectionsNode() *Node {
	var minConnNode *Node
	var minConn int64 = -1

	for _, node := range lb.nodes {
		if node.Active.Load() {
			conn := node.Requests.Load()
			if minConn == -1 || conn < minConn {
				minConn = conn
				minConnNode = node
			}
		}
	}

	if minConnNode != nil {
		minConnNode.Requests.Add(1)
	}

	return minConnNode
}
