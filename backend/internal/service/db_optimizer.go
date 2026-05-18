package service

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type QueryOptimizer struct {
	enablePreparedStatements bool
	slowQueryThreshold       time.Duration
	enableQueryCache         bool
	queryCacheTTL            time.Duration
	queryTimeout             time.Duration
	enableQueryRewrite       bool
	maxCacheSize             int
}

type QueryOption func(*QueryOptimizer)

func WithPreparedStatements(enable bool) QueryOption {
	return func(q *QueryOptimizer) {
		q.enablePreparedStatements = enable
	}
}

func WithSlowQueryThreshold(threshold time.Duration) QueryOption {
	return func(q *QueryOptimizer) {
		q.slowQueryThreshold = threshold
	}
}

func WithQueryCache(enable bool, ttl time.Duration) QueryOption {
	return func(q *QueryOptimizer) {
		q.enableQueryCache = enable
		q.queryCacheTTL = ttl
	}
}

func NewQueryOptimizer(opts ...QueryOption) *QueryOptimizer {
	cfg := config.GetConfig()

	qo := &QueryOptimizer{
		enablePreparedStatements: cfg.Database.QueryOptimization.EnablePreparedStatements,
		slowQueryThreshold:       time.Duration(cfg.Database.SlowQueryThresholdMs) * time.Millisecond,
		enableQueryCache:         cfg.Database.QueryOptimization.EnableQueryCache,
		queryCacheTTL:            time.Duration(cfg.Database.QueryOptimization.QueryCacheTTLSecs) * time.Second,
		queryTimeout:             30 * time.Second,
		enableQueryRewrite:       true,
		maxCacheSize:             cfg.Database.QueryOptimization.MaxQueryCacheSize,
	}

	for _, opt := range opts {
		opt(qo)
	}

	return qo
}

type QueryMetrics struct {
	TotalQueries   int64
	SlowQueries    int64
	FailedQueries  int64
	AvgDuration    time.Duration
	MaxDuration    time.Duration
	MinDuration    time.Duration
	CacheHits      int64
	CacheMisses    int64
	TimeoutQueries int64
}

type QueryMetricsCollector struct {
	mu             sync.RWMutex
	totalQueries   int64
	slowQueries    int64
	failedQueries  int64
	cacheHits      int64
	cacheMisses    int64
	timeoutQueries int64
	durations      []time.Duration
	maxDuration    time.Duration
}

var queryMetrics = &QueryMetricsCollector{
	durations: make([]time.Duration, 0),
}

type QueryCacheEntry struct {
	Value      interface{}
	Expiration time.Time
	QueryHash  string
	AccessTime time.Time
	HitCount   int64
}

type QueryCache struct {
	mu      sync.RWMutex
	entries map[string]*QueryCacheEntry
	maxSize int
	ttl     time.Duration
	hits    int64
	misses  int64
}

var queryCache *QueryCache

func init() {
	cfg := config.GetConfig()
	queryCache = &QueryCache{
		entries: make(map[string]*QueryCacheEntry),
		maxSize: cfg.Database.QueryOptimization.MaxQueryCacheSize,
		ttl:     time.Duration(cfg.Database.QueryOptimization.QueryCacheTTLSecs) * time.Second,
	}
	go queryCache.startCleanup()
}

func (qc *QueryCache) generateKey(query string, args ...interface{}) string {
	data := query
	for _, arg := range args {
		argBytes, _ := json.Marshal(arg)
		data += string(argBytes)
	}
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (qc *QueryCache) Get(key string) (interface{}, bool) {
	qc.mu.RLock()
	entry, exists := qc.entries[key]
	qc.mu.RUnlock()

	if !exists {
		atomic.AddInt64(&qc.misses, 1)
		return nil, false
	}

	if time.Now().After(entry.Expiration) {
		qc.mu.Lock()
		delete(qc.entries, key)
		qc.mu.Unlock()
		atomic.AddInt64(&qc.misses, 1)
		return nil, false
	}

	atomic.AddInt64(&qc.hits, 1)
	atomic.AddInt64(&entry.HitCount, 1)
	entry.AccessTime = time.Now()
	return entry.Value, true
}

func (qc *QueryCache) Set(key string, value interface{}) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	if len(qc.entries) >= qc.maxSize {
		qc.evictLRU()
	}

	qc.entries[key] = &QueryCacheEntry{
		Value:      value,
		Expiration: time.Now().Add(qc.ttl),
		QueryHash:  key,
		AccessTime: time.Now(),
		HitCount:   0,
	}
}

