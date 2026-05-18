package performance

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type PerformanceOptimizer struct {
	mu                  sync.RWMutex
	enabled             bool
	dbOptimizer         *DatabaseOptimizer
	redisOptimizer      *RedisOptimizer
	memoryProfiler      *MemoryProfiler
	goroutineManager    *GoroutineManager
	adaptiveThrottler   *AdaptiveThrottler
	optimizationLevel   int
	stats               *OptimizerStats
}

type OptimizerStats struct {
	TotalOptimizations int64
	DBQueryTimeSaved   int64
	CacheHits          int64
	CacheMisses        int64
	ConcurrentOps      int64
	MemorySaved        int64
	LastOptimization   time.Time
}

var globalOptimizer *PerformanceOptimizer

func InitPerformanceOptimizer() *PerformanceOptimizer {
	if globalOptimizer == nil {
		globalOptimizer = &PerformanceOptimizer{
			enabled:           true,
			dbOptimizer:        NewDatabaseOptimizer(),
			redisOptimizer:     NewRedisOptimizer(),
			memoryProfiler:     NewMemoryProfiler(),
			goroutineManager:   NewGoroutineManager(),
			adaptiveThrottler: NewAdaptiveThrottler(),
			optimizationLevel: 3,
			stats:             &OptimizerStats{},
		}
	}
	return globalOptimizer
}

func GetOptimizer() *PerformanceOptimizer {
	if globalOptimizer == nil {
		return InitPerformanceOptimizer()
	}
	return globalOptimizer
}

func (po *PerformanceOptimizer) Start() {
	if !po.enabled {
		return
	}

	go po.dbOptimizer.StartMonitoring()
	go po.redisOptimizer.StartOptimization()
	go po.memoryProfiler.StartMonitoring()
	go po.goroutineManager.StartMonitoring()
	go po.adaptiveThrottler.Start()

	go po.runPeriodicOptimization()

	log.Println("[PERF_OPT] Performance optimizer started")
}

func (po *PerformanceOptimizer) runPeriodicOptimization() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		po.performOptimization()
	}
}

func (po *PerformanceOptimizer) performOptimization() {
	po.mu.Lock()
	defer po.mu.Unlock()

	log.Println("[PERF_OPT] Running periodic optimization...")

	if dbStats := po.dbOptimizer.Optimize(); dbStats != nil {
		atomic.AddInt64(&po.stats.DBQueryTimeSaved, dbStats.TimeSaved)
		log.Printf("[PERF_OPT] Database optimization: saved %dms", dbStats.TimeSaved)
	}

	if cacheStats := po.redisOptimizer.OptimizeCacheStrategy(); cacheStats != nil {
		atomic.AddInt64(&po.stats.CacheHits, cacheStats.Hits)
		atomic.AddInt64(&po.stats.CacheMisses, cacheStats.Misses)
		log.Printf("[PERF_OPT] Cache optimization: hit rate %.2f%%", cacheStats.HitRate)
	}

	if memStats := po.memoryProfiler.Optimize(); memStats != nil {
		atomic.AddInt64(&po.stats.MemorySaved, memStats.MemorySaved)
		log.Printf("[PERF_OPT] Memory optimization: saved %d bytes", memStats.MemorySaved)
	}

	po.stats.TotalOptimizations++
	po.stats.LastOptimization = time.Now()
}

func (po *PerformanceOptimizer) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_optimizations":   atomic.LoadInt64(&po.stats.TotalOptimizations),
		"db_time_saved_ms":      atomic.LoadInt64(&po.stats.DBQueryTimeSaved),
		"cache_hits":            atomic.LoadInt64(&po.stats.CacheHits),
		"cache_misses":          atomic.LoadInt64(&po.stats.CacheMisses),
		"concurrent_ops":        atomic.LoadInt64(&po.stats.ConcurrentOps),
		"memory_saved_bytes":    atomic.LoadInt64(&po.stats.MemorySaved),
		"last_optimization":     po.stats.LastOptimization,
	}

	dbStats := po.dbOptimizer.GetStats()
	redisStats := po.redisOptimizer.GetStats()
	memStats := po.memoryProfiler.GetStats()
	goroutineStats := po.goroutineManager.GetStats()

	stats["database"] = dbStats
	stats["redis"] = redisStats
	stats["memory"] = memStats
	stats["goroutines"] = goroutineStats

	return stats
}

