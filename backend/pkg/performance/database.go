package performance

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
)

type DatabaseOptimizer struct {
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	isRunning    bool
	queryCache   *QueryCache
	connectionMgr *ConnectionManager
	stats        *DatabaseStats
}

type DatabaseStats struct {
	TotalQueries      atomic.Int64
	CachedQueries     atomic.Int64
	CacheHits         atomic.Int64
	CacheMisses       atomic.Int64
	SlowQueries       atomic.Int64
	ActiveConnections atomic.Int64
	IdleConnections   atomic.Int64
	WaitCount         atomic.Int64
	TotalLatency      atomic.Int64
	MaxLatency        atomic.Int64
}

type QueryCache struct {
	mu         sync.RWMutex
	cache      map[string]*CachedQuery
	maxSize    int
	ttl        time.Duration
	hits       atomic.Int64
	misses     atomic.Int64
}

type CachedQuery struct {
	key        string
	data       interface{}
	createdAt  time.Time
	accessedAt time.Time
	hitCount   int
	latency    time.Duration
}

type ConnectionManager struct {
	mu            sync.RWMutex
	maxOpen       int
	maxIdle       int
	minIdle       int
	maxLifetime   time.Duration
	idleTimeout   time.Duration
	autoTune      bool
	targetPool    int
}

func NewDatabaseOptimizer() *DatabaseOptimizer {
	ctx, cancel := context.WithCancel(context.Background())
	return &DatabaseOptimizer{
		ctx:          ctx,
		cancel:       cancel,
		queryCache:   NewQueryCache(10000, 5*time.Minute),
		connectionMgr: NewConnectionManager(),
		stats:        &DatabaseStats{},
	}
}

func NewQueryCache(maxSize int, ttl time.Duration) *QueryCache {
	return &QueryCache{
		cache:    make(map[string]*CachedQuery, maxSize),
		maxSize:  maxSize,
		ttl:      ttl,
	}
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		maxOpen:     500,
		maxIdle:     100,
		minIdle:     20,
		maxLifetime: 30 * time.Minute,
		idleTimeout: 10 * time.Minute,
		autoTune:    true,
		targetPool:  100,
	}
}

func (d *DatabaseOptimizer) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.isRunning {
		return nil
	}

	d.isRunning = true

	d.applyConnectionSettings()

	go d.cleanupCache()
	go d.monitorConnections()
	go d.autoTuneConnections()

	log.Println("[DatabaseOptimizer] Started successfully")
	return nil
}

func (d *DatabaseOptimizer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isRunning {
		return
	}

	d.cancel()
	d.isRunning = false

	log.Println("[DatabaseOptimizer] Stopped")
}

func (d *DatabaseOptimizer) applyConnectionSettings() {
	if database.DB == nil {
		return
	}

	sqlDB, err := database.DB.DB()
	if err != nil {
		log.Printf("[DatabaseOptimizer] Failed to get SQL DB: %v", err)
		return
	}

	sqlDB.SetMaxOpenConns(d.connectionMgr.maxOpen)
	sqlDB.SetMaxIdleConns(d.connectionMgr.maxIdle)
	sqlDB.SetConnMaxLifetime(d.connectionMgr.maxLifetime)
	sqlDB.SetConnMaxIdleTime(d.connectionMgr.idleTimeout)

	log.Printf("[DatabaseOptimizer] Applied connection settings: maxOpen=%d, maxIdle=%d", 
		d.connectionMgr.maxOpen, d.connectionMgr.maxIdle)
}

func (d *DatabaseOptimizer) cleanupCache() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.queryCache.cleanup()
		}
	}
}

func (d *DatabaseOptimizer) monitorConnections() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.collectConnectionStats()
		}
	}
}

func (d *DatabaseOptimizer) collectConnectionStats() {
	if database.DB == nil {
		return
	}

	sqlDB, err := database.DB.DB()
	if err != nil {
		return
	}

	stats := sqlDB.Stats()

	d.stats.ActiveConnections.Store(int64(stats.InUse))
	d.stats.IdleConnections.Store(int64(stats.Idle))
	d.stats.WaitCount.Store(stats.WaitCount)
}

func (d *DatabaseOptimizer) autoTuneConnections() {
	if !d.connectionMgr.autoTune {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.optimizeConnections()
		}
	}
}

