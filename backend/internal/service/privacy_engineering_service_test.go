package service

import (
	"context"
	"testing"
	"time"
)

func TestPrivacyEngineeringService_CreatePrivacyImpactAssessment(t *testing.T) {
	svc := NewPrivacyEngineeringService()

	tests := []struct {
		name   string
		pia    *PIAProject
		wantErr bool
	}{
		{
			name: "valid_pia",
			pia: &PIAProject{
				Name:        "New User Analytics Project",
				Description: "Implementation of user behavior analytics",
				DataTypes:   []string{"email", "behavior"},
				Purpose:     "analytics",
				LegalBasis:  "consent",
				CreatedBy:   "admin",
			},
			wantErr: false,
		},
		{
			name: "pia_with_third_parties",
			pia: &PIAProject{
				Name:        "Third-party Integration Project",
				Description: "Integration with external partners",
				DataTypes:   []string{"email", "name", "purchase_history"},
				Purpose:     "service_delivery",
				LegalBasis:  "contract",
				ThirdParties: []ThirdPartyInvolvement{
					{
						Name:       "Analytics Partner",
						Purpose:    "analytics",
						DataShared: []string{"email", "behavior"},
						Country:    "US",
						HasDPA:     true,
					},
				},
				CreatedBy: "admin",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := svc.CreatePrivacyImpactAssessment(ctx, tt.pia)

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
				t.Errorf("Expected PIA but got nil")
				return
			}

			if result.ProjectID == "" {
				t.Error("Expected non-empty project ID")
			}

			if result.Status != "draft" {
				t.Errorf("Expected status 'draft', got %s", result.Status)
			}

			if result.RiskAssessment.RiskScore < 0 || result.RiskAssessment.RiskScore > 100 {
				t.Errorf("Invalid risk score: %f", result.RiskAssessment.RiskScore)
			}
		})
	}
}

func TestPrivacyEngineeringService_GetPrivacyImpactAssessment(t *testing.T) {
	svc := NewPrivacyEngineeringService()
	ctx := context.Background()

	pia := &PIAProject{
		Name:        "Test Project",
		Description: "Test Description",
		DataTypes:   []string{"email"},
		Purpose:     "testing",
		LegalBasis:  "consent",
		CreatedBy:   "admin",
	}

	created, err := svc.CreatePrivacyImpactAssessment(ctx, pia)
	if err != nil {
		t.Fatalf("Failed to create PIA: %v", err)
	}

	retrieved, err := svc.GetPrivacyImpactAssessment(ctx, created.ProjectID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if retrieved == nil {
		t.Errorf("Expected PIA but got nil")
		return
	}

	if retrieved.ProjectID != created.ProjectID {
		t.Errorf("ProjectID mismatch: got %s, want %s", retrieved.ProjectID, created.ProjectID)
	}

	if retrieved.Name != created.Name {
		t.Errorf("Name mismatch: got %s, want %s", retrieved.Name, created.Name)
	}

	_, err = svc.GetPrivacyImpactAssessment(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent PIA")
	}
}

func TestPrivacyEngineeringService_AssessDataMinimization(t *testing.T) {
	svc := NewPrivacyEngineeringService()

	tests := []struct {
		name      string
		check     *DataMinimizationCheck
		wantValid bool
		wantErr   bool
	}{
		{
			name: "compliant_check",
			check: &DataMinimizationCheck{
				CheckID:       "dm-001",
				DataType:      "user_profile",
				Purpose:       "service_delivery",
				CollectedData: []string{"id", "email", "name"},
				RequiredData:  []string{"id", "email", "name"},
				LegalBasis:    "contract",
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "excess_data_collection",
			check: &DataMinimizationCheck{
				CheckID:       "dm-002",
				DataType:      "user_profile",
				Purpose:       "marketing",
				CollectedData: []string{"id", "email", "name", "phone", "address", "ssn", "credit_card"},
				RequiredData:  []string{"id", "email"},
				LegalBasis:    "consent",
			},
			wantValid: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := svc.AssessDataMinimization(ctx, tt.check)

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

			if result.IsCompliant != tt.wantValid {
				t.Errorf("IsCompliant mismatch: got %v, want %v", result.IsCompliant, tt.wantValid)
			}

			if result.ComplianceScore < 0 || result.ComplianceScore > 100 {
				t.Errorf("Invalid compliance score: %f", result.ComplianceScore)
			}
		})
	}
}

func TestPrivacyEngineeringService_ManageConsent(t *testing.T) {
	svc := NewPrivacyEngineeringService()

	tests := []struct {
		name      string
		consent   *ConsentRecord
		wantErr   bool
	}{
		{
			name: "grant_marketing_consent",
			consent: &ConsentRecord{
				UserID:        1,
				ConsentType:   ConsentMarketing,
				Purpose:       "marketing_communications",
				DataProcessed: []string{"email", "name"},
				Granted:       true,
				Method:        "web_form",
				IPAddress:     "192.0.2.1",
				UserAgent:     "Mozilla/5.0",
				Version:       "1.0",
			},
			wantErr: false,
		},
		{
			name: "grant_analytics_consent",
			consent: &ConsentRecord{
				UserID:        2,
				ConsentType:   ConsentAnalytics,
				Purpose:       "usage_analytics",
				DataProcessed: []string{"behavior"},
				Granted:       true,
				Method:        "web_form",
				IPAddress:     "192.0.2.2",
				UserAgent:     "Mozilla/5.0",
				Version:       "1.0",
			},
			wantErr: false,
		},
		{
			name: "essential_consent",
			consent: &ConsentRecord{
				UserID:        3,
				ConsentType:   ConsentEssential,
				Purpose:       "service_operation",
				DataProcessed: []string{"id", "email"},
				Granted:       false,
				Method:        "implicit",
				IPAddress:     "192.0.2.3",
				UserAgent:     "Mozilla/5.0",
				Version:       "1.0",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := svc.ManageConsent(ctx, tt.consent)

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
				t.Errorf("Expected consent record but got nil")
				return
			}

			if result.RecordID == "" {
				t.Error("Expected non-empty record ID")
			}

			if tt.consent.ConsentType == ConsentEssential && !result.Granted {
				t.Error("Essential consent should always be granted")
			}
		})
	}
}

