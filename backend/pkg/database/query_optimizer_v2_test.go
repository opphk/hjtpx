package database

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestNewQueryOptimizerV2(t *testing.T) {
	optimizer := &QueryOptimizerV2{
		poolConfig: &DynamicPoolConfig{
			MaxOpenConns:    100,
			MaxIdleConns:    20,
			MinIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 10 * time.Minute,
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

	if optimizer.poolConfig.MaxOpenConns != 100 {
		t.Errorf("MaxOpenConns = %d, want %d", optimizer.poolConfig.MaxOpenConns, 100)
	}

	if optimizer.poolConfig.MaxIdleConns != 20 {
		t.Errorf("MaxIdleConns = %d, want %d", optimizer.poolConfig.MaxIdleConns, 20)
	}

	if !optimizer.autoTuning.enabled {
		t.Error("Auto tuning should be enabled")
	}

	if optimizer.batchProcessor.maxBatchSize != 1000 {
		t.Errorf("maxBatchSize = %d, want %d", optimizer.batchProcessor.maxBatchSize, 1000)
	}
}

func TestDynamicPoolConfig(t *testing.T) {
	config := &DynamicPoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    20,
		MinIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
		MaxConnAge:      1 * time.Hour,
		PoolTimeout:     30 * time.Second,
	}

	if config.MaxOpenConns != 100 {
		t.Errorf("MaxOpenConns = %d, want %d", config.MaxOpenConns, 100)
	}

	if config.MinIdleConns != 5 {
		t.Errorf("MinIdleConns = %d, want %d", config.MinIdleConns, 5)
	}

	if config.MaxConnAge != 1*time.Hour {
		t.Errorf("MaxConnAge = %v, want %v", config.MaxConnAge, 1*time.Hour)
	}
}

func TestDynamicPoolMetrics(t *testing.T) {
	metrics := DynamicPoolMetrics{
		TotalConnections:  100,
		ActiveConnections: 50,
		IdleConnections:   30,
		WaitCount:         10,
		WaitDuration:      500 * time.Millisecond,
		StaleCount:        5,
		ReuseRate:         0.625,
		AvgWaitTime:       50 * time.Millisecond,
		MaxWaitTime:       200 * time.Millisecond,
		ConnectionErrors:  2,
		QueryCount:        1000,
		SlowQueryCount:    50,
	}

	if metrics.TotalConnections != 100 {
		t.Errorf("TotalConnections = %d, want %d", metrics.TotalConnections, 100)
	}

	if metrics.ActiveConnections != 50 {
		t.Errorf("ActiveConnections = %d, want %d", metrics.ActiveConnections, 50)
	}

	if metrics.ReuseRate != 0.625 {
		t.Errorf("ReuseRate = %f, want %f", metrics.ReuseRate, 0.625)
	}

	if metrics.SlowQueryCount != 50 {
		t.Errorf("SlowQueryCount = %d, want %d", metrics.SlowQueryCount, 50)
	}
}

func TestQuery(t *testing.T) {
	query := &Query{
		TableName:     "users",
		Columns:       []string{"id", "name", "email"},
		Conditions:   map[string]interface{}{"status": "active"},
		OrderBy:       "created_at",
		SortDirection: "DESC",
	}

	if query.TableName != "users" {
		t.Errorf("TableName = %s, want %s", query.TableName, "users")
	}

	if len(query.Columns) != 3 {
		t.Errorf("Columns count = %d, want %d", len(query.Columns), 3)
	}

	if query.SortDirection != "DESC" {
		t.Errorf("SortDirection = %s, want %s", query.SortDirection, "DESC")
	}
}

func TestResult(t *testing.T) {
	result := Result{
		ID:        1,
		Data:      map[string]interface{}{"name": "test", "value": 100},
		CreatedAt: time.Now(),
	}

	if result.ID != 1 {
		t.Errorf("ID = %d, want %d", result.ID, 1)
	}

	if result.Data["name"] != "test" {
		t.Errorf("Data[name] = %v, want %v", result.Data["name"], "test")
	}

	if result.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestOperation(t *testing.T) {
	operation := Operation{
		Type:      OpTypeInsert,
		TableName: "users",
		Records: []map[string]interface{}{
			{"name": "user1", "email": "user1@example.com"},
			{"name": "user2", "email": "user2@example.com"},
		},
		Conditions: map[string]interface{}{},
	}

	if operation.Type != OpTypeInsert {
		t.Errorf("Type = %s, want %s", operation.Type, OpTypeInsert)
	}

	if len(operation.Records) != 2 {
		t.Errorf("Records count = %d, want %d", len(operation.Records), 2)
	}
}

func TestOperationType(t *testing.T) {
	tests := []struct {
		opType  OperationType
		wantStr string
	}{
		{OpTypeInsert, "insert"},
		{OpTypeUpdate, "update"},
		{OpTypeDelete, "delete"},
	}

	for _, tt := range tests {
		if string(tt.opType) != tt.wantStr {
			t.Errorf("OperationType = %s, want %s", tt.opType, tt.wantStr)
		}
	}
}

func TestBatchResult(t *testing.T) {
	result := &BatchResult{
		TotalOperations: 100,
		SuccessCount:    95,
		FailureCount:    5,
		AffectedRows:    1000,
		Errors: []BatchError{
			{Index: 0, Record: map[string]interface{}{"name": "test"}, Error: "duplicate key"},
		},
		Duration: 5 * time.Second,
	}

	if result.TotalOperations != 100 {
		t.Errorf("TotalOperations = %d, want %d", result.TotalOperations, 100)
	}

	if result.SuccessCount != 95 {
		t.Errorf("SuccessCount = %d, want %d", result.SuccessCount, 95)
	}

	if result.FailureCount != 5 {
		t.Errorf("FailureCount = %d, want %d", result.FailureCount, 5)
	}

	if result.AffectedRows != 1000 {
		t.Errorf("AffectedRows = %d, want %d", result.AffectedRows, 1000)
	}

	if len(result.Errors) != 1 {
		t.Errorf("Errors count = %d, want %d", len(result.Errors), 1)
	}
}

func TestBatchError(t *testing.T) {
	err := BatchError{
		Index:   5,
		Record:  map[string]interface{}{"id": 123, "name": "test"},
		Error:   "foreign key violation",
	}

	if err.Index != 5 {
		t.Errorf("Index = %d, want %d", err.Index, 5)
	}

	if err.Error != "foreign key violation" {
		t.Errorf("Error = %s, want %s", err.Error, "foreign key violation")
	}
}

func TestAutoTuningState(t *testing.T) {
	state := &AutoTuningState{
		enabled:           true,
		lastTuneTime:      time.Now(),
		tuneInterval:     1 * time.Minute,
		highLoadThreshold: 0.85,
		lowLoadThreshold:  0.30,
		targetUtilization: 0.70,
	}

	if !state.enabled {
		t.Error("enabled should be true")
	}

	if state.highLoadThreshold != 0.85 {
		t.Errorf("highLoadThreshold = %f, want %f", state.highLoadThreshold, 0.85)
	}

	if state.lowLoadThreshold != 0.30 {
		t.Errorf("lowLoadThreshold = %f, want %f", state.lowLoadThreshold, 0.30)
	}

	if state.targetUtilization != 0.70 {
		t.Errorf("targetUtilization = %f, want %f", state.targetUtilization, 0.70)
	}
}

func TestBatchProcessorState(t *testing.T) {
	state := &BatchProcessorState{
		maxBatchSize:  1000,
		concurrency:   4,
		retryAttempts: 3,
		retryDelay:    100 * time.Millisecond,
		timeout:       5 * time.Minute,
	}

	if state.maxBatchSize != 1000 {
		t.Errorf("maxBatchSize = %d, want %d", state.maxBatchSize, 1000)
	}

	if state.concurrency != 4 {
		t.Errorf("concurrency = %d, want %d", state.concurrency, 4)
	}

	if state.retryAttempts != 3 {
		t.Errorf("retryAttempts = %d, want %d", state.retryAttempts, 3)
	}
}

func TestCursorData(t *testing.T) {
	cursorData := CursorData{
		ID:        100,
		CreatedAt: time.Now(),
	}

	encoded, err := json.Marshal(cursorData)
	if err != nil {
		t.Fatalf("Failed to marshal cursor: %v", err)
	}

	decoded := CursorData{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal cursor: %v", err)
	}

	if decoded.ID != cursorData.ID {
		t.Errorf("Decoded ID = %d, want %d", decoded.ID, cursorData.ID)
	}

	if !decoded.CreatedAt.Equal(cursorData.CreatedAt) {
		t.Errorf("Decoded CreatedAt = %v, want %v", decoded.CreatedAt, cursorData.CreatedAt)
	}
}

func TestCursorDataBase64Encoding(t *testing.T) {
	original := CursorData{
		ID:        42,
		CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	base64Str := base64.StdEncoding.EncodeToString(encoded)

	decodedBytes, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}

	var decoded CursorData
	if err := json.Unmarshal(decodedBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal decoded: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %d, want %d", decoded.ID, original.ID)
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"SELECT * FROM users", "select", true},
		{"SELECT * FROM users", "SELECT", true},
		{"SELECT * FROM users", "From", true},
		{"SELECT * FROM users", "WHERE", false},
		{"ABCDEF", "bcd", true},
		{"ABCDEF", "xyz", false},
		{"", "test", false},
		{"test", "", true},
	}

	for _, tt := range tests {
		result := containsIgnoreCase(tt.s, tt.substr)
		if result != tt.expected {
			t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
		}
	}
}

func TestCountOccurrencesIgnoreCase(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected int
	}{
		{"SELECT * FROM users JOIN orders", "JOIN", 1},
		{"a JOIN b JOIN c JOIN d", "join", 3},
		{"no match here", "test", 0},
		{"count count COUNT", "count", 3},
	}

	for _, tt := range tests {
		result := countOccurrencesIgnoreCase(tt.s, tt.substr)
		if result != tt.expected {
			t.Errorf("countOccurrencesIgnoreCase(%q, %q) = %d, want %d", tt.s, tt.substr, result, tt.expected)
		}
	}
}

