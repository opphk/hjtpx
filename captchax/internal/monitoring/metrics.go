package monitoring

import (
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	requestsTotal   int64
	requestsSuccess int64
	requestsFailed  int64
	requestDuration *Histogram
	cacheHit        int64
	cacheMiss       int64
	mu              sync.Mutex
	lastSnapshotTime time.Time
	snapshotCache    MetricsSnapshot
}

type Histogram struct {
	counts [11]int64
	mu     sync.Mutex
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

func (h *Histogram) GetPercentiles(p50, p90, p99, p999 *float64) {
	counts := h.GetCounts()
	total := int64(0)
	for _, c := range counts {
		total += c
	}

	if total == 0 {
		return
	}

	targets := []struct {
		percent float64
		result  *float64
	}{
		{0.50, p50},
		{0.90, p90},
		{0.99, p99},
		{0.999, p999},
	}

	for _, target := range targets {
		cumulative := int64(0)
		threshold := int64(float64(total) * target.percent)
		for i, count := range counts {
			cumulative += count
			if cumulative >= threshold {
				*target.result = float64((i + 1) * 10)
				break
			}
		}
	}
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
	P50Latency    float64
	P90Latency    float64
	P99Latency    float64
	P999Latency   float64
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

	m.mu.Lock()
	if time.Since(m.lastSnapshotTime) < 100*time.Millisecond && m.snapshotCache.RequestsTotal == total {
		m.mu.Unlock()
		return m.snapshotCache
	}

	snapshot := MetricsSnapshot{
		RequestsTotal: total,
		SuccessRate:  float64(success) / float64(total),
		CacheHitRate: float64(hit) / float64(cacheTotal),
	}

	m.requestDuration.GetPercentiles(&snapshot.P50Latency, &snapshot.P90Latency, &snapshot.P99Latency, &snapshot.P999Latency)

	m.snapshotCache = snapshot
	m.lastSnapshotTime = time.Now()
	m.mu.Unlock()

	return snapshot
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

func (m *Metrics) Reset() {
	atomic.StoreInt64(&m.requestsTotal, 0)
	atomic.StoreInt64(&m.requestsSuccess, 0)
	atomic.StoreInt64(&m.requestsFailed, 0)
	atomic.StoreInt64(&m.cacheHit, 0)
	atomic.StoreInt64(&m.cacheMiss, 0)

	if m.requestDuration != nil {
		for i := 0; i < 11; i++ {
			atomic.StoreInt64(&m.requestDuration.counts[i], 0)
		}
	}

	m.mu.Lock()
	m.lastSnapshotTime = time.Time{}
	m.mu.Unlock()
}

func (m *Metrics) GetRequestsTotal() int64 {
	return atomic.LoadInt64(&m.requestsTotal)
}

func (m *Metrics) GetRequestsSuccess() int64 {
	return atomic.LoadInt64(&m.requestsSuccess)
}

func (m *Metrics) GetRequestsFailed() int64 {
	return atomic.LoadInt64(&m.requestsFailed)
}

func (m *Metrics) GetCacheHit() int64 {
	return atomic.LoadInt64(&m.cacheHit)
}

func (m *Metrics) GetCacheMiss() int64 {
	return atomic.LoadInt64(&m.cacheMiss)
}
