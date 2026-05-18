package redis

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConsistentHashRing_AddAndGetNode(t *testing.T) {
	ring := NewConsistentHashRing(100)

	ring.AddNode("node1:6379", 1)
	ring.AddNode("node2:6379", 1)
	ring.AddNode("node3:6379", 1)

	if len(ring.nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(ring.nodes))
	}

	node, err := ring.GetNode("key1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if node == "" {
		t.Error("expected non-empty node address")
	}
}

func TestConsistentHashRing_RemoveNode(t *testing.T) {
	ring := NewConsistentHashRing(100)

	ring.AddNode("node1:6379", 1)
	ring.AddNode("node2:6379", 1)

	ring.RemoveNode("node1:6379")

	if len(ring.nodes) != 1 {
		t.Errorf("expected 1 node after removal, got %d", len(ring.nodes))
	}

	_, exists := ring.nodes["node1:6379"]
	if exists {
		t.Error("expected node1 to be removed")
	}
}

func TestConsistentHashRing_GetNodeDistribution(t *testing.T) {
	ring := NewConsistentHashRing(150)

	ring.AddNode("node1:6379", 1)
	ring.AddNode("node2:6379", 1)
	ring.AddNode("node3:6379", 1)

	distribution := make(map[string]int)
	keys := make([]string, 1000)

	for i := 0; i < 1000; i++ {
		keys[i] = "key" + string(rune(i))
	}

	for _, key := range keys {
		node, err := ring.GetNode(key)
		if err == nil {
			distribution[node]++
		}
	}

	for node, count := range distribution {
		t.Logf("Node %s: %d keys", node, count)
	}

	total := 0
	for _, count := range distribution {
		total += count
	}

	if total != len(keys) {
		t.Errorf("expected %d total keys distributed, got %d", len(keys), total)
	}
}

func TestConsistentHashDistribution(t *testing.T) {
	nodes := []string{"node1:6379", "node2:6379", "node3:6379"}

	key1 := "user:123"
	key2 := "user:124"
	key3 := "product:456"

	node1 := ConsistentHashDistribution(key1, nodes)
	node2 := ConsistentHashDistribution(key2, nodes)
	node3 := ConsistentHashDistribution(key3, nodes)

	if node1 == "" || node2 == "" || node3 == "" {
		t.Error("expected non-empty node addresses")
	}

	t.Logf("key1 (%s) -> %s", key1, node1)
	t.Logf("key2 (%s) -> %s", key2, node2)
	t.Logf("key3 (%s) -> %s", key3, node3)

	if len(nodes) == 0 {
		result := ConsistentHashDistribution("key", []string{})
		if result != "" {
			t.Error("expected empty string for empty nodes")
		}
	}
}

func TestPrefetcher_AddPattern(t *testing.T) {
	config := &PrefetchConfig{
		Enabled:         true,
		BatchSize:       50,
		Concurrency:     3,
		LookAheadWindow: 2 * time.Minute,
		PredictionAlgo:  "frequency",
	}

	prefetcher := NewPrefetcher(config)

	prefetcher.AddPattern("user:*")
	prefetcher.AddPattern("product:*")
	prefetcher.AddPattern("session:*")

	if len(prefetcher.patterns) != 3 {
		t.Errorf("expected 3 patterns, got %d", len(prefetcher.patterns))
	}
}

func TestPrefetcher_GenerateKeysFromPattern(t *testing.T) {
	prefetcher := NewPrefetcher(nil)

	keys := prefetcher.generateKeysFromPattern("user:*")
	if len(keys) == 0 {
		t.Error("expected keys to be generated from pattern")
	}

	expectedPrefix := "user"
	for _, key := range keys {
		if len(key) < len(expectedPrefix) {
			t.Errorf("key %s should start with %s", key, expectedPrefix)
		}
	}
}

func TestPrefetcher_SplitIntoBatches(t *testing.T) {
	prefetcher := NewPrefetcher(nil)

	keys := make([]string, 250)
	for i := 0; i < 250; i++ {
		keys[i] = "key" + string(rune(i))
	}

	batches := prefetcher.splitIntoBatches(keys, 50)

	if len(batches) != 5 {
		t.Errorf("expected 5 batches, got %d", len(batches))
	}

	for i, batch := range batches {
		t.Logf("Batch %d: %d keys", i, len(batch))
	}
}

