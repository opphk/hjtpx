package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"captchax/pkg/cache"
)

type RateLimitStrategy int

const (
	TokenBucketStrategy RateLimitStrategy = iota
	SlidingWindowStrategy
	LeakyBucketStrategy
	AdaptiveStrategy
)

type RateLimitConfig struct {
	Strategy           RateLimitStrategy
	Rate               float64
	Burst              int
	Window             time.Duration
	BlockDuration      time.Duration
	RedisEnabled       bool
	MetricsEnabled     bool
}

type RateLimitResult struct {
	Allowed   bool
	Remaining int
	ResetAt   time.Time
	RetryIn   time.Duration
}

type RateLimiter interface {
	Allow(ctx context.Context, key string) (*RateLimitResult, error)
	AllowN(ctx context.Context, key string, n int) (*RateLimitResult, error)
	Reset(ctx context.Context, key string) error
	GetLimit(ctx context.Context, key string) (*RateLimitResult, error)
}

type MetricsRateLimiter struct {
	limiter     RateLimiter
	mu          sync.RWMutex
	metrics     map[string]*RateLimitMetrics
}

type RateLimitMetrics struct {
	TotalRequests  atomic.Int64
	AllowedRequests atomic.Int64
	DeniedRequests  atomic.Int64
	LastRequestTime atomic.Int64
}

type TokenBucketLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*TokenBucketState
	rate      float64
	burst     int
	ttl       time.Duration
	stopClean chan struct{}
}

type TokenBucketState struct {
	tokens     float64
	lastUpdate time.Time
	blocked    bool
	blockUntil time.Time
}

type SlidingWindowLimiter struct {
	mu      sync.RWMutex
	windows map[string]*SlidingWindowState
	config  *RateLimitConfig
	redis   *cache.RedisClient
}

type SlidingWindowState struct {
	requests []time.Time
	mu       sync.Mutex
}

type LeakyBucketLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*LeakyBucketState
	config  *RateLimitConfig
	redis   *cache.RedisClient
}

type LeakyBucketState struct {
	level        float64
	lastLeakTime time.Time
	mu           sync.Mutex
}

type AdaptiveRateLimiter struct {
	tokenBucket *TokenBucketLimiter
	slidingWindow *SlidingWindowLimiter
	config     *RateLimitConfig
	redis      *cache.RedisClient
}

type DistributedRateLimiter struct {
	redis      *cache.RedisClient
	config     *RateLimitConfig
	localCache *LocalRateLimitCache
}

type LocalRateLimitCache struct {
	mu      sync.RWMutex
	entries map[string]*RateLimitEntry
	ttl     time.Duration
}

type RateLimitEntry struct {
	Count    int
	WindowStart time.Time
}

func NewTokenBucketLimiter(rate float64, burst int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		buckets:  make(map[string]*TokenBucketState),
		rate:    rate,
		burst:   burst,
		ttl:     1 * time.Hour,
		stopClean: make(chan struct{}),
	}
}

func (tb *TokenBucketLimiter) Allow(ctx context.Context, key string) (*RateLimitResult, error) {
	return tb.AllowN(ctx, key, 1)
}

func (tb *TokenBucketLimiter) AllowN(ctx context.Context, key string, n int) (*RateLimitResult, error) {
	tb.mu.Lock()
	state, exists := tb.buckets[key]
	if !exists {
		state = &TokenBucketState{
			tokens:     float64(tb.burst),
			lastUpdate: time.Now(),
		}
		tb.buckets[key] = state
	}

	if state.blocked && time.Now().Before(state.blockUntil) {
		tb.mu.Unlock()
		return &RateLimitResult{
			Allowed:   false,
			Remaining: 0,
			ResetAt:   state.blockUntil,
			RetryIn:   time.Until(state.blockUntil),
		}, nil
	}

	now := time.Now()
	elapsed := now.Sub(state.lastUpdate).Seconds()
	state.lastUpdate = now

	state.tokens += elapsed * tb.rate
	if state.tokens > float64(tb.burst) {
		state.tokens = float64(tb.burst)
	}

	allowed := state.tokens >= float64(n)
	if allowed {
		state.tokens -= float64(n)
	}

	remaining := int(state.tokens)
	if remaining < 0 {
		remaining = 0
	}

	tb.mu.Unlock()

	return &RateLimitResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   now.Add(time.Duration(float64(tb.burst-remaining)/tb.rate) * time.Second),
		RetryIn:   0,
	}, nil
}

