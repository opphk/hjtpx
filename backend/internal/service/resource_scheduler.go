package service

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type ResourceScheduler struct {
	mu               sync.RWMutex
	k8sAutoscaler   *K8sAutoscaler
	serviceMesh      *ServiceMeshManager
	predictor        *ResourcePredictor
	costOptimizer    *CostPerformanceOptimizer
	metrics          *SchedulerMetrics
	config           *SchedulerConfig
	initialized      atomic.Bool
	startTime        time.Time
	replicas         atomic.Int64
	targetReplicas   atomic.Int64
	metricsCollector *MetricsCollector
}

type K8sAutoscaler struct {
	mu               sync.RWMutex
	minReplicas      int
	maxReplicas      int
	currentReplicas  atomic.Int64
	targetReplicas   atomic.Int64
	metrics          *K8sAutoscalerMetrics
	scaleUpPolicy    *ScalingPolicy
	scaleDownPolicy  *ScalingPolicy
	lastScaleTime    time.Time
	scaleCooldown    time.Duration
	enabled          atomic.Bool
	metricsHistory   []ResourceMetric
	hpaEnabled       atomic.Bool
}

type K8sAutoscalerMetrics struct {
	ScaleEvents      atomic.Int64
	ScaleUpEvents    atomic.Int64
	ScaleDownEvents  atomic.Int64
	CurrentCPUUsage  atomic.Value
	CurrentMemUsage  atomic.Value
	CurrentQPS       atomic.Value
	AvgLatencyMs     atomic.Value
	TargetLatencyMs  atomic.Value
	StabilizationWindow time.Duration
}

type ScalingPolicy struct {
	MetricName       string
	TargetValue      float64
	Tolerance        float64
	EvaluationPeriod time.Duration
}

type ServiceMeshManager struct {
	mu            sync.RWMutex
	trafficRules  map[string]*TrafficRule
	routingTable  *RoutingTable
	circuitBreakers map[string]*CircuitBreaker
	canaryConfigs  map[string]*CanaryConfig
	metrics        *ServiceMeshMetrics
	enabled        atomic.Bool
}

type TrafficRule struct {
	Name         string
	Source       string
	Destination  string
	Weight       float64
	Headers      map[string]string
	Timeout      time.Duration
	Retries      int
	MirrorRatio  float64
}

type RoutingTable struct {
	Entries    map[string][]*RouteEntry
	mu         sync.RWMutex
	version    atomic.Int64
}

type RouteEntry struct {
	Service    string
	Subset     string
	Weight     float64
	Label      map[string]string
}

type CircuitBreaker struct {
	Name           string
	State          CircuitState
	FailureThreshold int
	SuccessThreshold int
	Timeout        time.Duration
	MaxConnections int
	RequestCount   atomic.Int64
	FailureCount   atomic.Int64
	SuccessCount   atomic.Int64
	LastFailure    time.Time
	StateChanged   atomic.Value
}

type CircuitState int

const (
	CircuitStateClosed CircuitState = iota
	CircuitStateOpen
	CircuitStateHalfOpen
)

type CanaryConfig struct {
	Name           string
	Version        string
	Weight         float64
	MaxWeight      float64
	StepWeight     float64
	AnalysisWindow time.Duration
	SuccessRate    float64
	ErrorRate      float64
	MetricsThreshold map[string]float64
	AutoPromote    bool
}

type ServiceMeshMetrics struct {
	TotalRequests    atomic.Int64
	FailedRequests   atomic.Int64
	RoutedRequests   atomic.Int64
	CircuitBreakerTrips atomic.Int64
	CanaryPromotions atomic.Int64
}

type ResourcePredictor struct {
	mu           sync.RWMutex
	models       map[string]*PredictionModelV2
	metrics      *PredictionMetrics
	windowSize   time.Duration
	predictionHorizon time.Duration
	retrainingInterval time.Duration
	enabled      atomic.Bool
}

type PredictionModelV2 struct {
	Name         string
	Type         ModelType
	DataPoints   []MetricDataPoint
	TickSize     time.Duration
	IsTrained    atomic.Bool
	Accuracy     float64
	LastTrained  time.Time
	Features     []string
}

type MetricDataPoint struct {
	Timestamp time.Time
	Value     float64
}

type ModelType int

const (
	ModelTypeLinear ModelType = iota
	ModelTypeExponential
	ModelTypePolynomial
	ModelTypeNeuralNetwork
)

type PredictionMetrics struct {
	PredictionsMade    atomic.Int64
	PredictionErrors   atomic.Int64
	AvgPredictionError float64
	RetrainingCount    atomic.Int64
}

