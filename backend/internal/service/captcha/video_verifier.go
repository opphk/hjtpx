package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
)

type VideoVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type VerifyVideoCaptchaRequest struct {
	SessionID   string                `json:"session_id" binding:"required"`
	Answer      string                `json:"answer" binding:"required"`
	BehaviorData VideoBehaviorData    `json:"behavior_data"`
}

type VideoBehaviorData struct {
	StartTime       int64   `json:"start_time"`
	EndTime         int64   `json:"end_time"`
	Duration        int64   `json:"duration"`
	ViewCount       int     `json:"view_count"`
	ReplayCount     int     `json:"replay_count"`
	IsMobile        bool    `json:"is_mobile"`
	DeviceType      string  `json:"device_type"`
	NetworkType     string  `json:"network_type"`
	Latency         int     `json:"latency"`
	ClickCount      int     `json:"click_count"`
	AnswerTime      int64   `json:"answer_time"`
}

type VideoVerifyResult struct {
	Success      bool                  `json:"success"`
	Message      string                `json:"message"`
	Score        float64               `json:"score"`
	SessionID    string                `json:"session_id"`
	CorrectAnswer string               `json:"correct_answer,omitempty"`
	RiskAnalysis *VideoRiskAnalysis     `json:"risk_analysis,omitempty"`
}

type VideoRiskAnalysis struct {
	IsBot          bool    `json:"is_bot"`
	Confidence     float64 `json:"confidence"`
	RiskScore      float64 `json:"risk_score"`
	RiskIndicators []string `json:"risk_indicators"`
}

func NewVideoVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *VideoVerifierService {
	return &VideoVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func NewVideoVerifierServiceSimple() *VideoVerifierService {
	return &VideoVerifierService{}
}

func (v *VideoVerifierService) Verify(ctx context.Context, req *VerifyVideoCaptchaRequest) (*VideoVerifyResult, error) {
	session, err := v.getSession(ctx, req.SessionID)
	if err != nil {
		return &VideoVerifyResult{
			Success:   false,
			Message:   "会话不存在",
			SessionID: req.SessionID,
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &VideoVerifyResult{
			Success:   false,
			Message:   "验证码已过期",
			SessionID: req.SessionID,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VideoVerifyResult{
			Success:   false,
			Message:   "验证次数已用完",
			SessionID: req.SessionID,
		}, nil
	}

	v.incrementVerifyCount(ctx, req.SessionID)

	if session.Status == "verified" {
		return &VideoVerifyResult{
			Success:   true,
			Message:   "验证码已验证通过",
			SessionID: req.SessionID,
			Score:     session.RiskScore,
		}, nil
	}

	riskAnalysis := v.analyzeBehavior(req.BehaviorData, session)
	
	answerCorrect := v.checkAnswer(session.TargetAction, req.Answer)
	
	if answerCorrect {
		v.markAsVerified(ctx, session.SessionID)
		
		adjustedScore := v.calculateScore(riskAnalysis, session)
		
		return &VideoVerifyResult{
			Success:      true,
			Message:      "验证成功",
			SessionID:    req.SessionID,
			Score:        adjustedScore,
			RiskAnalysis: riskAnalysis,
		}, nil
	}

	baseScore := 0.5
	if riskAnalysis != nil {
		baseScore = 1.0 - riskAnalysis.RiskScore
	}

	return &VideoVerifyResult{
		Success:       false,
		Message:       "答案错误",
		SessionID:     req.SessionID,
		Score:         baseScore,
		CorrectAnswer: session.TargetAction,
		RiskAnalysis:  riskAnalysis,
	}, nil
}

func (v *VideoVerifierService) getSession(ctx context.Context, sessionID string) (*VideoCaptchaSession, error) {
	if v.sessionCache != nil {
		sessionData, err := v.sessionCache.GetRaw(ctx, sessionID)
		if err == nil && sessionData != "" {
			var session VideoCaptchaSession
			if err := json.Unmarshal([]byte(sessionData), &session); err == nil {
				return &session, nil
			}
		}
	}
	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (v *VideoVerifierService) incrementVerifyCount(ctx context.Context, sessionID string) {
	if v.sessionCache != nil {
		sessionData, err := v.sessionCache.GetRaw(ctx, sessionID)
		if err == nil && sessionData != "" {
			var session VideoCaptchaSession
			if err := json.Unmarshal([]byte(sessionData), &session); err == nil {
				session.VerifyCount++
				sessionJSON, _ := json.Marshal(session)
				remainingTime := time.Until(session.ExpiredAt)
				if remainingTime > 0 {
					v.sessionCache.SetRaw(ctx, sessionID, string(sessionJSON), remainingTime)
				}
			}
		}
	}
}

func (v *VideoVerifierService) markAsVerified(ctx context.Context, sessionID string) {
	if v.sessionCache != nil {
		sessionData, err := v.sessionCache.GetRaw(ctx, sessionID)
		if err == nil && sessionData != "" {
			var session VideoCaptchaSession
			if err := json.Unmarshal([]byte(sessionData), &session); err == nil {
				session.Status = "verified"
				sessionJSON, _ := json.Marshal(session)
				remainingTime := time.Until(session.ExpiredAt)
				if remainingTime > 0 {
					v.sessionCache.SetRaw(ctx, sessionID, string(sessionJSON), remainingTime)
				}
			}
		}
	}
}

func (v *VideoVerifierService) checkAnswer(targetAction, userAnswer string) bool {
	if targetAction == userAnswer {
		return true
	}

	targetLower := strings.ToLower(targetAction)
	answerLower := strings.ToLower(userAnswer)
	
	if targetLower == answerLower {
		return true
	}

	if strings.Contains(targetLower, answerLower) || strings.Contains(answerLower, targetLower) {
		return true
	}

	actionAliases := map[string][]string{
		"举手":   {"举手", "举手过头顶", "抬起手", "举手过头"},
		"挥手":   {"挥手", "挥动手臂", "摆动手", "挥挥手"},
		"点头":   {"点头", "向下点头", "头部向下", "点点头"},
		"摇头":   {"摇头", "向左摇头", "向右摇头", "摇一摇"},
		"眨眼":   {"眨眼", "快速眨眼", "眨眼睛", "眨眨眼"},
		"张嘴":   {"张嘴", "张开嘴巴", "张大嘴", "张嘴吧"},
	}

	if aliases, ok := actionAliases[targetAction]; ok {
		for _, alias := range aliases {
			if strings.Contains(strings.ToLower(alias), answerLower) || 
			   strings.Contains(answerLower, strings.ToLower(alias)) {
				return true
			}
		}
	}

	return false
}

func (v *VideoVerifierService) analyzeBehavior(behaviorData VideoBehaviorData, session *VideoCaptchaSession) *VideoRiskAnalysis {
	riskIndicators := []string{}
	riskScore := 0.0
	confidence := 0.8

	if behaviorData.Duration > 0 {
		if behaviorData.Duration < 1000 {
			riskIndicators = append(riskIndicators, "响应时间过短")
			riskScore += 0.2
		}
		
		optimalDuration := int64(session.Duration) * 1000
		if behaviorData.Duration > int64(optimalDuration)*3 {
			riskIndicators = append(riskIndicators, "响应时间过长")
			riskScore += 0.1
		}
	}

	if behaviorData.ViewCount == 0 {
		riskIndicators = append(riskIndicators, "未观看视频")
		riskScore += 0.3
		confidence -= 0.3
	} else if behaviorData.ViewCount < 2 {
		riskIndicators = append(riskIndicators, "视频观看次数过少")
		riskScore += 0.1
		confidence -= 0.1
	}

	if behaviorData.ReplayCount > 3 {
		riskIndicators = append(riskIndicators, "视频重复播放次数过多")
		riskScore += 0.05
	}

	if behaviorData.AnswerTime > 0 && behaviorData.StartTime > 0 {
		answerDelay := behaviorData.AnswerTime - behaviorData.StartTime
		if answerDelay < 500 {
			riskIndicators = append(riskIndicators, "答题延迟过短")
			riskScore += 0.15
			confidence -= 0.15
		}
	}

	if riskScore >= 0.5 {
		riskIndicators = append(riskIndicators, "综合风险评分过高")
	}

	if len(riskIndicators) == 0 {
		riskIndicators = append(riskIndicators, "无异常")
	}

	return &VideoRiskAnalysis{
		IsBot:          riskScore >= 0.5,
		Confidence:     confidence,
		RiskScore:      riskScore,
		RiskIndicators: riskIndicators,
	}
}

func (v *VideoVerifierService) calculateScore(riskAnalysis *VideoRiskAnalysis, session *VideoCaptchaSession) float64 {
	baseScore := 0.85

	if riskAnalysis != nil {
		baseScore -= riskAnalysis.RiskScore * 0.5
		
		if riskAnalysis.Confidence < 0.5 {
			baseScore -= 0.1
		}
	}

	if session.Difficulty == 3 {
		baseScore += 0.05
	}

	if baseScore > 1.0 {
		baseScore = 1.0
	}
	if baseScore < 0.0 {
		baseScore = 0.0
	}

	return baseScore
}

func (v *VideoVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*VideoCaptchaSession, error) {
	return v.getSession(ctx, sessionID)
}

func (v *VideoVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
	session, err := v.getSession(ctx, sessionID)
	if err != nil {
		return false, "会话不存在"
	}

	if time.Now().After(session.ExpiredAt) {
		return false, "会话已过期"
	}

	if session.Status == "verified" {
		return false, "会话已验证"
	}

	if session.VerifyCount >= session.MaxAttempts {
		return false, "验证次数已用完"
	}

	return true, "会话有效"
}
