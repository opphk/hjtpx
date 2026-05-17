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

type ThreeDPuzzle struct {
	Pieces      []ThreeDPiece `json:"pieces"`
	GridSize    int           `json:"gridSize"`
	Difficulty  string        `json:"difficulty"`
	TargetRotX float64       `json:"targetRotX"`
	TargetRotY float64       `json:"targetRotY"`
	TargetRotZ float64       `json:"targetRotZ"`
}

type ThreeDPiece struct {
	ID        int     `json:"id"`
	Type      string  `json:"type"`
	Color     string  `json:"color"`
	PositionX float64 `json:"positionX"`
	PositionY float64 `json:"positionY"`
	PositionZ float64 `json:"positionZ"`
	RotationX float64 `json:"rotationX"`
	RotationY float64 `json:"rotationY"`
	RotationZ float64 `json:"rotationZ"`
	Scale     float64 `json:"scale"`
}

type CreateThreeDRequest struct {
	Difficulty string `json:"difficulty"`
	ClientIP   string `json:"clientIP"`
	UserAgent  string `json:"userAgent"`
	Fingerprint string `json:"fingerprint"`
}

type CreateThreeDResponse struct {
	SessionID  string        `json:"sessionID"`
	Puzzle     *ThreeDPuzzle `json:"puzzle"`
	ExpiresIn  int64         `json:"expiresIn"`
	ExpiresAt  int64         `json:"expiresAt"`
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

var pieceColors = []string{
	"#e74c3c", "#3498db", "#2ecc71", "#f39c12", "#9b59b6",
	"#1abc9c", "#e91e63", "#00bcd4", "#8bc34a", "#ff9800",
}

var pieceTypes = []string{
	"cube", "cylinder", "sphere", "cone", "torus",
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
		piece := ThreeDPiece{
			ID:        i,
			Type:      pieceTypes[rand.Intn(len(pieceTypes))],
			Color:     pieceColors[rand.Intn(len(pieceColors))],
			PositionX: float64(i%gridSize) - float64(gridSize-1)/2,
			PositionY: float64(i/gridSize) - float64(gridSize-1)/2,
			PositionZ: 0,
			Scale:     0.8,
		}

		switch difficulty {
		case "easy":
			piece.RotationX = rand.Float64() * 90
			piece.RotationY = rand.Float64() * 90
		case "medium":
			piece.RotationX = rand.Float64() * 180
			piece.RotationY = rand.Float64() * 180
			piece.RotationZ = rand.Float64() * 90
		case "hard":
			piece.RotationX = rand.Float64() * 360
			piece.RotationY = rand.Float64() * 360
			piece.RotationZ = rand.Float64() * 180
		case "expert":
			piece.RotationX = rand.Float64() * 360
			piece.RotationY = rand.Float64() * 360
			piece.RotationZ = rand.Float64() * 360
		}

		pieces = append(pieces, piece)
	}

	return &ThreeDPuzzle{
		Pieces:      pieces,
		GridSize:    gridSize,
		Difficulty:  difficulty,
		TargetRotX: targetRotX,
		TargetRotY: targetRotY,
		TargetRotZ: targetRotZ,
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
