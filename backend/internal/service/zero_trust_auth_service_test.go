package service

import (
	"context"
	"testing"
)

func TestContinuousAuthService_ValidateContinuousAuth(t *testing.T) {
	svc := NewContinuousAuthService()

	tests := []struct {
		name      string
		sessionID string
		riskScore float64
		wantValid bool
		wantErr   bool
	}{
		{
			name:      "low_risk_session",
			sessionID: "session-001",
			riskScore: 10.0,
			wantValid: true,
			wantErr:   false,
		},
		{
			name:      "medium_risk_session",
			sessionID: "session-002",
			riskScore: 50.0,
			wantValid: true,
			wantErr:   false,
		},
		{
			name:      "high_risk_session",
			sessionID: "session-003",
			riskScore: 65.0,
			wantValid: true,
			wantErr:   false,
		},
		{
			name:      "critical_risk_session",
			sessionID: "session-004",
			riskScore: 85.0,
			wantValid: false,
			wantErr:   false,
		},
		{
			name:      "invalid_risk_score_negative",
			sessionID: "session-005",
			riskScore: -10.0,
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "invalid_risk_score_over_max",
			sessionID: "session-006",
			riskScore: 150.0,
			wantValid: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := svc.ValidateContinuousAuth(ctx, tt.sessionID, tt.riskScore)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.SessionID != tt.sessionID {
				t.Errorf("SessionID mismatch: got %s, want %s", result.SessionID, tt.sessionID)
			}

			if result.IsValid != tt.wantValid {
				t.Errorf("IsValid mismatch: got %v, want %v", result.IsValid, tt.wantValid)
			}

			if result.RiskScore != tt.riskScore {
				t.Errorf("RiskScore mismatch: got %f, want %f", result.RiskScore, tt.riskScore)
			}
		})
	}
}

func TestContinuousAuthService_UpdateSessionRiskScore(t *testing.T) {
	svc := NewContinuousAuthService()
	ctx := context.Background()

	_, err := svc.ValidateContinuousAuth(ctx, "session-001", 20.0)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	err = svc.UpdateSessionRiskScore(ctx, "session-001", 50.0)
	if err != nil {
		t.Errorf("Unexpected error updating session risk: %v", err)
	}

	err = svc.UpdateSessionRiskScore(ctx, "nonexistent-session", 50.0)
	if err == nil {
		t.Errorf("Expected error for nonexistent session")
	}
}

func TestContinuousAuthService_RevokeSession(t *testing.T) {
	svc := NewContinuousAuthService()
	ctx := context.Background()

	_, err := svc.ValidateContinuousAuth(ctx, "session-001", 20.0)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	err = svc.RevokeSession(ctx, "session-001", "security_violation")
	if err != nil {
		t.Errorf("Unexpected error revoking session: %v", err)
	}

	err = svc.RevokeSession(ctx, "nonexistent-session", "test")
	if err == nil {
		t.Errorf("Expected error for nonexistent session")
	}
}

func TestContinuousAuthService_CheckThreatIntelligence(t *testing.T) {
	svc := NewContinuousAuthService()
	ctx := context.Background()

	result, err := svc.CheckThreatIntelligence(ctx, "192.0.2.0/24")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Indicator != "192.0.2.0/24" {
		t.Errorf("Expected indicator 192.0.2.0/24, got %s", result.Indicator)
	}

	if result.Reputation != "suspicious" {
		t.Errorf("Expected reputation 'suspicious', got %s", result.Reputation)
	}

	result, err = svc.CheckThreatIntelligence(ctx, "unknown-indicator")
	if err != nil {
		t.Errorf("Unexpected error for unknown indicator: %v", err)
	}

	if result.Reputation != "unknown" {
		t.Errorf("Expected reputation 'unknown' for unknown indicator, got %s", result.Reputation)
	}
}

func TestContinuousAuthService_EnforceLeastPrivilege(t *testing.T) {
	svc := NewContinuousAuthService()
	ctx := context.Background()

	permissions, err := svc.EnforceLeastPrivilege(ctx, 1, "user-data")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(permissions) != 1 || permissions[0] != "read" {
		t.Errorf("Expected default permission 'read', got %v", permissions)
	}
}

func TestContinuousAuthService_ReportSecurityIncident(t *testing.T) {
	svc := NewContinuousAuthService()
	ctx := context.Background()

	incident := &SecurityIncident{
		Type:        "unauthorized_access",
		Severity:    "high",
		Source:      "automated_detection",
		Description: "Unauthorized access attempt detected",
		Status:      "open",
	}

	err := svc.ReportSecurityIncident(ctx, incident)
	if err != nil {
		t.Errorf("Unexpected error reporting incident: %v", err)
	}

	err = svc.ReportSecurityIncident(ctx, nil)
	if err == nil {
		t.Errorf("Expected error for nil incident")
	}
}

func TestZeroTrustAuthService_CalculateRiskLevel(t *testing.T) {
	svc := &zeroTrustAuthService{}

	tests := []struct {
		score    float64
		expected string
	}{
		{0.0, "minimal"},
		{15.0, "minimal"},
		{20.0, "low"},
		{35.0, "low"},
		{40.0, "medium"},
		{55.0, "medium"},
		{60.0, "high"},
		{75.0, "high"},
		{80.0, "critical"},
		{95.0, "critical"},
	}

	for _, tt := range tests {
		result := svc.calculateRiskLevel(tt.score)
		if result != tt.expected {
			t.Errorf("For score %f, expected %s, got %s", tt.score, tt.expected, result)
		}
	}
}
