package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	github.com/hjtpx/hjtpx/internal/repository/cache"
	github.com/hjtpx/hjtpx/internal/repository/db"
)

type VRARMode string

const (
	VRARModeVR         VRARMode = "vr"
	VRARModeAR         VRARMode = "ar"
	VRARModeHybrid     VRARMode = "hybrid"
	VRARModeInteractive VRARMode = "interactive"
)

type VRARType string

const (
	VRARType3DPlacement  VRARType = "3d_placement"
	VRARTypeGesture      VRARType = "gesture"
	VRARTypeEyeTracking  VRARType = "eye_tracking"
	VRARTypeObjectRotation VRARType = "object_rotation"
	VRARTypeSpatialPuzzle VRARType = "spatial_puzzle"
	VRARTypeSequential   VRARType = "sequential"
)

type VRARCaptchaRequest struct {
	Mode        VRARMode `json:"mode"`
	Type        VRARType `json:"type"`
	Difficulty  string   `json:"difficulty"`
	ClientIP    string   `json:"client_ip"`
	UserAgent   string   `json:"user_agent"`
	Fingerprint string   `json:"fingerprint"`
}

type VRARCaptchaResponse struct {
	SessionID     string         `json:"session_id"`
	Mode          VRARMode       `json:"mode"`
	Type          VRARType       `json:"type"`
	SceneConfig   *VRARSceneConfig `json:"scene_config"`
	Instructions  string         `json:"instructions"`
	WebXRConfig   *WebXRConfig   `json:"webxr_config"`
	GestureConfig *VRGestureConfig `json:"gesture_config,omitempty"`
	ExpiresIn     int64          `json:"expires_in"`
	ExpiresAt     int64          `json:"expires_at"`
}

type VRARVerifyRequest struct {
	SessionID    string                 `json:"session_id"`
	Interaction  *VRInteractionData     `json:"interaction"`
	GestureData  *VRHandGestureData     `json:"gesture_data,omitempty"`
	EyeData      *VREyeTrackingData     `json:"eye_data,omitempty"`
	ARGesture    *ARGestureData         `json:"ar_gesture,omitempty"`
	BehaviorData map[string]interface{} `json:"behavior_data,omitempty"`
	TraceData    interface{}            `json:"trace_data,omitempty"`
}

type VRARVerifyResponse struct {
	Success   bool        `json:"success"`
	Score     float64     `json:"score"`
	Message   string      `json:"message"`
	Accuracy  float64     `json:"accuracy"`
	Feedback  string      `json:"feedback,omitempty"`
	Analytics *VRAnalytics `json:"analytics,omitempty"`
}

type VRARSceneConfig struct {
	Mode          VRARMode       `json:"mode"`
	Type          VRARType       `json:"type"`
	Environment   string         `json:"environment"`
	Objects       []*VRObject    `json:"objects"`
	Targets       []*VRTarget    `json:"targets,omitempty"`
	Gestures      []string       `json:"gestures,omitempty"`
	TargetGesture string         `json:"target_gesture,omitempty"`
	Lighting      *VRLighting    `json:"lighting"`
	Camera        *VRCamera      `json:"camera"`
	Physics       bool           `json:"physics"`
	Constraints   []VRConstraint `json:"constraints,omitempty"`
	Hints         []string       `json:"hints,omitempty"`
	ARTargetZone  *ARTargetZone  `json:"ar_target_zone,omitempty"`
}

type VRARSession struct {
	SessionID     string                `json:"session_id"`
	Mode          VRARMode              `json:"mode"`
	Type          VRARType              `json:"type"`
	SceneConfig   *VRARSceneConfig      `json:"scene_config"`
	CorrectActions []string              `json:"correct_actions"`
	TargetResults  map[string]*VRTargetResult `json:"target_results"`
	Status        string                `json:"status"`
	VerifyCount   int                   `json:"verify_count"`
	MaxAttempts   int                   `json:"max_attempts"`
	CreatedAt     time.Time             `json:"created_at"`
	ExpiredAt     time.Time             `json:"expired_at"`
	ClientIP      string                `json:"client_ip"`
	UserAgent     string                `json:"user_agent"`
	Fingerprint   string                `json:"fingerprint"`
}

type VRARGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
	vrGenerator  *VRGeneratorService
	arGenerator  *ARGeneratorService
}

func NewVRARGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *VRARGeneratorService {
	return &VRARGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
		vrGenerator:  NewVRGeneratorService(sessionCache, captchaRepo),
		arGenerator:  NewARGeneratorService(sessionCache, captchaRepo),
	}
}

