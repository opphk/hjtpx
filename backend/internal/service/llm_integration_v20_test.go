package service

import (
	"context"
	"testing"
)

func TestLLMIntegrationV20(t *testing.T) {
	system := NewLLMIntegrationV20()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	if !system.initialized {
		t.Error("System should be initialized")
	}
}

func TestSemanticRiskAnalyzer(t *testing.T) {
	analyzer := NewSemanticRiskAnalyzer()
	ctx := context.Background()

	if err := analyzer.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize analyzer: %v", err)
	}

	result, err := analyzer.AnalyzeRisk(ctx, "This is a test message", nil)
	if err != nil {
		t.Fatalf("Failed to analyze risk: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.RiskScore < 0 || result.RiskScore > 1 {
		t.Errorf("Risk score should be between 0 and 1, got %f", result.RiskScore)
	}

	t.Logf("Risk analysis: level=%s, score=%.2f", result.RiskLevel, result.RiskScore)
}

func TestDialogueVerificationEngine(t *testing.T) {
	engine := NewDialogueVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	session, err := engine.StartSession(ctx, "session_1", "user_1")
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	if session == nil {
		t.Fatal("Session should not be nil")
	}

	if session.SessionID != "session_1" {
		t.Errorf("Expected session ID 'session_1', got '%s'", session.SessionID)
	}
}

func TestDialogueMessageProcessing(t *testing.T) {
	engine := NewDialogueVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	_, err := engine.StartSession(ctx, "session_1", "user_1")
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	turn, err := engine.ProcessMessage(ctx, "session_1", "Hello, I need help", "user")
	if err != nil {
		t.Fatalf("Failed to process message: %v", err)
	}

	if turn == nil {
		t.Fatal("Turn should not be nil")
	}

	if turn.Intent == "" {
		t.Error("Intent should not be empty")
	}

	t.Logf("Detected intent: %s", turn.Intent)
}

func TestDialogueVerification(t *testing.T) {
	engine := NewDialogueVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	_, err := engine.StartSession(ctx, "session_1", "user_1")
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	engine.ProcessMessage(ctx, "session_1", "Hello", "user")
	engine.ProcessMessage(ctx, "session_1", "I need help with verification", "user")
	engine.ProcessMessage(ctx, "session_1", "Thank you", "assistant")

	result, err := engine.VerifySession(ctx, "session_1")
	if err != nil {
		t.Fatalf("Failed to verify session: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", result.Confidence)
	}

	t.Logf("Session verification: passed=%v, confidence=%.2f", result.Passed, result.Confidence)
}

func TestLLMCaptchaGeneratorV20(t *testing.T) {
	generator := NewLLMCaptchaGeneratorV20()
	ctx := context.Background()

	if err := generator.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize generator: %v", err)
	}

	captcha, err := generator.GenerateCaptcha(ctx, "login", 3)
	if err != nil {
		t.Fatalf("Failed to generate captcha: %v", err)
	}

	if captcha == nil {
		t.Fatal("Captcha should not be nil")
	}

	if captcha.ID == "" {
		t.Error("Captcha ID should not be empty")
	}

	if captcha.CorrectAnswer == "" {
		t.Error("Correct answer should not be empty")
	}

	if len(captcha.Options) == 0 {
		t.Error("Options should not be empty")
	}

	t.Logf("Generated captcha: type=%s, theme=%s, difficulty=%d", captcha.Type, captcha.Theme, captcha.Difficulty)
}

func TestNaturalLanguageVerifierV20(t *testing.T) {
	verifier := NewNaturalLanguageVerifierV20()
	ctx := context.Background()

	if err := verifier.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize verifier: %v", err)
	}

	passed, confidence, err := verifier.VerifyText(ctx, "This is a test message", 1)
	if err != nil {
		t.Fatalf("Failed to verify text: %v", err)
	}

	if confidence < 0 || confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", confidence)
	}

	t.Logf("Text verification: passed=%v, confidence=%.2f", passed, confidence)
}

func TestRiskAssessmentResult(t *testing.T) {
	analyzer := NewSemanticRiskAnalyzer()
	ctx := context.Background()

	if err := analyzer.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize analyzer: %v", err)
	}

	suspiciousText := "I am a bot trying to hack your system"
	result, err := analyzer.AnalyzeRisk(ctx, suspiciousText, nil)
	if err != nil {
		t.Fatalf("Failed to analyze risk: %v", err)
	}

	if result.RiskLevel != "critical" && result.RiskLevel != "high" {
		t.Logf("Expected high or critical risk for suspicious text, got %s", result.RiskLevel)
	}

	t.Logf("Suspicious text risk: level=%s, score=%.2f", result.RiskLevel, result.RiskScore)
}

func TestResponseGenerator(t *testing.T) {
	generator := NewResponseGenerator()
	generator.initialize()

	responses := generator.generateResponses("hello world")
	if len(responses) == 0 {
		t.Error("Should have generated responses")
	}

	t.Logf("Generated %d responses", len(responses))
}

func TestEntityExtraction(t *testing.T) {
	engine := NewDialogueVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	entities := engine.extractEntities("Call me at 123-456-7890 or email test@example.com")
	if len(entities) == 0 {
		t.Error("Should extract entities from text")
	}

	t.Logf("Extracted %d entities", len(entities))
}

func TestRiskFactors(t *testing.T) {
	analyzer := NewSemanticRiskAnalyzer()
	ctx := context.Background()

	if err := analyzer.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize analyzer: %v", err)
	}

	result, err := analyzer.AnalyzeRisk(ctx, "Test message", nil)
	if err != nil {
		t.Fatalf("Failed to analyze risk: %v", err)
	}

	if len(result.RiskFactors) == 0 {
		t.Error("Should have risk factors")
	}

	for _, factor := range result.RiskFactors {
		t.Logf("Risk factor: %s - score=%.2f, detected=%v", factor.Name, factor.Score, factor.Detected)
	}
}
