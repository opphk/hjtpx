package performance

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type GlobalEdgeNode struct {
	ID              string
	Name            string
	Address         string
	Port            int
	Region          string
	Zone            string
	Continent       string
	Country         string
	City            string
	Latitude        float64
	Longitude       float64
	Capacity        int
	CurrentLoad     int32
	ActiveConns     int32
	Healthy         atomic.Bool
	Latency         atomic.Int64
	LastHeartbeat   atomic.Int64
	Priority        int
	Weight          int
	Features        []string
	Protocols       []string
	BandwidthGbps   int
	AvailableCPU    float64
	AvailableMemory int64
	CreatedAt       time.Time
	Stats           *EdgeNodeStats
	mu              sync.RWMutex
}

type EdgeNodeStats struct {
	TotalRequests      atomic.Int64
	SuccessfulRequests atomic.Int64
	FailedRequests     atomic.Int64
	CacheHits         atomic.Int64
	CacheMisses       atomic.Int64
	BytesTransferred  atomic.Int64
	AvgLatencyMs      atomic.Int64
	P99LatencyMs      atomic.Int64
	ActiveUsers       atomic.Int64
}

type GlobalEdgeNetwork struct {
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	isRunning   bool
	nodes       map[string]*GlobalEdgeNode
	regions     map[string]*EdgeRegion
	clusters    map[string]*EdgeCluster
	strategy    *RoutingStrategy
	loadBalance *EdgeLoadBalancer
	healthCheck *HealthChecker
	syncMgr     *DataSyncManager
	failover    *EdgeFailoverManager
	metrics     *EdgeNetworkMetrics
	cache       *EdgeCache
}

type EdgeRegion struct {
	ID              string
	Name            string
	Code            string
	Continent       string
	Nodes           map[string]*GlobalEdgeNode
	PrimaryNode     *GlobalEdgeNode
	ActiveNodes     int32
	TotalCapacity   int32
	TotalLoad       int32
	AvgLatencyMs    int64
	Healthy         bool
	FailoverTarget  string
}

type EdgeCluster struct {
	ID           string
	Name         string
	Region       string
	Nodes        map[string]*GlobalEdgeNode
	MasterNode   *GlobalEdgeNode
	SyncEnabled  bool
	LoadBalance  bool
}

type RoutingStrategy struct {
	mu          sync.RWMutex
	primary     string
	fallbacks   []string
	geoEnabled  bool
	loadEnabled bool
	latencyEnabled bool
}

type EdgeLoadBalancer struct {
	mu          sync.RWMutex
	algorithm   string
	nodes       []*GlobalEdgeNode
	index       uint32
	weights     map[string]int
}

type HealthChecker struct {
	mu           sync.RWMutex
	interval     time.Duration
	timeout      time.Duration
	thresholds   *HealthThresholds
	healthyNodes map[string]bool
}

type HealthThresholds struct {
	MaxLatencyMs       int64
	MaxFailureRate     float64
	MinHealthyRatio    float64
	MaxLoadPercent     int
}

type DataSyncManager struct {
	mu             sync.RWMutex
	interval       time.Duration
	syncType       string
	compression    bool
	encryption     bool
	lastSyncTime   atomic.Int64
	pendingItems   atomic.Int32
	syncedItems    atomic.Int64
	syncErrors     atomic.Int32
}

type EdgeFailoverManager struct {
	mu             sync.RWMutex
	autoFailover   bool
	failoverCount  atomic.Int32
	recoveryCount  atomic.Int32
	failovers      map[string]*EdgeFailover
}

type EdgeFailover struct {
	FailedNodeID    string
	BackupNodeID    string
	FailoverTime    time.Time
	RecoveryTime    *time.Time
	Status          string
	Reason          string
}

type EdgeNetworkMetrics struct {
	TotalRequests       atomic.Int64
	SuccessfulRequests  atomic.Int64
	FailedRequests      atomic.Int64
	CrossRegionRequests atomic.Int64
	CacheHits          atomic.Int64
	CacheMisses        atomic.Int64
	FailoverEvents     atomic.Int64
	RecoveryEvents     atomic.Int64
	AvgLatencyMs       atomic.Int64
	P99LatencyMs       atomic.Int64
	ActiveNodes        atomic.Int64
	HealthyRegions     atomic.Int64
	AvgNodeLoad        atomic.Int64
	NetworkBandwidth   atomic.Int64
	LastUpdate         atomic.Value
}

