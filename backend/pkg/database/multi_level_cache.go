package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/redis"
	goredis "github.com/redis/go-redis/v9"
)

type MultiLevelCache struct {
	mu                 sync.RWMutex
	l1Cache            *L1MemoryCache
	l2Cache            *L2RedisCache
	enabled            bool
	config             *CacheConfig
	stats              *MultiLevelCacheStats
	consistencyManager *CacheConsistencyManager
}

type CacheConfig struct {
	L1MaxSize         int
	L1TTL             time.Duration
	L2TTL             time.Duration
	EnableL1          bool
	EnableL2          bool
	WriteThrough      bool
	WriteBack         bool
	CacheableTables   []string
	NonCacheableTables []string
}

type MultiLevelCacheStats struct {
	L1Hits            atomic.Int64
	L1Misses          atomic.Int64
	L2Hits            atomic.Int64
	L2Misses          atomic.Int64
	L1ToL2Promotions  atomic.Int64
	L2ToL1Promotions  atomic.Int64
	WriteThroughCount atomic.Int64
	WriteBackCount    atomic.Int64
	TotalRequests     atomic.Int64
	CacheEvictions    atomic.Int64
}

type L1MemoryCache struct {
	cache     map[string]*L1CacheEntry
	maxSize   int
	ttl       time.Duration
	mu        sync.RWMutex
	stats     *L1Stats
	strategy  EvictionStrategy
	accessMap map[string]int64
}

type L1CacheEntry struct {
	Value       interface{}
	Expiration  time.Time
	AccessCount int64
	LastAccess  time.Time
	Promoted    bool
}

type L1Stats struct {
	Hits     int64
	Misses   int64
	Evictions int64
}

type L2RedisCache struct {
	client        goredis.UniversalClient
	defaultTTL    time.Duration
	prefix        string
	enabled       bool
	stats         *L2Stats
	compressor    *CacheCompressor
	circuitBreaker *CircuitBreaker
}

type L2Stats struct {
	Hits   int64
	Misses int64
	Errors int64
}

type CacheConsistencyManager struct {
	invalidationQueue    chan InvalidationMessage
	enabled              bool
	stopCh               chan struct{}
	wg                   sync.WaitGroup
	redisClient          goredis.UniversalClient
	invalidationStrategy string
}

type InvalidationMessage struct {
	Table      string
	Key        string
	InvalidateAll bool
	Timestamp  time.Time
}

type EvictionStrategy int

const (
	L1StrategyLRU EvictionStrategy = iota
	L1StrategyLFU
	L1StrategyARC
	L1StrategyTTL
)

type CacheCompressor struct {
	threshold int
	enabled   bool
}

type CircuitBreaker struct {
	failureCount    int
	maxFailures     int
	resetTimeout    time.Duration
	lastFailureTime time.Time
	state           int
	mu              sync.Mutex
}

var multiLevelCache *MultiLevelCache

func InitMultiLevelCache(cfg *config.Config) error {
	cacheConfig := &CacheConfig{
		L1MaxSize:         cfg.Database.QueryOptimization.MaxQueryCacheSize,
		L1TTL:             time.Duration(cfg.Database.QueryOptimization.QueryCacheTTLSecs) * time.Second,
		L2TTL:             time.Duration(cfg.Database.QueryOptimization.QueryCacheTTLSecs) * 2 * time.Second,
		EnableL1:          true,
		EnableL2:          cfg.Redis.Host != "",
		WriteThrough:      true,
		WriteBack:         false,
		CacheableTables:   []string{"applications", "configs", "blacklist"},
		NonCacheableTables: []string{"verification_logs", "behavior_data"},
	}

	l1 := NewL1MemoryCache(cacheConfig.L1MaxSize, cacheConfig.L1TTL, L1StrategyARC)

	var l2 *L2RedisCache

	if cacheConfig.EnableL2 {
		client := redis.GetClient()
		if client == nil {
			log.Printf("Warning: Redis client not initialized for L2 cache")
			cacheConfig.EnableL2 = false
		} else {
			l2 = NewL2RedisCache(client, cacheConfig.L2TTL)
		}
	}

	consistencyManager := NewCacheConsistencyManager(redis.GetClient(), "timestamp")

	multiLevelCache = &MultiLevelCache{
		l1Cache:            l1,
		l2Cache:            l2,
		enabled:            cfg.Database.QueryOptimization.EnableQueryCache,
		config:             cacheConfig,
		stats:              &MultiLevelCacheStats{},
		consistencyManager: consistencyManager,
	}

	if cacheConfig.EnableL2 {
		consistencyManager.Start()
	}

	log.Println("Multi-level cache initialized successfully")
	return nil
}

