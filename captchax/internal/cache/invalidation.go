package cache

import (
	"context"
	"strings"
	"sync"
	"time"
)

type InvalidationStrategy int

const (
	TTLBased InvalidationStrategy = iota
	ActiveInvalidation
	HybridInvalidation
)

type CacheInvalidator interface {
	Invalidate(ctx context.Context, key string) error
	InvalidatePattern(ctx context.Context, pattern string) error
	InvalidateWithTag(ctx context.Context, tag string) error
	RegisterTag(tag string, keys ...string)
}

type InvalidationConfig struct {
	Strategy            InvalidationStrategy
	TTL                 time.Duration
	MaxTTL              time.Duration
	CleanupInterval     time.Duration
	StaleThresholdRatio  float64
}

type InvalidationRedis interface {
	GetBytes(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Keys(ctx context.Context, pattern string) ([]string, error)
}

type TTLCache struct {
	mu         sync.RWMutex
	items      map[string]*ttlItem
	redis      InvalidationRedis
	keyPrefix  string
	defaultTTL time.Duration
	maxTTL     time.Duration
	stopCh     chan struct{}
}

type ttlItem struct {
	Value     []byte
	ExpiresAt time.Time
	Tags      []string
}

type TagManager struct {
	mu  sync.RWMutex
	tags map[string]map[string]bool
}

func NewTagManager() *TagManager {
	return &TagManager{
		tags: make(map[string]map[string]bool),
	}
}

func (tm *TagManager) RegisterTag(tag string, keys ...string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tags[tag]; !exists {
		tm.tags[tag] = make(map[string]bool)
	}

	for _, key := range keys {
		tm.tags[tag][key] = true
	}
}

func (tm *TagManager) GetKeysByTag(tag string) []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if keys, exists := tm.tags[tag]; exists {
		result := make([]string, 0, len(keys))
		for k := range keys {
			result = append(result, k)
		}
		return result
	}
	return nil
}

func (tm *TagManager) RemoveTag(tag string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.tags, tag)
}

type CacheInvalidationManager struct {
	ttlCache    *TTLCache
	tagManager  *TagManager
	redis       InvalidationRedis
	config      *InvalidationConfig
	stopCh      chan struct{}
}

func NewCacheInvalidationManager(redis InvalidationRedis, cfg *InvalidationConfig) *CacheInvalidationManager {
	if cfg == nil {
		cfg = &InvalidationConfig{
			Strategy:            HybridInvalidation,
			TTL:                 5 * time.Minute,
			MaxTTL:              1 * time.Hour,
			CleanupInterval:     1 * time.Minute,
			StaleThresholdRatio: 0.2,
		}
	}

	cim := &CacheInvalidationManager{
		ttlCache: &TTLCache{
			items:      make(map[string]*ttlItem),
			redis:      redis,
			keyPrefix:  "captchax:inv:",
			defaultTTL: cfg.TTL,
			maxTTL:     cfg.MaxTTL,
			stopCh:     make(chan struct{}),
		},
		tagManager: NewTagManager(),
		redis:      redis,
		config:     cfg,
		stopCh:     make(chan struct{}),
	}

	go cim.startCleanup()
	return cim
}

func (cim *CacheInvalidationManager) Set(ctx context.Context, key string, value []byte, ttl time.Duration, tags ...string) error {
	if ttl > cim.config.MaxTTL {
		ttl = cim.config.MaxTTL
	}
	if ttl == 0 {
		ttl = cim.config.TTL
	}

	cim.ttlCache.mu.Lock()
	cim.ttlCache.items[key] = &ttlItem{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
		Tags:      tags,
	}
	cim.ttlCache.mu.Unlock()

	for _, tag := range tags {
		cim.tagManager.RegisterTag(tag, key)
	}

	if cim.redis != nil {
		return cim.redis.Set(ctx, cim.ttlCache.keyPrefix+key, value, ttl)
	}
	return nil
}