type CostPerformanceOptimizer struct {
	mu            sync.RWMutex
	enabled       atomic.Bool
	balanceMode   BalanceMode
	costWeights   *CostWeights
	perfWeights   *PerformanceWeights
	currentCost   atomic.Value
	currentPerf   atomic.Value
	optimizationHistory []OptimizationDecision
	metrics       *CostMetrics
}

type BalanceMode int

const (
	BalanceModeCostOptimized BalanceMode = iota
	BalanceModePerformanceOptimized
	BalanceModeBalanced
	BalanceModeDynamic
)

type CostWeights struct {
	ComputeCost   float64
	MemoryCost    float64
	NetworkCost   float64
	StorageCost   float64
	HourlyRate    float64
}

type PerformanceWeights struct {
	LatencyWeight    float64
	ThroughputWeight float64
	AvailabilityWeight float64
	ReliabilityWeight float64
}

type OptimizationDecision struct {
	Timestamp      time.Time
	DecisionType   string
	Changes        map[string]interface{}
	CostImpact     float64
	PerfImpact     float64
	Reason         string
}

type CostMetrics struct {
	TotalCost          atomic.Value
	DailyCost          float64
	HourlyCost         atomic.Value
	ProjectedMonthlyCost atomic.Value
	CostPerRequest     atomic.Value
	CostSavings        float64
}

type SchedulerMetrics struct {
	TotalSchedules    atomic.Int64
	SuccessfulSchedules atomic.Int64
	FailedSchedules   atomic.Int64
	ScaleOperations   atomic.Int64
	ReplicasChange    atomic.Int64
	ResourceEfficiency float64
	CostEfficiency    float64
	AvgQueueTime      atomic.Int64
	AvgWaitTime       atomic.Int64
}

type MetricsCollector struct {
	mu           sync.RWMutex
	metrics      map[string]*ResourceMetric
	collectionInterval time.Duration
	retentionPeriod   time.Duration
	enabled       atomic.Bool
}

type ResourceMetric struct {
	Timestamp   time.Time
	CPUUsage    float64
	MemoryUsage float64
	DiskIO      float64
	NetworkIO   float64
	RequestRate float64
	Latency     float64
	ErrorRate   float64
}

type SchedulerConfig struct {
	EnableK8sAutoscaler    bool
	EnableServiceMesh      bool
	EnablePrediction       bool
	EnableCostOptimization bool
	MinReplicas           int
	MaxReplicas           int
	ScaleUpThreshold      float64
	ScaleDownThreshold    float64
	ScaleCooldown         time.Duration
	TargetLatency         time.Duration
	TargetQPS             int64
	PredictionWindow      time.Duration
	BalanceMode           BalanceMode
}

const (
	DefaultMinReplicas = 1
	DefaultMaxReplicas = 100
	DefaultScaleCooldown = 5 * time.Minute
	DefaultTargetLatency = 100 * time.Millisecond
)

func NewResourceScheduler(config *SchedulerConfig) *ResourceScheduler {
	if config == nil {
		config = &SchedulerConfig{
			EnableK8sAutoscaler:    true,
			EnableServiceMesh:       true,
			EnablePrediction:        true,
			EnableCostOptimization:  true,
			MinReplicas:            DefaultMinReplicas,
			MaxReplicas:            DefaultMaxReplicas,
			ScaleUpThreshold:       0.7,
			ScaleDownThreshold:     0.3,
			ScaleCooldown:          DefaultScaleCooldown,
			TargetLatency:          DefaultTargetLatency,
			TargetQPS:              10000,
			PredictionWindow:       10 * time.Minute,
			BalanceMode:            BalanceModeBalanced,
		}
	}

	scheduler := &ResourceScheduler{
		config:      config,
		metrics:     &SchedulerMetrics{},
		startTime:    time.Now(),
		replicas:    atomic.Int64{},
		targetReplicas: atomic.Int64{},
		metricsCollector: NewMetricsCollector(10 * time.Second),
	}

	if config.EnableK8sAutoscaler {
		scheduler.k8sAutoscaler = NewK8sAutoscaler(config)
	}

	if config.EnableServiceMesh {
		scheduler.serviceMesh = NewServiceMeshManager()
	}

	if config.EnablePrediction {
		scheduler.predictor = NewResourcePredictor(config.PredictionWindow)
	}

	if config.EnableCostOptimization {
		scheduler.costOptimizer = NewCostPerformanceOptimizer(config.BalanceMode)
	}

	scheduler.k8sAutoscaler.currentReplicas.Store(int64(config.MinReplicas))
	scheduler.replicas.Store(int64(config.MinReplicas))
	scheduler.targetReplicas.Store(int64(config.MinReplicas))

	scheduler.metricsCollector.Start()

	scheduler.initialized.Store(true)
	return scheduler
}

