package handler

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/metrics"
	"github.com/hjtpx/hjtpx/pkg/postgres"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

var (
	startTime = time.Now()
)

type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Uptime    string                 `json:"uptime"`
	Services  map[string]interface{} `json:"services"`
	Metrics   map[string]interface{} `json:"metrics"`
	System    map[string]interface{} `json:"system"`
}

func HealthCheck(c *gin.Context) {
	status := HealthStatus{
		Timestamp: time.Now().Format(time.RFC3339),
		Uptime:    time.Since(startTime).String(),
		Services:  make(map[string]interface{}),
		Metrics: map[string]interface{}{
			"total_requests": metrics.GetRequestCount(),
			"success_count":  metrics.GetSuccessCount(),
			"failure_count":  metrics.GetFailureCount(),
			"success_rate":   metrics.GetSuccessRate(),
		},
		System: getSystemMetrics(),
	}

	overallStatus := "healthy"

	if postgres.DB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := postgres.DB.PingContext(ctx); err != nil {
			status.Services["postgres"] = map[string]interface{}{
				"status":  "unhealthy",
				"message": err.Error(),
			}
			overallStatus = "degraded"
		} else {
			status.Services["postgres"] = map[string]interface{}{
				"status": "healthy",
			}
		}
	} else {
		status.Services["postgres"] = map[string]interface{}{
			"status":  "disconnected",
			"message": "database not initialized",
		}
		overallStatus = "degraded"
	}

	if redis.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := redis.Client.Ping(ctx).Err(); err != nil {
			status.Services["redis"] = map[string]interface{}{
				"status":  "unhealthy",
				"message": err.Error(),
			}
			overallStatus = "degraded"
		} else {
			status.Services["redis"] = map[string]interface{}{
				"status": "healthy",
			}
		}
	} else {
		status.Services["redis"] = map[string]interface{}{
			"status":  "disconnected",
			"message": "redis not initialized",
		}
		overallStatus = "degraded"
	}

	status.Status = overallStatus

	statusCode := http.StatusOK
	if overallStatus != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, status)
}

func getSystemMetrics() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"go_version":   runtime.Version(),
		"go_routines":  runtime.NumGoroutine(),
		"memory_alloc": m.Alloc,
		"memory_total": m.TotalAlloc,
		"memory_sys":   m.Sys,
		"gc_runs":      m.NumGC,
		"cpu_num":      runtime.NumCPU(),
	}
}

type ReadinessCheck struct{}

func NewReadinessCheck() *ReadinessCheck {
	return &ReadinessCheck{}
}

func (r *ReadinessCheck) IsReady() bool {
	if postgres.DB == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := postgres.DB.PingContext(ctx); err != nil {
		return false
	}
	return true
}

func Readiness(c *gin.Context) {
	if !NewReadinessCheck().IsReady() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "not_ready",
			"message": "service dependencies not available",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

type LivenessCheck struct{}

func NewLivenessCheck() *LivenessCheck {
	return &LivenessCheck{}
}

func (l *LivenessCheck) IsAlive() bool {
	return true
}

func Liveness(c *gin.Context) {
	if !NewLivenessCheck().IsAlive() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "dead",
			"message": "service is dead",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "alive",
		"uptime":  time.Since(startTime).String(),
		"version": "1.0.0",
	})
}
