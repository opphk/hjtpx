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

type EnhancedIndexOptimizer struct {
	db                  *gorm.DB
	recommendations     []*EnhancedIndexRecommendation
	queryPatterns       map[string]*QueryPatternAnalysis
	lastAnalysis        time.Time
	analysisInterval    time.Duration
	enableAutoCreate    bool
	enableAutoAnalyze   bool
	enableAutoDrop      bool
	minQueryCount       int64
	minImprovementRatio float64
	mu                  sync.RWMutex
	optimizerStats      *OptimizerStats
}

type EnhancedIndexRecommendation struct {
	TableName         string
	IndexName         string
	Columns           []string
	IndexType         string
	Priority          string
	EstimatedSize     string
	QueryBenefits     []string
	CreationSQL       string
	EstimatedImpact   string
	SupportingQueries []string
	Confidence        float64
	Action            string
}

type QueryPatternAnalysis struct {
	Pattern           string
	TableNames        []string
	ColumnUsage       map[string]int64
	JoinCount         int64
	WhereCount        int64
	OrderByCount      int64
	GroupByCount      int64
	ExecutionCount    int64
	AvgDuration       time.Duration
	MaxDuration       time.Duration
	LastExecuted      time.Time
	IndexBenefit      float64
	RecommendedIndex  *EnhancedIndexRecommendation
}

type OptimizerStats struct {
	TotalAnalyses      int64
	IndexesRecommended int64
	IndexesCreated     int64
	IndexesDropped     int64
	QueriesAnalyzed    int64
	LastAnalysisTime   time.Time
	TotalSavings       time.Duration
}

type IndexAnalysisResult struct {
	TableName          string
	CurrentIndexes     []IndexDetail
	MissingIndexes     []*EnhancedIndexRecommendation
	RedundantIndexes   []RedundantIndex
	UnusedIndexes      []string
	TableStats         TableStatistics
	Recommendations    []*EnhancedIndexRecommendation
}

type IndexDetail struct {
	IndexName     string
	Columns       []string
	IndexType     string
	IsUnique      bool
	IsPrimary     bool
	Size          string
	UsageCount    int64
	LastUsed      time.Time
}

type TableStatistics struct {
	TableName      string
	RowCount       int64
	TableSize      string
	IndexSize      string
	TotalSize      string
	IndexCount     int
	IndexRatio     float64
	Fragmentation  float64
	LastVacuum     time.Time
	LastAnalyze    time.Time
}

var enhancedIndexOptimizer *EnhancedIndexOptimizer

func NewEnhancedIndexOptimizer(db *gorm.DB) *EnhancedIndexOptimizer {
	return &EnhancedIndexOptimizer{
		db:                 db,
		recommendations:    make([]*EnhancedIndexRecommendation, 0),
		queryPatterns:      make(map[string]*QueryPatternAnalysis),
		analysisInterval:   24 * time.Hour,
		enableAutoCreate:   true,
		enableAutoAnalyze:  true,
		enableAutoDrop:     false,
		minQueryCount:      100,
		minImprovementRatio: 0.3,
		optimizerStats:     &OptimizerStats{},
	}
}

func InitEnhancedIndexOptimizer(db *gorm.DB) {
	if enhancedIndexOptimizer == nil {
		enhancedIndexOptimizer = NewEnhancedIndexOptimizer(db)
	}
}

func GetEnhancedIndexOptimizer() *EnhancedIndexOptimizer {
	return enhancedIndexOptimizer
}

func (o *EnhancedIndexOptimizer) Analyze(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if time.Since(o.lastAnalysis) < o.analysisInterval {
		return nil
	}

	log.Println("[INDEX_OPTIMIZER_V2] Starting comprehensive index analysis...")

	o.recommendations = make([]*EnhancedIndexRecommendation, 0)

	if err := o.analyzeQueryPatterns(ctx); err != nil {
		log.Printf("[INDEX_OPTIMIZER_V2] Failed to analyze query patterns: %v", err)
	}

	if err := o.analyzeTableIndexes(ctx); err != nil {
		log.Printf("[INDEX_OPTIMIZER_V2] Failed to analyze table indexes: %v", err)
	}

	if err := o.identifyRedundantIndexes(ctx); err != nil {
		log.Printf("[INDEX_OPTIMIZER_V2] Failed to identify redundant indexes: %v", err)
	}

	o.lastAnalysis = time.Now()
	o.optimizerStats.TotalAnalyses++
	o.optimizerStats.LastAnalysisTime = time.Now()

	log.Printf("[INDEX_OPTIMIZER_V2] Analysis completed. Found %d recommendations", len(o.recommendations))
	return nil
}

