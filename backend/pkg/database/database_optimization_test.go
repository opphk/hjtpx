package database

import (
	"testing"
	"time"
)

func TestDataArchiverCreation(t *testing.T) {
	archiver := &DataArchiver{
		enabled:          true,
		archiveThreshold: 30 * 24 * time.Hour,
		archivePrefix:    "archive_",
		cleanupEnabled:   true,
		cleanupThreshold: 365 * 24 * time.Hour,
	}

	if !archiver.enabled {
		t.Error("Archiver should be enabled")
	}

	if archiver.archiveThreshold != 30*24*time.Hour {
		t.Errorf("Archive threshold = %v, want %v", archiver.archiveThreshold, 30*24*time.Hour)
	}
}

func TestArchiveStats(t *testing.T) {
	stats := &ArchiveStats{
		TotalArchivedRecords: 1000,
		TotalCleanedRecords:  500,
		LastArchiveTime:      time.Now(),
		LastCleanupTime:      time.Now(),
		ArchiveErrors:        5,
		CleanupErrors:        3,
	}

	if stats.TotalArchivedRecords != 1000 {
		t.Errorf("TotalArchivedRecords = %d, want %d", stats.TotalArchivedRecords, 1000)
	}

	if stats.TotalCleanedRecords != 500 {
		t.Errorf("TotalCleanedRecords = %d, want %d", stats.TotalCleanedRecords, 500)
	}
}

func TestHotColdSeparator(t *testing.T) {
	separator := NewHotColdSeparator(24*time.Hour, 30*24*time.Hour)

	if separator.hotThreshold != 24*time.Hour {
		t.Errorf("Hot threshold = %v, want %v", separator.hotThreshold, 24*time.Hour)
	}

	if separator.coldThreshold != 30*24*time.Hour {
		t.Errorf("Cold threshold = %v, want %v", separator.coldThreshold, 30*24*time.Hour)
	}
}

func TestTimeBasedArchiveStrategy(t *testing.T) {
	strategy := NewTimeBasedArchiveStrategy("logs", "created_at", 30*24*time.Hour)

	if strategy.tableName != "logs" {
		t.Errorf("Table name = %s, want %s", strategy.tableName, "logs")
	}

	if strategy.dateField != "created_at" {
		t.Errorf("Date field = %s, want %s", strategy.dateField, "created_at")
	}

	if strategy.GetName() != "time_based_archive" {
		t.Errorf("GetName() = %s, want %s", strategy.GetName(), "time_based_archive")
	}

	if strategy.GetStatus() != "pending" {
		t.Errorf("Initial status = %s, want %s", strategy.GetStatus(), "pending")
	}
}

func TestSizeBasedArchiveStrategy(t *testing.T) {
	strategy := NewSizeBasedArchiveStrategy("logs", "created_at", 1000000)

	if strategy.tableName != "logs" {
		t.Errorf("Table name = %s, want %s", strategy.tableName, "logs")
	}

	if strategy.maxSize != 1000000 {
		t.Errorf("Max size = %d, want %d", strategy.maxSize, 1000000)
	}

	if strategy.GetName() != "size_based_archive" {
		t.Errorf("GetName() = %s, want %s", strategy.GetName(), "size_based_archive")
	}
}

func TestArchiveScheduler(t *testing.T) {
	scheduler := NewArchiveScheduler()

	if len(scheduler.strategies) != 0 {
		t.Errorf("Initial strategies count = %d, want %d", len(scheduler.strategies), 0)
	}

	strategy := NewTimeBasedArchiveStrategy("logs", "created_at", 30*24*time.Hour)
	scheduler.AddStrategy(strategy)

	if len(scheduler.strategies) != 1 {
		t.Errorf("After add, strategies count = %d, want %d", len(scheduler.strategies), 1)
	}

	strategies := scheduler.GetStrategies()
	if len(strategies) != 1 {
		t.Errorf("GetStrategies count = %d, want %d", len(strategies), 1)
	}
}

func TestCompressionArchiver(t *testing.T) {
	archiver := NewCompressionArchiver(6)

	if !archiver.enabled {
		t.Error("Compression archiver should be enabled")
	}

	if archiver.compressionLevel != 6 {
		t.Errorf("Compression level = %d, want %d", archiver.compressionLevel, 6)
	}
}

