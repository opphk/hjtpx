package handler

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/middleware"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/response"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// LoginRequest 登录请求参数
type LoginRequest struct {
	Username string `json:"username" binding:"required"` // 用户名
	Password string `json:"password" binding:"required"` // 密码
}

// LoginResponse 登录响应数据
type LoginResponse struct {
	Token        string    `json:"token"`         // 访问令牌
	RefreshToken string    `json:"refresh_token,omitempty"` // 刷新令牌
	ExpiresIn    int64     `json:"expires_in"`    // 过期时间(秒)
	User         AdminInfo `json:"user"`          // 用户信息
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AdminInfo struct {
	ID           uint   `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email,omitempty"`
	Role         string `json:"role"`
	Status       string `json:"status"`
	IsSuperAdmin bool   `json:"is_super_admin"`
	LastLoginAt  string `json:"last_login_at,omitempty"`
}

type LoginLogEntry struct {
	ID         uint   `json:"id"`
	IPAddress  string `json:"ip_address"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	FailReason string `json:"fail_reason,omitempty"`
}

func getClientIP(c *gin.Context) string {
	ip := c.GetHeader("X-Forwarded-For")
	if ip == "" {
		ip = c.GetHeader("X-Real-IP")
	}
	if ip == "" {
		ip = c.ClientIP()
	}
	ips := strings.Split(ip, ",")
	if len(ips) > 0 {
		ip = strings.TrimSpace(ips[0])
	}
	return ip
}

func getUserAgent(c *gin.Context) string {
	return c.GetHeader("User-Agent")
}

func recordLoginLog(db *gorm.DB, adminID uint, ip, userAgent, status, failReason string) {
	if db == nil {
		return
	}
	log := models.AdminLoginLog{
		AdminID:   adminID,
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    status,
	}
	if failReason != "" {
		log.FailReason = failReason
	}
	db.Create(&log)
}

func updateAdminLogin(db *gorm.DB, adminID uint, ip string) {
	if db == nil {
		return
	}
	now := time.Now()
	db.Model(&models.Admin{}).Where("id = ?", adminID).Updates(map[string]interface{}{
		"last_login_at": now,
		"last_login_ip": ip,
	})
}

// Login 管理员登录
// @Summary 管理员登录
// @Description 使用用户名和密码进行管理员登录，返回访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body LoginRequest true "登录请求参数"
// @Success 200 {object} LoginResponse "登录成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "认证失败"
// @Failure 403 {object} map[string]interface{} "账户被禁用"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/auth/login [post]
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		response.BadRequest(c, "username cannot be empty")
		return
	}

	if len(req.Password) < 6 {
		response.BadRequest(c, "password must be at least 6 characters")
		return
	}

	clientIP := getClientIP(c)
	userAgent := getUserAgent(c)

	var admin models.Admin
	if err := database.DB.Where("username = ?", req.Username).First(&admin).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			recordLoginLog(database.DB, 0, clientIP, userAgent, "failed", "user not found")
			response.Fail(c, response.CodeUnauthorized, "用户名或密码错误")
			return
		}
		recordLoginLog(database.DB, 0, clientIP, userAgent, "failed", "database error")
		response.InternalServerError(c, "")
		return
	}

	if admin.Status == "disabled" {
		recordLoginLog(database.DB, admin.ID, clientIP, userAgent, "failed", "account disabled")
		response.Fail(c, response.CodeForbidden, "账户已被禁用")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		recordLoginLog(database.DB, admin.ID, clientIP, userAgent, "failed", "wrong password")
		response.Fail(c, response.CodeUnauthorized, "用户名或密码错误")
		return
	}

	token, err := jwt.GenerateToken(admin.ID, admin.Username)
	if err != nil {
		recordLoginLog(database.DB, admin.ID, clientIP, userAgent, "failed", "token generation failed")
		response.InternalServerError(c, "")
		return
	}

	updateAdminLogin(database.DB, admin.ID, clientIP)
	recordLoginLog(database.DB, admin.ID, clientIP, userAgent, "success", "")

	if redis.Client != nil {
		ctx := c.Request.Context()
		loginTime := time.Now().Add(24 * time.Hour)
		_ = loginTime
		redis.Client.Set(ctx, "admin:last_login:"+admin.Username, loginTime.Format(time.RFC3339), 24*time.Hour)
	}

	response.Success(c, LoginResponse{
		Token:     token,
		ExpiresIn: 86400,
		User: AdminInfo{
			ID:           admin.ID,
			Username:     admin.Username,
			Role:         "admin",
			IsSuperAdmin: admin.IsSuperAdmin,
		},
	})
}

func RefreshTokenHandler(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	claims, err := jwt.ParseToken(req.RefreshToken)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	newToken, err := jwt.GenerateToken(claims.AdminID, claims.Username)
	if err != nil {
		response.InternalServerError(c, "")
		return
	}

	response.Success(c, gin.H{
		"token": newToken,
	})
}

func Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	if token != "" && redis.Client != nil {
		ctx := c.Request.Context()
		_, err := jwt.ParseToken(token)
		if err == nil {
			redis.Client.Set(ctx, "logout:"+token, "1", 0)
		}
	}

	response.Success(c, nil)
}

func GetLoginHistory(c *gin.Context) {
	adminID := middleware.GetAdminID(c)
	if adminID == 0 {
		response.Unauthorized(c)
		return
	}

	page := 1
	pageSize := 20

	var logs []models.AdminLoginLog
	var total int64

	database.DB.Model(&models.AdminLoginLog{}).Where("admin_id = ?", adminID).Count(&total)

	offset := (page - 1) * pageSize
	database.DB.Where("admin_id = ?", adminID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&logs)

	logEntries := make([]LoginLogEntry, len(logs))
	for i, log := range logs {
		logEntries[i] = LoginLogEntry{
			ID:         log.ID,
			IPAddress:  log.IPAddress,
			Status:     log.Status,
			CreatedAt:  log.CreatedAt.Format(time.RFC3339),
			FailReason: log.FailReason,
		}
	}

	response.Success(c, gin.H{
		"logs":      logEntries,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func ChangePassword(c *gin.Context) {
	type ChangePasswordRequest struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	adminID := middleware.GetAdminID(c)
	if adminID == 0 {
		response.Unauthorized(c)
		return
	}

	if len(req.NewPassword) < 6 {
		response.BadRequest(c, "新密码长度至少为6个字符")
		return
	}

	var admin models.Admin
	if err := database.DB.First(&admin, adminID).Error; err != nil {
		response.InternalServerError(c, "")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.OldPassword)); err != nil {
		response.Fail(c, response.CodeUnauthorized, "原密码错误")
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		response.InternalServerError(c, "")
		return
	}

	database.DB.Model(&admin).Update("password_hash", string(newHash))

	response.Success(c, nil)
}

func GetCurrentUser(c *gin.Context) {
	adminID := middleware.GetAdminID(c)
	if adminID == 0 {
		response.Unauthorized(c)
		return
	}

	var admin models.Admin
	if err := database.DB.First(&admin, adminID).Error; err != nil {
		response.InternalServerError(c, "")
		return
	}

	response.Success(c, AdminInfo{
		ID:           admin.ID,
		Username:     admin.Username,
		Email:        admin.Email,
		Role:         "admin",
		Status:       admin.Status,
		IsSuperAdmin: admin.IsSuperAdmin,
		LastLoginAt:  "",
	})
}

func AdminDashboard(c *gin.Context) {
	adminID := middleware.GetAdminID(c)
	if adminID == 0 {
		response.Unauthorized(c)
		return
	}

	var admin models.Admin
	if err := database.DB.First(&admin, adminID).Error; err != nil {
		response.InternalServerError(c, "")
		return
	}

	var totalVerifications int64
	var todayVerifications int64
	var blockedRequests int64

	database.DB.Model(&models.Verification{}).Count(&totalVerifications)
	database.DB.Model(&models.Verification{}).Where("created_at >= ?", time.Now().AddDate(0, 0, -1)).Count(&todayVerifications)
	database.DB.Model(&models.Verification{}).Where("status = ?", "blocked").Count(&blockedRequests)

	var activeUsers int64
	database.DB.Model(&models.User{}).Where("status = ?", "active").Count(&activeUsers)

	var activeApps int64
	database.DB.Model(&models.Application{}).Where("is_active = ?", true).Count(&activeApps)

	var recentLogs []models.AdminLoginLog
	database.DB.Where("admin_id = ?", adminID).Order("created_at DESC").Limit(10).Find(&recentLogs)

	logEntries := make([]LoginLogEntry, len(recentLogs))
	for i, log := range recentLogs {
		logEntries[i] = LoginLogEntry{
			ID:        log.ID,
			IPAddress: log.IPAddress,
			Status:    log.Status,
			CreatedAt: log.CreatedAt.Format(time.RFC3339),
		}
	}

	response.Success(c, gin.H{
		"admin": AdminInfo{
			ID:           admin.ID,
			Username:     admin.Username,
			Role:         "admin",
			IsSuperAdmin: admin.IsSuperAdmin,
		},
		"stats": gin.H{
			"total_verifications":  totalVerifications,
			"today_verifications":   todayVerifications,
			"blocked_requests":      blockedRequests,
			"active_users":          activeUsers,
			"active_applications":   activeApps,
		},
		"recent_activity": logEntries,
	})
}
