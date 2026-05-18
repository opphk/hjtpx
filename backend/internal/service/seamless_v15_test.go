package service

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewSeamlessV15Service(t *testing.T) {
	svc := NewSeamlessV15Service()
	if svc == nil {
		t.Fatal("NewSeamlessV15Service returned nil")
	}

	if svc.deviceFingerprintLearner == nil {
		t.Error("deviceFingerprintLearner is nil")
	}

	if svc.behaviorModeler == nil {
		t.Error("behaviorModeler is nil")
	}

	if svc.trustScoreEngine == nil {
		t.Error("trustScoreEngine is nil")
	}

	if svc.switchController == nil {
		t.Error("switchController is nil")
	}

	if svc.reportGenerator == nil {
		t.Error("reportGenerator is nil")
	}
}

func TestDeviceFingerprintLearning(t *testing.T) {
	svc := NewSeamlessV15Service()

	fingerprint := "test_fingerprint_123"
	components := map[string]string{
		"canvas": "canvas_hash_123",
		"webgl": "webgl_hash_456",
		"audio": "audio_hash_789",
	}

	usage := &FingerprintUsage{
		Timestamp:  time.Now(),
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0 Test Browser",
		RiskScore:  30.0,
		Success:    true,
		BehaviorHash: "behavior_hash",
	}

	svc.LearnDeviceFingerprint(fingerprint, components, usage)

	model := svc.deviceFingerprintLearner.getFingerprintModel(fingerprint)
	if model == nil {
		t.Fatal("Fingerprint model not found after learning")
	}

	if model.StabilityScore <= 0 {
		t.Error("Stability score should be positive")
	}

	if model.ConfidenceLevel <= 0 {
		t.Error("Confidence level should be positive")
	}

	if model.SuccessfulVerifies != 1 {
		t.Errorf("Expected 1 successful verify, got %d", model.SuccessfulVerifies)
	}

	if len(model.ComponentHashes) != len(components) {
		t.Errorf("Expected %d component hashes, got %d", len(components), len(model.ComponentHashes))
	}
}

func TestStabilityScoreCalculation(t *testing.T) {
	svc := NewSeamlessV15Service()

	testCases := []struct {
		name            string
		usageHistory    []*FingerprintUsage
		expectedMinScore float64
		expectedMaxScore float64
	}{
		{
			name:             "No usage history",
			usageHistory:     []*FingerprintUsage{},
			expectedMinScore: 0.4,
			expectedMaxScore: 0.6,
		},
		{
			name: "Single IP usage",
			usageHistory: []*FingerprintUsage{
				{IPAddress: "192.168.1.1", UserAgent: "Mozilla/5.0", Success: true},
			},
			expectedMinScore: 0.3,
			expectedMaxScore: 1.0,
		},
		{
			name: "Multiple IPs with failures",
			usageHistory: []*FingerprintUsage{
				{IPAddress: "192.168.1.1", UserAgent: "Mozilla/5.0", Success: true},
				{IPAddress: "192.168.1.2", UserAgent: "Mozilla/5.0", Success: true},
				{IPAddress: "10.0.0.1", UserAgent: "Chrome", Success: false},
			},
			expectedMinScore: 0.0,
			expectedMaxScore: 0.8,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fingerprint := "test_" + tc.name
			if len(tc.usageHistory) == 0 {
				svc.LearnDeviceFingerprint(fingerprint, nil, nil)
			}
			for _, usage := range tc.usageHistory {
				svc.LearnDeviceFingerprint(fingerprint, nil, usage)
			}

			model := svc.deviceFingerprintLearner.getFingerprintModel(fingerprint)
			if model == nil {
				t.Fatalf("Model not found for %s", tc.name)
			}

			if model.StabilityScore < tc.expectedMinScore || model.StabilityScore > tc.expectedMaxScore {
				t.Errorf("Stability score %f not in expected range [%f, %f]",
					model.StabilityScore, tc.expectedMinScore, tc.expectedMaxScore)
			}
		})
	}
}

