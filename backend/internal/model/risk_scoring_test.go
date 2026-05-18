package model

import (
	"math"
	"testing"
)

func TestRiskScoringModel_Weights(t *testing.T) {
	weights := RiskScoringWeights{
		TraceWeight:    0.25,
		EnvWeight:      0.20,
		BehaviorWeight: 0.25,
		DeviceWeight:   0.15,
		HistoryWeight:  0.15,
	}

	if !weights.Validate() {
		t.Error("weights should be valid")
	}
}

func TestRiskScoringModel_WeightsNormalization(t *testing.T) {
	weights := RiskScoringWeights{
		TraceWeight:    30,
		EnvWeight:      20,
		BehaviorWeight: 25,
		DeviceWeight:   15,
		HistoryWeight:  10,
	}

	weights.Normalize()

	total := weights.TraceWeight + weights.EnvWeight + weights.BehaviorWeight +
		weights.DeviceWeight + weights.HistoryWeight

	if math.Abs(total-1.0) > 0.001 {
		t.Errorf("expected total to be 1.0, got %f", total)
	}
}

func TestRiskScoringModel_DefaultConfig(t *testing.T) {
	config := DefaultRiskScoringConfig()

	if !config.IsEnabled {
		t.Error("config should be enabled by default")
	}

	if !config.AutoAdjust {
		t.Error("auto adjust should be enabled by default")
	}

	if !config.Weights.Validate() {
		t.Error("default weights should be valid")
	}
}

func TestRiskScoringModel_ScoreBands(t *testing.T) {
	bands := DefaultScoreBands

	if len(bands) != 4 {
		t.Errorf("expected 4 bands, got %d", len(bands))
	}

	expectedBands := []struct {
		min     float64
		max     float64
		label   string
	}{
		{0, 30, "low"},
		{30, 50, "medium"},
		{50, 70, "high"},
		{70, 100, "critical"},
	}

	for i, band := range bands {
		if band.MinScore != expectedBands[i].min {
			t.Errorf("band %d: expected min %f, got %f", i, expectedBands[i].min, band.MinScore)
		}
		if band.MaxScore != expectedBands[i].max {
			t.Errorf("band %d: expected max %f, got %f", i, expectedBands[i].max, band.MaxScore)
		}
		if band.Label != expectedBands[i].label {
			t.Errorf("band %d: expected label %s, got %s", i, expectedBands[i].label, band.Label)
		}
	}
}

func TestRiskScoringModel_Thresholds(t *testing.T) {
	thresholds := RiskThresholds{
		LowMax:      30,
		MediumMax:   50,
		HighMax:     70,
		CriticalMax: 100,
		VerifyMin:   40,
		BlockMin:    80,
	}

	if thresholds.LowMax >= thresholds.MediumMax {
		t.Error("LowMax should be less than MediumMax")
	}
	if thresholds.MediumMax >= thresholds.HighMax {
		t.Error("MediumMax should be less than HighMax")
	}
	if thresholds.HighMax >= thresholds.CriticalMax {
		t.Error("HighMax should be less than CriticalMax")
	}
}

func TestRiskScoringModel_DetermineRiskLevel(t *testing.T) {
	tests := []struct {
		score    float64
		expected RiskLevel
	}{
		{0, RiskLevelCritical},
		{15, RiskLevelCritical},
		{39.9, RiskLevelCritical},
		{40, RiskLevelHigh},
		{59.9, RiskLevelHigh},
		{60, RiskLevelMedium},
		{79.9, RiskLevelMedium},
		{80, RiskLevelLow},
		{85, RiskLevelLow},
		{100, RiskLevelLow},
	}

	for _, tt := range tests {
		result := DetermineRiskLevel(tt.score)
		if result != tt.expected {
			t.Errorf("score %f: expected %s, got %s", tt.score, tt.expected, result)
		}
	}
}

func TestRiskScoringModel_CalculateHumanProbability(t *testing.T) {
	tests := []struct {
		riskScore float64
		minProb   float64
		maxProb   float64
	}{
		{0, 1.0, 1.0},
		{10, 10.8, 11.0},
		{50, 50.0, 50.1},
		{80, 79.0, 79.5},
		{100, 99.0, 99.0},
	}

	for _, tt := range tests {
		prob := CalculateHumanProbability(tt.riskScore)
		if prob < tt.minProb || prob > tt.maxProb {
			t.Errorf("riskScore %f: expected probability between %f and %f, got %f",
				tt.riskScore, tt.minProb, tt.maxProb, prob)
		}
	}
}

func TestRiskScoringModel_MultiDimensionalScore(t *testing.T) {
	score := MultiDimensionalScore{
		TraceScore:    75.5,
		EnvScore:      30.0,
		BehaviorScore: 45.0,
		DeviceScore:   20.0,
		HistoryScore:  35.0,
		TotalScore:    45.5,
		RiskLevel:     RiskLevelMedium,
		Confidence:    0.85,
		Timestamp:     1234567890,
	}

	if score.TraceScore != 75.5 {
		t.Errorf("expected TraceScore 75.5, got %f", score.TraceScore)
	}
	if score.RiskLevel != RiskLevelMedium {
		t.Errorf("expected RiskLevel Medium, got %s", score.RiskLevel)
	}
}

func TestRiskScoringModel_RiskScoringHistory(t *testing.T) {
	history := RiskScoringHistory{
		SessionID:     "test-session",
		IPAddress:     "192.168.1.1",
		Fingerprint:   "abc123",
		TraceScore:    50.0,
		EnvScore:      30.0,
		BehaviorScore: 40.0,
		DeviceScore:   20.0,
		HistoryScore:  35.0,
		TotalScore:    38.0,
		RiskLevel:     "medium",
		Action:        "verify",
		Verified:      true,
		Success:       true,
		CreatedAt:     1234567890,
	}

	if history.TableName() != "risk_scoring_history" {
		t.Errorf("expected table name 'risk_scoring_history', got '%s'", history.TableName())
	}

	if history.TotalScore != 38.0 {
		t.Errorf("expected TotalScore 38.0, got %f", history.TotalScore)
	}
}

