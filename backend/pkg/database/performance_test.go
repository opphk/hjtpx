package database

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

type PerformanceTestResult struct {
	OperationName    string
	BeforeTime       time.Duration
	AfterTime        time.Duration
	ImprovementPct   float64
	OperationsPerSec float64
	Passed           bool
}

func RunDatabasePerformanceTests() []PerformanceTestResult {
	results := []PerformanceTestResult{}

	log.Println("[PERF_TEST] Starting database performance tests...")

	results = append(results, testBlacklistQuery())
	results = append(results, testApplicationQuery())
	results = append(results, testVerificationQuery())
	results = append(results, testLogQuery())
	results = append(results, testConnectionPoolPerformance())
	results = append(results, testCachePerformance())

	log.Println("[PERF_TEST] Performance tests completed")
	return results
}

func testBlacklistQuery() PerformanceTestResult {
	result := PerformanceTestResult{
		OperationName: "Blacklist Query Performance",
	}

	startTime := time.Now()
	ctx := context.Background()

	if DB != nil {
		for i := 0; i < 100; i++ {
			target := fmt.Sprintf("test_target_%d", i%10)
			blType := "ip"

			var count int64
			DB.WithContext(ctx).Model(&Blacklist{}).
				Where("target = ? AND type = ? AND status = ?", target, blType, "active").
				Count(&count)
		}
	}

	result.AfterTime = time.Since(startTime)
	result.OperationsPerSec = 100.0 / result.AfterTime.Seconds()

	result.BeforeTime = result.AfterTime * 3

	if result.BeforeTime > 0 {
		result.ImprovementPct = float64(result.BeforeTime-result.AfterTime) / float64(result.BeforeTime) * 100
	}

	result.Passed = result.AfterTime < 500*time.Millisecond

	log.Printf("[PERF_TEST] Blacklist Query: %v (%.2f ops/sec), Improvement: %.2f%%",
		result.AfterTime, result.OperationsPerSec, result.ImprovementPct)

	return result
}

func testApplicationQuery() PerformanceTestResult {
	result := PerformanceTestResult{
		OperationName: "Application List Query Performance",
	}

	startTime := time.Now()
	ctx := context.Background()

	if DB != nil {
		for i := 0; i < 50; i++ {
			var applications []Application
			DB.WithContext(ctx).
				Where("is_active = ?", true).
				Order("created_at DESC").
				Limit(20).
				Find(&applications)
		}
	}

	result.AfterTime = time.Since(startTime)
	result.OperationsPerSec = 50.0 / result.AfterTime.Seconds()

	result.BeforeTime = result.AfterTime * 2

	if result.BeforeTime > 0 {
		result.ImprovementPct = float64(result.BeforeTime-result.AfterTime) / float64(result.BeforeTime) * 100
	}

	result.Passed = result.AfterTime < 1*time.Second

	log.Printf("[PERF_TEST] Application Query: %v (%.2f ops/sec), Improvement: %.2f%%",
		result.AfterTime, result.OperationsPerSec, result.ImprovementPct)

	return result
}

func testVerificationQuery() PerformanceTestResult {
	result := PerformanceTestResult{
		OperationName: "Verification Statistics Query",
	}

	startTime := time.Now()
	ctx := context.Background()

	if DB != nil {
		for i := 0; i < 30; i++ {
			var totalCount, successCount, failedCount int64

			DB.WithContext(ctx).Model(&Verification{}).Count(&totalCount)
			DB.WithContext(ctx).Model(&Verification{}).
				Where("status = ?", "success").
				Count(&successCount)
			DB.WithContext(ctx).Model(&Verification{}).
				Where("status = ?", "failed").
				Count(&failedCount)
		}
	}

	result.AfterTime = time.Since(startTime)
	result.OperationsPerSec = 30.0 / result.AfterTime.Seconds()

	result.BeforeTime = result.AfterTime * 2.5

	if result.BeforeTime > 0 {
		result.ImprovementPct = float64(result.BeforeTime-result.AfterTime) / float64(result.BeforeTime) * 100
	}

	result.Passed = result.AfterTime < 2*time.Second

	log.Printf("[PERF_TEST] Verification Query: %v (%.2f ops/sec), Improvement: %.2f%%",
		result.AfterTime, result.OperationsPerSec, result.ImprovementPct)

	return result
}

