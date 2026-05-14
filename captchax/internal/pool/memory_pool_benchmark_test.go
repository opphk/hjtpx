package pool

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkManagedPoolGetPut(b *testing.B) {
	pool := NewManagedPool(PoolConfig{
		InitialSize: 100,
		MaxSize:     1000,
		New: func() interface{} {
			return make([]byte, 1024)
		},
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := pool.Get()
			pool.Put(obj)
		}
	})
}

func BenchmarkBytesPoolGetPut(b *testing.B) {
	bp := NewBytesPool(100, 1000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := bp.Get()
			buf = buf[:1024]
			bp.Put(buf)
		}
	})
}

func BenchmarkImageBufferPool(b *testing.B) {
	ibp := NewImageBufferPool(50, 500)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := ibp.Get()
			buf.Write(make([]byte, 1024))
			ibp.Put(buf)
		}
	})
}

func BenchmarkTaskResultPool(b *testing.B) {
	trp := NewTaskResultPool(50, 500)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tr := trp.Get()
			tr.Set("key", "value")
			trp.Put(tr)
		}
	})
}

func TestManagedPoolStats(t *testing.T) {
	pool := NewManagedPool(PoolConfig{
		InitialSize: 10,
		MaxSize:     100,
		New: func() interface{} {
			return make([]byte, 1024)
		},
	})

	for i := 0; i < 50; i++ {
		obj := pool.Get()
		pool.Put(obj)
	}

	stats := pool.Stats()
	if stats.CurrentSize < 0 {
		t.Errorf("unexpected current size: %d", stats.CurrentSize)
	}
	if stats.ActiveGet != 50 {
		t.Errorf("expected 50 active gets, got %d", stats.ActiveGet)
	}
}

func TestBytesPool(t *testing.T) {
	bp := NewBytesPool(10, 100)

	buf := bp.Get()
	if cap(buf) < 4096 {
		t.Errorf("expected capacity >= 4096, got %d", cap(buf))
	}

	buf = append(buf, make([]byte, 1024)...)
	bp.Put(buf)

	stats := bp.Stats()
	if stats.CurrentSize < 0 {
		t.Errorf("unexpected current size: %d", stats.CurrentSize)
	}
}

func TestImageBuffer(t *testing.T) {
	ib := &ImageBuffer{data: make([]byte, 0, 1024)}

	data := []byte("hello world")
	n, err := ib.Write(data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected %d bytes written, got %d", len(data), n)
	}

	if ib.Len() != len(data) {
		t.Errorf("expected length %d, got %d", len(data), ib.Len())
	}

	result := ib.Bytes()
	if string(result) != "hello world" {
		t.Errorf("unexpected bytes: %s", string(result))
	}

	ib.Reset()
	if ib.Len() != 0 {
		t.Errorf("expected length 0 after reset, got %d", ib.Len())
	}
}

func TestTaskResult(t *testing.T) {
	tr := &TaskResult{data: make(map[string]interface{})}

	tr.Set("name", "test")
	if tr.Get("name") != "test" {
		t.Error("unexpected value for 'name'")
	}

	tr.SetSuccess(true)
	if !tr.Success() {
		t.Error("expected success to be true")
	}

	tr.Reset()
	if tr.Success() {
		t.Error("expected success to be false after reset")
	}
	if tr.Get("name") != nil {
		t.Error("expected name to be nil after reset")
	}
}

func TestRingBuffer(t *testing.T) {
	rb := newRingBuffer(5)

	for i := 0; i < 5; i++ {
		rb.Push(i)
	}

	if rb.Len() != 5 {
		t.Errorf("expected length 5, got %d", rb.Len())
	}

	rb.Push(5)
	if rb.Len() != 5 {
		t.Errorf("expected length 5 after overflow, got %d", rb.Len())
	}

	for i := 0; i < 5; i++ {
		_, ok := rb.Pop()
		if !ok {
			t.Error("expected pop to succeed")
		}
	}

	_, ok := rb.Pop()
	if ok {
		t.Error("expected pop to fail on empty buffer")
	}
}

func TestPoolRegistry(t *testing.T) {
	pool := NewManagedPool(PoolConfig{
		MaxSize: 100,
		New: func() interface{} {
			return make([]byte, 1024)
		},
	})

	RegisterPool("test_pool", pool)

	retrieved, ok := GetPool("test_pool")
	if !ok {
		t.Error("expected pool to be registered")
	}
	if retrieved != pool {
		t.Error("expected retrieved pool to match original")
	}

	stats := GetAllPoolStats()
	if _, ok := stats["test_pool"]; !ok {
		t.Error("expected test_pool in all stats")
	}
}

func BenchmarkPoolWithoutReset(b *testing.B) {
	pool := NewManagedPool(PoolConfig{
		InitialSize: 100,
		MaxSize:     10000,
		New: func() interface{} {
			return make([]byte, 1024)
		},
		Reset: nil,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := pool.Get()
			pool.Put(obj)
		}
	})
}

func BenchmarkPoolWithReset(b *testing.B) {
	pool := NewManagedPool(PoolConfig{
		InitialSize: 100,
		MaxSize:     10000,
		New: func() interface{} {
			return make([]byte, 1024)
		},
		Reset: func(obj interface{}) {
			if b, ok := obj.([]byte); ok {
				for i := range b {
					b[i] = 0
				}
			}
		},
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := pool.Get()
			pool.Put(obj)
		}
	})
}

func BenchmarkSyncPool(b *testing.B) {
	pool := sync.Pool{
		New: func() interface{} {
			return make([]byte, 1024)
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := pool.Get()
			pool.Put(obj)
		}
	})
}

func BenchmarkManagedPoolVsSyncPool(b *testing.B) {
	mp := NewManagedPool(PoolConfig{
		InitialSize: 100,
		MaxSize:     1000,
		New: func() interface{} {
			return make([]byte, 1024)
		},
	})

	sp := sync.Pool{
		New: func() interface{} {
			return make([]byte, 1024)
		},
	}

	b.Run("ManagedPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			obj := mp.Get()
			mp.Put(obj)
		}
	})

	b.Run("SyncPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			obj := sp.Get()
			sp.Put(obj)
		}
	})
}

func TestPoolConcurrency(t *testing.T) {
	pool := NewManagedPool(PoolConfig{
		InitialSize: 10,
		MaxSize:     1000,
		New: func() interface{} {
			return make([]byte, 1024)
		},
	})

	var wg sync.WaitGroup
	concurrency := 100

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				obj := pool.Get()
				time.Sleep(time.Microsecond)
				pool.Put(obj)
			}
		}()
	}

	wg.Wait()

	stats := pool.Stats()
	if stats.ActiveGet != int64(concurrency*100) {
		t.Errorf("expected %d active gets, got %d", concurrency*100, stats.ActiveGet)
	}
}

func BenchmarkAtomicCounter(b *testing.B) {
	var counter int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			atomic.AddInt64(&counter, 1)
		}
	})
}
