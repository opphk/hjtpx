package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewCacheOptimizer(t *testing.T) {
	optimizer := NewCacheOptimizer(nil)

	if optimizer == nil {
		t.Fatal("NewCacheOptimizer returned nil")
	}

	if optimizer.config == nil {
		t.Fatal("config is nil")
	}

	if optimizer.keyManager == nil {
		t.Fatal("keyManager is nil")
	}

	if optimizer.expiration == nil {
		t.Fatal("expiration is nil")
	}

	if optimizer.preheat == nil {
		t.Fatal("preheat is nil")
	}

	if optimizer.monitor == nil {
		t.Fatal("monitor is nil")
	}
}

func TestNewCacheOptimizerWithConfig(t *testing.T) {
	config := &OptimizerConfig{
		EnableKeyOptimization:   true,
		EnableExpirationOptimize: true,
		EnablePreheat:          true,
		EnableMonitoring:       true,
		DefaultTTL:             15 * time.Minute,
		MinTTL:                2 * time.Minute,
		MaxTTL:                2 * time.Hour,
		PreheatBatchSize:       100,
		PreheatConcurrency:     20,
		MonitorInterval:        1 * time.Minute,
		HotKeyThreshold:        200,
		MaxMemoryPercent:       0.9,
	}

	optimizer := NewCacheOptimizer(config)

	if optimizer.config.DefaultTTL != 15*time.Minute {
		t.Errorf("Expected DefaultTTL 15m, got %v", optimizer.config.DefaultTTL)
	}

	if optimizer.config.MinTTL != 2*time.Minute {
		t.Errorf("Expected MinTTL 2m, got %v", optimizer.config.MinTTL)
	}

	if optimizer.config.MaxTTL != 2*time.Hour {
		t.Errorf("Expected MaxTTL 2h, got %v", optimizer.config.MaxTTL)
	}
}

func TestOptimizedKeyManager_RegisterPrefix(t *testing.T) {
	okm := NewOptimizedKeyManager()

	prefix := &KeyPrefix{
		Name:        "test",
		Separator:   ":",
		Version:     1,
		Description: "Test prefix",
	}

	okm.RegisterPrefix(prefix)

	okm.mu.RLock()
	defer okm.mu.RUnlock()

	if _, exists := okm.prefixes["test"]; !exists {
		t.Error("Prefix not registered")
	}
}

func TestOptimizedKeyManager_BuildKey(t *testing.T) {
	okm := NewOptimizedKeyManager()

	okm.RegisterPrefix(&KeyPrefix{
		Name:      "captcha",
		Separator: ":",
		Version:   1,
	})

	key := okm.BuildKey("captcha", "session123", "verify")

	expected := "captcha:session123:verify"
	if key != expected {
		t.Errorf("Expected key %s, got %s", expected, key)
	}
}

func TestOptimizedKeyManager_BuildCaptchaKey(t *testing.T) {
	okm := NewOptimizedKeyManager()

	okm.RegisterPrefix(&KeyPrefix{
		Name:      "captcha",
		Separator: ":",
		Version:   1,
	})

	key := okm.BuildCaptchaKey("captcha-123")

	if key == "" {
		t.Error("BuildCaptchaKey returned empty string")
	}

	if !contains(key, "captcha") {
		t.Error("Key does not contain 'captcha'")
	}
}

func TestOptimizedKeyManager_BuildSessionKey(t *testing.T) {
	okm := NewOptimizedKeyManager()

	key := okm.BuildSessionKey("session-456")

	if key == "" {
		t.Error("BuildSessionKey returned empty string")
	}

	if !contains(key, "session") {
		t.Error("Key does not contain 'session'")
	}
}

func TestOptimizedKeyManager_BuildPattern(t *testing.T) {
	okm := NewOptimizedKeyManager()

	okm.RegisterPrefix(&KeyPrefix{
		Name:      "captcha",
		Separator: ":",
		Version:   1,
	})

	pattern := okm.BuildPattern("captcha")

	if pattern == "" {
		t.Error("BuildPattern returned empty string")
	}

	if !contains(pattern, "*") {
		t.Error("Pattern does not contain wildcard")
	}
}

func TestOptimizedKeyManager_RecordAccess(t *testing.T) {
	okm := NewOptimizedKeyManager()

	okm.RecordAccess("key1")
	okm.RecordAccess("key1")
	okm.RecordAccess("key2")

	stats := okm.GetKeyStats("key1")
	if stats == nil {
		t.Fatal("GetKeyStats returned nil")
	}

	if atomic.LoadInt64(&stats.AccessCount) != 2 {
		t.Errorf("Expected access count 2, got %d", stats.AccessCount)
	}

	stats2 := okm.GetKeyStats("key2")
	if atomic.LoadInt64(&stats2.AccessCount) != 1 {
		t.Errorf("Expected access count 1, got %d", stats2.AccessCount)
	}
}

