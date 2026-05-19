package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

var arObjectTypes = []string{
	"cube", "sphere", "cylinder", "cone", "torus",
	"pyramid", "diamond", "ring", "star", "heart",
}

var arTargetShapes = []string{
	"circle", "square", "triangle", "diamond", "star",
	"hexagon", "octagon", "cross", "arrow", "heart",
}

var arColors = []string{
	"#e74c3c", "#3498db", "#2ecc71", "#f39c12", "#9b59b6",
	"#1abc9c", "#e91e63", "#00bcd4", "#8bc34a", "#ff9800",
}

type ARScene struct {
	SessionID         string        `json:"sessionID"`
	TargetShape       string        `json:"targetShape"`
	TargetColor       string        `json:"targetColor"`
	Objects           []ARObject    `json:"objects"`
	TargetPosition    ARPosition    `json:"targetPosition"`
	GridSize          int           `json:"gridSize"`
	Difficulty        string        `json:"difficulty"`
	TimeLimit         int           `json:"timeLimit"`
	RequiredGesture   string        `json:"requiredGesture"`
	Environment       AREnvironment `json:"environment"`
}

type ARObject struct {
	ID           int        `json:"id"`
	Type         string     `json:"type"`
	Color        string     `json:"color"`
	Position     ARPosition `json:"position"`
	Rotation     ARRotation `json:"rotation"`
	Scale        float64    `json:"scale"`
	IsTarget     bool       `json:"isTarget"`
	Animation    ARAnimation `json:"animation"`
}

type ARPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type ARRotation struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type ARAnimation struct {
	Enabled bool    `json:"enabled"`
	Speed   float64 `json:"speed"`
	Type    string  `json:"type"`
}

type AREnvironment struct {
	Background string `json:"background"`
	Gravity    bool   `json:"gravity"`
	Lighting   string `json:"lighting"`
	FloorPlane bool   `json:"floorPlane"`
}

type CreateARRequest struct {
	Difficulty  string `json:"difficulty"`
	ClientIP    string `json:"clientIP"`
	UserAgent   string `json:"userAgent"`
	Fingerprint string `json:"fingerprint"`
}

type CreateARResponse struct {
	SessionID string   `json:"sessionID"`
	Scene     *ARScene `json:"scene"`
	ExpiresIn int64    `json:"expiresIn"`
	ExpiresAt int64    `json:"expiresAt"`
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
	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = "medium"
	}

	scene := s.generateScene(difficulty)

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	scene.SessionID = sessionID

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

func (s *ARGeneratorService) generateScene(difficulty string) *ARScene {
	rand.Seed(time.Now().UnixNano())

	gridSize := s.getGridSize(difficulty)
	targetShape := arTargetShapes[rand.Intn(len(arTargetShapes))]
	targetColor := arColors[rand.Intn(len(arColors))]

	objects := s.generateObjects(gridSize, difficulty, targetShape, targetColor)
	targetPosition := s.generateTargetPosition(gridSize)
	requiredGesture := s.getRequiredGesture(difficulty)

	environment := AREnvironment{
		Background: s.getBackground(difficulty),
		Gravity:    rand.Float64() > 0.3,
		Lighting:   s.getLighting(difficulty),
		FloorPlane: true,
	}

	timeLimit := s.getTimeLimit(difficulty)

	return &ARScene{
		TargetShape:     targetShape,
		TargetColor:     targetColor,
		Objects:         objects,
		TargetPosition:  targetPosition,
		GridSize:        gridSize,
		Difficulty:      difficulty,
		TimeLimit:       timeLimit,
		RequiredGesture: requiredGesture,
		Environment:     environment,
	}
}

