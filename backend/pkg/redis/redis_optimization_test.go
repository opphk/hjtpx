package redis

import (
	"context"
	"testing"
	"time"
)

func TestRedisClient(t *testing.T) {
	rc := &RedisClient{}
	if rc == nil {
		t.Fatal("RedisClient should not be nil")
	}
}

func TestNewRedisClient(t *testing.T) {
	rc, err := NewRedisClient(nil)
	if err != nil {
		t.Logf("NewRedisClient returned error (expected without server): %v", err)
	}
	if rc == nil && err == nil {
		t.Fatal("NewRedisClient should either return client or error")
	}
}

func TestGetConnectionMetrics(t *testing.T) {
	metrics := GetConnectionMetrics()
	if metrics == nil {
		t.Fatal("GetConnectionMetrics should not return nil")
	}
}

func TestCacheKeyGenerator(t *testing.T) {
	kg := NewCacheKeyGenerator("user")

	tests := []struct {
		name     string
		keys     []string
		expected string
	}{
		{
			name:     "No keys",
			keys:     []string{},
			expected: "user",
		},
		{
			name:     "Single key",
			keys:     []string{"123"},
			expected: "user:123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := kg.Generate(tt.keys...)
			if result != tt.expected {
				t.Errorf("Generate(%v) = %q, want %q", tt.keys, result, tt.expected)
			}
		})
	}
}

func TestCacheKeyGeneratorWithVersion(t *testing.T) {
	kg := NewCacheKeyGenerator("user")

	result := kg.GenerateWithVersion(1, "123")
	expected := "user:123:v1"
	if result != expected {
		t.Errorf("GenerateWithVersion(1, 123) = %q, want %q", result, expected)
	}
}

func TestCacheKeyGeneratorPattern(t *testing.T) {
	kg := NewCacheKeyGenerator("user")

	result := kg.GeneratePattern()
	expected := "user:*"
	if result != expected {
		t.Errorf("GeneratePattern() = %q, want %q", result, expected)
	}
}

func TestDistributedLock(t *testing.T) {
	lock := NewDistributedLock(nil, "test_key", "test_value", 10*time.Second)
	if lock == nil {
		t.Fatal("NewDistributedLock should not return nil")
	}

	if lock.key != "test_key" {
		t.Errorf("Lock key = %q, want %q", lock.key, "test_key")
	}

	if lock.value != "test_value" {
		t.Errorf("Lock value = %q, want %q", lock.value, "test_value")
	}
}

func TestDistributedLockAcquireWithNilClient(t *testing.T) {
	lock := NewDistributedLock(nil, "test_key", "test_value", 10*time.Second)

	ctx := context.Background()
	result, err := lock.Acquire(ctx)
	if err != nil {
		t.Logf("Acquire returned error (expected with nil client): %v", err)
	}
	if result {
		t.Error("Acquire should return false with nil client")
	}
}

func TestDistributedLockReleaseWithNilClient(t *testing.T) {
	lock := NewDistributedLock(nil, "test_key", "test_value", 10*time.Second)

	ctx := context.Background()
	err := lock.Release(ctx)
	if err != nil {
		t.Logf("Release returned error (expected with nil client): %v", err)
	}
}

func TestDistributedLockExtendWithNilClient(t *testing.T) {
	lock := NewDistributedLock(nil, "test_key", "test_value", 10*time.Second)

	ctx := context.Background()
	err := lock.Extend(ctx, 10*time.Second)
	if err != nil {
		t.Logf("Extend returned error (expected with nil client): %v", err)
	}
}

func TestPipelineOptimizer(t *testing.T) {
	optimizer := NewPipelineOptimizer(nil, 100)
	if optimizer == nil {
		t.Fatal("NewPipelineOptimizer should not return nil")
	}

	if optimizer.batchSize != 100 {
		t.Errorf("Batch size = %d, want %d", optimizer.batchSize, 100)
	}
}

func TestPipelineOptimizerWithZeroBatchSize(t *testing.T) {
	optimizer := NewPipelineOptimizer(nil, 0)
	if optimizer.batchSize != 100 {
		t.Errorf("Default batch size should be 100, got %d", optimizer.batchSize)
	}
}

func TestPipelineOptimizerWithNegativeBatchSize(t *testing.T) {
	optimizer := NewPipelineOptimizer(nil, -10)
	if optimizer.batchSize != 100 {
		t.Errorf("Default batch size should be 100, got %d", optimizer.batchSize)
	}
}

func TestPipelineSetWithNilClient(t *testing.T) {
	optimizer := NewPipelineOptimizer(nil, 100)

	ctx := context.Background()
	items := map[string]string{"key1": "value1", "key2": "value2"}
	err := optimizer.PipelineSet(ctx, items, 10*time.Second)
	if err == nil {
		t.Error("PipelineSet should return error with nil client")
	}
}

func TestPipelineGetWithNilClient(t *testing.T) {
	optimizer := NewPipelineOptimizer(nil, 100)

	ctx := context.Background()
	keys := []string{"key1", "key2"}
	result, err := optimizer.PipelineGet(ctx, keys)
	if err == nil {
		t.Error("PipelineGet should return error with nil client")
	}
	if result != nil {
		t.Error("Result should be nil with nil client")
	}
}

