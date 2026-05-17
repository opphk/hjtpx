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

type LianLianKanTile struct {
	Type  int `json:"type"`
	X     int `json:"x"`
	Y     int `json:"y"`
	Index int `json:"index"`
}

type LianLianKanBoard struct {
	Tiles       [][]LianLianKanTile `json:"tiles"`
	Width       int                 `json:"width"`
	Height      int                 `json:"height"`
	PairCount   int                 `json:"pair_count"`
	Shuffled    bool                `json:"shuffled"`
}

type CreateLianLianKanRequest struct {
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	TileTypes   int    `json:"tile_types"`
	ClientIP    string `json:"client_ip"`
	UserAgent   string `json:"user_agent"`
	Fingerprint string `json:"fingerprint"`
}

type CreateLianLianKanResponse struct {
	SessionID  string            `json:"session_id"`
	Board      *LianLianKanBoard `json:"board"`
	ExpiresIn  int64             `json:"expires_in"`
	ExpiresAt  int64             `json:"expires_at"`
	TileIcons  []string          `json:"tile_icons"`
}

type LianLianKanGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

var tileIcons = []string{
	"🍎", "🍊", "🍋", "🍇", "🍓", "🍒", "🍑", "🥝",
	"🍌", "🍉", "🥭", "🍐", "🍍", "🥥", "🍈", "🍏",
	"🍆", "🥑", "🥦", "🌽", "🥕", "🍠", "🥔", "🌶️",
}

func NewLianLianKanGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *LianLianKanGeneratorService {
	return &LianLianKanGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (s *LianLianKanGeneratorService) Create(ctx context.Context, req *CreateLianLianKanRequest) (*CreateLianLianKanResponse, error) {
	width := req.Width
	height := req.Height
	tileTypes := req.TileTypes

	if width <= 0 {
		width = 6
	}
	if height <= 0 {
		height = 6
	}
	if tileTypes <= 0 {
		tileTypes = 8
	}

	if tileTypes > len(tileIcons) {
		tileTypes = len(tileIcons)
	}

	board, err := generateLianLianKanBoard(width, height, tileTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate board: %w", err)
	}

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	boardData, err := json.Marshal(board)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal board: %w", err)
	}

	session := &models.CaptchaSession{
		SessionID:     sessionID,
		BackgroundURL: string(boardData),
		SliderURL:     "",
		GapX:          0,
		GapY:          0,
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

	selectedIcons := tileIcons[:tileTypes]

	return &CreateLianLianKanResponse{
		SessionID: sessionID,
		Board:     board,
		ExpiresIn: int64(5 * time.Minute / time.Second),
		ExpiresAt: expiresAt.Unix(),
		TileIcons: selectedIcons,
	}, nil
}

func generateLianLianKanBoard(width, height, tileTypes int) (*LianLianKanBoard, error) {
	rand.Seed(time.Now().UnixNano())

	totalTiles := width * height
	if totalTiles%2 != 0 {
		return nil, fmt.Errorf("total tiles must be even")
	}

	pairCount := totalTiles / 2

	tiles := make([]LianLianKanTile, totalTiles)
	for i := 0; i < pairCount; i++ {
		tileType := i % tileTypes
		tiles[i*2] = LianLianKanTile{Type: tileType, Index: i * 2}
		tiles[i*2+1] = LianLianKanTile{Type: tileType, Index: i*2 + 1}
	}

	for i := range tiles {
		j := rand.Intn(i + 1)
		tiles[i], tiles[j] = tiles[j], tiles[i]
	}

	board := make([][]LianLianKanTile, height)
	for y := 0; y < height; y++ {
		board[y] = make([]LianLianKanTile, width)
		for x := 0; x < width; x++ {
			idx := y*width + x
			board[y][x] = tiles[idx]
			board[y][x].X = x
			board[y][x].Y = y
		}
	}

	return &LianLianKanBoard{
		Tiles:     board,
		Width:     width,
		Height:    height,
		PairCount: pairCount,
		Shuffled:  true,
	}, nil
}

func (s *LianLianKanGeneratorService) GetSession(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
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

func (s *LianLianKanGeneratorService) DeleteSession(ctx context.Context, sessionID string) error {
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
