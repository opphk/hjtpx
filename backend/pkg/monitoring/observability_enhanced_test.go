package monitoring

import (
	"context"
	"testing"
	"time"
)

func TestEnhancedObservability_New(t *testing.T) {
	cfg := &ObservabilityConfig{
		MetricsEnabled:       true,
		TracingEnabled:      true,
		LoggingEnabled:      true,
		HealthCheckEnabled:   true,
		AlertEnabled:        true,
		MetricsConfig: &MetricsSettings{
			MetricsPort:        9090,
			MetricsPath:        "/metrics",
			CollectionInterval:  15 * time.Second,
			AggregationWindow:  1 * time.Minute,
			Quantiles:          []float64{0.5, 0.75, 0.95, 0.99},
		},
		TracingConfig: &TracingSettings{
			Provider:     "jaeger",
			Endpoint:    "http://localhost:14268/api/traces",
			SamplingRate: 0.1,
		},
		LoggingConfig: &LoggingSettings{
			Format:     "json",
			Level:      "info",
			Encoding:   "json",
			OutputPaths: []string{"stdout"},
		},
		HealthCheckConfig: &HealthSettings{
			Enabled:  true,
			Period:   10 * time.Second,
			Timeout:  5 * time.Second,
		},
		AlertConfig: &AlertSettings{
			Provider:          "prometheus",
			AggregationWindow: 5 * time.Minute,
		},
	}

	obs, err := NewEnhancedObservability(cfg)
	if err != nil {
		t.Fatalf("failed to create observability: %v", err)
	}

	if obs == nil {
		t.Fatal("observability should not be nil")
	}

	if obs.metrics == nil {
		t.Error("metrics should not be nil")
	}

	if obs.tracing == nil {
		t.Error("tracing should not be nil")
	}

	if obs.logging == nil {
		t.Error("logging should not be nil")
	}

	if obs.dashboards == nil {
		t.Error("dashboards should not be nil")
	}

	if obs.alerts == nil {
		t.Error("alerts should not be nil")
	}

	if obs.healthChecks == nil {
		t.Error("healthChecks should not be nil")
	}

	obs.Stop()
}

func TestMetricsAggregator_Counter(t *testing.T) {
	cfg := &MetricsSettings{
		MetricsPort:        9090,
		CollectionInterval: 15 * time.Second,
	}

	agg := NewMetricsAggregator(cfg)

	counter := agg.NewCounter("test_counter", map[string]string{"label": "value"})
	counter.Inc()

	if counter.Value() != 1 {
		t.Errorf("expected counter value 1, got %d", counter.Value())
	}

	counter.Add(5)
	if counter.Value() != 6 {
		t.Errorf("expected counter value 6, got %d", counter.Value())
	}

	counter2 := agg.NewCounter("test_counter", map[string]string{"label": "value"})
	if counter2.Value() != 6 {
		t.Errorf("expected counter value 6, got %d", counter2.Value())
	}
}

func TestMetricsAggregator_Gauge(t *testing.T) {
	cfg := &MetricsSettings{
		MetricsPort:        9090,
		CollectionInterval: 15 * time.Second,
	}

	agg := NewMetricsAggregator(cfg)

	gauge := agg.NewGauge("test_gauge", map[string]string{"label": "value"})
	gauge.Set(42.5)

	if gauge.Value() != 42.5 {
		t.Errorf("expected gauge value 42.5, got %f", gauge.Value())
	}

	gauge.Add(10.5)
	if gauge.Value() != 53.0 {
		t.Errorf("expected gauge value 53.0, got %f", gauge.Value())
	}
}

func TestMetricsAggregator_Histogram(t *testing.T) {
	cfg := &MetricsSettings{
		MetricsPort:        9090,
		CollectionInterval: 15 * time.Second,
	}

	agg := NewMetricsAggregator(cfg)

	histogram := agg.NewHistogram("test_histogram", map[string]string{"label": "value"}, []float64{0.1, 0.5, 1.0, 5.0})

	histogram.Observe(0.05)
	histogram.Observe(0.3)
	histogram.Observe(0.8)
	histogram.Observe(3.0)
	histogram.Observe(10.0)

	if histogram.Count() != 5 {
		t.Errorf("expected histogram count 5, got %d", histogram.Count())
	}

	if histogram.Sum() != 14.15 {
		t.Errorf("expected histogram sum 14.15, got %f", histogram.Sum())
	}
}

