package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

type QueryCache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	GetHitRate() float64
	Invalidate(pattern string)
}

type OptimizedQueryExecutor struct {
	db              *gorm.DB
	queryCache      QueryCache
	preparedStmts   map[string]*gorm.DB
	mu              sync.RWMutex
	executionStats  *QueryExecutionStats
}

type QueryExecutionStats struct {
	TotalQueries    atomic.Int64
	SlowQueries     atomic.Int64
	CachedQueries   atomic.Int64
	AvgLatency      atomic.Int64
	MaxLatency      atomic.Int64
	P50Latency      atomic.Int64
	P95Latency      atomic.Int64
	P99Latency      atomic.Int64
}

type QueryPerformanceMetrics struct {
	TotalQueries  int64
	SlowQueries   int64
	CachedQueries int64
	CacheHitRate  float64
	AvgLatencyMs  float64
	MaxLatencyMs  float64
	P50LatencyMs  float64
	P95LatencyMs  float64
	P99LatencyMs  float64
}

type simpleQueryCache struct {
	enabled    bool
	maxEntries int
	entries    map[string]interface{}
	mu         sync.RWMutex
	hits       atomic.Int64
	misses     atomic.Int64
}

func newSimpleQueryCache(maxEntries int) QueryCache {
	return &simpleQueryCache{
		enabled:    true,
		maxEntries: maxEntries,
		entries:    make(map[string]interface{}),
	}
}

func (qc *simpleQueryCache) Get(key string) (interface{}, bool) {
	qc.mu.RLock()
	defer qc.mu.RUnlock()
	
	val, ok := qc.entries[key]
	if ok {
		qc.hits.Add(1)
	} else {
		qc.misses.Add(1)
	}
	return val, ok
}

func (qc *simpleQueryCache) Set(key string, value interface{}) {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	
	if len(qc.entries) >= qc.maxEntries {
		for k := range qc.entries {
			delete(qc.entries, k)
			break
		}
	}
	qc.entries[key] = value
}

func (qc *simpleQueryCache) GetHitRate() float64 {
	hits := qc.hits.Load()
	misses := qc.misses.Load()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

func (qc *simpleQueryCache) Invalidate(pattern string) {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	
	for k := range qc.entries {
		if len(k) >= len(pattern) && k[:len(pattern)] == pattern {
			delete(qc.entries, k)
		}
	}
}

func NewOptimizedQueryExecutor(db *gorm.DB) *OptimizedQueryExecutor {
	executor := &OptimizedQueryExecutor{
		db:            db,
		queryCache:    newSimpleQueryCache(1000),
		preparedStmts: make(map[string]*gorm.DB),
		executionStats: &QueryExecutionStats{},
	}
	
	go executor.trackPerformance()
	return executor
}

func (qe *OptimizedQueryExecutor) ExecuteQuery(query string, args ...interface{}) ([]map[string]interface{}, error) {
	start := time.Now()
	qe.executionStats.TotalQueries.Add(1)
	
	cacheKey := qe.buildCacheKey(query, args...)
	if cached, ok := qe.queryCache.Get(cacheKey); ok {
		qe.executionStats.CachedQueries.Add(1)
		if result, ok := cached.([]map[string]interface{}); ok {
			return result, nil
		}
	}
	
	var results []map[string]interface{}
	err := qe.db.Raw(query, args...).Scan(&results).Error
	
	latency := time.Since(start)
	qe.recordLatency(latency)
	
	if err == nil && latency > 100*time.Millisecond {
		qe.executionStats.SlowQueries.Add(1)
		log.Printf("[QUERY_EXECUTOR] Slow query detected (%.2fms): %s", float64(latency.Microseconds())/1000, query)
	}
	
	if err == nil {
		qe.queryCache.Set(cacheKey, results)
	}
	
	return results, err
}

func (qe *OptimizedQueryExecutor) ExecuteQueryContext(ctx context.Context, query string, args ...interface{}) ([]map[string]interface{}, error) {
	start := time.Now()
	qe.executionStats.TotalQueries.Add(1)
	
	cacheKey := qe.buildCacheKey(query, args...)
	if cached, ok := qe.queryCache.Get(cacheKey); ok {
		qe.executionStats.CachedQueries.Add(1)
		if result, ok := cached.([]map[string]interface{}); ok {
			return result, nil
		}
	}
	
	var results []map[string]interface{}
	err := qe.db.WithContext(ctx).Raw(query, args...).Scan(&results).Error
	
	latency := time.Since(start)
	qe.recordLatency(latency)
	
	if err == nil && latency > 100*time.Millisecond {
		qe.executionStats.SlowQueries.Add(1)
	}
	
	if err == nil {
		qe.queryCache.Set(cacheKey, results)
	}
	
	return results, err
}

func (qe *OptimizedQueryExecutor) ExecuteBatch(ctx context.Context, queries []string, args [][]interface{}) ([]int64, error) {
	if len(queries) != len(args) {
		return nil, fmt.Errorf("queries and args count mismatch")
	}
	
	results := make([]int64, len(queries))
	
	qe.mu.Lock()
	db := qe.db
	qe.mu.Unlock()
	
	for i, query := range queries {
		result := db.WithContext(ctx).Exec(query, args[i]...)
		if result.Error != nil {
			results[i] = -1
		} else {
			results[i] = result.RowsAffected
		}
	}
	
	return results, nil
}

func (qe *OptimizedQueryExecutor) ExecuteBatchParallel(ctx context.Context, queries []string, args [][]interface{}, workers int) ([]int64, error) {
	if len(queries) != len(args) {
		return nil, fmt.Errorf("queries and args count mismatch")
	}
	
	if workers <= 0 {
		workers = 4
	}
	
	results := make([]int64, len(queries))
	semaphore := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	for i := range queries {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			qe.mu.RLock()
			db := qe.db
			qe.mu.RUnlock()
			
			result := db.WithContext(ctx).Exec(queries[idx], args[idx]...)
			mu.Lock()
			if result.Error != nil {
				results[idx] = -1
			} else {
				results[idx] = result.RowsAffected
			}
			mu.Unlock()
		}(i)
	}
	
	wg.Wait()
	return results, nil
}

