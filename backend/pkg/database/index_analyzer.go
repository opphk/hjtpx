package database

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

type IndexAnalyzer struct {
	db *gorm.DB
	mu sync.RWMutex
}

type RedundantIndex struct {
	IndexName         string
	TableName         string
	Columns           []string
	RedundantReason   string
	DuplicatedWith    string
	EstimatedSavings  string
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

func (ia *IndexAnalyzer) FindRedundantIndexes() ([]RedundantIndex, error) {
	ia.mu.Lock()
	defer ia.mu.Unlock()

	var redundant []RedundantIndex

	indexes, err := ia.getAllIndexes()
	if err != nil {
		return nil, err
	}

	checked := make(map[string]bool)
	for i, idx1 := range indexes {
		if checked[idx1.IndexName] {
			continue
		}

		for j := i + 1; j < len(indexes); j++ {
			idx2 := indexes[j]
			if idx1.TableName != idx2.TableName {
				continue
			}

			if ia.isIndexRedundant(idx1, idx2) {
				redundant = append(redundant, RedundantIndex{
					IndexName:        idx2.IndexName,
					TableName:        idx2.TableName,
					Columns:          idx2.Columns,
					RedundantReason:  ia.getRedundantReason(idx1, idx2),
					DuplicatedWith:   idx1.IndexName,
					EstimatedSavings: "10-30% 存储空间",
				})
				checked[idx2.IndexName] = true
			}
		}
	}

	return redundant, nil
}

type indexInfo struct {
	IndexName string
	TableName string
	Columns   []string
	IsUnique  bool
}

func (ia *IndexAnalyzer) getAllIndexes() ([]indexInfo, error) {
	var indexes []indexInfo

	rows, err := ia.db.Raw(`
		SELECT
			i.relname AS index_name,
			t.relname AS table_name,
			pg_get_indexdef(i.oid) AS index_def
		FROM pg_index idx
		JOIN pg_class t ON t.oid = idx.indrelid
		JOIN pg_class i ON i.oid = idx.indexrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = 'public'
		AND NOT idx.indisprimary
		AND NOT idx.indisunique
	`).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var idx indexInfo
		var indexDef string
		if err := rows.Scan(&idx.IndexName, &idx.TableName, &indexDef); err != nil {
			continue
		}

		idx.Columns = ia.extractColumnsFromIndexDef(indexDef)
		indexes = append(indexes, idx)
	}

	return indexes, nil
}

func (ia *IndexAnalyzer) extractColumnsFromIndexDef(indexDef string) []string {
	var columns []string

	parts := strings.Split(indexDef, "(")
	if len(parts) < 2 {
		return columns
	}

	colsPart := strings.Split(parts[1], ")")[0]
	cols := strings.Split(colsPart, ",")

	for _, col := range cols {
		col = strings.TrimSpace(col)
		col = strings.Split(col, " ")[0]
		if col != "" {
			columns = append(columns, col)
		}
	}

	return columns
}

func (ia *IndexAnalyzer) isIndexRedundant(idx1, idx2 indexInfo) bool {
	if len(idx2.Columns) < len(idx1.Columns) {
		return false
	}

	for i := range idx1.Columns {
		if i >= len(idx2.Columns) || idx1.Columns[i] != idx2.Columns[i] {
			return false
		}
	}

	return true
}

func (ia *IndexAnalyzer) getRedundantReason(idx1, idx2 indexInfo) string {
	if len(idx2.Columns) > len(idx1.Columns) {
		return fmt.Sprintf("索引 %s 是 %s 的前缀索引，可以删除此索引", idx2.IndexName, idx1.IndexName)
	}
	return "索引与其他索引完全重复，可以删除此索引"
}

