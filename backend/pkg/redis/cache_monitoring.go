package redis

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheMetrics struct {
	TotalOperations   int64
	ReadOperations     int64
	WriteOperations    int64
	DeleteOperations   int64
	HitCount          int64
	MissCount         int64
	ErrorCount        int64
	TotalLatency      int64
	AvgLatency        float64
	MinLatency        int64
	MaxLatency        int64
	LastUpdated       time.Time
}

type CacheMonitor struct {
	client *redis.Client
	metrics *CacheMetrics
	alertThresholds *AlertThresholds
	stopChan chan struct{}
	mu sync.RWMutex
	avgLatencyMu sync.Mutex
}

type AlertThresholds struct {
	HitRateThreshold      float64
	ErrorRateThreshold   float64
	LatencyThresholdMs   int64
	MemoryThresholdMB    int64
	ConnectionThreshold   int
}

type MonitoringConfig struct {
	CollectInterval time.Duration
	ReportInterval   time.Duration
	EnableAlerts     bool
	AlertWebhook     string
}

var globalMonitor *CacheMonitor

func NewCacheMonitor(client *redis.Client) *CacheMonitor {
	if globalMonitor == nil {
		globalMonitor = &CacheMonitor{
			client: client,
			metrics: &CacheMetrics{
				LastUpdated: time.Now(),
				MinLatency:  1<<63 - 1,
			},
			alertThresholds: &AlertThresholds{
				HitRateThreshold:    80.0,
				ErrorRateThreshold:  5.0,
				LatencyThresholdMs:  100,
				MemoryThresholdMB:   1024,
				ConnectionThreshold: 100,
			},
			stopChan: make(chan struct{}),
		}
	}
	return globalMonitor
}

func (m *CacheMonitor) Start(ctx context.Context) error {
	go m.collectMetrics(ctx)
	go m.checkAlerts(ctx)
	go m.reportMetrics(ctx)

	log.Println("Cache monitor started")
	return nil
}

func (m *CacheMonitor) Stop() {
	close(m.stopChan)
	log.Println("Cache monitor stopped")
}

func (m *CacheMonitor) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.collectOnce(ctx)
		}
	}
}

func (m *CacheMonitor) collectOnce(ctx context.Context) {
	info, err := m.client.Info(ctx, "stats", "memory", "clients").Result()
	if err != nil {
		atomic.AddInt64(&m.metrics.ErrorCount, 1)
		return
	}

	stats := m.client.PoolStats()

	m.mu.Lock()
	m.metrics.HitCount = atomic.LoadInt64(&m.metrics.HitCount)
	m.metrics.MissCount = atomic.LoadInt64(&m.metrics.MissCount)
	m.metrics.TotalOperations = atomic.LoadInt64(&m.metrics.TotalOperations)
	m.metrics.ErrorCount = atomic.LoadInt64(&m.metrics.ErrorCount)
	m.mu.Unlock()

	_ = stats

	logLines := splitLines(info)
	for _, line := range logLines {
		parts := splitKeyValue(line)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]

		switch key {
		case "keyspace_hits":
			var hits int64
			fmt.Sscanf(value, "%d", &hits)
			atomic.StoreInt64(&m.metrics.HitCount, hits)
		case "keyspace_misses":
			var misses int64
			fmt.Sscanf(value, "%d", &misses)
			atomic.StoreInt64(&m.metrics.MissCount, misses)
		}
	}

	m.mu.Lock()
	m.metrics.LastUpdated = time.Now()
	m.mu.Unlock()
}

func (m *CacheMonitor) checkAlerts(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkAlertConditions(ctx)
		}
	}
}

func (m *CacheMonitor) checkAlertConditions(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := m.metrics

	totalOps := metrics.HitCount + metrics.MissCount
	if totalOps == 0 {
		return
	}

	hitRate := float64(metrics.HitCount) / float64(totalOps) * 100
	if hitRate < m.alertThresholds.HitRateThreshold {
		log.Printf("[ALERT] Cache hit rate is below threshold: %.2f%% < %.2f%%", hitRate, m.alertThresholds.HitRateThreshold)
	}

	if metrics.ErrorCount > 0 {
		errorRate := float64(metrics.ErrorCount) / float64(totalOps) * 100
		if errorRate > m.alertThresholds.ErrorRateThreshold {
			log.Printf("[ALERT] Cache error rate is above threshold: %.2f%% > %.2f%%", errorRate, m.alertThresholds.ErrorRateThreshold)
		}
	}

	avgLatency := metrics.AvgLatency
	if avgLatency > float64(m.alertThresholds.LatencyThresholdMs) {
		log.Printf("[ALERT] Average cache latency is above threshold: %.2fms > %dms", avgLatency, m.alertThresholds.LatencyThresholdMs)
	}
}