func TestUserBehaviorModeling(t *testing.T) {
	svc := NewSeamlessV15Service()

	userID := "test_user_123"
	sessionData := &SessionPattern{
		SessionID:      "session_001",
		StartTime:      time.Now(),
		Duration:       5 * time.Minute,
		MouseMoves:     150,
		KeyboardEvents: 200,
		Clicks:         30,
		ScrollEvents:   20,
		AverageSpeed:   0.5,
		Outcome:        "success",
	}

	svc.ModelUserBehavior(userID, sessionData)

	model := svc.behaviorModeler.getUserBehaviorModel(userID)
	if model == nil {
		t.Fatal("User behavior model not found after modeling")
	}

	if model.TotalSessions != 1 {
		t.Errorf("Expected 1 total session, got %d", model.TotalSessions)
	}

	if model.SuccessfulSessions != 1 {
		t.Errorf("Expected 1 successful session, got %d", model.SuccessfulSessions)
	}

	if model.TypingProfile == nil {
		t.Error("Typing profile should be initialized")
	}

	if model.MouseProfile == nil {
		t.Error("Mouse profile should be initialized")
	}
}

func TestBehaviorEntropyCalculation(t *testing.T) {
	svc := NewSeamlessV15Service()

	userID := "entropy_test_user"

	for i := 0; i < 10; i++ {
		session := &SessionPattern{
			SessionID:      "session_" + string(rune('0'+i)),
			StartTime:      time.Now().Add(time.Duration(i) * time.Hour),
			Duration:       5 * time.Minute,
			MouseMoves:     100,
			KeyboardEvents: 150,
			Outcome:        "success",
		}
		svc.ModelUserBehavior(userID, session)
	}

	model := svc.behaviorModeler.getUserBehaviorModel(userID)
	if model == nil {
		t.Fatal("Model not found")
	}

	if model.BehavioralEntropy < 0 || model.BehavioralEntropy > 1 {
		t.Errorf("Entropy should be between 0 and 1, got %f", model.BehavioralEntropy)
	}
}

func TestHabitStrengthCalculation(t *testing.T) {
	svc := NewSeamlessV15Service()

	userID := "habit_test_user"

	for i := 0; i < 5; i++ {
		session := &SessionPattern{
			SessionID: "session_" + string(rune('0'+i)),
			StartTime: time.Date(2024, 1, 1, 10+i, 0, 0, 0, time.UTC),
			Duration:  5 * time.Minute,
			Outcome:   "success",
		}
		svc.ModelUserBehavior(userID, session)
	}

	model := svc.behaviorModeler.getUserBehaviorModel(userID)
	if model == nil {
		t.Fatal("Model not found")
	}

	if model.HabitStrength < 0 || model.HabitStrength > 1 {
		t.Errorf("Habit strength should be between 0 and 1, got %f", model.HabitStrength)
	}
}

func TestTrustScoreCalculation(t *testing.T) {
	svc := NewSeamlessV15Service()

	fingerprint := "trust_test_fp"
	userID := "trust_test_user"

	svc.LearnDeviceFingerprint(fingerprint, map[string]string{
		"canvas": "hash1",
		"webgl":  "hash2",
	}, &FingerprintUsage{
		Timestamp: time.Now(),
		Success:   true,
		RiskScore: 20,
	})

	svc.ModelUserBehavior(userID, &SessionPattern{
		SessionID: "sess1",
		StartTime: time.Now(),
		Duration:  5 * time.Minute,
		Outcome:   "success",
	})

	trustScore := svc.CalculateTrustScore(userID, fingerprint, 0.7, 0.6, 0.8)

	if trustScore < 0 || trustScore > 1 {
		t.Errorf("Trust score should be between 0 and 1, got %f", trustScore)
	}
}

func TestTrustScoreCaching(t *testing.T) {
	svc := NewSeamlessV15Service()

	fingerprint := "cache_test_fp"
	userID := "cache_test_user"

	score1 := svc.CalculateTrustScore(userID, fingerprint, 0.5, 0.5, 0.5)
	score2 := svc.CalculateTrustScore(userID, fingerprint, 0.5, 0.5, 0.5)

	if score1 != score2 {
		t.Errorf("Cached scores should be equal: %f != %f", score1, score2)
	}

	cacheKey := userID + ":" + fingerprint
	cached, exists := svc.trustScoreEngine.scoreCache[cacheKey]
	if !exists {
		t.Error("Score should be cached")
	}

	if !time.Now().Before(cached.ExpiresAt) {
		t.Error("Cache should not be expired")
	}
}

