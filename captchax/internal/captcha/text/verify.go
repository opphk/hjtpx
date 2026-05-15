package text

import (
	"captchax/config"
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidCode = errors.New("invalid captcha code")
	ErrInvalidID   = errors.New("invalid captcha id")
)

// CacheManagerInterface 缓存管理器接口
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

// NewVerifyServiceWithInterface 使用接口创建验证服务
func NewVerifyServiceWithInterface(cfg *config.CaptchaConfig, cache CacheManagerInterface) *VerifyService {
	return &VerifyService{
		cache: cache,
		cfg:   cfg,
	}
}

// Verify 验证验证码
func (vs *VerifyService) Verify(ctx context.Context, req *VerifyRequest) (*VerifyResult, error) {
	// 检查验证码ID
	if req.CaptchaID == "" {
		return &VerifyResult{
			Success: false,
			Message: "captcha id is required",
		}, ErrInvalidID
	}

	// 检查验证码
	if req.Code == "" {
		return &VerifyResult{
			Success: false,
			Message: "captcha code is required",
		}, ErrInvalidCode
	}

	// 从缓存获取验证码数据
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

	// 检查是否已验证
	if captchaData.Verified {
		return &VerifyResult{
			Success: false,
			Message: "captcha already verified",
		}, ErrCaptchaVerified
	}

	// 验证验证码（忽略大小写）
	if !strings.EqualFold(strings.TrimSpace(req.Code), captchaData.Code) {
		return &VerifyResult{
			Success: false,
			Message: "captcha code is incorrect",
		}, ErrInvalidCode
	}

	// 验证成功，删除缓存
	if err := vs.cache.Delete(ctx, req.CaptchaID); err != nil {
		return nil, fmt.Errorf("failed to delete captcha after verification: %w", err)
	}

	return &VerifyResult{
		Success: true,
		Message: "verification successful",
	}, nil
}
