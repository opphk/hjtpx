package database

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type QueryOptimizerV2 struct {
	db              *gorm.DB
	poolConfig      *DynamicPoolConfig
	mu              sync.RWMutex
	autoTuning      *AutoTuningState
	batchProcessor  *BatchProcessorState
}

type DynamicPoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	MinIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	MaxConnAge      time.Duration
	PoolTimeout     time.Duration
}

type DynamicPoolMetrics struct {
	TotalConnections  int
	ActiveConnections int
	IdleConnections   int
	WaitCount         int64
	WaitDuration      time.Duration
	StaleCount        int64
	ReuseRate         float64
	AvgWaitTime       time.Duration
	MaxWaitTime       time.Duration
	ConnectionErrors  int64
	QueryCount        int64
	SlowQueryCount    int64
}

type Query struct {
	TableName       string
	Columns         []string
	Conditions      map[string]interface{}
	OrderBy         string
	SortDirection   string
	WhereClause     string
	JoinClauses     []string
	GroupBy         string
	HavingClause    string
}

type Result struct {
	ID        uint
	Data      map[string]interface{}
	CreatedAt time.Time
}

type Operation struct {
	Type       OperationType
	TableName  string
	Records    []map[string]interface{}
	Conditions map[string]interface{}
}

type OperationType string

const (
	OpTypeInsert OperationType = "insert"
	OpTypeUpdate OperationType = "update"
	OpTypeDelete OperationType = "delete"
)

type BatchResult struct {
	TotalOperations int
	SuccessCount    int
	FailureCount    int
	AffectedRows    int64
	Errors          []BatchError
	Duration        time.Duration
}

type BatchError struct {
	Index   int
	Record  map[string]interface{}
	Error   string
}

type AutoTuningState struct {
	enabled          bool
	lastTuneTime     time.Time
	tuneInterval     time.Duration
	highLoadThreshold float64
	lowLoadThreshold  float64
	targetUtilization float64
}

type BatchProcessorState struct {
	maxBatchSize    int
	concurrency     int
	retryAttempts   int
	retryDelay      time.Duration
	timeout         time.Duration
}

func NewQueryOptimizerV2(db *gorm.DB) *QueryOptimizerV2 {
	optimizer := &QueryOptimizerV2{
		db: db,
		poolConfig: &DynamicPoolConfig{
			MaxOpenConns:    100,
			MaxIdleConns:    20,
			MinIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 10 * time.Minute,
			MaxConnAge:      1 * time.Hour,
			PoolTimeout:     30 * time.Second,
		},
		autoTuning: &AutoTuningState{
			enabled:           true,
			tuneInterval:      1 * time.Minute,
			highLoadThreshold:  0.85,
			lowLoadThreshold:   0.30,
			targetUtilization:  0.70,
		},
		batchProcessor: &BatchProcessorState{
			maxBatchSize:  1000,
			concurrency:   4,
			retryAttempts: 3,
			retryDelay:    100 * time.Millisecond,
			timeout:       5 * time.Minute,
		},
	}

	go optimizer.startAutoTuning()
	return optimizer
}

func (o *QueryOptimizerV2) OptimizeComplexQuery(ctx context.Context, query string) (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	optimizedQuery, err := o.analyzeAndOptimizeQuery(ctx, query)
	if err != nil {
		return query, fmt.Errorf("query optimization failed: %w", err)
	}

	return optimizedQuery, nil
}

func (o *QueryOptimizerV2) analyzeAndOptimizeQuery(ctx context.Context, query string) (string, error) {
	var explainResults []ExplainResult

	err := o.db.WithContext(ctx).Raw("EXPLAIN (FORMAT JSON) "+query).Scan(&explainResults).Error
	if err != nil {
		return query, fmt.Errorf("failed to get query plan: %w", err)
	}

	if len(explainResults) == 0 || explainResults[0].Plan == nil {
		return query, nil
	}

	plan := explainResults[0].Plan

	if plan.TotalCost > 10000 && plan.PlanType == "Seq Scan" {
		if hasAggregation(query) {
			return o.optimizeWithWindowFunctions(query, plan)
		}
	}

	if hasSubqueryInFrom(query) {
		query, err = o.optimizeSubqueries(query)
		if err != nil {
			return query, err
		}
	}

	if hasMultipleJoins(query) {
		query, err = o.optimizeJoinOrder(query)
		if err != nil {
			return query, err
		}
	}

	if hasLikePrefix(query) {
		query, err = o.optimizeLikeQuery(query)
		if err != nil {
			return query, err
		}
	}

	query, err = o.addQueryHints(query, plan)
	if err != nil {
		return query, err
	}

	return query, nil
}

