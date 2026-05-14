package puzzle

import (
	"captchax/config"
	"context"
	"errors"
	"fmt"
	"math"
)

var (
	ErrInvalidX     = errors.New("invalid x position")
	ErrInvalidY     = errors.New("invalid y position")
	ErrInvalidID    = errors.New("invalid captcha id")
	ErrVerification = errors.New("verification failed")
	ErrMaxAttempts  = errors.New("max verification attempts exceeded")
)

type VerifyRequest struct {
	CaptchaID string `json:"captcha_id"`
	TargetX   int    `json:"target_x"`
	TargetY   int    `json:"target_y"`
}

type VerifyResult struct {
	Success        bool    `json:"success"`
	Message        string  `json:"message"`
	Distance       float64 `json:"distance"`
	RemainingHints int     `json:"remaining_hints,omitempty"`
}

type VerifyService struct {
	cache *CacheManager
	cfg   *config.CaptchaConfig
}

type Hints struct {
	LeftHint    bool `json:"left_hint"`
	RightHint   bool `json:"right_hint"`
	TopHint     bool `json:"top_hint"`
	BottomHint  bool `json:"bottom_hint"`
	CenterHint  bool `json:"center_hint"`
}

type VerifyStats struct {
	TotalVerified   int64   `json:"total_verified"`
	TotalFailed    int64   `json:"total_failed"`
	SuccessRate    float64 `json:"success_rate"`
	AvgAttempts    float64 `json:"avg_attempts"`
	AvgDistance    float64 `json:"avg_distance"`
}

func NewVerifyService(cfg *config.CaptchaConfig, cache *CacheManager) *VerifyService {
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

	if err := vs.ValidatePosition(req.TargetX, req.TargetY); err != nil {
		return &VerifyResult{
			Success: false,
			Message: err.Error(),
		}, err
	}

	captchaData, err := vs.cache.Get(ctx, req.CaptchaID)
	if err != nil {
		if errors.Is(err, ErrPuzzleNotFound) {
			return &VerifyResult{
				Success: false,
				Message: "puzzle captcha not found or expired",
			}, ErrPuzzleNotFound
		}
		if errors.Is(err, ErrPuzzleExpired) {
			return &VerifyResult{
				Success: false,
				Message: "puzzle captcha has expired",
			}, ErrPuzzleExpired
		}
		return nil, fmt.Errorf("failed to get captcha data: %w", err)
	}

	if captchaData.Verified {
		return &VerifyResult{
			Success: false,
			Message: "puzzle captcha already verified",
		}, ErrPuzzleVerified
	}

	attempts, err := vs.cache.IncrementAttempts(ctx, req.CaptchaID)
	if err != nil {
		if errors.Is(err, fmt.Errorf("max attempts exceeded")) {
			return &VerifyResult{
				Success: false,
				Message: "maximum verification attempts exceeded",
			}, ErrMaxAttempts
		}
		return nil, fmt.Errorf("failed to increment attempts: %w", err)
	}

	if attempts > vs.cfg.MaxAttempts && vs.cfg.MaxAttempts > 0 {
		_ = vs.cache.Delete(ctx, req.CaptchaID)
		return &VerifyResult{
			Success: false,
			Message: "maximum verification attempts exceeded",
		}, ErrMaxAttempts
	}

	tolerance := vs.cfg.Tolerance
	if tolerance == 0 {
		tolerance = 10
	}

	distance := vs.calculateDistance(req.TargetX, req.TargetY, captchaData.TargetX, captchaData.TargetY)

	xDiff := absInt(req.TargetX - captchaData.TargetX)
	yDiff := absInt(req.TargetY - captchaData.TargetY)

	if xDiff <= tolerance && yDiff <= tolerance {
		if err := vs.cache.MarkVerified(ctx, req.CaptchaID); err != nil {
			return nil, fmt.Errorf("failed to mark captcha as verified: %w", err)
		}

		return &VerifyResult{
			Success:  true,
			Message:  "verification successful",
			Distance: distance,
		}, nil
	}

	remaining, _ := vs.cache.RemainingAttempts(ctx, req.CaptchaID)
	return &VerifyResult{
		Success:        false,
		Message:        fmt.Sprintf("verification failed: piece position incorrect (distance: %.1f, tolerance: %d)", distance, tolerance),
		Distance:       distance,
		RemainingHints: remaining,
	}, ErrVerification
}

func (vs *VerifyService) calculateDistance(x1, y1, x2, y2 int) float64 {
	dx := float64(x1 - x2)
	dy := float64(y1 - y2)
	return math.Sqrt(dx*dx + dy*dy)
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

func (vs *VerifyService) GetHints(ctx context.Context, captchaID string) (*Hints, error) {
	data, err := vs.cache.Get(ctx, captchaID)
	if err != nil {
		return nil, err
	}

	centerX := data.TargetX
	centerY := data.TargetY
	width := vs.cfg.Width
	height := vs.cfg.Height

	thirdW := width / 3
	thirdH := height / 3

	hints := &Hints{}

	if centerX < thirdW {
		hints.RightHint = true
	} else if centerX > 2*thirdW {
		hints.LeftHint = true
	} else {
		hints.CenterHint = true
	}

	if centerY < thirdH {
		hints.BottomHint = true
	} else if centerY > 2*thirdH {
		hints.TopHint = true
	}

	return hints, nil
}

func (vs *VerifyService) CheckProximity(ctx context.Context, captchaID string, x, y int) (bool, float64, error) {
	data, err := vs.cache.Get(ctx, captchaID)
	if err != nil {
		return false, 0, err
	}

	distance := vs.calculateDistance(x, y, data.TargetX, data.TargetY)
	tolerance := float64(vs.cfg.Tolerance)
	if tolerance == 0 {
		tolerance = 10
	}

	return distance <= tolerance, distance, nil
}

func (vs *VerifyService) GetDirectionHint(ctx context.Context, captchaID string, x, y int) (string, error) {
	data, err := vs.cache.Get(ctx, captchaID)
	if err != nil {
		return "", err
	}

	dx := data.TargetX - x
	dy := data.TargetY - y

	direction := ""

	if math.Abs(float64(dx)) > math.Abs(float64(dy)) {
		if dx > 0 {
			direction = "right"
		} else {
			direction = "left"
		}
	} else {
		if dy > 0 {
			direction = "down"
		} else {
			direction = "up"
		}
	}

	return direction, nil
}
