package api_performance

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type QueryOptimizer struct {
	config       *QueryOptimizerConfig
	queryCache   *QueryCache
	stmtCache    *StatementCache
	connPool     *ConnectionPool
	indexAnalyzer *IndexAnalyzer
	mu           sync.RWMutex
	stats        *QueryOptimizerStats
}

type QueryOptimizerConfig struct {
	EnableQueryCache     bool
	EnableStmtCache      bool
	EnableIndexAnalysis  bool
	MaxQueryCacheSize    int
	MaxStmtCacheSize     int
	QueryTimeout         time.Duration
	EnableSlowQueryLog   bool
	SlowQueryThreshold   time.Duration
	EnableNPlusOneDetect bool
	EnableBatchQuery     bool
	BatchSize            int
	BatchTimeout         time.Duration
	RetryAttempts        int
	RetryDelay           time.Duration
}

var DefaultQueryOptimizerConfig = &QueryOptimizerConfig{
	EnableQueryCache:     true,
	EnableStmtCache:      true,
	EnableIndexAnalysis:  true,
	MaxQueryCacheSize:    1000,
	MaxStmtCacheSize:     500,
	QueryTimeout:         5 * time.Second,
	EnableSlowQueryLog:   true,
	SlowQueryThreshold:   100 * time.Millisecond,
	EnableNPlusOneDetect: true,
	EnableBatchQuery:     true,
	BatchSize:            100,
	BatchTimeout:         10 * time.Millisecond,
	RetryAttempts:        3,
	RetryDelay:           50 * time.Millisecond,
}

type QueryOptimizerStats struct {
	TotalQueries        atomic.Int64
	CacheHits           atomic.Int64
	CacheMisses         atomic.Int64
	StmtCacheHits       atomic.Int64
	StmtCacheMisses     atomic.Int64
	SlowQueries         atomic.Int64
	FailedQueries       atomic.Int64
	AvgQueryTime        atomic.Int64
	P50QueryTime        atomic.Int64
	P95QueryTime        atomic.Int64
	P99QueryTime        atomic.Int64
	NPlusOneDetected    atomic.Int64
	BatchQueries        atomic.Int64
	Retries             atomic.Int64
	QueryTimeouts       atomic.Int64
}

type QueryCache struct {
	cache    *sync.Map
	maxSize  int
	mu       sync.RWMutex
	hits     atomic.Int64
	misses   atomic.Int64
	evictions atomic.Int64
}

type QueryCacheEntry struct {
	Result    *sql.Rows
	Columns   []string
	Values    [][]interface{}
	ExpiresAt time.Time
	Query     string
	Args      []interface{}
}

type StatementCache struct {
	cache    *sync.Map
	maxSize  int
	mu       sync.RWMutex
	hits     atomic.Int64
	misses   atomic.Int64
	evictions atomic.Int64
}

type CachedStmt struct {
	Stmt     *sql.Stmt
	Query    string
	LastUsed time.Time
	UseCount int64
}

type ConnectionPool struct {
	db         *sql.DB
	maxOpen    int
	maxIdle    int
	mu         sync.RWMutex
	stats      *PoolStats
}

type PoolStats struct {
	ActiveConnections atomic.Int64
	IdleConnections   atomic.Int64
	TotalConnections  atomic.Int64
	WaitCount         atomic.Int64
	WaitDuration      atomic.Int64
	MaxIdleClosed     atomic.Int64
	MaxLifetimeClosed  atomic.Int64
}

type IndexAnalyzer struct {
	enabled  bool
	indexes  map[string][]*IndexInfo
	mu       sync.RWMutex
	suggestions chan *IndexSuggestion
}

type IndexInfo struct {
	TableName  string
	IndexName  string
	Columns    []string
	Unique     bool
	Cardinality int64
	UsageCount int64
	LastUsed   time.Time
}

type IndexSuggestion struct {
	TableName    string
	IndexName    string
	Columns      []string
	SuggestionType string
	Reason       string
	ImpactScore  float64
}

type BatchQuery struct {
	query    string
	args     [][]interface{}
	results  chan *BatchQueryResult
}

type BatchQueryResult struct {
	Index   int
	Values  []interface{}
	Error   error
}

type NPlusOneDetector struct {
	enabled       bool
	queryPatterns *sync.Map
	mu            sync.RWMutex
	threshold     int
}

func NewQueryOptimizer(config *QueryOptimizerConfig) *QueryOptimizer {
	if config == nil {
		config = DefaultQueryOptimizerConfig
	}

	return &QueryOptimizer{
		config:        config,
		queryCache:    NewQueryCache(config.MaxQueryCacheSize),
		stmtCache:     NewStatementCache(config.MaxStmtCacheSize),
		indexAnalyzer: NewIndexAnalyzer(config.EnableIndexAnalysis),
		stats:         &QueryOptimizerStats{},
	}
}

