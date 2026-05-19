package database

import (
	"testing"
	"time"
)

func TestRedisQueryCacheCreation(t *testing.T) {
	cache := &RedisQueryCache{
		redisClient:        nil,
		defaultTTL:         5 * time.Minute,
		maxSize:            10000,
		enabled:            true,
		prefix:             "db_cache:",
		stats:              &CacheStats{},
		compressThreshold:  1024,
	}

	if cache == nil {
		t.Fatal("RedisQueryCache should not be nil")
	}

	if cache.defaultTTL != 5*time.Minute {
		t.Errorf("Default TTL = %v, want %v", cache.defaultTTL, 5*time.Minute)
	}

	if cache.maxSize != 10000 {
		t.Errorf("Max size = %d, want %d", cache.maxSize, 10000)
	}

	if !cache.enabled {
		t.Error("Cache should be enabled")
	}

	if cache.prefix != "db_cache:" {
		t.Errorf("Prefix = %q, want %q", cache.prefix, "db_cache:")
	}
}

func TestCacheStats(t *testing.T) {
	stats := &CacheStats{
		Hits:          100,
		Misses:        20,
		Errors:        5,
		LastHitTime:   time.Now(),
		LastMissTime:  time.Now(),
		TotalKeys:     50,
	}

	if stats.Hits != 100 {
		t.Errorf("Hits = %d, want 100", stats.Hits)
	}

	if stats.Misses != 20 {
		t.Errorf("Misses = %d, want 20", stats.Misses)
	}

	if stats.Errors != 5 {
		t.Errorf("Errors = %d, want 5", stats.Errors)
	}

	if stats.TotalKeys != 50 {
		t.Errorf("TotalKeys = %d, want 50", stats.TotalKeys)
	}
}

func TestRedisCachedQuery(t *testing.T) {
	now := time.Now()
	cached := &RedisCachedQuery{
		Key:       "test_key",
		Data:      "test_data",
		ExpiresAt: now.Add(5 * time.Minute),
		CreatedAt: now,
		AccessedAt: now,
	}

	if cached.Key != "test_key" {
		t.Errorf("Key = %q, want %q", cached.Key, "test_key")
	}

	if cached.Data != "test_data" {
		t.Errorf("Data = %v, want %v", cached.Data, "test_data")
	}

	if cached.ExpiresAt.Before(now) {
		t.Error("ExpiresAt should be in the future")
	}
}

func TestCacheKey(t *testing.T) {
	cacheKey := &CacheKey{
		Table:  "users",
		ID:     "123",
		Suffix: "profile",
	}

	if cacheKey.Table != "users" {
		t.Errorf("Table = %q, want %q", cacheKey.Table, "users")
	}

	if cacheKey.ID != "123" {
		t.Errorf("ID = %q, want %q", cacheKey.ID, "123")
	}

	if cacheKey.Suffix != "profile" {
		t.Errorf("Suffix = %q, want %q", cacheKey.Suffix, "profile")
	}
}

func TestRedisQueryCacheBuildKey(t *testing.T) {
	cache := &RedisQueryCache{
		prefix: "db_cache:",
	}

	key := cache.buildKey("users:123")
	expected := "db_cache:users:123"

	if key != expected {
		t.Errorf("buildKey() = %q, want %q", key, expected)
	}
}

func TestRedisQueryCacheGenerateKey(t *testing.T) {
	cache := &RedisQueryCache{}

	key := cache.GenerateKey("users", "123")
	if key != "users:123" {
		t.Errorf("GenerateKey() = %q, want %q", key, "users:123")
	}

	keyWithSuffix := cache.GenerateKey("users", "123", "profile")
	if keyWithSuffix != "users:123:profile" {
		t.Errorf("GenerateKey() with suffix = %q, want %q", keyWithSuffix, "users:123:profile")
	}
}

func TestRedisQueryCacheEnableDisable(t *testing.T) {
	cache := &RedisQueryCache{
		enabled: true,
	}

	cache.Disable()
	if cache.enabled {
		t.Error("Cache should be disabled after Disable()")
	}

	cache.Enable()
	if !cache.enabled {
		t.Error("Cache should be enabled after Enable()")
	}
}

