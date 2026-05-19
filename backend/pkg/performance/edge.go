package performance

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type EdgeCompute struct {
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	isRunning    bool
	nodes        []*EdgeNode
	loadBalancer *LoadBalancer
	cache        *EdgeCache
	stats        *EdgeStats
}

type EdgeNode struct {
	ID         string
	Address    string
	Capacity   int
	Load       int
	Healthy    bool
	LastCheck  time.Time
}

type LoadBalancer struct {
	mu          sync.RWMutex
	strategy    string
	roundRobin  int
}

type EdgeCache struct {
	mu       sync.RWMutex
	cache    map[string][]byte
	maxSize  int
}

type EdgeStats struct {
	TotalRequests   atomic.Int64
	EdgeRequests    atomic.Int64
	CacheHits       atomic.Int64
	CacheMisses     atomic.Int64
	AvgLatency      atomic.Int64
	ActiveNodes     atomic.Int64
	TotalNodes      atomic.Int64
	LastUpdate      atomic.Value
}

func NewEdgeCompute() *EdgeCompute {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &EdgeCompute{
		ctx:          ctx,
		cancel:       cancel,
		nodes:        make([]*EdgeNode, 0),
		loadBalancer: NewLoadBalancer(),
		cache:        NewEdgeCache(10000),
		stats:        &EdgeStats{},
	}
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		strategy: "round-robin",
	}
}

func NewEdgeCache(maxSize int) *EdgeCache {
	return &EdgeCache{
		cache:   make(map[string][]byte, maxSize),
		maxSize: maxSize,
	}
}

func (e *EdgeCompute) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.isRunning {
		return nil
	}

	e.isRunning = true

	go e.monitorNodes()
	go e.cleanCache()

	log.Println("[EdgeCompute] Started successfully")
	return nil
}

func (e *EdgeCompute) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning {
		return
	}

	e.cancel()
	e.isRunning = false

	log.Println("[EdgeCompute] Stopped")
}

func (e *EdgeCompute) AddNode(id, address string, capacity int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	node := &EdgeNode{
		ID:        id,
		Address:   address,
		Capacity:  capacity,
		Healthy:   true,
		LastCheck: time.Now(),
	}

	e.nodes = append(e.nodes, node)
	e.stats.TotalNodes.Store(int64(len(e.nodes)))
	log.Printf("[EdgeCompute] Added edge node: %s", id)
}

func (e *EdgeCompute) RemoveNode(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, node := range e.nodes {
		if node.ID == id {
			e.nodes = append(e.nodes[:i], e.nodes[i+1:]...)
			e.stats.TotalNodes.Store(int64(len(e.nodes)))
			log.Printf("[EdgeCompute] Removed edge node: %s", id)
			break
		}
	}
}

func (e *EdgeCompute) RouteRequest(key string) (*EdgeNode, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	e.stats.TotalRequests.Add(1)

	if _, ok := e.cache.get(key); ok {
		e.stats.CacheHits.Add(1)
		e.stats.EdgeRequests.Add(1)
		return nil, true
	}

	e.stats.CacheMisses.Add(1)

	node := e.loadBalancer.selectNode(e.nodes)
	if node != nil && node.Healthy {
		e.stats.EdgeRequests.Add(1)
		return node, false
	}

	return nil, false
}

func (e *EdgeCompute) CacheResult(key string, data []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cache.set(key, data)
}

func (e *EdgeCompute) monitorNodes() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.checkNodeHealth()
		}
	}
}

func (e *EdgeCompute) checkNodeHealth() {
	e.mu.Lock()
	defer e.mu.Unlock()

	healthyCount := 0
	for _, node := range e.nodes {
		// In real implementation, this would check actual node health
		node.Healthy = time.Since(node.LastCheck) < 5*time.Minute
		if node.Healthy {
			healthyCount++
		}
	}

	e.stats.ActiveNodes.Store(int64(healthyCount))
	e.stats.LastUpdate.Store(time.Now())
}

func (e *EdgeCompute) cleanCache() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.mu.Lock()
			if len(e.cache.cache) > e.cache.maxSize/2 {
				e.cache.cache = make(map[string][]byte, e.cache.maxSize)
			}
			e.mu.Unlock()
		}
	}
}

func (e *EdgeCompute) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_requests":  e.stats.TotalRequests.Load(),
		"edge_requests":   e.stats.EdgeRequests.Load(),
		"cache_hits":      e.stats.CacheHits.Load(),
		"cache_misses":    e.stats.CacheMisses.Load(),
		"avg_latency":     e.stats.AvgLatency.Load(),
		"active_nodes":    e.stats.ActiveNodes.Load(),
		"total_nodes":     e.stats.TotalNodes.Load(),
		"last_update":     e.stats.LastUpdate.Load(),
	}
}

func (lb *LoadBalancer) selectNode(nodes []*EdgeNode) *EdgeNode {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(nodes) == 0 {
		return nil
	}

	switch lb.strategy {
	case "round-robin":
		lb.roundRobin = (lb.roundRobin + 1) % len(nodes)
		for i := 0; i < len(nodes); i++ {
			node := nodes[(lb.roundRobin+i)%len(nodes)]
			if node.Healthy {
				return node
			}
		}
	case "least-load":
		var selected *EdgeNode
		for _, node := range nodes {
			if node.Healthy && (selected == nil || node.Load < selected.Load) {
				selected = node
			}
		}
		return selected
	}

	return nil
}

func (ec *EdgeCache) get(key string) ([]byte, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	data, ok := ec.cache[key]
	return data, ok
}

func (ec *EdgeCache) set(key string, data []byte) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	if len(ec.cache) >= ec.maxSize {
		for k := range ec.cache {
			delete(ec.cache, k)
			break
		}
	}
	ec.cache[key] = data
}
