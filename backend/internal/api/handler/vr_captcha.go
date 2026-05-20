package handler

import (
	"github.com/gin-gonic/gin"
	github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var vrGeneratorService *captcha.VRGeneratorService
var vrVerifierService *captcha.VRVerifierService

func InitVRCaptchaHandler(
	gen *captcha.VRGeneratorService,
	ver *captcha.VRVerifierService,
) {
	vrGeneratorService = gen
	vrVerifierService = ver
}

type VRCaptchaRequest struct {
	Mode       string `json:"mode"`
	Type       string `json:"type"`
	Difficulty string `json:"difficulty"`
}

type VRVerifyRequest struct {
	SessionID    string                      `json:"sessionID" binding:"required"`
	Interaction *captcha.VRInteractionData      `json:"interaction"`
	GestureData *captcha.VRHandGestureData  `json:"gestureData,omitempty"`
	EyeData     *captcha.VREyeTrackingData  `json:"eyeData,omitempty"`
	BehaviorData map[string]interface{}   `json:"behaviorData,omitempty"`
	TraceData    interface{}           `json:"traceData,omitempty"`
}

func CreateVRCaptcha(c *gin.Context) {
	var req VRCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = VRCaptchaRequest{}
	}

	createReq := &captcha.VRCaptchaRequest{
		Mode:        captcha.VRMode(req.Mode),
		Type:        captcha.VRCaptchaType(req.Type),
		Difficulty:  req.Difficulty,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := vrGeneratorService.Generate(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成VR验证码失败")
		return
	}

	response.Success(c, result)
}

func VerifyVRCaptcha(c *gin.Context) {
	var req VRVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.VRVerifyRequest{
		SessionID:    req.SessionID,
		Interaction: req.Interaction,
		GestureData: req.GestureData,
		EyeData:     req.EyeData,
		BehaviorData: req.BehaviorData,
		TraceData:    req.TraceData,
	}

	result, err := vrVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func GetVRCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	session, err := vrVerifierService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}
