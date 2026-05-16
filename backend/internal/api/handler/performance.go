package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type PerformanceMonitor struct {
	requestCount    uint64
	errorCount      uint64
	totalDuration   time.Duration
	maxDuration     time.Duration
	minDuration     time.Duration
	mu              sync.RWMutex
	durationCount   int
}

var perfMonitor = &PerformanceMonitor{
	maxDuration: 0,
	minDuration: 0,
}

type EndpointMetrics struct {
	Path        string `json:"path"`
	Method      string `json:"method"`
	Count       uint64 `json:"count"`
	AvgDuration string `json:"avg_duration"`
	MaxDuration string `json:"max_duration"`
	MinDuration string `json:"min_duration"`
	ErrorRate   string `json:"error_rate"`
}

type SystemMetrics struct {
	GoVersion     string  `json:"go_version"`
	NumCPU        int     `json:"num_cpu"`
	NumGoroutine  int     `json:"num_goroutine"`
	MemAlloc      uint64  `json:"mem_alloc"`
	MemTotalAlloc uint64  `json:"mem_total_alloc"`
	MemSys       uint64  `json:"mem_sys"`
	MemLookups    uint64  `json:"mem_lookups"`
	MemMallocs    uint64  `json:"mem_mallocs"`
	MemFrees     uint64  `json:"mem_frees"`
	HeapAlloc    uint64  `json:"heap_alloc"`
	HeapSys      uint64  `json:"heap_sys"`
	HeapIdle     uint64  `json:"heap_idle"`
	HeapInuse    uint64  `json:"heap_inuse"`
	StackInuse   uint64  `json:"stack_inuse"`
	StackSys     uint64  `json:"stack_sys"`
	MSpanInuse   uint64  `json:"mspan_inuse"`
	MSpanSys     uint64  `json:"mspan_sys"`
	MCacheInuse  uint64  `json:"mcache_inuse"`
	MCacheSys    uint64  `json:"mcache_sys"`
	BuckHashSys  uint64  `json:"buck_hash_sys"`
	GCSys        uint64  `json:"gc_sys"`
	OtherSys     uint64  `json:"other_sys"`
	NextGC       uint64  `json:"next_gc"`
	LastGC       uint64  `json:"last_gc"`
	NumForcedGC  uint32  `json:"num_forced_gc"`
	GCCPUFraction float64 `json:"gc_cpu_fraction"`
	PauseTotalNs uint64  `json:"pause_total_ns"`
	NumGC        uint32  `json:"num_gc"`
}

type DatabasePoolMetrics struct {
	MaxOpenConnections int   `json:"max_open_connections"`
	OpenConnections   int   `json:"open_connections"`
	InUse             int   `json:"in_use"`
	Idle              int   `json:"idle"`
	WaitCount         int64 `json:"wait_count"`
	WaitDuration      string `json:"wait_duration"`
	MaxIdleClosed     int64 `json:"max_idle_closed"`
	MaxLifetimeClosed int64 `json:"max_lifetime_closed"`
}

type RedisPoolMetrics struct {
	TotalConns    uint32 `json:"total_conns"`
	IdleConns     uint32 `json:"idle_conns"`
	StaleConns    uint32 `json:"stale_conns"`
	Hits          uint32 `json:"hits"`
	Misses        uint32 `json:"misses"`
	Timeouts      uint32 `json:"timeouts"`
}

type PerformanceMetricsResponse struct {
	Timestamp       time.Time              `json:"timestamp"`
	RequestMetrics  *RequestMetricsSummary `json:"request_metrics"`
	SystemMetrics   *SystemMetrics         `json:"system_metrics"`
	DBPoolMetrics   *DatabasePoolMetrics   `json:"db_pool_metrics"`
	RedisPoolMetrics *RedisPoolMetrics     `json:"redis_pool_metrics"`
	QPS             float64                `json:"qps"`
	P50Latency      string                 `json:"p50_latency"`
	P95Latency      string                 `json:"p95_latency"`
	P99Latency      string                 `json:"p99_latency"`
}

type RequestMetricsSummary struct {
	TotalRequests    uint64  `json:"total_requests"`
	SuccessfulRequests uint64 `json:"successful_requests"`
	FailedRequests    uint64  `json:"failed_requests"`
	ErrorRate         float64 `json:"error_rate"`
	AvgDuration       string  `json:"avg_duration"`
	MaxDuration       string  `json:"max_duration"`
	MinDuration       string  `json:"min_duration"`
	RequestsPerSecond float64 `json:"requests_per_second"`
}

