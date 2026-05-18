package redis

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCacheKeyManager(t *testing.T) {
	manager := NewCacheKeyManager(NamespaceGlobal)
	if manager == nil {
		t.Fatal("NewCacheKeyManager should not return nil")
	}
}

func TestCacheKeyBuilder(t *testing.T) {
	builder := NewCacheKeyBuilder(PrefixCaptcha)
	
	result := builder.Namespace(NamespaceUser).AddSegment("user123").Build()
	expected := "captcha:user:user123"
	
	if result != expected {
		t.Errorf("Build() = %q, want %q", result, expected)
	}
}

func TestCacheKeyBuilderWithVersion(t *testing.T) {
	builder := NewCacheKeyBuilder(PrefixSession)
	
	result := builder.Version(2).BuildWithVersion()
	expected := "session::v2"
	if result != expected {
		t.Errorf("BuildWithVersion() = %q, want %q", result, expected)
	}
}

func TestCacheKeyBuilderBuildPattern(t *testing.T) {
	builder := NewCacheKeyBuilder(PrefixBlacklist)
	
	result := builder.BuildPattern()
	expected := "blacklist::*"
	
	if result != expected {
		t.Errorf("BuildPattern() = %q, want %q", result, expected)
	}
}

func TestBuildCaptchaKey(t *testing.T) {
	manager := GetCacheKeyManager()
	key := manager.BuildCaptchaKey("captcha123")
	
	if key == "" {
		t.Error("BuildCaptchaKey should not return empty string")
	}
	
	if key != "captcha:captcha123" {
		t.Errorf("BuildCaptchaKey() = %q, want %q", key, "captcha:captcha123")
	}
}

func TestBuildSessionKey(t *testing.T) {
	manager := GetCacheKeyManager()
	key := manager.BuildSessionKey("token456")
	
	expected := "session:token456"
	if key != expected {
		t.Errorf("BuildSessionKey() = %q, want %q", key, expected)
	}
}

func TestBuildApplicationKey(t *testing.T) {
	manager := GetCacheKeyManager()
	key := manager.BuildApplicationKey("api_key_789")
	
	expected := "app:api_key_789"
	if key != expected {
		t.Errorf("BuildApplicationKey() = %q, want %q", key, expected)
	}
}

func TestBuildRateLimitKey(t *testing.T) {
	manager := GetCacheKeyManager()
	key := manager.BuildRateLimitKey("identifier", 60)
	
	expected := "ratelimit:identifier:w60"
	if key != expected {
		t.Errorf("BuildRateLimitKey() = %q, want %q", key, expected)
	}
}

func TestCacheMonitoringCollector(t *testing.T) {
	collector := NewCacheMonitoringCollector(nil)
	if collector == nil {
		t.Fatal("NewCacheMonitoringCollector should not return nil")
	}
}

func TestCacheMonitoringCollectorRecordHit(t *testing.T) {
	collector := NewCacheMonitoringCollector(nil)
	
	collector.RecordHit()
	collector.RecordHit()
	collector.RecordMiss()
	
	metrics := collector.GetDetailedMetrics()
	
	if metrics["hits"].(int64) != 2 {
		t.Errorf("hits = %d, want 2", metrics["hits"])
	}
	
	if metrics["misses"].(int64) != 1 {
		t.Errorf("misses = %d, want 1", metrics["misses"])
	}
}

func TestCacheMonitoringCollectorRecordLatency(t *testing.T) {
	collector := NewCacheMonitoringCollector(nil)
	
	collector.RecordLatency(100 * time.Microsecond)
	collector.RecordLatency(5 * time.Millisecond)
	collector.RecordLatency(50 * time.Millisecond)
	
	metrics := collector.GetDetailedMetrics()
	
	if metrics["p95_latency_ms"].(float64) <= 0 {
		t.Error("P95 latency should be greater than 0")
	}
}

func TestCacheMonitoringCollectorRecordKeyAccess(t *testing.T) {
	collector := NewCacheMonitoringCollector(&CacheMonitoringConfig{
		EnableHotKeyTracking: true,
		HotKeyThreshold:     5,
	})
	
	for i := 0; i < 10; i++ {
		collector.RecordKeyAccess("hot_key_1")
	}
	
	collector.RecordKeyAccess("cold_key")
	
	hotKeys := collector.GetHotKeys(10)
	
	if len(hotKeys) == 0 {
		t.Error("Should have at least one hot key")
	}
}

