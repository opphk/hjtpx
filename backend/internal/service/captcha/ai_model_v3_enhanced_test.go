package captcha

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestEnhancedGPTCaptchaGenerator_GenerateCaptcha(t *testing.T) {
	generator := NewEnhancedGPTCaptchaGenerator()

	err := generator.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	prompt, err := generator.GenerateCaptcha(context.Background(), CaptchaSceneLogin, 3, "zh-CN")
	if err != nil {
		t.Fatalf("GenerateCaptcha failed: %v", err)
	}

	if prompt.CaptchaID == "" {
		t.Error("CaptchaID should not be empty")
	}

	if prompt.Question == "" {
		t.Error("Question should not be empty")
	}

	if len(prompt.Options) == 0 {
		t.Error("Options should not be empty")
	}

	if prompt.Language != "zh-CN" {
		t.Errorf("Expected language 'zh-CN', got '%s'", prompt.Language)
	}

	if prompt.Difficulty != 3 {
		t.Errorf("Expected difficulty 3, got %d", prompt.Difficulty)
	}

	if prompt.GeneratedAt <= 0 {
		t.Error("GeneratedAt should be set")
	}

	if prompt.ExpiresAt <= prompt.GeneratedAt {
		t.Error("ExpiresAt should be after GeneratedAt")
	}
}

func TestEnhancedGPTCaptchaGenerator_GenerateCaptcha_English(t *testing.T) {
	generator := NewEnhancedGPTCaptchaGenerator()

	err := generator.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	prompt, err := generator.GenerateCaptcha(context.Background(), CaptchaScenePayment, 2, "en-US")
	if err != nil {
		t.Fatalf("GenerateCaptcha failed: %v", err)
	}

	if prompt.Language != "en-US" {
		t.Errorf("Expected language 'en-US', got '%s'", prompt.Language)
	}

	if prompt.Type != "gpt_enhanced" {
		t.Errorf("Expected type 'gpt_enhanced', got '%s'", prompt.Type)
	}
}

func TestEnhancedGPTCaptchaGenerator_GenerateCaptcha_DefaultLanguage(t *testing.T) {
	generator := NewEnhancedGPTCaptchaGenerator()

	err := generator.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	prompt, err := generator.GenerateCaptcha(context.Background(), CaptchaSceneGeneral, 2, "unknown-lang")
	if err != nil {
		t.Fatalf("GenerateCaptcha should fallback to English: %v", err)
	}

	if prompt.Language != "en-US" {
		t.Errorf("Expected fallback to 'en-US', got '%s'", prompt.Language)
	}
}

func TestEnhancedGPTCaptchaGenerator_NotInitialized(t *testing.T) {
	generator := NewEnhancedGPTCaptchaGenerator()

	_, err := generator.GenerateCaptcha(context.Background(), CaptchaSceneLogin, 2, "en-US")
	if err == nil {
		t.Error("Should return error when not initialized")
	}
}

func TestEnhancedGPTCaptchaGenerator_GenerateOptions(t *testing.T) {
	generator := NewEnhancedGPTCaptchaGenerator()

	testCases := []struct {
		theme     CaptchaThemeType
		language  string
		difficulty int
	}{
		{CaptchaThemeNature, "zh-CN", 2},
		{CaptchaThemeNature, "en-US", 3},
		{CaptchaThemeCity, "zh-CN", 2},
		{CaptchaThemeCity, "en-US", 3},
		{CaptchaThemeAbstract, "zh-CN", 2},
		{CaptchaThemeAbstract, "en-US", 3},
		{CaptchaThemeGame, "zh-CN", 2},
		{CaptchaThemeGame, "en-US", 3},
	}

	for _, tc := range testCases {
		t.Run(string(tc.theme)+"_"+tc.language, func(t *testing.T) {
			options := generator.generateOptions(tc.theme, tc.difficulty, tc.language)
			expectedMin := 3 + tc.difficulty
			if len(options) < expectedMin {
				t.Errorf("Expected at least %d options, got %d", expectedMin, len(options))
			}
		})
	}
}

