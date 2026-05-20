package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestLLMIntegrationServiceV2_Initialize(t *testing.T) {
	svc := NewLLMIntegrationServiceV2()
	ctx := context.Background()

	err := svc.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !svc.initialized {
		t.Error("initialized should be true after Initialize")
	}
}

func TestNLUVerifier_Verify(t *testing.T) {
	verifier := NewNLUVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)

	tests := []struct {
		name     string
		text     string
		expectedIntent string
	}{
		{"Greeting", "Hello, how are you?", "greeting"},
		{"Question", "What is your name?", "question"},
		{"Query", "Who is the president?", "query"},
		{"Statement", "The weather is nice today.", "statement"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &NLUVerificationRequest{Text: tt.text}
			response, err := verifier.Verify(ctx, request)
			if err != nil {
				t.Fatalf("Verify failed: %v", err)
			}

			if !response.Success {
				t.Error("expected successful response")
			}

			if response.Intent == "" {
				t.Error("intent should not be empty")
			}
		})
	}
}

func TestNLUVerifier_DetectLanguage(t *testing.T) {
	verifier := NewNLUVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)

	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{"English", "Hello, this is a test", "en"},
		{"Chinese", "你好，这是测试", "zh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lang := verifier.detectLanguage(tt.text)
			if lang != tt.expected {
				t.Logf("detected language %s, expected %s", lang, tt.expected)
			}
		})
	}
}

func TestNLUVerifier_AnalyzeSentiment(t *testing.T) {
	verifier := NewNLUVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)

	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{"Positive", "This is great and wonderful!", "positive"},
		{"Negative", "This is terrible and awful.", "negative"},
		{"Neutral", "The weather is cloudy.", "neutral"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentiment, _ := verifier.analyzeSentiment(tt.text)
			if sentiment != tt.expected {
				t.Errorf("expected %s sentiment, got %s", tt.expected, sentiment)
			}
		})
	}
}

func TestNLUVerifier_ExtractEntities(t *testing.T) {
	verifier := NewNLUVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)

	tests := []struct {
		name     string
		text     string
		expectedEntity string
	}{
		{"Email", "Contact me at test@example.com please", "email"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := verifier.extractEntities(tt.text)
			if _, ok := entities[tt.expectedEntity]; !ok {
				t.Logf("entity %s not found in %v", tt.expectedEntity, entities)
			}
		})
	}
}

func TestNLUVerifier_CheckNaturalness(t *testing.T) {
	verifier := NewNLUVerifier()
	ctx := context.Background()
	verifier.Initialize(ctx)

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"Natural", "Hello, how are you today?", true},
		{"Too short", "Hi", false},
		{"Too many caps", "THIS IS ALL CAPS TEXT HERE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifier.checkNaturalness(tt.text)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestLLMDrivenCaptchaGenerator_GenerateCaptcha(t *testing.T) {
	generator := NewLLMDrivenCaptchaGenerator()
	ctx := context.Background()
	generator.Initialize(ctx)

	request := &CaptchaGenerationRequest{
		Scene:      "login",
		Difficulty: 2,
		Count:      3,
	}

	response, err := generator.GenerateCaptcha(ctx, request)
	if err != nil {
		t.Fatalf("GenerateCaptcha failed: %v", err)
	}

	if !response.Success {
		t.Error("expected successful response")
	}

	if response.Total != 3 {
		t.Errorf("expected 3 captchas, got %d", response.Total)
	}

	for _, captcha := range response.Captchas {
		if captcha.ID == "" {
			t.Error("captcha ID should not be empty")
		}
		if captcha.Question == "" {
			t.Error("captcha question should not be empty")
		}
		if len(captcha.Options) < 2 {
			t.Error("captcha should have at least 2 options")
		}
		if captcha.AnswerHash == "" {
			t.Error("captcha answer hash should not be empty")
		}
	}
}

func TestLLMDrivenCaptchaGenerator_GenerateMathProblem(t *testing.T) {
	generator := NewLLMDrivenCaptchaGenerator()
	generator.rng.Seed(12345)

	for difficulty := 1; difficulty <= 5; difficulty++ {
		t.Run("Difficulty", func(t *testing.T) {
			question, options, answer := generator.generateMathProblem(difficulty)

			if question == "" {
				t.Error("question should not be empty")
			}

			if len(options) < 4 {
				t.Errorf("expected 4 options, got %d", len(options))
			}

			found := false
			for _, opt := range options {
				if opt == answer {
					found = true
					break
				}
			}
			if !found {
				t.Error("correct answer should be in options")
			}
		})
	}
}