type EdgeCache struct {
	mu        sync.RWMutex
	items     map[string]*CacheItem
	maxSize   int
	eviction  string
	hits      atomic.Int64
	misses    atomic.Int64
}

type CacheItem struct {
	Key        string
	Value      []byte
	ExpiresAt  time.Time
	NodeID     string
	Region     string
	AccessCount atomic.Int32
	LastAccess atomic.Int64
}

type EdgeVerificationRequest struct {
	RequestID     string
	UserID        string
	UserIP        string
	UserLocation  *GeoLocation
	Data          []byte
	UseCache      bool
	CacheKey      string
	TTL           time.Duration
	Priority      int
	Timeout       time.Duration
	AllowedRegions []string
}

type EdgeVerificationResponse struct {
	Success       bool
	RequestID     string
	NodeID        string
	NodeName      string
	Region        string
	Result        []byte
	FromCache     bool
	LatencyMs     int64
	Error         string
	Retryable     bool
}

type NodeRegistration struct {
	Name          string
	Address       string
	Port          int
	Region        string
	Zone          string
	Continent     string
	Country       string
	City          string
	Latitude      float64
	Longitude     float64
	Capacity      int
	Priority      int
	Weight        int
	Features      []string
	Protocols     []string
	BandwidthGbps int
}

const (
	StrategyGeoBased      = "geo_based"
	StrategyLeastLoad     = "least_load"
	StrategyLatencyBased  = "latency_based"
	StrategyWeighted      = "weighted"
	StrategyFailover      = "failover"

	AlgorithmRoundRobin   = "round_robin"
	AlgorithmLeastConn    = "least_conn"
	AlgorithmWeightedRR   = "weighted_rr"
	AlgorithmLatency      = "latency"

	SyncTypeFull          = "full"
	SyncTypeIncremental   = "incremental"
	SyncTypeAsync         = "async"

	EvictionLRU           = "lru"
	EvictionLFU           = "lfu"
	EvictionTTL           = "ttl"
)

func NewGlobalEdgeNetwork() *GlobalEdgeNetwork {
	ctx, cancel := context.WithCancel(context.Background())

	return &GlobalEdgeNetwork{
		ctx:       ctx,
		cancel:    cancel,
		nodes:     make(map[string]*GlobalEdgeNode),
		regions:   make(map[string]*EdgeRegion),
		clusters:  make(map[string]*EdgeCluster),
		strategy:  NewRoutingStrategy(),
		loadBalance: NewEdgeLoadBalancer(),
		healthCheck: NewHealthChecker(),
		syncMgr:   NewDataSyncManager(),
		failover:  NewEdgeFailoverManager(),
		metrics:   &EdgeNetworkMetrics{},
		cache:     NewEdgeCache(10000),
	}
}

func NewGlobalEdgeNode(id string, reg *NodeRegistration) *GlobalEdgeNode {
	return &GlobalEdgeNode{
		ID:             id,
		Name:           reg.Name,
		Address:        reg.Address,
		Port:           reg.Port,
		Region:         reg.Region,
		Zone:           reg.Zone,
		Continent:      reg.Continent,
		Country:        reg.Country,
		City:           reg.City,
		Latitude:       reg.Latitude,
		Longitude:      reg.Longitude,
		Capacity:       reg.Capacity,
		CurrentLoad:    0,
		Healthy:        atomic.Bool{},
		Priority:       reg.Priority,
		Weight:         reg.Weight,
		Features:       reg.Features,
		Protocols:      reg.Protocols,
		BandwidthGbps:  reg.BandwidthGbps,
		CreatedAt:      time.Now(),
		Stats:          &EdgeNodeStats{},
	}
}

func (n *GlobalEdgeNode) IsHealthy() bool {
	return n.Healthy.Load()
}

func (n *GlobalEdgeNode) UpdateHeartbeat() {
	n.LastHeartbeat.Store(time.Now().Unix())
}

