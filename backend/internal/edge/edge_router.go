package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

type RoutingStrategy string

const (
	RoutingLatency       RoutingStrategy = "latency"
	RoutingGeo           RoutingStrategy = "geographic"
	RoutingLoad          RoutingStrategy = "load_balanced"
	RoutingFailover      RoutingStrategy = "failover"
	RoutingPerformance   RoutingStrategy = "performance"
	RoutingCostEffective RoutingStrategy = "cost_effective"
)

type RouteRule struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Priority      int            `json:"priority"`
	Strategy      RoutingStrategy `json:"strategy"`
	SourceRegions []Region       `json:"source_regions"`
	SourceCountries []string     `json:"source_countries"`
	TargetRegions []Region       `json:"target_regions"`
	Conditions    []RouteCondition `json:"conditions"`
	Actions       []RouteAction   `json:"actions"`
	Weight        int            `json:"weight"`
	Enabled       bool           `json:"enabled"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type RouteCondition struct {
	Type      string      `json:"type"`
	Field     string      `json:"field"`
	Operator  string      `json:"operator"`
	Value     interface{} `json:"value"`
}

type RouteAction struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type Route struct {
	RuleID      string    `json:"rule_id"`
	NodeID      string    `json:"node_id"`
	IPAddress   string    `json:"ip_address"`
	Port        int       `json:"port"`
	LatencyMs   float64   `json:"latency_ms"`
	DistanceKm  float64   `json:"distance_km"`
	LoadPercent float64   `json:"load_percent"`
	Score       float64   `json:"score"`
	Reason      string    `json:"reason"`
	Timestamp   time.Time `json:"timestamp"`
}

type RoutingTable struct {
	Rules    map[string]*RouteRule
	Index    map[int][]string
	mu       sync.RWMutex
	version  int64
}

type GeoRouter struct {
	table         *RoutingTable
	nodeManager   *EdgeNodeManager
	redisClient   *redis.Client
	resolver      *DNSResolver
	metrics       *RouterMetrics
	mu            sync.RWMutex
	updateChan    chan *RouteUpdate
	metricsWindow time.Duration
}

type RouterMetrics struct {
	TotalRequests     int64           `json:"total_requests"`
	CacheHits        int64           `json:"cache_hits"`
	CacheMisses      int64           `json:"cache_misses"`
	StrategyCounts   map[RoutingStrategy]int64 `json:"strategy_counts"`
	RegionCounts     map[Region]int64 `json:"region_counts"`
	LatencyHistogram []int64         `json:"latency_histogram"`
	mu               sync.RWMutex
}

type RouteUpdate struct {
	Type      string      `json:"type"`
	RuleID    string      `json:"rule_id,omitempty"`
	NodeID    string      `json:"node_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

type RouteRequest struct {
	SourceIP       string            `json:"source_ip"`
	SourceRegion   Region            `json:"source_region,omitempty"`
	SourceCountry  string            `json:"source_country,omitempty"`
	TargetType     EdgeNodeType      `json:"target_type"`
	Strategy       RoutingStrategy   `json:"strategy"`
	Conditions     map[string]interface{} `json:"conditions"`
	GeoLocation    *GeoLocation      `json:"geo_location,omitempty"`
	IncludeBackup  bool              `json:"include_backup"`
	MaxResults     int               `json:"max_results"`
	Timeout        time.Duration     `json:"timeout"`
}

type RouteResponse struct {
	PrimaryRoute   *Route             `json:"primary_route"`
	BackupRoutes   []*Route           `json:"backup_routes,omitempty"`
	MatchedRule    *RouteRule         `json:"matched_rule"`
	TotalCandidates int               `json:"total_candidates"`
	SelectionTime  time.Duration      `json:"selection_time"`
	Timestamp      time.Time          `json:"timestamp"`
}

type RouteCache struct {
	entries map[string]*RouteCacheEntry
	mu      sync.RWMutex
	maxSize int
	ttl     time.Duration
}