func TestDistributedTracer_TraceSampler(t *testing.T) {
	cfg := &TracingSettings{
		Provider:     "jaeger",
		SamplingRate: 1.0,
	}

	tracer := NewDistributedTracerEnhanced(cfg)

	sampler := NewTraceSampler(1.0)

	for i := 0; i < 100; i++ {
		if !sampler.ShouldSample() {
			t.Error("expected all samples to be taken when rate is 1.0")
			break
		}
	}

	sampler.SetSamplingRate(0.0)

	for i := 0; i < 100; i++ {
		if sampler.ShouldSample() {
			t.Error("expected no samples when rate is 0.0")
			break
		}
	}
}

func TestDistributedTracer_SpanCreation(t *testing.T) {
	cfg := &TracingSettings{
		Provider:     "jaeger",
		SamplingRate: 1.0,
	}

	tracer := NewDistributedTracerEnhanced(cfg)
	ctx := context.Background()

	ctx, span := tracer.StartSpan(ctx, "test_operation")
	if span == nil {
		t.Error("span should not be nil")
	}

	span.SetName("updated_operation")
	span.SetAttributes(
		Attribute{Key: "key1", Value: "value1"},
		Attribute{Key: "key2", Value: 123},
	)
	span.AddEvent("test_event")
	span.End()
}

func TestStructuredLogger(t *testing.T) {
	cfg := &LoggingSettings{
		Format:     "json",
		Level:      "info",
		Encoding:   "json",
		OutputPaths: []string{"stdout"},
	}

	logger := NewStructuredLogger(cfg)
	if logger == nil {
		t.Error("logger should not be nil")
	}

	if logger.config.Format != "json" {
		t.Errorf("expected format json, got %s", logger.config.Format)
	}

	if logger.config.Level != "info" {
		t.Errorf("expected level info, got %s", logger.config.Level)
	}
}

func TestLogSampler(t *testing.T) {
	sampler := NewLogSampler(100, 1)

	count := 0
	for i := 0; i < 150; i++ {
		if sampler.ShouldSample() {
			count++
		}
	}

	if count != 150 {
		t.Errorf("expected 150 samples with initial=100, thereafter=1, got %d", count)
	}
}

func TestDashboardManager(t *testing.T) {
	manager := NewDashboardManager()

	dashboard := &Dashboard{
		ID:          "test-dashboard",
		Name:        "Test Dashboard",
		Description: "A test dashboard",
		Type:        "metrics",
		Widgets: []Widget{
			{
				ID:   "widget-1",
				Type: "graph",
				Title: "CPU Usage",
				Query: "rate(cpu_usage[5m])",
				Position: WidgetPosition{
					X: 0,
					Y: 0,
				},
				Size: WidgetSize{
					Width:  4,
					Height: 3,
				},
			},
		},
		Variables: []Variable{
			{
				Name:        "env",
				Label:       "Environment",
				Type:        "query",
				MultiSelect: false,
				IncludeAll:  false,
			},
		},
		RefreshRate: 30 * time.Second,
		TimeRange: TimeRange{
			From: time.Now().Add(-1 * time.Hour),
			To:   time.Now(),
		},
		Layout: DashboardLayout{
			Columns:  12,
			RowHeight: 100,
		},
	}

	err := manager.CreateDashboard(dashboard)
	if err != nil {
		t.Fatalf("failed to create dashboard: %v", err)
	}

	retrieved, exists := manager.GetDashboard("test-dashboard")
	if !exists {
		t.Fatal("dashboard should exist")
	}

	if retrieved.Name != "Test Dashboard" {
		t.Errorf("expected name 'Test Dashboard', got '%s'", retrieved.Name)
	}

	if len(retrieved.Widgets) != 1 {
		t.Errorf("expected 1 widget, got %d", len(retrieved.Widgets))
	}
}

