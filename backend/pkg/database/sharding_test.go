package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

func TestShardingStrategy(t *testing.T) {
	cfg := ShardingConfig{
		Enabled:           true,
		ShardCount:        4,
		VirtualNodes:      100,
		ShardingKey:       "user_id",
		ShardingAlgorithm: "hash",
	}

	ss, err := NewShardingStrategy(cfg)
	if err != nil {
		t.Fatalf("Failed to create sharding strategy: %v", err)
	}

	if !ss.IsEnabled() {
		t.Error("Expected sharding to be enabled")
	}

	if ss.GetShardCount() != cfg.ShardCount {
		t.Errorf("Expected shard count %d, got %d", cfg.ShardCount, ss.GetShardCount())
	}

	shards := ss.GetAllShards()
	if len(shards) != 0 {
		t.Errorf("Expected 0 shards initially, got %d", len(shards))
	}
}

func TestShardingStrategyDisabled(t *testing.T) {
	cfg := ShardingConfig{
		Enabled:   false,
		ShardCount: 4,
	}

	ss, err := NewShardingStrategy(cfg)
	if err != nil {
		t.Fatalf("Failed to create sharding strategy: %v", err)
	}

	if ss.IsEnabled() {
		t.Error("Expected sharding to be disabled")
	}

	_, err = ss.GetShard("test_key")
	if err == nil {
		t.Error("Expected error when sharding is disabled")
	}
}

func TestShardRouter(t *testing.T) {
	sr := newShardRouter(4, 100)

	if sr.totalSlots != 400 {
		t.Errorf("Expected total slots 400, got %d", sr.totalSlots)
	}

	if len(sr.vNodeMap) != 400 {
		t.Errorf("Expected 400 virtual nodes, got %d", len(sr.vNodeMap))
	}

	for slot, shardIndex := range sr.vNodeMap {
		if shardIndex < 0 || shardIndex >= 4 {
			t.Errorf("Invalid shard index %d for slot %d", shardIndex, slot)
		}
	}
}

func TestQueryCache(t *testing.T) {
	cache := NewQueryCache(10, time.Minute)

	if cache == nil {
		t.Fatal("Expected cache to be created")
	}

	_, ok := cache.Get("test_key")
	if ok {
		t.Error("Expected cache miss for non-existent key")
	}

	cache.Set("test_key", "test_value", "SELECT * FROM test", nil)

	cached, ok := cache.Get("test_key")
	if !ok {
		t.Error("Expected cache hit after setting")
	}

	if cached.Result != "test_value" {
		t.Errorf("Expected 'test_value', got %v", cached.Result)
	}
}

func TestQueryCacheExpiration(t *testing.T) {
	cache := NewQueryCache(10, 100*time.Millisecond)

	cache.Set("test_key", "test_value", "SELECT * FROM test", nil)

	time.Sleep(50 * time.Millisecond)

	_, ok := cache.Get("test_key")
	if !ok {
		t.Error("Expected cache hit before expiration")
	}

	time.Sleep(100 * time.Millisecond)

	_, ok = cache.Get("test_key")
	if ok {
		t.Error("Expected cache miss after expiration")
	}
}

func TestQueryCacheEviction(t *testing.T) {
	cache := NewQueryCache(2, time.Hour)

	cache.Set("key1", "value1", "SELECT 1", nil)
	cache.Set("key2", "value2", "SELECT 2", nil)
	cache.Set("key3", "value3", "SELECT 3", nil)

	_, ok := cache.Get("key1")
	if ok {
		t.Error("Expected key1 to be evicted")
	}

	_, ok = cache.Get("key2")
	if !ok {
		t.Error("Expected key2 to still exist")
	}

	_, ok = cache.Get("key3")
	if !ok {
		t.Error("Expected key3 to exist")
	}
}

func TestQueryCacheInvalidation(t *testing.T) {
	cache := NewQueryCache(10, time.Hour)

	cache.Set("user:1", "user_data_1", "SELECT * FROM users WHERE id=1", nil)
	cache.Set("user:2", "user_data_2", "SELECT * FROM users WHERE id=2", nil)
	cache.Set("order:1", "order_data_1", "SELECT * FROM orders WHERE id=1", nil)

	cache.InvalidatePattern("user:")

	_, ok := cache.Get("user:1")
	if ok {
		t.Error("Expected user:1 to be invalidated")
	}

	_, ok = cache.Get("user:2")
	if ok {
		t.Error("Expected user:2 to be invalidated")
	}

	_, ok = cache.Get("order:1")
	if !ok {
		t.Error("Expected order:1 to still exist")
	}
}