func TestRedisQueryCacheIsEnabled(t *testing.T) {
	cache := &RedisQueryCache{
		enabled: true,
	}

	if !cache.IsEnabled() {
		t.Error("IsEnabled() should return true")
	}

	cache.enabled = false
	if cache.IsEnabled() {
		t.Error("IsEnabled() should return false")
	}
}

func TestRedisQueryCacheGetTTL(t *testing.T) {
	cache := &RedisQueryCache{
		defaultTTL: 10 * time.Minute,
	}

	if cache.GetTTL() != 10*time.Minute {
		t.Errorf("GetTTL() = %v, want %v", cache.GetTTL(), 10*time.Minute)
	}
}

func TestRedisQueryCacheSetTTL(t *testing.T) {
	cache := &RedisQueryCache{
		defaultTTL: 5 * time.Minute,
	}

	cache.SetTTL(15 * time.Minute)
	if cache.defaultTTL != 15*time.Minute {
		t.Errorf("defaultTTL = %v, want %v", cache.defaultTTL, 15*time.Minute)
	}
}

func TestRedisQueryCacheSetPrefix(t *testing.T) {
	cache := &RedisQueryCache{
		prefix: "old_prefix:",
	}

	cache.SetPrefix("new_prefix:")
	if cache.prefix != "new_prefix:" {
		t.Errorf("prefix = %q, want %q", cache.prefix, "new_prefix:")
	}
}

func TestRedisQueryCacheGetStats(t *testing.T) {
	cache := &RedisQueryCache{
		stats: &CacheStats{
			Hits:   100,
			Misses: 20,
			Errors: 5,
		},
		maxSize: 10000,
	}

	stats := cache.GetStats()
	if stats.Hits != 100 {
		t.Errorf("Stats Hits = %d, want 100", stats.Hits)
	}

	if stats.Misses != 20 {
		t.Errorf("Stats Misses = %d, want 20", stats.Misses)
	}

	if stats.TotalKeys != 10000 {
		t.Errorf("Stats TotalKeys = %d, want 10000", stats.TotalKeys)
	}
}

func TestRedisQueryCacheGetHitRate(t *testing.T) {
	cache := &RedisQueryCache{
		stats: &CacheStats{
			Hits:   80,
			Misses: 20,
		},
	}

	hitRate := cache.getHitRate()
	expected := 80.0

	if hitRate != expected {
		t.Errorf("getHitRate() = %f, want %f", hitRate, expected)
	}
}

func TestRedisQueryCacheGetHitRateZero(t *testing.T) {
	cache := &RedisQueryCache{
		stats: &CacheStats{
			Hits:   0,
			Misses: 0,
		},
	}

	hitRate := cache.getHitRate()
	if hitRate != 0 {
		t.Errorf("getHitRate() = %f, want 0", hitRate)
	}
}

func TestQueryCacheWarmerCreation(t *testing.T) {
	warmer := &QueryCacheWarmer{
		cache:        nil,
		tablesToWarm: []string{"users", "products"},
		interval:     1 * time.Hour,
		enabled:      true,
	}

	if warmer == nil {
		t.Fatal("QueryCacheWarmer should not be nil")
	}

	if len(warmer.tablesToWarm) != 2 {
		t.Errorf("Tables to warm count = %d, want 2", len(warmer.tablesToWarm))
	}

	if warmer.interval != 1*time.Hour {
		t.Errorf("Interval = %v, want %v", warmer.interval, 1*time.Hour)
	}

	if !warmer.enabled {
		t.Error("Warmer should be enabled")
	}
}

func TestQueryCacheWarmerAddTable(t *testing.T) {
	warmer := &QueryCacheWarmer{
		tablesToWarm: []string{"users"},
	}

	warmer.AddTable("products")
	warmer.AddTable("orders")

	if len(warmer.tablesToWarm) != 3 {
		t.Errorf("Tables to warm count = %d, want 3", len(warmer.tablesToWarm))
	}
}

