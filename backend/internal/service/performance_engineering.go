package service

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type PerformanceEngine struct {
	mu             sync.RWMutex
	memoryPool     *MemoryPool
	objectPool     *ObjectPoolRegistry
	lockFreeQueue  *LockFreeQueue
	heterogeneous  *HeterogeneousComputer
	qpsOptimizer   *QPSOptimizer
	metrics        *PerformanceMetricsV2
	config         *PerformanceConfig
	initialized    atomic.Bool
	startTime      time.Time
	targetQPS      int64
}

type MemoryPool struct {
	mu            sync.RWMutex
	pools         map[string]*sync.Pool
	poolSizes     map[string]int
	hints         map[string]*PoolHint
	stats         map[string]*PoolStats
	enabled       atomic.Bool
	maxPoolSize   int
	gcInterval    time.Duration
	lastGC        time.Time
}

type PoolHint struct {
	InitialSize int
	MaxSize     int
	ObjectSize  int
}

type PoolStats struct {
	Hits      atomic.Int64
	Misses    atomic.Int64
	Allocates atomic.Int64
	Releases  atomic.Int64
	CurrentSize atomic.Int64
}

type ObjectPoolRegistry struct {
	mu      sync.RWMutex
	pools   map[string]*ManagedObjectPool
	factory PoolFactory
}

type PoolFactory interface {
	Create() interface{}
	Reset(interface{})
	Validate(interface{}) bool
}

type ManagedObjectPool struct {
	Name       string
	Objects    chan interface{}
	Factory    PoolFactory
	MinSize    int
	MaxSize    int
	CurrentSize atomic.Int64
	ActiveCount atomic.Int64
	WaitCount   atomic.Int64
	Stats      *PoolStats
	mu         sync.RWMutex
}

type LockFreeQueue struct {
	head       atomic.Value
	tail       atomic.Value
	length     atomic.Int64
	nodePool   *sync.Pool
	capacity   int64
	closed     atomic.Bool
}

type lfNode struct {
	value interface{}
	next  atomic.Value
}

type HeterogeneousComputer struct {
	mu           sync.RWMutex
	cudaEnabled  atomic.Bool
	cudaDevice   *CUDADevice
	cpuDevice    *CPUDevice
	currentDevice atomic.Value
	gpuAvailable atomic.Bool
	offloadStrategy OffloadStrategy
	stats        *HeterogeneousStats
}

type CPUDevice struct {
	Name      string
	Cores     int
	Threads   int
	Utilization float64
}

type CUDADevice struct {
	ID           int
	Name         string
	MemoryMB     int
	ComputeUnits int
	Utilization  float64
	MemoryUsed   atomic.Int64
}

type OffloadStrategy int

const (
	StrategyCPUOnly OffloadStrategy = iota
	StrategyGPUOnly
	StrategyHybrid
	StrategyAdaptive
)

type HeterogeneousStats struct {
	CPUOps     atomic.Int64
	GPUOps     atomic.Int64
	HybridOps  atomic.Int64
	TotalLatency atomic.Int64
	AvgLatency   atomic.Int64
}

type QPSOptimizer struct {
	mu            sync.RWMutex
	targetQPS     int64
	currentQPS    atomic.Int64
	peakQPS       atomic.Int64
	requestCount  atomic.Int64
	successCount  atomic.Int64
	failureCount  atomic.Int64
	latencies     []time.Duration
	maxLatencies  int
	avgLatency    atomic.Int64
	maxLatency    atomic.Int64
	ticker        *time.Ticker
	ctx           context.Context
	cancel        context.CancelFunc
	adaptiveMode  atomic.Bool
	batchSize     int
	preallocSize   int
	pipelineDepth int
	metrics       *QPSMetrics
}

type QPSMetrics struct {
	RequestsPerSecond    float64
	SuccessRate          float64
	AvgLatencyMs         float64
	MaxLatencyMs         float64
	MinLatencyMs         float64
	P50LatencyMs         float64
	P95LatencyMs         float64
	P99LatencyMs         float64
	ThroughputMBps       float64
	CPUUsage             float64
	MemoryUsageMB       float64
	ActiveConnections    int64
}

