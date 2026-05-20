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
		HoneypotType:  "high_interaction",
		Name:          "Test Honeypot",
		Description:   "A test honeypot for unit testing",
		Port:          22,
		Protocol:      "tcp",
		IPAddress:     "192.168.1.100",
		Complexity:    5,
		Attractiveness: 70,
		ResponseDelay: 100,
		Services:      []string{"ssh", "ftp", "http"},
	}

	honeypotID, err := service.CreateHoneypot(ctx, config)
	if err != nil {
		t.Fatalf("CreateHoneypot failed: %v", err)
	}

	if honeypotID == "" {
		t.Error("HoneypotID should not be empty")
	}

	if config.HoneypotID != honeypotID {
		t.Errorf("HoneypotID mismatch: got %s, want %s", config.HoneypotID, honeypotID)
	}

	if config.IsActive {
		t.Error("Honeypot should not be active after creation")
	}

	_, err = service.CreateHoneypot(ctx, nil)
	if err != ErrInvalidConfig {
		t.Errorf("Should return ErrInvalidConfig for nil config, got %v", err)
	}
}

func TestActiveDefenseService_ActivateDeactivateHoneypot(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	config := &HoneypotConfig{
		Name: "Test Honeypot",
		Port: 22,
	}

	honeypotID, _ := service.CreateHoneypot(ctx, config)

	err := service.ActivateHoneypot(ctx, honeypotID)
	if err != nil {
		t.Fatalf("ActivateHoneypot failed: %v", err)
	}

	status, err := service.MonitorHoneypot(ctx, honeypotID)
	if err != nil {
		t.Fatalf("MonitorHoneypot failed: %v", err)
	}

	if !status.IsActive {
		t.Error("Honeypot should be active after activation")
	}

	err = service.DeactivateHoneypot(ctx, honeypotID)
	if err != nil {
		t.Fatalf("DeactivateHoneypot failed: %v", err)
	}

	status, err = service.MonitorHoneypot(ctx, honeypotID)
	if err != nil {
		t.Fatalf("MonitorHoneypot failed: %v", err)
	}

	if status.IsActive {
		t.Error("Honeypot should be inactive after deactivation")
	}

	err = service.ActivateHoneypot(ctx, "nonexistent_id")
	if err != ErrHoneypotNotFound {
		t.Errorf("Should return ErrHoneypotNotFound, got %v", err)
	}

	err = service.DeactivateHoneypot(ctx, "nonexistent_id")
	if err != ErrHoneypotNotFound {
		t.Errorf("Should return ErrHoneypotNotFound, got %v", err)
	}
}

func TestActiveDefenseService_MonitorHoneypot(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	config := &HoneypotConfig{
		Name: "Monitor Test",
		Port: 8080,
	}

	honeypotID, _ := service.CreateHoneypot(ctx, config)
	service.ActivateHoneypot(ctx, honeypotID)

	status, err := service.MonitorHoneypot(ctx, honeypotID)
	if err != nil {
		t.Fatalf("MonitorHoneypot failed: %v", err)
	}

	if status == nil {
		t.Fatal("Status should not be nil")
	}

	if status.HoneypotID != honeypotID {
		t.Errorf("HoneypotID mismatch: got %s, want %s", status.HoneypotID, honeypotID)
	}

	if status.VisitorsByIP == nil {
		t.Error("VisitorsByIP should be initialized")
	}

	_, err = service.MonitorHoneypot(ctx, "nonexistent_id")
	if err != ErrHoneypotNotFound {
		t.Errorf("Should return ErrHoneypotNotFound, got %v", err)
	}
}

func TestActiveDefenseService_TrackIntruder(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	intruderInfo := &IntruderInfo{
		IPAddress:     "203.0.113.50",
		ASNumber:      "AS12345",
		Country:       "Unknown",
		ISP:           "Test ISP",
		FirstSeen:     time.Now().Add(-48 * time.Hour),
		LastSeen:      time.Now(),
		Tactics:       []string{"initial_access", "execution"},
		Techniques:    []string{"T1190", "T1059"},
		Tools:         []string{"nmap", "metasploit"},
		Targets:       []string{"web_server", "database"},
		SuccessRate:   0.6,
		ActivityCount: 150,
		Reputation:    "suspicious",
		ThreatLevel:   "high",
	}

	profile, err := service.TrackIntruder(ctx, intruderInfo)
	if err != nil {
		t.Fatalf("TrackIntruder failed: %v", err)
	}

	if profile == nil {
		t.Fatal("Profile should not be nil")
	}

	if profile.IntruderID == "" {
		t.Error("IntruderID should be set")
	}

	if profile.Classification == "" {
		t.Error("Classification should be set")
	}

	if profile.RiskScore == 0 {
		t.Error("RiskScore should not be zero")
	}

	if profile.BehavioralProfile == nil {
		t.Error("BehavioralProfile should be initialized")
	}

	if !profile.IsKnownActor {
		t.Error("Known threat actor should be marked as known actor")
	}

	_, err = service.TrackIntruder(ctx, nil)
	if err == nil {
		t.Error("Should return error for nil intruder info")
	}
}

