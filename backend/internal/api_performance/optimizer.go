package api_performance

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type LatencyTarget int

const (
	TargetUltraLow  LatencyTarget = 20
	TargetLow       LatencyTarget = 50
	TargetMedium    LatencyTarget = 100
	TargetStandard  LatencyTarget = 200
)

type APIOptimizationConfig struct {
	TargetLatency    LatencyTarget
	EnablePrefetch   bool
	EnableBatch      bool
	BatchSize        int
	BatchTimeout     time.Duration
	CacheEnabled     bool
	CacheTTL         time.Duration
	CompressionLevel int
	MaxRetries       int
	RetryDelay       time.Duration
}

var DefaultOptimizationConfig = &APIOptimizationConfig{
	TargetLatency:    TargetLow,
	EnablePrefetch:   true,
	EnableBatch:      true,
	BatchSize:        100,
	BatchTimeout:     5 * time.Millisecond,
	CacheEnabled:     true,
	CacheTTL:         5 * time.Minute,
	CompressionLevel: 5,
	MaxRetries:       3,
	RetryDelay:       100 * time.Millisecond,
}

type APIOptimizer struct {
	config       *APIOptimizationConfig
	queryCache   *APIQueryCache
	batchManager *BatchManager
	perfMonitor  *PerformanceMonitor
	prefetcher   *Prefetcher
	mu           sync.RWMutex
	stats        *OptimizerStats
}

type OptimizerStats struct {
	TotalRequests      atomic.Int64
	CacheHits          atomic.Int64
	CacheMisses        atomic.Int64
	BatchHits          atomic.Int64
	PrefetchHits       atomic.Int64
	AvgLatency         atomic.Int64
	P50Latency         atomic.Int64
	P95Latency         atomic.Int64
	P99Latency         atomic.Int64
	TargetAchieved     atomic.Int64
	Retries            atomic.Int64
	Errors             atomic.Int64
	LastLatencyUpdate  atomic.Value
}

type APIQueryCache struct {
	cache    *sync.Map
	hits     atomic.Int64
	misses   atomic.Int64
	evictions atomic.Int64
	maxSize  int
	mu       sync.RWMutex
}

type APICacheEntry struct {
	Value      []byte
	ExpiresAt  time.Time
	Version    int64
	AccessTime time.Time
	HitCount   int64
}

type BatchManager struct {
	pending  *sync.Map
	queue    chan *BatchRequest
	batchers map[string]*Batcher
	mu       sync.RWMutex
	config   *APIOptimizationConfig
	metrics  *BatchMetrics
}

type BatchRequest struct {
	Key      string
	Promise  chan []byte
	Callback func() ([]byte, error)
}

type Batcher struct {
	key      string
	queue    chan *BatchRequest
	ticker   *time.Ticker
	stopChan chan struct{}
	wg       sync.WaitGroup
}

type BatchMetrics struct {
	TotalBatches    atomic.Int64
	BatchSize       atomic.Int64
	AvgBatchLatency atomic.Int64
	MaxBatchLatency atomic.Int64
}

