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

type ARVerifyRequest struct {
	SessionID      string        `json:"sessionID" binding:"required"`
	Scene          *ARScene      `json:"scene" binding:"required"`
	UserGesture    string        `json:"userGesture"`
	PlacedObjectID int           `json:"placedObjectID"`
	FinalPosition  ARPosition    `json:"finalPosition"`
	RiskScore      float64       `json:"riskScore"`
}

type ARVerifyResult struct {
	Success       bool    `json:"success"`
	Message       string  `json:"message"`
	Score         float64 `json:"score"`
	TargetMatched bool    `json:"targetMatched"`
	GestureValid  bool    `json:"gestureValid"`
}

type ARVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

func NewARVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *ARVerifierService {
	return &ARVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (v *ARVerifierService) Verify(ctx context.Context, req *ARVerifyRequest) (*ARVerifyResult, error) {
	session, err := v.getSession(req.SessionID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiredAt) {
		return &ARVerifyResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &ARVerifyResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	v.incrementVerifyCount(req.SessionID)

	if session.Status == "verified" {
		return &ARVerifyResult{
			Success: true,
			Message: "验证码已验证通过",
			Score:   100,
		}, nil
	}

	originalScene := &ARScene{}
	if err := json.Unmarshal([]byte(session.BackgroundURL), originalScene); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scene: %w", err)
	}

	isValid, score, targetMatched, gestureValid := v.validateScene(req, originalScene)

	if isValid {
		v.markAsVerified(req.SessionID, req.RiskScore, 0, 0)
		return &ARVerifyResult{
			Success:       true,
			Message:       "验证成功",
			Score:         score,
			TargetMatched: targetMatched,
			GestureValid:  gestureValid,
		}, nil
	}

	return &ARVerifyResult{
		Success:       false,
		Message:       v.getFailureMessage(targetMatched, gestureValid),
		Score:         score,
		TargetMatched: targetMatched,
		GestureValid:  gestureValid,
	}, nil
}

func (v *ARVerifierService) validateScene(req *ARVerifyRequest, originalScene *ARScene) (bool, float64, bool, bool) {
	if req.Scene == nil || originalScene == nil {
		return false, 0, false, false
	}

	targetMatched := v.validateTargetObject(req, originalScene)
	gestureValid := v.validateGesture(req, originalScene)
	positionScore := v.validatePosition(req, originalScene)

	weights := v.getValidationWeights(originalScene.Difficulty)
	totalScore := weights.Target*float64(btoi(targetMatched))*100 +
		weights.Gesture*float64(btoi(gestureValid))*100 +
		weights.Position*positionScore

	passThreshold := v.getPassThreshold(originalScene.Difficulty)

	return totalScore >= passThreshold, totalScore, targetMatched, gestureValid
}

func (v *ARVerifierService) validateTargetObject(req *ARVerifyRequest, originalScene *ARScene) bool {
	if req.PlacedObjectID < 0 || req.PlacedObjectID >= len(originalScene.Objects) {
		return false
	}

	return originalScene.Objects[req.PlacedObjectID].IsTarget
}

func (v *ARVerifierService) validateGesture(req *ARVerifyRequest, originalScene *ARScene) bool {
	if originalScene.RequiredGesture == "" {
		return true
	}

	return req.UserGesture == originalScene.RequiredGesture
}

func (v *ARVerifierService) validatePosition(req *ARVerifyRequest, originalScene *ARScene) float64 {
	targetPos := originalScene.TargetPosition
	userPos := req.FinalPosition

	xDiff := math.Abs(userPos.X - targetPos.X)
	yDiff := math.Abs(userPos.Y - targetPos.Y)
	zDiff := math.Abs(userPos.Z - targetPos.Z)

	maxDiff := v.getMaxPositionDiff(originalScene.Difficulty)

	totalDiff := (xDiff + yDiff + zDiff) / 3

	if totalDiff <= 0.2 {
		return 100
	}
	if totalDiff >= maxDiff {
		return 0
	}

	return math.Max(0, 100*(1-totalDiff/maxDiff))
}

func (v *ARVerifierService) getMaxPositionDiff(difficulty string) float64 {
	switch difficulty {
	case "easy":
		return 1.0
	case "medium":
		return 0.7
	case "hard":
		return 0.5
	case "expert":
		return 0.3
	default:
		return 0.7
	}
}

func (v *ARVerifierService) getValidationWeights(difficulty string) struct {
	Target   float64
	Gesture  float64
	Position float64
} {
	switch difficulty {
	case "easy":
		return struct {
			Target   float64
			Gesture  float64
			Position float64
		}{0.5, 0.2, 0.3}
	case "medium":
		return struct {
			Target   float64
			Gesture  float64
			Position float64
		}{0.4, 0.3, 0.3}
	case "hard":
		return struct {
			Target   float64
			Gesture  float64
			Position float64
		}{0.35, 0.35, 0.3}
	case "expert":
		return struct {
			Target   float64
			Gesture  float64
			Position float64
		}{0.3, 0.4, 0.3}
	default:
		return struct {
			Target   float64
			Gesture  float64
			Position float64
		}{0.4, 0.3, 0.3}
	}
}

func (v *ARVerifierService) getPassThreshold(difficulty string) float64 {
	switch difficulty {
	case "easy":
		return 60
	case "medium":
		return 70
	case "hard":
		return 80
	case "expert":
		return 90
	default:
		return 70
	}
}

func (v *ARVerifierService) getFailureMessage(targetMatched, gestureValid bool) string {
	if !targetMatched && !gestureValid {
		return "请选择正确的目标物体并执行正确的手势"
	}
	if !targetMatched {
		return "请选择正确的目标物体"
	}
	return "请执行正确的手势"
}

func (v *ARVerifierService) getSession(sessionID string) (*models.CaptchaSession, error) {
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

func (v *ARVerifierService) incrementVerifyCount(sessionID string) {
	if v.sessionCache != nil {
		_ = v.sessionCache.IncrementVerifyCount(context.Background(), sessionID)
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateVerifyCount(sessionID)
	}
}

func (v *ARVerifierService) markAsVerified(sessionID string, riskScore, traceScore, envScore float64) {
	if v.sessionCache != nil {
		_ = v.sessionCache.UpdateStatus(context.Background(), sessionID, "verified")
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateStatus(sessionID, "verified")
		_ = v.captchaRepo.UpdateRiskScore(sessionID, riskScore, traceScore, envScore)
	}
}

func (v *ARVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	return v.getSession(sessionID)
}

func (v *ARVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
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

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
