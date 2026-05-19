package monitoring

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

type OpenTelemetryProvider struct {
	config  *config.OpenTelemetryConfig
	enabled bool
	mu      sync.RWMutex
	tracer traceProvider
}

type traceProvider interface {
	StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)
}

type SpanOption func(*SpanConfig)
type SpanConfig struct {
	SpanKind SpanKind
}

type SpanKind int

const (
	SpanKindUnspecified SpanKind = iota
	SpanKindInternal
	SpanKindServer
	SpanKindClient
	SpanKindProducer
	SpanKindConsumer
)

type Span interface {
	End()
	AddEvent(string, ...EventOption)
	IsRecording() bool
	SpanContext() SpanContext
	SetStatus(Status)
	SetName(string)
	SetAttributes(...Attribute)
	RecordError(error, ...EventOption)
}

type EventOption func(*EventConfig)
type EventConfig struct {
	Timestamp time.Time
	Attributes []Attribute
}

type Attribute struct {
	Key   string
	Value interface{}
}

type SpanContext struct {
	TraceID string
	SpanID  string
}

type Status struct {
	Code    StatusCode
	Message string
}

type StatusCode int

const (
	StatusCodeUnset StatusCode = iota
	StatusCodeOk
	StatusCodeError
)

func NewOpenTelemetryProvider(cfg *config.OpenTelemetryConfig) (*OpenTelemetryProvider, error) {
	provider := &OpenTelemetryProvider{
		config:  cfg,
		enabled: cfg.Enabled,
		tracer: &noopTracer{},
	}

	if !cfg.Enabled {
		log.Println("OpenTelemetry is disabled")
		return provider, nil
	}

	log.Printf("OpenTelemetry initialized with endpoint: %s", cfg.Endpoint)
	return provider, nil
}

func (p *OpenTelemetryProvider) Tracer() traceProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.tracer
}

func (p *OpenTelemetryProvider) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

func (p *OpenTelemetryProvider) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.enabled {
		return ctx, &noopSpan{}
	}

	return p.tracer.StartSpan(ctx, name, opts...)
}

func (p *OpenTelemetryProvider) End() error {
	return nil
}

func (p *OpenTelemetryProvider) AddSpanAttributes(ctx context.Context, attrs map[string]interface{}) {
	span := SpanFromContext(ctx)
	if span != nil && span.IsRecording() {
		for k, v := range attrs {
			span.SetAttributes(Attribute{Key: k, Value: fmt.Sprintf("%v", v)})
		}
	}
}

func (p *OpenTelemetryProvider) RecordError(ctx context.Context, err error, attrs map[string]interface{}) {
	span := SpanFromContext(ctx)
	if span != nil && span.IsRecording() {
		span.RecordError(err)
		for k, v := range attrs {
			span.SetAttributes(Attribute{Key: k, Value: fmt.Sprintf("%v", v)})
		}
	}
}

type noopSpan struct{}

func (n *noopSpan) End()                                       {}
func (n *noopSpan) AddEvent(string, ...EventOption)           {}
func (n *noopSpan) IsRecording() bool                         { return false }
func (n *noopSpan) SpanContext() SpanContext                  { return SpanContext{} }
func (n *noopSpan) SetStatus(Status)                         {}
func (n *noopSpan) SetName(string)                            {}
func (n *noopSpan) SetAttributes(...Attribute)                 {}
func (n *noopSpan) RecordError(error, ...EventOption)         {}

type noopTracer struct{}

func (t *noopTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	return ctx, &noopSpan{}
}

func SpanFromContext(ctx context.Context) Span {
	return nil
}

type DistributedTracer struct {
	provider  *OpenTelemetryProvider
	mu        sync.RWMutex
}

func NewDistributedTracer(provider *OpenTelemetryProvider) *DistributedTracer {
	return &DistributedTracer{
		provider: provider,
	}
}

func (t *DistributedTracer) ExtractTraceID(ctx context.Context) string {
	return ""
}

func (t *DistributedTracer) ExtractSpanID(ctx context.Context) string {
	return ""
}

func (t *DistributedTracer) InjectTraceContext(ctx context.Context, carrier map[string]string) context.Context {
	return ctx
}

func (t *DistributedTracer) ExtractTraceContext(ctx context.Context, carrier map[string]string) context.Context {
	return ctx
}

func (t *DistributedTracer) StartRemoteSpan(ctx context.Context, operationName string) (context.Context, Span) {
	return t.provider.StartSpan(ctx, operationName,
		WithSpanKind(SpanKindServer),
	)
}

func WithSpanKind(kind SpanKind) SpanOption {
	return func(cfg *SpanConfig) {
		cfg.SpanKind = kind
	}
}

type SpanCollector struct {
	spans    []*TraceSpan
	mu       sync.Mutex
	maxSpans int
}

type TraceSpan struct {
	Span      Span
	StartTime time.Time
	EndTime   time.Time
	Attrs     map[string]interface{}
}

func NewSpanCollector(maxSpans int) *SpanCollector {
	return &SpanCollector{
		spans:    make([]*TraceSpan, 0, maxSpans),
		maxSpans: maxSpans,
	}
}

func (c *SpanCollector) AddSpan(span *TraceSpan) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.spans = append(c.spans, span)
	if len(c.spans) > c.maxSpans {
		c.spans = c.spans[1:]
	}
}

