package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/handler"
	"github.com/hjtpx/hjtpx/internal/api/middleware"
)

func SetupCDNRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/cdn")
	{
		// 区域管理
		regions := api.Group("/regions")
		{
			regions.GET("", handler.GetRegions)
			regions.GET("/:region_id", handler.GetRegion)
			regions.POST("", handler.CreateRegion)
			regions.PUT("/:region_id", handler.UpdateRegion)
			regions.DELETE("/:region_id", handler.DeleteRegion)
			regions.GET("/:region_id/stats", handler.GetRegionStats)
		}

		// 全局统计
		api.GET("/stats/global", handler.GetGlobalStats)

		// 节点管理
		nodes := api.Group("/nodes")
		{
			nodes.GET("", handler.GetNodes)
			nodes.GET("/:node_id", handler.GetNode)
			nodes.POST("", handler.RegisterNode)
			nodes.DELETE("/:node_id", handler.RemoveNode)
			nodes.GET("/healthy", handler.GetHealthyNodes)
			nodes.PUT("/:node_id/health", handler.UpdateNodeHealth)
		}

		// 边缘函数
		api.POST("/edge-function/:function_name", handler.ExecuteEdgeFunction)

		// 缓存管理
		cache := api.Group("/cache")
		{
			cache.GET("/stats", handler.GetCacheStats)
			cache.POST("/clear", handler.ClearCache)
			cache.POST("/purge", handler.PurgeCache)
			cache.POST("/warmup", handler.WarmupCache)
		}

		// 智能路由
		routing := api.Group("/routing")
		{
			routing.GET("/location", handler.GetClientLocation)
			routing.GET("/decision", handler.GetRoutingDecision)
		}
	}
}

func SetupRoutes(r *gin.Engine) {
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.AuditLogMiddleware())

	SetupCDNRoutes(r)

	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		// ============ SSO路由 ============
		handler.RegisterSSORoutes(api)
		handler.RegisterOIDCRoutes(api)

		// ============ SCIM路由 ============
		handler.RegisterSCIMRoutes(api)

		// ============ 审计日志路由 ============
		handler.RegisterAuditLogRoutes(api)

		// ============ 合规报告路由 ============
		handler.RegisterComplianceRoutes(api)

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

		// 组合验证流程 API
		combo := api.Group("/combo-verification")
		{
			// 验证流程管理
			combo.POST("/flow", handler.CreateComboVerificationFlow)
			combo.GET("/flow/:flow_id", handler.GetComboVerificationFlow)
			combo.DELETE("/flow/:flow_id", handler.DeleteComboVerificationFlow)
			
			// 步骤验证
			combo.POST("/verify/step", handler.VerifyComboVerificationStep)
			combo.POST("/verify/all", handler.VerifyComboVerificationAll)
		}

		// 智能验证码选择 API
		smart := api.Group("/smart-captcha")
		{
			smart.POST("/select", handler.SelectSmartCaptcha)
			smart.POST("/select-multiple", handler.SelectMultipleCaptchas)
			smart.GET("/capabilities", handler.GetAllCaptchaCapabilities)
			smart.GET("/capabilities/:captcha_type", handler.GetCaptchaCapability)
			smart.GET("/analyze/:user_id", handler.AnalyzeUserCaptchaHistory)
		}

		// 动态难度调整 API
		difficulty := api.Group("/dynamic-difficulty")
		{
			difficulty.POST("/behavior", handler.UpdateBehaviorAndAdjustDifficulty)
			difficulty.POST("/behavior/batch", handler.BatchUpdateBehavior)
			difficulty.GET("/user/:user_id", handler.GetDynamicUserDifficulty)
			difficulty.POST("/risk-score", handler.SetRiskScore)
			difficulty.GET("/report/:user_id", handler.GetDifficultyAnalysisReport)
			difficulty.GET("/stats", handler.GetGlobalDifficultyStats)
			difficulty.POST("/session/refresh/:user_id", handler.RefreshDifficultySession)
			difficulty.GET("/history/:user_id", handler.GetAdjustmentHistory)
		}

		// AR验证码
		api.POST("/captcha/ar/create", handler.CreateARCaptcha)
		api.POST("/captcha/ar/verify", handler.VerifyARCaptcha)
		api.GET("/captcha/ar/status/:sessionID", handler.GetARCaptchaStatus)
		api.GET("/captcha/ar/check/:sessionID", handler.CheckARCaptchaValid)
		api.GET("/captcha/ar/webxr-support", handler.GetWebXRSupport)

		// 视频验证码
		api.POST("/captcha/video/create", handler.CreateVideoCaptcha)
		api.POST("/captcha/video/verify", handler.VerifyVideoCaptcha)
		api.GET("/captcha/video/status/:session_id", handler.GetVideoCaptchaStatus)
		api.GET("/captcha/video/check/:session_id", handler.CheckVideoCaptchaValid)

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

			// ============ 租户管理路由 ============
			tenants := admin.Group("/tenants", middleware.AuthMiddleware())
			{
				tenants.GET("", handler.ListTenants)
				tenants.POST("", handler.CreateTenant)
				tenants.GET("/:id", handler.GetTenant)
				tenants.PUT("/:id", handler.UpdateTenant)
				tenants.DELETE("/:id", handler.DeleteTenant)

				// 租户用户管理
				tenants.GET("/:tenant_id/users", handler.ListTenantUsers)
				tenants.POST("/:tenant_id/users", handler.AddTenantUser)
				tenants.PUT("/:tenant_id/users/:user_id/role", handler.UpdateTenantUserRole)
				tenants.DELETE("/:tenant_id/users/:user_id", handler.RemoveTenantUser)

				// 租户邀请管理
				tenants.POST("/:tenant_id/invitations", handler.CreateInvitation)
				tenants.POST("/invitations/:token/accept", handler.AcceptInvitation)
				tenants.POST("/invitations/:id/revoke", handler.RevokeInvitation)

				// 租户配额管理
				tenants.GET("/:tenant_id/quotas", handler.ListTenantQuotas)
				tenants.GET("/:tenant_id/quotas/detail", handler.GetTenantQuota)
				tenants.PUT("/:tenant_id/quotas", handler.UpdateTenantQuota)

				// 租户账单管理
				tenants.GET("/:tenant_id/bills", handler.ListTenantBills)
				tenants.GET("/bills/:bill_id", handler.GetTenantBill)
				tenants.POST("/:tenant_id/bills/generate", handler.GenerateMonthlyBill)

				// 租户支付管理
				tenants.POST("/bills/:bill_id/payments", handler.CreatePayment)
				tenants.PUT("/payments/:payment_id/status", handler.UpdatePaymentStatus)
				tenants.GET("/:tenant_id/payments", handler.ListTenantPayments)
			}

			// ============ 账单管理路由（管理员） ============
			billing := admin.Group("/billing", middleware.AuthMiddleware())
			{
				billing.GET("/bills", handler.ListAllBills)
			}
		}
	}
}
