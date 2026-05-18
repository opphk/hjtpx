package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/redis"
)

// TokenBucketConfig 令牌桶配置
type TokenBucketConfig struct {
	Rate          float64 // 每秒填充的令牌数
	Capacity      float64 // 桶的最大容量
	BurstSize     float64 // 突发流量大小
	InitialTokens float64 // 初始令牌数
}

// TokenBucketResult 令牌桶限流结果
type TokenBucketResult struct {
	Allowed    bool          // 是否允许请求
	Tokens     float64       // 当前令牌数
	Capacity   float64       // 桶容量
	RetryAfter time.Duration // 重试等待时间
	WaitTime   time.Duration // 建议等待时间
	IsBurst    bool          // 是否为突发请求
}

// TokenBucket 令牌桶结构
type TokenBucket struct {
	mu         sync.Mutex
	key        string
	capacity   float64
	rate       float64
	tokens     float64
	lastRefill time.Time
	burstSize  float64
}

// TokenBucketRateLimitService 令牌桶限流服务
type TokenBucketRateLimitService struct {
	buckets      map[string]*TokenBucket
	mu           sync.RWMutex
	redisEnabled bool
}

const (
	tokenBucketPrefix = "tokenbucket:"
)

var defaultTokenBucketConfig = TokenBucketConfig{
	Rate:          10,
	Capacity:      100,
	BurstSize:     50,
	InitialTokens: 100,
}

// NewTokenBucketRateLimitService 创建令牌桶限流服务
func NewTokenBucketRateLimitService() *TokenBucketRateLimitService {
	service := &TokenBucketRateLimitService{
		buckets:      make(map[string]*TokenBucket),
		redisEnabled: redis.Client != nil,
	}
	go service.cleanupExpiredBuckets()
	return service
}

// getBucket 获取或创建令牌桶
func (s *TokenBucketRateLimitService) getBucket(key string, config *TokenBucketConfig) *TokenBucket {
	s.mu.RLock()
	bucket, exists := s.buckets[key]
	s.mu.RUnlock()

	if !exists {
		s.mu.Lock()
		defer s.mu.Unlock()
		// 再次检查，避免竞态条件
		if bucket, exists = s.buckets[key]; !exists {
			bucket = &TokenBucket{
				key:        key,
				capacity:   config.Capacity,
				rate:       config.Rate,
				tokens:     config.InitialTokens,
				lastRefill: time.Now(),
				burstSize:  config.BurstSize,
			}
			s.buckets[key] = bucket
		}
	}
	return bucket
}

// refill 填充令牌
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastRefill = now
}

// tryConsume 尝试消耗令牌
func (tb *TokenBucket) tryConsume(tokens float64) *TokenBucketResult {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	result := &TokenBucketResult{
		Allowed:  false,
		Tokens:   tb.tokens,
		Capacity: tb.capacity,
		IsBurst:  false,
	}

	// 正常令牌消费
	if tb.tokens >= tokens {
		tb.tokens -= tokens
		result.Allowed = true
		result.Tokens = tb.tokens
		return result
	}

	// 尝试突发处理
	if tb.tokens+tb.burstSize >= tokens {
		remaining := tokens - tb.tokens
		tb.tokens = 0
		tb.burstSize -= remaining
		result.Allowed = true
		result.IsBurst = true
		result.Tokens = tb.tokens
		result.RetryAfter = time.Duration(remaining/tb.rate) * time.Second
		return result
	}

	// 计算需要等待的时间
	needed := tokens - tb.tokens
	result.WaitTime = time.Duration(needed/tb.rate) * time.Second
	result.RetryAfter = result.WaitTime
	return result
}

// CheckTokenBucketRateLimit 检查令牌桶限流（基于内存）
func (s *TokenBucketRateLimitService) CheckTokenBucketRateLimit(
	ctx context.Context,
	key string,
	config *TokenBucketConfig,
) (*TokenBucketResult, error) {
	if config == nil {
		config = &defaultTokenBucketConfig
	}
	bucketKey := tokenBucketPrefix + key
	bucket := s.getBucket(bucketKey, config)
	return bucket.tryConsume(1), nil
}

// CheckTokenBucketRateLimitRedis 检查令牌桶限流（基于 Redis）
func (s *TokenBucketRateLimitService) CheckTokenBucketRateLimitRedis(
	ctx context.Context,
	key string,
	config *TokenBucketConfig,
) (*TokenBucketResult, error) {
	if !s.redisEnabled {
		return s.CheckTokenBucketRateLimit(ctx, key, config)
	}

	if config == nil {
		config = &defaultTokenBucketConfig
	}

	redisKey := tokenBucketPrefix + key
	now := time.Now().UnixNano()

	// 使用 Lua 脚本保证原子性
	script := `
	local key = KEYS[1]
	local rate = tonumber(ARGV[1])
	local capacity = tonumber(ARGV[2])
	local burstSize = tonumber(ARGV[3])
	local now = tonumber(ARGV[4])
	local tokensToConsume = tonumber(ARGV[5])

	local data = redis.call('GET', key)
	local tokens, lastRefill, currentBurst
	if data then
		local parts = {}
		for part in string.gmatch(data, "([^|]+)") do
			table.insert(parts, part)
		end
		tokens = tonumber(parts[1])
		lastRefill = tonumber(parts[2])
		currentBurst = tonumber(parts[3])
	else
		tokens = capacity
		lastRefill = now
		currentBurst = burstSize
	end

	local elapsed = (now - lastRefill) / 1000000000.0
	tokens = tokens + elapsed * rate
	if tokens > capacity then
		tokens = capacity
	end

	local allowed = 0
	local isBurst = 0
	local waitTime = 0

	if tokens >= tokensToConsume then
		tokens = tokens - tokensToConsume
		allowed = 1
	elseif tokens + currentBurst >= tokensToConsume then
		local needed = tokensToConsume - tokens
		tokens = 0
		currentBurst = currentBurst - needed
		allowed = 1
		isBurst = 1
		waitTime = needed / rate
	else
		local needed = tokensToConsume - tokens
		waitTime = needed / rate
	end

	local newData = string.format("%.4f|%d|%.4f", tokens, now, currentBurst)
	redis.call('SET', key, newData)
	redis.call('EXPIRE', key, 3600)

	return {allowed, tokens, waitTime, isBurst}
`

	result, err := redis.Client.Eval(ctx, script, []string{redisKey},
		config.Rate, config.Capacity, config.BurstSize, now, 1.0).Result()

	if err != nil {
		return &TokenBucketResult{Allowed: true}, nil
	}

	values := result.([]interface{})
	allowed := values[0].(int64) == 1
	tokens := values[1].(float64)
	waitTime := values[2].(float64)
	isBurst := values[3].(int64) == 1

	return &TokenBucketResult{
		Allowed:    allowed,
		Tokens:     tokens,
		Capacity:   config.Capacity,
		RetryAfter: time.Duration(waitTime) * time.Second,
		WaitTime:   time.Duration(waitTime) * time.Second,
		IsBurst:    isBurst,
	}, nil
}

