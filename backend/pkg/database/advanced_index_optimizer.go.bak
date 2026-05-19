package database

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

type AdvancedIndexOptimizer struct {
	db                *gorm.DB
	indexHistory      []IndexChangeRecord
	maxHistoryLen     int
	autoOptimizeEnabled bool
	optimizationInterval time.Duration
}

type IndexChangeRecord struct {
	Timestamp      time.Time
	IndexName     string
	TableName     string
	Action        string
	Reason        string
	Success       bool
	ExecutionTime time.Duration
}

type IndexOptimizationPlan struct {
	IndexesToCreate []IndexCreationPlan
	IndexesToDrop   []IndexDropPlan
	TablesToAnalyze []string
	EstimatedImpact string
	RiskLevel      string
}

type IndexCreationPlan struct {
	IndexName     string
	TableName     string
	Columns       []string
	IndexType     string
	Unique        bool
	WhereClause   string
	Priority      string
	Reason        string
	EstimatedSize string
	EstimatedCost string
}

type IndexDropPlan struct {
	IndexName     string
	TableName     string
	Reason        string
	CurrentSize   string
	ScanCount     int64
	Impact        string
}

var advancedOptimizer *AdvancedIndexOptimizer

func InitAdvancedIndexOptimizer(db *gorm.DB) {
	advancedOptimizer = &AdvancedIndexOptimizer{
		db:                   db,
		indexHistory:         make([]IndexChangeRecord, 0),
		maxHistoryLen:        1000,
		autoOptimizeEnabled:   true,
		optimizationInterval: 24 * time.Hour,
	}

	go advancedOptimizer.runPeriodicOptimization()
	log.Println("Advanced index optimizer initialized with auto-optimization")
}

func GetAdvancedIndexOptimizer() *AdvancedIndexOptimizer {
	if advancedOptimizer == nil {
		return nil
	}
	return advancedOptimizer
}

func (o *AdvancedIndexOptimizer) runPeriodicOptimization() {
	if !o.autoOptimizeEnabled {
		return
	}

	ticker := time.NewTicker(o.optimizationInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		if err := o.AutomatedIndexOptimization(ctx); err != nil {
			log.Printf("[INDEX_OPT] Periodic optimization failed: %v", err)
		}
	}
}

func (o *AdvancedIndexOptimizer) AutomatedIndexOptimization(ctx context.Context) error {
	log.Println("[INDEX_OPT] Starting automated index optimization...")

	plan, err := o.CreateOptimizationPlan()
	if err != nil {
		return fmt.Errorf("failed to create optimization plan: %w", err)
	}

	if len(plan.IndexesToCreate) > 0 {
		log.Printf("[INDEX_OPT] Creating %d new indexes", len(plan.IndexesToCreate))
		for _, idx := range plan.IndexesToCreate {
			if idx.Priority == "high" {
				if err := o.createIndexSafely(idx); err != nil {
					log.Printf("[INDEX_OPT] Failed to create index %s: %v", idx.IndexName, err)
				}
			}
		}
	}

	if len(plan.IndexesToDrop) > 0 {
		log.Printf("[INDEX_OPT] Found %d unused indexes to potentially drop", len(plan.IndexesToDrop))
		for _, idx := range plan.IndexesToDrop {
			if idx.ScanCount == 0 && idx.Impact == "low" {
				log.Printf("[INDEX_OPT] Index %s is unused, consider dropping", idx.IndexName)
			}
		}
	}

	o.analyzeTables(plan.TablesToAnalyze)

	log.Println("[INDEX_OPT] Automated optimization completed")
	return nil
}

func (o *AdvancedIndexOptimizer) CreateOptimizationPlan() (*IndexOptimizationPlan, error) {
	plan := &IndexOptimizationPlan{
		IndexesToCreate: o.recommendNewIndexes(),
		IndexesToDrop:   o.findIndexesToDrop(),
		TablesToAnalyze: []string{},
	}

	unusedIndexes, err := o.findUnusedIndexes()
	if err == nil && len(unusedIndexes) > 0 {
		plan.EstimatedImpact = "medium"
		plan.RiskLevel = "low"
	}

	if len(plan.IndexesToCreate) > 10 {
		plan.EstimatedImpact = "high"
		plan.RiskLevel = "medium"
	}

	return plan, nil
}

