package performance

import (
	"context"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	TargetQPS          = 15000
	DefaultConcurrency = 1000
	DefaultQueueSize   = 50000
)

type PerformanceEngine struct {
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	isRunning bool

	// 核心组件
	dbOptimizer     *DatabaseOptimizer
	cacheOptimizer  *CacheOptimizer
	concurrencyMgr  *ConcurrencyManager
	resourceMgr     *ResourceManager
	edgeCompute     *EdgeCompute
	wasmEngine      *WASMEngine

	// 性能指标
	metrics *EngineMetrics
}

type EngineMetrics struct {
	TotalRequests       atomic.Int64
	SuccessfulRequests  atomic.Int64
	FailedRequests      atomic.Int64
	CurrentQPS          atomic.Int64
	PeakQPS             atomic.Int64
	AvgLatency          atomic.Int64
	P95Latency          atomic.Int64
	P99Latency          atomic.Int64
	MemoryUsage         atomic.Int64
	ActiveGoroutines    atomic.Int64
	DBConnections       atomic.Int64
	CacheHitRate        atomic.Int64
	LastUpdate          atomic.Value
}

func NewPerformanceEngine() *PerformanceEngine {
	ctx, cancel := context.WithCancel(context.Background())

	engine := &PerformanceEngine{
		ctx:            ctx,
		cancel:         cancel,
		dbOptimizer:    NewDatabaseOptimizer(),
		cacheOptimizer: NewCacheOptimizer(),
		concurrencyMgr: NewConcurrencyManager(),
		resourceMgr:    NewResourceManager(),
		edgeCompute:    NewEdgeCompute(),
		wasmEngine:     NewWASMEngine(),
		metrics:        &EngineMetrics{},
	}

	engine.metrics.LastUpdate.Store(time.Now())
	return engine
}

func (e *PerformanceEngine) Start(ctx ...context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.isRunning {
		return nil
	}

	e.isRunning = true

	log.Println("[PerformanceEngine] Starting all optimization components...")

	if err := e.dbOptimizer.Start(e.ctx); err != nil {
		log.Printf("[PerformanceEngine] Database optimizer failed to start: %v", err)
	}

	if err := e.cacheOptimizer.Start(e.ctx); err != nil {
		log.Printf("[PerformanceEngine] Cache optimizer failed to start: %v", err)
	}

	if err := e.concurrencyMgr.Start(e.ctx); err != nil {
		log.Printf("[PerformanceEngine] Concurrency manager failed to start: %v", err)
	}

	if err := e.resourceMgr.Start(e.ctx); err != nil {
		log.Printf("[PerformanceEngine] Resource manager failed to start: %v", err)
	}

	if err := e.edgeCompute.Start(e.ctx); err != nil {
		log.Printf("[PerformanceEngine] Edge compute failed to start: %v", err)
	}

	if err := e.wasmEngine.Start(e.ctx); err != nil {
		log.Printf("[PerformanceEngine] WASM engine failed to start: %v", err)
	}

	go e.monitorMetrics()
	go e.adaptiveOptimization()

	log.Println("[PerformanceEngine] All optimization components started successfully")
	return nil
}

func (e *PerformanceEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning {
		return
	}

	e.cancel()
	e.isRunning = false

	log.Println("[PerformanceEngine] Stopping all optimization components...")

	e.dbOptimizer.Stop()
	e.cacheOptimizer.Stop()
	e.concurrencyMgr.Stop()
	e.resourceMgr.Stop()
	e.edgeCompute.Stop()
	e.wasmEngine.Stop()

	log.Println("[PerformanceEngine] All optimization components stopped")
}