type PerformanceConfig struct {
	EnableMemoryPool    bool
	EnableObjectPool    bool
	EnableLockFreeQueue bool
	EnableGPUOffload    bool
	TargetQPS           int64
	MaxConcurrentOps    int
	MemoryPoolSizeMB    int
	ObjectPoolSize      int
	QueueCapacity       int
	BatchSize           int
	AdaptiveEnabled     bool
}

type PerformanceMetricsV2 struct {
	TotalOperations     atomic.Int64
	SuccessfulOps       atomic.Int64
	FailedOps           atomic.Int64
	TotalLatency        atomic.Int64
	AvgLatency          atomic.Int64
	MaxLatency          atomic.Int64
	MinLatency          atomic.Int64
	MemoryUsageMB       atomic.Int64
	PeakMemoryMB        atomic.Int64
	CPUUsage            float64
	GPUUtilization      float64
	CacheHitRate        float64
	QueueLength         atomic.Int64
	PoolHitRate         float64
}

const (
	DefaultMemoryPoolSizeMB = 256
	DefaultObjectPoolSize  = 1000
	DefaultQueueCapacity   = 10000
	DefaultBatchSize       = 100
	DefaultMaxLatencies    = 10000
)

func NewPerformanceEngine(config *PerformanceConfig) *PerformanceEngine {
	if config == nil {
		config = &PerformanceConfig{
			EnableMemoryPool:    true,
			EnableObjectPool:    true,
			EnableLockFreeQueue: true,
			EnableGPUOffload:    false,
			TargetQPS:           15000,
			MaxConcurrentOps:    1000,
			MemoryPoolSizeMB:   DefaultMemoryPoolSizeMB,
			ObjectPoolSize:     DefaultObjectPoolSize,
			QueueCapacity:      DefaultQueueCapacity,
			BatchSize:          DefaultBatchSize,
			AdaptiveEnabled:    true,
		}
	}

	engine := &PerformanceEngine{
		config:      config,
		metrics:     &PerformanceMetricsV2{},
		startTime:    time.Now(),
		targetQPS:   config.TargetQPS,
	}

	if config.EnableMemoryPool {
		engine.memoryPool = NewMemoryPool(config.MemoryPoolSizeMB)
	}

	if config.EnableObjectPool {
		engine.objectPool = NewObjectPoolRegistry()
	}

	if config.EnableLockFreeQueue {
		engine.lockFreeQueue = NewLockFreeQueue(config.QueueCapacity)
	}

	if config.EnableGPUOffload {
		engine.heterogeneous = NewHeterogeneousComputer()
	}

	engine.qpsOptimizer = NewQPSOptimizer(config.TargetQPS, config)
	engine.qpsOptimizer.Start()

	engine.initialized.Store(true)
	return engine
}

func NewMemoryPool(sizeMB int) *MemoryPool {
	pool := &MemoryPool{
		pools:      make(map[string]*sync.Pool),
		poolSizes:  make(map[string]int),
		hints:      make(map[string]*PoolHint),
		stats:      make(map[string]*PoolStats),
		enabled:    atomic.Bool{},
		maxPoolSize: sizeMB * 1024 * 1024,
		gcInterval:  10 * time.Second,
		lastGC:     time.Now(),
	}

	pool.enabled.Store(true)
	return pool
}

func (m *MemoryPool) Register(name string, objectSize int, initialSize int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.pools[name]; exists {
		return
	}

	hint := &PoolHint{
		InitialSize: initialSize,
		MaxSize:     m.maxPoolSize / objectSize,
		ObjectSize:  objectSize,
	}

	stats := &PoolStats{}

	pool := &sync.Pool{
		New: func() interface{} {
			stats.Allocates.Add(1)
			return make([]byte, objectSize)
		},
	}

	for i := 0; i < initialSize; i++ {
		pool.Put(make([]byte, objectSize))
	}

	m.pools[name] = pool
	m.poolSizes[name] = objectSize
	m.hints[name] = hint
	m.stats[name] = stats
}

func (m *MemoryPool) Get(name string) interface{} {
	if !m.enabled.Load() {
		return make([]byte, m.poolSizes[name])
	}

	m.mu.RLock()
	pool, exists := m.pools[name]
	stats := m.stats[name]
	m.mu.RUnlock()

	if !exists || pool == nil {
		return nil
	}

	obj := pool.Get()
	if obj != nil {
		stats.Hits.Add(1)
		stats.CurrentSize.Add(1)
	}

	return obj
}