func TestPoolStats(t *testing.T) {
	stats := &PoolStats{
		TotalConns:      100,
		IdleConns:       20,
		StaleConns:      5,
		MetricsTotal:    1000,
		MetricsHits:     950,
		MetricsMisses:   50,
		MetricsTimeouts: 10,
	}

	if stats.TotalConns != 100 {
		t.Errorf("TotalConns = %d, want %d", stats.TotalConns, 100)
	}

	if stats.MetricsTotal != stats.MetricsHits+stats.MetricsMisses {
		t.Error("MetricsTotal should equal Hits + Misses")
	}
}

func TestConfig(t *testing.T) {
	cfg := &Config{
		PoolSize:     100,
		MinIdleConns: 10,
		MaxIdleConns: 50,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	}

	if cfg.PoolSize != 100 {
		t.Errorf("PoolSize = %d, want %d", cfg.PoolSize, 100)
	}

	if cfg.MinIdleConns != 10 {
		t.Errorf("MinIdleConns = %d, want %d", cfg.MinIdleConns, 10)
	}
}

func TestGetContext(t *testing.T) {
	ctx := GetContext()
	if ctx == nil {
		t.Fatal("GetContext should not return nil")
	}
}

func TestGetClient(t *testing.T) {
	client := GetClient()
	if client == nil {
		t.Log("GetClient returns nil when no client is connected")
	}
}

func TestGetClusterClient(t *testing.T) {
	client := GetClusterClient()
	if client == nil {
		t.Log("GetClusterClient returns nil when no cluster is connected")
	}
}

func TestNewRedisClusterWithEmptyAddrs(t *testing.T) {
	_, err := NewRedisCluster([]string{})
	if err == nil {
		t.Error("NewRedisCluster with empty addrs should return error")
	}
}

func TestRedisClientWithNilClient(t *testing.T) {
	rc := &RedisClient{client: nil}

	_, err := rc.Get(context.Background(), "test_key")
	if err == nil {
		t.Error("Get with nil client should return error")
	}

	err = rc.Set(context.Background(), "test_key", "value", 10*time.Second)
	if err == nil {
		t.Error("Set with nil client should return error")
	}

	err = rc.Delete(context.Background(), "test_key")
	if err == nil {
		t.Error("Delete with nil client should return error")
	}

	_, err = rc.Exists(context.Background(), "test_key")
	if err == nil {
		t.Error("Exists with nil client should return error")
	}

	err = rc.Expire(context.Background(), "test_key", 10*time.Second)
	if err == nil {
		t.Error("Expire with nil client should return error")
	}

	_, err = rc.TTL(context.Background(), "test_key")
	if err == nil {
		t.Error("TTL with nil client should return error")
	}

	_, err = rc.Incr(context.Background(), "test_key")
	if err == nil {
		t.Error("Incr with nil client should return error")
	}

	_, err = rc.IncrBy(context.Background(), "test_key", 1)
	if err == nil {
		t.Error("IncrBy with nil client should return error")
	}
}

func TestRedisClientGetPoolStats(t *testing.T) {
	rc := &RedisClient{client: nil}

	stats := rc.GetPoolStats()
	if stats == nil {
		t.Fatal("GetPoolStats should not return nil")
	}
}

func TestRedisClientSetPoolConfig(t *testing.T) {
	rc := &RedisClient{client: nil}

	err := rc.SetPoolConfig(nil)
	if err == nil {
		t.Error("SetPoolConfig with nil client should return error")
	}
}

func TestRedisClientClose(t *testing.T) {
	rc := &RedisClient{client: nil}

	err := rc.Close()
	if err != nil {
		t.Errorf("Close with nil client should not return error: %v", err)
	}
}

func TestRedisClusterWithNilClient(t *testing.T) {
	rc := &RedisCluster{client: nil}

	err := rc.Set(context.Background(), "test_key", "value")
	if err == nil {
		t.Error("Set with nil client should return error")
	}

	err = rc.SetWithTTL(context.Background(), "test_key", "value", 10*time.Second)
	if err == nil {
		t.Error("SetWithTTL with nil client should return error")
	}

	_, err = rc.Get(context.Background(), "test_key")
	if err == nil {
		t.Error("Get with nil client should return error")
	}

	err = rc.Delete(context.Background(), "test_key")
	if err == nil {
		t.Error("Delete with nil client should return error")
	}

	_, err = rc.Exists(context.Background(), "test_key")
	if err == nil {
		t.Error("Exists with nil client should return error")
	}

	err = rc.Expire(context.Background(), "test_key", 10*time.Second)
	if err == nil {
		t.Error("Expire with nil client should return error")
	}

	_, err = rc.TTL(context.Background(), "test_key")
	if err == nil {
		t.Error("TTL with nil client should return error")
	}

	_, err = rc.Incr(context.Background(), "test_key")
	if err == nil {
		t.Error("Incr with nil client should return error")
	}
}

func TestRedisClusterClose(t *testing.T) {
	rc := &RedisCluster{client: nil}

	err := rc.Close()
	if err != nil {
		t.Errorf("Close with nil client should not return error: %v", err)
	}
}