func TestQueryCacheWarmerStop(t *testing.T) {
	warmer := &QueryCacheWarmer{
		enabled: true,
	}

	warmer.Stop()
	if warmer.enabled {
		t.Error("Warmer should be disabled after Stop()")
	}
}

func TestCacheKeyBuilderCreation(t *testing.T) {
	builder := NewCacheKeyBuilder("prefix")

	if builder == nil {
		t.Fatal("NewCacheKeyBuilder should not return nil")
	}

	if builder.prefix != "prefix" {
		t.Errorf("prefix = %q, want %q", builder.prefix, "prefix")
	}
}

func TestCacheKeyBuilderBuild(t *testing.T) {
	builder := NewCacheKeyBuilder("db")

	key := builder.Build()
	if key != "db" {
		t.Errorf("Build() with no keys = %q, want %q", key, "db")
	}

	key = builder.Build("users")
	if key != "db:users" {
		t.Errorf("Build() with one key = %q, want %q", key, "db:users")
	}

	key = builder.Build("users", "123")
	if key != "db:users:123" {
		t.Errorf("Build() with two keys = %q, want %q", key, "db:users:123")
	}
}

func TestCacheKeyBuilderBuildWithVersion(t *testing.T) {
	builder := NewCacheKeyBuilder("db")

	key := builder.BuildWithVersion(1, "users", "123")
	expected := "db:users:123:v1"

	if key != expected {
		t.Errorf("BuildWithVersion() = %q, want %q", key, expected)
	}
}

func TestCacheKeyBuilderPattern(t *testing.T) {
	builder := NewCacheKeyBuilder("db")

	pattern := builder.Pattern()
	expected := "db:*"

	if pattern != expected {
		t.Errorf("Pattern() = %q, want %q", pattern, expected)
	}
}

func TestDistributedCacheInvalidatorCreation(t *testing.T) {
	invalidator := &DistributedCacheInvalidator{
		redisClient: nil,
		prefix:      "cache:",
	}

	if invalidator == nil {
		t.Fatal("DistributedCacheInvalidator should not be nil")
	}

	if invalidator.prefix != "cache:" {
		t.Errorf("prefix = %q, want %q", invalidator.prefix, "cache:")
	}
}

func TestRedisQueryCacheDisabledGet(t *testing.T) {
	cache := &RedisQueryCache{
		enabled: false,
	}

	_, err := cache.Get(nil, "test_key")
	if err == nil {
		t.Error("Get should return error when cache is disabled")
	}
}

func TestRedisQueryCacheDisabledSet(t *testing.T) {
	cache := &RedisQueryCache{
		enabled: false,
	}

	err := cache.Set(nil, "test_key", "test_data")
	if err != nil {
		t.Error("Set should return nil when cache is disabled")
	}
}

func TestRedisQueryCacheDisabledDelete(t *testing.T) {
	cache := &RedisQueryCache{
		enabled: false,
	}

	err := cache.Delete(nil, "test_key")
	if err != nil {
		t.Error("Delete should return nil when cache is disabled")
	}
}

func TestRedisQueryCacheDisabledInvalidateTable(t *testing.T) {
	cache := &RedisQueryCache{
		enabled: false,
	}

	err := cache.InvalidateTable("users")
	if err != nil {
		t.Error("InvalidateTable should return nil when cache is disabled")
	}
}

func TestRedisQueryCacheDisabledInvalidatePattern(t *testing.T) {
	cache := &RedisQueryCache{
		enabled: false,
	}

	err := cache.InvalidatePattern("users:*")
	if err != nil {
		t.Error("InvalidatePattern should return nil when cache is disabled")
	}
}

func TestRedisQueryCacheDisabledClear(t *testing.T) {
	cache := &RedisQueryCache{
		enabled: false,
	}

	err := cache.Clear()
	if err != nil {
		t.Error("Clear should return nil when cache is disabled")
	}
}

func TestQueryCacheWarmerDisabledStart(t *testing.T) {
	warmer := &QueryCacheWarmer{
		enabled: false,
	}

	warmer.Start()
	if warmer.enabled {
		t.Error("Warmer should remain disabled after Start() when disabled")
	}
}