func TestAlertManager(t *testing.T) {
	cfg := &AlertSettings{
		Provider:          "prometheus",
		AggregationWindow: 5 * time.Minute,
		SlackEnabled:      true,
		SlackWebhookURL:  "https://hooks.slack.com/test",
	}

	manager := NewAlertManager(cfg)

	alert := &Alert{
		ID:          "alert-1",
		Name:        "High CPU Usage",
		Description: "CPU usage above threshold",
		Severity:    "warning",
		Status:      "firing",
		Fingerprint: "abc123",
		Labels: map[string]string{
			"service": "api",
		},
		Annotations: map[string]string{
			"summary": "High CPU usage detected",
		},
		StartsAt: time.Now(),
	}

	err := manager.CreateAlert(alert)
	if err != nil {
		t.Fatalf("failed to create alert: %v", err)
	}

	retrieved, exists := manager.GetAlert("alert-1")
	if !exists {
		t.Fatal("alert should exist")
	}

	if retrieved.Name != "High CPU Usage" {
		t.Errorf("expected name 'High CPU Usage', got '%s'", retrieved.Name)
	}

	if retrieved.Severity != "warning" {
		t.Errorf("expected severity 'warning', got '%s'", retrieved.Severity)
	}
}

func TestHealthCheckRegistry(t *testing.T) {
	cfg := &HealthSettings{
		Enabled:  true,
		Period:   10 * time.Second,
		Timeout:  5 * time.Second,
	}

	registry := NewHealthCheckRegistry(cfg)

	testCheck := &testHealthCheck{name: "test_check"}
	err := registry.Register(testCheck)
	if err != nil {
		t.Fatalf("failed to register health check: %v", err)
	}

	if len(registry.checks) != 1 {
		t.Errorf("expected 1 health check, got %d", len(registry.checks))
	}

	registry.Unregister("test_check")
	if len(registry.checks) != 0 {
		t.Errorf("expected 0 health checks after unregister, got %d", len(registry.checks))
	}
}

type testHealthCheck struct {
	name string
}

func (c *testHealthCheck) Check(ctx context.Context) error {
	return nil
}

func (c *testHealthCheck) Name() string {
	return c.name
}

func TestSpanCache(t *testing.T) {
	cache := NewSpanCache(10)

	span := &CachedSpan{
		SpanID:    "span-1",
		TraceID:   "trace-1",
		StartTime: time.Now(),
		Operation: "test_op",
		Tags:      make(map[string]string),
	}

	cache.Add(span)

	retrieved, exists := cache.Get("span-1")
	if !exists {
		t.Fatal("span should exist in cache")
	}

	if retrieved.Operation != "test_op" {
		t.Errorf("expected operation 'test_op', got '%s'", retrieved.Operation)
	}

	_, exists = cache.Get("non-existent")
	if exists {
		t.Error("non-existent span should not be found")
	}
}

func TestExportMetricsJSON(t *testing.T) {
	cfg := &MetricsSettings{
		MetricsPort:        9090,
		CollectionInterval: 15 * time.Second,
	}

	agg := NewMetricsAggregator(cfg)

	counter := agg.NewCounter("test_counter", map[string]string{"label": "value"})
	counter.Inc()
	counter.Inc()

	gauge := agg.NewGauge("test_gauge", map[string]string{"label": "value"})
	gauge.Set(42.0)

	histogram := agg.NewHistogram("test_histogram", map[string]string{"label": "value"}, []float64{0.1, 0.5, 1.0})
	histogram.Observe(0.5)

	json, err := ExportMetricsJSON(agg)
	if err != nil {
		t.Fatalf("failed to export metrics: %v", err)
	}

	if json == "" {
		t.Error("exported JSON should not be empty")
	}
}

func TestExportDashboardJSON(t *testing.T) {
	manager := NewDashboardManager()

	dashboard := &Dashboard{
		ID:   "test-dashboard",
		Name: "Test Dashboard",
		Type: "metrics",
	}

	manager.CreateDashboard(dashboard)

	json, err := ExportDashboardJSON(manager)
	if err != nil {
		t.Fatalf("failed to export dashboards: %v", err)
	}

	if json == "" {
		t.Error("exported JSON should not be empty")
	}
}

