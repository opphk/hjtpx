package edge

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type GlobalNetwork struct {
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	isRunning    bool

	nodes        map[string]*GlobalNode
	regions      map[string]*Region
	loadBalancer *GlobalLoadBalancer
	syncManager  *CrossRegionSync
	failover     *FailoverManager
	stats        *GlobalNetworkStats
}

type GlobalNode struct {
	ID           string
	Name         string
	Address      string
	Port         int
	Region       string
	Zone         string
	Continent    string
	Capacity     int
	CurrentLoad  int32
	Healthy      bool
	Latency      time.Duration
	Priority     int
	Features     []string
	CreatedAt    time.Time
	LastSync     time.Time
	Stats        *NodeStats
}

type Region struct {
	ID           string
	Name         string
	Code         string
	Continent    string
	Nodes        map[string]*GlobalNode
	ActiveNodes  int32
	TotalCapacity int32
	LatencyAvg   time.Duration
	Healthy      bool
}

type GlobalLoadBalancer struct {
	mu          sync.RWMutex
	strategy    string
	roundRobin  int32
	geolocation *GeoLocationService
}

type GeoLocationService struct {
	mu      sync.RWMutex
	cache   map[string]*GeoLocation
}

type GeoLocation struct {
	Country   string
	Region    string
	City      string
	Latitude  float64
	Longitude float64
	Timezone  string
}

type CrossRegionSync struct {
	mu            sync.RWMutex
	syncInterval  time.Duration
	lastSync      time.Time
	pendingSyncs  int32
	completedSyncs int64
	syncErrors    int64
}

type FailoverManager struct {
	mu              sync.RWMutex
	activeFailover  bool
	failoverCount  int32
	recoveryCount   int32
	failoverNodes   map[string]*FailoverNode
}

type FailoverNode struct {
	OriginalNodeID string
	BackupNodeID   string
	FailoverTime   time.Time
	RecoveryTime   *time.Time
	Status         string
}

type GlobalNetworkStats struct {
	TotalRequests     atomic.Int64
	RegionalRequests   map[string]*atomic.Int64
	CrossRegionRequests atomic.Int64
	CacheHits          atomic.Int64
	CacheMisses        atomic.Int64
	FailoverEvents     atomic.Int64
	RecoveryEvents     atomic.Int64
	AvgLatency         atomic.Int64
	P99Latency         atomic.Int64
	ActiveNodes        atomic.Int64
	HealthyRegions     atomic.Int64
	LastUpdate         atomic.Value
}

type VerificationRequest struct {
	RequestID   string
	UserID      string
	NodeID      string
	Region      string
	Data        []byte
	UseCache    bool
	CacheKey    string
	TTL         time.Duration
	Priority    int
}

type VerificationResponse struct {
	Success       bool
	RequestID    string
	NodeID       string
	Region       string
	Result       []byte
	FromCache    bool
	Latency      time.Duration
	Error        string
}

type NodeRegistrationRequest struct {
	Name        string
	Address     string
	Port        int
	Region      string
	Zone        string
	Continent   string
	Capacity    int
	Priority    int
	Features    []string
}

type NodeRegistrationResponse struct {
	Success bool
	NodeID  string
	Message string
}

type SyncRequest struct {
	NodeID       string
	Data         map[string]interface{}
	Timestamp    time.Time
	Checksum     string
}

type SyncResponse struct {
	Success     bool
	SyncedAt    time.Time
	DataVersion string
	Error       string
}

func NewGlobalNetwork() *GlobalNetwork {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &GlobalNetwork{
		ctx:             ctx,
		cancel:          cancel,
		nodes:           make(map[string]*GlobalNode),
		regions:         make(map[string]*Region),
		loadBalancer:    NewGlobalLoadBalancer(),
		syncManager:     NewCrossRegionSync(),
		failover:        NewFailoverManager(),
		stats:           &GlobalNetworkStats{},
	}
}

func NewGlobalLoadBalancer() *GlobalLoadBalancer {
	return &GlobalLoadBalancer{
		strategy:    "geo_least_load",
		geolocation: &GeoLocationService{cache: make(map[string]*GeoLocation)},
	}
}

func NewCrossRegionSync() *CrossRegionSync {
	return &CrossRegionSync{
		syncInterval: 30 * time.Second,
	}
}

