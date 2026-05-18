package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type ARGestureType string

const (
	ARGestureRotateX    ARGestureType = "rotate_x"
	ARGestureRotateY    ARGestureType = "rotate_y"
	ARGestureRotateZ    ARGestureType = "rotate_z"
	ARGesturePinch      ARGestureType = "pinch"
	ARGestureSwipe      ARGestureType = "swipe"
)

type ARObject struct {
	ID           string           `json:"id"`
	Type         string           `json:"type"`
	Color        string           `json:"color"`
	InitialRotX  float64          `json:"initial_rot_x"`
	InitialRotY  float64          `json:"initial_rot_y"`
	InitialRotZ  float64          `json:"initial_rot_z"`
	TargetRotX   float64          `json:"target_rot_x"`
	TargetRotY   float64          `json:"target_rot_y"`
	TargetRotZ   float64          `json:"target_rot_z"`
	Scale        float64          `json:"scale"`
	PositionX    float64          `json:"position_x"`
	PositionY    float64          `json:"position_y"`
	PositionZ    float64          `json:"position_z"`
	Vertices     [][]float64      `json:"vertices"`
	Faces        [][]int          `json:"faces"`
}

type ARCaptchaPuzzle struct {
	Object        *ARObject      `json:"object"`
	GestureType   ARGestureType  `json:"gesture_type"`
	TargetAngle   float64        `json:"target_angle"`
	Tolerance     float64        `json:"tolerance"`
	Difficulty    string         `json:"difficulty"`
	Instructions  string         `json:"instructions"`
	ARFeatures    []string       `json:"ar_features"`
	WebXRRequired bool           `json:"webxr_required"`
}

type CreateARRequest struct {
	Difficulty  string `json:"difficulty"`
	ObjectType  string `json:"object_type"`
	GestureType string `json:"gesture_type"`
	ClientIP    string `json:"client_ip"`
	UserAgent   string `json:"user_agent"`
	Fingerprint string `json:"fingerprint"`
}

type CreateARResponse struct {
	SessionID     string          `json:"session_id"`
	Puzzle        *ARCaptchaPuzzle `json:"puzzle"`
	ModelDataURL  string          `json:"model_data_url"`
	ExpiresIn     int64           `json:"expires_in"`
	ExpiresAt     int64           `json:"expires_at"`
}

type VerifyARRequest struct {
	SessionID      string            `json:"session_id" binding:"required"`
	RotationX      float64           `json:"rotation_x"`
	RotationY      float64           `json:"rotation_y"`
	RotationZ      float64           `json:"rotation_z"`
	Scale          float64           `json:"scale"`
	GestureData    []ARGesturePoint  `json:"gesture_data"`
	TouchPoints    []TouchPoint      `json:"touch_points"`
	DeviceMotion   *DeviceMotionData `json:"device_motion"`
	RiskScore      float64           `json:"risk_score"`
}

type ARGesturePoint struct {
	Timestamp    int64   `json:"timestamp"`
	RotationX    float64 `json:"rotation_x"`
	RotationY    float64 `json:"rotation_y"`
	RotationZ    float64 `json:"rotation_z"`
	Scale        float64 `json:"scale"`
	GestureType  string  `json:"gesture_type"`
}

type TouchPoint struct {
	X         int   `json:"x"`
	Y         int   `json:"y"`
	Pressure  float64 `json:"pressure"`
	Timestamp int64  `json:"timestamp"`
}

type DeviceMotionData struct {
	AccelerationX float64 `json:"acceleration_x"`
	AccelerationY float64 `json:"acceleration_y"`
	AccelerationZ float64 `json:"acceleration_z"`
	RotationAlpha float64 `json:"rotation_alpha"`
	RotationBeta  float64 `json:"rotation_beta"`
	RotationGamma float64 `json:"rotation_gamma"`
	Timestamp     int64   `json:"timestamp"`
}

type VerifyARResult struct {
	Success          bool    `json:"success"`
	Message          string  `json:"message"`
	Score            float64 `json:"score"`
	Accuracy         float64 `json:"accuracy"`
	GestureScore     float64 `json:"gesture_score"`
	DeviceScore      float64 `json:"device_score"`
	NaturalnessScore float64 `json:"naturalness_score"`
}

type ARGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type ARVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