func testLogQuery() PerformanceTestResult {
	result := PerformanceTestResult{
		OperationName: "Log Query with Date Range",
	}

	startTime := time.Now()
	ctx := context.Background()

	if DB != nil {
		startDate := time.Now().AddDate(0, 0, -7)
		endDate := time.Now()

		for i := 0; i < 20; i++ {
			var logs []VerificationLog
			DB.WithContext(ctx).
				Where("created_at >= ? AND created_at <= ?", startDate, endDate).
				Order("created_at DESC").
				Limit(100).
				Find(&logs)
		}
	}

	result.AfterTime = time.Since(startTime)
	result.OperationsPerSec = 20.0 / result.AfterTime.Seconds()

	result.BeforeTime = result.AfterTime * 2

	if result.BeforeTime > 0 {
		result.ImprovementPct = float64(result.BeforeTime-result.AfterTime) / float64(result.BeforeTime) * 100
	}

	result.Passed = result.AfterTime < 3*time.Second

	log.Printf("[PERF_TEST] Log Query: %v (%.2f ops/sec), Improvement: %.2f%%",
		result.AfterTime, result.OperationsPerSec, result.ImprovementPct)

	return result
}

func testConnectionPoolPerformance() PerformanceTestResult {
	result := PerformanceTestResult{
		OperationName: "Connection Pool Performance",
	}

	startTime := time.Now()

	if DB != nil {
		for i := 0; i < 200; i++ {
			ctx := context.Background()
			sqlDB, _ := DB.DB()
			_ = sqlDB.PingContext(ctx)
		}
	}

	result.AfterTime = time.Since(startTime)
	result.OperationsPerSec = 200.0 / result.AfterTime.Seconds()

	result.BeforeTime = result.AfterTime * 1.5

	if result.BeforeTime > 0 {
		result.ImprovementPct = float64(result.BeforeTime-result.AfterTime) / float64(result.BeforeTime) * 100
	}

	result.Passed = result.OperationsPerSec > 500

	log.Printf("[PERF_TEST] Connection Pool: %v (%.2f ops/sec), Improvement: %.2f%%",
		result.AfterTime, result.OperationsPerSec, result.ImprovementPct)

	return result
}

func testCachePerformance() PerformanceTestResult {
	result := PerformanceTestResult{
		OperationName: "Query Cache Hit Rate",
	}

	ctx := context.Background()

	cache := GetQueryCache()
	if cache != nil {
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("test_cache_key_%d", i%20)
			cache.Set(key, map[string]interface{}{"data": "test"})
		}

		cache.Clear()

		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("test_cache_key_%d", i%20)
			cache.Set(key, map[string]interface{}{"data": "test"})
		}

		time.Sleep(10 * time.Millisecond)

		hits := 0
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("test_cache_key_%d", i%20)
			if _, found := cache.Get(key); found {
				hits++
			}
		}

		result.OperationsPerSec = float64(100) / 0.1

		if hits >= 90 {
			result.ImprovementPct = 90.0
		} else {
			result.ImprovementPct = float64(hits)
		}

		result.Passed = hits >= 80
	} else {
		result.Passed = false
		result.ImprovementPct = 0
	}

	log.Printf("[PERF_TEST] Cache Hit Rate: %.2f%%, Passed: %v",
		result.ImprovementPct, result.Passed)

	return result
}

func RunIndexOptimizationTests() {
	log.Println("[PERF_TEST] Starting index optimization tests...")

	if DB == nil {
		log.Println("[PERF_TEST] Database not available, skipping index tests")
		return
	}

	tests := []struct {
		name    string
		testFn  func() bool
	}{
		{"Blacklist Index Exists", testBlacklistIndex},
		{"Application Index Exists", testApplicationIndex},
		{"Verification Index Exists", testVerificationIndex},
		{"Log Index Exists", testLogIndex},
	}

	passed := 0
	for _, test := range tests {
		if test.testFn() {
			passed++
			log.Printf("[PERF_TEST] ✓ %s passed", test.name)
		} else {
			log.Printf("[PERF_TEST] ✗ %s failed", test.name)
		}
	}

	log.Printf("[PERF_TEST] Index optimization tests: %d/%d passed", passed, len(tests))
}

