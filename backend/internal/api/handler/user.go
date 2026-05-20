package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/middleware"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type UserHandler struct {
	userService     *service.UserService
	behaviorService *service.BehaviorAnalysisService
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		userService:     service.NewUserService(),
		behaviorService: service.NewBehaviorAnalysisService(),
	}
}

// RegisterRequest 用户注册请求
// @Description 用户注册请求参数
type RegisterRequest struct {
	Username     string `json:"username" binding:"required,min=3,max=50"`  // 用户名
	Email        string `json:"email" binding:"required,email"`            // 邮箱
	Password     string `json:"password" binding:"required,min=6,max=128"` // 密码
	BehaviorData string `json:"behavior_data,omitempty"`                   // 行为数据（可选）
}

// RegisterResponse 用户注册响应
// @Description 用户注册响应数据
type RegisterResponse struct {
	UserID           uint   `json:"user_id"`                     // 用户ID
	Username         string `json:"username"`                    // 用户名
	Email            string `json:"email"`                       // 邮箱
	VerificationLink string `json:"verification_link,omitempty"` // 验证链接
	Message          string `json:"message"`                     // 消息
}

// LoginRequest 用户登录请求
// @Description 用户登录请求参数
type UserLoginRequest struct {
	Username     string `json:"username" binding:"required"` // 用户名
	Password     string `json:"password" binding:"required"` // 密码
	CaptchaToken string `json:"captcha_token,omitempty"`     // 验证码令牌
	BehaviorData string `json:"behavior_data,omitempty"`     // 行为数据
}

// LoginResponse 用户登录响应
// @Description 用户登录响应数据
type UserLoginResponse struct {
	AccessToken  string                `json:"access_token"`  // 访问令牌
	RefreshToken string                `json:"refresh_token"` // 刷新令牌
	ExpiresIn    int64                 `json:"expires_in"`    // 过期时间（秒）
	User         *service.UserResponse `json:"user"`          // 用户信息
}

// RefreshRequest 刷新令牌请求
// @Description 刷新访问令牌请求
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"` // 刷新令牌
}

// RefreshResponse 刷新令牌响应
// @Description 刷新令牌响应数据
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`  // 新访问令牌
	RefreshToken string `json:"refresh_token"` // 新刷新令牌
	ExpiresIn    int64  `json:"expires_in"`    // 过期时间（秒）
}

// UpdateProfileRequest 更新用户资料请求
// @Description 更新用户资料请求参数
type UpdateProfileRequest struct {
	Nickname string `json:"nickname" binding:"max=100"` // 昵称
	Avatar   string `json:"avatar" binding:"max=500"`   // 头像URL
	Phone    string `json:"phone" binding:"max=20"`     // 电话
	Bio      string `json:"bio" binding:"max=500"`      // 个人简介
}

// ChangePasswordRequest 修改密码请求
// @Description 修改密码请求参数
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`               // 原密码
	NewPassword string `json:"new_password" binding:"required,min=6,max=128"` // 新密码
}

// RequestPasswordResetRequest 请求密码重置请求
// @Description 请求密码重置请求参数
type RequestPasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"` // 邮箱
}

// PasswordResetResponse 密码重置响应
// @Description 密码重置响应数据
type PasswordResetResponse struct {
	ResetLink string `json:"reset_link,omitempty"` // 重置链接
	Message   string `json:"message"`              // 消息
}

// ResetPasswordRequest 重置密码请求
// @Description 重置密码请求参数
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`                      // 重置令牌
	NewPassword string `json:"new_password" binding:"required,min=6,max=128"` // 新密码
}

// ResendVerificationRequest 重新发送验证邮件请求
// @Description 重新发送验证邮件请求参数
type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"` // 邮箱
}