func (tb *TokenBucketLimiter) Reset(ctx context.Context, key string) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	delete(tb.buckets, key)
	return nil
}

func (tb *TokenBucketLimiter) GetLimit(ctx context.Context, key string) (*RateLimitResult, error) {
	tb.mu.RLock()
	state, exists := tb.buckets[key]
	tb.mu.RUnlock()

	if !exists {
		return &RateLimitResult{
			Allowed:   true,
			Remaining: tb.burst,
			ResetAt:   time.Now(),
		}, nil
	}

	remaining := int(state.tokens)
	if remaining < 0 {
		remaining = 0
	}

	return &RateLimitResult{
		Allowed:   remaining > 0,
		Remaining: remaining,
		ResetAt:   time.Now().Add(time.Duration(float64(tb.burst-remaining)/tb.rate) * time.Second),
	}, nil
}

func (tb *TokenBucketLimiter) startCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tb.cleanup()
		case <-tb.stopClean:
			return
		}
	}
}

func (tb *TokenBucketLimiter) cleanup() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	for key, state := range tb.buckets {
		if state.blocked && now.After(state.blockUntil) {
			state.blocked = false
		}

		if now.Sub(state.lastUpdate) > tb.ttl {
			delete(tb.buckets, key)
		}
	}
}

func (tb *TokenBucketLimiter) Stop() {
	close(tb.stopClean)
}

func NewSlidingWindowLimiter(redis *cache.RedisClient, cfg *RateLimitConfig) *SlidingWindowLimiter {
	if cfg == nil {
		cfg = &RateLimitConfig{
			Strategy: SlidingWindowStrategy,
			Rate:    60,
			Burst:   10,
			Window:  1 * time.Minute,
		}
	}

	return &SlidingWindowLimiter{
		windows: make(map[string]*SlidingWindowState),
		config:  cfg,
		redis:   redis,
	}
}

func (sw *SlidingWindowLimiter) Allow(ctx context.Context, key string) (*RateLimitResult, error) {
	return sw.AllowN(ctx, key, 1)
}

func (sw *SlidingWindowLimiter) AllowN(ctx context.Context, key string, n int) (*RateLimitResult, error) {
	now := time.Now()
	windowStart := now.Add(-sw.config.Window)

	if sw.redis != nil {
		return sw.allowDistributed(ctx, key, n, now, windowStart)
	}

	state := sw.getOrCreateState(key)
	state.mu.Lock()

	var validRequests []time.Time
	for _, t := range state.requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	allowed := len(validRequests)+n <= sw.config.Burst
	remaining := sw.config.Burst - len(validRequests) - n
	if remaining < 0 {
		remaining = 0
	}

	if allowed {
		validRequests = append(validRequests, now)
	}

	state.requests = validRequests
	state.mu.Unlock()

	return &RateLimitResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   now.Add(sw.config.Window),
		RetryIn:   0,
	}, nil
}

func (sw *SlidingWindowLimiter) allowDistributed(ctx context.Context, key string, n int, now, windowStart time.Time) (*RateLimitResult, error) {
	redisKey := fmt.Sprintf("ratelimit:sw:%s", key)

	currentCount, err := sw.redis.Get(ctx, redisKey)
	if err != nil {
		currentCount = "0"
	}

	count := 0
	fmt.Sscanf(currentCount, "%d", &count)

	validCount := 0
	windowKey := fmt.Sprintf("ratelimit:sw:history:%s", key)

	timestamps, _ := sw.redis.Get(ctx, windowKey)
	var tsList []int64
	if timestamps != "" {
		for _, ts := range splitString(timestamps, ",") {
			var t int64
			fmt.Sscanf(ts, "%d", &t)
			if t > windowStart.Unix() {
				tsList = append(tsList, t)
			}
		}
	}

	validCount = len(tsList)

	allowed := validCount+n <= sw.config.Burst
	remaining := sw.config.Burst - validCount - n
	if remaining < 0 {
		remaining = 0
	}

	if allowed {
		for i := 0; i < n; i++ {
			tsList = append(tsList, now.Unix())
		}

		tsStr := joinInt64(tsList, ",")
		sw.redis.Set(ctx, windowKey, tsStr, sw.config.Window*2)
	}

	return &RateLimitResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   now.Add(sw.config.Window),
	}, nil
}