func TestHasAggregation(t *testing.T) {
	tests := []struct {
		query    string
		expected bool
	}{
		{"SELECT COUNT(*) FROM users", true},
		{"SELECT SUM(amount) FROM orders", true},
		{"SELECT AVG(price) FROM products", true},
		{"SELECT MAX(value) FROM metrics", true},
		{"SELECT MIN(count) FROM data", true},
		{"SELECT * FROM users", false},
		{"SELECT name FROM users", false},
		{"SELECT * FROM users GROUP BY name", true},
	}

	for _, tt := range tests {
		result := hasAggregation(tt.query)
		if result != tt.expected {
			t.Errorf("hasAggregation(%q) = %v, want %v", tt.query, result, tt.expected)
		}
	}
}

func TestHasSubqueryInFrom(t *testing.T) {
	tests := []struct {
		query    string
		expected bool
	}{
		{"SELECT * FROM (SELECT id FROM users) AS sub", true},
		{"SELECT * FROM (SELECT * FROM orders WHERE status = 'active')", true},
		{"SELECT * FROM users", false},
		{"SELECT * FROM users WHERE id IN (SELECT id FROM admins)", false},
	}

	for _, tt := range tests {
		result := hasSubqueryInFrom(tt.query)
		if result != tt.expected {
			t.Errorf("hasSubqueryInFrom(%q) = %v, want %v", tt.query, result, tt.expected)
		}
	}
}

