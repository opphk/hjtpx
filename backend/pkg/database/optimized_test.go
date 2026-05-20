package database

import (
	"testing"
	"time"
)

func TestOptimizedQueryCacheBasicOperations(t *testing.T) {
	cache := &OptimizedQueryCache{
		cache:    make(map[string]*CacheItem),
		maxSize:  100,
		baseTTL:  5 * time.Minute,
		enabled:  true,
		stats:   &CacheStatistics{},
	}

	cache.Set("test_key", "test_value")
	value, exists := cache.Get("test_key")
	if !exists {
		t.Error("Expected key to exist after Set")
	}
	if value != "test_value" {
		t.Errorf("Expected value 'test_value', got '%v'", value)
	}

	cache.Delete("test_key")
	_, exists = cache.Get("test_key")
	if exists {
		t.Error("Expected key to not exist after Delete")
	}
}

func TestOptimizedQueryCacheTTL(t *testing.T) {
	cache := &OptimizedQueryCache{
		cache:    make(map[string]*CacheItem),
		maxSize:  100,
		baseTTL:  1 * time.Millisecond,
		enabled:  true,
		stats:   &CacheStatistics{},
	}

	cache.Set("ttl_key", "ttl_value", 100*time.Millisecond)

	_, exists := cache.Get("ttl_key")
	if !exists {
		t.Error("Expected key to exist immediately after Set")
	}

	time.Sleep(150 * time.Millisecond)

	_, exists = cache.Get("ttl_key")
	if exists {
		t.Error("Expected key to not exist after TTL expired")
	}
}

func TestOptimizedQueryCacheEviction(t *testing.T) {
	cache := &OptimizedQueryCache{
		cache:    make(map[string]*CacheItem),
		maxSize:  5,
		baseTTL:  5 * time.Minute,
		enabled:  true,
		stats:   &CacheStatistics{},
	}

	for i := 0; i < 10; i++ {
		cache.Set(string(rune('a'+i)), i)
	}

	stats := cache.GetStats()
	size := stats["current_size"].(int64)
	if size > 5 {
		t.Errorf("Expected cache size <= 5 after eviction, got %d", size)
	}
}

func TestOptimizedQueryCacheClear(t *testing.T) {
	cache := &OptimizedQueryCache{
		cache:    make(map[string]*CacheItem),
		maxSize:  100,
		baseTTL:  5 * time.Minute,
		enabled:  true,
		stats:   &CacheStatistics{},
	}

	for i := 0; i < 5; i++ {
		cache.Set(string(rune('a'+i)), i)
	}

	cache.Clear()

	stats := cache.GetStats()
	size := stats["current_size"].(int64)
	if size != 0 {
		t.Errorf("Expected cache size 0 after Clear, got %d", size)
	}
}

func TestOptimizedQueryCacheHitRate(t *testing.T) {
	cache := &OptimizedQueryCache{
		cache:    make(map[string]*CacheItem),
		maxSize:  100,
		baseTTL:  5 * time.Minute,
		enabled:  true,
		stats:   &CacheStatistics{},
	}

	cache.Set("key1", "value1")

	cache.Get("key1")
	cache.Get("key1")
	cache.Get("key1")
	cache.Get("key1")

	cache.Get("nonexistent")

	stats := cache.GetStats()
	hitRate := stats["hit_rate"].(string)

	t.Logf("Hit rate: %s", hitRate)

	if stats["total_hits"].(int64) < 4 {
		t.Errorf("Expected at least 4 hits, got %d", stats["total_hits"].(int64))
	}
}

func TestOptimizedConnectionPool(t *testing.T) {
	settings := &PoolSettings{
		MaxOpenConns:        100,
		MaxIdleConns:        20,
		MinIdleConns:        5,
		ConnMaxLifetime:     30 * time.Minute,
		ConnMaxIdleTime:     10 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
		AutoTuning:         true,
		HighLoadThreshold:   80,
		LowLoadThreshold:   20,
	}

	if settings.MaxOpenConns != 100 {
		t.Errorf("Expected MaxOpenConns=100, got %d", settings.MaxOpenConns)
	}
	if settings.AutoTuning != true {
		t.Error("Expected AutoTuning=true")
	}
}

func TestOptimizedQueryAnalyzer(t *testing.T) {
	analyzer := NewOptimizedQueryAnalyzer(nil, 50*time.Millisecond)

	if analyzer == nil {
		t.Fatal("Expected analyzer to be created")
	}

	if analyzer.slowThreshold != 50*time.Millisecond {
		t.Errorf("Expected slowThreshold=50ms, got %v", analyzer.slowThreshold)
	}
}

func TestQueryPatternRecording(t *testing.T) {
	analyzer := NewOptimizedQueryAnalyzer(nil, 50*time.Millisecond)

	analyzer.RecordQuery("SELECT * FROM users WHERE id = ?", 100*time.Millisecond)
	analyzer.RecordQuery("SELECT * FROM users WHERE id = ?", 150*time.Millisecond)
	analyzer.RecordQuery("SELECT * FROM users WHERE id = ?", 120*time.Millisecond)

	patterns := analyzer.GetTopPatterns(10)
	if len(patterns) == 0 {
		t.Error("Expected at least one pattern recorded")
	}
}
