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

type VerifyARRequest struct {
	SessionID   string      `json:"sessionID" binding:"required"`
	UserGesture *UserGesture `json:"userGesture" binding:"required"`
	ObjectID    int         `json:"objectID"`
	PositionX   float64     `json:"positionX"`
	PositionY   float64     `json:"positionY"`
	PositionZ   float64     `json:"positionZ"`
	RiskScore   float64     `json:"riskScore"`
}

type UserGesture struct {
	Type        string        `json:"type"`
	Points      []GesturePoint `json:"points"`
	Duration    int64         `json:"duration"`
	GestureType string        `json:"gestureType"`
}

type VerifyARResult struct {
	Success       bool    `json:"success"`
	Message       string  `json:"message"`
	Score         float64 `json:"score"`
	GestureScore  float64 `json:"gestureScore"`
	PositionScore float64 `json:"positionScore"`
	ObjectScore   float64 `json:"objectScore"`
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

func (v *ARVerifierService) Verify(ctx context.Context, req *VerifyARRequest) (*VerifyARResult, error) {
	session, err := v.getSession(req.SessionID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifyARResult{
			Success:       false,
			Message:       "验证码已过期",
			Score:         0,
			GestureScore:  0,
			PositionScore: 0,
			ObjectScore:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifyARResult{
			Success:       false,
			Message:       "验证次数已用完",
			Score:         0,
			GestureScore:  0,
			PositionScore: 0,
			ObjectScore:   0,
		}, nil
	}

	v.incrementVerifyCount(req.SessionID)

	if session.Status == "verified" {
		return &VerifyARResult{
			Success:       true,
			Message:       "验证码已验证通过",
			Score:         100,
			GestureScore:  100,
			PositionScore: 100,
			ObjectScore:   100,
		}, nil
	}

	originalScene := &ARScene{}
	if err := json.Unmarshal([]byte(session.BackgroundURL), originalScene); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scene: %w", err)
	}

	gestureScore := v.validateGesture(req.UserGesture, originalScene)
	positionScore := v.validatePosition(req.PositionX, req.PositionY, req.PositionZ, originalScene)
	objectScore := v.validateObjectSelection(req.ObjectID, originalScene)

	totalScore := (gestureScore + positionScore + objectScore) / 3

	isValid := v.isScoreSufficient(totalScore, originalScene.Difficulty)

	if isValid {
		v.markAsVerified(req.SessionID, req.RiskScore, 0, 0)
		return &VerifyARResult{
			Success:       true,
			Message:       "验证成功",
			Score:         totalScore,
			GestureScore:  gestureScore,
			PositionScore: positionScore,
			ObjectScore:   objectScore,
		}, nil
	}

	return &VerifyARResult{
		Success:       false,
		Message:       "验证失败",
		Score:         totalScore,
		GestureScore:  gestureScore,
		PositionScore: positionScore,
		ObjectScore:   objectScore,
	}, nil
}

func (v *ARVerifierService) validateGesture(userGesture *UserGesture, scene *ARScene) float64 {
	if userGesture == nil || len(userGesture.Points) == 0 {
		return 0
	}

	expectedGesture := scene.GesturePath
	if len(expectedGesture) == 0 {
		return 50
	}

	pathScore := v.calculatePathSimilarity(userGesture.Points, expectedGesture)
	typeScore := v.calculateGestureTypeScore(userGesture.GestureType, scene.GestureType)
	durationScore := v.calculateDurationScore(userGesture.Duration, len(userGesture.Points))

	totalScore := pathScore*0.6 + typeScore*0.3 + durationScore*0.1

	return totalScore * 100
}

func (v *ARVerifierService) calculatePathSimilarity(userPoints, expectedPoints []GesturePoint) float64 {
	if len(userPoints) == 0 || len(expectedPoints) == 0 {
		return 0
	}

	userLen := len(userPoints)
	expLen := len(expectedPoints)

	minLen := userLen
	if expLen < minLen {
		minLen = expLen
	}

	resampledUser := v.resamplePoints(userPoints, minLen)
	resampledExp := v.resamplePoints(expectedPoints, minLen)

	var totalDist float64
	for i := 0; i < minLen; i++ {
		dist := math.Sqrt(
			math.Pow(resampledUser[i].X-resampledExp[i].X, 2) +
			math.Pow(resampledUser[i].Y-resampledExp[i].Y, 2),
		)
		totalDist += dist
	}

	avgDist := totalDist / float64(minLen)
	maxDist := 1.0

	similarity := 1.0 - math.Min(avgDist/maxDist, 1.0)
	return similarity
}

func (v *ARVerifierService) resamplePoints(points []GesturePoint, targetLen int) []GesturePoint {
	if len(points) == 0 {
		return points
	}

	if len(points) == targetLen {
		return points
	}

	resampled := make([]GesturePoint, targetLen)
	resampled[0] = points[0]

	var totalLen float64
	for i := 1; i < len(points); i++ {
		totalLen += math.Sqrt(
			math.Pow(points[i].X-points[i-1].X, 2) +
			math.Pow(points[i].Y-points[i-1].Y, 2),
		)
	}

	segmentLen := totalLen / float64(targetLen-1)
	accumLen := 0.0
	idx := 0

	for i := 1; i < targetLen && idx < len(points)-1; i++ {
		targetDist := float64(i) * segmentLen

		for accumLen+math.Sqrt(
			math.Pow(points[idx+1].X-points[idx].X, 2)+
				math.Pow(points[idx+1].Y-points[idx].Y, 2),
		) < targetDist {
			accumLen += math.Sqrt(
				math.Pow(points[idx+1].X-points[idx].X, 2)+
					math.Pow(points[idx+1].Y-points[idx].Y, 2),
			)
			idx++
			if idx >= len(points)-1 {
				break
			}
		}

		segLen := math.Sqrt(
			math.Pow(points[idx+1].X-points[idx].X, 2)+
				math.Pow(points[idx+1].Y-points[idx].Y, 2),
		)
		if segLen > 0 {
			t := (targetDist - accumLen) / segLen
			resampled[i] = GesturePoint{
				X:     points[idx].X + t*(points[idx+1].X-points[idx].X),
				Y:     points[idx].Y + t*(points[idx+1].Y-points[idx].Y),
				Pressure: points[idx].Pressure + t*(points[idx+1].Pressure-points[idx].Pressure),
			}
		} else {
			resampled[i] = points[idx]
		}
	}

	resampled[targetLen-1] = points[len(points)-1]

	return resampled
}

func (v *ARVerifierService) calculateGestureTypeScore(userType, expectedType string) float64 {
	if userType == expectedType {
		return 1.0
	}

	similarity := 0.0
	for _, gt := range gestureTypes {
		if userType == gt {
			similarity = 0.5
			break
		}
	}

	return similarity
}

func (v *ARVerifierService) calculateDurationScore(duration int64, pointCount int) float64 {
	if pointCount == 0 {
		return 0
	}

	expectedDurationPerPoint := 50
	expectedDuration := float64(pointCount * expectedDurationPerPoint)

	if expectedDuration == 0 {
		return 0.5
	}

	ratio := float64(duration) / expectedDuration

	if ratio < 0.5 {
		return 0.3
	} else if ratio > 2.0 {
		return 0.5
	}

	optimalRatio := 1.0
	diff := math.Abs(ratio - optimalRatio)
	score := 1.0 - diff

	return math.Max(0, math.Min(score, 1.0))
}

func (v *ARVerifierService) validatePosition(x, y, z float64, scene *ARScene) float64 {
	if scene.SceneType != "object_placement" {
		return 50
	}

	if len(scene.GesturePath) == 0 {
		return 50
	}

	lastPoint := scene.GesturePath[len(scene.GesturePath)-1]
	
	posXDiff := math.Abs(x - lastPoint.X)
	posYDiff := math.Abs(y - lastPoint.Y)
	posZDiff := 0.0

	posXScore := v.calculatePositionComponentScore(posXDiff)
	posYScore := v.calculatePositionComponentScore(posYDiff)
	posZScore := v.calculatePositionComponentScore(posZDiff)

	return (posXScore + posYScore + posZScore) / 3
}

func (v *ARVerifierService) calculatePositionComponentScore(diff float64) float64 {
	maxDiff := 0.2
	if diff <= 0.05 {
		return 1.0
	}
	if diff >= maxDiff {
		return 0.0
	}
	return math.Max(0, 1-diff/maxDiff)
}

func (v *ARVerifierService) validateObjectSelection(objectID int, scene *ARScene) float64 {
	if scene.SceneType != "object_tracking" && scene.SceneType != "spatial_puzzle" {
		return 50
	}

	if objectID == scene.TargetObject {
		return 1.0
	}

	return 0.0
}

func (v *ARVerifierService) isScoreSufficient(score float64, difficulty string) bool {
	threshold := v.getPassThreshold(difficulty)
	return score >= threshold
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
		return 85
	default:
		return 70
	}
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