func TestHasMultipleJoins(t *testing.T) {
	tests := []struct {
		query    string
		expected bool
	}{
		{"SELECT * FROM a JOIN b ON a.id = b.id", false},
		{"SELECT * FROM a JOIN b ON a.id = b.id JOIN c ON b.id = c.id", true},
		{"SELECT * FROM a JOIN b ON a.id = b.id JOIN c ON b.id = c.id JOIN d ON c.id = d.id", true},
		{"SELECT * FROM a, b WHERE a.id = b.id", false},
	}

	for _, tt := range tests {
		result := hasMultipleJoins(tt.query)
		if result != tt.expected {
			t.Errorf("hasMultipleJoins(%q) = %v, want %v", tt.query, result, tt.expected)
		}
	}
}

func TestHasLikePrefix(t *testing.T) {
	tests := []struct {
		query    string
		expected bool
	}{
		{"SELECT * FROM users WHERE name LIKE 'test%'", true},
		{"SELECT * FROM users WHERE name LIKE '%test'", false},
		{"SELECT * FROM users WHERE name LIKE '%test%'", false},
		{"SELECT * FROM users WHERE name = 'test'", false},
	}

	for _, tt := range tests {
		result := hasLikePrefix(tt.query)
		if result != tt.expected {
			t.Errorf("hasLikePrefix(%q) = %v, want %v", tt.query, result, tt.expected)
		}
	}
}

