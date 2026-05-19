package service

import (
	"context"
	"testing"
)

func TestNewAutoRemediation(t *testing.T) {
	remediation := NewAutoRemediation()

	if remediation == nil {
		t.Fatal("NewAutoRemediation returned nil")
	}

	if len(remediation.playbooks) == 0 {
		t.Error("No playbooks initialized")
	}
}

func TestAutoRemediation_RecommendActions(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	alert := Alert{
		ID:       "test-alert",
		Type:     "anomaly",
		Severity: "critical",
		Title:    "High CPU usage",
	}

	actions := remediation.RecommendActions(ctx, alert)

	if len(actions) == 0 {
		t.Error("Expected actions for critical alert")
	}
}

func TestAutoRemediation_ExecuteAction(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	action := RemediationAction{
		ID:          "test-action",
		Type:        "command",
		Description: "Test action",
		Command:     "echo 'test'",
	}

	result, err := remediation.ExecuteAction(ctx, action)
	if err != nil {
		t.Fatalf("ExecuteAction failed: %v", err)
	}

	if result == nil {
		t.Fatal("ExecuteAction returned nil")
	}

	if result.Status != "success" {
		t.Errorf("Expected success status, got %s", result.Status)
	}
}

func TestAutoRemediation_GetPlaybooks(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	playbooks, err := remediation.GetPlaybooks(ctx)
	if err != nil {
		t.Fatalf("GetPlaybooks failed: %v", err)
	}

	if len(playbooks) == 0 {
		t.Error("GetPlaybooks returned no playbooks")
	}
}

func TestAutoRemediation_GetPlaybook(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	playbook, err := remediation.GetPlaybook(ctx, "high-cpu")
	if err != nil {
		t.Fatalf("GetPlaybook failed: %v", err)
	}

	if playbook == nil {
		t.Fatal("GetPlaybook returned nil")
	}

	if playbook.ID != "high-cpu" {
		t.Errorf("Playbook ID mismatch: got %s, want high-cpu", playbook.ID)
	}

	_, err = remediation.GetPlaybook(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent playbook")
	}
}

func TestAutoRemediation_EnablePlaybook(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	err := remediation.EnablePlaybook(ctx, "high-cpu")
	if err != nil {
		t.Errorf("EnablePlaybook failed: %v", err)
	}

	playbook, _ := remediation.GetPlaybook(ctx, "high-cpu")
	if !playbook.Enabled {
		t.Error("Playbook not enabled")
	}
}

func TestAutoRemediation_DisablePlaybook(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	err := remediation.DisablePlaybook(ctx, "high-cpu")
	if err != nil {
		t.Errorf("DisablePlaybook failed: %v", err)
	}

	playbook, _ := remediation.GetPlaybook(ctx, "high-cpu")
	if playbook.Enabled {
		t.Error("Playbook not disabled")
	}
}

func TestAutoRemediation_CreatePlaybook(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	newPlaybook := &RemediationPlaybook{
		Name:        "Test Playbook",
		Description: "Test description",
		Steps: []PlaybookStep{
			{ID: "step-1", Name: "Test Step", Type: "command", Command: "echo 'test'"},
		},
	}

	err := remediation.CreatePlaybook(ctx, newPlaybook)
	if err != nil {
		t.Errorf("CreatePlaybook failed: %v", err)
	}
}

func TestAutoRemediation_UpdatePlaybook(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	playbook, _ := remediation.GetPlaybook(ctx, "high-cpu")
	playbook.Name = "Updated Name"

	err := remediation.UpdatePlaybook(ctx, playbook)
	if err != nil {
		t.Errorf("UpdatePlaybook failed: %v", err)
	}

	updated, _ := remediation.GetPlaybook(ctx, "high-cpu")
	if updated.Name != "Updated Name" {
		t.Error("Playbook not updated")
	}
}

func TestAutoRemediation_DeletePlaybook(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	err := remediation.DeletePlaybook(ctx, "high-cpu")
	if err != nil {
		t.Errorf("DeletePlaybook failed: %v", err)
	}

	_, err = remediation.GetPlaybook(ctx, "high-cpu")
	if err == nil {
		t.Error("Playbook still exists after deletion")
	}
}

func TestAutoRemediation_GetExecutionHistory(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	history, err := remediation.GetExecutionHistory(ctx, 10)
	if err != nil {
		t.Fatalf("GetExecutionHistory failed: %v", err)
	}

	if history == nil {
		t.Error("GetExecutionHistory returned nil")
	}
}

func TestAutoRemediation_GetTemplates(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	templates, err := remediation.GetTemplates(ctx)
	if err != nil {
		t.Fatalf("GetTemplates failed: %v", err)
	}

	if len(templates) == 0 {
		t.Error("GetTemplates returned no templates")
	}
}

