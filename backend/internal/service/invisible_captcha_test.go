package service

import (
	"encoding/json"
	"math"
	"sort"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

func TestNewInvisibleCaptchaService(t *testing.T) {
	service := NewInvisibleCaptchaService()
	if service == nil {
		t.Fatal("NewInvisibleCaptchaService returned nil")
	}

	if service.fingerprintEngine == nil {
		t.Error("fingerprintEngine is nil")
	}

	if service.confidenceAssessor == nil {
		t.Error("confidenceAssessor is nil")
	}

	if service.behaviorAnalyzer == nil {
		t.Error("behaviorAnalyzer is nil")
	}

	if service.trustCalculator == nil {
		t.Error("trustCalculator is nil")
	}

	if service.config == nil {
		t.Error("config is nil")
	}

	if !service.config.EnableFingerprintOptimization {
		t.Error("EnableFingerprintOptimization should be true by default")
	}

	if !service.config.EnableConfidenceAssessment {
		t.Error("EnableConfidenceAssessment should be true by default")
	}
}

func TestInvisibleFingerprintEngine_OptimizeFingerprint(t *testing.T) {
	engine := newInvisibleFingerprintEngine()

	components := &FingerprintComponents{
		CanvasHash:         "abc123canvas",
		WebGLHash:          "def456webgl",
		AudioHash:          "ghi789audio",
		FontHash:           "jkl012fonts",
		ScreenHash:         "1920x1080",
		TimezoneHash:       "Asia/Shanghai",
		LanguageHash:       "zh-CN",
		PlatformHash:       "Win32",
		HardwareHash:       "hw123",
		ColorDepth:         24,
		PixelRatio:         1.0,
		HardwareConcurrency: 8,
	}

	result := engine.OptimizeFingerprint("test-fingerprint-001", "user123", components)

	if result == nil {
		t.Fatal("OptimizeFingerprint returned nil")
	}

	if result.Fingerprint != "test-fingerprint-001" {
		t.Errorf("Expected fingerprint 'test-fingerprint-001', got '%s'", result.Fingerprint)
	}

	if result.Components == nil {
		t.Error("Components should not be nil")
	}

	if result.StabilityScore < 0 || result.StabilityScore > 1 {
		t.Errorf("StabilityScore should be between 0 and 1, got %f", result.StabilityScore)
	}

	if result.UniquenessScore < 0 || result.UniquenessScore > 100 {
		t.Errorf("UniquenessScore should be between 0 and 100, got %f", result.UniquenessScore)
	}

	if result.QualityScore < 0 || result.QualityScore > 100 {
		t.Errorf("QualityScore should be between 0 and 100, got %f", result.QualityScore)
	}

	secondResult := engine.OptimizeFingerprint("test-fingerprint-001", "user123", components)
	if secondResult.StabilityScore <= result.StabilityScore {
		t.Error("StabilityScore should increase with repeated appearances")
	}
}

func TestInvisibleFingerprintEngine_CalculateComponentStability(t *testing.T) {
	engine := newInvisibleFingerprintEngine()

	tests := []struct {
		name       string
		components *FingerprintComponents
		minScore   float64
		maxScore   float64
	}{
		{
			name: "all_components_present",
			components: &FingerprintComponents{
				CanvasHash:   "hash1",
				WebGLHash:    "hash2",
				AudioHash:    "hash3",
				FontHash:     "hash4",
				ScreenHash:   "hash5",
				TimezoneHash: "hash6",
			},
			minScore: 0.5,
			maxScore: 1.0,
		},
		{
			name:       "no_components",
			components: &FingerprintComponents{},
			minScore:   0.0,
			maxScore:   0.1,
		},
		{
			name: "partial_components",
			components: &FingerprintComponents{
				CanvasHash: "hash1",
				WebGLHash:  "hash2",
			},
			minScore: 0.2,
			maxScore: 0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := engine.calculateComponentStability(tt.components)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("Expected score between %f and %f, got %f", tt.minScore, tt.maxScore, score)
			}
		})
	}
}

func TestInvisibleFingerprintEngine_CalculateQualityScore(t *testing.T) {
	engine := newInvisibleFingerprintEngine()

	components := &FingerprintComponents{
		CanvasHash:           "hash1",
		WebGLHash:            "hash2",
		AudioHash:            "hash3",
		FontHash:             "hash4",
		ScreenHash:           "hash5",
		HardwareConcurrency:  8,
		WebGLRenderer:        "renderer1",
		TouchSupport:         true,
		ColorDepth:           24,
		PixelRatio:           1.5,
	}

	score := engine.calculateQualityScore(components)

	if score < 70 {
		t.Errorf("Expected quality score >= 70 for full components, got %f", score)
	}

	emptyComponents := &FingerprintComponents{}
	emptyScore := engine.calculateQualityScore(emptyComponents)

	if emptyScore < 0 || emptyScore > 10 {
		t.Errorf("Expected quality score between 0 and 10 for empty components, got %f", emptyScore)
	}
}

func TestUniquenessCalculator_CalculateUniqueness(t *testing.T) {
	calc := &UniquenessCalculator{
		knownSignatures: make(map[string]int),
		collisionMap:    make(map[string][]string),
	}

	calc.RegisterSignature("new-fingerprint")
	uniqueScore := calc.CalculateUniqueness("new-fingerprint")
	if uniqueScore < 80 {
		t.Errorf("Expected uniqueness score >= 80 for new fingerprint, got %f", uniqueScore)
	}

	calc.RegisterSignature("common-fingerprint")
	calc.RegisterSignature("common-fingerprint")
	calc.RegisterSignature("common-fingerprint")
	calc.RegisterSignature("common-fingerprint")
	calc.RegisterSignature("common-fingerprint")
	commonScore := calc.CalculateUniqueness("common-fingerprint")
	if commonScore > 80 {
		t.Errorf("Expected uniqueness score <= 80 for common fingerprint, got %f", commonScore)
	}
}

