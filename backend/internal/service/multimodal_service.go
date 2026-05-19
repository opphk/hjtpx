package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"
)

type ModalityType string

const (
	ModalityVisual      ModalityType = "visual"
	ModalityVoice       ModalityType = "voice"
	ModalityGesture     ModalityType = "gesture"
	ModalityTouch       ModalityType = "touch"
	ModalityAR          ModalityType = "ar"
	ModalityBiometric   ModalityType = "biometric"
)

type MultimodalConfig struct {
	EnabledModalities   []ModalityType `json:"enabled_modalities"`
	PrimaryModality     ModalityType   `json:"primary_modality"`
	FallbackModality    ModalityType   `json:"fallback_modality"`
	CrossDeviceEnabled  bool           `json:"cross_device_enabled"`
	ConfidenceThreshold float64        `json:"confidence_threshold"`
	TimeoutSeconds      int            `json:"timeout_seconds"`
}

type VoiceChallenge struct {
	ID           string   `json:"id"`
	Text         string   `json:"text"`
	AudioURL     string   `json:"audio_url"`
	Duration     int      `json:"duration"`
	Difficulty   int      `json:"difficulty"`
	RequiredPhrases []string `json:"required_phrases"`
}

type GestureChallenge struct {
	ID          string              `json:"id"`
	Type        string              `json:"type"`
	Description string              `json:"description"`
	Points      []GesturePoint      `json:"points"`
	Duration    int                 `json:"duration"`
	Difficulty  int                 `json:"difficulty"`
}

type GesturePoint struct {
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Pressure float64 `json:"pressure"`
	Timestamp int64  `json:"timestamp"`
}

type TouchChallenge struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	TargetArea   TouchArea     `json:"target_area"`
	TouchPattern []TouchPoint  `json:"touch_pattern"`
	Duration     int           `json:"duration"`
	Difficulty   int           `json:"difficulty"`
}

type TouchArea struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type TouchPoint struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Timestamp int64   `json:"timestamp"`
	Pressure  float64 `json:"pressure"`
	Fingers   int     `json:"fingers"`
}

type ARChallenge struct {
	ID          string     `json:"id"`
	SceneType   string     `json:"scene_type"`
	Objects     []ARObject `json:"objects"`
	Instructions string    `json:"instructions"`
	Difficulty  int        `json:"difficulty"`
}

type ARObject struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Position []float64   `json:"position"`
	Rotation []float64   `json:"rotation"`
	Scale    float64     `json:"scale"`
	Target   bool        `json:"target"`
}

type CrossDeviceSession struct {
	ID              string       `json:"id"`
	PrimaryDevice   string       `json:"primary_device"`
	SecondaryDevice string       `json:"secondary_device"`
	Modality        ModalityType `json:"modality"`
	Status          string       `json:"status"`
	CreatedAt       time.Time    `json:"created_at"`
	ExpiresAt       time.Time    `json:"expires_at"`
	Data            string       `json:"data"`
}

type MultimodalVerificationRequest struct {
	Modality      ModalityType          `json:"modality"`
	ChallengeID   string                `json:"challenge_id"`
	Response      interface{}           `json:"response"`
	DeviceInfo    string                `json:"device_info"`
	SessionToken  string                `json:"session_token"`
	Timestamp     int64                `json:"timestamp"`
}

type MultimodalVerificationResult struct {
	IsValid       bool                   `json:"is_valid"`
	Confidence    float64                `json:"confidence"`
	Score         float64                `json:"score"`
	Details       string                 `json:"details"`
	Modality      ModalityType           `json:"modality"`
	ProcessingTime int64                 `json:"processing_time"`
	Metrics       map[string]interface{} `json:"metrics"`
}

type MultimodalService struct {
	config     *MultimodalConfig
	challenges map[string]interface{}
	sessions   map[string]*CrossDeviceSession
	mu         sync.RWMutex
}

func NewMultimodalService() *MultimodalService {
	return &MultimodalService{
		config: &MultimodalConfig{
			EnabledModalities: []ModalityType{
				ModalityVisual,
				ModalityVoice,
				ModalityGesture,
				ModalityTouch,
				ModalityAR,
			},
			PrimaryModality:    ModalityVisual,
			FallbackModality:   ModalityVoice,
			CrossDeviceEnabled: true,
			ConfidenceThreshold: 0.85,
			TimeoutSeconds:      60,
		},
		challenges: make(map[string]interface{}),
		sessions:   make(map[string]*CrossDeviceSession),
	}
}

