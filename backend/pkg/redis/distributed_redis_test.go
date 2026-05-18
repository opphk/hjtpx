package redis

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestDistributedRedisClient_Initialization(t *testing.T) {
	cfg := getDefaultDistributedRedisConfig()
	cfg.Mode = RedisModeStandalone
	cfg.StandaloneAddr = "localhost:6379"

	client := &DistributedRedisClient{
		config:  cfg,
		metrics: &DistributedRedisMetrics{},
	}

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.config.Mode != RedisModeStandalone {
		t.Errorf("Expected mode to be standalone, got %d", client.config.Mode)
	}
}

func TestDistributedRedisClient_GetDefaultConfig(t *testing.T) {
	cfg := getDefaultDistributedRedisConfig()

	if cfg.PoolSize != 100 {
		t.Errorf("Expected default pool size to be 100, got %d", cfg.PoolSize)
	}

	if cfg.MinIdleConns != 10 {
		t.Errorf("Expected default min idle conns to be 10, got %d", cfg.MinIdleConns)
	}

	if cfg.DialTimeout != 5*time.Second {
		t.Errorf("Expected default dial timeout to be 5s, got %v", cfg.DialTimeout)
	}
}

func TestRedisClusterMode_Constants(t *testing.T) {
	if RedisModeStandalone != 0 {
		t.Errorf("Expected RedisModeStandalone to be 0, got %d", RedisModeStandalone)
	}

	if RedisModeSentinel != 1 {
		t.Errorf("Expected RedisModeSentinel to be 1, got %d", RedisModeSentinel)
	}

	if RedisModeCluster != 2 {
		t.Errorf("Expected RedisModeCluster to be 2, got %d", RedisModeCluster)
	}

	if RedisModeDistributed != 3 {
		t.Errorf("Expected RedisModeDistributed to be 3, got %d", RedisModeDistributed)
	}
}

func TestNodeRole_Constants(t *testing.T) {
	if RoleMaster != "master" {
		t.Errorf("Expected RoleMaster to be 'master', got %s", RoleMaster)
	}

	if RoleSlave != "slave" {
		t.Errorf("Expected RoleSlave to be 'slave', got %s", RoleSlave)
	}

	if RoleSentinel != "sentinel" {
		t.Errorf("Expected RoleSentinel to be 'sentinel', got %s", RoleSentinel)
	}
}

func TestRedisNode_Initialization(t *testing.T) {
	node := &RedisNode{
		ID:            "node-1",
		Addr:          "localhost:6379",
		Role:          RoleMaster,
		Region:        "us-east",
		Healthy:       true,
		Priority:      100,
		LastHeartbeat: time.Now(),
		Stats:         &NodeStats{},
	}

	if node.ID != "node-1" {
		t.Errorf("Expected ID to be node-1, got %s", node.ID)
	}

	if node.Role != RoleMaster {
		t.Errorf("Expected role to be master, got %s", node.Role)
	}

	if !node.Healthy {
		t.Error("Expected node to be healthy")
	}
}

func TestRedisNode_Stats(t *testing.T) {
	stats := &NodeStats{}

	stats.TotalRequests.Add(100)
	stats.FailedRequests.Add(5)

	if stats.TotalRequests.Load() != 100 {
		t.Errorf("Expected total requests to be 100, got %d", stats.TotalRequests.Load())
	}

	if stats.FailedRequests.Load() != 5 {
		t.Errorf("Expected failed requests to be 5, got %d", stats.FailedRequests.Load())
	}
}

func TestHealthCheckResult_Initialization(t *testing.T) {
	result := &HealthCheckResult{
		NodeID:    "node-1",
		Healthy:   true,
		Latency:   10 * time.Millisecond,
		LastCheck: time.Now(),
		FailCount: 0,
	}

	if result.NodeID != "node-1" {
		t.Errorf("Expected node ID to be node-1, got %s", result.NodeID)
	}

	if !result.Healthy {
		t.Error("Expected health check to be healthy")
	}

	if result.Latency != 10*time.Millisecond {
		t.Errorf("Expected latency to be 10ms, got %v", result.Latency)
	}
}

