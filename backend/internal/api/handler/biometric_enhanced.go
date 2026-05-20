package handler

import (
	"github.com/gin-gonic/gin"
	github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var biometricCaptchaHandler *BiometricEnhancedHandler

// BiometricEnhancedHandler 生物识别增强版处理器
type BiometricEnhancedHandler struct {
	biometricCaptchaService *service.BiometricCaptchaService
}

// NewBiometricEnhancedHandler 创建新的生物识别增强版处理器
func NewBiometricEnhancedHandler() *BiometricEnhancedHandler {
	return &BiometricEnhancedHandler{
		biometricCaptchaService: service.NewBiometricCaptchaService(),
	}
}

// InitBiometricEnhancedHandler 初始化处理器
func InitBiometricEnhancedHandler() {
	biometricCaptchaHandler = NewBiometricEnhancedHandler()
}

// GenerateBiometricCaptchaRequest 生成生物识别验证码请求
type GenerateBiometricCaptchaRequest struct {
	ChallengeType string `json:"challenge_type,omitempty"` // "keyboard", "mouse", "multimodal"
}

// GenerateBiometricCaptcha 生成生物识别验证码
// @Summary 生成生物识别验证码
// @Description 生成基于生物特征的验证码挑战
// @Tags 生物识别验证码
// @Accept json
// @Produce json
// @Param body body GenerateBiometricCaptchaRequest true "生成请求"
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/captcha/biometric/generate [post]
func GenerateBiometricCaptcha(c *gin.Context) {
	var req GenerateBiometricCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有 body，尝试从 query 参数获取
		req.ChallengeType = c.Query("challenge_type")
	}

	if biometricCaptchaHandler == nil {
		InitBiometricEnhancedHandler()
	}

	challenge, err := biometricCaptchaHandler.biometricCaptchaService.GenerateBiometricCaptcha(req.ChallengeType)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"challenge": challenge,
		"message":   "Biometric captcha generated successfully",
	})
}

// VerifyBiometricCaptcha 验证生物识别验证码
// @Summary 验证生物识别验证码
// @Description 验证基于生物特征的验证码响应
// @Tags 生物识别验证码
// @Accept json
// @Produce json
// @Param body body service.BiometricCaptchaVerifyRequest true "验证请求"
// @Success 200 {object} map[string]interface{} "验证结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/captcha/biometric/verify [post]
func VerifyBiometricCaptcha(c *gin.Context) {
	var req service.BiometricCaptchaVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}

	if biometricCaptchaHandler == nil {
		InitBiometricEnhancedHandler()
	}

	result, err := biometricCaptchaHandler.biometricCaptchaService.VerifyBiometricCaptcha(&req)
	if err != nil {
		response.InternalServerError(c, "Failed to verify biometric captcha")
		return
	}

	response.Success(c, gin.H{
		"result":  result,
		"message": "Biometric captcha verification completed",
	})
}
