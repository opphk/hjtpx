package api

import (
	"net/http"

	"hjtpx/internal/api/handler"
	"hjtpx/internal/api/middleware"
	"hjtpx/internal/config"
	"hjtpx/internal/database"
	"hjtpx/internal/repository"
	"hjtpx/internal/services"
	"hjtpx/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

var jwtManager *utils.JWTManager
var rateLimiter *middleware.RateLimiter
var signatureVerifier *middleware.SignatureVerifier

func SetupRouter(cfg *config.Config, db *gorm.DB, redisClient *redis.Client) *gin.Engine {
	router := gin.Default()

	jwtManager = middleware.InitAuthMiddleware(cfg.JWT)

	redisCache := database.NewRedisCache(redisClient)
	rateLimiter = middleware.NewRateLimiter(redisClient, cfg.RateLimit)
	signatureVerifier = middleware.InitSignatureMiddleware(cfg.Signature, redisClient)

	captchaRepo := repository.NewCaptchaRepository(db, redisCache)
	userRepo := repository.NewUserRepository(db, redisCache)
	appRepo := repository.NewAppRepository(db, redisCache)
	verificationLogRepo := repository.NewVerificationLogRepository(db, redisCache)

	adminService := services.NewAdminService(userRepo, appRepo, captchaRepo, verificationLogRepo)

	captchaHandler := handler.NewCaptchaHandler(captchaRepo)
	userHandler := handler.NewUserHandler(userRepo, jwtManager)
	adminHandler := handler.NewAdminHandlerWithService(userRepo, appRepo, captchaRepo, verificationLogRepo, adminService, jwtManager)

	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	router.Static("/static", "./static")
	router.LoadHTMLGlob("templates/*.html")
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	router.GET("/demo", func(c *gin.Context) {
		c.HTML(http.StatusOK, "demo.html", nil)
	})
	router.GET("/admin", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin.html", nil)
	})

	router.GET("/health", healthCheck)

	v1 := router.Group("/api/v1")
	{
		captcha := v1.Group("/captcha")
		captcha.Use(middleware.SignatureVerification(signatureVerifier))
		captcha.Use(middleware.RateLimitWithRedis(rateLimiter))
		{
			captcha.POST("/create", captchaHandler.Create)
			captcha.POST("/verify", captchaHandler.Verify)
			captcha.GET("/status/:token", captchaHandler.GetStatus)
		}

		user := v1.Group("/user")
		user.Use(middleware.Auth(jwtManager))
		{
			user.POST("/register", userHandler.Register)
			user.POST("/login", userHandler.Login)
			user.POST("/logout", userHandler.Logout)
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
		}

		admin := v1.Group("/admin")
		admin.Use(middleware.AdminAuth(jwtManager))
		admin.Use(middleware.RateLimitWithRedis(rateLimiter))
		{
			admin.GET("/dashboard", adminHandler.GetDashboard)
			admin.GET("/users", adminHandler.ListUsers)
			admin.GET("/users/:id", adminHandler.GetUser)
			admin.POST("/users", adminHandler.CreateUser)
			admin.PUT("/users/:id", adminHandler.UpdateUser)
			admin.DELETE("/users/:id", adminHandler.DeleteUser)
			admin.GET("/apps", adminHandler.ListApps)
			admin.GET("/apps/:id", adminHandler.GetApp)
			admin.POST("/apps", adminHandler.CreateApp)
			admin.PUT("/apps/:id", adminHandler.UpdateApp)
			admin.DELETE("/apps/:id", adminHandler.DeleteApp)
			admin.GET("/captchas", adminHandler.ListCaptchas)
			admin.GET("/captchas/stats", adminHandler.GetCaptchaStats)
		}

		admin.POST("/admin/create", adminHandler.CreateAdmin)
		admin.POST("/admin/login", adminHandler.AdminLogin)
	}

	return router
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func GetJWTManager() *utils.JWTManager {
	return jwtManager
}

func GetSignatureVerifier() *middleware.SignatureVerifier {
	return signatureVerifier
}
