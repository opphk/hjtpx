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

var gestureTypes = []string{
	"tap", "swipe_left", "swipe_right", "swipe_up", "swipe_down",
	"circle", "triangle", "square", "pinch", "rotate",
}

var objectTypes = []string{
	"cube", "sphere", "pyramid", "cylinder", "cone",
	"torus", "star", "heart", "diamond", "ring",
}

var sceneTypes = []string{
	"object_placement", "gesture_recognition", "spatial_puzzle",
	"object_tracking", "depth_estimation",
}

type ARScene struct {
	SceneType     string       `json:"sceneType"`
	Objects       []ARObject   `json:"objects"`
	TargetObject  int          `json:"targetObject"`
	GesturePath   []GesturePoint `json:"gesturePath"`
	GestureType   string       `json:"gestureType"`
	Difficulty    string       `json:"difficulty"`
	CameraConfig  CameraConfig `json:"cameraConfig"`
	LightingConfig LightingConfig `json:"lightingConfig"`
	BackgroundColor string     `json:"backgroundColor"`
	Annotations   []Annotation `json:"annotations"`
	TimeLimit     int          `json:"timeLimit"`
	SuccessCriteria string     `json:"successCriteria"`
}

type ARObject struct {
	ID              int     `json:"id"`
	Type            string  `json:"type"`
	PositionX       float64 `json:"positionX"`
	PositionY       float64 `json:"positionY"`
	PositionZ       float64 `json:"positionZ"`
	RotationX       float64 `json:"rotationX"`
	RotationY       float64 `json:"rotationY"`
	RotationZ       float64 `json:"rotationZ"`
	Scale           float64 `json:"scale"`
	Color           string  `json:"color"`
	Texture         string  `json:"texture"`
	IsTarget        bool    `json:"isTarget"`
	Label           string  `json:"label"`
	Hidden          bool    `json:"hidden"`
	Animation       string  `json:"animation"`
	PhysicsEnabled  bool    `json:"physicsEnabled"`
	Depth           float64 `json:"depth"`
	Opacity         float64 `json:"opacity"`
	EmissiveColor   string  `json:"emissiveColor"`
}

type GesturePoint struct {
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Timestamp  int64   `json:"timestamp"`
	Pressure   float64 `json:"pressure"`
	GesturePhase string `json:"gesturePhase"`
}

type CameraConfig struct {
	FOV             float64 `json:"fov"`
	NearClip        float64 `json:"nearClip"`
	FarClip         float64 `json:"farClip"`
	PositionX       float64 `json:"positionX"`
	PositionY       float64 `json:"positionY"`
	PositionZ       float64 `json:"positionZ"`
	AutoRotate     bool    `json:"autoRotate"`
	RotationSpeed   float64 `json:"rotationSpeed"`
}

type LightingConfig struct {
	AmbientIntensity float64 `json:"ambientIntensity"`
	DirectionalIntensity float64 `json:"directionalIntensity"`
	PointIntensity   float64 `json:"pointIntensity"`
	ShadowEnabled    bool    `json:"shadowEnabled"`
	AmbientColor     string  `json:"ambientColor"`
}

type Annotation struct {
	ID        int     `json:"id"`
	Type      string  `json:"type"`
	PositionX float64 `json:"positionX"`
	PositionY float64 `json:"positionY"`
	Text      string  `json:"text"`
	Color     string  `json:"color"`
	Visible   bool    `json:"visible"`
}

type CreateARRequest struct {
	SceneType   string `json:"sceneType"`
	Difficulty  string `json:"difficulty"`
	ClientIP    string `json:"clientIP"`
	UserAgent   string `json:"userAgent"`
	Fingerprint string `json:"fingerprint"`
}

type CreateARResponse struct {
	SessionID  string   `json:"sessionID"`
	Scene      *ARScene `json:"scene"`
	ExpiresIn  int64    `json:"expiresIn"`
	ExpiresAt  int64    `json:"expiresAt"`
}

type ARGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

func NewARGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *ARGeneratorService {
	return &ARGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func NewARGeneratorServiceSimple() *ARGeneratorService {
	return &ARGeneratorService{}
}

func (s *ARGeneratorService) Create(ctx context.Context, req *CreateARRequest) (*CreateARResponse, error) {
	sceneType := req.SceneType
	if sceneType == "" {
		sceneType = sceneTypes[rand.Intn(len(sceneTypes))]
	}

	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = "medium"
	}

	scene := s.generateScene(sceneType, difficulty)

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	sceneData, err := json.Marshal(scene)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scene: %w", err)
	}

	session := &models.CaptchaSession{
		SessionID:     sessionID,
		BackgroundURL: string(sceneData),
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
			return nil, fmt.Errorf("failed to save session: %w", err)
		}
	}

	return &CreateARResponse{
		SessionID: sessionID,
		Scene:     scene,
		ExpiresIn: int64(5 * time.Minute / time.Second),
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

func (s *ARGeneratorService) generateScene(sceneType, difficulty string) *ARScene {
	scene := &ARScene{
		SceneType:     sceneType,
		Difficulty:    difficulty,
		CameraConfig:  s.generateCameraConfig(difficulty),
		LightingConfig: s.generateLightingConfig(difficulty),
		BackgroundColor: s.getBackgroundColor(),
		TimeLimit:    s.getTimeLimit(difficulty),
		SuccessCriteria: s.getSuccessCriteria(sceneType),
	}

	objectCount := s.getObjectCount(difficulty)
	objects := s.generateObjects(objectCount, difficulty)
	scene.Objects = objects

	for i := range scene.Objects {
		if scene.Objects[i].IsTarget {
			scene.TargetObject = scene.Objects[i].ID
			break
		}
	}

	scene.GesturePath = s.generateGesturePath(sceneType, difficulty)
	scene.GestureType = gestureTypes[rand.Intn(len(gestureTypes))]
	scene.Annotations = s.generateAnnotations(difficulty)

	return scene
}

func (s *ARGeneratorService) getObjectCount(difficulty string) int {
	switch difficulty {
	case "easy":
		return 3
	case "medium":
		return 5
	case "hard":
		return 7
	case "expert":
		return 10
	default:
		return 5
	}
}

func (s *ARGeneratorService) generateObjects(count int, difficulty string) []ARObject {
	objects := make([]ARObject, 0, count)
	targetIndex := rand.Intn(count)

	for i := 0; i < count; i++ {
		obj := ARObject{
			ID:            i,
			Type:          objectTypes[rand.Intn(len(objectTypes))],
			PositionX:     s.getRandomPosition(difficulty),
			PositionY:     s.getRandomPosition(difficulty),
			PositionZ:     s.getRandomDepth(difficulty),
			RotationX:     rand.Float64() * 360,
			RotationY:     rand.Float64() * 360,
			RotationZ:     rand.Float64() * 360,
			Scale:         s.getRandomScale(difficulty),
			Color:         s.getRandomColor(),
			Texture:       s.getRandomTexture(),
			IsTarget:      i == targetIndex,
			Label:         fmt.Sprintf("Object_%d", i),
			Hidden:        false,
			Animation:     s.getRandomAnimation(),
			PhysicsEnabled: rand.Float64() > 0.7,
			Depth:         s.getRandomDepth(difficulty),
			Opacity:       0.9 + rand.Float64()*0.1,
			EmissiveColor: s.getEmissiveColor(),
		}
		objects = append(objects, obj)
	}

	return objects
}

func (s *ARGeneratorService) getRandomPosition(difficulty string) float64 {
	rangeVal := 2.0
	switch difficulty {
	case "easy":
		rangeVal = 1.0
	case "medium":
		rangeVal = 1.5
	case "hard":
		rangeVal = 2.0
	case "expert":
		rangeVal = 2.5
	}
	return (rand.Float64() - 0.5) * rangeVal * 2
}

func (s *ARGeneratorService) getRandomDepth(difficulty string) float64 {
	base := 3.0
	switch difficulty {
	case "easy":
		base = 2.0
	case "medium":
		base = 3.0
	case "hard":
		base = 4.0
	case "expert":
		base = 5.0
	}
	return base + rand.Float64()*2
}

func (s *ARGeneratorService) getRandomScale(difficulty string) float64 {
	minScale, maxScale := 0.8, 1.2
	switch difficulty {
	case "expert":
		minScale, maxScale = 0.5, 1.5
	case "hard":
		minScale, maxScale = 0.6, 1.3
	}
	return minScale + rand.Float64()*(maxScale-minScale)
}

func (s *ARGeneratorService) getRandomColor() string {
	colors := []string{
		"#e74c3c", "#3498db", "#2ecc71", "#f39c12", "#9b59b6",
		"#1abc9c", "#e91e63", "#00bcd4", "#8bc34a", "#ff9800",
	}
	return colors[rand.Intn(len(colors))]
}

func (s *ARGeneratorService) getRandomTexture() string {
	textures := []string{
		"solid", "wireframe", "gradient", "striped", "grid",
	}
	return textures[rand.Intn(len(textures))]
}

func (s *ARGeneratorService) getRandomAnimation() string {
	animations := []string{
		"none", "rotate", "pulse", "bounce", "float",
	}
	return animations[rand.Intn(len(animations))]
}

func (s *ARGeneratorService) getEmissiveColor() string {
	if rand.Float64() > 0.7 {
		return s.getRandomColor()
	}
	return "#000000"
}

func (s *ARGeneratorService) getBackgroundColor() string {
	colors := []string{
		"#f5f5f5", "#e8e8e8", "#d0d0d0", "#c0c0c0", "#f0f0f0",
	}
	return colors[rand.Intn(len(colors))]
}

func (s *ARGeneratorService) generateGesturePath(sceneType, difficulty string) []GesturePoint {
	path := make([]GesturePoint, 0)
	pointCount := s.getGesturePointCount(difficulty)

	startX := rand.Float64() * 0.6 + 0.2
	startY := rand.Float64() * 0.6 + 0.2

	switch sceneType {
	case "gesture_recognition":
		path = s.generateGesturePattern(int(startX*10), int(startY*10), pointCount, difficulty)
	case "object_placement":
		path = s.generatePlacementPath(startX, startY, pointCount)
	case "spatial_puzzle":
		path = s.generateSpatialPath(startX, startY, pointCount)
	default:
		path = s.generateRandomPath(startX, startY, pointCount)
	}

	return path
}

func (s *ARGeneratorService) getGesturePointCount(difficulty string) int {
	switch difficulty {
	case "easy":
		return 10
	case "medium":
		return 20
	case "hard":
		return 30
	case "expert":
		return 40
	default:
		return 20
	}
}

func (s *ARGeneratorService) generateGesturePattern(startX, startY int, count int, difficulty string) []GesturePoint {
	path := make([]GesturePoint, count)
	gestureType := gestureTypes[rand.Intn(len(gestureTypes))]

	baseTime := time.Now().UnixMilli()

	for i := 0; i < count; i++ {
		t := float64(i) / float64(count-1)
		x, y := s.calculateGesturePoint(float64(startX), float64(startY), t, gestureType, difficulty)
		path[i] = GesturePoint{
			X:          x,
			Y:          y,
			Timestamp:  baseTime + int64(i*50),
			Pressure:   0.5 + rand.Float64()*0.5,
			GesturePhase: s.getGesturePhase(t),
		}
	}

	return path
}

func (s *ARGeneratorService) calculateGesturePoint(startX, startY, t float64, gestureType, difficulty string) (float64, float64) {
	amplitude := s.getGestureAmplitude(difficulty)
	
	switch gestureType {
	case "tap":
		return startX, startY
	case "swipe_left":
		return startX - t*amplitude, startY
	case "swipe_right":
		return startX + t*amplitude, startY
	case "swipe_up":
		return startX, startY + t*amplitude
	case "swipe_down":
		return startX, startY - t*amplitude
	case "circle":
		angle := t * 2 * math.Pi
		return startX + amplitude*math.Cos(angle)*0.5, startY + amplitude*math.Sin(angle)*0.5
	case "triangle":
		return s.calculatePolygonPoint(startX, startY, t, 3, amplitude)
	case "square":
		return s.calculatePolygonPoint(startX, startY, t, 4, amplitude)
	case "pinch":
		return startX + (0.5-t)*amplitude*0.3, startY
	case "rotate":
		angle := t * 2 * math.Pi
		return startX + amplitude*math.Cos(angle)*0.3, startY + amplitude*math.Sin(angle)*0.3
	default:
		return startX + (rand.Float64()-0.5)*amplitude, startY + (rand.Float64()-0.5)*amplitude
	}
}

func (s *ARGeneratorService) calculatePolygonPoint(centerX, centerY, t float64, sides int, amplitude float64) (float64, float64) {
	angle := t * 2 * math.Pi * float64(sides)
	return centerX + amplitude*math.Cos(angle)*0.5, centerY + amplitude*math.Sin(angle)*0.5
}

func (s *ARGeneratorService) getGestureAmplitude(difficulty string) float64 {
	switch difficulty {
	case "easy":
		return 0.3
	case "medium":
		return 0.5
	case "hard":
		return 0.7
	case "expert":
		return 0.9
	default:
		return 0.5
	}
}

func (s *ARGeneratorService) getGesturePhase(t float64) string {
	if t < 0.1 {
		return "start"
	} else if t > 0.9 {
		return "end"
	}
	return "middle"
}

func (s *ARGeneratorService) generatePlacementPath(startX, startY float64, count int) []GesturePoint {
	path := make([]GesturePoint, count)
	baseTime := time.Now().UnixMilli()

	targetX := rand.Float64()*0.4 + 0.3
	targetY := rand.Float64()*0.4 + 0.3

	for i := 0; i < count; i++ {
		t := float64(i) / float64(count-1)
		path[i] = GesturePoint{
			X:          startX + (targetX-startX)*t,
			Y:          startY + (targetY-startY)*t,
			Timestamp:  baseTime + int64(i*50),
			Pressure:   0.7,
			GesturePhase: s.getGesturePhase(t),
		}
	}

	return path
}

func (s *ARGeneratorService) generateSpatialPath(startX, startY float64, count int) []GesturePoint {
	path := make([]GesturePoint, count)
	baseTime := time.Now().UnixMilli()

	for i := 0; i < count; i++ {
		t := float64(i) / float64(count-1)
		path[i] = GesturePoint{
			X:          startX + math.Sin(t*4*math.Pi)*0.2,
			Y:          startY + t*0.5,
			Timestamp:  baseTime + int64(i*50),
			Pressure:   0.5 + math.Sin(t*math.Pi)*0.3,
			GesturePhase: s.getGesturePhase(t),
		}
	}

	return path
}

func (s *ARGeneratorService) generateRandomPath(startX, startY float64, count int) []GesturePoint {
	path := make([]GesturePoint, count)
	baseTime := time.Now().UnixMilli()

	for i := 0; i < count; i++ {
		t := float64(i) / float64(count-1)
		path[i] = GesturePoint{
			X:          startX + (rand.Float64()-0.5)*0.2,
			Y:          startY + (rand.Float64()-0.5)*0.2,
			Timestamp:  baseTime + int64(i*50),
			Pressure:   0.5 + rand.Float64()*0.5,
			GesturePhase: s.getGesturePhase(t),
		}
	}

	return path
}

func (s *ARGeneratorService) generateAnnotations(difficulty string) []Annotation {
	count := 0
	switch difficulty {
	case "easy":
		count = 1
	case "medium":
		count = 2
	case "hard":
		count = 3
	case "expert":
		count = 4
	}

	annotations := make([]Annotation, 0, count)
	types := []string{"circle", "arrow", "text"}

	for i := 0; i < count; i++ {
		annotation := Annotation{
			ID:        i,
			Type:      types[rand.Intn(len(types))],
			PositionX: rand.Float64(),
			PositionY: rand.Float64(),
			Text:      fmt.Sprintf("Hint %d", i+1),
			Color:     s.getRandomColor(),
			Visible:   true,
		}
		annotations = append(annotations, annotation)
	}

	return annotations
}

func (s *ARGeneratorService) generateCameraConfig(difficulty string) CameraConfig {
	fov := 60.0
	switch difficulty {
	case "easy":
		fov = 50.0
	case "medium":
		fov = 60.0
	case "hard":
		fov = 70.0
	case "expert":
		fov = 80.0
	}

	return CameraConfig{
		FOV:           fov,
		NearClip:      0.1,
		FarClip:       100.0,
		PositionX:     0,
		PositionY:     0,
		PositionZ:     5,
		AutoRotate:    rand.Float64() > 0.5,
		RotationSpeed: 0.01 + rand.Float64()*0.02,
	}
}

func (s *ARGeneratorService) generateLightingConfig(difficulty string) LightingConfig {
	intensity := 0.8
	switch difficulty {
	case "easy":
		intensity = 0.6
	case "medium":
		intensity = 0.8
	case "hard":
		intensity = 1.0
	case "expert":
		intensity = 1.2
	}

	return LightingConfig{
		AmbientIntensity:     intensity * 0.5,
		DirectionalIntensity: intensity,
		PointIntensity:       intensity * 0.3,
		ShadowEnabled:        difficulty == "hard" || difficulty == "expert",
		AmbientColor:         "#444444",
	}
}

func (s *ARGeneratorService) getTimeLimit(difficulty string) int {
	switch difficulty {
	case "easy":
		return 30
	case "medium":
		return 20
	case "hard":
		return 15
	case "expert":
		return 10
	default:
		return 20
	}
}

func (s *ARGeneratorService) getSuccessCriteria(sceneType string) string {
	switch sceneType {
	case "gesture_recognition":
		return "accurate_gesture"
	case "object_placement":
		return "correct_position"
	case "spatial_puzzle":
		return "spatial_accuracy"
	case "object_tracking":
		return "tracking_completeness"
	case "depth_estimation":
		return "depth_accuracy"
	default:
		return "gesture_accuracy"
	}
}

func (s *ARGeneratorService) GetSession(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	if s.sessionCache != nil {
		session, err := s.sessionCache.Get(ctx, sessionID)
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

func (s *ARGeneratorService) DeleteSession(ctx context.Context, sessionID string) error {
	if s.sessionCache != nil {
		if err := s.sessionCache.Delete(ctx, sessionID); err != nil {
			return err
		}
	}

	if s.captchaRepo != nil {
		if err := s.captchaRepo.Delete(sessionID); err != nil {
			return err
		}
	}

	return nil
}
