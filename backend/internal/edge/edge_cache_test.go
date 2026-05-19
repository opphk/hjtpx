package edge

import (
	"context"
	"testing"
	"time"
)

func TestEdgeCacheSync_NewEdgeCacheSync(t *testing.T) {
	config := &CacheSyncConfig{
		Strategy:     SyncStrategyEventual,
		BatchSize:    100,
		SyncInterval: 5 * time.Second,
	}

	sync := NewEdgeCacheSync(nil, nil, config)

	if sync == nil {
		t.Fatal("Expected sync to not be nil")
	}

	if len(sync.cache) != 0 {
		t.Errorf("Expected 0 cache entries initially, got %d", len(sync.cache))
	}
}

func TestEdgeCacheSync_Set(t *testing.T) {
	sync := NewEdgeCacheSync(nil, nil, nil)
	ctx := context.Background()

	err := sync.Set(ctx, "key1", "value1", 5*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, err := sync.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if value != "value1" {
		t.Errorf("Expected value 'value1', got '%v'", value)
	}
}

func TestEdgeCacheSync_Get(t *testing.T) {
	sync := NewEdgeCacheSync(nil, nil, nil)
	ctx := context.Background()

	sync.Set(ctx, "key2", "value2", 5*time.Minute)

	value, err := sync.Get(ctx, "key2")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if value != "value2" {
		t.Errorf("Expected value 'value2', got '%v'", value)
	}
}

func TestEdgeCacheSync_GetMiss(t *testing.T) {
	sync := NewEdgeCacheSync(nil, nil, nil)
	ctx := context.Background()

	_, err := sync.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent key")
	}
}

func TestEdgeCacheSync_Delete(t *testing.T) {
	sync := NewEdgeCacheSync(nil, nil, nil)
	ctx := context.Background()

	sync.Set(ctx, "key3", "value3", 5*time.Minute)

	err := sync.Delete(ctx, "key3")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = sync.Get(ctx, "key3")
	if err == nil {
		t.Error("Expected error after deletion")
	}
}

func TestEdgeCacheSync_Exists(t *testing.T) {
	sync := NewEdgeCacheSync(nil, nil, nil)
	ctx := context.Background()

	sync.Set(ctx, "key4", "value4", 5*time.Minute)

	if !sync.Exists(ctx, "key4") {
		t.Error("Expected key to exist")
	}

	if sync.Exists(ctx, "nonexistent") {
		t.Error("Expected nonexistent key to not exist")
	}
}

func TestEdgeCacheSync_Flush(t *testing.T) {
	sync := NewEdgeCacheSync(nil, nil, nil)
	ctx := context.Background()

	sync.Set(ctx, "key5", "value5", 5*time.Minute)
	sync.Set(ctx, "key6", "value6", 5*time.Minute)

	err := sync.Flush(ctx)
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if sync.Size() != 0 {
		t.Errorf("Expected size 0 after flush, got %d", sync.Size())
	}
}

func TestEdgeCacheSync_GetOrSet(t *testing.T) {
	sync := NewEdgeCacheSync(nil, nil, nil)
	ctx := context.Background()

	sync.Set(ctx, "key7", "existing", 5*time.Minute)

	factoryCalled := false
	factory := func() (interface{}, error) {
		factoryCalled = true
		return "new-value", nil
	}

	value, err := sync.GetOrSet(ctx, "key7", factory, 5*time.Minute)
	if err != nil {
		t.Fatalf("GetOrSet failed: %v", err)
	}

	if value != "existing" {
		t.Errorf("Expected 'existing', got '%v'", value)
	}

	if factoryCalled {
		t.Error("Factory should not be called when key exists")
	}

	_, err = sync.GetOrSet(ctx, "key8", factory, 5*time.Minute)
	if err != nil {
		t.Fatalf("GetOrSet failed: %v", err)
	}

	if !factoryCalled {
		t.Error("Factory should be called when key doesn't exist")
	}
}

func TestEdgeCacheSync_GetKeys(t *testing.T) {
	sync := NewEdgeCacheSync(nil, nil, nil)
	ctx := context.Background()

	sync.Set(ctx, "user:1", "value1", 5*time.Minute)
	sync.Set(ctx, "user:2", "value2", 5*time.Minute)
	sync.Set(ctx, "session:1", "value3", 5*time.Minute)

	keys := sync.GetKeys("user:*")
	if len(keys) != 2 {
		t.Errorf("Expected 2 user keys, got %d", len(keys))
	}

	allKeys := sync.GetKeys("*")
	if len(allKeys) != 3 {
		t.Errorf("Expected 3 total keys, got %d", len(allKeys))
	}
}

func TestEdgeCacheSync_GetEntry(t *testing.T) {
	sync := NewEdgeCacheSync(nil, nil, nil)
	ctx := context.Background()

	sync.Set(ctx, "key9", "value9", 5*time.Minute)

	entry, err := sync.GetEntry("key9")
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}

	if entry.Value != "value9" {
		t.Errorf("Expected value 'value9', got '%v'", entry.Value)
	}
}

func TestEdgeCacheSync_WithTags(t *testing.T) {
	sync := NewEdgeCacheSync(nil, nil, nil)
	ctx := context.Background()

	sync.Set(ctx, "key10", "value10", 5*time.Minute, WithTags([]string{"tag1", "tag2"}))

	entries, err := sync.GetByTag(ctx, "tag1")
	if err != nil {
		t.Fatalf("GetByTag failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry with tag1, got %d", len(entries))
	}
}

func TestEdgeCacheSync_InvalidateByTag(t *testing.T) {
	sync := NewEdgeCacheSync(nil, nil, nil)
	ctx := context.Background()

	sync.Set(ctx, "key11", "value11", 5*time.Minute, WithTags([]string{"tag3"}))
	sync.Set(ctx, "key12", "value12", 5*time.Minute, WithTags([]string{"tag3"}))

	err := sync.InvalidateByTag(ctx, "tag3")
	if err != nil {
		t.Fatalf("InvalidateByTag failed: %v", err)
	}

	entries, _ := sync.GetByTag(ctx, "tag3")
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after invalidation, got %d", len(entries))
	}
}

func TestEdgeCacheSync_GetMetrics(t *testing.T) {
	sync := NewEdgeCacheSync(nil, nil, nil)

	metrics := sync.GetMetrics()

	if metrics == nil {
		t.Fatal("Expected metrics to not be nil")
	}

	_ = metrics.TotalSyncs
	_ = metrics.SuccessfulSyncs
	_ = metrics.FailedSyncs
}

func TestEdgeCacheSync_Size(t *testing.T) {
	sync := NewEdgeCacheSync(nil, nil, nil)
	ctx := context.Background()

	if sync.Size() != 0 {
		t.Errorf("Expected size 0 initially, got %d", sync.Size())
	}

	sync.Set(ctx, "key13", "value13", 5*time.Minute)
	if sync.Size() != 1 {
		t.Errorf("Expected size 1, got %d", sync.Size())
	}

	sync.Delete(ctx, "key13")
	if sync.Size() != 0 {
		t.Errorf("Expected size 0 after delete, got %d", sync.Size())
	}
}
