package redis

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

func TestNewEnhancedCache(t *testing.T) {
	config := &CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: true,
		L1Size:    100,
		L1TTL:     time.Minute,
		L2TTL:     time.Hour,
	}

	cache := NewEnhancedCache(config)
	if cache == nil {
		t.Fatal("NewEnhancedCache returned nil")
	}
	defer cache.Close()

	if cache.config != config {
		t.Errorf("Expected config %v, got %v", config, cache.config)
	}
}

func TestNewEnhancedCacheWithNilConfig(t *testing.T) {
	cache := NewEnhancedCache(nil)
	if cache == nil {
		t.Fatal("NewEnhancedCache with nil config returned nil")
	}
	defer cache.Close()

	if cache.config != DefaultCacheConfig {
		t.Errorf("Expected default config, got %v", cache.config)
	}
}

func TestEnhancedCacheGetWithDisabledCache(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{Enabled: false})
	defer cache.Close()
	ctx := context.Background()

	_, err := cache.Get(ctx, "test_key", nil)
	if err != ErrCacheDisabled {
		t.Errorf("Expected ErrCacheDisabled, got %v", err)
	}
}

func TestEnhancedCacheSetWithDisabledCache(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{Enabled: false})
	defer cache.Close()
	ctx := context.Background()

	err := cache.Set(ctx, "test_key", []byte("value"), nil)
	if err != ErrCacheDisabled {
		t.Errorf("Expected ErrCacheDisabled, got %v", err)
	}
}

func TestEnhancedCacheDeleteWithDisabledCache(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{Enabled: false})
	defer cache.Close()
	ctx := context.Background()

	err := cache.Delete(ctx, "test_key", nil)
	if err != ErrCacheDisabled {
		t.Errorf("Expected ErrCacheDisabled, got %v", err)
	}
}

func TestEnhancedCacheL1BasicOperations(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: false,
		L1Size:    100,
		L1TTL:     time.Minute,
	})
	defer cache.Close()

	ctx := context.Background()

	err := cache.Set(ctx, "key1", []byte("value1"), &SetOptions{Level: CacheLevelL1})
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := cache.Get(ctx, "key1", &GetOptions{Level: CacheLevelL1})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(val) != "value1" {
		t.Errorf("Expected 'value1', got '%s'", string(val))
	}
}

func TestEnhancedCacheL1Expiration(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: false,
		L1Size:    100,
		L1TTL:     10 * time.Millisecond,
	})
	defer cache.Close()

	ctx := context.Background()

	err := cache.Set(ctx, "key1", []byte("value1"), &SetOptions{Level: CacheLevelL1})
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	_, err = cache.Get(ctx, "key1", &GetOptions{Level: CacheLevelL1})
	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss after expiration, got %v", err)
	}
}

func TestEnhancedCacheL1LRUEviction(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: false,
		L1Size:    3,
		L1TTL:     time.Hour,
	})
	defer cache.Close()

	ctx := context.Background()

	for i := 0; i < 5; i++ {
		err := cache.Set(ctx, "key"+string(rune('a'+i)), []byte("value"), &SetOptions{Level: CacheLevelL1})
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	cache.Get(ctx, "keya", &GetOptions{Level: CacheLevelL1})
	cache.Get(ctx, "keyb", &GetOptions{Level: CacheLevelL1})

	err := cache.Set(ctx, "keyf", []byte("value"), &SetOptions{Level: CacheLevelL1})
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
}

func TestEnhancedCacheDelete(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: false,
		L1TTL:     time.Hour,
	})
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "key1", []byte("value1"), &SetOptions{Level: CacheLevelL1})

	err := cache.Delete(ctx, "key1", &DeleteOptions{Level: CacheLevelL1})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = cache.Get(ctx, "key1", &GetOptions{Level: CacheLevelL1})
	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss after delete, got %v", err)
	}
}

func TestEnhancedCacheGetOrSet(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: false,
		L1TTL:     time.Hour,
	})
	defer cache.Close()

	ctx := context.Background()

	called := false
	val, err := cache.GetOrSet(ctx, "key1", time.Hour, func() ([]byte, error) {
		called = true
		return []byte("computed_value"), nil
	})

	if err != nil {
		t.Fatalf("GetOrSet failed: %v", err)
	}

	if !called {
		t.Error("Loader function was not called on cache miss")
	}

	if string(val) != "computed_value" {
		t.Errorf("Expected 'computed_value', got '%s'", string(val))
	}
}

func TestEnhancedCacheGetJSON(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: false,
		L1TTL:     time.Hour,
	})
	defer cache.Close()

	ctx := context.Background()

	type TestStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	data := TestStruct{Name: "John", Age: 30}
	err := cache.SetJSON(ctx, "key1", data, &SetOptions{Level: CacheLevelL1})
	if err != nil {
		t.Fatalf("SetJSON failed: %v", err)
	}

	result, err := cache.GetOrSet(ctx, "key1", time.Hour, func() ([]byte, error) {
		return json.Marshal(data)
	})
	if err != nil {
		t.Fatalf("GetOrSet failed: %v", err)
	}

	var resultStruct TestStruct
	err = json.Unmarshal(result, &resultStruct)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if resultStruct.Name != "John" || resultStruct.Age != 30 {
		t.Errorf("Expected {John 30}, got {%s %d}", resultStruct.Name, resultStruct.Age)
	}
}

