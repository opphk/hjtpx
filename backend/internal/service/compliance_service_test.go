package service

import (
	"context"
	"testing"
	"time"
)

func TestComplianceService_CheckCompliance(t *testing.T) {
	svc := NewComplianceService()

	tests := []struct {
		name     string
		framework ComplianceFramework
		data     *ComplianceCheckData
		wantErr  bool
	}{
		{
			name:     "ccpa_compliance_check",
			framework: FrameworkCCPA,
			data: &ComplianceCheckData{
				UserID:           1,
				DataTypes:        []string{"name", "email", "phone"},
				ProcessingPurpose: "service_delivery",
				ConsentObtained:  true,
				Jurisdiction:     "California",
			},
			wantErr: false,
		},
		{
			name:     "pipl_compliance_check",
			framework: FrameworkPIPL,
			data: &ComplianceCheckData{
				UserID:           2,
				DataTypes:        []string{"name", "id_number"},
				ProcessingPurpose: "user_authentication",
				ConsentObtained:  true,
				Jurisdiction:     "CN",
			},
			wantErr: false,
		},
		{
			name:     "lgpd_compliance_check",
			framework: FrameworkLGPD,
			data: &ComplianceCheckData{
				UserID:           3,
				DataTypes:        []string{"email", "purchase_history"},
				ProcessingPurpose: "contract_execution",
				LegalBasis:       "contract",
				ConsentObtained:  true,
				Jurisdiction:     "Brazil",
			},
			wantErr: false,
		},
		{
			name:     "unsupported_framework",
			framework: "unsupported",
			data: &ComplianceCheckData{
				UserID: 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			report, err := svc.CheckCompliance(ctx, tt.framework, tt.data)

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

			if report.Framework != tt.framework {
				t.Errorf("Framework mismatch: got %s, want %s", report.Framework, tt.framework)
			}

			if report.ComplianceScore < 0 || report.ComplianceScore > 100 {
				t.Errorf("Invalid compliance score: %f", report.ComplianceScore)
			}
		})
	}
}

func TestComplianceService_GetDataSubjectRights(t *testing.T) {
	svc := NewComplianceService()

	tests := []struct {
		name      string
		framework ComplianceFramework
		userID    uint
		wantRights int
		wantErr   bool
	}{
		{
			name:       "ccpa_data_subject_rights",
			framework:  FrameworkCCPA,
			userID:     1,
			wantRights: 3,
			wantErr:    false,
		},
		{
			name:       "pipl_data_subject_rights",
			framework:  FrameworkPIPL,
			userID:     2,
			wantRights: 4,
			wantErr:    false,
		},
		{
			name:       "lgpd_data_subject_rights",
			framework:  FrameworkLGPD,
			userID:     3,
			wantRights: 4,
			wantErr:    false,
		},
		{
			name:      "unsupported_framework",
			framework: "unknown",
			userID:    1,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			rights, err := svc.GetDataSubjectRights(ctx, tt.framework, tt.userID)

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

			if len(rights.Rights) != tt.wantRights {
				t.Errorf("Expected %d rights, got %d", tt.wantRights, len(rights.Rights))
			}

			if rights.Framework != tt.framework {
				t.Errorf("Framework mismatch: got %s, want %s", rights.Framework, tt.framework)
			}
		})
	}
}

func TestComplianceService_ProcessDataSubjectRequest(t *testing.T) {
	svc := NewComplianceService()

	tests := []struct {
		name        string
		requestType string
		wantStatus  string
		wantErr     bool
	}{
		{
			name:        "access_request",
			requestType: "access",
			wantStatus:  "completed",
			wantErr:     false,
		},
		{
			name:        "deletion_request",
			requestType: "deletion",
			wantStatus:  "completed",
			wantErr:     false,
		},
		{
			name:        "correction_request",
			requestType: "correction",
			wantStatus:  "completed",
			wantErr:     false,
		},
		{
			name:        "portability_request",
			requestType: "portability",
			wantStatus:  "completed",
			wantErr:     false,
		},
		{
			name:        "unsupported_request",
			requestType: "unknown",
			wantStatus:  "failed",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			request := &DSRRequest{
				Type:        tt.requestType,
				UserID:      1,
				Framework:   FrameworkGDPR,
				RequestDate: time.Now(),
			}

			response, err := svc.ProcessDataSubjectRequest(ctx, FrameworkGDPR, request)

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

			if response.Status != tt.wantStatus {
				t.Errorf("Status mismatch: got %s, want %s", response.Status, tt.wantStatus)
			}

			if response.RequestID == "" {
				t.Error("Expected non-empty request ID")
			}
		})
	}
}

func TestComplianceService_GenerateComplianceReport(t *testing.T) {
	svc := NewComplianceService()

	tests := []struct {
		name     string
		framework ComplianceFramework
		period   string
		wantErr  bool
	}{
		{
			name:      "ccpa_report",
			framework: FrameworkCCPA,
			period:   "Q1-2024",
			wantErr:  false,
		},
		{
			name:      "pipl_report",
			framework: FrameworkPIPL,
			period:   "2024",
			wantErr:  false,
		},
		{
			name:      "lgpd_report",
			framework: FrameworkLGPD,
			period:   "January-2024",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			report, err := svc.GenerateComplianceReport(ctx, tt.framework, tt.period)

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

			if report.Period != tt.period {
				t.Errorf("Period mismatch: got %s, want %s", report.Period, tt.period)
			}

			if report.ComplianceScore < 0 || report.ComplianceScore > 100 {
				t.Errorf("Invalid compliance score: %f", report.ComplianceScore)
			}
		})
	}
}

func TestComplianceService_ValidateDataProcessing(t *testing.T) {
	svc := NewComplianceService()

	tests := []struct {
		name       string
		processing *DataProcessingActivity
		wantValid  bool
		wantErr    bool
	}{
		{
			name: "valid_processing",
			processing: &DataProcessingActivity{
				ActivityID:    "proc-001",
				Purpose:      "user_authentication",
				LegalBasis:   "consent",
				DataCategories: []string{"email", "password"},
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "missing_purpose",
			processing: &DataProcessingActivity{
				ActivityID: "proc-002",
				LegalBasis: "consent",
			},
			wantValid: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			validation, err := svc.ValidateDataProcessing(ctx, FrameworkGDPR, tt.processing)

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

			if validation.IsCompliant != tt.wantValid {
				t.Errorf("IsCompliant mismatch: got %v, want %v", validation.IsCompliant, tt.wantValid)
			}
		})
	}
}
