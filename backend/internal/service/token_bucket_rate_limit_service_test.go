package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTokenBucketRateLimitService(t *testing.T) {
	service := NewTokenBucketRateLimitService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.buckets)
}

func TestTokenBucketRateLimitService_CheckTokenBucketRateLimit(t *testing.T) {
	service := NewTokenBucketRateLimitService()
	ctx := context.Background()
	key := "test-key"

	config := &TokenBucketConfig{
		Rate:          10,
		Capacity:      100,
		BurstSize:     50,
		InitialTokens: 100,
	}

	// 第一次请求应该允许
	result, err := service.CheckTokenBucketRateLimit(ctx, key, config)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Greater(t, result.Tokens, 0.0)

	// 多次请求
	for i := 0; i < 10; i++ {
		result, _ := service.CheckTokenBucketRateLimit(ctx, key, config)
		assert.True(t, result.Allowed)
	}
}

func TestTokenBucketRateLimitService_GetBucketStats(t *testing.T) {
	service := NewTokenBucketRateLimitService()
	key := "test-stats-key"

	config := &TokenBucketConfig{
		Rate:          5,
		Capacity:      50,
		BurstSize:     25,
		InitialTokens: 50,
	}

	// 先调用 CheckTokenBucketRateLimit 来创建桶
	ctx := context.Background()
	_, _ = service.CheckTokenBucketRateLimit(ctx, key, config)

	stats := service.GetBucketStats(key)
	assert.NotNil(t, stats)
	assert.Equal(t, "tokenbucket:"+key, stats["key"])
}

func TestTokenBucketRateLimitService_ResetBucket(t *testing.T) {
	service := NewTokenBucketRateLimitService()
	ctx := context.Background()
	key := "test-reset-key"

	config := &TokenBucketConfig{
		Rate:          1,
		Capacity:      10,
		BurstSize:     5,
		InitialTokens: 10,
	}

	// 调用多次以减少令牌
	for i := 0; i < 5; i++ {
		service.CheckTokenBucketRateLimit(ctx, key, config)
	}

	// 重置桶
	err := service.ResetBucket(ctx, key)
	assert.NoError(t, err)

	// 再次调用，应该有足够的令牌
	result, _ := service.CheckTokenBucketRateLimit(ctx, key, config)
	assert.True(t, result.Allowed)
}

func TestTrafficShaper(t *testing.T) {
	config := &TokenBucketConfig{
		Rate:          10,
		Capacity:      100,
		BurstSize:     50,
		InitialTokens: 100,
	}

	shaper := NewTrafficShaper(config)
	assert.NotNil(t, shaper)
	defer shaper.Close()

	// 测试提交任务
	success := shaper.Submit(func() {
		// 任务会被执行
	})
	assert.True(t, success)

	// 等待任务完成
	time.Sleep(100 * time.Millisecond)
}

func TestTokenBucket_Refill(t *testing.T) {
	bucket := &TokenBucket{
		capacity:   100,
		rate:       10,
		tokens:     50,
		lastRefill: time.Now().Add(-time.Second),
		burstSize:  50,
	}

	bucket.mu.Lock()
	bucket.refill()
	bucket.mu.Unlock()

	// 应该填充了约 10 个令牌
	assert.Greater(t, bucket.tokens, 50.0)
}

func TestTokenBucket_TryConsume(t *testing.T) {
	bucket := &TokenBucket{
		capacity:   100,
		rate:       10,
		tokens:     100,
		lastRefill: time.Now(),
		burstSize:  50,
	}

	// 正常消费
	result := bucket.tryConsume(1)
	assert.True(t, result.Allowed)
	assert.False(t, result.IsBurst)

	// 使用突发处理
	bucket.tokens = 0
	result = bucket.tryConsume(10)
	assert.True(t, result.Allowed)
	assert.True(t, result.IsBurst)

	// 超过限制
	bucket.tokens = 0
	bucket.burstSize = 0
	result = bucket.tryConsume(1)
	assert.False(t, result.Allowed)
}

func TestDefaultTokenBucketConfig(t *testing.T) {
	assert.Equal(t, 10.0, defaultTokenBucketConfig.Rate)
	assert.Equal(t, 100.0, defaultTokenBucketConfig.Capacity)
	assert.Equal(t, 50.0, defaultTokenBucketConfig.BurstSize)
	assert.Equal(t, 100.0, defaultTokenBucketConfig.InitialTokens)
}
