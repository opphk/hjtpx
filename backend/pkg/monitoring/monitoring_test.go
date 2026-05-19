package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

func TestOpenTelemetryProviderCreation(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:     true,
		Endpoint:    "localhost:4317",
		ServiceName: "test-service",
		SamplingRate: 1.0,
	}

	provider, err := NewOpenTelemetryProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider == nil {
		t.Fatal("Expected provider to be created")
	}

	if !provider.IsEnabled() {
		t.Error("Expected provider to be enabled")
	}

	provider.End()
}

func TestOpenTelemetryProviderDisabled(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:     false,
		Endpoint:    "localhost:4317",
		ServiceName: "test-service",
	}

	provider, err := NewOpenTelemetryProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider.IsEnabled() {
		t.Error("Expected provider to be disabled")
	}
}

func TestOpenTelemetryStartSpan(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}

	provider, _ := NewOpenTelemetryProvider(cfg)

	ctx := context.Background()
	ctx, span := provider.StartSpan(ctx, "test-span")

	if span == nil {
		t.Error("Expected span to be non-nil")
	}

	span.End()
}

func TestDistributedTracer(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}

	provider, _ := NewOpenTelemetryProvider(cfg)
	tracer := NewDistributedTracer(provider)

	ctx := context.Background()
	carrier := make(map[string]string)

	tracer.InjectTraceContext(ctx, carrier)

	traceID := tracer.ExtractTraceID(ctx)
	if traceID != "" {
		t.Error("Expected empty trace ID for disabled provider")
	}

	newCtx := tracer.ExtractTraceContext(ctx, carrier)
	if newCtx == nil {
		t.Error("Expected non-nil context")
	}
}

func TestSpanCollector(t *testing.T) {
	collector := NewSpanCollector(100)

	if collector == nil {
		t.Fatal("Expected collector to be created")
	}

	spans := collector.GetSpans()
	if len(spans) != 0 {
		t.Errorf("Expected 0 spans initially, got %d", len(spans))
	}

	collector.AddSpan(&TraceSpan{
		Span:      nil,
		StartTime: time.Now(),
		Attrs:     make(map[string]interface{}),
	})

	spans = collector.GetSpans()
	if len(spans) != 1 {
		t.Errorf("Expected 1 span, got %d", len(spans))
	}

	collector.Clear()
	spans = collector.GetSpans()
	if len(spans) != 0 {
		t.Errorf("Expected 0 spans after clear, got %d", len(spans))
	}
}

func TestPerformanceProfiler(t *testing.T) {
	profiler := NewPerformanceProfiler(true)

	if profiler == nil {
		t.Fatal("Expected profiler to be created")
	}

	profiler.Start()

	profiler.RecordCPUProfile([]byte("test"), time.Second)

	profiles := profiler.GetCPUProfiles()
	if len(profiles) != 1 {
		t.Errorf("Expected 1 CPU profile, got %d", len(profiles))
	}

	profiler.RecordMemProfile()

	memProfiles := profiler.GetMemProfiles()
	if len(memProfiles) != 1 {
		t.Errorf("Expected 1 mem profile, got %d", len(memProfiles))
	}

	profiler.SetGoroutineCount(10)
	if profiler.GetGoroutineCount() != 10 {
		t.Errorf("Expected goroutine count 10, got %d", profiler.GetGoroutineCount())
	}

	profiler.Stop()
}

func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector(time.Minute)

	if collector == nil {
		t.Fatal("Expected collector to be created")
	}

	collector.RecordMetric("test_metric", 100.0, MetricTypeGauge, nil)

	metric, ok := collector.GetMetric("test_metric", nil)
	if !ok {
		t.Error("Expected metric to be found")
	}

	if metric.Value != 100.0 {
		t.Errorf("Expected value 100.0, got %f", metric.Value)
	}

	allMetrics := collector.GetAllMetrics()
	if len(allMetrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(allMetrics))
	}

	collector.Clear()
	allMetrics = collector.GetAllMetrics()
	if len(allMetrics) != 0 {
		t.Errorf("Expected 0 metrics after clear, got %d", len(allMetrics))
	}
}

func TestLatencyRecorder(t *testing.T) {
	recorder := NewLatencyRecorder(100)

	if recorder == nil {
		t.Fatal("Expected recorder to be created")
	}

	recorder.Record(10.0)
	recorder.Record(20.0)
	recorder.Record(30.0)
	recorder.Record(40.0)
	recorder.Record(50.0)

	if recorder.GetCount() != 5 {
		t.Errorf("Expected count 5, got %d", recorder.GetCount())
	}

	avg := recorder.GetAverage()
	if avg != 30.0 {
		t.Errorf("Expected average 30.0, got %f", avg)
	}

	min := recorder.GetMin()
	if min != 10.0 {
		t.Errorf("Expected min 10.0, got %f", min)
	}

	max := recorder.GetMax()
	if max != 50.0 {
		t.Errorf("Expected max 50.0, got %f", max)
	}

	p50 := recorder.GetPercentile(50)
	if p50 != 30.0 {
		t.Errorf("Expected P50 30.0, got %f", p50)
	}

	p95 := recorder.GetPercentile(95)
	if p95 != 50.0 {
		t.Errorf("Expected P95 50.0, got %f", p95)
	}
}

