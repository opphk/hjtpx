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

// CreateVoiceCaptchaRequest 语音验证码创建请求
// @Description 语音验证码创建请求参数
type CreateVoiceCaptchaRequest struct {
	Language string `json:"language"` // 语言: "zh-CN" 或 "en-US"
	Length   int    `json:"length"`   // 验证码位数，默认4位
}

// CreateVoiceCaptcha 创建语音验证码
// @Summary 创建语音验证码
// @Description 生成一个新的语音验证码，返回语音文件URL
// @Tags 验证码
// @Accept json
// @Produce json
// @Param body body CreateVoiceCaptchaRequest false "语音验证码创建请求"
// @Success 200 {object} map[string]interface{} "语音验证码数据"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/captcha/voice/create [post]
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

// VerifyVoiceCaptcha 验证语音验证码
// @Summary 验证语音验证码
// @Description 验证用户输入的语音验证码是否正确
// @Tags 验证码
// @Accept json
// @Produce json
// @Param body body captcha.VoiceVerifyRequest true "语音验证码验证请求"
// @Success 200 {object} map[string]interface{} "验证结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/captcha/voice/verify [post]
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
