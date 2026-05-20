package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestAIModelLifecycleService_RegisterModel(t *testing.T) {
	service := NewAIModelLifecycleService()

	model := &AIModel{
		Name:        "Test Model",
		Description: "A test model",
		Type:        "classification",
		Framework:   "tensorflow",
		Owner:      "test-owner",
		Team:       "ai-team",
	}

	err := service.RegisterModel(context.Background(), model)
	if err != nil {
		t.Errorf("RegisterModel() error = %v", err)
	}

	if model.ModelID == "" {
		t.Error("ModelID should be set after registration")
	}
}

func TestAIModelLifecycleService_GetModel(t *testing.T) {
	service := NewAIModelLifecycleService()

	model := &AIModel{
		Name:        "Test Model",
		Description: "A test model",
		Type:        "classification",
	}

	err := service.RegisterModel(context.Background(), model)
	if err != nil {
		t.Fatalf("RegisterModel() error = %v", err)
	}

	retrieved, err := service.GetModel(context.Background(), model.ModelID)
	if err != nil {
		t.Errorf("GetModel() error = %v", err)
	}

	if retrieved.Name != model.Name {
		t.Errorf("Expected name '%s', got '%s'", model.Name, retrieved.Name)
	}
}

func TestAIModelLifecycleService_UpdateModel(t *testing.T) {
	service := NewAIModelLifecycleService()

	model := &AIModel{
		Name:        "Test Model",
		Description: "A test model",
		Type:        "classification",
	}

	err := service.RegisterModel(context.Background(), model)
	if err != nil {
		t.Fatalf("RegisterModel() error = %v", err)
	}

	model.Description = "Updated description"
	err = service.UpdateModel(context.Background(), model)
	if err != nil {
		t.Errorf("UpdateModel() error = %v", err)
	}

	retrieved, _ := service.GetModel(context.Background(), model.ModelID)
	if retrieved.Description != "Updated description" {
		t.Error("Model description was not updated")
	}
}

func TestAIModelLifecycleService_DeleteModel(t *testing.T) {
	service := NewAIModelLifecycleService()

	model := &AIModel{
		Name:        "Test Model",
		Description: "A test model",
		Type:        "classification",
	}

	err := service.RegisterModel(context.Background(), model)
	if err != nil {
		t.Fatalf("RegisterModel() error = %v", err)
	}

	err = service.DeleteModel(context.Background(), model.ModelID)
	if err != nil {
		t.Errorf("DeleteModel() error = %v", err)
	}

	_, err = service.GetModel(context.Background(), model.ModelID)
	if err == nil {
		t.Error("GetModel() should return error after deletion")
	}
}

func TestAIModelLifecycleService_CreateVersion(t *testing.T) {
	service := NewAIModelLifecycleService()

	model := &AIModel{
		Name:        "Test Model",
		Description: "A test model",
		Type:        "classification",
	}

	err := service.RegisterModel(context.Background(), model)
	if err != nil {
		t.Fatalf("RegisterModel() error = %v", err)
	}

	version := &ModelVersion{
		Version: "1.0.0",
		Name:   "Initial Version",
		Metrics: &ModelVersionMetrics{
			Accuracy:  0.95,
			Precision: 0.93,
			Recall:   0.94,
		},
	}

	err = service.CreateVersion(context.Background(), model.ModelID, version)
	if err != nil {
		t.Errorf("CreateVersion() error = %v", err)
	}

	versions, err := service.ListVersions(context.Background(), model.ModelID)
	if err != nil {
		t.Errorf("ListVersions() error = %v", err)
	}

	if len(versions) != 1 {
		t.Errorf("Expected 1 version, got %d", len(versions))
	}
}

func TestAIModelLifecycleService_DeployModel(t *testing.T) {
	service := NewAIModelLifecycleService()

	model := &AIModel{
		Name:        "Test Model",
		Description: "A test model",
		Type:        "classification",
	}

	err := service.RegisterModel(context.Background(), model)
	if err != nil {
		t.Fatalf("RegisterModel() error = %v", err)
	}

	deployment := &ModelDeployment{
		ModelID:    model.ModelID,
		Name:      "Production Deployment",
		Replicas:  3,
		Resources: &DeploymentResources{
			CPU:    "2",
			Memory: "4Gi",
		},
	}

	result, err := service.DeployModel(context.Background(), deployment)
	if err != nil {
		t.Errorf("DeployModel() error = %v", err)
	}

	if !result.Success {
		t.Error("DeployModel() should succeed")
	}

	if result.DeploymentID == "" {
		t.Error("DeploymentID should be set")
	}
}