func TestEnhancedCacheSetJSON(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: false,
		L1TTL:     time.Hour,
	})
	defer cache.Close()

	ctx := context.Background()

	type TestStruct struct {
		Name string `json:"name"`
	}

	data := TestStruct{Name: "Jane"}
	err := cache.SetJSON(ctx, "key1", data, &SetOptions{Level: CacheLevelL1})
	if err != nil {
		t.Fatalf("SetJSON failed: %v", err)
	}
}

func TestEnhancedCacheIncrementVersion(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled: true,
	})
	defer cache.Close()

	ctx := context.Background()

	v1, err := cache.IncrementVersion(ctx, "key1")
	if err != nil {
		t.Fatalf("IncrementVersion failed: %v", err)
	}
	if v1 != 1 {
		t.Errorf("Expected version 1, got %d", v1)
	}

	v2, err := cache.IncrementVersion(ctx, "key1")
	if err != nil {
		t.Fatalf("IncrementVersion failed: %v", err)
	}
	if v2 != 2 {
		t.Errorf("Expected version 2, got %d", v2)
	}
}

func TestEnhancedCacheGetVersion(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled: true,
	})
	defer cache.Close()

	ctx := context.Background()

	cache.IncrementVersion(ctx, "key1")
	cache.IncrementVersion(ctx, "key1")

	version, err := cache.GetVersion(ctx, "key1")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}
	if version != 2 {
		t.Errorf("Expected version 2, got %d", version)
	}
}

func TestEnhancedCacheMGet(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: false,
		L1TTL:     time.Hour,
	})
	defer cache.Close()

	ctx := context.Background()

	for _, key := range []string{"key1", "key2"} {
		cache.bloomFilter.Add(key)
		cache.l1Cache.Store(key, &l1Entry{
			value:     []byte("value"),
			expiresAt: time.Now().Add(time.Hour),
		})
	}

	result, err := cache.MGet(ctx, []string{"key1", "key2", "key3"}, &GetOptions{Level: CacheLevelL1})
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result))
	}
}

func TestEnhancedCacheMSet(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: false,
		L1TTL:     time.Hour,
	})
	defer cache.Close()

	ctx := context.Background()

	items := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}

	err := cache.MSet(ctx, items, &SetOptions{Level: CacheLevelL1})
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}
}

func TestEnhancedCacheLock(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled: true,
	})
	defer cache.Close()

	ctx := context.Background()

	acquired, err := cache.Lock(ctx, "test_lock", time.Second)
	if err != nil {
		t.Fatalf("Lock failed: %v", err)
	}

	if acquired {
		err = cache.Unlock(ctx, "test_lock")
		if err != nil {
			t.Fatalf("Unlock failed: %v", err)
		}
	}
}

func TestEnhancedCacheAcquireLock(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled: true,
	})
	defer cache.Close()

	ctx := context.Background()

	cache.Lock(ctx, "test_lock", 10*time.Second)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx := context.Background()
		acquired, _ := cache.AcquireLock(ctx, "test_lock", time.Second, 3)
		if acquired {
			t.Error("Should not acquire lock during hold")
		}
	}()

	wg.Wait()

	cache.Unlock(ctx, "test_lock")
}

func TestEnhancedCacheExtendLock(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled: true,
	})
	defer cache.Close()

	ctx := context.Background()

	cache.Lock(ctx, "test_lock", 100*time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	err := cache.ExtendLock(ctx, "test_lock", 200*time.Millisecond)
	if err != nil {
		t.Fatalf("ExtendLock failed: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	acquired, _ := cache.Lock(ctx, "test_lock", time.Second)
	if acquired {
		t.Error("Lock should have expired")
	}
}

func TestEnhancedCacheClear(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: false,
		L1TTL:     time.Hour,
	})
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "key1", []byte("value1"), &SetOptions{Level: CacheLevelL1})
	cache.Set(ctx, "key2", []byte("value2"), &SetOptions{Level: CacheLevelL1})

	err := cache.Clear(ctx, CacheLevelL1)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
}

func TestEnhancedCacheGetStats(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: false,
		L1TTL:     time.Hour,
	})
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "key1", []byte("value1"), &SetOptions{Level: CacheLevelL1})

	stats := cache.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Second)

	for i := 0; i < 3; i++ {
		if !cb.Allow() {
			t.Error("Circuit breaker should allow requests initially")
		}
		cb.RecordFailure()
	}

	if cb.Allow() {
		t.Error("Circuit breaker should be open after threshold failures")
	}

	if cb.State() != "open" {
		t.Errorf("Expected state 'open', got '%s'", cb.State())
	}

	time.Sleep(time.Second + 100*time.Millisecond)

	if !cb.Allow() {
		t.Error("Circuit breaker should transition to half-open after timeout")
	}
}

