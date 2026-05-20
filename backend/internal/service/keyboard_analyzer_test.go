package service

import (
	"testing"
	"time"
)

func TestNewKeyboardAnalyzer(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()
	if analyzer == nil {
		t.Error("NewKeyboardAnalyzer returned nil")
	}
}

func TestKeyboardAnalyzer_AnalyzeKeystrokes(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{
			Key:           'a',
			PressTime:     time.Now(),
			ReleaseTime:   time.Now().Add(100 * time.Millisecond),
			IsShift:       false,
			IsCapsLock:    false,
		},
		{
			Key:           'b',
			PressTime:     time.Now().Add(200 * time.Millisecond),
			ReleaseTime:   time.Now().Add(300 * time.Millisecond),
			IsShift:       false,
			IsCapsLock:    false,
		},
	}

	result := analyzer.AnalyzeKeystrokes(samples)
	if result == nil {
		t.Error("AnalyzeKeystrokes returned nil")
	}
}

func TestKeyboardAnalyzer_AnalyzeKeystrokesEmpty(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	result := analyzer.AnalyzeKeystrokes([]KeyboardSample{})
	if result == nil {
		t.Error("AnalyzeKeystrokes should return result for empty samples")
	}
}

func TestKeyboardAnalyzer_AnalyzeKeystrokesSingleSample(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	sample := KeyboardSample{
		Key:         'a',
		PressTime:   time.Now(),
		ReleaseTime: time.Now().Add(50 * time.Millisecond),
	}

	result := analyzer.AnalyzeKeystrokes([]KeyboardSample{sample})
	if result == nil {
		t.Error("AnalyzeKeystrokes should return result for single sample")
	}
}

func TestKeyboardAnalyzer_CalculateTypingSpeed(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{
			Key:         'a',
			PressTime:   time.Now(),
			ReleaseTime: time.Now().Add(50 * time.Millisecond),
		},
		{
			Key:         'b',
			PressTime:   time.Now().Add(100 * time.Millisecond),
			ReleaseTime: time.Now().Add(150 * time.Millisecond),
		},
		{
			Key:         'c',
			PressTime:   time.Now().Add(200 * time.Millisecond),
			ReleaseTime: time.Now().Add(250 * time.Millisecond),
		},
	}

	speed := analyzer.CalculateTypingSpeed(samples)
	if speed < 0 {
		t.Error("Typing speed should not be negative")
	}
}

func TestKeyboardAnalyzer_CalculateDwellTime(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	sample := KeyboardSample{
		Key:         'a',
		PressTime:   time.Now(),
		ReleaseTime: time.Now().Add(100 * time.Millisecond),
	}

	dwellTime := analyzer.CalculateDwellTime(sample)
	if dwellTime < 0 {
		t.Error("Dwell time should not be negative")
	}
}

func TestKeyboardAnalyzer_CalculateFlightTime(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	sample1 := KeyboardSample{
		Key:         'a',
		PressTime:   time.Now(),
		ReleaseTime: time.Now().Add(50 * time.Millisecond),
	}
	sample2 := KeyboardSample{
		Key:         'b',
		PressTime:   time.Now().Add(100 * time.Millisecond),
		ReleaseTime: time.Now().Add(150 * time.Millisecond),
	}

	flightTime := analyzer.CalculateFlightTime(sample1, sample2)
	if flightTime < 0 {
		t.Error("Flight time should not be negative")
	}
}

func TestKeyboardAnalyzer_CalculateInterKeyLatency(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	sample1 := KeyboardSample{
		Key:         'a',
		PressTime:   time.Now(),
		ReleaseTime: time.Now().Add(50 * time.Millisecond),
	}
	sample2 := KeyboardSample{
		Key:         'b',
		PressTime:   time.Now().Add(100 * time.Millisecond),
		ReleaseTime: time.Now().Add(150 * time.Millisecond),
	}

	latency := analyzer.CalculateInterKeyLatency(sample1, sample2)
	if latency < 0 {
		t.Error("Inter-key latency should not be negative")
	}
}

