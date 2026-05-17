package highavailability

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreaker_InitialState(t *testing.T) {
	cb := NewCircuitBreaker("test", nil)

	assert.Equal(t, CircuitStateClosed, cb.GetState())
}

func TestCircuitBreaker_AllowRequest_Closed(t *testing.T) {
	cb := NewCircuitBreaker("test", nil)

	assert.True(t, cb.AllowRequest())
}

func TestCircuitBreaker_AllowRequest_Open(t *testing.T) {
	cb := NewCircuitBreaker("test", &CircuitBreakerConfig{
		FailureThreshold: 2,
		Timeout:         100 * time.Millisecond,
	})

	cb.RecordFailure()
	cb.RecordFailure()

	assert.False(t, cb.AllowRequest())
}

func TestCircuitBreaker_AllowRequest_OpenToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker("test", &CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:         50 * time.Millisecond,
	})

	cb.RecordFailure()
	assert.False(t, cb.AllowRequest())

	time.Sleep(60 * time.Millisecond)

	assert.True(t, cb.AllowRequest())
	assert.Equal(t, CircuitStateHalfOpen, cb.GetState())
}

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	cb := NewCircuitBreaker("test", nil)

	cb.RecordSuccess()

	stats := cb.GetStats()
	assert.Equal(t, uint64(1), stats.TotalSuccesses)
}

func TestCircuitBreaker_RecordFailure(t *testing.T) {
	cb := NewCircuitBreaker("test", nil)

	cb.RecordFailure()

	stats := cb.GetStats()
	assert.Equal(t, uint64(1), stats.TotalFailures)
}

func TestCircuitBreaker_TransitionToOpen(t *testing.T) {
	cb := NewCircuitBreaker("test", &CircuitBreakerConfig{
		FailureThreshold: 3,
	})

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	assert.Equal(t, CircuitStateOpen, cb.GetState())
}

func TestCircuitBreaker_HalfOpenRecovery(t *testing.T) {
	cb := NewCircuitBreaker("test", &CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:         10 * time.Millisecond,
	})

	cb.RecordFailure()

	time.Sleep(20 * time.Millisecond)

	assert.True(t, cb.AllowRequest())
	assert.Equal(t, CircuitStateHalfOpen, cb.GetState())

	cb.RecordSuccess()
	cb.RecordSuccess()

	assert.Equal(t, CircuitStateClosed, cb.GetState())
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker("test", &CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:         10 * time.Millisecond,
	})

	cb.RecordFailure()

	time.Sleep(20 * time.Millisecond)

	assert.True(t, cb.AllowRequest())
	assert.Equal(t, CircuitStateHalfOpen, cb.GetState())

	cb.RecordFailure()

	assert.Equal(t, CircuitStateOpen, cb.GetState())
}

func TestCircuitBreaker_GetStats(t *testing.T) {
	cb := NewCircuitBreaker("test", nil)

	cb.AllowRequest()
	cb.RecordSuccess()
	cb.AllowRequest()
	cb.RecordFailure()

	stats := cb.GetStats()

	assert.Equal(t, "test", stats.Name)
	assert.Equal(t, uint64(2), stats.TotalRequests)
	assert.Equal(t, uint64(1), stats.TotalSuccesses)
	assert.Equal(t, uint64(1), stats.TotalFailures)
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker("test", &CircuitBreakerConfig{
		FailureThreshold: 1,
	})

	cb.RecordFailure()
	assert.Equal(t, CircuitStateOpen, cb.GetState())

	cb.Reset()

	assert.Equal(t, CircuitStateClosed, cb.GetState())
	stats := cb.GetStats()
	assert.Equal(t, int32(0), stats.Failures)
}

func TestCircuitBreaker_ForceOpen(t *testing.T) {
	cb := NewCircuitBreaker("test", nil)

	cb.ForceOpen()

	assert.Equal(t, CircuitStateOpen, cb.GetState())
	assert.False(t, cb.AllowRequest())
}

