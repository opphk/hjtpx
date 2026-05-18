package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

type QueryAnalyzer struct {
	db                  *gorm.DB
	slowQueryThreshold  time.Duration
	maxQueriesToAnalyze int
}

func NewQueryAnalyzer(db *gorm.DB, thresholdMs int) *QueryAnalyzer {
	return &QueryAnalyzer{
		db:                  db,
		slowQueryThreshold: time.Duration(thresholdMs) * time.Millisecond,
		maxQueriesToAnalyze: 50,
	}
}

func (a *QueryAnalyzer) AnalyzeSlowQueries() ([]SlowQueryResult, error) {
	var results []SlowQueryResult

	err := a.db.Raw(`
		SELECT
			queryid,
			LEFT(query, 200) as query_preview,
			calls,
			total_exec_time / 1000 as total_time_ms,
			mean_exec_time / 1000 as mean_time_ms,
			max_exec_time / 1000 as max_time_ms,
			min_exec_time / 1000 as min_time_ms,
			rows as total_rows
		FROM pg_stat_statements
		WHERE mean_exec_time > ?
		ORDER BY mean_exec_time DESC
		LIMIT ?
	`, a.slowQueryThreshold.Milliseconds(), a.maxQueriesToAnalyze).Scan(&results).Error

	if err != nil {
		log.Printf("[SLOW_QUERY_ANALYZER] Failed to query pg_stat_statements: %v", err)
		return nil, err
	}

	return results, nil
}

type SlowQueryResult struct {
	QueryID       int64   `json:"query_id"`
	QueryPreview  string  `json:"query_preview"`
	Calls         int64   `json:"calls"`
	TotalTimeMs   float64 `json:"total_time_ms"`
	MeanTimeMs    float64 `json:"mean_time_ms"`
	MaxTimeMs     float64 `json:"max_time_ms"`
	MinTimeMs     float64 `json:"min_time_ms"`
	TotalRows     int64   `json:"total_rows"`
	Suggestions   []string `json:"suggestions"`
}

func (a *QueryAnalyzer) GenerateSuggestions(query string, meanTimeMs float64) []string {
	var suggestions []string

	if meanTimeMs > 100 {
		suggestions = append(suggestions, "HIGH: Query is very slow (>100ms), consider adding indexes or rewriting")
	} else if meanTimeMs > 50 {
		suggestions = append(suggestions, "MEDIUM: Query is slow (>50ms), consider optimization")
	}

	if len(query) > 500 {
		suggestions = append(suggestions, "Consider simplifying the query or breaking it into smaller parts")
	}

	if containsWildcardPrefix(query) {
		suggestions = append(suggestions, "Query contains leading wildcard, which cannot use indexes efficiently")
	}

	if containsSubquery(query) {
		suggestions = append(suggestions, "Query contains subquery, consider using JOIN instead")
	}

	if containsSelectStar(query) {
		suggestions = append(suggestions, "Query uses SELECT *, specify only needed columns")
	}

	if containsNotIn(query) {
		suggestions = append(suggestions, "Consider using NOT EXISTS or LEFT JOIN WHERE NULL instead of NOT IN")
	}

	return suggestions
}

func containsWildcardPrefix(query string) bool {
	return containsPattern(query, "%'") || containsPattern(query, "%\"")
}

func containsSubquery(query string) bool {
	return containsPattern(query, "SELECT") && containsPattern(query, "(")
}

func containsSelectStar(query string) bool {
	return containsPattern(query, "SELECT *")
}

func containsNotIn(query string) bool {
	return containsPattern(query, "NOT IN")
}

func containsPattern(query, pattern string) bool {
	return len(query) >= len(pattern) && containsString(query, pattern)
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (a *QueryAnalyzer) AnalyzeQueryPlan(query string) (*QueryPlanAnalysis, error) {
	var plan string
	err := a.db.Raw("EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) " + query).Scan(&plan).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get query plan: %w", err)
	}

	analysis := &QueryPlanAnalysis{
		Query:       query,
		Plan:        plan,
		ScanTypes:   a.extractScanTypes(plan),
		CostInfo:    a.extractCostInfo(plan),
		Warnings:    a.analyzePlanWarnings(plan),
		Score:       a.calculatePerformanceScore(plan),
	}

	return analysis, nil
}

type QueryPlanAnalysis struct {
	Query      string            `json:"query"`
	Plan       string            `json:"plan"`
	ScanTypes  []string          `json:"scan_types"`
	CostInfo   map[string]string `json:"cost_info"`
	Warnings   []string          `json:"warnings"`
	Score      int               `json:"performance_score"`
}

func (a *QueryAnalyzer) extractScanTypes(plan string) []string {
	var scanTypes []string
	patterns := []string{"Seq Scan", "Index Scan", "Bitmap Heap Scan", "Index Only Scan"}

	for _, pattern := range patterns {
		if containsString(plan, pattern) {
			scanTypes = append(scanTypes, pattern)
		}
	}

	return scanTypes
}