func TestCompressionArchiverInvalidLevel(t *testing.T) {
	archiver := NewCompressionArchiver(15)

	if archiver.compressionLevel != 6 {
		t.Errorf("Invalid level should default to 6, got %d", archiver.compressionLevel)
	}

	archiver2 := NewCompressionArchiver(0)
	if archiver2.compressionLevel != 6 {
		t.Errorf("Invalid level should default to 6, got %d", archiver2.compressionLevel)
	}
}

func TestCompressionArchiverGetRatio(t *testing.T) {
	archiver := NewCompressionArchiver(6)

	ratio, err := archiver.GetCompressionRatio("test_table")
	if err != nil {
		t.Errorf("GetCompressionRatio should not error: %v", err)
	}

	if ratio <= 0 || ratio > 1 {
		t.Errorf("Compression ratio = %f, should be between 0 and 1", ratio)
	}
}

func TestConnectionPoolConfig(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    20,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	if err := config.Validate(); err != nil {
		t.Errorf("Valid config should not error: %v", err)
	}
}

func TestConnectionPoolConfigInvalidMaxOpenConns(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxOpenConns:    0,
		MaxIdleConns:    20,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	if err := config.Validate(); err == nil {
		t.Error("Invalid config should return error")
	}
}

func TestConnectionPoolConfigInvalidMaxIdleConns(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    0,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	if err := config.Validate(); err == nil {
		t.Error("Invalid config should return error")
	}
}

func TestConnectionPoolConfigIdleExceedsOpen(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxOpenConns:    10,
		MaxIdleConns:    20,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	if err := config.Validate(); err == nil {
		t.Error("maxIdleConns > maxOpenConns should return error")
	}
}

func TestConnectionPoolConfigOptimize(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    50,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	config.Optimize(0.3)

	if config.MaxIdleConns >= 50 {
		t.Errorf("Idle connections should be reduced at low ratio, got %d", config.MaxIdleConns)
	}

	config2 := &ConnectionPoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    50,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	config2.Optimize(0.95)

	if config2.MaxIdleConns <= 50 {
		t.Errorf("Idle connections should be increased at high ratio, got %d", config2.MaxIdleConns)
	}
}

func TestConnectionPoolOptimizer(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    20,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	optimizer := NewConnectionPoolOptimizer(config)

	if optimizer.currentConfig != config {
		t.Error("Optimizer should use provided config")
	}

	if optimizer.healthCheckInterval != 30*time.Second {
		t.Errorf("Health check interval = %v, want %v", optimizer.healthCheckInterval, 30*time.Second)
	}

	if !optimizer.autoTuningEnabled {
		t.Error("Auto tuning should be enabled by default")
	}
}

func TestConnectionPoolOptimizerGetConfig(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    20,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	optimizer := NewConnectionPoolOptimizer(config)
	result := optimizer.GetConfig()

	if result.MaxOpenConns != 100 {
		t.Errorf("MaxOpenConns = %d, want %d", result.MaxOpenConns, 100)
	}
}

func TestConnectionPoolOptimizerEnableAutoTuning(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    20,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	optimizer := NewConnectionPoolOptimizer(config)

	optimizer.EnableAutoTuning(false)
	if optimizer.autoTuningEnabled {
		t.Error("Auto tuning should be disabled")
	}

	optimizer.EnableAutoTuning(true)
	if !optimizer.autoTuningEnabled {
		t.Error("Auto tuning should be enabled")
	}
}

func TestConnectionPoolHealthCheck(t *testing.T) {
	checker := NewConnectionPoolHealthCheck()

	if len(checker.checkHistory) != 0 {
		t.Errorf("Initial history count = %d, want %d", len(checker.checkHistory), 0)
	}

	result := checker.GetLastCheck()
	if result != nil {
		t.Error("GetLastCheck should return nil when no checks have been run")
	}
}

func TestConnectionPoolHealthCheckGetHistory(t *testing.T) {
	checker := NewConnectionPoolHealthCheck()

	history := checker.GetHistory(10)
	if history == nil {
		t.Error("GetHistory should not return nil")
	}

	if len(history) != 0 {
		t.Errorf("Empty history length = %d, want %d", len(history), 0)
	}

	history = checker.GetHistory(0)
	if len(history) != 0 {
		t.Error("GetHistory with 0 should return empty")
	}
}