func NewVRARGeneratorServiceSimple() *VRARGeneratorService {
	return &VRARGeneratorService{
		vrGenerator: NewVRGeneratorServiceSimple(),
	}
}

func (s *VRARGeneratorService) Generate(ctx context.Context, req *VRARCaptchaRequest) (*VRARCaptchaResponse, error) {
	if req.Mode == "" {
		req.Mode = s.selectDefaultMode()
	}
	if req.Type == "" {
		req.Type = s.selectDefaultType(req.Mode)
	}
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}

	sessionID := s.generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	sceneConfig := s.generateSceneConfig(req.Mode, req.Type, req.Difficulty)
	instructions := s.generateInstructions(req.Mode, req.Type, req.Difficulty)
	webXRConfig := s.generateWebXRConfig(req.Mode, req.Type)
	gestureConfig := s.generateGestureConfig(req.Mode, req.Type, req.Difficulty)

	correctActions := s.extractCorrectActions(sceneConfig)

	session := &VRARSession{
		SessionID:     sessionID,
		Mode:          req.Mode,
		Type:          req.Type,
		SceneConfig:   sceneConfig,
		CorrectActions: correctActions,
		TargetResults:  make(map[string]*VRTargetResult),
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
		CreatedAt:     time.Now(),
		ExpiredAt:     expiresAt,
		ClientIP:      req.ClientIP,
		UserAgent:     req.UserAgent,
		Fingerprint:   req.Fingerprint,
	}

	if err := s.saveSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return &VRARCaptchaResponse{
		SessionID:     sessionID,
		Mode:          req.Mode,
		Type:          req.Type,
		SceneConfig:   sceneConfig,
		Instructions:  instructions,
		WebXRConfig:   webXRConfig,
		GestureConfig: gestureConfig,
		ExpiresIn:     int64(5 * time.Minute / time.Second),
		ExpiresAt:     expiresAt.Unix(),
	}, nil
}

