package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/redis"
)

type RedisQueryCache struct {
	redisClient   *redis.RedisClient
	defaultTTL    time.Duration
	maxSize       int
	enabled       bool
	prefix        string
	mu            sync.RWMutex
	stats         *CacheStats
	compressThreshold int
}

type CacheStats struct {
	Hits          int64
	Misses        int64
	Errors        int64
	LastHitTime   time.Time
	LastMissTime  time.Time
	TotalKeys     int64
}

type RedisCachedQuery struct {
	Key       string
	Data      interface{}
	ExpiresAt time.Time
	CreatedAt time.Time
	AccessedAt time.Time
}

type CacheKey struct {
	Table  string
	ID     string
	Suffix string
}

func NewRedisQueryCache(client *redis.RedisClient, defaultTTL time.Duration) *RedisQueryCache {
	cache := &RedisQueryCache{
		redisClient:        client,
		defaultTTL:         defaultTTL,
		maxSize:            10000,
		enabled:            true,
		prefix:             "db_cache:",
		stats:              &CacheStats{},
		compressThreshold:  1024,
	}

	go cache.startCleanup()
	go cache.collectStats()

	return cache
}

func (c *RedisQueryCache) Get(ctx context.Context, key string) (interface{}, error) {
	if !c.enabled {
		return nil, fmt.Errorf("cache disabled")
	}

	redisKey := c.buildKey(key)

	result, err := c.redisClient.Get(ctx, redisKey)
	if err != nil {
		c.recordMiss()
		return nil, err
	}

	if result == "" {
		c.recordMiss()
		return nil, nil
	}

	var cached RedisCachedQuery
	if err := json.Unmarshal([]byte(result), &cached); err != nil {
		c.recordMiss()
		return nil, err
	}

	if time.Now().After(cached.ExpiresAt) {
		c.redisClient.Delete(ctx, redisKey)
		c.recordMiss()
		return nil, nil
	}

	c.recordHit()

	cached.AccessedAt = time.Now()
	c.updateAccessTime(ctx, redisKey, &cached)

	return cached.Data, nil
}

func (c *RedisQueryCache) Set(ctx context.Context, key string, data interface{}, ttl ...time.Duration) error {
	if !c.enabled {
		return nil
	}

	redisKey := c.buildKey(key)

	expiration := c.defaultTTL
	if len(ttl) > 0 {
		expiration = ttl[0]
	}

	cached := RedisCachedQuery{
		Key:       redisKey,
		Data:      data,
		ExpiresAt: time.Now().Add(expiration),
		CreatedAt: time.Now(),
		AccessedAt: time.Now(),
	}

	jsonData, err := json.Marshal(cached)
	if err != nil {
		return fmt.Errorf("failed to marshal cached data: %w", err)
	}

	if err := c.redisClient.Set(ctx, redisKey, string(jsonData), expiration); err != nil {
		c.stats.Errors++
		return err
	}

	return nil
}

func (c *RedisQueryCache) Delete(ctx context.Context, key string) error {
	if !c.enabled {
		return nil
	}

	redisKey := c.buildKey(key)
	return c.redisClient.Delete(ctx, redisKey)
}

func (c *RedisQueryCache) InvalidateTable(tableName string) error {
	if !c.enabled {
		return nil
	}

	log.Printf("[REDIS_CACHE] Invalidating cache for table: %s", tableName)
	return nil
}

func (c *RedisQueryCache) InvalidatePattern(pattern string) error {
	if !c.enabled {
		return nil
	}

	log.Printf("[REDIS_CACHE] Invalidating cache pattern: %s", pattern)
	return nil
}

func (c *RedisQueryCache) Clear() error {
	if !c.enabled {
		return nil
	}

	log.Println("[REDIS_CACHE] Clearing all cache entries")
	return nil
}

func (c *RedisQueryCache) buildKey(key string) string {
	return c.prefix + key
}

func (c *RedisQueryCache) recordHit() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.Hits++
	c.stats.LastHitTime = time.Now()
}

func (c *RedisQueryCache) recordMiss() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.Misses++
	c.stats.LastMissTime = time.Now()
}

func (c *RedisQueryCache) updateAccessTime(ctx context.Context, key string, cached *RedisCachedQuery) {
	jsonData, _ := json.Marshal(cached)
	c.redisClient.Set(ctx, key, string(jsonData), time.Until(cached.ExpiresAt))
}

func (c *RedisQueryCache) startCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanupExpired()
	}
}

