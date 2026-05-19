package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

type AdvancedConnectionPoolManager struct {
	mu                  sync.RWMutex
	poolConfig          *PoolConfiguration
	performanceTracker  *PoolPerformanceTracker
	autoTuner          *PoolAutoTuner
	healthMonitor       *PoolHealthMonitor
	circuitBreaker      *PoolCircuitBreaker
	metricsCollector    *PoolMetricsCollector
}

type PoolConfiguration struct {
	MaxOpenConns        int
	MaxIdleConns        int
	MinIdleConns         int
	ConnMaxLifetime      time.Duration
	ConnMaxIdleTime      time.Duration
	ConnMaxProbeTime     time.Duration
	ConnKeepAlive        time.Duration
	WaitPoolTimeout      time.Duration
	PoolWriteTimeout     time.Duration
	PoolReadTimeout      time.Duration
	EnableStats          bool
	EnableAutoTune       bool
	AutoTuneInterval     time.Duration
	HealthCheckInterval  time.Duration
}

type PoolPerformanceTracker struct {
	mu              sync.RWMutex
	queriesExecuted  int64
	queriesSlow      int64
	queriesFailed    int64
	totalDuration    time.Duration
	maxDuration      time.Duration
	minDuration      time.Duration
	lastQueryTime    time.Time
	queryHistory     []QueryExecutionRecord
	maxHistoryLen    int
}

type QueryExecutionRecord struct {
	Timestamp      time.Time
	Duration       time.Duration
	QueryType      string
	Success        bool
	ErrorMessage   string
	ConnectionID   int64
}

type PoolAutoTuner struct {
	mu                  sync.RWMutex
	enabled             bool
	interval            time.Duration
	tuningHistory       []TuningDecision
	maxHistoryLen       int
	lastDecisionTime    time.Time
}

type TuningDecision struct {
	Timestamp    time.Time
	Action       string
	OldConfig    PoolConfiguration
	NewConfig    PoolConfiguration
	Reason       string
	Success      bool
}

type PoolHealthMonitor struct {
	mu              sync.RWMutex
	checks          []HealthCheck
	maxChecks       int
	lastCheckTime   time.Time
	lastCheckResult bool
	alertThresholds *HealthThresholds
}

type HealthCheck struct {
	Timestamp      time.Time
	IsHealthy      bool
	ResponseTime    time.Duration
	ActiveConns    int
	IdleConns      int
	WaitCount      int64
	Errors         []string
}

type HealthThresholds struct {
	MaxWaitTime       time.Duration
	MaxActiveRatio    float64
	MaxErrorRate      float64
	MinHealthyRatio   float64
}

type PoolCircuitBreaker struct {
	mu                  sync.RWMutex
	state               CircuitState
	failureCount        int64
	successCount        int64
	lastFailureTime     time.Time
	threshold           int64
	timeout             time.Duration
	recoveryTimeout     time.Duration
}

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

type PoolMetricsCollector struct {
	mu              sync.RWMutex
	metrics         []PoolMetricSnapshot
	maxMetrics      int
	collectionInterval time.Duration
}

type PoolMetricSnapshot struct {
	Timestamp        time.Time
	ActiveConnections int
	IdleConnections   int
	TotalConnections  int
	WaitCount         int64
	WaitDuration      time.Duration
	MaxLifetimeClosed int64
	MaxIdleClosed     int64
	QueriesPerSecond  float64
}

var advancedPoolManager *AdvancedConnectionPoolManager

func InitAdvancedConnectionPool(cfg *config.Config) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	poolConfig := &PoolConfiguration{
		MaxOpenConns:       cfg.Database.ConnectionPool.MaxOpenConns,
		MaxIdleConns:       cfg.Database.ConnectionPool.MaxIdleConns,
		MinIdleConns:       5,
		ConnMaxLifetime:    time.Duration(cfg.Database.ConnectionPool.ConnMaxLifetimeSecs) * time.Second,
		ConnMaxIdleTime:    time.Duration(cfg.Database.ConnectionPool.ConnMaxIdleTimeSecs) * time.Second,
		EnableStats:        cfg.Database.Monitoring.EnableConnectionMetrics,
		EnableAutoTune:     cfg.Database.ConnectionPool.EnableAutoTuning,
		AutoTuneInterval:   30 * time.Second,
		HealthCheckInterval: 1 * time.Minute,
	}

	advancedPoolManager = &AdvancedConnectionPoolManager{
		poolConfig:         poolConfig,
		performanceTracker: newPerformanceTracker(),
		autoTuner:          newPoolAutoTuner(poolConfig.EnableAutoTune),
		healthMonitor:       newPoolHealthMonitor(),
		circuitBreaker:      newCircuitBreaker(),
		metricsCollector:    newPoolMetricsCollector(),
	}

	advancedPoolManager.applyConfiguration(sqlDB)

	if poolConfig.EnableStats {
		go advancedPoolManager.startMetricsCollection()
		go advancedPoolManager.startHealthMonitoring()
	}

	if poolConfig.EnableAutoTune {
		go advancedPoolManager.startAutoTuning()
	}

	log.Println("Advanced connection pool manager initialized")
	return nil
}

