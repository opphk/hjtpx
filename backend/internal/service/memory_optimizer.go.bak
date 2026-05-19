package service

import (
	"bytes"
	"encoding/gob"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type MemoryPool struct {
	pools map[string]*sync.Pool
	stats map[string]*PoolStats
	mu    sync.RWMutex
}

type PoolStats struct {
	TotalAlloc   atomic.Int64
	TotalGet     atomic.Int64
	TotalPut     atomic.Int64
	Hits         atomic.Int64
	Misses       atomic.Int64
	BytesSaved   atomic.Int64
}

func NewMemoryPool() *MemoryPool {
	mp := &MemoryPool{
		pools: make(map[string]*sync.Pool),
		stats: make(map[string]*PoolStats),
	}

	mp.registerDefaultPools()

	return mp
}

func (mp *MemoryPool) registerDefaultPools() {
	mp.Register("bytes.Buffer", func() interface{} {
		return &bytes.Buffer{}
	}, func(v interface{}) interface{} {
		buf := v.(*bytes.Buffer)
		buf.Reset()
		return buf
	})

	mp.Register("[]byte.64", func() interface{} {
		return make([]byte, 0, 64)
	}, func(v interface{}) interface{} {
		return v.([]byte)[:0]
	})

	mp.Register("[]byte.256", func() interface{} {
		return make([]byte, 0, 256)
	}, func(v interface{}) interface{} {
		return v.([]byte)[:0]
	})

	mp.Register("[]byte.1024", func() interface{} {
		return make([]byte, 0, 1024)
	}, func(v interface{}) interface{} {
		return v.([]byte)[:0]
	})

	mp.Register("map[string]interface{}", func() interface{} {
		return make(map[string]interface{})
	}, func(v interface{}) interface{} {
		return clearMap(v.(map[string]interface{}))
	})

	mp.Register("[]int", func() interface{} {
		return make([]int, 0, 16)
	}, func(v interface{}) interface{} {
		return v.([]int)[:0]
	})

	mp.Register("[]string", func() interface{} {
		return make([]string, 0, 16)
	}, func(v interface{}) interface{} {
		return v.([]string)[:0]
	})
}

func clearMap(m map[string]interface{}) map[string]interface{} {
	for k := range m {
		delete(m, k)
	}
	return m
}

func (mp *MemoryPool) Register(name string, newFunc func() interface{}, resetFunc func(interface{}) interface{}) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mp.pools[name] = &sync.Pool{
		New: newFunc,
	}

	mp.stats[name] = &PoolStats{}
}

func (mp *MemoryPool) Get(name string) (interface{}, bool) {
	mp.mu.RLock()
	pool, exists := mp.pools[name]
	stats := mp.stats[name]
	mp.mu.RUnlock()

	if !exists {
		return nil, false
	}

	item := pool.Get()

	if stats != nil {
		stats.TotalGet.Add(1)
	}

	if item == nil {
		if stats != nil {
			stats.Misses.Add(1)
		}
		return nil, false
	}

	if stats != nil {
		stats.Hits.Add(1)
	}

	return item, true
}

func (mp *MemoryPool) Put(name string, item interface{}) {
	mp.mu.RLock()
	pool, exists := mp.pools[name]
	stats := mp.stats[name]
	mp.mu.RUnlock()

	if !exists || pool == nil {
		return
	}

	if stats != nil {
		stats.TotalPut.Add(1)
	}

	pool.Put(item)
}

func (mp *MemoryPool) GetBytes(name string) (*bytes.Buffer, bool) {
	item, ok := mp.Get(name)
	if !ok {
		return nil, false
	}
	return item.(*bytes.Buffer), true
}

func (mp *MemoryPool) PutBytes(name string, buf *bytes.Buffer) {
	if buf != nil {
		buf.Reset()
		mp.Put(name, buf)
	}
}

func (mp *MemoryPool) GetSlice(name string) ([]byte, bool) {
	item, ok := mp.Get(name)
	if !ok {
		return nil, false
	}
	return item.([]byte), true
}

func (mp *MemoryPool) PutSlice(name string, slice []byte) {
	mp.Put(name, slice[:0])
}

func (mp *MemoryPool) GetMap() (map[string]interface{}, bool) {
	val, ok := mp.Get("map[string]interface{}")
	if !ok {
		return nil, false
	}
	return val.(map[string]interface{}), true
}

func (mp *MemoryPool) PutMap(m map[string]interface{}) {
	mp.Put("map[string]interface{}", m)
}

