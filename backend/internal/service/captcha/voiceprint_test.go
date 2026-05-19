package captcha

import (
	"context"
	"testing"
)

func TestVoiceprintGeneratorService_Generate(t *testing.T) {
	gen := NewVoiceprintGeneratorService(nil, nil)

	req := &VoiceprintCaptchaRequest{
		PatternType: "sequence",
		Complexity: 3,
		ClientIP:   "127.0.0.1",
		UserAgent:  "test-agent",
	}

	ctx := context.Background()
	result, err := gen.Generate(ctx, req)

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.SessionID == "" {
		t.Error("SessionID is empty")
	}

	if result.Pattern == nil {
		t.Error("Pattern is nil")
	}

	if result.Pattern.TargetPhrase == "" {
		t.Error("TargetPhrase is empty")
	}

	if len(result.Pattern.Frequencies) == 0 {
		t.Error("Frequencies array is empty")
	}

	if result.AudioData == "" {
		t.Error("AudioData is empty")
	}

	if result.ExpiresIn == 0 {
		t.Error("ExpiresIn is 0")
	}

	if result.ExpiresAt == 0 {
		t.Error("ExpiresAt is 0")
	}
}

func TestVoiceprintGeneratorService_GenerateWithDifferentComplexity(t *testing.T) {
	gen := NewVoiceprintGeneratorService(nil, nil)

	testCases := []struct {
		complexity int
		expectedLen int
	}{
		{1, 1},
		{3, 3},
		{5, 5},
	}

	for _, tc := range testCases {
		req := &VoiceprintCaptchaRequest{
			Complexity: tc.complexity,
		}

		result, err := gen.Generate(context.Background(), req)
		if err != nil {
			t.Errorf("Complexity %d: Generate failed: %v", tc.complexity, err)
			continue
		}

		if len(result.Pattern.Frequencies) != tc.expectedLen {
			t.Errorf("Complexity %d: expected %d frequencies, got %d",
				tc.complexity, tc.expectedLen, len(result.Pattern.Frequencies))
		}
	}
}

