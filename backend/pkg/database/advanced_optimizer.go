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

type PreparedQueryCache struct {
	mu      sync.RWMutex
	stmts   map[string]*gorm.DB
	maxSize int
	hits    atomic.Int64
	misses  atomic.Int64
}

func NewPreparedQueryCache(maxSize int) *PreparedQueryCache {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &PreparedQueryCache{
		stmts:   make(map[string]*gorm.DB),
		maxSize: maxSize,
	}
}

func (c *PreparedQueryCache) Get(key string) (*gorm.DB, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stmt, exists := c.stmts[key]
	if exists {
		c.hits.Add(1)
	}
	return stmt, exists
}

func (c *PreparedQueryCache) Set(key string, stmt *gorm.DB) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.stmts) >= c.maxSize {
		c.evictOldest()
	}

	c.stmts[key] = stmt
}

func (c *PreparedQueryCache) evictOldest() {
	if len(c.stmts) > 0 {
		for key := range c.stmts {
			delete(c.stmts, key)
			return
		}
	}
}

func (c *PreparedQueryCache) GetStats() map[string]interface{} {
	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"size":      len(c.stmts),
		"max_size":  c.maxSize,
		"hits":      hits,
		"misses":    misses,
		"hit_rate":  hitRate,
	}
}

type AdvancedQueryOptimizer struct {
	db                  *gorm.DB
	preparedStmts       *PreparedQueryCache
	slowQueryThreshold  time.Duration
	enableQueryAnalysis bool
	mu                  sync.RWMutex
	queryPatterns       map[string]*QueryPatternInfo
}

type QueryPatternInfo struct {
	Query         string
	ExecutionCount int64
	TotalDuration time.Duration
	AvgDuration   time.Duration
	LastExecuted  time.Time
	SuggestedIndex string
}

func NewAdvancedQueryOptimizer(db *gorm.DB, threshold time.Duration) *AdvancedQueryOptimizer {
	return &AdvancedQueryOptimizer{
		db:                  db,
		preparedStmts:      NewPreparedQueryCache(100),
		slowQueryThreshold:  threshold,
		enableQueryAnalysis: true,
		queryPatterns:      make(map[string]*QueryPatternInfo),
	}
}

func (qo *AdvancedQueryOptimizer) ExecuteWithCache(ctx context.Context, key string, queryFunc func() error) error {
	if _, exists := qo.preparedStmts.Get(key); exists {
		return queryFunc()
	}

	err := queryFunc()
	if err == nil {
		qo.preparedStmts.Set(key, nil)
	}

	return err
}

func (qo *AdvancedQueryOptimizer) RecordQuery(query string, duration time.Duration) {
	qo.mu.Lock()
	defer qo.mu.Unlock()

	info, exists := qo.queryPatterns[query]
	if !exists {
		info = &QueryPatternInfo{Query: query}
		qo.queryPatterns[query] = info
	}

	info.ExecutionCount++
	info.TotalDuration += duration
	info.AvgDuration = info.TotalDuration / time.Duration(info.ExecutionCount)
	info.LastExecuted = time.Now()

	if duration > qo.slowQueryThreshold && info.SuggestedIndex == "" {
		info.SuggestedIndex = qo.analyzeQuery(query)
	}
}

func (qo *AdvancedQueryOptimizer) analyzeQuery(query string) string {
	return ""
}

func (qo *AdvancedQueryOptimizer) GetHotQueries(limit int) []*QueryPatternInfo {
	qo.mu.RLock()
	defer qo.mu.RUnlock()

	type hotQuery struct {
		info     *QueryPatternInfo
		duration time.Duration
	}

	var hotQueries []hotQuery
	for _, info := range qo.queryPatterns {
		if info.ExecutionCount > 10 {
			hotQueries = append(hotQueries, hotQuery{info: info, duration: info.AvgDuration})
		}
	}

	for i := 0; i < len(hotQueries)-1; i++ {
		for j := i + 1; j < len(hotQueries); j++ {
			if hotQueries[j].duration > hotQueries[i].duration {
				hotQueries[i], hotQueries[j] = hotQueries[j], hotQueries[i]
			}
		}
	}

	if limit > len(hotQueries) {
		limit = len(hotQueries)
	}

	result := make([]*QueryPatternInfo, limit)
	for i := 0; i < limit; i++ {
		result[i] = hotQueries[i].info
	}

	return result
}

type ConnectionPoolAdvanced struct {
	db                 *gorm.DB
	config             *config.Config
	autoTuningEnabled bool
	stats             *PoolAdvancedStats
	mu                sync.RWMutex
}

type PoolAdvancedStats struct {
	MaxOpenConnections int
	OpenConnections    int
	InUse              int
	Idle               int
	WaitCount          int64
	WaitDuration       time.Duration
	Sets               atomic.Int64
	Gets                atomic.Int64
	Hits               atomic.Int64
	Misses             atomic.Int64
	LastReset         time.Time
}

func NewConnectionPoolAdvanced(db *gorm.DB, cfg *config.Config) *ConnectionPoolAdvanced {
	sqlDB, _ := db.DB()
	stats := sqlDB.Stats()

	return &ConnectionPoolAdvanced{
		db:     db,
		config: cfg,
		autoTuningEnabled: true,
		stats: &PoolAdvancedStats{
			MaxOpenConnections: stats.MaxOpenConnections,
			OpenConnections:    stats.OpenConnections,
			InUse:              stats.InUse,
			Idle:               stats.Idle,
			WaitCount:          stats.WaitCount,
			WaitDuration:       stats.WaitDuration,
			LastReset:         time.Now(),
		},
	}
}