func (qc *QueryCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for k, v := range qc.entries {
		if oldestKey == "" || v.AccessTime.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.AccessTime
		}
	}

	if oldestKey != "" {
		delete(qc.entries, oldestKey)
	}
}

func (qc *QueryCache) startCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		qc.cleanupExpired()
	}
}

func (qc *QueryCache) cleanupExpired() {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	now := time.Now()
	for k, v := range qc.entries {
		if now.After(v.Expiration) {
			delete(qc.entries, k)
		}
	}
}

func (qc *QueryCache) Clear() {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	qc.entries = make(map[string]*QueryCacheEntry)
	atomic.StoreInt64(&qc.hits, 0)
	atomic.StoreInt64(&qc.misses, 0)
}

func (qc *QueryCache) GetStats() map[string]interface{} {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	total := atomic.LoadInt64(&qc.hits) + atomic.LoadInt64(&qc.misses)
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(atomic.LoadInt64(&qc.hits)) / float64(total) * 100
	}

	return map[string]interface{}{
		"size":     len(qc.entries),
		"max_size": qc.maxSize,
		"hits":     atomic.LoadInt64(&qc.hits),
		"misses":   atomic.LoadInt64(&qc.misses),
		"hit_rate": hitRate,
	}
}

func (q *QueryOptimizer) OptimizeQuery(query string) string {
	optimized := query

	optimized = strings.ReplaceAll(optimized, "SELECT *", "SELECT")
	optimized = strings.ReplaceAll(optimized, "select *", "select")

	optimized = strings.ReplaceAll(optimized, "OR 1=1", "")
	optimized = strings.ReplaceAll(optimized, "or 1=1", "")
	optimized = strings.ReplaceAll(optimized, "OR '1'='1'", "")
	optimized = strings.ReplaceAll(optimized, "or '1'='1'", "")

	optimized = q.optimizeLikePatterns(optimized)
	optimized = q.optimizeNotInPatterns(optimized)
	optimized = q.optimizeOrConditions(optimized)
	optimized = q.optimizeCountQueries(optimized)

	return optimized
}

func (q *QueryOptimizer) optimizeLikePatterns(query string) string {
	likeRegex := regexp.MustCompile(`LIKE '%([^%]+)%'`)
	matches := likeRegex.FindAllStringSubmatch(query, -1)

	for _, match := range matches {
		if len(match) > 1 {
			pattern := match[0]
			term := match[1]

			if len(term) > 3 {
				optimizedPattern := fmt.Sprintf("LIKE '%%%s%%'", term[:len(term)/2])

				query = strings.Replace(query, pattern, optimizedPattern, 1)
			}
		}
	}

	return query
}

func (q *QueryOptimizer) optimizeNotInPatterns(query string) string {
	notInRegex := regexp.MustCompile(`NOT IN\s*\(\s*SELECT`)
	if notInRegex.MatchString(query) {
		query = regexp.MustCompile(`NOT IN\s*\(\s*SELECT`).ReplaceAllString(query, "NOT EXISTS (SELECT")
		query = strings.Replace(query, ")", ")", 1)
	}

	return query
}

func (q *QueryOptimizer) optimizeOrConditions(query string) string {
	orCount := strings.Count(query, " OR ")
	if orCount > 3 {
		query = strings.Replace(query, " OR ", " UNION ", orCount/2)
	}

	return query
}

func (q *QueryOptimizer) optimizeCountQueries(query string) string {
	countRegex := regexp.MustCompile(`(?i)SELECT\s+COUNT\(\*?\)\s+FROM`)
	if countRegex.MatchString(query) {
		query = regexp.MustCompile(`(?i)SELECT\s+COUNT`).ReplaceAllString(query, "SELECT COUNT /*+ INDEX(*) */")
	}

	return query
}