func (n *GlobalEdgeNode) SetHealthy(healthy bool) {
	n.Healthy.Store(healthy)
}

func (n *GlobalEdgeNode) GetLoadPercent() int {
	if n.Capacity == 0 {
		return 100
	}
	return int(float64(n.CurrentLoad) / float64(n.Capacity) * 100)
}

func (n *GlobalEdgeNode) CanAcceptRequest() bool {
	return n.IsHealthy() && n.GetLoadPercent() < 95
}

func NewEdgeRegion(id, name, code, continent string) *EdgeRegion {
	return &EdgeRegion{
		ID:     id,
		Name:   name,
		Code:   code,
		Continent: continent,
		Nodes:  make(map[string]*GlobalEdgeNode),
		Healthy: true,
	}
}

func NewRoutingStrategy() *RoutingStrategy {
	return &RoutingStrategy{
		primary:        StrategyGeoBased,
		fallbacks:      []string{StrategyLeastLoad, StrategyLatencyBased},
		geoEnabled:     true,
		loadEnabled:   true,
		latencyEnabled: true,
	}
}

func NewEdgeLoadBalancer() *EdgeLoadBalancer {
	return &EdgeLoadBalancer{
		algorithm: AlgorithmLeastConn,
		weights:   make(map[string]int),
	}
}

func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		interval: 10 * time.Second,
		timeout:  3 * time.Second,
		thresholds: &HealthThresholds{
			MaxLatencyMs:    100,
			MaxFailureRate:  0.05,
			MinHealthyRatio: 0.8,
			MaxLoadPercent:  90,
		},
		healthyNodes: make(map[string]bool),
	}
}

func NewDataSyncManager() *DataSyncManager {
	return &DataSyncManager{
		interval:    30 * time.Second,
		syncType:    SyncTypeIncremental,
		compression: true,
		encryption:  true,
	}
}

func NewEdgeFailoverManager() *EdgeFailoverManager {
	return &EdgeFailoverManager{
		autoFailover: true,
		failovers:    make(map[string]*EdgeFailover),
	}
}

func NewEdgeCache(maxSize int) *EdgeCache {
	return &EdgeCache{
		items:    make(map[string]*CacheItem),
		maxSize:  maxSize,
		eviction: EvictionLRU,
	}
}

func (n *GlobalEdgeNetwork) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.isRunning {
		return nil
	}

	n.isRunning = true

	go n.healthCheck.runHealthChecks(n.ctx, n)
	go n.syncMgr.runSync(n.ctx, n)
	go n.failover.runFailoverMonitor(n.ctx, n)
	go n.collectMetrics()
	go n.cleanupCache()

	log.Println("[GlobalEdgeNetwork] Started successfully")
	return nil
}

func (n *GlobalEdgeNetwork) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.isRunning {
		return
	}

	n.cancel()
	n.isRunning = false
	log.Println("[GlobalEdgeNetwork] Stopped")
}

func (n *GlobalEdgeNetwork) RegisterNode(ctx context.Context, reg *NodeRegistration) (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	nodeID := generateEdgeNodeID()
	node := NewGlobalEdgeNode(nodeID, reg)
	node.Healthy.Store(true)
	node.Stats = &EdgeNodeStats{}
	node.CreatedAt = time.Now()
	node.LastHeartbeat.Store(time.Now().Unix())

	n.nodes[nodeID] = node
	n.updateRegionForNode(node)
	n.loadBalance.addNode(node)

	n.metrics.ActiveNodes.Add(1)
	log.Printf("[GlobalEdgeNetwork] Registered node: %s in region %s, continent %s", nodeID, reg.Region, reg.Continent)

	return nodeID, nil
}

func (n *GlobalEdgeNetwork) DeregisterNode(ctx context.Context, nodeID string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	node, exists := n.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	n.removeNodeFromRegion(node)
	delete(n.nodes, nodeID)
	n.loadBalance.removeNode(node)

	n.metrics.ActiveNodes.Add(-1)
	log.Printf("[GlobalEdgeNetwork] Deregistered node: %s", nodeID)
	return nil
}

