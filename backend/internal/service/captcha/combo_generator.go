package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	github.com/hjtpx/hjtpx/internal/repository/cache"
	github.com/hjtpx/hjtpx/internal/repository/db"
)

type ComboGeneratorService struct {
	sessionCache       *cache.SessionCache
	captchaRepo        *db.CaptchaRepository
	semanticGenerator  *SemanticGeneratorService
	emojiGenerator     *EmojiGeneratorService
	threeDGenerator   *ThreeDGeneratorService
}

type CreateComboCaptchaRequest struct {
	Types       []string `json:"types"`
	Strategy    string   `json:"strategy"`
	Difficulty  string   `json:"difficulty"`
	ClientIP    string   `json:"client_ip"`
	UserAgent   string   `json:"user_agent"`
	Fingerprint string   `json:"fingerprint"`
}

type CreateComboCaptchaResponse struct {
	SessionID      string                   `json:"session_id"`
	Captchas       []ComboCaptchaItem       `json:"captchas"`
	Strategy      string                   `json:"strategy"`
	TotalRequired int                      `json:"total_required"`
	ExpiresIn     int64                    `json:"expires_in"`
	ExpiresAt     int64                    `json:"expires_at"`
}

type ComboCaptchaItem struct {
	Type       string      `json:"type"`
	SessionID  string      `json:"session_id"`
	Data       interface{} `json:"data"`
	Order      int         `json:"order"`
	Mandatory  bool        `json:"mandatory"`
}

type VerifyComboCaptchaRequest struct {
	SessionID string                   `json:"session_id"`
	Answers   []ComboCaptchaAnswer     `json:"answers"`
}

type ComboCaptchaAnswer struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Answer    interface{} `json:"answer"`
}

type VerifyComboCaptchaResponse struct {
	Success       bool                      `json:"success"`
	Message       string                    `json:"message"`
	Score         float64                   `json:"score"`
	Results       []ComboCaptchaResult      `json:"results"`
	PassedCount   int                      `json:"passed_count"`
	FailedCount   int                      `json:"failed_count"`
}

type ComboCaptchaResult struct {
	Type    string  `json:"type"`
	Success bool    `json:"success"`
	Score   float64 `json:"score"`
	Message string  `json:"message"`
}

type ComboCaptchaSession struct {
	SessionID      string                    `json:"session_id"`
	Captchas       []ComboCaptchaItem        `json:"captchas"`
	Strategy       string                    `json:"strategy"`
	TotalRequired  int                       `json:"total_required"`
	Status         string                    `json:"status"`
	VerifiedCount  int                       `json:"verified_count"`
	VerifyCount    int                       `json:"verify_count"`
	MaxAttempts    int                       `json:"max_attempts"`
	RiskScore      float64                   `json:"risk_score"`
	TraceScore     float64                   `json:"trace_score"`
	EnvScore       float64                   `json:"env_score"`
	CreatedAt      time.Time                 `json:"created_at"`
	ExpiredAt      time.Time                 `json:"expired_at"`
	ClientIP       string                    `json:"client_ip"`
	UserAgent      string                    `json:"user_agent"`
	Fingerprint    string                    `json:"fingerprint"`
}

func NewComboGeneratorService(
	sessionCache *cache.SessionCache,
	captchaRepo *db.CaptchaRepository,
	semanticGenerator *SemanticGeneratorService,
	emojiGenerator *EmojiGeneratorService,
	threeDGenerator *ThreeDGeneratorService,
) *ComboGeneratorService {
	return &ComboGeneratorService{
		sessionCache:      sessionCache,
		captchaRepo:       captchaRepo,
		semanticGenerator: semanticGenerator,
		emojiGenerator:    emojiGenerator,
		threeDGenerator:   threeDGenerator,
	}
}

func NewComboGeneratorServiceSimple() *ComboGeneratorService {
	return &ComboGeneratorService{
		semanticGenerator: NewSemanticGeneratorServiceSimple(),
		emojiGenerator:    NewEmojiGeneratorServiceSimple(),
		threeDGenerator:   NewThreeDGeneratorServiceSimple(),
	}
}

