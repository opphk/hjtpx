package database

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

type QueryOptimizer struct {
	db                *gorm.DB
	config            *config.Config
	slowQueryThreshold time.Duration
	stats             *QueryOptimizerStats
	mu                sync.RWMutex
	enabled           bool
}

type QueryOptimizerStats struct {
	TotalQueries      int64
	SlowQueries       int64
	CachedQueries     int64
	OptimizedQueries  int64
	AvgQueryTime      int64
	MaxQueryTime      int64
	LastSlowQueryTime time.Time
	QueryCountByType  map[string]int64
	mu                sync.RWMutex
}

var queryOptimizer *QueryOptimizer

func InitQueryOptimizer(db *gorm.DB, cfg *config.Config) {
	threshold := time.Duration(cfg.Database.SlowQueryThresholdMs) * time.Millisecond
	if threshold <= 0 {
		threshold = 100 * time.Millisecond
	}

	queryOptimizer = &QueryOptimizer{
		db:    db,
		config: cfg,
		slowQueryThreshold: threshold,
		enabled: cfg.Database.QueryOptimization.EnableQueryCache,
		stats: &QueryOptimizerStats{
			QueryCountByType: make(map[string]int64),
		},
	}

	if queryOptimizer.enabled {
		registerQueryCallbacks(db)
		log.Println("Query optimizer initialized with slow query threshold:", threshold)
	}
}

func GetQueryOptimizer() *QueryOptimizer {
	return queryOptimizer
}

func registerQueryCallbacks(db *gorm.DB) {
	db.Callback().Query().Before("gorm:query").Register("query_optimizer_before", func(db *gorm.DB) {
		db.InstanceSet("query_start_time", time.Now())
		db.InstanceSet("query_type", detectQueryType(db.Statement.SQL.String()))
	})

	db.Callback().Query().After("gorm:query").Register("query_optimizer_after", func(db *gorm.DB) {
		if optimizer := GetQueryOptimizer(); optimizer != nil {
			optimizer.recordQueryMetrics(db)
		}
	})

	db.Callback().Create().Before("gorm:create").Register("query_optimizer_create_before", func(db *gorm.DB) {
		db.InstanceSet("query_start_time", time.Now())
	})

	db.Callback().Create().After("gorm:create").Register("query_optimizer_create_after", func(db *gorm.DB) {
		if optimizer := GetQueryOptimizer(); optimizer != nil {
			optimizer.recordWriteMetrics(db, "INSERT")
		}
	})

	db.Callback().Update().Before("gorm:update").Register("query_optimizer_update_before", func(db *gorm.DB) {
		db.InstanceSet("query_start_time", time.Now())
	})

	db.Callback().Update().After("gorm:update").Register("query_optimizer_update_after", func(db *gorm.DB) {
		if optimizer := GetQueryOptimizer(); optimizer != nil {
			optimizer.recordWriteMetrics(db, "UPDATE")
		}
	})

	db.Callback().Delete().Before("gorm:delete").Register("query_optimizer_delete_before", func(db *gorm.DB) {
		db.InstanceSet("query_start_time", time.Now())
	})

	db.Callback().Delete().After("gorm:delete").Register("query_optimizer_delete_after", func(db *gorm.DB) {
		if optimizer := GetQueryOptimizer(); optimizer != nil {
			optimizer.recordWriteMetrics(db, "DELETE")
		}
	})
}

func detectQueryType(sql string) string {
	sql = strings.ToUpper(strings.TrimSpace(sql))
	if strings.HasPrefix(sql, "SELECT") {
		if strings.Contains(sql, "COUNT(") {
			return "SELECT_COUNT"
		}
		if strings.Contains(sql, "JOIN") {
			return "SELECT_JOIN"
		}
		return "SELECT"
	}
	if strings.HasPrefix(sql, "INSERT") {
		return "INSERT"
	}
	if strings.HasPrefix(sql, "UPDATE") {
		return "UPDATE"
	}
	if strings.HasPrefix(sql, "DELETE") {
		return "DELETE"
	}
	return "OTHER"
}

func (o *QueryOptimizer) recordQueryMetrics(db *gorm.DB) {
	startTime, ok := db.InstanceGet("query_start_time")
	if !ok {
		return
	}

	duration := time.Since(startTime.(time.Time))
	queryType, _ := db.InstanceGet("query_type")

	atomic.AddInt64(&o.stats.TotalQueries, 1)

	if queryTypeStr, ok := queryType.(string); ok {
		o.stats.mu.Lock()
		o.stats.QueryCountByType[queryTypeStr]++
		o.stats.mu.Unlock()
	}

	if duration > o.slowQueryThreshold {
		atomic.AddInt64(&o.stats.SlowQueries, 1)
		o.stats.LastSlowQueryTime = time.Now()
		log.Printf("[SLOW_QUERY] type=%s duration=%v sql=%s", queryType, duration, db.Statement.SQL.String())
	}

	currentAvg := atomic.LoadInt64(&o.stats.AvgQueryTime)
	totalQueries := atomic.LoadInt64(&o.stats.TotalQueries)
	newAvg := (currentAvg*(totalQueries-1) + duration.Milliseconds()) / totalQueries
	atomic.StoreInt64(&o.stats.AvgQueryTime, newAvg)

	maxTime := atomic.LoadInt64(&o.stats.MaxQueryTime)
	if duration.Milliseconds() > maxTime {
		atomic.StoreInt64(&o.stats.MaxQueryTime, duration.Milliseconds())
	}
}