func TestEnhancedGPTCaptchaGenerator_GenerateHint(t *testing.T) {
	generator := NewEnhancedGPTCaptchaGenerator()

	languages := []string{"zh-CN", "en-US"}

	for _, lang := range languages {
		for difficulty := 1; difficulty <= 5; difficulty++ {
			hint := generator.generateHint(difficulty, lang)
			if hint == "" {
				t.Errorf("Hint should not be empty for language %s difficulty %d", lang, difficulty)
			}
		}
	}
}

func TestLSTMFeatureExtractor_ExtractFeatures(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	traceData := &model.TraceData{
		PointCount:         50,
		TotalTime:          5000,
		TotalDistance:      100.5,
		AvgDistance:        2.01,
		AvgSpeed:           20.1,
		SpeedVariance:      0.5,
		MinSpeed:           10.0,
		MaxSpeed:           30.0,
		DirectionChanges:   5,
		AvgCurvature:       0.1,
		CurvatureVariance:  0.05,
	}

	features, err := extractor.ExtractFeatures(context.Background(), traceData)
	if err != nil {
		t.Fatalf("ExtractFeatures failed: %v", err)
	}

	if len(features) != 20 {
		t.Errorf("Expected 20 features, got %d", len(features))
	}

	if features[0] != 50 {
		t.Errorf("Expected PointCount 50, got %f", features[0])
	}

	if features[1] != 5.0 {
		t.Errorf("Expected TotalTime/1000 = 5.0, got %f", features[1])
	}
}

func TestLSTMFeatureExtractor_ExtractFeatures_NilTrace(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	features, err := extractor.ExtractFeatures(context.Background(), nil)
	if err != nil {
		t.Fatalf("ExtractFeatures should not return error for nil trace: %v", err)
	}

	if len(features) != 20 {
		t.Errorf("Expected 20 default features, got %d", len(features))
	}
}

func TestLSTMFeatureExtractor_CalculateSmoothness(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	testCases := []struct {
		name            string
		traceData       *model.TraceData
		expectMin       float64
	}{
		{"low variance", &model.TraceData{CurvatureVariance: 0.01}, 0.9},
		{"high variance", &model.TraceData{CurvatureVariance: 1.0}, 0.4},
		{"zero variance", &model.TraceData{CurvatureVariance: 0}, 1.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractor.calculateSmoothness(tc.traceData)
			if result < tc.expectMin {
				t.Errorf("Expected smoothness >= %f, got %f", tc.expectMin, result)
			}
		})
	}
}

func TestLSTMFeatureExtractor_CalculateNaturalness(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	testCases := []struct {
		name        string
		avgSpeed    float64
		expectMin   float64
		expectMax   float64
	}{
		{"normal speed", 0.2, 0.7, 0.9},
		{"too slow", 0.05, 0.2, 0.4},
		{"too fast", 0.5, 0.2, 0.4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			traceData := &model.TraceData{AvgSpeed: tc.avgSpeed}
			result := extractor.calculateNaturalness(traceData)
			if result < tc.expectMin || result > tc.expectMax {
				t.Errorf("Expected naturalness between %f and %f, got %f", tc.expectMin, tc.expectMax, result)
			}
		})
	}
}

func TestEnhancedRiskAssessor_AssessRisk(t *testing.T) {
	assessor := NewEnhancedRiskAssessor()

	err := assessor.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	features := make([]float64, 20)
	for i := range features {
		features[i] = 0.5 + (rand.Float64() - 0.5)*0.3
	}

	traceData := &model.TraceData{
		PointCount:        50,
		TotalTime:        5000,
		TotalDistance:    100.0,
		AvgSpeed:         20.0,
		SpeedVariance:    0.3,
		DirectionChanges: 5,
	}

	result, err := assessor.AssessRisk(context.Background(), features, nil, traceData)
	if err != nil {
		t.Fatalf("AssessRisk failed: %v", err)
	}

	if result.RiskScore < 0 || result.RiskScore > 1 {
		t.Errorf("RiskScore should be between 0 and 1, got %f", result.RiskScore)
	}

	if result.RiskLevel == "" {
		t.Error("RiskLevel should be set")
	}

	if result.ModelVersion != "v3.0-enhanced" {
		t.Errorf("Expected model version 'v3.0-enhanced', got '%s'", result.ModelVersion)
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", result.Confidence)
	}
}

