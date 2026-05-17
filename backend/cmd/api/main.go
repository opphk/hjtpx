package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hjtpx/hjtpx/internal/api/handler"
	"github.com/hjtpx/hjtpx/internal/api/router"
	"github.com/hjtpx/hjtpx/internal/repository"
	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/i18n"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/postgres"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"golang.org/x/crypto/bcrypt"
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

	seedAdmin()

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

	if database.DB != nil {
		handler.InitAlertService(database.DB)
		log.Println("Alert service initialized successfully")
	}

	var sessionCache *cache.SessionCache
	if redis.Client != nil {
		sessionCache = cache.NewSessionCache()
		log.Println("Session cache initialized successfully")
	}

	captchaRepo := db.NewCaptchaRepository()
	generatorService := captcha.NewGeneratorService(sessionCache, captchaRepo)
	verifierService := captcha.NewVerifierService(sessionCache, captchaRepo)
	handler.InitSliderCaptchaHandler(generatorService, verifierService)
	log.Println("Slider captcha service initialized successfully")

	if database.DB != nil {
		configRepo := repository.NewConfigRepo(database.DB)
		configCache := service.NewConfigCache(redis.Client)
		configService := service.NewConfigService(configRepo, configCache)
		if err := configService.InitializeDefaults(); err != nil {
			log.Printf("Warning: Failed to initialize default configs: %v", err)
		} else {
			log.Println("Config service initialized successfully")
		}
		handler.InitConfigService(configService)
	}

	ctx := context.Background()
	if cacheOptimizer := service.GetCacheOptimizer(); cacheOptimizer != nil {
		if err := cacheOptimizer.Initialize(ctx); err != nil {
			log.Printf("Warning: Cache initialization failed: %v", err)
		} else {
			log.Println("Cache optimizer initialized successfully")
		}
	}

	r := router.SetupRouter()

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

	if cacheOptimizer := service.GetCacheOptimizer(); cacheOptimizer != nil {
		cacheOptimizer.Shutdown(ctxShutdown)
	}

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

func seedAdmin() {
	if database.DB == nil {
		log.Println("Database not available, skipping admin user creation")
		return
	}

	var count int64
	database.DB.Model(&models.Admin{}).Count(&count)
	if count > 0 {
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash admin password: %v", err)
		return
	}

	admin := models.Admin{
		Username:     "admin",
		PasswordHash: string(hash),
		IsSuperAdmin: true,
	}

	if err := database.DB.Create(&admin).Error; err != nil {
		log.Printf("Failed to seed admin user: %v", err)
		return
	}

	log.Println("Default admin user created (username: admin, password: admin123)")
}
