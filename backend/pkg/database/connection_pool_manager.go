package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

type ConnectionPoolStats struct {
	TotalConnections    int64
	ActiveConnections   int64
	IdleConnections     int64
	WaitCount           int64
	WaitDuration        time.Duration
	MaxIdleTimeout      time.Duration
	MaxLifetime         time.Duration
	LastCheckTime       time.Time
	HealthCheckDuration time.Duration
	ConnectionErrors    int64
}

type ConnectionPoolManager struct {
	mu               sync.RWMutex
	config           *config.ConnectionPoolConfig
	db               *gorm.DB
	stats            *ConnectionPoolStats
	isRunning        bool
	stopChan         chan struct{}
	healthTicker     *time.Ticker
	autoTuneTicker   *time.Ticker
}

var (
	globalPoolManager *ConnectionPoolManager
	managerOnce       sync.Once
)

func NewConnectionPoolManager(cfg *config.ConnectionPoolConfig, db *gorm.DB) *ConnectionPoolManager {
	managerOnce.Do(func() {
		globalPoolManager = &ConnectionPoolManager{
			config: cfg,
			db:     db,
			stats: &ConnectionPoolStats{
				MaxIdleTimeout: 10 * time.Minute,
				MaxLifetime:    time.Hour,
			},
			isRunning: false,
			stopChan:  make(chan struct{}),
		}
	})
	return globalPoolManager
}

func GetConnectionPoolManager() *ConnectionPoolManager {
	return globalPoolManager
}

func (m *ConnectionPoolManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("connection pool manager already running")
	}

	m.isRunning = true

	if m.config.EnableWarmup && m.config.WarmupConns > 0 {
		m.warmupConnections()
	}

	if m.config.HealthCheckInterval > 0 {
		interval := time.Duration(m.config.HealthCheckInterval) * time.Second
		m.healthTicker = time.NewTicker(interval)
		go m.healthCheckLoop()
	}

	if m.config.EnableAutoTuning {
		m.autoTuneTicker = time.NewTicker(time.Minute * 5)
		go m.autoTuneLoop()
	}

	log.Println("Connection pool manager started successfully")
	return nil
}

func (m *ConnectionPoolManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return fmt.Errorf("connection pool manager not running")
	}

	m.isRunning = false

	if m.healthTicker != nil {
		m.healthTicker.Stop()
	}

	if m.autoTuneTicker != nil {
		m.autoTuneTicker.Stop()
	}

	close(m.stopChan)

	log.Println("Connection pool manager stopped")
	return nil
}

func (m *ConnectionPoolManager) warmupConnections() {
	log.Printf("Warming up %d connections...", m.config.WarmupConns)
	ctx := context.Background()

	for i := 0; i < m.config.WarmupConns; i++ {
		sqlDB, err := m.db.DB()
		if err != nil {
			log.Printf("Failed to get underlying DB for warmup: %v", err)
			continue
		}

		if err := sqlDB.PingContext(ctx); err != nil {
			log.Printf("Failed to ping connection during warmup: %v", err)
			continue
		}
	}

	log.Printf("Connection warmup completed, %d connections ready", m.config.WarmupConns)
}

func (m *ConnectionPoolManager) healthCheckLoop() {
	for {
		select {
		case <-m.healthTicker.C:
			m.performHealthCheck()
		case <-m.stopChan:
			return
		}
	}
}

func (m *ConnectionPoolManager) performHealthCheck() {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sqlDB, err := m.db.DB()
	if err != nil {
		log.Printf("Health check failed: unable to get DB: %v", err)
		atomic.AddInt64(&m.stats.ConnectionErrors, 1)
		return
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		log.Printf("Health check failed: ping error: %v", err)
		atomic.AddInt64(&m.stats.ConnectionErrors, 1)
		return
	}

	m.stats.HealthCheckDuration = time.Since(start)
	m.stats.LastCheckTime = time.Now()

	if err := m.refreshStats(); err != nil {
		log.Printf("Failed to refresh stats: %v", err)
	}
}

func (m *ConnectionPoolManager) refreshStats() error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return err
	}

	stats := sqlDB.Stats()

	atomic.StoreInt64(&m.stats.TotalConnections, int64(stats.MaxOpenConnections))
	atomic.StoreInt64(&m.stats.ActiveConnections, int64(stats.InUse))
	atomic.StoreInt64(&m.stats.IdleConnections, int64(stats.Idle))
	atomic.StoreInt64(&m.stats.WaitCount, int64(stats.WaitCount))
	m.stats.WaitDuration = stats.WaitDuration

	return nil
}

func (m *ConnectionPoolManager) autoTuneLoop() {
	for {
		select {
		case <-m.autoTuneTicker.C:
			m.performAutoTuning()
		case <-m.stopChan:
			return
		}
	}
}

func (m *ConnectionPoolManager) performAutoTuning() {
	sqlDB, err := m.db.DB()
	if err != nil {
		log.Printf("Auto-tuning failed: unable to get DB: %v", err)
		return
	}

	stats := sqlDB.Stats()

	loadPercent := float64(stats.InUse) / float64(stats.MaxOpenConnections) * 100

	if loadPercent >= float64(m.config.HighLoadThreshold) {
		newMaxOpen := int(float64(m.config.MaxOpenConns) * 1.2)
		if newMaxOpen > m.config.MaxOpenConns*2 {
			newMaxOpen = m.config.MaxOpenConns * 2
		}

		sqlDB.SetMaxOpenConns(newMaxOpen)
		log.Printf("Auto-tuning: increased max connections from %d to %d due to high load (%.1f%%)",
			m.config.MaxOpenConns, newMaxOpen, loadPercent)
	} else if loadPercent < float64(m.config.LowLoadThreshold) {
		newMaxOpen := int(float64(m.config.MaxOpenConns) * 0.8)
		if newMaxOpen < m.config.MinIdleConns {
			newMaxOpen = m.config.MinIdleConns
		}

		if newMaxOpen < m.config.MaxOpenConns {
			sqlDB.SetMaxOpenConns(newMaxOpen)
			log.Printf("Auto-tuning: decreased max connections from %d to %d due to low load (%.1f%%)",
				m.config.MaxOpenConns, newMaxOpen, loadPercent)
		}
	}
}

func (m *ConnectionPoolManager) GetStats() *ConnectionPoolStats {
	return m.stats
}

func (m *ConnectionPoolManager) AdjustPoolSize(maxOpen, maxIdle, minIdle int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sqlDB, err := m.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get DB: %w", err)
	}

	if maxOpen > 0 {
		sqlDB.SetMaxOpenConns(maxOpen)
	}

	if maxIdle > 0 {
		sqlDB.SetMaxIdleConns(maxIdle)
	}

	if minIdle > 0 {
		log.Printf("Adjusting min idle connections to %d", minIdle)
	}

	m.config.MaxOpenConns = maxOpen
	m.config.MaxIdleConns = maxIdle
	m.config.MinIdleConns = minIdle

	log.Printf("Pool size adjusted: max_open=%d, max_idle=%d, min_idle=%d", maxOpen, maxIdle, minIdle)
	return nil
}

func (m *ConnectionPoolManager) ForceCloseIdleConnections() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sqlDB, err := m.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close connections: %w", err)
	}

	log.Println("All idle connections closed")
	return nil
}

func (m *ConnectionPoolManager) Reconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sqlDB, err := m.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		log.Printf("Warning: error closing connections: %v", err)
	}

	time.Sleep(time.Second)

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}

	log.Println("Database reconnected successfully")
	return nil
}
