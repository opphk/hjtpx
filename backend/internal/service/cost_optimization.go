package service

import (
	"encoding/json"
	"sync"
	"time"
)

type ResourceType string

const (
	ResourceTypeCPU     ResourceType = "cpu"
	ResourceTypeMemory  ResourceType = "memory"
	ResourceTypeStorage ResourceType = "storage"
	ResourceTypeNetwork ResourceType = "network"
	ResourceTypeDatabase ResourceType = "database"
)

type ResourceUsage struct {
	Type       ResourceType `json:"type"`
	Used       float64      `json:"used"`
	Unit       string       `json:"unit"`
	CostPerUnit float64     `json:"cost_per_unit"`
}

type ComponentResourceUsage struct {
	ComponentID   string           `json:"component_id"`
	ComponentName string           `json:"component_name"`
	Resources     []ResourceUsage  `json:"resources"`
	TotalCost     float64          `json:"total_cost"`
	Period        string           `json:"period"`
}

type CostAllocation struct {
	TenantID       string  `json:"tenant_id"`
	TenantName     string  `json:"tenant_name"`
	ResourceType   string  `json:"resource_type"`
	AllocatedCost  float64 `json:"allocated_cost"`
	UsagePercent   float64 `json:"usage_percent"`
	SharePercent   float64 `json:"share_percent"`
}

type CostAllocationReport struct {
	Period          string           `json:"period"`
	TotalCost       float64          `json:"total_cost"`
	Allocations     []CostAllocation `json:"allocations"`
	GeneratedAt     time.Time        `json:"generated_at"`
}

type UtilizationMetrics struct {
	ResourceType      ResourceType `json:"resource_type"`
	CurrentUtilization float64     `json:"current_utilization_percent"`
	AverageUtilization float64     `json:"average_utilization_percent"`
	PeakUtilization    float64     `json:"peak_utilization_percent"`
	IdleTimePercent    float64     `json:"idle_time_percent"`
	OptimizationPotential float64 `json:"optimization_potential_percent"`
}

type AutoScalingPolicy struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Enabled         bool    `json:"enabled"`
	Metric          string  `json:"metric"`
	MinReplicas     int     `json:"min_replicas"`
	MaxReplicas     int     `json:"max_replicas"`
	ScaleUpThreshold float64 `json:"scale_up_threshold_percent"`
	ScaleDownThreshold float64 `json:"scale_down_threshold_percent"`
	CooldownSeconds int     `json:"cooldown_seconds"`
}

type ScalingRecommendation struct {
	ID            string  `json:"id"`
	ResourceType  string  `json:"resource_type"`
	Action        string  `json:"action"`
	CurrentValue  float64 `json:"current_value"`
	RecommendedValue float64 `json:"recommended_value"`
	Savings       float64 `json:"estimated_savings"`
	Reason        string  `json:"reason"`
	Priority      int     `json:"priority"`
}

type CostOptimizationService struct {
	mu                sync.RWMutex
	resourceUsage     map[string]*ComponentResourceUsage
	costAllocations   []CostAllocation
	utilizationMetrics map[ResourceType]*UtilizationMetrics
	scalingPolicies   map[string]*AutoScalingPolicy
	recommendations   []ScalingRecommendation
	billingCycleStart time.Time
	costHistory       []CostSnapshot
}

type CostSnapshot struct {
	Timestamp  time.Time `json:"timestamp"`
	TotalCost float64   `json:"total_cost"`
	ByResource map[ResourceType]float64 `json:"by_resource"`
}

type ResourceOptimizationAction struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Target      string  `json:"target"`
	Action      string  `json:"action"`
	Status      string  `json:"status"`
	Savings     float64 `json:"estimated_savings"`
	ExecutedAt  time.Time `json:"executed_at"`
}

func NewCostOptimizationService() *CostOptimizationService {
	cos := &CostOptimizationService{
		resourceUsage:     make(map[string]*ComponentResourceUsage),
		costAllocations:   make([]CostAllocation, 0),
		utilizationMetrics: make(map[ResourceType]*UtilizationMetrics),
		scalingPolicies:   make(map[string]*AutoScalingPolicy),
		recommendations:   make([]ScalingRecommendation, 0),
		billingCycleStart: time.Now().AddDate(0, 0, -15),
		costHistory:       make([]CostSnapshot, 0),
	}

	cos.initDefaultMetrics()
	cos.initDefaultPolicies()

	return cos
}

