package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type BiometricsHandler struct {
	biometricsService *service.BiometricsService
}

func NewBiometricsHandler() *BiometricsHandler {
	return &BiometricsHandler{
		biometricsService: service.NewBiometricsService(),
	}
}

func GetBiometricsHandler() *BiometricsHandler {
	return NewBiometricsHandler()
}

// RegisterBiometricProfileRequest 注册生物特征档案请求
type RegisterBiometricProfileRequest struct {
	UserID         string                  `json:"user_id" binding:"required"`
	KeyboardSample *service.KeyboardSample `json:"keyboard_sample,omitempty"`
	MouseSample    *service.MouseSample    `json:"mouse_sample,omitempty"`
}

// VerifyBiometricsRequest 生物特征验证请求
type VerifyBiometricsRequest struct {
	UserID         string                  `json:"user_id" binding:"required"`
	KeyboardSample *service.KeyboardSample `json:"keyboard_sample,omitempty"`
	MouseSample    *service.MouseSample    `json:"mouse_sample,omitempty"`
}

// RegisterBiometricProfile 注册生物特征档案
// @Summary 注册生物特征档案
// @Description 注册用户键盘和鼠标操作特征，用于后续生物特征验证
// @Tags 生物特征
// @Accept json
// @Produce json
// @Param body body RegisterBiometricProfileRequest true "生物特征注册请求"
// @Success 200 {object} map[string]interface{} "注册成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/biometrics/register [post]
func RegisterBiometricProfile(c *gin.Context) {
	var req RegisterBiometricProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	handler := GetBiometricsHandler()
	profile, err := handler.biometricsService.RegisterProfile(req.UserID, req.KeyboardSample, req.MouseSample)
	if err != nil {
		response.InternalServerError(c, "注册生物特征档案失败")
		return
	}

	response.Success(c, gin.H{
		"profile": profile,
		"message": "生物特征档案注册成功",
	})
}

// VerifyBiometrics 生物特征验证
// @Summary 生物特征验证
// @Description 使用已注册的生物特征进行身份验证
// @Tags 生物特征
// @Accept json
// @Produce json
// @Param body body VerifyBiometricsRequest true "生物特征验证请求"
// @Success 200 {object} map[string]interface{} "验证结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/biometrics/verify [post]
func VerifyBiometrics(c *gin.Context) {
	var req VerifyBiometricsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	handler := GetBiometricsHandler()
	result, err := handler.biometricsService.Verify(req.UserID, req.KeyboardSample, req.MouseSample)
	if err != nil {
		response.InternalServerError(c, "生物特征验证失败")
		return
	}

	response.Success(c, gin.H{
		"result":  result,
		"message": "生物特征验证完成",
	})
}

// GetBiometricProfile 获取生物特征档案
// @Summary 获取生物特征档案
// @Description 获取用户的生物特征档案信息
// @Tags 生物特征
// @Accept json
// @Produce json
// @Param user_id query string true "用户ID"
// @Success 200 {object} map[string]interface{} "档案信息"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/biometrics/profile [get]
func GetBiometricProfile(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		response.BadRequest(c, "用户ID不能为空")
		return
	}

	// 暂时返回简单的成功响应
	response.Success(c, gin.H{
		"user_id": userID,
		"message": "获取生物特征档案成功",
	})
}