func (o *EnhancedIndexOptimizer) analyzeQueryPatterns(ctx context.Context) error {
	var queries []struct {
		Query       string
		Executions  int64
		AvgLatency  float64
	}

	err := o.db.Raw(`
		SELECT query, calls as executions, total_time / calls as avg_latency
		FROM pg_stat_statements
		WHERE calls > $1
		ORDER BY total_time DESC
		LIMIT 100
	`, o.minQueryCount).Scan(&queries).Error

	if err != nil {
		return fmt.Errorf("failed to query pg_stat_statements: %w", err)
	}

	for _, q := range queries {
		analysis := o.analyzeSingleQuery(q.Query, q.Executions, time.Duration(q.AvgLatency*float64(time.Millisecond)))
		o.queryPatterns[q.Query] = analysis

		if analysis.RecommendedIndex != nil {
			o.recommendations = append(o.recommendations, analysis.RecommendedIndex)
		}
	}

	o.optimizerStats.QueriesAnalyzed += int64(len(queries))
	return nil
}

func (o *EnhancedIndexOptimizer) analyzeSingleQuery(query string, executions int64, avgLatency time.Duration) *QueryPatternAnalysis {
	analysis := &QueryPatternAnalysis{
		Pattern:       query,
		ColumnUsage:   make(map[string]int64),
		ExecutionCount: executions,
		AvgDuration:   avgLatency,
	}

	analysis.TableNames = extractTableNames(query)
	analysis.ColumnUsage = extractColumnUsage(query)
	analysis.WhereCount = countOccurrences(query, "WHERE")
	analysis.OrderByCount = countOccurrences(query, "ORDER BY")
	analysis.GroupByCount = countOccurrences(query, "GROUP BY")
	analysis.JoinCount = countOccurrences(query, "JOIN")

	if avgLatency > 50*time.Millisecond && executions > o.minQueryCount {
		recommendation := o.generateIndexRecommendation(analysis)
		if recommendation != nil {
			analysis.RecommendedIndex = recommendation
			analysis.IndexBenefit = estimateBenefit(analysis)
		}
	}

	return analysis
}

func extractTableNames(query string) []string {
	var tables []string
	keywords := []string{"FROM ", "JOIN ", "INNER JOIN ", "LEFT JOIN ", "RIGHT JOIN "}

	for _, kw := range keywords {
		idx := strings.Index(strings.ToUpper(query), kw)
		if idx != -1 {
			remaining := query[idx+len(kw):]
			endIdx := strings.IndexAny(remaining, " ,\n\r")
			if endIdx == -1 {
				endIdx = len(remaining)
			}
			tableName := strings.TrimSpace(remaining[:endIdx])
			if tableName != "" {
				tables = append(tables, tableName)
			}
		}
	}

	return tables
}

func extractColumnUsage(query string) map[string]int64 {
	usage := make(map[string]int64)
	whereStart := strings.Index(strings.ToUpper(query), "WHERE")
	if whereStart != -1 {
		whereClause := query[whereStart+6:]
		columns := strings.FieldsFunc(whereClause, func(r rune) bool {
			return strings.ContainsRune("=<>!><= >=AND OR()", r)
		})
		for _, col := range columns {
			col = strings.TrimSpace(col)
			if col != "" && !strings.Contains(strings.ToUpper(col), "SELECT") && !strings.Contains(strings.ToUpper(col), "FROM") {
				usage[col]++
			}
		}
	}
	return usage
}

func countOccurrences(str, substr string) int64 {
	count := int64(0)
	idx := 0
	upperStr := strings.ToUpper(str)
	upperSubstr := strings.ToUpper(substr)

	for {
		idx = strings.Index(upperStr[idx:], upperSubstr)
		if idx == -1 {
			break
		}
		count++
		idx += len(upperSubstr)
	}
	return count
}