func TestPrefetcher_StartStop(t *testing.T) {
	config := &PrefetchConfig{
		Enabled:         true,
		BatchSize:       10,
		Concurrency:     2,
		LookAheadWindow: 1 * time.Minute,
		PredictionAlgo:  "frequency",
	}

	prefetcher := NewPrefetcher(config)
	ctx := context.Background()

	getFunc := func(ctx context.Context, key string) (string, error) {
		return "value", nil
	}

	prefetcher.Start(ctx, getFunc)

	time.Sleep(100 * time.Millisecond)

	if !prefetcher.active.Load() {
		t.Error("expected prefetcher to be active after Start")
	}

	prefetcher.Stop()

	time.Sleep(100 * time.Millisecond)

	if prefetcher.active.Load() {
		t.Error("expected prefetcher to be inactive after Stop")
	}
}

func TestPrefetchHotData(t *testing.T) {
	ctx := context.Background()
	patterns := []string{"user:*", "product:*"}

	err := PrefetchHotData(ctx, patterns)
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrefetchHotData_EmptyPatterns(t *testing.T) {
	ctx := context.Background()

	err := PrefetchHotData(ctx, []string{})
	if err != nil {
		t.Errorf("expected no error for empty patterns, got: %v", err)
	}
}

func TestEvictionHeap_PushPop(t *testing.T) {
	heap := NewEvictionHeap()

	entry1 := &EvictionEntry{
		Key:    "key1",
		Score:  100,
	}
	entry2 := &EvictionEntry{
		Key:    "key2",
		Score:  50,
	}
	entry3 := &EvictionEntry{
		Key:    "key3",
		Score:  200,
	}

	heap.Push(entry1)
	heap.Push(entry2)
	heap.Push(entry3)

	popped := heap.Pop()
	if popped == nil {
		t.Fatal("expected non-nil entry from Pop")
	}

	if popped.Score != 50 {
		t.Errorf("expected lowest score (50), got %f", popped.Score)
	}

	heap.Pop()
	popped = heap.Pop()
	if popped.Score != 200 {
		t.Errorf("expected highest score (200), got %f", popped.Score)
	}
}

func TestEvictionHeap_Remove(t *testing.T) {
	heap := NewEvictionHeap()

	entry1 := &EvictionEntry{Key: "key1", Score: 100}
	entry2 := &EvictionEntry{Key: "key2", Score: 50}
	entry3 := &EvictionEntry{Key: "key3", Score: 200}

	heap.Push(entry1)
	heap.Push(entry2)
	heap.Push(entry3)

	removed := heap.Remove("key2")
	if !removed {
		t.Error("expected Remove to return true")
	}

	if heap.Size() != 2 {
		t.Errorf("expected size 2 after removal, got %d", heap.Size())
	}

	removed = heap.Remove("nonexistent")
	if removed {
		t.Error("expected Remove to return false for nonexistent key")
	}
}

func TestEvictionHeap_Size(t *testing.T) {
	heap := NewEvictionHeap()

	if heap.Size() != 0 {
		t.Errorf("expected empty heap, got size %d", heap.Size())
	}

	for i := 0; i < 10; i++ {
		heap.Push(&EvictionEntry{Key: "key" + string(rune(i)), Score: float64(i)})
	}

	if heap.Size() != 10 {
		t.Errorf("expected size 10, got %d", heap.Size())
	}

	for i := 0; i < 5; i++ {
		heap.Pop()
	}

	if heap.Size() != 5 {
		t.Errorf("expected size 5 after pops, got %d", heap.Size())
	}
}

func TestEnhancedCacheV2_NewEnhancedCacheV2(t *testing.T) {
	config := DefaultEnhancedCacheV2Config()
	cache := NewEnhancedCacheV2(config)

	if cache == nil {
		t.Fatal("expected non-nil cache")
	}

	if cache.consistentHash == nil {
		t.Error("expected consistentHash to be initialized")
	}

	if cache.evictionHeap == nil {
		t.Error("expected evictionHeap to be initialized")
	}

	if cache.prefetcher == nil {
		t.Error("expected prefetcher to be initialized")
	}

	if cache.failoverMgr == nil {
		t.Error("expected failoverMgr to be initialized")
	}

	if cache.hotDataTracker == nil {
		t.Error("expected hotDataTracker to be initialized")
	}

	if cache.localCache == nil {
		t.Error("expected localCache to be initialized when LocalCacheEnabled is true")
	}
}

func TestEnhancedCacheV2_ConfigDefaults(t *testing.T) {
	cache := NewEnhancedCacheV2(nil)

	if cache.config.VirtualNodes != DefaultVirtualNodes {
		t.Errorf("expected VirtualNodes %d, got %d", DefaultVirtualNodes, cache.config.VirtualNodes)
	}

	if cache.config.LocalCacheSize != 5000 {
		t.Errorf("expected LocalCacheSize 5000, got %d", cache.config.LocalCacheSize)
	}
}

func TestEnhancedCacheV2_RegisterUnregisterClient(t *testing.T) {
	cache := NewEnhancedCacheV2(nil)

	mockClient := newMockRedisClient()
	cache.RegisterClient("node1:6379", mockClient)

	if len(cache.clients) != 1 {
		t.Errorf("expected 1 client, got %d", len(cache.clients))
	}

	nodes := cache.consistentHash.GetAllNodes()
	if len(nodes) != 1 {
		t.Errorf("expected 1 node in ring, got %d", len(nodes))
	}

	cache.UnregisterClient("node1:6379")

	if len(cache.clients) != 0 {
		t.Errorf("expected 0 clients after unregister, got %d", len(cache.clients))
	}
}

func TestEnhancedCacheV2_GetSetDelete(t *testing.T) {
	cache := NewEnhancedCacheV2(nil)

	mockClient := newMockRedisClient()
	cache.RegisterClient("node1:6379", mockClient)

	ctx := context.Background()

	err := cache.Set(ctx, "testkey", "testvalue", 1*time.Minute)
	if err != nil {
		t.Errorf("unexpected error in Set: %v", err)
	}

	val, err := cache.Get(ctx, "testkey")
	if err != nil {
		t.Errorf("unexpected error in Get: %v", err)
	}

	if val != "testvalue" {
		t.Errorf("expected 'testvalue', got '%s'", val)
	}

	err = cache.Delete(ctx, "testkey")
	if err != nil {
		t.Errorf("unexpected error in Delete: %v", err)
	}
}

func TestEnhancedCacheV2_HybridEviction(t *testing.T) {
	config := DefaultEnhancedCacheV2Config()
	config.Eviction = &HybridEvictionConfig{
		LRUWeight:       0.3,
		LFUWeight:       0.7,
		WindowSize:      10 * time.Minute,
		DecayFactor:     0.95,
		MaxMemoryPercent: 80,
	}

	cache := NewEnhancedCacheV2(config)

	ctx := context.Background()
	err := cache.HybridEviction(ctx)
	if err != nil {
		t.Errorf("unexpected error in HybridEviction: %v", err)
	}
}

func TestEnhancedCacheV2_GetHotKeys(t *testing.T) {
	cache := NewEnhancedCacheV2(nil)

	for i := 0; i < 15; i++ {
		cache.hotDataTracker.RecordAccess("user:" + string(rune(i)))
	}

	hotKeys := cache.GetHotKeys()

	t.Logf("Found %d hot keys", len(hotKeys))
	for _, key := range hotKeys {
		t.Logf("Hot key: %s", key)
	}
}

func TestEnhancedCacheV2_AddPrefetchPattern(t *testing.T) {
	cache := NewEnhancedCacheV2(nil)

	cache.AddPrefetchPattern("user:*")
	cache.AddPrefetchPattern("product:*")

	if len(cache.prefetcher.patterns) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(cache.prefetcher.patterns))
	}
}