func TestLogAggregator(t *testing.T) {
	cfg := &config.LogAggregationConfig{
		Enabled:   true,
		Provider:  "mock",
		Endpoints: []string{"http://localhost:3100"},
		Aggregation: config.AggregationPolicy{
			ByService:      true,
			BySeverity:     true,
			ByTimeWindow:   60,
			MaxBatchSize:  100,
			FlushInterval:  5,
		},
	}

	aggregator, err := NewLogAggregator(cfg)
	if err != nil {
		t.Fatalf("Failed to create aggregator: %v", err)
	}

	if !aggregator.enabled {
		t.Error("Expected aggregator to be enabled")
	}

	aggregator.Start()

	aggregator.LogInfo("test-service", "test message", nil)
	aggregator.LogError("test-service", "error message", nil, nil)
	aggregator.LogWarn("test-service", "warn message", nil)
	aggregator.LogDebug("test-service", "debug message", nil)

	aggregator.Stop()
}

func TestLogAggregatorDisabled(t *testing.T) {
	cfg := &config.LogAggregationConfig{
		Enabled: false,
	}

	aggregator, err := NewLogAggregator(cfg)
	if err != nil {
		t.Fatalf("Failed to create aggregator: %v", err)
	}

	if aggregator.enabled {
		t.Error("Expected aggregator to be disabled")
	}

	aggregator.LogInfo("test-service", "test message", nil)
}

func TestAlertAggregator(t *testing.T) {
	cfg := &config.AlertAggregationConfig{
		Enabled:     true,
		Provider:    "alertmanager",
		GroupBy:     []string{"service", "severity"},
		TimeWindow:  300,
		Threshold:   10,
		Deduplication: config.DeduplicationConfig{
			Enabled:    true,
			WindowSecs: 300,
			MaxCount:   100,
		},
	}

	aggregator, err := NewAlertAggregator(cfg)
	if err != nil {
		t.Fatalf("Failed to create aggregator: %v", err)
	}

	if !aggregator.enabled {
		t.Error("Expected aggregator to be enabled")
	}

	aggregator.Start()

	alert := &AggregatedAlert{
		ID:          "alert-1",
		Name:        "HighCPU",
		GroupKey:    "test-service:critical",
		Severity:    "critical",
		Service:     "test-service",
		Description: "High CPU usage",
		Labels:      map[string]string{},
	}

	aggregator.ProcessAlert(alert)

	alerts := aggregator.GetAlerts()
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}

	retrievedAlert, ok := aggregator.GetAlertByGroupKey("test-service:critical")
	if !ok {
		t.Error("Expected alert to be found")
	}

	if retrievedAlert.Count != 1 {
		t.Errorf("Expected count 1, got %d", retrievedAlert.Count)
	}

	aggregator.ProcessAlert(alert)
	aggregator.ProcessAlert(alert)

	retrievedAlert, _ = aggregator.GetAlertByGroupKey("test-service:critical")
	if retrievedAlert.Count != 3 {
		t.Errorf("Expected count 3, got %d", retrievedAlert.Count)
	}

	aggregator.ResolveAlert("test-service:critical")

	retrievedAlert, ok = aggregator.GetAlertByGroupKey("test-service:critical")
	if ok {
		t.Error("Expected alert to be resolved")
	}

	aggregator.Stop()
}

func TestAlertAggregatorDisabled(t *testing.T) {
	cfg := &config.AlertAggregationConfig{
		Enabled: false,
	}

	aggregator, err := NewAlertAggregator(cfg)
	if err != nil {
		t.Fatalf("Failed to create aggregator: %v", err)
	}

	if aggregator.enabled {
		t.Error("Expected aggregator to be disabled")
	}
}

func TestAlertAggregatorCleanup(t *testing.T) {
	cfg := &config.AlertAggregationConfig{
		Enabled:     true,
		Provider:    "alertmanager",
		GroupBy:     []string{"service"},
		TimeWindow:  1,
		Threshold:   10,
		Deduplication: config.DeduplicationConfig{
			Enabled:    true,
			WindowSecs: 1,
			MaxCount:   100,
		},
	}

	aggregator, _ := NewAlertAggregator(cfg)
	aggregator.Start()

	alert := &AggregatedAlert{
		ID:       "alert-1",
		Name:     "TestAlert",
		GroupKey: "test-service",
		Service:  "test-service",
		Labels:   map[string]string{},
	}

	aggregator.ProcessAlert(alert)

	time.Sleep(1500 * time.Millisecond)

	aggregator.cleanup()

	alerts := aggregator.GetAlerts()
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts after cleanup, got %d", len(alerts))
	}

	aggregator.Stop()
}

func BenchmarkLatencyRecorderRecord(b *testing.B) {
	recorder := NewLatencyRecorder(100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder.Record(float64(i))
	}
}

func BenchmarkLatencyRecorderGetPercentile(b *testing.B) {
	recorder := NewLatencyRecorder(100000)

	for i := 0; i < 100000; i++ {
		recorder.Record(float64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder.GetPercentile(95)
	}
}

func BenchmarkMetricsCollectorRecord(b *testing.B) {
	collector := NewMetricsCollector(time.Minute)
	labels := map[string]string{"service": "test", "env": "prod"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordMetric("metric", float64(i), MetricTypeGauge, labels)
	}
}
