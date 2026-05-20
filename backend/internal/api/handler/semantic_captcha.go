package handler

import (
	"sync"

	"github.com/gin-gonic/gin"
	github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var (
	semanticGeneratorService *captcha.SemanticGeneratorService
	semanticVerifierService  *captcha.SemanticVerifierService
	comboGeneratorService    *captcha.ComboGeneratorService
	semanticInitOnce         sync.Once
)

func initSemanticServices() {
	semanticInitOnce.Do(func() {
		semanticGeneratorService = captcha.NewSemanticGeneratorServiceSimple()
		semanticVerifierService = captcha.NewSemanticVerifierServiceSimple()
		comboGeneratorService = captcha.NewComboGeneratorServiceSimple()
	})
}

type SemanticCaptchaRequest struct {
	Language   string `json:"language"`
	Difficulty string `json:"difficulty"`
}

type SemanticVerifyRequest struct {
	SessionID    string `json:"session_id" binding:"required"`
	Answer       string `json:"answer" binding:"required"`
	ReadTime     int64  `json:"read_time"`
	DecisionTime int64  `json:"decision_time"`
	TotalTime    int64  `json:"total_time"`
	IsMobile     bool   `json:"is_mobile"`
	ClickCount   int    `json:"click_count"`
}

type ComboCaptchaRequest struct {
	Types      []string `json:"types"`
	Strategy   string  `json:"strategy"`
	Difficulty string  `json:"difficulty"`
}

type ComboVerifyRequest struct {
	SessionID string                    `json:"session_id" binding:"required"`
	Answers   []ComboCaptchaAnswerItem `json:"answers" binding:"required"`
}

type ComboCaptchaAnswerItem struct {
	Type      string      `json:"type" binding:"required"`
	SessionID string      `json:"session_id" binding:"required"`
	Answer    interface{} `json:"answer" binding:"required"`
}

func CreateSemanticCaptcha(c *gin.Context) {
	initSemanticServices()
	
	var req SemanticCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = SemanticCaptchaRequest{}
	}

	if req.Language == "" {
		req.Language = "en"
	}
	if req.Difficulty == "" {
		req.Difficulty = "easy"
	}

	createReq := &captcha.CreateSemanticCaptchaRequest{
		Language:    req.Language,
		Difficulty:  req.Difficulty,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := semanticGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成语义验证码失败")
		return
	}

	response.Success(c, result)
}

func VerifySemanticCaptcha(c *gin.Context) {
	initSemanticServices()
	
	var req SemanticVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	behaviorData := captcha.SemanticBehaviorData{
		ReadTime:     req.ReadTime,
		DecisionTime: req.DecisionTime,
		TotalTime:    req.TotalTime,
		IsMobile:     req.IsMobile,
	}

	verifyReq := &captcha.VerifySemanticCaptchaRequest{
		SessionID:    req.SessionID,
		Answer:       req.Answer,
		BehaviorData: behaviorData,
	}

	result, err := semanticVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func GetSemanticCaptchaStatus(c *gin.Context) {
	initSemanticServices()
	
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	session, err := semanticGeneratorService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}

func CreateComboCaptcha(c *gin.Context) {
	initSemanticServices()
	
	var req ComboCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = ComboCaptchaRequest{}
	}

	if len(req.Types) == 0 {
		req.Types = []string{"semantic", "emoji"}
	}
	if req.Strategy == "" {
		req.Strategy = "all"
	}
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}

	createReq := &captcha.CreateComboCaptchaRequest{
		Types:      req.Types,
		Strategy:   req.Strategy,
		Difficulty: req.Difficulty,
		ClientIP:   c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
	}

	result, err := comboGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成组合验证码失败")
		return
	}

	response.Success(c, result)
}

func VerifyComboCaptcha(c *gin.Context) {
	initSemanticServices()
	
	var req ComboVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	answers := make([]captcha.ComboCaptchaAnswer, len(req.Answers))
	for i, ans := range req.Answers {
		answers[i] = captcha.ComboCaptchaAnswer{
			Type:      ans.Type,
			SessionID: ans.SessionID,
			Answer:    ans.Answer,
		}
	}

	verifyReq := &captcha.VerifyComboCaptchaRequest{
		SessionID: req.SessionID,
		Answers:   answers,
	}

	result, err := comboGeneratorService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func GetComboCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	session, err := comboGeneratorService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}
