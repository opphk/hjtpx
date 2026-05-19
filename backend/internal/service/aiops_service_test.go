package service

import (
	"context"
	"testing"
	"time"
)

func TestNewAIOpsService(t *testing.T) {
	service := NewAIOpsService()

	if service == nil {
		t.Fatal("NewAIOpsService returned nil")
	}

	if service.logAnomalyDetector == nil {
		t.Error("logAnomalyDetector is nil")
	}

	if service.performancePredictor == nil {
		t.Error("performancePredictor is nil")
	}

	if service.rootCauseAnalyzer == nil {
		t.Error("rootCauseAnalyzer is nil")
	}

	if service.autoRemediation == nil {
		t.Error("autoRemediation is nil")
	}

	if service.costAnalysis == nil {
		t.Error("costAnalysis is nil")
	}

	if service.budgetAlert == nil {
		t.Error("budgetAlert is nil")
	}
}

func TestAIOpsService_PerformAnalysis(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	snapshot, err := service.PerformAnalysis(ctx)
	if err != nil {
		t.Fatalf("PerformAnalysis failed: %v", err)
	}

	if snapshot == nil {
		t.Fatal("PerformAnalysis returned nil snapshot")
	}

	if snapshot.HealthScore < 0 || snapshot.HealthScore > 100 {
		t.Errorf("Invalid health score: %f", snapshot.HealthScore)
	}

	if snapshot.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}

	if len(snapshot.Metrics.CPUUsage) == 0 {
		t.Log("Warning: CPU usage is empty")
	}
}

func TestAIOpsService_GetDashboard(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	dashboard, err := service.GetDashboard(ctx)
	if err != nil {
		t.Fatalf("GetDashboard failed: %v", err)
	}

	if dashboard == nil {
		t.Fatal("GetDashboard returned nil")
	}

	if dashboard.OverallHealth < 0 || dashboard.OverallHealth > 100 {
		t.Errorf("Invalid overall health: %f", dashboard.OverallHealth)
	}

	if dashboard.ActiveAlerts < 0 {
		t.Errorf("Invalid active alerts count: %d", dashboard.ActiveAlerts)
	}
}

func TestAIOpsService_GetAlerts(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	alerts, err := service.GetAlerts(ctx, AlertFilter{})
	if err != nil {
		t.Fatalf("GetAlerts failed: %v", err)
	}

	if alerts == nil {
		t.Error("GetAlerts returned nil")
	}
}

func TestAIOpsService_GetAlertsWithFilter(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	filter := AlertFilter{
		Type:     "anomaly",
		Severity: "critical",
	}

	alerts, err := service.GetAlerts(ctx, filter)
	if err != nil {
		t.Fatalf("GetAlerts with filter failed: %v", err)
	}

	for _, alert := range alerts {
		if alert.Type != filter.Type {
			t.Errorf("Alert type mismatch: got %s, want %s", alert.Type, filter.Type)
		}
	}
}

func TestAIOpsService_AcknowledgeAlert(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	err := service.AcknowledgeAlert(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent alert")
	}

	snapshot, _ := service.PerformAnalysis(ctx)
	if len(snapshot.Alerts) > 0 {
		alertID := snapshot.Alerts[0].ID
		err = service.AcknowledgeAlert(ctx, alertID)
		if err != nil {
			t.Errorf("AcknowledgeAlert failed: %v", err)
		}
	}
}

func TestAIOpsService_ResolveAlert(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	err := service.ResolveAlert(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent alert")
	}

	snapshot, _ := service.PerformAnalysis(ctx)
	if len(snapshot.Alerts) > 0 {
		alertID := snapshot.Alerts[0].ID
		err = service.ResolveAlert(ctx, alertID)
		if err != nil {
			t.Errorf("ResolveAlert failed: %v", err)
		}
	}
}

func TestAIOpsService_GetAnalysisHistory(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	service.PerformAnalysis(ctx)
	service.PerformAnalysis(ctx)
	service.PerformAnalysis(ctx)

	history, err := service.GetAnalysisHistory(ctx, 2)
	if err != nil {
		t.Fatalf("GetAnalysisHistory failed: %v", err)
	}

	if len(history) > 2 {
		t.Errorf("Expected max 2 history items, got %d", len(history))
	}
}

func TestAIOpsService_GetMetrics(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	metrics, err := service.GetMetrics(ctx)
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}

	if metrics.CPUUsage < 0 || metrics.CPUUsage > 100 {
		t.Errorf("Invalid CPU usage: %f", metrics.CPUUsage)
	}

	if metrics.MemoryUsage < 0 || metrics.MemoryUsage > 100 {
		t.Errorf("Invalid memory usage: %f", metrics.MemoryUsage)
	}
}

func TestAIOpsService_GetPredictions(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	predictions, err := service.GetPredictions(ctx)
	if err != nil {
		t.Fatalf("GetPredictions failed: %v", err)
	}

	if predictions == nil {
		t.Error("GetPredictions returned nil")
	}
}