func TestConnectionPoolHealthCheckIsHealthy(t *testing.T) {
	checker := NewConnectionPoolHealthCheck()

	if checker.IsHealthy() {
		t.Error("Initial health should be false")
	}
}

func TestConnectionPoolManager(t *testing.T) {
	manager := GetConnectionPoolManager()

	if manager == nil {
		t.Fatal("GetConnectionPoolManager should not return nil")
	}

	if manager.optimizer == nil {
		t.Error("Optimizer should not be nil")
	}

	if manager.healthCheck == nil {
		t.Error("Health check should not be nil")
	}
}

func TestConnectionWrapper(t *testing.T) {
	wrapper := &ConnectionWrapper{}

	if wrapper.db != nil {
		t.Error("Initial db should be nil")
	}

	wrapper.RecordStats()

	stats := wrapper.GetStats()
	if stats.MaxOpenConnections != 0 {
		t.Error("Stats should be zero when db is nil")
	}
}

func TestPerformanceMonitorCreation(t *testing.T) {
	monitor := &PerformanceMonitor{
		queryMetrics:    make([]QueryMetric, 0),
		slowQueries:     make([]QueryMetric, 0),
		maxMetricsLen:   10000,
		maxSlowQueryLen: 1000,
		enabled:         true,
		slowThreshold:   50 * time.Millisecond,
	}

	if !monitor.enabled {
		t.Error("Monitor should be enabled")
	}

	if monitor.maxMetricsLen != 10000 {
		t.Errorf("Max metrics len = %d, want %d", monitor.maxMetricsLen, 10000)
	}
}

func TestPerformanceStats(t *testing.T) {
	stats := &PerformanceStats{
		TotalQueries:     1000,
		SlowQueries:      50,
		FailedQueries:    10,
		AvgDuration:      25 * time.Millisecond,
		MaxDuration:      500 * time.Millisecond,
		MinDuration:      5 * time.Millisecond,
		TotalDuration:    25 * time.Second,
		QueriesPerSecond: 100.5,
		SlowQueryRatio:   5.0,
	}

	if stats.TotalQueries != 1000 {
		t.Errorf("TotalQueries = %d, want %d", stats.TotalQueries, 1000)
	}

	if stats.SlowQueryRatio != 5.0 {
		t.Errorf("SlowQueryRatio = %f, want %f", stats.SlowQueryRatio, 5.0)
	}
}

func TestSlowQueryInfo(t *testing.T) {
	info := &SlowQueryInfo{
		Query:          "SELECT * FROM users",
		Count:          100,
		TotalDuration:  10 * time.Second,
		AvgDuration:    100 * time.Millisecond,
		MaxDuration:    500 * time.Millisecond,
		LastOccurrence: time.Now(),
		Suggestions:    []string{"Add index", "Use limit"},
	}

	if info.Count != 100 {
		t.Errorf("Count = %d, want %d", info.Count, 100)
	}

	if len(info.Suggestions) != 2 {
		t.Errorf("Suggestions count = %d, want %d", len(info.Suggestions), 2)
	}
}

func TestSlowQueryAnalyzer(t *testing.T) {
	analyzer := NewSlowQueryAnalyzer()

	if analyzer == nil {
		t.Fatal("NewSlowQueryAnalyzer should not return nil")
	}

	if len(analyzer.queries) != 0 {
		t.Error("Initial queries should be empty")
	}

	analyzer.Record("SELECT * FROM users", 100*time.Millisecond)

	if len(analyzer.queries) != 1 {
		t.Errorf("After record, queries count = %d, want %d", len(analyzer.queries), 1)
	}

	queries := analyzer.GetTopQueries(10)
	if len(queries) != 1 {
		t.Errorf("GetTopQueries should return 1, got %d", len(queries))
	}
}

func TestSlowQueryAnalyzerClear(t *testing.T) {
	analyzer := NewSlowQueryAnalyzer()

	analyzer.Record("SELECT * FROM users", 100*time.Millisecond)
	analyzer.Clear()

	if len(analyzer.queries) != 0 {
		t.Errorf("After clear, queries count = %d, want %d", len(analyzer.queries), 0)
	}
}

