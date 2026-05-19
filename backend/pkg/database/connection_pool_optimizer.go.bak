package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type PoolMetrics struct {
	OpenConnections      int           `json:"open_connections"`
	InUse                int           `json:"in_use"`
	Idle                 int           `json:"idle"`
	WaitCount            int64         `json:"wait_count"`
	WaitDuration         time.Duration `json:"wait_duration"`
	MaxIdleClosed        int64         `json:"max_idle_closed"`
	MaxLifetimeClosed    int64         `json:"max_lifetime_closed"`
	AvgAcquireTime       time.Duration `json:"avg_acquire_time"`
	HealthCheckFailures  int64         `json:"health_check_failures"`
	TotalQueriesExecuted int64         `json:"total_queries_executed"`
}

type PoolOptimizerConfig struct {
	InitialMaxOpenConns  int           `json:"initial_max_open_conns"`
	InitialMaxIdleConns  int           `json:"initial_max_idle_conns"`
	ConnMaxLifetime      time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime      time.Duration `json:"conn_max_idle_time"`
	HealthCheckInterval  time.Duration `json:"health_check_interval"`
	EnableAutoScaling    bool          `json:"enable_auto_scaling"`
	ScaleUpThreshold     float64       `json:"scale_up_threshold"` // 使用率阈值（0-1）
	ScaleDownThreshold   float64       `json:"scale_down_threshold"`
	MaxScaledConns       int           `json:"max_scaled_conns"`
	MinScaledConns       int           `json:"min_scaled_conns"`
}

var DefaultPoolOptimizerConfig = &PoolOptimizerConfig{
	InitialMaxOpenConns: 25,
	InitialMaxIdleConns: 10,
	ConnMaxLifetime:     1 * time.Hour,
	ConnMaxIdleTime:     30 * time.Minute,
	HealthCheckInterval: 10 * time.Second,
	EnableAutoScaling:   true,
	ScaleUpThreshold:    0.8,
	ScaleDownThreshold:  0.3,
	MaxScaledConns:      100,
	MinScaledConns:      5,
}

type ConnectionPoolOptimizer struct {
	db              *sql.DB
	config          *PoolOptimizerConfig
	metrics         *PoolMetrics
	mu              sync.RWMutex
	stopCh          chan struct{}
	wg              sync.WaitGroup
	acquireTimings  []time.Duration
	timingMu        sync.Mutex
	lastScaleTime   time.Time
	scaleCooldown   time.Duration
}

func NewConnectionPoolOptimizer(db *sql.DB, config *PoolOptimizerConfig) *ConnectionPoolOptimizer {
	if config == nil {
		config = DefaultPoolOptimizerConfig
	}

	if db != nil {
		db.SetMaxOpenConns(config.InitialMaxOpenConns)
		db.SetMaxIdleConns(config.InitialMaxIdleConns)
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
		db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}

	return &ConnectionPoolOptimizer{
		db:            db,
		config:        config,
		metrics:       &PoolMetrics{},
		stopCh:        make(chan struct{}),
		scaleCooldown: 30 * time.Second,
	}
}

func (cpo *ConnectionPoolOptimizer) Start() {
	cpo.wg.Add(1)
	go cpo.monitorLoop()
}

func (cpo *ConnectionPoolOptimizer) Stop() {
	close(cpo.stopCh)
	cpo.wg.Wait()
}

func (cpo *ConnectionPoolOptimizer) monitorLoop() {
	defer cpo.wg.Done()

	ticker := time.NewTicker(cpo.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cpo.stopCh:
			return
		case <-ticker.C:
			cpo.checkHealth()
			if cpo.config.EnableAutoScaling {
				cpo.autoScale()
			}
		}
	}
}

func (cpo *ConnectionPoolOptimizer) checkHealth() {
	if cpo.db == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := cpo.db.PingContext(ctx)
	if err != nil {
		cpo.metrics.HealthCheckFailures++
	}

	stats := cpo.db.Stats()
	cpo.mu.Lock()
	cpo.metrics.OpenConnections = stats.OpenConnections
	cpo.metrics.InUse = stats.InUse
	cpo.metrics.Idle = stats.Idle
	cpo.metrics.WaitCount = stats.WaitCount
	cpo.metrics.WaitDuration = stats.WaitDuration
	cpo.metrics.MaxIdleClosed = stats.MaxIdleClosed
	cpo.metrics.MaxLifetimeClosed = stats.MaxLifetimeClosed
	cpo.mu.Unlock()
}

func (cpo *ConnectionPoolOptimizer) autoScale() {
	if time.Since(cpo.lastScaleTime) < cpo.scaleCooldown {
		return
	}

	stats := cpo.db.Stats()
	usageRate := float64(stats.InUse) / float64(stats.MaxOpenConnections)

	cpo.mu.Lock()
	defer cpo.mu.Unlock()

	currentMaxOpen := stats.MaxOpenConnections

	if usageRate >= cpo.config.ScaleUpThreshold && currentMaxOpen < cpo.config.MaxScaledConns {
		newMax := int(float64(currentMaxOpen) * 1.2)
		if newMax > cpo.config.MaxScaledConns {
			newMax = cpo.config.MaxScaledConns
		}
		cpo.db.SetMaxOpenConns(newMax)
		newMaxIdle := newMax / 2
		if newMaxIdle > cpo.config.InitialMaxIdleConns {
			cpo.db.SetMaxIdleConns(newMaxIdle)
		}
		cpo.lastScaleTime = time.Now()
	} else if usageRate <= cpo.config.ScaleDownThreshold && currentMaxOpen > cpo.config.MinScaledConns {
		newMax := int(float64(currentMaxOpen) * 0.8)
		if newMax < cpo.config.MinScaledConns {
			newMax = cpo.config.MinScaledConns
		}
		cpo.db.SetMaxOpenConns(newMax)
		cpo.lastScaleTime = time.Now()
	}
}

