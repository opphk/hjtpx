package service

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type OptimizedMemoryPool struct {
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	pools        map[int]*SizeClassPool
	sizes        []int
	stats        *PoolStats
	gcController *GCController
	enableMetrics bool
}

type SizeClassPool struct {
	size       int
	spanSize   int
	pool       chan []byte
	allocCount int64
	freeCount  int64
	hits       int64
	misses     int64
	mu         sync.Mutex
}

type PoolStats struct {
	TotalAllocations  atomic.Int64
	TotalFrees        atomic.Int64
	PoolHits          atomic.Int64
	PoolMisses        atomic.Int64
	CurrentMemory     atomic.Int64
	PeakMemory        atomic.Int64
	GCCycles          atomic.Int64
	AvgAllocTime      atomic.Int64
	LastUpdate        atomic.Value
}

type GCController struct {
	mu               sync.RWMutex
	enabled          bool
	threshold        uint64
	interval         time.Duration
	lastGCTime       time.Time
	gcCount          int
	memoryBeforeGC   uint64
	memoryAfterGC    uint64
}

type ObjectPool interface {
	Get() interface{}
	Put(interface{})
}

type TypedObjectPool[T any] struct {
	pool    sync.Pool
	factory func() T
	reset   func(T)
	stats   *ObjectPoolStats
}

type ObjectPoolStats struct {
	TotalGets   atomic.Int64
	TotalPuts   atomic.Int64
	PoolCreates atomic.Int64
	Hits        atomic.Int64
	Misses      atomic.Int64
}

func NewOptimizedMemoryPool(ctx context.Context) *OptimizedMemoryPool {
	if ctx == nil {
		ctx = context.Background()
	}
	childCtx, cancel := context.WithCancel(ctx)

	sizes := []int{32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536}

	pools := make(map[int]*SizeClassPool)
	for _, size := range sizes {
		pools[size] = newSizeClassPool(size, 1024)
	}

	return &OptimizedMemoryPool{
		ctx:           childCtx,
		cancel:        cancel,
		pools:         pools,
		sizes:         sizes,
		stats:         &PoolStats{},
		gcController:  newGCController(),
		enableMetrics: true,
	}
}

func newSizeClassPool(size, capacity int) *SizeClassPool {
	return &SizeClassPool{
		size:     size,
		spanSize: (size + 15) &^ 15,
		pool:     make(chan []byte, capacity),
	}
}

func newGCController() *GCController {
	return &GCController{
		enabled:   true,
		threshold: 100 * 1024 * 1024,
		interval:  30 * time.Second,
	}
}

func (p *OptimizedMemoryPool) Start() {
	go p.monitor()
	go p.gcController.run(p.ctx)
}

func (p *OptimizedMemoryPool) Stop() {
	p.cancel()
}

func (p *OptimizedMemoryPool) Get(size int) []byte {
	start := time.Now()
	p.stats.TotalAllocations.Add(1)

	selectedSize := p.findBestSize(size)

	p.mu.RLock()
	pool, exists := p.pools[selectedSize]
	p.mu.RUnlock()

	if !exists {
		p.stats.PoolMisses.Add(1)
		atomic.AddInt64(&p.stats.CurrentMemory, int64(size))
		return make([]byte, size)
	}

	pool.mu.Lock()
	select {
	case buf := <-pool.pool:
		pool.hits++
		pool.allocCount++
		pool.mu.Unlock()
		p.stats.PoolHits.Add(1)
		if cap(buf) >= size {
			atomic.AddInt64(&p.stats.CurrentMemory, int64(cap(buf)))
			return buf[:size]
		}
	default:
		pool.misses++
		pool.allocCount++
		pool.mu.Unlock()
		p.stats.PoolMisses.Add(1)
		atomic.AddInt64(&p.stats.CurrentMemory, int64(selectedSize))
		return make([]byte, size, selectedSize)
	}

	elapsed := time.Since(start).Nanoseconds()
	oldAvg := p.stats.AvgAllocTime.Load()
	count := p.stats.TotalAllocations.Load()
	if count > 0 {
		p.stats.AvgAllocTime.Store((oldAvg*(count-1) + elapsed) / count)
	}
	p.stats.LastUpdate.Store(time.Now())
}

