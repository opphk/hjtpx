package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

type CostAnalysisService struct {
	mu              sync.RWMutex
	costModels      map[string]*CostModel
	usageRecords    []UsageRecord
	historicalCosts []CostSnapshot
}

type CostModel struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Provider        string             `json:"provider"`
	ServiceType     string             `json:"service_type"`
	Pricing         *PricingInfo       `json:"pricing"`
	ResourceSpecs   map[string]float64 `json:"resource_specs"`
	EffectiveDate   time.Time         `json:"effective_date"`
	ExpirationDate  *time.Time        `json:"expiration_date,omitempty"`
}

type PricingInfo struct {
	Currency      string             `json:"currency"`
	Unit           string             `json:"unit"`
	UnitPrice     float64            `json:"unit_price"`
	MonthlyPrice  float64            `json:"monthly_price"`
	OnDemandPrice float64            `json:"on_demand_price"`
	TieredPricing []PricingTier      `json:"tiered_pricing,omitempty"`
	Discounts     []Discount         `json:"discounts,omitempty"`
}

type PricingTier struct {
	MinUsage  float64 `json:"min_usage"`
	MaxUsage  float64 `json:"max_usage"`
	UnitPrice float64 `json:"unit_price"`
}

type Discount struct {
	Type         string  `json:"type"`
	Name         string  `json:"name"`
	Percentage   float64 `json:"percentage"`
	MinCommitment float64 `json:"min_commitment,omitempty"`
}

type UsageRecord struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	ServiceName   string    `json:"service_name"`
	ResourceType  string    `json:"resource_type"`
	Quantity      float64   `json:"quantity"`
	Unit          string    `json:"unit"`
	Cost          float64   `json:"cost"`
	Tags          map[string]string `json:"tags"`
}

type CostSnapshot struct {
	Timestamp     time.Time        `json:"timestamp"`
	Period        string           `json:"period"`
	TotalCost     float64          `json:"total_cost"`
	ByService     map[string]float64 `json:"by_service"`
	ByResource    map[string]float64 `json:"by_resource"`
	Trend         string            `json:"trend"`
	ChangePercent float64           `json:"change_percent"`
}

type CostSummary struct {
	CurrentPeriod     CostPeriod          `json:"current_period"`
	PreviousPeriod    CostPeriod          `json:"previous_period"`
	TotalCost         float64             `json:"total_cost"`
	ProjectedCost     float64             `json:"projected_cost"`
	CostBreakdown     []CostBreakdownItem `json:"cost_breakdown"`
	TopCostDrivers    []CostDriver        `json:"top_cost_drivers"`
	CostTrend         []CostTrendPoint    `json:"cost_trend"`
	Recommendations   []CostRecommendation `json:"recommendations"`
}

type CostPeriod struct {
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	TotalDays   int       `json:"total_days"`
	ElapsedDays int       `json:"elapsed_days"`
	Cost        float64   `json:"cost"`
	DailyAverage float64  `json:"daily_average"`
}

type CostBreakdownItem struct {
	Category    string  `json:"category"`
	Amount      float64 `json:"amount"`
	Percentage  float64 `json:"percentage"`
	Trend       string  `json:"trend"`
	ChangeRate  float64 `json:"change_rate"`
}

type CostDriver struct {
	ServiceName    string   `json:"service_name"`
	ResourceType   string   `json:"resource_type"`
	Cost           float64  `json:"cost"`
	UsageQuantity  float64  `json:"usage_quantity"`
	UnitCost       float64  `json:"unit_cost"`
	Trend          string   `json:"trend"`
	OptimizationPotential float64 `json:"optimization_potential"`
}

type CostTrendPoint struct {
	Date    time.Time `json:"date"`
	Cost    float64   `json:"cost"`
	Accumulated float64 `json:"accumulated"`
}

type CostRecommendation struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Savings     float64 `json:"savings"`
	Effort      string   `json:"effort"`
	Priority    int      `json:"priority"`
	Actions     []string `json:"actions"`
}

type ResourceCost struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceName string                 `json:"resource_name"`
	Type         string                 `json:"type"`
	HourlyCost   float64                `json:"hourly_cost"`
	DailyCost    float64                `json:"daily_cost"`
	MonthlyCost  float64                `json:"monthly_cost"`
	UsageHours   float64                `json:"usage_hours"`
	Metrics      map[string]interface{} `json:"metrics"`
}