func TestEnhancedRiskAssessor_AssessRisk_NotInitialized(t *testing.T) {
	assessor := NewEnhancedRiskAssessor()

	_, err := assessor.AssessRisk(context.Background(), make([]float64, 20), nil, nil)
	if err == nil {
		t.Error("Should return error when not initialized")
	}
}

func TestEnhancedRiskAssessor_CalculateAnomalyScore(t *testing.T) {
	assessor := NewEnhancedRiskAssessor()

	testCases := []struct {
		name      string
		features  []float64
		traceData *model.TraceData
		expectMax float64
	}{
		{
			"normal behavior",
			make([]float64, 20),
			&model.TraceData{SpeedVariance: 0.3, DirectionChanges: 5, PointCount: 50},
			0.8,
		},
		{
			"suspicious behavior",
			make([]float64, 20),
			&model.TraceData{SpeedVariance: 0.05, DirectionChanges: 0, PointCount: 3},
			1.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := assessor.calculateAnomalyScore(tc.features, tc.traceData)
			if score > tc.expectMax {
				t.Errorf("Expected anomaly score <= %f, got %f", tc.expectMax, score)
			}
		})
	}
}

func TestEnhancedRiskAssessor_DetectThreats(t *testing.T) {
	assessor := NewEnhancedRiskAssessor()

	features := make([]float64, 20)
	traceData := &model.TraceData{
		SpeedVariance:    0.05,
		DirectionChanges: 0,
		PointCount:       1001,
	}

	threats := assessor.detectThreats(features, traceData)

	if len(threats) == 0 {
		t.Error("Should detect at least one threat")
	}

	hasSuspiciousSpeed := false
	for _, threat := range threats {
		if threat.Type == "suspicious_speed" {
			hasSuspiciousSpeed = true
			break
		}
	}

	if !hasSuspiciousSpeed {
		t.Error("Should detect suspicious_speed threat")
	}
}

func TestEnhancedBehaviorLearningSystem_LearnFromExample(t *testing.T) {
	system := NewEnhancedBehaviorLearningSystem()

	err := system.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	features := make([]float64, 10)
	for i := range features {
		features[i] = rand.Float64()
	}

	err = system.LearnFromExample(context.Background(), features, true, 0.9, nil)
	if err != nil {
		t.Fatalf("LearnFromExample failed: %v", err)
	}

	pattern, similarity, err := system.MatchPattern(context.Background(), features)
	if err != nil {
		t.Fatalf("MatchPattern failed: %v", err)
	}

	if pattern == nil {
		t.Error("Should match the learned pattern")
	}

	if similarity < 0.7 {
		t.Errorf("Expected similarity >= 0.7, got %f", similarity)
	}
}

func TestEnhancedBehaviorLearningSystem_MultipleLearning(t *testing.T) {
	system := NewEnhancedBehaviorLearningSystem()

	features1 := make([]float64, 10)
	for i := range features1 {
		features1[i] = 0.3
	}

	features2 := make([]float64, 10)
	for i := range features2 {
		features2[i] = 0.7
	}

	features3 := make([]float64, 10)
	for i := range features3 {
		features3[i] = 0.3
	}

	err := system.LearnFromExample(context.Background(), features1, false, 0.8, nil)
	if err != nil {
		t.Fatalf("LearnFromExample 1 failed: %v", err)
	}

	err = system.LearnFromExample(context.Background(), features2, true, 0.9, nil)
	if err != nil {
		t.Fatalf("LearnFromExample 2 failed: %v", err)
	}

	err = system.LearnFromExample(context.Background(), features3, false, 0.7, nil)
	if err != nil {
		t.Fatalf("LearnFromExample 3 failed: %v", err)
	}

	stats, err := system.GetStatistics(context.Background())
	if err != nil {
		t.Fatalf("GetStatistics failed: %v", err)
	}

	if stats["total_patterns"].(int) < 1 {
		t.Errorf("Should have at least 1 pattern, got %d", stats["total_patterns"])
	}
}

