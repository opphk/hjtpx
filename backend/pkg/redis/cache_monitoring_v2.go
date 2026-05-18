package redis

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type CacheMonitoringServiceV2 struct {
	config           *MonitoringConfigV2
	collector        *MetricsCollectorV2
	alerts           *AlertManagerV2
	hotKeyTracker    *HotKeyTrackerV2
	latencyTracker   *LatencyDistributionTracker
	trendAnalyzer    *TrendAnalyzerV2
	exporter         *MetricsExporterV2
	healthChecker    *CacheHealthCheckerV2
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	started          time.Time
}

type MonitoringConfigV2 struct {
	Enabled                bool
	MetricsEnabled         bool
	MetricsInterval        time.Duration
	AlertEnabled           bool
	HitRateAlertThreshold  float64
	ErrorRateAlertThreshold float64
	LatencyAlertThreshold  time.Duration
	EnableHotKeyTracking   bool
	HotKeyTopN             int
	HotKeyWindow           time.Duration
	EnableLatencyHistogram bool
	ExportPrometheus       bool
	ExportInterval         time.Duration
	ExportEndpoint        string
	RetentionPeriod        time.Duration
}

type MetricsCollectorV2 struct {
	mu               sync.RWMutex
	snapshots        []*MetricsSnapshotV2
	maxSnapshots     int
	currentSnapshot  *MetricsSnapshotV2
}

type MetricsSnapshotV2 struct {
	Timestamp      time.Time
	L1Hits         int64
	L1Misses       int64
	L2Hits         int64
	L2Misses       int64
	TotalHits      int64
	TotalMisses    int64
	HitRate        float64
	L1HitRate      float64
	L2HitRate      float64
	TotalRequests  int64
	Errors         int64
	ErrorRate      float64
	AvgLatency     time.Duration
	P50Latency     time.Duration
	P95Latency     time.Duration
	P99Latency     time.Duration
	P999Latency    time.Duration
	MaxLatency     time.Duration
	MinLatency     time.Duration
	Compressed     int64
	Decompressed   int64
	Evictions      int64
	CurrentL1Size  int64
	CurrentL2Size  int64
	L1MemoryUsage  int64
	L2MemoryUsage  int64
	HotKeysCount   int64
	PeakConcurrent int64
}

type AlertManagerV2 struct {
	mu       sync.RWMutex
	alerts   []AlertV2
	maxAlerts int
	rules     map[string]*AlertRuleV2
}

type AlertRuleV2 struct {
	Name          string
	Metric        string
	Condition     string
	Threshold     float64
	Severity      string
	Duration      time.Duration
	Cooldown      time.Duration
	lastTriggered time.Time
	triggered     bool
}

type AlertV2 struct {
	ID        string
	Name      string
	Type      string
	Severity  string
	Message   string
	Timestamp time.Time
	Key       string
	Metrics   map[string]interface{}
	Resolved  bool
}

type HotKeyTrackerV2 struct {
	mu            sync.RWMutex
	trackedKeys   map[string]*HotKeyInfoV2
	maxKeys       int
	window        time.Duration
	updateCounter int64
}

type HotKeyInfoV2 struct {
	Key           string
	AccessCount   int64
	LastAccess    time.Time
	HitRate       float64
	TotalRequests int64
	AvgLatency    time.Duration
	TotalLatency  int64
	IsHot         bool
}

type LatencyDistributionTracker struct {
	mu          sync.RWMutex
	buckets     map[time.Duration]int64
	samples     []time.Duration
	maxSamples  int
	totalLatency int64
	count       int64
}

type TrendAnalyzerV2 struct {
	mu           sync.RWMutex
	history      []TrendDataPointV2
	windowSize   int
	seasonality  float64
	trend        float64
}

type TrendDataPointV2 struct {
	Timestamp time.Time
	HitRate   float64
	Load      float64
}

type MetricsExporterV2 struct {
	enabled     bool
	format      string
	endpoint    string
	httpClient  *HTTPClientV2
	authToken   string
}

type HTTPClientV2 struct {
	timeout time.Duration
}

type CacheHealthCheckerV2 struct {
	mu               sync.RWMutex
	status           CacheHealthStatusV2
	lastCheck        time.Time
	consecutiveFails int
	maxFails         int
	checks           map[string]HealthCheckV2
}