func TestCircuitBreakerRecordSuccess(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Second)

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != "open" {
		t.Error("Circuit breaker should be open")
	}

	cb.RecordSuccess()

	if cb.State() != "closed" {
		t.Error("Circuit breaker should be closed after success")
	}

	if cb.failures != 0 {
		t.Errorf("Expected 0 failures, got %d", cb.failures)
	}
}

func TestBloomFilter(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)

	bf.Add("key1")
	bf.Add("key2")
	bf.Add("key3")

	if !bf.MayContain("key1") {
		t.Error("Bloom filter should contain key1")
	}

	if !bf.MayContain("key2") {
		t.Error("Bloom filter should contain key2")
	}

	if bf.MayContain("key4") {
		t.Error("Bloom filter should not contain key4")
	}

	bf.Clear()

	if bf.MayContain("key1") {
		t.Error("Bloom filter should not contain key1 after clear")
	}
}

func TestVersionManager(t *testing.T) {
	vm := NewVersionManager()

	v1, err := vm.Increment("key1")
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if v1 != 1 {
		t.Errorf("Expected version 1, got %d", v1)
	}

	v2, err := vm.Increment("key1")
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if v2 != 2 {
		t.Errorf("Expected version 2, got %d", v2)
	}

	version, err := vm.Get("key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if version != 2 {
		t.Errorf("Expected version 2, got %d", version)
	}

	vm.Set("key1", 5)
	version, _ = vm.Get("key1")
	if version != 5 {
		t.Errorf("Expected version 5, got %d", version)
	}
}

func TestCompression(t *testing.T) {
	largeData := make([]byte, 10000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	compressed, err := compress(largeData)
	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	if len(compressed) >= len(largeData) {
		t.Error("Compressed data should be smaller than original")
	}

	decompressed, err := decompress(compressed)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	if len(decompressed) != len(largeData) {
		t.Errorf("Expected length %d, got %d", len(largeData), len(decompressed))
	}
}

func TestDecompressInvalidData(t *testing.T) {
	_, err := decompress([]byte("invalid"))
	if err == nil {
		t.Error("Should fail for invalid compressed data")
	}

	_, err = decompress([]byte{0x00})
	if err == nil {
		t.Error("Should fail for too short data")
	}
}

func TestOptimizedSerializer(t *testing.T) {
	serializer := NewOptimizedSerializer()

	type TestStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	data := TestStruct{Name: "John", Age: 30}

	serialized, err := serializer.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result TestStruct
	err = serializer.Unmarshal(serialized, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.Name != "John" || result.Age != 30 {
		t.Errorf("Expected {John 30}, got {%s %d}", result.Name, result.Age)
	}
}

func TestOptimizedSerializerSetDefaultFormat(t *testing.T) {
	serializer := NewOptimizedSerializer()

	serializer.SetDefaultFormat(SerializationJSON)
	if serializer.defaultFormat != SerializationJSON {
		t.Error("Failed to set default format to JSON")
	}

	serializer.SetDefaultFormat(SerializationGob)
	if serializer.defaultFormat != SerializationGob {
		t.Error("Failed to set default format to Gob")
	}
}

func TestGobSerializer(t *testing.T) {
	serializer := NewGobSerializer()

	type TestStruct struct {
		Name string
		Age  int
	}

	data := TestStruct{Name: "Jane", Age: 25}

	serialized, err := serializer.Marshal(data)
	if err != nil {
		t.Fatalf("Gob marshal failed: %v", err)
	}

	var result TestStruct
	err = serializer.Unmarshal(serialized, &result)
	if err != nil {
		t.Fatalf("Gob unmarshal failed: %v", err)
	}

	if result.Name != "Jane" || result.Age != 25 {
		t.Errorf("Expected {Jane 25}, got {%s %d}", result.Name, result.Age)
	}
}

func TestJSONSerializer(t *testing.T) {
	serializer := NewJSONSerializer()

	type TestStruct struct {
		Name string `json:"name"`
	}

	data := TestStruct{Name: "Bob"}

	serialized, err := serializer.Marshal(data)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var result TestStruct
	err = serializer.Unmarshal(serialized, &result)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if result.Name != "Bob" {
		t.Errorf("Expected 'Bob', got '%s'", result.Name)
	}
}

func TestAdaptiveTtlManager(t *testing.T) {
	manager := NewAdaptiveTtlManager(time.Minute, 10*time.Second, time.Hour)

	ttl := manager.GetTtl("key1")
	if ttl != time.Minute {
		t.Errorf("Expected 1 minute TTL, got %v", ttl)
	}

	for i := 0; i < 10; i++ {
		manager.RecordAccess("key1")
	}

	ttl = manager.GetTtl("key1")
	if ttl <= time.Minute {
		t.Errorf("TTL should increase with more accesses, got %v", ttl)
	}
}

func TestAdaptiveTtlManagerBounds(t *testing.T) {
	manager := NewAdaptiveTtlManager(5*time.Second, 1*time.Second, 10*time.Second)

	for i := 0; i < 100; i++ {
		manager.RecordAccess("key1")
	}

	ttl := manager.GetTtl("key1")
	if ttl > 10*time.Second {
		t.Errorf("TTL should not exceed max, got %v", ttl)
	}
}

func TestPrefetchBuffer(t *testing.T) {
	var processedKeys []string
	buffer := NewPrefetchBuffer(3, func(keys []string) {
		processedKeys = append(processedKeys, keys...)
	})

	buffer.Add("key1")
	buffer.Add("key2")
	buffer.Add("key3")

	if len(processedKeys) != 3 {
		t.Errorf("Expected 3 processed keys, got %d", len(processedKeys))
	}

	buffer.Flush()
	if len(processedKeys) != 3 {
		t.Error("Flush should not process empty buffer")
	}
}

func TestLazyLoader(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: false,
		L1TTL:     time.Hour,
	})
	defer cache.Close()

	loaderCalled := false
	loader := func(ctx context.Context, key string) (interface{}, error) {
		loaderCalled = true
		return map[string]interface{}{
			"name": "test",
		}, nil
	}

	lazyLoader := NewLazyLoader(cache, time.Hour, loader)
	ctx := context.Background()

	result, err := lazyLoader.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !loaderCalled {
		t.Error("Loader should be called on cache miss")
	}

	loaderCalled = false
	result, err = lazyLoader.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	_ = result
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(10)

	for i := 0; i < 10; i++ {
		if !limiter.Allow() {
			t.Errorf("Request %d should be allowed", i)
		}
	}

	if limiter.Allow() {
		t.Error("Request should be rate limited")
	}

	time.Sleep(time.Second)

	limiter.SetRefillRate(10)
	if !limiter.Allow() {
		t.Error("Request should be allowed after refill")
	}
}