func (s *MultimodalService) GetConfig() *MultimodalConfig {
	return s.config
}

func (s *MultimodalService) UpdateConfig(config *MultimodalConfig) {
	s.config = config
}

func (s *MultimodalService) IsModalityEnabled(modality ModalityType) bool {
	for _, m := range s.config.EnabledModalities {
		if m == modality {
			return true
		}
	}
	return false
}

func (s *MultimodalService) GenerateVoiceChallenge(difficulty int) (*VoiceChallenge, error) {
	if !s.IsModalityEnabled(ModalityVoice) {
		return nil, fmt.Errorf("voice modality is not enabled")
	}

	challenge := &VoiceChallenge{
		ID:          fmt.Sprintf("voice_%d", time.Now().UnixNano()),
		Text:        s.generateVoiceText(difficulty),
		Duration:    s.calculateDuration(difficulty),
		Difficulty:  difficulty,
		RequiredPhrases: s.extractPhrases(difficulty),
	}

	s.challenges[challenge.ID] = challenge
	return challenge, nil
}

func (s *MultimodalService) generateVoiceText(difficulty int) string {
	phrases := []string{
		"请说出屏幕显示的验证码",
		"请朗读以下数字",
		"请按顺序说出这几个字",
		"请大声朗读这段文字",
	}

	texts := []string{
		"3847",
		"AB12",
		"验证",
		"安全",
		"通过",
		"请拖动滑块完成拼图",
	}

	basePhrase := phrases[0]
	baseText := texts[0]

	switch difficulty {
	case 1:
		return fmt.Sprintf("%s：%s", basePhrase, baseText[:2])
	case 2:
		return fmt.Sprintf("%s：%s", basePhrase, baseText[:3])
	case 3:
		return fmt.Sprintf("%s：%s %s", basePhrase, baseText[:2], baseText[2:])
	case 4:
		return fmt.Sprintf("%s：%s", basePhrase, baseText)
	default:
		return fmt.Sprintf("%s：%s", basePhrase, baseText[:2])
	}
}

func (s *MultimodalService) extractPhrases(difficulty int) []string {
	phrases := []string{"验证", "安全", "通过", "滑块"}
	count := 2
	switch difficulty {
	case 1:
		count = 1
	case 2:
		count = 2
	case 3:
		count = 3
	case 4:
		count = 4
	}
	if count > len(phrases) {
		count = len(phrases)
	}
	return phrases[:count]
}

func (s *MultimodalService) calculateDuration(difficulty int) int {
	baseDuration := 5
	switch difficulty {
	case 1:
		return baseDuration
	case 2:
		return baseDuration + 3
	case 3:
		return baseDuration + 6
	case 4:
		return baseDuration + 10
	default:
		return baseDuration
	}
}

func (s *MultimodalService) GenerateGestureChallenge(difficulty int) (*GestureChallenge, error) {
	if !s.IsModalityEnabled(ModalityGesture) {
		return nil, fmt.Errorf("gesture modality is not enabled")
	}

	gestureTypes := []string{"circle", "triangle", "square", "check", "wave"}
	gestureType := gestureTypes[0]

	switch difficulty {
	case 1:
		gestureType = "circle"
	case 2:
		gestureType = "square"
	case 3:
		gestureType = "triangle"
	case 4:
		gestureType = "check"
	default:
		gestureType = "circle"
	}

	challenge := &GestureChallenge{
		ID:          fmt.Sprintf("gesture_%d", time.Now().UnixNano()),
		Type:        gestureType,
		Description: s.getGestureDescription(gestureType),
		Points:      s.generateGesturePoints(gestureType, difficulty),
		Duration:    s.calculateDuration(difficulty),
		Difficulty:  difficulty,
	}

	s.challenges[challenge.ID] = challenge
	return challenge, nil
}

func (s *MultimodalService) getGestureDescription(gestureType string) string {
	descriptions := map[string]string{
		"circle":   "请在空中画一个圆圈",
		"triangle": "请在空中画一个三角形",
		"square":   "请在空中画一个正方形",
		"check":    "请在空中画一个对勾",
		"wave":     "请在空中挥动手掌",
	}
	if desc, ok := descriptions[gestureType]; ok {
		return desc
	}
	return "请按要求完成手势"
}