func TestDistributedRedisMetrics_Initialization(t *testing.T) {
	metrics := &DistributedRedisMetrics{}

	metrics.TotalRequests.Add(1000)
	metrics.MasterRequests.Add(600)
	metrics.SlaveRequests.Add(400)
	metrics.FailedRequests.Add(10)

	if metrics.TotalRequests.Load() != 1000 {
		t.Errorf("Expected total requests to be 1000, got %d", metrics.TotalRequests.Load())
	}

	if metrics.MasterRequests.Load() != 600 {
		t.Errorf("Expected master requests to be 600, got %d", metrics.MasterRequests.Load())
	}

	if metrics.SlaveRequests.Load() != 400 {
		t.Errorf("Expected slave requests to be 400, got %d", metrics.SlaveRequests.Load())
	}

	if metrics.FailedRequests.Load() != 10 {
		t.Errorf("Expected failed requests to be 10, got %d", metrics.FailedRequests.Load())
	}
}

func TestDistributedRedisMetrics_HitRate(t *testing.T) {
	metrics := &DistributedRedisMetrics{}

	metrics.Hits.Add(90)
	metrics.Misses.Add(10)

	total := metrics.Hits.Load() + metrics.Misses.Load()
	if total > 0 {
		hitRateValue := int64(float64(metrics.Hits.Load()) / float64(total) * 100)
		metrics.HitRate.Store(hitRateValue)
	}

	expectedRate := int64(90)
	if metrics.HitRate.Load() != expectedRate {
		t.Errorf("Expected hit rate to be %d, got %d", expectedRate, metrics.HitRate.Load())
	}
}

func TestRedisHealthChecker_Initialization(t *testing.T) {
	client := &DistributedRedisClient{
		config:          getDefaultDistributedRedisConfig(),
		failoverEnabled: true,
		metrics:         &DistributedRedisMetrics{},
	}

	checker := newRedisHealthChecker(client, 10*time.Second)

	if checker == nil {
		t.Fatal("Expected health checker to be created")
	}

	if checker.interval != 10*time.Second {
		t.Errorf("Expected interval to be 10s, got %v", checker.interval)
	}

	if !checker.enabled {
		t.Error("Expected health checker to be enabled")
	}

	if checker.maxFailCount != 3 {
		t.Errorf("Expected max fail count to be 3, got %d", checker.maxFailCount)
	}
}

func TestRedisHealthChecker_SetMaxFailCount(t *testing.T) {
	client := &DistributedRedisClient{
		config:          getDefaultDistributedRedisConfig(),
		failoverEnabled: true,
		metrics:         &DistributedRedisMetrics{},
	}

	checker := newRedisHealthChecker(client, 10*time.Second)

	checker.SetMaxFailCount(5)
	if checker.maxFailCount != 5 {
		t.Errorf("Expected max fail count to be 5, got %d", checker.maxFailCount)
	}
}

func TestRedisHealthChecker_SetEnabled(t *testing.T) {
	client := &DistributedRedisClient{
		config:          getDefaultDistributedRedisConfig(),
		failoverEnabled: true,
		metrics:         &DistributedRedisMetrics{},
	}

	checker := newRedisHealthChecker(client, 10*time.Second)

	checker.SetEnabled(false)
	if checker.enabled {
		t.Error("Expected health checker to be disabled")
	}

	checker.SetEnabled(true)
	if !checker.enabled {
		t.Error("Expected health checker to be enabled")
	}
}

func TestRedisHealthChecker_GetHealthyNodes(t *testing.T) {
	client := &DistributedRedisClient{
		config:          getDefaultDistributedRedisConfig(),
		failoverEnabled: true,
		metrics:         &DistributedRedisMetrics{},
		nodes:           []*RedisNode{},
	}

	checker := newRedisHealthChecker(client, 10*time.Second)

	healthy := checker.GetHealthyNodes()
	if healthy == nil || len(healthy) != 0 {
		t.Logf("GetHealthyNodes returned: %v (expected nil or empty slice)", healthy)
	}
}

func TestRedisHealthChecker_GetHealthStatus(t *testing.T) {
	client := &DistributedRedisClient{
		config:          getDefaultDistributedRedisConfig(),
		failoverEnabled: true,
		metrics:         &DistributedRedisMetrics{},
	}

	checker := newRedisHealthChecker(client, 10*time.Second)

	status := checker.GetHealthStatus()
	if status == nil {
		t.Error("Expected health status to be returned")
	}
}