func (p *OptimizedMemoryPool) Put(buf []byte) {
	if buf == nil {
		return
	}

	size := cap(buf)
	if size == 0 {
		return
	}

	p.stats.TotalFrees.Add(1)

	selectedSize := p.findBestSize(size)

	p.mu.RLock()
	pool, exists := p.pools[selectedSize]
	p.mu.RUnlock()

	if !exists || cap(buf) < selectedSize {
		atomic.AddInt64(&p.stats.CurrentMemory, -int64(cap(buf)))
		return
	}

	pool.mu.Lock()
	select {
	case pool.pool <- buf[:0]:
		pool.freeCount++
		pool.mu.Unlock()
		atomic.AddInt64(&p.stats.CurrentMemory, -int64(cap(buf)))
	default:
		pool.mu.Unlock()
		atomic.AddInt64(&p.stats.CurrentMemory, -int64(cap(buf)))
	}
}

func (p *OptimizedMemoryPool) findBestSize(requested int) int {
	for _, size := range p.sizes {
		if size >= requested {
			return size
		}
	}
	return p.sizes[len(p.sizes)-1] * 2
}

func (p *OptimizedMemoryPool) monitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastMemory int64

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			current := p.stats.CurrentMemory.Load()
			if current > p.stats.PeakMemory.Load() {
				p.stats.PeakMemory.Store(current)
			}

			delta := current - lastMemory
			if delta > int64(p.gcController.threshold) || delta < -int64(p.gcController.threshold) {
				runtime.GC()
				p.stats.GCCycles.Add(1)
			}
			lastMemory = current
		}
	}
}

func (gc *GCController) run(ctx context.Context) {
	if !gc.enabled {
		return
	}

	ticker := time.NewTicker(gc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			gc.triggerGC()
		}
	}
}

func (gc *GCController) triggerGC() {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if time.Since(gc.lastGCTime) < gc.interval {
		return
	}

	var mBefore runtime.MemStats
	runtime.ReadMemStats(&mBefore)
	gc.memoryBeforeGC = mBefore.Alloc

	runtime.GC()

	var mAfter runtime.MemStats
	runtime.ReadMemStats(&mAfter)
	gc.memoryAfterGC = mAfter.Alloc
	gc.lastGCTime = time.Now()
	gc.gcCount++
}

func (p *OptimizedMemoryPool) GetStats() map[string]interface{} {
	poolStats := make(map[string]interface{})
	for size, pool := range p.pools {
		pool.mu.Lock()
		poolStats[fmt.Sprintf("size_%d", size)] = map[string]interface{}{
			"hits":  pool.hits,
			"misses": pool.misses,
			"allocs": pool.allocCount,
			"frees":  pool.freeCount,
		}
		pool.mu.Unlock()
	}

	gc.mu.RLock()
	gcStats := map[string]interface{}{
		"enabled":         gc.enabled,
		"threshold":       gc.threshold,
		"interval":        gc.interval,
		"gc_count":        gc.gcCount,
		"memory_before":   gc.memoryBeforeGC,
		"memory_after":    gc.memoryAfterGC,
		"memory_saved":    gc.memoryBeforeGC - gc.memoryAfterGC,
		"last_gc_time":    gc.lastGCTime,
	}
	gc.mu.RUnlock()

	return map[string]interface{}{
		"total_allocations": p.stats.TotalAllocations.Load(),
		"total_frees":       p.stats.TotalFrees.Load(),
		"pool_hits":         p.stats.PoolHits.Load(),
		"pool_misses":       p.stats.PoolMisses.Load(),
		"current_memory":    p.stats.CurrentMemory.Load(),
		"peak_memory":       p.stats.PeakMemory.Load(),
		"gc_cycles":         p.stats.GCCycles.Load(),
		"avg_alloc_time_ns": p.stats.AvgAllocTime.Load(),
		"pools":             poolStats,
		"gc_controller":     gcStats,
		"last_update":       p.stats.LastUpdate.Load(),
	}
}

func NewTypedObjectPool[T any](factory func() T, reset func(T)) *TypedObjectPool[T] {
	return &TypedObjectPool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return factory()
			},
		},
		factory: factory,
		reset:   reset,
		stats:   &ObjectPoolStats{},
	}
}

