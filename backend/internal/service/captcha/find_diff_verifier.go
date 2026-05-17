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

type VerifyFindDiffRequest struct {
	SessionID   string           `json:"session_id" binding:"required"`
	Differences []FindDifference `json:"differences" binding:"required"`
	RiskScore   float64          `json:"risk_score"`
}

type VerifyFindDiffResult struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Score   float64 `json:"score"`
}

type FindDiffVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

func NewFindDiffVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *FindDiffVerifierService {
	return &FindDiffVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (v *FindDiffVerifierService) Verify(ctx context.Context, req *VerifyFindDiffRequest) (*VerifyFindDiffResult, error) {
	session, err := v.getSession(req.SessionID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifyFindDiffResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifyFindDiffResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	v.incrementVerifyCount(req.SessionID)

	if session.Status == "verified" {
		return &VerifyFindDiffResult{
			Success: true,
			Message: "验证码已验证通过",
			Score:   100,
		}, nil
	}

	originalImage := &FindDiffImage{}
	if err := json.Unmarshal([]byte(session.BackgroundURL), originalImage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal image: %w", err)
	}

	isValid, score := v.validateDifferences(originalImage, req.Differences)

	if isValid {
		v.markAsVerified(req.SessionID, req.RiskScore, 0, 0)
		return &VerifyFindDiffResult{
			Success: true,
			Message: "验证成功",
			Score:   score,
		}, nil
	}

	return &VerifyFindDiffResult{
		Success: false,
		Message: "验证失败",
		Score:   score,
	}, nil
}

func (v *FindDiffVerifierService) getSession(sessionID string) (*models.CaptchaSession, error) {
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

func (v *FindDiffVerifierService) incrementVerifyCount(sessionID string) {
	if v.sessionCache != nil {
		_ = v.sessionCache.IncrementVerifyCount(context.Background(), sessionID)
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateVerifyCount(sessionID)
	}
}

func (v *FindDiffVerifierService) markAsVerified(sessionID string, riskScore, traceScore, envScore float64) {
	if v.sessionCache != nil {
		_ = v.sessionCache.UpdateStatus(context.Background(), sessionID, "verified")
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateStatus(sessionID, "verified")
		_ = v.captchaRepo.UpdateRiskScore(sessionID, riskScore, traceScore, envScore)
	}
}

func (v *FindDiffVerifierService) validateDifferences(originalImage *FindDiffImage, userDiffs []FindDifference) (bool, float64) {
	if originalImage == nil {
		return false, 0
	}

	totalDiffs := len(originalImage.Differences)
	if totalDiffs == 0 {
		return false, 0
	}

	matchedDiffs := 0
	matched := make(map[int]bool)

	for _, userDiff := range userDiffs {
		for i, originalDiff := range originalImage.Differences {
			if matched[i] {
				continue
			}

			distance := v.calculateDistance(userDiff.X, userDiff.Y, originalDiff.X, originalDiff.Y)
			tolerance := originalDiff.Radius + 20

			if distance <= tolerance {
				matched[i] = true
				matchedDiffs++
				break
			}
		}
	}

	score := float64(matchedDiffs) / float64(totalDiffs) * 100
	return matchedDiffs == totalDiffs, score
}

func (v *FindDiffVerifierService) calculateDistance(x1, y1, x2, y2 int) int {
	dx := x1 - x2
	dy := y1 - y2
	return int(v.sqrt(float64(dx*dx + dy*dy)))
}

func (v *FindDiffVerifierService) sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}

	z := float64(1.0)
	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

func (v *FindDiffVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	return v.getSession(sessionID)
}

func (v *FindDiffVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
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