func TestExportAlertsJSON(t *testing.T) {
	cfg := &AlertSettings{
		Provider: "prometheus",
	}

	manager := NewAlertManager(cfg)

	alert := &Alert{
		ID:   "alert-1",
		Name: "Test Alert",
	}

	manager.CreateAlert(alert)

	json, err := ExportAlertsJSON(manager)
	if err != nil {
		t.Fatalf("failed to export alerts: %v", err)
	}

	if json == "" {
		t.Error("exported JSON should not be empty")
	}
}

func TestEnhancedObservability_RecordMetric(t *testing.T) {
	cfg := &ObservabilityConfig{
		MetricsEnabled: true,
		MetricsConfig: &MetricsSettings{
			MetricsPort:        9090,
			CollectionInterval: 15 * time.Second,
		},
	}

	obs, err := NewEnhancedObservability(cfg)
	if err != nil {
		t.Fatalf("failed to create observability: %v", err)
	}

	obs.RecordMetric("test_counter", 0, "counter", map[string]string{"label": "value"})
	obs.RecordMetric("test_gauge", 42.0, "gauge", map[string]string{"label": "value"})
	obs.RecordMetric("test_histogram", 0.5, "histogram", map[string]string{"label": "value"})

	obs.Stop()
}

func TestEnhancedObservability_StartSpan(t *testing.T) {
	cfg := &ObservabilityConfig{
		TracingEnabled: true,
		TracingConfig: &TracingSettings{
			Provider:     "jaeger",
			SamplingRate: 1.0,
		},
	}

	obs, err := NewEnhancedObservability(cfg)
	if err != nil {
		t.Fatalf("failed to create observability: %v", err)
	}

	ctx := context.Background()
	ctx, span := obs.StartSpan(ctx, "test_span",
		WithSpanKind(SpanKindServer),
		WithSpanTag("key", "value"),
	)

	if span == nil {
		t.Error("span should not be nil")
	}

	span.End()
	obs.Stop()
}

func TestEnhancedObservability_Log(t *testing.T) {
	cfg := &ObservabilityConfig{
		LoggingEnabled: true,
		LoggingConfig: &LoggingSettings{
			Format:     "json",
			Level:      "info",
			OutputPaths: []string{"stdout"},
		},
	}

	obs, err := NewEnhancedObservability(cfg)
	if err != nil {
		t.Fatalf("failed to create observability: %v", err)
	}

	obs.Log("info", "test message", map[string]interface{}{
		"key": "value",
	})

	obs.Stop()
}

func TestEnhancedObservability_GetHealthStatus(t *testing.T) {
	cfg := &ObservabilityConfig{
		HealthCheckEnabled: true,
		HealthCheckConfig: &HealthSettings{
			Enabled: true,
			Period: 10 * time.Second,
			Timeout: 5 * time.Second,
		},
	}

	obs, err := NewEnhancedObservability(cfg)
	if err != nil {
		t.Fatalf("failed to create observability: %v", err)
	}

	status := obs.GetHealthStatus()
	if status == nil {
		t.Error("health status should not be nil")
	}

	obs.Stop()
}

func TestWidgetConfig(t *testing.T) {
	widget := Widget{
		ID:    "widget-1",
		Type:  "graph",
		Title: "Test Widget",
		Query: "rate(cpu[5m])",
		Visualization: Visualization{
			Type: "line",
			Options: map[string]interface{}{
				"showPoints": true,
			},
			ColorScheme: []string{"blue", "red", "green"},
			Legend: &LegendConfig{
				Position:      "right",
				DisplayAsTable: true,
			},
			Tooltip: &TooltipConfig{
				Mode:   "single",
				SortBy: "value",
			},
			Axes: &AxesConfig{
				XAxis: &AxisConfig{
					Label: "Time",
					Scale: "time",
				},
				YAxis: &AxisConfig{
					Label: "CPU %",
					Min:   0,
					Max:   100,
					Scale: "linear",
				},
			},
		},
		Position: WidgetPosition{
			X:  0,
			Y:  0,
			Row: 1,
		},
		Size: WidgetSize{
			Width:  6,
			Height: 4,
		},
		Thresholds: []Threshold{
			{
				Value:    80,
				Color:    "red",
				Label:    "High",
				Severity: "critical",
			},
		},
		Links: []WidgetLink{
			{
				Title:  "View Details",
				URL:    "/metrics/details",
				NewTab: true,
			},
		},
	}

	if widget.ID != "widget-1" {
		t.Errorf("expected ID 'widget-1', got '%s'", widget.ID)
	}

	if len(widget.Thresholds) != 1 {
		t.Errorf("expected 1 threshold, got %d", len(widget.Thresholds))
	}

	if len(widget.Links) != 1 {
		t.Errorf("expected 1 link, got %d", len(widget.Links))
	}
}