func TestVerificationDecision(t *testing.T) {
	svc := NewSeamlessV15Service()

	testCases := []struct {
		name           string
		userID         string
		fingerprint    string
		baseRiskScore  float64
		expectedTypes  []string
	}{
		{
			name:          "New device high risk",
			userID:        "new_user",
			fingerprint:   "new_device",
			baseRiskScore: 85,
			expectedTypes: []string{"block", "strong"},
		},
		{
			name:          "Trusted device low risk",
			userID:        "trusted_user",
			fingerprint:   "trusted_device",
			baseRiskScore: 20,
			expectedTypes: []string{"seamless", "strong", "progressive"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.fingerprint == "trusted_device" {
				for i := 0; i < 10; i++ {
					svc.LearnDeviceFingerprint(tc.fingerprint, nil, &FingerprintUsage{
						Timestamp: time.Now(),
						Success:   true,
						RiskScore: 10,
					})
				}
			}

			decision := svc.DetermineVerificationType(tc.userID, tc.fingerprint, tc.baseRiskScore)

			validType := false
			for _, expected := range tc.expectedTypes {
				if decision.RecommendedType == expected {
					validType = true
					break
				}
			}

			if !validType {
				t.Errorf("Expected type in %v, got %s", tc.expectedTypes, decision.RecommendedType)
			}
		})
	}
}

func TestProgressiveLevelCalculation(t *testing.T) {
	svc := NewSeamlessV15Service()

	testCases := []struct {
		trustScore float64
		riskScore  float64
		expectedMin int
		expectedMax int
	}{
		{0.9, 10, 0, 0},
		{0.7, 30, 0, 1},
		{0.5, 50, 1, 2},
		{0.3, 70, 2, 3},
	}

	for _, tc := range testCases {
		level := svc.calculateProgressiveLevel(tc.trustScore, tc.riskScore)

		if level < tc.expectedMin || level > tc.expectedMax {
			t.Errorf("For trust=%.2f, risk=%.2f: expected level %d-%d, got %d",
				tc.trustScore, tc.riskScore, tc.expectedMin, tc.expectedMax, level)
		}
	}
}

func TestReportGeneration(t *testing.T) {
	svc := NewSeamlessV15Service()

	for i := 0; i < 20; i++ {
		fp := "fp_" + string(rune('0'+i/10)) + string(rune('0'+i%10))
		for j := 0; j < 3; j++ {
			svc.LearnDeviceFingerprint(fp, nil, &FingerprintUsage{
				Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
				Success:   j > 0,
				RiskScore: float64(20 + j*10),
			})
		}

		userID := "user_" + string(rune('0'+i/10)) + string(rune('0'+i%10))
		for j := 0; j < 3; j++ {
			svc.ModelUserBehavior(userID, &SessionPattern{
				SessionID: "sess_" + fp + "_" + string(rune('0'+j)),
				StartTime: time.Now().Add(-time.Duration(i) * time.Hour),
				Duration:  5 * time.Minute,
				Outcome:   "success",
			})
		}

		svc.DetermineVerificationType(userID, fp, float64(20+i%30))
	}

	periodStart := time.Now().AddDate(0, 0, -7)
	periodEnd := time.Now()

	report := svc.GenerateReport(periodStart, periodEnd)

	if report == nil {
		t.Fatal("Report should not be nil")
	}

	if report.ReportID == "" {
		t.Error("Report ID should not be empty")
	}

	if report.Summary == nil {
		t.Error("Summary should not be nil")
	}

	if report.DeviceAnalysis == nil {
		t.Error("Device analysis should not be nil")
	}

	if report.BehaviorAnalysis == nil {
		t.Error("Behavior analysis should not be nil")
	}

	if report.TrustAnalysis == nil {
		t.Error("Trust analysis should not be nil")
	}

	if report.SwitchAnalysis == nil {
		t.Error("Switch analysis should not be nil")
	}

	if len(report.Recommendations) == 0 {
		t.Error("Should have at least one recommendation")
	}
}

