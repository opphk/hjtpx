package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/redis"
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

type TokenBucket struct {
	tokens        float64
	lastRefill    time.Time
	capacity      int64
	refillRate    float64
	maxTokens     float64
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

type RateLimitConfig struct {
	MaxRequests int
	WindowSecs  int
}

type RateLimitResult struct {
	Allowed   bool
	Remaining int
	ResetAt   time.Time
	RetryAfter int
}

type DistributedRateLimitConfig struct {
	RedisEnabled    bool
	ConsistencyLevel string
	Nodes           []string
	SyncInterval    time.Duration
}

type DistributedRateLimitService struct {
	localService *SlidingWindowRateLimitService
	redisEnabled bool
	config       DistributedRateLimitConfig
	nodeID       string
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

func NewTokenBucketRateLimitServiceWithConfig(config TokenBucketConfig) *TokenBucketRateLimitService {
	if config.RefillRate == 0 {
		config.RefillRate = 10
	}
	if config.RefillPerSec == 0 {
		config.RefillPerSec = 10
	}

	return &TokenBucketRateLimitService{
		config:  config,
		buckets: make(map[string]*TokenBucket),
	}
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

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

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

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

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
		Remaining:  remaining,
		ResetAt:    now.Add(time.Duration(float64(config.Capacity)/config.RefillRate) * time.Second),
		RetryAfter: retryAfter,
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
		Remaining:  int(remaining),
		ResetAt:    resetAt,
		RetryAfter: retryAfter,
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

func NewDistributedRateLimitService(configs ...DistributedRateLimitConfig) *DistributedRateLimitService {
	cfg := DistributedRateLimitConfig{
		RedisEnabled:    redis.Client != nil,
		ConsistencyLevel: "eventual",
		SyncInterval:    100 * time.Millisecond,
	}

	if len(configs) > 0 {
		cfg = configs[0]
	}

	service := &DistributedRateLimitService{
		localService: NewSlidingWindowRateLimitService(),
		redisEnabled: cfg.RedisEnabled && redis.Client != nil,
		config:       cfg,
		nodeID:       fmt.Sprintf("node-%d", time.Now().UnixNano()),
	}

	if service.redisEnabled {
		go service.syncRoutine()
	}

	return service
}

func (s *DistributedRateLimitService) Allow(ctx context.Context, key string, maxRequests int64) (bool, error) {
	if !s.redisEnabled {
		result, err := s.localService.CheckSlidingWindow(ctx, key, maxRequests)
		if err != nil {
			return false, err
		}
		return result.Allowed, nil
	}

	redisKey := fmt.Sprintf("ratelimit:distributed:%s", key)

	allowed, err := s.redisAllow(ctx, redisKey, maxRequests)
	if err != nil {
		result, err := s.localService.CheckSlidingWindow(ctx, key, maxRequests)
		if err != nil {
			return false, err
		}
		return result.Allowed, nil
	}

	return allowed, nil
}

func (s *DistributedRateLimitService) redisAllow(ctx context.Context, key string, maxRequests int64) (bool, error) {
	if redis.Client == nil {
		return false, fmt.Errorf("redis not available")
	}

	now := time.Now()
	windowStart := now.Truncate(time.Minute)
	windowKey := fmt.Sprintf("%s:%d", key, windowStart.Unix())

	count, err := redis.Client.Incr(ctx, windowKey).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		redis.Client.Expire(ctx, windowKey, 2*time.Minute)
	}

	return count <= maxRequests, nil
}

func (s *DistributedRateLimitService) Check(ctx context.Context, key string, maxRequests int64) (*RateLimitResult, error) {
	if !s.redisEnabled {
		return s.localService.CheckSlidingWindow(ctx, key, maxRequests)
	}

	redisKey := fmt.Sprintf("ratelimit:distributed:%s", key)
	windowKey := fmt.Sprintf("%s:%d", redisKey, time.Now().Truncate(time.Minute).Unix())

	count, err := redis.Client.Get(ctx, windowKey).Int64()
	if err != nil && err.Error() != "redis: nil" {
		return s.localService.CheckSlidingWindow(ctx, key, maxRequests)
	}

	remaining := maxRequests - count
	if remaining < 0 {
		remaining = 0
	}

	ttl, _ := redis.Client.TTL(ctx, windowKey).Result()
	resetAt := time.Now().Add(ttl)

	retryAfter := 0
	if remaining == 0 {
		retryAfter = int(ttl.Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
	}

	return &RateLimitResult{
		Allowed:    count < maxRequests,
		Remaining:  int(remaining),
		ResetAt:    resetAt,
		RetryAfter: retryAfter,
	}, nil
}

func (s *DistributedRateLimitService) Reset(ctx context.Context, key string) error {
	if !s.redisEnabled {
		s.localService.Reset(key)
		return nil
	}

	redisKey := fmt.Sprintf("ratelimit:distributed:%s", key)

	pattern := redisKey + ":*"
	keys, err := redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return redis.Client.Del(ctx, keys...).Err()
	}

	return nil
}

func (s *DistributedRateLimitService) syncRoutine() {
	if !s.redisEnabled || redis.Client == nil {
		return
	}

	ticker := time.NewTicker(s.config.SyncInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.syncLocalToRedis()
	}
}

func (s *DistributedRateLimitService) syncLocalToRedis() {
	if redis.Client == nil {
		return
	}

	ctx := context.Background()
	stats := s.localService.GetStats()
	windows, ok := stats["windows"].(map[string]interface{})
	if !ok {
		return
	}

	for key := range windows {
		result, _ := s.localService.CheckSlidingWindow(ctx, key, 0)
		if result != nil {
			_ = result
		}
	}
}

func (s *DistributedRateLimitService) GetNodeID() string {
	return s.nodeID
}

func (s *DistributedRateLimitService) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"redis_enabled":     s.redisEnabled,
		"node_id":           s.nodeID,
		"consistency_level": s.config.ConsistencyLevel,
		"local_stats":       s.localService.GetStats(),
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

type AdaptiveRateLimitService struct {
	tokenBucket *TokenBucketRateLimitService
	slidingWindow *SlidingWindowRateLimitService
	distributed *DistributedRateLimitService
	config      AdaptiveRateLimitConfig
	mu          sync.RWMutex
}

type AdaptiveRateLimitConfig struct {
	BaseLimit      int64
	PeakLimit      int64
	OffPeakLimit   int64
	OffPeakStart   int
	OffPeakEnd     int
	EnableDynamic  bool
	CooldownPeriod time.Duration
}

func NewAdaptiveRateLimitService() *AdaptiveRateLimitService {
	return &AdaptiveRateLimitService{
		tokenBucket:   NewTokenBucketRateLimitService(),
		slidingWindow: NewSlidingWindowRateLimitService(),
		distributed:   NewDistributedRateLimitService(),
		config: AdaptiveRateLimitConfig{
			BaseLimit:      100,
			PeakLimit:      200,
			OffPeakLimit:   500,
			OffPeakStart:   0,
			OffPeakEnd:     6,
			EnableDynamic:  true,
			CooldownPeriod: 1 * time.Minute,
		},
	}
}

func (s *AdaptiveRateLimitService) Allow(ctx context.Context, key string) (bool, error) {
	s.mu.RLock()
	limit := s.calculateDynamicLimit()
	s.mu.RUnlock()

	return s.distributed.Allow(ctx, key, limit)
}

func (s *AdaptiveRateLimitService) calculateDynamicLimit() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hour := time.Now().Hour()

	if s.config.EnableDynamic {
		if hour >= s.config.OffPeakStart && hour < s.config.OffPeakEnd {
			return s.config.OffPeakLimit
		}
	}

	if isPeakHour(hour) {
		return s.config.PeakLimit
	}

	return s.config.BaseLimit
}

func isPeakHour(hour int) bool {
	return (hour >= 9 && hour <= 11) || (hour >= 14 && hour <= 17)
}

func (s *AdaptiveRateLimitService) UpdateConfig(config AdaptiveRateLimitConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

func (s *AdaptiveRateLimitService) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"token_bucket": s.tokenBucket.GetStats(),
		"sliding_window": s.slidingWindow.GetStats(),
		"distributed":  s.distributed.GetStats(),
		"config":       s.config,
	}
}