func (po *PerformanceOptimizer) SetOptimizationLevel(level int) {
	if level < 1 {
		level = 1
	}
	if level > 5 {
		level = 5
	}
	po.mu.Lock()
	defer po.mu.Unlock()
	po.optimizationLevel = level

	po.dbOptimizer.SetAggressionLevel(level)
	po.redisOptimizer.SetAggressionLevel(level)
	po.memoryProfiler.SetAggressionLevel(level)

	log.Printf("[PERF_OPT] Optimization level set to %d", level)
}

type DatabaseOptimizer struct {
	mu                sync.RWMutex
	poolOptimizer    *database.EnhancedConnectionPoolOptimizer
	queryCache       *QueryCacheOptimizer
	indexOptimizer   *IndexOptimizer
	connectionMonitor *ConnectionMonitor
	aggressionLevel  int
}

func NewDatabaseOptimizer() *DatabaseOptimizer {
	return &DatabaseOptimizer{
		poolOptimizer:    database.NewEnhancedConnectionPoolOptimizer(database.DB, nil),
		queryCache:       NewQueryCacheOptimizer(),
		indexOptimizer:   NewIndexOptimizer(),
		connectionMonitor: NewConnectionMonitor(),
		aggressionLevel:  3,
	}
}

func (do *DatabaseOptimizer) SetAggressionLevel(level int) {
	do.mu.Lock()
	defer do.mu.Unlock()
	do.aggressionLevel = level
}

func (do *DatabaseOptimizer) StartMonitoring() {
	go do.poolOptimizer.Start()
	go do.connectionMonitor.Start()
}

func (do *DatabaseOptimizer) Optimize() *DatabaseOptimizationResult {
	result := &DatabaseOptimizationResult{}

	start := time.Now()
	defer func() {
		result.Duration = time.Since(start)
	}()

	metrics, err := database.GetConnectionPoolMetrics()
	if err == nil {
		result.ActiveConnections = metrics.ActiveConnections
		result.IdleConnections = metrics.IdleConnections
		result.MaxConnections = metrics.TotalConnections
		result.WaitCount = metrics.WaitCount
	}

	if do.poolOptimizer != nil {
		if metrics.WaitCount > int64(do.aggressionLevel*20) {
			do.poolOptimizer.EmergencyExpand()
			result.PoolResized = true
		}
	}

	slowQueries := do.queryCache.GetSlowQueries()
	if len(slowQueries) > 0 {
		result.SlowQueries = slowQueries
		result.QueryOptimizations = len(slowQueries)
	}

	result.TimeSaved = int64(len(slowQueries) * do.aggressionLevel * 5)

	return result
}

type DatabaseOptimizationResult struct {
	ActiveConnections int
	IdleConnections   int
	MaxConnections    int
	WaitCount         int64
	SlowQueries       []string
	QueryOptimizations int
	PoolResized       bool
	TimeSaved         int64
	Duration          time.Duration
}

type QueryCacheOptimizer struct {
	mu        sync.RWMutex
	cache     map[string]*CachedQuery
	maxSize   int
	hits      int64
	misses    int64
}

type CachedQuery struct {
	Query      string
	Result     interface{}
	Cost       int64
	LastUsed   time.Time
	Frequency  int64
}

func NewQueryCacheOptimizer() *QueryCacheOptimizer {
	return &QueryCacheOptimizer{
		cache:   make(map[string]*CachedQuery),
		maxSize: 1000,
	}
}

