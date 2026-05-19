package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type ScalingMetric string

const (
	MetricCPUUtilization   ScalingMetric = "cpu_utilization"
	MetricMemoryUsage      ScalingMetric = "memory_usage"
	MetricRequestCount     ScalingMetric = "request_count"
	MetricRequestLatency   ScalingMetric = "request_latency"
	MetricConcurrency       ScalingMetric = "concurrency"
	MetricQueueDepth       ScalingMetric = "queue_depth"
	MetricCustomMetric     ScalingMetric = "custom_metric"
)

type ScalingPolicyType string

const (
	PolicyTypeTargetTracking ScalingPolicyType = "target_tracking"
	PolicyTypeStepScaling    ScalingPolicyType = "step_scaling"
	PolicyTypeScheduled      ScalingPolicyType = "scheduled"
	PolicyTypePredictive      ScalingPolicyType = "predictive"
)

type ScalingAction struct {
	Type         string  `json:"type"`
	InstanceChange int   `json:"instance_change"`
	MinAdjustment int    `json:"min_adjustment"`
	MaxAdjustment int    `json:"max_adjustment"`
}

type ScalingPolicy struct {
	PolicyName       string                 `json:"policy_name"`
	PolicyType       ScalingPolicyType       `json:"policy_type"`
	FunctionName     string                 `json:"function_name"`
	Metric           ScalingMetric           `json:"metric"`
	TargetValue      float64                `json:"target_value"`
	MinAdjustment    int                    `json:"min_adjustment"`
	MaxAdjustment    int                    `json:"max_adjustment"`
	Cooldown         time.Duration           `json:"cooldown"`
	Warmup           time.Duration           `json:"warmup"`
	StepAdjustments  []StepAdjustment       `json:"step_adjustments,omitempty"`
	ScheduledConfig  *ScheduledScalingConfig `json:"scheduled_config,omitempty"`
	Enabled          bool                   `json:"enabled"`
}

type StepAdjustment struct {
	LowerBound     float64 `json:"lower_bound"`
	UpperBound     float64 `json:"upper_bound"`
	Adjustment     int     `json:"adjustment"`
	AdjustmentType string  `json:"adjustment_type"`
}

type ScheduledScalingConfig struct {
	Schedule       string          `json:"schedule"`
	MinCapacity    int             `json:"min_capacity"`
	MaxCapacity    int             `json:"max_capacity"`
	TargetCapacity int             `json:"target_capacity"`
}

type AutoScaler struct {
	manager       *ServerlessManager
	policies      map[string]*ScalingPolicy
	scalers       map[string]*FunctionScaler
	metrics       *scalingMetrics
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	enabled       atomic.Bool
}

type FunctionScaler struct {
	functionName    string
	currentInstances atomic.Int32
	targetInstances  atomic.Int32
	metrics          []MetricDataPoint
	mu               sync.RWMutex
	cooldownUntil    time.Time
}

type MetricDataPoint struct {
	Timestamp time.Time
	Value     float64
}

type scalingMetrics struct {
	TotalScaleUp     atomic.Int64
	TotalScaleDown   atomic.Int64
	ScaleUpErrors    atomic.Int64
	ScaleDownErrors  atomic.Int64
	AvgScaleUpTime   atomic.Int64
	AvgScaleDownTime atomic.Int64
}

type ScalingResult struct {
	FunctionName    string        `json:"function_name"`
	Action          string        `json:"action"`
	PreviousInstances int32       `json:"previous_instances"`
	CurrentInstances int32       `json:"current_instances"`
	TargetInstances  int32        `json:"target_instances"`
	Reason          string        `json:"reason"`
	Duration        time.Duration `json:"duration"`
	Metrics         map[string]float64 `json:"metrics"`
}

type TargetTrackingConfig struct {
	TargetValue           float64
	PreferLowerInstance   bool
	ScaleInCooldown       time.Duration
	ScaleOutCooldown      time.Duration
	EvaluationPeriods     int
}