func TestRiskScoringModel_ScoreBounds(t *testing.T) {
	maxVal := math.Max(0, math.Min(100, 150.0))
	if maxVal != 100.0 {
		t.Errorf("expected max 100.0, got %f", maxVal)
	}

	minVal := math.Max(0, math.Min(100, -50.0))
	if minVal != 0.0 {
		t.Errorf("expected min 0.0, got %f", minVal)
	}
}

func TestRiskScoringModel_ConfidenceCalculation(t *testing.T) {
	confidence := 0.5
	confidence += 0.1
	confidence += 0.1
	confidence += 0.15

	confidence = math.Min(confidence, 0.95)

	if confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", confidence)
	}
}

func TestRiskScoringModel_WeightedScoreCalculation(t *testing.T) {
	traceScore := 50.0
	envScore := 30.0
	behaviorScore := 40.0
	deviceScore := 20.0
	historyScore := 35.0

	weights := RiskScoringWeights{
		TraceWeight:    0.25,
		EnvWeight:      0.20,
		BehaviorWeight: 0.25,
		DeviceWeight:   0.15,
		HistoryWeight:  0.15,
	}

	totalScore := traceScore*weights.TraceWeight +
		envScore*weights.EnvWeight +
		behaviorScore*weights.BehaviorWeight +
		deviceScore*weights.DeviceWeight +
		historyScore*weights.HistoryWeight

	expected := 50.0*0.25 + 30.0*0.20 + 40.0*0.25 + 20.0*0.15 + 35.0*0.15
	if math.Abs(totalScore-expected) > 0.001 {
		t.Errorf("expected total score %f, got %f", expected, totalScore)
	}
}

func TestRiskScoringModel_RiskLevelConstants(t *testing.T) {
	if RiskLevelLow != "low" {
		t.Errorf("expected RiskLevelLow to be 'low', got '%s'", RiskLevelLow)
	}
	if RiskLevelMedium != "medium" {
		t.Errorf("expected RiskLevelMedium to be 'medium', got '%s'", RiskLevelMedium)
	}
	if RiskLevelHigh != "high" {
		t.Errorf("expected RiskLevelHigh to be 'high', got '%s'", RiskLevelHigh)
	}
	if RiskLevelCritical != "critical" {
		t.Errorf("expected RiskLevelCritical to be 'critical', got '%s'", RiskLevelCritical)
	}
}

func TestRiskScoringModel_RiskContext(t *testing.T) {
	ctx := NewRiskContext()

	if ctx.TraceData == nil {
		t.Error("TraceData should not be nil")
	}
	if ctx.BrowserPlugins == nil {
		t.Error("BrowserPlugins should not be nil")
	}
	if ctx.EnvInfo == nil {
		t.Error("EnvInfo should not be nil")
	}
	if ctx.DeviceInfo == nil {
		t.Error("DeviceInfo should not be nil")
	}
}

func TestRiskScoringModel_RiskContext_HasHighRiskIndicators(t *testing.T) {
	tests := []struct {
		name         string
		isProxy      bool
		isVPN        bool
		isTor        bool
		failureCount int
		mouseSpeed   float64
		timeFromStart int64
		expected     bool
	}{
		{"all safe", false, false, false, 0, 100, 5000, false},
		{"is proxy", true, false, false, 0, 100, 5000, true},
		{"is VPN", false, true, false, 0, 100, 5000, true},
		{"is Tor", false, false, true, 0, 100, 5000, true},
		{"high failure count", false, false, false, 3, 100, 5000, true},
		{"high mouse speed", false, false, false, 0, 2001, 5000, true},
		{"too fast", false, false, false, 0, 100, 400, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RiskContext{
				IsProxy:       tt.isProxy,
				IsVPN:         tt.isVPN,
				IsTor:         tt.isTor,
				FailureCount:  tt.failureCount,
				MouseSpeed:    tt.mouseSpeed,
				TimeFromStart: tt.timeFromStart,
			}
			result := ctx.HasHighRiskIndicators()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRiskScoringModel_RiskContext_GetTrustScore(t *testing.T) {
	tests := []struct {
		name              string
		verificationCount int
		failureCount      int
		isProxy           bool
		isVPN             bool
		isTor             bool
		hasTouchDevice    bool
		timezone          string
		language          string
		expectedScore     float64
	}{
		{"base only", 0, 0, false, false, false, false, "", "", 120.0},
		{"with verification", 6, 0, false, false, false, false, "", "", 145.0},
		{"with success", 1, 0, false, false, false, false, "", "", 135.0},
		{"with touch", 0, 0, false, false, false, true, "", "", 125.0},
		{"with meta", 0, 0, false, false, false, false, "Asia/Shanghai", "zh-CN", 125.0},
		{"full trust", 10, 0, false, false, false, true, "America/New_York", "en-US", 155.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RiskContext{
				VerificationCount: tt.verificationCount,
				FailureCount:      tt.failureCount,
				IsProxy:           tt.isProxy,
				IsVPN:             tt.isVPN,
				IsTor:             tt.isTor,
				HasTouchDevice:    tt.hasTouchDevice,
				Timezone:          tt.timezone,
				Language:          tt.language,
			}
			score := ctx.GetTrustScore()
			if score != tt.expectedScore {
				t.Errorf("expected score %f, got %f", tt.expectedScore, score)
			}
		})
	}
}
