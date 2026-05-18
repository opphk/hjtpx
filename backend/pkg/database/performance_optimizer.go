package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

type PerformanceOptimizer struct {
	db            *gorm.DB
	config        *config.Config
	indexReady    bool
	indexMu       sync.RWMutex
	preparedStmts map[string]*gorm.DB
	stmtMu        sync.RWMutex
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

type TableFragmentation struct {
	TableName      string  `json:"table_name"`
	TotalSize      string  `json:"total_size"`
	TableSize      string  `json:"table_size"`
	IndexSize      string  `json:"index_size"`
	DeadTuples     int64   `json:"dead_tuples"`
	LiveTuples     int64   `json:"live_tuples"`
	FragmentationRatio float64 `json:"fragmentation_ratio"`
	LastVacuum     time.Time `json:"last_vacuum"`
	LastAutoVacuum time.Time `json:"last_auto_vacuum"`
	LastAnalyze    time.Time `json:"last_analyze"`
}

func (o *PerformanceOptimizer) AnalyzeTableFragmentation(ctx context.Context, threshold float64) ([]TableFragmentation, error) {
	var fragmented []TableFragmentation

	err := o.db.WithContext(ctx).Raw(`
		SELECT
			pg_class.relname AS table_name,
			pg_size_pretty(pg_total_relation_size(pg_class.oid)) AS total_size,
			pg_size_pretty(pg_relation_size(pg_class.oid)) AS table_size,
			pg_size_pretty(pg_indexes_size(pg_class.oid)) AS index_size,
			pg_stat_user_tables.n_dead_tup AS dead_tuples,
			pg_stat_user_tables.n_live_tup AS live_tuples,
			CASE
				WHEN pg_stat_user_tables.n_live_tup = 0 THEN 0.0
				ELSE ROUND(pg_stat_user_tables.n_dead_tup::numeric / NULLIF(pg_stat_user_tables.n_live_tup + pg_stat_user_tables.n_dead_tup, 0) * 100, 2)
			END AS fragmentation_ratio,
			pg_stat_user_tables.last_vacuum,
			pg_stat_user_tables.last_autovacuum,
			pg_stat_user_tables.last_analyze
		FROM pg_class
		JOIN pg_stat_user_tables ON pg_class.relname = pg_stat_user_tables.relname
		JOIN pg_namespace ON pg_class.relnamespace = pg_namespace.oid
		WHERE pg_namespace.nspname = 'public'
			AND pg_class.relkind = 'r'
			AND CASE
				WHEN pg_stat_user_tables.n_live_tup = 0 THEN 0.0
				ELSE ROUND(pg_stat_user_tables.n_dead_tup::numeric / NULLIF(pg_stat_user_tables.n_live_tup + pg_stat_user_tables.n_dead_tup, 0) * 100, 2)
			END >= $1
		ORDER BY fragmentation_ratio DESC
	`, threshold).Scan(&fragmented).Error

	return fragmented, err
}

func (o *PerformanceOptimizer) AnalyzeAllTableFragmentation(ctx context.Context) ([]TableFragmentation, error) {
	return o.AnalyzeTableFragmentation(ctx, 0)
}

func (o *PerformanceOptimizer) VacuumTable(ctx context.Context, tableName string, full bool, analyze bool) error {
	query := "VACUUM"
	if full {
		query += " FULL"
	}
	query += " " + tableName
	if analyze {
		query += " ANALYZE"
	}

	return o.db.WithContext(ctx).Exec(query).Error
}

func (o *PerformanceOptimizer) VacuumFragmentedTables(ctx context.Context, threshold float64, full bool) error {
	fragmented, err := o.AnalyzeTableFragmentation(ctx, threshold)
	if err != nil {
		return err
	}

	if len(fragmented) == 0 {
		log.Println("[PERFORMANCE_OPTIMIZER] No fragmented tables found")
		return nil
	}

	log.Printf("[PERFORMANCE_OPTIMIZER] Found %d fragmented tables with fragmentation >= %.2f%%", len(fragmented), threshold)

	for _, table := range fragmented {
		log.Printf("[PERFORMANCE_OPTIMIZER] Vacuuming table: %s (fragmentation: %.2f%%)", table.TableName, table.FragmentationRatio)
		if err := o.VacuumTable(ctx, table.TableName, full, true); err != nil {
			log.Printf("[PERFORMANCE_OPTIMIZER] Failed to vacuum table %s: %v", table.TableName, err)
			continue
		}
		log.Printf("[PERFORMANCE_OPTIMIZER] Successfully vacuumed table: %s", table.TableName)
	}

	return nil
}

func (o *PerformanceOptimizer) ReindexTable(ctx context.Context, tableName string, concurrently bool) error {
	query := "REINDEX"
	if concurrently {
		query += " CONCURRENTLY"
	}
	query += " TABLE " + tableName

	return o.db.WithContext(ctx).Exec(query).Error
}

func (o *PerformanceOptimizer) ReindexAllTables(ctx context.Context, concurrently bool) error {
	var tables []string

	err := o.db.WithContext(ctx).Raw(`
		SELECT relname
		FROM pg_class
		JOIN pg_namespace ON pg_class.relnamespace = pg_namespace.oid
		WHERE pg_namespace.nspname = 'public'
			AND pg_class.relkind = 'r'
	`).Scan(&tables).Error
	if err != nil {
		return err
	}

	log.Printf("[PERFORMANCE_OPTIMIZER] Reindexing %d tables", len(tables))

	for _, table := range tables {
		log.Printf("[PERFORMANCE_OPTIMIZER] Reindexing table: %s", table)
		if err := o.ReindexTable(ctx, table, concurrently); err != nil {
			log.Printf("[PERFORMANCE_OPTIMIZER] Failed to reindex table %s: %v", table, err)
			continue
		}
		log.Printf("[PERFORMANCE_OPTIMIZER] Successfully reindexed table: %s", table)
	}

	return nil
}

func (o *PerformanceOptimizer) AutoMaintainFragmentation(ctx context.Context, vacuumThreshold, reindexThreshold float64) error {
	fragmented, err := o.AnalyzeTableFragmentation(ctx, vacuumThreshold)
	if err != nil {
		return err
	}

	for _, table := range fragmented {
		if table.FragmentationRatio >= reindexThreshold {
			log.Printf("[PERFORMANCE_OPTIMIZER] Table %s has high fragmentation (%.2f%%), reindexing", table.TableName, table.FragmentationRatio)
			if err := o.ReindexTable(ctx, table.TableName, true); err != nil {
				log.Printf("[PERFORMANCE_OPTIMIZER] Failed to reindex table %s, falling back to vacuum", table.TableName)
				if err := o.VacuumTable(ctx, table.TableName, true, true); err != nil {
					log.Printf("[PERFORMANCE_OPTIMIZER] Failed to vacuum table %s: %v", table.TableName, err)
				}
			}
		} else {
			log.Printf("[PERFORMANCE_OPTIMIZER] Vacuuming table %s (fragmentation: %.2f%%)", table.TableName, table.FragmentationRatio)
			if err := o.VacuumTable(ctx, table.TableName, false, true); err != nil {
				log.Printf("[PERFORMANCE_OPTIMIZER] Failed to vacuum table %s: %v", table.TableName, err)
			}
		}
	}

	return nil
}

type FragmentationReport struct {
	Timestamp       time.Time           `json:"timestamp"`
	TotalTables     int                 `json:"total_tables"`
	FragmentedTables int                `json:"fragmented_tables"`
	AverageFragmentation float64        `json:"average_fragmentation"`
	MaxFragmentation float64            `json:"max_fragmentation"`
	MinFragmentation float64            `json:"min_fragmentation"`
	Details         []TableFragmentation `json:"details"`
}

func (o *PerformanceOptimizer) GenerateFragmentationReport(ctx context.Context) (*FragmentationReport, error) {
	tables, err := o.AnalyzeAllTableFragmentation(ctx)
	if err != nil {
		return nil, err
	}

	report := &FragmentationReport{
		Timestamp:       time.Now(),
		TotalTables:     len(tables),
		FragmentedTables: 0,
		Details:         tables,
	}

	if len(tables) > 0 {
		report.MaxFragmentation = tables[0].FragmentationRatio
		report.MinFragmentation = tables[0].FragmentationRatio

		var totalFragmentation float64
		fragmentedCount := 0

		for _, table := range tables {
			totalFragmentation += table.FragmentationRatio

			if table.FragmentationRatio > report.MaxFragmentation {
				report.MaxFragmentation = table.FragmentationRatio
			}
			if table.FragmentationRatio < report.MinFragmentation {
				report.MinFragmentation = table.FragmentationRatio
			}
			if table.FragmentationRatio > 10 {
				fragmentedCount++
			}
		}

		report.AverageFragmentation = totalFragmentation / float64(len(tables))
		report.FragmentedTables = fragmentedCount
	}

	return report, nil
}

func (o *PerformanceOptimizer) GetTableStatistics(ctx context.Context, tableName string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	var stats struct {
		TableName      string
		TotalSize      string
		TableSize      string
		IndexSize      string
		RowCount       int64
		DeadTuples     int64
		LiveTuples     int64
		Fragmentation  float64
		LastVacuum     time.Time
		LastAnalyze    time.Time
		IndexCount     int
	}

	err := o.db.WithContext(ctx).Raw(`
		SELECT
			pc.relname AS table_name,
			pg_size_pretty(pg_total_relation_size(pc.oid)) AS total_size,
			pg_size_pretty(pg_relation_size(pc.oid)) AS table_size,
			pg_size_pretty(pg_indexes_size(pc.oid)) AS index_size,
			(SELECT reltuples FROM pg_class WHERE oid = pc.oid) AS row_count,
			COALESCE(pst.n_dead_tup, 0) AS dead_tuples,
			COALESCE(pst.n_live_tup, 0) AS live_tuples,
			CASE
				WHEN COALESCE(pst.n_live_tup, 0) = 0 THEN 0.0
				ELSE ROUND(COALESCE(pst.n_dead_tup, 0)::numeric / NULLIF(COALESCE(pst.n_live_tup, 0) + COALESCE(pst.n_dead_tup, 0), 0) * 100, 2)
			END AS fragmentation,
			pst.last_vacuum,
			pst.last_analyze,
			(SELECT COUNT(*) FROM pg_index WHERE indrelid = pc.oid) AS index_count
		FROM pg_class pc
		LEFT JOIN pg_stat_user_tables pst ON pc.relname = pst.relname
		JOIN pg_namespace pn ON pc.relnamespace = pn.oid
		WHERE pn.nspname = 'public' AND pc.relname = $1
	`, tableName).Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	result["table_name"] = stats.TableName
	result["total_size"] = stats.TotalSize
	result["table_size"] = stats.TableSize
	result["index_size"] = stats.IndexSize
	result["row_count"] = stats.RowCount
	result["dead_tuples"] = stats.DeadTuples
	result["live_tuples"] = stats.LiveTuples
	result["fragmentation_ratio"] = stats.Fragmentation
	result["last_vacuum"] = stats.LastVacuum
	result["last_analyze"] = stats.LastAnalyze
	result["index_count"] = stats.IndexCount

	var recommendations []string
	if stats.Fragmentation > 30 {
		recommendations = append(recommendations, "建议执行 VACUUM FULL 或 REINDEX")
	} else if stats.Fragmentation > 10 {
		recommendations = append(recommendations, "建议执行 VACUUM ANALYZE")
	}
	if stats.LastAnalyze.IsZero() || time.Since(stats.LastAnalyze) > 7*24*time.Hour {
		recommendations = append(recommendations, "建议执行 ANALYZE 更新统计信息")
	}
	if stats.RowCount > 1000000 {
		recommendations = append(recommendations, "表数据量较大，建议考虑分区")
	}

	result["recommendations"] = recommendations

	return result, nil
}
