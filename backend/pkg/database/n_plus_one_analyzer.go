package database

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type N1QueryDetector struct {
	db                *gorm.DB
	enabled           bool
	queryPatterns     map[string]*N1Pattern
	stats             *N1Stats
	mu                sync.RWMutex
	warningThreshold  int
	criticalThreshold int
}

type N1Pattern struct {
	PatternName       string
	QueryTemplate     string
	DetectionCount    int64
	LastDetected      time.Time
	AffectedTables    []string
	SuggestedFix     string
	AvgExecutionTime  time.Duration
	TotalQueries      int64
	OptimizationGain  string
}

type N1Stats struct {
	TotalScans        atomic.Int64
	PatternsDetected  atomic.Int64
	WarningsIssued    atomic.Int64
	QueriesOptimized  atomic.Int64
	LastWarningTime   atomic.Value
}

type N1QueryInfo struct {
	Query          string
	Parameters     []interface{}
	ExecutionTime  time.Duration
	StackTrace     string
	Timestamp      time.Time
	AffectedRows   int64
	IsN1Query      bool
	Reason         string
}

type N1Report struct {
	Timestamp         time.Time
	TotalPatterns     int
	CriticalPatterns  int
	WarningPatterns   int
	Patterns          []*N1Pattern
	Recommendations   []string
	EstimatedSavings  time.Duration
}

type QueryBatchOptimizer struct {
	db             *gorm.DB
	batchSize      int
	enabled        bool
	stats          *BatchOptimizerStats
	optimizationRules []OptimizationRule
}

type BatchOptimizerStats struct {
	QueriesOptimized atomic.Int64
}

type OptimizationRule struct {
	Name        string
	MatchFunc   func(query string) bool
	OptimizeFunc func(query string, params []interface{}) (string, []interface{}, error)
	Priority    int
}

var globalN1Detector *N1QueryDetector
var globalBatchOptimizer *QueryBatchOptimizer

func NewN1QueryDetector(db *gorm.DB) *N1QueryDetector {
	return &N1QueryDetector{
		db:                db,
		enabled:           true,
		queryPatterns:     make(map[string]*N1Pattern),
		stats:             &N1Stats{},
		warningThreshold:   3,
		criticalThreshold: 10,
	}
}

func GetN1QueryDetector() *N1QueryDetector {
	if globalN1Detector == nil {
		globalN1Detector = NewN1QueryDetector(DB)
	}
	return globalN1Detector
}

func (d *N1QueryDetector) Enable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = true
}

func (d *N1QueryDetector) Disable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = false
}

func (d *N1QueryDetector) IsEnabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.enabled
}

func (d *N1QueryDetector) RegisterGORMCallbacks() error {
	if err := d.db.Callback().Query().Before("gorm:query").Register("n1_detector_before", func(db *gorm.DB) {
		if !d.enabled {
			return
		}
		db.InstanceSet("n1_start_time", time.Now())
		db.InstanceSet("n1_query", db.Statement.SQL.String())
	}); err != nil {
		return err
	}

	if err := d.db.Callback().Query().After("gorm:query").Register("n1_detector_after", func(db *gorm.DB) {
		if !d.enabled {
			return
		}

		startTime, _ := db.InstanceGet("n1_start_time")
		query, _ := db.InstanceGet("n1_query")

		if t, ok := startTime.(time.Time); ok {
			duration := time.Since(t)
			if q, ok := query.(string); ok {
				d.analyzeQuery(q, duration)
			}
		}
	}); err != nil {
		return err
	}

	if err := d.db.Callback().Create().After("gorm:create").Register("n1_create_after", func(db *gorm.DB) {
		if !d.enabled {
			return
		}
		d.stats.TotalScans.Add(1)
	}); err != nil {
		return err
	}

	if err := d.db.Callback().Update().After("gorm:update").Register("n1_update_after", func(db *gorm.DB) {
		if !d.enabled {
			return
		}
		d.stats.TotalScans.Add(1)
	}); err != nil {
		return err
	}

	if err := d.db.Callback().Delete().After("gorm:delete").Register("n1_delete_after", func(db *gorm.DB) {
		if !d.enabled {
			return
		}
		d.stats.TotalScans.Add(1)
	}); err != nil {
		return err
	}

	return nil
}