func (o *QueryOptimizer) recordWriteMetrics(db *gorm.DB, operation string) {
	startTime, ok := db.InstanceGet("query_start_time")
	if !ok {
		return
	}

	duration := time.Since(startTime.(time.Time))
	if duration > o.slowQueryThreshold {
		log.Printf("[SLOW_WRITE] operation=%s duration=%v rows=%d", operation, duration, db.RowsAffected)
	}
}

func (o *QueryOptimizer) OptimizeQuery(ctx context.Context, query string, args ...interface{}) (*gorm.DB, error) {
	if !o.enabled {
		return o.db.WithContext(ctx).Raw(query, args...), nil
	}

	startTime := time.Now()
	result := o.db.WithContext(ctx).Raw(query, args...)
	duration := time.Since(startTime)

	if duration > o.slowQueryThreshold {
		atomic.AddInt64(&o.stats.SlowQueries, 1)
		optimized := o.analyzeAndOptimize(query, args)
		if optimized != query {
			atomic.AddInt64(&o.stats.OptimizedQueries, 1)
			log.Printf("[QUERY_OPTIMIZED] original=%s optimized=%s", query, optimized)
		}
	}

	return result, nil
}

func (o *QueryOptimizer) analyzeAndOptimize(query string, args []interface{}) string {
	optimized := query

	optimized = strings.ReplaceAll(optimized, "SELECT *", "SELECT <columns>")
	optimized = strings.ReplaceAll(optimized, "select *", "select <columns>")

	return optimized
}

func (o *QueryOptimizer) GetStats() *QueryOptimizerStats {
	stats := &QueryOptimizerStats{}
	stats.TotalQueries = atomic.LoadInt64(&o.stats.TotalQueries)
	stats.SlowQueries = atomic.LoadInt64(&o.stats.SlowQueries)
	stats.CachedQueries = atomic.LoadInt64(&o.stats.CachedQueries)
	stats.OptimizedQueries = atomic.LoadInt64(&o.stats.OptimizedQueries)
	stats.AvgQueryTime = atomic.LoadInt64(&o.stats.AvgQueryTime)
	stats.MaxQueryTime = atomic.LoadInt64(&o.stats.MaxQueryTime)
	stats.LastSlowQueryTime = o.stats.LastSlowQueryTime

	stats.mu.Lock()
	stats.QueryCountByType = make(map[string]int64)
	for k, v := range o.stats.QueryCountByType {
		stats.QueryCountByType[k] = v
	}
	stats.mu.Unlock()

	return stats
}

func (o *QueryOptimizer) ResetStats() {
	atomic.StoreInt64(&o.stats.TotalQueries, 0)
	atomic.StoreInt64(&o.stats.SlowQueries, 0)
	atomic.StoreInt64(&o.stats.CachedQueries, 0)
	atomic.StoreInt64(&o.stats.OptimizedQueries, 0)
	atomic.StoreInt64(&o.stats.AvgQueryTime, 0)
	atomic.StoreInt64(&o.stats.MaxQueryTime, 0)
	o.stats.LastSlowQueryTime = time.Time{}

	o.stats.mu.Lock()
	o.stats.QueryCountByType = make(map[string]int64)
	o.stats.mu.Unlock()
}

func (o *QueryOptimizer) SetSlowQueryThreshold(threshold time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.slowQueryThreshold = threshold
}

func (o *QueryOptimizer) Enable(enabled bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.enabled = enabled
}

func (o *QueryOptimizer) BatchQuery(ctx context.Context, query string, batchSize int, callback func(rows *gorm.DB) error) error {
	offset := 0
	for {
		var count int64
		o.db.WithContext(ctx).Raw(query+" LIMIT ? OFFSET ?", batchSize, offset).Scan(&count)

		if count == 0 {
			break
		}

		rows, err := o.db.WithContext(ctx).Raw(query+" LIMIT ? OFFSET ?", batchSize, offset).Rows()
		if err != nil {
			return err
		}

		for rows.Next() {
			var row gorm.DB
			if err := rows.Scan(&row); err != nil {
				rows.Close()
				return err
			}
			if err := callback(&row); err != nil {
				rows.Close()
				return err
			}
		}

		rows.Close()
		offset += batchSize

		if count < int64(batchSize) {
			break
		}
	}

	return nil
}

type QueryHint struct {
	EnableSeqScan    bool
	EnableIndexScan bool
	EnableHashJoin  bool
	EnableNestedLoop bool
	WorkMem         string
	RandomPageCost  string
}

