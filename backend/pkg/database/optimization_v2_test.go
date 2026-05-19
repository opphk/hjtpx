package database

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestEnhancedDBRouter(t *testing.T) {
	router := &EnhancedDBRouter{
		enabled:         true,
		loadBalanceMode: "round_robin",
		slaveDBs:        make([]*gorm.DB, 0),
		slaveStatus:     make([]*SlaveNodeStatus, 0),
	}

	if !router.IsEnabled() {
		t.Error("Router should be enabled")
	}

	if router.loadBalanceMode != "round_robin" {
		t.Errorf("Load balance mode = %s, want round_robin", router.loadBalanceMode)
	}

	router.SetLoadBalanceMode("least_latency")
	if router.loadBalanceMode != "least_latency" {
		t.Errorf("Load balance mode = %s, want least_latency", router.loadBalanceMode)
	}
}

func (r *EnhancedDBRouter) SetLoadBalanceMode(mode string) {
	r.mu.Lock()
	r.loadBalanceMode = mode
	r.mu.Unlock()
}

func TestSlaveNodeStatus(t *testing.T) {
	status := &SlaveNodeStatus{
		Index:          0,
		Host:           "localhost",
		Port:           "5432",
		Healthy:        true,
		Latency:        10 * time.Millisecond,
		ReplicationLag: 5 * time.Second,
		Weight:         100,
	}

	if status.Index != 0 {
		t.Errorf("Index = %d, want 0", status.Index)
	}
	if !status.Healthy {
		t.Error("Healthy should be true")
	}
	if status.Latency != 10*time.Millisecond {
		t.Errorf("Latency = %v, want 10ms", status.Latency)
	}
}

func TestReplicationMonitor(t *testing.T) {
	monitor := NewReplicationMonitor(nil, 5*time.Second, 30*time.Second)

	if monitor.checkInterval != 5*time.Second {
		t.Errorf("checkInterval = %v, want 5s", monitor.checkInterval)
	}
	if monitor.maxLagThreshold != 30*time.Second {
		t.Errorf("maxLagThreshold = %v, want 30s", monitor.maxLagThreshold)
	}
	if !monitor.enabled {
		t.Error("enabled should be true")
	}
}

func TestEnhancedIndexOptimizer(t *testing.T) {
	optimizer := NewEnhancedIndexOptimizer(nil)

	if optimizer == nil {
		t.Error("Expected optimizer to be created")
	}

	if optimizer.minQueryCount != 100 {
		t.Errorf("minQueryCount = %d, want 100", optimizer.minQueryCount)
	}

	if !optimizer.enableAutoCreate {
		t.Error("enableAutoCreate should be true")
	}

	optimizer.SetAutoCreate(false)
	if optimizer.enableAutoCreate {
		t.Error("enableAutoCreate should be false after disable")
	}
}

func TestEnhancedIndexRecommendation(t *testing.T) {
	rec := &EnhancedIndexRecommendation{
		TableName: "test_table",
		IndexName: "idx_test",
		Columns:   []string{"col1", "col2"},
		IndexType: "btree",
		Priority:  "high",
		Action:    "create",
		Confidence: 0.9,
	}

	if rec.TableName != "test_table" {
		t.Errorf("TableName = %s, want test_table", rec.TableName)
	}
	if len(rec.Columns) != 2 {
		t.Errorf("Columns length = %d, want 2", len(rec.Columns))
	}
	if rec.Priority != "high" {
		t.Errorf("Priority = %s, want high", rec.Priority)
	}
	if rec.Confidence != 0.9 {
		t.Errorf("Confidence = %f, want 0.9", rec.Confidence)
	}
}

func TestQueryPatternAnalysis(t *testing.T) {
	analysis := &QueryPatternAnalysis{
		Pattern:        "SELECT * FROM users WHERE id = ?",
		ExecutionCount: 1000,
		AvgDuration:    15 * time.Millisecond,
	}

	if analysis.Pattern != "SELECT * FROM users WHERE id = ?" {
		t.Errorf("Pattern = %s, want SELECT * FROM users WHERE id = ?", analysis.Pattern)
	}
	if analysis.ExecutionCount != 1000 {
		t.Errorf("ExecutionCount = %d, want 1000", analysis.ExecutionCount)
	}
	if analysis.AvgDuration != 15*time.Millisecond {
		t.Errorf("AvgDuration = %v, want 15ms", analysis.AvgDuration)
	}
}

