package service

import (
	"context"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

// ============================================
// 统一AI服务集成层
// 整合验证码生成、风险评估和行为学习三个模块
// ============================================

type AIServiceIntegration struct {
	gptCaptchaService         *GPTCaptchaService
	riskAssessmentService     *DeepLearningRiskAssessmentService
	behaviorLearningService   *RealTimeBehaviorLearningService
	initialized               bool
}

type AIIntegrationRequest struct {
	Type           string  `json:"type"` // captcha, risk_assessment, behavior_learning, integrated
	UserID         string  `json:"user_id,omitempty"`
	SessionID      string  `json:"session_id"`
	RiskLevel      float64 `json:"risk_level"`
	CaptchaConfig  *CaptchaConfig
	RiskConfig     *RiskAssessmentConfig
	BehaviorConfig *BehaviorLearningConfig
}

type CaptchaConfig struct {
	Difficulty  string `json:"difficulty"`
	ContentType string `json:"content_type"`
}

type RiskAssessmentConfig struct {
	ModelVersion string `json:"model_version"`
	IncludeFeatures bool `json:"include_features"`
}

type BehaviorLearningConfig struct {
	EnableLearning bool `json:"enable_learning"`
	UpdateProfile  bool `json:"update_profile"`
}

type AIIntegrationResponse struct {
	Success          bool                    `json:"success"`
	Type             string                  `json:"type"`
	CaptchaResult    *AICaptchaResponse      `json:"captcha_result,omitempty"`
	RiskResult       *DeepRiskResult         `json:"risk_result,omitempty"`
	LearningResult   *LearningResult         `json:"learning_result,omitempty"`
	IntegratedResult *IntegratedResult       `json:"integrated_result,omitempty"`
	Error           string                  `json:"error,omitempty"`
	ProcessingTime  time.Duration           `json:"processing_time"`
}

type IntegratedResult struct {
	OverallRiskScore     float64 `json:"overall_risk_score"`
	RecommendedAction    string  `json:"recommended_action"` // allow, challenge, block
	CaptchaDifficulty    string  `json:"captcha_difficulty"`
	ShouldChallenge      bool    `json:"should_challenge"`
	UserBehaviorChanged bool    `json:"user_behavior_changed"`
	Confidence          float64 `json:"confidence"`
}

func NewAIServiceIntegration() *AIServiceIntegration {
	return &AIServiceIntegration{
		gptCaptchaService:       NewGPTCaptchaService(),
		riskAssessmentService:   NewDeepLearningRiskAssessmentService(),
		behaviorLearningService: NewRealTimeBehaviorLearningService(),
	}
}

func (s *AIServiceIntegration) Initialize(ctx context.Context) error {
	if s.initialized {
		return nil
	}
	
	if err := s.riskAssessmentService.Initialize(ctx); err != nil {
		return err
	}
	
	if err := s.behaviorLearningService.StartLearning(ctx); err != nil {
		return err
	}
	
	s.initialized = true
	return nil
}

func (s *AIServiceIntegration) ProcessRequest(ctx context.Context, req *AIIntegrationRequest) (*AIIntegrationResponse, error) {
	start := time.Now()
	
	switch req.Type {
	case "captcha":
		return s.handleCaptchaRequest(ctx, req)
	case "risk_assessment":
		return s.handleRiskAssessmentRequest(ctx, req)
	case "behavior_learning":
		return s.handleBehaviorLearningRequest(ctx, req)
	case "integrated":
		return s.handleIntegratedRequest(ctx, req)
	default:
		return &AIIntegrationResponse{
			Success:         false,
			Type:            req.Type,
			Error:           "invalid request type",
			ProcessingTime:  time.Since(start),
		}, nil
	}
}

func (s *AIServiceIntegration) handleCaptchaRequest(ctx context.Context, req *AIIntegrationRequest) (*AIIntegrationResponse, error) {
	if req.CaptchaConfig == nil {
		req.CaptchaConfig = &CaptchaConfig{
			Difficulty:  "medium",
			ContentType: "smart",
		}
	}
	
	captchaReq := &AICaptchaRequest{
		UserID:      req.UserID,
		SessionID:   req.SessionID,
		Difficulty:  req.CaptchaConfig.Difficulty,
		ContentType: req.CaptchaConfig.ContentType,
		RiskLevel:   req.RiskLevel,
	}
	
	result, err := s.gptCaptchaService.GenerateCaptcha(ctx, captchaReq)
	if err != nil {
		return &AIIntegrationResponse{
			Success:         false,
			Type:            "captcha",
			Error:           err.Error(),
			ProcessingTime:  time.Since(time.Now()),
		}, err
	}
	
	return &AIIntegrationResponse{
		Success:         true,
		Type:            "captcha",
		CaptchaResult:   result,
		ProcessingTime:  time.Since(time.Now()),
	}, nil
}

func (s *AIServiceIntegration) handleRiskAssessmentRequest(ctx context.Context, req *AIIntegrationRequest) (*AIIntegrationResponse, error) {
	traceData := &model.TraceData{}
	
	result, err := s.riskAssessmentService.AssessRisk(ctx, traceData, req.UserID)
	if err != nil {
		return &AIIntegrationResponse{
			Success:         false,
			Type:            "risk_assessment",
			Error:           err.Error(),
			ProcessingTime:  time.Since(time.Now()),
		}, err
	}
	
	return &AIIntegrationResponse{
		Success:         true,
		Type:            "risk_assessment",
		RiskResult:      result,
		ProcessingTime:  time.Since(time.Now()),
	}, nil
}

func (s *AIServiceIntegration) handleBehaviorLearningRequest(ctx context.Context, req *AIIntegrationRequest) (*AIIntegrationResponse, error) {
	if req.BehaviorConfig != nil && !req.BehaviorConfig.EnableLearning {
		return &AIIntegrationResponse{
			Success:         true,
			Type:            "behavior_learning",
			LearningResult: &LearningResult{
				UserID:         req.UserID,
				LearningStatus: "learning_disabled",
				UpdatedAt:      time.Now(),
			},
			ProcessingTime: time.Since(time.Now()),
		}, nil
	}
	
	traceData := &model.TraceData{}
	
	result, err := s.behaviorLearningService.LearnFromBehavior(ctx, req.UserID, traceData)
	if err != nil {
		return &AIIntegrationResponse{
			Success:         false,
			Type:            "behavior_learning",
			Error:           err.Error(),
			ProcessingTime:  time.Since(time.Now()),
		}, err
	}
	
	return &AIIntegrationResponse{
		Success:         true,
		Type:            "behavior_learning",
		LearningResult:  result,
		ProcessingTime:  time.Since(time.Now()),
	}, nil
}

func (s *AIServiceIntegration) handleIntegratedRequest(ctx context.Context, req *AIIntegrationRequest) (*AIIntegrationResponse, error) {
	start := time.Now()
	
	traceData := &model.TraceData{}
	
	riskResult, err := s.riskAssessmentService.AssessRisk(ctx, traceData, req.UserID)
	if err != nil {
		return &AIIntegrationResponse{
			Success:         false,
			Type:            "integrated",
			Error:           err.Error(),
			ProcessingTime:  time.Since(start),
		}, err
	}
	
	var learningResult *LearningResult
	if req.BehaviorConfig == nil || req.BehaviorConfig.EnableLearning {
		learningResult, err = s.behaviorLearningService.LearnFromBehavior(ctx, req.UserID, traceData)
		if err != nil {
			return &AIIntegrationResponse{
				Success:         false,
				Type:            "integrated",
				Error:           err.Error(),
				ProcessingTime:  time.Since(start),
			}, err
		}
	}
	
	integratedResult := s.generateIntegratedResult(riskResult, learningResult, req.RiskLevel)
	
	var captchaResult *AICaptchaResponse
	if integratedResult.ShouldChallenge {
		captchaReq := &AICaptchaRequest{
			UserID:      req.UserID,
			SessionID:   req.SessionID,
			Difficulty:  integratedResult.CaptchaDifficulty,
			ContentType: "smart",
			RiskLevel:   req.RiskLevel,
		}
		captchaResult, err = s.gptCaptchaService.GenerateCaptcha(ctx, captchaReq)
		if err != nil {
			return &AIIntegrationResponse{
				Success:         false,
				Type:            "integrated",
				Error:           err.Error(),
				ProcessingTime:  time.Since(start),
			}, err
		}
	}
	
	return &AIIntegrationResponse{
		Success:          true,
		Type:             "integrated",
		RiskResult:       riskResult,
		LearningResult:   learningResult,
		CaptchaResult:    captchaResult,
		IntegratedResult: integratedResult,
		ProcessingTime:   time.Since(start),
	}, nil
}

func (s *AIServiceIntegration) generateIntegratedResult(riskResult *DeepRiskResult, learningResult *LearningResult, riskLevel float64) *IntegratedResult {
	overallScore := riskResult.RiskScore
	
	if learningResult != nil && learningResult.DriftDetected {
		overallScore = (overallScore*0.7 + learningResult.DriftSeverity*0.3)
	}
	
	var recommendedAction, difficulty string
	shouldChallenge := false
	behaviorChanged := learningResult != nil && learningResult.DriftDetected
	
	switch {
	case overallScore >= 0.9:
		recommendedAction = "block"
		difficulty = "expert"
		shouldChallenge = true
	case overallScore >= 0.7:
		recommendedAction = "challenge"
		difficulty = "hard"
		shouldChallenge = true
	case overallScore >= 0.5:
		recommendedAction = "challenge"
		difficulty = "medium"
		shouldChallenge = true
	case overallScore >= 0.3:
		recommendedAction = "challenge"
		difficulty = "easy"
		shouldChallenge = behaviorChanged
	default:
		recommendedAction = "allow"
		difficulty = "easy"
		shouldChallenge = false
	}
	
	if riskLevel >= 0.8 {
		shouldChallenge = true
		if difficulty == "easy" {
			difficulty = "medium"
		}
	}
	
	return &IntegratedResult{
		OverallRiskScore:     overallScore,
		RecommendedAction:    recommendedAction,
		CaptchaDifficulty:    difficulty,
		ShouldChallenge:      shouldChallenge,
		UserBehaviorChanged:  behaviorChanged,
		Confidence:          riskResult.Confidence,
	}
}

func (s *AIServiceIntegration) GetUserProfile(ctx context.Context, userID string) (*UserBehaviorSignature, error) {
	return s.behaviorLearningService.GetUserSignature(ctx, userID)
}

func (s *AIServiceIntegration) UpdateModelFeedback(ctx context.Context, feedback *RiskFeedback) error {
	return s.riskAssessmentService.UpdateModel(ctx, feedback)
}

func (s *AIServiceIntegration) IsInitialized() bool {
	return s.initialized
}

func (s *AIServiceIntegration) Shutdown() error {
	s.behaviorLearningService.StopLearning()
	return nil
}

// ============================================
// AI服务API端点包装
// ============================================

func (s *AIServiceIntegration) CreateCaptcha(ctx context.Context, sessionID, userID, difficulty, contentType string, riskLevel float64) (*AICaptchaResponse, error) {
	req := &AIIntegrationRequest{
		Type:      "captcha",
		UserID:    userID,
		SessionID: sessionID,
		RiskLevel: riskLevel,
		CaptchaConfig: &CaptchaConfig{
			Difficulty:  difficulty,
			ContentType: contentType,
		},
	}
	
	resp, err := s.ProcessRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(resp.Error)
	}
	return resp.CaptchaResult, nil
}