type StepScalingConfig struct {
	MetricExpression    string
	StepAdjustments      []StepAdjustment
	Cooldown             time.Duration
	AdjustmentType       string
}

func NewAutoScaler(manager *ServerlessManager) *AutoScaler {
	ctx, cancel := context.WithCancel(context.Background())
	
	scaler := &AutoScaler{
		manager:   manager,
		policies:  make(map[string]*ScalingPolicy),
		scalers:   make(map[string]*FunctionScaler),
		metrics:   &scalingMetrics{},
		ctx:       ctx,
		cancel:    cancel,
	}
	
	scaler.enabled.Store(true)
	
	return scaler
}

func (s *AutoScaler) CreateScalingPolicy(functionName string, policy *ScalingPolicy) error {
	if functionName == "" {
		return fmt.Errorf("function name is required")
	}
	
	if policy == nil {
		return fmt.Errorf("policy is required")
	}
	
	if policy.PolicyName == "" {
		return fmt.Errorf("policy name is required")
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	policy.FunctionName = functionName
	s.policies[policy.PolicyName] = policy
	
	s.scalers[functionName] = &FunctionScaler{
		functionName: functionName,
		metrics:      make([]MetricDataPoint, 0),
	}
	
	return nil
}

func (s *AutoScaler) GetScalingPolicy(policyName string) (*ScalingPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	policy, exists := s.policies[policyName]
	if !exists {
		return nil, fmt.Errorf("scaling policy %s not found", policyName)
	}
	
	return policy, nil
}

func (s *AutoScaler) ListScalingPolicies(functionName string) []*ScalingPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var policies []*ScalingPolicy
	for _, policy := range s.policies {
		if functionName == "" || policy.FunctionName == functionName {
			policies = append(policies, policy)
		}
	}
	
	return policies
}

func (s *AutoScaler) DeleteScalingPolicy(policyName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.policies[policyName]; !exists {
		return fmt.Errorf("scaling policy %s not found", policyName)
	}
	
	delete(s.policies, policyName)
	
	return nil
}

func (s *AutoScaler) Enable() {
	s.enabled.Store(true)
}

func (s *AutoScaler) Disable() {
	s.enabled.Store(false)
}

func (s *AutoScaler) IsEnabled() bool {
	return s.enabled.Load()
}

func (s *AutoScaler) Scale(ctx context.Context, functionName string) (*ScalingResult, error) {
	if !s.enabled.Load() {
		return nil, fmt.Errorf("auto scaler is disabled")
	}
	
	scaler, exists := s.scalers[functionName]
	if !exists {
		return nil, fmt.Errorf("no scaler configured for function %s", functionName)
	}
	
	metadata, err := s.manager.GetFunction(functionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}
	
	previousInstances := scaler.currentInstances.Load()
	
	metrics := s.collectMetrics(ctx, functionName)
	
	var targetInstances int32
	var reason string
	
	policies := s.getActivePolicies(functionName)
	
	for _, policy := range policies {
		instances, policyReason := s.calculateTargetInstances(ctx, policy, metrics)
		if instances != targetInstances {
			targetInstances = instances
			reason = policyReason
		}
	}
	
	if targetInstances < int32(metadata.MinInstances) {
		targetInstances = int32(metadata.MinInstances)
	}
	
	if targetInstances > int32(metadata.MaxInstances) {
		targetInstances = int32(metadata.MaxInstances)
	}
	
	scaler.targetInstances.Store(targetInstances)
	
	result := &ScalingResult{
		FunctionName:      functionName,
		PreviousInstances: previousInstances,
		CurrentInstances:  previousInstances,
		TargetInstances:   targetInstances,
		Reason:            reason,
		Metrics:           metrics,
	}
	
	if targetInstances != previousInstances {
		start := time.Now()
		
		if targetInstances > previousInstances {
			result.Action = "scale_up"
			if err := s.scaleUp(ctx, functionName, int(targetInstances-previousInstances)); err != nil {
				s.metrics.ScaleUpErrors.Add(1)
				return nil, fmt.Errorf("scale up failed: %w", err)
			}
			s.metrics.TotalScaleUp.Add(1)
		} else {
			result.Action = "scale_down"
			if err := s.scaleDown(ctx, functionName, int(previousInstances-targetInstances)); err != nil {
				s.metrics.ScaleDownErrors.Add(1)
				return nil, fmt.Errorf("scale down failed: %w", err)
			}
			s.metrics.TotalScaleDown.Add(1)
		}
		
		result.Duration = time.Since(start)
		result.CurrentInstances = targetInstances
		scaler.currentInstances.Store( targetInstances)
		
		if result.Action == "scale_up" {
			s.metrics.AvgScaleUpTime.Store(result.Duration.Nanoseconds())
		} else {
			s.metrics.AvgScaleDownTime.Store(result.Duration.Nanoseconds())
		}
	} else {
		result.Action = "no_change"
	}
	
	return result, nil
}

