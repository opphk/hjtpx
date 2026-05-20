package captcha

import (
	"context"
	"fmt"
	"math"
	"time"

	github.com/hjtpx/hjtpx/internal/repository/cache"
	github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type VerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type VerifyRequest struct {
	SessionID  string  `json:"session_id" binding:"required"`
	PositionX  int     `json:"position_x" binding:"required"`
	PositionY  int     `json:"position_y" binding:"required"`
	RiskScore  float64 `json:"risk_score"`
	TraceScore float64 `json:"trace_score"`
	EnvScore   float64 `json:"env_score"`
}

type VerifyResult struct {
	Success      bool    `json:"success"`
	Message      string  `json:"message"`
	Score        float64 `json:"score"`
	PositionDiff int     `json:"position_diff"`
}

func NewVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *VerifierService {
	return &VerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (v *VerifierService) Verify(ctx context.Context, req *VerifyRequest) (*VerifyResult, error) {
	session, err := v.getSession(req.SessionID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifyResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifyResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	v.incrementVerifyCount(req.SessionID)

	if session.Status == "verified" {
		return &VerifyResult{
			Success: true,
			Message: "验证码已验证通过",
			Score:   100,
		}, nil
	}

	diffX := abs(session.GapX - req.PositionX)
	diffY := abs(session.GapY - req.PositionY)

	tolerance := 5
	if diffX <= tolerance && diffY <= tolerance {
		v.markAsVerified(req.SessionID, req.RiskScore, req.TraceScore, req.EnvScore)

		return &VerifyResult{
			Success:      true,
			Message:      "验证成功",
			Score:        100,
			PositionDiff: diffX,
		}, nil
	}

	score := calculatePartialScore(diffX, diffY)

	return &VerifyResult{
		Success:      false,
		Message:      "验证失败",
		Score:        score,
		PositionDiff: diffX,
	}, nil
}

func (v *VerifierService) getSession(sessionID string) (*models.CaptchaSession, error) {
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

func (v *VerifierService) incrementVerifyCount(sessionID string) {
	if v.sessionCache != nil {
		_ = v.sessionCache.IncrementVerifyCount(context.Background(), sessionID)
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateVerifyCount(sessionID)
	}
}

func (v *VerifierService) markAsVerified(sessionID string, riskScore, traceScore, envScore float64) {
	if v.sessionCache != nil {
		_ = v.sessionCache.UpdateStatus(context.Background(), sessionID, "verified")
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateStatus(sessionID, "verified")
		_ = v.captchaRepo.UpdateRiskScore(sessionID, riskScore, traceScore, envScore)
	}
}

func (v *VerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	return v.getSession(sessionID)
}

func (v *VerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
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

func calculatePartialScore(diffX, diffY int) float64 {
	distance := math.Sqrt(float64(diffX*diffX + diffY*diffY))

	maxDistance := 100.0
	if distance >= maxDistance {
		return 0
	}

	score := 100 * (1 - distance/maxDistance)

	if diffX > 20 || diffY > 20 {
		score *= 0.7
	}

	if diffX > 40 || diffY > 40 {
		score *= 0.5
	}

	if score < 0 {
		score = 0
	}

	return math.Round(score*100) / 100
}
