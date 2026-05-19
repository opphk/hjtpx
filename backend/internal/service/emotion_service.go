package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"
)

type EmotionType string

const (
	EmotionNeutral   EmotionType = "neutral"
	EmotionHappy     EmotionType = "happy"
	EmotionSad       EmotionType = "sad"
	EmotionAngry     EmotionType = "angry"
	EmotionFearful   EmotionType = "fearful"
	EmotionSurprised EmotionType = "surprised"
	EmotionDisgusted EmotionType = "disgusted"
	EmotionConfused  EmotionType = "confused"
)

type FaceAnalysis struct {
	FrameID      int64                `json:"frame_id"`
	Timestamp    int64                `json:"timestamp"`
	Emotion      EmotionType          `json:"emotion"`
	Confidence   float64              `json:"confidence"`
	EmotionScores map[EmotionType]float64 `json:"emotion_scores"`
	FaceBox      FaceBox              `json:"face_box"`
	Features     FaceFeatures          `json:"features"`
}

type FaceBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type FaceFeatures struct {
	EyeOpenness   float64 `json:"eye_openness"`
	MouthOpenness float64 `json:"mouth_openness"`
	SmileIntensity float64 `json:"smile_intensity"`
	BrowRaise    float64 `json:"brow_raise"`
	GazeDirection string  `json:"gaze_direction"`
}

type VoiceAnalysis struct {
	Timestamp   int64              `json:"timestamp"`
	Emotion     EmotionType        `json:"emotion"`
	Confidence  float64            `json:"confidence"`
	EmotionScores map[EmotionType]float64 `json:"emotion_scores"`
	Features    VoiceFeatures       `json:"features"`
}

type VoiceFeatures struct {
	Pitch         float64 `json:"pitch"`
	Energy        float64 `json:"energy"`
	SpeakingRate  float64 `json:"speaking_rate"`
	SilenceRatio  float64 `json:"silence_ratio"`
	Tremor       float64 `json:"tremor"`
	Jitter       float64 `json:"jitter"`
}

type BehaviorRhythm struct {
	Timestamp      int64   `json:"timestamp"`
	ActionType     string  `json:"action_type"`
	Duration       int64   `json:"duration"`
	Interval       int64   `json:"interval"`
	Regularity     float64 `json:"regularity"`
	Consistency    float64 `json:"consistency"`
}

type AttentionMetrics struct {
	Timestamp        int64   `json:"timestamp"`
	FocusScore       float64 `json:"focus_score"`
	GazeStability    float64 `json:"gaze_stability"`
	ResponseTime     int64   `json:"response_time"`
	TaskCompletion   float64 `json:"task_completion"`
	DistractionCount int     `json:"distraction_count"`
}

type EmotionVerificationRequest struct {
	SessionID      string              `json:"session_id"`
	UserID        string              `json:"user_id"`
	FaceFrames     []FaceAnalysis      `json:"face_frames"`
	VoiceSamples   []VoiceAnalysis     `json:"voice_samples"`
	BehaviorData   []BehaviorRhythm    `json:"behavior_data"`
	AttentionData  []AttentionMetrics  `json:"attention_data"`
	TargetEmotion  EmotionType         `json:"target_emotion"`
	Timestamp      int64              `json:"timestamp"`
}

type EmotionVerificationResult struct {
	IsValid         bool                   `json:"is_valid"`
	Confidence      float64                `json:"confidence"`
	EmotionMatch    float64                `json:"emotion_match"`
	AttentionScore  float64                `json:"attention_score"`
	RhythmScore     float64                `json:"rhythm_score"`
	AuthenticityScore float64              `json:"authenticity_score"`
	OverallScore   float64                `json:"overall_score"`
	Details        string                  `json:"details"`
	Metrics        map[string]interface{}  `json:"metrics"`
	ProcessingTime int64                  `json:"processing_time"`
}