func (c *SpanCollector) GetSpans() []*TraceSpan {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]*TraceSpan, len(c.spans))
	copy(result, c.spans)
	return result
}

func (c *SpanCollector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.spans = c.spans[:0]
}

type PerformanceProfiler struct {
	cpuProfiles     []CPUProfile
	memProfiles     []MemProfile
	goroutineCount  int64
	mu              sync.RWMutex
	stopCh          chan struct{}
	enabled         bool
}

type CPUProfile struct {
	Timestamp   time.Time
	Duration    time.Duration
	Data        []byte
	StackTraces map[string]int
}

type MemProfile struct {
	Timestamp   time.Time
	AllocBytes  int64
	TotalAlloc  int64
	SysBytes    int64
	NumGC       uint32
	StackTraces map[string]int
}

func NewPerformanceProfiler(enabled bool) *PerformanceProfiler {
	return &PerformanceProfiler{
		enabled: enabled,
		stopCh:  make(chan struct{}),
		mu:      sync.RWMutex{},
	}
}

func (p *PerformanceProfiler) Start() {
	if !p.enabled {
		return
	}

	log.Println("Performance profiler started")
}

func (p *PerformanceProfiler) Stop() {
	if !p.enabled {
		return
	}

	close(p.stopCh)
	log.Println("Performance profiler stopped")
}

func (p *PerformanceProfiler) RecordCPUProfile(data []byte, duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	profile := CPUProfile{
		Timestamp:   time.Now(),
		Duration:   duration,
		Data:       data,
		StackTraces: make(map[string]int),
	}

	p.cpuProfiles = append(p.cpuProfiles, profile)
}

func (p *PerformanceProfiler) RecordMemProfile() {
	p.mu.Lock()
	defer p.mu.Unlock()

	profile := MemProfile{
		Timestamp:   time.Now(),
		StackTraces: make(map[string]int),
	}

	p.memProfiles = append(p.memProfiles, profile)
}

func (p *PerformanceProfiler) GetCPUProfiles() []CPUProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]CPUProfile, len(p.cpuProfiles))
	copy(result, p.cpuProfiles)
	return result
}

func (p *PerformanceProfiler) GetMemProfiles() []MemProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]MemProfile, len(p.memProfiles))
	copy(result, p.memProfiles)
	return result
}

func (p *PerformanceProfiler) GetGoroutineCount() int {
	return int(atomic.LoadInt64(&p.goroutineCount))
}

func (p *PerformanceProfiler) SetGoroutineCount(count int) {
	atomic.StoreInt64(&p.goroutineCount, int64(count))
}

type MetricsCollector struct {
	metrics    map[string]*Metric
	mu         sync.RWMutex
	timeWindow time.Duration
}

type Metric struct {
	Name      string
	Value     float64
	Type      MetricType
	Labels    map[string]string
	Timestamp time.Time
}

type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge    MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

func NewMetricsCollector(timeWindow time.Duration) *MetricsCollector {
	return &MetricsCollector{
		metrics:    make(map[string]*Metric),
		timeWindow: timeWindow,
	}
}

func (c *MetricsCollector) RecordMetric(name string, value float64, metricType MetricType, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.makeKey(name, labels)

	c.metrics[key] = &Metric{
		Name:      name,
		Value:     value,
		Type:      metricType,
		Labels:    labels,
		Timestamp: time.Now(),
	}
}

func (c *MetricsCollector) GetMetric(name string, labels map[string]string) (*Metric, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.makeKey(name, labels)
	metric, ok := c.metrics[key]
	return metric, ok
}

func (c *MetricsCollector) GetAllMetrics() map[string]*Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*Metric, len(c.metrics))
	for k, v := range c.metrics {
		result[k] = v
	}
	return result
}

func (c *MetricsCollector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics = make(map[string]*Metric)
}

func (c *MetricsCollector) makeKey(name string, labels map[string]string) string {
	return fmt.Sprintf("%s:%v", name, labels)
}

type LatencyRecorder struct {
	latencies  []float64
	mu         sync.Mutex
	maxSamples int
}

func NewLatencyRecorder(maxSamples int) *LatencyRecorder {
	return &LatencyRecorder{
		latencies:  make([]float64, 0, maxSamples),
		maxSamples: maxSamples,
	}
}

func (r *LatencyRecorder) Record(latencyMs float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.latencies = append(r.latencies, latencyMs)
	if len(r.latencies) > r.maxSamples {
		r.latencies = r.latencies[1:]
	}
}

func (r *LatencyRecorder) GetPercentile(percentile float64) float64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.latencies) == 0 {
		return 0
	}

	sorted := make([]float64, len(r.latencies))
	copy(sorted, r.latencies)

	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := int(float64(len(sorted)) * percentile / 100)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func (r *LatencyRecorder) GetAverage() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.latencies) == 0 {
		return 0
	}

	var sum float64
	for _, v := range r.latencies {
		sum += v
	}
	return sum / float64(len(r.latencies))
}

func (r *LatencyRecorder) GetMax() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.latencies) == 0 {
		return 0
	}

	var max float64
	for _, v := range r.latencies {
		if v > max {
			max = v
		}
	}
	return max
}

func (r *LatencyRecorder) GetMin() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.latencies) == 0 {
		return 0
	}

	min := r.latencies[0]
	for _, v := range r.latencies {
		if v < min {
			min = v
		}
	}
	return min
}

func (r *LatencyRecorder) GetCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.latencies)
}