func (cos *CostOptimizationService) initDefaultMetrics() {
	cos.utilizationMetrics[ResourceTypeCPU] = &UtilizationMetrics{
		ResourceType: ResourceTypeCPU,
		CurrentUtilization: 65.0,
		AverageUtilization: 55.0,
		PeakUtilization: 85.0,
		IdleTimePercent: 35.0,
		OptimizationPotential: 20.0,
	}

	cos.utilizationMetrics[ResourceTypeMemory] = &UtilizationMetrics{
		ResourceType: ResourceTypeMemory,
		CurrentUtilization: 72.0,
		AverageUtilization: 68.0,
		PeakUtilization: 88.0,
		IdleTimePercent: 28.0,
		OptimizationPotential: 15.0,
	}

	cos.utilizationMetrics[ResourceTypeStorage] = &UtilizationMetrics{
		ResourceType: ResourceTypeStorage,
		CurrentUtilization: 45.0,
		AverageUtilization: 42.0,
		PeakUtilization: 60.0,
		IdleTimePercent: 55.0,
		OptimizationPotential: 30.0,
	}

	cos.utilizationMetrics[ResourceTypeNetwork] = &UtilizationMetrics{
		ResourceType: ResourceTypeNetwork,
		CurrentUtilization: 38.0,
		AverageUtilization: 35.0,
		PeakUtilization: 70.0,
		IdleTimePercent: 62.0,
		OptimizationPotential: 35.0,
	}

	cos.utilizationMetrics[ResourceTypeDatabase] = &UtilizationMetrics{
		ResourceType: ResourceTypeDatabase,
		CurrentUtilization: 55.0,
		AverageUtilization: 50.0,
		PeakUtilization: 75.0,
		IdleTimePercent: 45.0,
		OptimizationPotential: 25.0,
	}
}

func (cos *CostOptimizationService) initDefaultPolicies() {
	cos.scalingPolicies["cpu-scaling"] = &AutoScalingPolicy{
		ID: "cpu-scaling",
		Name: "CPU Auto Scaling",
		Enabled: true,
		Metric: "cpu_utilization",
		MinReplicas: 2,
		MaxReplicas: 10,
		ScaleUpThreshold: 75.0,
		ScaleDownThreshold: 25.0,
		CooldownSeconds: 300,
	}

	cos.scalingPolicies["memory-scaling"] = &AutoScalingPolicy{
		ID: "memory-scaling",
		Name: "Memory Auto Scaling",
		Enabled: true,
		Metric: "memory_utilization",
		MinReplicas: 2,
		MaxReplicas: 8,
		ScaleUpThreshold: 80.0,
		ScaleDownThreshold: 30.0,
		CooldownSeconds: 300,
	}
}

func (cos *CostOptimizationService) RecordResourceUsage(usage *ComponentResourceUsage) {
	cos.mu.Lock()
	defer cos.mu.Unlock()

	usage.TotalCost = cos.calculateTotalCost(usage.Resources)
	cos.resourceUsage[usage.ComponentID] = usage

	cos.updateUtilizationMetrics(usage)
}

func (cos *CostOptimizationService) calculateTotalCost(resources []ResourceUsage) float64 {
	var total float64
	for _, res := range resources {
		total += res.Used * res.CostPerUnit
	}
	return total
}

func (cos *CostOptimizationService) updateUtilizationMetrics(usage *ComponentResourceUsage) {
	for _, res := range usage.Resources {
		if metrics, exists := cos.utilizationMetrics[res.Type]; exists {
			metrics.CurrentUtilization = res.Used
			if res.Used > metrics.PeakUtilization {
				metrics.PeakUtilization = res.Used
			}
		}
	}
}

func (cos *CostOptimizationService) GetResourceUsage(componentID string) (*ComponentResourceUsage, bool) {
	cos.mu.RLock()
	defer cos.mu.RUnlock()

	usage, exists := cos.resourceUsage[componentID]
	return usage, exists
}

func (cos *CostOptimizationService) GetAllResourceUsage() []*ComponentResourceUsage {
	cos.mu.RLock()
	defer cos.mu.RUnlock()

	usages := make([]*ComponentResourceUsage, 0, len(cos.resourceUsage))
	for _, usage := range cos.resourceUsage {
		usages = append(usages, usage)
	}

	return usages
}

func (cos *CostOptimizationService) GetTotalCost() float64 {
	cos.mu.RLock()
	defer cos.mu.RUnlock()

	var total float64
	for _, usage := range cos.resourceUsage {
		total += usage.TotalCost
	}

	return total
}

func (cos *CostOptimizationService) GetCostByResourceType() map[ResourceType]float64 {
	cos.mu.RLock()
	defer cos.mu.RUnlock()

	costs := make(map[ResourceType]float64)
	for _, usage := range cos.resourceUsage {
		for _, res := range usage.Resources {
			costs[res.Type] += res.Used * res.CostPerUnit
		}
	}

	return costs
}

