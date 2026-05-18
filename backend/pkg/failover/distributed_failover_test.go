package failover

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDistributedFailoverManager_Initialization(t *testing.T) {
	cfg := getDefaultFailoverConfig()
	cfg.Enabled = true

	mgr := NewDistributedFailoverManager(cfg)
	if mgr == nil {
		t.Fatal("Expected failover manager to be created")
	}

	if !mgr.failoverEnabled {
		t.Error("Expected failover to be enabled")
	}

	if len(mgr.strategies) == 0 {
		t.Error("Expected default strategies to be registered")
	}
}

func TestFailoverConfig_Defaults(t *testing.T) {
	cfg := getDefaultFailoverConfig()

	if !cfg.Enabled {
		t.Error("Expected failover to be enabled by default")
	}

	if !cfg.AutoDetectEnabled {
		t.Error("Expected auto-detect to be enabled by default")
	}

	if !cfg.RegionFailover {
		t.Error("Expected region failover to be enabled by default")
	}

	if cfg.MaxRetries != 3 {
		t.Errorf("Expected max retries to be 3, got %d", cfg.MaxRetries)
	}

	if cfg.RetryInterval != 5*time.Second {
		t.Errorf("Expected retry interval to be 5s, got %v", cfg.RetryInterval)
	}
}

func TestDistributedFailoverManager_RegisterStrategy(t *testing.T) {
	cfg := getDefaultFailoverConfig()
	cfg.Enabled = true

	mgr := NewDistributedFailoverManager(cfg)

	initialCount := len(mgr.strategies)

	strategy := NewNodeFailoverStrategy()
	mgr.RegisterStrategy(strategy)

	if len(mgr.strategies) <= initialCount {
		t.Error("Expected strategy to be registered")
	}
}

func TestDistributedFailoverManager_SetFailoverEnabled(t *testing.T) {
	cfg := getDefaultFailoverConfig()
	cfg.Enabled = true

	mgr := NewDistributedFailoverManager(cfg)

	mgr.SetFailoverEnabled(false)
	if mgr.failoverEnabled {
		t.Error("Expected failover to be disabled")
	}

	mgr.SetFailoverEnabled(true)
	if !mgr.failoverEnabled {
		t.Error("Expected failover to be enabled")
	}
}

func TestDistributedFailoverManager_GetMetrics(t *testing.T) {
	cfg := getDefaultFailoverConfig()
	cfg.Enabled = true

	mgr := NewDistributedFailoverManager(cfg)

	metrics := mgr.GetMetrics()
	if metrics == nil {
		t.Fatal("Expected metrics to be returned")
	}

	metrics.TotalFailovers.Add(10)
	metrics.SuccessfulFailovers.Add(8)
	metrics.FailedFailovers.Add(2)

	if metrics.TotalFailovers.Load() != 10 {
		t.Errorf("Expected 10 total failovers, got %d", metrics.TotalFailovers.Load())
	}

	if metrics.SuccessfulFailovers.Load() != 8 {
		t.Errorf("Expected 8 successful failovers, got %d", metrics.SuccessfulFailovers.Load())
	}
}

func TestDistributedFailoverManager_TriggerFailover(t *testing.T) {
	cfg := getDefaultFailoverConfig()
	cfg.Enabled = true

	mgr := NewDistributedFailoverManager(cfg)

	info := &FailoverInfo{
		ID:        "failover-1",
		Type:      FailoverTypeNode,
		Source:    "redis-node-1",
		Reason:    "connection timeout",
		Severity:  SeverityHigh,
		Timestamp: time.Now(),
	}

	err := mgr.TriggerFailover(context.Background(), info)
	if err != nil {
		t.Logf("TriggerFailover failed (expected without dependencies): %v", err)
	}

	mgr.Stop()
}