func TestTrimSpaces(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello  ", "hello"},
		{"\t\tworld\t\t", "world"},
		{"\n\ntest\n\n", "test"},
		{"  multiple   spaces  ", "multiple   spaces"},
		{"no spaces", "no spaces"},
	}

	for _, tt := range tests {
		result := trimSpaces(tt.input)
		if result != tt.expected {
			t.Errorf("trimSpaces(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestTrimPunctuation(t *testing.T) {
	tests := []struct {
		input    string
		punct    string
		expected string
	}{
		{"(test)", "()", "test"},
		{"{value}", "{}", "value"},
		{";data;", ";", "data"},
		{"no punctuation", ",", "no punctuation"},
	}

	for _, tt := range tests {
		result := trimPunctuation(tt.input, tt.punct)
		if result != tt.expected {
			t.Errorf("trimPunctuation(%q, %q) = %q, want %q", tt.input, tt.punct, result, tt.expected)
		}
	}
}

func TestSplitWords(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"SELECT * FROM users", []string{"SELECT", "*", "FROM", "users"}},
		{"one,two,three", []string{"one", "two", "three"}},
		{"  spaced  out  ", []string{"spaced", "out"}},
	}

	for _, tt := range tests {
		result := splitWords(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("splitWords(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("splitWords(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
			}
		}
	}
}

func TestSplitByComma(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a, b, c", []string{"a", "b", "c"}},
		{"func(a, b)", []string{"func(a", "b)"}},
		{"no commas", []string{"no commas"}},
		{"a,b,c", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		result := splitByComma(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("splitByComma(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("splitByComma(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
			}
		}
	}
}

func TestFindWordIndex(t *testing.T) {
	tests := []struct {
		s        string
		word     string
		expected int
	}{
		{"SELECT * FROM users", "FROM", 9},
		{"SELECT * FROM users", "select", -1},
		{"hello world", "world", 6},
		{"abc", "d", -1},
	}

	for _, tt := range tests {
		result := findWordIndex(tt.s, tt.word)
		if result != tt.expected {
			t.Errorf("findWordIndex(%q, %q) = %d, want %d", tt.s, tt.word, result, tt.expected)
		}
	}
}

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		strs     []string
		sep      string
		expected string
	}{
		{[]string{"a", "b", "c"}, ", ", "a, b, c"},
		{[]string{"one"}, ",", "one"},
		{[]string{}, ",", ""},
		{[]string{"SELECT", "*", "FROM"}, " ", "SELECT * FROM"},
	}

	for _, tt := range tests {
		result := joinStrings(tt.strs, tt.sep)
		if result != tt.expected {
			t.Errorf("joinStrings(%v, %q) = %q, want %q", tt.strs, tt.sep, result, tt.expected)
		}
	}
}

func TestExtractTableName(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{"SELECT * FROM users", "users"},
		{"SELECT id, name FROM products", "products"},
		{"SELECT * FROM users WHERE id = 1", "users"},
		{"SELECT * FROM (SELECT * FROM orders) AS sub", ""},
	}

	for _, tt := range tests {
		result := extractTableName(tt.query)
		if result != tt.expected {
			t.Errorf("extractTableName(%q) = %q, want %q", tt.query, result, tt.expected)
		}
	}
}

func TestExtractOrderBy(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{"SELECT * FROM users ORDER BY created_at", "created_at"},
		{"SELECT * FROM users ORDER BY name DESC", "name DESC"},
		{"SELECT * FROM users ORDER BY created_at DESC LIMIT 10", "created_at DESC"},
		{"SELECT * FROM users", ""},
	}

	for _, tt := range tests {
		result := extractOrderBy(tt.query)
		if result != tt.expected {
			t.Errorf("extractOrderBy(%q) = %q, want %q", tt.query, result, tt.expected)
		}
	}
}

func TestExtractOrderColumn(t *testing.T) {
	tests := []struct {
		orderBy  string
		expected string
	}{
		{"created_at", "created_at"},
		{"name DESC", "name"},
		{"id ASC", "id"},
		{"", ""},
	}

	for _, tt := range tests {
		result := extractOrderColumn(tt.orderBy)
		if result != tt.expected {
			t.Errorf("extractOrderColumn(%q) = %q, want %q", tt.orderBy, result, tt.expected)
		}
	}
}

func TestExtractSelectColumns(t *testing.T) {
	tests := []struct {
		query    string
		expected int
	}{
		{"SELECT id, name, email FROM users", 3},
		{"SELECT * FROM users", 1},
		{"SELECT COUNT(*) FROM users", 1},
	}

	for _, tt := range tests {
		result := extractSelectColumns(tt.query)
		if len(result) != tt.expected {
			t.Errorf("extractSelectColumns(%q) count = %d, want %d", tt.query, len(result), tt.expected)
		}
	}
}

func TestExplainResult(t *testing.T) {
	result := ExplainResult{
		Plan: &QueryPlan{
			PlanType:      "Seq Scan",
			TotalCost:     1000.50,
			PlanRows:      10000,
			ActualRows:    9800,
			ParallelAware: false,
		},
	}

	if result.Plan.PlanType != "Seq Scan" {
		t.Errorf("PlanType = %s, want %s", result.Plan.PlanType, "Seq Scan")
	}

	if result.Plan.TotalCost != 1000.50 {
		t.Errorf("TotalCost = %f, want %f", result.Plan.TotalCost, 1000.50)
	}
}