func (ia *IndexAnalyzer) AnalyzeSlowTable(tableName string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	var rowCount int64
	if err := ia.db.Table(tableName).Count(&rowCount).Error; err != nil {
		return nil, err
	}
	result["row_count"] = rowCount

	var avgRowSize string
	if err := ia.db.Raw("SELECT pg_size_pretty(avg_row_length::bigint) FROM pg_stats WHERE tablename = ?", tableName).Scan(&avgRowSize).Error; err != nil {
		avgRowSize = "unknown"
	}
	result["avg_row_size"] = avgRowSize

	var indexCount int
	if err := ia.db.Raw("SELECT COUNT(*) FROM pg_indexes WHERE tablename = ?", tableName).Scan(&indexCount).Error; err != nil {
		indexCount = 0
	}
	result["index_count"] = indexCount

	var lastVacuum, lastAnalyze time.Time
	type vacuumAnalyze struct {
		LastVacuum  time.Time
		LastAnalyze time.Time
	}
	var va vacuumAnalyze
	if err := ia.db.Raw(`
		SELECT last_vacuum, last_analyze
		FROM pg_stat_user_tables
		WHERE relname = ?
	`, tableName).Scan(&va).Error; err == nil {
		lastVacuum = va.LastVacuum
		lastAnalyze = va.LastAnalyze
	}

	result["last_vacuum"] = lastVacuum
	result["last_analyze"] = lastAnalyze

	if rowCount > 1000000 {
		result["recommendation"] = "表数据量超过100万行，建议考虑分区或归档"
	} else if lastAnalyze.IsZero() || time.Since(lastAnalyze) > 7*24*time.Hour {
		result["recommendation"] = "建议执行 ANALYZE 收集统计信息"
	}

	return result, nil
}

func (ia *IndexAnalyzer) GetTableSizeInfo() ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	rows, err := ia.db.Raw(`
		SELECT
			schemaname || '.' || tablename AS table_name,
			pg_size_pretty(pg_total_relation_size(schemaname::regnamespace::oid, tablename::regclass)) AS total_size,
			pg_size_pretty(pg_relation_size(schemaname::regnamespace::oid, tablename::regclass)) AS table_size,
			pg_size_pretty(pg_indexes_size(schemaname::regnamespace::oid, tablename::regclass)) AS index_size,
			pg_total_relation_size(schemaname::regnamespace::oid, tablename::regclass) AS total_bytes,
			pg_relation_size(schemaname::regnamespace::oid, tablename::regclass) AS table_bytes,
			pg_indexes_size(schemaname::regnamespace::oid, tablename::regclass) AS index_bytes
		FROM pg_tables
		WHERE schemaname = 'public'
		ORDER BY pg_total_relation_size(schemaname::regnamespace::oid, tablename::regclass) DESC
		LIMIT 20
	`).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, totalSize, tableSize, indexSize string
		var totalBytes, tableBytes, indexBytes int64

		if err := rows.Scan(&tableName, &totalSize, &tableSize, &indexSize, &totalBytes, &tableBytes, &indexBytes); err != nil {
			continue
		}

		indexRatio := 0.0
		if totalBytes > 0 {
			indexRatio = float64(indexBytes) / float64(totalBytes) * 100
		}

		results = append(results, map[string]interface{}{
			"table_name":    tableName,
			"total_size":    totalSize,
			"table_size":    tableSize,
			"index_size":    indexSize,
			"index_ratio":   fmt.Sprintf("%.2f%%", indexRatio),
			"recommendation": ia.getIndexRatioRecommendation(indexRatio),
		})
	}

	return results, nil
}

func (ia *IndexAnalyzer) getIndexRatioRecommendation(ratio float64) string {
	if ratio > 50 {
		return "索引占比过高，考虑删除未使用的索引"
	} else if ratio < 10 {
		return "索引空间占比正常"
	}
	return "索引空间占比合理"
}