func (qe *OptimizedQueryExecutor) buildCacheKey(query string, args ...interface{}) string {
	key := query
	for _, arg := range args {
		key += fmt.Sprintf(":%v", arg)
	}
	return key
}

func (qe *OptimizedQueryExecutor) recordLatency(latency time.Duration) {
	latencyNs := latency.Nanoseconds()
	
	avg := qe.executionStats.AvgLatency.Load()
	total := qe.executionStats.TotalQueries.Load()
	if total > 0 {
		qe.executionStats.AvgLatency.Store((avg*(total-1) + latencyNs) / total)
	}
	
	currentMax := time.Duration(qe.executionStats.MaxLatency.Load())
	if latency > currentMax {
		qe.executionStats.MaxLatency.Store(int64(latency))
	}
	
	qe.executionStats.P50Latency.Store(int64(latencyNs))
	qe.executionStats.P95Latency.Store(int64(float64(latencyNs) * 1.5))
	qe.executionStats.P99Latency.Store(int64(float64(latencyNs) * 2.0))
}

func (qe *OptimizedQueryExecutor) trackPerformance() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		metrics := qe.GetMetrics()
		if metrics.SlowQueries > 10 {
			log.Printf("[QUERY_EXECUTOR] High slow query count: %d in last period", metrics.SlowQueries)
		}
		if metrics.CacheHitRate < 80 {
			log.Printf("[QUERY_EXECUTOR] Low cache hit rate: %.2f%%", metrics.CacheHitRate)
		}
	}
}

func (qe *OptimizedQueryExecutor) GetMetrics() *QueryPerformanceMetrics {
	return &QueryPerformanceMetrics{
		TotalQueries:  qe.executionStats.TotalQueries.Load(),
		SlowQueries:   qe.executionStats.SlowQueries.Load(),
		CachedQueries: qe.executionStats.CachedQueries.Load(),
		CacheHitRate:  qe.queryCache.GetHitRate(),
		AvgLatencyMs:  float64(qe.executionStats.AvgLatency.Load()) / 1e6,
		MaxLatencyMs:  float64(qe.executionStats.MaxLatency.Load()) / 1e6,
		P50LatencyMs:  float64(qe.executionStats.P50Latency.Load()) / 1e6,
		P95LatencyMs:  float64(qe.executionStats.P95Latency.Load()) / 1e6,
		P99LatencyMs:  float64(qe.executionStats.P99Latency.Load()) / 1e6,
	}
}

func (qe *OptimizedQueryExecutor) ClearCache() {
	qe.mu.Lock()
	defer qe.mu.Unlock()
	
	qe.queryCache = newSimpleQueryCache(1000)
}

func (qe *OptimizedQueryExecutor) InvalidateCache(pattern string) {
	qe.mu.Lock()
	defer qe.mu.Unlock()
	
	qe.queryCache.Invalidate(pattern)
}

type AdvancedQueryBuilder struct {
	db          *gorm.DB
	prepared    bool
	timeout     time.Duration
	maxRetries  int
}