type EndpointStats struct {
	Path        string
	Method      string
	Count       uint64
	TotalDuration time.Duration
	MaxDuration time.Duration
	ErrorCount uint64
}

var endpointStats = make(map[string]*EndpointStats)
var endpointStatsMu sync.RWMutex

func GetPerformanceMetrics(c *gin.Context) {
	response := &PerformanceMetricsResponse{
		Timestamp: time.Now(),
	}

	response.RequestMetrics = collectRequestMetrics()

	response.SystemMetrics = collectSystemMetrics()

	response.DBPoolMetrics = collectDBPoolMetrics()

	response.RedisPoolMetrics = collectRedisPoolMetrics()

	response.QPS = calculateQPS()
	response.P50Latency = calculatePercentile(50)
	response.P95Latency = calculatePercentile(95)
	response.P99Latency = calculatePercentile(99)

	c.JSON(http.StatusOK, response)
}

func collectRequestMetrics() *RequestMetricsSummary {
	metrics := &RequestMetricsSummary{}

	perfMonitor.mu.RLock()
	total := perfMonitor.requestCount
	errors := perfMonitor.errorCount
	avgDuration := time.Duration(0)
	if perfMonitor.durationCount > 0 {
		avgDuration = perfMonitor.totalDuration / time.Duration(perfMonitor.durationCount)
	}
	perfMonitor.mu.RUnlock()

	metrics.TotalRequests = atomic.LoadUint64(&total)
	metrics.SuccessfulRequests = metrics.TotalRequests - atomic.LoadUint64(&errors)
	metrics.FailedRequests = atomic.LoadUint64(&errors)

	if metrics.TotalRequests > 0 {
		metrics.ErrorRate = float64(metrics.FailedRequests) / float64(metrics.TotalRequests) * 100
	}

	metrics.AvgDuration = avgDuration.String()

	perfMonitor.mu.RLock()
	metrics.MaxDuration = perfMonitor.maxDuration.String()
	metrics.MinDuration = perfMonitor.minDuration.String()
	perfMonitor.mu.RUnlock()

	metrics.RequestsPerSecond = calculateQPS()

	return metrics
}

func collectSystemMetrics() *SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &SystemMetrics{
		GoVersion:     runtime.Version(),
		NumCPU:        runtime.NumCPU(),
		NumGoroutine:  runtime.NumGoroutine(),
		MemAlloc:      m.Alloc,
		MemTotalAlloc: m.TotalAlloc,
		MemSys:        m.Sys,
		MemLookups:    m.Lookups,
		MemMallocs:    m.Mallocs,
		MemFrees:     m.Frees,
		HeapAlloc:    m.HeapAlloc,
		HeapSys:      m.HeapSys,
		HeapIdle:     m.HeapIdle,
		HeapInuse:    m.HeapInuse,
		StackInuse:   m.StackInuse,
		StackSys:     m.StackSys,
		MSpanInuse:   m.MSpanInuse,
		MSpanSys:     m.MSpanSys,
		MCacheInuse:  m.MCacheInuse,
		MCacheSys:    m.MCacheSys,
		BuckHashSys:  m.BuckHashSys,
		GCSys:        m.GCSys,
		OtherSys:     m.OtherSys,
		NextGC:       m.NextGC,
		LastGC:       m.LastGC,
		NumForcedGC:  m.NumForcedGC,
		GCCPUFraction: m.GCCPUFraction,
		PauseTotalNs: m.PauseTotalNs,
		NumGC:        m.NumGC,
	}
}

func collectDBPoolMetrics() *DatabasePoolMetrics {
	metrics := &DatabasePoolMetrics{}

	db := database.GetDB()
	if db == nil {
		return metrics
	}

	sqlDB, err := db.DB()
	if err != nil {
		return metrics
	}

	stats := sqlDB.Stats()
	metrics.MaxOpenConnections = stats.MaxOpenConnections
	metrics.OpenConnections = stats.OpenConnections
	metrics.InUse = stats.InUse
	metrics.Idle = stats.Idle
	metrics.WaitCount = stats.WaitCount
	metrics.WaitDuration = stats.WaitDuration.String()
	metrics.MaxIdleClosed = stats.MaxIdleClosed
	metrics.MaxLifetimeClosed = stats.MaxLifetimeClosed

	return metrics
}

func collectRedisPoolMetrics() *RedisPoolMetrics {
	metrics := &RedisPoolMetrics{}

	client := redis.GetClient()
	if client == nil {
		return metrics
	}

	stats := client.PoolStats()
	metrics.TotalConns = stats.TotalConns
	metrics.IdleConns = stats.IdleConns
	metrics.StaleConns = stats.StaleConns
	metrics.Hits = stats.Hits
	metrics.Misses = stats.Misses
	metrics.Timeouts = stats.Timeouts

	return metrics
}

