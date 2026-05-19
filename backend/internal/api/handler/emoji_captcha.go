package handler

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var (
	emojiGeneratorService *captcha.EmojiGeneratorService
	emojiVerifierService  *captcha.EmojiVerifierService
	emojiInitOnce         sync.Once
)

func initEmojiServices() {
	emojiInitOnce.Do(func() {
		emojiGeneratorService = captcha.NewEmojiGeneratorServiceSimple()
		emojiVerifierService = captcha.NewEmojiVerifierServiceSimple()
	})
}

func CreateEmojiCaptcha(c *gin.Context) {
	initEmojiServices()
	
	createReq := &captcha.CreateEmojiCaptchaRequest{
		ClientIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := emojiGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成表情验证码失败")
		return
	}

	response.Success(c, result)
}

func VerifyEmojiCaptcha(c *gin.Context) {
	initEmojiServices()
	
	var req struct {
		SessionID      string                  `json:"session_id" binding:"required"`
		SelectedEmojis []string                `json:"selected_emojis" binding:"required"`
		BehaviorData   captcha.EmojiBehaviorData `json:"behavior_data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.VerifyEmojiCaptchaRequest{
		SessionID:      req.SessionID,
		SelectedEmojis: req.SelectedEmojis,
		BehaviorData:   req.BehaviorData,
	}

	result, err := emojiVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}
