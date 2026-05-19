package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
)

var videoGeneratorService = captcha.NewVideoGeneratorService(nil, nil)
var videoVerifierService = captcha.NewVideoVerifierService()

type CreateVideoCaptchaRequest struct {
	Type     string `json:"type"`     // content, action, sequence
	Duration int    `json:"duration"` // 视频时长（秒）
	Language string `json:"language"` // zh-CN or en-US
}

type VerifyVideoCaptchaRequest struct {
	SessionID    string   `json:"session_id" binding:"required"`
	Answer       string   `json:"answer"`
	ActionResult []int    `json:"action_result"`
	Sequence     []string `json:"sequence"`
}

func CreateVideoCaptcha(c *gin.Context) {
	var req CreateVideoCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
			"msg":   err.Error(),
		})
		return
	}

	videoType := captcha.VideoCaptchaType(req.Type)
	if videoType == "" {
		videoType = captcha.VideoTypeContent
	}

	request := &captcha.VideoCaptchaRequest{
		Type:         videoType,
		Duration:     req.Duration,
		Language:     req.Language,
		ClientIP:     c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Fingerprint:  c.GetHeader("X-Fingerprint"),
	}

	response, err := videoGeneratorService.Generate(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate video captcha",
			"msg":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"session_id":    response.SessionID,
		"video_data":    response.VideoData,
		"video_type":    response.VideoType,
		"question":      response.Question,
		"options":       response.Options,
		"action_hint":   response.ActionHint,
		"sequence_count": response.SequenceCount,
		"expires_in":    response.ExpiresIn,
		"expires_at":    response.ExpiresAt,
		"language":      response.Language,
	})
}

func VerifyVideoCaptcha(c *gin.Context) {
	var req VerifyVideoCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
			"msg":   err.Error(),
		})
		return
	}

	request := &captcha.VideoVerifyRequest{
		SessionID:    req.SessionID,
		Answer:       req.Answer,
		ActionResult: req.ActionResult,
		Sequence:     req.Sequence,
	}

	result, err := videoVerifierService.Verify(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to verify video captcha",
			"msg":     err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       result.Success,
		"message":       result.Message,
		"score":         result.Score,
		"session_id":    result.SessionID,
		"attempts_left": result.AttemptsLeft,
	})
}

func GetVideoCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Session ID is required",
		})
		return
	}

	session, err := videoVerifierService.GetSessionStatus(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Session not found",
			"msg":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"session_id":    session.SessionID,
		"type":          session.Type,
		"status":        session.Status,
		"verify_count":  session.VerifyCount,
		"max_attempts":  session.MaxAttempts,
		"question":      session.Question,
		"expired_at":    session.ExpiredAt.Unix(),
	})
}

func CheckVideoCaptchaValid(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Session ID is required",
		})
		return
	}

	valid, message := videoVerifierService.CheckSessionValid(c.Request.Context(), sessionID)

	c.JSON(http.StatusOK, gin.H{
		"valid":   valid,
		"message": message,
	})
}