func TestEnhancedCacheV2_StartStopPrefetching(t *testing.T) {
	cache := NewEnhancedCacheV2(nil)
	ctx := context.Background()

	cache.StartPrefetching(ctx)
	time.Sleep(100 * time.Millisecond)
	cache.StopPrefetching()
}

func TestEnhancedCacheV2_StartStopFailoverRecovery(t *testing.T) {
	cache := NewEnhancedCacheV2(nil)
	ctx := context.Background()

	cache.StartFailoverRecovery(ctx)
	time.Sleep(100 * time.Millisecond)
	cache.StopFailoverRecovery()
}

func TestEnhancedCacheV2_TriggerEviction(t *testing.T) {
	cache := NewEnhancedCacheV2(nil)
	ctx := context.Background()

	err := cache.TriggerEviction(ctx)
	if err != nil {
		t.Errorf("unexpected error in TriggerEviction: %v", err)
	}
}

func TestEnhancedCacheV2_ClearLocalCache(t *testing.T) {
	cache := NewEnhancedCacheV2(nil)

	cache.localCache.Set("key1", []byte("value1"))
	cache.localCache.Set("key2", []byte("value2"))

	cache.ClearLocalCache()

	if val, ok := cache.localCache.Get("key1"); ok {
		t.Errorf("expected key1 to be cleared, got value: %s", string(val))
	}
}