func (sw *SlidingWindowLimiter) Reset(ctx context.Context, key string) error {
	state := sw.getOrCreateState(key)
	state.mu.Lock()
	state.requests = nil
	state.mu.Unlock()

	if sw.redis != nil {
		sw.redis.Del(ctx, fmt.Sprintf("ratelimit:sw:%s", key))
		sw.redis.Del(ctx, fmt.Sprintf("ratelimit:sw:history:%s", key))
	}
	return nil
}

func (sw *SlidingWindowLimiter) GetLimit(ctx context.Context, key string) (*RateLimitResult, error) {
	state := sw.getOrCreateState(key)
	state.mu.Lock()
	defer state.mu.Unlock()

	windowStart := time.Now().Add(-sw.config.Window)
	validCount := 0
	for _, t := range state.requests {
		if t.After(windowStart) {
			validCount++
		}
	}

	remaining := sw.config.Burst - validCount
	if remaining < 0 {
		remaining = 0
	}

	return &RateLimitResult{
		Allowed:   remaining > 0,
		Remaining: remaining,
		ResetAt:   time.Now().Add(sw.config.Window),
	}, nil
}

func (sw *SlidingWindowLimiter) getOrCreateState(key string) *SlidingWindowState {
	sw.mu.RLock()
	state, exists := sw.windows[key]
	sw.mu.RUnlock()

	if !exists {
		sw.mu.Lock()
		state, exists = sw.windows[key]
		if !exists {
			state = &SlidingWindowState{
				requests: make([]time.Time, 0),
			}
			sw.windows[key] = state
		}
		sw.mu.Unlock()
	}

	return state
}

func NewLeakyBucketLimiter(redis *cache.RedisClient, cfg *RateLimitConfig) *LeakyBucketLimiter {
	if cfg == nil {
		cfg = &RateLimitConfig{
			Strategy: LeakyBucketStrategy,
			Rate:    1,
			Burst:   10,
		}
	}

	return &LeakyBucketLimiter{
		buckets: make(map[string]*LeakyBucketState),
		config:  cfg,
		redis:   redis,
	}
}

func (lb *LeakyBucketLimiter) Allow(ctx context.Context, key string) (*RateLimitResult, error) {
	return lb.AllowN(ctx, key, 1)
}

func (lb *LeakyBucketLimiter) AllowN(ctx context.Context, key string, n int) (*RateLimitResult, error) {
	state := lb.getOrCreateState(key)
	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(state.lastLeakTime).Seconds()
	state.lastLeakTime = now

	state.level -= elapsed * lb.config.Rate
	if state.level < 0 {
		state.level = 0
	}

	allowed := state.level+float64(n) <= float64(lb.config.Burst)

	if allowed {
		state.level += float64(n)
	}

	remaining := lb.config.Burst - int(state.level)
	if remaining < 0 {
		remaining = 0
	}

	return &RateLimitResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   now.Add(time.Duration(state.level/lb.config.Rate) * time.Second),
	}, nil
}

func (lb *LeakyBucketLimiter) Reset(ctx context.Context, key string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	delete(lb.buckets, key)
	return nil
}

func (lb *LeakyBucketLimiter) GetLimit(ctx context.Context, key string) (*RateLimitResult, error) {
	state := lb.getOrCreateState(key)
	state.mu.Lock()
	defer state.mu.Unlock()

	remaining := lb.config.Burst - int(state.level)
	if remaining < 0 {
		remaining = 0
	}

	return &RateLimitResult{
		Allowed:   remaining > 0,
		Remaining: remaining,
	}, nil
}