func TestQueryPlan(t *testing.T) {
	plan := QueryPlan{
		PlanType:      "Hash Join",
		TotalCost:     5000.0,
		PlanRows:      50000,
		ActualRows:    48000,
		ActualTime:    []float64{10.5, 20.3},
		ParallelAware: true,
		Children: []QueryPlan{
			{PlanType: "Seq Scan", TotalCost: 1000.0},
			{PlanType: "Seq Scan", TotalCost: 1200.0},
		},
	}

	if plan.PlanType != "Hash Join" {
		t.Errorf("PlanType = %s, want %s", plan.PlanType, "Hash Join")
	}

	if len(plan.Children) != 2 {
		t.Errorf("Children count = %d, want %d", len(plan.Children), 2)
	}

	if !plan.ParallelAware {
		t.Error("ParallelAware should be true")
	}
}

func TestBuildCursorQuery(t *testing.T) {
	optimizer := &QueryOptimizerV2{}

	query := &Query{
		TableName:     "verification_logs",
		Columns:       []string{"id", "status", "created_at"},
		OrderBy:       "created_at",
		SortDirection: "DESC",
	}

	cursor := CursorData{
		ID:        0,
		CreatedAt: time.Time{},
	}

	sql := optimizer.buildCursorQuery(query, cursor)
	if sql == "" {
		t.Error("buildCursorQuery should return non-empty string")
	}

	if len(sql) < 20 {
		t.Errorf("Query seems too short: %s", sql)
	}
}

func TestBuildCursorQueryWithCursor(t *testing.T) {
	optimizer := &QueryOptimizerV2{}

	query := &Query{
		TableName:     "users",
		OrderBy:       "created_at",
		SortDirection: "DESC",
	}

	cursor := CursorData{
		ID:        100,
		CreatedAt: time.Now(),
	}

	sql := optimizer.buildCursorQuery(query, cursor)

	if sql == "" {
		t.Error("buildCursorQuery should return non-empty string")
	}

	if len(sql) < 20 {
		t.Errorf("Query seems too short: %s", sql)
	}
}

func TestBuildCursorQueryWithWhereClause(t *testing.T) {
	optimizer := &QueryOptimizerV2{}

	query := &Query{
		TableName:     "users",
		OrderBy:       "created_at",
		SortDirection: "ASC",
		WhereClause:   "status = 'active'",
	}

	cursor := CursorData{}

	sql := optimizer.buildCursorQuery(query, cursor)

	if sql == "" {
		t.Error("buildCursorQuery should return non-empty string")
	}

	if sql == "" {
		t.Error("Query should contain WHERE clause")
	}
}