func (n *GlobalEdgeNetwork) ProcessVerification(ctx context.Context, req *EdgeVerificationRequest) (*EdgeVerificationResponse, error) {
	n.metrics.TotalRequests.Add(1)
	start := time.Now()

	if req.UseCache && req.CacheKey != "" {
		if cached := n.cache.Get(req.CacheKey); cached != nil {
			n.metrics.CacheHits.Add(1)
			return &EdgeVerificationResponse{
				Success:   true,
				RequestID: req.RequestID,
				FromCache: true,
				LatencyMs: time.Since(start).Milliseconds(),
				Result:    cached.Value,
				NodeID:    cached.NodeID,
				Region:    cached.Region,
			}, nil
		}
		n.metrics.CacheMisses.Add(1)
	}

	var node *GlobalEdgeNode
	var err error

	if len(req.AllowedRegions) > 0 {
		node, err = n.selectNodeForRegions(ctx, req.AllowedRegions, req.UserLocation)
	} else {
		node, err = n.selectOptimalNode(ctx, req.UserLocation)
	}

	if err != nil {
		if failoverNode := n.failover.getAvailableBackup(ctx, n); failoverNode != nil {
			node = failoverNode
			n.metrics.FailoverEvents.Add(1)
		} else {
			n.metrics.FailedRequests.Add(1)
			return nil, fmt.Errorf("no available nodes: %w", err)
		}
	}

	response := n.processOnNode(ctx, req, node)
	response.LatencyMs = time.Since(start).Milliseconds()

	if response.Success {
		n.metrics.SuccessfulRequests.Add(1)
	} else {
		n.metrics.FailedRequests.Add(1)
	}

	return response, nil
}