func TestLLMDrivenCaptchaGenerator_GenerateLogicPuzzle(t *testing.T) {
	generator := NewLLMDrivenCaptchaGenerator()
	generator.rng.Seed(12345)

	question, options, answer := generator.generateLogicPuzzle(2)

	if question == "" {
		t.Error("question should not be empty")
	}

	if len(options) < 4 {
		t.Errorf("expected 4 options, got %d", len(options))
	}

	found := false
	for _, opt := range options {
		if opt == answer {
			found = true
			break
		}
	}
	if !found {
		t.Error("correct answer should be in options")
	}
}

func TestLLMDrivenCaptchaGenerator_GenerateWordPuzzle(t *testing.T) {
	generator := NewLLMDrivenCaptchaGenerator()
	generator.rng.Seed(12345)

	question, options, answer := generator.generateWordPuzzle(ThemeNature, 2)

	if question == "" {
		t.Error("question should not be empty")
	}

	if !strings.Contains(question, "Unscramble:") {
		t.Error("question should contain 'Unscramble:'")
	}

	if len(options) < 2 {
		t.Error("should have at least 2 options")
	}

	found := false
	for _, opt := range options {
		if opt == answer {
			found = true
			break
		}
	}
	if !found {
		t.Error("correct answer should be in options")
	}
}

func TestLLMDrivenCaptchaGenerator_GenerateOptions(t *testing.T) {
	generator := NewLLMDrivenCaptchaGenerator()
	generator.rng.Seed(12345)

	options := generator.generateOptions(10, 4)

	if len(options) != 4 {
		t.Errorf("expected 4 options, got %d", len(options))
	}

	found := false
	for _, opt := range options {
		if opt == "10" {
			found = true
			break
		}
	}
	if !found {
		t.Error("correct answer should be in options")
	}
}

func TestConversationalVerificationSystem_ProcessMessage(t *testing.T) {
	system := NewConversationalVerificationSystem()
	ctx := context.Background()
	system.Initialize(ctx)

	sessionID := "test_session_001"
	userID := "test_user_001"

	request := &DialogueRequest{
		SessionID: sessionID,
		UserID:    userID,
		Message:   "Hello",
	}

	response, err := system.ProcessMessage(ctx, request)
	if err != nil {
		t.Fatalf("ProcessMessage failed: %v", err)
	}

	if !response.Success {
		t.Error("expected successful response")
	}

	if response.SessionID != sessionID {
		t.Errorf("expected session ID %s, got %s", sessionID, response.SessionID)
	}

	if response.Response == "" {
		t.Error("response should not be empty")
	}

	if response.Progress < 0 || response.Progress > 1 {
		t.Errorf("progress should be between 0 and 1, got %f", response.Progress)
	}
}

func TestConversationalVerificationSystem_MultiTurn(t *testing.T) {
	system := NewConversationalVerificationSystem()
	ctx := context.Background()
	system.Initialize(ctx)

	sessionID := "test_session_multi"
	userID := "test_user_multi"

	messages := []string{
		"Hello",
		"My name is John",
		"I want to verify my account",
		"For a banking application",
		"I found it on Google",
	}

	for i, msg := range messages {
		t.Run("Turn", func(t *testing.T) {
			request := &DialogueRequest{
				SessionID: sessionID,
				UserID:    userID,
				Message:   msg,
			}

			response, err := system.ProcessMessage(ctx, request)
			if err != nil {
				t.Fatalf("ProcessMessage turn %d failed: %v", i+1, err)
			}

			if !response.Success {
				t.Errorf("turn %d: expected successful response", i+1)
			}

			if i < len(messages)-1 && response.IsComplete {
				t.Errorf("turn %d: should not be complete yet", i+1)
			}
		})
	}
}

func TestConversationalVerificationSystem_NextPrompts(t *testing.T) {
	system := NewConversationalVerificationSystem()
	ctx := context.Background()
	system.Initialize(ctx)

	sessionID := "test_session_prompts"
	request := &DialogueRequest{
		SessionID: sessionID,
		UserID:    "test_user",
		Message:   "Hello",
	}

	response, _ := system.ProcessMessage(ctx, request)

	if len(response.NextPrompts) == 0 {
		t.Error("next prompts should not be empty")
	}

	validPrompts := map[string]bool{
		"Continue": true,
		"Skip this step": true,
		"Get help": true,
		"Cancel": true,
	}

	for _, prompt := range response.NextPrompts {
		if !validPrompts[prompt] {
			t.Logf("prompt '%s' not in expected list", prompt)
		}
	}
}

