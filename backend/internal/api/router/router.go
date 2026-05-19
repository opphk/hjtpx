package router

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/handler"
	"github.com/hjtpx/hjtpx/internal/api/middleware"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/i18n"
	"github.com/hjtpx/hjtpx/pkg/metrics"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var once sync.Once
var routerInstance *gin.Engine

func SetupRouter() *gin.Engine {
	once.Do(func() {
		r := gin.New()

		r.HandleMethodNotAllowed = true
		r.RemoveExtraSlash = true

		r.Use(middleware.Logger())
		r.Use(middleware.Recovery())
		r.Use(middleware.ErrorHandler())
		r.Use(middleware.GzipCompression())
		r.Use(middleware.PerformanceMonitoring())
		r.Use(middleware.RequestID())
		r.Use(i18n.Middleware())
		r.Use(metrics.PrometheusMiddleware())

		middleware.SetupSecurityMiddleware(r)

		setupStaticFiles(r)

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
		setupRoutes(r)
		routerInstance = r
	})
	return routerInstance
}

func setupStaticFiles(r *gin.Engine) {
	staticConfig := static.LocalFile("./static", true)
	r.Use(static.Serve("/static", staticConfig))
	r.Use(static.Serve("/uploads", static.LocalFile("./uploads", true)))

	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusMethodNotAllowed)
			return
		}
		c.Status(http.StatusNotFound)
	})
}

func setupRoutes(r *gin.Engine) {
	cfg := config.GetConfig()
	backupHandler := handler.GetBackupHandler(cfg)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/docs", func(c *gin.Context) {
		c.HTML(200, "docs.html", gin.H{})
	})
	r.GET("/docs/error_codes", func(c *gin.Context) {
		content, err := os.ReadFile("docs/error_codes.md")
		if err != nil {
			c.String(500, "Failed to read error codes documentation")
			return
		}
		c.Header("Content-Type", "text/markdown; charset=utf-8")
		c.String(200, string(content))
	})

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

	metrics.RegisterMetricsEndpoint(r)

	setupHTMLRoutes(r)
	setupAdminRoutes(r)
	setupAPIRoutes(r, backupHandler)
}

func setupHTMLRoutes(r *gin.Engine) {
	r.GET("/lianliankan", func(c *gin.Context) {
		c.HTML(200, "lianliankan.html", gin.H{"title": "连连看验证码"})
	})

	r.GET("/3d-captcha", func(c *gin.Context) {
		c.HTML(200, "3dcaptcha.html", gin.H{"title": "3D 验证码"})
	})

	r.GET("/voice-captcha", func(c *gin.Context) {
		c.HTML(200, "voice-captcha.html", gin.H{"title": "语音验证码"})
	})

	r.GET("/mfa-setup", func(c *gin.Context) {
		c.HTML(200, "mfa-setup.html", gin.H{"title": "MFA 设置"})
	})

	r.GET("/mfa-verify", func(c *gin.Context) {
		c.HTML(200, "mfa-verify.html", gin.H{"title": "MFA 验证"})
	})

	r.GET("/websocket-demo", func(c *gin.Context) {
		c.HTML(200, "captcha.html", gin.H{"title": "WebSocket 实时验证演示"})
	})

	r.GET("/emoji-captcha", func(c *gin.Context) {
		c.HTML(200, "emoji.html", gin.H{"title": "表情验证码"})
	})
}

func setupAdminRoutes(r *gin.Engine) {
	adminRouter := r.Group("/admin")

	adminRouter.GET("/", func(c *gin.Context) {
		c.HTML(200, "dashboard.html", gin.H{"title": "仪表盘"})
	})
	adminRouter.GET("/dashboard", func(c *gin.Context) {
		c.HTML(200, "dashboard.html", gin.H{"title": "仪表盘"})
	})
	adminRouter.GET("/stats", func(c *gin.Context) {
		c.HTML(200, "stats.html", gin.H{"title": "统计分析"})
	})
	adminRouter.GET("/logs", func(c *gin.Context) {
		c.HTML(200, "logs.html", gin.H{"title": "验证日志"})
	})
	adminRouter.GET("/config", func(c *gin.Context) {
		c.HTML(200, "config.html", gin.H{"title": "系统配置"})
	})
	adminRouter.GET("/whitelabel", func(c *gin.Context) {
		c.HTML(200, "whitelabel.html", gin.H{"title": "主题设置"})
	})
	adminRouter.GET("/behavior-analytics", func(c *gin.Context) {
		c.HTML(200, "behavior-analytics.html", gin.H{"title": "用户行为分析"})
	})
	adminRouter.GET("/adaptive-config", func(c *gin.Context) {
		c.HTML(200, "adaptive-config.html", gin.H{"title": "自适应难度配置"})
	})
	adminRouter.GET("/rate-limit", func(c *gin.Context) {
		c.HTML(200, "rate-limit.html", gin.H{"title": "限流配置"})
	})
	adminRouter.GET("/ab-test", func(c *gin.Context) {
		c.HTML(200, "ab-test.html", gin.H{"title": "A/B 测试管理"})
	})
	adminRouter.GET("/user-profile", func(c *gin.Context) {
		c.HTML(200, "user-profile.html", gin.H{"title": "用户画像分析"})
	})

	adminRouter.GET("/api/dashboard", handler.GetDashboardData)
	adminRouter.GET("/api/recent-verifications", handler.GetRecentVerifications)
	adminRouter.GET("/api/system-status", handler.GetSystemStatus)
	adminRouter.GET("/api/alerts", handler.GetDashboardAlerts)
	adminRouter.GET("/api/dashboard/extended", handler.GetExtendedDashboardStats)
	adminRouter.GET("/export", handler.ExportDashboardData)
	adminRouter.GET("/api/attack-distribution", handler.GetAttackTypeDistribution)
	adminRouter.GET("/api/risk-score-distribution", handler.GetDashboardRiskScoreDistribution)
	adminRouter.GET("/api/dashboard/ws", handler.DashboardWebSocketHandler)

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

	// 白标主题 API
	adminRouter.GET("/api/whitelabel", handler.GetWhitelabelConfig)
	adminRouter.PUT("/api/whitelabel", handler.UpdateWhitelabelConfig)
	adminRouter.GET("/api/whitelabel/css", handler.GetWhitelabelCSS)
	adminRouter.POST("/api/whitelabel/logo/:type", handler.UploadLogo)
	adminRouter.DELETE("/api/whitelabel/logo/:type", handler.DeleteLogo)
	adminRouter.POST("/api/whitelabel/reset", handler.ResetWhitelabelConfig)
}

