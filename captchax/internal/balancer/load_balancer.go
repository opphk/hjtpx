package balancer

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type Node interface {
	Address() string
	Weight() int
	IsHealthy() bool
}

type NodeStatus string

const (
	NodeStatusHealthy   NodeStatus = "healthy"
	NodeStatusUnhealthy NodeStatus = "unhealthy"
	NodeStatusDraining  NodeStatus = "draining"
)

type BaseNode struct {
	address  string
	weight  int
	healthy  atomic.Bool
	mu      sync.RWMutex
	metadata map[string]interface{}
}

func NewBaseNode(address string, weight int) *BaseNode {
	if weight <= 0 {
		weight = 1
	}
	node := &BaseNode{
		address:  address,
		weight:  weight,
		metadata: make(map[string]interface{}),
	}
	node.healthy.Store(true)
	return node
}

func (n *BaseNode) Address() string {
	return n.address
}

func (n *BaseNode) Weight() int {
	return n.weight
}

func (n *BaseNode) IsHealthy() bool {
	return n.healthy.Load()
}

func (n *BaseNode) SetHealthy(healthy bool) {
	n.healthy.Store(healthy)
}

func (n *BaseNode) SetWeight(weight int) {
	if weight <= 0 {
		weight = 1
	}
	n.mu.Lock()
	n.weight = weight
	n.mu.Unlock()
}

func (n *BaseNode) SetMetadata(key string, value interface{}) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.metadata[key] = value
}

func (n *BaseNode) GetMetadata(key string) (interface{}, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	v, ok := n.metadata[key]
	return v, ok
}

type NodeMetrics struct {
	NodeAddress    string
	RequestsTotal  int64
	SuccessTotal   int64
	FailureTotal   int64
	AvgLatencyMs   float64
	CurrentRPS     float64
	LastHealthCheck time.Time
	Weight         int
	Status         NodeStatus
}

type WeightedRandomBalancer struct {
	nodes    []Node
	weights  []int
	mu       sync.RWMutex
	stats    map[string]*NodeMetrics
	statsMu  sync.RWMutex
	selector *weightedSelector
}

type weightedSelector struct {
	nodes  []Node
	cums   []int
	total  int
	r      *rand.Rand
	mu     sync.Mutex
}

func newWeightedSelector(nodes []Node) *weightedSelector {
	ws := &weightedSelector{
		nodes: make([]Node, len(nodes)),
		cums:  make([]int, len(nodes)),
		r:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	copy(ws.nodes, nodes)

	ws.rebuild()
	return ws
}

func (ws *weightedSelector) rebuild() {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.total = 0
	for i, node := range ws.nodes {
		weight := node.Weight()
		ws.total += weight
		ws.cums[i] = ws.total
	}
}

func (ws *weightedSelector) Select() (Node, error) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.total == 0 {
		return nil, errors.New("no available nodes")
	}

	r := ws.r.Intn(ws.total)

	for i, cum := range ws.cums {
		if r < cum {
			return ws.nodes[i], nil
		}
	}

	return ws.nodes[len(ws.nodes)-1], nil
}

func NewWeightedRandomBalancer() *WeightedRandomBalancer {
	return &WeightedRandomBalancer{
		nodes: make([]Node, 0),
		stats: make(map[string]*NodeMetrics),
	}
}

func (b *WeightedRandomBalancer) AddNode(node Node) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nodes = append(b.nodes, node)
	b.selector = newWeightedSelector(b.nodes)

	b.statsMu.Lock()
	b.stats[node.Address()] = &NodeMetrics{
		NodeAddress: node.Address(),
		Weight:      node.Weight(),
		Status:      NodeStatusHealthy,
	}
	b.statsMu.Unlock()
}

func (b *WeightedRandomBalancer) RemoveNode(address string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	newNodes := make([]Node, 0, len(b.nodes))
	for _, node := range b.nodes {
		if node.Address() != address {
			newNodes = append(newNodes, node)
		}
	}

	b.nodes = newNodes
	b.selector = newWeightedSelector(b.nodes)

	b.statsMu.Lock()
	if stats, ok := b.stats[address]; ok {
		stats.Status = NodeStatusDraining
	}
	b.statsMu.Unlock()
}

func (b *WeightedRandomBalancer) SelectNode() (Node, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.nodes) == 0 {
		return nil, errors.New("no nodes available")
	}

	if b.selector == nil {
		b.mu.RUnlock()
		b.mu.Lock()
		b.selector = newWeightedSelector(b.nodes)
		b.mu.Unlock()
		b.mu.RLock()
	}

	node, err := b.selector.Select()
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (b *WeightedRandomBalancer) SelectWithFallback() (Node, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	healthyNodes := make([]Node, 0)
	for _, node := range b.nodes {
		if node.IsHealthy() {
			healthyNodes = append(healthyNodes, node)
		}
	}

	if len(healthyNodes) == 0 {
		for _, node := range b.nodes {
			healthyNodes = append(healthyNodes, node)
		}
	}

	if len(healthyNodes) == 0 {
		return nil, errors.New("no nodes available")
	}

	selector := newWeightedSelector(healthyNodes)
	return selector.Select()
}