func (lb *LeakyBucketLimiter) getOrCreateState(key string) *LeakyBucketState {
	lb.mu.RLock()
	state, exists := lb.buckets[key]
	lb.mu.RUnlock()

	if !exists {
		lb.mu.Lock()
		state, exists = lb.buckets[key]
		if !exists {
			state = &LeakyBucketState{
				level:        0,
				lastLeakTime: time.Now(),
			}
			lb.buckets[key] = state
		}
		lb.mu.Unlock()
	}

	return state
}

func NewAdaptiveRateLimiter(redis *cache.RedisClient, cfg *RateLimitConfig) *AdaptiveRateLimiter {
	if cfg == nil {
		cfg = &RateLimitConfig{
			Strategy: AdaptiveStrategy,
			Rate:    60,
			Burst:   10,
			Window:  1 * time.Minute,
		}
	}

	return &AdaptiveRateLimiter{
		tokenBucket:   NewTokenBucketLimiter(cfg.Rate, cfg.Burst),
		slidingWindow: NewSlidingWindowLimiter(redis, cfg),
		config:       cfg,
		redis:        redis,
	}
}

func (a *AdaptiveRateLimiter) Allow(ctx context.Context, key string) (*RateLimitResult, error) {
	tbResult, _ := a.tokenBucket.Allow(ctx, key)

	swResult, _ := a.slidingWindow.Allow(ctx, key)

	allowed := tbResult.Allowed && swResult.Allowed
	remaining := tbResult.Remaining
	if swResult.Remaining < remaining {
		remaining = swResult.Remaining
	}

	return &RateLimitResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   tbResult.ResetAt,
	}, nil
}

func (a *AdaptiveRateLimiter) AllowN(ctx context.Context, key string, n int) (*RateLimitResult, error) {
	return a.Allow(ctx, key)
}

func (a *AdaptiveRateLimiter) Reset(ctx context.Context, key string) error {
	a.tokenBucket.Reset(ctx, key)
	return a.slidingWindow.Reset(ctx, key)
}

func (a *AdaptiveRateLimiter) GetLimit(ctx context.Context, key string) (*RateLimitResult, error) {
	return a.tokenBucket.GetLimit(ctx, key)
}

func (a *AdaptiveRateLimiter) Stop() {
	a.tokenBucket.Stop()
}

func NewDistributedRateLimiter(redis *cache.RedisClient, cfg *RateLimitConfig) *DistributedRateLimiter {
	if cfg == nil {
		cfg = &RateLimitConfig{
			Strategy:    SlidingWindowStrategy,
			Rate:        60,
			Burst:       100,
			Window:      1 * time.Minute,
			RedisEnabled: true,
		}
	}

	return &DistributedRateLimiter{
		redis:      redis,
		config:     cfg,
		localCache: NewLocalRateLimitCache(1 * time.Minute),
	}
}

func (d *DistributedRateLimiter) Allow(ctx context.Context, key string) (*RateLimitResult, error) {
	return d.AllowN(ctx, key, 1)
}

func (d *DistributedRateLimiter) AllowN(ctx context.Context, key string, n int) (*RateLimitResult, error) {
	if d.redis == nil {
		return d.allowLocal(key, n)
	}

	return d.allowDistributed(ctx, key, n)
}

func (d *DistributedRateLimiter) allowLocal(key string, n int) (*RateLimitResult, error) {
	now := time.Now()

	d.localCache.mu.Lock()
	entry, exists := d.localCache.entries[key]
	if !exists {
		entry = &RateLimitEntry{
			WindowStart: now,
			Count:       0,
		}
		d.localCache.entries[key] = entry
	}

	if now.Sub(entry.WindowStart) > d.config.Window {
		entry.WindowStart = now
		entry.Count = 0
	}

	allowed := entry.Count+n <= d.config.Burst
	remaining := d.config.Burst - entry.Count - n
	if remaining < 0 {
		remaining = 0
	}

	if allowed {
		entry.Count += n
	}

	d.localCache.mu.Unlock()

	return &RateLimitResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   entry.WindowStart.Add(d.config.Window),
	}, nil
}

