package performance

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type MetricsCollector struct {
	mu sync.RWMutex

	requestCounter    *prometheus.CounterVec
	latencyHistogram  *prometheus.HistogramVec
	activeRequests    prometheus.Gauge
	cacheHitCounter   *prometheus.CounterVec
	dbQueryCounter    *prometheus.CounterVec
	dbQueryDuration   *prometheus.HistogramVec
	redisOpCounter    *prometheus.CounterVec
	redisOpDuration   *prometheus.HistogramVec

	stats *MetricsStats
}

type MetricsStats struct {
	TotalRequests    atomic.Int64
	SuccessRequests  atomic.Int64
	FailedRequests   atomic.Int64
	CacheHits        atomic.Int64
	CacheMisses      atomic.Int64
	DBQueries        atomic.Int64
	DBErrors         atomic.Int64
	RedisOps         atomic.Int64
	RedisErrors      atomic.Int64
	AvgLatencyMs     atomic.Int64
	P50LatencyMs     atomic.Int64
	P95LatencyMs     atomic.Int64
	P99LatencyMs     atomic.Int64

	latencyWindow []int64
	windowMu      sync.Mutex
	windowSize    int
}

var globalMetricsCollector *MetricsCollector
var metricsOnce sync.Once

func NewMetricsCollector() *MetricsCollector {
	metricsOnce.Do(func() {
		globalMetricsCollector = &MetricsCollector{
			stats: &MetricsStats{
				latencyWindow: make([]int64, 0, 1000),
				windowSize:    1000,
			},
		}
		globalMetricsCollector.initMetrics()
	})
	return globalMetricsCollector
}

func GetMetricsCollector() *MetricsCollector {
	return NewMetricsCollector()
}

func (m *MetricsCollector) initMetrics() {
	m.requestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	m.latencyHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_ms",
			Help:    "HTTP request latency in milliseconds",
			Buckets: []float64{1, 5, 10, 20, 30, 40, 50, 60, 80, 100, 150, 200, 300, 500, 1000},
		},
		[]string{"method", "path"},
	)

	m.activeRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_requests",
			Help: "Number of active HTTP requests",
		},
	)

	m.cacheHitCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_operations_total",
			Help: "Total number of cache operations",
		},
		[]string{"operation", "result"},
	)

	m.dbQueryCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "status"},
	)

	m.dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_ms",
			Help:    "Database query duration in milliseconds",
			Buckets: []float64{1, 5, 10, 20, 30, 50, 100, 200, 500, 1000},
		},
		[]string{"operation"},
	)

	m.redisOpCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_operations_total",
			Help: "Total number of Redis operations",
		},
		[]string{"operation", "status"},
	)

	m.redisOpDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_operation_duration_ms",
			Help:    "Redis operation duration in milliseconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 20, 50, 100},
		},
		[]string{"operation"},
	)
}

func (m *MetricsCollector) RecordRequest(method, path, status string, latencyMs float64) {
	m.requestCounter.WithLabelValues(method, path, status).Inc()
	m.latencyHistogram.WithLabelValues(method, path).Observe(latencyMs)

	m.stats.TotalRequests.Add(1)
	if status >= "200" && status < "400" {
		m.stats.SuccessRequests.Add(1)
	} else {
		m.stats.FailedRequests.Add(1)
	}

	m.updateLatencyStats(int64(latencyMs))
}

func (m *MetricsCollector) RecordCacheHit(operation string) {
	m.cacheHitCounter.WithLabelValues(operation, "hit").Inc()
	m.stats.CacheHits.Add(1)
}

func (m *MetricsCollector) RecordCacheMiss(operation string) {
	m.cacheHitCounter.WithLabelValues(operation, "miss").Inc()
	m.stats.CacheMisses.Add(1)
}

func (m *MetricsCollector) RecordDBQuery(operation, status string, durationMs float64) {
	m.dbQueryCounter.WithLabelValues(operation, status).Inc()
	m.dbQueryDuration.WithLabelValues(operation).Observe(durationMs)

	m.stats.DBQueries.Add(1)
	if status != "success" {
		m.stats.DBErrors.Add(1)
	}
}

func (m *MetricsCollector) RecordRedisOp(operation, status string, durationMs float64) {
	m.redisOpCounter.WithLabelValues(operation, status).Inc()
	m.redisOpDuration.WithLabelValues(operation).Observe(durationMs)

	m.stats.RedisOps.Add(1)
	if status != "success" {
		m.stats.RedisErrors.Add(1)
	}
}

func (m *MetricsCollector) IncActiveRequests() {
	m.activeRequests.Inc()
}

func (m *MetricsCollector) DecActiveRequests() {
	m.activeRequests.Dec()
}