func setupAPIRoutes(r *gin.Engine, backupHandler *handler.BackupHandler) {
	api := r.Group("/api/v1")

	setupCaptchaRoutes(api)
	setupWebSocketRoutes(api)
	setupAuthRoutes(api)
	setupBiometricsRoutes(api)
	setupAdaptiveRoutes(api)
	setupBehaviorRoutes(api)
	setupVerifyRoutes(api)
	setupMFARoutes(api)
	// GDPR routes暂时禁用，等待handler实现完成
	// setupGDPRRoutes(api)
	setupAPIV1AdminRoutes(api, backupHandler)
}

func setupCaptchaRoutes(api *gin.RouterGroup) {
	captcha := api.Group("/captcha")
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

	// 表情验证码
	captcha.POST("/emoji/create", handler.CreateEmojiCaptcha)
	captcha.POST("/emoji/verify", handler.VerifyEmojiCaptcha)

	// 多因素滑块验证
	captcha.POST("/verify-multi-factor", handler.VerifySliderWithMultiFactor)
}

func setupWebSocketRoutes(api *gin.RouterGroup) {
	websocket := api.Group("/websocket")
	websocket.GET("/verify", handler.WebSocketVerificationHandler)
	websocket.GET("/stats", handler.GetWebSocketStats)
	websocket.POST("/broadcast", handler.BroadcastWebSocketMessage)
}

func setupAuthRoutes(api *gin.RouterGroup) {
	auth := api.Group("/auth")
	auth.POST("/register", handler.Register)
	auth.POST("/login", handler.Login)
	auth.POST("/logout", handler.Logout)
	auth.GET("/profile", handler.GetProfile)
	auth.PUT("/profile", handler.UpdateProfile)
	auth.POST("/change-password", handler.ChangePassword)
	auth.POST("/refresh-token", handler.RefreshToken)
}

func setupBiometricsRoutes(api *gin.RouterGroup) {
	biometrics := api.Group("/biometrics")
	biometrics.POST("/register", handler.RegisterBiometricProfile)
	biometrics.POST("/verify", handler.VerifyBiometrics)
	biometrics.GET("/profile", handler.GetBiometricProfile)
}

func setupAdaptiveRoutes(api *gin.RouterGroup) {
	adaptive := api.Group("/adaptive")
	adaptive.GET("/difficulty", handler.GetUserDifficulty)
	adaptive.POST("/result", handler.UpdateUserResult)
	adaptive.GET("/config", handler.GetAdaptiveConfig)
	adaptive.PUT("/config", handler.UpdateAdaptiveConfig)
	adaptive.GET("/profiles", handler.GetAllAdaptiveProfiles)
	adaptive.POST("/flag", handler.AddBehaviorFlag)
	adaptive.GET("/captcha-difficulty", handler.GetDifficultyForCaptcha)
}

func setupBehaviorRoutes(api *gin.RouterGroup) {
	behavior := api.Group("/behavior")
	behavior.GET("/heatmap", handler.GetBehaviorHeatmap)
	behavior.GET("/trajectories", handler.GetUserTrajectories)
	behavior.GET("/anomalies", handler.GetBehaviorAnomalies)
	behavior.GET("/risk-distribution", handler.GetRiskDistribution)
	behavior.GET("/export", handler.ExportBehaviorData)
	behavior.POST("/trajectory/replay", handler.ReplayTrajectory)
}

func setupVerifyRoutes(api *gin.RouterGroup) {
	verify := api.Group("/verify")
	verify.POST("/email", handler.VerifyEmail)
	verify.POST("/phone", handler.VerifyPhone)
}

func setupMFARoutes(api *gin.RouterGroup) {
	mfa := api.Group("/mfa")
	mfa.Use(middleware.AuthMiddleware())
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

func setupAPIV1AdminRoutes(api *gin.RouterGroup, backupHandler *handler.BackupHandler) {
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware())

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

	arHandler := handler.GetAdvancedRateLimitHandler()
	arHandler.RegisterRoutes(admin)

	admin.GET("/ab-tests", handler.ListABTests)
	admin.GET("/ab-tests/summary", handler.GetABTestSummary)
	admin.GET("/ab-tests/:id", handler.GetABTest)
	admin.POST("/ab-tests", handler.CreateABTest)
	admin.PUT("/ab-tests/:id", handler.UpdateABTest)
	admin.DELETE("/ab-tests/:id", handler.DeleteABTest)
	admin.POST("/ab-tests/:id/start", handler.StartABTest)
	admin.POST("/ab-tests/:id/stop", handler.StopABTest)
	admin.GET("/ab-tests/:id/report", handler.GetTestReport)
	admin.GET("/ab-tests/:id/compare", handler.CompareVariants)
	admin.GET("/ab-tests/:id/variant/:variantId/analytics", handler.GetVariantAnalytics)
	admin.GET("/ab-tests/:id/recommendations", handler.GetTestRecommendations)
}