type RouteCacheEntry struct {
	Routes   []*Route
	RuleID   string
	ExpiresAt time.Time
}

func NewGeoRouter(nodeManager *EdgeNodeManager, redisClient *redis.Client) *GeoRouter {
	return &GeoRouter{
		table: &RoutingTable{
			Rules: make(map[string]*RouteRule),
			Index: make(map[int][]string),
		},
		nodeManager:   nodeManager,
		redisClient:   redisClient,
		metrics:       &RouterMetrics{
			StrategyCounts: make(map[RoutingStrategy]int64),
			RegionCounts:   make(map[Region]int64),
		},
		updateChan:    make(chan *RouteUpdate, 100),
		metricsWindow: 5 * time.Minute,
	}
}

func (r *GeoRouter) AddRule(rule *RouteRule) error {
	r.table.mu.Lock()
	defer r.table.mu.Unlock()

	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule-%d", time.Now().UnixNano())
	}

	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	rule.Enabled = true

	r.table.Rules[rule.ID] = rule
	r.table.Index[rule.Priority] = append(r.table.Index[rule.Priority], rule.ID)
	atomic.AddInt64(&r.table.version, 1)

	r.emitUpdate(&RouteUpdate{
		Type:      "rule_added",
		RuleID:    rule.ID,
		Timestamp: time.Now(),
	})

	return nil
}

func (r *GeoRouter) DeleteRule(ruleID string) error {
	r.table.mu.Lock()
	defer r.table.mu.Unlock()

	rule, exists := r.table.Rules[ruleID]
	if !exists {
		return fmt.Errorf("rule not found: %s", ruleID)
	}

	delete(r.table.Rules, ruleID)

	ruleIDs := r.table.Index[rule.Priority]
	for i, id := range ruleIDs {
		if id == ruleID {
			r.table.Index[rule.Priority] = append(ruleIDs[:i], ruleIDs[i+1:]...)
			break
		}
	}

	atomic.AddInt64(&r.table.version, 1)

	r.emitUpdate(&RouteUpdate{
		Type:      "rule_deleted",
		RuleID:    ruleID,
		Timestamp: time.Now(),
	})

	_ = rule
	return nil
}

func (r *GeoRouter) UpdateRule(rule *RouteRule) error {
	r.table.mu.Lock()
	defer r.table.mu.Unlock()

	existing, exists := r.table.Rules[rule.ID]
	if !exists {
		return fmt.Errorf("rule not found: %s", rule.ID)
	}

	delete(r.table.Index, existing.Priority)

	existing.Name = rule.Name
	existing.Priority = rule.Priority
	existing.Strategy = rule.Strategy
	existing.SourceRegions = rule.SourceRegions
	existing.SourceCountries = rule.SourceCountries
	existing.TargetRegions = rule.TargetRegions
	existing.Conditions = rule.Conditions
	existing.Actions = rule.Actions
	existing.Weight = rule.Weight
	existing.Enabled = rule.Enabled
	existing.UpdatedAt = time.Now()

	r.table.Index[rule.Priority] = append(r.table.Index[rule.Priority], rule.ID)
	atomic.AddInt64(&r.table.version, 1)

	r.emitUpdate(&RouteUpdate{
		Type:      "rule_updated",
		RuleID:    rule.ID,
		Timestamp: time.Now(),
	})

	return nil
}

func (r *GeoRouter) GetRules() []*RouteRule {
	r.table.mu.RLock()
	defer r.table.mu.RUnlock()

	var rules []*RouteRule
	for _, rule := range r.table.Rules {
		rules = append(rules, rule)
	}

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	return rules
}