func TestBehaviorDataUpdate(t *testing.T) {
	svc := NewSeamlessV15Service()

	behaviorData := &BehaviorUpdateData{
		SessionID:     "update_test_session",
		Timestamp:     time.Now(),
		Duration:      3000,
		MouseMoves:    100,
		KeyboardEvents: 150,
		Clicks:        25,
		ScrollEvents:  10,
		AverageSpeed:  0.4,
		RiskScore:     25,
		Success:       true,
		BehaviorHash: "test_behavior_hash",
		Fingerprint:   "update_test_fp",
		FingerprintComponents: map[string]string{
			"canvas": "hash1",
			"webgl":  "hash2",
		},
		IPAddress: "192.168.1.100",
		UserAgent: "Mozilla/5.0 Test",
	}

	err := svc.UpdateBehaviorData("update_test_user", behaviorData)
	if err != nil {
		t.Fatalf("UpdateBehaviorData failed: %v", err)
	}

	trustResult := svc.GetTrustScore("update_test_user", "update_test_fp")
	if trustResult == nil {
		t.Fatal("Trust score result should not be nil")
	}

	if trustResult.TrustScore <= 0 {
		t.Error("Trust score should be positive after update")
	}
}

func TestGetTrustScore(t *testing.T) {
	svc := NewSeamlessV15Service()

	for i := 0; i < 5; i++ {
		fp := "ts_fp_" + string(rune('0'+i))
		for j := 0; j < 3; j++ {
			svc.LearnDeviceFingerprint(fp, nil, &FingerprintUsage{
				Timestamp: time.Now(),
				Success:   true,
				RiskScore: 20,
			})
		}

		userID := "ts_user_" + string(rune('0'+i))
		for j := 0; j < 2; j++ {
			svc.ModelUserBehavior(userID, &SessionPattern{
				SessionID: "sess_" + fp + "_" + string(rune('0'+j)),
				StartTime: time.Now(),
				Duration:  5 * time.Minute,
				Outcome:  "success",
			})
		}
	}

	result := svc.GetTrustScore("ts_user_2", "ts_fp_2")

	if result == nil {
		t.Fatal("GetTrustScore returned nil")
	}

	if result.UserID != "ts_user_2" {
		t.Errorf("Expected user ID 'ts_user_2', got '%s'", result.UserID)
	}

	if result.Fingerprint != "ts_fp_2" {
		t.Errorf("Expected fingerprint 'ts_fp_2', got '%s'", result.Fingerprint)
	}

	if result.TrustScore < 0 || result.TrustScore > 1 {
		t.Errorf("Trust score should be between 0 and 1, got %f", result.TrustScore)
	}
}

func TestPerformSeamlessVerification(t *testing.T) {
	svc := NewSeamlessV15Service()

	for i := 0; i < 10; i++ {
		svc.LearnDeviceFingerprint("verified_fp", nil, &FingerprintUsage{
			Timestamp: time.Now(),
			Success:   true,
			RiskScore: 15,
		})
	}

	svc.ModelUserBehavior("verified_user", &SessionPattern{
		SessionID: "verified_session",
		StartTime: time.Now(),
		Duration:  5 * time.Minute,
		Outcome:   "success",
	})

	result := svc.PerformSeamlessVerification("verified_user", "verified_fp", 25)

	if result == nil {
		t.Fatal("Verification result should not be nil")
	}

	if result.VerificationType == "" {
		t.Error("Verification type should not be empty")
	}

	if result.TrustScore < 0 || result.TrustScore > 1 {
		t.Errorf("Trust score should be between 0 and 1, got %f", result.TrustScore)
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", result.Confidence)
	}

	if result.VerificationType != "block" && result.Token == "" {
		t.Error("Token should be generated for non-blocked verifications")
	}
}

func TestGlobalStats(t *testing.T) {
	svc := NewSeamlessV15Service()

	for i := 0; i < 5; i++ {
		fp := "stats_fp_" + string(rune('0'+i))
		for j := 0; j < 3; j++ {
			svc.LearnDeviceFingerprint(fp, nil, &FingerprintUsage{
				Timestamp: time.Now(),
				Success:   true,
			})
		}

		userID := "stats_user_" + string(rune('0'+i))
		svc.ModelUserBehavior(userID, &SessionPattern{
			SessionID: "sess_" + fp,
			StartTime: time.Now(),
			Duration:  5 * time.Minute,
			Outcome:   "success",
		})

		svc.PerformSeamlessVerification(userID, fp, 30)
	}

	stats := svc.GetGlobalStats()

	if stats == nil {
		t.Fatal("GetGlobalStats returned nil")
	}

	if stats["total_devices"].(int) != 5 {
		t.Errorf("Expected 5 devices, got %d", stats["total_devices"].(int))
	}

	if stats["total_users"].(int) != 5 {
		t.Errorf("Expected 5 users, got %d", stats["total_users"].(int))
	}
}