func (o *AdvancedIndexOptimizer) recommendNewIndexes() []IndexCreationPlan {
	var recommendations []IndexCreationPlan

	recommendations = append(recommendations,
		IndexCreationPlan{
			IndexName:     "idx_audit_logs_user_action",
			TableName:     "audit_logs",
			Columns:       []string{"user_id", "action", "created_at"},
			IndexType:     "btree",
			Priority:      "high",
			Reason:        "Frequent query pattern for user audit trail",
			EstimatedSize: "50MB",
		},
		IndexCreationPlan{
			IndexName:     "idx_risk_logs_severity_created",
			TableName:     "risk_logs",
			Columns:       []string{"severity", "created_at"},
			IndexType:     "btree",
			Priority:      "medium",
			Reason:        "Risk log filtering and analytics",
			EstimatedSize: "80MB",
		},
		IndexCreationPlan{
			IndexName:     "idx_user_sessions_expired",
			TableName:     "sessions",
			Columns:       []string{"user_id", "expired_at"},
			IndexType:     "btree",
			Priority:      "high",
			Reason:        "Session cleanup and validation queries",
			EstimatedSize: "30MB",
		},
		IndexCreationPlan{
			IndexName:     "idx_admin_login_admin_created",
			TableName:     "admin_login_logs",
			Columns:       []string{"admin_id", "created_at"},
			IndexType:     "btree",
			Priority:      "medium",
			Reason:        "Admin login history queries",
			EstimatedSize: "20MB",
		},
		IndexCreationPlan{
			IndexName:     "idx_api_keys_active_expires",
			TableName:     "api_keys",
			Columns:       []string{"is_active", "expires_at"},
			IndexType:     "btree",
			Priority:      "high",
			Reason:        "API key validation and cleanup",
			EstimatedSize: "10MB",
		},
		IndexCreationPlan{
			IndexName:     "idx_rate_limits_key_window",
			TableName:     "rate_limits",
			Columns:       []string{"key", "window_start"},
			IndexType:     "btree",
			Priority:      "high",
			Reason:        "Rate limiting queries",
			EstimatedSize: "15MB",
		},
		IndexCreationPlan{
			IndexName:     "idx_notifications_user_read",
			TableName:     "notifications",
			Columns:       []string{"user_id", "is_read", "created_at"},
			IndexType:     "btree",
			Priority:      "medium",
			Reason:        "User notification queries",
			EstimatedSize: "40MB",
		},
		IndexCreationPlan{
			IndexName:     "idx_webhook_deliveries_status",
			TableName:     "webhook_deliveries",
			Columns:       []string{"status", "next_retry_at"},
			IndexType:     "btree",
			Priority:      "medium",
			Reason:        "Webhook retry processing",
			EstimatedSize: "25MB",
		},
	)

	return recommendations
}

func (o *AdvancedIndexOptimizer) findIndexesToDrop() []IndexDropPlan {
	var toDrop []IndexDropPlan

	var unusedIndexes []struct {
		IndexName string
		TableName string
		ScanCount int64
	}

	err := o.db.Raw(`
		SELECT
			indexname AS index_name,
			tablename AS table_name,
			COALESCE(idx_scan, 0) AS scan_count
		FROM pg_stat_user_indexes
		WHERE idx_scan = 0
		AND indexname NOT LIKE '%_pkey'
		AND indexname NOT LIKE '%_fkey'
		AND indexname NOT LIKE '%_xmin'
	`).Scan(&unusedIndexes).Error

	if err != nil {
		log.Printf("[INDEX_OPT] Failed to find unused indexes: %v", err)
		return toDrop
	}

	for _, idx := range unusedIndexes {
		var size string
		o.db.Raw(`
			SELECT pg_size_pretty(pg_relation_size(indexrelid))
			FROM pg_stat_user_indexes
			WHERE indexrelname = ?
		`, idx.IndexName).Scan(&size)

		toDrop = append(toDrop, IndexDropPlan{
			IndexName:   idx.IndexName,
			TableName:   idx.TableName,
			Reason:      "Index has never been scanned",
			CurrentSize: size,
			ScanCount:   idx.ScanCount,
			Impact:      "low",
		})
	}

	return toDrop
}

