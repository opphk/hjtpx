package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenBucketRateLimitService(t *testing.T) {
	t.Run("AllowRequest", func(t *testing.T) {
		svc := NewTokenBucketRateLimitService()

		allowed := svc.Allow("test-key")
		assert.True(t, allowed)
	})

	t.Run("MultipleRequests", func(t *testing.T) {
		svc := NewTokenBucketRateLimitService()

		for i := 0; i < 5; i++ {
			allowed := svc.Allow("test-key-2")
			assert.True(t, allowed)
		}
	})

	t.Run("ResetBucket", func(t *testing.T) {
		svc := NewTokenBucketRateLimitService()

		svc.Allow("test-key-3")
		svc.Reset("test-key-3")

		allowed := svc.Allow("test-key-3")
		assert.True(t, allowed)
	})

	t.Run("ResetAll", func(t *testing.T) {
		svc := NewTokenBucketRateLimitService()

		svc.Allow("key1")
		svc.Allow("key2")
		svc.ResetAll()

		allowed1 := svc.Allow("key1")
		allowed2 := svc.Allow("key2")
		assert.True(t, allowed1)
		assert.True(t, allowed2)
	})

	t.Run("GetTokens", func(t *testing.T) {
		svc := NewTokenBucketRateLimitService()

		svc.Allow("test-key-4")
		tokens := svc.GetTokens("test-key-4")
		assert.GreaterOrEqual(t, tokens, float64(0))
	})

	t.Run("GetConfig", func(t *testing.T) {
		svc := NewTokenBucketRateLimitService()
		config := svc.GetConfig()

		assert.Equal(t, int64(100), config.Capacity)
		assert.Equal(t, float64(10), config.RefillRate)
	})

	t.Run("UpdateConfig", func(t *testing.T) {
		svc := NewTokenBucketRateLimitService()
		newConfig := TokenBucketConfig{
			Capacity:    50,
			RefillRate:  5,
			RefillPerSec: 5,
		}

		svc.UpdateConfig(newConfig)
		config := svc.GetConfig()

		assert.Equal(t, int64(50), config.Capacity)
		assert.Equal(t, float64(5), config.RefillRate)
	})

	t.Run("GetStats", func(t *testing.T) {
		svc := NewTokenBucketRateLimitService()
		svc.Allow("stats-key")

		stats := svc.GetStats()
		assert.NotNil(t, stats)
		assert.Contains(t, stats, "total_buckets")
	})
}

func TestSlidingWindowRateLimitService(t *testing.T) {
	t.Run("AllowRequest", func(t *testing.T) {
		svc := NewSlidingWindowRateLimitService()

		allowed := svc.Allow("window-key")
		assert.True(t, allowed)
	})

	t.Run("CheckWindow", func(t *testing.T) {
		svc := NewSlidingWindowRateLimitService()

		svc.Allow("check-key")
		allowed, remaining, resetAt := svc.Check("check-key")

		assert.True(t, allowed)
		assert.GreaterOrEqual(t, remaining, int64(0))
		assert.False(t, resetAt.IsZero())
	})

	t.Run("CheckSlidingWindow", func(t *testing.T) {
		svc := NewSlidingWindowRateLimitService()

		result, err := svc.CheckSlidingWindow(context.Background(), "sliding-key", 100)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
	})

	t.Run("ResetWindow", func(t *testing.T) {
		svc := NewSlidingWindowRateLimitService()

		svc.Allow("reset-key")
		svc.Reset("reset-key")

		allowed := svc.Allow("reset-key")
		assert.True(t, allowed)
	})

	t.Run("GetConfig", func(t *testing.T) {
		svc := NewSlidingWindowRateLimitService()
		config := svc.GetConfig()

		assert.Equal(t, time.Minute, config.WindowSize)
		assert.Equal(t, int64(100), config.MaxRequests)
	})
}

func TestRateLimitService(t *testing.T) {
	t.Run("CheckIPRateLimit", func(t *testing.T) {
		svc := NewRateLimitService()

		result, err := svc.CheckIPRateLimit(context.Background(), "192.168.1.1", nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
	})

	t.Run("CheckUserRateLimit", func(t *testing.T) {
		svc := NewRateLimitService()

		result, err := svc.CheckUserRateLimit(context.Background(), 123, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
	})

	t.Run("CheckAppRateLimit", func(t *testing.T) {
		svc := NewRateLimitService()

		result, err := svc.CheckAppRateLimit(context.Background(), 456, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
	})
}

func TestLeakyBucketRateLimitService(t *testing.T) {
	t.Run("AllowRequest", func(t *testing.T) {
		svc := NewLeakyBucketRateLimitService()

		allowed := svc.Allow("leaky-key")
		assert.True(t, allowed)
	})

	t.Run("Reset", func(t *testing.T) {
		svc := NewLeakyBucketRateLimitService()

		svc.Allow("leaky-reset-key")
		svc.Reset("leaky-reset-key")

		allowed := svc.Allow("leaky-reset-key")
		assert.True(t, allowed)
	})

	t.Run("GetConfig", func(t *testing.T) {
		svc := NewLeakyBucketRateLimitService()
		config := svc.GetConfig()

		assert.Equal(t, int64(100), config.Capacity)
		assert.Equal(t, float64(10), config.LeakRate)
	})
}

func TestRateLimitConfig(t *testing.T) {
	t.Run("DefaultIPConfig", func(t *testing.T) {
		assert.Equal(t, 60, DefaultIPConfig.MaxRequests)
		assert.Equal(t, 60, DefaultIPConfig.WindowSecs)
	})

	t.Run("DefaultUserConfig", func(t *testing.T) {
		assert.Equal(t, 100, DefaultUserConfig.MaxRequests)
		assert.Equal(t, 60, DefaultUserConfig.WindowSecs)
	})

	t.Run("DefaultAppConfig", func(t *testing.T) {
		assert.Equal(t, 200, DefaultAppConfig.MaxRequests)
		assert.Equal(t, 60, DefaultAppConfig.WindowSecs)
	})
}

func TestRateLimitConstants(t *testing.T) {
	t.Run("PrefixConstants", func(t *testing.T) {
		assert.Equal(t, "ratelimit:ip:", PrefixIPRateLimit)
		assert.Equal(t, "ratelimit:user:", PrefixUserRateLimit)
		assert.Equal(t, "ratelimit:app:", PrefixAppRateLimit)
		assert.Equal(t, "blacklist:", PrefixBlacklist)
		assert.Equal(t, "whitelist:", PrefixWhitelist)
	})
}