type EmotionProfile struct {
	UserID         string                   `json:"user_id"`
	BaselineEmotion EmotionType             `json:"baseline_emotion"`
	EmotionHistory  []EmotionData          `json:"emotion_history"`
	AttentionBaseline float64              `json:"attention_baseline"`
	RhythmBaseline   BehaviorRhythm        `json:"rhythm_baseline"`
	UpdatedAt      time.Time               `json:"updated_at"`
}

type EmotionData struct {
	Emotion     EmotionType          `json:"emotion"`
	Intensity   float64              `json:"intensity"`
	Timestamp   int64                `json:"timestamp"`
	Context     string               `json:"context"`
}

type EmotionService struct {
	profiles     map[string]*EmotionProfile
	analyses     map[string][]*FaceAnalysis
	voiceAnalyses map[string][]*VoiceAnalysis
	config      *EmotionConfig
	mu          sync.RWMutex
}

type EmotionConfig struct {
	EmotionThreshold    float64 `json:"emotion_threshold"`
	AttentionThreshold  float64 `json:"attention_threshold"`
	RhythmThreshold     float64 `json:"rhythm_threshold"`
	AuthenticityWeight  float64 `json:"authenticity_weight"`
	FaceWeight         float64 `json:"face_weight"`
	VoiceWeight        float64 `json:"voice_weight"`
	BehaviorWeight     float64 `json:"behavior_weight"`
	MinFaceFrames      int     `json:"min_face_frames"`
	MinVoiceSamples    int     `json:"min_voice_samples"`
	MinBehaviorData    int     `json:"min_behavior_data"`
}

func NewEmotionService() *EmotionService {
	return &EmotionService{
		profiles:      make(map[string]*EmotionProfile),
		analyses:      make(map[string][]*FaceAnalysis),
		voiceAnalyses: make(map[string][]*VoiceAnalysis),
		config: &EmotionConfig{
			EmotionThreshold:   0.75,
			AttentionThreshold: 0.70,
			RhythmThreshold:    0.65,
			AuthenticityWeight: 0.3,
			FaceWeight:         0.4,
			VoiceWeight:        0.3,
			BehaviorWeight:     0.3,
			MinFaceFrames:      3,
			MinVoiceSamples:    2,
			MinBehaviorData:    5,
		},
	}
}

func (s *EmotionService) GetConfig() *EmotionConfig {
	return s.config
}

func (s *EmotionService) UpdateConfig(config *EmotionConfig) {
	s.config = config
}

func (s *EmotionService) AnalyzeFace(frame []byte, frameID int64) (*FaceAnalysis, error) {
	analysis := &FaceAnalysis{
		FrameID:   frameID,
		Timestamp: time.Now().UnixMilli(),
	}

	analysis.Emotion = s.detectEmotionFromFace(frame)
	analysis.Confidence = s.calculateEmotionConfidence(analysis.Emotion)

	analysis.EmotionScores = s.calculateEmotionScores(frame)

	analysis.FaceBox = FaceBox{
		X:      100,
		Y:      100,
		Width:  200,
		Height: 200,
	}

	analysis.Features = FaceFeatures{
		EyeOpenness:    0.8,
		MouthOpenness:  0.2,
		SmileIntensity: 0.6,
		BrowRaise:     0.3,
		GazeDirection: "center",
	}

	return analysis, nil
}

func (s *EmotionService) detectEmotionFromFace(frame []byte) EmotionType {
	emotions := []EmotionType{
		EmotionNeutral, EmotionHappy, EmotionSad, EmotionAngry,
		EmotionSurprised, EmotionConfused,
	}

	dominantIndex := int(time.Now().UnixNano()) % len(emotions)
	return emotions[dominantIndex]
}

func (s *EmotionService) calculateEmotionConfidence(emotion EmotionType) float64 {
	baseConfidence := 0.75

	emotionBonus := map[EmotionType]float64{
		EmotionNeutral:   0.10,
		EmotionHappy:     0.12,
		EmotionSad:       0.08,
		EmotionAngry:     0.09,
		EmotionSurprised: 0.11,
		EmotionConfused:  0.07,
	}

	if bonus, ok := emotionBonus[emotion]; ok {
		baseConfidence += bonus
	}

	return math.Min(1.0, baseConfidence)
}