func (mp *MemoryPool) GetStats() map[string]interface{} {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	result := make(map[string]interface{})

	for name, stats := range mp.stats {
		hits := stats.Hits.Load()
		misses := stats.Misses.Load()
		total := hits + misses

		var hitRate float64
		if total > 0 {
			hitRate = float64(hits) / float64(total) * 100
		}

		result[name] = map[string]interface{}{
			"total_get":     stats.TotalGet.Load(),
			"total_put":     stats.TotalPut.Load(),
			"hits":          hits,
			"misses":        misses,
			"hit_rate":      hitRate,
			"bytes_saved":   stats.BytesSaved.Load(),
		}
	}

	return result
}

type ObjectPool[T any] struct {
	pool     chan T
	factory  func() T
	reset    func(T)
	maxSize  int
	stats    *ObjectPoolStats
	mu       sync.RWMutex
}

type ObjectPoolStats struct {
	TotalGet     atomic.Int64
	TotalPut     atomic.Int64
	TotalNew     atomic.Int64
	TotalHits    atomic.Int64
	TotalMisses  atomic.Int64
}

func NewObjectPool[T any](factory func() T, reset func(T), maxSize int) *ObjectPool[T] {
	if maxSize <= 0 {
		maxSize = 1000
	}

	return &ObjectPool[T]{
		pool:    make(chan T, maxSize),
		factory: factory,
		reset:   reset,
		maxSize: maxSize,
		stats:   &ObjectPoolStats{},
	}
}

func (p *ObjectPool[T]) Get() T {
	select {
	case item := <-p.pool:
		p.stats.TotalGet.Add(1)
		p.stats.TotalHits.Add(1)
		return item
	default:
		p.stats.TotalMisses.Add(1)
		p.stats.TotalNew.Add(1)
		return p.factory()
	}
}

func (p *ObjectPool[T]) Put(item T) {
	if p.reset != nil {
		p.reset(item)
	}

	select {
	case p.pool <- item:
		p.stats.TotalPut.Add(1)
	default:
	}
}

func (p *ObjectPool[T]) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_get":    p.stats.TotalGet.Load(),
		"total_put":    p.stats.TotalPut.Load(),
		"total_new":    p.stats.TotalNew.Load(),
		"hits":         p.stats.TotalHits.Load(),
		"misses":       p.stats.TotalMisses.Load(),
	}
}

type BufferPool struct {
	*ObjectPool[*bytes.Buffer]
}

func NewBufferPool(maxSize int) *BufferPool {
	pool := NewObjectPool(
		func() *bytes.Buffer {
			return new(bytes.Buffer)
		},
		func(b *bytes.Buffer) {
			b.Reset()
		},
		maxSize,
	)

	return &BufferPool{ObjectPool: pool}
}

func (bp *BufferPool) Get() *bytes.Buffer {
	return bp.ObjectPool.Get()
}

func (bp *BufferPool) Put(buf *bytes.Buffer) {
	if buf != nil {
		buf.Reset()
		bp.ObjectPool.Put(buf)
	}
}

type StringBuilderPool struct {
	*ObjectPool[*stringsBuilder]
}

type stringsBuilder struct {
	buf bytes.Buffer
}

func NewStringBuilderPool(maxSize int) *StringBuilderPool {
	pool := NewObjectPool(
		func() *stringsBuilder {
			return &stringsBuilder{}
		},
		func(sb *stringsBuilder) {
			sb.buf.Reset()
		},
		maxSize,
	)

	return &StringBuilderPool{ObjectPool: pool}
}

func (sbp *StringBuilderPool) Get() *stringsBuilder {
	return sbp.ObjectPool.Get()
}

func (sbp *StringBuilderPool) Put(sb *stringsBuilder) {
	if sb != nil {
		sb.buf.Reset()
		sbp.ObjectPool.Put(sb)
	}
}

func (sb *stringsBuilder) Write(p []byte) (int, error) {
	return sb.buf.Write(p)
}

func (sb *stringsBuilder) WriteString(s string) (int, error) {
	return sb.buf.WriteString(s)
}

func (sb *stringsBuilder) String() string {
	return sb.buf.String()
}

func (sb *stringsBuilder) Len() int {
	return sb.buf.Len()
}

func (sb *stringsBuilder) Reset() {
	sb.buf.Reset()
}

type MapPool[K comparable, V any] struct {
	*ObjectPool[map[K]V]
}

func NewMapPool[K comparable, V any](maxSize int) *MapPool[K, V] {
	pool := NewObjectPool(
		func() map[K]V {
			return make(map[K]V)
		},
		func(m map[K]V) {
			for k := range m {
				delete(m, k)
			}
		},
		maxSize,
	)

	return &MapPool[K, V]{ObjectPool: pool}
}

