package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service_performance"
)

func AdvancedPerformanceMonitoring() gin.HandlerFunc {
	collector := performance.NewMetricsCollector()

	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		method := c.Request.Method

		collector.IncActiveRequests()
		defer collector.DecActiveRequests()

		c.Next()

		duration := time.Since(start)
		ms := float64(duration.Milliseconds())
		status := strconv.Itoa(c.Writer.Status())

		collector.RecordRequest(method, path, status, ms)

		c.Header("X-Response-Time", strconv.FormatInt(duration.Milliseconds(), 10)+"ms")
		c.Header("X-Request-ID", c.GetHeader("X-Request-ID"))
	}
}

func PerformanceMonitoring() gin.HandlerFunc {
	return AdvancedPerformanceMonitoring()
}

func GetPerformanceStatsHandler() gin.HandlerFunc {
	collector := performance.NewMetricsCollector()

	return func(c *gin.Context) {
		stats := collector.GetStats()
		c.JSON(200, gin.H{
			"success": true,
			"data":    stats,
		})
	}
}

func GetComplianceStatusHandler() gin.HandlerFunc {
	optimizer := performance.NewPerformanceOptimizer()

	return func(c *gin.Context) {
		status := optimizer.GetComplianceStatus()
		c.JSON(200, gin.H{
			"success": true,
			"data":    status,
		})
	}
}