func TestOptimizedKeyManager_RecordHit(t *testing.T) {
	okm := NewOptimizedKeyManager()

	okm.RecordHit("key1")
	okm.RecordHit("key1")
	okm.RecordMiss("key1")

	stats := okm.GetKeyStats("key1")
	if stats == nil {
		t.Fatal("GetKeyStats returned nil")
	}

	if atomic.LoadInt64(&stats.HitCount) != 2 {
		t.Errorf("Expected hit count 2, got %d", stats.HitCount)
	}

	if atomic.LoadInt64(&stats.MissCount) != 1 {
		t.Errorf("Expected miss count 1, got %d", stats.MissCount)
	}
}

func TestOptimizedKeyManager_GetHotKeys(t *testing.T) {
	okm := NewOptimizedKeyManager()

	for i := 0; i < 5; i++ {
		okm.RecordAccess("hotkey")
	}

	for i := 0; i < 2; i++ {
		okm.RecordAccess("warmkey")
	}

	hotKeys := okm.GetHotKeys(10)

	if len(hotKeys) == 0 {
		t.Error("GetHotKeys returned empty slice")
	}

	if hotKeys[0].AccessCount < hotKeys[1].AccessCount {
		t.Error("Hot keys not sorted by access count")
	}
}

func TestOptimizedKeyManager_HashKey(t *testing.T) {
	okm := NewOptimizedKeyManager()

	key1 := "test-key"
	key2 := "test-key"
	key3 := "different-key"

	hash1 := okm.HashKey(key1)
	hash2 := okm.HashKey(key2)
	hash3 := okm.HashKey(key3)

	if hash1 != hash2 {
		t.Error("Same keys should have same hash")
	}

	if hash1 == hash3 {
		t.Error("Different keys should have different hashes")
	}

	if hash1 == "" {
		t.Error("Hash should not be empty")
	}
}

func TestExpirationOptimizer_RegisterPolicy(t *testing.T) {
	eo := NewExpirationOptimizer(nil)

	policy := &ExpirationPolicy{
		Name:         "test",
		BaseTTL:      10 * time.Minute,
		MinTTL:       1 * time.Minute,
		MaxTTL:       30 * time.Minute,
		Strategy:     StrategyAdaptive,
		RefreshRatio: 0.8,
	}

	eo.RegisterPolicy(policy)

	eo.mu.RLock()
	defer eo.mu.RUnlock()

	if _, exists := eo.policies["test"]; !exists {
		t.Error("Policy not registered")
	}
}

func TestExpirationOptimizer_CalculateTTL_Fixed(t *testing.T) {
	eo := NewExpirationOptimizer(nil)

	policy := &ExpirationPolicy{
		Name:     "fixed",
		BaseTTL:  10 * time.Minute,
		Strategy: StrategyFixed,
	}

	eo.RegisterPolicy(policy)

	ttl := eo.CalculateTTL("key1", "fixed")

	if ttl != 10*time.Minute {
		t.Errorf("Expected TTL 10m, got %v", ttl)
	}
}

func TestExpirationOptimizer_CalculateTTL_Sliding(t *testing.T) {
	eo := NewExpirationOptimizer(nil)

	policy := &ExpirationPolicy{
		Name:          "sliding",
		BaseTTL:       10 * time.Minute,
		Strategy:      StrategySliding,
		SlidingWindow: 2 * time.Minute,
		MaxTTL:        30 * time.Minute,
	}

	eo.RegisterPolicy(policy)

	ttl := eo.CalculateTTL("key1", "sliding")

	expected := 12 * time.Minute
	if ttl != expected {
		t.Errorf("Expected TTL %v, got %v", expected, ttl)
	}
}

func TestExpirationOptimizer_CalculateTTL_Adaptive(t *testing.T) {
	eo := NewExpirationOptimizer(nil)

	policy := &ExpirationPolicy{
		Name:     "adaptive",
		BaseTTL:  10 * time.Minute,
		Strategy: StrategyAdaptive,
		MinTTL:   5 * time.Minute,
		MaxTTL:   30 * time.Minute,
	}

	eo.RegisterPolicy(policy)

	for i := 0; i < 50; i++ {
		eo.RecordAccess("key1", "frequent")
	}

	ttl := eo.CalculateTTL("key1", "adaptive")

	if ttl < 10*time.Minute {
		t.Errorf("Adaptive TTL should be >= base TTL, got %v", ttl)
	}
}

