package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var hybridGeneratorService *captcha.HybridGeneratorService
var hybridVerifierService *captcha.HybridVerifierService

func InitHybridCaptchaHandler(
	gen *captcha.HybridGeneratorService,
	ver *captcha.HybridVerifierService,
) {
	hybridGeneratorService = gen
	hybridVerifierService = ver
}

type HybridCaptchaRequest struct {
	Width        int `json:"width"`
	Height       int `json:"height"`
	SliderWidth  int `json:"slider_width"`
	SliderHeight int `json:"slider_height"`
	ClickCount   int `json:"click_count"`
}

type HybridSliderVerifyRequest struct {
	SessionID  string                  `json:"session_id" binding:"required"`
	PositionX  int                     `json:"position_x" binding:"required"`
	PositionY  int                     `json:"position_y" binding:"required"`
	Trajectory []captcha.TrajectoryData `json:"trajectory"`
	RiskScore  float64                  `json:"risk_score"`
}

type HybridClickVerifyRequest struct {
	SessionID  string `json:"session_id" binding:"required"`
	ClickX     int    `json:"click_x" binding:"required"`
	ClickY     int    `json:"click_y" binding:"required"`
	ClickIndex int    `json:"click_index" binding:"required"`
	ClickTime  int64  `json:"click_time"`
	RiskScore  float64 `json:"risk_score"`
}

func CreateHybridCaptcha(c *gin.Context) {
	var req HybridCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = HybridCaptchaRequest{}
	}

	createReq := &captcha.CreateHybridRequest{
		Width:        req.Width,
		Height:       req.Height,
		SliderWidth:  req.SliderWidth,
		SliderHeight: req.SliderHeight,
		ClickCount:   req.ClickCount,
		ClientIP:     c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := hybridGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成混合验证码失败")
		return
	}

	response.Success(c, result)
}

func VerifyHybridSliderCaptcha(c *gin.Context) {
	var req HybridSliderVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.VerifyHybridSliderRequest{
		SessionID:  req.SessionID,
		PositionX:  req.PositionX,
		PositionY:  req.PositionY,
		Trajectory: req.Trajectory,
		RiskScore:  req.RiskScore,
	}

	result, err := hybridVerifierService.VerifySlider(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func VerifyHybridClickCaptcha(c *gin.Context) {
	var req HybridClickVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.VerifyHybridClickRequest{
		SessionID:  req.SessionID,
		ClickX:     req.ClickX,
		ClickY:     req.ClickY,
		ClickIndex: req.ClickIndex,
		ClickTime:  req.ClickTime,
		RiskScore:  req.RiskScore,
	}

	result, err := hybridVerifierService.VerifyClick(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func GetHybridCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	session, err := hybridVerifierService.GetSessionStatus(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}

func CheckHybridCaptchaValid(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	valid, message := hybridVerifierService.CheckSessionValid(c.Request.Context(), sessionID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"valid":   valid,
			"message": message,
		},
	})
}

func GetHybridCaptchaData(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	data, err := hybridVerifierService.GetHybridData(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, data)
}
