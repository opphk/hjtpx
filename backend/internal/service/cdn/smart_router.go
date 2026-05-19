package cdn

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrNoRouteAvailable   = errors.New("no available route")
	ErrUnknownLocation    = errors.New("cannot determine client location")
)

type SmartRouter struct {
	redisClient  *redis.Client
	regionCache  map[string]string
	networkStats map[string]*NetworkStats
	latencyCache map[string]map[string]float64
	mu           sync.RWMutex
}

type NetworkStats struct {
	RegionID     string  `json:"region_id"`
	Availability float64 `json:"availability"`
	LatencyAvg   float64 `json:"latency_avg_ms"`
	Throughput   float64 `json:"throughput_mbps"`
	LastUpdate   time.Time `json:"last_update"`
}

type RouteResult struct {
	Node         *EdgeNode `json:"node"`
	RegionID     string    `json:"region_id"`
	LatencyMs    float64   `json:"latency_ms"`
	LoadScore    float64   `json:"load_score"`
	RouteType    string    `json:"route_type"`
}

type GeoLocation struct {
	IP         string  `json:"ip"`
	Country    string  `json:"country"`
	Region     string  `json:"region"`
	City       string  `json:"city"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
}

func NewSmartRouter(redisClient *redis.Client) *SmartRouter {
	return &SmartRouter{
		redisClient:  redisClient,
		regionCache:  make(map[string]string),
		networkStats: make(map[string]*NetworkStats),
		latencyCache: make(map[string]map[string]float64),
	}
}

func (r *SmartRouter) Route(ctx context.Context, clientIP string) (*EdgeNode, error) {
	if clientIP == "" {
		return nil, errors.New("client IP is required")
	}

	location, err := r.GetClientLocation(clientIP)
	if err != nil {
		return nil, err
	}

	targetRegion, err := r.FindBestRegion(location)
	if err != nil {
		return nil, err
	}

	node, err := r.SelectBestNode(targetRegion)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (r *SmartRouter) GetClientLocation(clientIP string) (*GeoLocation, error) {
	if clientIP == "127.0.0.1" || clientIP == "localhost" {
		return &GeoLocation{
			IP:        clientIP,
			Country:   "Local",
			Region:    "Local",
			City:      "Local",
			Latitude:  39.9042,
			Longitude: 116.4074,
		}, nil
	}

	r.mu.RLock()
	cachedRegion, exists := r.regionCache[clientIP]
	r.mu.RUnlock()

	if exists {
		return &GeoLocation{
			IP:        clientIP,
			Country:   cachedRegion,
			Region:    cachedRegion,
			City:      cachedRegion,
			Latitude:  0,
			Longitude: 0,
		}, nil
	}

	region := r.geoIPLookup(clientIP)

	r.mu.Lock()
	r.regionCache[clientIP] = region
	r.mu.Unlock()

	return &GeoLocation{
		IP:        clientIP,
		Country:   region,
		Region:    region,
		City:      region,
		Latitude:  0,
		Longitude: 0,
	}, nil
}

func (r *SmartRouter) geoIPLookup(ip string) string {
	regionMapping := map[string]string{
		"1.":     "ap-east-1",
		"2.":     "ap-east-1",
		"3.":     "us-east-1",
		"4.":     "us-east-1",
		"10.":    "ap-east-1",
		"172.":   "ap-east-1",
		"192.":   "ap-east-1",
		"203.":   "ap-east-1",
		"202.":   "ap-east-1",
		"114.":   "ap-east-1",
		"115.":   "ap-east-1",
		"120.":   "ap-east-1",
		"180.":   "ap-east-1",
		"198.":   "us-east-1",
		"199.":   "us-west-1",
		"208.":   "us-east-1",
		"209.":   "us-west-1",
		"64.":    "us-east-1",
		"65.":    "us-west-1",
		"8.":     "us-west-1",
		"9.":     "eu-west-1",
		"31.":    "eu-west-1",
		"37.":    "eu-west-1",
		"46.":    "eu-west-1",
		"62.":    "eu-west-1",
		"77.":    "eu-west-1",
		"80.":    "eu-west-1",
		"82.":    "eu-west-1",
		"85.":    "eu-west-1",
		"87.":    "eu-west-1",
		"91.":    "eu-west-1",
		"92.":    "eu-west-1",
		"94.":    "eu-west-1",
		"141.":   "ap-northeast-1",
		"142.":   "ap-northeast-1",
		"150.":   "ap-northeast-1",
		"151.":   "ap-northeast-1",
		"210.":   "ap-northeast-1",
		"211.":   "ap-northeast-1",
		"220.":   "ap-northeast-1",
		"221.":   "ap-northeast-1",
		"222.":   "ap-northeast-1",
		"223.":   "ap-northeast-1",
		"1.1.":   "ap-south-1",
		"5.":     "eu-west-1",
		"15.":    "us-east-1",
		"18.":    "us-west-1",
		"34.":    "us-west-1",
		"35.":    "us-west-1",
		"3.0.":   "ap-south-1",
		"13.":    "ap-south-1",
		"52.":    "ap-south-1",
		"103.":   "ap-south-1",
		"104.":   "ap-south-1",
		"163.":   "ap-south-1",
		"182.":   "ap-south-1",
		"183.":   "ap-south-1",
		"49.":    "ap-east-1",
		"119.":   "ap-east-1",
		"121.":   "ap-east-1",
		"122.":   "ap-east-1",
		"123.":   "ap-east-1",
		"140.":   "ap-east-1",
	}

	for prefix, region := range regionMapping {
		if len(ip) >= len(prefix) && ip[:len(prefix)] == prefix {
			return region
		}
	}

	return "ap-east-1"
}

func (r *SmartRouter) FindBestRegion(location *GeoLocation) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	regionScores := make(map[string]float64)

	for regionID, stats := range r.networkStats {
		distanceScore := r.calculateDistanceScore(regionID, location)
		availabilityScore := stats.Availability
		latencyScore := 1 - (stats.LatencyAvg / 1000)

		regionScores[regionID] = (distanceScore * 0.4) + (availabilityScore * 0.3) + (latencyScore * 0.3)
	}

	if len(regionScores) == 0 {
		return location.Region, nil
	}

	bestRegion := ""
	bestScore := 0.0

	for regionID, score := range regionScores {
		if score > bestScore {
			bestScore = score
			bestRegion = regionID
		}
	}

	if bestRegion == "" {
		return location.Region, nil
	}

	return bestRegion, nil
}

func (r *SmartRouter) calculateDistanceScore(regionID string, location *GeoLocation) float64 {
	regionCoords := map[string][2]float64{
		"ap-east-1":      {22.3193, 114.1694},
		"ap-south-1":     {1.3521, 103.8198},
		"ap-northeast-1": {35.6762, 139.6503},
		"ap-southeast-1": {-33.8688, 151.2093},
		"us-east-1":      {40.7128, -74.0060},
		"us-west-1":      {37.7749, -122.4194},
		"eu-west-1":      {52.5200, 13.4050},
		"eu-central-1":   {48.2082, 16.3738},
		"af-south-1":     {-33.9249, 18.4241},
		"sa-east-1":      {-23.5505, -46.6333},
	}

	coords, exists := regionCoords[regionID]
	if !exists {
		return 0.5
	}

	if location.Latitude == 0 && location.Longitude == 0 {
		return 0.7
	}

	distance := r.haversineDistance(coords[0], coords[1], location.Latitude, location.Longitude)

	maxDistance := 20000.0
	score := 1 - (distance / maxDistance)

	return math.Max(0, score)
}

func (r *SmartRouter) haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0

	lat1Rad := math.Pi * lat1 / 180
	lon1Rad := math.Pi * lon1 / 180
	lat2Rad := math.Pi * lat2 / 180
	lon2Rad := math.Pi * lon2 / 180

	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func (r *SmartRouter) SelectBestNode(regionID string) (*EdgeNode, error) {
	return nil, ErrNoRouteAvailable
}

func (r *SmartRouter) UpdateNetworkStats(regionID string, stats *NetworkStats) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.networkStats[regionID] = stats
}

func (r *SmartRouter) GetNetworkStats(regionID string) (*NetworkStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats, exists := r.networkStats[regionID]
	if !exists {
		return nil, errors.New("network stats not found")
	}
	return stats, nil
}

func (r *SmartRouter) RecordLatency(fromIP, toRegion string, latencyMs float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.latencyCache[fromIP]; !exists {
		r.latencyCache[fromIP] = make(map[string]float64)
	}
	r.latencyCache[fromIP][toRegion] = latencyMs
}

func (r *SmartRouter) GetLatency(fromIP, toRegion string) (float64, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if regionMap, exists := r.latencyCache[fromIP]; exists {
		if latency, exists := regionMap[toRegion]; exists {
			return latency, true
		}
	}
	return 0, false
}

func (r *SmartRouter) ClearCache() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.regionCache = make(map[string]string)
	r.latencyCache = make(map[string]map[string]float64)
}

func (r *SmartRouter) GetRoutingDecision(clientIP string) (*RouteResult, error) {
	ctx := context.Background()
	node, err := r.Route(ctx, clientIP)
	if err != nil {
		return nil, err
	}

	if node == nil {
		return &RouteResult{
			RegionID: r.geoIPLookup(clientIP),
			RouteType: "default",
		}, nil
	}

	return &RouteResult{
		Node:         node,
		RegionID:     node.RegionID,
		LatencyMs:    node.LatencyMs,
		LoadScore:    float64(node.CurrentLoad) / float64(node.Capacity),
		RouteType:    "geographic",
	}, nil
}

func (r *SmartRouter) String() string {
	return fmt.Sprintf("SmartRouter{cachedRegions=%d, networkStats=%d}",
		len(r.regionCache), len(r.networkStats))
}