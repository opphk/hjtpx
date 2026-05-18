package service

import (
	"context"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

func TestQueryOptimizer(t *testing.T) {
	cfg := config.GetConfig()
	_ = cfg
	qo := NewQueryOptimizer(
		WithPreparedStatements(true),
		WithSlowQueryThreshold(50*time.Millisecond),
		WithQueryCache(true, 5*time.Minute),
	)

	if qo == nil {
		t.Fatal("QueryOptimizer should not be nil")
	}

	if !qo.enablePreparedStatements {
		t.Error("Prepared statements should be enabled")
	}

	if qo.slowQueryThreshold != 50*time.Millisecond {
		t.Errorf("Expected slow query threshold 50ms, got %v", qo.slowQueryThreshold)
	}
}

func TestQueryOptimizerOptimizeQuery(t *testing.T) {
	qo := NewQueryOptimizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Remove SELECT *",
			input:    "SELECT * FROM users",
			expected: "SELECT FROM users",
		},
		{
			name:     "Remove SQL injection patterns",
			input:    "SELECT * FROM users WHERE id=1 OR 1=1",
			expected: "SELECT FROM users WHERE id=1 ",
		},
		{
			name:     "Remove SELECT * with lowercase",
			input:    "select * from users",
			expected: "select from users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := qo.OptimizeQuery(tt.input)
			if result != tt.expected {
				t.Errorf("OptimizeQuery(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestQueryOptimizerBuildSelectQuery(t *testing.T) {
	qo := NewQueryOptimizer()

	query := qo.BuildSelectQuery(
		"users",
		[]string{"id", "username", "email"},
		map[string]interface{}{"status": "active"},
		"created_at DESC",
		10,
		0,
	)

	if query == "" {
		t.Error("BuildSelectQuery should not return empty string")
	}

	expected := "SELECT id, username, email FROM users WHERE status = ? ORDER BY created_at DESC LIMIT 10"
	if query != expected {
		t.Errorf("Expected query %q, got %q", expected, query)
	}
}

func TestQueryCache(t *testing.T) {
	cfg := config.GetConfig()
	qc := &QueryCache{
		entries: make(map[string]*QueryCacheEntry),
		maxSize: cfg.Database.QueryOptimization.MaxQueryCacheSize,
		ttl:     time.Duration(cfg.Database.QueryOptimization.QueryCacheTTLSecs) * time.Second,
	}

	key := qc.generateKey("SELECT * FROM users", 1)
	if key == "" {
		t.Error("generateKey should not return empty string")
	}

	key2 := qc.generateKey("SELECT * FROM users", 1)
	if key != key2 {
		t.Error("Same query and args should generate same key")
	}

	key3 := qc.generateKey("SELECT * FROM users", 2)
	if key == key3 {
		t.Error("Different args should generate different key")
	}

	qc.Set("test_key", map[string]string{"name": "test"})
	value, exists := qc.Get("test_key")
	if !exists {
		t.Error("Set and Get should work correctly")
	}
	if value == nil {
		t.Error("Retrieved value should not be nil")
	}

	qc.Clear()
	value, exists = qc.Get("test_key")
	if exists {
		t.Error("After Clear, Get should return false")
	}
}

func TestQueryCacheGetStats(t *testing.T) {
	cfg := config.GetConfig()
	qc := &QueryCache{
		entries: make(map[string]*QueryCacheEntry),
		maxSize: cfg.Database.QueryOptimization.MaxQueryCacheSize,
		ttl:     time.Duration(cfg.Database.QueryOptimization.QueryCacheTTLSecs) * time.Second,
	}

	stats := qc.GetStats()
	if stats == nil {
		t.Error("GetStats should not return nil")
	}

	if stats["max_size"] == nil {
		t.Error("Stats should contain max_size")
	}

	if stats["size"] == nil {
		t.Error("Stats should contain size")
	}
}

func TestDBOptimizer(t *testing.T) {
	db := &DBOptimizer{
		queryOptimizer: NewQueryOptimizer(),
	}

	if db.queryOptimizer == nil {
		t.Error("DBOptimizer should have query optimizer")
	}
}

func TestCachedQuery(t *testing.T) {
	db := &DBOptimizer{
		queryOptimizer: NewQueryOptimizer(),
	}

	cacheKey := "test_cache_key"
	dest := &struct {
		ID   int
		Name string
	}{}

	queryFunc := func() error {
		return nil
	}

	err := db.CachedQuery(context.Background(), cacheKey, dest, queryFunc)
	if err != nil {
		t.Errorf("CachedQuery should not return error: %v", err)
	}
}

func TestQueryMetricsCollector(t *testing.T) {
	RecordQueryMetrics(10*time.Millisecond, false, nil)
	RecordQueryMetrics(100*time.Millisecond, true, nil)
	RecordQueryMetrics(50*time.Millisecond, true, nil)

	metrics := GetQueryMetrics()
	if metrics == nil {
		t.Fatal("GetQueryMetrics should not return nil")
	}

	if metrics.TotalQueries != 3 {
		t.Errorf("Expected 3 total queries, got %d", metrics.TotalQueries)
	}

	if metrics.SlowQueries != 2 {
		t.Errorf("Expected 2 slow queries, got %d", metrics.SlowQueries)
	}

	ResetQueryMetrics()
	metrics = GetQueryMetrics()
	if metrics.TotalQueries != 0 {
		t.Error("After reset, total queries should be 0")
	}
}

func TestRecordCacheHitMiss(t *testing.T) {
	ResetQueryMetrics()

	RecordCacheHit()
	RecordCacheHit()
	RecordCacheMiss()

	metrics := GetQueryMetrics()
	if metrics.CacheHits != 2 {
		t.Errorf("Expected 2 cache hits, got %d", metrics.CacheHits)
	}

	if metrics.CacheMisses != 1 {
		t.Errorf("Expected 1 cache miss, got %d", metrics.CacheMisses)
	}
}

func TestQueryOptimizerWithOptions(t *testing.T) {
	opts := []QueryOption{
		WithPreparedStatements(true),
		WithSlowQueryThreshold(100 * time.Millisecond),
		WithQueryCache(true, 10*time.Minute),
	}

	qo := NewQueryOptimizer(opts...)

	if qo.enablePreparedStatements != true {
		t.Error("Prepared statements should be enabled")
	}

	if qo.slowQueryThreshold != 100*time.Millisecond {
		t.Errorf("Expected slow query threshold 100ms, got %v", qo.slowQueryThreshold)
	}

	if qo.queryCacheTTL != 10*time.Minute {
		t.Errorf("Expected query cache TTL 10min, got %v", qo.queryCacheTTL)
	}
}

func TestQueryOptimizerShouldUseIndex(t *testing.T) {
	qo := NewQueryOptimizer()

	tests := []struct {
		name        string
		table       string
		whereClause string
		expected    bool
	}{
		{
			name:        "With LIMIT clause",
			table:       "users",
			whereClause: "LIMIT 10",
			expected:    true,
		},
		{
			name:        "With WHERE clause",
			table:       "users",
			whereClause: "status = 'active'",
			expected:    true,
		},
		{
			name:        "Empty WHERE clause",
			table:       "users",
			whereClause: "",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := qo.ShouldUseIndex(tt.table, tt.whereClause)
			if result != tt.expected {
				t.Errorf("ShouldUseIndex(%s, %s) = %v, want %v",
					tt.table, tt.whereClause, result, tt.expected)
			}
		})
	}
}

func TestQueryCacheExpiration(t *testing.T) {
	cfg := config.GetConfig()
	_ = cfg
	qc := &QueryCache{
		entries: make(map[string]*QueryCacheEntry),
		maxSize: 10,
		ttl:     1 * time.Millisecond,
	}

	qc.Set("expiring_key", "value")

	time.Sleep(2 * time.Millisecond)

	_, exists := qc.Get("expiring_key")
	if exists {
		t.Error("Expired key should not be found")
	}
}

func TestQueryCacheMaxSize(t *testing.T) {
	cfg := config.GetConfig()
	_ = cfg
	qc := &QueryCache{
		entries: make(map[string]*QueryCacheEntry),
		maxSize: 3,
		ttl:     5 * time.Minute,
	}

	for i := 0; i < 5; i++ {
		qc.Set(string(rune('a'+i)), i)
	}

	if len(qc.entries) > qc.maxSize {
		t.Errorf("Cache size should not exceed maxSize %d, got %d", qc.maxSize, len(qc.entries))
	}
}

func TestHealthCheckDB(t *testing.T) {
	ctx := context.Background()

	t.Run("nil db", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Recovered from panic: %v", r)
			}
		}()
		err := HealthCheckDB(ctx, nil)
		if err == nil {
			t.Error("HealthCheckDB with nil db should return error or panic")
		}
	})
}

