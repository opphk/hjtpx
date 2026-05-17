package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"hjtpx/internal/captcha"
	"hjtpx/internal/models"
	"hjtpx/internal/repository"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

var (
	ErrCaptchaNotFound       = errors.New("captcha not found")
	ErrCaptchaExpired       = errors.New("captcha has expired")
	ErrCaptchaAlreadyUsed   = errors.New("captcha already verified")
	ErrMaxAttemptsExceeded  = errors.New("maximum verification attempts exceeded")
	ErrInvalidChallenge     = errors.New("invalid challenge")
	ErrInvalidTrackData     = errors.New("invalid track data")
	ErrVerificationFailed   = errors.New("verification failed")
)

const (
	CaptchaExpiration   = 5 * time.Minute
	MaxVerifyAttempts   = 3
	RedisCaptchaPrefix  = "captcha:challenge:"
	RedisAttemptsPrefix = "captcha:attempts:"
)

type CaptchaService struct {
	repo          *repository.CaptchaRepository
	redisClient   *redis.Client
	factory       *captcha.GeneratorFactory
	verifier      *captcha.TrajectoryVerifier
}

type CaptchaServiceConfig struct {
	RedisClient *redis.Client
}

type CreateSliderRequest struct {
	UserID    uint
	AppID     uint
	IPAddress string
	UserAgent string
	Metadata  string
}

type VerifySliderRequest struct {
	Token     string
	Position  float64
	TrackData string
}

type SliderChallenge struct {
	Token         string    `json:"token"`
	BackgroundImg string    `json:"background_image"`
	SliderImg     string    `json:"slider_image"`
	ExpiresAt     int64     `json:"expires_at"`
	X             int       `json:"x"`
	Y             int       `json:"y"`
	Attempts      int       `json:"attempts"`
	MaxAttempts   int       `json:"max_attempts"`
}

type VerifyResult struct {
	Success       bool                       `json:"success"`
	Message       string                     `json:"message"`
	Attempts      int                        `json:"attempts_remaining"`
	RiskLevel     captcha.RiskLevel          `json:"risk_level,omitempty"`
	Score         float64                    `json:"score,omitempty"`
	Verification  *captcha.VerificationResult `json:"verification,omitempty"`
}

func NewCaptchaService(repo *repository.CaptchaRepository, redisClient *redis.Client) *CaptchaService {
	return &CaptchaService{
		repo:        repo,
		redisClient: redisClient,
		factory:     captcha.NewGeneratorFactory(),
		verifier:    captcha.NewTrajectoryVerifier(nil),
	}
}

func (s *CaptchaService) CreateSliderChallenge(ctx context.Context, req *CreateSliderRequest) (*SliderChallenge, error) {
	generator, err := s.factory.GetGenerator(captcha.GeneratorTypeSlider)
	if err != nil {
		return nil, fmt.Errorf("failed to get slider generator: %w", err)
	}

	result, err := generator.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate slider: %w", err)
	}

	challenge, ok := result.(*captcha.CaptchaChallenge)
	if !ok {
		return nil, errors.New("invalid challenge type")
	}

	token := s.generateToken()

	if err := s.storeChallengeInRedis(ctx, token, challenge); err != nil {
		return nil, fmt.Errorf("failed to store challenge: %w", err)
	}

	captchaModel := &models.Captcha{
		Token:       token,
		Challenge:   challenge.Answer,
		Type:        models.CaptchaTypeImage,
		Status:      models.CaptchaStatusPending,
		ExpiresAt:   challenge.ExpiresAt,
		UserID:      req.UserID,
		AppID:       req.AppID,
		IPAddress:   req.IPAddress,
		UserAgent:   req.UserAgent,
		Metadata:    req.Metadata,
		MaxVerify:   MaxVerifyAttempts,
	}

	if err := s.repo.Create(captchaModel); err != nil {
		s.deleteChallengeFromRedis(ctx, token)
		return nil, fmt.Errorf("failed to save captcha: %w", err)
	}

	return &SliderChallenge{
		Token:         token,
		BackgroundImg: challenge.Background,
		SliderImg:     challenge.SliderImage,
		ExpiresAt:     challenge.ExpiresAt.Unix(),
		X:             challenge.X,
		Y:             challenge.Y,
		Attempts:      0,
		MaxAttempts:   MaxVerifyAttempts,
	}, nil
}

