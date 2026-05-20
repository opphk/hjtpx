package service

import (
	"context"
	"testing"
	"time"
)

func TestZTAEnhancedService_StartContinuousValidation(t *testing.T) {
	service := NewZTAEnhancedService()
	ctx := context.Background()

	config := &ValidationConfig{
		SessionID:            "test-session-001",
		ValidationInterval:   5 * time.Minute,
		RiskThresholds:       RiskThresholds{Low: 20, Medium: 40, High: 60, Critical: 80},
		EnabledFactors:       []string{"device", "location", "behavior"},
		MaxGracePeriod:       30 * time.Minute,
		AutoRevokeOnHighRisk: true,
	}

	session, err := service.StartContinuousValidation(ctx, config.SessionID, config)
	if err != nil {
		t.Fatalf("StartContinuousValidation failed: %v", err)
	}

	if session == nil {
		t.Fatal("Session should not be nil")
	}

	if session.SessionID != config.SessionID {
		t.Errorf("SessionID mismatch: expected %s, got %s", config.SessionID, session.SessionID)
	}

	if session.Status != "active" {
		t.Errorf("Status mismatch: expected active, got %s", session.Status)
	}
}

func TestZTAEnhancedService_ValidateAccessRequest(t *testing.T) {
	service := NewZTAEnhancedService()
	ctx := context.Background()

	sessionID := "test-session-002"
	_, err := service.StartContinuousValidation(ctx, sessionID, nil)
	if err != nil {
		t.Fatalf("StartContinuousValidation failed: %v", err)
	}

	request := &AccessRequest{
		RequestID: "req-001",
		UserID:    1,
		SessionID: sessionID,
		Resource:  "/api/admin/data",
		Action:    "read",
		SourceIP:  "192.168.1.100",
		UserAgent: "Mozilla/5.0",
	}

	decision, err := service.ValidateAccessRequest(ctx, request)
	if err != nil {
		t.Fatalf("ValidateAccessRequest failed: %v", err)
	}

	if decision == nil {
		t.Fatal("Decision should not be nil")
	}

	if decision.Decision == "" {
		t.Error("Decision should not be empty")
	}
}

func TestZTAEnhancedService_GetRiskScore(t *testing.T) {
	service := NewZTAEnhancedService()
	ctx := context.Background()

	sessionID := "test-session-003"
	_, err := service.StartContinuousValidation(ctx, sessionID, nil)
	if err != nil {
		t.Fatalf("StartContinuousValidation failed: %v", err)
	}

	result, err := service.GetRiskScore(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetRiskScore failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.TotalScore < 0 || result.TotalScore > 100 {
		t.Errorf("TotalScore out of range: %f", result.TotalScore)
	}
}

func TestZTAEnhancedService_CreateMicrosegment(t *testing.T) {
	service := NewZTAEnhancedService()
	ctx := context.Background()

	definition := &MicrosegmentDefinition{
		SegmentID:      "seg-001",
		Name:          "web-to-db",
		SourceWorkload: "web-server-01",
		DestWorkload:   "db-server-01",
		Port:           3306,
		Protocol:       "tcp",
		AllowedUsers:   []string{"admin", "app-service"},
	}

	segment, err := service.CreateMicrosegment(ctx, definition)
	if err != nil {
		t.Fatalf("CreateMicrosegment failed: %v", err)
	}

	if segment == nil {
		t.Fatal("Segment should not be nil")
	}

	if !segment.IsActive {
		t.Error("Segment should be active")
	}
}

func TestZTAEnhancedService_ValidateTraffic(t *testing.T) {
	service := NewZTAEnhancedService()
	ctx := context.Background()

	definition := &MicrosegmentDefinition{
		SegmentID:      "seg-002",
		Name:          "allow-segment",
		SourceWorkload: "allowed-client",
		DestWorkload:   "allowed-server",
		Port:           8080,
		Protocol:       "tcp",
	}

	_, err := service.CreateMicrosegment(ctx, definition)
	if err != nil {
		t.Fatalf("CreateMicrosegment failed: %v", err)
	}

	decision, err := service.ValidateTraffic(ctx, "allowed-client", "allowed-server", "tcp")
	if err != nil {
		t.Fatalf("ValidateTraffic failed: %v", err)
	}

	if !decision.Allowed {
		t.Error("Traffic should be allowed")
	}
}

func TestZTAEnhancedService_ComputePermissions(t *testing.T) {
	service := NewZTAEnhancedService()
	ctx := context.Background()

	permCtx := &PermissionContext{
		TimeOfDay:   time.Now(),
		DayOfWeek:   1,
		RiskScore:   20,
		AuthMethods: []string{"password", "totp"},
	}

	computed, err := service.ComputePermissions(ctx, 1, "/api/data", permCtx)
	if err != nil {
		t.Fatalf("ComputePermissions failed: %v", err)
	}

	if computed == nil {
		t.Fatal("Computed permissions should not be nil")
	}

	if len(computed.Permissions) == 0 {
		t.Error("Permissions should not be empty")
	}

	if computed.Confidence <= 0 || computed.Confidence > 1 {
		t.Errorf("Confidence out of range: %f", computed.Confidence)
	}
}

func TestZTAEnhancedService_GetNearestEdge(t *testing.T) {
	service := NewZTAEnhancedService()
	ctx := context.Background()

	coordinates := &Coordinates{
		Latitude:  40.7128,
		Longitude: -74.0060,
	}

	edge, err := service.GetNearestEdge(ctx, coordinates)
	if err != nil {
		t.Fatalf("GetNearestEdge failed: %v", err)
	}

	if edge == nil {
		t.Fatal("Edge should not be nil")
	}

	if edge.EdgeID == "" {
		t.Error("EdgeID should not be empty")
	}
}

func TestZTAEnhancedService_InvalidInputs(t *testing.T) {
	service := NewZTAEnhancedService()
	ctx := context.Background()

	_, err := service.StopContinuousValidation(ctx, "non-existent-session")
	if err == nil {
		t.Error("StopContinuousValidation should fail with non-existent session")
	}

	_, err = service.GetValidationStatus(ctx, "non-existent-session")
	if err == nil {
		t.Error("GetValidationStatus should fail with non-existent session")
	}

	err = service.RestoreWorkload(ctx, "non-existent-workload")
	if err == nil {
		t.Error("RestoreWorkload should fail with non-existent workload")
	}
}