type ExplainResult struct {
	Plan *QueryPlan `json:"Plan"`
}

type QueryPlan struct {
	PlanType      string     `json:"Node Type"`
	TotalCost     float64    `json:"Total Cost"`
	PlanRows      int64      `json:"Plan Rows"`
	ActualRows    int64      `json:"Actual Rows"`
	ActualTime    []float64  `json:"Actual Execution Time"`
	ParallelAware bool       `json:"Parallel Aware"`
	Children      []QueryPlan `json:"Plans"`
}

func hasAggregation(query string) bool {
	aggregations := []string{"GROUP BY", "COUNT(", "SUM(", "AVG(", "MAX(", "MIN("}
	for _, agg := range aggregations {
		if containsIgnoreCase(query, agg) {
			return true
		}
	}
	return false
}

func hasSubqueryInFrom(query string) bool {
	return containsIgnoreCase(query, "FROM (SELECT") || containsIgnoreCase(query, "FROM (")
}

func hasMultipleJoins(query string) bool {
	joinCount := countOccurrencesIgnoreCase(query, "JOIN")
	return joinCount > 2
}

func hasLikePrefix(query string) bool {
	return containsIgnoreCase(query, "LIKE '") && containsIgnoreCase(query, "%'")
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && findSubstringIgnoreCase(s, substr)
}

func findSubstringIgnoreCase(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func countOccurrencesIgnoreCase(s, substr string) int {
	count := 0
	sLower := toLower(s)
	substrLower := toLower(substr)
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			count++
			i += len(substrLower) - 1
		}
	}
	return count
}

func (o *QueryOptimizerV2) optimizeWithWindowFunctions(query string, plan *QueryPlan) (string, error) {
	if containsIgnoreCase(query, "ROW_NUMBER()") {
		return query, nil
	}

	if containsIgnoreCase(query, "DISTINCT") && !containsIgnoreCase(query, "COUNT(DISTINCT") {
		return o.convertDistinctToWindow(query)
	}

	if containsIgnoreCase(query, "GROUP BY") && containsIgnoreCase(query, "ORDER BY") {
		return o.useRankFunctions(query)
	}

	if containsIgnoreCase(query, "MAX(") || containsIgnoreCase(query, "MIN(") {
		return o.useWindowExtremes(query)
	}

	if containsIgnoreCase(query, "COUNT(*) OVER") == false {
		return o.addRunningTotals(query)
	}

	return query, nil
}

func (o *QueryOptimizerV2) convertDistinctToWindow(query string) (string, error) {
	tableName := extractTableName(query)
	columns := extractSelectColumns(query)

	if len(columns) == 0 {
		return query, nil
	}

	firstCol := columns[0]

	return fmt.Sprintf(`
		SELECT DISTINCT ON (1) *
		FROM (
			SELECT %s,
				ROW_NUMBER() OVER (PARTITION BY %s ORDER BY created_at DESC) as rn
			FROM %s
		) sub
		WHERE rn = 1`,
		firstCol, firstCol, tableName), nil
}

func (o *QueryOptimizerV2) useRankFunctions(query string) (string, error) {
	if containsIgnoreCase(query, "DENSE_RANK()") {
		return query, nil
	}

	return o.addRanking(query)
}

func (o *QueryOptimizerV2) useWindowExtremes(query string) (string, error) {
	if containsIgnoreCase(query, "FIRST_VALUE(") || containsIgnoreCase(query, "LAST_VALUE(") {
		return query, nil
	}

	return query, nil
}

func (o *QueryOptimizerV2) addRunningTotals(query string) (string, error) {
	if containsIgnoreCase(query, "SUM(") && !containsIgnoreCase(query, "OVER") {
		col := extractAggregateColumn(query)
		if col != "" {
			return query, nil
		}
	}
	return query, nil
}

func (o *QueryOptimizerV2) addRanking(query string) (string, error) {
	if containsIgnoreCase(query, "RANK()") || containsIgnoreCase(query, "ROW_NUMBER()") {
		return query, nil
	}

	orderBy := extractOrderBy(query)
	if orderBy != "" {
		orderCol := extractOrderColumn(orderBy)
		if orderCol != "" {
			return query, nil
		}
	}

	return query, nil
}

