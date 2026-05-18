package database

import (
	"sync"
	"testing"
	"time"
)

func TestIndexAnalyzerCreation(t *testing.T) {
	analyzer := NewIndexAnalyzer(nil)
	
	if analyzer == nil {
		t.Fatal("NewIndexAnalyzer should not return nil")
	}
}

func TestMissingIndex(t *testing.T) {
	idx := &MissingIndex{
		TableName:  "users",
		IndexName: "idx_users_email",
		Columns:   []string{"email"},
		QueryCount: 1000,
		Priority:   "high",
	}
	
	if idx.TableName != "users" {
		t.Errorf("TableName = %q, want %q", idx.TableName, "users")
	}
	
	if idx.Priority != "high" {
		t.Errorf("Priority = %q, want %q", idx.Priority, "high")
	}
	
	if len(idx.Columns) != 1 {
		t.Errorf("Columns count = %d, want 1", len(idx.Columns))
	}
}

func TestFindMissingIndexes(t *testing.T) {
	analyzer := NewIndexAnalyzer(nil)
	indexes := analyzer.findMissingIndexes()
	
	if len(indexes) == 0 {
		t.Error("findMissingIndexes should return at least one index")
	}
}

func TestIndexUsageStats(t *testing.T) {
	stats := &IndexUsageStats{
		IndexName:     "idx_users_email",
		IndexSize:     "10MB",
		NumberOfScans: 1000,
		TuplesRead:    5000,
		TuplesFetched: 4800,
	}
	
	if stats.IndexName != "idx_users_email" {
		t.Errorf("IndexName = %q, want %q", stats.IndexName, "idx_users_email")
	}
	
	if stats.NumberOfScans != 1000 {
		t.Errorf("NumberOfScans = %d, want 1000", stats.NumberOfScans)
	}
}

func TestTableBloat(t *testing.T) {
	bloat := &TableBloat{
		TableName:  "verification_logs",
		TotalSize:  "1GB",
		TableSize:  "500MB",
		BloatRatio: 50.0,
	}
	
	if bloat.TableName != "verification_logs" {
		t.Errorf("TableName = %q, want %q", bloat.TableName, "verification_logs")
	}
	
	if bloat.BloatRatio != 50.0 {
		t.Errorf("BloatRatio = %f, want 50.0", bloat.BloatRatio)
	}
}

func TestIndexOptimizerCreation(t *testing.T) {
	optimizer := NewIndexOptimizer(nil)
	
	if optimizer == nil {
		t.Fatal("NewIndexOptimizer should not return nil")
	}
	
	if optimizer.analysisInterval != 24*time.Hour {
		t.Errorf("Analysis interval = %v, want %v", optimizer.analysisInterval, 24*time.Hour)
	}
	
	if !optimizer.enableAutoCreate {
		t.Error("Auto create should be enabled by default")
	}
	
	if !optimizer.enableAutoAnalyze {
		t.Error("Auto analyze should be enabled by default")
	}
}

func TestIndexOptimizerAnalyzeIndexes(t *testing.T) {
	optimizer := NewIndexOptimizer(nil)
	
	err := optimizer.AnalyzeIndexes()
	if err != nil {
		t.Errorf("AnalyzeIndexes should not return error: %v", err)
	}
	
	recommendations := optimizer.GetRecommendations()
	if len(recommendations) == 0 {
		t.Error("GetRecommendations should return at least one recommendation")
	}
}

func TestIndexOptimizerGenerateRecommendations(t *testing.T) {
	optimizer := NewIndexOptimizer(nil)
	recommendations := optimizer.generateRecommendations()
	
	if len(recommendations) == 0 {
		t.Error("generateRecommendations should return at least one recommendation")
	}
	
	for _, rec := range recommendations {
		if rec.TableName == "" {
			t.Error("Recommendation should have TableName")
		}
		if rec.IndexName == "" {
			t.Error("Recommendation should have IndexName")
		}
		if len(rec.Columns) == 0 {
			t.Error("Recommendation should have at least one column")
		}
	}
}

func TestIndexOptimizerGetRecommendations(t *testing.T) {
	optimizer := NewIndexOptimizer(nil)
	
	recommendations := optimizer.GetRecommendations()
	if len(recommendations) != 0 {
		t.Errorf("Initial recommendations should be empty, got %d", len(recommendations))
	}
	
	optimizer.recommendations = []*IndexRecommendation{
		{
			TableName: "users",
			IndexName: "idx_users_test",
			Columns:   []string{"test_column"},
		},
	}
	
	recommendations = optimizer.GetRecommendations()
	if len(recommendations) != 1 {
		t.Errorf("Expected 1 recommendation, got %d", len(recommendations))
	}
}

