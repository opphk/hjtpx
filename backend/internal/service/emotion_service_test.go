package service

import (
	"testing"
)

func TestEmotionServiceGetConfig(t *testing.T) {
	service := NewEmotionService()

	config := service.GetConfig()
	if config == nil {
		t.Error("GetConfig() returned nil")
	}

	if config.EmotionThreshold != 0.75 {
		t.Errorf("GetConfig() EmotionThreshold = %f, want 0.75", config.EmotionThreshold)
	}
}

func TestEmotionServiceAnalyzeFace(t *testing.T) {
	service := NewEmotionService()

	frame := []byte{0x00, 0x01, 0x02, 0x03}

	analysis, err := service.AnalyzeFace(frame, 1)
	if err != nil {
		t.Errorf("AnalyzeFace() error = %v", err)
		return
	}

	if analysis == nil {
		t.Error("AnalyzeFace() returned nil")
		return
	}

	if analysis.FrameID != 1 {
		t.Errorf("AnalyzeFace() FrameID = %d, want 1", analysis.FrameID)
	}

	if analysis.Emotion == "" {
		t.Error("AnalyzeFace() returned empty Emotion")
	}

	if analysis.Confidence <= 0 || analysis.Confidence > 1 {
		t.Error("AnalyzeFace() Confidence out of range")
	}
}

func TestEmotionServiceAnalyzeVoice(t *testing.T) {
	service := NewEmotionService()

	audioData := []byte{0x00, 0x01, 0x02, 0x03}

	analysis, err := service.AnalyzeVoice(audioData, 1000)
	if err != nil {
		t.Errorf("AnalyzeVoice() error = %v", err)
		return
	}

	if analysis == nil {
		t.Error("AnalyzeVoice() returned nil")
		return
	}

	if analysis.Timestamp != 1000 {
		t.Errorf("AnalyzeVoice() Timestamp = %d, want 1000", analysis.Timestamp)
	}

	if analysis.Emotion == "" {
		t.Error("AnalyzeVoice() returned empty Emotion")
	}

	if analysis.Features.Pitch <= 0 {
		t.Error("AnalyzeVoice() Pitch should be positive")
	}
}

func TestEmotionServiceAnalyzeBehavior(t *testing.T) {
	service := NewEmotionService()

	data := []BehaviorRhythm{
		{Timestamp: 1000, ActionType: "tap", Duration: 100, Interval: 200, Regularity: 0.8, Consistency: 0.9},
		{Timestamp: 2000, ActionType: "tap", Duration: 110, Interval: 190, Regularity: 0.85, Consistency: 0.85},
		{Timestamp: 3000, ActionType: "tap", Duration: 95, Interval: 210, Regularity: 0.9, Consistency: 0.95},
	}

	analysis, err := service.AnalyzeBehavior(data)
	if err != nil {
		t.Errorf("AnalyzeBehavior() error = %v", err)
		return
	}

	if analysis == nil {
		t.Error("AnalyzeBehavior() returned nil")
		return
	}

	if analysis.ActionType != "composite" {
		t.Errorf("AnalyzeBehavior() ActionType = %s, want composite", analysis.ActionType)
	}
}

func TestEmotionServiceAnalyzeAttention(t *testing.T) {
	service := NewEmotionService()

	data := []AttentionMetrics{
		{Timestamp: 1000, FocusScore: 0.9, GazeStability: 0.85, ResponseTime: 200, TaskCompletion: 0.95, DistractionCount: 1},
		{Timestamp: 2000, FocusScore: 0.85, GazeStability: 0.8, ResponseTime: 250, TaskCompletion: 0.9, DistractionCount: 0},
		{Timestamp: 3000, FocusScore: 0.88, GazeStability: 0.82, ResponseTime: 220, TaskCompletion: 0.92, DistractionCount: 1},
	}

	analysis, err := service.AnalyzeAttention(data)
	if err != nil {
		t.Errorf("AnalyzeAttention() error = %v", err)
		return
	}

	if analysis == nil {
		t.Error("AnalyzeAttention() returned nil")
		return
	}

	if analysis.FocusScore <= 0 || analysis.FocusScore > 1 {
		t.Error("AnalyzeAttention() FocusScore out of range")
	}
}

func TestEmotionServiceCreateEmotionProfile(t *testing.T) {
	service := NewEmotionService()

	profile, err := service.CreateEmotionProfile("user123", EmotionHappy)
	if err != nil {
		t.Errorf("CreateEmotionProfile() error = %v", err)
		return
	}

	if profile == nil {
		t.Error("CreateEmotionProfile() returned nil")
		return
	}

	if profile.UserID != "user123" {
		t.Errorf("CreateEmotionProfile() UserID = %s, want user123", profile.UserID)
	}

	if profile.BaselineEmotion != EmotionHappy {
		t.Errorf("CreateEmotionProfile() BaselineEmotion = %s, want happy", profile.BaselineEmotion)
	}
}