type Prefetcher struct {
	enabled      bool
	prefetchChan chan *PrefetchRequest
	workerCount  int
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

type PrefetchRequest struct {
	Key      string
	Callback func() ([]byte, error)
}

type PerformanceMonitor struct {
	latencies    []int64
	latenciesMu  sync.Mutex
	requestCount atomic.Int64
	errorCount   atomic.Int64
	startTime    time.Time
}

func NewAPIOptimizer(config *APIOptimizationConfig) *APIOptimizer {
	if config == nil {
		config = DefaultOptimizationConfig
	}

	optimizer := &APIOptimizer{
		config:       config,
		queryCache:   NewAPIQueryCache(10000),
		batchManager: NewBatchManager(config),
		perfMonitor:  NewPerformanceMonitor(),
		stats:        &OptimizerStats{},
	}

	if config.EnablePrefetch {
		optimizer.prefetcher = NewPrefetcher(10)
	}

	return optimizer
}

func NewAPIQueryCache(maxSize int) *APIQueryCache {
	return &APIQueryCache{
		cache:   &sync.Map{},
		maxSize: maxSize,
	}
}

func (qc *APIQueryCache) Get(key string) ([]byte, bool) {
	val, ok := qc.cache.Load(key)
	if !ok {
		qc.misses.Add(1)
		return nil, false
	}

	entry := val.(*APICacheEntry)
	if time.Now().After(entry.ExpiresAt) {
		qc.cache.Delete(key)
		qc.evictions.Add(1)
		qc.misses.Add(1)
		return nil, false
	}

	entry.AccessTime = time.Now()
	entry.HitCount++
	qc.hits.Add(1)
	return entry.Value, true
}

func (qc *APIQueryCache) Set(key string, value []byte, ttl time.Duration) {
	if qc.getSize() >= qc.maxSize {
		qc.evictLRU()
	}

	entry := &APICacheEntry{
		Value:      value,
		ExpiresAt:  time.Now().Add(ttl),
		AccessTime: time.Now(),
		Version:    1,
		HitCount:   0,
	}
	qc.cache.Store(key, entry)
}

func (qc *APIQueryCache) Delete(key string) {
	qc.cache.Delete(key)
}

func (qc *APIQueryCache) Clear() {
	qc.cache = &sync.Map{}
}

func (qc *APIQueryCache) getSize() int {
	size := 0
	qc.cache.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}

func (qc *APIQueryCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	qc.cache.Range(func(key, value interface{}) bool {
		entry := value.(*APICacheEntry)
		if oldestKey == "" || entry.AccessTime.Before(oldestTime) {
			oldestKey = key.(string)
			oldestTime = entry.AccessTime
		}
		return true
	})

	if oldestKey != "" {
		qc.cache.Delete(oldestKey)
		qc.evictions.Add(1)
	}
}

func (qc *APIQueryCache) GetStats() map[string]interface{} {
	hits := qc.hits.Load()
	misses := qc.misses.Load()
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"hits":        hits,
		"misses":      misses,
		"hit_rate":    hitRate,
		"evictions":   qc.evictions.Load(),
		"current_size": qc.getSize(),
		"max_size":    qc.maxSize,
	}
}

func NewBatchManager(config *APIOptimizationConfig) *BatchManager {
	bm := &BatchManager{
		pending:  &sync.Map{},
		queue:    make(chan *BatchRequest, config.BatchSize*10),
		batchers: make(map[string]*Batcher),
		config:   config,
		metrics:  &BatchMetrics{},
	}

	if config.EnableBatch {
		go bm.processQueue()
	}

	return bm
}

func (bm *BatchManager) processQueue() {
	for req := range bm.queue {
		bm.mu.RLock()
		batcher, exists := bm.batchers[req.Key]
		bm.mu.RUnlock()

		if !exists {
			bm.mu.Lock()
			if _, exists := bm.batchers[req.Key]; !exists {
				batcher = &Batcher{
					key:      req.Key,
					queue:    make(chan *BatchRequest, bm.config.BatchSize),
					ticker:   time.NewTicker(bm.config.BatchTimeout),
					stopChan: make(chan struct{}),
				}
				bm.batchers[req.Key] = batcher
				batcher.wg.Add(1)
				go batcher.processBatch(bm)
			}
			bm.mu.Unlock()
		}

		select {
		case batcher.queue <- req:
		default:
			go bm.executeDirectly(req)
		}
	}
}