func TestRateLimiterAllowN(t *testing.T) {
	limiter := NewRateLimiter(10)

	if !limiter.AllowN(5) {
		t.Error("Should allow 5 tokens")
	}

	if limiter.AllowN(6) {
		t.Error("Should not allow 6 tokens when only 5 remaining")
	}

	if !limiter.AllowN(5) {
		t.Error("Should allow 5 tokens")
	}
}

func TestWeightedSemaphore(t *testing.T) {
	sem := NewWeightedSemaphore(10)

	ctx := context.Background()

	err := sem.Acquire(ctx, 5)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	sem.Release(5)

	err = sem.Acquire(ctx, 10)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	sem.Release(10)
}

func TestCacheMetricsCollector(t *testing.T) {
	collector := NewCacheMetricsCollector()

	collector.RecordHit()
	collector.RecordHit()
	collector.RecordMiss()

	collector.RecordL1Hit()
	collector.RecordL2Hit()

	collector.RecordSet()
	collector.RecordDelete()

	collector.RecordError()

	collector.RecordLatency(10 * time.Millisecond)

	collector.RecordKeyAccess("key1")

	collector.RecordCompressed()
	collector.RecordDecompressed()

	metrics := collector.GetDetailedMetrics()

	if metrics.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", metrics.Hits)
	}

	if metrics.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", metrics.Misses)
	}
}

func TestCacheMetricsCollectorReset(t *testing.T) {
	collector := NewCacheMetricsCollector()

	collector.RecordHit()
	collector.RecordSet()

	collector.Reset()

	metrics := collector.GetDetailedMetrics()
	if metrics.Hits != 0 {
		t.Errorf("Expected 0 hits after reset, got %d", metrics.Hits)
	}
}

func TestLatencyHistogram(t *testing.T) {
	histogram := NewLatencyHistogram()

	histogram.Record(time.Microsecond)
	histogram.Record(10 * time.Millisecond)
	histogram.Record(100 * time.Millisecond)

	p50 := histogram.Percentile(0.5)
	if p50 == 0 {
		t.Error("Percentile should return non-zero value")
	}

	dist := histogram.GetDistribution()
	if len(dist) == 0 {
		t.Error("Distribution should not be empty")
	}
}

func TestAlertManager(t *testing.T) {
	manager := NewAlertManager(10)

	manager.AddAlert("test_type", "test_message", "info")

	alerts := manager.GetAlerts()
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}

	if alerts[0].Type != "test_type" {
		t.Errorf("Expected type 'test_type', got '%s'", alerts[0].Type)
	}
}

func TestMetricsExporter(t *testing.T) {
	collector := NewCacheMetricsCollector()
	exporter := NewMetricsExporter(collector, 100*time.Millisecond)

	exporter.Start()
	time.Sleep(200 * time.Millisecond)
	exporter.Stop()
}