func (o *EnhancedIndexOptimizer) generateIndexRecommendation(analysis *QueryPatternAnalysis) *EnhancedIndexRecommendation {
	if len(analysis.TableNames) == 0 || len(analysis.ColumnUsage) == 0 {
		return nil
	}

	tableName := analysis.TableNames[0]
	var priority string
	var estimatedImpact string

	if analysis.AvgDuration > 200*time.Millisecond {
		priority = "high"
		estimatedImpact = "significant"
	} else if analysis.AvgDuration > 100*time.Millisecond {
		priority = "medium"
		estimatedImpact = "moderate"
	} else {
		priority = "low"
		estimatedImpact = "minor"
	}

	columns := make([]string, 0, len(analysis.ColumnUsage))
	for col := range analysis.ColumnUsage {
		columns = append(columns, col)
	}

	indexName := fmt.Sprintf("idx_%s_opt_%d", tableName, time.Now().UnixNano()%100000)

	return &EnhancedIndexRecommendation{
		TableName:         tableName,
		IndexName:         indexName,
		Columns:           columns,
		IndexType:         "btree",
		Priority:          priority,
		EstimatedSize:     estimateIndexSize(tableName, len(columns)),
		QueryBenefits:     []string{fmt.Sprintf("优化查询模式，执行次数: %d", analysis.ExecutionCount)},
		CreationSQL:       fmt.Sprintf("CREATE INDEX CONCURRENTLY %s ON %s (%s)", indexName, tableName, strings.Join(columns, ", ")),
		EstimatedImpact:   estimatedImpact,
		SupportingQueries: []string{analysis.Pattern[:min(50, len(analysis.Pattern))] + "..."},
		Confidence:        calculateConfidence(analysis),
		Action:            "create",
	}
}

func estimateIndexSize(tableName string, columnCount int) string {
	baseSizeMB := 10 + (columnCount * 5)
	return fmt.Sprintf("%dMB", baseSizeMB)
}

func estimateBenefit(analysis *QueryPatternAnalysis) float64 {
	benefit := 0.0
	if analysis.AvgDuration > 100*time.Millisecond {
		benefit += 0.4
	}
	if analysis.ExecutionCount > 1000 {
		benefit += 0.3
	}
	if analysis.WhereCount > 2 {
		benefit += 0.2
	}
	if analysis.JoinCount > 0 {
		benefit += 0.1
	}
	return benefit
}

func calculateConfidence(analysis *QueryPatternAnalysis) float64 {
	confidence := 0.5
	if analysis.ExecutionCount > 1000 {
		confidence += 0.2
	}
	if analysis.AvgDuration > 100*time.Millisecond {
		confidence += 0.15
	}
	if len(analysis.ColumnUsage) > 1 {
		confidence += 0.1
	}
	if analysis.OrderByCount > 0 {
		confidence += 0.05
	}
	if confidence > 0.95 {
		return 0.95
	}
	return confidence
}

func (o *EnhancedIndexOptimizer) analyzeTableIndexes(ctx context.Context) error {
	var tables []string
	err := o.db.Raw(`
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public'
	`).Scan(&tables).Error

	if err != nil {
		return fmt.Errorf("failed to get table list: %w", err)
	}

	for _, table := range tables {
		stats, err := o.getTableStatistics(table)
		if err != nil {
			continue
		}

		if stats.RowCount > 10000 && stats.IndexRatio < 0.1 {
			recommendations, err := o.recommendIndexesForTable(table)
			if err != nil {
				continue
			}
			o.recommendations = append(o.recommendations, recommendations...)
		}
	}

	return nil
}

func (o *EnhancedIndexOptimizer) getTableStatistics(tableName string) (*TableStatistics, error) {
	stats := &TableStatistics{TableName: tableName}

	err := o.db.Raw(`
		SELECT reltuples::bigint AS row_count
		FROM pg_class
		WHERE relname = ?
	`, tableName).Scan(&stats.RowCount).Error

	if err != nil {
		return nil, err
	}

	var sizes []struct {
		TableSize string
		IndexSize string
		TotalSize string
	}

	err = o.db.Raw(`
		SELECT
			pg_size_pretty(pg_relation_size(?)) AS table_size,
			pg_size_pretty(pg_indexes_size(?)) AS index_size,
			pg_size_pretty(pg_total_relation_size(?)) AS total_size
	`, tableName, tableName, tableName).Scan(&sizes).Error

	if err == nil && len(sizes) > 0 {
		stats.TableSize = sizes[0].TableSize
		stats.IndexSize = sizes[0].IndexSize
		stats.TotalSize = sizes[0].TotalSize
	}

	err = o.db.Raw(`
		SELECT COUNT(*) AS index_count
		FROM pg_indexes
		WHERE tablename = ?
	`, tableName).Scan(&stats.IndexCount).Error

	return stats, nil
}

