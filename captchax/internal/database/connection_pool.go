package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ConnectionPoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func DefaultPoolConfig() ConnectionPoolConfig {
	return ConnectionPoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    20,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}
}

type DBConnection struct {
	db         *gorm.DB
	sqlDB      *sql.DB
	stats      PoolStats
	statsMu    sync.RWMutex
	leakDetector *LeakDetector
}

type PoolStats struct {
	MaxOpenConnections int           `json:"max_open_connections"`
	OpenConnections    int           `json:"open_connections"`
	InUse               int           `json:"in_use"`
	Idle                int           `json:"idle"`
	WaitCount           int64         `json:"wait_count"`
	WaitDuration        time.Duration `json:"wait_duration"`
	MaxIdleClosed       int64         `json:"max_idle_closed"`
	MaxLifetimeClosed   int64         `json:"max_lifetime_closed"`
	HealthCheckStatus   string        `json:"health_check_status"`
	LastHealthCheck     time.Time     `json:"last_health_check"`
}

type ConnectionMetadata struct {
	ID          int64
	CreatedAt   time.Time
	LastUsedAt  time.Time
	InUse       bool
	QueryCount  int64
	QueryTime   time.Duration
}

type LeakDetector struct {
	activeConns map[*sql.Conn]*ConnectionMetadata
	mu          sync.RWMutex
	threshold   time.Duration
	enabled     bool
}

func NewLeakDetector(threshold time.Duration) *LeakDetector {
	return &LeakDetector{
		activeConns: make(map[*sql.Conn]*ConnectionMetadata),
		threshold:   threshold,
		enabled:     true,
	}
}

func (ld *LeakDetector) Register(conn *sql.Conn) {
	if !ld.enabled {
		return
	}
	ld.mu.Lock()
	defer ld.mu.Unlock()
	ld.activeConns[conn] = &ConnectionMetadata{
		ID:         time.Now().UnixNano(),
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
		InUse:      true,
	}
}

func (ld *LeakDetector) Unregister(conn *sql.Conn) {
	if !ld.enabled {
		return
	}
	ld.mu.Lock()
	defer ld.mu.Unlock()
	delete(ld.activeConns, conn)
}

func (ld *LeakDetector) RecordUsage(conn *sql.Conn, duration time.Duration) {
	if !ld.enabled {
		return
	}
	ld.mu.Lock()
	defer ld.mu.Unlock()
	if meta, ok := ld.activeConns[conn]; ok {
		meta.LastUsedAt = time.Now()
		meta.QueryCount++
		meta.QueryTime += duration
	}
}

func (ld *LeakDetector) GetActiveConnections() []*ConnectionMetadata {
	ld.mu.RLock()
	defer ld.mu.RUnlock()
	metas := make([]*ConnectionMetadata, 0, len(ld.activeConns))
	for _, meta := range ld.activeConns {
		metas = append(metas, meta)
	}
	return metas
}

func (ld *LeakDetector) DetectLeaks() []*ConnectionMetadata {
	ld.mu.RLock()
	defer ld.mu.RUnlock()
	now := time.Now()
	var leaks []*ConnectionMetadata
	for conn, meta := range ld.activeConns {
		if meta.InUse && now.Sub(meta.LastUsedAt) > ld.threshold {
			leaks = append(leaks, &ConnectionMetadata{
				ID:         meta.ID,
				CreatedAt:  meta.CreatedAt,
				LastUsedAt: meta.LastUsedAt,
				InUse:      true,
				QueryCount: meta.QueryCount,
				QueryTime:  meta.QueryTime,
			})
			_ = conn
		}
	}
	return leaks
}

func (ld *LeakDetector) GetStats() (total int, leaking int) {
	ld.mu.RLock()
	defer ld.mu.RUnlock()
	total = len(ld.activeConns)
	now := time.Now()
	for _, meta := range ld.activeConns {
		if meta.InUse && now.Sub(meta.LastUsedAt) > ld.threshold {
			leaking++
		}
	}
	return
}

func NewConnectionPool(dsn string, config ConnectionPoolConfig) (*DBConnection, error) {
	if config.MaxOpenConns <= 0 {
		config.MaxOpenConns = 100
	}
	if config.MaxIdleConns <= 0 {
		config.MaxIdleConns = 20
	}
	if config.ConnMaxLifetime <= 0 {
		config.ConnMaxLifetime = 30 * time.Minute
	}
	if config.ConnMaxIdleTime <= 0 {
		config.ConnMaxIdleTime = 10 * time.Minute
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	leakDetector := NewLeakDetector(5 * time.Minute)

	conn := &DBConnection{
		db:           db,
		sqlDB:        sqlDB,
		leakDetector: leakDetector,
	}

	go conn.monitorConnections()

	return conn, nil
}

func (c *DBConnection) monitorConnections() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.checkConnectionHealth()
		c.detectConnectionLeaks()
	}
}

func (c *DBConnection) checkConnectionHealth() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	if err := c.sqlDB.PingContext(ctx); err != nil {
		c.stats.HealthCheckStatus = "unhealthy"
	} else {
		c.stats.HealthCheckStatus = "healthy"
	}
	c.stats.LastHealthCheck = time.Now()
}

func (c *DBConnection) detectConnectionLeaks() {
	leaks := c.leakDetector.DetectLeaks()
	if len(leaks) > 0 {
		fmt.Printf("[WARN] Detected %d potential connection leaks\n", len(leaks))
	}
}

