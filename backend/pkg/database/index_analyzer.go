package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

type IndexAnalyzer struct {
	db *gorm.DB
}

func NewIndexAnalyzer(db *gorm.DB) *IndexAnalyzer {
	return &IndexAnalyzer{db: db}
}

func (a *IndexAnalyzer) AnalyzeAndCreateMissingIndexes() error {
	missingIndexes := a.findMissingIndexes()

	for _, idx := range missingIndexes {
		log.Printf("[INDEX_ANALYZER] Creating missing index: %s on %s (%v)",
			idx.IndexName, idx.TableName, idx.Columns)

		if err := a.createIndex(idx); err != nil {
			log.Printf("[INDEX_ANALYZER] Failed to create index %s: %v", idx.IndexName, err)
			continue
		}

		log.Printf("[INDEX_ANALYZER] Successfully created index: %s", idx.IndexName)
	}

	return nil
}

type MissingIndex struct {
	TableName  string
	IndexName  string
	Columns    []string
	QueryCount int64
	Priority   string
}

func (a *IndexAnalyzer) findMissingIndexes() []MissingIndex {
	var missing []MissingIndex

	missing = append(missing,
		MissingIndex{
			TableName:  "captcha_sessions",
			IndexName: "idx_captcha_sessions_status_expired",
			Columns:   []string{"status", "expired_at"},
			QueryCount: 1000,
			Priority:   "high",
		},
		MissingIndex{
			TableName:  "captcha_sessions",
			IndexName: "idx_captcha_sessions_created",
			Columns:   []string{"created_at"},
			QueryCount: 500,
			Priority:   "medium",
		},
		MissingIndex{
			TableName:  "blacklist",
			IndexName: "idx_blacklist_target_type_status",
			Columns:   []string{"target", "type", "status"},
			QueryCount: 10000,
			Priority:   "high",
		},
		MissingIndex{
			TableName:  "blacklist",
			IndexName: "idx_blacklist_created_status",
			Columns:   []string{"created_at", "status"},
			QueryCount: 1000,
			Priority:   "medium",
		},
		MissingIndex{
			TableName:  "applications",
			IndexName: "idx_applications_name_search",
			Columns:   []string{"name", "is_active"},
			QueryCount: 2000,
			Priority:   "medium",
		},
		MissingIndex{
			TableName:  "verification_logs",
			IndexName: "idx_verification_logs_app_status_created",
			Columns:   []string{"application_id", "status", "created_at"},
			QueryCount: 5000,
			Priority:   "high",
		},
		MissingIndex{
			TableName:  "verification_logs",
			IndexName: "idx_verification_logs_created_status",
			Columns:   []string{"created_at", "status"},
			QueryCount: 3000,
			Priority:   "high",
		},
		MissingIndex{
			TableName:  "configs",
			IndexName: "idx_configs_group_visible",
			Columns:   []string{"group", "is_visible"},
			QueryCount: 500,
			Priority:   "medium",
		},
		MissingIndex{
			TableName:  "users",
			IndexName: "idx_users_status_created",
			Columns:   []string{"status", "created_at"},
			QueryCount: 1000,
			Priority:   "medium",
		},
	)

	return missing
}

func (a *IndexAnalyzer) createIndex(idx MissingIndex) error {
	var count int64
	err := a.db.Raw(`
		SELECT COUNT(*)
		FROM pg_indexes
		WHERE indexname = ?
	`, idx.IndexName).Scan(&count).Error

	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}

	if count > 0 {
		log.Printf("[INDEX_ANALYZER] Index %s already exists, skipping", idx.IndexName)
		return nil
	}

	columnsStr := ""
	for i, col := range idx.Columns {
		if i > 0 {
			columnsStr += ", "
		}
		columnsStr += col
	}

	createSQL := fmt.Sprintf("CREATE INDEX CONCURRENTLY %s ON %s (%s)",
		idx.IndexName, idx.TableName, columnsStr)

	return a.db.Exec(createSQL).Error
}

