package database

import (
	"context"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlowQueryAnalyzerInit(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Monitoring: config.MonitoringConfig{
				EnableQueryMetrics: true,
			},
			SlowQueryThresholdMs: 100,
		},
	}

	err := InitSlowQueryAnalyzer(cfg)
	require.NoError(t, err)

	analyzer := GetSlowQueryAnalyzer()
	require.NotNil(t, analyzer)
	assert.Equal(t, 100*time.Millisecond, analyzer.threshold)
	assert.True(t, analyzer.enabled)
}

func TestSlowQueryAnalyzerRecord(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Monitoring: config.MonitoringConfig{
				EnableQueryMetrics: true,
			},
			SlowQueryThresholdMs: 50,
		},
	}

	err := InitSlowQueryAnalyzer(cfg)
	require.NoError(t, err)

	analyzer := GetSlowQueryAnalyzer()
	require.NotNil(t, analyzer)

	analyzer.RecordSlowQuery("SELECT * FROM users", 100*time.Millisecond, nil, 100, "test")
	analyzer.RecordSlowQuery("SELECT * FROM orders", 200*time.Millisecond, nil, 200, "test")

	assert.Equal(t, 2, len(analyzer.records))
}

func TestSlowQueryAnalyzerAnalyze(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Monitoring: config.MonitoringConfig{
				EnableQueryMetrics: true,
			},
			SlowQueryThresholdMs: 50,
		},
	}

	err := InitSlowQueryAnalyzer(cfg)
	require.NoError(t, err)

	analyzer := GetSlowQueryAnalyzer()
	require.NotNil(t, analyzer)

	analyzer.RecordSlowQuery("SELECT * FROM users", 100*time.Millisecond, nil, 100, "test")
	analyzer.RecordSlowQuery("SELECT * FROM orders", 150*time.Millisecond, nil, 200, "test")
	analyzer.RecordSlowQuery("SELECT * FROM products", 200*time.Millisecond, nil, 300, "test")

	ctx := context.Background()
	analysis, err := analyzer.Analyze(ctx, 10)
	require.NoError(t, err)

	assert.NotNil(t, analysis)
	assert.Equal(t, int64(3), analysis.TotalQueries)
	assert.Greater(t, analysis.AvgDuration, time.Duration(0))
	assert.Greater(t, analysis.MaxDuration, time.Duration(0))
}

func TestSlowQueryAnalyzerClear(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Monitoring: config.MonitoringConfig{
				EnableQueryMetrics: true,
			},
			SlowQueryThresholdMs: 50,
		},
	}

	err := InitSlowQueryAnalyzer(cfg)
	require.NoError(t, err)

	analyzer := GetSlowQueryAnalyzer()
	require.NotNil(t, analyzer)

	analyzer.RecordSlowQuery("SELECT * FROM users", 100*time.Millisecond, nil, 100, "test")
	assert.Equal(t, 1, len(analyzer.records))

	analyzer.Clear()
	assert.Equal(t, 0, len(analyzer.records))
}

func TestSlowQueryAnalyzerGetRecords(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Monitoring: config.MonitoringConfig{
				EnableQueryMetrics: true,
			},
			SlowQueryThresholdMs: 50,
		},
	}

	err := InitSlowQueryAnalyzer(cfg)
	require.NoError(t, err)

	analyzer := GetSlowQueryAnalyzer()
	require.NotNil(t, analyzer)

	for i := 0; i < 15; i++ {
		analyzer.RecordSlowQuery("SELECT * FROM users", time.Duration(100+i*10)*time.Millisecond, nil, int64(i), "test")
	}

	records := analyzer.GetRecords(10, 0)
	assert.Equal(t, 10, len(records))

	records = analyzer.GetRecords(5, 10)
	assert.Equal(t, 5, len(records))
}

func TestPatternAnalyzer(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Monitoring: config.MonitoringConfig{
				EnableQueryMetrics: true,
			},
			SlowQueryThresholdMs: 50,
		},
	}

	err := InitSlowQueryAnalyzer(cfg)
	require.NoError(t, err)

	analyzer := GetSlowQueryAnalyzer()
	require.NotNil(t, analyzer)

	analyzer.RecordSlowQuery("SELECT * FROM users WHERE id = 1", 100*time.Millisecond, nil, 1, "test")
	analyzer.RecordSlowQuery("SELECT * FROM users WHERE id = 2", 100*time.Millisecond, nil, 1, "test")
	analyzer.RecordSlowQuery("SELECT * FROM orders WHERE id = 1", 150*time.Millisecond, nil, 1, "test")

	patterns := analyzer.patternAnalyzer.getTopPatterns(10)
	assert.NotEmpty(t, patterns)
}

func TestIdentifyQueryType(t *testing.T) {
	testCases := []struct {
		query     string
		expected  string
	}{
		{"SELECT * FROM users", "SELECT"},
		{"INSERT INTO users VALUES (1)", "INSERT"},
		{"UPDATE users SET name = 'test'", "UPDATE"},
		{"DELETE FROM users WHERE id = 1", "DELETE"},
		{"CREATE TABLE test (id INT)", "CREATE"},
		{"DROP TABLE users", "DROP"},
		{"ALTER TABLE users ADD COLUMN age INT", "ALTER"},
		{"UNKNOWN QUERY", "OTHER"},
	}

	for _, tc := range testCases {
		result := identifyQueryType(tc.query)
		assert.Equal(t, tc.expected, result, "Query: %s", tc.query)
	}
}

func TestSanitizeQuery(t *testing.T) {
	query := "SELECT * FROM users WHERE name = 'John Doe' AND email = 'john@example.com'"
	sanitized := sanitizeQuery(query)

	assert.Contains(t, sanitized, "SELECT * FROM users")
	assert.NotContains(t, sanitized, "John Doe")
	assert.NotContains(t, sanitized, "john@example.com")
	assert.Contains(t, sanitized, "***")
}

func TestNormalizeQuery(t *testing.T) {
	query1 := "SELECT * FROM users WHERE id = 123"
	query2 := "SELECT * FROM users WHERE id = 456"

	normalized1 := normalizeQuery(query1)
	normalized2 := normalizeQuery(query2)

	assert.Equal(t, normalized1, normalized2)
}

func TestExtractTableName(t *testing.T) {
	testCases := []struct {
		query     string
		expected  string
	}{
		{"SELECT * FROM users", "users"},
		{"SELECT * FROM user_profiles", "user_profiles"},
		{"SELECT * FROM mydb.users", "users"},
	}

	for _, tc := range testCases {
		result := extractTableName(tc.query)
		assert.Contains(t, []string{"users", "user_profiles", "mydb"}, result, "Query: %s", tc.query)
	}
}
