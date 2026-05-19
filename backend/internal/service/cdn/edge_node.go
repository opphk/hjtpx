package cdn

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

var (
	ErrFunctionNotFound     = errors.New("edge function not found")
	ErrFunctionExecution    = errors.New("edge function execution failed")
	ErrNodeCapacityExceeded = errors.New("node capacity exceeded")
)

type EdgeNode struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	RegionID        string            `json:"region_id"`
	IPAddress       string            `json:"ip_address"`
	Hostname        string            `json:"hostname"`
	Port            int               `json:"port"`
	IsHealthy       bool              `json:"is_healthy"`
	LatencyMs       float64           `json:"latency_ms"`
	Capacity        int               `json:"capacity"`
	CurrentLoad     int               `json:"current_load"`
	TrafficBytes    int64             `json:"traffic_bytes"`
	RequestCount    int64             `json:"request_count"`
	Functions       map[string]EdgeFunction `json:"functions"`
	CreatedAt       time.Time         `json:"created_at"`
	LastHealthCheck time.Time         `json:"last_health_check"`
	mu              sync.RWMutex
	httpClient      *http.Client
}

type EdgeFunction struct {
	Name        string            `json:"name"`
	Code        string            `json:"code"`
	Runtime     string            `json:"runtime"`
	MemoryLimit int               `json:"memory_limit_mb"`
	Timeout     time.Duration     `json:"timeout_seconds"`
	Enabled     bool              `json:"enabled"`
	DeployedAt  time.Time         `json:"deployed_at"`
}

type EdgeNodeStats struct {
	NodeID         string  `json:"node_id"`
	NodeName       string  `json:"node_name"`
	RegionID       string  `json:"region_id"`
	IsHealthy      bool    `json:"is_healthy"`
	LatencyMs      float64 `json:"latency_ms"`
	Capacity       int     `json:"capacity"`
	CurrentLoad    int     `json:"current_load"`
	LoadPercentage float64 `json:"load_percentage"`
	TrafficBytes   int64   `json:"traffic_bytes"`
	RequestCount   int64   `json:"request_count"`
}

func NewEdgeNode(id, name, regionID, ipAddress, hostname string, port int) *EdgeNode {
	return &EdgeNode{
		ID:         id,
		Name:       name,
		RegionID:   regionID,
		IPAddress:  ipAddress,
		Hostname:   hostname,
		Port:       port,
		IsHealthy:  true,
		Capacity:   1000,
		Functions:  make(map[string]EdgeFunction),
		httpClient: createHTTPClient(),
	}
}

func createHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			MaxIdleConns:    100,
			IdleConnTimeout: 30 * time.Second,
		},
		Timeout: 30 * time.Second,
	}
}

func (n *EdgeNode) HealthCheck() bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	start := time.Now()
	url := fmt.Sprintf("http://%s:%d/health", n.IPAddress, n.Port)

	resp, err := n.httpClient.Get(url)
	if err != nil {
		n.IsHealthy = false
		n.LatencyMs = 9999
		n.LastHealthCheck = time.Now()
		return false
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	latency := time.Since(start).Milliseconds()
	n.LatencyMs = float64(latency)
	n.IsHealthy = resp.StatusCode == http.StatusOK
	n.LastHealthCheck = time.Now()

	return n.IsHealthy
}

func (n *EdgeNode) ExecuteFunction(ctx context.Context, functionName string, params map[string]interface{}) (*model.EdgeExecutionResult, error) {
	n.mu.RLock()
	function, exists := n.Functions[functionName]
	n.mu.RUnlock()

	if !exists {
		return nil, ErrFunctionNotFound
	}

	if !function.Enabled {
		return nil, errors.New("function is disabled")
	}

	n.mu.Lock()
	if n.CurrentLoad >= n.Capacity {
		n.mu.Unlock()
		return nil, ErrNodeCapacityExceeded
	}
	n.CurrentLoad++
	n.RequestCount++
	n.mu.Unlock()

	defer func() {
		n.mu.Lock()
		n.CurrentLoad--
		n.mu.Unlock()
	}()

	result := &model.EdgeExecutionResult{
		NodeID:       n.ID,
		FunctionName: functionName,
		StartTime:    time.Now(),
		Success:      true,
	}

	time.Sleep(10 * time.Millisecond)

	result.EndTime = time.Now()
	result.DurationMs = float64(result.EndTime.Sub(result.StartTime).Milliseconds())
	result.Output = map[string]interface{}{
		"message": "Function executed successfully",
		"params":  params,
	}

	return result, nil
}

func (n *EdgeNode) DeployFunction(function EdgeFunction) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	function.DeployedAt = time.Now()
	n.Functions[function.Name] = function

	return nil
}

func (n *EdgeNode) RemoveFunction(functionName string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if _, exists := n.Functions[functionName]; !exists {
		return ErrFunctionNotFound
	}

	delete(n.Functions, functionName)
	return nil
}

func (n *EdgeNode) GetFunction(functionName string) (EdgeFunction, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	function, exists := n.Functions[functionName]
	if !exists {
		return EdgeFunction{}, ErrFunctionNotFound
	}
	return function, nil
}

