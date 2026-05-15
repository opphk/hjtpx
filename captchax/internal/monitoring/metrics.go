package monitoring

import (
	"sync/atomic"
	"time"
)

// Metrics provides atomic-based metrics tracking for high-performance concurrent scenarios.
// Optimizations:
// - Uses atomic operations instead of mutex locks for all counter fields
// - Reduces lock contention significantly in high-throughput environments
type Metrics struct {
	requestsTotal   int64
	requestsSuccess int64
	requestsFailed  int64
	requestDuration *Histogram
	cacheHit        int64
	cacheMiss       int64
}

// Histogram uses atomic operations for lock-free observations.
type Histogram struct {
	counts [11]int64
}

func (h *Histogram) Observe(duration time.Duration) {
	ms := duration.Milliseconds()
	bucket := min(int(ms/10), 10)

	atomic.AddInt64(&h.counts[bucket], 1)
}

func (h *Histogram) GetCounts() [11]int64 {
	var counts [11]int64
	for i := 0; i < 11; i++ {
		counts[i] = atomic.LoadInt64(&h.counts[i])
	}
	return counts
}

func (h *Histogram) GetBucket(index int) int64 {
	if index < 0 || index >= 11 {
		return 0
	}
	return atomic.LoadInt64(&h.counts[index])
}

func (m *Metrics) RecordRequest(duration time.Duration, success bool) {
	atomic.AddInt64(&m.requestsTotal, 1)
	if success {
		atomic.AddInt64(&m.requestsSuccess, 1)
	} else {
		atomic.AddInt64(&m.requestsFailed, 1)
	}

	m.requestDuration.Observe(duration)
}

func (m *Metrics) RecordCacheHit() {
	atomic.AddInt64(&m.cacheHit, 1)
}

func (m *Metrics) RecordCacheMiss() {
	atomic.AddInt64(&m.cacheMiss, 1)
}

type MetricsSnapshot struct {
	RequestsTotal int64
	SuccessRate   float64
	AvgDuration   time.Duration
	CacheHitRate  float64
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	total := atomic.LoadInt64(&m.requestsTotal)
	if total == 0 {
		return MetricsSnapshot{}
	}

	success := atomic.LoadInt64(&m.requestsSuccess)
	hit := atomic.LoadInt64(&m.cacheHit)
	miss := atomic.LoadInt64(&m.cacheMiss)
	cacheTotal := hit + miss

	return MetricsSnapshot{
		RequestsTotal: total,
		SuccessRate:  float64(success) / float64(total),
		CacheHitRate: float64(hit) / float64(cacheTotal),
	}
}

func NewMetrics() *Metrics {
	return &Metrics{
		requestDuration: &Histogram{},
	}
}

func (m *Metrics) GetResponseTimeDistribution() []int64 {
	if m.requestDuration == nil {
		counts := make([]int64, 11)
		return counts
	}
	counts := m.requestDuration.GetCounts()
	return counts[:]
}

func (m *Metrics) GetHistogram() *Histogram {
	return m.requestDuration
}
