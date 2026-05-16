package router

import (
	"html/template"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/handler"
	"github.com/hjtpx/hjtpx/internal/api/middleware"
)

// SetupRouter 设置路由
func SetupRouter() *gin.Engine {
	r := gin.New()

	// 全局中间件
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.AddSecurityHeaders())
	r.Use(middleware.SQLInjectionProtection())

	// 健康检查
	r.GET("/health", handler.HealthCheck)
	r.GET("/healthz", handler.Liveness)
	r.GET("/readyz", handler.Readiness)

	// 静态文件服务
	r.Static("/static", "../frontend/static")
	r.Static("/admin/static", "../admin/static")

	// 加载HTML模板（合并前端和管理端模板，避免多次LoadHTMLGlob覆盖）
	templates := template.Must(template.New("").Parse(""))
	for _, glob := range []string{"../frontend/templates/*", "../admin/templates/*"} {
		matches, err := filepath.Glob(glob)
		if err != nil {
			continue
		}
		for _, match := range matches {
			template.Must(templates.ParseFiles(match))
		}
	}
	r.SetHTMLTemplate(templates)

	// 前端页面路由
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "home.html", nil)
	})
	r.GET("/captcha", func(c *gin.Context) {
		c.HTML(200, "captcha.html", nil)
	})
	r.GET("/login", func(c *gin.Context) {
		c.HTML(200, "home.html", nil)
	})
	r.GET("/register", func(c *gin.Context) {
		c.HTML(200, "home.html", nil)
	})
	r.GET("/products", func(c *gin.Context) {
		c.HTML(200, "home.html", nil)
	})
	r.GET("/about", func(c *gin.Context) {
		c.HTML(200, "home.html", nil)
	})
	r.GET("/contact", func(c *gin.Context) {
		c.HTML(200, "home.html", nil)
	})

	// 管理端页面路由
	r.GET("/admin/login", func(c *gin.Context) {
		c.HTML(200, "login.html", nil)
	})
	r.GET("/admin", func(c *gin.Context) {
		c.HTML(200, "dashboard.html", nil)
	})
	r.GET("/admin/stats", func(c *gin.Context) {
		c.HTML(200, "stats.html", nil)
	})
	r.GET("/admin/applications", func(c *gin.Context) {
		c.HTML(200, "applications.html", nil)
	})
	r.GET("/admin/logs", func(c *gin.Context) {
		c.HTML(200, "logs.html", nil)
	})
	r.GET("/admin/risk-rules", func(c *gin.Context) {
		c.HTML(200, "risk-rules.html", nil)
	})
	r.GET("/admin/blacklist", func(c *gin.Context) {
		c.HTML(200, "blacklist.html", nil)
	})

	// API路由组
	api := r.Group("/api/v1")
	{
		// 验证码相关路由
		captcha := api.Group("/captcha")
		{
			captcha.GET("/slider", handler.GetSliderCaptcha)
			captcha.GET("/click", handler.GetClickCaptcha)
			captcha.GET("/rotation", handler.GenerateRotationCaptcha)
			captcha.POST("/rotation/verify", handler.VerifyRotationCaptcha)
			captcha.POST("/verify", handler.VerifyCaptcha)
			captcha.GET("/shuffle/click", handler.GetShuffleClickCaptcha)
			captcha.POST("/shuffle/verify", handler.VerifyShuffleClickCaptcha)
		}

		// 环境检测路由
		api.GET("/detect/script", handler.GetDetectionScript)
		api.POST("/detect/submit", handler.SubmitDetectionData)
		api.POST("/detect/check", handler.EnvironmentCheck)
		api.GET("/detect/fingerprint", handler.GetFingerprintInfo)
		api.GET("/detect/stats", handler.GetFingerprintStats)

		// 高级环境检测路由
		api.GET("/detect/advanced/script", handler.GetAdvancedDetectionScript)
		api.POST("/detect/advanced/submit", handler.SubmitAdvancedDetection)
		api.GET("/detect/advanced/result", handler.GetAdvancedDetectionResult)
		api.POST("/detect/advanced/browser-engine", handler.AnalyzeBrowserEngine)
		api.POST("/detect/advanced/vm", handler.DetectVMEnvironment)
		api.POST("/detect/advanced/cloud", handler.DetectCloudEnvironment)
		api.POST("/detect/advanced/container", handler.DetectContainerEnvironment)
		api.POST("/detect/advanced/headless", handler.DetectHeadlessBrowser)
		api.POST("/detect/advanced/canvas", handler.EnhancedCanvasFingerprint)
		api.POST("/detect/advanced/webgl", handler.EnhancedWebGLFingerprint)
		api.POST("/detect/advanced/batch", handler.BatchDetection)

		// 认证路由（供前端调用）
		auth := api.Group("/auth")
		{
			userHandler := handler.GetUserHandler()
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)
			auth.POST("/logout", userHandler.Logout)
			auth.POST("/refresh", userHandler.RefreshToken)
			auth.GET("/verify-email", userHandler.VerifyEmail)
			auth.POST("/resend-verification", userHandler.ResendVerification)
			auth.POST("/request-password-reset", userHandler.RequestPasswordReset)
			auth.POST("/reset-password", userHandler.ResetPassword)
		}

		// 用户路由
		user := api.Group("/user")
		user.Use(middleware.UserAuthMiddleware())
		{
			userHandler := handler.GetUserHandler()
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
			user.POST("/change-password", userHandler.ChangePassword)
		}

		// 管理员路由
		admin := api.Group("/admin")
		{
			admin.POST("/login", handler.Login)
			admin.POST("/logout", handler.Logout)

			// 需要JWT认证的路由
			adminAuth := admin.Group("")
			adminAuth.Use(middleware.AuthMiddleware())
			{
				// 仪表盘数据
				dashboard := adminAuth.Group("/dashboard")
				{
					dashboard.GET("/stats", handler.GetDashboardStats)
					dashboard.GET("/activity", handler.GetRecentActivity)
					dashboard.GET("/system-status", handler.GetSystemStatus)
					dashboard.GET("/request-trend", handler.GetRequestTrend)
				}

				// 统计数据
				stats := adminAuth.Group("/stats")
				{
					stats.GET("/verification", handler.GetVerificationStats)
					stats.GET("/chart", handler.GetChartData)
					stats.GET("/trend", handler.GetTrendData)
					stats.GET("/hourly", handler.GetHourlyStats)
					stats.GET("/realtime", handler.GetRealtimeStats)
					stats.GET("/risk-distribution", handler.GetRiskDistribution)
					stats.GET("/top-ips", handler.GetTopIPs)
					stats.GET("/application", handler.GetApplicationStats)
					stats.GET("/captcha-type", handler.GetCaptchaTypeStats)
					stats.GET("/report", handler.GenerateReport)
				}

				// 应用管理
				applications := adminAuth.Group("/applications")
				{
					applications.GET("/summary", handler.GetApplicationsSummary)
					applications.GET("", handler.ListApplications)
					applications.POST("", handler.CreateApplication)
					applications.GET("/:id", handler.GetApplication)
					applications.PUT("/:id", handler.UpdateApplication)
					applications.DELETE("/:id", handler.DeleteApplication)
					applications.POST("/:id/regenerate-key", handler.RegenerateApplicationKey)
					applications.GET("/:id/config", handler.GetApplicationConfig)
					applications.PUT("/:id/config", handler.UpdateApplicationConfig)
					applications.GET("/:id/statistics", handler.GetApplicationStatistics)
				}

				// 验证日志查询
				logs := adminAuth.Group("/logs")
				{
					logs.GET("/summary", handler.GetLogsSummary)
					logs.GET("", handler.GetVerificationLogs)
					logs.GET("/statistics", handler.GetLogStatistics)
					logs.GET("/export", handler.ExportLogs)
					logs.GET("/session/:session_id", handler.GetLogsBySession)
					logs.DELETE("/cleanup", handler.DeleteOldLogs)
					logs.POST("/clear", handler.ClearLogs)
					logs.GET("/:id", handler.GetLogDetail)
				}

				// 黑名单管理
				blacklist := adminAuth.Group("/blacklist")
				{
					blacklist.GET("/summary", handler.GetBlacklistSummary)
					blacklist.GET("", handler.ListBlacklist)
					blacklist.POST("", handler.CreateBlacklist)
					blacklist.POST("/import", handler.ImportBlacklist)
					blacklist.GET("/:id", handler.GetBlacklistByID)
					blacklist.PUT("/:id", handler.UpdateBlacklist)
					blacklist.DELETE("/:id", handler.DeleteBlacklist)
					blacklist.POST("/:id/unblock", handler.UnblockBlacklist)
				}

				// 风控规则管理
				riskRules := adminAuth.Group("/risk-rules")
				{
					riskRules.GET("/summary", handler.GetRiskRulesSummary)
					riskRules.GET("", handler.ListRiskRules)
					riskRules.POST("", handler.CreateRiskRule)
					riskRules.GET("/:id", handler.GetRiskRule)
					riskRules.PUT("/:id", handler.UpdateRiskRule)
					riskRules.DELETE("/:id", handler.DeleteRiskRule)
					riskRules.POST("/:id/toggle", handler.ToggleRiskRule)
				}

				// CSS来源切换
				adminAuth.GET("/css-source", handler.GetCSSSource)
				adminAuth.POST("/css-source", handler.SetCSSSource)
			}
		}

		// 示例路由
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})

		// 性能监控路由
		metrics := api.Group("/metrics")
		{
			metrics.GET("/performance", handler.GetPerformanceMetrics)
			metrics.GET("/endpoints", handler.GetEndpointStats)
			metrics.GET("/health", handler.GetHealthCheckMetrics)
			metrics.POST("/reset", handler.ResetMetrics)
		}
	}

	return r
}