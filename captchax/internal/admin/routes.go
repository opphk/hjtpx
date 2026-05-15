package admin

import (
	"database/sql"
	"time"

	"captchax/internal/monitoring"
	"captchax/internal/repository"
	"captchax/pkg/response"

	"github.com/gin-gonic/gin"
)

type Router struct {
	handlers       *AdminHandlers
	rbacHandlers   *RBACHandlers
	userHandlers   *UserManagementHandlers
	settingsHandlers *SettingsHandlers
	auditHandlers  *AuditHandlers
	exportHandlers *ExportHandlers
	auth           *AuthService
}

func NewRouter(
	adminRepo *repository.AdminRepo,
	whitelistRepo *repository.WhitelistRepo,
	blacklistRepo *repository.BlacklistRepo,
	configRepo *repository.ConfigRepo,
	captchaRepo *repository.CaptchaRepo,
	jwtSecret string,
	tokenTTLSeconds int,
	db *sql.DB,
	metrics *monitoring.Metrics,
) *Router {
	tokenTTL := time.Duration(tokenTTLSeconds) * time.Second
	if tokenTTLSeconds <= 0 {
		tokenTTL = 24 * time.Hour
	}

	authService := NewAuthService(adminRepo, jwtSecret, tokenTTL)
	analyticsService := NewAnalyticsService(captchaRepo, db)
	exportService := NewExportService(captchaRepo, db)

	roleRepo := repository.NewRoleRepo(db)
	permRepo := repository.NewPermissionRepo(db)
	adminRoleRepo := repository.NewAdminRoleRepo(db)

	rbacService := NewRBACService(roleRepo, permRepo, adminRoleRepo, adminRepo)

	handlers := NewAdminHandlers(
		authService,
		adminRepo,
		whitelistRepo,
		blacklistRepo,
		configRepo,
		captchaRepo,
		analyticsService,
		metrics,
	)

	exportHandlers := NewExportHandlers(exportService)
	rbacHandlers := NewRBACHandlers(rbacService)
	userHandlers := NewUserManagementHandlers(db)
	settingsHandlers := NewSettingsHandlers(db)
	auditHandlers := NewAuditHandlers(db)

	return &Router{
		handlers:       handlers,
		rbacHandlers:   rbacHandlers,
		userHandlers:   userHandlers,
		settingsHandlers: settingsHandlers,
		auditHandlers:  auditHandlers,
		exportHandlers: exportHandlers,
		auth:           authService,
	}
}