func TestExpirationOptimizer_ShouldRefresh(t *testing.T) {
	eo := NewExpirationOptimizer(nil)

	policy := &ExpirationPolicy{
		Name:         "test",
		BaseTTL:      10 * time.Minute,
		RefreshRatio: 0.8,
	}

	eo.RegisterPolicy(policy)

	for i := 0; i < 10; i++ {
		eo.RecordAccess("key1", "frequent")
	}

	shouldRefresh := eo.ShouldRefresh("key1", 1*time.Minute)

	if !shouldRefresh {
		t.Error("ShouldRefresh should return true for low TTL")
	}

	shouldRefresh = eo.ShouldRefresh("key1", 9*time.Minute)
	if shouldRefresh {
		t.Error("ShouldRefresh should return false for high TTL")
	}
}

func TestPreheatManager_RegisterProfile(t *testing.T) {
	pm := NewPreheatManager(nil)

	profile := &PreheatProfile{
		Name:        "test",
		Priority:    5,
		BatchSize:   50,
		Concurrency: 10,
		Timeout:     30 * time.Second,
		Enabled:     true,
	}

	pm.RegisterProfile(profile)

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if _, exists := pm.profiles["test"]; !exists {
		t.Error("Profile not registered")
	}
}

func TestPreheatManager_RecordAccess(t *testing.T) {
	pm := NewPreheatManager(nil)

	pm.RecordAccess("key1")
	pm.RecordAccess("key1")
	pm.RecordAccess("key2")

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	log1 := pm.accessLog["key1"]
	if log1 == nil {
		t.Fatal("accessLog for key1 is nil")
	}

	if atomic.LoadInt64(&log1.Count) != 2 {
		t.Errorf("Expected count 2, got %d", log1.Count)
	}

	log2 := pm.accessLog["key2"]
	if atomic.LoadInt64(&log2.Count) != 1 {
		t.Errorf("Expected count 1, got %d", log2.Count)
	}
}

func TestPreheatManager_PredictHotKeys(t *testing.T) {
	pm := NewPreheatManager(nil)

	for i := 0; i < 100; i++ {
		pm.RecordAccess("hotkey")
	}

	for i := 0; i < 10; i++ {
		pm.RecordAccess("warmkey")
	}

	hotKeys := pm.PredictHotKeys(5)

	if len(hotKeys) == 0 {
		t.Error("PredictHotKeys returned empty slice")
	}

	if hotKeys[0] != "hotkey" {
		t.Errorf("Expected first hot key to be 'hotkey', got %s", hotKeys[0])
	}
}

func TestPreheatExecutor_Execute(t *testing.T) {
	config := &OptimizerConfig{
		PreheatBatchSize:   10,
		PreheatConcurrency: 5,
	}

	executor := NewPreheatExecutor(config)

	profile := &PreheatProfile{
		Name:        "test",
		Keys:        []string{"key1", "key2", "key3"},
		BatchSize:   5,
		Concurrency: 2,
		Timeout:     5 * time.Second,
		DataLoader: func(ctx context.Context, key string) (interface{}, error) {
			return map[string]string{"key": key, "value": "test"}, nil
		},
	}

	executor.Execute(profile)

	if profile.WarmmedCount == 0 && profile.FailedCount == 0 {
		t.Error("Execute did not process any keys")
	}
}

func TestCacheMonitor_RecordHit(t *testing.T) {
	cm := NewCacheMonitor(nil)

	cm.RecordHit()
	cm.RecordHit()
	cm.RecordMiss()

	metrics := cm.GetMetrics()

	if atomic.LoadInt64(&metrics.TotalHits) != 2 {
		t.Errorf("Expected 2 hits, got %d", metrics.TotalHits)
	}

	if atomic.LoadInt64(&metrics.TotalMisses) != 1 {
		t.Errorf("Expected 1 miss, got %d", metrics.TotalMisses)
	}
}

func TestCacheMonitor_RecordLatency(t *testing.T) {
	cm := NewCacheMonitor(nil)

	latencies := []time.Duration{
		1 * time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
	}

	for _, lat := range latencies {
		cm.RecordLatency(lat)
	}

	metrics := cm.GetMetrics()

	if metrics.AvgLatency == 0 {
		t.Error("Average latency should not be 0")
	}

	if metrics.P50Latency == 0 {
		t.Error("P50 latency should not be 0")
	}

	if metrics.P95Latency == 0 {
		t.Error("P95 latency should not be 0")
	}
}

