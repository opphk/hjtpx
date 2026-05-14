package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"captchax/config"
	"captchax/internal/log"
	"captchax/pkg/cache"
	"captchax/pkg/database"

	"github.com/gin-gonic/gin"
)

type metrics struct {
	requestsTotal    atomic.Int64
	requestsSuccess atomic.Int64
	requestsFailed  atomic.Int64
	activeRequests  atomic.Int64
	requestDuration atomic.Int64
	imageGenerated  atomic.Int64
	imageCacheHits  atomic.Int64
	imageCacheMisses atomic.Int64
}

var appMetrics = &metrics{}

func init() {
	go recordMetrics()
}

func recordMetrics() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	var lastReqTotal, lastReqDuration int64

	for range ticker.C {
		currentReqTotal := appMetrics.requestsTotal.Load()
		currentDuration := appMetrics.requestDuration.Load()

		reqDelta := currentReqTotal - lastReqTotal
		durationDelta := currentDuration - lastReqDuration

		if reqDelta > 0 {
			avgDuration := durationDelta / reqDelta
			lastReqTotal = currentReqTotal
			lastReqDuration = currentDuration

			logger := log.Default()
			logger.Info("metrics_sample", map[string]interface{}{
				"requests_delta":   reqDelta,
				"avg_duration_ms": avgDuration / 1e6,
			})
		}
	}
}

func recordRequest(success bool, duration time.Duration) {
	appMetrics.requestsTotal.Add(1)
	if success {
		appMetrics.requestsSuccess.Add(1)
	} else {
		appMetrics.requestsFailed.Add(1)
	}
	appMetrics.requestDuration.Add(duration.Nanoseconds())
	appMetrics.activeRequests.Add(-1)
}

func setupPPROF(router *gin.Engine) {
	router.Any("/debug/pprof/*action", pprofHandler)
}

func pprofHandler(c *gin.Context) {
	switch c.Param("action") {
	case "/profile":
		pprof.Profile(c.Writer, c.Request)
	case "/trace":
		pprof.Trace(c.Writer, c.Request)
	default:
		pprof.Index(c.Writer, c.Request)
	}
}

func setupMetrics(router *gin.Engine) {
	router.GET("/metrics", metricsHandler)

	router.GET("/metrics/counters", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"requests_total":   appMetrics.requestsTotal.Load(),
			"requests_success": appMetrics.requestsSuccess.Load(),
			"requests_failed": appMetrics.requestsFailed.Load(),
			"active_requests":  appMetrics.activeRequests.Load(),
			"images_generated": appMetrics.imageGenerated.Load(),
			"cache_hits":      appMetrics.imageCacheHits.Load(),
			"cache_misses":    appMetrics.imageCacheMisses.Load(),
		})
	})

	router.GET("/metrics/runtime", func(c *gin.Context) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		c.JSON(http.StatusOK, gin.H{
			"go_version":        runtime.Version(),
			"go_routines":       runtime.NumGoroutine(),
			"mem_alloc":         m.Alloc,
			"mem_total_alloc":   m.TotalAlloc,
			"mem_sys":           m.Sys,
			"mem_lookups":       m.Lookups,
			"mem_mallocs":       m.Mallocs,
			"mem_frees":         m.Frees,
			"mem_heap_alloc":    m.HeapAlloc,
			"mem_heap_sys":      m.HeapSys,
			"mem_heap_idle":     m.HeapIdle,
			"mem_heap_inuse":    m.HeapInuse,
			"mem_heap_released": m.HeapReleased,
			"mem_heap_objects":  m.HeapObjects,
			"mem_stack_inuse":   m.StackInuse,
			"mem_stack_sys":     m.StackSys,
			"mem_mspan_inuse":   m.MSpanInuse,
			"mem_mspan_sys":     m.MSpanSys,
			"mem_mcache_inuse":  m.MCacheInuse,
			"mem_mcache_sys":    m.MCacheSys,
			"gc_count":          m.NumGC,
			"gc_pause_total":    m.PauseTotalNs,
		})
	})
}

