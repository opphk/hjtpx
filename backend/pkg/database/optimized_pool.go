package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

type OptimizedConnectionPool struct {
	db                    *sql.DB
	config                *PoolSettings
	healthChecker         *HealthChecker
	pressureMonitor        *PressureMeter
	metrics               *PoolMetrics
	isRunning              bool
	stopChan              chan struct{}
	lastScaleTime         time.Time
	mu                    sync.RWMutex
}

type PoolSettings struct {
	MaxOpenConns          int
	MaxIdleConns          int
	MinIdleConns          int
	ConnMaxLifetime       time.Duration
	ConnMaxIdleTime       time.Duration
	HealthCheckInterval   time.Duration
	AutoTuning            bool
	HighLoadThreshold     int
	LowLoadThreshold     int
	WarmupEnabled         bool
	WarmupConns          int
	PressureThreshold     float64
	MaxScaleFactor        float64
}

type HealthChecker struct {
	mu              sync.RWMutex
	lastCheck       *HealthResult
	checkHistory    []HealthResult
}

type HealthResult struct {
	Timestamp    time.Time
	IsHealthy    bool
	ResponseTime time.Duration
	ActiveConns  int
	IdleConns    int
	InUse        int
	Issues       []string
}

type PressureMeter struct {
	mu               sync.RWMutex
	currentPressure  float64
	pressureHistory []PressureSnapshot
	maxHistory      int
}

type PressureSnapshot struct {
	Timestamp   time.Time
	Pressure    float64
	Level       string
	ActiveConns int
}

type PoolMetrics struct {
	TotalConnections  int64
	ActiveConnections int64
	IdleConnections   int64
	InUseConnections  int64
	WaitCount        int64
	ReuseRate        float64
	HealthScore      int
}

var optimizedPool *OptimizedConnectionPool

func NewOptimizedConnectionPool(db *sql.DB, cfg *config.Config) *OptimizedConnectionPool {
	settings := &PoolSettings{
		MaxOpenConns:        cfg.Database.ConnectionPool.MaxOpenConns,
		MaxIdleConns:        cfg.Database.ConnectionPool.MaxIdleConns,
		MinIdleConns:        cfg.Database.ConnectionPool.MinIdleConns,
		ConnMaxLifetime:     time.Duration(cfg.Database.ConnectionPool.ConnMaxLifetimeSecs) * time.Second,
		ConnMaxIdleTime:     time.Duration(cfg.Database.ConnectionPool.ConnMaxIdleTimeSecs) * time.Second,
		HealthCheckInterval: time.Duration(cfg.Database.ConnectionPool.HealthCheckInterval) * time.Second,
		AutoTuning:         cfg.Database.ConnectionPool.EnableAutoTuning,
		HighLoadThreshold:   cfg.Database.ConnectionPool.HighLoadThreshold,
		LowLoadThreshold:   cfg.Database.ConnectionPool.LowLoadThreshold,
		WarmupEnabled:      cfg.Database.ConnectionPool.EnableWarmup,
		WarmupConns:       cfg.Database.ConnectionPool.WarmupConns,
		PressureThreshold:   0.8,
		MaxScaleFactor:     2.0,
	}

	if settings.MaxOpenConns <= 0 {
		settings.MaxOpenConns = 100
	}
	if settings.MaxIdleConns <= 0 {
		settings.MaxIdleConns = 20
	}
	if settings.MinIdleConns <= 0 {
		settings.MinIdleConns = 5
	}
	if settings.HealthCheckInterval <= 0 {
		settings.HealthCheckInterval = 30 * time.Second
	}

	pool := &OptimizedConnectionPool{
		db:           db,
		config:       settings,
		healthChecker: &HealthChecker{
			checkHistory: make([]HealthResult, 0),
		},
		pressureMonitor: &PressureMeter{
			pressureHistory: make([]PressureSnapshot, 0),
			maxHistory:     60,
		},
		metrics:   &PoolMetrics{},
		stopChan: make(chan struct{}),
	}

	db.SetMaxOpenConns(settings.MaxOpenConns)
	db.SetMaxIdleConns(settings.MaxIdleConns)
	db.SetConnMaxLifetime(settings.ConnMaxLifetime)
	db.SetConnMaxIdleTime(settings.ConnMaxIdleTime)

	return pool
}