func (r *GeoRouter) Route(ctx context.Context, req *RouteRequest) (*RouteResponse, error) {
	startTime := time.Now()

	atomic.AddInt64(&r.metrics.TotalRequests, 1)

	var geoLoc *GeoLocation
	var err error

	if req.GeoLocation != nil {
		geoLoc = req.GeoLocation
	} else if req.SourceIP != "" {
		geoLoc, err = r.nodeManager.ResolveIPRegion(req.SourceIP)
		if err != nil {
			geoLoc = &GeoLocation{IP: req.SourceIP}
		}
	}

	if geoLoc != nil && geoLoc.Region == "" && req.SourceRegion != "" {
		geoLoc.Region = req.SourceRegion
	}
	if geoLoc != nil && geoLoc.Country == "" && req.SourceCountry != "" {
		geoLoc.Country = req.SourceCountry
	}

	candidates, err := r.getCandidates(req, geoLoc)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available nodes for routing")
	}

	matchedRule := r.matchRule(req, geoLoc)

	var scoredRoutes []*Route
	for _, node := range candidates {
		route := &Route{
			NodeID:    node.ID,
			IPAddress: node.PublicIP,
			Port:      node.Port,
			LatencyMs: node.LatencyMs,
		}

		if geoLoc != nil {
			route.DistanceKm = r.nodeManager.CalculateDistance(
				geoLoc.Latitude, geoLoc.Longitude,
				node.Latitude, node.Longitude,
			)
		}

		if node.Metrics != nil {
			route.LoadPercent = node.Metrics.CPUUsage
		}

		route.Score = r.calculateScore(route, matchedRule, req.Strategy)
		scoredRoutes = append(scoredRoutes, route)
	}

	sort.Slice(scoredRoutes, func(i, j int) bool {
		return scoredRoutes[i].Score > scoredRoutes[j].Score
	})

	var primaryRoute *Route
	var backupRoutes []*Route

	if len(scoredRoutes) > 0 {
		primaryRoute = scoredRoutes[0]
		primaryRoute.Timestamp = time.Now()
		primaryRoute.Reason = fmt.Sprintf("Selected by %s strategy", req.Strategy)

		if req.IncludeBackup && len(scoredRoutes) > 1 {
			maxBackups := 3
			if req.MaxResults > 0 && req.MaxResults < maxBackups {
				maxBackups = req.MaxResults
			}
			backupRoutes = scoredRoutes[1:maxBackups]
			for _, route := range backupRoutes {
				route.Timestamp = time.Now()
				route.Reason = "Backup route"
			}
		}
	}

	if primaryRoute != nil {
		r.metrics.mu.Lock()
		r.metrics.StrategyCounts[req.Strategy]++
		if geoLoc != nil {
			r.metrics.RegionCounts[geoLoc.Region]++
		}
		r.metrics.mu.Unlock()

		latencyBucket := int(primaryRoute.LatencyMs / 10)
		if latencyBucket >= 0 && latencyBucket < len(r.metrics.LatencyHistogram) {
			atomic.AddInt64(&r.metrics.LatencyHistogram[latencyBucket], 1)
		}
	}

	return &RouteResponse{
		PrimaryRoute:    primaryRoute,
		BackupRoutes:    backupRoutes,
		MatchedRule:     matchedRule,
		TotalCandidates: len(candidates),
		SelectionTime:   time.Since(startTime),
		Timestamp:       time.Now(),
	}, nil
}

func (r *GeoRouter) getCandidates(req *RouteRequest, geoLoc *GeoLocation) ([]*EdgeNode, error) {
	opts := &NodeSelectorOptions{
		Status:   NodeStatusActive,
		NodeType: req.TargetType,
	}

	return r.nodeManager.ListNodes(opts)
}

func (r *GeoRouter) matchRule(req *RouteRequest, geoLoc *GeoLocation) *RouteRule {
	r.table.mu.RLock()
	defer r.table.mu.RUnlock()

	var matched *RouteRule
	highestPriority := math.MaxInt

	for priority, ruleIDs := range r.table.Index {
		if priority > highestPriority {
			continue
		}

		for _, ruleID := range ruleIDs {
			rule := r.table.Rules[ruleID]
			if !rule.Enabled {
				continue
			}

			if r.evaluateConditions(rule.Conditions, req, geoLoc) {
				if priority < highestPriority {
					highestPriority = priority
					matched = rule
				}
			}
		}
	}

	return matched
}

