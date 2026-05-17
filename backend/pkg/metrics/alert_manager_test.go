package metrics

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAlertManager(t *testing.T) {
	am := NewAlertManager(100, 100*time.Millisecond)
	require.NotNil(t, am)
	assert.Equal(t, 100, am.maxBufferSize)
	assert.Equal(t, 100*time.Millisecond, am.flushInterval)
	assert.NotNil(t, am.alertBuffer)
	assert.NotNil(t, am.alertHandlers)
	assert.NotNil(t, am.metrics)
	assert.True(t, am.enabled.Load())
}

func TestAlertManagerRegisterHandler(t *testing.T) {
	am := NewAlertManager(100, 100*time.Millisecond)
	require.NotNil(t, am)

	handler := &testAlertHandler{name: "test-handler"}
	am.RegisterHandler(handler)

	assert.Contains(t, am.alertHandlers, "test-handler")
}

func TestAlertManagerUnregisterHandler(t *testing.T) {
	am := NewAlertManager(100, 100*time.Millisecond)
	require.NotNil(t, am)

	handler := &testAlertHandler{name: "test-handler"}
	am.RegisterHandler(handler)
	assert.Contains(t, am.alertHandlers, "test-handler")

	am.UnregisterHandler("test-handler")
	assert.NotContains(t, am.alertHandlers, "test-handler")
}

func TestAlertManagerSend(t *testing.T) {
	am := NewAlertManager(100, 100*time.Millisecond)
	require.NotNil(t, am)

	alert := &Alert{
		ID:       "test-alert-1",
		Type:     "test",
		Severity: SeverityHigh,
		Message:  "Test alert message",
		Source:   "test",
		Data:     map[string]interface{}{"key": "value"},
	}

	result := am.Send(alert)
	assert.True(t, result)
	assert.Equal(t, uint64(1), am.metrics.alertsReceived.Load())
}

func TestAlertManagerSendDisabled(t *testing.T) {
	am := NewAlertManager(100, 100*time.Millisecond)
	require.NotNil(t, am)

	am.SetEnabled(false)

	alert := &Alert{
		ID:       "test-alert-1",
		Type:     "test",
		Severity: SeverityHigh,
		Message:  "Test alert message",
	}

	result := am.Send(alert)
	assert.False(t, result)
}

func TestAlertManagerSendBufferFull(t *testing.T) {
	am := NewAlertManager(2, 1*time.Hour)
	require.NotNil(t, am)

	alert1 := &Alert{ID: "alert-1", Type: "test", Severity: SeverityHigh}
	alert2 := &Alert{ID: "alert-2", Type: "test", Severity: SeverityHigh}
	alert3 := &Alert{ID: "alert-3", Type: "test", Severity: SeverityHigh}

	result1 := am.Send(alert1)
	result2 := am.Send(alert2)
	result3 := am.Send(alert3)

	assert.True(t, result1)
	assert.True(t, result2)
	assert.False(t, result3)
}

func TestAlertManagerStartStop(t *testing.T) {
	am := NewAlertManager(100, 50*time.Millisecond)
	require.NotNil(t, am)

	ctx := context.Background()
	am.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	am.Stop()
	time.Sleep(50 * time.Millisecond)
}

func TestAlertManagerWithHandler(t *testing.T) {
	am := NewAlertManager(100, 50*time.Millisecond)
	require.NotNil(t, am)

	handler := &testAlertHandler{
		name:       "test-handler",
		handleFunc: func(ctx context.Context, alert *Alert) error {
			return nil
		},
	}
	am.RegisterHandler(handler)

	ctx := context.Background()
	am.Start(ctx)
	defer am.Stop()

	alert := &Alert{
		ID:       "test-alert-1",
		Type:     "test",
		Severity: SeverityHigh,
		Message:  "Test alert message",
	}

	am.Send(alert)
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, uint64(1), am.metrics.alertsReceived.Load())
}

func TestAlertManagerHandlerError(t *testing.T) {
	am := NewAlertManager(100, 50*time.Millisecond)
	require.NotNil(t, am)

	handler := &testAlertHandler{
		name: "error-handler",
		handleFunc: func(ctx context.Context, alert *Alert) error {
			return assert.AnError
		},
	}
	am.RegisterHandler(handler)

	ctx := context.Background()
	am.Start(ctx)
	defer am.Stop()

	alert := &Alert{
		ID:       "test-alert-1",
		Type:     "test",
		Severity: SeverityHigh,
		Message:  "Test alert message",
	}

	am.Send(alert)
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, uint64(1), am.metrics.alertsReceived.Load())
}

