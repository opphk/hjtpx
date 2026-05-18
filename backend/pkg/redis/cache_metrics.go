package redis

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type CacheMetricsMonitor struct {
	mu                  sync.RWMutex
	enabled             bool
	collectInterval     time.Duration
	metricsHistory      []*CacheMetricSnapshot
	maxHistorySize      int
	l1HitRate           float64
	l2HitRate           float64
	totalHitRate        float64
	totalHits           atomic.Int64
	totalMisses         atomic.Int64
	l1Hits              atomic.Int64
	l1Misses            atomic.Int64
	l2Hits              atomic.Int64
	l2Misses            atomic.Int64
	totalSets           atomic.Int64
	totalDeletes        atomic.Int64
	totalErrors         atomic.Int64
	totalEvictions      atomic.Int64
	totalBytesRead      atomic.Int64
	totalBytesWritten   atomic.Int64
	avgGetLatency       atomic.Int64
	avgSetLatency       atomic.Int64
	lastSnapshot        *CacheMetricSnapshot
	alerts              []*CacheMetricAlert
	alertThresholds     *CacheAlertThresholds
}

type CacheMetricSnapshot struct {
	Timestamp           time.Time
	TotalHits          int64
	TotalMisses        int64
	L1Hits            int64
	L1Misses          int64
	L2Hits            int64
	L2Misses          int64
	HitRate           float64
	L1HitRate         float64
	L2HitRate         float64
	TotalSets         int64
	TotalDeletes      int64
	TotalErrors       int64
	TotalEvictions    int64
	BytesRead         int64
	BytesWritten      int64
	AvgGetLatencyMs   float64
	AvgSetLatencyMs   float64
	CurrentL1Size     int64
}

type CacheMetricAlert struct {
	Timestamp    time.Time
	AlertType    string
	Message      string
	Severity     string
	MetricValue  float64
	Threshold    float64
}

type CacheAlertThresholds struct {
	LowHitRateThreshold     float64
	HighErrorRateThreshold  float64
	HighLatencyThreshold    time.Duration
	LowL1HitRateThreshold   float64
	HighEvictionThreshold    int64
}

var DefaultAlertThresholds = &CacheAlertThresholds{
	LowHitRateThreshold:    80.0,
	HighErrorRateThreshold: 0.01,
	HighLatencyThreshold:   10 * time.Millisecond,
	LowL1HitRateThreshold:  60.0,
	HighEvictionThreshold:  1000,
}

func NewCacheMetricsMonitor(interval time.Duration) *CacheMetricsMonitor {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	
	monitor := &CacheMetricsMonitor{
		enabled:         true,
		collectInterval: interval,
		metricsHistory: make([]*CacheMetricSnapshot, 0),
		maxHistorySize:  360,
		alertThresholds: DefaultAlertThresholds,
	}
	
	go monitor.startCollection()
	return monitor
}

func (m *CacheMetricsMonitor) startCollection() {
	ticker := time.NewTicker(m.collectInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		if !m.enabled {
			continue
		}
		m.collectSnapshot()
	}
}

func (m *CacheMetricsMonitor) collectSnapshot() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	snapshot := &CacheMetricSnapshot{
		Timestamp:         time.Now(),
		TotalHits:        m.totalHits.Load(),
		TotalMisses:      m.totalMisses.Load(),
		L1Hits:          m.l1Hits.Load(),
		L1Misses:        m.l1Misses.Load(),
		L2Hits:          m.l2Hits.Load(),
		L2Misses:        m.l2Misses.Load(),
		TotalSets:       m.totalSets.Load(),
		TotalDeletes:    m.totalDeletes.Load(),
		TotalErrors:    m.totalErrors.Load(),
		TotalEvictions: m.totalEvictions.Load(),
		BytesRead:      m.totalBytesRead.Load(),
		BytesWritten:   m.totalBytesWritten.Load(),
	}
	
	total := snapshot.TotalHits + snapshot.TotalMisses
	if total > 0 {
		snapshot.HitRate = float64(snapshot.TotalHits) / float64(total) * 100
	}
	
	l1Total := snapshot.L1Hits + snapshot.L1Misses
	if l1Total > 0 {
		snapshot.L1HitRate = float64(snapshot.L1Hits) / float64(l1Total) * 100
	}
	
	l2Total := snapshot.L2Hits + snapshot.L2Misses
	if l2Total > 0 {
		snapshot.L2HitRate = float64(snapshot.L2Hits) / float64(l2Total) * 100
	}
	
	totalGets := m.totalHits.Load() + m.totalMisses.Load()
	if totalGets > 0 {
		snapshot.AvgGetLatencyMs = float64(m.avgGetLatency.Load()) / float64(totalGets) / 1e6
	}
	
	totalSets := m.totalSets.Load()
	if totalSets > 0 {
		snapshot.AvgSetLatencyMs = float64(m.avgSetLatency.Load()) / float64(totalSets) / 1e6
	}
	
	snapshot.CurrentL1Size = totalSets + m.totalHits.Load()
	
	m.metricsHistory = append(m.metricsHistory, snapshot)
	if len(m.metricsHistory) > m.maxHistorySize {
		m.metricsHistory = m.metricsHistory[1:]
	}
	
	m.lastSnapshot = snapshot
	m.checkAlerts(snapshot)
}