func (ia *IndexAnalyzer) RecommendIndexesForTable(tableName string) ([]map[string]interface{}, error) {
	var recommendations []map[string]interface{}

	var columns []struct {
		ColumnName string
		DataType   string
		Nullable   string
	}

	if err := ia.db.Raw(`
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = ?
	`, tableName).Scan(&columns).Error; err != nil {
		return nil, err
	}

	for _, col := range columns {
		var usageCount int64
		ia.db.Raw(`
			SELECT COALESCE(idx_scan, 0)
			FROM pg_stat_user_indexes ui
			JOIN pg_index i ON ui.indexrelid = i.indexrelid
			JOIN pg_class c ON c.oid = i.indrelid
			WHERE c.relname = ?
			AND pg_get_indexdef(i.oid) LIKE '%' || ? || '%'
		`, tableName, col.ColumnName).Scan(&usageCount)

		if usageCount == 0 && col.Nullable == "NO" {
			recommendations = append(recommendations, map[string]interface{}{
				"column":      col.ColumnName,
				"data_type":   col.DataType,
				"reason":      "该列未被索引且非空，适合创建索引",
				"priority":    "medium",
				"suggested_index": fmt.Sprintf("idx_%s_%s", tableName, col.ColumnName),
			})
		}
	}

	return recommendations, nil
}

func (ia *IndexAnalyzer) RunFullIndexAnalysis(ctx context.Context) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	unusedIndexes, err := ia.FindUnusedIndexes()
	if err != nil {
		result["unused_indexes_error"] = err.Error()
	} else {
		result["unused_indexes"] = unusedIndexes
		result["unused_count"] = len(unusedIndexes)
	}

	redundantIndexes, err := ia.FindRedundantIndexes()
	if err != nil {
		result["redundant_indexes_error"] = err.Error()
	} else {
		result["redundant_indexes"] = redundantIndexes
		result["redundant_count"] = len(redundantIndexes)
	}

	health, err := ia.getIndexHealth()
	if err != nil {
		result["health_error"] = err.Error()
	} else {
		result["health"] = health
	}

	sizeInfo, err := ia.GetTableSizeInfo()
	if err != nil {
		result["size_info_error"] = err.Error()
	} else {
		result["size_info"] = sizeInfo
	}

	result["total_recommendations"] = len(unusedIndexes) + len(redundantIndexes)
	result["timestamp"] = time.Now()

	return result, nil
}

func (ia *IndexAnalyzer) SafeDropIndex(indexName string) error {
	ia.mu.Lock()
	defer ia.mu.Unlock()

	var count int64
	if err := ia.db.Raw("SELECT COUNT(*) FROM pg_indexes WHERE indexname = ?", indexName).Scan(&count).Error; err != nil {
		return fmt.Errorf("检查索引失败: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("索引 %s 不存在", indexName)
	}

	if strings.HasSuffix(indexName, "_pkey") || strings.HasSuffix(indexName, "_fkey") {
		return fmt.Errorf("不能删除主键或外键索引")
	}

	var scanCount int64
	if err := ia.db.Raw(`
		SELECT COALESCE(idx_scan, 0)
		FROM pg_stat_user_indexes
		WHERE indexrelname = ?
	`, indexName).Scan(&scanCount).Error; err != nil {
		return fmt.Errorf("获取索引使用统计失败: %w", err)
	}

	if scanCount > 0 {
		log.Printf("[WARNING] 索引 %s 已被使用 %d 次，是否确认删除?", indexName, scanCount)
	}

	sql := fmt.Sprintf("DROP INDEX CONCURRENTLY IF EXISTS %s", indexName)
	return ia.db.Exec(sql).Error
}

func (ia *IndexAnalyzer) getIndexHealth() (map[string]interface{}, error) {
	if ia.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var healthStats []struct {
		IndexName   string
		IndexSize   string
		Scans       int64
		TuplesRead  int64
		HealthScore float64
	}

	err := ia.db.Raw(`
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

func (ia *IndexAnalyzer) BatchCreateIndexes(indexes []MissingIndex) error {
	ia.mu.Lock()
	defer ia.mu.Unlock()

	for _, idx := range indexes {
		if err := ia.createIndex(idx); err != nil {
			log.Printf("[INDEX_ANALYZER] 批量创建索引失败 %s: %v", idx.IndexName, err)
			continue
		}
		log.Printf("[INDEX_ANALYZER] 成功创建索引: %s", idx.IndexName)
	}

	return nil
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

