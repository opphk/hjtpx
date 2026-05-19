package handler

import (
	"encoding/base64"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var voiceprintGeneratorService *captcha.VoiceprintGeneratorService
var voiceprintVerifierService *captcha.VoiceprintVerifierService
var hapticGeneratorService *captcha.HapticGeneratorService
var hapticVerifierService *captcha.HapticVerifierService

func InitBiometricCaptchaHandlers(
	voiceprintGen *captcha.VoiceprintGeneratorService,
	voiceprintVer *captcha.VoiceprintVerifierService,
	hapticGen *captcha.HapticGeneratorService,
	hapticVer *captcha.HapticVerifierService,
) {
	voiceprintGeneratorService = voiceprintGen
	voiceprintVerifierService = voiceprintVer
	hapticGeneratorService = hapticGen
	hapticVerifierService = hapticVer
}

type CreateVoiceprintCaptchaRequest struct {
	PatternType string `json:"pattern_type"`
	Complexity  int    `json:"complexity"`
}

func CreateVoiceprintCaptcha(c *gin.Context) {
	var req CreateVoiceprintCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = CreateVoiceprintCaptchaRequest{}
	}

	createReq := &captcha.VoiceprintCaptchaRequest{
		PatternType:  req.PatternType,
		Complexity:   req.Complexity,
		ClientIP:     c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := voiceprintGeneratorService.Generate(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成声纹验证码失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

type VerifyVoiceprintCaptchaRequest struct {
	SessionID string                      `json:"session_id" binding:"required"`
	VoiceData string                      `json:"voice_data"`
	Features  *captcha.VoiceFeatures       `json:"features"`
}

func VerifyVoiceprintCaptcha(c *gin.Context) {
	var req VerifyVoiceprintCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	if req.VoiceData != "" {
		audioBytes, err := base64.StdEncoding.DecodeString(req.VoiceData)
		if err == nil {
			features := voiceprintVerifierService.ExtractFeatures(audioBytes)
			req.Features = features
		}
	}

	verifyReq := &captcha.VoiceprintVerifyRequest{
		SessionID: req.SessionID,
		VoiceData: req.VoiceData,
		Features:  req.Features,
	}

	result, err := voiceprintVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

func GetVoiceprintCaptchaOptions(c *gin.Context) {
	options := map[string]interface{}{
		"pattern_types": []string{"sequence", "frequency", "amplitude"},
		"complexity_levels": []int{1, 2, 3, 4, 5},
		"default_complexity": 3,
		"max_attempts": 3,
		"expiry_seconds": 300,
	}
	response.Success(c, options)
}

type CreateHapticCaptchaRequest struct {
	PatternType string `json:"pattern_type"`
	Difficulty  string `json:"difficulty"`
	GridSize    int    `json:"grid_size"`
}

func CreateHapticCaptcha(c *gin.Context) {
	var req CreateHapticCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = CreateHapticCaptchaRequest{}
	}

	createReq := &captcha.HapticCaptchaRequest{
		PatternType: req.PatternType,
		Difficulty:  req.Difficulty,
		GridSize:    req.GridSize,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := hapticGeneratorService.Generate(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成触觉验证码失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

type VerifyHapticCaptchaRequest struct {
	SessionID string                `json:"session_id" binding:"required"`
	UserInput *captcha.HapticUserInput `json:"user_input"`
}

func VerifyHapticCaptcha(c *gin.Context) {
	var req VerifyHapticCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	if req.UserInput == nil {
		response.Fail(c, response.CodeInvalidParams, "用户输入数据不能为空")
		return
	}

	valid, msg := hapticVerifierService.ValidateInput(req.UserInput)
	if !valid {
		response.Fail(c, response.CodeInvalidParams, msg)
		return
	}

	verifyReq := &captcha.HapticVerifyRequest{
		SessionID: req.SessionID,
		UserInput: req.UserInput,
	}

	result, err := hapticVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

func GetHapticCaptchaOptions(c *gin.Context) {
	options := map[string]interface{}{
		"pattern_types": []string{"sequence", "grid", "direction", "pressure"},
		"difficulties":  []string{"easy", "medium", "hard"},
		"grid_sizes":    []int{3, 4, 5, 6},
		"default_difficulty": "medium",
		"default_grid_size":  3,
		"max_attempts":  3,
		"expiry_seconds": 300,
	}
	response.Success(c, options)
}

func AnalyzeHapticPattern(c *gin.Context) {
	var req VerifyHapticCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	if req.UserInput == nil {
		response.Fail(c, response.CodeInvalidParams, "用户输入数据不能为空")
		return
	}

	analysis := hapticVerifierService.AnalyzeHapticPattern(req.UserInput)
	response.Success(c, analysis)
}

type BiometricCaptchaStatusRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Type      string `json:"type" binding:"required"`
}

func GetBiometricCaptchaStatus(c *gin.Context) {
	sessionID := c.Query("session_id")
	captchaType := c.Query("type")

	if sessionID == "" || captchaType == "" {
		response.Fail(c, response.CodeInvalidParams, "缺少必要参数")
		return
	}

	var status string
	var score float64

	switch strings.ToLower(captchaType) {
	case "voiceprint":
		session, err := voiceprintVerifierService.GetSessionForStatus(c.Request.Context(), sessionID)
		if err != nil || session == nil {
			response.Fail(c, response.CodeInvalidParams, "Session not found")
			return
		}
		status = session.Status
		score = session.SimilarityScore

	case "haptic":
		session, err := hapticVerifierService.GetSessionForStatus(c.Request.Context(), sessionID)
		if err != nil || session == nil {
			response.Fail(c, response.CodeInvalidParams, "Session not found")
			return
		}
		status = session.Status
		score = session.MatchScore

	default:
		response.Fail(c, response.CodeInvalidParams, "Invalid captcha type")
		return
	}

	result := map[string]interface{}{
		"session_id": sessionID,
		"status":     status,
		"score":      score,
	}
	response.Success(c, result)
}
