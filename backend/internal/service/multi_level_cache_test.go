package service

import (
	"context"
	"testing"
	"time"
)

func TestNewOptimizedLocalCache(t *testing.T) {
	cache := NewOptimizedLocalCache(100, 10*1024, 5*time.Minute)
	if cache == nil {
		t.Fatal("Expected cache to be created")
	}
}

func TestLocalCacheGetSet(t *testing.T) {
	cache := NewOptimizedLocalCache(100, 10*1024, 5*time.Minute)
	
	key := "test-key"
	value := []byte("test-value")
	
	cache.Set(key, value, 5*time.Minute)
	
	retrieved, ok := cache.Get(key)
	if !ok {
		t.Fatal("Expected value to be found")
	}
	
	if string(retrieved) != string(value) {
		t.Errorf("Expected value %s, got %s", value, retrieved)
	}
}

func TestLocalCacheDelete(t *testing.T) {
	cache := NewOptimizedLocalCache(100, 10*1024, 5*time.Minute)
	
	key := "test-key"
	value := []byte("test-value")
	
	cache.Set(key, value, 5*time.Minute)
	cache.Delete(key)
	
	_, ok := cache.Get(key)
	if ok {
		t.Fatal("Expected value to be deleted")
	}
}

func TestLocalCacheClear(t *testing.T) {
	cache := NewOptimizedLocalCache(100, 10*1024, 5*time.Minute)
	
	for i := 0; i < 10; i++ {
		key := string(rune('a' + i))
		cache.Set(key, []byte(key), 5*time.Minute)
	}
	
	cache.Clear()
	
	for i := 0; i < 10; i++ {
		key := string(rune('a' + i))
		_, ok := cache.Get(key)
		if ok {
			t.Errorf("Expected key %s to be cleared", key)
		}
	}
}

func TestNewMultiLevelCacheService(t *testing.T) {
	config := DefaultMultiLevelConfig
	cache := NewMultiLevelCacheService(config)
	
	if cache == nil {
		t.Fatal("Expected multi-level cache to be created")
	}
	
	if !cache.IsEnabled() {
		t.Fatal("Expected cache to be enabled by default")
	}
}

func TestMultiLevelCacheGetSet(t *testing.T) {
	cache := NewMultiLevelCacheService(DefaultMultiLevelConfig)
	ctx := context.Background()
	
	key := "test-key"
	value := []byte("test-value")
	
	err := cache.Set(ctx, key, value, 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}
	
	retrieved, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}
	
	if string(retrieved) != string(value) {
		t.Errorf("Expected value %s, got %s", value, retrieved)
	}
}

func TestMultiLevelCacheDelete(t *testing.T) {
	cache := NewMultiLevelCacheService(DefaultMultiLevelConfig)
	ctx := context.Background()
	
	key := "test-key"
	value := []byte("test-value")
	
	err := cache.Set(ctx, key, value, 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}
	
	err = cache.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to delete value: %v", err)
	}
	
	_, err = cache.Get(ctx, key)
	if err == nil {
		t.Fatal("Expected error when getting deleted key")
	}
}

func TestPromotionPolicy(t *testing.T) {
	policy := NewPromotionPolicy()
	
	if policy == nil {
		t.Fatal("Expected promotion policy to be created")
	}
	
	policy.RecordAccess("hot-key", 10*time.Millisecond)
	
	hotKeys := policy.GetHotKeys(10)
	if len(hotKeys) > 0 {
		t.Log("Hot keys detected:", hotKeys)
	}
}