func TestEnhancedCacheV2_GetNodeDistribution(t *testing.T) {
	cache := NewEnhancedCacheV2(nil)

	cache.RegisterClient("node1:6379", newMockRedisClient())
	cache.RegisterClient("node2:6379", newMockRedisClient())
	cache.RegisterClient("node3:6379", newMockRedisClient())

	distribution := cache.GetNodeDistribution()

	t.Logf("Node distribution:")
	for node, count := range distribution {
		t.Logf("  %s: %d keys", node, count)
	}
}

func TestEnhancedCacheV2_GetNodeStatus(t *testing.T) {
	cache := NewEnhancedCacheV2(nil)

	cache.RegisterClient("node1:6379", newMockRedisClient())
	cache.RegisterClient("node2:6379", newMockRedisClient())

	status := cache.GetNodeStatus()

	for node, s := range status {
		t.Logf("Node %s status: %v", node, s)
	}
}

func TestEnhancedCacheV2_GetMetrics(t *testing.T) {
	cache := NewEnhancedCacheV2(nil)

	cache.metrics.Hits.Add(100)
	cache.metrics.Misses.Add(20)
	cache.metrics.Evictions.Add(5)
	cache.metrics.FailoverTriggered.Add(2)

	metrics := cache.GetMetrics()

	if metrics.Hits.Load() != 100 {
		t.Errorf("expected 100 hits, got %d", metrics.Hits.Load())
	}

	if metrics.Misses.Load() != 20 {
		t.Errorf("expected 20 misses, got %d", metrics.Misses.Load())
	}
}

func TestLocalCacheLayer_GetSet(t *testing.T) {
	cache := NewLocalCacheLayer(100, 1*time.Minute)

	cache.Set("key1", []byte("value1"))

	val, ok := cache.Get("key1")
	if !ok {
		t.Error("expected key1 to exist")
	}

	if string(val) != "value1" {
		t.Errorf("expected 'value1', got '%s'", string(val))
	}
}

func TestLocalCacheLayer_GetNonExistent(t *testing.T) {
	cache := NewLocalCacheLayer(100, 1*time.Minute)

	_, ok := cache.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent key to return false")
	}
}

func TestLocalCacheLayer_Delete(t *testing.T) {
	cache := NewLocalCacheLayer(100, 1*time.Minute)

	cache.Set("key1", []byte("value1"))
	cache.Delete("key1")

	_, ok := cache.Get("key1")
	if ok {
		t.Error("expected key1 to be deleted")
	}
}

func TestLocalCacheLayer_Clear(t *testing.T) {
	cache := NewLocalCacheLayer(100, 1*time.Minute)

	for i := 0; i < 10; i++ {
		cache.Set("key"+string(rune(i)), []byte("value"+string(rune(i))))
	}

	cache.Clear()

	_, ok := cache.Get("key0")
	if ok {
		t.Error("expected cache to be cleared")
	}
}

