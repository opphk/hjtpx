package router

import (
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

		// 认证路由（供前端调用）
		auth := api.Group("/auth")
		{
			auth.POST("/login", handler.Login)
			auth.POST("/logout", handler.Logout)
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
			}
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