func (s *VRARGeneratorService) GetSession(ctx context.Context, sessionID string) (*VRARSession, error) {
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

func (s *VRARGeneratorService) saveSession(ctx context.Context, session *VRARSession) error {
	if s.sessionCache != nil {
		data, err := json.Marshal(session)
		if err != nil {
			return err
		}
		return s.sessionCache.SetRaw(ctx, session.SessionID, string(data), 5*time.Minute)
	}
	return nil
}

func (s *VRARGeneratorService) getCachedSession(ctx context.Context, sessionID string) (*VRARSession, error) {
	data, err := s.sessionCache.GetRaw(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	var session VRARSession
	err = json.Unmarshal([]byte(data), &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *VRARGeneratorService) getDatabaseSession(sessionID string) (*VRARSession, error) {
	return nil, fmt.Errorf("session not found in database")
}

func (s *VRARGeneratorService) selectDefaultMode() VRARMode {
	modes := []VRARMode{VRARModeVR, VRARModeAR, VRARModeHybrid}
	return modes[rand.Intn(len(modes))]
}

func (s *VRARGeneratorService) selectDefaultType(mode VRARMode) VRARType {
	if mode == VRARModeAR {
		return VRARTypeObjectRotation
	}
	return VRARType3DPlacement
}

func (s *VRARGeneratorService) generateSceneConfig(mode VRARMode, captchaType VRARType, difficulty string) *VRARSceneConfig {
	rand.Seed(time.Now().UnixNano())

	config := &VRARSceneConfig{
		Mode:        mode,
		Type:        captchaType,
		Environment: "simple_room",
		Objects:     []*VRObject{},
		Targets:     []*VRTarget{},
		Lighting:    s.generateLighting(difficulty),
		Camera:      s.generateCamera(difficulty),
		Physics:     true,
		Constraints: []VRConstraint{},
		Hints:       []string{},
	}

	switch captchaType {
	case VRARType3DPlacement:
		config.Objects = s.generatePlacementObjects(difficulty)
		config.Targets = s.generateTargets(difficulty, len(config.Objects))
		for i, obj := range config.Objects {
			if i < len(config.Targets) {
				config.Targets[i].ObjectID = obj.ID
				obj.TargetPosition = config.Targets[i].Position
				constraint := VRConstraint{
					Type:       "distance",
					TargetID:   obj.ID,
					Property:   "position",
					Tolerance:  s.getToleranceByDifficulty(difficulty),
					Weight:     0.4,
				}
				config.Constraints = append(config.Constraints, constraint)
			}
		}

	case VRARTypeGesture:
		config.Gestures = []string{"pinch", "point", "wave", "fist", "open_palm", "thumbs_up", "peace", "ok_sign"}
		config.TargetGesture = config.Gestures[rand.Intn(len(config.Gestures))]
		config.Objects = s.generateGestureHints(difficulty)

	case VRARTypeObjectRotation:
		config.Objects = s.generateRotationObjects(difficulty)
		for _, obj := range config.Objects {
			if obj.TargetRotation != nil {
				constraint := VRConstraint{
					Type:       "angle",
					TargetID:   obj.ID,
					Property:   "rotation",
					Tolerance:  s.getAngleToleranceByDifficulty(difficulty),
					Weight:     0.3,
				}
				config.Constraints = append(config.Constraints, constraint)
			}
		}

	case VRARTypeSpatialPuzzle:
		config.Objects = s.generatePuzzleObjects(difficulty)
		config.Targets = s.generatePuzzleTargets(difficulty, len(config.Objects))
		for i, obj := range config.Objects {
			if i < len(config.Targets) {
				config.Targets[i].ObjectID = obj.ID
				config.Targets[i].Sequence = i
				obj.TargetRotation = []float64{0, float64(rand.Intn(4)*90), 0}
				constraint := VRConstraint{
					Type:       "sequence",
					TargetID:   obj.ID,
					Property:   "order",
					Min:        float64(i),
					Max:        float64(i),
					Tolerance:  0,
					Weight:     0.2 / float64(len(config.Objects)),
				}
				config.Constraints = append(config.Constraints, constraint)
			}
		}

	case VRARTypeEyeTracking:
		config.Targets = s.generateEyeTrackingTargets(difficulty)

	case VRARTypeSequential:
		config.Objects = s.generateSequentialObjects(difficulty)
		for i := 0; i < len(config.Objects); i++ {
			constraint := VRConstraint{
				Type:       "sequence",
				TargetID:   config.Objects[i].ID,
				Property:   "order",
				Min:        float64(i),
				Max:        float64(i),
				Tolerance:  0,
				Weight:     0.2 / float64(len(config.Objects)),
			}
			config.Constraints = append(config.Constraints, constraint)
		}
	}

	return config
}

func (s *VRARGeneratorService) generateLighting(difficulty string) *VRLighting {
	return &VRLighting{
		AmbientIntensity:    0.4,
		DirectionalIntensity: 0.8,
		AmbientColor:        "#ffffff",
		DirectionalColor:    "#ffffff",
		ShadowEnabled:       difficulty != "easy",
	}
}

func (s *VRARGeneratorService) generateCamera(difficulty string) *VRCamera {
	return &VRCamera{
		Position: []float64{0, 1.6, 3},
		Rotation: []float64{0, 0, 0},
		FOV:      60,
		Near:     0.01,
		Far:      100,
	}
}

func (s *VRARGeneratorService) generatePlacementObjects(difficulty string) []*VRObject {
	shapes := []string{"cube", "sphere", "cylinder", "pyramid", "torus", "cone"}
	colors := []string{"#e74c3c", "#3498db", "#2ecc71", "#f39c12", "#9b59b6", "#1abc9c"}
	materials := []string{"plastic", "metal", "wood", "glass", "rubber"}

	objectCount := s.getObjectCountByDifficulty(difficulty)
	objects := make([]*VRObject, objectCount)

	for i := 0; i < objectCount; i++ {
		obj := &VRObject{
			ID:           fmt.Sprintf("vrar_obj_%d", i),
			Type:         shapes[rand.Intn(len(shapes))],
			Position:     []float64{float64(rand.Intn(3)-1), 0.5, float64(rand.Intn(3)-1)},
			Rotation:     []float64{0, float64(rand.Intn(360)), 0},
			Scale:        []float64{0.2, 0.2, 0.2},
			Color:        colors[i%len(colors)],
			Material:     materials[rand.Intn(len(materials))],
			Interactable: true,
			Grabbable:    true,
		}
		objects[i] = obj
	}

	return objects
}

func (s *VRARGeneratorService) generateTargets(difficulty string, count int) []*VRTarget {
	targets := make([]*VRTarget, count)
	colors := []string{"#ff0000", "#00ff00", "#0000ff", "#ffff00", "#ff00ff", "#00ffff"}

	for i := 0; i < count; i++ {
		target := &VRTarget{
			ID:        fmt.Sprintf("target_%d", i),
			Position:  []float64{float64(rand.Intn(3)-1), 0.1, float64(rand.Intn(3)-1) - 1.5},
			Size:      []float64{0.3, 0.3, 0.3},
			Shape:     "cube",
			Color:     colors[i%len(colors)],
			Visible:   difficulty == "easy" || difficulty == "medium",
		}
		targets[i] = target
	}

	return targets
}

func (s *VRARGeneratorService) generateGestureHints(difficulty string) []*VRObject {
	hints := []*VRObject{
		{
			ID:           "hint_board",
			Type:         "plane",
			Position:     []float64{0, 1.5, -1},
			Rotation:     []float64{0, 0, 0},
			Scale:        []float64{0.8, 0.5, 0.1},
			Color:        "#ffffff",
			Material:     "plastic",
			Interactable: false,
			Grabbable:    false,
		},
	}
	return hints
}

func (s *VRARGeneratorService) generateRotationObjects(difficulty string) []*VRObject {
	shapes := []string{"cube", "pyramid", "prism"}
	colors := []string{"#ff4444", "#44ff44", "#4444ff"}

	count := s.getObjectCountByDifficulty(difficulty)
	objects := make([]*VRObject, count)

	for i := 0; i < count; i++ {
		targetRotationY := float64(rand.Intn(4) * 90)
		obj := &VRObject{
			ID:             fmt.Sprintf("rot_obj_%d", i),
			Type:           shapes[i%len(shapes)],
			Position:       []float64{0, 0.2, -1},
			Rotation:       []float64{0, float64(rand.Intn(4)*90), 0},
			Scale:          []float64{0.3, 0.3, 0.3},
			Color:          colors[i%len(colors)],
			TargetRotation: []float64{0, targetRotationY, 0},
			Interactable:   true,
			Grabbable:      true,
		}
		objects[i] = obj
	}

	return objects
}

func (s *VRARGeneratorService) generatePuzzleObjects(difficulty string) []*VRObject {
	shapes := []string{"cube", "pyramid", "cylinder"}
	colors := []string{"#ff4444", "#44ff44", "#4444ff", "#ffff44", "#ff44ff"}

	count := s.getObjectCountByDifficulty(difficulty)
	objects := make([]*VRObject, count)

	for i := 0; i < count; i++ {
		obj := &VRObject{
			ID:           fmt.Sprintf("puzzle_%d", i),
			Type:         shapes[i%len(shapes)],
			Position:     []float64{float64(i)*0.5 - float64(count-1)*0.25, 0.3, 0},
			Rotation:     []float64{0, float64(rand.Intn(360)), 0},
			Scale:        []float64{0.25, 0.25, 0.25},
			Color:        colors[i%len(colors)],
			Material:     "plastic",
			Interactable: true,
			Grabbable:    true,
		}
		objects[i] = obj
	}

	return objects
}

func (s *VRARGeneratorService) generatePuzzleTargets(difficulty string, count int) []*VRTarget {
	targets := make([]*VRTarget, count)

	for i := 0; i < count; i++ {
		target := &VRTarget{
			ID:        fmt.Sprintf("puzzle_target_%d", i),
			Position:  []float64{float64(i)*0.5 - float64(count-1)*0.25, 0.05, -1},
			Size:      []float64{0.35, 0.35, 0.35},
			Shape:     "plane",
			Color:     "#cccccc",
			Visible:   true,
			Sequence:  i,
		}
		targets[i] = target
	}

	return targets
}

func (s *VRARGeneratorService) generateEyeTrackingTargets(difficulty string) []*VRTarget {
	count := 3 + s.getDifficultyLevel(difficulty)
	targets := make([]*VRTarget, count)

	for i := 0; i < count; i++ {
		target := &VRTarget{
			ID:        fmt.Sprintf("eye_target_%d", i),
			Position:  []float64{float64(rand.Intn(5)-2), float64(rand.Intn(3)), float64(rand.Intn(3)-2)},
			Size:      []float64{0.1, 0.1, 0.1},
			Shape:     "sphere",
			Color:     "#ff0000",
			Visible:   true,
			Sequence:  i,
		}
		targets[i] = target
	}

	return targets
}

func (s *VRARGeneratorService) generateSequentialObjects(difficulty string) []*VRObject {
	colors := []string{"#ff4444", "#44ff44", "#4444ff", "#ffff44", "#ff44ff"}
	sequenceLength := 2 + s.getDifficultyLevel(difficulty)
	objects := make([]*VRObject, sequenceLength)

	for i := 0; i < sequenceLength; i++ {
		obj := &VRObject{
			ID:           fmt.Sprintf("seq_obj_%d", i),
			Type:         "sphere",
			Position:     []float64{float64(i)*0.5, 0.3, 0},
			Rotation:     []float64{0, 0, 0},
			Scale:        []float64{0.15, 0.15, 0.15},
			Color:        colors[i%len(colors)],
			Material:     "plastic",
			Interactable: true,
			Grabbable:    false,
		}
		objects[i] = obj
	}

	return objects
}

func (s *VRARGeneratorService) generateInstructions(mode VRARMode, captchaType VRARType, difficulty string) string {
	modePrefix := ""
	if mode == VRARModeAR {
		modePrefix = "在AR环境中，"
	} else if mode == VRARModeHybrid {
		modePrefix = "在混合现实环境中，"
	}

	switch captchaType {
	case VRARType3DPlacement:
		if difficulty == "easy" {
			return modePrefix + "请抓取物体并放置到对应的目标区域"
		} else if difficulty == "medium" {
			return modePrefix + "请将所有物体按顺序放置到目标位置"
		}
		return modePrefix + "请精确地将每个物体放置到指定的目标位置"

	case VRARTypeGesture:
		return modePrefix + "请做出指定的手势动作"

	case VRARTypeObjectRotation:
		if difficulty == "easy" {
			return modePrefix + "请旋转物体使其朝向正确方向"
		}
		return modePrefix + "请将所有物体旋转到指定角度"

	case VRARTypeSpatialPuzzle:
		if difficulty == "easy" {
			return modePrefix + "请按顺序点击并放置拼图块"
		}
		return modePrefix + "请完成空间拼图，按正确顺序放置所有块"

	case VRARTypeEyeTracking:
		return modePrefix + "请按照提示用眼睛注视目标点"

	case VRARTypeSequential:
		if difficulty == "easy" {
			return modePrefix + "请按顺序点击物体"
		}
		return modePrefix + "请按颜色顺序点击物体"
	}

	return modePrefix + "请完成验证任务"
}

func (s *VRARGeneratorService) generateWebXRConfig(mode VRARMode, captchaType VRARType) *WebXRConfig {
	sessionMode := "immersive-vr"
	if mode == VRARModeAR || mode == VRARModeHybrid {
		sessionMode = "immersive-ar"
	}

	config := &WebXRConfig{
		RequiredFeatures:   []string{"local-floor"},
		OptionalFeatures:   []string{"hand-tracking", "hit-test"},
		ReferenceSpaceType: "local-floor",
		SessionMode:        sessionMode,
		HandTracking:       captchaType == VRARTypeGesture,
		EyeTracking:        captchaType == VRARTypeEyeTracking,
		HitTest:           true,
		PlaneDetection:    mode == VRARModeAR || mode == VRARModeHybrid,
	}

	if captchaType == VRARTypeEyeTracking {
		config.OptionalFeatures = append(config.OptionalFeatures, "eye-tracking")
	}

	return config
}

func (s *VRARGeneratorService) generateGestureConfig(mode VRARMode, captchaType VRARType, difficulty string) *VRGestureConfig {
	if captchaType != VRARTypeGesture {
		return nil
	}

	return &VRGestureConfig{
		SupportedGestures: []string{"pinch", "point", "wave", "fist", "open_palm"},
		TargetGesture:     "pinch",
		RequiredConfidence: 0.7,
		Hand:              "either",
	}
}

func (s *VRARGeneratorService) extractCorrectActions(config *VRARSceneConfig) []string {
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

	for _, target := range config.Targets {
		if target.Sequence >= 0 {
			actions = append(actions, fmt.Sprintf("sequence:%s:%d", target.ID, target.Sequence))
		}
	}

	return actions
}

func (s *VRARGeneratorService) getObjectCountByDifficulty(difficulty string) int {
	switch difficulty {
	case "easy":
		return 2
	case "medium":
		return 3
	case "hard":
		return 4
	case "expert":
		return 5
	default:
		return 3
	}
}

func (s *VRARGeneratorService) getDifficultyLevel(difficulty string) int {
	switch difficulty {
	case "easy":
		return 0
	case "medium":
		return 1
	case "hard":
		return 2
	case "expert":
		return 3
	default:
		return 1
	}
}

func (s *VRARGeneratorService) getToleranceByDifficulty(difficulty string) float64 {
	switch difficulty {
	case "easy":
		return 0.3
	case "medium":
		return 0.2
	case "hard":
		return 0.15
	case "expert":
		return 0.1
	default:
		return 0.2
	}
}

func (s *VRARGeneratorService) getAngleToleranceByDifficulty(difficulty string) float64 {
	switch difficulty {
	case "easy":
		return 30.0
	case "medium":
		return 20.0
	case "hard":
		return 15.0
	case "expert":
		return 10.0
	default:
		return 20.0
	}
}

func (s *VRARGeneratorService) generateSessionID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("vrar_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

func (s *VRARGeneratorService) MarshalSession(session *VRARSession) []byte {
	data, _ := json.Marshal(session)
	return data
}