func GetAdvancedConnectionPoolManager() *AdvancedConnectionPoolManager {
	return advancedPoolManager
}

func newPerformanceTracker() *PoolPerformanceTracker {
	return &PoolPerformanceTracker{
		queryHistory:  make([]QueryExecutionRecord, 0),
		maxHistoryLen: 1000,
		minDuration:   24 * time.Hour,
	}
}

func newPoolAutoTuner(enabled bool) *PoolAutoTuner {
	return &PoolAutoTuner{
		enabled:    enabled,
		interval:   30 * time.Second,
		tuningHistory: make([]TuningDecision, 0),
		maxHistoryLen: 100,
	}
}

func newPoolHealthMonitor() *PoolHealthMonitor {
	return &PoolHealthMonitor{
		checks:    make([]HealthCheck, 0),
		maxChecks: 100,
		alertThresholds: &HealthThresholds{
			MaxWaitTime:     5 * time.Second,
			MaxActiveRatio:  0.9,
			MaxErrorRate:    0.1,
			MinHealthyRatio: 0.8,
		},
	}
}

func newCircuitBreaker() *PoolCircuitBreaker {
	return &PoolCircuitBreaker{
		state:           StateClosed,
		threshold:       10,
		timeout:         30 * time.Second,
		recoveryTimeout: 60 * time.Second,
	}
}

func newPoolMetricsCollector() *PoolMetricsCollector {
	return &PoolMetricsCollector{
		metrics:           make([]PoolMetricSnapshot, 0),
		maxMetrics:        1000,
		collectionInterval: 15 * time.Second,
	}
}

func (m *AdvancedConnectionPoolManager) applyConfiguration(db *sql.DB) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg := m.poolConfig

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	log.Printf("[POOL] Configuration applied: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v",
		cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetime)
}

func (m *AdvancedConnectionPoolManager) RecordQuery(duration time.Duration, queryType string, success bool, err error) {
	m.performanceTracker.mu.Lock()
	defer m.performanceTracker.mu.Unlock()

	atomic.AddInt64(&m.performanceTracker.queriesExecuted, 1)

	record := QueryExecutionRecord{
		Timestamp:   time.Now(),
		Duration:    duration,
		QueryType:   queryType,
		Success:     success,
	}

	if err != nil {
		record.ErrorMessage = err.Error()
		atomic.AddInt64(&m.performanceTracker.queriesFailed, 1)
		m.circuitBreaker.RecordFailure()
	} else {
		m.circuitBreaker.RecordSuccess()
	}

	if duration > 100*time.Millisecond {
		atomic.AddInt64(&m.performanceTracker.queriesSlow, 1)
	}

	m.performanceTracker.totalDuration += duration
	if duration > m.performanceTracker.maxDuration {
		m.performanceTracker.maxDuration = duration
	}
	if duration < m.performanceTracker.minDuration {
		m.performanceTracker.minDuration = duration
	}

	m.performanceTracker.queryHistory = append(m.performanceTracker.queryHistory, record)
	if len(m.performanceTracker.queryHistory) > m.performanceTracker.maxHistoryLen {
		m.performanceTracker.queryHistory = m.performanceTracker.queryHistory[1:]
	}

	m.performanceTracker.lastQueryTime = time.Now()
}

func (m *PoolCircuitBreaker) RecordFailure() {
	atomic.AddInt64(&m.failureCount, 1)
	atomic.StoreInt64(&m.successCount, 0)
	m.lastFailureTime = time.Now()

	if atomic.LoadInt64(&m.failureCount) >= m.threshold {
		m.mu.Lock()
		m.state = StateOpen
		m.mu.Unlock()
		log.Println("[CIRCUIT_BREAKER] Connection pool circuit OPEN")
	}
}