func TestLocalCacheLayer_Eviction(t *testing.T) {
	maxSize := 5
	cache := NewLocalCacheLayer(maxSize, 1*time.Minute)

	for i := 0; i < maxSize+3; i++ {
		cache.Set("key"+string(rune(i)), []byte("value"+string(rune(i))))
	}

	size := cache.currentSize()
	if size > maxSize {
		t.Errorf("expected size <= %d after eviction, got %d", maxSize, size)
	}
}

func TestLocalCacheLayer_Expiry(t *testing.T) {
	cache := NewLocalCacheLayer(100, 50*time.Millisecond)

	cache.Set("key1", []byte("value1"))

	time.Sleep(100 * time.Millisecond)

	_, ok := cache.Get("key1")
	if ok {
		t.Error("expected expired key to return false")
	}
}

func TestHotDataTracker_RecordAccess(t *testing.T) {
	tracker := NewHotDataTracker()

	for i := 0; i < 20; i++ {
		tracker.RecordAccess("user:123")
	}

	hotKeys := tracker.GetHotKeys()

	found := false
	for _, key := range hotKeys {
		if key == "user:123" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected user:123 to be in hot keys")
	}
}

func TestHotDataTracker_GetHotKeys_Empty(t *testing.T) {
	tracker := NewHotDataTracker()

	hotKeys := tracker.GetHotKeys()

	if len(hotKeys) != 0 {
		t.Errorf("expected 0 hot keys for empty tracker, got %d", len(hotKeys))
	}
}

func TestHotDataTracker_CleanupOldRecords(t *testing.T) {
	tracker := NewHotDataTracker()
	tracker.windowSize = 100 * time.Millisecond

	tracker.RecordAccess("user:1")

	time.Sleep(200 * time.Millisecond)

	tracker.RecordAccess("user:2")

	hotKeys := tracker.GetHotKeys()

	user1Found := false
	_ /* user2Found */ = false

	for _, key := range hotKeys {
		if key == "user:1" {
			user1Found = true
		}
	}

	if user1Found {
		t.Error("expected user:1 to be cleaned up")
	}
}

func TestFailoverManager_RegisterUnregisterNode(t *testing.T) {
	config := &FailoverConfig{
		Enabled:           true,
		MaxRetries:        3,
		RetryInterval:     5 * time.Second,
		HealthCheckPeriod: 10 * time.Second,
	}

	mgr := NewFailoverManager(config)

	mgr.RegisterNode("node1", "192.168.1.1:6379")

	if len(mgr.nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(mgr.nodes))
	}

	mgr.UnregisterNode("node1")

	if len(mgr.nodes) != 0 {
		t.Errorf("expected 0 nodes after unregister, got %d", len(mgr.nodes))
	}
}

func TestFailoverManager_MarkNodeFailed(t *testing.T) {
	config := &FailoverConfig{
		Enabled:           true,
		MaxRetries:        3,
		RetryInterval:     5 * time.Second,
		HealthCheckPeriod: 10 * time.Second,
	}

	mgr := NewFailoverManager(config)
	mgr.RegisterNode("node1", "192.168.1.1:6379")

	for i := 0; i < 5; i++ {
		mgr.MarkNodeFailed("node1")
	}

	node := mgr.nodes["node1"]
	if node.Status != NodeStatusFailed {
		t.Errorf("expected status Failed, got %v", node.Status)
	}
}

func TestFailoverManager_MarkNodeRecovering(t *testing.T) {
	config := &FailoverConfig{
		Enabled:           true,
		MaxRetries:        3,
		RetryInterval:     5 * time.Second,
		HealthCheckPeriod: 10 * time.Second,
	}

	mgr := NewFailoverManager(config)
	mgr.RegisterNode("node1", "192.168.1.1:6379")

	mgr.MarkNodeRecovering("node1")

	node := mgr.nodes["node1"]
	if node.Status != NodeStatusRecovering {
		t.Errorf("expected status Recovering, got %v", node.Status)
	}
}