func TestL1MemoryCache(t *testing.T) {
	cache := NewL1MemoryCache(100, 5*time.Minute, L1StrategyLRU)

	if cache.maxSize != 100 {
		t.Errorf("maxSize = %d, want 100", cache.maxSize)
	}
	if cache.ttl != 5*time.Minute {
		t.Errorf("ttl = %v, want 5m", cache.ttl)
	}

	ctx := context.Background()
	cache.Set(ctx, "test_key", "test_value")

	value, found := cache.Get(ctx, "test_key")
	if !found {
		t.Error("Expected to find key")
	}
	if value != "test_value" {
		t.Errorf("Value = %v, want test_value", value)
	}

	cache.Delete("test_key")
	_, found = cache.Get(ctx, "test_key")
	if found {
		t.Error("Expected key to be deleted")
	}
}

func TestL1EvictionStrategies(t *testing.T) {
	cache := NewL1MemoryCache(3, 5*time.Minute, L1StrategyLRU)
	ctx := context.Background()

	cache.Set(ctx, "key1", "value1")
	cache.Set(ctx, "key2", "value2")
	cache.Set(ctx, "key3", "value3")
	cache.Set(ctx, "key4", "value4")

	_, found := cache.Get(ctx, "key1")
	if found {
		t.Error("Expected key1 to be evicted (LRU)")
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)

	if !cb.AllowRequest() {
		t.Error("Should allow request initially")
	}

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.AllowRequest() {
		t.Error("Should not allow request after 3 failures")
	}
}

func TestMultiLevelCacheConfig(t *testing.T) {
	config := &CacheConfig{
		L1MaxSize:      1000,
		L1TTL:          5 * time.Minute,
		L2TTL:          10 * time.Minute,
		EnableL1:       true,
		EnableL2:       true,
		WriteThrough:   true,
		CacheableTables: []string{"users", "posts"},
	}

	if config.L1MaxSize != 1000 {
		t.Errorf("L1MaxSize = %d, want 1000", config.L1MaxSize)
	}
	if !config.EnableL1 {
		t.Error("EnableL1 should be true")
	}
	if !config.EnableL2 {
		t.Error("EnableL2 should be true")
	}
}

func TestEnhancedPoolConfigV2(t *testing.T) {
	config := &EnhancedPoolConfig{
		MaxOpenConns:      100,
		MaxIdleConns:      50,
		MinIdleConns:      10,
		ConnMaxLifetime:   30 * time.Minute,
		ConnMaxIdleTime:   10 * time.Minute,
		EnableAutoTuning:  true,
		HighLoadThreshold: 0.8,
		LowLoadThreshold:  0.2,
	}

	if config.MaxOpenConns != 100 {
		t.Errorf("MaxOpenConns = %d, want 100", config.MaxOpenConns)
	}
	if config.MinIdleConns != 10 {
		t.Errorf("MinIdleConns = %d, want 10", config.MinIdleConns)
	}
	if !config.EnableAutoTuning {
		t.Error("EnableAutoTuning should be true")
	}
}

func TestHighLoadDetector(t *testing.T) {
	detector := NewHighLoadDetector(5, 0.8)

	if detector.threshold != 0.8 {
		t.Errorf("threshold = %f, want 0.8", detector.threshold)
	}

	isHigh := detector.RecordLoad(0.9)
	if isHigh {
		t.Error("Should not trigger on single high load")
	}

	detector.RecordLoad(0.9)
	isHigh = detector.RecordLoad(0.9)
	if !isHigh {
		t.Error("Should trigger after 3 consecutive high loads")
	}
}

func TestConnectionWarmupManager(t *testing.T) {
	wm := NewConnectionWarmupManager(10)

	if wm.targetConns != 10 {
		t.Errorf("targetConns = %d, want 10", wm.targetConns)
	}
	if wm.maxParallel != 5 {
		t.Errorf("maxParallel = %d, want 5", wm.maxParallel)
	}
}

