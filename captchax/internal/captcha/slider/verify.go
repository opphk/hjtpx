package slider

import (
	"captchax/config"
	"context"
	"errors"
	"fmt"
)

var (
	ErrInvalidX     = errors.New("invalid x position")
	ErrInvalidY     = errors.New("invalid y position")
	ErrInvalidID    = errors.New("invalid captcha id")
	ErrVerification = errors.New("verification failed")
)

type VerifyRequest struct {
	CaptchaID string `json:"captcha_id"`
	TargetX   int    `json:"target_x"`
	TargetY   int    `json:"target_y"`
}

type VerifyResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type CacheManagerInterface interface {
	Get(ctx context.Context, id string) (*CacheData, error)
	Delete(ctx context.Context, id string) error
	Set(ctx context.Context, id string, data *CacheData) error
	Exists(ctx context.Context, id string) (bool, error)
}

type VerifyService struct {
	cache CacheManagerInterface
	cfg   *config.CaptchaConfig
}

func NewVerifyService(cfg *config.CaptchaConfig, cache *CacheManager) *VerifyService {
	return &VerifyService{
		cache: cache,
		cfg:   cfg,
	}
}

func NewVerifyServiceWithInterface(cfg *config.CaptchaConfig, cache CacheManagerInterface) *VerifyService {
	return &VerifyService{
		cache: cache,
		cfg:   cfg,
	}
}

func (vs *VerifyService) Verify(ctx context.Context, req *VerifyRequest) (*VerifyResult, error) {
	if req.CaptchaID == "" {
		return &VerifyResult{
			Success: false,
			Message: "captcha id is required",
		}, ErrInvalidID
	}

	captchaData, err := vs.cache.Get(ctx, req.CaptchaID)
	if err != nil {
		if errors.Is(err, ErrCaptchaNotFound) {
			return &VerifyResult{
				Success: false,
				Message: "captcha not found or expired",
			}, ErrCaptchaNotFound
		}
		if errors.Is(err, ErrCaptchaExpired) {
			return &VerifyResult{
				Success: false,
				Message: "captcha has expired",
			}, ErrCaptchaExpired
		}
		return nil, fmt.Errorf("failed to get captcha data: %w", err)
	}

	if captchaData.Verified {
		return &VerifyResult{
			Success: false,
			Message: "captcha already verified",
		}, ErrCaptchaVerified
	}

	tolerance := vs.cfg.Tolerance
	if tolerance == 0 {
		tolerance = 5
	}

	if !vs.isWithinTolerance(req.TargetX, captchaData.TargetX, tolerance) {
		return &VerifyResult{
			Success: false,
			Message: fmt.Sprintf("verification failed: x position off by %d (tolerance: %d)",
				absDiff(req.TargetX, captchaData.TargetX), tolerance),
		}, ErrVerification
	}

	if !vs.isWithinTolerance(req.TargetY, captchaData.TargetY, tolerance) {
		return &VerifyResult{
			Success: false,
			Message: fmt.Sprintf("verification failed: y position off by %d (tolerance: %d)",
				absDiff(req.TargetY, captchaData.TargetY), tolerance),
		}, ErrVerification
	}

	if err := vs.cache.Delete(ctx, req.CaptchaID); err != nil {
		return nil, fmt.Errorf("failed to delete captcha after verification: %w", err)
	}

	return &VerifyResult{
		Success: true,
		Message: "verification successful",
	}, nil
}

func (vs *VerifyService) isWithinTolerance(actual, expected, tolerance int) bool {
	diff := absDiff(actual, expected)
	return diff <= tolerance
}

func absDiff(x, y int) int {
	if x < 0 {
		x = -x
	}
	if y < 0 {
		y = -y
	}
	diff := x - y
	if diff < 0 {
		return -diff
	}
	return diff
}

func (vs *VerifyService) ValidatePosition(x, y int) error {
	if x < 0 {
		return ErrInvalidX
	}
	if y < 0 {
		return ErrInvalidY
	}
	return nil
}

type VerifyStats struct {
	TotalVerified int64   `json:"total_verified"`
	TotalFailed   int64   `json:"total_failed"`
	SuccessRate   float64 `json:"success_rate"`
}