func TestCircuitBreaker_ForceClosed(t *testing.T) {
	cb := NewCircuitBreaker("test", &CircuitBreakerConfig{
		FailureThreshold: 1,
	})

	cb.RecordFailure()
	assert.Equal(t, CircuitStateOpen, cb.GetState())

	cb.ForceClosed()

	assert.Equal(t, CircuitStateClosed, cb.GetState())
	assert.True(t, cb.AllowRequest())
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	cb := NewCircuitBreaker("test", nil)
	ctx := context.Background()

	err := cb.Execute(ctx, func() error {
		return nil
	})

	require.NoError(t, err)

	stats := cb.GetStats()
	assert.Equal(t, uint64(1), stats.TotalRequests)
	assert.Equal(t, uint64(1), stats.TotalSuccesses)
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	cb := NewCircuitBreaker("test", nil)
	ctx := context.Background()

	err := cb.Execute(ctx, func() error {
		return assert.AnError
	})

	assert.Error(t, err)

	stats := cb.GetStats()
	assert.Equal(t, uint64(1), stats.TotalRequests)
	assert.Equal(t, uint64(1), stats.TotalFailures)
}

func TestCircuitBreaker_Execute_ContextCanceled(t *testing.T) {
	cb := NewCircuitBreaker("test", nil)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := cb.Execute(ctx, func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestCircuitBreaker_Execute_OpenCircuit(t *testing.T) {
	cb := NewCircuitBreaker("test", &CircuitBreakerConfig{
		FailureThreshold: 5,
	})

	for i := 0; i < 6; i++ {
		cb.RecordFailure()
	}

	ctx := context.Background()

	err := cb.Execute(ctx, func() error {
		return fmt.Errorf("simulated error to keep circuit open")
	})

	assert.Error(t, err)
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker("test", nil)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.AllowRequest()
			cb.RecordSuccess()
		}()
	}
	wg.Wait()

	stats := cb.GetStats()
	assert.Equal(t, uint64(10), stats.TotalRequests)
	assert.Equal(t, uint64(10), stats.TotalSuccesses)
}

func TestCircuitBreakerGroup_GetOrCreate(t *testing.T) {
	group := NewCircuitBreakerGroup(nil)

	cb1 := group.GetOrCreate("service1")
	cb2 := group.GetOrCreate("service1")
	cb3 := group.GetOrCreate("service2")

	assert.Same(t, cb1, cb2)
	assert.NotSame(t, cb1, cb3)
}

func TestCircuitBreakerGroup_Get(t *testing.T) {
	group := NewCircuitBreakerGroup(nil)

	group.GetOrCreate("service1")

	cb, ok := group.Get("service1")
	assert.True(t, ok)
	assert.NotNil(t, cb)

	_, ok = group.Get("non-existent")
	assert.False(t, ok)
}

func TestCircuitBreakerGroup_GetAllStats(t *testing.T) {
	group := NewCircuitBreakerGroup(nil)

	group.GetOrCreate("service1")
	group.GetOrCreate("service2")

	stats := group.GetAllStats()
	assert.Len(t, stats, 2)
}

func TestCircuitBreakerGroup_ResetAll(t *testing.T) {
	group := NewCircuitBreakerGroup(nil)

	cb := group.GetOrCreate("service1")
	cb.ForceOpen()

	group.ResetAll()

	assert.Equal(t, CircuitStateClosed, cb.GetState())
}

func TestHeartbeatManager_Register(t *testing.T) {
	hm := NewHeartbeatManager(nil)

	err := hm.Register("heartbeat-1", "test-heartbeat", nil)
	require.NoError(t, err)

	hb, ok := hm.GetHeartbeat("heartbeat-1")
	assert.True(t, ok)
	assert.Equal(t, "heartbeat-1", hb.ID)
	assert.Equal(t, "test-heartbeat", hb.Name)
	assert.True(t, hb.Active)
}

