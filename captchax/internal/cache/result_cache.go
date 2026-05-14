package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"captchax/pkg/cache"
)

type VerifyResult struct {
	Valid       bool
	CaptchaID   string
	ClientToken string
	Attempts    int
	VerifiedAt  time.Time
}

type ResultCacheConfig struct {
	DefaultTTL    time.Duration
	MaxAttempts   int
	CleanupPeriod time.Duration
}

type ResultCache struct {
	mu         sync.RWMutex
	redis      *cache.RedisClient
	config     *ResultCacheConfig
	localCache map[string]*VerifyResult
	localTTL   time.Duration
	stopCleanup chan struct{}
}

func NewResultCache(redisClient *cache.RedisClient, cfg *ResultCacheConfig) *ResultCache {
	if cfg == nil {
		cfg = &ResultCacheConfig{
			DefaultTTL:    5 * time.Minute,
			MaxAttempts:   3,
			CleanupPeriod: 1 * time.Minute,
		}
	}

	rc := &ResultCache{
		redis:      redisClient,
		config:     cfg,
		localCache: make(map[string]*VerifyResult),
		localTTL:   30 * time.Second,
		stopCleanup: make(chan struct{}),
	}

	go rc.startCleanup()
	return rc
}

func (c *ResultCache) redisKey(captchaID, clientToken string) string {
	return "captchax:verify:" + captchaID + ":" + clientToken
}

func (c *ResultCache) clientKey(captchaID, clientToken string) string {
	return captchaID + ":" + clientToken
}

func (c *ResultCache) Set(ctx context.Context, captchaID, clientToken string, result *VerifyResult) error {
	key := c.clientKey(captchaID, clientToken)

	c.mu.Lock()
	c.localCache[key] = result
	c.mu.Unlock()

	if c.redis != nil {
		redisKey := c.redisKey(captchaID, clientToken)
		data, err := json.Marshal(result)
		if err != nil {
			return err
		}
		return c.redis.Set(ctx, redisKey, string(data), c.config.DefaultTTL)
	}
	return nil
}

func (c *ResultCache) Get(ctx context.Context, captchaID, clientToken string) (*VerifyResult, bool) {
	key := c.clientKey(captchaID, clientToken)

	c.mu.RLock()
	if local, ok := c.localCache[key]; ok {
		c.mu.RUnlock()
		if time.Since(local.VerifiedAt) < c.localTTL {
			return local, true
		}
	}
	c.mu.RUnlock()

	if c.redis != nil {
		redisKey := c.redisKey(captchaID, clientToken)
		data, err := c.redis.Get(ctx, redisKey)
		if err == nil {
			var result VerifyResult
			if err := json.Unmarshal([]byte(data), &result); err == nil {
				c.mu.Lock()
				c.localCache[key] = &result
				c.mu.Unlock()
				return &result, true
			}
		}
	}

	return nil, false
}

func (c *ResultCache) IncrementAttempts(ctx context.Context, captchaID, clientToken string) (int, error) {
	key := c.clientKey(captchaID, clientToken)

	c.mu.Lock()
	defer c.mu.Unlock()

	result, ok := c.localCache[key]
	if !ok {
		result = &VerifyResult{
			CaptchaID:   captchaID,
			ClientToken: clientToken,
			Attempts:    0,
		}
	}

	result.Attempts++
	attempts := result.Attempts
	c.localCache[key] = result

	if c.redis != nil {
		redisKey := c.redisKey(captchaID, clientToken)
		data, err := json.Marshal(result)
		if err != nil {
			return attempts, err
		}
		return attempts, c.redis.Set(ctx, redisKey, string(data), c.config.DefaultTTL)
	}

	return attempts, nil
}

func (c *ResultCache) IsMaxAttempts(ctx context.Context, captchaID, clientToken string) bool {
	result, ok := c.Get(ctx, captchaID, clientToken)
	if !ok {
		return false
	}
	return result.Attempts >= c.config.MaxAttempts
}

