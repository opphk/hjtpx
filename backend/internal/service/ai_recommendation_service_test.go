package service

import (
	"context"
	"testing"
)

func TestAIRecommendationService_NewAIRecommendationService(t *testing.T) {
	svc := NewAIRecommendationService()

	if svc == nil {
		t.Fatal("NewAIRecommendationService returned nil")
	}

	if svc.behaviorAnalysis == nil {
		t.Error("behaviorAnalysis is nil")
	}

	if svc.envDetector == nil {
		t.Error("envDetector is nil")
	}

	if svc.userHistory == nil {
		t.Error("userHistory is nil")
	}

	if svc.cacheExpiration != 30*minute {
		t.Errorf("cacheExpiration = %v, want %v", svc.cacheExpiration, 30*minute)
	}
}

func TestAIRecommendationService_GetRecommendation_Basic(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	req := &CaptchaRecommendationRequest{
		UserID:      "user123",
		Fingerprint: "fp123",
		RiskScore:   30,
		TimeOfDay:   14,
	}

	resp, err := svc.GetRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetRecommendation failed: %v", err)
	}

	if resp == nil {
		t.Fatal("GetRecommendation returned nil response")
	}

	if resp.RecommendedType == "" {
		t.Error("RecommendedType is empty")
	}

	if resp.Confidence < 0 || resp.Confidence > 1 {
		t.Errorf("Confidence = %v, want between 0 and 1", resp.Confidence)
	}

	if resp.Difficulty.Level == "" {
		t.Error("Difficulty.Level is empty")
	}

	if resp.Difficulty.Score < 0 || resp.Difficulty.Score > 100 {
		t.Errorf("Difficulty.Score = %v, want between 0 and 100", resp.Difficulty.Score)
	}

	if resp.EstimatedDuration < 0 {
		t.Errorf("EstimatedDuration = %v, want non-negative", resp.EstimatedDuration)
	}

	if resp.Reason == "" {
		t.Error("Reason is empty")
	}
}

func TestAIRecommendationService_GetRecommendation_HighRisk(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	req := &CaptchaRecommendationRequest{
		UserID:      "user456",
		Fingerprint: "fp456",
		RiskScore:   85,
		TimeOfDay:   3,
		AccessFrequency: 100,
	}

	resp, err := svc.GetRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetRecommendation failed for high risk: %v", err)
	}

	if resp.RecommendedType == "" {
		t.Error("RecommendedType should not be empty")
	}
}

func TestAIRecommendationService_GetRecommendation_LowRisk(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	req := &CaptchaRecommendationRequest{
		UserID:      "user789",
		Fingerprint: "fp789",
		RiskScore:   15,
		TimeOfDay:   14,
	}

	resp, err := svc.GetRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetRecommendation failed for low risk: %v", err)
	}

	if resp.Difficulty.Level != "easy" {
		t.Errorf("Low risk should result in easy difficulty, got %s", resp.Difficulty.Level)
	}
}

func TestAIRecommendationService_GetRecommendation_WithEnvInfo(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	envInfo := &EnvInfo{
		UserAgent:        "Mozilla/5.0",
		Platform:        "Win32",
		Language:        "zh-CN",
		CanvasFingerprint: "canvas123",
		WebGLRenderer:   "WebGL Renderer",
		WebGLVendor:     "WebGL Vendor",
	}

	req := &CaptchaRecommendationRequest{
		UserID:      "user_env",
		Fingerprint: "fp_env",
		RiskScore:   50,
		EnvInfo:    envInfo,
	}

	resp, err := svc.GetRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetRecommendation with envInfo failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}
}

func TestAIRecommendationService_GetRecommendation_WithBehaviorData(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	behaviorData := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 0, Event: "move"},
		{X: 120, Y: 115, Timestamp: 50, Event: "move"},
		{X: 145, Y: 135, Timestamp: 100, Event: "move"},
		{X: 175, Y: 160, Timestamp: 160, Event: "click"},
	}

	req := &CaptchaRecommendationRequest{
		UserID:       "user_behavior",
		Fingerprint:  "fp_behavior",
		RiskScore:    40,
		BehaviorData: behaviorData,
	}

	resp, err := svc.GetRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetRecommendation with behaviorData failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}
}