func TestHeartbeatManager_Unregister(t *testing.T) {
	hm := NewHeartbeatManager(nil)

	err := hm.Register("heartbeat-1", "test-heartbeat", nil)
	require.NoError(t, err)

	err = hm.Unregister("heartbeat-1")
	require.NoError(t, err)

	_, ok := hm.GetHeartbeat("heartbeat-1")
	assert.False(t, ok)
}

func TestHeartbeatManager_Beat(t *testing.T) {
	hm := NewHeartbeatManager(nil)

	err := hm.Register("heartbeat-1", "test-heartbeat", nil)
	require.NoError(t, err)

	err = hm.Beat("heartbeat-1")
	require.NoError(t, err)

	hb, _ := hm.GetHeartbeat("heartbeat-1")
	assert.True(t, hb.Active)
}

func TestHeartbeatManager_GetAllHeartbeats(t *testing.T) {
	hm := NewHeartbeatManager(nil)

	hm.Register("heartbeat-1", "test-1", nil)
	hm.Register("heartbeat-2", "test-2", nil)

	heartbeats := hm.GetAllHeartbeats()
	assert.Len(t, heartbeats, 2)
}

func TestHeartbeatManager_GetActiveCount(t *testing.T) {
	hm := NewHeartbeatManager(nil)

	hm.Register("heartbeat-1", "test-1", nil)
	hm.Register("heartbeat-2", "test-2", nil)

	assert.Equal(t, 2, hm.GetActiveCount())
}

func TestHeartbeatManager_Listener(t *testing.T) {
	hm := NewHeartbeatManager(nil)

	var timeoutCalled bool
	var failureCalled bool
	var recoveryCalled bool

	listener := &testHeartbeatListener{
		onTimeout: func(id, name string) {
			timeoutCalled = true
		},
		onFailure: func(id, name string, count int) {
			failureCalled = true
		},
		onRecovery: func(id, name string) {
			recoveryCalled = true
		},
	}

	hm.AddListener(listener)

	hm.Register("heartbeat-1", "test-heartbeat", nil)

	hm.notifyTimeout("heartbeat-1", "test-heartbeat")
	assert.True(t, timeoutCalled)

	hm.notifyFailure("heartbeat-1", "test-heartbeat", 1)
	assert.True(t, failureCalled)

	hm.notifyRecovery("heartbeat-1", "test-heartbeat")
	assert.True(t, recoveryCalled)

	timeoutCalled = false
	failureCalled = false
	recoveryCalled = false

	listener.OnHeartbeatTimeout("id", "name")
	assert.True(t, timeoutCalled)

	listener.OnHeartbeatFailure("id", "name", 1)
	assert.True(t, failureCalled)

	listener.OnHeartbeatRecovery("id", "name")
	assert.True(t, recoveryCalled)
}

type testHeartbeatListener struct {
	onTimeout   func(id, name string)
	onFailure  func(id, name string, count int)
	onRecovery func(id, name string)
}

func (t *testHeartbeatListener) OnHeartbeatTimeout(id, name string) {
	if t.onTimeout != nil {
		t.onTimeout(id, name)
	}
}

func (t *testHeartbeatListener) OnHeartbeatFailure(id, name string, count int) {
	if t.onFailure != nil {
		t.onFailure(id, name, count)
	}
}

func (t *testHeartbeatListener) OnHeartbeatRecovery(id, name string) {
	if t.onRecovery != nil {
		t.onRecovery(id, name)
	}
}

func TestHeartbeatCallback(t *testing.T) {
	var called bool
	cb := HeartbeatCallback(func(id, name string) {
		called = true
	})

	cb.OnHeartbeatTimeout("id", "name")
	assert.True(t, called)

	called = false
	cb.OnHeartbeatFailure("id", "name", 1)
	assert.False(t, called)

	cb.OnHeartbeatRecovery("id", "name")
	assert.False(t, called)
}

