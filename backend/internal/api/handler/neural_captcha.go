package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	github.com/hjtpx/hjtpx/internal/service"
)

var neuralCaptchaService *service.NeuralCaptchaService

// InitNeuralCaptchaHandler 初始化脑神经验证handler
func InitNeuralCaptchaHandler(service *service.NeuralCaptchaService) {
	neuralCaptchaService = service
}

// CreateNeuralCaptcha 创建脑神经验证码
func CreateNeuralCaptcha(c *gin.Context) {
	var req service.NeuralCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 获取客户端信息
	if req.ClientIP == "" {
		req.ClientIP = c.ClientIP()
	}
	if req.UserAgent == "" {
		req.UserAgent = c.GetHeader("User-Agent")
	}

	resp, err := neuralCaptchaService.Generate(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "生成验证码失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resp,
	})
}

// VerifyNeuralCaptcha 验证脑神经验证码
func VerifyNeuralCaptcha(c *gin.Context) {
	var req service.NeuralVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	resp, err := neuralCaptchaService.Verify(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "验证失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resp,
	})
}

// GetNeuralCaptchaStatus 获取脑神经验证码状态
func GetNeuralCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "会话ID不能为空",
		})
		return
	}

	session, exists := neuralCaptchaService.GetSession(sessionID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "会话不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"session_id":  session.SessionID,
			"status":      session.Status,
			"verify_count": session.VerifyCount,
			"created_at":  session.CreatedAt,
			"expired_at":  session.ExpiredAt,
		},
	})
}