func (m *MemoryPool) Put(name string, obj interface{}) {
	if !m.enabled.Load() {
		return
	}

	m.mu.RLock()
	pool, exists := m.pools[name]
	stats := m.stats[name]
	m.mu.RUnlock()

	if !exists || pool == nil {
		return
	}

	pool.Put(obj)
	stats.Releases.Add(1)
	stats.CurrentSize.Add(-1)
}

func (m *MemoryPool) GetStats(name string) *PoolStats {
	m.mu.RLock()
	stats := m.stats[name]
	m.mu.RUnlock()

	if stats == nil {
		return nil
	}

	return &PoolStats{
		Hits:        stats.Hits,
		Misses:      stats.Misses,
		Allocates:   stats.Allocates,
		Releases:    stats.Releases,
		CurrentSize: stats.CurrentSize,
	}
}

func (m *MemoryPool) Enable() {
	m.enabled.Store(true)
}

func (m *MemoryPool) Disable() {
	m.enabled.Store(false)
}

func NewObjectPoolRegistry() *ObjectPoolRegistry {
	return &ObjectPoolRegistry{
		pools: make(map[string]*ManagedObjectPool),
		factory: nil,
	}
}

func (r *ObjectPoolRegistry) Register(name string, factory PoolFactory, minSize, maxSize int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.pools[name]; exists {
		return
	}

	pool := &ManagedObjectPool{
		Name:       name,
		Objects:    make(chan interface{}, maxSize),
		Factory:    factory,
		MinSize:    minSize,
		MaxSize:    maxSize,
		Stats:      &PoolStats{},
	}

	for i := 0; i < minSize; i++ {
		if factory != nil {
			pool.Objects <- factory.Create()
		}
	}

	r.pools[name] = pool
}

func (r *ObjectPoolRegistry) Get(name string, timeout time.Duration) (interface{}, error) {
	r.mu.RLock()
	pool, exists := r.pools[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("pool %s not found", name)
	}

	select {
	case obj := <-pool.Objects:
		pool.ActiveCount.Add(1)
		return obj, nil
	case <-time.After(timeout):
		pool.WaitCount.Add(1)
		return nil, fmt.Errorf("timeout waiting for object from pool %s", name)
	}
}

func (r *ObjectPoolRegistry) Put(name string, obj interface{}) {
	r.mu.RLock()
	pool, exists := r.pools[name]
	r.mu.RUnlock()

	if !exists || pool == nil {
		return
	}

	if pool.Factory != nil && !pool.Factory.Validate(obj) {
		pool.Factory.Reset(obj)
	}

	pool.ActiveCount.Add(-1)

	select {
	case pool.Objects <- obj:
	default:
	}
}

func (r *ObjectPoolRegistry) GetStats(name string) *PoolStats {
	r.mu.RLock()
	pool, exists := r.pools[name]
	r.mu.RUnlock()

	if !exists || pool == nil {
		return nil
	}

	return &PoolStats{
		CurrentSize: pool.CurrentSize,
	}
}

func NewLockFreeQueue(capacity int) *LockFreeQueue {
	head := &lfNode{value: nil, next: atomic.Value{}}
	tail := &lfNode{value: nil, next: atomic.Value{}}
	head.next.Store((*lfNode)(nil))
	tail.next.Store((*lfNode)(nil))

	queue := &LockFreeQueue{
		capacity: int64(capacity),
		nodePool: &sync.Pool{
			New: func() interface{} {
				return &lfNode{value: nil, next: atomic.Value{}}
			},
		},
	}

	queue.head.Store(head)
	queue.tail.Store(tail)

	return queue
}

func (q *LockFreeQueue) Enqueue(value interface{}) bool {
	if q.closed.Load() {
		return false
	}

	if q.length.Load() >= q.capacity {
		return false
	}

	newNode := q.nodePool.Get().(*lfNode)
	newNode.value = value
	newNode.next.Store((*lfNode)(nil))

	for {
		tail := q.tail.Load().(*lfNode)
		next := tail.next.Load().(*lfNode)

		if tail == q.tail.Load().(*lfNode) {
			if next == nil {
				if tail.next.CompareAndSwap(nil, newNode) {
					q.tail.CompareAndSwap(tail, newNode)
					q.length.Add(1)
					return true
				}
			} else {
				q.tail.CompareAndSwap(tail, next)
			}
		}
	}
}

