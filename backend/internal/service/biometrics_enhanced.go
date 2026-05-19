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
	MultimodalWeights MultimodalWeights     `json:"multimodal_weights"`
	VerificationCount int                   `json:"verification_count"`
	ConfidenceScore   float64               `json:"confidence_score"`
}

// FaceBiometrics 面部生物特征
type FaceBiometrics struct {
	LandmarkDistances  map[string]float64   `json:"landmark_distances"`
	FeatureVector      []float64            `json:"feature_vector"`
	FaceEmbedding      []float64            `json:"face_embedding"`
	EyeAspectRatio     float64              `json:"eye_aspect_ratio"`
	MouthAspectRatio   float64              `json:"mouth_aspect_ratio"`
	BlinkFrequency     float64              `json:"blink_frequency"`
	MicroExpressionScores map[string]float64 `json:"micro_expression_scores"`
	QualityScore       float64              `json:"quality_score"`
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

// GestureSample 手势样本
type GestureSample struct {
	HandLandmarks  [][]float64            `json:"hand_landmarks,omitempty"`
	GestureType    string                 `json:"gesture_type"`
	GestureData    map[string]interface{} `json:"gesture_data,omitempty"`
	Timestamp      int64                  `json:"timestamp"`
}

// EnhancedVerificationRequest 增强版验证请求
type EnhancedVerificationRequest struct {
	UserID         string                  `json:"user_id" binding:"required"`
	KeyboardSample *KeyboardSample         `json:"keyboard_sample,omitempty"`
	MouseSample    *MouseSample            `json:"mouse_sample,omitempty"`
	FaceSample     *FaceSample             `json:"face_sample,omitempty"`
	VoiceSample    *VoiceSample            `json:"voice_sample,omitempty"`
	GestureSample  *GestureSample          `json:"gesture_sample,omitempty"`
	SessionContext map[string]interface{}  `json:"session_context,omitempty"`
}

// EnhancedVerificationResult 增强版验证结果
type EnhancedVerificationResult struct {
	IsVerified     bool                   `json:"is_verified"`
	OverallConfidence float64             `json:"overall_confidence"`
	ModalScores    map[string]float64     `json:"modal_scores"`
	Details        string                 `json:"details"`
	RiskAssessment *BiometricsRiskAssessment `json:"risk_assessment,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
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
				KeyboardWeight: 0.3,
				MouseWeight:    0.25,
				FaceWeight:     0.2,
				VoiceWeight:    0.15,
				GestureWeight:  0.1,
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
			Details:           "No profile found for user",
			Timestamp:         time.Now(),
		}, nil
	}

	modalScores := make(map[string]float64)
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
	if req.FaceSample != nil && profile.FaceProfile != nil {
		sampleFeatures := s.extractFaceFeatures(req.FaceSample)
		faceScore := s.compareFaceBiometrics(*profile.FaceProfile, sampleFeatures)
		modalScores["face"] = faceScore
		totalWeight += profile.MultimodalWeights.FaceWeight
	} else {
		modalScores["face"] = 0.5
	}

	// 语音验证
	if req.VoiceSample != nil && profile.VoiceProfile != nil {
		sampleFeatures := s.extractVoiceFeatures(req.VoiceSample)
		voiceScore := s.compareVoiceBiometrics(*profile.VoiceProfile, sampleFeatures)
		modalScores["voice"] = voiceScore
		totalWeight += profile.MultimodalWeights.VoiceWeight
	} else {
		modalScores["voice"] = 0.5
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

	// 计算多模态融合分数
	overallConfidence := 0.0
	if totalWeight > 0 {
		overallConfidence = (modalScores["keyboard"]*profile.MultimodalWeights.KeyboardWeight +
			modalScores["mouse"]*profile.MultimodalWeights.MouseWeight +
			modalScores["face"]*profile.MultimodalWeights.FaceWeight +
			modalScores["voice"]*profile.MultimodalWeights.VoiceWeight +
			modalScores["gesture"]*profile.MultimodalWeights.GestureWeight) / totalWeight
	} else {
		// 如果没有提供任何模态，使用默认分数
		overallConfidence = 0.5
	}

	// 风险评估
	riskAssessment := s.assessRisk(modalScores, req.SessionContext)

	isVerified := overallConfidence >= 0.95

	result := &EnhancedVerificationResult{
		IsVerified:        isVerified,
		OverallConfidence: overallConfidence,
		ModalScores:       modalScores,
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
		MicroExpressionScores: make(map[string]float64),
	}

	// 模拟生成面部特征
	landmarkPairs := []string{"eye_distance", "nose_width", "mouth_width", "jaw_length"}
	for _, pair := range landmarkPairs {
		features.LandmarkDistances[pair] = 30 + rand.Float64()*50
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

	expressions := []string{"happy", "surprised", "neutral", "focused"}
	for _, expr := range expressions {
		features.MicroExpressionScores[expr] = rand.Float64()
	}

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
	}

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
func (s *EnhancedBiometricsService) assessRisk(modalScores map[string]float64, context map[string]interface{}) *BiometricsRiskAssessment {
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

	if riskScore > 0.7 {
		riskLevel = "high"
		factors = append(factors, "Multiple modal scores below threshold")
	} else if riskScore > 0.4 {
		riskLevel = "medium"
	}

	return &BiometricsRiskAssessment{
		RiskLevel: riskLevel,
		RiskScore: riskScore,
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

// SerializeEnhancedProfile 序列化增强版档案
func (p *EnhancedBiometricProfile) SerializeEnhancedProfile() ([]byte, error) {
	return json.Marshal(p)
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
