package captcha

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type VoiceVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type VoiceVerifyRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Code      string `json:"code" binding:"required"`
}

type VoiceVerifyResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func NewVoiceVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *VoiceVerifierService {
	return &VoiceVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (v *VoiceVerifierService) Verify(ctx context.Context, req *VoiceVerifyRequest) (*VoiceVerifyResult, error) {
	session, err := v.getSession(ctx, req.SessionID)
	if err != nil {
		return &VoiceVerifyResult{
			Success: false,
			Message: "Session not found",
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &VoiceVerifyResult{
			Success: false,
			Message: "验证码已过期",
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VoiceVerifyResult{
			Success: false,
			Message: "验证次数已用完",
		}, nil
	}

	v.incrementVerifyCount(ctx, session.SessionID)

	if session.Status == "verified" {
		return &VoiceVerifyResult{
			Success: true,
			Message: "验证码已验证通过",
		}, nil
	}

	if session.Code == req.Code {
		v.markAsVerified(ctx, session.SessionID)
		return &VoiceVerifyResult{
			Success: true,
			Message: "验证成功",
		}, nil
	}

	return &VoiceVerifyResult{
		Success: false,
		Message: "验证码错误",
	}, nil
}

func (v *VoiceVerifierService) getSession(ctx context.Context, sessionID string) (*models.VoiceCaptchaSession, error) {
	if v.sessionCache != nil {
		if session, err := v.sessionCache.GetVoice(ctx, sessionID); err == nil && session != nil {
			return session, nil
		}
	}

	if v.captchaRepo != nil {
		if session, err := v.captchaRepo.GetVoiceSession(sessionID); err == nil && session != nil {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found")
}

func (v *VoiceVerifierService) incrementVerifyCount(ctx context.Context, sessionID string) {
	if v.sessionCache != nil {
		if err := v.sessionCache.IncrementVoiceVerifyCount(ctx, sessionID); err != nil {
			log.Printf("增加语音验证计数失败: %v", err)
		}
	}

	if v.captchaRepo != nil {
		if err := v.captchaRepo.IncrementVoiceVerifyCount(sessionID); err != nil {
			log.Printf("数据库增加语音验证计数失败: %v", err)
		}
	}
}

func (v *VoiceVerifierService) markAsVerified(ctx context.Context, sessionID string) {
	if v.sessionCache != nil {
		if err := v.sessionCache.MarkVoiceAsVerified(ctx, sessionID); err != nil {
			log.Printf("缓存标记语音验证失败: %v", err)
		}
	}

	if v.captchaRepo != nil {
		if err := v.captchaRepo.MarkVoiceAsVerified(sessionID); err != nil {
			log.Printf("数据库标记语音验证失败: %v", err)
		}
	}
}
