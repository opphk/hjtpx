package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var sliderGeneratorService *captcha.GeneratorService
var sliderVerifierService *captcha.VerifierService

func InitSliderCaptchaHandler(gen *captcha.GeneratorService, ver *captcha.VerifierService) {
	sliderGeneratorService = gen
	sliderVerifierService = ver
}

// SliderCaptchaRequest 滑动验证码创建请求
type SliderCaptchaRequest struct {
	Width        int `json:"width"`         // 验证码图片宽度
	Height       int `json:"height"`        // 验证码图片高度
	SliderWidth  int `json:"slider_width"`  // 滑动块宽度
	SliderHeight int `json:"slider_height"` // 滑动块高度
}

// SliderVerifyRequest 滑动验证码验证请求
type SliderVerifyRequest struct {
	SessionID  string  `json:"session_id" binding:"required"` // 会话 ID
	PositionX  int     `json:"position_x" binding:"required"` // X 坐标
	PositionY  int     `json:"position_y" binding:"required"` // Y 坐标
	RiskScore  float64 `json:"risk_score"`                    // 风险评分
	TraceScore float64 `json:"trace_score"`                   // 轨迹评分
	EnvScore   float64 `json:"env_score"`                     // 环境评分
}

// CreateSliderCaptcha 创建滑动验证码
// @Summary 创建滑动验证码
// @Description 生成一个新的滑动验证码
// @Tags 验证码
// @Accept json
// @Produce json
// @Param body body SliderCaptchaRequest false "验证码参数"
// @Success 200 {object} map[string]interface{} "成功返回验证码数据"
// @Failure 500 {object} map[string]interface{} "生成失败"
// @Router /api/v1/captcha/create [post]
func CreateSliderCaptcha(c *gin.Context) {
	var req SliderCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = SliderCaptchaRequest{}
	}

	createReq := &captcha.CreateCaptchaRequest{
		Width:        req.Width,
		Height:       req.Height,
		SliderWidth:  req.SliderWidth,
		SliderHeight: req.SliderHeight,
		ClientIP:     c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Fingerprint:  c.GetHeader("X-Fingerprint"),
	}

	result, err := sliderGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成验证码失败")
		return
	}

	response.Success(c, result)
}

// VerifySliderCaptcha 验证滑动验证码
// @Summary 验证滑动验证码
// @Description 验证用户对滑动验证码的操作
// @Tags 验证码
// @Accept json
// @Produce json
// @Param body body SliderVerifyRequest true "验证请求"
// @Success 200 {object} map[string]interface{} "验证结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "验证失败"
// @Router /api/v1/captcha/verify-v2 [post]
func VerifySliderCaptcha(c *gin.Context) {
	var req SliderVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.VerifyRequest{
		SessionID:  req.SessionID,
		PositionX:  req.PositionX,
		PositionY:  req.PositionY,
		RiskScore:  req.RiskScore,
		TraceScore: req.TraceScore,
		EnvScore:   req.EnvScore,
	}

	result, err := sliderVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

// GetSliderCaptchaStatus 获取验证码会话状态
// @Summary 获取验证码会话状态
// @Description 通过 session_id 获取验证码会话的当前状态
// @Tags 验证码
// @Accept json
// @Produce json
// @Param session_id path string true "会话 ID"
// @Success 200 {object} map[string]interface{} "会话状态"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 404 {object} map[string]interface{} "会话不存在"
// @Router /api/v1/captcha/status/{session_id} [get]
func GetSliderCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	session, err := sliderVerifierService.GetSessionStatus(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}

// CheckSliderCaptchaValid 检查验证码有效性
// @Summary 检查验证码是否有效
// @Description 检查验证码会话是否仍然有效
// @Tags 验证码
// @Accept json
// @Produce json
// @Param session_id path string true "会话 ID"
// @Success 200 {object} map[string]interface{} "检查结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /api/v1/captcha/check/{session_id} [get]
func CheckSliderCaptchaValid(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	valid, message := sliderVerifierService.CheckSessionValid(c.Request.Context(), sessionID)

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"valid":   valid,
			"message": message,
		},
	})
}