func NewFailoverManager() *FailoverManager {
	return &FailoverManager{
		failoverNodes: make(map[string]*FailoverNode),
	}
}

func (g *GlobalNetwork) Initialize(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.isRunning {
		return nil
	}

	g.isRunning = true

	go g.healthMonitor()
	go g.syncManager.runSync(g.ctx, g)
	go g.failover.runFailoverMonitor(g.ctx, g)
	go g.collectStats()

	log.Println("[GlobalNetwork] Initialized successfully")
	return nil
}

func (g *GlobalNetwork) Shutdown() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.isRunning {
		return nil
	}

	g.cancel()
	g.isRunning = false
	log.Println("[GlobalNetwork] Shutdown complete")
	return nil
}

func (g *GlobalNetwork) RegisterNode(ctx context.Context, req *NodeRegistrationRequest) (*NodeRegistrationResponse, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	nodeID := generateGlobalNodeID()

	node := &GlobalNode{
		ID:          nodeID,
		Name:        req.Name,
		Address:     req.Address,
		Port:        req.Port,
		Region:      req.Region,
		Zone:        req.Zone,
		Continent:   req.Continent,
		Capacity:    req.Capacity,
		CurrentLoad: 0,
		Healthy:     true,
		Priority:    req.Priority,
		Features:    req.Features,
		CreatedAt:   time.Now(),
		LastSync:    time.Now(),
		Stats:       &NodeStats{},
	}

	g.nodes[nodeID] = node
	g.updateRegion(node)

	g.stats.ActiveNodes.Add(1)
	log.Printf("[GlobalNetwork] Registered node: %s in region %s", nodeID, req.Region)

	return &NodeRegistrationResponse{
		Success: true,
		NodeID:  nodeID,
		Message: "Node registered successfully",
	}, nil
}

func (g *GlobalNetwork) DeregisterNode(ctx context.Context, nodeID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	node, exists := g.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	g.removeFromRegion(node)
	delete(g.nodes, nodeID)
	g.stats.ActiveNodes.Add(-1)

	log.Printf("[GlobalNetwork] Deregistered node: %s", nodeID)
	return nil
}

func (g *GlobalNetwork) ProcessVerification(ctx context.Context, req *VerificationRequest) (*VerificationResponse, error) {
	g.stats.TotalRequests.Add(1)

	start := time.Now()

	if req.UseCache && req.CacheKey != "" {
		if cached := g.getFromCache(req.CacheKey); cached != nil {
			g.stats.CacheHits.Add(1)
			return &VerificationResponse{
				Success:    true,
				RequestID:  req.RequestID,
				FromCache:  true,
				Latency:    time.Since(start),
				Result:     cached,
			}, nil
		}
		g.stats.CacheMisses.Add(1)
	}

	node, err := g.loadBalancer.SelectNode(g)
	if err != nil {
		if failoverNode := g.failover.getFailoverNode(ctx, g); failoverNode != nil {
			return g.processOnNode(ctx, req, failoverNode, start)
		}
		return nil, fmt.Errorf("no available nodes: %w", err)
	}

	return g.processOnNode(ctx, req, node, start)
}

func (g *GlobalNetwork) processOnNode(ctx context.Context, req *VerificationRequest, node *GlobalNode, start time.Time) (*VerificationResponse, error) {
	atomic.AddInt32(&node.CurrentLoad, 1)
	defer atomic.AddInt32(&node.CurrentLoad, -1)

	latency := time.Since(start)

	g.mu.RLock()
	region := g.regions[node.Region]
	g.mu.RUnlock()

	response := &VerificationResponse{
		Success:   true,
		RequestID: req.RequestID,
		NodeID:    node.ID,
		Region:    node.Region,
		Result:    req.Data,
		FromCache: false,
		Latency:   latency,
	}

	if region != nil {
		region.LatencyAvg = (region.LatencyAvg + latency) / 2
	}

	if req.UseCache && req.CacheKey != "" {
		g.setInCache(req.CacheKey, req.Data, req.TTL)
	}

	return response, nil
}

func (g *GlobalNetwork) SyncNodeData(ctx context.Context, req *SyncRequest) (*SyncResponse, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	node, exists := g.nodes[req.NodeID]
	if !exists {
		return nil, fmt.Errorf("node %s not found", req.NodeID)
	}

	node.LastSync = time.Now()
	g.syncManager.pendingSyncs.Add(-1)
	g.syncManager.completedSyncs.Add(1)
	g.syncManager.lastSync = time.Now()

	return &SyncResponse{
		Success:     true,
		SyncedAt:    time.Now(),
		DataVersion: fmt.Sprintf("v%d", time.Now().Unix()),
	}, nil
}