func (r *Router) RegisterRoutes(router *gin.Engine) {
	router.LoadHTMLGlob("web/templates/admin/*.html")

	// 页面路由
	router.GET("/admin/login", r.handlers.ShowLoginPage)
	router.GET("/admin/dashboard", r.handlers.ShowDashboardPage)
	router.GET("/admin/realtime", r.handlers.ShowRealtimePage)
	router.GET("/admin/stats", r.handlers.ShowStatsPage)
	router.GET("/admin/analytics", r.handlers.ShowAnalyticsPage)
	router.GET("/admin/config", r.handlers.ShowConfigPage)
	router.GET("/admin/whitelist", r.handlers.ShowWhitelistPage)
	router.GET("/admin/blacklist", r.handlers.ShowBlacklistPage)
	router.GET("/admin/admins", r.handlers.ShowAdminsPage)
	router.GET("/admin/roles", r.handlers.ShowRolesPage)
	router.GET("/admin/users", r.userHandlers.ShowUsersPage)
	router.GET("/admin/settings", r.settingsHandlers.ShowSettingsPage)
	router.GET("/admin/audit", r.auditHandlers.ShowAuditPage)

	router.GET("/admin/ws", r.auth.AuthMiddleware(), r.handlers.HandleWebSocket)
	router.GET("/admin/api/realtime/stats", r.auth.AuthMiddleware(), r.handlers.GetRealtimeStats)
	router.GET("/admin/api/realtime/charts", r.auth.AuthMiddleware(), r.handlers.GetRealtimeCharts)

	apiGroup := router.Group("/admin/api")
	{
		apiGroup.POST("/login", r.handlers.Login)
		apiGroup.POST("/logout", r.handlers.Logout)

		protected := apiGroup.Group("")
		protected.Use(r.auth.AuthMiddleware())
		{
			protected.GET("/dashboard", r.handlers.GetDashboard)

			protected.GET("/stats", r.handlers.GetStats)
			protected.GET("/stats/trend", r.handlers.GetTrend)
			protected.GET("/stats/captcha-distribution", r.handlers.GetCaptchaDistribution)
			protected.GET("/stats/ip-ranking", r.handlers.GetIPRanking)

			protected.GET("/analytics/overview", r.handlers.GetAnalyticsOverview)
			protected.GET("/analytics/trends", r.handlers.GetAnalyticsTrends)
			protected.GET("/analytics/distribution", r.handlers.GetAnalyticsDistribution)
			protected.GET("/analytics/geo", r.handlers.GetAnalyticsGeo)
			protected.GET("/analytics/devices", r.handlers.GetAnalyticsDevices)
			protected.GET("/analytics/risk", r.handlers.GetAnalyticsRisk)

			protected.GET("/export/count", r.exportHandlers.GetExportCount)
			protected.GET("/export/captchas", r.exportHandlers.ExportCaptchas)
			protected.GET("/export/stats", r.exportHandlers.ExportStats)
			protected.GET("/export/logs", r.exportHandlers.ExportLogs)

			protected.GET("/config", r.handlers.GetConfig)
			protected.POST("/config", r.auth.SuperAdminOnly(), r.handlers.UpdateConfig)

			protected.GET("/whitelist", r.handlers.GetWhitelist)
			protected.POST("/whitelist", r.handlers.AddWhitelist)
			protected.DELETE("/whitelist/:id", r.handlers.DeleteWhitelist)

			protected.GET("/blacklist", r.handlers.GetBlacklist)
			protected.POST("/blacklist", r.handlers.AddBlacklist)
			protected.DELETE("/blacklist/:id", r.handlers.DeleteBlacklist)

			protected.GET("/admins", r.rbacHandlers.GetAdmins)
			protected.GET("/admins/:id", r.rbacHandlers.GetAdmin)
			protected.POST("/admins", r.auth.SuperAdminOnly(), r.rbacHandlers.CreateAdmin)
			protected.PUT("/admins/:id", r.auth.SuperAdminOnly(), r.rbacHandlers.UpdateAdmin)
			protected.DELETE("/admins/:id", r.auth.SuperAdminOnly(), r.rbacHandlers.DeleteAdmin)
			protected.PUT("/admins/:id/roles", r.auth.SuperAdminOnly(), r.rbacHandlers.AssignRoles)

			protected.GET("/roles", r.rbacHandlers.GetRoles)
			protected.GET("/roles/:id", r.rbacHandlers.GetRole)
			protected.POST("/roles", r.auth.SuperAdminOnly(), r.rbacHandlers.CreateRole)
			protected.PUT("/roles/:id", r.auth.SuperAdminOnly(), r.rbacHandlers.UpdateRole)
			protected.DELETE("/roles/:id", r.auth.SuperAdminOnly(), r.rbacHandlers.DeleteRole)

			protected.GET("/permissions", r.rbacHandlers.GetPermissions)
			protected.GET("/me/permissions", r.rbacHandlers.GetMyPermissions)

			protected.GET("/users", r.userHandlers.GetUsers)
			protected.GET("/users/:id", r.userHandlers.GetUser)
			protected.POST("/users", r.auth.SuperAdminOnly(), r.userHandlers.CreateUser)
			protected.PUT("/users/:id", r.userHandlers.UpdateUser)
			protected.DELETE("/users/:id", r.auth.SuperAdminOnly(), r.userHandlers.DeleteUser)
			protected.PUT("/users/:id/role", r.auth.SuperAdminOnly(), r.userHandlers.UpdateUserRole)
			protected.PUT("/users/:id/status", r.userHandlers.UpdateUserStatus)

			protected.GET("/settings", r.settingsHandlers.GetSettings)
			protected.PUT("/settings", r.auth.SuperAdminOnly(), r.settingsHandlers.UpdateSettings)

			protected.GET("/audit-logs", r.auditHandlers.GetAuditLogs)
			protected.GET("/audit-logs/export", r.auditHandlers.ExportAuditLogs)
		}
	}

	router.NoRoute(func(c *gin.Context) {
		response.NotFound(c, "endpoint not found")
	})
}