func (s *EmotionService) calculateEmotionScores(frame []byte) map[EmotionType]float64 {
	scores := make(map[EmotionType]float64)
	total := 0.0

	emotions := []EmotionType{
		EmotionNeutral, EmotionHappy, EmotionSad, EmotionAngry,
		EmotionFearful, EmotionSurprised, EmotionDisgusted, EmotionConfused,
	}

	for i, emotion := range emotions {
		baseScore := 1.0 / float64(len(emotions))
		variation := math.Sin(float64(i) + float64(time.Now().UnixNano()%100)/10.0) * 0.1
		score := math.Max(0.1, baseScore+variation)
		scores[emotion] = score
		total += score
	}

	for emotion, score := range scores {
		scores[emotion] = score / total
	}

	return scores
}

func (s *EmotionService) AnalyzeVoice(audioData []byte, timestamp int64) (*VoiceAnalysis, error) {
	analysis := &VoiceAnalysis{
		Timestamp: timestamp,
	}

	analysis.Emotion = s.detectEmotionFromVoice(audioData)
	analysis.Confidence = s.calculateVoiceEmotionConfidence(analysis.Emotion)

	analysis.EmotionScores = s.calculateVoiceEmotionScores(audioData)

	analysis.Features = VoiceFeatures{
		Pitch:        200.0 + math.Sin(float64(timestamp)/1000.0)*50.0,
		Energy:       0.7 + math.Sin(float64(timestamp)/500.0)*0.2,
		SpeakingRate: 4.5 + math.Sin(float64(timestamp)/200.0)*0.5,
		SilenceRatio: 0.15,
		Tremor:       0.02,
		Jitter:       0.01,
	}

	return analysis, nil
}

func (s *EmotionService) detectEmotionFromVoice(audioData []byte) EmotionType {
	emotions := []EmotionType{
		EmotionNeutral, EmotionHappy, EmotionSad, EmotionAngry, EmotionSurprised,
	}

	dominantIndex := int(time.Now().UnixNano()) % len(emotions)
	return emotions[dominantIndex]
}

func (s *EmotionService) calculateVoiceEmotionConfidence(emotion EmotionType) float64 {
	baseConfidence := 0.70

	emotionBonus := map[EmotionType]float64{
		EmotionNeutral:   0.10,
		EmotionHappy:     0.12,
		EmotionSad:       0.08,
		EmotionAngry:     0.09,
		EmotionSurprised: 0.11,
	}

	if bonus, ok := emotionBonus[emotion]; ok {
		baseConfidence += bonus
	}

	return math.Min(1.0, baseConfidence)
}

func (s *EmotionService) calculateVoiceEmotionScores(audioData []byte) map[EmotionType]float64 {
	scores := make(map[EmotionType]float64)
	total := 0.0

	emotions := []EmotionType{
		EmotionNeutral, EmotionHappy, EmotionSad, EmotionAngry,
		EmotionFearful, EmotionSurprised, EmotionDisgusted,
	}

	for i, emotion := range emotions {
		baseScore := 1.0 / float64(len(emotions))
		variation := math.Cos(float64(i) + float64(time.Now().UnixNano()%100)/10.0) * 0.1
		score := math.Max(0.1, baseScore+variation)
		scores[emotion] = score
		total += score
	}

	for emotion, score := range scores {
		scores[emotion] = score / total
	}

	return scores
}

func (s *EmotionService) AnalyzeBehavior(data []BehaviorRhythm) (*BehaviorRhythm, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no behavior data provided")
	}

	analysis := &BehaviorRhythm{
		Timestamp:  time.Now().UnixMilli(),
		ActionType: "composite",
	}

	var totalDuration, totalInterval int64
	var durations, intervals []int64

	for _, d := range data {
		totalDuration += d.Duration
		totalInterval += d.Interval
		durations = append(durations, d.Duration)
		intervals = append(intervals, d.Interval)
	}

	analysis.Duration = totalDuration / int64(len(data))
	analysis.Interval = totalInterval / int64(len(data))

	analysis.Regularity = s.calculateRegularity(durations)
	analysis.Consistency = s.calculateConsistency(intervals)

	return analysis, nil
}