func TestPoolHealthChecker(t *testing.T) {
	hc := NewPoolHealthChecker(nil, 30*time.Second)

	if hc.interval != 30*time.Second {
		t.Errorf("interval = %v, want 30s", hc.interval)
	}

	hc.addHealthHistory(true)
	hc.addHealthHistory(true)
	hc.addHealthHistory(false)

	score := hc.GetHealthScore()
	if score != 2.0/3.0 {
		t.Errorf("Health score = %f, want 0.666", score)
	}
}

func TestPoolMetricsCollector(t *testing.T) {
	mc := NewPoolMetricsCollector(10)

	if mc.maxHistory != 10 {
		t.Errorf("maxHistory = %d, want 10", mc.maxHistory)
	}

	snapshot := mc.takeSnapshot()
	if snapshot.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestTuningRecordV2(t *testing.T) {
	oldConfig := &EnhancedPoolConfig{MaxOpenConns: 100}
	newConfig := &EnhancedPoolConfig{MaxOpenConns: 150}

	record := &TuningRecord{
		Timestamp: time.Now(),
		OldConfig: oldConfig,
		NewConfig: newConfig,
		Reason:    "high load",
	}

	if record.OldConfig.MaxOpenConns != 100 {
		t.Errorf("OldConfig.MaxOpenConns = %d, want 100", record.OldConfig.MaxOpenConns)
	}
	if record.NewConfig.MaxOpenConns != 150 {
		t.Errorf("NewConfig.MaxOpenConns = %d, want 150", record.NewConfig.MaxOpenConns)
	}
	if record.Reason != "high load" {
		t.Errorf("Reason = %s, want high load", record.Reason)
	}
}

func TestPoolHealthStatusV2(t *testing.T) {
	status := &PoolHealthStatus{
		IsHealthy:       true,
		Score:           0.95,
		Issues:          []string{},
		Recommendations: []string{"optimize queries"},
		LastCheck:       time.Now(),
	}

	if !status.IsHealthy {
		t.Error("IsHealthy should be true")
	}
	if status.Score != 0.95 {
		t.Errorf("Score = %f, want 0.95", status.Score)
	}
}

func TestConnectionPressureV2(t *testing.T) {
	pressure := &ConnectionPressure{
		Timestamp:      time.Now(),
		OpenConnections: 100,
		InUse:          80,
		Idle:           20,
		WaitCount:      5,
		PressureLevel:  "high",
		Advice:         "consider increasing pool size",
	}

	if pressure.OpenConnections != 100 {
		t.Errorf("OpenConnections = %d, want 100", pressure.OpenConnections)
	}
	if pressure.PressureLevel != "high" {
		t.Errorf("PressureLevel = %s, want high", pressure.PressureLevel)
	}
}

func TestQueryCacheStrategy(t *testing.T) {
	strategy := StrategyLRU
	if strategy != 0 {
		t.Errorf("StrategyLRU should be 0, got %d", strategy)
	}

	strategy = StrategyLFU
	if strategy != 1 {
		t.Errorf("StrategyLFU should be 1, got %d", strategy)
	}
}

func TestL1EvictionStrategy(t *testing.T) {
	strategy := L1StrategyLRU
	if strategy != 0 {
		t.Errorf("L1StrategyLRU should be 0, got %d", strategy)
	}

	strategy = L1StrategyLFU
	if strategy != 1 {
		t.Errorf("L1StrategyLFU should be 1, got %d", strategy)
	}

	strategy = L1StrategyARC
	if strategy != 2 {
		t.Errorf("L1StrategyARC should be 2, got %d", strategy)
	}
}

func TestCacheConsistencyManager(t *testing.T) {
	cm := NewCacheConsistencyManager(nil, "timestamp")

	if cm.enabled {
		t.Error("enabled should be false when client is nil")
	}
}

func TestInvalidationMessage(t *testing.T) {
	msg := InvalidationMessage{
		Table:      "users",
		Key:        "user:1",
		InvalidateAll: false,
		Timestamp:  time.Now(),
	}

	if msg.Table != "users" {
		t.Errorf("Table = %s, want users", msg.Table)
	}
	if msg.Key != "user:1" {
		t.Errorf("Key = %s, want user:1", msg.Key)
	}
}