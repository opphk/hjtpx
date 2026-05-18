package handler

import (
	"net/http"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type PerformanceHandler struct {
	requestCount        int64
	errorCount          int64
	totalDuration       int64
	peakConcurrency     int32
	currentConcurrency  int32
}

var perfHandler = &PerformanceHandler{}

func (h *PerformanceHandler) recordRequest(duration time.Duration) {
	atomic.AddInt64(&h.requestCount, 1)
	atomic.AddInt64(&h.totalDuration, int64(duration.Milliseconds()))
}

func (h *PerformanceHandler) recordError() {
	atomic.AddInt64(&h.errorCount, 1)
}

func (h *PerformanceHandler) startRequest() func() {
	atomic.AddInt32(&h.currentConcurrency, 1)

	current := atomic.LoadInt32(&h.currentConcurrency)
	for {
		peak := atomic.LoadInt32(&h.peakConcurrency)
		if current <= peak {
			break
		}
		if atomic.CompareAndSwapInt32(&h.peakConcurrency, peak, current) {
			break
		}
	}

	return func() {
		atomic.AddInt32(&h.currentConcurrency, -1)
	}
}

func GetPerformanceHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		finish := perfHandler.startRequest()
		defer finish()

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		perfHandler.recordRequest(duration)

		if c.Writer.Status() >= 400 {
			perfHandler.recordError()
		}

		c.Header("X-Response-Time-Ms", duration.String())
	}
}

type PerformanceMetricsHandler struct{}

func NewPerformanceMetricsHandler() *PerformanceMetricsHandler {
	return &PerformanceMetricsHandler{}
}

func (h *PerformanceMetricsHandler) GetMetrics(c *gin.Context) {
	metrics := h.collectAllMetrics()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    metrics,
	})
}

func (h *PerformanceMetricsHandler) collectAllMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	metrics["request"] = h.getRequestMetrics()
	metrics["memory"] = h.getMemoryMetrics()
	metrics["goroutine"] = h.getGoroutineMetrics()
	metrics["gc"] = h.getGCMetrics()
	metrics["cache"] = h.getCacheMetrics()

	return metrics
}

func (h *PerformanceMetricsHandler) getRequestMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_requests":        atomic.LoadInt64(&perfHandler.requestCount),
		"total_errors":          atomic.LoadInt64(&perfHandler.errorCount),
		"peak_concurrency":      atomic.LoadInt32(&perfHandler.peakConcurrency),
		"current_concurrency":   atomic.LoadInt32(&perfHandler.currentConcurrency),
	}
}

func (h *PerformanceMetricsHandler) getMemoryMetrics() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"alloc":       memStats.Alloc / 1024 / 1024,
		"total_alloc": memStats.TotalAlloc / 1024 / 1024,
		"sys":         memStats.Sys / 1024 / 1024,
		"gc_runs":     memStats.NumGC,
		"goroutines":  runtime.NumGoroutine(),
		"cpu_count":   runtime.NumCPU(),
	}
}

func (h *PerformanceMetricsHandler) getGoroutineMetrics() map[string]interface{} {
	return map[string]interface{}{
		"count":     runtime.NumGoroutine(),
		"cpu_count": runtime.NumCPU(),
	}
}

func (h *PerformanceMetricsHandler) getGCMetrics() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"num_gc":        memStats.NumGC,
		"pause_total_ns": memStats.PauseTotalNs,
	}
}

func (h *PerformanceMetricsHandler) getCacheMetrics() map[string]interface{} {
	metrics := redis.GetConnectionMetrics()
	if metrics == nil {
		return nil
	}

	return map[string]interface{}{
		"total_connections":  metrics.TotalConnections,
		"active_connections": metrics.ActiveConnections,
		"idle_connections":   metrics.IdleConnections,
		"timeouts":           metrics.Timeouts,
		"hit_rate":          metrics.HitRate,
	}
}

func (h *PerformanceMetricsHandler) GetRedisMetrics(c *gin.Context) {
	redisMetrics := redis.GetConnectionMetrics()
	if redisMetrics == nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": map[string]interface{}{
			"total_connections":  redisMetrics.TotalConnections,
			"active_connections": redisMetrics.ActiveConnections,
			"idle_connections":   redisMetrics.IdleConnections,
			"stale_connections":  redisMetrics.StaleConnections,
			"timeouts":           redisMetrics.Timeouts,
			"hit_rate":          redisMetrics.HitRate,
		},
	})
}

func (h *PerformanceMetricsHandler) GetPoolMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": map[string]interface{}{
			"worker_count":      0,
			"active_count":      0,
			"queued_count":      0,
			"total_tasks":       0,
			"completed":         0,
			"failed":            0,
			"utilization":       0,
			"queue_usage":       0,
		},
	})
}

func (h *PerformanceMetricsHandler) GetCompressionStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": map[string]interface{}{
			"requests_compressed": 0,
			"bytes_saved":        0,
			"compression_ratio":  0,
		},
	})
}

func (h *PerformanceMetricsHandler) GetSystemMetrics(c *gin.Context) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	metrics := map[string]interface{}{
		"memory": map[string]interface{}{
			"alloc":       memStats.Alloc,
			"total_alloc": memStats.TotalAlloc,
			"sys":         memStats.Sys,
			"gc_runs":     memStats.NumGC,
			"gc_pause_ns": memStats.PauseNs,
		},
		"runtime": map[string]interface{}{
			"goroutines": runtime.NumGoroutine(),
			"cpu_count":  runtime.NumCPU(),
			"go_version": runtime.Version(),
		},
		"requests": map[string]interface{}{
			"total":             atomic.LoadInt64(&perfHandler.requestCount),
			"errors":            atomic.LoadInt64(&perfHandler.errorCount),
			"peak_concurrent":   atomic.LoadInt32(&perfHandler.peakConcurrency),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    metrics,
	})
}

func ResetPerformanceMetrics() {
	atomic.StoreInt64(&perfHandler.requestCount, 0)
	atomic.StoreInt64(&perfHandler.errorCount, 0)
	atomic.StoreInt64(&perfHandler.totalDuration, 0)
	atomic.StoreInt32(&perfHandler.peakConcurrency, 0)
	atomic.StoreInt32(&perfHandler.currentConcurrency, 0)
}
