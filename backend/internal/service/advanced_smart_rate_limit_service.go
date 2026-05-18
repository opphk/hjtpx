package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/internal/pkg/logger"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type LoadLevel int

const (
	LoadLevelLow    LoadLevel = iota
	LoadLevelNormal LoadLevel = iota
	LoadLevelMedium LoadLevel = iota
	LoadLevelHigh   LoadLevel = iota
	LoadLevelCritical LoadLevel = iota
)

func (l LoadLevel) String() string {
	switch l {
	case LoadLevelLow:
		return "low"
	case LoadLevelNormal:
		return "normal"
	case LoadLevelMedium:
		return "medium"
	case LoadLevelHigh:
		return "high"
	case LoadLevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}

type AdaptiveRateLimitConfig struct {
	BaseRate             float64
	BaseCapacity         float64
	MinCapacity          float64
	MaxCapacity          float64
	LoadCheckInterval    time.Duration
	AdjustmentInterval   time.Duration
	HighLoadThreshold    float64
	CriticalLoadThreshold float64
	CooldownPeriod       time.Duration
	RecoveryRate         float64
	LoadDecayFactor      float64
}

type AdaptiveRateLimitResult struct {
	Allowed       bool
	Tokens        float64
	Capacity      float64
	CurrentRate   float64
	LoadLevel     LoadLevel
	LoadFactor    float64
	RetryAfter    time.Duration
	Adjustment    string
}

type AdaptiveTokenBucket struct {
	mu         sync.Mutex
	key        string
	config     AdaptiveRateLimitConfig
	capacity   float64
	rate       float64
	tokens     float64
	lastRefill time.Time
	loadFactor float64
	loadHistory []float64
	muHistory   int
	adjustment  string
}

type AdaptiveRateLimitService struct {
	buckets       map[string]*AdaptiveTokenBucket
	mu            sync.RWMutex
	redisEnabled  bool
	config        AdaptiveRateLimitConfig
	loadLevel     atomic.Int64
	lastLoadCheck atomic.Value
	stopCh        chan struct{}
	wg            sync.WaitGroup
	nodeID        string
}

var defaultAdaptiveConfig = AdaptiveRateLimitConfig{
	BaseRate:              100,
	BaseCapacity:          1000,
	MinCapacity:           100,
	MaxCapacity:           5000,
	LoadCheckInterval:     5 * time.Second,
	AdjustmentInterval:    30 * time.Second,
	HighLoadThreshold:     0.7,
	CriticalLoadThreshold: 0.9,
	CooldownPeriod:        60 * time.Second,
	RecoveryRate:          0.1,
	LoadDecayFactor:       0.95,
}

func NewAdaptiveRateLimitService(configs ...AdaptiveRateLimitConfig) *AdaptiveRateLimitService {
	config := defaultAdaptiveConfig
	if len(configs) > 0 {
		config = configs[0]
	}

	service := &AdaptiveRateLimitService{
		buckets:      make(map[string]*AdaptiveTokenBucket),
		redisEnabled: redis.Client != nil,
		config:       config,
		stopCh:       make(chan struct{}),
		nodeID:       fmt.Sprintf("node-%d", time.Now().UnixNano()),
	}

	service.loadLevel.Store(int64(LoadLevelNormal))

	service.wg.Add(1)
	go service.loadMonitor()

	service.wg.Add(1)
	go service.adjustmentWorker()

	return service
}

func (s *AdaptiveRateLimitService) loadMonitor() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.config.LoadCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.updateLoadLevel()
		}
	}
}

func (s *AdaptiveRateLimitService) updateLoadLevel() {
	currentLoad := s.getCurrentSystemLoad()
	s.lastLoadCheck.Store(currentLoad)

	var level LoadLevel
	switch {
	case currentLoad >= s.config.CriticalLoadThreshold:
		level = LoadLevelCritical
	case currentLoad >= s.config.HighLoadThreshold:
		level = LoadLevelHigh
	case currentLoad >= s.config.HighLoadThreshold * 0.5:
		level = LoadLevelMedium
	case currentLoad < 0.3:
		level = LoadLevelLow
	default:
		level = LoadLevelNormal
	}

	oldLevel := LoadLevel(s.loadLevel.Load())
	if oldLevel != level {
		logger.Info("AdaptiveRateLimit: load level changed",
			logger.Fields{
				"from": oldLevel.String(),
				"to":   level.String(),
				"load": fmt.Sprintf("%.2f", currentLoad),
			})
	}
	s.loadLevel.Store(int64(level))
}

func (s *AdaptiveRateLimitService) getCurrentSystemLoad() float64 {
	if !s.redisEnabled {
		return 0.5
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	keys := []string{
		"ratelimit:*",
		"tokenbucket:*",
		"quota:*",
	}

	totalKeys := 0
	for _, pattern := range keys {
		count, err := redis.Client.Keys(ctx, pattern).Result()
		if err == nil {
			totalKeys += len(count)
		}
	}

	load := float64(totalKeys) / 10000.0
	if load > 1.0 {
		load = 1.0
	}

	return load
}

func (s *AdaptiveRateLimitService) adjustmentWorker() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.config.AdjustmentInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.adjustAllBuckets()
		}
	}
}

