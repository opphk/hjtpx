package database

import (
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
)

type EnhancedConnectionPoolOptimizer struct {
	db                   *gorm.DB
	config               *EnhancedPoolConfig
	metrics              *PoolMetrics
	mu                   sync.RWMutex
	adaptiveEnabled      bool
	healthCheckInterval  time.Duration
	lastOptimizationTime time.Time
}

type EnhancedPoolConfig struct {
	InitialMaxOpenConns     int
	InitialMaxIdleConns     int
	MinIdleConns            int
	MaxIdleConns            int
	ConnMaxLifetime         time.Duration
	ConnMaxIdleTime         time.Duration
	HealthCheckPeriod       time.Duration
	OptimizationThreshold   float64
	ConnectionTimeout       time.Duration
	RetryAttempts           int
	RetryDelay              time.Duration
}

type PoolMetrics struct {
	mu                     sync.RWMutex
	TotalRequests          int64
	TotalWaitTime          time.Duration
	AvgWaitTime            time.Duration
	MaxWaitTime            time.Duration
	TotalErrors            int64
	ConnectionTimeouts     int64
	QueryTimeouts          int64
	HealthyChecks          int64
	UnhealthyChecks        int64
	LastHealthCheckTime    time.Time
	LastError              error
	OptimizationCount      int
	CurrentConfigSnapshot  *ConnectionPoolConfig
}

var defaultEnhancedConfig = &EnhancedPoolConfig{
	InitialMaxOpenConns:   100,
	InitialMaxIdleConns:   20,
	MinIdleConns:           5,
	MaxIdleConns:           50,
	ConnMaxLifetime:        30 * time.Minute,
	ConnMaxIdleTime:        10 * time.Minute,
	HealthCheckPeriod:      30 * time.Second,
	OptimizationThreshold:  0.8,
	ConnectionTimeout:      10 * time.Second,
	RetryAttempts:          3,
	RetryDelay:             100 * time.Millisecond,
}

func NewEnhancedConnectionPoolOptimizer(db *gorm.DB, cfg *EnhancedPoolConfig) *EnhancedConnectionPoolOptimizer {
	if cfg == nil {
		cfg = defaultEnhancedConfig
	}

	optimizer := &EnhancedConnectionPoolOptimizer{
		db:                   db,
		config:               cfg,
		metrics:              &PoolMetrics{},
		adaptiveEnabled:      true,
		healthCheckInterval:  cfg.HealthCheckPeriod,
	}

	sqlDB, err := db.DB()
	if err == nil {
		optimizer.metrics.CurrentConfigSnapshot = &ConnectionPoolConfig{
			MaxOpenConns:    sqlDB.Stats().MaxOpenConnections,
			MaxIdleConns:    sqlDB.Stats().MaxOpenConnections,
			ConnMaxLifetime: cfg.ConnMaxLifetime,
			ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		}
	}

	return optimizer
}

func (o *EnhancedConnectionPoolOptimizer) Start() {
	go o.runHealthCheckLoop()
	go o.runMetricsCollector()
	log.Println("[ENHANCED_POOL_OPTIMIZER] Started")
}

func (o *EnhancedConnectionPoolOptimizer) WarmUpConnections() error {
	log.Println("[ENHANCED_POOL_OPTIMIZER] Warming up connection pool...")

	sqlDB, err := o.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	for i := 0; i < o.config.InitialMaxIdleConns; i++ {
		if err := sqlDB.Ping(); err != nil {
			return fmt.Errorf("failed to warm up connection %d: %w", i, err)
		}
	}

	log.Printf("[ENHANCED_POOL_OPTIMIZER] Warmed up %d connections", o.config.InitialMaxIdleConns)
	return nil
}

func (o *EnhancedConnectionPoolOptimizer) runHealthCheckLoop() {
	ticker := time.NewTicker(o.healthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		o.performHealthCheck()
	}
}

func (o *EnhancedConnectionPoolOptimizer) performHealthCheck() {
	sqlDB, err := o.db.DB()
	if err != nil {
		o.recordHealthCheckError(err)
		return
	}

	stats := sqlDB.Stats()

	o.metrics.mu.Lock()
	o.metrics.LastHealthCheckTime = time.Now()

	if stats.InUse > stats.MaxOpenConnections {
		o.metrics.UnhealthyChecks++
		o.metrics.LastError = fmt.Errorf("connection overflow: %d/%d", stats.InUse, stats.MaxOpenConnections)
		log.Printf("[POOL_WARNING] Connection overflow detected: %d/%d", stats.InUse, stats.MaxOpenConnections)
	} else if float64(stats.InUse)/float64(stats.MaxOpenConnections) > o.config.OptimizationThreshold {
		o.metrics.UnhealthyChecks++
		if o.adaptiveEnabled {
			o.adjustPoolSize(true)
		}
	} else {
		o.metrics.HealthyChecks++
	}
	o.metrics.mu.Unlock()
}

