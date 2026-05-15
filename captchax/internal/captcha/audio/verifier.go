package audio

import (
	"context"
	"errors"
	"strings"
	"time"
)

type Verifier struct {
	cacheManager *CacheManager
}

func NewVerifier(cacheManager *CacheManager) *Verifier {
	return &Verifier{
		cacheManager: cacheManager,
	}
}

func (v *Verifier) Verify(ctx context.Context, captchaID, userCode string) (*VerifyResult, error) {
	if captchaID == "" {
		return &VerifyResult{
			Success: false,
			Message: "captcha ID is required",
		}, nil
	}

	if userCode == "" {
		return &VerifyResult{
			Success: false,
			Message: "verification code is required",
		}, nil
	}

	data, err := v.cacheManager.Get(ctx, captchaID)
	if err != nil {
		if errors.Is(err, ErrCaptchaNotFound) {
			return &VerifyResult{
				Success: false,
				Message: "captcha not found or expired",
			}, nil
		}
		return nil, err
	}

	if data.Verified {
		return &VerifyResult{
			Success: false,
			Message: "captcha already verified",
		}, nil
	}

	userCode = strings.ToUpper(strings.TrimSpace(userCode))
	expectedCode := strings.ToUpper(data.Code)

	if !v.validateCodeFormat(userCode) {
		return &VerifyResult{
			Success: false,
			Message: "invalid code format",
		}, nil
	}

	success := userCode == expectedCode

	if success {
		if err := v.cacheManager.MarkVerified(ctx, captchaID); err != nil {
			return nil, err
		}
	}

	remainingAttempts := v.calculateRemainingAttempts(data.CreatedAt)

	if !success && remainingAttempts <= 0 {
		_ = v.cacheManager.Delete(ctx, captchaID)
		return &VerifyResult{
			Success: false,
			Message: "too many failed attempts",
		}, nil
	}

	if success {
		return &VerifyResult{
			Success: true,
			Message: "verification successful",
		}, nil
	}

	return &VerifyResult{
		Success: false,
		Message: "incorrect verification code",
	}, nil
}

func (v *Verifier) validateCodeFormat(code string) bool {
	if len(code) < 4 || len(code) > 8 {
		return false
	}

	for _, c := range code {
		if !isValidChar(c) {
			return false
		}
	}

	return true
}

func isValidChar(c rune) bool {
	return (c >= '0' && c <= '9') ||
		(c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z')
}

func (v *Verifier) calculateRemainingAttempts(createdAt int64) int {
	maxAttempts := 5
	elapsed := time.Now().Unix() - createdAt
	decayFactor := elapsed / 60

	remaining := maxAttempts - int(decayFactor)
	if remaining < 1 {
		return 1
	}
	return remaining
}