func (m *CacheMetricsMonitor) checkAlerts(snapshot *CacheMetricSnapshot) {
	if snapshot.HitRate < m.alertThresholds.LowHitRateThreshold {
		m.addAlert(CacheMetricAlert{
			Timestamp:   time.Now(),
			AlertType:   "LowHitRate",
			Message:     fmt.Sprintf("Cache hit rate %.2f%% below threshold %.2f%%", snapshot.HitRate, m.alertThresholds.LowHitRateThreshold),
			Severity:    "warning",
			MetricValue: snapshot.HitRate,
			Threshold:   m.alertThresholds.LowHitRateThreshold,
		})
	}
	
	totalOps := snapshot.TotalSets + snapshot.TotalDeletes
	if totalOps > 0 {
		errorRate := float64(snapshot.TotalErrors) / float64(totalOps)
		if errorRate > m.alertThresholds.HighErrorRateThreshold {
			m.addAlert(CacheMetricAlert{
				Timestamp:   time.Now(),
				AlertType:   "HighErrorRate",
				Message:     fmt.Sprintf("Cache error rate %.4f%% exceeds threshold %.4f%%", errorRate, m.alertThresholds.HighErrorRateThreshold),
				Severity:    "critical",
				MetricValue: errorRate,
				Threshold:   m.alertThresholds.HighErrorRateThreshold,
			})
		}
	}
	
	if snapshot.AvgGetLatencyMs > float64(m.alertThresholds.HighLatencyThreshold)/float64(time.Millisecond) {
		m.addAlert(CacheMetricAlert{
			Timestamp:   time.Now(),
			AlertType:   "HighLatency",
			Message:     fmt.Sprintf("Cache get latency %.2fms exceeds threshold %.2fms", snapshot.AvgGetLatencyMs, float64(m.alertThresholds.HighLatencyThreshold)/float64(time.Millisecond)),
			Severity:    "warning",
			MetricValue: snapshot.AvgGetLatencyMs,
			Threshold:   float64(m.alertThresholds.HighLatencyThreshold)/float64(time.Millisecond),
		})
	}
	
	if snapshot.L1HitRate < m.alertThresholds.LowL1HitRateThreshold && snapshot.L1Misses > 100 {
		m.addAlert(CacheMetricAlert{
			Timestamp:   time.Now(),
			AlertType:   "LowL1HitRate",
			Message:     fmt.Sprintf("L1 cache hit rate %.2f%% below threshold %.2f%%", snapshot.L1HitRate, m.alertThresholds.LowL1HitRateThreshold),
			Severity:    "warning",
			MetricValue: snapshot.L1HitRate,
			Threshold:   m.alertThresholds.LowL1HitRateThreshold,
		})
	}
}

func (m *CacheMetricsMonitor) addAlert(alert CacheMetricAlert) {
	m.alerts = append(m.alerts, &alert)
	if len(m.alerts) > 100 {
		m.alerts = m.alerts[1:]
	}
	
	switch alert.Severity {
	case "critical":
		log.Printf("[CACHE_ALERT_CRITICAL] %s", alert.Message)
	case "warning":
		log.Printf("[CACHE_ALERT_WARNING] %s", alert.Message)
	}
}

func (m *CacheMetricsMonitor) RecordHit(isL1 bool) {
	m.totalHits.Add(1)
	if isL1 {
		m.l1Hits.Add(1)
	} else {
		m.l2Hits.Add(1)
	}
}

func (m *CacheMetricsMonitor) RecordMiss(isL1 bool) {
	m.totalMisses.Add(1)
	if isL1 {
		m.l1Misses.Add(1)
	} else {
		m.l2Misses.Add(1)
	}
}

func (m *CacheMetricsMonitor) RecordSet(latency time.Duration) {
	m.totalSets.Add(1)
	m.avgSetLatency.Add(latency.Nanoseconds())
}

func (m *CacheMetricsMonitor) RecordGet(latency time.Duration) {
	m.avgGetLatency.Add(latency.Nanoseconds())
}