type HealthCheckV2 struct {
	Name        string
	Status      string
	LastRun     time.Time
	LastSuccess time.Time
	LastFailure time.Time
	Message     string
}

type CacheHealthStatusV2 struct {
	OverallStatus string
	Checks        map[string]HealthCheckV2
	Score         float64
	LastUpdated   time.Time
}

var DefaultMonitoringConfigV2 = &MonitoringConfigV2{
	Enabled:                true,
	MetricsEnabled:         true,
	MetricsInterval:        10 * time.Second,
	AlertEnabled:          true,
	HitRateAlertThreshold:  80.0,
	ErrorRateAlertThreshold: 5.0,
	LatencyAlertThreshold:  100 * time.Millisecond,
	EnableHotKeyTracking:   true,
	HotKeyTopN:             100,
	HotKeyWindow:           30 * time.Minute,
	EnableLatencyHistogram: true,
	ExportPrometheus:       true,
	ExportInterval:         15 * time.Second,
	ExportEndpoint:         "/metrics",
	RetentionPeriod:        24 * time.Hour,
}

func NewCacheMonitoringServiceV2(config *MonitoringConfigV2) *CacheMonitoringServiceV2 {
	if config == nil {
		config = DefaultMonitoringConfigV2
	}

	ctx, cancel := context.WithCancel(context.Background())

	cms := &CacheMonitoringServiceV2{
		config:          config,
		collector:       NewMetricsCollectorV2(1000),
		alerts:          NewAlertManagerV2(1000),
		hotKeyTracker:   NewHotKeyTrackerV2(config.HotKeyTopN, config.HotKeyWindow),
		latencyTracker:  NewLatencyDistributionTracker(10000),
		trendAnalyzer:   NewTrendAnalyzerV2(100),
		exporter:        NewMetricsExporterV2(config.ExportPrometheus, config.ExportEndpoint),
		healthChecker:   NewCacheHealthCheckerV2(5),
		ctx:             ctx,
		cancel:          cancel,
		started:         time.Now(),
	}

	return cms
}

func NewMetricsCollectorV2(maxSnapshots int) *MetricsCollectorV2 {
	if maxSnapshots <= 0 {
		maxSnapshots = 1000
	}

	return &MetricsCollectorV2{
		snapshots:    make([]*MetricsSnapshotV2, 0, maxSnapshots),
		maxSnapshots: maxSnapshots,
	}
}

func (mc *MetricsCollectorV2) RecordSnapshot(snapshot *MetricsSnapshotV2) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.snapshots = append(mc.snapshots, snapshot)
	mc.currentSnapshot = snapshot

	if len(mc.snapshots) > mc.maxSnapshots {
		mc.snapshots = mc.snapshots[1:]
	}
}

func (mc *MetricsCollectorV2) GetCurrentSnapshot() *MetricsSnapshotV2 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.currentSnapshot
}

func (mc *MetricsCollectorV2) GetHistoricalMetrics(window time.Duration) []*MetricsSnapshotV2 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	cutoff := time.Now().Add(-window)
	var snapshots []*MetricsSnapshotV2

	for _, snap := range mc.snapshots {
		if snap.Timestamp.After(cutoff) {
			snapshots = append(snapshots, snap)
		}
	}

	return snapshots
}

func (mc *MetricsCollectorV2) GetAverageHitRate(window time.Duration) float64 {
	snapshots := mc.GetHistoricalMetrics(window)
	if len(snapshots) == 0 {
		return 0
	}

	var total float64
	for _, snap := range snapshots {
		total += snap.HitRate
	}

	return total / float64(len(snapshots))
}

func (mc *MetricsCollectorV2) GetAverageLatency(window time.Duration) time.Duration {
	snapshots := mc.GetHistoricalMetrics(window)
	if len(snapshots) == 0 {
		return 0
	}

	var totalLatency int64
	var totalRequests int64

	for _, snap := range snapshots {
		totalLatency += int64(snap.AvgLatency) * snap.TotalRequests
		totalRequests += snap.TotalRequests
	}

	if totalRequests == 0 {
		return 0
	}

	return time.Duration(totalLatency / totalRequests)
}