func (s *CaptchaService) VerifySliderChallenge(ctx context.Context, req *VerifySliderRequest) (*VerifyResult, error) {
	attemptsKey := RedisAttemptsPrefix + req.Token

	attempts, err := s.redisClient.Get(ctx, attemptsKey).Int()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get attempts: %w", err)
	}

	if attempts >= MaxVerifyAttempts {
		return &VerifyResult{
			Success:  false,
			Message:  "Maximum verification attempts exceeded",
			Attempts: 0,
		}, ErrMaxAttemptsExceeded
	}

	challenge, err := s.getChallengeFromRedis(ctx, req.Token)
	if err != nil {
		captchaModel, err := s.repo.GetByToken(req.Token)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrCaptchaNotFound
			}
			return nil, fmt.Errorf("failed to get captcha: %w", err)
		}

		if captchaModel.IsExpired() {
			return nil, ErrCaptchaExpired
		}

		if captchaModel.IsVerified() {
			return nil, ErrCaptchaAlreadyUsed
		}

		if captchaModel.VerifyCount >= MaxVerifyAttempts {
			return nil, ErrMaxAttemptsExceeded
		}
	}

	var targetX int
	if challenge != nil {
		targetX = challenge.X
	} else {
		captchaModel, err := s.repo.GetByToken(req.Token)
		if err != nil {
			return nil, err
		}
		targetX = 0
		var challengeData map[string]interface{}
		if err := json.Unmarshal([]byte(captchaModel.Challenge), &challengeData); err == nil {
			if x, ok := challengeData["x"].(float64); ok {
				targetX = int(x)
			}
		}
	}

	var trackData *captcha.TrackData
	if req.TrackData != "" {
		trackData, err = s.verifier.ParseTrackData(req.TrackData)
		if err != nil {
			s.incrementAttempts(ctx, attemptsKey)
			remaining := MaxVerifyAttempts - attempts - 1
			return &VerifyResult{
				Success:  false,
				Message:  "Invalid track data format",
				Attempts: remaining,
			}, ErrInvalidTrackData
		}

		if err := s.verifier.ValidateTrackFormat(trackData); err != nil {
			s.incrementAttempts(ctx, attemptsKey)
			remaining := MaxVerifyAttempts - attempts - 1
			return &VerifyResult{
				Success:  false,
				Message:  fmt.Sprintf("Track validation failed: %v", err),
				Attempts: remaining,
			}, ErrInvalidTrackData
		}
	}

	verificationResult := s.verifier.Verify(trackData, targetX)

	if !verificationResult.Valid {
		s.incrementAttempts(ctx, attemptsKey)
		remaining := MaxVerifyAttempts - attempts - 1

		captchaModel, _ := s.repo.GetByToken(req.Token)
		if captchaModel != nil {
			captchaModel.VerifyCount++
			if remaining <= 0 {
				captchaModel.Status = models.CaptchaStatusFailed
			}
			s.repo.Update(captchaModel)
		}

		return &VerifyResult{
			Success:      false,
			Message:      "Verification failed",
			Attempts:     remaining,
			RiskLevel:    verificationResult.RiskLevel,
			Score:        verificationResult.Score,
			Verification: verificationResult,
		}, ErrVerificationFailed
	}

	s.markAsVerified(ctx, req.Token)

	captchaModel, _ := s.repo.GetByToken(req.Token)
	if captchaModel != nil {
		captchaModel.Status = models.CaptchaStatusVerified
		s.repo.Update(captchaModel)
	}

	return &VerifyResult{
		Success:      true,
		Message:      "Verification successful",
		Attempts:     MaxVerifyAttempts - attempts - 1,
		RiskLevel:    verificationResult.RiskLevel,
		Score:        verificationResult.Score,
		Verification: verificationResult,
	}, nil
}

func (s *CaptchaService) GetChallengeStatus(ctx context.Context, token string) (*SliderChallenge, error) {
	captchaModel, err := s.repo.GetByToken(token)
	if err != nil {
		return nil, ErrCaptchaNotFound
	}

	if captchaModel.IsExpired() {
		return nil, ErrCaptchaExpired
	}

	challenge, _ := s.getChallengeFromRedis(ctx, token)

	var x, y int
	if challenge != nil {
		x = challenge.X
		y = challenge.Y
	}

	return &SliderChallenge{
		Token:       token,
		ExpiresAt:   captchaModel.ExpiresAt.Unix(),
		Attempts:    captchaModel.VerifyCount,
		MaxAttempts: captchaModel.MaxVerify,
		X:           x,
		Y:           y,
	}, nil
}

func (s *CaptchaService) storeChallengeInRedis(ctx context.Context, token string, challenge *captcha.CaptchaChallenge) error {
	data, err := json.Marshal(challenge)
	if err != nil {
		return err
	}

	key := RedisCaptchaPrefix + token
	return s.redisClient.Set(ctx, key, data, CaptchaExpiration).Err()
}

func (s *CaptchaService) getChallengeFromRedis(ctx context.Context, token string) (*captcha.CaptchaChallenge, error) {
	key := RedisCaptchaPrefix + token
	data, err := s.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var challenge captcha.CaptchaChallenge
	if err := json.Unmarshal(data, &challenge); err != nil {
		return nil, err
	}

	return &challenge, nil
}

func (s *CaptchaService) deleteChallengeFromRedis(ctx context.Context, token string) error {
	key := RedisCaptchaPrefix + token
	return s.redisClient.Del(ctx, key).Err()
}

func (s *CaptchaService) incrementAttempts(ctx context.Context, key string) error {
	return s.redisClient.Incr(ctx, key).Err()
}

func (s *CaptchaService) markAsVerified(ctx context.Context, token string) error {
	if err := s.deleteChallengeFromRedis(ctx, token); err != nil {
		return err
	}

	attemptsKey := RedisAttemptsPrefix + token
	return s.redisClient.Del(ctx, attemptsKey).Err()
}

func (s *CaptchaService) generateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *CaptchaService) CleanExpiredChallenges(ctx context.Context) (int64, error) {
	pattern := RedisCaptchaPrefix + "*"
	keys, err := s.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return 0, err
	}

	if len(keys) == 0 {
		return 0, nil
	}

	deleted, err := s.redisClient.Del(ctx, keys...).Result()
	if err != nil {
		return 0, err
	}

	return deleted, nil
}

func (s *CaptchaService) GetServiceStats(ctx context.Context) (map[string]interface{}, error) {
	dbStats, err := s.repo.GetStats()
	if err != nil {
		return nil, err
	}

	pattern := RedisCaptchaPrefix + "*"
	redisKeys, err := s.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	pattern = RedisAttemptsPrefix + "*"
	attemptKeys, err := s.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"database":          dbStats,
		"active_challenges": len(redisKeys),
		"pending_attempts":  len(attemptKeys),
	}

	return stats, nil
}