func (r *GeoRouter) evaluateConditions(conditions []RouteCondition, req *RouteRequest, geoLoc *GeoLocation) bool {
	if len(conditions) == 0 {
		return true
	}

	for _, cond := range conditions {
		if !r.evaluateCondition(cond, req, geoLoc) {
			return false
		}
	}
	return true
}

func (r *GeoRouter) evaluateCondition(cond RouteCondition, req *RouteRequest, geoLoc *GeoLocation) bool {
	var fieldValue interface{}

	switch cond.Field {
	case "source_region":
		if geoLoc != nil {
			fieldValue = string(geoLoc.Region)
		}
	case "source_country":
		if geoLoc != nil {
			fieldValue = geoLoc.Country
		}
	case "target_type":
		fieldValue = string(req.TargetType)
	case "source_ip":
		fieldValue = req.SourceIP
	default:
		if req.Conditions != nil {
			fieldValue = req.Conditions[cond.Field]
		}
	}

	switch cond.Operator {
	case "eq":
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", cond.Value)
	case "neq":
		return fmt.Sprintf("%v", fieldValue) != fmt.Sprintf("%v", cond.Value)
	case "in":
		values, ok := cond.Value.([]interface{})
		if !ok {
			return false
		}
		for _, v := range values {
			if fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", v) {
				return true
			}
		}
		return false
	case "contains":
		return strings.Contains(fmt.Sprintf("%v", fieldValue), fmt.Sprintf("%v", cond.Value))
	case "regex":
		return false
	default:
		return false
	}
}

func (r *GeoRouter) calculateScore(route *Route, rule *RouteRule, strategy RoutingStrategy) float64 {
	var score float64

	switch strategy {
	case RoutingLatency:
		latencyScore := 1.0 - (route.LatencyMs / 500.0)
		if latencyScore < 0 {
			latencyScore = 0
		}
		score = latencyScore * 100

	case RoutingGeo:
		distanceScore := 1.0 - (route.DistanceKm / 20000.0)
		if distanceScore < 0 {
			distanceScore = 0
		}
		score = distanceScore * 100

	case RoutingLoad:
		loadScore := 1.0 - route.LoadPercent
		score = loadScore * 100

	case RoutingPerformance:
		latencyScore := 1.0 - (route.LatencyMs / 500.0)
		loadScore := 1.0 - route.LoadPercent
		distanceScore := 1.0 - (route.DistanceKm / 20000.0)
		if latencyScore < 0 {
			latencyScore = 0
		}
		if distanceScore < 0 {
			distanceScore = 0
		}
		score = (latencyScore*0.4 + loadScore*0.3 + distanceScore*0.3) * 100

	case RoutingCostEffective:
		score = 100.0 - route.LoadPercent*0.5

	case RoutingFailover:
		score = route.LoadPercent * -1
	}

	if rule != nil && rule.Weight > 0 {
		score *= float64(rule.Weight) / 100.0
	}

	return score
}

func (r *GeoRouter) emitUpdate(update *RouteUpdate) {
	select {
	case r.updateChan <- update:
	default:
	}
}

func (r *GeoRouter) SubscribeUpdates() <-chan *RouteUpdate {
	return r.updateChan
}