func TestAlertRule(t *testing.T) {
	rule := AlertRule{
		ID:          "rule-1",
		Name:        "High Memory Alert",
		Query:       "memory_usage > 80",
		Duration:    5 * time.Minute,
		Severity:    "warning",
		For:         2 * time.Minute,
		Labels: map[string]string{
			"service": "api",
			"env":     "prod",
		},
		Annotations: map[string]string{
			"summary": "High memory usage",
		},
		LabelsMatchers: []LabelMatcher{
			{
				Name:  "service",
				Value: "api",
				Regex: false,
			},
		},
	}

	if rule.Name != "High Memory Alert" {
		t.Errorf("expected name 'High Memory Alert', got '%s'", rule.Name)
	}

	if rule.Severity != "warning" {
		t.Errorf("expected severity 'warning', got '%s'", rule.Severity)
	}

	if len(rule.LabelsMatchers) != 1 {
		t.Errorf("expected 1 label matcher, got %d", len(rule.LabelsMatchers))
	}
}

func TestAlertRoute(t *testing.T) {
	route := AlertRoute{
		Receiver: "slack-receiver",
		Match: map[string]string{
			"severity": "critical",
		},
		Continue: false,
		Routes: []*AlertRoute{
			{
				Receiver: "pagerduty-receiver",
				Match: map[string]string{
					"team": "oncall",
				},
				Continue: true,
			},
		},
	}

	if route.Receiver != "slack-receiver" {
		t.Errorf("expected receiver 'slack-receiver', got '%s'", route.Receiver)
	}

	if !route.Continue {
		t.Error("expected continue to be true")
	}

	if len(route.Routes) != 1 {
		t.Errorf("expected 1 nested route, got %d", len(route.Routes))
	}
}

func TestTraceSpan(t *testing.T) {
	span := &TraceSpan{
		SpanID:    "span-123",
		TraceID:   "trace-456",
		StartTime: time.Now(),
		Operation: "test_operation",
		Service:   "test-service",
		Status:    "ok",
		Tags:      make(map[string]string),
		Logs:      make([]SpanLog, 0),
	}

	span.SetName("updated_operation")
	span.SetAttributes(
		Attribute{Key: "user.id", Value: "12345"},
		Attribute{Key: "http.status_code", Value: 200},
	)
	span.AddEvent("test_event")
	span.End()

	if span.Operation != "updated_operation" {
		t.Errorf("expected operation 'updated_operation', got '%s'", span.Operation)
	}

	if span.EndTime.IsZero() {
		t.Error("end time should be set after End()")
	}

	if len(span.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(span.Tags))
	}

	if len(span.Logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(span.Logs))
	}
}

func TestMetricsAggregator_MultipleLabels(t *testing.T) {
	cfg := &MetricsSettings{
		MetricsPort:        9090,
		CollectionInterval: 15 * time.Second,
	}

	agg := NewMetricsAggregator(cfg)

	counter1 := agg.NewCounter("requests_total", map[string]string{"method": "GET", "path": "/api/users"})
	counter1.Inc()

	counter2 := agg.NewCounter("requests_total", map[string]string{"method": "POST", "path": "/api/users"})
	counter2.Inc()

	if counter1.Value() != 1 {
		t.Errorf("expected counter1 value 1, got %d", counter1.Value())
	}

	if counter2.Value() != 1 {
		t.Errorf("expected counter2 value 1, got %d", counter2.Value())
	}
}

func TestNewReconciler(t *testing.T) {
	op := &HjtpxOperator{}

	if op == nil {
		t.Error("operator should not be nil")
	}
}
