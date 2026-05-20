package service

import (
	"context"
	"testing"
	"time"
)

func TestAdaptiveEvolutionEngine_Initialize(t *testing.T) {
	engine := NewAdaptiveEvolutionEngine()

	if engine.ecosystemMetrics.TotalAttempts != 0 {
		t.Errorf("Expected TotalAttempts 0, got %d", engine.ecosystemMetrics.TotalAttempts)
	}

	if engine.ecosystemMetrics.SuccessRate != 0.5 {
		t.Errorf("Expected SuccessRate 0.5, got %.2f", engine.ecosystemMetrics.SuccessRate)
	}

	if len(engine.difficultyLevels) != 5 {
		t.Errorf("Expected 5 difficulty levels, got %d", len(engine.difficultyLevels))
	}
}

func TestAdaptiveEvolutionEngine_AnalyzeUserAndGenerateChallenge(t *testing.T) {
	engine := NewAdaptiveEvolutionEngine()

	challenge, err := engine.AnalyzeUserAndGenerateChallenge(context.Background(), "user123", 1)
	if err != nil {
		t.Fatalf("AnalyzeUserAndGenerateChallenge failed: %v", err)
	}

	if challenge == nil {
		t.Fatal("Challenge should not be nil")
	}

	if challenge.Difficulty < 1 || challenge.Difficulty > 5 {
		t.Errorf("Difficulty should be between 1 and 5, got %d", challenge.Difficulty)
	}

	if challenge.Config == nil {
		t.Error("Difficulty config should not be nil")
	}

	if challenge.Type == "" {
		t.Error("Challenge type should not be empty")
	}

	if engine.ecosystemMetrics.TotalAttempts != 1 {
		t.Errorf("Expected TotalAttempts 1, got %d", engine.ecosystemMetrics.TotalAttempts)
	}
}

func TestAdaptiveEvolutionEngine_CalculateRecommendedDifficulty(t *testing.T) {
	engine := NewAdaptiveEvolutionEngine()

	testCases := []struct {
		name             string
		successRate      float64
		timeHistory      []float64
		difficultyHistory []int
		minExpected      int
		maxExpected      int
	}{
		{
			"high success rate",
			0.9,
			[]float64{5, 5, 5, 5, 5},
			[]int{4, 4, 4},
			4,
			5,
		},
		{
			"low success rate",
			0.3,
			[]float64{5, 5, 5, 5, 5},
			[]int{2, 2, 2},
			1,
			2,
		},
		{
			"no history",
			0.5,
			[]float64{},
			[]int{},
			1,
			2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			profile := &UserBehaviorProfile{
				SuccessHistory:     boolSlice(tc.successRate, 5),
				TimeHistory:        tc.timeHistory,
				DifficultyHistory:  tc.difficultyHistory,
			}

			difficulty := engine.calculateRecommendedDifficulty(profile, 1)

			if difficulty < tc.minExpected || difficulty > tc.maxExpected {
				t.Errorf("Expected difficulty between %d and %d, got %d", tc.minExpected, tc.maxExpected, difficulty)
			}
		})
	}
}

func TestAdaptiveEvolutionEngine_RecordAttempt(t *testing.T) {
	engine := NewAdaptiveEvolutionEngine()

	err := engine.RecordAttempt(context.Background(), "user123", true, 5*time.Second, 3)
	if err != nil {
		t.Fatalf("RecordAttempt failed: %v", err)
	}

	profile, _ := engine.GetUserProfile(context.Background(), "user123")
	if profile == nil {
		t.Fatal("Profile should exist after recording attempt")
	}

	if len(profile.SuccessHistory) != 1 {
		t.Errorf("Expected 1 success history entry, got %d", len(profile.SuccessHistory))
	}

	if profile.SessionCount != 1 {
		t.Errorf("Expected SessionCount 1, got %d", profile.SessionCount)
	}

	if engine.ecosystemMetrics.TotalAttempts != 1 {
		t.Errorf("Expected TotalAttempts 1, got %d", engine.ecosystemMetrics.TotalAttempts)
	}
}