func (b *Batcher) processBatch(bm *BatchManager) {
	defer b.wg.Done()

	var requests []*BatchRequest
	flush := func() {
		if len(requests) == 0 {
			return
		}

		bm.metrics.TotalBatches.Add(1)
		bm.metrics.BatchSize.Add(int64(len(requests)))

		results := make([][]byte, len(requests))
		for i, req := range requests {
			result, err := req.Callback()
			if err == nil {
				results[i] = result
			}
		}

		for i, req := range requests {
			if results[i] != nil {
				req.Promise <- results[i]
			} else {
				req.Promise <- nil
			}
			close(req.Promise)
		}

		requests = requests[:0]
	}

	for {
		select {
		case req := <-b.queue:
			requests = append(requests, req)
			if len(requests) >= bm.config.BatchSize {
				flush()
			}
		case <-b.ticker.C:
			flush()
		case <-b.stopChan:
			flush()
			return
		}
	}
}

func (bm *BatchManager) Stop() {
	close(bm.queue)
	for _, batcher := range bm.batchers {
		close(batcher.stopChan)
		batcher.wg.Wait()
	}
}

func (bm *BatchManager) AddRequest(key string, callback func() ([]byte, error)) <-chan []byte {
	promise := make(chan []byte, 1)

	req := &BatchRequest{
		Key:      key,
		Promise:  promise,
		Callback: callback,
	}

	select {
	case bm.queue <- req:
	default:
		go bm.executeDirectly(req)
	}

	return promise
}

func (bm *BatchManager) executeDirectly(req *BatchRequest) {
	result, _ := req.Callback()
	req.Promise <- result
	close(req.Promise)
}

func NewPrefetcher(workerCount int) *Prefetcher {
	pf := &Prefetcher{
		enabled:      true,
		prefetchChan: make(chan *PrefetchRequest, 1000),
		workerCount:  workerCount,
		stopChan:     make(chan struct{}),
	}

	for i := 0; i < workerCount; i++ {
		pf.wg.Add(1)
		go pf.worker()
	}

	return pf
}

func (pf *Prefetcher) worker() {
	defer pf.wg.Done()

	for {
		select {
		case req := <-pf.prefetchChan:
			req.Callback()
		case <-pf.stopChan:
			return
		}
	}
}

func (pf *Prefetcher) Prefetch(key string, callback func() ([]byte, error)) {
	if !pf.enabled {
		return
	}

	select {
	case pf.prefetchChan <- &PrefetchRequest{Key: key, Callback: callback}:
	default:
	}
}

func (pf *Prefetcher) Stop() {
	pf.enabled = false
	close(pf.stopChan)
	pf.wg.Wait()
}

func NewPerformanceMonitor() *PerformanceMonitor {
	pm := &PerformanceMonitor{
		latencies: make([]int64, 0, 10000),
		startTime: time.Now(),
	}
	return pm
}

func (pm *PerformanceMonitor) Record(latencyNs int64) {
	pm.latenciesMu.Lock()
	defer pm.latenciesMu.Unlock()

	pm.latencies = append(pm.latencies, latencyNs)

	if len(pm.latencies) > 10000 {
		pm.latencies = pm.latencies[len(pm.latencies)-10000:]
	}

	pm.requestCount.Add(1)
}

func (pm *PerformanceMonitor) RecordError() {
	pm.errorCount.Add(1)
}

func (pm *PerformanceMonitor) GetStats() map[string]interface{} {
	pm.latenciesMu.Lock()
	defer pm.latenciesMu.Unlock()

	if len(pm.latencies) == 0 {
		return map[string]interface{}{
			"avg_latency_ns": 0,
			"p50_latency_ns": 0,
			"p95_latency_ns": 0,
			"p99_latency_ns": 0,
			"total_requests": pm.requestCount.Load(),
			"error_count":    pm.errorCount.Load(),
			"uptime":         time.Since(pm.startTime).String(),
		}
	}

	sorted := make([]int64, len(pm.latencies))
	copy(sorted, pm.latencies)
	quickSort(sorted)

	n := len(sorted)
	avgLatency := int64(0)
	for _, lat := range sorted {
		avgLatency += lat
	}
	avgLatency /= int64(n)

	return map[string]interface{}{
		"avg_latency_ns": avgLatency,
		"p50_latency_ns": sorted[n*50/100],
		"p95_latency_ns": sorted[n*95/100],
		"p99_latency_ns": sorted[n*99/100],
		"total_requests":  pm.requestCount.Load(),
		"error_count":    pm.errorCount.Load(),
		"uptime":         time.Since(pm.startTime).String(),
	}
}

