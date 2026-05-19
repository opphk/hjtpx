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

var pieceColors = []string{
	"#e74c3c", "#3498db", "#2ecc71", "#f39c12", "#9b59b6",
	"#1abc9c", "#e91e63", "#00bcd4", "#8bc34a", "#ff9800",
	"#607d8b", "#ff5722", "#795548", "#673ab7", "#009688",
}

var pieceTypes = []string{
	"cube", "cylinder", "sphere", "cone", "torus",
	"octahedron", "tetrahedron", "dodecahedron", "icosahedron", "box",
}

var edgeTypes = []string{"smooth", "sharp", "rounded"}

type ThreeDPuzzle struct {
	Pieces         []ThreeDPiece `json:"pieces"`
	GridSize       int           `json:"gridSize"`
	Difficulty     string        `json:"difficulty"`
	TargetRotX     float64       `json:"targetRotX"`
	TargetRotY     float64       `json:"targetRotY"`
	TargetRotZ     float64       `json:"targetRotZ"`
	LightIntensity float64       `json:"lightIntensity"`
	AmbientColor   string        `json:"ambientColor"`
	BackgroundColor string       `json:"backgroundColor"`
	AntiAlias      bool          `json:"antiAlias"`
	ShadowEnabled  bool          `json:"shadowEnabled"`
	RenderQuality  string        `json:"renderQuality"`
}

type ThreeDPiece struct {
	ID              int     `json:"id"`
	Type            string  `json:"type"`
	Color           string  `json:"color"`
	PositionX       float64 `json:"positionX"`
	PositionY       float64 `json:"positionY"`
	PositionZ       float64 `json:"positionZ"`
	RotationX       float64 `json:"rotationX"`
	RotationY       float64 `json:"rotationY"`
	RotationZ       float64 `json:"rotationZ"`
	Scale           float64 `json:"scale"`
	EdgeType        string  `json:"edgeType"`
	Opacity         float64 `json:"opacity"`
	Shininess       int     `json:"shininess"`
	EmissiveColor   string  `json:"emissiveColor"`
	Wireframe       bool    `json:"wireframe"`
	OriginalRotX    float64 `json:"originalRotX"`
	OriginalRotY    float64 `json:"originalRotY"`
	OriginalRotZ    float64 `json:"originalRotZ"`
	TargetRotation  bool    `json:"targetRotation"`
	AnimationSpeed  float64 `json:"animationSpeed"`
}

type CreateThreeDRequest struct {
	Difficulty  string `json:"difficulty"`
	ClientIP    string `json:"clientIP"`
	UserAgent   string `json:"userAgent"`
	Fingerprint string `json:"fingerprint"`
}

type CreateThreeDResponse struct {
	SessionID string        `json:"sessionID"`
	Puzzle    *ThreeDPuzzle `json:"puzzle"`
	ExpiresIn int64         `json:"expiresIn"`
	ExpiresAt int64         `json:"expiresAt"`
}

type ThreeDGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

func NewThreeDGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *ThreeDGeneratorService {
	return &ThreeDGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func NewThreeDGeneratorServiceSimple() *ThreeDGeneratorService {
	return &ThreeDGeneratorService{}
}

func (s *ThreeDGeneratorService) Create(ctx context.Context, req *CreateThreeDRequest) (*CreateThreeDResponse, error) {
	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = "medium"
	}

	gridSize := s.getGridSizeByDifficulty(difficulty)
	puzzle := s.generatePuzzle(gridSize, difficulty)

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	puzzleData, err := json.Marshal(puzzle)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal puzzle: %w", err)
	}

	session := &models.CaptchaSession{
		SessionID:     sessionID,
		BackgroundURL: string(puzzleData),
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

	return &CreateThreeDResponse{
		SessionID: sessionID,
		Puzzle:    puzzle,
		ExpiresIn: int64(5 * time.Minute / time.Second),
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

func (s *ThreeDGeneratorService) getGridSizeByDifficulty(difficulty string) int {
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

func (s *ThreeDGeneratorService) generatePuzzle(gridSize int, difficulty string) *ThreeDPuzzle {
	rand.Seed(time.Now().UnixNano())

	targetRotX := rand.Float64() * 360
	targetRotY := rand.Float64() * 360
	targetRotZ := rand.Float64() * 360

	pieceCount := gridSize * gridSize
	pieces := make([]ThreeDPiece, 0, pieceCount)

	for i := 0; i < pieceCount; i++ {
		rotationX, rotationY, rotationZ := s.generateRotations(difficulty)
		
		piece := ThreeDPiece{
			ID:             i,
			Type:           pieceTypes[rand.Intn(len(pieceTypes))],
			Color:          pieceColors[rand.Intn(len(pieceColors))],
			PositionX:      float64(i%gridSize) - float64(gridSize-1)/2,
			PositionY:      float64(i/gridSize) - float64(gridSize-1)/2,
			PositionZ:      s.getRandomZOffset(difficulty),
			RotationX:      rotationX,
			RotationY:      rotationY,
			RotationZ:      rotationZ,
			Scale:          s.getRandomScale(difficulty),
			EdgeType:       edgeTypes[rand.Intn(len(edgeTypes))],
			Opacity:        s.getRandomOpacity(),
			Shininess:      rand.Intn(100) + 50,
			EmissiveColor:  s.getEmissiveColor(),
			Wireframe:      rand.Float64() > 0.9,
			OriginalRotX:   rotationX,
			OriginalRotY:   rotationY,
			OriginalRotZ:   rotationZ,
			TargetRotation: rand.Float64() > 0.5,
			AnimationSpeed: s.getAnimationSpeed(difficulty),
		}

		pieces = append(pieces, piece)
	}

	renderQuality := s.getRenderQuality(difficulty)

	return &ThreeDPuzzle{
		Pieces:         pieces,
		GridSize:       gridSize,
		Difficulty:     difficulty,
		TargetRotX:     targetRotX,
		TargetRotY:     targetRotY,
		TargetRotZ:     targetRotZ,
		LightIntensity: 0.8 + rand.Float64()*0.4,
		AmbientColor:   "#444444",
		BackgroundColor: "#f5f5f5",
		AntiAlias:      renderQuality != "low",
		ShadowEnabled:  renderQuality == "high",
		RenderQuality:  renderQuality,
	}
}

func (s *ThreeDGeneratorService) getRenderQuality(difficulty string) string {
	switch difficulty {
	case "easy":
		return "high"
	case "medium":
		return "medium"
	case "hard":
		return "medium"
	case "expert":
		return "low"
	default:
		return "medium"
	}
}

func (s *ThreeDGeneratorService) generateRotations(difficulty string) (float64, float64, float64) {
	switch difficulty {
	case "easy":
		return rand.Float64() * 90, rand.Float64() * 90, 0
	case "medium":
		return rand.Float64() * 180, rand.Float64() * 180, rand.Float64() * 90
	case "hard":
		return rand.Float64() * 360, rand.Float64() * 360, rand.Float64() * 180
	case "expert":
		return rand.Float64() * 360, rand.Float64() * 360, rand.Float64() * 360
	default:
		return rand.Float64() * 180, rand.Float64() * 180, rand.Float64() * 90
	}
}

func (s *ThreeDGeneratorService) getRandomZOffset(difficulty string) float64 {
	maxOffset := 0.0
	switch difficulty {
	case "easy":
		maxOffset = 0.2
	case "medium":
		maxOffset = 0.4
	case "hard":
		maxOffset = 0.6
	case "expert":
		maxOffset = 0.8
	}
	return (rand.Float64() - 0.5) * maxOffset * 2
}

func (s *ThreeDGeneratorService) getRandomScale(difficulty string) float64 {
	minScale, maxScale := 0.7, 0.9
	switch difficulty {
	case "expert":
		minScale, maxScale = 0.6, 1.0
	case "hard":
		minScale, maxScale = 0.65, 0.95
	}
	return minScale + rand.Float64()*(maxScale-minScale)
}

func (s *ThreeDGeneratorService) getRandomOpacity() float64 {
	return 0.85 + rand.Float64()*0.15
}

func (s *ThreeDGeneratorService) getEmissiveColor() string {
	if rand.Float64() > 0.7 {
		return pieceColors[rand.Intn(len(pieceColors))]
	}
	return "#000000"
}

func (s *ThreeDGeneratorService) getAnimationSpeed(difficulty string) float64 {
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

func (s *ThreeDGeneratorService) GetSession(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
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

func (s *ThreeDGeneratorService) DeleteSession(ctx context.Context, sessionID string) error {
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