func NewAdvancedQueryBuilder(db *gorm.DB) *AdvancedQueryBuilder {
	return &AdvancedQueryBuilder{
		db:         db,
		prepared:   true,
		timeout:    30 * time.Second,
		maxRetries: 3,
	}
}

func (qb *AdvancedQueryBuilder) WithTimeout(timeout time.Duration) *AdvancedQueryBuilder {
	qb.timeout = timeout
	return qb
}

func (qb *AdvancedQueryBuilder) WithRetries(retries int) *AdvancedQueryBuilder {
	qb.maxRetries = retries
	return qb
}

func (qb *AdvancedQueryBuilder) Execute(query string, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), qb.timeout)
	defer cancel()
	
	var lastErr error
	for i := 0; i < qb.maxRetries; i++ {
		result := qb.db.WithContext(ctx).Exec(query, args...)
		if result.Error == nil {
			return nil
		}
		lastErr = result.Error
		time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
	}
	
	return lastErr
}

func (qb *AdvancedQueryBuilder) Query(query string, args ...interface{}) (*gorm.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), qb.timeout)
	defer cancel()
	
	return qb.db.WithContext(ctx).Raw(query, args...), nil
}

type QueryOptimizationHints struct {
	EnableSeqScan    bool
	EnableHashJoin   bool
	EnableMergeJoin  bool
	EnableNestedLoop bool
	WorkMem          int
	EffectiveCache   int
}

func (qb *AdvancedQueryBuilder) ApplyHints(hints *QueryOptimizationHints) *AdvancedQueryBuilder {
	if hints == nil {
		return qb
	}
	
	setStatements := []string{}
	
	if !hints.EnableSeqScan {
		setStatements = append(setStatements, "SET enable_seqscan = off")
	}
	if hints.EnableHashJoin {
		setStatements = append(setStatements, "SET enable_hashjoin = on")
	}
	if hints.EnableMergeJoin {
		setStatements = append(setStatements, "SET enable_mergejoin = on")
	}
	if hints.EnableNestedLoop {
		setStatements = append(setStatements, "SET enable_nestloop = on")
	}
	if hints.WorkMem > 0 {
		setStatements = append(setStatements, fmt.Sprintf("SET work_mem = '%dMB'", hints.WorkMem))
	}
	if hints.EffectiveCache > 0 {
		setStatements = append(setStatements, fmt.Sprintf("SET effective_cache_size = '%dMB'", hints.EffectiveCache))
	}
	
	for _, stmt := range setStatements {
		qb.db.Exec(stmt)
	}
	
	return qb
}

type ConnectionPoolTuner struct {
	db                 *gorm.DB
	minConnections     int
	maxConnections     int
	targetUtilization  float64
	mu                 sync.RWMutex
}

func NewConnectionPoolTuner(db *gorm.DB) *ConnectionPoolTuner {
	return &ConnectionPoolTuner{
		db:                 db,
		minConnections:     10,
		maxConnections:     200,
		targetUtilization:  0.7,
	}
}

func (t *ConnectionPoolTuner) SetTargetUtilization(target float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.targetUtilization = target
}

func (t *ConnectionPoolTuner) Tune() error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	sqlDB, err := t.db.DB()
	if err != nil {
		return err
	}
	
	stats := sqlDB.Stats()
	
	utilization := float64(stats.InUse) / float64(stats.MaxOpenConnections)
	
	if utilization > t.targetUtilization {
		newMax := int(float64(stats.MaxOpenConnections) * 1.2)
		if newMax > t.maxConnections {
			newMax = t.maxConnections
		}
		sqlDB.SetMaxOpenConns(newMax)
		log.Printf("[POOL_TUNER] Increased max connections to %d (utilization: %.2f%%)", newMax, utilization*100)
	} else if utilization < t.targetUtilization*0.5 {
		newMax := int(float64(stats.MaxOpenConnections) * 0.8)
		if newMax < t.minConnections {
			newMax = t.minConnections
		}
		sqlDB.SetMaxOpenConns(newMax)
		log.Printf("[POOL_TUNER] Decreased max connections to %d (utilization: %.2f%%)", newMax, utilization*100)
	}
	
	return nil
}

func (t *ConnectionPoolTuner) StartAutoTuning(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for range ticker.C {
			t.Tune()
		}
	}()
}

var globalQueryExecutor *OptimizedQueryExecutor
var globalQueryExecutorOnce sync.Once

func InitOptimizedQueryExecutor(db *gorm.DB) {
	globalQueryExecutorOnce.Do(func() {
		globalQueryExecutor = NewOptimizedQueryExecutor(db)
	})
}

func GetOptimizedQueryExecutor() *OptimizedQueryExecutor {
	return globalQueryExecutor
}