// CheckIPTokenBucketLimit IP 级别的令牌桶限流
func (s *TokenBucketRateLimitService) CheckIPTokenBucketLimit(
	ctx context.Context,
	ip string,
	config *TokenBucketConfig,
) (*TokenBucketResult, error) {
	key := fmt.Sprintf("ip:%s", ip)
	return s.CheckTokenBucketRateLimitRedis(ctx, key, config)
}

// CheckUserTokenBucketLimit 用户级别的令牌桶限流
func (s *TokenBucketRateLimitService) CheckUserTokenBucketLimit(
	ctx context.Context,
	userID uint,
	config *TokenBucketConfig,
) (*TokenBucketResult, error) {
	key := fmt.Sprintf("user:%d", userID)
	return s.CheckTokenBucketRateLimitRedis(ctx, key, config)
}

// CheckAppTokenBucketLimit 应用级别的令牌桶限流
func (s *TokenBucketRateLimitService) CheckAppTokenBucketLimit(
	ctx context.Context,
	appID uint,
	config *TokenBucketConfig,
) (*TokenBucketResult, error) {
	key := fmt.Sprintf("app:%d", appID)
	return s.CheckTokenBucketRateLimitRedis(ctx, key, config)
}

// ResetBucket 重置令牌桶
func (s *TokenBucketRateLimitService) ResetBucket(ctx context.Context, key string) error {
	bucketKey := tokenBucketPrefix + key

	s.mu.Lock()
	delete(s.buckets, bucketKey)
	s.mu.Unlock()

	if s.redisEnabled {
		return redis.Client.Del(ctx, bucketKey).Err()
	}
	return nil
}

// GetBucketStats 获取桶统计信息
func (s *TokenBucketRateLimitService) GetBucketStats(key string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bucketKey := tokenBucketPrefix + key
	if bucket, exists := s.buckets[bucketKey]; exists {
		bucket.mu.Lock()
		defer bucket.mu.Unlock()
		bucket.refill()
		return map[string]interface{}{
			"key":         bucket.key,
			"tokens":      bucket.tokens,
			"capacity":    bucket.capacity,
			"rate":        bucket.rate,
			"burst_size":  bucket.burstSize,
			"last_refill": bucket.lastRefill,
		}
	}
	return nil
}

// cleanupExpiredBuckets 清理过期的桶
func (s *TokenBucketRateLimitService) cleanupExpiredBuckets() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, bucket := range s.buckets {
			bucket.mu.Lock()
			if now.Sub(bucket.lastRefill) > 1*time.Hour {
				delete(s.buckets, key)
			}
			bucket.mu.Unlock()
		}
		s.mu.Unlock()
	}
}

// TrafficShaper 流量整形器
type TrafficShaper struct {
	queue  chan func()
	bucket *TokenBucket
	wg     sync.WaitGroup
	closed bool
	mu     sync.Mutex
}

// NewTrafficShaper 创建流量整形器
func NewTrafficShaper(config *TokenBucketConfig) *TrafficShaper {
	if config == nil {
		config = &defaultTokenBucketConfig
	}
	shaper := &TrafficShaper{
		queue: make(chan func(), 1000),
		bucket: &TokenBucket{
			capacity:   config.Capacity,
			rate:       config.Rate,
			tokens:     config.InitialTokens,
			lastRefill: time.Now(),
			burstSize:  config.BurstSize,
		},
	}
	shaper.wg.Add(1)
	go shaper.processQueue()
	return shaper
}

// Submit 提交任务
func (ts *TrafficShaper) Submit(task func()) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.closed {
		return false
	}
	select {
	case ts.queue <- task:
		return true
	default:
		return false
	}
}

// processQueue 处理队列
func (ts *TrafficShaper) processQueue() {
	defer ts.wg.Done()
	for task := range ts.queue {
		for {
			result := ts.bucket.tryConsume(1)
			if result.Allowed {
				task()
				break
			}
			time.Sleep(result.WaitTime)
		}
	}
}

// Close 关闭流量整形器
func (ts *TrafficShaper) Close() {
	ts.mu.Lock()
	if !ts.closed {
		ts.closed = true
		close(ts.queue)
	}
	ts.mu.Unlock()
	ts.wg.Wait()
}