func TestVoiceprintVerifierService_Verify(t *testing.T) {
	gen := NewVoiceprintGeneratorService(nil, nil)
	ver := NewVoiceprintVerifierService(nil, nil)

	session, err := gen.Generate(context.Background(), &VoiceprintCaptchaRequest{
		Complexity: 3,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	verifyReq := &VoiceprintVerifyRequest{
		SessionID: session.SessionID,
	}

	result, err := ver.Verify(context.Background(), verifyReq)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.SimilarityScore < 0 || result.SimilarityScore > 1 {
		t.Errorf("SimilarityScore out of range: %f", result.SimilarityScore)
	}
}

func TestVoiceprintVerifierService_VerifyWithFeatures(t *testing.T) {
	gen := NewVoiceprintGeneratorService(nil, nil)
	ver := NewVoiceprintVerifierService(nil, nil)

	session, err := gen.Generate(context.Background(), &VoiceprintCaptchaRequest{
		Complexity: 3,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	features := &VoiceFeatures{
		MFCC:             []float64{1.0, 2.0, 3.0},
		SpectralFlux:     []float64{0.5, 0.6},
		Formants:         []float64{100.0, 200.0, 300.0},
		FundamentalFreq:  150.0,
		Energy:           0.5,
	}

	verifyReq := &VoiceprintVerifyRequest{
		SessionID: session.SessionID,
		Features:  features,
	}

	result, err := ver.Verify(context.Background(), verifyReq)
	if err != nil {
		t.Fatalf("Verify with features failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.MatchLevel != "" && result.SimilarityScore >= 0 && result.SimilarityScore <= 1 {
		t.Logf("Verification result: score=%f, level=%s", result.SimilarityScore, result.MatchLevel)
	}
}

func TestVoiceprintVerifierService_CosineSimilarity(t *testing.T) {
	ver := NewVoiceprintVerifierService(nil, nil)

	testCases := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1.0, 0.0},
			b:        []float64{1.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1.0, 0.0},
			b:        []float64{0.0, 1.0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1.0, 0.0},
			b:        []float64{-1.0, 0.0},
			expected: -1.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ver.cosineSimilarity(tc.a, tc.b)
			if result < tc.expected-0.01 || result > tc.expected+0.01 {
				t.Errorf("cosineSimilarity(%v, %v) = %f, expected %f",
					tc.a, tc.b, result, tc.expected)
			}
		})
	}
}

func TestVoiceprintVerifierService_ExtractFeatures(t *testing.T) {
	ver := NewVoiceprintVerifierService(nil, nil)

	audioData := make([]byte, 1024)
	for i := 0; i < len(audioData); i++ {
		audioData[i] = byte(i % 256)
	}

	features := ver.ExtractFeatures(audioData)

	if features == nil {
		t.Fatal("Features is nil")
	}

	if len(features.MFCC) == 0 {
		t.Error("MFCC is empty")
	}

	if len(features.SpectralFlux) == 0 {
		t.Error("SpectralFlux is empty")
	}

	if len(features.Formants) == 0 {
		t.Error("Formants is empty")
	}
}

func TestVoiceprintVerifierService_GetMatchLevel(t *testing.T) {
	ver := NewVoiceprintVerifierService(nil, nil)

	testCases := []struct {
		score     float64
		expected  string
	}{
		{0.9, VoiceprintMatchLevelHigh},
		{0.85, VoiceprintMatchLevelHigh},
		{0.8, VoiceprintMatchLevelMedium},
		{0.7, VoiceprintMatchLevelMedium},
		{0.6, VoiceprintMatchLevelLow},
		{0.3, VoiceprintMatchLevelLow},
	}

	for _, tc := range testCases {
		result := ver.getMatchLevel(tc.score)
		if result != tc.expected {
			t.Errorf("getMatchLevel(%f) = %s, expected %s",
				tc.score, result, tc.expected)
		}
	}
}

func TestVoiceprintPattern_Generation(t *testing.T) {
	gen := NewVoiceprintGeneratorService(nil, nil)

	pattern := gen.generatePattern(3)

	if pattern == nil {
		t.Fatal("Pattern is nil")
	}

	if pattern.TargetPhrase == "" {
		t.Error("TargetPhrase is empty")
	}

	if len(pattern.Frequencies) != 3 {
		t.Errorf("Expected 3 frequencies, got %d", len(pattern.Frequencies))
	}

	if len(pattern.Durations) != 3 {
		t.Errorf("Expected 3 durations, got %d", len(pattern.Durations))
	}

	if len(pattern.Amplitudes) != 3 {
		t.Errorf("Expected 3 amplitudes, got %d", len(pattern.Amplitudes))
	}
}

func TestVoiceprintVerifierService_ExpiredSession(t *testing.T) {
	ver := NewVoiceprintVerifierService(nil, nil)

	verifyReq := &VoiceprintVerifyRequest{
		SessionID: "non-existent-session",
	}

	result, err := ver.Verify(context.Background(), verifyReq)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if result.Success {
		t.Error("Expected verification to fail for non-existent session")
	}

	if result.Message != "Session not found" {
		t.Errorf("Expected 'Session not found', got '%s'", result.Message)
	}
}

func TestVoiceprintVerifierService_ExtractFeaturesEmpty(t *testing.T) {
	ver := NewVoiceprintVerifierService(nil, nil)

	features := ver.ExtractFeatures(nil)

	if features != nil {
		t.Error("Expected nil features for empty audio data")
	}

	features = ver.ExtractFeatures([]byte{})

	if features != nil {
		t.Error("Expected nil features for empty audio data")
	}
}

func TestVoiceprintGeneratorService_WAVHeader(t *testing.T) {
	header := createVoiceprintWAVHeader()

	if len(header) != 44 {
		t.Errorf("Expected header length 44, got %d", len(header))
	}

	if string(header[0:4]) != "RIFF" {
		t.Error("Expected RIFF marker")
	}

	if string(header[8:12]) != "WAVE" {
		t.Error("Expected WAVE marker")
	}

	if string(header[12:16]) != "fmt " {
		t.Error("Expected fmt marker")
	}

	if string(header[36:40]) != "data" {
		t.Error("Expected data marker")
	}
}

func TestVoiceprintGeneratorService_VoiceWave(t *testing.T) {
	header := createVoiceprintWAVHeader()
	
	wave := generateVoiceprintWave(440.0, 0.5, 0.5, 0.5, 44100)
	
	if len(wave) == 0 {
		t.Error("Generated wave is empty")
	}

	wav := append(header, wave...)
	
	if len(wav) <= 44 {
		t.Error("Generated WAV data is too short")
	}
}

func TestVoiceprintGeneratorService_DefaultValues(t *testing.T) {
	gen := NewVoiceprintGeneratorService(nil, nil)

	req := &VoiceprintCaptchaRequest{}

	ctx := context.Background()
	result, err := gen.Generate(ctx, req)

	if err != nil {
		t.Fatalf("Generate with default values failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.Pattern == nil {
		t.Fatal("Pattern is nil")
	}

	if len(result.Pattern.Frequencies) != 3 {
		t.Errorf("Expected default complexity 3, got %d", len(result.Pattern.Frequencies))
	}
}

func TestVoiceprintVerifierService_FundamentalFreq(t *testing.T) {
	ver := NewVoiceprintVerifierService(nil, nil)

	sampleRate := 44100
	samples := make([]float64, 2048)

	freq := 200.0
	period := float64(sampleRate) / freq

	for i := 0; i < len(samples); i++ {
		samples[i] = 0.5 * (float64(i%int(period)) / period)
	}

	audioData := make([]byte, len(samples)*2)
	for i, s := range samples {
		sample := int16(s * 32767)
		audioData[i*2] = byte(sample & 0xff)
		audioData[i*2+1] = byte((sample >> 8) & 0xff)
	}

	features := ver.ExtractFeatures(audioData)

	if features != nil && features.FundamentalFreq > 0 {
		t.Logf("Extracted fundamental frequency: %f", features.FundamentalFreq)
	}
}

func TestVoiceprintVerifierService_Energy(t *testing.T) {
	ver := NewVoiceprintVerifierService(nil, nil)

	audioData := make([]byte, 1000)
	for i := 0; i < len(audioData)/2; i++ {
		sample := int16(10000)
		audioData[i*2] = byte(sample & 0xff)
		audioData[i*2+1] = byte((sample >> 8) & 0xff)
	}

	energy := ver.calculateEnergy(audioData)

	if energy <= 0 {
		t.Error("Energy should be positive for non-zero audio")
	}
}
