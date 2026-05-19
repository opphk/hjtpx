package performance

import (
	"context"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type ResourceManager struct {
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.RWMutex
	isRunning      bool
	cpuMonitor     *CPUMonitor
	memoryMonitor  *MemoryMonitor
	networkMonitor *NetworkMonitor
	autoscaler     *AutoScaler
	stats          *ResourceStats
}

type CPUMonitor struct {
	mu        sync.RWMutex
	usage     float64
	history   []float64
	threshold float64
}

type MemoryMonitor struct {
	mu        sync.RWMutex
	usage     uint64
	max       uint64
	threshold float64
}

type NetworkMonitor struct {
	mu        sync.RWMutex
	bandwidth int64
	latency   time.Duration
}

type AutoScaler struct {
	mu         sync.RWMutex
	minReplicas int
	maxReplicas int
	currentReplicas int
	targetCPU float64
	cooldown time.Duration
	lastScale time.Time
}

type ResourceStats struct {
	CPUUsage        atomic.Value
	MemoryUsage     atomic.Value
	NetworkTraffic  atomic.Value
	CurrentReplicas atomic.Int64
	ScaleEvents     atomic.Int64
	LastUpdate      atomic.Value
}

func NewResourceManager() *ResourceManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return &ResourceManager{
		ctx:            ctx,
		cancel:         cancel,
		cpuMonitor:     NewCPUMonitor(80.0),
		memoryMonitor:  NewMemoryMonitor(memStats.Sys, 80.0),
		networkMonitor: NewNetworkMonitor(),
		autoscaler:     NewAutoScaler(1, 10, 50.0, 30*time.Second),
		stats:          &ResourceStats{},
	}
}

func NewCPUMonitor(threshold float64) *CPUMonitor {
	return &CPUMonitor{
		history:   make([]float64, 0, 60),
		threshold: threshold,
	}
}

func NewMemoryMonitor(max uint64, threshold float64) *MemoryMonitor {
	return &MemoryMonitor{
		max:       max,
		threshold: threshold,
	}
}

func NewNetworkMonitor() *NetworkMonitor {
	return &NetworkMonitor{}
}

func NewAutoScaler(min, max int, targetCPU float64, cooldown time.Duration) *AutoScaler {
	return &AutoScaler{
		minReplicas: min,
		maxReplicas: max,
		currentReplicas: min,
		targetCPU: targetCPU,
		cooldown: cooldown,
		lastScale: time.Now(),
	}
}

func (r *ResourceManager) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isRunning {
		return nil
	}

	r.isRunning = true

	go r.monitorResources()
	go r.runAutoscaler()

	log.Println("[ResourceManager] Started successfully")
	return nil
}

func (r *ResourceManager) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isRunning {
		return
	}

	r.cancel()
	r.isRunning = false

	log.Println("[ResourceManager] Stopped")
}

func (r *ResourceManager) monitorResources() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.updateStats()
		}
	}
}

func (r *ResourceManager) runAutoscaler() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.checkAndScale()
		}
	}
}

func (r *ResourceManager) updateStats() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	r.cpuMonitor.update()
	r.memoryMonitor.update(memStats.Alloc)
	
	r.stats.CPUUsage.Store(r.cpuMonitor.getUsage())
	r.stats.MemoryUsage.Store(r.memoryMonitor.getUsage())
	r.stats.CurrentReplicas.Store(int64(r.autoscaler.currentReplicas))
	r.stats.LastUpdate.Store(time.Now())
}

func (r *ResourceManager) checkAndScale() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if time.Since(r.autoscaler.lastScale) < r.autoscaler.cooldown {
		return
	}

	cpuUsage := r.cpuMonitor.getAverage(10)
	
	if cpuUsage > r.autoscaler.targetCPU+10 {
		r.scaleUp()
	} else if cpuUsage < r.autoscaler.targetCPU-20 {
		r.scaleDown()
	}
}

func (r *ResourceManager) scaleUp() {
	if r.autoscaler.currentReplicas >= r.autoscaler.maxReplicas {
		return
	}

	r.autoscaler.currentReplicas = min(r.autoscaler.currentReplicas*2, r.autoscaler.maxReplicas)
	r.autoscaler.lastScale = time.Now()
	r.stats.ScaleEvents.Add(1)

	log.Printf("[ResourceManager] Scaled up to %d replicas", r.autoscaler.currentReplicas)
}

func (r *ResourceManager) scaleDown() {
	if r.autoscaler.currentReplicas <= r.autoscaler.minReplicas {
		return
	}

	r.autoscaler.currentReplicas = max(r.autoscaler.currentReplicas/2, r.autoscaler.minReplicas)
	r.autoscaler.lastScale = time.Now()
	r.stats.ScaleEvents.Add(1)

	log.Printf("[ResourceManager] Scaled down to %d replicas", r.autoscaler.currentReplicas)
}

func (r *ResourceManager) ScaleUp() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scaleUp()
}

func (r *ResourceManager) ScaleDown() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scaleDown()
}

func (r *ResourceManager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"cpu_usage":        r.stats.CPUUsage.Load(),
		"memory_usage":     r.stats.MemoryUsage.Load(),
		"current_replicas": r.stats.CurrentReplicas.Load(),
		"scale_events":     r.stats.ScaleEvents.Load(),
		"last_update":      r.stats.LastUpdate.Load(),
	}
}

func (c *CPUMonitor) update() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple CPU usage estimation using GC stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	c.usage = float64(memStats.NumGC) / 10.0 // Simplified
	if c.usage > 100 {
		c.usage = 100
	}

	c.history = append(c.history, c.usage)
	if len(c.history) > 60 {
		c.history = c.history[len(c.history)-60:]
	}
}

func (c *CPUMonitor) getUsage() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.usage
}

func (c *CPUMonitor) getAverage(n int) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.history) == 0 {
		return 0
	}

	start := max(0, len(c.history)-n)
	sum := 0.0
	for i := start; i < len(c.history); i++ {
		sum += c.history[i]
	}

	return sum / float64(len(c.history)-start)
}

func (m *MemoryMonitor) update(usage uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.usage = usage
}

func (m *MemoryMonitor) getUsage() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return float64(m.usage) / float64(m.max) * 100
}

func (n *NetworkMonitor) updateBandwidth(bytes int64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.bandwidth = bytes
}

func (n *NetworkMonitor) updateLatency(latency time.Duration) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.latency = latency
}