func (s *MultimodalService) generateGesturePoints(gestureType string, difficulty int) []GesturePoint {
	points := make([]GesturePoint, 0)
	pointCount := 20

	switch difficulty {
	case 1:
		pointCount = 15
	case 2:
		pointCount = 20
	case 3:
		pointCount = 30
	case 4:
		pointCount = 40
	}

	switch gestureType {
	case "circle":
		for i := 0; i < pointCount; i++ {
			angle := float64(i) / float64(pointCount) * 2 * math.Pi
			points = append(points, GesturePoint{
				X:        0.5 + 0.2*math.Cos(angle),
				Y:        0.5 + 0.2*math.Sin(angle),
				Pressure: 0.8,
				Timestamp: time.Now().UnixMilli() + int64(i*50),
			})
		}
	case "square":
		sidePoints := pointCount / 4
		for i := 0; i < pointCount; i++ {
			side := i / sidePoints
			pos := float64(i % sidePoints) / float64(sidePoints)
			var x, y float64
			switch side {
			case 0:
				x, y = 0.3+pos*0.4, 0.3
			case 1:
				x, y = 0.7, 0.3+pos*0.4
			case 2:
				x, y = 0.7-pos*0.4, 0.7
			case 3:
				x, y = 0.3, 0.7-pos*0.4
			}
			points = append(points, GesturePoint{
				X:        x,
				Y:        y,
				Pressure: 0.8,
				Timestamp: time.Now().UnixMilli() + int64(i*50),
			})
		}
	default:
		for i := 0; i < pointCount; i++ {
			points = append(points, GesturePoint{
				X:        0.5,
				Y:        0.5,
				Pressure: 0.8,
				Timestamp: time.Now().UnixMilli() + int64(i*50),
			})
		}
	}

	return points
}

func (s *MultimodalService) GenerateTouchChallenge(difficulty int) (*TouchChallenge, error) {
	if !s.IsModalityEnabled(ModalityTouch) {
		return nil, fmt.Errorf("touch modality is not enabled")
	}

	touchTypes := []string{"tap", "swipe", "multi_tap", "pattern"}
	touchType := touchTypes[0]

	switch difficulty {
	case 1:
		touchType = "tap"
	case 2:
		touchType = "swipe"
	case 3:
		touchType = "multi_tap"
	case 4:
		touchType = "pattern"
	default:
		touchType = "tap"
	}

	challenge := &TouchChallenge{
		ID:         fmt.Sprintf("touch_%d", time.Now().UnixNano()),
		Type:       touchType,
		TargetArea: s.generateTargetArea(difficulty),
		TouchPattern: s.generateTouchPattern(touchType, difficulty),
		Duration:   s.calculateDuration(difficulty),
		Difficulty: difficulty,
	}

	s.challenges[challenge.ID] = challenge
	return challenge, nil
}

func (s *MultimodalService) generateTargetArea(difficulty int) TouchArea {
	size := 100.0
	switch difficulty {
	case 1:
		size = 150.0
	case 2:
		size = 100.0
	case 3:
		size = 80.0
	case 4:
		size = 60.0
	}

	return TouchArea{
		X:      0.5 - size/2,
		Y:      0.5 - size/2,
		Width:  size,
		Height: size,
	}
}

func (s *MultimodalService) generateTouchPattern(touchType string, difficulty int) []TouchPoint {
	points := make([]TouchPoint, 0)
	pointCount := 5

	switch difficulty {
	case 1:
		pointCount = 3
	case 2:
		pointCount = 5
	case 3:
		pointCount = 7
	case 4:
		pointCount = 9
	}

	switch touchType {
	case "tap":
		for i := 0; i < pointCount; i++ {
			points = append(points, TouchPoint{
				X:         0.3 + float64(i)*0.1,
				Y:         0.5,
				Timestamp: time.Now().UnixMilli() + int64(i*500),
				Pressure:  0.8,
				Fingers:  1,
			})
		}
	case "swipe":
		for i := 0; i < pointCount; i++ {
			progress := float64(i) / float64(pointCount-1)
			points = append(points, TouchPoint{
				X:         0.2 + progress*0.6,
				Y:         0.5,
				Timestamp: time.Now().UnixMilli() + int64(i*100),
				Pressure:  0.8,
				Fingers:  1,
			})
		}
	case "multi_tap":
		for i := 0; i < pointCount; i++ {
			points = append(points, TouchPoint{
				X:         0.3 + float64(i%3)*0.2,
				Y:         0.4 + float64(i/3)*0.2,
				Timestamp: time.Now().UnixMilli() + int64(i*600),
				Pressure:  0.9,
				Fingers:  2,
			})
		}
	default:
		points = append(points, TouchPoint{
			X:         0.5,
			Y:         0.5,
			Timestamp: time.Now().UnixMilli(),
			Pressure: 0.8,
			Fingers:  1,
		})
	}

	return points
}