func (s *EmotionService) calculateRegularity(values []int64) float64 {
	if len(values) < 2 {
		return 1.0
	}

	mean := float64(sumInt64(values)) / float64(len(values))
	variance := 0.0

	for _, v := range values {
		diff := float64(v) - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	stdDev := math.Sqrt(variance)
	coefficientOfVariation := stdDev / math.Max(mean, 1.0)

	return math.Max(0.0, math.Min(1.0, 1.0-coefficientOfVariation))
}

func (s *EmotionService) calculateConsistency(values []int64) float64 {
	if len(values) < 2 {
		return 1.0
	}

	uniqueCount := len(uniqueInt64(values))
	consistencyRatio := float64(uniqueCount) / float64(len(values))

	return consistencyRatio
}

func sumInt64(values []int64) int64 {
	var sum int64
	for _, v := range values {
		sum += v
	}
	return sum
}

func uniqueInt64(values []int64) []int64 {
	seen := make(map[int64]bool)
	var unique []int64
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			unique = append(unique, v)
		}
	}
	return unique
}

func (s *EmotionService) AnalyzeAttention(data []AttentionMetrics) (*AttentionMetrics, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no attention data provided")
	}

	analysis := &AttentionMetrics{
		Timestamp: time.Now().UnixMilli(),
	}

	var totalFocus, totalGaze, totalTaskCompletion float64
	var totalResponseTime int64
	var totalDistractions int

	for _, d := range data {
		totalFocus += d.FocusScore
		totalGaze += d.GazeStability
		totalTaskCompletion += d.TaskCompletion
		totalResponseTime += d.ResponseTime
		totalDistractions += d.DistractionCount
	}

	count := float64(len(data))
	analysis.FocusScore = totalFocus / count
	analysis.GazeStability = totalGaze / count
	analysis.TaskCompletion = totalTaskCompletion / count
	analysis.ResponseTime = totalResponseTime / int64(count)
	analysis.DistractionCount = totalDistractions / int(count)

	return analysis, nil
}

func (s *EmotionService) CreateEmotionProfile(userID string, targetEmotion EmotionType) (*EmotionProfile, error) {
	profile := &EmotionProfile{
		UserID:         userID,
		BaselineEmotion: targetEmotion,
		EmotionHistory: []EmotionData{},
		AttentionBaseline: 0.85,
		UpdatedAt:      time.Now(),
	}

	s.profiles[userID] = profile
	return profile, nil
}

func (s *EmotionService) GetEmotionProfile(userID string) (*EmotionProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile, exists := s.profiles[userID]
	if !exists {
		return nil, fmt.Errorf("emotion profile not found for user: %s", userID)
	}

	return profile, nil
}

func (s *EmotionService) UpdateEmotionProfile(userID string, emotion EmotionType, intensity float64, context string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, exists := s.profiles[userID]
	if !exists {
		profile = &EmotionProfile{
			UserID: userID,
			EmotionHistory: []EmotionData{},
			UpdatedAt: time.Now(),
		}
		s.profiles[userID] = profile
	}

	emotionData := EmotionData{
		Emotion:   emotion,
		Intensity: intensity,
		Timestamp: time.Now().UnixMilli(),
		Context:   context,
	}

	profile.EmotionHistory = append(profile.EmotionHistory, emotionData)

	if len(profile.EmotionHistory) > 100 {
		profile.EmotionHistory = profile.EmotionHistory[len(profile.EmotionHistory)-100:]
	}

	profile.UpdatedAt = time.Now()

	return nil
}

func (s *EmotionService) StoreFaceAnalysis(sessionID string, analysis *FaceAnalysis) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.analyses[sessionID] = append(s.analyses[sessionID], analysis)
}

