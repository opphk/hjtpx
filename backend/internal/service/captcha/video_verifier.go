package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

type VideoVerifierService struct {
}

type VideoVerifyRequest struct {
	SessionID    string   `json:"session_id" binding:"required"`
	Answer       string   `json:"answer"`           // 文本答案（内容理解）
	ActionResult []int    `json:"action_result"`    // 动作序列（动作识别）
	Sequence     []string `json:"sequence"`         // 序列答案（序列验证）
	RiskScore    float64  `json:"risk_score"`
	TraceScore   float64  `json:"trace_score"`
	EnvScore     float64  `json:"env_score"`
}

type VideoVerifyResult struct {
	Success      bool    `json:"success"`
	Message      string  `json:"message"`
	Score        float64 `json:"score"`
	SessionID    string  `json:"session_id"`
	AttemptsLeft int     `json:"attempts_left"`
}

func NewVideoVerifierService() *VideoVerifierService {
	return &VideoVerifierService{}
}

func (v *VideoVerifierService) Verify(ctx context.Context, req *VideoVerifyRequest) (*VideoVerifyResult, error) {
	session := v.getSession(req.SessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", req.SessionID)
	}

	if time.Now().After(session.ExpiredAt) {
		return &VideoVerifyResult{
			Success:      false,
			Message:      "验证码已过期",
			Score:        0,
			SessionID:    req.SessionID,
			AttemptsLeft: 0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VideoVerifyResult{
			Success:      false,
			Message:      "验证次数已用完",
			Score:        0,
			SessionID:    req.SessionID,
			AttemptsLeft: 0,
		}, nil
	}

	if session.Status == "verified" {
		return &VideoVerifyResult{
			Success:      true,
			Message:      "验证码已验证通过",
			Score:        100,
			SessionID:    req.SessionID,
			AttemptsLeft: session.MaxAttempts - session.VerifyCount,
		}, nil
	}

	videoType := VideoCaptchaType(session.Type)
	var success bool
	var score float64
	var message string

	switch videoType {
	case VideoTypeContent:
		success, score, message = v.verifyContent(session, req.Answer)
	case VideoTypeAction:
		success, score, message = v.verifyAction(session, req.ActionResult)
	case VideoTypeSequence:
		success, score, message = v.verifySequence(session, req.Sequence)
	default:
		success, score, message = v.verifyContent(session, req.Answer)
	}

	if success {
		session.Status = "verified"
		return &VideoVerifyResult{
			Success:      true,
			Message:      message,
			Score:        score,
			SessionID:    req.SessionID,
			AttemptsLeft: session.MaxAttempts - session.VerifyCount - 1,
		}, nil
	}

	session.VerifyCount++
	return &VideoVerifyResult{
		Success:      false,
		Message:      message,
		Score:        score,
		SessionID:    req.SessionID,
		AttemptsLeft: session.MaxAttempts - session.VerifyCount - 1,
	}, nil
}

func (v *VideoVerifierService) verifyContent(session *models.VideoCaptchaSession, answer string) (bool, float64, string) {
	if strings.TrimSpace(answer) == "" {
		return false, 0, "请提供答案"
	}

	if strings.EqualFold(strings.TrimSpace(answer), session.CorrectAnswer) {
		return true, 100, "验证成功"
	}

	return false, 0, "答案不正确，请重试"
}

func (v *VideoVerifierService) verifyAction(session *models.VideoCaptchaSession, actionResult []int) (bool, float64, string) {
	if actionResult == nil || len(actionResult) == 0 {
		return false, 0, "请提供动作序列"
	}

	var expectedPattern []int
	if session.ActionPattern != "" {
		err := json.Unmarshal([]byte(session.ActionPattern), &expectedPattern)
		if err != nil {
			return false, 0, "无法解析动作模式"
		}
	}

	if len(actionResult) != len(expectedPattern) {
		score := calculateActionScore(actionResult, expectedPattern)
		return false, score, "动作序列长度不正确"
	}

	if reflect.DeepEqual(actionResult, expectedPattern) {
		return true, 100, "动作验证成功"
	}

	score := calculateActionScore(actionResult, expectedPattern)
	return false, score, "动作序列不正确，请重试"
}

func (v *VideoVerifierService) verifySequence(session *models.VideoCaptchaSession, sequence []string) (bool, float64, string) {
	if sequence == nil || len(sequence) == 0 {
		return false, 0, "请提供序列答案"
	}

	var expectedSequence []string
	if session.SequenceData != "" {
		err := json.Unmarshal([]byte(session.SequenceData), &expectedSequence)
		if err != nil {
			return false, 0, "无法解析序列数据"
		}
	}

	if len(sequence) != len(expectedSequence) {
		return false, 0, "序列长度不正确"
	}

	for i := 0; i < len(sequence); i++ {
		if !strings.EqualFold(strings.TrimSpace(sequence[i]), expectedSequence[i]) {
			score := float64(i) / float64(len(expectedSequence)) * 100
			return false, score, "序列不正确，请重试"
		}
	}

	return true, 100, "序列验证成功"
}

func calculateActionScore(actual, expected []int) float64 {
	if len(actual) == 0 || len(expected) == 0 {
		return 0
	}

	matchingPositions := 0
	minLen := min(len(actual), len(expected))

	for i := 0; i < minLen; i++ {
		if actual[i] == expected[i] {
			matchingPositions++
		}
	}

	score := float64(matchingPositions) / float64(len(expected)) * 100
	return score
}

func (v *VideoVerifierService) getSession(sessionID string) *models.VideoCaptchaSession {
	return &models.VideoCaptchaSession{
		SessionID:     sessionID,
		Type:          string(VideoTypeContent),
		Question:      "视频中出现的动物是什么？",
		CorrectAnswer: "猫",
		Options:       `["狗", "猫", "鸟", "鱼"]`,
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
		ExpiredAt:     time.Now().Add(5 * time.Minute),
	}
}

func (v *VideoVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*models.VideoCaptchaSession, error) {
	session := v.getSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return session, nil
}

func (v *VideoVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
	session := v.getSession(sessionID)
	if session == nil {
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