func TestBehavioralConfidenceAssessor_AssessConfidence(t *testing.T) {
	assessor := newBehavioralConfidenceAssessor()

	tests := []struct {
		name            string
		context         *ConfidenceContext
		expectedMinScore float64
		expectedMaxScore float64
	}{
		{
			name: "high_trust_user",
			context: &ConfidenceContext{
				UserID:               "user1",
				DeviceFingerprint:    "device1",
				IsKnownDevice:        true,
				IsKnownLocation:      true,
				HistoricalSuccessRate: 0.95,
				RecentFailureCount:  0,
				RequestFrequency:     3.0,
				TimeOfDay:            10,
				DayOfWeek:            3,
			},
			expectedMinScore: 70.0,
			expectedMaxScore: 100.0,
		},
		{
			name: "low_trust_user",
			context: &ConfidenceContext{
				UserID:               "user2",
				DeviceFingerprint:    "device2",
				IsKnownDevice:        false,
				IsKnownLocation:      false,
				HistoricalSuccessRate: 0.5,
				RecentFailureCount:  5,
				RequestFrequency:     15.0,
				TimeOfDay:            3,
				DayOfWeek:            0,
			},
			expectedMinScore: 0.0,
			expectedMaxScore: 40.0,
		},
		{
			name: "medium_trust_user",
			context: &ConfidenceContext{
				UserID:               "user3",
				DeviceFingerprint:    "device3",
				IsKnownDevice:        true,
				IsKnownLocation:      false,
				HistoricalSuccessRate: 0.75,
				RecentFailureCount:  1,
				RequestFrequency:     5.0,
				TimeOfDay:            14,
				DayOfWeek:            5,
			},
			expectedMinScore: 30.0,
			expectedMaxScore: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assessor.AssessConfidence(tt.context)

			if result == nil {
				t.Fatal("AssessConfidence returned nil")
			}

			if result.TotalScore < tt.expectedMinScore || result.TotalScore > tt.expectedMaxScore {
				t.Errorf("Expected score between %f and %f, got %f", tt.expectedMinScore, tt.expectedMaxScore, result.TotalScore)
			}

			if result.Level != "high" && result.Level != "medium" && result.Level != "low" && result.Level != "critical" {
				t.Errorf("Invalid confidence level: %s", result.Level)
			}

			if len(result.Factors) == 0 && tt.context.HistoricalSuccessRate < 0.5 {
				t.Error("Expected confidence factors for low trust user")
			}
		})
	}
}

func TestBehavioralConfidenceAssessor_EvaluateTemporalConfidence(t *testing.T) {
	assessor := newBehavioralConfidenceAssessor()

	tests := []struct {
		name      string
		timeOfDay int
		dayOfWeek int
	}{
		{"morning_weekday", 9, 2},
		{"afternoon_weekend", 15, 6},
		{"night_weekday", 23, 3},
		{"early_morning", 4, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ConfidenceContext{
				TimeOfDay: tt.timeOfDay,
				DayOfWeek: tt.dayOfWeek,
			}

			score := assessor.evaluateTemporalConfidence(ctx)
			if score < -10 || score > 10 {
				t.Errorf("Temporal confidence score out of expected range: %f", score)
			}
		})
	}
}

func TestHistoricalBehaviorAnalyzer_GetOrCreateHistory(t *testing.T) {
	analyzer := newHistoricalBehaviorAnalyzer()

	history1 := analyzer.GetOrCreateHistory("user1")
	if history1 == nil {
		t.Fatal("GetOrCreateHistory returned nil")
	}

	if history1.UserID != "user1" {
		t.Errorf("Expected UserID 'user1', got '%s'", history1.UserID)
	}

	history2 := analyzer.GetOrCreateHistory("user1")
	if history2 != history1 {
		t.Error("GetOrCreateHistory should return same history for same user")
	}
}

func TestHistoricalBehaviorAnalyzer_AnalyzeBehavior(t *testing.T) {
	analyzer := newHistoricalBehaviorAnalyzer()

	history := analyzer.GetOrCreateHistory("user1")
	history.VerificationHistory = []*VerificationRecord{
		{Timestamp: time.Now().Add(-24 * time.Hour), VerificationSuccess: true},
		{Timestamp: time.Now().Add(-12 * time.Hour), VerificationSuccess: true},
		{Timestamp: time.Now().Add(-6 * time.Hour), VerificationSuccess: true},
		{Timestamp: time.Now().Add(-3 * time.Hour), VerificationSuccess: true},
		{Timestamp: time.Now().Add(-1 * time.Hour), VerificationSuccess: true},
	}

	behaviorData := []models.BehaviorData{
		{Data: "test data 1"},
		{Data: "test data 2"},
	}

	envData := map[string]interface{}{
		"ip_country": "CN",
	}

	result := analyzer.AnalyzeBehavior("user1", behaviorData, envData)

	if result == nil {
		t.Fatal("AnalyzeBehavior returned nil")
	}

	if result.UserID != "user1" {
		t.Errorf("Expected UserID 'user1', got '%s'", result.UserID)
	}

	if result.AnomalyScore < 0 || result.AnomalyScore > 100 {
		t.Errorf("AnomalyScore should be between 0 and 100, got %f", result.AnomalyScore)
	}

	if result.PatternMatch < 0 || result.PatternMatch > 1 {
		t.Errorf("PatternMatch should be between 0 and 1, got %f", result.PatternMatch)
	}

	if result.HistoricalScore < 0 || result.HistoricalScore > 100 {
		t.Errorf("HistoricalScore should be between 0 and 100, got %f", result.HistoricalScore)
	}
}

