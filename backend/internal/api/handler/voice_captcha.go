package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var voiceGeneratorService *captcha.VoiceGeneratorService
var voiceVerifierService *captcha.VoiceVerifierService

func InitVoiceCaptchaHandler(gen *captcha.VoiceGeneratorService, ver *captcha.VoiceVerifierService) {
	voiceGeneratorService = gen
	voiceVerifierService = ver
}

type CreateVoiceCaptchaRequest struct {
	Language string `json:"language"` // "zh-CN" or "en-US"
	Length   int    `json:"length"`   // number of digits, default 4
}

func CreateVoiceCaptcha(c *gin.Context) {
	var req CreateVoiceCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = CreateVoiceCaptchaRequest{}
	}

	createReq := &captcha.VoiceCaptchaRequest{
		Language:    req.Language,
		Length:      req.Length,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := voiceGeneratorService.Generate(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成语音验证码失败")
		return
	}

	response.Success(c, result)
}

func VerifyVoiceCaptcha(c *gin.Context) {
	var req captcha.VoiceVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	result, err := voiceVerifierService.Verify(c.Request.Context(), &req)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}
