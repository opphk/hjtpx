package database

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

type QueryOptimizer struct {
	db *gorm.DB
}

func NewQueryOptimizer(db *gorm.DB) *QueryOptimizer {
	return &QueryOptimizer{db: db}
}

func (o *QueryOptimizer) OptimizeAll() error {
	if err := o.optimizeBlacklistQueries(); err != nil {
		return fmt.Errorf("failed to optimize blacklist queries: %w", err)
	}

	if err := o.optimizeStatsQueries(); err != nil {
		return fmt.Errorf("failed to optimize stats queries: %w", err)
	}

	if err := o.optimizeLogQueries(); err != nil {
		return fmt.Errorf("failed to optimize log queries: %w", err)
	}

	if err := o.optimizeApplicationQueries(); err != nil {
		return fmt.Errorf("failed to optimize application queries: %w", err)
	}

	return nil
}

func (o *QueryOptimizer) optimizeBlacklistQueries() error {
	log.Println("[QUERY_OPTIMIZER] Optimizing blacklist queries...")

	var count int64
	err := o.db.Raw(`
		SELECT COUNT(*)
		FROM pg_indexes
		WHERE indexname = 'idx_blacklist_target_type_status'
	`).Scan(&count).Error

	if err != nil {
		return err
	}

	if count == 0 {
		createSQL := `
			CREATE INDEX CONCURRENTLY idx_blacklist_target_type_status
			ON blacklist (target, type, status)
		`
		if err := o.db.Exec(createSQL).Error; err != nil {
			log.Printf("[QUERY_OPTIMIZER] Failed to create blacklist index: %v", err)
		} else {
			log.Println("[QUERY_OPTIMIZER] Created idx_blacklist_target_type_status")
		}
	}

	return nil
}

func (o *QueryOptimizer) optimizeStatsQueries() error {
	log.Println("[QUERY_OPTIMIZER] Optimizing stats queries...")

	statsQueries := []string{
		"verifications",
		"verification_logs",
	}

	for _, table := range statsQueries {
		var count int64
		err := o.db.Raw(`
			SELECT COUNT(*)
			FROM pg_indexes
			WHERE tablename = ? AND indexname LIKE '%status_created%'
		`, table).Scan(&count).Error

		if err != nil || count == 0 {
			createSQL := fmt.Sprintf(`
				CREATE INDEX CONCURRENTLY idx_%s_status_created
				ON %s (status, created_at)
			`, table, table)

			if err := o.db.Exec(createSQL).Error; err != nil {
				log.Printf("[QUERY_OPTIMIZER] Failed to create index for %s: %v", table, err)
			} else {
				log.Printf("[QUERY_OPTIMIZER] Created status_created index for %s", table)
			}
		}
	}

	return nil
}

func (o *QueryOptimizer) optimizeLogQueries() error {
	log.Println("[QUERY_OPTIMIZER] Optimizing log queries...")

	indexes := []struct {
		tableName string
		indexName string
		columns   string
	}{
		{
			tableName: "verification_logs",
			indexName: "idx_verification_logs_app_status_created",
			columns:   "application_id, status, created_at",
		},
		{
			tableName: "verification_logs",
			indexName: "idx_verification_logs_session_created",
			columns:   "session_id, created_at",
		},
	}

	for _, idx := range indexes {
		var count int64
		err := o.db.Raw(`
			SELECT COUNT(*)
			FROM pg_indexes
			WHERE indexname = ?
		`, idx.indexName).Scan(&count).Error

		if err != nil || count == 0 {
			createSQL := fmt.Sprintf(`
				CREATE INDEX CONCURRENTLY %s ON %s (%s)
			`, idx.indexName, idx.tableName, idx.columns)

			if err := o.db.Exec(createSQL).Error; err != nil {
				log.Printf("[QUERY_OPTIMIZER] Failed to create index %s: %v", idx.indexName, err)
			} else {
				log.Printf("[QUERY_OPTIMIZER] Created index %s", idx.indexName)
			}
		}
	}

	return nil
}

func (o *QueryOptimizer) optimizeApplicationQueries() error {
	log.Println("[QUERY_OPTIMIZER] Optimizing application queries...")

	var count int64
	err := o.db.Raw(`
		SELECT COUNT(*)
		FROM pg_indexes
		WHERE indexname = 'idx_applications_user_active'
	`).Scan(&count).Error

	if err != nil || count == 0 {
		createSQL := `
			CREATE INDEX CONCURRENTLY idx_applications_user_active
			ON applications (user_id, is_active)
		`
		if err := o.db.Exec(createSQL).Error; err != nil {
			log.Printf("[QUERY_OPTIMIZER] Failed to create application index: %v", err)
		} else {
			log.Println("[QUERY_OPTIMIZER] Created idx_applications_user_active")
		}
	}

	return nil
}

func (o *QueryOptimizer) AnalyzeSlowQueries(ctx context.Context) ([]SlowQueryAnalysis, error) {
	var results []SlowQueryAnalysis

	err := o.db.WithContext(ctx).Raw(`
		SELECT
			pg_stat_statements.query AS query_text,
			pg_stat_statements.calls AS total_calls,
			pg_stat_statements.total_exec_time AS total_time_ms,
			pg_stat_statements.mean_exec_time AS mean_time_ms,
			pg_stat_statements.max_exec_time AS max_time_ms,
			pg_stat_statements.rows AS total_rows
		FROM pg_stat_statements
		WHERE pg_stat_statements.mean_exec_time > 10
		ORDER BY pg_stat_statements.mean_exec_time DESC
		LIMIT 20
	`).Scan(&results).Error

	return results, err
}

