package benchmark

import (
	"testing"
	"time"

	"captchax/internal/resilience"
)

func BenchmarkTokenBucketAllow(b *testing.B) {
	tb := resilience.NewTokenBucket(1000, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.Allow()
	}
}

func BenchmarkTokenBucketAllowN(b *testing.B) {
	tb := resilience.NewTokenBucket(1000, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.AllowN(5)
	}
}

func BenchmarkSlidingWindowLimiter(b *testing.B) {
	swl := resilience.NewSlidingWindowLimiter(1*time.Second, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		swl.Allow()
	}
}

func BenchmarkCircuitBreakerAllow(b *testing.B) {
	cb := resilience.NewCircuitBreaker(5, 2, 30*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Allow()
	}
}

func BenchmarkCircuitBreakerRecordSuccess(b *testing.B) {
	cb := resilience.NewCircuitBreaker(5, 2, 30*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.RecordSuccess()
	}
}

func BenchmarkCircuitBreakerRecordFailure(b *testing.B) {
	cb := resilience.NewCircuitBreaker(5, 2, 30*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.RecordFailure()
	}
}
