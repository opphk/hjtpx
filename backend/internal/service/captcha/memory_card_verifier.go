package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type MemoryCardsMatch struct {
	Card1 *MemoryCard `json:"card1"`
	Card2 *MemoryCard `json:"card2"`
}

type VerifyMemoryCardsRequest struct {
	SessionID string             `json:"session_id" binding:"required"`
	Board     *MemoryCardsBoard  `json:"board" binding:"required"`
	Matches   []MemoryCardsMatch `json:"matches" binding:"required"`
	TimeUsed  int                `json:"time_used"`
	RiskScore float64            `json:"risk_score"`
}

type VerifyMemoryCardsResult struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Score   float64 `json:"score"`
}

type MemoryCardsVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

func NewMemoryCardsVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *MemoryCardsVerifierService {
	return &MemoryCardsVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (v *MemoryCardsVerifierService) Verify(ctx context.Context, req *VerifyMemoryCardsRequest) (*VerifyMemoryCardsResult, error) {
	session, err := v.getSession(req.SessionID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifyMemoryCardsResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifyMemoryCardsResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	v.incrementVerifyCount(req.SessionID)

	if session.Status == "verified" {
		return &VerifyMemoryCardsResult{
			Success: true,
			Message: "验证码已验证通过",
			Score:   100,
		}, nil
	}

	originalBoard := &MemoryCardsBoard{}
	if err := json.Unmarshal([]byte(session.BackgroundURL), originalBoard); err != nil {
		return nil, fmt.Errorf("failed to unmarshal board: %w", err)
	}

	isValid, score := v.validateBoard(req.Board, originalBoard, req.Matches, req.TimeUsed)

	if isValid {
		v.markAsVerified(req.SessionID, req.RiskScore, 0, 0)
		return &VerifyMemoryCardsResult{
			Success: true,
			Message: "验证成功",
			Score:   score,
		}, nil
	}

	return &VerifyMemoryCardsResult{
		Success: false,
		Message: "验证失败",
		Score:   score,
	}, nil
}

func (v *MemoryCardsVerifierService) getSession(sessionID string) (*models.CaptchaSession, error) {
	if v.sessionCache != nil {
		session, err := v.sessionCache.Get(context.Background(), sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	if v.captchaRepo != nil {
		session, err := v.captchaRepo.GetBySessionID(sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (v *MemoryCardsVerifierService) incrementVerifyCount(sessionID string) {
	if v.sessionCache != nil {
		_ = v.sessionCache.IncrementVerifyCount(context.Background(), sessionID)
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateVerifyCount(sessionID)
	}
}

func (v *MemoryCardsVerifierService) markAsVerified(sessionID string, riskScore, traceScore, envScore float64) {
	if v.sessionCache != nil {
		_ = v.sessionCache.UpdateStatus(context.Background(), sessionID, "verified")
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateStatus(sessionID, "verified")
		_ = v.captchaRepo.UpdateRiskScore(sessionID, riskScore, traceScore, envScore)
	}
}

func (v *MemoryCardsVerifierService) validateBoard(userBoard, originalBoard *MemoryCardsBoard, matches []MemoryCardsMatch, timeUsed int) (bool, float64) {
	if userBoard == nil || originalBoard == nil {
		return false, 0
	}

	if userBoard.Width != originalBoard.Width || userBoard.Height != originalBoard.Height {
		return false, 0
	}

	totalPairs := originalBoard.PairCount
	matchedPairs := 0

	visited := make(map[int]bool)

	for _, match := range matches {
		if match.Card1 == nil || match.Card2 == nil {
			continue
		}

		card1 := match.Card1
		card2 := match.Card2

		if visited[card1.Index] || visited[card2.Index] {
			continue
		}

		if card1.Type == card2.Type && card1.Index != card2.Index {
			visited[card1.Index] = true
			visited[card2.Index] = true
			matchedPairs++
		}
	}

	baseScore := float64(matchedPairs) / float64(totalPairs) * 100

	// Time bonus: faster completion gets higher score
	var timeBonus float64
	if matchedPairs == totalPairs {
		if timeUsed <= 0 {
			timeBonus = 0
		} else if timeUsed <= 30 {
			timeBonus = 10
		} else if timeUsed <= 60 {
			timeBonus = 5
		} else {
			timeBonus = 0
		}
	}

	totalScore := baseScore + timeBonus
	if totalScore > 100 {
		totalScore = 100
	}

	return matchedPairs == totalPairs, totalScore
}

func (v *MemoryCardsVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	return v.getSession(sessionID)
}

func (v *MemoryCardsVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
	session, err := v.getSession(sessionID)
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
