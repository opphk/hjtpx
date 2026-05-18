package database

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

type QueryMetric struct {
	Query        string
	Duration     time.Duration
	Timestamp    time.Time
	IsSlow       bool
	Error        error
	ConnectionID uint64
	TableNames   []string
}

type PerformanceMonitor struct {
	mu              sync.RWMutex
	queryMetrics    []QueryMetric
	slowQueries     []QueryMetric
	maxMetricsLen   int
	maxSlowQueryLen int
	enabled         bool
	slowThreshold   time.Duration
	totalDuration   time.Duration
	queryCount      int64
}

type PerformanceStats struct {
	TotalQueries     int64
	SlowQueries      int64
	FailedQueries    int64
	AvgDuration      time.Duration
	MaxDuration      time.Duration
	MinDuration      time.Duration
	TotalDuration    time.Duration
	QueriesPerSecond float64
	SlowQueryRatio   float64
}

var perfMonitor *PerformanceMonitor
var monitorStartTime time.Time

func InitPerformanceMonitor(cfg *config.Config) {
	perfMonitor = &PerformanceMonitor{
		queryMetrics:    make([]QueryMetric, 0),
		slowQueries:     make([]QueryMetric, 0),
		maxMetricsLen:   10000,
		maxSlowQueryLen: 1000,
		enabled:         cfg.Database.Monitoring.EnableQueryMetrics,
		slowThreshold:   time.Duration(cfg.Database.SlowQueryThresholdMs) * time.Millisecond,
	}
	monitorStartTime = time.Now()

	if perfMonitor.enabled {
		log.Println("Performance monitor initialized")
		go perfMonitor.startPeriodicAnalysis()
	}
}

func GetPerformanceMonitor() *PerformanceMonitor {
	return perfMonitor
}

func (pm *PerformanceMonitor) RecordQuery(query string, duration time.Duration, err error) {
	if !pm.enabled {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	isSlow := duration > pm.slowThreshold

	metric := QueryMetric{
		Query:     query,
		Duration:  duration,
		Timestamp: time.Now(),
		IsSlow:    isSlow,
		Error:     err,
	}

	pm.queryMetrics = append(pm.queryMetrics, metric)
	pm.totalDuration += duration
	atomic.AddInt64(&pm.queryCount, 1)

	if len(pm.queryMetrics) > pm.maxMetricsLen {
		pm.queryMetrics = pm.queryMetrics[1:]
	}

	if isSlow {
		log.Printf("SLOW QUERY: %s took %v", query, duration)
		pm.slowQueries = append(pm.slowQueries, metric)

		if len(pm.slowQueries) > pm.maxSlowQueryLen {
			pm.slowQueries = pm.slowQueries[1:]
		}
	}
}

func (pm *PerformanceMonitor) GetStats() *PerformanceStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := &PerformanceStats{}
	if len(pm.queryMetrics) == 0 {
		return stats
	}

	var totalDuration time.Duration
	var minDuration time.Duration = pm.queryMetrics[0].Duration
	var maxDuration time.Duration
	var failedCount int64

	for _, m := range pm.queryMetrics {
		stats.TotalQueries++
		totalDuration += m.Duration

		if m.IsSlow {
			stats.SlowQueries++
		}
		if m.Error != nil {
			failedCount++
		}
		if m.Duration > maxDuration {
			maxDuration = m.Duration
		}
		if m.Duration < minDuration {
			minDuration = m.Duration
		}
	}

	stats.TotalDuration = totalDuration
	stats.AvgDuration = totalDuration / time.Duration(stats.TotalQueries)
	stats.MaxDuration = maxDuration
	stats.MinDuration = minDuration
	stats.FailedQueries = failedCount

	uptime := time.Since(monitorStartTime)
	if uptime > 0 {
		stats.QueriesPerSecond = float64(stats.TotalQueries) / uptime.Seconds()
	}

	if stats.TotalQueries > 0 {
		stats.SlowQueryRatio = float64(stats.SlowQueries) / float64(stats.TotalQueries) * 100
	}

	return stats
}

