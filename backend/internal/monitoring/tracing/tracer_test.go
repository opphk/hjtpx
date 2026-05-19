package tracing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTracerService(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T, ts *TracerService)
	}{
		{
			name: "Create tracer service disabled",
			test: func(t *testing.T, ts *TracerService) {
				assert.False(t, ts.IsEnabled())
			},
		},
		{
			name: "Start span with disabled tracer",
			test: func(t *testing.T, ts *TracerService) {
				ctx, span := ts.StartSpan(context.Background(), "test-span")
				assert.NotNil(t, ctx)
				assert.Nil(t, span)
			},
		},
		{
			name: "Get trace context with disabled tracer",
			test: func(t *testing.T, ts *TracerService) {
				ctx := context.Background()
				traceCtx := ts.GetTraceContext(ctx)
				assert.Nil(t, traceCtx)
			},
		},
		{
			name: "Inject trace context",
			test: func(t *testing.T, ts *TracerService) {
				ctx := context.Background()
				headers := ts.InjectTraceContext(ctx, nil)
				assert.NotNil(t, headers)
			},
		},
		{
			name: "Record error with disabled tracer",
			test: func(t *testing.T, ts *TracerService) {
				ctx := context.Background()
				err := ts.RecordError(ctx, nil)
				assert.Nil(t, err)
			},
		},
		{
			name: "Set span attributes with disabled tracer",
			test: func(t *testing.T, ts *TracerService) {
				ctx := context.Background()
				ts.SetSpanAttributes(ctx, map[string]interface{}{"key": "value"})
			},
		},
		{
			name: "Shutdown disabled tracer",
			test: func(t *testing.T, ts *TracerService) {
				err := ts.Shutdown(context.Background())
				assert.Nil(t, err)
			},
		},
	}

	ts := NewTracerService("test-service", "", 0.1)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t, ts)
		})
	}
}

func TestTraceContext(t *testing.T) {
	traceCtx := &TraceContext{
		TraceID:    "test-trace-id",
		SpanID:     "test-span-id",
		ParentSpanID: "parent-span-id",
		IsSampled:  true,
	}

	assert.Equal(t, "test-trace-id", traceCtx.TraceID)
	assert.Equal(t, "test-span-id", traceCtx.SpanID)
	assert.Equal(t, "parent-span-id", traceCtx.ParentSpanID)
	assert.True(t, traceCtx.IsSampled)
}

func TestSpanOptions(t *testing.T) {
	opts := SpanOptions{
		Attributes: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
		StartTime: time.Now().UnixNano(),
	}

	assert.NotNil(t, opts.Attributes)
	assert.Equal(t, "value1", opts.Attributes["key1"])
	assert.Equal(t, 123, opts.Attributes["key2"])
	assert.NotZero(t, opts.StartTime)
}