func (s *AutoScaler) calculateTargetInstances(ctx context.Context, policy *ScalingPolicy, metrics map[string]float64) (int32, string) {
	if !policy.Enabled {
		return 0, "policy disabled"
	}
	
	switch policy.PolicyType {
	case PolicyTypeTargetTracking:
		return s.calculateTargetTracking(ctx, policy, metrics)
	case PolicyTypeStepScaling:
		return s.calculateStepScaling(ctx, policy, metrics)
	default:
		return 0, "unknown policy type"
	}
}

func (s *AutoScaler) calculateTargetTracking(ctx context.Context, policy *ScalingPolicy, metrics map[string]float64) (int32, string) {
	metricValue := metrics[string(policy.Metric)]
	
	if metricValue == 0 {
		return 1, "no metric data"
	}
	
	diff := metricValue - policy.TargetValue
	percentageDiff := diff / policy.TargetValue * 100
	
	if math.Abs(percentageDiff) < 5 {
		return 0, "within tolerance"
	}
	
	adjustment := int(math.Ceil(percentageDiff / 10))
	if adjustment == 0 {
		if percentageDiff > 0 {
			adjustment = 1
		} else {
			adjustment = -1
		}
	}
	
	if adjustment > policy.MaxAdjustment {
		adjustment = policy.MaxAdjustment
	}
	if adjustment < -policy.MaxAdjustment {
		adjustment = -policy.MaxAdjustment
	}
	
	if adjustment > 0 {
		return int32(adjustment), fmt.Sprintf("scale up: %.2f%% above target", percentageDiff)
	}
	if adjustment < 0 {
		return int32(adjustment), fmt.Sprintf("scale down: %.2f%% below target", -percentageDiff)
	}
	
	return 0, "no adjustment needed"
}

func (s *AutoScaler) calculateStepScaling(ctx context.Context, policy *ScalingPolicy, metrics map[string]float64) (int32, string) {
	metricValue := metrics[string(policy.Metric)]
	
	for _, step := range policy.StepAdjustments {
		if metricValue >= step.LowerBound && (step.UpperBound < 0 || metricValue < step.UpperBound) {
			return int32(step.Adjustment), fmt.Sprintf("step adjustment: %d", step.Adjustment)
		}
	}
	
	return 0, "no matching step"
}

func (s *AutoScaler) scaleUp(ctx context.Context, functionName string, count int) error {
	scaler, exists := s.scalers[functionName]
	if !exists {
		return fmt.Errorf("scaler not found")
	}
	
	scaler.currentInstances.Add( int32(count))
	
	s.manager.SetFunctionState(functionName, FunctionStateScaling)
	
	return nil
}

func (s *AutoScaler) scaleDown(ctx context.Context, functionName string, count int) error {
	scaler, exists := s.scalers[functionName]
	if !exists {
		return fmt.Errorf("scaler not found")
	}
	
	current := scaler.currentInstances.Load()
	newValue := current - int32(count)
	if newValue < 0 {
		newValue = 0
	}
	scaler.currentInstances.Store( newValue)
	
	return nil
}