func NewQueryCache(maxSize int) *QueryCache {
	return &QueryCache{
		cache:   &sync.Map{},
		maxSize: maxSize,
	}
}

func (qc *QueryCache) Get(query string, args []interface{}) (*QueryCacheEntry, bool) {
	key := qc.generateKey(query, args)
	val, ok := qc.cache.Load(key)
	if !ok {
		qc.misses.Add(1)
		return nil, false
	}

	entry := val.(*QueryCacheEntry)
	if time.Now().After(entry.ExpiresAt) {
		qc.cache.Delete(key)
		qc.evictions.Add(1)
		qc.misses.Add(1)
		return nil, false
	}

	qc.hits.Add(1)
	return entry, true
}

func (qc *QueryCache) Set(query string, args []interface{}, entry *QueryCacheEntry) {
	if qc.getSize() >= qc.maxSize {
		qc.evictLRU()
	}

	key := qc.generateKey(query, args)
	qc.cache.Store(key, entry)
}

func (qc *QueryCache) generateKey(query string, args []interface{}) string {
	key := query
	for _, arg := range args {
		key += fmt.Sprintf(":%v", arg)
	}
	return key
}

func (qc *QueryCache) getSize() int {
	size := 0
	qc.cache.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}

func (qc *QueryCache) evictLRU() {
	var oldestKey string

	qc.cache.Range(func(key, value interface{}) bool {
		if oldestKey == "" {
			oldestKey = key.(string)
		}
		return true
	})

	if oldestKey != "" {
		qc.cache.Delete(oldestKey)
		qc.evictions.Add(1)
	}
}

func (qc *QueryCache) Clear() {
	qc.cache = &sync.Map{}
}

func NewStatementCache(maxSize int) *StatementCache {
	return &StatementCache{
		cache:   &sync.Map{},
		maxSize: maxSize,
	}
}

func (sc *StatementCache) Get(query string) (*CachedStmt, bool) {
	val, ok := sc.cache.Load(query)
	if !ok {
		sc.misses.Add(1)
		return nil, false
	}

	stmt := val.(*CachedStmt)
	stmt.LastUsed = time.Now()
	stmt.UseCount++
	sc.hits.Add(1)
	return stmt, true
}

func (sc *StatementCache) Set(query string, stmt *sql.Stmt) {
	if sc.getSize() >= sc.maxSize {
		sc.evictLRU()
	}

	sc.cache.Store(query, &CachedStmt{
		Stmt:     stmt,
		Query:    query,
		LastUsed: time.Now(),
		UseCount: 1,
	})
}

func (sc *StatementCache) Delete(query string) {
	sc.cache.Delete(query)
}

func (sc *StatementCache) Clear() {
	sc.cache.Range(func(key, value interface{}) bool {
		stmt := value.(*CachedStmt)
		stmt.Stmt.Close()
		return true
	})
	sc.cache = &sync.Map{}
}

func (sc *StatementCache) getSize() int {
	size := 0
	sc.cache.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}

func (sc *StatementCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	sc.cache.Range(func(key, value interface{}) bool {
		stmt := value.(*CachedStmt)
		if oldestKey == "" || stmt.LastUsed.Before(oldestTime) {
			oldestKey = key.(string)
			oldestTime = stmt.LastUsed
		}
		return true
	})

	if oldestKey != "" {
		if val, ok := sc.cache.Load(oldestKey); ok {
			stmt := val.(*CachedStmt)
			stmt.Stmt.Close()
		}
		sc.cache.Delete(oldestKey)
		sc.evictions.Add(1)
	}
}

func NewIndexAnalyzer(enabled bool) *IndexAnalyzer {
	return &IndexAnalyzer{
		enabled:    enabled,
		indexes:    make(map[string][]*IndexInfo),
		suggestions: make(chan *IndexSuggestion, 100),
	}
}

func (ia *IndexAnalyzer) RecordQuery(tableName string, query string, duration time.Duration) {
	if !ia.enabled {
		return
	}

	ia.mu.Lock()
	defer ia.mu.Unlock()

	indexes := ia.indexes[tableName]
	for _, idx := range indexes {
		if ia.isIndexUsed(query, idx.Columns) {
			idx.UsageCount++
			idx.LastUsed = time.Now()
		}
	}
}

