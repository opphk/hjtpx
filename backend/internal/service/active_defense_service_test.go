package service

import (
	"context"
	"testing"
	"time"
)

func TestActiveDefenseService_CreateHoneypot(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	config := &HoneypotConfig{
		HoneypotID:      "hp-test-001",
		Name:            "Test Honeypot",
		Type:            "high_interaction",
		Services:        []string{"http", "ssh"},
		Vulnerabilities: []string{"weak_credentials"},
		IsActive:        true,
	}

	honeypot, err := service.CreateHoneypot(ctx, config)
	if err != nil {
		t.Fatalf("CreateHoneypot failed: %v", err)
	}

	if honeypot == nil {
		t.Fatal("Honeypot should not be nil")
	}

	if honeypot.Name != config.Name {
		t.Errorf("Name mismatch: expected %s, got %s", config.Name, honeypot.Name)
	}
}

func TestActiveDefenseService_RecordInteraction(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	honeypot, err := service.CreateHoneypot(ctx, &HoneypotConfig{
		HoneypotID: "hp-test-002",
		Name:       "Test Honeypot",
	})
	if err != nil {
		t.Fatalf("CreateHoneypot failed: %v", err)
	}

	interaction := &HoneypotInteraction{
		HoneypotID: honeypot.HoneypotID,
		SourceIP:   "192.0.2.0",
		Action:     "brute_force",
		Target:     "/login",
	}

	err = service.RecordInteraction(ctx, interaction)
	if err != nil {
		t.Fatalf("RecordInteraction failed: %v", err)
	}

	if interaction.InteractionID == "" {
		t.Error("InteractionID should be generated")
	}
}

func TestActiveDefenseService_MonitorInteraction(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	honeypot, err := service.CreateHoneypot(ctx, &HoneypotConfig{
		HoneypotID: "hp-test-003",
		Name:       "Test Honeypot",
	})
	if err != nil {
		t.Fatalf("CreateHoneypot failed: %v", err)
	}

	interactions := []*HoneypotInteraction{
		{HoneypotID: honeypot.HoneypotID, SourceIP: "192.0.2.1", Action: "login_attempt", Authenticated: true},
		{HoneypotID: honeypot.HoneypotID, SourceIP: "192.0.2.2", Action: "sql_injection"},
	}

	for _, interaction := range interactions {
		err := service.RecordInteraction(ctx, interaction)
		if err != nil {
			t.Fatalf("RecordInteraction failed: %v", err)
		}
	}

	metrics, err := service.MonitorInteraction(ctx, honeypot.HoneypotID)
	if err != nil {
		t.Fatalf("MonitorInteraction failed: %v", err)
	}

	if metrics.TotalInteractions != 2 {
		t.Errorf("TotalInteractions mismatch: expected 2, got %d", metrics.TotalInteractions)
	}
}

func TestActiveDefenseService_GenerateDeceptiveElements(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	config := &DeceptionConfig{
		ConfigID:       "dec-config-001",
		DeceptionType: "credential",
		Density:        0.5,
	}

	elements, err := service.GenerateDeceptiveElements(ctx, config)
	if err != nil {
		t.Fatalf("GenerateDeceptiveElements failed: %v", err)
	}

	if len(elements) == 0 {
		t.Error("Should generate deceptive elements")
	}
}

func TestActiveDefenseService_CreateCanaryResource(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	resource := &CanaryResource{
		ResourceID: "canary-001",
		Name:       "Fake API Key",
		Type:       "api_key",
		Path:       "/etc/secrets/fake_key",
	}

	err := service.CreateCanaryResource(ctx, resource)
	if err != nil {
		t.Fatalf("CreateCanaryResource failed: %v", err)
	}

	if !resource.IsActive {
		t.Error("Resource should be active")
	}
}

func TestActiveDefenseService_AnalyzeAttackPaths(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	target := &AttackTarget{
		TargetID:       "target-001",
		Type:          "web_application",
		Components:    []string{"frontend", "backend", "database"},
		SecurityLevel: "medium",
		Exposure:      0.7,
	}

	paths, err := service.AnalyzeAttackPaths(ctx, target)
	if err != nil {
		t.Fatalf("AnalyzeAttackPaths failed: %v", err)
	}

	if len(paths) == 0 {
		t.Error("Should predict at least one attack path")
	}

	path := paths[0]
	if path.TotalRiskScore <= 0 || path.TotalRiskScore > 100 {
		t.Errorf("TotalRiskScore out of range: %f", path.TotalRiskScore)
	}
}

func TestActiveDefenseService_DetectThreat(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	threat := &Threat{
		ThreatID: "threat-001",
		Type:     "sql_injection",
		Severity: "high",
		SourceIP: "192.0.2.0",
		Count:    15,
	}

	analysis, err := service.DetectThreat(ctx, threat)
	if err != nil {
		t.Fatalf("DetectThreat failed: %v", err)
	}

	if analysis == nil {
		t.Fatal("Analysis should not be nil")
	}

	if analysis.RiskScore <= 0 || analysis.RiskScore > 100 {
		t.Errorf("RiskScore out of range: %f", analysis.RiskScore)
	}
}

func TestActiveDefenseService_ExecuteCountermeasure(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	action := &CountermeasureAction{
		ThreatID:   "threat-002",
		ActionType: "block_ip",
		Target:     "192.0.2.0",
	}

	result, err := service.ExecuteCountermeasure(ctx, action)
	if err != nil {
		t.Fatalf("ExecuteCountermeasure failed: %v", err)
	}

	if !result.Success {
		t.Error("Countermeasure should succeed")
	}
}

func TestActiveDefenseService_InvalidInputs(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	_, err := service.CreateHoneypot(ctx, nil)
	if err == nil {
		t.Error("CreateHoneypot should fail with nil config")
	}

	err = service.ActivateHoneypot(ctx, "non-existent")
	if err == nil {
		t.Error("ActivateHoneypot should fail with non-existent honeypot")
	}

	_, err = service.MonitorInteraction(ctx, "non-existent")
	if err == nil {
		t.Error("MonitorInteraction should fail with non-existent honeypot")
	}
}
