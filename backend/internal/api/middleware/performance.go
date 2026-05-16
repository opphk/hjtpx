package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type PerformanceMiddleware struct {
	enableTracing    bool
	enableMetrics    bool
	requestBuffer    chan *RequestInfo
	bufferSize       int
	flushInterval    time.Duration
	stopCh          chan struct{}
}

type RequestInfo struct {
	Method       string
	Path         string
	StatusCode   int
	Duration     time.Duration
	BytesIn      int64
	BytesOut     int64
	ClientIP     string
	UserAgent    string
	Timestamp    time.Time
}

type LatencyTracker struct {
	mu          sync.RWMutex
	latencies   []time.Duration
	maxSamples  int
	percentiles  []int
}

var globalLatencyTracker *LatencyTracker

func init() {
	globalLatencyTracker = NewLatencyTracker(10000, []int{50, 75, 90, 95, 99})
}

func NewLatencyTracker(maxSamples int, percentiles []int) *LatencyTracker {
	return &LatencyTracker{
		latencies:  make([]time.Duration, 0, maxSamples),
		maxSamples: maxSamples,
		percentiles: percentiles,
	}
}

func (lt *LatencyTracker) Record(duration time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	lt.latencies = append(lt.latencies, duration)

	if len(lt.latencies) > lt.maxSamples {
		lt.latencies = lt.latencies[1:]
	}
}

func (lt *LatencyTracker) GetPercentile(pctl int) time.Duration {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if len(lt.latencies) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(lt.latencies))
	copy(sorted, lt.latencies)

	quickSort(sorted, 0, len(sorted)-1)

	index := int(float64(len(sorted)) * float64(pctl) / 100)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

func quickSort(arr []time.Duration, low, high int) {
	if low < high {
		pivot := partition(arr, low, high)
		quickSort(arr, low, pivot-1)
		quickSort(arr, pivot+1, high)
	}
}

func partition(arr []time.Duration, low, high int) int {
	pivot := arr[high]
	i := low - 1

	for j := low; j < high; j++ {
		if arr[j] <= pivot {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}

	arr[i+1], arr[high] = arr[high], arr[i+1]
	return i + 1
}

func NewPerformanceMiddleware() *PerformanceMiddleware {
	return &PerformanceMiddleware{
		enableTracing: true,
		enableMetrics: true,
		bufferSize:    10000,
		flushInterval: 5 * time.Second,
		stopCh:        make(chan struct{}),
	}
}

func (pm *PerformanceMiddleware) SetBufferSize(size int) {
	pm.bufferSize = size
}

func (pm *PerformanceMiddleware) SetFlushInterval(interval time.Duration) {
	pm.flushInterval = interval
}

func (pm *PerformanceMiddleware) EnableTracing(enable bool) {
	pm.enableTracing = enable
}

func (pm *PerformanceMiddleware) EnableMetrics(enable bool) {
	pm.enableMetrics = enable
}

func (pm *PerformanceMiddleware) Handler() gin.HandlerFunc {
	pm.requestBuffer = make(chan *RequestInfo, pm.bufferSize)

	go pm.processRequests()

	return func(c *gin.Context) {
		if !pm.enableTracing && !pm.enableMetrics {
			c.Next()
			return
		}

		start := time.Now()

		c.Next()

		duration := time.Since(start)

		if pm.enableMetrics {
			globalLatencyTracker.Record(duration)
		}

		if pm.enableTracing {
			info := &RequestInfo{
				Method:     c.Request.Method,
				Path:       c.Request.URL.Path,
				StatusCode: c.Writer.Status(),
				Duration:   duration,
				BytesIn:    c.Request.ContentLength,
				BytesOut:   int64(c.Writer.Size()),
				ClientIP:   c.ClientIP(),
				UserAgent:  c.Request.UserAgent(),
				Timestamp:  start,
			}

			select {
			case pm.requestBuffer <- info:
			default:
			}
		}
	}
}

func (pm *PerformanceMiddleware) processRequests() {
	ticker := time.NewTicker(pm.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.stopCh:
			return
		case <-ticker.C:
			pm.flush()
		case info := <-pm.requestBuffer:
			if info != nil {
				pm.processRequest(info)
			}
		}
	}
}

