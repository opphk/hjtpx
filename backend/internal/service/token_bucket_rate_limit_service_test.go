package service

import (
	"context"
	"testing"

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
	ip := "127.0.0.1"

	config := &TokenBucketConfig{
		Capacity:     100,
		RefillRate:   10,
		RefillPerSec: 10,
	}

	result, err := service.CheckIPTokenBucketLimit(ctx, ip, config)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Greater(t, result.Remaining, 0.0)

	for i := 0; i < 10; i++ {
		result, _ = service.CheckIPTokenBucketLimit(ctx, ip, config)
		assert.True(t, result.Allowed)
	}
}

func TestTokenBucketRateLimitService_GetBucketStats(t *testing.T) {
	service := NewTokenBucketRateLimitService()
	ip := "127.0.0.1"

	config := &TokenBucketConfig{
		Capacity:     50,
		RefillRate:   5,
		RefillPerSec: 5,
	}

	ctx := context.Background()
	_, _ = service.CheckIPTokenBucketLimit(ctx, ip, config)

	stats := service.GetBucketStats("ip:" + ip)
	assert.NotNil(t, stats)
	assert.Equal(t, "ip:"+ip, stats["key"])
}

func TestTokenBucketRateLimitService_ResetBucket(t *testing.T) {
	service := NewTokenBucketRateLimitService()
	ctx := context.Background()
	ip := "192.168.1.100"

	config := &TokenBucketConfig{
		Capacity:     10,
		RefillRate:   1,
		RefillPerSec: 1,
	}

	for i := 0; i < 5; i++ {
		service.CheckIPTokenBucketLimit(ctx, ip, config)
	}

	err := service.ResetBucket(ctx, "ip:"+ip)
	assert.NoError(t, err)

	result, _ := service.CheckIPTokenBucketLimit(ctx, ip, config)
	assert.True(t, result.Allowed)
}
