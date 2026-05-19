package service

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"
)

type OptimizedLocalCache struct {
	maxEntries  int
	maxSize     int
	defaultTTL  time.Duration
	entries     map[string]cacheEntry
	totalSize   int
	mu          sync.RWMutex
}

type cacheEntry struct {
	value     []byte
	expireAt  time.Time
	size      int
}

type MultiLevelCacheConfig struct {
	Enabled        bool
	LocalMaxEntries int
	LocalMaxSize   int
	LocalTTL       time.Duration
	RemoteEnabled  bool
	RemoteTTL      time.Duration
}

type MultiLevelCacheService struct {
	config     MultiLevelCacheConfig
	localCache *OptimizedLocalCache
}

type PromotionPolicy struct {
	accessCounts map[string]int
	lastAccess   map[string]time.Time
	mu           sync.RWMutex
}

var DefaultMultiLevelConfig = MultiLevelCacheConfig{
	Enabled:        true,
	LocalMaxEntries: 1000,
	LocalMaxSize:   10 * 1024 * 1024,
	LocalTTL:       5 * time.Minute,
	RemoteEnabled:  false,
	RemoteTTL:      30 * time.Minute,
}

func NewOptimizedLocalCache(maxEntries, maxSize int, defaultTTL time.Duration) *OptimizedLocalCache {
	return &OptimizedLocalCache{
		maxEntries: maxEntries,
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
		entries:    make(map[string]cacheEntry),
	}
}

func (c *OptimizedLocalCache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	expireAt := time.Now().Add(ttl)
	if ttl == 0 {
		expireAt = time.Now().Add(c.defaultTTL)
	}
	
	entrySize := len(value)
	
	if c.totalSize+entrySize > c.maxSize && c.totalSize > 0 {
		c.evict()
	}
	
	if len(c.entries) >= c.maxEntries {
		c.evictOldest()
	}
	
	c.entries[key] = cacheEntry{
		value:    value,
		expireAt: expireAt,
		size:     entrySize,
	}
	c.totalSize += entrySize
}

func (c *OptimizedLocalCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	
	if time.Now().After(entry.expireAt) {
		return nil, false
	}
	
	return entry.value, true
}

func (c *OptimizedLocalCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if entry, ok := c.entries[key]; ok {
		c.totalSize -= entry.size
		delete(c.entries, key)
	}
}

func (c *OptimizedLocalCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.entries = make(map[string]cacheEntry)
	c.totalSize = 0
}

func (c *OptimizedLocalCache) evict() {
	threshold := c.maxSize * 80 / 100
	for c.totalSize > threshold {
		c.evictOldest()
	}
}

func (c *OptimizedLocalCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range c.entries {
		if oldestKey == "" || entry.expireAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expireAt
		}
	}
	
	if oldestKey != "" {
		c.totalSize -= c.entries[oldestKey].size
		delete(c.entries, oldestKey)
	}
}

func NewMultiLevelCacheService(config MultiLevelCacheConfig) *MultiLevelCacheService {
	return &MultiLevelCacheService{
		config: config,
		localCache: NewOptimizedLocalCache(
			config.LocalMaxEntries,
			config.LocalMaxSize,
			config.LocalTTL,
		),
	}
}

func (c *MultiLevelCacheService) IsEnabled() bool {
	return c.config.Enabled
}

func (c *MultiLevelCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.localCache.Set(key, value, ttl)
	return nil
}

func (c *MultiLevelCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	value, ok := c.localCache.Get(key)
	if !ok {
		return nil, errors.New("key not found")
	}
	return value, nil
}

func (c *MultiLevelCacheService) Delete(ctx context.Context, key string) error {
	c.localCache.Delete(key)
	return nil
}

func NewPromotionPolicy() *PromotionPolicy {
	return &PromotionPolicy{
		accessCounts: make(map[string]int),
		lastAccess:   make(map[string]time.Time),
	}
}

func (p *PromotionPolicy) RecordAccess(key string, duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.accessCounts[key]++
	p.lastAccess[key] = time.Now()
}

func (p *PromotionPolicy) GetHotKeys(limit int) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	keys := make([]string, 0, len(p.accessCounts))
	for key := range p.accessCounts {
		keys = append(keys, key)
	}
	
	return keys[:int(math.Min(float64(len(keys)), float64(limit)))]
}


