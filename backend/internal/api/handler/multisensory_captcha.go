package handler

import (
	"sync"

	"github.com/gin-gonic/gin"
	github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var (
	multisensoryGeneratorService *captcha.MultisensoryGeneratorService
	multisensoryVerifierService  *captcha.MultisensoryVerifierService
	multisensoryInitOnce         sync.Once
)

func initMultisensoryServices() {
	multisensoryInitOnce.Do(func() {
		multisensoryGeneratorService = captcha.NewMultisensoryGeneratorServiceSimple()
		multisensoryVerifierService = captcha.NewMultisensoryVerifierServiceSimple()
	})
}

type MultisensoryCaptchaCreateRequest struct {
	Types       []string `json:"types"`
	VisualType  string   `json:"visual_type"`
	Language    string   `json:"language"`
}

type MultisensoryCaptchaVerifyRequest struct {
	SessionID  string            `json:"session_id" binding:"required"`
	Answers    map[string]string `json:"answers"`
	RequireAll bool              `json:"require_all"`
}

func CreateMultisensoryCaptcha(c *gin.Context) {
	initMultisensoryServices()

	var req MultisensoryCaptchaCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = MultisensoryCaptchaCreateRequest{}
	}

	createReq := &captcha.MultisensoryCaptchaRequest{
		Types:       req.Types,
		VisualType:  req.VisualType,
		Language:    req.Language,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := multisensoryGeneratorService.Generate(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成多感官验证码失败")
		return
	}

	response.Success(c, result)
}

func VerifyMultisensoryCaptcha(c *gin.Context) {
	initMultisensoryServices()

	var req MultisensoryCaptchaVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.MultisensoryVerifyRequest{
		SessionID:  req.SessionID,
		Answers:    req.Answers,
		RequireAll: req.RequireAll,
	}

	result, err := multisensoryVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func GetMultisensoryCaptchaStatus(c *gin.Context) {
	initMultisensoryServices()

	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	session, err := multisensoryVerifierService.GetSessionStatus(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}
