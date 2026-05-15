package service

import (
	serverconfig "captchax/config"
	"captchax/internal/captcha/click"
	"captchax/internal/captcha/icon"
	"captchax/internal/captcha/rotate"
	"captchax/internal/captcha/slider"
	"captchax/internal/captcha/text"
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
	rotateGen    *rotate.Rotate
	rotateVerify *rotate.VerifyService
	rotateCache  *rotate.CacheManager
	textGen      *text.Text
	textVerify   *text.VerifyService
	textCache    *text.CacheManager
	iconGen      *icon.IconCaptcha
	iconVerify   *icon.VerifyService
	iconCache    *icon.CacheManager
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

type RotateCaptchaResult struct {
	ID          string `json:"id"`
	ImageB64    string `json:"image_b64"`
	OriginalB64 string `json:"original_b64"`
}

type RotateVerifyResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type TextCaptchaResult struct {
	ID       string `json:"id"`
	ImageB64 string `json:"image_b64"`
}

type TextVerifyResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type IconInfoDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	SVG    string `json:"svg"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type IconCaptchaResult struct {
	ID          string        `json:"id"`
	TargetIcons []IconInfoDTO `json:"target_icons"`
	AllIcons    []IconInfoDTO `json:"all_icons"`
	GridCols    int           `json:"grid_cols"`
	GridRows    int           `json:"grid_rows"`
	IconSize    int           `json:"icon_size"`
}

type IconVerifyResult struct {
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

	rotateCache := rotate.NewCacheManager(&cfg.Captcha, redisClient)
	rotateGen := rotate.New(&cfg.Captcha, redisClient)
	rotateVerify := rotate.NewVerifyService(&cfg.Captcha, rotateCache)

	textCache := text.NewCacheManager(&cfg.Captcha, redisClient)
	textGen := text.New(&cfg.Captcha, redisClient)
	textVerify := text.NewVerifyService(&cfg.Captcha, textCache)

	iconCache := icon.NewCacheManager(&cfg.Captcha, redisClient)
	iconGen := icon.New(&cfg.Captcha, redisClient)
	iconVerify := icon.NewVerifyService(&cfg.Captcha, iconCache)

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
		rotateGen:    rotateGen,
		rotateVerify: rotateVerify,
		rotateCache:  rotateCache,
		textGen:      textGen,
		textVerify:   textVerify,
		textCache:    textCache,
		iconGen:      iconGen,
		iconVerify:   iconVerify,
		iconCache:    iconCache,
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

func (s *CaptchaService) GenerateRotateCaptcha(ctx context.Context, appID, clientInfo string) (*RotateCaptchaResult, error) {
	behavior := &risk.BehaviorData{
		SessionID: uuid.New().String(),
	}

	riskResult := s.riskEngine.CalculateRiskScore(ctx, behavior, "", appID)
	if riskResult.Recommended == risk.ActionBlock {
		return nil, errors.New("risk level too high")
	}

	result, err := s.rotateGen.GenerateCaptcha(ctx)
	if err != nil {
		return nil, err
	}

	captcha := &model.Captcha{
		ID:         result.ID,
		AppID:      appID,
		Type:       string(model.CaptchaTypeRotate),
		ImageData:  result.ImageB64,
		Status:     int(model.CaptchaStatusPending),
		ClientInfo: clientInfo,
		ExpiredAt:  time.Now().Add(time.Duration(s.cfg.Captcha.ExpireMinutes) * time.Minute),
	}
	if err := s.saveCaptchaRecord(captcha); err != nil {
		log.Error("failed to save rotate captcha record", map[string]interface{}{
			"error": err.Error(),
			"id":    result.ID,
		})
	}

	return &RotateCaptchaResult{
		ID:          result.ID,
		ImageB64:    result.ImageB64,
		OriginalB64: result.OriginalB64,
	}, nil
}

func (s *CaptchaService) VerifyRotateCaptcha(ctx context.Context, captchaID string, angle int) (*RotateVerifyResult, error) {
	req := &rotate.VerifyRequest{
		CaptchaID: captchaID,
		Angle:     angle,
	}

	result, err := s.rotateVerify.Verify(ctx, req)
	if err != nil {
		return &RotateVerifyResult{
			Success: false,
			Message: result.Message,
		}, err
	}

	if result.Success {
		s.updateCaptchaStatus(captchaID, model.CaptchaStatusVerified)
	}

	return &RotateVerifyResult{
		Success: result.Success,
		Message: result.Message,
	}, nil
}

func (s *CaptchaService) GenerateTextCaptcha(ctx context.Context, appID, clientInfo string) (*TextCaptchaResult, error) {
	behavior := &risk.BehaviorData{
		SessionID: uuid.New().String(),
	}

	riskResult := s.riskEngine.CalculateRiskScore(ctx, behavior, "", appID)
	if riskResult.Recommended == risk.ActionBlock {
		return nil, errors.New("risk level too high")
	}

	result, err := s.textGen.GenerateCaptcha(ctx)
	if err != nil {
		return nil, err
	}

	captcha := &model.Captcha{
		ID:         result.ID,
		AppID:      appID,
		Type:       string(model.CaptchaTypeImage),
		ImageData:  result.ImageB64,
		Status:     int(model.CaptchaStatusPending),
		ClientInfo: clientInfo,
		ExpiredAt:  time.Now().Add(time.Duration(s.cfg.Captcha.ExpireMinutes) * time.Minute),
	}
	if err := s.saveCaptchaRecord(captcha); err != nil {
		log.Error("failed to save text captcha record", map[string]interface{}{
			"error": err.Error(),
			"id":    result.ID,
		})
	}

	return &TextCaptchaResult{
		ID:       result.ID,
		ImageB64: result.ImageB64,
	}, nil
}

func (s *CaptchaService) VerifyTextCaptcha(ctx context.Context, captchaID string, code string) (*TextVerifyResult, error) {
	req := &text.VerifyRequest{
		CaptchaID: captchaID,
		Code:      code,
	}

	result, err := s.textVerify.Verify(ctx, req)
	if err != nil {
		return &TextVerifyResult{
			Success: false,
			Message: result.Message,
		}, err
	}

	if result.Success {
		s.updateCaptchaStatus(captchaID, model.CaptchaStatusVerified)
	}

	return &TextVerifyResult{
		Success: result.Success,
		Message: result.Message,
	}, nil
}

func (s *CaptchaService) GenerateIconCaptcha(ctx context.Context, appID, clientInfo string) (*IconCaptchaResult, error) {
	behavior := &risk.BehaviorData{
		SessionID: uuid.New().String(),
	}

	riskResult := s.riskEngine.CalculateRiskScore(ctx, behavior, "", appID)
	if riskResult.Recommended == risk.ActionBlock {
		return nil, errors.New("risk level too high")
	}

	result, err := s.iconGen.GenerateCaptcha(ctx)
	if err != nil {
		return nil, err
	}

	captcha := &model.Captcha{
		ID:         result.ID,
		AppID:      appID,
		Type:       string(model.CaptchaTypeImage),
		Status:     int(model.CaptchaStatusPending),
		ClientInfo: clientInfo,
		ExpiredAt:  time.Now().Add(time.Duration(s.cfg.Captcha.ExpireMinutes) * time.Minute),
	}
	if err := s.saveCaptchaRecord(captcha); err != nil {
		log.Error("failed to save icon captcha record", map[string]interface{}{
			"error": err.Error(),
			"id":    result.ID,
		})
	}

	targetIcons := make([]IconInfoDTO, len(result.TargetIcons))
	for i, icon := range result.TargetIcons {
		targetIcons[i] = IconInfoDTO{
			ID:     icon.ID,
			Name:   icon.Name,
			SVG:    icon.SVG,
			Width:  icon.Width,
			Height: icon.Height,
		}
	}

	allIcons := make([]IconInfoDTO, len(result.AllIcons))
	for i, icon := range result.AllIcons {
		allIcons[i] = IconInfoDTO{
			ID:     icon.ID,
			Name:   icon.Name,
			SVG:    icon.SVG,
			Width:  icon.Width,
			Height: icon.Height,
		}
	}

	return &IconCaptchaResult{
		ID:          result.ID,
		TargetIcons: targetIcons,
		AllIcons:    allIcons,
		GridCols:    result.GridCols,
		GridRows:    result.GridRows,
		IconSize:    result.IconSize,
	}, nil
}

func (s *CaptchaService) VerifyIconCaptcha(ctx context.Context, captchaID string, iconIDs []string) (*IconVerifyResult, error) {
	req := &icon.VerifyRequest{
		CaptchaID: captchaID,
		IconIDs:   iconIDs,
	}

	result, err := s.iconVerify.Verify(ctx, req)
	if err != nil {
		return &IconVerifyResult{
			Success: false,
			Message: result.Message,
		}, err
	}

	if result.Success {
		s.updateCaptchaStatus(captchaID, model.CaptchaStatusVerified)
	}

	return &IconVerifyResult{
		Success: result.Success,
		Message: result.Message,
	}, nil
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
