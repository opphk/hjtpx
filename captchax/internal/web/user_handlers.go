package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandlers struct {
}

func NewUserHandlers() *UserHandlers {
	return &UserHandlers{}
}

func (h *UserHandlers) ShowHomePage(c *gin.Context) {
	c.HTML(http.StatusOK, "home.html", gin.H{
		"title": "CaptchaX - 智能行为验证系统",
	})
}

func (h *UserHandlers) ShowLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "用户登录 - CaptchaX",
	})
}

func (h *UserHandlers) ShowRegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{
		"title": "用户注册 - CaptchaX",
	})
}

func (h *UserHandlers) ShowDashboardPage(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "用户仪表盘 - CaptchaX",
	})
}

func (h *UserHandlers) ShowProfilePage(c *gin.Context) {
	c.HTML(http.StatusOK, "profile.html", gin.H{
		"title": "个人资料 - CaptchaX",
	})
}