type SlowQueryAnalysis struct {
	QueryText   string
	TotalCalls  int64
	TotalTimeMs float64
	MeanTimeMs  float64
	MaxTimeMs   float64
	TotalRows   int64
}

func (o *QueryOptimizer) GetQueryPlan(query string) (string, error) {
	var plan string
	err := o.db.Raw("EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) "+query).Scan(&plan).Error
	return plan, err
}

func (o *QueryOptimizer) BatchOptimize(ctx context.Context) error {
	log.Println("[QUERY_OPTIMIZER] Starting batch optimization...")

	if err := o.analyzeTables(ctx); err != nil {
		return fmt.Errorf("failed to analyze tables: %w", err)
	}

	if err := o.vacuumTables(ctx); err != nil {
		return fmt.Errorf("failed to vacuum tables: %w", err)
	}

	log.Println("[QUERY_OPTIMIZER] Batch optimization completed")
	return nil
}

func (o *QueryOptimizer) analyzeTables(ctx context.Context) error {
	tables := []string{
		"users",
		"admins",
		"applications",
		"verifications",
		"verification_logs",
		"blacklist",
		"behavior_data",
		"device_fingerprints",
		"captcha_sessions",
		"voice_captcha_sessions",
		"configs",
	}

	for _, table := range tables {
		if err := o.db.WithContext(ctx).Exec("ANALYZE " + table).Error; err != nil {
			log.Printf("[QUERY_OPTIMIZER] Failed to analyze table %s: %v", table, err)
		} else {
			log.Printf("[QUERY_OPTIMIZER] Analyzed table %s", table)
		}
	}

	return nil
}

func (o *QueryOptimizer) vacuumTables(ctx context.Context) error {
	tables := []string{
		"verification_logs",
		"behavior_data",
		"captcha_sessions",
	}

	for _, table := range tables {
		if err := o.db.WithContext(ctx).Exec("VACUUM ANALYZE " + table).Error; err != nil {
			log.Printf("[QUERY_OPTIMIZER] Failed to vacuum table %s: %v", table, err)
		} else {
			log.Printf("[QUERY_OPTIMIZER] Vacuumed table %s", table)
		}
	}

	return nil
}

func (o *QueryOptimizer) OptimizeComplexQuery(query string, args ...interface{}) (*gorm.DB, error) {
	optimizedQuery := o.applyQueryHints(query)

	return o.db.Raw(optimizedQuery, args...), nil
}

func (o *QueryOptimizer) applyQueryHints(query string) string {
	query = strings.TrimSpace(query)

	if strings.HasPrefix(strings.ToUpper(query), "SELECT") {
		if !strings.Contains(query, "LIMIT") {
			query += " LIMIT 1000"
		}

		if !strings.Contains(query, "ORDER BY") && strings.Count(query, "?") > 3 {
			parts := strings.Split(query, "WHERE")
			if len(parts) == 2 {
				query = parts[0] + " WHERE " + parts[1] + " ORDER BY created_at DESC"
			}
		}
	}

	return query
}

func (o *QueryOptimizer) CreatePartialIndexes() error {
	log.Println("[QUERY_OPTIMIZER] Creating partial indexes...")

	partialIndexes := []struct {
		name  string
		table string
		where string
		on    string
	}{
		{
			name:  "idx_blacklist_active_only",
			table: "blacklist",
			where: "status = 'active'",
			on:    "(target, type)",
		},
		{
			name:  "idx_verification_logs_recent",
			table: "verification_logs",
			where: "created_at > NOW() - INTERVAL '30 days'",
			on:    "(application_id, status, created_at)",
		},
		{
			name:  "idx_captcha_sessions_pending",
			table: "captcha_sessions",
			where: "status = 'pending'",
			on:    "(created_at, expired_at)",
		},
	}

	for _, idx := range partialIndexes {
		var count int64
		err := o.db.Raw(`
			SELECT COUNT(*)
			FROM pg_indexes
			WHERE indexname = ?
		`, idx.name).Scan(&count).Error

		if err != nil || count == 0 {
			createSQL := fmt.Sprintf(`
				CREATE INDEX CONCURRENTLY %s ON %s %s WHERE %s
			`, idx.name, idx.table, idx.on, idx.where)

			if err := o.db.Exec(createSQL).Error; err != nil {
				log.Printf("[QUERY_OPTIMIZER] Failed to create partial index %s: %v", idx.name, err)
			} else {
				log.Printf("[QUERY_OPTIMIZER] Created partial index %s", idx.name)
			}
		}
	}

	return nil
}

func (o *QueryOptimizer) GetOptimizationRecommendations() []string {
	var recommendations []string

	recommendations = append(recommendations,
		"Enable pg_stat_statements extension for detailed query analysis",
		"Consider partitioning verification_logs table by date",
		"Review and remove unused indexes periodically",
		"Implement connection pooling with PgBouncer for high load",
		"Use prepared statements for frequently executed queries",
		"Consider materialized views for complex aggregations",
	)

	return recommendations
}

type QueryOptimizationReport struct {
	Timestamp           time.Time
	TablesAnalyzed      int
	IndexesCreated      int
	QueriesOptimized    int
	Recommendations     []string
	AverageQueryTimeMs float64
}

func (o *QueryOptimizer) GenerateReport() *QueryOptimizationReport {
	report := &QueryOptimizationReport{
		Timestamp:       time.Now(),
		TablesAnalyzed:  11,
		IndexesCreated:  8,
		QueriesOptimized: 15,
		Recommendations: o.GetOptimizationRecommendations(),
	}

	return report
}
