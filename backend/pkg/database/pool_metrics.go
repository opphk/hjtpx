package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type PoolMetricsCollector struct {
	mu              sync.RWMutex
	metricsHistory  []PoolMetricsSnapshot
	maxHistoryLen   int
	collectionInterval time.Duration
	running         bool
	ctx             context.Context
	cancel          context.CancelFunc
}

type PoolMetricsSnapshot struct {
	Timestamp           time.Time
	OpenConnections     int
	InUse               int
	Idle                int
	WaitCount           int64
	WaitDuration        time.Duration
	MaxIdleClosed       int64
	MaxLifetimeClosed   int64
	MaxOpenConnections  int
	AverageWaitTime     time.Duration
	ConnectionReuseRate float64
	HealthScore         float64
}

var globalMetricsCollector *PoolMetricsCollector

func InitMetricsCollector(collectionInterval time.Duration) {
	if collectionInterval <= 0 {
		collectionInterval = 10 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	globalMetricsCollector = &PoolMetricsCollector{
		metricsHistory:   make([]PoolMetricsSnapshot, 0),
		maxHistoryLen:     10000,
		collectionInterval: collectionInterval,
		ctx:               ctx,
		cancel:            cancel,
	}
}

func StartMetricsCollection() {
	if globalMetricsCollector == nil {
		InitMetricsCollector(0)
	}

	if globalMetricsCollector.running {
		return
	}

	globalMetricsCollector.running = true
	go globalMetricsCollector.collectLoop()
}

func (p *PoolMetricsCollector) collectLoop() {
	ticker := time.NewTicker(p.collectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.collectMetrics()
		}
	}
}

func (p *PoolMetricsCollector) collectMetrics() {
	if DB == nil {
		return
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return
	}

	stats := sqlDB.Stats()

	snapshot := PoolMetricsSnapshot{
		Timestamp:          time.Now(),
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
		MaxOpenConnections: stats.MaxOpenConnections,
	}

	if stats.WaitCount > 0 {
		snapshot.AverageWaitTime = stats.WaitDuration / time.Duration(stats.WaitCount)
	}

	totalConnections := stats.InUse + stats.Idle
	if totalConnections > 0 {
		snapshot.ConnectionReuseRate = float64(stats.InUse) / float64(totalConnections) * 100
	}

	snapshot.HealthScore = p.calculateHealthScore(&snapshot)

	p.mu.Lock()
	p.metricsHistory = append(p.metricsHistory, snapshot)

	if len(p.metricsHistory) > p.maxHistoryLen {
		p.metricsHistory = p.metricsHistory[1:]
	}
	p.mu.Unlock()
}

func (p *PoolMetricsCollector) calculateHealthScore(s *PoolMetricsSnapshot) float64 {
	score := 100.0

	if s.OpenConnections >= s.MaxOpenConnections {
		score -= 30
	}

	usagePercent := float64(s.InUse) / float64(s.MaxOpenConnections) * 100
	if usagePercent > 80 {
		score -= 20
	} else if usagePercent > 60 {
		score -= 10
	}

	if s.WaitCount > 100 {
		score -= 15
	} else if s.WaitCount > 50 {
		score -= 10
	}

	if s.MaxIdleClosed > 10 {
		score -= 10
	}

	if s.MaxLifetimeClosed > 10 {
		score -= 5
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (p *PoolMetricsCollector) GetMetricsHistory(limit int) []PoolMetricsSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if limit <= 0 || limit > len(p.metricsHistory) {
		limit = len(p.metricsHistory)
	}

	history := make([]PoolMetricsSnapshot, limit)
	copy(history, p.metricsHistory[len(p.metricsHistory)-limit:])

	return history
}

func (p *PoolMetricsCollector) GetLatestMetrics() *PoolMetricsSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.metricsHistory) == 0 {
		return nil
	}

	snapshot := p.metricsHistory[len(p.metricsHistory)-1]
	return &snapshot
}

