package service

import (
	"context"
	"testing"
	"time"
)

func TestNewCostAnalysisService(t *testing.T) {
	service := NewCostAnalysisService()

	if service == nil {
		t.Fatal("NewCostAnalysisService returned nil")
	}

	if len(service.costModels) == 0 {
		t.Error("No cost models initialized")
	}
}

func TestCostAnalysisService_GetCostSummary(t *testing.T) {
	service := NewCostAnalysisService()
	ctx := context.Background()

	summary, err := service.GetCostSummary(ctx)
	if err != nil {
		t.Fatalf("GetCostSummary failed: %v", err)
	}

	if summary == nil {
		t.Fatal("GetCostSummary returned nil")
	}

	if summary.TotalCost < 0 {
		t.Errorf("Invalid total cost: %f", summary.TotalCost)
	}

	if len(summary.CostBreakdown) == 0 {
		t.Error("No cost breakdown")
	}
}

func TestCostAnalysisService_GetCostByService(t *testing.T) {
	service := NewCostAnalysisService()
	ctx := context.Background()

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	costs, err := service.GetCostByService(ctx, startDate, endDate)
	if err != nil {
		t.Fatalf("GetCostByService failed: %v", err)
	}

	if costs == nil {
		t.Error("GetCostByService returned nil")
	}
}

func TestCostAnalysisService_GetCostByResource(t *testing.T) {
	service := NewCostAnalysisService()
	ctx := context.Background()

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	costs, err := service.GetCostByResource(ctx, startDate, endDate)
	if err != nil {
		t.Fatalf("GetCostByResource failed: %v", err)
	}

	if costs == nil {
		t.Error("GetCostByResource returned nil")
	}
}

func TestCostAnalysisService_GetResourceCosts(t *testing.T) {
	service := NewCostAnalysisService()
	ctx := context.Background()

	resources, err := service.GetResourceCosts(ctx)
	if err != nil {
		t.Fatalf("GetResourceCosts failed: %v", err)
	}

	if len(resources) == 0 {
		t.Error("GetResourceCosts returned no resources")
	}

	for _, resource := range resources {
		if resource.MonthlyCost < 0 {
			t.Errorf("Invalid monthly cost for %s: %f", resource.ResourceName, resource.MonthlyCost)
		}
	}
}

func TestCostAnalysisService_GetCostAllocation(t *testing.T) {
	service := NewCostAnalysisService()
	ctx := context.Background()

	allocations, err := service.GetCostAllocation(ctx)
	if err != nil {
		t.Fatalf("GetCostAllocation failed: %v", err)
	}

	if len(allocations) == 0 {
		t.Error("GetCostAllocation returned no allocations")
	}

	var totalPercentage float64
	for _, alloc := range allocations {
		totalPercentage += alloc.Percentage
	}
}

func TestCostAnalysisService_GetCostForecast(t *testing.T) {
	service := NewCostAnalysisService()
	ctx := context.Background()

	forecast, err := service.GetCostForecast(ctx, 7)
	if err != nil {
		t.Fatalf("GetCostForecast failed: %v", err)
	}

	if len(forecast) == 0 {
		t.Error("GetCostForecast returned no forecast")
	}

	for _, point := range forecast {
		if point.Cost < 0 {
			t.Errorf("Invalid cost in forecast: %f", point.Cost)
		}
	}
}

func TestCostAnalysisService_GetPricingModels(t *testing.T) {
	service := NewCostAnalysisService()
	ctx := context.Background()

	models, err := service.GetPricingModels(ctx)
	if err != nil {
		t.Fatalf("GetPricingModels failed: %v", err)
	}

	if len(models) == 0 {
		t.Error("GetPricingModels returned no models")
	}
}

