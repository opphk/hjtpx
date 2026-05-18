package service

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type TokenBucketConfig struct {
	Capacity      int64
	RefillRate    float64
	RefillPerSec  float64
}

type TokenBucketRateLimitService struct {
	config  TokenBucketConfig
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
}

type RateLimitResult struct {
	Allowed    bool
	Remaining  float64
	ResetAt    time.Time
	RetryAfter float64
}

func NewTokenBucketRateLimitService() *TokenBucketRateLimitService {
	return &TokenBucketRateLimitService{
		config: TokenBucketConfig{
			Capacity:     100,
			RefillRate:   10,
			RefillPerSec: 10,
		},
		buckets: make(map[string]*TokenBucket),
	}
}

type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	lastRefill time.Time
	capacity   int64
	refillRate float64
	maxTokens  float64
}

type SlidingWindowConfig struct {
	WindowSize time.Duration
	MaxRequests int64
}

type SlidingWindowRateLimitService struct {
	config     SlidingWindowConfig
	windows    map[string]*SlidingWindow
	mu         sync.RWMutex
}

type SlidingWindow struct {
	timestamps []time.Time
	windowSize time.Duration
	maxRequests int64
	mu         sync.Mutex
}



func (s *TokenBucketRateLimitService) Allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	bucket, exists := s.buckets[key]
	if !exists {
		bucket = &TokenBucket{
			tokens:     float64(s.config.Capacity),
			lastRefill: time.Now(),
			capacity:   s.config.Capacity,
			refillRate: s.config.RefillRate,
			maxTokens:  float64(s.config.Capacity),
		}
		s.buckets[key] = bucket
	}

	bucket.refill()

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true
	}

	return false
}

func (s *TokenBucketRateLimitService) Check(key string) (bool, float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bucket, exists := s.buckets[key]
	if !exists {
		return true, float64(s.config.Capacity)
	}

	elapsed := time.Since(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * bucket.refillRate
	if bucket.tokens > bucket.maxTokens {
		bucket.tokens = bucket.maxTokens
	}

	return bucket.tokens >= 1, bucket.tokens
}

func (b *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()

	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}

	b.lastRefill = now
}

func (s *TokenBucketRateLimitService) GetTokens(key string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bucket, exists := s.buckets[key]
	if !exists {
		return float64(s.config.Capacity)
	}

	return bucket.tokens
}

func (s *TokenBucketRateLimitService) Reset(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.buckets, key)
}

func (s *TokenBucketRateLimitService) ResetAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buckets = make(map[string]*TokenBucket)
}

func (s *TokenBucketRateLimitService) CheckIPTokenBucketLimit(ctx context.Context, ip string, config *TokenBucketConfig) (*RateLimitResult, error) {
	key := "ip:" + ip

	if config == nil {
		config = &TokenBucketConfig{
			Capacity:     100,
			RefillRate:   10,
			RefillPerSec: 10,
		}
	}

	s.mu.Lock()
	bucket, exists := s.buckets[key]
	if !exists {
		bucket = &TokenBucket{
			tokens:     float64(config.Capacity),
			lastRefill: time.Now(),
			capacity:   config.Capacity,
			refillRate: config.RefillRate,
			maxTokens:  float64(config.Capacity),
		}
		s.buckets[key] = bucket
	} else {
		bucket.capacity = config.Capacity
		bucket.refillRate = config.RefillRate
		bucket.maxTokens = float64(config.Capacity)
	}
	s.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * bucket.refillRate
	if bucket.tokens > bucket.maxTokens {
		bucket.tokens = bucket.maxTokens
	}
	bucket.lastRefill = now

	allowed := bucket.tokens >= 1
	if allowed {
		bucket.tokens--
	}

	retryAfter := 0
	if !allowed {
		retryAfter = int((1 - bucket.tokens) / bucket.refillRate)
		if retryAfter < 1 {
			retryAfter = 1
		}
	}

	remaining := int(bucket.tokens)
	if remaining < 0 {
		remaining = 0
	}

	return &RateLimitResult{
		Allowed:    allowed,
		Remaining:  float64(remaining),
		ResetAt:    now.Add(time.Duration(float64(config.Capacity)/config.RefillRate) * time.Second),
		RetryAfter: float64(retryAfter),
	}, nil
}

