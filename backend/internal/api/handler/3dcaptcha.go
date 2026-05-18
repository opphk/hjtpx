package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var threeDGeneratorService *captcha.ThreeDGeneratorService
var threeDVerifierService *captcha.ThreeDVerifierService

func InitThreeDCaptchaHandler(
	gen *captcha.ThreeDGeneratorService,
	ver *captcha.ThreeDVerifierService,
) {
	threeDGeneratorService = gen
	threeDVerifierService = ver
}

type ThreeDCaptchaRequest struct {
	Difficulty string `json:"difficulty"`
}

type ThreeDVerifyRequest struct {
	SessionID string                `json:"sessionID" binding:"required"`
	Puzzle    *captcha.ThreeDPuzzle `json:"puzzle" binding:"required"`
	RiskScore float64               `json:"riskScore"`
}

func CreateThreeDCaptcha(c *gin.Context) {
	var req ThreeDCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = ThreeDCaptchaRequest{}
	}

	createReq := &captcha.CreateThreeDRequest{
		Difficulty:  req.Difficulty,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := threeDGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成验证码失败")
		return
	}

	response.Success(c, result)
}

func VerifyThreeDCaptcha(c *gin.Context) {
	var req ThreeDVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.VerifyThreeDRequest{
		SessionID: req.SessionID,
		Puzzle:    req.Puzzle,
		RiskScore: req.RiskScore,
	}

	result, err := threeDVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func GetThreeDCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "sessionID不能为空")
		return
	}

	session, err := threeDVerifierService.GetSessionStatus(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}

func CheckThreeDCaptchaValid(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "sessionID不能为空")
		return
	}

	valid, message := threeDVerifierService.CheckSessionValid(c.Request.Context(), sessionID)

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"valid":   valid,
			"message": message,
		},
	})
}
