package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hjtpx/hjtpx/internal/api/router"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/postgres"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.LoadConfig()

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
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