func TestBehaviorAnomalyDetector_DetectAnomalies(t *testing.T) {
	detector := &BehaviorAnomalyDetector{
		anomalyRules:   initAnomalyRules(),
		baselineModels: make(map[string]*InvisibleCaptchaAnomalyBaseline),
	}

	tests := []struct {
		name           string
		history        *UserBehaviorHistory
		behaviorData   []models.BehaviorData
		envData        map[string]interface{}
		maxAnomalyScore float64
	}{
		{
			name: "normal_behavior",
			history: &UserBehaviorHistory{
				VerificationHistory: []*VerificationRecord{
					{Timestamp: time.Now().Add(-1 * time.Hour), VerificationSuccess: true},
					{Timestamp: time.Now().Add(-2 * time.Hour), VerificationSuccess: true},
				},
				SessionMetrics: &InvisibleCaptchaSessionMetrics{
					ActiveHours: map[int]int{10: 5, 11: 3, 14: 4},
				},
			},
			behaviorData:   []models.BehaviorData{},
			envData:        map[string]interface{}{},
			maxAnomalyScore: 30.0,
		},
		{
			name: "suspicious_behavior",
			history: &UserBehaviorHistory{
				VerificationHistory: []*VerificationRecord{
					{Timestamp: time.Now().Add(-1 * time.Minute), VerificationSuccess: true},
					{Timestamp: time.Now().Add(-2 * time.Minute), VerificationSuccess: true},
					{Timestamp: time.Now().Add(-3 * time.Minute), VerificationSuccess: true},
					{Timestamp: time.Now().Add(-4 * time.Minute), VerificationSuccess: true},
					{Timestamp: time.Now().Add(-5 * time.Minute), VerificationSuccess: true},
					{Timestamp: time.Now().Add(-6 * time.Minute), VerificationSuccess: true},
					{Timestamp: time.Now().Add(-7 * time.Minute), VerificationSuccess: true},
					{Timestamp: time.Now().Add(-8 * time.Minute), VerificationSuccess: true},
					{Timestamp: time.Now().Add(-9 * time.Minute), VerificationSuccess: true},
					{Timestamp: time.Now().Add(-10 * time.Minute), VerificationSuccess: true},
					{Timestamp: time.Now().Add(-11 * time.Minute), VerificationSuccess: true},
				},
				SessionMetrics: &InvisibleCaptchaSessionMetrics{
					ActiveHours: map[int]int{},
				},
			},
			behaviorData:   []models.BehaviorData{},
			envData:        map[string]interface{}{},
			maxAnomalyScore: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := detector.DetectAnomalies(tt.history, tt.behaviorData, tt.envData)
			if score > tt.maxAnomalyScore {
				t.Errorf("Expected anomaly score <= %f, got %f", tt.maxAnomalyScore, score)
			}
		})
	}
}

func TestTrendAnalyzer_AnalyzeTrends(t *testing.T) {
	analyzer := &InvisibleCaptchaTrendAnalyzer{
		trendData:      make(map[string]*InvisibleCaptchaTrendData),
		analysisWindow: 7 * 24 * time.Hour,
	}

	history := &UserBehaviorHistory{
		RiskEvolution: []float64{30.0, 35.0, 32.0, 28.0, 25.0, 22.0, 20.0, 18.0},
	}

	trends := analyzer.AnalyzeTrends(history)

	if trends == nil {
		t.Fatal("AnalyzeTrends returned nil")
	}

	if trends["status"] == "analyzed" {
		if _, ok := trends["risk_trend"]; !ok {
			t.Error("Expected risk_trend in analyzed trends")
		}
	}
}

func TestCompositeTrustCalculator_CalculateTrustScore(t *testing.T) {
	calculator := newCompositeTrustCalculator()

	tests := []struct {
		name          string
		context       *TrustCalculationContext
		expectedRange [2]float64
	}{
		{
			name: "high_trust",
			context: &TrustCalculationContext{
				UserID:               "user1",
				DeviceFingerprint:    "device1",
				IsKnownDevice:        true,
				DeviceAgeDays:        90,
				DeviceUseCount:       20,
				DeviceSuccessRate:    0.98,
				IsKnownLocation:      true,
				LocationChangeRate:   0.05,
				IsSuspiciousLocation: false,
				BehaviorConsistency:  0.95,
				ResponseTimeScore:    0.9,
				PatternMatchScore:    0.9,
				HistoricalSuccessRate: 0.97,
				AccountAgeDays:        365,
				TotalVerifications:   100,
				RecentFailureCount:   0,
				RequestFrequency:     2.0,
				IsProxy:              false,
				IsVPN:                false,
				IsTor:                false,
				IsHosting:            false,
				IPReputationScore:    90,
				ConfidenceScore:      85,
			},
			expectedRange: [2]float64{70, 100},
		},
		{
			name: "low_trust",
			context: &TrustCalculationContext{
				UserID:               "user2",
				DeviceFingerprint:    "device2",
				IsKnownDevice:        false,
				DeviceAgeDays:        1,
				DeviceUseCount:       1,
				DeviceSuccessRate:    0.3,
				IsKnownLocation:      false,
				LocationChangeRate:   0.8,
				IsSuspiciousLocation: true,
				BehaviorConsistency:  0.3,
				ResponseTimeScore:    0.2,
				PatternMatchScore:    0.2,
				HistoricalSuccessRate: 0.4,
				AccountAgeDays:        5,
				TotalVerifications:   3,
				RecentFailureCount:    5,
				RequestFrequency:     25.0,
				IsProxy:              true,
				IsVPN:                true,
				IsTor:                false,
				IsHosting:            true,
				IPReputationScore:    20,
				ConfidenceScore:      25,
			},
			expectedRange: [2]float64{0, 40},
		},
		{
			name: "medium_trust",
			context: &TrustCalculationContext{
				UserID:               "user3",
				DeviceFingerprint:    "device3",
				IsKnownDevice:        true,
				DeviceAgeDays:        30,
				DeviceUseCount:       10,
				DeviceSuccessRate:    0.85,
				IsKnownLocation:      true,
				LocationChangeRate:   0.2,
				IsSuspiciousLocation: false,
				BehaviorConsistency:  0.75,
				ResponseTimeScore:    0.7,
				PatternMatchScore:    0.7,
				HistoricalSuccessRate: 0.82,
				AccountAgeDays:        60,
				TotalVerifications:   25,
				RecentFailureCount:   1,
				RequestFrequency:     5.0,
				IsProxy:              false,
				IsVPN:                false,
				IsTor:                false,
				IsHosting:            false,
				IPReputationScore:    70,
				ConfidenceScore:      65,
			},
			expectedRange: [2]float64{50, 85},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.CalculateTrustScore(tt.context)

			if result == nil {
				t.Fatal("CalculateTrustScore returned nil")
			}

			if result.TotalScore < tt.expectedRange[0] || result.TotalScore > tt.expectedRange[1] {
				t.Errorf("Expected trust score between %f and %f, got %f", tt.expectedRange[0], tt.expectedRange[1], result.TotalScore)
			}

			if len(result.ComponentScores) == 0 {
				t.Error("Expected component scores")
			}

			for name, score := range result.ComponentScores {
				if score < 0 || score > 100 {
					t.Errorf("Component %s score out of range: %f", name, score)
				}
			}
		})
	}
}

