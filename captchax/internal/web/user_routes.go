package web

import (
	"github.com/gin-gonic/gin"
)

type UserRouter struct {
	handlers *UserHandlers
}

func NewUserRouter() *UserRouter {
	handlers := NewUserHandlers()
	return &UserRouter{
		handlers: handlers,
	}
}

func (r *UserRouter) RegisterRoutes(router *gin.Engine) {
	router.GET("/", r.handlers.ShowHomePage)
	router.GET("/login", r.handlers.ShowLoginPage)
	router.GET("/register", r.handlers.ShowRegisterPage)
	router.GET("/dashboard", r.handlers.ShowDashboardPage)
	router.GET("/profile", r.handlers.ShowProfilePage)
}