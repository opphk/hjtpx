package db

import (
	"context"
	"testing"
	"time"
)

func TestNewDBOptimizer(t *testing.T) {
	config := &OptimizerConfig{
		EnableQueryCache:   true,
		QueryCacheTTL:      5 * time.Minute,
		MaxQueryCacheSize:  100,
		BatchSize:          50,
		MaxQueryTimeout:    30 * time.Second,
		SlowQueryThreshold: 100 * time.Millisecond,
	}

	optimizer := NewDBOptimizer(nil, config)

	if optimizer == nil {
		t.Fatal("NewDBOptimizer should not return nil")
	}

	if optimizer.queryCache == nil {
		t.Error("QueryCache should be initialized")
	}

	if optimizer.indexManager == nil {
		t.Error("IndexManager should be initialized")
	}

	if optimizer.batchProcessor == nil {
		t.Error("BatchProcessor should be initialized")
	}

	if optimizer.ormOptimizer == nil {
		t.Error("ORMQueryOptimizer should be initialized")
	}

	if optimizer.metricsCollector == nil {
		t.Error("MetricsCollector should be initialized")
	}
}

func TestNewDBOptimizerWithNilConfig(t *testing.T) {
	optimizer := NewDBOptimizer(nil, nil)

	if optimizer == nil {
		t.Fatal("NewDBOptimizer should not return nil with nil config")
	}

	if optimizer.queryCache == nil {
		t.Error("QueryCache should use default config")
	}
}

func TestDBOptimizerReplica(t *testing.T) {
	optimizer := NewDBOptimizer(nil, nil)

	optimizer.EnableReplica()
	if !optimizer.useReplica.Load() {
		t.Error("EnableReplica should set useReplica to true")
	}

	optimizer.DisableReplica()
	if optimizer.useReplica.Load() {
		t.Error("DisableReplica should set useReplica to false")
	}
}

func TestDBOptimizerGetDB(t *testing.T) {
	optimizer := NewDBOptimizer(nil, nil)

	db := optimizer.GetDB()
	if db != nil {
		t.Error("GetDB with nil db should return nil")
	}

	optimizer.EnableReplica()
	db = optimizer.GetDB()
	if db != nil {
		t.Error("GetDB with nil replica should return nil")
	}
}

func TestRepositoryQueryCache(t *testing.T) {
	cache := NewRepositoryQueryCache(100, 5*time.Minute)

	if cache == nil {
		t.Fatal("NewRepositoryQueryCache should not return nil")
	}

	if cache.maxSize != 100 {
		t.Errorf("Expected maxSize 100, got %d", cache.maxSize)
	}
}

func TestRepositoryQueryCacheGenerateKey(t *testing.T) {
	cache := NewRepositoryQueryCache(100, 5*time.Minute)

	key1 := cache.generateKey("SELECT * FROM users", 1)
	key2 := cache.generateKey("SELECT * FROM users", 1)
	key3 := cache.generateKey("SELECT * FROM users", 2)

	if key1 != key2 {
		t.Error("Same query and args should generate same key")
	}

	if key1 == key3 {
		t.Error("Different args should generate different keys")
	}

	if key1 == "" {
		t.Error("Generated key should not be empty")
	}
}

func TestRepositoryQueryCacheSetAndGet(t *testing.T) {
	cache := NewRepositoryQueryCache(100, 5*time.Minute)

	cache.Set("test_key", "test_value")
	value, exists := cache.Get("test_key")

	if !exists {
		t.Error("Set and Get should work correctly")
	}

	if value == nil {
		t.Error("Retrieved value should not be nil")
	}
}

func TestRepositoryQueryCacheExpiration(t *testing.T) {
	cache := NewRepositoryQueryCache(100, 1*time.Millisecond)

	cache.Set("expiring_key", "value")

	time.Sleep(5 * time.Millisecond)

	_, exists := cache.Get("expiring_key")
	if exists {
		t.Error("Expired key should not be found")
	}
}

