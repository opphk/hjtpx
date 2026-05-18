package database

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

type QueryCache struct {
	mu      sync.RWMutex
	cache   map[string]cacheEntry
	maxSize int
	ttl     time.Duration
	enabled bool
}

type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

var queryCache *QueryCache

func InitQueryCache(cfg *config.Config) {
	queryCache = &QueryCache{
		cache:   make(map[string]cacheEntry),
		maxSize: cfg.Database.QueryOptimization.MaxQueryCacheSize,
		ttl:     time.Duration(cfg.Database.QueryOptimization.QueryCacheTTLSecs) * time.Second,
		enabled: cfg.Database.QueryOptimization.EnableQueryCache,
	}

	go queryCache.startCleanup()

	if queryCache.enabled {
		log.Println("Query cache initialized")
	}
}

func GetQueryCache() *QueryCache {
	return queryCache
}

func (c *QueryCache) generateKey(query string, args ...interface{}) string {
	data := query
	for _, arg := range args {
		argBytes, _ := json.Marshal(arg)
		data += string(argBytes)
	}
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (c *QueryCache) Get(key string) (interface{}, bool) {
	if !c.enabled {
		return nil, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiration) {
		delete(c.cache, key)
		return nil, false
	}

	return entry.value, true
}

func (c *QueryCache) Set(key string, value interface{}) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cache) >= c.maxSize {
		c.evictOldest()
	}

	c.cache[key] = cacheEntry{
		value:      value,
		expiration: time.Now().Add(c.ttl),
	}
}

func (c *QueryCache) evictOldest() {
	oldestKey := ""
	oldestTime := time.Now().Add(24 * time.Hour)

	for k, v := range c.cache {
		if v.expiration.Before(oldestTime) {
			oldestTime = v.expiration
			oldestKey = k
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
	}
}

func (c *QueryCache) startCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanupExpired()
	}
}

func (c *QueryCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, v := range c.cache {
		if now.After(v.expiration) {
			delete(c.cache, k)
		}
	}
}

func (c *QueryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]cacheEntry)
}

func (c *QueryCache) ClearPattern(pattern string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k := range c.cache {
		if len(k) >= len(pattern) && k[:len(pattern)] == pattern {
			delete(c.cache, k)
		}
	}
}

func (c *QueryCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"size":     len(c.cache),
		"max_size": c.maxSize,
		"enabled":  c.enabled,
	}
}

func CachedQuery(keySuffix string, dest interface{}, queryFunc func() error, ttl ...time.Duration) error {
	if !queryCache.enabled {
		return queryFunc()
	}

	key := queryCache.generateKey(keySuffix)
	if cached, ok := queryCache.Get(key); ok {
		if cachedData, err := json.Marshal(cached); err == nil {
			json.Unmarshal(cachedData, dest)
			return nil
		}
	}

	if err := queryFunc(); err != nil {
		return err
	}

	queryCache.Set(key, dest)
	return nil
}

func InvalidateQueryCache(tableName string) {
	if queryCache == nil {
		return
	}
	queryCache.ClearPattern(fmt.Sprintf("table:%s:", tableName))
}