func (s *TokenBucketRateLimitService) GetConfig() TokenBucketConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *TokenBucketRateLimitService) UpdateConfig(config TokenBucketConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

func (s *TokenBucketRateLimitService) GetGlobalStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalTokens := float64(0)
	for _, bucket := range s.buckets {
		totalTokens += bucket.tokens
	}

	return map[string]interface{}{
		"total_buckets":    len(s.buckets),
		"total_tokens":     totalTokens,
		"config":           s.config,
	}
}

func (s *TokenBucketRateLimitService) GetBucketList() []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bucketList := make([]map[string]interface{}, 0, len(s.buckets))
	for key, bucket := range s.buckets {
		bucketList = append(bucketList, map[string]interface{}{
			"key":          key,
			"tokens":       bucket.tokens,
			"capacity":     bucket.capacity,
			"refill_rate":  bucket.refillRate,
			"last_refill":  bucket.lastRefill,
		})
	}

	return bucketList
}

func (s *TokenBucketRateLimitService) GetBucketStats(key string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bucket, exists := s.buckets[key]
	if !exists {
		return nil
	}

	return map[string]interface{}{
		"key":          key,
		"tokens":       bucket.tokens,
		"capacity":     bucket.capacity,
		"refill_rate":  bucket.refillRate,
		"max_tokens":   bucket.maxTokens,
		"last_refill":  bucket.lastRefill,
	}
}

func (s *TokenBucketRateLimitService) UpdateBucketConfig(key string, config *TokenBucketConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bucket, exists := s.buckets[key]
	if !exists {
		return fmt.Errorf("bucket not found: %s", key)
	}

	if config.Capacity > 0 {
		bucket.capacity = config.Capacity
		bucket.maxTokens = float64(config.Capacity)
	}
	if config.RefillRate > 0 {
		bucket.refillRate = config.RefillRate
	}

	return nil
}

func (s *TokenBucketRateLimitService) ResetBucket(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if bucket, exists := s.buckets[key]; exists {
		bucket.tokens = float64(bucket.capacity)
		bucket.lastRefill = time.Now()
	}

	return nil
}

func (s *TokenBucketRateLimitService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})
	for key, bucket := range s.buckets {
		stats[key] = map[string]interface{}{
			"tokens":      bucket.tokens,
			"capacity":    bucket.capacity,
			"refill_rate": bucket.refillRate,
			"last_refill": bucket.lastRefill,
		}
	}

	return map[string]interface{}{
		"total_buckets": len(s.buckets),
		"buckets":       stats,
		"config":         s.config,
	}
}

func NewSlidingWindowRateLimitService() *SlidingWindowRateLimitService {
	return &SlidingWindowRateLimitService{
		config: SlidingWindowConfig{
			WindowSize:  1 * time.Minute,
			MaxRequests: 100,
		},
		windows: make(map[string]*SlidingWindow),
	}
}

func NewSlidingWindowRateLimitServiceWithConfig(config SlidingWindowConfig) *SlidingWindowRateLimitService {
	if config.WindowSize == 0 {
		config.WindowSize = 1 * time.Minute
	}

	return &SlidingWindowRateLimitService{
		config:  config,
		windows: make(map[string]*SlidingWindow),
	}
}

func (s *SlidingWindowRateLimitService) Allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	window, exists := s.windows[key]
	if !exists {
		window = &SlidingWindow{
			timestamps:  make([]time.Time, 0),
			windowSize:  s.config.WindowSize,
			maxRequests: s.config.MaxRequests,
		}
		s.windows[key] = window
	}

	return window.Allow()
}

func (s *SlidingWindowRateLimitService) Check(key string) (bool, int64, time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	window, exists := s.windows[key]
	if !exists {
		return true, int64(s.config.MaxRequests), time.Now().Add(s.config.WindowSize)
	}

	return window.Check()
}

func (w *SlidingWindow) Allow() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.cleanup()

	if int64(len(w.timestamps)) < w.maxRequests {
		w.timestamps = append(w.timestamps, time.Now())
		return true
	}

	return false
}

func (w *SlidingWindow) Check() (bool, int64, time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.cleanup()

	remaining := w.maxRequests - int64(len(w.timestamps))
	if remaining < 0 {
		remaining = 0
	}

	resetAt := time.Now().Add(w.windowSize)
	if len(w.timestamps) > 0 {
		oldest := w.timestamps[0]
		resetAt = oldest.Add(w.windowSize)
	}

	return int64(len(w.timestamps)) < w.maxRequests, remaining, resetAt
}