func (s *MultimodalService) GenerateARChallenge(difficulty int) (*ARChallenge, error) {
	if !s.IsModalityEnabled(ModalityAR) {
		return nil, fmt.Errorf("AR modality is not enabled")
	}

	sceneTypes := []string{"object_placement", "object_rotation", "object_scale", "object_arrangement"}
	sceneType := sceneTypes[0]

	switch difficulty {
	case 1:
		sceneType = "object_placement"
	case 2:
		sceneType = "object_rotation"
	case 3:
		sceneType = "object_scale"
	case 4:
		sceneType = "object_arrangement"
	default:
		sceneType = "object_placement"
	}

	objectCount := 1
	switch difficulty {
	case 1:
		objectCount = 1
	case 2:
		objectCount = 2
	case 3:
		objectCount = 3
	case 4:
		objectCount = 4
	}

	objects := make([]ARObject, objectCount)
	for i := 0; i < objectCount; i++ {
		objects[i] = ARObject{
			ID:       fmt.Sprintf("obj_%d", i),
			Type:     "cube",
			Position: []float64{float64(i) * 0.3, 0, -2},
			Rotation: []float64{0, 0, 0},
			Scale:    1.0,
			Target:   i == 0,
		}
	}

	challenge := &ARChallenge{
		ID:           fmt.Sprintf("ar_%d", time.Now().UnixNano()),
		SceneType:    sceneType,
		Objects:      objects,
		Instructions: s.getARInstructions(sceneType),
		Difficulty:   difficulty,
	}

	s.challenges[challenge.ID] = challenge
	return challenge, nil
}

func (s *MultimodalService) getARInstructions(sceneType string) string {
	instructions := map[string]string{
		"object_placement":    "请将虚拟物体放置到指定位置",
		"object_rotation":      "请旋转虚拟物体使其朝向正确",
		"object_scale":         "请缩放虚拟物体到指定大小",
		"object_arrangement":  "请按正确顺序排列虚拟物体",
	}
	if inst, ok := instructions[sceneType]; ok {
		return inst
	}
	return "请按要求完成AR操作"
}

func (s *MultimodalService) CreateCrossDeviceSession(primaryDevice, secondaryDevice string, modality ModalityType) (*CrossDeviceSession, error) {
	if !s.config.CrossDeviceEnabled {
		return nil, fmt.Errorf("cross-device verification is not enabled")
	}

	session := &CrossDeviceSession{
		ID:              fmt.Sprintf("cross_%d", time.Now().UnixNano()),
		PrimaryDevice:   primaryDevice,
		SecondaryDevice: secondaryDevice,
		Modality:        modality,
		Status:          "pending",
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(time.Duration(s.config.TimeoutSeconds) * time.Second),
	}

	s.sessions[session.ID] = session
	return session, nil
}

func (s *MultimodalService) GetCrossDeviceSession(sessionID string) (*CrossDeviceSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("cross-device session not found: %s", sessionID)
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("cross-device session has expired")
	}

	return session, nil
}

func (s *MultimodalService) UpdateCrossDeviceSession(sessionID string, status string, data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("cross-device session not found: %s", sessionID)
	}

	session.Status = status
	session.Data = data

	return nil
}