func (mp *MapPool[K, V]) Get() map[K]V {
	return mp.ObjectPool.Get()
}

func (mp *MapPool[K, V]) Put(m map[K]V) {
	mp.ObjectPool.Put(m)
}

type SlicePool[T any] struct {
	*ObjectPool[[]T]
	capacity int
}

func NewSlicePool[T any](capacity, maxSize int) *SlicePool[T] {
	if capacity <= 0 {
		capacity = 16
	}

	pool := NewObjectPool(
		func() []T {
			return make([]T, 0, capacity)
		},
		func(s []T) {
			for i := range s {
				var zero T
				s[i] = zero
			}
			s = s[:0]
		},
		maxSize,
	)

	return &SlicePool[T]{
		ObjectPool: pool,
		capacity:   capacity,
	}
}

func (sp *SlicePool[T]) Get() []T {
	return sp.ObjectPool.Get()
}

func (sp *SlicePool[T]) Put(s []T) {
	sp.ObjectPool.Put(s)
}

type MemoryOptimizer struct {
	enabled    bool
	gcInterval time.Duration
	threshold  int64
	stats      *MemoryStats
	mu         sync.RWMutex
}

type MemoryStats struct {
	Alloc        atomic.Int64
	TotalAlloc   atomic.Int64
	Sys          atomic.Int64
	NumGC        atomic.Int32
	LastGC       time.Time
	BySize       [67]runtime.MemStats
	Optimizations atomic.Int64
}

func NewMemoryOptimizer(gcInterval time.Duration, threshold int64) *MemoryOptimizer {
	return &MemoryOptimizer{
		enabled:    true,
		gcInterval: gcInterval,
		threshold:  threshold,
		stats:      &MemoryStats{},
	}
}

func (mo *MemoryOptimizer) Start() {
	go mo.monitor()
}

func (mo *MemoryOptimizer) monitor() {
	ticker := time.NewTicker(mo.gcInterval)
	defer ticker.Stop()

	var lastGCStats runtime.MemStats

	for range ticker.C {
		if !mo.enabled {
			continue
		}

		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)

		mo.updateStats(&stats)

		if mo.shouldTriggerGC(&stats, &lastGCStats) {
			runtime.GC()
			mo.stats.Optimizations.Add(1)
		}

		lastGCStats = stats
	}
}

func (mo *MemoryOptimizer) updateStats(stats *runtime.MemStats) {
	mo.stats.Alloc.Store(int64(stats.Alloc))
	mo.stats.TotalAlloc.Store(int64(stats.TotalAlloc))
	mo.stats.Sys.Store(int64(stats.Sys))
	mo.stats.NumGC.Store(int32(stats.NumGC))
}

func (mo *MemoryOptimizer) shouldTriggerGC(current, last *runtime.MemStats) bool {
	if int64(current.Alloc) > mo.threshold {
		increaseRate := float64(current.Alloc-last.Alloc) / float64(current.Alloc)
		if increaseRate > 0.5 {
			return true
		}
	}

	if current.NumGC > last.NumGC {
		lastGCTime := time.Unix(0, int64(current.LastGC))
		if time.Since(lastGCTime) > 10*time.Minute {
			return true
		}
	}

	return false
}

func (mo *MemoryOptimizer) GetStats() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"alloc":         memStats.Alloc,
		"total_alloc":   memStats.TotalAlloc,
		"sys":           memStats.Sys,
		"num_gc":        memStats.NumGC,
		"num_go":        runtime.NumGoroutine(),
		"threshold":     mo.threshold,
		"optimizations": mo.stats.Optimizations.Load(),
	}
}

func (mo *MemoryOptimizer) Enable(enabled bool) {
	mo.mu.Lock()
	defer mo.mu.Unlock()
	mo.enabled = enabled
}

func (mo *MemoryOptimizer) SetThreshold(threshold int64) {
	mo.mu.Lock()
	defer mo.mu.Unlock()
	mo.threshold = threshold
}

func (mo *MemoryOptimizer) SetGCInterval(interval time.Duration) {
	mo.mu.Lock()
	defer mo.mu.Unlock()
	mo.gcInterval = interval
}

func (mo *MemoryOptimizer) TriggerGC() {
	runtime.GC()
	mo.stats.Optimizations.Add(1)
}

type GCOptimizer struct {
	enabled       bool
	threshold     int64
	gcPercent     int32
	autoTuning    bool
	lastThreshold int64
	stats         *GCStats
}

type GCStats struct {
	GCCount      atomic.Int64
	GCTime       atomic.Int64
	MemorySaved  atomic.Int64
	LastGCTime   time.Time
}

