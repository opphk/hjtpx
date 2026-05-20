package api_performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type MetricsCollector struct {
	config        *MetricsConfig
	metrics      *APIMetrics
	counters     map[string]*Counter
	gauges       map[string]*Gauge
	histograms   map[string]*Histogram
	timers       map[string]*Timer
	percentiles  []float64
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	flushTicker  *time.Ticker
	alertManager *AlertManager
}

type MetricsConfig struct {
	EnableCounters      bool
	EnableGauges        bool
	EnableHistograms    bool
	EnableTimers        bool
	EnablePercentiles   bool
	Percentiles         []float64
	FlushInterval       time.Duration
	RetentionPeriod     time.Duration
	MaxMetrics          int
	EnableAlerting      bool
	AlertThreshold      float64
}

var DefaultMetricsConfig = &MetricsConfig{
	EnableCounters:      true,
	EnableGauges:        true,
	EnableHistograms:    true,
	EnableTimers:        true,
	EnablePercentiles:   true,
	Percentiles:         []float64{0.5, 0.75, 0.95, 0.99, 0.999},
	FlushInterval:       10 * time.Second,
	RetentionPeriod:     1 * time.Hour,
	MaxMetrics:          10000,
	EnableAlerting:      true,
	AlertThreshold:      100.0,
}

type APIMetrics struct {
	TotalRequests       atomic.Int64
	SuccessRequests     atomic.Int64
	ErrorRequests       atomic.Int64
	TimeoutRequests     atomic.Int64
	AvgLatency          atomic.Int64
	MinLatency          atomic.Int64
	MaxLatency          atomic.Int64
	P50Latency          atomic.Int64
	P75Latency          atomic.Int64
	P95Latency          atomic.Int64
	P99Latency          atomic.Int64
	ActiveConnections    atomic.Int64
	PeakConnections     atomic.Int64
	CacheHits           atomic.Int64
	CacheMisses         atomic.Int64
	DBQueries           atomic.Int64
	AvgDBQueryTime      atomic.Int64
	SlowQueries         atomic.Int64
	WorkerPoolUtilization atomic.Int64
	QueueDepth          atomic.Int64
	Retries             atomic.Int64
}

type Counter struct {
	name   string
	value  atomic.Int64
	rate   float64
	rateMu sync.Mutex
	lastValue int64
	lastTime time.Time
}

type Gauge struct {
	name    string
	value   atomic.Int64
}

type Histogram struct {
	name     string
	count    atomic.Int64
	sum      atomic.Int64
	min      atomic.Int64
	max      atomic.Int64
	buckets  []int64
	bucketBoundaries []float64
}

type Timer struct {
	name    string
	count   atomic.Int64
	sum     atomic.Int64
	min     atomic.Int64
	max     atomic.Int64
	values  []float64
	mu      sync.Mutex
}

type AlertManager struct {
	enabled    bool
	thresholds map[string]*AlertThreshold
	alerts     chan *Alert
	mu         sync.RWMutex
}

type AlertThreshold struct {
	MetricName string
	Operator   string
	Value      float64
	Duration   time.Duration
}

type Alert struct {
	ID        string
	Metric    string
	Current   float64
	Threshold float64
	Timestamp time.Time
	Severity  string
	Message   string
}

type MetricsSnapshot struct {
	Timestamp           time.Time
	TotalRequests      int64
	SuccessRate        float64
	AvgLatencyMs       float64
	P50LatencyMs       float64
	P95LatencyMs       float64
	P99LatencyMs       float64
	CacheHitRate       float64
	DBQueryTimeMs      float64
	ActiveConnections   int64
	PeakConnections    int64
}

func NewMetricsCollector(config *MetricsConfig) *MetricsCollector {
	if config == nil {
		config = DefaultMetricsConfig
	}

	ctx, cancel := context.WithCancel(context.Background())

	mc := &MetricsCollector{
		config:        config,
		metrics:       &APIMetrics{},
		counters:      make(map[string]*Counter),
		gauges:       make(map[string]*Gauge),
		histograms:   make(map[string]*Histogram),
		timers:       make(map[string]*Timer),
		percentiles:  config.Percentiles,
		ctx:          ctx,
		cancel:       cancel,
		flushTicker:  time.NewTicker(config.FlushInterval),
		alertManager: NewAlertManager(config.EnableAlerting),
	}

	return mc
}

