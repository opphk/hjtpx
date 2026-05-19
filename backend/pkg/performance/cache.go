package performance

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type CacheOptimizer struct {
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	isRunning   bool
	localCache  *LocalCache
	redisCache  *RedisCache
	stats       *CacheStats
	prefetcher  *Prefetcher
}

type CacheStats struct {
	TotalRequests      atomic.Int64
	LocalHits          atomic.Int64
	LocalMisses        atomic.Int64
	RedisHits          atomic.Int64
	RedisMisses        atomic.Int64
	TotalHits          atomic.Int64
	TotalMisses        atomic.Int64
	Evictions          atomic.Int64
	MemoryUsed         atomic.Int64
	AverageLatency     atomic.Int64
	HotKeys            atomic.Int64
}

type LocalCache struct {
	mu         sync.RWMutex
	cache      map[string]*CacheItem
	maxSize    int
	ttl        time.Duration
	hits       atomic.Int64
	misses     atomic.Int64
}

type CacheItem struct {
	key        string
	value      []byte
	expiresAt  time.Time
	accessCount int64
	size       int
}

type RedisCache struct {
	ttl       time.Duration
	hits      atomic.Int64
	misses    atomic.Int64
}

type Prefetcher struct {
	mu           sync.RWMutex
	hotKeys      map[string]int64
	threshold    int64
	prefetchFunc func(key string) ([]byte, error)
}

func NewCacheOptimizer() *CacheOptimizer {
	ctx, cancel := context.WithCancel(context.Background())
	return &CacheOptimizer{
		ctx:        ctx,
		cancel:     cancel,
		localCache: NewLocalCache(100000, 5*time.Minute),
		redisCache: NewRedisCache(100, 30*time.Minute),
		stats:      &CacheStats{},
		prefetcher: NewPrefetcher(100, nil),
	}
}

func NewLocalCache(maxSize int, ttl time.Duration) *LocalCache {
	return &LocalCache{
		cache:    make(map[string]*CacheItem, maxSize),
		maxSize:  maxSize,
		ttl:      ttl,
	}
}

func NewRedisCache(poolSize int, ttl time.Duration) *RedisCache {
	return &RedisCache{
		ttl: ttl,
	}
}

func NewPrefetcher(threshold int64, prefetchFunc func(key string) ([]byte, error)) *Prefetcher {
	return &Prefetcher{
		hotKeys:      make(map[string]int64),
		threshold:    threshold,
		prefetchFunc: prefetchFunc,
	}
}

func (c *CacheOptimizer) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning {
		return nil
	}

	c.isRunning = true

	go c.cleanupLoop()
	go c.monitorHotKeys()
	go c.prefetchLoop()

	log.Println("[CacheOptimizer] Started successfully")
	return nil
}

func (c *CacheOptimizer) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning {
		return
	}

	c.cancel()
	c.isRunning = false

	log.Println("[CacheOptimizer] Stopped")
}

func (c *CacheOptimizer) Get(key string) ([]byte, error) {
	c.stats.TotalRequests.Add(1)

	if data, ok := c.localCache.Get(key); ok {
		c.stats.LocalHits.Add(1)
		c.stats.TotalHits.Add(1)
		c.trackHotKey(key)
		return data, nil
	}

	c.stats.LocalMisses.Add(1)

	if data, err := c.redisCache.Get(key); err == nil && data != nil {
		c.stats.RedisHits.Add(1)
		c.stats.TotalHits.Add(1)
		c.localCache.Set(key, data)
		c.trackHotKey(key)
		return data, nil
	}

	c.stats.RedisMisses.Add(1)
	c.stats.TotalMisses.Add(1)
	return nil, nil
}

func (c *CacheOptimizer) Set(key string, value []byte) error {
	c.localCache.Set(key, value)
	return c.redisCache.Set(key, value)
}

func (c *CacheOptimizer) Delete(key string) error {
	c.localCache.Delete(key)
	return c.redisCache.Delete(key)
}

func (c *CacheOptimizer) GetMulti(keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	missingKeys := make([]string, 0, len(keys))

	for _, key := range keys {
		if data, ok := c.localCache.Get(key); ok {
			result[key] = data
			c.stats.LocalHits.Add(1)
			c.stats.TotalHits.Add(1)
			c.trackHotKey(key)
		} else {
			missingKeys = append(missingKeys, key)
			c.stats.LocalMisses.Add(1)
		}
	}

	if len(missingKeys) > 0 {
		redisData, err := c.redisCache.GetMulti(missingKeys)
		if err == nil {
			for k, v := range redisData {
				result[k] = v
				c.localCache.Set(k, v)
				c.stats.RedisHits.Add(1)
				c.stats.TotalHits.Add(1)
				c.trackHotKey(k)
			}
		} else {
			c.stats.RedisMisses.Add(int64(len(missingKeys)))
			c.stats.TotalMisses.Add(int64(len(missingKeys)))
		}
	}

	return result, nil
}

func (c *CacheOptimizer) SetMulti(items map[string][]byte) error {
	for k, v := range items {
		c.localCache.Set(k, v)
	}
	return c.redisCache.SetMulti(items)
}

func (c *CacheOptimizer) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.localCache.Cleanup()
		}
	}
}

