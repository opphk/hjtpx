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

type SliderCaptchaRequest struct {
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	SliderWidth  int    `json:"slider_width"`
	SliderHeight int    `json:"slider_height"`
}

type SliderVerifyRequest struct {
	SessionID  string  `json:"session_id" binding:"required"`
	PositionX  int     `json:"position_x" binding:"required"`
	PositionY  int     `json:"position_y" binding:"required"`
	RiskScore  float64 `json:"risk_score"`
	TraceScore float64 `json:"trace_score"`
	EnvScore   float64 `json:"env_score"`
}

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
