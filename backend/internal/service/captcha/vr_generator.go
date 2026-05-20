package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
)

type VRMode string

const (
	VRModeInteractive VRMode = "interactive"
	VRModePuzzle      VRMode = "puzzle"
	VRModeGesture     VRMode = "gesture"
	VRModeNavigation  VRMode = "navigation"
	VRModeSequence    VRMode = "sequence"
)

type VRCaptchaType string

const (
	VRCaptcha3DPlacement  VRCaptchaType = "3d_placement"
	VRCaptchaHandTracking VRCaptchaType = "hand_tracking"
	VRCaptchaEyeTracking  VRCaptchaType = "eye_tracking"
	VRCaptchaSpatialPuzzle VRCaptchaType = "spatial_puzzle"
	VRCaptchaVRGesture    VRCaptchaType = "vr_gesture"
)

type VRCaptchaRequest struct {
	Mode        VRMode        `json:"mode"`
	Type        VRCaptchaType `json:"type"`
	Difficulty  string        `json:"difficulty"`
	ClientIP    string        `json:"client_ip"`
	UserAgent   string        `json:"user_agent"`
	Fingerprint string        `json:"fingerprint"`
}

type VRCaptchaResponse struct {
	SessionID     string          `json:"session_id"`
	VRConfig      *VRSceneConfig  `json:"vr_config"`
	Instructions  string          `json:"instructions"`
	WebXRConfig   *WebXRConfig    `json:"webxr_config"`
	GestureConfig *VRGestureConfig `json:"gesture_config,omitempty"`
	ExpiresIn     int64           `json:"expires_in"`
	ExpiresAt     int64           `json:"expires_at"`
}

type VRVerifyRequest struct {
	SessionID    string                `json:"session_id"`
	Interaction  *VRInteractionData   `json:"interaction"`
	GestureData  *VRHandGestureData   `json:"gesture_data,omitempty"`
	EyeData      *VREyeTrackingData   `json:"eye_data,omitempty"`
	BehaviorData map[string]interface{} `json:"behavior_data,omitempty"`
	TraceData    interface{}           `json:"trace_data,omitempty"`
}

type VRVerifyResponse struct {
	Success   bool      `json:"success"`
	Score     float64   `json:"score"`
	Message   string    `json:"message"`
	Accuracy  float64   `json:"accuracy"`
	Feedback  string    `json:"feedback,omitempty"`
	Analytics *VRAnalytics `json:"analytics,omitempty"`
}

type VRSceneConfig struct {
	Mode         VRMode         `json:"mode"`
	Type         VRCaptchaType  `json:"type"`
	Environment  string         `json:"environment"`
	Objects      []*VRObject    `json:"objects"`
	Targets      []*VRTarget    `json:"targets,omitempty"`
	Gestures     []string       `json:"gestures,omitempty"`
	TargetGesture string        `json:"target_gesture,omitempty"`
	Lighting     *VRLighting    `json:"lighting"`
	Camera       *VRCamera      `json:"camera"`
	Physics      bool           `json:"physics"`
	Constraints  []VRConstraint `json:"constraints,omitempty"`
	Hints        []string       `json:"hints,omitempty"`
}

type VRObject struct {
	ID             string      `json:"id"`
	Type           string      `json:"type"`
	Position       []float64   `json:"position"`
	Rotation       []float64   `json:"rotation"`
	Scale          []float64   `json:"scale"`
	Color          string      `json:"color"`
	Material       string      `json:"material"`
	Interactable   bool        `json:"interactable"`
	Grabbable      bool        `json:"grabbable"`
	TargetPosition []float64   `json:"target_position,omitempty"`
	TargetRotation []float64   `json:"target_rotation,omitempty"`
	Animation      string      `json:"animation,omitempty"`
	Texture        string      `json:"texture,omitempty"`
}

type VRTarget struct {
	ID        string    `json:"id"`
	Position  []float64 `json:"position"`
	Size      []float64 `json:"size"`
	Shape     string    `json:"shape"`
	Color     string    `json:"color"`
	Visible   bool      `json:"visible"`
	ObjectID  string    `json:"object_id,omitempty"`
	Sequence  int       `json:"sequence,omitempty"`
}

type VRLighting struct {
	AmbientIntensity    float64 `json:"ambient_intensity"`
	DirectionalIntensity float64 `json:"directional_intensity"`
	AmbientColor        string  `json:"ambient_color"`
	DirectionalColor    string  `json:"directional_color"`
	ShadowEnabled       bool    `json:"shadow_enabled"`
}