func TestFailoverMetrics(t *testing.T) {
	metrics := &FailoverMetrics{}

	metrics.TotalFailovers.Add(100)
	metrics.SuccessfulFailovers.Add(85)
	metrics.FailedFailovers.Add(15)
	metrics.RegionFailovers.Add(20)
	metrics.NodeFailovers.Add(80)

	if metrics.TotalFailovers.Load() != 100 {
		t.Errorf("Expected 100 total failovers, got %d", metrics.TotalFailovers.Load())
	}

	if metrics.SuccessfulFailovers.Load() != 85 {
		t.Errorf("Expected 85 successful failovers, got %d", metrics.SuccessfulFailovers.Load())
	}

	if metrics.RegionFailovers.Load() != 20 {
		t.Errorf("Expected 20 region failovers, got %d", metrics.RegionFailovers.Load())
	}

	if metrics.NodeFailovers.Load() != 80 {
		t.Errorf("Expected 80 node failovers, got %d", metrics.NodeFailovers.Load())
	}
}

func TestNodeFailoverStrategy_Name(t *testing.T) {
	strategy := NewNodeFailoverStrategy()
	if strategy.Name() != "node_failover" {
		t.Errorf("Expected name to be node_failover, got %s", strategy.Name())
	}
}

func TestNodeFailoverStrategy_CanHandle(t *testing.T) {
	strategy := NewNodeFailoverStrategy()

	info := &FailoverInfo{
		Type:     FailoverTypeNode,
		Source:   "redis-node-1",
		Reason:   "connection timeout",
		Severity: SeverityHigh,
	}

	if !strategy.CanHandle(info) {
		t.Error("Expected node failover strategy to handle node failover")
	}
}

func TestNodeFailoverStrategy_Execute(t *testing.T) {
	strategy := NewNodeFailoverStrategy()

	info := &FailoverInfo{
		ID:        "failover-1",
		Type:      FailoverTypeNode,
		Source:    "redis-node-1",
		Reason:    "connection timeout",
		Severity:  SeverityHigh,
		Timestamp: time.Now(),
	}

	err := strategy.Execute(context.Background(), info)
	if err != nil {
		t.Logf("Execute failed (expected without dependencies): %v", err)
	}
}

func TestRegionFailoverStrategy_Name(t *testing.T) {
	strategy := NewRegionFailoverStrategy()
	if strategy.Name() != "region_failover" {
		t.Errorf("Expected name to be region_failover, got %s", strategy.Name())
	}
}

func TestRegionFailoverStrategy_CanHandle(t *testing.T) {
	strategy := NewRegionFailoverStrategy()

	info := &FailoverInfo{
		Type:     FailoverTypeRegion,
		Source:   "us-east",
		Target:   "us-west",
		Reason:   "region unhealthy",
		Severity: SeverityCritical,
	}

	if !strategy.CanHandle(info) {
		t.Error("Expected region failover strategy to handle region failover")
	}

	info.Type = FailoverTypeDatacenter
	if !strategy.CanHandle(info) {
		t.Error("Expected region failover strategy to handle datacenter failover")
	}
}

func TestRegionFailoverStrategy_Execute(t *testing.T) {
	strategy := NewRegionFailoverStrategy()

	info := &FailoverInfo{
		ID:        "failover-1",
		Type:      FailoverTypeRegion,
		Source:    "us-east",
		Target:   "us-west",
		Reason:   "region unhealthy",
		Severity:  SeverityCritical,
		Timestamp: time.Now(),
	}

	err := strategy.Execute(context.Background(), info)
	if err != nil {
		t.Logf("Execute failed (expected without dependencies): %v", err)
	}
}

func TestDatabaseFailoverStrategy_Name(t *testing.T) {
	strategy := NewDatabaseFailoverStrategy()
	if strategy.Name() != "database_failover" {
		t.Errorf("Expected name to be database_failover, got %s", strategy.Name())
	}
}

func TestDatabaseFailoverStrategy_CanHandle(t *testing.T) {
	strategy := NewDatabaseFailoverStrategy()

	info := &FailoverInfo{
		Type:     FailoverTypeNode,
		Source:   "db:localhost:5432",
		Reason:   "connection error",
		Severity: SeverityError,
	}

	if !strategy.CanHandle(info) {
		t.Error("Expected database failover strategy to handle database failover")
	}
}