func (w *SlidingWindow) cleanup() {
	cutoff := time.Now().Add(-w.windowSize)
	i := 0
	for i < len(w.timestamps) && w.timestamps[i].Before(cutoff) {
		i++
	}
	if i > 0 {
		w.timestamps = w.timestamps[i:]
	}
}

func (s *SlidingWindowRateLimitService) Reset(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.windows, key)
}

func (s *SlidingWindowRateLimitService) ResetAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.windows = make(map[string]*SlidingWindow)
}

func (s *SlidingWindowRateLimitService) CheckSlidingWindow(ctx context.Context, key string, maxRequests int64) (*RateLimitResult, error) {
	s.mu.Lock()
	window, exists := s.windows[key]
	if !exists {
		window = &SlidingWindow{
			timestamps:  make([]time.Time, 0),
			windowSize:  s.config.WindowSize,
			maxRequests: s.config.MaxRequests,
		}
		s.windows[key] = window
	}
	if maxRequests > 0 {
		window.maxRequests = maxRequests
	}
	s.mu.Unlock()

	window.mu.Lock()
	defer window.mu.Unlock()

	window.cleanup()

	now := time.Now()
	allowed := int64(len(window.timestamps)) < window.maxRequests

	if allowed {
		window.timestamps = append(window.timestamps, now)
	}

	remaining := window.maxRequests - int64(len(window.timestamps))
	if remaining < 0 {
		remaining = 0
	}

	var resetAt time.Time
	if len(window.timestamps) > 0 {
		resetAt = window.timestamps[0].Add(window.windowSize)
	} else {
		resetAt = now.Add(window.windowSize)
	}

	retryAfter := 0
	if !allowed {
		if len(window.timestamps) > 0 {
			retryAfter = int(window.timestamps[0].Add(window.windowSize).Sub(now).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
		}
	}

	return &RateLimitResult{
		Allowed:    allowed,
		Remaining:  float64(remaining),
		ResetAt:    resetAt,
		RetryAfter: float64(retryAfter),
	}, nil
}

func (s *SlidingWindowRateLimitService) GetConfig() SlidingWindowConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *SlidingWindowRateLimitService) UpdateConfig(config SlidingWindowConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config

	for _, window := range s.windows {
		window.mu.Lock()
		window.windowSize = config.WindowSize
		window.maxRequests = config.MaxRequests
		window.mu.Unlock()
	}
}

func (s *SlidingWindowRateLimitService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})
	for key, window := range s.windows {
		window.mu.Lock()
		stats[key] = map[string]interface{}{
			"request_count": len(window.timestamps),
			"max_requests":  window.maxRequests,
			"window_size":    window.windowSize,
		}
		window.mu.Unlock()
	}

	return map[string]interface{}{
		"total_windows": len(s.windows),
		"windows":        stats,
		"config":         s.config,
	}
}

type LeakyBucketConfig struct {
	Capacity    int64
	LeakRate    float64
}

type LeakyBucketRateLimitService struct {
	config  LeakyBucketConfig
	buckets map[string]*LeakyBucket
	mu      sync.RWMutex
}

type LeakyBucket struct {
	water      int64
	lastLeak   time.Time
	capacity   int64
	leakRate   float64
	mu         sync.Mutex
}

func NewLeakyBucketRateLimitService() *LeakyBucketRateLimitService {
	return &LeakyBucketRateLimitService{
		config: LeakyBucketConfig{
			Capacity: 100,
			LeakRate: 10,
		},
		buckets: make(map[string]*LeakyBucket),
	}
}

func (s *LeakyBucketRateLimitService) Allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	bucket, exists := s.buckets[key]
	if !exists {
		bucket = &LeakyBucket{
			water:     0,
			lastLeak:  time.Now(),
			capacity:  s.config.Capacity,
			leakRate:  s.config.LeakRate,
		}
		s.buckets[key] = bucket
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	bucket.leak()

	if bucket.water < bucket.capacity {
		bucket.water++
		return true
	}

	return false
}

func (b *LeakyBucket) leak() {
	now := time.Now()
	elapsed := now.Sub(b.lastLeak).Seconds()

	leaked := int64(elapsed * b.leakRate)
	if leaked > 0 {
		if leaked >= b.water {
			b.water = 0
		} else {
			b.water -= leaked
		}
		b.lastLeak = now
	}
}

func (s *LeakyBucketRateLimitService) Reset(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.buckets, key)
}

func (s *LeakyBucketRateLimitService) GetConfig() LeakyBucketConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}