func (s *AdaptiveRateLimitService) adjustAllBuckets() {
	s.mu.Lock()
	defer s.mu.Unlock()

	level := LoadLevel(s.loadLevel.Load())

	for _, bucket := range s.buckets {
		bucket.mu.Lock()
		s.adjustBucketCapacity(bucket, level)
		bucket.mu.Unlock()
	}
}

func (s *AdaptiveRateLimitService) adjustBucketCapacity(bucket *AdaptiveTokenBucket, level LoadLevel) {
	var targetCapacity, targetRate float64
	now := time.Now()

	switch level {
	case LoadLevelCritical:
		targetCapacity = s.config.BaseCapacity * 0.3
		targetRate = s.config.BaseRate * 0.3
		bucket.adjustment = "critical_reduction"
	case LoadLevelHigh:
		targetCapacity = s.config.BaseCapacity * 0.5
		targetRate = s.config.BaseRate * 0.5
		bucket.adjustment = "high_reduction"
	case LoadLevelMedium:
		targetCapacity = s.config.BaseCapacity * 0.75
		targetRate = s.config.BaseRate * 0.75
		bucket.adjustment = "medium_reduction"
	case LoadLevelLow:
		elapsed := now.Sub(bucket.lastRefill).Seconds()
		if elapsed > s.config.CooldownPeriod.Seconds() {
			targetCapacity = math.Min(bucket.capacity*(1+s.config.RecoveryRate), s.config.MaxCapacity)
			targetRate = math.Min(bucket.rate*(1+s.config.RecoveryRate), s.config.BaseRate*1.5)
			bucket.adjustment = "recovery"
		} else {
			targetCapacity = bucket.capacity
			targetRate = bucket.rate
		}
	case LoadLevelNormal:
		targetCapacity = s.config.BaseCapacity
		targetRate = s.config.BaseRate
		bucket.adjustment = "normal"
	}

	bucket.capacity = targetCapacity
	bucket.rate = targetRate

	if bucket.tokens > bucket.capacity {
		bucket.tokens = bucket.capacity
	}
}