func calculateQPS() float64 {
	perfMonitor.mu.RLock()
	total := perfMonitor.requestCount
	perfMonitor.mu.RUnlock()

	startTime := time.Now().Add(-1 * time.Minute)
	elapsed := time.Since(startTime)
	if elapsed.Seconds() <= 0 {
		return 0
	}

	return float64(total) / elapsed.Seconds()
}

func calculatePercentile(pctl int) string {
	return "0ms"
}

func RecordRequest(duration time.Duration, isError bool) {
	atomic.AddUint64(&perfMonitor.requestCount, 1)

	if isError {
		atomic.AddUint64(&perfMonitor.errorCount, 1)
	}

	perfMonitor.mu.Lock()
	perfMonitor.totalDuration += duration
	perfMonitor.durationCount++

	if duration > perfMonitor.maxDuration {
		perfMonitor.maxDuration = duration
	}
	if perfMonitor.minDuration == 0 || duration < perfMonitor.minDuration {
		perfMonitor.minDuration = duration
	}
	perfMonitor.mu.Unlock()
}

func RecordEndpointRequest(path, method string, duration time.Duration, isError bool) {
	key := path + ":" + method

	endpointStatsMu.Lock()
	defer endpointStatsMu.Unlock()

	stats, exists := endpointStats[key]
	if !exists {
		stats = &EndpointStats{
			Path:        path,
			Method:      method,
			MaxDuration: duration,
		}
		endpointStats[key] = stats
	}

	atomic.AddUint64(&stats.Count, 1)
	stats.TotalDuration += duration
	if duration > stats.MaxDuration {
		stats.MaxDuration = duration
	}
	if isError {
		atomic.AddUint64(&stats.ErrorCount, 1)
	}
}

func GetEndpointStats(c *gin.Context) {
	endpointStatsMu.RLock()
	defer endpointStatsMu.RUnlock()

	statsList := make([]*EndpointMetrics, 0, len(endpointStats))
	for _, stats := range endpointStats {
		avgDuration := time.Duration(0)
		if stats.Count > 0 {
			avgDuration = stats.TotalDuration / time.Duration(stats.Count)
		}

		errorRate := float64(0)
		if stats.Count > 0 {
			errorRate = float64(stats.ErrorCount) / float64(stats.Count) * 100
		}

		statsList = append(statsList, &EndpointMetrics{
			Path:        stats.Path,
			Method:      stats.Method,
			Count:       atomic.LoadUint64(&stats.Count),
			AvgDuration: avgDuration.String(),
			MaxDuration: stats.MaxDuration.String(),
			MinDuration: "0s",
			ErrorRate:   formatFloat(errorRate),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"endpoints": statsList,
		"timestamp": time.Now(),
	})
}

func formatFloat(f float64) string {
	return json.Number(json.Number(string(rune('0')))).String()
}

type HealthCheckMetrics struct {
	Status           string `json:"status"`
	Timestamp        time.Time `json:"timestamp"`
	DatabaseHealthy  bool   `json:"database_healthy"`
	RedisHealthy     bool   `json:"redis_healthy"`
	Uptime           string `json:"uptime"`
	Version          string `json:"version"`
}

func GetHealthCheckMetrics(c *gin.Context) {
	metrics := &HealthCheckMetrics{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}

	db := database.GetDB()
	if db != nil {
		sqlDB, err := db.DB()
		if err == nil {
			err = sqlDB.Ping()
			metrics.DatabaseHealthy = (err == nil)
		}
	}

	client := redis.GetClient()
	if client != nil {
		err := client.Ping(redis.GetContext()).Err()
		metrics.RedisHealthy = (err == nil)
	}

	if !metrics.DatabaseHealthy || !metrics.RedisHealthy {
		metrics.Status = "degraded"
	}

	c.JSON(http.StatusOK, metrics)
}

func ResetMetrics(c *gin.Context) {
	atomic.StoreUint64(&perfMonitor.requestCount, 0)
	atomic.StoreUint64(&perfMonitor.errorCount, 0)

	perfMonitor.mu.Lock()
	perfMonitor.totalDuration = 0
	perfMonitor.maxDuration = 0
	perfMonitor.minDuration = 0
	perfMonitor.durationCount = 0
	perfMonitor.mu.Unlock()

	endpointStatsMu.Lock()
	endpointStats = make(map[string]*EndpointStats)
	endpointStatsMu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"message": "Metrics reset successfully",
		"timestamp": time.Now(),
	})
}
