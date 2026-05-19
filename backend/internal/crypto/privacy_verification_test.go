package crypto

import (
	"testing"
	"time"
)

func TestPrivacyVerificationService(t *testing.T) {
	service := NewPrivacyVerificationService(nil)
	if service == nil {
		t.Fatal("service should not be nil")
	}
}

func TestPrivacyVerificationRequest(t *testing.T) {
	service := NewPrivacyVerificationService(nil)

	subject := &DataSubject{
		SubjectID:    "subject_123",
		Pseudonym:    "anon_456",
		PrivacyLevel: PrivacyLevelMedium,
		Consents: map[ConsentType]bool{
			ConsentDataCollection: true,
			ConsentDataProcessing: true,
		},
		CreatedAt: time.Now(),
	}

	request := &PrivacyVerificationRequest{
		RequestID:     "req_789",
		DataSubject:   subject,
		DataToVerify:  map[string]interface{}{"score": 85, "status": "active"},
		StatementType: StatementKnowledge,
		PrivacyLevel:  PrivacyLevelMedium,
		Timestamp:     time.Now().Unix(),
		SessionID:     "session_abc",
	}

	result, err := service.Verify(request)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result.RequestID != "req_789" {
		t.Errorf("request ID mismatch")
	}

	if result.VerifiedAt.IsZero() {
		t.Error("verified at should be set")
	}
}

func TestConsentManagement(t *testing.T) {
	service := NewPrivacyVerificationService(nil)

	consent := &ConsentRecord{
		UserID:      "user_123",
		ConsentType: ConsentDataCollection,
		Granted:     true,
		GrantedAt:   time.Now(),
		Version:     "1.0",
	}

	err := service.RecordConsent(consent)
	if err != nil {
		t.Fatalf("failed to record consent: %v", err)
	}

	retrieved := service.GetConsent("user_123", ConsentDataCollection)
	if retrieved == nil {
		t.Fatal("consent should be retrievable")
	}

	if !retrieved.Granted {
		t.Error("consent should be granted")
	}

	err = service.RevokeConsent("user_123", ConsentDataCollection)
	if err != nil {
		t.Fatalf("failed to revoke consent: %v", err)
	}

	retrieved = service.GetConsent("user_123", ConsentDataCollection)
	if retrieved != nil {
		t.Error("revoked consent should not be retrievable")
	}
}

func TestDataSubjectCreation(t *testing.T) {
	service := NewPrivacyVerificationService(nil)

	subject := service.CreateDataSubject("user_456", PrivacyLevelHigh)
	if subject == nil {
		t.Fatal("subject should not be nil")
	}

	if subject.SubjectID != "user_456" {
		t.Errorf("subject ID mismatch: expected user_456, got %s", subject.SubjectID)
	}

	if subject.Pseudonym == "" {
		t.Error("pseudonym should be generated")
	}

	if subject.PrivacyLevel != PrivacyLevelHigh {
		t.Errorf("privacy level mismatch")
	}

	retrieved := service.GetDataSubject("user_456")
	if retrieved == nil {
		t.Fatal("subject should be retrievable")
	}
}

func TestPrivacyLevelManagement(t *testing.T) {
	service := NewPrivacyVerificationService(nil)

	service.CreateDataSubject("user_789", PrivacyLevelBasic)

	err := service.SetPrivacyLevel("user_789", PrivacyLevelMaximum)
	if err != nil {
		t.Fatalf("failed to set privacy level: %v", err)
	}

	subject := service.GetDataSubject("user_789")
	if subject.PrivacyLevel != PrivacyLevelMaximum {
		t.Errorf("privacy level should be maximum")
	}
}

func TestBudgetTracking(t *testing.T) {
	service := NewPrivacyVerificationService(nil)

	budget := service.GetBudget("user_budget")
	if budget == nil {
		t.Fatal("budget should not be nil")
	}

	if budget.TotalBudget <= 0 {
		t.Error("total budget should be positive")
	}

	if budget.RemainingBudget != budget.TotalBudget {
		t.Error("initial remaining budget should equal total budget")
	}

	service.ResetBudget("user_budget")

	budget = service.GetBudget("user_budget")
	if budget.RemainingBudget != budget.TotalBudget {
		t.Error("remaining budget should equal total after reset")
	}
}

func TestPrivacyVerificationStats(t *testing.T) {
	service := NewPrivacyVerificationService(nil)

	stats := service.GetStats()
	if stats == nil {
		t.Fatal("stats should not be nil")
	}

	if stats.TotalVerifications != 0 {
		t.Error("initial total verifications should be 0")
	}
}

func TestPrivacyLevels(t *testing.T) {
	levels := []PrivacyLevel{
		PrivacyLevelNone,
		PrivacyLevelBasic,
		PrivacyLevelMedium,
		PrivacyLevelHigh,
		PrivacyLevelMaximum,
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			if level.String() == "" {
				t.Error("level string should not be empty")
			}

			curveType := level.GetCurveType()
			if curveType == "" {
				t.Error("curve type should not be empty")
			}
		})
	}
}