func TestMetricsAggregator(t *testing.T) {
	aggregator := NewMetricsAggregator(5*time.Minute, 60)

	if aggregator == nil {
		t.Fatal("NewMetricsAggregator should not return nil")
	}

	if len(aggregator.windows) != 0 {
		t.Error("Initial windows should be empty")
	}

	if aggregator.maxWindows != 60 {
		t.Errorf("Max windows = %d, want %d", aggregator.maxWindows, 60)
	}
}

func TestMetricsAggregatorRecordWindow(t *testing.T) {
	aggregator := NewMetricsAggregator(5*time.Minute, 5)

	stats := &PerformanceStats{
		TotalQueries: 100,
	}

	aggregator.RecordWindow(stats)

	if len(aggregator.windows) != 1 {
		t.Errorf("After record, windows count = %d, want %d", len(aggregator.windows), 1)
	}
}

func TestMetricsAggregatorGetTrend(t *testing.T) {
	aggregator := NewMetricsAggregator(5*time.Minute, 60)

	trend := aggregator.GetTrend()
	if trend != "insufficient_data" {
		t.Errorf("With no windows, trend = %s, want %s", trend, "insufficient_data")
	}
}

func TestAlertConfig(t *testing.T) {
	config := &AlertConfig{
		SlowQueryThreshold:      50 * time.Millisecond,
		AvgDurationThreshold:    30 * time.Millisecond,
		SlowQueryRatioThreshold: 5.0,
	}

	if config.SlowQueryThreshold != 50*time.Millisecond {
		t.Errorf("SlowQueryThreshold = %v, want %v", config.SlowQueryThreshold, 50*time.Millisecond)
	}
}

func TestQueryMetric(t *testing.T) {
	metric := QueryMetric{
		Query:        "SELECT * FROM users",
		Duration:     100 * time.Millisecond,
		Timestamp:    time.Now(),
		IsSlow:       true,
		ConnectionID: 123,
	}

	if metric.Duration != 100*time.Millisecond {
		t.Errorf("Duration = %v, want %v", metric.Duration, 100*time.Millisecond)
	}

	if !metric.IsSlow {
		t.Error("IsSlow should be true")
	}
}