func (o *QueryOptimizer) ExecuteWithHints(ctx context.Context, hints *QueryHint, query string, args ...interface{}) (*gorm.DB, error) {
	var hintStrings []string

	if hints != nil {
		if !hints.EnableSeqScan {
			hintStrings = append(hintStrings, "SET enable_seqscan = off")
		}
		if !hints.EnableIndexScan {
			hintStrings = append(hintStrings, "SET enable_indexscan = off")
		}
		if !hints.EnableHashJoin {
			hintStrings = append(hintStrings, "SET enable_hashjoin = off")
		}
		if !hints.EnableNestedLoop {
			hintStrings = append(hintStrings, "SET enable_nestloop = off")
		}
		if hints.WorkMem != "" {
			hintStrings = append(hintStrings, fmt.Sprintf("SET work_mem = '%s'", hints.WorkMem))
		}
		if hints.RandomPageCost != "" {
			hintStrings = append(hintStrings, fmt.Sprintf("SET random_page_cost = '%s'", hints.RandomPageCost))
		}
	}

	for _, hint := range hintStrings {
		if err := o.db.Exec(hint).Error; err != nil {
			log.Printf("Failed to set hint: %s, error: %v", hint, err)
		}
	}

	result := o.db.WithContext(ctx).Raw(query, args...)

	for _, hint := range hintStrings {
		resetHint := strings.Replace(hint, "SET ", "SET ", 1)
		if strings.Contains(hint, "enable_seqscan") {
			resetHint = "SET enable_seqscan = on"
		} else if strings.Contains(hint, "enable_indexscan") {
			resetHint = "SET enable_indexscan = on"
		} else if strings.Contains(hint, "enable_hashjoin") {
			resetHint = "SET enable_hashjoin = on"
		} else if strings.Contains(hint, "enable_nestloop") {
			resetHint = "SET enable_nestloop = on"
		} else {
			resetHint = ""
		}

		if resetHint != "" {
			o.db.Exec(resetHint)
		}
	}

	return result, nil
}

func ExplainQueryPlan(ctx context.Context, query string, args ...interface{}) (string, error) {
	db := GetDB()
	if db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	var result string
	err := db.WithContext(ctx).Raw("EXPLAIN (FORMAT TEXT) "+query, args...).Scan(&result).Error
	if err != nil {
		return "", err
	}

	return result, nil
}

func ExplainQueryAnalyze(ctx context.Context, query string, args ...interface{}) (string, error) {
	db := GetDB()
	if db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	var result string
	err := db.WithContext(ctx).Raw("EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) "+query, args...).Scan(&result).Error
	if err != nil {
		return "", err
	}

	return result, nil
}

func GetQueryPlan(ctx context.Context, query string, args ...interface{}) (*QueryPlan, error) {
	explainResult, err := ExplainQueryAnalyze(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	plan := &QueryPlan{
		RawPlan:    explainResult,
		TotalCost:  parseTotalCost(explainResult),
		PlanTime:   parsePlanTime(explainResult),
		ActualRows: parseActualRows(explainResult),
	}

	return plan, nil
}

type QueryPlan struct {
	RawPlan    string
	TotalCost  float64
	PlanTime   float64
	ActualRows int64
	Nodes      []*PlanNode
}

type PlanNode struct {
	NodeType string
	Relation string
	Cost     float64
	Rows     int64
	Buffers  int64
}

func parseTotalCost(explainResult string) float64 {
	return 0
}

func parsePlanTime(explainResult string) float64 {
	return 0
}

func parseActualRows(explainResult string) int64 {
	return 0
}

func (o *QueryOptimizer) AnalyzeAndRecommendIndexes(ctx context.Context, tableName string) ([]IndexRecommendation, error) {
	var recommendations []IndexRecommendation

	var slowQueries []struct {
		Query   string
		Calls   int64
		MeanTime float64
		Rows    int64
	}

	db := GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if err := db.WithContext(ctx).Raw(`
		SELECT query, calls, mean_time, rows
		FROM pg_stat_statements
		WHERE query LIKE ?
		ORDER BY mean_time DESC
		LIMIT 10
	`, "%"+tableName+"%").Scan(&slowQueries).Error; err != nil {
		return nil, err
	}

	for _, sq := range slowQueries {
		if sq.MeanTime > 10 {
			rec := IndexRecommendation{
				TableName:      tableName,
				Query:          sq.Query,
				EstimatedTime:  sq.MeanTime,
				PotentialGain:  sq.MeanTime * float64(sq.Calls) * 0.3,
				SuggestedIndex: generateIndexSuggestion(sq.Query, tableName),
				Priority:       determinePriority(sq.MeanTime, sq.Calls),
			}
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations, nil
}

type IndexRecommendation struct {
	TableName       string
	Query           string
	EstimatedTime   float64
	PotentialGain   float64
	SuggestedIndex  string
	Priority        string
}

func generateIndexSuggestion(query, tableName string) string {
	return fmt.Sprintf("CREATE INDEX idx_%s_optimized ON %s (<columns>)", tableName, tableName)
}

func determinePriority(meanTime float64, calls int64) string {
	impact := meanTime * float64(calls)
	if impact > 10000 {
		return "high"
	} else if impact > 1000 {
		return "medium"
	}
	return "low"
}
