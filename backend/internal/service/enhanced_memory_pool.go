package service

import (
	"sync"
	"sync/atomic"
	"time"
)

type MemoryPoolStats struct {
	TotalAllocations atomic.Int64
	TotalFrees       atomic.Int64
	PoolHits         atomic.Int64
	PoolMisses       atomic.Int64
	CurrentUsage     atomic.Int64
	PeakUsage        atomic.Int64
}

type SizedPool struct {
	pool     *sync.Pool
	size     int
	created  int64
	reused   int64
	mu       sync.Mutex
}

func NewSizedPool(size int) *SizedPool {
	return &SizedPool{
		pool: &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 0, size)
				return &buf
			},
		},
		size: size,
	}
}

func (sp *SizedPool) Get() []byte {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	item := sp.pool.Get()
	if item != nil {
		bufPtr := item.(*[]byte)
		buf := *bufPtr
		if cap(buf) >= sp.size {
			sp.reused++
			return buf[:0]
		}
	}

	sp.created++
	newBuf := make([]byte, 0, sp.size)
	return newBuf
}

func (sp *SizedPool) Put(buf []byte) {
	if cap(buf) < sp.size || cap(buf) > sp.size*2 {
		return
	}

	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.pool.Put(&buf)
}

func (sp *SizedPool) GetStats() (created, reused int64) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.created, sp.reused
}

type EnhancedMemoryPool struct {
	pools         map[int]*SizedPool
	sizes         []int
	mu            sync.RWMutex
	stats         *MemoryPoolStats
	gcThreshold   int64
	lastGCTime    time.Time
	gcInterval    time.Duration
}

func NewEnhancedMemoryPool(initialSizes []int) *EnhancedMemoryPool {
	if len(initialSizes) == 0 {
		initialSizes = []int{64, 256, 1024, 4096, 16384}
	}

	emp := &EnhancedMemoryPool{
		pools:       make(map[int]*SizedPool),
		sizes:       initialSizes,
		stats:       &MemoryPoolStats{},
		gcThreshold: 100 * 1024 * 1024, // 100MB
		gcInterval:  5 * time.Minute,
	}

	for _, size := range initialSizes {
		emp.pools[size] = NewSizedPool(size)
	}

	return emp
}

func (emp *EnhancedMemoryPool) Get(size int) []byte {
	emp.mu.RLock()

	// 找到最适合的大小
	selectedSize := emp.findBestSize(size)
	pool, exists := emp.pools[selectedSize]

	emp.mu.RUnlock()

	if exists {
		buf := pool.Get()
		emp.stats.PoolHits.Add(1)
		emp.stats.CurrentUsage.Add(int64(selectedSize))
		emp.updatePeakUsage()
		return buf[:size]
	}

	emp.stats.PoolMisses.Add(1)
	emp.stats.TotalAllocations.Add(1)
	emp.stats.CurrentUsage.Add(int64(size))
	emp.updatePeakUsage()

	return make([]byte, size)
}

func (emp *EnhancedMemoryPool) Put(buf []byte) {
	if buf == nil {
		return
	}

	size := cap(buf)
	emp.mu.RLock()
	pool, exists := emp.pools[size]
	emp.mu.RUnlock()

	if exists {
		pool.Put(buf)
		emp.stats.TotalFrees.Add(1)
		emp.stats.CurrentUsage.Add(int64(-size))
	}
}

func (emp *EnhancedMemoryPool) findBestSize(requested int) int {
	bestSize := emp.sizes[0]
	for _, size := range emp.sizes {
		if size >= requested {
			return size
		}
		bestSize = size
	}
	return bestSize
}

func (emp *EnhancedMemoryPool) updatePeakUsage() {
	current := emp.stats.CurrentUsage.Load()
	peak := emp.stats.PeakUsage.Load()
	if current > peak {
		emp.stats.PeakUsage.Store(current)
	}
}

func (emp *EnhancedMemoryPool) GetStats() *MemoryPoolStats {
	return emp.stats
}

