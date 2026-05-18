package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

// VerifyGestureCaptchaRequest 手势验证码验证请求
// @Description 手势验证码验证请求参数
type VerifyGestureCaptchaRequest struct {
	ID      string `json:"id" binding:"required"`       // 验证码ID
	Pattern string `json:"pattern" binding:"required"` // 手势模式
}

// GenerateGestureCaptcha 生成手势验证码
// @Summary 生成手势验证码
// @Description 生成一个新的手势点连验证码
// @Tags 验证码
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "手势验证码数据"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/captcha/gesture [get]
func GenerateGestureCaptcha(c *gin.Context) {
	// 模拟生成手势验证码
	response.Success(c, gin.H{
		"id":      "gesture-123",
		"pattern": "1-2-3-5-7",
		"hint":    "Connect the dots in order",
	})
}

// VerifyGestureCaptcha 验证手势验证码
// @Summary 验证手势验证码
// @Description 验证用户绘制的手势是否正确
// @Tags 验证码
// @Accept json
// @Produce json
// @Param body body VerifyGestureCaptchaRequest true "手势验证请求"
// @Success 200 {object} map[string]interface{} "验证结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/captcha/gesture/verify [post]
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
