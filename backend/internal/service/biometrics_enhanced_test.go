package service

import (
	"testing"
	"time"
)

func TestNewEnhancedBiometricsService(t *testing.T) {
	service := NewEnhancedBiometricsService()
	if service == nil {
		t.Error("NewEnhancedBiometricsService returned nil")
	}
	if service.profiles == nil {
		t.Error("service.profiles is nil")
	}
}

func TestRegisterEnhancedProfile(t *testing.T) {
	service := NewEnhancedBiometricsService()
	userID := "test_user_123"

	keyboardSample := &KeyboardSample{
		KeyEvents: []KeyEvent{
			{Key: "t", Type: "keydown", Timestamp: 1000, KeyCode: 84},
			{Key: "t", Type: "keyup", Timestamp: 1100, KeyCode: 84},
			{Key: "e", Type: "keydown", Timestamp: 1200, KeyCode: 69},
			{Key: "e", Type: "keyup", Timestamp: 1300, KeyCode: 69},
			{Key: "s", Type: "keydown", Timestamp: 1400, KeyCode: 83},
			{Key: "s", Type: "keyup", Timestamp: 1500, KeyCode: 83},
		},
		Timestamp: 1000,
	}

	profile, err := service.RegisterEnhancedProfile(
		userID,
		keyboardSample,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	if err != nil {
		t.Errorf("RegisterEnhancedProfile returned error: %v", err)
	}
	if profile == nil {
		t.Error("RegisterEnhancedProfile returned nil profile")
	}
	if profile.UserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, profile.UserID)
	}
	if profile.VerificationCount != 1 {
		t.Errorf("Expected VerificationCount 1, got %d", profile.VerificationCount)
	}
}

func TestRegisterEnhancedProfile_EmptyUserID(t *testing.T) {
	service := NewEnhancedBiometricsService()
	_, err := service.RegisterEnhancedProfile("", nil, nil, nil, nil, nil, nil)
	if err == nil {
		t.Error("Expected error for empty userID, got nil")
	}
}

func TestVerifyEnhanced_NoProfile(t *testing.T) {
	service := NewEnhancedBiometricsService()
	req := &EnhancedVerificationRequest{
		UserID: "nonexistent_user",
	}
	result, err := service.VerifyEnhanced(req)
	if err != nil {
		t.Errorf("VerifyEnhanced returned error: %v", err)
	}
	if result == nil {
		t.Error("VerifyEnhanced returned nil result")
	}
	if result.IsVerified {
		t.Error("Expected IsVerified false for non-existent user")
	}
}

func TestVerifyEnhanced_KeyboardOnly(t *testing.T) {
	service := NewEnhancedBiometricsService()
	userID := "test_keyboard_user"

	keyboardSample := &KeyboardSample{
		KeyEvents: []KeyEvent{
			{Key: "a", Type: "keydown", Timestamp: 1000, KeyCode: 65},
			{Key: "a", Type: "keyup", Timestamp: 1100, KeyCode: 65},
			{Key: "b", Type: "keydown", Timestamp: 1200, KeyCode: 66},
			{Key: "b", Type: "keyup", Timestamp: 1300, KeyCode: 66},
			{Key: "c", Type: "keydown", Timestamp: 1400, KeyCode: 67},
			{Key: "c", Type: "keyup", Timestamp: 1500, KeyCode: 67},
		},
		Timestamp: 1000,
	}

	_, err := service.RegisterEnhancedProfile(userID, keyboardSample, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to register profile: %v", err)
	}

	req := &EnhancedVerificationRequest{
		UserID:          userID,
		KeyboardSample:  keyboardSample,
	}

	result, err := service.VerifyEnhanced(req)
	if err != nil {
		t.Errorf("VerifyEnhanced returned error: %v", err)
	}
	if result == nil {
		t.Error("VerifyEnhanced returned nil result")
	}
	if result.OverallConfidence < 0.0 || result.OverallConfidence > 1.0 {
		t.Errorf("OverallConfidence out of range: %f", result.OverallConfidence)
	}
	if _, ok := result.ModalScores["keyboard"]; !ok {
		t.Error("Expected keyboard modal score")
	}
}

