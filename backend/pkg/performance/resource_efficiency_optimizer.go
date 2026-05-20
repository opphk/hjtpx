package performance

import (
	"context"
	"log"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type ResourceEfficiencyOptimizer struct {
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	isRunning   bool

	// 内存池
	memoryPool *MemoryPool

	// 对象复用
	objectPool *ObjectPool

	// GC 优化
	gcOptimizer *GCOptimizer

	// 绿色计算优化
	greenCompute *GreenComputeOptimizer

	// 统计信息
	stats *ResourceEfficiencyStats
}

type MemoryPool struct {
	mu         sync.Mutex
	pools      map[int]*PooledSlice
	poolSizes  []int
}

type PooledSlice struct {
	pool   chan []byte
	size   int
}

type ObjectPool struct {
	mu         sync.RWMutex
	pools      map[string]*GenericPool
}

type GenericPool struct {
	pool       chan interface{}
	newFn      func() interface{}
	resetFn    func(interface{})
}

type GCOptimizer struct {
	mu                sync.RWMutex
	gcThreshold       float64
	lastGC            time.Time
	gcInterval        time.Duration
	forceGCCount      atomic.Int64
}

type GreenComputeOptimizer struct {
	mu               sync.RWMutex
	lowPowerMode     bool
	currentFrequency float64
	loadHistory      []float64
	lastAdjustment   time.Time
}

type ResourceEfficiencyStats struct {
	MemoryAllocations  atomic.Int64
	MemoryReuses       atomic.Int64
	ObjectAllocations  atomic.Int64
	ObjectReuses       atomic.Int64
	GCForcedCount      atomic.Int64
	PowerSavingMode    atomic.Bool
	LastUpdate         atomic.Value
}

func NewResourceEfficiencyOptimizer() *ResourceEfficiencyOptimizer {
	ctx, cancel := context.WithCancel(context.Background())
	return &ResourceEfficiencyOptimizer{
		ctx:            ctx,
		cancel:         cancel,
		memoryPool:     NewMemoryPool(),
		objectPool:     NewObjectPool(),
		gcOptimizer:    NewGCOptimizer(0.7, 5*time.Minute),
		greenCompute:   NewGreenComputeOptimizer(),
		stats:          &ResourceEfficiencyStats{},
	}
}

func NewMemoryPool() *MemoryPool {
	sizes := []int{64, 256, 1024, 4096, 16384, 65536}
	pools := make(map[int]*PooledSlice)
	for _, size := range sizes {
		pools[size] = NewPooledSlice(size, 1000)
	}
	return &MemoryPool{
		pools:     pools,
		poolSizes: sizes,
	}
}

func NewPooledSlice(size, count int) *PooledSlice {
	pool := make(chan []byte, count)
	for i := 0; i < count; i++ {
		pool <- make([]byte, 0, size)
	}
	return &PooledSlice{
		pool: pool,
		size: size,
	}
}

func NewObjectPool() *ObjectPool {
	return &ObjectPool{
		pools: make(map[string]*GenericPool),
	}
}

func NewGCOptimizer(threshold float64, interval time.Duration) *GCOptimizer {
	return &GCOptimizer{
		gcThreshold: threshold,
		gcInterval:  interval,
	}
}

func NewGreenComputeOptimizer() *GreenComputeOptimizer {
	return &GreenComputeOptimizer{
		currentFrequency: 1.0,
		loadHistory:      make([]float64, 0, 60),
	}
}

func (r *ResourceEfficiencyOptimizer) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isRunning {
		return nil
	}
	r.isRunning = true

	go r.optimizeMemory()
	go r.adaptiveGC()
	go r.optimizeGreenCompute()

	log.Println("[ResourceEfficiencyOptimizer] Started successfully")
	return nil
}

func (r *ResourceEfficiencyOptimizer) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isRunning {
		return
	}
	r.isRunning = false
	r.cancel()

	log.Println("[ResourceEfficiencyOptimizer] Stopped")
}

func (r *ResourceEfficiencyOptimizer) GetMemory(size int) []byte {
	buf := r.memoryPool.get(size)
	if buf != nil {
		r.stats.MemoryReuses.Add(1)
		return buf
	}
	r.stats.MemoryAllocations.Add(1)
	return make([]byte, 0, size)
}

func (r *ResourceEfficiencyOptimizer) ReleaseMemory(buf []byte) {
	r.memoryPool.put(buf)
}

func (r *ResourceEfficiencyOptimizer) RegisterObjectPool(name string, newFn func() interface{}, resetFn func(interface{})) {
	r.objectPool.register(name, newFn, resetFn, 100)
}

func (r *ResourceEfficiencyOptimizer) GetObject(name string) interface{} {
	obj := r.objectPool.get(name)
	if obj != nil {
		r.stats.ObjectReuses.Add(1)
		return obj
	}
	r.stats.ObjectAllocations.Add(1)
	return nil
}

func (r *ResourceEfficiencyOptimizer) ReleaseObject(name string, obj interface{}) {
	r.objectPool.put(name, obj)
}

func (r *ResourceEfficiencyOptimizer) ForceGC() {
	r.gcOptimizer.forceGC()
	r.stats.GCForcedCount.Add(1)
}