func (n *EdgeNode) ListFunctions() []EdgeFunction {
	n.mu.RLock()
	defer n.mu.RUnlock()

	result := make([]EdgeFunction, 0, len(n.Functions))
	for _, fn := range n.Functions {
		result = append(result, fn)
	}
	return result
}

func (n *EdgeNode) RecordTraffic(bytes int64) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.TrafficBytes += bytes
}

func (n *EdgeNode) GetStats() EdgeNodeStats {
	n.mu.RLock()
	defer n.mu.RUnlock()

	loadPercentage := 0.0
	if n.Capacity > 0 {
		loadPercentage = float64(n.CurrentLoad) / float64(n.Capacity) * 100
	}

	return EdgeNodeStats{
		NodeID:         n.ID,
		NodeName:       n.Name,
		RegionID:       n.RegionID,
		IsHealthy:      n.IsHealthy,
		LatencyMs:      n.LatencyMs,
		Capacity:       n.Capacity,
		CurrentLoad:    n.CurrentLoad,
		LoadPercentage: loadPercentage,
		TrafficBytes:   n.TrafficBytes,
		RequestCount:   n.RequestCount,
	}
}

func (n *EdgeNode) UpdateLoad(delta int) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.CurrentLoad = max(0, n.CurrentLoad+delta)
}

func (n *EdgeNode) String() string {
	return fmt.Sprintf("EdgeNode{ID=%s, Region=%s, IP=%s, Healthy=%v, Load=%d/%d}",
		n.ID, n.RegionID, n.IPAddress, n.IsHealthy, n.CurrentLoad, n.Capacity)
}

type EdgeNodeManager struct {
	nodes      map[string]*EdgeNode
	healthTicker *time.Ticker
	mu         sync.RWMutex
}

func NewEdgeNodeManager() *EdgeNodeManager {
	manager := &EdgeNodeManager{
		nodes: make(map[string]*EdgeNode),
	}
	manager.startHealthChecker()
	return manager
}

func (m *EdgeNodeManager) startHealthChecker() {
	m.healthTicker = time.NewTicker(30 * time.Second)
	go func() {
		for range m.healthTicker.C {
			m.checkAllNodesHealth()
		}
	}()
}

func (m *EdgeNodeManager) checkAllNodesHealth() {
	m.mu.RLock()
	nodes := make([]*EdgeNode, 0, len(m.nodes))
	for _, node := range m.nodes {
		nodes = append(nodes, node)
	}
	m.mu.RUnlock()

	for _, node := range nodes {
		node.HealthCheck()
	}
}

func (m *EdgeNodeManager) RegisterNode(node *EdgeNode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.nodes[node.ID]; exists {
		return ErrNodeAlreadyExists
	}

	m.nodes[node.ID] = node
	return nil
}

func (m *EdgeNodeManager) UnregisterNode(nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.nodes[nodeID]; !exists {
		return ErrNodeNotFound
	}

	delete(m.nodes, nodeID)
	return nil
}

func (m *EdgeNodeManager) GetNode(nodeID string) (*EdgeNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return nil, ErrNodeNotFound
	}
	return node, nil
}

func (m *EdgeNodeManager) ListNodes(regionID string) []*EdgeNode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []*EdgeNode{}
	for _, node := range m.nodes {
		if regionID == "" || node.RegionID == regionID {
			result = append(result, node)
		}
	}
	return result
}

func (m *EdgeNodeManager) GetHealthyNodes(regionID string) []*EdgeNode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []*EdgeNode{}
	for _, node := range m.nodes {
		if node.IsHealthy && (regionID == "" || node.RegionID == regionID) {
			result = append(result, node)
		}
	}
	return result
}

func (m *EdgeNodeManager) GetLeastLoadedNode(regionID string) (*EdgeNode, error) {
	nodes := m.GetHealthyNodes(regionID)
	if len(nodes) == 0 {
		return nil, ErrNoHealthyNode
	}

	var bestNode *EdgeNode
	minLoad := float64(100)

	for _, node := range nodes {
		stats := node.GetStats()
		if stats.LoadPercentage < minLoad {
			minLoad = stats.LoadPercentage
			bestNode = node
		}
	}

	return bestNode, nil
}

func (m *EdgeNodeManager) GetFastestNode(regionID string) (*EdgeNode, error) {
	nodes := m.GetHealthyNodes(regionID)
	if len(nodes) == 0 {
		return nil, ErrNoHealthyNode
	}

	var bestNode *EdgeNode
	minLatency := float64(9999)

	for _, node := range nodes {
		if node.LatencyMs < minLatency {
			minLatency = node.LatencyMs
			bestNode = node
		}
	}

	return bestNode, nil
}

func (m *EdgeNodeManager) GetAllStats() []EdgeNodeStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]EdgeNodeStats, 0, len(m.nodes))
	for _, node := range m.nodes {
		result = append(result, node.GetStats())
	}
	return result
}

func (m *EdgeNodeManager) Stop() {
	if m.healthTicker != nil {
		m.healthTicker.Stop()
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}