func TestKeyboardAnalyzer_DetectAutomation(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := make([]KeyboardSample, 20)
	baseTime := time.Now()
	for i := 0; i < 20; i++ {
		samples[i] = KeyboardSample{
			Key:         rune('a' + i%26),
			PressTime:   baseTime.Add(time.Duration(i*50) * time.Millisecond),
			ReleaseTime: baseTime.Add(time.Duration(i*50+20) * time.Millisecond),
		}
	}

	result := analyzer.DetectAutomation(samples)
	if result == nil {
		t.Error("DetectAutomation returned nil")
	}
}

func TestKeyboardAnalyzer_ExtractPattern(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{Key: 'a', PressTime: time.Now()},
		{Key: 'b', PressTime: time.Now().Add(100 * time.Millisecond)},
		{Key: 'c', PressTime: time.Now().Add(200 * time.Millisecond)},
	}

	pattern := analyzer.ExtractPattern(samples)
	if pattern == nil {
		t.Error("ExtractPattern returned nil")
	}
}

func TestKeyboardAnalyzer_MatchPattern(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	pattern := []float64{100.0, 150.0, 120.0}
	samples := []KeyboardSample{
		{Key: 'a', PressTime: time.Now()},
		{Key: 'b', PressTime: time.Now().Add(100 * time.Millisecond)},
		{Key: 'c', PressTime: time.Now().Add(250 * time.Millisecond)},
	}

	match := analyzer.MatchPattern(pattern, samples)
	if match < 0.0 || match > 1.0 {
		t.Error("Pattern match score should be between 0 and 1")
	}
}

func TestKeyboardAnalyzer_CalculateConsistency(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{Key: 'a', PressTime: time.Now(), ReleaseTime: time.Now().Add(50 * time.Millisecond)},
		{Key: 'a', PressTime: time.Now().Add(1000 * time.Millisecond), ReleaseTime: time.Now().Add(1050 * time.Millisecond)},
		{Key: 'a', PressTime: time.Now().Add(2000 * time.Millisecond), ReleaseTime: time.Now().Add(2050 * time.Millisecond)},
	}

	consistency := analyzer.CalculateConsistency(samples)
	if consistency < 0.0 || consistency > 1.0 {
		t.Error("Consistency score should be between 0 and 1")
	}
}

func TestKeyboardAnalyzer_GetErrorRate(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{Key: 'a', PressTime: time.Now(), ReleaseTime: time.Now().Add(50 * time.Millisecond)},
		{Key: 'b', PressTime: time.Now().Add(100 * time.Millisecond), ReleaseTime: time.Now().Add(150 * time.Millisecond)},
		{Key: 'a', PressTime: time.Now().Add(300 * time.Millisecond), ReleaseTime: time.Now().Add(350 * time.Millisecond)},
	}

	errorRate := analyzer.GetErrorRate(samples)
	if errorRate < 0.0 || errorRate > 1.0 {
		t.Error("Error rate should be between 0 and 1")
	}
}

func TestKeyboardAnalyzer_GetStatistics(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	stats := analyzer.GetStatistics()
	if stats == nil {
		t.Error("GetStatistics returned nil")
	}
}

func TestKeyboardAnalyzer_GetFeatureVector(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{Key: 'a', PressTime: time.Now(), ReleaseTime: time.Now().Add(50 * time.Millisecond)},
		{Key: 'b', PressTime: time.Now().Add(100 * time.Millisecond), ReleaseTime: time.Now().Add(150 * time.Millisecond)},
	}

	vector := analyzer.GetFeatureVector(samples)
	if vector == nil {
		t.Error("GetFeatureVector returned nil")
	}
	if len(vector) == 0 {
		t.Error("Feature vector should not be empty")
	}
}