func (s *MultimodalService) VerifyVoiceResponse(challengeID string, audioData string) (*MultimodalVerificationResult, error) {
	startTime := time.Now()

	challenge, exists := s.challenges[challengeID]
	if !exists {
		return &MultimodalVerificationResult{
			IsValid:       false,
			Confidence:    0,
			Details:       "challenge not found",
			Modality:      ModalityVoice,
			ProcessingTime: time.Since(startTime).Milliseconds(),
		}, nil
	}

	voiceChallenge, ok := challenge.(*VoiceChallenge)
	if !ok {
		return nil, fmt.Errorf("invalid challenge type")
	}

	score := s.calculateVoiceScore(voiceChallenge, audioData)
	confidence := score

	result := &MultimodalVerificationResult{
		IsValid:       confidence >= s.config.ConfidenceThreshold,
		Confidence:    confidence,
		Score:         score,
		Details:       fmt.Sprintf("voice verification score: %.2f", score),
		Modality:      ModalityVoice,
		ProcessingTime: time.Since(startTime).Milliseconds(),
		Metrics: map[string]interface{}{
			"challenge_id":    challengeID,
			"difficulty":      voiceChallenge.Difficulty,
			"required_phrases": voiceChallenge.RequiredPhrases,
		},
	}

	return result, nil
}

func (s *MultimodalService) calculateVoiceScore(challenge *VoiceChallenge, audioData string) float64 {
	baseScore := 0.7

	baseScore += float64(len(challenge.RequiredPhrases)) * 0.05

	switch challenge.Difficulty {
	case 1:
		baseScore += 0.15
	case 2:
		baseScore += 0.10
	case 3:
		baseScore += 0.05
	case 4:
		baseScore += 0.0
	}

	if len(audioData) > 0 {
		baseScore += 0.1
	}

	return math.Min(1.0, baseScore)
}

func (s *MultimodalService) VerifyGestureResponse(challengeID string, gesturePoints []GesturePoint) (*MultimodalVerificationResult, error) {
	startTime := time.Now()

	challenge, exists := s.challenges[challengeID]
	if !exists {
		return &MultimodalVerificationResult{
			IsValid:       false,
			Confidence:    0,
			Details:       "challenge not found",
			Modality:      ModalityGesture,
			ProcessingTime: time.Since(startTime).Milliseconds(),
		}, nil
	}

	gestureChallenge, ok := challenge.(*GestureChallenge)
	if !ok {
		return nil, fmt.Errorf("invalid challenge type")
	}

	score := s.calculateGestureScore(gestureChallenge, gesturePoints)
	confidence := score

	result := &MultimodalVerificationResult{
		IsValid:       confidence >= s.config.ConfidenceThreshold,
		Confidence:    confidence,
		Score:         score,
		Details:       fmt.Sprintf("gesture verification score: %.2f", score),
		Modality:      ModalityGesture,
		ProcessingTime: time.Since(startTime).Milliseconds(),
		Metrics: map[string]interface{}{
			"challenge_id": challengeID,
			"gesture_type": gestureChallenge.Type,
			"difficulty":   gestureChallenge.Difficulty,
		},
	}

	return result, nil
}

func (s *MultimodalService) calculateGestureScore(challenge *GestureChallenge, userPoints []GesturePoint) float64 {
	if len(userPoints) == 0 || len(challenge.Points) == 0 {
		return 0.0
	}

	typeScore := 0.0
	switch challenge.Type {
	case "circle":
		typeScore = s.evaluateCircleGesture(userPoints, challenge.Points)
	case "square":
		typeScore = s.evaluateSquareGesture(userPoints, challenge.Points)
	default:
		typeScore = s.evaluateGenericGesture(userPoints, challenge.Points)
	}

	difficultyBonus := float64(challenge.Difficulty) * 0.02

	return math.Min(1.0, typeScore+difficultyBonus)
}

func (s *MultimodalService) evaluateCircleGesture(userPoints, targetPoints []GesturePoint) float64 {
	if len(userPoints) < 10 {
		return 0.3
	}

	centroidX, centroidY := 0.0, 0.0
	for _, p := range userPoints {
		centroidX += p.X
		centroidY += p.Y
	}
	centroidX /= float64(len(userPoints))
	centroidY /= float64(len(userPoints))

	avgRadius := 0.0
	for _, p := range userPoints {
		dx := p.X - centroidX
		dy := p.Y - centroidY
		avgRadius += math.Sqrt(dx*dx + dy*dy)
	}
	avgRadius /= float64(len(userPoints))

	radiusVariance := 0.0
	for _, p := range userPoints {
		dx := p.X - centroidX
		dy := p.Y - centroidY
		r := math.Sqrt(dx*dx + dy*dy)
		radiusVariance += (r - avgRadius) * (r - avgRadius)
	}
	radiusVariance /= float64(len(userPoints))

	varianceScore := math.Max(0, 1.0-radiusVariance*10)

	closureScore := 0.0
	if len(userPoints) > 1 {
		first := userPoints[0]
		last := userPoints[len(userPoints)-1]
		dx := first.X - last.X
		dy := first.Y - last.Y
		closure := math.Sqrt(dx*dx + dy*dy)
		closureScore = math.Max(0, 1.0-closure*5)
	}

	return (varianceScore*0.6 + closureScore*0.4) * 0.9
}