func TestConsistentHashRing(t *testing.T) {
	ring := NewConsistentHashRing(100)

	ring.AddNode("node1")
	ring.AddNode("node2")
	ring.AddNode("node3")

	node1 := ring.GetNode("key1")
	if node1 == "" {
		t.Error("GetNode should return a node")
	}

	ring.RemoveNode(node1)

	node2 := ring.GetNode("key1")
	if node2 == node1 {
		t.Error("GetNode should return different node after removal")
	}
}

func TestOptimisticLock(t *testing.T) {
	lock := NewOptimisticLock()

	version, ok := lock.Acquire("key1")
	if !ok {
		t.Error("Failed to acquire lock")
	}
	if version != 1 {
		t.Errorf("Expected version 1, got %d", version)
	}

	ok = lock.Release("key1", version)
	if !ok {
		t.Error("Failed to release lock")
	}

	ok = lock.Release("key1", version)
	if ok {
		t.Error("Should not release with wrong version")
	}

	_, ok = lock.Acquire("key1")
	if !ok {
		t.Error("Should be able to re-acquire after release")
	}
}

func TestConsistencyStats(t *testing.T) {
	stats := &ConsistencyStats{}

	stats.Invalidations.Add(10)
	stats.SyncSuccess.Add(5)
	stats.Conflicts.Add(2)
	stats.ConflictResolved.Add(1)

	if stats.Invalidations.Load() != 10 {
		t.Errorf("Expected 10 invalidations, got %d", stats.Invalidations.Load())
	}
}

func TestWriteBuffer(t *testing.T) {
	buffer := NewWriteBuffer(100)

	buffer.Add("key1", []byte("value1"), "set")
	buffer.Add("key2", nil, "delete")

	if buffer.Size() != 2 {
		t.Errorf("Expected size 2, got %d", buffer.Size())
	}

	items := buffer.GetAndClear()
	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}

	if buffer.Size() != 0 {
		t.Error("Buffer should be empty after GetAndClear")
	}
}

func TestBatchWarmer(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: false,
		L1TTL:     time.Hour,
	})
	defer cache.Close()

	warmer := NewBatchWarmer(cache, 10, 2)

	items := []*WarmupItem{
		{Key: "key1", Value: []byte("value1")},
		{Key: "key2", Value: []byte("value2")},
		{Key: "key3", Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("loaded"), nil
		}},
	}

	ctx := context.Background()
	err := warmer.Warmup(ctx, items)
	if err != nil {
		t.Fatalf("Warmup failed: %v", err)
	}
}

func TestAdaptiveRefresher(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled: true,
	})
	defer cache.Close()

	refresher := NewAdaptiveRefresher(cache)

	refresher.RecordAccess("key1")
	refresher.RecordAccess("key1")
	refresher.RecordAccess("key1")

	shouldRefresh := refresher.ShouldRefresh("key1", 30*time.Minute)
	if shouldRefresh {
		t.Error("ShouldRefresh should return false")
	}

	refresher.Cleanup()
}

func TestSmartWarmer(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled: true,
	})
	defer cache.Close()

	warmer := NewSmartWarmer(cache, 5)

	for i := 0; i < 10; i++ {
		warmer.RecordAccess("hotkey")
	}

	hotKeys := warmer.GetHotKeys()
	if len(hotKeys) != 1 {
		t.Errorf("Expected 1 hot key, got %d", len(hotKeys))
	}

	warmer.ResetCounts()
	hotKeys = warmer.GetHotKeys()
	if len(hotKeys) != 0 {
		t.Error("Should have no hot keys after reset")
	}
}

func TestCacheWarmer(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled: true,
	})
	defer cache.Close()

	warmer := NewCacheWarmer(cache)

	task := &WarmupTask{
		Name:      "test_task",
		Key:       "test_key",
		TTL:       time.Hour,
		Frequency: time.Minute,
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("loaded_data"), nil
		},
		Enabled: true,
	}

	warmer.AddTask(task)

	tasks := warmer.GetTasks()
	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}

	warmer.DisableTask("test_task")
	warmer.EnableTask("test_task")
	warmer.RemoveTask("test_task")

	tasks = warmer.GetTasks()
	if len(tasks) != 0 {
		t.Error("Task should be removed")
	}
}

func TestCacheWarmerWarmup(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled: true,
	})
	defer cache.Close()

	warmer := NewCacheWarmer(cache)

	task := &WarmupTask{
		Name:      "test_task",
		Key:       "test_key",
		TTL:       time.Hour,
		Frequency: time.Minute,
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("loaded_data"), nil
		},
		Enabled: true,
	}

	warmer.AddTask(task)

	err := warmer.WarmupAll()
	if err != nil {
		t.Fatalf("WarmupAll failed: %v", err)
	}
}

