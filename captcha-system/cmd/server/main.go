package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opphk/captcha-system/internal/config"
	"github.com/opphk/captcha-system/internal/handler"
	"github.com/opphk/captcha-system/internal/middleware"
	"github.com/opphk/captcha-system/internal/repository"
	"github.com/opphk/captcha-system/pkg/captcha"
)

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := repository.NewDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	redisClient, err := config.NewRedisClient(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
	defer redisClient.Close()

	cacheService := captcha.NewCacheService(redisClient)

	challengeRepo := repository.NewChallengeRepository(db)
	attemptRepo := repository.NewAttemptRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	statsRepo := repository.NewStatsRepository(db)
	adminRepo := repository.NewAdminRepository(db)

	captchaService := captcha.NewCaptchaService(challengeRepo, attemptRepo, sessionRepo, cacheService)
	adminService := captcha.NewAdminService(adminRepo, statsRepo)
	jwtService := captcha.NewJWTService(cfg.JWT.Secret, cfg.JWT.ExpiresHours)

	challengeHandler := handler.NewChallengeHandler(captchaService)
	adminHandler := handler.NewAdminHandler(adminService, jwtService)
	healthHandler := handler.NewHealthHandler(db, redisClient)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(middleware.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.CORS())

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthHandler.Health)
		v1.GET("/health/detailed", healthHandler.DetailedHealth)

		captchaGroup := v1.Group("/captcha")
		{
			captchaGroup.POST("/slider/create", challengeHandler.CreateSliderCaptcha)
			captchaGroup.POST("/slider/verify", challengeHandler.VerifySliderCaptcha)
			captchaGroup.POST("/click/create", challengeHandler.CreateClickCaptcha)
			captchaGroup.POST("/click/verify", challengeHandler.VerifyClickCaptcha)
			captchaGroup.POST("/rotate/create", challengeHandler.CreateRotateCaptcha)
			captchaGroup.POST("/rotate/verify", challengeHandler.VerifyRotateCaptcha)
			captchaGroup.POST("/behavior/analyze", challengeHandler.AnalyzeBehavior)
			captchaGroup.POST("/session/create", challengeHandler.CreateSession)
			captchaGroup.POST("/session/validate", challengeHandler.ValidateSession)
			captchaGroup.GET("/config", challengeHandler.GetConfig)
		}

		adminGroup := v1.Group("/admin")
		{
			adminGroup.POST("/login", adminHandler.Login)
			adminGroup.POST("/refresh", adminHandler.RefreshToken)

			protected := adminGroup.Group("")
			protected.Use(handler.JWTAuth(&handler.JWTAuthConfig{
				Secret:       cfg.JWT.Secret,
				ExpiresHours: cfg.JWT.ExpiresHours,
			}))
			{
				protected.POST("/logout", adminHandler.Logout)
				protected.GET("/me", adminHandler.GetCurrentUser)
				protected.GET("/stats", adminHandler.GetStats)
				protected.GET("/challenges", adminHandler.GetChallenges)
				protected.GET("/attempts", adminHandler.GetAttempts)
				protected.PUT("/config", adminHandler.UpdateConfig)
				protected.GET("/logs", adminHandler.GetLogs)
			}
		}
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
