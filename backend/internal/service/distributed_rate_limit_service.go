package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/internal/pkg/logger"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

// DistributedRateLimitService 分布式限流服务
// 支持四种限流算法：
// 1. 固定窗口算法（Fixed Window）：将时间划分为固定窗口，统计窗口内请求数
// 2. 滑动窗口算法（Sliding Window）：使用Redis有序集合实现精确的滑动窗口计数
// 3. 令牌桶算法（Token Bucket）：以固定速率向桶中添加令牌，支持突发流量
// 4. 漏桶算法（Leaky Bucket）：以固定速率处理请求，超出容量的请求排队或拒绝
// 
// 特性：
// - 本地模式：当Redis不可用时自动降级到本地内存计数
// - 多节点同步：通过Redis实现跨节点状态同步
// - 原子操作：使用Redis Pipeline保证计数准确性

type DistributedRateLimitType int

const (
	DistributedFixedWindow   DistributedRateLimitType = iota
	DistributedSlidingWindow DistributedRateLimitType = iota
	DistributedTokenBucket   DistributedRateLimitType = iota
	DistributedLeakyBucket   DistributedRateLimitType = iota
)

type DistributedRateLimitConfig struct {
	Type            DistributedRateLimitType
	MaxRequests     int
	WindowSecs      int
	RedisKeyPrefix  string
	NodeID          string
	SyncInterval    time.Duration
	ConsistencyMode bool
}

type DistributedRateLimitResult struct {
	Allowed     bool
	Remaining   int
	ResetAt     time.Time
	TotalCount  int64
	NodeID      string
	GlobalCount int64
}

type DistributedRateLimitService struct {
	config       DistributedRateLimitConfig
	redisEnabled bool
	nodeID       string
	counter      *sync.Map
	mu           sync.Mutex
	stopCh       chan struct{}
	wg           sync.WaitGroup
	sequence     atomic.Int64
}

type localCounter struct {
	count    int64
	windowStart time.Time
	mu        sync.Mutex
}

func NewDistributedRateLimitService(configs ...DistributedRateLimitConfig) *DistributedRateLimitService {
	config := DistributedRateLimitConfig{
		Type:            DistributedTokenBucket,
		MaxRequests:     100,
		WindowSecs:      60,
		RedisKeyPrefix:  "dist:ratelimit:",
		NodeID:          fmt.Sprintf("node-%d", time.Now().UnixNano()),
		SyncInterval:    5 * time.Second,
		ConsistencyMode: false,
	}
	if len(configs) > 0 {
		config = configs[0]
	}

	service := &DistributedRateLimitService{
		config:      config,
		redisEnabled: redis.Client != nil,
		nodeID:      config.NodeID,
		counter:     &sync.Map{},
		stopCh:      make(chan struct{}),
	}

	service.wg.Add(1)
	go service.syncWorker()

	return service
}

func (s *DistributedRateLimitService) syncWorker() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.cleanupExpiredCounters()
		}
	}
}

func (s *DistributedRateLimitService) cleanupExpiredCounters() {
	now := time.Now()
	windowDuration := time.Duration(s.config.WindowSecs) * time.Second

	s.counter.Range(func(key, value interface{}) bool {
		counter := value.(*localCounter)
		counter.mu.Lock()
		if now.Sub(counter.windowStart) > windowDuration*2 {
			s.counter.Delete(key)
		}
		counter.mu.Unlock()
		return true
	})
}

func (s *DistributedRateLimitService) getLocalCounter(key string) *localCounter {
	counter, exists := s.counter.Load(key)
	if !exists {
		counter, _ = s.counter.LoadOrStore(key, &localCounter{
			count:       0,
			windowStart: time.Now(),
		})
	}
	return counter.(*localCounter)
}

func (s *DistributedRateLimitService) CheckRateLimit(ctx context.Context, key string) (*DistributedRateLimitResult, error) {
	return s.CheckRateLimitWithCount(ctx, key, 1)
}

