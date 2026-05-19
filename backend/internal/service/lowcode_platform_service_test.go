package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLowCodePlatformService(t *testing.T) {
	svc := NewLowCodePlatformService()
	assert.NotNil(t, svc)
}

func TestCreateIntegration(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	integration := &Integration{
		Name:        "Test Integration",
		Description: "Test description",
		AppID:       "app-123",
		Type:        "workflow",
		Steps: []IntegrationStep{
			{StepID: "step1", Name: "First Step", Type: "http_request", Order: 1},
			{StepID: "step2", Name: "Second Step", Type: "transform", Order: 2},
		},
	}

	err := svc.CreateIntegration(ctx, integration)

	require.NoError(t, err)
	assert.NotEmpty(t, integration.ID)
	assert.Equal(t, "draft", integration.Status)
	assert.Equal(t, "1.0.0", integration.Version)
}

func TestCreateIntegration_Nil(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	err := svc.CreateIntegration(ctx, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestGetIntegration(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	integration := &Integration{
		Name:    "Get Test",
		AppID:   "app-get",
		Steps:   []IntegrationStep{},
	}

	err := svc.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	retrieved, err := svc.GetIntegration(ctx, integration.ID)

	require.NoError(t, err)
	assert.Equal(t, integration.ID, retrieved.ID)
	assert.Equal(t, integration.Name, retrieved.Name)
}

func TestGetIntegration_NotFound(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	integration, err := svc.GetIntegration(ctx, "non-existent")

	assert.Error(t, err)
	assert.Nil(t, integration)
	assert.Equal(t, ErrIntegrationNotFound, err)
}

func TestUpdateIntegration(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	integration := &Integration{
		Name:  "Update Test",
		AppID: "app-update",
		Steps: []IntegrationStep{},
	}

	err := svc.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	integration.Name = "Updated Name"
	integration.Status = "active"

	err = svc.UpdateIntegration(ctx, integration)
	require.NoError(t, err)

	updated, err := svc.GetIntegration(ctx, integration.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, "active", updated.Status)
}

func TestUpdateIntegration_NotFound(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	integration := &Integration{
		ID:    "non-existent",
		Name:  "Test",
		AppID: "app",
	}

	err := svc.UpdateIntegration(ctx, integration)

	assert.Error(t, err)
	assert.Equal(t, ErrIntegrationNotFound, err)
}

func TestDeleteIntegration(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	integration := &Integration{
		Name:  "Delete Test",
		AppID: "app-delete",
		Steps: []IntegrationStep{},
	}

	err := svc.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	err = svc.DeleteIntegration(ctx, integration.ID)
	require.NoError(t, err)

	_, err = svc.GetIntegration(ctx, integration.ID)
	assert.Error(t, err)
}

func TestDeleteIntegration_NotFound(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	err := svc.DeleteIntegration(ctx, "non-existent")

	assert.Error(t, err)
	assert.Equal(t, ErrIntegrationNotFound, err)
}

func TestListIntegrations(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		integration := &Integration{
			Name:  "List Test " + string(rune('0'+i)),
			AppID: "app-list",
			Steps: []IntegrationStep{},
		}
		err := svc.CreateIntegration(ctx, integration)
		require.NoError(t, err)
	}

	integrations, err := svc.ListIntegrations(ctx, "app-list", 10, 0)

	require.NoError(t, err)
	assert.Len(t, integrations, 5)
}

func TestListIntegrations_WithPagination(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		integration := &Integration{
			Name:  "Pagination Test " + string(rune('0'+i)),
			AppID: "app-paginate",
			Steps: []IntegrationStep{},
		}
		err := svc.CreateIntegration(ctx, integration)
		require.NoError(t, err)
	}

	page1, err := svc.ListIntegrations(ctx, "app-paginate", 3, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 3)

	page2, err := svc.ListIntegrations(ctx, "app-paginate", 3, 3)
	require.NoError(t, err)
	assert.Len(t, page2, 3)

	page3, err := svc.ListIntegrations(ctx, "app-paginate", 3, 6)
	require.NoError(t, err)
	assert.Len(t, page3, 3)

	page4, err := svc.ListIntegrations(ctx, "app-paginate", 3, 9)
	require.NoError(t, err)
	assert.Len(t, page4, 1)
}

func TestExecuteIntegration(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	integration := &Integration{
		Name:  "Execute Test",
		AppID: "app-exec",
		Steps: []IntegrationStep{
			{StepID: "step1", Name: "First", Type: "http_request", Order: 1},
			{StepID: "step2", Name: "Second", Type: "transform", Order: 2},
		},
	}

	err := svc.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	result, err := svc.ExecuteIntegration(ctx, integration.ID, map[string]interface{}{"test": "data"})

	require.NoError(t, err)
	assert.NotEmpty(t, result.ExecutionID)
	assert.Equal(t, integration.ID, result.IntegrationID)
	assert.Len(t, result.StepsResults, 2)
	assert.Equal(t, "completed", result.Status)
}

func TestExecuteIntegration_NotFound(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	result, err := svc.ExecuteIntegration(ctx, "non-existent", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrIntegrationNotFound, err)
}

func TestExecuteIntegration_AllStepTypes(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	stepTypes := []string{"http_request", "transform", "condition", "verify_captcha", "report_risk", "blockchain_record", "iot_auth"}

	steps := []IntegrationStep{}
	for i, stepType := range stepTypes {
		steps = append(steps, IntegrationStep{
			StepID: "step" + string(rune('0'+i)),
			Name:   stepType,
			Type:   stepType,
			Order:  i,
		})
	}

	integration := &Integration{
		Name:  "All Types Test",
		AppID: "app-types",
		Steps: steps,
	}

	err := svc.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	result, err := svc.ExecuteIntegration(ctx, integration.ID, nil)

	require.NoError(t, err)
	assert.Len(t, result.StepsResults, len(stepTypes))
}

