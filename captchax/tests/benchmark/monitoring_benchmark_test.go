package benchmark

import (
	"testing"
	"time"

	"captchax/internal/monitoring"
)

func BenchmarkMetricsRecordRequest(b *testing.B) {
	m := monitoring.NewMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordRequest(100*time.Millisecond, true)
	}
}

func BenchmarkMetricsSnapshot(b *testing.B) {
	m := monitoring.NewMetrics()
	for i := 0; i < 1000; i++ {
		m.RecordRequest(100*time.Millisecond, true)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Snapshot()
	}
}

func BenchmarkHistogramObserve(b *testing.B) {
	h := &monitoring.Histogram{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Observe(50 * time.Millisecond)
	}
}