func NewAlertManagerV2(maxAlerts int) *AlertManagerV2 {
	return &AlertManagerV2{
		alerts:    make([]AlertV2, 0, maxAlerts),
		maxAlerts: maxAlerts,
		rules:     make(map[string]*AlertRuleV2),
	}
}

func (am *AlertManagerV2) AddAlert(alert AlertV2) {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert.ID = fmt.Sprintf("alert-%d", time.Now().UnixNano())
	am.alerts = append(am.alerts, alert)

	if len(am.alerts) > am.maxAlerts {
		am.alerts = am.alerts[1:]
	}
}

func (am *AlertManagerV2) GetRecentAlerts(limit int) []AlertV2 {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := am.alerts
	if len(alerts) > limit {
		return alerts[len(alerts)-limit:]
	}
	return alerts
}

func (am *AlertManagerV2) GetAlertsBySeverity(severity string) []AlertV2 {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var result []AlertV2
	for _, alert := range am.alerts {
		if alert.Severity == severity {
			result = append(result, alert)
		}
	}
	return result
}

func (am *AlertManagerV2) ResolveAlert(id string) bool {
	am.mu.Lock()
	defer am.mu.Unlock()

	for i := range am.alerts {
		if am.alerts[i].ID == id {
			am.alerts[i].Resolved = true
			return true
		}
	}
	return false
}

func NewHotKeyTrackerV2(maxKeys int, window time.Duration) *HotKeyTrackerV2 {
	return &HotKeyTrackerV2{
		trackedKeys: make(map[string]*HotKeyInfoV2),
		maxKeys:     maxKeys,
		window:      window,
	}
}

func (hkt *HotKeyTrackerV2) RecordAccess(key string, latency time.Duration) {
	hkt.mu.Lock()
	defer hkt.mu.Unlock()

	info, exists := hkt.trackedKeys[key]
	if !exists {
		info = &HotKeyInfoV2{
			Key: key,
		}
		hkt.trackedKeys[key] = info
	}

	atomic.AddInt64(&info.AccessCount, 1)
	atomic.AddInt64(&info.TotalLatency, latency.Nanoseconds())
	atomic.AddInt64(&info.TotalRequests, 1)
	info.LastAccess = time.Now()

	info.AvgLatency = time.Duration(atomic.LoadInt64(&info.TotalLatency) / atomic.LoadInt64(&info.TotalRequests))

	hkt.cleanup()
}

func (hkt *HotKeyTrackerV2) cleanup() {
	cutoff := time.Now().Add(-hkt.window)
	for key, info := range hkt.trackedKeys {
		if info.LastAccess.Before(cutoff) {
			delete(hkt.trackedKeys, key)
		}
	}
}

func (hkt *HotKeyTrackerV2) GetTopHotKeys(n int) []*HotKeyInfoV2 {
	hkt.mu.RLock()
	defer hkt.mu.RUnlock()

	keys := make([]*HotKeyInfoV2, 0, len(hkt.trackedKeys))
	for _, info := range hkt.trackedKeys {
		if atomic.LoadInt64(&info.AccessCount) > 10 {
			info.IsHot = true
			keys = append(keys, info)
		}
	}

	for i := 0; i < len(keys)-1; i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i].AccessCount < keys[j].AccessCount {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	if len(keys) > n {
		return keys[:n]
	}
	return keys
}

func NewLatencyDistributionTracker(maxSamples int) *LatencyDistributionTracker {
	return &LatencyDistributionTracker{
		buckets:    make(map[time.Duration]int64),
		samples:    make([]time.Duration, 0, maxSamples),
		maxSamples: maxSamples,
	}
}

func (ldt *LatencyDistributionTracker) Record(latency time.Duration) {
	ldt.mu.Lock()
	defer ldt.mu.Unlock()

	bucket := ldt.getBucket(latency)
	ldt.buckets[bucket]++

	ldt.samples = append(ldt.samples, latency)
	if len(ldt.samples) > ldt.maxSamples {
		ldt.samples = ldt.samples[1:]
	}

	atomic.AddInt64(&ldt.totalLatency, latency.Nanoseconds())
	atomic.AddInt64(&ldt.count, 1)
}

func (ldt *LatencyDistributionTracker) getBucket(latency time.Duration) time.Duration {
	buckets := []time.Duration{
		1 * time.Microsecond,
		10 * time.Microsecond,
		100 * time.Microsecond,
		500 * time.Microsecond,
		1 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}

	for _, bucket := range buckets {
		if latency <= bucket {
			return bucket
		}
	}

	return 10 * time.Second
}