func NewK8sAutoscaler(config *SchedulerConfig) *K8sAutoscaler {
	autoscaler := &K8sAutoscaler{
		minReplicas:     config.MinReplicas,
		maxReplicas:     config.MaxReplicas,
		metrics:         &K8sAutoscalerMetrics{},
		scaleCooldown:   config.ScaleCooldown,
		enabled:         atomic.Bool{},
		metricsHistory:  make([]ResourceMetric, 0),
		hpaEnabled:      atomic.Bool{},
	}

	autoscaler.enabled.Store(config.EnableK8sAutoscaler)
	autoscaler.scaleUpPolicy = &ScalingPolicy{
		MetricName:       "cpu_usage",
		TargetValue:      config.ScaleUpThreshold,
		Tolerance:        0.1,
		EvaluationPeriod: 1 * time.Minute,
	}

	autoscaler.scaleDownPolicy = &ScalingPolicy{
		MetricName:       "cpu_usage",
		TargetValue:      config.ScaleDownThreshold,
		Tolerance:        0.1,
		EvaluationPeriod: 5 * time.Minute,
	}

	return autoscaler
}

func (a *K8sAutoscaler) Scale(ctx context.Context, currentMetrics *ResourceMetric) (*ScaleRecommendation, error) {
	if !a.enabled.Load() {
		return &ScaleRecommendation{
			Action: ScaleActionNoChange,
			Reason: "autoscaler disabled",
		}, nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if time.Since(a.lastScaleTime) < a.scaleCooldown {
		return &ScaleRecommendation{
			Action: ScaleActionNoChange,
			Reason: "in cooldown period",
		}, nil
	}

	a.metricsHistory = append(a.metricsHistory, *currentMetrics)
	if len(a.metricsHistory) > 100 {
		a.metricsHistory = a.metricsHistory[len(a.metricsHistory)-100:]
	}

	currentReplicas := int(a.currentReplicas.Load())
	avgCPU := a.calculateAverageCPU()
	avgMem := a.calculateAverageMemory()

	a.metrics.CurrentCPUUsage.Store(avgCPU)
	a.metrics.CurrentMemUsage.Store(avgMem)

	if avgCPU > a.scaleUpPolicy.TargetValue {
		newReplicas := a.calculateScaleUpReplicas(avgCPU, currentReplicas)
		if newReplicas > currentReplicas && newReplicas <= a.maxReplicas {
			a.currentReplicas.Store(int64(newReplicas))
			a.targetReplicas.Store(int64(newReplicas))
			a.lastScaleTime = time.Now()
			a.metrics.ScaleUpEvents.Add(1)
			a.metrics.ScaleEvents.Add(1)

			return &ScaleRecommendation{
				Action:       ScaleActionScaleUp,
				NewReplicas:  newReplicas,
				Reason:       fmt.Sprintf("CPU usage %.2f above threshold %.2f", avgCPU, a.scaleUpPolicy.TargetValue),
				CurrentCPU:   avgCPU,
				CurrentMem:   avgMem,
			}, nil
		}
	}

	if avgCPU < a.scaleDownPolicy.TargetValue && currentReplicas > a.minReplicas {
		newReplicas := a.calculateScaleDownReplicas(avgCPU, currentReplicas)
		if newReplicas < currentReplicas && newReplicas >= a.minReplicas {
			a.currentReplicas.Store(int64(newReplicas))
			a.targetReplicas.Store(int64(newReplicas))
			a.lastScaleTime = time.Now()
			a.metrics.ScaleDownEvents.Add(1)
			a.metrics.ScaleEvents.Add(1)

			return &ScaleRecommendation{
				Action:      ScaleActionScaleDown,
				NewReplicas: newReplicas,
				Reason:      fmt.Sprintf("CPU usage %.2f below threshold %.2f", avgCPU, a.scaleDownPolicy.TargetValue),
				CurrentCPU:  avgCPU,
				CurrentMem:  avgMem,
			}, nil
		}
	}

	return &ScaleRecommendation{
		Action: ScaleActionNoChange,
		Reason: "metrics within acceptable range",
	}, nil
}

type ScaleRecommendation struct {
	Action      ScaleAction
	NewReplicas int
	Reason      string
	CurrentCPU  float64
	CurrentMem  float64
}

type ScaleAction int

const (
	ScaleActionNoChange ScaleAction = iota
	ScaleActionScaleUp
	ScaleActionScaleDown
)

func (a *K8sAutoscaler) calculateAverageCPU() float64 {
	if len(a.metricsHistory) == 0 {
		return 0
	}

	var sum float64
	for _, m := range a.metricsHistory {
		sum += m.CPUUsage
	}

	return sum / float64(len(a.metricsHistory))
}

func (a *K8sAutoscaler) calculateAverageMemory() float64 {
	if len(a.metricsHistory) == 0 {
		return 0
	}

	var sum float64
	for _, m := range a.metricsHistory {
		sum += m.MemoryUsage
	}

	return sum / float64(len(a.metricsHistory))
}

func (a *K8sAutoscaler) calculateScaleUpReplicas(cpuUsage float64, currentReplicas int) int {
	scaleFactor := cpuUsage / a.scaleUpPolicy.TargetValue
	newReplicas := int(math.Ceil(float64(currentReplicas) * scaleFactor))

	if newReplicas == currentReplicas {
		newReplicas = currentReplicas + 1
	}

	return newReplicas
}

func (a *K8sAutoscaler) calculateScaleDownReplicas(cpuUsage float64, currentReplicas int) int {
	scaleFactor := a.scaleDownPolicy.TargetValue / cpuUsage
	newReplicas := int(math.Ceil(float64(currentReplicas) * scaleFactor))

	if newReplicas == currentReplicas {
		newReplicas = currentReplicas - 1
	}

	return newReplicas
}

func (a *K8sAutoscaler) GetMetrics() *K8sAutoscalerMetrics {
	return &K8sAutoscalerMetrics{
		ScaleEvents:     a.metrics.ScaleEvents,
		ScaleUpEvents:   a.metrics.ScaleUpEvents,
		ScaleDownEvents: a.metrics.ScaleDownEvents,
		CurrentCPUUsage: a.metrics.CurrentCPUUsage,
		CurrentMemUsage: a.metrics.CurrentMemUsage,
		CurrentQPS:     a.metrics.CurrentQPS,
		AvgLatencyMs:   a.metrics.AvgLatencyMs,
		TargetLatencyMs: a.metrics.TargetLatencyMs,
	}
}

func (a *K8sAutoscaler) SetReplicas(replicas int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if replicas < a.minReplicas {
		replicas = a.minReplicas
	}
	if replicas > a.maxReplicas {
		replicas = a.maxReplicas
	}

	a.currentReplicas.Store(int64(replicas))
	a.targetReplicas.Store(int64(replicas))
}

func (a *K8sAutoscaler) Enable() {
	a.enabled.Store(true)
}

func (a *K8sAutoscaler) Disable() {
	a.enabled.Store(false)
}

func NewServiceMeshManager() *ServiceMeshManager {
	return &ServiceMeshManager{
		trafficRules:     make(map[string]*TrafficRule),
		routingTable:    &RoutingTable{Entries: make(map[string][]*RouteEntry)},
		circuitBreakers:  make(map[string]*CircuitBreaker),
		canaryConfigs:    make(map[string]*CanaryConfig),
		metrics:          &ServiceMeshMetrics{},
		enabled:          atomic.Bool{},
	}
}

func (m *ServiceMeshManager) Enable() {
	m.enabled.Store(true)
}

func (m *ServiceMeshManager) Disable() {
	m.enabled.Store(false)
}

func (m *ServiceMeshManager) AddTrafficRule(rule *TrafficRule) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.trafficRules[rule.Name] = rule
	m.updateRoutingTable()
}