func quickSort(arr []int64) {
	if len(arr) <= 1 {
		return
	}

	pivot := arr[len(arr)/2]
	i := 0
	j := len(arr) - 1

	for i < j {
		for arr[i] < pivot {
			i++
		}
		for arr[j] > pivot {
			j--
		}
		if i < j {
			arr[i], arr[j] = arr[j], arr[i]
			i++
			j--
		}
	}

	quickSort(arr[:i])
	quickSort(arr[i:])
}

func (o *APIOptimizer) Get(ctx context.Context, key string, callback func() ([]byte, error)) ([]byte, error) {
	start := time.Now()
	o.stats.TotalRequests.Add(1)

	if o.config.CacheEnabled {
		if val, ok := o.queryCache.Get(key); ok {
			o.stats.CacheHits.Add(1)
			o.perfMonitor.Record(time.Since(start).Nanoseconds())
			return val, nil
		}
		o.stats.CacheMisses.Add(1)
	}

	result, err := callback()
	if err != nil {
		o.stats.Errors.Add(1)
		o.perfMonitor.RecordError()
		return nil, err
	}

	if o.config.CacheEnabled && result != nil {
		o.queryCache.Set(key, result, o.config.CacheTTL)
	}

	latency := time.Since(start).Nanoseconds()
	o.perfMonitor.Record(latency)
	o.updateLatencyStats(latency)

	return result, nil
}

func (o *APIOptimizer) BatchGet(ctx context.Context, key string, callback func() ([]byte, error)) <-chan []byte {
	o.stats.TotalRequests.Add(1)

	if o.config.EnableBatch {
		o.stats.BatchHits.Add(1)
		return o.batchManager.AddRequest(key, callback)
	}

	result := make(chan []byte, 1)
	go func() {
		data, _ := callback()
		result <- data
		close(result)
	}()
	return result
}

func (o *APIOptimizer) Prefetch(key string, callback func() ([]byte, error)) {
	if o.prefetcher != nil {
		o.stats.PrefetchHits.Add(1)
		o.prefetcher.Prefetch(key, callback)
	}
}

func (o *APIOptimizer) updateLatencyStats(latencyNs int64) {
	o.stats.AvgLatency.Store(latencyNs)

	if o.config.TargetLatency > 0 && latencyNs <= int64(o.config.TargetLatency)*1e6 {
		o.stats.TargetAchieved.Add(1)
	}
}

func (o *APIOptimizer) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_requests": o.stats.TotalRequests.Load(),
		"cache_hits":     o.stats.CacheHits.Load(),
		"cache_misses":   o.stats.CacheMisses.Load(),
		"batch_hits":     o.stats.BatchHits.Load(),
		"prefetch_hits":  o.stats.PrefetchHits.Load(),
		"target_latency": fmt.Sprintf("%dms", o.config.TargetLatency),
		"cache_hit_rate": o.calculateCacheHitRate(),
		"target_achieved_rate": o.calculateTargetAchievedRate(),
	}

	for k, v := range o.queryCache.GetStats() {
		stats["query_cache_"+k] = v
	}

	for k, v := range o.perfMonitor.GetStats() {
		stats["perf_"+k] = v
	}

	return stats
}