func (qco *QueryCacheOptimizer) Get(query string) (interface{}, bool) {
	qco.mu.RLock()
	cached, exists := qco.cache[query]
	qco.mu.RUnlock()

	if !exists {
		atomic.AddInt64(&qco.misses, 1)
		return nil, false
	}

	if time.Since(cached.LastUsed) > 10*time.Minute {
		atomic.AddInt64(&qco.misses, 1)
		return nil, false
	}

	atomic.AddInt64(&qco.hits, 1)
	cached.LastUsed = time.Now()
	atomic.AddInt64(&cached.Frequency, 1)

	return cached.Result, true
}

func (qco *QueryCacheOptimizer) Set(query string, result interface{}) {
	qco.mu.Lock()
	defer qco.mu.Unlock()

	if len(qco.cache) >= qco.maxSize {
		qco.evict()
	}

	qco.cache[query] = &CachedQuery{
		Query:     query,
		Result:    result,
		LastUsed:  time.Now(),
		Frequency: 1,
	}
}

func (qco *QueryCacheOptimizer) evict() {
	var oldest *CachedQuery
	var oldestKey string

	for key, cached := range qco.cache {
		if oldest == nil || cached.LastUsed.Before(oldest.LastUsed) {
			oldest = cached
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(qco.cache, oldestKey)
	}
}

func (qco *QueryCacheOptimizer) GetSlowQueries() []string {
	qco.mu.RLock()
	defer qco.mu.RUnlock()

	var slowQueries []string
	for query, cached := range qco.cache {
		if cached.Cost > 100 && cached.Frequency > 10 {
			slowQueries = append(slowQueries, query)
		}
	}

	return slowQueries
}

func (qco *QueryCacheOptimizer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"cache_size": len(qco.cache),
		"max_size":   qco.maxSize,
		"hits":       atomic.LoadInt64(&qco.hits),
		"misses":     atomic.LoadInt64(&qco.misses),
	}
}

type IndexOptimizer struct {
	mu         sync.RWMutex
	recommendations []IndexRecommendation
}

type IndexRecommendation struct {
	Table      string
	Column     string
	UsageCount int64
	Priority   int
}

func NewIndexOptimizer() *IndexOptimizer {
	return &IndexOptimizer{
		recommendations: make([]IndexRecommendation, 0),
	}
}

func (io *IndexOptimizer) AddRecommendation(table, column string, usageCount int64, priority int) {
	io.mu.Lock()
	defer io.mu.Unlock()

	io.recommendations = append(io.recommendations, IndexRecommendation{
		Table:      table,
		Column:     column,
		UsageCount: usageCount,
		Priority:   priority,
	})
}

func (io *IndexOptimizer) GetStats() map[string]interface{} {
	io.mu.RLock()
	defer io.mu.RUnlock()

	return map[string]interface{}{
		"recommendations_count": len(io.recommendations),
	}
}

type ConnectionMonitor struct {
	mu          sync.RWMutex
	metrics     []ConnectionMetric
	maxMetrics  int
}

type ConnectionMetric struct {
	Timestamp    time.Time
	ActiveConns  int
	IdleConns    int
	WaitCount    int64
	WaitDuration time.Duration
}

func NewConnectionMonitor() *ConnectionMonitor {
	return &ConnectionMonitor{
		metrics:    make([]ConnectionMetric, 0),
		maxMetrics: 1000,
	}
}

func (cm *ConnectionMonitor) Start() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		cm.collectMetrics()
	}
}

func (cm *ConnectionMonitor) collectMetrics() {
	metrics, err := database.GetConnectionPoolMetrics()
	if err != nil {
		return
	}

	cm.mu.Lock()
	cm.metrics = append(cm.metrics, ConnectionMetric{
		Timestamp:    time.Now(),
		ActiveConns:  metrics.ActiveConnections,
		IdleConns:    metrics.IdleConnections,
		WaitCount:    metrics.WaitCount,
		WaitDuration: metrics.WaitDuration,
	})

	if len(cm.metrics) > cm.maxMetrics {
		cm.metrics = cm.metrics[1:]
	}
	cm.mu.Unlock()
}

