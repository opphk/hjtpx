package metrics

import (
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type PerformanceMetrics struct {
	requestDuration    *prometheus.HistogramVec
	requestLatencyP50  prometheus.Gauge
	requestLatencyP95  prometheus.Gauge
	requestLatencyP99  prometheus.Gauge

	goroutineCount  prometheus.Gauge
	gcCountTotal   prometheus.Counter
	gcPauseTotal   prometheus.Counter
	gcPauseSeconds prometheus.Histogram

	memoryUsageHeap     prometheus.Gauge
	memoryUsageStack   prometheus.Gauge
	memoryUsageTotal   prometheus.Gauge
	memoryAllocationRate prometheus.Gauge

	cpuUsage prometheus.Gauge

	databaseConnectionsActive  prometheus.Gauge
	databaseConnectionsIdle   prometheus.Gauge
	databaseConnectionsInUse  prometheus.Gauge
	databaseQueryDuration     *prometheus.HistogramVec
	databaseQueryErrors       *prometheus.CounterVec

	redisConnectionsActive prometheus.Gauge
	redisCommandsTotal     *prometheus.CounterVec
	redisCommandDuration   *prometheus.HistogramVec
	redisErrors            prometheus.Counter

	cacheHitTotal  prometheus.Counter
	cacheMissTotal prometheus.Counter
	cacheHitRate   prometheus.Gauge

	bandwidthIn  prometheus.Counter
	bandwidthOut prometheus.Counter

	threadCount prometheus.Gauge

	mu sync.RWMutex
}

func newPerformanceMetrics(registry *prometheus.Registry) *PerformanceMetrics {
	pm := &PerformanceMetrics{
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path", "status"},
		),
		requestLatencyP50: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_request_latency_p50_ms",
				Help: "HTTP request latency p50 in milliseconds",
			},
		),
		requestLatencyP95: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_request_latency_p95_ms",
				Help: "HTTP request latency p95 in milliseconds",
			},
		),
		requestLatencyP99: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_request_latency_p99_ms",
				Help: "HTTP request latency p99 in milliseconds",
			},
		),
		goroutineCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "go_goroutines",
				Help: "Number of goroutines",
			},
		),
		gcCountTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "go_gc_count_total",
				Help: "Total number of garbage collections",
			},
		),
		gcPauseTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "go_gc_pause_total_seconds",
				Help: "Total GC pause time in seconds",
			},
		),
		gcPauseSeconds: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "go_gc_pause_seconds",
				Help:    "GC pause duration in seconds",
				Buckets: []float64{.0001, .0005, .001, .005, .01, .05, .1, .5, 1},
			},
		),
		memoryUsageHeap: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "go_memory_heap_bytes",
				Help: "Go memory heap usage in bytes",
			},
		),
		memoryUsageStack: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "go_memory_stack_bytes",
				Help: "Go memory stack usage in bytes",
			},
		),
		memoryUsageTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "go_memory_total_bytes",
				Help: "Total Go memory usage in bytes",
			},
		),
		memoryAllocationRate: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "go_memory_allocation_rate_bytes_per_second",
				Help: "Memory allocation rate in bytes per second",
			},
		),
		cpuUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "go_cpu_usage_percent",
				Help: "CPU usage percentage",
			},
		),
		databaseConnectionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "database_connections_active",
				Help: "Number of active database connections",
			},
		),
		databaseConnectionsIdle: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "database_connections_idle",
				Help: "Number of idle database connections",
			},
		),
		databaseConnectionsInUse: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "database_connections_in_use",
				Help: "Number of database connections in use",
			},
		),
		databaseQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "database_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
			},
			[]string{"operation", "table"},
		),
		databaseQueryErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "database_query_errors_total",
				Help: "Total database query errors",
			},
			[]string{"operation", "table"},
		),
		redisConnectionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "redis_connections_active",
				Help: "Number of active Redis connections",
			},
		),
		redisCommandsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "redis_commands_total",
				Help: "Total Redis commands",
			},
			[]string{"command"},
		),
		redisCommandDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "redis_command_duration_seconds",
				Help:    "Redis command duration in seconds",
				Buckets: []float64{.0001, .0005, .001, .005, .01, .025, .05, .1},
			},
			[]string{"command"},
		),
		redisErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "redis_errors_total",
				Help: "Total Redis errors",
			},
		),
		cacheHitTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total cache hits",
			},
		),
		cacheMissTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total cache misses",
			},
		),
		cacheHitRate: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "cache_hit_rate",
				Help: "Cache hit rate",
			},
		),
		bandwidthIn: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "network_bandwidth_in_bytes_total",
				Help: "Total network bandwidth in bytes",
			},
		),
		bandwidthOut: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "network_bandwidth_out_bytes_total",
				Help: "Total network bandwidth out bytes",
			},
		),
		threadCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "go_threads",
				Help: "Number of OS threads",
			},
		),
	}

	registry.MustRegister(pm.requestDuration)
	registry.MustRegister(pm.requestLatencyP50)
	registry.MustRegister(pm.requestLatencyP95)
	registry.MustRegister(pm.requestLatencyP99)
	registry.MustRegister(pm.goroutineCount)
	registry.MustRegister(pm.gcCountTotal)
	registry.MustRegister(pm.gcPauseTotal)
	registry.MustRegister(pm.gcPauseSeconds)
	registry.MustRegister(pm.memoryUsageHeap)
	registry.MustRegister(pm.memoryUsageStack)
	registry.MustRegister(pm.memoryUsageTotal)
	registry.MustRegister(pm.memoryAllocationRate)
	registry.MustRegister(pm.cpuUsage)
	registry.MustRegister(pm.databaseConnectionsActive)
	registry.MustRegister(pm.databaseConnectionsIdle)
	registry.MustRegister(pm.databaseConnectionsInUse)
	registry.MustRegister(pm.databaseQueryDuration)
	registry.MustRegister(pm.databaseQueryErrors)
	registry.MustRegister(pm.redisConnectionsActive)
	registry.MustRegister(pm.redisCommandsTotal)
	registry.MustRegister(pm.redisCommandDuration)
	registry.MustRegister(pm.redisErrors)
	registry.MustRegister(pm.cacheHitTotal)
	registry.MustRegister(pm.cacheMissTotal)
	registry.MustRegister(pm.cacheHitRate)
	registry.MustRegister(pm.bandwidthIn)
	registry.MustRegister(pm.bandwidthOut)
	registry.MustRegister(pm.threadCount)

	go pm.collectRuntimeMetrics()

	return pm
}

