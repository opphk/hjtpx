package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var socialGeneratorService *captcha.SocialGeneratorService
var socialVerifierService *captcha.SocialVerifierService

func InitSocialCaptchaHandler(
	gen *captcha.SocialGeneratorService,
	ver *captcha.SocialVerifierService,
) {
	socialGeneratorService = gen
	socialVerifierService = ver
}

type SocialCaptchaRequest struct {
	Difficulty   string `json:"difficulty"`
	BehaviorType string `json:"behavior_type"`
	PatternCount  int    `json:"pattern_count"`
}

type SocialVerifyRequest struct {
	SessionID  string               `json:"session_id" binding:"required"`
	TraceData  []captcha.TracePoint `json:"trace_data" binding:"required"`
	PatternType string              `json:"pattern_type"`
	StartTime  int64                `json:"start_time"`
	EndTime    int64                `json:"end_time"`
	TouchPoints []captcha.TouchPoint `json:"touch_points"`
	MouseTrail  []captcha.TracePoint `json:"mouse_trail"`
	RiskScore   float64             `json:"risk_score"`
}

func CreateSocialCaptcha(c *gin.Context) {
	var req SocialCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = SocialCaptchaRequest{}
	}

	createReq := &captcha.CreateSocialRequest{
		Difficulty:   req.Difficulty,
		BehaviorType: req.BehaviorType,
		PatternCount: req.PatternCount,
		ClientIP:     c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Fingerprint:  c.GetHeader("X-Fingerprint"),
	}

	result, err := socialGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成社交行为验证码失败")
		return
	}

	response.Success(c, result)
}

func VerifySocialCaptcha(c *gin.Context) {
	var req SocialVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.VerifySocialRequest{
		SessionID:   req.SessionID,
		TraceData:   req.TraceData,
		PatternType: req.PatternType,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		TouchPoints: req.TouchPoints,
		MouseTrail:  req.MouseTrail,
		RiskScore:   req.RiskScore,
	}

	result, err := socialVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func GetSocialCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	session, err := socialVerifierService.GetSessionStatus(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}

func CheckSocialCaptchaValid(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	valid, message := socialVerifierService.CheckSessionValid(c.Request.Context(), sessionID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"valid":   valid,
			"message": message,
		},
	})
}