func (m *CacheMetricsMonitor) RecordDelete() {
	m.totalDeletes.Add(1)
}

func (m *CacheMetricsMonitor) RecordError() {
	m.totalErrors.Add(1)
}

func (m *CacheMetricsMonitor) RecordEviction() {
	m.totalEvictions.Add(1)
}

func (m *CacheMetricsMonitor) RecordBytesRead(bytes int64) {
	m.totalBytesRead.Add(bytes)
}

func (m *CacheMetricsMonitor) RecordBytesWritten(bytes int64) {
	m.totalBytesWritten.Add(bytes)
}

func (m *CacheMetricsMonitor) GetCurrentMetrics() *CacheMetricSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.lastSnapshot == nil {
		return &CacheMetricSnapshot{}
	}
	
	return m.lastSnapshot
}

func (m *CacheMetricsMonitor) GetMetricsHistory(limit int) []*CacheMetricSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if limit <= 0 || limit > len(m.metricsHistory) {
		limit = len(m.metricsHistory)
	}
	
	history := make([]*CacheMetricSnapshot, limit)
	copy(history, m.metricsHistory[len(m.metricsHistory)-limit:])
	return history
}

func (m *CacheMetricsMonitor) GetAlerts(limit int) []*CacheMetricAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if limit <= 0 || limit > len(m.alerts) {
		limit = len(m.alerts)
	}
	
	alerts := make([]*CacheMetricAlert, limit)
	copy(alerts, m.alerts[len(m.alerts)-limit:])
	return alerts
}

func (m *CacheMetricsMonitor) GetPerformanceReport() *CachePerformanceReport {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	snapshot := m.lastSnapshot
	if snapshot == nil {
		snapshot = &CacheMetricSnapshot{}
	}
	
	report := &CachePerformanceReport{
		Timestamp: time.Now(),
		CurrentMetrics: *snapshot,
		AggregatedMetrics: m.calculateAggregatedMetrics(),
		TrendAnalysis: m.analyzeTrends(),
		Recommendations: m.generateRecommendations(),
	}
	
	return report
}

func (m *CacheMetricsMonitor) calculateAggregatedMetrics() AggregatedCacheMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var totalHits, totalMisses, l1Hits, l1Misses int64
	var totalSets, totalDeletes, totalErrors, totalEvictions int64
	var totalGetLatency, totalSetLatency int64
	var getCount, setCount int64
	
	for _, snap := range m.metricsHistory {
		totalHits += snap.TotalHits
		totalMisses += snap.TotalMisses
		l1Hits += snap.L1Hits
		l1Misses += snap.L1Misses
		totalSets += snap.TotalSets
		totalDeletes += snap.TotalDeletes
		totalErrors += snap.TotalErrors
		totalEvictions += snap.TotalEvictions
	}
	
	avgGetLat := float64(totalGetLatency) / float64(getCount) / 1e6
	avgSetLat := float64(totalSetLatency) / float64(setCount) / 1e6
	
	return AggregatedCacheMetrics{
		TotalHits:        totalHits,
		TotalMisses:      totalMisses,
		OverallHitRate:   calculateHitRate(totalHits, totalMisses),
		L1HitRate:        calculateHitRate(l1Hits, l1Misses),
		TotalSets:        totalSets,
		TotalDeletes:     totalDeletes,
		TotalErrors:      totalErrors,
		TotalEvictions:   totalEvictions,
		AvgGetLatencyMs: avgGetLat,
		AvgSetLatencyMs: avgSetLat,
		SnapshotsCount:   len(m.metricsHistory),
	}
}

func (m *CacheMetricsMonitor) analyzeTrends() CacheTrendAnalysis {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if len(m.metricsHistory) < 2 {
		return CacheTrendAnalysis{}
	}
	
	recent := m.metricsHistory[len(m.metricsHistory)-1]
	older := m.metricsHistory[len(m.metricsHistory)/2]
	
	return CacheTrendAnalysis{
		HitRateTrend:     recent.HitRate - older.HitRate,
		L1HitRateTrend:   recent.L1HitRate - older.L1HitRate,
		LatencyTrend:     recent.AvgGetLatencyMs - older.AvgGetLatencyMs,
		EvictionTrend:    recent.TotalEvictions - older.TotalEvictions,
		ErrorRateTrend:   calculateErrorRate(recent.TotalErrors, recent.TotalSets+recent.TotalDeletes) - 
			                calculateErrorRate(older.TotalErrors, older.TotalSets+older.TotalDeletes),
	}
}

