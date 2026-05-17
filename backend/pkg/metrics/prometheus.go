package metrics

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	collectorInstance *PrometheusCollector
	collectorOnce    sync.Once
)

type PrometheusCollector struct {
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestsInFlight prometheus.Gauge

	businessMetrics    *BusinessMetrics
	performanceMetrics *PerformanceMetrics
	securityMetrics     *SecurityMetrics

	server           *http.Server
	registry         *prometheus.Registry
	mu               sync.RWMutex
}

func GetPrometheusCollector() *PrometheusCollector {
	collectorOnce.Do(func() {
		collectorInstance = newPrometheusCollector()
	})
	return collectorInstance
}

func newPrometheusCollector() *PrometheusCollector {
	registry := prometheus.NewRegistry()

	collector := &PrometheusCollector{
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path"},
		),
		httpRequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),
		registry: registry,
	}

	collector.businessMetrics = newBusinessMetrics(registry)
	collector.performanceMetrics = newPerformanceMetrics(registry)
	collector.securityMetrics = newSecurityMetrics(registry)

	return collector
}

func (pc *PrometheusCollector) Start(addr string) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.server != nil {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(pc.registry, promhttp.HandlerOpts{}))

	pc.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		_ = pc.server.ListenAndServe()
	}()

	return nil
}

func (pc *PrometheusCollector) Stop() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.server == nil {
		return nil
	}

	pc.server.Close()
	pc.server = nil
	return nil
}

func (pc *PrometheusCollector) RecordHTTPRequest(method, path string, status int, duration time.Duration) {
	pc.httpRequestsTotal.WithLabelValues(method, path, statusCodeToString(status)).Inc()
	pc.httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

func (pc *PrometheusCollector) IncrementInFlight() {
	pc.httpRequestsInFlight.Inc()
}

func (pc *PrometheusCollector) DecrementInFlight() {
	pc.httpRequestsInFlight.Dec()
}

func (pc *PrometheusCollector) GetBusinessMetrics() *BusinessMetrics {
	return pc.businessMetrics
}

func (pc *PrometheusCollector) GetPerformanceMetrics() *PerformanceMetrics {
	return pc.performanceMetrics
}

func (pc *PrometheusCollector) GetSecurityMetrics() *SecurityMetrics {
	return pc.securityMetrics
}

func statusCodeToString(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "2xx"
	case status >= 300 && status < 400:
		return "3xx"
	case status >= 400 && status < 500:
		return "4xx"
	case status >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}
