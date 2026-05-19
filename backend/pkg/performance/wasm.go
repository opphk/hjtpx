package performance

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type WASMEngine struct {
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	isRunning    bool
	modules      map[string]*WASMModule
	pool         *ModulePool
	compiler     *WASMCompiler
	stats        *WASMStats
}

type WASMModule struct {
	ID           string
	Data         []byte
	LoadedAt     time.Time
	LastUsed     time.Time
	UseCount     int64
}

type ModulePool struct {
	pool        sync.Pool
	maxSize     int
	currentSize int
	mu          sync.Mutex
}

type WASMCompiler struct {
	mu           sync.Mutex
	cache        map[string][]byte
	cacheEnabled bool
}

type WASMStats struct {
	TotalExecutions atomic.Int64
	CacheHits       atomic.Int64
	CacheMisses     atomic.Int64
	PoolHits        atomic.Int64
	PoolMisses      atomic.Int64
	ActiveModules   atomic.Int64
	TotalMemory     atomic.Int64
	AvgDuration     atomic.Int64
	LastUpdate      atomic.Value
}

func NewWASMEngine() *WASMEngine {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WASMEngine{
		ctx:       ctx,
		cancel:    cancel,
		modules:   make(map[string]*WASMModule),
		pool:      NewModulePool(100),
		compiler:  NewWASMCompiler(),
		stats:     &WASMStats{},
	}
}

func NewModulePool(maxSize int) *ModulePool {
	return &ModulePool{
		maxSize: maxSize,
		pool: sync.Pool{
			New: func() interface{} {
				return &WASMModule{}
			},
		},
	}
}

func NewWASMCompiler() *WASMCompiler {
	return &WASMCompiler{
		cache:        make(map[string][]byte),
		cacheEnabled: true,
	}
}

func (w *WASMEngine) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isRunning {
		return nil
	}

	w.isRunning = true

	go w.cleanupModules()

	log.Println("[WASMEngine] Started successfully")
	return nil
}

func (w *WASMEngine) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isRunning {
		return
	}

	w.cancel()
	w.isRunning = false

	log.Println("[WASMEngine] Stopped")
}

func (w *WASMEngine) LoadModule(id string, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.modules[id]; exists {
		log.Printf("[WASMEngine] Module %s already loaded", id)
		return nil
	}

	module := &WASMModule{
		ID:       id,
		Data:     data,
		LoadedAt: time.Now(),
		LastUsed: time.Now(),
	}

	w.modules[id] = module
	w.stats.ActiveModules.Store(int64(len(w.modules)))
	w.stats.TotalMemory.Add(int64(len(data)))

	log.Printf("[WASMEngine] Loaded module %s (%d bytes)", id, len(data))
	return nil
}

func (w *WASMEngine) UnloadModule(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if module, exists := w.modules[id]; exists {
		w.stats.TotalMemory.Add(-int64(len(module.Data)))
		delete(w.modules, id)
		w.stats.ActiveModules.Store(int64(len(w.modules)))
		log.Printf("[WASMEngine] Unloaded module %s", id)
	}
}

func (w *WASMEngine) ExecuteModule(id string, input []byte) ([]byte, error) {
	start := time.Now()
	w.stats.TotalExecutions.Add(1)

	w.mu.RLock()
	module, exists := w.modules[id]
	w.mu.RUnlock()

	if !exists {
		w.stats.CacheMisses.Add(1)
		return nil, nil
	}

	w.stats.CacheHits.Add(1)
	atomic.AddInt64(&module.UseCount, 1)

	// Try to get from pool
	pooled := w.pool.get()
	if pooled != nil {
		w.stats.PoolHits.Add(1)
	} else {
		w.stats.PoolMisses.Add(1)
		pooled = module
	}

	// Simulate execution
	result := make([]byte, 0)
	for _, b := range input {
		result = append(result, b^0xFF)
	}

	// Update module usage
	w.mu.Lock()
	module.LastUsed = time.Now()
	w.mu.Unlock()

	// Return to pool
	w.pool.put(pooled)

	duration := time.Since(start).Nanoseconds()
	oldAvg := w.stats.AvgDuration.Load()
	count := w.stats.TotalExecutions.Load()
	newAvg := (oldAvg*(count-1) + duration) / count
	w.stats.AvgDuration.Store(newAvg)
	w.stats.LastUpdate.Store(time.Now())

	return result, nil
}

func (w *WASMEngine) ExecuteBatch(id string, inputs [][]byte) ([][]byte, error) {
	results := make([][]byte, len(inputs))
	
	// Use concurrency for batch processing
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	for i, input := range inputs {
		wg.Add(1)
		go func(idx int, in []byte) {
			defer wg.Done()
			result, _ := w.ExecuteModule(id, in)
			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, input)
	}
	
	wg.Wait()
	return results, nil
}

func (w *WASMEngine) cleanupModules() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.mu.Lock()
			
			now := time.Now()
			for id, module := range w.modules {
				if now.Sub(module.LastUsed) > 30*time.Minute && module.UseCount == 0 {
					w.stats.TotalMemory.Add(-int64(len(module.Data)))
					delete(w.modules, id)
					log.Printf("[WASMEngine] Cleaned up module %s (unused)", id)
				}
			}
			
			w.stats.ActiveModules.Store(int64(len(w.modules)))
			w.mu.Unlock()
		}
	}
}

func (w *WASMEngine) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_executions": w.stats.TotalExecutions.Load(),
		"cache_hits":       w.stats.CacheHits.Load(),
		"cache_misses":     w.stats.CacheMisses.Load(),
		"pool_hits":        w.stats.PoolHits.Load(),
		"pool_misses":      w.stats.PoolMisses.Load(),
		"active_modules":   w.stats.ActiveModules.Load(),
		"total_memory":     w.stats.TotalMemory.Load(),
		"avg_duration":     w.stats.AvgDuration.Load(),
		"last_update":      w.stats.LastUpdate.Load(),
	}
}

func (mp *ModulePool) get() *WASMModule {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.currentSize <= 0 {
		return nil
	}

	mp.currentSize--
	module := mp.pool.Get().(*WASMModule)
	return module
}

func (mp *ModulePool) put(module *WASMModule) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.currentSize < mp.maxSize {
		mp.currentSize++
		mp.pool.Put(module)
	}
}

func (wc *WASMCompiler) Compile(source []byte) ([]byte, error) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	key := string(source)
	
	if wc.cacheEnabled {
		if cached, ok := wc.cache[key]; ok {
			return cached, nil
		}
	}

	// Simulate compilation
	result := make([]byte, len(source))
	for i, b := range source {
		result[i] = b + 1
	}

	if wc.cacheEnabled {
		wc.cache[key] = result
	}

	return result, nil
}