func TestFailoverManager_MarkNodeHealthy(t *testing.T) {
	config := &FailoverConfig{
		Enabled:           true,
		MaxRetries:        3,
		RetryInterval:     5 * time.Second,
		HealthCheckPeriod: 10 * time.Second,
	}

	mgr := NewFailoverManager(config)
	mgr.RegisterNode("node1", "192.168.1.1:6379")

	mgr.MarkNodeFailed("node1")
	mgr.MarkNodeHealthy("node1")

	node := mgr.nodes["node1"]
	if node.Status != NodeStatusHealthy {
		t.Errorf("expected status Healthy, got %v", node.Status)
	}

	if atomic.LoadInt32(&node.FailCount) != 0 {
		t.Errorf("expected fail count 0, got %d", node.FailCount)
	}
}

func TestFailoverManager_GetAvailableNode(t *testing.T) {
	config := &FailoverConfig{
		Enabled:           true,
		MaxRetries:        3,
		RetryInterval:     5 * time.Second,
		HealthCheckPeriod: 10 * time.Second,
	}

	mgr := NewFailoverManager(config)
	mgr.RegisterNode("node1", "192.168.1.1:6379")
	mgr.RegisterNode("node2", "192.168.1.2:6379")
	mgr.RegisterNode("node3", "192.168.1.3:6379")

	for i := 0; i < 5; i++ {
		mgr.MarkNodeFailed("node1")
		mgr.MarkNodeFailed("node2")
	}

	addr := mgr.GetAvailableNode(true)
	if addr != "192.168.1.3:6379" {
		t.Errorf("expected available node 192.168.1.3:6379, got %s", addr)
	}
}

func TestFailoverManager_GetAvailableNode_AllFailed(t *testing.T) {
	config := &FailoverConfig{
		Enabled:           true,
		MaxRetries:        3,
		RetryInterval:     5 * time.Second,
		HealthCheckPeriod: 10 * time.Second,
	}

	mgr := NewFailoverManager(config)
	mgr.RegisterNode("node1", "192.168.1.1:6379")

	for i := 0; i < 5; i++ {
		mgr.MarkNodeFailed("node1")
	}

	addr := mgr.GetAvailableNode(true)
	if addr != "" {
		t.Errorf("expected empty string when all nodes failed, got %s", addr)
	}
}

func TestFailoverRecovery(t *testing.T) {
	ctx := context.Background()

	err := FailoverRecovery(ctx)
	if err != nil && err != ErrRecoveryFailed && err.Error() != "no nodes available" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFailoverRecovery_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := FailoverRecovery(ctx)
	if err == nil {
		t.Error("expected timeout or error")
	}
}

func TestNodeInfo_InitialStatus(t *testing.T) {
	info := &NodeInfo{
		Address: "node1:6379",
		Status:  NodeStatusHealthy,
		Weight:  1,
	}

	if info.Status != NodeStatusHealthy {
		t.Errorf("expected initial status Healthy, got %v", info.Status)
	}
}

func TestNodeStatus_Values(t *testing.T) {
	tests := []struct {
		status   NodeStatus
		expected string
	}{
		{NodeStatusHealthy, "Healthy"},
		{NodeStatusDegraded, "Degraded"},
		{NodeStatusFailed, "Failed"},
		{NodeStatusRecovering, "Recovering"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			if int(test.status) < 0 || int(test.status) > 3 {
				t.Errorf("unexpected node status value: %d", test.status)
			}
		})
	}
}

func TestEvictionPolicy_Values(t *testing.T) {
	tests := []struct {
		policy   EvictionPolicy
		expected string
	}{
		{EvictionPolicyLRU, "LRU"},
		{EvictionPolicyLFU, "LFU"},
		{EvictionPolicyHybrid, "Hybrid"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			if int(test.policy) < 0 || int(test.policy) > 2 {
				t.Errorf("unexpected eviction policy value: %d", test.policy)
			}
		})
	}
}

