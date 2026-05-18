package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDistributedRateLimitService(t *testing.T) {
	svc := NewDistributedRateLimitService()
	assert.NotNil(t, svc)
}

func TestDistributedRateLimitService_Check(t *testing.T) {
	svc := NewDistributedRateLimitService()

	ctx := context.Background()
	key := "test-dist-key"

	result, err := svc.Check(ctx, key, 100)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
	assert.GreaterOrEqual(t, result.Remaining, 0)
}

func TestDistributedRateLimitService_CheckRateLimitWithCount(t *testing.T) {
	svc := NewDistributedRateLimitService()

	ctx := context.Background()
	key := "test-count-key"

	result, err := svc.CheckRateLimitWithCount(ctx, key, 5)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
}

func TestDistributedRateLimitService_MultipleRequests(t *testing.T) {
	svc := NewDistributedRateLimitService()

	ctx := context.Background()
	key := "test-multi-dist-key"

	for i := 0; i < 10; i++ {
		result, err := svc.Check(ctx, key, 100)
		assert.NoError(t, err)
		assert.True(t, result.Allowed)
	}
}

func TestDistributedRateLimitService_Allow(t *testing.T) {
	svc := NewDistributedRateLimitService()

	ctx := context.Background()
	key := "test-allow-key"

	allowed, err := svc.Allow(ctx, key, 50)
	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestDistributedRateLimitService_Reset(t *testing.T) {
	svc := NewDistributedRateLimitService()

	ctx := context.Background()
	key := "test-reset-key"

	svc.Check(ctx, key, 100)
	err := svc.Reset(ctx, key)
	assert.NoError(t, err)
}

func TestDistributedRateLimitService_GetNodeID(t *testing.T) {
	svc := NewDistributedRateLimitService()

	nodeID := svc.GetNodeID()
	assert.NotEmpty(t, nodeID)
}

func TestDistributedRateLimitService_GetStats(t *testing.T) {
	svc := NewDistributedRateLimitService()

	stats := svc.GetStats()
	assert.NotNil(t, stats)
}

func TestDistributedRateLimitService_UpdateConfig(t *testing.T) {
	svc := NewDistributedRateLimitService()

	config := DistributedRateLimitConfig{
		RedisEnabled:     false,
		ConsistencyLevel: "strong",
		SyncInterval:    10 * time.Second,
	}

	svc.UpdateConfig(config)

	t.Log("UpdateConfig 测试通过")
}

func TestDistributedRateLimitService_Concurrent(t *testing.T) {
	svc := NewDistributedRateLimitService()

	ctx := context.Background()
	key := "test-concurrent-key"

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			result, err := svc.Check(ctx, key, 100)
			if err == nil && result != nil {
				done <- true
			} else {
				done <- false
			}
		}()
	}

	for i := 0; i < 10; i++ {
		result := <-done
		assert.True(t, result)
	}
}