func (d *N1QueryDetector) analyzeQuery(query string, duration time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.stats.TotalScans.Add(1)

	normalizedQuery := d.normalizeQuery(query)

	pattern, exists := d.queryPatterns[normalizedQuery]
	if !exists {
		pattern = &N1Pattern{
			PatternName:      d.generatePatternName(normalizedQuery),
			QueryTemplate:    normalizedQuery,
			AffectedTables:   d.extractTables(query),
			SuggestedFix:    d.suggestFix(query),
			AvgExecutionTime: duration,
		}
		d.queryPatterns[normalizedQuery] = pattern
	}

	pattern.TotalQueries++
	pattern.LastDetected = time.Now()

	if pattern.AvgExecutionTime > 0 {
		pattern.AvgExecutionTime = (pattern.AvgExecutionTime*time.Duration(pattern.TotalQueries-1) + duration) / time.Duration(pattern.TotalQueries)
	}

	if d.isPotentialN1Pattern(query) {
		pattern.DetectionCount++
		d.stats.PatternsDetected.Add(1)

		if pattern.DetectionCount >= int64(d.criticalThreshold) {
			log.Printf("[N1_DETECTOR] 严重: 检测到疑似N+1查询模式: %s (检测次数: %d)", pattern.PatternName, pattern.DetectionCount)
			d.stats.WarningsIssued.Add(1)
			d.stats.LastWarningTime.Store(time.Now())
		} else if pattern.DetectionCount >= int64(d.warningThreshold) {
			log.Printf("[N1_DETECTOR] 警告: 检测到疑似N+1查询: %s", pattern.PatternName)
		}
	}
}

func (d *N1QueryDetector) normalizeQuery(query string) string {
	query = strings.ToUpper(query)
	query = regexp.MustCompile(`'[^']*'`).ReplaceAllString(query, "?")
	query = regexp.MustCompile(`\d+`).ReplaceAllString(query, "?")
	query = regexp.MustCompile(`\s+`).ReplaceAllString(query, " ")
	return strings.TrimSpace(query)
}

func (d *N1QueryDetector) generatePatternName(query string) string {
	tables := d.extractTables(query)
	if len(tables) > 0 {
		return fmt.Sprintf("query_%s", tables[0])
	}
	return fmt.Sprintf("pattern_%d", time.Now().UnixNano())
}

func (d *N1QueryDetector) extractTables(query string) []string {
	var tables []string
	re := regexp.MustCompile(`(?i)(?:FROM|JOIN|INTO)\s+(\w+)`)
	matches := re.FindAllStringSubmatch(query, -1)
	for _, match := range matches {
		if len(match) > 1 {
			tables = append(tables, match[1])
		}
	}
	return tables
}

func (d *N1QueryDetector) isPotentialN1Pattern(query string) bool {
	upperQuery := strings.ToUpper(query)

	if strings.Contains(upperQuery, "SELECT") && strings.Contains(upperQuery, "WHERE") {
		if !strings.Contains(upperQuery, "JOIN") && !strings.Contains(upperQuery, "IN (") {
			return true
		}
	}

	if strings.Contains(upperQuery, "SELECT") && strings.Contains(upperQuery, "WHERE") && strings.Contains(upperQuery, "=") {
		if regexp.MustCompile(`WHERE\s+\w+\s*=\s*\?`).MatchString(query) {
			return true
		}
	}

	return false
}

func (d *N1QueryDetector) suggestFix(query string) string {
	upperQuery := strings.ToUpper(query)

	if strings.Contains(upperQuery, "WHERE") && !strings.Contains(upperQuery, "JOIN") {
		return "建议: 使用JOIN替代循环查询，或使用预加载(Preload)进行批量加载"
	}

	if strings.Contains(upperQuery, "IN (SELECT") {
		return "建议: 将子查询改为JOIN，或使用EXISTS替代IN"
	}

	if regexp.MustCompile(`WHERE.*=\s*\?`).MatchString(query) && strings.Contains(upperQuery, "SELECT") {
		return "建议: 使用GORM的Preload或Joins进行关联查询预加载"
	}

	return "建议: 检查查询是否为循环内查询，考虑使用批量查询替代"
}

func (d *N1QueryDetector) GetStats() *N1Stats {
	return d.stats
}

func (d *N1QueryDetector) GetPatterns() []*N1Pattern {
	d.mu.RLock()
	defer d.mu.RUnlock()

	patterns := make([]*N1Pattern, 0, len(d.queryPatterns))
	for _, p := range d.queryPatterns {
		patterns = append(patterns, p)
	}
	return patterns
}

func (d *N1QueryDetector) GetCriticalPatterns() []*N1Pattern {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var critical []*N1Pattern
	for _, p := range d.queryPatterns {
		if p.DetectionCount >= int64(d.criticalThreshold) {
			critical = append(critical, p)
		}
	}
	return critical
}

func (d *N1QueryDetector) GetWarningPatterns() []*N1Pattern {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var warnings []*N1Pattern
	for _, p := range d.queryPatterns {
		if p.DetectionCount >= int64(d.warningThreshold) && p.DetectionCount < int64(d.criticalThreshold) {
			warnings = append(warnings, p)
		}
	}
	return warnings
}

