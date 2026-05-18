package redis

import (
	"context"
	"testing"
	"time"
)

func BenchmarkLRUCacheGet(b *testing.B) {
	cache := NewLRUCache(10000)

	for i := 0; i < 1000; i++ {
		cache.Set(string(rune(i)), []byte("value"), 5*time.Minute, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("500")
	}
}

func BenchmarkLRUCacheSet(b *testing.B) {
	cache := NewLRUCache(10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(string(rune(i)), []byte("value"), 5*time.Minute, 0)
	}
}

func BenchmarkLRUCacheDelete(b *testing.B) {
	cache := NewLRUCache(10000)

	for i := 0; i < 1000; i++ {
		cache.Set(string(rune(i)), []byte("value"), 5*time.Minute, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Delete("500")
	}
}

func TestLRUCache(t *testing.T) {
	cache := NewLRUCache(100)

	cache.Set("key1", []byte("value1"), 5*time.Minute, 0)

	val, err := cache.Get("key1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if string(val) != "value1" {
		t.Errorf("Expected 'value1', got '%s'", string(val))
	}

	deleted := cache.Delete("key1")
	if !deleted {
		t.Error("Expected delete to return true")
	}

	_, err = cache.Get("key1")
	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss, got %v", err)
	}
}

func TestLRUCacheEviction(t *testing.T) {
	cache := NewLRUCache(10)

	for i := 0; i < 20; i++ {
		cache.Set(string(rune(i)), []byte("value"), 5*time.Minute, 0)
	}

	if cache.Size() != 10 {
		t.Errorf("Expected size 10, got %d", cache.Size())
	}
}

func TestLRUCacheExpiration(t *testing.T) {
	cache := NewLRUCache(100)

	cache.Set("key1", []byte("value1"), 1*time.Millisecond, 0)

	_, err := cache.Get("key1")
	if err != nil {
		t.Errorf("Expected no error immediately after set, got %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	_, err = cache.Get("key1")
	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss after expiration, got %v", err)
	}
}

func BenchmarkPipelineBatcher(b *testing.B) {
	batcher := NewPipelineBatcher(100, 10*time.Millisecond)
	batcher.Start()
	defer batcher.Stop()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batcher.AddSet(ctx, string(rune(i)), "value", 5*time.Minute)
	}
}

func TestPipelineBatcher(t *testing.T) {
	batcher := NewPipelineBatcher(10, 100*time.Millisecond)
	batcher.Start()
	defer batcher.Stop()

	ctx := context.Background()

	for i := 0; i < 5; i++ {
		batcher.AddSet(ctx, string(rune(i)), "value", 5*time.Minute)
	}

	time.Sleep(200 * time.Millisecond)
}
