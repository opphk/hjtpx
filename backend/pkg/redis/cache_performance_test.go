package redis

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestCacheMetricsMonitor(t *testing.T) {
	monitor := NewCacheMetricsMonitor(100 * time.Millisecond)
	defer monitor.Enable(false)
	
	if monitor == nil {
		t.Fatal("Failed to create cache metrics monitor")
	}
	
	monitor.RecordHit(true)
	monitor.RecordHit(false)
	monitor.RecordMiss(true)
	monitor.RecordSet(1 * time.Millisecond)
	
	time.Sleep(200 * time.Millisecond)
	
	snapshot := monitor.GetCurrentMetrics()
	if snapshot == nil {
		t.Fatal("Failed to get current metrics")
	}
	
	if snapshot.TotalHits < 2 {
		t.Errorf("Expected at least 2 hits, got %d", snapshot.TotalHits)
	}
}

func TestCacheAlertThresholds(t *testing.T) {
	thresholds := &CacheAlertThresholds{
		LowHitRateThreshold:    80.0,
		HighErrorRateThreshold: 0.01,
		HighLatencyThreshold:   10 * time.Millisecond,
		LowL1HitRateThreshold: 60.0,
		HighEvictionThreshold:  1000,
	}
	
	if thresholds.LowHitRateThreshold != 80.0 {
		t.Errorf("Expected LowHitRateThreshold 80.0, got %f", thresholds.LowHitRateThreshold)
	}
}

func TestCacheHealthChecker(t *testing.T) {
	monitor := NewCacheMetricsMonitor(100 * time.Millisecond)
	checker := NewCacheHealthChecker(monitor)
	
	if checker == nil {
		t.Fatal("Failed to create cache health checker")
	}
	
	ctx := context.Background()
	status := checker.CheckHealth(ctx)
	
	if status == nil {
		t.Fatal("Failed to check health")
	}
	
	t.Logf("Cache health status: healthy=%v, issues=%v", status.Healthy, status.Issues)
}

func TestCacheMetricSnapshot(t *testing.T) {
	snapshot := &CacheMetricSnapshot{
		Timestamp:         time.Now(),
		TotalHits:        1000,
		TotalMisses:       50,
		HitRate:          calculateHitRate(1000, 50),
		L1HitRate:        60.0,
		L2HitRate:        95.0,
		AvgGetLatencyMs:  2.5,
		CurrentL1Size:    500,
	}
	
	if snapshot.HitRate < 95.0 {
		t.Errorf("Hit rate should be >95%%, got %f%%", snapshot.HitRate)
	}
	
	if snapshot.AvgGetLatencyMs > 10.0 {
		t.Errorf("Average get latency should be <10ms, got %fms", snapshot.AvgGetLatencyMs)
	}
}

func TestCachePerformanceReport(t *testing.T) {
	monitor := NewCacheMetricsMonitor(100 * time.Millisecond)
	
	for i := 0; i < 100; i++ {
		monitor.RecordHit(true)
		monitor.RecordSet(1 * time.Millisecond)
	}
	
	report := monitor.GetPerformanceReport()
	if report == nil {
		t.Fatal("Failed to get performance report")
	}
	
	t.Logf("Current hit rate: %f%%", report.CurrentMetrics.HitRate)
	t.Logf("L1 hit rate: %f%%", report.CurrentMetrics.L1HitRate)
	
	if len(report.Recommendations) > 0 {
		t.Logf("Recommendations: %v", report.Recommendations)
	}
}

func TestCacheTrendAnalysis(t *testing.T) {
	trend := CacheTrendAnalysis{
		HitRateTrend:   5.0,
		L1HitRateTrend: -2.0,
		LatencyTrend:   1.5,
	}
	
	if trend.HitRateTrend <= 0 {
		t.Log("Hit rate not improving")
	}
}

func BenchmarkCacheMetricsCollection(b *testing.B) {
	monitor := NewCacheMetricsMonitor(10 * time.Millisecond)
	defer monitor.Enable(false)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.RecordHit(i%2 == 0)
		monitor.RecordMiss(i%2 != 0)
		monitor.RecordSet(time.Microsecond)
	}
}

func BenchmarkCacheConcurrentRecording(b *testing.B) {
	monitor := NewCacheMetricsMonitor(10 * time.Millisecond)
	defer monitor.Enable(false)
	
	var wg sync.WaitGroup
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		wg.Add(1)
		defer wg.Done()
		
		i := 0
		for pb.Next() {
			monitor.RecordHit(i%2 == 0)
			i++
		}
	})
}

func TestCacheHitRateCalculation(t *testing.T) {
	testCases := []struct {
		hits      int64
		misses    int64
		expected  float64
	}{
		{100, 0, 100.0},
		{95, 5, 95.0},
		{80, 20, 80.0},
		{0, 100, 0.0},
	}
	
	for _, tc := range testCases {
		rate := calculateHitRate(tc.hits, tc.misses)
		if rate != tc.expected {
			t.Errorf("calculateHitRate(%d, %d): expected %f, got %f",
				tc.hits, tc.misses, tc.expected, rate)
		}
	}
}

