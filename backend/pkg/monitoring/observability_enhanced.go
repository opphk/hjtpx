package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

type EnhancedObservability struct {
	config        *ObservabilityConfig
	metrics       *MetricsAggregator
	tracing       *DistributedTracer
	logging       *StructuredLogger
	dashboards    *DashboardManager
	alerts        *AlertManager
	healthChecks  *HealthCheckRegistry
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

type ObservabilityConfig struct {
	MetricsEnabled   bool              `json:"metricsEnabled"`
	TracingEnabled  bool              `json:"tracingEnabled"`
	LoggingEnabled  bool              `json:"loggingEnabled"`
	HealthCheckEnabled bool            `json:"healthCheckEnabled"`
	AlertEnabled    bool              `json:"alertEnabled"`
	MetricsConfig   *MetricsSettings   `json:"metricsConfig,omitempty"`
	TracingConfig   *TracingSettings   `json:"tracingConfig,omitempty"`
	LoggingConfig   *LoggingSettings   `json:"loggingConfig,omitempty"`
	HealthCheckConfig *HealthSettings  `json:"healthCheckConfig,omitempty"`
	AlertConfig     *AlertSettings     `json:"alertConfig,omitempty"`
}

type MetricsSettings struct {
	PrometheusEnabled   bool     `json:"prometheusEnabled"`
	GraphiteEnabled     bool     `json:"graphiteEnabled"`
	DataDogEnabled      bool     `json:"datadogEnabled"`
	StatsDEnabled        bool     `json:"statsdEnabled"`
	MetricsPort          int      `json:"metricsPort"`
	MetricsPath          string   `json:"metricsPath"`
	CollectionInterval   time.Duration `json:"collectionInterval"`
	RetentionPeriod      time.Duration `json:"retentionPeriod"`
	AggregationWindow    time.Duration `json:"aggregationWindow"`
	Quantiles            []float64 `json:"quantiles"`
	AdditionalLabels     map[string]string `json:"additionalLabels"`
}

type TracingSettings struct {
	Provider              string  `json:"provider"`
	JaegerEnabled         bool    `json:"jaegerEnabled"`
	ZipkinEnabled         bool    `json:"zipkinEnabled"`
	LightStepEnabled      bool    `json:"lightstepEnabled"`
	OTLPEnabled           bool    `json:"otlpEnabled"`
	Endpoint              string  `json:"endpoint"`
	SamplingRate          float64 `json:"samplingRate"`
	MaxTracesPerSecond    int     `json:"maxTracesPerSecond"`
	MaxTraceAge           time.Duration `json:"maxTraceAge"`
	BackendBufferSize     int     `json:"backendBufferSize"`
	PropagationFormat     string  `json:"propagationFormat"`
}

type LoggingSettings struct {
	Format              string            `json:"format"`
	Level               string            `json:"level"`
	OutputPaths         []string          `json:"outputPaths"`
	Encoding            string            `json:"encoding"`
	FileRotation        *RotationConfig   `json:"fileRotation,omitempty"`
	FluentdEnabled      bool              `json:"fluentdEnabled"`
	FluentdHost         string            `json:"fluentdHost,omitempty"`
	FluentdPort         int               `json:"fluentdPort,omitempty"`
	SamplingConfig      *SamplingConfig   `json:"sampling,omitempty"`
	StructuredFields     []string          `json:"structuredFields"`
}

type RotationConfig struct {
	MaxSize    int64         `json:"maxSize"`
	MaxBackups int           `json:"maxBackups"`
	MaxAge     int           `json:"maxAge"`
	Compress   bool          `json:"compress"`
}

type SamplingConfig struct {
	Initial    int     `json:"initial"`
	Thereafter int     `json:"thereafter"`
}

type HealthSettings struct {
	Enabled              bool          `json:"enabled"`
	ReadinessEnabled     bool          `json:"readinessEnabled"`
	LivenessEnabled      bool          `json:"livenessEnabled"`
	ReadinessPath        string        `json:"readinessPath"`
	LivenessPath         string        `json:"livenessPath"`
	InitialDelay         time.Duration `json:"initialDelay"`
	Period               time.Duration `json:"period"`
	Timeout              time.Duration `json:"timeout"`
	FailureThreshold     int           `json:"failureThreshold"`
	SuccessThreshold     int           `json:"successThreshold"`
}

type AlertSettings struct {
	Provider              string            `json:"provider"`
	SlackEnabled          bool              `json:"slackEnabled"`
	SlackWebhookURL       string            `json:"slackWebhookURL"`
	EmailEnabled          bool              `json:"emailEnabled"`
	EmailRecipients       []string          `json:"emailRecipients"`
	PagerDutyEnabled      bool              `json:"pagerdutyEnabled"`
	PagerDutyKey          string            `json:"pagerdutyKey,omitempty"`
	OpsGenieEnabled       bool              `json:"opsgenieEnabled"`
	OpsGenieKey           string            `json:"opsgenieKey,omitempty"`
	WebhookEnabled        bool              `json:"webhookEnabled"`
	WebhookURL            string            `json:"webhookURL,omitempty"`
	AggregationWindow     time.Duration     `json:"aggregationWindow"`
	SeverityConfig        *SeveritySettings `json:"severityConfig,omitempty"`
}

type SeveritySettings struct {
	CriticalThreshold  float64 `json:"criticalThreshold"`
	WarningThreshold   float64 `json:"warningThreshold"`
	InfoThreshold      float64 `json:"infoThreshold"`
	EscalationEnabled  bool    `json:"escalationEnabled"`
	EscalationDelay    time.Duration `json:"escalationDelay"`
}

type MetricsAggregator struct {
	counters     map[string]*Counter
	gauges       map[string]*Gauge
	histograms   map[string]*Histogram
	summaries    map[string]*Summary
	mu           sync.RWMutex
	config       *MetricsSettings
	timeWindow   time.Duration
}

type Counter struct {
	name   string
	labels map[string]string
	value  uint64
	mu     sync.RWMutex
}

type Gauge struct {
	name    string
	labels  map[string]string
	value   float64
	mu      sync.RWMutex
}

type Histogram struct {
	name     string
	labels   map[string]string
	buckets  map[float64]uint64
	count    uint64
	sum      float64
	mu       sync.RWMutex
}

type Summary struct {
	name      string
	labels    map[string]string
	count     uint64
	sum       float64
	quantiles map[float64]float64
	mu        sync.RWMutex
}

type DistributedTracer struct {
	config       *TracingSettings
	spanCache    *SpanCache
	propagators  map[string]Propagator
	mu           sync.RWMutex
	sampler      *TraceSampler
}

type SpanCache struct {
	spans      map[string]*CachedSpan
	maxSize    int
	mu         sync.RWMutex
	evictionPolicy string
}

type CachedSpan struct {
	SpanID       string
	TraceID      string
	ParentSpanID string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Operation    string
	Service      string
	Status       string
	Tags         map[string]string
	Logs         []SpanLog
	Children     []string
}

type SpanLog struct {
	Timestamp time.Time
	Fields    map[string]string
}

type Propagator interface {
	Inject(ctx context.Context, carrier map[string]string) error
	Extract(ctx context.Context, carrier map[string]string) error
}

type W3CTraceContextPropagator struct{}

type B3Propagator struct{}

type JaegerPropagator struct{}

type TraceSampler struct {
	samplingRate float64
	mu           sync.RWMutex
	random       *math.Rand
}

type StructuredLogger struct {
	config   *LoggingSettings
	outputs  []LoggerOutput
	mu       sync.RWMutex
	sampler  *LogSampler
}

type LoggerOutput interface {
	Write(entry *LogEntry) error
	Close() error
}

type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                  `json:"level"`
	Message     string                  `json:"message"`
	Service     string                  `json:"service"`
	Version     string                  `json:"version"`
	Environment string                  `json:"environment"`
	TraceID     string                  `json:"traceId,omitempty"`
	SpanID      string                  `json:"spanId,omitempty"`
	Logger      string                  `json:"logger"`
	Caller      string                  `json:"caller"`
	StackTrace  string                  `json:"stackTrace,omitempty"`
	Fields      map[string]interface{}  `json:"fields,omitempty"`
	Error       *LogError               `json:"error,omitempty"`
}

