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

type IntelligentRouter struct {
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	isRunning     bool
	strategies    map[string]*RoutingStrategyConfig
	currentStrategy string
	analytics     *RoutingAnalytics
	adaptor       *AdaptiveRouter
	metricsCollector *RoutingMetricsCollector
}

type RoutingStrategyConfig struct {
	Name        string
	Type        string
	Priority    int
	Enabled     bool
	Weight      float64
	Conditions  []*RoutingCondition
	Targets     []string
	Metrics     *StrategyMetrics
}

type RoutingCondition struct {
	Type        string
	Field       string
	Operator    string
	Value       interface{}
}

type RoutingRule struct {
	ID          string
	Name        string
	Priority    int
	Strategy    string
	Conditions  []*RoutingCondition
	TargetNodes []string
	TargetRegion string
	Weight      float64
	Enabled     bool
	Stats       *RuleStats
}

type RuleStats struct {
	TotalHits    atomic.Int64
	TotalMisses  atomic.Int64
	AvgLatency   atomic.Int64
	SuccessRate  atomic.Int64
}

type AdaptiveRouter struct {
	mu            sync.RWMutex
	learningRate  float64
	model         *RoutingMLModel
	history       []*RoutingDecision
	maxHistory    int
	currentState  *RouterState
}

type RoutingMLModel struct {
	mu           sync.RWMutex
	weights      map[string]float64
	bias         float64
	features     []string
	trained      bool
	accuracy     float64
}

type RouterState struct {
	TotalDecisions  int64
	CorrectDecisions int64
	AvgLatency      float64
	LoadBalance     float64
	SuccessRate     float64
}

type RoutingDecision struct {
	Timestamp    time.Time
	Request      *RoutingRequest
	SelectedNode *GlobalEdgeNode
	LatencyMs    int64
	Success      bool
	Features     map[string]float64
}

type RoutingRequest struct {
	RequestID     string
	UserID        string
	UserIP        string
	UserLocation  *GeoLocation
	DeviceType    string
	BrowserType   string
	OS            string
	NetworkType   string
	ASNumber      int
	ISP           string
	TLSVersion    int
	RequestSize   int
	Headers       map[string]string
	Priority      int
	Timeout       time.Duration
	Metadata      map[string]interface{}
}

type RoutingResponse struct {
	RequestID   string
	NodeID      string
	NodeAddress string
	Region      string
	LatencyMs   int64
	FromCache   bool
	Success     bool
	Error       string
	Alternatives []*AlternativeNode
}

type AlternativeNode struct {
	NodeID      string
	Address     string
	Region      string
	LoadPercent int
	LatencyMs   int64
	Score       float64
}

type RoutingMetricsCollector struct {
	mu            sync.RWMutex
	totalRequests atomic.Int64
	totalLatency  atomic.Int64
	avgLatency    atomic.Int64
	p99Latency    atomic.Int64
	successCount  atomic.Int64
	failureCount  atomic.Int64
	latencyBuckets map[int64]int64
}

type TrafficManager struct {
	mu            sync.RWMutex
	rules         map[string]*RoutingRule
	activeRules   []*RoutingRule
	weightMap     map[string]float64
	rateLimiters  map[string]*RateLimiter
	circuitBreakers map[string]*CircuitBreaker
}

type RateLimiter struct {
	mu           sync.RWMutex
	maxRequests  int
	windowSize   time.Duration
	requests     []time.Time
	currentCount atomic.Int32
}

type CircuitBreaker struct {
	mu           sync.RWMutex
	state        string
	failureCount atomic.Int32
	successCount atomic.Int32
	threshold    int
	timeout      time.Duration
	lastFailure  time.Time
}

type GeoLocation struct {
	Country     string
	Region      string
	City        string
	Latitude    float64
	Longitude   float64
	Timezone    string
	ASN         int
	ISP         string
}

