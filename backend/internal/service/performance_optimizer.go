package service

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type PerformanceOptimizer struct {
	db                  *sql.DB
	redisClient         *redis.Client
	queryCache          *QueryCache
	connectionPoolStats *ConnectionPoolStats
	redisPoolStats      *RedisPoolStats
	mu                  sync.RWMutex
}

type QueryCache struct {
	mu       sync.RWMutex
	cache    map[string]*CacheEntry
	maxSize  int
	hits     int64
	misses   int64
}

type CacheEntry struct {
	Value      interface{}
	Expiration time.Time
	CreatedAt  time.Time
}

type ConnectionPoolStats struct {
	MaxOpenConns    int32
	OpenConns       int32
	InUse           int32
	Idle            int32
	WaitCount       int64
	WaitDuration    time.Duration
	MaxIdleClosed   int64
	MaxLifetimeClosed int64
	LastUpdated     time.Time
}

type RedisPoolStats struct {
	TotalConns     uint32
	IdleConns      uint32
	StaleConns     uint32
	Hits           uint32
	Misses         uint32
	Timeouts       uint32
	LastUpdated    time.Time
}

type PerformanceMetrics struct {
	CPUUsage        float64
	MemoryUsage     uint64
	GoRoutines      int
	DBPoolStats     *ConnectionPoolStats
	RedisPoolStats  *RedisPoolStats
	QueryCacheHits  int64
	QueryCacheMisses int64
	CacheHitRate    float64
}

func NewPerformanceOptimizer(db *sql.DB) *PerformanceOptimizer {
	return &PerformanceOptimizer{
		db:                  db,
		queryCache:          NewQueryCache(1000),
		connectionPoolStats: &ConnectionPoolStats{},
		redisPoolStats:      &RedisPoolStats{},
	}
}

func NewQueryCache(maxSize int) *QueryCache {
	return &QueryCache{
		cache:   make(map[string]*CacheEntry),
		maxSize: maxSize,
	}
}

func (qc *QueryCache) Get(key string) (interface{}, bool) {
	qc.mu.RLock()
	entry, exists := qc.cache[key]
	qc.mu.RUnlock()

	if !exists {
		atomic.AddInt64(&qc.misses, 1)
		return nil, false
	}

	if time.Now().After(entry.Expiration) {
		qc.mu.Lock()
		delete(qc.cache, key)
		qc.mu.Unlock()
		atomic.AddInt64(&qc.misses, 1)
		return nil, false
	}

	atomic.AddInt64(&qc.hits, 1)
	return entry.Value, true
}

func (qc *QueryCache) Set(key string, value interface{}, ttl time.Duration) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	if len(qc.cache) >= qc.maxSize {
		qc.evictOldest()
	}

	qc.cache[key] = &CacheEntry{
		Value:      value,
		Expiration: time.Now().Add(ttl),
		CreatedAt:  time.Now(),
	}
}

func (qc *QueryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range qc.cache {
		if first || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
			first = false
		}
	}

	if oldestKey != "" {
		delete(qc.cache, oldestKey)
	}
}

func (qc *QueryCache) Delete(key string) {
	qc.mu.Lock()
	delete(qc.cache, key)
	qc.mu.Unlock()
}

func (qc *QueryCache) Clear() {
	qc.mu.Lock()
	qc.cache = make(map[string]*CacheEntry)
	qc.mu.Unlock()
}

func (qc *QueryCache) Stats() (hits, misses int64) {
	return atomic.LoadInt64(&qc.hits), atomic.LoadInt64(&qc.misses)
}

func (po *PerformanceOptimizer) UpdateDBPoolStats() error {
	if po.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	stats := po.db.Stats()
	atomic.StoreInt32(&po.connectionPoolStats.MaxOpenConns, int32(stats.MaxOpenConnections))
	atomic.StoreInt32(&po.connectionPoolStats.OpenConns, int32(stats.OpenConnections))
	atomic.StoreInt32(&po.connectionPoolStats.InUse, int32(stats.InUse))
	atomic.StoreInt32(&po.connectionPoolStats.Idle, int32(stats.Idle))
	atomic.StoreInt64(&po.connectionPoolStats.WaitCount, stats.WaitCount)
	atomic.StoreInt64(&po.connectionPoolStats.WaitDuration, stats.WaitDuration)
	atomic.StoreInt64(&po.connectionPoolStats.MaxIdleClosed, stats.MaxIdleClosed)
	atomic.StoreInt64(&po.connectionPoolStats.MaxLifetimeClosed, stats.MaxLifetimeClosed)
	po.connectionPoolStats.LastUpdated = time.Now()

	return nil
}