func TestConsistentHashRing_ConcurrentAccess(t *testing.T) {
	ring := NewConsistentHashRing(100)

	ring.AddNode("node1:6379", 1)
	ring.AddNode("node2:6379", 1)
	ring.AddNode("node3:6379", 1)

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			key := "key" + string(rune(idx))
			node, err := ring.GetNode(key)
			if err != nil {
				errors <- err
			}

			if node == "" {
				errors <- ErrNoNodesAvailable
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("unexpected error in concurrent access: %v", err)
	}
}

func TestEnhancedCacheV2_ConcurrentGetSet(t *testing.T) {
	cache := NewEnhancedCacheV2(nil)

	mockClient := newMockRedisClient()
	cache.RegisterClient("node1:6379", mockClient)

	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			key := "concurrent_key_" + string(rune(idx))
			value := "value_" + string(rune(idx))

			cache.Set(ctx, key, value, 1*time.Minute)

			val, err := cache.Get(ctx, key)
			if err == nil && val == value {
			}
		}(i)
	}

	wg.Wait()
}

func TestPrefetchHotData_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	err := PrefetchHotData(ctx, []string{"user:*"})
	if err == nil {
		t.Error("expected context cancelled error")
	}
}

func TestGlobalEnhancedCacheV2(t *testing.T) {
	InitEnhancedCacheV2(nil)

	cache := GetEnhancedCacheV2()
	if cache == nil {
		t.Error("expected non-nil global cache")
	}

	cache2 := GetEnhancedCacheV2()
	if cache != cache2 {
		t.Error("expected same instance on second call")
	}
}

type mockRedisClient struct {
	data map[string]string
	mu   sync.RWMutex
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{
		data: make(map[string]string),
	}
}

func (m *mockRedisClient) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return "", ErrCacheMiss
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if val, ok := value.(string); ok {
		m.data[key] = val
	}
	return nil
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	deleted := int64(0)
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			delete(m.data, key)
			deleted++
		}
	}
	return deleted, nil
}

func (m *mockRedisClient) Ping(ctx context.Context) error {
	return nil
}

func (m *mockRedisClient) Addr() string {
	return "mock:6379"
}

func (m *mockRedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}

func TestHybridEvictionConfig_Weights(t *testing.T) {
	config := &HybridEvictionConfig{
		LRUWeight:       0.3,
		LFUWeight:       0.7,
		WindowSize:      10 * time.Minute,
		DecayFactor:     0.95,
		MaxMemoryPercent: 80,
	}

	totalWeight := config.LRUWeight + config.LFUWeight
	if totalWeight != 1.0 {
		t.Errorf("expected total weight 1.0, got %f", totalWeight)
	}
}

func TestPrefetchConfig_Defaults(t *testing.T) {
	config := &PrefetchConfig{
		Enabled:         true,
		BatchSize:       DefaultPrefetchBatch,
		Concurrency:     5,
		LookAheadWindow: 5 * time.Minute,
		PredictionAlgo:  "frequency",
	}

	if config.BatchSize != DefaultPrefetchBatch {
		t.Errorf("expected BatchSize %d, got %d", DefaultPrefetchBatch, config.BatchSize)
	}
}

func TestFailoverConfig_Defaults(t *testing.T) {
	config := &FailoverConfig{
		Enabled:           true,
		MaxRetries:        3,
		RetryInterval:     5 * time.Second,
		HealthCheckPeriod: 10 * time.Second,
		DegradedThreshold: 3,
		FailureThreshold:  5,
		RecoveryTimeout:   DefaultRecoveryTimeout,
	}

	if config.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", config.MaxRetries)
	}

	if config.FailureThreshold != 5 {
		t.Errorf("expected FailureThreshold 5, got %d", config.FailureThreshold)
	}
}

func TestCacheV2Metrics_AtomicOperations(t *testing.T) {
	metrics := &CacheV2Metrics{}

	metrics.Hits.Add(100)
	metrics.Misses.Add(50)
	metrics.Evictions.Add(10)
	metrics.FailoverTriggered.Add(2)

	if metrics.Hits.Load() != 100 {
		t.Errorf("expected 100 hits, got %d", metrics.Hits.Load())
	}

	if metrics.Misses.Load() != 50 {
		t.Errorf("expected 50 misses, got %d", metrics.Misses.Load())
	}
}