func (s *MultimodalService) evaluateSquareGesture(userPoints, targetPoints []GesturePoint) float64 {
	if len(userPoints) < 8 {
		return 0.3
	}

	segments := s.identifySegments(userPoints)
	segmentScore := math.Min(1.0, float64(len(segments))/4.0)

	straightnessScore := 0.0
	for _, segment := range segments {
		straightnessScore += s.calculateStraightness(segment)
	}
	straightnessScore /= float64(len(segments))

	return (segmentScore*0.6 + straightnessScore*0.4) * 0.9
}

func (s *MultimodalService) identifySegments(points []GesturePoint) [][]GesturePoint {
	segments := make([][]GesturePoint, 0)
	currentSegment := make([]GesturePoint, 0)
	threshold := 0.05

	for i := 0; i < len(points); i++ {
		if len(currentSegment) == 0 {
			currentSegment = append(currentSegment, points[i])
			continue
		}

		prev := currentSegment[len(currentSegment)-1]
		curr := points[i]
		dx := curr.X - prev.X
		dy := curr.Y - prev.Y
		direction := math.Atan2(dy, dx)

		lastDir := 0.0
		if len(currentSegment) > 1 {
			p1 := currentSegment[len(currentSegment)-2]
			p2 := currentSegment[len(currentSegment)-1]
			lastDir = math.Atan2(p2.Y-p1.Y, p2.X-p1.X)
		}

		dirChange := math.Abs(direction - lastDir)
		if dirChange > threshold {
			segments = append(segments, currentSegment)
			currentSegment = []GesturePoint{curr}
		} else {
			currentSegment = append(currentSegment, curr)
		}
	}

	if len(currentSegment) > 0 {
		segments = append(segments, currentSegment)
	}

	return segments
}

func (s *MultimodalService) calculateStraightness(points []GesturePoint) float64 {
	if len(points) < 2 {
		return 0.0
	}

	first := points[0]
	last := points[len(points)-1]
	directDist := math.Sqrt(math.Pow(last.X-first.X, 2) + math.Pow(last.Y-first.Y, 2))

	totalDist := 0.0
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		totalDist += math.Sqrt(dx*dx + dy*dy)
	}

	if totalDist == 0 {
		return 0.0
	}

	return directDist / totalDist
}

func (s *MultimodalService) evaluateGenericGesture(userPoints, targetPoints []GesturePoint) float64 {
	if len(userPoints) == 0 {
		return 0.0
	}

	avgDist := 0.0
	for _, up := range userPoints {
		minDist := math.MaxFloat64
		for _, tp := range targetPoints {
			dx := up.X - tp.X
			dy := up.Y - tp.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < minDist {
				minDist = dist
			}
		}
		avgDist += minDist
	}
	avgDist /= float64(len(userPoints))

	return math.Max(0, 1.0-avgDist*5) * 0.8
}

func (s *MultimodalService) VerifyTouchResponse(challengeID string, touchPoints []TouchPoint) (*MultimodalVerificationResult, error) {
	startTime := time.Now()

	challenge, exists := s.challenges[challengeID]
	if !exists {
		return &MultimodalVerificationResult{
			IsValid:       false,
			Confidence:    0,
			Details:       "challenge not found",
			Modality:      ModalityTouch,
			ProcessingTime: time.Since(startTime).Milliseconds(),
		}, nil
	}

	touchChallenge, ok := challenge.(*TouchChallenge)
	if !ok {
		return nil, fmt.Errorf("invalid challenge type")
	}

	score := s.calculateTouchScore(touchChallenge, touchPoints)
	confidence := score

	result := &MultimodalVerificationResult{
		IsValid:       confidence >= s.config.ConfidenceThreshold,
		Confidence:    confidence,
		Score:         score,
		Details:       fmt.Sprintf("touch verification score: %.2f", score),
		Modality:      ModalityTouch,
		ProcessingTime: time.Since(startTime).Milliseconds(),
		Metrics: map[string]interface{}{
			"challenge_id": challengeID,
			"touch_type":   touchChallenge.Type,
			"difficulty":   touchChallenge.Difficulty,
		},
	}

	return result, nil
}