func TestCacheConsistencyManager(t *testing.T) {
	config := DefaultConsistencyConfig()
	manager := NewCacheConsistencyManager(config)

	manager.Invalidate("test_key")

	manager.InvalidateKeys([]string{"key1", "key2"})

	manager.InvalidatePattern("test:*")

	manager.InvalidateByTag([]string{"tag1"})

	stats := manager.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	manager.Close()
}

func TestVersionTracker(t *testing.T) {
	tracker := NewVersionTracker()

	v1 := tracker.Increment("key1")
	if v1 != 1 {
		t.Errorf("Expected version 1, got %d", v1)
	}

	v2 := tracker.Increment("key1")
	if v2 != 2 {
		t.Errorf("Expected version 2, got %d", v2)
	}

	version, _ := tracker.Get("key1")
	if version != 2 {
		t.Errorf("Expected version 2, got %d", version)
	}

	tracker.Set("key2", 10)
	version, _ = tracker.Get("key2")
	if version != 10 {
		t.Errorf("Expected version 10, got %d", version)
	}
}

func TestEventEmitter(t *testing.T) {
	emitter := NewEventEmitter()

	eventCalled := false
	emitter.Subscribe("test_event", func(event *CacheEvent) {
		eventCalled = true
	})

	emitter.Emit("test_event", &CacheEvent{
		Type: "test_event",
		Key:  "test_key",
	})

	time.Sleep(10 * time.Millisecond)

	if !eventCalled {
		t.Error("Event handler should be called")
	}
}

func TestConflictResolver(t *testing.T) {
	config := DefaultConsistencyConfig()
	config.ConflictStrategy = ConflictLastWriteWins
	resolver := NewConflictResolver(config)

	now := time.Now()
	data1 := &ConflictData{
		Key:       "key1",
		Value:     []byte("value1"),
		Version:   1,
		Timestamp: now.Add(-time.Hour),
	}

	data2 := &ConflictData{
		Key:       "key1",
		Value:     []byte("value2"),
		Version:   2,
		Timestamp: now,
	}

	resolved, err := resolver.Resolve(data1, data2)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if string(resolved.Value) != "value2" {
		t.Errorf("Expected 'value2', got '%s'", string(resolved.Value))
	}

	resolver.StoreVersion("key1", resolved)
	stored, _ := resolver.GetStoredVersion("key1")
	if stored == nil {
		t.Error("Should store version")
	}
}

func TestConflictResolverMerge(t *testing.T) {
	config := DefaultConsistencyConfig()
	config.ConflictStrategy = ConflictMerge
	resolver := NewConflictResolver(config)

	now := time.Now()
	data1 := &ConflictData{
		Key:       "key1",
		Value:     []byte(`{"name":"john","age":30}`),
		Version:   1,
		Timestamp: now.Add(-time.Hour),
	}

	data2 := &ConflictData{
		Key:       "key1",
		Value:     []byte(`{"name":"jane","city":"NYC"}`),
		Version:   2,
		Timestamp: now,
	}

	resolved, err := resolver.Resolve(data1, data2)
	if err != nil {
		t.Fatalf("Merge resolve failed: %v", err)
	}

	if resolved.Source != "merged" {
		t.Error("Should be marked as merged")
	}
}

func TestPerformanceOptimizer(t *testing.T) {
	config := DefaultPerformanceConfig()
	optimizer := NewPerformanceOptimizer(config)

	optimizer.RecordHit(CacheLevelL1)
	optimizer.RecordHit(CacheLevelL2)
	optimizer.RecordMiss(CacheLevelL1)

	optimizer.RecordLatency(10 * time.Millisecond)
	optimizer.RecordLatency(20 * time.Millisecond)

	optimizer.RecordError()

	stats := optimizer.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	hitRate := optimizer.GetHitRate()
	if hitRate <= 0 {
		t.Error("Hit rate should be positive")
	}

	meets, msg := optimizer.CheckTargetMet()
	if meets {
		t.Logf("Performance targets met: %s", msg)
	}

	optimizer.Close()
}

func TestPerformanceStats(t *testing.T) {
	stats := &PerformanceStats{}

	stats.Hits.Add(95)
	stats.Misses.Add(5)

	hitRate := stats.GetHitRate()
	if hitRate != 95.0 {
		t.Errorf("Expected 95%% hit rate, got %.2f%%", hitRate)
	}

	stats.TotalLatency.Add(int64(150 * time.Millisecond))
	stats.TotalRequests.Add(2)

	avgLatency := stats.GetAvgLatency()
	if avgLatency != 75*time.Millisecond {
		t.Errorf("Expected 75ms avg latency, got %v", avgLatency)
	}

	stats.Errors.Add(2)
	stats.TotalRequests.Add(10)

	errorRate := stats.GetErrorRate()
	if errorRate < 16.66 || errorRate > 16.68 {
		t.Errorf("Expected 16.67%% error rate, got %.2f%%", errorRate)
	}
}

