package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDistributedRateLimitService(t *testing.T) {
	service := NewDistributedRateLimitService()
	assert.NotNil(t, service)
	defer service.Close()
}

func TestDistributedRateLimitService_CheckRateLimit(t *testing.T) {
	service := NewDistributedRateLimitService()
	defer service.Close()

	ctx := context.Background()
	key := "test-dist-key"

	result, err := service.CheckRateLimit(ctx, key)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
	assert.GreaterOrEqual(t, result.Remaining, 0)
}

func TestDistributedRateLimitService_CheckRateLimitWithCount(t *testing.T) {
	service := NewDistributedRateLimitService()
	defer service.Close()

	ctx := context.Background()
	key := "test-count-key"

	result, err := service.CheckRateLimitWithCount(ctx, key, 5)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
}

func TestDistributedRateLimitService_MultipleRequests(t *testing.T) {
	service := NewDistributedRateLimitService()
	defer service.Close()

	ctx := context.Background()
	key := "test-multi-dist-key"

	for i := 0; i < 50; i++ {
		result, err := service.CheckRateLimit(ctx, key)
		assert.NoError(t, err)
		assert.True(t, result.Allowed)
	}
}

func TestDistributedRateLimitService_CheckIPRateLimit(t *testing.T) {
	service := NewDistributedRateLimitService()
	defer service.Close()

	ctx := context.Background()
	ip := "192.168.1.200"

	result, err := service.CheckIPRateLimit(ctx, ip)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
}

func TestDistributedRateLimitService_CheckUserRateLimit(t *testing.T) {
	service := NewDistributedRateLimitService()
	defer service.Close()

	ctx := context.Background()
	userID := uint(99999)

	result, err := service.CheckUserRateLimit(ctx, userID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
}

func TestDistributedRateLimitService_CheckAppRateLimit(t *testing.T) {
	service := NewDistributedRateLimitService()
	defer service.Close()

	ctx := context.Background()
	appID := uint(88888)

	result, err := service.CheckAppRateLimit(ctx, appID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
}

func TestDistributedRateLimitService_GetNodeID(t *testing.T) {
	service := NewDistributedRateLimitService()
	defer service.Close()

	nodeID := service.GetNodeID()
	assert.NotEmpty(t, nodeID)
	assert.Contains(t, nodeID, "node-")
}

func TestDistributedRateLimitService_GetStats(t *testing.T) {
	service := NewDistributedRateLimitService()
	defer service.Close()

	ctx := context.Background()
	service.CheckRateLimit(ctx, "stats-dist-key")

	stats := service.GetStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "type")
	assert.Contains(t, stats, "max_requests")
	assert.Contains(t, stats, "node_id")
	assert.Contains(t, stats, "redis_enabled")
}

func TestDistributedRateLimitService_ResetKey(t *testing.T) {
	service := NewDistributedRateLimitService()
	defer service.Close()

	ctx := context.Background()
	key := "reset-dist-key"

	service.CheckRateLimit(ctx, key)
	err := service.ResetKey(ctx, key)
	assert.NoError(t, err)
}

func TestDistributedRateLimitService_UpdateConfig(t *testing.T) {
	service := NewDistributedRateLimitService()
	defer service.Close()

	config := DistributedRateLimitConfig{
		Type:            DistributedSlidingWindow,
		MaxRequests:     200,
		WindowSecs:      120,
		SyncInterval:   10 * time.Second,
		ConsistencyMode: true,
	}

	service.UpdateConfig(config)
	stats := service.GetStats()
	assert.Equal(t, DistributedSlidingWindow, stats["type"])
}

func TestDistributedRateLimitService_CustomConfig(t *testing.T) {
	config := DistributedRateLimitConfig{
		Type:            DistributedLeakyBucket,
		MaxRequests:     50,
		WindowSecs:      30,
		SyncInterval:   3 * time.Second,
		ConsistencyMode: false,
	}

	service := NewDistributedRateLimitService(config)
	defer service.Close()

	stats := service.GetStats()
	assert.Equal(t, DistributedLeakyBucket, stats["type"])
	assert.Equal(t, 50, stats["max_requests"])
}

func TestDistributedRateLimitService_FixedWindow(t *testing.T) {
	config := DistributedRateLimitConfig{
		Type:           DistributedFixedWindow,
		MaxRequests:    20,
		WindowSecs:     10,
		SyncInterval:   2 * time.Second,
	}

	service := NewDistributedRateLimitService(config)
	defer service.Close()

	ctx := context.Background()
	key := "fixed-window-key"

	result, err := service.CheckRateLimit(ctx, key)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, 20, result.Remaining+int(result.TotalCount))
}

func TestDistributedRateLimitService_SlidingWindow(t *testing.T) {
	config := DistributedRateLimitConfig{
		Type:           DistributedSlidingWindow,
		MaxRequests:    30,
		WindowSecs:     60,
		SyncInterval:   5 * time.Second,
	}

	service := NewDistributedRateLimitService(config)
	defer service.Close()

	ctx := context.Background()
	key := "sliding-window-key"

	result, err := service.CheckRateLimit(ctx, key)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestDistributedRateLimitService_TokenBucket(t *testing.T) {
	config := DistributedRateLimitConfig{
		Type:           DistributedTokenBucket,
		MaxRequests:    100,
		WindowSecs:     60,
		SyncInterval:   5 * time.Second,
	}

	service := NewDistributedRateLimitService(config)
	defer service.Close()

	ctx := context.Background()
	key := "dist-token-bucket-key"

	result, err := service.CheckRateLimit(ctx, key)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestDistributedRateLimitService_LeakyBucket(t *testing.T) {
	config := DistributedRateLimitConfig{
		Type:           DistributedLeakyBucket,
		MaxRequests:    50,
		WindowSecs:     30,
		SyncInterval:   5 * time.Second,
	}

	service := NewDistributedRateLimitService(config)
	defer service.Close()

	ctx := context.Background()
	key := "leaky-bucket-key"

	result, err := service.CheckRateLimit(ctx, key)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestDistributedRateLimitService_RateLimitExceeded(t *testing.T) {
	config := DistributedRateLimitConfig{
		Type:           DistributedFixedWindow,
		MaxRequests:    5,
		WindowSecs:     60,
		SyncInterval:   10 * time.Second,
	}

	service := NewDistributedRateLimitService(config)
	defer service.Close()

	ctx := context.Background()
	key := "exceeded-key"

	for i := 0; i < 5; i++ {
		result, err := service.CheckRateLimit(ctx, key)
		assert.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	result, err := service.CheckRateLimit(ctx, key)
	assert.NoError(t, err)
	assert.False(t, result.Allowed)
}

func TestLocalCounter(t *testing.T) {
	counter := &localCounter{
		count:       0,
		windowStart: time.Now(),
	}

	counter.mu.Lock()
	counter.count = 10
	counter.mu.Unlock()

	assert.Equal(t, int64(10), counter.count)
}