type CostAllocation struct {
	Project     string   `json:"project"`
	Environment string   `json:"environment"`
	Team        string   `json:"team"`
	Cost        float64  `json:"cost"`
	Percentage  float64  `json:"percentage"`
}

func NewCostAnalysisService() *CostAnalysisService {
	service := &CostAnalysisService{
		costModels:      make(map[string]*CostModel),
		usageRecords:    make([]UsageRecord, 0),
		historicalCosts: make([]CostSnapshot, 0),
	}
	service.initializeCostModels()
	service.generateHistoricalData()
	return service
}

func (s *CostAnalysisService) initializeCostModels() {
	s.costModels = map[string]*CostModel{
		"compute": {
			ID:          "compute",
			Name:        "计算实例",
			Provider:    "aws",
			ServiceType: "ec2",
			Pricing: &PricingInfo{
				Currency:      "USD",
				Unit:          "instance-hour",
				UnitPrice:     0.1,
				MonthlyPrice:  72.0,
				OnDemandPrice: 0.1,
				TieredPricing: []PricingTier{
					{MinUsage: 0, MaxUsage: 100, UnitPrice: 0.1},
					{MinUsage: 100, MaxUsage: 500, UnitPrice: 0.09},
					{MinUsage: 500, MaxUsage: math.MaxFloat64, UnitPrice: 0.08},
				},
				Discounts: []Discount{
					{Type: "reserved", Name: "1年预留实例", Percentage: 40},
					{Type: "reserved", Name: "3年预留实例", Percentage: 60},
				},
			},
			ResourceSpecs: map[string]float64{
				"cpu":     2.0,
				"memory":  8.0,
				"storage": 100.0,
			},
			EffectiveDate: time.Now().AddDate(0, -6, 0),
		},
		"storage": {
			ID:          "storage",
			Name:        "对象存储",
			Provider:    "aws",
			ServiceType: "s3",
			Pricing: &PricingInfo{
				Currency:      "USD",
				Unit:          "GB-month",
				UnitPrice:     0.023,
				MonthlyPrice:  0,
				OnDemandPrice: 0.023,
			},
			ResourceSpecs: map[string]float64{
				"storage_gb": 1000.0,
			},
			EffectiveDate: time.Now().AddDate(0, -6, 0),
		},
		"database": {
			ID:          "database",
			Name:        "数据库服务",
			Provider:    "aws",
			ServiceType: "rds",
			Pricing: &PricingInfo{
				Currency:      "USD",
				Unit:          "instance-hour",
				UnitPrice:     0.17,
				MonthlyPrice:  122.4,
				OnDemandPrice: 0.17,
				TieredPricing: []PricingTier{
					{MinUsage: 0, MaxUsage: 100, UnitPrice: 0.17},
					{MinUsage: 100, MaxUsage: math.MaxFloat64, UnitPrice: 0.15},
				},
			},
			ResourceSpecs: map[string]float64{
				"cpu":     2.0,
				"memory":  16.0,
				"storage": 500.0,
			},
			EffectiveDate: time.Now().AddDate(0, -6, 0),
		},
		"network": {
			ID:          "network",
			Name:        "网络传输",
			Provider:    "aws",
			ServiceType: "data-transfer",
			Pricing: &PricingInfo{
				Currency:      "USD",
				Unit:          "GB",
				UnitPrice:     0.09,
				MonthlyPrice:  0,
				OnDemandPrice: 0.09,
			},
			ResourceSpecs: map[string]float64{
				"data_transfer_gb": 100.0,
			},
			EffectiveDate: time.Now().AddDate(0, -6, 0),
		},
		"cache": {
			ID:          "cache",
			Name:        "缓存服务",
			Provider:    "aws",
			ServiceType: "elasticache",
			Pricing: &PricingInfo{
				Currency:      "USD",
				Unit:          "node-hour",
				UnitPrice:     0.05,
				MonthlyPrice:  36.0,
				OnDemandPrice: 0.05,
			},
			ResourceSpecs: map[string]float64{
				"cpu":     1.0,
				"memory":  3.0,
			},
			EffectiveDate: time.Now().AddDate(0, -6, 0),
		},
		"api-gateway": {
			ID:          "api-gateway",
			Name:        "API网关",
			Provider:    "aws",
			ServiceType: "apigateway",
			Pricing: &PricingInfo{
				Currency:      "USD",
				Unit:          "million-requests",
				UnitPrice:     3.5,
				MonthlyPrice:  0,
				OnDemandPrice: 3.5,
			},
			ResourceSpecs: map[string]float64{
				"requests_million": 10.0,
			},
			EffectiveDate: time.Now().AddDate(0, -6, 0),
		},
	}
}