func NewAlertManager(enabled bool) *AlertManager {
	return &AlertManager{
		enabled:    enabled,
		thresholds: make(map[string]*AlertThreshold),
		alerts:     make(chan *Alert, 100),
	}
}

func (am *AlertManager) AddThreshold(metric string, threshold *AlertThreshold) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.thresholds[metric] = threshold
}

func (am *AlertManager) CheckAndAlert(metric string, value float64) *Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	threshold, exists := am.thresholds[metric]
	if !exists || !am.enabled {
		return nil
	}

	var triggered bool
	switch threshold.Operator {
	case ">":
		triggered = value > threshold.Value
	case ">=":
		triggered = value >= threshold.Value
	case "<":
		triggered = value < threshold.Value
	case "<=":
		triggered = value <= threshold.Value
	case "==":
		triggered = value == threshold.Value
	}

	if triggered {
		return &Alert{
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
			Metric:    metric,
			Current:   value,
			Threshold: threshold.Value,
			Timestamp: time.Now(),
			Severity:  "warning",
			Message:   fmt.Sprintf("%s %s %.2f (threshold: %.2f)", metric, threshold.Operator, value, threshold.Value),
		}
	}

	return nil
}

func (mc *MetricsCollector) IncrementCounter(name string, value int64) {
	if !mc.config.EnableCounters {
		return
	}

	mc.mu.Lock()
	counter, exists := mc.counters[name]
	if !exists {
		counter = &Counter{name: name}
		mc.counters[name] = counter
	}
	mc.mu.Unlock()

	counter.value.Add(value)
}

func (mc *MetricsCollector) SetGauge(name string, value int64) {
	if !mc.config.EnableGauges {
		return
	}

	mc.mu.Lock()
	gauge, exists := mc.gauges[name]
	if !exists {
		gauge = &Gauge{name: name}
		mc.gauges[name] = gauge
	}
	mc.mu.Unlock()

	gauge.value.Store(value)
}

func (mc *MetricsCollector) RecordHistogram(name string, value float64) {
	if !mc.config.EnableHistograms {
		return
	}

	mc.mu.Lock()
	histogram, exists := mc.histograms[name]
	if !exists {
		histogram = &Histogram{
			name:     name,
			buckets:  make([]int64, len(mc.percentiles)),
			bucketBoundaries: mc.percentiles,
		}
		mc.histograms[name] = histogram
	}
	mc.mu.Unlock()

	histogram.count.Add(1)
	histogram.sum.Add(int64(value))

	if value < float64(histogram.min.Load()) || histogram.min.Load() == 0 {
		histogram.min.Store(int64(value))
	}
	if value > float64(histogram.max.Load()) {
		histogram.max.Store(int64(value))
	}

	for i, boundary := range histogram.bucketBoundaries {
		if value <= boundary*1000 {
			histogram.buckets[i]++
		}
	}
}

func (mc *MetricsCollector) RecordTimer(name string, duration time.Duration) {
	if !mc.config.EnableTimers {
		return
	}

	mc.mu.Lock()
	timer, exists := mc.timers[name]
	if !exists {
		timer = &Timer{
			name:   name,
			values: make([]float64, 0, 1000),
		}
		mc.timers[name] = timer
	}
	mc.mu.Unlock()

	value := duration.Seconds() * 1000

	timer.count.Add(1)
	timer.sum.Add(int64(value))

	if value < float64(timer.min.Load()) || timer.min.Load() == 0 {
		timer.min.Store(int64(value))
	}
	if value > float64(timer.max.Load()) {
		timer.max.Store(int64(value))
	}

	timer.mu.Lock()
	timer.values = append(timer.values, value)
	if len(timer.values) > 1000 {
		timer.values = timer.values[1:]
	}
	timer.mu.Unlock()
}