func TestCacheMonitor_RecordAccess(t *testing.T) {
	cm := NewCacheMonitor(nil)

	cm.RecordAccess("key1", 1*time.Millisecond, true)
	cm.RecordAccess("key1", 2*time.Millisecond, true)
	cm.RecordAccess("key2", 3*time.Millisecond, false)

	metrics := cm.GetMetrics()

	if atomic.LoadInt64(&metrics.TotalHits) != 2 {
		t.Errorf("Expected 2 hits, got %d", metrics.TotalHits)
	}

	if atomic.LoadInt64(&metrics.TotalMisses) != 1 {
		t.Errorf("Expected 1 miss, got %d", metrics.TotalMisses)
	}

	hotKeys := cm.GetHotKeys(10)
	if len(hotKeys) == 0 {
		t.Error("GetHotKeys returned empty slice")
	}
}

func TestCacheMonitor_GetAlerts(t *testing.T) {
	cm := NewCacheMonitor(nil)

	cm.mu.Lock()
	for i := 0; i < 5; i++ {
		cm.alerts = append(cm.alerts, &CacheAlert{
			ID:        fmt.Sprintf("alert-%d", i),
			Type:      AlertLowHitRate,
			Severity:  SeverityWarning,
			Message:   fmt.Sprintf("Test alert %d", i),
			Timestamp: time.Now(),
		})
	}
	cm.mu.Unlock()

	alerts := cm.GetAlerts("", 3)

	if len(alerts) != 3 {
		t.Errorf("Expected 3 alerts, got %d", len(alerts))
	}
}

func TestHotKeyTracker_Record(t *testing.T) {
	hkt := NewHotKeyTracker(100, 30*time.Minute)

	hkt.Record("key1", 1*time.Millisecond)
	hkt.Record("key1", 2*time.Millisecond)
	hkt.Record("key2", 3*time.Millisecond)

	topKeys := hkt.GetTopKeys(10)

	if len(topKeys) == 0 {
		t.Error("GetTopKeys returned empty slice")
	}

	if topKeys[0].Count < topKeys[1].Count {
		t.Error("Keys not sorted by count")
	}
}

func TestLatencyTracker_Record(t *testing.T) {
	lt := NewLatencyTracker(1000)

	latencies := []time.Duration{
		100 * time.Microsecond,
		500 * time.Microsecond,
		1 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
	}

	for _, lat := range latencies {
		lt.Record(lat)
	}

	avg := lt.GetAverage()
	if avg == 0 {
		t.Error("Average should not be 0")
	}

	p50 := lt.GetPercentile(0.50)
	if p50 == 0 {
		t.Error("P50 should not be 0")
	}

	p95 := lt.GetPercentile(0.95)
	if p95 == 0 {
		t.Error("P95 should not be 0")
	}

	p99 := lt.GetPercentile(0.99)
	if p99 == 0 {
		t.Error("P99 should not be 0")
	}
}

func TestHealthChecker_RegisterCheck(t *testing.T) {
	hc := NewHealthChecker()

	hc.RegisterCheck("redis")

	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if _, exists := hc.checks["redis"]; !exists {
		t.Error("Check not registered")
	}
}

func TestHealthChecker_RunCheck(t *testing.T) {
	hc := NewHealthChecker()

	hc.RegisterCheck("redis")

	checkFunc := func() error {
		return nil
	}

	hc.RunCheck("redis", checkFunc)

	status := hc.GetStatus()

	if status.Overall != "healthy" {
		t.Errorf("Expected status 'healthy', got %s", status.Overall)
	}

	if status.Score != 100 {
		t.Errorf("Expected score 100, got %f", status.Score)
	}
}

func TestHealthChecker_RunCheckWithError(t *testing.T) {
	hc := NewHealthChecker()

	hc.RegisterCheck("redis")

	checkFunc := func() error {
		return fmt.Errorf("connection failed")
	}

	hc.RunCheck("redis", checkFunc)

	status := hc.GetStatus()

	if status.Overall != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got %s", status.Overall)
	}

	if status.Score != 0 {
		t.Errorf("Expected score 0, got %f", status.Score)
	}
}

func TestCacheOptimizer_StartStop(t *testing.T) {
	optimizer := NewCacheOptimizer(nil)

	optimizer.Start()

	if !optimizer.started {
		t.Error("Optimizer should be started")
	}

	optimizer.Stop()

	optimizer.mu.RLock()
	started := optimizer.started
	optimizer.mu.RUnlock()

	if started {
		t.Error("Optimizer should be stopped")
	}
}

