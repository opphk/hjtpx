package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type K8sAutoscaler struct {
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	isRunning        bool
	config          *AutoscalerConfig
	metricsCollector *K8sMetricsCollector
	predictor       *ResourcePredictor
	policyEngine    *ScalingPolicyEngine
	stats           *AutoscalerStats
}

type AutoscalerConfig struct {
	MinReplicas          int
	MaxReplicas          int
	TargetCPUUtilization float64
	TargetMemoryUtilization float64
	ScaleUpStabilization time.Duration
	ScaleDownStabilization time.Duration
	ScaleUpThreshold     float64
	ScaleDownThreshold   float64
	CooldownPeriod       time.Duration
	MetricsWindow        time.Duration
	EnablePrediction     bool
	PredictionWindow     time.Duration
}

type AutoscalerStats struct {
	TotalScaleEvents    atomic.Int64
	ScaleUpEvents       atomic.Int64
	ScaleDownEvents     atomic.Int64
	CurrentReplicas     atomic.Int64
	TargetReplicas      atomic.Int64
	LastScaleTime       atomic.Value
	LastScaleDirection  atomic.Value
	PredictedReplicas   atomic.Int64
	ActualReplicas      atomic.Int64
	ErrorRate           atomic.Float64
	StabilizationWindow time.Duration
}

type K8sMetricsCollector struct {
	mu           sync.RWMutex
	cpuMetrics   []MetricPoint
	memoryMetrics []MetricPoint
	requestMetrics []MetricPoint
	windowSize   time.Duration
}

type MetricPoint struct {
	Timestamp time.Time
	Value     float64
}

type ResourcePredictor struct {
	mu           sync.RWMutex
	model        PredictorModel
	windowSize   time.Duration
	confidence   float64
}

type PredictorModel interface {
	Predict(history []MetricPoint) (float64, float64)
	Train(data []MetricPoint)
}

type ScalingPolicyEngine struct {
	mu       sync.RWMutex
	policies []ScalingPolicy
}

type ScalingPolicy struct {
	Name       string
	Priority   int
	Condition  PolicyCondition
	Action     ScalingAction
	Enabled    bool
}

type PolicyCondition struct {
	Metric      string
	Operator    string
	Threshold  float64
	Duration    time.Duration
}

type ScalingAction struct {
	Type      string
	Replicas  int
	Percent   float64
}

type HPAClient struct {
	mu            sync.RWMutex
	client        interface{}
	namespace     string
}

func NewK8sAutoscaler(config *AutoscalerConfig) *K8sAutoscaler {
	if config == nil {
		config = DefaultAutoscalerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &K8sAutoscaler{
		ctx:             ctx,
		cancel:          cancel,
		config:          config,
		metricsCollector: NewK8sMetricsCollector(config.MetricsWindow),
		predictor:       NewResourcePredictor(config.PredictionWindow),
		policyEngine:    NewScalingPolicyEngine(),
		stats:           &AutoscalerStats{},
	}
}

func DefaultAutoscalerConfig() *AutoscalerConfig {
	return &AutoscalerConfig{
		MinReplicas:           1,
		MaxReplicas:           100,
		TargetCPUUtilization:  70,
		TargetMemoryUtilization: 80,
		ScaleUpStabilization:  3 * time.Minute,
		ScaleDownStabilization: 5 * time.Minute,
		ScaleUpThreshold:      80,
		ScaleDownThreshold:    50,
		CooldownPeriod:        5 * time.Minute,
		MetricsWindow:         5 * time.Minute,
		EnablePrediction:      true,
		PredictionWindow:      10 * time.Minute,
	}
}

func (k *K8sAutoscaler) Start() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.isRunning {
		return nil
	}

	k.isRunning = true

	go k.collectMetrics()
	go k.evaluateScaling()
	go k.applyScaling()

	log.Println("[K8sAutoscaler] Started successfully")
	return nil
}