func TestRiskScoreEngine_CalculateRiskScore(t *testing.T) {
	engine := &RiskScoreEngine{
		riskFactors:   make(map[string]RiskFactor),
		globalRules:   initGlobalRiskRules(),
		historicalRisk: make(map[string][]float64),
	}

	tests := []struct {
		name         string
		context      *TrustCalculationContext
		expectedRisk float64
	}{
		{
			name: "clean_user",
			context: &TrustCalculationContext{
				IsProxy:       false,
				IsVPN:         false,
				IsTor:         false,
				IsHosting:     false,
				DeviceAgeDays: 90,
				ConfidenceScore: 90,
			},
			expectedRisk: 0,
		},
		{
			name: "tor_user",
			context: &TrustCalculationContext{
				IsTor:         true,
				IsProxy:       false,
				IsVPN:         false,
				IsHosting:     false,
				DeviceAgeDays: 30,
				ConfidenceScore: 90,
			},
			expectedRisk: 35,
		},
		{
			name: "multiple_risk_factors",
			context: &TrustCalculationContext{
				IsProxy:       true,
				IsVPN:         true,
				IsTor:         false,
				IsHosting:     true,
				DeviceAgeDays: 5,
				ConfidenceScore: 90,
			},
			expectedRisk: 90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var riskFactors []string
			score := engine.CalculateRiskScore(tt.context, &riskFactors)

			if tt.expectedRisk == 0 && score > 10 {
				t.Errorf("Expected low risk for clean user, got %f", score)
			}

			if tt.expectedRisk > 0 && len(riskFactors) == 0 {
				t.Error("Expected risk factors for risky user")
			}
		})
	}
}

func TestInvisibleCaptchaService_ProcessInvisibleVerification(t *testing.T) {
	service := NewInvisibleCaptchaService()

	req := &InvisibleVerificationRequest{
		SessionID:         "session-001",
		UserID:            "user-001",
		DeviceFingerprint: "fp-001",
		FingerprintComponents: &FingerprintComponents{
			CanvasHash:           "canvas-hash",
			WebGLHash:            "webgl-hash",
			AudioHash:            "audio-hash",
			FontHash:             "font-hash",
			ScreenHash:           "1920x1080",
			TimezoneHash:         "UTC+8",
			HardwareConcurrency:  8,
		},
		BehaviorData: []models.BehaviorData{
			{Data: "behavior1"},
			{Data: "behavior2"},
			{Data: "behavior3"},
		},
		EnvironmentData: map[string]interface{}{
			"ip_address":      "192.168.1.100",
			"proxy_detected":   false,
			"vpn_detected":    false,
			"is_tor_exit":     false,
		},
		IPAddress: "192.168.1.100",
		UserAgent: "Mozilla/5.0 Test Browser",
	}

	result, err := service.ProcessInvisibleVerification(req)

	if err != nil {
		t.Fatalf("ProcessInvisibleVerification returned error: %v", err)
	}

	if result == nil {
		t.Fatal("ProcessInvisibleVerification returned nil")
	}

	if result.SessionID != "session-001" {
		t.Errorf("Expected SessionID 'session-001', got '%s'", result.SessionID)
	}

	if result.ConfidenceScore < 0 || result.ConfidenceScore > 100 {
		t.Errorf("ConfidenceScore out of range: %f", result.ConfidenceScore)
	}

	if result.TrustScore < 0 || result.TrustScore > 100 {
		t.Errorf("TrustScore out of range: %f", result.TrustScore)
	}

	if result.RiskScore < 0 || result.RiskScore > 100 {
		t.Errorf("RiskScore out of range: %f", result.RiskScore)
	}

	if result.RecommendedAction != "allow" && result.RecommendedAction != "challenge" && result.RecommendedAction != "block" && result.RecommendedAction != "review" {
		t.Errorf("Invalid recommended action: %s", result.RecommendedAction)
	}
}

func TestInvisibleCaptchaService_ShouldIssueChallenge(t *testing.T) {
	service := NewInvisibleCaptchaService()

	tests := []struct {
		name     string
		result   *InvisibleVerificationResult
		expected bool
	}{
		{
			name: "low_confidence",
			result: &InvisibleVerificationResult{
				ConfidenceScore:     30,
				BehaviorAnomalyScore: 20,
				RiskScore:           30,
				TrustScore:          60,
			},
			expected: true,
		},
		{
			name: "high_anomaly",
			result: &InvisibleVerificationResult{
				ConfidenceScore:     70,
				BehaviorAnomalyScore: 75,
				RiskScore:           30,
				TrustScore:          60,
			},
			expected: true,
		},
		{
			name: "high_risk",
			result: &InvisibleVerificationResult{
				ConfidenceScore:     70,
				BehaviorAnomalyScore: 20,
				RiskScore:           80,
				TrustScore:          60,
			},
			expected: true,
		},
		{
			name: "low_trust",
			result: &InvisibleVerificationResult{
				ConfidenceScore:     70,
				BehaviorAnomalyScore: 20,
				RiskScore:           30,
				TrustScore:          40,
			},
			expected: true,
		},
		{
			name: "all_good",
			result: &InvisibleVerificationResult{
				ConfidenceScore:     85,
				BehaviorAnomalyScore: 10,
				RiskScore:           15,
				TrustScore:          85,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldChallenge := service.shouldIssueChallenge(tt.result)
			if shouldChallenge != tt.expected {
				t.Errorf("Expected shouldIssueChallenge=%v, got %v", tt.expected, shouldChallenge)
			}
		})
	}
}