func (s *ComboGeneratorService) Create(ctx context.Context, req *CreateComboCaptchaRequest) (*CreateComboCaptchaResponse, error) {
	sessionID := generateComboSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	types := req.Types
	if len(types) == 0 {
		types = []string{"slider", "emoji"}
	}

	strategy := req.Strategy
	if strategy == "" {
		strategy = "all"
	}

	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = "medium"
	}

	captchaItems := make([]ComboCaptchaItem, len(types))
	for i, captchaType := range types {
		item, err := s.generateCaptchaByType(ctx, captchaType, difficulty, req)
		if err != nil {
			continue
		}
		item.Order = i + 1
		item.Mandatory = (strategy == "all")
		captchaItems[i] = *item
	}

	totalRequired := len(captchaItems)
	if strategy == "any" {
		totalRequired = 1
	} else if strategy == "majority" {
		totalRequired = len(captchaItems)/2 + 1
	}

	session := &ComboCaptchaSession{
		SessionID:     sessionID,
		Captchas:      captchaItems,
		Strategy:      strategy,
		TotalRequired: totalRequired,
		Status:        "pending",
		VerifiedCount: 0,
		VerifyCount:   0,
		MaxAttempts:   3,
		RiskScore:     0,
		TraceScore:    0,
		EnvScore:      0,
		CreatedAt:     time.Now(),
		ExpiredAt:     expiresAt,
		ClientIP:      req.ClientIP,
		UserAgent:     req.UserAgent,
		Fingerprint:   req.Fingerprint,
	}

	if s.sessionCache != nil {
		sessionJSON, err := json.Marshal(session)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal session: %w", err)
		}
		if err := s.sessionCache.SetRaw(ctx, sessionID, string(sessionJSON), 5*time.Minute); err != nil {
			return nil, fmt.Errorf("failed to cache session: %w", err)
		}
	}

	response := &CreateComboCaptchaResponse{
		SessionID:      sessionID,
		Captchas:       captchaItems,
		Strategy:       strategy,
		TotalRequired:  totalRequired,
		ExpiresIn:      int64(5 * time.Minute / time.Second),
		ExpiresAt:      expiresAt.Unix(),
	}

	return response, nil
}

