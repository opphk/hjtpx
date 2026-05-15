package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/response"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  AdminInfo   `json:"user"`
}

type AdminInfo struct {
	ID           uint   `json:"id"`
	Username     string `json:"username"`
	IsSuperAdmin bool   `json:"is_super_admin"`
}

// Login 管理员登录
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	var admin models.Admin
	if err := database.DB.Where("username = ?", req.Username).First(&admin).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Unauthorized(c)
		} else {
			response.InternalServerError(c, "")
		}
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		response.Unauthorized(c)
		return
	}

	token, err := jwt.GenerateToken(admin.ID, admin.Username)
	if err != nil {
		response.InternalServerError(c, "")
		return
	}

	response.Success(c, LoginResponse{
		Token: token,
		User: AdminInfo{
			ID:           admin.ID,
			Username:     admin.Username,
			IsSuperAdmin: admin.IsSuperAdmin,
		},
	})
}

// Logout 管理员登出
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
