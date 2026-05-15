package monitoring

import (
	"sync"
	"time"
)

type Metrics struct {
	requestsTotal   int64
	requestsSuccess int64
	requestsFailed  int64
	requestDuration *Histogram
	cacheHit        int64
	cacheMiss       int64
	mu              sync.RWMutex
}

type Histogram struct {
	counts [11]int64
	mu     sync.RWMutex
}

func (h *Histogram) Observe(duration time.Duration) {
	ms := duration.Milliseconds()
	bucket := min(int(ms/10), 10)

	h.mu.Lock()
	h.counts[bucket]++
	h.mu.Unlock()
}

func (m *Metrics) RecordRequest(duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestsTotal++
	if success {
		m.requestsSuccess++
	} else {
		m.requestsFailed++
	}

	m.requestDuration.Observe(duration)
}

func (m *Metrics) RecordCacheHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheHit++
}

func (m *Metrics) RecordCacheMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheMiss++
}

type MetricsSnapshot struct {
	RequestsTotal int64
	SuccessRate   float64
	AvgDuration   time.Duration
	CacheHitRate  float64
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.requestsTotal
	if total == 0 {
		return MetricsSnapshot{}
	}

	cacheTotal := m.cacheHit + m.cacheMiss

	return MetricsSnapshot{
		RequestsTotal: total,
		SuccessRate:  float64(m.requestsSuccess) / float64(total),
		CacheHitRate: float64(m.cacheHit) / float64(cacheTotal),
	}
}

func NewMetrics() *Metrics {
	return &Metrics{
		requestDuration: &Histogram{},
	}
}
