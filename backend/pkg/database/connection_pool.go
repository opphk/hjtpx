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

type PoolMonitor struct {
	mu            sync.RWMutex
	statsHistory  []PoolStatsRecord
	maxHistoryLen int
}

type PoolStatsRecord struct {
	Timestamp time.Time
	Stats     PoolStats
}

var poolMonitor *PoolMonitor
var connectionReuseRate atomic.Int64
var totalWaitCount atomic.Int64
var lastHighLoadTime time.Time

func init() {
	connectionReuseRate.Store(0)
	totalWaitCount.Store(0)
	lastHighLoadTime = time.Now()
}

func InitConnectionPool(cfg *config.Config) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	poolConfig := cfg.Database.ConnectionPool

	maxOpenConns := poolConfig.MaxOpenConns
	maxIdleConns := poolConfig.MaxIdleConns
	_ = poolConfig.MinIdleConns
	connMaxLifetime := time.Duration(poolConfig.ConnMaxLifetimeSecs) * time.Second
	connMaxIdleTime := time.Duration(poolConfig.ConnMaxIdleTimeSecs) * time.Second

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	poolMonitor = &PoolMonitor{
		statsHistory:  make([]PoolStatsRecord, 0),
		maxHistoryLen: 1000,
	}

	if cfg.Database.Monitoring.EnableConnectionMetrics {
		go startPoolMonitoring(cfg)
		go startConnectionReuseCalculation()
	}

	if poolConfig.EnableWarmup {
		go warmupConnections(cfg, poolConfig.WarmupConns)
	}

	if poolConfig.EnableAutoTuning {
		go startAutoTuning(cfg)
	}

	log.Println("Connection pool configured successfully")
	return nil
}

func warmupConnections(cfg *config.Config, warmupConns int) {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Printf("Failed to get database connection for warmup: %v", err)
		return
	}

	log.Printf("Starting connection pool warmup with %d connections", warmupConns)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for i := 0; i < warmupConns; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			conn, err := sqlDB.Conn(ctx)
			if err != nil {
				log.Printf("Failed to warm up connection %d: %v", i, err)
				return
			}

			var result int
			err = conn.QueryRowContext(ctx, "SELECT 1").Scan(&result)
			if err != nil {
				log.Printf("Failed to test warmup connection %d: %v", i, err)
			}

			conn.Close()
		}()
	}

	wg.Wait()
	log.Printf("Connection pool warmup completed")
}

func startAutoTuning(cfg *config.Config) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		autoTunePool(cfg)
	}
}

func autoTunePool(cfg *config.Config) {
	metrics, err := GetConnectionPoolMetrics()
	if err != nil {
		return
	}

	poolConfig := cfg.Database.ConnectionPool
	sqlDB, err := DB.DB()
	if err != nil {
		return
	}

	usagePercent := float64(metrics.ActiveConnections) / float64(metrics.TotalConnections) * 100

	if usagePercent > float64(poolConfig.HighLoadThreshold) && time.Since(lastHighLoadTime) > 5*time.Minute {
		newMaxOpen := int(float64(metrics.TotalConnections) * 1.2)
		if newMaxOpen > poolConfig.MaxOpenConns*2 {
			newMaxOpen = poolConfig.MaxOpenConns * 2
		}

		sqlDB.SetMaxOpenConns(newMaxOpen)
		newMaxIdle := newMaxOpen / 2
		if newMaxIdle > poolConfig.MaxIdleConns {
			newMaxIdle = poolConfig.MaxIdleConns
		}
		sqlDB.SetMaxIdleConns(newMaxIdle)

		lastHighLoadTime = time.Now()
		log.Printf("[AUTO_TUNE] Increased pool size to MaxOpen=%d, MaxIdle=%d due to high load (%.1f%%)",
			newMaxOpen, newMaxIdle, usagePercent)
	} else if usagePercent < float64(poolConfig.LowLoadThreshold) {
		stats := sqlDB.Stats()
		if stats.Idle > poolConfig.MinIdleConns {
			newMaxIdle := stats.Idle - 1
			if newMaxIdle < poolConfig.MinIdleConns {
				newMaxIdle = poolConfig.MinIdleConns
			}
			sqlDB.SetMaxIdleConns(newMaxIdle)
			log.Printf("[AUTO_TUNE] Reduced idle connections to %d due to low load (%.1f%%)",
				newMaxIdle, usagePercent)
		}
	}
}