func (o *EnhancedIndexOptimizer) recommendIndexesForTable(tableName string) ([]*EnhancedIndexRecommendation, error) {
	var recommendations []*EnhancedIndexRecommendation

	var columns []struct {
		ColumnName string
		DataType   string
		Nullable   string
	}

	err := o.db.Raw(`
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = ?
	`, tableName).Scan(&columns).Error

	if err != nil {
		return nil, err
	}

	var highFreqColumns []string
	for _, col := range columns {
		if col.Nullable == "NO" {
			highFreqColumns = append(highFreqColumns, col.ColumnName)
		}
	}

	if len(highFreqColumns) >= 2 {
		indexName := fmt.Sprintf("idx_%s_composite", tableName)
		recommendations = append(recommendations, &EnhancedIndexRecommendation{
			TableName:       tableName,
			IndexName:       indexName,
			Columns:         highFreqColumns[:min(3, len(highFreqColumns))],
			IndexType:       "btree",
			Priority:        "medium",
			EstimatedSize:   "20MB",
			QueryBenefits:   []string{"复合索引优化多条件查询"},
			CreationSQL:     fmt.Sprintf("CREATE INDEX CONCURRENTLY %s ON %s (%s)", indexName, tableName, strings.Join(highFreqColumns[:min(3, len(highFreqColumns))], ", ")),
			EstimatedImpact: "moderate",
			Confidence:      0.7,
			Action:          "create",
		})
	}

	return recommendations, nil
}

func (o *EnhancedIndexOptimizer) identifyRedundantIndexes(ctx context.Context) error {
	var indexes []struct {
		IndexName string
		TableName string
		IndexDef  string
	}

	err := o.db.Raw(`
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
	`).Scan(&indexes).Error

	if err != nil {
		return fmt.Errorf("failed to get indexes: %w", err)
	}

	indexMap := make(map[string][]*indexInfo)
	for _, idx := range indexes {
		columns := extractColumns(idx.IndexDef)
		indexMap[idx.TableName] = append(indexMap[idx.TableName], &indexInfo{
			IndexName: idx.IndexName,
			TableName: idx.TableName,
			Columns:   columns,
		})
	}

	for tableName, tableIndexes := range indexMap {
		for i, idx1 := range tableIndexes {
			for j := i + 1; j < len(tableIndexes); j++ {
				idx2 := tableIndexes[j]
				if o.isRedundant(idx1, idx2) {
					o.recommendations = append(o.recommendations, &EnhancedIndexRecommendation{
						TableName:       tableName,
						IndexName:       idx2.IndexName,
						Columns:         idx2.Columns,
						IndexType:       "btree",
						Priority:        "low",
						EstimatedSize:   "N/A",
						QueryBenefits:   []string{"移除冗余索引"},
						CreationSQL:     fmt.Sprintf("DROP INDEX CONCURRENTLY %s", idx2.IndexName),
						EstimatedImpact: "storage",
						Confidence:      0.9,
						Action:          "drop",
					})
				}
			}
		}
	}

	return nil
}

func extractColumns(indexDef string) []string {
	parts := strings.Split(indexDef, "(")
	if len(parts) < 2 {
		return []string{}
	}
	colsPart := strings.Split(parts[1], ")")[0]
	cols := strings.Split(colsPart, ",")
	var columns []string
	for _, col := range cols {
		col = strings.TrimSpace(col)
		col = strings.Split(col, " ")[0]
		if col != "" {
			columns = append(columns, col)
		}
	}
	return columns
}

func (o *EnhancedIndexOptimizer) isRedundant(idx1, idx2 *indexInfo) bool {
	if len(idx1.Columns) > len(idx2.Columns) {
		idx1, idx2 = idx2, idx1
	}

	for i, col := range idx1.Columns {
		if i >= len(idx2.Columns) || idx2.Columns[i] != col {
			return false
		}
	}
	return true
}

func (o *EnhancedIndexOptimizer) GetRecommendations() []*EnhancedIndexRecommendation {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.recommendations
}