func TestAIRecommendationService_GetRecommendation_Alternatives(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	req := &CaptchaRecommendationRequest{
		UserID:      "user_alts",
		Fingerprint: "fp_alts",
		RiskScore:   50,
	}

	resp, err := svc.GetRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetRecommendation failed: %v", err)
	}

	if len(resp.Alternatives) == 0 {
		t.Error("Alternatives should not be empty")
	}

	for i, alt := range resp.Alternatives {
		if alt.Type == "" {
			t.Errorf("Alternative[%d] has empty Type", i)
		}

		if alt.Score < 0 || alt.Score > 100 {
			t.Errorf("Alternative[%d] Score = %v, want between 0 and 100", i, alt.Score)
		}

		if alt.Reason == "" {
			t.Errorf("Alternative[%d] has empty Reason", i)
		}
	}
}

func TestAIRecommendationService_GetRecommendation_Factors(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	req := &CaptchaRecommendationRequest{
		UserID:      "user_factors",
		Fingerprint: "fp_factors",
		RiskScore:   45,
	}

	resp, err := svc.GetRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetRecommendation failed: %v", err)
	}

	if len(resp.Factors) == 0 {
		t.Error("Factors should not be empty")
	}

	for i, factor := range resp.Factors {
		if factor.Name == "" {
			t.Errorf("Factor[%d] has empty Name", i)
		}

		if factor.Weight < 0 || factor.Weight > 1 {
			t.Errorf("Factor[%d] Weight = %v, want between 0 and 1", i, factor.Weight)
		}

		if factor.Score < 0 || factor.Score > 100 {
			t.Errorf("Factor[%d] Score = %v, want between 0 and 100", i, factor.Score)
		}
	}
}

func TestAIRecommendationService_GetDifficultyRecommendation_Basic(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	req := &DifficultyRequest{
		UserID:      "user_diff",
		Fingerprint: "fp_diff",
		CaptchaType: CaptchaTypeSlider,
		RiskScore:   50,
		TimeOfDay:   14,
	}

	resp, err := svc.GetDifficultyRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetDifficultyRecommendation failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	if resp.RecommendedLevel == "" {
		t.Error("RecommendedLevel is empty")
	}

	if resp.Confidence < 0 || resp.Confidence > 1 {
		t.Errorf("Confidence = %v, want between 0 and 1", resp.Confidence)
	}

	if resp.AdjustmentReason == "" {
		t.Error("AdjustmentReason is empty")
	}
}

func TestAIRecommendationService_GetDifficultyRecommendation_AllTypes(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	captchaTypes := []CaptchaType{
		CaptchaTypeSlider,
		CaptchaTypeClick,
		CaptchaTypeGesture,
		CaptchaTypeLianLianKan,
		CaptchaTypeVoice,
		CaptchaType3D,
	}

	for _, captchaType := range captchaTypes {
		req := &DifficultyRequest{
			CaptchaType: captchaType,
			RiskScore:   50,
		}

		resp, err := svc.GetDifficultyRecommendation(ctx, req)
		if err != nil {
			t.Errorf("GetDifficultyRecommendation for %s failed: %v", captchaType, err)
			continue
		}

		if resp.Difficulty.Level == "" {
			t.Errorf("Difficulty.Level is empty for %s", captchaType)
		}
	}
}

func TestAIRecommendationService_GetDifficultyRecommendation_HighSuccessRate(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	req := &DifficultyRequest{
		CaptchaType: CaptchaTypeSlider,
		RiskScore:   50,
		SuccessRate: 0.95,
	}

	resp, err := svc.GetDifficultyRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetDifficultyRecommendation failed: %v", err)
	}

	if resp.Difficulty.Level != "easy" {
		t.Errorf("High success rate should result in easy difficulty, got %s", resp.Difficulty.Level)
	}
}

