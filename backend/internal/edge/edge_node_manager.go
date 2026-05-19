package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

type Region string

const (
	RegionAP    Region = "ap-east-1"
	RegionEU    Region = "eu-west-1"
	RegionUS    Region = "us-east-1"
	RegionCN    Region = "cn-north-1"
	RegionSA    Region = "sa-east-1"
	RegionME    Region = "me-central-1"
	RegionOC    Region = "ap-southeast-1"
	RegionJP    Region = "ap-northeast-1"
	RegionIN    Region = "ap-south-1"
	RegionAF    Region = "af-south-1"
)

type NodeStatus string

const (
	NodeStatusActive   NodeStatus = "active"
	NodeStatusInactive NodeStatus = "inactive"
	NodeStatusUnhealthy NodeStatus = "unhealthy"
	NodeStatusDraining  NodeStatus = "draining"
)

type EdgeNodeType string

const (
	NodeTypeCDN       EdgeNodeType = "cdn"
	NodeTypeInference EdgeNodeType = "inference"
	NodeTypeMixed     EdgeNodeType = "mixed"
)

type EdgeNode struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Region       Region         `json:"region"`
	Zone         string         `json:"zone"`
	IPAddress    string         `json:"ip_address"`
	PublicIP     string         `json:"public_ip"`
	Port         int            `json:"port"`
	Type         EdgeNodeType   `json:"type"`
	Status       NodeStatus     `json:"status"`
	Weight       int            `json:"weight"`
	Capacity     int            `json:"capacity"`
	CurrentLoad  int32          `json:"current_load"`
	LatencyMs    float64        `json:"latency_ms"`
	Latitude     float64        `json:"latitude"`
	Longitude    float64        `json:"longitude"`
	Country      string         `json:"country"`
	City         string         `json:"city"`
	IsCSP        bool           `json:"is_csp"`
	CSPProvider  string         `json:"csp_provider,omitempty"`
	LastHeartbeat time.Time     `json:"last_heartbeat"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	Tags         map[string]string `json:"tags"`
	Metrics      *NodeMetrics   `json:"metrics"`
}

type NodeMetrics struct {
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	DiskUsage      float64 `json:"disk_usage"`
	NetworkInBps   float64 `json:"network_in_bps"`
	NetworkOutBps  float64 `json:"network_out_bps"`
	RequestCount   int64   `json:"request_count"`
	ErrorCount     int64   `json:"error_count"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	P99LatencyMs   float64 `json:"p99_latency_ms"`
	ActiveConns    int32   `json:"active_connections"`
	CacheHitRate   float64 `json:"cache_hit_rate"`
	ModelLoadTime  float64 `json:"model_load_time"`
	InferenceTime  float64 `json:"inference_time"`
}

type EdgeNodeManager struct {
	nodes          map[string]*EdgeNode
	regionIndex     map[Region][]string
	typeIndex       map[EdgeNodeType][]string
	mu              sync.RWMutex
	redisClient     *redis.Client
	heartbeatTTL    time.Duration
	checkInterval   time.Duration
	metricsWindow   time.Duration
	metricsData     map[string]*NodeMetrics
	metricsMu       sync.RWMutex
	version         int64
}

type NodeSelector func(*EdgeNode) bool

type NodeSelectorOptions struct {
	Region      Region
	NodeType    EdgeNodeType
	MinCapacity int
	MaxLatency  time.Duration
	Status      NodeStatus
	Tags        map[string]string
}

func NewEdgeNodeManager(redisClient *redis.Client) *EdgeNodeManager {
	manager := &EdgeNodeManager{
		nodes:         make(map[string]*EdgeNode),
		regionIndex:   make(map[Region][]string),
		typeIndex:     make(map[EdgeNodeType][]string),
		redisClient:   redisClient,
		heartbeatTTL:   30 * time.Second,
		checkInterval:  10 * time.Second,
		metricsWindow:  5 * time.Minute,
		metricsData:    make(map[string]*NodeMetrics),
	}
	go manager.startHeartbeatChecker()
	return manager
}

