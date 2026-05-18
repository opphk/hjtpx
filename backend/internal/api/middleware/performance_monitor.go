package middleware

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type RequestStats struct {
	TotalRequests   atomic.Int64
	SuccessRequests atomic.Int64
	ErrorRequests   atomic.Int64
	TotalLatency    atomic.Int64
	MaxLatency       atomic.Int64
	MinLatency       atomic.Int64
	RequestCount     map[string]int64
	mu              sync.RWMutex
}

type PerformanceMonitor struct {
	stats          *RequestStats
	enabled        bool
	slowThreshold  time.Duration
	slowRequests    []*SlowRequestInfo
	slowMu          sync.Mutex
	maxSlowRequests int
	startTime      time.Time
}

type SlowRequestInfo struct {
	Method     string
	Path       string
	Latency    time.Duration
	Timestamp  time.Time
	StatusCode int
}

type PerformanceMetrics struct {
	TotalRequests   int64
	SuccessRequests int64
	ErrorRequests   int64
	RequestRate     float64
	AvgLatency    time.Duration
	MaxLatency    time.Duration
	MinLatency    time.Duration
	ErrorRate      float64
	Uptime         time.Duration
	SlowRequests   []*SlowRequestInfo
}

func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		stats: &RequestStats{
			RequestCount: make(map[string]int64),
		},
		enabled:        true,
		slowThreshold: 500 * time.Millisecond,
		maxSlowRequests: 100,
		startTime:      time.Now(),
	}
}

func (pm *PerformanceMonitor) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !pm.enabled {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		path := r.URL.Path
		method := r.Method

		lrw := &latencyResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(&lrw, r)

		latency := time.Since(start)

		pm.recordRequest(method, path, latency, lrw.statusCode)
	})
}

func (pm *PerformanceMonitor) recordRequest(method, path string, latency time.Duration, statusCode int) {
	pm.stats.TotalRequests.Add(1)
	pm.stats.mu.Lock()
	pm.stats.RequestCount[method+" "+path]++
	pm.stats.mu.Unlock()

	if statusCode >= 200 && statusCode < 400 {
		pm.stats.SuccessRequests.Add(1)
	} else {
		pm.stats.ErrorRequests.Add(1)
	}

	latencyNs := latency.Nanoseconds()
	pm.stats.TotalLatency.Add(latencyNs)

	// 更新最大/最小延迟
	for {
		currentMax := pm.stats.MaxLatency.Load()
		if latencyNs <= currentMax {
			break
		}
		if pm.stats.MaxLatency.CompareAndSwap(currentMax, latencyNs) {
			break
		}
	}

	// 初始化或更新最小延迟
	if pm.stats.MinLatency.Load() == 0 {
		pm.stats.MinLatency.Store(latencyNs)
	} else {
		for {
			currentMin := pm.stats.MinLatency.Load()
			if latencyNs >= currentMin {
				break
			}
			if pm.stats.MinLatency.CompareAndSwap(currentMin, latencyNs) {
				break
			}
		}
	}

	// 记录慢请求
	if latency >= pm.slowThreshold {
		pm.recordSlowRequest(method, path, latency, statusCode)
	}
}

func (pm *PerformanceMonitor) recordSlowRequest(method, path string, latency time.Duration, statusCode int) {
	pm.slowMu.Lock()
	defer pm.slowMu.Unlock()

	info := &SlowRequestInfo{
		Method:     method,
		Path:       path,
		Latency:    latency,
		Timestamp:  time.Now(),
		StatusCode: statusCode,
	}

	pm.slowRequests = append(pm.slowRequests, info)
	if len(pm.slowRequests) > pm.maxSlowRequests {
		pm.slowRequests = pm.slowRequests[1:]
	}
}

func (pm *PerformanceMonitor) GetMetrics() *PerformanceMetrics {
	total := pm.stats.TotalRequests.Load()
	success := pm.stats.SuccessRequests.Load()
	errors := pm.stats.ErrorRequests.Load()

	var avgLatency time.Duration
	if total > 0 {
		avgLatency = time.Duration(pm.stats.TotalLatency.Load() / total)
	}

	errorRate := 0.0
	if total > 0 {
		errorRate = float64(errors) / float64(total) * 100
	}

	uptime := time.Since(pm.startTime)
	requestRate := 0.0
	if uptime.Seconds() > 0 {
		requestRate = float64(total) / uptime.Seconds()
	}

	pm.slowMu.Lock()
	slowReqs := make([]*SlowRequestInfo, len(pm.slowRequests))
	copy(slowReqs, pm.slowRequests)
	pm.slowMu.Unlock()

	return &PerformanceMetrics{
		TotalRequests:   total,
		SuccessRequests: success,
		ErrorRequests:   errors,
		RequestRate:     requestRate,
		AvgLatency:    avgLatency,
		MaxLatency:    time.Duration(pm.stats.MaxLatency.Load()),
		MinLatency:    time.Duration(pm.stats.MinLatency.Load()),
		ErrorRate:      errorRate,
		Uptime:         uptime,
		SlowRequests:   slowReqs,
	}
}

func (pm *PerformanceMonitor) Reset() {
	pm.stats.TotalRequests.Store(0)
	pm.stats.SuccessRequests.Store(0)
	pm.stats.ErrorRequests.Store(0)
	pm.stats.TotalLatency.Store(0)
	pm.stats.MaxLatency.Store(0)
	pm.stats.MinLatency.Store(0)
	pm.stats.mu.Lock()
	pm.stats.RequestCount = make(map[string]int64)
	pm.stats.mu.Unlock()
	pm.slowMu.Lock()
	pm.slowRequests = nil
	pm.slowMu.Unlock()
	pm.startTime = time.Now()
}

type latencyResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *latencyResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

var (
	globalPerformanceMonitor *PerformanceMonitor
	performanceOnce           sync.Once
)

func InitPerformanceMonitor() {
	performanceOnce.Do(func() {
		globalPerformanceMonitor = NewPerformanceMonitor()
	})
}

func GetPerformanceMonitor() *PerformanceMonitor {
	if globalPerformanceMonitor == nil {
		InitPerformanceMonitor()
	}
	return globalPerformanceMonitor
}

func PerformanceMonitorMiddleware(next http.Handler) http.Handler {
	return GetPerformanceMonitor().Middleware(next)
}