func (r *GeoRouter) GetMetrics() *RouterMetrics {
	r.metrics.mu.RLock()
	defer r.metrics.mu.RUnlock()

	metricsCopy := &RouterMetrics{
		TotalRequests:   atomic.LoadInt64(&r.metrics.TotalRequests),
		CacheHits:       atomic.LoadInt64(&r.metrics.CacheHits),
		CacheMisses:      atomic.LoadInt64(&r.metrics.CacheMisses),
		StrategyCounts:   make(map[RoutingStrategy]int64),
		RegionCounts:     make(map[Region]int64),
		LatencyHistogram: make([]int64, len(r.metrics.LatencyHistogram)),
	}

	for k, v := range r.metrics.StrategyCounts {
		metricsCopy.StrategyCounts[k] = atomic.LoadInt64(&v)
	}
	for k, v := range r.metrics.RegionCounts {
		metricsCopy.RegionCounts[k] = atomic.LoadInt64(&v)
	}
	copy(metricsCopy.LatencyHistogram, r.metrics.LatencyHistogram)

	return metricsCopy
}

func (r *GeoRouter) GetVersion() int64 {
	return atomic.LoadInt64(&r.table.version)
}

func (r *GeoRouter) SyncToRedis(ctx context.Context) error {
	if r.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	r.table.mu.RLock()
	defer r.table.mu.RUnlock()

	data, err := json.Marshal(r.table.Rules)
	if err != nil {
		return err
	}

	return r.redisClient.Set(ctx, "edge:routes:rules", data, 24*time.Hour).Err()
}

func (r *GeoRouter) SyncFromRedis(ctx context.Context) error {
	if r.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	data, err := r.redisClient.Get(ctx, "edge:routes:rules").Bytes()
	if err != nil {
		return err
	}

	var rules map[string]*RouteRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return err
	}

	r.table.mu.Lock()
	defer r.table.mu.Unlock()

	r.table.Rules = rules
	r.table.Index = make(map[int][]string)
	for _, rule := range rules {
		r.table.Index[rule.Priority] = append(r.table.Index[rule.Priority], rule.ID)
	}

	return nil
}

func (r *GeoRouter) CreateDefaultRules() error {
	defaultRules := []*RouteRule{
		{
			Name:     "Default Low Latency",
			Priority: 100,
			Strategy: RoutingLatency,
			Conditions: []RouteCondition{
				{Type: "default", Field: "all", Operator: "eq", Value: true},
			},
			Weight:  100,
			Enabled: true,
		},
		{
			Name:     "APAC Traffic",
			Priority: 50,
			Strategy: RoutingGeo,
			SourceRegions: []Region{RegionAP, RegionJP, RegionIN, RegionOC},
			TargetRegions: []Region{RegionAP, RegionJP},
			Conditions: []RouteCondition{
				{Type: "region", Field: "source_region", Operator: "in", Value: []interface{}{"ap-east-1", "ap-northeast-1", "ap-south-1", "ap-southeast-1"}},
			},
			Weight:  100,
			Enabled: true,
		},
		{
			Name:     "Europe Traffic",
			Priority: 50,
			Strategy: RoutingGeo,
			SourceRegions: []Region{RegionEU, RegionME},
			TargetRegions: []Region{RegionEU},
			Conditions: []RouteCondition{
				{Type: "region", Field: "source_region", Operator: "in", Value: []interface{}{"eu-west-1"}},
			},
			Weight:  100,
			Enabled: true,
		},
		{
			Name:     "Americas Traffic",
			Priority: 50,
			Strategy: RoutingPerformance,
			SourceRegions: []Region{RegionUS, RegionSA},
			TargetRegions: []Region{RegionUS},
			Conditions: []RouteCondition{
				{Type: "region", Field: "source_region", Operator: "in", Value: []interface{}{"us-east-1", "sa-east-1"}},
			},
			Weight:  100,
			Enabled: true,
		},
		{
			Name:     "China Traffic",
			Priority: 30,
			Strategy: RoutingGeo,
			SourceRegions: []Region{RegionCN},
			TargetRegions: []Region{RegionCN},
			Conditions: []RouteCondition{
				{Type: "region", Field: "source_region", Operator: "eq", Value: "cn-north-1"},
			},
			Weight:  100,
			Enabled: true,
		},
	}

	for _, rule := range defaultRules {
		if err := r.AddRule(rule); err != nil {
			return err
		}
	}

	return nil
}