func (o *QueryOptimizerV2) optimizeSubqueries(query string) (string, error) {
	if containsIgnoreCase(query, "LATERAL") {
		return query, nil
	}

	return o.useLateralJoin(query)
}

func (o *QueryOptimizerV2) useLateralJoin(query string) (string, error) {
	if containsIgnoreCase(query, "LIMIT") && !containsIgnoreCase(query, "LIMIT ALL") {
		return query, nil
	}
	return query, nil
}

func (o *QueryOptimizerV2) optimizeJoinOrder(query string) (string, error) {
	if containsIgnoreCase(query, "/*+") {
		return query, nil
	}

	return query, nil
}

func (o *QueryOptimizerV2) optimizeLikeQuery(query string) (string, error) {
	return o.useTrigramIndex(query)
}

func (o *QueryOptimizerV2) useTrigramIndex(query string) (string, error) {
	return query, nil
}

func (o *QueryOptimizerV2) addQueryHints(query string, plan *QueryPlan) (string, error) {
	if plan.ParallelAware {
		return query, nil
	}

	return query, nil
}

func extractTableName(query string) string {
	fromIdx := findWordIndex(query, "FROM")
	if fromIdx == -1 {
		return ""
	}

	rest := trimSpaces(query[fromIdx+4:])
	words := splitWords(rest)

	if len(words) == 0 {
		return ""
	}

	name := words[0]
	name = trimPunctuation(name, "(),;")

	return name
}

func extractSelectColumns(query string) []string {
	selectIdx := findWordIndex(query, "SELECT")
	if selectIdx == -1 {
		return nil
	}

	rest := query[selectIdx+6:]
	fromIdx := findWordIndex(rest, "FROM")
	if fromIdx == -1 {
		return nil
	}

	columnsStr := trimSpaces(rest[:fromIdx])
	return splitByComma(columnsStr)
}

func extractOrderBy(query string) string {
	orderIdx := findWordIndex(query, "ORDER BY")
	if orderIdx == -1 {
		return ""
	}

	rest := query[orderIdx+8:]
	limitIdx := findWordIndex(rest, "LIMIT")
	if limitIdx != -1 {
		rest = rest[:limitIdx]
	}

	return trimSpaces(rest)
}

func extractOrderColumn(orderBy string) string {
	words := splitWords(orderBy)
	if len(words) > 0 {
		return trimPunctuation(words[0], ",;")
	}
	return ""
}

func extractAggregateColumn(query string) string {
	patterns := []string{"COUNT(*)", "SUM(", "AVG(", "MAX(", "MIN("}
	for _, pattern := range patterns {
		idx := findWordIndex(query, pattern)
		if idx != -1 {
			start := idx + len(pattern) - 1
			end := start
			for end < len(query) && query[end] != ')' {
				end++
			}
			if end < len(query) {
				return query[start+1 : end]
			}
		}
	}
	return ""
}

func findWordIndex(s, word string) int {
	for i := 0; i <= len(s)-len(word); i++ {
		if s[i:i+len(word)] == word {
			return i
		}
	}
	return -1
}

func trimSpaces(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}

func trimPunctuation(s, punct string) string {
	result := s
	for len(result) > 0 && containsChar(punct, result[0]) {
		result = result[1:]
	}
	for len(result) > 0 && containsChar(punct, result[len(result)-1]) {
		result = result[:len(result)-1]
	}
	return result
}

func containsChar(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return true
		}
	}
	return false
}

func splitWords(s string) []string {
	var words []string
	var current []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == ',' {
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
		} else {
			current = append(current, c)
		}
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	return words
}

func splitByComma(s string) []string {
	var result []string
	var current []byte
	depth := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '(' {
			depth++
			current = append(current, c)
		} else if c == ')' {
			depth--
			current = append(current, c)
		} else if c == ',' && depth == 0 {
			if len(current) > 0 {
				result = append(result, trimSpaces(string(current)))
				current = nil
			}
		} else {
			current = append(current, c)
		}
	}
	if len(current) > 0 {
		result = append(result, trimSpaces(string(current)))
	}
	return result
}

