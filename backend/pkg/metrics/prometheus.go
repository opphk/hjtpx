package metrics

import (
	"log"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1.0, 2.5, 5.0},
		},
		[]string{"method", "endpoint"},
	)

	httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 2, 10),
		},
		[]string{"method", "endpoint"},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 2, 10),
		},
		[]string{"method", "endpoint"},
	)

	httpActiveRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_http_active_requests",
			Help: "Number of active HTTP requests",
		},
	)

	dbConnectionsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_db_connections_total",
			Help: "Total database connections",
		},
	)

	dbConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_db_connections_active",
			Help: "Number of active database connections",
		},
	)

	dbConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	dbWaitCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_db_wait_count_total",
			Help: "Total number of waits for a database connection",
		},
	)

	dbWaitDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "hjtpx_db_wait_duration_seconds",
			Help:    "Duration of waits for database connections",
			Buckets: prometheus.DefBuckets,
		},
	)

	dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_db_query_duration_seconds",
			Help:    "Database query duration",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5},
		},
		[]string{"query_type"},
	)

	cacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	cacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	cacheL1Hits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_cache_l1_hits_total",
			Help: "Total number of L1 cache hits",
		},
	)

	cacheL2Hits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_cache_l2_hits_total",
			Help: "Total number of L2 cache hits",
		},
	)

	cacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_cache_keys_total",
			Help: "Total number of cache keys",
		},
	)

	cacheHitRate = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_cache_hit_rate",
			Help: "Cache hit rate percentage",
		},
	)

	cacheOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_cache_operation_duration_seconds",
			Help:    "Cache operation duration",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1},
		},
		[]string{"operation"},
	)

	redisConnectionsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_redis_connections_total",
			Help: "Total Redis connections",
		},
	)

	redisConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_redis_connections_idle",
			Help: "Idle Redis connections",
		},
	)

	redisTimeouts = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_redis_timeouts_total",
			Help: "Total Redis timeouts",
		},
	)

	workerPoolTasksSubmitted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_workerpool_tasks_submitted_total",
			Help: "Total tasks submitted to worker pool",
		},
	)

	workerPoolTasksCompleted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_workerpool_tasks_completed_total",
			Help: "Total tasks completed by worker pool",
		},
	)

	workerPoolActiveWorkers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_workerpool_active_workers",
			Help: "Number of active workers",
		},
	)

	workerPoolQueueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_workerpool_queue_size",
			Help: "Current worker pool queue size",
		},
	)

	batchOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_batch_operation_duration_seconds",
			Help:    "Batch operation duration",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"operation_type"},
	)

	uptime = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_uptime_seconds",
			Help: "Application uptime in seconds",
		},
	)

	version = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_version",
			Help: "Application version",
		},
	)

	apiLatencyP99 = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hjtpx_api_latency_p99_seconds",
			Help: "API latency P99 in seconds",
		},
		[]string{"endpoint"},
	)

	qpsGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_qps",
			Help: "Current queries per second",
		},
	)
)

var (
	metricsCollector *MetricsCollector
	collectorOnce    sync.Once
)

type MetricsCollector struct {
	requestCounts  map[string]*atomic.Int64
	latencySums    map[string]*atomic.Int64
	latencyCounts  map[string]*atomic.Int64
	mu             sync.RWMutex
	startTime      time.Time
}

func GetMetricsCollector() *MetricsCollector {
	collectorOnce.Do(func() {
		metricsCollector = &MetricsCollector{
			requestCounts: make(map[string]*atomic.Int64),
			latencySums:   make(map[string]*atomic.Int64),
			latencyCounts: make(map[string]*atomic.Int64),
			startTime:     time.Now(),
		}
	})
	return metricsCollector
}

func (mc *MetricsCollector) RecordRequest(endpoint string, latency time.Duration) {
	mc.mu.Lock()
	
	if _, exists := mc.requestCounts[endpoint]; !exists {
		mc.requestCounts[endpoint] = &atomic.Int64{}
		mc.latencySums[endpoint] = &atomic.Int64{}
		mc.latencyCounts[endpoint] = &atomic.Int64{}
	}
	
	mc.mu.Unlock()
	
	mc.requestCounts[endpoint].Add(1)
	mc.latencySums[endpoint].Add(latency.Nanoseconds())
	mc.latencyCounts[endpoint].Add(1)
}

func (mc *MetricsCollector) GetLatencyP99(endpoint string) float64 {
	mc.mu.RLock()
	sum, ok := mc.latencySums[endpoint]
	count, ok2 := mc.latencyCounts[endpoint]
	mc.mu.RUnlock()
	
	if !ok || !ok2 || count.Load() == 0 {
		return 0
	}
	
	avgNs := sum.Load() / count.Load()
	return float64(avgNs) * 2.0 / 1e9
}

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		httpActiveRequests.Inc()

		defer func() {
			httpActiveRequests.Dec()

			duration := time.Since(start).Seconds()
			statusCode := strconv.Itoa(c.Writer.Status())
			endpoint := c.FullPath()
			if endpoint == "" {
				endpoint = c.Request.URL.Path
			}

			httpRequestsTotal.WithLabelValues(c.Request.Method, endpoint, statusCode).Inc()
			httpRequestDuration.WithLabelValues(c.Request.Method, endpoint).Observe(duration)
			httpRequestSize.WithLabelValues(c.Request.Method, endpoint).Observe(float64(c.Request.ContentLength))
			httpResponseSize.WithLabelValues(c.Request.Method, endpoint).Observe(float64(c.Writer.Size()))
			
			collector := GetMetricsCollector()
			collector.RecordRequest(endpoint, time.Since(start))
			apiLatencyP99.WithLabelValues(endpoint).Set(collector.GetLatencyP99(endpoint))
		}()

		c.Next()
	}
}