const (
	ConditionTypeGeo      = "geo"
	ConditionTypeNetwork   = "network"
	ConditionTypeDevice   = "device"
	ConditionTypeTime      = "time"
	ConditionTypeLoad      = "load"
	ConditionTypeLatency   = "latency"

	OperatorEquals      = "eq"
	OperatorNotEquals  = "ne"
	OperatorGreaterThan = "gt"
	OperatorLessThan    = "lt"
	OperatorIn         = "in"
	OperatorContains   = "contains"

	StrategyGeo       = "geo"
	StrategyLatency   = "latency"
	StrategyAI        = "ai"

	CircuitBreakerClosed   = "closed"
	CircuitBreakerOpen     = "open"
	CircuitBreakerHalfOpen = "half_open"
)

func NewIntelligentRouter() *IntelligentRouter {
	ctx, cancel := context.WithCancel(context.Background())

	return &IntelligentRouter{
		ctx:              ctx,
		cancel:           cancel,
		strategies:       make(map[string]*RoutingStrategyConfig),
		currentStrategy:  StrategyWeighted,
		analytics:        NewRoutingAnalytics(),
		adaptor:          NewAdaptiveRouter(),
		metricsCollector: NewRoutingMetricsCollector(),
	}
}

func NewRoutingAnalytics() *RoutingAnalytics {
	return &RoutingAnalytics{
		decisions:  make([]*RoutingDecision, 0, 10000),
		nodeStats:  make(map[string]*NodeRoutingStats),
		regionStats: make(map[string]*RegionRoutingStats),
	}
}

type RoutingAnalytics struct {
	mu          sync.RWMutex
	decisions   []*RoutingDecision
	nodeStats   map[string]*NodeRoutingStats
	regionStats map[string]*RegionRoutingStats
}

type NodeRoutingStats struct {
	NodeID        string
	TotalRequests atomic.Int64
	AvgLatencyMs  atomic.Int64
	P99LatencyMs  atomic.Int64
	SuccessRate   atomic.Int64
	CurrentLoad   atomic.Int32
	LastUpdated   time.Time
}

type RegionRoutingStats struct {
	Region        string
	TotalRequests atomic.Int64
	AvgLatencyMs  atomic.Int64
	ActiveNodes   atomic.Int64
	SuccessRate   atomic.Int64
}

func NewAdaptiveRouter() *AdaptiveRouter {
	return &AdaptiveRouter{
		learningRate: 0.01,
		model: &RoutingMLModel{
			weights:  make(map[string]float64),
			features: []string{"latency", "load", "distance", "capacity"},
		},
		history:     make([]*RoutingDecision, 0, 1000),
		maxHistory:  1000,
		currentState: &RouterState{},
	}
}

func NewRoutingMetricsCollector() *RoutingMetricsCollector {
	return &RoutingMetricsCollector{
		latencyBuckets: make(map[int64]int64),
	}
}

func (r *IntelligentRouter) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isRunning {
		return nil
	}

	r.isRunning = true

	r.initializeStrategies()
	
	go r.runOptimizationLoop()
	go r.metricsCollector.collectLoop(r.ctx)

	log.Println("[IntelligentRouter] Started successfully")
	return nil
}

func (r *IntelligentRouter) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isRunning {
		return
	}

	r.cancel()
	r.isRunning = false
	log.Println("[IntelligentRouter] Stopped")
}

func (r *IntelligentRouter) initializeStrategies() {
	r.strategies[StrategyWeighted] = &RoutingStrategyConfig{
		Name:     "Weighted Round Robin",
		Type:     StrategyWeighted,
		Priority: 1,
		Enabled:  true,
		Weight:   1.0,
		Metrics:  &StrategyMetrics{},
	}

	r.strategies[StrategyGeo] = &RoutingStrategyConfig{
		Name:     "Geo-based Routing",
		Type:     StrategyGeo,
		Priority: 2,
		Enabled:  true,
		Weight:   0.8,
		Metrics:  &StrategyMetrics{},
	}

	r.strategies[StrategyLeastLoad] = &RoutingStrategyConfig{
		Name:     "Least Load Routing",
		Type:    StrategyLeastLoad,
		Priority: 3,
		Enabled:  true,
		Weight:   0.7,
		Metrics:  &StrategyMetrics{},
	}

	r.strategies[StrategyLatency] = &RoutingStrategyConfig{
		Name:     "Latency-based Routing",
		Type:     StrategyLatency,
		Priority: 4,
		Enabled:  true,
		Weight:   0.6,
		Metrics:  &StrategyMetrics{},
	}

	r.strategies[StrategyAI] = &RoutingStrategyConfig{
		Name:     "AI-powered Routing",
		Type:     StrategyAI,
		Priority: 5,
		Enabled:  true,
		Weight:   0.9,
		Metrics:  &StrategyMetrics{},
	}
}