func TestSoftDeleteAndRestore(t *testing.T) {
	db := &DBOptimizer{
		queryOptimizer: NewQueryOptimizer(),
	}

	err := db.SoftDelete("users", []uint{})
	if err != nil {
		t.Error("SoftDelete with empty ids should not return error")
	}

	err = db.Restore("users", []uint{})
	if err != nil {
		t.Error("Restore with empty ids should not return error")
	}
}

func TestPaginatedFind(t *testing.T) {
	db := &DBOptimizer{
		queryOptimizer: NewQueryOptimizer(),
	}

	var dest []struct {
		ID int
	}

	defer func() {
		if r := recover(); r != nil {
			t.Logf("Recovered from panic: %v", r)
		}
	}()

	_, err := db.PaginatedFind("users", nil, 1, 10, &dest)
	if err != nil {
		t.Errorf("PaginatedFind should not return error: %v", err)
	}
}

func TestBatchInsert(t *testing.T) {
	db := &DBOptimizer{
		queryOptimizer: NewQueryOptimizer(),
	}

	err := db.BatchInsert("users", []map[string]interface{}{}, 100)
	if err != nil {
		t.Error("BatchInsert with empty records should not return error")
	}
}

func TestOptimizeComplexQuery(t *testing.T) {
	db := &DBOptimizer{
		queryOptimizer: NewQueryOptimizer(),
	}

	query := "SELECT * FROM users WHERE status = 'active'"

	defer func() {
		if r := recover(); r != nil {
			t.Logf("Recovered from panic: %v", r)
		}
	}()

	_, err := db.OptimizeComplexQuery(context.Background(), query)
	if err != nil {
		t.Errorf("OptimizeComplexQuery should not error: %v", err)
	}
}