func TestCacheOptimizer_RecordCacheAccess(t *testing.T) {
	optimizer := NewCacheOptimizer(nil)

	optimizer.RecordCacheAccess("key1", 1*time.Millisecond, true)
	optimizer.RecordCacheAccess("key1", 2*time.Millisecond, true)
	optimizer.RecordCacheAccess("key2", 3*time.Millisecond, false)

	stats := optimizer.GetKeyManager().GetKeyStats("key1")
	if stats == nil {
		t.Fatal("Key stats for key1 is nil")
	}

	if atomic.LoadInt64(&stats.AccessCount) != 2 {
		t.Errorf("Expected access count 2, got %d", stats.AccessCount)
	}

	metrics := optimizer.GetMonitor().GetMetrics()
	if atomic.LoadInt64(&metrics.TotalHits) != 2 {
		t.Errorf("Expected 2 hits, got %d", metrics.TotalHits)
	}
}

func TestCacheOptimizer_GetCacheStats(t *testing.T) {
	optimizer := NewCacheOptimizer(nil)

	optimizer.RecordCacheAccess("key1", 1*time.Millisecond, true)

	stats := optimizer.GetCacheStats()

	if stats == nil {
		t.Fatal("GetCacheStats returned nil")
	}

	if _, exists := stats["metrics"]; !exists {
		t.Error("Stats should contain 'metrics'")
	}

	if _, exists := stats["hot_keys"]; !exists {
		t.Error("Stats should contain 'hot_keys'")
	}

	if _, exists := stats["alerts"]; !exists {
		t.Error("Stats should contain 'alerts'")
	}

	if _, exists := stats["health"]; !exists {
		t.Error("Stats should contain 'health'")
	}
}

func TestCacheOptimizer_OptimizeKey(t *testing.T) {
	optimizer := NewCacheOptimizer(nil)

	key := "test-key"
	optimized := optimizer.OptimizeKey(key)

	if optimized == "" {
		t.Error("OptimizeKey returned empty string")
	}

	if optimized == key {
		t.Error("OptimizeKey should return a hashed key")
	}
}

func TestCacheOptimizer_GetOptimalTTL(t *testing.T) {
	optimizer := NewCacheOptimizer(nil)

	optimizer.GetExpirationOptimizer().RegisterPolicy(&ExpirationPolicy{
		Name:     "frequent",
		BaseTTL:  20 * time.Minute,
		Strategy: StrategyAdaptive,
	})

	for i := 0; i < 100; i++ {
		optimizer.GetExpirationOptimizer().RecordAccess("hotkey", "frequent")
	}

	ttl := optimizer.GetOptimalTTL("hotkey")

	if ttl == 0 {
		t.Error("GetOptimalTTL returned 0")
	}
}

func TestGlobalCacheOptimizer(t *testing.T) {
	opt1 := GetGlobalCacheOptimizer()
	opt2 := GetGlobalCacheOptimizer()

	if opt1 != opt2 {
		t.Error("GetGlobalCacheOptimizer should return the same instance")
	}
}

func TestCacheOptimizer_ConcurrentAccess(t *testing.T) {
	optimizer := NewCacheOptimizer(nil)

	var wg sync.WaitGroup
	concurrency := 100

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id%10)
			optimizer.RecordCacheAccess(key, time.Duration(id)*time.Millisecond, id%2 == 0)
		}(i)
	}

	wg.Wait()

	metrics := optimizer.GetMonitor().GetMetrics()
	total := atomic.LoadInt64(&metrics.TotalHits) + atomic.LoadInt64(&metrics.TotalMisses)

	if total != int64(concurrency) {
		t.Errorf("Expected %d total accesses, got %d", concurrency, total)
	}
}

func TestCacheOptimizer_LargeScaleAccess(t *testing.T) {
	optimizer := NewCacheOptimizer(nil)

	keyCount := 1000
	accessCount := 100

	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("key-%d", i)
		for j := 0; j < accessCount; j++ {
			optimizer.RecordCacheAccess(key, time.Duration(j)*time.Millisecond, j%2 == 0)
		}
	}

	stats := optimizer.GetCacheStats()
	if stats == nil {
		t.Fatal("GetCacheStats returned nil")
	}

	metrics := optimizer.GetMonitor().GetMetrics()
	total := atomic.LoadInt64(&metrics.TotalHits) + atomic.LoadInt64(&metrics.TotalMisses)

	expectedTotal := int64(keyCount * accessCount)
	if total != expectedTotal {
		t.Errorf("Expected %d total accesses, got %d", expectedTotal, total)
	}

	hotKeys := optimizer.GetMonitor().GetHotKeys(10)
	if len(hotKeys) == 0 {
		t.Error("Should have hot keys after large scale access")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