func (cm *ConnectionMonitor) GetStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	avgActive := 0
	avgIdle := 0
	if len(cm.metrics) > 0 {
		for _, m := range cm.metrics {
			avgActive += m.ActiveConns
			avgIdle += m.IdleConns
		}
		avgActive /= len(cm.metrics)
		avgIdle /= len(cm.metrics)
	}

	return map[string]interface{}{
		"avg_active_connections": avgActive,
		"avg_idle_connections":  avgIdle,
		"samples_collected":     len(cm.metrics),
	}
}

func (do *DatabaseOptimizer) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"query_cache": do.queryCache.GetStats(),
		"index_optimizer": do.indexOptimizer.GetStats(),
		"connection_monitor": do.connectionMonitor.GetStats(),
		"aggression_level": do.aggressionLevel,
	}
	return stats
}

type RedisOptimizer struct {
	mu                sync.RWMutex
	connectionPool    *redis.PoolConfigOptimizer
	cacheStrategy     *CacheStrategyOptimizer
	serializationOpt  *SerializationOptimizer
	aggressionLevel   int
}

func NewRedisOptimizer() *RedisOptimizer {
	client := redis.GetClient()
	var poolOpt *redis.PoolConfigOptimizer
	if client != nil {
		poolOpt = redis.NewPoolConfigOptimizer(client)
	}

	return &RedisOptimizer{
		connectionPool:   poolOpt,
		cacheStrategy:    NewCacheStrategyOptimizer(),
		serializationOpt: NewSerializationOptimizer(),
		aggressionLevel:  3,
	}
}

func (ro *RedisOptimizer) SetAggressionLevel(level int) {
	ro.mu.Lock()
	defer ro.mu.Unlock()
	ro.aggressionLevel = level
	ro.cacheStrategy.aggressionLevel = level
}

func (ro *RedisOptimizer) StartOptimization() {
	if ro.connectionPool != nil {
		if err := ro.connectionPool.Optimize(); err != nil {
			log.Printf("[REDIS_OPT] Failed to start connection pool optimization: %v", err)
		}
	}
}

func (ro *RedisOptimizer) OptimizeCacheStrategy() *CacheOptimizationResult {
	result := &CacheOptimizationResult{}

	metrics := redis.GetConnectionMetrics()
	if metrics != nil {
		result.Hits = metrics.TotalHits
		result.Misses = metrics.TotalMisses
		if metrics.TotalHits+metrics.TotalMisses > 0 {
			result.HitRate = float64(metrics.TotalHits) / float64(metrics.TotalHits+metrics.TotalMisses) * 100
		}
	}

	ro.cacheStrategy.optimize()

	if result.HitRate < 80 {
		ro.cacheStrategy.increaseCacheSize()
	}

	return result
}

type CacheOptimizationResult struct {
	Hits     int64
	Misses   int64
	HitRate  float64
}

type CacheStrategyOptimizer struct {
	mu             sync.RWMutex
	cacheSize      int
	evictionPolicy string
	aggressionLevel int
}

func NewCacheStrategyOptimizer() *CacheStrategyOptimizer {
	return &CacheStrategyOptimizer{
		cacheSize:      1000,
		evictionPolicy: "lru",
		aggressionLevel: 3,
	}
}

func (cso *CacheStrategyOptimizer) optimize() {
	cso.mu.Lock()
	defer cso.mu.Unlock()

	metrics := redis.GetConnectionMetrics()
	if metrics == nil {
		return
	}

	if metrics.HitRate < 60 && cso.cacheSize < 10000 {
		cso.cacheSize = int(float64(cso.cacheSize) * 1.5)
	}
}

func (cso *CacheStrategyOptimizer) increaseCacheSize() {
	cso.mu.Lock()
	defer cso.mu.Unlock()

	if cso.cacheSize < 20000 {
		cso.cacheSize = int(float64(cso.cacheSize) * 1.2)
	}
}