func (s *EmotionService) StoreVoiceAnalysis(sessionID string, analysis *VoiceAnalysis) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.voiceAnalyses[sessionID] = append(s.voiceAnalyses[sessionID], analysis)
}

func (s *EmotionService) Verify(request *EmotionVerificationRequest) (*EmotionVerificationResult, error) {
	startTime := time.Now()

	result := &EmotionVerificationResult{
		Metrics: make(map[string]interface{}),
	}

	if len(request.FaceFrames) < s.config.MinFaceFrames {
		result.Details = fmt.Sprintf("insufficient face frames: need %d, got %d",
			s.config.MinFaceFrames, len(request.FaceFrames))
		return result, nil
	}

	if len(request.VoiceSamples) < s.config.MinVoiceSamples {
		result.Details = fmt.Sprintf("insufficient voice samples: need %d, got %d",
			s.config.MinVoiceSamples, len(request.VoiceSamples))
		return result, nil
	}

	if len(request.BehaviorData) < s.config.MinBehaviorData {
		result.Details = fmt.Sprintf("insufficient behavior data: need %d, got %d",
			s.config.MinBehaviorData, len(request.BehaviorData))
		return result, nil
	}

	faceScore := s.calculateFaceEmotionMatch(request.FaceFrames, request.TargetEmotion)
	voiceScore := s.calculateVoiceEmotionMatch(request.VoiceSamples, request.TargetEmotion)
	behaviorScore := s.calculateBehaviorRhythmScore(request.BehaviorData)
	attentionScore := s.calculateAttentionScore(request.AttentionData)

	emotionMatch := faceScore*s.config.FaceWeight + voiceScore*s.config.VoiceWeight

	authenticityScore := s.evaluateAuthenticity(request)

	result.EmotionMatch = emotionMatch
	result.AttentionScore = attentionScore
	result.RhythmScore = behaviorScore
	result.AuthenticityScore = authenticityScore

	result.Confidence = faceScore*s.config.FaceWeight +
		voiceScore*s.config.VoiceWeight +
		behaviorScore*s.config.BehaviorWeight

	result.OverallScore = result.Confidence*s.config.EmotionThreshold +
		result.AttentionScore*s.config.AttentionThreshold*s.config.AuthenticityWeight +
		result.AuthenticityScore*s.config.AuthenticityWeight

	result.IsValid = result.OverallScore >= s.config.EmotionThreshold &&
		result.AttentionScore >= s.config.AttentionThreshold

	result.Details = fmt.Sprintf(
		"emotion match: %.2f, attention: %.2f, behavior: %.2f, authenticity: %.2f, overall: %.2f",
		emotionMatch, attentionScore, behaviorScore, authenticityScore, result.OverallScore,
	)

	result.ProcessingTime = time.Since(startTime).Milliseconds()

	result.Metrics["face_frames"] = len(request.FaceFrames)
	result.Metrics["voice_samples"] = len(request.VoiceSamples)
	result.Metrics["behavior_data"] = len(request.BehaviorData)
	result.Metrics["dominant_emotion"] = s.getDominantEmotion(request.FaceFrames)
	result.Metrics["voice_emotion"] = s.getDominantVoiceEmotion(request.VoiceSamples)

	return result, nil
}

func (s *EmotionService) calculateFaceEmotionMatch(frames []FaceAnalysis, target EmotionType) float64 {
	if len(frames) == 0 {
		return 0.0
	}

	targetCount := 0
	totalConfidence := 0.0

	for _, frame := range frames {
		if frame.Emotion == target {
			targetCount++
		}
		totalConfidence += frame.Confidence
	}

	matchRatio := float64(targetCount) / float64(len(frames))
	avgConfidence := totalConfidence / float64(len(frames))

	emotionScore := matchRatio*0.7 + avgConfidence*0.3

	return math.Min(1.0, emotionScore)
}

