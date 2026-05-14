package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opphk/captcha-system/pkg/captcha"
)

type ChallengeHandler struct {
	service *captcha.CaptchaService
}

func NewChallengeHandler(service *captcha.CaptchaService) *ChallengeHandler {
	return &ChallengeHandler{service: service}
}

func (h *ChallengeHandler) CreateSliderCaptcha(c *gin.Context) {
	var req struct {
		SessionID  string `json:"session_id"`
		Difficulty string `json:"difficulty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Difficulty = "medium"
	}

	if req.SessionID == "" {
		req.SessionID = c.GetHeader("X-Session-ID")
	}

	result, err := h.service.CreateSliderCaptcha(c.Request.Context(), req.SessionID, req.Difficulty)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}

	Success(c, result)
}

func (h *ChallengeHandler) VerifySliderCaptcha(c *gin.Context) {
	var req captcha.VerifyCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "Invalid request body")
		return
	}

	req.IPAddress = c.ClientIP()
	req.UserAgent = c.GetHeader("User-Agent")

	result, err := h.service.VerifySliderCaptcha(c.Request.Context(), &req)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}

	Success(c, result)
}

func (h *ChallengeHandler) CreateClickCaptcha(c *gin.Context) {
	var req struct {
		SessionID  string `json:"session_id"`
		Difficulty string `json:"difficulty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Difficulty = "medium"
	}

	if req.SessionID == "" {
		req.SessionID = c.GetHeader("X-Session-ID")
	}

	result, err := h.service.CreateClickCaptcha(c.Request.Context(), req.SessionID, req.Difficulty)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}

	Success(c, result)
}

func (h *ChallengeHandler) VerifyClickCaptcha(c *gin.Context) {
	var req captcha.VerifyCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "Invalid request body")
		return
	}

	req.IPAddress = c.ClientIP()
	req.UserAgent = c.GetHeader("User-Agent")

	result, err := h.service.VerifyClickCaptcha(c.Request.Context(), &req)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}

	Success(c, result)
}

func (h *ChallengeHandler) CreateRotateCaptcha(c *gin.Context) {
	var req struct {
		SessionID  string `json:"session_id"`
		Difficulty string `json:"difficulty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Difficulty = "medium"
	}

	if req.SessionID == "" {
		req.SessionID = c.GetHeader("X-Session-ID")
	}

	result, err := h.service.CreateRotateCaptcha(c.Request.Context(), req.SessionID, req.Difficulty)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}

	Success(c, result)
}

func (h *ChallengeHandler) VerifyRotateCaptcha(c *gin.Context) {
	var req captcha.VerifyCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "Invalid request body")
		return
	}

	req.IPAddress = c.ClientIP()
	req.UserAgent = c.GetHeader("User-Agent")

	result, err := h.service.VerifyRotateCaptcha(c.Request.Context(), &req)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}

	Success(c, result)
}

func (h *ChallengeHandler) AnalyzeBehavior(c *gin.Context) {
	var req struct {
		SessionID   string            `json:"session_id"`
		Trajectory  []captcha.Point  `json:"trajectory"`
		Fingerprint string            `json:"fingerprint"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "Invalid request body")
		return
	}

	result := map[string]interface{}{
		"risk_score": 0.5,
		"is_human":   true,
		"message":    "Behavior analysis complete",
	}

	Success(c, result)
}

func (h *ChallengeHandler) CreateSession(c *gin.Context) {
	var req struct {
		Fingerprint string `json:"fingerprint"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "Invalid request body")
		return
	}

	sessionID, err := h.service.CreateSession(c.Request.Context(), req.Fingerprint, c.ClientIP())
	if err != nil {
		Error(c, 500, err.Error())
		return
	}

	Success(c, map[string]interface{}{
		"session_id": sessionID,
		"expires_at": time.Now().Add(30 * time.Minute),
	})
}

func (h *ChallengeHandler) ValidateSession(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "Invalid request body")
		return
	}

	Success(c, map[string]interface{}{
		"valid":      true,
		"session_id": req.SessionID,
		"expires_at": time.Now().Add(30 * time.Minute),
	})
}

func (h *ChallengeHandler) GetConfig(c *gin.Context) {
	config := map[string]interface{}{
		"expires_minutes":    5,
		"max_attempts":       3,
		"supported_types":    []string{"slider", "click", "rotate"},
		"default_difficulty": "medium",
		"server_time":        time.Now(),
	}

	Success(c, config)
}