func (s *CostAnalysisService) generateHistoricalData() {
	now := time.Now()

	for i := 30; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		snapshot := s.generateDailySnapshot(date)
		s.historicalCosts = append(s.historicalCosts, snapshot)
	}
}

func (s *CostAnalysisService) generateDailySnapshot(date time.Time) CostSnapshot {
	dayOfWeek := float64(date.Weekday())
	weekendFactor := 1.0
	if dayOfWeek == 0 || dayOfWeek == 6 {
		weekendFactor = 0.7
	}

	baseCost := 500.0 * weekendFactor
	variation := math.Mod(float64(date.UnixNano()), 50)

	computeCost := baseCost * 0.4 * (1 + variation/1000)
	storageCost := baseCost * 0.2 * (1 + variation/1500)
	databaseCost := baseCost * 0.25 * (1 + variation/2000)
	networkCost := baseCost * 0.1 * (1 + variation/2500)
	otherCost := baseCost * 0.05

	trend := "stable"
	if variation > 25 {
		trend = "increasing"
	} else if variation < -25 {
		trend = "decreasing"
	}

	daysSinceStart := 30 - int(time.Since(date).Hours()/24)
	changePercent := float64(daysSinceStart) * 0.5

	return CostSnapshot{
		Timestamp: date,
		Period:     "daily",
		TotalCost:  computeCost + storageCost + databaseCost + networkCost + otherCost,
		ByService: map[string]float64{
			"compute":  computeCost,
			"storage":  storageCost,
			"database": databaseCost,
			"network":  networkCost,
			"other":    otherCost,
		},
		ByResource: map[string]float64{
			"cpu_hours":      computeCost / 0.3,
			"storage_gb":     storageCost / 0.023,
			"db_hours":       databaseCost / 0.17,
			"data_transfer":  networkCost / 0.09,
		},
		Trend:         trend,
		ChangePercent: changePercent,
	}
}

func (s *CostAnalysisService) GetCostSummary(ctx context.Context) (*CostSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	currentPeriod := CostPeriod{
		StartDate:   startOfMonth,
		EndDate:     endOfMonth,
		TotalDays:   endOfMonth.Day(),
		ElapsedDays:  now.Day(),
	}

	var monthlyTotal float64
	var byService = make(map[string]float64)

	for _, snapshot := range s.historicalCosts {
		if snapshot.Timestamp.After(startOfMonth) {
			monthlyTotal += snapshot.TotalCost
			for service, cost := range snapshot.ByService {
				byService[service] += cost
			}
		}
	}

	currentPeriod.Cost = monthlyTotal
	currentPeriod.DailyAverage = monthlyTotal / float64(currentPeriod.ElapsedDays)

	projectedCost := currentPeriod.DailyAverage * float64(currentPeriod.TotalDays)

	previousPeriod := CostPeriod{
		StartDate:    startOfMonth.AddDate(0, -1, 0),
		EndDate:      startOfMonth,
		TotalDays:    30,
		ElapsedDays:  30,
		Cost:         monthlyTotal * 0.95,
		DailyAverage: monthlyTotal * 0.95 / 30,
	}

	var costBreakdown []CostBreakdownItem
	var totalByCategory float64
	for _, cost := range byService {
		totalByCategory += cost
	}

	for category, cost := range byService {
		percentage := 0.0
		if totalByCategory > 0 {
			percentage = cost / totalByCategory * 100
		}

		trend := "stable"
		if percentage > 30 {
			trend = "increasing"
		}

		costBreakdown = append(costBreakdown, CostBreakdownItem{
			Category:   category,
			Amount:      cost,
			Percentage:  percentage,
			Trend:       trend,
			ChangeRate:  5.0,
		})
	}

	sort.Slice(costBreakdown, func(i, j int) bool {
		return costBreakdown[i].Amount > costBreakdown[j].Amount
	})

	var topCostDrivers []CostDriver
	for serviceName, cost := range byService {
		driver := CostDriver{
			ServiceName:    serviceName,
			ResourceType:   serviceName,
			Cost:           cost,
			UsageQuantity:  cost / 0.1,
			UnitCost:       0.1,
			Trend:          "stable",
			OptimizationPotential: cost * 0.15,
		}
		topCostDrivers = append(topCostDrivers, driver)
	}

	sort.Slice(topCostDrivers, func(i, j int) bool {
		return topCostDrivers[i].Cost > topCostDrivers[j].Cost
	})

	if len(topCostDrivers) > 5 {
		topCostDrivers = topCostDrivers[:5]
	}

	var costTrend []CostTrendPoint
	var accumulated float64
	for _, snapshot := range s.historicalCosts {
		accumulated += snapshot.TotalCost
		costTrend = append(costTrend, CostTrendPoint{
			Date:        snapshot.Timestamp,
			Cost:        snapshot.TotalCost,
			Accumulated: accumulated,
		})
	}

	recommendations := s.generateRecommendations(byService, monthlyTotal)

	return &CostSummary{
		CurrentPeriod:     currentPeriod,
		PreviousPeriod:    previousPeriod,
		TotalCost:         monthlyTotal,
		ProjectedCost:     projectedCost,
		CostBreakdown:     costBreakdown,
		TopCostDrivers:    topCostDrivers,
		CostTrend:          costTrend,
		Recommendations:   recommendations,
	}, nil
}