func TestPoolStatsRecord(t *testing.T) {
	record := PoolStatsRecord{
		Timestamp: time.Now(),
		Stats:     PoolStats{},
	}

	if record.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestQueryCacheStrategies(t *testing.T) {
	cache := &QueryCache{
		cache:    make(map[string]cacheEntry),
		maxSize:  100,
		ttl:      5 * time.Minute,
		enabled:  true,
		strategy: StrategyLRU,
	}

	if cache.strategy != StrategyLRU {
		t.Errorf("Initial strategy should be LRU, got %d", cache.strategy)
	}

	cache.SetStrategy(StrategyLFU)
	if cache.strategy != StrategyLFU {
		t.Errorf("Strategy should be LFU, got %d", cache.strategy)
	}

	cache.SetStrategy(StrategyAdaptive)
	if cache.strategy != StrategyAdaptive {
		t.Errorf("Strategy should be Adaptive, got %d", cache.strategy)
	}
}

func TestQueryCacheEvictionStrategies(t *testing.T) {
	cache := &QueryCache{
		cache:    make(map[string]cacheEntry),
		maxSize:  3,
		ttl:      1 * time.Hour,
		enabled:  true,
		strategy: StrategyLRU,
	}

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	if len(cache.cache) != 3 {
		t.Errorf("Cache size should be 3, got %d", len(cache.cache))
	}

	cache.Set("key4", "value4")
	if len(cache.cache) != 3 {
		t.Errorf("After eviction, cache size should be 3, got %d", len(cache.cache))
	}

	cache.SetStrategy(StrategyLFU)
	cache.Set("key5", "value5")
	if len(cache.cache) != 3 {
		t.Errorf("After LFU eviction, cache size should be 3, got %d", len(cache.cache))
	}
}

func TestQueryCacheAutoAdjustStrategy(t *testing.T) {
	cache := &QueryCache{
		cache:    make(map[string]cacheEntry),
		maxSize:  100,
		ttl:      5 * time.Minute,
		enabled:  true,
		strategy: StrategyLRU,
	}

	cache.hits.Store(40)
	cache.misses.Store(60)
	cache.AutoAdjustStrategy()
	if cache.strategy != StrategyLFU {
		t.Errorf("With hit rate 40%, strategy should be LFU, got %s", cache.getStrategyName())
	}

	cache.hits.Store(60)
	cache.misses.Store(40)
	cache.AutoAdjustStrategy()
	if cache.strategy != StrategyAdaptive {
		t.Errorf("With hit rate 60%, strategy should be Adaptive, got %s", cache.getStrategyName())
	}

	cache.hits.Store(80)
	cache.misses.Store(20)
	cache.AutoAdjustStrategy()
	if cache.strategy != StrategyLRU {
		t.Errorf("With hit rate 80%, strategy should be LRU, got %s", cache.getStrategyName())
	}
}

func TestQueryCacheWarmup(t *testing.T) {
	cache := &QueryCache{
		cache:    make(map[string]cacheEntry),
		maxSize:  100,
		ttl:      5 * time.Minute,
		enabled:  true,
		strategy: StrategyLRU,
	}

	data := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	cache.Warmup(data, 10*time.Minute)

	if len(cache.cache) != 3 {
		t.Errorf("After warmup, cache size should be 3, got %d", len(cache.cache))
	}

	status := cache.GetWarmupStatus()
	if status["total_entries"] != 3 {
		t.Errorf("Warmup status total_entries should be 3, got %v", status["total_entries"])
	}
}

func TestQueryCacheInvalidation(t *testing.T) {
	cache := &QueryCache{
		cache:    make(map[string]cacheEntry),
		maxSize:  100,
		ttl:      5 * time.Minute,
		enabled:  true,
		strategy: StrategyLRU,
	}

	cache.Set("table:users:1", "user1")
	cache.Set("table:users:2", "user2")
	cache.Set("table:products:1", "product1")

	if len(cache.cache) != 3 {
		t.Errorf("Cache size should be 3, got %d", len(cache.cache))
	}

	cache.InvalidateKeysByPattern("table:users")

	if len(cache.cache) != 1 {
		t.Errorf("After invalidation, cache size should be 1, got %d", len(cache.cache))
	}
}

func TestDBRouterLoadBalanceStrategies(t *testing.T) {
	router := &DBRouter{
		enabled:         true,
		loadBalanceMode: "round_robin",
	}

	if router.loadBalanceMode != "round_robin" {
		t.Errorf("Load balance mode should be round_robin, got %s", router.loadBalanceMode)
	}

	router.SetLoadBalanceMode("random")
	if router.loadBalanceMode != "random" {
		t.Errorf("Load balance mode should be random, got %s", router.loadBalanceMode)
	}

	router.SetLoadBalanceMode("weighted_round_robin")
	if router.loadBalanceMode != "weighted_round_robin" {
		t.Errorf("Load balance mode should be weighted_round_robin, got %s", router.loadBalanceMode)
	}
}

func TestDBRouterMetrics(t *testing.T) {
	router := &DBRouter{
		enabled:         true,
		loadBalanceMode: "round_robin",
		metrics:         &RouterMetrics{},
	}

	router.RecordQuery(true, 10*time.Millisecond)
	router.RecordQuery(false, 5*time.Millisecond)

	metrics := router.GetMetrics()
	if metrics["master_queries"] != int64(1) {
		t.Errorf("Master queries should be 1, got %v", metrics["master_queries"])
	}
	if metrics["slave_queries"] != int64(1) {
		t.Errorf("Slave queries should be 1, got %v", metrics["slave_queries"])
	}

	router.RecordFailure()
	metrics = router.GetMetrics()
	if metrics["failed_queries"] != int64(1) {
		t.Errorf("Failed queries should be 1, got %v", metrics["failed_queries"])
	}
}

func TestSlaveHealthChecker(t *testing.T) {
	checker := NewSlaveHealthChecker(nil, 30*time.Second)

	if !checker.enabled {
		t.Error("Health checker should be enabled by default")
	}

	if checker.maxFailCount != 3 {
		t.Errorf("Max fail count should be 3, got %d", checker.maxFailCount)
	}

	checker.SetMaxFailCount(5)
	if checker.maxFailCount != 5 {
		t.Errorf("Max fail count should be 5, got %d", checker.maxFailCount)
	}

	checker.SetFailoverEnabled(false)
	if checker.failoverEnabled {
		t.Error("Failover should be disabled")
	}
}

func TestIndexFragmentationStruct(t *testing.T) {
	frag := IndexFragmentation{
		IndexName:         "idx_test",
		TableName:         "test_table",
		IndexSize:         "10MB",
		FragmentationRatio: 25.5,
		ScanCount:         1000,
	}

	if frag.IndexName != "idx_test" {
		t.Errorf("IndexName should be idx_test, got %s", frag.IndexName)
	}

	if frag.FragmentationRatio != 25.5 {
		t.Errorf("FragmentationRatio should be 25.5, got %f", frag.FragmentationRatio)
	}
}

func TestIndexEfficiencyStruct(t *testing.T) {
	eff := IndexEfficiency{
		IndexName:     "idx_test",
		TableName:     "test_table",
		IndexSize:     "10MB",
		ScanCount:     1000,
		HitRatio:      85.5,
		UsageLevel:    "high",
	}

	if eff.HitRatio != 85.5 {
		t.Errorf("HitRatio should be 85.5, got %f", eff.HitRatio)
	}

	if eff.UsageLevel != "high" {
		t.Errorf("UsageLevel should be high, got %s", eff.UsageLevel)
	}
}

func TestTableFragmentationStruct(t *testing.T) {
	frag := TableFragmentation{
		TableName:         "test_table",
		TotalSize:         "100MB",
		TableSize:         "80MB",
		IndexSize:         "20MB",
		DeadTuples:        1000,
		LiveTuples:        9000,
		FragmentationRatio: 10.0,
	}

	if frag.FragmentationRatio != 10.0 {
		t.Errorf("FragmentationRatio should be 10.0, got %f", frag.FragmentationRatio)
	}

	if frag.DeadTuples != 1000 {
		t.Errorf("DeadTuples should be 1000, got %d", frag.DeadTuples)
	}
}

func TestFragmentationReportStruct(t *testing.T) {
	report := &FragmentationReport{
		Timestamp:       time.Now(),
		TotalTables:     10,
		FragmentedTables: 3,
		AverageFragmentation: 15.5,
		MaxFragmentation: 30.0,
		MinFragmentation: 5.0,
	}

	if report.TotalTables != 10 {
		t.Errorf("TotalTables should be 10, got %d", report.TotalTables)
	}

	if report.AverageFragmentation != 15.5 {
		t.Errorf("AverageFragmentation should be 15.5, got %f", report.AverageFragmentation)
	}
}

func TestIntelligentQueryCachePatterns(t *testing.T) {
	iqc := &IntelligentQueryCache{
		queryPatterns: make(map[string]*QueryPattern),
	}

	iqc.RecordQuery("SELECT * FROM users", 50*time.Millisecond, true)

	if len(iqc.queryPatterns) != 1 {
		t.Errorf("Query patterns count should be 1, got %d", len(iqc.queryPatterns))
	}

	stats := iqc.GetPatternStats()
	if stats["total_patterns"] != 1 {
		t.Errorf("Total patterns should be 1, got %v", stats["total_patterns"])
	}

	ttl := iqc.GetOptimalTTL("SELECT * FROM users")
	if ttl != 5*time.Minute {
		t.Errorf("Optimal TTL should be 5m, got %v", ttl)
	}
}

func TestIntelligentQueryCacheAutoAdjustTTL(t *testing.T) {
	iqc := &IntelligentQueryCache{
		queryPatterns: make(map[string]*QueryPattern),
	}

	iqc.queryPatterns["test_query"] = &QueryPattern{
		Pattern:    "test_query",
		HitRate:    95.0,
		OptimalTTL: 5 * time.Minute,
	}

	iqc.AutoAdjustTTL()

	pattern := iqc.queryPatterns["test_query"]
	if pattern.OptimalTTL != 10*time.Minute {
		t.Errorf("High hit rate should increase TTL to 10m, got %v", pattern.OptimalTTL)
	}

	pattern.HitRate = 20.0
	iqc.AutoAdjustTTL()
	if pattern.OptimalTTL != 2*time.Minute+30*time.Second {
		t.Errorf("Low hit rate should decrease TTL, got %v", pattern.OptimalTTL)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{1000, "1000 B"},
		{1500, "1.46 KB"},
		{2000000, "1.91 MB"},
		{3000000000, "2.79 GB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}