func TestAIRecommendationService_GetDifficultyRecommendation_LowSuccessRate(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	req := &DifficultyRequest{
		CaptchaType: CaptchaTypeSlider,
		RiskScore:   50,
		SuccessRate: 0.4,
	}

	resp, err := svc.GetDifficultyRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetDifficultyRecommendation failed: %v", err)
	}

	if resp.Difficulty.Level != "hard" && resp.Difficulty.Level != "extreme" {
		t.Errorf("Low success rate should result in hard or extreme difficulty, got %s", resp.Difficulty.Level)
	}
}

func TestAIRecommendationService_GetDifficultyRecommendation_ManyFailures(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	req := &DifficultyRequest{
		CaptchaType:  CaptchaTypeSlider,
		RiskScore:    30,
		FailureCount: 5,
	}

	resp, err := svc.GetDifficultyRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetDifficultyRecommendation failed: %v", err)
	}

	if resp.AdjustmentReason == "" {
		t.Error("AdjustmentReason should not be empty when failures are present")
	}
}

func TestAIRecommendationService_GetDifficultyRecommendation_Factors(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	req := &DifficultyRequest{
		CaptchaType:  CaptchaTypeSlider,
		RiskScore:    60,
		SuccessRate:  0.7,
		FailureCount: 2,
	}

	resp, err := svc.GetDifficultyRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetDifficultyRecommendation failed: %v", err)
	}

	if len(resp.Factors) == 0 {
		t.Error("Factors should not be empty")
	}

	for i, factor := range resp.Factors {
		if factor.Name == "" {
			t.Errorf("Factor[%d] has empty Name", i)
		}
	}
}

func TestAIRecommendationService_UpdateUserHistory_Success(t *testing.T) {
	svc := NewAIRecommendationService()

	svc.UpdateUserHistory("user1", "fp1", "192.168.1.1", CaptchaTypeSlider, true, 3000)

	history := svc.GetUserStats("user1", "")
	if history == nil {
		t.Fatal("History should not be nil after update")
	}

	if history.TotalAttempts != 1 {
		t.Errorf("TotalAttempts = %d, want 1", history.TotalAttempts)
	}

	if history.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", history.SuccessCount)
	}

	if history.FailureCount != 0 {
		t.Errorf("FailureCount = %d, want 0", history.FailureCount)
	}

	if history.SuccessRate != 1.0 {
		t.Errorf("SuccessRate = %v, want 1.0", history.SuccessRate)
	}
}

func TestAIRecommendationService_UpdateUserHistory_Failure(t *testing.T) {
	svc := NewAIRecommendationService()

	svc.UpdateUserHistory("user2", "fp2", "192.168.1.2", CaptchaTypeClick, false, 8000)

	history := svc.GetUserStats("user2", "")
	if history == nil {
		t.Fatal("History should not be nil after update")
	}

	if history.TotalAttempts != 1 {
		t.Errorf("TotalAttempts = %d, want 1", history.TotalAttempts)
	}

	if history.SuccessCount != 0 {
		t.Errorf("SuccessCount = %d, want 0", history.SuccessCount)
	}

	if history.FailureCount != 1 {
		t.Errorf("FailureCount = %d, want 1", history.FailureCount)
	}
}