func (s *MultimodalService) calculateTouchScore(challenge *TouchChallenge, userPoints []TouchPoint) float64 {
	if len(userPoints) == 0 || len(challenge.TouchPattern) == 0 {
		return 0.0
	}

	accuracyScore := s.calculateTouchAccuracy(challenge, userPoints)

	durationScore := 0.0
	expectedDuration := int64(challenge.Duration * 1000)
	if len(userPoints) > 1 {
		actualDuration := userPoints[len(userPoints)-1].Timestamp - userPoints[0].Timestamp
		durationDiff := math.Abs(float64(actualDuration) - float64(expectedDuration))
		durationScore = math.Max(0, 1.0-durationDiff/float64(expectedDuration))
	}

	patternScore := s.evaluateTouchPattern(challenge.Type, challenge.TouchPattern, userPoints)

	return (accuracyScore*0.5 + durationScore*0.2 + patternScore*0.3) * 0.95
}

func (s *MultimodalService) calculateTouchAccuracy(challenge *TouchChallenge, userPoints []TouchPoint) float64 {
	hits := 0
	for _, up := range userPoints {
		if up.X >= challenge.TargetArea.X &&
			up.X <= challenge.TargetArea.X+challenge.TargetArea.Width &&
			up.Y >= challenge.TargetArea.Y &&
			up.Y <= challenge.TargetArea.Y+challenge.TargetArea.Height {
			hits++
		}
	}

	return float64(hits) / float64(len(userPoints))
}

func (s *MultimodalService) evaluateTouchPattern(patternType string, expected, actual []TouchPoint) float64 {
	if len(actual) == 0 || len(expected) == 0 {
		return 0.0
	}

	avgDist := 0.0
	matchCount := 0
	for i, ep := range expected {
		if i >= len(actual) {
			break
		}
		ap := actual[i]
		dx := ap.X - ep.X
		dy := ap.Y - ep.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		avgDist += dist
		if dist < 0.1 {
			matchCount++
		}
	}

	avgDist /= float64(len(expected))
	accuracyScore := math.Max(0, 1.0-avgDist*10)
	matchScore := float64(matchCount) / float64(len(expected))

	return accuracyScore*0.6 + matchScore*0.4
}

func (s *MultimodalService) VerifyARResponse(challengeID string, objectPositions []ARObject) (*MultimodalVerificationResult, error) {
	startTime := time.Now()

	challenge, exists := s.challenges[challengeID]
	if !exists {
		return &MultimodalVerificationResult{
			IsValid:       false,
			Confidence:    0,
			Details:       "challenge not found",
			Modality:      ModalityAR,
			ProcessingTime: time.Since(startTime).Milliseconds(),
		}, nil
	}

	arChallenge, ok := challenge.(*ARChallenge)
	if !ok {
		return nil, fmt.Errorf("invalid challenge type")
	}

	score := s.calculateARScore(arChallenge, objectPositions)
	confidence := score

	result := &MultimodalVerificationResult{
		IsValid:       confidence >= s.config.ConfidenceThreshold,
		Confidence:    confidence,
		Score:         score,
		Details:       fmt.Sprintf("AR verification score: %.2f", score),
		Modality:      ModalityAR,
		ProcessingTime: time.Since(startTime).Milliseconds(),
		Metrics: map[string]interface{}{
			"challenge_id": challengeID,
			"scene_type":   arChallenge.SceneType,
			"difficulty":   arChallenge.Difficulty,
		},
	}

	return result, nil
}

