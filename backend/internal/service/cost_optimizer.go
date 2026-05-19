package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type PricingModel string

const (
	PricingOnDemand     PricingModel = "on_demand"
	PricingSpotInstance PricingModel = "spot_instance"
	PricingReserved     PricingModel = "reserved"
	PricingSavingsPlan  PricingModel = "savings_plan"
)

type ServerlessCostAllocation struct {
	FunctionName    string            `json:"function_name"`
	ComputeCost     float64          `json:"compute_cost"`
	RequestCost     float64          `json:"request_cost"`
	NetworkCost     float64          `json:"network_cost"`
	StorageCost     float64          `json:"storage_cost"`
	TotalCost       float64          `json:"total_cost"`
	Invocations     int64            `json:"invocations"`
	CostPerInvoke   float64          `json:"cost_per_invoke"`
	CostPerGBSecond float64          `json:"cost_per_gb_second"`
}

type CostOptimizationConfig struct {
	PricingModel      PricingModel    `json:"pricing_model"`
	ReservedCapacity  int             `json:"reserved_capacity"`
	SavingsPlanHours  int             `json:"savings_plan_hours"`
	MemoryOptimization bool           `json:"memory_optimization"`
	TimeoutOptimization bool          `json:"timeout_optimization"`
	AutoScalingPolicy string         `json:"auto_scaling_policy"`
	BudgetLimit      float64         `json:"budget_limit"`
	AlertThreshold   float64         `json:"alert_threshold"`
}

type CostMetrics struct {
	TotalSpend        float64
	TotalSpendMu      sync.Mutex
	ComputeSpend      float64
	ComputeSpendMu    sync.Mutex
	RequestSpend      float64
	RequestSpendMu    sync.Mutex
	NetworkSpend      float64
	NetworkSpendMu    sync.Mutex
	StorageSpend      float64
	StorageSpendMu    sync.Mutex
	TotalInvocations  atomic.Int64
	TotalGBSeconds    float64
	TotalGBSecondsMu  sync.Mutex
	AvgCostPerInvoke  float64
	AvgCostPerInvokeMu sync.Mutex
	LastUpdated       time.Time
}

type ServerlessBudgetAlert struct {
	AlertID      string    `json:"alert_id"`
	Threshold   float64   `json:"threshold"`
	CurrentCost float64   `json:"current_cost"`
	Percentage  float64   `json:"percentage"`
	TriggeredAt time.Time `json:"triggered_at"`
	Dismissed   bool      `json:"dismissed"`
}

type ServerlessCostReport struct {
	ReportID       string            `json:"report_id"`
	StartDate      time.Time         `json:"start_date"`
	EndDate        time.Time         `json:"end_date"`
	TotalCost      float64           `json:"total_cost"`
	ByFunction     []ServerlessCostAllocation  `json:"by_function"`
	ByDay          []ServerlessDailyCost `json:"by_day"`
	ByResource     []ServerlessResourceCost `json:"by_resource"`
	Recommendations []ServerlessCostRecommendation `json:"recommendations"`
	GeneratedAt    time.Time         `json:"generated_at"`
}

type ServerlessDailyCost struct {
	Date     time.Time `json:"date"`
	Cost     float64  `json:"cost"`
	Previous float64   `json:"previous_period_cost"`
	Change   float64   `json:"change_percent"`
}

type ServerlessResourceCost struct {
	ResourceType string  `json:"resource_type"`
	Cost         float64 `json:"cost"`
	Usage        float64 `json:"usage"`
	UnitCost     float64 `json:"unit_cost"`
}