func TestCostAnalysisService_CalculateResourceCost(t *testing.T) {
	service := NewCostAnalysisService()
	ctx := context.Background()

	cost, err := service.CalculateResourceCost(ctx, "compute", 10, 24*time.Hour)
	if err != nil {
		t.Fatalf("CalculateResourceCost failed: %v", err)
	}

	if cost < 0 {
		t.Errorf("Invalid cost: %f", cost)
	}

	_, err = service.CalculateResourceCost(ctx, "non_existent", 10, 24*time.Hour)
	if err == nil {
		t.Error("Expected error for non-existent resource type")
	}
}

func TestCostAnalysisService_GetCostAnomalies(t *testing.T) {
	service := NewCostAnalysisService()
	ctx := context.Background()

	anomalies, err := service.GetCostAnomalies(ctx, 1.5)
	if err != nil {
		t.Fatalf("GetCostAnomalies failed: %v", err)
	}

	if anomalies == nil {
		t.Error("GetCostAnomalies returned nil")
	}
}

func TestCostAnalysisService_AddUsageRecord(t *testing.T) {
	service := NewCostAnalysisService()
	ctx := context.Background()

	record := UsageRecord{
		ServiceName:  "test-service",
		ResourceType: "compute",
		Quantity:     10,
		Cost:         25.0,
		Tags:         map[string]string{"env": "test"},
	}

	err := service.AddUsageRecord(ctx, record)
	if err != nil {
		t.Errorf("AddUsageRecord failed: %v", err)
	}
}

func TestCostAnalysisService_GetUsageRecords(t *testing.T) {
	service := NewCostAnalysisService()
	ctx := context.Background()

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	records, err := service.GetUsageRecords(ctx, startDate, endDate)
	if err != nil {
		t.Fatalf("GetUsageRecords failed: %v", err)
	}

	if records == nil {
		t.Error("GetUsageRecords returned nil")
	}
}

func TestCostAnalysisService_ExportCostReport(t *testing.T) {
	service := NewCostAnalysisService()
	ctx := context.Background()

	data, err := service.ExportCostReport(ctx, "json")
	if err != nil {
		t.Fatalf("ExportCostReport failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("ExportCostReport returned empty data")
	}
}

func TestCostAnalysisService_GetCostTrend(t *testing.T) {
	service := NewCostAnalysisService()
	ctx := context.Background()

	trend, err := service.GetCostTrend(ctx, "daily")
	if err != nil {
		t.Fatalf("GetCostTrend failed: %v", err)
	}

	if trend == nil {
		t.Error("GetCostTrend returned nil")
	}
}

func TestCostAnalysisService_ComparePeriods(t *testing.T) {
	service := NewCostAnalysisService()
	ctx := context.Background()

	comparison, err := service.ComparePeriods(ctx, "monthly", "weekly")
	if err != nil {
		t.Fatalf("ComparePeriods failed: %v", err)
	}

	if comparison == nil {
		t.Fatal("ComparePeriods returned nil")
	}
}

func TestGenerateDailySnapshot(t *testing.T) {
	service := NewCostAnalysisService()

	date := time.Now()
	snapshot := service.generateDailySnapshot(date)

	if snapshot.Timestamp != date {
		t.Error("Snapshot timestamp mismatch")
	}

	if snapshot.TotalCost < 0 {
		t.Errorf("Invalid total cost: %f", snapshot.TotalCost)
	}

	if snapshot.Period != "daily" {
		t.Error("Snapshot period should be daily")
	}
}

func TestGenerateRecommendations(t *testing.T) {
	service := NewCostAnalysisService()

	byService := map[string]float64{
		"compute":  2500,
		"storage":  800,
		"database": 1000,
	}

	recommendations := service.generateRecommendations(byService, 5000)

	if len(recommendations) == 0 {
		t.Error("Expected recommendations")
	}

	for _, rec := range recommendations {
		if rec.ID == "" {
			t.Error("Recommendation ID is empty")
		}
		if rec.Savings < 0 {
			t.Errorf("Invalid savings for %s: %f", rec.Title, rec.Savings)
		}
	}
}
