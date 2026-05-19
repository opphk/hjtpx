package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/middleware"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/i18n"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/hjtpx/hjtpx/pkg/postgres"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

func main() {
	cfg := config.LoadConfig()

	i18nConfig := i18n.LocaleConfig{
		DefaultLang:     cfg.I18n.DefaultLang,
		TranslationsDir: cfg.I18n.TranslationsDir,
		SupportedLangs:  cfg.I18n.SupportedLangs,
	}
	if err := i18n.Init(i18nConfig); err != nil {
		log.Printf("Warning: Failed to initialize i18n: %v", err)
		log.Println("Continuing startup without i18n...")
	} else {
		log.Println("i18n initialized successfully")
	}

	if err := i18n.SetDefaultTimezone(cfg.I18n.DefaultTimezone); err != nil {
		log.Printf("Warning: Failed to set default timezone: %v", err)
	} else {
		log.Printf("Default timezone set to: %s", cfg.I18n.DefaultTimezone)
	}

	if err := database.InitDB(cfg); err != nil {
		log.Printf("Warning: Failed to initialize database: %v", err)
		log.Println("Continuing startup without database...")
	}

	jwt.InitJWT(cfg.JWT.Secret)
	jwt.InitUserJWT(cfg.JWT.Secret)

	if err := postgres.Connect(&cfg.Postgres); err != nil {
		log.Printf("Warning: Failed to connect to PostgreSQL: %v", err)
	} else {
		log.Println("PostgreSQL connected successfully")
	}

	if err := redis.ConnectRedis(&cfg.Redis); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
	} else {
		log.Println("Redis connected successfully")
		redis.InitEnhancedCache(nil)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":    "HJTPX Captcha API",
			"version": "16.0",
			"status":  "running",
		})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s...", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	if postgres.DB != nil {
		postgres.DB.Close()
	}
	if redis.Client != nil {
		redis.Client.Close()
	}

	log.Println("Server exited")
}
