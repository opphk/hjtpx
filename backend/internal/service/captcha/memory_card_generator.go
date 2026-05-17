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

type MemoryCard struct {
	Type  int `json:"type"`
	X     int `json:"x"`
	Y     int `json:"y"`
	Index int `json:"index"`
}

type MemoryCardsBoard struct {
	Cards     [][]MemoryCard `json:"cards"`
	Width     int            `json:"width"`
	Height    int            `json:"height"`
	PairCount int            `json:"pair_count"`
	Shuffled  bool           `json:"shuffled"`
}

type CreateMemoryCardsRequest struct {
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	CardTypes   int    `json:"card_types"`
	ShowTime    int    `json:"show_time"`
	ClientIP    string `json:"client_ip"`
	UserAgent   string `json:"user_agent"`
	Fingerprint string `json:"fingerprint"`
}

type CreateMemoryCardsResponse struct {
	SessionID string            `json:"session_id"`
	Board     *MemoryCardsBoard `json:"board"`
	ExpiresIn int64             `json:"expires_in"`
	ExpiresAt int64             `json:"expires_at"`
	CardIcons []string          `json:"card_icons"`
	ShowTime  int64             `json:"show_time"`
}

type MemoryCardsGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

var cardIcons = []string{
	"🍎", "🍊", "🍋", "🍇", "🍓", "🍒", "🍑", "🥝",
	"🍌", "🍉", "🥭", "🍐", "🍍", "🥥", "🍈", "🍏",
	"🍆", "🥑", "🥦", "🌽", "🥕", "🍠", "🥔", "🌶️",
}

func NewMemoryCardsGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *MemoryCardsGeneratorService {
	return &MemoryCardsGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (s *MemoryCardsGeneratorService) Create(ctx context.Context, req *CreateMemoryCardsRequest) (*CreateMemoryCardsResponse, error) {
	width := req.Width
	height := req.Height
	cardTypes := req.CardTypes
	showTime := req.ShowTime

	if width <= 0 {
		width = 4
	}
	if height <= 0 {
		height = 4
	}
	if cardTypes <= 0 {
		cardTypes = 8
	}
	if showTime <= 0 {
		showTime = 5
	}

	if cardTypes > len(cardIcons) {
		cardTypes = len(cardIcons)
	}

	board, err := generateMemoryCardsBoard(width, height, cardTypes)
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

	selectedIcons := cardIcons[:cardTypes]

	return &CreateMemoryCardsResponse{
		SessionID: sessionID,
		Board:     board,
		ExpiresIn: int64(5 * time.Minute / time.Second),
		ExpiresAt: expiresAt.Unix(),
		CardIcons: selectedIcons,
		ShowTime:  int64(showTime),
	}, nil
}

func generateMemoryCardsBoard(width, height, cardTypes int) (*MemoryCardsBoard, error) {
	rand.Seed(time.Now().UnixNano())

	totalCards := width * height
	if totalCards%2 != 0 {
		return nil, fmt.Errorf("total cards must be even")
	}

	pairCount := totalCards / 2

	cards := make([]MemoryCard, totalCards)
	for i := 0; i < pairCount; i++ {
		cardType := i % cardTypes
		cards[i*2] = MemoryCard{Type: cardType, Index: i * 2}
		cards[i*2+1] = MemoryCard{Type: cardType, Index: i*2 + 1}
	}

	for i := range cards {
		j := rand.Intn(i + 1)
		cards[i], cards[j] = cards[j], cards[i]
	}

	board := make([][]MemoryCard, height)
	for y := 0; y < height; y++ {
		board[y] = make([]MemoryCard, width)
		for x := 0; x < width; x++ {
			idx := y*width + x
			board[y][x] = cards[idx]
			board[y][x].X = x
			board[y][x].Y = y
		}
	}

	return &MemoryCardsBoard{
		Cards:     board,
		Width:     width,
		Height:    height,
		PairCount: pairCount,
		Shuffled:  true,
	}, nil
}

func (s *MemoryCardsGeneratorService) GetSession(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
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

func (s *MemoryCardsGeneratorService) DeleteSession(ctx context.Context, sessionID string) error {
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