func TestAIModelLifecycleService_MonitorModel(t *testing.T) {
	service := NewAIModelLifecycleService()

	model := &AIModel{
		Name:        "Test Model",
		Description: "A test model",
		Type:        "classification",
	}

	err := service.RegisterModel(context.Background(), model)
	if err != nil {
		t.Fatalf("RegisterModel() error = %v", err)
	}

	metrics, err := service.MonitorModel(context.Background(), model.ModelID)
	if err != nil {
		t.Errorf("MonitorModel() error = %v", err)
	}

	if metrics.RequestsCount == 0 {
		t.Error("MonitorModel() should return metrics")
	}
}

func TestABTestingPlatformService_CreateExperiment(t *testing.T) {
	service := NewABTestingPlatformService()

	experiment := &Experiment{
		Name:        "Test Experiment",
		Description: "A/B test experiment",
		Type:       "ab_test",
		Variants: []Variant{
			{
				VariantID:  "control-variant",
				Name:       "Control",
				Allocation: 50,
				Control:   true,
				Metrics:   &VariantMetrics{},
			},
			{
				VariantID:  "variant-a",
				Name:       "Variant A",
				Allocation: 50,
				Control:   false,
				Metrics:   &VariantMetrics{},
			},
		},
	}

	err := service.CreateExperiment(context.Background(), experiment)
	if err != nil {
		t.Errorf("CreateExperiment() error = %v", err)
	}

	if experiment.ExperimentID == "" {
		t.Error("ExperimentID should be set")
	}
}

func TestABTestingPlatformService_AllocateVariant(t *testing.T) {
	service := NewABTestingPlatformService()

	experiment := &Experiment{
		Name:        "Test Experiment",
		Description: "A/B test experiment",
		Type:       "ab_test",
		Variants: []Variant{
			{
				VariantID:  "control-variant",
				Name:       "Control",
				Allocation: 50,
				Control:   true,
				Metrics:   &VariantMetrics{},
			},
			{
				VariantID:  "variant-a",
				Name:       "Variant A",
				Allocation: 50,
				Control:   false,
				Metrics:   &VariantMetrics{},
			},
		},
	}

	err := service.CreateExperiment(context.Background(), experiment)
	if err != nil {
		t.Fatalf("CreateExperiment() error = %v", err)
	}

	err = service.StartExperiment(context.Background(), experiment.ExperimentID)
	if err != nil {
		t.Fatalf("StartExperiment() error = %v", err)
	}

	allocation, err := service.AllocateVariant(context.Background(), experiment.ExperimentID, "user-123")
	if err != nil {
		t.Errorf("AllocateVariant() error = %v", err)
	}

	if allocation == nil || allocation.VariantID == "" {
		t.Error("VariantID should be set")
	}
}

func TestExperimentTrackingService_CreateRun(t *testing.T) {
	service := NewExperimentTrackingService()

	run := &ExperimentRun{
		ExperimentID: "exp-123",
		Name:         "Test Run",
		Description:  "A test run",
	}

	err := service.CreateRun(context.Background(), run)
	if err != nil {
		t.Errorf("CreateRun() error = %v", err)
	}

	if run.RunID == "" {
		t.Error("RunID should be set")
	}
}

func TestExperimentTrackingService_LogMetric(t *testing.T) {
	service := NewExperimentTrackingService()

	run := &ExperimentRun{
		ExperimentID: "exp-123",
		Name:         "Test Run",
	}

	err := service.CreateRun(context.Background(), run)
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	metric := &MetricLog{
		Name:  "accuracy",
		Value: 0.95,
		Step: 1,
	}

	err = service.LogMetric(context.Background(), run.RunID, metric)
	if err != nil {
		t.Errorf("LogMetric() error = %v", err)
	}

	metrics, err := service.GetMetrics(context.Background(), run.RunID)
	if err != nil {
		t.Errorf("GetMetrics() error = %v", err)
	}

	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(metrics))
	}
}

func TestModelMonitoringService_CreateMonitor(t *testing.T) {
	service := NewModelMonitoringService()

	monitor := &ModelMonitor{
		ModelID:    "model-123",
		Name:       "Test Monitor",
		Description: "A test monitor",
		Type:       "performance",
		Metrics:    []string{"latency", "accuracy"},
		Thresholds: []Threshold{
			{
				Metric:    "latency",
				Operator:  ">",
				Value:     100,
				Severity:  "warning",
			},
		},
	}

	err := service.CreateMonitor(context.Background(), monitor)
	if err != nil {
		t.Errorf("CreateMonitor() error = %v", err)
	}

	if monitor.MonitorID == "" {
		t.Error("MonitorID should be set")
	}
}

