package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type QueryOptimizer struct {
	enablePreparedStatements bool
	slowQueryThreshold      time.Duration
	enableQueryCache        bool
	queryCacheTTL           time.Duration
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
		enableQueryCache:        cfg.Database.QueryOptimization.EnableQueryCache,
		queryCacheTTL:           time.Duration(cfg.Database.QueryOptimization.QueryCacheTTLSecs) * time.Second,
	}

	for _, opt := range opts {
		opt(qo)
	}

	return qo
}

type QueryMetrics struct {
	TotalQueries  int64
	SlowQueries   int64
	FailedQueries int64
	AvgDuration   time.Duration
	MaxDuration   time.Duration
	MinDuration   time.Duration
	CacheHits     int64
	CacheMisses   int64
}

type QueryMetricsCollector struct {
	mu            sync.RWMutex
	totalQueries  int64
	slowQueries   int64
	failedQueries int64
	cacheHits    int64
	cacheMisses  int64
	durations    []time.Duration
}

var queryMetrics = &QueryMetricsCollector{
	durations: make([]time.Duration, 0),
}

func (q *QueryOptimizer) OptimizeQuery(query string) string {
	optimized := query

	optimized = strings.ReplaceAll(optimized, "SELECT *", "SELECT")
	optimized = strings.ReplaceAll(optimized, "select *", "select")

	optimized = strings.ReplaceAll(optimized, "OR 1=1", "")
	optimized = strings.ReplaceAll(optimized, "or 1=1", "")
	optimized = strings.ReplaceAll(optimized, "OR '1'='1'", "")
	optimized = strings.ReplaceAll(optimized, "or '1'='1'", "")

	return optimized
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
	queryMetrics.durations = nil
}

type DBOptimizer struct {
	db             *gorm.DB
	queryOptimizer *QueryOptimizer
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

			if duration > o.queryOptimizer.slowQueryThreshold {
				fmt.Printf("[SLOW_QUERY] Duration: %v, Query: %s\n", duration, sql)
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