func TestRepositoryQueryCacheLRUEviction(t *testing.T) {
	cache := NewRepositoryQueryCache(3, 5*time.Minute)

	cache.Set("key1", "value1")
	time.Sleep(1 * time.Millisecond)
	cache.Set("key2", "value2")
	time.Sleep(1 * time.Millisecond)
	cache.Set("key3", "value3")
	time.Sleep(1 * time.Millisecond)

	cache.Set("key4", "value4")

	_, exists := cache.Get("key1")
	if exists {
		t.Error("key1 should have been evicted due to LRU")
	}

	_, exists = cache.Get("key2")
	if !exists {
		t.Error("key2 should still exist")
	}
}

func TestRepositoryQueryCacheGetOrSet(t *testing.T) {
	cache := NewRepositoryQueryCache(100, 5*time.Minute)

	dest := ""
	called := false

	queryFunc := func() error {
		called = true
		dest = "query_result"
		return nil
	}

	err := cache.GetOrSet("cache_key", &dest, queryFunc)
	if err != nil {
		t.Errorf("GetOrSet should not return error: %v", err)
	}

	if !called {
		t.Error("queryFunc should have been called")
	}

	if dest != "query_result" {
		t.Errorf("Expected dest to be 'query_result', got '%s'", dest)
	}

	called = false
	err = cache.GetOrSet("cache_key", &dest, queryFunc)
	if err != nil {
		t.Errorf("GetOrSet should not return error: %v", err)
	}

	if called {
		t.Error("queryFunc should not have been called on cache hit")
	}
}

func TestRepositoryQueryCacheInvalidate(t *testing.T) {
	cache := NewRepositoryQueryCache(100, 5*time.Minute)

	cache.Set("table:users:1", "value1")
	cache.Set("table:users:2", "value2")
	cache.Set("table:apps:1", "value3")

	cache.Invalidate("table:users")

	_, exists := cache.Get("table:users:1")
	if exists {
		t.Error("Keys matching pattern should be invalidated")
	}

	_, exists = cache.Get("table:users:2")
	if exists {
		t.Error("Keys matching pattern should be invalidated")
	}

	_, exists = cache.Get("table:apps:1")
	if !exists {
		t.Error("Keys not matching pattern should remain")
	}
}

func TestRepositoryQueryCacheClear(t *testing.T) {
	cache := NewRepositoryQueryCache(100, 5*time.Minute)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	cache.Clear()

	_, exists := cache.Get("key1")
	if exists {
		t.Error("After Clear, keys should not exist")
	}

	if len(cache.entries) != 0 {
		t.Errorf("Cache should be empty after Clear, got %d entries", len(cache.entries))
	}
}

func TestRepositoryQueryCacheStats(t *testing.T) {
	cache := NewRepositoryQueryCache(100, 5*time.Minute)

	cache.Set("key1", "value1")
	cache.Get("key1")
	cache.Get("key1")
	cache.Get("nonexistent")

	stats := cache.GetStats()

	if stats["max_size"] != 100 {
		t.Errorf("Expected max_size 100, got %v", stats["max_size"])
	}

	if stats["hits"].(int64) != 2 {
		t.Errorf("Expected 2 hits, got %v", stats["hits"])
	}

	if stats["misses"].(int64) != 1 {
		t.Errorf("Expected 1 miss, got %v", stats["misses"])
	}
}

func TestIndexManager(t *testing.T) {
	manager := NewIndexManager(nil)

	if manager == nil {
		t.Fatal("NewIndexManager should not return nil")
	}
}

func TestIndexManagerGetRequiredIndexes(t *testing.T) {
	manager := NewIndexManager(nil)
	indexes := manager.GetRequiredIndexes()

	if len(indexes) == 0 {
		t.Fatal("GetRequiredIndexes should return non-empty slice")
	}

	seen := make(map[string]bool)
	for _, idx := range indexes {
		if seen[idx.IndexName] {
			t.Errorf("Duplicate index name: %s", idx.IndexName)
		}
		seen[idx.IndexName] = true

		if idx.IndexName == "" {
			t.Error("IndexName should not be empty")
		}

		if idx.TableName == "" {
			t.Error("TableName should not be empty")
		}

		if len(idx.Columns) == 0 {
			t.Error("Columns should not be empty")
		}
	}
}

