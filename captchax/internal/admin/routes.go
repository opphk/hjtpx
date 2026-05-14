package admin

import (
	"time"

	"captchax/internal/repository"
	"captchax/pkg/response"

	"github.com/gin-gonic/gin"
)

type Router struct {
	handlers *AdminHandlers
	auth     *AuthService
}

func NewRouter(
	adminRepo *repository.AdminRepo,
	whitelistRepo *repository.WhitelistRepo,
	blacklistRepo *repository.BlacklistRepo,
	configRepo *repository.ConfigRepo,
	captchaRepo *repository.CaptchaRepo,
	jwtSecret string,
	tokenTTLSeconds int,
) *Router {
	tokenTTL := time.Duration(tokenTTLSeconds) * time.Second
	if tokenTTLSeconds <= 0 {
		tokenTTL = 24 * time.Hour
	}

	authService := NewAuthService(adminRepo, jwtSecret, tokenTTL)

	handlers := NewAdminHandlers(
		authService,
		adminRepo,
		whitelistRepo,
		blacklistRepo,
		configRepo,
		captchaRepo,
	)

	return &Router{
		handlers: handlers,
		auth:     authService,
	}
}

func (r *Router) RegisterRoutes(router *gin.Engine) {
	router.LoadHTMLGlob("templates/*.html")

	router.GET("/admin/login", r.handlers.ShowLoginPage)
	router.GET("/admin/dashboard.html", r.handlers.ShowDashboardPage)

	apiGroup := router.Group("/admin/api")
	{
		apiGroup.POST("/login", r.handlers.Login)
		apiGroup.POST("/logout", r.handlers.Logout)

		protected := apiGroup.Group("")
		protected.Use(r.auth.AuthMiddleware())
		{
			protected.GET("/dashboard", r.handlers.GetDashboard)
			protected.GET("/stats", r.handlers.GetStats)
			protected.GET("/config", r.handlers.GetConfig)
			protected.POST("/config", r.auth.SuperAdminOnly(), r.handlers.UpdateConfig)
			protected.GET("/whitelist", r.handlers.GetWhitelist)
			protected.POST("/whitelist", r.handlers.AddWhitelist)
			protected.DELETE("/whitelist/:id", r.handlers.DeleteWhitelist)
			protected.GET("/blacklist", r.handlers.GetBlacklist)
			protected.POST("/blacklist", r.handlers.AddBlacklist)
			protected.DELETE("/blacklist/:id", r.handlers.DeleteBlacklist)
		}
	}

	router.NoRoute(func(c *gin.Context) {
		response.NotFound(c, "endpoint not found")
	})
}