func (m *MetricsCollector) updateLatencyStats(latencyMs int64) {
	m.stats.windowMu.Lock()
	defer m.stats.windowMu.Unlock()

	m.stats.latencyWindow = append(m.stats.latencyWindow, latencyMs)
	if len(m.stats.latencyWindow) > m.stats.windowSize {
		m.stats.latencyWindow = m.stats.latencyWindow[1:]
	}

	var sum int64
	for _, v := range m.stats.latencyWindow {
		sum += v
	}
	m.stats.AvgLatencyMs.Store(sum / int64(len(m.stats.latencyWindow)))

	m.stats.P50LatencyMs.Store(m.calculatePercentile(50))
	m.stats.P95LatencyMs.Store(m.calculatePercentile(95))
	m.stats.P99LatencyMs.Store(m.calculatePercentile(99))
}

func (m *MetricsCollector) calculatePercentile(p int) int64 {
	m.stats.windowMu.Lock()
	defer m.stats.windowMu.Unlock()

	if len(m.stats.latencyWindow) == 0 {
		return 0
	}

	sorted := make([]int64, len(m.stats.latencyWindow))
	copy(sorted, m.stats.latencyWindow)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := int(float64(len(sorted)) * float64(p) / 100.0)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	if index < 0 {
		index = 0
	}

	return sorted[index]
}

func (m *MetricsCollector) GetStats() map[string]interface{} {
	stats := m.stats

	totalRequests := stats.TotalRequests.Load()
	successRequests := stats.SuccessRequests.Load()
	failedRequests := stats.FailedRequests.Load()

	successRate := float64(0)
	if totalRequests > 0 {
		successRate = float64(successRequests) / float64(totalRequests) * 100
	}

	cacheHits := stats.CacheHits.Load()
	cacheMisses := stats.CacheMisses.Load()
	cacheTotal := cacheHits + cacheMisses

	cacheHitRate := float64(0)
	if cacheTotal > 0 {
		cacheHitRate = float64(cacheHits) / float64(cacheTotal) * 100
	}

	return map[string]interface{}{
		"requests": map[string]interface{}{
			"total":         totalRequests,
			"success":       successRequests,
			"failed":        failedRequests,
			"success_rate":  successRate,
		},
		"latency": map[string]interface{}{
			"avg_ms":    stats.AvgLatencyMs.Load(),
			"p50_ms":    stats.P50LatencyMs.Load(),
			"p95_ms":    stats.P95LatencyMs.Load(),
			"p99_ms":    stats.P99LatencyMs.Load(),
			"target_ms": 60,
			"p99_met":   stats.P99LatencyMs.Load() <= 60,
		},
		"cache": map[string]interface{}{
			"hits":         cacheHits,
			"misses":       cacheMisses,
			"hit_rate":     cacheHitRate,
			"target_rate":  95.0,
			"hit_rate_met": cacheHitRate >= 95.0,
		},
		"database": map[string]interface{}{
			"queries":   stats.DBQueries.Load(),
			"errors":    stats.DBErrors.Load(),
		},
		"redis": map[string]interface{}{
			"operations": stats.RedisOps.Load(),
			"errors":     stats.RedisErrors.Load(),
		},
	}
}

func (m *MetricsCollector) ResetStats() {
	m.stats.TotalRequests.Store(0)
	m.stats.SuccessRequests.Store(0)
	m.stats.FailedRequests.Store(0)
	m.stats.CacheHits.Store(0)
	m.stats.CacheMisses.Store(0)
	m.stats.DBQueries.Store(0)
	m.stats.DBErrors.Store(0)
	m.stats.RedisOps.Store(0)
	m.stats.RedisErrors.Store(0)
	m.stats.AvgLatencyMs.Store(0)
	m.stats.P50LatencyMs.Store(0)
	m.stats.P95LatencyMs.Store(0)
	m.stats.P99LatencyMs.Store(0)

	m.stats.windowMu.Lock()
	m.stats.latencyWindow = make([]int64, 0, m.stats.windowSize)
	m.stats.windowMu.Unlock()
}

type QueryCache struct {
	mu       sync.RWMutex
	cache    map[string]*QueryCacheEntry
	maxSize  int
	ttl      time.Duration
	hits     atomic.Int64
	misses   atomic.Int64
}

type QueryCacheEntry struct {
	Key        string
	Data       interface{}
	CreatedAt  time.Time
	AccessedAt time.Time
	AccessCount int64
}