func (o *EnhancedConnectionPoolOptimizer) adjustPoolSize(increase bool) {
	sqlDB, err := o.db.DB()
	if err != nil {
		return
	}

	currentStats := sqlDB.Stats()
	newMaxOpen := currentStats.MaxOpenConnections
	newMaxIdle := currentStats.MaxOpenConnections

	if increase {
		newMaxOpen = int(float64(currentStats.MaxOpenConnections) * 1.2)
		newMaxIdle = int(float64(currentStats.MaxOpenConnections) * 0.4)
	} else {
		newMaxOpen = int(float64(currentStats.MaxOpenConnections) * 0.8)
		newMaxIdle = int(float64(currentStats.MaxOpenConnections) * 0.2)
	}

	if newMaxOpen > 500 {
		newMaxOpen = 500
	}
	if newMaxOpen < 10 {
		newMaxOpen = 10
	}
	if newMaxIdle > newMaxOpen {
		newMaxIdle = newMaxOpen
	}
	if newMaxIdle < o.config.MinIdleConns {
		newMaxIdle = o.config.MinIdleConns
	}

	sqlDB.SetMaxOpenConns(newMaxOpen)
	sqlDB.SetMaxIdleConns(newMaxIdle)

	o.metrics.mu.Lock()
	o.metrics.OptimizationCount++
	o.metrics.CurrentConfigSnapshot = &ConnectionPoolConfig{
		MaxOpenConns:    newMaxOpen,
		MaxIdleConns:    newMaxIdle,
		ConnMaxLifetime: o.config.ConnMaxLifetime,
		ConnMaxIdleTime: o.config.ConnMaxIdleTime,
	}
	o.metrics.mu.Unlock()

	o.lastOptimizationTime = time.Now()
	log.Printf("[POOL_OPTIMIZATION] Adjusted pool: MaxOpen=%d, MaxIdle=%d", newMaxOpen, newMaxIdle)
}

func (o *EnhancedConnectionPoolOptimizer) runMetricsCollector() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		o.collectMetrics()
	}
}

func (o *EnhancedConnectionPoolOptimizer) collectMetrics() {
	sqlDB, err := o.db.DB()
	if err != nil {
		return
	}

	stats := sqlDB.Stats()

	o.metrics.mu.Lock()
	total := stats.InUse + stats.Idle
	if total > 0 {
		o.metrics.AvgWaitTime = stats.WaitDuration / time.Duration(total)
	}
	if stats.WaitDuration > o.metrics.MaxWaitTime {
		o.metrics.MaxWaitTime = stats.WaitDuration
	}
	o.metrics.mu.Unlock()
}

func (o *EnhancedConnectionPoolOptimizer) recordHealthCheckError(err error) {
	o.metrics.mu.Lock()
	defer o.metrics.mu.Unlock()

	o.metrics.UnhealthyChecks++
	o.metrics.TotalErrors++
	o.metrics.LastError = err
}

func (o *EnhancedConnectionPoolOptimizer) Optimize() {
	o.adjustPoolSize(true)
}

func (o *EnhancedConnectionPoolOptimizer) GetMetrics() *PoolMetrics {
	o.metrics.mu.RLock()
	defer o.metrics.mu.RUnlock()

	metricsCopy := *o.metrics
	return &metricsCopy
}

func (o *EnhancedConnectionPoolOptimizer) GetOptimizationHistory() []OptimizationEvent {
	return nil
}

type OptimizationEvent struct {
	Timestamp   time.Time
	EventType   string
	OldValue    interface{}
	NewValue    interface{}
	Reason      string
}

func (o *EnhancedConnectionPoolOptimizer) EnableAdaptiveMode(enabled bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.adaptiveEnabled = enabled
	log.Printf("[POOL_OPTIMIZER] Adaptive mode %s", map[bool]string{true: "enabled", false: "disabled"}[enabled])
}

func (o *EnhancedConnectionPoolOptimizer) ForceOptimization() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	sqlDB, err := o.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	stats := sqlDB.Stats()
	usageRatio := float64(stats.InUse) / float64(stats.MaxOpenConnections)

	if usageRatio > o.config.OptimizationThreshold {
		o.adjustPoolSizeLocked(true)
	}

	return nil
}

