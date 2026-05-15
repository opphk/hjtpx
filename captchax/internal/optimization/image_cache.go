package optimization

import (
	"sync"
	"time"
)

type ImageCache struct {
	cache   map[string]*CacheItem
	mu      sync.RWMutex
	maxSize int
	ttl     time.Duration
}

type CacheItem struct {
	Data      []byte
	ExpiresAt time.Time
}

func NewImageCache(maxSize int, ttl time.Duration) *ImageCache {
	cache := &ImageCache{
		cache:   make(map[string]*CacheItem),
		maxSize: maxSize,
		ttl:     ttl,
	}
	go cache.cleanup()
	return cache
}

func (c *ImageCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.cache[key]
	if !exists || time.Now().After(item.ExpiresAt) {
		return nil, false
	}
	return item.Data, true
}

func (c *ImageCache) Set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cache) >= c.maxSize {
		c.evict()
	}

	c.cache[key] = &CacheItem{
		Data:      data,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

func (c *ImageCache) evict() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range c.cache {
		if oldestTime.IsZero() || item.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.ExpiresAt
		}
	}

	delete(c.cache, oldestKey)
}

func (c *ImageCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.cache {
			if now.After(item.ExpiresAt) {
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}

func (c *ImageCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

func (c *ImageCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*CacheItem)
}