func (emp *EnhancedMemoryPool) RegisterSize(size int) {
	emp.mu.Lock()
	defer emp.mu.Unlock()

	if _, exists := emp.pools[size]; !exists {
		emp.pools[size] = NewSizedPool(size)
		emp.sizes = append(emp.sizes, size)
	}
}

type BufferPool struct {
	pool *sync.Pool
}

func NewBufferPool() *BufferPool {
	return &BufferPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 1024)
			},
		},
	}
}

func (bp *BufferPool) Get() []byte {
	return bp.pool.Get().([]byte)
}

func (bp *BufferPool) Put(buf []byte) {
	if cap(buf) > 0 && cap(buf) < 1024*10 {
		bp.pool.Put(buf[:0])
	}
}

type ObjectPool[T any] struct {
	pool    *sync.Pool
	factory func() T
	reset   func(T)
}

func NewObjectPool[T any](factory func() T, reset func(T)) *ObjectPool[T] {
	return &ObjectPool[T]{
		pool: &sync.Pool{
			New: func() interface{} {
				return factory()
			},
		},
		factory: factory,
		reset:   reset,
	}
}

func (op *ObjectPool[T]) Get() T {
	return op.pool.Get().(T)
}

func (op *ObjectPool[T]) Put(obj T) {
	if op.reset != nil {
		op.reset(obj)
	}
	op.pool.Put(obj)
}

type ConcurrentMemoryPool struct {
	small  *SizedPool
	medium *SizedPool
	large  *SizedPool
	huge   *SizedPool
	stats  *MemoryPoolStats
}

func NewConcurrentMemoryPool() *ConcurrentMemoryPool {
	return &ConcurrentMemoryPool{
		small:  NewSizedPool(64),
		medium: NewSizedPool(512),
		large:  NewSizedPool(4096),
		huge:   NewSizedPool(32768),
		stats:  &MemoryPoolStats{},
	}
}

func (cmp *ConcurrentMemoryPool) Get(size int) []byte {
	switch {
	case size <= 64:
		buf := cmp.small.Get()
		cmp.stats.PoolHits.Add(1)
		return buf[:size]
	case size <= 512:
		buf := cmp.medium.Get()
		cmp.stats.PoolHits.Add(1)
		return buf[:size]
	case size <= 4096:
		buf := cmp.large.Get()
		cmp.stats.PoolHits.Add(1)
		return buf[:size]
	default:
		buf := cmp.huge.Get()
		if cap(buf) >= size {
			cmp.stats.PoolHits.Add(1)
			return buf[:size]
		}
		cmp.stats.PoolMisses.Add(1)
		return make([]byte, size)
	}
}

func (cmp *ConcurrentMemoryPool) Put(buf []byte) {
	size := cap(buf)
	switch {
	case size <= 64:
		cmp.small.Put(buf)
	case size <= 512:
		cmp.medium.Put(buf)
	case size <= 4096:
		cmp.large.Put(buf)
	case size <= 32768:
		cmp.huge.Put(buf)
	}
}

func (cmp *ConcurrentMemoryPool) GetStats() *MemoryPoolStats {
	return cmp.stats
}

var (
	globalEnhancedPool  *EnhancedMemoryPool
	globalConcurrentPool *ConcurrentMemoryPool
	poolOnce           sync.Once
)

func InitMemoryPools() {
	poolOnce.Do(func() {
		globalEnhancedPool = NewEnhancedMemoryPool(nil)
		globalConcurrentPool = NewConcurrentMemoryPool()
	})
}

func GetEnhancedMemoryPool() *EnhancedMemoryPool {
	if globalEnhancedPool == nil {
		InitMemoryPools()
	}
	return globalEnhancedPool
}

func GetConcurrentMemoryPool() *ConcurrentMemoryPool {
	if globalConcurrentPool == nil {
		InitMemoryPools()
	}
	return globalConcurrentPool
}
