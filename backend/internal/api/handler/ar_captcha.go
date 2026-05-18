package handler

import (
	"net/http"

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
	Difficulty  string `json:"difficulty"`
	ObjectType  string `json:"object_type"`
	GestureType string `json:"gesture_type"`
}

type ARVerifyRequest struct {
	SessionID     string                   `json:"session_id" binding:"required"`
	RotationX     float64                  `json:"rotation_x"`
	RotationY     float64                  `json:"rotation_y"`
	RotationZ     float64                  `json:"rotation_z"`
	Scale         float64                  `json:"scale"`
	GestureData   []captcha.ARGesturePoint `json:"gesture_data"`
	TouchPoints   []captcha.TouchPoint     `json:"touch_points"`
	DeviceMotion  *captcha.DeviceMotionData `json:"device_motion"`
	RiskScore     float64                  `json:"risk_score"`
}

func CreateARCaptcha(c *gin.Context) {
	var req ARCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = ARCaptchaRequest{}
	}

	createReq := &captcha.CreateARRequest{
		Difficulty:  req.Difficulty,
		ObjectType:  req.ObjectType,
		GestureType: req.GestureType,
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

	verifyReq := &captcha.VerifyARRequest{
		SessionID:    req.SessionID,
		RotationX:    req.RotationX,
		RotationY:    req.RotationY,
		RotationZ:    req.RotationZ,
		Scale:        req.Scale,
		GestureData:  req.GestureData,
		TouchPoints:  req.TouchPoints,
		DeviceMotion: req.DeviceMotion,
		RiskScore:    req.RiskScore,
	}

	result, err := arVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func GetARCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
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
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	valid, message := arVerifierService.CheckSessionValid(c.Request.Context(), sessionID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"valid":   valid,
			"message": message,
		},
	})
}