func (ldt *LatencyDistributionTracker) GetPercentile(p float64) time.Duration {
	ldt.mu.RLock()
	defer ldt.mu.RUnlock()

	if len(ldt.samples) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(ldt.samples))
	copy(sorted, ldt.samples)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := int(float64(len(sorted)) * p)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func (ldt *LatencyDistributionTracker) GetAverageLatency() time.Duration {
	count := atomic.LoadInt64(&ldt.count)
	if count == 0 {
		return 0
	}
	return time.Duration(atomic.LoadInt64(&ldt.totalLatency) / count)
}

func NewTrendAnalyzerV2(windowSize int) *TrendAnalyzerV2 {
	return &TrendAnalyzerV2{
		history:    make([]TrendDataPointV2, 0, windowSize),
		windowSize: windowSize,
	}
}

func (ta *TrendAnalyzerV2) RecordDataPoint(hitRate, load float64) {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	point := TrendDataPointV2{
		Timestamp: time.Now(),
		HitRate:   hitRate,
		Load:      load,
	}

	ta.history = append(ta.history, point)
	if len(ta.history) > ta.windowSize {
		ta.history = ta.history[1:]
	}

	ta.calculateTrend()
}

func (ta *TrendAnalyzerV2) calculateTrend() {
	if len(ta.history) < 2 {
		return
	}

	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(ta.history))

	for i, point := range ta.history {
		x := float64(i)
		y := point.HitRate
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denominator := n*sumX2 - sumX*sumX
	if denominator != 0 {
		ta.trend = (n*sumXY - sumX*sumY) / denominator
	}
}

func (ta *TrendAnalyzerV2) GetTrend() float64 {
	ta.mu.RLock()
	defer ta.mu.RUnlock()
	return ta.trend
}

func (ta *TrendAnalyzerV2) PredictNextHitRate() float64 {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	if len(ta.history) == 0 {
		return 0
	}

	lastHitRate := ta.history[len(ta.history)-1].HitRate
	return lastHitRate + ta.trend
}

func NewMetricsExporterV2(enabled bool, endpoint string) *MetricsExporterV2 {
	return &MetricsExporterV2{
		enabled:   enabled,
		format:    "prometheus",
		endpoint:  endpoint,
	}
}

func (me *MetricsExporterV2) Export(snapshot *MetricsSnapshotV2) string {
	return fmt.Sprintf(`# HELP cache_hit_rate Cache hit rate percentage
# TYPE cache_hit_rate gauge
cache_hit_rate %.2f

# HELP cache_hits_total Total cache hits
# TYPE cache_hits_total counter
cache_hits_total %d

# HELP cache_misses_total Total cache misses
# TYPE cache_misses_total counter
cache_misses_total %d

# HELP cache_requests_total Total cache requests
# TYPE cache_requests_total counter
cache_requests_total %d

# HELP cache_errors_total Total cache errors
# TYPE cache_errors_total counter
cache_errors_total %d

# HELP cache_l1_hit_rate L1 cache hit rate
# TYPE cache_l1_hit_rate gauge
cache_l1_hit_rate %.2f

# HELP cache_l2_hit_rate L2 cache hit rate
# TYPE cache_l2_hit_rate gauge
cache_l2_hit_rate %.2f

# HELP cache_latency_seconds Cache operation latency in seconds
# TYPE cache_latency_seconds gauge
cache_latency_seconds{quantile="0.50"} %.6f
cache_latency_seconds{quantile="0.95"} %.6f
cache_latency_seconds{quantile="0.99"} %.6f
cache_latency_seconds{quantile="0.999"} %.6f

# HELP cache_compressed_total Total number of compressions
# TYPE cache_compressed_total counter
cache_compressed_total %d

# HELP cache_evictions_total Total number of evictions
# TYPE cache_evictions_total counter
cache_evictions_total %d

# HELP cache_l1_size Current L1 cache size
# TYPE cache_l1_size gauge
cache_l1_size %d

# HELP cache_l2_size Current L2 cache size
# TYPE cache_l2_size gauge
cache_l2_size %d
`,
		snapshot.HitRate/100,
		snapshot.TotalHits,
		snapshot.TotalMisses,
		snapshot.TotalRequests,
		snapshot.Errors,
		snapshot.L1HitRate/100,
		snapshot.L2HitRate/100,
		snapshot.AvgLatency.Seconds(),
		snapshot.P95Latency.Seconds(),
		snapshot.P99Latency.Seconds(),
		snapshot.P999Latency.Seconds(),
		snapshot.Compressed,
		snapshot.Evictions,
		snapshot.CurrentL1Size,
		snapshot.CurrentL2Size,
	)
}

