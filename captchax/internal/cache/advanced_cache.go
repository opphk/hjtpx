package cache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type RedisWrapper struct {
	client RedisClient
}

type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Keys(ctx context.Context, pattern string) ([]string, error)
	Close() error
}

func NewRedisWrapper(client RedisClient) *RedisWrapper {
	return &RedisWrapper{client: client}
}

func (w *RedisWrapper) Get(ctx context.Context, key string) (string, error) {
	if w.client == nil {
		return "", fmt.Errorf("redis client not initialized")
	}
	return w.client.Get(ctx, key)
}

func (w *RedisWrapper) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if w.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return w.client.Set(ctx, key, value, ttl)
}

func (w *RedisWrapper) Del(ctx context.Context, keys ...string) error {
	if w.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return w.client.Del(ctx, keys...)
}

func (w *RedisWrapper) DelPattern(ctx context.Context, pattern string) error {
	if w.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	keys, err := w.client.Keys(ctx, pattern)
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return w.client.Del(ctx, keys...)
	}
	return nil
}

func (w *RedisWrapper) Close() error {
	if w.client != nil {
		return w.client.Close()
	}
	return nil
}

type LocalLRUCache struct {
	cache map[string]*localCacheItem
	mu    sync.RWMutex
	maxSz int
	ttl   time.Duration
	keys  []string
}

type localCacheItem struct {
	Data      string
	ExpiresAt time.Time
}

func NewLocalLRUCache(maxSize int, ttl time.Duration) *LocalLRUCache {
	lc := &LocalLRUCache{
		cache: make(map[string]*localCacheItem),
		maxSz: maxSize,
		ttl:   ttl,
		keys:  make([]string, 0, maxSize),
	}
	go lc.cleanup()
	return lc
}

func (l *LocalLRUCache) Get(key string) (string, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	item, exists := l.cache[key]
	if !exists || time.Now().After(item.ExpiresAt) {
		return "", false
	}
	return item.Data, true
}

func (l *LocalLRUCache) Set(key string, value string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.cache) >= l.maxSz {
		l.evict()
	}

	l.cache[key] = &localCacheItem{
		Data:      value,
		ExpiresAt: time.Now().Add(l.ttl),
	}
	l.keys = append(l.keys, key)
}

func (l *LocalLRUCache) Delete(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, key := range keys {
		delete(l.cache, key)
	}
}

func (l *LocalLRUCache) evict() {
	if len(l.keys) > 0 {
		oldest := l.keys[0]
		delete(l.cache, oldest)
		l.keys = l.keys[1:]
	}
}

func (l *LocalLRUCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for key, item := range l.cache {
			if now.After(item.ExpiresAt) {
				delete(l.cache, key)
			}
		}
		l.mu.Unlock()
	}
}

func (l *LocalLRUCache) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.cache)
}

type AdvancedCache struct {
	redis *RedisWrapper
	local *LocalLRUCache
}

func NewAdvancedCache(redisWrapper *RedisWrapper, localCache *LocalLRUCache) *AdvancedCache {
	return &AdvancedCache{
		redis: redisWrapper,
		local: localCache,
	}
}

func (m *AdvancedCache) Get(ctx context.Context, key string) (string, error) {
	if val, found := m.local.Get(key); found {
		return val, nil
	}

	if m.redis != nil {
		val, err := m.redis.Get(ctx, key)
		if err == nil && val != "" {
			m.local.Set(key, val)
			return val, nil
		}
	}

	return "", nil
}

func (m *AdvancedCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.local.Set(key, fmt.Sprintf("%v", value))

	if m.redis != nil {
		return m.redis.Set(ctx, key, value, ttl)
	}
	return nil
}

func (m *AdvancedCache) Delete(ctx context.Context, key string) error {
	m.local.Delete(key)

	if m.redis != nil {
		return m.redis.Del(ctx, key)
	}
	return nil
}

func (m *AdvancedCache) DeletePattern(ctx context.Context, pattern string) error {
	if m.redis != nil {
		return m.redis.DelPattern(ctx, pattern)
	}
	return nil
}

func (m *AdvancedCache) GetLocal() *LocalLRUCache {
	return m.local
}

func (m *AdvancedCache) GetRedis() *RedisWrapper {
	return m.redis
}
