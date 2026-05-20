package solution

import (
	"context"
	"testing"
	"time"
)

func TestHealthcareComplianceService_ValidatePHI(t *testing.T) {
	service := NewHealthcareComplianceService().(HealthcareComplianceService)

	tests := []struct {
		name      string
		data      *PHIData
		wantValid bool
	}{
		{
			name: "valid PHI data",
			data: &PHIData{
				PatientID:   "PAT001",
				PatientName: "John Doe",
				DateOfBirth: time.Now().AddDate(-30, 0, 0),
				MedicalRecordNumber: "MRN001",
				SSN:         "123-45-6789",
			},
			wantValid: true,
		},
		{
			name: "missing patient ID",
			data: &PHIData{
				PatientName: "John Doe",
				DateOfBirth: time.Now().AddDate(-30, 0, 0),
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ValidatePHI(context.Background(), tt.data)
			if err != nil {
				t.Fatalf("ValidatePHI() error = %v", err)
			}
			if result.Valid != tt.wantValid {
				t.Errorf("ValidatePHI() valid = %v, want %v", result.Valid, tt.wantValid)
			}
		})
	}
}

func TestHealthcareComplianceService_AnonymizeData(t *testing.T) {
	service := NewHealthcareComplianceService().(HealthcareComplianceService)

	data := &PHIData{
		PatientID:   "PAT001",
		PatientName: "John Doe",
		SSN:         "123-45-6789",
		DateOfBirth: time.Now().AddDate(-30, 0, 0),
	}

	rules := &AnonymizationRules{
		DirectIdentifiers: []string{"patient_name", "ssn"},
		KAnonymity:        5,
		LDiversity:        2,
		DateShift:         true,
		DateShiftRange:    30,
	}

	result, err := service.AnonymizeData(context.Background(), data, rules)
	if err != nil {
		t.Fatalf("AnonymizeData() error = %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	if resultMap["patient_name"] != "[REDACTED]" {
		t.Errorf("Expected patient_name to be [REDACTED], got %v", resultMap["patient_name"])
	}

	if resultMap["ssn"] != "[REDACTED]" {
		t.Errorf("Expected SSN to be [REDACTED], got %v", resultMap["ssn"])
	}
}

func TestHealthcareComplianceService_CheckHIPAACompliance(t *testing.T) {
	service := NewHealthcareComplianceService().(HealthcareComplianceService)

	operation := &HealthcareOperation{
		OperationType: "access",
		PatientID:     "PAT001",
		OperatorID:    "USER001",
		Purpose:       "treatment",
		Timestamp:     time.Now(),
	}

	result, err := service.CheckHIPAACompliance(context.Background(), operation)
	if err != nil {
		t.Fatalf("CheckHIPAACompliance() error = %v", err)
	}

	if !result.Compliant {
		t.Error("Expected operation to be compliant for valid purpose")
	}
}

func TestHealthcareComplianceService_ManageConsent(t *testing.T) {
	service := NewHealthcareComplianceService().(HealthcareComplianceService)

	consent := &PatientConsent{
		ConsentID:     "CONSENT001",
		PatientID:     "PAT001",
		ConsentType:   "treatment",
		Granted:       true,
		EffectiveDate: time.Now(),
		ExpirationDate: time.Now().AddDate(1, 0, 0),
		Scope:         []string{"read", "write"},
		Revocable:     true,
	}

	err := service.ManageConsent(context.Background(), consent)
	if err != nil {
		t.Fatalf("ManageConsent() error = %v", err)
	}

	err = service.ManageConsent(context.Background(), &PatientConsent{
		PatientID: "PAT001",
	})
	if err == nil {
		t.Error("Expected error for consent without ID")
	}
}

func TestHealthcareComplianceService_AuditAccess(t *testing.T) {
	service := NewHealthcareComplianceService().(HealthcareComplianceService)

	access := &PHIAccess{
		PatientID:    "PAT001",
		UserID:       "USER001",
		AccessType:   "read",
		Timestamp:    time.Now(),
		DataElements: []string{"diagnosis", "medications"},
		Purpose:      "treatment",
		IPAddress:    "192.168.1.1",
		Success:      true,
	}

	err := service.AuditAccess(context.Background(), access)
	if err != nil {
		t.Fatalf("AuditAccess() error = %v", err)
	}
}

func TestHealthcareComplianceService_GeneratePrivacyReport(t *testing.T) {
	service := NewHealthcareComplianceService().(HealthcareComplianceService)

	consent := &PatientConsent{
		ConsentID:     "CONSENT001",
		PatientID:     "PAT001",
		ConsentType:   "treatment",
		Granted:       true,
		EffectiveDate: time.Now(),
		ExpirationDate: time.Now().AddDate(1, 0, 0),
	}
	service.ManageConsent(context.Background(), consent)

	report, err := service.GeneratePrivacyReport(context.Background(), "PAT001")
	if err != nil {
		t.Fatalf("GeneratePrivacyReport() error = %v", err)
	}

	if len(report.Consents) == 0 {
		t.Error("Expected at least one consent in report")
	}
}