func (m *EdgeNodeManager) RegisterNode(ctx context.Context, node *EdgeNode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if node.ID == "" {
		return fmt.Errorf("node ID is required")
	}
	if node.IPAddress == "" && node.PublicIP == "" {
		return fmt.Errorf("at least one IP address is required")
	}
	if node.Port <= 0 || node.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", node.Port)
	}

	node.CreatedAt = time.Now()
	node.UpdatedAt = time.Now()
	node.LastHeartbeat = time.Now()
	node.Status = NodeStatusActive

	if node.Metrics == nil {
		node.Metrics = &NodeMetrics{}
	}
	if node.Tags == nil {
		node.Tags = make(map[string]string)
	}

	m.nodes[node.ID] = node
	m.regionIndex[node.Region] = append(m.regionIndex[node.Region], node.ID)
	m.typeIndex[node.Type] = append(m.typeIndex[node.Type], node.ID)
	m.metricsData[node.ID] = &NodeMetrics{}

	if m.redisClient != nil {
		data, _ := json.Marshal(node)
		key := fmt.Sprintf("edge:node:%s", node.ID)
		m.redisClient.Set(ctx, key, data, 24*time.Hour)
	}

	atomic.AddInt64(&m.version, 1)
	return nil
}

func (m *EdgeNodeManager) UnregisterNode(ctx context.Context, nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	node.Status = NodeStatusDraining
	node.UpdatedAt = time.Now()

	m.regionIndex[node.Region] = removeFromSlice(m.regionIndex[node.Region], nodeID)
	m.typeIndex[node.Type] = removeFromSlice(m.typeIndex[node.Type], nodeID)

	go func() {
		time.Sleep(5 * time.Second)
		m.mu.Lock()
		defer m.mu.Unlock()
		delete(m.nodes, nodeID)
		delete(m.metricsData, nodeID)
		if m.redisClient != nil {
			m.redisClient.Del(ctx, fmt.Sprintf("edge:node:%s", nodeID))
		}
	}()

	atomic.AddInt64(&m.version, 1)
	return nil
}

func (m *EdgeNodeManager) UpdateNodeStatus(ctx context.Context, nodeID string, status NodeStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	node.Status = status
	node.UpdatedAt = time.Now()
	atomic.AddInt64(&m.version, 1)

	if m.redisClient != nil {
		data, _ := json.Marshal(node)
		m.redisClient.Set(ctx, fmt.Sprintf("edge:node:%s", nodeID), data, 24*time.Hour)
	}

	return nil
}

