package circuitbreaker

import (
	"sync"
	"testing"
	"time"
)

func BenchmarkCircuitBreakerAllow(b *testing.B) {
	cb := New("test", Config{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Allow()
		}
	})
}

func BenchmarkCircuitBreakerRecordSuccess(b *testing.B) {
	cb := New("test", Config{
		FailureThreshold: 5,
		Timeout:          30 * time.Second,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.RecordSuccess()
		}
	})
}

func BenchmarkCircuitBreakerRecordFailure(b *testing.B) {
	cb := New("test", Config{
		FailureThreshold: 5,
		Timeout:          30 * time.Second,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.RecordFailure()
		}
	})
}

func TestCircuitBreakerStateTransitions(t *testing.T) {
	cb := New("test", Config{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
		HalfOpenMaxReqs:  2,
	})

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	if cb.State() != StateOpen {
		t.Errorf("expected state Open, got %s", cb.State())
	}

	time.Sleep(1100 * time.Millisecond)

	allowed, _ := cb.Allow()
	if !allowed {
		t.Error("expected request to be allowed in half-open state")
	}

	if cb.State() != StateHalfOpen {
		t.Errorf("expected state HalfOpen, got %s", cb.State())
	}
}

func TestCircuitBreakerRecovery(t *testing.T) {
	cb := New("test", Config{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
	})

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	time.Sleep(1100 * time.Millisecond)

	cb.Allow()
	cb.RecordSuccess()
	cb.Allow()
	cb.RecordSuccess()

	if cb.State() != StateClosed {
		t.Errorf("expected state Closed after recovery, got %s", cb.State())
	}
}

func TestCircuitBreakerOpenOnFailure(t *testing.T) {
	cb := New("test", Config{
		FailureThreshold: 2,
		Timeout:          1 * time.Second,
	})

	cb.RecordSuccess()
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != StateOpen {
		t.Errorf("expected state Open, got %s", cb.State())
	}

	allowed, err := cb.Allow()
	if allowed {
		t.Error("expected request to be rejected when circuit is open")
	}
	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreakerHalfOpenToOpen(t *testing.T) {
	cb := New("test", Config{
		FailureThreshold: 2,
		SuccessThreshold: 3,
		Timeout:          100 * time.Millisecond,
		HalfOpenMaxReqs:  1,
	})

	for i := 0; i < 2; i++ {
		cb.RecordFailure()
	}

	time.Sleep(150 * time.Millisecond)

	cb.Allow()
	cb.RecordFailure()

	if cb.State() != StateOpen {
		t.Errorf("expected state Open after failure in half-open, got %s", cb.State())
	}
}

func TestCircuitBreakerStats(t *testing.T) {
	cb := New("test", Config{
		FailureThreshold: 5,
		Timeout:          30 * time.Second,
	})

	for i := 0; i < 10; i++ {
		cb.Allow()
		cb.RecordSuccess()
	}

	for i := 0; i < 5; i++ {
		cb.Allow()
		cb.RecordFailure()
	}

	stats := cb.Stats()
	if stats.Requests != 15 {
		t.Errorf("expected 15 requests, got %d", stats.Requests)
	}
	if stats.Successes != 10 {
		t.Errorf("expected 10 successes, got %d", stats.Successes)
	}
	if stats.Failures != 5 {
		t.Errorf("expected 5 failures, got %d", stats.Failures)
	}
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	cb := New("test", Config{
		FailureThreshold: 100,
		Timeout:          30 * time.Second,
	})

	var wg sync.WaitGroup
	concurrency := 100

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cb.Allow()
				if j%2 == 0 {
					cb.RecordSuccess()
				} else {
					cb.RecordFailure()
				}
			}
		}()
	}

	wg.Wait()

	stats := cb.Stats()
	if stats.Requests != int64(concurrency*100) {
		t.Errorf("expected %d requests, got %d", concurrency*100, stats.Requests)
	}
}

func BenchmarkTokenBucketLimiter(b *testing.B) {
	limiter := NewTokenBucketLimiter(1000, 100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow()
		}
	})
}

func BenchmarkSlidingWindowLimiter(b *testing.B) {
	limiter := NewSlidingWindowLimiter(time.Minute, 1000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow()
		}
	})
}

func TestTokenBucketLimiter(t *testing.T) {
	limiter := NewTokenBucketLimiter(10, 1)

	for i := 0; i < 10; i++ {
		if !limiter.Allow() {
			t.Errorf("request %d should be allowed", i)
		}
	}

	if limiter.Allow() {
		t.Error("request should be rejected after bucket is empty")
	}
}