func TestKeyboardAnalyzer_CompareBiometrics(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples1 := []KeyboardSample{
		{Key: 'a', PressTime: time.Now(), ReleaseTime: time.Now().Add(50 * time.Millisecond)},
		{Key: 'b', PressTime: time.Now().Add(100 * time.Millisecond), ReleaseTime: time.Now().Add(150 * time.Millisecond)},
	}

	samples2 := []KeyboardSample{
		{Key: 'a', PressTime: time.Now().Add(500 * time.Millisecond), ReleaseTime: time.Now().Add(550 * time.Millisecond)},
		{Key: 'b', PressTime: time.Now().Add(600 * time.Millisecond), ReleaseTime: time.Now().Add(650 * time.Millisecond)},
	}

	similarity := analyzer.CompareBiometrics(samples1, samples2)
	if similarity < 0.0 || similarity > 1.0 {
		t.Error("Similarity score should be between 0 and 1")
	}
}

func TestKeyboardAnalyzer_DetectUnusualBehavior(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{Key: 'a', PressTime: time.Now(), ReleaseTime: time.Now().Add(2000 * time.Millisecond)},
		{Key: 'b', PressTime: time.Now().Add(3000 * time.Millisecond), ReleaseTime: time.Now().Add(3500 * time.Millisecond)},
	}

	result := analyzer.DetectUnusualBehavior(samples)
	if result == nil {
		t.Error("DetectUnusualBehavior returned nil")
	}
}

func TestKeyboardAnalyzer_UpdateConfig(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	analyzer.UpdateConfig(&KeyboardAnalyzerConfig{
		EnablePatternMatching: true,
		PatternThreshold:     0.85,
	})

	if analyzer.config.PatternThreshold != 0.85 {
		t.Error("Config not updated correctly")
	}
}

func TestKeyboardAnalyzer_GetConfig(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	config := analyzer.GetConfig()
	if config == nil {
		t.Error("GetConfig returned nil")
	}
}

func TestKeyboardAnalyzer_LearnPattern(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{Key: 'h', PressTime: time.Now(), ReleaseTime: time.Now().Add(50 * time.Millisecond)},
		{Key: 'e', PressTime: time.Now().Add(100 * time.Millisecond), ReleaseTime: time.Now().Add(150 * time.Millisecond)},
		{Key: 'l', PressTime: time.Now().Add(200 * time.Millisecond), ReleaseTime: time.Now().Add(250 * time.Millisecond)},
		{Key: 'l', PressTime: time.Now().Add(300 * time.Millisecond), ReleaseTime: time.Now().Add(350 * time.Millisecond)},
		{Key: 'o', PressTime: time.Now().Add(400 * time.Millisecond), ReleaseTime: time.Now().Add(450 * time.Millisecond)},
	}

	err := analyzer.LearnPattern("hello", samples)
	if err != nil {
		t.Errorf("LearnPattern failed: %v", err)
	}
}

func TestKeyboardAnalyzer_GetLearnedPatterns(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	patterns := analyzer.GetLearnedPatterns()
	if patterns == nil {
		t.Error("GetLearnedPatterns returned nil")
	}
}

func TestKeyboardAnalyzer_ClearLearnedPatterns(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	analyzer.ClearLearnedPatterns()
}

func TestKeyboardAnalyzer_CalculateRhythmScore(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{Key: 'a', PressTime: time.Now(), ReleaseTime: time.Now().Add(50 * time.Millisecond)},
		{Key: 'b', PressTime: time.Now().Add(100 * time.Millisecond), ReleaseTime: time.Now().Add(150 * time.Millisecond)},
		{Key: 'c', PressTime: time.Now().Add(200 * time.Millisecond), ReleaseTime: time.Now().Add(250 * time.Millisecond)},
	}

	score := analyzer.CalculateRhythmScore(samples)
	if score < 0.0 || score > 1.0 {
		t.Error("Rhythm score should be between 0 and 1")
	}
}