func NewCacheHealthCheckerV2(maxFails int) *CacheHealthCheckerV2 {
	return &CacheHealthCheckerV2{
		maxFails: maxFails,
		checks:   make(map[string]HealthCheckV2),
		status: CacheHealthStatusV2{
			Checks: make(map[string]HealthCheckV2),
		},
	}
}

func (chc *CacheHealthCheckerV2) RegisterCheck(name string) {
	chc.mu.Lock()
	defer chc.mu.Unlock()
	chc.checks[name] = HealthCheckV2{Name: name}
}

func (chc *CacheHealthCheckerV2) RunCheck(name string, checkFunc func() error) {
	chc.mu.Lock()
	defer chc.mu.Unlock()

	check, exists := chc.checks[name]
	if !exists {
		return
	}

	check.LastRun = time.Now()
	err := checkFunc()

	if err != nil {
		check.Status = "failed"
		check.LastFailure = time.Now()
		check.Message = err.Error()
		chc.consecutiveFails++
	} else {
		check.Status = "healthy"
		check.LastSuccess = time.Now()
		check.Message = "OK"
		chc.consecutiveFails = 0
	}

	chc.checks[name] = check
	chc.updateOverallStatus()
}

func (chc *CacheHealthCheckerV2) updateOverallStatus() {
	healthyCount := 0
	totalCount := len(chc.checks)

	for _, check := range chc.checks {
		if check.Status == "healthy" {
			healthyCount++
		}
	}

	if chc.consecutiveFails >= chc.maxFails {
		chc.status.OverallStatus = "unhealthy"
		chc.status.Score = 0
	} else if healthyCount == totalCount {
		chc.status.OverallStatus = "healthy"
		chc.status.Score = 100
	} else {
		chc.status.OverallStatus = "degraded"
		chc.status.Score = float64(healthyCount) / float64(totalCount) * 100
	}

	chc.status.LastUpdated = time.Now()
	chc.status.Checks = chc.checks
}

func (cms *CacheMonitoringServiceV2) Start() {
	if !cms.config.Enabled {
		return
	}

	cms.wg.Add(1)
	go cms.collectLoop()

	if cms.config.AlertEnabled {
		cms.wg.Add(1)
		go cms.alertLoop()
	}

	if cms.config.ExportPrometheus {
		cms.wg.Add(1)
		go cms.exportLoop()
	}

	log.Println("[CACHE_MONITORING_V2] Started cache monitoring service")
}

func (cms *CacheMonitoringServiceV2) Stop() {
	cms.cancel()
	cms.wg.Wait()
	log.Println("[CACHE_MONITORING_V2] Stopped cache monitoring service")
}

func (cms *CacheMonitoringServiceV2) collectLoop() {
	defer cms.wg.Done()

	ticker := time.NewTicker(cms.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cms.ctx.Done():
			return
		case <-ticker.C:
			cms.collectMetrics()
		}
	}
}

func (cms *CacheMonitoringServiceV2) collectMetrics() {
	snapshot := cms.createSnapshot()

	cms.collector.RecordSnapshot(snapshot)

	if cms.config.EnableHotKeyTracking {
		cms.hotKeyTracker.cleanup()
	}

	snapshot.HitRate = cms.calculateHitRate(snapshot)
	snapshot.L1HitRate = cms.calculateL1HitRate(snapshot)
	snapshot.L2HitRate = cms.calculateL2HitRate(snapshot)
}

