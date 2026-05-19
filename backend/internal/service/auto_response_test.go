package service

import (
	"context"
	"testing"
	"time"
)

func TestAutoResponseService_ProcessThreat(t *testing.T) {
	service := NewAutoResponseService()
	ctx := context.Background()

	tests := []struct {
		name         string
		threat       *ThreatContext
		wantExecuted bool
	}{
		{
			name: "Normal threat",
			threat: &ThreatContext{
				IP:          "192.0.2.1",
				ThreatLevel: 1,
				ThreatTypes: []string{"low"},
				Confidence:  0.3,
			},
			wantExecuted: false,
		},
		{
			name: "Critical threat",
			threat: &ThreatContext{
				IP:          "192.0.2.2",
				ThreatLevel: 5,
				ThreatTypes: []string{"sql_injection"},
				Confidence:  0.9,
			},
			wantExecuted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ProcessThreat(ctx, tt.threat)
			if err != nil {
				t.Errorf("ProcessThreat() error = %v", err)
				return
			}

			if tt.wantExecuted && result == nil {
				t.Error("ProcessThreat() returned nil for threat that should execute")
			}
		})
	}
}

func TestAutoResponseService_GetResponseRules(t *testing.T) {
	service := NewAutoResponseService()

	rules := service.GetResponseRules()
	if len(rules) == 0 {
		t.Error("GetResponseRules() returned empty list")
	}
}

func TestAutoResponseService_CreateResponseRule(t *testing.T) {
	service := NewAutoResponseService()

	initialCount := len(service.responseRules)

	rule := &ResponseRule{
		Name:      "Test Rule",
		Priority: 50,
		IsActive: true,
		TriggerCondition: &TriggerCondition{
			ThreatLevel: []int{3},
		},
		Action: ResponseAction{
			Type:     ActionTypeBlockIP,
			Duration: 1 * time.Hour,
		},
	}

	err := service.CreateResponseRule(rule)
	if err != nil {
		t.Errorf("CreateResponseRule() error = %v", err)
	}

	if len(service.responseRules) != initialCount+1 {
		t.Errorf("CreateResponseRule() did not add rule")
	}
}

func TestAutoResponseService_UpdateResponseRule(t *testing.T) {
	service := NewAutoResponseService()

	for id := range service.responseRules {
		updates := &ResponseRule{
			Name:     "Updated Rule Name",
			Priority: 100,
			IsActive: false,
		}

		err := service.UpdateResponseRule(id, updates)
		if err != nil {
			t.Errorf("UpdateResponseRule() error = %v", err)
		}
		break
	}
}

func TestAutoResponseService_DeleteResponseRule(t *testing.T) {
	service := NewAutoResponseService()

	var ruleID string
	for id := range service.responseRules {
		ruleID = id
		break
	}

	if ruleID == "" {
		t.Skip("No rules available")
	}

	err := service.DeleteResponseRule(ruleID)
	if err != nil {
		t.Errorf("DeleteResponseRule() error = %v", err)
	}
}

func TestAutoResponseService_GetActiveActions(t *testing.T) {
	service := NewAutoResponseService()

	actions := service.GetActiveActions()
	if actions == nil {
		t.Error("GetActiveActions() returned nil")
	}
}

func TestAutoResponseService_GetActionHistory(t *testing.T) {
	service := NewAutoResponseService()

	history, err := service.GetActionHistory(nil)
	if err != nil {
		t.Errorf("GetActionHistory() error = %v", err)
	}

	if history == nil {
		t.Error("GetActionHistory() returned nil")
	}
}

func TestAutoResponseService_AddNotificationChannel(t *testing.T) {
	service := NewAutoResponseService()

	initialCount := len(service.notificationChannels)

	channel := &NotificationChannel{
		Type:     NotificationTypeSlack,
		Endpoint: "https://hooks.slack.com/test",
	}

	err := service.AddNotificationChannel(channel)
	if err != nil {
		t.Errorf("AddNotificationChannel() error = %v", err)
	}

	if len(service.notificationChannels) != initialCount+1 {
		t.Errorf("AddNotificationChannel() did not add channel")
	}
}

