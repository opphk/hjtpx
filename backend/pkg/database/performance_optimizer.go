package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

type PerformanceOptimizer struct {
	db             *gorm.DB
	config         *config.Config
	indexReady     bool
	indexMu        sync.RWMutex
	preparedStmts  map[string]*gorm.DB
	stmtMu         sync.RWMutex
}

var globalOptimizer *PerformanceOptimizer
var optimizerOnce sync.Once

func NewPerformanceOptimizer(db *gorm.DB, cfg *config.Config) *PerformanceOptimizer {
	return &PerformanceOptimizer{
		db:            db,
		config:        cfg,
		preparedStmts: make(map[string]*gorm.DB),
	}
}

func GetPerformanceOptimizer() *PerformanceOptimizer {
	return globalOptimizer
}

func InitPerformanceOptimizer(db *gorm.DB, cfg *config.Config) {
	optimizerOnce.Do(func() {
		globalOptimizer = NewPerformanceOptimizer(db, cfg)
	})
}

func (o *PerformanceOptimizer) OptimizeAll() error {
	if err := o.CreateOptimizedIndexes(); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	if err := o.ConfigureConnectionPool(); err != nil {
		return fmt.Errorf("failed to configure connection pool: %w", err)
	}

	if err := o.EnableQueryStatistics(); err != nil {
		return fmt.Errorf("failed to enable query stats: %w", err)
	}

	o.indexMu.Lock()
	o.indexReady = true
	o.indexMu.Unlock()

	return nil
}

func (o *PerformanceOptimizer) CreateOptimizedIndexes() error {
	indexes := []struct {
		TableName string
		Columns   []string
		IndexName string
		Unique    bool
		Where     string
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
			Columns:   []string{"status", "created_at"},
			IndexName: "idx_users_status_created",
			Unique:    false,
		},
		{
			TableName: "applications",
			Columns:   []string{"api_key"},
			IndexName: "idx_applications_api_key",
			Unique:    true,
		},
		{
			TableName: "applications",
			Columns:   []string{"user_id", "is_active", "created_at"},
			IndexName: "idx_applications_user_active_created",
			Unique:    false,
		},
		{
			TableName: "verifications",
			Columns:   []string{"session_id"},
			IndexName: "idx_verifications_session_id",
			Unique:    true,
		},
		{
			TableName: "verifications",
			Columns:   []string{"application_id", "created_at", "status"},
			IndexName: "idx_verifications_app_created_status",
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
			Columns:   []string{"ip_address", "created_at"},
			IndexName: "idx_verifications_ip_created",
			Unique:    false,
		},
		{
			TableName: "verifications",
			Columns:   []string{"status", "created_at"},
			IndexName: "idx_verifications_status_created",
			Unique:    false,
		},
		{
			TableName: "blacklist",
			Columns:   []string{"target", "type", "status"},
			IndexName: "idx_blacklist_target_type_status",
			Unique:    false,
		},
		{
			TableName: "blacklist",
			Columns:   []string{"status", "expires_at"},
			IndexName: "idx_blacklist_status_expires",
			Unique:    false,
		},
		{
			TableName: "verification_logs",
			Columns:   []string{"session_id", "created_at"},
			IndexName: "idx_verification_logs_session_created",
			Unique:    false,
		},
		{
			TableName: "verification_logs",
			Columns:   []string{"application_id", "created_at"},
			IndexName: "idx_verification_logs_app_created",
			Unique:    false,
		},
		{
			TableName: "device_fingerprints",
			Columns:   []string{"fingerprint"},
			IndexName: "idx_device_fingerprint",
			Unique:    true,
		},
		{
			TableName: "device_fingerprints",
			Columns:   []string{"ip_address", "last_seen"},
			IndexName: "idx_device_ip_last_seen",
			Unique:    false,
		},
		{
			TableName: "api_key_histories",
			Columns:   []string{"application_id", "changed_at"},
			IndexName: "idx_api_key_histories_app_changed",
			Unique:    false,
		},
	}

	for _, idx := range indexes {
		if err := o.createIndexIfNotExists(idx.TableName, idx.Columns, idx.IndexName, idx.Unique, idx.Where); err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.IndexName, err)
		}
	}

	return nil
}