func startPoolMonitoring(cfg *config.Config) {
	ticker := time.NewTicker(time.Duration(cfg.Database.Monitoring.MetricsIntervalSecs) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats, err := GetPoolStats()
		if err != nil {
			log.Printf("Error getting pool stats: %v", err)
			continue
		}

		poolMonitor.mu.Lock()
		poolMonitor.statsHistory = append(poolMonitor.statsHistory, PoolStatsRecord{
			Timestamp: time.Now(),
			Stats:     *stats,
		})

		if len(poolMonitor.statsHistory) > poolMonitor.maxHistoryLen {
			poolMonitor.statsHistory = poolMonitor.statsHistory[1:]
		}
		poolMonitor.mu.Unlock()

		log.Printf("Pool stats: Open=%d, InUse=%d, Idle=%d, WaitCount=%d",
			stats.OpenConnections,
			stats.InUse,
			stats.Idle,
			stats.WaitCount,
		)
	}
}

func startConnectionReuseCalculation() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats, err := GetPoolStats()
		if err != nil {
			continue
		}

		totalConnections := stats.InUse + stats.Idle
		if totalConnections > 0 {
			reuseRate := int64(float64(stats.InUse) / float64(totalConnections) * 100)
			connectionReuseRate.Store(reuseRate)
		}
	}
}

func GetPoolStatsHistory() []PoolStatsRecord {
	if poolMonitor == nil {
		return nil
	}

	poolMonitor.mu.RLock()
	defer poolMonitor.mu.RUnlock()

	history := make([]PoolStatsRecord, len(poolMonitor.statsHistory))
	copy(history, poolMonitor.statsHistory)
	return history
}

func UpdateConnectionPool(maxOpen, maxIdle int, maxLifetime, maxIdleTime time.Duration) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	if maxOpen > 0 {
		sqlDB.SetMaxOpenConns(maxOpen)
	}
	if maxIdle > 0 {
		sqlDB.SetMaxIdleConns(maxIdle)
	}
	if maxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(maxLifetime)
	}
	if maxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(maxIdleTime)
	}

	log.Printf("Connection pool updated: MaxOpen=%d, MaxIdle=%d", maxOpen, maxIdle)
	return nil
}

func CheckConnectionHealth() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func GetRawDB() (*sql.DB, error) {
	return DB.DB()
}

type ConnectionPoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func (c *ConnectionPoolConfig) Validate() error {
	if c.MaxOpenConns <= 0 {
		return fmt.Errorf("max_open_conns must be positive")
	}
	if c.MaxIdleConns <= 0 {
		return fmt.Errorf("max_idle_conns must be positive")
	}
	if c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("max_idle_conns cannot exceed max_open_conns")
	}
	if c.ConnMaxLifetime <= 0 {
		return fmt.Errorf("conn_max_lifetime must be positive")
	}
	return nil
}

func (c *ConnectionPoolConfig) Optimize(ratio float64) {
	if ratio < 0.5 {
		c.MaxIdleConns = int(float64(c.MaxIdleConns) * 0.8)
		if c.MaxIdleConns < 5 {
			c.MaxIdleConns = 5
		}
	} else if ratio > 0.9 {
		newIdle := int(float64(c.MaxIdleConns) * 1.2)
		if newIdle > c.MaxOpenConns {
			newIdle = c.MaxOpenConns
		}
		c.MaxIdleConns = newIdle
	}
}