func (ia *IndexAnalyzer) isIndexUsed(query string, columns []string) bool {
	for _, col := range columns {
		if contains(query, col) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (ia *IndexAnalyzer) AddIndex(tableName string, idx *IndexInfo) {
	ia.mu.Lock()
	defer ia.mu.Unlock()

	ia.indexes[tableName] = append(ia.indexes[tableName], idx)
}

func (ia *IndexAnalyzer) GetSuggestions() []*IndexSuggestion {
	ia.mu.RLock()
	defer ia.mu.RUnlock()

	var suggestions []*IndexSuggestion

	for tableName, indexes := range ia.indexes {
		for _, idx := range indexes {
			if idx.UsageCount == 0 {
				suggestions = append(suggestions, &IndexSuggestion{
					TableName:       tableName,
					IndexName:       idx.IndexName,
					Columns:         idx.Columns,
					SuggestionType: "DROP",
					Reason:          "Index not used",
					ImpactScore:     0.8,
				})
			}
		}
	}

	return suggestions
}

func (ia *IndexAnalyzer) GetStats() map[string]interface{} {
	ia.mu.RLock()
	defer ia.mu.RUnlock()

	totalIndexes := 0
	unusedIndexes := 0
	totalUsage := int64(0)

	for _, indexes := range ia.indexes {
		for _, idx := range indexes {
			totalIndexes++
			if idx.UsageCount == 0 {
				unusedIndexes++
			}
			totalUsage += idx.UsageCount
		}
	}

	return map[string]interface{}{
		"total_indexes":   totalIndexes,
		"unused_indexes":  unusedIndexes,
		"total_usage":     totalUsage,
		"suggestions_len": len(ia.suggestions),
	}
}

func (o *QueryOptimizer) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	o.stats.TotalQueries.Add(1)
	start := time.Now()

	if o.config.EnableQueryCache {
		if entry, ok := o.queryCache.Get(query, args); ok {
			o.stats.CacheHits.Add(1)
			return o.rowsToResult(entry), nil
		}
		o.stats.CacheMisses.Add(1)
	}

	ctx, cancel := context.WithTimeout(ctx, o.config.QueryTimeout)
	defer cancel()

	rows, err := o.queryWithRetry(ctx, query, args...)

	duration := time.Since(start)
	o.recordQueryDuration(duration)

	if o.config.EnableIndexAnalysis {
		o.indexAnalyzer.RecordQuery("", query, duration)
	}

	if o.config.EnableSlowQueryLog && duration > o.config.SlowQueryThreshold {
		o.stats.SlowQueries.Add(1)
	}

	if err != nil {
		o.stats.FailedQueries.Add(1)
		return nil, err
	}

	if o.config.EnableQueryCache {
		entry := &QueryCacheEntry{
			ExpiresAt: time.Now().Add(5 * time.Minute),
			Query:     query,
			Args:      args,
		}
		o.queryCache.Set(query, args, entry)
	}

	return rows, nil
}

func (o *QueryOptimizer) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	o.stats.TotalQueries.Add(1)
	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, o.config.QueryTimeout)
	defer cancel()

	row := o.queryRowWithRetry(ctx, query, args...)

	duration := time.Since(start)
	o.recordQueryDuration(duration)

	if o.config.EnableSlowQueryLog && duration > o.config.SlowQueryThreshold {
		o.stats.SlowQueries.Add(1)
	}

	return row
}

func (o *QueryOptimizer) queryWithRetry(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var lastErr error

	for i := 0; i < o.config.RetryAttempts; i++ {
		if i > 0 {
			time.Sleep(o.config.RetryDelay * time.Duration(i))
			o.stats.Retries.Add(1)
		}

		rows, err := o.executeQuery(ctx, query, args...)
		if err == nil {
			return rows, nil
		}

		lastErr = err

		if !isRetryableError(err) {
			break
		}
	}

	return nil, lastErr
}

func (o *QueryOptimizer) queryRowWithRetry(ctx context.Context, query string, args ...interface{}) *sql.Row {
	for i := 0; i < o.config.RetryAttempts; i++ {
		if i > 0 {
			time.Sleep(o.config.RetryDelay * time.Duration(i))
			o.stats.Retries.Add(1)
		}

		row := o.executeQueryRow(ctx, query, args...)
		return row
	}

	o.stats.FailedQueries.Add(1)
	return nil
}

func (o *QueryOptimizer) executeQuery(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (o *QueryOptimizer) executeQueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

func (o *QueryOptimizer) recordQueryDuration(duration time.Duration) {
	durationMs := duration.Milliseconds()

	old := o.stats.AvgQueryTime.Load()
	count := o.stats.TotalQueries.Load()
	if count > 0 {
		newAvg := (old*(count-1) + durationMs) / count
		o.stats.AvgQueryTime.Store(newAvg)
	}
}

func isRetryableError(err error) bool {
	return false
}

func (o *QueryOptimizer) rowsToResult(entry *QueryCacheEntry) *sql.Rows {
	return nil
}

func (o *QueryOptimizer) BatchQuery(ctx context.Context, query string, argsBatch [][]interface{}) [][]interface{} {
	if !o.config.EnableBatchQuery {
		return nil
	}

	o.stats.BatchQueries.Add(1)
	results := make([][]interface{}, len(argsBatch))

	var wg sync.WaitGroup
	batchSize := o.config.BatchSize

	for i := 0; i < len(argsBatch); i += batchSize {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()

			end := start + batchSize
			if end > len(argsBatch) {
				end = len(argsBatch)
			}

			for j := start; j < end; j++ {
				rows, err := o.Query(ctx, query, argsBatch[j]...)
				if err != nil {
					continue
				}

				values, err := scanRows(rows)
				if err == nil {
					results[j] = values
				}
			}
		}(i)
	}

	wg.Wait()
	return results
}