func (p *OptimizedConnectionPool) Start() {
	p.mu.Lock()
	if p.isRunning {
		p.mu.Unlock()
		return
	}
	p.isRunning = true
	p.mu.Unlock()

	if p.config.WarmupEnabled {
		go p.warmupConnections()
	}

	go p.runHealthCheck()
	go p.monitorPressure()
	go p.autoTune()

	log.Println("Optimized connection pool started")
}

func (p *OptimizedConnectionPool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return
	}

	close(p.stopChan)
	p.isRunning = false
	log.Println("Optimized connection pool stopped")
}

func (p *OptimizedConnectionPool) warmupConnections() {
	log.Printf("Warming up connection pool with %d connections", p.config.WarmupConns)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for i := 0; i < p.config.WarmupConns; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			conn, err := p.db.Conn(ctx)
			if err != nil {
				return
			}

			var result int
			conn.PingContext(ctx)
			conn.QueryRowContext(ctx, "SELECT 1").Scan(&result)
			conn.Close()
		}()
	}

	wg.Wait()
	log.Println("Connection pool warmup completed")
}

func (p *OptimizedConnectionPool) runHealthCheck() {
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.performHealthCheck()
		}
	}
}

func (p *OptimizedConnectionPool) performHealthCheck() {
	result := &HealthResult{
		Timestamp: time.Now(),
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.db.PingContext(ctx); err != nil {
		result.IsHealthy = false
		result.Issues = append(result.Issues, fmt.Sprintf("Ping failed: %v", err))
	} else {
		result.ResponseTime = time.Since(start)
		result.IsHealthy = true
	}

	stats := p.db.Stats()
	result.ActiveConns = stats.OpenConnections
	result.IdleConns = stats.Idle
	result.InUse = stats.InUse

	usagePercent := float64(stats.InUse) / float64(stats.MaxOpenConnections) * 100
	if usagePercent > 90 {
		result.Issues = append(result.Issues, fmt.Sprintf("High usage: %.1f%%", usagePercent))
	}

	p.healthChecker.mu.Lock()
	p.healthChecker.lastCheck = result
	p.healthChecker.checkHistory = append(p.healthChecker.checkHistory, *result)
	if len(p.healthChecker.checkHistory) > 100 {
		p.healthChecker.checkHistory = p.healthChecker.checkHistory[1:]
	}
	p.healthChecker.mu.Unlock()
}

func (p *OptimizedConnectionPool) monitorPressure() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.updatePressure()
		}
	}
}

func (p *OptimizedConnectionPool) updatePressure() {
	stats := p.db.Stats()

	usage := float64(stats.InUse) / float64(stats.MaxOpenConnections)
	waitImpact := float64(stats.WaitCount) / float64(p.config.MaxOpenConns)

	pressure := usage*0.7 + waitImpact*0.3

	level := "normal"
	if pressure > 0.9 {
		level = "critical"
	} else if pressure > 0.8 {
		level = "high"
	} else if pressure > 0.6 {
		level = "medium"
	}

	snapshot := PressureSnapshot{
		Timestamp:   time.Now(),
		Pressure:    pressure,
		Level:       level,
		ActiveConns: stats.InUse,
	}

	p.pressureMonitor.mu.Lock()
	p.pressureMonitor.currentPressure = pressure
	p.pressureMonitor.pressureHistory = append(p.pressureMonitor.pressureHistory, snapshot)
	if len(p.pressureMonitor.pressureHistory) > p.pressureMonitor.maxHistory {
		p.pressureMonitor.pressureHistory = p.pressureMonitor.pressureHistory[1:]
	}
	p.pressureMonitor.mu.Unlock()

	if pressure > p.config.PressureThreshold {
		p.handleHighPressure(pressure)
	}
}