type ConnectionPoolMetrics struct {
	TotalConnections  int
	ActiveConnections int
	IdleConnections   int
	StaleConnections  int
	WaitCount         int64
	WaitDuration      time.Duration
	MaxIdleClosed     int64
	MaxLifetimeClosed int64
	ReuseRate         float64
}

func GetConnectionPoolMetrics() (*ConnectionPoolMetrics, error) {
	if DB == nil {
		return &ConnectionPoolMetrics{
			TotalConnections:  0,
			ActiveConnections: 0,
			IdleConnections:   0,
			WaitCount:         0,
			WaitDuration:      0,
			MaxIdleClosed:     0,
			MaxLifetimeClosed: 0,
			ReuseRate:         0,
		}, nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return nil, err
	}

	stats := sqlDB.Stats()

	return &ConnectionPoolMetrics{
		TotalConnections:  stats.MaxOpenConnections,
		ActiveConnections: stats.InUse,
		IdleConnections:   stats.Idle,
		WaitCount:         stats.WaitCount,
		WaitDuration:      stats.WaitDuration,
		MaxIdleClosed:     stats.MaxIdleClosed,
		MaxLifetimeClosed: stats.MaxLifetimeClosed,
		ReuseRate:         float64(connectionReuseRate.Load()),
	}, nil
}

type ConnectionPoolOptimizer struct {
	mu                  sync.RWMutex
	currentConfig       *ConnectionPoolConfig
	healthCheckInterval time.Duration
	autoTuningEnabled   bool
}

func NewConnectionPoolOptimizer(config *ConnectionPoolConfig) *ConnectionPoolOptimizer {
	return &ConnectionPoolOptimizer{
		currentConfig:       config,
		healthCheckInterval: 30 * time.Second,
		autoTuningEnabled:   true,
	}
}

func (o *ConnectionPoolOptimizer) Start() {
	go o.runHealthCheck()
}

func (o *ConnectionPoolOptimizer) runHealthCheck() {
	ticker := time.NewTicker(o.healthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		o.checkAndOptimize()
	}
}

func (o *ConnectionPoolOptimizer) checkAndOptimize() {
	if !o.autoTuningEnabled {
		return
	}

	metrics, err := GetConnectionPoolMetrics()
	if err != nil {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if metrics.WaitCount > 100 {
		ratio := float64(metrics.ActiveConnections) / float64(metrics.TotalConnections)
		o.currentConfig.Optimize(ratio)
		if err := UpdateConnectionPool(
			o.currentConfig.MaxOpenConns,
			o.currentConfig.MaxIdleConns,
			o.currentConfig.ConnMaxLifetime,
			o.currentConfig.ConnMaxIdleTime,
		); err != nil {
			log.Printf("更新连接池配置失败: %v", err)
		}
	}

	if metrics.ReuseRate > 95 {
		log.Printf("[POOL_OPTIMIZATION] High reuse rate: %.2f%%", metrics.ReuseRate)
	}
}

func (o *ConnectionPoolOptimizer) GetConfig() *ConnectionPoolConfig {
	o.mu.RLock()
	defer o.mu.RUnlock()

	configCopy := *o.currentConfig
	return &configCopy
}

func (o *ConnectionPoolOptimizer) SetConfig(config *ConnectionPoolConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	o.currentConfig = config
	return UpdateConnectionPool(
		config.MaxOpenConns,
		config.MaxIdleConns,
		config.ConnMaxLifetime,
		config.ConnMaxIdleTime,
	)
}

func (o *ConnectionPoolOptimizer) EnableAutoTuning(enabled bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.autoTuningEnabled = enabled
}

type ConnectionPoolHealthCheck struct {
	mu              sync.RWMutex
	lastCheckTime   time.Time
	lastCheckResult bool
	checkHistory    []HealthCheckResult
}

type HealthCheckResult struct {
	Timestamp    time.Time
	IsHealthy    bool
	Error        string
	ResponseTime time.Duration
	ActiveConns  int
	IdleConns    int
}

func NewConnectionPoolHealthCheck() *ConnectionPoolHealthCheck {
	return &ConnectionPoolHealthCheck{
		checkHistory: make([]HealthCheckResult, 0),
	}
}

func (h *ConnectionPoolHealthCheck) Run() *HealthCheckResult {
	result := &HealthCheckResult{
		Timestamp: time.Now(),
	}

	start := time.Now()

	sqlDB, err := DB.DB()
	if err != nil {
		result.IsHealthy = false
		result.Error = err.Error()
		return h.recordResult(result)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		result.IsHealthy = false
		result.Error = err.Error()
		return h.recordResult(result)
	}

	result.ResponseTime = time.Since(start)
	result.IsHealthy = true

	stats := sqlDB.Stats()
	result.ActiveConns = stats.InUse
	result.IdleConns = stats.Idle

	if stats.InUse > stats.MaxOpenConnections*80/100 {
		result.IsHealthy = false
		result.Error = fmt.Sprintf("high connection usage: %d/%d", stats.InUse, stats.MaxOpenConnections)
	}

	return h.recordResult(result)
}

func (h *ConnectionPoolHealthCheck) recordResult(result *HealthCheckResult) *HealthCheckResult {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.lastCheckTime = result.Timestamp
	h.lastCheckResult = result.IsHealthy

	h.checkHistory = append(h.checkHistory, *result)
	if len(h.checkHistory) > 100 {
		h.checkHistory = h.checkHistory[1:]
	}

	return result
}

func (h *ConnectionPoolHealthCheck) GetLastCheck() *HealthCheckResult {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.checkHistory) == 0 {
		return nil
	}
	return &h.checkHistory[len(h.checkHistory)-1]
}