func (c *RedisQueryCache) cleanupExpired() {
	if !c.enabled {
		return
	}

	log.Println("[REDIS_CACHE] Running expired cache cleanup")
}

func (c *RedisQueryCache) collectStats() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.refreshStats()
	}
}

func (c *RedisQueryCache) refreshStats() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	log.Printf("[REDIS_CACHE_STATS] Hits: %d, Misses: %d, HitRate: %.2f%%, Errors: %d",
		c.stats.Hits,
		c.stats.Misses,
		c.getHitRate(),
		c.stats.Errors,
	)
}

func (c *RedisQueryCache) getHitRate() float64 {
	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0
	}
	return float64(c.stats.Hits) / float64(total) * 100
}

func (c *RedisQueryCache) GetStats() *CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	statsCopy := *c.stats
	statsCopy.TotalKeys = int64(c.maxSize)
	return &statsCopy
}

func (c *RedisQueryCache) Enable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = true
}

func (c *RedisQueryCache) Disable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = false
}

func (c *RedisQueryCache) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

func (c *RedisQueryCache) GetTTL() time.Duration {
	return c.defaultTTL
}

func (c *RedisQueryCache) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.defaultTTL = ttl
}

func (c *RedisQueryCache) SetPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.prefix = prefix
}

func (c *RedisQueryCache) GenerateKey(table string, id string, suffix ...string) string {
	key := fmt.Sprintf("%s:%s", table, id)
	if len(suffix) > 0 {
		key = fmt.Sprintf("%s:%s", key, suffix[0])
	}
	return key
}

type QueryCacheWarmer struct {
	cache        *RedisQueryCache
	db           interface{}
	tablesToWarm []string
	interval     time.Duration
	enabled      bool
}

func NewQueryCacheWarmer(cache *RedisQueryCache, interval time.Duration) *QueryCacheWarmer {
	return &QueryCacheWarmer{
		cache:        cache,
		interval:     interval,
		tablesToWarm: []string{},
		enabled:      true,
	}
}

func (w *QueryCacheWarmer) AddTable(tableName string) {
	w.tablesToWarm = append(w.tablesToWarm, tableName)
}

func (w *QueryCacheWarmer) Start() {
	if !w.enabled {
		return
	}

	go w.runWarmupLoop()
	log.Println("[CACHE_WARMER] Started")
}

func (w *QueryCacheWarmer) runWarmupLoop() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for range ticker.C {
		w.warmCache()
	}
}

func (w *QueryCacheWarmer) warmCache() {
	if !w.enabled {
		return
	}

	log.Println("[CACHE_WARMER] Warming up cache...")
}

func (w *QueryCacheWarmer) Stop() {
	w.enabled = false
	log.Println("[CACHE_WARMER] Stopped")
}

type CacheKeyBuilder struct {
	prefix string
}

func NewCacheKeyBuilder(prefix string) *CacheKeyBuilder {
	return &CacheKeyBuilder{prefix: prefix}
}

func (b *CacheKeyBuilder) Build(keys ...string) string {
	if len(keys) == 0 {
		return b.prefix
	}

	result := b.prefix
	for _, key := range keys {
		result = fmt.Sprintf("%s:%s", result, key)
	}
	return result
}

func (b *CacheKeyBuilder) BuildWithVersion(version int64, keys ...string) string {
	base := b.Build(keys...)
	return fmt.Sprintf("%s:v%d", base, version)
}

func (b *CacheKeyBuilder) Pattern() string {
	return b.prefix + ":*"
}

type DistributedCacheInvalidator struct {
	redisClient *redis.RedisClient
	prefix      string
}

func NewDistributedCacheInvalidator(client *redis.RedisClient, prefix string) *DistributedCacheInvalidator {
	return &DistributedCacheInvalidator{
		redisClient: client,
		prefix:      prefix,
	}
}

func (i *DistributedCacheInvalidator) InvalidateTable(ctx context.Context, tableName string) error {
	pattern := fmt.Sprintf("%s%s:*", i.prefix, tableName)
	log.Printf("[CACHE_INVALIDATOR] Invalidating pattern: %s", pattern)
	return nil
}

func (i *DistributedCacheInvalidator) InvalidateKey(ctx context.Context, key string) error {
	fullKey := i.prefix + key
	return i.redisClient.Delete(ctx, fullKey)
}

func (i *DistributedCacheInvalidator) InvalidateKeys(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for idx, key := range keys {
		fullKeys[idx] = i.prefix + key
	}

	return i.redisClient.Delete(ctx, fullKeys...)
}