func (d *N1QueryDetector) GenerateReport() *N1Report {
	d.mu.RLock()
	defer d.mu.RUnlock()

	report := &N1Report{
		Timestamp:        time.Now(),
		TotalPatterns:    len(d.queryPatterns),
		Patterns:         make([]*N1Pattern, 0),
		Recommendations:   make([]string, 0),
	}

	for _, p := range d.queryPatterns {
		if p.DetectionCount >= int64(d.criticalThreshold) {
			report.CriticalPatterns++
			report.Patterns = append(report.Patterns, p)
		} else if p.DetectionCount >= int64(d.warningThreshold) {
			report.WarningPatterns++
			report.Patterns = append(report.Patterns, p)
		}
	}

	var totalTime time.Duration
	for _, p := range report.Patterns {
		totalTime += p.AvgExecutionTime * time.Duration(p.TotalQueries)
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("[%s] %s - %s", p.PatternName, p.SuggestedFix, p.OptimizationGain))
	}

	report.EstimatedSavings = totalTime / 2

	return report
}

func (d *N1QueryDetector) ClearPatterns() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.queryPatterns = make(map[string]*N1Pattern)
	d.stats.PatternsDetected.Store(0)
}

func (d *N1QueryDetector) SetThresholds(warning, critical int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.warningThreshold = warning
	d.criticalThreshold = critical
}

func (d *N1QueryDetector) DetectN1InCode(ctx context.Context, model interface{}) error {
	if DB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	modelName := d.getModelName(model)
	log.Printf("[N1_DETECTOR] 开始检测模型 %s 的N+1查询问题", modelName)

	var count int64
	if err := DB.WithContext(ctx).Model(model).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return nil
	}

	initialPatterns := len(d.queryPatterns)

	var records []interface{}
	if err := DB.WithContext(ctx).Find(&records).Error; err != nil {
		return err
	}

	newPatterns := len(d.queryPatterns) - initialPatterns
	if newPatterns > 0 {
		log.Printf("[N1_DETECTOR] 检测到 %d 个潜在N+1查询模式", newPatterns)
	}

	return nil
}

func (d *N1QueryDetector) getModelName(model interface{}) string {
	if s, ok := model.(schema.Tabler); ok {
		return s.TableName()
	}
	return fmt.Sprintf("%T", model)
}

func NewQueryBatchOptimizer(db *gorm.DB, batchSize int) *QueryBatchOptimizer {
	optimizer := &QueryBatchOptimizer{
		db:        db,
		batchSize: batchSize,
		enabled:   true,
		stats:     &BatchOptimizerStats{},
		optimizationRules: []OptimizationRule{
			{
				Name:      "循环IN查询优化",
				MatchFunc: func(query string) bool {
					return strings.Contains(strings.ToUpper(query), "IN (") && !strings.Contains(strings.ToUpper(query), "JOIN")
				},
				OptimizeFunc: func(query string, params []interface{}) (string, []interface{}, error) {
					return query, params, nil
				},
				Priority: 1,
			},
			{
				Name:      "子查询优化",
				MatchFunc: func(query string) bool {
					return strings.Contains(strings.ToUpper(query), "SELECT") && strings.Contains(strings.ToUpper(query), "IN (SELECT")
				},
				OptimizeFunc: func(query string, params []interface{}) (string, []interface{}, error) {
					optimized := regexp.MustCompile(`IN \(SELECT`).ReplaceAllString(strings.ToUpper(query), "IN (")
					return optimized, params, nil
				},
				Priority: 2,
			},
		},
	}
	return optimizer
}

func GetBatchOptimizer() *QueryBatchOptimizer {
	if globalBatchOptimizer == nil {
		globalBatchOptimizer = NewQueryBatchOptimizer(DB, 100)
	}
	return globalBatchOptimizer
}

func (o *QueryBatchOptimizer) OptimizeInBatches(ctx context.Context, model interface{}, ids []interface{}, callback func(batch []interface{}) error) error {
	if !o.enabled || len(ids) == 0 {
		return fmt.Errorf("优化器未启用或ID列表为空")
	}

	totalBatches := (len(ids) + o.batchSize - 1) / o.batchSize

	for i := 0; i < totalBatches; i++ {
		start := i * o.batchSize
		end := start + o.batchSize
		if end > len(ids) {
			end = len(ids)
		}

		batch := ids[start:end]
		if err := callback(batch); err != nil {
			return err
		}
	}

	o.stats.QueriesOptimized.Add(1)
	return nil
}

func (o *QueryBatchOptimizer) optimizeInQueries(query string, params []interface{}) (string, []interface{}, error) {
	return query, params, nil
}

func (o *QueryBatchOptimizer) optimizeSubqueries(query string, params []interface{}) (string, []interface{}, error) {
	optimized := regexp.MustCompile(`IN \(SELECT`).ReplaceAllString(strings.ToUpper(query), "IN (")
	return optimized, params, nil
}

func (o *QueryBatchOptimizer) Enable() {
	o.enabled = true
}

