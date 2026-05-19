package tracing

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type TracerService struct {
	mu           sync.RWMutex
	tracer       trace.Tracer
	exporter     *jaeger.Exporter
	serviceName  string
	enabled      bool
	samplerRatio float64
}

type TraceContext struct {
	TraceID    string
	SpanID     string
	ParentSpanID string
	IsSampled  bool
}

type SpanOptions struct {
	Attributes map[string]interface{}
	StartTime  int64
}

var instance *TracerService
var once sync.Once

func NewTracerService(serviceName string, jaegerEndpoint string, samplerRatio float64) *TracerService {
	once.Do(func() {
		instance = &TracerService{
			serviceName:  serviceName,
			samplerRatio: samplerRatio,
			enabled:      jaegerEndpoint != "",
		}

		if instance.enabled {
			instance.initTracer(jaegerEndpoint)
		}
	})
	return instance
}

func (ts *TracerService) initTracer(endpoint string) {
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
	if err != nil {
		fmt.Printf("Failed to create Jaeger exporter: %v\n", err)
		return
	}

	ts.exporter = exporter

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(ts.serviceName),
		),
	)
	if err != nil {
		fmt.Printf("Failed to create resource: %v\n", err)
		return
	}

	traceProvider := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.ParentBased(tracesdk.TraceIDRatioBased(ts.samplerRatio))),
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(res),
	)

	otel.SetTracerProvider(traceProvider)
	ts.tracer = traceProvider.Tracer(ts.serviceName)
}

func (ts *TracerService) StartSpan(ctx context.Context, name string, opts ...SpanOptions) (context.Context, trace.Span) {
	if !ts.enabled || ts.tracer == nil {
		return ctx, nil
	}

	var spanOptions []trace.SpanOption
	for _, opt := range opts {
		if opt.Attributes != nil {
			for k, v := range opt.Attributes {
				spanOptions = append(spanOptions, trace.WithAttributes(ts.toAttribute(k, v)))
			}
		}
	}

	return ts.tracer.Start(ctx, name, spanOptions...)
}

func (ts *TracerService) toAttribute(key string, value interface{}) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	default:
		return attribute.String(key, fmt.Sprintf("%v", v))
	}
}

func (ts *TracerService) GetTraceContext(ctx context.Context) *TraceContext {
	if !ts.enabled {
		return nil
	}

	span := trace.SpanFromContext(ctx)
	if span == nil || !span.SpanContext().IsValid() {
		return nil
	}

	sc := span.SpanContext()
	return &TraceContext{
		TraceID:      sc.TraceID().String(),
		SpanID:       sc.SpanID().String(),
		ParentSpanID: "",
		IsSampled:    sc.IsSampled(),
	}
}

func (ts *TracerService) InjectTraceContext(ctx context.Context, headers map[string]string) map[string]string {
	if !ts.enabled {
		return headers
	}

	span := trace.SpanFromContext(ctx)
	if span == nil {
		return headers
	}

	sc := span.SpanContext()
	if headers == nil {
		headers = make(map[string]string)
	}

	headers["X-Trace-ID"] = sc.TraceID().String()
	headers["X-Span-ID"] = sc.SpanID().String()
	return headers
}

func (ts *TracerService) RecordError(ctx context.Context, err error, attrs ...attribute.KeyValue) {
	if !ts.enabled {
		return
	}

	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}

	span.RecordError(err, attrs...)
	span.SetStatus(trace.Status{Code: trace.Error, Description: err.Error()})
}

func (ts *TracerService) SetSpanAttributes(ctx context.Context, attrs map[string]interface{}) {
	if !ts.enabled {
		return
	}

	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}

	for k, v := range attrs {
		span.SetAttributes(ts.toAttribute(k, v))
	}
}

func (ts *TracerService) IsEnabled() bool {
	return ts.enabled
}

func (ts *TracerService) Shutdown(ctx context.Context) error {
	if ts.exporter != nil {
		return ts.exporter.Shutdown(ctx)
	}
	return nil
}