func (b *WeightedRandomBalancer) UpdateNodeHealth(address string, healthy bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, node := range b.nodes {
		if node.Address() == address {
			if bn, ok := node.(*BaseNode); ok {
				bn.SetHealthy(healthy)
			}

			b.statsMu.Lock()
			if stats, exists := b.stats[address]; exists {
				stats.Status = NodeStatusHealthy
				if !healthy {
					stats.Status = NodeStatusUnhealthy
				}
				stats.LastHealthCheck = time.Now()
			}
			b.statsMu.Unlock()
			break
		}
	}
}

func (b *WeightedRandomBalancer) UpdateNodeWeight(address string, weight int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, node := range b.nodes {
		if node.Address() == address {
			if bn, ok := node.(*BaseNode); ok {
				bn.SetWeight(weight)
			}

			b.statsMu.Lock()
			if stats, exists := b.stats[address]; exists {
				stats.Weight = weight
			}
			b.statsMu.Unlock()

			b.selector = newWeightedSelector(b.nodes)
			break
		}
	}
}

func (b *WeightedRandomBalancer) RecordRequest(address string, latencyMs float64, success bool) {
	b.statsMu.Lock()
	defer b.statsMu.Unlock()

	stats, ok := b.stats[address]
	if !ok {
		stats = &NodeMetrics{NodeAddress: address}
		b.stats[address] = stats
	}

	stats.RequestsTotal++
	if success {
		stats.SuccessTotal++
	} else {
		stats.FailureTotal++
	}

	totalRequests := stats.RequestsTotal
	if totalRequests > 0 {
		stats.AvgLatencyMs = (stats.AvgLatencyMs*float64(totalRequests-1) + latencyMs) / float64(totalRequests)
	}
}

func (b *WeightedRandomBalancer) GetNodeStats(address string) (*NodeMetrics, bool) {
	b.statsMu.RLock()
	defer b.statsMu.RUnlock()

	stats, ok := b.stats[address]
	return stats, ok
}

func (b *WeightedRandomBalancer) GetAllStats() map[string]*NodeMetrics {
	b.statsMu.RLock()
	defer b.statsMu.RUnlock()

	result := make(map[string]*NodeMetrics)
	for k, v := range b.stats {
		result[k] = v
	}
	return result
}

func (b *WeightedRandomBalancer) GetHealthyNodes() []Node {
	b.mu.RLock()
	defer b.mu.RUnlock()

	healthy := make([]Node, 0)
	for _, node := range b.nodes {
		if node.IsHealthy() {
			healthy = append(healthy, node)
		}
	}
	return healthy
}

func (b *WeightedRandomBalancer) NodeCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.nodes)
}

func (b *WeightedRandomBalancer) HealthyNodeCount() int {
	return len(b.GetHealthyNodes())
}

type HealthChecker interface {
	CheckHealth(ctx context.Context, node Node) bool
}

type TCPPingChecker struct {
	timeout time.Duration
}

func NewTCPPingChecker(timeout time.Duration) *TCPPingChecker {
	return &TCPPingChecker{timeout: timeout}
}

func (c *TCPPingChecker) CheckHealth(ctx context.Context, node Node) bool {
	return true
}

type BalancerWithHealthCheck struct {
	balancer    *WeightedRandomBalancer
	checker     HealthChecker
	interval    time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewBalancerWithHealthCheck(balancer *WeightedRandomBalancer, checker HealthChecker, interval time.Duration) *BalancerWithHealthCheck {
	ctx, cancel := context.WithCancel(context.Background())

	return &BalancerWithHealthCheck{
		balancer: balancer,
		checker:  checker,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (b *BalancerWithHealthCheck) Start() {
	b.wg.Add(1)
	go b.healthCheckLoop()
}

func (b *BalancerWithHealthCheck) healthCheckLoop() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			b.checkAllNodes()
		}
	}
}

func (b *BalancerWithHealthCheck) checkAllNodes() {
	nodes := b.balancer.GetHealthyNodes()

	for _, node := range nodes {
		ctx, cancel := context.WithTimeout(b.ctx, 5*time.Second)

		healthy := b.checker.CheckHealth(ctx, node)

		cancel()

		b.balancer.UpdateNodeHealth(node.Address(), healthy)
	}
}

func (b *BalancerWithHealthCheck) Stop() {
	b.cancel()
	b.wg.Wait()
}

type LoadBalancer interface {
	SelectNode() (Node, error)
	SelectWithFallback() (Node, error)
	AddNode(node Node)
	RemoveNode(address string)
	UpdateNodeHealth(address string, healthy bool)
	UpdateNodeWeight(address string, weight int)
	RecordRequest(address string, latencyMs float64, success bool)
	GetAllStats() map[string]*NodeMetrics
}

func EnsureLoadBalancer(balancer LoadBalancer) LoadBalancer {
	if balancer == nil {
		return NewWeightedRandomBalancer()
	}
	return balancer
}