func (n *GlobalEdgeNetwork) selectOptimalNode(ctx context.Context, location *GeoLocation) (*GlobalEdgeNode, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if len(n.nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}

	var candidates []*GlobalEdgeNode
	for _, node := range n.nodes {
		if node.CanAcceptRequest() {
			candidates = append(candidates, node)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no healthy nodes with capacity")
	}

	if location != nil && n.strategy.geoEnabled {
		return n.selectGeoOptimalNode(candidates, location)
	}

	if n.strategy.loadEnabled {
		return n.selectLeastLoadedNode(candidates)
	}

	return candidates[0], nil
}

func (n *GlobalEdgeNetwork) selectGeoOptimalNode(nodes []*GlobalEdgeNode, location *GeoLocation) (*GlobalEdgeNode, error) {
	var bestNode *GlobalEdgeNode
	minDistance := math.MaxFloat64

	for _, node := range nodes {
		if node.Region == location.Region || node.Country == location.Country {
			load := float64(node.GetLoadPercent())
			if bestNode == nil || load < minDistance {
				bestNode = node
				minDistance = load
			}
		}
	}

	if bestNode != nil {
		return bestNode, nil
	}

	return n.selectLeastLoadedNode(nodes)
}

func (n *GlobalEdgeNetwork) selectLeastLoadedNode(nodes []*GlobalEdgeNode) (*GlobalEdgeNode, error) {
	var bestNode *GlobalEdgeNode
	minLoad := math.MaxInt32

	for _, node := range nodes {
		load := node.GetLoadPercent()
		if load < minLoad {
			minLoad = load
			bestNode = node
		}
	}

	return bestNode, nil
}

func (n *GlobalEdgeNetwork) selectNodeForRegions(ctx context.Context, regions []string, location *GeoLocation) (*GlobalEdgeNode, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, region := range regions {
		if r, exists := n.regions[region]; exists && r.Healthy {
			for _, node := range r.Nodes {
				if node.CanAcceptRequest() {
					return node, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no available nodes in allowed regions")
}

func (n *GlobalEdgeNetwork) processOnNode(ctx context.Context, req *EdgeVerificationRequest, node *GlobalEdgeNode) *EdgeVerificationResponse {
	atomic.AddInt32(&node.CurrentLoad, 1)
	defer atomic.AddInt32(&node.CurrentLoad, -1)

	node.Stats.TotalRequests.Add(1)

	result := req.Data
	if result == nil {
		result = []byte(fmt.Sprintf(`{"node":"%s","region":"%s","verified":true}`, node.ID, node.Region))
	}

	response := &EdgeVerificationResponse{
		Success:   true,
		RequestID: req.RequestID,
		NodeID:    node.ID,
		NodeName:  node.Name,
		Region:    node.Region,
		Result:    result,
		FromCache: false,
	}

	if req.UseCache && req.CacheKey != "" {
		n.cache.Set(req.CacheKey, result, req.TTL, node.ID, node.Region)
	}

	return response
}

func (n *GlobalEdgeNetwork) GetNodeStats(ctx context.Context, nodeID string) (map[string]interface{}, error) {
	n.mu.RLock()
	node, exists := n.nodes[nodeID]
	n.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	return map[string]interface{}{
		"node_id":          node.ID,
		"name":             node.Name,
		"region":           node.Region,
		"healthy":          node.IsHealthy(),
		"capacity":         node.Capacity,
		"current_load":     node.CurrentLoad,
		"load_percent":     node.GetLoadPercent(),
		"latency_ms":       node.Latency.Load(),
		"total_requests":   node.Stats.TotalRequests.Load(),
		"cache_hits":       node.Stats.CacheHits.Load(),
		"cache_misses":     node.Stats.CacheMisses.Load(),
		"avg_latency_ms":   node.Stats.AvgLatencyMs.Load(),
	}, nil
}

func (n *GlobalEdgeNetwork) GetNetworkStats(ctx context.Context) map[string]interface{} {
	regionStats := make(map[string]interface{})
	
	n.mu.RLock()
	for regionID, region := range n.regions {
		regionStats[regionID] = map[string]interface{}{
			"name":          region.Name,
			"active_nodes":  region.ActiveNodes,
			"total_capacity": region.TotalCapacity,
			"healthy":       region.Healthy,
		}
	}
	n.mu.RUnlock()

	return map[string]interface{}{
		"total_requests":    n.metrics.TotalRequests.Load(),
		"successful_requests": n.metrics.SuccessfulRequests.Load(),
		"failed_requests":    n.metrics.FailedRequests.Load(),
		"cache_hits":         n.metrics.CacheHits.Load(),
		"cache_misses":       n.metrics.CacheMisses.Add(0),
		"failover_events":    n.metrics.FailoverEvents.Load(),
		"avg_latency_ms":     n.metrics.AvgLatencyMs.Load(),
		"active_nodes":       n.metrics.ActiveNodes.Load(),
		"healthy_regions":    n.metrics.HealthyRegions.Load(),
		"region_stats":       regionStats,
	}
}

func (n *GlobalEdgeNetwork) UpdateNodeHealth(nodeID string, healthy bool) {
	n.mu.RLock()
	node, exists := n.nodes[nodeID]
	n.mu.RUnlock()

	if exists {
		node.SetHealthy(healthy)
		node.UpdateHeartbeat()
		
		if !healthy {
			n.failover.triggerFailover(n, nodeID)
		}
	}
}

func (n *GlobalEdgeNetwork) updateRegionForNode(node *GlobalEdgeNode) {
	region, exists := n.regions[node.Region]
	if !exists {
		region = NewEdgeRegion(node.Region, node.Region, node.Region, node.Continent)
		n.regions[node.Region] = region
	}

	region.Nodes[node.ID] = node
	region.ActiveNodes++
	region.TotalCapacity += int32(node.Capacity)

	if region.PrimaryNode == nil || node.Priority > region.PrimaryNode.Priority {
		region.PrimaryNode = node
	}
}

func (n *GlobalEdgeNetwork) removeNodeFromRegion(node *GlobalEdgeNode) {
	if region, exists := n.regions[node.Region]; exists {
		delete(region.Nodes, node.ID)
		region.ActiveNodes--
		region.TotalCapacity -= int32(node.Capacity)

		if region.PrimaryNode != nil && region.PrimaryNode.ID == node.ID {
			var newPrimary *GlobalEdgeNode
			for _, n := range region.Nodes {
				if newPrimary == nil || n.Priority > newPrimary.Priority {
					newPrimary = n
				}
			}
			region.PrimaryNode = newPrimary
		}
	}
}

func (n *GlobalEdgeNetwork) collectMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var totalLoad int64
	var nodeCount int64

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.mu.RLock()
			healthyRegions := int64(0)
			totalLoad = 0
			nodeCount = 0

			for _, region := range n.regions {
				if region.Healthy && region.ActiveNodes > 0 {
					healthyRegions++
				}
			}

			for _, node := range n.nodes {
				totalLoad += int64(node.GetLoadPercent())
				nodeCount++
			}
			n.mu.RUnlock()

			if nodeCount > 0 {
				n.metrics.AvgNodeLoad.Store(totalLoad / nodeCount)
			}
			n.metrics.HealthyRegions.Store(healthyRegions)
			n.metrics.LastUpdate.Store(time.Now())
		}
	}
}

func (n *GlobalEdgeNetwork) cleanupCache() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.cache.Cleanup()
		}
	}
}