func (a *IndexAnalyzer) AnalyzeQueryPerformance() error {
	log.Println("[INDEX_ANALYZER] Starting query performance analysis...")

	queries := a.getFrequentQueries()

	for _, query := range queries {
		suggestions := a.analyzeQuery(query)
		if len(suggestions) > 0 {
			log.Printf("[INDEX_ANALYZER] Query suggestions for %s: %v", query.Name, suggestions)
		}
	}

	return nil
}

type QueryInfo struct {
	Name           string
	QueryPattern   string
	ExecutionCount int64
	AvgDuration    time.Duration
}

func (a *IndexAnalyzer) getFrequentQueries() []QueryInfo {
	return []QueryInfo{
		{
			Name:           "CheckBlacklist",
			QueryPattern:   "SELECT * FROM blacklist WHERE target = ? AND type = ? AND status = ?",
			ExecutionCount: 10000,
			AvgDuration:    5 * time.Millisecond,
		},
		{
			Name:           "ListApplications",
			QueryPattern:   "SELECT * FROM applications WHERE name LIKE ? OR description LIKE ?",
			ExecutionCount: 5000,
			AvgDuration:    15 * time.Millisecond,
		},
		{
			Name:           "QueryLogs",
			QueryPattern:   "SELECT * FROM verification_logs WHERE application_id = ? AND created_at >= ?",
			ExecutionCount: 8000,
			AvgDuration:    20 * time.Millisecond,
		},
	}
}

func (a *IndexAnalyzer) analyzeQuery(query QueryInfo) []string {
	var suggestions []string

	if query.AvgDuration > 10*time.Millisecond {
		suggestions = append(suggestions, "Query is slow, consider adding index")
	}

	if query.ExecutionCount > 5000 {
		suggestions = append(suggestions, "High execution frequency, optimize with caching")
	}

	return suggestions
}

func (a *IndexAnalyzer) GetIndexUsageStats() ([]IndexUsageStats, error) {
	var stats []IndexUsageStats

	err := a.db.Raw(`
		SELECT
			idxrelname AS index_name,
			pg_size_pretty(pg_relation_size(i.indexrelid)) AS index_size,
			idx_scan AS number_of_scans,
			idx_tup_read AS tuples_read,
			idx_tup_fetch AS tuples_fetched
		FROM pg_stat_user_indexes ui
		JOIN pg_index i ON ui.indexrelid = i.indexrelid
		ORDER BY idx_scan DESC
		LIMIT 20
	`).Scan(&stats).Error

	return stats, err
}

type IndexUsageStats struct {
	IndexName      string
	IndexSize      string
	NumberOfScans  int64
	TuplesRead     int64
	TuplesFetched  int64
}

func (a *IndexAnalyzer) FindUnusedIndexes() ([]string, error) {
	var unusedIndexes []string

	err := a.db.Raw(`
		SELECT indexname
		FROM pg_stat_user_indexes
		WHERE idx_scan = 0
		AND indexname NOT LIKE '%_pkey'
		AND indexname NOT LIKE '%_fkey'
	`).Scan(&unusedIndexes).Error

	return unusedIndexes, err
}

func (a *IndexAnalyzer) GetTableBloat() ([]TableBloat, error) {
	var bloat []TableBloat

	err := a.db.Raw(`
		SELECT
			schemaname || '.' || tablename AS table_name,
			pg_size_pretty(pg_total_relation_size(schemaname::regnamespace::oid, tablename::regclass)) AS total_size,
			pg_size_pretty(pg_relation_size(schemaname::regnamespace::oid, tablename::regclass)) AS table_size,
			ROUND(100 * pg_total_relation_size(schemaname::regnamespace::oid, tablename::regclass) /
				NULLIF(pg_total_relation_size(schemaname::regnamespace::oid, tablename::regclass), 0), 2) AS bloat_ratio
		FROM pg_tables
		WHERE schemaname = 'public'
		ORDER BY pg_total_relation_size(schemaname::regnamespace::oid, tablename::regclass) DESC
		LIMIT 10
	`).Scan(&bloat).Error

	return bloat, err
}

type TableBloat struct {
	TableName   string
	TotalSize   string
	TableSize   string
	BloatRatio  float64
}