func (m *ServiceMeshManager) updateRoutingTable() {
	m.routingTable.mu.Lock()
	defer m.routingTable.mu.Unlock()

	for name, rule := range m.trafficRules {
		entry := &RouteEntry{
			Service: rule.Destination,
			Weight:  rule.Weight,
		}

		m.routingTable.Entries[name] = []*RouteEntry{entry}
		m.routingTable.version.Add(1)
	}
}

func (m *ServiceMeshManager) GetRoute(service string) (*RouteEntry, error) {
	m.routingTable.mu.RLock()
	defer m.routingTable.mu.RUnlock()

	entries, exists := m.routingTable.Entries[service]
	if !exists || len(entries) == 0 {
		return nil, fmt.Errorf("no route found for service: %s", service)
	}

	return entries[0], nil
}

func (m *ServiceMeshManager) AddCircuitBreaker(cb *CircuitBreaker) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cb.StateChanged.Store(time.Now())
	m.circuitBreakers[cb.Name] = cb
}

func (m *ServiceMeshManager) RecordRequest(service string, success bool) {
	m.metrics.TotalRequests.Add(1)
	if !success {
		m.metrics.FailedRequests.Add(1)
	}
	m.metrics.RoutedRequests.Add(1)

	m.mu.RLock()
	cb, exists := m.circuitBreakers[service]
	m.mu.RUnlock()

	if exists {
		cb.RequestCount.Add(1)
		if success {
			cb.SuccessCount.Add(1)
			m.handleCircuitBreakerSuccess(cb)
		} else {
			cb.FailureCount.Add(1)
			cb.LastFailure = time.Now()
			m.handleCircuitBreakerFailure(cb)
		}
	}
}