func (s *MultimodalService) calculateARScore(challenge *ARChallenge, userObjects []ARObject) float64 {
	if len(userObjects) == 0 || len(challenge.Objects) == 0 {
		return 0.0
	}

	positionScore := 0.0
	rotationScore := 0.0
	scaleScore := 0.0

	for i, co := range challenge.Objects {
		if i >= len(userObjects) {
			break
		}
		uo := userObjects[i]

		posDist := math.Sqrt(
			math.Pow(uo.Position[0]-co.Position[0], 2) +
				math.Pow(uo.Position[1]-co.Position[1], 2) +
				math.Pow(uo.Position[2]-co.Position[2], 2),
		)
		positionScore += math.Max(0, 1.0-posDist)

		rotDiff := 0.0
		for j := 0; j < 3; j++ {
			if j < len(uo.Rotation) && j < len(co.Rotation) {
				rotDiff += math.Abs(uo.Rotation[j] - co.Rotation[j])
			}
		}
		rotationScore += math.Max(0, 1.0-rotDiff/(2*math.Pi))

		scaleDiff := math.Abs(uo.Scale - co.Scale)
		scaleScore += math.Max(0, 1.0-scaleDiff)
	}

	count := float64(len(challenge.Objects))
	positionScore /= count
	rotationScore /= count
	scaleScore /= count

	return (positionScore*0.5 + rotationScore*0.3 + scaleScore*0.2) * 0.95
}

func (s *MultimodalService) Verify(request *MultimodalVerificationRequest) (*MultimodalVerificationResult, error) {
	switch request.Modality {
	case ModalityVoice:
		var audioData string
		if data, ok := request.Response.(string); ok {
			audioData = data
		}
		return s.VerifyVoiceResponse(request.ChallengeID, audioData)

	case ModalityGesture:
		var gesturePoints []GesturePoint
		if data, ok := request.Response.([]interface{}); ok {
			for _, item := range data {
				if m, ok := item.(map[string]interface{}); ok {
					gesturePoints = append(gesturePoints, GesturePoint{
						X:        toFloat64(m["x"]),
						Y:        toFloat64(m["y"]),
						Pressure: toFloat64(m["pressure"]),
						Timestamp: toInt64(m["timestamp"]),
					})
				}
			}
		}
		return s.VerifyGestureResponse(request.ChallengeID, gesturePoints)

	case ModalityTouch:
		var touchPoints []TouchPoint
		if data, ok := request.Response.([]interface{}); ok {
			for _, item := range data {
				if m, ok := item.(map[string]interface{}); ok {
					touchPoints = append(touchPoints, TouchPoint{
						X:         toFloat64(m["x"]),
						Y:         toFloat64(m["y"]),
						Pressure:  toFloat64(m["pressure"]),
						Timestamp: toInt64(m["timestamp"]),
						Fingers:   toInt(m["fingers"]),
					})
				}
			}
		}
		return s.VerifyTouchResponse(request.ChallengeID, touchPoints)

	case ModalityAR:
		var arObjects []ARObject
		if data, ok := request.Response.([]interface{}); ok {
			for _, item := range data {
				if m, ok := item.(map[string]interface{}); ok {
					obj := ARObject{
						ID:       toString(m["id"]),
						Type:     toString(m["type"]),
						Position: toFloat64Array(m["position"]),
						Rotation: toFloat64Array(m["rotation"]),
						Scale:    toFloat64(m["scale"]),
						Target:   toBool(m["target"]),
					}
					arObjects = append(arObjects, obj)
				}
			}
		}
		return s.VerifyARResponse(request.ChallengeID, arObjects)

	default:
		return nil, fmt.Errorf("unsupported modality: %s", request.Modality)
	}
}

func toFloat64(v interface{}) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	if i, ok := v.(int); ok {
		return float64(i)
	}
	if i, ok := v.(int64); ok {
		return float64(i)
	}
	return 0
}

func toInt64(v interface{}) int64 {
	if i, ok := v.(int64); ok {
		return i
	}
	if i, ok := v.(int); ok {
		return int64(i)
	}
	if f, ok := v.(float64); ok {
		return int64(f)
	}
	return 0
}

func toInt(v interface{}) int {
	if i, ok := v.(int); ok {
		return i
	}
	if i, ok := v.(int64); ok {
		return int(i)
	}
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toBool(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func toFloat64Array(v interface{}) []float64 {
	if arr, ok := v.([]interface{}); ok {
		result := make([]float64, len(arr))
		for i, item := range arr {
			result[i] = toFloat64(item)
		}
		return result
	}
	return []float64{}
}

func (s *MultimodalService) ExportConfig() ([]byte, error) {
	return json.Marshal(s.config)
}