func TestDatabaseFailoverStrategy_Execute(t *testing.T) {
	strategy := NewDatabaseFailoverStrategy()

	info := &FailoverInfo{
		ID:        "failover-1",
		Type:      FailoverTypeNode,
		Source:    "db:localhost:5432",
		Reason:   "connection error",
		Severity:  SeverityError,
		Timestamp: time.Now(),
	}

	err := strategy.Execute(context.Background(), info)
	if err != nil {
		t.Logf("Execute failed (expected without dependencies): %v", err)
	}
}

func TestRedisFailoverStrategy_Name(t *testing.T) {
	strategy := NewRedisFailoverStrategy()
	if strategy.Name() != "redis_failover" {
		t.Errorf("Expected name to be redis_failover, got %s", strategy.Name())
	}
}

func TestRedisFailoverStrategy_CanHandle(t *testing.T) {
	strategy := NewRedisFailoverStrategy()

	info := &FailoverInfo{
		Type:     FailoverTypeNode,
		Source:   "redis:localhost:6379",
		Reason:   "connection error",
		Severity: SeverityError,
	}

	if !strategy.CanHandle(info) {
		t.Error("Expected redis failover strategy to handle redis failover")
	}
}

func TestRedisFailoverStrategy_Execute(t *testing.T) {
	strategy := NewRedisFailoverStrategy()

	info := &FailoverInfo{
		ID:        "failover-1",
		Type:      FailoverTypeNode,
		Source:   "redis:localhost:6379",
		Reason:   "connection error",
		Severity:  SeverityError,
		Timestamp: time.Now(),
	}

	err := strategy.Execute(context.Background(), info)
	if err != nil {
		t.Logf("Execute failed (expected without dependencies): %v", err)
	}
}

func TestNetworkFailoverStrategy_Name(t *testing.T) {
	strategy := NewNetworkFailoverStrategy()
	if strategy.Name() != "network_failover" {
		t.Errorf("Expected name to be network_failover, got %s", strategy.Name())
	}
}

func TestNetworkFailoverStrategy_CanHandle(t *testing.T) {
	strategy := NewNetworkFailoverStrategy()

	info := &FailoverInfo{
		Type:     FailoverTypeNetwork,
		Source:   "network-1",
		Reason:   "network partition",
		Severity: SeverityWarning,
	}

	if !strategy.CanHandle(info) {
		t.Error("Expected network failover strategy to handle network failover")
	}
}

func TestNetworkFailoverStrategy_Execute(t *testing.T) {
	strategy := NewNetworkFailoverStrategy()

	info := &FailoverInfo{
		ID:        "failover-1",
		Type:      FailoverTypeNetwork,
		Source:   "network-1",
		Reason:   "network partition",
		Severity:  SeverityWarning,
		Timestamp: time.Now(),
	}

	err := strategy.Execute(context.Background(), info)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
}

func TestFailoverInfo_Initialization(t *testing.T) {
	info := &FailoverInfo{
		ID:            "failover-1",
		Type:          FailoverTypeNode,
		Source:        "node-1",
		Target:        "node-2",
		Region:        "us-east",
		DC:            "dc1",
		Reason:        "node failure",
		Severity:      SeverityCritical,
		Timestamp:     time.Now(),
		RetryCount:    0,
		Metadata:      map[string]string{"key": "value"},
	}

	if info.ID != "failover-1" {
		t.Errorf("Expected ID to be failover-1, got %s", info.ID)
	}

	if info.Type != FailoverTypeNode {
		t.Errorf("Expected type to be node, got %s", info.Type)
	}

	if info.Severity != SeverityCritical {
		t.Errorf("Expected severity to be critical, got %s", info.Severity)
	}
}