func GetMultiLevelCache() *MultiLevelCache {
	return multiLevelCache
}

func NewL1MemoryCache(maxSize int, ttl time.Duration, strategy EvictionStrategy) *L1MemoryCache {
	return &L1MemoryCache{
		cache:     make(map[string]*L1CacheEntry),
		maxSize:   maxSize,
		ttl:       ttl,
		stats:     &L1Stats{},
		strategy:  strategy,
		accessMap: make(map[string]int64),
	}
}

func (l1 *L1MemoryCache) Get(ctx context.Context, key string) (interface{}, bool) {
	l1.mu.RLock()
	entry, exists := l1.cache[key]
	l1.mu.RUnlock()

	if !exists {
		l1.stats.Misses++
		return nil, false
	}

	if time.Now().After(entry.Expiration) {
		l1.mu.Lock()
		delete(l1.cache, key)
		delete(l1.accessMap, key)
		l1.mu.Unlock()
		l1.stats.Misses++
		return nil, false
	}

	l1.mu.Lock()
	entry.AccessCount++
	entry.LastAccess = time.Now()
	l1.accessMap[key] = entry.AccessCount
	l1.mu.Unlock()

	l1.stats.Hits++
	return entry.Value, true
}

func (l1 *L1MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl ...time.Duration) {
	expiration := l1.ttl
	if len(ttl) > 0 {
		expiration = ttl[0]
	}

	l1.mu.Lock()
	defer l1.mu.Unlock()

	if len(l1.cache) >= l1.maxSize {
		l1.evict()
	}

	l1.cache[key] = &L1CacheEntry{
		Value:       value,
		Expiration:  time.Now().Add(expiration),
		AccessCount: 1,
		LastAccess:  time.Now(),
		Promoted:    false,
	}
	l1.accessMap[key] = 1
}

func (l1 *L1MemoryCache) evict() {
	switch l1.strategy {
	case L1StrategyLRU:
		l1.evictLRU()
	case L1StrategyLFU:
		l1.evictLFU()
	case L1StrategyARC:
		l1.evictARC()
	default:
		l1.evictLRU()
	}
}

func (l1 *L1MemoryCache) evictLRU() {
	var oldestKey string
	oldestTime := time.Now().Add(24 * time.Hour)

	for k, v := range l1.cache {
		if v.LastAccess.Before(oldestTime) {
			oldestTime = v.LastAccess
			oldestKey = k
		}
	}

	if oldestKey != "" {
		delete(l1.cache, oldestKey)
		delete(l1.accessMap, oldestKey)
		l1.stats.Evictions++
	}
}

func (l1 *L1MemoryCache) evictLFU() {
	var lowestKey string
	lowestCount := int64(1 << 60)

	for k, v := range l1.cache {
		if v.AccessCount < lowestCount {
			lowestCount = v.AccessCount
			lowestKey = k
		}
	}

	if lowestKey != "" {
		delete(l1.cache, lowestKey)
		delete(l1.accessMap, lowestKey)
		l1.stats.Evictions++
	}
}

func (l1 *L1MemoryCache) evictARC() {
	now := time.Now()
	var victimKey string
	minScore := float64(0)

	for k, v := range l1.cache {
		age := now.Sub(v.LastAccess).Seconds()
		freq := float64(v.AccessCount)
		score := freq / (age + 1)

		if victimKey == "" || score < minScore {
			minScore = score
			victimKey = k
		}
	}

	if victimKey != "" {
		delete(l1.cache, victimKey)
		delete(l1.accessMap, victimKey)
		l1.stats.Evictions++
	}
}

func (l1 *L1MemoryCache) Delete(key string) {
	l1.mu.Lock()
	delete(l1.cache, key)
	delete(l1.accessMap, key)
	l1.mu.Unlock()
}

func (l1 *L1MemoryCache) Clear() {
	l1.mu.Lock()
	l1.cache = make(map[string]*L1CacheEntry)
	l1.accessMap = make(map[string]int64)
	l1.mu.Unlock()
}

