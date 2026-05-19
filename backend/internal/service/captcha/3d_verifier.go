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
		pieceScore := v.calculatePieceScore(userPiece, &originalPuzzle.Pieces[i], originalPuzzle.Difficulty)
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
	maxDiff := v.getMaxAllowedDiff(difficulty)
	
	rotXDiff := v.normalizeAngleDiff(userPiece.RotationX, originalPiece.RotationX)
	rotYDiff := v.normalizeAngleDiff(userPiece.RotationY, originalPiece.RotationY)
	rotZDiff := v.normalizeAngleDiff(userPiece.RotationZ, originalPiece.RotationZ)

	rotXScore := v.calculateRotationScore(rotXDiff, maxDiff)
	rotYScore := v.calculateRotationScore(rotYDiff, maxDiff)
	rotZScore := v.calculateRotationScore(rotZDiff, maxDiff)

	posXDiff := math.Abs(userPiece.PositionX - originalPiece.PositionX)
	posYDiff := math.Abs(userPiece.PositionY - originalPiece.PositionY)
	posZDiff := math.Abs(userPiece.PositionZ - originalPiece.PositionZ)

	posXScore := v.calculatePositionScore(posXDiff)
	posYScore := v.calculatePositionScore(posYDiff)
	posZScore := v.calculatePositionScore(posZDiff)

	scaleDiff := math.Abs(userPiece.Scale - originalPiece.Scale)
	scaleScore := v.calculateScaleScore(scaleDiff)

	weights := v.getScoreWeights(difficulty)
	totalScore := weights.Rotation*(rotXScore+rotYScore+rotZScore)/3 +
		weights.Position*(posXScore+posYScore+posZScore)/3 +
		weights.Scale*scaleScore

	return totalScore * 100
}

func (v *ThreeDVerifierService) calculateRotationScore(diff, maxDiff float64) float64 {
	if diff <= 5 {
		return 1.0
	}
	if diff <= 15 {
		return 0.9 + (15-diff)/15*0.1
	}
	if diff >= maxDiff {
		return 0.0
	}
	return math.Max(0, 1-diff/maxDiff)
}

func (v *ThreeDVerifierService) calculatePositionScore(diff float64) float64 {
	maxPosDiff := 0.5
	if diff <= 0.1 {
		return 1.0
	}
	if diff >= maxPosDiff {
		return 0.0
	}
	return math.Max(0, 1-diff/maxPosDiff)
}

func (v *ThreeDVerifierService) calculateScaleScore(diff float64) float64 {
	maxScaleDiff := 0.3
	if diff <= 0.05 {
		return 1.0
	}
	if diff >= maxScaleDiff {
		return 0.0
	}
	return math.Max(0, 1-diff/maxScaleDiff)
}

type ScoreWeights struct {
	Rotation float64
	Position float64
	Scale    float64
}

func (v *ThreeDVerifierService) getScoreWeights(difficulty string) ScoreWeights {
	switch difficulty {
	case "easy":
		return ScoreWeights{Rotation: 0.7, Position: 0.2, Scale: 0.1}
	case "medium":
		return ScoreWeights{Rotation: 0.75, Position: 0.15, Scale: 0.1}
	case "hard":
		return ScoreWeights{Rotation: 0.8, Position: 0.12, Scale: 0.08}
	case "expert":
		return ScoreWeights{Rotation: 0.85, Position: 0.1, Scale: 0.05}
	default:
		return ScoreWeights{Rotation: 0.75, Position: 0.15, Scale: 0.1}
	}
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