func (r *IntelligentRouter) RouteRequest(ctx context.Context, req *RoutingRequest, nodes []*GlobalEdgeNode) (*RoutingResponse, error) {
	start := time.Now()

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no available nodes")
	}

	var selectedNode *GlobalEdgeNode
	var err error

	switch r.currentStrategy {
	case StrategyAI:
		selectedNode, err = r.adaptor.routeWithAI(ctx, req, nodes)
	case StrategyGeo:
		selectedNode, err = r.routeGeoBased(ctx, req, nodes)
	case StrategyLeastLoad:
		selectedNode, err = r.routeLeastLoad(ctx, nodes)
	case StrategyLatency:
		selectedNode, err = r.routeLatencyBased(ctx, req, nodes)
	default:
		selectedNode, err = r.routeWeighted(ctx, req, nodes)
	}

	if err != nil {
		selectedNode = nodes[0]
	}

	latency := time.Since(start).Milliseconds()

	response := &RoutingResponse{
		RequestID:   req.RequestID,
		NodeID:      selectedNode.ID,
		NodeAddress: fmt.Sprintf("%s:%d", selectedNode.Address, selectedNode.Port),
		Region:      selectedNode.Region,
		LatencyMs:   latency,
		Success:     true,
		Alternatives: r.getAlternativeNodes(nodes, selectedNode),
	}

	r.recordDecision(req, selectedNode, latency, true)
	r.metricsCollector.recordRequest(latency, true)

	return response, nil
}

func (r *IntelligentRouter) routeWeighted(ctx context.Context, req *RoutingRequest, nodes []*GlobalEdgeNode) (*GlobalEdgeNode, error) {
	totalWeight := 0.0
	for _, node := range nodes {
		totalWeight += float64(node.Weight)
	}

	if totalWeight == 0 {
		return nodes[0], nil
	}

	target := req.Hash() % int(totalWeight)
	current := 0.0

	for _, node := range nodes {
		current += float64(node.Weight)
		if float64(target) < current && node.CanAcceptRequest() {
			return node, nil
		}
	}

	return nodes[0], nil
}

func (r *IntelligentRouter) routeGeoBased(ctx context.Context, req *RoutingRequest, nodes []*GlobalEdgeNode) (*GlobalEdgeNode, error) {
	if req.UserLocation == nil {
		return r.routeLeastLoad(ctx, nodes)
	}

	var bestNode *GlobalEdgeNode
	minDistance := math.MaxFloat64

	for _, node := range nodes {
		if !node.CanAcceptRequest() {
			continue
		}

		lat := req.UserLocation.Latitude
		lon := req.UserLocation.Longitude
		distance := haversineDistance(lat, lon, node.Latitude, node.Longitude)

		if distance < minDistance {
			minDistance = distance
			bestNode = node
		}
	}

	if bestNode != nil {
		return bestNode, nil
	}

	return nodes[0], nil
}

func (r *IntelligentRouter) routeLeastLoad(ctx context.Context, nodes []*GlobalEdgeNode) (*GlobalEdgeNode, error) {
	var bestNode *GlobalEdgeNode
	minLoad := math.MaxInt32

	for _, node := range nodes {
		if !node.CanAcceptRequest() {
			continue
		}

		load := node.GetLoadPercent()
		if load < minLoad {
			minLoad = load
			bestNode = node
		}
	}

	if bestNode != nil {
		return bestNode, nil
	}

	return nil, fmt.Errorf("no available nodes")
}

func (r *IntelligentRouter) routeLatencyBased(ctx context.Context, req *RoutingRequest, nodes []*GlobalEdgeNode) (*GlobalEdgeNode, error) {
	var bestNode *GlobalEdgeNode
	minLatency := int64(math.MaxInt64)

	for _, node := range nodes {
		if !node.CanAcceptRequest() {
			continue
		}

		latency := node.Latency.Load()
		if latency < minLatency {
			minLatency = latency
			bestNode = node
		}
	}

	if bestNode != nil {
		return bestNode, nil
	}

	return nodes[0], nil
}