func (mc *MetricsCollector) RecordRequest(success bool, latency time.Duration) {
	mc.metrics.TotalRequests.Add(1)

	if success {
		mc.metrics.SuccessRequests.Add(1)
	} else {
		mc.metrics.ErrorRequests.Add(1)
	}

	mc.updateLatencyStats(latency)
}

func (mc *MetricsCollector) updateLatencyStats(latency time.Duration) {
	latencyMs := latency.Milliseconds()

	old := mc.metrics.AvgLatency.Load()
	count := mc.metrics.TotalRequests.Load()
	if count > 0 {
		newAvg := (old*(count-1) + latencyMs) / count
		mc.metrics.AvgLatency.Store(newAvg)
	}

	if mc.metrics.MinLatency.Load() == 0 || latencyMs < mc.metrics.MinLatency.Load() {
		mc.metrics.MinLatency.Store(latencyMs)
	}

	if latencyMs > mc.metrics.MaxLatency.Load() {
		mc.metrics.MaxLatency.Store(latencyMs)
	}

	mc.RecordHistogram("api.latency", latency.Seconds()*1000)
}

func (mc *MetricsCollector) RecordDBQuery(duration time.Duration, slow bool) {
	mc.metrics.DBQueries.Add(1)

	old := mc.metrics.AvgDBQueryTime.Load()
	count := mc.metrics.DBQueries.Load()
	if count > 0 {
		newAvg := (old*(count-1) + int64(duration.Milliseconds())) / count
		mc.metrics.AvgDBQueryTime.Store(newAvg)
	}

	if slow {
		mc.metrics.SlowQueries.Add(1)
	}
}

func (mc *MetricsCollector) RecordCacheHit() {
	mc.metrics.CacheHits.Add(1)
}

func (mc *MetricsCollector) RecordCacheMiss() {
	mc.metrics.CacheMisses.Add(1)
}

func (mc *MetricsCollector) SetActiveConnections(count int64) {
	mc.metrics.ActiveConnections.Store(count)

	peak := mc.metrics.PeakConnections.Load()
	if count > peak {
		mc.metrics.PeakConnections.Store(count)
	}
}

func (mc *MetricsCollector) SetQueueDepth(depth int64) {
	mc.metrics.QueueDepth.Store(depth)
}

func (mc *MetricsCollector) SetWorkerPoolUtilization(utilization int64) {
	mc.metrics.WorkerPoolUtilization.Store(utilization)
}

func (mc *MetricsCollector) GetSnapshot() *MetricsSnapshot {
	total := mc.metrics.TotalRequests.Load()
	success := mc.metrics.SuccessRequests.Load()

	var successRate float64
	if total > 0 {
		successRate = float64(success) / float64(total) * 100
	}

	cacheHits := mc.metrics.CacheHits.Load()
	cacheMisses := mc.metrics.CacheMisses.Load()
	var cacheHitRate float64
	if cacheHits+cacheMisses > 0 {
		cacheHitRate = float64(cacheHits) / float64(cacheHits+cacheMisses) * 100
	}

	return &MetricsSnapshot{
		Timestamp:          time.Now(),
		TotalRequests:      total,
		SuccessRate:        successRate,
		AvgLatencyMs:       float64(mc.metrics.AvgLatency.Load()),
		P50LatencyMs:       float64(mc.metrics.P50Latency.Load()),
		P95LatencyMs:       float64(mc.metrics.P95Latency.Load()),
		P99LatencyMs:       float64(mc.metrics.P99Latency.Load()),
		CacheHitRate:       cacheHitRate,
		DBQueryTimeMs:      float64(mc.metrics.AvgDBQueryTime.Load()),
		ActiveConnections:  mc.metrics.ActiveConnections.Load(),
		PeakConnections:    mc.metrics.PeakConnections.Load(),
	}
}