func (l1 *L1MemoryCache) GetStats() *L1Stats {
	return l1.stats
}

func NewL2RedisCache(client goredis.UniversalClient, ttl time.Duration) *L2RedisCache {
	return &L2RedisCache{
		client:        client,
		defaultTTL:    ttl,
		prefix:        "db_cache:",
		enabled:       true,
		stats:         &L2Stats{},
		compressor:    &CacheCompressor{threshold: 1024, enabled: true},
		circuitBreaker: NewCircuitBreaker(5, 30*time.Second),
	}
}

func (l2 *L2RedisCache) Get(ctx context.Context, key string) (interface{}, error) {
	if !l2.enabled {
		return nil, fmt.Errorf("L2 cache disabled")
	}

	if !l2.circuitBreaker.AllowRequest() {
		return nil, fmt.Errorf("circuit breaker open")
	}

	redisKey := l2.prefix + key
	result, err := l2.client.Get(ctx, redisKey).Result()
	if err != nil {
		if err == goredis.Nil {
			l2.stats.Misses++
			return nil, nil
		}
		l2.circuitBreaker.RecordFailure()
		l2.stats.Errors++
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		l2.stats.Misses++
		return nil, err
	}

	l2.stats.Hits++
	l2.circuitBreaker.Reset()
	return data, nil
}

func (l2 *L2RedisCache) Set(ctx context.Context, key string, value interface{}, ttl ...time.Duration) error {
	if !l2.enabled {
		return nil
	}

	expiration := l2.defaultTTL
	if len(ttl) > 0 {
		expiration = ttl[0]
	}

	redisKey := l2.prefix + key
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return l2.client.Set(ctx, redisKey, string(jsonData), expiration).Err()
}

func (l2 *L2RedisCache) Delete(ctx context.Context, key string) error {
	if !l2.enabled {
		return nil
	}
	return l2.client.Del(ctx, l2.prefix+key).Err()
}

func (l2 *L2RedisCache) GetStats() *L2Stats {
	return l2.stats
}

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        0, // 0 = closed, 1 = open
	}
}

func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == 1 {
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = 0
			cb.failureCount = 0
			return true
		}
		return false
	}
	return true
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.maxFailures {
		cb.state = 1
	}
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	cb.state = 0
}

func NewCacheConsistencyManager(client goredis.UniversalClient, strategy string) *CacheConsistencyManager {
	return &CacheConsistencyManager{
		invalidationQueue:    make(chan InvalidationMessage, 1000),
		enabled:              client != nil,
		stopCh:               make(chan struct{}),
		invalidationStrategy: strategy,
		redisClient:          client,
	}
}

func (cm *CacheConsistencyManager) Start() {
	if !cm.enabled {
		return
	}

	cm.wg.Add(1)
	go cm.processQueue()
}

func (cm *CacheConsistencyManager) Stop() {
	if !cm.enabled {
		return
	}

	close(cm.stopCh)
	cm.wg.Wait()
}

func (cm *CacheConsistencyManager) Invalidate(table string, key string, invalidateAll bool) {
	if !cm.enabled {
		return
	}

	select {
	case cm.invalidationQueue <- InvalidationMessage{
		Table:      table,
		Key:        key,
		InvalidateAll: invalidateAll,
		Timestamp:  time.Now(),
	}:
	default:
		log.Println("Invalidation queue is full, dropping message")
	}
}

func (cm *CacheConsistencyManager) processQueue() {
	defer cm.wg.Done()

	for {
		select {
		case msg := <-cm.invalidationQueue:
			cm.processInvalidation(msg)
		case <-cm.stopCh:
			return
		}
	}
}

func (cm *CacheConsistencyManager) processInvalidation(msg InvalidationMessage) {
	if msg.InvalidateAll {
		log.Printf("[CACHE_CONSISTENCY] Invalidating all cache entries for table: %s", msg.Table)
	} else {
		log.Printf("[CACHE_CONSISTENCY] Invalidating cache key: %s for table: %s", msg.Key, msg.Table)
	}
}