func (cos *CostOptimizationService) GetCostByTenant() []CostAllocation {
	cos.mu.RLock()
	defer cos.mu.RUnlock()

	allocations := make([]CostAllocation, len(cos.costAllocations))
	copy(allocations, cos.costAllocations)

	return allocations
}

func (cos *CostOptimizationService) SetCostAllocation(allocation CostAllocation) {
	cos.mu.Lock()
	defer cos.mu.Unlock()

	cos.costAllocations = append(cos.costAllocations, allocation)
}

func (cos *CostOptimizationService) GenerateCostAllocationReport(period string) *CostAllocationReport {
	cos.mu.RLock()
	defer cos.mu.RUnlock()

	var totalCost float64
	for _, usage := range cos.resourceUsage {
		totalCost += usage.TotalCost
	}

	for i := range cos.costAllocations {
		cos.costAllocations[i].AllocatedCost = totalCost * cos.costAllocations[i].SharePercent / 100
	}

	return &CostAllocationReport{
		Period:      period,
		TotalCost:   totalCost,
		Allocations: cos.costAllocations,
		GeneratedAt: time.Now(),
	}
}

func (cos *CostOptimizationService) GetUtilizationMetrics() []UtilizationMetrics {
	cos.mu.RLock()
	defer cos.mu.RUnlock()

	metrics := make([]UtilizationMetrics, 0, len(cos.utilizationMetrics))
	for _, m := range cos.utilizationMetrics {
		metrics = append(metrics, *m)
	}

	return metrics
}

func (cos *CostOptimizationService) GetUtilizationByType(resourceType ResourceType) (*UtilizationMetrics, bool) {
	cos.mu.RLock()
	defer cos.mu.RUnlock()

	metrics, exists := cos.utilizationMetrics[resourceType]
	return metrics, exists
}

func (cos *CostOptimizationService) UpdateUtilization(resourceType ResourceType, utilization float64) {
	cos.mu.Lock()
	defer cos.mu.Unlock()

	if metrics, exists := cos.utilizationMetrics[resourceType]; exists {
		metrics.CurrentUtilization = utilization
		if utilization > metrics.PeakUtilization {
			metrics.PeakUtilization = utilization
		}
	}
}

func (cos *CostOptimizationService) GetOptimizationRecommendations() []ScalingRecommendation {
	cos.mu.RLock()
	defer cos.mu.RUnlock()

	cos.recommendations = make([]ScalingRecommendation, 0)

	for _, metrics := range cos.utilizationMetrics {
		if metrics.IdleTimePercent > 40 {
			cos.recommendations = append(cos.recommendations, ScalingRecommendation{
				ID: generateID(),
				ResourceType: string(metrics.ResourceType),
				Action: "reduce",
				CurrentValue: metrics.CurrentUtilization,
				RecommendedValue: metrics.AverageUtilization,
				Savings: metrics.IdleTimePercent * 0.5,
				Reason: "High idle time detected",
				Priority: 1,
			})
		}

		if metrics.CurrentUtilization < metrics.AverageUtilization*0.7 {
			cos.recommendations = append(cos.recommendations, ScalingRecommendation{
				ID: generateID(),
				ResourceType: string(metrics.ResourceType),
				Action: "optimize",
				CurrentValue: metrics.CurrentUtilization,
				RecommendedValue: metrics.AverageUtilization * 0.8,
				Savings: (metrics.AverageUtilization - metrics.CurrentUtilization) * 0.3,
				Reason: "Current utilization below average",
				Priority: 2,
			})
		}
	}

	return cos.recommendations
}

func (cos *CostOptimizationService) GetAutoScalingPolicies() []*AutoScalingPolicy {
	cos.mu.RLock()
	defer cos.mu.RUnlock()

	policies := make([]*AutoScalingPolicy, 0, len(cos.scalingPolicies))
	for _, policy := range cos.scalingPolicies {
		policies = append(policies, policy)
	}

	return policies
}

func (cos *CostOptimizationService) UpdateAutoScalingPolicy(policy *AutoScalingPolicy) error {
	cos.mu.Lock()
	defer cos.mu.Unlock()

	cos.scalingPolicies[policy.ID] = policy
	return nil
}

func (cos *CostOptimizationService) EvaluateScalingNeeds(metric string, value float64) (bool, int) {
	cos.mu.RLock()
	defer cos.mu.RUnlock()

	for _, policy := range cos.scalingPolicies {
		if policy.Metric == metric && policy.Enabled {
			if value > policy.ScaleUpThreshold {
				return true, policy.MaxReplicas
			} else if value < policy.ScaleDownThreshold {
				return true, policy.MinReplicas
			}
		}
	}

	return false, 0
}