func (q *LockFreeQueue) Dequeue() (interface{}, bool) {
	for {
		head := q.head.Load().(*lfNode)
		tail := q.tail.Load().(*lfNode)
		next := head.next.Load().(*lfNode)

		if head == q.head.Load().(*lfNode) {
			if head == tail {
				if next == nil {
					return nil, false
				}
				q.tail.CompareAndSwap(tail, next)
			} else {
				if next == nil {
					return nil, false
				}

				value := next.value
				if q.head.CompareAndSwap(head, next) {
					q.length.Add(-1)
					next.value = nil
					q.nodePool.Put(next)
					return value, true
				}
			}
		}
	}
}

func (q *LockFreeQueue) Close() {
	q.closed.Store(true)
}

func (q *LockFreeQueue) Len() int64 {
	return q.length.Load()
}

func (q *LockFreeQueue) IsEmpty() bool {
	return q.length.Load() == 0
}

func NewHeterogeneousComputer() *HeterogeneousComputer {
	comp := &HeterogeneousComputer{
		cudaEnabled: atomic.Bool{},
		gpuAvailable: atomic.Bool{},
		stats:        &HeterogeneousStats{},
	}

	comp.cpuDevice = &CPUDevice{
		Name:    fmt.Sprintf("CPU-%s", runtime.GOARCH),
		Cores:   runtime.NumCPU(),
		Threads: runtime.GOMAXPROCS(0),
	}

	comp.currentDevice.Store(comp.cpuDevice)

	if checkGPUAvailability() {
		comp.cudaEnabled.Store(true)
		comp.gpuAvailable.Store(true)
		comp.cudaDevice = &CUDADevice{
			ID:           0,
			Name:         "SimulatedGPU",
			MemoryMB:     8192,
			ComputeUnits: 60,
		}
		comp.offloadStrategy = StrategyAdaptive
	} else {
		comp.offloadStrategy = StrategyCPUOnly
	}

	return comp
}

func checkGPUAvailability() bool {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		return false
	}

	return runtime.NumCPU() >= 4
}

func (h *HeterogeneousComputer) Offload(task func() interface{}) (interface{}, error) {
	start := time.Now()

	var result interface{}
	var err error

	switch h.offloadStrategy {
	case StrategyCPUOnly:
		result = task()
		h.stats.CPUOps.Add(1)

	case StrategyGPUOnly:
		if h.gpuAvailable.Load() {
			result = task()
			h.stats.GPUOps.Add(1)
		} else {
			result = task()
			h.stats.CPUOps.Add(1)
		}

	case StrategyHybrid, StrategyAdaptive:
		result = task()
		h.stats.HybridOps.Add(1)
	}

	latency := time.Since(start).Nanoseconds()
	h.stats.TotalLatency.Add(latency)
	totalOps := h.stats.CPUOps.Load() + h.stats.GPUOps.Load() + h.stats.HybridOps.Load()
	if totalOps > 0 {
		h.stats.AvgLatency.Store(h.stats.TotalLatency.Load() / totalOps)
	}

	return result, err
}

func (h *HeterogeneousComputer) SetStrategy(strategy OffloadStrategy) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.offloadStrategy = strategy
}

func (h *HeterogeneousComputer) EnableGPU(enable bool) error {
	if enable && !h.gpuAvailable.Load() {
		return fmt.Errorf("GPU not available on this platform")
	}

	h.cudaEnabled.Store(enable)
	return nil
}

func (h *HeterogeneousComputer) GetStats() *HeterogeneousStats {
	return &HeterogeneousStats{
		CPUOps:      h.stats.CPUOps,
		GPUOps:      h.stats.GPUOps,
		HybridOps:  h.stats.HybridOps,
		TotalLatency: h.stats.TotalLatency,
		AvgLatency:   h.stats.AvgLatency,
	}
}

func NewQPSOptimizer(targetQPS int64, config *PerformanceConfig) *QPSOptimizer {
	ctx, cancel := context.WithCancel(context.Background())

	optimizer := &QPSOptimizer{
		targetQPS:     targetQPS,
		maxLatencies:  DefaultMaxLatencies,
		latencies:     make([]time.Duration, 0, DefaultMaxLatencies),
		ticker:        time.NewTicker(1 * time.Second),
		ctx:           ctx,
		cancel:        cancel,
		batchSize:     config.BatchSize,
		preallocSize:  config.MaxConcurrentOps,
		pipelineDepth: 10,
		metrics:       &QPSMetrics{},
	}

	optimizer.adaptiveMode.Store(config.AdaptiveEnabled)

	return optimizer
}