func TestBatchProcessor(t *testing.T) {
	processor := NewBatchProcessor(nil, 100)

	if processor == nil {
		t.Fatal("NewBatchProcessor should not return nil")
	}

	if processor.batchSize != 100 {
		t.Errorf("Expected batchSize 100, got %d", processor.batchSize)
	}
}

func TestORMQueryOptimizer(t *testing.T) {
	optimizer := NewORMQueryOptimizer(nil)

	if optimizer == nil {
		t.Fatal("NewORMQueryOptimizer should not return nil")
	}
}

func TestORMQueryOptimizerOptimize(t *testing.T) {
	optimizer := NewORMQueryOptimizer(nil)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Optimize SELECT with LIMIT",
			input:    "SELECT * FROM users",
			expected: "SELECT FROM users LIMIT 1000",
		},
		{
			name:     "Add LIMIT",
			input:    "SELECT id FROM users WHERE status = 'active'",
			expected: "SELECT id FROM users WHERE status = 'active' LIMIT 1000",
		},
		{
			name:     "Keep existing LIMIT",
			input:    "SELECT id FROM users LIMIT 10",
			expected: "SELECT id FROM users LIMIT 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := optimizer.Optimize(tt.input)
			if result != tt.expected {
				t.Errorf("Optimize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()

	if collector == nil {
		t.Fatal("NewMetricsCollector should not return nil")
	}
}

func TestMetricsCollectorRecordQuery(t *testing.T) {
	collector := NewMetricsCollector()

	collector.RecordQuery(50*time.Millisecond, false, "test")
	collector.RecordQuery(200*time.Millisecond, false, "test")
	collector.RecordQuery(100*time.Millisecond, true, "test")

	metrics := collector.GetMetrics()

	if metrics.TotalQueries != 3 {
		t.Errorf("Expected TotalQueries 3, got %d", metrics.TotalQueries)
	}

	if metrics.SlowQueries != 1 {
		t.Errorf("Expected SlowQueries 1, got %d", metrics.SlowQueries)
	}

	if metrics.FailedQueries != 1 {
		t.Errorf("Expected FailedQueries 1, got %d", metrics.FailedQueries)
	}
}

func TestMetricsCollectorRecordCache(t *testing.T) {
	collector := NewMetricsCollector()

	collector.RecordCacheHit()
	collector.RecordCacheHit()
	collector.RecordCacheMiss()

	metrics := collector.GetMetrics()

	if metrics.CacheHits != 2 {
		t.Errorf("Expected CacheHits 2, got %d", metrics.CacheHits)
	}

	if metrics.CacheMisses != 1 {
		t.Errorf("Expected CacheMisses 1, got %d", metrics.CacheMisses)
	}

	if metrics.CacheHitRate == 0 {
		t.Error("CacheHitRate should not be 0")
	}
}

func TestMetricsCollectorReset(t *testing.T) {
	collector := NewMetricsCollector()

	collector.RecordQuery(50*time.Millisecond, false, "test")
	collector.RecordCacheHit()

	collector.Reset()

	metrics := collector.GetMetrics()

	if metrics.TotalQueries != 0 {
		t.Errorf("Expected TotalQueries 0 after reset, got %d", metrics.TotalQueries)
	}

	if metrics.CacheHits != 0 {
		t.Errorf("Expected CacheHits 0 after reset, got %d", metrics.CacheHits)
	}
}

func TestOptimizerMetrics(t *testing.T) {
	metrics := &OptimizerMetrics{
		TotalQueries:     100,
		SlowQueries:     10,
		FailedQueries:    5,
		WriteOperations:  50,
		CacheHits:       80,
		CacheMisses:     20,
		CacheHitRate:    80.0,
		AvgQueryDuration: 50 * time.Millisecond,
		MaxQueryDuration: 500 * time.Millisecond,
	}

	if metrics.TotalQueries != 100 {
		t.Errorf("Expected TotalQueries 100, got %d", metrics.TotalQueries)
	}

	if metrics.CacheHitRate != 80.0 {
		t.Errorf("Expected CacheHitRate 80.0, got %f", metrics.CacheHitRate)
	}
}

func TestDBOptimizerHealthCheck(t *testing.T) {
	optimizer := NewDBOptimizer(nil, nil)

	err := optimizer.HealthCheck(context.Background())
	if err == nil {
		t.Error("HealthCheck with nil db should return error")
	}
}

func TestDBOptimizerCacheStats(t *testing.T) {
	optimizer := NewDBOptimizer(nil, nil)

	optimizer.queryCache.Set("test_key", "test_value")

	stats := optimizer.GetCacheStats()

	if stats["size"].(int) != 1 {
		t.Errorf("Expected cache size 1, got %v", stats["size"])
	}
}

func TestDBOptimizerGetMetrics(t *testing.T) {
	optimizer := NewDBOptimizer(nil, nil)

	optimizer.metricsCollector.RecordQuery(50*time.Millisecond, false, "test")

	metrics := optimizer.GetMetrics()

	if metrics.TotalQueries != 1 {
		t.Errorf("Expected TotalQueries 1, got %d", metrics.TotalQueries)
	}
}

func TestDBOptimizerClearCache(t *testing.T) {
	optimizer := NewDBOptimizer(nil, nil)

	optimizer.queryCache.Set("key1", "value1")
	optimizer.ClearCache()

	stats := optimizer.GetCacheStats()
	if stats["size"].(int) != 0 {
		t.Errorf("Expected cache size 0 after clear, got %v", stats["size"])
	}
}

func TestDBOptimizerInvalidateCache(t *testing.T) {
	optimizer := NewDBOptimizer(nil, nil)

	optimizer.queryCache.Set("table:users:1", "value1")
	optimizer.queryCache.Set("table:users:2", "value2")
	optimizer.queryCache.Set("table:apps:1", "value3")

	optimizer.InvalidateCache("table:users")

	stats := optimizer.GetCacheStats()
	if stats["size"].(int) != 1 {
		t.Errorf("Expected cache size 1 after invalidation, got %v", stats["size"])
	}
}

func TestMinFunction(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 1},
		{5, 3, 3},
		{10, 10, 10},
		{0, 5, 0},
		{-1, 1, -1},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestDBOptimizerSoftDelete(t *testing.T) {
	optimizer := NewDBOptimizer(nil, nil)

	err := optimizer.SoftDelete("users", []uint{})
	if err != nil {
		t.Error("SoftDelete with empty ids should not return error")
	}
}

func TestDBOptimizerRestore(t *testing.T) {
	optimizer := NewDBOptimizer(nil, nil)

	err := optimizer.Restore("users", []uint{})
	if err != nil {
		t.Error("Restore with empty ids should not return error")
	}
}

func TestDBOptimizerPaginatedQuery(t *testing.T) {
	optimizer := NewDBOptimizer(nil, nil)

	var dest []struct {
		ID int
	}

	_, err := optimizer.PaginatedQuery(context.Background(), "users", nil, 1, 10, &dest)
	if err == nil {
		t.Error("PaginatedQuery with nil db should return error")
	}
}

func TestDBOptimizerOptimizeQuery(t *testing.T) {
	optimizer := NewDBOptimizer(nil, nil)

	query := "SELECT * FROM users WHERE status = 'active'"
	_, err := optimizer.OptimizeQuery(query)
	if err == nil {
		t.Error("OptimizeQuery with nil db should return error")
	}
}

func TestORMQueryOptimizerMethods(t *testing.T) {
	optimizer := NewORMQueryOptimizer(nil)

	result := optimizer.Optimize("SELECT * FROM users")
	if result == "" {
		t.Error("Optimize should return non-empty string")
	}
}