func (m *CacheMetricsMonitor) generateRecommendations() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var recommendations []string
	
	if m.lastSnapshot != nil {
		if m.lastSnapshot.HitRate < 80 {
			recommendations = append(recommendations, "建议增加缓存预热策略以提高命中率")
		}
		if m.lastSnapshot.L1HitRate < 50 {
			recommendations = append(recommendations, "L1缓存命中率偏低，考虑增加L1缓存容量")
		}
		if m.lastSnapshot.AvgGetLatencyMs > 5 {
			recommendations = append(recommendations, "缓存延迟偏高，检查网络或Redis性能")
		}
		if m.lastSnapshot.TotalEvictions > 100 {
			recommendations = append(recommendations, "缓存驱逐频繁，考虑增加缓存容量")
		}
	}
	
	if len(m.metricsHistory) >= 60 {
		trend := m.analyzeTrends()
		if trend.HitRateTrend < -5 {
			recommendations = append(recommendations, "缓存命中率呈下降趋势，需要检查缓存策略")
		}
		if trend.LatencyTrend > 2 {
			recommendations = append(recommendations, "缓存延迟呈上升趋势")
		}
	}
	
	return recommendations
}

type AggregatedCacheMetrics struct {
	TotalHits        int64
	TotalMisses      int64
	OverallHitRate   float64
	L1HitRate        float64
	TotalSets        int64
	TotalDeletes     int64
	TotalErrors      int64
	TotalEvictions   int64
	AvgGetLatencyMs  float64
	AvgSetLatencyMs  float64
	SnapshotsCount   int
}

type CacheTrendAnalysis struct {
	HitRateTrend    float64
	L1HitRateTrend  float64
	LatencyTrend    float64
	EvictionTrend   int64
	ErrorRateTrend  float64
}

type CachePerformanceReport struct {
	Timestamp          time.Time
	CurrentMetrics     CacheMetricSnapshot
	AggregatedMetrics  AggregatedCacheMetrics
	TrendAnalysis      CacheTrendAnalysis
	Recommendations    []string
}

func calculateHitRate(hits, misses int64) float64 {
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

func calculateErrorRate(errors, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(errors) / float64(total)
}

func (m *CacheMetricsMonitor) SetAlertThresholds(thresholds *CacheAlertThresholds) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alertThresholds = thresholds
}

func (m *CacheMetricsMonitor) Enable(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
}

func (m *CacheMetricsMonitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.totalHits.Store(0)
	m.totalMisses.Store(0)
	m.l1Hits.Store(0)
	m.l1Misses.Store(0)
	m.l2Hits.Store(0)
	m.l2Misses.Store(0)
	m.totalSets.Store(0)
	m.totalDeletes.Store(0)
	m.totalErrors.Store(0)
	m.totalEvictions.Store(0)
	m.totalBytesRead.Store(0)
	m.totalBytesWritten.Store(0)
	m.avgGetLatency.Store(0)
	m.avgSetLatency.Store(0)
	m.metricsHistory = make([]*CacheMetricSnapshot, 0)
	m.lastSnapshot = nil
}

var globalCacheMetricsMonitor *CacheMetricsMonitor
var globalCacheMonitorOnce sync.Once

func InitCacheMetricsMonitor(interval time.Duration) {
	globalCacheMonitorOnce.Do(func() {
		globalCacheMetricsMonitor = NewCacheMetricsMonitor(interval)
	})
}

func GetCacheMetricsMonitor() *CacheMetricsMonitor {
	if globalCacheMetricsMonitor == nil {
		InitCacheMetricsMonitor(10 * time.Second)
	}
	return globalCacheMetricsMonitor
}

type CacheHealthChecker struct {
	monitor *CacheMetricsMonitor
}

func NewCacheHealthChecker(monitor *CacheMetricsMonitor) *CacheHealthChecker {
	return &CacheHealthChecker{monitor: monitor}
}

func (h *CacheHealthChecker) CheckHealth(ctx context.Context) *CacheHealthReport {
	status := &CacheHealthReport{
		Timestamp: time.Now(),
		Healthy:   true,
	}
	
	snapshot := h.monitor.GetCurrentMetrics()
	
	if snapshot.HitRate < 50 {
		status.Healthy = false
		status.Issues = append(status.Issues, "缓存命中率过低")
	}
	
	if snapshot.TotalErrors > 0 {
		errorRate := float64(snapshot.TotalErrors) / float64(snapshot.TotalSets+snapshot.TotalDeletes+1)
		if errorRate > 0.01 {
			status.Healthy = false
			status.Issues = append(status.Issues, fmt.Sprintf("错误率过高: %.4f%%", errorRate*100))
		}
	}
	
	return status
}

type CacheHealthReport struct {
	Timestamp time.Time
	Healthy   bool
	Issues    []string
}