func (k *K8sAutoscaler) Stop() {
	k.mu.Lock()
	defer k.mu.Unlock()

	if !k.isRunning {
		return
	}

	k.cancel()
	k.isRunning = false
	log.Println("[K8sAutoscaler] Stopped")
}

func (k *K8sAutoscaler) collectMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-k.ctx.Done():
			return
		case <-ticker.C:
			k.gatherMetrics()
		}
	}
}

func (k *K8sAutoscaler) gatherMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	cpuUsage := k.estimateCPUUsage()
	memoryUsage := float64(memStats.Alloc) / float64(memStats.Sys) * 100

	k.metricsCollector.Record("cpu", cpuUsage)
	k.metricsCollector.Record("memory", memoryUsage)

	log.Printf("[K8sAutoscaler] Metrics: CPU=%.2f%%, Memory=%.2f%%", cpuUsage, memoryUsage)
}

func (k *K8sAutoscaler) estimateCPUUsage() float64 {
	var cpuStats struct {
		lastCPU    int64
		lastSys    int64
		lastTime   time.Time
	}
	
	now := time.Now()
	elapsed := now.Sub(cpuStats.lastTime).Nanoseconds()
	if elapsed > 0 {
		cpuFraction := float64(cpuStats.lastCPU) / float64(elapsed)
		return cpuFraction * 100
	}
	
	return 50.0
}

func (k *K8sAutoscaler) evaluateScaling() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-k.ctx.Done():
			return
		case <-ticker.C:
			k.evaluateAndScale()
		}
	}
}

func (k *K8sAutoscaler) evaluateAndScale() {
	currentMetrics := k.metricsCollector.GetAverages()
	currentReplicas := k.stats.CurrentReplicas.Load()

	var targetReplicas int
	var direction string

	if k.config.EnablePrediction {
		predicted := k.predictor.Predict(k.metricsCollector.cpuMetrics)
		k.stats.PredictedReplicas.Store(int64(predicted))
	}

	cpuMetric := currentMetrics["cpu"]
	memoryMetric := currentMetrics["memory"]

	if cpuMetric > k.config.ScaleUpThreshold || memoryMetric > k.config.ScaleUpThreshold {
		targetReplicas = k.calculateScaleUp(currentReplicas, cpuMetric)
		direction = "up"
	} else if cpuMetric < k.config.ScaleDownThreshold && memoryMetric < k.config.ScaleDownThreshold {
		targetReplicas = k.calculateScaleDown(currentReplicas, cpuMetric)
		direction = "down"
	} else {
		targetReplicas = int(currentReplicas)
		direction = "stable"
	}

	if targetReplicas != int(currentReplicas) {
		if k.checkStabilization(direction) {
			k.stats.TargetReplicas.Store(int64(targetReplicas))
			k.stats.ScaleUpEvents.Add(1)
			log.Printf("[K8sAutoscaler] Scaling %s from %d to %d replicas", direction, currentReplicas, targetReplicas)
		}
	}
}

func (k *K8sAutoscaler) calculateScaleUp(current int64, metric float64) int {
	scaleFactor := metric / k.config.TargetCPUUtilization
	target := int(float64(current) * scaleFactor)

	if target < k.config.MinReplicas {
		target = k.config.MinReplicas
	}
	if target > k.config.MaxReplicas {
		target = k.config.MaxReplicas
	}

	return target
}

func (k *K8sAutoscaler) calculateScaleDown(current int64, metric float64) int {
	scaleFactor := k.config.TargetCPUUtilization / metric
	target := int(float64(current) * scaleFactor)

	if target < k.config.MinReplicas {
		target = k.config.MinReplicas
	}
	if target > k.config.MaxReplicas {
		target = k.config.MaxReplicas
	}

	return target
}

