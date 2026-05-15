package database

import (
	"context"
	"strings"
	"time"
)

type QueryOptimizer struct {
	slowQueryThreshold time.Duration
}

func NewQueryOptimizer(slowQueryThreshold time.Duration) *QueryOptimizer {
	return &QueryOptimizer{
		slowQueryThreshold: slowQueryThreshold,
	}
}

type QueryAnalysis struct {
	Query          string
	ExecutionTime  time.Duration
	IsSlow         bool
	Suggestions    []string
	ExplainResults string
}

func (q *QueryOptimizer) Analyze(ctx context.Context, query string) (*QueryAnalysis, error) {
	start := time.Now()

	suggestions := q.generateSuggestions(query, "")

	executionTime := time.Since(start)

	return &QueryAnalysis{
		Query:          query,
		ExecutionTime:  executionTime,
		IsSlow:         executionTime > q.slowQueryThreshold,
		Suggestions:    suggestions,
		ExplainResults: "",
	}, nil
}

func (q *QueryOptimizer) generateSuggestions(query string, explainResults string) []string {
	var suggestions []string

	upperQuery := strings.ToUpper(query)

	if strings.Contains(upperQuery, "WHERE") && !strings.Contains(explainResults, "Using index") {
		suggestions = append(suggestions, "Consider adding index on WHERE clause columns")
	}

	if strings.Contains(upperQuery, "SELECT *") {
		suggestions = append(suggestions, "Avoid SELECT *, specify needed columns")
	}

	if strings.Contains(upperQuery, "JOIN") {
		suggestions = append(suggestions, "Ensure JOIN columns are indexed")
	}

	if strings.Contains(upperQuery, "ORDER BY") && !strings.Contains(explainResults, "Using index") {
		suggestions = append(suggestions, "Consider adding covering index for ORDER BY")
	}

	if strings.Contains(upperQuery, "LIKE '%") {
		suggestions = append(suggestions, "Avoid leading wildcards in LIKE patterns")
	}

	if strings.Contains(upperQuery, "NOT IN") || strings.Contains(upperQuery, "NOT EXISTS") {
		suggestions = append(suggestions, "Consider using JOIN or EXISTS instead of NOT IN/NOT EXISTS")
	}

	return suggestions
}

func (q *QueryOptimizer) OptimizeQuery(query string) string {
	var builder strings.Builder
	upperQuery := strings.ToUpper(query)

	if strings.Contains(upperQuery, "SELECT *") {
		builder.WriteString("Optimized: Avoid SELECT *, specify columns explicitly\n")
	}

	return builder.String()
}
