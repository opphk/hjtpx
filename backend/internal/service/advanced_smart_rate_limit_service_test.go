package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAdaptiveRateLimitService(t *testing.T) {
	svc := NewAdaptiveRateLimitService()
	assert.NotNil(t, svc)
	defer svc.Close()
}

func TestAdaptiveRateLimitService_CheckRateLimit(t *testing.T) {
	svc := NewAdaptiveRateLimitService()
	defer svc.Close()

	ctx := context.Background()
	key := "test-adaptive-key"

	result, err := svc.CheckRateLimit(ctx, key)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
}

func TestAdaptiveRateLimitService_CheckRateLimitWithTokens(t *testing.T) {
	svc := NewAdaptiveRateLimitService()
	defer svc.Close()

	ctx := context.Background()
	key := "test-tokens-key"

	result, err := svc.CheckRateLimitWithTokens(ctx, key, 1)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
}

func TestAdaptiveRateLimitService_MultipleRequests(t *testing.T) {
	svc := NewAdaptiveRateLimitService()
	defer svc.Close()

	ctx := context.Background()
	key := "test-multi-key"

	for i := 0; i < 10; i++ {
		result, err := svc.CheckRateLimit(ctx, key)
		assert.NoError(t, err)
		assert.True(t, result.Allowed)
	}
}

func TestAdaptiveRateLimitService_Allow(t *testing.T) {
	svc := NewAdaptiveRateLimitService()
	defer svc.Close()

	ctx := context.Background()
	key := "test-allow-key"

	allowed, err := svc.Allow(ctx, key)
	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestAdaptiveRateLimitService_GetStats(t *testing.T) {
	svc := NewAdaptiveRateLimitService()
	defer svc.Close()

	ctx := context.Background()
	svc.CheckRateLimit(ctx, "stats-test-key")

	stats := svc.GetStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "token_bucket")
	assert.Contains(t, stats, "sliding_window")
	assert.Contains(t, stats, "load_factor")
	assert.Contains(t, stats, "load_level")
}

func TestAdaptiveRateLimitService_GetLoadLevel(t *testing.T) {
	svc := NewAdaptiveRateLimitService()
	defer svc.Close()

	level := svc.GetLoadLevel()
	assert.NotNil(t, level)
}

func TestAdaptiveRateLimitService_UpdateConfig(t *testing.T) {
	svc := NewAdaptiveRateLimitService()
	defer svc.Close()

	config := AdaptiveRateLimitConfig{
		BaseLimit:     100,
		PeakLimit:     200,
		OffPeakLimit:  500,
		EnableDynamic: true,
	}

	svc.UpdateConfig(config)
	stats := svc.GetStats()
	assert.NotNil(t, stats)
}

func TestAdaptiveRateLimitService_CustomConfig(t *testing.T) {
	config := AdaptiveRateLimitConfig{
		BaseLimit:      50,
		PeakLimit:      100,
		OffPeakLimit:   200,
		OffPeakStart:   0,
		OffPeakEnd:     6,
		EnableDynamic:  true,
	}

	svc := NewAdaptiveRateLimitService()
	defer svc.Close()
	svc.UpdateConfig(config)

	stats := svc.GetStats()
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

func TestAdaptiveRateLimitService_LoadAdjustment(t *testing.T) {
	svc := NewAdaptiveRateLimitService()
	defer svc.Close()

	ctx := context.Background()
	key := "load-adjust-key"

	for i := 0; i < 10; i++ {
		svc.CheckRateLimit(ctx, key)
	}

	stats := svc.GetStats()
	level := svc.GetLoadLevel()

	assert.NotNil(t, stats)
	t.Logf("Load level: %s", level.String())
}