type LogError struct {
	Message    string `json:"message"`
	StackTrace string `json:"stackTrace"`
	Type       string `json:"type"`
}

type LogSampler struct {
	initial    int
	thereafter int
	count      int
	mu         sync.Mutex
}

type DashboardManager struct {
	dashboards map[string]*Dashboard
	queries   map[string]*QueryConfig
	mu        sync.RWMutex
}

type Dashboard struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        string         `json:"type"`
	Widgets     []Widget       `json:"widgets"`
	Variables   []Variable     `json:"variables"`
	RefreshRate time.Duration  `json:"refreshRate"`
	TimeRange   TimeRange      `json:"timeRange"`
	Filters     []Filter       `json:"filters"`
	Layout      DashboardLayout `json:"layout"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type Widget struct {
	ID          string       `json:"id"`
	Type        string       `json:"type"`
	Title       string       `json:"title"`
	Query       string       `json:"query"`
	Visualization Visualization `json:"visualization"`
	Position    WidgetPosition `json:"position"`
	Size        WidgetSize    `json:"size"`
	Thresholds  []Threshold   `json:"thresholds,omitempty"`
	Links       []WidgetLink  `json:"links,omitempty"`
}

type Visualization struct {
	Type        string            `json:"type"`
	Options     map[string]interface{} `json:"options,omitempty"`
	ColorScheme []string          `json:"colorScheme,omitempty"`
	Legend      *LegendConfig     `json:"legend,omitempty"`
	Tooltip     *TooltipConfig    `json:"tooltip,omitempty"`
	Axes        *AxesConfig       `json:"axes,omitempty"`
}

type LegendConfig struct {
	Position string   `json:"position"`
	DisplayAsTable bool `json:"displayAsTable"`
	ShowValues []string `json:"showValues,omitempty"`
}

type TooltipConfig struct {
	Mode     string `json:"mode"`
	SortBy   string `json:"sortBy"`
}

type AxesConfig struct {
	XAxis *AxisConfig `json:"xAxis,omitempty"`
	YAxis *AxisConfig `json:"yAxis,omitempty"`
}

type AxisConfig struct {
	Label   string  `json:"label"`
	Min     float64 `json:"min,omitempty"`
	Max     float64 `json:"max,omitempty"`
	Scale   string  `json:"scale,omitempty"`
}

type WidgetPosition struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Row    int `json:"row,omitempty"`
}

type WidgetSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Threshold struct {
	Value    float64 `json:"value"`
	Color    string  `json:"color"`
	Label    string  `json:"label"`
	Severity string  `json:"severity"`
}

type WidgetLink struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	NewTab   bool   `json:"newTab"`
}

type Variable struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Query       string   `json:"query,omitempty"`
	Options     []string `json:"options,omitempty"`
	Default     string   `json:"default,omitempty"`
	MultiSelect bool     `json:"multiSelect"`
	IncludeAll  bool     `json:"includeAll"`
}

type QueryConfig struct {
	ID          string   `json:"id"`
	Query       string   `json:"query"`
	DataSource  string   `json:"dataSource"`
	Interval    time.Duration `json:"interval"`
	LegendFormat string  `json:"legendFormat,omitempty"`
	Expression  string   `json:"expression,omitempty"`
}

type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type Filter struct {
	Name      string   `json:"name"`
	Operator  string   `json:"operator"`
	Values    []string `json:"values"`
}

type DashboardLayout struct {
	Columns  int `json:"columns"`
	RowHeight int `json:"rowHeight"`
}

type AlertManager struct {
	alerts      map[string]*Alert
	rules       map[string]*AlertRule
	routes      map[string]*AlertRoute
	receivers   map[string]*AlertReceiver
	notifiers   []AlertNotifier
	mu          sync.RWMutex
	config      *AlertSettings
	history     []AlertHistory
	suppressions map[string]*AlertSuppression
}

type Alert struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Severity    string           `json:"severity"`
	Status      string           `json:"status"`
	Fingerprint string           `json:"fingerprint"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time        `json:"startsAt"`
	EndsAt      *time.Time       `json:"endsAt,omitempty"`
	GeneratorURL string          `json:"generatorURL,omitempty"`
	RunbookURL  string           `json:"runbookURL,omitempty"`
}

type AlertRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Query       string            `json:"query"`
	Duration    time.Duration     `json:"duration"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Severity    string            `json:"severity"`
	For         time.Duration     `json:"for"`
	LabelsMatchers []LabelMatcher `json:"labelMatchers,omitempty"`
}

type LabelMatcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Regex   bool   `json:"regex"`
}

type AlertRoute struct {
	Receiver   string   `json:"receiver"`
	Match      map[string]string `json:"match,omitempty"`
	MatchRE    map[string]string `json:"matchRE,omitempty"`
	Continue   bool    `json:"continue"`
	Routes     []*AlertRoute `json:"routes,omitempty"`
}

type AlertReceiver struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Config   map[string]interface{} `json:"config"`
	Notifier AlertNotifier
}

type AlertNotifier interface {
	Notify(ctx context.Context, alert *Alert) error
}

type AlertHistory struct {
	AlertID   string    `json:"alertId"`
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

type AlertSuppression struct {
	AlertID    string        `json:"alertId"`
	SuppressedBy []string    `json:"suppressedBy"`
	Until      time.Time     `json:"until"`
	Reason     string        `json:"reason"`
}

type HealthCheckRegistry struct {
	checks    map[string]HealthCheck
	mu        sync.RWMutex
	config    *HealthSettings
	status    *HealthStatus
}

type HealthCheck interface {
	Check(ctx context.Context) error
	Name() string
}

type HealthStatus struct {
	Status        string                  `json:"status"`
	Uptime        time.Duration           `json:"uptime"`
	Version       string                  `json:"version"`
	Checks        map[string]CheckResult  `json:"checks"`
	LastCheck     time.Time               `json:"lastCheck"`
}

type CheckResult struct {
	Name      string        `json:"name"`
	Status    string        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Duration  time.Duration `json:"duration"`
	LastCheck time.Time     `json:"lastCheck"`
}

func NewEnhancedObservability(cfg *ObservabilityConfig) (*EnhancedObservability, error) {
	obs := &EnhancedObservability{
		config:       cfg,
		metrics:      NewMetricsAggregator(cfg.MetricsConfig),
		tracing:      NewDistributedTracerEnhanced(cfg.TracingConfig),
		logging:      NewStructuredLogger(cfg.LoggingConfig),
		dashboards:   NewDashboardManager(),
		alerts:       NewAlertManager(cfg.AlertConfig),
		healthChecks: NewHealthCheckRegistry(cfg.HealthCheckConfig),
		stopCh:       make(chan struct{}),
	}

	if cfg.MetricsEnabled {
		obs.startMetricsCollection()
	}

	if cfg.HealthCheckEnabled {
		obs.startHealthChecks()
	}

	return obs, nil
}

func NewMetricsAggregator(cfg *MetricsSettings) *MetricsAggregator {
	if cfg == nil {
		cfg = &MetricsSettings{
			MetricsPort:     9090,
			MetricsPath:     "/metrics",
			CollectionInterval: 15 * time.Second,
			Quantiles:       []float64{0.5, 0.75, 0.95, 0.99},
		}
	}

	return &MetricsAggregator{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
		summaries:  make(map[string]*Summary),
		config:     cfg,
		timeWindow: cfg.AggregationWindow,
	}
}

func (m *MetricsAggregator) NewCounter(name string, labels map[string]string) *Counter {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.makeKey(name, labels)
	if c, exists := m.counters[key]; exists {
		return c
	}

	c := &Counter{
		name:   name,
		labels: labels,
		value:  0,
	}
	m.counters[key] = c
	return c
}

func (m *MetricsAggregator) NewGauge(name string, labels map[string]string) *Gauge {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.makeKey(name, labels)
	if g, exists := m.gauges[key]; exists {
		return g
	}

	g := &Gauge{
		name:   name,
		labels: labels,
		value:  0,
	}
	m.gauges[key] = g
	return g
}

func (m *MetricsAggregator) NewHistogram(name string, labels map[string]string, buckets []float64) *Histogram {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.makeKey(name, labels)
	if h, exists := m.histograms[key]; exists {
		return h
	}

	bucketMap := make(map[float64]uint64)
	for _, b := range buckets {
		bucketMap[b] = 0
	}

	h := &Histogram{
		name:    name,
		labels:  labels,
		buckets: bucketMap,
	}
	m.histograms[key] = h
	return h
}

func (m *MetricsAggregator) makeKey(name string, labels map[string]string) string {
	return fmt.Sprintf("%s:%v", name, labels)
}

func (c *Counter) Inc() {
	atomic.AddUint64(&c.value, 1)
}

func (c *Counter) Add(v uint64) {
	atomic.AddUint64(&c.value, v)
}

func (c *Counter) Value() uint64 {
	return atomic.LoadUint64(&c.value)
}

func (g *Gauge) Set(v float64) {
	atomic.StoreFloat64(&g.value, v)
}

func (g *Gauge) Add(v float64) {
	for {
		old := atomic.LoadUint64(&g.value)
		new := math.Float64bits(math.Float64frombits(old) + v)
		if atomic.CompareAndSwapUint64((*uint64)(&g.value), old, new) {
			return
		}
	}
}

func (g *Gauge) Value() float64 {
	return atomic.LoadFloat64(&g.value)
}

func (h *Histogram) Observe(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	atomic.AddUint64(&h.count, 1)
	atomic.AddFloat64(&h.sum, v)

	for bucket := range h.buckets {
		if v <= bucket {
			h.buckets[bucket]++
		}
	}
}

func (h *Histogram) Count() uint64 {
	return atomic.LoadUint64(&h.count)
}

func (h *Histogram) Sum() float64 {
	return atomic.LoadFloat64(&h.sum)
}

func NewDistributedTracerEnhanced(cfg *TracingSettings) *DistributedTracer {
	if cfg == nil {
		cfg = &TracingSettings{
			Provider:     "jaeger",
			SamplingRate: 0.1,
			Endpoint:     "http://localhost:14268/api/traces",
		}
	}

	tracer := &DistributedTracer{
		config:      cfg,
		spanCache:   NewSpanCache(10000),
		propagators: make(map[string]Propagator),
		sampler:     NewTraceSampler(cfg.SamplingRate),
	}

	tracer.propagators["w3c"] = &W3CTraceContextPropagator{}
	tracer.propagators["b3"] = &B3Propagator{}
	tracer.propagators["jaeger"] = &JaegerPropagator{}

	return tracer
}

func NewSpanCache(maxSize int) *SpanCache {
	return &SpanCache{
		spans:          make(map[string]*CachedSpan),
		maxSize:        maxSize,
		evictionPolicy: "lru",
	}
}

func (c *SpanCache) Add(span *CachedSpan) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.spans) >= c.maxSize {
		c.evict()
	}

	c.spans[span.SpanID] = span
}

func (c *SpanCache) Get(spanID string) (*CachedSpan, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	span, exists := c.spans[spanID]
	return span, exists
}

func (c *SpanCache) evict() {
	for id := range c.spans {
		delete(c.spans, id)
		return
	}
}

func (p *W3CTraceContextPropagator) Inject(ctx context.Context, carrier map[string]string) error {
	return nil
}

func (p *W3CTraceContextPropagator) Extract(ctx context.Context, carrier map[string]string) error {
	return nil
}

func (p *B3Propagator) Inject(ctx context.Context, carrier map[string]string) error {
	return nil
}

func (p *B3Propagator) Extract(ctx context.Context, carrier map[string]string) error {
	return nil
}

func (p *JaegerPropagator) Inject(ctx context.Context, carrier map[string]string) error {
	return nil
}

func (p *JaegerPropagator) Extract(ctx context.Context, carrier map[string]string) error {
	return nil
}

func NewTraceSampler(rate float64) *TraceSampler {
	return &TraceSampler{
		samplingRate: rate,
		random:       math.NewRand(time.Now().UnixNano()),
	}
}

func (s *TraceSampler) ShouldSample() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.random.Float64() < s.samplingRate
}

func (s *TraceSampler) SetSamplingRate(rate float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.samplingRate = rate
}

func NewStructuredLogger(cfg *LoggingSettings) *StructuredLogger {
	if cfg == nil {
		cfg = &LoggingSettings{
			Format:     "json",
			Level:      "info",
			Encoding:   "json",
			OutputPaths: []string{"stdout"},
		}
	}

	return &StructuredLogger{
		config:  cfg,
		sampler: NewLogSampler(100, 1),
	}
}

func NewLogSampler(initial, thereafter int) *LogSampler {
	return &LogSampler{
		initial:    initial,
		thereafter: thereafter,
		count:      0,
	}
}

func (s *LogSampler) ShouldSample() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.count++
	if s.count <= s.initial {
		return true
	}

	return s.count%s.thereafter == 0
}

func NewDashboardManager() *DashboardManager {
	return &DashboardManager{
		dashboards: make(map[string]*Dashboard),
		queries:    make(map[string]*QueryConfig),
	}
}

func (d *DashboardManager) CreateDashboard(dashboard *Dashboard) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if dashboard.ID == "" {
		return fmt.Errorf("dashboard ID is required")
	}

	d.dashboards[dashboard.ID] = dashboard
	return nil
}

func (d *DashboardManager) GetDashboard(id string) (*Dashboard, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	dashboard, exists := d.dashboards[id]
	return dashboard, exists
}

func NewAlertManager(cfg *AlertSettings) *AlertManager {
	if cfg == nil {
		cfg = &AlertSettings{
			AggregationWindow: 5 * time.Minute,
		}
	}

	return &AlertManager{
		alerts:       make(map[string]*Alert),
		rules:        make(map[string]*AlertRule),
		routes:       make(map[string]*AlertRoute),
		receivers:   make(map[string]*AlertReceiver),
		config:      cfg,
		history:     make([]AlertHistory, 0),
		suppressions: make(map[string]*AlertSuppression),
	}
}

func (a *AlertManager) CreateAlert(alert *Alert) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if alert.ID == "" {
		return fmt.Errorf("alert ID is required")
	}

	a.alerts[alert.ID] = alert
	a.history = append(a.history, AlertHistory{
		AlertID:   alert.ID,
		Event:     "created",
		Timestamp: time.Now(),
	})

	return nil
}

func (a *AlertManager) GetAlert(id string) (*Alert, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	alert, exists := a.alerts[id]
	return alert, exists
}

func NewHealthCheckRegistry(cfg *HealthSettings) *HealthCheckRegistry {
	if cfg == nil {
		cfg = &HealthSettings{
			Enabled:     true,
			ReadinessEnabled: true,
			LivenessEnabled:  true,
			Period:      10 * time.Second,
			Timeout:     5 * time.Second,
		}
	}

	return &HealthCheckRegistry{
		checks: make(map[string]HealthCheck),
		config: cfg,
		status: &HealthStatus{
			Status:  "starting",
			Checks:  make(map[string]CheckResult),
		},
	}
}

func (h *HealthCheckRegistry) Register(check HealthCheck) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.checks[check.Name()] = check
	return nil
}

func (h *HealthCheckRegistry) Unregister(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.checks, name)
}

func (o *EnhancedObservability) startMetricsCollection() {
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		ticker := time.NewTicker(o.config.MetricsConfig.CollectionInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				o.collectMetrics()
			case <-o.stopCh:
				return
			}
		}
	}()
}

func (o *EnhancedObservability) collectMetrics() {
	o.metrics.mu.RLock()
	defer o.metrics.mu.RUnlock()

	for name, counter := range o.metrics.counters {
		_ = name
		_ = counter.Value()
	}
}

func (o *EnhancedObservability) startHealthChecks() {
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		ticker := time.NewTicker(o.config.HealthCheckConfig.Period)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				o.runHealthChecks()
			case <-o.stopCh:
				return
			}
		}
	}()
}

func (o *EnhancedObservability) runHealthChecks() {
	o.healthChecks.mu.RLock()
	defer o.healthChecks.mu.RUnlock()

	for name, check := range o.healthChecks.checks {
		ctx, cancel := context.WithTimeout(context.Background(), o.config.HealthCheckConfig.Timeout)
		err := check.Check(ctx)
		cancel()

		result := CheckResult{
			Name:     name,
			Status:   "healthy",
			Duration: time.Millisecond * 10,
			LastCheck: time.Now(),
		}

		if err != nil {
			result.Status = "unhealthy"
			result.Message = err.Error()
		}

		o.healthChecks.status.Checks[name] = result
	}
}

func (o *EnhancedObservability) Stop() {
	close(o.stopCh)
	o.wg.Wait()
}

func (o *EnhancedObservability) GetMetrics() *MetricsAggregator {
	return o.metrics
}

func (o *EnhancedObservability) GetTracing() *DistributedTracer {
	return o.tracing
}

func (o *EnhancedObservability) GetLogging() *StructuredLogger {
	return o.logging
}

func (o *EnhancedObservability) GetDashboards() *DashboardManager {
	return o.dashboards
}

func (o *EnhancedObservability) GetAlerts() *AlertManager {
	return o.alerts
}

func (o *EnhancedObservability) GetHealthStatus() *HealthStatus {
	return o.healthChecks.status
}

func (o *EnhancedObservability) RecordMetric(name string, value float64, metricType string, labels map[string]string) {
	switch metricType {
	case "counter":
		counter := o.metrics.NewCounter(name, labels)
		counter.Inc()
	case "gauge":
		gauge := o.metrics.NewGauge(name, labels)
		gauge.Set(value)
	case "histogram":
		histogram := o.metrics.NewHistogram(name, labels, []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10})
		histogram.Observe(value)
	}
}

func (o *EnhancedObservability) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	if !o.config.TracingEnabled {
		return ctx, &noopSpan{}
	}

	return o.tracing.StartSpan(ctx, name, opts...)
}

func (t *DistributedTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	t.mu.RLock()
	shouldSample := t.sampler.ShouldSample()
	t.mu.RUnlock()

	if !shouldSample {
		return ctx, &noopSpan{}
	}

	span := &TraceSpan{
		StartTime: time.Now(),
		Operation: name,
		Tags:      make(map[string]string),
		Logs:      make([]SpanLog, 0),
	}

	return ctx, span
}

type TraceSpan struct {
	SpanID      string
	TraceID     string
	ParentSpanID string
	StartTime   time.Time
	EndTime     time.Time
	Operation   string
	Service     string
	Status      string
	Tags        map[string]string
	Logs        []SpanLog
	mu          sync.Mutex
}

func (s *TraceSpan) End() {
	s.EndTime = time.Now()
}

func (s *TraceSpan) AddEvent(name string, opts ...EventOption) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log := SpanLog{
		Timestamp: time.Now(),
		Fields:    make(map[string]string),
	}
	log.Fields["event"] = name

	s.Logs = append(s.Logs, log)
}

func (s *TraceSpan) IsRecording() bool {
	return s.EndTime.IsZero()
}

func (s *TraceSpan) SpanContext() SpanContext {
	return SpanContext{
		TraceID: s.TraceID,
		SpanID:  s.SpanID,
	}
}

func (s *TraceSpan) SetStatus(status Status) {
	s.Status = status.Message
}

func (s *TraceSpan) SetName(name string) {
	s.Operation = name
}

func (s *TraceSpan) SetAttributes(attrs ...Attribute) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, attr := range attrs {
		s.Tags[attr.Key] = fmt.Sprintf("%v", attr.Value)
	}
}

func (s *TraceSpan) RecordError(err error, opts ...EventOption) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log := SpanLog{
		Timestamp: time.Now(),
		Fields: map[string]string{
			"error": err.Error(),
		},
	}
	s.Logs = append(s.Logs, log)
}

func (o *EnhancedObservability) Log(level, message string, fields map[string]interface{}) {
	if !o.config.LoggingEnabled {
		return
	}

	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    fields,
		Service:   "hjtpx",
	}

	entryBytes, _ := json.Marshal(entry)
	fmt.Println(string(entryBytes))
}

type SpanOption func(*SpanConfig)

func WithSpanKind(kind SpanKind) SpanOption {
	return func(cfg *SpanConfig) {
		cfg.SpanKind = kind
	}
}

func WithSpanTag(key string, value interface{}) SpanOption {
	return func(cfg *SpanConfig) {
		if cfg.Attributes == nil {
			cfg.Attributes = make(map[string]interface{})
		}
		cfg.Attributes[key] = value
	}
}

func WithSpanParent(parent SpanContext) SpanOption {
	return func(cfg *SpanConfig) {
		cfg.Parent = parent
	}
}

type SpanConfig struct {
	SpanKind   SpanKind
	Parent     SpanContext
	Attributes map[string]interface{}
}

func ExportMetricsJSON(aggregator *MetricsAggregator) (string, error) {
	aggregator.mu.RLock()
	defer aggregator.mu.RUnlock()

	metrics := make(map[string]interface{})

	counters := make(map[string]uint64)
	for key, c := range aggregator.counters {
		counters[key] = c.Value()
	}
	metrics["counters"] = counters

	gauges := make(map[string]float64)
	for key, g := range aggregator.gauges {
		gauges[key] = g.Value()
	}
	metrics["gauges"] = gauges

	histograms := make(map[string]interface{})
	for key, h := range aggregator.histograms {
		histograms[key] = map[string]interface{}{
			"count": h.Count(),
			"sum":   h.Sum(),
		}
	}
	metrics["histograms"] = histograms

	data, err := json.Marshal(metrics)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func ExportDashboardJSON(manager *DashboardManager) (string, error) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	dashboards := make([]*Dashboard, 0, len(manager.dashboards))
	for _, d := range manager.dashboards {
		dashboards = append(dashboards, d)
	}

	data, err := json.Marshal(dashboards)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func ExportAlertsJSON(manager *AlertManager) (string, error) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	alerts := make([]*Alert, 0, len(manager.alerts))
	for _, a := range manager.alerts {
		alerts = append(alerts, a)
	}

	data, err := json.Marshal(alerts)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
