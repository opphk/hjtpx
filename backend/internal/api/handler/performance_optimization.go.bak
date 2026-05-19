package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type CacheMetricsHandler struct{}

type DatabaseMetricsHandler struct{}

func NewCacheMetricsHandler() *CacheMetricsHandler {
	return &CacheMetricsHandler{}
}

func NewDatabaseMetricsHandler() *DatabaseMetricsHandler {
	return &DatabaseMetricsHandler{}
}

func (h *CacheMetricsHandler) GetCacheHealth(c *gin.Context) {
	collector := redis.GetCacheMonitoringCollector()
	health := collector.GetHealthStatus()
	response.Success(c, health)
}

func (h *CacheMetricsHandler) GetCacheDetailedMetrics(c *gin.Context) {
	collector := redis.GetCacheMonitoringCollector()
	metrics := collector.GetDetailedMetrics()
	response.Success(c, metrics)
}

func (h *CacheMetricsHandler) GetCacheHotKeys(c *gin.Context) {
	limit := 10
	collector := redis.GetCacheMonitoringCollector()
	hotKeys := collector.GetHotKeys(limit)
	response.Success(c, hotKeys)
}

func (h *CacheMetricsHandler) GetCacheLatencyDistribution(c *gin.Context) {
	collector := redis.GetCacheMonitoringCollector()
	distribution := collector.GetLatencyDistribution()
	response.Success(c, distribution)
}

func (h *CacheMetricsHandler) GetCacheMemoryTrend(c *gin.Context) {
	collector := redis.GetCacheMonitoringCollector()
	trend := collector.GetMemoryTrend()
	response.Success(c, trend)
}

func (h *CacheMetricsHandler) GetCacheAlerts(c *gin.Context) {
	collector := redis.GetCacheMonitoringCollector()
	alerts := collector.GetAlerts()
	response.Success(c, alerts)
}

func (h *CacheMetricsHandler) AcknowledgeCacheAlert(c *gin.Context) {
	var req struct {
		AlertID int `json:"alert_id"`
	}
	if err := c.ShouldBindJSON(&req); err == nil {
		collector := redis.GetCacheMonitoringCollector()
		collector.AcknowledgeAlert(req.AlertID)
	}
	response.Success(c, nil)
}

func (h *CacheMetricsHandler) ClearCacheAlerts(c *gin.Context) {
	collector := redis.GetCacheMonitoringCollector()
	collector.ClearAlerts()
	response.Success(c, nil)
}

func (h *CacheMetricsHandler) ResetCacheMetrics(c *gin.Context) {
	collector := redis.GetCacheMonitoringCollector()
	collector.Reset()
	response.Success(c, nil)
}

func (h *CacheMetricsHandler) TriggerCacheWarmup(c *gin.Context) {
	warmer := redis.GetCacheWarmer()
	go warmer.WarmupAll()
	response.Success(c, map[string]string{"message": "Cache warmup triggered"})
}

func (h *CacheMetricsHandler) GetCacheWarmupStatus(c *gin.Context) {
	warmer := redis.GetCacheWarmer()
	status := warmer.GetStatus()
	response.Success(c, status)
}

func (h *CacheMetricsHandler) GetCacheConsistencyStatus(c *gin.Context) {
	dcc := redis.GetDistributedCacheConsistency()
	status := dcc.GetStatus()
	response.Success(c, status)
}

func (h *DatabaseMetricsHandler) GetDatabaseHealth(c *gin.Context) {
	monitor := database.GetPerformanceMonitor()
	stats := monitor.GetStats()

	health := map[string]interface{}{
		"status":    getDatabaseStatus(stats),
		"stats":     stats,
		"timestamp": monitor.GenerateReport().GeneratedAt.Unix(),
	}

	response.Success(c, health)
}

func getDatabaseStatus(stats *database.PerformanceStats) string {
	if stats.SlowQueryRatio > 20 {
		return "critical"
	} else if stats.SlowQueryRatio > 10 {
		return "warning"
	}
	return "healthy"
}

func (h *DatabaseMetricsHandler) GetSlowQueries(c *gin.Context) {
	limit := 20
	monitor := database.GetPerformanceMonitor()
	slowQueries := monitor.GetSlowQueries(limit)
	response.Success(c, slowQueries)
}

func (h *DatabaseMetricsHandler) GetTopQueries(c *gin.Context) {
	limit := 20
	monitor := database.GetPerformanceMonitor()
	queries := monitor.GetTopQueries(limit)
	response.Success(c, queries)
}

func (h *DatabaseMetricsHandler) GetQueryDistribution(c *gin.Context) {
	monitor := database.GetPerformanceMonitor()
	distribution := monitor.GetQueryDistribution()
	response.Success(c, distribution)
}

func (h *DatabaseMetricsHandler) GeneratePerformanceReport(c *gin.Context) {
	monitor := database.GetPerformanceMonitor()
	report := monitor.GenerateReport()
	response.Success(c, report)
}

func (h *DatabaseMetricsHandler) GetOptimizationSuggestions(c *gin.Context) {
	monitor := database.GetPerformanceMonitor()
	report := monitor.GenerateReport()

	suggestions := []map[string]interface{}{}

	if report.DatabaseStats.SlowQueryRatio > 10 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":     "slow_queries",
			"priority": "high",
			"message":  "慢查询比例较高，建议添加索引或优化查询",
		})
	}

	if report.DatabaseStats.AvgDuration.Milliseconds() > 100 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":     "avg_duration",
			"priority": "high",
			"message":  "平均查询时间过长，建议优化查询或增加缓存",
		})
	}

	if report.ConnectionStats != nil && report.ConnectionStats.ReuseRate < 70 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":     "connection_pool",
			"priority": "medium",
			"message":  "连接复用率低，建议优化连接池配置",
		})
	}

	for _, rec := range report.Recommendations {
		suggestions = append(suggestions, map[string]interface{}{
			"type":     "general",
			"priority": "medium",
			"message":  rec,
		})
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":     "good",
			"priority": "low",
			"message":  "数据库性能良好，无明显优化建议",
		})
	}

	response.Success(c, suggestions)
}

func (h *DatabaseMetricsHandler) ClearPerformanceMetrics(c *gin.Context) {
	monitor := database.GetPerformanceMonitor()
	monitor.Clear()
	response.Success(c, nil)
}