var arObjectTypes = []string{
	"cube", "sphere", "pyramid", "cylinder", "torus",
	"cone", "icosahedron", "octahedron", "star", "heart",
}

var arColors = []string{
	"#E74C3C", "#3498DB", "#2ECC71", "#F39C12", "#9B59B6",
	"#1ABC9C", "#E91E63", "#00BCD4", "#8BC34A", "#FF9800",
}

func NewARGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *ARGeneratorService {
	return &ARGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func NewARVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *ARVerifierService {
	return &ARVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (s *ARGeneratorService) Create(ctx context.Context, req *CreateARRequest) (*CreateARResponse, error) {
	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = "medium"
	}

	objectType := req.ObjectType
	if objectType == "" {
		objectType = arObjectTypes[rand.Intn(len(arObjectTypes))]
	}

	gestureType := ARGestureType(req.GestureType)
	if gestureType == "" {
		gestures := []ARGestureType{ARGestureRotateX, ARGestureRotateY, ARGestureRotateZ}
		gestureType = gestures[rand.Intn(len(gestures))]
	}

	puzzle := s.generateARPuzzle(objectType, gestureType, difficulty)

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	puzzleData, err := json.Marshal(puzzle)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal puzzle: %w", err)
	}

	session := &models.CaptchaSession{
		SessionID:     sessionID,
		BackgroundURL: string(puzzleData),
		SliderURL:     string(puzzleData),
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
		CreatedAt:     time.Now(),
		ExpiredAt:     expiresAt,
		ClientIP:      req.ClientIP,
		UserAgent:     req.UserAgent,
		Fingerprint:   req.Fingerprint,
	}

	if s.sessionCache != nil {
		if err := s.sessionCache.Set(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to cache session: %w", err)
		}
	}

	if s.captchaRepo != nil {
		if err := s.captchaRepo.Create(session); err != nil {
			return nil, fmt.Errorf("failed to save session to database: %w", err)
		}
	}

	modelDataURL := s.generateModelData(puzzle)

	return &CreateARResponse{
		SessionID:    sessionID,
		Puzzle:       puzzle,
		ModelDataURL: modelDataURL,
		ExpiresIn:    int64(5 * time.Minute / time.Second),
		ExpiresAt:    expiresAt.Unix(),
	}, nil
}

func (s *ARGeneratorService) generateARPuzzle(objectType string, gestureType ARGestureType, difficulty string) *ARCaptchaPuzzle {
	rand.Seed(time.Now().UnixNano())

	color := arColors[rand.Intn(len(arColors))]

	initialRotX := rand.Float64() * 360
	initialRotY := rand.Float64() * 360
	initialRotZ := rand.Float64() * 360

	targetAngle := s.getTargetAngle(difficulty, gestureType)
	tolerance := s.getTolerance(difficulty)

	var targetRotX, targetRotY, targetRotZ float64

	switch gestureType {
	case ARGestureRotateX:
		targetRotX = math.Mod(initialRotX+targetAngle, 360)
		targetRotY = initialRotY
		targetRotZ = initialRotZ
	case ARGestureRotateY:
		targetRotX = initialRotX
		targetRotY = math.Mod(initialRotY+targetAngle, 360)
		targetRotZ = initialRotZ
	case ARGestureRotateZ:
		targetRotX = initialRotX
		targetRotY = initialRotY
		targetRotZ = math.Mod(initialRotZ+targetAngle, 360)
	default:
		targetRotX = math.Mod(initialRotX+targetAngle, 360)
		targetRotY = initialRotY
		targetRotZ = initialRotZ
	}

	obj := &ARObject{
		ID:          fmt.Sprintf("ar_obj_%d", rand.Intn(10000)),
		Type:        objectType,
		Color:       color,
		InitialRotX: initialRotX,
		InitialRotY: initialRotY,
		InitialRotZ: initialRotZ,
		TargetRotX:  targetRotX,
		TargetRotY:  targetRotY,
		TargetRotZ:  targetRotZ,
		Scale:       1.0,
		PositionX:   0,
		PositionY:   0,
		PositionZ:   0,
		Vertices:    s.generateObjectVertices(objectType),
		Faces:       s.generateObjectFaces(objectType),
	}

	instructions := s.generateInstructions(gestureType, targetAngle)

	return &ARCaptchaPuzzle{
		Object:        obj,
		GestureType:   gestureType,
		TargetAngle:   targetAngle,
		Tolerance:     tolerance,
		Difficulty:    difficulty,
		Instructions:  instructions,
		ARFeatures:    []string{"gesture_tracking", "3d_rendering", "device_motion"},
		WebXRRequired: true,
	}
}