func TestExportImportModelData(t *testing.T) {
	svc := NewSeamlessV15Service()

	for i := 0; i < 3; i++ {
		fp := "export_fp_" + string(rune('0'+i))
		svc.LearnDeviceFingerprint(fp, map[string]string{"canvas": "hash"}, &FingerprintUsage{
			Timestamp: time.Now(),
			Success:   true,
		})

		userID := "export_user_" + string(rune('0'+i))
		svc.ModelUserBehavior(userID, &SessionPattern{
			SessionID: "sess_" + fp,
			StartTime: time.Now(),
			Duration:  5 * time.Minute,
			Outcome:   "success",
		})
	}

	jsonData, err := svc.ExportModelData()
	if err != nil {
		t.Fatalf("ExportModelData failed: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Exported data should not be empty")
	}

	var exported map[string]interface{}
	if err := json.Unmarshal(jsonData, &exported); err != nil {
		t.Fatalf("Failed to parse exported JSON: %v", err)
	}

	if exported["exported_at"] == nil {
		t.Error("Exported data should contain exported_at timestamp")
	}

	svc2 := NewSeamlessV15Service()
	if err := svc2.ImportModelData(jsonData); err != nil {
		t.Fatalf("ImportModelData failed: %v", err)
	}

	stats1 := svc.GetGlobalStats()
	stats2 := svc2.GetGlobalStats()

	if stats1["total_devices"].(int) != stats2["total_devices"].(int) {
		t.Error("Device count should match after import")
	}

	if stats1["total_users"].(int) != stats2["total_users"].(int) {
		t.Error("User count should match after import")
	}
}

func TestCleanupOldData(t *testing.T) {
	svc := NewSeamlessV15Service()

	for i := 0; i < 5; i++ {
		fp := "cleanup_fp_" + string(rune('0'+i))
		model := &FingerprintModel{
			Fingerprint:    fp,
			LastUpdatedAt: time.Now().Add(-time.Duration(i*24) * time.Hour),
		}
		svc.deviceFingerprintLearner.fingerprintModels[fp] = model

		userID := "cleanup_user_" + string(rune('0'+i))
		userModel := &UserBehaviorModel{
			UserID:        userID,
			LastUpdatedAt: time.Now().Add(-time.Duration(i*24) * time.Hour),
		}
		svc.behaviorModeler.userModels[userID] = userModel
	}

	cleaned := svc.CleanupOldData(2)

	if cleaned == 0 {
		t.Error("Should have cleaned some data")
	}

	remainingDevices := len(svc.deviceFingerprintLearner.fingerprintModels)
	remainingUsers := len(svc.behaviorModeler.userModels)

	if remainingDevices != 2 {
		t.Errorf("Expected 2 remaining devices, got %d", remainingDevices)
	}

	if remainingUsers != 2 {
		t.Errorf("Expected 2 remaining users, got %d", remainingUsers)
	}
}

func TestFingerprintSimilarityThreshold(t *testing.T) {
	svc := NewSeamlessV15Service()

	threshold := svc.deviceFingerprintLearner.learningConfig.SimilarityThreshold
	if threshold != 0.85 {
		t.Errorf("Expected similarity threshold 0.85, got %f", threshold)
	}
}

func TestAdaptiveScoringParams(t *testing.T) {
	svc := NewSeamlessV15Service()

	params := svc.trustScoreEngine.adaptiveParams

	if params.BaseTrustLevel != 0.5 {
		t.Errorf("Expected base trust level 0.5, got %f", params.BaseTrustLevel)
	}

	if params.MinTrustThreshold >= params.MaxTrustThreshold {
		t.Error("Min threshold should be less than max threshold")
	}
}

func TestSwitchControllerConfig(t *testing.T) {
	svc := NewSeamlessV15Service()

	config := svc.switchController.strategyConfig

	if config.SeamlessThreshold <= config.StrongThreshold {
		t.Error("Seamless threshold should be greater than strong threshold")
	}

	if !config.EnableProgressive {
		t.Error("Progressive verification should be enabled")
	}

	if config.ForceStrongOnNew != true {
		t.Error("Force strong verification on new device should be enabled")
	}
}

func TestReportRetention(t *testing.T) {
	svc := NewSeamlessV15Service()

	retentionDays := svc.reportGenerator.reportConfig.RetentionDays
	if retentionDays != 90 {
		t.Errorf("Expected retention days 90, got %d", retentionDays)
	}
}
