package captcha

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type GeneratorService struct {
	imageGenerator *ImageGenerator
	sessionCache   *cache.SessionCache
	captchaRepo    *db.CaptchaRepository
}

type CreateCaptchaRequest struct {
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	SliderWidth  int    `json:"slider_width"`
	SliderHeight int    `json:"slider_height"`
	ClientIP     string `json:"client_ip"`
	UserAgent    string `json:"user_agent"`
	Fingerprint  string `json:"fingerprint"`
}

type CreateCaptchaResponse struct {
	SessionID     string `json:"session_id"`
	BackgroundURL string `json:"background_url"`
	SliderURL     string `json:"slider_url"`
	GapX          int    `json:"gap_x"`
	GapY          int    `json:"gap_y"`
	ExpiresIn     int64  `json:"expires_in"`
	ExpiresAt     int64  `json:"expires_at"`
}

func NewGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *GeneratorService {
	return &GeneratorService{
		imageGenerator: NewImageGenerator(),
		sessionCache:   sessionCache,
		captchaRepo:    captchaRepo,
	}
}

func (s *GeneratorService) Create(ctx context.Context, req *CreateCaptchaRequest) (*CreateCaptchaResponse, error) {
	if req.Width > 0 && req.Height > 0 {
		s.imageGenerator.SetDimensions(req.Width, req.Height, req.SliderWidth, req.SliderHeight)
	}

	result, err := s.imageGenerator.GenerateSliderCaptcha()
	if err != nil {
		return nil, fmt.Errorf("failed to generate captcha images: %w", err)
	}

	sessionID := generateSessionID()

	expiresAt := time.Now().Add(5 * time.Minute)

	session := &models.CaptchaSession{
		SessionID:   sessionID,
		Status:      "pending",
		VerifyCount: 0,
		MaxAttempts: 3,
		RiskScore:   0,
		TraceScore:  0,
		EnvScore:    0,
		CreatedAt:   time.Now(),
		ExpiredAt:   expiresAt,
		ClientIP:    req.ClientIP,
		UserAgent:   req.UserAgent,
		Fingerprint: req.Fingerprint,
		GapX:        result.GapX,
		GapY:        result.GapY,
	}

	if s.sessionCache != nil {
		if err := s.sessionCache.Set(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to cache session: %w", err)
		}
	}

	if s.captchaRepo != nil {
		if err := s.captchaRepo.Create(session); err != nil {
			return nil, fmt.Errorf("failed to save session to database: %w", err)
		}
	}

	backgroundURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(result.Background)
	sliderURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(result.Slider)

	return &CreateCaptchaResponse{
		SessionID:     sessionID,
		BackgroundURL: backgroundURL,
		SliderURL:     sliderURL,
		GapX:          result.GapX,
		GapY:          result.GapY,
		ExpiresIn:     int64(5 * time.Minute / time.Second),
		ExpiresAt:     expiresAt.Unix(),
	}, nil
}

func (s *GeneratorService) GetSession(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	if s.sessionCache != nil {
		session, err := s.sessionCache.Get(ctx, sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	if s.captchaRepo != nil {
		session, err := s.captchaRepo.GetBySessionID(sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (s *GeneratorService) DeleteSession(ctx context.Context, sessionID string) error {
	if s.sessionCache != nil {
		if err := s.sessionCache.Delete(ctx, sessionID); err != nil {
			return err
		}
	}

	if s.captchaRepo != nil {
		if err := s.captchaRepo.Delete(sessionID); err != nil {
			return err
		}
	}

	return nil
}

func (s *GeneratorService) CleanupExpired(ctx context.Context) (int64, error) {
	var totalDeleted int64

	if s.captchaRepo != nil {
		deleted, err := s.captchaRepo.CleanupExpired(5 * time.Minute)
		if err != nil {
			return 0, err
		}
		totalDeleted += deleted
	}

	return totalDeleted, nil
}

func generateSessionID() string {
	return fmt.Sprintf("captcha_%d_%d", time.Now().UnixNano(), time.Now().UnixMicro()%10000)
}