func (s *AutoScaler) collectMetrics(ctx context.Context, functionName string) map[string]float64 {
	metrics := map[string]float64{
		string(MetricCPUUtilization): 0.5,
		string(MetricMemoryUsage):    0.4,
		string(MetricRequestCount):   100.0,
		string(MetricRequestLatency): 50.0,
		string(MetricConcurrency):     10.0,
		string(MetricQueueDepth):     0.0,
	}
	
	scaler, exists := s.scalers[functionName]
	if !exists {
		return metrics
	}
	
	scaler.mu.Lock()
	defer scaler.mu.Unlock()
	
	if len(scaler.metrics) > 100 {
		scaler.metrics = scaler.metrics[len(scaler.metrics)-100:]
	}
	
	return metrics
}

func (s *AutoScaler) getActivePolicies(functionName string) []*ScalingPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var activePolicies []*ScalingPolicy
	for _, policy := range s.policies {
		if policy.FunctionName == functionName && policy.Enabled {
			activePolicies = append(activePolicies, policy)
		}
	}
	
	return activePolicies
}

func (s *AutoScaler) RecordMetric(functionName string, metric ScalingMetric, value float64) {
	scaler, exists := s.scalers[functionName]
	if !exists {
		return
	}
	
	scaler.mu.Lock()
	defer scaler.mu.Unlock()
	
	scaler.metrics = append(scaler.metrics, MetricDataPoint{
		Timestamp: time.Now(),
		Value:     value,
	})
}

func (s *AutoScaler) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_scale_up":       s.metrics.TotalScaleUp.Load(),
		"total_scale_down":     s.metrics.TotalScaleDown.Load(),
		"scale_up_errors":      s.metrics.ScaleUpErrors.Load(),
		"scale_down_errors":    s.metrics.ScaleDownErrors.Load(),
		"avg_scale_up_ms":      float64(s.metrics.AvgScaleUpTime.Load()) / 1e6,
		"avg_scale_down_ms":    float64(s.metrics.AvgScaleDownTime.Load()) / 1e6,
		"active_policies":      len(s.policies),
		"enabled":              s.enabled.Load(),
	}
}

func (s *AutoScaler) GetCurrentInstances(functionName string) (int32, error) {
	scaler, exists := s.scalers[functionName]
	if !exists {
		return 0, fmt.Errorf("no scaler configured for function %s", functionName)
	}
	
	return scaler.currentInstances.Load(), nil
}

func (s *AutoScaler) SetCurrentInstances(functionName string, instances int32) error {
	scaler, exists := s.scalers[functionName]
	if !exists {
		return fmt.Errorf("no scaler configured for function %s", functionName)
	}
	
	scaler.currentInstances.Store( instances)
	
	return nil
}

func (s *AutoScaler) GetTargetInstances(functionName string) (int32, error) {
	scaler, exists := s.scalers[functionName]
	if !exists {
		return 0, fmt.Errorf("no scaler configured for function %s", functionName)
	}
	
	return scaler.targetInstances.Load(), nil
}

func (s *AutoScaler) Stop() {
	s.enabled.Store(false)
	s.cancel()
}

func CreateTargetTrackingPolicy(functionName, policyName string, metric ScalingMetric, targetValue float64) *ScalingPolicy {
	return &ScalingPolicy{
		PolicyName:    policyName,
		PolicyType:    PolicyTypeTargetTracking,
		FunctionName:  functionName,
		Metric:        metric,
		TargetValue:   targetValue,
		MinAdjustment: 1,
		MaxAdjustment: 10,
		Cooldown:     60 * time.Second,
		Warmup:       30 * time.Second,
		Enabled:       true,
	}
}