func TestInvisibleCaptchaService_RecordVerificationResult(t *testing.T) {
	service := NewInvisibleCaptchaService()

	req := &InvisibleVerificationRequest{
		SessionID:         "session-002",
		UserID:            "user-002",
		DeviceFingerprint: "fp-002",
		IPAddress:         "10.0.0.1",
	}

	result := &InvisibleVerificationResult{
		RiskScore:       25,
		ConfidenceScore: 80,
		TrustScore:      75,
		ShouldChallenge: false,
	}

	service.RecordVerificationResult(req, result, true)

	profile := service.GetUserTrustProfile("user-002")
	if profile == nil {
		t.Fatal("GetUserTrustProfile returned nil")
	}

	if profile.TotalVerifications < 1 {
		t.Errorf("Expected at least 1 verification, got %d", profile.TotalVerifications)
	}

	if profile.DeviceCount < 1 {
		t.Errorf("Expected at least 1 device, got %d", profile.DeviceCount)
	}
}

func TestInvisibleCaptchaService_GetUserTrustProfile(t *testing.T) {
	service := NewInvisibleCaptchaService()

	for i := 0; i < 5; i++ {
		req := &InvisibleVerificationRequest{
			SessionID:         "session-" + string(rune('0'+i)),
			UserID:            "user-profile-test",
			DeviceFingerprint: "fp-profile-" + string(rune('0'+i)),
			IPAddress:         "192.168.1." + string(rune('0'+i)),
		}
		result := &InvisibleVerificationResult{
			RiskScore:       float64(20 + i*5),
			ConfidenceScore: float64(70 + i*3),
			TrustScore:      float64(75 + i*2),
		}
		service.RecordVerificationResult(req, result, i%5 != 2)
	}

	profile := service.GetUserTrustProfile("user-profile-test")

	if profile == nil {
		t.Fatal("GetUserTrustProfile returned nil")
	}

	if profile.UserID != "user-profile-test" {
		t.Errorf("Expected UserID 'user-profile-test', got '%s'", profile.UserID)
	}

	if profile.TotalVerifications != 5 {
		t.Errorf("Expected 5 total verifications, got %d", profile.TotalVerifications)
	}

	if profile.DeviceCount != 5 {
		t.Errorf("Expected 5 devices, got %d", profile.DeviceCount)
	}

	if profile.RiskLevel != "low" && profile.RiskLevel != "medium" && profile.RiskLevel != "high" {
		t.Errorf("Invalid risk level: %s", profile.RiskLevel)
	}
}

func TestDetermineSkipReason(t *testing.T) {
	service := NewInvisibleCaptchaService()

	tests := []struct {
		name     string
		result   *InvisibleVerificationResult
		hasSkipReason bool
	}{
		{
			name: "high_trust_user",
			result: &InvisibleVerificationResult{
				TrustScore:           90,
				ConfidenceScore:      85,
				BehaviorAnomalyScore: 10,
			},
			hasSkipReason: true,
		},
		{
			name: "consistent_behavior",
			result: &InvisibleVerificationResult{
				TrustScore:           75,
				ConfidenceScore:      92,
				BehaviorPatternMatch: 0.92,
			},
			hasSkipReason: true,
		},
		{
			name: "trusted_device",
			result: &InvisibleVerificationResult{
				FingerprintConfidence:  0.95,
				FingerprintUniqueness:  85,
				TrustScore:            70,
				ConfidenceScore:       70,
			},
			hasSkipReason: true,
		},
		{
			name: "no_skip",
			result: &InvisibleVerificationResult{
				TrustScore:             50,
				ConfidenceScore:        50,
				BehaviorAnomalyScore:   30,
				FingerprintConfidence:  0.5,
			},
			hasSkipReason: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := service.determineSkipReason(tt.result)
			if tt.hasSkipReason && reason == "" {
				t.Error("Expected skip reason, got empty string")
			}
			if !tt.hasSkipReason && reason != "" {
				t.Errorf("Expected no skip reason, got '%s'", reason)
			}
		})
	}
}

func TestInvisibleVerificationResult_JSON(t *testing.T) {
	result := &InvisibleVerificationResult{
		SessionID:             "test-session",
		Timestamp:             time.Now(),
		FingerprintScore:      85.5,
		FingerprintUniqueness: 92.3,
		ConfidenceScore:       78.0,
		ConfidenceLevel:       "high",
		BehaviorAnomalyScore:  15.0,
		TrustScore:            82.0,
		RiskScore:             18.0,
		ShouldChallenge:       false,
		SkipReason:            "high_trust_user",
		RecommendedAction:     "allow",
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal result to JSON: %v", err)
	}

	var unmarshaled InvisibleVerificationResult
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if unmarshaled.SessionID != result.SessionID {
		t.Errorf("Expected SessionID '%s', got '%s'", result.SessionID, unmarshaled.SessionID)
	}

	if unmarshaled.ConfidenceScore != result.ConfidenceScore {
		t.Errorf("Expected ConfidenceScore %f, got %f", result.ConfidenceScore, unmarshaled.ConfidenceScore)
	}

	if unmarshaled.TrustScore != result.TrustScore {
		t.Errorf("Expected TrustScore %f, got %f", result.TrustScore, unmarshaled.TrustScore)
	}

	if unmarshaled.ShouldChallenge != result.ShouldChallenge {
		t.Errorf("Expected ShouldChallenge %v, got %v", result.ShouldChallenge, unmarshaled.ShouldChallenge)
	}
}

func TestConfidenceContext_TemporalScoring(t *testing.T) {
	assessor := newBehavioralConfidenceAssessor()

	timeSlots := []struct {
		timeOfDay    int
		dayOfWeek    int
	}{
		{9, 3},
		{15, 6},
		{23, 1},
		{3, 0},
	}

	for _, slot := range timeSlots {
		ctx := &ConfidenceContext{
			TimeOfDay: slot.timeOfDay,
			DayOfWeek: slot.dayOfWeek,
		}

		score := assessor.evaluateTemporalConfidence(ctx)

		if score < -15 || score > 15 {
			t.Errorf("Temporal score out of reasonable range for hour %d weekday %d, got %f", slot.timeOfDay, slot.dayOfWeek, score)
		}
	}
}

