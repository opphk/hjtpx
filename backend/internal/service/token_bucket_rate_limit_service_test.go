package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTokenBucketRateLimitService(t *testing.T) {
	svc := NewTokenBucketRateLimitService()
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.buckets)
}

func TestTokenBucketRateLimitService_CheckTokenBucketRateLimit(t *testing.T) {
	svc := NewTokenBucketRateLimitService()
	ctx := context.Background()
	key := "test-key"

	config := &TokenBucketConfig{
		Capacity:    100,
		RefillRate:  10,
		RefillPerSec: 10,
	}

	result, err := svc.CheckTokenBucketRateLimit(ctx, key, config)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)

	for i := 0; i < 10; i++ {
		result, _ := svc.CheckTokenBucketRateLimit(ctx, key, config)
		assert.True(t, result.Allowed)
	}
}

func TestTokenBucketRateLimitService_GetBucketStats(t *testing.T) {
	svc := NewTokenBucketRateLimitService()
	key := "test-stats-key"

	config := &TokenBucketConfig{
		Capacity:    50,
		RefillRate:  5,
		RefillPerSec: 5,
	}

	ctx := context.Background()
	_, _ = svc.CheckTokenBucketRateLimit(ctx, key, config)

	stats := svc.GetBucketStats(key)
	assert.NotNil(t, stats)
}

func TestTokenBucketRateLimitService_ResetBucket(t *testing.T) {
	svc := NewTokenBucketRateLimitService()
	ctx := context.Background()
	key := "test-reset-key"

	config := &TokenBucketConfig{
		Capacity:    10,
		RefillRate:  1,
		RefillPerSec: 1,
	}

	for i := 0; i < 5; i++ {
		svc.CheckTokenBucketRateLimit(ctx, key, config)
	}

	err := svc.ResetBucket(ctx, key)
	assert.NoError(t, err)

	result, _ := svc.CheckTokenBucketRateLimit(ctx, key, config)
	assert.True(t, result.Allowed)
}

func TestTokenBucketConfig_Defaults(t *testing.T) {
	cfg := defaultTokenBucketConfig
	assert.Equal(t, 100.0, cfg.Capacity)
	assert.Greater(t, cfg.Rate, 0.0)
}