func (s *EmotionService) calculateVoiceEmotionMatch(samples []VoiceAnalysis, target EmotionType) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	targetCount := 0
	totalConfidence := 0.0

	for _, sample := range samples {
		if sample.Emotion == target {
			targetCount++
		}
		totalConfidence += sample.Confidence
	}

	matchRatio := float64(targetCount) / float64(len(samples))
	avgConfidence := totalConfidence / float64(len(samples))

	voiceScore := matchRatio*0.6 + avgConfidence*0.4

	return math.Min(1.0, voiceScore)
}

func (s *EmotionService) calculateBehaviorRhythmScore(data []BehaviorRhythm) float64 {
	if len(data) == 0 {
		return 0.0
	}

	var totalRegularity, totalConsistency float64

	for _, d := range data {
		totalRegularity += d.Regularity
		totalConsistency += d.Consistency
	}

	count := float64(len(data))
	avgRegularity := totalRegularity / count
	avgConsistency := totalConsistency / count

	baselineRegularity := 0.8
	baselineConsistency := 0.75

	regularityScore := avgRegularity / baselineRegularity
	consistencyScore := avgConsistency / baselineConsistency

	rhythmScore := (regularityScore*0.6 + consistencyScore*0.4) * 0.9

	return math.Min(1.0, rhythmScore)
}

func (s *EmotionService) calculateAttentionScore(data []AttentionMetrics) float64 {
	if len(data) == 0 {
		return 0.0
	}

	var totalFocus, totalGaze, totalTaskCompletion float64
	var distractionPenalty float64

	for _, d := range data {
		totalFocus += d.FocusScore
		totalGaze += d.GazeStability
		totalTaskCompletion += d.TaskCompletion

		if d.DistractionCount > 5 {
			distractionPenalty += 0.1 * float64(d.DistractionCount-5)
		}
	}

	count := float64(len(data))
	avgFocus := totalFocus / count
	avgGaze := totalGaze / count
	avgTaskCompletion := totalTaskCompletion / count

	attentionScore := avgFocus*0.4 + avgGaze*0.3 + avgTaskCompletion*0.3
	attentionScore -= distractionPenalty

	return math.Max(0.0, math.Min(1.0, attentionScore))
}

func (s *EmotionService) evaluateAuthenticity(request *EmotionVerificationRequest) float64 {
	var score float64 = 1.0

	for _, frame := range request.FaceFrames {
		if frame.Confidence > 0.95 {
			score -= 0.05
		}
	}

	for _, sample := range request.VoiceSamples {
		if sample.Features.Tremor > 0.1 {
			score -= 0.03
		}
		if sample.Features.Jitter > 0.05 {
			score -= 0.02
		}
	}

	for _, behavior := range request.BehaviorData {
		if behavior.Consistency > 0.95 {
			score -= 0.04
		}
	}

	return math.Max(0.0, math.Min(1.0, score))
}

func (s *EmotionService) getDominantEmotion(frames []FaceAnalysis) EmotionType {
	if len(frames) == 0 {
		return EmotionNeutral
	}

	emotionCounts := make(map[EmotionType]int)
	for _, frame := range frames {
		emotionCounts[frame.Emotion]++
	}

	var dominant EmotionType
	maxCount := 0
	for emotion, count := range emotionCounts {
		if count > maxCount {
			maxCount = count
			dominant = emotion
		}
	}

	return dominant
}

func (s *EmotionService) getDominantVoiceEmotion(samples []VoiceAnalysis) EmotionType {
	if len(samples) == 0 {
		return EmotionNeutral
	}

	emotionCounts := make(map[EmotionType]int)
	for _, sample := range samples {
		emotionCounts[sample.Emotion]++
	}

	var dominant EmotionType
	maxCount := 0
	for emotion, count := range emotionCounts {
		if count > maxCount {
			maxCount = count
			dominant = emotion
		}
	}

	return dominant
}

func (s *EmotionService) ExportProfiles() (map[string]*EmotionProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*EmotionProfile)
	for id, profile := range s.profiles {
		result[id] = profile
	}

	return result, nil
}

func (s *EmotionService) ExportConfig() ([]byte, error) {
	return json.Marshal(s.config)
}