func (lb *EdgeLoadBalancer) addNode(node *GlobalEdgeNode) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.nodes = append(lb.nodes, node)
	lb.weights[node.ID] = node.Weight
}

func (lb *EdgeLoadBalancer) removeNode(node *GlobalEdgeNode) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, n := range lb.nodes {
		if n.ID == node.ID {
			lb.nodes = append(lb.nodes[:i], lb.nodes[i+1:]...)
			break
		}
	}
	delete(lb.weights, node.ID)
}

func (lb *EdgeLoadBalancer) SelectNode() *GlobalEdgeNode {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(lb.nodes) == 0 {
		return nil
	}

	switch lb.algorithm {
	case AlgorithmLeastConn:
		return lb.selectLeastConn()
	case AlgorithmWeightedRR:
		return lb.selectWeightedRR()
	default:
		return lb.selectRoundRobin()
	}
}

func (lb *EdgeLoadBalancer) selectLeastConn() *GlobalEdgeNode {
	var bestNode *GlobalEdgeNode
	minConn := math.MaxInt32

	for _, node := range lb.nodes {
		if node.CanAcceptRequest() {
			conns := atomic.LoadInt32(&node.ActiveConns)
			if int(conns) < minConn {
				minConn = int(conns)
				bestNode = node
			}
		}
	}

	return bestNode
}

func (lb *EdgeLoadBalancer) selectRoundRobin() *GlobalEdgeNode {
	lb.index++
	if int(lb.index) >= len(lb.nodes) {
		lb.index = 0
	}

	return lb.nodes[int(lb.index)]
}

func (lb *EdgeLoadBalancer) selectWeightedRR() *GlobalEdgeNode {
	totalWeight := 0
	for _, weight := range lb.weights {
		totalWeight += weight
	}

	if totalWeight == 0 {
		return lb.selectRoundRobin()
	}

	idx := int(lb.index) % totalWeight
	current := 0

	for _, node := range lb.nodes {
		current += lb.weights[node.ID]
		if idx < current && node.CanAcceptRequest() {
			lb.index++
			return node
		}
	}

	return lb.nodes[0]
}

func (hc *HealthChecker) runHealthChecks(ctx context.Context, n *GlobalEdgeNetwork) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.performHealthChecks(ctx, n)
		}
	}
}

func (hc *HealthChecker) performHealthChecks(ctx context.Context, n *GlobalEdgeNetwork) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	n.mu.RLock()
	for nodeID, node := range n.nodes {
		healthy := hc.checkNodeHealth(node)
		hc.healthyNodes[nodeID] = healthy
		node.SetHealthy(healthy)
	}
	n.mu.RUnlock()
}

func (hc *HealthChecker) checkNodeHealth(node *GlobalEdgeNode) bool {
	if node.GetLoadPercent() > hc.thresholds.MaxLoadPercent {
		return false
	}

	latency := node.Latency.Load()
	if latency > hc.thresholds.MaxLatencyMs {
		return false
	}

	lastHeartbeat := node.LastHeartbeat.Load()
	if time.Now().Unix()-lastHeartbeat > 30 {
		return false
	}

	return true
}

func (sm *DataSyncManager) runSync(ctx context.Context, n *GlobalEdgeNetwork) {
	ticker := time.NewTicker(sm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.performSync(ctx, n)
		}
	}
}