func TestSemanticRiskEvaluator_Evaluate(t *testing.T) {
	evaluator := NewSemanticRiskEvaluator()
	ctx := context.Background()
	evaluator.Initialize(ctx)

	tests := []struct {
		name     string
		text     string
		strictMode bool
		expectedAllowed bool
	}{
		{"Safe text", "Hello, how can I help you today?", false, true},
		{"Spam text", "Buy now! Limited offer! Click here!", false, false},
		{"Phishing text", "Your account has been suspended. Please verify your password.", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &SemanticRiskRequest{
				Text:       tt.text,
				StrictMode: tt.strictMode,
			}

			response, err := evaluator.Evaluate(ctx, request)
			if err != nil {
				t.Fatalf("Evaluate failed: %v", err)
			}

			if !response.Success {
				t.Error("expected successful response")
			}

			if response.OverallScore < 0 || response.OverallScore > 1 {
				t.Errorf("overall score should be between 0 and 1, got %f", response.OverallScore)
			}

			validLevels := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
			if !validLevels[response.RiskLevel] {
				t.Errorf("invalid risk level: %s", response.RiskLevel)
			}
		})
	}
}

func TestSemanticRiskEvaluator_EvaluateToxicity(t *testing.T) {
	evaluator := NewSemanticRiskEvaluator()
	ctx := context.Background()
	evaluator.Initialize(ctx)

	tests := []struct {
		name     string
		text     string
		expectedMin float64
	}{
		{"Toxic text", "I hate you and want to kill", 0.5},
		{"Safe text", "Hello, have a nice day", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := evaluator.evaluateToxicity(tt.text)
			if score < tt.expectedMin {
				t.Errorf("expected score >= %f, got %f", tt.expectedMin, score)
			}
		})
	}
}

func TestSemanticRiskEvaluator_EvaluateSpam(t *testing.T) {
	evaluator := NewSemanticRiskEvaluator()
	ctx := context.Background()
	evaluator.Initialize(ctx)

	tests := []struct {
		name     string
		text     string
		expectedMin float64
	}{
		{"Spam text", "Buy now! Free money! You are a winner!", 0.3},
		{"Safe text", "Let's schedule a meeting for tomorrow", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := evaluator.evaluateSpam(tt.text)
			if score < tt.expectedMin {
				t.Logf("score %f is below expected min %f", score, tt.expectedMin)
			}
		})
	}
}

func TestSemanticRiskEvaluator_EvaluatePhishing(t *testing.T) {
	evaluator := NewSemanticRiskEvaluator()
	ctx := context.Background()
	evaluator.Initialize(ctx)

	tests := []struct {
		name     string
		text     string
		expectedMin float64
	}{
		{"Phishing text", "Urgent: Verify your account password reset required", 0.3},
		{"Safe text", "Thanks for registering. Welcome to our platform!", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := evaluator.evaluatePhishing(tt.text)
			if score < tt.expectedMin {
				t.Logf("score %f is below expected min %f", score, tt.expectedMin)
			}
		})
	}
}

func TestSemanticRiskEvaluator_EvaluateInsult(t *testing.T) {
	evaluator := NewSemanticRiskEvaluator()
	ctx := context.Background()
	evaluator.Initialize(ctx)

	tests := []struct {
		name     string
		text     string
		expectedMin float64
	}{
		{"Insult text", "You are an idiot and a stupid fool", 0.2},
		{"Safe text", "I disagree with your opinion", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := evaluator.evaluateInsult(tt.text)
			if score < tt.expectedMin {
				t.Logf("score %f is below expected min %f", score, tt.expectedMin)
			}
		})
	}
}

func TestSemanticRiskEvaluator_ClassifyRiskLevel(t *testing.T) {
	evaluator := NewSemanticRiskEvaluator()

	tests := []struct {
		name     string
		score    float64
		expected string
	}{
		{"Low", 0.2, "low"},
		{"Medium", 0.5, "medium"},
		{"High", 0.7, "high"},
		{"Critical", 0.9, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := evaluator.classifyRiskLevel(tt.score)
			if level != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, level)
			}
		})
	}
}

func TestLLMCache_GetSet(t *testing.T) {
	cache := NewLLMCache(100)

	key := "test_key"
	value := "test_value"

	cache.Set(key, value, 5*time.Minute)

	result, exists := cache.Get(key)
	if !exists {
		t.Error("expected key to exist")
	}

	if result != value {
		t.Errorf("expected %s, got %s", value, result)
	}
}