func TestQueryCacheStats(t *testing.T) {
	cache := NewQueryCache(10, time.Hour)

	cache.Set("key1", "value1", "SELECT 1", nil)
	cache.Get("key1")
	cache.Get("key2")
	cache.Get("key3")

	stats := cache.GetStats()

	if stats["size"].(int) != 1 {
		t.Errorf("Expected size 1, got %v", stats["size"])
	}

	if stats["hits"].(int64) != 1 {
		t.Errorf("Expected 1 hit, got %v", stats["hits"])
	}

	if stats["misses"].(int64) != 2 {
		t.Errorf("Expected 2 misses, got %v", stats["misses"])
	}

	if stats["hit_rate"].(float64) != 33.33333333333333 {
		t.Errorf("Expected hit rate 33.33, got %v", stats["hit_rate"])
	}
}

func TestQueryCacheManager(t *testing.T) {
	manager := NewQueryCacheManager()

	cache1 := manager.CreateCache("users", 100, time.Hour)
	if cache1 == nil {
		t.Fatal("Expected cache to be created")
	}

	cache2 := manager.GetCache("users")
	if cache2 == nil {
		t.Fatal("Expected cache to be retrieved")
	}

	cache3 := manager.GetCache("orders")
	if cache3 != nil {
		t.Error("Expected nil for non-existent cache")
	}

	manager.CreateCache("orders", 50, 30*time.Minute)

	allStats := manager.GetAllStats()
	if len(allStats) != 2 {
		t.Errorf("Expected 2 caches in stats, got %d", len(allStats))
	}

	manager.DeleteCache("users")

	_, ok := manager.GetCache("users")
	if ok {
		t.Error("Expected users cache to be deleted")
	}
}

func TestShardingKeyGenerator(t *testing.T) {
	generator := NewShardingKeyGenerator("user_id")

	userKey := generator.GenerateFromUserID("123")
	expectedUserKey := "user:123"
	if userKey != expectedUserKey {
		t.Errorf("Expected '%s', got '%s'", expectedUserKey, userKey)
	}

	tenantKey := generator.GenerateFromTenantID("456")
	expectedTenantKey := "tenant:456"
	if tenantKey != expectedTenantKey {
		t.Errorf("Expected '%s', got '%s'", expectedTenantKey, tenantKey)
	}

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	timeKey := generator.GenerateFromTimeRange(start, end)
	expectedTimeKey := "time:20240101:20240131"
	if timeKey != expectedTimeKey {
		t.Errorf("Expected '%s', got '%s'", expectedTimeKey, timeKey)
	}

	composite := generator.GenerateComposite("user:123", "tenant:456")
	expectedComposite := "user:123:tenant:456"
	if composite != expectedComposite {
		t.Errorf("Expected '%s', got '%s'", expectedComposite, composite)
	}
}

func TestDatabaseHealthCheck(t *testing.T) {
	db := &MockDB{}

	hc := NewDatabaseHealthCheck(db, time.Second)

	if hc.IsHealthy() {
		t.Error("Expected initial health to be true")
	}

	hc.check()

	if !hc.IsHealthy() {
		t.Error("Expected health to be true after successful check")
	}

	status := hc.GetStatus()
	if status["healthy"].(bool) {
		t.Error("Expected health to be false after setting unhealthy")
	}
}

func TestDatabaseHealthCheckStartStop(t *testing.T) {
	db := &MockDB{}

	hc := NewDatabaseHealthCheck(db, 100*time.Millisecond)

	hc.Start()

	time.Sleep(250 * time.Millisecond)

	hc.Stop()
}

type MockDB struct{}

func (m *MockDB) DB() (interface{}, error) {
	return &MockSQLDB{}, nil
}

type MockSQLDB struct{}

func (m *MockSQLDB) PingContext(ctx context.Context) error {
	return nil
}

func BenchmarkQueryCacheGet(b *testing.B) {
	cache := NewQueryCache(10000, time.Hour)

	for i := 0; i < 1000; i++ {
		cache.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i), "SELECT *", nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("key999")
	}
}

func BenchmarkQueryCacheSet(b *testing.B) {
	cache := NewQueryCache(10000, time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i), "SELECT *", nil)
	}
}

func BenchmarkShardingKeyHash(b *testing.B) {
	key := "user:12345"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hashShardingKey(key, 1000)
	}
}