func TestAIOpsService_GetCostAnalysis(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	summary, err := service.GetCostAnalysis(ctx)
	if err != nil {
		t.Fatalf("GetCostAnalysis failed: %v", err)
	}

	if summary == nil {
		t.Error("GetCostAnalysis returned nil")
	}

	if summary.TotalCost < 0 {
		t.Errorf("Invalid total cost: %f", summary.TotalCost)
	}
}

func TestAIOpsService_GetBudgetStatus(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	status, err := service.GetBudgetStatus(ctx)
	if err != nil {
		t.Fatalf("GetBudgetStatus failed: %v", err)
	}

	if status == nil {
		t.Error("GetBudgetStatus returned nil")
	}
}

func TestAIOpsService_GetHealthScore(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	service.PerformAnalysis(ctx)

	score := service.GetHealthScore(ctx)
	if score < 0 || score > 100 {
		t.Errorf("Invalid health score: %f", score)
	}
}

func TestAIOpsService_ExportAnalysisReport(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	report, err := service.ExportAnalysisReport(ctx, "json")
	if err != nil {
		t.Fatalf("ExportAnalysisReport (json) failed: %v", err)
	}

	if len(report) == 0 {
		t.Error("ExportAnalysisReport returned empty data")
	}

	htmlReport, err := service.ExportAnalysisReport(ctx, "html")
	if err != nil {
		t.Fatalf("ExportAnalysisReport (html) failed: %v", err)
	}

	if len(htmlReport) == 0 {
		t.Error("ExportAnalysisReport (html) returned empty data")
	}

	_, err = service.ExportAnalysisReport(ctx, "invalid")
	if err == nil {
		t.Error("Expected error for invalid format")
	}
}

func TestCalculateHealthScore(t *testing.T) {
	service := NewAIOpsService()

	tests := []struct {
		name   string
		metrics OperationalMetrics
		alerts  []Alert
	}{
		{
			name: "Healthy system",
			metrics: OperationalMetrics{
				CPUUsage:    30,
				MemoryUsage: 40,
				ErrorRate:   1,
			},
			alerts: []Alert{},
		},
		{
			name: "High load system",
			metrics: OperationalMetrics{
				CPUUsage:    85,
				MemoryUsage: 90,
				ErrorRate:   5,
			},
			alerts: []Alert{},
		},
		{
			name: "System with critical alerts",
			metrics: OperationalMetrics{
				CPUUsage:    50,
				MemoryUsage: 50,
				ErrorRate:   2,
			},
			alerts: []Alert{
				{Severity: "critical", Resolved: false},
				{Severity: "critical", Resolved: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.calculateHealthScore(tt.metrics, tt.alerts)
			if score < 0 || score > 100 {
				t.Errorf("calculateHealthScore returned invalid score: %f", score)
			}
		})
	}
}

func TestCreateAlertFromAnomaly(t *testing.T) {
	service := NewAIOpsService()

	anomaly := Anomaly{
		Type:        "error_spike",
		Severity:    "critical",
		Description: "Test anomaly",
	}

	alert := service.createAlertFromAnomaly(anomaly)

	if alert.ID == "" {
		t.Error("Alert ID is empty")
	}

	if alert.Type != "anomaly" {
		t.Errorf("Alert type mismatch: got %s, want anomaly", alert.Type)
	}

	if alert.Severity != anomaly.Severity {
		t.Errorf("Alert severity mismatch: got %s, want %s", alert.Severity, anomaly.Severity)
	}

	if alert.Resolved {
		t.Error("Alert should not be resolved by default")
	}
}

func TestCreateAlertFromPrediction(t *testing.T) {
	service := NewAIOpsService()

	prediction := Prediction{
		MetricName:     "cpu_usage",
		CurrentValue:   50,
		PredictedValue: 85,
		AlertLevel:     "warning",
	}

	alert := service.createAlertFromPrediction(prediction)

	if alert.ID == "" {
		t.Error("Alert ID is empty")
	}

	if alert.Type != "prediction" {
		t.Errorf("Alert type mismatch: got %s, want prediction", alert.Type)
	}

	if alert.Severity != prediction.AlertLevel {
		t.Errorf("Alert severity mismatch: got %s, want %s", alert.Severity, prediction.AlertLevel)
	}
}

func TestGenerateRecommendations(t *testing.T) {
	service := NewAIOpsService()

	metrics := OperationalMetrics{
		CPUUsage:     75,
		MemoryUsage:  85,
		CacheHitRate: 70,
	}

	predictions := []Prediction{
		{
			MetricName:   "cpu_usage",
			AlertLevel:   "critical",
			Confidence:   0.9,
		},
	}

	recommendations := service.generateRecommendations(metrics, predictions)

	if len(recommendations) == 0 {
		t.Error("Expected recommendations, got none")
	}

	for _, rec := range recommendations {
		if rec.ID == "" {
			t.Error("Recommendation ID is empty")
		}
		if rec.Title == "" {
			t.Error("Recommendation title is empty")
		}
	}
}

func TestAnalyzeTrends(t *testing.T) {
	service := NewAIOpsService()

	trends := service.analyzeTrends()

	if trends.PerformanceTrend == "" {
		t.Error("Performance trend is empty")
	}

	if trends.CostTrend == "" {
		t.Error("Cost trend is empty")
	}
}
