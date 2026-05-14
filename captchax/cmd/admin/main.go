package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"captchax/config"
	"captchax/internal/admin"
	"captchax/internal/log"
	"captchax/internal/repository"
	"captchax/pkg/cache"
	"captchax/pkg/database"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	log.Init(cfg.Log.Level, cfg.Log.Format, cfg.Log.Output)
	logger := log.Default()

	logger.Info("starting captchax admin", map[string]interface{}{
		"host": cfg.Server.Host,
		"port": cfg.Server.Port + 1,
	})

	redisClient, err := cache.NewRedis(&cfg.Redis)
	if err != nil {
		logger.Fatal("failed to connect to redis", map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer redisClient.Close()

	db, err := database.NewPostgres(&cfg.Database)
	if err != nil {
		logger.Fatal("failed to connect to postgres", map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer db.Close()

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	sqlDB, err := db.SQLDB()
	if err != nil {
		logger.Fatal("failed to get sql.DB", map[string]interface{}{
			"error": err.Error(),
		})
	}

	adminRepo := repository.NewAdminRepo(sqlDB)
	whitelistRepo := repository.NewWhitelistRepo(sqlDB)
	blacklistRepo := repository.NewBlacklistRepo(sqlDB)
	configRepo := repository.NewConfigRepo(sqlDB)
	captchaRepo := repository.NewCaptchaRepo(sqlDB)

	jwtSecret := cfg.Admin.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "captchax-admin-default-secret-change-in-production"
	}
	tokenTTL := cfg.Admin.TokenTTLSeconds
	if tokenTTL <= 0 {
		tokenTTL = 86400
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	adminRouter := admin.NewRouter(
		adminRepo,
		whitelistRepo,
		blacklistRepo,
		configRepo,
		captchaRepo,
		jwtSecret,
		tokenTTL,
	)
	adminRouter.RegisterRoutes(router)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port+1),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("admin server listening", map[string]interface{}{
			"address": cfg.Server.Addr(),
		})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("admin server failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down admin server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("admin server forced to shutdown", map[string]interface{}{
			"error": err.Error(),
		})
	}

	logger.Info("admin server exited")
}