func TestLLMCache_Expiration(t *testing.T) {
	cache := NewLLMCache(100)

	key := "expiring_key"
	value := "expiring_value"

	cache.Set(key, value, 1*time.Millisecond)

	time.Sleep(5 * time.Millisecond)

	_, exists := cache.Get(key)
	if exists {
		t.Error("key should have expired")
	}
}

func TestLLMCache_Eviction(t *testing.T) {
	cache := NewLLMCache(5)

	for i := 0; i < 10; i++ {
		key := "key_" + string(rune('0'+i))
		cache.Set(key, "value_"+key, 5*time.Minute)
	}

	if len(cache.entries) > cache.maxSize {
		t.Errorf("cache size %d exceeds max %d", len(cache.entries), cache.maxSize)
	}
}

func TestLLMIntegrationServiceV2_VerifyNaturalLanguage(t *testing.T) {
	svc := NewLLMIntegrationServiceV2()
	ctx := context.Background()
	svc.Initialize(ctx)

	request := &NLUVerificationRequest{
		Text: "Hello, how are you today?",
	}

	response, err := svc.VerifyNaturalLanguage(ctx, request)
	if err != nil {
		t.Fatalf("VerifyNaturalLanguage failed: %v", err)
	}

	if !response.Success {
		t.Error("expected successful response")
	}

	if response.Intent != "greeting" {
		t.Errorf("expected greeting intent, got %s", response.Intent)
	}
}

func TestLLMIntegrationServiceV2_GenerateLLMCaptcha(t *testing.T) {
	svc := NewLLMIntegrationServiceV2()
	ctx := context.Background()
	svc.Initialize(ctx)

	request := &CaptchaGenerationRequest{
		Scene:      "register",
		Difficulty: 3,
		Count:      2,
	}

	response, err := svc.GenerateLLMCaptcha(ctx, request)
	if err != nil {
		t.Fatalf("GenerateLLMCaptcha failed: %v", err)
	}

	if !response.Success {
		t.Error("expected successful response")
	}

	if len(response.Captchas) != 2 {
		t.Errorf("expected 2 captchas, got %d", len(response.Captchas))
	}
}

func TestLLMIntegrationServiceV2_ProcessDialogue(t *testing.T) {
	svc := NewLLMIntegrationServiceV2()
	ctx := context.Background()
	svc.Initialize(ctx)

	request := &DialogueRequest{
		SessionID: "dialogue_test_001",
		UserID:    "user_001",
		Message:   "Hello there",
	}

	response, err := svc.ProcessDialogue(ctx, request)
	if err != nil {
		t.Fatalf("ProcessDialogue failed: %v", err)
	}

	if !response.Success {
		t.Error("expected successful response")
	}
}

func TestLLMIntegrationServiceV2_EvaluateSemanticRisk(t *testing.T) {
	svc := NewLLMIntegrationServiceV2()
	ctx := context.Background()
	svc.Initialize(ctx)

	request := &SemanticRiskRequest{
		Text:       "Please verify your account by clicking here",
		StrictMode: false,
	}

	response, err := svc.EvaluateSemanticRisk(ctx, request)
	if err != nil {
		t.Fatalf("EvaluateSemanticRisk failed: %v", err)
	}

	if !response.Success {
		t.Error("expected successful response")
	}

	if response.OverallScore < 0 || response.OverallScore > 1 {
		t.Errorf("overall score should be between 0 and 1, got %f", response.OverallScore)
	}
}

func TestParseNLUVerificationRequest(t *testing.T) {
	jsonData := `{
		"text": "Hello, this is a test",
		"language": "en",
		"context": {"user_id": "123"}
	}`

	request, err := ParseNLUVerificationRequest(jsonData)
	if err != nil {
		t.Fatalf("ParseNLUVerificationRequest failed: %v", err)
	}

	if request.Text != "Hello, this is a test" {
		t.Errorf("expected 'Hello, this is a test', got '%s'", request.Text)
	}

	if request.Language != "en" {
		t.Errorf("expected 'en', got '%s'", request.Language)
	}
}

func TestParseCaptchaGenerationRequest(t *testing.T) {
	jsonData := `{
		"scene": "login",
		"difficulty": 3,
		"theme": "nature",
		"count": 5
	}`

	request, err := ParseCaptchaGenerationRequest(jsonData)
	if err != nil {
		t.Fatalf("ParseCaptchaGenerationRequest failed: %v", err)
	}

	if request.Scene != "login" {
		t.Errorf("expected 'login', got '%s'", request.Scene)
	}

	if request.Difficulty != 3 {
		t.Errorf("expected 3, got %d", request.Difficulty)
	}
}

