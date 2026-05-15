package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=128"`
}

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "无效的请求参数: "+err.Error())
		return
	}

	var existing models.User
	if result := database.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&existing); result.RowsAffected > 0 {
		response.Error(c, 409, "用户名或邮箱已存在")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, 500, "密码加密失败")
		return
	}

	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}

	if err := database.DB.Create(&user).Error; err != nil {
		response.Error(c, 500, "注册失败")
		return
	}

	response.Success(c, gin.H{
		"user_id":  user.ID,
		"username": user.Username,
		"message":  "注册成功",
	})
}

type UserLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func UserLogin(c *gin.Context) {
	var req UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid request parameters")
		return
	}

	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		response.Error(c, 401, "用户名或密码错误")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		response.Error(c, 401, "用户名或密码错误")
		return
	}

	token, err := jwt.GenerateToken(user.ID, user.Username)
	if err != nil {
		response.Error(c, 500, "登录失败")
		return
	}

	response.Success(c, gin.H{
		"token":    token,
		"user_id":  user.ID,
		"username": user.Username,
	})
}