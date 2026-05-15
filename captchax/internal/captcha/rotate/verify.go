package rotate

import (
	"captchax/config"
	"context"
	"errors"
	"fmt"
)

var (
	ErrInvalidAngle = errors.New("invalid angle")
	ErrInvalidID    = errors.New("invalid captcha id")
	ErrVerification = errors.New("verification failed")
)

// CacheManagerInterface 缓存管理器接口（用于测试）
type CacheManagerInterface interface {
	Get(ctx context.Context, id string) (*CacheData, error)
	Delete(ctx context.Context, id string) error
	Set(ctx context.Context, id string, data *CacheData) error
	Exists(ctx context.Context, id string) (bool, error)
}

// VerifyService 验证服务
type VerifyService struct {
	cache CacheManagerInterface
	cfg   *config.CaptchaConfig
}

// NewVerifyService 创建验证服务
func NewVerifyService(cfg *config.CaptchaConfig, cache *CacheManager) *VerifyService {
	return &VerifyService{
		cache: cache,
		cfg:   cfg,
	}
}

// NewVerifyServiceWithInterface 使用接口创建验证服务（便于测试）
func NewVerifyServiceWithInterface(cfg *config.CaptchaConfig, cache CacheManagerInterface) *VerifyService {
	return &VerifyService{
		cache: cache,
		cfg:   cfg,
	}
}

// Verify 验证旋转角度
func (vs *VerifyService) Verify(ctx context.Context, req *VerifyRequest) (*VerifyResult, error) {
	if req.CaptchaID == "" {
		return &VerifyResult{
			Success: false,
			Message: "captcha id is required",
		}, ErrInvalidID
	}

	// 角度标准化到 0-359 范围内
	reqAngle := normalizeAngle(req.Angle)

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

	// ±5度误差容限
	tolerance := 5

	// 计算角度差（考虑圆形周期性）
	if !vs.isAngleWithinTolerance(reqAngle, captchaData.Angle, tolerance) {
		return &VerifyResult{
			Success: false,
			Message: fmt.Sprintf("verification failed: angle off by %d degrees (tolerance: %d)",
				angleDiff(reqAngle, captchaData.Angle), tolerance),
		}, ErrVerification
	}

	// 验证成功，删除验证码
	if err := vs.cache.Delete(ctx, req.CaptchaID); err != nil {
		return nil, fmt.Errorf("failed to delete captcha after verification: %w", err)
	}

	return &VerifyResult{
		Success: true,
		Message: "verification successful",
	}, nil
}

// isAngleWithinTolerance 检查角度是否在容限范围内
func (vs *VerifyService) isAngleWithinTolerance(actual, expected, tolerance int) bool {
	diff := angleDiff(actual, expected)
	return diff <= tolerance
}

// angleDiff 计算两个角度之间的最小差值（考虑圆形）
func angleDiff(a, b int) int {
	a = normalizeAngle(a)
	b = normalizeAngle(b)

	diff := absInt(a - b)
	if diff > 180 {
		diff = 360 - diff
	}
	return diff
}

// normalizeAngle 标准化角度到 0-359 度
func normalizeAngle(angle int) int {
	angle = angle % 360
	if angle < 0 {
		angle += 360
	}
	return angle
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