func TestPrivacyEngineeringService_GetConsentStatus(t *testing.T) {
	svc := NewPrivacyEngineeringService()
	ctx := context.Background()

	userID := uint(100)

	consent1 := &ConsentRecord{
		UserID:        userID,
		ConsentType:   ConsentAnalytics,
		Purpose:       "analytics",
		DataProcessed: []string{"behavior"},
		Granted:       true,
		Method:        "web_form",
		IPAddress:     "192.0.2.1",
		Version:       "1.0",
	}

	consent2 := &ConsentRecord{
		UserID:        userID,
		ConsentType:   ConsentMarketing,
		Purpose:       "marketing",
		DataProcessed: []string{"email"},
		Granted:       false,
		Method:        "web_form",
		IPAddress:     "192.0.2.1",
		Version:       "1.0",
	}

	_, err := svc.ManageConsent(ctx, consent1)
	if err != nil {
		t.Fatalf("Failed to create consent 1: %v", err)
	}

	_, err = svc.ManageConsent(ctx, consent2)
	if err != nil {
		t.Fatalf("Failed to create consent 2: %v", err)
	}

	status, err := svc.GetConsentStatus(ctx, userID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if status == nil {
		t.Errorf("Expected status but got nil")
		return
	}

	if status.UserID != userID {
		t.Errorf("UserID mismatch: got %d, want %d", status.UserID, userID)
	}

	if !status.Consents[ConsentAnalytics] {
		t.Error("Expected analytics consent to be granted")
	}

	if status.Consents[ConsentMarketing] {
		t.Error("Expected marketing consent to be withdrawn")
	}
}

func TestPrivacyEngineeringService_ProcessDataSubjectRights(t *testing.T) {
	svc := NewPrivacyEngineeringService()

	tests := []struct {
		name     string
		request  *DataSubjectRightsRequest
		wantErr  bool
	}{
		{
			name: "access_right",
			request: &DataSubjectRightsRequest{
				RequestID:  "req-001",
				UserID:     1,
				RightType:  RightAccess,
				Framework: "gdpr",
				RequestedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "deletion_right",
			request: &DataSubjectRightsRequest{
				RequestID:  "req-002",
				UserID:     2,
				RightType:  RightDeletion,
				Framework: "ccpa",
				RequestedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "portability_right",
			request: &DataSubjectRightsRequest{
				RequestID:  "req-003",
				UserID:     3,
				RightType:  RightPortability,
				Framework: "pipl",
				RequestedAt: time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := svc.ProcessDataSubjectRights(ctx, tt.request)

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

			if response.RequestID != tt.request.RequestID {
				t.Errorf("RequestID mismatch: got %s, want %s", response.RequestID, tt.request.RequestID)
			}

			if response.Deadline.Before(time.Now()) {
				t.Error("Deadline should be in the future")
			}
		})
	}
}

func TestPrivacyEngineeringService_VerifyDataMinimization(t *testing.T) {
	svc := NewPrivacyEngineeringService()
	ctx := context.Background()

	tests := []struct {
		name     string
		dataType string
		wantErr  bool
	}{
		{
			name:     "verify_user_data",
			dataType: "user_profile",
			wantErr:  false,
		},
		{
			name:     "verify_transaction_data",
			dataType: "transaction",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verification, err := svc.VerifyDataMinimization(ctx, tt.dataType)

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

			if verification == nil {
				t.Errorf("Expected verification but got nil")
				return
			}

			if verification.DataType != tt.dataType {
				t.Errorf("DataType mismatch: got %s, want %s", verification.DataType, tt.dataType)
			}

			if !verification.Collection.IsCompliant {
				t.Error("Collection should be compliant")
			}

			if !verification.Storage.IsCompliant {
				t.Error("Storage should be compliant")
			}

			if !verification.Retention.IsCompliant {
				t.Error("Retention should be compliant")
			}
		})
	}
}