func TestExtractTypingPatternFeatures(t *testing.T) {
	service := NewEnhancedBiometricsService()

	typingSample := &TypingPatternSample{
		KeyEvents: []KeyEvent{
			{Key: "h", Type: "keydown", Timestamp: 1000, KeyCode: 72},
			{Key: "h", Type: "keyup", Timestamp: 1080, KeyCode: 72},
			{Key: "e", Type: "keydown", Timestamp: 1150, KeyCode: 69},
			{Key: "e", Type: "keyup", Timestamp: 1220, KeyCode: 69},
			{Key: "l", Type: "keydown", Timestamp: 1290, KeyCode: 76},
			{Key: "l", Type: "keyup", Timestamp: 1360, KeyCode: 76},
			{Key: "l", Type: "keydown", Timestamp: 1430, KeyCode: 76},
			{Key: "l", Type: "keyup", Timestamp: 1500, KeyCode: 76},
			{Key: "o", Type: "keydown", Timestamp: 1570, KeyCode: 79},
			{Key: "o", Type: "keyup", Timestamp: 1640, KeyCode: 79},
		},
		TextContent: "hello",
		Timestamp:   1000,
	}

	features := service.extractTypingPatternFeatures(typingSample)

	if features.AverageHoldTime <= 0 {
		t.Error("Expected AverageHoldTime > 0")
	}
	if features.AverageFlightTime <= 0 {
		t.Error("Expected AverageFlightTime > 0")
	}
	if features.ConsistencyScore < 0 || features.ConsistencyScore > 1 {
		t.Error("ConsistencyScore out of range")
	}
	if features.RhythmScore < 0 || features.RhythmScore > 1 {
		t.Error("RhythmScore out of range")
	}
}

func TestExtractFaceFeatures(t *testing.T) {
	service := NewEnhancedBiometricsService()
	faceSample := &FaceSample{
		Timestamp: time.Now().Unix(),
	}

	features := service.extractFaceFeatures(faceSample)

	if features.QualityScore < 0 || features.QualityScore > 1 {
		t.Error("QualityScore out of range")
	}
	if features.LivenessConfidence < 0 || features.LivenessConfidence > 1 {
		t.Error("LivenessConfidence out of range")
	}
	if features.MicroExpressionScores.Neutral < 0 || features.MicroExpressionScores.Neutral > 1 {
		t.Error("MicroExpression Neutral score out of range")
	}
	if len(features.LandmarkDistances) == 0 {
		t.Error("Expected LandmarkDistances to be populated")
	}
}

func TestExtractVoiceFeatures(t *testing.T) {
	service := NewEnhancedBiometricsService()
	voiceSample := &VoiceSample{
		Timestamp: time.Now().Unix(),
	}

	features := service.extractVoiceFeatures(voiceSample)

	if features.PitchFeatures.MeanPitch <= 0 {
		t.Error("Expected MeanPitch > 0")
	}
	if features.LivenessFeatures.OverallLivenessScore < 0 || features.LivenessFeatures.OverallLivenessScore > 1 {
		t.Error("OverallLivenessScore out of range")
	}
	if features.VoiceQualityScore < 0 || features.VoiceQualityScore > 1 {
		t.Error("VoiceQualityScore out of range")
	}
}

func TestAssessRisk(t *testing.T) {
	service := NewEnhancedBiometricsService()

	modalScores := map[string]float64{
		"keyboard": 0.85,
		"mouse":    0.75,
		"face":     0.90,
	}
	livenessChecks := map[string]bool{
		"face_liveness":  true,
		"voice_liveness": true,
	}

	risk := service.assessRisk(modalScores, livenessChecks, nil)

	if risk.RiskScore < 0 || risk.RiskScore > 1 {
		t.Error("RiskScore out of range")
	}
	if risk.RiskLevel != "low" && risk.RiskLevel != "medium" {
		t.Errorf("Unexpected RiskLevel: %s", risk.RiskLevel)
	}
}

func TestAssessRisk_LivenessFailure(t *testing.T) {
	service := NewEnhancedBiometricsService()

	modalScores := map[string]float64{
		"keyboard": 0.9,
	}
	livenessChecks := map[string]bool{
		"face_liveness": false,
	}

	risk := service.assessRisk(modalScores, livenessChecks, nil)

	if len(risk.Factors) == 0 {
		t.Error("Expected risk factors for liveness failure")
	}
	if risk.RiskLevel != "medium" && risk.RiskLevel != "high" {
		t.Errorf("Expected elevated risk level, got: %s", risk.RiskLevel)
	}
}

func TestFuseMultimodalScores(t *testing.T) {
	service := NewEnhancedBiometricsService()

	scores := map[string]float64{
		"keyboard":       0.9,
		"mouse":          0.85,
		"face":           0.95,
		"voice":          0.88,
		"gesture":        0.7,
		"typing_pattern": 0.92,
	}

	weights := MultimodalWeights{
		KeyboardWeight: 0.25,
		MouseWeight:    0.2,
		FaceWeight:     0.2,
		VoiceWeight:    0.2,
		GestureWeight:  0.05,
		TypingWeight:   0.1,
	}

	totalWeight := 0.25 + 0.2 + 0.2 + 0.2 + 0.05 + 0.1
	result := service.fuseMultimodalScores(scores, weights, totalWeight)

	if result < 0.0 || result > 1.0 {
		t.Errorf("Fused score out of range: %f", result)
	}
}

func TestMeanEnhanced(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}
	result := meanEnhanced(values)
	expected := 3.0
	if result != expected {
		t.Errorf("Expected mean %f, got %f", expected, result)
	}
}