func (cos *CostOptimizationService) GetCostTrend(days int) []CostSnapshot {
	cos.mu.RLock()
	defer cos.mu.RUnlock()

	if days <= 0 || days > len(cos.costHistory) {
		days = len(cos.costHistory)
	}

	snapshots := make([]CostSnapshot, days)
	copy(snapshots, cos.costHistory[len(cos.costHistory)-days:])

	return snapshots
}

func (cos *CostOptimizationService) TakeCostSnapshot() {
	cos.mu.Lock()
	defer cos.mu.Unlock()

	var totalCost float64
	byResource := make(map[ResourceType]float64)

	for _, usage := range cos.resourceUsage {
		totalCost += usage.TotalCost
		for _, res := range usage.Resources {
			byResource[res.Type] += res.Used * res.CostPerUnit
		}
	}

	snapshot := CostSnapshot{
		Timestamp:  time.Now(),
		TotalCost:  totalCost,
		ByResource: byResource,
	}

	cos.costHistory = append(cos.costHistory, snapshot)

	if len(cos.costHistory) > 365 {
		cos.costHistory = cos.costHistory[1:]
	}
}

func (cos *CostOptimizationService) GetCostForecast(days int) map[string]float64 {
	cos.mu.RLock()
	defer cos.mu.RUnlock()

	var avgDailyCost float64
	if len(cos.costHistory) > 0 {
		var total float64
		for _, snapshot := range cos.costHistory {
			total += snapshot.TotalCost
		}
		avgDailyCost = total / float64(len(cos.costHistory))
	}

	forecast := make(map[string]float64)
	forecast["daily"] = avgDailyCost
	forecast["weekly"] = avgDailyCost * 7
	forecast["monthly"] = avgDailyCost * 30
	forecast["yearly"] = avgDailyCost * 365

	return forecast
}

func (cos *CostOptimizationService) GetCostOptimizationReport() map[string]interface{} {
	cos.mu.RLock()
	defer cos.mu.RUnlock()

	totalCost := cos.GetTotalCost()
	costByResource := cos.GetCostByResourceType()
	recommendations := cos.GetOptimizationRecommendations()

	var potentialSavings float64
	for _, rec := range recommendations {
		potentialSavings += rec.Savings
	}

	var avgUtilization float64
	for _, metrics := range cos.utilizationMetrics {
		avgUtilization += metrics.CurrentUtilization
	}
	if len(cos.utilizationMetrics) > 0 {
		avgUtilization /= float64(len(cos.utilizationMetrics))
	}

	return map[string]interface{}{
		"period": map[string]interface{}{
			"start": cos.billingCycleStart,
			"end": time.Now(),
		},
		"total_cost": totalCost,
		"cost_by_resource": costByResource,
		"recommendations_count": len(recommendations),
		"potential_savings": potentialSavings,
		"utilization": map[string]interface{}{
			"average_utilization_percent": avgUtilization,
			"metrics": cos.utilizationMetrics,
		},
		"scaling_policies": len(cos.scalingPolicies),
		"enabled_policies": func() int {
			count := 0
			for _, p := range cos.scalingPolicies {
				if p.Enabled {
					count++
				}
			}
			return count
		}(),
		"forecast": cos.GetCostForecast(30),
	}
}

func (cos *CostOptimizationService) SimulateCostScenario(changes map[string]interface{}) map[string]float64 {
	cos.mu.RLock()
	defer cos.mu.RUnlock()

	currentCost := cos.GetTotalCost()

	scaleFactor := 1.0
	if scale, ok := changes["scale_factor"].(float64); ok {
		scaleFactor = scale
	}

	optimizationFactor := 1.0
	if opt, ok := changes["optimization_factor"].(float64); ok {
		optimizationFactor = opt
	}

	return map[string]float64{
		"current_cost": currentCost,
		"scenario_cost": currentCost * scaleFactor * optimizationFactor,
		"monthly_savings": currentCost * 30 * (1 - optimizationFactor),
		"yearly_savings": currentCost * 365 * (1 - optimizationFactor),
	}
}

func (cos *CostOptimizationService) ExportCostReport(format string) (string, error) {
	report := cos.GetCostOptimizationReport()

	switch format {
	case "json":
		data, err := json.Marshal(report)
		if err != nil {
			return "", err
		}
		return string(data), nil

	case "csv":
		return cos.exportCSV(report), nil

	default:
		return "", ErrInvalidParameter
	}
}

func (cos *CostOptimizationService) exportCSV(report map[string]interface{}) string {
	csv := "Resource Type,Cost\n"

	costByResource := report["cost_by_resource"].(map[ResourceType]float64)
	for resourceType, cost := range costByResource {
		csv += string(resourceType) + "," + string(rune(int(cost))) + "\n"
	}

	return csv
}