func (mc *MetricsCollector) GetStats() map[string]interface{} {
	snapshot := mc.GetSnapshot()

	stats := map[string]interface{}{
		"total_requests":        snapshot.TotalRequests,
		"success_rate":          snapshot.SuccessRate,
		"avg_latency_ms":        snapshot.AvgLatencyMs,
		"min_latency_ms":        mc.metrics.MinLatency.Load(),
		"max_latency_ms":        mc.metrics.MaxLatency.Load(),
		"p50_latency_ms":        snapshot.P50LatencyMs,
		"p95_latency_ms":        snapshot.P95LatencyMs,
		"p99_latency_ms":        snapshot.P99LatencyMs,
		"cache_hit_rate":        snapshot.CacheHitRate,
		"cache_hits":            mc.metrics.CacheHits.Load(),
		"cache_misses":          mc.metrics.CacheMisses.Load(),
		"db_queries":            mc.metrics.DBQueries.Load(),
		"avg_db_query_time_ms": snapshot.DBQueryTimeMs,
		"slow_queries":          mc.metrics.SlowQueries.Load(),
		"active_connections":    snapshot.ActiveConnections,
		"peak_connections":      snapshot.PeakConnections,
		"worker_pool_util":      mc.metrics.WorkerPoolUtilization.Load(),
		"queue_depth":           mc.metrics.QueueDepth.Load(),
		"retries":               mc.metrics.Retries.Load(),
	}

	mc.mu.RLock()
	defer mc.mu.RUnlock()

	counterStats := make(map[string]interface{})
	for name, counter := range mc.counters {
		counterStats[name] = map[string]interface{}{
			"value": counter.value.Load(),
		}
	}
	stats["counters"] = counterStats

	gaugeStats := make(map[string]interface{})
	for name, gauge := range mc.gauges {
		gaugeStats[name] = map[string]interface{}{
			"value": gauge.value.Load(),
		}
	}
	stats["gauges"] = gaugeStats

	histogramStats := make(map[string]interface{})
	for name, histogram := range mc.histograms {
		histogramStats[name] = map[string]interface{}{
			"count": histogram.count.Load(),
			"sum":   histogram.sum.Load(),
			"min":   histogram.min.Load(),
			"max":   histogram.max.Load(),
		}
	}
	stats["histograms"] = histogramStats

	timerStats := make(map[string]interface{})
	for name, timer := range mc.timers {
		timer.mu.Lock()
		values := make([]float64, len(timer.values))
		copy(values, timer.values)
		timer.mu.Unlock()

		var sum int64
		for _, v := range values {
			sum += int64(v)
		}
		var avg float64
		if len(values) > 0 {
			avg = float64(sum) / float64(len(values))
		}

		timerStats[name] = map[string]interface{}{
			"count":     timer.count.Load(),
			"sum_ms":    timer.sum.Load(),
			"avg_ms":    avg,
			"min_ms":    timer.min.Load(),
			"max_ms":    timer.max.Load(),
		}
	}
	stats["timers"] = timerStats

	return stats
}

func (mc *MetricsCollector) Reset() {
	mc.metrics = &APIMetrics{}
	mc.mu.Lock()
	mc.counters = make(map[string]*Counter)
	mc.gauges = make(map[string]*Gauge)
	mc.histograms = make(map[string]*Histogram)
	mc.timers = make(map[string]*Timer)
	mc.mu.Unlock()
}

func (mc *MetricsCollector) Start() {
	go mc.collectSystemMetrics()
}

func (mc *MetricsCollector) Stop() {
	mc.cancel()
}

func (mc *MetricsCollector) collectSystemMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			mc.SetGauge("system.memory.alloc", int64(memStats.Alloc))
			mc.SetGauge("system.memory.total", int64(memStats.TotalAlloc))
			mc.SetGauge("system.memory.sys", int64(memStats.Sys))
			mc.SetGauge("system.gc.count", int64(memStats.NumGC))
			mc.SetGauge("system.goroutines", int64(runtime.NumGoroutine()))
			mc.SetGauge("system.cpu.num", int64(runtime.NumCPU()))

			if mc.config.EnableAlerting {
				alert := mc.alertManager.CheckAndAlert("system.memory.alloc",
					float64(memStats.Alloc)/float64(memStats.Sys)*100)
				if alert != nil {
					mc.handleAlert(alert)
				}
			}
		}
	}
}