func (e *PerformanceEngine) monitorMetrics() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	requestWindow := make([]int64, 0, 10)
	lastRequestCount := int64(0)

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			currentRequests := e.metrics.TotalRequests.Load()
			qps := currentRequests - lastRequestCount
			lastRequestCount = currentRequests

			requestWindow = append(requestWindow, qps)
			if len(requestWindow) > 10 {
				requestWindow = requestWindow[1:]
			}

			var avgQPS int64
			for _, r := range requestWindow {
				avgQPS += r
			}
			avgQPS /= int64(len(requestWindow))

			e.metrics.CurrentQPS.Store(avgQPS)
			if avgQPS > e.metrics.PeakQPS.Load() {
				e.metrics.PeakQPS.Store(avgQPS)
			}

			e.metrics.MemoryUsage.Store(getMemoryUsage())
			e.metrics.ActiveGoroutines.Store(int64(runtime.NumGoroutine()))
			e.metrics.LastUpdate.Store(time.Now())

			if avgQPS > TargetQPS*90/100 {
				log.Printf("[PerformanceEngine] High QPS detected: %d (target: %d)", avgQPS, TargetQPS)
			}
		}
	}
}

func (e *PerformanceEngine) adaptiveOptimization() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			currentQPS := e.metrics.CurrentQPS.Load()
			memoryUsage := e.metrics.MemoryUsage.Load()
			activeGoroutines := e.metrics.ActiveGoroutines.Load()

			if currentQPS < TargetQPS*30/100 {
				e.resourceMgr.ScaleDown()
			} else if currentQPS > TargetQPS*80/100 {
				e.resourceMgr.ScaleUp()
			}

			if memoryUsage > 80*1024*1024*1024 { // 80GB
				runtime.GC()
			}

			if activeGoroutines > 50000 {
				e.concurrencyMgr.LimitGoroutines(50000)
			}
		}
	}
}

func (e *PerformanceEngine) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_requests":       e.metrics.TotalRequests.Load(),
		"successful_requests":  e.metrics.SuccessfulRequests.Load(),
		"failed_requests":      e.metrics.FailedRequests.Load(),
		"current_qps":          e.metrics.CurrentQPS.Load(),
		"peak_qps":             e.metrics.PeakQPS.Load(),
		"target_qps":           TargetQPS,
		"avg_latency_ns":       e.metrics.AvgLatency.Load(),
		"memory_usage_bytes":   e.metrics.MemoryUsage.Load(),
		"active_goroutines":    e.metrics.ActiveGoroutines.Load(),
		"db_connections":       e.metrics.DBConnections.Load(),
		"cache_hit_rate":       e.metrics.CacheHitRate.Load(),
		"last_update":          e.metrics.LastUpdate.Load(),
	}
}

func (e *PerformanceEngine) GetStats() map[string]interface{} {
	return e.GetMetrics()
}

func (e *PerformanceEngine) RecordRequest(success bool, latency time.Duration) {
	e.metrics.TotalRequests.Add(1)
	if success {
		e.metrics.SuccessfulRequests.Add(1)
	} else {
		e.metrics.FailedRequests.Add(1)
	}

	latencyNs := latency.Nanoseconds()
	old := e.metrics.AvgLatency.Load()
	count := e.metrics.SuccessfulRequests.Load()
	if count > 0 {
		newAvg := (old*(count-1) + latencyNs) / count
		e.metrics.AvgLatency.Store(newAvg)
	}
}

func (e *PerformanceEngine) GetDatabaseOptimizer() *DatabaseOptimizer {
	return e.dbOptimizer
}

func (e *PerformanceEngine) GetCacheOptimizer() *CacheOptimizer {
	return e.cacheOptimizer
}

func (e *PerformanceEngine) GetConcurrencyManager() *ConcurrencyManager {
	return e.concurrencyMgr
}

func (e *PerformanceEngine) GetResourceManager() *ResourceManager {
	return e.resourceMgr
}

func (e *PerformanceEngine) GetEdgeCompute() *EdgeCompute {
	return e.edgeCompute
}

func (e *PerformanceEngine) GetWASMEngine() *WASMEngine {
	return e.wasmEngine
}

func getMemoryUsage() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Alloc)
}

var (
	globalEngine *PerformanceEngine
	engineOnce   sync.Once
)

func GetEngine() *PerformanceEngine {
	engineOnce.Do(func() {
		globalEngine = NewPerformanceEngine()
	})
	return globalEngine
}

func InitEngine() error {
	return GetEngine().Start()
}

func StopEngine() {
	if globalEngine != nil {
		globalEngine.Stop()
	}
}
