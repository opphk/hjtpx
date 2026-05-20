package database

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

type OptimizedQueryAnalyzer struct {
	db                 *gorm.DB
	slowThreshold      time.Duration
	queryPatterns      map[string]*QueryPattern
	mu                 sync.RWMutex
}

type QueryPattern struct {
	Pattern       string
	Count         int64
	TotalDuration time.Duration
	AvgDuration   time.Duration
	MaxDuration   time.Duration
	MinDuration   time.Duration
	Complexity    int
	Suggestions   []string
}

var optimizedAnalyzer *OptimizedQueryAnalyzer

func NewOptimizedQueryAnalyzer(db *gorm.DB, threshold time.Duration) *OptimizedQueryAnalyzer {
	return &OptimizedQueryAnalyzer{
		db:            db,
		slowThreshold: threshold,
		queryPatterns: make(map[string]*QueryPattern),
	}
}

func (a *OptimizedQueryAnalyzer) AnalyzeSlowQueries() ([]SlowQueryResult, error) {
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
		LIMIT 50
	`, a.slowThreshold.Milliseconds()).Scan(&results).Error

	if err != nil {
		log.Printf("[QUERY_ANALYZER] Failed to query: %v", err)
		return nil, err
	}

	for i := range results {
		results[i].Suggestions = a.GenerateSuggestions(results[i].QueryPreview, results[i].MeanTimeMs)
	}

	return results, nil
}

func (a *OptimizedQueryAnalyzer) GenerateSuggestions(query string, meanTimeMs float64) []string {
	var suggestions []string

	if meanTimeMs > 100 {
		suggestions = append(suggestions, fmt.Sprintf("HIGH: Query very slow (%.1fms), consider adding indexes", meanTimeMs))
		suggestions = append(suggestions, "Review with EXPLAIN ANALYZE")
	} else if meanTimeMs > 50 {
		suggestions = append(suggestions, fmt.Sprintf("MEDIUM: Query slow (%.1fms), consider optimization", meanTimeMs))
	}

	if strings.Contains(strings.ToUpper(query), "SELECT *") {
		suggestions = append(suggestions, "Use specific columns instead of SELECT *")
	}

	if strings.Contains(strings.ToUpper(query), "LIKE '%") {
		suggestions = append(suggestions, "Leading wildcard prevents index usage, consider full-text search")
	}

	if strings.Contains(strings.ToUpper(query), "NOT IN") {
		suggestions = append(suggestions, "Consider NOT EXISTS or LEFT JOIN WHERE NULL")
	}

	if strings.Contains(strings.ToUpper(query), "OR ") {
		suggestions = append(suggestions, "OR conditions may prevent index usage, consider UNION")
	}

	return suggestions
}

func (a *OptimizedQueryAnalyzer) RecordQuery(query string, duration time.Duration) {
	pattern := normalizeQuery(query)

	a.mu.Lock()
	defer a.mu.Unlock()

	if p, exists := a.queryPatterns[pattern]; exists {
		p.Count++
		p.TotalDuration += duration
		p.AvgDuration = p.TotalDuration / time.Duration(p.Count)
		if duration > p.MaxDuration {
			p.MaxDuration = duration
		}
		if p.MinDuration == 0 || duration < p.MinDuration {
			p.MinDuration = duration
		}
	} else {
		a.queryPatterns[pattern] = &QueryPattern{
			Pattern:       pattern,
			Count:         1,
			TotalDuration: duration,
			AvgDuration:   duration,
			MaxDuration:   duration,
			MinDuration:   duration,
		}
	}
}

func (a *OptimizedQueryAnalyzer) GetTopPatterns(limit int) []*QueryPattern {
	a.mu.RLock()
	defer a.mu.RUnlock()

	patterns := make([]*QueryPattern, 0, len(a.queryPatterns))
	for _, p := range a.queryPatterns {
		patterns = append(patterns, p)
	}

	for i := 0; i < len(patterns)-1; i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[j].AvgDuration > patterns[i].AvgDuration {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}

	if limit > 0 && len(patterns) > limit {
		return patterns[:limit]
	}
	return patterns
}

func (a *OptimizedQueryAnalyzer) GenerateReport() map[string]interface{} {
	patterns := a.GetTopPatterns(10)

	var totalDuration time.Duration
	var totalQueries int64
	for _, p := range patterns {
		totalDuration += p.TotalDuration
		totalQueries += p.Count
	}

	avgDuration := time.Duration(0)
	if totalQueries > 0 {
		avgDuration = totalDuration / time.Duration(totalQueries)
	}

	return map[string]interface{}{
		"timestamp":         time.Now(),
		"total_patterns":    len(patterns),
		"total_queries":     totalQueries,
		"avg_duration":      avgDuration.String(),
		"top_patterns":      patterns,
		"slow_query_count":  len(patterns),
	}
}

func normalizeQuery(query string) string {
	for i := 0; i < 10; i++ {
		query = strings.Replace(query, fmt.Sprintf("%d", i), "?", 1)
	}

	parts := strings.Split(query, "'")
	if len(parts) > 1 {
		query = parts[0] + "?" + strings.Join(parts[2:], "'")
	}

	return strings.TrimSpace(query)
}

func InitOptimizedQueryAnalyzer(db *gorm.DB, thresholdMs int) {
	optimizedAnalyzer = NewOptimizedQueryAnalyzer(db, time.Duration(thresholdMs)*time.Millisecond)
	log.Println("Optimized query analyzer initialized")
}

func GetOptimizedQueryAnalyzer() *OptimizedQueryAnalyzer {
	return optimizedAnalyzer
}