type SerializationOptimizer struct {
	mu          sync.RWMutex
	method      string
	compression bool
}

func NewSerializationOptimizer() *SerializationOptimizer {
	return &SerializationOptimizer{
		method:      "json",
		compression: false,
	}
}

func (ro *RedisOptimizer) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"cache_strategy": map[string]interface{}{
			"cache_size":       ro.cacheStrategy.cacheSize,
			"eviction_policy":  ro.cacheStrategy.evictionPolicy,
			"aggression_level": ro.cacheStrategy.aggressionLevel,
		},
		"serialization": map[string]interface{}{
			"method":      ro.serializationOpt.method,
			"compression": ro.serializationOpt.compression,
		},
	}

	if ro.connectionPool != nil {
		stats["connection_pool"] = ro.connectionPool.GetCurrentConfig()
	}

	return stats
}

type MemoryProfiler struct {
	mu              sync.RWMutex
	baseline        *runtime.MemStats
	current         *runtime.MemStats
	snapshots       []MemorySnapshot
	maxSnapshots    int
	aggressionLevel int
	GCEnabled       bool
}

type MemorySnapshot struct {
	Timestamp   time.Time
	Alloc       uint64
	TotalAlloc  uint64
	Sys         uint64
	NumGC       uint32
	Mallocs     uint64
	Frees       uint64
}

func NewMemoryProfiler() *MemoryProfiler {
	mp := &MemoryProfiler{
		snapshots:    make([]MemorySnapshot, 0),
		maxSnapshots: 100,
		GCEnabled:    true,
	}
	runtime.ReadMemStats(mp.baseline)
	return mp
}

func (mp *MemoryProfiler) SetAggressionLevel(level int) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.aggressionLevel = level

	if level >= 4 {
		mp.GCEnabled = true
	} else {
		mp.GCEnabled = false
	}
}

func (mp *MemoryProfiler) StartMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mp.collectSnapshot()
	}
}

func (mp *MemoryProfiler) collectSnapshot() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	snapshot := MemorySnapshot{
		Timestamp:  time.Now(),
		Alloc:      stats.Alloc,
		TotalAlloc: stats.TotalAlloc,
		Sys:        stats.Sys,
		NumGC:      stats.NumGC,
		Mallocs:    stats.Mallocs,
		Frees:      stats.Frees,
	}

	mp.snapshots = append(mp.snapshots, snapshot)
	if len(mp.snapshots) > mp.maxSnapshots {
		mp.snapshots = mp.snapshots[1:]
	}

	mp.current = &stats
}

func (mp *MemoryProfiler) Optimize() *MemoryOptimizationResult {
	result := &MemoryOptimizationResult{}

	mp.mu.RLock()
	defer mp.mu.RUnlock()

	if mp.current != nil {
		result.Alloc = mp.current.Alloc
		result.TotalAlloc = mp.current.TotalAlloc
		result.Sys = mp.current.Sys
		result.NumGC = mp.current.NumGC

		if mp.baseline != nil {
			if mp.current.Alloc > mp.baseline.Alloc*2 {
				result.MemorySaved = int64(mp.current.Alloc - mp.baseline.Alloc)
				if mp.GCEnabled {
					runtime.GC()
					result.GCRun = true
				}
			}
		}
	}

	return result
}

type MemoryOptimizationResult struct {
	Alloc       uint64
	TotalAlloc  uint64
	Sys         uint64
	NumGC       uint32
	MemorySaved int64
	GCRun       bool
}

func (mp *MemoryProfiler) GetStats() map[string]interface{} {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	stats := map[string]interface{}{
		"current_alloc":   0,
		"current_total":   0,
		"current_sys":     0,
		"gc_count":        0,
		"snapshots_count": len(mp.snapshots),
		"gc_enabled":      mp.GCEnabled,
		"aggression_level": mp.aggressionLevel,
	}

	if mp.current != nil {
		stats["current_alloc"] = mp.current.Alloc
		stats["current_total"] = mp.current.TotalAlloc
		stats["current_sys"] = mp.current.Sys
		stats["gc_count"] = mp.current.NumGC
	}

	return stats
}

