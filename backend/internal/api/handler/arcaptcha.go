package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var arGeneratorService *captcha.ARGeneratorService
var arVerifierService *captcha.ARVerifierService

func InitARCaptchaHandler(
	gen *captcha.ARGeneratorService,
	ver *captcha.ARVerifierService,
) {
	arGeneratorService = gen
	arVerifierService = ver
}

type ARCaptchaRequest struct {
	Difficulty string `json:"difficulty"`
}

type ARVerifyRequest struct {
	SessionID      string              `json:"sessionID" binding:"required"`
	Scene          *captcha.ARScene    `json:"scene" binding:"required"`
	UserGesture    string              `json:"userGesture"`
	PlacedObjectID int                 `json:"placedObjectID"`
	FinalPosition  *captcha.ARPosition `json:"finalPosition"`
	RiskScore      float64             `json:"riskScore"`
}

func CreateARCaptcha(c *gin.Context) {
	var req ARCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = ARCaptchaRequest{}
	}

	createReq := &captcha.CreateARRequest{
		Difficulty:  req.Difficulty,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := arGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成AR验证码失败")
		return
	}

	response.Success(c, result)
}

func VerifyARCaptcha(c *gin.Context) {
	var req ARVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.ARVerifyRequest{
		SessionID:      req.SessionID,
		Scene:          req.Scene,
		UserGesture:    req.UserGesture,
		PlacedObjectID: req.PlacedObjectID,
		RiskScore:      req.RiskScore,
	}

	if req.FinalPosition != nil {
		verifyReq.FinalPosition = *req.FinalPosition
	}

	result, err := arVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func GetARCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "sessionID不能为空")
		return
	}

	session, err := arVerifierService.GetSessionStatus(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}

func CheckARCaptchaValid(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "sessionID不能为空")
		return
	}

	valid, message := arVerifierService.CheckSessionValid(c.Request.Context(), sessionID)

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"valid":   valid,
			"message": message,
		},
	})
}

func GetWebXRSupport(c *gin.Context) {
	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"supportsWebXR":           true,
			"supportsAR":              true,
			"supportsVR":              true,
			"requiredFeatures":        []string{"local-floor", "hit-test", "anchors"},
			"recommendedFeatures":     []string{"depth-sensing", "mesh"},
			"browserSupport":          "现代浏览器支持",
			"mobileSupport":           true,
			"desktopSupport":          false,
			"apiVersion":              "WebXR API Level 1",
			"capabilities":            []string{"手势识别", "空间定位", "3D物体追踪"},
		},
	})
}
