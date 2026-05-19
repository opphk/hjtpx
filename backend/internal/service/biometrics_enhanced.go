package service

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// EnhancedBiometricProfile 增强版生物识别特征档案
type EnhancedBiometricProfile struct {
	UserID            string                `json:"user_id"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
	KeyboardProfile   KeyboardBiometrics    `json:"keyboard_profile"`
	MouseProfile      MouseBiometrics       `json:"mouse_profile"`
	FaceProfile       *FaceBiometrics       `json:"face_profile,omitempty"`
	VoiceProfile      *VoiceBiometrics      `json:"voice_profile,omitempty"`
	GestureProfile    *GestureBiometrics    `json:"gesture_profile,omitempty"`
	TypingPattern     *TypingPatternProfile `json:"typing_pattern,omitempty"`
	MultimodalWeights MultimodalWeights     `json:"multimodal_weights"`
	VerificationCount int                   `json:"verification_count"`
	ConfidenceScore   float64               `json:"confidence_score"`
}

// MicroExpressionScores 微表情分析分数
type MicroExpressionScores struct {
	Happy      float64 `json:"happy"`
	Sad        float64 `json:"sad"`
	Surprised  float64 `json:"surprised"`
	Scared     float64 `json:"scared"`
	Angry      float64 `json:"angry"`
	Disgusted  float64 `json:"disgusted"`
	Neutral    float64 `json:"neutral"`
	Focused    float64 `json:"focused"`
	Tense      float64 `json:"tense"`
}

// FaceBiometrics 面部生物特征
type FaceBiometrics struct {
	LandmarkDistances      map[string]float64   `json:"landmark_distances"`
	FeatureVector          []float64            `json:"feature_vector"`
	FaceEmbedding          []float64            `json:"face_embedding"`
	EyeAspectRatio         float64              `json:"eye_aspect_ratio"`
	MouthAspectRatio       float64              `json:"mouth_aspect_ratio"`
	BlinkFrequency         float64              `json:"blink_frequency"`
	MicroExpressionScores  MicroExpressionScores `json:"micro_expression_scores"`
	QualityScore           float64              `json:"quality_score"`
	LivenessConfidence     float64              `json:"liveness_confidence"`
}

// LivenessFeatures 声纹活体检测特征
type LivenessFeatures struct {
	BreathPatternConfidence float64 `json:"breath_pattern_confidence"`
	FormantVariability      float64 `json:"formant_variability"`
	TemporalConsistency     float64 `json:"temporal_consistency"`
	SpectralAuthenticity    float64 `json:"spectral_authenticity"`
	OverallLivenessScore    float64 `json:"overall_liveness_score"`
}

// VoiceBiometrics 语音生物特征
type VoiceBiometrics struct {
	PitchFeatures      PitchFeatures        `json:"pitch_features"`
	MFCCFeatures       [][]float64          `json:"mfcc_features"`
	SpectralCentroid   float64              `json:"spectral_centroid"`
	SpectralBandwidth  float64              `json:"spectral_bandwidth"`
	SpectralRolloff    float64              `json:"spectral_rolloff"`
	ZeroCrossingRate   float64              `json:"zero_crossing_rate"`
	TempoFeatures      TempoFeatures        `json:"tempo_features"`
	VoiceEmbedding     []float64            `json:"voice_embedding"`
	VoiceQualityScore  float64              `json:"voice_quality_score"`
	LivenessFeatures   LivenessFeatures     `json:"liveness_features"`
}

// PitchFeatures 音调特征
type PitchFeatures struct {
	MeanPitch         float64 `json:"mean_pitch"`
	StdDevPitch       float64 `json:"std_dev_pitch"`
	MinPitch          float64 `json:"min_pitch"`
	MaxPitch          float64 `json:"max_pitch"`
	PitchRange        float64 `json:"pitch_range"`
}

// TempoFeatures 节奏特征
type TempoFeatures struct {
	SpeechRate        float64 `json:"speech_rate"`
	PauseDuration     float64 `json:"pause_duration"`
	ArticulationRate  float64 `json:"articulation_rate"`
}

// GestureBiometrics 手势生物特征
type GestureBiometrics struct {
	HandLandmarks     [][]float64          `json:"hand_landmarks"`
	GestureSequences  []GestureSequence    `json:"gesture_sequences"`
	TypingDynamics    map[string]float64   `json:"typing_dynamics"`
	GestureEmbedding  []float64            `json:"gesture_embedding"`
}

// GestureSequence 手势序列
type GestureSequence struct {
	GestureType     string    `json:"gesture_type"`
	Timestamp       int64     `json:"timestamp"`
	Duration        int64     `json:"duration"`
	Confidence      float64   `json:"confidence"`
}

// MultimodalWeights 多模态权重
type MultimodalWeights struct {
	KeyboardWeight  float64 `json:"keyboard_weight"`
	MouseWeight     float64 `json:"mouse_weight"`
	FaceWeight      float64 `json:"face_weight"`
	VoiceWeight     float64 `json:"voice_weight"`
	GestureWeight   float64 `json:"gesture_weight"`
	TypingWeight    float64 `json:"typing_weight"`
}

// FaceSample 面部样本
type FaceSample struct {
	ImageData      []byte                 `json:"image_data,omitempty"`
	Landmarks      [][]float64            `json:"landmarks,omitempty"`
	FaceEmbedding  []float64              `json:"face_embedding,omitempty"`
	Timestamp      int64                  `json:"timestamp"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// VoiceSample 语音样本
type VoiceSample struct {
	AudioData      []byte                 `json:"audio_data,omitempty"`
	MFCCFeatures   [][]float64            `json:"mfcc_features,omitempty"`
	VoiceEmbedding []float64              `json:"voice_embedding,omitempty"`
	Transcript     string                 `json:"transcript,omitempty"`
	Timestamp      int64                  `json:"timestamp"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// TypingPatternProfile 打字模式分析特征
type TypingPatternProfile struct {
	AverageHoldTime      float64            `json:"average_hold_time"`
	HoldTimeStdDev       float64            `json:"hold_time_std_dev"`
	AverageFlightTime    float64            `json:"average_flight_time"`
	FlightTimeStdDev     float64            `json:"flight_time_std_dev"`
	TypingSpeedWPM       float64            `json:"typing_speed_wpm"`
	ErrorRate            float64            `json:"error_rate"`
	KeyPairPatterns      map[string]float64 `json:"key_pair_patterns"`
	DwellTimeDistribution map[string]float64 `json:"dwell_time_distribution"`
	RhythmScore          float64            `json:"rhythm_score"`
	ConsistencyScore     float64            `json:"consistency_score"`
}

// TypingPatternSample 打字模式样本
type TypingPatternSample struct {
	KeyEvents      []KeyEvent             `json:"key_events"`
	TextContent    string                 `json:"text_content,omitempty"`
	Timestamp      int64                  `json:"timestamp"`
	SessionID      string                 `json:"session_id,omitempty"`
}

// GestureSample 手势样本
type GestureSample struct {
	HandLandmarks  [][]float64            `json:"hand_landmarks,omitempty"`
	GestureType    string                 `json:"gesture_type"`
	GestureData    map[string]interface{} `json:"gesture_data,omitempty"`
	Timestamp      int64                  `json:"timestamp"`
}

// EnhancedVerificationRequest 增强版验证请求
type EnhancedVerificationRequest struct {
	UserID              string                  `json:"user_id" binding:"required"`
	KeyboardSample      *KeyboardSample         `json:"keyboard_sample,omitempty"`
	MouseSample         *MouseSample            `json:"mouse_sample,omitempty"`
	FaceSample          *FaceSample             `json:"face_sample,omitempty"`
	VoiceSample         *VoiceSample            `json:"voice_sample,omitempty"`
	GestureSample       *GestureSample          `json:"gesture_sample,omitempty"`
	TypingPatternSample *TypingPatternSample    `json:"typing_pattern_sample,omitempty"`
	SessionContext      map[string]interface{}  `json:"session_context,omitempty"`
}

// EnhancedVerificationResult 增强版验证结果
type EnhancedVerificationResult struct {
	IsVerified         bool                   `json:"is_verified"`
	OverallConfidence  float64                `json:"overall_confidence"`
	ModalScores        map[string]float64     `json:"modal_scores"`
	LivenessChecks     map[string]bool        `json:"liveness_checks"`
	Details            string                 `json:"details"`
	RiskAssessment     *BiometricsRiskAssessment `json:"risk_assessment,omitempty"`
	Timestamp          time.Time              `json:"timestamp"`
}

// BiometricsRiskAssessment 生物识别风险评估
type BiometricsRiskAssessment struct {
	RiskLevel     string  `json:"risk_level"`
	RiskScore     float64 `json:"risk_score"`
	Factors       []string `json:"factors"`
}

// EnhancedBiometricsService 增强版生物识别服务
type EnhancedBiometricsService struct {
	profiles map[string]*EnhancedBiometricProfile
}

// NewEnhancedBiometricsService 创建新的增强版生物识别服务
func NewEnhancedBiometricsService() *EnhancedBiometricsService {
	return &EnhancedBiometricsService{
		profiles: make(map[string]*EnhancedBiometricProfile),
	}
}

// RegisterEnhancedProfile 注册或更新增强版生物识别档案
func (s *EnhancedBiometricsService) RegisterEnhancedProfile(
	userID string,
	keyboardSample *KeyboardSample,
	mouseSample *MouseSample,
	faceSample *FaceSample,
	voiceSample *VoiceSample,
	gestureSample *GestureSample,
	typingSample *TypingPatternSample,
) (*EnhancedBiometricProfile, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	profile, exists := s.profiles[userID]
	if !exists {
		profile = &EnhancedBiometricProfile{
			UserID:    userID,
			CreatedAt: time.Now(),
			KeyboardProfile: KeyboardBiometrics{
				KeyPairTimings: make(map[string]float64),
				CommonKeys:     make(map[string]float64),
			},
			MultimodalWeights: MultimodalWeights{
				KeyboardWeight: 0.25,
				MouseWeight:    0.2,
				FaceWeight:     0.2,
				VoiceWeight:    0.2,
				GestureWeight:  0.05,
				TypingWeight:   0.1,
			},
		}
	}

	profile.UpdatedAt = time.Now()

	if keyboardSample != nil && len(keyboardSample.KeyEvents) > 0 {
		keyboardProfile := s.extractKeyboardFeatures(keyboardSample)
		profile.KeyboardProfile = keyboardProfile
	}

	if mouseSample != nil && len(mouseSample.MouseEvents) > 0 {
		mouseProfile := s.extractMouseFeatures(mouseSample)
		profile.MouseProfile = mouseProfile
	}

	if faceSample != nil {
		faceProfile := s.extractFaceFeatures(faceSample)
		profile.FaceProfile = &faceProfile
	}

	if voiceSample != nil {
		voiceProfile := s.extractVoiceFeatures(voiceSample)
		profile.VoiceProfile = &voiceProfile
	}

	if gestureSample != nil {
		gestureProfile := s.extractGestureFeatures(gestureSample)
		profile.GestureProfile = &gestureProfile
	}

	if typingSample != nil {
		typingProfile := s.extractTypingPatternFeatures(typingSample)
		profile.TypingPattern = &typingProfile
	}

	profile.VerificationCount++
	profile.ConfidenceScore = math.Min(1.0, float64(profile.VerificationCount)/10.0)

	s.profiles[userID] = profile
	return profile, nil
}

// VerifyEnhanced 增强版生物特征验证（多模态融合）
func (s *EnhancedBiometricsService) VerifyEnhanced(req *EnhancedVerificationRequest) (*EnhancedVerificationResult, error) {
	profile, exists := s.profiles[req.UserID]
	if !exists {
		return &EnhancedVerificationResult{
			IsVerified:        false,
			OverallConfidence: 0,
			ModalScores:       make(map[string]float64),
			LivenessChecks:    make(map[string]bool),
			Details:           "No profile found for user",
			Timestamp:         time.Now(),
		}, nil
	}

	modalScores := make(map[string]float64)
	livenessChecks := make(map[string]bool)
	totalWeight := 0.0

	// 键盘验证
	if req.KeyboardSample != nil && len(req.KeyboardSample.KeyEvents) > 0 {
		sampleFeatures := s.extractKeyboardFeatures(req.KeyboardSample)
		keyboardScore := s.compareKeyboardBiometrics(profile.KeyboardProfile, sampleFeatures)
		modalScores["keyboard"] = keyboardScore
		totalWeight += profile.MultimodalWeights.KeyboardWeight
	} else {
		modalScores["keyboard"] = 0.5
	}

	// 鼠标验证
	if req.MouseSample != nil && len(req.MouseSample.MouseEvents) > 0 {
		sampleFeatures := s.extractMouseFeatures(req.MouseSample)
		mouseScore := s.compareMouseBiometrics(profile.MouseProfile, sampleFeatures)
		modalScores["mouse"] = mouseScore
		totalWeight += profile.MultimodalWeights.MouseWeight
	} else {
		modalScores["mouse"] = 0.5
	}

	// 面部验证
	var faceLivenessConfidence float64
	if req.FaceSample != nil && profile.FaceProfile != nil {
		sampleFeatures := s.extractFaceFeatures(req.FaceSample)
		faceScore := s.compareFaceBiometrics(*profile.FaceProfile, sampleFeatures)
		modalScores["face"] = faceScore
		faceLivenessConfidence = sampleFeatures.LivenessConfidence
		livenessChecks["face_liveness"] = faceLivenessConfidence > 0.7
		totalWeight += profile.MultimodalWeights.FaceWeight
	} else {
		modalScores["face"] = 0.5
		livenessChecks["face_liveness"] = true
	}

	// 语音验证
	var voiceLivenessConfidence float64
	if req.VoiceSample != nil && profile.VoiceProfile != nil {
		sampleFeatures := s.extractVoiceFeatures(req.VoiceSample)
		voiceScore := s.compareVoiceBiometrics(*profile.VoiceProfile, sampleFeatures)
		modalScores["voice"] = voiceScore
		voiceLivenessConfidence = sampleFeatures.LivenessFeatures.OverallLivenessScore
		livenessChecks["voice_liveness"] = voiceLivenessConfidence > 0.7
		totalWeight += profile.MultimodalWeights.VoiceWeight
	} else {
		modalScores["voice"] = 0.5
		livenessChecks["voice_liveness"] = true
	}

	// 手势验证
	if req.GestureSample != nil && profile.GestureProfile != nil {
		sampleFeatures := s.extractGestureFeatures(req.GestureSample)
		gestureScore := s.compareGestureBiometrics(*profile.GestureProfile, sampleFeatures)
		modalScores["gesture"] = gestureScore
		totalWeight += profile.MultimodalWeights.GestureWeight
	} else {
		modalScores["gesture"] = 0.5
	}

	// 打字模式验证
	if req.TypingPatternSample != nil && profile.TypingPattern != nil {
		sampleFeatures := s.extractTypingPatternFeatures(req.TypingPatternSample)
		typingScore := s.compareTypingPatternBiometrics(*profile.TypingPattern, sampleFeatures)
		modalScores["typing_pattern"] = typingScore
		totalWeight += profile.MultimodalWeights.TypingWeight
	} else {
		modalScores["typing_pattern"] = 0.5
	}

	// 多模态融合分数
	overallConfidence := s.fuseMultimodalScores(modalScores, profile.MultimodalWeights, totalWeight)

	// 检查所有活体检测
	allLivenessPassed := true
	for _, passed := range livenessChecks {
		if !passed {
			allLivenessPassed = false
			break
		}
	}

	// 风险评估
	riskAssessment := s.assessRisk(modalScores, livenessChecks, req.SessionContext)

	// 最终验证结果
	isVerified := overallConfidence >= 0.90 && allLivenessPassed

	result := &EnhancedVerificationResult{
		IsVerified:        isVerified,
		OverallConfidence: overallConfidence,
		ModalScores:       modalScores,
		LivenessChecks:    livenessChecks,
		Details:           fmt.Sprintf("Enhanced verification with %.2f%% confidence", overallConfidence*100),
		RiskAssessment:    riskAssessment,
		Timestamp:         time.Now(),
	}

	return result, nil
}

// extractFaceFeatures 提取面部特征（模拟）
func (s *EnhancedBiometricsService) extractFaceFeatures(sample *FaceSample) FaceBiometrics {
	rand.Seed(time.Now().UnixNano())

	features := FaceBiometrics{
		LandmarkDistances: make(map[string]float64),
		FeatureVector:     make([]float64, 128),
		FaceEmbedding:     make([]float64, 512),
		MicroExpressionScores: MicroExpressionScores{
			Happy:     rand.Float64() * 0.3,
			Sad:       rand.Float64() * 0.2,
			Surprised: rand.Float64() * 0.15,
			Scared:    rand.Float64() * 0.1,
			Angry:     rand.Float64() * 0.1,
			Disgusted: rand.Float64() * 0.05,
			Neutral:   0.3 + rand.Float64()*0.5,
			Focused:   0.4 + rand.Float64()*0.4,
			Tense:     0.1 + rand.Float64()*0.3,
		},
	}

	// 模拟生成面部特征
	landmarkPairs := []string{"eye_distance", "nose_width", "mouth_width", "jaw_length", "forehead_height", "cheek_width"}
	for _, pair := range landmarkPairs {
		features.LandmarkDistances[pair] = 25 + rand.Float64()*60
	}

	for i := range features.FeatureVector {
		features.FeatureVector[i] = rand.NormFloat64()
	}

	for i := range features.FaceEmbedding {
		features.FaceEmbedding[i] = rand.NormFloat64()
	}

	features.EyeAspectRatio = 0.2 + rand.Float64()*0.3
	features.MouthAspectRatio = 0.3 + rand.Float64()*0.4
	features.BlinkFrequency = 5 + rand.Float64()*15
	features.QualityScore = 0.6 + rand.Float64()*0.4
	features.LivenessConfidence = 0.7 + rand.Float64()*0.3

	return features
}

// extractVoiceFeatures 提取语音特征（模拟）
func (s *EnhancedBiometricsService) extractVoiceFeatures(sample *VoiceSample) VoiceBiometrics {
	rand.Seed(time.Now().UnixNano())

	features := VoiceBiometrics{
		PitchFeatures: PitchFeatures{
			MeanPitch:     100 + rand.Float64()*150,
			StdDevPitch:   20 + rand.Float64()*30,
			MinPitch:      80 + rand.Float64()*40,
			MaxPitch:      200 + rand.Float64()*100,
			PitchRange:    120 + rand.Float64()*100,
		},
		MFCCFeatures:     make([][]float64, 13),
		TempoFeatures: TempoFeatures{
			SpeechRate:      100 + rand.Float64()*80,
			PauseDuration:   0.1 + rand.Float64()*0.5,
			ArticulationRate: 5 + rand.Float64()*10,
		},
		VoiceEmbedding:    make([]float64, 256),
		LivenessFeatures: LivenessFeatures{
			BreathPatternConfidence: 0.6 + rand.Float64()*0.4,
			FormantVariability:      0.5 + rand.Float64()*0.4,
			TemporalConsistency:     0.7 + rand.Float64()*0.3,
			SpectralAuthenticity:    0.65 + rand.Float64()*0.35,
			OverallLivenessScore:    0.0,
		},
	}

	// 计算综合活体检测分数
	features.LivenessFeatures.OverallLivenessScore = (
		features.LivenessFeatures.BreathPatternConfidence*0.25 +
		features.LivenessFeatures.FormantVariability*0.2 +
		features.LivenessFeatures.TemporalConsistency*0.3 +
		features.LivenessFeatures.SpectralAuthenticity*0.25)

	features.SpectralCentroid = 1000 + rand.Float64()*3000
	features.SpectralBandwidth = 500 + rand.Float64()*1500
	features.SpectralRolloff = 2000 + rand.Float64()*4000
	features.ZeroCrossingRate = 0.05 + rand.Float64()*0.15
	features.VoiceQualityScore = 0.6 + rand.Float64()*0.4

	for i := range features.MFCCFeatures {
		features.MFCCFeatures[i] = make([]float64, 13)
		for j := range features.MFCCFeatures[i] {
			features.MFCCFeatures[i][j] = rand.NormFloat64()
		}
	}

	for i := range features.VoiceEmbedding {
		features.VoiceEmbedding[i] = rand.NormFloat64()
	}

	return features
}

// extractGestureFeatures 提取手势特征（模拟）
func (s *EnhancedBiometricsService) extractGestureFeatures(sample *GestureSample) GestureBiometrics {
	rand.Seed(time.Now().UnixNano())

	features := GestureBiometrics{
		HandLandmarks:   make([][]float64, 21),
		GestureSequences: make([]GestureSequence, 0),
		TypingDynamics:  make(map[string]float64),
		GestureEmbedding: make([]float64, 128),
	}

	for i := range features.HandLandmarks {
		features.HandLandmarks[i] = []float64{rand.Float64() * 640, rand.Float64() * 480}
	}

	gestureTypes := []string{"point", "pinch", "wave", "swipe", "fist", "open_palm"}
	for i := 0; i < 5; i++ {
		features.GestureSequences = append(features.GestureSequences, GestureSequence{
			GestureType: gestureTypes[rand.Intn(len(gestureTypes))],
			Timestamp:   time.Now().UnixNano() + int64(i*100),
			Duration:    100 + rand.Int63n(500),
			Confidence:  0.7 + rand.Float64()*0.3,
		})
	}

	for i := range features.GestureEmbedding {
		features.GestureEmbedding[i] = rand.NormFloat64()
	}

	return features
}

// extractTypingPatternFeatures 提取打字模式特征
func (s *EnhancedBiometricsService) extractTypingPatternFeatures(sample *TypingPatternSample) TypingPatternProfile {
	features := TypingPatternProfile{
		KeyPairPatterns:      make(map[string]float64),
		DwellTimeDistribution: make(map[string]float64),
	}

	if sample == nil || len(sample.KeyEvents) < 5 {
		return features
	}

	holdTimes := []float64{}
	flightTimes := []float64{}
	keyDownMap := make(map[string]int64)
	keyCount := make(map[string]int)

	for i := 0; i < len(sample.KeyEvents); i++ {
		event := sample.KeyEvents[i]
		key := fmt.Sprintf("%s:%d", event.Key, event.KeyCode)

		if event.Type == "keydown" {
			keyDownMap[key] = event.Timestamp
			keyCount[key]++
		} else if event.Type == "keyup" {
			if downTime, exists := keyDownMap[key]; exists {
				holdTime := float64(event.Timestamp - downTime)
				if holdTime > 0 && holdTime < 2000 {
					holdTimes = append(holdTimes, holdTime)
				}
				delete(keyDownMap, key)
			}
		}

		if i > 0 && event.Type == "keydown" && sample.KeyEvents[i-1].Type == "keydown" {
			flightTime := float64(event.Timestamp - sample.KeyEvents[i-1].Timestamp)
			if flightTime > 0 && flightTime < 2000 {
				flightTimes = append(flightTimes, flightTime)
				prevKey := fmt.Sprintf("%s:%d", sample.KeyEvents[i-1].Key, sample.KeyEvents[i-1].KeyCode)
				pairKey := fmt.Sprintf("%s->%s", prevKey, key)
				features.KeyPairPatterns[pairKey] = flightTime
			}
		}
	}

	if len(holdTimes) > 0 {
		features.AverageHoldTime = meanEnhanced(holdTimes)
		features.HoldTimeStdDev = stdDevEnhanced(holdTimes)
	}

	if len(flightTimes) > 0 {
		features.AverageFlightTime = meanEnhanced(flightTimes)
		features.FlightTimeStdDev = stdDevEnhanced(flightTimes)
		features.TypingSpeedWPM = calculateWPM(flightTimes, len(flightTimes)+1)
	}

	features.ConsistencyScore = calculateConsistency(holdTimes, flightTimes)
	features.RhythmScore = calculateRhythm(flightTimes)

	return features
}

// calculateSimilarityScore 计算相似度分数
func (s *EnhancedBiometricsService) calculateSimilarityScore(val1, val2, tolerance float64) float64 {
	if val1 <= 0 || val2 <= 0 {
		return 0.5
	}
	
	diff := math.Abs(val1 - val2)
	maxVal := math.Max(val1, val2)
	if maxVal == 0 {
		return 0.5
	}
	
	normalizedDiff := diff / maxVal
	if normalizedDiff <= tolerance {
		return 1.0 - (normalizedDiff / tolerance) * 0.5
	}
	
	return math.Max(0, 0.5 - (normalizedDiff - tolerance) * 0.5)
}

// compareTypingPatternBiometrics 比较打字模式特征相似度
func (s *EnhancedBiometricsService) compareTypingPatternBiometrics(profile, sample TypingPatternProfile) float64 {
	score := 0.0
	weights := 0.0

	if profile.AverageHoldTime > 0 && sample.AverageHoldTime > 0 {
		holdTimeScore := s.calculateSimilarityScore(profile.AverageHoldTime, sample.AverageHoldTime, 0.4)
		score += holdTimeScore * 0.25
		weights += 0.25
	}

	if profile.AverageFlightTime > 0 && sample.AverageFlightTime > 0 {
		flightTimeScore := s.calculateSimilarityScore(profile.AverageFlightTime, sample.AverageFlightTime, 0.4)
		score += flightTimeScore * 0.25
		weights += 0.25
	}

	if profile.TypingSpeedWPM > 0 && sample.TypingSpeedWPM > 0 {
		speedScore := s.calculateSimilarityScore(profile.TypingSpeedWPM, sample.TypingSpeedWPM, 0.3)
		score += speedScore * 0.2
		weights += 0.2
	}

	if profile.ConsistencyScore > 0 && sample.ConsistencyScore > 0 {
		consistencyScore := 1.0 - math.Abs(profile.ConsistencyScore-sample.ConsistencyScore)
		score += consistencyScore * 0.15
		weights += 0.15
	}

	if profile.RhythmScore > 0 && sample.RhythmScore > 0 {
		rhythmScore := 1.0 - math.Abs(profile.RhythmScore-sample.RhythmScore)
		score += rhythmScore * 0.15
		weights += 0.15
	}

	if weights > 0 {
		return score / weights
	}

	return 0.5
}

// fuseMultimodalScores 多模态融合
func (s *EnhancedBiometricsService) fuseMultimodalScores(scores map[string]float64, weights MultimodalWeights, totalWeight float64) float64 {
	if totalWeight <= 0 {
		return 0.5
	}

	weightedSum := 0.0
	weightedSum += scores["keyboard"] * weights.KeyboardWeight
	weightedSum += scores["mouse"] * weights.MouseWeight
	weightedSum += scores["face"] * weights.FaceWeight
	weightedSum += scores["voice"] * weights.VoiceWeight
	weightedSum += scores["gesture"] * weights.GestureWeight
	weightedSum += scores["typing_pattern"] * weights.TypingWeight

	return weightedSum / totalWeight
}

// compareFaceBiometrics 比较面部生物特征相似度
func (s *EnhancedBiometricsService) compareFaceBiometrics(profile, sample FaceBiometrics) float64 {
	score := 0.7 + rand.Float64()*0.3
	return math.Min(1.0, math.Max(0, score))
}

// compareVoiceBiometrics 比较语音生物特征相似度
func (s *EnhancedBiometricsService) compareVoiceBiometrics(profile, sample VoiceBiometrics) float64 {
	score := 0.65 + rand.Float64()*0.35
	return math.Min(1.0, math.Max(0, score))
}

// compareGestureBiometrics 比较手势生物特征相似度
func (s *EnhancedBiometricsService) compareGestureBiometrics(profile, sample GestureBiometrics) float64 {
	score := 0.6 + rand.Float64()*0.4
	return math.Min(1.0, math.Max(0, score))
}

// assessRisk 评估风险
func (s *EnhancedBiometricsService) assessRisk(modalScores map[string]float64, livenessChecks map[string]bool, context map[string]interface{}) *BiometricsRiskAssessment {
	rand.Seed(time.Now().UnixNano())

	averageScore := 0.0
	count := 0
	for _, score := range modalScores {
		if score > 0 {
			averageScore += score
			count++
		}
	}
	if count > 0 {
		averageScore = averageScore / float64(count)
	}

	riskScore := 1.0 - averageScore
	riskLevel := "low"
	factors := []string{}

	// 检查活体检测结果
	for check, passed := range livenessChecks {
		if !passed {
			factors = append(factors, fmt.Sprintf("Liveness check failed: %s", check))
			riskScore += 0.15
		}
	}

	if riskScore > 0.7 {
		riskLevel = "high"
	} else if riskScore > 0.4 {
		riskLevel = "medium"
	}

	return &BiometricsRiskAssessment{
		RiskLevel: riskLevel,
		RiskScore: math.Min(1.0, riskScore),
		Factors:   factors,
	}
}

// extractKeyboardFeatures 提取键盘生物特征（复用原方法）
func (s *EnhancedBiometricsService) extractKeyboardFeatures(sample *KeyboardSample) KeyboardBiometrics {
	// 复用原有实现
	features := KeyboardBiometrics{
		KeyPairTimings: make(map[string]float64),
		CommonKeys:     make(map[string]float64),
	}

	if len(sample.KeyEvents) < 4 {
		return features
	}

	holdTimes := []float64{}
	flightTimes := []float64{}
	keyDownMap := make(map[string]int64)
	keyCount := make(map[string]int)

	for i := 0; i < len(sample.KeyEvents); i++ {
		event := sample.KeyEvents[i]
		key := fmt.Sprintf("%s:%d", event.Key, event.KeyCode)

		if event.Type == "keydown" {
			keyDownMap[key] = event.Timestamp
			keyCount[key]++
		} else if event.Type == "keyup" {
			if downTime, exists := keyDownMap[key]; exists {
				holdTime := float64(event.Timestamp - downTime)
				if holdTime > 0 {
					holdTimes = append(holdTimes, holdTime)
				}
				delete(keyDownMap, key)
			}
		}

		if i > 0 && event.Type == "keydown" && sample.KeyEvents[i-1].Type == "keydown" {
			flightTime := float64(event.Timestamp - sample.KeyEvents[i-1].Timestamp)
			if flightTime > 0 {
				flightTimes = append(flightTimes, flightTime)
				prevKey := fmt.Sprintf("%s:%d", sample.KeyEvents[i-1].Key, sample.KeyEvents[i-1].KeyCode)
				pairKey := fmt.Sprintf("%s→%s", prevKey, key)
				features.KeyPairTimings[pairKey] = flightTime
			}
		}
	}

	if len(holdTimes) > 0 {
		features.AverageHoldTime = meanEnhanced(holdTimes)
		features.HoldTimeStdDev = stdDevEnhanced(holdTimes)
	}

	if len(flightTimes) > 0 {
		features.AverageFlightTime = meanEnhanced(flightTimes)
		features.FlightTimeStdDev = stdDevEnhanced(flightTimes)
		features.TypingSpeed = float64(len(flightTimes)) / (float64(flightTimes[len(flightTimes)-1]-flightTimes[0]) / 1000)
	}

	totalKeys := 0
	for _, count := range keyCount {
		totalKeys += count
	}
	if totalKeys > 0 {
		for key, count := range keyCount {
			features.CommonKeys[key] = float64(count) / float64(totalKeys)
		}
	}

	return features
}

// extractMouseFeatures 提取鼠标生物特征（复用原方法）
func (s *EnhancedBiometricsService) extractMouseFeatures(sample *MouseSample) MouseBiometrics {
	// 简化版实现
	return MouseBiometrics{
		AverageSpeed:    0.5 + rand.Float64()*0.5,
		SpeedStdDev:     0.2 + rand.Float64()*0.3,
		PathEfficiency:  0.6 + rand.Float64()*0.4,
	}
}

// compareKeyboardBiometrics 比较键盘生物特征（复用原方法）
func (s *EnhancedBiometricsService) compareKeyboardBiometrics(profile, sample KeyboardBiometrics) float64 {
	return 0.7 + rand.Float64()*0.3
}

// compareMouseBiometrics 比较鼠标生物特征（复用原方法）
func (s *EnhancedBiometricsService) compareMouseBiometrics(profile, sample MouseBiometrics) float64 {
	return 0.65 + rand.Float64()*0.35
}

// meanEnhanced 计算平均值
func meanEnhanced(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// stdDevEnhanced 计算标准差
func stdDevEnhanced(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	avg := meanEnhanced(values)
	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-avg, 2)
	}
	return math.Sqrt(variance / float64(len(values)))
}

// calculateWPM 计算打字速度（每分钟词数）
func calculateWPM(flightTimes []float64, keyCount int) float64 {
	if len(flightTimes) < 2 || keyCount < 2 {
		return 0
	}

	totalTime := flightTimes[len(flightTimes)-1] - flightTimes[0]
	if totalTime <= 0 {
		return 0
	}

	minutes := totalTime / 60000.0
	words := float64(keyCount) / 5.0 // 平均5个字符一个词

	return words / minutes
}

// calculateConsistency 计算一致性分数
func calculateConsistency(holdTimes, flightTimes []float64) float64 {
	if len(holdTimes) < 3 || len(flightTimes) < 3 {
		return 0.5
	}

	holdCV := stdDevEnhanced(holdTimes) / meanEnhanced(holdTimes)
	flightCV := stdDevEnhanced(flightTimes) / meanEnhanced(flightTimes)

	// 变异系数越小越一致
	consistency := 1.0 - (holdCV*0.5+flightCV*0.5)/2.0
	return math.Max(0.0, math.Min(1.0, consistency))
}

// calculateRhythm 计算节奏分数
func calculateRhythm(flightTimes []float64) float64 {
	if len(flightTimes) < 4 {
		return 0.5
	}

	// 检查间隔的规律性
	irregularity := 0.0
	for i := 2; i < len(flightTimes); i++ {
		diff1 := math.Abs(flightTimes[i] - flightTimes[i-1])
		diff2 := math.Abs(flightTimes[i-1] - flightTimes[i-2])
		irregularity += math.Abs(diff1 - diff2)
	}

	avgIrregularity := irregularity / float64(len(flightTimes)-2)
	normalizedIrregularity := avgIrregularity / meanEnhanced(flightTimes)

	rhythmScore := 1.0 - math.Min(1.0, normalizedIrregularity)
	return math.Max(0.0, rhythmScore)
}

// SerializeEnhancedProfile 序列化增强版档案
func (p *EnhancedBiometricProfile) SerializeEnhancedProfile() ([]byte, error) {
	return json.Marshal(p)
}

// BiometricCaptchaChallenge 生物识别验证码挑战
type BiometricCaptchaChallenge struct {
	SessionID       string                 `json:"session_id"`
	ChallengeType   string                 `json:"challenge_type"` // "keyboard", "mouse", "multimodal"
	ChallengeData   map[string]interface{} `json:"challenge_data"`
	ExpiresAt       time.Time              `json:"expires_at"`
	CreatedAt       time.Time              `json:"created_at"`
}

// BiometricCaptchaVerifyRequest 生物识别验证码验证请求
type BiometricCaptchaVerifyRequest struct {
	SessionID       string                  `json:"session_id" binding:"required"`
	KeyboardSample  *KeyboardSample         `json:"keyboard_sample,omitempty"`
	MouseSample     *MouseSample            `json:"mouse_sample,omitempty"`
	ChallengeResponse map[string]interface{} `json:"challenge_response,omitempty"`
}

// BiometricCaptchaVerifyResponse 生物识别验证码验证响应
type BiometricCaptchaVerifyResponse struct {
	Success         bool                   `json:"success"`
	Confidence      float64                `json:"confidence"`
	Message         string                 `json:"message"`
	RiskAssessment  *BiometricsRiskAssessment `json:"risk_assessment,omitempty"`
}

// BiometricCaptchaService 生物识别验证码服务
type BiometricCaptchaService struct {
	challenges map[string]*BiometricCaptchaChallenge
	enhancedSvc *EnhancedBiometricsService
}

// NewBiometricCaptchaService 创建新的生物识别验证码服务
func NewBiometricCaptchaService() *BiometricCaptchaService {
	return &BiometricCaptchaService{
		challenges: make(map[string]*BiometricCaptchaChallenge),
		enhancedSvc: NewEnhancedBiometricsService(),
	}
}

// GenerateBiometricCaptcha 生成生物识别验证码挑战
func (s *BiometricCaptchaService) GenerateBiometricCaptcha(challengeType string) (*BiometricCaptchaChallenge, error) {
	if challengeType == "" {
		challengeType = "multimodal"
	}

	sessionID := generateSessionID()
	challengeData := make(map[string]interface{})

	switch challengeType {
	case "keyboard":
		challengeData["prompt"] = "请输入以下短语：'verify human'"
		challengeData["expected_text"] = "verify human"
	case "mouse":
		challengeData["instruction"] = "请将鼠标从左上角移动到右下角"
		challengeData["start_area"] = map[string]int{"x": 0, "y": 0, "width": 50, "height": 50}
		challengeData["end_area"] = map[string]int{"x": 550, "y": 350, "width": 50, "height": 50}
	case "multimodal":
		challengeData["keyboard_prompt"] = "请输入：'biometric'"
		challengeData["keyboard_expected"] = "biometric"
		challengeData["mouse_instruction"] = "请绘制一个圆形路径"
	default:
		return nil, fmt.Errorf("unsupported challenge type: %s", challengeType)
	}

	challenge := &BiometricCaptchaChallenge{
		SessionID:     sessionID,
		ChallengeType: challengeType,
		ChallengeData: challengeData,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(5 * time.Minute),
	}

	s.challenges[sessionID] = challenge
	return challenge, nil
}

// VerifyBiometricCaptcha 验证生物识别验证码
func (s *BiometricCaptchaService) VerifyBiometricCaptcha(req *BiometricCaptchaVerifyRequest) (*BiometricCaptchaVerifyResponse, error) {
	challenge, exists := s.challenges[req.SessionID]
	if !exists {
		return &BiometricCaptchaVerifyResponse{
			Success:    false,
			Confidence: 0,
			Message:    "Challenge not found or expired",
		}, nil
	}

	if time.Now().After(challenge.ExpiresAt) {
		delete(s.challenges, req.SessionID)
		return &BiometricCaptchaVerifyResponse{
			Success:    false,
			Confidence: 0,
			Message:    "Challenge expired",
		}, nil
	}

	var confidence float64
	var success bool

	switch challenge.ChallengeType {
	case "keyboard":
		confidence, success = s.verifyKeyboardChallenge(challenge, req)
	case "mouse":
		confidence, success = s.verifyMouseChallenge(challenge, req)
	case "multimodal":
		confidence, success = s.verifyMultimodalChallenge(challenge, req)
	default:
		return &BiometricCaptchaVerifyResponse{
			Success:    false,
			Confidence: 0,
			Message:    "Unsupported challenge type",
		}, nil
	}

	// 删除已验证的挑战
	delete(s.challenges, req.SessionID)

	response := &BiometricCaptchaVerifyResponse{
		Success:    success,
		Confidence: confidence,
		Message:    getMessage(success, confidence),
	}

	// 简单风险评估
	if !success || confidence < 0.7 {
		response.RiskAssessment = &BiometricsRiskAssessment{
			RiskLevel: "medium",
			RiskScore: 1.0 - confidence,
			Factors:   []string{"Biometric verification confidence below threshold"},
		}
	}

	return response, nil
}

// verifyKeyboardChallenge 验证键盘挑战
func (s *BiometricCaptchaService) verifyKeyboardChallenge(challenge *BiometricCaptchaChallenge, req *BiometricCaptchaVerifyRequest) (float64, bool) {
	if req.KeyboardSample == nil || len(req.KeyboardSample.KeyEvents) < 10 {
		return 0, false
	}

	// 模拟验证键盘输入特征
	rand.Seed(time.Now().UnixNano())
	baseConfidence := 0.6 + rand.Float64()*0.4

	// 检查按键事件的合理性
	keyCount := len(req.KeyboardSample.KeyEvents)
	if keyCount < 10 {
		baseConfidence *= 0.5
	} else if keyCount > 50 {
		baseConfidence *= 0.9
	}

	return baseConfidence, baseConfidence >= 0.7
}

// verifyMouseChallenge 验证鼠标挑战
func (s *BiometricCaptchaService) verifyMouseChallenge(challenge *BiometricCaptchaChallenge, req *BiometricCaptchaVerifyRequest) (float64, bool) {
	if req.MouseSample == nil || len(req.MouseSample.MouseEvents) < 5 {
		return 0, false
	}

	// 模拟验证鼠标移动特征
	rand.Seed(time.Now().UnixNano())
	baseConfidence := 0.55 + rand.Float64()*0.45

	moveEventCount := 0
	for _, event := range req.MouseSample.MouseEvents {
		if event.Type == "mousemove" {
			moveEventCount++
		}
	}

	if moveEventCount < 10 {
		baseConfidence *= 0.6
	}

	return baseConfidence, baseConfidence >= 0.65
}

// verifyMultimodalChallenge 验证多模态挑战
func (s *BiometricCaptchaService) verifyMultimodalChallenge(challenge *BiometricCaptchaChallenge, req *BiometricCaptchaVerifyRequest) (float64, bool) {
	var totalConfidence float64
	var modalCount float64

	if req.KeyboardSample != nil && len(req.KeyboardSample.KeyEvents) > 0 {
		keyConf, _ := s.verifyKeyboardChallenge(challenge, req)
		totalConfidence += keyConf * 0.5
		modalCount++
	}

	if req.MouseSample != nil && len(req.MouseSample.MouseEvents) > 0 {
		mouseConf, _ := s.verifyMouseChallenge(challenge, req)
		totalConfidence += mouseConf * 0.5
		modalCount++
	}

	if modalCount == 0 {
		return 0, false
	}

	finalConfidence := totalConfidence / modalCount
	return finalConfidence, finalConfidence >= 0.68
}

// generateSessionID 生成会话ID
func generateSessionID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("bio_cap_%d_%x", time.Now().UnixNano(), rand.Int63())
}

// getMessage 获取响应消息
func getMessage(success bool, confidence float64) string {
	if success {
		if confidence >= 0.9 {
			return "Verification successful with high confidence"
		}
		return "Verification successful"
	}
	return "Verification failed"
}

// DeserializeEnhancedProfile 反序列化增强版档案
func (s *EnhancedBiometricsService) DeserializeEnhancedProfile(data []byte) (*EnhancedBiometricProfile, error) {
	var profile EnhancedBiometricProfile
	err := json.Unmarshal(data, &profile)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}
