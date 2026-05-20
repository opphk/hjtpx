package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
"github.com/hjtpx/hjtpx/internal/service"
)

var spatioTemporalCaptchaService *service.SpatioTemporalCaptchaService

// InitSpatioTemporalCaptchaHandler 初始化时空验证handler
func InitSpatioTemporalCaptchaHandler(service *service.SpatioTemporalCaptchaService) {
	spatioTemporalCaptchaService = service
}

// CreateSpatioTemporalCaptcha 创建时空验证码
func CreateSpatioTemporalCaptcha(c *gin.Context) {
	var req service.SpatioTemporalCaptchaRequest
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

	resp, err := spatioTemporalCaptchaService.Generate(&req)
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

// VerifySpatioTemporalCaptcha 验证时空验证码
func VerifySpatioTemporalCaptcha(c *gin.Context) {
	var req service.SpatioTemporalVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	resp, err := spatioTemporalCaptchaService.Verify(&req)
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

// GetSpatioTemporalCaptchaStatus 获取时空验证码状态
func GetSpatioTemporalCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "会话ID不能为空",
		})
		return
	}

	session, exists := spatioTemporalCaptchaService.GetSession(sessionID)
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
