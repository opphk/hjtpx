package service

import (
	"context"
	"testing"
	"time"
)

func TestAdaptiveVerificationEngine(t *testing.T) {
	engine := NewAdaptiveVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	if !engine.initialized {
		t.Error("Engine should be initialized")
	}
}

func TestRealtimeRiskAssessor(t *testing.T) {
	assessor := NewRealtimeRiskAssessor()
	ctx := context.Background()

	if err := assessor.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize assessor: %v", err)
	}

	result, err := assessor.AssessRisk(ctx, nil, nil)
	if err != nil {
		t.Fatalf("Failed to assess risk: %v", err)
	}

	if result == nil {
		t.Fatal("Risk result should not be nil")
	}

	if result.RiskScore < 0 || result.RiskScore > 100 {
		t.Errorf("Risk score should be between 0 and 100, got %f", result.RiskScore)
	}
}

func TestDynamicStrategyEngine(t *testing.T) {
	engine := NewDynamicStrategyEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize strategy engine: %v", err)
	}

	riskResult := &AdaptiveRiskResult{
		RiskScore:      75,
		RiskLevel:     "high",
		ContributingFactors: []string{"异常鼠标移动速度"},
	}

	result, err := engine.DetermineStrategy(ctx, riskResult, nil)
	if err != nil {
		t.Fatalf("Failed to determine strategy: %v", err)
	}

	if result == nil {
		t.Fatal("Strategy result should not be nil")
	}

	if result.Strategy == nil {
		t.Error("Strategy should not be nil")
	}

	if result.Strategy.Difficulty < 1 || result.Strategy.Difficulty > 5 {
		t.Errorf("Difficulty should be between 1 and 5, got %d", result.Strategy.Difficulty)
	}
}

func TestPersonalizationEngine(t *testing.T) {
	engine := NewPersonalizationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize personalization engine: %v", err)
	}

	riskResult := &AdaptiveRiskResult{
		RiskScore: 50,
		RiskLevel: "medium",
	}

	result, err := engine.GetPersonalization(ctx, "test_user", riskResult)
	if err != nil {
		t.Fatalf("Failed to get personalization: %v", err)
	}

	if result == nil {
		t.Fatal("Personalization result should not be nil")
	}

	if result.PreferredCaptcha == "" {
		t.Error("Preferred captcha should be set")
	}

	if result.OptimalDifficulty < 1 || result.OptimalDifficulty > 5 {
		t.Errorf("Optimal difficulty should be between 1 and 5, got %d", result.OptimalDifficulty)
	}
}

func TestSelfLearningEngine(t *testing.T) {
	engine := NewSelfLearningEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize learning engine: %v", err)
	}

	features := []float64{0.5, 0.6, 0.7, 0.8}
	if err := engine.RecordSample(ctx, features, true, nil); err != nil {
		t.Fatalf("Failed to record sample: %v", err)
	}

	if len(engine.trainingData) != 1 {
		t.Errorf("Expected 1 training sample, got %d", len(engine.trainingData))
	}

	updateResult, err := engine.UpdateModel(ctx)
	if err != nil {
		t.Fatalf("Failed to update model: %v", err)
	}

	if updateResult == nil {
		t.Fatal("Update result should not be nil")
	}
}

func TestPerformAdaptiveAssessment(t *testing.T) {
	engine := NewAdaptiveVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	result, err := engine.PerformAdaptiveAssessment(ctx, nil, nil, "test_user")
	if err != nil {
		t.Fatalf("Failed to perform adaptive assessment: %v", err)
	}

	if result == nil {
		t.Fatal("Assessment result should not be nil")
	}

	if result["recommended_action"] == nil {
		t.Error("Recommended action should be set")
	}
}

func TestRecordInteraction(t *testing.T) {
	engine := NewAdaptiveVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	interaction := &InteractionData{
		UserID:         "test_user",
		Difficulty:     3,
		Success:        true,
		CompletionTime: 5 * time.Second,
		CaptchaType:   "slider",
	}

	if err := engine.RecordInteraction(ctx, interaction); err != nil {
		t.Fatalf("Failed to record interaction: %v", err)
	}
}
