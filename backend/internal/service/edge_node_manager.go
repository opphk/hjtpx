
package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type EdgeNodeManager struct {
	mu            sync.RWMutex
	nodes         map[string]*EdgeNode
	healthCheck   *HealthChecker
	loadBalancer  *EdgeLoadBalancer
	initialized   bool
}

type EdgeNode struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Address          string        `json:"address"`
	Port             int           `json:"port"`
	Region           string        `json:"region"`
	Status           string        `json:"status"`
	Healthy          bool          `json:"healthy"`
	LastHealthCheck  time.Time     `json:"last_health_check"`
	Capacity         int           `json:"capacity"`
	CurrentLoad      int           `json:"current_load"`
	Latency          time.Duration `json:"latency"`
	SupportedFeatures []string    `json:"supported_features"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

type HealthChecker struct {
	ctx            context.Context
	cancel         context.CancelFunc
	checkInterval  time.Duration
	timeout        time.Duration
}

type EdgeLoadBalancer struct {
	mu            sync.RWMutex
	strategy      string
	nodes         []*EdgeNode
}

type NodeRegistrationRequest struct {
	Name     string   `json:"name"`
	Address  string   `json:"address"`
	Port     int      `json:"port"`
	Region   string   `json:"region"`
	Capacity int      `json:"capacity"`
	Features []string `json:"features"`
}

type NodeRegistrationResponse struct {
	Success bool   `json:"success"`
	NodeID  string `json:"node_id"`
	Message string `json:"message"`
}

type NodeStatusResponse struct {
	Node  *EdgeNode  `json:"node"`
	Stats *NodeStats `json:"stats"`
}

type NodeStats struct {
	TotalRequests  int64         `json:"total_requests"`
	ActiveRequests int           `json:"active_requests"`
	AvgLatency     time.Duration `json:"avg_latency"`
	ErrorRate      float64       `json:"error_rate"`
	Uptime         time.Duration `json:"uptime"`
}

func NewEdgeNodeManager() *EdgeNodeManager {
	return &EdgeNodeManager{
		nodes:        make(map[string]*EdgeNode),
		healthCheck:  NewHealthChecker(),
		loadBalancer: NewEdgeLoadBalancer(),
	}
}

func NewHealthChecker() *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthChecker{
		ctx:            ctx,
		cancel:         cancel,
		checkInterval:  30 * time.Second,
		timeout:        5 * time.Second,
	}
}

func NewEdgeLoadBalancer() *EdgeLoadBalancer {
	return &EdgeLoadBalancer{
		strategy: "least_load",
		nodes:    make([]*EdgeNode, 0),
	}
}

func (m *EdgeNodeManager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return nil
	}

	m.healthCheck.Start(ctx, m)
	m.initialized = true
	log.Println("[EdgeNodeManager] Initialized successfully")
	return nil
}

func (m *EdgeNodeManager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return nil
	}

	m.healthCheck.Stop()
	m.initialized = false
	log.Println("[EdgeNodeManager] Shutdown complete")
	return nil
}

func (m *EdgeNodeManager) RegisterNode(ctx context.Context, req *NodeRegistrationRequest) (*NodeRegistrationResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	nodeID := generateNodeID()

	node := &EdgeNode{
		ID:               nodeID,
		Name:             req.Name,
		Address:          req.Address,
		Port:             req.Port,
		Region:           req.Region,
		Status:           "active",
		Healthy:          true,
		LastHealthCheck:  time.Now(),
		Capacity:         req.Capacity,
		CurrentLoad:      0,
		Latency:          0,
		SupportedFeatures: req.Features,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	m.nodes[nodeID] = node
	m.loadBalancer.AddNode(node)

	log.Printf("[EdgeNodeManager] Registered node: %s (%s)", nodeID, req.Name)

	return &NodeRegistrationResponse{
		Success: true,
		NodeID:  nodeID,
		Message: "Node registered successfully",
	}, nil
}

func (m *EdgeNodeManager) DeregisterNode(ctx context.Context, nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	node.Status = "inactive"
	delete(m.nodes, nodeID)
	m.loadBalancer.RemoveNode(nodeID)

	log.Printf("[EdgeNodeManager] Deregistered node: %s", nodeID)
	return nil
}

func (m *EdgeNodeManager) GetNode(ctx context.Context, nodeID string) (*EdgeNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	return node, nil
}

func (m *EdgeNodeManager) ListNodes(ctx context.Context) []*EdgeNode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	nodes := make([]*EdgeNode, 0, len(m.nodes))
	for _, node := range m.nodes {
		nodes = append(nodes, node)
	}

	return nodes
}

func (m *EdgeNodeManager) ListActiveNodes(ctx context.Context) []*EdgeNode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	nodes := make([]*EdgeNode, 0)
	for _, node := range m.nodes {
		if node.Status == "active" && node.Healthy {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

func (m *EdgeNodeManager) SelectNode(ctx context.Context, feature string) (*EdgeNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.loadBalancer.SelectNode(feature)
}

func (m *EdgeNodeManager) UpdateNodeLoad(ctx context.Context, nodeID string, load int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	node.CurrentLoad = load
	node.UpdatedAt = time.Now()
	return nil
}

func (m *EdgeNodeManager) ReportNodeHealth(ctx context.Context, nodeID string, healthy bool, latency time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	node.Healthy = healthy
	node.Latency = latency
	node.LastHealthCheck = time.Now()
	node.UpdatedAt = time.Now()

	return nil
}

func (m *EdgeNodeManager) GetNodeStats(ctx context.Context, nodeID string) (*NodeStatusResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	stats := &NodeStats{
		TotalRequests:  0,
		ActiveRequests: node.CurrentLoad,
		AvgLatency:     node.Latency,
		ErrorRate:      0.0,
		Uptime:         time.Since(node.CreatedAt),
	}

	return &NodeStatusResponse{
		Node:  node,
		Stats: stats,
	}, nil
}

func (h *HealthChecker) Start(ctx context.Context, manager *EdgeNodeManager) {
	go func() {
		ticker := time.NewTicker(h.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("[HealthChecker] Stopping health checks")
				return
			case <-ticker.C:
				h.checkAllNodes(manager)
			}
		}
	}()
}

func (h *HealthChecker) Stop() {
	h.cancel()
}

func (h *HealthChecker) checkAllNodes(manager *EdgeNodeManager) {
	manager.mu.RLock()
	nodes := make([]*EdgeNode, 0, len(manager.nodes))
	for _, node := range manager.nodes {
		nodes = append(nodes, node)
	}
	manager.mu.RUnlock()

	for _, node := range nodes {
		h.checkNode(h.ctx, manager, node)
	}
}

func (h *HealthChecker) checkNode(ctx context.Context, manager *EdgeNodeManager, node *EdgeNode) {
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	start := time.Now()
	healthy := h.performHealthCheck(ctx, node)
	latency := time.Since(start)

	manager.ReportNodeHealth(ctx, node.ID, healthy, latency)
}

func (h *HealthChecker) performHealthCheck(ctx context.Context, node *EdgeNode) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(100 * time.Millisecond):
		return true
	}
}

func (lb *EdgeLoadBalancer) AddNode(node *EdgeNode) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.nodes = append(lb.nodes, node)
}

func (lb *EdgeLoadBalancer) RemoveNode(nodeID string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, node := range lb.nodes {
		if node.ID == nodeID {
			lb.nodes = append(lb.nodes[:i], lb.nodes[i+1:]...)
			break
		}
	}
}

func (lb *EdgeLoadBalancer) SelectNode(feature string) (*EdgeNode, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}

	switch lb.strategy {
	case "least_load":
		return lb.selectLeastLoadNode(feature)
	case "round_robin":
		return lb.selectRoundRobinNode(feature)
	case "region_aware":
		return lb.selectRegionAwareNode(feature)
	default:
		return lb.selectLeastLoadNode(feature)
	}
}

func (lb *EdgeLoadBalancer) selectLeastLoadNode(feature string) (*EdgeNode, error) {
	var selected *EdgeNode
	minLoad := int(^uint(0) >> 1)

	for _, node := range lb.nodes {
		if !node.Healthy || node.Status != "active" {
			continue
		}

		if feature != "" && !hasFeature(node.SupportedFeatures, feature) {
			continue
		}

		if node.CurrentLoad < minLoad {
			minLoad = node.CurrentLoad
			selected = node
		}
	}

	if selected == nil {
		return nil, fmt.Errorf("no suitable node found")
	}

	return selected, nil
}

func (lb *EdgeLoadBalancer) selectRoundRobinNode(feature string) (*EdgeNode, error) {
	for _, node := range lb.nodes {
		if !node.Healthy || node.Status != "active" {
			continue
		}

		if feature != "" && !hasFeature(node.SupportedFeatures, feature) {
			continue
		}

		return node, nil
	}

	return nil, fmt.Errorf("no suitable node found")
}

func (lb *EdgeLoadBalancer) selectRegionAwareNode(feature string) (*EdgeNode, error) {
	return lb.selectLeastLoadNode(feature)
}

func hasFeature(features []string, feature string) bool {
	for _, f := range features {
		if f == feature {
			return true
		}
	}
	return false
}

func generateNodeID() string {
	return fmt.Sprintf("node_%d", time.Now().UnixNano())
}