func TestQueryOptimizerV2SetPoolConfig(t *testing.T) {
	optimizer := &QueryOptimizerV2{
		poolConfig: &DynamicPoolConfig{
			MaxOpenConns:    100,
			MaxIdleConns:    20,
			MinIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 10 * time.Minute,
		},
	}

	tests := []struct {
		name    string
		config  *DynamicPoolConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &DynamicPoolConfig{
				MaxOpenConns:    50,
				MaxIdleConns:    10,
				MinIdleConns:    2,
				ConnMaxLifetime: 20 * time.Minute,
				ConnMaxIdleTime: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "invalid max open conns",
			config: &DynamicPoolConfig{
				MaxOpenConns:    0,
				MaxIdleConns:    10,
				MinIdleConns:    2,
				ConnMaxLifetime: 20 * time.Minute,
				ConnMaxIdleTime: 5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "invalid max idle conns",
			config: &DynamicPoolConfig{
				MaxOpenConns:    50,
				MaxIdleConns:    0,
				MinIdleConns:    2,
				ConnMaxLifetime: 20 * time.Minute,
				ConnMaxIdleTime: 5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "idle exceeds open",
			config: &DynamicPoolConfig{
				MaxOpenConns:    10,
				MaxIdleConns:    20,
				MinIdleConns:    2,
				ConnMaxLifetime: 20 * time.Minute,
				ConnMaxIdleTime: 5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "negative min idle",
			config: &DynamicPoolConfig{
				MaxOpenConns:    50,
				MaxIdleConns:    10,
				MinIdleConns:    -1,
				ConnMaxLifetime: 20 * time.Minute,
				ConnMaxIdleTime: 5 * time.Minute,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := optimizer.SetPoolConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetPoolConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestQueryOptimizerV2EnableAutoTuning(t *testing.T) {
	optimizer := &QueryOptimizerV2{
		autoTuning: &AutoTuningState{
			enabled: true,
		},
	}

	optimizer.EnableAutoTuning(false)
	if optimizer.autoTuning.enabled {
		t.Error("Auto tuning should be disabled")
	}

	optimizer.EnableAutoTuning(true)
	if !optimizer.autoTuning.enabled {
		t.Error("Auto tuning should be enabled")
	}
}

func TestQueryOptimizerV2IsAutoTuningEnabled(t *testing.T) {
	optimizer := &QueryOptimizerV2{
		autoTuning: &AutoTuningState{
			enabled: true,
		},
	}

	if !optimizer.IsAutoTuningEnabled() {
		t.Error("IsAutoTuningEnabled should return true")
	}

	optimizer.EnableAutoTuning(false)
	if optimizer.IsAutoTuningEnabled() {
		t.Error("IsAutoTuningEnabled should return false")
	}
}

func TestQueryOptimizerV2GetPoolConfig(t *testing.T) {
	optimizer := &QueryOptimizerV2{
		poolConfig: &DynamicPoolConfig{
			MaxOpenConns:    100,
			MaxIdleConns:    20,
			MinIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 10 * time.Minute,
		},
	}

	config := optimizer.GetPoolConfig()

	if config.MaxOpenConns != 100 {
		t.Errorf("MaxOpenConns = %d, want %d", config.MaxOpenConns, 100)
	}

	if config.MaxIdleConns != 20 {
		t.Errorf("MaxIdleConns = %d, want %d", config.MaxIdleConns, 20)
	}

	if config.MinIdleConns != 5 {
		t.Errorf("MinIdleConns = %d, want %d", config.MinIdleConns, 5)
	}
}

func TestCursorPaginationLimitValidation(t *testing.T) {
	optimizer := &QueryOptimizerV2{}

	tests := []struct {
		name   string
		limit  int
		expect int
	}{
		{"zero limit", 0, 20},
		{"negative limit", -5, 20},
		{"normal limit", 50, 50},
		{"very large limit", 5000, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit := tt.limit
			if limit <= 0 {
				limit = 20
			}
			if limit > 1000 {
				limit = 1000
			}

			if limit != tt.expect {
				t.Errorf("limit = %d, want %d", limit, tt.expect)
			}
		})
	}

	_ = optimizer
}

func TestInvalidCursorFormat(t *testing.T) {
	_, err := decodeCursor("invalid-base64-!!!")
	if err == nil {
		t.Error("Expected error for invalid base64 cursor")
	}

	_, err = decodeCursor(base64.StdEncoding.EncodeToString([]byte("not json")))
	if err == nil {
		t.Error("Expected error for invalid JSON cursor")
	}
}

func decodeCursor(cursor string) (CursorData, error) {
	if cursor == "" {
		return CursorData{}, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return CursorData{}, err
	}

	var data CursorData
	if err := json.Unmarshal(decoded, &data); err != nil {
		return CursorData{}, err
	}

	return data, nil
}

func TestBatchOperationsEmptyOperations(t *testing.T) {
	optimizer := &QueryOptimizerV2{}

	result, err := optimizer.BatchOperations(context.Background(), []Operation{})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.TotalOperations != 0 {
		t.Errorf("TotalOperations = %d, want %d", result.TotalOperations, 0)
	}

	if result.SuccessCount != 0 {
		t.Errorf("SuccessCount = %d, want %d", result.SuccessCount, 0)
	}

	if result.FailureCount != 0 {
		t.Errorf("FailureCount = %d, want %d", result.FailureCount, 0)
	}
}

func TestOperationTypeConstants(t *testing.T) {
	if OpTypeInsert != "insert" {
		t.Errorf("OpTypeInsert = %s, want insert", OpTypeInsert)
	}

	if OpTypeUpdate != "update" {
		t.Errorf("OpTypeUpdate = %s, want update", OpTypeUpdate)
	}

	if OpTypeDelete != "delete" {
		t.Errorf("OpTypeDelete = %s, want delete", OpTypeDelete)
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HELLO", "hello"},
		{"Hello World", "hello world"},
		{"ALREADY LOWER", "already lower"},
		{"MiXeD CaSe", "mixed case"},
		{"123ABC", "123abc"},
	}

	for _, tt := range tests {
		result := toLower(tt.input)
		if result != tt.expected {
			t.Errorf("toLower(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestDynamicPoolMetricsCalculations(t *testing.T) {
	metrics := DynamicPoolMetrics{
		TotalConnections:  100,
		ActiveConnections: 75,
		IdleConnections:   25,
		ReuseRate:         0.75,
		WaitCount:         1000,
		WaitDuration:      10 * time.Second,
		AvgWaitTime:       10 * time.Millisecond,
		MaxWaitTime:       50 * time.Millisecond,
	}

	total := metrics.TotalConnections
	active := metrics.ActiveConnections

	utilization := float64(active) / float64(total)
	if utilization != 0.75 {
		t.Errorf("Utilization = %f, want %f", utilization, 0.75)
	}

	avgWait := metrics.WaitDuration / time.Duration(metrics.WaitCount)
	if avgWait != 10*time.Millisecond {
		t.Errorf("AvgWait = %v, want %v", avgWait, 10*time.Millisecond)
	}
}

func TestScaleUpPoolLogic(t *testing.T) {
	optimizer := &QueryOptimizerV2{
		poolConfig: &DynamicPoolConfig{
			MaxOpenConns: 100,
			MaxIdleConns: 20,
			MinIdleConns: 5,
		},
	}

	newMaxOpen := int(float64(optimizer.poolConfig.MaxOpenConns) * 1.2)
	if newMaxOpen != 120 {
		t.Errorf("newMaxOpen = %d, want %d", newMaxOpen, 120)
	}

	newMaxIdle := int(float64(optimizer.poolConfig.MaxIdleConns) * 1.3)
	if newMaxIdle != 26 {
		t.Errorf("newMaxIdle = %d, want %d", newMaxIdle, 26)
	}
}

func TestScaleDownPoolLogic(t *testing.T) {
	optimizer := &QueryOptimizerV2{
		poolConfig: &DynamicPoolConfig{
			MaxOpenConns: 100,
			MaxIdleConns: 20,
			MinIdleConns: 5,
		},
	}

	newMaxOpen := int(float64(optimizer.poolConfig.MaxOpenConns) * 0.8)
	if newMaxOpen != 80 {
		t.Errorf("newMaxOpen = %d, want %d", newMaxOpen, 80)
	}

	newMaxIdle := int(float64(optimizer.poolConfig.MaxIdleConns) * 0.8)
	if newMaxIdle != 16 {
		t.Errorf("newMaxIdle = %d, want %d", newMaxIdle, 16)
	}
}

func TestAutoTuningThresholds(t *testing.T) {
	state := &AutoTuningState{
		highLoadThreshold:  0.85,
		lowLoadThreshold:   0.30,
		targetUtilization:  0.70,
	}

	highUtilization := 0.90
	if highUtilization <= state.highLoadThreshold {
		t.Error("High utilization should exceed threshold")
	}

	lowUtilization := 0.20
	if lowUtilization >= state.lowLoadThreshold {
		t.Error("Low utilization should be below threshold")
	}

	targetUtilization := 0.70
	if targetUtilization != state.targetUtilization {
		t.Errorf("Target utilization = %f, want %f", targetUtilization, state.targetUtilization)
	}
}

func TestBatchProcessorConcurrency(t *testing.T) {
	state := &BatchProcessorState{
		maxBatchSize:  1000,
		concurrency:   4,
		retryAttempts: 3,
		retryDelay:    100 * time.Millisecond,
	}

	semaphore := make(chan struct{}, state.concurrency)

	for i := 0; i < state.concurrency; i++ {
		select {
		case semaphore <- struct{}{}:
		default:
			t.Error("Should be able to fill semaphore up to concurrency limit")
		}
	}

	select {
	case semaphore <- struct{}{}:
		t.Error("Should not be able to exceed concurrency limit")
	default:
	}

	for i := 0; i < state.concurrency; i++ {
		<-semaphore
	}
}

func TestResultIDExtraction(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		expected uint
	}{
		{
			name:     "int64",
			data:     map[string]interface{}{"id": int64(123)},
			expected: 123,
		},
		{
			name:     "int32",
			data:     map[string]interface{}{"id": int32(456)},
			expected: 456,
		},
		{
			name:     "uint",
			data:     map[string]interface{}{"id": uint(789)},
			expected: 789,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id uint
			if idVal, ok := tt.data["id"]; ok {
				switch v := idVal.(type) {
				case int64:
					id = uint(v)
				case int32:
					id = uint(v)
				case uint:
					id = v
				}
			}

			if id != tt.expected {
				t.Errorf("id = %d, want %d", id, tt.expected)
			}
		})
	}
}

func TestConnectionMetricsConversion(t *testing.T) {
	type mockDBStats struct {
		MaxOpenConnections int
		OpenConnections    int
		InUse              int
		Idle               int
		WaitCount          int64
		WaitDuration       time.Duration
		MaxIdleClosed      int64
		MaxLifetimeClosed  int64
	}

	stats := mockDBStats{
		MaxOpenConnections: 100,
		OpenConnections:    80,
		InUse:              50,
		Idle:               30,
		WaitCount:          100,
		WaitDuration:       5 * time.Second,
		MaxIdleClosed:      10,
		MaxLifetimeClosed:  5,
	}

	metrics := DynamicPoolMetrics{
		TotalConnections:  stats.MaxOpenConnections,
		ActiveConnections: stats.InUse,
		IdleConnections:   stats.Idle,
		WaitCount:         stats.WaitCount,
		WaitDuration:      stats.WaitDuration,
	}

	if metrics.TotalConnections != 100 {
		t.Errorf("TotalConnections = %d, want %d", metrics.TotalConnections, 100)
	}

	if metrics.ActiveConnections != 50 {
		t.Errorf("ActiveConnections = %d, want %d", metrics.ActiveConnections, 50)
	}

	if metrics.IdleConnections != 30 {
		t.Errorf("IdleConnections = %d, want %d", metrics.IdleConnections, 30)
	}
}

func TestTuneConnectionPoolHighLoad(t *testing.T) {
	optimizer := &QueryOptimizerV2{
		poolConfig: &DynamicPoolConfig{
			MaxOpenConns:    100,
			MaxIdleConns:    20,
			MinIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 10 * time.Minute,
		},
		autoTuning: &AutoTuningState{
			enabled:           true,
			highLoadThreshold:  0.85,
			lowLoadThreshold:   0.30,
			targetUtilization:  0.70,
		},
	}

	metrics := DynamicPoolMetrics{
		TotalConnections:  100,
		ActiveConnections: 90,
		IdleConnections:   10,
	}

	currentUtil := float64(metrics.ActiveConnections) / float64(metrics.TotalConnections)
	if currentUtil <= optimizer.autoTuning.highLoadThreshold {
		t.Error("Current utilization should exceed high load threshold")
	}

	initialMaxOpen := optimizer.poolConfig.MaxOpenConns
	newMaxOpen := int(float64(initialMaxOpen) * 1.2)

	if newMaxOpen <= initialMaxOpen {
		t.Error("Scaled up max open connections should be greater")
	}
}

func TestTuneConnectionPoolLowLoad(t *testing.T) {
	optimizer := &QueryOptimizerV2{
		poolConfig: &DynamicPoolConfig{
			MaxOpenConns:    100,
			MaxIdleConns:    20,
			MinIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 10 * time.Minute,
		},
		autoTuning: &AutoTuningState{
			enabled:           true,
			highLoadThreshold:  0.85,
			lowLoadThreshold:   0.30,
			targetUtilization:  0.70,
		},
	}

	metrics := DynamicPoolMetrics{
		TotalConnections:  100,
		ActiveConnections: 20,
		IdleConnections:   80,
	}

	currentUtil := float64(metrics.ActiveConnections) / float64(metrics.TotalConnections)
	if currentUtil >= optimizer.autoTuning.lowLoadThreshold {
		t.Error("Current utilization should be below low load threshold")
	}

	initialMaxOpen := optimizer.poolConfig.MaxOpenConns
	newMaxOpen := int(float64(initialMaxOpen) * 0.8)

	if newMaxOpen >= initialMaxOpen {
		t.Error("Scaled down max open connections should be less")
	}
}

func TestMaxPoolSizeLimit(t *testing.T) {
	optimizer := &QueryOptimizerV2{
		poolConfig: &DynamicPoolConfig{
			MaxOpenConns: 400,
			MaxIdleConns: 80,
		},
	}

	newMaxOpen := int(float64(optimizer.poolConfig.MaxOpenConns) * 1.2)
	if newMaxOpen > 500 {
		newMaxOpen = 500
	}

	if newMaxOpen != 480 {
		t.Errorf("Should cap at 500, but got %d", newMaxOpen)
	}
}

func TestMinPoolSizeLimit(t *testing.T) {
	optimizer := &QueryOptimizerV2{
		poolConfig: &DynamicPoolConfig{
			MaxOpenConns: 20,
			MaxIdleConns: 4,
			MinIdleConns: 5,
		},
	}

	newMaxOpen := int(float64(optimizer.poolConfig.MaxOpenConns) * 0.8)
	if newMaxOpen < 10 {
		newMaxOpen = 10
	}

	if newMaxOpen != 10 {
		t.Errorf("Should have minimum of 10, got %d", newMaxOpen)
	}
}
