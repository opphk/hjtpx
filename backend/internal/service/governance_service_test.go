package service

import (
	"context"
	"testing"
	"time"
)

func TestGovernanceService_GetGovernanceDashboard(t *testing.T) {
	svc := NewGovernanceService()
	ctx := context.Background()

	dashboard, err := svc.GetGovernanceDashboard(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if dashboard == nil {
		t.Errorf("Expected dashboard but got nil")
		return
	}

	if dashboard.DashboardID == "" {
		t.Error("Expected non-empty dashboard ID")
	}

	if dashboard.OverallHealth == "" {
		t.Error("Expected non-empty overall health")
	}

	if dashboard.Metrics == nil {
		t.Error("Expected metrics but got nil")
	}

	if len(dashboard.Alerts) == 0 {
		t.Error("Expected alerts but got none")
	}

	for _, alert := range dashboard.Alerts {
		if alert.AlertID == "" {
			t.Error("Alert ID should not be empty")
		}
		if alert.Severity == "" {
			t.Error("Alert severity should not be empty")
		}
	}
}

func TestGovernanceService_GetRealTimeCompliance(t *testing.T) {
	svc := NewGovernanceService()
	ctx := context.Background()

	compliance, err := svc.GetRealTimeCompliance(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if compliance == nil {
		t.Errorf("Expected compliance but got nil")
		return
	}

	if compliance.ComplianceRate < 0 || compliance.ComplianceRate > 100 {
		t.Errorf("Invalid compliance rate: %f", compliance.ComplianceRate)
	}

	if len(compliance.Controls) == 0 {
		t.Error("Expected controls but got none")
	}

	for _, control := range compliance.Controls {
		if control.ControlID == "" {
			t.Error("Control ID should not be empty")
		}
		if control.Status == "" {
			t.Error("Control status should not be empty")
		}
	}
}

func TestGovernanceService_GenerateComplianceReport(t *testing.T) {
	svc := NewGovernanceService()

	tests := []struct {
		name   string
		req    *ComplianceReportRequest
		wantErr bool
	}{
		{
			name: "ccpa_report",
			req: &ComplianceReportRequest{
				Framework:  "ccpa",
				Period:     "Q1-2024",
				StartDate:  time.Now().Add(-90 * 24 * time.Hour),
				EndDate:    time.Now(),
				Format:     "json",
				IncludeRaw: false,
			},
			wantErr: false,
		},
		{
			name: "pipl_report_with_raw",
			req: &ComplianceReportRequest{
				Framework:  "pipl",
				Period:     "2024",
				StartDate:  time.Now().Add(-365 * 24 * time.Hour),
				EndDate:    time.Now(),
				Format:     "json",
				IncludeRaw: true,
			},
			wantErr: false,
		},
		{
			name: "lgpd_report",
			req: &ComplianceReportRequest{
				Framework:  "lgpd",
				Period:     "January-2024",
				StartDate:  time.Now().Add(-31 * 24 * time.Hour),
				EndDate:    time.Now(),
				Format:     "pdf",
				IncludeRaw: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			report, err := svc.GenerateComplianceReport(ctx, tt.req)

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

			if report == nil {
				t.Errorf("Expected report but got nil")
				return
			}

			if report.ReportID == "" {
				t.Error("Expected non-empty report ID")
			}

			if report.ComplianceScore < 0 || report.ComplianceScore > 100 {
				t.Errorf("Invalid compliance score: %f", report.ComplianceScore)
			}
		})
	}
}

func TestGovernanceService_CalculateRiskScore(t *testing.T) {
	svc := NewGovernanceService()

	tests := []struct {
		name       string
		entityID   string
		entityType string
		wantErr    bool
	}{
		{
			name:       "user_risk_score",
			entityID:   "user-001",
			entityType: "user",
			wantErr:    false,
		},
		{
			name:       "application_risk_score",
			entityID:   "app-001",
			entityType: "application",
			wantErr:    false,
		},
		{
			name:       "infrastructure_risk_score",
			entityID:   "infra-001",
			entityType: "infrastructure",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			score, err := svc.CalculateRiskScore(ctx, tt.entityID, tt.entityType)

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

			if score == nil {
				t.Errorf("Expected score but got nil")
				return
			}

			if score.EntityID != tt.entityID {
				t.Errorf("EntityID mismatch: got %s, want %s", score.EntityID, tt.entityID)
			}

			if score.OverallScore < 0 || score.OverallScore > 100 {
				t.Errorf("Invalid overall score: %f", score.OverallScore)
			}

			if len(score.Factors) == 0 {
				t.Error("Expected risk factors but got none")
			}

			totalWeight := 0.0
			for _, factor := range score.Factors {
				totalWeight += factor.Weight
			}

			if totalWeight < 0.99 || totalWeight > 1.01 {
				t.Errorf("Factor weights should sum to ~1.0, got %f", totalWeight)
			}
		})
	}
}

func TestGovernanceService_GetComplianceMetrics(t *testing.T) {
	svc := NewGovernanceService()

	tests := []struct {
		name     string
		framework string
		wantErr  bool
	}{
		{
			name:      "ccpa_metrics",
			framework: "ccpa",
			wantErr:   false,
		},
		{
			name:      "pipl_metrics",
			framework: "pipl",
			wantErr:   false,
		},
		{
			name:      "lgpd_metrics",
			framework: "lgpd",
			wantErr:   false,
		},
		{
			name:      "gdpr_metrics",
			framework: "gdpr",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			metrics, err := svc.GetComplianceMetrics(ctx, tt.framework)

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

			if metrics == nil {
				t.Errorf("Expected metrics but got nil")
				return
			}

			if metrics.Framework != tt.framework {
				t.Errorf("Framework mismatch: got %s, want %s", metrics.Framework, tt.framework)
			}

			if metrics.ComplianceRate < 0 || metrics.ComplianceRate > 100 {
				t.Errorf("Invalid compliance rate: %f", metrics.ComplianceRate)
			}

			if metrics.TotalControls != metrics.PassedControls+metrics.FailedControls {
				t.Error("Total controls should equal sum of passed and failed")
			}

			if len(metrics.ByCategory) == 0 {
				t.Error("Expected category scores but got none")
			}
		})
	}
}

func TestGovernanceService_MonitorComplianceStatus(t *testing.T) {
	svc := NewGovernanceService()
	ctx := context.Background()

	statuses, err := svc.MonitorComplianceStatus(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if len(statuses) == 0 {
		t.Error("Expected compliance statuses but got none")
	}

	for _, status := range statuses {
		if status.Regulation == "" {
			t.Error("Regulation should not be empty")
		}

		if status.Score < 0 || status.Score > 100 {
			t.Errorf("Invalid score for %s: %f", status.Regulation, status.Score)
		}

		if status.Owner == "" {
			t.Error("Owner should not be empty")
		}
	}
}