func (m *ServiceMeshManager) handleCircuitBreakerFailure(cb *CircuitBreaker) {
	if cb.State == CircuitStateClosed && cb.FailureCount.Load() >= int64(cb.FailureThreshold) {
		cb.State = CircuitStateOpen
		cb.StateChanged.Store(time.Now())
		m.metrics.CircuitBreakerTrips.Add(1)
	}
}

func (m *ServiceMeshManager) handleCircuitBreakerSuccess(cb *CircuitBreaker) {
	if cb.State == CircuitStateHalfOpen && cb.SuccessCount.Load() >= int64(cb.SuccessThreshold) {
		cb.State = CircuitStateClosed
		cb.StateChanged.Store(time.Now())
		cb.FailureCount.Store(0)
		cb.SuccessCount.Store(0)
	}
}

func (m *ServiceMeshManager) GetCircuitBreakerState(service string) CircuitState {
	m.mu.RLock()
	cb, exists := m.circuitBreakers[service]
	m.mu.RUnlock()

	if !exists {
		return CircuitStateClosed
	}

	if cb.State == CircuitStateOpen && time.Since(cb.LastFailure) > cb.Timeout {
		cb.State = CircuitStateHalfOpen
		cb.StateChanged.Store(time.Now())
	}

	return cb.State
}

func (m *ServiceMeshManager) AddCanaryConfig(config *CanaryConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.canaryConfigs[config.Name] = config
}

func (m *ServiceMeshManager) UpdateCanaryWeight(name string, weight float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config, exists := m.canaryConfigs[name]; exists {
		config.Weight = weight
	}
}

func (m *ServiceMeshManager) GetMetrics() *ServiceMeshMetrics {
	return &ServiceMeshMetrics{
		TotalRequests:    m.metrics.TotalRequests,
		FailedRequests:   m.metrics.FailedRequests,
		RoutedRequests:   m.metrics.RoutedRequests,
		CircuitBreakerTrips: m.metrics.CircuitBreakerTrips,
		CanaryPromotions: m.metrics.CanaryPromotions,
	}
}

func NewResourcePredictor(windowSize time.Duration) *ResourcePredictor {
	predictor := &ResourcePredictor{
		models:            make(map[string]*PredictionModelV2),
		metrics:           &PredictionMetrics{},
		windowSize:        windowSize,
		predictionHorizon: 5 * time.Minute,
		retrainingInterval: 1 * time.Hour,
		enabled:           atomic.Bool{},
	}

	predictor.enabled.Store(true)

	predictor.models["cpu"] = &PredictionModelV2{
		Name:     "cpu",
		Type:     ModelTypeLinear,
		TickSize: 1 * time.Minute,
		Features: []string{"historical_cpu", "time_of_day", "day_of_week"},
	}

	predictor.models["memory"] = &PredictionModelV2{
		Name:     "memory",
		Type:     ModelTypeLinear,
		TickSize: 1 * time.Minute,
		Features: []string{"historical_memory", "time_of_day"},
	}

	predictor.models["qps"] = &PredictionModelV2{
		Name:     "qps",
		Type:     ModelTypeExponential,
		TickSize: 1 * time.Minute,
		Features: []string{"historical_qps", "trend", "seasonality"},
	}

	return predictor
}

func (p *ResourcePredictor) AddDataPoint(metricType string, value float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	model, exists := p.models[metricType]
	if !exists {
		return
	}

	point := MetricDataPoint{
		Timestamp: time.Now(),
		Value:     value,
	}

	model.DataPoints = append(model.DataPoints, point)

	retention := int(p.windowSize / model.TickSize)
	if len(model.DataPoints) > retention {
		model.DataPoints = model.DataPoints[len(model.DataPoints)-retention:]
	}
}

func (p *ResourcePredictor) Predict(metricType string, horizon time.Duration) (float64, error) {
	if !p.enabled.Load() {
		return 0, fmt.Errorf("predictor disabled")
	}

	p.metrics.PredictionsMade.Add(1)

	p.mu.RLock()
	model, exists := p.models[metricType]
	p.mu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("unknown metric type: %s", metricType)
	}

	if len(model.DataPoints) < 10 {
		return 0, fmt.Errorf("insufficient data for prediction")
	}

	prediction := p.linearPrediction(model.DataPoints, horizon)

	return prediction, nil
}