func TestCacheMonitoringCollectorGetHealthStatus(t *testing.T) {
	collector := NewCacheMonitoringCollector(nil)
	
	for i := 0; i < 100; i++ {
		collector.RecordHit()
	}
	for i := 0; i < 20; i++ {
		collector.RecordMiss()
	}
	
	health := collector.GetHealthStatus()
	
	if health.Status != "healthy" {
		t.Errorf("Status = %q, want %q", health.Status, "healthy")
	}
	
	if health.HitRate < 80 {
		t.Errorf("HitRate = %f, want > 80", health.HitRate)
	}
}

func TestCacheMonitoringCollectorExportMetricsJSON(t *testing.T) {
	collector := NewCacheMonitoringCollector(nil)
	
	collector.RecordHit()
	collector.RecordHit()
	collector.RecordMiss()
	
	data, err := collector.ExportMetricsJSON()
	if err != nil {
		t.Fatalf("ExportMetricsJSON() error = %v", err)
	}
	
	if len(data) == 0 {
		t.Error("Exported JSON should not be empty")
	}
}

func TestCacheExpirationManager(t *testing.T) {
	manager := NewCacheExpirationManager(nil)
	if manager == nil {
		t.Fatal("NewCacheExpirationManager should not return nil")
	}
}

func TestCacheExpirationManagerCalculateTTL(t *testing.T) {
	manager := NewCacheExpirationManager(&CacheExpirationConfig{
		DefaultTTL:  10 * time.Minute,
		MinTTL:     1 * time.Minute,
		MaxTTL:     1 * time.Hour,
		Strategy:   ExpirationFixed,
	})
	
	ttl := manager.CalculateTTL("test_key", 10*time.Minute)
	
	if ttl != 10*time.Minute {
		t.Errorf("TTL = %v, want %v", ttl, 10*time.Minute)
	}
}

func TestCacheExpirationManagerCalculateTTLWithSliding(t *testing.T) {
	manager := NewCacheExpirationManager(&CacheExpirationConfig{
		DefaultTTL:    10 * time.Minute,
		MinTTL:       1 * time.Minute,
		MaxTTL:       1 * time.Hour,
		Strategy:     ExpirationSliding,
		SlidingWindow: 1 * time.Minute,
	})
	
	ttl := manager.CalculateTTL("test_key", 10*time.Minute)
	
	if ttl <= 10*time.Minute {
		t.Errorf("TTL = %v, want > 10*time.Minute for sliding strategy", ttl)
	}
}

func TestCacheExpirationManagerVersioning(t *testing.T) {
	manager := NewCacheExpirationManager(nil)
	
	version1 := manager.IncrementVersion("key1")
	version2 := manager.IncrementVersion("key1")
	version3 := manager.GetVersion("key1")
	
	if version1 != 1 {
		t.Errorf("First version = %d, want 1", version1)
	}
	
	if version2 != 2 {
		t.Errorf("Second version = %d, want 2", version2)
	}
	
	if version3 != 2 {
		t.Errorf("GetVersion = %d, want 2", version3)
	}
}

func TestCacheInvalidationManager(t *testing.T) {
	manager := NewCacheInvalidationManager(nil)
	if manager == nil {
		t.Fatal("NewCacheInvalidationManager should not return nil")
	}
}

func TestCacheInvalidationManagerBatchQueue(t *testing.T) {
	manager := NewCacheInvalidationManager(&CacheInvalidationConfig{
		Mode:          InvalidationBatch,
		BatchSize:     10,
		BatchInterval: 100 * time.Millisecond,
	})
	
	defer manager.Stop()
	
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key_%d", i)
		err := manager.Invalidate(context.Background(), key)
		if err != nil {
			t.Logf("Batch invalidate returned error (expected): %v", err)
		}
	}
}

func TestCacheInvalidationManagerVersioning(t *testing.T) {
	manager := NewCacheInvalidationManager(nil)
	
	version1 := manager.IncrementVersion("key1")
	version2 := manager.IncrementVersion("key1")
	
	if version1 != 1 {
		t.Errorf("First version = %d, want 1", version1)
	}
	
	if version2 != 2 {
		t.Errorf("Second version = %d, want 2", version2)
	}
}