func (g *GlobalNetwork) GetOptimalNode(ctx context.Context, userLocation *GeoLocation) (*GlobalNode, error) {
	return g.loadBalancer.SelectNodeForLocation(g, userLocation)
}

func (g *GlobalNetwork) ListNodes(ctx context.Context) []*GlobalNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nodes := make([]*GlobalNode, 0, len(g.nodes))
	for _, node := range g.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

func (g *GlobalNetwork) ListRegions(ctx context.Context) []*Region {
	g.mu.RLock()
	defer g.mu.RUnlock()

	regions := make([]*Region, 0, len(g.regions))
	for _, region := range g.regions {
		regions = append(regions, region)
	}
	return regions
}

func (g *GlobalNetwork) GetStats() map[string]interface{} {
	regionStats := make(map[string]int64)
	
	g.mu.RLock()
	for regionID, region := range g.regions {
		regionStats[regionID] = int64(region.ActiveNodes)
	}
	g.mu.RUnlock()

	return map[string]interface{}{
		"total_requests":         g.stats.TotalRequests.Load(),
		"cross_region_requests":  g.stats.CrossRegionRequests.Load(),
		"cache_hits":             g.stats.CacheHits.Load(),
		"cache_misses":           g.stats.CacheMisses.Load(),
		"failover_events":        g.stats.FailoverEvents.Load(),
		"recovery_events":        g.stats.RecoveryEvents.Load(),
		"avg_latency_ms":         g.stats.AvgLatency.Load(),
		"p99_latency_ms":         g.stats.P99Latency.Load(),
		"active_nodes":           g.stats.ActiveNodes.Load(),
		"healthy_regions":        g.stats.HealthyRegions.Load(),
		"region_stats":           regionStats,
		"last_update":            g.stats.LastUpdate.Load(),
	}
}

func (g *GlobalNetwork) healthMonitor() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-g.ctx.Done():
			return
		case <-ticker.C:
			g.checkNodeHealth()
		}
	}
}

func (g *GlobalNetwork) checkNodeHealth() {
	g.mu.Lock()
	defer g.mu.Unlock()

	healthyRegions := int64(0)

	for _, region := range g.regions {
		activeCount := int32(0)
		for _, node := range region.Nodes {
			if node.Healthy && int(node.CurrentLoad) < node.Capacity {
				activeCount++
			}
		}
		region.ActiveNodes = activeCount
		if activeCount > 0 {
			healthyRegions++
		}
		region.Healthy = activeCount > 0
	}

	g.stats.HealthyRegions.Store(healthyRegions)
	g.stats.LastUpdate.Store(time.Now())
}

func (g *GlobalNetwork) collectStats() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	latencies := make([]int64, 0, 1000)

	for {
		select {
		case <-g.ctx.Done():
			return
		case <-ticker.C:
			if len(latencies) > 0 {
				avg := int64(0)
				for _, l := range latencies {
					avg += l
				}
				avg /= int64(len(latencies))
				g.stats.AvgLatency.Store(avg)

				if len(latencies) > 10 {
					p99Index := int(float64(len(latencies)) * 0.99)
					g.stats.P99Latency.Store(latencies[p99Index])
				}
				latencies = latencies[:0]
			}
		}
	}
}

func (g *GlobalNetwork) updateRegion(node *GlobalNode) {
	region, exists := g.regions[node.Region]
	if !exists {
		region = &Region{
			ID:           node.Region,
			Name:         node.Region,
			Code:         node.Region,
			Continent:    node.Continent,
			Nodes:        make(map[string]*GlobalNode),
			TotalCapacity: int32(node.Capacity),
		}
		g.regions[node.Region] = region
	}

	region.Nodes[node.ID] = node
	region.ActiveNodes++
	region.TotalCapacity += int32(node.Capacity)
}

func (g *GlobalNetwork) removeFromRegion(node *GlobalNode) {
	if region, exists := g.regions[node.Region]; exists {
		delete(region.Nodes, node.ID)
		region.ActiveNodes--
		region.TotalCapacity -= int32(node.Capacity)
	}
}