func (p *ResourcePredictor) linearPrediction(dataPoints []MetricDataPoint, horizon time.Duration) float64 {
	if len(dataPoints) < 2 {
		return 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(dataPoints))

	baseTime := dataPoints[0].Timestamp
	for _, dp := range dataPoints {
		x := float64(dp.Timestamp.Sub(baseTime).Milliseconds()) / 1000.0
		y := dp.Value

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denom := n*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-10 {
		return sumY / n
	}

	slope := (n*sumXY - sumX*sumY) / denom
	intercept := (sumY - slope*sumX) / n

	predictionTime := horizon.Seconds()
	prediction := intercept + slope*predictionTime

	if prediction < 0 {
		prediction = 0
	}

	return prediction
}

func (p *ResourcePredictor) GetMetrics() *PredictionMetrics {
	return &PredictionMetrics{
		PredictionsMade:    p.metrics.PredictionsMade,
		PredictionErrors:   p.metrics.PredictionErrors,
		AvgPredictionError: p.metrics.AvgPredictionError,
		RetrainingCount:    p.metrics.RetrainingCount,
	}
}

func NewCostPerformanceOptimizer(balanceMode BalanceMode) *CostPerformanceOptimizer {
	optimizer := &CostPerformanceOptimizer{
		enabled:            atomic.Bool{},
		balanceMode:        balanceMode,
		costWeights:        &CostWeights{},
		perfWeights:        &PerformanceWeights{},
		optimizationHistory: make([]OptimizationDecision, 0),
		metrics:            &CostMetrics{},
	}

	optimizer.costWeights.ComputeCost = 0.05
	optimizer.costWeights.MemoryCost = 0.01
	optimizer.costWeights.NetworkCost = 0.005
	optimizer.costWeights.StorageCost = 0.0001
	optimizer.costWeights.HourlyRate = 0.5

	optimizer.perfWeights.LatencyWeight = 0.4
	optimizer.perfWeights.ThroughputWeight = 0.3
	optimizer.perfWeights.AvailabilityWeight = 0.2
	optimizer.perfWeights.ReliabilityWeight = 0.1

	optimizer.enabled.Store(true)

	return optimizer
}

func (o *CostPerformanceOptimizer) CalculateOptimalResources(currentMetrics *ResourceMetric) (*ResourceAllocationV2, error) {
	if !o.enabled.Load() {
		return nil, fmt.Errorf("optimizer disabled")
	}

	var costScore, perfScore float64

	costScore = o.calculateCostScore(currentMetrics)
	perfScore = o.calculatePerformanceScore(currentMetrics)

	o.currentCost.Store(costScore)
	o.currentPerf.Store(perfScore)

	// Apply balance mode adjustments
	switch o.balanceMode {
	case BalanceModeCostOptimized:
		costScore *= 1.2
	case BalanceModePerformanceOptimized:
		perfScore *= 1.2
	case BalanceModeBalanced:
		// Use default balanced approach
	default:
		// Dynamic mode handled separately
	}

	allocation := &ResourceAllocationV2{
		CPU:       o.calculateOptimalCPU(currentMetrics, perfScore),
		Memory:    o.calculateOptimalMemory(currentMetrics),
		Replicas:  int(o.calculateOptimalReplicas(currentMetrics, perfScore)),
		CostEstimate: costScore,
		PerformanceScore: perfScore,
	}

	decision := OptimizationDecision{
		Timestamp:    time.Now(),
		DecisionType: "resource_allocation",
		Changes: map[string]interface{}{
			"cpu":      allocation.CPU,
			"memory":   allocation.Memory,
			"replicas": allocation.Replicas,
		},
		CostImpact: costScore,
		PerfImpact: perfScore,
		Reason:     "calculated optimal allocation",
	}

	o.mu.Lock()
	o.optimizationHistory = append(o.optimizationHistory, decision)
	if len(o.optimizationHistory) > 100 {
		o.optimizationHistory = o.optimizationHistory[len(o.optimizationHistory)-100:]
	}
	o.mu.Unlock()

	o.updateCostMetrics(allocation)

	return allocation, nil
}

type ResourceAllocationV2 struct {
	CPU              float64
	Memory           float64
	Replicas         int
	CostEstimate     float64
	PerformanceScore float64
}

func (o *CostPerformanceOptimizer) calculateCostScore(metrics *ResourceMetric) float64 {
	cpuCost := metrics.CPUUsage * o.costWeights.ComputeCost
	memCost := metrics.MemoryUsage * o.costWeights.MemoryCost
	netCost := metrics.NetworkIO * o.costWeights.NetworkCost

	totalCost := cpuCost + memCost + netCost + o.costWeights.HourlyRate

	return totalCost
}

func (o *CostPerformanceOptimizer) calculatePerformanceScore(metrics *ResourceMetric) float64 {
	latencyScore := 1.0 / (1.0 + metrics.Latency/1000.0)
	throughputScore := math.Min(metrics.RequestRate/10000.0, 1.0)
	errorPenalty := metrics.ErrorRate * 0.1

	score := (latencyScore*o.perfWeights.LatencyWeight +
		throughputScore*o.perfWeights.ThroughputWeight) -
		errorPenalty*o.perfWeights.ReliabilityWeight

	return math.Max(0, math.Min(1, score))
}

func (o *CostPerformanceOptimizer) calculateOptimalCPU(metrics *ResourceMetric, perfScore float64) float64 {
	optimalCPU := metrics.CPUUsage * 0.8

	if perfScore < 0.5 {
		optimalCPU *= 1.2
	}

	return math.Min(optimalCPU, 1.0)
}

func (o *CostPerformanceOptimizer) calculateOptimalMemory(metrics *ResourceMetric) float64 {
	optimalMemory := metrics.MemoryUsage * 0.85
	return math.Min(optimalMemory, 1.0)
}

func (o *CostPerformanceOptimizer) calculateOptimalReplicas(metrics *ResourceMetric, perfScore float64) float64 {
	baseReplicas := 3

	if metrics.Latency > 500 {
		baseReplicas += 2
	}

	if perfScore < 0.7 {
		baseReplicas += 1
	}

	if metrics.RequestRate > 5000 {
		baseReplicas += int(metrics.RequestRate / 5000)
	}

	return math.Min(float64(baseReplicas), 50)
}

func (o *CostPerformanceOptimizer) updateCostMetrics(allocation *ResourceAllocationV2) {
	hourlyCost := allocation.CostEstimate
	o.metrics.TotalCost.Store(float64(hourlyCost))
	o.metrics.HourlyCost.Store(float64(hourlyCost))
	o.metrics.ProjectedMonthlyCost.Store(float64(hourlyCost * 24 * 30))
}

func (o *CostPerformanceOptimizer) GetMetrics() *CostMetrics {
	return &CostMetrics{
		TotalCost:          o.metrics.TotalCost,
		DailyCost:          o.metrics.DailyCost,
		HourlyCost:         o.metrics.HourlyCost,
		ProjectedMonthlyCost: o.metrics.ProjectedMonthlyCost,
		CostPerRequest:     o.metrics.CostPerRequest,
		CostSavings:        o.metrics.CostSavings,
	}
}

func NewMetricsCollector(interval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		metrics:          make(map[string]*ResourceMetric),
		collectionInterval: interval,
		retentionPeriod:   24 * time.Hour,
		enabled:          atomic.Bool{},
	}
}