func TestEnhancedBehaviorLearningSystem_GetStatistics(t *testing.T) {
	system := NewEnhancedBehaviorLearningSystem()

	features := make([]float64, 10)
	for i := range features {
		features[i] = rand.Float64()
	}

	system.LearnFromExample(context.Background(), features, true, 0.9, nil)

	stats, err := system.GetStatistics(context.Background())
	if err != nil {
		t.Fatalf("GetStatistics failed: %v", err)
	}

	expectedFields := []string{
		"total_patterns",
		"bot_patterns",
		"human_patterns",
		"total_occurrences",
		"high_risk_patterns",
		"medium_risk_patterns",
		"low_risk_patterns",
	}

	for _, field := range expectedFields {
		if _, ok := stats[field]; !ok {
			t.Errorf("Statistics should include '%s' field", field)
		}
	}
}

func TestEnhancedBehaviorLearningSystem_DetermineRiskLevel(t *testing.T) {
	system := NewEnhancedBehaviorLearningSystem()

	testCases := []struct {
		name     string
		features []float64
		expected string
	}{
		{"high risk", []float64{0.9, 0.8, 0.85}, "high"},
		{"medium risk", []float64{0.5, 0.4, 0.6}, "medium"},
		{"low risk", []float64{0.2, 0.1, 0.3}, "low"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := system.determineRiskLevel(tc.features)
			if result != tc.expected {
				t.Errorf("Expected risk level '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestComputeCosineSimilarity(t *testing.T) {
	testCases := []struct {
		name     string
		vec1     []float64
		vec2     []float64
		expected float64
		delta    float64
	}{
		{"identical", []float64{1, 0, 0}, []float64{1, 0, 0}, 1.0, 0.001},
		{"opposite", []float64{1, 0, 0}, []float64{-1, 0, 0}, -1.0, 0.001},
		{"perpendicular", []float64{1, 0, 0}, []float64{0, 1, 0}, 0.0, 0.001},
		{"partial", []float64{1, 1}, []float64{1, 0}, 0.707, 0.01},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := computeCosineSimilarity(tc.vec1, tc.vec2)
			diff := math.Abs(result - tc.expected)
			if diff > tc.delta {
				t.Errorf("Expected %f (±%f), got %f", tc.expected, tc.delta, result)
			}
		})
	}
}

func computeCosineSimilarity(vec1, vec2 []float64) float64 {
	minLen := len(vec1)
	if len(vec2) < minLen {
		minLen = len(vec2)
	}

	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	for i := 0; i < minLen; i++ {
		dotProduct += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

func TestEnhancedCaptchaPrompt_Structure(t *testing.T) {
	now := time.Now()
	prompt := &EnhancedCaptchaPrompt{
		CaptchaID:     "test_captcha",
		Type:          "gpt_enhanced",
		Theme:         CaptchaThemeNature,
		Question:      "测试问题",
		Hint:          "测试提示",
		Options:       []string{"A", "B", "C"},
		Difficulty:    3,
		Scene:         CaptchaSceneLogin,
		Language:      "zh-CN",
		GeneratedAt:   now.Unix(),
		ExpiresAt:     now.Add(5 * time.Minute).Unix(),
		Metadata:      map[string]interface{}{"key": "value"},
	}

	if prompt.CaptchaID != "test_captcha" {
		t.Errorf("Expected CaptchaID 'test_captcha', got '%s'", prompt.CaptchaID)
	}

	if prompt.Type != "gpt_enhanced" {
		t.Errorf("Expected Type 'gpt_enhanced', got '%s'", prompt.Type)
	}

	if prompt.Theme != CaptchaThemeNature {
		t.Errorf("Expected Theme 'nature', got '%s'", prompt.Theme)
	}

	if len(prompt.Options) != 3 {
		t.Errorf("Expected 3 options, got %d", len(prompt.Options))
	}

	if prompt.Language != "zh-CN" {
		t.Errorf("Expected Language 'zh-CN', got '%s'", prompt.Language)
	}
}

func TestEnhancedRiskResult_Structure(t *testing.T) {
	result := &EnhancedRiskResult{
		RiskScore:        0.75,
		RiskLevel:        "high",
		Confidence:       0.9,
		FeatureScores:    map[string]float64{"feature1": 0.5},
		Recommendations:  []string{"recommendation1"},
		ModelVersion:    "v3.0-enhanced",
		ThreatIndicators: []ThreatIndicator{
			{Type: "test", Score: 0.8, Severity: "high", Evidence: []string{"evidence1"}},
		},
		AnomalyScore: 0.6,
	}

	if result.RiskScore != 0.75 {
		t.Errorf("Expected RiskScore 0.75, got %f", result.RiskScore)
	}

	if result.RiskLevel != "high" {
		t.Errorf("Expected RiskLevel 'high', got '%s'", result.RiskLevel)
	}

	if len(result.ThreatIndicators) != 1 {
		t.Errorf("Expected 1 threat indicator, got %d", len(result.ThreatIndicators))
	}
}

func TestThreatIndicator_Structure(t *testing.T) {
	indicator := ThreatIndicator{
		Type:     "suspicious_speed",
		Score:    0.9,
		Severity: "high",
		Evidence: []string{"evidence1", "evidence2"},
	}

	if indicator.Type != "suspicious_speed" {
		t.Errorf("Expected Type 'suspicious_speed', got '%s'", indicator.Type)
	}

	if indicator.Score != 0.9 {
		t.Errorf("Expected Score 0.9, got %f", indicator.Score)
	}

	if indicator.Severity != "high" {
		t.Errorf("Expected Severity 'high', got '%s'", indicator.Severity)
	}

	if len(indicator.Evidence) != 2 {
		t.Errorf("Expected 2 evidence items, got %d", len(indicator.Evidence))
	}
}

func TestEnhancedBehaviorPattern_Structure(t *testing.T) {
	pattern := &EnhancedBehaviorPattern{
		ID:              "pattern_1",
		Type:            "human",
		FeatureVector:   []float64{0.3, 0.4, 0.5},
		IsBot:           false,
		Confidence:      0.85,
		Occurrences:     10,
		LastSeen:        time.Now(),
		Metadata:        map[string]interface{}{"key": "value"},
		SimilarPatterns: []string{"pattern_2"},
		RiskLevel:       "low",
	}

	if pattern.ID != "pattern_1" {
		t.Errorf("Expected ID 'pattern_1', got '%s'", pattern.ID)
	}

	if pattern.IsBot {
		t.Error("IsBot should be false")
	}

	if pattern.Confidence != 0.85 {
		t.Errorf("Expected Confidence 0.85, got %f", pattern.Confidence)
	}

	if pattern.RiskLevel != "low" {
		t.Errorf("Expected RiskLevel 'low', got '%s'", pattern.RiskLevel)
	}
}

func BenchmarkEnhancedGPTCaptchaGenerator_GenerateCaptcha(b *testing.B) {
	generator := NewEnhancedGPTCaptchaGenerator()
	generator.Initialize(context.Background())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.GenerateCaptcha(context.Background(), CaptchaSceneLogin, 3, "zh-CN")
	}
}

func BenchmarkLSTMFeatureExtractor_ExtractFeatures(b *testing.B) {
	extractor := NewLSTMFeatureExtractor()

	traceData := &model.TraceData{
		PointCount:         50,
		TotalTime:          5000,
		TotalDistance:      100.5,
		AvgDistance:        2.01,
		AvgSpeed:           20.1,
		SpeedVariance:      0.5,
		MinSpeed:           10.0,
		MaxSpeed:           30.0,
		DirectionChanges:   5,
		AvgCurvature:       0.1,
		CurvatureVariance:  0.05,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor.ExtractFeatures(context.Background(), traceData)
	}
}

func BenchmarkEnhancedRiskAssessor_AssessRisk(b *testing.B) {
	assessor := NewEnhancedRiskAssessor()
	assessor.Initialize(context.Background())

	features := make([]float64, 20)
	for i := range features {
		features[i] = 0.5 + (rand.Float64() - 0.5)*0.3
	}

	traceData := &model.TraceData{
		PointCount:        50,
		TotalTime:        5000,
		DirectionChanges: 5,
		SpeedVariance:    0.3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		assessor.AssessRisk(context.Background(), features, nil, traceData)
	}
}

func BenchmarkEnhancedBehaviorLearningSystem_MatchPattern(b *testing.B) {
	system := NewEnhancedBehaviorLearningSystem()

	features := make([]float64, 10)
	for i := range features {
		features[i] = rand.Float64()
	}

	system.Initialize(context.Background())
	system.LearnFromExample(context.Background(), features, true, 0.9, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		system.MatchPattern(context.Background(), features)
	}
}