func (c *DBConnection) HealthCheck(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return c.sqlDB.PingContext(pingCtx)
}

func (c *DBConnection) HealthCheckStatus() string {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	return c.stats.HealthCheckStatus
}

func (c *DBConnection) GetDB() *gorm.DB {
	return c.db
}

func (c *DBConnection) GetConn() *sql.DB {
	return c.sqlDB
}

func (c *DBConnection) GetStats() PoolStats {
	stats := c.sqlDB.Stats()

	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	c.stats.MaxOpenConnections = stats.MaxOpenConnections
	c.stats.OpenConnections = stats.OpenConnections
	c.stats.InUse = stats.InUse
	c.stats.Idle = stats.Idle
	c.stats.WaitCount = stats.WaitCount
	c.stats.WaitDuration = stats.WaitDuration
	c.stats.MaxIdleClosed = stats.MaxIdleClosed
	c.stats.MaxLifetimeClosed = stats.MaxLifetimeClosed

	return c.stats
}

func (c *DBConnection) GetLeakDetectorStats() (total int, leaking int) {
	return c.leakDetector.GetStats()
}

func (c *DBConnection) GetActiveConnections() []*ConnectionMetadata {
	return c.leakDetector.GetActiveConnections()
}

type TrackedConn struct {
	conn *sql.Conn
	c    *DBConnection
}

func (c *DBConnection) AcquireTrackedConn(ctx context.Context) (*TrackedConn, error) {
	conn, err := c.sqlDB.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}
	c.leakDetector.Register(conn)
	return &TrackedConn{conn: conn, c: c}, nil
}

func (tc *TrackedConn) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := tc.conn.ExecContext(ctx, query, args...)
	tc.c.leakDetector.RecordUsage(tc.conn, time.Since(start))
	return result, err
}

func (tc *TrackedConn) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := tc.conn.QueryContext(ctx, query, args...)
	tc.c.leakDetector.RecordUsage(tc.conn, time.Since(start))
	return rows, err
}

func (tc *TrackedConn) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := tc.conn.QueryRowContext(ctx, query, args...)
	tc.c.leakDetector.RecordUsage(tc.conn, time.Since(start))
	return row
}

func (tc *TrackedConn) Release() error {
	tc.c.leakDetector.Unregister(tc.conn)
	return tc.conn.Close()
}

func (c *DBConnection) Close() error {
	return c.sqlDB.Close()
}

func CreatePostgresDB(dsn string) (*sql.DB, error) {
	return sql.Open("postgres", dsn)
}

type ConnectionPoolMetrics struct {
	totalRequests    int64
	failedRequests   int64
	avgAcquireTime   time.Duration
	acquireCount     int64
	maxAcquireTime   time.Duration
	acquireTimes     []time.Duration
	acquireTimesMu   sync.Mutex
}

func NewConnectionPoolMetrics() *ConnectionPoolMetrics {
	return &ConnectionPoolMetrics{
		acquireTimes: make([]time.Duration, 0, 1000),
	}
}

func (m *ConnectionPoolMetrics) RecordAcquire(success bool, duration time.Duration) {
	atomic.AddInt64(&m.totalRequests, 1)
	if !success {
		atomic.AddInt64(&m.failedRequests, 1)
	}

	m.acquireTimesMu.Lock()
	m.acquireTimes = append(m.acquireTimes, duration)
	if len(m.acquireTimes) > 1000 {
		m.acquireTimes = m.acquireTimes[len(m.acquireTimes)-1000:]
	}

	if duration > m.maxAcquireTime {
		m.maxAcquireTime = duration
	}

	total := atomic.LoadInt64(&m.acquireCount)
	oldAvg := m.avgAcquireTime
	newCount := total + 1
	m.avgAcquireTime = time.Duration((int64(oldAvg)*total + int64(duration)) / newCount)
	atomic.StoreInt64(&m.acquireCount, newCount)
	m.acquireTimesMu.Unlock()
}

func (m *ConnectionPoolMetrics) GetStats() (total, failed int64, avgAcquire, maxAcquire time.Duration) {
	return atomic.LoadInt64(&m.totalRequests),
		atomic.LoadInt64(&m.failedRequests),
		m.avgAcquireTime,
		m.maxAcquireTime
}

type PoolManager struct {
	pools     map[string]*DBConnection
	mu        sync.RWMutex
	metrics   *ConnectionPoolMetrics
}

var defaultPoolManager *PoolManager
var poolManagerOnce sync.Once

func GetPoolManager() *PoolManager {
	poolManagerOnce.Do(func() {
		defaultPoolManager = &PoolManager{
			pools:   make(map[string]*DBConnection),
			metrics: NewConnectionPoolMetrics(),
		}
	})
	return defaultPoolManager
}

func (pm *PoolManager) RegisterPool(name string, pool *DBConnection) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.pools[name] = pool
}

func (pm *PoolManager) GetPool(name string) (*DBConnection, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	pool, ok := pm.pools[name]
	return pool, ok
}

func (pm *PoolManager) GetAllStats() map[string]PoolStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := make(map[string]PoolStats)
	for name, pool := range pm.pools {
		stats[name] = pool.GetStats()
	}
	return stats
}

func (pm *PoolManager) HealthCheckAll(ctx context.Context) map[string]bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	results := make(map[string]bool)
	for name, pool := range pm.pools {
		results[name] = pool.HealthCheck(ctx) == nil
	}
	return results
}