func (s *CostAnalysisService) generateRecommendations(byService map[string]float64, totalCost float64) []CostRecommendation {
	var recommendations []CostRecommendation

	if computeCost, ok := byService["compute"]; ok {
		if computeCost > totalCost*0.4 {
			recommendations = append(recommendations, CostRecommendation{
				ID:          "rec-001",
				Category:    "compute",
				Title:       "优化计算成本",
				Description: "计算成本占比过高，建议使用预留实例或优化实例类型",
				Savings:     computeCost * 0.3,
				Effort:      "medium",
				Priority:    1,
				Actions:     []string{"评估预留实例需求", "优化实例类型", "使用Spot实例处理批处理"},
			})
		}
	}

	if storageCost, ok := byService["storage"]; ok {
		if storageCost > totalCost*0.2 {
			recommendations = append(recommendations, CostRecommendation{
				ID:          "rec-002",
				Category:    "storage",
				Title:       "降低存储成本",
				Description: "存储成本较高，建议启用生命周期策略和压缩",
				Savings:     storageCost * 0.25,
				Effort:      "low",
				Priority:    2,
				Actions:     []string{"启用自动归档", "使用智能分层存储", "清理过期数据"},
			})
		}
	}

	recommendations = append(recommendations, CostRecommendation{
		ID:          "rec-003",
		Category:    "general",
		Title:       "启用成本监控",
		Description: "建议设置预算告警和成本异常检测",
		Savings:     totalCost * 0.1,
		Effort:      "low",
		Priority:    3,
		Actions:     []string{"设置月度预算", "配置成本告警", "启用预算报告"},
	})

	recommendations = append(recommendations, CostRecommendation{
		ID:          "rec-004",
		Category:    "architecture",
		Title:       "优化架构设计",
		Description: "建议使用无服务器架构处理间歇性工作负载",
		Savings:     totalCost * 0.2,
		Effort:      "high",
		Priority:    4,
		Actions:     []string{"评估无服务器适用场景", "迁移非关键服务", "优化事件驱动架构"},
	})

	return recommendations
}

func (s *CostAnalysisService) GetCostByService(ctx context.Context, startDate, endDate time.Time) (map[string]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	byService := make(map[string]float64)

	for _, snapshot := range s.historicalCosts {
		if snapshot.Timestamp.After(startDate) && snapshot.Timestamp.Before(endDate) {
			for service, cost := range snapshot.ByService {
				byService[service] += cost
			}
		}
	}

	return byService, nil
}

func (s *CostAnalysisService) GetCostByResource(ctx context.Context, startDate, endDate time.Time) (map[string]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	byResource := make(map[string]float64)

	for _, snapshot := range s.historicalCosts {
		if snapshot.Timestamp.After(startDate) && snapshot.Timestamp.Before(endDate) {
			for resource, cost := range snapshot.ByResource {
				byResource[resource] += cost
			}
		}
	}

	return byResource, nil
}