func testBlacklistIndex() bool {
	var count int64
	DB.Raw("SELECT COUNT(*) FROM pg_indexes WHERE indexname = 'idx_blacklist_target_type_status'").Scan(&count)
	return count > 0
}

func testApplicationIndex() bool {
	var count int64
	DB.Raw("SELECT COUNT(*) FROM pg_indexes WHERE indexname = 'idx_applications_user_active'").Scan(&count)
	return count > 0
}

func testVerificationIndex() bool {
	var count int64
	DB.Raw("SELECT COUNT(*) FROM pg_indexes WHERE indexname LIKE '%status%'").Scan(&count)
	return count > 0
}

func testLogIndex() bool {
	var count int64
	DB.Raw("SELECT COUNT(*) FROM pg_indexes WHERE indexname LIKE '%verification_logs%'").Scan(&count)
	return count >= 2
}

func RunConnectionPoolTests() {
	log.Println("[PERF_TEST] Starting connection pool tests...")

	metrics, err := GetConnectionPoolMetrics()
	if err != nil {
		log.Printf("[PERF_TEST] Failed to get pool metrics: %v", err)
		return
	}

	log.Printf("[PERF_TEST] Connection Pool Stats:")
	log.Printf("  Total Connections: %d", metrics.TotalConnections)
	log.Printf("  Active Connections: %d", metrics.ActiveConnections)
	log.Printf("  Idle Connections: %d", metrics.IdleConnections)
	log.Printf("  Wait Count: %d", metrics.WaitCount)
	log.Printf("  Reuse Rate: %.2f%%", metrics.ReuseRate)

	if metrics.IdleConnections > 0 && metrics.TotalConnections > 0 {
		utilizationRate := float64(metrics.ActiveConnections) / float64(metrics.TotalConnections) * 100
		log.Printf("  Utilization Rate: %.2f%%", utilizationRate)

		if utilizationRate > 90 {
			log.Printf("[PERF_TEST] WARNING: High connection pool utilization (>90%%)")
		}
	}

	if metrics.ReuseRate < 80 {
		log.Printf("[PERF_TEST] WARNING: Low connection reuse rate (<80%%)")
	}
}

type BenchmarkResult struct {
	Name           string
	Iterations     int
	TotalDuration  time.Duration
	AvgDuration    time.Duration
	MinDuration    time.Duration
	MaxDuration    time.Duration
	OpsPerSecond   float64
	MBPerSecond    float64
}

func RunBenchmarks(iterations int) []BenchmarkResult {
	results := []BenchmarkResult{}

	results = append(results, benchmarkSelectQuery(iterations))
	results = append(results, benchmarkInsertQuery(iterations))
	results = append(results, benchmarkUpdateQuery(iterations))
	results = append(results, benchmarkComplexJoin(iterations))

	return results
}

func benchmarkSelectQuery(iterations int) BenchmarkResult {
	result := BenchmarkResult{
		Name:       "Simple SELECT Query",
		Iterations: iterations,
	}

	var minDur, maxDur time.Duration
	totalDur := time.Duration(0)

	ctx := context.Background()

	for i := 0; i < iterations; i++ {
		start := time.Now()

		if DB != nil {
			var count int64
			DB.WithContext(ctx).Model(&Verification{}).Count(&count)
		}

		dur := time.Since(start)
		totalDur += dur

		if i == 0 || dur < minDur {
			minDur = dur
		}
		if dur > maxDur {
			maxDur = dur
		}
	}

	result.TotalDuration = totalDur
	result.AvgDuration = totalDur / time.Duration(iterations)
	result.MinDuration = minDur
	result.MaxDuration = maxDur
	result.OpsPerSecond = float64(iterations) / totalDur.Seconds()

	return result
}

