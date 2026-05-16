package database

import (
	"log"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

type QueryMetric struct {
	Query      string
	Duration   time.Duration
	Timestamp  time.Time
	IsSlow     bool
	Error      error
}

type PerformanceMonitor struct {
	mu              sync.RWMutex
	queryMetrics    []QueryMetric
	slowQueries     []QueryMetric
	maxMetricsLen   int
	maxSlowQueryLen int
	enabled         bool
	slowThreshold   time.Duration
}

type PerformanceStats struct {
	TotalQueries    int64
	SlowQueries     int64
	FailedQueries   int64
	AvgDuration     time.Duration
	MaxDuration     time.Duration
	MinDuration     time.Duration
	TotalDuration   time.Duration
}

var perfMonitor *PerformanceMonitor

func InitPerformanceMonitor(cfg *config.Config) {
	perfMonitor = &PerformanceMonitor{
		queryMetrics:    make([]QueryMetric, 0),
		slowQueries:     make([]QueryMetric, 0),
		maxMetricsLen:   10000,
		maxSlowQueryLen: 1000,
		enabled:         cfg.Database.Monitoring.EnableQueryMetrics,
		slowThreshold:   time.Duration(cfg.Database.SlowQueryThresholdMs) * time.Millisecond,
	}

	if perfMonitor.enabled {
		log.Println("Performance monitor initialized")
	}
}

func GetPerformanceMonitor() *PerformanceMonitor {
	return perfMonitor
}

func (pm *PerformanceMonitor) RecordQuery(query string, duration time.Duration, err error) {
	if !pm.enabled {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	isSlow := duration > pm.slowThreshold

	metric := QueryMetric{
		Query:     query,
		Duration:  duration,
		Timestamp: time.Now(),
		IsSlow:    isSlow,
		Error:     err,
	}

	pm.queryMetrics = append(pm.queryMetrics, metric)

	if len(pm.queryMetrics) > pm.maxMetricsLen {
		pm.queryMetrics = pm.queryMetrics[1:]
	}

	if isSlow {
		log.Printf("SLOW QUERY: %s took %v", query, duration)
		pm.slowQueries = append(pm.slowQueries, metric)

		if len(pm.slowQueries) > pm.maxSlowQueryLen {
			pm.slowQueries = pm.slowQueries[1:]
		}
	}
}

func (pm *PerformanceMonitor) GetStats() *PerformanceStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := &PerformanceStats{}
	if len(pm.queryMetrics) == 0 {
		return stats
	}

	var totalDuration time.Duration
	var minDuration time.Duration = pm.queryMetrics[0].Duration
	var maxDuration time.Duration

	for _, m := range pm.queryMetrics {
		stats.TotalQueries++
		totalDuration += m.Duration

		if m.IsSlow {
			stats.SlowQueries++
		}
		if m.Error != nil {
			stats.FailedQueries++
		}
		if m.Duration > maxDuration {
			maxDuration = m.Duration
		}
		if m.Duration < minDuration {
			minDuration = m.Duration
		}
	}

	stats.TotalDuration = totalDuration
	stats.AvgDuration = totalDuration / time.Duration(stats.TotalQueries)
	stats.MaxDuration = maxDuration
	stats.MinDuration = minDuration

	return stats
}

func (pm *PerformanceMonitor) GetSlowQueries(limit int) []QueryMetric {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if limit <= 0 || limit > len(pm.slowQueries) {
		limit = len(pm.slowQueries)
	}

	result := make([]QueryMetric, limit)
	copy(result, pm.slowQueries[len(pm.slowQueries)-limit:])
	return result
}

func (pm *PerformanceMonitor) Clear() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.queryMetrics = make([]QueryMetric, 0)
	pm.slowQueries = make([]QueryMetric, 0)
}

func GormQueryCallback(db *gorm.DB) {
	startTime := time.Now()

	db.InstanceSet("query_start_time", startTime)

	db.Callback().Query().After("gorm:query").Register("performance_monitor", func(d *gorm.DB) {
		if perfMonitor == nil {
			return
		}

		var duration time.Duration
		if startTime, ok := d.InstanceGet("query_start_time"); ok {
			duration = time.Since(startTime.(time.Time))
		}

		sql := d.Dialector.Explain(d.Statement.SQL.String(), d.Statement.Vars...)
		perfMonitor.RecordQuery(sql, duration, d.Error)
	})

	db.Callback().Create().After("gorm:create").Register("performance_monitor_write", func(d *gorm.DB) {
		perfMonitor.recordWriteOperation(d, "CREATE")
	})

	db.Callback().Update().After("gorm:update").Register("performance_monitor_write", func(d *gorm.DB) {
		perfMonitor.recordWriteOperation(d, "UPDATE")
	})

	db.Callback().Delete().After("gorm:delete").Register("performance_monitor_write", func(d *gorm.DB) {
		perfMonitor.recordWriteOperation(d, "DELETE")
	})
}

func (pm *PerformanceMonitor) recordWriteOperation(db *gorm.DB, opType string) {
	if !pm.enabled {
		return
	}

	var duration time.Duration
	if startTime, ok := db.InstanceGet("query_start_time"); ok {
		duration = time.Since(startTime.(time.Time))
	}

	sql := db.Dialector.Explain(db.Statement.SQL.String(), db.Statement.Vars...)
	pm.RecordQuery(opType+": "+sql, duration, db.Error)
}

func ExplainQuery(query string, args ...interface{}) (string, error) {
	var result string
	explainSQL := "EXPLAIN ANALYZE " + query
	err := DB.Raw(explainSQL, args...).Scan(&result).Error
	return result, err
}

func AnalyzeTable(tableName string) error {
	return DB.Exec("ANALYZE " + tableName).Error
}