func TestRegionFailover_Initialization(t *testing.T) {
	rf := &RegionFailover{
		Name:            "us-east",
		PrimaryRegion:   "us-east-1",
		FallbackRegions: []string{"us-west-2", "eu-west-1"},
		HealthStatus:    make(map[string]RegionHealth),
		ActiveRegion:    "us-east-1",
	}

	if rf.Name != "us-east" {
		t.Errorf("Expected name to be us-east, got %s", rf.Name)
	}

	if rf.PrimaryRegion != "us-east-1" {
		t.Errorf("Expected primary region to be us-east-1, got %s", rf.PrimaryRegion)
	}

	if len(rf.FallbackRegions) != 2 {
		t.Errorf("Expected 2 fallback regions, got %d", len(rf.FallbackRegions))
	}

	rf.SwitchCount.Add(1)
	if rf.SwitchCount.Load() != 1 {
		t.Errorf("Expected switch count to be 1, got %d", rf.SwitchCount.Load())
	}
}

func TestRegionHealth_Initialization(t *testing.T) {
	rh := RegionHealth{
		Region:       "us-east",
		Healthy:      true,
		Latency:      50 * time.Millisecond,
		LastCheck:    time.Now(),
		FailureCount: 0,
		SuccessCount: 10,
	}

	if rh.Region != "us-east" {
		t.Errorf("Expected region to be us-east, got %s", rh.Region)
	}

	if !rh.Healthy {
		t.Error("Expected health to be true")
	}

	if rh.Latency != 50*time.Millisecond {
		t.Errorf("Expected latency to be 50ms, got %v", rh.Latency)
	}
}

func TestDistributedFailoverManager_RegisterRegion(t *testing.T) {
	cfg := getDefaultFailoverConfig()
	cfg.Enabled = true

	mgr := NewDistributedFailoverManager(cfg)

	regionCfg := &RegionFailoverConfig{
		Name:            "us-east",
		PrimaryRegion:   "us-east-1",
		FallbackRegions: []string{"us-west-2"},
	}

	mgr.RegisterRegion(regionCfg)

	if len(mgr.regions) != 1 {
		t.Errorf("Expected 1 region, got %d", len(mgr.regions))
	}

	region, exists := mgr.regions["us-east"]
	if !exists {
		t.Fatal("Expected region us-east to exist")
	}

	if region.PrimaryRegion != "us-east-1" {
		t.Errorf("Expected primary region to be us-east-1, got %s", region.PrimaryRegion)
	}
}

func TestDistributedFailoverManager_GetRegionStatus(t *testing.T) {
	cfg := getDefaultFailoverConfig()
	cfg.Enabled = true

	mgr := NewDistributedFailoverManager(cfg)

	mgr.regions["us-east"] = &RegionFailover{
		Name:            "us-east",
		PrimaryRegion:   "us-east-1",
		FallbackRegions: []string{"us-west-2"},
		ActiveRegion:    "us-east-1",
	}

	status := mgr.GetRegionStatus()
	if status == nil {
		t.Fatal("Expected region status to be returned")
	}

	if status["us-east"] == nil {
		t.Error("Expected us-east status to exist")
	}
}

func TestDistributedFailoverManager_IsFailoverActive(t *testing.T) {
	cfg := getDefaultFailoverConfig()
	cfg.Enabled = true

	mgr := NewDistributedFailoverManager(cfg)

	if mgr.IsFailoverActive() {
		t.Error("Expected failover to not be active initially")
	}

	mgr.metrics.CurrentActiveFailover.Store(true)
	if !mgr.IsFailoverActive() {
		t.Error("Expected failover to be active")
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr  string
		expected bool
	}{
		{"db:localhost:5432", "db", true},
		{"redis:localhost:6379", "redis", true},
		{"node:1", "node", true},
		{"", "db", false},
		{"db", "", true},
		{"abc", "d", false},
	}

	for _, tt := range tests {
		result := contains(tt.s, tt.substr)
		if result != tt.expected {
			t.Errorf("contains(%q, %q) = %v, expected %v", tt.s, tt.substr, result, tt.expected)
		}
	}
}