func (pm *PerformanceMetrics) collectRuntimeMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var lastNumGC uint32
	var lastMemStats runtime.MemStats

	for {
		select {
		case <-ticker.C:
			pm.updateRuntimeMetrics(&lastNumGC, &lastMemStats)
		}
	}
}

func (pm *PerformanceMetrics) updateRuntimeMetrics(lastNumGC *uint32, lastMemStats *runtime.MemStats) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	pm.goroutineCount.Set(float64(runtime.NumGoroutine()))
	pm.threadCount.Set(float64(runtime.NumCPU()))

	pm.memoryUsageHeap.Set(float64(memStats.HeapAlloc))
	pm.memoryUsageStack.Set(float64(memStats.StackInuse))
	pm.memoryUsageTotal.Set(float64(memStats.Alloc))

	allocationRate := float64(memStats.Alloc) - float64(lastMemStats.Alloc)
	pm.memoryAllocationRate.Set(allocationRate / 10)

	if memStats.NumGC > *lastNumGC {
		gcCount := memStats.NumGC - *lastNumGC
		pm.gcCountTotal.Add(float64(gcCount))

		if len(memStats.PauseNs) > 0 {
			pauseNs := memStats.PauseNs[(memStats.NumGC+255)%256]
			pauseSeconds := float64(pauseNs) / 1e9
			pm.gcPauseTotal.Add(pauseSeconds)
			pm.gcPauseSeconds.Observe(pauseSeconds)
		}
	}

	*lastNumGC = memStats.NumGC
	*lastMemStats = memStats
}

func (pm *PerformanceMetrics) RecordHTTPRequest(method, path string, status int, duration time.Duration) {
	pm.requestDuration.WithLabelValues(method, path, statusCodeToString(status)).Observe(duration.Seconds())
}

func (pm *PerformanceMetrics) UpdateLatencyPercentiles(p50, p95, p99 time.Duration) {
	pm.requestLatencyP50.Set(float64(p50.Milliseconds()))
	pm.requestLatencyP95.Set(float64(p95.Milliseconds()))
	pm.requestLatencyP99.Set(float64(p99.Milliseconds()))
}

func (pm *PerformanceMetrics) UpdateDatabaseConnections(active, idle, inUse int) {
	pm.databaseConnectionsActive.Set(float64(active))
	pm.databaseConnectionsIdle.Set(float64(idle))
	pm.databaseConnectionsInUse.Set(float64(inUse))
}

func (pm *PerformanceMetrics) RecordDatabaseQuery(operation, table string, duration time.Duration, err error) {
	pm.databaseQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
	if err != nil {
		pm.databaseQueryErrors.WithLabelValues(operation, table).Inc()
	}
}

func (pm *PerformanceMetrics) UpdateRedisConnections(active int) {
	pm.redisConnectionsActive.Set(float64(active))
}

func (pm *PerformanceMetrics) RecordRedisCommand(command string, duration time.Duration, err error) {
	pm.redisCommandsTotal.WithLabelValues(command).Inc()
	pm.redisCommandDuration.WithLabelValues(command).Observe(duration.Seconds())
	if err != nil {
		pm.redisErrors.Inc()
	}
}

func (pm *PerformanceMetrics) RecordCacheHit() {
	pm.cacheHitTotal.Inc()
	pm.updateCacheHitRate()
}

func (pm *PerformanceMetrics) RecordCacheMiss() {
	pm.cacheMissTotal.Inc()
	pm.updateCacheHitRate()
}

func (pm *PerformanceMetrics) updateCacheHitRate() {
	var hits, misses float64
	metrics, _ := prometheus.DefaultGatherer.Gather()
	for _, mf := range metrics {
		if mf.GetName() == "cache_hits_total" && len(mf.GetMetric()) > 0 {
			hits = float64(mf.GetMetric()[0].GetCounter().GetValue())
		}
		if mf.GetName() == "cache_misses_total" && len(mf.GetMetric()) > 0 {
			misses = float64(mf.GetMetric()[0].GetCounter().GetValue())
		}
	}
	total := hits + misses
	if total > 0 {
		pm.cacheHitRate.Set(hits / total)
	}
}

func (pm *PerformanceMetrics) RecordBandwidthIn(bytes uint64) {
	pm.bandwidthIn.Add(float64(bytes))
}

func (pm *PerformanceMetrics) RecordBandwidthOut(bytes uint64) {
	pm.bandwidthOut.Add(float64(bytes))
}