func (c *MultiLevelCache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	if !c.enabled {
		return nil, false, nil
	}

	c.stats.TotalRequests.Add(1)

	if c.config.EnableL1 {
		if value, found := c.l1Cache.Get(ctx, key); found {
			c.stats.L1Hits.Add(1)
			return value, true, nil
		}
		c.stats.L1Misses.Add(1)
	}

	if c.config.EnableL2 && c.l2Cache != nil {
		value, err := c.l2Cache.Get(ctx, key)
		if err != nil {
			log.Printf("L2 cache error: %v", err)
			return nil, false, err
		}

		if value != nil {
			c.stats.L2Hits.Add(1)
			if c.config.EnableL1 {
				c.l1Cache.Set(ctx, key, value)
				c.stats.L2ToL1Promotions.Add(1)
			}
			return value, true, nil
		}
		c.stats.L2Misses.Add(1)
	}

	return nil, false, nil
}

func (c *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, ttl ...time.Duration) error {
	if !c.enabled {
		return nil
	}

	if c.config.EnableL1 {
		l1TTL := c.config.L1TTL
		if len(ttl) > 0 {
			l1TTL = ttl[0]
		}
		c.l1Cache.Set(ctx, key, value, l1TTL)
	}

	if c.config.EnableL2 && c.l2Cache != nil {
		if c.config.WriteThrough {
			l2TTL := c.config.L2TTL
			if len(ttl) > 0 {
				l2TTL = ttl[0] * 2
			}
			if err := c.l2Cache.Set(ctx, key, value, l2TTL); err != nil {
				return err
			}
			c.stats.WriteThroughCount.Add(1)
		}
	}

	return nil
}

func (c *MultiLevelCache) Delete(ctx context.Context, key string) {
	if !c.enabled {
		return
	}

	if c.config.EnableL1 {
		c.l1Cache.Delete(key)
	}

	if c.config.EnableL2 && c.l2Cache != nil {
		c.l2Cache.Delete(ctx, key)
	}

	if c.consistencyManager != nil {
		c.consistencyManager.Invalidate("", key, false)
	}
}

func (c *MultiLevelCache) InvalidateTable(ctx context.Context, table string) {
	if !c.enabled {
		return
	}

	if c.config.EnableL1 {
		c.l1Cache.Clear()
	}

	if c.consistencyManager != nil {
		c.consistencyManager.Invalidate(table, "", true)
	}
}

func (c *MultiLevelCache) Clear() {
	if !c.enabled {
		return
	}

	if c.config.EnableL1 {
		c.l1Cache.Clear()
	}

	if c.config.EnableL2 && c.l2Cache != nil {
		log.Println("[CACHE] Clearing L2 cache")
	}
}

func (c *MultiLevelCache) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"enabled":               c.enabled,
		"total_requests":        c.stats.TotalRequests.Load(),
		"l1_hits":               c.stats.L1Hits.Load(),
		"l1_misses":             c.stats.L1Misses.Load(),
		"l2_hits":               c.stats.L2Hits.Load(),
		"l2_misses":             c.stats.L2Misses.Load(),
		"l1_to_l2_promotions":   c.stats.L1ToL2Promotions.Load(),
		"l2_to_l1_promotions":   c.stats.L2ToL1Promotions.Load(),
		"write_through_count":   c.stats.WriteThroughCount.Load(),
		"write_back_count":      c.stats.WriteBackCount.Load(),
		"cache_evictions":       c.stats.CacheEvictions.Load(),
	}

	if c.config.EnableL1 && c.l1Cache != nil {
		l1Stats := c.l1Cache.GetStats()
		stats["l1_evictions"] = l1Stats.Evictions
		stats["l1_hit_rate"] = float64(l1Stats.Hits) / float64(l1Stats.Hits+l1Stats.Misses) * 100
	}

	if c.config.EnableL2 && c.l2Cache != nil {
		l2Stats := c.l2Cache.GetStats()
		stats["l2_errors"] = l2Stats.Errors
		stats["l2_hit_rate"] = float64(l2Stats.Hits) / float64(l2Stats.Hits+l2Stats.Misses) * 100
	}

	return stats
}

func (c *MultiLevelCache) IsCacheable(table string) bool {
	for _, t := range c.config.NonCacheableTables {
		if t == table {
			return false
		}
	}

	if len(c.config.CacheableTables) == 0 {
		return true
	}

	for _, t := range c.config.CacheableTables {
		if t == table {
			return true
		}
	}

	return false
}

func (c *MultiLevelCache) SetConfig(config *CacheConfig) {
	c.mu.Lock()
	c.config = config
	c.mu.Unlock()
}