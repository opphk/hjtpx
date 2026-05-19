package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/handler"
	"github.com/hjtpx/hjtpx/internal/api/middleware"
)

var aiModelV3Handler *handler.AIModelV3Handler

func SetupRoutes(r *gin.Engine) {
	aiModelV3Handler = handler.NewAIModelV3Handler()

	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())

	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		// 滑块验证码
		api.POST("/captcha/slider/create", handler.CreateSliderCaptcha)
		api.POST("/captcha/slider/verify", handler.VerifySliderCaptcha)
		api.GET("/captcha/slider/status/:session_id", handler.GetSliderCaptchaStatus)
		api.GET("/captcha/slider/check/:session_id", handler.CheckSliderCaptchaValid)

		// 点选验证码
		api.GET("/captcha/click", handler.GetClickCaptcha)
		api.POST("/captcha/click/verify", handler.VerifyCaptcha)

		// 图形验证码
		api.GET("/captcha/image", handler.GetSliderCaptcha)
		api.POST("/captcha/image/verify", handler.VerifyCaptcha)

		// 手势验证码
		api.GET("/captcha/gesture", handler.GenerateGestureCaptcha)
		api.POST("/captcha/gesture/verify", handler.VerifyGestureCaptcha)
		api.GET("/captcha/gesture/status/:session_id", handler.GetGestureCaptchaStatus)
		api.GET("/captcha/gesture/grid", handler.GetGestureGridPoints)

		// 旋转验证码
		api.POST("/captcha/rotate/create", handler.GenerateRotateCaptcha)
		api.POST("/captcha/rotate/verify", handler.VerifyRotateCaptcha)

		// 拼图验证码
		api.POST("/captcha/puzzle/create", handler.GeneratePuzzleCaptcha)
		api.POST("/captcha/puzzle/verify", handler.VerifyPuzzleCaptcha)
		api.GET("/captcha/puzzle/status/:session_id", handler.GetPuzzleCaptchaStatus)

		// 表情识别验证码
		api.POST("/captcha/emoji/create", handler.CreateEmojiCaptcha)
		api.POST("/captcha/emoji/verify", handler.VerifyEmojiCaptcha)

		// 语义理解验证码
		api.POST("/captcha/semantic/create", handler.CreateSemanticCaptcha)
		api.POST("/captcha/semantic/verify", handler.VerifySemanticCaptcha)
		api.GET("/captcha/semantic/status/:session_id", handler.GetSemanticCaptchaStatus)

		// 组合验证码
		api.POST("/captcha/combo/create", handler.CreateComboCaptcha)
		api.POST("/captcha/combo/verify", handler.VerifyComboCaptcha)
		api.GET("/captcha/combo/status/:session_id", handler.GetComboCaptchaStatus)

		// ============ v17.0 新增验证码路由 ============
		// 视频验证码
		api.POST("/captcha/video/generate", handler.VideoCaptchaGenerate)
		api.POST("/captcha/video/verify", handler.VideoCaptchaVerify)
		api.GET("/captcha/video/options", handler.VideoCaptchaOptions)

		// AR验证码
		api.POST("/captcha/ar/generate", handler.ARCaptchaGenerate)
		api.POST("/captcha/ar/verify", handler.ARCaptchaVerify)
		api.GET("/captcha/ar/options", handler.ARCaptchaOptions)

		// 增强的组合验证码系统
		api.POST("/captcha/combo/generate", handler.ComboCaptchaGenerate)
		api.POST("/captcha/combo/verify", handler.ComboCaptchaVerify)
		api.GET("/captcha/combo/options", handler.ComboCaptchaOptions)

		// ============ v17.0 新增 AI 模型 v3 路由 ============
		api.POST("/ai/v3/smart-captcha/generate", aiModelV3Handler.GenerateSmartCaptcha)
		api.POST("/ai/v3/risk-assessment", aiModelV3Handler.ComprehensiveRiskAssessment)
		api.POST("/ai/v3/feedback", aiModelV3Handler.RecordFeedback)
		api.GET("/ai/v3/stats", aiModelV3Handler.GetLearningStats)

		// ============ v17.0 新增高级加密模块路由 ============
		api.POST("/crypto/v2/generate-key", handler.GenerateAdvancedKey)
		api.POST("/crypto/v2/encrypt", handler.EncryptAdvanced)
		api.POST("/crypto/v2/decrypt", handler.DecryptAdvanced)
		api.POST("/crypto/v2/quantum-hash", handler.GenerateQuantumHash)
		api.GET("/crypto/v2/keys", handler.GetActiveKeys)

		// 统一验证码验证接口
		api.POST("/captcha/verify", handler.VerifyCaptcha)

		// ============ 管理员认证路由 ============
		admin := api.Group("/admin")
		{
			// 登录
			admin.POST("/login", handler.Login)
			// 刷新token
			admin.POST("/refresh-token", handler.RefreshTokenHandler)
			// 登出
			admin.POST("/logout", middleware.AuthMiddleware(), handler.Logout)
			// 获取当前用户信息
			admin.GET("/me", middleware.AuthMiddleware(), handler.GetCurrentUser)
			// 修改密码
			admin.POST("/change-password", middleware.AuthMiddleware(), handler.ChangePassword)
			// 获取登录历史
			admin.GET("/login-history", middleware.AuthMiddleware(), handler.GetLoginHistory)
			// 获取管理员仪表盘数据
			admin.GET("/dashboard", middleware.AuthMiddleware(), handler.AdminDashboard)

			// ============ 应用管理路由 ============
			apps := admin.Group("/applications", middleware.AuthMiddleware())
			{
				apps.GET("", handler.ListApplications)
				apps.POST("", handler.CreateApplication)
				apps.GET("/:id", handler.GetApplication)
				apps.PUT("/:id", handler.UpdateApplication)
				apps.DELETE("/:id", handler.DeleteApplication)
				apps.POST("/:id/rotate-key", handler.RegenerateApplicationKey)
				apps.GET("/:id/statistics", handler.GetApplicationStatistics)
			}

			// ============ 日志管理路由 ============
			logs := admin.Group("/logs", middleware.AuthMiddleware())
			{
				logs.GET("", handler.GetVerificationLogs)
				logs.GET("/:id", handler.GetLogDetail)
				logs.POST("/search", handler.AdvancedSearchLogs)
				logs.GET("/export", handler.ExportLogs)
				logs.GET("/session/:session_id", handler.GetLogsBySession)
				logs.GET("/statistics", handler.GetLogStatistics)
				logs.POST("/save-search", handler.SaveLogSearch)
				logs.GET("/saved-searches", handler.GetSavedLogSearches)
				logs.DELETE("/saved-searches/:id", handler.DeleteSavedLogSearch)
			}

			// ============ 风控规则路由 ============
			rules := admin.Group("/risk-rules", middleware.AuthMiddleware())
			{
				rules.GET("", handler.ListRiskRules)
				rules.POST("", handler.CreateRiskRule)
				rules.GET("/:id", handler.GetRiskRule)
				rules.PUT("/:id", handler.UpdateRiskRule)
				rules.DELETE("/:id", handler.DeleteRiskRule)
			}

			// ============ 统计分析路由 ============
			stats := admin.Group("/statistics", middleware.AuthMiddleware())
			{
				stats.GET("/overview", handler.GetDashboardStats)
				stats.GET("/verification-trend", handler.GetTrendData)
				stats.GET("/risk-distribution", handler.GetRiskDistribution)
				stats.GET("/performance", handler.GetVerificationStats)
				stats.GET("/realtime", handler.GetRealtimeStats)
			}

			// ============ A/B测试路由 ============
			abtest := admin.Group("/ab-testing", middleware.AuthMiddleware())
			{
				abtest.GET("", handler.ListABTests)
				abtest.POST("", handler.CreateABTest)
				abtest.GET("/:id", handler.GetABTest)
				abtest.PUT("/:id", handler.UpdateABTest)
				abtest.DELETE("/:id", handler.DeleteABTest)
				abtest.POST("/:id/start", handler.StartABTest)
				abtest.POST("/:id/stop", handler.StopABTest)
				abtest.GET("/:id/report", handler.GetTestReport)
				abtest.GET("/active", handler.GetActiveTests)
				abtest.GET("/summary", handler.GetABTestSummary)
			}
		}
	}
}