func (c *CacheOptimizer) monitorHotKeys() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.prefetcher.Cleanup()
		}
	}
}

func (c *CacheOptimizer) prefetchLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.prefetchHotKeys()
		}
	}
}

func (c *CacheOptimizer) trackHotKey(key string) {
	c.prefetcher.RecordAccess(key)
}

func (c *CacheOptimizer) prefetchHotKeys() {
	hotKeys := c.prefetcher.GetHotKeys()
	for _, key := range hotKeys {
		if c.prefetcher.prefetchFunc != nil {
			if data, err := c.prefetcher.prefetchFunc(key); err == nil && data != nil {
				c.localCache.Set(key, data)
			}
		}
	}
}

func (l *LocalCache) Get(key string) ([]byte, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	item, exists := l.cache[key]
	if !exists {
		l.misses.Add(1)
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		delete(l.cache, key)
		l.misses.Add(1)
		return nil, false
	}

	item.accessCount++
	l.hits.Add(1)
	return item.value, true
}

func (l *LocalCache) Set(key string, value []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.cache) >= l.maxSize {
		l.evict()
	}

	item := &CacheItem{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(l.ttl),
		size:      len(value),
	}

	l.cache[key] = item
}

func (l *LocalCache) Delete(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.cache, key)
}

func (l *LocalCache) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for k, v := range l.cache {
		if now.After(v.expiresAt) {
			delete(l.cache, k)
		}
	}
}

func (l *LocalCache) evict() {
	var oldestKey string
	var oldestTime time.Time
	var minAccess int64 = 1<<63 - 1

	for k, v := range l.cache {
		if v.accessCount < minAccess || (v.accessCount == minAccess && v.expiresAt.Before(oldestTime)) {
			oldestKey = k
			oldestTime = v.expiresAt
			minAccess = v.accessCount
		}
	}

	if oldestKey != "" {
		delete(l.cache, oldestKey)
	}
}

func (r *RedisCache) Get(key string) ([]byte, error) {
	if redis.Client == nil {
		return nil, nil
	}

	data, err := redis.Client.Get(redis.GetContext(), key).Bytes()
	if err != nil {
		r.misses.Add(1)
		return nil, err
	}

	r.hits.Add(1)
	return data, nil
}

func (r *RedisCache) Set(key string, value []byte) error {
	if redis.Client == nil {
		return nil
	}

	return redis.Client.Set(redis.GetContext(), key, value, r.ttl).Err()
}

func (r *RedisCache) Delete(key string) error {
	if redis.Client == nil {
		return nil
	}

	return redis.Client.Del(redis.GetContext(), key).Err()
}

func (r *RedisCache) GetMulti(keys []string) (map[string][]byte, error) {
	if redis.Client == nil || len(keys) == 0 {
		return nil, nil
	}

	result := make(map[string][]byte)
	pipe := redis.Client.Pipeline()
	cmds := make(map[string]*goredis.StringCmd)

	for _, key := range keys {
		cmds[key] = pipe.Get(redis.GetContext(), key)
	}

	_, err := pipe.Exec(redis.GetContext())
	if err != nil && err != goredis.Nil {
		return nil, err
	}

	for key, cmd := range cmds {
		if data, err := cmd.Bytes(); err == nil {
			result[key] = data
			r.hits.Add(1)
		} else {
			r.misses.Add(1)
		}
	}

	return result, nil
}

func (r *RedisCache) SetMulti(items map[string][]byte) error {
	if redis.Client == nil || len(items) == 0 {
		return nil
	}

	pipe := redis.Client.Pipeline()
	for k, v := range items {
		pipe.Set(redis.GetContext(), k, v, r.ttl)
	}

	_, err := pipe.Exec(redis.GetContext())
	return err
}

func (p *Prefetcher) RecordAccess(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.hotKeys[key]++
}

func (p *Prefetcher) GetHotKeys() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []string
	for k, v := range p.hotKeys {
		if v >= p.threshold {
			result = append(result, k)
		}
	}
	return result
}

func (p *Prefetcher) Cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for k := range p.hotKeys {
		p.hotKeys[k] = p.hotKeys[k] * 9 / 10
		if p.hotKeys[k] < p.threshold/2 {
			delete(p.hotKeys, k)
		}
	}
}

func (c *CacheOptimizer) GetStats() map[string]interface{} {
	total := c.stats.TotalHits.Load() + c.stats.TotalMisses.Load()
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(c.stats.TotalHits.Load()) / float64(total) * 100
	}

	return map[string]interface{}{
		"total_requests": c.stats.TotalRequests.Load(),
		"local_hits":     c.stats.LocalHits.Load(),
		"local_misses":   c.stats.LocalMisses.Load(),
		"redis_hits":     c.stats.RedisHits.Load(),
		"redis_misses":   c.stats.RedisMisses.Load(),
		"total_hits":     c.stats.TotalHits.Load(),
		"total_misses":   c.stats.TotalMisses.Load(),
		"hit_rate":       hitRate,
		"hot_keys":       c.stats.HotKeys.Load(),
	}
}

func (c *CacheOptimizer) SetPrefetchFunc(fn func(key string) ([]byte, error)) {
	c.prefetcher.prefetchFunc = fn
}