func TestCacheConsistencyManager(t *testing.T) {
	manager := NewCacheConsistencyManager(100)
	if manager == nil {
		t.Fatal("NewCacheConsistencyManager should not return nil")
	}
}

func TestAdaptiveExpirationPolicy(t *testing.T) {
	policy := NewAdaptiveExpirationPolicy(
		10*time.Minute,
		1*time.Minute,
		1*time.Hour,
	)
	
	if policy == nil {
		t.Fatal("NewAdaptiveExpirationPolicy should not return nil")
	}
	
	for i := 0; i < 50; i++ {
		policy.RecordAccess("hot_key")
	}
	
	ttl := policy.CalculateTTL("hot_key")
	
	if ttl < 10*time.Minute {
		t.Errorf("TTL = %v, want >= 10*time.Minute", ttl)
	}
}

func TestCacheWarmupManager(t *testing.T) {
	manager := NewCacheWarmupManager(nil)
	if manager == nil {
		t.Fatal("NewCacheWarmupManager should not return nil")
	}
	
	tasks := manager.GetTasks()
	if len(tasks) != 0 {
		t.Errorf("Initial tasks count = %d, want 0", len(tasks))
	}
}

func TestCacheWarmupManagerAddTask(t *testing.T) {
	manager := NewCacheWarmupManager(nil)
	
	task := &CacheWarmupTask{
		Name:      "test_task",
		Key:       "warmup:test",
		TTL:       10 * time.Minute,
		Frequency: 5 * time.Minute,
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("test_data"), nil
		},
		Enabled:    true,
		MaxRetries: 3,
	}
	
	manager.AddTask(task)
	
	tasks := manager.GetTasks()
	if len(tasks) != 1 {
		t.Errorf("Tasks count after add = %d, want 1", len(tasks))
	}
}

func TestCacheWarmupManagerStartStop(t *testing.T) {
	manager := NewCacheWarmupManager(nil)
	
	task := &CacheWarmupTask{
		Name:      "test_task",
		Key:       "warmup:test",
		TTL:       10 * time.Minute,
		Frequency: 1 * time.Minute,
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("test_data"), nil
		},
		Enabled:    false,
		MaxRetries: 3,
	}
	
	manager.AddTask(task)
	manager.Start()
	defer manager.Stop()
	
	tasks := manager.GetTasks()
	if len(tasks) == 0 {
		t.Error("Should have tasks after start")
	}
}

func TestSmartWarmupStrategy(t *testing.T) {
	strategy := NewSmartWarmupStrategy(nil, 100)
	if strategy == nil {
		t.Fatal("NewSmartWarmupStrategy should not return nil")
	}
	
	for i := 0; i < 150; i++ {
		strategy.RecordAccess("hot_key_1")
	}
	
	recommendations := strategy.GetWarmupRecommendations()
	
	if len(recommendations) == 0 {
		t.Error("Should have warmup recommendations")
	}
}

func TestAccessTracker(t *testing.T) {
	tracker := NewAccessTracker()
	if tracker == nil {
		t.Fatal("NewAccessTracker should not return nil")
	}
	
	tracker.RecordAccess("key1")
	tracker.RecordAccess("key1")
	tracker.RecordAccess("key2")
	
	count1 := tracker.GetAccessCount("key1")
	count2 := tracker.GetAccessCount("key2")
	
	if count1 != 2 {
		t.Errorf("key1 count = %d, want 2", count1)
	}
	
	if count2 != 1 {
		t.Errorf("key2 count = %d, want 1", count2)
	}
}

func TestAccessTrackerHotKeys(t *testing.T) {
	tracker := NewAccessTracker()
	
	for i := 0; i < 10; i++ {
		tracker.RecordAccess("hot_key_1")
	}
	for i := 0; i < 5; i++ {
		tracker.RecordAccess("warm_key")
	}
	
	hotKeys := tracker.GetHotKeys(8)
	
	if len(hotKeys) != 1 {
		t.Errorf("Hot keys count = %d, want 1", len(hotKeys))
	}
}