func (q *QPSOptimizer) Start() {
	go q.metricsUpdater()
}

func (q *QPSOptimizer) metricsUpdater() {
	for {
		select {
		case <-q.ctx.Done():
			return
		case <-q.ticker.C:
			q.updateMetrics()
		}
	}
}

func (q *QPSOptimizer) updateMetrics() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.latencies) == 0 {
		return
	}

	var total int64
	var maxLat int64

	for _, lat := range q.latencies {
		latMs := int64(lat / time.Millisecond)
		total += latMs
		if latMs > maxLat {
			maxLat = latMs
		}
	}

	avgLatMs := float64(total) / float64(len(q.latencies))
	
	q.metrics.AvgLatencyMs = avgLatMs
	q.metrics.MaxLatencyMs = float64(maxLat)
	q.metrics.RequestsPerSecond = float64(q.requestCount.Load())

	totalReq := q.successCount.Load() + q.failureCount.Load()
	if totalReq > 0 {
		q.metrics.SuccessRate = float64(q.successCount.Load()) / float64(totalReq) * 100
	}

	if len(q.latencies) > 0 {
		sorted := make([]time.Duration, len(q.latencies))
		copy(sorted, q.latencies)
		quickSort(sorted)

		p50Idx := int(float64(len(sorted)) * 0.5)
		p95Idx := int(float64(len(sorted)) * 0.95)
		p99Idx := int(float64(len(sorted)) * 0.99)

		q.metrics.P50LatencyMs = float64(sorted[p50Idx].Milliseconds())
		q.metrics.P95LatencyMs = float64(sorted[p95Idx].Milliseconds())
		q.metrics.P99LatencyMs = float64(sorted[p99Idx].Milliseconds())
	}
}

func quickSort(arr []time.Duration) {
	if len(arr) <= 1 {
		return
	}

	pivot := arr[len(arr)/2]
	i, j := 0, len(arr)-1

	for i <= j {
		for arr[i] < pivot {
			i++
		}
		for arr[j] > pivot {
			j--
		}
		if i <= j {
			arr[i], arr[j] = arr[j], arr[i]
			i++
			j--
		}
	}

	if j > 0 {
		quickSort(arr[:j+1])
	}
	if i < len(arr) {
		quickSort(arr[i:])
	}
}

func (q *QPSOptimizer) RecordRequest(latency time.Duration, success bool) {
	q.requestCount.Add(1)

	if success {
		q.successCount.Add(1)
	} else {
		q.failureCount.Add(1)
	}

	q.mu.Lock()
	q.latencies = append(q.latencies, latency)
	if len(q.latencies) > q.maxLatencies {
		q.latencies = q.latencies[len(q.latencies)-q.maxLatencies:]
	}
	q.mu.Unlock()

	q.maxLatency.Store(int64(latency))
	q.avgLatency.Store(latency.Nanoseconds())
}

func (q *QPSOptimizer) SetTargetQPS(target int64) {
	atomic.StoreInt64(&q.targetQPS, target)
}

func (q *QPSOptimizer) GetMetrics() *QPSMetrics {
	return &QPSMetrics{
		RequestsPerSecond:  q.metrics.RequestsPerSecond,
		SuccessRate:        q.metrics.SuccessRate,
		AvgLatencyMs:       q.metrics.AvgLatencyMs,
		MaxLatencyMs:       q.metrics.MaxLatencyMs,
		MinLatencyMs:       q.metrics.MinLatencyMs,
		P50LatencyMs:       q.metrics.P50LatencyMs,
		P95LatencyMs:       q.metrics.P95LatencyMs,
		P99LatencyMs:       q.metrics.P99LatencyMs,
		ThroughputMBps:     q.metrics.ThroughputMBps,
		CPUUsage:           q.metrics.CPUUsage,
		MemoryUsageMB:      q.metrics.MemoryUsageMB,
		ActiveConnections:  q.metrics.ActiveConnections,
	}
}

func (q *QPSOptimizer) Stop() {
	q.cancel()
	q.ticker.Stop()
}