func (k *K8sAutoscaler) checkStabilization(direction string) bool {
	lastDirection := k.stats.LastScaleDirection.Load()
	if lastDirection != direction {
		k.stats.LastScaleTime.Store(time.Now())
		k.stats.LastScaleDirection.Store(direction)
		return false
	}

	lastTime := k.stats.LastScaleTime.Load().(time.Time)
	var requiredTime time.Duration
	if direction == "up" {
		requiredTime = k.config.ScaleUpStabilization
	} else {
		requiredTime = k.config.ScaleDownStabilization
	}

	if time.Since(lastTime) >= requiredTime {
		return true
	}

	return false
}

func (k *K8sAutoscaler) applyScaling() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-k.ctx.Done():
			return
		case <-ticker.C:
			targetReplicas := k.stats.TargetReplicas.Load()
			if targetReplicas > 0 {
				k.applyToCluster(int(targetReplicas))
			}
		}
	}
}

func (k *K8sAutoscaler) applyToCluster(replicas int) {
	k.stats.CurrentReplicas.Store(int64(replicas))
	k.stats.TotalScaleEvents.Add(1)
	k.stats.LastScaleTime.Store(time.Now())
}

func (k *K8sAutoscaler) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"current_replicas":    k.stats.CurrentReplicas.Load(),
		"target_replicas":     k.stats.TargetReplicas.Load(),
		"predicted_replicas":  k.stats.PredictedReplicas.Load(),
		"total_scale_events":  k.stats.TotalScaleEvents.Load(),
		"scale_up_events":     k.stats.ScaleUpEvents.Load(),
		"scale_down_events":   k.stats.ScaleDownEvents.Load(),
		"error_rate":          k.stats.ErrorRate.Load(),
		"last_scale_time":     k.stats.LastScaleTime.Load(),
		"last_scale_direction": k.stats.LastScaleDirection.Load(),
	}
}

func NewK8sMetricsCollector(windowSize time.Duration) *K8sMetricsCollector {
	return &K8sMetricsCollector{
		cpuMetrics:    make([]MetricPoint, 0),
		memoryMetrics: make([]MetricPoint, 0),
		requestMetrics: make([]MetricPoint, 0),
		windowSize:   windowSize,
	}
}

func (m *K8sMetricsCollector) Record(metricType string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	point := MetricPoint{
		Timestamp: time.Now(),
		Value:     value,
	}

	switch metricType {
	case "cpu":
		m.cpuMetrics = append(m.cpuMetrics, point)
	case "memory":
		m.memoryMetrics = append(m.memoryMetrics, point)
	case "request":
		m.requestMetrics = append(m.requestMetrics, point)
	}

	m.pruneOldMetrics()
}

func (m *K8sMetricsCollector) pruneOldMetrics() {
	cutoff := time.Now().Add(-m.windowSize)

	m.cpuMetrics = pruneMetricPoints(m.cpuMetrics, cutoff)
	m.memoryMetrics = pruneMetricPoints(m.memoryMetrics, cutoff)
	m.requestMetrics = pruneMetricPoints(m.requestMetrics, cutoff)
}

func pruneMetricPoints(points []MetricPoint, cutoff time.Time) []MetricPoint {
	var pruned []MetricPoint
	for _, p := range points {
		if p.Timestamp.After(cutoff) {
			pruned = append(pruned, p)
		}
	}
	return pruned
}

func (m *K8sMetricsCollector) GetAverages() map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgCPU := averageMetricPoints(m.cpuMetrics)
	avgMemory := averageMetricPoints(m.memoryMetrics)
	avgRequest := averageMetricPoints(m.requestMetrics)

	return map[string]float64{
		"cpu":     avgCPU,
		"memory":  avgMemory,
		"request": avgRequest,
	}
}

func averageMetricPoints(points []MetricPoint) float64 {
	if len(points) == 0 {
		return 0
	}

	sum := 0.0
	for _, p := range points {
		sum += p.Value
	}
	return sum / float64(len(points))
}