func (cim *CacheInvalidationManager) Get(ctx context.Context, key string) ([]byte, bool) {
	cim.ttlCache.mu.RLock()
	item, exists := cim.ttlCache.items[key]
	cim.ttlCache.mu.RUnlock()

	if exists && time.Now().Before(item.ExpiresAt) {
		return item.Value, true
	}

	if exists {
		cim.ttlCache.mu.Lock()
		delete(cim.ttlCache.items, key)
		cim.ttlCache.mu.Unlock()
	}

	if cim.redis != nil {
		data, err := cim.redis.GetBytes(ctx, cim.ttlCache.keyPrefix+key)
		if err == nil {
			cim.ttlCache.mu.Lock()
			cim.ttlCache.items[key] = &ttlItem{
				Value:     data,
				ExpiresAt: time.Now().Add(cim.ttlCache.defaultTTL),
			}
			cim.ttlCache.mu.Unlock()
			return data, true
		}
	}

	return nil, false
}

func (cim *CacheInvalidationManager) Invalidate(ctx context.Context, key string) error {
	cim.ttlCache.mu.Lock()
	delete(cim.ttlCache.items, key)
	cim.ttlCache.mu.Unlock()

	if cim.redis != nil {
		return cim.redis.Del(ctx, cim.ttlCache.keyPrefix+key)
	}
	return nil
}

func (cim *CacheInvalidationManager) InvalidatePattern(ctx context.Context, pattern string) error {
	cim.ttlCache.mu.Lock()
	for key := range cim.ttlCache.items {
		if matchPattern(key, pattern) {
			delete(cim.ttlCache.items, key)
		}
	}
	cim.ttlCache.mu.Unlock()

	if cim.redis != nil {
		keys, err := cim.redis.Keys(ctx, cim.ttlCache.keyPrefix+pattern)
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			unprefixedKeys := make([]string, len(keys))
			for i, k := range keys {
				unprefixedKeys[i] = k[len(cim.ttlCache.keyPrefix):]
			}
			return cim.redis.Del(ctx, keys...)
		}
	}
	return nil
}

func (cim *CacheInvalidationManager) InvalidateWithTag(ctx context.Context, tag string) error {
	keys := cim.tagManager.GetKeysByTag(tag)
	if len(keys) == 0 {
		return nil
	}

	for _, key := range keys {
		if err := cim.Invalidate(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (cim *CacheInvalidationManager) RegisterTag(tag string, keys ...string) {
	cim.tagManager.RegisterTag(tag, keys...)
}

func (cim *CacheInvalidationManager) startCleanup() {
	ticker := time.NewTicker(cim.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cim.cleanup()
		case <-cim.stopCh:
			return
		}
	}
}

func (cim *CacheInvalidationManager) cleanup() {
	now := time.Now()
	staleThresholdDuration := cim.ttlCache.defaultTTL * time.Duration(1+cim.config.StaleThresholdRatio)
	staleThreshold := now.Add(staleThresholdDuration)

	cim.ttlCache.mu.Lock()
	for key, item := range cim.ttlCache.items {
		if now.After(item.ExpiresAt) {
			delete(cim.ttlCache.items, key)
			if cim.redis != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				cim.redis.Del(ctx, cim.ttlCache.keyPrefix+key)
				cancel()
			}
		}
	}

	for key, item := range cim.ttlCache.items {
		if item.ExpiresAt.Before(staleThreshold) && now.Sub(item.ExpiresAt) < staleThresholdDuration {
			cim.ttlCache.items[key] = &ttlItem{
				Value:     item.Value,
				ExpiresAt: now.Add(cim.ttlCache.defaultTTL),
				Tags:      item.Tags,
			}
		}
	}
	cim.ttlCache.mu.Unlock()
}

func (cim *CacheInvalidationManager) Stop() {
	close(cim.stopCh)
}

func (cim *CacheInvalidationManager) GetStats() map[string]interface{} {
	cim.ttlCache.mu.RLock()
	localCount := len(cim.ttlCache.items)
	cim.ttlCache.mu.RUnlock()

	return map[string]interface{}{
		"strategy":            cim.config.Strategy,
		"local_items":         localCount,
		"default_ttl":         cim.config.TTL.String(),
		"max_ttl":             cim.config.MaxTTL.String(),
		"cleanup_interval":    cim.config.CleanupInterval.String(),
		"stale_threshold_ratio": cim.config.StaleThresholdRatio,
	}
}

func matchPattern(key, pattern string) bool {
	if pattern == "*" || pattern == "" {
		return true
	}

	if !strings.Contains(pattern, "*") {
		return key == pattern
	}

	pattern = strings.TrimPrefix(pattern, "*")
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(key, prefix)
	}
	return strings.Contains(key, pattern)
}