func (pm *PerformanceMiddleware) processRequest(info *RequestInfo) {
}

func (pm *PerformanceMiddleware) flush() {
}

func (pm *PerformanceMiddleware) Stop() {
	close(pm.stopCh)
}

func GetLatencyTracker() *LatencyTracker {
	return globalLatencyTracker
}

type ResponseWriter struct {
	gin.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

func NewResponseWriter(w gin.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:    http.StatusOK,
	}
}

func (rw *ResponseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *ResponseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

func (rw *ResponseWriter) StatusCode() int {
	return rw.statusCode
}

func (rw *ResponseWriter) BytesWritten() int {
	return rw.bytesWritten
}

type PoolStatus struct {
	MaxPoolSize     int `json:"max_pool_size"`
	CurrentPoolSize int `json:"current_pool_size"`
	IdleConnections int `json:"idle_connections"`
	ActiveRequests  int `json:"active_requests"`
	WaitCount       int64 `json:"wait_count"`
	WaitTime        time.Duration `json:"wait_time"`
}

type PerformanceStats struct {
	TotalRequests   int64 `json:"total_requests"`
	SuccessRequests int64 `json:"success_requests"`
	FailedRequests  int64 `json:"failed_requests"`
	AverageLatency  time.Duration `json:"average_latency"`
	P50Latency      time.Duration `json:"p50_latency"`
	P95Latency      time.Duration `json:"p95_latency"`
	P99Latency      time.Duration `json:"p99_latency"`
	MaxLatency      time.Duration `json:"max_latency"`
	MinLatency      time.Duration `json:"min_latency"`
	RequestsPerSec   float64 `json:"requests_per_second"`
}

var (
	stats      = &PerformanceStats{}
	statsMu    sync.RWMutex
	startTime  = time.Now()
)

func RecordStats(statusCode int, duration time.Duration) {
	statsMu.Lock()
	defer statsMu.Unlock()

	stats.TotalRequests++

	if statusCode >= 200 && statusCode < 400 {
		stats.SuccessRequests++
	} else {
		stats.FailedRequests++
	}

	if stats.MaxLatency == 0 || duration > stats.MaxLatency {
		stats.MaxLatency = duration
	}
	if stats.MinLatency == 0 || duration < stats.MinLatency {
		stats.MinLatency = duration
	}

	if stats.TotalRequests > 0 {
		stats.AverageLatency = time.Duration(float64(stats.AverageLatency) * float64(stats.TotalRequests-1) / float64(stats.TotalRequests))
		stats.AverageLatency += duration / time.Duration(stats.TotalRequests)
	}

	elapsed := time.Since(startTime).Seconds()
	if elapsed > 0 {
		stats.RequestsPerSec = float64(stats.TotalRequests) / elapsed
	}
}

func GetPerformanceStats() *PerformanceStats {
	statsMu.RLock()
	defer statsMu.RUnlock()

	stats.P50Latency = globalLatencyTracker.GetPercentile(50)
	stats.P95Latency = globalLatencyTracker.GetPercentile(95)
	stats.P99Latency = globalLatencyTracker.GetPercentile(99)

	return &PerformanceStats{
		TotalRequests:   stats.TotalRequests,
		SuccessRequests: stats.SuccessRequests,
		FailedRequests:  stats.FailedRequests,
		AverageLatency:  stats.AverageLatency,
		P50Latency:      stats.P50Latency,
		P95Latency:      stats.P95Latency,
		P99Latency:      stats.P99Latency,
		MaxLatency:      stats.MaxLatency,
		MinLatency:      stats.MinLatency,
		RequestsPerSec:   stats.RequestsPerSec,
	}
}

func ResetPerformanceStats() {
	statsMu.Lock()
	defer statsMu.Unlock()

	stats = &PerformanceStats{}
	startTime = time.Now()

	globalLatencyTracker = NewLatencyTracker(10000, []int{50, 75, 90, 95, 99})
}