func (g *GlobalNetwork) getFromCache(key string) []byte {
	return nil
}

func (g *GlobalNetwork) setInCache(key string, data []byte, ttl time.Duration) {
}

func (lb *GlobalLoadBalancer) SelectNode(g *GlobalNetwork) (*GlobalNode, error) {
	lb.mu.Lock()
	lb.roundRobin++
	lb.mu.Unlock()

	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}

	var selected *GlobalNode
	minLoad := math.MaxInt32

	for _, node := range g.nodes {
		if !node.Healthy {
			continue
		}

		load := int(atomic.LoadInt32(&node.CurrentLoad))
		if load < minLoad && load < node.Capacity {
			minLoad = load
			selected = node
		}
	}

	if selected == nil {
		return nil, fmt.Errorf("no suitable node found")
	}

	return selected, nil
}

func (lb *GlobalLoadBalancer) SelectNodeForLocation(g *GlobalNetwork, location *GeoLocation) (*GlobalNode, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}

	var sameRegionNode, anyNode *GlobalNode
	minLoad := math.MaxInt32

	for _, node := range g.nodes {
		if !node.Healthy {
			continue
		}

		load := int(atomic.LoadInt32(&node.CurrentLoad))
		if load >= node.Capacity {
			continue
		}

		if location != nil && node.Region == location.Region {
			if sameRegionNode == nil || load < minLoad {
				sameRegionNode = node
				minLoad = load
			}
		}

		if anyNode == nil || load < minLoad {
			anyNode = node
			minLoad = load
		}
	}

	if sameRegionNode != nil {
		return sameRegionNode, nil
	}

	return anyNode, nil
}

func (sm *CrossRegionSync) runSync(ctx context.Context, g *GlobalNetwork) {
	ticker := time.NewTicker(sm.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.performSync(ctx, g)
		}
	}
}

func (sm *CrossRegionSync) performSync(ctx context.Context, g *GlobalNetwork) {
	sm.mu.RLock()
	g.mu.RLock()
	nodes := make([]*GlobalNode, 0, len(g.nodes))
	for _, node := range g.nodes {
		nodes = append(nodes, node)
	}
	g.mu.RUnlock()
	sm.mu.RUnlock()

	for _, node := range nodes {
		sm.pendingSyncs.Add(1)
		go func(n *GlobalNode) {
			req := &SyncRequest{
				NodeID:    n.ID,
				Timestamp: time.Now(),
			}
			g.SyncNodeData(ctx, req)
		}(node)
	}
}

func (fm *FailoverManager) runFailoverMonitor(ctx context.Context, g *GlobalNetwork) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fm.checkFailover(ctx, g)
		}
	}
}

func (fm *FailoverManager) checkFailover(ctx context.Context, g *GlobalNetwork) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for nodeID, node := range g.nodes {
		if !node.Healthy || int(node.CurrentLoad) >= node.Capacity {
			fm.triggerFailover(g, nodeID)
		}
	}
}

func (fm *FailoverManager) triggerFailover(g *GlobalNetwork, failedNodeID string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if _, exists := fm.failoverNodes[failedNodeID]; exists {
		return
	}

	g.mu.RLock()
	var backupNode *GlobalNode
	for nodeID, node := range g.nodes {
		if nodeID != failedNodeID && node.Healthy && int(node.CurrentLoad) < node.Capacity {
			backupNode = node
			break
		}
	}
	g.mu.RUnlock()

	if backupNode != nil {
		fm.failoverNodes[failedNodeID] = &FailoverNode{
			OriginalNodeID: failedNodeID,
			BackupNodeID:   backupNode.ID,
			FailoverTime:   time.Now(),
			Status:         "active",
		}
		atomic.AddInt32(&fm.failoverCount, 1)
	}
}

func (fm *FailoverManager) getFailoverNode(ctx context.Context, g *GlobalNetwork) *GlobalNode {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	for _, fo := range fm.failoverNodes {
		if fo.Status == "active" {
			g.mu.RLock()
			node := g.nodes[fo.BackupNodeID]
			g.mu.RUnlock()
			if node != nil && node.Healthy {
				return node
			}
		}
	}
	return nil
}

func generateGlobalNodeID() string {
	return fmt.Sprintf("global_node_%d", time.Now().UnixNano())
}