func (a *QueryAnalyzer) extractCostInfo(plan string) map[string]string {
	info := make(map[string]string)

	start := indexOf(plan, "cost=")
	if start >= 0 {
		end := start + 20
		if end > len(plan) {
			end = len(plan)
		}
		for i := start; i < end && plan[i] != ' '; i++ {
			if i < end {
				info["cost_range"] = plan[start:i]
			}
		}
	}

	if indexOf(plan, "actual time=") >= 0 {
		info["executed"] = "true"
	}

	return info
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func (a *QueryAnalyzer) analyzePlanWarnings(plan string) []string {
	var warnings []string

	if containsString(plan, "Seq Scan on") {
		warnings = append(warnings, "Full table scan detected, consider adding indexes")
	}

	if containsString(plan, "Hash Join") {
		warnings = append(warnings, "Hash join used, ensure join columns are indexed")
	}

	if containsString(plan, "Nested Loop") {
		warnings = append(warnings, "Nested loop join, ensure inner table has index on join column")
	}

	if containsString(plan, "Sort") && !containsString(plan, "Index Scan") {
		warnings = append(warnings, "Explicit sort operation, consider adding index to avoid sorting")
	}

	return warnings
}

func (a *QueryAnalyzer) FindUnusedIndexes() ([]string, error) {
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

func (a *QueryAnalyzer) GetIndexUsageStats() ([]IndexUsageStats, error) {
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

func (a *QueryAnalyzer) calculatePerformanceScore(plan string) int {
	score := 100

	if containsString(plan, "Seq Scan") {
		score -= 30
	}

	if containsString(plan, "Bitmap Heap Scan") {
		score -= 10
	}

	if containsString(plan, "Hash Join") {
		score -= 5
	}

	if containsString(plan, "Nested Loop") && containsString(plan, "Seq Scan") {
		score -= 20
	}

	if containsString(plan, "Sort") && !containsString(plan, "Index") {
		score -= 10
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (a *QueryAnalyzer) GetIndexRecommendations(tableName string) ([]IndexRecommendation, error) {
	var recommendations []IndexRecommendation

	var missingIndexes []struct {
		TableName    string
		IndexName    string
		Columns      string
		Unique       bool
		EstSizeKb    int64
		UsageFreq    int64
	}

	err := a.db.Raw(`
		SELECT
			schemaname as table_name,
			tablename,
			'idx_' || tablename || '_recommended' as index_name,
			'(' || string_agg(column_name, ', ') || ')' as columns,
			false as unique_flag,
			0 as estimated_size_kb,
			0 as usage_frequency
		FROM information_schema.columns
		WHERE table_schema = 'public'
		AND tablename = ?
		AND column_name IN ('created_at', 'updated_at', 'status', 'user_id', 'application_id')
		GROUP BY schemaname, tablename
	`, tableName).Scan(&missingIndexes).Error

	if err != nil {
		return nil, err
	}

	for _, idx := range missingIndexes {
		recommendations = append(recommendations, IndexRecommendation{
			TableName:       idx.TableName,
			IndexName:       idx.IndexName,
			Columns:         idx.Columns,
			Priority:        "medium",
			EstimatedSizeKb: idx.EstSizeKb,
			UsageFrequency:  idx.UsageFreq,
		})
	}

	return recommendations, nil
}

type IndexRecommendation struct {
	TableName       string `json:"table_name"`
	IndexName       string `json:"index_name"`
	Columns         string `json:"columns"`
	Priority        string `json:"priority"`
	EstimatedSizeKb int64  `json:"estimated_size_kb"`
	UsageFrequency  int64  `json:"usage_frequency"`
	DDL             string `json:"ddl"`
}

func (a *QueryAnalyzer) GenerateIndexDDL(rec IndexRecommendation) string {
	return fmt.Sprintf("CREATE INDEX CONCURRENTLY %s ON %s %s;",
		rec.IndexName, rec.TableName, rec.Columns)
}

func (a *QueryAnalyzer) RunFullAnalysis(ctx context.Context) (*FullAnalysisReport, error) {
	report := &FullAnalysisReport{
		Timestamp: time.Now(),
	}

	slowQueries, err := a.AnalyzeSlowQueries()
	if err != nil {
		log.Printf("[SLOW_QUERY_ANALYZER] Failed to analyze slow queries: %v", err)
	} else {
		report.SlowQueries = slowQueries
		report.SlowQueryCount = len(slowQueries)
	}

	unusedIndexes, err := a.FindUnusedIndexes()
	if err != nil {
		log.Printf("[SLOW_QUERY_ANALYZER] Failed to find unused indexes: %v", err)
	} else {
		report.UnusedIndexes = unusedIndexes
	}

	stats, err := a.GetIndexUsageStats()
	if err != nil {
		log.Printf("[SLOW_QUERY_ANALYZER] Failed to get index stats: %v", err)
	} else {
		report.IndexStats = stats
	}

	report.Recommendations = a.generateRecommendations(report)

	return report, nil
}

type FullAnalysisReport struct {
	Timestamp        time.Time                  `json:"timestamp"`
	SlowQueries      []SlowQueryResult          `json:"slow_queries"`
	SlowQueryCount   int                        `json:"slow_query_count"`
	UnusedIndexes    []string                   `json:"unused_indexes"`
	IndexStats       []IndexUsageStats          `json:"index_stats"`
	Recommendations  []string                   `json:"recommendations"`
	OverallScore     int                        `json:"overall_score"`
}

func (a *QueryAnalyzer) generateRecommendations(report *FullAnalysisReport) []string {
	var recs []string

	if report.SlowQueryCount > 10 {
		recs = append(recs, "HIGH: Many slow queries detected, immediate optimization needed")
	} else if report.SlowQueryCount > 5 {
		recs = append(recs, "MEDIUM: Some slow queries detected, consider optimization")
	}

	if len(report.UnusedIndexes) > 0 {
		recs = append(recs, fmt.Sprintf("Found %d unused indexes, consider removing them", len(report.UnusedIndexes)))
	}

	recs = append(recs, "Enable pg_stat_statements extension for detailed query analysis")
	recs = append(recs, "Consider partitioning large tables by date")
	recs = append(recs, "Review query execution plans regularly")

	return recs
}
