package router

import (
	"html/template"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/handler"
	"github.com/hjtpx/hjtpx/internal/api/middleware"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/i18n"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRouter() *gin.Engine {
	r := gin.New()

	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.GzipCompression())
	r.Use(middleware.PerformanceMonitoring())
	r.Use(middleware.RequestID())
	r.Use(i18n.Middleware())

	middleware.SetupSecurityMiddleware(r)

	r.Static("/static", "./static")
	r.Static("/uploads", "./uploads")

	r.SetFuncMap(template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"formatUnixTime": func(t int64) string {
			return time.Unix(t, 0).Format("2006-01-02 15:04:05")
		},
		"safeHTML": func(str string) template.HTML {
			return template.HTML(str)
		},
	})

	r.LoadHTMLGlob(filepath.Join(".", "templates", "*"))

	cfg := config.GetConfig()
	backupHandler := handler.GetBackupHandler(cfg)
	cacheHandler := handler.NewCacheMetricsHandler()
	dbHandler := handler.NewDatabaseMetricsHandler()

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health Check
	r.GET("/health", handler.HealthCheck)

	// API文档
	r.GET("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"code":    0,
			"message": "Welcome to HJTPX API",
			"version": "1.0.0",
		})
	})

	// 主验证码页面
	r.GET("/captcha.html", func(c *gin.Context) {
		c.HTML(200, "captcha.html", gin.H{
			"title": "验证码",
		})
	})

	r.GET("/seamless", func(c *gin.Context) {
		c.HTML(200, "seamless.html", gin.H{
			"title": "无缝验证码",
		})
	})

	r.GET("/lianliankan", func(c *gin.Context) {
		c.HTML(200, "lianliankan.html", gin.H{
			"title": "连连看验证码",
		})
	})

	r.GET("/3d-captcha", func(c *gin.Context) {
		c.HTML(200, "3dcaptcha.html", gin.H{
			"title": "3D 验证码",
		})
	})

	r.GET("/voice-captcha", func(c *gin.Context) {
		c.HTML(200, "voice-captcha.html", gin.H{
			"title": "语音验证码",
		})
	})

	r.GET("/mfa-setup", func(c *gin.Context) {
		c.HTML(200, "mfa-setup.html", gin.H{
			"title": "MFA 设置",
		})
	})

	r.GET("/mfa-verify", func(c *gin.Context) {
		c.HTML(200, "mfa-verify.html", gin.H{
			"title": "MFA 验证",
		})
	})

	r.GET("/websocket-demo", func(c *gin.Context) {
		c.HTML(200, "captcha.html", gin.H{
			"title": "WebSocket 实时验证演示",
		})
	})

	adminRouter := r.Group("/admin")
	{
		adminRouter.GET("/", func(c *gin.Context) {
			c.HTML(200, "dashboard.html", gin.H{
				"title": "仪表盘",
			})
		})

		adminRouter.GET("/dashboard", func(c *gin.Context) {
			c.HTML(200, "dashboard.html", gin.H{
				"title": "仪表盘",
			})
		})

		adminRouter.GET("/stats", func(c *gin.Context) {
			c.HTML(200, "stats.html", gin.H{
				"title": "统计分析",
			})
		})

		adminRouter.GET("/logs", func(c *gin.Context) {
			c.HTML(200, "logs.html", gin.H{
				"title": "验证日志",
			})
		})

		adminRouter.GET("/config", func(c *gin.Context) {
			c.HTML(200, "config.html", gin.H{
				"title": "系统配置",
			})
		})

		adminRouter.GET("/whitelabel", func(c *gin.Context) {
			c.HTML(200, "whitelabel.html", gin.H{
				"title": "主题设置",
			})
		})

		adminRouter.GET("/behavior-analytics", func(c *gin.Context) {
			c.HTML(200, "behavior-analytics.html", gin.H{
				"title": "用户行为分析",
			})
		})

		adminRouter.GET("/adaptive-config", func(c *gin.Context) {
			c.HTML(200, "adaptive-config.html", gin.H{
				"title": "自适应难度配置",
			})
		})

		adminRouter.GET("/rate-limit", func(c *gin.Context) {
			c.HTML(200, "rate-limit.html", gin.H{
				"title": "限流配置",
			})
		})

		adminRouter.GET("/api/dashboard", handler.GetDashboardData)
		adminRouter.GET("/api/recent-verifications", handler.GetRecentVerifications)
		adminRouter.GET("/api/system-status", handler.GetSystemStatus)
		adminRouter.GET("/api/alerts", handler.GetDashboardAlerts)
		adminRouter.GET("/api/dashboard/extended", handler.GetExtendedDashboardStats)
		adminRouter.GET("/export", handler.ExportDashboardData)

		adminRouter.GET("/api/logs", handler.GetVerificationLogs)
		adminRouter.GET("/api/logs/:id", handler.GetLogDetail)
		adminRouter.GET("/api/logs/export", handler.ExportLogs)
		adminRouter.DELETE("/api/logs/clear", handler.DeleteOldLogs)
		adminRouter.GET("/api/logs/session/:session_id", handler.GetLogsBySession)
		adminRouter.GET("/api/logs/statistics", handler.GetLogStatistics)
		adminRouter.GET("/api/config", handler.GetAllConfig)
		adminRouter.PUT("/api/config", handler.UpdateConfig)
		adminRouter.GET("/api/config/export", handler.ExportConfig)
		adminRouter.POST("/api/config/reset", handler.ResetConfig)

		// 白标主题 API
		adminRouter.GET("/api/whitelabel", handler.GetWhitelabelConfig)
	adminRouter.PUT("/api/whitelabel", handler.UpdateWhitelabelConfig)
	adminRouter.GET("/api/whitelabel/css", handler.GetWhitelabelCSS)
	adminRouter.POST("/api/whitelabel/logo/:type", handler.UploadLogo)
	adminRouter.DELETE("/api/whitelabel/logo/:type", handler.DeleteLogo)
	adminRouter.POST("/api/whitelabel/reset", handler.ResetWhitelabelConfig)
	
	// 仪表盘扩展功能
	adminRouter.GET("/api/dashboard/config", handler.GetDashboardConfig)
	adminRouter.PUT("/api/dashboard/config", handler.SaveDashboardConfig)
	adminRouter.PUT("/api/dashboard/theme", handler.UpdateDashboardTheme)
	adminRouter.GET("/api/dashboard/widgets", handler.GetDashboardWidgets)
	adminRouter.PUT("/api/dashboard/widgets", handler.SaveDashboardWidgets)
	
	// 通知功能
	adminRouter.GET("/api/notifications", handler.GetNotifications)
	adminRouter.PUT("/api/notifications/:id/read", handler.MarkNotificationRead)
	adminRouter.PUT("/api/notifications/read-all", handler.MarkAllNotificationsRead)
	adminRouter.DELETE("/api/notifications/:id", handler.DeleteNotification)
	adminRouter.GET("/api/notifications/unread-count", handler.GetUnreadNotificationCount)
	adminRouter.POST("/api/notifications/broadcast", handler.BroadcastNotification)
	
	// 风控规则引擎 - 暂时注释
	// adminRouter.POST("/api/risk-templates/init", handler.InitializeTemplates)
	// adminRouter.GET("/api/risk-templates", handler.GetRuleTemplates)
	// adminRouter.GET("/api/risk-templates/:id", handler.GetRuleTemplate)
	// adminRouter.POST("/api/risk-templates", handler.CreateRuleTemplate)
	// adminRouter.PUT("/api/risk-templates/:id", handler.UpdateRuleTemplate)
	// adminRouter.DELETE("/api/risk-templates/:id", handler.DeleteRuleTemplate)
	// adminRouter.POST("/api/risk-templates/apply", handler.ApplyTemplate)
	
	// adminRouter.GET("/api/risk-rules", handler.GetRiskRules)
	// adminRouter.GET("/api/risk-rules/:id", handler.GetRiskRule)
	// adminRouter.POST("/api/risk-rules", handler.CreateRiskRule)
	// adminRouter.PUT("/api/risk-rules/:id", handler.UpdateRiskRule)
	// adminRouter.DELETE("/api/risk-rules/:id", handler.DeleteRiskRule)
	// adminRouter.PUT("/api/risk-rules/:id/toggle", handler.ToggleRiskRule)
	// adminRouter.POST("/api/risk-rules/:id/evaluate", handler.EvaluateRule)
	
	// adminRouter.GET("/api/risk-rules/:id/trigger-history", handler.GetRuleTriggerHistories)
	// adminRouter.GET("/api/risk-rules/:id/performance", handler.GetRulePerformance)
	// adminRouter.GET("/api/risk-rules/performance-overview", handler.GetAllRulesPerformance)
	// adminRouter.GET("/api/risk-rules/audit-logs", handler.GetRuleAuditLogs)
}

	api := r.Group("/api/v1")
	{
		captcha := api.Group("/captcha")
		{
			captcha.GET("/slider", handler.GetSliderCaptcha)
			captcha.GET("/click", handler.GetClickCaptcha)
			captcha.POST("/verify", handler.VerifyCaptcha)
			captcha.GET("/gesture", handler.GenerateGestureCaptcha)
			captcha.POST("/gesture/verify", handler.VerifyGestureCaptcha)
			captcha.GET("/jigsaw", handler.GenerateJigsawCaptcha)
			captcha.POST("/jigsaw/verify", handler.VerifyJigsawCaptcha)

			captcha.POST("/create", handler.CreateSliderCaptcha)
			captcha.POST("/verify-v2", handler.VerifySliderCaptcha)
			captcha.GET("/status/:session_id", handler.GetSliderCaptchaStatus)
			captcha.GET("/check/:session_id", handler.CheckSliderCaptchaValid)
			captcha.POST("/lianliankan/create", handler.CreateLianLianKanCaptcha)
			captcha.POST("/lianliankan/verify", handler.VerifyLianLianKanCaptcha)
			captcha.GET("/lianliankan/status/:session_id", handler.GetLianLianKanCaptchaStatus)
			captcha.GET("/lianliankan/check/:session_id", handler.CheckLianLianKanCaptchaValid)

			captcha.POST("/voice/create", handler.CreateVoiceCaptcha)
			captcha.POST("/voice/verify", handler.VerifyVoiceCaptcha)

			captcha.POST("/3d/create", handler.CreateThreeDCaptcha)
			captcha.POST("/3d/verify", handler.VerifyThreeDCaptcha)
			captcha.GET("/3d/status/:sessionID", handler.GetThreeDCaptchaStatus)
			captcha.GET("/3d/check/:sessionID", handler.CheckThreeDCaptchaValid)
			
			// 多因素滑块验证
			captcha.POST("/verify-multi-factor", handler.VerifySliderWithMultiFactor)
		}

		// 高级限流 API
		// advancedRateLimit := api.Group("/advanced-rate-limit")
		// {
		// 	// arHandler := handler.NewAdvancedRateLimitHandler()
		//
		// 	// 令牌桶限流
		// 	tokenBucket := advancedRateLimit.Group("/token-bucket")
		// 	{
		// 		// tokenBucket.POST("/check", arHandler.CheckTokenBucket)
		// 		// tokenBucket.POST("/reset", arHandler.ResetTokenBucket)
		// 		// tokenBucket.GET("/stats", arHandler.GetBucketStats)
		// 	}
		//
		// 	// 配额管理
		// 	quota := advancedRateLimit.Group("/quota")
		// 	{
		// 		quota.POST("/create", arHandler.CreateQuota)
		// 		quota.GET("/status", arHandler.GetQuotaStatus)
		// 		quota.POST("/consume", arHandler.ConsumeQuota)
		// 		quota.POST("/reset", arHandler.ResetQuota)
		// 		quota.DELETE("/delete", arHandler.DeleteQuota)
		// 		quota.GET("/list", arHandler.ListQuotas)
		// 	}
		//
		// 	// 综合限流
		// 	advancedRateLimit.POST("/combined-check", arHandler.CombinedCheck)
		// }

		// WebSocket 验证路由
		websocket := api.Group("/websocket")
		{
			websocket.GET("/verify", handler.WebSocketVerificationHandler)
			websocket.GET("/stats", handler.GetWebSocketStats)
			websocket.POST("/broadcast", handler.BroadcastWebSocketMessage)
		}

		auth := api.Group("/auth")
		{
			auth.POST("/register", handler.Register)
			auth.POST("/login", handler.Login)
			auth.POST("/logout", handler.Logout)
			auth.GET("/profile", handler.GetProfile)
			auth.PUT("/profile", handler.UpdateProfile)
			auth.POST("/change-password", handler.ChangePassword)
			auth.POST("/refresh-token", handler.RefreshToken)
		}

		biometrics := api.Group("/biometrics")
		{
			biometrics.POST("/register", handler.RegisterBiometricProfile)
			biometrics.POST("/verify", handler.VerifyBiometrics)
			biometrics.GET("/profile", handler.GetBiometricProfile)
		}

		adaptive := api.Group("/adaptive")
		{
			adaptive.GET("/difficulty", handler.GetUserDifficulty)
			adaptive.POST("/result", handler.UpdateUserResult)
			adaptive.GET("/config", handler.GetAdaptiveConfig)
			adaptive.PUT("/config", handler.UpdateAdaptiveConfig)
			adaptive.GET("/profiles", handler.GetAllAdaptiveProfiles)
			adaptive.POST("/flag", handler.AddBehaviorFlag)
			adaptive.GET("/captcha-difficulty", handler.GetDifficultyForCaptcha)
		}

		behavior := api.Group("/behavior")
		{
			behavior.GET("/heatmap", handler.GetBehaviorHeatmap)
			behavior.GET("/trajectories", handler.GetUserTrajectories)
			behavior.GET("/anomalies", handler.GetBehaviorAnomalies)
			behavior.GET("/risk-distribution", handler.GetRiskDistribution)
			behavior.GET("/export", handler.ExportBehaviorData)
			behavior.POST("/trajectory/replay", handler.ReplayTrajectory)
		}

		verify := api.Group("/verify")
		{
			verify.POST("/email", handler.VerifyEmail)
			verify.POST("/phone", handler.VerifyPhone)
		}

		// MFA 路由
		mfa := api.Group("/mfa")
		mfa.Use(middleware.AuthMiddleware())
		{
			mfa.GET("/status", handler.GetMFAStatusHandler)
			mfa.GET("/totp/generate", handler.GenerateTOTPHandler)
			mfa.POST("/totp/verify", handler.VerifyTOTPHandler)
			mfa.POST("/totp/enable", handler.EnableTOTPHandler)
			mfa.POST("/sms/send", handler.SendSMSCodeHandler)
			mfa.POST("/email/send", handler.SendEmailCodeHandler)
			mfa.POST("/code/verify", handler.VerifyCodeHandler)
			mfa.POST("/enable", handler.EnableMFAHandler)
			mfa.POST("/disable", handler.DisableMFAHandler)
			mfa.GET("/backup-codes/generate", handler.GenerateBackupCodesHandler)
			mfa.POST("/backup-codes/verify", handler.VerifyBackupCodeHandler)
		}

		// GDPR 路由
		gdpr := api.Group("/gdpr")
		gdpr.Use(middleware.AuthMiddleware())
		{
			gdpr.GET("/consent", handler.GetGDPRHandler().GetConsent)
			gdpr.PUT("/consent", handler.GetGDPRHandler().UpdateConsent)
			gdpr.POST("/data-export", handler.GetGDPRHandler().RequestDataExport)
			gdpr.GET("/data-export/:id", handler.GetGDPRHandler().GetExportStatus)
			gdpr.GET("/data-export/:id/download", handler.GetGDPRHandler().DownloadExport)
			gdpr.POST("/data-deletion", handler.GetGDPRHandler().RequestDataDeletion)
			gdpr.GET("/data-deletion/:id", handler.GetGDPRHandler().GetDeletionStatus)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddleware())
		{
			admin.GET("/stats", handler.GetStats)
			admin.GET("/stats/detailed", handler.GetDetailedStats)
			admin.GET("/users", handler.ListUsers)
			admin.POST("/users", handler.CreateUser)
			admin.PUT("/users/:id", handler.UpdateUser)
			admin.DELETE("/users/:id", handler.DeleteUser)
			admin.PUT("/users/:id/status", handler.UpdateUserStatus)
			admin.POST("/users/:id/reset-password", handler.ResetUserPassword)

			admin.GET("/applications", handler.ListApplications)
			admin.POST("/applications", handler.CreateApplication)
			admin.PUT("/applications/:id", handler.UpdateApplication)
			admin.DELETE("/applications/:id", handler.DeleteApplication)
			admin.POST("/applications/:id/approve", handler.ApproveApplication)
			admin.POST("/applications/:id/reject", handler.RejectApplication)

			admin.GET("/api-keys", handler.ListAPIKeys)
			admin.POST("/api-keys", handler.CreateAPIKey)
			admin.DELETE("/api-keys/:id", handler.DeleteAPIKey)
			admin.POST("/api-keys/:id/regenerate", handler.RegenerateAPIKey)

			admin.GET("/verifications", handler.ListVerifications)
			admin.GET("/verifications/:id", handler.GetVerificationDetail)
			admin.POST("/verifications/:id/review", handler.ReviewVerification)

			admin.GET("/blacklist", handler.ListBlacklist)
			admin.POST("/blacklist", handler.AddToBlacklist)
			admin.DELETE("/blacklist/:id", handler.RemoveFromBlacklist)

			admin.GET("/settings", handler.GetSettings)
			admin.PUT("/settings", handler.UpdateSettings)

			admin.GET("/risk-events", handler.ListRiskEvents)
			admin.GET("/risk-events/:id", handler.GetRiskEventDetail)

			admin.GET("/traces", handler.ListTraces)
			admin.GET("/traces/:id", handler.GetTraceDetail)

			admin.GET("/alerts/channels", handler.ListAlertChannels)
			admin.POST("/alerts/channels", handler.CreateAlertChannel)
			admin.PUT("/alerts/channels/:id", handler.UpdateAlertChannel)
			admin.DELETE("/alerts/channels/:id", handler.DeleteAlertChannel)

			admin.GET("/alerts/rules", handler.ListAlertRules)
			admin.POST("/alerts/rules", handler.CreateAlertRule)
			admin.PUT("/alerts/rules/:id", handler.UpdateAlertRule)
			admin.DELETE("/alerts/rules/:id", handler.DeleteAlertRule)
			admin.POST("/alerts/rules/:id/enable", handler.EnableAlertRule)
			admin.POST("/alerts/rules/:id/disable", handler.DisableAlertRule)

			admin.GET("/alerts/history", handler.ListAlertHistory)
			admin.POST("/alerts/history/:id/acknowledge", handler.AcknowledgeAlert)
			admin.POST("/alerts/history/:id/resolve", handler.ResolveAlert)

			// 行为分析
			admin.GET("/behavior-analytics", handler.GetBehaviorAnalytics)

			// 深度学习轨迹分析模块
			admin.GET("/model-performance", handler.GetModelPerformanceReport)
			admin.POST("/model-update/queue", handler.QueueModelUpdate)
			admin.POST("/model-update/:action", handler.ToggleOnlineUpdate)
			admin.POST("/trajectory-visualization", handler.GetTrajectoryVisualization)
			admin.GET("/slider-difficulty", handler.GetCurrentDifficulty)
			admin.POST("/slider-difficulty/adjust", handler.AdjustDifficulty)
			admin.GET("/security-assessment", handler.GetSecurityReport)
			admin.POST("/security-assessment", handler.PerformSecurityAssessment)
			admin.POST("/record-prediction", handler.RecordModelPrediction)

			// 备份管理
			admin.GET("/backups", backupHandler.ListBackups)
			admin.POST("/backups", backupHandler.CreateBackup)
			admin.GET("/backups/:id", backupHandler.GetBackup)
			admin.DELETE("/backups/:id", backupHandler.DeleteBackup)
			admin.POST("/backups/:id/restore", backupHandler.RestoreBackup)
			admin.POST("/backups/:id/verify", backupHandler.VerifyBackup)
			admin.POST("/backups/cleanup", backupHandler.CleanupOldBackups)
			admin.GET("/backup-config", backupHandler.GetBackupConfig)

			// 缓存性能监控与优化
			admin.GET("/cache/health", cacheHandler.GetCacheHealth)
			admin.GET("/cache/metrics", cacheHandler.GetCacheDetailedMetrics)
			admin.GET("/cache/hot-keys", cacheHandler.GetCacheHotKeys)
			admin.GET("/cache/latency", cacheHandler.GetCacheLatencyDistribution)
			admin.GET("/cache/memory-trend", cacheHandler.GetCacheMemoryTrend)
			admin.GET("/cache/alerts", cacheHandler.GetCacheAlerts)
			admin.POST("/cache/alerts/acknowledge", cacheHandler.AcknowledgeCacheAlert)
			admin.POST("/cache/alerts/clear", cacheHandler.ClearCacheAlerts)
			admin.POST("/cache/metrics/reset", cacheHandler.ResetCacheMetrics)
			admin.POST("/cache/warmup", cacheHandler.TriggerCacheWarmup)
			admin.GET("/cache/warmup-status", cacheHandler.GetCacheWarmupStatus)
			admin.GET("/cache/consistency-status", cacheHandler.GetCacheConsistencyStatus)

			// 数据库性能监控与优化
			admin.GET("/database/health", dbHandler.GetDatabaseHealth)
			admin.GET("/database/slow-queries", dbHandler.GetSlowQueries)
			admin.GET("/database/top-queries", dbHandler.GetTopQueries)
			admin.GET("/database/query-distribution", dbHandler.GetQueryDistribution)
			admin.GET("/database/performance-report", dbHandler.GeneratePerformanceReport)
			admin.GET("/database/optimization-suggestions", dbHandler.GetOptimizationSuggestions)
			admin.POST("/database/clear-metrics", dbHandler.ClearPerformanceMetrics)

			arHandler := handler.GetAdvancedRateLimitHandler()
			arHandler.RegisterRoutes(admin)
		}
	}

	return r
}