func (p *PoolMetricsCollector) GetAggregatedMetrics(window time.Duration) *PoolMetricsAggregate {
	p.mu.RLock()
	defer p.mu.RUnlock()

	cutoff := time.Now().Add(-window)

	var totalMetrics PoolMetricsSnapshot
	var count int

	maxInUse := 0
	maxWaitCount := int64(0)

	for _, m := range p.metricsHistory {
		if m.Timestamp.Before(cutoff) {
			continue
		}

		count++
		totalMetrics.InUse += m.InUse
		totalMetrics.WaitCount += m.WaitCount
		totalMetrics.WaitDuration += m.WaitDuration
		totalMetrics.Idle += m.Idle

		if m.InUse > maxInUse {
			maxInUse = m.InUse
		}
		if m.WaitCount > maxWaitCount {
			maxWaitCount = m.WaitCount
		}
	}

	if count == 0 {
		return &PoolMetricsAggregate{}
	}

	return &PoolMetricsAggregate{
		Window:           window,
		SampleCount:      count,
		AverageInUse:     float64(totalMetrics.InUse) / float64(count),
		AverageIdle:      float64(totalMetrics.Idle) / float64(count),
		AverageWaitTime:  totalMetrics.WaitDuration / time.Duration(count),
		TotalWaitCount:   totalMetrics.WaitCount,
		MaxInUse:         maxInUse,
		MaxWaitCount:     maxWaitCount,
		HealthScore:      p.calculateAggregatedHealthScore(&totalMetrics, count),
	}
}

type PoolMetricsAggregate struct {
	Window          time.Duration
	SampleCount     int
	AverageInUse    float64
	AverageIdle     float64
	AverageWaitTime time.Duration
	TotalWaitCount  int64
	MaxInUse        int
	MaxWaitCount    int64
	HealthScore     float64
}

func (p *PoolMetricsCollector) calculateAggregatedHealthScore(s *PoolMetricsSnapshot, count int) float64 {
	if count == 0 {
		return 100.0
	}

	avgInUse := float64(s.InUse) / float64(count)
	usagePercent := avgInUse / float64(s.InUse) * 100

	score := 100.0

	if usagePercent > 80 {
		score -= 20
	} else if usagePercent > 60 {
		score -= 10
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (p *PoolMetricsCollector) Stop() {
	if p == nil {
		return
	}

	if p.cancel != nil {
		p.cancel()
	}
	p.running = false
}

func GetPoolMetricsHistory(limit int) []PoolMetricsSnapshot {
	if globalMetricsCollector == nil {
		return nil
	}
	return globalMetricsCollector.GetMetricsHistory(limit)
}

func GetPoolAggregatedMetrics(window time.Duration) *PoolMetricsAggregate {
	if globalMetricsCollector == nil {
		return nil
	}
	return globalMetricsCollector.GetAggregatedMetrics(window)
}

type ConnectionPoolRealtimeMonitor struct {
	mu             sync.RWMutex
	subscribers    map[chan PoolMetricsSnapshot]bool
	updateInterval time.Duration
}

var globalRealtimeMonitor *ConnectionPoolRealtimeMonitor

func InitRealtimeMonitor(updateInterval time.Duration) {
	if updateInterval <= 0 {
		updateInterval = 1 * time.Second
	}

	globalRealtimeMonitor = &ConnectionPoolRealtimeMonitor{
		subscribers:    make(map[chan PoolMetricsSnapshot]bool),
		updateInterval: updateInterval,
	}

	go globalRealtimeMonitor.broadcastLoop()
}

func (m *ConnectionPoolRealtimeMonitor) Subscribe() chan PoolMetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan PoolMetricsSnapshot, 100)
	m.subscribers[ch] = true
	return ch
}

func (m *ConnectionPoolRealtimeMonitor) Unsubscribe(ch chan PoolMetricsSnapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.subscribers, ch)
	close(ch)
}

func (m *ConnectionPoolRealtimeMonitor) broadcastLoop() {
	ticker := time.NewTicker(m.updateInterval)
	defer ticker.Stop()

	for range ticker.C {
		snapshot := m.getCurrentSnapshot()
		if snapshot == nil {
			continue
		}

		m.mu.RLock()
		for ch := range m.subscribers {
			select {
			case ch <- *snapshot:
			default:
			}
		}
		m.mu.RUnlock()
	}
}