func TestKeyboardAnalyzer_DetectCopyPaste(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{Key: 'c', PressTime: time.Now(), ReleaseTime: time.Now().Add(20 * time.Millisecond)},
		{Key: 't', PressTime: time.Now().Add(25 * time.Millisecond), ReleaseTime: time.Now().Add(45 * time.Millisecond)},
		{Key: 'r', PressTime: time.Now().Add(30 * time.Millisecond), ReleaseTime: time.Now().Add(50 * time.Millisecond)},
		{Key: 'l', PressTime: time.Now().Add(35 * time.Millisecond), ReleaseTime: time.Now().Add(55 * time.Millisecond)},
	}

	result := analyzer.DetectCopyPaste(samples)
	if result == nil {
		t.Error("DetectCopyPaste returned nil")
	}
}

func TestKeyboardAnalyzer_GetPressureAnalysis(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{Key: 'a', PressTime: time.Now(), ReleaseTime: time.Now().Add(50 * time.Millisecond), Pressure: 0.5},
		{Key: 'b', PressTime: time.Now().Add(100 * time.Millisecond), ReleaseTime: time.Now().Add(150 * time.Millisecond), Pressure: 0.8},
	}

	analysis := analyzer.GetPressureAnalysis(samples)
	if analysis == nil {
		t.Error("GetPressureAnalysis returned nil")
	}
}

func TestKeyboardAnalyzer_GetTimingDistribution(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{Key: 'a', PressTime: time.Now(), ReleaseTime: time.Now().Add(50 * time.Millisecond)},
		{Key: 'b', PressTime: time.Now().Add(100 * time.Millisecond), ReleaseTime: time.Now().Add(150 * time.Millisecond)},
		{Key: 'c', PressTime: time.Now().Add(200 * time.Millisecond), ReleaseTime: time.Now().Add(250 * time.Millisecond)},
	}

	distribution := analyzer.GetTimingDistribution(samples)
	if distribution == nil {
		t.Error("GetTimingDistribution returned nil")
	}
}

func TestKeyboardAnalyzer_ClassifyInputSource(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{Key: 'a', PressTime: time.Now(), ReleaseTime: time.Now().Add(50 * time.Millisecond)},
		{Key: 'b', PressTime: time.Now().Add(100 * time.Millisecond), ReleaseTime: time.Now().Add(150 * time.Millisecond)},
	}

	source := analyzer.ClassifyInputSource(samples)
	if source == "" {
		t.Error("ClassifyInputSource should return non-empty string")
	}
}

func TestKeyboardAnalyzer_GetBiometricSignature(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{Key: 'a', PressTime: time.Now(), ReleaseTime: time.Now().Add(50 * time.Millisecond)},
		{Key: 'b', PressTime: time.Now().Add(100 * time.Millisecond), ReleaseTime: time.Now().Add(150 * time.Millisecond)},
	}

	signature := analyzer.GetBiometricSignature(samples)
	if signature == "" {
		t.Error("GetBiometricSignature should return non-empty string")
	}
}

func TestKeyboardAnalyzer_VerifyBiometricSignature(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{Key: 'a', PressTime: time.Now(), ReleaseTime: time.Now().Add(50 * time.Millisecond)},
		{Key: 'b', PressTime: time.Now().Add(100 * time.Millisecond), ReleaseTime: time.Now().Add(150 * time.Millisecond)},
	}

	signature := analyzer.GetBiometricSignature(samples)

	result := analyzer.VerifyBiometricSignature(signature, samples)
	if result < 0.0 || result > 1.0 {
		t.Error("Verification score should be between 0 and 1")
	}
}