func (o *PerformanceOptimizer) createIndexIfNotExists(tableName string, columns []string, indexName string, unique bool, whereClause string) error {
	var count int64
	err := o.db.Raw(`
		SELECT COUNT(*) 
		FROM pg_indexes 
		WHERE indexname = ?
	`, indexName).Scan(&count).Error

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

	columnsStr := ""
	for i, col := range columns {
		if i > 0 {
			columnsStr += ", "
		}
		columnsStr += col
	}

	createSQL := fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)", uniqueStr, indexName, tableName, columnsStr)
	
	if whereClause != "" {
		createSQL += " WHERE " + whereClause
	}

	return o.db.Exec(createSQL).Error
}

func (o *PerformanceOptimizer) ConfigureConnectionPool() error {
	sqlDB, err := o.db.DB()
	if err != nil {
		return err
	}

	maxOpenConns := o.config.Postgres.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = 100
	}

	maxIdleConns := o.config.Postgres.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 20
	}

	connMaxLifetime := time.Duration(o.config.Postgres.ConnMaxLifetime) * time.Second
	if connMaxLifetime <= 0 {
		connMaxLifetime = 30 * time.Minute
	}

	connMaxIdleTime := 10 * time.Minute

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	return nil
}

func (o *PerformanceOptimizer) EnableQueryStatistics() error {
	if err := o.db.Callback().Query().Before("gorm:query").Register("performance_stats_before", func(db *gorm.DB) {
		db.InstanceSet("start_time", time.Now())
	}); err != nil {
		return err
	}

	if err := o.db.Callback().Query().After("gorm:query").Register("performance_stats_after", func(db *gorm.DB) {
		startTime, _ := db.InstanceGet("start_time")
		if t, ok := startTime.(time.Time); ok {
			duration := time.Since(t)
			slowThreshold := time.Duration(o.config.Database.SlowQueryThresholdMs) * time.Millisecond
			if duration > slowThreshold {
				sql := db.Statement.SQL.String()
				fmt.Printf("[SLOW_QUERY] %v - %s\n", duration, sql)
			}
		}
	}); err != nil {
		return err
	}

	return nil
}

func (o *PerformanceOptimizer) OptimizeQuery(tableName string, columns []string, where map[string]interface{}, orderBy string, limit, offset int) *gorm.DB {
	query := o.db.Table(tableName)
	
	if len(columns) > 0 {
		query = query.Select(columns)
	}
	
	if len(where) > 0 {
		query = query.Where(where)
	}
	
	if orderBy != "" {
		query = query.Order(orderBy)
	}
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	return query
}

func (o *PerformanceOptimizer) BatchInsert(ctx context.Context, tableName string, records []interface{}, batchSize int) error {
	if len(records) == 0 {
		return nil
	}

	if batchSize <= 0 {
		batchSize = 100
	}

	return o.db.WithContext(ctx).CreateInBatches(records, batchSize).Error
}

type ConnectionStats struct {
	MaxOpenConnections int           `json:"max_open_connections"`
	OpenConnections    int           `json:"open_connections"`
	InUse              int           `json:"in_use"`
	Idle               int           `json:"idle"`
	WaitCount          int64         `json:"wait_count"`
	WaitDuration       time.Duration `json:"wait_duration"`
	MaxIdleClosed      int64         `json:"max_idle_closed"`
	MaxLifetimeClosed  int64         `json:"max_lifetime_closed"`
}

func (o *PerformanceOptimizer) GetConnectionStats() (*ConnectionStats, error) {
	sqlDB, err := o.db.DB()
	if err != nil {
		return nil, err
	}

	stats := sqlDB.Stats()
	return &ConnectionStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}, nil
}

func (o *PerformanceOptimizer) AnalyzeTables(ctx context.Context) error {
	tables := []string{
		"users",
		"admins",
		"applications",
		"api_key_histories",
		"verifications",
		"behavior_data",
		"blacklist",
		"verification_logs",
		"device_fingerprints",
	}

	for _, table := range tables {
		if err := o.db.WithContext(ctx).Exec("ANALYZE " + table).Error; err != nil {
			return fmt.Errorf("failed to analyze table %s: %w", table, err)
		}
	}

	return nil
}

func (o *PerformanceOptimizer) VacuumAnalyze(ctx context.Context) error {
	return o.db.WithContext(ctx).Exec("VACUUM ANALYZE").Error
}
