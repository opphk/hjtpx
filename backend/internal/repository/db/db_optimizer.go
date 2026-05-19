package db

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type DBOptimizer struct {
	db              *gorm.DB
	queryCache      *RepositoryQueryCache
	indexManager    *IndexManager
	batchProcessor  *BatchProcessor
	ormOptimizer    *ORMQueryOptimizer
	metricsCollector *MetricsCollector
	readReplica     *gorm.DB
	useReplica      atomic.Bool
	mu              sync.RWMutex
}

type OptimizerConfig struct {
	EnableQueryCache    bool
	QueryCacheTTL       time.Duration
	MaxQueryCacheSize   int
	BatchSize           int
	MaxQueryTimeout     time.Duration
	SlowQueryThreshold  time.Duration
	EnableReadReplica   bool
}

var defaultOptimizerConfig = &OptimizerConfig{
	EnableQueryCache:    true,
	QueryCacheTTL:       5 * time.Minute,
	MaxQueryCacheSize:   1000,
	BatchSize:           100,
	MaxQueryTimeout:     30 * time.Second,
	SlowQueryThreshold:  100 * time.Millisecond,
	EnableReadReplica:   false,
}

func NewDBOptimizer(db *gorm.DB, cfg *OptimizerConfig) *DBOptimizer {
	if cfg == nil {
		cfg = defaultOptimizerConfig
	}

	optimizer := &DBOptimizer{
		db:             db,
		queryCache:     NewRepositoryQueryCache(cfg.MaxQueryCacheSize, cfg.QueryCacheTTL),
		indexManager:   NewIndexManager(db),
		batchProcessor: NewBatchProcessor(db, cfg.BatchSize),
		ormOptimizer:   NewORMQueryOptimizer(db),
		metricsCollector: NewMetricsCollector(),
	}

	if cfg.EnableQueryCache {
		go optimizer.queryCache.StartCleanup()
	}

	return optimizer
}

func (o *DBOptimizer) SetReadReplica(replica *gorm.DB) {
	o.readReplica = replica
}

func (o *DBOptimizer) EnableReplica() {
	o.useReplica.Store(true)
}

func (o *DBOptimizer) DisableReplica() {
	o.useReplica.Store(false)
}

func (o *DBOptimizer) GetDB() *gorm.DB {
	if o.useReplica.Load() && o.readReplica != nil {
		return o.readReplica
	}
	return o.db
}

func (o *DBOptimizer) CreateIndexes(ctx context.Context) error {
	return o.indexManager.CreateAllIndexes(ctx)
}

func (o *DBOptimizer) AnalyzeIndexes(ctx context.Context) ([]IndexAnalysis, error) {
	return o.indexManager.AnalyzeIndexes(ctx)
}

func (o *DBOptimizer) SuggestIndexes(ctx context.Context) ([]string, error) {
	return o.indexManager.SuggestIndexes(ctx)
}

func (o *DBOptimizer) CachedFind(ctx context.Context, cacheKey string, dest interface{}, queryFunc func() error, ttl ...time.Duration) error {
	return o.queryCache.GetOrSet(cacheKey, dest, queryFunc, ttl...)
}

func (o *DBOptimizer) InvalidateCache(pattern string) {
	o.queryCache.Invalidate(pattern)
}

func (o *DBOptimizer) ClearCache() {
	o.queryCache.Clear()
}

func (o *DBOptimizer) BatchInsert(ctx context.Context, tableName string, records []map[string]interface{}) error {
	startTime := time.Now()
	defer func() {
		o.metricsCollector.RecordWrite(time.Since(startTime), "batch_insert", tableName)
	}()

	return o.batchProcessor.BatchInsert(ctx, tableName, records)
}

func (o *DBOptimizer) BatchUpdate(ctx context.Context, tableName string, updates []map[string]interface{}, idField string) error {
	startTime := time.Now()
	defer func() {
		o.metricsCollector.RecordWrite(time.Since(startTime), "batch_update", tableName)
	}()

	return o.batchProcessor.BatchUpdate(ctx, tableName, updates, idField)
}