type ServerlessCostRecommendation struct {
	Category    string  `json:"category"`
	Priority    int     `json:"priority"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	EstimatedSavings float64 `json:"estimated_savings"`
	Action      string  `json:"action"`
}

type CostOptimizer struct {
	manager      *ServerlessManager
	metrics      *CostMetrics
	alerts       map[string]*ServerlessBudgetAlert
	configs      map[string]*CostOptimizationConfig
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	enabled      atomic.Bool
}

type ServerlessPricingTier struct {
	RangeStart    int
	RangeEnd      int
	PricePerGBSecond float64
	PricePerRequest float64
}

type SavingsPlan struct {
	PlanID         string        `json:"plan_id"`
	HoursCommitted int           `json:"hours_committed"`
	HoursUsed      int           `json:"hours_used"`
	HourlyRate     float64       `json:"hourly_rate"`
	StartDate      time.Time     `json:"start_date"`
	EndDate        time.Time     `json:"end_date"`
	Status         string        `json:"status"`
}

type ReservedCapacity struct {
	CapacityID    string        `json:"capacity_id"`
	FunctionName  string        `json:"function_name"`
	InstanceType  string        `json:"instance_type"`
	Quantity      int           `json:"quantity"`
	HoursCommitted int          `json:"hours_committed"`
	HourlyRate    float64       `json:"hourly_rate"`
	StartDate     time.Time     `json:"start_date"`
	EndDate       time.Time     `json:"end_date"`
	Status        string        `json:"status"`
}

const (
	PricePerGBSecond = 0.00001667
	PricePerRequest = 0.0000002
	PricePerGBOutbound = 0.00009
	PricePer1000Requests = 0.20
)

func NewCostOptimizer(manager *ServerlessManager) *CostOptimizer {
	ctx, cancel := context.WithCancel(context.Background())
	
	optimizer := &CostOptimizer{
		manager:   manager,
		metrics:   &CostMetrics{},
		alerts:    make(map[string]*ServerlessBudgetAlert),
		configs:   make(map[string]*CostOptimizationConfig),
		ctx:       ctx,
		cancel:    cancel,
	}
	
	optimizer.enabled.Store(true)
	optimizer.metrics.LastUpdated = time.Now()
	
	return optimizer
}

func (o *CostOptimizer) Configure(functionName string, config *CostOptimizationConfig) error {
	if functionName == "" {
		return fmt.Errorf("function name is required")
	}
	
	if config == nil {
		return fmt.Errorf("config is required")
	}
	
	o.mu.Lock()
	defer o.mu.Unlock()
	
	o.configs[functionName] = config
	
	return nil
}

func (o *CostOptimizer) GetCostAllocation(functionName string) (*ServerlessCostAllocation, error) {
	metadata, err := o.manager.GetFunction(functionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}
	
	config, err := o.getConfig(functionName)
	_ = config
	if err != nil {
		return nil, err
	}
	
	invocations := metadata.InvokeCount.Load()
	avgLatencyMs := float64(metadata.AvgLatency.Load()) / 1e6
	memoryMb := float64(metadata.Memory) / 1024.0
	
	gbSeconds := (avgLatencyMs / 1000.0) * memoryMb * float64(invocations)
	
	computeCost := gbSeconds * PricePerGBSecond
	
	requestCost := float64(invocations) * PricePerRequest
	
	networkCost := 0.0
	storageCost := 0.0
	
	totalCost := computeCost + requestCost + networkCost + storageCost
	
	var costPerInvoke float64
	if invocations > 0 {
		costPerInvoke = totalCost / float64(invocations)
	}
	
	return &ServerlessCostAllocation{
		FunctionName:    functionName,
		ComputeCost:     computeCost,
		RequestCost:     requestCost,
		NetworkCost:     networkCost,
		StorageCost:     storageCost,
		TotalCost:       totalCost,
		Invocations:     invocations,
		CostPerInvoke:   costPerInvoke,
		CostPerGBSecond: PricePerGBSecond,
	}, nil
}

func (o *CostOptimizer) CalculateTotalCost() float64 {
	functions := o.manager.ListFunctions()
	
	var totalCost float64
	for _, fn := range functions {
		allocation, err := o.GetCostAllocation(fn.FunctionName)
		if err == nil {
			totalCost += allocation.TotalCost
		}
	}
	
	return totalCost
}

func (o *CostOptimizer) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_spend":         o.metrics.TotalSpend,
		"compute_spend":       o.metrics.ComputeSpend,
		"request_spend":       o.metrics.RequestSpend,
		"network_spend":       o.metrics.NetworkSpend,
		"storage_spend":       o.metrics.StorageSpend,
		"total_invocations":   o.metrics.TotalInvocations.Load(),
		"total_gb_seconds":    o.metrics.TotalGBSeconds,
		"avg_cost_per_invoke": o.metrics.AvgCostPerInvoke,
		"last_updated":        o.metrics.LastUpdated,
	}
}

func (o *CostOptimizer) GenerateReport(ctx context.Context, startDate, endDate time.Time) (*ServerlessCostReport, error) {
	functions := o.manager.ListFunctions()
	
	report := &ServerlessCostReport{
		ReportID:    fmt.Sprintf("report-%d", time.Now().UnixNano()),
		StartDate:   startDate,
		EndDate:     endDate,
		ByFunction:  []ServerlessCostAllocation{},
		ByDay:       []ServerlessDailyCost{},
		ByResource: []ServerlessResourceCost{},
		GeneratedAt: time.Now(),
	}
	
	for _, fn := range functions {
		allocation, err := o.GetCostAllocation(fn.FunctionName)
		if err != nil {
			continue
		}
		
		report.ByFunction = append(report.ByFunction, *allocation)
		report.TotalCost += allocation.TotalCost
	}
	
	for d := startDate; d.Before(endDate); d = d.AddDate(0, 0, 1) {
		report.ByDay = append(report.ByDay, ServerlessDailyCost{
			Date: d,
			Cost: report.TotalCost / float64(endDate.Sub(startDate).Hours() * 24),
		})
	}
	
	report.ByResource = append(report.ByResource,
		ServerlessResourceCost{
			ResourceType: "compute",
			Cost:         o.metrics.ComputeSpend,
		},
		ServerlessResourceCost{
			ResourceType: "requests",
			Cost:         o.metrics.RequestSpend,
		},
	)
	
	report.Recommendations = o.generateRecommendations()
	
	return report, nil
}

func (o *CostOptimizer) generateRecommendations() []ServerlessCostRecommendation {
	recommendations := []ServerlessCostRecommendation{}
	
	recommendations = append(recommendations, ServerlessCostRecommendation{
		Category:         "compute",
		Priority:         1,
		Title:            "Memory Optimization",
		Description:      "Consider reducing memory allocation for functions with low utilization",
		EstimatedSavings: 15.0,
		Action:           "Review and adjust memory settings",
	})
	
	recommendations = append(recommendations, ServerlessCostRecommendation{
		Category:         "compute",
		Priority:         2,
		Title:            "Reserved Capacity",
		Description:      "Purchase reserved capacity for consistent workloads",
		EstimatedSavings: 40.0,
		Action:           "Review usage patterns and purchase reserved instances",
	})
	
	recommendations = append(recommendations, ServerlessCostRecommendation{
		Category:         "compute",
		Priority:         3,
		Title:            "Savings Plan",
		Description:      "Consider savings plan for predictable workloads",
		EstimatedSavings: 30.0,
		Action:           "Set up savings plan commitment",
	})
	
	return recommendations
}

func (o *CostOptimizer) SetBudgetAlert(functionName string, threshold float64) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	alert := &ServerlessBudgetAlert{
		AlertID:    fmt.Sprintf("alert-%s-%d", functionName, time.Now().UnixNano()),
		Threshold:  threshold,
		CurrentCost: 0,
		Percentage: 0,
		TriggeredAt: time.Time{},
		Dismissed:  false,
	}
	
	o.alerts[alert.AlertID] = alert
	
	return nil
}

func (o *CostOptimizer) CheckBudgetAlerts() []*ServerlessBudgetAlert {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	currentCost := o.CalculateTotalCost()
	
	var triggeredAlerts []*ServerlessBudgetAlert
	
	for _, alert := range o.alerts {
		if alert.Dismissed {
			continue
		}
		
		percentage := (currentCost / alert.Threshold) * 100
		
		if percentage >= 100 && !alert.TriggeredAt.IsZero() == false {
			alert.TriggeredAt = time.Now()
		}
		
		alert.CurrentCost = currentCost
		alert.Percentage = percentage
		
		if alert.TriggeredAt.IsZero() == false {
			triggeredAlerts = append(triggeredAlerts, alert)
		}
	}
	
	return triggeredAlerts
}

func (o *CostOptimizer) DismissAlert(alertID string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	alert, exists := o.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}
	
	alert.Dismissed = true
	
	return nil
}

func (o *CostOptimizer) getConfig(functionName string) (*CostOptimizationConfig, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	config, exists := o.configs[functionName]
	if !exists {
		return &CostOptimizationConfig{
			PricingModel:       PricingOnDemand,
			MemoryOptimization:  true,
			TimeoutOptimization: true,
		}, nil
	}
	
	return config, nil
}

func (o *CostOptimizer) OptimizeMemory(functionName string) (int, error) {
	metadata, err := o.manager.GetFunction(functionName)
	if err != nil {
		return 0, fmt.Errorf("failed to get function: %w", err)
	}
	
	currentMemory := int(metadata.Memory)
	
	optimalMemory := calculateOptimalMemory(currentMemory, metadata.InvokeCount.Load())
	
	if optimalMemory != currentMemory {
		config, _ := o.getConfig(functionName)
		if config.MemoryOptimization {
			newConfig := &FunctionConfig{
				FunctionName: functionName,
				Memory:       MemorySize(optimalMemory),
				Timeout:      metadata.Timeout,
				Runtime:      metadata.Runtime,
				Handler:      metadata.Handler,
			}
			
			if err := o.manager.UpdateFunction(functionName, newConfig); err != nil {
				return currentMemory, err
			}
			
			return optimalMemory, nil
		}
	}
	
	return currentMemory, nil
}

func (o *CostOptimizer) OptimizeTimeout(functionName string) (int, error) {
	metadata, err := o.manager.GetFunction(functionName)
	if err != nil {
		return 0, fmt.Errorf("failed to get function: %w", err)
	}
	
	currentTimeout := int(metadata.Timeout)
	
	avgLatencyMs := float64(metadata.AvgLatency.Load()) / 1e6
	
	optimalTimeout := int(avgLatencyMs * 1.5)
	if optimalTimeout < 3 {
		optimalTimeout = 3
	}
	if optimalTimeout > 600 {
		optimalTimeout = 600
	}
	
	if optimalTimeout < currentTimeout {
		config, _ := o.getConfig(functionName)
		if config.TimeoutOptimization {
			newConfig := &FunctionConfig{
				FunctionName: functionName,
				Memory:       metadata.Memory,
				Timeout:      TimeoutDuration(optimalTimeout),
				Runtime:      metadata.Runtime,
				Handler:      metadata.Handler,
			}
			
			if err := o.manager.UpdateFunction(functionName, newConfig); err != nil {
				return currentTimeout, err
			}
			
			return optimalTimeout, nil
		}
	}
	
	return currentTimeout, nil
}

func calculateOptimalMemory(currentMemory int, invocations int64) int {
	if invocations < 1000 {
		return 128
	}
	
	if invocations < 10000 {
		return 256
	}
	
	if invocations < 100000 {
		return 512
	}
	
	return 1024
}

func (o *CostOptimizer) PurchaseSavingsPlan(hours int, hourlyRate float64) (*SavingsPlan, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	plan := &SavingsPlan{
		PlanID:         fmt.Sprintf("sp-%d", time.Now().UnixNano()),
		HoursCommitted: hours,
		HoursUsed:      0,
		HourlyRate:     hourlyRate,
		StartDate:      time.Now(),
		EndDate:        time.Now().AddDate(0, 0, hours/24),
		Status:         "active",
	}
	
	return plan, nil
}

func (o *CostOptimizer) PurchaseReservedCapacity(functionName, instanceType string, quantity, hours int, hourlyRate float64) (*ReservedCapacity, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	capacity := &ReservedCapacity{
		CapacityID:     fmt.Sprintf("rc-%d", time.Now().UnixNano()),
		FunctionName:   functionName,
		InstanceType:   instanceType,
		Quantity:       quantity,
		HoursCommitted: hours,
		HourlyRate:     hourlyRate,
		StartDate:      time.Now(),
		EndDate:        time.Now().AddDate(0, 0, hours/24),
		Status:         "active",
	}
	
	return capacity, nil
}

func (o *CostOptimizer) CalculateSpotSavings(ondemandRate float64) float64 {
	spotDiscount := 0.7
	return ondemandRate * spotDiscount
}

func (o *CostOptimizer) CalculateReservedSavings(ondemandRate float64, hoursCommitted int) float64 {
	if hoursCommitted >= 8760 {
		return ondemandRate * 0.6
	}
	if hoursCommitted >= 4380 {
		return ondemandRate * 0.7
	}
	if hoursCommitted >= 2190 {
		return ondemandRate * 0.8
	}
	return ondemandRate * 0.9
}

func (o *CostOptimizer) CalculateSavingsPlanSavings(ondemandRate float64, hoursCommitted int) float64 {
	if hoursCommitted >= 8760 {
		return ondemandRate * 0.5
	}
	if hoursCommitted >= 4380 {
		return ondemandRate * 0.6
	}
	if hoursCommitted >= 2190 {
		return ondemandRate * 0.7
	}
	return ondemandRate * 0.8
}

func (o *CostOptimizer) UpdateMetrics() {
	functions := o.manager.ListFunctions()
	
	var totalCompute, totalRequest, totalNetwork, totalStorage float64
	var totalInvocations int64
	var totalGBSeconds float64
	
	for _, fn := range functions {
		allocation, err := o.GetCostAllocation(fn.FunctionName)
		if err != nil {
			continue
		}
		
		totalCompute += allocation.ComputeCost
		totalRequest += allocation.RequestCost
		totalNetwork += allocation.NetworkCost
		totalStorage += allocation.StorageCost
		totalInvocations += allocation.Invocations
	}
	
	if totalInvocations > 0 {
		totalGBSeconds = float64(totalInvocations) * 0.1
	}
	
	o.metrics.TotalSpend = totalCompute + totalRequest + totalNetwork + totalStorage
	o.metrics.ComputeSpend = totalCompute
	o.metrics.RequestSpend = totalRequest
	o.metrics.NetworkSpend = totalNetwork
	o.metrics.StorageSpend = totalStorage
	o.metrics.TotalInvocations.Store(totalInvocations)
	o.metrics.TotalGBSeconds = totalGBSeconds
	o.metrics.LastUpdated = time.Now()
	
	if totalInvocations > 0 {
		o.metrics.AvgCostPerInvoke = o.metrics.TotalSpend / float64(totalInvocations)
	}
}

func (o *CostOptimizer) GetHistoricalCost(functionName string, days int) (float64, error) {
	allocation, err := o.GetCostAllocation(functionName)
	if err != nil {
		return 0, err
	}
	
	return allocation.TotalCost * float64(days), nil
}

func (o *CostOptimizer) GetCostForecast(functionName string, days int) (float64, error) {
	allocation, err := o.GetCostAllocation(functionName)
	if err != nil {
		return 0, err
	}
	
	return allocation.TotalCost * float64(days), nil
}

func (o *CostOptimizer) ExportCostData(functionName string, format string) (string, error) {
	allocation, err := o.GetCostAllocation(functionName)
	if err != nil {
		return "", err
	}
	
	return fmt.Sprintf("%v", allocation), nil
}

func (o *CostOptimizer) Enable() {
	o.enabled.Store(true)
}

func (o *CostOptimizer) Disable() {
	o.enabled.Store(false)
}

func (o *CostOptimizer) IsEnabled() bool {
	return o.enabled.Load()
}

func (o *CostOptimizer) Stop() {
	o.cancel()
}