func (o *AdvancedIndexOptimizer) findUnusedIndexes() ([]string, error) {
	var unused []string

	err := o.db.Raw(`
		SELECT indexname
		FROM pg_stat_user_indexes
		WHERE idx_scan = 0
		AND indexname NOT LIKE '%_pkey'
		AND indexname NOT LIKE '%_fkey'
	`).Scan(&unused).Error

	return unused, err
}

func (o *AdvancedIndexOptimizer) createIndexSafely(plan IndexCreationPlan) error {
	var exists int64
	o.db.Raw("SELECT COUNT(*) FROM pg_indexes WHERE indexname = ?", plan.IndexName).Scan(&exists)

	if exists > 0 {
		log.Printf("[INDEX_OPT] Index %s already exists, skipping", plan.IndexName)
		return nil
	}

	var columnsStr string
	for i, col := range plan.Columns {
		if i > 0 {
			columnsStr += ", "
		}
		columnsStr += col
	}

	var sql string
	if plan.WhereClause != "" {
		sql = fmt.Sprintf("CREATE INDEX CONCURRENTLY %s ON %s (%s) WHERE %s",
			plan.IndexName, plan.TableName, columnsStr, plan.WhereClause)
	} else {
		sql = fmt.Sprintf("CREATE INDEX CONCURRENTLY %s ON %s (%s)",
			plan.IndexName, plan.TableName, columnsStr)
	}

	start := time.Now()
	err := o.db.Exec(sql).Error
	duration := time.Since(start)

	record := IndexChangeRecord{
		Timestamp:      time.Now(),
		IndexName:      plan.IndexName,
		TableName:      plan.TableName,
		Action:        "CREATE",
		Success:       err == nil,
		ExecutionTime: duration,
	}

	o.recordChange(record)

	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	log.Printf("[INDEX_OPT] Successfully created index %s in %v", plan.IndexName, duration)
	return nil
}

func (o *AdvancedIndexOptimizer) analyzeTables(tableNames []string) {
	for _, table := range tableNames {
		if err := o.db.Exec("ANALYZE " + table).Error; err != nil {
			log.Printf("[INDEX_OPT] Failed to analyze table %s: %v", table, err)
		} else {
			log.Printf("[INDEX_OPT] Analyzed table %s", table)
		}
	}
}

func (o *AdvancedIndexOptimizer) recordChange(record IndexChangeRecord) {
	o.indexHistory = append(o.indexHistory, record)
	if len(o.indexHistory) > o.maxHistoryLen {
		o.indexHistory = o.indexHistory[1:]
	}
}

func (o *AdvancedIndexOptimizer) GetIndexHistory() []IndexChangeRecord {
	return o.indexHistory
}

func (o *AdvancedIndexOptimizer) GetOptimizationPlanDetailed() (map[string]interface{}, error) {
	result := make(map[string]interface{})

	stats, err := o.getIndexStatistics()
	if err == nil {
		result["statistics"] = stats
	}

	recommendations := o.recommendNewIndexes()
	result["recommendations"] = recommendations
	result["recommendation_count"] = len(recommendations)

	toDrop := o.findIndexesToDrop()
	result["potential_drops"] = toDrop
	result["potential_drop_count"] = len(toDrop)

	result["auto_optimize_enabled"] = o.autoOptimizeEnabled
	result["last_optimization"] = o.indexHistory

	return result, nil
}

func (o *AdvancedIndexOptimizer) getIndexStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalIndexes, totalSize int64
	var unusedIndexes, redundantIndexes int

	o.db.Raw("SELECT COUNT(*) FROM pg_stat_user_indexes").Scan(&totalIndexes)
	o.db.Raw("SELECT COUNT(*) FROM pg_stat_user_indexes WHERE idx_scan = 0").Scan(&unusedIndexes)

	stats["total_indexes"] = totalIndexes
	stats["unused_indexes"] = unusedIndexes
	stats["used_indexes"] = totalIndexes - int64(unusedIndexes)

	if totalIndexes > 0 {
		stats["usage_percentage"] = float64(totalIndexes-int64(unusedIndexes)) / float64(totalIndexes) * 100
	}

	return stats, nil
}

func (o *AdvancedIndexOptimizer) SetAutoOptimize(enabled bool) {
	o.autoOptimizeEnabled = enabled
	log.Printf("[INDEX_OPT] Auto-optimization %s", map[bool]string{true: "enabled", false: "disabled"}[enabled])
}

