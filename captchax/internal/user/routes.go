package user

import (
	"github.com/gin-gonic/gin"
)

func (m *Module) RegisterRoutes(router *gin.Engine) {
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", m.handlers.Register)
		auth.POST("/login", m.handlers.Login)

		protected := auth.Group("")
		protected.Use(m.authMiddleware.Middleware())
		{
			protected.GET("/me", m.handlers.GetMe)
			protected.PUT("/password", m.handlers.ChangePassword)
		}
	}

	users := router.Group("/api/v1/users")
	users.Use(m.authMiddleware.Middleware())
	{
		users.GET("", m.handlers.ListUsers)
		users.GET("/:id", m.handlers.GetUser)
		users.PUT("/:id", m.handlers.UpdateUser)
		users.DELETE("/:id", m.handlers.DeleteUser)
		users.PUT("/:id/role", m.authMiddleware.AdminOnly(), m.handlers.UpdateUserRole)
		users.PUT("/:id/status", m.authMiddleware.AdminOnly(), m.handlers.UpdateUserStatus)
	}
}