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

	httpRequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_http_requests_in_flight",
			Help: "Current number of in-flight requests",
		},
	)

	// 业务错误指标
	businessErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_business_errors_total",
			Help: "Total number of business errors",
		},
		[]string{"error_type", "endpoint"},
	)

	// 认证安全指标
	authSuccessTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_auth_success_total",
			Help: "Total number of successful authentications",
		},
	)

	authFailureTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_auth_failure_total",
			Help: "Total number of failed authentications",
		},
		[]string{"reason"},
	)

	captchaSuccessTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_captcha_success_total",
			Help: "Total number of successful captcha verifications",
		},
	)

	captchaFailureTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_captcha_failure_total",
			Help: "Total number of failed captcha verifications",
		},
	)

	captchaBlockedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_captcha_blocked_total",
			Help: "Total number of blocked requests due to captcha",
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

	dbQueryCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_db_query_count_total",
			Help: "Total number of database queries",
		},
		[]string{"query_type"},
	)

	dbErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_db_errors_total",
			Help: "Total number of database errors",
		},
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

	cacheOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_cache_operations_total",
			Help: "Total number of cache operations",
		},
		[]string{"operation"},
	)

	// 限流指标
	rateLimitAccepted = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_rate_limit_accepted_total",
			Help: "Total number of requests accepted by rate limiter",
		},
	)

	rateLimitRejected = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_rate_limit_rejected_total",
			Help: "Total number of requests rejected by rate limiter",
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

	// 健康检查指标
	healthChecksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_health_checks_total",
			Help: "Total number of health checks",
		},
		[]string{"result"},
	)

	// 服务质量指标
	availability = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_availability",
			Help: "Service availability percentage",
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
		httpRequestsInFlight,
		businessErrorsTotal,
		authSuccessTotal,
		authFailureTotal,
		captchaSuccessTotal,
		captchaFailureTotal,
		captchaBlockedTotal,
		dbConnectionsTotal,
		dbConnectionsActive,
		dbConnectionsIdle,
		dbWaitCount,
		dbWaitDuration,
		dbQueryDuration,
		dbQueryCount,
		dbErrorsTotal,
		cacheHits,
		cacheMisses,
		cacheSize,
		cacheOperations,
		rateLimitAccepted,
		rateLimitRejected,
		uptime,
		version,
		healthChecksTotal,
		availability,
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
		httpRequestsInFlight.Inc()

		defer func() {
			httpActiveRequests.Dec()
			httpRequestsInFlight.Dec()

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
	cacheOperations.WithLabelValues("get").Inc()
}

func RecordCacheMiss() {
	cacheMisses.Inc()
	cacheOperations.WithLabelValues("get").Inc()
}

func RecordCacheSize(size int64) {
	cacheSize.Set(float64(size))
}

func RecordCacheOperation(operation string) {
	cacheOperations.WithLabelValues(operation).Inc()
}

func RecordDBQueryDuration(queryType string, duration time.Duration) {
	dbQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
	dbQueryCount.WithLabelValues(queryType).Inc()
}

func RecordDBError() {
	dbErrorsTotal.Inc()
}

func RecordBusinessError(errorType, endpoint string) {
	businessErrorsTotal.WithLabelValues(errorType, endpoint).Inc()
}

func RecordAuthSuccess() {
	authSuccessTotal.Inc()
}

func RecordAuthFailure(reason string) {
	authFailureTotal.WithLabelValues(reason).Inc()
}

func RecordCaptchaSuccess() {
	captchaSuccessTotal.Inc()
}

func RecordCaptchaFailure() {
	captchaFailureTotal.Inc()
}

func RecordCaptchaBlocked() {
	captchaBlockedTotal.Inc()
}

func RecordRateLimitAccepted() {
	rateLimitAccepted.Inc()
}

func RecordRateLimitRejected() {
	rateLimitRejected.Inc()
}

func RecordHealthCheck(result string) {
	healthChecksTotal.WithLabelValues(result).Inc()
}

func SetAvailability(avail float64) {
	availability.Set(avail)
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