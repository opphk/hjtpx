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

func TestAdaptiveRateLimitService_Allow(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	ctx := context.Background()
	key := "test-adaptive-key"

	allowed, err := service.Allow(ctx, key)
	assert.NoError(t, err)
	assert.True(t, allowed)
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

	allowed, err := service.Allow(ctx, ip)
	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestAdaptiveRateLimitService_CheckUserRateLimit(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	ctx := context.Background()
	userID := "user-12345"

	allowed, err := service.Allow(ctx, userID)
	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestAdaptiveRateLimitService_CheckAppRateLimit(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	ctx := context.Background()
	appID := "app-67890"

	allowed, err := service.Allow(ctx, appID)
	assert.NoError(t, err)
	assert.True(t, allowed)
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

func TestAdaptiveRateLimitService_UpdateConfig(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	config := AdaptiveRateLimitConfig{
		BaseLimit:     200,
		PeakLimit:     300,
		OffPeakLimit:  500,
		OffPeakStart:  0,
		OffPeakEnd:    6,
		EnableDynamic: true,
	}

	service.UpdateConfig(config)
	stats := service.GetStats()
	assert.NotNil(t, stats)
}

func TestAdaptiveRateLimitService_CustomConfig(t *testing.T) {
	service := NewAdaptiveRateLimitService()
	defer service.Close()

	config := AdaptiveRateLimitConfig{
		BaseLimit:     50,
		PeakLimit:     100,
		OffPeakLimit:  200,
		OffPeakStart:  0,
		OffPeakEnd:    6,
		EnableDynamic: true,
	}

	service.UpdateConfig(config)
	stats := service.GetStats()
	assert.NotNil(t, stats)
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

func TestLoadLevel_Values(t *testing.T) {
	assert.Equal(t, LoadLevel(0), LoadLevelLow)
	assert.Equal(t, LoadLevel(1), LoadLevelNormal)
	assert.Equal(t, LoadLevel(2), LoadLevelMedium)
	assert.Equal(t, LoadLevel(3), LoadLevelHigh)
	assert.Equal(t, LoadLevel(4), LoadLevelCritical)
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
