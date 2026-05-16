package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

// GenerateGestureCaptcha 生成手势验证码
func GenerateGestureCaptcha(c *gin.Context) {
	// 模拟生成手势验证码
	response.Success(c, gin.H{
		"id":      "gesture-123",
		"pattern": "1-2-3-5-7",
		"hint":    "Connect the dots in order",
	})
}

// VerifyGestureCaptcha 验证手势验证码
func VerifyGestureCaptcha(c *gin.Context) {
	var req struct {
		ID      string `json:"id" binding:"required"`
		Pattern string `json:"pattern" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	// 模拟验证
	success := req.Pattern == "1-2-3-5-7"
	response.Success(c, gin.H{
		"success": success,
		"message": func() string {
			if success {
				return "Verification successful"
			}
			return "Verification failed"
		}(),
	})
}