func (mc *MetricsCollector) handleAlert(alert *Alert) {
	if alert != nil {
		select {
		case mc.alertManager.alerts <- alert:
		default:
		}
	}
}

type PerformanceProfiler struct {
	enabled   bool
	samples   []*Sample
	mu        sync.RWMutex
	maxSamples int
}

type Sample struct {
	Timestamp    time.Time
	Operation    string
	Duration     time.Duration
	Metadata     map[string]interface{}
	StackTrace  string
}

func NewPerformanceProfiler(maxSamples int) *PerformanceProfiler {
	return &PerformanceProfiler{
		enabled:    true,
		samples:    make([]*Sample, 0, maxSamples),
		maxSamples: maxSamples,
	}
}

func (pp *PerformanceProfiler) Record(operation string, duration time.Duration, metadata map[string]interface{}) {
	if !pp.enabled {
		return
	}

	pp.mu.Lock()
	defer pp.mu.Unlock()

	sample := &Sample{
		Timestamp: time.Now(),
		Operation: operation,
		Duration:  duration,
		Metadata:  metadata,
	}

	pp.samples = append(pp.samples, sample)

	if len(pp.samples) > pp.maxSamples {
		pp.samples = pp.samples[1:]
	}
}

func (pp *PerformanceProfiler) GetSamples() []*Sample {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	samples := make([]*Sample, len(pp.samples))
	copy(samples, pp.samples)
	return samples
}

func (pp *PerformanceProfiler) GetSlowOperations(threshold time.Duration) []*Sample {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	var slowSamples []*Sample
	for _, sample := range pp.samples {
		if sample.Duration > threshold {
			slowSamples = append(slowSamples, sample)
		}
	}

	return slowSamples
}

func (pp *PerformanceProfiler) Reset() {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	pp.samples = make([]*Sample, 0, pp.maxSamples)
}

type HealthChecker struct {
	checks    map[string]*HealthCheck
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	ticker    *time.Ticker
}

type HealthCheck struct {
	Name        string
	Status      string
	Message     string
	LastChecked time.Time
	Interval    time.Duration
	CheckFunc   func() *HealthCheckResult
}

type HealthCheckResult struct {
	Status  string
	Message string
	Details map[string]interface{}
}

func NewHealthChecker(interval time.Duration) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())

	hc := &HealthChecker{
		checks: make(map[string]*HealthCheck),
		ctx:    ctx,
		cancel: cancel,
		ticker: time.NewTicker(interval),
	}

	return hc
}

func (hc *HealthChecker) Register(name string, check *HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[name] = check
}

func (hc *HealthChecker) RunChecks() map[string]*HealthCheckResult {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	results := make(map[string]*HealthCheckResult)
	for name, check := range hc.checks {
		if check.CheckFunc != nil {
			results[name] = check.CheckFunc()
		}
	}

	return results
}

func (hc *HealthChecker) Start() {
	go func() {
		for {
			select {
			case <-hc.ctx.Done():
				return
			case <-hc.ticker.C:
				hc.RunChecks()
			}
		}
	}()
}

func (hc *HealthChecker) Stop() {
	hc.cancel()
}

var globalMetricsCollector *MetricsCollector
var metricsOnce sync.Once

func InitMetricsCollector(config *MetricsConfig) {
	metricsOnce.Do(func() {
		globalMetricsCollector = NewMetricsCollector(config)
		globalMetricsCollector.Start()
	})
}

func GetMetricsCollector() *MetricsCollector {
	if globalMetricsCollector == nil {
		InitMetricsCollector(nil)
	}
	return globalMetricsCollector
}

func StopMetricsCollector() {
	if globalMetricsCollector != nil {
		globalMetricsCollector.Stop()
	}
}