func TestPrivacyPolicy(t *testing.T) {
	service := NewPrivacyVerificationService(nil)

	policy := &PrivacyPolicy{
		PolicyID:        "policy_123",
		UserID:          "user_456",
		DataCategories:  []string{"financial", "personal"},
		RetentionPeriod: 365 * 24 * time.Hour,
		SharingAllowed:  false,
		Version:         "1.0",
	}

	err := service.CreatePolicy(policy)
	if err != nil {
		t.Fatalf("failed to create policy: %v", err)
	}

	retrieved := service.GetPolicy("policy_123")
	if retrieved == nil {
		t.Fatal("policy should be retrievable")
	}

	if retrieved.PolicyID != "policy_123" {
		t.Errorf("policy ID mismatch")
	}

	err = service.UpdatePolicy(policy)
	if err != nil {
		t.Fatalf("failed to update policy: %v", err)
	}
}

func TestPrivacyBatchVerification(t *testing.T) {
	service := NewPrivacyVerificationService(nil)

	requests := make([]*PrivacyVerificationRequest, 0, 3)

	for i := 0; i < 3; i++ {
		subject := service.CreateDataSubject("user_batch", PrivacyLevelMedium)
		subject.Consents = map[ConsentType]bool{
			ConsentDataCollection: true,
			ConsentDataProcessing: true,
		}

		request := &PrivacyVerificationRequest{
			RequestID:     "batch_req_" + string(rune('0'+i)),
			DataSubject:   subject,
			DataToVerify:  map[string]interface{}{"index": i},
			StatementType: StatementKnowledge,
			PrivacyLevel:  PrivacyLevelMedium,
			SessionID:     "batch_session",
		}

		requests = append(requests, request)
	}

	results, err := service.BatchVerify(requests)
	if err != nil {
		t.Fatalf("batch verification failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestInvalidRequestValidation(t *testing.T) {
	service := NewPrivacyVerificationService(nil)

	tests := []struct {
		name    string
		request *PrivacyVerificationRequest
	}{
		{
			name:    "nil request",
			request: nil,
		},
		{
			name: "nil data subject",
			request: &PrivacyVerificationRequest{
				DataSubject: nil,
			},
		},
		{
			name: "empty subject ID",
			request: &PrivacyVerificationRequest{
				DataSubject: &DataSubject{
					SubjectID: "",
				},
			},
		},
		{
			name: "nil data to verify",
			request: &PrivacyVerificationRequest{
				DataSubject: &DataSubject{
					SubjectID: "user_123",
				},
				DataToVerify: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.Verify(tt.request)
			if err == nil && result != nil && result.Valid {
				t.Errorf("verification should fail for %s", tt.name)
			}
		})
	}
}

func TestConsentValidation(t *testing.T) {
	config := &PrivacyConfig{
		DefaultPrivacyLevel:  PrivacyLevelMedium,
		RequireConsent:       true,
		EnableBudgetTracking: false,
	}

	service := NewPrivacyVerificationService(config)

	subjectWithoutConsent := &DataSubject{
		SubjectID:    "no_consent_user",
		PrivacyLevel: PrivacyLevelMedium,
		Consents:     map[ConsentType]bool{},
	}

	request := &PrivacyVerificationRequest{
		RequestID:     "req_no_consent",
		DataSubject:   subjectWithoutConsent,
		DataToVerify:  map[string]interface{}{"test": "data"},
		StatementType: StatementKnowledge,
		PrivacyLevel:  PrivacyLevelMedium,
		SessionID:     "session",
	}

	result, err := service.Verify(request)
	if err == nil && result != nil {
		if result.Valid {
			t.Error("verification should fail without consent")
		}
	}
}

func TestPrivacyPreservation(t *testing.T) {
	service := NewPrivacyVerificationService(nil)

	levels := []PrivacyLevel{
		PrivacyLevelBasic,
		PrivacyLevelMedium,
		PrivacyLevelHigh,
		PrivacyLevelMaximum,
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			subject := &DataSubject{
				SubjectID:    "user_" + level.String(),
				PrivacyLevel: level,
				Consents: map[ConsentType]bool{
					ConsentDataCollection: true,
					ConsentDataProcessing: true,
				},
			}

			request := &PrivacyVerificationRequest{
				RequestID:     "req_" + level.String(),
				DataSubject:   subject,
				DataToVerify:  map[string]interface{}{"sensitive": "data"},
				StatementType: StatementKnowledge,
				PrivacyLevel:  level,
				SessionID:     "session",
			}

			result, err := service.Verify(request)
			if err != nil {
				t.Fatalf("verification failed for level %s: %v", level, err)
			}

			if result != nil && !result.PrivacyPreserved {
				t.Error("privacy should be preserved")
			}
		})
	}
}

func TestBudgetExceeded(t *testing.T) {
	config := &PrivacyConfig{
		DefaultPrivacyLevel:   PrivacyLevelMedium,
		EnableBudgetTracking:  true,
	}

	service := NewPrivacyVerificationService(config)

	subject := service.CreateDataSubject("budget_user", PrivacyLevelMedium)
	subject.Consents = map[ConsentType]bool{
		ConsentDataCollection: true,
		ConsentDataProcessing: true,
	}

	budget := service.GetBudget("budget_user")
	budget.TotalBudget = 1
	budget.UsedBudget = 1
	budget.RemainingBudget = 0

	request := &PrivacyVerificationRequest{
		RequestID:     "req_budget",
		DataSubject:   subject,
		DataToVerify:  map[string]interface{}{"test": "data"},
		StatementType: StatementKnowledge,
		PrivacyLevel:  PrivacyLevelMedium,
		SessionID:     "session",
	}

	result, err := service.Verify(request)
	if err == nil && result != nil {
		if result.Valid {
			t.Error("verification should fail when budget exceeded")
		}
	}
}