func (s *ARGeneratorService) getTargetAngle(difficulty string, gestureType ARGestureType) float64 {
	var baseAngle float64
	switch difficulty {
	case "easy":
		baseAngle = 45
	case "medium":
		baseAngle = 90
	case "hard":
		baseAngle = 135
	case "expert":
		baseAngle = 180
	default:
		baseAngle = 90
	}

	variation := rand.Float64() * 30
	return baseAngle + variation
}

func (s *ARGeneratorService) getTolerance(difficulty string) float64 {
	switch difficulty {
	case "easy":
		return 15
	case "medium":
		return 10
	case "hard":
		return 7
	case "expert":
		return 5
	default:
		return 10
	}
}

func (s *ARGeneratorService) generateObjectVertices(objectType string) [][]float64 {
	rand.Seed(time.Now().UnixNano())

	switch objectType {
	case "cube":
		return [][]float64{
			{-1, -1, -1}, {1, -1, -1}, {1, 1, -1}, {-1, 1, -1},
			{-1, -1, 1}, {1, -1, 1}, {1, 1, 1}, {-1, 1, 1},
		}
	case "sphere":
		vertices := make([][]float64, 0)
		for i := 0; i <= 20; i++ {
			for j := 0; j <= 20; j++ {
				theta := float64(i) / 20 * math.Pi
				phi := float64(j) / 20 * 2 * math.Pi
				x := math.Sin(theta) * math.Cos(phi)
				y := math.Sin(theta) * math.Sin(phi)
				z := math.Cos(theta)
				vertices = append(vertices, []float64{x, y, z})
			}
		}
		return vertices
	case "pyramid":
		return [][]float64{
			{-1, -1, -1}, {1, -1, -1}, {1, -1, 1}, {-1, -1, 1},
			{0, 1, 0},
		}
	default:
		return [][]float64{
			{-1, -1, -1}, {1, -1, -1}, {1, 1, -1}, {-1, 1, -1},
			{-1, -1, 1}, {1, -1, 1}, {1, 1, 1}, {-1, 1, 1},
		}
	}
}

func (s *ARGeneratorService) generateObjectFaces(objectType string) [][]int {
	switch objectType {
	case "cube":
		return [][]int{
			{0, 1, 2, 3}, {4, 5, 6, 7}, {0, 1, 5, 4},
			{2, 3, 7, 6}, {0, 3, 7, 4}, {1, 2, 6, 5},
		}
	case "pyramid":
		return [][]int{
			{0, 1, 4}, {1, 2, 4}, {2, 3, 4}, {3, 0, 4}, {0, 1, 2, 3},
		}
	default:
		return [][]int{
			{0, 1, 2, 3}, {4, 5, 6, 7}, {0, 1, 5, 4},
			{2, 3, 7, 6}, {0, 3, 7, 4}, {1, 2, 6, 5},
		}
	}
}

func (s *ARGeneratorService) generateInstructions(gestureType ARGestureType, targetAngle float64) string {
	direction := "顺时针"
	if rand.Float64() > 0.5 {
		direction = "逆时针"
	}

	switch gestureType {
	case ARGestureRotateX:
		return fmt.Sprintf("将物体沿X轴%s旋转%.0f度", direction, targetAngle)
	case ARGestureRotateY:
		return fmt.Sprintf("将物体沿Y轴%s旋转%.0f度", direction, targetAngle)
	case ARGestureRotateZ:
		return fmt.Sprintf("将物体沿Z轴%s旋转%.0f度", direction, targetAngle)
	case ARGesturePinch:
		return "双指缩放物体到指定大小"
	case ARGestureSwipe:
		return "滑动切换到下一个物体"
	default:
		return "按照提示旋转物体"
	}
}