func (d *DistributedRateLimiter) allowDistributed(ctx context.Context, key string, n int) (*RateLimitResult, error) {
	windowKey := fmt.Sprintf("ratelimit:dw:%s:%d", key, time.Now().Unix()/int64(d.config.Window.Seconds()))

	count, err := d.redis.Incr(ctx, windowKey)
	if err != nil {
		return d.allowLocal(key, n)
	}

	d.redis.Expire(ctx, windowKey, d.config.Window*2)

	allowed := int(count)+n <= d.config.Burst
	remaining := d.config.Burst - int(count) - n
	if remaining < 0 {
		remaining = 0
	}

	return &RateLimitResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   time.Now().Add(d.config.Window),
	}, nil
}

func (d *DistributedRateLimiter) Reset(ctx context.Context, key string) error {
	d.localCache.mu.Lock()
	delete(d.localCache.entries, key)
	d.localCache.mu.Unlock()

	if d.redis != nil {
		d.redis.Del(ctx, fmt.Sprintf("ratelimit:dw:%s:*", key))
	}
	return nil
}

func (d *DistributedRateLimiter) GetLimit(ctx context.Context, key string) (*RateLimitResult, error) {
	d.localCache.mu.RLock()
	entry, exists := d.localCache.entries[key]
	d.localCache.mu.RUnlock()

	if !exists {
		return &RateLimitResult{
			Allowed:   true,
			Remaining: d.config.Burst,
		}, nil
	}

	remaining := d.config.Burst - entry.Count
	if remaining < 0 {
		remaining = 0
	}

	return &RateLimitResult{
		Allowed:   remaining > 0,
		Remaining: remaining,
		ResetAt:   entry.WindowStart.Add(d.config.Window),
	}, nil
}

func NewLocalRateLimitCache(ttl time.Duration) *LocalRateLimitCache {
	return &LocalRateLimitCache{
		entries: make(map[string]*RateLimitEntry),
		ttl:     ttl,
	}
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	result := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i = start - 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func joinInt64(values []int64, sep string) string {
	if len(values) == 0 {
		return ""
	}
	result := fmt.Sprintf("%d", values[0])
	for i := 1; i < len(values); i++ {
		result += sep + fmt.Sprintf("%d", values[i])
	}
	return result
}

type CompositeRateLimiter struct {
	limiters []RateLimiter
	mu       sync.RWMutex
}

func NewCompositeRateLimiter(limiters ...RateLimiter) *CompositeRateLimiter {
	return &CompositeRateLimiter{
		limiters: limiters,
	}
}

func (c *CompositeRateLimiter) Allow(ctx context.Context, key string) (*RateLimitResult, error) {
	for _, limiter := range c.limiters {
		result, err := limiter.Allow(ctx, key)
		if err != nil {
			return nil, err
		}
		if !result.Allowed {
			return result, nil
		}
	}

	return &RateLimitResult{
		Allowed:   true,
		Remaining: c.limiters[0].(*TokenBucketLimiter).burst,
		ResetAt:   time.Now().Add(time.Minute),
	}, nil
}

func (c *CompositeRateLimiter) AllowN(ctx context.Context, key string, n int) (*RateLimitResult, error) {
	return c.Allow(ctx, key)
}

func (c *CompositeRateLimiter) Reset(ctx context.Context, key string) error {
	for _, limiter := range c.limiters {
		if err := limiter.Reset(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (c *CompositeRateLimiter) GetLimit(ctx context.Context, key string) (*RateLimitResult, error) {
	if len(c.limiters) > 0 {
		return c.limiters[0].GetLimit(ctx, key)
	}
	return &RateLimitResult{Allowed: true, Remaining: 0}, nil
}

func (c *CompositeRateLimiter) AddLimiter(limiter RateLimiter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.limiters = append(c.limiters, limiter)
}
