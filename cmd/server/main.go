package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hjtpx/internal/api"
	"hjtpx/internal/config"
	"hjtpx/internal/database"
	"hjtpx/internal/utils"

	"github.com/spf13/viper"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = ".env"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := utils.InitLogger(cfg.Log.Level, cfg.Log.OutputPath); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	utils.Info("Starting application...")

	postgresDB, err := database.InitPostgres(cfg.Database, database.PostgresConfig{
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
	})
	if err != nil {
		utils.Error("Failed to connect to PostgreSQL: %v", err)
		os.Exit(1)
	}

	if err := database.AutoMigrate(postgresDB); err != nil {
		utils.Error("Failed to migrate database: %v", err)
		os.Exit(1)
	}

	redisClient, err := database.InitRedis(cfg.Redis)
	if err != nil {
		utils.Error("Failed to connect to Redis: %v", err)
		os.Exit(1)
	}

	router := api.SetupRouter(cfg, postgresDB, redisClient)

	srv := &http.Server{
		Addr:    cfg.App.Addr(),
		Handler: router,
	}

	go func() {
		utils.Info("Server starting on %s", cfg.App.Addr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.Error("Server failed: %v", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	utils.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		utils.Error("Server forced to shutdown: %v", err)
	}

	if redisClient != nil {
		if err := database.CloseRedis(); err != nil {
			utils.Error("Redis connection close error: %v", err)
		}
	}

	if postgresDB != nil {
		if err := database.ClosePostgres(); err != nil {
			utils.Error("PostgreSQL connection close error: %v", err)
		}
	}

	utils.Info("Server exited")
}

func init() {
	viper.AutomaticEnv()
}