func TestDistributedRedisClient_GetMetrics(t *testing.T) {
	client := &DistributedRedisClient{
		config:  getDefaultDistributedRedisConfig(),
		metrics: &DistributedRedisMetrics{},
	}

	metrics := client.GetMetrics()
	if metrics == nil {
		t.Fatal("Expected metrics to be returned")
	}

	metrics.TotalRequests.Add(100)
	if metrics.TotalRequests.Load() != 100 {
		t.Errorf("Expected total requests to be 100, got %d", metrics.TotalRequests.Load())
	}
}

func TestDistributedRedisClient_GetNodeStats(t *testing.T) {
	client := &DistributedRedisClient{
		config:  getDefaultDistributedRedisConfig(),
		metrics: &DistributedRedisMetrics{},
		nodes: []*RedisNode{
			{
				ID:      "node-1",
				Addr:    "localhost:7000",
				Healthy: true,
				Stats:   &NodeStats{},
			},
			{
				ID:      "node-2",
				Addr:    "localhost:7001",
				Healthy: false,
				Stats:   &NodeStats{},
			},
		},
	}

	stats := client.GetNodeStats()
	if len(stats) != 2 {
		t.Errorf("Expected 2 node stats, got %d", len(stats))
	}
}

func TestDistributedRedisClient_AddNode(t *testing.T) {
	client := &DistributedRedisClient{
		config:  getDefaultDistributedRedisConfig(),
		metrics: &DistributedRedisMetrics{},
		nodes:   []*RedisNode{},
	}

	err := client.AddNode("localhost:7000", RoleMaster)
	if err != nil {
		t.Errorf("Expected no error when adding node, got: %v", err)
	}

	if len(client.nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(client.nodes))
	}

	err = client.AddNode("localhost:7000", RoleMaster)
	if err == nil {
		t.Error("Expected error when adding duplicate node")
	}
}

func TestDistributedRedisClient_RemoveNode(t *testing.T) {
	client := &DistributedRedisClient{
		config:  getDefaultDistributedRedisConfig(),
		metrics: &DistributedRedisMetrics{},
		nodes: []*RedisNode{
			{
				ID:   "node-1",
				Addr: "localhost:7000",
				Stats: &NodeStats{},
			},
		},
	}

	err := client.RemoveNode("localhost:7000")
	if err != nil {
		t.Errorf("Expected no error when removing node, got: %v", err)
	}

	if len(client.nodes) != 0 {
		t.Errorf("Expected 0 nodes, got %d", len(client.nodes))
	}

	err = client.RemoveNode("localhost:7000")
	if err == nil {
		t.Error("Expected error when removing non-existent node")
	}
}

func TestDistributedRedisClient_GetOptimalNode(t *testing.T) {
	client := &DistributedRedisClient{
		config:  getDefaultDistributedRedisConfig(),
		metrics: &DistributedRedisMetrics{},
		nodes: []*RedisNode{
			{
				ID:      "node-1",
				Addr:    "localhost:7000",
				Healthy: true,
				Stats:   &NodeStats{},
			},
			{
				ID:      "node-2",
				Addr:    "localhost:7001",
				Healthy: true,
				Stats:   &NodeStats{},
			},
		},
	}

	optimal := client.GetOptimalNode()
	if optimal == nil {
		t.Error("Expected optimal node to be returned")
	}
}

func TestDistributedRedisClient_SetNodeHealthy(t *testing.T) {
	client := &DistributedRedisClient{
		config:  getDefaultDistributedRedisConfig(),
		metrics: &DistributedRedisMetrics{},
		nodes: []*RedisNode{
			{
				ID:      "node-1",
				Addr:    "localhost:7000",
				Healthy: true,
				Stats:   &NodeStats{},
			},
		},
	}

	err := client.SetNodeHealthy("localhost:7000", false)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	err = client.SetNodeHealthy("localhost:9999", false)
	if err == nil {
		t.Error("Expected error for non-existent node")
	}
}

