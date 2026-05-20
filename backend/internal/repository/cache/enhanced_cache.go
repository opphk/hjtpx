package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

type EnhancedCache struct {
	client     *redis.Client
	hitCount   atomic.Int64
	missCount  atomic.Int64
	totalOps   atomic.Int64
	mu         sync.RWMutex
	prefixes   map[string]string
	defaultTTL time.Duration
}

var globalEnhancedCache *EnhancedCache
var cacheOnce sync.Once

func NewEnhancedCache(client *redis.Client) *EnhancedCache {
	return &EnhancedCache{
		client:     client,
		prefixes:   make(map[string]string),
		defaultTTL: 5 * time.Minute,
	}
}

func GetEnhancedCache() *EnhancedCache {
	return globalEnhancedCache
}

func InitEnhancedCache(client *redis.Client) {
	cacheOnce.Do(func() {
		if client != nil {
			globalEnhancedCache = NewEnhancedCache(client)
		}
	})
}

func (c *EnhancedCache) RegisterPrefix(name, prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.prefixes[name] = prefix
}

func (c *EnhancedCache) GetPrefix(name string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if prefix, ok := c.prefixes[name]; ok {
		return prefix
	}
	return name
}

func (c *EnhancedCache) buildKey(prefixName string, keys ...string) string {
	prefix := c.GetPrefix(prefixName)
	if len(keys) == 0 {
		return prefix
	}
	result := prefix
	for _, k := range keys {
		result = fmt.Sprintf("%s:%s", result, k)
	}
	return result
}

func (c *EnhancedCache) Set(ctx context.Context, prefixName string, key string, value interface{}, ttl time.Duration) error {
	c.totalOps.Add(1)
	if c.client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	fullKey := c.buildKey(prefixName, key)
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	return c.client.Set(ctx, fullKey, data, ttl).Err()
}

func (c *EnhancedCache) Get(ctx context.Context, prefixName string, key string, dest interface{}) error {
	c.totalOps.Add(1)
	if c.client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	fullKey := c.buildKey(prefixName, key)
	data, err := c.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			c.missCount.Add(1)
			return err
		}
		return err
	}

	c.hitCount.Add(1)
	return json.Unmarshal(data, dest)
}

func (c *EnhancedCache) GetString(ctx context.Context, prefixName string, key string) (string, error) {
	c.totalOps.Add(1)
	if c.client == nil {
		return "", fmt.Errorf("redis client not initialized")
	}

	fullKey := c.buildKey(prefixName, key)
	result, err := c.client.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			c.missCount.Add(1)
			return "", err
		}
		return "", err
	}

	c.hitCount.Add(1)
	return result, nil
}

func (c *EnhancedCache) SetString(ctx context.Context, prefixName string, key string, value string, ttl time.Duration) error {
	c.totalOps.Add(1)
	if c.client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	fullKey := c.buildKey(prefixName, key)
	if ttl == 0 {
		ttl = c.defaultTTL
	}

	return c.client.Set(ctx, fullKey, value, ttl).Err()
}

func (c *EnhancedCache) Delete(ctx context.Context, prefixName string, keys ...string) error {
	c.totalOps.Add(1)
	if c.client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	fullKeys := make([]string, len(keys))
	for i, k := range keys {
		fullKeys[i] = c.buildKey(prefixName, k)
	}

	return c.client.Del(ctx, fullKeys...).Err()
}

func (c *EnhancedCache) DeleteByPattern(ctx context.Context, pattern string) error {
	c.totalOps.Add(1)
	if c.client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	return nil
}

func (c *EnhancedCache) Exists(ctx context.Context, prefixName string, key string) (bool, error) {
	c.totalOps.Add(1)
	if c.client == nil {
		return false, fmt.Errorf("redis client not initialized")
	}

	fullKey := c.buildKey(prefixName, key)
	result, err := c.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (c *EnhancedCache) Expire(ctx context.Context, prefixName string, key string, ttl time.Duration) error {
	c.totalOps.Add(1)
	if c.client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	fullKey := c.buildKey(prefixName, key)
	return c.client.Expire(ctx, fullKey, ttl).Err()
}

func (c *EnhancedCache) TTL(ctx context.Context, prefixName string, key string) (time.Duration, error) {
	c.totalOps.Add(1)
	if c.client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}

	fullKey := c.buildKey(prefixName, key)
	return c.client.TTL(ctx, fullKey).Result()
}

func (c *EnhancedCache) Incr(ctx context.Context, prefixName string, key string) (int64, error) {
	c.totalOps.Add(1)
	if c.client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}

	fullKey := c.buildKey(prefixName, key)
	return c.client.Incr(ctx, fullKey).Result()
}