func (p *TypedObjectPool[T]) Get() T {
	p.stats.TotalGets.Add(1)

	item := p.pool.Get()
	if item != nil {
		p.stats.Hits.Add(1)
		return item.(T)
	}

	p.stats.Misses.Add(1)
	p.stats.PoolCreates.Add(1)
	return p.factory()
}

func (p *TypedObjectPool[T]) Put(item T) {
	p.stats.TotalPuts.Add(1)

	if p.reset != nil {
		p.reset(item)
	}
	p.pool.Put(item)
}

func (p *TypedObjectPool[T]) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_gets":    p.stats.TotalGets.Load(),
		"total_puts":    p.stats.TotalPuts.Load(),
		"pool_creates":  p.stats.PoolCreates.Load(),
		"hits":          p.stats.Hits.Load(),
		"misses":        p.stats.Misses.Load(),
		"hit_rate":      p.calculateHitRate(),
	}
}

func (p *TypedObjectPool[T]) calculateHitRate() float64 {
	gets := p.stats.TotalGets.Load()
	if gets == 0 {
		return 0
	}
	return float64(p.stats.Hits.Load()) / float64(gets)
}

type PoolConfig struct {
	InitialSize   int
	MaxSize       int
	MinSize       int
	Capacity      int
	EnableMetrics bool
}

func NewObjectPoolWithConfig[T any](factory func() T, reset func(T), config PoolConfig) *TypedObjectPool[T] {
	pool := &TypedObjectPool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return factory()
			},
		},
		factory: factory,
		reset:   reset,
		stats:   &ObjectPoolStats{},
	}

	return pool
}

type SlabPool struct {
	mu       sync.RWMutex
	slabs    map[int]*Slab
	minSize  int
	maxSize  int
	pageSize int
}

type Slab struct {
	size    int
	chunks  chan []byte
	alloced atomic.Int32
	freed   atomic.Int32
}

func NewSlabPool(minSize, maxSize, pageSize int) *SlabPool {
	return &SlabPool{
		slabs:    make(map[int]*Slab),
		minSize:  minSize,
		maxSize:  maxSize,
		pageSize: pageSize,
	}
}

func (s *SlabPool) Allocate(size int) []byte {
	slabSize := s.findSlabSize(size)

	s.mu.RLock()
	slab, exists := s.slabs[slabSize]
	s.mu.RUnlock()

	if !exists {
		s.mu.Lock()
		if _, exists = s.slabs[slabSize]; !exists {
			slab = &Slab{
				size:   slabSize,
				chunks: make(chan []byte, s.pageSize/slabSize),
			}
			s.slabs[slabSize] = slab
		} else {
			slab = s.slabs[slabSize]
		}
		s.mu.Unlock()
	}

	select {
	case chunk := <-slab.chunks:
		slab.alloced.Add(1)
		return chunk[:size]
	default:
		slab.alloced.Add(1)
		return make([]byte, size, slabSize)
	}
}

func (s *SlabPool) Release(buf []byte) {
	size := cap(buf)
	slabSize := s.findSlabSize(size)

	s.mu.RLock()
	slab, exists := s.slabs[slabSize]
	s.mu.RUnlock()

	if !exists || cap(buf) < slabSize {
		return
	}

	select {
	case slab.chunks <- buf[:0]:
		slab.freed.Add(1)
	default:
	}
}

func (s *SlabPool) findSlabSize(size int) int {
	slabSize := s.minSize
	for slabSize < s.maxSize {
		if slabSize >= size {
			return slabSize
		}
		slabSize *= 2
	}
	return s.maxSize
}

func (s *SlabPool) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	slabStats := make(map[string]interface{})
	for size, slab := range s.slabs {
		slabStats[fmt.Sprintf("size_%d", size)] = map[string]interface{}{
			"allocated": slab.alloced.Load(),
			"freed":     slab.freed.Load(),
			"available": len(slab.chunks),
		}
	}

	return map[string]interface{}{
		"slabs": slabStats,
	}
}

var (
	globalOptimizedPool *OptimizedMemoryPool
	poolOnce            sync.Once
)

func GetOptimizedMemoryPool() *OptimizedMemoryPool {
	poolOnce.Do(func() {
		globalOptimizedPool = NewOptimizedMemoryPool(nil)
		globalOptimizedPool.Start()
	})
	return globalOptimizedPool
}