func metricsHandler(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	output := fmt.Sprintf(`# HELP captchax_requests_total Total number of requests
# TYPE captchax_requests_total counter
captchax_requests_total %d

# HELP captchax_requests_success Number of successful requests
# TYPE captchax_requests_success counter
captchax_requests_success %d

# HELP captchax_requests_failed Number of failed requests
# TYPE captchax_requests_failed counter
captchax_requests_failed %d

# HELP captchax_active_requests Number of active requests
# TYPE captchax_active_requests gauge
captchax_active_requests %d

# HELP captchax_images_generated Number of images generated
# TYPE captchax_images_generated counter
captchax_images_generated %d

# HELP captchax_image_cache_hits Number of image cache hits
# TYPE captchax_image_cache_hits counter
captchax_image_cache_hits %d

# HELP captchax_image_cache_misses Number of image cache misses
# TYPE captchax_image_cache_misses counter
captchax_image_cache_misses %d

# HELP captchax_go_routines Number of goroutines
# TYPE captchax_go_routines gauge
captchax_go_routines %d

# HELP captchax_memory_alloc Bytes allocated
# TYPE captchax_memory_alloc gauge
captchax_memory_alloc %d

# HELP captchax_memory_sys System bytes
# TYPE captchax_memory_sys gauge
captchax_memory_sys %d

# HELP captchax_gc_runs Number of garbage collections
# TYPE captchax_gc_runs gauge
captchax_gc_runs %d
`,
		appMetrics.requestsTotal.Load(),
		appMetrics.requestsSuccess.Load(),
		appMetrics.requestsFailed.Load(),
		appMetrics.activeRequests.Load(),
		appMetrics.imageGenerated.Load(),
		appMetrics.imageCacheHits.Load(),
		appMetrics.imageCacheMisses.Load(),
		runtime.NumGoroutine(),
		m.Alloc,
		m.Sys,
		m.NumGC,
	)

	c.Header("Content-Type", "text/plain; version=0.0.4")
	c.String(http.StatusOK, output)
}

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		appMetrics.activeRequests.Add(1)

		c.Next()

		duration := time.Since(start)
		success := c.Writer.Status() < 400
		recordRequest(success, duration)
	}
}

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	log.Init(cfg.Log.Level, cfg.Log.Format, cfg.Log.Output)
	logger := log.Default()

	logger.Info("starting captchax server", map[string]interface{}{
		"host":    cfg.Server.Host,
		"port":    cfg.Server.Port,
		"pprof":   "enabled at /debug/pprof/*",
		"metrics": "enabled at /metrics",
	})

	redisClient, err := cache.NewRedis(&cfg.Redis)
	if err != nil {
		logger.Fatal("failed to connect to redis", map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer redisClient.Close()
	logger.Info("connected to redis", map[string]interface{}{
		"addr": cfg.Redis.Addr(),
	})

	db, err := database.NewPostgres(&cfg.Database)
	if err != nil {
		logger.Fatal("failed to connect to postgres", map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer db.Close()
	logger.Info("connected to postgres", map[string]interface{}{
		"host": cfg.Database.Host,
		"db":   cfg.Database.DBName,
	})

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	router.Use(MetricsMiddleware())

	setupPPROF(router)
	setupMetrics(router)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "captchax",
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		if err := redisClient.Ping(ctx).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"redis":  "disconnected",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"redis":  "connected",
		})
	})

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "CaptchaX",
			"version": "1.0.0",
		})
	})

	server := &http.Server{
		Addr:           cfg.Server.Addr(),
		Handler:        router,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		logger.Info("server listening", map[string]interface{}{
			"address": cfg.Server.Addr(),
		})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", map[string]interface{}{
			"error": err.Error(),
		})
	}

	logger.Info("server exited")
}