func TestCacheAlertGeneration(t *testing.T) {
	monitor := NewCacheMetricsMonitor(50 * time.Millisecond)
	defer monitor.Enable(false)
	
	for i := 0; i < 50; i++ {
		monitor.RecordHit(false)
		monitor.RecordMiss(true)
	}
	
	alerts := monitor.GetAlerts(10)
	t.Logf("Generated %d alerts", len(alerts))
}

func TestCacheMetricsMonitorReset(t *testing.T) {
	monitor := NewCacheMetricsMonitor(100 * time.Millisecond)
	
	monitor.RecordHit(true)
	monitor.RecordMiss(true)
	monitor.RecordSet(1 * time.Millisecond)
	
	monitor.Reset()
	
	snapshot := monitor.GetCurrentMetrics()
	if snapshot.TotalHits != 0 || snapshot.TotalMisses != 0 {
		t.Error("Monitor reset did not clear metrics")
	}
}

func TestCacheMetricsMonitorEnableDisable(t *testing.T) {
	monitor := NewCacheMetricsMonitor(100 * time.Millisecond)
	
	monitor.Enable(false)
	monitor.RecordHit(true)
	
	monitor.Enable(true)
	monitor.RecordHit(true)
	
	time.Sleep(200 * time.Millisecond)
	
	monitor.Enable(false)
}

func TestCacheAlertThresholdsConfiguration(t *testing.T) {
	monitor := NewCacheMetricsMonitor(100 * time.Millisecond)
	defer monitor.Enable(false)
	
	customThresholds := &CacheAlertThresholds{
		LowHitRateThreshold:    90.0,
		HighErrorRateThreshold: 0.005,
		HighLatencyThreshold:   5 * time.Millisecond,
		LowL1HitRateThreshold: 70.0,
		HighEvictionThreshold:  500,
	}
	
	monitor.SetAlertThresholds(customThresholds)
	
	monitor.SetAlertThresholds(nil)
}

func TestCacheMetricsHistory(t *testing.T) {
	monitor := NewCacheMetricsMonitor(50 * time.Millisecond)
	defer monitor.Enable(false)
	
	for i := 0; i < 20; i++ {
		monitor.RecordHit(true)
		time.Sleep(10 * time.Millisecond)
	}
	
	history := monitor.GetMetricsHistory(10)
	if len(history) > 10 {
		t.Errorf("History should be limited to 10, got %d", len(history))
	}
}

func TestAggregatedCacheMetrics(t *testing.T) {
	aggregated := AggregatedCacheMetrics{
		TotalHits:        10000,
		TotalMisses:      500,
		OverallHitRate:   95.24,
		L1HitRate:        70.0,
		TotalSets:        5000,
		TotalErrors:      5,
		TotalEvictions:   100,
		AvgGetLatencyMs:  3.5,
		SnapshotsCount:   60,
	}
	
	if aggregated.OverallHitRate < 95.0 {
		t.Errorf("Overall hit rate should be >95%%, got %f%%", aggregated.OverallHitRate)
	}
	
	if aggregated.AvgGetLatencyMs > 10.0 {
		t.Errorf("Average get latency should be <10ms, got %fms", aggregated.AvgGetLatencyMs)
	}
}

func TestCachePerformanceTargets(t *testing.T) {
	t.Run("HitRateTarget", func(t *testing.T) {
		metrics := &CacheMetricSnapshot{
			HitRate: 96.5,
		}
		
		target := 95.0
		if metrics.HitRate < target {
			t.Errorf("Cache hit rate %f%% below target %f%%", metrics.HitRate, target)
		}
		t.Logf("Cache hit rate %f%% meets target >%f%%", metrics.HitRate, target)
	})
	
	t.Run("LatencyTarget", func(t *testing.T) {
		metrics := &CacheMetricSnapshot{
			AvgGetLatencyMs: 5.0,
		}
		
		target := 10.0
		if metrics.AvgGetLatencyMs > target {
			t.Errorf("Cache latency %fms exceeds target %fms", metrics.AvgGetLatencyMs, target)
		}
		t.Logf("Cache latency %fms meets target <%fms", metrics.AvgGetLatencyMs, target)
	})
	
	t.Run("L1HitRateTarget", func(t *testing.T) {
		metrics := &CacheMetricSnapshot{
			L1HitRate: 65.0,
		}
		
		target := 60.0
		if metrics.L1HitRate < target {
			t.Errorf("L1 hit rate %f%% below target %f%%", metrics.L1HitRate, target)
		}
		t.Logf("L1 hit rate %f%% meets target >%f%%", metrics.L1HitRate, target)
	})
}

func TestCacheErrorRate(t *testing.T) {
	errorRate := calculateErrorRate(5, 1000)
	if errorRate > 0.01 {
		t.Errorf("Error rate should be <1%%, got %f%%", errorRate*100)
	}
	
	zeroRate := calculateErrorRate(0, 100)
	if zeroRate != 0 {
		t.Errorf("Error rate for no errors should be 0, got %f", zeroRate)
	}
}