func TestTokenBucketLimiterRefill(t *testing.T) {
	limiter := NewTokenBucketLimiter(5, 10)

	for i := 0; i < 5; i++ {
		limiter.Allow()
	}

	if limiter.Allow() {
		t.Error("bucket should be empty")
	}

	time.Sleep(200 * time.Millisecond)

	if !limiter.Allow() {
		t.Error("bucket should have refilled")
	}
}

func TestSlidingWindowLimiter(t *testing.T) {
	limiter := NewSlidingWindowLimiter(time.Second, 1000)

	for i := 0; i < 100; i++ {
		limiter.Allow()
	}

	stats := limiter.Stats()
	if stats.Allowed == 0 {
		t.Error("expected some requests to be allowed")
	}
}

func TestSlidingWindowLimiterReset(t *testing.T) {
	limiter := NewSlidingWindowLimiter(100*time.Millisecond, 1000)

	for i := 0; i < 10; i++ {
		limiter.Allow()
	}

	time.Sleep(150 * time.Millisecond)

	for i := 0; i < 10; i++ {
		limiter.Allow()
	}

	stats := limiter.Stats()
	if stats.Allowed == 0 {
		t.Error("expected some requests to be allowed")
	}
}

func TestAdaptiveRateLimiter(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(1000, 10, 10000)

	for i := 0; i < 200; i++ {
		limiter.Allow()
	}

	stats := limiter.Stats()
	if stats.Allowed < 200 {
		t.Errorf("expected at least 200 allowed, got %d", stats.Allowed)
	}
}

func BenchmarkLimiterRegistry(b *testing.B) {
	limiter := NewTokenBucketLimiter(1000, 100)
	RegisterLimiter("test", limiter)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetOrCreateLimiter("test", 1000, 100)
	}
}

func TestLimiterRegistry(t *testing.T) {
	limiter := NewTokenBucketLimiter(100, 10)
	RegisterLimiter("test_reg", limiter)

	retrieved, ok := GetLimiter("test_reg")
	if !ok {
		t.Error("expected limiter to be found")
	}
	if retrieved != limiter {
		t.Error("expected retrieved limiter to match original")
	}

	newLimiter := GetOrCreateLimiter("test_reg", 200, 20)
	if newLimiter != limiter {
		t.Error("expected existing limiter to be returned")
	}

	newLimiter = GetOrCreateLimiter("new_reg", 300, 30)
	if newLimiter == limiter {
		t.Error("expected new limiter to be created")
	}
}

func TestLimiterStats(t *testing.T) {
	limiter := NewTokenBucketLimiter(1000, 100)

	for i := 0; i < 50; i++ {
		limiter.Allow()
	}
	for i := 0; i < 20; i++ {
		limiter.AllowN(10)
	}

	stats := limiter.Stats()
	if stats.Allowed < 70 {
		t.Errorf("expected at least 70 allowed, got %d", stats.Allowed)
	}
}

func BenchmarkCircuitBreakerFullCycle(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb := New("bench", Config{
			FailureThreshold: 5,
			Timeout:          30 * time.Second,
		})

		for j := 0; j < 10; j++ {
			cb.Allow()
			cb.RecordSuccess()
		}
	}
}

func BenchmarkConcurrentCircuitBreakers(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		cb := New("concurrent", Config{
			FailureThreshold: 5,
			Timeout:          30 * time.Second,
		})

		for pb.Next() {
			cb.Allow()
			cb.RecordSuccess()
		}
	})
}

func TestCircuitBreakerString(t *testing.T) {
	cb := New("test_string", Config{
		FailureThreshold: 5,
		Timeout:          30 * time.Second,
	})

	str := cb.String()
	if str != "circuit_breaker{name=test_string, state=closed}" {
		t.Errorf("unexpected string: %s", str)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig{}
	if cfg.FailureThreshold() != 5 {
		t.Errorf("expected 5 failure threshold, got %d", cfg.FailureThreshold())
	}
	if cfg.SuccessThreshold() != 2 {
		t.Errorf("expected 2 success threshold, got %d", cfg.SuccessThreshold())
	}
	if cfg.Timeout() != 30*time.Second {
		t.Errorf("expected 30s timeout, got %s", cfg.Timeout())
	}
	if cfg.HalfOpenMaxReqs() != 3 {
		t.Errorf("expected 3 half-open max requests, got %d", cfg.HalfOpenMaxReqs())
	}
}
