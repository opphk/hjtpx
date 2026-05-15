package icon

import (
	"captchax/config"
	"context"
	"errors"
	"fmt"
)

var (
	ErrInvalidID        = errors.New("invalid captcha id")
	ErrMissingIconIDs   = errors.New("icon ids are required")
	ErrVerification     = errors.New("verification failed")
)

// VerifyService handles captcha verification
type VerifyService struct {
	cache CacheManagerInterface
	cfg   *config.CaptchaConfig
}

// CacheManagerInterface defines the interface for cache operations
type CacheManagerInterface interface {
	Get(ctx context.Context, id string) (*CacheData, error)
	Delete(ctx context.Context, id string) error
	Set(ctx context.Context, id string, data *CacheData) error
	Exists(ctx context.Context, id string) (bool, error)
}

// NewVerifyService creates a new VerifyService
func NewVerifyService(cfg *config.CaptchaConfig, cache *CacheManager) *VerifyService {
	return &VerifyService{
		cache: cache,
		cfg:   cfg,
	}
}

// NewVerifyServiceWithInterface creates a new VerifyService with a custom cache manager
func NewVerifyServiceWithInterface(cfg *config.CaptchaConfig, cache CacheManagerInterface) *VerifyService {
	return &VerifyService{
		cache: cache,
		cfg:   cfg,
	}
}

// Verify checks if the selected icons match the target icons
func (vs *VerifyService) Verify(ctx context.Context, req *VerifyRequest) (*VerifyResult, error) {
	// Validate request
	if req.CaptchaID == "" {
		return &VerifyResult{
			Success: false,
			Message: "captcha id is required",
		}, ErrInvalidID
	}

	if len(req.IconIDs) == 0 {
		return &VerifyResult{
			Success: false,
			Message: "icon ids are required",
		}, ErrMissingIconIDs
	}

	// Get captcha data from cache
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

	// Check if already verified
	if captchaData.Verified {
		return &VerifyResult{
			Success: false,
			Message: "captcha already verified",
		}, ErrCaptchaVerified
	}

	// Verify selected icons
	if !vs.verifyIcons(req.IconIDs, captchaData.TargetIconIDs) {
		return &VerifyResult{
			Success: false,
			Message: "verification failed",
		}, ErrVerification
	}

	// Delete from cache on successful verification
	if err := vs.cache.Delete(ctx, req.CaptchaID); err != nil {
		return nil, fmt.Errorf("failed to delete captcha after verification: %w", err)
	}

	return &VerifyResult{
		Success: true,
		Message: "verification successful",
	}, nil
}

// verifyIcons checks if selected icon IDs exactly match the target icon IDs
func (vs *VerifyService) verifyIcons(selectedIDs, targetIDs []string) bool {
	// Check if lengths match
	if len(selectedIDs) != len(targetIDs) {
		return false
	}

	// Create maps for O(1) lookups
	selectedMap := make(map[string]bool)
	for _, id := range selectedIDs {
		selectedMap[id] = true
	}

	// Check all targets are present and no extra
	for _, targetID := range targetIDs {
		if !selectedMap[targetID] {
			return false
		}
	}

	// Check no extra icons
	if len(selectedMap) != len(targetIDs) {
		return false
	}

	return true
}