func TestIndexOptimizerSetAutoCreate(t *testing.T) {
	optimizer := NewIndexOptimizer(nil)
	
	optimizer.SetAutoCreate(false)
	if optimizer.enableAutoCreate {
		t.Error("Auto create should be disabled")
	}
	
	optimizer.SetAutoCreate(true)
	if !optimizer.enableAutoCreate {
		t.Error("Auto create should be enabled")
	}
}

func TestIndexOptimizerSetAutoAnalyze(t *testing.T) {
	optimizer := NewIndexOptimizer(nil)
	
	optimizer.SetAutoAnalyze(false)
	if optimizer.enableAutoAnalyze {
		t.Error("Auto analyze should be disabled")
	}
	
	optimizer.SetAutoAnalyze(true)
	if !optimizer.enableAutoAnalyze {
		t.Error("Auto analyze should be enabled")
	}
}

func TestQueryInfo(t *testing.T) {
	query := &QueryInfo{
		Name:           "FindUser",
		QueryPattern:   "SELECT * FROM users WHERE id = ?",
		ExecutionCount: 5000,
		AvgDuration:    10 * time.Millisecond,
	}
	
	if query.Name != "FindUser" {
		t.Errorf("Name = %q, want %q", query.Name, "FindUser")
	}
	
	if query.ExecutionCount != 5000 {
		t.Errorf("ExecutionCount = %d, want 5000", query.ExecutionCount)
	}
}

func TestGetFrequentQueries(t *testing.T) {
	analyzer := NewIndexAnalyzer(nil)
	queries := analyzer.getFrequentQueries()
	
	if len(queries) == 0 {
		t.Error("getFrequentQueries should return at least one query")
	}
}

func TestAnalyzeQuery(t *testing.T) {
	analyzer := NewIndexAnalyzer(nil)
	
	tests := []struct {
		name        string
		query       QueryInfo
		expectWarn  bool
	}{
		{
			name: "Slow query warning",
			query: QueryInfo{
				Name:           "SlowQuery",
				QueryPattern:   "SELECT * FROM large_table",
				ExecutionCount: 1000,
				AvgDuration:    15 * time.Millisecond,
			},
			expectWarn: true,
		},
		{
			name: "High frequency warning",
			query: QueryInfo{
				Name:           "HighFreqQuery",
				QueryPattern:   "SELECT * FROM users WHERE id = ?",
				ExecutionCount: 10000,
				AvgDuration:    5 * time.Millisecond,
			},
			expectWarn: true,
		},
		{
			name: "No warning",
			query: QueryInfo{
				Name:           "FastQuery",
				QueryPattern:   "SELECT id FROM users WHERE id = ?",
				ExecutionCount: 100,
				AvgDuration:    1 * time.Millisecond,
			},
			expectWarn: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := analyzer.analyzeQuery(tt.query)
			hasWarning := len(suggestions) > 0
			if hasWarning != tt.expectWarn {
				t.Errorf("analyzeQuery() = %v suggestions, want warning: %v", suggestions, tt.expectWarn)
			}
		})
	}
}

func TestInitAndGetIndexOptimizer(t *testing.T) {
	globalIndexOptimizer = nil
	globalIndexOptimizerOnce = sync.Once{}
	
	InitIndexOptimizer(nil)
	optimizer := GetIndexOptimizer()
	
	if optimizer == nil {
		t.Fatal("GetIndexOptimizer should not return nil after Init")
	}
	
	optimizer2 := GetIndexOptimizer()
	if optimizer != optimizer2 {
		t.Error("GetIndexOptimizer should return the same instance")
	}
}

func TestIndexRecommendation(t *testing.T) {
	rec := &IndexRecommendation{
		TableName:       "behavior_data",
		IndexName:       "idx_behavior_user_time",
		Columns:         []string{"user_id", "created_at"},
		IndexType:       "btree",
		Priority:        "high",
		EstimatedSize:   "50MB",
		QueryBenefits:   []string{"User behavior queries", "Faster analytics"},
		CreationSQL:     "CREATE INDEX CONCURRENTLY idx_behavior_user_time ON behavior_data (user_id, created_at)",
		EstimatedImpact: "High",
	}
	
	if rec.IndexType != "btree" {
		t.Errorf("IndexType = %q, want %q", rec.IndexType, "btree")
	}
	
	if len(rec.QueryBenefits) != 2 {
		t.Errorf("QueryBenefits count = %d, want 2", len(rec.QueryBenefits))
	}
}
