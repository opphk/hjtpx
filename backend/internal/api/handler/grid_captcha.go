package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var gridGeneratorService *captcha.GridGeneratorService
var gridVerifierService *captcha.GridVerifierService

func InitGridCaptchaHandler(
	gen *captcha.GridGeneratorService,
	ver *captcha.GridVerifierService,
) {
	gridGeneratorService = gen
	gridVerifierService = ver
}

type GridCaptchaRequest struct {
	GridSize    int    `json:"grid_size"`
	TargetCount int    `json:"target_count"`
	Difficulty  string `json:"difficulty"`
	IconType    string `json:"icon_type"`
}

type GridVerifyRequest struct {
	SessionID     string                `json:"session_id" binding:"required"`
	SelectedOrder []int                 `json:"selected_order" binding:"required"`
	TimeSpent     int64                 `json:"time_spent"`
	ClickPattern  []captcha.ClickPoint  `json:"click_pattern"`
	RiskScore     float64               `json:"risk_score"`
}

func CreateGridCaptcha(c *gin.Context) {
	var req GridCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = GridCaptchaRequest{}
	}

	createReq := &captcha.CreateGridRequest{
		GridSize:    req.GridSize,
		TargetCount: req.TargetCount,
		Difficulty:  req.Difficulty,
		IconType:    req.IconType,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := gridGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成九宫格验证码失败")
		return
	}

	response.Success(c, result)
}

func VerifyGridCaptcha(c *gin.Context) {
	var req GridVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.VerifyGridRequest{
		SessionID:     req.SessionID,
		SelectedOrder: req.SelectedOrder,
		TimeSpent:     req.TimeSpent,
		ClickPattern:  req.ClickPattern,
		RiskScore:     req.RiskScore,
	}

	result, err := gridVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func GetGridCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	session, err := gridVerifierService.GetSessionStatus(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}

func CheckGridCaptchaValid(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	valid, message := gridVerifierService.CheckSessionValid(c.Request.Context(), sessionID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"valid":   valid,
			"message": message,
		},
	})
}