func (s *AIServiceIntegration) AssessBehaviorRisk(ctx context.Context, userID string, traceData *model.TraceData) (*DeepRiskResult, error) {
	result, err := s.riskAssessmentService.AssessRisk(ctx, traceData, userID)
	if err != nil {
		return nil, err
	}
	
	go func() {
		s.behaviorLearningService.LearnFromBehavior(ctx, userID, traceData)
	}()
	
	return result, nil
}

func (s *AIServiceIntegration) GetIntegratedDecision(ctx context.Context, userID, sessionID string, traceData *model.TraceData, riskLevel float64) (*IntegratedResult, *AICaptchaResponse, error) {
	req := &AIIntegrationRequest{
		Type:      "integrated",
		UserID:    userID,
		SessionID: sessionID,
		RiskLevel: riskLevel,
		BehaviorConfig: &BehaviorLearningConfig{
			EnableLearning: true,
			UpdateProfile:  true,
		},
	}
	
	resp, err := s.ProcessRequest(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	if !resp.Success {
		return nil, nil, fmt.Errorf(resp.Error)
	}
	
	return resp.IntegratedResult, resp.CaptchaResult, nil
}

func (s *AIServiceIntegration) GetServiceStatus(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"initialized":     s.IsInitialized(),
		"learning_active": s.behaviorLearningService.IsLearningEnabled(),
		"model_version":   "v2.0",
		"modules": []string{
			"gpt_captcha",
			"deep_learning_risk_assessment",
			"real_time_behavior_learning",
		},
	}
}