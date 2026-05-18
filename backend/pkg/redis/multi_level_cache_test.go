package redis

import (
	"context"
	"testing"
	"time"
)

func TestEnhancedCacheMultiLevel(t *testing.T) {
	config := &CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: false,
		L3Enabled: false,
		L1Size:    100,
		L1TTL:     1 * time.Minute,
	}

	cache := NewEnhancedCache(config)

	ctx := context.Background()
	key := "test:multi_level_key"
	value := []byte("test_value")

	err := cache.Set(ctx, key, value, &SetOptions{TTL: 10 * time.Second, Level: CacheLevelL1})
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := cache.Get(ctx, key, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(val) != string(value) {
		t.Errorf("Value mismatch: got %s, want %s", val, value)
	}

	err = cache.Delete(ctx, key, &DeleteOptions{Level: CacheLevelL1})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = cache.Get(ctx, key, nil)
	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss, got %v", err)
	}
}

func TestSmartTTLEngine(t *testing.T) {
	engine := NewSmartTTLEngine(nil)

	testKey := "test:ttl_key"

	for i := 0; i < 150; i++ {
		engine.RecordAccess(testKey)
	}

	ttl := engine.CalculateTTL(testKey)
	if ttl < engine.config.BaseTTL {
		t.Errorf("Expected TTL >= base TTL, got %v", ttl)
	}

	pattern := engine.GetAccessPattern(testKey)
	if pattern != AccessPatternFrequent {
		t.Errorf("Expected frequent access pattern, got %v", pattern)
	}
}

func TestSmartTTLEngineStrategies(t *testing.T) {
	tests := []struct {
		name     string
		strategy TTLStrategyType
	}{
		{"fixed", TTLStrategyFixed},
		{"sliding", TTLStrategySliding},
		{"adaptive", TTLStrategyAdaptive},
		{"access_based", TTLStrategyAccessBased},
		{"hot_key", TTLStrategyHotKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewSmartTTLEngine(nil)
			engine.SetStrategy(tt.strategy)

			ttl := engine.CalculateTTL("test_key")
			if ttl <= 0 {
				t.Errorf("TTL should be positive, got %v", ttl)
			}
		})
	}
}

func TestCacheWarmupManager(t *testing.T) {
	config := &WarmupConfig{
		Policy:      WarmupPolicyOnDemand,
		Concurrency: 1,
		BatchSize:   10,
		Enabled:     true,
	}

	manager := NewCacheWarmupManager(config)

	task := &CacheWarmupTask{
		Name:      "test_task",
		Key:       "test:warmup_key",
		Priority:  WarmupPriorityHigh,
		Policy:    WarmupPolicyOnDemand,
		TTL:       1 * time.Minute,
		Frequency: 0,
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("warmup_value"), nil
		},
		Enabled:    true,
		MaxRetries: 3,
	}

	manager.AddTask(task)
	manager.Start()

	time.Sleep(100 * time.Millisecond)

	if task.Stats.SuccessCount.Load() != 1 {
		t.Errorf("Expected 1 success, got %d", task.Stats.SuccessCount.Load())
	}

	manager.Stop()
}

func TestCacheWarmupManagerPriority(t *testing.T) {
	config := &WarmupConfig{
		Policy:      WarmupPolicyOnDemand,
		Concurrency: 1,
		BatchSize:   10,
		Enabled:     true,
	}

	manager := NewCacheWarmupManager(config)

	highDone := make(chan bool, 1)
	lowDone := make(chan bool, 1)

	lowTask := &CacheWarmupTask{
		Name:      "low_task",
		Key:       "test:low",
		Priority:  WarmupPriorityLow,
		Policy:    WarmupPolicyOnDemand,
		TTL:       1 * time.Minute,
		Frequency: 0,
		Loader: func(ctx context.Context) ([]byte, error) {
			time.Sleep(200 * time.Millisecond)
			lowDone <- true
			return []byte("low"), nil
		},
		Enabled:    true,
		MaxRetries: 3,
	}

	highTask := &CacheWarmupTask{
		Name:      "high_task",
		Key:       "test:high",
		Priority:  WarmupPriorityHigh,
		Policy:    WarmupPolicyOnDemand,
		TTL:       1 * time.Minute,
		Frequency: 0,
		Loader: func(ctx context.Context) ([]byte, error) {
			highDone <- true
			return []byte("high"), nil
		},
		Enabled:    true,
		MaxRetries: 3,
	}

	manager.AddTask(lowTask)
	manager.AddTask(highTask)
	manager.Start()

	select {
	case <-highDone:
	case <-time.After(500 * time.Millisecond):
		t.Error("High priority task should complete first")
	}

	manager.Stop()
}

func TestCacheWarmupManagerRetry(t *testing.T) {
	failureCount := 0
	config := &WarmupConfig{
		Policy:      WarmupPolicyOnDemand,
		Concurrency: 1,
		BatchSize:   10,
		Enabled:     true,
	}

	manager := NewCacheWarmupManager(config)

	task := &CacheWarmupTask{
		Name:      "retry_task",
		Key:       "test:retry",
		Priority:  WarmupPriorityNormal,
		Policy:    WarmupPolicyOnDemand,
		TTL:       1 * time.Minute,
		Frequency: 5 * time.Second,
		Loader: func(ctx context.Context) ([]byte, error) {
			failureCount++
			if failureCount <= 2 {
				return nil, ErrCacheMiss
			}
			return []byte("success"), nil
		},
		Enabled:    true,
		MaxRetries: 2,
	}

	manager.AddTask(task)
	manager.Start()

	time.Sleep(4 * time.Second)

	if task.Stats.SuccessCount.Load() != 1 {
		t.Errorf("Expected 1 success after retries, got %d", task.Stats.SuccessCount.Load())
	}

	if failureCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", failureCount)
	}

	manager.Stop()
}
