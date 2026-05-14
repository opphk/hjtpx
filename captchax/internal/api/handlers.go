package api

import (
	"captchax/internal/service"
	"captchax/pkg/response"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CaptchaServiceInterface interface {
	GenerateSliderCaptcha(ctx context.Context, appID, clientInfo string) (*service.SliderCaptchaResult, error)
	VerifySliderCaptcha(ctx context.Context, captchaID string, targetX, targetY int) (*service.SliderVerifyResult, error)
	GenerateClickCaptcha(ctx context.Context, appID string, charCount int, clientInfo string) (*service.ClickCaptchaResult, error)
	VerifyClickCaptcha(ctx context.Context, captchaID string, clicks []service.CharPositionDTO) (*service.ClickVerifyResult, error)
	GeneratePuzzleCaptcha(ctx context.Context, appID, clientInfo string) (*service.PuzzleCaptchaResult, error)
	VerifyPuzzleCaptcha(ctx context.Context, captchaID string, targetX, targetY int) (*service.PuzzleVerifyResult, error)
	LogVerification(ctx context.Context, appID, captchaType, captchaID string, success bool, message, ip string)
}

type Handler struct {
	captchaService CaptchaServiceInterface
}

func NewHandler(captchaService *service.CaptchaService) *Handler {
	return &Handler{
		captchaService: captchaService,
	}
}

type SliderGenerateRequest struct {
	AppID      string `json:"app_id"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	ClientInfo string `json:"client_info"`
}

type SliderGenerateResponse struct {
	ID            string `json:"id"`
	BackgroundB64 string `json:"background_b64"`
	SliderB64     string `json:"slider_b64"`
	TargetX       int    `json:"target_x"`
	TargetY       int    `json:"target_y"`
}

type SliderVerifyRequest struct {
	CaptchaID string `json:"captcha_id"`
	TargetX   int    `json:"target_x"`
	TargetY   int    `json:"target_y"`
}

type SliderVerifyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ClickGenerateRequest struct {
	AppID      string `json:"app_id"`
	CharCount  int    `json:"char_count"`
	ClientInfo string `json:"client_info"`
}

type ClickGenerateResponse struct {
	ID            string            `json:"id"`
	Image         string            `json:"image"`
	TargetChars   []string          `json:"target_chars"`
	CharPositions []service.CharPositionDTO `json:"char_positions"`
}

type ClickVerifyRequest struct {
	CaptchaID string                    `json:"captcha_id"`
	Clicks    []service.CharPositionDTO `json:"clicks"`
}

type ClickVerifyResponse struct {
	Success bool    `json:"success"`
	Score   float64 `json:"score"`
	Message string  `json:"message"`
}

type PuzzleGenerateRequest struct {
	AppID      string `json:"app_id"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	ClientInfo string `json:"client_info"`
}

type PuzzleGenerateResponse struct {
	ID            string `json:"id"`
	BackgroundB64 string `json:"background_b64"`
	PuzzleB64     string `json:"puzzle_b64"`
	TargetX       int    `json:"target_x"`
	TargetY       int    `json:"target_y"`
}

type PuzzleVerifyRequest struct {
	CaptchaID string `json:"captcha_id"`
	TargetX   int    `json:"target_x"`
	TargetY   int    `json:"target_y"`
}

type PuzzleVerifyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (h *Handler) healthCheck(c *gin.Context) {
	response.Success(c, gin.H{
		"status":  "healthy",
		"service": "captchax-api",
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