func NewQueryCache(maxSize int, ttl time.Duration) *QueryCache {
	return &QueryCache{
		cache:   make(map[string]*QueryCacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

func (qc *QueryCache) Get(ctx context.Context, key string) (interface{}, bool) {
	qc.mu.RLock()
	entry, exists := qc.cache[key]
	qc.mu.RUnlock()

	if !exists {
		qc.misses.Add(1)
		return nil, false
	}

	if time.Since(entry.CreatedAt) > qc.ttl {
		qc.mu.Lock()
		delete(qc.cache, key)
		qc.mu.Unlock()
		qc.misses.Add(1)
		return nil, false
	}

	qc.mu.Lock()
	entry.AccessedAt = time.Now()
	entry.AccessCount++
	qc.mu.Unlock()

	qc.hits.Add(1)
	return entry.Data, true
}

func (qc *QueryCache) Set(ctx context.Context, key string, data interface{}) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	if len(qc.cache) >= qc.maxSize {
		qc.evict()
	}

	qc.cache[key] = &QueryCacheEntry{
		Key:        key,
		Data:       data,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
	}
}

func (qc *QueryCache) evict() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range qc.cache {
		if first || entry.AccessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.AccessedAt
			first = false
		}
	}

	if oldestKey != "" {
		delete(qc.cache, oldestKey)
	}
}

func (qc *QueryCache) Delete(ctx context.Context, key string) {
	qc.mu.Lock()
	delete(qc.cache, key)
	qc.mu.Unlock()
}

func (qc *QueryCache) Clear(ctx context.Context) {
	qc.mu.Lock()
	qc.cache = make(map[string]*QueryCacheEntry)
	qc.mu.Unlock()
}

func (qc *QueryCache) GetStats() map[string]interface{} {
	hits := qc.hits.Load()
	misses := qc.misses.Load()
	total := hits + misses

	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	qc.mu.RLock()
	size := len(qc.cache)
	qc.mu.RUnlock()

	return map[string]interface{}{
		"hits":      hits,
		"misses":    misses,
		"total":     total,
		"hit_rate":  hitRate,
		"size":      size,
		"max_size":  qc.maxSize,
	}
}

type PerformanceOptimizer struct {
	metricsCollector *MetricsCollector
	queryCache      *QueryCache
	dbQueryTimeout  time.Duration
	enableQueryCache bool
}

func NewPerformanceOptimizer() *PerformanceOptimizer {
	return &PerformanceOptimizer{
		metricsCollector: NewMetricsCollector(),
		queryCache:      NewQueryCache(1000, 5*time.Minute),
		dbQueryTimeout:  30 * time.Second,
		enableQueryCache: true,
	}
}

func (po *PerformanceOptimizer) GetMetricsCollector() *MetricsCollector {
	return po.metricsCollector
}

func (po *PerformanceOptimizer) GetQueryCache() *QueryCache {
	return po.queryCache
}

func (po *PerformanceOptimizer) SetQueryCacheEnabled(enabled bool) {
	po.enableQueryCache = enabled
}

func (po *PerformanceOptimizer) OptimizeQuery(ctx context.Context, key string, queryFunc func() (interface{}, error)) (interface{}, error) {
	if po.enableQueryCache {
		if data, found := po.queryCache.Get(ctx, key); found {
			po.metricsCollector.RecordCacheHit("query")
			return data, nil
		}
	}

	result, err := queryFunc()
	if err != nil {
		return nil, err
	}

	if po.enableQueryCache {
		po.queryCache.Set(ctx, key, result)
	}

	return result, nil
}

func (po *PerformanceOptimizer) GetAllStats() map[string]interface{} {
	return map[string]interface{}{
		"metrics":   po.metricsCollector.GetStats(),
		"query_cache": po.queryCache.GetStats(),
	}
}

func (po *PerformanceOptimizer) IsP99Compliant() bool {
	stats := po.metricsCollector.GetStats()
	latency := stats["latency"].(map[string]interface{})
	return latency["p99_met"].(bool)
}

func (po *PerformanceOptimizer) IsCacheHitRateCompliant() bool {
	stats := po.metricsCollector.GetStats()
	cache := stats["cache"].(map[string]interface{})
	return cache["hit_rate_met"].(bool)
}

func (po *PerformanceOptimizer) GetComplianceStatus() map[string]interface{} {
	return map[string]interface{}{
		"api_latency_p99": map[string]interface{}{
			"target":  "60ms",
			"current": fmt.Sprintf("%dms", po.metricsCollector.stats.P99LatencyMs.Load()),
			"met":     po.IsP99Compliant(),
		},
		"cache_hit_rate": map[string]interface{}{
			"target":  "95%",
			"current": fmt.Sprintf("%.2f%%", po.getCacheHitRate()),
			"met":     po.IsCacheHitRateCompliant(),
		},
		"all_targets_met": po.IsP99Compliant() && po.IsCacheHitRateCompliant(),
	}
}

func (po *PerformanceOptimizer) getCacheHitRate() float64 {
	stats := po.metricsCollector.GetStats()
	cache := stats["cache"].(map[string]interface{})
	return cache["hit_rate"].(float64)
}
