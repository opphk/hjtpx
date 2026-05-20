package service

import (
	"context"
	"testing"
	"time"
)

func TestNewAIModelLifecycleService(t *testing.T) {
	service := NewAIModelLifecycleService()
	if service == nil {
		t.Fatal("Expected service instance, got nil")
	}
}

func TestListModels(t *testing.T) {
	service := NewAIModelLifecycleService()
	ctx := context.Background()

	models, err := service.ListModels(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(models) == 0 {
		t.Error("Expected at least one model, got none")
	}

	for _, model := range models {
		if model.ID == 0 {
			t.Error("Expected model ID to be set")
		}
		if model.Name == "" {
			t.Error("Expected model name to be set")
		}
		if model.Version == "" {
			t.Error("Expected model version to be set")
		}
		if model.Status == "" {
			t.Error("Expected model status to be set")
		}
	}
}

func TestGetModel(t *testing.T) {
	service := NewAIModelLifecycleService()
	ctx := context.Background()

	testID := uint(1)
	model, err := service.GetModel(ctx, testID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if model == nil {
		t.Fatal("Expected model, got nil")
	}

	if model.ID != testID {
		t.Errorf("Expected model ID %d, got %d", testID, model.ID)
	}
}

func TestUploadModel(t *testing.T) {
	service := NewAIModelLifecycleService()
	ctx := context.Background()

	req := &ModelUploadRequest{
		Name:        "Test Model",
		Description: "Test Description",
		Type:        "classification",
		Version:     "v1.0.0",
		Metadata: map[string]interface{}{
			"accuracy": 0.95,
		},
	}

	model, err := service.UploadModel(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if model == nil {
		t.Fatal("Expected model, got nil")
	}

	if model.Name != req.Name {
		t.Errorf("Expected name %s, got %s", req.Name, model.Name)
	}

	if model.Version != req.Version {
		t.Errorf("Expected version %s, got %s", req.Version, model.Version)
	}

	if model.Status != "draft" {
		t.Errorf("Expected status 'draft', got %s", model.Status)
	}
}

func TestUpdateModel(t *testing.T) {
	service := NewAIModelLifecycleService()
	ctx := context.Background()

	testID := uint(1)
	req := &ModelUploadRequest{
		Name:        "Updated Model",
		Description: "Updated Description",
		Type:        "detection",
		Version:     "v2.0.0",
	}

	model, err := service.UpdateModel(ctx, testID, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if model == nil {
		t.Fatal("Expected model, got nil")
	}

	if model.ID != testID {
		t.Errorf("Expected ID %d, got %d", testID, model.ID)
	}

	if model.Name != req.Name {
		t.Errorf("Expected name %s, got %s", req.Name, model.Name)
	}
}

func TestDeleteModel(t *testing.T) {
	service := NewAIModelLifecycleService()
	ctx := context.Background()

	testID := uint(1)
	err := service.DeleteModel(ctx, testID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestListVersions(t *testing.T) {
	service := NewAIModelLifecycleService()
	ctx := context.Background()

	modelID := uint(1)
	versions, err := service.ListVersions(ctx, modelID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(versions) == 0 {
		t.Error("Expected at least one version, got none")
	}

	for _, version := range versions {
		if version.ModelID != modelID {
			t.Errorf("Expected model ID %d, got %d", modelID, version.ModelID)
		}
		if version.Version == "" {
			t.Error("Expected version string to be set")
		}
	}
}

func TestDeployModel(t *testing.T) {
	service := NewAIModelLifecycleService()
	ctx := context.Background()

	req := &ModelDeployRequest{
		ModelID:       1,
		VersionID:     1,
		Environment:   "production",
		TrafficWeight: 1.0,
	}

	deployment, err := service.DeployModel(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if deployment == nil {
		t.Fatal("Expected deployment, got nil")
	}

	if deployment.ModelID != req.ModelID {
		t.Errorf("Expected model ID %d, got %d", req.ModelID, deployment.ModelID)
	}

	if deployment.Environment != req.Environment {
		t.Errorf("Expected environment %s, got %s", req.Environment, deployment.Environment)
	}
}

func TestGetDeployment(t *testing.T) {
	service := NewAIModelLifecycleService()
	ctx := context.Background()

	testID := uint(1)
	deployment, err := service.GetDeployment(ctx, testID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if deployment == nil {
		t.Fatal("Expected deployment, got nil")
	}

	if deployment.ID != testID {
		t.Errorf("Expected ID %d, got %d", testID, deployment.ID)
	}
}

func TestListDeployments(t *testing.T) {
	service := NewAIModelLifecycleService()
	ctx := context.Background()

	modelID := uint(1)
	deployments, err := service.ListDeployments(ctx, modelID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(deployments) == 0 {
		t.Error("Expected at least one deployment, got none")
	}

	for _, deployment := range deployments {
		if deployment.ModelID != modelID {
			t.Errorf("Expected model ID %d, got %d", modelID, deployment.ModelID)
		}
	}
}

func TestModelStructFields(t *testing.T) {
	model := &AIModel{
		ID:          1,
		Name:        "Test",
		Version:     "v1.0",
		Description: "Desc",
		Status:      "deployed",
		Type:        "classification",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    map[string]interface{}{"key": "value"},
	}

	if model.ID != 1 {
		t.Error("ID field not set correctly")
	}
	if model.Name != "Test" {
		t.Error("Name field not set correctly")
	}
	if model.Version != "v1.0" {
		t.Error("Version field not set correctly")
	}
	if model.Status != "deployed" {
		t.Error("Status field not set correctly")
	}
}

func TestModelVersionStruct(t *testing.T) {
	version := &ModelVersion{
		ID:         1,
		ModelID:    2,
		Version:    "v1.0",
		Checksum:   "sha256:abc123",
		FilePath:   "/path/to/model",
		Status:     "deployed",
		CreatedAt:  time.Now(),
		DeployedAt: time.Now(),
	}

	if version.ID != 1 {
		t.Error("ID field not set correctly")
	}
	if version.ModelID != 2 {
		t.Error("ModelID field not set correctly")
	}
	if version.Version != "v1.0" {
		t.Error("Version field not set correctly")
	}
}

func TestModelDeploymentStruct(t *testing.T) {
	deployment := &ModelDeployment{
		ID:            1,
		ModelID:       1,
		VersionID:     1,
		Environment:   "production",
		Status:        "deployed",
		TrafficWeight: 0.5,
		DeployedAt:    time.Now(),
	}

	if deployment.ID != 1 {
		t.Error("ID field not set correctly")
	}
	if deployment.Environment != "production" {
		t.Error("Environment field not set correctly")
	}
	if deployment.TrafficWeight != 0.5 {
		t.Error("TrafficWeight field not set correctly")
	}
}

func BenchmarkListModels(b *testing.B) {
	service := NewAIModelLifecycleService()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.ListModels(ctx)
	}
}

func BenchmarkGetModel(b *testing.B) {
	service := NewAIModelLifecycleService()
	ctx := context.Background()
	testID := uint(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetModel(ctx, testID)
	}
}