func TestBatchWarmupProcessor(t *testing.T) {
	processor := NewBatchWarmupProcessor(100, 5)
	if processor == nil {
		t.Fatal("NewBatchWarmupProcessor should not return nil")
	}
	
	if processor.batchSize != 100 {
		t.Errorf("batchSize = %d, want 100", processor.batchSize)
	}
	
	if processor.workers != 5 {
		t.Errorf("workers = %d, want 5", processor.workers)
	}
}

func TestCacheWarmupStats(t *testing.T) {
	stats := NewWarmupStats()
	if stats == nil {
		t.Fatal("NewWarmupStats should not return nil")
	}
	
	stats.SuccessCount.Add(10)
	stats.FailureCount.Add(2)
	stats.TotalRuns.Add(12)
	
	if stats.SuccessCount.Load() != 10 {
		t.Errorf("SuccessCount = %d, want 10", stats.SuccessCount.Load())
	}
	
	if stats.FailureCount.Load() != 2 {
		t.Errorf("FailureCount = %d, want 2", stats.FailureCount.Load())
	}
}

func TestCacheHealthStatus(t *testing.T) {
	health := &CacheHealthStatus{
		Status:        "healthy",
		HitRate:       95.5,
		ErrorRate:     0.5,
		L1HitRate:     80.0,
		L2HitRate:     90.0,
		AvgLatency:    1.5,
		P95Latency:    5.0,
		P99Latency:    10.0,
		TotalRequests: 1000,
		TotalErrors:   5,
		HotKeyCount:   10,
		LastChecked:   time.Now(),
	}
	
	if health.Status != "healthy" {
		t.Errorf("Status = %q, want %q", health.Status, "healthy")
	}
	
	if health.HitRate < 90 {
		t.Errorf("HitRate = %f, want >= 90", health.HitRate)
	}
}

func TestMemorySnapshot(t *testing.T) {
	snapshot := &MemorySnapshot{
		Timestamp: time.Now(),
		L1Size:    1000,
		L2Keys:    5000,
		L2Memory:  10 * 1024 * 1024,
	}
	
	if snapshot.L1Size != 1000 {
		t.Errorf("L1Size = %d, want 1000", snapshot.L1Size)
	}
}

func TestWarmupConfig(t *testing.T) {
	config := DefaultWarmupConfig
	
	if config.Policy != WarmupPolicyAdaptive {
		t.Errorf("Policy = %d, want %d", config.Policy, WarmupPolicyAdaptive)
	}
	
	if config.Concurrency != 5 {
		t.Errorf("Concurrency = %d, want 5", config.Concurrency)
	}
	
	if config.BatchSize != 100 {
		t.Errorf("BatchSize = %d, want 100", config.BatchSize)
	}
}

func TestCacheExpirationConfig(t *testing.T) {
	config := DefaultExpirationConfig
	
	if config.DefaultTTL != 10*time.Minute {
		t.Errorf("DefaultTTL = %v, want 10*time.Minute", config.DefaultTTL)
	}
	
	if config.Strategy != ExpirationSliding {
		t.Errorf("Strategy = %d, want %d", config.Strategy, ExpirationSliding)
	}
}

func TestCacheInvalidationConfig(t *testing.T) {
	config := DefaultInvalidationConfig
	
	if config.Mode != InvalidationImmediate {
		t.Errorf("Mode = %d, want %d", config.Mode, InvalidationImmediate)
	}
	
	if config.BatchSize != 100 {
		t.Errorf("BatchSize = %d, want 100", config.BatchSize)
	}
}

func TestGetCacheMonitoringCollector(t *testing.T) {
	collector := GetCacheMonitoringCollector()
	if collector == nil {
		t.Fatal("GetCacheMonitoringCollector should not return nil")
	}
	
	collector2 := GetCacheMonitoringCollector()
	if collector != collector2 {
		t.Error("Should return same instance")
	}
}

func TestGetExpirationManager(t *testing.T) {
	manager := GetExpirationManager()
	if manager == nil {
		t.Fatal("GetExpirationManager should not return nil")
	}
}

func TestGetInvalidationManager(t *testing.T) {
	manager := GetInvalidationManager()
	if manager == nil {
		t.Fatal("GetInvalidationManager should not return nil")
	}
}