func TestAIRecommendationService_UpdateUserHistory_Multiple(t *testing.T) {
	svc := NewAIRecommendationService()

	svc.UpdateUserHistory("user3", "fp3", "192.168.1.3", CaptchaTypeSlider, true, 3000)
	svc.UpdateUserHistory("user3", "fp3", "192.168.1.3", CaptchaTypeClick, true, 4000)
	svc.UpdateUserHistory("user3", "fp3", "192.168.1.3", CaptchaTypeSlider, false, 5000)
	svc.UpdateUserHistory("user3", "fp3", "192.168.1.3", CaptchaType3D, true, 3500)

	history := svc.GetUserStats("user3", "")
	if history == nil {
		t.Fatal("History should not be nil")
	}

	if history.TotalAttempts != 4 {
		t.Errorf("TotalAttempts = %d, want 4", history.TotalAttempts)
	}

	if history.SuccessCount != 3 {
		t.Errorf("SuccessCount = %d, want 3", history.SuccessCount)
	}

	if history.FailureCount != 1 {
		t.Errorf("FailureCount = %d, want 1", history.FailureCount)
	}

	expectedRate := 3.0 / 4.0
	if history.SuccessRate != expectedRate {
		t.Errorf("SuccessRate = %v, want %v", history.SuccessRate, expectedRate)
	}
}

func TestAIRecommendationService_UpdateUserHistory_PreferredTypes(t *testing.T) {
	svc := NewAIRecommendationService()

	for i := 0; i < 5; i++ {
		svc.UpdateUserHistory("user4", "fp4", "192.168.1.4", CaptchaTypeSlider, true, 3000)
	}
	for i := 0; i < 2; i++ {
		svc.UpdateUserHistory("user4", "fp4", "192.168.1.4", CaptchaTypeClick, true, 4000)
	}

	history := svc.GetUserStats("user4", "")
	if history == nil {
		t.Fatal("History should not be nil")
	}

	if count, exists := history.PreferredTypes[CaptchaTypeSlider]; !exists || count != 5 {
		t.Errorf("PreferredTypes[slider] = %d, want 5", count)
	}

	if count, exists := history.PreferredTypes[CaptchaTypeClick]; !exists || count != 2 {
		t.Errorf("PreferredTypes[click] = %d, want 2", count)
	}
}

func TestAIRecommendationService_UpdateUserHistory_AvgDuration(t *testing.T) {
	svc := NewAIRecommendationService()

	svc.UpdateUserHistory("user5", "fp5", "192.168.1.5", CaptchaTypeSlider, true, 3000)
	svc.UpdateUserHistory("user5", "fp5", "192.168.1.5", CaptchaTypeSlider, true, 5000)
	svc.UpdateUserHistory("user5", "fp5", "192.168.1.5", CaptchaTypeSlider, true, 7000)

	history := svc.GetUserStats("user5", "")
	if history == nil {
		t.Fatal("History should not be nil")
	}

	expectedAvg := int64(5000)
	if history.AvgDuration != expectedAvg {
		t.Errorf("AvgDuration = %d, want %d", history.AvgDuration, expectedAvg)
	}
}

func TestAIRecommendationService_GetUserStats_NotFound(t *testing.T) {
	svc := NewAIRecommendationService()

	history := svc.GetUserStats("nonexistent", "")
	if history != nil {
		t.Error("History should be nil for nonexistent user")
	}
}

func TestAIRecommendationService_GetUserStats_ByFingerprint(t *testing.T) {
	svc := NewAIRecommendationService()

	svc.UpdateUserHistory("", "fp6", "192.168.1.6", CaptchaTypeSlider, true, 3000)

	history := svc.GetUserStats("", "fp6")
	if history == nil {
		t.Fatal("History should not be nil when queried by fingerprint")
	}

	if history.Fingerprint != "fp6" {
		t.Errorf("Fingerprint = %s, want fp6", history.Fingerprint)
	}
}

func TestAIRecommendationService_GetCaptchaTypeStats(t *testing.T) {
	svc := NewAIRecommendationService()

	stats := svc.GetCaptchaTypeStats()
	if len(stats) == 0 {
		t.Fatal("CaptchaTypeStats should not be empty")
	}

	for _, stat := range stats {
		if stat.Type == "" {
			t.Error("Stat.Type is empty")
		}

		if stat.SuccessRate < 0 || stat.SuccessRate > 1 {
			t.Errorf("Stat.SuccessRate = %v, want between 0 and 1", stat.SuccessRate)
		}

		if stat.FailureRate < 0 || stat.FailureRate > 1 {
			t.Errorf("Stat.FailureRate = %v, want between 0 and 1", stat.FailureRate)
		}

		if stat.ComfortScore < 0 || stat.ComfortScore > 1 {
			t.Errorf("Stat.ComfortScore = %v, want between 0 and 1", stat.ComfortScore)
		}

		if stat.AvgDuration < 0 {
			t.Errorf("Stat.AvgDuration = %v, want non-negative", stat.AvgDuration)
		}
	}
}