func (r *AdaptiveRouter) routeWithAI(ctx context.Context, req *RoutingRequest, nodes []*GlobalEdgeNode) (*GlobalEdgeNode, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}

	features := r.extractFeatures(req, nodes)
	scores := r.predictScores(features, nodes)

	var bestNode *GlobalEdgeNode
	maxScore := math.MaxFloat64 * -1

	for i, node := range nodes {
		if !node.CanAcceptRequest() {
			continue
		}

		if scores[i] > maxScore {
			maxScore = scores[i]
			bestNode = node
		}
	}

	if bestNode != nil {
		return bestNode, nil
	}

	return nodes[0], nil
}

func (r *AdaptiveRouter) extractFeatures(req *RoutingRequest, nodes []*GlobalEdgeNode) map[string]float64 {
	features := make(map[string]float64)

	features["node_count"] = float64(len(nodes))
	
	if req.UserLocation != nil {
		features["user_lat"] = req.UserLocation.Latitude
		features["user_lon"] = req.UserLocation.Longitude
	}

	if req.NetworkType == "5G" {
		features["fast_network"] = 1.0
	} else if req.NetworkType == "4G" {
		features["fast_network"] = 0.7
	} else {
		features["fast_network"] = 0.3
	}

	features["request_priority"] = float64(req.Priority)

	return features
}

func (r *AdaptiveRouter) predictScores(features map[string]float64, nodes []*GlobalEdgeNode) []float64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	scores := make([]float64, len(nodes))

	for i, node := range nodes {
		score := r.model.bias

		for feature, value := range features {
			if weight, ok := r.model.weights[feature]; ok {
				score += weight * value
			}
		}

		loadScore := float64(100 - node.GetLoadPercent())
		score += loadScore * 0.3

		latencyScore := float64(200 - node.Latency.Load())
		score += latencyScore * 0.2

		scores[i] = score
	}

	return scores
}

func (r *AdaptiveRouter) learnFromOutcome(decision *RoutingDecision) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if decision.Success {
		r.currentState.CorrectDecisions++
	}

	r.currentState.TotalDecisions++
	
	if r.currentState.TotalDecisions > 0 {
		r.currentState.SuccessRate = float64(r.currentState.CorrectDecisions) / float64(r.currentState.TotalDecisions)
	}

	r.history = append(r.history, decision)
	if len(r.history) > r.maxHistory {
		r.history = r.history[1:]
	}

	r.updateModel(decision)
}

func (r *AdaptiveRouter) updateModel(decision *RoutingDecision) {
	if !r.model.trained {
		r.model.weights["latency"] = 0.3
		r.model.weights["load"] = 0.4
		r.model.weights["distance"] = 0.2
		r.model.weights["capacity"] = 0.1
		r.model.trained = true
	}

	for feature, value := range decision.Features {
		if _, exists := r.model.weights[feature]; exists {
			if decision.Success {
				r.model.weights[feature] += r.learningRate * value
			} else {
				r.model.weights[feature] -= r.learningRate * value
			}
		}
	}
}

func (r *IntelligentRouter) getAlternativeNodes(allNodes []*GlobalEdgeNode, selected *GlobalEdgeNode) []*AlternativeNode {
	var alternatives []*AlternativeNode

	for _, node := range allNodes {
		if node.ID == selected.ID || !node.CanAcceptRequest() {
			continue
		}

		alternatives = append(alternatives, &AlternativeNode{
			NodeID:      node.ID,
			Address:     node.Address,
			Region:      node.Region,
			LoadPercent: node.GetLoadPercent(),
			LatencyMs:   node.Latency.Load(),
			Score:       float64(100 - node.GetLoadPercent()),
		})

		if len(alternatives) >= 3 {
			break
		}
	}

	return alternatives
}