func NewResourcePredictor(windowSize time.Duration) *ResourcePredictor {
	return &ResourcePredictor{
		model:      &LinearRegressionPredictor{},
		windowSize: windowSize,
		confidence: 0.95,
	}
}

func (p *ResourcePredictor) Predict(history []MetricPoint) float64 {
	if len(history) < 3 {
		return float64(len(history) + 1)
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	prediction, _ := p.model.Predict(history)
	return prediction
}

type LinearRegressionPredictor struct{}

func (l *LinearRegressionPredictor) Predict(history []MetricPoint) (float64, float64) {
	if len(history) < 2 {
		return 1, 0
	}

	n := float64(len(history))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i, p := range history {
		x := float64(i)
		y := p.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return sumY / n, 0
	}

	slope := (n*sumXY - sumX*sumY) / denom
	intercept := (sumY - slope*sumX) / n

	nextX := float64(len(history))
	prediction := slope*nextX + intercept

	if prediction < 1 {
		prediction = 1
	}

	confidence := 1.0 - math.Min(1.0, math.Abs(slope)/100)

	return prediction, confidence
}

func (l *LinearRegressionPredictor) Train(data []MetricPoint) {}

func NewScalingPolicyEngine() *ScalingPolicyEngine {
	engine := &ScalingPolicyEngine{
		policies: make([]ScalingPolicy, 0),
	}

	engine.initializeDefaultPolicies()
	return engine
}

func (e *ScalingPolicyEngine) initializeDefaultPolicies() {
	e.policies = append(e.policies,
		ScalingPolicy{
			Name:    "high-cpu-scale-up",
			Priority: 1,
			Condition: PolicyCondition{
				Metric:   "cpu",
				Operator: ">",
				Threshold: 80,
			},
			Action: ScalingAction{
				Type:     "increase",
				Percent: 50,
			},
			Enabled: true,
		},
		ScalingPolicy{
			Name:    "low-cpu-scale-down",
			Priority: 2,
			Condition: PolicyCondition{
				Metric:   "cpu",
				Operator: "<",
				Threshold: 30,
			},
			Action: ScalingAction{
				Type:    "decrease",
				Percent: 30,
			},
			Enabled: true,
		},
	)
}

func (e *ScalingPolicyEngine) Evaluate(metrics map[string]float64) (int, string) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, policy := range e.policies {
		if !policy.Enabled {
			continue
		}

		value, ok := metrics[policy.Condition.Metric]
		if !ok {
			continue
		}

		triggered := false
		switch policy.Condition.Operator {
		case ">":
			triggered = value > policy.Condition.Threshold
		case "<":
			triggered = value < policy.Condition.Threshold
		case ">=":
			triggered = value >= policy.Condition.Threshold
		case "<=":
			triggered = value <= policy.Condition.Threshold
		case "==":
			triggered = value == policy.Condition.Threshold
		}

		if triggered {
			return int(policy.Action.Percent), policy.Action.Type
		}
	}

	return 0, "none"
}

func (e *ScalingPolicyEngine) AddPolicy(policy ScalingPolicy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policies = append(e.policies, policy)
}

func (e *ScalingPolicyEngine) RemovePolicy(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, p := range e.policies {
		if p.Name == name {
			e.policies = append(e.policies[:i], e.policies[i+1:]...)
			break
		}
	}
}

type HPAAdapter struct {
	mu         sync.RWMutex
	client     interface{}
	namespace  string
}

func NewHPAAdapter(namespace string) *HPAAdapter {
	return &HPAAdapter{
		namespace: namespace,
	}
}

func (h *HPAAdapter) UpdateReplicas(name string, replicas int) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	log.Printf("[HPAAdapter] Updating %s/%s replicas to %d", h.namespace, name, replicas)
	return nil
}

func (h *HPAAdapter) GetReplicas(name string) (int, error) {
	return 1, nil
}

var _ = runtime.NumCPU()
var _ = math.Min