func TestActiveDefenseService_GetAdaptiveResponse(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	threat := &ThreatInfoV2{
		ThreatID:    "threat_1",
		ThreatType:  "brute_force",
		Severity:    "high",
		Description: "Brute force attack detected",
	}

	response, err := service.GetAdaptiveResponse(ctx, threat)
	if err != nil {
		t.Fatalf("GetAdaptiveResponse failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.ResponseID == "" {
		t.Error("ResponseID should be set")
	}

	if response.ResponseType == "" {
		t.Error("ResponseType should be set")
	}

	if len(response.Actions) == 0 {
		t.Error("Should have at least one action")
	}

	if response.Confidence == 0 {
		t.Error("Confidence should not be zero")
	}

	if !response.IsAutomated {
		t.Error("Response should be automated")
	}

	criticalThreat := &ThreatInfoV2{
		ThreatID:    "critical_threat",
		ThreatType:  "exploitation",
		Severity:    "critical",
		Description: "Critical vulnerability exploited",
	}

	criticalResponse, err := service.GetAdaptiveResponse(ctx, criticalThreat)
	if err != nil {
		t.Fatalf("GetAdaptiveResponse failed for critical threat: %v", err)
	}

	if criticalResponse.Confidence < response.Confidence {
		t.Error("Critical threat should have higher confidence")
	}

	_, err = service.GetAdaptiveResponse(ctx, nil)
	if err == nil {
		t.Error("Should return error for nil threat")
	}
}

func TestActiveDefenseService_GenerateDeceptionElements(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	elements, err := service.GenerateDeceptionElements(ctx)
	if err != nil {
		t.Fatalf("GenerateDeceptionElements failed: %v", err)
	}

	if len(elements) == 0 {
		t.Error("Should generate at least one deception element")
	}

	for i, element := range elements {
		if element.ElementID == "" {
			t.Errorf("Element %d should have ElementID", i)
		}

		if element.ElementType == "" {
			t.Errorf("Element %d should have ElementType", i)
		}

		if element.DeceptionScore == 0 {
			t.Errorf("Element %d should have DeceptionScore", i)
		}

		if !element.IsActive {
			t.Errorf("Element %d should be active", i)
		}
	}

	metrics, err := service.GetDefenseMetrics(ctx)
	if err != nil {
		t.Fatalf("GetDefenseMetrics failed: %v", err)
	}

	if metrics.DeceptionElements == 0 {
		t.Error("Metrics should show deception elements")
	}
}

func TestActiveDefenseService_DeployDeceptionNetwork(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	elements := []*DeceptionElement{
		{ElementID: "elem_1", ElementType: "fake_credential", Name: "Decoy 1", IsActive: true, DeceptionScore: 80},
		{ElementID: "elem_2", ElementType: "decoy_file", Name: "Decoy 2", IsActive: true, DeceptionScore: 75},
	}

	network := &DeceptionNetwork{
		Name:      "Test Network",
		Elements:  elements,
		Complexity: 5,
		Coverage:  90,
	}

	err := service.DeployDeceptionNetwork(ctx, network)
	if err != nil {
		t.Fatalf("DeployDeceptionNetwork failed: %v", err)
	}

	if network.NetworkID == "" {
		t.Error("NetworkID should be set")
	}

	if !network.IsDeployed {
		t.Error("Network should be deployed")
	}

	err = service.DeployDeceptionNetwork(ctx, nil)
	if err == nil {
		t.Error("Should return error for nil network")
	}
}

func TestActiveDefenseService_AnalyzeAttackPattern(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	pattern := &AttackPattern{
		PatternType:  "sql_injection_pattern",
		SourceIP:     "192.168.1.50",
		TargetIP:     "192.168.1.100",
		AttackVector: "sql_injection",
		Frequency:    10.0,
		Similarity:   0.85,
		Indicators: []*AttackIndicator{
			{IndicatorID: "ind_1", IndicatorType: "payload", Value: "' OR '1'='1", Weight: 0.8, IsVerifed: true},
			{IndicatorID: "ind_2", IndicatorType: "frequency", Value: "high", Weight: 0.6, IsVerifed: true},
		},
	}

	analysis, err := service.AnalyzeAttackPattern(ctx, pattern)
	if err != nil {
		t.Fatalf("AnalyzeAttackPattern failed: %v", err)
	}

	if analysis == nil {
		t.Fatal("Analysis should not be nil")
	}

	if analysis.PatternID == "" {
		t.Error("PatternID should be set")
	}

	if analysis.PatternType == "" {
		t.Error("PatternType should be set")
	}

	if analysis.Confidence == 0 {
		t.Error("Confidence should not be zero")
	}

	if analysis.ThreatActor == "" {
		t.Error("ThreatActor should be identified")
	}

	if analysis.LikelyMotivation == "" {
		t.Error("LikelyMotivation should be determined")
	}

	if len(analysis.TTPs) == 0 {
		t.Error("Should have TTPs")
	}

	if len(analysis.Recommendations) == 0 {
		t.Error("Should have recommendations")
	}

	_, err = service.AnalyzeAttackPattern(ctx, nil)
	if err == nil {
		t.Error("Should return error for nil pattern")
	}
}

func TestActiveDefenseService_PredictAttackPath(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	targetID := "target_server_1"

	paths, err := service.PredictAttackPath(ctx, targetID)
	if err != nil {
		t.Fatalf("PredictAttackPath failed: %v", err)
	}

	if len(paths) == 0 {
		t.Error("Should predict at least one attack path")
	}

	for _, path := range paths {
		if path.PathID == "" {
			t.Error("PathID should be set")
		}

		if path.TargetID != targetID {
			t.Errorf("TargetID should be %s, got %s", targetID, path.TargetID)
		}

		if len(path.Steps) == 0 {
			t.Error("Should have at least one step")
		}

		if path.Probability == 0 {
			t.Error("Probability should not be zero")
		}

		if path.Complexity == 0 {
			t.Error("Complexity should not be zero")
		}

		if path.TimeToCompromise == 0 {
			t.Error("TimeToCompromise should be set")
		}

		if len(path.CriticalNodes) == 0 {
			t.Error("Should identify critical nodes")
		}
	}
}

func TestActiveDefenseService_GenerateCountermeasures(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	testCases := []struct {
		threatType string
		severity   string
	}{
		{"scanning", "medium"},
		{"brute_force", "high"},
		{"exploitation", "critical"},
	}

	for _, tc := range testCases {
		threat := &ThreatInfoV2{
			ThreatID:    "threat_" + tc.threatType,
			ThreatType:  tc.threatType,
			Severity:    tc.severity,
			Description: "Test threat",
		}

		countermeasures, err := service.GenerateCountermeasures(ctx, threat)
		if err != nil {
			t.Fatalf("GenerateCountermeasures failed for %s: %v", tc.threatType, err)
		}

		if len(countermeasures) == 0 {
			t.Errorf("Should generate countermeasures for %s", tc.threatType)
		}
	}

	_, err := service.GenerateCountermeasures(ctx, nil)
	if err == nil {
		t.Error("Should return error for nil threat")
	}
}

func TestActiveDefenseService_ExecuteCountermeasure(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	result, err := service.ExecuteCountermeasure(ctx, "block_ip")
	if err != nil {
		t.Fatalf("ExecuteCountermeasure failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if !result.Success {
		t.Error("Execution should be successful")
	}

	if result.CountermeasureID != "block_ip" {
		t.Errorf("CountermeasureID should be block_ip, got %s", result.CountermeasureID)
	}

	if result.ExecutionTime == 0 {
		t.Error("ExecutionTime should be set")
	}

	if result.Impact == "" {
		t.Error("Impact should be set")
	}

	if !result.RollbackAvailable {
		t.Error("block_ip countermeasure should be rollbackable")
	}

	_, err = service.ExecuteCountermeasure(ctx, "nonexistent")
	if err != ErrCountermeasureNotFound {
		t.Errorf("Should return ErrCountermeasureNotFound, got %v", err)
	}
}

func TestActiveDefenseService_AssessThreatLevel(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	testCases := []struct {
		threatType   string
		severity     string
		expectedRisk string
	}{
		{"scanning", "low", "low"},
		{"brute_force", "medium", "medium"},
		{"exploitation", "high", "high"},
		{"apt_attack", "critical", "critical"},
	}

	for _, tc := range testCases {
		threat := &ThreatInfoV2{
			ThreatID:    "threat_" + tc.threatType,
			ThreatType:  tc.threatType,
			Severity:    tc.severity,
			Description: "Test threat for " + tc.threatType,
		}

		assessment, err := service.AssessThreatLevel(ctx, threat)
		if err != nil {
			t.Fatalf("AssessThreatLevel failed for %s: %v", tc.threatType, err)
		}

		if assessment.ThreatType != threat.ThreatType {
			t.Errorf("ThreatType mismatch for %s", tc.threatType)
		}

		if assessment.Severity != tc.severity {
			t.Errorf("Severity should be %s, got %s", tc.severity, assessment.Severity)
		}

		if assessment.Likelihood == 0 {
			t.Errorf("Likelihood should not be zero for %s", tc.threatType)
		}

		if assessment.Impact == 0 {
			t.Errorf("Impact should not be zero for %s", tc.threatType)
		}

		if assessment.RiskScore == 0 {
			t.Errorf("RiskScore should not be zero for %s", tc.threatType)
		}

		if assessment.RiskLevel != tc.expectedRisk {
			t.Errorf("RiskLevel should be %s, got %s", tc.expectedRisk, assessment.RiskLevel)
		}

		if len(assessment.Mitigations) == 0 {
			t.Errorf("Should have mitigations for %s", tc.threatType)
		}

		if assessment.ResidualRisk == 0 {
			t.Errorf("ResidualRisk should be calculated for %s", tc.threatType)
		}
	}

	_, err := service.AssessThreatLevel(ctx, nil)
	if err == nil {
		t.Error("Should return error for nil threat")
	}
}

func TestActiveDefenseService_GetDefenseMetrics(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	metrics, err := service.GetDefenseMetrics(ctx)
	if err != nil {
		t.Fatalf("GetDefenseMetrics failed: %v", err)
	}

	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	if metrics.TotalHoneypots < 0 {
		t.Error("TotalHoneypots should be non-negative")
	}

	if metrics.ActiveHoneypots < 0 {
		t.Error("ActiveHoneypots should be non-negative")
	}

	if metrics.TotalIntruders < 0 {
		t.Error("TotalIntruders should be non-negative")
	}

	if metrics.BlockedAttacks < 0 {
		t.Error("BlockedAttacks should be non-negative")
	}

	if metrics.DeceptionElements < 0 {
		t.Error("DeceptionElements should be non-negative")
	}

	if metrics.AttackPatternsDetected < 0 {
		t.Error("AttackPatternsDetected should be non-negative")
	}

	if metrics.CountermeasuresDeployed < 0 {
		t.Error("CountermeasuresDeployed should be non-negative")
	}

	if metrics.DefenseCoverage < 0 || metrics.DefenseCoverage > 100 {
		t.Error("DefenseCoverage should be between 0 and 100")
	}

	if metrics.MetricsByType == nil {
		t.Error("MetricsByType should be initialized")
	}

	if metrics.TopThreatActors == nil {
		t.Error("TopThreatActors should not be nil")
	}

	if metrics.RecentActivities == nil {
		t.Error("RecentActivities should not be nil")
	}
}

func TestActiveDefenseService_IntruderClassification(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	testCases := []struct {
		name           string
		tactics        []string
		tools          []string
		successRate    float64
		expectedClass  string
	}{
		{"opportunistic", []string{}, []string{}, 0.5, "opportunistic"},
		{"script_kiddie", []string{"scanning"}, []string{"nmap"}, 0.3, "script_kiddie"},
		{"skilled", []string{"exploitation"}, []string{"metasploit", "sqlmap"}, 0.8, "skilled"},
		{"sophisticated", []string{"initial_access", "execution", "persistence", "privilege_escalation"}, []string{"custom_tool1", "custom_tool2", "custom_tool3", "custom_tool4"}, 0.9, "sophisticated"},
	}

	for _, tc := range testCases {
		intruder := &IntruderInfo{
			IPAddress:    "192.168.1." + string(rune('0'+len(tc.tools))),
			Tactics:      tc.tactics,
			Tools:        tc.tools,
			SuccessRate:  tc.successRate,
			ThreatLevel:  "medium",
			ActivityCount: 50,
		}

		profile, err := service.TrackIntruder(ctx, intruder)
		if err != nil {
			t.Fatalf("TrackIntruder failed for %s: %v", tc.name, err)
		}

		if profile.Classification != tc.expectedClass {
			t.Errorf("For %s, expected classification %s, got %s", tc.name, tc.expectedClass, profile.Classification)
		}
	}
}

func TestActiveDefenseService_DefaultCountermeasures(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	expectedCountermeasures := []string{"block_ip", "rate_limit", "honeypot_redirect", "alert_only"}

	for _, cmID := range expectedCountermeasures {
		cm, err := service.ExecuteCountermeasure(ctx, cmID)
		if err != nil {
			t.Errorf("Default countermeasure %s should be available: %v", cmID, err)
		}

		if cm == nil {
			t.Errorf("Countermeasure %s should not be nil", cmID)
		}
	}
}

func TestActiveDefenseService_HoneypotMetrics(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	config := &HoneypotConfig{Name: "Metrics Test", Port: 443}
	honeypotID, _ := service.CreateHoneypot(ctx, config)

	metrics, _ := service.GetDefenseMetrics(ctx)
	initialCount := metrics.TotalHoneypots

	service.ActivateHoneypot(ctx, honeypotID)

	metrics, _ = service.GetDefenseMetrics(ctx)
	if metrics.TotalHoneypots != initialCount {
		t.Errorf("TotalHoneypots should remain %d, got %d", initialCount, metrics.TotalHoneypots)
	}

	if metrics.ActiveHoneypots != 1 {
		t.Errorf("ActiveHoneypots should be 1, got %d", metrics.ActiveHoneypots)
	}

	service.DeactivateHoneypot(ctx, honeypotID)

	metrics, _ = service.GetDefenseMetrics(ctx)
	if metrics.ActiveHoneypots != 0 {
		t.Errorf("ActiveHoneypots should be 0, got %d", metrics.ActiveHoneypots)
	}
}

func TestActiveDefenseService_AttackPatternAnalysis(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	patterns := []*AttackPattern{
		{AttackVector: "sql_injection", Frequency: 15.0, Similarity: 0.95, Indicators: []*AttackIndicator{{Weight: 0.9}}},
		{AttackVector: "xss", Frequency: 8.0, Similarity: 0.75, Indicators: []*AttackIndicator{{Weight: 0.7}}},
		{AttackVector: "brute_force", Frequency: 20.0, Similarity: 0.85, Indicators: []*AttackIndicator{{Weight: 0.8}}},
	}

	for _, pattern := range patterns {
		analysis, err := service.AnalyzeAttackPattern(ctx, pattern)
		if err != nil {
			t.Fatalf("AnalyzeAttackPattern failed: %v", err)
		}

		if analysis.PatternType == "" {
			t.Error("PatternType should be classified")
		}

		if analysis.Confidence == 0 {
			t.Error("Confidence should be calculated")
		}

		if analysis.ThreatActor == "" {
			t.Error("ThreatActor should be identified")
		}

		if len(analysis.TTPs) == 0 {
			t.Error("Should have MITRE ATT&CK TTPs")
		}
	}
}

func TestActiveDefenseService_DefenseActivities(t *testing.T) {
	service := NewActiveDefenseService()
	ctx := context.Background()

	config := &HoneypotConfig{Name: "Activity Test", Port: 80}
	honeypotID, _ := service.CreateHoneypot(ctx, config)
	service.ActivateHoneypot(ctx, honeypotID)

	intruder := &IntruderInfo{
		IPAddress:   "10.0.0.100",
		ThreatLevel: "high",
		Tools:       []string{"scanner"},
	}
	service.TrackIntruder(ctx, intruder)

	countermeasure, _ := service.GenerateCountermeasures(ctx, &ThreatInfoV2{
		ThreatID:   "threat_1",
		ThreatType: "scanning",
		Severity:   "medium",
	})
	if len(countermeasure) > 0 {
		service.ExecuteCountermeasure(ctx, countermeasure[0].CountermeasureID)
	}

	metrics, err := service.GetDefenseMetrics(ctx)
	if err != nil {
		t.Fatalf("GetDefenseMetrics failed: %v", err)
	}

	if len(metrics.RecentActivities) == 0 {
		t.Error("Should have recent activities")
	}
}