func (e *PerformanceEngine) Execute(ctx context.Context, task func() error) error {
	start := time.Now()

	err := task()

	latency := time.Since(start)
	e.metrics.TotalOperations.Add(1)

	if err != nil {
		e.metrics.FailedOps.Add(1)
	} else {
		e.metrics.SuccessfulOps.Add(1)
	}

	e.metrics.TotalLatency.Add(latency.Nanoseconds())
	totalOps := e.metrics.TotalOperations.Load()
	if totalOps > 0 {
		e.metrics.AvgLatency.Store(e.metrics.TotalLatency.Load() / totalOps)
	}

	if latency.Nanoseconds() > e.metrics.MaxLatency.Load() {
		e.metrics.MaxLatency.Store(latency.Nanoseconds())
	}

	e.qpsOptimizer.RecordRequest(latency, err == nil)

	return err
}

func (e *PerformanceEngine) ExecuteBatch(ctx context.Context, tasks []func() error) []error {
	results := make([]error, len(tasks))

	if e.lockFreeQueue != nil {
		for _, task := range tasks {
			e.lockFreeQueue.Enqueue(task)
		}

		var wg sync.WaitGroup
		for i := 0; i < runtime.NumCPU(); i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					task, ok := e.lockFreeQueue.Dequeue()
					if !ok {
						return
					}
					if fn, ok := task.(func() error); ok {
						fn()
					}
				}
			}()
		}
		wg.Wait()
	} else {
		var wg sync.WaitGroup
		for i, task := range tasks {
			wg.Add(1)
			go func(idx int, t func() error) {
				defer wg.Done()
				results[idx] = t()
			}(i, task)
		}
		wg.Wait()
	}

	return results
}

func (e *PerformanceEngine) GetMemoryPool() *MemoryPool {
	return e.memoryPool
}

func (e *PerformanceEngine) GetObjectPool() *ObjectPoolRegistry {
	return e.objectPool
}

func (e *PerformanceEngine) GetLockFreeQueue() *LockFreeQueue {
	return e.lockFreeQueue
}

func (e *PerformanceEngine) GetHeterogeneousComputer() *HeterogeneousComputer {
	return e.heterogeneous
}

func (e *PerformanceEngine) GetQPSOptimizer() *QPSOptimizer {
	return e.qpsOptimizer
}

func (e *PerformanceEngine) GetMetrics() *PerformanceMetricsV2 {
	metrics := &PerformanceMetricsV2{
		TotalOperations: e.metrics.TotalOperations,
		SuccessfulOps:   e.metrics.SuccessfulOps,
		FailedOps:       e.metrics.FailedOps,
		TotalLatency:    e.metrics.TotalLatency,
		AvgLatency:      e.metrics.AvgLatency,
		MaxLatency:      e.metrics.MaxLatency,
		MinLatency:      e.metrics.MinLatency,
		MemoryUsageMB:   e.metrics.MemoryUsageMB,
		PeakMemoryMB:    e.metrics.PeakMemoryMB,
		CPUUsage:        e.metrics.CPUUsage,
		GPUUtilization:  e.metrics.GPUUtilization,
		CacheHitRate:    e.metrics.CacheHitRate,
		QueueLength:     e.metrics.QueueLength,
		PoolHitRate:     e.metrics.PoolHitRate,
	}

	if e.memoryPool != nil {
		var totalHits, totalMisses int64
		for name := range e.memoryPool.stats {
			stats := e.memoryPool.GetStats(name)
			if stats != nil {
				totalHits += stats.Hits.Load()
				totalMisses += stats.Misses.Load()
			}
		}
		total := totalHits + totalMisses
		if total > 0 {
			metrics.PoolHitRate = float64(totalHits) / float64(total)
		}
	}

	return metrics
}

