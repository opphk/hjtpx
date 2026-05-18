package database

import (
	"testing"
	"time"
)

func TestQueryAnalyzerCreation(t *testing.T) {
	analyzer := NewQueryAnalyzer(nil, 50)
	
	if analyzer == nil {
		t.Fatal("NewQueryAnalyzer should not return nil")
	}
	
	if analyzer.slowQueryThreshold != 50*time.Millisecond {
		t.Errorf("Slow query threshold = %v, want %v", analyzer.slowQueryThreshold, 50*time.Millisecond)
	}
	
	if analyzer.maxQueriesToAnalyze != 50 {
		t.Errorf("Max queries to analyze = %d, want 50", analyzer.maxQueriesToAnalyze)
	}
}

func TestSlowQueryResult(t *testing.T) {
	result := &SlowQueryResult{
		QueryID:      1,
		QueryPreview: "SELECT * FROM users WHERE status = 'active'",
		Calls:        1000,
		TotalTimeMs:  50000.0,
		MeanTimeMs:   50.0,
		MaxTimeMs:    200.0,
		MinTimeMs:    10.0,
		TotalRows:    10000,
		Suggestions:  []string{"Add index", "Optimize query"},
	}
	
	if result.QueryID != 1 {
		t.Errorf("QueryID = %d, want 1", result.QueryID)
	}
	
	if result.Calls != 1000 {
		t.Errorf("Calls = %d, want 1000", result.Calls)
	}
	
	if len(result.Suggestions) != 2 {
		t.Errorf("Suggestions count = %d, want 2", len(result.Suggestions))
	}
}

func TestGenerateSuggestions(t *testing.T) {
	analyzer := NewQueryAnalyzer(nil, 50)
	
	tests := []struct {
		name      string
		query     string
		meanTime  float64
		expectLen int
	}{
		{
			name:      "High slow query",
			query:     "SELECT * FROM large_table WHERE created_at < '2024-01-01'",
			meanTime:  150.0,
			expectLen: 2,
		},
		{
			name:      "Medium slow query",
			query:     "SELECT id, name FROM users WHERE status = 'active'",
			meanTime:  70.0,
			expectLen: 1,
		},
		{
			name:      "Fast query",
			query:     "SELECT id FROM users WHERE id = 1",
			meanTime:  5.0,
			expectLen: 0,
		},
		{
			name:      "Query with SELECT *",
			query:     "SELECT * FROM users WHERE status = 'active'",
			meanTime:  60.0,
			expectLen: 2,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := analyzer.GenerateSuggestions(tt.query, tt.meanTime)
			if len(suggestions) != tt.expectLen {
				t.Errorf("GenerateSuggestions() returned %d suggestions, want %d: %v", 
					len(suggestions), tt.expectLen, suggestions)
			}
		})
	}
}

func TestContainsPatterns(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		function func(string) bool
		expected bool
	}{
		{
			name:     "Contains SELECT *",
			query:    "SELECT * FROM users",
			function: containsSelectStar,
			expected: true,
		},
		{
			name:     "Does not contain SELECT *",
			query:    "SELECT id FROM users",
			function: containsSelectStar,
			expected: false,
		},
		{
			name:     "Contains NOT IN",
			query:    "WHERE id NOT IN (1, 2, 3)",
			function: containsNotIn,
			expected: true,
		},
		{
			name:     "Does not contain NOT IN",
			query:    "WHERE id IN (1, 2, 3)",
			function: containsNotIn,
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function(tt.query)
			if result != tt.expected {
				t.Errorf("%s(%q) = %v, want %v", tt.name, tt.query, result, tt.expected)
			}
		})
	}
}

func TestQueryPlanAnalysis(t *testing.T) {
	analysis := &QueryPlanAnalysis{
		Query:     "SELECT * FROM users WHERE id = 1",
		Plan:      "Index Scan using idx_users_id on users",
		ScanTypes: []string{"Index Scan"},
		CostInfo:  map[string]string{"cost_range": "cost=0.29..0.51"},
		Warnings:  []string{},
		Score:     100,
	}
	
	if analysis.Query != "SELECT * FROM users WHERE id = 1" {
		t.Errorf("Query = %q, want %q", analysis.Query, "SELECT * FROM users WHERE id = 1")
	}
	
	if len(analysis.ScanTypes) != 1 {
		t.Errorf("ScanTypes count = %d, want 1", len(analysis.ScanTypes))
	}
	
	if analysis.Score != 100 {
		t.Errorf("Score = %d, want 100", analysis.Score)
	}
}