func (o *AdvancedIndexOptimizer) SafeDropIndex(indexName string, force bool) error {
	var count int64
	o.db.Raw("SELECT COUNT(*) FROM pg_indexes WHERE indexname = ?", indexName).Scan(&count)

	if count == 0 {
		return fmt.Errorf("index %s does not exist", indexName)
	}

	if strings.HasSuffix(indexName, "_pkey") || strings.HasSuffix(indexName, "_fkey") {
		return fmt.Errorf("cannot drop primary key or foreign key index")
	}

	var scanCount int64
	o.db.Raw("SELECT COALESCE(idx_scan, 0) FROM pg_stat_user_indexes WHERE indexrelname = ?", indexName).Scan(&scanCount)

	if scanCount > 0 && !force {
		return fmt.Errorf("index is still being used (scan count: %d), use force flag to drop", scanCount)
	}

	sql := fmt.Sprintf("DROP INDEX CONCURRENTLY IF EXISTS %s", indexName)
	err := o.db.Exec(sql).Error

	if err == nil {
		record := IndexChangeRecord{
			Timestamp: time.Now(),
			IndexName: indexName,
			Action:    "DROP",
			Success:   true,
		}
		o.recordChange(record)
	}

	return err
}

type CompositeIndexOptimizer struct {
	db *gorm.DB
}

func NewCompositeIndexOptimizer(db *gorm.DB) *CompositeIndexOptimizer {
	return &CompositeIndexOptimizer{db: db}
}

func (o *CompositeIndexOptimizer) RecommendCompositeIndexes() []map[string]interface{} {
	var recommendations []map[string]interface{}

	recommendations = append(recommendations,
		map[string]interface{}{
			"table":      "verification_logs",
			"columns":    []string{"application_id", "status", "created_at"},
			"reason":     "Dashboard queries with multiple filters",
			"priority":   "high",
			"estimated":  "200MB",
		},
		map[string]interface{}{
			"table":      "captcha_sessions",
			"columns":    []string{"status", "expired_at", "created_at"},
			"reason":     "Session cleanup queries",
			"priority":   "high",
			"estimated":  "100MB",
		},
		map[string]interface{}{
			"table":      "blacklist",
			"columns":    []string{"target", "type", "status"},
			"reason":     "Blacklist lookup optimization",
			"priority":   "high",
			"estimated":  "50MB",
		},
		map[string]interface{}{
			"table":      "behavior_data",
			"columns":    []string{"user_id", "session_id", "created_at"},
			"reason":     "User behavior analysis",
			"priority":   "medium",
			"estimated":  "300MB",
		},
		map[string]interface{}{
			"table":      "trace_records",
			"columns":    []string{"session_id", "trace_type", "created_at"},
			"reason":     "Trace query optimization",
			"priority":   "medium",
			"estimated":  "150MB",
		},
	)

	return recommendations
}

func (o *CompositeIndexOptimizer) AnalyzeQueryPatterns() []map[string]interface{} {
	var patterns []map[string]interface{}

	patterns = append(patterns,
		map[string]interface{}{
			"pattern":    "WHERE application_id = ? AND status = ? AND created_at > ?",
			"table":      "verification_logs",
			"frequency":  "very_high",
			"recommendation": "Composite index on (application_id, status, created_at)",
		},
		map[string]interface{}{
			"pattern":    "WHERE user_id = ? AND status = 'active'",
			"table":      "sessions",
			"frequency":  "high",
			"recommendation": "Composite index on (user_id, status)",
		},
		map[string]interface{}{
			"pattern":    "WHERE session_id = ? ORDER BY created_at DESC",
			"table":      "captcha_sessions",
			"frequency":  "high",
			"recommendation": "Composite index on (session_id, created_at DESC)",
		},
		map[string]interface{}{
			"pattern":    "WHERE target = ? AND type = ?",
			"table":      "blacklist",
			"frequency":  "very_high",
			"recommendation": "Composite index on (target, type)",
		},
	)

	return patterns
}

var globalCompositeOptimizer *CompositeIndexOptimizer

func init() {
	if db := GetDB(); db != nil {
		globalCompositeOptimizer = NewCompositeIndexOptimizer(db)
	}
}

func GetCompositeIndexOptimizer() *CompositeIndexOptimizer {
	return globalCompositeOptimizer
}