func (m *EdgeNodeManager) UpdateNodeMetrics(ctx context.Context, nodeID string, metrics *NodeMetrics) error {
	m.metricsMu.Lock()
	defer m.metricsMu.Unlock()

	m.mu.RLock()
	node, exists := m.nodes[nodeID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	node.Metrics = metrics
	node.UpdatedAt = time.Now()
	m.metricsData[nodeID] = metrics

	if m.redisClient != nil {
		key := fmt.Sprintf("edge:metrics:%s", nodeID)
		data, _ := json.Marshal(metrics)
		m.redisClient.Set(ctx, key, data, m.metricsWindow)
	}

	return nil
}

func (m *EdgeNodeManager) GetNode(nodeID string) (*EdgeNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	nodeCopy := *node
	nodeCopy.Metrics = &NodeMetrics{}
	if node.Metrics != nil {
		nodeCopy.Metrics = node.Metrics
	}
	return &nodeCopy, nil
}

func (m *EdgeNodeManager) ListNodes(opts *NodeSelectorOptions) ([]*EdgeNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*EdgeNode

	for _, node := range m.nodes {
		if opts != nil {
			if opts.Region != "" && node.Region != opts.Region {
				continue
			}
			if opts.NodeType != "" && node.Type != opts.NodeType {
				continue
			}
			if opts.Status != "" && node.Status != opts.Status {
				continue
			}
			if opts.MinCapacity > 0 && node.Capacity < opts.MinCapacity {
				continue
			}
			if opts.MaxLatency > 0 && time.Duration(node.LatencyMs*float64(time.Millisecond)) > opts.MaxLatency {
				continue
			}
			if opts.Tags != nil {
				match := true
				for k, v := range opts.Tags {
					if node.Tags[k] != v {
						match = false
						break
					}
				}
				if !match {
					continue
				}
			}
		}

		nodeCopy := *node
		nodeCopy.Metrics = &NodeMetrics{}
		if node.Metrics != nil {
			nodeCopy.Metrics = node.Metrics
		}
		result = append(result, &nodeCopy)
	}

	return result, nil
}

func (m *EdgeNodeManager) SelectNode(opts *NodeSelectorOptions) (*EdgeNode, error) {
	nodes, err := m.ListNodes(opts)
	if err != nil {
		return nil, err
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no suitable node found")
	}

	var bestNode *EdgeNode
	bestScore := -1.0

	for _, node := range nodes {
		if node.Status != NodeStatusActive {
			continue
		}

		currentLoad := atomic.LoadInt32(&node.CurrentLoad)
		capacityScore := float64(node.Capacity-int(currentLoad)) / float64(node.Capacity)
		latencyScore := 1.0 - (node.LatencyMs / 1000.0)
		weightScore := float64(node.Weight) / 100.0

		score := capacityScore*0.4 + latencyScore*0.4 + weightScore*0.2

		if node.Metrics != nil {
			cpuScore := 1.0 - node.Metrics.CPUUsage
			memScore := 1.0 - node.Metrics.MemoryUsage
			score = score*0.7 + (cpuScore+memScore)/2*0.3
		}

		if score > bestScore {
			bestScore = score
			bestNode = node
		}
	}

	if bestNode == nil {
		return nil, fmt.Errorf("no healthy node available")
	}

	return bestNode, nil
}

func (m *EdgeNodeManager) GetNodesByRegion(region Region) ([]*EdgeNode, error) {
	return m.ListNodes(&NodeSelectorOptions{Region: region, Status: NodeStatusActive})
}

func (m *EdgeNodeManager) GetNodesByType(nodeType EdgeNodeType) ([]*EdgeNode, error) {
	return m.ListNodes(&NodeSelectorOptions{NodeType: nodeType, Status: NodeStatusActive})
}

func (m *EdgeNodeManager) UpdateNodeLoad(nodeID string, loadChange int32) {
	m.mu.RLock()
	node, exists := m.nodes[nodeID]
	m.mu.RUnlock()

	if exists {
		atomic.AddInt32(&node.CurrentLoad, loadChange)
		if node.CurrentLoad < 0 {
			atomic.StoreInt32(&node.CurrentLoad, 0)
		}
	}
}

func (m *EdgeNodeManager) UpdateNodeLatency(nodeID string, latencyMs float64) {
	m.mu.RLock()
	node, exists := m.nodes[nodeID]
	m.mu.RUnlock()

	if exists {
		node.LatencyMs = latencyMs
	}
}

func (m *EdgeNodeManager) RecordHeartbeat(nodeID string) error {
	m.mu.RLock()
	node, exists := m.nodes[nodeID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	node.LastHeartbeat = time.Now()
	if node.Status == NodeStatusUnhealthy {
		node.Status = NodeStatusActive
	}

	return nil
}

func (m *EdgeNodeManager) startHeartbeatChecker() {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		m.checkHeartbeats()
	}
}

func (m *EdgeNodeManager) checkHeartbeats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	threshold := m.heartbeatTTL

	for _, node := range m.nodes {
		if node.Status == NodeStatusDraining {
			continue
		}

		elapsed := now.Sub(node.LastHeartbeat)
		if elapsed > threshold {
			if node.Status == NodeStatusActive {
				node.Status = NodeStatusUnhealthy
				atomic.AddInt64(&m.version, 1)
			}
		}

		if elapsed > threshold*3 {
			node.Status = NodeStatusInactive
			atomic.AddInt64(&m.version, 1)
		}
	}
}

func (m *EdgeNodeManager) GetClusterStats() *ClusterStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &ClusterStats{
		TotalNodes:     len(m.nodes),
		RegionStats:    make(map[Region]*RegionStat),
		TypeStats:      make(map[EdgeNodeType]*TypeStat),
		StatusCounts:   make(map[NodeStatus]int),
	}

	var totalCapacity, totalLoad int32
	var totalLatency float64
	var activeCount int

	for _, node := range m.nodes {
		stats.StatusCounts[node.Status]++

		regionStat, ok := stats.RegionStats[node.Region]
		if !ok {
			regionStat = &RegionStat{Region: node.Region}
			stats.RegionStats[node.Region] = regionStat
		}
		regionStat.Count++
		regionStat.TotalCapacity += int32(node.Capacity)
		atomic.AddInt32(&regionStat.TotalLoad, atomic.LoadInt32(&node.CurrentLoad))

		typeStat, ok := stats.TypeStats[node.Type]
		if !ok {
			typeStat = &TypeStat{Type: node.Type}
			stats.TypeStats[node.Type] = typeStat
		}
		typeStat.Count++

		if node.Status == NodeStatusActive {
			activeCount++
			totalCapacity += int32(node.Capacity)
			totalLoad += atomic.LoadInt32(&node.CurrentLoad)
			totalLatency += node.LatencyMs
		}
	}

	if activeCount > 0 {
		stats.AverageLatencyMs = totalLatency / float64(activeCount)
	}
	if totalCapacity > 0 {
		stats.LoadPercentage = float64(totalLoad) / float64(totalCapacity) * 100
	}
	stats.ActiveNodes = activeCount

	return stats
}

