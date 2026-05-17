package database

import (
	"regexp"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLOptimizerInit(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			QueryOptimization: config.QueryOptimizationConfig{
				EnablePreparedStatements: true,
			},
		},
	}

	err := InitSQLOptimizer(nil, cfg)
	require.NoError(t, err)

	optimizer := GetSQLOptimizer()
	require.NotNil(t, optimizer)
	assert.True(t, optimizer.enabled)
}

func TestSQLOptimizerEstimateImprovement(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			QueryOptimization: config.QueryOptimizationConfig{
				EnablePreparedStatements: true,
			},
		},
	}

	err := InitSQLOptimizer(nil, cfg)
	require.NoError(t, err)

	optimizer := GetSQLOptimizer()
	require.NotNil(t, optimizer)

	changes := []QueryChange{
		{Type: "index_added", Description: "Added index"},
		{Type: "query_rewritten", Description: "Rewrote query"},
	}

	improvement := optimizer.estimateImprovement(changes)
	assert.Greater(t, improvement, 0.0)
}

func TestQueryRewriterRewrite(t *testing.T) {
	rewriter := &QueryRewriter{
		rules: initializeRewriteRules(),
	}

	testCases := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "RemoveSELECTStar",
			query:    "SELECT * FROM users",
			expected: "SELECT column_list FROM users",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			optimized, changes := rewriter.rewrite(tc.query)
			assert.NotEmpty(t, optimized)
			assert.NotNil(t, changes)
		})
	}
}

func TestInitializeRewriteRules(t *testing.T) {
	rules := initializeRewriteRules()
	assert.NotEmpty(t, rules)

	for _, rule := range rules {
		assert.NotEmpty(t, rule.Name)
		assert.NotEmpty(t, rule.Pattern)
		assert.NotEmpty(t, rule.Description)
	}
}

func TestSQLOptimizerGetStatistics(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			QueryOptimization: config.QueryOptimizationConfig{
				EnablePreparedStatements: true,
			},
		},
	}

	err := InitSQLOptimizer(nil, cfg)
	require.NoError(t, err)

	optimizer := GetSQLOptimizer()
	require.NotNil(t, optimizer)

	stats := optimizer.GetStatistics()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "pending_indexes")
	assert.Contains(t, stats, "analyzed_tables")
}

func TestIndexRecommenderGetRecommendations(t *testing.T) {
	recommender := &IndexRecommender{
		indexes: []IndexRecommendation{
			{
				TableName: "users",
				Columns:  []string{"email"},
				IndexType: "B-tree",
				Priority: 10,
			},
		},
	}

	recs := recommender.getRecommendations("SELECT * FROM users WHERE email = ?")
	assert.NotEmpty(t, recs)
	assert.Equal(t, "users", recs[0].TableName)
}

func TestExtractWhereColumns(t *testing.T) {
	testCases := []struct {
		query    string
		expected int
	}{
		{"SELECT * FROM users WHERE id = 1", 1},
		{"SELECT * FROM users WHERE id = 1 AND name = 'test'", 1},
		{"SELECT * FROM users WHERE users.id = 1", 1},
	}

	for _, tc := range testCases {
		columns := extractWhereColumns(tc.query)
		assert.GreaterOrEqual(t, len(columns), tc.expected, "Query: %s", tc.query)
	}
}

func TestPatternAnalyzerAnalyzeQueryPattern(t *testing.T) {
	analyzer := &PatternAnalyzer{
		patterns:     make(map[string]*QueryPattern),
		patternRegex: regexp.MustCompile(`(?i)(SELECT|INSERT|UPDATE|DELETE|FROM|WHERE|JOIN|ORDER BY|GROUP BY|HAVING|LIMIT|OFFSET)`),
	}

	analyzer.analyzeQueryPattern("SELECT * FROM users WHERE id = 1", 100*time.Millisecond)
	analyzer.analyzeQueryPattern("SELECT * FROM users WHERE id = 2", 100*time.Millisecond)
	analyzer.analyzeQueryPattern("SELECT * FROM users WHERE id = 3", 100*time.Millisecond)

	assert.NotEmpty(t, analyzer.patterns)
}

func TestPatternAnalyzerGenerateSuggestions(t *testing.T) {
	analyzer := &PatternAnalyzer{}

	testCases := []struct {
		query       string
		suggestions int
	}{
		{"SELECT * FROM users", 1},
		{"SELECT * FROM users WHERE name LIKE '%test'", 1},
		{"SELECT * FROM users ORDER BY created_at", 1},
	}

	for _, tc := range testCases {
		suggestions := analyzer.generateSuggestions(tc.query)
		assert.GreaterOrEqual(t, len(suggestions), tc.suggestions, "Query: %s", tc.query)
	}
}

func TestPatternAnalyzerGetTopPatterns(t *testing.T) {
	analyzer := &PatternAnalyzer{
		patterns: make(map[string]*QueryPattern),
	}

	for i := 0; i < 15; i++ {
		pattern := &QueryPattern{
			Pattern:       "test_pattern_" + string(rune(i)),
			Occurrences:  i + 1,
			TotalDuration: time.Duration(i+1) * time.Millisecond,
		}
		analyzer.patterns[pattern.Pattern] = pattern
	}

	topPatterns := analyzer.getTopPatterns(10)
	assert.Equal(t, 10, len(topPatterns))
}

func TestPatternAnalyzerClear(t *testing.T) {
	analyzer := &PatternAnalyzer{
		patterns: map[string]*QueryPattern{
			"test": {
				Pattern: "test",
			},
		},
	}

	analyzer.clear()
	assert.Empty(t, analyzer.patterns)
}