func TestAIRecommendationService_GetRecommendation_WithHistory(t *testing.T) {
	svc := NewAIRecommendationService()

	for i := 0; i < 10; i++ {
		svc.UpdateUserHistory("user7", "fp7", "192.168.1.7", CaptchaTypeSlider, true, 3000)
	}
	for i := 0; i < 3; i++ {
		svc.UpdateUserHistory("user7", "fp7", "192.168.1.7", CaptchaTypeClick, true, 4000)
	}

	ctx := context.Background()
	req := &CaptchaRecommendationRequest{
		UserID:      "user7",
		Fingerprint: "fp7",
		RiskScore:   30,
	}

	resp, err := svc.GetRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetRecommendation failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	if resp.Confidence < 0.7 {
		t.Errorf("Confidence should be high when user has history, got %v", resp.Confidence)
	}
}

func TestAIRecommendationService_GetRecommendation_CombinedRisk(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	envInfo := &EnvInfo{
		UserAgent:        "headless",
		CanvasFingerprint: "",
	}

	req := &CaptchaRecommendationRequest{
		UserID:      "user_risk",
		Fingerprint: "fp_risk",
		RiskScore:   85,
		EnvInfo:    envInfo,
		AccessFrequency: 100,
	}

	resp, err := svc.GetRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetRecommendation failed: %v", err)
	}

	if resp.RecommendedType == "" {
		t.Error("RecommendedType should not be empty")
	}
}

func TestAIRecommendationService_GetDifficultyRecommendation_NightTime(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	req := &DifficultyRequest{
		CaptchaType: CaptchaTypeSlider,
		RiskScore:   50,
		TimeOfDay:   23,
	}

	resp, err := svc.GetDifficultyRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetDifficultyRecommendation failed: %v", err)
	}

	if resp.Difficulty.Level == "extreme" {
		t.Error("Night time should not result in extreme difficulty")
	}
}

func TestAIRecommendationService_GetRecommendation_AccessFrequency(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	req := &CaptchaRecommendationRequest{
		UserID:           "user_freq",
		Fingerprint:      "fp_freq",
		RiskScore:        30,
		AccessFrequency: 50,
		DeviceTrust:     0.8,
	}

	resp, err := svc.GetRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetRecommendation failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}
}

func TestAIRecommendationService_GetRecommendation_DeviceTrust(t *testing.T) {
	svc := NewAIRecommendationService()
	ctx := context.Background()

	req := &CaptchaRecommendationRequest{
		UserID:      "user_trust",
		Fingerprint: "fp_trust",
		RiskScore:   20,
		DeviceTrust: 0.9,
	}

	resp, err := svc.GetRecommendation(ctx, req)
	if err != nil {
		t.Fatalf("GetRecommendation failed: %v", err)
	}

	if resp.Difficulty.Level != "easy" {
		t.Errorf("High device trust should result in easy difficulty, got %s", resp.Difficulty.Level)
	}
}

func TestCaptchaType_Constants(t *testing.T) {
	captchaTypes := []CaptchaType{
		CaptchaTypeSlider,
		CaptchaTypeClick,
		CaptchaTypeGesture,
		CaptchaTypeLianLianKan,
		CaptchaTypeVoice,
		CaptchaType3D,
		CaptchaTypeSeamless,
	}

	expectedValues := []string{
		"slider",
		"click",
		"gesture",
		"lianliankan",
		"voice",
		"3d",
		"seamless",
	}

	for i, ct := range captchaTypes {
		if string(ct) != expectedValues[i] {
			t.Errorf("CaptchaType constant = %s, want %s", ct, expectedValues[i])
		}
	}
}

