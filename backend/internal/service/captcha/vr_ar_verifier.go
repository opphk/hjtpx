package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
)

type VRARVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
	vrVerifier   *VRVerifierService
}

func NewVRARVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *VRARVerifierService {
	return &VRARVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
		vrVerifier:   NewVRVerifierService(sessionCache, captchaRepo),
	}
}

func NewVRARVerifierServiceSimple() *VRARVerifierService {
	return &VRARVerifierService{
		vrVerifier: NewVRVerifierServiceSimple(),
	}
}

func (v *VRARVerifierService) Verify(ctx context.Context, req *VRARVerifyRequest) (*VRARVerifyResponse, error) {
	session, err := v.GetSession(ctx, req.SessionID)
	if err != nil {
		return &VRARVerifyResponse{
			Success: false,
			Score:   0,
			Message: "会话不存在或已过期",
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &VRARVerifyResponse{
			Success: false,
			Score:   0,
			Message: "会话已过期",
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VRARVerifyResponse{
			Success: false,
			Score:   0,
			Message: "验证次数已用完",
		}, nil
	}

	session.VerifyCount++
	v.saveSession(ctx, session)

	totalScore := v.evaluateInteraction(req, session)
	accuracy := v.calculateAccuracy(req, session)
	success := totalScore >= 0.6 && accuracy >= 0.5

	feedback := ""
	if !success {
		feedback = v.generateFeedback(req, session)
	}

	analytics := v.generateAnalytics(req, session)

	if success {
		session.Status = "verified"
		v.saveSession(ctx, session)
	}

	return &VRARVerifyResponse{
		Success:   success,
		Score:     totalScore,
		Message:   map[bool]string{true: "验证成功", false: "验证失败"}[success],
		Accuracy:  accuracy,
		Feedback:  feedback,
		Analytics: analytics,
	}, nil
}

func (v *VRARVerifierService) GetSession(ctx context.Context, sessionID string) (*VRARSession, error) {
	if v.sessionCache != nil {
		session, err := v.getCachedSession(ctx, sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	if v.captchaRepo != nil {
		session, err := v.getDatabaseSession(sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (v *VRARVerifierService) saveSession(ctx context.Context, session *VRARSession) error {
	if v.sessionCache != nil {
		data, _ := json.Marshal(session)
		return v.sessionCache.SetRaw(ctx, session.SessionID, string(data), 5*time.Minute)
	}
	return nil
}

func (v *VRARVerifierService) getCachedSession(ctx context.Context, sessionID string) (*VRARSession, error) {
	data, err := v.sessionCache.GetRaw(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	var session VRARSession
	err = json.Unmarshal([]byte(data), &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (v *VRARVerifierService) getDatabaseSession(sessionID string) (*VRARSession, error) {
	return nil, fmt.Errorf("session not found in database")
}

func (v *VRARVerifierService) evaluateInteraction(req *VRARVerifyRequest, session *VRARSession) float64 {
	totalScore := 0.0
	weightSum := 0.0

	if req.Interaction != nil && req.Interaction.ObjectPositions != nil {
		for _, target := range session.SceneConfig.Targets {
			if target.ObjectID == "" {
				continue
			}

			pos, ok := req.Interaction.ObjectPositions[target.ObjectID]
			if !ok || len(pos) < 3 {
				continue
			}

			distance := calculateDistance3D_vr(pos, target.Position)

			for _, constraint := range session.SceneConfig.Constraints {
				if constraint.TargetID == target.ObjectID && constraint.Type == "distance" {
					tolerance := constraint.Tolerance
					var objScore float64
					if distance <= tolerance {
						objScore = 1.0
					} else {
						objScore = math.Max(0, 1.0-distance/tolerance)
					}
					totalScore += objScore * constraint.Weight
					weightSum += constraint.Weight
				}
			}
		}
	}

	if req.Interaction != nil && req.Interaction.ObjectRotations != nil {
		for _, obj := range session.SceneConfig.Objects {
			if obj.TargetRotation == nil {
				continue
			}

			rot, ok := req.Interaction.ObjectRotations[obj.ID]
			if !ok || len(rot) < 3 {
				continue
			}

			angleDiff := calculateAngleDifference_vr(rot, obj.TargetRotation)

			for _, constraint := range session.SceneConfig.Constraints {
				if constraint.TargetID == obj.ID && constraint.Type == "angle" {
					tolerance := constraint.Tolerance
					var rotScore float64
					if angleDiff <= tolerance {
						rotScore = 1.0
					} else {
						rotScore = math.Max(0, 1.0-angleDiff/tolerance)
					}
					totalScore += rotScore * constraint.Weight
					weightSum += constraint.Weight
				}
			}
		}
	}

	if req.GestureData != nil {
		if req.GestureData.Recognized && req.GestureData.Confidence >= 0.7 {
			totalScore += 0.3
			weightSum += 0.3
		} else if req.GestureData.Recognized {
			totalScore += 0.15
			weightSum += 0.3
		}
	}

	if req.ARGesture != nil {
		if req.ARGesture.Recognized && req.ARGesture.Confidence >= 0.7 {
			totalScore += 0.3
			weightSum += 0.3
		} else if req.ARGesture.Recognized {
			totalScore += 0.15
			weightSum += 0.3
		}
	}

	if req.EyeData != nil {
		eyeScore := v.evaluateEyeTracking(req.EyeData, session)
		totalScore += eyeScore * 0.4
		weightSum += 0.4
	}

	if req.BehaviorData != nil {
		behaviorScore := v.calculateBehaviorScore(req.BehaviorData)
		totalScore += behaviorScore * 0.2
		weightSum += 0.2
	}

	if weightSum > 0 {
		return totalScore / weightSum
	}

	return totalScore
}

func (v *VRARVerifierService) evaluateEyeTracking(eyeData *VREyeTrackingData, session *VRARSession) float64 {
	if eyeData.Confidence < 0.5 {
		return 0.3
	}

	score := eyeData.Confidence * 0.5

	if len(eyeData.FixationPoints) > 2 {
		score += 0.3
	}

	if eyeData.PupilDiameter > 2 && eyeData.PupilDiameter < 8 {
		score += 0.2
	}

	return math.Min(score, 1.0)
}

func (v *VRARVerifierService) calculateAccuracy(req *VRARVerifyRequest, session *VRARSession) float64 {
	switch session.SceneConfig.Type {
	case VRARType3DPlacement:
		if req.Interaction == nil || req.Interaction.ObjectPositions == nil {
			return 0
		}

		successCount := 0
		totalCount := 0

		for _, target := range session.SceneConfig.Targets {
			if target.ObjectID == "" {
				continue
			}
			totalCount++

			pos, ok := req.Interaction.ObjectPositions[target.ObjectID]
			if !ok {
				continue
			}

			distance := calculateDistance3D_vr(pos, target.Position)
			if distance <= 0.3 {
				successCount++
			}
		}

		if totalCount == 0 {
			return 1.0
		}
		return float64(successCount) / float64(totalCount)

	case VRARTypeGesture:
		if req.GestureData == nil && req.ARGesture == nil {
			return 0
		}
		if req.GestureData != nil && req.GestureData.Recognized && req.GestureData.Confidence >= 0.7 {
			return 1.0
		}
		if req.ARGesture != nil && req.ARGesture.Recognized && req.ARGesture.Confidence >= 0.7 {
			return 1.0
		}
		if req.GestureData != nil {
			return req.GestureData.Confidence
		}
		if req.ARGesture != nil {
			return req.ARGesture.Confidence
		}
		return 0

	case VRARTypeObjectRotation:
		if req.Interaction == nil || req.Interaction.ObjectRotations == nil {
			return 0
		}

		successCount := 0
		totalCount := 0

		for _, obj := range session.SceneConfig.Objects {
			if obj.TargetRotation == nil {
				continue
			}
			totalCount++

			rot, ok := req.Interaction.ObjectRotations[obj.ID]
			if !ok {
				continue
			}

			angleDiff := calculateAngleDifference_vr(rot, obj.TargetRotation)
			if angleDiff <= 30.0 {
				successCount++
			}
		}

		if totalCount == 0 {
			return 1.0
		}
		return float64(successCount) / float64(totalCount)

	case VRARTypeSequential, VRARTypeSpatialPuzzle:
		return 0.5
	}

	return 0.5
}

func (v *VRARVerifierService) calculateBehaviorScore(behaviorData map[string]interface{}) float64 {
	score := 0.5

	if moveCount, ok := behaviorData["move_count"].(float64); ok {
		if moveCount > 3 && moveCount < 20 {
			score += 0.1
		}
	}

	if timeSpent, ok := behaviorData["time_spent"].(float64); ok {
		if timeSpent >= 3.0 && timeSpent <= 60.0 {
			score += 0.15
		}
	}

	if accuracy, ok := behaviorData["accuracy"].(float64); ok {
		score += accuracy * 0.25
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (v *VRARVerifierService) generateFeedback(req *VRARVerifyRequest, session *VRARSession) string {
	switch session.SceneConfig.Type {
	case VRARType3DPlacement:
		if req.Interaction != nil && req.Interaction.ObjectPositions != nil {
			return "物体位置不够精确，请更仔细地放置"
		}
		return "请将物体抓取并放置到目标区域"

	case VRARTypeGesture:
		return "手势未能识别，请再试一次"

	case VRARTypeObjectRotation:
		return "物体角度不正确，请继续调整旋转"

	case VRARTypeSpatialPuzzle:
		return "拼图顺序或位置不正确，请继续尝试"

	case VRARTypeEyeTracking:
		return "请保持注视目标点"

	case VRARTypeSequential:
		return "请按正确顺序点击物体"
	}

	return "请仔细按照提示完成操作"
}

func (v *VRARVerifierService) generateAnalytics(req *VRARVerifyRequest, session *VRARSession) *VRAnalytics {
	analytics := &VRAnalytics{
		ErrorCount: session.VerifyCount - 1,
	}

	if req.Interaction != nil {
		analytics.CompletionTime = req.Interaction.TimeSpent
		analytics.MovementCount = req.Interaction.MovementCount
	}

	if req.GestureData != nil {
		analytics.HandDominance = req.GestureData.Hand
	}

	return analytics
}

func (v *VRARVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*VRARSession, error) {
	return v.GetSession(ctx, sessionID)
}

func (v *VRARVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
	session, err := v.GetSession(ctx, sessionID)
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