type VRCamera struct {
	Position []float64 `json:"position"`
	Rotation []float64 `json:"rotation"`
	FOV      float64   `json:"fov"`
	Near     float64   `json:"near"`
	Far      float64   `json:"far"`
}

type VRConstraint struct {
	Type     string  `json:"type"`
	TargetID string  `json:"target_id"`
	Property string  `json:"property"`
	Min      float64 `json:"min,omitempty"`
	Max      float64 `json:"max,omitempty"`
	Tolerance float64 `json:"tolerance"`
	Weight   float64 `json:"weight"`
}

type WebXRConfig struct {
	RequiredFeatures   []string `json:"required_features"`
	OptionalFeatures   []string `json:"optional_features"`
	ReferenceSpaceType string   `json:"reference_space_type"`
	SessionMode        string   `json:"session_mode"`
	HandTracking       bool     `json:"hand_tracking"`
	EyeTracking        bool     `json:"eye_tracking"`
	HitTest           bool     `json:"hit_test"`
	PlaneDetection    bool     `json:"plane_detection"`
}

type VRGestureConfig struct {
	SupportedGestures []string       `json:"supported_gestures"`
	TargetGesture     string         `json:"target_gesture"`
	RequiredConfidence float64      `json:"required_confidence"`
	Hand              string         `json:"hand"`
}

type VRInteractionData struct {
	ObjectPositions  map[string][]float64 `json:"object_positions"`
	ObjectRotations  map[string][]float64 `json:"object_rotations"`
	CompletionOrder  []string            `json:"completion_order,omitempty"`
	TimeSpent        float64             `json:"time_spent"`
	MovementCount    int                 `json:"movement_count"`
}

type VRHandGestureData struct {
	Hand            string          `json:"hand"`
	GestureType     string          `json:"gesture_type"`
	JointPositions  [][]float64     `json:"joint_positions"`
	Confidence      float64         `json:"confidence"`
	Duration        int             `json:"duration"`
	Recognized      bool            `json:"recognized"`
	FingerPositions map[string][]float64 `json:"finger_positions,omitempty"`
}

type VREyeTrackingData struct {
	GazePosition   []float64 `json:"gaze_position"`
	GazeDirection  []float64 `json:"gaze_direction"`
	PupilDiameter  float64   `json:"pupil_diameter"`
	BlinkCount     int       `json:"blink_count"`
	FixationPoints [][]float64 `json:"fixation_points"`
	Confidence     float64   `json:"confidence"`
}

type VRAnalytics struct {
	CompletionTime float64 `json:"completion_time"`
	MovementCount  int     `json:"movement_count"`
	ErrorCount     int     `json:"error_count"`
	Accuracy       float64 `json:"accuracy"`
	HandDominance  string  `json:"hand_dominance,omitempty"`
	EyePattern     string  `json:"eye_pattern,omitempty"`
}