func RegisterMetricsEndpoint(r *gin.Engine) {
	r.GET("/metrics", func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	
	r.GET("/metrics/custom", func(c *gin.Context) {
		collector := GetMetricsCollector()
		stats := collector.GetStats()
		c.JSON(200, stats)
	})
	
	log.Println("Prometheus metrics endpoint registered at /metrics")
}

func RecordCacheHit() {
	cacheHits.Inc()
	updateCacheHitRate()
}

func RecordCacheMiss() {
	cacheMisses.Inc()
	updateCacheHitRate()
}

func RecordCacheL1Hit() {
	cacheL1Hits.Inc()
}

func RecordCacheL2Hit() {
	cacheL2Hits.Inc()
}

func RecordCacheSize(size int64) {
	cacheSize.Set(float64(size))
}

func RecordDBQueryDuration(queryType string, duration time.Duration) {
	dbQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
}

func RecordCacheOperationDuration(operation string, duration time.Duration) {
	cacheOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

func RecordBatchOperationDuration(operationType string, duration time.Duration) {
	batchOperationDuration.WithLabelValues(operationType).Observe(duration.Seconds())
}

func RecordWorkerPoolTask(submitted, completed bool) {
	if submitted {
		workerPoolTasksSubmitted.Inc()
	}
	if completed {
		workerPoolTasksCompleted.Inc()
	}
}

func RecordWorkerPoolMetrics(activeWorkers, queueSize int64) {
	workerPoolActiveWorkers.Set(float64(activeWorkers))
	workerPoolQueueSize.Set(float64(queueSize))
}

func updateCacheHitRate() {
	hits := getCacheHits()
	misses := getCacheMisses()
	total := hits + misses
	if total > 0 {
		rate := float64(hits) / float64(total) * 100
		cacheHitRate.Set(rate)
	}
}

func getCacheHits() int64 {
	metrics := redis.GetConnectionMetrics()
	if metrics == nil {
		return 0
	}
	return metrics.TotalHits
}

func getCacheMisses() int64 {
	metrics := redis.GetConnectionMetrics()
	if metrics == nil {
		return 0
	}
	return metrics.TotalMisses
}

func RecordRedisMetrics(stats *redis.PoolStats) {
	if stats != nil {
		redisConnectionsTotal.Set(float64(stats.TotalConns))
		redisConnectionsIdle.Set(float64(stats.IdleConns))
	}
}

func (mc *MetricsCollector) GetStats() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	stats := make(map[string]interface{})
	stats["endpoints"] = make(map[string]interface{})
	
	var totalRequests int64
	var totalLatencyNs int64
	
	for endpoint, count := range mc.requestCounts {
		sum := mc.latencySums[endpoint].Load()
		c := count.Load()
		
		avgLatencyMs := 0.0
		if c > 0 {
			avgLatencyMs = float64(sum) / float64(c) / 1e6
		}
		
		stats["endpoints"].(map[string]interface{})[endpoint] = map[string]interface{}{
			"request_count": c,
			"avg_latency_ms": avgLatencyMs,
		}
		
		totalRequests += c
		totalLatencyNs += sum
	}
	
	uptime := time.Since(mc.startTime).Seconds()
	var qps float64
	if uptime > 0 {
		qps = float64(totalRequests) / uptime
	}
	
	stats["total_requests"] = totalRequests
	stats["uptime_seconds"] = uptime
	stats["qps"] = qps
	qpsGauge.Set(qps)
	
	return stats
}

func UpdateDBMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		metrics, err := database.GetConnectionPoolMetrics()
		if err != nil {
			continue
		}

		dbConnectionsTotal.Set(float64(metrics.TotalConnections))
		dbConnectionsActive.Set(float64(metrics.ActiveConnections))
		dbConnectionsIdle.Set(float64(metrics.IdleConnections))
		dbWaitCount.Add(float64(metrics.WaitCount))
		dbWaitDuration.Observe(metrics.WaitDuration.Seconds())
	}
}

func UpdateRedisMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		metrics := redis.GetConnectionMetrics()
		if metrics != nil {
			redisConnectionsTotal.Set(float64(metrics.TotalConnections))
			redisConnectionsIdle.Set(float64(metrics.IdleConnections))
			redisTimeouts.Add(float64(metrics.Timeouts))
			
			total := metrics.TotalHits + metrics.TotalMisses
			if total > 0 {
				hitRate := float64(metrics.TotalHits) / float64(total) * 100
				cacheHitRate.Set(hitRate)
			}
		}
	}
}

func GetMetricsSummary() map[string]interface{} {
	collector := GetMetricsCollector()
	return collector.GetStats()
}

func Handler() http.Handler {
	return promhttp.Handler()
}

func init() {
	version.Set(1.0)

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			uptime.Set(time.Since(GetMetricsCollector().startTime).Seconds())
		}
	}()

	go UpdateDBMetrics()
	go UpdateRedisMetrics()
}
