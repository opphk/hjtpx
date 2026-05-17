package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var memoryCardsGeneratorService *captcha.MemoryCardsGeneratorService
var memoryCardsVerifierService *captcha.MemoryCardsVerifierService

func InitMemoryCardsCaptchaHandler(
	gen *captcha.MemoryCardsGeneratorService,
	ver *captcha.MemoryCardsVerifierService,
) {
	memoryCardsGeneratorService = gen
	memoryCardsVerifierService = ver
}

type MemoryCardsCaptchaRequest struct {
	Width     int `json:"width"`
	Height    int `json:"height"`
	CardTypes int `json:"card_types"`
	ShowTime  int `json:"show_time"`
}

type MemoryCardsVerifyRequest struct {
	SessionID string                  `json:"session_id" binding:"required"`
	Board     *captcha.MemoryCardsBoard `json:"board" binding:"required"`
	Matches   []captcha.MemoryCardsMatch `json:"matches" binding:"required"`
	TimeUsed  int                     `json:"time_used"`
	RiskScore float64                 `json:"risk_score"`
}

func CreateMemoryCardsCaptcha(c *gin.Context) {
	var req MemoryCardsCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = MemoryCardsCaptchaRequest{}
	}

	createReq := &captcha.CreateMemoryCardsRequest{
		Width:       req.Width,
		Height:      req.Height,
		CardTypes:   req.CardTypes,
		ShowTime:    req.ShowTime,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := memoryCardsGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成验证码失败")
		return
	}

	response.Success(c, result)
}

func VerifyMemoryCardsCaptcha(c *gin.Context) {
	var req MemoryCardsVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.VerifyMemoryCardsRequest{
		SessionID: req.SessionID,
		Board:     req.Board,
		Matches:   req.Matches,
		TimeUsed:  req.TimeUsed,
		RiskScore: req.RiskScore,
	}

	result, err := memoryCardsVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func GetMemoryCardsCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	session, err := memoryCardsVerifierService.GetSessionStatus(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}

func CheckMemoryCardsCaptchaValid(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	valid, message := memoryCardsVerifierService.CheckSessionValid(c.Request.Context(), sessionID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"valid":   valid,
			"message": message,
		},
	})
}
