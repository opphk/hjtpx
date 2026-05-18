package database

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

type EnhancedConnectionPoolOptimizer struct {
	db               *gorm.DB
	config          *config.Config
	monitorInterval  time.Duration
	mu              sync.RWMutex
	running         bool
	stopChan        chan struct{}
}

func NewEnhancedConnectionPoolOptimizer(db *gorm.DB, cfg *config.Config) *EnhancedConnectionPoolOptimizer {
	return &EnhancedConnectionPoolOptimizer{
		db:              db,
		config:          cfg,
		monitorInterval: 30 * time.Second,
		stopChan:        make(chan struct{}),
	}
}

func (o *EnhancedConnectionPoolOptimizer) Start() {
	o.mu.Lock()
	if o.running {
		o.mu.Unlock()
		return
	}
	o.running = true
	o.mu.Unlock()

	go o.monitorAndOptimize()
	log.Println("[ENHANCED_POOL_OPTIMIZER] Started connection pool optimizer")
}

func (o *EnhancedConnectionPoolOptimizer) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.running {
		return
	}

	close(o.stopChan)
	o.running = false
	log.Println("[ENHANCED_POOL_OPTIMIZER] Stopped connection pool optimizer")
}

func (o *EnhancedConnectionPoolOptimizer) monitorAndOptimize() {
	ticker := time.NewTicker(o.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-o.stopChan:
			return
		case <-ticker.C:
			o.analyzeAndOptimize()
		}
	}
}

func (o *EnhancedConnectionPoolOptimizer) analyzeAndOptimize() {
	metrics, err := GetConnectionPoolMetrics()
	if err != nil {
		log.Printf("[ENHANCED_POOL_OPTIMIZER] Failed to get metrics: %v", err)
		return
	}

	if metrics.ReuseRate < 70 {
		log.Printf("[ENHANCED_POOL_OPTIMIZER] Low connection reuse rate: %.2f%%, considering optimization", metrics.ReuseRate)
		o.optimizeForLowReuse()
	}

	if metrics.WaitCount > 100 {
		log.Printf("[ENHANCED_POOL_OPTIMIZER] High wait count: %d, considering increasing pool size", metrics.WaitCount)
		o.optimizeForHighWaits()
	}

	if metrics.MaxIdleClosed > 100 {
		log.Printf("[ENHANCED_POOL_OPTIMIZER] High idle connection closure: %d", metrics.MaxIdleClosed)
		o.optimizeIdleConnections()
	}

	if metrics.MaxLifetimeClosed > 50 {
		log.Printf("[ENHANCED_POOL_OPTIMIZER] High lifetime closure: %d", metrics.MaxLifetimeClosed)
		o.adjustLifetimeSettings()
	}
}

func (o *EnhancedConnectionPoolOptimizer) optimizeForLowReuse() {
	sqlDB, err := o.db.DB()
	if err != nil {
		return
	}

	stats := sqlDB.Stats()
	currentMaxOpen := stats.MaxOpenConnections

	newMaxIdle := int(float64(currentMaxOpen) * 0.6)
	if newMaxIdle < 10 {
		newMaxIdle = 10
	}

	sqlDB.SetMaxIdleConns(newMaxIdle)
	log.Printf("[ENHANCED_POOL_OPTIMIZER] Optimized for low reuse: max_idle=%d", newMaxIdle)
}

func (o *EnhancedConnectionPoolOptimizer) optimizeForHighWaits() {
	sqlDB, err := o.db.DB()
	if err != nil {
		return
	}

	stats := sqlDB.Stats()
	currentMaxOpen := stats.MaxOpenConnections

	newMaxOpen := int(float64(currentMaxOpen) * 1.5)
	if newMaxOpen > 500 {
		newMaxOpen = 500
	}

	sqlDB.SetMaxOpenConns(newMaxOpen)
	log.Printf("[ENHANCED_POOL_OPTIMIZER] Optimized for high waits: max_open=%d", newMaxOpen)
}

func (o *EnhancedConnectionPoolOptimizer) optimizeIdleConnections() {
	sqlDB, err := o.db.DB()
	if err != nil {
		return
	}

	newIdleTime := 5 * time.Minute
	sqlDB.SetConnMaxIdleTime(newIdleTime)
	log.Printf("[ENHANCED_POOL_OPTIMIZER] Optimized idle connections: max_idle_time=%v", newIdleTime)
}

func (o *EnhancedConnectionPoolOptimizer) adjustLifetimeSettings() {
	sqlDB, err := o.db.DB()
	if err != nil {
		return
	}

	newLifetime := 15 * time.Minute
	sqlDB.SetConnMaxLifetime(newLifetime)
	log.Printf("[ENHANCED_POOL_OPTIMIZER] Adjusted lifetime: conn_max_lifetime=%v", newLifetime)
}

func (o *EnhancedConnectionPoolOptimizer) GetOptimizationHistory() []PoolOptimizationRecord {
	return []PoolOptimizationRecord{}
}

type PoolOptimizationRecord struct {
	Timestamp    time.Time
	Metric       string
	OldValue     interface{}
	NewValue     interface{}
	Reason       string
}

func (o *EnhancedConnectionPoolOptimizer) WarmUpConnections() error {
	sqlDB, err := o.db.DB()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return err
	}

	log.Println("[ENHANCED_POOL_OPTIMIZER] Connection pool warmed up successfully")
	return nil
}

type ConnectionPoolHealthStatus struct {
	IsHealthy       bool
	UtilizationRate float64
	Recommendations []string
	LastCheck       time.Time
}

func (o *EnhancedConnectionPoolOptimizer) GetHealthStatus() *ConnectionPoolHealthStatus {
	metrics, err := GetConnectionPoolMetrics()
	if err != nil {
		return &ConnectionPoolHealthStatus{
			IsHealthy:       false,
			Recommendations: []string{"Failed to retrieve metrics"},
		}
	}

	status := &ConnectionPoolHealthStatus{
		IsHealthy: true,
		LastCheck: time.Now(),
	}

	if metrics.TotalConnections > 0 {
		status.UtilizationRate = float64(metrics.ActiveConnections) / float64(metrics.TotalConnections) * 100
	}

	if status.UtilizationRate > 90 {
		status.Recommendations = append(status.Recommendations, "Connection pool utilization is very high, consider increasing pool size")
	}

	if metrics.ReuseRate < 80 {
		status.Recommendations = append(status.Recommendations, "Connection reuse rate is low, consider adjusting max_idle_conns")
	}

	if metrics.WaitCount > 1000 {
		status.Recommendations = append(status.Recommendations, "High connection wait count, consider enabling query caching or increasing pool size")
	}

	return status
}

func (o *EnhancedConnectionPoolOptimizer) ForceReconfigure(maxOpen, maxIdle int, maxLifetime, maxIdleTime time.Duration) error {
	sqlDB, err := o.db.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(maxLifetime)
	sqlDB.SetConnMaxIdleTime(maxIdleTime)

	log.Printf("[ENHANCED_POOL_OPTIMIZER] Force reconfigured: max_open=%d, max_idle=%d, max_lifetime=%v, max_idle_time=%v",
		maxOpen, maxIdle, maxLifetime, maxIdleTime)

	return nil
}