func (r *IntelligentRouter) recordDecision(req *RoutingRequest, node *GlobalEdgeNode, latencyMs int64, success bool) {
	r.analytics.mu.Lock()
	defer r.analytics.mu.Unlock()

	decision := &RoutingDecision{
		Timestamp:    time.Now(),
		Request:      req,
		SelectedNode: node,
		LatencyMs:    latencyMs,
		Success:      success,
	}

	r.analytics.decisions = append(r.analytics.decisions, decision)
	if len(r.analytics.decisions) > 10000 {
		r.analytics.decisions = r.analytics.decisions[1:]
	}

	if _, exists := r.analytics.nodeStats[node.ID]; !exists {
		r.analytics.nodeStats[node.ID] = &NodeRoutingStats{NodeID: node.ID}
	}
	r.analytics.nodeStats[node.ID].TotalRequests.Add(1)
}

func (r *IntelligentRouter) runOptimizationLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.optimizeStrategies()
		}
	}
}

func (r *IntelligentRouter) optimizeStrategies() {
	r.mu.Lock()
	defer r.mu.Unlock()

	var bestStrategy string
	var bestWeight float64 = 0

	for name, strategy := range r.strategies {
		if !strategy.Enabled {
			continue
		}

		metrics := strategy.Metrics
		if metrics != nil && float64(metrics.SuccessRate.Load()) > bestWeight {
			bestWeight = float64(metrics.SuccessRate.Load())
			bestStrategy = name
		}
	}

	if bestStrategy != "" && bestStrategy != r.currentStrategy {
		log.Printf("[IntelligentRouter] Switching strategy from %s to %s", r.currentStrategy, bestStrategy)
		r.currentStrategy = bestStrategy
	}
}

func (m *RoutingMetricsCollector) recordRequest(latencyMs int64, success bool) {
	m.totalRequests.Add(1)
	m.totalLatency.Add(latencyMs)

	if m.totalRequests.Load() > 0 {
		m.avgLatency.Store(m.totalLatency.Load() / m.totalRequests.Load())
	}

	if success {
		m.successCount.Add(1)
	} else {
		m.failureCount.Add(1)
	}

	bucket := (latencyMs / 10) * 10
	m.latencyBuckets[bucket]++
}

func (m *RoutingMetricsCollector) collectLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.calculateP99()
		}
	}
}

func (m *RoutingMetricsCollector) calculateP99() {
	m.mu.Lock()
	defer m.mu.Unlock()

	var total int64
	var p99Index int64

	for _, count := range m.latencyBuckets {
		total += count
	}

	p99Threshold := int64(float64(total) * 0.99)
	running := int64(0)

	for latency, count := range m.latencyBuckets {
		running += count
		if running >= p99Threshold {
			if latency > p99Index {
				p99Index = latency
			}
			break
		}
	}

	m.p99Latency.Store(p99Index)
}

func (r *IntelligentRouter) AddRule(ctx context.Context, rule *RoutingRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule_%d", time.Now().UnixNano())
	}
	
	rule.Stats = &RuleStats{}

	r.strategies[rule.ID] = &RoutingStrategyConfig{
		Name:     rule.Name,
		Type:     rule.Strategy,
		Priority: rule.Priority,
		Enabled:  rule.Enabled,
		Weight:   rule.Weight,
		Metrics:  &StrategyMetrics{},
	}

	return nil
}

func (r *IntelligentRouter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"current_strategy": r.currentStrategy,
		"total_requests":    r.metricsCollector.totalRequests.Load(),
		"avg_latency_ms":    r.metricsCollector.avgLatency.Load(),
		"p99_latency_ms":    r.metricsCollector.p99Latency.Load(),
		"success_rate":      r.calculateSuccessRate(),
	}
}

func (r *IntelligentRouter) calculateSuccessRate() float64 {
	total := r.metricsCollector.totalRequests.Load()
	if total == 0 {
		return 1.0
	}
	success := r.metricsCollector.successCount.Load()
	return float64(success) / float64(total)
}

type StrategyMetrics struct {
	TotalRequests  atomic.Int64
	SuccessRate    atomic.Int64
	AvgLatencyMs   atomic.Int64
	P99LatencyMs   atomic.Int64
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
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

func (req *RoutingRequest) Hash() int {
	hash := 0
	for _, c := range req.UserID {
		hash = 31*hash + int(c)
	}
	return hash
}