func (s *ARGeneratorService) generateModelData(puzzle *ARCaptchaPuzzle) string {
	modelData := map[string]interface{}{
		"type":      puzzle.Object.Type,
		"color":     puzzle.Object.Color,
		"vertices":  puzzle.Object.Vertices,
		"faces":     puzzle.Object.Faces,
		"scale":     puzzle.Object.Scale,
		"position":  []float64{puzzle.Object.PositionX, puzzle.Object.PositionY, puzzle.Object.PositionZ},
	}

	modelJSON, _ := json.Marshal(modelData)
	return "data:model/json;base64," + string(modelJSON)
}

func (s *ARVerifierService) Verify(ctx context.Context, req *VerifyARRequest) (*VerifyARResult, error) {
	session, err := s.getSession(req.SessionID)
	if err != nil {
		return &VerifyARResult{
			Success: false,
			Message: "会话不存在",
			Score:   0,
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifyARResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifyARResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	s.incrementVerifyCount(req.SessionID)

	var puzzle ARCaptchaPuzzle
	if err := json.Unmarshal([]byte(session.BackgroundURL), &puzzle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal puzzle: %w", err)
	}

	if session.Status == "verified" {
		return &VerifyARResult{
			Success: true,
			Message: "验证码已验证通过",
			Score:   100,
		}, nil
	}

	rotationScore := s.calculateRotationScore(&puzzle, req)

	gestureScore := s.analyzeGestureData(req.GestureData, &puzzle)

	deviceScore := s.analyzeDeviceMotion(req.DeviceMotion, &puzzle)

	naturalnessScore := s.calculateNaturalnessScore(req.GestureData, req.TouchPoints)

	totalScore := rotationScore*0.4 + gestureScore*0.3 + deviceScore*0.15 + naturalnessScore*0.15

	isSuccess := rotationScore >= 0.7 && gestureScore >= 0.5

	if isSuccess {
		session.Status = "verified"
		if s.sessionCache != nil {
			_ = s.sessionCache.UpdateStatus(ctx, req.SessionID, "verified")
		}
		if s.captchaRepo != nil {
			_ = s.captchaRepo.UpdateStatus(req.SessionID, "verified")
		}
	}

	return &VerifyARResult{
		Success:          isSuccess,
		Message:          func() string {
			if isSuccess {
				return "AR验证成功"
			}
			return fmt.Sprintf("验证失败，匹配度 %.0f%%", totalScore*100)
		}(),
		Score:            totalScore * 100,
		Accuracy:         rotationScore * 100,
		GestureScore:     gestureScore * 100,
		DeviceScore:      deviceScore * 100,
		NaturalnessScore: naturalnessScore * 100,
	}, nil
}

func (s *ARVerifierService) calculateRotationScore(puzzle *ARCaptchaPuzzle, req *VerifyARRequest) float64 {
	var expectedRot, actualRot float64

	switch puzzle.GestureType {
	case ARGestureRotateX:
		expectedRot = puzzle.Object.TargetRotX
		actualRot = req.RotationX
	case ARGestureRotateY:
		expectedRot = puzzle.Object.TargetRotY
		actualRot = req.RotationY
	case ARGestureRotateZ:
		expectedRot = puzzle.Object.TargetRotZ
		actualRot = req.RotationZ
	default:
		expectedRot = puzzle.Object.TargetRotX
		actualRot = req.RotationX
	}

	diff := math.Abs(normalizeAngle(expectedRot) - normalizeAngle(actualRot))
	if diff > 180 {
		diff = 360 - diff
	}

	accuracy := math.Max(0, 1-diff/puzzle.Tolerance)
	return math.Min(1, accuracy)
}

func (s *ARVerifierService) analyzeGestureData(gestureData []ARGesturePoint, puzzle *ARCaptchaPuzzle) float64 {
	if len(gestureData) < 5 {
		return 0.3
	}

	var totalVariation float64
	var directionChanges int

	for i := 1; i < len(gestureData); i++ {
		var prevDelta, currDelta float64

		switch puzzle.GestureType {
		case ARGestureRotateX:
			prevDelta = gestureData[i-1].RotationX
			currDelta = gestureData[i].RotationX
		case ARGestureRotateY:
			prevDelta = gestureData[i-1].RotationY
			currDelta = gestureData[i].RotationY
		case ARGestureRotateZ:
			prevDelta = gestureData[i-1].RotationZ
			currDelta = gestureData[i].RotationZ
		}

		totalVariation += math.Abs(currDelta - prevDelta)

		if i > 1 {
			prevDelta2 := 0.0
			switch puzzle.GestureType {
			case ARGestureRotateX:
				prevDelta2 = gestureData[i-2].RotationX
			case ARGestureRotateY:
				prevDelta2 = gestureData[i-2].RotationY
			case ARGestureRotateZ:
				prevDelta2 = gestureData[i-2].RotationZ
			}

			if (currDelta-prevDelta)*(prevDelta-prevDelta2) < 0 {
				directionChanges++
			}
		}
	}

	avgVariation := totalVariation / float64(len(gestureData)-1)

	expectedAngle := puzzle.TargetAngle
	variationRatio := avgVariation / expectedAngle

	naturalScore := 0.5
	if variationRatio > 0.3 && variationRatio < 2.0 {
		naturalScore += 0.3
	}

	changeRatio := float64(directionChanges) / float64(len(gestureData)-1)
	if changeRatio > 0.05 && changeRatio < 0.3 {
		naturalScore += 0.2
	}

	return math.Min(1, naturalScore)
}

func (s *ARVerifierService) analyzeDeviceMotion(motion *DeviceMotionData, puzzle *ARCaptchaPuzzle) float64 {
	if motion == nil {
		return 0.5
	}

	accelMagnitude := math.Sqrt(
		motion.AccelerationX*motion.AccelerationX +
			motion.AccelerationY*motion.AccelerationY +
			motion.AccelerationZ*motion.AccelerationZ,
	)

	if accelMagnitude < 0.1 {
		return 0.2
	}

	if accelMagnitude > 20 {
		return 0.3
	}

	return 0.6 + math.Min(0.4, accelMagnitude/20)
}

func (s *ARVerifierService) calculateNaturalnessScore(gestureData []ARGesturePoint, touchPoints []TouchPoint) float64 {
	if len(gestureData) < 3 {
		return 0.5
	}

	var timeGaps []int64
	for i := 1; i < len(gestureData); i++ {
		gap := gestureData[i].Timestamp - gestureData[i-1].Timestamp
		if gap > 0 && gap < 5000 {
			timeGaps = append(timeGaps, gap)
		}
	}

	if len(timeGaps) == 0 {
		return 0.5
	}

	var meanGap, varianceGap float64
	for _, gap := range timeGaps {
		meanGap += float64(gap)
	}
	meanGap /= float64(len(timeGaps))

	for _, gap := range timeGaps {
		diff := float64(gap) - meanGap
		varianceGap += diff * diff
	}
	varianceGap /= float64(len(timeGaps))

	stdDev := math.Sqrt(varianceGap)
	coefficientVariation := stdDev / meanGap

	if coefficientVariation < 0.1 {
		return 0.2
	}

	if coefficientVariation > 2 {
		return 0.4
	}

	return 0.6 + math.Min(0.4, 1/coefficientVariation)
}

func normalizeAngle(angle float64) float64 {
	for angle < 0 {
		angle += 360
	}
	for angle >= 360 {
		angle -= 360
	}
	return angle
}

func (s *ARVerifierService) getSession(sessionID string) (*models.CaptchaSession, error) {
	if s.sessionCache != nil {
		session, err := s.sessionCache.Get(context.Background(), sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	if s.captchaRepo != nil {
		session, err := s.captchaRepo.GetBySessionID(sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (s *ARVerifierService) incrementVerifyCount(sessionID string) {
	if s.sessionCache != nil {
		_ = s.sessionCache.IncrementVerifyCount(context.Background(), sessionID)
	}

	if s.captchaRepo != nil {
		_ = s.captchaRepo.UpdateVerifyCount(sessionID)
	}
}

func (s *ARVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	return s.getSession(sessionID)
}

func (s *ARVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
	session, err := s.getSession(sessionID)
	if err != nil {
		return false, "会话不存在"
	}

	if time.Now().After(session.ExpiredAt) {
		return false, "验证码已过期"
	}

	if session.Status == "verified" {
		return false, "验证码已验证通过"
	}

	if session.VerifyCount >= session.MaxAttempts {
		return false, "验证次数已用完"
	}

	return true, ""
}
