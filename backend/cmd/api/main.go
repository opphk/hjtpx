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
	"github.com/hjtpx/hjtpx/internal/api/router"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/i18n"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/hjtpx/hjtpx/pkg/postgres"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/performance"
)

func main() {
	cfg := config.LoadConfig()
	
	// Initialize Performance Engine (High Priority)
	perfEngine := performance.NewPerformanceEngine()
	ctx := context.Background()
	if err := perfEngine.Start(ctx); err != nil {
		log.Printf("Warning: Failed to start performance engine: %v", err)
	} else {
		log.Println("Performance engine started successfully")
	}
	defer perfEngine.Stop()
	
	// Initialize Database Optimizer
	dbOptimizer := performance.NewDatabaseOptimizer()
	if err := dbOptimizer.Start(ctx); err != nil {
		log.Printf("Warning: Failed to start database optimizer: %v", err)
	} else {
		log.Println("Database optimizer started successfully")
	}
	defer dbOptimizer.Stop()
	
	// Initialize Cache Optimizer
	cacheOptimizer := performance.NewCacheOptimizer()
	if err := cacheOptimizer.Start(ctx); err != nil {
		log.Printf("Warning: Failed to start cache optimizer: %v", err)
	} else {
		log.Println("Cache optimizer started successfully")
	}
	defer cacheOptimizer.Stop()
	
	// Initialize Concurrency Manager
	concurrencyMgr := performance.NewConcurrencyManager()
	if err := concurrencyMgr.Start(ctx); err != nil {
		log.Printf("Warning: Failed to start concurrency manager: %v", err)
	} else {
		log.Println("Concurrency manager started successfully")
	}
	defer concurrencyMgr.Stop()
	
	// Initialize Resource Manager
	resourceMgr := performance.NewResourceManager()
	if err := resourceMgr.Start(ctx); err != nil {
		log.Printf("Warning: Failed to start resource manager: %v", err)
	} else {
		log.Println("Resource manager started successfully")
	}
	defer resourceMgr.Stop()
	
	// Initialize Edge Compute
	edgeCompute := performance.NewEdgeCompute()
	if err := edgeCompute.Start(ctx); err != nil {
		log.Printf("Warning: Failed to start edge compute: %v", err)
	} else {
		log.Println("Edge compute started successfully")
	}
	defer edgeCompute.Stop()
	
	// Initialize WASM Engine
	wasmEngine := performance.NewWASMEngine()
	if err := wasmEngine.Start(ctx); err != nil {
		log.Printf("Warning: Failed to start WASM engine: %v", err)
	} else {
		log.Println("WASM engine started successfully")
	}
	defer wasmEngine.Stop()

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

	router.SetupRoutes(r)

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