func TestExtractScanTypes(t *testing.T) {
	analyzer := NewQueryAnalyzer(nil, 50)
	
	tests := []struct {
		name     string
		plan     string
		expected []string
	}{
		{
			name:     "Index Scan",
			plan:     "Index Scan using idx_users_id on users",
			expected: []string{"Index Scan"},
		},
		{
			name:     "Seq Scan",
			plan:     "Seq Scan on users",
			expected: []string{"Seq Scan"},
		},
		{
			name:     "Multiple scans",
			plan:     "Bitmap Heap Scan on users\nIndex Scan using idx_users_id",
			expected: []string{"Index Scan", "Bitmap Heap Scan"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanTypes := analyzer.extractScanTypes(tt.plan)
			if len(scanTypes) != len(tt.expected) {
				t.Errorf("extractScanTypes() returned %d types, want %d", 
					len(scanTypes), len(tt.expected))
			}
		})
	}
}

func TestAnalyzePlanWarnings(t *testing.T) {
	analyzer := NewQueryAnalyzer(nil, 50)
	
	tests := []struct {
		name     string
		plan     string
		wantWarn bool
	}{
		{
			name:     "Seq Scan warning",
			plan:     "Seq Scan on large_table",
			wantWarn: true,
		},
		{
			name:     "Index Scan no warning",
			plan:     "Index Scan using idx_users_id",
			wantWarn: false,
		},
		{
			name:     "Sort without index",
			plan:     "Sort  (cost=100.00..200.00 rows=10000 width=100)",
			wantWarn: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := analyzer.analyzePlanWarnings(tt.plan)
			hasWarning := len(warnings) > 0
			if hasWarning != tt.wantWarn {
				t.Errorf("analyzePlanWarnings() = %v warnings, want warning: %v", warnings, tt.wantWarn)
			}
		})
	}
}

func TestCalculatePerformanceScore(t *testing.T) {
	analyzer := NewQueryAnalyzer(nil, 50)
	
	tests := []struct {
		name     string
		plan     string
		expected int
	}{
		{
			name:     "Perfect score",
			plan:     "Index Scan using idx_users_id",
			expected: 100,
		},
		{
			name:     "Seq Scan penalty",
			plan:     "Seq Scan on users",
			expected: 70,
		},
		{
			name:     "Bitmap Heap Scan penalty",
			plan:     "Bitmap Heap Scan on users",
			expected: 90,
		},
		{
			name:     "Multiple penalties",
			plan:     "Seq Scan on users\nHash Join",
			expected: 65,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzer.calculatePerformanceScore(tt.plan)
			if score != tt.expected {
				t.Errorf("calculatePerformanceScore() = %d, want %d", score, tt.expected)
			}
		})
	}
}

func TestFullAnalysisReport(t *testing.T) {
	report := &FullAnalysisReport{
		Timestamp:      time.Now(),
		SlowQueries:    []SlowQueryResult{},
		SlowQueryCount: 0,
		UnusedIndexes:  []string{},
		IndexStats:     []IndexUsageStats{},
		Recommendations: []string{
			"Enable pg_stat_statements",
			"Consider partitioning",
		},
		OverallScore: 100,
	}
	
	if report.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	
	if len(report.Recommendations) != 2 {
		t.Errorf("Recommendations count = %d, want 2", len(report.Recommendations))
	}
}

func TestGenerateRecommendations(t *testing.T) {
	analyzer := NewQueryAnalyzer(nil, 50)
	
	tests := []struct {
		name               string
		slowQueryCount     int
		unusedIndexesCount int
		minRecCount        int
	}{
		{
			name:               "Many slow queries",
			slowQueryCount:     15,
			unusedIndexesCount: 3,
			minRecCount:        3,
		},
		{
			name:               "Some slow queries",
			slowQueryCount:     7,
			unusedIndexesCount: 0,
			minRecCount:        2,
		},
		{
			name:               "No issues",
			slowQueryCount:     2,
			unusedIndexesCount: 0,
			minRecCount:        3,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &FullAnalysisReport{
				SlowQueryCount:  tt.slowQueryCount,
				UnusedIndexes:  make([]string, tt.unusedIndexesCount),
			}
			recs := analyzer.generateRecommendations(report)
			if len(recs) < tt.minRecCount {
				t.Errorf("generateRecommendations() returned %d recommendations, want at least %d", 
					len(recs), tt.minRecCount)
			}
		})
	}
}

func TestGenerateIndexDDL(t *testing.T) {
	analyzer := NewQueryAnalyzer(nil, 50)
	
	rec := &IndexRecommendation{
		TableName: "users",
		IndexName: "idx_users_email",
		Columns:   []string{"email"},
	}
	
	ddl := analyzer.GenerateIndexDDL(*rec)
	expected := "CREATE INDEX CONCURRENTLY idx_users_email ON users (email);"
	
	if ddl != expected {
		t.Errorf("GenerateIndexDDL() = %q, want %q", ddl, expected)
	}
}