func (po *PerformanceOptimizer) UpdateRedisPoolStats() error {
	if redis.Client == nil {
		return fmt.Errorf("redis client is nil")
	}

	stats := redis.Client.PoolStats()
	atomic.StoreUint32(&po.redisPoolStats.TotalConns, stats.TotalConns)
	atomic.StoreUint32(&po.redisPoolStats.IdleConns, stats.IdleConns)
	atomic.StoreUint32(&po.redisPoolStats.StaleConns, stats.StaleConns)
	atomic.StoreUint32(&po.redisPoolStats.Hits, stats.Hits)
	atomic.StoreUint32(&po.redisPoolStats.Misses, stats.Misses)
	atomic.StoreUint32(&po.redisPoolStats.Timeouts, stats.Timeouts)
	po.redisPoolStats.LastUpdated = time.Now()

	return nil
}

func (po *PerformanceOptimizer) OptimizeDBPool(maxOpenConns, maxIdleConns int, connMaxLifetime, connMaxIdleTime time.Duration) error {
	if po.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	po.db.SetMaxOpenConns(maxOpenConns)
	po.db.SetMaxIdleConns(maxIdleConns)
	po.db.SetConnMaxLifetime(connMaxLifetime)
	po.db.SetConnMaxIdleTime(connMaxIdleTime)

	return po.UpdateDBPoolStats()
}

func (po *PerformanceOptimizer) OptimizeRedisPool(poolSize, minIdleConns int, poolTimeout time.Duration) error {
	if redis.Client == nil {
		return fmt.Errorf("redis client is nil")
	}

	redis.Client.ConfigReset(context.Background())

	return nil
}

func (po *PerformanceOptimizer) GetPerformanceMetrics() (*PerformanceMetrics, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	po.UpdateDBPoolStats()
	po.UpdateRedisPoolStats()

	hits, misses := po.queryCache.Stats()
	var hitRate float64
	if hits+misses > 0 {
		hitRate = float64(hits) / float64(hits+misses) * 100
	}

	return &PerformanceMetrics{
		CPUUsage:         getCPUUsage(),
		MemoryUsage:      m.Alloc,
		GoRoutines:       runtime.NumGoroutine(),
		DBPoolStats:      po.connectionPoolStats,
		RedisPoolStats:   po.redisPoolStats,
		QueryCacheHits:   hits,
		QueryCacheMisses: misses,
		CacheHitRate:     hitRate,
	}, nil
}

var cpuUsage float64
var lastCPUStats cpuStats

type cpuStats struct {
	idle   uint64
	total  uint64
}

func getCPUUsage() float64 {
	var stat runtime.Stat
	runtime.ReadStat(&stat)

	currentIdle := stat.Idle
	currentTotal := stat.User + stat.System + stat.Idle + stat.Nice + stat.IRQ + stat.SoftIRQ + stat.Steal

	idleDiff := currentIdle - lastCPUStats.idle
	totalDiff := currentTotal - lastCPUStats.total

	if totalDiff > 0 {
		cpuUsage = float64(idleDiff) / float64(totalDiff) * 100
	}

	lastCPUStats.idle = currentIdle
	lastCPUStats.total = currentTotal

	if cpuUsage < 0 {
		cpuUsage = 0
	}
	if cpuUsage > 100 {
		cpuUsage = 100
	}

	return 100 - cpuUsage
}

type CachedQuery struct {
	Key        string
	SQL        string
	Params     []interface{}
	Result     interface{}
	CacheTime  time.Time
	ExpireTime time.Time
}

type QueryCacheManager struct {
	mu      sync.RWMutex
	queries map[string]*CachedQuery
	maxAge  time.Duration
}

func NewQueryCacheManager(maxAge time.Duration) *QueryCacheManager {
	return &QueryCacheManager{
		queries: make(map[string]*CachedQuery),
		maxAge:  maxAge,
	}
}

