package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	github.com/hjtpx/hjtpx/internal/repository/cache"
	github.com/hjtpx/hjtpx/internal/repository/db"
)

type ARGeneratorService struct {
	sessionCache   *cache.SessionCache
	captchaRepo    *db.CaptchaRepository
	inMemoryStore  map[string]*ARSession
}

type ARCaptchaRequest struct {
	SceneType   string `json:"scene_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Difficulty  int    `json:"difficulty"`
	ClientIP    string `json:"client_ip"`
	UserAgent   string `json:"user_agent"`
	Fingerprint string `json:"fingerprint"`
}

type ARCaptchaResponse struct {
	SessionID     string                   `json:"session_id"`
	SceneType     string                   `json:"scene_type"`
	SceneConfig   *ARSceneConfig           `json:"scene_config"`
	Instructions  string                   `json:"instructions"`
	TargetActions []string                 `json:"target_actions"`
	ExpiresIn     int64                    `json:"expires_in"`
	ExpiresAt     int64                    `json:"expires_at"`
	Difficulty    int                      `json:"difficulty"`
	Width         int                      `json:"width"`
	Height        int                      `json:"height"`
	WebXRSupport  bool                     `json:"webxr_support"`
	ModelURL      string                   `json:"model_url,omitempty"`
}

type ARVerifyRequest struct {
	SessionID    string                 `json:"session_id"`
	ObjectPos    []float64             `json:"object_position"`
	ObjectRot    []float64             `json:"object_rotation"`
	GestureData  *ARGestureData         `json:"gesture_data"`
	BehaviorData map[string]interface{} `json:"behavior_data"`
	TraceData    interface{}            `json:"trace_data"`
}

type ARVerifyResponse struct {
	Success       bool                    `json:"success"`
	Score         float64                 `json:"score"`
	Message       string                  `json:"message"`
	Accuracy      float64                 `json:"accuracy"`
	Feedback      string                  `json:"feedback,omitempty"`
}

type ARSceneConfig struct {
	Type          string                 `json:"type"`
	Objects       []*ARObject            `json:"objects"`
	TargetZone    *ARTargetZone          `json:"target_zone,omitempty"`
	Gestures      []string               `json:"gestures,omitempty"`
	TargetGesture string                 `json:"target_gesture,omitempty"`
	Environment   string                 `json:"environment"`
	Lighting      string                 `json:"lighting"`
	Constraints   []ARConstraint        `json:"constraints,omitempty"`
}

type ARObject struct {
	ID            string       `json:"id"`
	Type          string       `json:"type"`
	Position      []float64    `json:"position"`
	Rotation      []float64    `json:"rotation"`
	Scale         []float64    `json:"scale"`
	Color         string       `json:"color"`
	TargetPosition []float64   `json:"target_position,omitempty"`
	TargetRotation []float64   `json:"target_rotation,omitempty"`
	Interactable  bool         `json:"interactable"`
	Physics       bool         `json:"physics"`
}

type ARTargetZone struct {
	Position []float64 `json:"position"`
	Size     []float64 `json:"size"`
	Shape    string    `json:"shape"`
	Visible  bool      `json:"visible"`
}

type ARConstraint struct {
	Type        string    `json:"type"`
	TargetID    string    `json:"target_id"`
	Property    string    `json:"property"`
	Min         float64   `json:"min,omitempty"`
	Max         float64   `json:"max,omitempty"`
	Tolerance   float64   `json:"tolerance"`
	Weight      float64   `json:"weight"`
}

type ARGestureData struct {
	Type         string      `json:"type"`
	Points       [][]float64 `json:"points"`
	Duration     int         `json:"duration"`
	Recognized   bool        `json:"recognized"`
	Confidence   float64     `json:"confidence"`
}

type ARSession struct {
	SessionID      string        `json:"session_id"`
	SceneType      string        `json:"scene_type"`
	SceneConfig    *ARSceneConfig `json:"scene_config"`
	CorrectActions  []string      `json:"correct_actions"`
	TargetObjects  map[string]*ARTargetResult `json:"target_objects"`
	Status         string        `json:"status"`
	VerifyCount    int           `json:"verify_count"`
	MaxAttempts    int           `json:"max_attempts"`
	CreatedAt      time.Time     `json:"created_at"`
	ExpiredAt      time.Time     `json:"expired_at"`
	ClientIP       string        `json:"client_ip"`
	UserAgent      string        `json:"user_agent"`
	Fingerprint    string        `json:"fingerprint"`
}

type ARTargetResult struct {
	ObjectID      string  `json:"object_id"`
	FinalPosition []float64 `json:"final_position"`
	FinalRotation []float64 `json:"final_rotation"`
	Distance      float64 `json:"distance"`
	AngleDiff     float64 `json:"angle_diff"`
	Score         float64 `json:"score"`
	Success       bool    `json:"success"`
}

func NewARGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *ARGeneratorService {
	return &ARGeneratorService{
		sessionCache:  sessionCache,
		captchaRepo:   captchaRepo,
		inMemoryStore: make(map[string]*ARSession),
	}
}

func (s *ARGeneratorService) Generate(ctx context.Context, req *ARCaptchaRequest) (*ARCaptchaResponse, error) {
	if req.SceneType == "" {
		req.SceneType = s.selectDefaultSceneType()
	}
	if req.Width <= 0 {
		req.Width = 640
	}
	if req.Height <= 0 {
		req.Height = 480
	}
	if req.Difficulty <= 0 {
		req.Difficulty = 2
	}
	if req.Difficulty > 5 {
		req.Difficulty = 5
	}

	sessionID := generateARSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	sceneConfig := s.generateSceneConfig(req.SceneType, req.Difficulty)
	instructions := s.generateInstructions(req.SceneType, req.Difficulty)
	targetActions := s.extractTargetActions(sceneConfig)

	session := &ARSession{
		SessionID:     sessionID,
		SceneType:     req.SceneType,
		SceneConfig:   sceneConfig,
		CorrectActions: targetActions,
		TargetObjects: make(map[string]*ARTargetResult),
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
		CreatedAt:     time.Now(),
		ExpiredAt:     expiresAt,
		ClientIP:      req.ClientIP,
		UserAgent:     req.UserAgent,
		Fingerprint:   req.Fingerprint,
	}

	// Always save to in-memory store first
	s.inMemoryStore[sessionID] = session

	if err := s.saveSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return &ARCaptchaResponse{
		SessionID:     sessionID,
		SceneType:     req.SceneType,
		SceneConfig:   sceneConfig,
		Instructions:  instructions,
		TargetActions: targetActions,
		ExpiresIn:     int64(5 * time.Minute / time.Second),
		ExpiresAt:     expiresAt.Unix(),
		Difficulty:    req.Difficulty,
		Width:         req.Width,
		Height:        req.Height,
		WebXRSupport:  true,
		ModelURL:      fmt.Sprintf("/api/v1/captcha/ar/model/%s", sessionID),
	}, nil
}

func (s *ARGeneratorService) Verify(ctx context.Context, req *ARVerifyRequest) (*ARVerifyResponse, error) {
	session, err := s.GetSession(ctx, req.SessionID)
	if err != nil {
		return &ARVerifyResponse{
			Success: false,
			Score:   0,
			Message: "会话不存在或已过期",
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &ARVerifyResponse{
			Success: false,
			Score:   0,
			Message: "会话已过期",
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &ARVerifyResponse{
			Success: false,
			Score:   0,
			Message: "验证次数已用完",
		}, nil
	}

	session.VerifyCount++
	s.saveSession(ctx, session)

	totalScore := s.evaluateInteraction(req, session)
	accuracy := s.calculateAccuracy(req, session)
	success := totalScore >= 0.6 && accuracy >= 0.5

	feedback := ""
	if !success {
		feedback = s.generateFeedback(req, session)
	}

	if success {
		session.Status = "verified"
		s.saveSession(ctx, session)
	}

	return &ARVerifyResponse{
		Success:  success,
		Score:    totalScore,
		Message:  map[bool]string{true: "验证成功", false: "验证失败"}[success],
		Accuracy: accuracy,
		Feedback: feedback,
	}, nil
}

func (s *ARGeneratorService) GetSession(ctx context.Context, sessionID string) (*ARSession, error) {
	// First check in-memory store
	if session, ok := s.inMemoryStore[sessionID]; ok {
		if time.Now().Before(session.ExpiredAt) {
			return session, nil
		}
		delete(s.inMemoryStore, sessionID)
	}

	if s.sessionCache != nil {
		session, err := s.getCachedSession(ctx, sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	if s.captchaRepo != nil {
		session, err := s.getDatabaseSession(sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (s *ARGeneratorService) saveSession(ctx context.Context, session *ARSession) error {
	// Always save to in-memory store first
	s.inMemoryStore[session.SessionID] = session

	if s.sessionCache != nil {
		data, err := json.Marshal(session)
		if err != nil {
			return err
		}
		return s.sessionCache.SetRaw(ctx, session.SessionID, string(data), 5*time.Minute)
	}
	return nil
}

func (s *ARGeneratorService) getCachedSession(ctx context.Context, sessionID string) (*ARSession, error) {
	return nil, fmt.Errorf("session not found in cache")
}

func (s *ARGeneratorService) getDatabaseSession(sessionID string) (*ARSession, error) {
	return nil, fmt.Errorf("session not found in database")
}

func (s *ARGeneratorService) selectDefaultSceneType() string {
	sceneTypes := []string{"object_placement", "gesture_recognition", "object_rotation", "sequential_action"}
	return sceneTypes[rand.Intn(len(sceneTypes))]
}

func (s *ARGeneratorService) generateSceneConfig(sceneType string, difficulty int) *ARSceneConfig {
	config := &ARSceneConfig{
		Type:        sceneType,
		Objects:      []*ARObject{},
		Environment: "room",
		Lighting:    "natural",
		Constraints: []ARConstraint{},
	}

	switch sceneType {
	case "object_placement":
		config.Objects = s.generatePlacementObjects(difficulty)
		config.TargetZone = s.generateTargetZone(difficulty)
		for _, obj := range config.Objects {
			if obj.TargetPosition != nil {
				constraint := ARConstraint{
					Type:       "distance",
					TargetID:   obj.ID,
					Property:   "position",
					Tolerance:  0.1 + float64(3-difficulty)*0.05,
					Weight:     0.4,
				}
				config.Constraints = append(config.Constraints, constraint)
			}
		}

	case "gesture_recognition":
		config.Gestures = []string{"wave", "point", "circle", "swipe_up", "swipe_down", "pinch", "rotate"}
		config.TargetGesture = config.Gestures[rand.Intn(len(config.Gestures))]

	case "object_rotation":
		config.Objects = s.generateRotationObjects(difficulty)
		for _, obj := range config.Objects {
			if obj.TargetRotation != nil {
				constraint := ARConstraint{
					Type:      "angle",
					TargetID:  obj.ID,
					Property:  "rotation",
					Tolerance: 15.0 + float64(3-difficulty)*5.0,
					Weight:    0.3,
				}
				config.Constraints = append(config.Constraints, constraint)
			}
		}

	case "sequential_action":
		config.Objects = s.generateSequentialObjects(difficulty)
		for i := 0; i < len(config.Objects); i++ {
			constraint := ARConstraint{
				Type:      "sequence",
				TargetID:  config.Objects[i].ID,
				Property:  "order",
				Min:       float64(i),
				Max:       float64(i),
				Tolerance: 0,
				Weight:    0.2 / float64(len(config.Objects)),
			}
			config.Constraints = append(config.Constraints, constraint)
		}
	}

	return config
}

func (s *ARGeneratorService) generatePlacementObjects(difficulty int) []*ARObject {
	shapes := []string{"cube", "sphere", "cylinder", "pyramid"}
	colors := []string{"red", "blue", "green", "yellow", "purple", "orange"}

	objectCount := 1 + difficulty
	objects := make([]*ARObject, objectCount)

	for i := 0; i < objectCount; i++ {
		obj := &ARObject{
			ID:            fmt.Sprintf("obj_%d", i),
			Type:          shapes[rand.Intn(len(shapes))],
			Position:      []float64{float64(rand.Intn(3) - 1), 0, float64(rand.Intn(3) - 1)},
			Rotation:      []float64{0, float64(rand.Intn(360)), 0},
			Scale:         []float64{0.2, 0.2, 0.2},
			Color:         colors[rand.Intn(len(colors))],
			TargetPosition: []float64{float64(rand.Intn(2) * 2), 0, float64(rand.Intn(2) * 2)},
			Interactable:  true,
			Physics:       true,
		}
		objects[i] = obj
	}

	return objects
}

func (s *ARGeneratorService) generateRotationObjects(difficulty int) []*ARObject {
	shapes := []string{"cube", "pyramid", "prism"}
	colors := []string{"red", "blue", "green"}

	objectCount := 1 + difficulty/2
	objects := make([]*ARObject, objectCount)

	for i := 0; i < objectCount; i++ {
		targetRotationY := float64(rand.Intn(4) * 90)
		obj := &ARObject{
			ID:             fmt.Sprintf("rot_obj_%d", i),
			Type:           shapes[rand.Intn(len(shapes))],
			Position:       []float64{0, 0.2, -1},
			Rotation:       []float64{0, float64(rand.Intn(4) * 90), 0},
			Scale:          []float64{0.3, 0.3, 0.3},
			Color:          colors[i%len(colors)],
			TargetRotation: []float64{0, targetRotationY, 0},
			Interactable:   true,
			Physics:        false,
		}
		objects[i] = obj
	}

	return objects
}

func (s *ARGeneratorService) generateSequentialObjects(difficulty int) []*ARObject {
	colors := []string{"red", "blue", "green", "yellow", "purple"}
	sequenceLength := 2 + difficulty
	objects := make([]*ARObject, sequenceLength)

	for i := 0; i < sequenceLength; i++ {
		obj := &ARObject{
			ID:           fmt.Sprintf("seq_obj_%d", i),
			Type:         "sphere",
			Position:     []float64{float64(i) * 0.5, 0, 0},
			Rotation:     []float64{0, 0, 0},
			Scale:        []float64{0.15, 0.15, 0.15},
			Color:        colors[i%len(colors)],
			Interactable: true,
			Physics:      false,
		}
		objects[i] = obj
	}

	return objects
}

func (s *ARGeneratorService) generateTargetZone(difficulty int) *ARTargetZone {
	return &ARTargetZone{
		Position: []float64{1, 0.05, 0},
		Size:     []float64{0.3, 0.3, 0.3},
		Shape:    "cube",
		Visible:  difficulty <= 2,
	}
}

func (s *ARGeneratorService) generateInstructions(sceneType string, difficulty int) string {
	switch sceneType {
	case "object_placement":
		if difficulty <= 2 {
			return "请将所有物体拖动到对应的目标区域"
		} else if difficulty <= 3 {
			return "请按顺序将物体放置到高亮区域"
		}
		return "请精确将所有物体放置到各自的目标位置"

	case "gesture_recognition":
		return "请做出手势：指向(point)"

	case "object_rotation":
		if difficulty <= 2 {
			return "请旋转物体使其朝向正确方向"
		}
		return "请将所有物体旋转到指定角度"

	case "sequential_action":
		if difficulty <= 2 {
			return "请按顺序点击物体"
		}
		return "请按颜色顺序点击物体"
	}

	return "请按照提示完成AR验证"
}

func (s *ARGeneratorService) extractTargetActions(config *ARSceneConfig) []string {
	actions := []string{}

	for _, obj := range config.Objects {
		if obj.TargetPosition != nil {
			actions = append(actions, fmt.Sprintf("place:%s", obj.ID))
		}
		if obj.TargetRotation != nil {
			actions = append(actions, fmt.Sprintf("rotate:%s", obj.ID))
		}
	}

	if config.TargetGesture != "" {
		actions = append(actions, fmt.Sprintf("gesture:%s", config.TargetGesture))
	}

	return actions
}

func (s *ARGeneratorService) evaluateInteraction(req *ARVerifyRequest, session *ARSession) float64 {
	totalScore := 0.0
	weightSum := 0.0

	if req.ObjectPos != nil && len(req.ObjectPos) >= 3 {
		for _, obj := range session.SceneConfig.Objects {
			if obj.TargetPosition == nil {
				continue
			}

			distance := calculateDistance3D(req.ObjectPos, obj.TargetPosition)
			tolerance := 0.2
			weight := 0.4

			var objScore float64
			if distance <= tolerance {
				objScore = 1.0
			} else if distance <= tolerance*2 {
				objScore = 0.5
			} else if distance <= tolerance*3 {
				objScore = 0.2
			}

			foundConstraint := false
			for _, constraint := range session.SceneConfig.Constraints {
				if constraint.TargetID == obj.ID && constraint.Type == "distance" {
					tolerance = constraint.Tolerance
					weight = constraint.Weight
					if distance <= tolerance {
						objScore = 1.0
					} else {
						objScore = math.Max(0, 1.0-distance/tolerance)
					}
					totalScore += objScore * weight
					weightSum += weight
					foundConstraint = true
				}
			}

			if !foundConstraint {
				totalScore += objScore * weight
				weightSum += weight
			}
		}
	}

	if req.ObjectRot != nil && len(req.ObjectRot) >= 3 {
		for _, obj := range session.SceneConfig.Objects {
			if obj.TargetRotation == nil {
				continue
			}

			angleDiff := calculateAngleDifference(req.ObjectRot, obj.TargetRotation)

			for _, constraint := range session.SceneConfig.Constraints {
				if constraint.TargetID == obj.ID && constraint.Type == "angle" {
					tolerance := constraint.Tolerance
					var rotScore float64
					if angleDiff <= tolerance {
						rotScore = 1.0
					} else {
						rotScore = math.Max(0, 1.0-angleDiff/tolerance)
					}
					totalScore += rotScore * constraint.Weight
					weightSum += constraint.Weight
				}
			}
		}
	}

	if req.GestureData != nil {
		if req.GestureData.Recognized && req.GestureData.Confidence >= 0.7 {
			totalScore += 0.3
			weightSum += 0.3
		} else if req.GestureData.Recognized {
			totalScore += 0.15
			weightSum += 0.3
		}
	}

	if req.BehaviorData != nil {
		behaviorScore := s.calculateBehaviorScore(req.BehaviorData)
		totalScore += behaviorScore * 0.2
		weightSum += 0.2
	}

	if weightSum > 0 {
		return totalScore / weightSum
	}

	return totalScore
}

func (s *ARGeneratorService) calculateAccuracy(req *ARVerifyRequest, session *ARSession) float64 {
	if session.SceneConfig.Type == "object_placement" {
		if req.ObjectPos == nil {
			return 0
		}

		successCount := 0
		totalCount := 0

		for _, obj := range session.SceneConfig.Objects {
			if obj.TargetPosition != nil {
				totalCount++
				distance := calculateDistance3D(req.ObjectPos, obj.TargetPosition)
				if distance <= 0.2 {
					successCount++
				}
			}
		}

		if totalCount == 0 {
			return 1.0
		}
		return float64(successCount) / float64(totalCount)
	}

	if session.SceneConfig.Type == "gesture_recognition" {
		if req.GestureData == nil {
			return 0
		}
		if req.GestureData.Recognized && req.GestureData.Confidence >= 0.7 {
			return 1.0
		}
		return req.GestureData.Confidence
	}

	return 0.5
}

func (s *ARGeneratorService) calculateBehaviorScore(behaviorData map[string]interface{}) float64 {
	score := 0.5

	if moveCount, ok := behaviorData["move_count"].(float64); ok {
		if moveCount > 5 {
			score += 0.1
		}
	}

	if timeSpent, ok := behaviorData["time_spent"].(float64); ok {
		if timeSpent >= 2.0 && timeSpent <= 30.0 {
			score += 0.15
		}
	}

	if accuracy, ok := behaviorData["accuracy"].(float64); ok {
		score += accuracy * 0.2
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (s *ARGeneratorService) generateFeedback(req *ARVerifyRequest, session *ARSession) string {
	if session.SceneConfig.Type == "object_placement" {
		if req.ObjectPos != nil {
			return "物体位置不够精确，请尝试更精确地放置"
		}
		return "请将物体拖动到目标区域"
	}

	if session.SceneConfig.Type == "gesture_recognition" {
		return "手势识别失败，请按照提示做出正确的动作"
	}

	if session.SceneConfig.Type == "object_rotation" {
		return "物体角度不正确，请继续调整旋转"
	}

	return "请仔细按照指令完成操作"
}

func calculateDistance3D(pos1, pos2 []float64) float64 {
	if len(pos1) < 3 || len(pos2) < 3 {
		return 999.0
	}

	dx := pos1[0] - pos2[0]
	dy := pos1[1] - pos2[1]
	dz := pos1[2] - pos2[2]

	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func calculateAngleDifference(rot1, rot2 []float64) float64 {
	if len(rot1) < 3 || len(rot2) < 3 {
		return 180.0
	}

	diff := math.Abs(rot1[1] - rot2[1])
	if diff > 180 {
		diff = 360 - diff
	}

	return diff
}

func generateARSessionID() string {
	return fmt.Sprintf("ar_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}