type VRSession struct {
	SessionID     string                `json:"session_id"`
	VRConfig      *VRSceneConfig        `json:"vr_config"`
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

type VRTargetResult struct {
	TargetID      string    `json:"target_id"`
	ObjectID      string    `json:"object_id"`
	FinalPosition []float64 `json:"final_position"`
	FinalRotation []float64 `json:"final_rotation"`
	Distance      float64   `json:"distance"`
	AngleDiff     float64   `json:"angle_diff"`
	Score         float64   `json:"score"`
	Success       bool      `json:"success"`
	Sequence      int       `json:"sequence,omitempty"`
}

type VRGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

func NewVRGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *VRGeneratorService {
	return &VRGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func NewVRGeneratorServiceSimple() *VRGeneratorService {
	return &VRGeneratorService{}
}

func (s *VRGeneratorService) Generate(ctx context.Context, req *VRCaptchaRequest) (*VRCaptchaResponse, error) {
	if req.Mode == "" {
		req.Mode = VRModeInteractive
	}
	if req.Type == "" {
		req.Type = VRCaptcha3DPlacement
	}
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}

	sessionID := generateVRSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	vrConfig := s.generateVRSceneConfig(req.Mode, req.Type, req.Difficulty)
	instructions := s.generateInstructions(req.Mode, req.Type, req.Difficulty)
	webXRConfig := s.generateWebXRConfig(req.Mode, req.Type)
	gestureConfig := s.generateGestureConfig(req.Mode, req.Type, req.Difficulty)

	correctActions := s.extractCorrectActions(vrConfig)

	session := &VRSession{
		SessionID:     sessionID,
		VRConfig:      vrConfig,
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

	return &VRCaptchaResponse{
		SessionID:     sessionID,
		VRConfig:      vrConfig,
		Instructions:  instructions,
		WebXRConfig:   webXRConfig,
		GestureConfig: gestureConfig,
		ExpiresIn:     int64(5 * time.Minute / time.Second),
		ExpiresAt:     expiresAt.Unix(),
	}, nil
}

func (s *VRGeneratorService) GetSession(ctx context.Context, sessionID string) (*VRSession, error) {
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

func (s *VRGeneratorService) saveSession(ctx context.Context, session *VRSession) error {
	if s.sessionCache != nil {
		data, err := json.Marshal(session)
		if err != nil {
			return err
		}
		return s.sessionCache.SetRaw(ctx, session.SessionID, string(data), 5*time.Minute)
	}
	return nil
}

func (s *VRGeneratorService) getCachedSession(ctx context.Context, sessionID string) (*VRSession, error) {
	return nil, fmt.Errorf("session not found in cache")
}

func (s *VRGeneratorService) getDatabaseSession(sessionID string) (*VRSession, error) {
	return nil, fmt.Errorf("session not found in database")
}

func (s *VRGeneratorService) generateVRSceneConfig(mode VRMode, captchaType VRCaptchaType, difficulty string) *VRSceneConfig {
	rand.Seed(time.Now().UnixNano())

	config := &VRSceneConfig{
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
	case VRCaptcha3DPlacement:
		config.Objects = s.generatePlacementObjects(difficulty)
		config.Targets = s.generateTargets(difficulty, len(config.Objects))
		for i, obj := range config.Objects {
			if i < len(config.Targets) {
				config.Targets[i].ObjectID = obj.ID
				obj.TargetPosition = config.Targets[i].Position
				constraint := VRConstraint{
					Type:      "distance",
					TargetID:  obj.ID,
					Property:  "position",
					Tolerance: s.getToleranceByDifficulty(difficulty),
					Weight:    0.4,
				}
				config.Constraints = append(config.Constraints, constraint)
			}
		}

	case VRCaptchaHandTracking, VRCaptchaVRGesture:
		config.Gestures = []string{"pinch", "point", "wave", "fist", "open_palm", "thumbs_up", "peace", "ok_sign"}
		config.TargetGesture = config.Gestures[rand.Intn(len(config.Gestures))]
		config.Objects = s.generateGestureHints(difficulty)

	case VRCaptchaSpatialPuzzle:
		config.Objects = s.generatePuzzleObjects(difficulty)
		config.Targets = s.generatePuzzleTargets(difficulty, len(config.Objects))
		for i, obj := range config.Objects {
			if i < len(config.Targets) {
				config.Targets[i].ObjectID = obj.ID
				config.Targets[i].Sequence = i
				obj.TargetRotation = []float64{0, float64(rand.Intn(4)*90), 0}
				constraint := VRConstraint{
					Type:      "sequence",
					TargetID:  obj.ID,
					Property:  "order",
					Min:       float64(i),
					Max:       float64(i),
					Tolerance: 0,
					Weight:    0.3 / float64(len(config.Objects)),
				}
				config.Constraints = append(config.Constraints, constraint)
			}
		}

	case VRCaptchaEyeTracking:
		config.Targets = s.generateEyeTrackingTargets(difficulty)
	}

	return config
}

func (s *VRGeneratorService) generateLighting(difficulty string) *VRLighting {
	return &VRLighting{
		AmbientIntensity:    0.4,
		DirectionalIntensity: 0.8,
		AmbientColor:        "#ffffff",
		DirectionalColor:    "#ffffff",
		ShadowEnabled:       difficulty != "easy",
	}
}

func (s *VRGeneratorService) generateCamera(difficulty string) *VRCamera {
	return &VRCamera{
		Position: []float64{0, 1.6, 3},
		Rotation: []float64{0, 0, 0},
		FOV:      60,
		Near:     0.01,
		Far:      100,
	}
}

func (s *VRGeneratorService) generatePlacementObjects(difficulty string) []*VRObject {
	shapes := []string{"cube", "sphere", "cylinder", "pyramid", "torus", "cone"}
	colors := []string{"#e74c3c", "#3498db", "#2ecc71", "#f39c12", "#9b59b6", "#1abc9c"}
	materials := []string{"plastic", "metal", "wood", "glass", "rubber"}

	objectCount := s.getObjectCountByDifficulty(difficulty)
	objects := make([]*VRObject, objectCount)

	for i := 0; i < objectCount; i++ {
		obj := &VRObject{
			ID:           fmt.Sprintf("vr_obj_%d", i),
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

func (s *VRGeneratorService) generateTargets(difficulty string, count int) []*VRTarget {
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

func (s *VRGeneratorService) generateGestureHints(difficulty string) []*VRObject {
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

func (s *VRGeneratorService) generatePuzzleObjects(difficulty string) []*VRObject {
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

func (s *VRGeneratorService) generatePuzzleTargets(difficulty string, count int) []*VRTarget {
	targets := make([]*VRTarget, count)

	for i := 0; i < count; i++ {
		target := &VRTarget{
			ID:       fmt.Sprintf("puzzle_target_%d", i),
			Position: []float64{float64(i)*0.5 - float64(count-1)*0.25, 0.05, -1},
			Size:     []float64{0.35, 0.35, 0.35},
			Shape:    "plane",
			Color:    "#cccccc",
			Visible:  true,
			Sequence: i,
		}
		targets[i] = target
	}

	return targets
}

func (s *VRGeneratorService) generateEyeTrackingTargets(difficulty string) []*VRTarget {
	count := 3 + s.getDifficultyLevel(difficulty)
	targets := make([]*VRTarget, count)

	for i := 0; i < count; i++ {
		target := &VRTarget{
			ID:       fmt.Sprintf("eye_target_%d", i),
			Position: []float64{float64(rand.Intn(5)-2), float64(rand.Intn(3)), float64(rand.Intn(3)-2)},
			Size:     []float64{0.1, 0.1, 0.1},
			Shape:    "sphere",
			Color:    "#ff0000",
			Visible:  true,
			Sequence: i,
		}
		targets[i] = target
	}

	return targets
}

func (s *VRGeneratorService) generateInstructions(mode VRMode, captchaType VRCaptchaType, difficulty string) string {
	switch captchaType {
	case VRCaptcha3DPlacement:
		if difficulty == "easy" {
			return "请抓取物体并放置到对应的目标区域"
		} else if difficulty == "medium" {
			return "请将所有物体按顺序放置到目标位置"
		}
		return "请精确地将每个物体放置到指定的目标位置"

	case VRCaptchaHandTracking, VRCaptchaVRGesture:
		return "请做出指定的手势动作"

	case VRCaptchaSpatialPuzzle:
		if difficulty == "easy" {
			return "请按顺序点击并放置拼图块"
		}
		return "请完成空间拼图，按正确顺序放置所有块"

	case VRCaptchaEyeTracking:
		return "请按照提示用眼睛注视目标点"
	}

	return "请完成VR验证任务"
}

func (s *VRGeneratorService) generateWebXRConfig(mode VRMode, captchaType VRCaptchaType) *WebXRConfig {
	config := &WebXRConfig{
		RequiredFeatures:   []string{"local-floor"},
		OptionalFeatures:   []string{"hand-tracking", "hit-test"},
		ReferenceSpaceType: "local-floor",
		SessionMode:        "immersive-vr",
		HandTracking:       captchaType == VRCaptchaHandTracking || captchaType == VRCaptchaVRGesture,
		EyeTracking:        captchaType == VRCaptchaEyeTracking,
		HitTest:           true,
		PlaneDetection:    false,
	}

	if captchaType == VRCaptchaEyeTracking {
		config.OptionalFeatures = append(config.OptionalFeatures, "eye-tracking")
	}

	return config
}

func (s *VRGeneratorService) generateGestureConfig(mode VRMode, captchaType VRCaptchaType, difficulty string) *VRGestureConfig {
	if captchaType != VRCaptchaHandTracking && captchaType != VRCaptchaVRGesture {
		return nil
	}

	return &VRGestureConfig{
		SupportedGestures: []string{"pinch", "point", "wave", "fist", "open_palm"},
		TargetGesture:     "pinch",
		RequiredConfidence: 0.7,
		Hand:              "either",
	}
}

func (s *VRGeneratorService) extractCorrectActions(config *VRSceneConfig) []string {
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

func (s *VRGeneratorService) getObjectCountByDifficulty(difficulty string) int {
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

func (s *VRGeneratorService) getDifficultyLevel(difficulty string) int {
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

func (s *VRGeneratorService) getToleranceByDifficulty(difficulty string) float64 {
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

func generateVRSessionID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("vr_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}
