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
)

type SlowQueryAnalyzer struct {
	mu              sync.RWMutex
	threshold       time.Duration
	maxRecords      int
	records         []SlowQueryRecord
	patternAnalyzer *PatternAnalyzer
	enabled         bool
}

type SlowQueryRecord struct {
	ID            string        `json:"id"`
	Query         string        `json:"query"`
	Duration      time.Duration `json:"duration"`
	Timestamp     time.Time     `json:"timestamp"`
	TableName     string        `json:"table_name"`
	QueryType     string        `json:"query_type"`
	RowsAffected  int64         `json:"rows_affected"`
	Error         error         `json:"error,omitempty"`
	Context       string        `json:"context,omitempty"`
	ExecutionPlan string        `json:"execution_plan,omitempty"`
}

type PatternAnalyzer struct {
	mu            sync.RWMutex
	patterns      map[string]*QueryPattern
	patternRegex  *regexp.Regexp
	tableRegex    *regexp.Regexp
	columnRegex   *regexp.Regexp
}

type QueryPattern struct {
	Pattern        string        `json:"pattern"`
	Occurrences   int           `json:"occurrences"`
	AvgDuration   time.Duration `json:"avg_duration"`
	TotalDuration time.Duration `json:"total_duration"`
	LastSeen      time.Time     `json:"last_seen"`
	Suggestions   []string      `json:"suggestions"`
}

type SlowQueryAnalysis struct {
	TotalQueries       int64              `json:"total_queries"`
	AvgDuration        time.Duration      `json:"avg_duration"`
	MaxDuration        time.Duration      `json:"max_duration"`
	MinDuration        time.Duration      `json:"min_duration"`
	SlowQueries       []SlowQueryRecord  `json:"slow_queries"`
	TopTables         []TableStats       `json:"top_tables"`
	TopPatterns        []QueryPattern     `json:"top_patterns"`
	Recommendations    []string           `json:"recommendations"`
	QueryTypeStats     map[string]int    `json:"query_type_stats"`
}

type TableStats struct {
	TableName     string        `json:"table_name"`
	QueryCount   int           `json:"query_count"`
	AvgDuration  time.Duration `json:"avg_duration"`
	TotalDuration time.Duration `json:"total_duration"`
}

var slowQueryAnalyzer *SlowQueryAnalyzer