func (s *AdaptiveRateLimitService) getOrCreateBucket(key string) *AdaptiveTokenBucket {
	s.mu.RLock()
	bucket, exists := s.buckets[key]
	s.mu.RUnlock()

	if exists {
		return bucket
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if bucket, exists = s.buckets[key]; exists {
		return bucket
	}

	bucket = &AdaptiveTokenBucket{
		key:         key,
		config:      s.config,
		capacity:    s.config.BaseCapacity,
		rate:        s.config.BaseRate,
		tokens:      s.config.BaseCapacity,
		lastRefill:  time.Now(),
		loadFactor:  1.0,
		loadHistory: make([]float64, 0, 60),
	}
	s.buckets[key] = bucket
	return bucket
}

func (tb *AdaptiveTokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastRefill = now
}

func (tb *AdaptiveTokenBucket) tryConsume(tokens float64) *AdaptiveRateLimitResult {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	result := &AdaptiveRateLimitResult{
		Allowed:     false,
		Tokens:      tb.tokens,
		Capacity:    tb.capacity,
		CurrentRate: tb.rate,
		LoadLevel:   LoadLevelNormal,
		LoadFactor:  tb.loadFactor,
	}

	if tb.tokens >= tokens {
		tb.tokens -= tokens
		result.Allowed = true
		result.Tokens = tb.tokens
		result.Adjustment = "normal"
		return result
	}

	needed := tokens - tb.tokens
	result.RetryAfter = time.Duration(needed/tb.rate) * time.Second
	return result
}

func (s *AdaptiveRateLimitService) CheckRateLimit(ctx context.Context, key string) (*AdaptiveRateLimitResult, error) {
	return s.CheckRateLimitWithTokens(ctx, key, 1)
}

func (s *AdaptiveRateLimitService) CheckRateLimitWithTokens(ctx context.Context, key string, tokens float64) (*AdaptiveRateLimitResult, error) {
	if s.redisEnabled {
		return s.checkRateLimitRedis(ctx, key, tokens)
	}
	return s.checkRateLimitLocal(key, tokens)
}

func (s *AdaptiveRateLimitService) checkRateLimitLocal(key string, tokens float64) (*AdaptiveRateLimitResult, error) {
	bucket := s.getOrCreateBucket(key)
	level := LoadLevel(s.loadLevel.Load())
	
	bucket.mu.Lock()
	s.adjustBucketCapacity(bucket, level)
	result := bucket.tryConsume(tokens)
	result.LoadLevel = level
	bucket.mu.Unlock()

	return result, nil
}

func (s *AdaptiveRateLimitService) checkRateLimitRedis(ctx context.Context, key string, tokens float64) (*AdaptiveRateLimitResult, error) {
	redisKey := fmt.Sprintf("adaptive:ratelimit:%s", key)
	now := time.Now().UnixNano()
	level := LoadLevel(s.loadLevel.Load())

	var rate, capacity float64
	switch level {
	case LoadLevelCritical:
		rate = s.config.BaseRate * 0.3
		capacity = s.config.BaseCapacity * 0.3
	case LoadLevelHigh:
		rate = s.config.BaseRate * 0.5
		capacity = s.config.BaseCapacity * 0.5
	case LoadLevelMedium:
		rate = s.config.BaseRate * 0.75
		capacity = s.config.BaseCapacity * 0.75
	case LoadLevelLow:
		rate = s.config.BaseRate * 1.5
		capacity = s.config.BaseCapacity * 1.2
	default:
		rate = s.config.BaseRate
		capacity = s.config.BaseCapacity
	}

	script := `
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local capacity = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local tokensToConsume = tonumber(ARGV[4])

		local data = redis.call('GET', key)
		local tokens, lastRefill
		if data then
			local parts = {}
			for part in string.gmatch(data, "([^|]+)") do
				table.insert(parts, part)
			end
			tokens = tonumber(parts[1])
			lastRefill = tonumber(parts[2])
		else
			tokens = capacity
			lastRefill = now
		end

		local elapsed = (now - lastRefill) / 1000000000.0
		tokens = tokens + elapsed * rate
		if tokens > capacity then
			tokens = capacity
		end

		local allowed = 0
		local waitTime = 0

		if tokens >= tokensToConsume then
			tokens = tokens - tokensToConsume
			allowed = 1
		else
			local needed = tokensToConsume - tokens
			waitTime = needed / rate
		end

		redis.call('SET', key, string.format("%.4f|%d", tokens, now))
		redis.call('EXPIRE', key, 3600)

		return {allowed, tokens, capacity, waitTime}
	`

	result, err := redis.Client.Eval(ctx, script, []string{redisKey}, rate, capacity, now, tokens).Result()
	if err != nil {
		return &AdaptiveRateLimitResult{Allowed: true}, nil
	}

	values := result.([]interface{})
	allowed := values[0].(int64) == 1
	remainingTokens := values[1].(float64)
	cap := values[2].(float64)
	waitTime := values[3].(float64)

	return &AdaptiveRateLimitResult{
		Allowed:     allowed,
		Tokens:      remainingTokens,
		Capacity:    cap,
		CurrentRate: rate,
		LoadLevel:   level,
		LoadFactor:  float64(level) / 5.0,
		RetryAfter:  time.Duration(waitTime) * time.Second,
		Adjustment:  level.String(),
	}, nil
}

func (s *AdaptiveRateLimitService) CheckIPRateLimit(ctx context.Context, ip string) (*AdaptiveRateLimitResult, error) {
	key := fmt.Sprintf("ip:%s", ip)
	return s.CheckRateLimit(ctx, key)
}

func (s *AdaptiveRateLimitService) CheckUserRateLimit(ctx context.Context, userID uint) (*AdaptiveRateLimitResult, error) {
	key := fmt.Sprintf("user:%d", userID)
	return s.CheckRateLimit(ctx, key)
}

func (s *AdaptiveRateLimitService) CheckAppRateLimit(ctx context.Context, appID uint) (*AdaptiveRateLimitResult, error) {
	key := fmt.Sprintf("app:%d", appID)
	return s.CheckRateLimit(ctx, key)
}

func (s *AdaptiveRateLimitService) GetLoadLevel() LoadLevel {
	return LoadLevel(s.loadLevel.Load())
}

func (s *AdaptiveRateLimitService) GetLoadFactor() float64 {
	return float64(s.loadLevel.Load()) / 5.0
}

func (s *AdaptiveRateLimitService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bucketCount := len(s.buckets)
	totalTokens := float64(0)
	
	for _, bucket := range s.buckets {
		bucket.mu.Lock()
		totalTokens += bucket.tokens
		bucket.mu.Unlock()
	}

	return map[string]interface{}{
		"bucket_count":    bucketCount,
		"total_tokens":    totalTokens,
		"load_level":      s.GetLoadLevel().String(),
		"load_factor":     s.GetLoadFactor(),
		"current_load":    s.lastLoadCheck.Load(),
		"base_rate":       s.config.BaseRate,
		"base_capacity":   s.config.BaseCapacity,
		"node_id":         s.nodeID,
	}
}

func (s *AdaptiveRateLimitService) ResetBucket(ctx context.Context, key string) error {
	s.mu.Lock()
	delete(s.buckets, key)
	s.mu.Unlock()

	if s.redisEnabled {
		redisKey := fmt.Sprintf("adaptive:ratelimit:%s", key)
		return redis.Client.Del(ctx, redisKey).Err()
	}
	return nil
}

func (s *AdaptiveRateLimitService) UpdateConfig(config AdaptiveRateLimitConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

func (s *AdaptiveRateLimitService) Close() {
	close(s.stopCh)
	s.wg.Wait()
}
