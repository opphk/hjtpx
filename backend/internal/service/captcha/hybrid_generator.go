package captcha

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type HybridCaptchaPhase string

const (
	HybridPhaseSlider    HybridCaptchaPhase = "slider"
	HybridPhaseClick     HybridCaptchaPhase = "click"
	HybridPhaseCompleted HybridCaptchaPhase = "completed"
)

type HybridCaptchaData struct {
	SliderGapX      int           `json:"slider_gap_x"`
	SliderGapY      int           `json:"slider_gap_y"`
	BackgroundURL   string        `json:"background_url"`
	SliderURL       string        `json:"slider_url"`
	ClickTargets    []ClickTarget `json:"click_targets"`
	ClickHints      []string      `json:"click_hints"`
	RequiredClicks  int           `json:"required_clicks"`
	CurrentPhase    HybridCaptchaPhase `json:"current_phase"`
	SliderVerified  bool          `json:"slider_verified"`
	ClickVerified   bool          `json:"click_verified"`
	ClickResults    []bool        `json:"click_results"`
}

type ClickTarget struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	ID     string `json:"id"`
}

type CreateHybridRequest struct {
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	SliderWidth  int    `json:"slider_width"`
	SliderHeight int    `json:"slider_height"`
	ClickCount   int    `json:"click_count"`
	ClientIP     string `json:"client_ip"`
	UserAgent    string `json:"user_agent"`
	Fingerprint  string `json:"fingerprint"`
}

type CreateHybridResponse struct {
	SessionID       string            `json:"session_id"`
	Phase           HybridCaptchaPhase `json:"phase"`
	BackgroundURL   string            `json:"background_url"`
	SliderURL       string            `json:"slider_url"`
	SliderGapX      int               `json:"slider_gap_x"`
	SliderGapY      int               `json:"slider_gap_y"`
	ClickPhaseHint  string            `json:"click_phase_hint"`
	ExpiresIn       int64             `json:"expires_in"`
	ExpiresAt       int64             `json:"expires_at"`
}

type VerifyHybridSliderRequest struct {
	SessionID  string          `json:"session_id" binding:"required"`
	PositionX  int             `json:"position_x" binding:"required"`
	PositionY  int             `json:"position_y" binding:"required"`
	Trajectory []TrajectoryData `json:"trajectory"`
	RiskScore  float64         `json:"risk_score"`
}

type VerifyHybridClickRequest struct {
	SessionID   string         `json:"session_id" binding:"required"`
	ClickX      int            `json:"click_x" binding:"required"`
	ClickY      int            `json:"click_y" binding:"required"`
	ClickIndex  int            `json:"click_index" binding:"required"`
	ClickTime   int64          `json:"click_time"`
	RiskScore   float64        `json:"risk_score"`
}

type VerifyHybridResult struct {
	Success        bool                `json:"success"`
	Message        string              `json:"message"`
	Score          float64             `json:"score"`
	Phase          HybridCaptchaPhase  `json:"phase"`
	NextPhaseHint  string              `json:"next_phase_hint"`
	TotalClicks    int                 `json:"total_clicks"`
	CorrectClicks  int                 `json:"correct_clicks"`
}

type HybridGeneratorService struct {
	imageGenerator *ImageGenerator
	sessionCache    *cache.SessionCache
	captchaRepo     *db.CaptchaRepository
}

type HybridVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type TrajectoryData struct {
	X         int   `json:"x"`
	Y         int   `json:"y"`
	Timestamp int64 `json:"timestamp"`
}

func NewHybridGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *HybridGeneratorService {
	return &HybridGeneratorService{
		imageGenerator: NewImageGenerator(),
		sessionCache:    sessionCache,
		captchaRepo:     captchaRepo,
	}
}

func NewHybridVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *HybridVerifierService {
	return &HybridVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (s *HybridGeneratorService) Create(ctx context.Context, req *CreateHybridRequest) (*CreateHybridResponse, error) {
	width := req.Width
	height := req.Height
	sliderWidth := req.SliderWidth
	sliderHeight := req.SliderHeight

	if width <= 0 {
		width = 320
	}
	if height <= 0 {
		height = 160
	}
	if sliderWidth <= 0 {
		sliderWidth = 40
	}
	if sliderHeight <= 0 {
		sliderHeight = 40
	}
	if req.ClickCount <= 0 {
		req.ClickCount = 3
	}

	s.imageGenerator.SetDimensions(width, height, sliderWidth, sliderHeight)

	sliderResult, err := s.imageGenerator.GenerateSliderCaptcha()
	if err != nil {
		return nil, fmt.Errorf("failed to generate slider captcha: %w", err)
	}

	backgroundBase64 := base64.StdEncoding.EncodeToString(sliderResult.Background)
	sliderBase64 := base64.StdEncoding.EncodeToString(sliderResult.Slider)
	clickTargets := s.generateClickTargets(width, height, req.ClickCount)

	hints := make([]string, len(clickTargets))
	for i := range hints {
		hints[i] = fmt.Sprintf("点击目标 %d", i+1)
	}

	hybridData := &HybridCaptchaData{
		SliderGapX:     sliderResult.GapX,
		SliderGapY:     sliderResult.GapY,
		BackgroundURL:  "data:image/png;base64," + backgroundBase64,
		SliderURL:     "data:image/png;base64," + sliderBase64,
		ClickTargets:  clickTargets,
		ClickHints:     hints,
		RequiredClicks: req.ClickCount,
		CurrentPhase:   HybridPhaseSlider,
		SliderVerified: false,
		ClickVerified:  false,
		ClickResults:   make([]bool, 0),
	}

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(10 * time.Minute)

	hybridDataJSON, err := json.Marshal(hybridData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal hybrid data: %w", err)
	}

	session := &models.CaptchaSession{
		SessionID:     sessionID,
		BackgroundURL: string(hybridDataJSON),
		SliderURL:     string(hybridDataJSON),
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   5,
		CreatedAt:     time.Now(),
		ExpiredAt:     expiresAt,
		ClientIP:      req.ClientIP,
		UserAgent:     req.UserAgent,
		Fingerprint:   req.Fingerprint,
		GapX:          sliderResult.GapX,
		GapY:          sliderResult.GapY,
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

	return &CreateHybridResponse{
		SessionID:       sessionID,
		Phase:           HybridPhaseSlider,
		BackgroundURL:   hybridData.BackgroundURL,
		SliderURL:       hybridData.SliderURL,
		SliderGapX:      hybridData.SliderGapX,
		SliderGapY:      hybridData.SliderGapY,
		ClickPhaseHint:  fmt.Sprintf("滑块验证通过后，请点击 %d 个指定目标", req.ClickCount),
		ExpiresIn:       int64(10 * time.Minute / time.Second),
		ExpiresAt:       expiresAt.Unix(),
	}, nil
}

func (s *HybridGeneratorService) generateClickTargets(width, height, count int) []ClickTarget {
	targets := make([]ClickTarget, count)
	margin := 20
	targetSize := 40

	positions := make(map[string]bool)
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < count; i++ {
		var x, y int
		maxAttempts := 100
		attempts := 0

		for {
			x = margin + rand.Intn(width-margin*2-targetSize)
			y = margin + rand.Intn(height-margin*2-targetSize)

			posKey := fmt.Sprintf("%d,%d", x/30, y/30)
			if !positions[posKey] {
				positions[posKey] = true
				break
			}

			attempts++
			if attempts >= maxAttempts {
				break
			}
		}

		targets[i] = ClickTarget{
			X:      x,
			Y:      y,
			Width:  targetSize,
			Height: targetSize,
			ID:     fmt.Sprintf("target_%d", i),
		}
	}

	return targets
}

func (s *HybridVerifierService) VerifySlider(ctx context.Context, req *VerifyHybridSliderRequest) (*VerifyHybridResult, error) {
	session, err := s.getSession(req.SessionID)
	if err != nil {
		return &VerifyHybridResult{
			Success: false,
			Message: "会话不存在",
			Score:   0,
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifyHybridResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifyHybridResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	s.incrementVerifyCount(req.SessionID)

	var hybridData HybridCaptchaData
	if err := json.Unmarshal([]byte(session.BackgroundURL), &hybridData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal hybrid data: %w", err)
	}

	if hybridData.SliderVerified {
		return &VerifyHybridResult{
			Success:       true,
			Message:       "滑块已验证",
			Score:         100,
			Phase:         HybridPhaseClick,
			NextPhaseHint: fmt.Sprintf("请点击 %d 个目标", hybridData.RequiredClicks),
			TotalClicks:   hybridData.RequiredClicks,
			CorrectClicks: 0,
		}, nil
	}

	tolerance := 10
	xDiff := abs(req.PositionX - hybridData.SliderGapX)

	trajectoryScore := s.analyzeSliderTrajectory(req.Trajectory, hybridData.SliderGapX)

	sliderValid := xDiff <= tolerance && trajectoryScore >= 0.6

	if sliderValid {
		hybridData.SliderVerified = true
		hybridData.CurrentPhase = HybridPhaseClick

		hybridDataJSON, _ := json.Marshal(hybridData)
		session.BackgroundURL = string(hybridDataJSON)
		session.SliderURL = string(hybridDataJSON)

		if s.sessionCache != nil {
			_ = s.sessionCache.Set(ctx, session)
		}

		return &VerifyHybridResult{
			Success:       true,
			Message:       "滑块验证成功",
			Score:         100,
			Phase:         HybridPhaseClick,
			NextPhaseHint: fmt.Sprintf("请点击 %d 个指定目标", hybridData.RequiredClicks),
			TotalClicks:   hybridData.RequiredClicks,
			CorrectClicks: 0,
		}, nil
	}

	return &VerifyHybridResult{
		Success:       false,
		Message:       "滑块位置不正确",
		Score:         float64(xDiff) / float64(tolerance+1) * 50,
		Phase:         HybridPhaseSlider,
		NextPhaseHint: "请将滑块拖动到正确位置",
		TotalClicks:   hybridData.RequiredClicks,
		CorrectClicks: 0,
	}, nil
}

func (s *HybridVerifierService) VerifyClick(ctx context.Context, req *VerifyHybridClickRequest) (*VerifyHybridResult, error) {
	session, err := s.getSession(req.SessionID)
	if err != nil {
		return &VerifyHybridResult{
			Success: false,
			Message: "会话不存在",
			Score:   0,
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifyHybridResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifyHybridResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	s.incrementVerifyCount(req.SessionID)

	var hybridData HybridCaptchaData
	if err := json.Unmarshal([]byte(session.BackgroundURL), &hybridData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal hybrid data: %w", err)
	}

	if !hybridData.SliderVerified {
		return &VerifyHybridResult{
			Success:       false,
			Message:       "请先完成滑块验证",
			Score:         0,
			Phase:         HybridPhaseSlider,
			NextPhaseHint: "请先完成滑块验证",
			TotalClicks:   hybridData.RequiredClicks,
			CorrectClicks: len(hybridData.ClickResults),
		}, nil
	}

	if req.ClickIndex < 0 || req.ClickIndex >= hybridData.RequiredClicks {
		return &VerifyHybridResult{
			Success:       false,
			Message:       "无效的点击索引",
			Score:         0,
			Phase:         HybridPhaseClick,
			NextPhaseHint: fmt.Sprintf("请点击目标 %d", req.ClickIndex+1),
			TotalClicks:   hybridData.RequiredClicks,
			CorrectClicks: len(hybridData.ClickResults),
		}, nil
	}

	if req.ClickIndex != len(hybridData.ClickResults) {
		return &VerifyHybridResult{
			Success:       false,
			Message:       "点击顺序不正确",
			Score:         0,
			Phase:         HybridPhaseClick,
			NextPhaseHint: fmt.Sprintf("请按顺序点击，还需点击 %d 个目标", hybridData.RequiredClicks-len(hybridData.ClickResults)),
			TotalClicks:   hybridData.RequiredClicks,
			CorrectClicks: len(hybridData.ClickResults),
		}, nil
	}

	target := hybridData.ClickTargets[req.ClickIndex]
	tolerance := 15

	xInRange := req.ClickX >= target.X-tolerance && req.ClickX <= target.X+target.Width+tolerance
	yInRange := req.ClickY >= target.Y-tolerance && req.ClickY <= target.Y+target.Height+tolerance

	clickValid := xInRange && yInRange
	hybridData.ClickResults = append(hybridData.ClickResults, clickValid)

	correctCount := 0
	for _, r := range hybridData.ClickResults {
		if r {
			correctCount++
		}
	}

	if len(hybridData.ClickResults) >= hybridData.RequiredClicks {
		hybridData.ClickVerified = true
		hybridData.CurrentPhase = HybridPhaseCompleted

		session.Status = "verified"

		if s.sessionCache != nil {
			_ = s.sessionCache.UpdateStatus(ctx, req.SessionID, "verified")
		}
		if s.captchaRepo != nil {
			_ = s.captchaRepo.UpdateStatus(req.SessionID, "verified")
		}
	}

	hybridDataJSON, _ := json.Marshal(hybridData)
	session.BackgroundURL = string(hybridDataJSON)
	session.SliderURL = string(hybridDataJSON)

	if s.sessionCache != nil {
		_ = s.sessionCache.Set(ctx, session)
	}

	if hybridData.ClickVerified {
		return &VerifyHybridResult{
			Success:       true,
			Message:       "全部验证成功",
			Score:         100,
			Phase:         HybridPhaseCompleted,
			NextPhaseHint: "",
			TotalClicks:   hybridData.RequiredClicks,
			CorrectClicks: correctCount,
		}, nil
	}

	return &VerifyHybridResult{
		Success:       clickValid,
		Message:       func() string {
			if clickValid {
				return fmt.Sprintf("第 %d 个目标点击正确", req.ClickIndex+1)
			}
			return fmt.Sprintf("第 %d 个目标点击错误", req.ClickIndex+1)
		}(),
		Score:         func() float64 {
			if clickValid {
				return 100
			}
			return 0
		}(),
		Phase:         HybridPhaseClick,
		NextPhaseHint: fmt.Sprintf("请点击目标 %d", len(hybridData.ClickResults)+1),
		TotalClicks:   hybridData.RequiredClicks,
		CorrectClicks: correctCount,
	}, nil
}

func (s *HybridVerifierService) analyzeSliderTrajectory(trajectory []TrajectoryData, targetX int) float64 {
	if len(trajectory) < 5 {
		return 0.5
	}

	var totalSpeed float64
	var speedChanges int

	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)

		if dt > 0 {
			speed := (dx*dx + dy*dy) / (dt * dt)
			totalSpeed += speed

			if i > 1 {
				prevDx := float64(trajectory[i-1].X - trajectory[i-2].X)
				prevDy := float64(trajectory[i-1].Y - trajectory[i-2].Y)
				prevDt := float64(trajectory[i-1].Timestamp - trajectory[i-2].Timestamp)
				if prevDt > 0 {
					prevSpeed := (prevDx*prevDx + prevDy*prevDy) / (prevDt * prevDt)
					if (speed-prevSpeed)/(prevSpeed+0.01) > 0.3 {
						speedChanges++
					}
				}
			}
		}
	}

	avgSpeed := totalSpeed / float64(len(trajectory)-1)

	yVariance := 0.0
	meanY := 0.0
	for _, p := range trajectory {
		meanY += float64(p.Y)
	}
	meanY /= float64(len(trajectory))

	for _, p := range trajectory {
		diff := float64(p.Y) - meanY
		yVariance += diff * diff
	}
	yVariance /= float64(len(trajectory))

	if yVariance < 5 && avgSpeed < 100 {
		return 0.3
	}

	speedVariationScore := float64(speedChanges) / float64(len(trajectory)-1)
	if speedVariationScore < 0.1 {
		return 0.4
	}

	return 0.7 + speedVariationScore*0.3
}

func (s *HybridVerifierService) getSession(sessionID string) (*models.CaptchaSession, error) {
	if s.sessionCache != nil {
		session, err := s.sessionCache.Get(context.Background(), sessionID)
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

func (s *HybridVerifierService) incrementVerifyCount(sessionID string) {
	if s.sessionCache != nil {
		_ = s.sessionCache.IncrementVerifyCount(context.Background(), sessionID)
	}

	if s.captchaRepo != nil {
		_ = s.captchaRepo.UpdateVerifyCount(sessionID)
	}
}

func (s *HybridVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	return s.getSession(sessionID)
}

func (s *HybridVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
	session, err := s.getSession(sessionID)
	if err != nil {
		return false, "会话不存在"
	}

	if time.Now().After(session.ExpiredAt) {
		return false, "验证码已过期"
	}

	if session.Status == "verified" {
		return false, "验证码已验证通过"
	}

	if session.VerifyCount >= session.MaxAttempts {
		return false, "验证次数已用完"
	}

	return true, ""
}

func (s *HybridVerifierService) GetHybridData(ctx context.Context, sessionID string) (*HybridCaptchaData, error) {
	session, err := s.getSession(sessionID)
	if err != nil {
		return nil, err
	}

	var hybridData HybridCaptchaData
	if err := json.Unmarshal([]byte(session.BackgroundURL), &hybridData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal hybrid data: %w", err)
	}

	return &hybridData, nil
}
