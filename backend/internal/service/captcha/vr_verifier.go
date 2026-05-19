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

type VRVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

func NewVRVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *VRVerifierService {
	return &VRVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func NewVRVerifierServiceSimple() *VRVerifierService {
	return &VRVerifierService{}
}

func (v *VRVerifierService) Verify(ctx context.Context, req *VRVerifyRequest) (*VRVerifyResponse, error) {
	session, err := v.GetSession(ctx, req.SessionID)
	if err != nil {
		return &VRVerifyResponse{
			Success: false,
			Score:   0,
			Message: "会话不存在或已过期",
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &VRVerifyResponse{
			Success: false,
			Score:   0,
			Message: "会话已过期",
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VRVerifyResponse{
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

	return &VRVerifyResponse{
		Success:   success,
		Score:     totalScore,
		Message:   map[bool]string{true: "验证成功", false: "验证失败"}[success],
		Accuracy:  accuracy,
		Feedback:  feedback,
		Analytics: analytics,
	}, nil
}

func (v *VRVerifierService) GetSession(ctx context.Context, sessionID string) (*VRSession, error) {
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

func (v *VRVerifierService) saveSession(ctx context.Context, session *VRSession) error {
	if v.sessionCache != nil {
		return v.sessionCache.SetRaw(ctx, session.SessionID, string(MarshalSession(session)), 5*time.Minute)
	}
	return nil
}

func (v *VRVerifierService) getCachedSession(ctx context.Context, sessionID string) (*VRSession, error) {
	return nil, fmt.Errorf("session not found in cache")
}

func (v *VRVerifierService) getDatabaseSession(sessionID string) (*VRSession, error) {
	return nil, fmt.Errorf("session not found in database")
}

func (v *VRVerifierService) evaluateInteraction(req *VRVerifyRequest, session *VRSession) float64 {
	totalScore := 0.0
	weightSum := 0.0

	if req.Interaction != nil && req.Interaction.ObjectPositions != nil {
		for _, target := range session.VRConfig.Targets {
			if target.ObjectID == "" {
				continue
			}

			pos, ok := req.Interaction.ObjectPositions[target.ObjectID]
			if !ok || len(pos) < 3 {
				continue
			}

			distance := calculateDistance3D_vr(pos, target.Position)

			for _, constraint := range session.VRConfig.Constraints {
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
		for _, obj := range session.VRConfig.Objects {
			if obj.TargetRotation == nil {
				continue
			}

			rot, ok := req.Interaction.ObjectRotations[obj.ID]
			if !ok || len(rot) < 3 {
				continue
			}

			angleDiff := calculateAngleDifference_vr(rot, obj.TargetRotation)

			for _, constraint := range session.VRConfig.Constraints {
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

func (v *VRVerifierService) evaluateEyeTracking(eyeData *VREyeTrackingData, session *VRSession) float64 {
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

func (v *VRVerifierService) calculateAccuracy(req *VRVerifyRequest, session *VRSession) float64 {
	if session.VRConfig.Type == VRCaptcha3DPlacement {
		if req.Interaction == nil || req.Interaction.ObjectPositions == nil {
			return 0
		}

		successCount := 0
		totalCount := 0

		for _, target := range session.VRConfig.Targets {
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
	}

	if session.VRConfig.Type == VRCaptchaHandTracking || session.VRConfig.Type == VRCaptchaVRGesture {
		if req.GestureData == nil {
			return 0
		}
		if req.GestureData.Recognized && req.GestureData.Confidence >= 0.7 {
			return 1.0
		}
		return req.GestureData.Confidence
	}

	return 0.5
}

func (v *VRVerifierService) calculateBehaviorScore(behaviorData map[string]interface{}) float64 {
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

func (v *VRVerifierService) generateFeedback(req *VRVerifyRequest, session *VRSession) string {
	switch session.VRConfig.Type {
	case VRCaptcha3DPlacement:
		if req.Interaction != nil && req.Interaction.ObjectPositions != nil {
			return "物体位置不够精确，请更仔细地放置"
		}
		return "请将物体抓取并放置到目标区域"

	case VRCaptchaHandTracking, VRCaptchaVRGesture:
		return "手势未能识别，请再试一次"

	case VRCaptchaSpatialPuzzle:
		return "拼图顺序或位置不正确，请继续尝试"

	case VRCaptchaEyeTracking:
		return "请保持注视目标点"
	}

	return "请仔细按照提示完成操作"
}

func (v *VRVerifierService) generateAnalytics(req *VRVerifyRequest, session *VRSession) *VRAnalytics {
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

func (v *VRVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*VRSession, error) {
	return v.GetSession(ctx, sessionID)
}

func (v *VRVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
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

func MarshalSession(session *VRSession) []byte {
	data, _ := json.Marshal(session)
	return data
}

func calculateDistance3D_vr(pos1, pos2 []float64) float64 {
	if len(pos1) < 3 || len(pos2) < 3 {
		return 0
	}
	return math.Sqrt(
		math.Pow(pos1[0]-pos2[0], 2) +
			math.Pow(pos1[1]-pos2[1], 2) +
			math.Pow(pos1[2]-pos2[2], 2),
	)
}

func calculateAngleDifference_vr(rot1, rot2 []float64) float64 {
	if len(rot1) < 3 || len(rot2) < 3 {
		return 0
	}
	var totalDiff float64
	for i := 0; i < 3; i++ {
		diff := math.Abs(rot1[i] - rot2[i])
		for diff > 180 {
			diff = 360 - diff
		}
		totalDiff += diff
	}
	return totalDiff / 3
}