func (p *OptimizedConnectionPool) handleHighPressure(pressure float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if time.Since(p.lastScaleTime) < 5*time.Minute {
		return
	}

	stats := p.db.Stats()
	usagePercent := float64(stats.InUse) / float64(stats.MaxOpenConnections) * 100

	if usagePercent > float64(p.config.HighLoadThreshold) {
		newMaxOpen := int(float64(p.config.MaxOpenConns) * 1.2)
		if newMaxOpen > p.config.MaxOpenConns*int(p.config.MaxScaleFactor) {
			newMaxOpen = p.config.MaxOpenConns * int(p.config.MaxScaleFactor)
		}

		newMaxIdle := newMaxOpen / 2
		if newMaxIdle < p.config.MinIdleConns {
			newMaxIdle = p.config.MinIdleConns
		}

		p.db.SetMaxOpenConns(newMaxOpen)
		p.db.SetMaxIdleConns(newMaxIdle)

		p.config.MaxOpenConns = newMaxOpen
		p.config.MaxIdleConns = newMaxIdle

		p.lastScaleTime = time.Now()

		log.Printf("[AUTO_TUNE] Pool scaled up: MaxOpen=%d, MaxIdle=%d (pressure=%.2f)", newMaxOpen, newMaxIdle, pressure)
	}
}

func (p *OptimizedConnectionPool) autoTune() {
	if !p.config.AutoTuning {
		return
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.performAutoTuning()
		}
	}
}

func (p *OptimizedConnectionPool) performAutoTuning() {
	p.mu.Lock()
	defer p.mu.Unlock()

	stats := p.db.Stats()
	usagePercent := float64(stats.InUse) / float64(stats.MaxOpenConnections) * 100

	if usagePercent < float64(p.config.LowLoadThreshold) && stats.Idle > p.config.MinIdleConns {
		newMaxIdle := stats.Idle - 2
		if newMaxIdle < p.config.MinIdleConns {
			newMaxIdle = p.config.MinIdleConns
		}

		p.db.SetMaxIdleConns(newMaxIdle)
		p.config.MaxIdleConns = newMaxIdle

		log.Printf("[AUTO_TUNE] Reduced idle connections to %d (usage=%.1f%%)", newMaxIdle, usagePercent)
	}
}

func (p *OptimizedConnectionPool) GetHealth() *HealthResult {
	p.healthChecker.mu.RLock()
	defer p.healthChecker.mu.RUnlock()
	return p.healthChecker.lastCheck
}

func (p *OptimizedConnectionPool) GetMetrics() *PoolMetrics {
	stats := p.db.Stats()

	return &PoolMetrics{
		TotalConnections:  int64(stats.MaxOpenConnections),
		ActiveConnections: int64(stats.OpenConnections),
		IdleConnections:   int64(stats.Idle),
		InUseConnections:  int64(stats.InUse),
		WaitCount:        stats.WaitCount,
		ReuseRate:        float64(stats.InUse) / float64(stats.InUse+stats.Idle) * 100,
	}
}

func (p *OptimizedConnectionPool) GetPressure() (float64, string) {
	p.pressureMonitor.mu.RLock()
	defer p.pressureMonitor.mu.RUnlock()

	level := "normal"
	pressure := p.pressureMonitor.currentPressure
	if pressure > 0.9 {
		level = "critical"
	} else if pressure > 0.8 {
		level = "high"
	} else if pressure > 0.6 {
		level = "medium"
	}

	return pressure, level
}

func (p *OptimizedConnectionPool) UpdateSettings(newSettings *PoolSettings) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if newSettings.MaxOpenConns > 0 {
		p.db.SetMaxOpenConns(newSettings.MaxOpenConns)
		p.config.MaxOpenConns = newSettings.MaxOpenConns
	}

	if newSettings.MaxIdleConns > 0 {
		if newSettings.MaxIdleConns > newSettings.MaxOpenConns {
			return fmt.Errorf("MaxIdleConns cannot exceed MaxOpenConns")
		}
		p.db.SetMaxIdleConns(newSettings.MaxIdleConns)
		p.config.MaxIdleConns = newSettings.MaxIdleConns
	}

	if newSettings.ConnMaxLifetime > 0 {
		p.db.SetConnMaxLifetime(newSettings.ConnMaxLifetime)
		p.config.ConnMaxLifetime = newSettings.ConnMaxLifetime
	}

	log.Printf("[POOL_CONFIG] Updated: MaxOpen=%d, MaxIdle=%d", p.config.MaxOpenConns, p.config.MaxIdleConns)
	return nil
}

func InitOptimizedConnectionPool(db *sql.DB, cfg *config.Config) {
	optimizedPool = NewOptimizedConnectionPool(db, cfg)
	optimizedPool.Start()
}

func GetOptimizedConnectionPool() *OptimizedConnectionPool {
	return optimizedPool
}