func (q *QueryOptimizer) ShouldUseIndex(table string, whereClause string) bool {
	if strings.Contains(strings.ToLower(whereClause), "limit") {
		return true
	}

	hasWhere := len(whereClause) > 0
	return hasWhere
}

func (q *QueryOptimizer) BuildSelectQuery(table string, columns []string, where map[string]interface{}, orderBy string, limit, offset int) string {
	var query strings.Builder

	query.WriteString("SELECT ")

	if len(columns) == 0 {
		query.WriteString("*")
	} else {
		query.WriteString(strings.Join(columns, ", "))
	}

	query.WriteString(" FROM ")
	query.WriteString(table)

	if len(where) > 0 {
		query.WriteString(" WHERE ")
		conditions := make([]string, 0, len(where))
		for key, value := range where {
			if value == nil {
				conditions = append(conditions, fmt.Sprintf("%s IS NULL", key))
			} else {
				conditions = append(conditions, fmt.Sprintf("%s = ?", key))
			}
		}
		query.WriteString(strings.Join(conditions, " AND "))
	}

	if orderBy != "" {
		query.WriteString(" ORDER BY ")
		query.WriteString(orderBy)
	}

	if limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d", limit))
	}

	if offset > 0 {
		query.WriteString(fmt.Sprintf(" OFFSET %d", offset))
	}

	return query.String()
}

func (q *QueryOptimizer) ExplainQuery(db *gorm.DB, query string) (string, error) {
	var result string
	err := db.Raw("EXPLAIN " + query).Scan(&result).Error
	return result, err
}

func (q *QueryOptimizer) AnalyzeTable(db *gorm.DB, tableName string) error {
	return db.Exec(fmt.Sprintf("ANALYZE TABLE %s", tableName)).Error
}

func (q *QueryOptimizer) CheckIndexes(db *gorm.DB, tableName string) ([]map[string]interface{}, error) {
	var indexes []map[string]interface{}
	err := db.Raw(fmt.Sprintf(`
		SELECT 
			indexname, 
			indexdef 
		FROM pg_indexes 
		WHERE tablename = '%s'
	`, tableName)).Scan(&indexes).Error
	return indexes, err
}

func (q *QueryOptimizer) SuggestIndexes(db *gorm.DB, tableName string) ([]string, error) {
	var suggestions []string

	var slowQueries []struct {
		Query string `gorm:"column:query"`
		Calls int64  `gorm:"column:calls"`
		Time  int64  `gorm:"column:total"`
	}

	err := db.Raw(`
		SELECT query, calls, total 
		FROM pg_stat_statements 
		WHERE query LIKE '%` + tableName + `%' 
		AND total > 1000000
		ORDER BY total DESC 
		LIMIT 10
	`).Scan(&slowQueries).Error

	if err != nil {
		return suggestions, nil
	}

	for _, sq := range slowQueries {
		if strings.Contains(sq.Query, "WHERE") {
			whereClause := strings.Split(sq.Query, "WHERE")[1]
			whereClause = strings.Split(whereClause, "ORDER")[0]
			whereClause = strings.Split(whereClause, "LIMIT")[0]

			fields := strings.Split(whereClause, "AND")
			var indexedFields []string
			for _, field := range fields {
				field = strings.TrimSpace(field)
				if strings.Contains(field, "=") {
					fieldName := strings.Split(field, "=")[0]
					fieldName = strings.TrimSpace(fieldName)
					indexedFields = append(indexedFields, fieldName)
				}
			}

			if len(indexedFields) > 0 {
				suggestion := fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s (%s)",
					tableName, strings.Join(indexedFields, "_"), tableName, strings.Join(indexedFields, ", "))
				suggestions = append(suggestions, suggestion)
			}
		}
	}

	return suggestions, nil
}

func (q *QueryOptimizer) ExecuteWithTimeout(ctx context.Context, db *gorm.DB, query string, dest interface{}, args ...interface{}) error {
	queryCtx, cancel := context.WithTimeout(ctx, q.queryTimeout)
	defer cancel()

	result := db.WithContext(queryCtx).Raw(query, args...).Scan(dest)
	if result.Error != nil {
		if queryCtx.Err() == context.DeadlineExceeded {
			atomic.AddInt64(&queryMetrics.timeoutQueries, 1)
			return fmt.Errorf("query timeout exceeded: %w", result.Error)
		}
		return result.Error
	}

	return nil
}