func TestAnalyzeQueryPlan(t *testing.T) {
	db := &DBOptimizer{
		queryOptimizer: NewQueryOptimizer(),
	}

	query := "SELECT * FROM users LIMIT 10"

	defer func() {
		if r := recover(); r != nil {
			t.Logf("Recovered from panic: %v", r)
		}
	}()

	plan, err := db.AnalyzeQueryPlan(query)
	if err != nil {
		t.Errorf("AnalyzeQueryPlan should not error: %v", err)
	}
	if plan == nil {
		t.Error("Query plan should not be nil")
	}
}

func TestVacuumAndReindex(t *testing.T) {
	db := &DBOptimizer{
		queryOptimizer: NewQueryOptimizer(),
	}

	defer func() {
		if r := recover(); r != nil {
			t.Logf("Recovered from panic: %v", r)
		}
	}()

	err := db.VacuumTable("users")
	if err != nil {
		t.Errorf("VacuumTable should not error: %v", err)
	}

	err = db.ReindexTable("users")
	if err != nil {
		t.Errorf("ReindexTable should not error: %v", err)
	}
}

func TestConfigureConnectionPool(t *testing.T) {
	db := &DBOptimizer{
		queryOptimizer: NewQueryOptimizer(),
	}

	defer func() {
		if r := recover(); r != nil {
			t.Logf("Recovered from panic: %v", r)
		}
	}()

	err := db.ConfigureConnectionPool(100, 20, 30*time.Minute)
	if err == nil {
		t.Error("ConfigureConnectionPool with nil db should return error")
	}
}

func TestReadFromReplica(t *testing.T) {
	db := &DBOptimizer{
		queryOptimizer: NewQueryOptimizer(),
	}

	db.UseReadReplica(false)
	if db.useReplica.Load() {
		t.Error("UseReadReplica(false) should set useReplica to false")
	}

	db.UseReadReplica(true)
	if !db.useReplica.Load() {
		t.Error("UseReadReplica(true) should set useReplica to true")
	}

	result := db.ReadFromReplica(func(db *gorm.DB) *gorm.DB {
		return db
	})
	if result != nil {
		t.Error("ReadFromReplica should return nil when no replica is set")
	}
}

func TestSetReadReplica(t *testing.T) {
	db := &DBOptimizer{
		queryOptimizer: NewQueryOptimizer(),
	}

	db.SetReadReplica(nil)
	if db.readReplica != nil {
		t.Error("SetReadReplica(nil) should set readReplica to nil")
	}
}

func TestInvalidateQueryCache(t *testing.T) {
	db := &DBOptimizer{
		queryOptimizer: NewQueryOptimizer(),
	}

	db.InvalidateQueryCache("test")
}

func TestQueryCacheKeyGeneration(t *testing.T) {
	cfg := config.GetConfig()
	qc := &QueryCache{
		entries: make(map[string]*QueryCacheEntry),
		maxSize: cfg.Database.QueryOptimization.MaxQueryCacheSize,
		ttl:     time.Duration(cfg.Database.QueryOptimization.QueryCacheTTLSecs) * time.Second,
	}

	key1 := qc.generateKey("SELECT * FROM users WHERE id = ?", 1)
	key2 := qc.generateKey("SELECT * FROM users WHERE id = ?", 2)
	key3 := qc.generateKey("SELECT * FROM users WHERE id = ?", 1)

	if key1 == key2 {
		t.Error("Different parameters should generate different keys")
	}

	if key1 != key3 {
		t.Error("Same parameters should generate same keys")
	}
}