func NewGCOptimizer(threshold int64) *GCOptimizer {
	return &GCOptimizer{
		enabled:    true,
		threshold: threshold,
		gcPercent: 100,
		autoTuning: true,
		stats:     &GCStats{},
	}
}

func (gco *GCOptimizer) Start() {
	if gco.autoTuning {
		go gco.tune()
	}
}

func (gco *GCOptimizer) tune() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !gco.enabled {
			continue
		}

		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		if memStats.PauseNs[(numGCGenerations+1)%256] > 10*1000000 {
			gco.adjustGCPercent(-10)
		}

		if int64(memStats.Alloc) > gco.threshold*2 {
			gco.adjustGCPercent(20)
		}
	}
}

const numGCGenerations = 5

func (gco *GCOptimizer) adjustGCPercent(delta int32) {
	newPercent := gco.gcPercent + delta
	if newPercent < 50 {
		newPercent = 50
	}
	if newPercent > 500 {
		newPercent = 500
	}

	gco.gcPercent = newPercent
}

func (gco *GCOptimizer) GetStats() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"gc_percent":    gco.gcPercent,
		"gc_count":      memStats.NumGC,
		"last_gc_time":  time.Unix(0, int64(memStats.LastGC)),
		"next_gc":       memStats.NextGC,
		"enabled":       gco.enabled,
		"auto_tuning":   gco.autoTuning,
	}
}

type MemoryAllocator struct {
	sizeClasses []int
	pools       map[int]*ObjectPool[[]byte]
	mu          sync.RWMutex
}

func NewMemoryAllocator(sizeClasses []int) *MemoryAllocator {
	if len(sizeClasses) == 0 {
		sizeClasses = []int{64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384}
	}

	ma := &MemoryAllocator{
		sizeClasses: sizeClasses,
		pools:       make(map[int]*ObjectPool[[]byte]),
	}

	for _, size := range sizeClasses {
		allocSize := size
		ma.pools[size] = NewObjectPool[[]byte](
			func() []byte {
				return make([]byte, allocSize)
			},
			func(b []byte) {
			},
			1000,
		)
	}

	return ma
}

func (ma *MemoryAllocator) Allocate(size int) []byte {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	bestSize := ma.findBestSize(size)

	if pool, exists := ma.pools[bestSize]; exists {
		slice := pool.Get()
		if cap(slice) >= size {
			return slice[:size]
		}
	}

	return make([]byte, size)
}

func (ma *MemoryAllocator) Release(buf []byte) {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	size := cap(buf)
	bestSize := ma.findBestSize(size)

	if pool, exists := ma.pools[bestSize]; exists {
		if cap(buf) == bestSize {
			pool.Put(buf)
		}
	}
}

func (ma *MemoryAllocator) findBestSize(size int) int {
	bestSize := ma.sizeClasses[0]

	for _, s := range ma.sizeClasses {
		if s >= size {
			return s
		}
		bestSize = s
	}

	return bestSize
}

type PooledBuffer struct {
	pool *BufferPool
	buf  *bytes.Buffer
}

func NewPooledBuffer(pool *BufferPool) *PooledBuffer {
	return &PooledBuffer{
		pool: pool,
		buf:  pool.Get(),
	}
}

func (pb *PooledBuffer) Write(p []byte) (int, error) {
	return pb.buf.Write(p)
}

func (pb *PooledBuffer) WriteString(s string) (int, error) {
	return pb.buf.WriteString(s)
}

func (pb *PooledBuffer) String() string {
	return pb.buf.String()
}

func (pb *PooledBuffer) Bytes() []byte {
	return pb.buf.Bytes()
}

func (pb *PooledBuffer) Len() int {
	return pb.buf.Len()
}

func (pb *PooledBuffer) Release() {
	pb.pool.Put(pb.buf)
	pb.buf = nil
	pb.pool = nil
}

type EncoderPool struct {
	*ObjectPool[*EncoderWrapper]
}

type EncoderWrapper struct {
	Encoder *gob.Encoder
}

func NewEncoderPool(size int) *EncoderPool {
	pool := NewObjectPool(
		func() *EncoderWrapper {
			var buf bytes.Buffer
			return &EncoderWrapper{
				Encoder: gob.NewEncoder(&buf),
			}
		},
		func(e *EncoderWrapper) {
		},
		size,
	)

	return &EncoderPool{ObjectPool: pool}
}

func (ep *EncoderPool) Get() *EncoderWrapper {
	return ep.ObjectPool.Get()
}

func (ep *EncoderPool) Put(e *EncoderWrapper) {
	ep.ObjectPool.Put(e)
}