func (cpo *ConnectionPoolOptimizer) GetMetrics() *PoolMetrics {
	cpo.mu.RLock()
	defer cpo.mu.RUnlock()

	metrics := *cpo.metrics
	return &metrics
}

func (cpo *ConnectionPoolOptimizer) RecordAcquireTime(duration time.Duration) {
	cpo.timingMu.Lock()
	defer cpo.timingMu.Unlock()

	cpo.acquireTimings = append(cpo.acquireTimings, duration)

	// 保留最近1000个样本
	if len(cpo.acquireTimings) > 1000 {
		cpo.acquireTimings = cpo.acquireTimings[1:]
	}

	// 计算平均获取时间
	var total time.Duration
	for _, t := range cpo.acquireTimings {
		total += t
	}
	cpo.mu.Lock()
	if len(cpo.acquireTimings) > 0 {
		cpo.metrics.AvgAcquireTime = total / time.Duration(len(cpo.acquireTimings))
	}
	cpo.mu.Unlock()
}

func (cpo *ConnectionPoolOptimizer) IncrementQueryCount() {
	atomic.AddInt64(&cpo.metrics.TotalQueriesExecuted, 1)
}

type ConnectionWrapper struct {
	db        *sql.DB
	optimizer *ConnectionPoolOptimizer
}

func NewConnectionWrapper(db *sql.DB, optimizer *ConnectionPoolOptimizer) *ConnectionWrapper {
	return &ConnectionWrapper{
		db:        db,
		optimizer: optimizer,
	}
}

func (cw *ConnectionWrapper) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if cw.optimizer != nil {
		cw.optimizer.IncrementQueryCount()
		start := time.Now()
		defer func() {
			cw.optimizer.RecordAcquireTime(time.Since(start))
		}()
	}
	return cw.db.QueryContext(ctx, query, args...)
}

func (cw *ConnectionWrapper) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if cw.optimizer != nil {
		cw.optimizer.IncrementQueryCount()
		start := time.Now()
		defer func() {
			cw.optimizer.RecordAcquireTime(time.Since(start))
		}()
	}
	return cw.db.ExecContext(ctx, query, args...)
}

func (cw *ConnectionWrapper) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if cw.optimizer != nil {
		cw.optimizer.IncrementQueryCount()
		start := time.Now()
		defer func() {
			cw.optimizer.RecordAcquireTime(time.Since(start))
		}()
	}
	return cw.db.QueryRowContext(ctx, query, args...)
}

type PoolHealthChecker struct {
	db            *sql.DB
	checkInterval time.Duration
	timeout       time.Duration
	mu            sync.Mutex
	isHealthy     bool
	lastError     error
	lastCheck     time.Time
}

func NewPoolHealthChecker(db *sql.DB, checkInterval, timeout time.Duration) *PoolHealthChecker {
	if checkInterval == 0 {
		checkInterval = 10 * time.Second
	}
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	return &PoolHealthChecker{
		db:            db,
		checkInterval: checkInterval,
		timeout:       timeout,
		isHealthy:     true,
	}
}

func (phc *PoolHealthChecker) Start() {
	go phc.checkLoop()
}

func (phc *PoolHealthChecker) checkLoop() {
	ticker := time.NewTicker(phc.checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		phc.check()
	}
}

func (phc *PoolHealthChecker) check() {
	phc.mu.Lock()
	defer phc.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), phc.timeout)
	defer cancel()

	start := time.Now()
	err := phc.db.PingContext(ctx)
	duration := time.Since(start)

	phc.lastCheck = time.Now()

	if err != nil {
		phc.isHealthy = false
		phc.lastError = err
	} else {
		phc.isHealthy = true
		phc.lastError = nil
	}
}

func (phc *PoolHealthChecker) IsHealthy() bool {
	phc.mu.Lock()
	defer phc.mu.Unlock()
	return phc.isHealthy
}

func (phc *PoolHealthChecker) GetStatus() (bool, error, time.Time) {
	phc.mu.Lock()
	defer phc.mu.Unlock()
	return phc.isHealthy, phc.lastError, phc.lastCheck
}

var (
	globalPoolOptimizer *ConnectionPoolOptimizer
	poolOptimizerOnce   sync.Once
)

func InitPoolOptimizer(db *sql.DB, config *PoolOptimizerConfig) {
	poolOptimizerOnce.Do(func() {
		globalPoolOptimizer = NewConnectionPoolOptimizer(db, config)
		globalPoolOptimizer.Start()
	})
}

func GetPoolOptimizer() *ConnectionPoolOptimizer {
	return globalPoolOptimizer
}

func OptimizeConnectionPool(db *sql.DB, maxOpen, maxIdle int, maxLifetime, maxIdleTime time.Duration) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	if maxLifetime > 0 {
		db.SetConnMaxLifetime(maxLifetime)
	}
	if maxIdleTime > 0 {
		db.SetConnMaxIdleTime(maxIdleTime)
	}

	return nil
}
