package database

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

type SQLOptimizer struct {
	db                 *gorm.DB
	config             *config.Config
	indexRecommender   *IndexRecommender
	queryRewriter      *QueryRewriter
	statisticsAnalyzer *StatisticsAnalyzer
	enabled            bool
	mu                 sync.RWMutex
}

type IndexRecommender struct {
	mu       sync.RWMutex
	indexes  []IndexRecommendation
	lastScan time.Time
}

type IndexRecommendation struct {
	TableName       string        `json:"table_name"`
	Columns         []string      `json:"columns"`
	IndexType       string        `json:"index_type"`
	EstimatedSize   int64         `json:"estimated_size"`
	QueryFrequency  int           `json:"query_frequency"`
	Impact          float64       `json:"impact"`
	Suggestion      string        `json:"suggestion"`
	CreatedAt       time.Time     `json:"created_at"`
	Priority        int           `json:"priority"`
}

type QueryRewriter struct {
	mu       sync.RWMutex
	rules    []RewriteRule
	strategy string
}

type RewriteRule struct {
	Name        string `json:"name"`
	Pattern     string `json:"pattern"`
	Replacement string `json:"replacement"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

type QueryOptimizationResult struct {
	OriginalQuery    string               `json:"original_query"`
	OptimizedQuery   string               `json:"optimized_query"`
	Changes         []QueryChange        `json:"changes"`
	EstimatedImprovement float64           `json:"estimated_improvement"`
	Warnings        []string              `json:"warnings"`
	NewIndexes      []IndexRecommendation `json:"new_indexes"`
	ExecutionPlan   string                `json:"execution_plan,omitempty"`
}

type QueryChange struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Before      string `json:"before"`
	After       string `json:"after"`
	Impact      string `json:"impact"`
}

type StatisticsAnalyzer struct {
	mu          sync.RWMutex
	tableStats  map[string]*TableStatistics
	lastAnalyze time.Time
}

type TableStatistics struct {
	TableName       string             `json:"table_name"`
	TotalRows       int64              `json:"total_rows"`
	TotalSize       int64              `json:"total_size"`
	IndexesSize     int64              `json:"indexes_size"`
	LastVacuum      time.Time          `json:"last_vacuum"`
	LastAnalyze     time.Time          `json:"last_analyze"`
	SeqScanRate     float64            `json:"seq_scan_rate"`
	IndexScanRate   float64            `json:"index_scan_rate"`
	CacheHitRate    float64            `json:"cache_hit_rate"`
	ColumnsStats    []ColumnStatistics `json:"columns_stats"`
}

type ColumnStatistics struct {
	ColumnName    string        `json:"column_name"`
	NullCount     int64         `json:"null_count"`
	UniqueCount   int64         `json:"unique_count"`
	AvgWidth      int           `json:"avg_width"`
	MostCommonVals []string     `json:"most_common_values"`
	HistogramBound []string     `json:"histogram_bounds"`
}

type SQLOptimizationConfig struct {
	EnableAutoOptimize   bool `yaml:"enable_auto_optimize"`
	MaxRecommendations   int  `yaml:"max_recommendations"`
	AnalyzeThreshold     int  `yaml:"analyze_threshold"`
}

var sqlOptimizer *SQLOptimizer

func InitSQLOptimizer(db *gorm.DB, cfg *config.Config) error {
	sqlOptimizer = &SQLOptimizer{
		db:   db,
		config: cfg,
		enabled: cfg.Database.QueryOptimization.EnablePreparedStatements,
		indexRecommender: &IndexRecommender{
			indexes: make([]IndexRecommendation, 0),
		},
		queryRewriter: &QueryRewriter{
			rules:    initializeRewriteRules(),
			strategy: "aggressive",
		},
		statisticsAnalyzer: &StatisticsAnalyzer{
			tableStats: make(map[string]*TableStatistics),
		},
	}

	if sqlOptimizer.enabled {
		go sqlOptimizer.startPeriodicOptimization()
	}

	log.Println("SQL optimizer initialized")
	return nil
}

func GetSQLOptimizer() *SQLOptimizer {
	return sqlOptimizer
}

func (o *SQLOptimizer) OptimizeQuery(ctx context.Context, query string) (*QueryOptimizationResult, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	result := &QueryOptimizationResult{
		OriginalQuery: query,
		Changes:       make([]QueryChange, 0),
		Warnings:      make([]string, 0),
		NewIndexes:    make([]IndexRecommendation, 0),
	}

	result.OptimizedQuery = query

	if o.queryRewriter != nil {
		rewrittenQuery, changes := o.queryRewriter.rewrite(query)
		result.OptimizedQuery = rewrittenQuery
		result.Changes = append(result.Changes, changes...)
	}

	execPlan, err := o.getExecutionPlan(ctx, result.OptimizedQuery)
	if err == nil {
		result.ExecutionPlan = execPlan
	}

	recommendations := o.indexRecommender.getRecommendations(query)
	result.NewIndexes = append(result.NewIndexes, recommendations...)

	result.EstimatedImprovement = o.estimateImprovement(result.Changes)

	return result, nil
}

func (o *SQLOptimizer) getExecutionPlan(ctx context.Context, query string) (string, error) {
	if o.db == nil {
		return "", fmt.Errorf("database connection not available")
	}

	explainQuery := fmt.Sprintf("EXPLAIN (FORMAT TEXT) %s", query)
	var plan string
	err := o.db.WithContext(ctx).Raw(explainQuery).Scan(&plan).Error
	if err != nil {
		return "", fmt.Errorf("failed to get execution plan: %w", err)
	}

	return plan, nil
}

func (o *SQLOptimizer) AnalyzeAndRecommend(ctx context.Context) ([]IndexRecommendation, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	queryMetrics := GetPerformanceMonitor()
	if queryMetrics == nil {
		return nil, fmt.Errorf("performance monitor not initialized")
	}

	slowQueries := queryMetrics.GetSlowQueries(1000)
	if len(slowQueries) == 0 {
		return []IndexRecommendation{}, nil
	}

	tableQueries := make(map[string]int)
	for _, sq := range slowQueries {
		tableName := extractTableNameFromQuery(sq.Query)
		if tableName != "" {
			tableQueries[tableName]++
		}
	}

	recommendations := make([]IndexRecommendation, 0)
	for tableName, queryCount := range tableQueries {
		if queryCount > 10 {
			columns := o.analyzeWhereColumns(tableName, slowQueries)
			if len(columns) > 0 {
				recomm := IndexRecommendation{
					TableName:      tableName,
					Columns:        columns,
					IndexType:      "B-tree",
					QueryFrequency: queryCount,
					Impact:         float64(queryCount) / float64(len(slowQueries)),
					Suggestion:     fmt.Sprintf("Consider creating composite index on %s.%s for better query performance", tableName, strings.Join(columns, ", ")),
					CreatedAt:      time.Now(),
					Priority:       queryCount,
				}
				recommendations = append(recommendations, recomm)
			}
		}
	}

	o.indexRecommender.mu.Lock()
	o.indexRecommender.indexes = recommendations
	o.indexRecommender.lastScan = time.Now()
	o.indexRecommender.mu.Unlock()

	return recommendations, nil
}

func (o *SQLOptimizer) analyzeWhereColumns(tableName string, queries []QueryMetric) []string {
	columnCounts := make(map[string]int)

	for _, q := range queries {
		if q.Query == "" {
			continue
		}

		if !strings.Contains(strings.ToUpper(q.Query), tableName) {
			continue
		}

		columns := extractWhereColumns(q.Query)
		for _, col := range columns {
			columnCounts[col]++
		}
	}

	sortedColumns := make([]string, 0)
	for col := range columnCounts {
		sortedColumns = append(sortedColumns, col)
	}

	sortByFrequency := func(i, j int) bool {
		return columnCounts[sortedColumns[i]] > columnCounts[sortedColumns[j]]
	}

	sort.Slice(sortedColumns, sortByFrequency)

	if len(sortedColumns) > 5 {
		sortedColumns = sortedColumns[:5]
	}

	return sortedColumns
}

func extractWhereColumns(query string) []string {
	columns := make([]string, 0)
	whereRegex := regexp.MustCompile(`(?i)WHERE\s+([a-zA-Z_][a-zA-Z0-9_\.]*)\s*[=<>]`)
	matches := whereRegex.FindAllStringSubmatch(query, -1)
	for _, match := range matches {
		if len(match) > 1 {
			columns = append(columns, match[1])
		}
	}
	return columns
}

func (o *SQLOptimizer) AnalyzeTableStatistics(ctx context.Context, tableName string) (*TableStatistics, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	stats := &TableStatistics{
		TableName:   tableName,
		ColumnsStats: make([]ColumnStatistics, 0),
	}

	var totalRows int64
	if err := o.db.WithContext(ctx).Table(tableName).Count(&totalRows).Error; err != nil {
		return nil, fmt.Errorf("failed to count rows: %w", err)
	}
	stats.TotalRows = totalRows

	sizeQuery := `
		SELECT 
			pg_total_relation_size($1) as total_size,
			pg_indexes_size($1) as indexes_size
	`
	var sizes struct {
		TotalSize   int64
		IndexesSize int64
	}
	if err := o.db.WithContext(ctx).Raw(sizeQuery, tableName).Scan(&sizes).Error; err != nil {
		log.Printf("Warning: failed to get table size: %v", err)
	} else {
		stats.TotalSize = sizes.TotalSize
		stats.IndexesSize = sizes.IndexesSize
	}

	stats.LastAnalyze = time.Now()

	o.statisticsAnalyzer.mu.Lock()
	o.statisticsAnalyzer.tableStats[tableName] = stats
	o.statisticsAnalyzer.lastAnalyze = time.Now()
	o.statisticsAnalyzer.mu.Unlock()

	return stats, nil
}

func (o *SQLOptimizer) RunVACUUM(tableName string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.db == nil {
		return fmt.Errorf("database connection not available")
	}

	query := fmt.Sprintf("VACUUM ANALYZE %s", tableName)
	return o.db.Exec(query).Error
}

func (o *SQLOptimizer) RunANALYZE(tableName string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.db == nil {
		return fmt.Errorf("database connection not available")
	}

	query := fmt.Sprintf("ANALYZE %s", tableName)
	return o.db.Exec(query).Error
}

func (o *SQLOptimizer) estimateImprovement(changes []QueryChange) float64 {
	if len(changes) == 0 {
		return 0.0
	}

	totalImprovement := 0.0
	for _, change := range changes {
		switch change.Type {
		case "index_added":
			totalImprovement += 0.5
		case "query_rewritten":
			totalImprovement += 0.3
		case "limit_added":
			totalImprovement += 0.2
		case "select_columns_reduced":
			totalImprovement += 0.15
		}
	}

	return totalImprovement
}

func (o *SQLOptimizer) startPeriodicOptimization() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		o.performPeriodicOptimization()
	}
}

func (o *SQLOptimizer) performPeriodicOptimization() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	_, err := o.AnalyzeAndRecommend(ctx)
	if err != nil {
		log.Printf("Failed to analyze recommendations: %v", err)
	}
}

func initializeRewriteRules() []RewriteRule {
	return []RewriteRule{
		{
			Name:        "RemoveSELECTStar",
			Pattern:     `(?i)SELECT\s+\*`,
			Replacement: "SELECT column_list",
			Description: "Replace SELECT * with specific column names",
			Example:     "SELECT * FROM users -> SELECT id, name, email FROM users",
		},
		{
			Name:        "AddLimitClause",
			Pattern:     `(?i)SELECT\s+(?!.*LIMIT)`,
			Replacement: "",
			Description: "Add LIMIT clause to prevent full table scans",
			Example:     "Add LIMIT 1000 to large result sets",
		},
		{
			Name:        "OptimizeLIKE",
			Pattern:     `(?i)LIKE\s+'%([^%]+)'`,
			Replacement: "LIKE 'prefix%'",
			Description: "Avoid leading wildcards in LIKE clauses",
			Example:     "LIKE '%abc' -> LIKE 'abc%'",
		},
	}
}

func (r *QueryRewriter) rewrite(query string) (string, []QueryChange) {
	optimized := query
	changes := make([]QueryChange, 0)

	for _, rule := range r.rules {
		matched, _ := regexp.MatchString(rule.Pattern, query)
		if matched {
			re := regexp.MustCompile(rule.Pattern)
			optimized = re.ReplaceAllString(optimized, rule.Replacement)

			changes = append(changes, QueryChange{
				Type:        "query_rewritten",
				Description: rule.Description,
				Before:      query,
				After:       optimized,
				Impact:      "medium",
			})
		}
	}

	return optimized, changes
}

func (i *IndexRecommender) getRecommendations(query string) []IndexRecommendation {
	i.mu.RLock()
	defer i.mu.RUnlock()

	recommendations := make([]IndexRecommendation, 0)
	for _, rec := range i.indexes {
		if strings.Contains(query, rec.TableName) {
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations
}

func (o *SQLOptimizer) GetStatistics() map[string]interface{} {
	o.mu.RLock()
	defer o.mu.RUnlock()

	stats := make(map[string]interface{})

	if o.indexRecommender != nil {
		o.indexRecommender.mu.RLock()
		stats["pending_indexes"] = len(o.indexRecommender.indexes)
		stats["last_scan"] = o.indexRecommender.lastScan
		o.indexRecommender.mu.RUnlock()
	}

	if o.statisticsAnalyzer != nil {
		o.statisticsAnalyzer.mu.RLock()
		stats["analyzed_tables"] = len(o.statisticsAnalyzer.tableStats)
		stats["last_analyze"] = o.statisticsAnalyzer.lastAnalyze
		o.statisticsAnalyzer.mu.RUnlock()
	}

	return stats
}

func (o *SQLOptimizer) CreateRecommendedIndexes(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.db == nil {
		return fmt.Errorf("database connection not available")
	}

	o.indexRecommender.mu.RLock()
	indexes := o.indexRecommender.indexes
	o.indexRecommender.mu.RUnlock()

	for _, rec := range indexes {
		if rec.Priority < 10 {
			continue
		}

		indexName := fmt.Sprintf("idx_%s_%s", rec.TableName, strings.Join(rec.Columns, "_"))
		createSQL := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)",
			indexName, rec.TableName, strings.Join(rec.Columns, ", "))

		if err := o.db.WithContext(ctx).Exec(createSQL).Error; err != nil {
			log.Printf("Failed to create index %s: %v", indexName, err)
		} else {
			log.Printf("Created index: %s", indexName)
		}
	}

	return nil
}

func (o *SQLOptimizer) RebuildIndexes(ctx context.Context, tableName string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.db == nil {
		return fmt.Errorf("database connection not available")
	}

	reindexSQL := fmt.Sprintf("REINDEX TABLE %s", tableName)
	return o.db.WithContext(ctx).Exec(reindexSQL).Error
}

func (o *SQLOptimizer) GetTableSizeInfo(ctx context.Context, tableName string) (map[string]int64, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	var sizeInfo struct {
		TotalSize   int64 `gorm:"column:total_size"`
		IndexesSize int64 `gorm:"column:indexes_size"`
		TableSize   int64 `gorm:"column:table_size"`
	}

	query := `
		SELECT 
			pg_total_relation_size($1) as total_size,
			pg_indexes_size($1) as indexes_size,
			pg_relation_size($1) as table_size
	`

	if err := o.db.WithContext(ctx).Raw(query, tableName).Scan(&sizeInfo).Error; err != nil {
		return nil, err
	}

	return map[string]int64{
		"total_size":   sizeInfo.TotalSize,
		"indexes_size": sizeInfo.IndexesSize,
		"table_size":   sizeInfo.TableSize,
	}, nil
}

func extractTableNameFromQuery(query string) string {
	if query == "" {
		return ""
	}

	tableRegex := regexp.MustCompile(`(?i)FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := tableRegex.FindStringSubmatch(query)
	if len(matches) > 1 {
		return matches[1]
	}

	joinRegex := regexp.MustCompile(`(?i)JOIN\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	joinMatches := joinRegex.FindStringSubmatch(query)
	if len(joinMatches) > 1 {
		return joinMatches[1]
	}

	return ""
}

