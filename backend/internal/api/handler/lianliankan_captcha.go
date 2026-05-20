package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var lianLianKanGeneratorService *captcha.LianLianKanGeneratorService
var lianLianKanVerifierService *captcha.LianLianKanVerifierService

func InitLianLianKanCaptchaHandler(
	gen *captcha.LianLianKanGeneratorService,
	ver *captcha.LianLianKanVerifierService,
) {
	lianLianKanGeneratorService = gen
	lianLianKanVerifierService = ver
}

type LianLianKanCaptchaRequest struct {
	Width     int `json:"width"`
	Height    int `json:"height"`
	TileTypes int `json:"tile_types"`
}

type LianLianKanVerifyRequest struct {
	SessionID string                    `json:"session_id" binding:"required"`
	Board     *captcha.LianLianKanBoard `json:"board" binding:"required"`
	Pairs     []captcha.LianLianKanPair `json:"pairs" binding:"required"`
	RiskScore float64                   `json:"risk_score"`
}

func CreateLianLianKanCaptcha(c *gin.Context) {
	var req LianLianKanCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = LianLianKanCaptchaRequest{}
	}

	createReq := &captcha.CreateLianLianKanRequest{
		Width:       req.Width,
		Height:      req.Height,
		TileTypes:   req.TileTypes,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := lianLianKanGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成验证码失败")
		return
	}

	response.Success(c, result)
}

func VerifyLianLianKanCaptcha(c *gin.Context) {
	var req LianLianKanVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.VerifyLianLianKanRequest{
		SessionID: req.SessionID,
		Board:     req.Board,
		Pairs:     req.Pairs,
		RiskScore: req.RiskScore,
	}

	result, err := lianLianKanVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func GetLianLianKanCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	session, err := lianLianKanVerifierService.GetSessionStatus(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}

func CheckLianLianKanCaptchaValid(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	valid, message := lianLianKanVerifierService.CheckSessionValid(c.Request.Context(), sessionID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"valid":   valid,
			"message": message,
		},
	})
}
