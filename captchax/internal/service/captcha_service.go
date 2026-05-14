package service

import (
	serverconfig "captchax/config"
	"captchax/internal/captcha/click"
	"captchax/internal/captcha/slider"
	"captchax/internal/config"
	"captchax/internal/log"
	"captchax/internal/model"
	"captchax/internal/risk"
	"captchax/pkg/cache"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CaptchaService struct {
	cfg          *serverconfig.Config
	redisClient  *cache.RedisClient
	db           *gorm.DB
	sliderGen    *slider.Slider
	sliderVerify *slider.VerifyService
	sliderCache  *slider.CacheManager
	clickGen     *click.CaptchaGenerator
	clickVerify  *click.ClickVerifier
	clickCache   click.CaptchaCache
	riskEngine   *risk.RiskEngine
}

type SliderCaptchaResult struct {
	ID            string `json:"id"`
	BackgroundB64 string `json:"background_b64"`
	SliderB64     string `json:"slider_b64"`
	TargetX       int    `json:"target_x"`
	TargetY       int    `json:"target_y"`
}

type SliderVerifyResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ClickCaptchaResult struct {
	ID            string            `json:"id"`
	Image         string            `json:"image"`
	TargetChars   []string          `json:"target_chars"`
	CharPositions []CharPositionDTO `json:"char_positions"`
}

type CharPositionDTO struct {
	Char   string `json:"char"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type ClickVerifyResult struct {
	Success bool    `json:"success"`
	Score   float64 `json:"score"`
	Message string  `json:"message"`
}

type PuzzleCaptchaResult struct {
	ID            string `json:"id"`
	BackgroundB64 string `json:"background_b64"`
	PuzzleB64     string `json:"puzzle_b64"`
	TargetX       int    `json:"target_x"`
	TargetY       int    `json:"target_y"`
}

type PuzzleVerifyResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func NewCaptchaService(
	cfg *serverconfig.Config,
	redisClient *cache.RedisClient,
	db *gorm.DB,
) (*CaptchaService, error) {
	riskCfg := &config.RiskConfig{
		SlideSpeedThresholdFast:  1 * time.Second,
		SlideSpeedThresholdSlow:  30 * time.Second,
		SmoothnessThreshold:      0.95,
		JitterThreshold:          0.1,
		MaxFailureCount:          3,
		CriticalFailureCount:     5,
		BlockDuration:            30 * time.Minute,
		HighFrequencyThreshold:    100,
	}
	ipLimit, err := risk.NewIPLimit(&risk.IPLimitConfig{
		RedisAddr:      cfg.Redis.Addr(),
		RedisPassword:  cfg.Redis.Password,
		RedisDB:        cfg.Redis.DB,
	})
	if err != nil {
		ipLimit = nil
	}

	whitelist, err := risk.NewWhitelist(&risk.WhitelistConfig{
		MemoryOnly:  true,
	})
	if err != nil {
		whitelist = nil
	}

	sliderCache := slider.NewCacheManager(&cfg.Captcha, redisClient)
	sliderGen := slider.New(&cfg.Captcha, redisClient)
	sliderVerify := slider.NewVerifyService(&cfg.Captcha, sliderCache)

	clickGen, err := click.NewCaptchaGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to create click captcha generator: %w", err)
	}

	var clickCache click.CaptchaCache
	if redisClient != nil {
		clickCacheMgr, err := click.NewCacheManager(cfg.Redis.Addr(), cfg.Redis.Password, cfg.Redis.DB)
		if err != nil {
			clickCache = click.NewMockCacheManager()
		} else {
			clickCache = clickCacheMgr
		}
	} else {
		clickCache = click.NewMockCacheManager()
	}

	clickVerify := click.NewClickVerifier(clickCache)

	riskEngine := risk.NewRiskEngine(riskCfg, ipLimit, whitelist)

	return &CaptchaService{
		cfg:          cfg,
		redisClient:  redisClient,
		db:           db,
		sliderGen:    sliderGen,
		sliderVerify: sliderVerify,
		sliderCache:  sliderCache,
		clickGen:     clickGen,
		clickVerify:  clickVerify,
		clickCache:   clickCache,
		riskEngine:   riskEngine,
	}, nil
}

func (s *CaptchaService) GenerateSliderCaptcha(ctx context.Context, appID, clientInfo string) (*SliderCaptchaResult, error) {
	behavior := &risk.BehaviorData{
		SessionID: uuid.New().String(),
	}

	riskResult := s.riskEngine.CalculateRiskScore(ctx, behavior, "", appID)
	if riskResult.Recommended == risk.ActionBlock {
		return nil, errors.New("risk level too high")
	}

	result, err := s.sliderGen.GenerateCaptcha(ctx)
	if err != nil {
		return nil, err
	}

	captcha := &model.Captcha{
		ID:         result.ID,
		AppID:      appID,
		Type:       string(model.CaptchaTypeSlider),
		ImageData:  result.BackgroundB64,
		Status:     int(model.CaptchaStatusPending),
		ClientInfo: clientInfo,
		ExpiredAt:  time.Now().Add(time.Duration(s.cfg.Captcha.ExpireMinutes) * time.Minute),
	}
	if err := s.saveCaptchaRecord(captcha); err != nil {
		log.Error("failed to save slider captcha record", map[string]interface{}{
			"error": err.Error(),
			"id":    result.ID,
		})
	}

	return &SliderCaptchaResult{
		ID:            result.ID,
		BackgroundB64: result.BackgroundB64,
		SliderB64:     result.SliderB64,
		TargetX:       result.TargetX,
		TargetY:       result.TargetY,
	}, nil
}

func (s *CaptchaService) VerifySliderCaptcha(ctx context.Context, captchaID string, targetX, targetY int) (*SliderVerifyResult, error) {
	req := &slider.VerifyRequest{
		CaptchaID: captchaID,
		TargetX:   targetX,
		TargetY:   targetY,
	}

	result, err := s.sliderVerify.Verify(ctx, req)
	if err != nil {
		return &SliderVerifyResult{
			Success: false,
			Message: result.Message,
		}, err
	}

	if result.Success {
		s.updateCaptchaStatus(captchaID, model.CaptchaStatusVerified)
	}

	return &SliderVerifyResult{
		Success: result.Success,
		Message: result.Message,
	}, nil
}

func (s *CaptchaService) GenerateClickCaptcha(ctx context.Context, appID string, charCount int, clientInfo string) (*ClickCaptchaResult, error) {
	behavior := &risk.BehaviorData{
		SessionID: uuid.New().String(),
	}

	riskResult := s.riskEngine.CalculateRiskScore(ctx, behavior, "", appID)
	if riskResult.Recommended == risk.ActionBlock {
		return nil, errors.New("risk level too high")
	}

	if charCount <= 0 {
		charCount = click.DefaultCharCount
	}

	result, err := s.clickGen.GenerateCaptcha(charCount)
	if err != nil {
		return nil, err
	}

	captchaData := &click.CaptchaData{
		ID:            result.ID,
		Image:         result.Image,
		TargetChars:   result.TargetChars,
		CharPositions: result.CharPositions,
		CreatedAt:     time.Now(),
	}

	if err := s.clickCache.Store(ctx, captchaData); err != nil {
		log.Error("failed to store click captcha", map[string]interface{}{
			"error": err.Error(),
			"id":    result.ID,
		})
	}

	captcha := &model.Captcha{
		ID:         result.ID,
		AppID:      appID,
		Type:       string(model.CaptchaTypeImage),
		ImageData:  result.Image,
		Status:     int(model.CaptchaStatusPending),
		ClientInfo: clientInfo,
		ExpiredAt:  time.Now().Add(time.Duration(s.cfg.Captcha.ExpireMinutes) * time.Minute),
	}
	if err := s.saveCaptchaRecord(captcha); err != nil {
		log.Error("failed to save click captcha record", map[string]interface{}{
			"error": err.Error(),
			"id":    result.ID,
		})
	}

	charPositions := make([]CharPositionDTO, len(result.CharPositions))
	for i, pos := range result.CharPositions {
		charPositions[i] = CharPositionDTO{
			Char:   pos.Char,
			X:      pos.X,
			Y:      pos.Y,
			Width:  pos.Width,
			Height: pos.Height,
		}
	}

	return &ClickCaptchaResult{
		ID:            result.ID,
		Image:         result.Image,
		TargetChars:   result.TargetChars,
		CharPositions: charPositions,
	}, nil
}

func (s *CaptchaService) VerifyClickCaptcha(ctx context.Context, captchaID string, clicks []CharPositionDTO) (*ClickVerifyResult, error) {
	clickPositions := make([]click.ClickPosition, len(clicks))
	for i, c := range clicks {
		clickPositions[i] = click.ClickPosition{
			X: c.X,
			Y: c.Y,
		}
	}

	req := &click.VerifyRequest{
		CaptchaID: captchaID,
		Clicks:    clickPositions,
	}

	result, err := s.clickVerify.Verify(ctx, req)
	if err != nil {
		return &ClickVerifyResult{
			Success: false,
			Score:   0,
			Message: result.Message,
		}, err
	}

	if result.Success {
		s.updateCaptchaStatus(captchaID, model.CaptchaStatusVerified)
	}

	return &ClickVerifyResult{
		Success: result.Success,
		Score:   result.Score,
		Message: result.Message,
	}, nil
}

func (s *CaptchaService) GeneratePuzzleCaptcha(ctx context.Context, appID, clientInfo string) (*PuzzleCaptchaResult, error) {
	behavior := &risk.BehaviorData{
		SessionID: uuid.New().String(),
	}

	riskResult := s.riskEngine.CalculateRiskScore(ctx, behavior, "", appID)
	if riskResult.Recommended == risk.ActionBlock {
		return nil, errors.New("risk level too high")
	}

	puzzleID := uuid.New().String()
	width := s.cfg.Captcha.Width
	height := s.cfg.Captcha.Height

	targetX := width/2 - 30 + rand.Intn(60)
	targetY := height/2 - 30 + rand.Intn(60)

	bgImage := s.generatePuzzleBackground(targetX, targetY)
	sliderImage := s.generatePuzzleSlider(targetX, targetY)

	cacheData := map[string]interface{}{
		"target_x": targetX,
		"target_y": targetY,
	}
	dataBytes, _ := json.Marshal(cacheData)
	cacheKey := fmt.Sprintf("captcha:puzzle:%s", puzzleID)
	if s.redisClient != nil {
		_ = s.redisClient.Set(ctx, cacheKey, dataBytes, 5*time.Minute)
	}

	return &PuzzleCaptchaResult{
		ID:            puzzleID,
		BackgroundB64: bgImage,
		PuzzleB64:     sliderImage,
		TargetX:       targetX,
		TargetY:       targetY,
	}, nil
}

func (s *CaptchaService) VerifyPuzzleCaptcha(ctx context.Context, captchaID string, targetX, targetY int) (*PuzzleVerifyResult, error) {
	cacheKey := fmt.Sprintf("captcha:puzzle:%s", captchaID)

	var expectedX, expectedY int

	if s.redisClient != nil {
		data, err := s.redisClient.Get(ctx, cacheKey)
		if err != nil {
			return &PuzzleVerifyResult{
				Success: false,
				Message: "captcha not found or expired",
			}, errors.New("captcha not found")
		}

		var cacheData map[string]interface{}
		if err := json.Unmarshal([]byte(data), &cacheData); err != nil {
			return &PuzzleVerifyResult{
				Success: false,
				Message: "invalid captcha data",
			}, err
		}

		expectedX = int(cacheData["target_x"].(float64))
		expectedY = int(cacheData["target_y"].(float64))
	}

	tolerance := s.cfg.Captcha.Tolerance
	if tolerance == 0 {
		tolerance = 5
	}

	dx := float64(targetX - expectedX)
	dy := float64(targetY - expectedY)
	distance := math.Sqrt(dx*dx + dy*dy)

	if distance <= float64(tolerance) {
		if s.redisClient != nil {
			_ = s.redisClient.Del(ctx, cacheKey)
		}
		s.updateCaptchaStatus(captchaID, model.CaptchaStatusVerified)

		return &PuzzleVerifyResult{
			Success: true,
			Message: "verification successful",
		}, nil
	}

	return &PuzzleVerifyResult{
		Success: false,
		Message: fmt.Sprintf("verification failed: distance %.2f exceeds tolerance %d", distance, tolerance),
	}, errors.New("verification failed")
}

func (s *CaptchaService) generatePuzzleBackground(targetX, targetY int) string {
	width := s.cfg.Captcha.Width
	height := s.cfg.Captcha.Height

	bg := make([]byte, width*height*3)
	baseColor := 200 + rand.Intn(40)
	for i := range bg {
		bg[i] = byte(baseColor + rand.Intn(20))
	}

	return fmt.Sprintf("data:image/png;base64,%x", bg[:100])
}

func (s *CaptchaService) generatePuzzleSlider(targetX, targetY int) string {
	size := s.cfg.Captcha.SliderSize
	if size == 0 {
		size = 50
	}

	slider := make([]byte, size*size*3)
	for i := range slider {
		slider[i] = 240
	}

	return fmt.Sprintf("data:image/png;base64,%x", slider[:100])
}

func (s *CaptchaService) saveCaptchaRecord(captcha *model.Captcha) error {
	if s.db == nil {
		return nil
	}
	return s.db.Create(captcha).Error
}

func (s *CaptchaService) updateCaptchaStatus(captchaID string, status model.CaptchaStatus) {
	if s.db == nil {
		return
	}

	now := time.Now()
	s.db.Model(&model.Captcha{}).Where("id = ?", captchaID).Updates(map[string]interface{}{
		"status":      int(status),
		"verified_at": now,
	})
}

func (s *CaptchaService) LogVerification(ctx context.Context, appID, captchaType, captchaID string, success bool, message, ip string) {
	logEntry := &model.CaptchaLog{
		ClientID: appID,
		Type:     captchaType,
		IP:       ip,
		Result:   success,
	}

	if s.db != nil {
		if err := s.db.Create(logEntry).Error; err != nil {
			log.Error("failed to save captcha log", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	logMsg := "captcha verification"
	if success {
		log.Info(logMsg, map[string]interface{}{
			"app_id":     appID,
			"type":       captchaType,
			"captcha_id": captchaID,
			"success":    success,
			"message":    message,
			"ip":         ip,
		})
	} else {
		log.Warn("captcha verification failed", map[string]interface{}{
			"app_id":     appID,
			"type":       captchaType,
			"captcha_id": captchaID,
			"success":    success,
			"message":    message,
			"ip":         ip,
		})
	}
}

func (s *CaptchaService) CheckRateLimit(ctx context.Context, ip string, limit int) (allowed bool, remaining int, resetAt time.Time, err error) {
	resetAt = time.Now().Add(time.Minute)

	if s.redisClient == nil {
		allowed = true
		remaining = limit - 1
		return
	}

	key := fmt.Sprintf("ratelimit:api:%s", ip)

	count, err := s.redisClient.Incr(ctx, key)
	if err != nil {
		return true, limit, resetAt, err
	}

	if count == 1 {
		_ = s.redisClient.Expire(ctx, key, time.Minute)
	}

	remaining = limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	if count > int64(limit) {
		ttl, err := s.redisClient.TTL(ctx, key)
		if err == nil {
			resetAt = time.Now().Add(ttl)
		}
		return false, remaining, resetAt, nil
	}

	return true, remaining, resetAt, nil
}