func TestKeyboardSample_Fields(t *testing.T) {
	sample := KeyboardSample{
		Key:         'a',
		PressTime:   time.Now(),
		ReleaseTime: time.Now().Add(50 * time.Millisecond),
		IsShift:     false,
		IsCapsLock:  false,
		IsCtrl:      false,
		IsAlt:       false,
		Pressure:    0.7,
	}

	if sample.Key != 'a' {
		t.Errorf("Key should be 'a', got %c", sample.Key)
	}
	if sample.Pressure != 0.7 {
		t.Errorf("Pressure should be 0.7, got %f", sample.Pressure)
	}
}

func TestKeystrokeAnalysis_Fields(t *testing.T) {
	analysis := &KeystrokeAnalysis{
		TypingSpeed:    60.0,
		AvgDwellTime:   80.0,
		AvgFlightTime:  120.0,
		Consistency:    0.85,
		ErrorRate:     0.02,
		IsAutomated:   false,
		Confidence:    0.90,
	}

	if analysis.TypingSpeed != 60.0 {
		t.Errorf("TypingSpeed should be 60.0, got %f", analysis.TypingSpeed)
	}
	if analysis.Consistency != 0.85 {
		t.Errorf("Consistency should be 0.85, got %f", analysis.Consistency)
	}
}

func TestKeyboardAnalyzerConfig_Fields(t *testing.T) {
	config := &KeyboardAnalyzerConfig{
		EnablePatternMatching: true,
		PatternThreshold:     0.85,
		MinSamplesRequired:   10,
		LearningEnabled:     true,
	}

	if !config.EnablePatternMatching {
		t.Error("EnablePatternMatching should be true")
	}
	if config.PatternThreshold != 0.85 {
		t.Errorf("PatternThreshold should be 0.85, got %f", config.PatternThreshold)
	}
}

func TestKeyboardAnalyzer_AnalyzeSequences(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	sequences := [][]KeyboardSample{
		{
			{Key: 'a', PressTime: time.Now(), ReleaseTime: time.Now().Add(50 * time.Millisecond)},
			{Key: 'b', PressTime: time.Now().Add(100 * time.Millisecond), ReleaseTime: time.Now().Add(150 * time.Millisecond)},
		},
		{
			{Key: 'c', PressTime: time.Now().Add(1000 * time.Millisecond), ReleaseTime: time.Now().Add(1050 * time.Millisecond)},
			{Key: 'd', PressTime: time.Now().Add(1100 * time.Millisecond), ReleaseTime: time.Now().Add(1150 * time.Millisecond)},
		},
	}

	result := analyzer.AnalyzeSequences(sequences)
	if result == nil {
		t.Error("AnalyzeSequences returned nil")
	}
}

func TestKeyboardAnalyzer_CalculateEntropy(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{Key: 'a', PressTime: time.Now(), ReleaseTime: time.Now().Add(50 * time.Millisecond)},
		{Key: 'b', PressTime: time.Now().Add(100 * time.Millisecond), ReleaseTime: time.Now().Add(150 * time.Millisecond)},
		{Key: 'c', PressTime: time.Now().Add(200 * time.Millisecond), ReleaseTime: time.Now().Add(250 * time.Millisecond)},
	}

	entropy := analyzer.CalculateEntropy(samples)
	if entropy < 0 {
		t.Error("Entropy should not be negative")
	}
}

func TestKeyboardAnalyzer_GetKeyTransitionMatrix(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	samples := []KeyboardSample{
		{Key: 'a', PressTime: time.Now(), ReleaseTime: time.Now().Add(50 * time.Millisecond)},
		{Key: 'b', PressTime: time.Now().Add(100 * time.Millisecond), ReleaseTime: time.Now().Add(150 * time.Millisecond)},
		{Key: 'a', PressTime: time.Now().Add(200 * time.Millisecond), ReleaseTime: time.Now().Add(250 * time.Millisecond)},
	}

	matrix := analyzer.GetKeyTransitionMatrix(samples)
	if matrix == nil {
		t.Error("GetKeyTransitionMatrix returned nil")
	}
}