func (pm *PerformanceMonitor) GetSlowQueries(limit int) []QueryMetric {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if limit <= 0 || limit > len(pm.slowQueries) {
		limit = len(pm.slowQueries)
	}

	result := make([]QueryMetric, limit)
	copy(result, pm.slowQueries[len(pm.slowQueries)-limit:])
	return result
}

func (pm *PerformanceMonitor) Clear() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.queryMetrics = make([]QueryMetric, 0)
	pm.slowQueries = make([]QueryMetric, 0)
	pm.totalDuration = 0
	atomic.StoreInt64(&pm.queryCount, 0)
	monitorStartTime = time.Now()
}

func (pm *PerformanceMonitor) startPeriodicAnalysis() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		pm.analyzeAndReport()
	}
}

func (pm *PerformanceMonitor) analyzeAndReport() {
	stats := pm.GetStats()

	if stats.TotalQueries == 0 {
		return
	}

	log.Printf("[PERF_ANALYSIS] Total: %d, Slow: %d (%.2f%%), Avg: %v, Max: %v, QPS: %.2f",
		stats.TotalQueries,
		stats.SlowQueries,
		stats.SlowQueryRatio,
		stats.AvgDuration,
		stats.MaxDuration,
		stats.QueriesPerSecond,
	)

	if stats.SlowQueryRatio > 10 {
		log.Printf("[PERF_WARNING] High slow query ratio: %.2f%%", stats.SlowQueryRatio)
	}

	if stats.AvgDuration > pm.slowThreshold {
		log.Printf("[PERF_WARNING] Average query time exceeds threshold: %v > %v",
			stats.AvgDuration, pm.slowThreshold)
	}
}

func GormQueryCallback(db *gorm.DB) {
	startTime := time.Now()

	db.InstanceSet("query_start_time", startTime)

	db.Callback().Query().After("gorm:query").Register("performance_monitor", func(d *gorm.DB) {
		if perfMonitor == nil {
			return
		}

		var duration time.Duration
		if startTime, ok := d.InstanceGet("query_start_time"); ok {
			duration = time.Since(startTime.(time.Time))
		}

		sql := d.Dialector.Explain(d.Statement.SQL.String(), d.Statement.Vars...)
		perfMonitor.RecordQuery(sql, duration, d.Error)
	})

	db.Callback().Create().After("gorm:create").Register("performance_monitor_write", func(d *gorm.DB) {
		perfMonitor.recordWriteOperation(d, "CREATE")
	})

	db.Callback().Update().After("gorm:update").Register("performance_monitor_write", func(d *gorm.DB) {
		perfMonitor.recordWriteOperation(d, "UPDATE")
	})

	db.Callback().Delete().After("gorm:delete").Register("performance_monitor_write", func(d *gorm.DB) {
		perfMonitor.recordWriteOperation(d, "DELETE")
	})
}

func (pm *PerformanceMonitor) recordWriteOperation(db *gorm.DB, opType string) {
	if !pm.enabled {
		return
	}

	var duration time.Duration
	if startTime, ok := db.InstanceGet("query_start_time"); ok {
		duration = time.Since(startTime.(time.Time))
	}

	sql := db.Dialector.Explain(db.Statement.SQL.String(), db.Statement.Vars...)
	pm.RecordQuery(opType+": "+sql, duration, db.Error)
}

func ExplainQuery(query string, args ...interface{}) (string, error) {
	var result string
	explainSQL := "EXPLAIN ANALYZE " + query
	err := DB.Raw(explainSQL, args...).Scan(&result).Error
	return result, err
}

func AnalyzeTable(tableName string) error {
	return DB.Exec("ANALYZE " + tableName).Error
}

type PerformanceReport struct {
	GeneratedAt     time.Time
	Uptime          time.Duration
	DatabaseStats   *PerformanceStats
	ConnectionStats *ConnectionPoolMetrics
	CacheStats      map[string]interface{}
	TopSlowQueries  []QueryMetric
	Recommendations []string
}

func (pm *PerformanceMonitor) GenerateReport() *PerformanceReport {
	report := &PerformanceReport{
		GeneratedAt:     time.Now(),
		Uptime:          time.Since(monitorStartTime),
		DatabaseStats:   pm.GetStats(),
		TopSlowQueries:  pm.GetSlowQueries(10),
		Recommendations: pm.generateRecommendations(),
	}

	if connStats, err := GetConnectionPoolMetrics(); err == nil {
		report.ConnectionStats = connStats
	}

	return report
}