func (o *QueryOptimizerV2) TuneConnectionPool(ctx context.Context, metrics DynamicPoolMetrics) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.autoTuning.enabled {
		return nil
	}

	if o.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	currentUtilization := float64(metrics.ActiveConnections) / float64(metrics.TotalConnections)

	if currentUtilization > o.autoTuning.highLoadThreshold {
		return o.scaleUpPool(metrics)
	}

	if currentUtilization < o.autoTuning.lowLoadThreshold {
		return o.scaleDownPool(metrics)
	}

	return o.optimizePoolParams(metrics)
}

func (o *QueryOptimizerV2) scaleUpPool(metrics DynamicPoolMetrics) error {
	if o.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	newMaxOpen := int(float64(o.poolConfig.MaxOpenConns) * 1.2)
	if newMaxOpen > 500 {
		newMaxOpen = 500
	}

	newMaxIdle := int(float64(o.poolConfig.MaxIdleConns) * 1.3)
	if newMaxIdle > newMaxOpen {
		newMaxIdle = newMaxOpen / 2
	}

	if newMaxIdle < o.poolConfig.MinIdleConns {
		newMaxIdle = o.poolConfig.MinIdleConns
	}

	o.poolConfig.MaxOpenConns = newMaxOpen
	o.poolConfig.MaxIdleConns = newMaxIdle

	return o.applyPoolConfig()
}

func (o *QueryOptimizerV2) scaleDownPool(metrics DynamicPoolMetrics) error {
	if o.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	newMaxOpen := int(float64(o.poolConfig.MaxOpenConns) * 0.8)
	if newMaxOpen < 10 {
		newMaxOpen = 10
	}

	newMaxIdle := int(float64(o.poolConfig.MaxIdleConns) * 0.8)
	if newMaxIdle < o.poolConfig.MinIdleConns {
		newMaxIdle = o.poolConfig.MinIdleConns
	}

	o.poolConfig.MaxOpenConns = newMaxOpen
	o.poolConfig.MaxIdleConns = newMaxIdle

	return o.applyPoolConfig()
}

func (o *QueryOptimizerV2) optimizePoolParams(metrics DynamicPoolMetrics) error {
	if o.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	if metrics.AvgWaitTime > 1*time.Second && metrics.MaxWaitTime > 5*time.Second {
		o.poolConfig.PoolTimeout = o.poolConfig.PoolTimeout + 10*time.Second
		if o.poolConfig.PoolTimeout > 2*time.Minute {
			o.poolConfig.PoolTimeout = 2 * time.Minute
		}
	}

	if metrics.ReuseRate < 0.5 {
		o.poolConfig.ConnMaxIdleTime = o.poolConfig.ConnMaxIdleTime - 1*time.Minute
		if o.poolConfig.ConnMaxIdleTime < 1*time.Minute {
			o.poolConfig.ConnMaxIdleTime = 1 * time.Minute
		}
	}

	return o.applyPoolConfig()
}