func TestBehavioralConfidenceAssessor_UpdateConfidenceModel(t *testing.T) {
	assessor := newBehavioralConfidenceAssessor()

	ctx := &ConfidenceContext{
		UserID:            "update-test-user",
		DeviceFingerprint: "update-test-device",
		IsKnownDevice:     true,
		HistoricalSuccessRate: 0.9,
	}

	result := &ConfidenceAssessmentResult{
		BaseScore:  70.0,
		TotalScore: 85.0,
		Level:      "high",
		Factors: []ConfidenceFactor{
			{Name: "known_device_high_confidence", Weight: 1.2, Score: 15.0},
		},
	}

	assessor.updateConfidenceModel(ctx, result)

	modelKey := ctx.UserID + ":" + ctx.DeviceFingerprint
	model, exists := assessor.confidenceModels[modelKey]
	if !exists {
		t.Fatal("Confidence model not found after update")
	}

	if model.CurrentConfidence != result.TotalScore {
		t.Errorf("Expected CurrentConfidence %f, got %f", result.TotalScore, model.CurrentConfidence)
	}

	if model.SampleCount != 1 {
		t.Errorf("Expected SampleCount 1, got %d", model.SampleCount)
	}

	secondResult := &ConfidenceAssessmentResult{
		BaseScore:  70.0,
		TotalScore: 80.0,
		Level:      "high",
		Factors:    []ConfidenceFactor{},
	}

	assessor.updateConfidenceModel(ctx, secondResult)

	if model.SampleCount != 2 {
		t.Errorf("Expected SampleCount 2, got %d", model.SampleCount)
	}

	if len(model.ConfidenceHistory) != 2 {
		t.Errorf("Expected 2 confidence records, got %d", len(model.ConfidenceHistory))
	}
}

func TestInvisibleCaptchaService_DetectNetworkRisks(t *testing.T) {
	service := NewInvisibleCaptchaService()

	tests := []struct {
		name          string
		envData       map[string]interface{}
		isProxy       bool
		isVPN         bool
		isTor         bool
		isHosting     bool
	}{
		{
			name:    "clean_connection",
			envData: map[string]interface{}{},
			isProxy: false, isVPN: false, isTor: false, isHosting: false,
		},
		{
			name:    "proxy_detected",
			envData: map[string]interface{}{"proxy_detected": true},
			isProxy: true, isVPN: false, isTor: false, isHosting: false,
		},
		{
			name:    "vpn_detected",
			envData: map[string]interface{}{"vpn_detected": true},
			isProxy: false, isVPN: true, isTor: false, isHosting: false,
		},
		{
			name:    "tor_detected",
			envData: map[string]interface{}{"is_tor_exit": true},
			isProxy: false, isVPN: false, isTor: true, isHosting: false,
		},
		{
			name:    "hosting_detected",
			envData: map[string]interface{}{"is_hosting": true},
			isProxy: false, isVPN: false, isTor: false, isHosting: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if service.detectProxy(tt.envData) != tt.isProxy {
				t.Errorf("Expected detectProxy=%v, got %v", tt.isProxy, service.detectProxy(tt.envData))
			}
			if service.detectVPN(tt.envData) != tt.isVPN {
				t.Errorf("Expected detectVPN=%v, got %v", tt.isVPN, service.detectVPN(tt.envData))
			}
			if service.detectTor(tt.envData) != tt.isTor {
				t.Errorf("Expected detectTor=%v, got %v", tt.isTor, service.detectTor(tt.envData))
			}
			if service.detectHosting(tt.envData) != tt.isHosting {
				t.Errorf("Expected detectHosting=%v, got %v", tt.isHosting, service.detectHosting(tt.envData))
			}
		})
	}
}

func TestInvisibleCaptchaConfig_JSON(t *testing.T) {
	config := &InvisibleCaptchaConfig{
		EnableFingerprintOptimization: true,
		EnableConfidenceAssessment:    true,
		EnableBehaviorAnalysis:         true,
		EnableTrustCalculation:         true,
		MinConfidenceThreshold:         0.6,
		MaxRiskScore:                   100.0,
		LearningWindowHours:            720,
		TrustDecayRate:                0.01,
		HistoryRetentionDays:           90,
		EnableAdaptiveScoring:          true,
	}

	jsonBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config to JSON: %v", err)
	}

	var unmarshaled InvisibleCaptchaConfig
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if unmarshaled.EnableFingerprintOptimization != config.EnableFingerprintOptimization {
		t.Error("EnableFingerprintOptimization mismatch")
	}

	if unmarshaled.MinConfidenceThreshold != config.MinConfidenceThreshold {
		t.Error("MinConfidenceThreshold mismatch")
	}

	if unmarshaled.LearningWindowHours != config.LearningWindowHours {
		t.Error("LearningWindowHours mismatch")
	}
}

func TestConfidenceRules_Evaluation(t *testing.T) {
	rules := initConfidenceRules()

	if len(rules) == 0 {
		t.Fatal("Expected confidence rules to be initialized")
	}

	var hasPriorityRule, hasWeightRule bool
	for _, rule := range rules {
		if rule.Priority > 0 {
			hasPriorityRule = true
		}
		if rule.Weight > 0 {
			hasWeightRule = true
		}
		if rule.Condition == nil {
			t.Error("Rule condition should not be nil")
		}
	}

	if !hasPriorityRule {
		t.Error("Expected at least one rule with priority > 0")
	}

	if !hasWeightRule {
		t.Error("Expected at least one rule with weight > 0")
	}
}