func (cms *CacheMonitoringServiceV2) createSnapshot() *MetricsSnapshotV2 {
	snapshot := &MetricsSnapshotV2{
		Timestamp: time.Now(),
	}

	if mlc := GetTieredCache(); mlc != nil {
		if stats := mlc.GetStats(); stats != nil {
			if v, ok := stats["l1_hits"].(int64); ok {
				snapshot.L1Hits = v
			}
			if v, ok := stats["l1_misses"].(int64); ok {
				snapshot.L1Misses = v
			}
			if v, ok := stats["l2_hits"].(int64); ok {
				snapshot.L2Hits = v
			}
			if v, ok := stats["l2_misses"].(int64); ok {
				snapshot.L2Misses = v
			}
			if v, ok := stats["total_hits"].(int64); ok {
				snapshot.TotalHits = v
			}
			if v, ok := stats["total_misses"].(int64); ok {
				snapshot.TotalMisses = v
			}
			snapshot.TotalRequests = snapshot.TotalHits + snapshot.TotalMisses
			if v, ok := stats["errors"].(int64); ok {
				snapshot.Errors = v
			}
			if v, ok := stats["compressed"].(int64); ok {
				snapshot.Compressed = v
			}
			if v, ok := stats["decompressed"].(int64); ok {
				snapshot.Decompressed = v
			}
		}
	}

	snapshot.AvgLatency = cms.latencyTracker.GetAverageLatency()
	snapshot.P50Latency = cms.latencyTracker.GetPercentile(0.50)
	snapshot.P95Latency = cms.latencyTracker.GetPercentile(0.95)
	snapshot.P99Latency = cms.latencyTracker.GetPercentile(0.99)
	snapshot.P999Latency = cms.latencyTracker.GetPercentile(0.999)

	return snapshot
}

func (cms *CacheMonitoringServiceV2) calculateHitRate(snapshot *MetricsSnapshotV2) float64 {
	total := snapshot.TotalHits + snapshot.TotalMisses
	if total == 0 {
		return 0
	}
	return float64(snapshot.TotalHits) / float64(total) * 100
}

func (cms *CacheMonitoringServiceV2) calculateL1HitRate(snapshot *MetricsSnapshotV2) float64 {
	total := snapshot.L1Hits + snapshot.L1Misses
	if total == 0 {
		return 0
	}
	return float64(snapshot.L1Hits) / float64(total) * 100
}

func (cms *CacheMonitoringServiceV2) calculateL2HitRate(snapshot *MetricsSnapshotV2) float64 {
	total := snapshot.L2Hits + snapshot.L2Misses
	if total == 0 {
		return 0
	}
	return float64(snapshot.L2Hits) / float64(total) * 100
}

func (cms *CacheMonitoringServiceV2) alertLoop() {
	defer cms.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cms.ctx.Done():
			return
		case <-ticker.C:
			cms.checkAlerts()
		}
	}
}

func (cms *CacheMonitoringServiceV2) checkAlerts() {
	snapshot := cms.collector.GetCurrentSnapshot()
	if snapshot == nil {
		return
	}

	if snapshot.HitRate < cms.config.HitRateAlertThreshold {
		cms.alerts.AddAlert(AlertV2{
			Name:     "low_hit_rate",
			Type:     "performance",
			Severity: "warning",
			Message:  fmt.Sprintf("Cache hit rate is %.2f%%, below threshold %.2f%%", snapshot.HitRate, cms.config.HitRateAlertThreshold),
			Metrics: map[string]interface{}{
				"hit_rate":     snapshot.HitRate,
				"threshold":    cms.config.HitRateAlertThreshold,
				"total_hits":   snapshot.TotalHits,
				"total_misses": snapshot.TotalMisses,
			},
		})
	}

	if snapshot.ErrorRate > cms.config.ErrorRateAlertThreshold {
		cms.alerts.AddAlert(AlertV2{
			Name:     "high_error_rate",
			Type:     "error",
			Severity: "critical",
			Message:  fmt.Sprintf("Cache error rate is %.2f%%, above threshold %.2f%%", snapshot.ErrorRate, cms.config.ErrorRateAlertThreshold),
			Metrics: map[string]interface{}{
				"error_rate": snapshot.ErrorRate,
				"errors":     snapshot.Errors,
			},
		})
	}

	if snapshot.P99Latency > cms.config.LatencyAlertThreshold {
		cms.alerts.AddAlert(AlertV2{
			Name:     "high_latency",
			Type:     "performance",
			Severity: "warning",
			Message:  fmt.Sprintf("P99 latency is %v, above threshold %v", snapshot.P99Latency, cms.config.LatencyAlertThreshold),
			Metrics: map[string]interface{}{
				"p99_latency": snapshot.P99Latency,
				"threshold":   cms.config.LatencyAlertThreshold,
			},
		})
	}
}