func TestModelMonitoringService_CreateAlert(t *testing.T) {
	service := NewModelMonitoringService()

	monitor := &ModelMonitor{
		ModelID:    "model-123",
		Name:       "Test Monitor",
		Type:       "performance",
		Metrics:    []string{"latency"},
	}

	err := service.CreateMonitor(context.Background(), monitor)
	if err != nil {
		t.Fatalf("CreateMonitor() error = %v", err)
	}

	alert := &MonitoringAlert{
		MonitorID:   monitor.MonitorID,
		Name:        "High Latency Alert",
		Description: "Alert when latency exceeds threshold",
		Condition:   "latency > 100",
		Threshold:  100,
		Severity:   "warning",
	}

	err = service.CreateAlert(context.Background(), alert)
	if err != nil {
		t.Errorf("CreateAlert() error = %v", err)
	}

	if alert.AlertID == "" {
		t.Error("AlertID should be set")
	}
}

func TestAIModel_Filters(t *testing.T) {
	service := NewAIModelLifecycleService()

	models := []*AIModel{
		{Name: "Model 1", Type: "classification", Team: "team-a"},
		{Name: "Model 2", Type: "regression", Team: "team-b"},
		{Name: "Model 3", Type: "classification", Team: "team-a"},
	}

	for _, model := range models {
		service.RegisterModel(context.Background(), model)
	}

	filters := &ModelFilters{
		Type: "classification",
		Team: "team-a",
	}

	results, err := service.ListModels(context.Background(), filters)
	if err != nil {
		t.Errorf("ListModels() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestExperiment_CompareRuns(t *testing.T) {
	service := NewExperimentTrackingService()

	runs := []*ExperimentRun{
		{ExperimentID: "exp-1", Name: "Run 1", Metrics: map[string]float64{"accuracy": 0.9}},
		{ExperimentID: "exp-1", Name: "Run 2", Metrics: map[string]float64{"accuracy": 0.95}},
		{ExperimentID: "exp-1", Name: "Run 3", Metrics: map[string]float64{"accuracy": 0.85}},
	}

	for _, run := range runs {
		service.CreateRun(context.Background(), run)
	}

	runIDs := []string{runs[0].RunID, runs[1].RunID, runs[2].RunID}
	comparison, err := service.CompareRuns(context.Background(), runIDs)
	if err != nil {
		t.Errorf("CompareRuns() error = %v", err)
	}

	if comparison.BestRunID != runs[1].RunID {
		t.Error("Best run should be Run 2 with highest accuracy")
	}
}

func TestMonitorData_Generation(t *testing.T) {
	service := NewModelMonitoringService()

	monitor := &ModelMonitor{
		ModelID:   "model-123",
		Name:      "Test Monitor",
		Type:      "performance",
		Metrics:   []string{"latency", "accuracy"},
		Thresholds: []Threshold{
			{Metric: "latency", Operator: ">", Value: 100, Severity: "warning"},
		},
	}

	err := service.CreateMonitor(context.Background(), monitor)
	if err != nil {
		t.Fatalf("CreateMonitor() error = %v", err)
	}

	period := &MonitoringPeriod{
		Start:       time.Now().Add(-24 * time.Hour),
		End:         time.Now(),
		Granularity: "1h",
	}

	data, err := service.GetMonitorData(context.Background(), monitor.MonitorID, period)
	if err != nil {
		t.Errorf("GetMonitorData() error = %v", err)
	}

	if len(data.DataPoints) == 0 {
		t.Error("MonitorData should contain data points")
	}
}

func TestAIModel_Serialization(t *testing.T) {
	model := &AIModel{
		ModelID:     "model-123",
		Name:        "Test Model",
		Description: "A test model",
		Type:        "classification",
		Stage:       StageDevelopment,
		Tags:        []string{"nlp", "transformer"},
		Metadata:    map[string]interface{}{"version": "1.0"},
	}

	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("Failed to marshal model: %v", err)
	}

	var unmarshaled AIModel
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal model: %v", err)
	}

	if unmarshaled.Name != model.Name {
		t.Errorf("Expected name '%s', got '%s'", model.Name, unmarshaled.Name)
	}
}