func CreateIndexes(db *gorm.DB) error {
	indexes := []struct {
		TableName string
		Columns   []string
		IndexName string
		Unique    bool
	}{
		{
			TableName: "users",
			Columns:   []string{"email"},
			IndexName: "idx_users_email",
			Unique:    true,
		},
		{
			TableName: "users",
			Columns:   []string{"username"},
			IndexName: "idx_users_username",
			Unique:    false,
		},
		{
			TableName: "users",
			Columns:   []string{"created_at"},
			IndexName: "idx_users_created_at",
			Unique:    false,
		},
		{
			TableName: "applications",
			Columns:   []string{"app_key"},
			IndexName: "idx_applications_app_key",
			Unique:    true,
		},
		{
			TableName: "applications",
			Columns:   []string{"user_id"},
			IndexName: "idx_applications_user_id",
			Unique:    false,
		},
		{
			TableName: "applications",
			Columns:   []string{"created_at"},
			IndexName: "idx_applications_created_at",
			Unique:    false,
		},
		{
			TableName: "verifications",
			Columns:   []string{"created_at"},
			IndexName: "idx_verifications_created_at",
			Unique:    false,
		},
		{
			TableName: "verifications",
			Columns:   []string{"user_id", "created_at"},
			IndexName: "idx_verifications_user_created",
			Unique:    false,
		},
		{
			TableName: "verifications",
			Columns:   []string{"application_id"},
			IndexName: "idx_verifications_app_id",
			Unique:    false,
		},
		{
			TableName: "verifications",
			Columns:   []string{"status"},
			IndexName: "idx_verifications_status",
			Unique:    false,
		},
		{
			TableName: "verifications",
			Columns:   []string{"ip_address"},
			IndexName: "idx_verifications_ip",
			Unique:    false,
		},
		{
			TableName: "blacklist",
			Columns:   []string{"ip_address", "type"},
			IndexName: "idx_blacklist_ip_type",
			Unique:    false,
		},
		{
			TableName: "blacklist",
			Columns:   []string{"expires_at"},
			IndexName: "idx_blacklist_expires",
			Unique:    false,
		},
		{
			TableName: "verification_logs",
			Columns:   []string{"created_at"},
			IndexName: "idx_verification_logs_created",
			Unique:    false,
		},
		{
			TableName: "verification_logs",
			Columns:   []string{"session_id"},
			IndexName: "idx_verification_logs_session",
			Unique:    false,
		},
		{
			TableName: "captcha_sessions",
			Columns:   []string{"status"},
			IndexName: "idx_captcha_sessions_status",
			Unique:    false,
		},
		{
			TableName: "captcha_sessions",
			Columns:   []string{"created_at"},
			IndexName: "idx_captcha_sessions_created",
			Unique:    false,
		},
		{
			TableName: "captcha_sessions",
			Columns:   []string{"client_ip", "created_at"},
			IndexName: "idx_captcha_sessions_ip_created",
			Unique:    false,
		},
		{
			TableName: "risk_logs",
			Columns:   []string{"risk_level"},
			IndexName: "idx_risk_logs_risk_level",
			Unique:    false,
		},
		{
			TableName: "risk_logs",
			Columns:   []string{"created_at"},
			IndexName: "idx_risk_logs_created",
			Unique:    false,
		},
		{
			TableName: "risk_logs",
			Columns:   []string{"session_id", "created_at"},
			IndexName: "idx_risk_logs_session_created",
			Unique:    false,
		},
		{
			TableName: "ab_test_experiments",
			Columns:   []string{"status", "start_time"},
			IndexName: "idx_ab_test_status_start",
			Unique:    false,
		},
		{
			TableName: "audit_logs",
			Columns:   []string{"user_id", "created_at"},
			IndexName: "idx_audit_logs_user_created",
			Unique:    false,
		},
	}

	for _, idx := range indexes {
		if err := createIndexIfNotExists(db, idx.TableName, idx.Columns, idx.IndexName, idx.Unique); err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.IndexName, err)
		}
	}

	return nil
}