func (cms *CacheMonitoringServiceV2) exportLoop() {
	defer cms.wg.Done()

	ticker := time.NewTicker(cms.config.ExportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cms.ctx.Done():
			return
		case <-ticker.C:
			cms.exportMetrics()
		}
	}
}

func (cms *CacheMonitoringServiceV2) exportMetrics() {
	snapshot := cms.collector.GetCurrentSnapshot()
	if snapshot == nil {
		return
	}

	metrics := cms.exporter.Export(snapshot)
	if len(metrics) == 0 {
		return
	}
	log.Printf("[CACHE_MONITORING_V2] Exported metrics: hit_rate=%.2f%%, requests=%d, p99=%v",
		snapshot.HitRate, snapshot.TotalRequests, snapshot.P99Latency)
}

func (cms *CacheMonitoringServiceV2) RecordAccess(key string, latency time.Duration, hit bool) {
	if cms.config.EnableHotKeyTracking {
		cms.hotKeyTracker.RecordAccess(key, latency)
	}

	if cms.config.EnableLatencyHistogram {
		cms.latencyTracker.Record(latency)
	}

	snapshot := cms.collector.GetCurrentSnapshot()
	if snapshot != nil {
		cms.trendAnalyzer.RecordDataPoint(snapshot.HitRate, 0)
	}
}

func (cms *CacheMonitoringServiceV2) GetMetrics() map[string]interface{} {
	snapshot := cms.collector.GetCurrentSnapshot()
	if snapshot == nil {
		return nil
	}

	return map[string]interface{}{
		"hit_rate":         snapshot.HitRate,
		"l1_hit_rate":      snapshot.L1HitRate,
		"l2_hit_rate":      snapshot.L2HitRate,
		"total_requests":   snapshot.TotalRequests,
		"total_hits":       snapshot.TotalHits,
		"total_misses":     snapshot.TotalMisses,
		"errors":           snapshot.Errors,
		"error_rate":       snapshot.ErrorRate,
		"avg_latency":      snapshot.AvgLatency,
		"p50_latency":      snapshot.P50Latency,
		"p95_latency":      snapshot.P95Latency,
		"p99_latency":      snapshot.P99Latency,
		"compressed":       snapshot.Compressed,
		"evictions":        snapshot.Evictions,
		"current_l1_size":  snapshot.CurrentL1Size,
		"current_l2_size":  snapshot.CurrentL2Size,
	}
}

func (cms *CacheMonitoringServiceV2) GetHotKeys(n int) []*HotKeyInfoV2 {
	return cms.hotKeyTracker.GetTopHotKeys(n)
}

func (cms *CacheMonitoringServiceV2) GetAlerts(limit int) []AlertV2 {
	return cms.alerts.GetRecentAlerts(limit)
}

func (cms *CacheMonitoringServiceV2) GetHealthStatus() *CacheHealthStatusV2 {
	return &cms.healthChecker.status
}

func (cms *CacheMonitoringServiceV2) GetTrend() float64 {
	return cms.trendAnalyzer.GetTrend()
}

func (cms *CacheMonitoringServiceV2) PredictHitRate() float64 {
	return cms.trendAnalyzer.PredictNextHitRate()
}

var globalCacheMonitoringServiceV2 *CacheMonitoringServiceV2
var globalCacheMonitoringOnceV2 sync.Once

func InitCacheMonitoringServiceV2(config *MonitoringConfigV2) {
	globalCacheMonitoringOnceV2.Do(func() {
		globalCacheMonitoringServiceV2 = NewCacheMonitoringServiceV2(config)
	})
}

func GetCacheMonitoringServiceV2() *CacheMonitoringServiceV2 {
	if globalCacheMonitoringServiceV2 == nil {
		InitCacheMonitoringServiceV2(nil)
	}
	return globalCacheMonitoringServiceV2
}

func StartCacheMonitoringV2() {
	GetCacheMonitoringServiceV2().Start()
}

func StopCacheMonitoringV2() {
	GetCacheMonitoringServiceV2().Stop()
}
