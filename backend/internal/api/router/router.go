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

	// 健康检查
	r.GET("/health", handler.HealthCheck)

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

	// API路由组
	api := r.Group("/api/v1")
	{
		// 验证码相关路由
		captcha := api.Group("/captcha")
		{
			captcha.GET("/slider", handler.GetSliderCaptcha)
			captcha.GET("/click", handler.GetClickCaptcha)
			captcha.POST("/verify", handler.VerifyCaptcha)
		}

		// 无感验证路由
		verify := api.Group("/verify")
		{
			verify.POST("/silent", handler.SilentVerify)
			verify.GET("/silent/status", handler.GetSilentVerifyStatus)
		}

		// 认证路由（供前端调用）
		auth := api.Group("/auth")
		{
			auth.POST("/login", handler.UserLogin)
			auth.POST("/logout", handler.Logout)
			auth.POST("/register", handler.Register)
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
				}

				// 统计数据
				stats := adminAuth.Group("/stats")
				{
					stats.GET("/verification", handler.GetVerificationStats)
					stats.GET("/chart", handler.GetChartData)
				}

				// 应用管理
				applications := adminAuth.Group("/applications")
				{
					applications.GET("", handler.ListApplications)
					applications.POST("", handler.CreateApplication)
					applications.PUT("/:id", handler.UpdateApplication)
					applications.DELETE("/:id", handler.DeleteApplication)
				}

				// 验证日志查询
				logs := adminAuth.Group("/logs")
				{
					logs.GET("", handler.GetVerificationLogs)
					logs.GET("/statistics", handler.GetLogStatistics)
					logs.GET("/:id", handler.GetLogDetail)
				}

				// CSS来源切换
				adminAuth.GET("/css-source", handler.GetCSSSource)
				adminAuth.POST("/css-source", handler.SetCSSSource)

				// 无感验证管理
				silent := adminAuth.Group("/silent")
				{
					silent.GET("/config", handler.GetSilentConfig)
					silent.POST("/config", handler.UpdateSilentConfig)
					silent.PUT("/rules/:id", handler.UpdateStrategyRule)
					silent.GET("/stats", handler.GetSilentStats)
					silent.POST("/rate-limit/reset", handler.ResetSilentRateLimit)
				}
			}
		}

		// 示例路由
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})

		// 设备指纹路由
		handler.RegisterFingerprintRoutes(r)
	}

	return r
}