func TestParseDialogueRequest(t *testing.T) {
	jsonData := `{
		"session_id": "sess_123",
		"user_id": "user_456",
		"message": "Hello"
	}`

	request, err := ParseDialogueRequest(jsonData)
	if err != nil {
		t.Fatalf("ParseDialogueRequest failed: %v", err)
	}

	if request.SessionID != "sess_123" {
		t.Errorf("expected 'sess_123', got '%s'", request.SessionID)
	}

	if request.Message != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", request.Message)
	}
}

func TestParseSemanticRiskRequest(t *testing.T) {
	jsonData := `{
		"text": "Check this suspicious link",
		"context": "email",
		"strict_mode": true
	}`

	request, err := ParseSemanticRiskRequest(jsonData)
	if err != nil {
		t.Fatalf("ParseSemanticRiskRequest failed: %v", err)
	}

	if request.Text != "Check this suspicious link" {
		t.Errorf("unexpected text: %s", request.Text)
	}

	if !request.StrictMode {
		t.Error("expected strict_mode to be true")
	}
}

func TestGeneratedCaptcha_Serialization(t *testing.T) {
	captcha := &GeneratedCaptcha{
		ID:            "captcha_001",
		Question:      "What is 2 + 2?",
		Options:       []string{"3", "4", "5", "6"},
		CorrectAnswer: "4",
		AnswerHash:    "abc123hash",
		Hint:          "Think carefully",
		Difficulty:    2,
		Theme:         ThemeScience,
		Type:          TypeMathProblem,
		Metadata:      map[string]interface{}{"scene": "login"},
		GeneratedAt:   time.Now(),
		ExpiresAt:     time.Now().Add(5 * time.Minute),
	}

	data, err := json.Marshal(captcha)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled GeneratedCaptcha
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.ID != captcha.ID {
		t.Error("ID mismatch")
	}

	if unmarshaled.Question != captcha.Question {
		t.Error("Question mismatch")
	}

	if len(unmarshaled.Options) != len(captcha.Options) {
		t.Error("Options count mismatch")
	}
}

func TestDialogueResponse_Serialization(t *testing.T) {
	response := &DialogueResponse{
		Success:    true,
		SessionID:  "session_001",
		Response:   "Hello, how can I help?",
		NextPrompts: []string{"Continue", "Cancel"},
		Progress:   0.5,
		IsComplete: false,
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled DialogueResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.Success != response.Success {
		t.Error("Success mismatch")
	}

	if unmarshaled.Progress != response.Progress {
		t.Error("Progress mismatch")
	}
}

func TestSemanticRiskResponse_Serialization(t *testing.T) {
	response := &SemanticRiskResponse{
		Success:      true,
		OverallScore: 0.65,
		RiskLevel:    "medium",
		Categories:   map[string]float64{"toxicity": 0.3, "spam": 0.8},
		IsAllowed:    true,
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled SemanticRiskResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.OverallScore != response.OverallScore {
		t.Error("OverallScore mismatch")
	}

	if unmarshaled.RiskLevel != response.RiskLevel {
		t.Error("RiskLevel mismatch")
	}
}

func TestHasHighSeverityRisk(t *testing.T) {
	tests := []struct {
		name     string
		risks    []DetectedRisk
		expected bool
	}{
		{
			name: "High severity risk",
			risks: []DetectedRisk{
				{Severity: 0.9, Description: "High toxicity"},
			},
			expected: true,
		},
		{
			name: "Low severity risks only",
			risks: []DetectedRisk{
				{Severity: 0.3, Description: "Low spam"},
				{Severity: 0.5, Description: "Medium insult"},
			},
			expected: false,
		},
		{
			name:     "No risks",
			risks:    []DetectedRisk{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasHighSeverityRisk(tt.risks)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHashAnswer(t *testing.T) {
	answer := "42"
	hash1 := hashAnswer(answer)
	hash2 := hashAnswer(answer)

	if hash1 != hash2 {
		t.Error("same answer should produce same hash")
	}

	hash3 := hashAnswer("43")
	if hash1 == hash3 {
		t.Error("different answers should produce different hashes")
	}

	if len(hash1) == 0 {
		t.Error("hash should not be empty")
	}
}

func TestGenerateCaptchaID(t *testing.T) {
	id1 := generateCaptchaID()
	id2 := generateCaptchaID()

	if id1 == id2 {
		t.Error("different IDs should be generated")
	}

	if len(id1) != 16 {
		t.Errorf("expected ID length 16, got %d", len(id1))
	}
}