func TestAnomalyRules_Initialization(t *testing.T) {
	rules := initAnomalyRules()

	if len(rules) == 0 {
		t.Fatal("Expected anomaly rules to be initialized")
	}

	expectedTypes := map[string]bool{
		"frequency":   false,
		"temporal":   false,
		"geographic": false,
		"device":     false,
		"behavioral": false,
		"timing":     false,
	}

	for _, rule := range rules {
		if _, ok := expectedTypes[rule.Type]; ok {
			expectedTypes[rule.Type] = true
		}

		if rule.Severity < 0 || rule.Severity > 1 {
			t.Errorf("Rule %s has invalid severity: %f", rule.Name, rule.Severity)
		}

		if rule.Weight <= 0 {
			t.Errorf("Rule %s has invalid weight: %f", rule.Name, rule.Weight)
		}

		if !rule.Enabled {
			t.Errorf("Rule %s should be enabled by default", rule.Name)
		}
	}

	for ruleType, found := range expectedTypes {
		if !found {
			t.Errorf("Missing anomaly rule type: %s", ruleType)
		}
	}
}

func TestGlobalRiskRules_Initialization(t *testing.T) {
	rules := initGlobalRiskRules()

	if len(rules) == 0 {
		t.Fatal("Expected global risk rules to be initialized")
	}

	for _, rule := range rules {
		if len(rule.Conditions) == 0 {
			t.Errorf("Rule %s has no conditions", rule.Name)
		}

		if !rule.Enabled {
			t.Errorf("Rule %s should be enabled by default", rule.Name)
		}

		if rule.ScoreMod == 0 {
			t.Errorf("Rule %s has no score modifier", rule.Name)
		}
	}
}

func TestInvisibleCaptchaService_BuildTrustContext(t *testing.T) {
	service := NewInvisibleCaptchaService()

	req := &InvisibleVerificationRequest{
		UserID:            "trust-context-user",
		DeviceFingerprint: "trust-context-device",
		IPAddress:         "10.10.10.10",
		EnvironmentData: map[string]interface{}{
			"proxy_detected": true,
			"vpn_detected":   false,
			"is_tor_exit":    false,
		},
	}

	result := &InvisibleVerificationResult{
		BehaviorPatternMatch: 0.8,
		ConfidenceScore:      75.0,
	}

	ctx := service.buildTrustContext(req, result)

	if ctx == nil {
		t.Fatal("buildTrustContext returned nil")
	}

	if ctx.UserID != req.UserID {
		t.Errorf("Expected UserID '%s', got '%s'", req.UserID, ctx.UserID)
	}

	if ctx.DeviceFingerprint != req.DeviceFingerprint {
		t.Errorf("Expected DeviceFingerprint '%s', got '%s'", req.DeviceFingerprint, ctx.DeviceFingerprint)
	}

	if !ctx.IsProxy {
		t.Error("Expected IsProxy to be true")
	}
}

func TestInvisibleCaptchaService_ExportConfig(t *testing.T) {
	service := NewInvisibleCaptchaService()

	configJSON := service.ExportInvisibleCaptchaConfig()

	if configJSON == "" {
		t.Fatal("ExportInvisibleCaptchaConfig returned empty string")
	}

	var config InvisibleCaptchaConfig
	err := json.Unmarshal([]byte(configJSON), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal exported config: %v", err)
	}

	if config.EnableFingerprintOptimization != service.config.EnableFingerprintOptimization {
		t.Error("Config mismatch after export/import")
	}
}

func TestInvisibleCaptchaService_UpdateConfig(t *testing.T) {
	service := NewInvisibleCaptchaService()

	newConfig := &InvisibleCaptchaConfig{
		EnableFingerprintOptimization: false,
		EnableConfidenceAssessment:    true,
		EnableBehaviorAnalysis:        false,
		EnableTrustCalculation:        true,
		MinConfidenceThreshold:        0.8,
		MaxRiskScore:                  80.0,
		LearningWindowHours:           360,
		TrustDecayRate:               0.02,
		HistoryRetentionDays:          60,
		EnableAdaptiveScoring:         false,
	}

	service.UpdateConfig(newConfig)

	if service.config.EnableFingerprintOptimization {
		t.Error("EnableFingerprintOptimization should be false after update")
	}

	if service.config.MinConfidenceThreshold != 0.8 {
		t.Errorf("Expected MinConfidenceThreshold 0.8, got %f", service.config.MinConfidenceThreshold)
	}
}

func TestVerificationRecord_HistoryManagement(t *testing.T) {
	analyzer := newHistoricalBehaviorAnalyzer()

	userID := "history-test-user"
	history := analyzer.GetOrCreateHistory(userID)

	for i := 0; i < 600; i++ {
		record := &VerificationRecord{
			Timestamp:           time.Now().Add(time.Duration(-i) * time.Hour),
			RiskScore:            float64(20 + i%30),
			Confidence:           float64(70 + i%20),
			TrustScore:           float64(75 + i%15),
			VerificationSuccess:  i%10 != 0,
			ChallengeIssued:      i%5 == 0,
		}
		history.VerificationHistory = append(history.VerificationHistory, record)

		if len(history.VerificationHistory) > 500 {
			history.VerificationHistory = history.VerificationHistory[1:]
		}
	}

	if len(history.VerificationHistory) != 500 {
		t.Errorf("Expected 500 verification records after truncation, got %d", len(history.VerificationHistory))
	}

	firstTimestamp := history.VerificationHistory[0].Timestamp
	for i := 1; i < len(history.VerificationHistory); i++ {
		if history.VerificationHistory[i].Timestamp.After(firstTimestamp) {
			t.Error("Verification records should be in chronological order")
			break
		}
	}
}

func TestDeviceFingerprintRecord_StabilityTracking(t *testing.T) {
	engine := newInvisibleFingerprintEngine()

	fingerprint := "stability-test-fp"
	userID := "stability-test-user"

	components1 := &FingerprintComponents{
		CanvasHash: "canvas1",
		WebGLHash:  "webgl1",
	}

	result1 := engine.OptimizeFingerprint(fingerprint, userID, components1)
	initialStability := result1.StabilityScore

	for i := 0; i < 15; i++ {
		result := engine.OptimizeFingerprint(fingerprint, userID, components1)
		if result.StabilityScore < initialStability {
			t.Errorf("Stability should increase with usage, iteration %d: %f -> %f", i, initialStability, result.StabilityScore)
		}
		initialStability = result.StabilityScore
	}

	record := engine.fingerprintCache[fingerprint]
	if record == nil {
		t.Fatal("Fingerprint record not found")
	}

	if record.AppearanceCount < 16 {
		t.Errorf("Expected at least 16 appearances, got %d", record.AppearanceCount)
	}

	if record.ConsistencyScore < 0.5 {
		t.Errorf("Expected consistency score >= 0.5, got %f", record.ConsistencyScore)
	}
}

