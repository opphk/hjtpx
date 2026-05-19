package service

import (
	"context"
	"testing"
)

func TestNewBudgetAlert(t *testing.T) {
	alert := NewBudgetAlert()

	if alert == nil {
		t.Fatal("NewBudgetAlert returned nil")
	}

	if len(alert.budgets) == 0 {
		t.Error("No budgets initialized")
	}

	if len(alert.channels) == 0 {
		t.Error("No alert channels initialized")
	}
}

func TestBudgetAlert_GetBudgetStatus(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	status, err := alert.GetBudgetStatus(ctx)
	if err != nil {
		t.Fatalf("GetBudgetStatus failed: %v", err)
	}

	if status == nil {
		t.Fatal("GetBudgetStatus returned nil")
	}

	if status.CurrentSpent < 0 {
		t.Errorf("Invalid current spent: %f", status.CurrentSpent)
	}

	if status.BudgetAmount < 0 {
		t.Errorf("Invalid budget amount: %f", status.BudgetAmount)
	}

	if status.Percentage < 0 || status.Percentage > 100 {
		t.Errorf("Invalid percentage: %f", status.Percentage)
	}
}

func TestBudgetAlert_GetAllBudgetStatus(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	statuses, err := alert.GetAllBudgetStatus(ctx)
	if err != nil {
		t.Fatalf("GetAllBudgetStatus failed: %v", err)
	}

	if len(statuses) == 0 {
		t.Error("GetAllBudgetStatus returned no statuses")
	}
}

func TestBudgetAlert_CreateBudget(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	budget := &Budget{
		Name:   "Test Budget",
		Amount: 1000,
		Period: "monthly",
		Enabled: true,
	}

	err := alert.CreateBudget(ctx, budget)
	if err != nil {
		t.Errorf("CreateBudget failed: %v", err)
	}

	if budget.ID == "" {
		t.Error("Budget ID not set")
	}
}

func TestBudgetAlert_UpdateBudget(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	existing, _ := alert.GetBudget(ctx, "monthly-total")
	existing.Name = "Updated Budget"

	err := alert.UpdateBudget(ctx, existing)
	if err != nil {
		t.Errorf("UpdateBudget failed: %v", err)
	}

	updated, _ := alert.GetBudget(ctx, "monthly-total")
	if updated.Name != "Updated Budget" {
		t.Error("Budget not updated")
	}
}

func TestBudgetAlert_DeleteBudget(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	err := alert.DeleteBudget(ctx, "monthly-total")
	if err != nil {
		t.Errorf("DeleteBudget failed: %v", err)
	}

	_, err = alert.GetBudget(ctx, "monthly-total")
	if err == nil {
		t.Error("Budget still exists after deletion")
	}
}

func TestBudgetAlert_GetBudget(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	budget, err := alert.GetBudget(ctx, "monthly-total")
	if err != nil {
		t.Fatalf("GetBudget failed: %v", err)
	}

	if budget == nil {
		t.Fatal("GetBudget returned nil")
	}

	if budget.ID != "monthly-total" {
		t.Errorf("Budget ID mismatch: got %s, want monthly-total", budget.ID)
	}

	_, err = alert.GetBudget(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent budget")
	}
}

func TestBudgetAlert_GetAllBudgets(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	budgets, err := alert.GetAllBudgets(ctx)
	if err != nil {
		t.Fatalf("GetAllBudgets failed: %v", err)
	}

	if len(budgets) == 0 {
		t.Error("GetAllBudgets returned no budgets")
	}
}

func TestBudgetAlert_GetAlerts(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	alerts, err := alert.GetAlerts(ctx, "monthly-total", 10)
	if err != nil {
		t.Fatalf("GetAlerts failed: %v", err)
	}

	if alerts == nil {
		t.Error("GetAlerts returned nil")
	}
}

func TestBudgetAlert_AcknowledgeAlert(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	err := alert.AcknowledgeAlert(ctx, "non-existent", "test-user")
	if err == nil {
		t.Error("Expected error for non-existent alert")
	}
}

func TestBudgetAlert_UpdateBudgetSpending(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	err := alert.UpdateBudgetSpending(ctx, "monthly-total", 3000)
	if err != nil {
		t.Errorf("UpdateBudgetSpending failed: %v", err)
	}

	budget, _ := alert.GetBudget(ctx, "monthly-total")
	if budget.CurrentSpent != 3000 {
		t.Errorf("Budget spent not updated: %f", budget.CurrentSpent)
	}
}

func TestBudgetAlert_GetChannels(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	channels, err := alert.GetChannels(ctx)
	if err != nil {
		t.Fatalf("GetChannels failed: %v", err)
	}

	if len(channels) == 0 {
		t.Error("GetChannels returned no channels")
	}
}

func TestBudgetAlert_EnableChannel(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	err := alert.EnableChannel(ctx, "sms")
	if err != nil {
		t.Errorf("EnableChannel failed: %v", err)
	}
}

func TestBudgetAlert_DisableChannel(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	err := alert.DisableChannel(ctx, "email")
	if err != nil {
		t.Errorf("DisableChannel failed: %v", err)
	}
}

