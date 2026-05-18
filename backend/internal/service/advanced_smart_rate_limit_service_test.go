package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAdaptiveRateLimitService(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	assert.NotNil(t, service)
	defer service.Close()
}

func TestAdaptiveRateLimitService_CheckRateLimit(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	ctx := context.Background()
	key := "test-adaptive-key"

	result, err := service.CheckRateLimit(ctx, key)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
	assert.Greater(t, result.Tokens, 0.0)
}

func TestAdaptiveRateLimitService_CheckRateLimitWithTokens(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	ctx := context.Background()
	key := "test-tokens-key"

	result, err := service.CheckRateLimitWithTokens(ctx, key, 5)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
}

func TestAdaptiveRateLimitService_MultipleRequests(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	ctx := context.Background()
	key := "test-multi-key"

	for i := 0; i < 100; i++ {
		result, err := service.CheckRateLimit(ctx, key)
		assert.NoError(t, err)
		assert.True(t, result.Allowed)
	}
}

func TestAdaptiveRateLimitService_CheckIPRateLimit(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	ctx := context.Background()
	ip := "192.168.1.100"

	result, err := service.CheckIPRateLimit(ctx, ip)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
}

func TestAdaptiveRateLimitService_CheckUserRateLimit(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	ctx := context.Background()
	userID := uint(12345)

	result, err := service.CheckUserRateLimit(ctx, userID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
}

func TestAdaptiveRateLimitService_CheckAppRateLimit(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	ctx := context.Background()
	appID := uint(67890)

	result, err := service.CheckAppRateLimit(ctx, appID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
}

func TestAdaptiveRateLimitService_GetStats(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	ctx := context.Background()
	service.CheckRateLimit(ctx, "stats-test-key")

	stats := service.GetStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "bucket_count")
	assert.Contains(t, stats, "load_level")
	assert.Contains(t, stats, "base_rate")
	assert.Contains(t, stats, "node_id")
}

func TestAdaptiveRateLimitService_GetLoadLevel(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	level := service.GetLoadLevel()
	assert.NotNil(t, level)
}

func TestAdaptiveRateLimitService_GetLoadFactor(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	factor := service.GetLoadFactor()
	assert.GreaterOrEqual(t, factor, 0.0)
	assert.LessOrEqual(t, factor, 1.0)
}

func TestAdaptiveRateLimitService_ResetBucket(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	ctx := context.Background()
	key := "reset-test-key"

	service.CheckRateLimit(ctx, key)
	err := service.ResetBucket(ctx, key)
	assert.NoError(t, err)
}

func TestAdaptiveRateLimitService_UpdateConfig(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	config := AdaptiveRateLimitConfig{
		BaseRate:              200,
		BaseCapacity:          2000,
		HighLoadThreshold:     0.8,
		CriticalLoadThreshold: 0.95,
	}

	service.UpdateConfig(config)
	stats := service.GetStats()
	assert.Equal(t, 200.0, stats["base_rate"])
	assert.Equal(t, 2000.0, stats["base_capacity"])
}

func TestAdaptiveRateLimitService_CustomConfig(t *testing.T) {
	config := AdaptiveRateLimitConfig{
		BaseRate:              50,
		BaseCapacity:          500,
		MinCapacity:           50,
		MaxCapacity:          1000,
		LoadCheckInterval:    2 * time.Second,
		AdjustmentInterval:   15 * time.Second,
		HighLoadThreshold:    0.6,
		CriticalLoadThreshold: 0.85,
	}

	service := NewAdaptiveRateLimitService(config)
	defer service.Close()

	stats := service.GetStats()
	assert.Equal(t, 50.0, stats["base_rate"])
	assert.Equal(t, 500.0, stats["base_capacity"])
}

func TestLoadLevel_String(t *testing.T) {
	tests := []struct {
		level    LoadLevel
		expected string
	}{
		{LoadLevelLow, "low"},
		{LoadLevelNormal, "normal"},
		{LoadLevelMedium, "medium"},
		{LoadLevelHigh, "high"},
		{LoadLevelCritical, "critical"},
		{LoadLevel(99), "unknown"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.level.String())
	}
}

func TestAdaptiveTokenBucket_Refill(t *testing.T) {
	bucket := &AdaptiveTokenBucket{
		capacity:   100,
		rate:       10,
		tokens:     50,
		lastRefill: time.Now().Add(-time.Second),
	}

	bucket.mu.Lock()
	bucket.refill()
	bucket.mu.Unlock()

	assert.Greater(t, bucket.tokens, 50.0)
}

func TestAdaptiveTokenBucket_TryConsume(t *testing.T) {
	bucket := &AdaptiveTokenBucket{
		capacity:   100,
		rate:       10,
		tokens:     100,
		lastRefill: time.Now(),
		loadFactor: 1.0,
	}

	result := bucket.tryConsume(10)
	assert.True(t, result.Allowed)
	assert.Equal(t, 90.0, result.Tokens)

	bucket.tokens = 0
	result = bucket.tryConsume(1)
	assert.False(t, result.Allowed)
	assert.Greater(t, result.RetryAfter, 0*time.Second)
}

func TestAdaptiveRateLimitService_LoadAdjustment(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	ctx := context.Background()
	key := "load-adjust-key"

	for i := 0; i < 1000; i++ {
		service.CheckRateLimit(ctx, key)
	}

	stats := service.GetStats()
	level := service.GetLoadLevel()
	
	assert.NotNil(t, stats)
	t.Logf("Load level: %s, Load factor: %.2f", level.String(), stats["load_factor"])
}