func TestHeartbeatManager_StartStop(t *testing.T) {
	hm := NewHeartbeatManager(&HeartbeatManagerConfig{
		Interval: 10 * time.Millisecond,
		Timeout:  50 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go hm.Start(ctx)

	time.Sleep(50 * time.Millisecond)
	cancel()

	hm.Stop()
}

func TestAutoRestartManager_RegisterProcess(t *testing.T) {
	arm := NewAutoRestartManager(nil)

	info := &ProcessInfo{
		ID:      "process-1",
		Name:    "test-process",
		Command: "test-command",
	}

	err := arm.RegisterProcess(info)
	require.NoError(t, err)

	process, ok := arm.GetProcess("process-1")
	assert.True(t, ok)
	assert.Equal(t, "process-1", process.ID)
	assert.Equal(t, "running", process.Status)
}

func TestAutoRestartManager_UnregisterProcess(t *testing.T) {
	arm := NewAutoRestartManager(nil)

	info := &ProcessInfo{
		ID:   "process-1",
		Name: "test-process",
	}

	err := arm.RegisterProcess(info)
	require.NoError(t, err)

	err = arm.UnregisterProcess("process-1")
	require.NoError(t, err)

	_, ok := arm.GetProcess("process-1")
	assert.False(t, ok)
}

func TestAutoRestartManager_MarkRunning(t *testing.T) {
	arm := NewAutoRestartManager(nil)

	info := &ProcessInfo{
		ID:   "process-1",
		Name: "test-process",
	}

	err := arm.RegisterProcess(info)
	require.NoError(t, err)

	err = arm.MarkRunning("process-1")
	require.NoError(t, err)

	process, _ := arm.GetProcess("process-1")
	assert.Equal(t, "running", process.Status)
}

func TestAutoRestartManager_MarkStopped(t *testing.T) {
	arm := NewAutoRestartManager(nil)

	info := &ProcessInfo{
		ID:   "process-1",
		Name: "test-process",
	}

	err := arm.RegisterProcess(info)
	require.NoError(t, err)

	err = arm.MarkStopped("process-1")
	require.NoError(t, err)

	process, _ := arm.GetProcess("process-1")
	assert.Equal(t, 1, process.RestartCount)
}

func TestAutoRestartManager_GetAllProcesses(t *testing.T) {
	arm := NewAutoRestartManager(nil)

	arm.RegisterProcess(&ProcessInfo{ID: "process-1", Name: "test-1"})
	arm.RegisterProcess(&ProcessInfo{ID: "process-2", Name: "test-2"})

	processes := arm.GetAllProcesses()
	assert.Len(t, processes, 2)
}

func TestAutoRestartManager_GetRestartDelay(t *testing.T) {
	arm := NewAutoRestartManager(nil)

	delay1 := arm.GetRestartDelay(1)
	delay2 := arm.GetRestartDelay(2)
	delay3 := arm.GetRestartDelay(3)

	assert.Greater(t, delay2, delay1)
	assert.Greater(t, delay3, delay2)
}

func TestAutoRestartManager_MaxRetries(t *testing.T) {
	arm := NewAutoRestartManager(&AutoRestartConfig{
		Restart: &RestartConfig{
			MaxRetries: 3,
		},
	})

	info := &ProcessInfo{
		ID:   "process-1",
		Name: "test-process",
	}

	arm.RegisterProcess(info)

	for i := 0; i < 3; i++ {
		arm.MarkStopped("process-1")
	}

	process, _ := arm.GetProcess("process-1")
	assert.Equal(t, "failed", process.Status)
}

func TestAutoRestartManager_Listener(t *testing.T) {
	arm := NewAutoRestartManager(nil)

	var restartCalled bool
	var crashCalled bool

	listener := &testRestartListener{
		onRestart: func(id, name string, count int) {
			restartCalled = true
		},
		onCrash: func(id, name string, err error) {
			crashCalled = true
		},
	}

	arm.AddListener(listener)

	info := &ProcessInfo{
		ID:   "process-1",
		Name: "test-process",
	}
	arm.RegisterProcess(info)

	arm.notifyRestart("process-1", "test-process", 1)
	assert.True(t, restartCalled)

	arm.notifyCrash("process-1", "test-process", fmt.Errorf("crash error"))
	assert.True(t, crashCalled)

	restartCalled = false
	crashCalled = false

	listener.OnProcessRestart("id", "name", 1)
	assert.True(t, restartCalled)

	listener.OnProcessCrash("id", "name", fmt.Errorf("error"))
	assert.True(t, crashCalled)
}

type testRestartListener struct {
	onRestart func(id, name string, count int)
	onCrash  func(id, name string, err error)
}

func (t *testRestartListener) OnProcessRestart(id, name string, count int) {
	if t.onRestart != nil {
		t.onRestart(id, name, count)
	}
}

func (t *testRestartListener) OnProcessCrash(id, name string, err error) {
	if t.onCrash != nil {
		t.onCrash(id, name, err)
	}
}

func TestMetricsCollector_RecordCounter(t *testing.T) {
	mc := NewMetricsCollector(time.Minute, time.Minute)

	mc.RecordCounter("requests", 1.0, nil)
	mc.RecordCounter("requests", 1.0, nil)

	metric, ok := mc.GetMetric("requests", nil)
	assert.True(t, ok)
	assert.Equal(t, 1.0, metric.Value)
}

func TestMetricsCollector_RecordGauge(t *testing.T) {
	mc := NewMetricsCollector(time.Minute, time.Minute)

	mc.RecordGauge("temperature", 25.5, nil)

	metric, ok := mc.GetMetric("temperature", nil)
	assert.True(t, ok)
	assert.Equal(t, 25.5, metric.Value)
}

func TestMetricsCollector_RecordHistogram(t *testing.T) {
	mc := NewMetricsCollector(time.Minute, time.Minute)

	mc.RecordHistogram("latency", 100.0, nil)
	mc.RecordHistogram("latency", 200.0, nil)
	mc.RecordHistogram("latency", 300.0, nil)

	agg, ok := mc.GetAggregator("latency", nil)
	assert.True(t, ok)
	assert.Equal(t, 3, agg.Count)
	assert.Equal(t, 600.0, agg.Sum)
}

func TestMetricsCollector_GetAverage(t *testing.T) {
	mc := NewMetricsCollector(time.Minute, time.Minute)

	mc.RecordGauge("value", 10.0, nil)
	mc.RecordGauge("value", 20.0, nil)
	mc.RecordGauge("value", 30.0, nil)

	avg := mc.GetAverage("value", nil)
	assert.Equal(t, 20.0, avg)
}

func TestMetricsCollector_GetPercentile(t *testing.T) {
	mc := NewMetricsCollector(time.Minute, time.Minute)

	for i := 1; i <= 10; i++ {
		mc.RecordGauge("latency", float64(i*10), nil)
	}

	p50 := mc.GetPercentile("latency", nil, 50)
	p90 := mc.GetPercentile("latency", nil, 90)

	assert.InDelta(t, 50.0, p50, 10.0)
	assert.InDelta(t, 90.0, p90, 10.0)
}

func TestMetricsCollector_GetPercentile_EmptyAggregator(t *testing.T) {
	mc := NewMetricsCollector(time.Minute, time.Minute)

	result := mc.GetPercentile("non-existent", nil, 50)
	assert.Equal(t, 0.0, result)
}

func TestMetricsCollector_GetPercentile_WithLabels(t *testing.T) {
	mc := NewMetricsCollector(time.Minute, time.Minute)

	mc.RecordGauge("latency", 100.0, map[string]string{"service": "api"})
	mc.RecordGauge("latency", 200.0, map[string]string{"service": "api"})

	result := mc.GetPercentile("latency", map[string]string{"service": "api"}, 50)
	assert.Greater(t, result, 0.0)
}

func TestMetricsCollector_StartStop(t *testing.T) {
	mc := NewMetricsCollector(time.Millisecond*10, time.Millisecond*10)
	ctx, cancel := context.WithCancel(context.Background())

	go mc.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()

	mc.Stop()
}

func TestMetricsCollector_makeKey(t *testing.T) {
	mc := NewMetricsCollector(time.Minute, time.Minute)

	key1 := mc.makeKey("test", nil)
	assert.Equal(t, "test", key1)

	key2 := mc.makeKey("test", map[string]string{"a": "1"})
	assert.Contains(t, key2, "test")
	assert.Contains(t, key2, "a=1")
}

func TestMetricsCollector_GetAllAggregators(t *testing.T) {
	mc := NewMetricsCollector(time.Minute, time.Minute)

	mc.RecordCounter("metric1", 1.0, nil)
	mc.RecordGauge("metric2", 2.0, nil)

	aggs := mc.GetAllAggregators()
	assert.Len(t, aggs, 2)
}

func TestAvailabilityTracker_RecordUp(t *testing.T) {
	at := NewAvailabilityTracker(time.Hour, 99.9)

	at.RecordUp()

	stats := at.GetStats()
	assert.True(t, stats["is_up"].(bool))
}

func TestAvailabilityTracker_RecordDown(t *testing.T) {
	at := NewAvailabilityTracker(time.Hour, 99.9)

	at.RecordUp()
	at.RecordDown()

	stats := at.GetStats()
	assert.False(t, stats["is_up"].(bool))
}

func TestAvailabilityTracker_RecordRequest(t *testing.T) {
	at := NewAvailabilityTracker(time.Hour, 99.9)

	at.RecordRequest()
	at.RecordRequest()
	at.RecordRequest()

	stats := at.GetStats()
	assert.Equal(t, uint64(3), stats["total_requests"])
}

func TestAvailabilityTracker_RecordError(t *testing.T) {
	at := NewAvailabilityTracker(time.Hour, 99.9)

	at.RecordError()

	stats := at.GetStats()
	assert.Equal(t, uint64(1), stats["total_errors"])
}

func TestAvailabilityTracker_GetAvailability(t *testing.T) {
	at := NewAvailabilityTracker(time.Hour, 99.9)

	availability := at.GetAvailability()
	assert.Greater(t, availability, 99.0)
}

func TestAvailabilityTracker_GetErrorRate(t *testing.T) {
	at := NewAvailabilityTracker(time.Hour, 99.9)

	for i := 0; i < 10; i++ {
		at.RecordRequest()
	}
	for i := 0; i < 2; i++ {
		at.RecordError()
	}

	errorRate := at.GetErrorRate()
	assert.InDelta(t, 20.0, errorRate, 1.0)
}

func TestAvailabilityTracker_GetSLOStatus(t *testing.T) {
	at := NewAvailabilityTracker(time.Hour, 99.9)

	assert.True(t, at.GetSLOStatus())

	at = NewAvailabilityTracker(time.Hour, 99.9)
	at.RecordDown()
	for i := 0; i < 100; i++ {
		at.RecordRequest()
		at.RecordError()
	}

	assert.False(t, at.GetSLOStatus())
}

func TestAvailabilityTracker_GetStats(t *testing.T) {
	at := NewAvailabilityTracker(time.Hour, 99.9)

	at.RecordRequest()
	at.RecordError()

	stats := at.GetStats()

	assert.Contains(t, stats, "uptime")
	assert.Contains(t, stats, "availability")
	assert.Contains(t, stats, "error_rate")
	assert.Contains(t, stats, "slo_target")
	assert.Contains(t, stats, "slo_met")
}