func scanRows(rows *sql.Rows) ([]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))

	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	if rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
	}

	return values, nil
}

func (o *QueryOptimizer) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_queries":     o.stats.TotalQueries.Load(),
		"cache_hits":        o.stats.CacheHits.Load(),
		"cache_misses":      o.stats.CacheMisses.Load(),
		"stmt_cache_hits":   o.stats.StmtCacheHits.Load(),
		"stmt_cache_misses": o.stats.StmtCacheMisses.Load(),
		"slow_queries":      o.stats.SlowQueries.Load(),
		"failed_queries":    o.stats.FailedQueries.Load(),
		"avg_query_time":    fmt.Sprintf("%dms", o.stats.AvgQueryTime.Load()),
		"p50_query_time":    fmt.Sprintf("%dms", o.stats.P50QueryTime.Load()),
		"p95_query_time":    fmt.Sprintf("%dms", o.stats.P95QueryTime.Load()),
		"p99_query_time":    fmt.Sprintf("%dms", o.stats.P99QueryTime.Load()),
		"batch_queries":     o.stats.BatchQueries.Load(),
		"retries":           o.stats.Retries.Load(),
		"query_timeouts":    o.stats.QueryTimeouts.Load(),
	}

	for k, v := range o.queryCache.GetStats() {
		stats["query_cache_"+k] = v
	}

	for k, v := range o.stmtCache.GetStats() {
		stats["stmt_cache_"+k] = v
	}

	for k, v := range o.indexAnalyzer.GetStats() {
		stats["index_"+k] = v
	}

	return stats
}

func (qc *QueryCache) GetStats() map[string]interface{} {
	hits := qc.hits.Load()
	misses := qc.misses.Load()
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"hits":      hits,
		"misses":    misses,
		"hit_rate":  hitRate,
		"evictions": qc.evictions.Load(),
		"size":      qc.getSize(),
		"max_size":  qc.maxSize,
	}
}

func (sc *StatementCache) GetStats() map[string]interface{} {
	hits := sc.hits.Load()
	misses := sc.misses.Load()
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"hits":      hits,
		"misses":    misses,
		"hit_rate":  hitRate,
		"evictions": sc.evictions.Load(),
		"size":      sc.getSize(),
		"max_size":  sc.maxSize,
	}
}

func (o *QueryOptimizer) ClearCaches() {
	o.queryCache.Clear()
	o.stmtCache.Clear()
}

type PreparedStatement struct {
	Query    string
	Stmt     *sql.Stmt
	UseCount int64
}

type QueryBuilder struct {
	table      string
	conditions []string
	orderBy   []string
	limit     int
	offset    int
	args      []interface{}
}

func NewQueryBuilder(table string) *QueryBuilder {
	return &QueryBuilder{
		table: table,
		args:  make([]interface{}, 0),
	}
}

func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	qb.conditions = append(qb.conditions, condition)
	qb.args = append(qb.args, args...)
	return qb
}

func (qb *QueryBuilder) OrderBy(field string, desc bool) *QueryBuilder {
	order := field
	if desc {
		order += " DESC"
	} else {
		order += " ASC"
	}
	qb.orderBy = append(qb.orderBy, order)
	return qb
}

func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

func (qb *QueryBuilder) Build() (string, []interface{}) {
	query := "SELECT * FROM " + qb.table

	if len(qb.conditions) > 0 {
		query += " WHERE "
		for i, cond := range qb.conditions {
			if i > 0 {
				query += " AND "
			}
			query += cond
		}
	}

	if len(qb.orderBy) > 0 {
		query += " ORDER BY "
		query += qb.orderBy[0]
	}

	if qb.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", qb.limit)
	}

	if qb.offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", qb.offset)
	}

	return query, qb.args
}

func (qb *QueryBuilder) BuildCount() (string, []interface{}) {
	query := "SELECT COUNT(*) FROM " + qb.table

	if len(qb.conditions) > 0 {
		query += " WHERE "
		for i, cond := range qb.conditions {
			if i > 0 {
				query += " AND "
			}
			query += cond
		}
	}

	return query, qb.args
}