func (c *EnhancedCache) IncrBy(ctx context.Context, prefixName string, key string, value int64) (int64, error) {
	c.totalOps.Add(1)
	if c.client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}

	fullKey := c.buildKey(prefixName, key)
	return c.client.IncrBy(ctx, fullKey, value).Result()
}

func (c *EnhancedCache) IncrWithExpire(ctx context.Context, prefixName string, key string, ttl time.Duration) (int64, error) {
	if c.client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}

	fullKey := c.buildKey(prefixName, key)
	pipe := c.client.Pipeline()
	incr := pipe.Incr(ctx, fullKey)
	pipe.Expire(ctx, fullKey, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	c.totalOps.Add(1)
	return incr.Val(), nil
}

func (c *EnhancedCache) SetNX(ctx context.Context, prefixName string, key string, value interface{}, ttl time.Duration) (bool, error) {
	c.totalOps.Add(1)
	if c.client == nil {
		return false, fmt.Errorf("redis client not initialized")
	}

	fullKey := c.buildKey(prefixName, key)
	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	return c.client.SetNX(ctx, fullKey, data, ttl).Result()
}

func (c *EnhancedCache) GetOrSet(ctx context.Context, prefixName string, key string, value interface{}, ttl time.Duration) (interface{}, bool, error) {
	c.totalOps.Add(1)
	if c.client == nil {
		return nil, false, fmt.Errorf("redis client not initialized")
	}

	fullKey := c.buildKey(prefixName, key)
	data, err := c.client.Get(ctx, fullKey).Bytes()
	if err == nil {
		c.hitCount.Add(1)
		var result interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, false, err
		}
		return result, true, nil
	}

	if err != redis.Nil {
		return nil, false, err
	}

	c.missCount.Add(1)

	data, err = json.Marshal(value)
	if err != nil {
		return nil, false, err
	}

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	if err := c.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
		return nil, false, err
	}

	return value, false, nil
}

func (c *EnhancedCache) GetCacheStats() map[string]interface{} {
	hits := c.hitCount.Load()
	misses := c.missCount.Load()
	total := hits + misses

	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"hits":         hits,
		"misses":       misses,
		"total_ops":    c.totalOps.Load(),
		"hit_rate":     hitRate,
		"target_hit_rate": 95.0,
		"hit_rate_met":    hitRate >= 95.0,
	}
}

func (c *EnhancedCache) ResetStats() {
	c.hitCount.Store(0)
	c.missCount.Store(0)
	c.totalOps.Store(0)
}

type CachePrefix string

const (
	PrefixCaptcha      CachePrefix = "captcha"
	PrefixSession      CachePrefix = "session"
	PrefixUser         CachePrefix = "user"
	PrefixStats        CachePrefix = "stats"
	PrefixRateLimit    CachePrefix = "ratelimit"
	PrefixConfig       CachePrefix = "config"
	PrefixVerification CachePrefix = "verification"
)

func (c *EnhancedCache) SetCaptchaSession(ctx context.Context, sessionID string, data interface{}, ttl time.Duration) error {
	return c.Set(ctx, string(PrefixCaptcha), sessionID, data, ttl)
}

func (c *EnhancedCache) GetCaptchaSession(ctx context.Context, sessionID string, dest interface{}) error {
	return c.Get(ctx, string(PrefixCaptcha), sessionID, dest)
}

func (c *EnhancedCache) DeleteCaptchaSession(ctx context.Context, sessionID string) error {
	return c.Delete(ctx, string(PrefixCaptcha), sessionID)
}

func (c *EnhancedCache) SetUserCache(ctx context.Context, userID string, data interface{}, ttl time.Duration) error {
	return c.Set(ctx, string(PrefixUser), userID, data, ttl)
}

func (c *EnhancedCache) GetUserCache(ctx context.Context, userID string, dest interface{}) error {
	return c.Get(ctx, string(PrefixUser), userID, dest)
}

func (c *EnhancedCache) SetRateLimit(ctx context.Context, identifier string, count int64, window time.Duration) error {
	return c.Set(ctx, string(PrefixRateLimit), identifier, count, window)
}

func (c *EnhancedCache) GetRateLimit(ctx context.Context, identifier string) (int64, error) {
	var count int64
	err := c.Get(ctx, string(PrefixRateLimit), identifier, &count)
	return count, err
}

func (c *EnhancedCache) IncrRateLimit(ctx context.Context, identifier string, window time.Duration) (int64, error) {
	return c.IncrWithExpire(ctx, string(PrefixRateLimit), identifier, window)
}
