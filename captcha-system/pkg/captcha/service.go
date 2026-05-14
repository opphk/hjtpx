package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/opphk/captcha-system/internal/model"
	"github.com/opphk/captcha-system/internal/repository"
)

type CaptchaService struct {
	challengeRepo *repository.ChallengeRepository
	attemptRepo  *repository.AttemptRepository
	sessionRepo  *repository.SessionRepository
	cache       *CacheService
}

func NewCaptchaService(
	challengeRepo *repository.ChallengeRepository,
	attemptRepo *repository.AttemptRepository,
	sessionRepo *repository.SessionRepository,
	cache *CacheService,
) *CaptchaService {
	return &CaptchaService{
		challengeRepo: challengeRepo,
		attemptRepo:  attemptRepo,
		sessionRepo:  sessionRepo,
		cache:       cache,
	}
}

type CreateCaptchaResponse struct {
	ChallengeID string          `json:"challenge_id"`
	Type       string          `json:"type"`
	Data       json.RawMessage `json:"data"`
	ExpiresAt  time.Time       `json:"expires_at"`
}

func (s *CaptchaService) CreateSliderCaptcha(ctx context.Context, sessionID, difficulty string) (*CreateCaptchaResponse, error) {
	generator := NewSliderGenerator()
	sliderData, solution, err := generator.Generate(difficulty)
	if err != nil {
		return nil, fmt.Errorf("failed to generate slider captcha: %w", err)
	}

	challengeID := uuid.New().String()
	expiresAt := time.Now().Add(5 * time.Minute)

	dataJSON, err := json.Marshal(sliderData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	solutionJSON, err := json.Marshal(solution)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal solution: %w", err)
	}

	challenge := &model.Challenge{
		ChallengeID: challengeID,
		Type:        model.CaptchaTypeSlider,
		Difficulty:  model.Difficulty(difficulty),
		Data:        dataJSON,
		Solution:    solutionJSON,
		ExpiresAt:   expiresAt,
	}

	if err := s.challengeRepo.Create(ctx, challenge); err != nil {
		return nil, fmt.Errorf("failed to save challenge: %w", err)
	}

	cacheData := &CaptchaCacheData{
		ChallengeID: challengeID,
		Type:        "slider",
		Data:        dataJSON,
		Solution:    solutionJSON,
		Difficulty:  difficulty,
		CreatedAt:   time.Now(),
	}

	if err := s.cache.SetChallenge(ctx, challengeID, cacheData, 5*time.Minute); err != nil {
		fmt.Printf("Failed to cache challenge: %v\n", err)
	}

	return &CreateCaptchaResponse{
		ChallengeID: challengeID,
		Type:       "slider",
		Data:       dataJSON,
		ExpiresAt:  expiresAt,
	}, nil
}

type VerifyCaptchaRequest struct {
	ChallengeID string          `json:"challenge_id"`
	SessionID  string          `json:"session_id"`
	UserAnswer json.RawMessage `json:"user_answer"`
	IPAddress  string          `json:"ip_address"`
	UserAgent  string          `json:"user_agent"`
	Fingerprint string         `json:"fingerprint"`
}

type VerifyCaptchaResponse struct {
	Success    bool    `json:"success"`
	Score      float64 `json:"score"`
	Message    string  `json:"message"`
	RetryCount int     `json:"retry_count"`
}

func (s *CaptchaService) VerifySliderCaptcha(ctx context.Context, req *VerifyCaptchaRequest) (*VerifyCaptchaResponse, error) {
	startTime := time.Now()

	challenge, err := s.challengeRepo.GetByChallengeID(ctx, req.ChallengeID)
	if err != nil || challenge == nil {
		cacheData, _ := s.cache.GetChallenge(ctx, req.ChallengeID)
		if cacheData == nil {
			return &VerifyCaptchaResponse{
				Success: false,
				Score:   0,
				Message: "Challenge not found or expired",
			}, nil
		}
		challenge = &model.Challenge{
			ChallengeID: cacheData.ChallengeID,
			Data:        cacheData.Data,
			Solution:    cacheData.Solution,
			ExpiresAt:   time.Now().Add(-time.Minute),
		}
	}

	if time.Now().After(challenge.ExpiresAt) {
		return &VerifyCaptchaResponse{
			Success: false,
			Score:   0,
			Message: "Challenge expired",
		}, nil
	}

	var solution SliderSolution
	if err := json.Unmarshal(challenge.Solution, &solution); err != nil {
		return nil, fmt.Errorf("failed to unmarshal solution: %w", err)
	}

	verifier := NewSliderVerifier()
	isValid, riskScore, err := verifier.Verify(&solution, req.UserAnswer)
	if err != nil {
		return nil, fmt.Errorf("failed to verify captcha: %w", err)
	}

	responseTime := int(time.Since(startTime).Milliseconds())

	attempt := &model.Attempt{
		ChallengeID:    req.ChallengeID,
		SessionID:     req.SessionID,
		UserAnswer:    req.UserAnswer,
		IsValid:       isValid,
		ResponseTimeMs: responseTime,
		IPAddress:     req.IPAddress,
		UserAgent:     req.UserAgent,
		Fingerprint:   req.Fingerprint,
		RiskScore:     riskScore,
	}

	if err := s.attemptRepo.Create(ctx, attempt); err != nil {
		fmt.Printf("Failed to save attempt: %v\n", err)
	}

	if isValid {
		s.cache.DeleteChallenge(ctx, req.ChallengeID)
	}

	message := "Verification passed"
	if !isValid {
		message = "Verification failed, please try again"
	}

	return &VerifyCaptchaResponse{
		Success: isValid,
		Score:   riskScore,
		Message: message,
	}, nil
}

