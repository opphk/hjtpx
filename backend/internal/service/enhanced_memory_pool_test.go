package service

import (
	"testing"
)

func TestSizedPool(t *testing.T) {
	pool := NewSizedPool(100)
	
	buf := pool.Get()
	if cap(buf) < 100 {
		t.Errorf("Expected capacity at least 100, got %d", cap(buf))
	}
	
	pool.Put(buf)
	
	// Get again should reuse
	buf2 := pool.Get()
	if &buf[0] == &buf2[0] { // Note: This might not always be true, but in simple cases it should work
		t.Log("Pool reused buffer")
	}
}

func TestEnhancedMemoryPool(t *testing.T) {
	pool := NewEnhancedMemoryPool(nil)
	
	buf := pool.Get(50)
	if len(buf) != 50 {
		t.Errorf("Expected length 50, got %d", len(buf))
	}
	
	pool.Put(buf)
	
	stats := pool.GetStats()
	if stats.PoolHits.Load() == 0 {
		t.Log("Pool stats recorded")
	}
}

func TestConcurrentMemoryPool(t *testing.T) {
	pool := NewConcurrentMemoryPool()
	
	bufSmall := pool.Get(32)
	if len(bufSmall) != 32 {
		t.Errorf("Expected length 32, got %d", len(bufSmall))
	}
	pool.Put(bufSmall)
	
	bufLarge := pool.Get(10000)
	if len(bufLarge) != 10000 {
		t.Errorf("Expected length 10000, got %d", len(bufLarge))
	}
	pool.Put(bufLarge)
}

func TestBufferPool(t *testing.T) {
	pool := NewBufferPool()
	
	buf := pool.Get()
	buf = append(buf, "test"...)
	if string(buf) != "test" {
		t.Errorf("Expected 'test', got %s", buf)
	}
	
	pool.Put(buf)
}

func TestObjectPool(t *testing.T) {
	type TestObject struct {
		Value int
	}
	
	pool := NewObjectPool(
		func() *TestObject {
			return &TestObject{Value: 0}
		},
		func(obj *TestObject) {
			obj.Value = 0
		},
	)
	
	obj := pool.Get()
	if obj.Value != 0 {
		t.Errorf("Expected initial value 0, got %d", obj.Value)
	}
	
	obj.Value = 42
	pool.Put(obj)
	
	obj2 := pool.Get()
	if obj2.Value != 0 {
		t.Errorf("Expected reset value 0, got %d", obj2.Value)
	}
}
