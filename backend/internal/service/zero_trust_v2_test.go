package service

import (
	"context"
	"testing"
	"time"
)

func TestZeroTrustV2Service_ContinuousValidate(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	behaviors := []*BehaviorMetric{
		{
			MetricID:   "behavior_1",
			MetricType: "normal_pattern",
			Value:      85.0,
			Timestamp:  time.Now(),
		},
		{
			MetricID:   "behavior_2",
			MetricType: "keystroke_dynamics",
			Value:      90.0,
			Timestamp:  time.Now(),
		},
	}

	result, err := service.ContinuousValidate(ctx, "session_123", 25.0, behaviors)
	if err != nil {
		t.Fatalf("ContinuousValidate failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.SessionID != "session_123" {
		t.Errorf("SessionID mismatch: got %s, want session_123", result.SessionID)
	}

	if result.RiskLevel != "low" {
		t.Errorf("RiskLevel should be low, got %s", result.RiskLevel)
	}

	if !result.IsValid {
		t.Error("Result should be valid for low risk score")
	}

	if result.RequireReauth {
		t.Error("Reauth should not be required for low risk")
	}

	if len(result.Factors) == 0 {
		t.Error("Should have risk factors")
	}
}

func TestZeroTrustV2Service_ContinuousValidate_HighRisk(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	behaviors := []*BehaviorMetric{
		{
			MetricID:   "behavior_1",
			MetricType: "unusual_time",
			Value:      20.0,
			Timestamp:  time.Now(),
		},
		{
			MetricID:   "behavior_2",
			MetricType: "suspicious_pattern",
			Value:      15.0,
			Timestamp:  time.Now(),
		},
	}

	result, err := service.ContinuousValidate(ctx, "session_456", 80.0, behaviors)
	if err != nil {
		t.Fatalf("ContinuousValidate failed: %v", err)
	}

	if result.IsValid {
		t.Error("Result should not be valid for critical risk")
	}

	if !result.RequireReauth {
		t.Error("Reauth should be required for critical risk")
	}

	if !result.ThreatDetected {
		t.Error("Threat should be detected")
	}
}

func TestZeroTrustV2Service_ContinuousValidate_InvalidRiskScore(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	_, err := service.ContinuousValidate(ctx, "session_789", -10.0, nil)
	if err != ErrInvalidRiskScoreV2 {
		t.Errorf("Should return ErrInvalidRiskScoreV2 for negative risk score, got %v", err)
	}

	_, err = service.ContinuousValidate(ctx, "session_789", 150.0, nil)
	if err != ErrInvalidRiskScoreV2 {
		t.Errorf("Should return ErrInvalidRiskScoreV2 for over max risk score, got %v", err)
	}
}

func TestZeroTrustV2Service_UpdateValidationStatus(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	status := &ValidationStatus{
		UserID:    123,
		Status:    "active",
		RiskScore: 35.0,
		RiskLevel: "medium",
		IsValid:   true,
	}

	err := service.UpdateValidationStatus(ctx, "session_update_test", status)
	if err != nil {
		t.Fatalf("UpdateValidationStatus failed: %v", err)
	}

	if status.StatusID == "" {
		t.Error("StatusID should be set")
	}

	if status.SessionID != "session_update_test" {
		t.Errorf("SessionID should be session_update_test, got %s", status.SessionID)
	}
}

func TestZeroTrustV2Service_GetValidationHistory(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	userID := uint(100)

	statuses := []*ValidationStatus{
		{UserID: userID, Status: "active", RiskScore: 20.0, RiskLevel: "low", IsValid: true},
		{UserID: userID, Status: "active", RiskScore: 35.0, RiskLevel: "medium", IsValid: true},
		{UserID: userID, Status: "active", RiskScore: 50.0, RiskLevel: "medium", IsValid: true},
		{UserID: 200, Status: "active", RiskScore: 40.0, RiskLevel: "medium", IsValid: true},
	}

	for i, status := range statuses {
		sessionID := "history_session"
		if i%2 == 0 {
			sessionID = "history_session"
		} else {
			sessionID = "other_session"
		}
		err := service.UpdateValidationStatus(ctx, sessionID, status)
		if err != nil {
			t.Fatalf("UpdateValidationStatus failed: %v", err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	history, err := service.GetValidationHistory(ctx, userID, 10, 0)
	if err != nil {
		t.Fatalf("GetValidationHistory failed: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("Should return 3 history records for user 100, got %d", len(history))
	}

	allHistory, err := service.GetValidationHistory(ctx, 0, 10, 0)
	if err != nil {
		t.Fatalf("GetValidationHistory failed: %v", err)
	}

	if len(allHistory) < 4 {
		t.Errorf("Should return at least 4 history records, got %d", len(allHistory))
	}

	paginatedHistory, err := service.GetValidationHistory(ctx, userID, 2, 0)
	if err != nil {
		t.Fatalf("GetValidationHistory failed: %v", err)
	}

	if len(paginatedHistory) != 2 {
		t.Errorf("Should return 2 history records with pagination, got %d", len(paginatedHistory))
	}
}

func TestZeroTrustV2Service_AnalyzeNetworkSegment(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	segment := &NetworkSegment{
		SegmentID:   "segment_test",
		SegmentName: "Test Segment",
		SegmentType: "internal",
		TrustLevel:  "medium",
		Devices: []*DeviceInfo{
			{
				DeviceID:         "device_1",
				DeviceType:      "server",
				IPAddress:       "10.0.0.1",
				ComplianceStatus: "compliant",
			},
			{
				DeviceID:         "device_2",
				DeviceType:      "workstation",
				IPAddress:       "10.0.0.2",
				ComplianceStatus: "non_compliant",
			},
		},
		Connections: []*ConnectionInfo{
			{
				SourceID:     "device_1",
				TargetID:     "device_2",
				Protocol:     "tcp",
				Port:         443,
				IsAllowed:    true,
				LastActivity: time.Now(),
			},
			{
				SourceID:     "unknown",
				TargetID:     "device_2",
				Protocol:     "tcp",
				Port:         22,
				IsAllowed:    false,
				LastActivity: time.Now(),
			},
		},
	}

	analysis, err := service.AnalyzeNetworkSegment(ctx, segment)
	if err != nil {
		t.Fatalf("AnalyzeNetworkSegment failed: %v", err)
	}

	if analysis == nil {
		t.Fatal("Analysis should not be nil")
	}

	if analysis.TotalDevices != 2 {
		t.Errorf("TotalDevices should be 2, got %d", analysis.TotalDevices)
	}

	if analysis.TotalConnections != 2 {
		t.Errorf("TotalConnections should be 2, got %d", analysis.TotalConnections)
	}

	if analysis.TrustScore == 0 {
		t.Error("TrustScore should not be zero")
	}

	if len(analysis.Vulnerabilities) == 0 {
		t.Error("Should detect non-compliant device")
	}

	if len(analysis.Threats) == 0 {
		t.Error("Should detect unauthorized connection")
	}
}

func TestZeroTrustV2Service_CreateMicrosegment(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	segment := &MicrosegmentV2{
		Name:        "New Microsegment",
		Description: "Test microsegment",
		SourceIP:    "192.168.100.0/24",
		Port:        8080,
		Protocol:    "tcp",
		AllowedUsers: []string{"user1", "user2"},
		Rules: []*SegmentRule{
			{
				RuleID:   "rule_1",
				RuleName: "Allow specific users",
				Conditions: []*RuleCondition{
					{Field: "user_id", Operator: "in", Value: "user1,user2"},
				},
				Action:    "allow",
				Priority:  100,
				IsEnabled: true,
			},
		},
	}

	segmentID, err := service.CreateMicrosegment(ctx, segment)
	if err != nil {
		t.Fatalf("CreateMicrosegment failed: %v", err)
	}

	if segmentID == "" {
		t.Error("SegmentID should not be empty")
	}

	if segment.SegmentID != segmentID {
		t.Errorf("SegmentID mismatch: got %s, want %s", segment.SegmentID, segmentID)
	}

	if !segment.IsActive {
		t.Error("Segment should be active after creation")
	}
}

func TestZeroTrustV2Service_UpdateMicrosegment(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	segment := &MicrosegmentV2{
		Name:   "Test Segment",
		Rules:  []*SegmentRule{},
	}
	segmentID, _ := service.CreateMicrosegment(ctx, segment)

	newRules := []*SegmentRule{
		{
			RuleID:   "updated_rule_1",
			RuleName: "Updated Rule",
			Conditions: []*RuleCondition{
				{Field: "ip", Operator: "eq", Value: "192.168.1.1"},
			},
			Action:    "allow",
			Priority:  90,
			IsEnabled: true,
		},
	}

	err := service.UpdateMicrosegment(ctx, segmentID, newRules)
	if err != nil {
		t.Fatalf("UpdateMicrosegment failed: %v", err)
	}

	err = service.UpdateMicrosegment(ctx, "nonexistent", newRules)
	if err != ErrSegmentNotFound {
		t.Errorf("Should return ErrSegmentNotFound, got %v", err)
	}
}

func TestZeroTrustV2Service_DeleteMicrosegment(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	segment := &MicrosegmentV2{Name: "Delete Test"}
	segmentID, _ := service.CreateMicrosegment(ctx, segment)

	err := service.DeleteMicrosegment(ctx, segmentID)
	if err != nil {
		t.Fatalf("DeleteMicrosegment failed: %v", err)
	}

	_, err = service.ValidateMicrosegmentAccess(ctx, segmentID, "resource_1", "user1")
	if err != ErrSegmentNotFound {
		t.Errorf("Should return ErrSegmentNotFound after deletion, got %v", err)
	}
}

func TestZeroTrustV2Service_ValidateMicrosegmentAccess(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	segment := &MicrosegmentV2{
		Name:         "Access Test Segment",
		AllowedUsers: []string{"allowed_user"},
		Rules: []*SegmentRule{
			{
				RuleID:   "rule_allow",
				RuleName: "Allow Rule",
				Conditions: []*RuleCondition{
					{Field: "user_id", Operator: "eq", Value: "allowed_user"},
				},
				Action:    "allow",
				Priority:  100,
				IsEnabled: true,
			},
			{
				RuleID:   "rule_deny",
				RuleName: "Deny Rule",
				Conditions: []*RuleCondition{
					{Field: "resource_id", Operator: "eq", Value: "forbidden_resource"},
				},
				Action:    "deny",
				Priority:  90,
				IsEnabled: true,
			},
		},
	}
	segmentID, _ := service.CreateMicrosegment(ctx, segment)

	decision, err := service.ValidateMicrosegmentAccess(ctx, segmentID, "resource_1", "allowed_user")
	if err != nil {
		t.Fatalf("ValidateMicrosegmentAccess failed: %v", err)
	}

	if decision == nil {
		t.Fatal("Decision should not be nil")
	}

	if !decision.Allowed {
		t.Error("Access should be allowed for allowed_user")
	}

	deniedDecision, err := service.ValidateMicrosegmentAccess(ctx, segmentID, "forbidden_resource", "allowed_user")
	if err != nil {
		t.Fatalf("ValidateMicrosegmentAccess failed: %v", err)
	}

	if deniedDecision.Allowed {
		t.Error("Access should be denied for forbidden_resource")
	}

	_, err = service.ValidateMicrosegmentAccess(ctx, "nonexistent_segment", "resource_1", "user1")
	if err != ErrSegmentNotFound {
		t.Errorf("Should return ErrSegmentNotFound, got %v", err)
	}
}

func TestZeroTrustV2Service_CalculateDynamicPermissions(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	accessCtx := &AccessContext{
		UserID:       100,
		Resource:     "api_endpoint",
		Action:       "read",
		Time:         time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Location:     "office",
		DeviceStatus: "compliant",
	}

	permissions, err := service.CalculateDynamicPermissions(ctx, 100, "api_endpoint", accessCtx)
	if err != nil {
		t.Fatalf("CalculateDynamicPermissions failed: %v", err)
	}

	if len(permissions) == 0 {
		t.Error("Should return at least one permission")
	}

	hasBasicAccess := false
	for _, perm := range permissions {
		if perm == "basic_access" {
			hasBasicAccess = true
			break
		}
	}
	if !hasBasicAccess {
		t.Error("Should include basic_access permission")
	}
}

func TestZeroTrustV2Service_RevokeDynamicPermission(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	userID := uint(200)

	err := service.RevokeDynamicPermission(ctx, userID, "nonexistent_permission")
	if err != ErrPermissionDenied {
		t.Errorf("Should return ErrPermissionDenied for nonexistent permission, got %v", err)
	}
}

func TestZeroTrustV2Service_EnrichSASEPolicy(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	policy := &SASEPolicy{
		PolicyID:   "test_policy",
		PolicyName: "Test Policy",
		PolicyType: "security",
		Priority:   50,
		Conditions: []*PolicyCondition{
			{ConditionType: "ip_list", Field: "source_ip", Operator: "in", Value: "malicious_ips"},
		},
		Actions:    []string{"block", "alert"},
		Targets:    []*PolicyTarget{{TargetType: "network", TargetID: "all"}},
		IsEnabled:  true,
	}

	enriched, err := service.EnrichSASEPolicy(ctx, policy)
	if err != nil {
		t.Fatalf("EnrichSASEPolicy failed: %v", err)
	}

	if enriched == nil {
		t.Fatal("Enriched policy should not be nil")
	}

	if enriched.ComputedRisk == 0 {
		t.Error("ComputedRisk should not be zero")
	}

	if enriched.MatchingUsers == 0 {
		t.Error("MatchingUsers should not be zero")
	}

	if len(enriched.Recommendations) == 0 {
		t.Error("Should have recommendations for high-risk policy")
	}

	nilPolicy, err := service.EnrichSASEPolicy(ctx, nil)
	if err == nil {
		t.Error("Should return error for nil policy")
	}
	if nilPolicy != nil {
		t.Error("Should return nil for nil policy")
	}
}

func TestZeroTrustV2Service_ValidateSASEPolicy(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	validPolicy := &SASEPolicy{
		PolicyID:   "valid_policy",
		PolicyName: "Valid Policy",
		Priority:   75,
		Conditions: []*PolicyCondition{
			{ConditionType: "risk", Field: "risk_score", Operator: "gt", Value: 50},
		},
		Actions:   []string{"alert"},
		IsEnabled: true,
	}

	svc := service.(*zeroTrustV2Service)
	svc.sasePolicies["valid_policy"] = validPolicy

	validation, err := service.ValidateSASEPolicy(ctx, "valid_policy")
	if err != nil {
		t.Fatalf("ValidateSASEPolicy failed: %v", err)
	}

	if !validation.IsValid {
		t.Error("Valid policy should pass validation")
	}

	if len(validation.Errors) != 0 {
		t.Error("Valid policy should have no errors")
	}

	_, err = service.ValidateSASEPolicy(ctx, "nonexistent_policy")
	if err != ErrPolicyNotFound {
		t.Errorf("Should return ErrPolicyNotFound, got %v", err)
	}
}

func TestZeroTrustV2Service_ProcessSASEEvent(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	event := &SASEEvent{
		EventType: "login_attempt",
		SourceIP:  "192.168.1.100",
		UserID:    123,
		DeviceID:  "device_abc",
		Action:    "login",
		Resource:  "api",
		Result:    "success",
		Metadata:  map[string]interface{}{"risk_score": 55.0},
	}

	result, err := service.ProcessSASEEvent(ctx, event)
	if err != nil {
		t.Fatalf("ProcessSASEEvent failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if !result.Processed {
		t.Error("Event should be processed")
	}

	if result.EventID == "" {
		t.Error("EventID should be set")
	}

	if len(result.Actions) == 0 {
		t.Error("Should have at least one action from matching policies")
	}

	if result.RiskScore < 0 {
		t.Error("RiskScore should not be negative")
	}

	nilEvent, err := service.ProcessSASEEvent(ctx, nil)
	if err == nil {
		t.Error("Should return error for nil event")
	}
	if nilEvent != nil {
		t.Error("Should return nil for nil event")
	}
}

func TestZeroTrustV2Service_GetSASEMetrics(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	events := []*SASEEvent{
		{EventType: "login", SourceIP: "10.0.0.1", UserID: 1, Action: "login", Result: "success"},
		{EventType: "access", SourceIP: "10.0.0.2", UserID: 2, Action: "access", Result: "success"},
		{EventType: "login", SourceIP: "10.0.0.3", UserID: 3, Action: "login", Result: "failed"},
	}

	for _, event := range events {
		_, err := service.ProcessSASEEvent(ctx, event)
		if err != nil {
			t.Fatalf("ProcessSASEEvent failed: %v", err)
		}
	}

	metrics, err := service.GetSASEMetrics(ctx)
	if err != nil {
		t.Fatalf("GetSASEMetrics failed: %v", err)
	}

	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	if metrics.TotalPolicies == 0 {
		t.Error("Should have default policies")
	}

	if metrics.ActivePolicies == 0 {
		t.Error("Should have active policies")
	}

	if metrics.TotalEvents != 3 {
		t.Errorf("TotalEvents should be 3, got %d", metrics.TotalEvents)
	}

	if len(metrics.MetricsByType) == 0 {
		t.Error("Should have metrics by type")
	}
}

func TestZeroTrustV2Service_RiskLevels(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	testCases := []struct {
		riskScore   float64
		expectedLevel string
	}{
		{10.0, "minimal"},
		{25.0, "low"},
		{45.0, "medium"},
		{65.0, "high"},
		{85.0, "critical"},
	}

	for _, tc := range testCases {
		result, err := service.ContinuousValidate(ctx, "session_test", tc.riskScore, nil)
		if err != nil {
			t.Fatalf("ContinuousValidate failed for risk score %f: %v", tc.riskScore, err)
		}

		if result.RiskLevel != tc.expectedLevel {
			t.Errorf("For risk score %f, expected level %s, got %s", tc.riskScore, tc.expectedLevel, result.RiskLevel)
		}
	}
}

func TestZeroTrustV2Service_DefaultPolicies(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	metrics, err := service.GetSASEMetrics(ctx)
	if err != nil {
		t.Fatalf("GetSASEMetrics failed: %v", err)
	}

	if metrics.TotalPolicies < 2 {
		t.Errorf("Should have at least 2 default policies, got %d", metrics.TotalPolicies)
	}

	if metrics.ActivePolicies != metrics.TotalPolicies {
		t.Error("All default policies should be active")
	}
}

func TestZeroTrustV2Service_DefaultMicrosegments(t *testing.T) {
	service := NewZeroTrustV2Service()
	ctx := context.Background()

	decision, err := service.ValidateMicrosegmentAccess(ctx, "internal_services", "resource", "user")
	if err != nil {
		t.Fatalf("ValidateMicrosegmentAccess failed: %v", err)
	}

	if !decision.Allowed {
		t.Error("Internal services segment should allow access by default rules")
	}

	dmzDecision, err := service.ValidateMicrosegmentAccess(ctx, "dmz", "resource", "user")
	if err != nil {
		t.Fatalf("ValidateMicrosegmentAccess failed: %v", err)
	}

	if dmzDecision.Allowed {
		t.Error("DMZ segment should deny non-HTTP traffic by default")
	}
}