func (s *ARGeneratorService) getGridSize(difficulty string) int {
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

func (s *ARGeneratorService) getTimeLimit(difficulty string) int {
	switch difficulty {
	case "easy":
		return 60
	case "medium":
		return 45
	case "hard":
		return 30
	case "expert":
		return 20
	default:
		return 45
	}
}

func (s *ARGeneratorService) getRequiredGesture(difficulty string) string {
	gestures := []string{"tap", "swipe", "pinch", "rotate"}
	if difficulty == "easy" {
		return gestures[0]
	}
	if difficulty == "medium" {
		return gestures[rand.Intn(2)]
	}
	return gestures[rand.Intn(len(gestures))]
}

func (s *ARGeneratorService) getBackground(difficulty string) string {
	backgrounds := []string{"ar", "gradient", "solid", "grid"}
	switch difficulty {
	case "easy":
		return "ar"
	case "medium":
		return backgrounds[rand.Intn(2)]
	default:
		return backgrounds[rand.Intn(len(backgrounds))]
	}
}

func (s *ARGeneratorService) getLighting(difficulty string) string {
	lightings := []string{"day", "night", "sunset", "studio"}
	switch difficulty {
	case "easy":
		return "day"
	default:
		return lightings[rand.Intn(len(lightings))]
	}
}

func (s *ARGeneratorService) generateObjects(gridSize int, difficulty, targetShape, targetColor string) []ARObject {
	objectCount := gridSize * gridSize
	objects := make([]ARObject, 0, objectCount)

	targetIndex := rand.Intn(objectCount)

	for i := 0; i < objectCount; i++ {
		isTarget := i == targetIndex
		objType := arObjectTypes[rand.Intn(len(arObjectTypes))]
		objColor := arColors[rand.Intn(len(arColors))]

		if isTarget {
			objColor = targetColor
		}

		object := ARObject{
			ID:     i,
			Type:   objType,
			Color:  objColor,
			IsTarget: isTarget,
			Position: ARPosition{
				X: float64(i%gridSize) - float64(gridSize-1)/2,
				Y: float64(i/gridSize) - float64(gridSize-1)/2,
				Z: s.getZOffset(difficulty),
			},
			Rotation: ARRotation{
				X: rand.Float64() * 360,
				Y: rand.Float64() * 360,
				Z: rand.Float64() * 360,
			},
			Scale: s.getScale(difficulty),
			Animation: ARAnimation{
				Enabled: difficulty != "easy" && rand.Float64() > 0.5,
				Speed:   s.getAnimationSpeed(difficulty),
				Type:    s.getAnimationType(),
			},
		}

		objects = append(objects, object)
	}

	return objects
}

func (s *ARGeneratorService) generateTargetPosition(gridSize int) ARPosition {
	rand.Seed(time.Now().UnixNano())
	return ARPosition{
		X: float64(rand.Intn(gridSize)) - float64(gridSize-1)/2,
		Y: float64(rand.Intn(gridSize)) - float64(gridSize-1)/2,
		Z: 0.5,
	}
}

func (s *ARGeneratorService) getZOffset(difficulty string) float64 {
	maxOffset := 0.0
	switch difficulty {
	case "easy":
		maxOffset = 0.1
	case "medium":
		maxOffset = 0.2
	case "hard":
		maxOffset = 0.3
	case "expert":
		maxOffset = 0.4
	}
	return (rand.Float64() - 0.5) * maxOffset * 2
}

func (s *ARGeneratorService) getScale(difficulty string) float64 {
	minScale, maxScale := 0.8, 1.0
	switch difficulty {
	case "expert":
		minScale, maxScale = 0.6, 1.2
	case "hard":
		minScale, maxScale = 0.7, 1.1
	}
	return minScale + rand.Float64()*(maxScale-minScale)
}

func (s *ARGeneratorService) getAnimationSpeed(difficulty string) float64 {
	switch difficulty {
	case "easy":
		return 0.0
	case "medium":
		return rand.Float64() * 0.005
	case "hard":
		return rand.Float64() * 0.01
	case "expert":
		return rand.Float64() * 0.015
	default:
		return 0.0
	}
}

func (s *ARGeneratorService) getAnimationType() string {
	types := []string{"rotate", "float", "pulse", "bounce"}
	return types[rand.Intn(len(types))]
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