func TestAdaptiveEvolutionEngine_RecordAttempt_Success(t *testing.T) {
	engine := NewAdaptiveEvolutionEngine()

	for i := 0; i < 5; i++ {
		_ = engine.RecordAttempt(context.Background(), "user123", true, 5*time.Second, 3)
	}

	profile, _ := engine.GetUserProfile(context.Background(), "user123")
	if profile.SuccessRate != 1.0 {
		t.Errorf("Expected SuccessRate 1.0, got %.2f", profile.SuccessRate)
	}

	if profile.AdaptationLevel < 5 {
		t.Errorf("Expected AdaptationLevel >= 5, got %d", profile.AdaptationLevel)
	}
}

func TestAdaptiveEvolutionEngine_RecordAttempt_Failure(t *testing.T) {
	engine := NewAdaptiveEvolutionEngine()

	for i := 0; i < 5; i++ {
		_ = engine.RecordAttempt(context.Background(), "user123", false, 5*time.Second, 3)
	}

	profile, _ := engine.GetUserProfile(context.Background(), "user123")
	if profile.SuccessRate != 0.0 {
		t.Errorf("Expected SuccessRate 0.0, got %.2f", profile.SuccessRate)
	}

	if profile.FailedAttempts != 5 {
		t.Errorf("Expected FailedAttempts 5, got %d", profile.FailedAttempts)
	}
}

func TestAdaptiveEvolutionEngine_RecordAttack(t *testing.T) {
	engine := NewAdaptiveEvolutionEngine()

	pattern := &AttackPattern{
		Type:         "brute_force",
		AttackVector: "rapid_submission",
		Success:      false,
	}

	err := engine.RecordAttack(context.Background(), pattern)
	if err != nil {
		t.Fatalf("RecordAttack failed: %v", err)
	}

	if engine.ecosystemMetrics.AttackCount != 1 {
		t.Errorf("Expected AttackCount 1, got %d", engine.ecosystemMetrics.AttackCount)
	}

	if engine.ecosystemMetrics.DefenseCount != 1 {
		t.Errorf("Expected DefenseCount 1, got %d", engine.ecosystemMetrics.DefenseCount)
	}
}

func TestAdaptiveEvolutionEngine_OptimizeDifficulty(t *testing.T) {
	engine := NewAdaptiveEvolutionEngine()

	engine.ecosystemMetrics.TotalAttempts = 100
	engine.ecosystemMetrics.HumanSuccessRate = 0.9
	engine.ecosystemMetrics.AvgDifficulty = 2.0

	err := engine.OptimizeDifficulty()
	if err != nil {
		t.Fatalf("OptimizeDifficulty failed: %v", err)
	}

	if engine.ecosystemMetrics.EvolutionCycles != 1 {
		t.Errorf("Expected EvolutionCycles 1, got %d", engine.ecosystemMetrics.EvolutionCycles)
	}

	if engine.ecosystemMetrics.StabilityScore < 0 || engine.ecosystemMetrics.StabilityScore > 1 {
		t.Errorf("StabilityScore should be between 0 and 1, got %.2f", engine.ecosystemMetrics.StabilityScore)
	}
}

func TestAdaptiveEvolutionEngine_GenerateEcosystemReport(t *testing.T) {
	engine := NewAdaptiveEvolutionEngine()

	report := engine.GenerateEcosystemReport()

	if len(report) == 0 {
		t.Error("Report should not be empty")
	}

	if report == "" {
		t.Error("Report should be generated")
	}
}

func TestAdaptiveEvolutionEngine_GetUserProfile(t *testing.T) {
	engine := NewAdaptiveEvolutionEngine()

	_, _ = engine.AnalyzeUserAndGenerateChallenge(context.Background(), "user123", 1)

	profile, err := engine.GetUserProfile(context.Background(), "user123")
	if err != nil {
		t.Fatalf("GetUserProfile failed: %v", err)
	}

	if profile.UserID != "user123" {
		t.Errorf("Expected UserID 'user123', got '%s'", profile.UserID)
	}
}

