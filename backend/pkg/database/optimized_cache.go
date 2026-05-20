package database

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

type OptimizedQueryCache struct {
	mu           sync.RWMutex
	cache        map[string]*CacheItem
	maxSize      int
	baseTTL      time.Duration
	enabled      bool
	stats        *CacheStatistics
}

type CacheItem struct {
	Value        interface{}
	CreatedAt    time.Time
	LastAccessed time.Time
	AccessCount  int64
	TTL          time.Duration
	Priority     int
}

type CacheStatistics struct {
	TotalHits      int64
	TotalMisses    int64
	TotalSets      int64
	TotalEvictions int64
	CurrentSize    int64
	HitRate        float64
}

var optimizedCache *OptimizedQueryCache

func NewOptimizedQueryCache(cfg *config.Config) *OptimizedQueryCache {
	baseTTL := time.Duration(cfg.Database.QueryOptimization.QueryCacheTTLSecs) * time.Second
	if baseTTL <= 0 {
		baseTTL = 5 * time.Minute
	}

	maxSize := cfg.Database.QueryOptimization.MaxQueryCacheSize
	if maxSize <= 0 {
		maxSize = 10000
	}

	return &OptimizedQueryCache{
		cache:    make(map[string]*CacheItem),
		maxSize:  maxSize,
		baseTTL:  baseTTL,
		enabled:  cfg.Database.QueryOptimization.EnableQueryCache,
		stats:   &CacheStatistics{},
	}
}

func (c *OptimizedQueryCache) Get(key string) (interface{}, bool) {
	if !c.enabled {
		return nil, false
	}

	c.mu.RLock()
	item, exists := c.cache[key]
	c.mu.RUnlock()

	if !exists {
		c.stats.TotalMisses++
		return nil, false
	}

	if time.Now().After(item.CreatedAt.Add(item.TTL)) {
		c.Delete(key)
		c.stats.TotalMisses++
		return nil, false
	}

	item.LastAccessed = time.Now()
	atomic.AddInt64(&item.AccessCount, 1)
	c.stats.TotalHits++
	c.updateHitRate()

	return item.Value, true
}

func (c *OptimizedQueryCache) Set(key string, value interface{}, ttl ...time.Duration) {
	if !c.enabled {
		return
	}

	calculatedTTL := c.baseTTL
	if len(ttl) > 0 && ttl[0] > 0 {
		calculatedTTL = ttl[0]
	}

	item := &CacheItem{
		Value:        value,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		AccessCount:  1,
		TTL:          calculatedTTL,
		Priority:     1,
	}

	c.mu.Lock()

	if len(c.cache) >= c.maxSize {
		c.evictLeastRecentlyUsed()
	}

	c.cache[key] = item
	c.stats.TotalSets++
	c.stats.CurrentSize++

	c.mu.Unlock()
}

func (c *OptimizedQueryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.cache[key]; exists {
		delete(c.cache, key)
		c.stats.CurrentSize--
		c.stats.TotalEvictions++
	}
}

func (c *OptimizedQueryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CacheItem)
	c.stats.CurrentSize = 0
}

func (c *OptimizedQueryCache) ClearPattern(pattern string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.cache {
		if len(key) >= len(pattern) && key[:len(pattern)] == pattern {
			delete(c.cache, key)
			c.stats.CurrentSize--
			c.stats.TotalEvictions++
		}
	}
}

func (c *OptimizedQueryCache) evictLeastRecentlyUsed() {
	oldestKey := ""
	oldestTime := time.Now().Add(24 * time.Hour)

	for key, item := range c.cache {
		if item.LastAccessed.Before(oldestTime) {
			oldestTime = item.LastAccessed
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
		c.stats.CurrentSize--
		c.stats.TotalEvictions++
	}
}

func (c *OptimizedQueryCache) updateHitRate() {
	total := c.stats.TotalHits + c.stats.TotalMisses
	if total > 0 {
		c.stats.HitRate = float64(c.stats.TotalHits) / float64(total) * 100
	}
}

func (c *OptimizedQueryCache) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_hits":      c.stats.TotalHits,
		"total_misses":    c.stats.TotalMisses,
		"total_sets":      c.stats.TotalSets,
		"current_size":    c.stats.CurrentSize,
		"hit_rate":        fmt.Sprintf("%.2f%%", c.stats.HitRate),
		"enabled":         c.enabled,
	}
}

func (c *OptimizedQueryCache) GenerateKey(query string, args ...interface{}) string {
	data := query
	for _, arg := range args {
		argBytes, _ := json.Marshal(arg)
		data += string(argBytes)
	}
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (c *OptimizedQueryCache) SetEnabled(enabled bool) {
	c.enabled = enabled
}

func (c *OptimizedQueryCache) SetBaseTTL(ttl time.Duration) {
	c.baseTTL = ttl
}

func (c *OptimizedQueryCache) GetTopQueries(limit int) []CacheItem {
	c.mu.RLock()
	defer c.mu.RUnlock()

	items := make([]CacheItem, 0, len(c.cache))
	for _, item := range c.cache {
		items = append(items, CacheItem{
			AccessCount:  atomic.LoadInt64(&item.AccessCount),
			LastAccessed: item.LastAccessed,
			CreatedAt:    item.CreatedAt,
		})
	}

	if limit > 0 && len(items) > limit {
		for i := 0; i < len(items)-1; i++ {
			for j := i + 1; j < len(items); j++ {
				if items[j].AccessCount > items[i].AccessCount {
					items[i], items[j] = items[j], items[i]
				}
			}
		}
		return items[:limit]
	}

	return items
}

func InitOptimizedQueryCache(cfg *config.Config) {
	optimizedCache = NewOptimizedQueryCache(cfg)
}

func GetOptimizedQueryCache() *OptimizedQueryCache {
	return optimizedCache
}