func (o *APIOptimizer) calculateCacheHitRate() float64 {
	hits := o.stats.CacheHits.Load()
	misses := o.stats.CacheMisses.Load()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

func (o *APIOptimizer) calculateTargetAchievedRate() float64 {
	total := o.stats.TotalRequests.Load()
	achieved := o.stats.TargetAchieved.Load()
	if total == 0 {
		return 100
	}
	return float64(achieved) / float64(total) * 100
}

func (o *APIOptimizer) ClearCache() {
	o.queryCache.Clear()
}

func (o *APIOptimizer) Stop() {
	if o.prefetcher != nil {
		o.prefetcher.Stop()
	}
	if o.batchManager != nil {
		o.batchManager.Stop()
	}
}

type OptimizedHandler struct {
	optimizer *APIOptimizer
	metrics  *HandlerMetrics
}

type HandlerMetrics struct {
	TotalRequests     atomic.Int64
	SuccessResponses  atomic.Int64
	ErrorResponses   atomic.Int64
	AvgResponseTime  atomic.Int64
	MaxResponseTime  atomic.Int64
	MinResponseTime  atomic.Int64
	SlowRequests     atomic.Int64
	FastRequests     atomic.Int64
}

func NewOptimizedHandler(optimizer *APIOptimizer) *OptimizedHandler {
	return &OptimizedHandler{
		optimizer: optimizer,
		metrics:  &HandlerMetrics{},
	}
}

func (h *OptimizedHandler) HandleRequest(ctx context.Context, key string, callback func() ([]byte, error)) ([]byte, error) {
	h.metrics.TotalRequests.Add(1)
	start := time.Now()

	result, err := h.optimizer.Get(ctx, key, callback)

	latency := time.Since(start)
	latencyMs := latency.Milliseconds()

	if err != nil {
		h.metrics.ErrorResponses.Add(1)
		return nil, err
	}

	h.metrics.SuccessResponses.Add(1)

	oldAvg := h.metrics.AvgResponseTime.Load()
	count := h.metrics.SuccessResponses.Load()
	if count > 0 {
		newAvg := (oldAvg*(count-1) + latencyMs) / count
		h.metrics.AvgResponseTime.Store(newAvg)
	}

	currentMax := h.metrics.MaxResponseTime.Load()
	if latencyMs > currentMax {
		h.metrics.MaxResponseTime.Store(latencyMs)
	}

	currentMin := h.metrics.MinResponseTime.Load()
	if currentMin == 0 || latencyMs < currentMin {
		h.metrics.MinResponseTime.Store(latencyMs)
	}

	if latencyMs > 100 {
		h.metrics.SlowRequests.Add(1)
	} else {
		h.metrics.FastRequests.Add(1)
	}

	return result, nil
}

func (h *OptimizedHandler) GetMetrics() map[string]interface{} {
	total := h.metrics.TotalRequests.Load()
	success := h.metrics.SuccessResponses.Load()
	errorCount := h.metrics.ErrorResponses.Load()
	fast := h.metrics.FastRequests.Load()

	var successRate float64
	if total > 0 {
		successRate = float64(success) / float64(total) * 100
	}

	var fastRate float64
	if total > 0 {
		fastRate = float64(fast) / float64(total) * 100
	}

	return map[string]interface{}{
		"total_requests":    total,
		"success_responses": success,
		"error_responses":   errorCount,
		"success_rate":      successRate,
		"avg_response_time": fmt.Sprintf("%dms", h.metrics.AvgResponseTime.Load()),
		"max_response_time": fmt.Sprintf("%dms", h.metrics.MaxResponseTime.Load()),
		"min_response_time": fmt.Sprintf("%dms", h.metrics.MinResponseTime.Load()),
		"fast_requests":     fast,
		"fast_requests_rate": fastRate,
	}
}

var globalOptimizer *APIOptimizer
var optimizerOnce sync.Once

func InitOptimizer(config *APIOptimizationConfig) {
	optimizerOnce.Do(func() {
		globalOptimizer = NewAPIOptimizer(config)
	})
}

func GetOptimizer() *APIOptimizer {
	if globalOptimizer == nil {
		InitOptimizer(nil)
	}
	return globalOptimizer
}

func StopOptimizer() {
	if globalOptimizer != nil {
		globalOptimizer.Stop()
	}
}