func (d *DatabaseOptimizer) optimizeConnections() {
	active := d.stats.ActiveConnections.Load()
	idle := d.stats.IdleConnections.Load()
	waitCount := d.stats.WaitCount.Load()

	// total := active + idle // Unused variable
	usageRatio := float64(active) / float64(d.connectionMgr.maxOpen)

	d.mu.Lock()
	defer d.mu.Unlock()

	if usageRatio > 0.85 || waitCount > 100 {
		newMaxOpen := int(float64(d.connectionMgr.maxOpen) * 1.2)
		if newMaxOpen > 1000 {
			newMaxOpen = 1000
		}

		newMaxIdle := newMaxOpen / 5
		if newMaxIdle < d.connectionMgr.minIdle {
			newMaxIdle = d.connectionMgr.minIdle
		}

		d.connectionMgr.maxOpen = newMaxOpen
		d.connectionMgr.maxIdle = newMaxIdle
		d.applyConnectionSettings()

		log.Printf("[DatabaseOptimizer] Scaled up connections: maxOpen=%d, maxIdle=%d", 
			newMaxOpen, newMaxIdle)
	} else if usageRatio < 0.3 && idle > int64(d.connectionMgr.minIdle)*2 {
		newMaxOpen := d.connectionMgr.maxOpen * 4 / 5
		if newMaxOpen < 100 {
			newMaxOpen = 100
		}

		newMaxIdle := newMaxOpen / 5
		if newMaxIdle < d.connectionMgr.minIdle {
			newMaxIdle = d.connectionMgr.minIdle
		}

		d.connectionMgr.maxOpen = newMaxOpen
		d.connectionMgr.maxIdle = newMaxIdle
		d.applyConnectionSettings()

		log.Printf("[DatabaseOptimizer] Scaled down connections: maxOpen=%d, maxIdle=%d", 
			newMaxOpen, newMaxIdle)
	}
}

func (q *QueryCache) Get(key string) (interface{}, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	item, exists := q.cache[key]
	if !exists {
		q.misses.Add(1)
		return nil, false
	}

	if time.Since(item.createdAt) > q.ttl {
		delete(q.cache, key)
		q.misses.Add(1)
		return nil, false
	}

	item.accessedAt = time.Now()
	item.hitCount++
	q.hits.Add(1)

	return item.data, true
}

func (q *QueryCache) Set(key string, data interface{}, latency time.Duration) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.cache) >= q.maxSize {
		q.evict()
	}

	q.cache[key] = &CachedQuery{
		key:        key,
		data:       data,
		createdAt:  time.Now(),
		accessedAt: time.Now(),
		latency:    latency,
		hitCount:   0,
	}
}

func (q *QueryCache) evict() {
	var oldestKey string
	var oldestTime time.Time

	for k, v := range q.cache {
		if oldestKey == "" || v.accessedAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.accessedAt
		}
	}

	if oldestKey != "" {
		delete(q.cache, oldestKey)
	}
}

func (q *QueryCache) cleanup() {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	for k, v := range q.cache {
		if now.Sub(v.createdAt) > q.ttl {
			delete(q.cache, k)
		}
	}
}

func (q *QueryCache) GetStats() map[string]interface{} {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return map[string]interface{}{
		"size":   len(q.cache),
		"max":    q.maxSize,
		"hits":   q.hits.Load(),
		"misses": q.misses.Load(),
	}
}

func (d *DatabaseOptimizer) ExecuteCached(key string, queryFunc func() (interface{}, error)) (interface{}, error) {
	d.stats.TotalQueries.Add(1)

	if data, ok := d.queryCache.Get(key); ok {
		d.stats.CacheHits.Add(1)
		d.stats.CachedQueries.Add(1)
		return data, nil
	}

	d.stats.CacheMisses.Add(1)

	start := time.Now()
	data, err := queryFunc()
	latency := time.Since(start)

	d.stats.TotalLatency.Add(latency.Nanoseconds())

	if latency.Nanoseconds() > d.stats.MaxLatency.Load() {
		d.stats.MaxLatency.Store(latency.Nanoseconds())
	}

	if latency > 100*time.Millisecond {
		d.stats.SlowQueries.Add(1)
	}

	if err == nil && data != nil {
		d.queryCache.Set(key, data, latency)
	}

	return data, err
}

func (d *DatabaseOptimizer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_queries":      d.stats.TotalQueries.Load(),
		"cached_queries":     d.stats.CachedQueries.Load(),
		"cache_hits":         d.stats.CacheHits.Load(),
		"cache_misses":       d.stats.CacheMisses.Load(),
		"slow_queries":       d.stats.SlowQueries.Load(),
		"active_connections": d.stats.ActiveConnections.Load(),
		"idle_connections":   d.stats.IdleConnections.Load(),
		"wait_count":         d.stats.WaitCount.Load(),
		"max_latency_ns":     d.stats.MaxLatency.Load(),
		"cache":              d.queryCache.GetStats(),
	}
}

func (d *DatabaseOptimizer) SetConnectionPool(maxOpen, maxIdle int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.connectionMgr.maxOpen = maxOpen
	d.connectionMgr.maxIdle = maxIdle
	d.applyConnectionSettings()
}

func (d *DatabaseOptimizer) EnableAutoTune(enable bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.connectionMgr.autoTune = enable
}
