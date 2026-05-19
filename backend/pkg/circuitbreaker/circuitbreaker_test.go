package circuitbreaker

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCircuitBreakerClosedState(t *testing.T) {
	opts := Options{
		Name:                "test-closed",
		MaxRequests:         3,
		Interval:            10,
		Timeout:             30,
		FailureThreshold:    0.6,
		SuccessThreshold:    2,
		HalfOpenMaxRequests: 2,
	}

	cb := NewCircuitBreaker(opts)

	for i := 0; i < 5; i++ {
		err := cb.Execute(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Expected no error in closed state, got %v", err)
		}
	}

	if cb.State() != StateClosed {
		t.Errorf("Expected state to be closed, got %s", cb.State())
	}
}

func TestCircuitBreakerOpenState(t *testing.T) {
	opts := Options{
		Name:                "test-open",
		MaxRequests:         3,
		Interval:            10,
		Timeout:             30,
		FailureThreshold:    0.5,
		SuccessThreshold:    2,
		HalfOpenMaxRequests: 1,
	}

	cb := NewCircuitBreaker(opts)

	for i := 0; i < 5; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("error")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("Expected state to be open after failures, got %s", cb.State())
	}

	err := cb.Execute(func() error {
		return nil
	})

	if err != ErrOpenState {
		t.Errorf("Expected ErrOpenState when circuit is open, got %v", err)
	}
}

func TestCircuitBreakerHalfOpenState(t *testing.T) {
	opts := Options{
		Name:                "test-half-open",
		MaxRequests:         2,
		Interval:            1,
		Timeout:             1,
		FailureThreshold:    0.5,
		SuccessThreshold:    2,
		HalfOpenMaxRequests: 2,
	}

	cb := NewCircuitBreaker(opts)

	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("error")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("Expected state to be open, got %s", cb.State())
	}

	time.Sleep(1100 * time.Millisecond)

	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected nil error after timeout, got %v", err)
	}

	if cb.State() != StateHalfOpen {
		t.Errorf("Expected state to be half_open after successful request in open state, got %s", cb.State())
	}
}

func TestCircuitBreakerRecovery(t *testing.T) {
	opts := Options{
		Name:                "test-recovery",
		MaxRequests:         2,
		Interval:            1,
		Timeout:             1,
		FailureThreshold:    0.5,
		SuccessThreshold:    2,
		HalfOpenMaxRequests: 2,
	}

	cb := NewCircuitBreaker(opts)

	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("error")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("Expected state to be open, got %s", cb.State())
	}

	time.Sleep(1500 * time.Millisecond)

	cb.Execute(func() error {
		return nil
	})

	if cb.State() != StateHalfOpen {
		t.Errorf("Expected state to be half_open, got %s", cb.State())
	}

	cb.Execute(func() error {
		return nil
	})

	if cb.State() != StateClosed {
		t.Errorf("Expected state to be closed after recovery, got %s", cb.State())
	}
}

func TestCircuitBreakerCounts(t *testing.T) {
	opts := Options{
		Name:                "test-counts",
		MaxRequests:         5,
		Interval:            10,
		Timeout:             30,
		FailureThreshold:    0.5,
		SuccessThreshold:    2,
		HalfOpenMaxRequests: 3,
	}

	cb := NewCircuitBreaker(opts)

	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return nil
		})
	}

	counts := cb.Counts()

	if counts.TotalSuccesses != 3 {
		t.Errorf("Expected 3 total successes, got %d", counts.TotalSuccesses)
	}

	if counts.ConsecutiveSuccesses != 3 {
		t.Errorf("Expected 3 consecutive successes, got %d", counts.ConsecutiveSuccesses)
	}

	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("error")
		})
	}

	counts = cb.Counts()

	if counts.TotalFailures != 2 {
		t.Errorf("Expected 2 total failures, got %d", counts.TotalFailures)
	}

	if counts.ConsecutiveSuccesses != 0 {
		t.Errorf("Expected 0 consecutive successes after failures, got %d", counts.ConsecutiveSuccesses)
	}

	if counts.ConsecutiveFailures != 2 {
		t.Errorf("Expected 2 consecutive failures, got %d", counts.ConsecutiveFailures)
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	opts := Options{
		Name:                "test-reset",
		MaxRequests:         3,
		Interval:            10,
		Timeout:             30,
		FailureThreshold:    0.5,
		SuccessThreshold:    2,
		HalfOpenMaxRequests: 1,
	}

	cb := NewCircuitBreaker(opts)

	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("error")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("Expected state to be open, got %s", cb.State())
	}

	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("Expected state to be closed after reset, got %s", cb.State())
	}

	counts := cb.Counts()
	if counts.Requests != 0 {
		t.Errorf("Expected requests to be 0 after reset, got %d", counts.Requests)
	}
}