func (o *DBOptimizer) BatchUpsert(ctx context.Context, tableName string, records []map[string]interface{}, conflictKeys []string) error {
	startTime := time.Now()
	defer func() {
		o.metricsCollector.RecordWrite(time.Since(startTime), "batch_upsert", tableName)
	}()

	return o.batchProcessor.BatchUpsert(ctx, tableName, records, conflictKeys)
}

func (o *DBOptimizer) OptimizeQuery(query string, args ...interface{}) (*gorm.DB, error) {
	if o.db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}
	optimized := o.ormOptimizer.Optimize(query, args...)
	return o.db.Raw(optimized, args...), nil
}

func (o *DBOptimizer) ExecuteWithTimeout(ctx context.Context, timeout time.Duration, query string, dest interface{}, args ...interface{}) error {
	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startTime := time.Now()
	err := o.db.WithContext(queryCtx).Raw(query, args...).Scan(dest).Error

	duration := time.Since(startTime)
	o.metricsCollector.RecordQuery(duration, err != nil, query)

	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}

	return nil
}

func (o *DBOptimizer) PaginatedQuery(ctx context.Context, tableName string, where map[string]interface{}, page, pageSize int, dest interface{}) (int64, error) {
	startTime := time.Now()
	defer func() {
		o.metricsCollector.RecordQuery(time.Since(startTime), false, "paginated_query")
	}()

	db := o.GetDB()
	if db == nil {
		return 0, fmt.Errorf("database is not initialized")
	}

	return o.ormOptimizer.PaginatedFind(ctx, db, tableName, where, page, pageSize, dest)
}

func (o *DBOptimizer) PreloadOptimized(query *gorm.DB, preloadFields []string) *gorm.DB {
	return o.ormOptimizer.PreloadWithBatch(query, preloadFields)
}

func (o *DBOptimizer) SelectOptimized(query *gorm.DB, columns []string) *gorm.DB {
	return o.ormOptimizer.SelectColumns(query, columns)
}

func (o *DBOptimizer) GetMetrics() *OptimizerMetrics {
	return o.metricsCollector.GetMetrics()
}

func (o *DBOptimizer) GetCacheStats() map[string]interface{} {
	return o.queryCache.GetStats()
}

func (o *DBOptimizer) HealthCheck(ctx context.Context) error {
	if o.db == nil {
		return fmt.Errorf("database is not initialized")
	}

	sqlDB, err := o.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	stats := sqlDB.Stats()
	if stats.InUse > stats.MaxOpenConnections/2 {
		return fmt.Errorf("high connection usage: %d/%d", stats.InUse, stats.MaxOpenConnections)
	}

	return nil
}

func (o *DBOptimizer) VacuumTable(ctx context.Context, tableName string) error {
	return o.db.WithContext(ctx).Exec(fmt.Sprintf("VACUUM ANALYZE %s", tableName)).Error
}

func (o *DBOptimizer) ReindexTable(ctx context.Context, tableName string) error {
	return o.db.WithContext(ctx).Exec(fmt.Sprintf("REINDEX TABLE %s", tableName)).Error
}

func (o *DBOptimizer) AnalyzeTable(ctx context.Context, tableName string) error {
	return o.db.WithContext(ctx).Exec(fmt.Sprintf("ANALYZE %s", tableName)).Error
}

func (o *DBOptimizer) GetQueryPlan(ctx context.Context, query string) (string, error) {
	var plan string
	err := o.db.WithContext(ctx).Raw("EXPLAIN (FORMAT JSON) " + query).Scan(&plan).Error
	return plan, err
}

type RepositoryQueryCache struct {
	mu       sync.RWMutex
	entries  map[string]*CacheEntry
	maxSize  int
	ttl      time.Duration
	hits     atomic.Int64
	misses   atomic.Int64
	evictions atomic.Int64
	stopCh   chan struct{}
}