func (m *PoolCircuitBreaker) RecordSuccess() {
	atomic.AddInt64(&m.successCount, 1)
	if atomic.LoadInt64(&m.successCount) >= 5 {
		m.mu.Lock()
		m.state = StateClosed
		atomic.StoreInt64(&m.failureCount, 0)
		m.mu.Unlock()
		log.Println("[CIRCUIT_BREAKER] Connection pool circuit CLOSED")
	}
}

func (m *PoolCircuitBreaker) AllowRequest() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch m.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(m.lastFailureTime) > m.timeout {
			m.mu.RUnlock()
			m.mu.Lock()
			m.state = StateHalfOpen
			m.mu.Unlock()
			m.mu.RLock()
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

func (m *AdvancedConnectionPoolManager) GetPoolStats() map[string]interface{} {
	sqlDB, err := DB.DB()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	stats := sqlDB.Stats()

	return map[string]interface{}{
		"open_connections":   stats.OpenConnections,
		"in_use":             stats.InUse,
		"idle":               stats.Idle,
		"wait_count":         stats.WaitCount,
		"wait_duration":      stats.WaitDuration,
		"max_open_conns":     stats.MaxOpenConnections,
		"max_idle_closed":    stats.MaxIdleClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
		"circuit_state":      m.circuitBreaker.getStateString(),
	}
}

func (m *PoolCircuitBreaker) getStateString() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch m.state {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

func (m *AdvancedConnectionPoolManager) startMetricsCollection() {
	ticker := time.NewTicker(m.metricsCollector.collectionInterval)
	defer ticker.Stop()

	for range ticker.C {
		m.collectMetrics()
	}
}

func (m *AdvancedConnectionPoolManager) collectMetrics() {
	sqlDB, err := DB.DB()
	if err != nil {
		return
	}

	stats := sqlDB.Stats()

	m.performanceTracker.mu.RLock()
	queries := atomic.LoadInt64(&m.performanceTracker.queriesExecuted)
	m.performanceTracker.mu.RUnlock()

	snapshot := PoolMetricSnapshot{
		Timestamp:         time.Now(),
		ActiveConnections: stats.InUse,
		IdleConnections:   stats.Idle,
		TotalConnections:  stats.OpenConnections,
		WaitCount:         stats.WaitCount,
		WaitDuration:      stats.WaitDuration,
		MaxLifetimeClosed: stats.MaxLifetimeClosed,
		MaxIdleClosed:     stats.MaxIdleClosed,
	}

	m.metricsCollector.mu.Lock()
	m.metricsCollector.metrics = append(m.metricsCollector.metrics, snapshot)
	if len(m.metricsCollector.metrics) > m.metricsCollector.maxMetrics {
		m.metricsCollector.metrics = m.metricsCollector.metrics[1:]
	}
	m.metricsCollector.mu.Unlock()

	log.Printf("[POOL_METRICS] Active=%d, Idle=%d, Wait=%d, TotalWait=%v",
		snapshot.ActiveConnections, snapshot.IdleConnections, snapshot.WaitCount, snapshot.WaitDuration)
}

func (m *AdvancedConnectionPoolManager) startHealthMonitoring() {
	ticker := time.NewTicker(m.poolConfig.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		m.performHealthCheck()
	}
}

func (m *AdvancedConnectionPoolManager) performHealthCheck() {
	check := HealthCheck{
		Timestamp: time.Now(),
	}

	sqlDB, err := DB.DB()
	if err != nil {
		check.IsHealthy = false
		check.Errors = append(check.Errors, err.Error())
		m.recordHealthCheck(check)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	if err := sqlDB.PingContext(ctx); err != nil {
		check.IsHealthy = false
		check.Errors = append(check.Errors, err.Error())
	}

	check.ResponseTime = time.Since(start)
	check.IsHealthy = check.IsHealthy && err == nil

	stats := sqlDB.Stats()
	check.ActiveConns = stats.InUse
	check.IdleConns = stats.Idle
	check.WaitCount = stats.WaitCount

	if float64(stats.InUse)/float64(stats.MaxOpenConnections) > m.healthMonitor.alertThresholds.MaxActiveRatio {
		check.Errors = append(check.Errors, fmt.Sprintf("High connection usage: %d/%d", stats.InUse, stats.MaxOpenConnections))
	}

	m.recordHealthCheck(check)

	if len(check.Errors) > 0 {
		log.Printf("[HEALTH_CHECK] Unhealthy: %v", check.Errors)
	}
}

func (m *AdvancedConnectionPoolManager) recordHealthCheck(check HealthCheck) {
	m.healthMonitor.mu.Lock()
	defer m.healthMonitor.mu.Unlock()

	m.healthMonitor.lastCheckTime = check.Timestamp
	m.healthMonitor.lastCheckResult = check.IsHealthy

	m.healthMonitor.checks = append(m.healthMonitor.checks, check)
	if len(m.healthMonitor.checks) > m.healthMonitor.maxChecks {
		m.healthMonitor.checks = m.healthMonitor.checks[1:]
	}
}

func (m *AdvancedConnectionPoolManager) startAutoTuning() {
	ticker := time.NewTicker(m.autoTuner.interval)
	defer ticker.Stop()

	for range ticker.C {
		m.autoTunePool()
	}
}

func (m *AdvancedConnectionPoolManager) autoTunePool() {
	m.autoTuner.mu.Lock()
	if !m.autoTuner.enabled {
		m.autoTuner.mu.Unlock()
		return
	}
	m.autoTuner.mu.Unlock()

	sqlDB, err := DB.DB()
	if err != nil {
		return
	}

	stats := sqlDB.Stats()

	usageRatio := float64(stats.InUse) / float64(stats.MaxOpenConnections)
	waitRatio := float64(stats.WaitCount) / 1000.0

	if usageRatio > 0.85 || waitRatio > 0.5 {
		m.increasePoolSize(stats, usageRatio)
	} else if usageRatio < 0.3 && stats.Idle > m.poolConfig.MinIdleConns+5 {
		m.decreasePoolSize(stats)
	}
}

func (m *AdvancedConnectionPoolManager) increasePoolSize(stats sql.DBStats, ratio float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	newMaxOpen := m.poolConfig.MaxOpenConns
	if ratio > 0.9 {
		newMaxOpen = int(float64(m.poolConfig.MaxOpenConns) * 1.3)
	} else if ratio > 0.85 {
		newMaxOpen = int(float64(m.poolConfig.MaxOpenConns) * 1.2)
	}

	if newMaxOpen > m.poolConfig.MaxOpenConns*2 {
		newMaxOpen = m.poolConfig.MaxOpenConns * 2
	}

	newMaxIdle := newMaxOpen / 2

	m.recordTuningDecision("INCREASE", newMaxOpen, newMaxIdle, "High usage ratio detected")

	sqlDB, _ := DB.DB()
	sqlDB.SetMaxOpenConns(newMaxOpen)
	sqlDB.SetMaxIdleConns(newMaxIdle)

	m.poolConfig.MaxOpenConns = newMaxOpen
	m.poolConfig.MaxIdleConns = newMaxIdle

	log.Printf("[AUTO_TUNE] Increased pool size: MaxOpen=%d, MaxIdle=%d", newMaxOpen, newMaxIdle)
}

func (m *AdvancedConnectionPoolManager) decreasePoolSize(stats sql.DBStats) {
	m.mu.Lock()
	defer m.mu.Unlock()

	newMaxIdle := stats.Idle - 2
	if newMaxIdle < m.poolConfig.MinIdleConns {
		newMaxIdle = m.poolConfig.MinIdleConns
	}

	if newMaxIdle < m.poolConfig.MaxIdleConns {
		m.recordTuningDecision("DECREASE", m.poolConfig.MaxOpenConns, newMaxIdle, "Low usage ratio detected")

		sqlDB, _ := DB.DB()
		sqlDB.SetMaxIdleConns(newMaxIdle)

		m.poolConfig.MaxIdleConns = newMaxIdle

		log.Printf("[AUTO_TUNE] Decreased idle connections: MaxIdle=%d", newMaxIdle)
	}
}

func (m *AdvancedConnectionPoolManager) recordTuningDecision(action string, maxOpen, maxIdle int, reason string) {
	m.autoTuner.mu.Lock()
	defer m.autoTuner.mu.Unlock()

	decision := TuningDecision{
		Timestamp: time.Now(),
		Action:    action,
		OldConfig: *m.poolConfig,
		NewConfig: PoolConfiguration{
			MaxOpenConns: maxOpen,
			MaxIdleConns: maxIdle,
		},
		Reason: reason,
		Success: true,
	}

	m.autoTuner.tuningHistory = append(m.autoTuner.tuningHistory, decision)
	if len(m.autoTuner.tuningHistory) > m.autoTuner.maxHistoryLen {
		m.autoTuner.tuningHistory = m.autoTuner.tuningHistory[1:]
	}

	m.autoTuner.lastDecisionTime = time.Now()
}

func (m *AdvancedConnectionPoolManager) GetPerformanceReport() map[string]interface{} {
	report := make(map[string]interface{})

	m.performanceTracker.mu.RLock()
	defer m.performanceTracker.mu.RUnlock()

	report["queries_executed"] = atomic.LoadInt64(&m.performanceTracker.queriesExecuted)
	report["queries_slow"] = atomic.LoadInt64(&m.performanceTracker.queriesSlow)
	report["queries_failed"] = atomic.LoadInt64(&m.performanceTracker.queriesFailed)
	report["total_duration"] = m.performanceTracker.totalDuration
	report["max_duration"] = m.performanceTracker.maxDuration
	report["min_duration"] = m.performanceTracker.minDuration

	total := atomic.LoadInt64(&m.performanceTracker.queriesExecuted)
	if total > 0 {
		report["avg_duration"] = m.performanceTracker.totalDuration / time.Duration(total)
		report["slow_query_ratio"] = float64(atomic.LoadInt64(&m.performanceTracker.queriesSlow)) / float64(total) * 100
		report["failure_rate"] = float64(atomic.LoadInt64(&m.performanceTracker.queriesFailed)) / float64(total) * 100
	}

	report["pool_stats"] = m.GetPoolStats()
	report["circuit_breaker"] = m.circuitBreaker.getStateString()

	return report
}

func (m *AdvancedConnectionPoolManager) GetHealthReport() map[string]interface{} {
	m.healthMonitor.mu.RLock()
	defer m.healthMonitor.mu.RUnlock()

	report := map[string]interface{}{
		"is_healthy":     m.healthMonitor.lastCheckResult,
		"last_check":     m.healthMonitor.lastCheckTime,
		"total_checks":   len(m.healthMonitor.checks),
	}

	healthy := 0
	for _, check := range m.healthMonitor.checks {
		if check.IsHealthy {
			healthy++
		}
	}

	if len(m.healthMonitor.checks) > 0 {
		report["health_ratio"] = float64(healthy) / float64(len(m.healthMonitor.checks)) * 100
	}

	return report
}

func (m *AdvancedConnectionPoolManager) SetAutoTune(enabled bool) {
	m.autoTuner.mu.Lock()
	defer m.autoTuner.mu.Unlock()
	m.autoTuner.enabled = enabled

	log.Printf("[POOL] Auto-tuning %s", map[bool]string{true: "enabled", false: "disabled"}[enabled])
}

func (m *AdvancedConnectionPoolManager) GetTuningHistory() []TuningDecision {
	m.autoTuner.mu.RLock()
	defer m.autoTuner.mu.RUnlock()

	history := make([]TuningDecision, len(m.autoTuner.tuningHistory))
	copy(history, m.autoTuner.tuningHistory)
	return history
}

func (m *AdvancedConnectionPoolManager) GetMetricsHistory() []PoolMetricSnapshot {
	m.metricsCollector.mu.RLock()
	defer m.metricsCollector.mu.RUnlock()

	metrics := make([]PoolMetricSnapshot, len(m.metricsCollector.metrics))
	copy(metrics, m.metricsCollector.metrics)
	return metrics
}

func (m *AdvancedConnectionPoolManager) ResetStats() {
	m.performanceTracker.mu.Lock()
	defer m.performanceTracker.mu.Unlock()

	atomic.StoreInt64(&m.performanceTracker.queriesExecuted, 0)
	atomic.StoreInt64(&m.performanceTracker.queriesSlow, 0)
	atomic.StoreInt64(&m.performanceTracker.queriesFailed, 0)
	m.performanceTracker.totalDuration = 0
	m.performanceTracker.maxDuration = 0
	m.performanceTracker.minDuration = 24 * time.Hour
	m.performanceTracker.queryHistory = make([]QueryExecutionRecord, 0)

	log.Println("[POOL] Performance stats reset")
}

type ConnectionPoolExporter struct {
	manager *AdvancedConnectionPoolManager
}

func NewConnectionPoolExporter() *ConnectionPoolExporter {
	return &ConnectionPoolExporter{manager: advancedPoolManager}
}

func (e *ConnectionPoolExporter) ExportMetrics() map[string]interface{} {
	if e.manager == nil {
		return map[string]interface{}{"error": "manager not initialized"}
	}

	return map[string]interface{}{
		"performance": e.manager.GetPerformanceReport(),
		"health":      e.manager.GetHealthReport(),
		"pool":        e.manager.GetPoolStats(),
		"tuning":      e.manager.GetTuningHistory(),
	}
}

var globalPoolExporter *ConnectionPoolExporter

func init() {
	globalPoolExporter = NewConnectionPoolExporter()
}

func GetConnectionPoolExporter() *ConnectionPoolExporter {
	return globalPoolExporter
}