func TestFailoverType_Constants(t *testing.T) {
	if FailoverTypeNode != "node" {
		t.Errorf("Expected FailoverTypeNode to be 'node', got %s", FailoverTypeNode)
	}

	if FailoverTypeRegion != "region" {
		t.Errorf("Expected FailoverTypeRegion to be 'region', got %s", FailoverTypeRegion)
	}

	if FailoverTypeDatacenter != "datacenter" {
		t.Errorf("Expected FailoverTypeDatacenter to be 'datacenter', got %s", FailoverTypeDatacenter)
	}

	if FailoverTypeNetwork != "network" {
		t.Errorf("Expected FailoverTypeNetwork to be 'network', got %s", FailoverTypeNetwork)
	}
}

func TestFailoverSeverity_Constants(t *testing.T) {
	if SeverityInfo != "info" {
		t.Errorf("Expected SeverityInfo to be 'info', got %s", SeverityInfo)
	}

	if SeverityWarning != "warning" {
		t.Errorf("Expected SeverityWarning to be 'warning', got %s", SeverityWarning)
	}

	if SeverityError != "error" {
		t.Errorf("Expected SeverityError to be 'error', got %s", SeverityError)
	}

	if SeverityCritical != "critical" {
		t.Errorf("Expected SeverityCritical to be 'critical', got %s", SeverityCritical)
	}
}

func TestFailoverResult_Constants(t *testing.T) {
	if FailoverResultSuccess != "success" {
		t.Errorf("Expected FailoverResultSuccess to be 'success', got %s", FailoverResultSuccess)
	}

	if FailoverResultFailed != "failed" {
		t.Errorf("Expected FailoverResultFailed to be 'failed', got %s", FailoverResultFailed)
	}

	if FailoverResultSkipped != "skipped" {
		t.Errorf("Expected FailoverResultSkipped to be 'skipped', got %s", FailoverResultSkipped)
	}

	if FailoverResultPartial != "partial" {
		t.Errorf("Expected FailoverResultPartial to be 'partial', got %s", FailoverResultPartial)
	}
}

func TestConcurrentFailoverTrigger(t *testing.T) {
	cfg := getDefaultFailoverConfig()
	cfg.Enabled = true

	mgr := NewDistributedFailoverManager(cfg)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			info := &FailoverInfo{
				ID:        "failover-" + string(rune('0'+id)),
				Type:      FailoverTypeNode,
				Source:    "node-" + string(rune('0'+id)),
				Reason:    "test failure",
				Severity:  SeverityMedium,
				Timestamp: time.Now(),
			}
			mgr.TriggerFailover(context.Background(), info)
		}(i)
	}

	wg.Wait()

	mgr.Stop()
}

func TestFailoverMetrics_AtomicOperations(t *testing.T) {
	metrics := &FailoverMetrics{}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			metrics.TotalFailovers.Add(1)
		}()
	}

	wg.Wait()

	if metrics.TotalFailovers.Load() != 100 {
		t.Errorf("Expected 100 total failovers, got %d", metrics.TotalFailovers.Load())
	}
}

func TestRecordMetrics(t *testing.T) {
	cfg := getDefaultFailoverConfig()
	cfg.Enabled = true

	mgr := NewDistributedFailoverManager(cfg)

	info := &FailoverInfo{
		ID:        "failover-1",
		Type:      FailoverTypeRegion,
		Region:    "us-east",
		Reason:    "region unhealthy",
		Severity:  SeverityCritical,
		Timestamp: time.Now(),
	}

	mgr.recordMetrics(info, nil, 100*time.Millisecond)

	if mgr.metrics.TotalFailovers.Load() != 1 {
		t.Errorf("Expected 1 total failover, got %d", mgr.metrics.TotalFailovers.Load())
	}

	if mgr.metrics.RegionFailovers.Load() != 1 {
		t.Errorf("Expected 1 region failover, got %d", mgr.metrics.RegionFailovers.Load())
	}

	if mgr.metrics.SuccessfulFailovers.Load() != 1 {
		t.Errorf("Expected 1 successful failover, got %d", mgr.metrics.SuccessfulFailovers.Load())
	}

	mgr.Stop()
}
