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

	// 验证码指标
	captchaTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_captcha_total",
			Help: "Total number of captcha requests",
		},
		[]string{"type", "status"},
	)

	captchaVerifyDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_captcha_verify_duration_seconds",
			Help:    "Captcha verification duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type"},
	)

	captchaSuccessRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hjtpx_captcha_success_rate",
			Help: "Captcha success rate",
		},
		[]string{"type"},
	)

	captchaBlocked = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_captcha_blocked_total",
			Help: "Total number of blocked captcha attempts",
		},
		[]string{"reason"},
	)

	// WebSocket指标
	websocketConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_websocket_connections_active",
			Help: "Number of active WebSocket connections",
		},
	)

	websocketMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_websocket_messages_total",
			Help: "Total number of WebSocket messages",
		},
		[]string{"direction", "type"},
	)

	websocketMessageDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_websocket_message_duration_seconds",
			Help:    "WebSocket message processing duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type"},
	)

	// 安全指标
	securityEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_security_events_total",
			Help: "Total number of security events",
		},
		[]string{"type", "severity"},
	)

	rateLimitHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"type"},
	)

	botDetectionTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_bot_detection_total",
			Help: "Total number of bot detections",
		},
		[]string{"type", "action"},
	)

	// 业务指标
	activeUsers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_active_users",
			Help: "Number of active users",
		},
	)

	authAttempts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_auth_attempts_total",
			Help: "Total authentication attempts",
		},
		[]string{"status", "method"},
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
		captchaTotal,
		captchaVerifyDuration,
		captchaSuccessRate,
		captchaBlocked,
		websocketConnections,
		websocketMessagesTotal,
		websocketMessageDuration,
		securityEventsTotal,
		rateLimitHits,
		botDetectionTotal,
		activeUsers,
		authAttempts,
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

func RecordCaptchaRequest(captchaType, status string) {
	captchaTotal.WithLabelValues(captchaType, status).Inc()
}

func RecordCaptchaVerifyDuration(captchaType string, duration time.Duration) {
	captchaVerifyDuration.WithLabelValues(captchaType).Observe(duration.Seconds())
}

func RecordCaptchaSuccessRate(captchaType string, rate float64) {
	captchaSuccessRate.WithLabelValues(captchaType).Set(rate)
}

func RecordCaptchaBlocked(reason string) {
	captchaBlocked.WithLabelValues(reason).Inc()
}

func RecordWebSocketConnection() {
	websocketConnections.Inc()
}

func RecordWebSocketDisconnection() {
	websocketConnections.Dec()
}

func RecordWebSocketMessage(direction, msgType string) {
	websocketMessagesTotal.WithLabelValues(direction, msgType).Inc()
}

func RecordWebSocketMessageDuration(msgType string, duration time.Duration) {
	websocketMessageDuration.WithLabelValues(msgType).Observe(duration.Seconds())
}

func RecordSecurityEvent(eventType, severity string) {
	securityEventsTotal.WithLabelValues(eventType, severity).Inc()
}

func RecordRateLimitHit(rateLimitType string) {
	rateLimitHits.WithLabelValues(rateLimitType).Inc()
}

func RecordBotDetection(detectionType, action string) {
	botDetectionTotal.WithLabelValues(detectionType, action).Inc()
}

func SetActiveUsers(count float64) {
	activeUsers.Set(count)
}

func RecordAuthAttempt(status, method string) {
	authAttempts.WithLabelValues(status, method).Inc()
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