func (h *ConnectionPoolHealthCheck) GetHistory(limit int) []HealthCheckResult {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if limit <= 0 || limit > len(h.checkHistory) {
		limit = len(h.checkHistory)
	}

	history := make([]HealthCheckResult, limit)
	copy(history, h.checkHistory[len(h.checkHistory)-limit:])
	return history
}

func (h *ConnectionPoolHealthCheck) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastCheckResult
}

type ConnectionPoolManager struct {
	optimizer   *ConnectionPoolOptimizer
	healthCheck *ConnectionPoolHealthCheck
	mu          sync.RWMutex
}

var globalPoolManager *ConnectionPoolManager

func GetConnectionPoolManager() *ConnectionPoolManager {
	if globalPoolManager == nil {
		globalPoolManager = &ConnectionPoolManager{
			optimizer: NewConnectionPoolOptimizer(&ConnectionPoolConfig{
				MaxOpenConns:    100,
				MaxIdleConns:    20,
				ConnMaxLifetime: 30 * time.Minute,
				ConnMaxIdleTime: 10 * time.Minute,
			}),
			healthCheck: NewConnectionPoolHealthCheck(),
		}
	}
	return globalPoolManager
}

func (m *ConnectionPoolManager) Start() {
	m.optimizer.Start()
	go m.runPeriodicHealthCheck()
}

func (m *ConnectionPoolManager) runPeriodicHealthCheck() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.healthCheck.Run()
	}
}

func (m *ConnectionPoolManager) GetMetrics() (*ConnectionPoolMetrics, error) {
	return GetConnectionPoolMetrics()
}

func (m *ConnectionPoolManager) GetHealth() *HealthCheckResult {
	return m.healthCheck.GetLastCheck()
}

func (m *ConnectionPoolManager) Optimize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.optimizer.currentConfig.Validate()
}

type ConnectionWrapper struct {
	db    *sql.DB
	stats sql.DBStats
	mu    sync.RWMutex
}

func (w *ConnectionWrapper) RecordStats() {
	if w.db == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stats = w.db.Stats()
}

func (w *ConnectionWrapper) GetStats() sql.DBStats {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.stats
}
