package tracing

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type TracingMiddleware struct {
	tracer trace.Tracer
}

func NewTracingMiddleware(serviceName string) *TracingMiddleware {
	return &TracingMiddleware{
		tracer: otel.Tracer(serviceName),
	}
}

func (tm *TracingMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		spanName := c.Request.Method + " " + c.Request.URL.Path

		ctx, span := tm.tracer.Start(c.Request.Context(), spanName)
		defer span.End()

		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.path", c.Request.URL.Path),
			attribute.String("http.host", c.Request.Host),
			attribute.String("http.user_agent", c.Request.UserAgent()),
			attribute.String("client.ip", c.ClientIP()),
		)

		c.Request = c.Request.WithContext(ctx)

		traceCtx := span.SpanContext()
		c.Set("trace_id", traceCtx.TraceID().String())
		c.Set("span_id", traceCtx.SpanID().String())

		c.Next()

		duration := time.Since(start)
		span.SetAttributes(
			attribute.Int("http.status_code", c.Writer.Status()),
			attribute.Int64("http.duration_ms", duration.Milliseconds()),
		)

		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				span.RecordError(err.Err)
			}
			span.SetStatus(trace.Status{Code: trace.Error, Description: c.Errors.String()})
		}
	}
}

func ExtractTraceIDFromRequest(r *http.Request) string {
	if ctx := r.Context(); ctx != nil {
		if span := trace.SpanFromContext(ctx); span != nil {
			return span.SpanContext().TraceID().String()
		}
	}
	return r.Header.Get("X-Trace-ID")
}

func InjectTraceIDToResponse(w http.ResponseWriter, traceID string) {
	w.Header().Set("X-Trace-ID", traceID)
}

type SpanRecorder struct {
	span trace.Span
}

func NewSpanRecorder(ctx context.Context, operationName string) *SpanRecorder {
	tracer := otel.Tracer("hjtpx")
	_, span := tracer.Start(ctx, operationName)
	return &SpanRecorder{span: span}
}

func (sr *SpanRecorder) SetAttribute(key string, value interface{}) {
	switch v := value.(type) {
	case string:
		sr.span.SetAttributes(attribute.String(key, v))
	case int:
		sr.span.SetAttributes(attribute.Int(key, v))
	case int64:
		sr.span.SetAttributes(attribute.Int64(key, v))
	case float64:
		sr.span.SetAttributes(attribute.Float64(key, v))
	case bool:
		sr.span.SetAttributes(attribute.Bool(key, v))
	}
}

func (sr *SpanRecorder) RecordError(err error) {
	sr.span.RecordError(err)
	sr.span.SetStatus(trace.Status{Code: trace.Error, Description: err.Error()})
}

func (sr *SpanRecorder) End() {
	sr.span.End()
}