func TestGetIntegrationTemplates(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	templates, err := svc.GetIntegrationTemplates(ctx)

	require.NoError(t, err)
	assert.Greater(t, len(templates), 0)
}

func TestGetIntegrationTemplates_DefaultTemplates(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	templates, err := svc.GetIntegrationTemplates(ctx)

	require.NoError(t, err)

	webTemplate := false
	iotTemplate := false
	apiTemplate := false

	for _, tmpl := range templates {
		if tmpl.TemplateID == "web-login-flow" {
			webTemplate = true
			assert.NotEmpty(t, tmpl.Variables)
		}
		if tmpl.TemplateID == "iot-authentication" {
			iotTemplate = true
		}
		if tmpl.TemplateID == "api-protection" {
			apiTemplate = true
		}
	}

	assert.True(t, webTemplate)
	assert.True(t, iotTemplate)
	assert.True(t, apiTemplate)
}

func TestValidateIntegration_Valid(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	integration := &Integration{
		Name:        "Valid Integration",
		Description: "Test",
		AppID:       "app-valid",
		Type:        "workflow",
		Steps: []IntegrationStep{
			{StepID: "step1", Name: "Step 1", Type: "http_request", Order: 1},
			{StepID: "step2", Name: "Step 2", Type: "transform", Order: 2},
		},
	}

	result, err := svc.ValidateIntegration(ctx, integration)

	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Len(t, result.Errors, 0)
}

func TestValidateIntegration_MissingName(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	integration := &Integration{
		Name: "",
		AppID: "app-no-name",
		Steps: []IntegrationStep{
			{StepID: "step1", Name: "Step", Type: "http_request", Order: 1},
		},
	}

	result, err := svc.ValidateIntegration(ctx, integration)

	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "name")
}

func TestValidateIntegration_MissingAppID(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	integration := &Integration{
		Name:  "Test",
		AppID: "",
		Steps: []IntegrationStep{
			{StepID: "step1", Name: "Step", Type: "http_request", Order: 1},
		},
	}

	result, err := svc.ValidateIntegration(ctx, integration)

	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "App ID")
}

func TestValidateIntegration_NoSteps(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	integration := &Integration{
		Name:  "No Steps",
		AppID: "app-no-steps",
		Steps: []IntegrationStep{},
	}

	result, err := svc.ValidateIntegration(ctx, integration)

	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "step")
}

func TestValidateIntegration_DuplicateStepIDs(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	integration := &Integration{
		Name:  "Duplicate IDs",
		AppID: "app-dup",
		Steps: []IntegrationStep{
			{StepID: "step1", Name: "Step 1", Type: "http_request", Order: 1},
			{StepID: "step1", Name: "Step 2", Type: "transform", Order: 2},
		},
	}

	result, err := svc.ValidateIntegration(ctx, integration)

	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "Duplicate")
}

func TestGetExecutionHistory(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	integration := &Integration{
		Name:  "History Test",
		AppID: "app-history",
		Steps: []IntegrationStep{
			{StepID: "step1", Name: "Step", Type: "http_request", Order: 1},
		},
	}

	err := svc.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		_, err := svc.ExecuteIntegration(ctx, integration.ID, nil)
		require.NoError(t, err)
	}

	history, err := svc.GetExecutionHistory(ctx, integration.ID, 10, 0)

	require.NoError(t, err)
	assert.Len(t, history, 3)
}

func TestGetExecutionHistory_Empty(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	history, err := svc.GetExecutionHistory(ctx, "non-existent", 10, 0)

	require.NoError(t, err)
	assert.Len(t, history, 0)
}

func TestRunAutomatedTest(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	integration := &Integration{
		Name:  "Test Integration",
		AppID: "app-test",
		Steps: []IntegrationStep{
			{StepID: "step1", Name: "Step 1", Type: "http_request", Order: 1},
			{StepID: "step2", Name: "Step 2", Type: "transform", Order: 2},
		},
	}

	err := svc.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	result, err := svc.RunAutomatedTest(ctx, integration.ID)

	require.NoError(t, err)
	assert.NotEmpty(t, result.TestID)
	assert.Equal(t, "completed", result.Status)
	assert.Greater(t, result.TotalTests, 0)
	assert.Greater(t, result.PassedTests, 0)
}

func TestRunAutomatedTest_NotFound(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	result, err := svc.RunAutomatedTest(ctx, "non-existent")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestExecutionMetrics(t *testing.T) {
	svc := NewLowCodePlatformService()
	ctx := context.Background()

	integration := &Integration{
		Name:  "Metrics Test",
		AppID: "app-metrics",
		Steps: []IntegrationStep{
			{StepID: "step1", Name: "Step 1", Type: "http_request", Order: 1},
			{StepID: "step2", Name: "Step 2", Type: "transform", Order: 2},
			{StepID: "step3", Name: "Step 3", Type: "verify_captcha", Order: 3},
		},
	}

	err := svc.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	result, err := svc.ExecuteIntegration(ctx, integration.ID, nil)
	require.NoError(t, err)

	assert.Equal(t, 3, result.Metrics.TotalSteps)
	assert.Equal(t, 3, result.Metrics.CompletedSteps)
	assert.Equal(t, 0, result.Metrics.FailedSteps)
	assert.Greater(t, result.Duration, time.Duration(0))
	assert.NotEmpty(t, result.Metrics.StepTimes)
}
