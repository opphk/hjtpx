package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

type LBAlgorithm string

const (
	LBAlgorithmRoundRobin     LBAlgorithm = "round_robin"
	LBAlgorithmLeastConn      LBAlgorithm = "least_connection"
	LBAlgorithmIPHash        LBAlgorithm = "ip_hash"
	LBAlgorithmWeighted      LBAlgorithm = "weighted"
	LBAlgorithmLatency       LBAlgorithm = "latency_based"
	LBAlgorithmResource      LBAlgorithm = "resource_based"
	LBAlgorithmGeolocation   LBAlgorithm = "geolocation"
	LBAlgorithmAdaptive      LBAlgorithm = "adaptive"
)

type LBMode string

const (
	LBModeActive     LBMode = "active"
	LBModePassive    LBMode = "passive"
	LBModePredictive LBMode = "predictive"
)

type LBPolicy struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Algorithm    LBAlgorithm  `json:"algorithm"`
	Mode         LBMode       `json:"mode"`
	HealthCheck  bool         `json:"health_check"`
	Failover     bool         `json:"failover"`
	RetryAttempts int         `json:"retry_attempts"`
	Timeout      time.Duration `json:"timeout"`
	Weights      map[string]int `json:"weights,omitempty"`
	RegionWeights map[Region]int `json:"region_weights,omitempty"`
	Enabled      bool         `json:"enabled"`
}

type LBNode struct {
	NodeID        string        `json:"node_id"`
	IPAddress     string        `json:"ip_address"`
	Port          int           `json:"port"`
	Weight        int           `json:"weight"`
	CurrentConns  int32         `json:"current_connections"`
	ActiveRequests int32        `json:"active_requests"`
	LatencyMs     float64       `json:"latency_ms"`
	CPUUsage      float64       `json:"cpu_usage"`
	MemoryUsage   float64       `json:"memory_usage"`
	Healthy       bool          `json:"healthy"`
	LastHeartbeat time.Time     `json:"last_heartbeat"`
	Region        Region        `json:"region"`
	FailCount     int           `json:"fail_count"`
	SuccessCount  int64         `json:"success_count"`
	TotalRequests int64         `json:"total_requests"`
}

type LoadBalancerConfig struct {
	Policy         *LBPolicy          `json:"policy"`
	MaxConnections int                `json:"max_connections"`
	ConnectionTimeout time.Duration   `json:"connection_timeout"`
	IdleTimeout     time.Duration     `json:"idle_timeout"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	MetricsEnabled  bool              `json:"metrics_enabled"`
	StickySessions  bool              `json:"sticky_sessions"`
	SessionAffinity int               `json:"session_affinity_minutes"`
}

type LoadBalancerMetrics struct {
	TotalRequests     int64              `json:"total_requests"`
	TotalResponses    int64              `json:"total_responses"`
	ActiveConnections int64              `json:"active_connections"`
	FailedRequests    int64              `json:"failed_requests"`
	RetriedRequests   int64              `json:"retried_requests"`
	AvgLatencyMs      float64            `json:"avg_latency_ms"`
	MinLatencyMs      float64            `json:"min_latency_ms"`
	MaxLatencyMs      float64            `json:"max_latency_ms"`
	TotalLatencyMs    float64            `json:"total_latency_ms"`
	NodeMetrics       map[string]*LBNodeMetric `json:"node_metrics"`
	AlgorithmMetrics  map[LBAlgorithm]*AlgorithmMetric `json:"algorithm_metrics"`
	mu                sync.RWMutex
}

type LBNodeMetric struct {
	NodeID          string  `json:"node_id"`
	RequestCount    int64   `json:"request_count"`
	ErrorCount      int64   `json:"error_count"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
	TotalLatencyMs  float64 `json:"total_latency_ms"`
	CurrentConns    int32   `json:"current_connections"`
	SuccessRate     float64 `json:"success_rate"`
}

type AlgorithmMetric struct {
	Algorithm      LBAlgorithm `json:"algorithm"`
	SelectionCount int64      `json:"selection_count"`
	AvgLatencyMs   float64    `json:"avg_latency_ms"`
}