func (m *CacheMonitor) reportMetrics(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.logMetrics()
		}
	}
}

func (m *CacheMonitor) logMetrics() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := m.metrics

	totalOps := metrics.HitCount + metrics.MissCount
	hitRate := 0.0
	if totalOps > 0 {
		hitRate = float64(metrics.HitCount) / float64(totalOps) * 100
	}

	log.Printf("[CACHE METRICS] Operations: %d, Hits: %d, Misses: %d, HitRate: %.2f%%, AvgLatency: %.2fms, Errors: %d",
		metrics.TotalOperations,
		metrics.HitCount,
		metrics.MissCount,
		hitRate,
		metrics.AvgLatency,
		metrics.ErrorCount)
}

func (m *CacheMonitor) RecordHit() {
	atomic.AddInt64(&m.metrics.HitCount, 1)
	atomic.AddInt64(&m.metrics.TotalOperations, 1)
}

func (m *CacheMonitor) RecordMiss() {
	atomic.AddInt64(&m.metrics.MissCount, 1)
	atomic.AddInt64(&m.metrics.TotalOperations, 1)
}

func (m *CacheMonitor) RecordRead(latencyMs int64) {
	atomic.AddInt64(&m.metrics.ReadOperations, 1)
	atomic.AddInt64(&m.metrics.TotalOperations, 1)
	m.recordLatency(latencyMs)
}

func (m *CacheMonitor) RecordWrite(latencyMs int64) {
	atomic.AddInt64(&m.metrics.WriteOperations, 1)
	atomic.AddInt64(&m.metrics.TotalOperations, 1)
	m.recordLatency(latencyMs)
}

func (m *CacheMonitor) RecordDelete() {
	atomic.AddInt64(&m.metrics.DeleteOperations, 1)
	atomic.AddInt64(&m.metrics.TotalOperations, 1)
}

func (m *CacheMonitor) RecordError() {
	atomic.AddInt64(&m.metrics.ErrorCount, 1)
}

func (m *CacheMonitor) recordLatency(latencyMs int64) {
	atomic.AddInt64(&m.metrics.TotalLatency, latencyMs)

	totalOps := atomic.LoadInt64(&m.metrics.TotalOperations)
	if totalOps > 0 {
		avgLatency := float64(atomic.LoadInt64(&m.metrics.TotalLatency)) / float64(totalOps)
		m.avgLatencyMu.Lock()
		m.metrics.AvgLatency = avgLatency
		m.avgLatencyMu.Unlock()
	}

	currentMin := atomic.LoadInt64(&m.metrics.MinLatency)
	if latencyMs < currentMin {
		atomic.StoreInt64(&m.metrics.MinLatency, latencyMs)
	}

	currentMax := atomic.LoadInt64(&m.metrics.MaxLatency)
	if latencyMs > currentMax {
		atomic.StoreInt64(&m.metrics.MaxLatency, latencyMs)
	}
}

func (m *CacheMonitor) GetMetrics() *CacheMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := &CacheMetrics{}
	*metrics = *m.metrics

	metrics.HitCount = atomic.LoadInt64(&m.metrics.HitCount)
	metrics.MissCount = atomic.LoadInt64(&m.metrics.MissCount)
	metrics.TotalOperations = atomic.LoadInt64(&m.metrics.TotalOperations)
	metrics.ErrorCount = atomic.LoadInt64(&m.metrics.ErrorCount)
	metrics.ReadOperations = atomic.LoadInt64(&m.metrics.ReadOperations)
	metrics.WriteOperations = atomic.LoadInt64(&m.metrics.WriteOperations)
	metrics.DeleteOperations = atomic.LoadInt64(&m.metrics.DeleteOperations)
	m.avgLatencyMu.Lock()
	metrics.AvgLatency = m.metrics.AvgLatency
	m.avgLatencyMu.Unlock()
	metrics.MinLatency = atomic.LoadInt64(&m.metrics.MinLatency)
	metrics.MaxLatency = atomic.LoadInt64(&m.metrics.MaxLatency)

	return metrics
}

func (m *CacheMonitor) SetAlertThresholds(thresholds *AlertThresholds) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.alertThresholds = thresholds
}

func (m *CacheMonitor) GetAlertThresholds() *AlertThresholds {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.alertThresholds
}

func (m *CacheMonitor) ResetMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics = &CacheMetrics{
		LastUpdated: time.Now(),
		MinLatency:  1<<63 - 1,
	}
}