type CacheEntry struct {
	Value       interface{}
	Expiration  time.Time
	AccessCount int64
	LastAccess  time.Time
	Key         string
}

func NewRepositoryQueryCache(maxSize int, ttl time.Duration) *RepositoryQueryCache {
	return &RepositoryQueryCache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
		stopCh:  make(chan struct{}),
	}
}

func (c *RepositoryQueryCache) generateKey(query string, args ...interface{}) string {
	data := query
	for _, arg := range args {
		argBytes, _ := json.Marshal(arg)
		data += string(argBytes)
	}
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (c *RepositoryQueryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		c.misses.Add(1)
		return nil, false
	}

	if time.Now().After(entry.Expiration) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		c.misses.Add(1)
		return nil, false
	}

	c.mu.Lock()
	entry.AccessCount++
	entry.LastAccess = time.Now()
	c.mu.Unlock()

	c.hits.Add(1)
	return entry.Value, true
}

func (c *RepositoryQueryCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.maxSize {
		c.evictLRU()
	}

	entry := &CacheEntry{
		Value:       value,
		Expiration:  time.Now().Add(c.ttl),
		AccessCount: 1,
		LastAccess:  time.Now(),
		Key:         key,
	}

	c.entries[key] = entry
}

func (c *RepositoryQueryCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for k, v := range c.entries {
		if oldestKey == "" || v.LastAccess.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.LastAccess
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.evictions.Add(1)
	}
}

func (c *RepositoryQueryCache) GetOrSet(key string, dest interface{}, queryFunc func() error, ttl ...time.Duration) error {
	if cached, ok := c.Get(key); ok {
		if data, err := json.Marshal(cached); err == nil {
			json.Unmarshal(data, dest)
			return nil
		}
	}

	if err := queryFunc(); err != nil {
		return err
	}

	c.Set(key, dest)
	return nil
}

func (c *RepositoryQueryCache) Invalidate(pattern string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k := range c.entries {
		if strings.Contains(k, pattern) {
			delete(c.entries, k)
		}
	}
}

func (c *RepositoryQueryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
	c.hits.Store(0)
	c.misses.Store(0)
	c.evictions.Store(0)
}

func (c *RepositoryQueryCache) StartCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.stopCh:
			return
		}
	}
}

func (c *RepositoryQueryCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, v := range c.entries {
		if now.After(v.Expiration) {
			delete(c.entries, k)
			c.evictions.Add(1)
		}
	}
}

func (c *RepositoryQueryCache) GetStats() map[string]interface{} {
	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"size":       len(c.entries),
		"max_size":   c.maxSize,
		"hits":       hits,
		"misses":     misses,
		"hit_rate":   hitRate,
		"evictions":  c.evictions.Load(),
	}
}

type IndexManager struct {
	db *gorm.DB
}

func NewIndexManager(db *gorm.DB) *IndexManager {
	return &IndexManager{db: db}
}

type IndexDefinition struct {
	TableName  string
	Columns    []string
	IndexName  string
	IsUnique   bool
	IsPartial  bool
	WhereClause string
}

