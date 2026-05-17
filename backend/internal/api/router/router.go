package router

import (
	"html/template"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/handler"
	"github.com/hjtpx/hjtpx/internal/api/middleware"
	"github.com/hjtpx/hjtpx/pkg/i18n"
)

// SetupRouter 设置路由
func SetupRouter() *gin.Engine {
	r := gin.New()

	// 全局中间件
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.ErrorHandler())
	// 增强的安全中间件 - OWASP Top 10
	r.Use(middleware.OWASPTop10SecurityMiddleware())
	// 性能优化中间件
	r.Use(middleware.GzipCompression())
	r.Use(middleware.PerformanceMonitoring())
	r.Use(middleware.RequestID())
	// i18n 中间件
	r.Use(i18n.Middleware())

	// 翻译文件静态服务
	translationsGroup := r.Group("/translations")
	translationsGroup.Use(middleware.CacheControl(1 * time.Hour))
	translationsGroup.Static("", "translations")

	adminTranslationsGroup := r.Group("/admin/translations")
	adminTranslationsGroup.Use(middleware.CacheControl(1 * time.Hour))
	adminTranslationsGroup.Static("", "translations")

	// 健康检查
	r.GET("/health", handler.HealthCheck)
	r.GET("/healthz", handler.Liveness)
	r.GET("/readyz", handler.Readiness)

	// 静态文件服务 - 带缓存优化
	staticGroup := r.Group("/static")
	staticGroup.Use(middleware.CacheControl(7 * 24 * time.Hour))
	staticGroup.Static("", "../frontend/static")
	
	adminStaticGroup := r.Group("/admin/static")
	adminStaticGroup.Use(middleware.CacheControl(7 * 24 * time.Hour))
	adminStaticGroup.Static("", "../admin/static")

	// 开发者工具静态文件服务
	devtoolsStaticGroup := r.Group("/devtools/static")
	devtoolsStaticGroup.Use(middleware.CacheControl(7 * 24 * time.Hour))
	devtoolsStaticGroup.Static("", "../devtools/static")

	// 加载HTML模板（合并前端、管理端和开发者工具模板，避免多次LoadHTMLGlob覆盖）
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
	r.GET("/admin/advanced-analytics", func(c *gin.Context) {
		c.HTML(200, "advanced-analytics.html", nil)
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
	r.GET("/admin/monitoring", func(c *gin.Context) {
		c.HTML(200, "monitoring.html", nil)
	})
	r.GET("/admin/ab-testing", func(c *gin.Context) {
		c.HTML(200, "ab-testing.html", nil)
	})
	r.GET("/admin/seamless", func(c *gin.Context) {
		c.HTML(200, "seamless.html", nil)
	})
	r.GET("/admin/real-time-screen", func(c *gin.Context) {
		c.HTML(200, "real-time-screen.html", nil)
	})
	r.GET("/admin/reports", func(c *gin.Context) {
		c.HTML(200, "reports.html", nil)
	})
	r.GET("/admin/visualization", func(c *gin.Context) {
		c.HTML(200, "visualization.html", nil)
	})
	r.GET("/admin/batch-operations", func(c *gin.Context) {
		c.HTML(200, "batch-operations.html", nil)
	})

	// 开发者工具页面路由
	r.GET("/devtools", func(c *gin.Context) {
		c.HTML(200, "api-console.html", gin.H{"Title": "API 调试控制台", "ActivePage": "api-console"})
	})
	r.GET("/devtools/api-console", func(c *gin.Context) {
		c.HTML(200, "api-console.html", gin.H{"Title": "API 调试控制台", "ActivePage": "api-console"})
	})
	r.GET("/devtools/captcha-test", func(c *gin.Context) {
		c.HTML(200, "captcha-test.html", gin.H{"Title": "验证码在线测试", "ActivePage": "captcha-test"})
	})
	r.GET("/devtools/code-generator", func(c *gin.Context) {
		c.HTML(200, "code-generator.html", gin.H{"Title": "SDK 代码生成器", "ActivePage": "code-generator"})
	})
	r.GET("/devtools/examples", func(c *gin.Context) {
		c.HTML(200, "examples.html", gin.H{"Title": "集成示例", "ActivePage": "examples"})
	})
	r.GET("/devtools/docs", func(c *gin.Context) {
		c.HTML(200, "docs.html", gin.H{"Title": "API 文档", "ActivePage": "docs"})
	})

	// 初始化导出处理器
	exportHandler := handler.NewExportHandler()

	// 初始化行为分析处理器
	behaviorHandler := handler.GetBehaviorHandler()

	// API路由组
	api := r.Group("/api/v1")
	{
		// 验证码相关路由
		captcha := api.Group("/captcha")
		{
			captcha.GET("/slider", handler.GetSliderCaptcha)
			captcha.GET("/click", handler.GetClickCaptcha)
			captcha.POST("/verify", handler.VerifyCaptcha)
			captcha.GET("/gesture", handler.GenerateGestureCaptcha)
			captcha.POST("/gesture/verify", handler.VerifyGestureCaptcha)
			captcha.GET("/jigsaw", handler.GenerateJigsawCaptcha)
			captcha.POST("/jigsaw/verify", handler.VerifyJigsawCaptcha)
		}

		// 环境检测路由
		api.GET("/detect/script", handler.GetDetectionScript)
		api.POST("/detect/submit", handler.SubmitDetectionData)
		api.POST("/detect/check", handler.EnvironmentCheck)
		api.GET("/detect/fingerprint", handler.GetFingerprintStats)

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

		// 无感验证路由
		seamless := api.Group("/seamless")
		{
			seamless.POST("/verify", handler.SeamlessVerify)
		}

		// 用户路由
		user := api.Group("/user")
		user.Use(middleware.UserAuthMiddleware())
		{
			userHandler := handler.GetUserHandler()
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
			user.POST("/change-password", userHandler.ChangePassword)
			user.GET("/trusted-devices", handler.GetTrustedDevices)
			user.POST("/trusted-devices", handler.TrustDevice)
			user.DELETE("/trusted-devices/:id", handler.RevokeTrustedDevice)
		}

		// 管理员路由
		admin := api.Group("/admin")
		{
			admin.POST("/login", handler.Login)
			admin.POST("/logout", handler.Logout)
			admin.GET("/monitoring/ws", handler.WebSocketHandler)

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
					logs.GET("/export/enhanced", exportHandler.EnhancedExportLogs) // 新增增强导出
					logs.GET("/session/:session_id", handler.GetLogsBySession)
					logs.DELETE("/cleanup", handler.DeleteOldLogs)
					logs.POST("/clear", handler.ClearLogs)
					logs.GET("/:id", handler.GetLogDetail)
				}
				
				// 定时导出管理
				scheduledExports := adminAuth.Group("/scheduled-exports")
				{
					scheduledExports.GET("", exportHandler.ListScheduledExports)
					scheduledExports.POST("", exportHandler.CreateScheduledExport)
					scheduledExports.GET("/:id", exportHandler.GetScheduledExport)
					scheduledExports.PUT("/:id", exportHandler.UpdateScheduledExport)
					scheduledExports.DELETE("/:id", exportHandler.DeleteScheduledExport)
					scheduledExports.POST("/:id/execute", exportHandler.ExecuteScheduledExport)
				}
				
				// 报表模板管理
				reportTemplates := adminAuth.Group("/report-templates")
				{
					reportTemplates.GET("", exportHandler.ListReportTemplates)
					reportTemplates.POST("", exportHandler.CreateReportTemplate)
					reportTemplates.GET("/:id", exportHandler.GetReportTemplate)
					reportTemplates.PUT("/:id", exportHandler.UpdateReportTemplate)
					reportTemplates.DELETE("/:id", exportHandler.DeleteReportTemplate)
				}
				
				// 导出历史
				exportHistory := adminAuth.Group("/export-history")
				{
					exportHistory.GET("", exportHandler.ListExportHistory)
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

				// 高级分析
				analytics := adminAuth.Group("/analytics")
				{
					analytics.GET("/user-behavior", handler.GetUserBehaviorAnalysis)
					analytics.GET("/attack-trend", handler.GetAttackTrendAnalysis)
					analytics.POST("/generate-report", handler.GenerateRiskReport)
					analytics.GET("/visualization", handler.GetVisualizationData)
					analytics.GET("/report-configs", handler.ListReportConfigs)
					analytics.POST("/report-configs", handler.CreateReportConfig)
					analytics.GET("/report-configs/:id", handler.GetReportConfig)
					analytics.PUT("/report-configs/:id", handler.UpdateReportConfig)
					analytics.DELETE("/report-configs/:id", handler.DeleteReportConfig)
				}

				// 用户行为分析
				behavior := adminAuth.Group("/behavior")
				{
					behavior.GET("/trajectories", behaviorHandler.ListTrajectories)
					behavior.GET("/trajectories/:id", behaviorHandler.GetTrajectory)
					behavior.POST("/trajectories", behaviorHandler.SaveTrajectory)
					behavior.POST("/trajectories/:id/analyze", behaviorHandler.AnalyzeTrajectory)
					behavior.GET("/trajectories/visualization", behaviorHandler.GetTrajectoryVisualization)
					behavior.GET("/trajectories/statistics", behaviorHandler.GetTrajectoryStatistics)
					behavior.GET("/trajectories/user/:user_id/summary", behaviorHandler.GetUserTrajectorySummary)

					behavior.GET("/profiles", behaviorHandler.ListProfiles)
					behavior.GET("/profiles/:user_id", behaviorHandler.GetUserProfile)
					behavior.POST("/profiles", behaviorHandler.CreateOrUpdateProfile)

					behavior.GET("/anomalies", behaviorHandler.GetAnomalies)
					behavior.GET("/anomalies/recent", behaviorHandler.GetRecentAnomalies)
					behavior.POST("/anomalies/:id/process", behaviorHandler.ProcessAnomaly)
					behavior.GET("/anomalies/statistics", behaviorHandler.GetAnomalyStatistics)
					behavior.GET("/anomalies/export", behaviorHandler.ExportAnomalies)
					behavior.GET("/anomalies/patterns", behaviorHandler.AnalyzePatterns)

					behavior.GET("/rules", behaviorHandler.ListRules)
					behavior.POST("/rules", behaviorHandler.CreateRule)
					behavior.DELETE("/rules/:id", behaviorHandler.DeleteRule)
					behavior.POST("/rules/:id/toggle", behaviorHandler.ToggleRule)
				}

				// 实时监控
				monitoring := adminAuth.Group("/monitoring")
				{
					monitoring.GET("/data", handler.GetMonitoringData)
					monitoring.GET("/alerts", handler.GetAlerts)
					monitoring.POST("/alerts/:id/acknowledge", handler.AcknowledgeAlert)
					monitoring.GET("/system-metrics", handler.GetSystemMetrics)
					monitoring.GET("/request-metrics", handler.GetRequestMetrics)
					monitoring.GET("/api-stats", handler.GetApiStats)
				}

				// 告警系统
				alerts := adminAuth.Group("/alerts")
				{
					// 告警渠道管理
					alerts.GET("/channels", handler.ListAlertChannels)
					alerts.POST("/channels", handler.CreateAlertChannel)
					alerts.GET("/channels/:id", handler.GetAlertChannel)
					alerts.PUT("/channels/:id", handler.UpdateAlertChannel)
					alerts.DELETE("/channels/:id", handler.DeleteAlertChannel)

					// 告警规则管理
					alerts.GET("/rules", handler.ListAlertRules)
					alerts.POST("/rules", handler.CreateAlertRule)
					alerts.GET("/rules/:id", handler.GetAlertRule)
					alerts.PUT("/rules/:id", handler.UpdateAlertRule)
					alerts.DELETE("/rules/:id", handler.DeleteAlertRule)

					// 告警记录管理
					alerts.GET("", handler.ListAlerts)
					alerts.GET("/:id", handler.GetAlert)
					alerts.POST("/:id/resolve", handler.ResolveAlert)
					alerts.GET("/:id/history", handler.GetAlertHistory)

					// 测试告警
					alerts.POST("/test", handler.SendTestAlert)
				}

				// A/B测试管理
				abTesting := adminAuth.Group("/ab-testing")
				{
					abTesting.GET("/summary", handler.GetABTestSummary)
					abTesting.GET("/active", handler.GetActiveTests)
					abTesting.GET("", handler.ListABTests)
					abTesting.POST("", handler.CreateABTest)
					abTesting.GET("/:id", handler.GetABTest)
					abTesting.PUT("/:id", handler.UpdateABTest)
					abTesting.DELETE("/:id", handler.DeleteABTest)
					abTesting.POST("/:id/start", handler.StartABTest)
					abTesting.POST("/:id/stop", handler.StopABTest)
					abTesting.GET("/:id/report", handler.GetTestReport)
				}

				// 无感验证管理
				seamless := adminAuth.Group("/seamless")
				{
					seamless.GET("/dashboard", handler.GetDashboardStats)
					seamless.GET("/fingerprint", handler.GetFingerprintInfo)
					seamless.GET("/trust", handler.GetTrustedDevices)
					seamless.POST("/trust", handler.TrustDevice)
					seamless.DELETE("/trust/:fingerprint", handler.RevokeTrustedDevice)
				}
			}
		}

		// A/B测试客户端API（无需认证，用于前端获取变体和追踪事件）
		abTestClient := api.Group("/ab-testing")
		{
			abTestClient.POST("/assign", handler.AssignVariant)
			abTestClient.POST("/event", handler.TrackEvent)
		}

		// 示例路由
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})
	}

	return r
}