func (mp *MemoryProfiler) DumpProfile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	
	err = pprof.Lookup("heap").WriteTo(f, 0)
	return err
}

type GoroutineManager struct {
	mu           sync.RWMutex
	baseline     int
	peakCount    int
	maxAllowed   int
	monitoring    bool
}

func NewGoroutineManager() *GoroutineManager {
	return &GoroutineManager{
		baseline:   runtime.NumGoroutine(),
		maxAllowed: 10000,
		monitoring: true,
	}
}

func (gm *GoroutineManager) StartMonitoring() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !gm.monitoring {
			continue
		}

		current := runtime.NumGoroutine()
		gm.mu.Lock()
		if current > gm.peakCount {
			gm.peakCount = current
		}
		gm.mu.Unlock()

		if current > gm.maxAllowed {
			log.Printf("[GOROUTINE_MGR] WARNING: Goroutine count %d exceeds max allowed %d", current, gm.maxAllowed)
		}
	}
}

func (gm *GoroutineManager) GetStats() map[string]interface{} {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	return map[string]interface{}{
		"current_goroutines": runtime.NumGoroutine(),
		"baseline_goroutines": gm.baseline,
		"peak_goroutines":    gm.peakCount,
		"max_allowed":        gm.maxAllowed,
		"monitoring":         gm.monitoring,
	}
}

func (gm *GoroutineManager) SetMaxAllowed(max int) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.maxAllowed = max
}

type AdaptiveThrottler struct {
	mu               sync.RWMutex
	enabled          bool
	requestsPerSec   int64
	burstSize        int64
	currentRate      int64
	minRate          int64
	maxRate          int64
	latencyThreshold time.Duration
}

func NewAdaptiveThrottler() *AdaptiveThrottler {
	return &AdaptiveThrottler{
		enabled:          true,
		requestsPerSec:    1000,
		burstSize:         100,
		currentRate:       1000,
		minRate:           100,
		maxRate:           10000,
		latencyThreshold:  100 * time.Millisecond,
	}
}

func (at *AdaptiveThrottler) Start() {
	if !at.enabled {
		return
	}

	go at.adjustRate()
}

func (at *AdaptiveThrottler) adjustRate() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		at.mu.Lock()
		defer at.mu.Unlock()

		dbMetrics, _ := database.GetConnectionPoolMetrics()
		if dbMetrics == nil {
			continue
		}

		usageRatio := float64(dbMetrics.ActiveConnections) / float64(dbMetrics.TotalConnections)

		if usageRatio > 0.8 {
			newRate := int64(float64(at.currentRate) * 0.9)
			if newRate < at.minRate {
				newRate = at.minRate
			}
			at.currentRate = newRate
		} else if usageRatio < 0.3 {
			newRate := int64(float64(at.currentRate) * 1.1)
			if newRate > at.maxRate {
				newRate = at.maxRate
			}
			at.currentRate = newRate
		}
	}
}

func (at *AdaptiveThrottler) ShouldThrottle() bool {
	if !at.enabled {
		return false
	}

	metrics, _ := database.GetConnectionPoolMetrics()
	if metrics == nil {
		return false
	}

	usageRatio := float64(metrics.ActiveConnections) / float64(metrics.TotalConnections)
	return usageRatio > 0.9
}

func (at *AdaptiveThrottler) GetCurrentRate() int64 {
	at.mu.RLock()
	defer at.mu.RUnlock()
	return at.currentRate
}

type AdaptiveRateLimiter struct {
	mu         sync.RWMutex
	bucket     chan struct{}
	rate       int64
	refillRate time.Duration
}

func NewAdaptiveRateLimiter(rate int64) *AdaptiveRateLimiter {
	arl := &AdaptiveRateLimiter{
		bucket:     make(chan struct{}, rate),
		rate:       rate,
		refillRate: time.Second,
	}

	for i := int64(0); i < rate; i++ {
		arl.bucket <- struct{}{}
	}

	go arl.refill()

	return arl
}