func (m *IndexManager) GetRequiredIndexes() []IndexDefinition {
	return []IndexDefinition{
		{TableName: "users", Columns: []string{"email"}, IndexName: "idx_users_email", IsUnique: true},
		{TableName: "users", Columns: []string{"username"}, IndexName: "idx_users_username", IsUnique: false},
		{TableName: "users", Columns: []string{"status", "created_at"}, IndexName: "idx_users_status_created", IsUnique: false},
		{TableName: "applications", Columns: []string{"app_key"}, IndexName: "idx_applications_app_key", IsUnique: true},
		{TableName: "applications", Columns: []string{"user_id", "is_active"}, IndexName: "idx_applications_user_active", IsUnique: false},
		{TableName: "verifications", Columns: []string{"session_id"}, IndexName: "idx_verifications_session", IsUnique: true},
		{TableName: "verifications", Columns: []string{"status", "created_at"}, IndexName: "idx_verifications_status_created", IsUnique: false},
		{TableName: "verifications", Columns: []string{"application_id", "status"}, IndexName: "idx_verifications_app_status", IsUnique: false},
		{TableName: "blacklist", Columns: []string{"blacklisted_value", "blacklist_type"}, IndexName: "idx_blacklist_value_type", IsUnique: true, IsPartial: true, WhereClause: "is_active = true"},
		{TableName: "verification_logs", Columns: []string{"session_id", "created_at"}, IndexName: "idx_logs_session_created", IsUnique: false},
		{TableName: "verification_logs", Columns: []string{"application_id", "status", "created_at"}, IndexName: "idx_logs_app_status_created", IsUnique: false},
		{TableName: "captcha_sessions", Columns: []string{"session_id"}, IndexName: "idx_captcha_session_id", IsUnique: true},
		{TableName: "captcha_sessions", Columns: []string{"status", "expired_at"}, IndexName: "idx_captcha_status_expires", IsUnique: false, IsPartial: true, WhereClause: "status = 'pending'"},
		{TableName: "risk_logs", Columns: []string{"session_id", "created_at"}, IndexName: "idx_risk_session_created", IsUnique: false},
		{TableName: "risk_logs", Columns: []string{"risk_level", "created_at"}, IndexName: "idx_risk_level_created", IsUnique: false},
		{TableName: "admin_login_logs", Columns: []string{"username", "created_at"}, IndexName: "idx_admin_login_user_created", IsUnique: false},
		{TableName: "configs", Columns: []string{"category", "config_key"}, IndexName: "idx_configs_category_key", IsUnique: true},
	}
}

func (m *IndexManager) CreateAllIndexes(ctx context.Context) error {
	indexes := m.GetRequiredIndexes()

	for _, idx := range indexes {
		exists, err := m.IndexExists(ctx, idx.IndexName)
		if err != nil {
			continue
		}

		if exists {
			continue
		}

		if err := m.CreateIndex(ctx, idx); err != nil {
			continue
		}
	}

	return nil
}

