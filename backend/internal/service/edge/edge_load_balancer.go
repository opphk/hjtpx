package edge

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/internal/repository"
	"github.com/hjtpx/hjtpx/pkg/config"
)

type LoadBalanceStrategy string

const (
	StrategyLeastLoad    LoadBalanceStrategy = "least_load"
	StrategyRoundRobin   LoadBalanceStrategy = "round_robin"
	StrategyRandom       LoadBalanceStrategy = "random"
	StrategyWeighted     LoadBalanceStrategy = "weighted"
)

type EdgeLoadBalancer interface {
	SelectNode(ctx context.Context, region, zone string) (*model.EdgeNode, error)
	SelectNodeWithStrategy(ctx context.Context, region, zone string, strategy LoadBalanceStrategy) (*model.EdgeNode, error)
	UpdateNodeLoad(ctx context.Context, nodeID string, load model.EdgeLoadMetrics) error
	GetAllNodes(ctx context.Context) ([]model.EdgeNode, error)
	GetOnlineNodes(ctx context.Context, region, zone string) ([]model.EdgeNode, error)
}

type edgeLoadBalancer struct {
	repo        repository.EdgeRepository
	cfg         *config.Config
	mu          sync.RWMutex
	roundRobin  map[string]int
	randSource  *rand.Rand
}

func NewEdgeLoadBalancer(repo repository.EdgeRepository, cfg *config.Config) EdgeLoadBalancer {
	return &edgeLoadBalancer{
		repo:        repo,
		cfg:         cfg,
		roundRobin:  make(map[string]int),
		randSource:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (lb *edgeLoadBalancer) SelectNode(ctx context.Context, region, zone string) (*model.EdgeNode, error) {
	strategy := LoadBalanceStrategy(lb.cfg.Edge.LoadBalanceStrategy)
	return lb.SelectNodeWithStrategy(ctx, region, zone, strategy)
}

func (lb *edgeLoadBalancer) SelectNodeWithStrategy(ctx context.Context, region, zone string, strategy LoadBalanceStrategy) (*model.EdgeNode, error) {
	nodes, err := lb.GetOnlineNodes(ctx, region, zone)
	if err != nil {
		return nil, err
	}

	if len(nodes) == 0 {
		return nil, ErrNoAvailableNodes
	}

	switch strategy {
	case StrategyLeastLoad:
		return lb.selectLeastLoadNode(nodes), nil
	case StrategyRoundRobin:
		return lb.selectRoundRobinNode(nodes, region, zone), nil
	case StrategyRandom:
		return lb.selectRandomNode(nodes), nil
	case StrategyWeighted:
		return lb.selectWeightedNode(nodes), nil
	default:
		return lb.selectLeastLoadNode(nodes), nil
	}
}

func (lb *edgeLoadBalancer) selectLeastLoadNode(nodes []model.EdgeNode) *model.EdgeNode {
	var bestNode *model.EdgeNode
	minLoad := float64(100)

	for _, node := range nodes {
		nodeLoad := lb.calculateLoad(node)
		if nodeLoad < minLoad {
			minLoad = nodeLoad
			bestNode = &node
		}
	}

	return bestNode
}

func (lb *edgeLoadBalancer) selectRoundRobinNode(nodes []model.EdgeNode, region, zone string) *model.EdgeNode {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	key := region + ":" + zone
	index := lb.roundRobin[key] % len(nodes)
	node := nodes[index]
	lb.roundRobin[key]++

	return &node
}

func (lb *edgeLoadBalancer) selectRandomNode(nodes []model.EdgeNode) *model.EdgeNode {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	index := lb.randSource.Intn(len(nodes))
	return &nodes[index]
}

func (lb *edgeLoadBalancer) selectWeightedNode(nodes []model.EdgeNode) *model.EdgeNode {
	totalWeight := 0
	for _, node := range nodes {
		totalWeight += node.Capacity.MaxRequestsPerSecond
	}

	randomWeight := lb.randSource.Intn(totalWeight)
	currentWeight := 0

	for _, node := range nodes {
		currentWeight += node.Capacity.MaxRequestsPerSecond
		if currentWeight > randomWeight {
			return &node
		}
	}

	return &nodes[0]
}

func (lb *edgeLoadBalancer) calculateLoad(node model.EdgeNode) float64 {
	cpuLoad := float64(node.CurrentLoad.CPUUsagePercent) / 100
	memoryLoad := float64(node.CurrentLoad.MemoryUsageMB) / float64(node.Capacity.MemoryLimitMB)
	rpsLoad := float64(node.CurrentLoad.CurrentRequestsPerSecond) / float64(node.Capacity.MaxRequestsPerSecond)
	connLoad := float64(node.CurrentLoad.CurrentConcurrentRequests) / float64(node.Capacity.MaxConcurrentRequests)

	return (cpuLoad*0.3 + memoryLoad*0.2 + rpsLoad*0.3 + connLoad*0.2)
}

func (lb *edgeLoadBalancer) UpdateNodeLoad(ctx context.Context, nodeID string, load model.EdgeLoadMetrics) error {
	return lb.repo.UpdateNodeHeartbeat(ctx, nodeID, load)
}

func (lb *edgeLoadBalancer) GetAllNodes(ctx context.Context) ([]model.EdgeNode, error) {
	return lb.repo.ListAllNodes(ctx)
}

func (lb *edgeLoadBalancer) GetOnlineNodes(ctx context.Context, region, zone string) ([]model.EdgeNode, error) {
	nodes, err := lb.repo.ListNodes(ctx, region, zone, model.EdgeNodeStatusOnline)
	if err != nil {
		return nil, err
	}

	filtered := make([]model.EdgeNode, 0)
	now := time.Now()
	for _, node := range nodes {
		if now.Sub(node.LastHeartbeat) < time.Duration(lb.cfg.Edge.HeartbeatIntervalSecs*2)*time.Second {
			filtered = append(filtered, node)
		}
	}

	return filtered, nil
}