func (s *CostAnalysisService) GetResourceCosts(ctx context.Context) ([]ResourceCost, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resources := []ResourceCost{
		{
			ResourceID:   "res-001",
			ResourceName: "API Server",
			Type:         "compute",
			HourlyCost:   0.15,
			DailyCost:    3.6,
			MonthlyCost:  108.0,
			UsageHours:   24,
			Metrics: map[string]interface{}{
				"cpu_utilization": 45.0,
				"memory_utilization": 60.0,
			},
		},
		{
			ResourceID:   "res-002",
			ResourceName: "Database Primary",
			Type:         "database",
			HourlyCost:   0.34,
			DailyCost:    8.16,
			MonthlyCost:  244.8,
			UsageHours:   24,
			Metrics: map[string]interface{}{
				"connections": 85,
				"queries_per_second": 150,
			},
		},
		{
			ResourceID:   "res-003",
			ResourceName: "Cache Cluster",
			Type:         "cache",
			HourlyCost:   0.10,
			DailyCost:    2.4,
			MonthlyCost:  72.0,
			UsageHours:   24,
			Metrics: map[string]interface{}{
				"hit_rate": 92.0,
				"memory_usage": 75.0,
			},
		},
		{
			ResourceID:   "res-004",
			ResourceName: "Object Storage",
			Type:         "storage",
			HourlyCost:   0.023,
			DailyCost:    0.552,
			MonthlyCost:  16.56,
			UsageHours:   24,
			Metrics: map[string]interface{}{
				"storage_used_gb": 720,
				"requests_count": 10000,
			},
		},
	}

	return resources, nil
}

func (s *CostAnalysisService) GetCostAllocation(ctx context.Context) ([]CostAllocation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	allocations := []CostAllocation{
		{
			Project:     "核心业务",
			Environment: "production",
			Team:        "平台团队",
			Cost:        3500.0,
			Percentage:  70.0,
		},
		{
			Project:     "内部工具",
			Environment: "staging",
			Team:        "研发团队",
			Cost:        1000.0,
			Percentage:  20.0,
		},
		{
			Project:     "测试环境",
			Environment: "test",
			Team:        "QA团队",
			Cost:        500.0,
			Percentage:  10.0,
		},
	}

	return allocations, nil
}

func (s *CostAnalysisService) GetCostForecast(ctx context.Context, horizonDays int) ([]CostTrendPoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var forecast []CostTrendPoint

	if len(s.historicalCosts) == 0 {
		return forecast, nil
	}

	lastSnapshot := s.historicalCosts[len(s.historicalCosts)-1]
	avgDailyCost := lastSnapshot.TotalCost

	for i := 1; i <= horizonDays; i++ {
		date := time.Now().AddDate(0, 0, i)

		trend := 1.0 + (float64(i) * 0.002)

		weekendFactor := 1.0
		if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
			weekendFactor = 0.7
		}

		projectedCost := avgDailyCost * trend * weekendFactor

		forecast = append(forecast, CostTrendPoint{
			Date:        date,
			Cost:        projectedCost,
			Accumulated: 0,
		})
	}

	return forecast, nil
}

func (s *CostAnalysisService) GetPricingModels(ctx context.Context) ([]*CostModel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	models := make([]*CostModel, 0, len(s.costModels))
	for _, model := range s.costModels {
		models = append(models, model)
	}

	return models, nil
}

func (s *CostAnalysisService) CalculateResourceCost(ctx context.Context, resourceType string, quantity float64, duration time.Duration) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	model, exists := s.costModels[resourceType]
	if !exists {
		return 0, fmt.Errorf("cost model not found: %s", resourceType)
	}

	hours := duration.Hours()
	cost := model.Pricing.UnitPrice * quantity * hours

	if len(model.Pricing.TieredPricing) > 0 {
		for _, tier := range model.Pricing.TieredPricing {
			if quantity >= tier.MinUsage && quantity < tier.MaxUsage {
				cost = tier.UnitPrice * quantity * hours
				break
			}
		}
	}

	return cost, nil
}