func (pm *PerformanceMonitor) generateRecommendations() []string {
	var recommendations []string

	stats := pm.GetStats()

	if stats.SlowQueryRatio > 5 {
		recommendations = append(recommendations, "Consider adding indexes for slow queries")
	}

	if stats.AvgDuration > 50*time.Millisecond {
		recommendations = append(recommendations, "Average query duration is high, review query optimization")
	}

	if stats.MaxDuration > 500*time.Millisecond {
		recommendations = append(recommendations, "Some queries are very slow, consider query rewrite or optimization")
	}

	if connStats, err := GetConnectionPoolMetrics(); err == nil {
		if connStats.ReuseRate < 80 {
			recommendations = append(recommendations, "Connection reuse rate is low, consider tuning connection pool")
		}

		if connStats.WaitCount > 1000 {
			recommendations = append(recommendations, "High connection wait count, consider increasing pool size")
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System performance is within acceptable ranges")
	}

	return recommendations
}

func (pm *PerformanceMonitor) ExportJSON() ([]byte, error) {
	report := pm.GenerateReport()
	return json.MarshalIndent(report, "", "  ")
}

func (pm *PerformanceMonitor) GetTopQueries(limit int) []QueryMetric {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if limit <= 0 || limit > len(pm.queryMetrics) {
		limit = len(pm.queryMetrics)
	}

	queries := make([]QueryMetric, limit)
	copy(queries, pm.queryMetrics[len(pm.queryMetrics)-limit:])
	return queries
}

func (pm *PerformanceMonitor) GetQueryDistribution() map[string]int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	distribution := make(map[string]int)

	for _, m := range pm.queryMetrics {
		durationBucket := pm.getDurationBucket(m.Duration)
		distribution[durationBucket]++
	}

	return distribution
}

func (pm *PerformanceMonitor) getDurationBucket(duration time.Duration) string {
	switch {
	case duration < 5*time.Millisecond:
		return "<5ms"
	case duration < 10*time.Millisecond:
		return "5-10ms"
	case duration < 50*time.Millisecond:
		return "10-50ms"
	case duration < 100*time.Millisecond:
		return "50-100ms"
	case duration < 500*time.Millisecond:
		return "100-500ms"
	default:
		return ">500ms"
	}
}

type AlertConfig struct {
	SlowQueryThreshold      time.Duration
	AvgDurationThreshold    time.Duration
	SlowQueryRatioThreshold float64
}

var defaultAlertConfig = &AlertConfig{
	SlowQueryThreshold:      50 * time.Millisecond,
	AvgDurationThreshold:    30 * time.Millisecond,
	SlowQueryRatioThreshold: 5.0,
}

func (pm *PerformanceMonitor) CheckAlerts(config *AlertConfig) []string {
	if config == nil {
		config = defaultAlertConfig
	}

	var alerts []string
	stats := pm.GetStats()

	if stats.SlowQueries > 100 {
		alerts = append(alerts, fmt.Sprintf("High slow query count: %d", stats.SlowQueries))
	}

	if stats.AvgDuration > config.AvgDurationThreshold {
		alerts = append(alerts, fmt.Sprintf("Average duration exceeds threshold: %v > %v",
			stats.AvgDuration, config.AvgDurationThreshold))
	}

	if stats.SlowQueryRatio > config.SlowQueryRatioThreshold {
		alerts = append(alerts, fmt.Sprintf("Slow query ratio exceeds threshold: %.2f%% > %.2f%%",
			stats.SlowQueryRatio, config.SlowQueryRatioThreshold))
	}

	return alerts
}

type MetricsAggregator struct {
	mu         sync.RWMutex
	interval   time.Duration
	windows    []MetricsWindow
	maxWindows int
}

type MetricsWindow struct {
	StartTime time.Time
	EndTime   time.Time
	Stats     *PerformanceStats
}

