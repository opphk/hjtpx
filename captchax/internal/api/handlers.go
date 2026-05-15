package api

import (
	"bytes"
	"captchax/internal/service"
	"captchax/pkg/response"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CaptchaServiceInterface interface {
	GenerateSliderCaptcha(ctx context.Context, appID, clientInfo string) (*service.SliderCaptchaResult, error)
	VerifySliderCaptcha(ctx context.Context, captchaID string, targetX, targetY int) (*service.SliderVerifyResult, error)
	GenerateClickCaptcha(ctx context.Context, appID string, charCount int, clientInfo string) (*service.ClickCaptchaResult, error)
	VerifyClickCaptcha(ctx context.Context, captchaID string, clicks []service.CharPositionDTO) (*service.ClickVerifyResult, error)
	GeneratePuzzleCaptcha(ctx context.Context, appID, clientInfo string) (*service.PuzzleCaptchaResult, error)
	VerifyPuzzleCaptcha(ctx context.Context, captchaID string, targetX, targetY int) (*service.PuzzleVerifyResult, error)
	GenerateRotateCaptcha(ctx context.Context, appID, clientInfo string) (*service.RotateCaptchaResult, error)
	VerifyRotateCaptcha(ctx context.Context, captchaID string, angle int) (*service.RotateVerifyResult, error)
	GenerateTextCaptcha(ctx context.Context, appID, clientInfo string) (*service.TextCaptchaResult, error)
	VerifyTextCaptcha(ctx context.Context, captchaID string, code string) (*service.TextVerifyResult, error)
	GenerateIconCaptcha(ctx context.Context, appID, clientInfo string) (*service.IconCaptchaResult, error)
	VerifyIconCaptcha(ctx context.Context, captchaID string, iconIDs []string) (*service.IconVerifyResult, error)
	LogVerification(ctx context.Context, appID, captchaType, captchaID string, success bool, message, ip string)
}

type Handler struct {
	captchaService CaptchaServiceInterface
	scenarios      map[string]*Scenario
	scenariosMu    sync.RWMutex
	webhooks       map[string]*Webhook
	webhooksMu     sync.RWMutex
}

func NewHandler(captchaService *service.CaptchaService) *Handler {
	return &Handler{
		captchaService: captchaService,
		scenarios:      make(map[string]*Scenario),
		webhooks:      make(map[string]*Webhook),
	}
}

type SliderGenerateRequest struct {
	AppID      string `json:"app_id" binding:"required"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	ClientInfo string `json:"client_info"`
	ScenarioID string `json:"scenario_id"`
}

type SliderGenerateResponse struct {
	ID            string `json:"id"`
	BackgroundB64 string `json:"background_b64"`
	SliderB64     string `json:"slider_b64"`
	TargetX       int    `json:"target_x"`
	TargetY       int    `json:"target_y"`
}

type SliderVerifyRequest struct {
	CaptchaID string `json:"captcha_id" binding:"required"`
	TargetX   int    `json:"target_x" binding:"required"`
	TargetY   int    `json:"target_y"`
}

type SliderVerifyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ClickGenerateRequest struct {
	AppID      string `json:"app_id" binding:"required"`
	CharCount  int    `json:"char_count"`
	ClientInfo string `json:"client_info"`
	ScenarioID string `json:"scenario_id"`
}

type ClickGenerateResponse struct {
	ID            string            `json:"id"`
	Image         string            `json:"image"`
	TargetChars   []string          `json:"target_chars"`
	CharPositions []service.CharPositionDTO `json:"char_positions"`
}

type ClickVerifyRequest struct {
	CaptchaID string                    `json:"captcha_id" binding:"required"`
	Clicks    []service.CharPositionDTO `json:"clicks" binding:"required"`
}

type ClickVerifyResponse struct {
	Success bool    `json:"success"`
	Score   float64 `json:"score"`
	Message string  `json:"message"`
}

type PuzzleGenerateRequest struct {
	AppID      string `json:"app_id" binding:"required"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	ClientInfo string `json:"client_info"`
	ScenarioID string `json:"scenario_id"`
}

type PuzzleGenerateResponse struct {
	ID            string `json:"id"`
	BackgroundB64 string `json:"background_b64"`
	PuzzleB64     string `json:"puzzle_b64"`
	TargetX       int    `json:"target_x"`
	TargetY       int    `json:"target_y"`
}

type PuzzleVerifyRequest struct {
	CaptchaID string `json:"captcha_id" binding:"required"`
	TargetX   int    `json:"target_x" binding:"required"`
	TargetY   int    `json:"target_y"`
}

type PuzzleVerifyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type RotateGenerateRequest struct {
	AppID      string `json:"app_id" binding:"required"`
	ClientInfo string `json:"client_info"`
	ScenarioID string `json:"scenario_id"`
}

type RotateGenerateResponse struct {
	ID          string `json:"id"`
	ImageB64    string `json:"image_b64"`
	OriginalB64 string `json:"original_b64"`
}

type RotateVerifyRequest struct {
	CaptchaID string `json:"captcha_id" binding:"required"`
	Angle     int    `json:"angle" binding:"required"`
}

type RotateVerifyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type TextGenerateRequest struct {
	AppID      string `json:"app_id" binding:"required"`
	ClientInfo string `json:"client_info"`
	ScenarioID string `json:"scenario_id"`
}

type TextGenerateResponse struct {
	ID       string `json:"id"`
	ImageB64 string `json:"image_b64"`
}

type TextVerifyRequest struct {
	CaptchaID string `json:"captcha_id" binding:"required"`
	Code      string `json:"code" binding:"required"`
}

type TextVerifyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type IconGenerateRequest struct {
	AppID      string `json:"app_id" binding:"required"`
	ClientInfo string `json:"client_info"`
	ScenarioID string `json:"scenario_id"`
}

type IconInfoDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	SVG    string `json:"svg"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type IconGenerateResponse struct {
	ID          string        `json:"id"`
	TargetIcons []IconInfoDTO `json:"target_icons"`
	AllIcons    []IconInfoDTO `json:"all_icons"`
	GridCols    int           `json:"grid_cols"`
	GridRows    int           `json:"grid_rows"`
	IconSize    int           `json:"icon_size"`
}

type IconVerifyRequest struct {
	CaptchaID string   `json:"captcha_id" binding:"required"`
	IconIDs   []string `json:"icon_ids" binding:"required"`
}

type IconVerifyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type Scenario struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Difficulty  string                 `json:"difficulty"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type Webhook struct {
	ID        string            `json:"id"`
	AppID     string            `json:"app_id" binding:"required"`
	URL       string            `json:"url" binding:"required,url"`
	Secret    string            `json:"secret"`
	Events    []string          `json:"events" binding:"required"`
	Enabled   bool              `json:"enabled"`
	Headers   map[string]string `json:"headers"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type BatchVerifyRequest struct {
	Items []BatchVerifyItem `json:"items" binding:"required,dive"`
}

type BatchVerifyItem struct {
	CaptchaID string                    `json:"captcha_id" binding:"required"`
	Type      string                    `json:"type" binding:"required"`
	TargetX   int                       `json:"target_x" binding:"required"`
	TargetY   int                       `json:"target_y"`
	Clicks    []service.CharPositionDTO `json:"clicks"`
}

type BatchVerifyResponse struct {
	Results []BatchVerifyResult `json:"results"`
	Summary BatchVerifySummary  `json:"summary"`
}

type BatchVerifyResult struct {
	CaptchaID string  `json:"captcha_id"`
	Success   bool    `json:"success"`
	Message   string  `json:"message"`
	Score     float64 `json:"score,omitempty"`
}

type BatchVerifySummary struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

func (h *Handler) HealthCheck(c *gin.Context) {
	response.Success(c, gin.H{
		"status":    "healthy",
		"service":   "captchax-api",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "2.0.0",
	})
}

func (h *Handler) getSliderCaptcha(c *gin.Context) {
	var req SliderGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.GenerateSliderCaptcha(ctx, req.AppID, req.ClientInfo)
	if err != nil {
		h.captchaService.LogVerification(ctx, req.AppID, "slider", "", false, "generate_failed: "+err.Error(), c.ClientIP())
		response.InternalError(c, "failed to generate slider captcha")
		return
	}

	h.captchaService.LogVerification(ctx, req.AppID, "slider", result.ID, false, "generated", c.ClientIP())
	response.Success(c, result)
}

func (h *Handler) verifySliderCaptcha(c *gin.Context) {
	var req SliderVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.VerifySliderCaptcha(ctx, req.CaptchaID, req.TargetX, req.TargetY)
	if err != nil {
		h.captchaService.LogVerification(ctx, "", "slider", req.CaptchaID, false, err.Error(), c.ClientIP())
		response.InternalError(c, "verification failed")
		return
	}

	h.captchaService.LogVerification(ctx, "", "slider", req.CaptchaID, result.Success, result.Message, c.ClientIP())
	response.Success(c, result)
}

func (h *Handler) getClickCaptcha(c *gin.Context) {
	var req ClickGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.GenerateClickCaptcha(ctx, req.AppID, req.CharCount, req.ClientInfo)
	if err != nil {
		h.captchaService.LogVerification(ctx, req.AppID, "click", "", false, "generate_failed: "+err.Error(), c.ClientIP())
		response.InternalError(c, "failed to generate click captcha")
		return
	}

	h.captchaService.LogVerification(ctx, req.AppID, "click", result.ID, false, "generated", c.ClientIP())
	response.Success(c, result)
}

func (h *Handler) verifyClickCaptcha(c *gin.Context) {
	var req ClickVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.VerifyClickCaptcha(ctx, req.CaptchaID, req.Clicks)
	if err != nil {
		h.captchaService.LogVerification(ctx, "", "click", req.CaptchaID, false, err.Error(), c.ClientIP())
		response.InternalError(c, "verification failed")
		return
	}

	h.captchaService.LogVerification(ctx, "", "click", req.CaptchaID, result.Success, result.Message, c.ClientIP())
	response.Success(c, result)
}

func (h *Handler) getPuzzleCaptcha(c *gin.Context) {
	var req PuzzleGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.GeneratePuzzleCaptcha(ctx, req.AppID, req.ClientInfo)
	if err != nil {
		h.captchaService.LogVerification(ctx, req.AppID, "puzzle", "", false, "generate_failed: "+err.Error(), c.ClientIP())
		response.InternalError(c, "failed to generate puzzle captcha")
		return
	}

	h.captchaService.LogVerification(ctx, req.AppID, "puzzle", result.ID, false, "generated", c.ClientIP())
	response.Success(c, result)
}

func (h *Handler) verifyPuzzleCaptcha(c *gin.Context) {
	var req PuzzleVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.VerifyPuzzleCaptcha(ctx, req.CaptchaID, req.TargetX, req.TargetY)
	if err != nil {
		h.captchaService.LogVerification(ctx, "", "puzzle", req.CaptchaID, false, err.Error(), c.ClientIP())
		response.InternalError(c, "verification failed")
		return
	}

	h.captchaService.LogVerification(ctx, "", "puzzle", req.CaptchaID, result.Success, result.Message, c.ClientIP())
	response.Success(c, result)
}

func (h *Handler) getRotateCaptcha(c *gin.Context) {
	var req RotateGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.GenerateRotateCaptcha(ctx, req.AppID, req.ClientInfo)
	if err != nil {
		h.captchaService.LogVerification(ctx, req.AppID, "rotate", "", false, "generate failed: "+err.Error(), c.ClientIP())
		response.InternalError(c, "failed to generate rotate captcha")
		return
	}

	h.captchaService.LogVerification(ctx, req.AppID, "rotate", result.ID, false, "generated", c.ClientIP())
	response.Success(c, result)
}

func (h *Handler) verifyRotateCaptcha(c *gin.Context) {
	var req RotateVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.VerifyRotateCaptcha(ctx, req.CaptchaID, req.Angle)
	if err != nil {
		h.captchaService.LogVerification(ctx, "", "rotate", req.CaptchaID, false, err.Error(), c.ClientIP())
		response.InternalError(c, "verification failed")
		return
	}

	h.captchaService.LogVerification(ctx, "", "rotate", req.CaptchaID, result.Success, result.Message, c.ClientIP())
	response.Success(c, result)
}

func (h *Handler) getTextCaptcha(c *gin.Context) {
	var req TextGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.GenerateTextCaptcha(ctx, req.AppID, req.ClientInfo)
	if err != nil {
		h.captchaService.LogVerification(ctx, req.AppID, "text", "", false, "generate failed: "+err.Error(), c.ClientIP())
		response.InternalError(c, "failed to generate text captcha")
		return
	}

	h.captchaService.LogVerification(ctx, req.AppID, "text", result.ID, false, "generated", c.ClientIP())
	response.Success(c, result)
}

func (h *Handler) verifyTextCaptcha(c *gin.Context) {
	var req TextVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.VerifyTextCaptcha(ctx, req.CaptchaID, req.Code)
	if err != nil {
		h.captchaService.LogVerification(ctx, "", "text", req.CaptchaID, false, err.Error(), c.ClientIP())
		response.InternalError(c, "verification failed")
		return
	}

	h.captchaService.LogVerification(ctx, "", "text", req.CaptchaID, result.Success, result.Message, c.ClientIP())
	response.Success(c, result)
}

func (h *Handler) getIconCaptcha(c *gin.Context) {
	var req IconGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.GenerateIconCaptcha(ctx, req.AppID, req.ClientInfo)
	if err != nil {
		h.captchaService.LogVerification(ctx, req.AppID, "icon", "", false, "generate failed: "+err.Error(), c.ClientIP())
		response.InternalError(c, "failed to generate icon captcha")
		return
	}

	h.captchaService.LogVerification(ctx, req.AppID, "icon", result.ID, false, "generated", c.ClientIP())
	response.Success(c, result)
}

func (h *Handler) verifyIconCaptcha(c *gin.Context) {
	var req IconVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.VerifyIconCaptcha(ctx, req.CaptchaID, req.IconIDs)
	if err != nil {
		h.captchaService.LogVerification(ctx, "", "icon", req.CaptchaID, false, err.Error(), c.ClientIP())
		response.InternalError(c, "verification failed")
		return
	}

	h.captchaService.LogVerification(ctx, "", "icon", req.CaptchaID, result.Success, result.Message, c.ClientIP())
	response.Success(c, result)
}

func (h *Handler) GetSliderCaptchaV2(c *gin.Context) {
	var req SliderGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	var difficulty string
	h.scenariosMu.RLock()
	if req.ScenarioID != "" {
		if scenario, ok := h.scenarios[req.ScenarioID]; ok {
			difficulty = scenario.Difficulty
		}
	}
	h.scenariosMu.RUnlock()

	ctx := context.Background()
	result, err := h.captchaService.GenerateSliderCaptcha(ctx, req.AppID, req.ClientInfo)
	if err != nil {
		h.captchaService.LogVerification(ctx, req.AppID, "slider", "", false, "generate_failed: "+err.Error(), c.ClientIP())
		response.InternalError(c, "failed to generate slider captcha")
		return
	}

	h.captchaService.LogVerification(ctx, req.AppID, "slider", result.ID, false, "generated", c.ClientIP())

	response.Success(c, gin.H{
		"id":             result.ID,
		"background_b64": result.BackgroundB64,
		"slider_b64":     result.SliderB64,
		"target_x":       result.TargetX,
		"target_y":       result.TargetY,
		"difficulty":     difficulty,
		"expires_in":     300,
	})
}

func (h *Handler) VerifySliderCaptchaV2(c *gin.Context) {
	var req SliderVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.VerifySliderCaptcha(ctx, req.CaptchaID, req.TargetX, req.TargetY)
	if err != nil {
		h.captchaService.LogVerification(ctx, "", "slider", req.CaptchaID, false, err.Error(), c.ClientIP())
		response.InternalError(c, "verification failed")
		return
	}

	h.captchaService.LogVerification(ctx, "", "slider", req.CaptchaID, result.Success, result.Message, c.ClientIP())

	h.triggerWebhooks(ctx, "verification.completed", gin.H{
		"captcha_id": req.CaptchaID,
		"type":       "slider",
		"success":    result.Success,
		"message":    result.Message,
		"timestamp":  time.Now().Format(time.RFC3339),
	})

	response.Success(c, result)
}

func (h *Handler) GetClickCaptchaV2(c *gin.Context) {
	var req ClickGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	var difficulty string
	h.scenariosMu.RLock()
	if req.ScenarioID != "" {
		if scenario, ok := h.scenarios[req.ScenarioID]; ok {
			difficulty = scenario.Difficulty
		}
	}
	h.scenariosMu.RUnlock()

	ctx := context.Background()
	charCount := req.CharCount
	if charCount <= 0 {
		charCount = 4
	}

	result, err := h.captchaService.GenerateClickCaptcha(ctx, req.AppID, charCount, req.ClientInfo)
	if err != nil {
		h.captchaService.LogVerification(ctx, req.AppID, "click", "", false, "generate_failed: "+err.Error(), c.ClientIP())
		response.InternalError(c, "failed to generate click captcha")
		return
	}

	h.captchaService.LogVerification(ctx, req.AppID, "click", result.ID, false, "generated", c.ClientIP())

	response.Success(c, gin.H{
		"id":             result.ID,
		"image":          result.Image,
		"target_chars":   result.TargetChars,
		"char_positions": result.CharPositions,
		"difficulty":     difficulty,
		"expires_in":     300,
	})
}

func (h *Handler) VerifyClickCaptchaV2(c *gin.Context) {
	var req ClickVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.VerifyClickCaptcha(ctx, req.CaptchaID, req.Clicks)
	if err != nil {
		h.captchaService.LogVerification(ctx, "", "click", req.CaptchaID, false, err.Error(), c.ClientIP())
		response.InternalError(c, "verification failed")
		return
	}

	h.captchaService.LogVerification(ctx, "", "click", req.CaptchaID, result.Success, result.Message, c.ClientIP())

	h.triggerWebhooks(ctx, "verification.completed", gin.H{
		"captcha_id": req.CaptchaID,
		"type":       "click",
		"success":    result.Success,
		"score":     result.Score,
		"message":   result.Message,
		"timestamp": time.Now().Format(time.RFC3339),
	})

	response.Success(c, result)
}

func (h *Handler) GetPuzzleCaptchaV2(c *gin.Context) {
	var req PuzzleGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	var difficulty string
	h.scenariosMu.RLock()
	if req.ScenarioID != "" {
		if scenario, ok := h.scenarios[req.ScenarioID]; ok {
			difficulty = scenario.Difficulty
		}
	}
	h.scenariosMu.RUnlock()

	ctx := context.Background()
	result, err := h.captchaService.GeneratePuzzleCaptcha(ctx, req.AppID, req.ClientInfo)
	if err != nil {
		h.captchaService.LogVerification(ctx, req.AppID, "puzzle", "", false, "generate_failed: "+err.Error(), c.ClientIP())
		response.InternalError(c, "failed to generate puzzle captcha")
		return
	}

	h.captchaService.LogVerification(ctx, req.AppID, "puzzle", result.ID, false, "generated", c.ClientIP())

	response.Success(c, gin.H{
		"id":             result.ID,
		"background_b64": result.BackgroundB64,
		"puzzle_b64":    result.PuzzleB64,
		"target_x":      result.TargetX,
		"target_y":      result.TargetY,
		"difficulty":    difficulty,
		"expires_in":    300,
	})
}

func (h *Handler) VerifyPuzzleCaptchaV2(c *gin.Context) {
	var req PuzzleVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.VerifyPuzzleCaptcha(ctx, req.CaptchaID, req.TargetX, req.TargetY)
	if err != nil {
		h.captchaService.LogVerification(ctx, "", "puzzle", req.CaptchaID, false, err.Error(), c.ClientIP())
		response.InternalError(c, "verification failed")
		return
	}

	h.captchaService.LogVerification(ctx, "", "puzzle", req.CaptchaID, result.Success, result.Message, c.ClientIP())

	h.triggerWebhooks(ctx, "verification.completed", gin.H{
		"captcha_id": req.CaptchaID,
		"type":       "puzzle",
		"success":    result.Success,
		"message":    result.Message,
		"timestamp":  time.Now().Format(time.RFC3339),
	})

	response.Success(c, result)
}

func (h *Handler) GetRotateCaptchaV2(c *gin.Context) {
	var req RotateGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	var difficulty string
	h.scenariosMu.RLock()
	if req.ScenarioID != "" {
		if scenario, ok := h.scenarios[req.ScenarioID]; ok {
			difficulty = scenario.Difficulty
		}
	}
	h.scenariosMu.RUnlock()

	ctx := context.Background()
	result, err := h.captchaService.GenerateRotateCaptcha(ctx, req.AppID, req.ClientInfo)
	if err != nil {
		h.captchaService.LogVerification(ctx, req.AppID, "rotate", "", false, "generate failed: "+err.Error(), c.ClientIP())
		response.InternalError(c, "failed to generate rotate captcha")
		return
	}

	h.captchaService.LogVerification(ctx, req.AppID, "rotate", result.ID, false, "generated", c.ClientIP())

	response.Success(c, gin.H{
		"id":           result.ID,
		"image_b64":    result.ImageB64,
		"original_b64": result.OriginalB64,
		"difficulty":   difficulty,
		"expires_in":   300,
	})
}

func (h *Handler) VerifyRotateCaptchaV2(c *gin.Context) {
	var req RotateVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.VerifyRotateCaptcha(ctx, req.CaptchaID, req.Angle)
	if err != nil {
		h.captchaService.LogVerification(ctx, "", "rotate", req.CaptchaID, false, err.Error(), c.ClientIP())
		response.InternalError(c, "verification failed")
		return
	}

	h.captchaService.LogVerification(ctx, "", "rotate", req.CaptchaID, result.Success, result.Message, c.ClientIP())

	h.triggerWebhooks(ctx, "verification.completed", gin.H{
		"captcha_id": req.CaptchaID,
		"type":       "rotate",
		"success":    result.Success,
		"message":    result.Message,
		"timestamp":  time.Now().Format(time.RFC3339),
	})

	response.Success(c, result)
}

func (h *Handler) GetTextCaptchaV2(c *gin.Context) {
	var req TextGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	var difficulty string
	h.scenariosMu.RLock()
	if req.ScenarioID != "" {
		if scenario, ok := h.scenarios[req.ScenarioID]; ok {
			difficulty = scenario.Difficulty
		}
	}
	h.scenariosMu.RUnlock()

	ctx := context.Background()
	result, err := h.captchaService.GenerateTextCaptcha(ctx, req.AppID, req.ClientInfo)
	if err != nil {
		h.captchaService.LogVerification(ctx, req.AppID, "text", "", false, "generate failed: "+err.Error(), c.ClientIP())
		response.InternalError(c, "failed to generate text captcha")
		return
	}

	h.captchaService.LogVerification(ctx, req.AppID, "text", result.ID, false, "generated", c.ClientIP())

	response.Success(c, gin.H{
		"id":         result.ID,
		"image_b64":  result.ImageB64,
		"difficulty": difficulty,
		"expires_in": 300,
	})
}

func (h *Handler) VerifyTextCaptchaV2(c *gin.Context) {
	var req TextVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.VerifyTextCaptcha(ctx, req.CaptchaID, req.Code)
	if err != nil {
		h.captchaService.LogVerification(ctx, "", "text", req.CaptchaID, false, err.Error(), c.ClientIP())
		response.InternalError(c, "verification failed")
		return
	}

	h.captchaService.LogVerification(ctx, "", "text", req.CaptchaID, result.Success, result.Message, c.ClientIP())

	h.triggerWebhooks(ctx, "verification.completed", gin.H{
		"captcha_id": req.CaptchaID,
		"type":       "text",
		"success":    result.Success,
		"message":    result.Message,
		"timestamp":  time.Now().Format(time.RFC3339),
	})

	response.Success(c, result)
}

func (h *Handler) GetIconCaptchaV2(c *gin.Context) {
	var req IconGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	var difficulty string
	h.scenariosMu.RLock()
	if req.ScenarioID != "" {
		if scenario, ok := h.scenarios[req.ScenarioID]; ok {
			difficulty = scenario.Difficulty
		}
	}
	h.scenariosMu.RUnlock()

	ctx := context.Background()
	result, err := h.captchaService.GenerateIconCaptcha(ctx, req.AppID, req.ClientInfo)
	if err != nil {
		h.captchaService.LogVerification(ctx, req.AppID, "icon", "", false, "generate failed: "+err.Error(), c.ClientIP())
		response.InternalError(c, "failed to generate icon captcha")
		return
	}

	h.captchaService.LogVerification(ctx, req.AppID, "icon", result.ID, false, "generated", c.ClientIP())

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

	response.Success(c, gin.H{
		"id":           result.ID,
		"target_icons":  targetIcons,
		"all_icons":    allIcons,
		"grid_cols":    result.GridCols,
		"grid_rows":    result.GridRows,
		"icon_size":    result.IconSize,
		"difficulty":    difficulty,
		"expires_in":    300,
	})
}

func (h *Handler) VerifyIconCaptchaV2(c *gin.Context) {
	var req IconVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	ctx := context.Background()
	result, err := h.captchaService.VerifyIconCaptcha(ctx, req.CaptchaID, req.IconIDs)
	if err != nil {
		h.captchaService.LogVerification(ctx, "", "icon", req.CaptchaID, false, err.Error(), c.ClientIP())
		response.InternalError(c, "verification failed")
		return
	}

	h.captchaService.LogVerification(ctx, "", "icon", req.CaptchaID, result.Success, result.Message, c.ClientIP())

	h.triggerWebhooks(ctx, "verification.completed", gin.H{
		"captcha_id": req.CaptchaID,
		"type":       "icon",
		"success":    result.Success,
		"message":    result.Message,
		"timestamp":  time.Now().Format(time.RFC3339),
	})

	response.Success(c, result)
}

func (h *Handler) BatchVerifyCaptcha(c *gin.Context) {
	var req BatchVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if len(req.Items) == 0 {
		response.BadRequest(c, "no items to verify")
		return
	}

	if len(req.Items) > 100 {
		response.BadRequest(c, "maximum 100 items per batch")
		return
	}

	results := make([]BatchVerifyResult, len(req.Items))
	successCount := 0
	failedCount := 0
	skippedCount := 0

	ctx := context.Background()

	for i, item := range req.Items {
		var result BatchVerifyResult
		result.CaptchaID = item.CaptchaID

		if item.Type == "" {
			result.Success = false
			result.Message = "type is required"
			skippedCount++
			results[i] = result
			continue
		}

		switch item.Type {
		case "slider", "puzzle":
			verifyResult, err := h.captchaService.VerifySliderCaptcha(ctx, item.CaptchaID, item.TargetX, item.TargetY)
			if err != nil {
				result.Success = false
				result.Message = err.Error()
				failedCount++
			} else {
				result.Success = verifyResult.Success
				result.Message = verifyResult.Message
				if verifyResult.Success {
					successCount++
				} else {
					failedCount++
				}
			}

		case "click":
			if len(item.Clicks) == 0 {
				result.Success = false
				result.Message = "clicks are required for click type"
				skippedCount++
				results[i] = result
				continue
			}
			verifyResult, err := h.captchaService.VerifyClickCaptcha(ctx, item.CaptchaID, item.Clicks)
			if err != nil {
				result.Success = false
				result.Message = err.Error()
				result.Score = 0
				failedCount++
			} else {
				result.Success = verifyResult.Success
				result.Message = verifyResult.Message
				result.Score = verifyResult.Score
				if verifyResult.Success {
					successCount++
				} else {
					failedCount++
				}
			}

		default:
			result.Success = false
			result.Message = fmt.Sprintf("unsupported captcha type: %s", item.Type)
			skippedCount++
		}

		results[i] = result
	}

	h.triggerWebhooks(ctx, "batch.verification.completed", gin.H{
		"results":   results,
		"summary":   BatchVerifySummary{Total: len(req.Items), Success: successCount, Failed: failedCount, Skipped: skippedCount},
		"timestamp": time.Now().Format(time.RFC3339),
	})

	response.Success(c, BatchVerifyResponse{
		Results: results,
		Summary: BatchVerifySummary{
			Total:   len(req.Items),
			Success: successCount,
			Failed:  failedCount,
			Skipped: skippedCount,
		},
	})
}

func (h *Handler) ListScenarios(c *gin.Context) {
	h.scenariosMu.RLock()
	defer h.scenariosMu.RUnlock()

	scenarios := make([]*Scenario, 0, len(h.scenarios))
	for _, s := range h.scenarios {
		scenarios = append(scenarios, s)
	}

	response.Success(c, gin.H{
		"scenarios": scenarios,
		"total":     len(scenarios),
	})
}

func (h *Handler) CreateScenario(c *gin.Context) {
	var scenario Scenario
	if err := c.ShouldBindJSON(&scenario); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	scenario.ID = uuid.New().String()
	scenario.CreatedAt = time.Now()
	scenario.UpdatedAt = time.Now()

	if scenario.Difficulty == "" {
		scenario.Difficulty = "medium"
	}

	h.scenariosMu.Lock()
	h.scenarios[scenario.ID] = &scenario
	h.scenariosMu.Unlock()

	response.Success(c, scenario)
}

func (h *Handler) GetScenario(c *gin.Context) {
	id := c.Param("id")

	h.scenariosMu.RLock()
	scenario, ok := h.scenarios[id]
	h.scenariosMu.RUnlock()

	if !ok {
		response.NotFound(c, "scenario not found")
		return
	}

	response.Success(c, scenario)
}

func (h *Handler) UpdateScenario(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	h.scenariosMu.Lock()
	defer h.scenariosMu.Unlock()

	scenario, ok := h.scenarios[id]
	if !ok {
		response.NotFound(c, "scenario not found")
		return
	}

	if name, ok := updates["name"].(string); ok {
		scenario.Name = name
	}
	if description, ok := updates["description"].(string); ok {
		scenario.Description = description
	}
	if difficulty, ok := updates["difficulty"].(string); ok {
		scenario.Difficulty = difficulty
	}
	if config, ok := updates["config"].(map[string]interface{}); ok {
		scenario.Config = config
	}
	scenario.UpdatedAt = time.Now()

	response.Success(c, scenario)
}

func (h *Handler) DeleteScenario(c *gin.Context) {
	id := c.Param("id")

	h.scenariosMu.Lock()
	defer h.scenariosMu.Unlock()

	if _, ok := h.scenarios[id]; !ok {
		response.NotFound(c, "scenario not found")
		return
	}

	delete(h.scenarios, id)
	response.Success(c, gin.H{"deleted": true})
}

func (h *Handler) RegisterWebhook(c *gin.Context) {
	var webhook Webhook
	if err := c.ShouldBindJSON(&webhook); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	webhook.ID = uuid.New().String()
	webhook.CreatedAt = time.Now()
	webhook.UpdatedAt = time.Now()
	webhook.Enabled = true

	if webhook.Secret == "" {
		webhook.Secret = uuid.New().String()
	}

	h.webhooksMu.Lock()
	h.webhooks[webhook.ID] = &webhook
	h.webhooksMu.Unlock()

	response.Success(c, webhook)
}

func (h *Handler) UnregisterWebhook(c *gin.Context) {
	id := c.Param("id")

	h.webhooksMu.Lock()
	defer h.webhooksMu.Unlock()

	if _, ok := h.webhooks[id]; !ok {
		response.NotFound(c, "webhook not found")
		return
	}

	delete(h.webhooks, id)
	response.Success(c, gin.H{"deleted": true})
}

func (h *Handler) ListWebhooks(c *gin.Context) {
	h.webhooksMu.RLock()
	defer h.webhooksMu.RUnlock()

	webhooks := make([]*Webhook, 0, len(h.webhooks))
	for _, w := range h.webhooks {
		webhooks = append(webhooks, w)
	}

	appID := c.Query("app_id")
	if appID != "" {
		filtered := make([]*Webhook, 0)
		for _, w := range webhooks {
			if w.AppID == appID {
				filtered = append(filtered, w)
			}
		}
		webhooks = filtered
	}

	response.Success(c, gin.H{
		"webhooks": webhooks,
		"total":    len(webhooks),
	})
}

func (h *Handler) UpdateWebhook(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	h.webhooksMu.Lock()
	defer h.webhooksMu.Unlock()

	webhook, ok := h.webhooks[id]
	if !ok {
		response.NotFound(c, "webhook not found")
		return
	}

	if url, ok := updates["url"].(string); ok {
		webhook.URL = url
	}
	if secret, ok := updates["secret"].(string); ok {
		webhook.Secret = secret
	}
	if events, ok := updates["events"].([]interface{}); ok {
		eventStrings := make([]string, len(events))
		for i, e := range events {
			if s, ok := e.(string); ok {
				eventStrings[i] = s
			}
		}
		webhook.Events = eventStrings
	}
	if enabled, ok := updates["enabled"].(bool); ok {
		webhook.Enabled = enabled
	}
	if headers, ok := updates["headers"].(map[string]interface{}); ok {
		headerStrings := make(map[string]string)
		for k, v := range headers {
			if s, ok := v.(string); ok {
				headerStrings[k] = s
			}
		}
		webhook.Headers = headerStrings
	}
	webhook.UpdatedAt = time.Now()

	response.Success(c, webhook)
}

func (h *Handler) triggerWebhooks(ctx context.Context, event string, payload map[string]interface{}) {
	h.webhooksMu.RLock()
	defer h.webhooksMu.RUnlock()

	payloadBytes, _ := json.Marshal(payload)

	for _, webhook := range h.webhooks {
		if !webhook.Enabled {
			continue
		}

		eventMatch := false
		for _, e := range webhook.Events {
			if e == event || e == "*" {
				eventMatch = true
				break
			}
		}

		if !eventMatch {
			continue
		}

		go func(w *Webhook) {
			req, _ := http.NewRequestWithContext(ctx, "POST", w.URL, bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Webhook-Event", event)
			req.Header.Set("X-Webhook-ID", w.ID)

			if w.Secret != "" {
				req.Header.Set("X-Webhook-Signature", w.Secret)
			}

			for k, v := range w.Headers {
				req.Header.Set(k, v)
			}

			client := &http.Client{Timeout: 10 * time.Second}
			client.Do(req)
		}(webhook)
	}
}
