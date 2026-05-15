package main

import (
	"log"
	"net/http"
	"time"

	"github.com/hjtpx/hjtpx/internal/api/router"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/postgres"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

func main() {
	cfg := config.LoadConfig()

	if err := database.InitDB(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	if err := postgres.Connect(&cfg.Postgres); err != nil {
		log.Printf("Failed to connect to PostgreSQL: %v", err)
	} else {
		log.Println("PostgreSQL connected successfully")
	}

	if err := redis.ConnectRedis(&cfg.Redis); err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
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

	log.Printf("Server starting on port %s...", cfg.Server.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}