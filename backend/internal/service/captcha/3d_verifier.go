package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type VerifyThreeDRequest struct {
	SessionID string        `json:"sessionID" binding:"required"`
	Puzzle    *ThreeDPuzzle `json:"puzzle" binding:"required"`
	RiskScore float64       `json:"riskScore"`
}

type VerifyThreeDResult struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Score   float64 `json:"score"`
}

type ThreeDVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

func NewThreeDVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *ThreeDVerifierService {
	return &ThreeDVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (v *ThreeDVerifierService) Verify(ctx context.Context, req *VerifyThreeDRequest) (*VerifyThreeDResult, error) {
	session, err := v.getSession(req.SessionID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifyThreeDResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifyThreeDResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	v.incrementVerifyCount(req.SessionID)

	if session.Status == "verified" {
		return &VerifyThreeDResult{
			Success: true,
			Message: "验证码已验证通过",
			Score:   100,
		}, nil
	}

	originalPuzzle := &ThreeDPuzzle{}
	if err := json.Unmarshal([]byte(session.BackgroundURL), originalPuzzle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal puzzle: %w", err)
	}

	isValid, score := v.validatePuzzle(req.Puzzle, originalPuzzle)

	if isValid {
		v.markAsVerified(req.SessionID, req.RiskScore, 0, 0)
		return &VerifyThreeDResult{
			Success: true,
			Message: "验证成功",
			Score:   score,
		}, nil
	}

	return &VerifyThreeDResult{
		Success: false,
		Message: "验证失败",
		Score:   score,
	}, nil
}

func (v *ThreeDVerifierService) validatePuzzle(userPuzzle, originalPuzzle *ThreeDPuzzle) (bool, float64) {
	if userPuzzle == nil || originalPuzzle == nil {
		return false, 0
	}

	if userPuzzle.GridSize != originalPuzzle.GridSize {
		return false, 0
	}

	if len(userPuzzle.Pieces) != len(originalPuzzle.Pieces) {
		return false, 0
	}

	totalScore := 0.0
	validPieces := 0

	for i := range originalPuzzle.Pieces {
		userPiece := findPieceByID(userPuzzle.Pieces, originalPuzzle.Pieces[i].ID)
		if userPiece == nil {
			continue
		}

		validPieces++
		pieceScore := v.calculatePieceScore(userPiece, originalPuzzle.Pieces[i], originalPuzzle.Difficulty)
		totalScore += pieceScore
	}

	if validPieces == 0 {
		return false, 0
	}

	avgScore := totalScore / float64(validPieces)
	passThreshold := v.getPassThreshold(originalPuzzle.Difficulty)

	return avgScore >= passThreshold, avgScore
}

func (v *ThreeDVerifierService) calculatePieceScore(userPiece, originalPiece *ThreeDPiece, difficulty string) float64 {
	rotXDiff := v.normalizeAngleDiff(userPiece.RotationX, originalPiece.RotationX)
	rotYDiff := v.normalizeAngleDiff(userPiece.RotationY, originalPiece.RotationY)
	rotZDiff := v.normalizeAngleDiff(userPiece.RotationZ, originalPiece.RotationZ)

	maxDiff := v.getMaxAllowedDiff(difficulty)

	rotXScore := math.Max(0, 1-rotXDiff/maxDiff)
	rotYScore := math.Max(0, 1-rotYDiff/maxDiff)
	rotZScore := math.Max(0, 1-rotZDiff/maxDiff)

	posXDiff := math.Abs(userPiece.PositionX - originalPiece.PositionX)
	posYDiff := math.Abs(userPiece.PositionY - originalPiece.PositionY)
	posZDiff := math.Abs(userPiece.PositionZ - originalPiece.PositionZ)

	posXScore := math.Max(0, 1-posXDiff/2)
	posYScore := math.Max(0, 1-posYDiff/2)
	posZScore := math.Max(0, 1-posZDiff/2)

	totalScore := (rotXScore + rotYScore + rotZScore + posXScore + posYScore + posZScore) / 6
	return totalScore * 100
}

func (v *ThreeDVerifierService) normalizeAngleDiff(angle1, angle2 float64) float64 {
	diff := math.Abs(angle1 - angle2)
	for diff > 180 {
		diff = 360 - diff
	}
	return diff
}

func (v *ThreeDVerifierService) getMaxAllowedDiff(difficulty string) float64 {
	switch difficulty {
	case "easy":
		return 45
	case "medium":
		return 30
	case "hard":
		return 20
	case "expert":
		return 15
	default:
		return 30
	}
}

func (v *ThreeDVerifierService) getPassThreshold(difficulty string) float64 {
	switch difficulty {
	case "easy":
		return 70
	case "medium":
		return 80
	case "hard":
		return 85
	case "expert":
		return 90
	default:
		return 80
	}
}

func findPieceByID(pieces []ThreeDPiece, id int) *ThreeDPiece {
	for i := range pieces {
		if pieces[i].ID == id {
			return &pieces[i]
		}
	}
	return nil
}

func (v *ThreeDVerifierService) getSession(sessionID string) (*models.CaptchaSession, error) {
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

func (v *ThreeDVerifierService) incrementVerifyCount(sessionID string) {
	if v.sessionCache != nil {
		_ = v.sessionCache.IncrementVerifyCount(context.Background(), sessionID)
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateVerifyCount(sessionID)
	}
}

func (v *ThreeDVerifierService) markAsVerified(sessionID string, riskScore, traceScore, envScore float64) {
	if v.sessionCache != nil {
		_ = v.sessionCache.UpdateStatus(context.Background(), sessionID, "verified")
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateStatus(sessionID, "verified")
		_ = v.captchaRepo.UpdateRiskScore(sessionID, riskScore, traceScore, envScore)
	}
}

func (v *ThreeDVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	return v.getSession(sessionID)
}

func (v *ThreeDVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
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