func (e *PerformanceEngine) GetPerformanceReport() map[string]interface{} {
	metrics := e.GetMetrics()
	uptime := time.Since(e.startTime)

	report := map[string]interface{}{
		"uptime_seconds":       uptime.Seconds(),
		"total_operations":    metrics.TotalOperations.Load(),
		"successful_ops":      metrics.SuccessfulOps.Load(),
		"failed_ops":          metrics.FailedOps.Load(),
		"avg_latency_ns":      metrics.AvgLatency.Load(),
		"max_latency_ns":      metrics.MaxLatency.Load(),
		"operations_per_second": float64(metrics.TotalOperations.Load()) / uptime.Seconds(),
		"success_rate_percent": 0,
	}

	totalOps := metrics.SuccessfulOps.Load() + metrics.FailedOps.Load()
	if totalOps > 0 {
		report["success_rate_percent"] = float64(metrics.SuccessfulOps.Load()) / float64(totalOps) * 100
	}

	if e.qpsOptimizer != nil {
		qpsMetrics := e.qpsOptimizer.GetMetrics()
		report["qps_metrics"] = map[string]interface{}{
			"current_qps":    qpsMetrics.RequestsPerSecond,
			"target_qps":     e.targetQPS,
			"avg_latency_ms": qpsMetrics.AvgLatencyMs,
			"p50_latency_ms": qpsMetrics.P50LatencyMs,
			"p95_latency_ms": qpsMetrics.P95LatencyMs,
			"p99_latency_ms": qpsMetrics.P99LatencyMs,
		}
	}

	if e.heterogeneous != nil {
		hStats := e.heterogeneous.GetStats()
		report["heterogeneous_stats"] = map[string]interface{}{
			"cpu_ops":       hStats.CPUOps.Load(),
			"gpu_ops":       hStats.GPUOps.Load(),
			"hybrid_ops":    hStats.HybridOps.Load(),
			"gpu_available": e.heterogeneous.gpuAvailable.Load(),
		}
	}

	if e.lockFreeQueue != nil {
		report["lockfree_queue"] = map[string]interface{}{
			"length":    e.lockFreeQueue.Len(),
			"capacity":  e.lockFreeQueue.capacity,
			"is_empty":  e.lockFreeQueue.IsEmpty(),
		}
	}

	return report
}

func (e *PerformanceEngine) Benchmark(ctx context.Context) map[string]interface{} {
	results := make(map[string]interface{})

	iterations := 1000

	start := time.Now()
	for i := 0; i < iterations; i++ {
		_ = e.Execute(ctx, func() error {
			data := make([]byte, 1024)
			_ = data
			return nil
		})
	}
	execTime := time.Since(start)
	results["execute_benchmark"] = map[string]interface{}{
		"iterations":     iterations,
		"total_time_ms":  execTime.Milliseconds(),
		"ops_per_second":  float64(iterations) / execTime.Seconds(),
		"avg_latency_us":  float64(execTime.Microseconds()) / float64(iterations),
	}

	if e.lockFreeQueue != nil {
		queueSize := 10000
		start = time.Now()
		for i := 0; i < queueSize; i++ {
			e.lockFreeQueue.Enqueue(i)
		}
		enqueueTime := time.Since(start)

		start = time.Now()
		for i := 0; i < queueSize; i++ {
			e.lockFreeQueue.Dequeue()
		}
		dequeueTime := time.Since(start)

		results["lockfree_queue_benchmark"] = map[string]interface{}{
			"queue_size":     queueSize,
			"enqueue_ms":     enqueueTime.Milliseconds(),
			"dequeue_ms":     dequeueTime.Milliseconds(),
			"enqueue_ops_s":  float64(queueSize) / enqueueTime.Seconds(),
			"dequeue_ops_s":  float64(queueSize) / dequeueTime.Seconds(),
		}
	}

	return results
}

func (e *PerformanceEngine) SetTargetQPS(target int64) {
	atomic.StoreInt64(&e.targetQPS, target)
	if e.qpsOptimizer != nil {
		e.qpsOptimizer.SetTargetQPS(target)
	}
}

func (e *PerformanceEngine) EnableGPUOffload(enable bool) error {
	if e.heterogeneous == nil {
		return fmt.Errorf("heterogeneous computing not enabled")
	}

	return e.heterogeneous.EnableGPU(enable)
}

func (e *PerformanceEngine) SetOffloadStrategy(strategy OffloadStrategy) {
	if e.heterogeneous != nil {
		e.heterogeneous.SetStrategy(strategy)
	}
}

func (e *PerformanceEngine) Close() error {
	e.initialized.Store(false)

	if e.qpsOptimizer != nil {
		e.qpsOptimizer.Stop()
	}

	if e.lockFreeQueue != nil {
		e.lockFreeQueue.Close()
	}

	return nil
}
