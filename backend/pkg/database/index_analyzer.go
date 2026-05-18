package database

import (
	"fmt"
	"log"
	"sync"
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

type IndexRecommendation struct {
	TableName         string
	IndexName         string
	Columns           []string
	IndexType         string
	Priority          string
	EstimatedSize     string
	QueryBenefits     []string
	CreationSQL       string
	EstimatedImpact   string
}

type IndexOptimizer struct {
	db                  *gorm.DB
	recommendations    []*IndexRecommendation
	lastAnalysis       time.Time
	enableAutoCreate   bool
	enableAutoAnalyze  bool
	analysisInterval   time.Duration
}

func NewIndexOptimizer(db *gorm.DB) *IndexOptimizer {
	return &IndexOptimizer{
		db:                 db,
		recommendations:   make([]*IndexRecommendation, 0),
		lastAnalysis:      time.Time{},
		enableAutoCreate:   true,
		enableAutoAnalyze:  true,
		analysisInterval:   24 * time.Hour,
	}
}

func (io *IndexOptimizer) AnalyzeIndexes() error {
	if time.Since(io.lastAnalysis) < io.analysisInterval {
		return nil
	}

	io.recommendations = io.generateRecommendations()

	io.lastAnalysis = time.Now()
	return nil
}

func (io *IndexOptimizer) generateRecommendations() []*IndexRecommendation {
	var recs []*IndexRecommendation

	recs = append(recs,
		&IndexRecommendation{
			TableName:     "behavior_data",
			IndexName:    "idx_behavior_user_time",
			Columns:      []string{"user_id", "created_at"},
			IndexType:    "btree",
			Priority:     "high",
			EstimatedSize: "50MB",
			QueryBenefits: []string{"用户行为查询加速", "轨迹分析加速"},
			CreationSQL:   "CREATE INDEX CONCURRENTLY idx_behavior_user_time ON behavior_data (user_id, created_at)",
		},
		&IndexRecommendation{
			TableName:     "trace_records",
			IndexName:    "idx_trace_session_created",
			Columns:      []string{"session_id", "created_at"},
			IndexType:    "btree",
			Priority:     "high",
			EstimatedSize: "100MB",
			QueryBenefits: []string{"轨迹查询加速", "会话分析加速"},
			CreationSQL:   "CREATE INDEX CONCURRENTLY idx_trace_session_created ON trace_records (session_id, created_at)",
		},
		&IndexRecommendation{
			TableName:     "verification_logs",
			IndexName:    "idx_verification_app_created",
			Columns:      []string{"application_id", "created_at", "status"},
			IndexType:    "btree",
			Priority:     "high",
			EstimatedSize: "200MB",
			QueryBenefits: []string{"应用验证统计加速", "日志查询加速"},
			CreationSQL:   "CREATE INDEX CONCURRENTLY idx_verification_app_created ON verification_logs (application_id, created_at, status)",
		},
		&IndexRecommendation{
			TableName:     "device_fingerprints",
			IndexName:    "idx_device_fingerprint_hash",
			Columns:      []string{"fingerprint_hash", "created_at"},
			IndexType:    "btree",
			Priority:     "medium",
			EstimatedSize: "80MB",
			QueryBenefits: []string{"设备查询加速", "指纹匹配加速"},
			CreationSQL:   "CREATE INDEX CONCURRENTLY idx_device_fingerprint_hash ON device_fingerprints (fingerprint_hash, created_at)",
		},
	)

	return recs
}

func (io *IndexOptimizer) GetRecommendations() []*IndexRecommendation {
	return io.recommendations
}

func (io *IndexOptimizer) ApplyRecommendation(rec *IndexRecommendation) error {
	if io.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var count int64
	err := io.db.Raw(`
		SELECT COUNT(*)
		FROM pg_indexes
		WHERE indexname = ?
	`, rec.IndexName).Scan(&count).Error

	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("index %s already exists", rec.IndexName)
	}

	return io.db.Exec(rec.CreationSQL).Error
}

func (io *IndexOptimizer) GetIndexHealth() (map[string]interface{}, error) {
	if io.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var healthStats []struct {
		IndexName    string
		IndexSize    string
		Scans        int64
		TuplesRead   int64
		HealthScore  float64
	}

	err := io.db.Raw(`
		SELECT
			idxrelname AS index_name,
			pg_size_pretty(pg_relation_size(i.indexrelid)) AS index_size,
			COALESCE(idx_scan, 0) AS number_of_scans,
			COALESCE(idx_tup_read, 0) AS tuples_read,
			CASE
				WHEN COALESCE(idx_scan, 0) = 0 THEN 0.0
				ELSE LEAST(100.0, (idx_tup_read::float / NULLIF(idx_scan, 0)) * 10)
			END AS health_score
		FROM pg_stat_user_indexes ui
		JOIN pg_index i ON ui.indexrelid = i.indexrelid
		WHERE schemaname = 'public'
		ORDER BY health_score DESC
		LIMIT 50
	`).Scan(&healthStats).Error

	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	result["total_indexes"] = len(healthStats)

	healthy := 0
	unhealthy := 0
	for _, stat := range healthStats {
		if stat.HealthScore >= 50.0 {
			healthy++
		} else {
			unhealthy++
		}
	}

	result["healthy_indexes"] = healthy
	result["unhealthy_indexes"] = unhealthy
	result["indexes"] = healthStats

	return result, nil
}

func (io *IndexOptimizer) AnalyzeTableFragmentation() ([]map[string]interface{}, error) {
	if io.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var fragmentation []map[string]interface{}

	err := io.db.Raw(`
		SELECT
			schemaname || '.' || tablename AS table_name,
			pg_size_pretty(pg_total_relation_size(schemaname::regnamespace::oid, tablename::regclass)) AS total_size,
			pg_size_pretty(pg_relation_size(schemaname::regnamespace::oid, tablename::regclass)) AS table_size,
			pg_size_pretty(pg_indexes_size(schemaname::regnamespace::oid, tablename::regclass)) AS index_size,
			ROUND(100 * pg_indexes_size(schemaname::regnamespace::oid, tablename::regclass) /
				NULLIF(pg_total_relation_size(schemaname::regnamespace::oid, tablename::regclass), 0), 2) AS index_ratio
		FROM pg_tables
		WHERE schemaname = 'public'
		ORDER BY pg_total_relation_size(schemaname::regnamespace::oid, tablename::regclass) DESC
		LIMIT 20
	`).Scan(&fragmentation).Error

	return fragmentation, err
}

func (io *IndexOptimizer) AutoOptimize() error {
	if err := io.AnalyzeIndexes(); err != nil {
		return err
	}

	if !io.enableAutoCreate {
		return nil
	}

	for _, rec := range io.recommendations {
		if rec.Priority == "high" {
			if err := io.ApplyRecommendation(rec); err != nil {
				continue
			}
		}
	}

	return nil
}

func (io *IndexOptimizer) SetAutoCreate(enabled bool) {
	io.enableAutoCreate = enabled
}

func (io *IndexOptimizer) SetAutoAnalyze(enabled bool) {
	io.enableAutoAnalyze = enabled
}

var globalIndexOptimizer *IndexOptimizer
var globalIndexOptimizerOnce sync.Once

func InitIndexOptimizer(db *gorm.DB) {
	globalIndexOptimizerOnce.Do(func() {
		globalIndexOptimizer = NewIndexOptimizer(db)
	})
}

func GetIndexOptimizer() *IndexOptimizer {
	return globalIndexOptimizer
}