// Register 用户注册
// @Summary 用户注册
// @Description 创建新用户账户
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param body body RegisterRequest true "注册请求参数"
// @Success 200 {object} RegisterResponse "注册成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 409 {object} map[string]interface{} "用户名或邮箱已存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters: "+err.Error())
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Username == "" || req.Email == "" || req.Password == "" {
		response.BadRequest(c, "username, email and password are required")
		return
	}

	var behaviorData []interface{}
	var riskScore float64 = 0.0

	if req.BehaviorData != "" {
		if err := json.Unmarshal([]byte(req.BehaviorData), &behaviorData); err == nil {
			modelBehaviorData := convertToBehaviorData(behaviorData)
			var bdModels []models.BehaviorData
			for _, bd := range modelBehaviorData {
				bdModels = append(bdModels, models.BehaviorData{Data: bd.Data})
			}
			riskScore = h.behaviorService.CalculateRiskScore(nil, bdModels)
		}
	}

	input := &service.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}

	user, verificationToken, err := h.userService.Register(input, riskScore)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			response.Error(c, 409, "username or email already exists")
			return
		}
		response.InternalServerError(c, "registration failed: "+err.Error())
		return
	}

	verificationLink := ""
	if verificationToken != "" {
		verificationLink = fmt.Sprintf("/api/v1/auth/verify-email?token=%s", verificationToken)
	}

	response.Success(c, RegisterResponse{
		UserID:           user.ID,
		Username:         user.Username,
		Email:            user.Email,
		VerificationLink: verificationLink,
		Message:          "registration successful, please verify your email",
	})
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户使用用户名和密码登录
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param body body UserLoginRequest true "登录请求参数"
// @Success 200 {object} UserLoginResponse "登录成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "用户名或密码错误"
// @Failure 403 {object} map[string]interface{} "账户被禁用"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters: "+err.Error())
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		response.BadRequest(c, "username and password are required")
		return
	}

	var behaviorData []interface{}

	if req.BehaviorData != "" {
		if err := json.Unmarshal([]byte(req.BehaviorData), &behaviorData); err == nil {
			modelBehaviorData := convertToBehaviorData(behaviorData)
			var bdModels []models.BehaviorData
			for _, bd := range modelBehaviorData {
				bdModels = append(bdModels, models.BehaviorData{Data: bd.Data})
			}
			_ = h.behaviorService.CalculateRiskScore(nil, bdModels)
		}
	}

	clientIP := c.ClientIP()

	input := &service.LoginInput{
		Username: req.Username,
		Password: req.Password,
	}

	user, _, err := h.userService.Login(input, clientIP)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) || errors.Is(err, service.ErrInvalidPassword) {
			response.Error(c, 401, "invalid username or password")
			return
		}
		if errors.Is(err, service.ErrUserDisabled) {
			response.Error(c, 403, "account is disabled")
			return
		}
		response.InternalServerError(c, "login failed: "+err.Error())
		return
	}

	accessToken, refreshToken, err := jwt.GenerateUserTokenWithRefresh(user.ID, user.Username)
	if err != nil {
		response.InternalServerError(c, "failed to generate token")
		return
	}

	if redis.Client != nil {
		ctx := c.Request.Context()
		redis.Client.Set(ctx, "user_refresh:"+user.Username, refreshToken, 7*24*60*60*1000*1000*1000)
	}

	response.Success(c, UserLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900,
		User:         service.ToUserResponse(user),
	})
}

func (h *UserHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters: "+err.Error())
		return
	}

	claims, err := jwt.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	if redis.Client != nil {
		ctx := c.Request.Context()
		storedToken, err := redis.Client.Get(ctx, "user_refresh:"+claims.Username).Result()
		if err == nil && storedToken != req.RefreshToken {
			response.Unauthorized(c)
			return
		}
	}

	user, err := h.userService.GetUserByID(claims.UserID)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	if user.Status != "active" {
		response.Forbidden(c)
		return
	}

	accessToken, newRefreshToken, err := jwt.GenerateUserTokenWithRefresh(user.ID, user.Username)
	if err != nil {
		response.InternalServerError(c, "failed to generate token")
		return
	}

	if redis.Client != nil {
		ctx := c.Request.Context()
		redis.Client.Set(ctx, "user_refresh:"+user.Username, newRefreshToken, 7*24*60*60*1000*1000*1000)
	}

	response.Success(c, RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    900,
	})
}

func (h *UserHandler) Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	if token != "" && redis.Client != nil {
		ctx := c.Request.Context()
		redis.Client.Set(ctx, "user_logout:"+token, "1", 24*60*60*1000*1000*1000)
	}

	userID := middleware.GetUserID(c)
	if userID > 0 {
		username := middleware.GetUsername(c)
		if username != "" && redis.Client != nil {
			ctx := c.Request.Context()
			redis.Client.Del(ctx, "user_refresh:"+username)
		}
	}

	response.Success(c, nil)
}