func (qcm *QueryCacheManager) GetQuery(key string) (*CachedQuery, bool) {
	qcm.mu.RLock()
	defer qcm.mu.RUnlock()

	query, exists := qcm.queries[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(query.ExpireTime) {
		return nil, false
	}

	return query, true
}

func (qcm *QueryCacheManager) SetQuery(key string, query *CachedQuery) {
	qcm.mu.Lock()
	defer qcm.mu.Unlock()

	query.CacheTime = time.Now()
	query.ExpireTime = time.Now().Add(qcm.maxAge)
	qcm.queries[key] = query
}

func (qcm *QueryCacheManager) InvalidateQuery(key string) {
	qcm.mu.Lock()
	defer qcm.mu.Unlock()
	delete(qcm.queries, key)
}

func (qcm *QueryCacheManager) InvalidatePattern(pattern string) {
	qcm.mu.Lock()
	defer qcm.mu.Unlock()

	for key := range qcm.queries {
		if matchPattern(key, pattern) {
			delete(qcm.queries, key)
		}
	}
}

func (qcm *QueryCacheManager) Clear() {
	qcm.mu.Lock()
	defer qcm.mu.Unlock()
	qcm.queries = make(map[string]*CachedQuery)
}

func matchPattern(key, pattern string) bool {
	if pattern == "" {
		return true
	}
	if pattern == "*" {
		return true
	}
	return len(key) > 0 && key[:min(len(key), len(pattern))] == pattern
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type PreparedStatement struct {
	SQL     string
	stmt    *sql.Stmt
	created time.Time
}

type PreparedStatementCache struct {
	mu       sync.RWMutex
	stmts    map[string]*PreparedStatement
	maxSize  int
	hits     int64
	misses   int64
	evictions int64
}

func NewPreparedStatementCache(maxSize int) *PreparedStatementCache {
	return &PreparedStatementCache{
		stmts:   make(map[string]*PreparedStatement),
		maxSize: maxSize,
	}
}

func (psc *PreparedStatementCache) Get(db *sql.DB, sql string) (*sql.Stmt, error) {
	psc.mu.RLock()
	stmt, exists := psc.stmts[sql]
	psc.mu.RUnlock()

	if exists {
		atomic.AddInt64(&psc.hits, 1)
		return stmt.stmt, nil
	}

	atomic.AddInt64(&psc.misses, 1)

	prepared, err := db.Prepare(sql)
	if err != nil {
		return nil, err
	}

	psc.mu.Lock()
	defer psc.mu.Unlock()

	if len(psc.stmts) >= psc.maxSize {
		psc.evictOldest()
	}

	psc.stmts[sql] = &PreparedStatement{
		SQL:     sql,
		stmt:    prepared,
		created: time.Now(),
	}

	return prepared, nil
}

func (psc *PreparedStatementCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, stmt := range psc.stmts {
		if first || stmt.created.Before(oldestTime) {
			oldestKey = key
			oldestTime = stmt.created
			first = false
		}
	}

	if oldestKey != "" {
		if psc.stmts[oldestKey].stmt != nil {
			psc.stmts[oldestKey].stmt.Close()
		}
		delete(psc.stmts, oldestKey)
		atomic.AddInt64(&psc.evictions, 1)
	}
}

func (psc *PreparedStatementCache) Stats() (hits, misses, evictions int64) {
	return atomic.LoadInt64(&psc.hits), atomic.LoadInt64(&psc.misses), atomic.LoadInt64(&psc.evictions)
}

func (psc *PreparedStatementCache) Close() {
	psc.mu.Lock()
	defer psc.mu.Unlock()

	for _, stmt := range psc.stmts {
		if stmt.stmt != nil {
			stmt.stmt.Close()
		}
	}
	psc.stmts = make(map[string]*PreparedStatement)
}

type BatchOperation struct {
	operation string
	table     string
	data      []interface{}
	batchSize int
}

func NewBatchOperation(operation, table string, data []interface{}, batchSize int) *BatchOperation {
	return &BatchOperation{
		operation: operation,
		table:     table,
		data:      data,
		batchSize: batchSize,
	}
}

func (bo *BatchOperation) Execute(db *sql.DB) error {
	if len(bo.data) == 0 {
		return nil
	}

	for i := 0; i < len(bo.data); i += bo.batchSize {
		end := i + bo.batchSize
		if end > len(bo.data) {
			end = len(bo.data)
		}

		batch := bo.data[i:end]
		if err := bo.executeBatch(db, batch); err != nil {
			return err
		}
	}

	return nil
}

func (bo *BatchOperation) executeBatch(db *sql.DB, batch []interface{}) error {
	switch bo.operation {
	case "insert":
		return bo.batchInsert(db, batch)
	case "update":
		return bo.batchUpdate(db, batch)
	case "delete":
		return bo.batchDelete(db, batch)
	default:
		return fmt.Errorf("unsupported operation: %s", bo.operation)
	}
}

func (bo *BatchOperation) batchInsert(db *sql.DB, batch []interface{}) error {
	query := fmt.Sprintf("INSERT INTO %s VALUES %s", bo.table, bo.generatePlaceholders(len(batch)))
	_, err := db.Exec(query, batch...)
	return err
}

func (bo *BatchOperation) batchUpdate(db *sql.DB, batch []interface{}) error {
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", bo.table, bo.generateUpdatePlaceholders())
	_, err := db.Exec(query, batch...)
	return err
}

func (bo *BatchOperation) batchDelete(db *sql.DB, batch []interface{}) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id IN %s", bo.table, bo.generatePlaceholders(len(batch)))
	_, err := db.Exec(query, batch...)
	return err
}

func (bo *BatchOperation) generatePlaceholders(count int) string {
	if count <= 0 {
		return "()"
	}
	placeholders := make([]string, count)
	for i := range placeholders {
		placeholders[i] = "?"
	}
	return fmt.Sprintf("(%s)", joinStrings(placeholders, ", "))
}

func (bo *BatchOperation) generateUpdatePlaceholders() string {
	return "column = ?"
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