type ClusterStats struct {
	TotalNodes      int                    `json:"total_nodes"`
	ActiveNodes     int                    `json:"active_nodes"`
	AverageLatencyMs float64               `json:"average_latency_ms"`
	LoadPercentage  float64                `json:"load_percentage"`
	RegionStats     map[Region]*RegionStat `json:"region_stats"`
	TypeStats       map[EdgeNodeType]*TypeStat `json:"type_stats"`
	StatusCounts    map[NodeStatus]int     `json:"status_counts"`
}

type RegionStat struct {
	Region       Region `json:"region"`
	Count        int    `json:"count"`
	TotalCapacity int32 `json:"total_capacity"`
	TotalLoad    int32  `json:"total_load"`
}

type TypeStat struct {
	Type  EdgeNodeType `json:"type"`
	Count int           `json:"count"`
}

func (m *EdgeNodeManager) SyncFromRedis(ctx context.Context) error {
	if m.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	pattern := "edge:node:*"
	iter := m.redisClient.Scan(ctx, 0, pattern, 100).Iterator()

	m.mu.Lock()
	defer m.mu.Unlock()

	for iter.Next(ctx) {
		key := iter.Val()
		data, err := m.redisClient.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var node EdgeNode
		if err := json.Unmarshal(data, &node); err != nil {
			continue
		}

		m.nodes[node.ID] = &node
	}

	return iter.Err()
}

func (m *EdgeNodeManager) GetVersion() int64 {
	return atomic.LoadInt64(&m.version)
}

func (m *EdgeNodeManager) CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

func (m *EdgeNodeManager) FindNearestNode(lat, lon float64) (*EdgeNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var nearest *EdgeNode
	minDistance := math.MaxFloat64

	for _, node := range m.nodes {
		if node.Status != NodeStatusActive {
			continue
		}

		distance := m.CalculateDistance(lat, lon, node.Latitude, node.Longitude)
		if distance < minDistance {
			minDistance = distance
			nearest = node
		}
	}

	if nearest == nil {
		return nil, fmt.Errorf("no active node found")
	}

	return nearest, nil
}

func (m *EdgeNodeManager) ResolveIPRegion(ip string) (*GeoLocation, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	isPublic := !parsedIP.IsLoopback() && !parsedIP.IsUnspecified() && !parsedIP.IsLinkLocalUnicast()
	location := &GeoLocation{
		IP:       ip,
		IsPublic: isPublic,
	}

	if !location.IsPublic {
		return location, nil
	}

	location.Latitude = getLatFromIP(parsedIP)
	location.Longitude = getLonFromIP(parsedIP)
	location.Region = getRegionFromIP(parsedIP)
	location.Country = getCountryFromIP(parsedIP)
	location.City = getCityFromIP(parsedIP)

	return location, nil
}

type GeoLocation struct {
	IP        string  `json:"ip"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Region    Region  `json:"region"`
	Country   string  `json:"country"`
	City      string  `json:"city"`
	IsPublic  bool    `json:"is_public"`
}

func getLatFromIP(ip net.IP) float64 {
	h := hashIP(ip)
	regions := []struct{ lat float64 }{
		{39.9},
		{51.5},
		{40.7},
		{31.2},
		{-23.5},
		{25.2},
		{1.3},
		{35.6},
		{19.0},
		{-33.8},
	}
	return regions[int(h)%len(regions)].lat
}

func getLonFromIP(ip net.IP) float64 {
	h := hashIP(ip)
	regions := []struct{ lon float64 }{
		{116.4},
		{-0.1},
		{-74.0},
		{121.4},
		{-46.6},
		{55.3},
		{103.8},
		{139.6},
		{72.5},
		{151.2},
	}
	return regions[int(h)%len(regions)].lon
}

func getRegionFromIP(ip net.IP) Region {
	h := hashIP(ip)
	regions := []Region{RegionCN, RegionUS, RegionEU, RegionAP, RegionJP, RegionIN, RegionSA, RegionOC, RegionME, RegionAF}
	return regions[int(h)%len(regions)]
}

func getCountryFromIP(ip net.IP) string {
	h := hashIP(ip)
	countries := []string{"CN", "US", "DE", "JP", "IN", "BR", "AU", "SG", "AE", "ZA"}
	return countries[int(h)%len(countries)]
}

func getCityFromIP(ip net.IP) string {
	h := hashIP(ip)
	cities := []string{"Beijing", "New York", "Berlin", "Tokyo", "Mumbai", "São Paulo", "Sydney", "Singapore", "Dubai", "Johannesburg"}
	return cities[int(h)%len(cities)]
}

func hashIP(ip net.IP) int64 {
	var h int64
	for _, b := range ip {
		h = h*31 + int64(b)
	}
	return h
}

func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