func TestGetConsistencyManager(t *testing.T) {
	manager := GetConsistencyManager()
	if manager == nil {
		t.Fatal("GetConsistencyManager should not return nil")
	}
}

func TestGetCacheWarmupManager(t *testing.T) {
	manager := GetCacheWarmupManager()
	if manager == nil {
		t.Fatal("GetCacheWarmupManager should not return nil")
	}
}

func TestCacheKeyPrefixes(t *testing.T) {
	prefixes := []CacheKeyPrefix{
		PrefixCaptcha,
		PrefixSession,
		PrefixBlacklist,
		PrefixApplication,
		PrefixStats,
		PrefixRateLimit,
		PrefixBehavior,
		PrefixConfig,
		PrefixUser,
		PrefixToken,
		PrefixLock,
		PrefixWarmup,
		PrefixMetrics,
		PrefixVersion,
		PrefixTag,
		PrefixMeta,
		PrefixAnalytics,
		PrefixWhitelist,
		PrefixAlert,
	}
	
	expectedPrefixes := []string{
		"captcha",
		"session",
		"blacklist",
		"app",
		"stats",
		"ratelimit",
		"behavior",
		"config",
		"user",
		"token",
		"lock",
		"warmup",
		"metrics",
		"version",
		"tag",
		"meta",
		"analytics",
		"whitelist",
		"alert",
	}
	
	for i, prefix := range prefixes {
		if string(prefix) != expectedPrefixes[i] {
			t.Errorf("Prefix[%d] = %q, want %q", i, string(prefix), expectedPrefixes[i])
		}
	}
}

func TestCacheKeyNamespace(t *testing.T) {
	namespaces := []CacheKeyNamespace{
		NamespaceGlobal,
		NamespaceApp,
		NamespaceUser,
		NamespaceSession,
		NamespaceAPI,
	}
	
	expectedNamespaces := []string{
		"global",
		"app",
		"user",
		"session",
		"api",
	}
	
	for i, ns := range namespaces {
		if string(ns) != expectedNamespaces[i] {
			t.Errorf("Namespace[%d] = %q, want %q", i, string(ns), expectedNamespaces[i])
		}
	}
}

func TestLatencyHistogram(t *testing.T) {
	histogram := NewLatencyHistogram()
	
	histogram.Record(100 * time.Microsecond)
	histogram.Record(5 * time.Millisecond)
	histogram.Record(50 * time.Millisecond)
	
	p95 := histogram.Percentile(0.95)
	if p95 <= 0 {
		t.Error("P95 should be greater than 0")
	}
	
	dist := histogram.GetDistribution()
	if len(dist) == 0 {
		t.Error("Distribution should not be empty")
	}
}

func TestCacheMonitoringCollectorReset(t *testing.T) {
	collector := NewCacheMonitoringCollector(nil)
	
	collector.RecordHit()
	collector.RecordHit()
	collector.RecordMiss()
	
	collector.Reset()
	
	metrics := collector.GetDetailedMetrics()
	if metrics["hits"].(int64) != 0 {
		t.Errorf("hits after reset = %d, want 0", metrics["hits"])
	}
}

func TestExpirationManagerSignalRefresh(t *testing.T) {
	manager := NewCacheExpirationManager(nil)
	
	manager.SignalRefresh("test_key")
	
	select {
	case <-manager.GetRefreshChannel():
	default:
		t.Error("Should have refresh signal")
	}
}

func TestAdaptiveExpirationPolicyShouldRefresh(t *testing.T) {
	policy := NewAdaptiveExpirationPolicy(
		10*time.Minute,
		1*time.Minute,
		1*time.Hour,
	)
	
	for i := 0; i < 50; i++ {
		policy.RecordAccess("hot_key")
	}
	
	shouldRefresh := policy.ShouldRefresh("hot_key", 1*time.Minute)
	if !shouldRefresh {
		t.Error("Hot key with short TTL should be refreshed")
	}
	
	shouldRefreshCold := policy.ShouldRefresh("cold_key", 5*time.Minute)
	if !shouldRefreshCold {
		t.Error("Cold key with adequate TTL should still be refreshed based on TTL threshold")
	}
}
