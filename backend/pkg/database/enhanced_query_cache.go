package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/redis/go-redis/v9"
)

type EnhancedQueryCache struct {
	redisClient *redis.Client
	localCache  *LocalQueryCache
	config      *CacheConfig
	mu          sync.RWMutex
	enabled     bool
}

type CacheConfig struct {
	UseRedis           bool
	RedisTTL           time.Duration
	LocalTTL           time.Duration
	MaxLocalSize       int
	EnableCompression   bool
	CacheKeyPrefix     string
	EnableCacheMetrics bool
}

type LocalQueryCache struct {
	mu      sync.RWMutex
	cache   map[string]*CacheEntry
	maxSize int
	ttl     time.Duration
	hits    int64
	misses  int64
}

type CacheEntry struct {
	Value      interface{}
	Expiration time.Time
	CreatedAt  time.Time
}

var enhancedCache *EnhancedQueryCache

func InitEnhancedQueryCache(redisClient *redis.Client, cfg *config.Config) *EnhancedQueryCache {
	cacheConfig := &CacheConfig{
		UseRedis:         redisClient != nil,
		RedisTTL:         time.Duration(cfg.Database.QueryOptimization.QueryCacheTTLSecs) * time.Second,
		LocalTTL:         5 * time.Minute,
		MaxLocalSize:     cfg.Database.QueryOptimization.MaxQueryCacheSize,
		EnableCompression: false,
		CacheKeyPrefix:   "query_cache:",
		EnableCacheMetrics: true,
	}

	enhancedCache = &EnhancedQueryCache{
		redisClient: redisClient,
		localCache: &LocalQueryCache{
			cache:   make(map[string]*CacheEntry),
			maxSize: cacheConfig.MaxLocalSize,
			ttl:     cacheConfig.LocalTTL,
		},
		config:  cacheConfig,
		enabled: true,
	}

	go enhancedCache.startCleanup()
	go enhancedCache.startMetricsReporter()

	log.Printf("[ENHANCED_CACHE] Initialized with Redis: %v, Local size: %d", cacheConfig.UseRedis, cacheConfig.MaxLocalSize)
	return enhancedCache
}

func GetEnhancedQueryCache() *EnhancedQueryCache {
	return enhancedCache
}

func (c *EnhancedQueryCache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	if !c.enabled {
		return nil, false, nil
	}

	cacheKey := c.config.CacheKeyPrefix + key

	if c.config.UseRedis && c.redisClient != nil {
		val, err := c.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var result interface{}
			if err := json.Unmarshal([]byte(val), &result); err == nil {
				c.localCache.RecordHit()
				return result, true, nil
			}
		}
	}

	if localVal, found := c.localCache.Get(key); found {
		return localVal, true, nil
	}

	c.localCache.RecordMiss()
	return nil, false, nil
}

func (c *EnhancedQueryCache) Set(ctx context.Context, key string, value interface{}) error {
	if !c.enabled {
		return nil
	}

	cacheKey := c.config.CacheKeyPrefix + key

	if c.config.UseRedis && c.redisClient != nil {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal cache value: %w", err)
		}

		if err := c.redisClient.Set(ctx, cacheKey, data, c.config.RedisTTL).Err(); err != nil {
			log.Printf("[ENHANCED_CACHE] Failed to set Redis cache: %v", err)
		}
	}

	c.localCache.Set(key, value)
	return nil
}

func (c *EnhancedQueryCache) Delete(ctx context.Context, key string) error {
	if !c.enabled {
		return nil
	}

	cacheKey := c.config.CacheKeyPrefix + key

	if c.config.UseRedis && c.redisClient != nil {
		if err := c.redisClient.Del(ctx, cacheKey).Err(); err != nil {
			log.Printf("[ENHANCED_CACHE] Failed to delete from Redis: %v", err)
		}
	}

	c.localCache.Delete(key)
	return nil
}

func (c *EnhancedQueryCache) DeletePattern(ctx context.Context, pattern string) error {
	if !c.enabled {
		return nil
	}

	cacheKey := c.config.CacheKeyPrefix + pattern

	if c.config.UseRedis && c.redisClient != nil {
		keys, err := c.redisClient.Keys(ctx, cacheKey).Result()
		if err == nil && len(keys) > 0 {
			if err := c.redisClient.Del(ctx, keys...).Err(); err != nil {
				log.Printf("[ENHANCED_CACHE] Failed to delete pattern from Redis: %v", err)
			}
		}
	}

	c.localCache.DeletePattern(pattern)
	return nil
}

func (c *EnhancedQueryCache) Clear(ctx context.Context) error {
	if !c.enabled {
		return nil
	}

	if c.config.UseRedis && c.redisClient != nil {
		pattern := c.config.CacheKeyPrefix + "*"
		keys, err := c.redisClient.Keys(ctx, pattern).Result()
		if err == nil && len(keys) > 0 {
			if err := c.redisClient.Del(ctx, keys...).Err(); err != nil {
				log.Printf("[ENHANCED_CACHE] Failed to clear Redis cache: %v", err)
			}
		}
	}

	c.localCache.Clear()
	return nil
}

func (c *EnhancedQueryCache) GetOrSet(ctx context.Context, key string, fetchFunc func() (interface{}, error), ttl ...time.Duration) (interface{}, error) {
	if val, found, err := c.Get(ctx, key); found && err == nil {
		return val, nil
	}

	value, err := fetchFunc()
	if err != nil {
		return nil, err
	}

	if cacheErr := c.Set(ctx, key, value); cacheErr != nil {
		log.Printf("[ENHANCED_CACHE] Failed to cache value: %v", cacheErr)
	}

	return value, nil
}

