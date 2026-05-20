package performance

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type ColdStartOptimizer struct {
	mu            sync.RWMutex
	enabled       bool
	prewarmEnabled bool
	prewarmDelay  time.Duration
	warmInstances map[string]*WarmInstance
	metrics       *ColdStartMetrics
}

type WarmInstance struct {
	FunctionID   string
	InitializedAt time.Time
	LastUsed    time.Time
	Ready       bool
	MemoryMB    int
}

type ColdStartMetrics struct {
	TotalColdStarts  atomic.Int64
	AvgColdStartMs  atomic.Int64
	MinColdStartMs  atomic.Int64
	MaxColdStartMs  atomic.Int64
	WarmInstances    atomic.Int64
}

func NewColdStartOptimizer() *ColdStartOptimizer {
	return &ColdStartOptimizer{
		enabled:        true,
		prewarmEnabled: true,
		prewarmDelay:   5 * time.Minute,
		warmInstances: make(map[string]*WarmInstance),
		metrics:       &ColdStartMetrics{},
	}
}

func (c *ColdStartOptimizer) Prewarm(functionID string, memoryMB int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	start := time.Now()

	instance := &WarmInstance{
		FunctionID:   functionID,
		InitializedAt: time.Now(),
		LastUsed:    time.Now(),
		Ready:       true,
		MemoryMB:    memoryMB,
	}

	c.warmInstances[functionID] = instance
	c.metrics.WarmInstances.Add(1)

	coldStartMs := time.Since(start).Milliseconds()
	c.recordColdStart(coldStartMs)

	log.Printf("[ColdStartOptimizer] Prewarmed function %s, cold start: %dms", functionID, coldStartMs)
	return nil
}

func (c *ColdStartOptimizer) IsWarmed(functionID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	instance, exists := c.warmInstances[functionID]
	if !exists {
		return false
	}

	return instance.Ready && time.Since(instance.LastUsed) < c.prewarmDelay
}

func (c *ColdStartOptimizer) MarkUsed(functionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if instance, exists := c.warmInstances[functionID]; exists {
		instance.LastUsed = time.Now()
	}
}

func (c *ColdStartOptimizer) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for id, instance := range c.warmInstances {
		if now.Sub(instance.LastUsed) > c.prewarmDelay {
			delete(c.warmInstances, id)
			c.metrics.WarmInstances.Add(-1)
		}
	}
}

func (c *ColdStartOptimizer) recordColdStart(ms int64) {
	total := c.metrics.TotalColdStarts.Load()
	if total == 0 {
		c.metrics.MinColdStartMs.Store(ms)
		c.metrics.MaxColdStartMs.Store(ms)
	} else {
		if ms < c.metrics.MinColdStartMs.Load() {
			c.metrics.MinColdStartMs.Store(ms)
		}
		if ms > c.metrics.MaxColdStartMs.Load() {
			c.metrics.MaxColdStartMs.Store(ms)
		}
	}

	avg := c.metrics.AvgColdStartMs.Load()
	newAvg := (avg*total + ms) / (total + 1)
	c.metrics.AvgColdStartMs.Store(newAvg)
}

func (c *ColdStartOptimizer) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_cold_starts": c.metrics.TotalColdStarts.Load(),
		"avg_cold_start_ms": c.metrics.AvgColdStartMs.Load(),
		"min_cold_start_ms": c.metrics.MinColdStartMs.Load(),
		"max_cold_start_ms": c.metrics.MaxColdStartMs.Load(),
		"warm_instances":    c.metrics.WarmInstances.Load(),
	}
}

func (c *ColdStartOptimizer) SetEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = enabled
}

func (c *ColdStartOptimizer) SetPrewarmDelay(delay time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.prewarmDelay = delay
}