func TestAlertManagerMultipleHandlers(t *testing.T) {
	am := NewAlertManager(100, 50*time.Millisecond)
	require.NotNil(t, am)

	var handler1Called, handler2Called atomic.Bool

	handler1 := &testAlertHandler{
		name: "handler-1",
		handleFunc: func(ctx context.Context, alert *Alert) error {
			handler1Called.Store(true)
			return nil
		},
		priority: 1,
	}

	handler2 := &testAlertHandler{
		name: "handler-2",
		handleFunc: func(ctx context.Context, alert *Alert) error {
			handler2Called.Store(true)
			return nil
		},
		priority: 2,
	}

	am.RegisterHandler(handler1)
	am.RegisterHandler(handler2)

	ctx := context.Background()
	am.Start(ctx)
	defer am.Stop()

	alert := &Alert{
		ID:       "test-alert-1",
		Type:     "test",
		Severity: SeverityHigh,
		Message:  "Test alert message",
	}

	am.Send(alert)
	time.Sleep(200 * time.Millisecond)

	assert.True(t, handler1Called.Load())
	assert.True(t, handler2Called.Load())
}

func TestAlertManagerGetStats(t *testing.T) {
	am := NewAlertManager(100, 50*time.Millisecond)
	require.NotNil(t, am)

	handler := &testAlertHandler{
		name: "test-handler",
		handleFunc: func(ctx context.Context, alert *Alert) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}
	am.RegisterHandler(handler)

	ctx := context.Background()
	am.Start(ctx)
	defer am.Stop()

	for i := 0; i < 5; i++ {
		alert := &Alert{
			ID:       "test-alert-" + string(rune('1'+i)),
			Type:     "test",
			Severity: SeverityHigh,
			Message:  "Test alert message",
		}
		am.Send(alert)
	}

	time.Sleep(500 * time.Millisecond)

	stats := am.GetStats()
	assert.Equal(t, uint64(5), stats.TotalReceived)
	assert.Equal(t, 1, stats.HandlerCount)
	assert.Equal(t, 0, stats.BufferSize)
}

func TestAlertManagerSetEnabled(t *testing.T) {
	am := NewAlertManager(100, 100*time.Millisecond)
	require.NotNil(t, am)

	assert.True(t, am.enabled.Load())

	am.SetEnabled(false)
	assert.False(t, am.enabled.Load())

	am.SetEnabled(true)
	assert.True(t, am.enabled.Load())
}

func TestAlertManagerSetTargetResponseTime(t *testing.T) {
	am := NewAlertManager(100, 100*time.Millisecond)
	require.NotNil(t, am)

	am.SetTargetResponseTime(10 * time.Second)
	assert.Equal(t, 10*time.Second, am.targetResponseTime)
}

func TestAlertManagerMeetsTargetResponseTime(t *testing.T) {
	am := NewAlertManager(100, 100*time.Millisecond)
	require.NotNil(t, am)

	am.metrics.RecordResponseTime(10 * time.Millisecond)
	assert.True(t, am.MeetsTargetResponseTime())

	am.SetTargetResponseTime(1 * time.Millisecond)
	assert.False(t, am.MeetsTargetResponseTime())
}

func TestAlertManagerConcurrentSend(t *testing.T) {
	am := NewAlertManager(1000, 100*time.Millisecond)
	require.NotNil(t, am)

	ctx := context.Background()
	am.Start(ctx)
	defer am.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				alert := &Alert{
					ID:       "alert-" + string(rune(idx)) + "-" + string(rune(j)),
					Type:     "test",
					Severity: SeverityHigh,
					Message:  "Test alert message",
				}
				am.Send(alert)
			}
		}(i)
	}
	wg.Wait()

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, uint64(500), am.metrics.alertsReceived.Load())
}

