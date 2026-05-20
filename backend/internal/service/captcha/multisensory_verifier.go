package captcha

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type MultisensoryVerifierService struct {
}

type MultisensoryVerifyRequest struct {
	SessionID  string            `json:"session_id" binding:"required"`
	Answers    map[string]string `json:"answers"` // type -> answer
	RequireAll bool              `json:"require_all"`
}

type MultisensoryVerifyResult struct {
	Success    bool            `json:"success"`
	Message    string          `json:"message"`
	Verified   map[string]bool `json:"verified"`
	AllPassed  bool            `json:"all_passed"`
}

func NewMultisensoryVerifierServiceSimple() *MultisensoryVerifierService {
	return &MultisensoryVerifierService{}
}

func (v *MultisensoryVerifierService) Verify(ctx context.Context, req *MultisensoryVerifyRequest) (*MultisensoryVerifyResult, error) {
	session, err := v.getSession(req.SessionID)
	if err != nil {
		return &MultisensoryVerifyResult{
			Success:  false,
			Message:  "Session not found",
			Verified: make(map[string]bool),
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &MultisensoryVerifyResult{
			Success:  false,
			Message:  "验证码已过期",
			Verified: session.Verified,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &MultisensoryVerifyResult{
			Success:  false,
			Message:  "验证次数已用完",
			Verified: session.Verified,
		}, nil
	}

	v.incrementVerifyCount(req.SessionID)

	if session.Status == "verified" {
		return &MultisensoryVerifyResult{
			Success:   true,
			Message:   "验证码已验证通过",
			Verified:  session.Verified,
			AllPassed: true,
		}, nil
	}

	verified := make(map[string]bool)
	for t := range session.Verified {
		verified[t] = session.Verified[t]
	}

	for t, answer := range req.Answers {
		if !verified[t] {
			correct, err := v.verifyType(session, t, answer)
			if err == nil && correct {
				verified[t] = true
			}
		}
	}

	allPassed := true
	for _, t := range session.Types {
		if !verified[t] {
			allPassed = false
			break
		}
	}

	result := &MultisensoryVerifyResult{
		Success:   !req.RequireAll || allPassed,
		Message:   "",
		Verified:  verified,
		AllPassed: allPassed,
	}

	if allPassed {
		v.markAsVerified(req.SessionID, verified)
		result.Message = "验证成功"
	} else if req.RequireAll {
		result.Message = "请完成所有验证"
	} else {
		result.Message = "部分验证通过"
	}

	return result, nil
}

func (v *MultisensoryVerifierService) verifyType(session *MultisensoryCaptchaSession, captchaType, answer string) (bool, error) {
	switch captchaType {
	case "visual":
		return v.verifyVisual(session, answer)
	case "audio":
		return v.verifyAudio(session, answer)
	case "tactile":
		return v.verifyTactile(session, answer)
	default:
		return false, fmt.Errorf("unknown captcha type: %s", captchaType)
	}
}

func (v *MultisensoryVerifierService) verifyVisual(session *MultisensoryCaptchaSession, answer string) (bool, error) {
	if session.VisualAnswer == "" {
		return false, fmt.Errorf("no visual answer")
	}

	parts := strings.Split(session.VisualAnswer, ",")
	if len(parts) == 2 {
		targetX, _ := strconv.Atoi(parts[0])
		targetY, _ := strconv.Atoi(parts[1])

		answerParts := strings.Split(answer, ",")
		if len(answerParts) == 2 {
			userX, err := strconv.Atoi(answerParts[0])
			if err != nil {
				return false, err
			}
			userY, err := strconv.Atoi(answerParts[1])
			if err != nil {
				return false, err
			}

			tolerance := 5
			return abs(targetX-userX) <= tolerance && abs(targetY-userY) <= tolerance, nil
		}
	}

	return session.VisualAnswer == answer, nil
}

func (v *MultisensoryVerifierService) verifyAudio(session *MultisensoryCaptchaSession, answer string) (bool, error) {
	if session.AudioAnswer == "" {
		return false, fmt.Errorf("no audio answer")
	}
	return session.AudioAnswer == answer, nil
}

func (v *MultisensoryVerifierService) verifyTactile(session *MultisensoryCaptchaSession, answer string) (bool, error) {
	if session.TactileAnswer == "" {
		return false, fmt.Errorf("no tactile answer")
	}
	return session.TactileAnswer == answer, nil
}

func (v *MultisensoryVerifierService) getSession(sessionID string) (*MultisensoryCaptchaSession, error) {
	multisensoryMu.RLock()
	session, ok := multisensorySessions[sessionID]
	multisensoryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

func (v *MultisensoryVerifierService) incrementVerifyCount(sessionID string) {
	multisensoryMu.Lock()
	if session, ok := multisensorySessions[sessionID]; ok {
		session.VerifyCount++
	}
	multisensoryMu.Unlock()
}

func (v *MultisensoryVerifierService) markAsVerified(sessionID string, verified map[string]bool) {
	multisensoryMu.Lock()
	if session, ok := multisensorySessions[sessionID]; ok {
		session.Status = "verified"
		session.Verified = verified
	}
	multisensoryMu.Unlock()
}

func (v *MultisensoryVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*MultisensoryCaptchaSession, error) {
	return v.getSession(sessionID)
}
