package service

import (
	"context"
	"testing"
	"time"
)

func TestAIModelLifecycleService_CreateModel(t *testing.T) {
	service := NewAIModelLifecycleService()

	model := &AIModel{
		Name:        "Test Model",
		Version:     "v1.0.0",
		Type:        "classification",
		Framework:   "pytorch",
		Description: "Test model description",
		Status:      "draft",
		CreatedBy:   "admin",
	}

	err := service.CreateModel(context.Background(), model)
	if err != nil {
		t.Fatalf("CreateModel() error = %v", err)
	}

	if model.ModelID == "" {
		t.Error("Expected ModelID to be set")
	}

	retrieved, err := service.GetModel(context.Background(), model.ModelID)
	if err != nil {
		t.Fatalf("GetModel() error = %v", err)
	}

	if retrieved.Name != model.Name {
		t.Errorf("Expected model name %s, got %s", model.Name, retrieved.Name)
	}
}

func TestAIModelLifecycleService_ListModels(t *testing.T) {
	service := NewAIModelLifecycleService()

	models := []*AIModel{
		{Name: "Model 1", Type: "classification", Status: "deployed"},
		{Name: "Model 2", Type: "detection", Status: "draft"},
		{Name: "Model 3", Type: "classification", Status: "deployed"},
	}

	for _, m := range models {
		service.CreateModel(context.Background(), m)
	}

	filter := &ModelFilter{Type: "classification"}
	result, err := service.ListModels(context.Background(), filter)
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 classification models, got %d", len(result))
	}
}

func TestAIModelLifecycleService_TrainModel(t *testing.T) {
	service := NewAIModelLifecycleService()

	model := &AIModel{
		Name: "Training Test Model",
		Type: "classification",
	}
	service.CreateModel(context.Background(), model)

	config := &TrainingConfig{
		ModelID:    model.ModelID,
		MaxEpochs:  10,
		BatchSize:  32,
	}

	job, err := service.TrainModel(context.Background(), config)
	if err != nil {
		t.Fatalf("TrainModel() error = %v", err)
	}

	if job.JobID == "" {
		t.Error("Expected JobID to be set")
	}

	if job.Status != "pending" {
		t.Errorf("Expected status 'pending', got %s", job.Status)
	}

	time.Sleep(2 * time.Second)

	updatedJob, err := service.GetTrainingJob(context.Background(), job.JobID)
	if err != nil {
		t.Fatalf("GetTrainingJob() error = %v", err)
	}

	if updatedJob.CurrentEpoch == 0 {
		t.Error("Expected epoch to progress")
	}
}

func TestAIModelLifecycleService_DeployModel(t *testing.T) {
	service := NewAIModelLifecycleService()

	model := &AIModel{
		Name:    "Deployment Test Model",
		Version: "v1.0.0",
		Type:    "classification",
		Status:  "trained",
	}
	service.CreateModel(context.Background(), model)

	config := &DeploymentConfig{
		ModelID:     model.ModelID,
		ModelVersion: model.Version,
		Environment: "production",
		Replicas:    3,
	}

	deployment, err := service.DeployModel(context.Background(), model.ModelID, config)
	if err != nil {
		t.Fatalf("DeployModel() error = %v", err)
	}

	if deployment.DeploymentID == "" {
		t.Error("Expected DeploymentID to be set")
	}

	if deployment.Status == "" {
		t.Error("Expected Status to be set")
	}

	if deployment.Endpoints == nil {
		t.Error("Expected Endpoints to be set")
	}

	time.Sleep(2 * time.Second)

	deployment, err = service.GetDeployment(context.Background(), deployment.DeploymentID)
	if err != nil {
		t.Fatalf("GetDeployment() error = %v", err)
	}

	if deployment.ReadyReplicas != config.Replicas {
		t.Errorf("Expected %d ready replicas, got %d", config.Replicas, deployment.ReadyReplicas)
	}
}

func TestAIModelLifecycleService_ScaleDeployment(t *testing.T) {
	service := NewAIModelLifecycleService()

	model := &AIModel{Name: "Scale Test Model", Type: "classification", Status: "trained"}
	service.CreateModel(context.Background(), model)

	deployment, _ := service.DeployModel(context.Background(), model.ModelID, &DeploymentConfig{
		ModelID:     model.ModelID,
		Environment: "production",
		Replicas:    2,
	})

	time.Sleep(2 * time.Second)

	err := service.ScaleDeployment(context.Background(), deployment.DeploymentID, 5)
	if err != nil {
		t.Fatalf("ScaleDeployment() error = %v", err)
	}

	updated, _ := service.GetDeployment(context.Background(), deployment.DeploymentID)
	if updated.Replicas != 5 {
		t.Errorf("Expected 5 replicas, got %d", updated.Replicas)
	}
}

func TestExperimentTrackingService(t *testing.T) {
	service := NewExperimentTrackingService()

	exp := &Experiment{
		Name:        "Test Experiment",
		Description: "Test description",
		Type:        "hyperparameter",
		ModelID:    "MODEL001",
		Status:     "pending",
		CreatedBy:  "admin",
	}

	err := service.CreateExperiment(context.Background(), exp)
	if err != nil {
		t.Fatalf("CreateExperiment() error = %v", err)
	}

	metric := &MetricLog{
		ExperimentID: exp.ExperimentID,
		Step:        1,
		Metrics: map[string]float64{
			"accuracy": 0.95,
			"loss":     0.05,
		},
	}

	err = service.LogMetric(context.Background(), metric)
	if err != nil {
		t.Fatalf("LogMetric() error = %v", err)
	}

	metrics, err := service.GetMetrics(context.Background(), exp.ExperimentID)
	if err != nil {
		t.Fatalf("GetMetrics() error = %v", err)
	}

	if len(metrics) == 0 {
		t.Error("Expected metrics to be logged")
	}
}

func TestModelMonitoringService(t *testing.T) {
	service := NewModelMonitoringService()

	metrics, err := service.GetModelMetrics(context.Background(), "MODEL001")
	if err != nil {
		t.Fatalf("GetModelMetrics() error = %v", err)
	}

	if metrics.Performance == nil {
		t.Error("Expected performance metrics")
	}

	if metrics.Health == nil {
		t.Error("Expected health metrics")
	}

	if metrics.ResourceUsage == nil {
		t.Error("Expected resource usage metrics")
	}
}

func TestModelMonitoringService_Alerts(t *testing.T) {
	service := NewModelMonitoringService()

	alert := &Alert{
		ModelID:    "MODEL001",
		Type:       "performance",
		Severity:   "warning",
		Title:      "High Latency",
		Description: "Model latency exceeded threshold",
		Status:     "active",
	}

	err := service.CreateAlert(context.Background(), alert)
	if err != nil {
		t.Fatalf("CreateAlert() error = %v", err)
	}

	activeAlerts, err := service.GetActiveAlerts(context.Background())
	if err != nil {
		t.Fatalf("GetActiveAlerts() error = %v", err)
	}

	if len(activeAlerts) != 1 {
		t.Errorf("Expected 1 active alert, got %d", len(activeAlerts))
	}

	err = service.AcknowledgeAlert(context.Background(), alert.AlertID)
	if err != nil {
		t.Fatalf("AcknowledgeAlert() error = %v", err)
	}

	err = service.ResolveAlert(context.Background(), alert.AlertID, "Fixed by scaling replicas")
	if err != nil {
		t.Fatalf("ResolveAlert() error = %v", err)
	}
}