func createIndexIfNotExists(db *gorm.DB, tableName string, columns []string, indexName string, unique bool) error {
	var count int64
	err := db.Raw(fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM pg_indexes 
		WHERE indexname = '%s'
	`, indexName)).Scan(&count).Error

	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	uniqueStr := ""
	if unique {
		uniqueStr = "UNIQUE "
	}

	columnsStr := strings.Join(columns, ", ")
	createSQL := fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)", uniqueStr, indexName, tableName, columnsStr)

	return db.Exec(createSQL).Error
}

func GetQueryMetrics() *QueryMetrics {
	metrics := &QueryMetrics{}

	queryMetrics.mu.RLock()
	defer queryMetrics.mu.RUnlock()

	metrics.TotalQueries = queryMetrics.totalQueries
	metrics.SlowQueries = queryMetrics.slowQueries
	metrics.FailedQueries = queryMetrics.failedQueries
	metrics.CacheHits = queryMetrics.cacheHits
	metrics.CacheMisses = queryMetrics.cacheMisses
	metrics.TimeoutQueries = queryMetrics.timeoutQueries

	if len(queryMetrics.durations) > 0 {
		var totalDuration time.Duration
		var maxDuration time.Duration
		var minDuration time.Duration = queryMetrics.durations[0]

		for _, d := range queryMetrics.durations {
			totalDuration += d
			if d > maxDuration {
				maxDuration = d
			}
			if d < minDuration {
				minDuration = d
			}
		}

		metrics.AvgDuration = totalDuration / time.Duration(len(queryMetrics.durations))
		metrics.MaxDuration = maxDuration
		metrics.MinDuration = minDuration
	}

	return metrics
}

func ResetQueryMetrics() {
	queryMetrics.mu.Lock()
	defer queryMetrics.mu.Unlock()

	queryMetrics.totalQueries = 0
	queryMetrics.slowQueries = 0
	queryMetrics.failedQueries = 0
	queryMetrics.cacheHits = 0
	queryMetrics.cacheMisses = 0
	queryMetrics.timeoutQueries = 0
	queryMetrics.durations = nil
}

func RecordQueryMetrics(duration time.Duration, isSlow bool, err error) {
	queryMetrics.mu.Lock()
	defer queryMetrics.mu.Unlock()

	queryMetrics.totalQueries++
	queryMetrics.durations = append(queryMetrics.durations, duration)

	if len(queryMetrics.durations) > 10000 {
		queryMetrics.durations = queryMetrics.durations[len(queryMetrics.durations)-5000:]
	}

	if isSlow {
		queryMetrics.slowQueries++
	}

	if err != nil {
		queryMetrics.failedQueries++
	}

	if duration > queryMetrics.maxDuration {
		queryMetrics.maxDuration = duration
	}
}

func RecordCacheHit() {
	atomic.AddInt64(&queryMetrics.cacheHits, 1)
}

func RecordCacheMiss() {
	atomic.AddInt64(&queryMetrics.cacheMisses, 1)
}

type DBOptimizer struct {
	db             *gorm.DB
	queryOptimizer *QueryOptimizer
	readReplica    *gorm.DB
	useReplica     atomic.Bool
}

func NewDBOptimizer(db *gorm.DB, opts ...QueryOption) *DBOptimizer {
	return &DBOptimizer{
		db:             db,
		queryOptimizer: NewQueryOptimizer(opts...),
	}
}

func (o *DBOptimizer) ConfigureConnectionPool(maxOpenConns, maxIdleConns int, connMaxLifetime time.Duration) error {
	sqlDB, err := o.db.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	return nil
}

func (o *DBOptimizer) EnableQueryLogging() error {
	return o.db.Callback().Query().Before("gorm:query").Register("query_logger", func(db *gorm.DB) {
		start := time.Now()
		db.Callback().Query().After("gorm:query").Register("query_logger_after", func(db *gorm.DB) {
			duration := time.Since(start)
			sql := db.Statement.SQL.String()

			RecordQueryMetrics(duration, duration > o.queryOptimizer.slowQueryThreshold, db.Error)

			if duration > o.queryOptimizer.slowQueryThreshold {
				log.Printf("[SLOW_QUERY] Duration: %v, Query: %s\n", duration, sql)
			}
		})
	})
}

func (o *DBOptimizer) OptimizeWrites() error {
	return o.db.Callback().Create().Before("gorm:create").Register("optimize_create", func(db *gorm.DB) {
		if db.Statement.Schema != nil {
			for _, field := range db.Statement.Schema.Fields {
				if field.AutoIncrement {
					db.Statement.AddClause(clause.OnConflict{DoNothing: true})
					break
				}
			}
		}
	})
}

func (o *DBOptimizer) BatchInsert(tableName string, records []map[string]interface{}, batchSize int) error {
	if len(records) == 0 {
		return nil
	}

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]
		if err := o.db.Table(tableName).CreateInBatches(batch, batchSize).Error; err != nil {
			return err
		}
	}

	return nil
}

func (o *DBOptimizer) PaginatedFind(tableName string, where map[string]interface{}, page, pageSize int, dest interface{}) (int64, error) {
	var total int64

	query := o.db.Table(tableName).Where(where)
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(dest).Error; err != nil {
		return 0, err
	}

	return total, nil
}

func (o *DBOptimizer) SoftDelete(tableName string, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	return o.db.Table(tableName).
		Where("id IN ?", ids).
		Update("deleted_at", time.Now()).Error
}

func (o *DBOptimizer) Restore(tableName string, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	return o.db.Table(tableName).
		Where("id IN ?", ids).
		Update("deleted_at", nil).Error
}

func (o *DBOptimizer) CachedQuery(ctx context.Context, cacheKey string, dest interface{}, queryFunc func() error, ttl ...time.Duration) error {
	if cached, ok := queryCache.Get(cacheKey); ok {
		RecordCacheHit()
		if cachedData, err := json.Marshal(cached); err == nil {
			json.Unmarshal(cachedData, dest)
			return nil
		}
	}

	RecordCacheMiss()
	if err := queryFunc(); err != nil {
		return err
	}

	queryCache.Set(cacheKey, dest)
	return nil
}

func (o *DBOptimizer) InvalidateQueryCache(pattern string) {
	queryCache.Clear()
}

func HealthCheckDB(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return err
	}

	stats := sqlDB.Stats()
	if stats.OpenConnections == 0 {
		return fmt.Errorf("no open connections")
	}

	if stats.InUse > stats.MaxOpenConnections/2 {
		return fmt.Errorf("high connection usage: %d/%d", stats.InUse, stats.MaxOpenConnections)
	}

	return nil
}

func (o *DBOptimizer) OptimizeComplexQuery(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	optimizedQuery := o.queryOptimizer.OptimizeQuery(query)

	if len(args) > 0 {
		optimizedQuery = fmt.Sprintf(optimizedQuery, args...)
	}

	return o.db.WithContext(ctx).Raw(optimizedQuery).Rows()
}

func (o *DBOptimizer) SetReadReplica(db *gorm.DB) {
	o.readReplica = db
}

func (o *DBOptimizer) UseReadReplica(enable bool) {
	o.useReplica.Store(enable)
}

func (o *DBOptimizer) ReadFromReplica(queryFunc func(*gorm.DB) *gorm.DB) *gorm.DB {
	if o.useReplica.Load() && o.readReplica != nil {
		return queryFunc(o.readReplica)
	}
	return queryFunc(o.db)
}

type QueryPlan struct {
	Query       string
	Plan        string
	Cost        float64
	EstimatedMs float64
}

func (o *DBOptimizer) AnalyzeQueryPlan(query string) (*QueryPlan, error) {
	var plan string
	err := o.db.Raw("EXPLAIN (FORMAT JSON) " + query).Scan(&plan).Error
	if err != nil {
		return nil, err
	}

	return &QueryPlan{
		Query: query,
		Plan:  plan,
		Cost:  0,
	}, nil
}

func (o *DBOptimizer) VacuumTable(tableName string) error {
	return o.db.Exec(fmt.Sprintf("VACUUM ANALYZE %s", tableName)).Error
}

func (o *DBOptimizer) ReindexTable(tableName string) error {
	return o.db.Exec(fmt.Sprintf("REINDEX TABLE %s", tableName)).Error
}
