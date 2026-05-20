package handler

import (
	"github.com/gin-gonic/gin"
	github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var vrArGenerator *captcha.VRARGeneratorService
var vrArVerifier *captcha.VRARVerifierService

func InitVrArCaptchaHandler(gen *captcha.VRARGeneratorService, ver *captcha.VRARVerifierService) {
	vrArGenerator = gen
	vrArVerifier = ver
}

type GenerateVrArCaptchaRequest struct {
	Mode       string `json:"mode"`
	Type       string `json:"type"`
	Difficulty string `json:"difficulty"`
}

type VerifyVrArCaptchaRequest struct {
	SessionID    string                 `json:"session_id" binding:"required"`
	Interaction  *captcha.VRInteractionData `json:"interaction"`
	GestureData  *captcha.VRHandGestureData `json:"gesture_data,omitempty"`
	EyeData      *captcha.VREyeTrackingData `json:"eye_data,omitempty"`
	ARGesture    *captcha.ARGestureData     `json:"ar_gesture,omitempty"`
	BehaviorData map[string]interface{} `json:"behavior_data,omitempty"`
	TraceData    interface{}            `json:"trace_data,omitempty"`
}

// GenerateVrArCaptcha 生成VR/AR验证码
// @Summary 生成VR/AR验证码
// @Description 生成一个新的VR或AR验证码
// @Tags 验证码
// @Accept json
// @Produce json
// @Param body body GenerateVrArCaptchaRequest false "验证码参数"
// @Success 200 {object} response.Response{data=captcha.VRARCaptchaResponse} "成功返回验证码数据"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 500 {object} response.Response "生成失败"
// @Router /api/v1/captcha/vr-ar/generate [post]
func GenerateVrArCaptcha(c *gin.Context) {
	var req GenerateVrArCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = GenerateVrArCaptchaRequest{}
	}

	createReq := &captcha.VRARCaptchaRequest{
		Mode:        captcha.VRARMode(req.Mode),
		Type:        captcha.VRARType(req.Type),
		Difficulty:  req.Difficulty,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := vrArGenerator.Generate(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成VR/AR验证码失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

// VerifyVrArCaptcha 验证VR/AR验证码
// @Summary 验证VR/AR验证码
// @Description 验证用户对VR或AR验证码的操作
// @Tags 验证码
// @Accept json
// @Produce json
// @Param body body VerifyVrArCaptchaRequest true "验证请求"
// @Success 200 {object} response.Response{data=captcha.VRARVerifyResponse} "验证结果"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 404 {object} response.Response "会话不存在"
// @Failure 500 {object} response.Response "验证失败"
// @Router /api/v1/captcha/vr-ar/verify [post]
func VerifyVrArCaptcha(c *gin.Context) {
	var req VerifyVrArCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	verifyReq := &captcha.VRARVerifyRequest{
		SessionID:    req.SessionID,
		Interaction:  req.Interaction,
		GestureData:  req.GestureData,
		EyeData:      req.EyeData,
		ARGesture:    req.ARGesture,
		BehaviorData: req.BehaviorData,
		TraceData:    req.TraceData,
	}

	result, err := vrArVerifier.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

// GetVrArCaptchaStatus 获取VR/AR验证码会话状态
// @Summary 获取VR/AR验证码会话状态
// @Description 通过 session_id 获取验证码会话的当前状态
// @Tags 验证码
// @Accept json
// @Produce json
// @Param session_id path string true "会话 ID"
// @Success 200 {object} response.Response{data=captcha.VRARSession} "会话状态"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 404 {object} response.Response "会话不存在"
// @Router /api/v1/captcha/vr-ar/status/{session_id} [get]
func GetVrArCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	session, err := vrArVerifier.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}