func (sm *DataSyncManager) performSync(ctx context.Context, n *GlobalEdgeNetwork) {
	sm.mu.RLock()
	syncType := sm.syncType
	sm.mu.RUnlock()

	n.mu.RLock()
	nodeCount := len(n.nodes)
	n.mu.RUnlock()

	if nodeCount == 0 {
		return
	}

	sm.pendingItems.Add(int32(nodeCount))
	sm.syncedItems.Add(int64(nodeCount))
	sm.lastSyncTime.Store(time.Now().Unix())

	if syncType == SyncTypeFull {
		log.Printf("[DataSyncManager] Performed full sync across %d nodes", nodeCount)
	} else {
		log.Printf("[DataSyncManager] Performed incremental sync across %d nodes", nodeCount)
	}
}

func (fm *EdgeFailoverManager) runFailoverMonitor(ctx context.Context, n *GlobalEdgeNetwork) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fm.checkAndRecover(ctx, n)
		}
	}
}

func (fm *EdgeFailoverManager) checkAndRecover(ctx context.Context, n *GlobalEdgeNetwork) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for nodeID, fo := range fm.failovers {
		if fo.Status == "active" && fo.RecoveryTime != nil && time.Now().After(*fo.RecoveryTime) {
			fm.recoverNode(ctx, n, nodeID)
		}
	}
}

func (fm *EdgeFailoverManager) triggerFailover(n *GlobalEdgeNetwork, failedNodeID string) {
	if !fm.autoFailover {
		return
	}

	fm.mu.Lock()
	defer fm.mu.Unlock()

	if _, exists := fm.failovers[failedNodeID]; exists {
		return
	}

	n.mu.RLock()
	var backup *GlobalEdgeNode
	for _, node := range n.nodes {
		if node.ID != failedNodeID && node.CanAcceptRequest() {
			backup = node
			break
		}
	}
	n.mu.RUnlock()

	if backup != nil {
		recoveryTime := time.Now().Add(5 * time.Minute)
		fm.failovers[failedNodeID] = &EdgeFailover{
			FailedNodeID: failedNodeID,
			BackupNodeID: backup.ID,
			FailoverTime: time.Now(),
			RecoveryTime: &recoveryTime,
			Status:       "active",
			Reason:       "node_unhealthy",
		}
		fm.failoverCount.Add(1)
	}
}

func (fm *EdgeFailoverManager) recoverNode(ctx context.Context, n *GlobalEdgeNetwork, nodeID string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fo, exists := fm.failovers[nodeID]; exists {
		fo.Status = "recovered"
		fm.recoveryCount.Add(1)
		delete(fm.failovers, nodeID)
	}
}

func (fm *EdgeFailoverManager) getAvailableBackup(ctx context.Context, n *GlobalEdgeNetwork) *GlobalEdgeNode {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	for _, fo := range fm.failovers {
		if fo.Status == "active" {
			n.mu.RLock()
			node := n.nodes[fo.BackupNodeID]
			n.mu.RUnlock()
			if node != nil && node.CanAcceptRequest() {
				return node
			}
		}
	}
	return nil
}

func (c *EdgeCache) Get(key string) *CacheItem {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		c.misses.Add(1)
		return nil
	}

	if time.Now().After(item.ExpiresAt) {
		delete(c.items, key)
		c.misses.Add(1)
		return nil
	}

	item.AccessCount.Add(1)
	item.LastAccess.Store(time.Now().Unix())
	c.hits.Add(1)
	return item
}

func (c *EdgeCache) Set(key string, value []byte, ttl time.Duration, nodeID, region string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.items) >= c.maxSize {
		c.evict()
	}

	c.items[key] = &CacheItem{
		Key:        key,
		Value:      value,
		ExpiresAt:  time.Now().Add(ttl),
		NodeID:     nodeID,
		Region:     region,
		AccessCount: atomic.Int32{},
		LastAccess: atomic.Int64{},
	}
}

func (c *EdgeCache) evict() {
	var oldestKey string
	var oldestTime int64 = math.MaxInt64

	for key, item := range c.items {
		if item.LastAccess.Load() < oldestTime {
			oldestTime = item.LastAccess.Load()
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

func (c *EdgeCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.ExpiresAt) {
			delete(c.items, key)
		}
	}
}

func generateEdgeNodeID() string {
	return fmt.Sprintf("edge_%d_%d", time.Now().Unix(), time.Now().UnixNano()%10000)
}