func (s *DistributedRateLimitService) CheckRateLimitWithCount(ctx context.Context, key string, count int) (*DistributedRateLimitResult, error) {
	seq := s.sequence.Add(1)
	requestID := fmt.Sprintf("%s:%s:%d", s.nodeID, key, seq)

	switch s.config.Type {
	case DistributedFixedWindow:
		return s.checkFixedWindow(ctx, key, count, requestID)
	case DistributedSlidingWindow:
		return s.checkSlidingWindow(ctx, key, count, requestID)
	case DistributedTokenBucket:
		return s.checkDistributedTokenBucket(ctx, key, count, requestID)
	case DistributedLeakyBucket:
		return s.checkLeakyBucket(ctx, key, count, requestID)
	default:
		return s.checkDistributedTokenBucket(ctx, key, count, requestID)
	}
}

func (s *DistributedRateLimitService) checkFixedWindow(ctx context.Context, key string, count int, requestID string) (*DistributedRateLimitResult, error) {
	redisKey := s.config.RedisKeyPrefix + "fixed:" + key
	now := time.Now()
	windowStart := now.Truncate(time.Duration(s.config.WindowSecs) * time.Second)
	windowKey := fmt.Sprintf("%s:%d", redisKey, windowStart.Unix())

	result := &DistributedRateLimitResult{
		Allowed:   true,
		Remaining: s.config.MaxRequests - count,
		ResetAt:   windowStart.Add(time.Duration(s.config.WindowSecs) * time.Second),
		NodeID:    s.nodeID,
	}

	if !s.redisEnabled {
		counter := s.getLocalCounter(windowKey)
		counter.mu.Lock()
		if now.Sub(counter.windowStart) > time.Duration(s.config.WindowSecs)*time.Second {
			counter.count = 0
			counter.windowStart = windowStart
		}
		newCount := atomic.AddInt64(&counter.count, int64(count))
		counter.mu.Unlock()

		result.TotalCount = newCount
		if int(newCount) > s.config.MaxRequests {
			result.Allowed = false
			result.Remaining = 0
		} else {
			result.Remaining = s.config.MaxRequests - int(newCount)
		}
		return result, nil
	}

	pipe := redis.Client.Pipeline()
	incrCmd := pipe.IncrBy(ctx, windowKey, int64(count))
	pipe.Expire(ctx, windowKey, time.Duration(s.config.WindowSecs*2)*time.Second)
	_, err := pipe.Exec(ctx)
	if err != nil {
		logger.Error("DistributedRateLimit: failed to check fixed window",
			logger.Fields{"error": err.Error()})
		return result, nil
	}

	result.TotalCount = incrCmd.Val()
	result.GlobalCount = result.TotalCount

	if int(result.TotalCount) > s.config.MaxRequests {
		result.Allowed = false
		result.Remaining = 0
	} else {
		result.Remaining = s.config.MaxRequests - int(result.TotalCount)
	}

	return result, nil
}

func (s *DistributedRateLimitService) checkSlidingWindow(ctx context.Context, key string, count int, requestID string) (*DistributedRateLimitResult, error) {
	redisKey := s.config.RedisKeyPrefix + "sliding:" + key
	now := time.Now()
	_ = now.Add(-time.Duration(s.config.WindowSecs) * time.Second)

	result := &DistributedRateLimitResult{
		Allowed:   true,
		Remaining: s.config.MaxRequests - count,
		ResetAt:   now.Add(time.Duration(s.config.WindowSecs) * time.Second),
		NodeID:    s.nodeID,
	}

	if !s.redisEnabled {
		counter := s.getLocalCounter(redisKey)
		counter.mu.Lock()
		if now.Sub(counter.windowStart) > time.Duration(s.config.WindowSecs)*time.Second {
			counter.count = 0
			counter.windowStart = now
		}
		newCount := atomic.AddInt64(&counter.count, int64(count))
		counter.mu.Unlock()

		result.TotalCount = newCount
		if int(newCount) > s.config.MaxRequests {
			result.Allowed = false
			result.Remaining = 0
		} else {
			result.Remaining = s.config.MaxRequests - int(newCount)
		}
		return result, nil
	}

	script := `
		local key = KEYS[1]
		local windowMs = tonumber(ARGV[1])
		local now = tonumber(ARGV[2])
		local maxRequests = tonumber(ARGV[3])
		local count = tonumber(ARGV[4])

		redis.call('ZREMRANGEBYSCORE', key, 0, now - windowMs)
		local currentCount = redis.call('ZCARD', key)

		local allowed = 1
		if currentCount + count > maxRequests then
			allowed = 0
		else
			for i = 1, count do
				redis.call('ZADD', key, now + i, now .. ':' .. i)
			end
		end

		redis.call('EXPIRE', key, windowMs / 1000 + 10)

		return {allowed, maxRequests - currentCount - count, currentCount + count}
	`

	execResult, err := redis.Client.Eval(ctx, script, []string{redisKey},
		s.config.WindowSecs*1000, now.UnixMilli(), s.config.MaxRequests, count).Result()
	if err != nil {
		logger.Error("DistributedRateLimit: failed to check sliding window",
			logger.Fields{"error": err.Error()})
		return result, nil
	}

	values := execResult.([]interface{})
	allowed := values[0].(int64) == 1
	remaining := values[1].(int64)
	totalCount := values[2].(int64)

	result.Allowed = allowed
	result.Remaining = int(remaining)
	result.TotalCount = totalCount
	result.GlobalCount = totalCount

	return result, nil
}

