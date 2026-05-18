package metrics

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP请求指标
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	httpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 2, 10),
		},
		[]string{"method", "endpoint"},
	)

	httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 2, 10),
		},
		[]string{"method", "endpoint"},
	)

	httpActiveRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_http_active_requests",
			Help: "Number of active HTTP requests",
		},
	)

	// 数据库指标
	dbConnectionsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_db_connections_total",
			Help: "Total database connections",
		},
	)

	dbConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_db_connections_active",
			Help: "Number of active database connections",
		},
	)

	dbConnectionsIdle = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	dbWaitCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_db_wait_count_total",
			Help: "Total number of waits for a database connection",
		},
	)

	dbWaitDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "hjtpx_db_wait_duration_seconds",
			Help:    "Duration of waits for database connections",
			Buckets: prometheus.DefBuckets,
		},
	)

	dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_db_query_duration_seconds",
			Help:    "Database query duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"query_type"},
	)

	// 缓存指标
	cacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	cacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	cacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_cache_keys_total",
			Help: "Total number of cache keys",
		},
	)

	// 应用指标
	uptime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_uptime_seconds",
			Help: "Application uptime in seconds",
		},
	)

	version = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_version",
			Help: "Application version",
		},
	)
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		httpRequestSize,
		httpResponseSize,
		httpActiveRequests,
		dbConnectionsTotal,
		dbConnectionsActive,
		dbConnectionsIdle,
		dbWaitCount,
		dbWaitDuration,
		dbQueryDuration,
		cacheHits,
		cacheMisses,
		cacheSize,
		uptime,
		version,
	)

	version.Set(1.0)

	go updateUptime()
	go updateDBMetrics()
}

func updateUptime() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		uptime.Set(time.Since(startTime).Seconds())
	}
}

func updateDBMetrics() {
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
		}()

		c.Next()
	}
}

func RegisterMetricsEndpoint(r *gin.Engine) {
	r.GET("/metrics", func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	log.Println("Prometheus metrics endpoint registered at /metrics")
}

func RecordCacheHit() {
	cacheHits.Inc()
}

func RecordCacheMiss() {
	cacheMisses.Inc()
}

func RecordCacheSize(size int64) {
	cacheSize.Set(float64(size))
}

func RecordDBQueryDuration(queryType string, duration time.Duration) {
	dbQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
}

func GetMetricsSummary() map[string]interface{} {
	return map[string]interface{}{
		"http_requests_total": GetRequestCount(),
		"http_success_rate":   GetSuccessRate(),
		"uptime_seconds":      GetUptime().Seconds(),
	}
}

func Handler() http.Handler {
	return promhttp.Handler()
}