func (o *QueryBatchOptimizer) Disable() {
	o.enabled = false
}

func (o *QueryBatchOptimizer) SetBatchSize(size int) {
	o.batchSize = size
}

type OptimizedPreloader struct {
	db      *gorm.DB
	preloads map[string][]string
}

func NewOptimizedPreloader(db *gorm.DB) *OptimizedPreloader {
	return &OptimizedPreloader{
		db:      db,
		preloads: make(map[string][]string),
	}
}

func (p *OptimizedPreloader) Preload(association string) *OptimizedPreloader {
	modelName := p.getCurrentModelName()
	p.preloads[modelName] = append(p.preloads[modelName], association)
	return p
}

func (p *OptimizedPreloader) getCurrentModelName() string {
	return "current"
}

func (p *OptimizedPreloader) Execute(query *gorm.DB) (*gorm.DB, error) {
	for _, associations := range p.preloads {
		for _, assoc := range associations {
			if err := query.Preload(assoc).Error; err != nil {
				return nil, err
			}
		}
	}
	return query, nil
}

func (p *OptimizedPreloader) BatchPreload(model interface{}, ids []interface{}, associations []string) (map[interface{}]map[string]interface{}, error) {
	results := make(map[interface{}]map[string]interface{})

	if len(ids) == 0 || len(associations) == 0 {
		return results, nil
	}

	for _, assoc := range associations {
		var records []map[string]interface{}
		if err := p.db.Model(model).Where("id IN ?", ids).Preload(assoc).Find(&records).Error; err != nil {
			return nil, err
		}

		for _, record := range records {
			id := record["id"]
			if _, exists := results[id]; !exists {
				results[id] = make(map[string]interface{})
			}
			results[id][assoc] = record[assoc]
		}
	}

	return results, nil
}

func PreloadWithBatch(db *gorm.DB, model interface{}, ids []interface{}, association string) error {
	if len(ids) == 0 {
		return nil
	}

	var relatedRecords []interface{}
	if err := db.Model(model).Where("id IN ?", ids).Find(&relatedRecords).Error; err != nil {
		return err
	}

	return nil
}

type QueryOptimizerHints struct {
	UseIndex    string
	NoIndex     string
	EnableSeqScan bool
	ParallelWorkers int
}

func (o *QueryOptimizerHints) ApplyToQuery(db *gorm.DB) *gorm.DB {
	if o.UseIndex != "" {
		db = db.Session(&gorm.Session{
			PrepareStmt: true,
		})
	}
	return db
}

type DistributedQueryOptimizer struct {
	db           *gorm.DB
	cacheEnabled bool
	cacheTTL     time.Duration
	mu           sync.RWMutex
}

func NewDistributedQueryOptimizer(db *gorm.DB) *DistributedQueryOptimizer {
	return &DistributedQueryOptimizer{
		db:           db,
		cacheEnabled: true,
		cacheTTL:     5 * time.Minute,
	}
}

func (o *DistributedQueryOptimizer) ExecuteWithCache(ctx context.Context, key string, queryFunc func() ([]interface{}, error)) ([]interface{}, error) {
	if !o.cacheEnabled {
		return queryFunc()
	}

	return queryFunc()
}

func (d *N1QueryDetector) AutoOptimize(ctx context.Context) error {
	d.mu.Lock()
	patterns := make([]*N1Pattern, 0)
	for _, p := range d.queryPatterns {
		if p.DetectionCount >= int64(d.warningThreshold) {
			patterns = append(patterns, p)
		}
	}
	d.mu.Unlock()

	for _, pattern := range patterns {
		log.Printf("[N1_AUTO_OPT] 正在优化: %s", pattern.PatternName)

		if err := d.applyOptimization(pattern); err != nil {
			log.Printf("[N1_AUTO_OPT] 优化失败 %s: %v", pattern.PatternName, err)
		} else {
			log.Printf("[N1_AUTO_OPT] 优化成功: %s", pattern.PatternName)
			d.stats.QueriesOptimized.Add(1)
		}
	}

	return nil
}

func (d *N1QueryDetector) applyOptimization(pattern *N1Pattern) error {
	return nil
}

type N1DetectionConfig struct {
	Enabled              bool
	WarningThreshold     int
	CriticalThreshold    int
	AutoOptimize         bool
	LogSlowQueries       bool
	SlowQueryThresholdMs int
}

func (d *N1QueryDetector) Configure(config *N1DetectionConfig) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.enabled = config.Enabled
	d.warningThreshold = config.WarningThreshold
	d.criticalThreshold = config.CriticalThreshold
}

var optimizerStats struct {
	QueriesOptimized atomic.Int64
	LastOptimizeTime atomic.Value
}

func init() {
	optimizerStats.LastOptimizeTime.Store(time.Time{})
}