func CreateStepScalingPolicy(functionName, policyName string, metric ScalingMetric, adjustments []StepAdjustment) *ScalingPolicy {
	return &ScalingPolicy{
		PolicyName:      policyName,
		PolicyType:      PolicyTypeStepScaling,
		FunctionName:    functionName,
		Metric:          metric,
		StepAdjustments: adjustments,
		Cooldown:        60 * time.Second,
		Enabled:         true,
	}
}

func CreateScheduledScalingPolicy(functionName, policyName, schedule string, minCapacity, maxCapacity int) *ScalingPolicy {
	return &ScalingPolicy{
		PolicyName: policyName,
		PolicyType: PolicyTypeScheduled,
		FunctionName: functionName,
		ScheduledConfig: &ScheduledScalingConfig{
			Schedule:    schedule,
			MinCapacity: minCapacity,
			MaxCapacity: maxCapacity,
		},
		Enabled: true,
	}
}

func (s *AutoScaler) ApplyScheduledScaling() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, policy := range s.policies {
		if policy.PolicyType != PolicyTypeScheduled || !policy.Enabled {
			continue
		}
		
		if policy.ScheduledConfig == nil {
			continue
		}
		
		config := policy.ScheduledConfig
		
		if config.MinCapacity > 0 || config.MaxCapacity > 0 {
			scaler, exists := s.scalers[policy.FunctionName]
			if !exists {
				continue
			}
			
			scaler.targetInstances.Store( int32(config.TargetCapacity))
		}
	}
	
	return nil
}

func (s *AutoScaler) PredictScaling(ctx context.Context, functionName string, lookAhead time.Duration) (int32, error) {
	scaler, exists := s.scalers[functionName]
	if !exists {
		return 0, fmt.Errorf("no scaler configured for function %s", functionName)
	}
	
	scaler.mu.RLock()
	defer scaler.mu.RUnlock()
	
	if len(scaler.metrics) < 10 {
		return scaler.currentInstances.Load(), nil
	}
	
	recentMetrics := scaler.metrics[len(scaler.metrics)-10:]
	
	var sum float64
	for _, m := range recentMetrics {
		sum += m.Value
	}
	avg := sum / float64(len(recentMetrics))
	
	trend := calculateTrend(recentMetrics)
	
	predictedValue := avg + trend*float64(lookAhead/time.Minute)
	
	policy, err := s.GetScalingPolicy(fmt.Sprintf("%s-target", functionName))
	if err != nil {
		return scaler.currentInstances.Load(), nil
	}
	
	instances := int32(math.Ceil(predictedValue / policy.TargetValue))
	
	return instances, nil
}

func calculateTrend(metrics []MetricDataPoint) float64 {
	if len(metrics) < 2 {
		return 0
	}
	
	n := len(metrics)
	sumY := 0.0
	for i, m := range metrics {
		sumY += float64(i) * m.Value
	}
	
	avgX := float64(n-1) / 2
	avgY := sumY / float64(n)
	
	var numerator, denominator float64
	for i, m := range metrics {
		xDiff := float64(i) - avgX
		yDiff := m.Value - avgY
		numerator += xDiff * yDiff
		denominator += xDiff * xDiff
	}
	
	if denominator == 0 {
		return 0
	}
	
	return numerator / denominator
}

func (s *AutoScaler) SetCooldown(functionName string, cooldown time.Duration) error {
	scaler, exists := s.scalers[functionName]
	if !exists {
		return fmt.Errorf("no scaler configured for function %s", functionName)
	}
	
	scaler.mu.Lock()
	defer scaler.mu.Unlock()
	
	scaler.cooldownUntil = time.Now().Add(cooldown)
	
	return nil
}

func (s *AutoScaler) IsInCooldown(functionName string) bool {
	scaler, exists := s.scalers[functionName]
	if !exists {
		return false
	}
	
	scaler.mu.RLock()
	defer scaler.mu.RUnlock()
	
	return time.Now().Before(scaler.cooldownUntil)
}

func (s *AutoScaler) GetScalingHistory(functionName string, limit int) []*ScalingResult {
	return []*ScalingResult{}
}