func (m *IndexManager) IndexExists(ctx context.Context, indexName string) (bool, error) {
	var count int64
	err := m.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM pg_indexes WHERE indexname = ?
	`, indexName).Scan(&count).Error

	return count > 0, err
}

func (m *IndexManager) CreateIndex(ctx context.Context, def IndexDefinition) error {
	var sql strings.Builder
	sql.WriteString("CREATE ")

	if def.IsUnique {
		sql.WriteString("UNIQUE ")
	}

	sql.WriteString(fmt.Sprintf("INDEX %s ON %s (", def.IndexName, def.TableName))
	sql.WriteString(strings.Join(def.Columns, ", "))
	sql.WriteString(")")

	if def.IsPartial && def.WhereClause != "" {
		sql.WriteString(" WHERE ")
		sql.WriteString(def.WhereClause)
	}

	return m.db.WithContext(ctx).Exec(sql.String()).Error
}

func (m *IndexManager) DropIndex(ctx context.Context, indexName string) error {
	return m.db.WithContext(ctx).Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)).Error
}

type IndexAnalysis struct {
	IndexName     string
	TableName     string
	IndexSize     string
	IndexScans    int64
	IndexTuples   int64
	IndexDefinition string
}

func (m *IndexManager) AnalyzeIndexes(ctx context.Context) ([]IndexAnalysis, error) {
	var results []IndexAnalysis

	err := m.db.WithContext(ctx).Raw(`
		SELECT 
			idx.relname AS indexname,
			tbl.relname AS tablename,
			pg_size_pretty(pg_relation_size(idx.oid)) AS index_size,
			idx.scan_count AS index_scans,
			idx.tuple_read AS index_tuples,
			indexdef
		FROM pg_stat_user_indexes idx
		JOIN pg_index USING (indexrelid)
		JOIN pg_class tbl ON tbl.oid = idx.relid
		JOIN pg_indexes ON pg_indexes.indexname = idx.relname
		WHERE idx.scan_count > 0
		ORDER BY idx.scan_count DESC
	`).Scan(&results).Error

	return results, err
}

func (m *IndexManager) SuggestIndexes(ctx context.Context) ([]string, error) {
	var suggestions []string

	var slowQueries []struct {
		Query string `gorm:"column:query"`
		Calls int64  `gorm:"column:calls"`
		Time  int64  `gorm:"column:total"`
	}

	err := m.db.WithContext(ctx).Raw(`
		SELECT query, calls, total 
		FROM pg_stat_statements 
		WHERE total > 1000000
		ORDER BY total DESC 
		LIMIT 20
	`).Scan(&slowQueries).Error

	if err != nil {
		return suggestions, err
	}

	for _, sq := range slowQueries {
		if strings.Contains(strings.ToLower(sq.Query), "where") {
			whereClause := extractWhereClause(sq.Query)
			if whereClause != "" {
				suggestion := fmt.Sprintf("-- Consider creating index for: %s\nCREATE INDEX CONCURRENTLY idx_suggested ON <table> (%s);",
					sq.Query[:min(100, len(sq.Query))], whereClause)
				suggestions = append(suggestions, suggestion)
			}
		}
	}

	return suggestions, nil
}

func extractWhereClause(query string) string {
	upperQuery := strings.ToUpper(query)
	whereIdx := strings.Index(upperQuery, "WHERE")
	if whereIdx == -1 {
		return ""
	}

	query = query[whereIdx+5:]
	orderIdx := strings.Index(strings.ToUpper(query), "ORDER")
	limitIdx := strings.Index(strings.ToUpper(query), "LIMIT")

	if orderIdx != -1 {
		query = query[:orderIdx]
	}
	if limitIdx != -1 {
		query = query[:min(limitIdx, len(query))]
	}

	query = strings.TrimSpace(query)
	fields := strings.FieldsFunc(query, func(r rune) bool {
		return r == '=' || r == '<' || r == '>'
	})

	var columns []string
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if strings.HasPrefix(strings.ToUpper(f), "AND ") || strings.HasPrefix(strings.ToUpper(f), "OR ") {
			f = f[4:]
		}
		f = strings.TrimPrefix(f, ".")
		if f != "" && !strings.Contains(f, "?") && len(f) < 50 {
			columns = append(columns, f)
		}
	}

	if len(columns) > 0 {
		return strings.Join(columns, ", ")
	}
	return ""
}

type BatchProcessor struct {
	db        *gorm.DB
	batchSize int
}

func NewBatchProcessor(db *gorm.DB, batchSize int) *BatchProcessor {
	return &BatchProcessor{
		db:        db,
		batchSize: batchSize,
	}
}

func (p *BatchProcessor) BatchInsert(ctx context.Context, tableName string, records []map[string]interface{}) error {
	if len(records) == 0 {
		return nil
	}

	for i := 0; i < len(records); i += p.batchSize {
		end := i + p.batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]
		if err := p.db.WithContext(ctx).Table(tableName).CreateInBatches(batch, p.batchSize).Error; err != nil {
			return fmt.Errorf("batch insert failed at batch %d: %w", i/p.batchSize, err)
		}
	}

	return nil
}

func (p *BatchProcessor) BatchUpdate(ctx context.Context, tableName string, updates []map[string]interface{}, idField string) error {
	if len(updates) == 0 {
		return nil
	}

	for i := 0; i < len(updates); i += p.batchSize {
		end := i + p.batchSize
		if end > len(updates) {
			end = len(updates)
		}

		batch := updates[i:end]

		for _, update := range batch {
			id, ok := update[idField]
			if !ok {
				continue
			}

			delete(update, idField)
			if err := p.db.WithContext(ctx).Table(tableName).Where(idField+" = ?", id).Updates(update).Error; err != nil {
				return fmt.Errorf("batch update failed: %w", err)
			}
		}
	}

	return nil
}

func (p *BatchProcessor) BatchUpsert(ctx context.Context, tableName string, records []map[string]interface{}, conflictKeys []string) error {
	if len(records) == 0 {
		return nil
	}

	for i := 0; i < len(records); i += p.batchSize {
		end := i + p.batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]
		if err := p.db.WithContext(ctx).Table(tableName).Clauses(clause.OnConflict{
			Columns:   p.columnsToClause(conflictKeys),
			DoUpdates: clause.AssignmentColumns(p.getUpdateColumns(batch[0], conflictKeys)),
		}).CreateInBatches(batch, p.batchSize).Error; err != nil {
			return fmt.Errorf("batch upsert failed: %w", err)
		}
	}

	return nil
}

func (p *BatchProcessor) columnsToClause(columns []string) []clause.Column {
	clauseCols := make([]clause.Column, len(columns))
	for i, col := range columns {
		clauseCols[i] = clause.Column{Name: col}
	}
	return clauseCols
}

func (p *BatchProcessor) getUpdateColumns(record map[string]interface{}, excludeKeys []string) []string {
	excludeMap := make(map[string]bool)
	for _, k := range excludeKeys {
		excludeMap[k] = true
	}

	var columns []string
	for k := range record {
		if !excludeMap[k] {
			columns = append(columns, k)
		}
	}
	return columns
}

type ORMQueryOptimizer struct {
	db *gorm.DB
}

func NewORMQueryOptimizer(db *gorm.DB) *ORMQueryOptimizer {
	return &ORMQueryOptimizer{db: db}
}

func (o *ORMQueryOptimizer) Optimize(query string, args ...interface{}) string {
	optimized := query

	optimized = strings.ReplaceAll(optimized, "SELECT *", "SELECT")
	optimized = strings.ReplaceAll(optimized, "select *", "select")

	if !strings.Contains(optimized, "LIMIT") {
		optimized = strings.TrimSuffix(optimized, ";") + " LIMIT 1000"
	}

	return optimized
}

func (o *ORMQueryOptimizer) SelectColumns(query *gorm.DB, columns []string) *gorm.DB {
	if len(columns) == 0 {
		return query
	}
	return query.Select(columns)
}

func (o *ORMQueryOptimizer) PreloadWithBatch(query *gorm.DB, preloadFields []string) *gorm.DB {
	for _, field := range preloadFields {
		query = query.Preload(field)
	}
	return query
}

func (o *ORMQueryOptimizer) PaginatedFind(ctx context.Context, db *gorm.DB, tableName string, where map[string]interface{}, page, pageSize int, dest interface{}) (int64, error) {
	var total int64

	countQuery := db.WithContext(ctx).Table(tableName).Where(where)
	if err := countQuery.Count(&total).Error; err != nil {
		return 0, fmt.Errorf("count query failed: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := db.WithContext(ctx).Table(tableName).Where(where).Offset(offset).Limit(pageSize).Find(dest).Error; err != nil {
		return 0, fmt.Errorf("data query failed: %w", err)
	}

	return total, nil
}

func (o *ORMQueryOptimizer) UseIndex(query *gorm.DB, indexName string) *gorm.DB {
	return query.Session(&gorm.Session{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

type MetricsCollector struct {
	mu             sync.RWMutex
	totalQueries   int64
	slowQueries    int64
	failedQueries  int64
	queryDurations []time.Duration
	maxDuration    time.Duration
	cacheHits      int64
	cacheMisses    int64
	writeOps       int64
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		queryDurations: make([]time.Duration, 0),
	}
}

func (m *MetricsCollector) RecordQuery(duration time.Duration, isError bool, queryType string) {
	atomic.AddInt64(&m.totalQueries, 1)

	if duration > 100*time.Millisecond {
		atomic.AddInt64(&m.slowQueries, 1)
	}

	if isError {
		atomic.AddInt64(&m.failedQueries, 1)
	}

	m.mu.Lock()
	m.queryDurations = append(m.queryDurations, duration)
	if len(m.queryDurations) > 1000 {
		m.queryDurations = m.queryDurations[len(m.queryDurations)-500:]
	}
	if duration > m.maxDuration {
		m.maxDuration = duration
	}
	m.mu.Unlock()
}

func (m *MetricsCollector) RecordWrite(duration time.Duration, operation, tableName string) {
	atomic.AddInt64(&m.writeOps, 1)

	m.mu.Lock()
	m.queryDurations = append(m.queryDurations, duration)
	if duration > m.maxDuration {
		m.maxDuration = duration
	}
	m.mu.Unlock()
}

func (m *MetricsCollector) RecordCacheHit() {
	atomic.AddInt64(&m.cacheHits, 1)
}

func (m *MetricsCollector) RecordCacheMiss() {
	atomic.AddInt64(&m.cacheMisses, 1)
}

func (m *MetricsCollector) GetMetrics() *OptimizerMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := &OptimizerMetrics{}

	metrics.TotalQueries = atomic.LoadInt64(&m.totalQueries)
	metrics.SlowQueries = atomic.LoadInt64(&m.slowQueries)
	metrics.FailedQueries = atomic.LoadInt64(&m.failedQueries)
	metrics.WriteOperations = atomic.LoadInt64(&m.writeOps)
	metrics.CacheHits = atomic.LoadInt64(&m.cacheHits)
	metrics.CacheMisses = atomic.LoadInt64(&m.cacheMisses)
	metrics.MaxQueryDuration = m.maxDuration

	if len(m.queryDurations) > 0 {
		var totalDuration time.Duration
		for _, d := range m.queryDurations {
			totalDuration += d
		}
		metrics.AvgQueryDuration = totalDuration / time.Duration(len(m.queryDurations))
	}

	totalCache := metrics.CacheHits + metrics.CacheMisses
	if totalCache > 0 {
		metrics.CacheHitRate = float64(metrics.CacheHits) / float64(totalCache) * 100
	}

	return metrics
}

func (m *MetricsCollector) Reset() {
	atomic.StoreInt64(&m.totalQueries, 0)
	atomic.StoreInt64(&m.slowQueries, 0)
	atomic.StoreInt64(&m.failedQueries, 0)
	atomic.StoreInt64(&m.writeOps, 0)
	atomic.StoreInt64(&m.cacheHits, 0)
	atomic.StoreInt64(&m.cacheMisses, 0)

	m.mu.Lock()
	m.queryDurations = make([]time.Duration, 0)
	m.maxDuration = 0
	m.mu.Unlock()
}

type OptimizerMetrics struct {
	TotalQueries       int64
	SlowQueries        int64
	FailedQueries      int64
	WriteOperations    int64
	CacheHits          int64
	CacheMisses        int64
	CacheHitRate       float64
	AvgQueryDuration   time.Duration
	MaxQueryDuration   time.Duration
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (o *DBOptimizer) SoftDelete(tableName string, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	return o.db.Model(&models.User{}).
		Where("id IN ?", ids).
		Update("deleted_at", time.Now()).Error
}

func (o *DBOptimizer) Restore(tableName string, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	return o.db.Model(&models.User{}).
		Where("id IN ?", ids).
		Update("deleted_at", nil).Error
}

func (o *DBOptimizer) BulkSoftDelete(tableName string, where map[string]interface{}) (int64, error) {
	result := o.db.Table(tableName).Where(where).Update("deleted_at", time.Now())
	return result.RowsAffected, result.Error
}