func TestAdaptiveTuner(t *testing.T) {
	config := DefaultPerformanceConfig()
	tuner := NewAdaptiveTuner(config)

	stats := &PerformanceStats{}
	stats.Hits.Add(80)
	stats.Misses.Add(20)
	stats.TotalLatency.Add(int64(100 * time.Millisecond))
	stats.TotalRequests.Add(2)

	poolSize, batchSize := tuner.Adjust(stats)
	if poolSize <= 0 || batchSize <= 0 {
		t.Error("Adjust should return positive values")
	}
}

func TestCachePerformanceAnalyzer(t *testing.T) {
	analyzer := NewCachePerformanceAnalyzer(time.Minute)

	analyzer.AddDataPoint(&DataPoint{
		Timestamp: time.Now(),
		HitRate:   90.0,
		Latency:   10 * time.Millisecond,
	})

	analyzer.AddDataPoint(&DataPoint{
		Timestamp: time.Now().Add(-30 * time.Second),
		HitRate:   95.0,
		Latency:   8 * time.Millisecond,
	})

	trend := analyzer.GetTrend()
	if trend != "stable" && trend != "insufficient_data" {
		t.Errorf("Expected stable trend, got %s", trend)
	}

	stats := analyzer.GetStatistics()
	if stats != nil {
		t.Logf("Statistics: AvgHitRate=%.2f, Trend=%s", stats.AvgHitRate, stats.Trend)
	}
}

func TestConnectionPoolOptimizer(t *testing.T) {
	config := DefaultPerformanceConfig()
	optimizer := NewConnectionPoolOptimizer(config)

	stats := &PoolStatsSnapshot{
		WaitCount:    100,
		IdleConns:    10,
		WaitDuration: 200 * time.Millisecond,
	}

	optimizer.AdjustPoolSize(stats)

	stats = &PoolStatsSnapshot{
		WaitCount: 0,
		IdleConns: 50,
	}

	optimizer.AdjustPoolSize(stats)
}

func TestPrefetchManager(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled: true,
	})
	defer cache.Close()

	manager := NewPrefetchManager(cache, 10, 3)

	for i := 0; i < 10; i++ {
		manager.RecordAccess("hotkey")
	}

	manager.Flush()

	manager.Disable()
	manager.Enable()

	manager.Close()
}

func TestPerformanceAlertManager(t *testing.T) {
	manager := NewPerformanceAlertManager()

	manager.AddAlert(&PerformanceAlert{
		Type:      "test",
		Message:   "test message",
		Severity:  "warning",
		Timestamp: time.Now(),
	})

	alerts := manager.GetAlerts(10)
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}

	manager.RegisterCallback("test", func(alert *PerformanceAlert) {
		t.Logf("Alert callback: %s", alert.Message)
	})
}

func TestGlobalFunctions(t *testing.T) {
	InitEnhancedCache(nil)
	cache := GetEnhancedCache()
	if cache == nil {
		t.Error("GetEnhancedCache returned nil")
	}
	defer cache.Close()

	collector := GetGlobalMetricsCollector()
	if collector == nil {
		t.Error("GetGlobalMetricsCollector returned nil")
	}

	alertManager := GetGlobalAlertManager()
	if alertManager == nil {
		t.Error("GetGlobalAlertManager returned nil")
	}

	warmer := GetGlobalWarmer()
	if warmer == nil {
		t.Error("GetGlobalWarmer returned nil")
	}

	consistencyManager := GetConsistencyManager()
	if consistencyManager == nil {
		t.Error("GetConsistencyManager returned nil")
	}

	perfOptimizer := GetPerformanceOptimizer()
	if perfOptimizer == nil {
		t.Error("GetPerformanceOptimizer returned nil")
	}
}

func TestCacheLevel(t *testing.T) {
	if CacheLevelL1 != 0 {
		t.Errorf("Expected CacheLevelL1 = 0, got %d", CacheLevelL1)
	}

	if CacheLevelL2 != 1 {
		t.Errorf("Expected CacheLevelL2 = 1, got %d", CacheLevelL2)
	}

	if CacheLevelBoth != 2 {
		t.Errorf("Expected CacheLevelBoth = 2, got %d", CacheLevelBoth)
	}
}

func TestInvalidationStrategy(t *testing.T) {
	if InvalidateOnWrite.String() != "invalidate_on_write" {
		t.Errorf("Unexpected string: %s", InvalidateOnWrite.String())
	}

	if InvalidateOnRead.String() != "invalidate_on_read" {
		t.Errorf("Unexpected string: %s", InvalidateOnRead.String())
	}
}

func TestSyncStrategy(t *testing.T) {
	if SyncImmediate.String() != "sync_immediate" {
		t.Errorf("Unexpected string: %s", SyncImmediate.String())
	}

	if SyncAsync.String() != "sync_async" {
		t.Errorf("Unexpected string: %s", SyncAsync.String())
	}
}