func (c *ResultCache) Delete(ctx context.Context, captchaID, clientToken string) error {
	key := c.clientKey(captchaID, clientToken)

	c.mu.Lock()
	delete(c.localCache, key)
	c.mu.Unlock()

	if c.redis != nil {
		redisKey := c.redisKey(captchaID, clientToken)
		return c.redis.Del(ctx, redisKey)
	}
	return nil
}

func (c *ResultCache) MarkVerified(ctx context.Context, captchaID, clientToken string) error {
	result := &VerifyResult{
		Valid:       true,
		CaptchaID:   captchaID,
		ClientToken: clientToken,
		Attempts:    0,
		VerifiedAt:  time.Now(),
	}

	return c.Set(ctx, captchaID, clientToken, result)
}

func (c *ResultCache) IsVerified(ctx context.Context, captchaID, clientToken string) bool {
	result, ok := c.Get(ctx, captchaID, clientToken)
	if !ok {
		return false
	}
	return result.Valid
}

func (c *ResultCache) startCleanup() {
	ticker := time.NewTicker(c.config.CleanupPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

func (c *ResultCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, result := range c.localCache {
		if now.Sub(result.VerifiedAt) > c.localTTL {
			delete(c.localCache, k)
		}
	}
}

func (c *ResultCache) Stop() {
	close(c.stopCleanup)
}

func (c *ResultCache) Stats() map[string]interface{} {
	c.mu.RLock()
	localCount := len(c.localCache)
	c.mu.RUnlock()

	stats := map[string]interface{}{
		"local_cache_size": localCount,
		"max_attempts":     c.config.MaxAttempts,
		"default_ttl":      c.config.DefaultTTL.String(),
		"local_ttl":        c.localTTL.String(),
	}

	if c.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		count, err := c.redis.Keys(ctx, "captchax:verify:*")
		cancel()
		if err == nil {
			stats["redis_cache_size"] = len(count)
		}
	}

	return stats
}

type DeduplicationCache struct {
	mu           sync.RWMutex
	seen         map[string]time.Time
	ttl          time.Duration
	stopCleanup  chan struct{}
}

func NewDeduplicationCache(ttl time.Duration) *DeduplicationCache {
	if ttl == 0 {
		ttl = 10 * time.Minute
	}

	dc := &DeduplicationCache{
		seen:        make(map[string]time.Time),
		ttl:         ttl,
		stopCleanup: make(chan struct{}),
	}

	go dc.startCleanup()
	return dc
}

func (dc *DeduplicationCache) CheckAndMark(key string) bool {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if _, exists := dc.seen[key]; exists {
		return false
	}

	dc.seen[key] = time.Now()
	return true
}

func (dc *DeduplicationCache) IsDuplicate(key string) bool {
	dc.mu.RLock()
	_, exists := dc.seen[key]
	dc.mu.RUnlock()
	return exists
}

func (dc *DeduplicationCache) Remove(key string) {
	dc.mu.Lock()
	delete(dc.seen, key)
	dc.mu.Unlock()
}

func (dc *DeduplicationCache) startCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dc.cleanup()
		case <-dc.stopCleanup:
			return
		}
	}
}

func (dc *DeduplicationCache) cleanup() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	now := time.Now()
	for k, ts := range dc.seen {
		if now.Sub(ts) > dc.ttl {
			delete(dc.seen, k)
		}
	}
}

func (dc *DeduplicationCache) Stop() {
	close(dc.stopCleanup)
}

func (dc *DeduplicationCache) Len() int {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return len(dc.seen)
}

type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	rate       float64
	maxTokens  float64
	lastUpdate time.Time
}

func NewTokenBucket(rate, maxTokens float64) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		rate:       rate,
		maxTokens:  maxTokens,
		lastUpdate: time.Now(),
	}
}

func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

func (tb *TokenBucket) AllowN(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastUpdate).Seconds()
	tb.lastUpdate = now

	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}

	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}

	return false
}

func (tb *TokenBucket) Wait(ctx context.Context) error {
	for {
		if tb.Allow() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}