func (c *MetricsCollector) Enable() {
	c.enabled.Store(true)
}

func (c *MetricsCollector) Disable() {
	c.enabled.Store(false)
}

func (c *MetricsCollector) Start() {
	c.Enable()

	go c.collect()
}

func (c *MetricsCollector) collect() {
	ticker := time.NewTicker(c.collectionInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !c.enabled.Load() {
			continue
		}

		metric := &ResourceMetric{
			Timestamp:   time.Now(),
			CPUUsage:    c.collectCPUUsage(),
			MemoryUsage: c.collectMemoryUsage(),
			DiskIO:      c.collectDiskIO(),
			NetworkIO:   c.collectNetworkIO(),
			RequestRate: c.collectRequestRate(),
			Latency:     c.collectLatency(),
			ErrorRate:   c.collectErrorRate(),
		}

		c.mu.Lock()
		c.metrics[uuid.New().String()] = metric
		c.mu.Unlock()
	}
}

func (c *MetricsCollector) collectCPUUsage() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Sys) / float64(1<<30)
}

func (c *MetricsCollector) collectMemoryUsage() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) / float64(1<<30)
}

func (c *MetricsCollector) collectDiskIO() float64 {
	return 0
}

func (c *MetricsCollector) collectNetworkIO() float64 {
	return 0
}

func (c *MetricsCollector) collectRequestRate() float64 {
	return 1000
}

func (c *MetricsCollector) collectLatency() float64 {
	return 50
}

func (c *MetricsCollector) collectErrorRate() float64 {
	return 0.01
}