func (o *EnhancedConnectionPoolOptimizer) adjustPoolSizeLocked(increase bool) {
	sqlDB, _ := o.db.DB()
	stats := sqlDB.Stats()

	newMaxOpen := stats.MaxOpenConnections
	newMaxIdle := stats.MaxOpenConnections

	if increase {
		newMaxOpen = int(float64(stats.MaxOpenConnections) * 1.2)
		newMaxIdle = int(float64(stats.MaxOpenConnections) * 0.4)
	} else {
		newMaxOpen = int(float64(stats.MaxOpenConnections) * 0.8)
		newMaxIdle = int(float64(stats.MaxOpenConnections) * 0.2)
	}

	if newMaxOpen > 500 {
		newMaxOpen = 500
	}
	if newMaxOpen < 10 {
		newMaxOpen = 10
	}
	if newMaxIdle > newMaxOpen {
		newMaxIdle = newMaxOpen
	}
	if newMaxIdle < o.config.MinIdleConns {
		newMaxIdle = o.config.MinIdleConns
	}

	sqlDB.SetMaxOpenConns(newMaxOpen)
	sqlDB.SetMaxIdleConns(newMaxIdle)

	o.metrics.OptimizationCount++
}

func (o *EnhancedConnectionPoolOptimizer) GetCurrentConfiguration() *EnhancedPoolConfig {
	o.mu.RLock()
	defer o.mu.RUnlock()

	configCopy := *o.config
	return &configCopy
}

func (o *EnhancedConnectionPoolOptimizer) UpdateConfiguration(cfg *EnhancedPoolConfig) error {
	if cfg.InitialMaxOpenConns <= 0 || cfg.InitialMaxIdleConns <= 0 {
		return fmt.Errorf("invalid pool configuration")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	o.config = cfg
	o.healthCheckInterval = cfg.HealthCheckPeriod

	sqlDB, err := o.db.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxOpenConns(cfg.InitialMaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.InitialMaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	log.Printf("[POOL_OPTIMIZER] Configuration updated: MaxOpen=%d, MaxIdle=%d",
		cfg.InitialMaxOpenConns, cfg.InitialMaxIdleConns)

	return nil
}

func (o *EnhancedConnectionPoolOptimizer) GenerateReport() *PoolOptimizationReport {
	metrics := o.GetMetrics()
	config := o.GetCurrentConfiguration()

	report := &PoolOptimizationReport{
		Timestamp:             time.Now(),
		Configuration:         config,
		Metrics:               metrics,
		OptimizationAvailable: o.shouldOptimize(metrics),
		Recommendations:       o.generateRecommendations(metrics),
	}

	return report
}

func (o *EnhancedConnectionPoolOptimizer) shouldOptimize(metrics *PoolMetrics) bool {
	if metrics.TotalErrors > 100 {
		return true
	}
	if metrics.ConnectionTimeouts > 10 {
		return true
	}
	return false
}

func (o *EnhancedConnectionPoolOptimizer) generateRecommendations(metrics *PoolMetrics) []string {
	var recs []string

	if metrics.TotalErrors > 100 {
		recs = append(recs, "HIGH: Many connection errors detected, review database server health")
	}

	if metrics.ConnectionTimeouts > 10 {
		recs = append(recs, "MEDIUM: Connection timeouts detected, consider increasing pool size or timeout values")
	}

	if metrics.UnhealthyChecks > metrics.HealthyChecks {
		recs = append(recs, "Connection pool health is degraded, adaptive optimization may help")
	}

	recs = append(recs, "Monitor these metrics over time to identify patterns")
	recs = append(recs, "Consider enabling connection pooling at database level (e.g., PgBouncer) for production")

	return recs
}

type PoolOptimizationReport struct {
	Timestamp             time.Time                `json:"timestamp"`
	Configuration         *EnhancedPoolConfig      `json:"configuration"`
	Metrics               *PoolMetrics             `json:"metrics"`
	OptimizationAvailable bool                     `json:"optimization_available"`
	Recommendations       []string                 `json:"recommendations"`
}

func (o *EnhancedConnectionPoolOptimizer) ResetMetrics() {
	o.metrics.mu.Lock()
	defer o.metrics.mu.Unlock()

	o.metrics.TotalRequests = 0
	o.metrics.TotalWaitTime = 0
	o.metrics.TotalErrors = 0
	o.metrics.ConnectionTimeouts = 0
	o.metrics.QueryTimeouts = 0
	o.metrics.OptimizationCount = 0
}

func (o *EnhancedConnectionPoolOptimizer) Shutdown() error {
	log.Println("[POOL_OPTIMIZER] Shutting down optimizer...")

	sqlDB, err := o.db.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxOpenConns(0)
	sqlDB.SetMaxIdleConns(0)

	log.Println("[POOL_OPTIMIZER] Optimizer shutdown complete")
	return nil
}