func TestEmotionServiceUpdateEmotionProfile(t *testing.T) {
	service := NewEmotionService()

	_, _ = service.CreateEmotionProfile("user123", EmotionNeutral)

	err := service.UpdateEmotionProfile("user123", EmotionHappy, 0.8, "test_context")
	if err != nil {
		t.Errorf("UpdateEmotionProfile() error = %v", err)
		return
	}

	profile, _ := service.GetEmotionProfile("user123")
	if len(profile.EmotionHistory) != 1 {
		t.Errorf("UpdateEmotionProfile() EmotionHistory length = %d, want 1", len(profile.EmotionHistory))
	}

	if profile.EmotionHistory[0].Emotion != EmotionHappy {
		t.Errorf("UpdateEmotionProfile() Emotion = %s, want happy", profile.EmotionHistory[0].Emotion)
	}
}

func TestEmotionServiceVerify(t *testing.T) {
	service := NewEmotionService()

	request := &EmotionVerificationRequest{
		SessionID: "session123",
		UserID:    "user123",
		TargetEmotion: EmotionHappy,
	}

	request.FaceFrames = []FaceAnalysis{
		{FrameID: 1, Timestamp: 1000, Emotion: EmotionHappy, Confidence: 0.85, EmotionScores: map[EmotionType]float64{EmotionHappy: 0.7}},
		{FrameID: 2, Timestamp: 1500, Emotion: EmotionHappy, Confidence: 0.80, EmotionScores: map[EmotionType]float64{EmotionHappy: 0.65}},
		{FrameID: 3, Timestamp: 2000, Emotion: EmotionHappy, Confidence: 0.82, EmotionScores: map[EmotionType]float64{EmotionHappy: 0.68}},
	}

	request.VoiceSamples = []VoiceAnalysis{
		{Timestamp: 1000, Emotion: EmotionHappy, Confidence: 0.75, EmotionScores: map[EmotionType]float64{EmotionHappy: 0.6}},
		{Timestamp: 2000, Emotion: EmotionHappy, Confidence: 0.78, EmotionScores: map[EmotionType]float64{EmotionHappy: 0.63}},
	}

	request.BehaviorData = []BehaviorRhythm{
		{Timestamp: 1000, ActionType: "tap", Duration: 100, Interval: 200, Regularity: 0.8, Consistency: 0.9},
		{Timestamp: 2000, ActionType: "tap", Duration: 110, Interval: 190, Regularity: 0.85, Consistency: 0.85},
		{Timestamp: 3000, ActionType: "tap", Duration: 95, Interval: 210, Regularity: 0.9, Consistency: 0.95},
		{Timestamp: 4000, ActionType: "tap", Duration: 105, Interval: 195, Regularity: 0.88, Consistency: 0.88},
		{Timestamp: 5000, ActionType: "tap", Duration: 100, Interval: 200, Regularity: 0.82, Consistency: 0.92},
	}

	request.AttentionData = []AttentionMetrics{
		{Timestamp: 1000, FocusScore: 0.9, GazeStability: 0.85, ResponseTime: 200, TaskCompletion: 0.95, DistractionCount: 1},
		{Timestamp: 2000, FocusScore: 0.85, GazeStability: 0.8, ResponseTime: 250, TaskCompletion: 0.9, DistractionCount: 0},
	}

	result, err := service.Verify(request)
	if err != nil {
		t.Errorf("Verify() error = %v", err)
		return
	}

	if result == nil {
		t.Error("Verify() returned nil")
		return
	}

	if result.ProcessingTime < 0 {
		t.Error("Verify() ProcessingTime should not be negative")
	}
}

func TestEmotionServiceGetDominantEmotion(t *testing.T) {
	service := NewEmotionService()

	frames := []FaceAnalysis{
		{Emotion: EmotionHappy, Confidence: 0.8},
		{Emotion: EmotionHappy, Confidence: 0.75},
		{Emotion: EmotionSad, Confidence: 0.7},
		{Emotion: EmotionHappy, Confidence: 0.85},
	}

	dominant := service.getDominantEmotion(frames)
	if dominant != EmotionHappy {
		t.Errorf("getDominantEmotion() = %s, want happy", dominant)
	}
}

func TestEmotionServiceExportConfig(t *testing.T) {
	service := NewEmotionService()

	data, err := service.ExportConfig()
	if err != nil {
		t.Errorf("ExportConfig() error = %v", err)
		return
	}

	if len(data) == 0 {
		t.Error("ExportConfig() returned empty data")
	}
}