func (s *ResourceScheduler) Scale(ctx context.Context) (*ScaleRecommendation, error) {
	if s.k8sAutoscaler == nil {
		return &ScaleRecommendation{
			Action: ScaleActionNoChange,
			Reason: "autoscaler not enabled",
		}, nil
	}

	metric := &ResourceMetric{
		Timestamp:   time.Now(),
		CPUUsage:    s.collectCurrentCPU(),
		MemoryUsage: s.collectCurrentMemory(),
		RequestRate: float64(s.metricsCollector.getCurrentQPS()),
		Latency:     s.collectCurrentLatency(),
	}

	recommendation, err := s.k8sAutoscaler.Scale(ctx, metric)
	if err != nil {
		return nil, err
	}

	if recommendation.Action != ScaleActionNoChange {
		s.replicas.Store(int64(recommendation.NewReplicas))
		s.metrics.ScaleOperations.Add(1)
		s.metrics.ReplicasChange.Add(1)
	}

	return recommendation, nil
}

func (s *ResourceScheduler) collectCurrentCPU() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Sys) / float64(1<<30)
}

func (s *ResourceScheduler) collectCurrentMemory() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) / float64(1<<30)
}

func (s *ResourceScheduler) collectCurrentLatency() float64 {
	return 50
}

func (c *MetricsCollector) getCurrentQPS() int64 {
	return 1000
}

func (s *ResourceScheduler) GetK8sAutoscaler() *K8sAutoscaler {
	return s.k8sAutoscaler
}

func (s *ResourceScheduler) GetServiceMesh() *ServiceMeshManager {
	return s.serviceMesh
}

func (s *ResourceScheduler) GetPredictor() *ResourcePredictor {
	return s.predictor
}

func (s *ResourceScheduler) GetCostOptimizer() *CostPerformanceOptimizer {
	return s.costOptimizer
}

func (s *ResourceScheduler) GetMetrics() *SchedulerMetrics {
	return &SchedulerMetrics{
		TotalSchedules:      s.metrics.TotalSchedules,
		SuccessfulSchedules: s.metrics.SuccessfulSchedules,
		FailedSchedules:     s.metrics.FailedSchedules,
		ScaleOperations:     s.metrics.ScaleOperations,
		ReplicasChange:      s.metrics.ReplicasChange,
		ResourceEfficiency:  s.metrics.ResourceEfficiency,
		CostEfficiency:     s.metrics.CostEfficiency,
		AvgQueueTime:        s.metrics.AvgQueueTime,
		AvgWaitTime:         s.metrics.AvgWaitTime,
	}
}

func (s *ResourceScheduler) GetReport() map[string]interface{} {
	report := map[string]interface{}{
		"uptime_seconds":  time.Since(s.startTime).Seconds(),
		"current_replicas": s.replicas.Load(),
		"target_replicas":  s.targetReplicas.Load(),
	}

	if s.k8sAutoscaler != nil {
		k8sMetrics := s.k8sAutoscaler.GetMetrics()
		report["k8s_autoscaler"] = map[string]interface{}{
			"scale_events":     k8sMetrics.ScaleEvents.Load(),
			"scale_up_events":  k8sMetrics.ScaleUpEvents.Load(),
			"scale_down_events": k8sMetrics.ScaleDownEvents.Load(),
			"current_cpu":      k8sMetrics.CurrentCPUUsage.Load(),
			"current_memory":   k8sMetrics.CurrentMemUsage.Load(),
		}
	}

	if s.serviceMesh != nil {
		meshMetrics := s.serviceMesh.GetMetrics()
		report["service_mesh"] = map[string]interface{}{
			"total_requests":    meshMetrics.TotalRequests.Load(),
			"failed_requests":   meshMetrics.FailedRequests.Load(),
			"circuit_breaks":    meshMetrics.CircuitBreakerTrips.Load(),
		}
	}

	if s.predictor != nil {
		predMetrics := s.predictor.GetMetrics()
		report["prediction"] = map[string]interface{}{
			"predictions_made": predMetrics.PredictionsMade.Load(),
		}
	}

	if s.costOptimizer != nil {
		costMetrics := s.costOptimizer.GetMetrics()
		report["cost_optimization"] = map[string]interface{}{
			"total_cost":            costMetrics.TotalCost.Load(),
			"projected_monthly":     costMetrics.ProjectedMonthlyCost.Load(),
			"cost_per_request":      costMetrics.CostPerRequest.Load(),
		}
	}

	return report
}

func (s *ResourceScheduler) SetReplicas(replicas int) {
	if s.k8sAutoscaler != nil {
		s.k8sAutoscaler.SetReplicas(replicas)
	}
	s.replicas.Store(int64(replicas))
}

func (s *ResourceScheduler) GetReplicas() int {
	return int(s.replicas.Load())
}

func (s *ResourceScheduler) Close() error {
	s.initialized.Store(false)

	if s.metricsCollector != nil {
		s.metricsCollector.Disable()
	}

	if s.k8sAutoscaler != nil {
		s.k8sAutoscaler.Disable()
	}

	if s.serviceMesh != nil {
		s.serviceMesh.Disable()
	}

	return nil
}