// GetProfile 获取用户资料
// @Summary 获取用户资料
// @Description 获取当前登录用户的详细资料
// @Tags 用户认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "用户资料"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 404 {object} map[string]interface{} "用户不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/auth/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c)
		return
	}

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		response.InternalServerError(c, "failed to get profile")
		return
	}

	response.Success(c, service.ToUserResponse(user))
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c)
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters: "+err.Error())
		return
	}

	input := &service.UpdateProfileInput{
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
		Phone:    req.Phone,
		Bio:      req.Bio,
	}

	user, err := h.userService.UpdateProfile(userID, input)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		response.InternalServerError(c, "failed to update profile: "+err.Error())
		return
	}

	response.Success(c, service.ToUserResponse(user))
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Description 修改当前用户的登录密码
// @Tags 用户认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body ChangePasswordRequest true "修改密码请求"
// @Success 200 {object} map[string]interface{} "修改成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "原密码错误"
// @Failure 404 {object} map[string]interface{} "用户不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/auth/change-password [post]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c)
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters: "+err.Error())
		return
	}

	input := &service.ChangePasswordInput{
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}

	err := h.userService.ChangePassword(userID, input)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		if errors.Is(err, service.ErrInvalidPassword) {
			response.Error(c, 400, "invalid old password")
			return
		}
		response.InternalServerError(c, "failed to change password: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "password changed successfully"})
}

func (h *UserHandler) RequestPasswordReset(c *gin.Context) {
	var req RequestPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid email format")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	resetToken, err := h.userService.RequestPasswordReset(req.Email)
	if err != nil {
		response.InternalServerError(c, "failed to process request")
		return
	}

	if resetToken == "" {
		response.Success(c, PasswordResetResponse{
			Message: "if the email exists, a password reset link will be sent",
		})
		return
	}

	resetLink := fmt.Sprintf("/api/v1/auth/reset-password?token=%s", resetToken)

	response.Success(c, PasswordResetResponse{
		ResetLink: resetLink,
		Message:   "password reset link generated (simulated email sending)",
	})
}

// ResetPassword 重置密码
// @Summary 重置密码
// @Description 使用令牌重置用户密码
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param body body ResetPasswordRequest true "重置密码请求"
// @Success 200 {object} map[string]interface{} "重置成功"
// @Failure 400 {object} map[string]interface{} "令牌无效或已过期"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/auth/reset-password [post]
func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters: "+err.Error())
		return
	}

	input := &service.ResetPasswordInput{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	}

	err := h.userService.ResetPassword(input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			response.Error(c, 400, "invalid or expired reset token")
			return
		}
		if errors.Is(err, service.ErrTokenExpired) {
			response.Error(c, 400, "reset token has expired")
			return
		}
		response.InternalServerError(c, "failed to reset password: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "password reset successfully"})
}

func (h *UserHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		response.BadRequest(c, "verification token is required")
		return
	}

	err := h.userService.VerifyEmail(token)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			response.Error(c, 400, "invalid verification token")
			return
		}
		response.InternalServerError(c, "failed to verify email: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "email verified successfully"})
}

// ResendVerification 重新发送验证邮件
// @Summary 重新发送验证邮件
// @Description 重新发送邮箱验证链接
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param body body ResendVerificationRequest true "重新验证请求"
// @Success 200 {object} map[string]interface{} "发送成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 400 {object} map[string]interface{} "邮箱已验证"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/auth/resend-verification [post]
func (h *UserHandler) ResendVerification(c *gin.Context) {
	var req ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid email format")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	user, err := h.userService.GetUserByEmail(req.Email)
	if err != nil {
		response.Success(c, gin.H{"message": "if the email exists and is not verified, a new verification link will be sent"})
		return
	}

	if user.IsVerified {
		response.Error(c, 400, "email is already verified")
		return
	}

	newToken, err := h.userService.GenerateEmailVerificationToken(user.ID)
	if err != nil {
		response.InternalServerError(c, "failed to generate verification token")
		return
	}

	verificationLink := fmt.Sprintf("/api/v1/auth/verify-email?token=%s", newToken)

	response.Success(c, gin.H{
		"verification_link": verificationLink,
		"message":           "new verification link generated (simulated email sending)",
	})
}

func convertToBehaviorData(data []interface{}) []models.BehaviorData {
	result := make([]models.BehaviorData, 0, len(data))
	for _, item := range data {
		if m, ok := item.(map[string]interface{}); ok {
			jsonData, _ := json.Marshal(m)
			bd := models.BehaviorData{
				Data: string(jsonData),
			}
			result = append(result, bd)
		}
	}
	return result
}

var userHandler *UserHandler

func GetUserHandler() *UserHandler {
	if userHandler == nil {
		userHandler = NewUserHandler()
	}
	return userHandler
}