func TestAutoResponseService_GetNotificationChannels(t *testing.T) {
	service := NewAutoResponseService()

	channels := service.GetNotificationChannels()
	if len(channels) == 0 {
		t.Error("GetNotificationChannels() returned empty list")
	}
}

func TestAutoResponseService_CreateContainmentPolicy(t *testing.T) {
	service := NewAutoResponseService()

	initialCount := len(service.containmentPolicies)

	policy := &ContainmentPolicy{
		Name:       "Test Policy",
		AttackType: "ddos",
		Priority:   50,
		IsActive:  true,
		Actions: []ResponseAction{
			{Type: ActionTypeRateLimit},
		},
	}

	err := service.CreateContainmentPolicy(policy)
	if err != nil {
		t.Errorf("CreateContainmentPolicy() error = %v", err)
	}

	if len(service.containmentPolicies) != initialCount+1 {
		t.Errorf("CreateContainmentPolicy() did not add policy")
	}
}

func TestAutoResponseService_GetContainmentPolicies(t *testing.T) {
	service := NewAutoResponseService()

	policies := service.GetContainmentPolicies()
	if len(policies) == 0 {
		t.Error("GetContainmentPolicies() returned empty list")
	}
}

func TestAutoResponseService_EnableDisable(t *testing.T) {
	service := NewAutoResponseService()

	service.Disable()
	if service.IsEnabled() {
		t.Error("Disable() did not disable service")
	}

	service.Enable()
	if !service.IsEnabled() {
		t.Error("Enable() did not enable service")
	}
}

func TestAutoResponseService_GetResponseStatistics(t *testing.T) {
	service := NewAutoResponseService()

	stats := service.GetResponseStatistics()

	if stats == nil {
		t.Error("GetResponseStatistics() returned nil")
	}

	if stats["total_rules"] == nil {
		t.Error("GetResponseStatistics() missing total_rules")
	}

	if stats["is_enabled"] == nil {
		t.Error("GetResponseStatistics() missing is_enabled")
	}
}

func TestAutoResponseService_TestResponseAction(t *testing.T) {
	service := NewAutoResponseService()
	ctx := context.Background()

	target := &ActionTarget{
		IP: "192.0.2.100",
	}

	result, err := service.TestResponseAction(ctx, ActionTypeBlockIP, target)
	if err != nil {
		t.Errorf("TestResponseAction() error = %v", err)
	}

	if result == nil {
		t.Error("TestResponseAction() returned nil")
	}
}

func TestAutoResponseService_CancelAction(t *testing.T) {
	service := NewAutoResponseService()

	err := service.CancelAction("non-existent-action")
	if err == nil {
		t.Error("CancelAction() should return error for non-existent action")
	}
}

func TestAutoResponseService_AnalyzeResponseEffectiveness(t *testing.T) {
	service := NewAutoResponseService()

	report := service.AnalyzeResponseEffectiveness()

	if report == nil {
		t.Error("AnalyzeResponseEffectiveness() returned nil")
	}

	if report.GeneratedAt.IsZero() {
		t.Error("AnalyzeResponseEffectiveness() did not set GeneratedAt")
	}
}

func TestAutoResponseService_ExportImport(t *testing.T) {
	service := NewAutoResponseService()

	data, err := service.ExportConfiguration()
	if err != nil {
		t.Errorf("ExportConfiguration() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("ExportConfiguration() returned empty data")
	}

	newService := NewAutoResponseService()
	err = newService.ImportConfiguration(data)
	if err != nil {
		t.Errorf("ImportConfiguration() error = %v", err)
	}
}

func TestAutoResponseService_CreateAutomationWorkflow(t *testing.T) {
	service := NewAutoResponseService()

	workflow := &AutomationWorkflow{
		Name:    "Test Workflow",
		Trigger: &WorkflowTrigger{Type: "threat_level", ThreatLevel: 3},
		Steps: []WorkflowStep{
			{
				Order: 1,
				Action: ResponseAction{
					Type: ActionTypeNotify,
				},
			},
		},
	}

	err := service.CreateAutomationWorkflow(workflow)
	if err != nil {
		t.Errorf("CreateAutomationWorkflow() error = %v", err)
	}
}