func (s *DistributedRateLimitService) checkDistributedTokenBucket(ctx context.Context, key string, count int, requestID string) (*DistributedRateLimitResult, error) {
	redisKey := s.config.RedisKeyPrefix + "token:" + key
	now := time.Now()
	rate := float64(s.config.MaxRequests) / float64(s.config.WindowSecs)

	result := &DistributedRateLimitResult{
		Allowed:   true,
		Remaining: s.config.MaxRequests - count,
		ResetAt:   now.Add(time.Duration(s.config.WindowSecs) * time.Second),
		NodeID:    s.nodeID,
	}

	if !s.redisEnabled {
		counter := s.getLocalCounter(redisKey)
		counter.mu.Lock()
		elapsed := now.Sub(counter.windowStart).Seconds()
		tokens := float64(counter.count) + elapsed*rate
		if tokens > float64(s.config.MaxRequests) {
			tokens = float64(s.config.MaxRequests)
		}

		if tokens >= float64(count) {
			counter.count = int64(tokens) - int64(count)
			counter.windowStart = now
			result.Remaining = int(tokens) - count
		} else {
			result.Allowed = false
			result.Remaining = 0
		}
		counter.mu.Unlock()
		return result, nil
	}

	script := `
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local capacity = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local count = tonumber(ARGV[4])

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

		local allowed = 1
		if tokens >= count then
			tokens = tokens - count
		else
			allowed = 0
		end

		redis.call('SET', key, string.format("%.4f|%d", tokens, now))
		redis.call('EXPIRE', key, 3600)

		return {allowed, tokens, capacity}
	`

	execResult, err := redis.Client.Eval(ctx, script, []string{redisKey}, rate, s.config.MaxRequests, now, count).Result()
	if err != nil {
		logger.Error("DistributedRateLimit: failed to check token bucket",
			logger.Fields{"error": err.Error()})
		return result, nil
	}

	values := execResult.([]interface{})
	allowed := values[0].(int64) == 1
	tokens := values[1].(float64)
	capacity := values[2].(float64)

	result.Allowed = allowed
	result.Remaining = int(tokens)
	result.TotalCount = int64(capacity - tokens)
	result.GlobalCount = result.TotalCount

	return result, nil
}