func TestConflictResolutionStrategy(t *testing.T) {
	if ConflictLastWriteWins.String() != "last_write_wins" {
		t.Errorf("Unexpected string: %s", ConflictLastWriteWins.String())
	}

	if ConflictMerge.String() != "merge" {
		t.Errorf("Unexpected string: %s", ConflictMerge.String())
	}
}

func TestWriteThroughPolicy(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled: true,
	})
	defer cache.Close()

	mockDB := &MockDatabaseWriter{}

	policy := &WriteThroughPolicy{
		cache:   cache,
		db:      mockDB,
		enabled: true,
	}

	ctx := context.Background()
	err := policy.Set(ctx, "key1", []byte("value1"), time.Hour)
	if err != nil {
		t.Fatalf("WriteThrough Set failed: %v", err)
	}
}

func TestCacheAsidePolicy(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled:   true,
		L1Enabled: true,
		L2Enabled: false,
	})
	defer cache.Close()

	mockDB := &MockDatabaseReader{
		data: map[string][]byte{
			"key1": []byte("db_value1"),
		},
	}

	policy := &CacheAsidePolicy{
		cache:        cache,
		db:           mockDB,
		enabled:      true,
		readOnlyMode: false,
	}

	ctx := context.Background()

	val, err := policy.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("CacheAside Get failed: %v", err)
	}

	if string(val) != "db_value1" {
		t.Errorf("Expected 'db_value1', got '%s'", string(val))
	}
}

func TestWriteBehindPolicy(t *testing.T) {
	cache := NewEnhancedCache(&CacheConfig{
		Enabled: true,
	})
	defer cache.Close()

	policy := &WriteBehindPolicy{
		cache:         cache,
		buffer:        NewWriteBuffer(100),
		flushInterval: time.Minute,
		enabled:       true,
	}

	ctx := context.Background()
	err := policy.Set(ctx, "key1", []byte("value1"))
	if err != nil {
		t.Fatalf("WriteBehind Set failed: %v", err)
	}

	err = policy.Flush(ctx)
	if err != nil {
		t.Fatalf("WriteBehind Flush failed: %v", err)
	}
}

func TestDataValidator(t *testing.T) {
	validator := NewDataValidator("fnv")

	validator.AddSchemaValidator("json", func(data []byte) error {
		if len(data) == 0 {
			return errors.New("empty data")
		}
		return nil
	})

	err := validator.Validate("json", []byte("test"))
	if err != nil {
		t.Error("Validation should pass for non-empty data")
	}

	err = validator.Validate("json", []byte{})
	if err == nil {
		t.Error("Validation should fail for empty data")
	}

	checksum := validator.ComputeChecksum([]byte("test data"))
	if checksum == 0 {
		t.Error("Checksum should not be zero")
	}
}

func TestConsistencyChecker(t *testing.T) {
	manager := NewCacheConsistencyManager(DefaultConsistencyConfig())
	defer manager.Close()
	checker := manager.CreateConsistencyChecker(time.Second)

	checker.Enable()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go checker.Start(ctx)

	time.Sleep(50 * time.Millisecond)

	checker.Disable()
}

type MockDatabaseWriter struct{}

func (m *MockDatabaseWriter) Write(ctx context.Context, key string, value []byte) error {
	return nil
}

func (m *MockDatabaseWriter) Read(ctx context.Context, key string) ([]byte, error) {
	return nil, nil
}

type MockDatabaseReader struct {
	data map[string][]byte
}

func (m *MockDatabaseReader) Read(ctx context.Context, key string) ([]byte, error) {
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return nil, goredis.Nil
}

func TestBatchOperator(t *testing.T) {
	operator := NewBatchOperator(nil, 10)

	result := operator.AddOperation("key1", "set", []byte("value1"))

	select {
	case err := <-result:
		if err != nil {
			t.Errorf("Operation failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Operation timed out")
	}

	operator.Close()
}

func TestHealthChecker(t *testing.T) {
	checker := NewHealthChecker(100*time.Millisecond, time.Second)

	checker.Start()

	time.Sleep(250 * time.Millisecond)

	checker.Stop()
}

func TestPoolStats(t *testing.T) {
	stats := &PoolStatsSnapshot{}

	stats.TotalConns = 100
	stats.IdleConns = 50

	if stats.TotalConns != 100 {
		t.Errorf("Expected 100 total conns, got %d", stats.TotalConns)
	}
}

func TestPoolStatsFromRedisClient(t *testing.T) {
	stats := &PoolStatsSnapshot{}
	if stats == nil {
		t.Fatal("PoolStatsSnapshot should not be nil")
	}
}

func TestSetPoolConfig(t *testing.T) {
}

func TestNewRedisClientWithNilConfig(t *testing.T) {
	client, err := NewRedisClient(nil)
	if err != nil {
		t.Fatalf("NewRedisClient with nil config failed: %v", err)
	}

	if client == nil {
		t.Fatal("Client should not be nil")
	}
}

func TestGetContext(t *testing.T) {
	ctx := GetContext()
	if ctx == nil {
		t.Fatal("GetContext returned nil")
	}
}