func NewMetricsAggregator(interval time.Duration, maxWindows int) *MetricsAggregator {
	if maxWindows <= 0 {
		maxWindows = 60
	}

	return &MetricsAggregator{
		interval:   interval,
		windows:    make([]MetricsWindow, 0),
		maxWindows: maxWindows,
	}
}

func (ma *MetricsAggregator) RecordWindow(stats *PerformanceStats) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	window := MetricsWindow{
		StartTime: time.Now().Add(-ma.interval),
		EndTime:   time.Now(),
		Stats:     stats,
	}

	ma.windows = append(ma.windows, window)

	if len(ma.windows) > ma.maxWindows {
		ma.windows = ma.windows[1:]
	}
}

func (ma *MetricsAggregator) GetWindows() []MetricsWindow {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	result := make([]MetricsWindow, len(ma.windows))
	copy(result, ma.windows)
	return result
}

func (ma *MetricsAggregator) GetTrend() string {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	if len(ma.windows) < 2 {
		return "insufficient_data"
	}

	var totalChange float64
	count := 0

	for i := 1; i < len(ma.windows); i++ {
		prev := ma.windows[i-1].Stats.AvgDuration
		curr := ma.windows[i].Stats.AvgDuration

		if prev > 0 {
			change := float64(curr-prev) / float64(prev) * 100
			totalChange += change
			count++
		}
	}

	if count == 0 {
		return "stable"
	}

	avgChange := totalChange / float64(count)

	if avgChange > 10 {
		return "degrading"
	} else if avgChange < -10 {
		return "improving"
	}
	return "stable"
}

type SlowQueryAnalyzer struct {
	mu      sync.RWMutex
	queries map[string]*SlowQueryInfo
}

type SlowQueryInfo struct {
	Query          string
	Count          int64
	TotalDuration  time.Duration
	AvgDuration    time.Duration
	MaxDuration    time.Duration
	LastOccurrence time.Time
	Suggestions    []string
}

func NewSlowQueryAnalyzer() *SlowQueryAnalyzer {
	return &SlowQueryAnalyzer{
		queries: make(map[string]*SlowQueryInfo),
	}
}

func (a *SlowQueryAnalyzer) Record(query string, duration time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()

	info, exists := a.queries[query]
	if !exists {
		info = &SlowQueryInfo{
			Query:       query,
			Suggestions: a.generateSuggestions(query),
		}
		a.queries[query] = info
	}

	info.Count++
	info.TotalDuration += duration
	info.AvgDuration = info.TotalDuration / time.Duration(info.Count)
	if duration > info.MaxDuration {
		info.MaxDuration = duration
	}
	info.LastOccurrence = time.Now()
}

func (a *SlowQueryAnalyzer) generateSuggestions(query string) []string {
	var suggestions []string

	if len(query) > 200 {
		suggestions = append(suggestions, "Query is complex, consider simplifying")
	}

	if !containsIndexHint(query) {
		suggestions = append(suggestions, "Consider adding index hints")
	}

	if !containsLimit(query) {
		suggestions = append(suggestions, "Query may return many rows, consider adding LIMIT")
	}

	return suggestions
}

func containsIndexHint(query string) bool {
	return len(query) > 0
}

func containsLimit(query string) bool {
	return len(query) > 0
}

func (a *SlowQueryAnalyzer) GetTopQueries(limit int) []*SlowQueryInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	infos := make([]*SlowQueryInfo, 0, len(a.queries))
	for _, info := range a.queries {
		infos = append(infos, info)
	}

	if len(infos) <= limit {
		return infos
	}

	for i := 0; i < len(infos)-1; i++ {
		for j := i + 1; j < len(infos); j++ {
			if infos[j].AvgDuration > infos[i].AvgDuration {
				infos[i], infos[j] = infos[j], infos[i]
			}
		}
	}

	return infos[:limit]
}

func (a *SlowQueryAnalyzer) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.queries = make(map[string]*SlowQueryInfo)
}

var globalSlowQueryAnalyzer *SlowQueryAnalyzer

func init() {
	globalSlowQueryAnalyzer = NewSlowQueryAnalyzer()
}

func GetSlowQueryAnalyzer() *SlowQueryAnalyzer {
	return globalSlowQueryAnalyzer
}