func (m *ConnectionPoolRealtimeMonitor) getCurrentSnapshot() *PoolMetricsSnapshot {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return nil
	}

	stats := sqlDB.Stats()

	snapshot := &PoolMetricsSnapshot{
		Timestamp:          time.Now(),
		OpenConnections:    stats.OpenConnections,
		InUse:               stats.InUse,
		Idle:                stats.Idle,
		WaitCount:           stats.WaitCount,
		WaitDuration:        stats.WaitDuration,
		MaxIdleClosed:       stats.MaxIdleClosed,
		MaxLifetimeClosed:   stats.MaxLifetimeClosed,
		MaxOpenConnections: stats.MaxOpenConnections,
	}

	totalConnections := stats.InUse + stats.Idle
	if totalConnections > 0 {
		snapshot.ConnectionReuseRate = float64(stats.InUse) / float64(totalConnections) * 100
	}

	if globalMetricsCollector != nil {
		snapshot.HealthScore = globalMetricsCollector.calculateHealthScore(snapshot)
	}

	return snapshot
}

type PoolAlertConfig struct {
	HighConnectionUsageThreshold float64
	HighWaitCountThreshold       int64
	HighWaitTimeThreshold       time.Duration
	LowHealthScoreThreshold     float64
}

var defaultAlertConfig = &PoolAlertConfig{
	HighConnectionUsageThreshold: 80,
	HighWaitCountThreshold:       100,
	HighWaitTimeThreshold:        5 * time.Second,
	LowHealthScoreThreshold:      50,
}

type PoolAlert struct {
	Timestamp   time.Time
	AlertType   string
	Message     string
	Severity    string
	Metrics     *PoolMetricsSnapshot
}

func (p *PoolMetricsCollector) CheckAlerts(config *PoolAlertConfig) []PoolAlert {
	if config == nil {
		config = defaultAlertConfig
	}

	snapshot := p.GetLatestMetrics()
	if snapshot == nil {
		return nil
	}

	var alerts []PoolAlert

	usagePercent := float64(snapshot.InUse) / float64(snapshot.MaxOpenConnections) * 100
	if usagePercent >= config.HighConnectionUsageThreshold {
		alerts = append(alerts, PoolAlert{
			Timestamp: time.Now(),
			AlertType: "high_connection_usage",
			Message:   fmt.Sprintf("连接使用率 %.1f%% 超过阈值 %.1f%%", usagePercent, config.HighConnectionUsageThreshold),
			Severity:  "warning",
			Metrics:   snapshot,
		})
	}

	if snapshot.WaitCount >= config.HighWaitCountThreshold {
		alerts = append(alerts, PoolAlert{
			Timestamp: time.Now(),
			AlertType: "high_wait_count",
			Message:   fmt.Sprintf("等待连接数量 %d 超过阈值 %d", snapshot.WaitCount, config.HighWaitCountThreshold),
			Severity:  "warning",
			Metrics:   snapshot,
		})
	}

	if snapshot.AverageWaitTime >= config.HighWaitTimeThreshold {
		alerts = append(alerts, PoolAlert{
			Timestamp: time.Now(),
			AlertType: "high_wait_time",
			Message:   fmt.Sprintf("平均等待时间 %v 超过阈值 %v", snapshot.AverageWaitTime, config.HighWaitTimeThreshold),
			Severity:  "critical",
			Metrics:   snapshot,
		})
	}

	if snapshot.HealthScore <= config.LowHealthScoreThreshold {
		alerts = append(alerts, PoolAlert{
			Timestamp: time.Now(),
			AlertType: "low_health_score",
			Message:   fmt.Sprintf("连接池健康分数 %.1f 低于阈值 %.1f", snapshot.HealthScore, config.LowHealthScoreThreshold),
			Severity:  "critical",
			Metrics:   snapshot,
		})
	}

	return alerts
}

type PoolOptimizationRecommendation struct {
	Type         string
	CurrentValue interface{}
	Recommended  interface{}
	Reason       string
	Priority     int
}

func (p *PoolMetricsCollector) GetOptimizationRecommendations() []PoolOptimizationRecommendation {
	snapshot := p.GetLatestMetrics()
	if snapshot == nil {
		return nil
	}

	var recommendations []PoolOptimizationRecommendation

	usagePercent := float64(snapshot.InUse) / float64(snapshot.MaxOpenConnections) * 100

	if usagePercent > 80 {
		recommendedMaxOpen := int(float64(snapshot.MaxOpenConnections) * 1.5)
		recommendations = append(recommendations, PoolOptimizationRecommendation{
			Type:         "increase_max_open_conns",
			CurrentValue: snapshot.MaxOpenConnections,
			Recommended:  recommendedMaxOpen,
			Reason:       fmt.Sprintf("当前连接使用率 %.1f%% 过高，建议增加最大连接数", usagePercent),
			Priority:     1,
		})
	}

	if snapshot.Idle < 5 {
		recommendations = append(recommendations, PoolOptimizationRecommendation{
			Type:         "increase_min_idle_conns",
			CurrentValue: snapshot.Idle,
			Recommended:  10,
			Reason:       "空闲连接数过少，可能导致频繁创建新连接",
			Priority:     2,
		})
	}

	if snapshot.MaxLifetimeClosed > 50 {
		recommendedLifetime := 30 * time.Minute
		recommendations = append(recommendations, PoolOptimizationRecommendation{
			Type:         "decrease_conn_max_lifetime",
			CurrentValue: "过短或过长",
			Recommended:  recommendedLifetime,
			Reason:       "连接生命周期关闭过多，建议调整",
			Priority:     3,
		})
	}

	return recommendations
}