func (s *ComboGeneratorService) generateCaptchaByType(ctx context.Context, captchaType string, difficulty string, req *CreateComboCaptchaRequest) (*ComboCaptchaItem, error) {
	switch captchaType {
	case "semantic":
		semanticReq := &CreateSemanticCaptchaRequest{
			Language:    "en",
			Difficulty:  difficulty,
			ClientIP:    req.ClientIP,
			UserAgent:   req.UserAgent,
			Fingerprint: req.Fingerprint,
		}
		result, err := s.semanticGenerator.Create(ctx, semanticReq)
		if err != nil {
			return nil, err
		}
		return &ComboCaptchaItem{
			Type:      "semantic",
			SessionID: result.SessionID,
			Data:      result,
		}, nil

	case "emoji":
		emojiReq := &CreateEmojiCaptchaRequest{
			ClientIP:    req.ClientIP,
			UserAgent:   req.UserAgent,
			Fingerprint: req.Fingerprint,
		}
		result, err := s.emojiGenerator.Create(ctx, emojiReq)
		if err != nil {
			return nil, err
		}
		return &ComboCaptchaItem{
			Type:      "emoji",
			SessionID: result.SessionID,
			Data:      result,
		}, nil

	case "3d":
		threeDReq := &CreateThreeDRequest{
			Difficulty:  difficulty,
			ClientIP:   req.ClientIP,
			UserAgent:  req.UserAgent,
			Fingerprint: req.Fingerprint,
		}
		result, err := s.threeDGenerator.Create(ctx, threeDReq)
		if err != nil {
			return nil, err
		}
		return &ComboCaptchaItem{
			Type:      "3d",
			SessionID: result.SessionID,
			Data:      result,
		}, nil

	case "slider":
		sliderReq := &CreateCaptchaRequest{
			Width:        320,
			Height:       160,
			SliderWidth:  40,
			SliderHeight: 40,
			ClientIP:     req.ClientIP,
			UserAgent:    req.UserAgent,
			Fingerprint:  req.Fingerprint,
		}
		gen := NewGeneratorService(s.sessionCache, s.captchaRepo)
		result, err := gen.Create(ctx, sliderReq)
		if err != nil {
			return nil, err
		}
		return &ComboCaptchaItem{
			Type:      "slider",
			SessionID: result.SessionID,
			Data:      result,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported captcha type: %s", captchaType)
	}
}

func (s *ComboGeneratorService) Verify(ctx context.Context, req *VerifyComboCaptchaRequest) (*VerifyComboCaptchaResponse, error) {
	session, err := s.GetSession(ctx, req.SessionID)
	if err != nil {
		return &VerifyComboCaptchaResponse{
			Success: false,
			Message: "会话不存在或已过期",
			Score:   0,
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifyComboCaptchaResponse{
			Success: false,
			Message: "会话已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifyComboCaptchaResponse{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	session.VerifyCount++
	s.UpdateSession(ctx, session)

	results := make([]ComboCaptchaResult, 0, len(req.Answers))
	passedCount := 0

	for _, answer := range req.Answers {
		result := s.verifyCaptchaAnswer(ctx, answer, session)
		results = append(results, result)
		if result.Success {
			passedCount++
		}
	}

	success := false
	message := ""

	switch session.Strategy {
	case "all":
		success = passedCount == len(session.Captchas)
		if success {
			message = "全部验证成功"
		} else {
			message = fmt.Sprintf("需要全部验证通过，当前通过 %d/%d", passedCount, len(session.Captchas))
		}
	case "any":
		success = passedCount >= 1
		if success {
			message = "至少一项验证成功"
		} else {
			message = "需要至少一项验证通过"
		}
	case "majority":
		success = passedCount >= session.TotalRequired
		if success {
			message = "多数验证成功"
		} else {
			message = fmt.Sprintf("需要多数验证通过，当前通过 %d/%d", passedCount, session.TotalRequired)
		}
	default:
		success = passedCount >= session.TotalRequired
		message = fmt.Sprintf("验证通过 %d/%d", passedCount, session.TotalRequired)
	}

	score := float64(0)
	if len(results) > 0 {
		totalScore := float64(0)
		for _, r := range results {
			totalScore += r.Score
		}
		score = totalScore / float64(len(results))
	}

	if success {
		session.Status = "verified"
		s.UpdateSession(ctx, session)
	}

	return &VerifyComboCaptchaResponse{
		Success:     success,
		Message:     message,
		Score:       score,
		Results:     results,
		PassedCount: passedCount,
		FailedCount: len(results) - passedCount,
	}, nil
}

func (s *ComboGeneratorService) verifyCaptchaAnswer(ctx context.Context, answer ComboCaptchaAnswer, session *ComboCaptchaSession) ComboCaptchaResult {
	switch answer.Type {
	case "semantic":
		ver := NewSemanticVerifierService(s.semanticGenerator)
		verifyReq := &VerifySemanticCaptchaRequest{
			SessionID: answer.SessionID,
		}

		if strAns, ok := answer.Answer.(string); ok {
			verifyReq.Answer = strAns
		}

		result, err := ver.Verify(ctx, verifyReq)
		if err != nil {
			return ComboCaptchaResult{
				Type:    "semantic",
				Success: false,
				Score:   0,
				Message: "验证失败",
			}
		}

		return ComboCaptchaResult{
			Type:    "semantic",
			Success: result.Success,
			Score:   result.Score,
			Message: result.Message,
		}

	case "emoji":
		ver := NewEmojiVerifierService(s.emojiGenerator)
		verifyReq := &VerifyEmojiCaptchaRequest{
			SessionID: answer.SessionID,
		}

		if arrAns, ok := answer.Answer.([]interface{}); ok {
			emojis := make([]string, len(arrAns))
			for i, v := range arrAns {
				if str, ok := v.(string); ok {
					emojis[i] = str
				}
			}
			verifyReq.SelectedEmojis = emojis
		}

		result, err := ver.Verify(ctx, verifyReq)
		if err != nil {
			return ComboCaptchaResult{
				Type:    "emoji",
				Success: false,
				Score:   0,
				Message: "验证失败",
			}
		}

		return ComboCaptchaResult{
			Type:    "emoji",
			Success: result.Success,
			Score:   100,
			Message: result.Message,
		}

	case "3d":
		return ComboCaptchaResult{
			Type:    "3d",
			Success: false,
			Score:   0,
			Message: "3D验证码暂不支持组合验证",
		}

	case "slider":
		ver := NewVerifierService(s.sessionCache, s.captchaRepo)
		verifyReq := &VerifyRequest{}

		if mapAns, ok := answer.Answer.(map[string]interface{}); ok {
			if x, ok := mapAns["position_x"].(float64); ok {
				verifyReq.PositionX = int(x)
			}
			if y, ok := mapAns["position_y"].(float64); ok {
				verifyReq.PositionY = int(y)
			}
		}
		verifyReq.SessionID = answer.SessionID

		result, err := ver.Verify(ctx, verifyReq)
		if err != nil {
			return ComboCaptchaResult{
				Type:    "slider",
				Success: false,
				Score:   0,
				Message: "验证失败",
			}
		}

		return ComboCaptchaResult{
			Type:    "slider",
			Success: result.Success,
			Score:   result.Score,
			Message: result.Message,
		}

	default:
		return ComboCaptchaResult{
			Type:    answer.Type,
			Success: false,
			Score:   0,
			Message: "未知验证码类型",
		}
	}
}

func (s *ComboGeneratorService) GetSession(ctx context.Context, sessionID string) (*ComboCaptchaSession, error) {
	if s.sessionCache != nil {
		sessionData, err := s.sessionCache.GetRaw(ctx, sessionID)
		if err == nil && sessionData != "" {
			var session ComboCaptchaSession
			if err := json.Unmarshal([]byte(sessionData), &session); err == nil {
				return &session, nil
			}
		}
	}
	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (s *ComboGeneratorService) UpdateSession(ctx context.Context, session *ComboCaptchaSession) error {
	if s.sessionCache != nil {
		sessionJSON, err := json.Marshal(session)
		if err != nil {
			return fmt.Errorf("failed to marshal session: %w", err)
		}
		remainingTime := time.Until(session.ExpiredAt)
		if remainingTime <= 0 {
			return fmt.Errorf("session expired")
		}
		if err := s.sessionCache.SetRaw(ctx, session.SessionID, string(sessionJSON), remainingTime); err != nil {
			return fmt.Errorf("failed to update session cache: %w", err)
		}
	}
	return nil
}

func generateComboSessionID() string {
	return fmt.Sprintf("combo_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}