func (s *CostAnalysisService) GetCostAnomalies(ctx context.Context, threshold float64) ([]CostSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var anomalies []CostSnapshot

	if len(s.historicalCosts) < 2 {
		return anomalies, nil
	}

	var totalCost float64
	for _, snapshot := range s.historicalCosts {
		totalCost += snapshot.TotalCost
	}
	avgCost := totalCost / float64(len(s.historicalCosts))

	for _, snapshot := range s.historicalCosts {
		ratio := snapshot.TotalCost / avgCost
		if ratio > threshold {
			anomalies = append(anomalies, snapshot)
		}
	}

	return anomalies, nil
}

func (s *CostAnalysisService) AddUsageRecord(ctx context.Context, record UsageRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record.ID = fmt.Sprintf("usage-%d", len(s.usageRecords)+1)
	record.Timestamp = time.Now()

	s.usageRecords = append(s.usageRecords, record)

	return nil
}

func (s *CostAnalysisService) GetUsageRecords(ctx context.Context, startDate, endDate time.Time) ([]UsageRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var records []UsageRecord
	for _, record := range s.usageRecords {
		if record.Timestamp.After(startDate) && record.Timestamp.Before(endDate) {
			records = append(records, record)
		}
	}

	return records, nil
}

func (s *CostAnalysisService) ExportCostReport(ctx context.Context, format string) ([]byte, error) {
	summary, err := s.GetCostSummary(ctx)
	if err != nil {
		return nil, err
	}

	report := fmt.Sprintf("Cost Analysis Report - %s\n", time.Now().Format(time.RFC3339))
	report += fmt.Sprintf("Total Cost: $%.2f\n", summary.TotalCost)
	report += fmt.Sprintf("Projected Cost: $%.2f\n", summary.ProjectedCost)
	report += fmt.Sprintf("Daily Average: $%.2f\n\n", summary.CurrentPeriod.DailyAverage)

	report += "Cost Breakdown:\n"
	for _, item := range summary.CostBreakdown {
		report += fmt.Sprintf("  %s: $%.2f (%.1f%%)\n", item.Category, item.Amount, item.Percentage)
	}

	report += "\nTop Cost Drivers:\n"
	for i, driver := range summary.TopCostDrivers {
		report += fmt.Sprintf("  %d. %s: $%.2f (Potential savings: $%.2f)\n", i+1, driver.ServiceName, driver.Cost, driver.OptimizationPotential)
	}

	report += "\nRecommendations:\n"
	for _, rec := range summary.Recommendations {
		report += fmt.Sprintf("  - %s: Potential savings $%.2f\n", rec.Title, rec.Savings)
	}

	return []byte(report), nil
}

func (s *CostAnalysisService) GetCostTrend(ctx context.Context, period string) ([]CostTrendPoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var trend []CostTrendPoint
	var accumulated float64

	for _, snapshot := range s.historicalCosts {
		accumulated += snapshot.TotalCost
		trend = append(trend, CostTrendPoint{
			Date:        snapshot.Timestamp,
			Cost:        snapshot.TotalCost,
			Accumulated: accumulated,
		})
	}

	return trend, nil
}

func (s *CostAnalysisService) ComparePeriods(ctx context.Context, period1, period2 string) (*CostComparison, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var cost1, cost2 float64

	for _, snapshot := range s.historicalCosts {
		if snapshot.Period == period1 {
			cost1 += snapshot.TotalCost
		}
		if snapshot.Period == period2 {
			cost2 += snapshot.TotalCost
		}
	}

	changePercent := 0.0
	if cost1 > 0 {
		changePercent = (cost2 - cost1) / cost1 * 100
	}

	return &CostComparison{
		Period1:        period1,
		Period2:        period2,
		Cost1:          cost1,
		Cost2:          cost2,
		ChangePercent:  changePercent,
		ChangeAmount:   cost2 - cost1,
	}, nil
}

type CostComparison struct {
	Period1       string  `json:"period1"`
	Period2       string  `json:"period2"`
	Cost1         float64 `json:"cost1"`
	Cost2         float64 `json:"cost2"`
	ChangePercent float64 `json:"change_percent"`
	ChangeAmount  float64 `json:"change_amount"`
}

func (s *CostAnalysisService) SetBudget(ctx context.Context, budget *CostBudget) error {
	return nil
}

type CostBudget struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Amount      float64   `json:"amount"`
	Period      string    `json:"period"`
	AlertThreshold float64 `json:"alert_threshold"`
}