func (o *QueryOptimizerV2) applyPoolConfig() error {
	sqlDB, err := o.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(o.poolConfig.MaxOpenConns)
	sqlDB.SetMaxIdleConns(o.poolConfig.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(o.poolConfig.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(o.poolConfig.ConnMaxIdleTime)

	log.Printf("[POOL_V2] Pool config updated: MaxOpen=%d, MaxIdle=%d, MaxIdleTime=%v",
		o.poolConfig.MaxOpenConns, o.poolConfig.MaxIdleConns, o.poolConfig.ConnMaxIdleTime)

	return nil
}

func (o *QueryOptimizerV2) startAutoTuning() {
	ticker := time.NewTicker(o.autoTuning.tuneInterval)
	defer ticker.Stop()

	for range ticker.C {
		o.performAutoTuning()
	}
}

func (o *QueryOptimizerV2) performAutoTuning() {
	if !o.autoTuning.enabled {
		return
	}

	metrics := o.collectDynamicPoolMetrics()

	if err := o.TuneConnectionPool(context.Background(), metrics); err != nil {
		log.Printf("[POOL_V2] Auto tuning failed: %v", err)
	}

	o.autoTuning.lastTuneTime = time.Now()
}

func (o *QueryOptimizerV2) collectDynamicPoolMetrics() DynamicPoolMetrics {
	sqlDB, err := o.db.DB()
	if err != nil {
		return DynamicPoolMetrics{}
	}

	stats := sqlDB.Stats()

	total := stats.MaxOpenConnections
	active := stats.InUse
	idle := stats.Idle

	var reuseRate float64
	if total > 0 {
		reuseRate = float64(active) / float64(total)
	}

	var avgWait, maxWait time.Duration
	if stats.WaitCount > 0 {
		avgWait = stats.WaitDuration / time.Duration(stats.WaitCount)
		maxWait = avgWait * 3
	}

	return DynamicPoolMetrics{
		TotalConnections:  total,
		ActiveConnections: active,
		IdleConnections:   idle,
		WaitCount:         stats.WaitCount,
		WaitDuration:      stats.WaitDuration,
		ReuseRate:         reuseRate,
		AvgWaitTime:       avgWait,
		MaxWaitTime:       maxWait,
	}
}

func (o *QueryOptimizerV2) CursorPagination(query *Query, cursor string, limit int) ([]Result, string, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 1000 {
		limit = 1000
	}

	var lastCursor CursorData
	if cursor != "" {
		decoded, err := base64.StdEncoding.DecodeString(cursor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor format: %w", err)
		}

		if err := json.Unmarshal(decoded, &lastCursor); err != nil {
			return nil, "", fmt.Errorf("failed to decode cursor: %w", err)
		}
	}

	results, err := o.executeCursorQuery(query, lastCursor, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("cursor query execution failed: %w", err)
	}

	var nextCursor string
	if len(results) > limit {
		results = results[:limit]
		lastResult := results[len(results)-1]
		cursorData := CursorData{
			ID:        lastResult.ID,
			CreatedAt: lastResult.CreatedAt,
		}
		encoded, _ := json.Marshal(cursorData)
		nextCursor = base64.StdEncoding.EncodeToString(encoded)
	}

	return results, nextCursor, nil
}

type CursorData struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func (o *QueryOptimizerV2) executeCursorQuery(query *Query, cursor CursorData, limit int) ([]Result, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	var results []Result

	if cursor.ID != 0 && cursor.CreatedAt.IsZero() == false {
		cursor.CreatedAt = cursor.CreatedAt.Add(time.Nanosecond)
	}

	sqlQuery := o.buildCursorQuery(query, cursor)

	var rows *sql.Rows
	var err error

	dsn := o.db.Dialector.(*postgres.Dialector).DSN
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err = db.Query(sqlQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rowData := make(map[string]interface{})
		for i, col := range columns {
			rowData[col] = values[i]
		}

		var id uint
		var createdAt time.Time
		if idVal, ok := rowData["id"]; ok {
			switch v := idVal.(type) {
			case int64:
				id = uint(v)
			case int32:
				id = uint(v)
			case uint:
				id = v
			}
		}
		if createdAtVal, ok := rowData["created_at"]; ok {
			switch v := createdAtVal.(type) {
			case time.Time:
				createdAt = v
			}
		}

		results = append(results, Result{
			ID:        id,
			Data:      rowData,
			CreatedAt: createdAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (o *QueryOptimizerV2) buildCursorQuery(query *Query, cursor CursorData) string {
	orderCol := query.OrderBy
	if orderCol == "" {
		orderCol = "created_at"
	}
	sortDir := query.SortDirection
	if sortDir == "" {
		sortDir = "DESC"
	}

	columns := "*"
	if len(query.Columns) > 0 {
		columns = joinStrings(query.Columns, ", ")
	}

	var whereClause string
	if cursor.ID != 0 {
		op := ">"
		if sortDir == "DESC" {
			op = "<"
		}
		whereClause = fmt.Sprintf("WHERE (%s %s $1 OR (%s = $1 AND id %s $2))",
			orderCol, op, orderCol, op)
	} else if len(query.WhereClause) > 0 {
		whereClause = "WHERE " + query.WhereClause
	}

	var joinClause string
	if len(query.JoinClauses) > 0 {
		joinClause = " " + joinStrings(query.JoinClauses, " ")
	}

	var groupByClause string
	if len(query.GroupBy) > 0 {
		groupByClause = " GROUP BY " + query.GroupBy
	}

	var havingClause string
	if len(query.HavingClause) > 0 {
		havingClause = " HAVING " + query.HavingClause
	}

	return fmt.Sprintf("SELECT %s FROM %s%s %s%s%s ORDER BY %s %s, id %s LIMIT $3",
		columns, query.TableName, joinClause, whereClause, groupByClause, havingClause,
		orderCol, sortDir, sortDir)
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

func (o *QueryOptimizerV2) BatchOperations(ctx context.Context, operations []Operation) (*BatchResult, error) {
	startTime := time.Now()

	result := &BatchResult{
		TotalOperations: len(operations),
		Errors:          make([]BatchError, 0),
	}

	if len(operations) == 0 {
		result.Duration = time.Since(startTime)
		return result, nil
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	concurrency := 4
	if o.batchProcessor != nil {
		concurrency = o.batchProcessor.concurrency
	}
	semaphore := make(chan struct{}, concurrency)

	for i, op := range operations {
		wg.Add(1)
		go func(idx int, operation Operation) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			opResult, err := o.executeOperation(ctx, operation)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				result.FailureCount++
				var recordInfo interface{}
				if len(operation.Records) > 0 {
					recordInfo = fmt.Sprintf("batch[%d], records: %d", idx, len(operation.Records))
				} else {
					recordInfo = fmt.Sprintf("batch[%d]", idx)
				}
				result.Errors = append(result.Errors, BatchError{
					Index:  idx,
					Record: map[string]interface{}{"info": recordInfo},
					Error:  err.Error(),
				})
			} else {
				result.SuccessCount++
				result.AffectedRows += opResult
			}
		}(i, op)
	}

	wg.Wait()

	result.Duration = time.Since(startTime)

	if result.FailureCount > 0 && result.SuccessCount > 0 {
		log.Printf("[BATCH_V2] Completed: %d success, %d failures out of %d total",
			result.SuccessCount, result.FailureCount, result.TotalOperations)
	}

	return result, nil
}

func (o *QueryOptimizerV2) executeOperation(ctx context.Context, operation Operation) (int64, error) {
	var totalAffected int64

	switch operation.Type {
	case OpTypeInsert:
		affected, err := o.executeBatchInsert(ctx, operation)
		if err != nil {
			return 0, err
		}
		totalAffected += affected

	case OpTypeUpdate:
		affected, err := o.executeBatchUpdate(ctx, operation)
		if err != nil {
			return 0, err
		}
		totalAffected += affected

	case OpTypeDelete:
		affected, err := o.executeBatchDelete(ctx, operation)
		if err != nil {
			return 0, err
		}
		totalAffected += affected

	default:
		return 0, fmt.Errorf("unknown operation type: %s", operation.Type)
	}

	return totalAffected, nil
}

func (o *QueryOptimizerV2) executeBatchInsert(ctx context.Context, operation Operation) (int64, error) {
	if len(operation.Records) == 0 {
		return 0, nil
	}

	tableName := operation.TableName
	if tableName == "" {
		return 0, fmt.Errorf("table name is required for insert operation")
	}

	columns := make([]string, 0)
	columnSet := make(map[string]bool)
	for _, record := range operation.Records {
		for col := range record {
			if !columnSet[col] {
				columns = append(columns, col)
				columnSet[col] = true
			}
		}
	}

	if len(columns) == 0 {
		return 0, fmt.Errorf("no columns specified for insert")
	}

	var totalAffected int64

	for i := 0; i < len(operation.Records); i += o.batchProcessor.maxBatchSize {
		end := i + o.batchProcessor.maxBatchSize
		if end > len(operation.Records) {
			end = len(operation.Records)
		}
		batch := operation.Records[i:end]

		affected, err := o.insertBatchChunk(ctx, tableName, columns, batch)
		if err != nil {
			return totalAffected, err
		}
		totalAffected += affected
	}

	return totalAffected, nil
}

func (o *QueryOptimizerV2) insertBatchChunk(ctx context.Context, tableName string, columns []string, records []map[string]interface{}) (int64, error) {
	if len(records) == 0 {
		return 0, nil
	}

	valueStrings := make([]string, 0, len(records))
	valueArgs := make([]interface{}, 0, len(records)*len(columns))

	for _, record := range records {
		valueParts := make([]string, len(columns))
		for i, col := range columns {
			valueParts[i] = fmt.Sprintf("$%d", len(valueArgs)+i+1)
			valueArgs = append(valueArgs, record[col])
		}
		valueStrings = append(valueStrings, "("+joinStrings(valueParts, ", ")+")")
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		tableName, joinStrings(columns, ", "), joinStrings(valueStrings, ", "))

	result := o.db.WithContext(ctx).Exec(query, valueArgs...)
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

func (o *QueryOptimizerV2) executeBatchUpdate(ctx context.Context, operation Operation) (int64, error) {
	if len(operation.Records) == 0 {
		return 0, nil
	}

	tableName := operation.TableName
	if tableName == "" {
		return 0, fmt.Errorf("table name is required for update operation")
	}

	var totalAffected int64

	for i := 0; i < len(operation.Records); i += o.batchProcessor.maxBatchSize {
		end := i + o.batchProcessor.maxBatchSize
		if end > len(operation.Records) {
			end = len(operation.Records)
		}
		batch := operation.Records[i:end]

		affected, err := o.updateBatchChunk(ctx, tableName, batch, operation.Conditions)
		if err != nil {
			return totalAffected, err
		}
		totalAffected += affected
	}

	return totalAffected, nil
}

func (o *QueryOptimizerV2) updateBatchChunk(ctx context.Context, tableName string, records []map[string]interface{}, conditions map[string]interface{}) (int64, error) {
	if len(records) == 0 {
		return 0, nil
	}

	tx := o.db.WithContext(ctx).Table(tableName)

	for key, value := range conditions {
		tx = tx.Where(key+" = ?", value)
	}

	if len(records) == 1 {
		result := tx.Updates(records[0])
		return result.RowsAffected, result.Error
	}

	column := ""
	ids := make([]interface{}, 0)

	for _, record := range records {
		for k, v := range record {
			if k == "id" || k == "ID" || k == "Id" {
				column = k
				ids = append(ids, v)
			}
		}
	}

	if column == "" || len(ids) == 0 {
		return 0, fmt.Errorf("no ID column found in records")
	}

	setClause := make([]string, 0)

	for col := range records[0] {
		if col != column && col != "ID" && col != "id" && col != "Id" {
			setClause = append(setClause, col+" = ?")
		}
	}

	if len(setClause) == 0 {
		return 0, fmt.Errorf("no columns to update")
	}

	assignments := make(map[string]interface{})
	for _, col := range setClause {
		colName := col[:len(col)-4]
		var val interface{}
		for _, record := range records {
			if v, ok := record[colName]; ok {
				val = v
				break
			}
		}
		assignments[colName] = val
	}

	result := tx.Where(column+" IN ?", ids).Updates(assignments)

	return result.RowsAffected, result.Error
}

func (o *QueryOptimizerV2) executeBatchDelete(ctx context.Context, operation Operation) (int64, error) {
	if len(operation.Conditions) == 0 {
		return 0, fmt.Errorf("conditions required for delete operation to prevent accidental data loss")
	}

	tableName := operation.TableName
	if tableName == "" {
		return 0, fmt.Errorf("table name is required for delete operation")
	}

	tx := o.db.WithContext(ctx).Table(tableName)

	for key, value := range operation.Conditions {
		tx = tx.Where(key+" = ?", value)
	}

	result := tx.Delete(&struct{}{})
	return result.RowsAffected, result.Error
}

func (o *QueryOptimizerV2) GetDynamicPoolMetrics(ctx context.Context) (*DynamicPoolMetrics, error) {
	metrics := o.collectDynamicPoolMetrics()
	return &metrics, nil
}

func (o *QueryOptimizerV2) GetPoolConfig() *DynamicPoolConfig {
	o.mu.RLock()
	defer o.mu.RUnlock()

	configCopy := *o.poolConfig
	return &configCopy
}

func (o *QueryOptimizerV2) SetPoolConfig(config *DynamicPoolConfig) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if config.MaxOpenConns <= 0 {
		return fmt.Errorf("max_open_conns must be positive")
	}
	if config.MaxIdleConns <= 0 {
		return fmt.Errorf("max_idle_conns must be positive")
	}
	if config.MaxIdleConns > config.MaxOpenConns {
		return fmt.Errorf("max_idle_conns cannot exceed max_open_conns")
	}
	if config.MinIdleConns < 0 {
		return fmt.Errorf("min_idle_conns cannot be negative")
	}

	o.poolConfig = config

	if o.db == nil {
		return nil
	}

	return o.applyPoolConfig()
}

func (o *QueryOptimizerV2) EnableAutoTuning(enabled bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.autoTuning.enabled = enabled
}

func (o *QueryOptimizerV2) IsAutoTuningEnabled() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.autoTuning.enabled
}