func (s *DistributedRateLimitService) checkLeakyBucket(ctx context.Context, key string, count int, requestID string) (*DistributedRateLimitResult, error) {
	redisKey := s.config.RedisKeyPrefix + "leaky:" + key
	now := time.Now().UnixMilli()
	leakRate := float64(s.config.WindowSecs*1000) / float64(s.config.MaxRequests)

	result := &DistributedRateLimitResult{
		Allowed:   true,
		Remaining: s.config.MaxRequests,
		ResetAt:   time.Now().Add(time.Duration(s.config.WindowSecs) * time.Second),
		NodeID:    s.nodeID,
	}

	if !s.redisEnabled {
		counter := s.getLocalCounter(redisKey)
		counter.mu.Lock()
		elapsed := now - counter.windowStart.UnixMilli()
		leaked := float64(elapsed) / leakRate
		currentCount := float64(counter.count) - leaked
		if currentCount < 0 {
			currentCount = 0
		}

		if currentCount+float64(count) <= float64(s.config.MaxRequests) {
			counter.count = int64(currentCount) + int64(count)
			counter.windowStart = time.Now()
			result.Remaining = s.config.MaxRequests - int(currentCount) - count
		} else {
			result.Allowed = false
			result.Remaining = 0
			waitTime := int64((currentCount + float64(count) - float64(s.config.MaxRequests)) * leakRate)
			result.ResetAt = time.Now().Add(time.Duration(waitTime) * time.Millisecond)
		}
		counter.mu.Unlock()
		return result, nil
	}

	script := `
		local key = KEYS[1]
		local leakRate = tonumber(ARGV[1])
		local capacity = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local count = tonumber(ARGV[4])

		local data = redis.call('GET', key)
		local waterLevel, lastTime
		if data then
			local parts = {}
			for part in string.gmatch(data, "([^|]+)") do
				table.insert(parts, part)
			end
			waterLevel = tonumber(parts[1])
			lastTime = tonumber(parts[2])
		else
			waterLevel = 0
			lastTime = now
		end

		local elapsed = now - lastTime
		local leaked = elapsed / leakRate
		waterLevel = math.max(0, waterLevel - leaked)

		local allowed = 1
		if waterLevel + count > capacity then
			allowed = 0
		else
			waterLevel = waterLevel + count
		end

		redis.call('SET', key, string.format("%.2f|%d", waterLevel, now))
		redis.call('EXPIRE', key, 3600)

		return {allowed, capacity - waterLevel, waterLevel}
	`

	execResult, err := redis.Client.Eval(ctx, script, []string{redisKey}, leakRate, s.config.MaxRequests, now, count).Result()
	if err != nil {
		logger.Error("DistributedRateLimit: failed to check leaky bucket",
			logger.Fields{"error": err.Error()})
		return result, nil
	}

	values := execResult.([]interface{})
	allowed := values[0].(int64) == 1
	remaining := values[1].(float64)
	waterLevel := values[2].(float64)

	result.Allowed = allowed
	result.Remaining = int(remaining)
	result.TotalCount = int64(waterLevel)
	result.GlobalCount = result.TotalCount

	return result, nil
}

func (s *DistributedRateLimitService) CheckIPRateLimit(ctx context.Context, ip string) (*DistributedRateLimitResult, error) {
	key := fmt.Sprintf("ip:%s", ip)
	return s.CheckRateLimit(ctx, key)
}

func (s *DistributedRateLimitService) CheckUserRateLimit(ctx context.Context, userID uint) (*DistributedRateLimitResult, error) {
	key := fmt.Sprintf("user:%d", userID)
	return s.CheckRateLimit(ctx, key)
}

func (s *DistributedRateLimitService) CheckAppRateLimit(ctx context.Context, appID uint) (*DistributedRateLimitResult, error) {
	key := fmt.Sprintf("app:%d", appID)
	return s.CheckRateLimit(ctx, key)
}

func (s *DistributedRateLimitService) GetNodeID() string {
	return s.nodeID
}

func (s *DistributedRateLimitService) GetStats() map[string]interface{} {
	counterCount := 0
	s.counter.Range(func(_, _ interface{}) bool {
		counterCount++
		return true
	})

	return map[string]interface{}{
		"type":              s.config.Type,
		"max_requests":      s.config.MaxRequests,
		"window_secs":       s.config.WindowSecs,
		"node_id":           s.nodeID,
		"redis_enabled":     s.redisEnabled,
		"local_counters":    counterCount,
		"sync_interval":     s.config.SyncInterval.String(),
		"consistency_mode":  s.config.ConsistencyMode,
	}
}

func (s *DistributedRateLimitService) ResetKey(ctx context.Context, key string) error {
	if s.redisEnabled {
		patterns := []string{
			s.config.RedisKeyPrefix + "fixed:" + key,
			s.config.RedisKeyPrefix + "sliding:" + key,
			s.config.RedisKeyPrefix + "token:" + key,
			s.config.RedisKeyPrefix + "leaky:" + key,
		}
		for _, pattern := range patterns {
			redis.Client.Del(ctx, pattern)
		}
	}
	s.counter.Delete(key)
	return nil
}

func (s *DistributedRateLimitService) UpdateConfig(config DistributedRateLimitConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

func (s *DistributedRateLimitService) Close() {
	close(s.stopCh)
	s.wg.Wait()
}