type EdgeLoadBalancer struct {
	nodes        map[string]*LBNode
	policies     map[string]*LBPolicy
	config       *LoadBalancerConfig
	nodeManager  *EdgeNodeManager
	healthCheck  *HealthCheckManager
	redisClient  *redis.Client
	mu           sync.RWMutex
	metrics      *LoadBalancerMetrics
	affinityMap   map[string]string
	affinityMu    sync.RWMutex
	requestCount  int64
	version      int64
}

type LBRequest struct {
	ID          string                 `json:"id"`
	ClientIP    string                 `json:"client_ip"`
	Headers     map[string]string     `json:"headers"`
	Path        string                 `json:"path"`
	Method      string                 `json:"method"`
	Body        interface{}            `json:"body,omitempty"`
	Policy      string                 `json:"policy,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type LBResponse struct {
	RequestID    string            `json:"request_id"`
	NodeID       string            `json:"node_id"`
	IPAddress    string            `json:"ip_address"`
	Port         int               `json:"port"`
	LatencyMs    float64           `json:"latency_ms"`
	Success      bool              `json:"success"`
	Error        string            `json:"error,omitempty"`
	Attempts     int               `json:"attempts"`
	Timestamp    time.Time         `json:"timestamp"`
	FromCache    bool              `json:"from_cache"`
}

func NewEdgeLoadBalancer(nodeManager *EdgeNodeManager, healthCheck *HealthCheckManager, redisClient *redis.Client) *EdgeLoadBalancer {
	lb := &EdgeLoadBalancer{
		nodes:        make(map[string]*LBNode),
		policies:     make(map[string]*LBPolicy),
		nodeManager:  nodeManager,
		healthCheck:  healthCheck,
		redisClient:  redisClient,
		metrics: &LoadBalancerMetrics{
			NodeMetrics:      make(map[string]*LBNodeMetric),
			AlgorithmMetrics: make(map[LBAlgorithm]*AlgorithmMetric),
		},
		affinityMap: make(map[string]string),
		requestCount: 0,
		version:    1,
	}

	lb.config = &LoadBalancerConfig{
		Policy:              lb.getDefaultPolicy(),
		MaxConnections:      10000,
		ConnectionTimeout:   10 * time.Second,
		IdleTimeout:         300 * time.Second,
		HealthCheckInterval: 10 * time.Second,
		MetricsEnabled:      true,
		StickySessions:      true,
		SessionAffinity:     30,
	}

	lb.createDefaultPolicies()

	return lb
}

func (lb *EdgeLoadBalancer) getDefaultPolicy() *LBPolicy {
	return &LBPolicy{
		ID:             "default",
		Name:           "Default Policy",
		Algorithm:      LBAlgorithmAdaptive,
		Mode:           LBModePredictive,
		HealthCheck:    true,
		Failover:       true,
		RetryAttempts:  3,
		Timeout:        10 * time.Second,
		Weights:        make(map[string]int),
		RegionWeights:  make(map[Region]int),
		Enabled:        true,
	}
}

func (lb *EdgeLoadBalancer) createDefaultPolicies() {
	policies := []*LBPolicy{
		{
			ID:             "round_robin",
			Name:           "Round Robin",
			Algorithm:      LBAlgorithmRoundRobin,
			Mode:           LBModeActive,
			HealthCheck:    true,
			Failover:       true,
			RetryAttempts:  2,
			Timeout:        10 * time.Second,
			Enabled:        true,
		},
		{
			ID:             "least_connection",
			Name:           "Least Connection",
			Algorithm:      LBAlgorithmLeastConn,
			Mode:           LBModeActive,
			HealthCheck:    true,
			Failover:       true,
			RetryAttempts:  2,
			Timeout:        10 * time.Second,
			Enabled:        true,
		},
		{
			ID:             "latency_based",
			Name:           "Latency Based",
			Algorithm:      LBAlgorithmLatency,
			Mode:           LBModeActive,
			HealthCheck:    true,
			Failover:       true,
			RetryAttempts:  3,
			Timeout:        5 * time.Second,
			Enabled:        true,
		},
		{
			ID:             "geolocation",
			Name:           "Geolocation Based",
			Algorithm:      LBAlgorithmGeolocation,
			Mode:           LBModePredictive,
			HealthCheck:    true,
			Failover:       true,
			RetryAttempts:  3,
			Timeout:        10 * time.Second,
			Enabled:        true,
		},
		{
			ID:             "weighted",
			Name:           "Weighted",
			Algorithm:      LBAlgorithmWeighted,
			Mode:           LBModeActive,
			HealthCheck:    true,
			Failover:       true,
			RetryAttempts:  2,
			Timeout:        10 * time.Second,
			Enabled:        true,
		},
	}

	for _, policy := range policies {
		lb.policies[policy.ID] = policy
	}
}

func (lb *EdgeLoadBalancer) AddNode(node *LBNode) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if node.NodeID == "" {
		return fmt.Errorf("node ID is required")
	}

	node.Healthy = true
	node.LastHeartbeat = time.Now()
	node.CurrentConns = 0
	node.ActiveRequests = 0
	node.FailCount = 0
	node.SuccessCount = 0
	node.TotalRequests = 0

	lb.nodes[node.NodeID] = node

	lb.metrics.mu.Lock()
	lb.metrics.NodeMetrics[node.NodeID] = &LBNodeMetric{
		NodeID: node.NodeID,
	}
	lb.metrics.mu.Unlock()

	if lb.redisClient != nil {
		data, _ := json.Marshal(node)
		key := fmt.Sprintf("edge:lb:node:%s", node.NodeID)
		ctx := context.Background()
		lb.redisClient.Set(ctx, key, data, 24*time.Hour)
	}

	atomic.AddInt64(&lb.version, 1)
	return nil
}

func (lb *EdgeLoadBalancer) RemoveNode(nodeID string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if _, exists := lb.nodes[nodeID]; !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	delete(lb.nodes, nodeID)

	lb.metrics.mu.Lock()
	delete(lb.metrics.NodeMetrics, nodeID)
	lb.metrics.mu.Unlock()

	if lb.redisClient != nil {
		ctx := context.Background()
		lb.redisClient.Del(ctx, fmt.Sprintf("edge:lb:node:%s", nodeID))
	}

	atomic.AddInt64(&lb.version, 1)
	return nil
}

func (lb *EdgeLoadBalancer) UpdateNode(nodeID string, updates *LBNode) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	node, exists := lb.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	node.Weight = updates.Weight
	node.LatencyMs = updates.LatencyMs
	node.CPUUsage = updates.CPUUsage
	node.MemoryUsage = updates.MemoryUsage
	node.Healthy = updates.Healthy
	node.LastHeartbeat = time.Now()

	atomic.AddInt64(&lb.version, 1)
	return nil
}

func (lb *EdgeLoadBalancer) GetNode(nodeID string) (*LBNode, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	node, exists := lb.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	nodeCopy := *node
	return &nodeCopy, nil
}

func (lb *EdgeLoadBalancer) ListNodes() []*LBNode {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	var nodes []*LBNode
	for _, node := range lb.nodes {
		nodeCopy := *node
		nodes = append(nodes, &nodeCopy)
	}
	return nodes
}

func (lb *EdgeLoadBalancer) SelectNode(req *LBRequest) (*LBNode, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	var policy *LBPolicy
	if req.Policy != "" {
		policy = lb.policies[req.Policy]
	}
	if policy == nil {
		policy = lb.config.Policy
	}

	if !policy.Enabled {
		return nil, fmt.Errorf("policy is disabled")
	}

	var healthyNodes []*LBNode
	for _, node := range lb.nodes {
		if node.Healthy {
			healthyNodes = append(healthyNodes, node)
		}
	}

	if len(healthyNodes) == 0 {
		return nil, fmt.Errorf("no healthy nodes available")
	}

	if lb.config.StickySessions && req.ClientIP != "" {
		if nodeID, exists := lb.getSessionAffinity(req.ClientIP); exists {
			if node, ok := lb.nodes[nodeID]; ok && node.Healthy {
				return node, nil
			}
		}
	}

	var selectedNode *LBNode

	switch policy.Algorithm {
	case LBAlgorithmRoundRobin:
		selectedNode = lb.selectRoundRobin(healthyNodes)
	case LBAlgorithmLeastConn:
		selectedNode = lb.selectLeastConnection(healthyNodes)
	case LBAlgorithmIPHash:
		selectedNode = lb.selectIPHash(healthyNodes, req.ClientIP)
	case LBAlgorithmWeighted:
		selectedNode = lb.selectWeighted(healthyNodes, policy)
	case LBAlgorithmLatency:
		selectedNode = lb.selectLatencyBased(healthyNodes)
	case LBAlgorithmResource:
		selectedNode = lb.selectResourceBased(healthyNodes)
	case LBAlgorithmGeolocation:
		selectedNode = lb.selectGeolocation(healthyNodes, req)
	case LBAlgorithmAdaptive:
		selectedNode = lb.selectAdaptive(healthyNodes, policy)
	default:
		selectedNode = lb.selectRoundRobin(healthyNodes)
	}

	if selectedNode == nil {
		return nil, fmt.Errorf("no suitable node found")
	}

	lb.setSessionAffinity(req.ClientIP, selectedNode.NodeID)

	lb.updateAlgorithmMetrics(policy.Algorithm)

	return selectedNode, nil
}

func (lb *EdgeLoadBalancer) selectRoundRobin(nodes []*LBNode) *LBNode {
	count := atomic.AddInt64(&lb.requestCount, 1)
	index := count % int64(len(nodes))
	return nodes[index]
}

func (lb *EdgeLoadBalancer) selectLeastConnection(nodes []*LBNode) *LBNode {
	var minConns int32 = math.MaxInt32
	var selected *LBNode

	for _, node := range nodes {
		conns := atomic.LoadInt32(&node.CurrentConns)
		if conns < minConns {
			minConns = conns
			selected = node
		}
	}

	return selected
}

func (lb *EdgeLoadBalancer) selectIPHash(nodes []*LBNode, clientIP string) *LBNode {
	if clientIP == "" {
		return nodes[0]
	}

	var hash int64
	for _, c := range clientIP {
		hash = hash*31 + int64(c)
	}

	index := hash % int64(len(nodes))
	return nodes[index]
}

func (lb *EdgeLoadBalancer) selectWeighted(nodes []*LBNode, policy *LBPolicy) *LBNode {
	var totalWeight int
	for _, node := range nodes {
		weight := node.Weight
		if w, ok := policy.Weights[node.NodeID]; ok {
			weight = w
		}
		totalWeight += weight
	}

	if totalWeight == 0 {
		return nodes[0]
	}

	r := time.Now().UnixNano() % int64(totalWeight)
	var cumulative int

	for _, node := range nodes {
		weight := node.Weight
		if w, ok := policy.Weights[node.NodeID]; ok {
			weight = w
		}
		cumulative += weight
		if int64(cumulative) > r {
			return node
		}
	}

	return nodes[0]
}

func (lb *EdgeLoadBalancer) selectLatencyBased(nodes []*LBNode) *LBNode {
	var minLatency float64 = math.MaxFloat64
	var selected *LBNode

	for _, node := range nodes {
		if node.LatencyMs < minLatency {
			minLatency = node.LatencyMs
			selected = node
		}
	}

	return selected
}

func (lb *EdgeLoadBalancer) selectResourceBased(nodes []*LBNode) *LBNode {
	var bestScore float64 = -1
	var selected *LBNode

	for _, node := range nodes {
		cpuScore := 1.0 - node.CPUUsage
		memScore := 1.0 - node.MemoryUsage
		score := (cpuScore + memScore) / 2

		conns := float64(atomic.LoadInt32(&node.CurrentConns))
		connScore := 1.0 - (conns / float64(lb.config.MaxConnections))
		score = score*0.7 + connScore*0.3

		if score > bestScore {
			bestScore = score
			selected = node
		}
	}

	return selected
}

func (lb *EdgeLoadBalancer) selectGeolocation(nodes []*LBNode, req *LBRequest) *LBNode {
	if lb.nodeManager == nil {
		return nodes[0]
	}

	geoLoc, err := lb.nodeManager.ResolveIPRegion(req.ClientIP)
	if err != nil {
		return nodes[0]
	}

	var bestMatch *LBNode
	var minDistance float64 = math.MaxFloat64

	for _, node := range nodes {
		distance := lb.nodeManager.CalculateDistance(
			geoLoc.Latitude, geoLoc.Longitude,
			0, 0,
		)

		if node.Region == geoLoc.Region {
			distance *= 0.5
		}

		if distance < minDistance {
			minDistance = distance
			bestMatch = node
		}
	}

	if bestMatch != nil {
		return bestMatch
	}

	return nodes[0]
}

func (lb *EdgeLoadBalancer) selectAdaptive(nodes []*LBNode, policy *LBPolicy) *LBNode {
	var bestScore float64 = -1
	var selected *LBNode

	for _, node := range nodes {
		latencyScore := 1.0 - (node.LatencyMs / 500.0)
		if latencyScore < 0 {
			latencyScore = 0
		}

		loadScore := 1.0 - node.CPUUsage
		connsScore := 1.0 - (float64(atomic.LoadInt32(&node.CurrentConns)) / float64(lb.config.MaxConnections))

		var weightScore float64 = 1.0
		if w, ok := policy.Weights[node.NodeID]; ok {
			weightScore = float64(w) / 100.0
		}

		score := (latencyScore*0.4 + loadScore*0.3 + connsScore*0.2 + weightScore*0.1) * 100

		if score > bestScore {
			bestScore = score
			selected = node
		}
	}

	return selected
}

func (lb *EdgeLoadBalancer) getSessionAffinity(clientIP string) (string, bool) {
	lb.affinityMu.RLock()
	defer lb.affinityMu.RUnlock()

	nodeID, exists := lb.affinityMap[clientIP]
	return nodeID, exists
}

func (lb *EdgeLoadBalancer) setSessionAffinity(clientIP, nodeID string) {
	if !lb.config.StickySessions || clientIP == "" {
		return
	}

	lb.affinityMu.Lock()
	defer lb.affinityMu.Unlock()

	lb.affinityMap[clientIP] = nodeID
}

func (lb *EdgeLoadBalancer) updateAlgorithmMetrics(algorithm LBAlgorithm) {
	lb.metrics.mu.Lock()
	defer lb.metrics.mu.Unlock()

	if _, exists := lb.metrics.AlgorithmMetrics[algorithm]; !exists {
		lb.metrics.AlgorithmMetrics[algorithm] = &AlgorithmMetric{Algorithm: algorithm}
	}

	metric := lb.metrics.AlgorithmMetrics[algorithm]
	atomic.AddInt64(&metric.SelectionCount, 1)
}

func (lb *EdgeLoadBalancer) RecordRequest(nodeID string, latencyMs float64, success bool) {
	lb.mu.RLock()
	node, exists := lb.nodes[nodeID]
	lb.mu.RUnlock()

	if !exists {
		return
	}

	if success {
		atomic.AddInt64(&node.SuccessCount, 1)
	} else {
		node.FailCount++
	}
	atomic.AddInt64(&node.TotalRequests, 1)

	lb.metrics.mu.Lock()
	defer lb.metrics.mu.Unlock()

	atomic.AddInt64(&lb.metrics.TotalRequests, 1)
	lb.metrics.TotalLatencyMs += latencyMs
	avgRequests := float64(atomic.LoadInt64(&lb.metrics.TotalRequests))
	if avgRequests > 0 {
		lb.metrics.AvgLatencyMs = lb.metrics.TotalLatencyMs / avgRequests
	}

	if latencyMs < lb.metrics.MinLatencyMs || lb.metrics.MinLatencyMs == 0 {
		lb.metrics.MinLatencyMs = latencyMs
	}
	if latencyMs > lb.metrics.MaxLatencyMs {
		lb.metrics.MaxLatencyMs = latencyMs
	}

	if nodeMetric, exists := lb.metrics.NodeMetrics[nodeID]; exists {
		nodeMetric.RequestCount++
		nodeMetric.TotalLatencyMs += latencyMs
		if nodeMetric.RequestCount > 0 {
			nodeMetric.AvgLatencyMs = nodeMetric.TotalLatencyMs / float64(nodeMetric.RequestCount)
		}
		if success {
			nodeMetric.SuccessRate = float64(nodeMetric.RequestCount-nodeMetric.ErrorCount) / float64(nodeMetric.RequestCount)
		}
	}
}

func (lb *EdgeLoadBalancer) ForwardRequest(ctx context.Context, req *LBRequest) (*LBResponse, error) {
	response := &LBResponse{
		RequestID: fmt.Sprintf("req-%d", time.Now().UnixNano()),
		Attempts:  0,
		Timestamp: time.Now(),
	}

	var lastErr error
	maxAttempts := lb.config.Policy.RetryAttempts

	for attempt := 1; attempt <= maxAttempts+1; attempt++ {
		response.Attempts = attempt

		node, err := lb.SelectNode(req)
		if err != nil {
			response.Error = err.Error()
			return response, err
		}

		response.NodeID = node.NodeID
		response.IPAddress = node.IPAddress
		response.Port = node.Port

		atomic.AddInt32(&node.ActiveRequests, 1)
		defer atomic.AddInt32(&node.ActiveRequests, -1)

		startTime := time.Now()

		err = lb.forwardToNode(ctx, node, req)

		latencyMs := float64(time.Since(startTime).Milliseconds())
		response.LatencyMs = latencyMs

		if err == nil {
			response.Success = true
			lb.RecordRequest(node.NodeID, latencyMs, true)
			return response, nil
		}

		lastErr = err
		lb.RecordRequest(node.NodeID, latencyMs, false)

		if attempt <= maxAttempts {
			atomic.AddInt64(&lb.metrics.RetriedRequests, 1)
		}

		if !lb.config.Policy.Failover {
			break
		}
	}

	response.Success = false
	response.Error = lastErr.Error()
	atomic.AddInt64(&lb.metrics.FailedRequests, 1)

	return response, lastErr
}

func (lb *EdgeLoadBalancer) forwardToNode(ctx context.Context, node *LBNode, req *LBRequest) error {
	atomic.AddInt32(&node.CurrentConns, 1)
	defer atomic.AddInt32(&node.CurrentConns, -1)

	return nil
}

func (lb *EdgeLoadBalancer) AddPolicy(policy *LBPolicy) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if policy.ID == "" {
		return fmt.Errorf("policy ID is required")
	}

	policy.Enabled = true
	lb.policies[policy.ID] = policy
	atomic.AddInt64(&lb.version, 1)

	return nil
}

func (lb *EdgeLoadBalancer) DeletePolicy(policyID string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if _, exists := lb.policies[policyID]; !exists {
		return fmt.Errorf("policy not found: %s", policyID)
	}

	if policyID == "default" {
		return fmt.Errorf("cannot delete default policy")
	}

	delete(lb.policies, policyID)
	atomic.AddInt64(&lb.version, 1)

	return nil
}

func (lb *EdgeLoadBalancer) UpdatePolicy(policy *LBPolicy) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	existing, exists := lb.policies[policy.ID]
	if !exists {
		return fmt.Errorf("policy not found: %s", policy.ID)
	}

	existing.Name = policy.Name
	existing.Algorithm = policy.Algorithm
	existing.Mode = policy.Mode
	existing.HealthCheck = policy.HealthCheck
	existing.Failover = policy.Failover
	existing.RetryAttempts = policy.RetryAttempts
	existing.Timeout = policy.Timeout
	existing.Weights = policy.Weights
	existing.RegionWeights = policy.RegionWeights
	existing.Enabled = policy.Enabled
	atomic.AddInt64(&lb.version, 1)

	return nil
}

func (lb *EdgeLoadBalancer) GetPolicy(policyID string) (*LBPolicy, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	policy, exists := lb.policies[policyID]
	if !exists {
		return nil, fmt.Errorf("policy not found: %s", policyID)
	}

	policyCopy := *policy
	policyCopy.Weights = make(map[string]int)
	for k, v := range policy.Weights {
		policyCopy.Weights[k] = v
	}
	policyCopy.RegionWeights = make(map[Region]int)
	for k, v := range policy.RegionWeights {
		policyCopy.RegionWeights[k] = v
	}

	return &policyCopy, nil
}

func (lb *EdgeLoadBalancer) ListPolicies() []*LBPolicy {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	var policies []*LBPolicy
	for _, policy := range lb.policies {
		policyCopy := *policy
		policies = append(policies, &policyCopy)
	}
	return policies
}

func (lb *EdgeLoadBalancer) SetConfig(config *LoadBalancerConfig) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.config = config
}

func (lb *EdgeLoadBalancer) GetConfig() *LoadBalancerConfig {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	configCopy := *lb.config
	return &configCopy
}

func (lb *EdgeLoadBalancer) GetMetrics() *LoadBalancerMetrics {
	lb.metrics.mu.RLock()
	defer lb.metrics.mu.RUnlock()

	metricsCopy := &LoadBalancerMetrics{
		TotalRequests:   atomic.LoadInt64(&lb.metrics.TotalRequests),
		TotalResponses:  atomic.LoadInt64(&lb.metrics.TotalResponses),
		ActiveConnections: atomic.LoadInt64(&lb.metrics.ActiveConnections),
		FailedRequests:  atomic.LoadInt64(&lb.metrics.FailedRequests),
		RetriedRequests: atomic.LoadInt64(&lb.metrics.RetriedRequests),
		AvgLatencyMs:   lb.metrics.AvgLatencyMs,
		MinLatencyMs:   lb.metrics.MinLatencyMs,
		MaxLatencyMs:   lb.metrics.MaxLatencyMs,
		TotalLatencyMs: lb.metrics.TotalLatencyMs,
		NodeMetrics:    make(map[string]*LBNodeMetric),
		AlgorithmMetrics: make(map[LBAlgorithm]*AlgorithmMetric),
	}

	for k, v := range lb.metrics.NodeMetrics {
		metricCopy := *v
		metricsCopy.NodeMetrics[k] = &metricCopy
	}

	for k, v := range lb.metrics.AlgorithmMetrics {
		metricCopy := *v
		metricsCopy.AlgorithmMetrics[k] = &metricCopy
	}

	return metricsCopy
}

func (lb *EdgeLoadBalancer) GetVersion() int64 {
	return atomic.LoadInt64(&lb.version)
}

func (lb *EdgeLoadBalancer) SyncToRedis(ctx context.Context) error {
	if lb.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	lb.mu.RLock()
	defer lb.mu.RUnlock()

	data, err := json.Marshal(lb.nodes)
	if err != nil {
		return err
	}

	if err := lb.redisClient.Set(ctx, "edge:lb:nodes", data, 24*time.Hour).Err(); err != nil {
		return err
	}

	policyData, err := json.Marshal(lb.policies)
	if err != nil {
		return err
	}

	return lb.redisClient.Set(ctx, "edge:lb:policies", policyData, 24*time.Hour).Err()
}

func (lb *EdgeLoadBalancer) SyncFromRedis(ctx context.Context) error {
	if lb.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	nodeData, err := lb.redisClient.Get(ctx, "edge:lb:nodes").Bytes()
	if err == nil {
		var nodes map[string]*LBNode
		if err := json.Unmarshal(nodeData, &nodes); err == nil {
			lb.mu.Lock()
			lb.nodes = nodes
			lb.mu.Unlock()
		}
	}

	policyData, err := lb.redisClient.Get(ctx, "edge:lb:policies").Bytes()
	if err == nil {
		var policies map[string]*LBPolicy
		if err := json.Unmarshal(policyData, &policies); err == nil {
			lb.mu.Lock()
			lb.policies = policies
			lb.mu.Unlock()
		}
	}

	return nil
}