func InitSlowQueryAnalyzer(cfg *config.Config) error {
	slowQueryAnalyzer = &SlowQueryAnalyzer{
		threshold:  time.Duration(cfg.Database.SlowQueryThresholdMs) * time.Millisecond,
		maxRecords: 10000,
		records:    make([]SlowQueryRecord, 0),
		enabled:    cfg.Database.Monitoring.EnableQueryMetrics,
		patternAnalyzer: &PatternAnalyzer{
			patterns:     make(map[string]*QueryPattern),
			patternRegex: regexp.MustCompile(`(?i)(SELECT|INSERT|UPDATE|DELETE|FROM|WHERE|JOIN|ORDER BY|GROUP BY|HAVING|LIMIT|OFFSET)`),
			tableRegex:   regexp.MustCompile(`(?i)FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
			columnRegex:  regexp.MustCompile(`(?i)WHERE\s+([a-zA-Z_][a-zA-Z0-9_\.]*)`),
		},
	}

	if cfg.Database.Monitoring.EnableQueryMetrics {
		go slowQueryAnalyzer.startPeriodicAnalysis()
	}

	log.Println("Slow query analyzer initialized")
	return nil
}

func GetSlowQueryAnalyzer() *SlowQueryAnalyzer {
	return slowQueryAnalyzer
}

func (a *SlowQueryAnalyzer) RecordSlowQuery(query string, duration time.Duration, err error, rowsAffected int64, context string) {
	if !a.enabled || duration < a.threshold {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	record := SlowQueryRecord{
		ID:           generateUUID(),
		Query:         sanitizeQuery(query),
		Duration:     duration,
		Timestamp:    time.Now(),
		TableName:    extractTableName(query),
		QueryType:    identifyQueryType(query),
		RowsAffected: rowsAffected,
		Error:        err,
		Context:      context,
	}

	a.records = append(a.records, record)

	if len(a.records) > a.maxRecords {
		a.records = a.records[1:]
	}

	a.patternAnalyzer.analyzeQueryPattern(query, duration)
}

func (a *SlowQueryAnalyzer) Analyze(ctx context.Context, limit int) (*SlowQueryAnalysis, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.records) == 0 {
		return &SlowQueryAnalysis{}, nil
	}

	analysis := &SlowQueryAnalysis{
		TotalQueries:   int64(len(a.records)),
		SlowQueries:    make([]SlowQueryRecord, 0),
		TopTables:      make([]TableStats, 0),
		TopPatterns:    make([]QueryPattern, 0),
		Recommendations: make([]string, 0),
		QueryTypeStats: make(map[string]int),
	}

	var totalDuration time.Duration
	analysis.MaxDuration = a.records[0].Duration
	analysis.MinDuration = a.records[0].Duration

	tableStats := make(map[string]*TableStats)
	queryTypeCount := make(map[string]int)

	for _, record := range a.records {
		totalDuration += record.Duration

		if record.Duration > analysis.MaxDuration {
			analysis.MaxDuration = record.Duration
		}
		if record.Duration < analysis.MinDuration {
			analysis.MinDuration = record.Duration
		}

		queryTypeCount[record.QueryType]++
		if tableStats[record.TableName] == nil {
			tableStats[record.TableName] = &TableStats{TableName: record.TableName}
		}
		tableStats[record.TableName].QueryCount++
		tableStats[record.TableName].TotalDuration += record.Duration
	}

	analysis.AvgDuration = totalDuration / time.Duration(analysis.TotalQueries)
	analysis.QueryTypeStats = queryTypeCount

	for _, stats := range tableStats {
		stats.AvgDuration = stats.TotalDuration / time.Duration(stats.QueryCount)
	}

	tableStatsList := make([]TableStats, 0, len(tableStats))
	for _, stats := range tableStats {
		tableStatsList = append(tableStatsList, *stats)
	}
	sort.Slice(tableStatsList, func(i, j int) bool {
		return tableStatsList[i].TotalDuration > tableStatsList[j].TotalDuration
	})
	if len(tableStatsList) > 10 {
		analysis.TopTables = tableStatsList[:10]
	} else {
		analysis.TopTables = tableStatsList
	}

	if limit <= 0 || limit > len(a.records) {
		limit = len(a.records)
	}
	analysis.SlowQueries = a.records[len(a.records)-limit:]

	analysis.Recommendations = a.generateRecommendations(tableStats, queryTypeCount)

	patterns := a.patternAnalyzer.getTopPatterns(10)
	analysis.TopPatterns = patterns

	return analysis, nil
}

func (a *SlowQueryAnalyzer) generateRecommendations(tableStats map[string]*TableStats, queryTypeStats map[string]int) []string {
	recommendations := make([]string, 0)

	for tableName, stats := range tableStats {
		if stats.QueryCount > 100 && stats.AvgDuration > a.threshold*2 {
			recommendations = append(recommendations, fmt.Sprintf(
				"Table '%s' has high query count (%d) and slow average duration. Consider adding indexes or optimizing queries.",
				tableName, stats.QueryCount))
		}

		if selectCount, ok := queryTypeStats["SELECT"]; ok && selectCount > 50 {
			if stats.AvgDuration > a.threshold {
				recommendations = append(recommendations, fmt.Sprintf(
					"SELECT queries on '%s' are slow. Consider adding covering indexes for frequently queried columns.",
					tableName))
			}
		}
	}

	if deleteCount, ok := queryTypeStats["DELETE"]; ok && deleteCount > 10 {
		recommendations = append(recommendations, fmt.Sprintf(
			"Multiple DELETE operations detected (%d). Consider batch deletes or archiving old data.",
			deleteCount))
	}

	if updateCount, ok := queryTypeStats["UPDATE"]; ok && updateCount > 20 {
		recommendations = append(recommendations, fmt.Sprintf(
			"High UPDATE frequency (%d). Ensure indexes cover WHERE clause columns to avoid full table scans.",
			updateCount))
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "No critical performance issues detected. Continue monitoring.")
	}

	return recommendations
}

func (a *SlowQueryAnalyzer) GetExecutionPlan(ctx context.Context, query string) (string, error) {
	if DB == nil {
		return "", fmt.Errorf("database connection not available")
	}

	explainQuery := fmt.Sprintf("EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) %s", query)
	var plan string
	err := DB.WithContext(ctx).Raw(explainQuery).Scan(&plan).Error
	if err != nil {
		return "", fmt.Errorf("failed to get execution plan: %w", err)
	}

	return plan, nil
}

func (a *SlowQueryAnalyzer) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.records = make([]SlowQueryRecord, 0)
	a.patternAnalyzer.clear()
}

func (a *SlowQueryAnalyzer) GetRecords(limit, offset int) []SlowQueryRecord {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > len(a.records) {
		limit = len(a.records)
	}
	if offset >= len(a.records) {
		return []SlowQueryRecord{}
	}

	endIndex := offset + limit
	if endIndex > len(a.records) {
		endIndex = len(a.records)
	}

	records := make([]SlowQueryRecord, endIndex-offset)
	copy(records, a.records[offset:endIndex])
	return records
}

func (a *SlowQueryAnalyzer) startPeriodicAnalysis() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		a.performPeriodicAnalysis()
	}
}

func (a *SlowQueryAnalyzer) performPeriodicAnalysis() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	analysis, err := a.Analyze(ctx, 100)
	if err != nil {
		log.Printf("Failed to perform periodic analysis: %v", err)
		return
	}

	if len(analysis.Recommendations) > 0 {
		log.Printf("[SLOW_QUERY_ANALYSIS] Total slow queries: %d, Avg duration: %v, Top tables: %d",
			analysis.TotalQueries, analysis.AvgDuration, len(analysis.TopTables))
	}
}

func (p *PatternAnalyzer) analyzeQueryPattern(query string, duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	normalizedQuery := normalizeQuery(query)
	if _, exists := p.patterns[normalizedQuery]; !exists {
		p.patterns[normalizedQuery] = &QueryPattern{
			Pattern:      normalizedQuery,
			Suggestions: p.generateSuggestions(normalizedQuery),
		}
	}

	pattern := p.patterns[normalizedQuery]
	pattern.Occurrences++
	pattern.TotalDuration += duration
	pattern.AvgDuration = pattern.TotalDuration / time.Duration(pattern.Occurrences)
	pattern.LastSeen = time.Now()
}

func (p *PatternAnalyzer) generateSuggestions(query string) []string {
	suggestions := make([]string, 0)

	if strings.Contains(strings.ToUpper(query), "SELECT *") {
		suggestions = append(suggestions, "Avoid using SELECT *, specify columns explicitly")
	}

	if strings.Contains(strings.ToUpper(query), "LIKE '%") {
		suggestions = append(suggestions, "Leading wildcard in LIKE prevents index usage. Consider full-text search")
	}

	if !strings.Contains(strings.ToUpper(query), "LIMIT") {
		suggestions = append(suggestions, "Consider adding LIMIT to restrict result set size")
	}

	if !strings.Contains(strings.ToUpper(query), "WHERE") {
		suggestions = append(suggestions, "Query without WHERE clause may scan entire table")
	}

	subqueryCount := strings.Count(strings.ToUpper(query), "SELECT")
	if subqueryCount > 2 {
		suggestions = append(suggestions, "Multiple subqueries detected. Consider using JOIN or CTEs instead")
	}

	if strings.Contains(strings.ToUpper(query), "ORDER BY") && !strings.Contains(strings.ToUpper(query), "LIMIT") {
		suggestions = append(suggestions, "ORDER BY without LIMIT may cause sorting overhead for large datasets")
	}

	return suggestions
}

func (p *PatternAnalyzer) getTopPatterns(limit int) []QueryPattern {
	p.mu.RLock()
	defer p.mu.RUnlock()

	patterns := make([]QueryPattern, 0, len(p.patterns))
	for _, pattern := range p.patterns {
		patterns = append(patterns, *pattern)
	}

	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].TotalDuration > patterns[j].TotalDuration
	})

	if len(patterns) > limit {
		return patterns[:limit]
	}
	return patterns
}

func (p *PatternAnalyzer) clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.patterns = make(map[string]*QueryPattern)
}

func generateUUID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().UnixNano()%10000)
}

func sanitizeQuery(query string) string {
	re := regexp.MustCompile(`'(?:[^'\\]|\\.)*'`)
	sanitized := re.ReplaceAllStringFunc(query, func(match string) string {
		return "'***'"
	})
	return strings.TrimSpace(sanitized)
}

func extractTableName(query string) string {
	matches := tableRegex.FindStringSubmatch(query)
	if len(matches) > 1 {
		return matches[1]
	}
	return "unknown"
}

func identifyQueryType(query string) string {
	upperQuery := strings.ToUpper(strings.TrimSpace(query))
	for _, prefix := range []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER"} {
		if strings.HasPrefix(upperQuery, prefix) {
			return prefix
		}
	}
	return "OTHER"
}

func normalizeQuery(query string) string {
	re := regexp.MustCompile(`'(?:[^'\\]|\\.)*'`)
	normalized := re.ReplaceAllString(query, "'?'")

	normalized = regexp.MustCompile(`\d+`).ReplaceAllString(normalized, "?")

	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	return strings.TrimSpace(normalized)
}

var tableRegex = regexp.MustCompile(`(?i)FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