func TestAdaptiveEvolutionEngine_GetEcosystemMetrics(t *testing.T) {
	engine := NewAdaptiveEvolutionEngine()

	metrics, err := engine.GetEcosystemMetrics(context.Background())
	if err != nil {
		t.Fatalf("GetEcosystemMetrics failed: %v", err)
	}

	if metrics.TotalAttempts != 0 {
		t.Errorf("Expected TotalAttempts 0, got %d", metrics.TotalAttempts)
	}

	if metrics.EvolutionCycles != 0 {
		t.Errorf("Expected EvolutionCycles 0, got %d", metrics.EvolutionCycles)
	}
}

func TestAdaptiveEvolutionEngine_ResetUserProfile(t *testing.T) {
	engine := NewAdaptiveEvolutionEngine()

	_, _ = engine.AnalyzeUserAndGenerateChallenge(context.Background(), "user123", 1)

	err := engine.ResetUserProfile("user123")
	if err != nil {
		t.Fatalf("ResetUserProfile failed: %v", err)
	}

	profile, err := engine.GetUserProfile(context.Background(), "user123")
	if err == nil {
		t.Error("Should return error for nonexistent profile")
	}
	if profile != nil {
		t.Error("Profile should be nil after reset")
	}
}

func TestAdaptiveEvolutionEngine_PredictUserBehavior(t *testing.T) {
	engine := NewAdaptiveEvolutionEngine()

	profile := engine.getOrCreateUserProfile("user123")
	profile.SuccessHistory = []bool{true, true, true, true, true}
	profile.SuccessRate = 1.0
	profile.LearningRate = 0.5

	prediction, err := engine.PredictUserBehavior("user123", 10)
	if err != nil {
		t.Fatalf("PredictUserBehavior failed: %v", err)
	}

	if prediction < 0.5 || prediction > 1.0 {
		t.Errorf("Prediction should be between 0.5 and 1.0, got %.2f", prediction)
	}
}

func TestAdaptiveEvolutionEngine_GenerateAdaptiveHints(t *testing.T) {
	engine := NewAdaptiveEvolutionEngine()

	testCases := []struct {
		challengeType string
		difficulty    int
	}{
		{"visual", 2},
		{"audio", 3},
		{"tactile", 4},
	}

	for _, tc := range testCases {
		t.Run(tc.challengeType, func(t *testing.T) {
			challenge := &CaptchaChallenge{
				Type:       tc.challengeType,
				Difficulty: tc.difficulty,
			}

			hints := engine.generateAdaptiveHints(challenge)

			if len(hints) == 0 {
				t.Error("Should generate at least one hint")
			}
		})
	}
}

func TestAverageFunction(t *testing.T) {
	testCases := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"normal values", []float64{1, 2, 3, 4, 5}, 3.0},
		{"single value", []float64{10}, 10.0},
		{"empty", []float64{}, 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := average(tc.values)
			if result != tc.expected {
				t.Errorf("Expected %.2f, got %.2f", tc.expected, result)
			}
		})
	}
}

func TestSumFunction(t *testing.T) {
	testCases := []struct {
		name     string
		values   []int
		expected int
	}{
		{"normal values", []int{1, 2, 3, 4, 5}, 15},
		{"single value", []int{10}, 10},
		{"empty", []int{}, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sum(tc.values)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestBoolToFloatFunction(t *testing.T) {
	testCases := []struct {
		value    bool
		expected float64
	}{
		{true, 1.0},
		{false, 0.0},
	}

	for _, tc := range testCases {
		result := boolToFloat(tc.value)
		if result != tc.expected {
			t.Errorf("Expected %.2f, got %.2f", tc.expected, result)
		}
	}
}

func boolSlice(value float64, count int) []bool {
	result := make([]bool, count)
	for i := 0; i < count; i++ {
		result[i] = value >= 0.5
	}
	return result
}

func BenchmarkAdaptiveEvolutionEngine_AnalyzeUserAndGenerateChallenge(b *testing.B) {
	engine := NewAdaptiveEvolutionEngine()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.AnalyzeUserAndGenerateChallenge(context.Background(), "bench_user", 1)
	}
}

func BenchmarkAdaptiveEvolutionEngine_RecordAttempt(b *testing.B) {
	engine := NewAdaptiveEvolutionEngine()
	_, _ = engine.AnalyzeUserAndGenerateChallenge(context.Background(), "bench_user", 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.RecordAttempt(context.Background(), "bench_user", true, 5*time.Second, 3)
	}
}