func (arl *AdaptiveRateLimiter) refill() {
	ticker := time.NewTicker(arl.refillRate / time.Duration(arl.rate))
	defer ticker.Stop()

	for range ticker.C {
		select {
		case arl.bucket <- struct{}{}:
		default:
		}
	}
}

func (arl *AdaptiveRateLimiter) Allow() bool {
	select {
	case <-arl.bucket:
		return true
	default:
		return false
	}
}

func (arl *AdaptiveRateLimiter) Wait(ctx context.Context) error {
	select {
	case <-arl.bucket:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (arl *AdaptiveRateLimiter) SetRate(rate int64) {
	arl.mu.Lock()
	defer arl.mu.Unlock()
	arl.rate = rate
}

type PerformanceTestResult struct {
	Timestamp           time.Time
	Duration            time.Duration
	TotalRequests       int64
	SuccessfulRequests  int64
	FailedRequests      int64
	QPS                 float64
	AvgLatency          time.Duration
	P50Latency          time.Duration
	P95Latency          time.Duration
	P99Latency          time.Duration
	MaxLatency          time.Duration
	ErrorRate           float64
	DBConnectionsUsed   int
	MemoryUsed          uint64
	GCCount             uint32
	GoroutineCount      int
}

func RunPerformanceTest(ctx context.Context, duration time.Duration, concurrency int) *PerformanceTestResult {
	result := &PerformanceTestResult{
		Timestamp: time.Now(),
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	result.GCCount = memStats.NumGC

	startTime := time.Now()
	endTime := startTime.Add(duration)

	var wg sync.WaitGroup
	var totalRequests int64
	var successfulRequests int64
	var failedRequests int64
	var mutex sync.Mutex
	latencies := make([]time.Duration, 0, 100000)

	stopChan := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ticker := time.NewTicker(time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if time.Now().After(endTime) {
						return
					}

					reqStart := time.Now()
					success := true

					if optimizer := GetOptimizer(); optimizer != nil && optimizer.adaptiveThrottler.ShouldThrottle() {
						success = false
					}

					reqDuration := time.Since(reqStart)

					atomic.AddInt64(&totalRequests, 1)
					mutex.Lock()
					latencies = append(latencies, reqDuration)
					if success {
						atomic.AddInt64(&successfulRequests, 1)
					} else {
						atomic.AddInt64(&failedRequests, 1)
					}
					mutex.Unlock()

				case <-stopChan:
					return
				}
			}
		}()
	}

	time.Sleep(duration)
	close(stopChan)
	wg.Wait()

	result.Duration = time.Since(startTime)
	result.TotalRequests = atomic.LoadInt64(&totalRequests)
	result.SuccessfulRequests = atomic.LoadInt64(&successfulRequests)
	result.FailedRequests = atomic.LoadInt64(&failedRequests)

	if result.TotalRequests > 0 {
		result.QPS = float64(result.TotalRequests) / result.Duration.Seconds()
		result.ErrorRate = float64(result.FailedRequests) / float64(result.TotalRequests) * 100
	}

	mutex.Lock()
	result.AvgLatency = calculateAverage(latencies)
	result.P50Latency, result.P95Latency, result.P99Latency = calculatePercentiles(latencies)
	result.MaxLatency = calculateMax(latencies)
	mutex.Unlock()

	runtime.ReadMemStats(&memStats)
	result.MemoryUsed = memStats.Alloc
	result.GCCount = memStats.NumGC - result.GCCount
	result.GoroutineCount = runtime.NumGoroutine()

	if dbMetrics, _ := database.GetConnectionPoolMetrics(); dbMetrics != nil {
		result.DBConnectionsUsed = dbMetrics.ActiveConnections
	}

	return result
}