func (cp *ConnectionPoolAdvanced) StartAutoTuning(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if cp.autoTuningEnabled {
				cp.analyzeAndTune()
			}
		}
	}()
}

func (cp *ConnectionPoolAdvanced) analyzeAndTune() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	sqlDB, err := cp.db.DB()
	if err != nil {
		return
	}

	stats := sqlDB.Stats()

	cp.stats.MaxOpenConnections = stats.MaxOpenConnections
	cp.stats.OpenConnections = stats.OpenConnections
	cp.stats.InUse = stats.InUse
	cp.stats.Idle = stats.Idle

	usageRatio := float64(stats.InUse) / float64(stats.MaxOpenConnections)

	if usageRatio > 0.9 && stats.WaitCount > 50 {
		newMaxOpen := int(float64(stats.MaxOpenConnections) * 1.2)
		sqlDB.SetMaxOpenConns(newMaxOpen)
		log.Printf("[POOL_TUNING] Increased max connections to %d due to high usage", newMaxOpen)
	} else if usageRatio < 0.3 && stats.Idle > 10 {
		newMaxIdle := stats.Idle / 2
		if newMaxIdle < 5 {
			newMaxIdle = 5
		}
		sqlDB.SetMaxIdleConns(newMaxIdle)
		log.Printf("[POOL_TUNING] Reduced idle connections to %d due to low usage", newMaxIdle)
	}

	if stats.WaitCount > 100 {
		log.Printf("[POOL_ALERT] High wait count: %d, wait duration: %v", stats.WaitCount, stats.WaitDuration)
	}
}

func (cp *ConnectionPoolAdvanced) GetStats() *PoolAdvancedStats {
	sqlDB, err := cp.db.DB()
	if err != nil {
		return cp.stats
	}

	stats := sqlDB.Stats()
	cp.stats.MaxOpenConnections = stats.MaxOpenConnections
	cp.stats.OpenConnections = stats.OpenConnections
	cp.stats.InUse = stats.InUse
	cp.stats.Idle = stats.Idle

	return cp.stats
}

func (cp *ConnectionPoolAdvanced) EnableAutoTuning(enabled bool) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.autoTuningEnabled = enabled
}

type PreparedQueryExecutor struct {
	db      *gorm.DB
	cache   *PreparedQueryCache
	timeout time.Duration
}

func NewPreparedQueryExecutor(db *gorm.DB) *PreparedQueryExecutor {
	return &PreparedQueryExecutor{
		db:      db,
		cache:   NewPreparedQueryCache(100),
		timeout: 30 * time.Second,
	}
}

func (qo *AdvancedQueryOptimizer) OptimizeAll() error {
	return nil
}

func (p *PreparedQueryExecutor) ExecutePrepared(ctx context.Context, query string, args ...interface{}) *gorm.DB {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	key := fmt.Sprintf("%s:%v", query, args)
	if _, exists := p.cache.Get(key); exists {
		return p.db.WithContext(ctx).Raw(query, args...)
	}

	result := p.db.WithContext(ctx).Raw(query, args...)
	p.cache.Set(key, nil)

	return result
}

type QueryCacheManager struct {
	cache      *QueryCache
	optimizer  *QueryOptimizer
	expiration time.Duration
}

func NewQueryCacheManager(cfg *config.Config) *QueryCacheManager {
	ttl := 5 * time.Minute
	if cfg != nil {
		ttl = time.Duration(cfg.Database.QueryOptimization.QueryCacheTTLSecs) * time.Second
	}

	return &QueryCacheManager{
		cache:      GetQueryCache(),
		optimizer:  nil,
		expiration: ttl,
	}
}

func (qcm *QueryCacheManager) GetOrExecute(ctx context.Context, key string, queryFunc func() (interface{}, error)) (interface{}, error) {
	if cached, ok := qcm.cache.Get(key); ok {
		return cached, nil
	}

	result, err := queryFunc()
	if err != nil {
		return nil, err
	}

	qcm.cache.Set(key, result)
	return result, nil
}

type DatabasePerformanceMonitor struct {
	queries     map[string]*QueryMetrics
	mu          sync.RWMutex
	enabled     bool
	sampleRate  float64
}

type QueryMetrics struct {
	Count        int64
	TotalLatency time.Duration
	AvgLatency   time.Duration
	P50Latency   time.Duration
	P95Latency   time.Duration
	P99Latency   time.Duration
	Errors       int64
}

func NewDatabasePerformanceMonitor() *DatabasePerformanceMonitor {
	return &DatabasePerformanceMonitor{
		queries:    make(map[string]*QueryMetrics),
		enabled:    true,
		sampleRate: 1.0,
	}
}

func (m *DatabasePerformanceMonitor) RecordQuery(query string, latency time.Duration, err error) {
	if !m.enabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	metrics, exists := m.queries[query]
	if !exists {
		metrics = &QueryMetrics{}
		m.queries[query] = metrics
	}

	metrics.Count++
	metrics.TotalLatency += latency
	metrics.AvgLatency = metrics.TotalLatency / time.Duration(metrics.Count)

	if err != nil {
		metrics.Errors++
	}
}

func (m *DatabasePerformanceMonitor) GetMetrics(query string) *QueryMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if metrics, exists := m.queries[query]; exists {
		return metrics
	}

	return nil
}

func (m *DatabasePerformanceMonitor) GetAllMetrics() map[string]*QueryMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*QueryMetrics)
	for k, v := range m.queries {
		result[k] = v
	}

	return result
}

func (m *DatabasePerformanceMonitor) Enable(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
}

func (m *DatabasePerformanceMonitor) SetSampleRate(rate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sampleRate = rate
}