func (r *ResourceEfficiencyOptimizer) SetLowPowerMode(enabled bool) {
	r.greenCompute.setLowPowerMode(enabled)
	r.stats.PowerSavingMode.Store(enabled)
}

func (r *ResourceEfficiencyOptimizer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"memory_allocations": r.stats.MemoryAllocations.Load(),
		"memory_reuses":      r.stats.MemoryReuses.Load(),
		"object_allocations": r.stats.ObjectAllocations.Load(),
		"object_reuses":      r.stats.ObjectReuses.Load(),
		"gc_forced_count":    r.stats.GCForcedCount.Load(),
		"power_saving_mode":  r.stats.PowerSavingMode.Load(),
		"last_update":        r.stats.LastUpdate.Load(),
	}
}

func (r *ResourceEfficiencyOptimizer) optimizeMemory() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.stats.LastUpdate.Store(time.Now())
		}
	}
}

func (r *ResourceEfficiencyOptimizer) adaptiveGC() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.gcOptimizer.checkAndRun()
		}
	}
}

func (r *ResourceEfficiencyOptimizer) optimizeGreenCompute() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.greenCompute.adjustResources()
		}
	}
}

func (mp *MemoryPool) get(size int) []byte {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	actualSize := mp.findNearestSize(size)
	if pool, exists := mp.pools[actualSize]; exists {
		select {
		case buf := <-pool.pool:
			return buf[:0]
		default:
		}
	}
	return nil
}

func (mp *MemoryPool) put(buf []byte) {
	if buf == nil {
		return
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	capacity := cap(buf)
	if pool, exists := mp.pools[capacity]; exists {
		select {
		case pool.pool <- buf:
		default:
		}
	}
}

func (mp *MemoryPool) findNearestSize(size int) int {
	for _, s := range mp.poolSizes {
		if s >= size {
			return s
		}
	}
	return mp.poolSizes[len(mp.poolSizes)-1]
}

func (op *ObjectPool) register(name string, newFn func() interface{}, resetFn func(interface{}), size int) {
	op.mu.Lock()
	defer op.mu.Unlock()

	pool := &GenericPool{
		pool:    make(chan interface{}, size),
		newFn:   newFn,
		resetFn: resetFn,
	}

	for i := 0; i < size/2; i++ {
		pool.pool <- newFn()
	}

	op.pools[name] = pool
}

func (op *ObjectPool) get(name string) interface{} {
	op.mu.RLock()
	pool, exists := op.pools[name]
	op.mu.RUnlock()

	if !exists {
		return nil
	}

	select {
	case obj := <-pool.pool:
		if pool.resetFn != nil {
			pool.resetFn(obj)
		}
		return obj
	default:
		if pool.newFn != nil {
			return pool.newFn()
		}
		return nil
	}
}

func (op *ObjectPool) put(name string, obj interface{}) {
	op.mu.RLock()
	pool, exists := op.pools[name]
	op.mu.RUnlock()

	if !exists || obj == nil {
		return
	}

	select {
	case pool.pool <- obj:
	default:
	}
}

func (gco *GCOptimizer) checkAndRun() {
	gco.mu.RLock()
	defer gco.mu.RUnlock()

	if time.Since(gco.lastGC) < gco.gcInterval {
		return
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	heapUsage := float64(memStats.HeapAlloc) / float64(memStats.HeapSys)
	if heapUsage > gco.gcThreshold {
		gco.forceGC()
	}
}

func (gco *GCOptimizer) forceGC() {
	gco.mu.Lock()
	defer gco.mu.Unlock()

	debug.FreeOSMemory()
	gco.lastGC = time.Now()
	gco.forceGCCount.Add(1)
	log.Println("[GCOptimizer] Forced GC executed")
}

func (gco *GreenComputeOptimizer) setLowPowerMode(enabled bool) {
	gco.mu.Lock()
	defer gco.mu.Unlock()
	gco.lowPowerMode = enabled
}

func (gco *GreenComputeOptimizer) adjustResources() {
	gco.mu.Lock()
	defer gco.mu.Unlock()

	if time.Since(gco.lastAdjustment) < 10*time.Second {
		return
	}

	numGoroutines := float64(runtime.NumGoroutine())
	gco.loadHistory = append(gco.loadHistory, numGoroutines)
	if len(gco.loadHistory) > 60 {
		gco.loadHistory = gco.loadHistory[1:]
	}

	avgLoad := gco.calculateAverageLoad()
	gco.adjustFrequency(avgLoad)
	gco.lastAdjustment = time.Now()
}

func (gco *GreenComputeOptimizer) calculateAverageLoad() float64 {
	if len(gco.loadHistory) == 0 {
		return 0
	}

	sum := 0.0
	for _, load := range gco.loadHistory {
		sum += load
	}
	return sum / float64(len(gco.loadHistory))
}

func (gco *GreenComputeOptimizer) adjustFrequency(avgLoad float64) {
	targetFreq := 1.0
	if gco.lowPowerMode {
		targetFreq = 0.5
	} else if avgLoad < 100 {
		targetFreq = 0.7
	} else if avgLoad > 1000 {
		targetFreq = 1.0
	}

	if targetFreq != gco.currentFrequency {
		gco.currentFrequency = targetFreq
		log.Printf("[GreenComputeOptimizer] Adjusted frequency to %.2f", targetFreq)
	}
}