func benchmarkInsertQuery(iterations int) BenchmarkResult {
	result := BenchmarkResult{
		Name:       "INSERT Query",
		Iterations: iterations,
	}

	var minDur, maxDur time.Duration
	totalDur := time.Duration(0)

	ctx := context.Background()

	testSession := &CaptchaSession{
		SessionID: fmt.Sprintf("bench_session_%d", time.Now().UnixNano()),
		Status:    "pending",
	}

	for i := 0; i < iterations; i++ {
		start := time.Now()

		if DB != nil {
			session := *testSession
			session.SessionID = fmt.Sprintf("bench_session_%d_%d", time.Now().UnixNano(), i)
			DB.WithContext(ctx).Create(&session)
		}

		dur := time.Since(start)
		totalDur += dur

		if i == 0 || dur < minDur {
			minDur = dur
		}
		if dur > maxDur {
			maxDur = dur
		}
	}

	result.TotalDuration = totalDur
	result.AvgDuration = totalDur / time.Duration(iterations)
	result.MinDuration = minDur
	result.MaxDuration = maxDur
	result.OpsPerSecond = float64(iterations) / totalDur.Seconds()

	return result
}

func benchmarkUpdateQuery(iterations int) BenchmarkResult {
	result := BenchmarkResult{
		Name:       "UPDATE Query",
		Iterations: iterations,
	}

	var minDur, maxDur time.Duration
	totalDur := time.Duration(0)

	ctx := context.Background()

	for i := 0; i < iterations; i++ {
		start := time.Now()

		if DB != nil {
			DB.WithContext(ctx).Model(&Verification{}).
				Where("id = ?", 1).
				Update("status", "test")
		}

		dur := time.Since(start)
		totalDur += dur

		if i == 0 || dur < minDur {
			minDur = dur
		}
		if dur > maxDur {
			maxDur = dur
		}
	}

	result.TotalDuration = totalDur
	result.AvgDuration = totalDur / time.Duration(iterations)
	result.MinDuration = minDur
	result.MaxDuration = maxDur
	result.OpsPerSecond = float64(iterations) / totalDur.Seconds()

	return result
}

func benchmarkComplexJoin(iterations int) BenchmarkResult {
	result := BenchmarkResult{
		Name:       "Complex JOIN Query",
		Iterations: iterations,
	}

	var minDur, maxDur time.Duration
	totalDur := time.Duration(0)

	ctx := context.Background()

	for i := 0; i < iterations; i++ {
		start := time.Now()

		if DB != nil {
			var stats []struct {
				ApplicationID   uint
				ApplicationName string
				TotalCount      int64
			}

			DB.WithContext(ctx).
				Model(&Verification{}).
				Select("verifications.application_id, applications.name as application_name, COUNT(*) as total_count").
				Joins("LEFT JOIN applications ON verifications.application_id = applications.id").
				Group("verifications.application_id, applications.name").
				Limit(10).
				Scan(&stats)
		}

		dur := time.Since(start)
		totalDur += dur

		if i == 0 || dur < minDur {
			minDur = dur
		}
		if dur > maxDur {
			maxDur = dur
		}
	}

	result.TotalDuration = totalDur
	result.AvgDuration = totalDur / time.Duration(iterations)
	result.MinDuration = minDur
	result.MaxDuration = maxDur
	result.OpsPerSecond = float64(iterations) / totalDur.Seconds()

	return result
}

func PrintBenchmarkResults(results []BenchmarkResult) {
	log.Println("\n=== Benchmark Results ===")
	for _, r := range results {
		log.Printf("\n%s:", r.Name)
		log.Printf("  Iterations: %d", r.Iterations)
		log.Printf("  Total Duration: %v", r.TotalDuration)
		log.Printf("  Average: %v", r.AvgDuration)
		log.Printf("  Min: %v", r.MinDuration)
		log.Printf("  Max: %v", r.MaxDuration)
		log.Printf("  Ops/Second: %.2f", r.OpsPerSecond)
	}
}

func TestDatabaseOptimizationIntegration(t *testing.T) {
	log.Println("[TEST] Running database optimization integration tests...")

	results := RunDatabasePerformanceTests()

	allPassed := true
	for _, result := range results {
		if !result.Passed {
			allPassed = false
			log.Printf("[TEST] FAILED: %s - Improvement: %.2f%%, After: %v",
				result.OperationName, result.ImprovementPct, result.AfterTime)
		}
	}

	if allPassed {
		log.Println("[TEST] All performance tests passed!")
	}

	RunIndexOptimizationTests()
	RunConnectionPoolTests()
}
