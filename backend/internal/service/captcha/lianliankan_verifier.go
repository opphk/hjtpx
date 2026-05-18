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

type LianLianKanPair struct {
	Tile1 *LianLianKanTile `json:"tile1"`
	Tile2 *LianLianKanTile `json:"tile2"`
}

type LianLianKanPath struct {
	Points []struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"points"`
}

type VerifyLianLianKanRequest struct {
	SessionID string            `json:"session_id" binding:"required"`
	Board     *LianLianKanBoard `json:"board" binding:"required"`
	Pairs     []LianLianKanPair `json:"pairs" binding:"required"`
	RiskScore float64           `json:"risk_score"`
}

type VerifyLianLianKanResult struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Score   float64 `json:"score"`
}

type LianLianKanVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

func NewLianLianKanVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *LianLianKanVerifierService {
	return &LianLianKanVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (v *LianLianKanVerifierService) Verify(ctx context.Context, req *VerifyLianLianKanRequest) (*VerifyLianLianKanResult, error) {
	session, err := v.getSession(req.SessionID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifyLianLianKanResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifyLianLianKanResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	v.incrementVerifyCount(req.SessionID)

	if session.Status == "verified" {
		return &VerifyLianLianKanResult{
			Success: true,
			Message: "验证码已验证通过",
			Score:   100,
		}, nil
	}

	originalBoard := &LianLianKanBoard{}
	if err := json.Unmarshal([]byte(session.BackgroundURL), originalBoard); err != nil {
		return nil, fmt.Errorf("failed to unmarshal board: %w", err)
	}

	isValid, score := v.validateBoard(req.Board, originalBoard, req.Pairs)

	if isValid {
		v.markAsVerified(req.SessionID, req.RiskScore, 0, 0)
		return &VerifyLianLianKanResult{
			Success: true,
			Message: "验证成功",
			Score:   score,
		}, nil
	}

	return &VerifyLianLianKanResult{
		Success: false,
		Message: "验证失败",
		Score:   score,
	}, nil
}

func (v *LianLianKanVerifierService) getSession(sessionID string) (*models.CaptchaSession, error) {
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

func (v *LianLianKanVerifierService) incrementVerifyCount(sessionID string) {
	if v.sessionCache != nil {
		_ = v.sessionCache.IncrementVerifyCount(context.Background(), sessionID)
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateVerifyCount(sessionID)
	}
}

func (v *LianLianKanVerifierService) markAsVerified(sessionID string, riskScore, traceScore, envScore float64) {
	if v.sessionCache != nil {
		_ = v.sessionCache.UpdateStatus(context.Background(), sessionID, "verified")
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateStatus(sessionID, "verified")
		_ = v.captchaRepo.UpdateRiskScore(sessionID, riskScore, traceScore, envScore)
	}
}

func (v *LianLianKanVerifierService) validateBoard(userBoard, originalBoard *LianLianKanBoard, pairs []LianLianKanPair) (bool, float64) {
	if userBoard == nil || originalBoard == nil {
		return false, 0
	}

	if userBoard.Width != originalBoard.Width || userBoard.Height != originalBoard.Height {
		return false, 0
	}

	totalPairs := originalBoard.PairCount
	matchedPairs := 0

	visited := make(map[int]bool)

	for _, pair := range pairs {
		if pair.Tile1 == nil || pair.Tile2 == nil {
			continue
		}

		tile1 := pair.Tile1
		tile2 := pair.Tile2

		if visited[tile1.Index] || visited[tile2.Index] {
			continue
		}

		if tile1.Type == tile2.Type && tile1.Index != tile2.Index {
			visited[tile1.Index] = true
			visited[tile2.Index] = true
			matchedPairs++
		}
	}

	score := float64(matchedPairs) / float64(totalPairs) * 100

	return matchedPairs == totalPairs, score
}

func (v *LianLianKanVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	return v.getSession(sessionID)
}

func (v *LianLianKanVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
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