func TestAutoRemediation_SetAutoExecution(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	err := remediation.SetAutoExecution(ctx, false)
	if err != nil {
		t.Errorf("SetAutoExecution failed: %v", err)
	}

	enabled, _ := remediation.IsAutoExecutionEnabled(ctx)
	if enabled {
		t.Error("Auto execution should be disabled")
	}
}

func TestAutoRemediation_ExecutePlaybook(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	record, err := remediation.ExecutePlaybook(ctx, "high-cpu", "test-alert")
	if err != nil {
		t.Fatalf("ExecutePlaybook failed: %v", err)
	}

	if record == nil {
		t.Fatal("ExecutePlaybook returned nil")
	}

	if record.Status != "completed" {
		t.Errorf("Expected completed status, got %s", record.Status)
	}
}

func TestAutoRemediation_GetExecutionStats(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	stats, err := remediation.GetExecutionStats(ctx)
	if err != nil {
		t.Fatalf("GetExecutionStats failed: %v", err)
	}

	if stats == nil {
		t.Fatal("GetExecutionStats returned nil")
	}

	if stats.TotalExecutions < 0 {
		t.Error("Invalid total executions count")
	}
}

func TestAutoRemediation_ExportPlaybooks(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	data, err := remediation.ExportPlaybooks(ctx, "json")
	if err != nil {
		t.Fatalf("ExportPlaybooks failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("ExportPlaybooks returned empty data")
	}
}

func TestAutoRemediation_GetPlaybookByCategory(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	playbooks, err := remediation.GetPlaybookByCategory(ctx, "performance")
	if err != nil {
		t.Fatalf("GetPlaybookByCategory failed: %v", err)
	}

	for _, playbook := range playbooks {
		if playbook.Category != "performance" {
			t.Errorf("Expected category performance, got %s", playbook.Category)
		}
	}
}

func TestAutoRemediation_GetEnabledPlaybooks(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	playbooks, err := remediation.GetEnabledPlaybooks(ctx)
	if err != nil {
		t.Fatalf("GetEnabledPlaybooks failed: %v", err)
	}

	for _, playbook := range playbooks {
		if !playbook.Enabled {
			t.Error("Expected enabled playbook")
		}
	}
}

func TestAutoRemediation_ValidatePlaybook(t *testing.T) {
	remediation := NewAutoRemediation()
	ctx := context.Background()

	validPlaybook := &RemediationPlaybook{
		Name:   "Valid Playbook",
		Steps:  []PlaybookStep{{ID: "step-1", Name: "Step", Type: "command"}},
		Timeout: 10,
	}

	errors, err := remediation.ValidatePlaybook(ctx, validPlaybook)
	if err != nil {
		t.Errorf("ValidatePlaybook failed: %v", err)
	}

	if len(errors) > 0 {
		t.Errorf("Expected no errors for valid playbook, got %v", errors)
	}

	invalidPlaybook := &RemediationPlaybook{
		Name:   "",
		Steps:  []PlaybookStep{},
		Timeout: -1,
	}

	errors, err = remediation.ValidatePlaybook(ctx, invalidPlaybook)
	if err != nil {
		t.Errorf("ValidatePlaybook failed: %v", err)
	}

	if len(errors) == 0 {
		t.Error("Expected errors for invalid playbook")
	}
}

func TestAssessRisk(t *testing.T) {
	remediation := NewAutoRemediation()

	tests := []struct {
		stepType string
		expected string
	}{
		{"restart", "high"},
		{"scale", "high"},
		{"delete", "high"},
		{"config", "medium"},
		{"update", "medium"},
		{"command", "low"},
		{"notify", "low"},
	}

	for _, tt := range tests {
		t.Run(tt.stepType, func(t *testing.T) {
			step := PlaybookStep{Type: tt.stepType}
			risk := remediation.assessRisk(step)
			if risk != tt.expected {
				t.Errorf("assessRisk(%s) = %s, want %s", tt.stepType, risk, tt.expected)
			}
		})
	}
}

func TestMatchesAlert(t *testing.T) {
	remediation := NewAutoRemediation()

	alert := Alert{
		ID:       "test-alert",
		Type:     "anomaly",
		Severity: "critical",
	}

	playbook := remediation.playbooks["high-cpu"]

	matched := remediation.matchesAlert(alert, playbook)

	if !matched {
		t.Error("Expected alert to match playbook")
	}
}

func TestConvertToFloat64(t *testing.T) {
	remediation := NewAutoRemediation()

	tests := []struct {
		input    interface{}
		expected float64
	}{
		{float64(5.5), 5.5},
		{int(10), 10.0},
		{int64(15), 15.0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		result := remediation.convertToFloat64(tt.input)
		if result != tt.expected {
			t.Errorf("convertToFloat64(%v) = %f, want %f", tt.input, result, tt.expected)
		}
	}
}