func TestCircuitBreakerCanAttempt(t *testing.T) {
	opts := Options{
		Name:                "test-can-attempt",
		MaxRequests:         2,
		Interval:            10,
		Timeout:             30,
		FailureThreshold:    0.5,
		SuccessThreshold:    2,
		HalfOpenMaxRequests: 1,
	}

	cb := NewCircuitBreaker(opts)

	if !cb.CanAttempt() {
		t.Error("Expected CanAttempt to return true in closed state")
	}

	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("error")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("Expected state to be open, got %s", cb.State())
	}

	if cb.CanAttempt() {
		t.Error("Expected CanAttempt to return false in open state")
	}
}

func TestCircuitBreakerManager(t *testing.T) {
	defaultOpts := Options{
		Name:                "default",
		MaxRequests:         5,
		Interval:            10,
		Timeout:             30,
		FailureThreshold:    0.5,
		SuccessThreshold:    2,
		HalfOpenMaxRequests: 3,
	}

	manager := NewManager(defaultOpts)

	cb1 := manager.Get("service1")
	if cb1 == nil {
		t.Error("Expected circuit breaker to be created for service1")
	}

	cb2 := manager.Get("service1")
	if cb1 != cb2 {
		t.Error("Expected same circuit breaker instance for same service")
	}

	opts := Options{
		Name:                "service2",
		MaxRequests:         10,
		FailureThreshold:    0.3,
	}

	cb3 := manager.Register("service2", opts)
	if cb3 == nil {
		t.Error("Expected circuit breaker to be registered for service2")
	}

	states := manager.GetAllStates()
	if len(states) != 2 {
		t.Errorf("Expected 2 states, got %d", len(states))
	}

	manager.ResetAll()
	for name, state := range manager.GetAllStates() {
		if state != StateClosed {
			t.Errorf("Expected state to be closed for %s after ResetAll, got %s", name, state)
		}
	}
}

func TestCircuitBreakerContext(t *testing.T) {
	opts := Options{
		Name:                "test-context",
		MaxRequests:         3,
		Interval:            10,
		Timeout:             30,
		FailureThreshold:    0.5,
		SuccessThreshold:    2,
		HalfOpenMaxRequests: 2,
	}

	cb := NewCircuitBreaker(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := cb.Execute(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	})

	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	elapsed := time.Since(start)
	if elapsed < 100*time.Millisecond {
		t.Errorf("Expected at least 100ms elapsed, got %v", elapsed)
	}
}

func TestCircuitBreakerMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()

	collector.RecordSuccess()
	collector.RecordSuccess()
	collector.RecordFailure()
	collector.RecordStateChange("closed", "open")
	collector.RecordStateChange("open", "half_open")
	collector.RecordStateChange("half_open", "closed")

	metrics := collector.GetMetrics()

	if metrics["total_successes"].(int64) != 2 {
		t.Errorf("Expected 2 total successes, got %v", metrics["total_successes"])
	}

	if metrics["total_failures"].(int64) != 1 {
		t.Errorf("Expected 1 total failures, got %v", metrics["total_failures"])
	}

	if metrics["total_opens"].(int64) != 1 {
		t.Errorf("Expected 1 total opens, got %v", metrics["total_opens"])
	}

	if metrics["total_half_opens"].(int64) != 1 {
		t.Errorf("Expected 1 total half_opens, got %v", metrics["total_half_opens"])
	}

	if metrics["total_closes"].(int64) != 1 {
		t.Errorf("Expected 1 total closes, got %v", metrics["total_closes"])
	}
}

func TestCircuitBreakerStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateHalfOpen, "half_open"},
		{StateOpen, "open"},
		{State(100), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("State(%d).String() = %v, want %v", tt.state, got, tt.expected)
		}
	}
}

func TestCircuitBreakerOnCallbacks(t *testing.T) {
	opts := Options{
		Name:                "test-callbacks",
		MaxRequests:         3,
		Interval:            10,
		Timeout:             30,
		FailureThreshold:    0.5,
		SuccessThreshold:    2,
		HalfOpenMaxRequests: 2,
	}

	cb := NewCircuitBreaker(opts)

	successCalled := false
	failureCalled := false
	stateChangeCalled := false

	cb.OnSuccess(func() {
		successCalled = true
	})

	cb.OnFailure(func() {
		failureCalled = true
	})

	cb.OnStateChange(func(name, from, to string) {
		stateChangeCalled = true
	})

	cb.Execute(func() error {
		return nil
	})

	if !successCalled {
		t.Error("Expected success callback to be called")
	}

	cb.Execute(func() error {
		return fmt.Errorf("error")
	})

	if !failureCalled {
		t.Error("Expected failure callback to be called")
	}

	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("error")
		})
	}

	if !stateChangeCalled {
		t.Error("Expected state change callback to be called")
	}
}

func BenchmarkCircuitBreakerExecute(b *testing.B) {
	opts := Options{
		Name:                "benchmark",
		MaxRequests:         5,
		Interval:            10,
		Timeout:             30,
		FailureThreshold:    0.5,
		SuccessThreshold:    2,
		HalfOpenMaxRequests: 3,
	}

	cb := NewCircuitBreaker(opts)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Execute(func() error {
			return nil
		})
	}
}
