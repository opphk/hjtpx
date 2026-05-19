package service

import (
	"context"
	"testing"
	"time"
)

func TestSOARService_TriggerPlaybook(t *testing.T) {
	svc := NewSOARService()

	tests := []struct {
		name      string
		playbookID string
		trigger   *PlaybookTrigger
		wantErr   bool
	}{
		{
			name:      "phishing_response_playbook",
			playbookID: "phishing-response",
			trigger: &PlaybookTrigger{
				IncidentID:  "inc-001",
				TriggerType: "email_phishing",
				Severity:    "high",
				Source:      "email_gateway",
			},
			wantErr: false,
		},
		{
			name:      "credential_compromise_playbook",
			playbookID: "credential-compromise",
			trigger: &PlaybookTrigger{
				IncidentID:  "inc-002",
				TriggerType: "credential_compromise",
				Severity:    "critical",
				Source:      "authentication_system",
			},
			wantErr: false,
		},
		{
			name:      "ddos_mitigation_playbook",
			playbookID: "ddos-mitigation",
			trigger: &PlaybookTrigger{
				IncidentID:  "inc-003",
				TriggerType: "ddos_detected",
				Severity:    "high",
				Source:      "ddos_protection",
			},
			wantErr: false,
		},
		{
			name:      "nonexistent_playbook",
			playbookID: "nonexistent",
			trigger: &PlaybookTrigger{
				IncidentID: "inc-004",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			execution, err := svc.TriggerPlaybook(ctx, tt.playbookID, tt.trigger)

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

			if execution == nil {
				t.Errorf("Expected execution but got nil")
				return
			}

			if execution.PlaybookID != tt.playbookID {
				t.Errorf("PlaybookID mismatch: got %s, want %s", execution.PlaybookID, tt.playbookID)
			}

			if execution.ExecutionID == "" {
				t.Error("Expected non-empty execution ID")
			}
		})
	}
}

func TestSOARService_GetPlaybook(t *testing.T) {
	svc := NewSOARService()

	tests := []struct {
		name      string
		playbookID string
		wantErr   bool
	}{
		{
			name:      "existing_playbook",
			playbookID: "phishing-response",
			wantErr:   false,
		},
		{
			name:      "nonexistent_playbook",
			playbookID: "nonexistent",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			playbook, err := svc.GetPlaybook(ctx, tt.playbookID)

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

			if playbook == nil {
				t.Errorf("Expected playbook but got nil")
				return
			}

			if playbook.ID != tt.playbookID {
				t.Errorf("ID mismatch: got %s, want %s", playbook.ID, tt.playbookID)
			}

			if !playbook.Enabled {
				t.Error("Playbook should be enabled")
			}

			if len(playbook.Steps) == 0 {
				t.Error("Playbook should have steps")
			}
		})
	}
}

func TestSOARService_ListPlaybooks(t *testing.T) {
	svc := NewSOARService()
	ctx := context.Background()

	playbooks, err := svc.ListPlaybooks(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if len(playbooks) == 0 {
		t.Error("Expected playbooks but got none")
	}

	if len(playbooks) < 3 {
		t.Errorf("Expected at least 3 playbooks, got %d", len(playbooks))
	}

	for _, playbook := range playbooks {
		if playbook.ID == "" {
			t.Error("Playbook ID should not be empty")
		}
		if playbook.Name == "" {
			t.Error("Playbook name should not be empty")
		}
	}
}

func TestSOARService_ExecuteThreatHunt(t *testing.T) {
	svc := NewSOARService()

	tests := []struct {
		name   string
		hunt   *ThreatHuntRequest
		wantErr bool
	}{
		{
			name: "basic_threat_hunt",
			hunt: &ThreatHuntRequest{
				HuntID:     "hunt-001",
				Hypothesis: "Potential C2 communication detected",
				Indicators: []string{"192.0.2.1", "malware.exe"},
				DataSources: []string{"network_logs", "endpoint_logs"},
				StartTime:  time.Now().Add(-24 * time.Hour),
				EndTime:    time.Now(),
			},
			wantErr: false,
		},
		{
			name: "threat_hunt_with_ip",
			hunt: &ThreatHuntRequest{
				HuntID:     "hunt-002",
				Hypothesis: "Suspicious outbound connections",
				Indicators: []string{"203.0.113.50"},
				DataSources: []string{"firewall_logs"},
				StartTime:  time.Now().Add(-48 * time.Hour),
				EndTime:    time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := svc.ExecuteThreatHunt(ctx, tt.hunt)

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

			if result == nil {
				t.Errorf("Expected result but got nil")
				return
			}

			if result.HuntID != tt.hunt.HuntID {
				t.Errorf("HuntID mismatch: got %s, want %s", result.HuntID, tt.hunt.HuntID)
			}

			if result.Status == "" {
				t.Error("Expected non-empty status")
			}
		})
	}
}

func TestSOARService_GetSecurityPosture(t *testing.T) {
	svc := NewSOARService()
	ctx := context.Background()

	posture, err := svc.GetSecurityPosture(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if posture == nil {
		t.Errorf("Expected posture but got nil")
		return
	}

	if posture.OverallScore < 0 || posture.OverallScore > 100 {
		t.Errorf("Invalid overall score: %f", posture.OverallScore)
	}

	if posture.MTTR < 0 {
		t.Errorf("Invalid MTTR: %v", posture.MTTR)
	}

	if posture.MTTD < 0 {
		t.Errorf("Invalid MTTD: %v", posture.MTTD)
	}

	if len(posture.CategoryScores) == 0 {
		t.Error("Expected category scores but got none")
	}
}

func TestSOARService_AutoRespond(t *testing.T) {
	svc := NewSOARService()

	tests := []struct {
		name     string
		incident *SecurityIncident
		wantErr  bool
	}{
		{
			name: "critical_incident",
			incident: &SecurityIncident{
				ID:          1,
				Type:        "unauthorized_access",
				Severity:    "critical",
				Source:      "ids",
				Description: "Critical security incident",
				Status:      "open",
				CreatedAt:   time.Now().Add(-5 * time.Minute),
			},
			wantErr: false,
		},
		{
			name: "high_incident",
			incident: &SecurityIncident{
				ID:          2,
				Type:        "malware_detection",
				Severity:    "high",
				Source:      "antivirus",
				Description: "Malware detected",
				Status:      "open",
				CreatedAt:   time.Now().Add(-10 * time.Minute),
			},
			wantErr: false,
		},
		{
			name: "medium_incident",
			incident: &SecurityIncident{
				ID:          3,
				Type:        "policy_violation",
				Severity:    "medium",
				Source:      "dlp",
				Description: "Policy violation detected",
				Status:      "open",
				CreatedAt:   time.Now().Add(-30 * time.Minute),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := svc.AutoRespond(ctx, tt.incident)

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

			if response == nil {
				t.Errorf("Expected response but got nil")
				return
			}

			if response.ResponseID == "" {
				t.Error("Expected non-empty response ID")
			}

			if response.IncidentID == "" {
				t.Error("Expected non-empty incident ID")
			}

			if len(response.ActionsTaken) == 0 {
				t.Error("Expected actions to be taken")
			}
		})
	}
}
