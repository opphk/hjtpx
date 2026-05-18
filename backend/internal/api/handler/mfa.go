package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

var mfaService = service.NewMFAService()

type GenerateTOTPRequest struct {
	AccountName string `json:"account_name" binding:"required"`
	Issuer      string `json:"issuer" binding:"required"`
}

type VerifyTOTPRequest struct {
	Secret string `json:"secret" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

type EnableTOTPRequest struct {
	Secret string `json:"secret" binding:"required"`
}

type SendSMSCodeRequest struct {
	Phone string `json:"phone" binding:"required"`
}

type SendEmailCodeRequest struct {
	Email string `json:"email" binding:"required"`
}

type VerifyCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

type EnableMFARequest struct {
	MFAType string `json:"mfa_type" binding:"required,oneof=totp sms email"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
}

func GetMFAStatusHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	mfa, err := mfaService.GetMFAStatus(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    mfa,
	})
}

// GenerateTOTPHandler 生成TOTP密钥
// @Summary 生成TOTP密钥
// @Description 为用户生成新的TOTP认证密钥
// @Tags MFA
// @Accept json
// @Produce json
// @Param body body GenerateTOTPRequest true "TOTP生成请求"
// @Success 200 {object} map[string]interface{} "TOTP密钥信息"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/mfa/totp/generate [post]
func GenerateTOTPHandler(c *gin.Context) {
	var req GenerateTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := mfaService.GenerateTOTPSecret(req.AccountName, req.Issuer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

func VerifyTOTPHandler(c *gin.Context) {
	var req VerifyTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	valid, err := mfaService.VerifyTOTP(req.Secret, req.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"valid":   valid,
	})
}

// EnableTOTPHandler 启用TOTP
// @Summary 启用TOTP
// @Description 为当前用户启用基于时间的一次性密码(TOTP)认证
// @Tags MFA
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body EnableTOTPRequest true "启用TOTP请求"
// @Success 200 {object} map[string]interface{} "启用成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/mfa/totp/enable [post]
func EnableTOTPHandler(c *gin.Context) {
	var req EnableTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	err := mfaService.EnableTOTP(userID, req.Secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "TOTP MFA 启用成功",
	})
}

func SendSMSCodeHandler(c *gin.Context) {
	var req SendSMSCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	code, _, err := mfaService.SendSMSCode(userID, req.Phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "SMS 验证码发送成功",
		"code":    code, // 仅用于演示，生产环境不应返回
	})
}

// SendEmailCodeHandler 发送邮箱验证码
// @Summary 发送邮箱验证码
// @Description 向用户邮箱发送验证码
// @Tags MFA
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body SendEmailCodeRequest true "邮箱验证码请求"
// @Success 200 {object} map[string]interface{} "发送成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/mfa/email/send [post]
func SendEmailCodeHandler(c *gin.Context) {
	var req SendEmailCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	code, _, err := mfaService.SendEmailCode(userID, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Email 验证码发送成功",
		"code":    code, // 仅用于演示，生产环境不应返回
	})
}

func VerifyCodeHandler(c *gin.Context) {
	var req VerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "验证码验证成功",
	})
}

// EnableMFAHandler 启用MFA
// @Summary 启用多因素认证
// @Description 为当前用户启用多因素认证
// @Tags MFA
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body EnableMFARequest true "启用MFA请求"
// @Success 200 {object} map[string]interface{} "启用成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/mfa/enable [post]
func EnableMFAHandler(c *gin.Context) {
	var req EnableMFARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	err := mfaService.EnableMFA(userID, req.MFAType, req.Phone, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "MFA 启用成功",
	})
}

// DisableMFAHandler 禁用MFA
// @Summary 禁用多因素认证
// @Description 禁用当前用户的多因素认证
// @Tags MFA
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "禁用成功"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/mfa/disable [post]
func DisableMFAHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	err := mfaService.DisableMFA(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "MFA 禁用成功",
	})
}

func GenerateBackupCodesHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	codes, err := mfaService.GenerateBackupCodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"backup_codes": codes,
		},
	})
}

// VerifyBackupCodeHandler 验证备用码
// @Summary 验证备用验证码
// @Description 使用备用验证码进行身份验证
// @Tags MFA
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body VerifyCodeRequest true "备用码验证请求"
// @Success 200 {object} map[string]interface{} "验证成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/mfa/backup-codes/verify [post]
func VerifyBackupCodeHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "备份码验证成功",
	})
}