func TestMeanEnhanced_Empty(t *testing.T) {
	result := meanEnhanced([]float64{})
	if result != 0 {
		t.Errorf("Expected 0 for empty slice, got %f", result)
	}
}

func TestStdDevEnhanced(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}
	result := stdDevEnhanced(values)
	if result < 0 {
		t.Errorf("StdDev cannot be negative: %f", result)
	}
}

func TestStdDevEnhanced_TooFew(t *testing.T) {
	result := stdDevEnhanced([]float64{1})
	if result != 0 {
		t.Errorf("Expected 0 for single value, got %f", result)
	}
}

func TestCalculateWPM(t *testing.T) {
	flightTimes := []float64{100, 200, 300, 400, 500, 600}
	keyCount := 7
	result := calculateWPM(flightTimes, keyCount)
	if result < 0 {
		t.Errorf("WPM cannot be negative: %f", result)
	}
}

func TestCalculateConsistency(t *testing.T) {
	holdTimes := []float64{80, 75, 82, 78, 81}
	flightTimes := []float64{150, 145, 155, 148}
	result := calculateConsistency(holdTimes, flightTimes)
	if result < 0 || result > 1 {
		t.Errorf("Consistency score out of range: %f", result)
	}
}

func TestCalculateRhythm(t *testing.T) {
	flightTimes := []float64{100, 110, 105, 115, 108}
	result := calculateRhythm(flightTimes)
	if result < 0 || result > 1 {
		t.Errorf("Rhythm score out of range: %f", result)
	}
}

func TestSerializeDeserializeEnhancedProfile(t *testing.T) {
	service := NewEnhancedBiometricsService()
	userID := "serialize_test_user"

	profile, err := service.RegisterEnhancedProfile(userID, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to register profile: %v", err)
	}

	data, err := profile.SerializeEnhancedProfile()
	if err != nil {
		t.Errorf("Serialization failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("Serialized data is empty")
	}

	deserialized, err := service.DeserializeEnhancedProfile(data)
	if err != nil {
		t.Errorf("Deserialization failed: %v", err)
	}
	if deserialized.UserID != userID {
		t.Errorf("UserID mismatch after deserialize: expected %s, got %s", userID, deserialized.UserID)
	}
}

func TestVerifyEnhanced_Multimodal(t *testing.T) {
	service := NewEnhancedBiometricsService()
	userID := "multimodal_test_user"

	keyboardSample := &KeyboardSample{
		KeyEvents: []KeyEvent{
			{Key: "t", Type: "keydown", Timestamp: 1000, KeyCode: 84},
			{Key: "t", Type: "keyup", Timestamp: 1080, KeyCode: 84},
			{Key: "e", Type: "keydown", Timestamp: 1150, KeyCode: 69},
			{Key: "e", Type: "keyup", Timestamp: 1230, KeyCode: 69},
		},
		Timestamp: 1000,
	}

	mouseSample := &MouseSample{
		MouseEvents: []MouseEvent{
			{Type: "mousemove", X: 100, Y: 100, Timestamp: 1000},
			{Type: "mousemove", X: 120, Y: 120, Timestamp: 1100},
			{Type: "mousemove", X: 140, Y: 140, Timestamp: 1200},
			{Type: "mousemove", X: 160, Y: 160, Timestamp: 1300},
			{Type: "click", X: 180, Y: 180, Timestamp: 1400, Button: 1},
		},
		Timestamp: 1000,
	}

	faceSample := &FaceSample{
		Timestamp: time.Now().Unix(),
	}

	voiceSample := &VoiceSample{
		Timestamp: time.Now().Unix(),
	}

	typingSample := &TypingPatternSample{
		KeyEvents: keyboardSample.KeyEvents,
		Timestamp: 1000,
	}

	_, err := service.RegisterEnhancedProfile(userID, keyboardSample, mouseSample, faceSample, voiceSample, nil, typingSample)
	if err != nil {
		t.Fatalf("Failed to register profile: %v", err)
	}

	req := &EnhancedVerificationRequest{
		UserID:              userID,
		KeyboardSample:      keyboardSample,
		MouseSample:         mouseSample,
		FaceSample:          faceSample,
		VoiceSample:         voiceSample,
		TypingPatternSample: typingSample,
	}

	result, err := service.VerifyEnhanced(req)
	if err != nil {
		t.Errorf("VerifyEnhanced returned error: %v", err)
	}
	if result == nil {
		t.Error("VerifyEnhanced returned nil result")
	}

	expectedModals := []string{"keyboard", "mouse", "face", "voice", "gesture", "typing_pattern"}
	for _, modal := range expectedModals {
		if _, ok := result.ModalScores[modal]; !ok {
			t.Errorf("Expected modal score for %s", modal)
		}
	}

	if result.RiskAssessment == nil {
		t.Error("Expected risk assessment")
	}
}
