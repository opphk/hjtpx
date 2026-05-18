package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var semanticGeneratorService *captcha.SemanticGeneratorService
var semanticVerifierService *captcha.SemanticVerifierService

func InitSemanticCaptchaHandler(
	gen *captcha.SemanticGeneratorService,
	ver *captcha.SemanticVerifierService,
) {
	semanticGeneratorService = gen
	semanticVerifierService = ver
}

type SemanticCaptchaRequest struct {
	Difficulty   string `json:"difficulty"`
	Category     string `json:"category"`
	AnalysisType string `json:"analysis_type"`
	ImageCount   int    `json:"image_count"`
}

type SemanticVerifyRequest struct {
	SessionID       string   `json:"session_id" binding:"required"`
	Answer          string   `json:"answer" binding:"required"`
	AnswerIndex     int      `json:"answer_index"`
	ConfidenceScore float64  `json:"confidence_score"`
	ResponseTime    int64    `json:"response_time"`
	AnalysisMethod  string   `json:"analysis_method"`
	Keywords        []string `json:"keywords"`
	RiskScore       float64  `json:"risk_score"`
}

func CreateSemanticCaptcha(c *gin.Context) {
	var req SemanticCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = SemanticCaptchaRequest{}
	}

	createReq := &captcha.CreateSemanticRequest{
		Difficulty:    req.Difficulty,
		Category:      req.Category,
		AnalysisType:  req.AnalysisType,
		ImageCount:    req.ImageCount,
		ClientIP:      c.ClientIP(),
		UserAgent:     c.GetHeader("User-Agent"),
		Fingerprint:   c.GetHeader("X-Fingerprint"),
	}

	result, err := semanticGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成语义验证码失败")
		return
	}

	response.Success(c, result)
}

func VerifySemanticCaptcha(c *gin.Context) {
	var req SemanticVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.VerifySemanticRequest{
		SessionID:       req.SessionID,
		Answer:          req.Answer,
		AnswerIndex:     req.AnswerIndex,
		ConfidenceScore: req.ConfidenceScore,
		ResponseTime:    req.ResponseTime,
		AnalysisMethod:  req.AnalysisMethod,
		Keywords:        req.Keywords,
		RiskScore:       req.RiskScore,
	}

	result, err := semanticVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func GetSemanticCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	session, err := semanticVerifierService.GetSessionStatus(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}

func CheckSemanticCaptchaValid(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	valid, message := semanticVerifierService.CheckSessionValid(c.Request.Context(), sessionID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"valid":   valid,
			"message": message,
		},
	})
}