func TestAIRecommendationService_calculateCombinedRisk(t *testing.T) {
	svc := NewAIRecommendationService()

	tests := []struct {
		name          string
		clientRisk    float64
		envRisk       float64
		behaviorRisk  float64
		minExpected   float64
		maxExpected   float64
	}{
		{"AllLow", 10, 10, 10, 0, 20},
		{"AllMedium", 50, 50, 50, 40, 60},
		{"AllHigh", 90, 90, 90, 70, 100},
		{"Mixed", 30, 60, 90, 40, 80},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.calculateCombinedRisk(tt.clientRisk, tt.envRisk, tt.behaviorRisk)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("calculateCombinedRisk(%v, %v, %v) = %v, want between %v and %v",
					tt.clientRisk, tt.envRisk, tt.behaviorRisk, result, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestAIRecommendationService_selectOptimalCaptchaType(t *testing.T) {
	svc := NewAIRecommendationService()

	history := &AIRecommendationUserHistory{
		TotalAttempts: 10,
		SuccessRate:   0.9,
		PreferredTypes: map[CaptchaType]int{
			CaptchaTypeSlider: 8,
		},
	}

	req := &CaptchaRecommendationRequest{
		RiskScore: 30,
	}

	captchaType := svc.selectOptimalCaptchaType(history, 30, req)
	if captchaType == "" {
		t.Error("selectOptimalCaptchaType returned empty type")
	}
}

func TestAIRecommendationService_calculateDifficulty(t *testing.T) {
	svc := NewAIRecommendationService()

	history := &AIRecommendationUserHistory{
		TotalAttempts: 10,
		SuccessRate:   0.85,
		AvgDuration:   5000,
	}

	req := &CaptchaRecommendationRequest{}

	difficulty := svc.calculateDifficulty(CaptchaTypeSlider, 50, history, req)

	if difficulty.Level == "" {
		t.Error("Difficulty.Level is empty")
	}

	if difficulty.Score < 0 || difficulty.Score > 100 {
		t.Errorf("Difficulty.Score = %v, want between 0 and 100", difficulty.Score)
	}
}

func TestAIRecommendationService_estimateDuration(t *testing.T) {
	svc := NewAIRecommendationService()

	captchaTypes := []CaptchaType{
		CaptchaTypeSlider,
		CaptchaTypeClick,
		CaptchaTypeGesture,
		CaptchaTypeLianLianKan,
		CaptchaTypeVoice,
		CaptchaType3D,
		CaptchaTypeSeamless,
	}

	for _, ct := range captchaTypes {
		difficulty := CaptchaDifficulty{Level: "medium"}
		duration := svc.estimateDuration(ct, difficulty)
		if duration < 0 {
			t.Errorf("estimateDuration for %s returned negative: %d", ct, duration)
		}
	}
}

func TestAIRecommendationService_generateRecommendationReason(t *testing.T) {
	svc := NewAIRecommendationService()

	history := &AIRecommendationUserHistory{
		TotalAttempts:  10,
		SuccessRate:    0.9,
		LastCaptchaType: CaptchaTypeSlider,
	}

	reason := svc.generateRecommendationReason(CaptchaTypeSlider, 30, history)
	if reason == "" {
		t.Error("generateRecommendationReason returned empty string")
	}

	reason2 := svc.generateRecommendationReason(CaptchaTypeSlider, 80, nil)
	if reason2 == "" {
		t.Error("generateRecommendationReason returned empty string for high risk")
	}
}

func TestAIRecommendationService_calculateConfidence(t *testing.T) {
	svc := NewAIRecommendationService()

	history := &AIRecommendationUserHistory{
		TotalAttempts: 15,
		PreferredTypes: map[CaptchaType]int{
			CaptchaTypeSlider: 10,
		},
	}

	confidence := svc.calculateConfidence(CaptchaTypeSlider, history, 30)
	if confidence < 0.5 || confidence > 0.95 {
		t.Errorf("calculateConfidence = %v, want between 0.5 and 0.95", confidence)
	}

	confidenceNoHistory := svc.calculateConfidence(CaptchaTypeSlider, nil, 30)
	if confidenceNoHistory > 0.7 {
		t.Errorf("calculateConfidence without history = %v, should be lower", confidenceNoHistory)
	}
}

func TestAIRecommendationService_convertToTrajectory(t *testing.T) {
	svc := NewAIRecommendationService()

	behaviorData := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 0, Event: "move"},
		{X: 120, Y: 115, Timestamp: 50, Event: "move"},
		{X: 145, Y: 135, Timestamp: 100, Event: "click"},
	}

	trajectory := svc.convertToTrajectory(behaviorData)

	if len(trajectory) != len(behaviorData) {
		t.Errorf("trajectory length = %d, want %d", len(trajectory), len(behaviorData))
	}

	for i, point := range trajectory {
		if point.X != behaviorData[i].X || point.Y != behaviorData[i].Y {
			t.Errorf("trajectory[%d] = (%d, %d), want (%d, %d)",
				i, point.X, point.Y, behaviorData[i].X, behaviorData[i].Y)
		}
	}
}

func TestAIRecommendationService_getTopPreferredTypes(t *testing.T) {
	svc := NewAIRecommendationService()

	history := &AIRecommendationUserHistory{
		PreferredTypes: map[CaptchaType]int{
			CaptchaTypeSlider:     5,
			CaptchaTypeClick:     10,
			CaptchaType3D:        3,
			CaptchaTypeVoice:     8,
		},
	}

	topTypes := svc.getTopPreferredTypes(history)

	if len(topTypes) != 4 {
		t.Errorf("getTopPreferredTypes returned %d types, want 4", len(topTypes))
	}

	if topTypes[0] != CaptchaTypeClick {
		t.Errorf("First type should be Click (highest count), got %s", topTypes[0])
	}
}

func TestAIRecommendationService_getTopPreferredTypes_Empty(t *testing.T) {
	svc := NewAIRecommendationService()

	topTypes := svc.getTopPreferredTypes(nil)
	if len(topTypes) != 0 {
		t.Errorf("getTopPreferredTypes with nil should return empty, got %d", len(topTypes))
	}

	topTypesEmpty := svc.getTopPreferredTypes(&AIRecommendationUserHistory{})
	if len(topTypesEmpty) != 0 {
		t.Errorf("getTopPreferredTypes with empty history should return empty, got %d", len(topTypesEmpty))
	}
}

func TestAIRecommendationService_generateAlternatives(t *testing.T) {
	svc := NewAIRecommendationService()

	history := &AIRecommendationUserHistory{
		TotalAttempts: 10,
		PreferredTypes: map[CaptchaType]int{
			CaptchaTypeSlider: 5,
		},
	}

	req := &CaptchaRecommendationRequest{}

	alts := svc.generateAlternatives(CaptchaTypeSlider, history, 50, req)

	if len(alts) > 3 {
		t.Errorf("generateAlternatives returned %d items, should be max 3", len(alts))
	}

	for _, alt := range alts {
		if alt.Type == CaptchaTypeSlider {
			t.Error("Alternative should not include primary type")
		}
	}
}

func TestAIRecommendationService_generateAlternativeReason(t *testing.T) {
	svc := NewAIRecommendationService()

	history := &AIRecommendationUserHistory{
		PreferredTypes: map[CaptchaType]int{
			CaptchaTypeClick: 5,
		},
	}

	reason := svc.generateAlternativeReason(CaptchaTypeClick, 80, history, 30)
	if reason == "" {
		t.Error("generateAlternativeReason returned empty string")
	}

	reasonHighRisk := svc.generateAlternativeReason(CaptchaType3D, 80, nil, 80)
	if reasonHighRisk == "" {
		t.Error("generateAlternativeReason returned empty string for high risk")
	}
}

func TestAIRecommendationService_getBaseDifficulty(t *testing.T) {
	svc := NewAIRecommendationService()

	types := []CaptchaType{
		CaptchaTypeSlider,
		CaptchaTypeClick,
		CaptchaType3D,
	}

	for _, ct := range types {
		base := svc.getBaseDifficulty(ct)
		if base.Level != "medium" {
			t.Errorf("Base level for %s should be 'medium', got %s", ct, base.Level)
		}
		if base.Score != 50 {
			t.Errorf("Base score for %s should be 50, got %v", ct, base.Score)
		}
	}
}

func TestAIRecommendationService_analyzeDifficultyFactors(t *testing.T) {
	svc := NewAIRecommendationService()

	difficulty := CaptchaDifficulty{Score: 60}
	req := &DifficultyRequest{
		SuccessRate:  0.8,
		RiskScore:    65,
		FailureCount: 2,
	}

	factors := svc.analyzeDifficultyFactors(difficulty, req, nil)

	if len(factors) == 0 {
		t.Error("analyzeDifficultyFactors returned empty")
	}
}

func TestAIRecommendationService_calculateDifficultyConfidence(t *testing.T) {
	svc := NewAIRecommendationService()

	history := &AIRecommendationUserHistory{
		TotalAttempts: 15,
	}

	req := &DifficultyRequest{
		SuccessRate: 0.85,
		RiskScore:   50,
	}

	confidence := svc.calculateDifficultyConfidence(req, history)
	if confidence < 0.7 || confidence > 0.95 {
		t.Errorf("calculateDifficultyConfidence = %v, want between 0.7 and 0.95", confidence)
	}

	confidenceNoHistory := svc.calculateDifficultyConfidence(req, nil)
	if confidenceNoHistory > 0.8 {
		t.Errorf("calculateDifficultyConfidence without history = %v, should be lower", confidenceNoHistory)
	}
}

func TestAIRecommendationService_getUserKey(t *testing.T) {
	svc := NewAIRecommendationService()

	tests := []struct {
		name        string
		userID      string
		fingerprint string
		expected    string
	}{
		{"WithUserID", "user123", "", "user:user123"},
		{"WithFingerprint", "", "fp123", "fp:fp123"},
		{"WithBoth", "user123", "fp123", "user:user123"},
		{"WithNeither", "", "", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.getUserKey(tt.userID, tt.fingerprint)
			if result != tt.expected {
				t.Errorf("getUserKey(%s, %s) = %s, want %s", tt.userID, tt.fingerprint, result, tt.expected)
			}
		})
	}
}

func TestAIRecommendationService_calculateCaptchaTypeScore(t *testing.T) {
	svc := NewAIRecommendationService()

	stats := &CaptchaTypeRecommendationStats{
		Type:         CaptchaTypeSlider,
		SuccessRate:  0.85,
		AvgDuration:  5000,
		ComfortScore: 0.8,
	}

	history := &AIRecommendationUserHistory{
		TotalAttempts: 10,
		PreferredTypes: map[CaptchaType]int{
			CaptchaTypeSlider: 8,
		},
	}

	req := &CaptchaRecommendationRequest{
		DeviceTrust: 0.8,
	}

	score := svc.calculateCaptchaTypeScore(CaptchaTypeSlider, stats, history, 30, req)
	if score < 0 || score > 100 {
		t.Errorf("calculateCaptchaTypeScore = %v, want between 0 and 100", score)
	}

	scoreNoHistory := svc.calculateCaptchaTypeScore(CaptchaTypeSlider, stats, nil, 30, nil)
	if scoreNoHistory < 0 || scoreNoHistory > 100 {
		t.Errorf("calculateCaptchaTypeScore without history = %v, want between 0 and 100", scoreNoHistory)
	}
}

const minute = 60 * second
const second = 1000000000