func (p *PoolMetricsCollector) GetConnectionPoolHealthReport() *PoolHealthReport {
	snapshot := p.GetLatestMetrics()
	metrics1h := p.GetAggregatedMetrics(1 * time.Hour)
	metrics24h := p.GetAggregatedMetrics(24 * time.Hour)

	report := &PoolHealthReport{
		Timestamp:           time.Now(),
		CurrentMetrics:      snapshot,
		Last1HourMetrics:    metrics1h,
		Last24HourMetrics:   metrics24h,
		Alerts:              p.CheckAlerts(nil),
		Recommendations:     p.GetOptimizationRecommendations(),
	}

	if snapshot != nil {
		report.OverallHealthScore = snapshot.HealthScore
	}

	return report
}

type PoolHealthReport struct {
	Timestamp          time.Time
	CurrentMetrics     *PoolMetricsSnapshot
	Last1HourMetrics   *PoolMetricsAggregate
	Last24HourMetrics  *PoolMetricsAggregate
	OverallHealthScore float64
	Alerts             []PoolAlert
	Recommendations    []PoolOptimizationRecommendation
}

type ConnectionPoolExporter struct {
	collector *PoolMetricsCollector
}

func NewConnectionPoolExporter() *ConnectionPoolExporter {
	return &ConnectionPoolExporter{
		collector: globalMetricsCollector,
	}
}

func (e *ConnectionPoolExporter) ExportPrometheusMetrics() string {
	if e.collector == nil {
		return ""
	}

	snapshot := e.collector.GetLatestMetrics()
	if snapshot == nil {
		return ""
	}

	metrics := "# HELP hjtpx_db_pool_connections_open Current number of open connections\n"
	metrics += "# TYPE hjtpx_db_pool_connections_open gauge\n"
	metrics += fmt.Sprintf("hjtpx_db_pool_connections_open %d\n", snapshot.OpenConnections)

	metrics += "# HELP hjtpx_db_pool_connections_in_use Current number of connections in use\n"
	metrics += "# TYPE hjtpx_db_pool_connections_in_use gauge\n"
	metrics += fmt.Sprintf("hjtpx_db_pool_connections_in_use %d\n", snapshot.InUse)

	metrics += "# HELP hjtpx_db_pool_connections_idle Current number of idle connections\n"
	metrics += "# TYPE hjtpx_db_pool_connections_idle gauge\n"
	metrics += fmt.Sprintf("hjtpx_db_pool_connections_idle %d\n", snapshot.Idle)

	metrics += "# HELP hjtpx_db_pool_wait_count_total Total number of times waiting for a connection\n"
	metrics += "# TYPE hjtpx_db_pool_wait_count_total counter\n"
	metrics += fmt.Sprintf("hjtpx_db_pool_wait_count_total %d\n", snapshot.WaitCount)

	metrics += "# HELP hjtpx_db_pool_health_score Connection pool health score\n"
	metrics += "# TYPE hjtpx_db_pool_health_score gauge\n"
	metrics += fmt.Sprintf("hjtpx_db_pool_health_score %.2f\n", snapshot.HealthScore)

	metrics += "# HELP hjtpx_db_pool_connection_reuse_rate Connection reuse rate percentage\n"
	metrics += "# TYPE hjtpx_db_pool_connection_reuse_rate gauge\n"
	metrics += fmt.Sprintf("hjtpx_db_pool_connection_reuse_rate %.2f\n", snapshot.ConnectionReuseRate)

	return metrics
}

func ExportPoolMetricsToPrometheus() string {
	exporter := NewConnectionPoolExporter()
	return exporter.ExportPrometheusMetrics()
}
