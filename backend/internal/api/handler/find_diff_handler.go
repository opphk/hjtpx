package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var findDiffGeneratorService *captcha.FindDiffGeneratorService
var findDiffVerifierService *captcha.FindDiffVerifierService

func InitFindDiffHandler(
	gen *captcha.FindDiffGeneratorService,
	ver *captcha.FindDiffVerifierService,
) {
	findDiffGeneratorService = gen
	findDiffVerifierService = ver
}

type FindDiffCaptchaRequest struct {
	Width     int `json:"width"`
	Height    int `json:"height"`
	DiffCount int `json:"diff_count"`
}

type FindDiffVerifyRequest struct {
	SessionID   string                  `json:"session_id" binding:"required"`
	Differences []captcha.FindDifference `json:"differences" binding:"required"`
	RiskScore   float64                 `json:"risk_score"`
}

func CreateFindDiffCaptcha(c *gin.Context) {
	var req FindDiffCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = FindDiffCaptchaRequest{}
	}

	createReq := &captcha.CreateFindDiffRequest{
		Width:       req.Width,
		Height:      req.Height,
		DiffCount:   req.DiffCount,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := findDiffGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成验证码失败")
		return
	}

	response.Success(c, result)
}

func VerifyFindDiffCaptcha(c *gin.Context) {
	var req FindDiffVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.VerifyFindDiffRequest{
		SessionID:   req.SessionID,
		Differences: req.Differences,
		RiskScore:   req.RiskScore,
	}

	result, err := findDiffVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func GetFindDiffCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	session, err := findDiffVerifierService.GetSessionStatus(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}

func CheckFindDiffCaptchaValid(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	valid, message := findDiffVerifierService.CheckSessionValid(c.Request.Context(), sessionID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"valid":   valid,
			"message": message,
		},
	})
}