func TestBudgetAlert_GetBudgetReport(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	report, err := alert.GetBudgetReport(ctx, "monthly")
	if err != nil {
		t.Fatalf("GetBudgetReport failed: %v", err)
	}

	if report == nil {
		t.Fatal("GetBudgetReport returned nil")
	}

	if report.TotalBudget < 0 {
		t.Errorf("Invalid total budget: %f", report.TotalBudget)
	}

	if report.TotalSpent < 0 {
		t.Errorf("Invalid total spent: %f", report.TotalSpent)
	}

	if report.ComplianceRate < 0 || report.ComplianceRate > 100 {
		t.Errorf("Invalid compliance rate: %f", report.ComplianceRate)
	}
}

func TestBudgetAlert_ExportBudgetData(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	data, err := alert.ExportBudgetData(ctx, "json")
	if err != nil {
		t.Fatalf("ExportBudgetData failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("ExportBudgetData returned empty data")
	}
}

func TestBudgetAlert_GetSpendingForecast(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	forecast, err := alert.GetSpendingForecast(ctx, "monthly-total")
	if err != nil {
		t.Fatalf("GetSpendingForecast failed: %v", err)
	}

	if forecast == nil {
		t.Fatal("GetSpendingForecast returned nil")
	}

	if forecast.CurrentSpent < 0 {
		t.Errorf("Invalid current spent: %f", forecast.CurrentSpent)
	}

	if forecast.ProjectedSpend < 0 {
		t.Errorf("Invalid projected spend: %f", forecast.ProjectedSpend)
	}
}

func TestBudgetAlert_SetBudgetAmount(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	err := alert.SetBudgetAmount(ctx, "monthly-total", 6000)
	if err != nil {
		t.Errorf("SetBudgetAmount failed: %v", err)
	}

	budget, _ := alert.GetBudget(ctx, "monthly-total")
	if budget.Amount != 6000 {
		t.Errorf("Budget amount not updated: %f", budget.Amount)
	}
}

func TestBudgetAlert_AddThreshold(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	threshold := AlertThreshold{
		Percentage: 85,
		Level:      "warning",
		Message:    "Budget at 85%",
		Color:      "orange",
	}

	err := alert.AddThreshold(ctx, "monthly-total", threshold)
	if err != nil {
		t.Errorf("AddThreshold failed: %v", err)
	}
}

func TestBudgetAlert_RemoveThreshold(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	err := alert.RemoveThreshold(ctx, "monthly-total", 50)
	if err != nil {
		t.Errorf("RemoveThreshold failed: %v", err)
	}
}

func TestBudgetAlert_EnableBudget(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	err := alert.EnableBudget(ctx, "monthly-total")
	if err != nil {
		t.Errorf("EnableBudget failed: %v", err)
	}

	budget, _ := alert.GetBudget(ctx, "monthly-total")
	if !budget.Enabled {
		t.Error("Budget not enabled")
	}
}

func TestBudgetAlert_DisableBudget(t *testing.T) {
	alert := NewBudgetAlert()
	ctx := context.Background()

	err := alert.DisableBudget(ctx, "monthly-total")
	if err != nil {
		t.Errorf("DisableBudget failed: %v", err)
	}

	budget, _ := alert.GetBudget(ctx, "monthly-total")
	if budget.Enabled {
		t.Error("Budget not disabled")
	}
}

func TestTriggerAlert(t *testing.T) {
	alert := NewBudgetAlert()
	budget := alert.budgets["monthly-total"]

	threshold := AlertThreshold{
		Percentage: 50,
		Level:      "info",
		Message:    "Budget at 50%",
		Color:      "green",
	}

	percentage := 55.0

	alert.triggerAlert(budget, threshold, percentage)

	alerts := alert.alerts[budget.ID]
	if len(alerts) == 0 {
		t.Error("Expected alert to be triggered")
	}
}

func TestSendNotifications(t *testing.T) {
	alert := NewBudgetAlert()

	budget := &Budget{
		Notifications: []Notification{
			{Channel: "email", Recipients: []string{"test@example.com"}, Enabled: true},
			{Channel: "slack", Recipients: []string{"#test"}, Enabled: true},
		},
	}

	record := &BudgetAlertRecord{
		ID:       "test-alert",
		Message:  "Test alert",
		Level:    "warning",
	}

	alert.sendNotifications(budget, record)
}

func TestExecuteAutoActions(t *testing.T) {
	alert := NewBudgetAlert()

	budget := &Budget{
		AutoActions: []AutoAction{
			{Type: "notify", Threshold: 75, Enabled: true, Parameters: map[string]interface{}{"channels": []string{"email"}}},
			{Type: "scale_down", Threshold: 80, Enabled: true, Parameters: map[string]interface{}{"percentage": 20}},
		},
	}

	threshold := AlertThreshold{
		Percentage: 85,
		Level:      "warning",
	}

	alert.executeAutoActions(budget, threshold)
}

func TestCheckAllBudgets(t *testing.T) {
	alert := NewBudgetAlert()

	alert.checkAllBudgets()
}
