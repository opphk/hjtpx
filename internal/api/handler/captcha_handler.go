package handler

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"hjtpx/internal/models"
	"hjtpx/internal/repository"
	"hjtpx/internal/utils"

	"github.com/gin-gonic/gin"
)

type CaptchaHandler struct {
	captchaRepo *repository.CaptchaRepository
}

func NewCaptchaHandler(captchaRepo *repository.CaptchaRepository) *CaptchaHandler {
	return &CaptchaHandler{
		captchaRepo: captchaRepo,
	}
}

func (h *CaptchaHandler) Create(c *gin.Context) {
	var req models.CreateCaptchaRequest
	if !utils.ValidateParams(c, &req) {
		return
	}

	if req.Type != models.CaptchaTypeImage && req.Type != models.CaptchaTypeVideo && req.Type != models.CaptchaTypeAudio {
		utils.BadRequest(c, "Invalid captcha type")
		return
	}

	token := generateToken(32)
	expiresAt := time.Now().Add(5 * time.Minute)

	captcha := &models.Captcha{
		Token:     token,
		Challenge: generateChallenge(),
		Type:      req.Type,
		Status:    models.CaptchaStatusPending,
		ExpiresAt: expiresAt,
		UserID:    req.UserID,
		AppID:     req.AppID,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Metadata:  req.Metadata,
		MaxVerify: 3,
	}

	if err := h.captchaRepo.Create(captcha); err != nil {
		utils.InternalError(c, "Failed to create captcha")
		return
	}

	response := models.CaptchaResponse{
		Token:     token,
		Type:      captcha.Type,
		ExpiresAt: expiresAt.Unix(),
		ImageURL:  "/api/v1/captcha/image/" + token,
	}

	utils.Created(c, response)
}

func (h *CaptchaHandler) Verify(c *gin.Context) {
	var req models.VerifyCaptchaRequest
	if !utils.ValidateParams(c, &req) {
		return
	}

	captcha, err := h.captchaRepo.FindByToken(req.Token)
	if err != nil {
		utils.NotFound(c, "Captcha not found")
		return
	}

	if captcha.IsExpired() {
		utils.BadRequest(c, "Captcha expired")
		return
	}

	if captcha.Status != models.CaptchaStatusPending {
		utils.BadRequest(c, "Captcha already used")
		return
	}

	if captcha.VerifyCount >= captcha.MaxVerify {
		utils.BadRequest(c, "Maximum verification attempts exceeded")
		return
	}

	captcha.VerifyCount++

	if req.Challenge == captcha.Challenge {
		captcha.Status = models.CaptchaStatusVerified
		if err := h.captchaRepo.Update(captcha); err != nil {
			utils.InternalError(c, "Failed to update captcha")
			return
		}
		utils.Success(c, gin.H{
			"verified": true,
			"message":  "Captcha verified successfully",
		})
		return
	}

	if err := h.captchaRepo.Update(captcha); err != nil {
		utils.InternalError(c, "Failed to update captcha")
		return
	}

	if captcha.VerifyCount >= captcha.MaxVerify {
		captcha.Status = models.CaptchaStatusFailed
		h.captchaRepo.Update(captcha)
		utils.BadRequest(c, "Maximum verification attempts exceeded")
		return
	}

	utils.BadRequest(c, "Incorrect captcha challenge")
}

func (h *CaptchaHandler) GetStatus(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.BadRequest(c, "Token is required")
		return
	}

	captcha, err := h.captchaRepo.FindByToken(token)
	if err != nil {
		utils.NotFound(c, "Captcha not found")
		return
	}

	utils.Success(c, gin.H{
		"token":        captcha.Token,
		"status":       captcha.Status,
		"type":         captcha.Type,
		"expires_at":   captcha.ExpiresAt.Unix(),
		"verified":     captcha.IsVerified(),
		"expired":      captcha.IsExpired(),
		"verify_count": captcha.VerifyCount,
		"max_verify":   captcha.MaxVerify,
	})
}

func generateToken(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func generateChallenge() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