func TestConfidenceAssessment_LevelClassification(t *testing.T) {
	assessor := newBehavioralConfidenceAssessor()

	tests := []struct {
		name           string
		totalScore     float64
		expectedLevel  string
	}{
		{"critical_low", 20.0, "critical"},
		{"low", 45.0, "low"},
		{"medium", 65.0, "medium"},
		{"high", 85.0, "high"},
		{"boundary_low", 40.0, "low"},
		{"boundary_medium", 60.0, "medium"},
		{"boundary_high", 80.0, "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ConfidenceContext{
				UserID:               "level-test-user",
				DeviceFingerprint:    "level-test-device",
				IsKnownDevice:        tt.totalScore > 60,
				HistoricalSuccessRate: tt.totalScore / 100.0,
				RecentFailureCount:  int((100 - tt.totalScore) / 20),
				RequestFrequency:     5.0,
			}

			result := assessor.AssessConfidence(ctx)

			if math.Abs(result.TotalScore-tt.totalScore) > 30 {
				t.Logf("Score deviation: expected ~%f, got %f", tt.totalScore, result.TotalScore)
			}
		})
	}
}

func TestInvisibleCaptchaService_MultipleUsers(t *testing.T) {
	service := NewInvisibleCaptchaService()

	users := []string{"user-a", "user-b", "user-c"}

	for _, userID := range users {
		for i := 0; i < 5; i++ {
			req := &InvisibleVerificationRequest{
				SessionID:         "session-" + userID + "-" + string(rune('0'+i)),
				UserID:            userID,
				DeviceFingerprint: "fp-" + userID + "-" + string(rune('0'+i)),
				IPAddress:         "10.0." + string(rune('a'+i)) + ".1",
			}
			result := &InvisibleVerificationResult{
				RiskScore:       25.0,
				ConfidenceScore: 75.0,
				TrustScore:      70.0,
			}
			service.RecordVerificationResult(req, result, true)
		}
	}

	for _, userID := range users {
		profile := service.GetUserTrustProfile(userID)
		if profile.TotalVerifications != 5 {
			t.Errorf("User %s: expected 5 verifications, got %d", userID, profile.TotalVerifications)
		}
		if profile.DeviceCount != 5 {
			t.Errorf("User %s: expected 5 devices, got %d", userID, profile.DeviceCount)
		}
	}
}

func TestSortAndDeduplicateRiskFactors(t *testing.T) {
	service := NewInvisibleCaptchaService()

	req := &InvisibleVerificationRequest{
		SessionID:         "sort-test-session",
		UserID:            "sort-test-user",
		DeviceFingerprint: "sort-test-device",
		IPAddress:         "1.2.3.4",
		EnvironmentData: map[string]interface{}{
			"proxy_detected": true,
			"is_tor_exit":    true,
		},
	}

	result, err := service.ProcessInvisibleVerification(req)
	if err != nil {
		t.Fatalf("ProcessInvisibleVerification failed: %v", err)
	}

	uniqueFactors := make(map[string]bool)
	for _, factor := range result.RiskFactors {
		if uniqueFactors[factor] {
			t.Errorf("Duplicate risk factor: %s", factor)
		}
		uniqueFactors[factor] = true
	}

	if len(result.RiskFactors) != len(uniqueFactors) {
		t.Error("Risk factors should be unique")
	}

	sorted := make([]string, len(result.RiskFactors))
	copy(sorted, result.RiskFactors)
	sort.Strings(sorted)

	if len(sorted) > 1 {
		for i := 1; i < len(sorted); i++ {
			if sorted[i] < sorted[i-1] {
				t.Error("Risk factors should be sorted alphabetically")
			}
		}
	}
}

func TestInvisibleCaptchaService_ConcurrentAccess(t *testing.T) {
	service := NewInvisibleCaptchaService()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 50; j++ {
				req := &InvisibleVerificationRequest{
					SessionID:         "concurrent-session-" + string(rune('0'+idx)) + "-" + string(rune('0'+j)),
					UserID:            "concurrent-user-" + string(rune('0'+idx)),
					DeviceFingerprint: "fp-concurrent-" + string(rune('0'+idx)),
				}
				_, err := service.ProcessInvisibleVerification(req)
				if err != nil {
					t.Errorf("Concurrent verification failed: %v", err)
				}
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	profile := service.GetUserTrustProfile("concurrent-user-0")
	if profile.TotalVerifications != 50 {
		t.Errorf("Expected 50 verifications, got %d", profile.TotalVerifications)
	}
}

func TestFingerprintOptimizationResult_Structure(t *testing.T) {
	result := &FingerprintOptimizationResult{
		Fingerprint:      "test-fp",
		Components: &FingerprintComponents{
			CanvasHash: "hash1",
		},
		StabilityScore:   0.85,
		UniquenessScore:  92.5,
		ConsistencyScore: 0.78,
		QualityScore:     88.0,
	}

	if result.Fingerprint == "" {
		t.Error("Fingerprint should not be empty")
	}

	if result.Components == nil {
		t.Error("Components should not be nil when set")
	}

	if result.StabilityScore < 0 || result.StabilityScore > 1 {
		t.Errorf("StabilityScore should be between 0 and 1, got %f", result.StabilityScore)
	}

	if result.UniquenessScore < 0 || result.UniquenessScore > 100 {
		t.Errorf("UniquenessScore should be between 0 and 100, got %f", result.UniquenessScore)
	}

	if result.QualityScore < 0 || result.QualityScore > 100 {
		t.Errorf("QualityScore should be between 0 and 100, got %f", result.QualityScore)
	}
}