func TestAlertHandlerFunc(t *testing.T) {
	var called bool
	handler := AlertHandlerFunc(func(ctx context.Context, alert *Alert) error {
		called = true
		return nil
	})

	assert.Equal(t, "anonymous", handler.Name())
	assert.Equal(t, 0, handler.Priority())

	err := handler.Handle(context.Background(), &Alert{ID: "test"})
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestAlertSeverityConstants(t *testing.T) {
	assert.Equal(t, AlertSeverity("critical"), SeverityCritical)
	assert.Equal(t, AlertSeverity("high"), SeverityHigh)
	assert.Equal(t, AlertSeverity("medium"), SeverityMedium)
	assert.Equal(t, AlertSeverity("low"), SeverityLow)
	assert.Equal(t, AlertSeverity("info"), SeverityInfo)
}

func TestAlertMetricsRecordResponseTime(t *testing.T) {
	metrics := &AlertMetrics{}

	metrics.RecordResponseTime(100 * time.Millisecond)
	metrics.RecordResponseTime(200 * time.Millisecond)
	metrics.RecordResponseTime(300 * time.Millisecond)

	assert.Equal(t, uint64(3), metrics.responseCount.Load())
	assert.Greater(t, metrics.responseTimeSum.Load(), uint64(0))
}

func TestAlertMetricsGetAverageResponseTime(t *testing.T) {
	metrics := &AlertMetrics{}

	avg := metrics.GetAverageResponseTime()
	assert.Equal(t, time.Duration(0), avg)

	metrics.RecordResponseTime(100 * time.Millisecond)
	metrics.RecordResponseTime(200 * time.Millisecond)

	avg = metrics.GetAverageResponseTime()
	assert.Equal(t, 150*time.Millisecond, avg)
}

type testAlertHandler struct {
	name       string
	priority   int
	handleFunc func(ctx context.Context, alert *Alert) error
	called     bool
}

func (h *testAlertHandler) Handle(ctx context.Context, alert *Alert) error {
	h.called = true
	if h.handleFunc != nil {
		return h.handleFunc(ctx, alert)
	}
	return nil
}

func (h *testAlertHandler) Name() string {
	return h.name
}

func (h *testAlertHandler) Priority() int {
	return h.priority
}

func TestAlertManagerResponseTimeTracking(t *testing.T) {
	am := NewAlertManager(100, 50*time.Millisecond)
	require.NotNil(t, am)

	handler := &testAlertHandler{
		name: "slow-handler",
		handleFunc: func(ctx context.Context, alert *Alert) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		},
	}
	am.RegisterHandler(handler)

	ctx := context.Background()
	am.Start(ctx)
	defer am.Stop()

	alert := &Alert{
		ID:       "test-alert-1",
		Type:     "test",
		Severity: SeverityHigh,
		Message:  "Test alert message",
	}

	am.Send(alert)
	time.Sleep(200 * time.Millisecond)

	stats := am.GetStats()
	assert.True(t, stats.AverageResponseTime >= 50*time.Millisecond)
	assert.True(t, stats.P50ResponseTime >= 50*time.Millisecond)
	assert.True(t, stats.P95ResponseTime >= 50*time.Millisecond)
	assert.True(t, stats.P99ResponseTime >= 50*time.Millisecond)
}

func TestAlertManagerFlushBuffer(t *testing.T) {
	am := NewAlertManager(100, 10*time.Second)
	require.NotNil(t, am)

	handler := &testAlertHandler{
		name: "test-handler",
		handleFunc: func(ctx context.Context, alert *Alert) error {
			return nil
		},
	}
	am.RegisterHandler(handler)

	ctx := context.Background()
	am.Start(ctx)
	defer am.Stop()

	for i := 0; i < 10; i++ {
		alert := &Alert{
			ID:       "test-alert-" + string(rune('0'+i)),
			Type:     "test",
			Severity: SeverityHigh,
			Message:  "Test alert message",
		}
		am.Send(alert)
	}

	am.flushBuffer(ctx)

	stats := am.GetStats()
	assert.Equal(t, uint64(10), stats.TotalReceived)
	assert.Equal(t, 0, stats.BufferSize)
}

func TestAlertManagerGetSortedHandlers(t *testing.T) {
	am := NewAlertManager(100, 100*time.Millisecond)
	require.NotNil(t, am)

	am.RegisterHandler(&testAlertHandler{name: "handler-3", priority: 3})
	am.RegisterHandler(&testAlertHandler{name: "handler-1", priority: 1})
	am.RegisterHandler(&testAlertHandler{name: "handler-2", priority: 2})

	handlers := am.getSortedHandlers()
	assert.Len(t, handlers, 3)
	assert.Equal(t, "handler-1", handlers[0].Name())
	assert.Equal(t, "handler-2", handlers[1].Name())
	assert.Equal(t, "handler-3", handlers[2].Name())
}