func (s *CaptchaService) CreateSession(ctx context.Context, fingerprint, ipAddress string) (string, error) {
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(30 * time.Minute)

	session := &model.Session{
		SessionID:    sessionID,
		Fingerprint: fingerprint,
		IPAddress:   ipAddress,
		AttemptCount: 0,
		ExpiresAt:   expiresAt,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	sessionData := &SessionCacheData{
		SessionID:   sessionID,
		Fingerprint: fingerprint,
		IPAddress:   ipAddress,
		Attempts:   0,
		Blocked:    false,
		CreatedAt:  time.Now(),
	}

	if err := s.cache.SetSession(ctx, sessionID, sessionData, 30*time.Minute); err != nil {
		fmt.Printf("Failed to cache session: %v\n", err)
	}

	return sessionID, nil
}

func (s *CaptchaService) CreateClickCaptcha(ctx context.Context, sessionID, difficulty string) (*CreateCaptchaResponse, error) {
	generator := NewClickGenerator()
	clickData, solution, err := generator.Generate(difficulty)
	if err != nil {
		return nil, fmt.Errorf("failed to generate click captcha: %w", err)
	}

	challengeID := uuid.New().String()
	expiresAt := time.Now().Add(5 * time.Minute)

	dataJSON, err := json.Marshal(clickData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	solutionJSON, err := json.Marshal(solution)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal solution: %w", err)
	}

	challenge := &model.Challenge{
		ChallengeID: challengeID,
		Type:        model.CaptchaTypeClick,
		Difficulty:  model.Difficulty(difficulty),
		Data:        dataJSON,
		Solution:    solutionJSON,
		ExpiresAt:   expiresAt,
	}

	if err := s.challengeRepo.Create(ctx, challenge); err != nil {
		return nil, fmt.Errorf("failed to save challenge: %w", err)
	}

	cacheData := &CaptchaCacheData{
		ChallengeID: challengeID,
		Type:        "click",
		Data:        dataJSON,
		Solution:    solutionJSON,
		Difficulty:  difficulty,
		CreatedAt:   time.Now(),
	}

	if err := s.cache.SetChallenge(ctx, challengeID, cacheData, 5*time.Minute); err != nil {
		fmt.Printf("Failed to cache challenge: %v\n", err)
	}

	return &CreateCaptchaResponse{
		ChallengeID: challengeID,
		Type:        "click",
		Data:        dataJSON,
		ExpiresAt:   expiresAt,
	}, nil
}

func (s *CaptchaService) VerifyClickCaptcha(ctx context.Context, req *VerifyCaptchaRequest) (*VerifyCaptchaResponse, error) {
	startTime := time.Now()

	challenge, err := s.challengeRepo.GetByChallengeID(ctx, req.ChallengeID)
	if err != nil || challenge == nil {
		cacheData, _ := s.cache.GetChallenge(ctx, req.ChallengeID)
		if cacheData == nil {
			return &VerifyCaptchaResponse{
				Success: false,
				Score:   0,
				Message: "Challenge not found or expired",
			}, nil
		}
		challenge = &model.Challenge{
			ChallengeID: cacheData.ChallengeID,
			Data:        cacheData.Data,
			Solution:    cacheData.Solution,
			ExpiresAt:   time.Now().Add(-time.Minute),
		}
	}

	if time.Now().After(challenge.ExpiresAt) {
		return &VerifyCaptchaResponse{
			Success: false,
			Score:   0,
			Message: "Challenge expired",
		}, nil
	}

	var solution ClickSolution
	if err := json.Unmarshal(challenge.Solution, &solution); err != nil {
		return nil, fmt.Errorf("failed to unmarshal solution: %w", err)
	}

	verifier := NewClickVerifier()
	isValid, riskScore, err := verifier.Verify(&solution, req.UserAnswer)
	if err != nil {
		return nil, fmt.Errorf("failed to verify captcha: %w", err)
	}

	responseTime := int(time.Since(startTime).Milliseconds())

	attempt := &model.Attempt{
		ChallengeID:     req.ChallengeID,
		SessionID:      req.SessionID,
		UserAnswer:     req.UserAnswer,
		IsValid:        isValid,
		ResponseTimeMs: responseTime,
		IPAddress:      req.IPAddress,
		UserAgent:      req.UserAgent,
		Fingerprint:    req.Fingerprint,
		RiskScore:      riskScore,
	}

	if err := s.attemptRepo.Create(ctx, attempt); err != nil {
		fmt.Printf("Failed to save attempt: %v\n", err)
	}

	if isValid {
		s.cache.DeleteChallenge(ctx, req.ChallengeID)
	}

	message := "Verification passed"
	if !isValid {
		message = "Verification failed, please try again"
	}

	return &VerifyCaptchaResponse{
		Success: isValid,
		Score:   riskScore,
		Message: message,
	}, nil
}

func (s *CaptchaService) CreateRotateCaptcha(ctx context.Context, sessionID, difficulty string) (*CreateCaptchaResponse, error) {
	generator := NewRotateGenerator()
	rotateData, solution, err := generator.Generate(difficulty)
	if err != nil {
		return nil, fmt.Errorf("failed to generate rotate captcha: %w", err)
	}

	challengeID := uuid.New().String()
	expiresAt := time.Now().Add(5 * time.Minute)

	dataJSON, err := json.Marshal(rotateData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	solutionJSON, err := json.Marshal(solution)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal solution: %w", err)
	}

	challenge := &model.Challenge{
		ChallengeID: challengeID,
		Type:        model.CaptchaTypeRotate,
		Difficulty:  model.Difficulty(difficulty),
		Data:        dataJSON,
		Solution:    solutionJSON,
		ExpiresAt:   expiresAt,
	}

	if err := s.challengeRepo.Create(ctx, challenge); err != nil {
		return nil, fmt.Errorf("failed to save challenge: %w", err)
	}

	cacheData := &CaptchaCacheData{
		ChallengeID: challengeID,
		Type:        "rotate",
		Data:        dataJSON,
		Solution:    solutionJSON,
		Difficulty:  difficulty,
		CreatedAt:   time.Now(),
	}

	if err := s.cache.SetChallenge(ctx, challengeID, cacheData, 5*time.Minute); err != nil {
		fmt.Printf("Failed to cache challenge: %v\n", err)
	}

	return &CreateCaptchaResponse{
		ChallengeID: challengeID,
		Type:        "rotate",
		Data:        dataJSON,
		ExpiresAt:   expiresAt,
	}, nil
}

func (s *CaptchaService) VerifyRotateCaptcha(ctx context.Context, req *VerifyCaptchaRequest) (*VerifyCaptchaResponse, error) {
	startTime := time.Now()

	challenge, err := s.challengeRepo.GetByChallengeID(ctx, req.ChallengeID)
	if err != nil || challenge == nil {
		cacheData, _ := s.cache.GetChallenge(ctx, req.ChallengeID)
		if cacheData == nil {
			return &VerifyCaptchaResponse{
				Success: false,
				Score:   0,
				Message: "Challenge not found or expired",
			}, nil
		}
		challenge = &model.Challenge{
			ChallengeID: cacheData.ChallengeID,
			Data:        cacheData.Data,
			Solution:    cacheData.Solution,
			ExpiresAt:   time.Now().Add(-time.Minute),
		}
	}

	if time.Now().After(challenge.ExpiresAt) {
		return &VerifyCaptchaResponse{
			Success: false,
			Score:   0,
			Message: "Challenge expired",
		}, nil
	}

	var solution RotateSolution
	if err := json.Unmarshal(challenge.Solution, &solution); err != nil {
		return nil, fmt.Errorf("failed to unmarshal solution: %w", err)
	}

	verifier := NewRotateVerifier()
	isValid, riskScore, err := verifier.Verify(&solution, req.UserAnswer)
	if err != nil {
		return nil, fmt.Errorf("failed to verify captcha: %w", err)
	}

	responseTime := int(time.Since(startTime).Milliseconds())

	attempt := &model.Attempt{
		ChallengeID:    req.ChallengeID,
		SessionID:      req.SessionID,
		UserAnswer:     req.UserAnswer,
		IsValid:        isValid,
		ResponseTimeMs: responseTime,
		IPAddress:      req.IPAddress,
		UserAgent:      req.UserAgent,
		Fingerprint:    req.Fingerprint,
		RiskScore:      riskScore,
	}

	if err := s.attemptRepo.Create(ctx, attempt); err != nil {
		fmt.Printf("Failed to save attempt: %v\n", err)
	}

	if isValid {
		s.cache.DeleteChallenge(ctx, req.ChallengeID)
	}

	message := "Verification passed"
	if !isValid {
		message = "Verification failed, please try again"
	}

	return &VerifyCaptchaResponse{
		Success: isValid,
		Score:   riskScore,
		Message: message,
	}, nil
}