func (c *EnhancedQueryCache) GetStats() *CacheStats {
	stats := &CacheStats{
		Enabled: c.enabled,
		Config: &CacheConfigInfo{
			UseRedis:         c.config.UseRedis,
			RedisTTL:         c.config.RedisTTL,
			LocalTTL:         c.config.LocalTTL,
			MaxLocalSize:     c.config.MaxLocalSize,
			EnableCompression: c.config.EnableCompression,
		},
	}

	if c.config.EnableCacheMetrics {
		stats.LocalStats = c.localCache.GetStats()
	}

	return stats
}

func (c *EnhancedQueryCache) startCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.localCache.CleanupExpired()
	}
}

func (c *EnhancedQueryCache) startMetricsReporter() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		stats := c.localCache.GetStats()
		if stats.TotalRequests > 0 {
			hitRate := float64(stats.Hits) / float64(stats.TotalRequests) * 100
			log.Printf("[ENHANCED_CACHE] Stats: hits=%d, misses=%d, hit_rate=%.2f%%, size=%d",
				stats.Hits, stats.Misses, hitRate, stats.Size)
		}
	}
}

func (l *LocalQueryCache) Get(key string) (interface{}, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entry, exists := l.cache[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.Expiration) {
		delete(l.cache, key)
		return nil, false
	}

	return entry.Value, true
}

func (l *LocalQueryCache) Set(key string, value interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.cache) >= l.maxSize {
		l.evictOldest()
	}

	l.cache[key] = &CacheEntry{
		Value:      value,
		Expiration: time.Now().Add(l.ttl),
		CreatedAt:  time.Now(),
	}
}

func (l *LocalQueryCache) Delete(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.cache, key)
}

func (l *LocalQueryCache) DeletePattern(pattern string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for key := range l.cache {
		if len(key) >= len(pattern) && key[:len(pattern)] == pattern {
			delete(l.cache, key)
		}
	}
}

func (l *LocalQueryCache) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache = make(map[string]*CacheEntry)
	l.hits = 0
	l.misses = 0
}

func (l *LocalQueryCache) CleanupExpired() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for key, entry := range l.cache {
		if now.After(entry.Expiration) {
			delete(l.cache, key)
		}
	}
}

func (l *LocalQueryCache) RecordHit() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hits++
}

func (l *LocalQueryCache) RecordMiss() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.misses++
}

func (l *LocalQueryCache) GetStats() *LocalCacheStats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return &LocalCacheStats{
		Size:         len(l.cache),
		MaxSize:      l.maxSize,
		Hits:         l.hits,
		Misses:       l.misses,
		TotalRequests: l.hits + l.misses,
	}
}

func (l *LocalQueryCache) evictOldest() {
	if len(l.cache) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time = time.Now().Add(24 * time.Hour)

	for key, entry := range l.cache {
		if entry.CreatedAt.Before(oldestTime) {
			oldestTime = entry.CreatedAt
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(l.cache, oldestKey)
	}
}

type CacheStats struct {
	Enabled    bool
	LocalStats *LocalCacheStats
	Config     *CacheConfigInfo
}

type LocalCacheStats struct {
	Size          int
	MaxSize       int
	Hits          int64
	Misses        int64
	TotalRequests int64
}

type CacheConfigInfo struct {
	UseRedis          bool
	RedisTTL          time.Duration
	LocalTTL          time.Duration
	MaxLocalSize      int
	EnableCompression bool
}

type CacheWarmer struct {
	cache      *EnhancedQueryCache
	queries    []CacheWarmQuery
	interval   time.Duration
	mu         sync.RWMutex
	running    bool
	stopChan   chan struct{}
}

type CacheWarmQuery struct {
	Key        string
	Query      string
	Params     []interface{}
	TTL        time.Duration
	ExecuteAt  time.Time
}

func NewCacheWarmer(cache *EnhancedQueryCache) *CacheWarmer {
	return &CacheWarmer{
		cache:    cache,
		queries:  make([]CacheWarmQuery, 0),
		interval: 30 * time.Minute,
		stopChan: make(chan struct{}),
	}
}

func (w *CacheWarmer) AddQuery(key, query string, params []interface{}, ttl time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.queries = append(w.queries, CacheWarmQuery{
		Key:   key,
		Query: query,
		Params: params,
		TTL:   ttl,
	})
}

func (w *CacheWarmer) Start(ctx context.Context) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	go w.warmingLoop(ctx)
	log.Println("[CACHE_WARMER] Started cache warming service")
}

func (w *CacheWarmer) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	close(w.stopChan)
	w.running = false
	log.Println("[CACHE_WARMER] Stopped cache warming service")
}

func (w *CacheWarmer) warmingLoop(ctx context.Context) {
	w.warmNow(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.warmNow(ctx)
		}
	}
}

func (w *CacheWarmer) warmNow(ctx context.Context) {
	w.mu.RLock()
	queries := make([]CacheWarmQuery, len(w.queries))
	copy(queries, w.queries)
	w.mu.RUnlock()

	for _, query := range queries {
		if !query.ExecuteAt.IsZero() && time.Now().Before(query.ExecuteAt) {
			continue
		}

		log.Printf("[CACHE_WARMER] Warming cache for key: %s", query.Key)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
}

func GenerateCacheKey(query string, args ...interface{}) string {
	data := query
	for _, arg := range args {
		data += fmt.Sprintf("%v", arg)
	}

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