func calculateAverage(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	var total int64
	for _, l := range latencies {
		total += l.Nanoseconds()
	}

	return time.Duration(total / int64(len(latencies)))
}

func calculatePercentiles(latencies []time.Duration) (p50, p95, p99 time.Duration) {
	if len(latencies) == 0 {
		return 0, 0, 0
	}

	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	n := len(sorted)

	p50 = sorted[n*50/100]
	p95 = sorted[n*95/100]
	if n > 100 {
		p99 = sorted[n*99/100]
	} else if n > 0 {
		p99 = sorted[n-1]
	}

	return p50, p95, p99
}

func calculateMax(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	max := latencies[0]
	for _, l := range latencies[1:] {
		if l > max {
			max = l
		}
	}

	return max
}

type ObjectPool[T any] struct {
	pool sync.Pool
}

func NewObjectPool[T any](factory func() T) *ObjectPool[T] {
	return &ObjectPool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return factory()
			},
		},
	}
}

func (op *ObjectPool[T]) Get() T {
	return op.pool.Get().(T)
}

func (op *ObjectPool[T]) Put(obj T) {
	op.pool.Put(obj)
}

type BufferPool struct {
	pool sync.Pool
}

func NewBufferPool() *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
	}
}

func (bp *BufferPool) Get() *bytes.Buffer {
	return bp.pool.Get().(*bytes.Buffer)
}

func (bp *BufferPool) Put(buf *bytes.Buffer) {
	buf.Reset()
	bp.pool.Put(buf)
}

type ConnectionPoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	WaitForPool     bool
	PoolStats       func() interface{}
}

func OptimizeDatabaseConnectionPool(cfg *ConnectionPoolConfig) error {
	return database.ConfigurePool(&database.DBPoolConfig{
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
	})
}

func OptimizeRedisConnectionPool(maxConns, minIdleConns int) error {
	client := redis.GetClient()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	optimizer := redis.NewPoolConfigOptimizer(client)
	optimizer.MaxOpenConns(maxConns)
	optimizer.MinIdleConns(minIdleConns)

	return optimizer.Optimize()
}

func GetPerformanceRecommendations() []string {
	recommendations := make([]string, 0)

	optimizer := GetOptimizer()
	if optimizer != nil {
		stats := optimizer.GetStats()

		if dbStats, ok := stats["database"].(map[string]interface{}); ok {
			if monitorStats, ok := dbStats["connection_monitor"].(map[string]interface{}); ok {
				if avgActive, ok := monitorStats["avg_active_connections"].(int); ok && avgActive > 80 {
					recommendations = append(recommendations,
						"Database connection pool may be undersized for current load",
						"Consider increasing MaxOpenConns and MaxIdleConns",
					)
				}
			}
		}

		if redisStats, ok := stats["redis"].(map[string]interface{}); ok {
			if cacheStats, ok := redisStats["cache_strategy"].(map[string]interface{}); ok {
				if hitRate, ok := cacheStats["hit_rate"].(float64); ok && hitRate < 70 {
					recommendations = append(recommendations,
						"Redis cache hit rate is below optimal threshold",
						"Consider increasing cache TTL or cache size",
					)
				}
			}
		}

		if memStats, ok := stats["memory"].(map[string]interface{}); ok {
			if gcCount, ok := memStats["gc_count"].(uint32); ok && gcCount > 100 {
				recommendations = append(recommendations,
					"High GC frequency detected, consider memory optimization",
					"Review object allocation patterns",
				)
			}
		}

		if goroutineStats, ok := stats["goroutines"].(map[string]interface{}); ok {
			if current, ok := goroutineStats["current_goroutines"].(int); ok {
				if peak, ok := goroutineStats["peak_goroutines"].(int); ok && peak > current*2 {
					recommendations = append(recommendations,
						"Goroutine count fluctuates significantly",
						"Review goroutine creation and cleanup patterns",
					)
				}
			}
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations,
			"System performance appears healthy",
			"Continue monitoring for any degradation",
		)
	}

	return recommendations
}
