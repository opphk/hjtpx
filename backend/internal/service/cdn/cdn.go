package cdn

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/redis/go-redis/v9"
)

var (
	ErrRegionNotFound      = errors.New("region not found")
	ErrNodeNotFound        = errors.New("node not found")
	ErrInvalidRegion       = errors.New("invalid region")
	ErrNodeAlreadyExists   = errors.New("node already exists")
	ErrNoHealthyNode       = errors.New("no healthy node available")
)

type CDNService struct {
	regions        map[string]*Region
	nodes          map[string]*EdgeNode
	smartRouter    *SmartRouter
	staticAccelerator *StaticAssetAccelerator
	mu             sync.RWMutex
	redisClient    *redis.Client
	ctx            context.Context
}

type Region struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Code        string            `json:"code"`
	Continent   string            `json:"continent"`
	Latitude    float64           `json:"latitude"`
	Longitude   float64           `json:"longitude"`
	Nodes       []string          `json:"node_ids"`
	Enabled     bool              `json:"enabled"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type RegionStats struct {
	RegionID     string            `json:"region_id"`
	RegionName   string            `json:"region_name"`
	NodeCount    int               `json:"node_count"`
	HealthyNodes int               `json:"healthy_nodes"`
	TrafficBytes int64             `json:"traffic_bytes"`
	RequestCount int64             `json:"request_count"`
	LatencyMs    float64           `json:"latency_ms"`
}

func NewCDNService(redisClient *redis.Client) *CDNService {
	ctx := context.Background()
	return &CDNService{
		regions:        make(map[string]*Region),
		nodes:          make(map[string]*EdgeNode),
		smartRouter:    NewSmartRouter(redisClient),
		staticAccelerator: NewStaticAssetAccelerator(redisClient),
		redisClient:    redisClient,
		ctx:            ctx,
	}
}

func (s *CDNService) InitializeDefaultRegions() error {
	defaultRegions := []*Region{
		{ID: "ap-east-1", Name: "亚太东部", Code: "HK", Continent: "Asia", Latitude: 22.3193, Longitude: 114.1694, Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "ap-south-1", Name: "亚太南部", Code: "SG", Continent: "Asia", Latitude: 1.3521, Longitude: 103.8198, Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "ap-northeast-1", Name: "亚太东北部", Code: "JP", Continent: "Asia", Latitude: 35.6762, Longitude: 139.6503, Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "ap-southeast-1", Name: "亚太东南部", Code: "AU", Continent: "Oceania", Latitude: -33.8688, Longitude: 151.2093, Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "us-east-1", Name: "美国东部", Code: "US-E", Continent: "North America", Latitude: 40.7128, Longitude: -74.0060, Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "us-west-1", Name: "美国西部", Code: "US-W", Continent: "North America", Latitude: 37.7749, Longitude: -122.4194, Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "eu-west-1", Name: "欧洲西部", Code: "EU-W", Continent: "Europe", Latitude: 52.5200, Longitude: 13.4050, Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "eu-central-1", Name: "欧洲中部", Code: "EU-C", Continent: "Europe", Latitude: 48.2082, Longitude: 16.3738, Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "af-south-1", Name: "非洲南部", Code: "ZA", Continent: "Africa", Latitude: -33.9249, Longitude: 18.4241, Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "sa-east-1", Name: "南美东部", Code: "BR", Continent: "South America", Latitude: -23.5505, Longitude: -46.6333, Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, region := range defaultRegions {
		if _, exists := s.regions[region.ID]; !exists {
			s.regions[region.ID] = region
		}
	}

	return s.persistRegions()
}

func (s *CDNService) GetRegion(regionID string) (*Region, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	region, exists := s.regions[regionID]
	if !exists {
		return nil, ErrRegionNotFound
	}
	return region, nil
}

func (s *CDNService) ListRegions() []*Region {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Region, 0, len(s.regions))
	for _, region := range s.regions {
		result = append(result, region)
	}
	return result
}

func (s *CDNService) AddRegion(region *Region) error {
	if region.ID == "" || region.Code == "" {
		return ErrInvalidRegion
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.regions[region.ID]; exists {
		return ErrNodeAlreadyExists
	}

	region.CreatedAt = time.Now()
	region.UpdatedAt = time.Now()
	s.regions[region.ID] = region

	return s.persistRegions()
}

func (s *CDNService) UpdateRegion(regionID string, updates map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	region, exists := s.regions[regionID]
	if !exists {
		return ErrRegionNotFound
	}

	if name, ok := updates["name"].(string); ok {
		region.Name = name
	}
	if enabled, ok := updates["enabled"].(bool); ok {
		region.Enabled = enabled
	}

	region.UpdatedAt = time.Now()
	return s.persistRegions()
}

func (s *CDNService) DeleteRegion(regionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.regions[regionID]; !exists {
		return ErrRegionNotFound
	}

	delete(s.regions, regionID)
	return s.persistRegions()
}

func (s *CDNService) GetRegionStats(regionID string) (*RegionStats, error) {
	region, err := s.GetRegion(regionID)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	healthyCount := 0
	var totalLatency float64
	var totalTraffic, totalRequests int64

	for _, nodeID := range region.Nodes {
		if node, exists := s.nodes[nodeID]; exists {
			if node.IsHealthy {
				healthyCount++
				totalLatency += node.LatencyMs
				totalTraffic += node.TrafficBytes
				totalRequests += node.RequestCount
			}
		}
	}

	avgLatency := 0.0
	if healthyCount > 0 {
		avgLatency = totalLatency / float64(healthyCount)
	}

	return &RegionStats{
		RegionID:     region.ID,
		RegionName:   region.Name,
		NodeCount:    len(region.Nodes),
		HealthyNodes: healthyCount,
		TrafficBytes: totalTraffic,
		RequestCount: totalRequests,
		LatencyMs:    avgLatency,
	}, nil
}

func (s *CDNService) GetGlobalStats() ([]*RegionStats, error) {
	var stats []*RegionStats
	for _, region := range s.regions {
		regionStats, err := s.GetRegionStats(region.ID)
		if err != nil {
			return nil, err
		}
		stats = append(stats, regionStats)
	}
	return stats, nil
}

func (s *CDNService) persistRegions() error {
	if s.redisClient == nil {
		return nil
	}
	return nil
}

func (s *CDNService) RouteRequest(ctx context.Context, clientIP string) (*EdgeNode, error) {
	return s.smartRouter.Route(ctx, clientIP)
}

func (s *CDNService) AccelerateStaticAsset(ctx context.Context, assetPath string, clientIP string) (*AssetResponse, error) {
	return s.staticAccelerator.ServeAsset(ctx, assetPath, clientIP)
}

func (s *CDNService) UpdateNodeHealth(nodeID string, isHealthy bool, latencyMs float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	node, exists := s.nodes[nodeID]
	if !exists {
		return ErrNodeNotFound
	}

	node.IsHealthy = isHealthy
	node.LatencyMs = latencyMs
	node.LastHealthCheck = time.Now()

	return nil
}

func (s *CDNService) RegisterNode(node *EdgeNode) error {
	if node.ID == "" || node.RegionID == "" {
		return errors.New("node ID and region ID are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[node.ID]; exists {
		return ErrNodeAlreadyExists
	}

	node.CreatedAt = time.Now()
	node.LastHealthCheck = time.Now()
	s.nodes[node.ID] = node

	region, exists := s.regions[node.RegionID]
	if exists {
		region.Nodes = append(region.Nodes, node.ID)
		region.UpdatedAt = time.Now()
	}

	return s.persistRegions()
}

func (s *CDNService) GetNode(nodeID string) (*EdgeNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, exists := s.nodes[nodeID]
	if !exists {
		return nil, ErrNodeNotFound
	}
	return node, nil
}

func (s *CDNService) ListNodes(regionID string) []*EdgeNode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := []*EdgeNode{}
	for _, node := range s.nodes {
		if regionID == "" || node.RegionID == regionID {
			result = append(result, node)
		}
	}
	return result
}

func (s *CDNService) RemoveNode(nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	node, exists := s.nodes[nodeID]
	if !exists {
		return ErrNodeNotFound
	}

	delete(s.nodes, nodeID)

	if region, exists := s.regions[node.RegionID]; exists {
		for i, nid := range region.Nodes {
			if nid == nodeID {
				region.Nodes = append(region.Nodes[:i], region.Nodes[i+1:]...)
				break
			}
		}
		region.UpdatedAt = time.Now()
	}

	return s.persistRegions()
}

func (s *CDNService) GetHealthyNodes(regionID string) []*EdgeNode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := []*EdgeNode{}
	for _, node := range s.nodes {
		if node.IsHealthy && (regionID == "" || node.RegionID == regionID) {
			result = append(result, node)
		}
	}
	return result
}

func (s *CDNService) ExecuteEdgeFunction(ctx context.Context, functionName string, params map[string]interface{}) (*model.EdgeExecutionResult, error) {
	nodes := s.GetHealthyNodes("")
	if len(nodes) == 0 {
		return nil, ErrNoHealthyNode
	}

	node := nodes[0]
	return node.ExecuteFunction(ctx, functionName, params)
}

func (s *CDNService) GetCacheStats() *CacheStats {
	return s.staticAccelerator.GetStats()
}

func (s *CDNService) ClearCache() {
	s.staticAccelerator.Clear()
}

func (s *CDNService) PurgeCache(assetPath string) error {
	return s.staticAccelerator.Purge(assetPath)
}

func (s *CDNService) WarmupCache(paths []string) error {
	return s.staticAccelerator.Warmup(paths)
}

func (s *CDNService) GetClientLocation(clientIP string) *GeoLocation {
	loc, _ := s.smartRouter.GetClientLocation(clientIP)
	return loc
}

func (s *CDNService) GetRoutingDecision(clientIP string) (*RouteResult, error) {
	return s.smartRouter.GetRoutingDecision(clientIP)
}

func (s *CDNService) String() string {
	return fmt.Sprintf("CDNService{regions=%d, nodes=%d}", len(s.regions), len(s.nodes))
}