func (o *EnhancedIndexOptimizer) ApplyRecommendation(ctx context.Context, rec *EnhancedIndexRecommendation) error {
	if o.db == nil {
		return fmt.Errorf("database not initialized")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if rec.Action == "create" {
		var count int64
		err := o.db.Raw(`SELECT COUNT(*) FROM pg_indexes WHERE indexname = ?`, rec.IndexName).Scan(&count).Error
		if err != nil {
			return fmt.Errorf("failed to check index existence: %w", err)
		}
		if count > 0 {
			return fmt.Errorf("index %s already exists", rec.IndexName)
		}

		if err := o.db.Exec(rec.CreationSQL).Error; err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
		o.optimizerStats.IndexesCreated++
		log.Printf("[INDEX_OPTIMIZER_V2] Created index: %s", rec.IndexName)

	} else if rec.Action == "drop" {
		if !o.enableAutoDrop {
			return fmt.Errorf("auto-drop is disabled")
		}

		if err := o.db.Exec(rec.CreationSQL).Error; err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}
		o.optimizerStats.IndexesDropped++
		log.Printf("[INDEX_OPTIMIZER_V2] Dropped index: %s", rec.IndexName)
	}

	return nil
}

func (o *EnhancedIndexOptimizer) ApplyAllRecommendations(ctx context.Context, minPriority string) error {
	o.mu.RLock()
	recommendations := o.recommendations
	o.mu.RUnlock()

	priorityOrder := map[string]int{"high": 3, "medium": 2, "low": 1}
	minPrio := priorityOrder[minPriority]

	for _, rec := range recommendations {
		if priorityOrder[rec.Priority] >= minPrio {
			if err := o.ApplyRecommendation(ctx, rec); err != nil {
				log.Printf("[INDEX_OPTIMIZER_V2] Failed to apply recommendation for %s: %v", rec.IndexName, err)
			}
		}
	}

	return nil
}

func (o *EnhancedIndexOptimizer) GetOptimizerStats() *OptimizerStats {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.optimizerStats
}

func (o *EnhancedIndexOptimizer) SetAutoCreate(enabled bool) {
	o.mu.Lock()
	o.enableAutoCreate = enabled
	o.mu.Unlock()
}

func (o *EnhancedIndexOptimizer) SetAutoDrop(enabled bool) {
	o.mu.Lock()
	o.enableAutoDrop = enabled
	o.mu.Unlock()
}

func (o *EnhancedIndexOptimizer) AnalyzeTable(ctx context.Context, tableName string) (*IndexAnalysisResult, error) {
	result := &IndexAnalysisResult{TableName: tableName}

	currentIndexes, err := o.getCurrentIndexes(tableName)
	if err != nil {
		return nil, err
	}
	result.CurrentIndexes = currentIndexes

	tableStats, err := o.getTableStatistics(tableName)
	if err != nil {
		return nil, err
	}
	result.TableStats = *tableStats

	unusedIndexes, err := o.findUnusedIndexes(tableName)
	if err != nil {
		return nil, err
	}
	result.UnusedIndexes = unusedIndexes

	recommendations, err := o.recommendIndexesForTable(tableName)
	if err != nil {
		return nil, err
	}
	result.Recommendations = recommendations

	return result, nil
}

func (o *EnhancedIndexOptimizer) getCurrentIndexes(tableName string) ([]IndexDetail, error) {
	var details []IndexDetail

	rows, err := o.db.Raw(`
		SELECT
			i.relname AS index_name,
			pg_get_indexdef(i.oid) AS index_def,
			pg_size_pretty(pg_relation_size(i.oid)) AS size,
			COALESCE(idx_scan, 0) AS usage_count,
			idx.indisunique AS is_unique,
			idx.indisprimary AS is_primary
		FROM pg_index idx
		JOIN pg_class t ON t.oid = idx.indrelid
		JOIN pg_class i ON i.oid = idx.indexrelid
		LEFT JOIN pg_stat_user_indexes ui ON ui.indexrelid = i.oid
		WHERE t.relname = ?
	`, tableName).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var detail IndexDetail
		var indexDef, size string
		var usageCount int64
		var isUnique, isPrimary bool

		if err := rows.Scan(&detail.IndexName, &indexDef, &size, &usageCount, &isUnique, &isPrimary); err != nil {
			continue
		}

		detail.Columns = extractColumns(indexDef)
		detail.IndexType = "btree"
		detail.IsUnique = isUnique
		detail.IsPrimary = isPrimary
		detail.Size = size
		detail.UsageCount = usageCount

		details = append(details, detail)
	}

	return details, nil
}

func (o *EnhancedIndexOptimizer) findUnusedIndexes(tableName string) ([]string, error) {
	var unused []string

	err := o.db.Raw(`
		SELECT indexname
		FROM pg_stat_user_indexes
		WHERE relname = ?
		AND idx_scan = 0
	`, tableName).Scan(&unused).Error

	return unused, err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}