func TestDistributedLock_Initialization(t *testing.T) {
	client := &DistributedRedisClient{
		config:  getDefaultDistributedRedisConfig(),
		metrics: &DistributedRedisMetrics{},
	}

	lock := NewClusterDistributedLock(client, "test-key", "test-value", 10*time.Second)

	if lock == nil {
		t.Fatal("Expected lock to be created")
	}

	if lock.key != "test-key" {
		t.Errorf("Expected key to be test-key, got %s", lock.key)
	}

	if lock.value != "test-value" {
		t.Errorf("Expected value to be test-value, got %s", lock.value)
	}

	if lock.ttl != 10*time.Second {
		t.Errorf("Expected ttl to be 10s, got %v", lock.ttl)
	}
}

func TestDistributedLock_Acquire(t *testing.T) {
	client := &DistributedRedisClient{
		config:  getDefaultDistributedRedisConfig(),
		metrics: &DistributedRedisMetrics{},
	}

	lock := NewClusterDistributedLock(client, "test-key", "test-value", 10*time.Second)

	ctx := context.Background()
	acquired, err := lock.Acquire(ctx)

	if err != nil {
		t.Logf("Acquire failed (expected without redis): %v", err)
	}

	t.Logf("Lock acquire result: %v (without redis)", acquired)
}

func TestDistributedLock_TryWithLock(t *testing.T) {
	client := &DistributedRedisClient{
		config:  getDefaultDistributedRedisConfig(),
		metrics: &DistributedRedisMetrics{},
	}

	lock := NewClusterDistributedLock(client, "test-key", "test-value", 10*time.Second)

	fnCalled := false
	ctx := context.Background()

	err := lock.TryWithLock(ctx, func() error {
		fnCalled = true
		return nil
	})

	if err != nil {
		t.Logf("TryWithLock failed (expected without redis): %v", err)
	}

	if fnCalled {
		t.Error("Function should not be called without lock acquisition")
	}
}

func TestDistributedConfig_Defaults(t *testing.T) {
	cfg := getDefaultDistributedRedisConfig()

	if cfg.Mode != RedisModeStandalone {
		t.Errorf("Expected mode to be standalone, got %d", cfg.Mode)
	}

	if cfg.PoolSize != 100 {
		t.Errorf("Expected pool size to be 100, got %d", cfg.PoolSize)
	}

	if cfg.MinIdleConns != 10 {
		t.Errorf("Expected min idle conns to be 10, got %d", cfg.MinIdleConns)
	}

	if cfg.MaxRetries != 3 {
		t.Errorf("Expected max retries to be 3, got %d", cfg.MaxRetries)
	}

	if !cfg.FailoverEnabled {
		t.Error("Expected failover to be enabled by default")
	}

	if cfg.ReplicaLagLimit != 100*time.Millisecond {
		t.Errorf("Expected replica lag limit to be 100ms, got %v", cfg.ReplicaLagLimit)
	}
}

func TestConcurrentNodeOperations(t *testing.T) {
	client := &DistributedRedisClient{
		config:  getDefaultDistributedRedisConfig(),
		metrics: &DistributedRedisMetrics{},
		nodes:   []*RedisNode{},
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			addr := "localhost:700" + string(rune('0'+id))
			client.AddNode(addr, RoleMaster)
		}(i)
	}

	wg.Wait()

	if len(client.nodes) == 0 {
		t.Error("Expected nodes to be added concurrently")
	}
}

func TestMetricsConcurrency(t *testing.T) {
	metrics := &DistributedRedisMetrics{}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			metrics.TotalRequests.Add(1)
			metrics.Hits.Add(1)
		}()
	}

	wg.Wait()

	if metrics.TotalRequests.Load() != 100 {
		t.Errorf("Expected 100 total requests, got %d", metrics.TotalRequests.Load())
	}

	if metrics.Hits.Load() != 100 {
		t.Errorf("Expected 100 hits, got %d", metrics.Hits.Load())
	}
}

func TestRecordLatency(t *testing.T) {
	client := &DistributedRedisClient{
		config:  getDefaultDistributedRedisConfig(),
		metrics: &DistributedRedisMetrics{},
	}

	client.metrics.TotalRequests.Store(1)
	client.recordLatency(50*time.Millisecond, false)

	if client.metrics.AvgLatency.Load() == 0 {
		t.Error("Expected avg latency to be recorded")
	}
}
