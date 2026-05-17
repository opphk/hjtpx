package redis

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

type CacheMetricsCollector struct {
	stats            *CacheStats
	hotKeys          *sync.Map
	keyAccessCount   *sync.Map
	queryLatency     *LatencyHistogram
	memoryUsage      *MemoryStats
	mu               sync.RWMutex
	started          time.Time
}

type LatencyHistogram struct {
	buckets []time.Duration
	counts  []int64
	mu      sync.RWMutex
}

type MemoryStats struct {
	L1Size       int64
	L2Keys       int64
	L2MemoryUsed int64
}

type DetailedMetrics struct {
	Uptime            time.Duration                `json:"uptime"`
	HitRate           float64                      `json:"hit_rate"`
	TotalRequests     int64                        `json:"total_requests"`
	Hits              int64                        `json:"hits"`
	Misses            int64                        `json:"misses"`
	L1Hits            int64                        `json:"l1_hits"`
	L1Misses          int64                        `json:"l1_misses"`
	L2Hits            int64                        `json:"l2_hits"`
	L2Misses          int64                        `json:"l2_misses"`
	Sets              int64                        `json:"sets"`
	Deletes           int64                        `json:"deletes"`
	Errors            int64                        `json:"errors"`
	AverageLatency    time.Duration                `json:"average_latency"`
	P95Latency        time.Duration                `json:"p95_latency"`
	P99Latency        time.Duration                `json:"p99_latency"`
	L1HitRate         float64                      `json:"l1_hit_rate"`
	L2HitRate         float64                      `json:"l2_hit_rate"`
	CompressedCount   int64                        `json:"compressed_count"`
	DecompressedCount int64                        `json:"decompressed_count"`
	HotKeys           []HotKeyMetric               `json:"hot_keys"`
	MemoryUsage       MemoryStats                  `json:"memory_usage"`
	LatencyDistribution map[string]int64           `json:"latency_distribution"`
}

type HotKeyMetric struct {
	Key         string        `json:"key"`
	AccessCount int64         `json:"access_count"`
	LastAccess  time.Time     `json:"last_access"`
}

func NewCacheMetricsCollector() *CacheMetricsCollector {
	cmc := &CacheMetricsCollector{
		stats:          &CacheStats{},
		hotKeys:        &sync.Map{},
		keyAccessCount: &sync.Map{},
		queryLatency:   NewLatencyHistogram(),
		memoryUsage:    &MemoryStats{},
		started:        time.Now(),
	}
	return cmc
}

func NewLatencyHistogram() *LatencyHistogram {
	return &LatencyHistogram{
		buckets: []time.Duration{
			time.Microsecond,
			10 * time.Microsecond,
			100 * time.Microsecond,
			time.Millisecond,
			10 * time.Millisecond,
			100 * time.Millisecond,
			time.Second,
			10 * time.Second,
		},
		counts: make([]int64, 9),
	}
}

func (lh *LatencyHistogram) Record(duration time.Duration) {
	lh.mu.Lock()
	defer lh.mu.Unlock()

	for i, bucket := range lh.buckets {
		if duration <= bucket {
			atomic.AddInt64(&lh.counts[i], 1)
			return
		}
	}
	atomic.AddInt64(&lh.counts[len(lh.counts)-1], 1)
}

func (lh *LatencyHistogram) Percentile(p float64) time.Duration {
	lh.mu.RLock()
	defer lh.mu.RUnlock()

	total := int64(0)
	for _, count := range lh.counts {
		total += count
	}

	if total == 0 {
		return 0
	}

	target := int64(float64(total) * p)
	current := int64(0)

	for i, count := range lh.counts {
		current += count
		if current >= target {
			return lh.buckets[i]
		}
	}

	return lh.buckets[len(lh.buckets)-1]
}

func (lh *LatencyHistogram) GetDistribution() map[string]int64 {
	lh.mu.RLock()
	defer lh.mu.RUnlock()

	dist := make(map[string]int64)
	for i, bucket := range lh.buckets {
		dist[bucket.String()] = atomic.LoadInt64(&lh.counts[i])
	}
	return dist
}

func (cmc *CacheMetricsCollector) RecordHit() {
	cmc.stats.Hits.Add(1)
	cmc.stats.RequestCount.Add(1)
}

func (cmc *CacheMetricsCollector) RecordMiss() {
	cmc.stats.Misses.Add(1)
	cmc.stats.RequestCount.Add(1)
}

func (cmc *CacheMetricsCollector) RecordL1Hit() {
	cmc.stats.L1Hits.Add(1)
}

func (cmc *CacheMetricsCollector) RecordL1Miss() {
	cmc.stats.L1Misses.Add(1)
}

func (cmc *CacheMetricsCollector) RecordL2Hit() {
	cmc.stats.L2Hits.Add(1)
}

func (cmc *CacheMetricsCollector) RecordL2Miss() {
	cmc.stats.L2Misses.Add(1)
}

func (cmc *CacheMetricsCollector) RecordSet() {
	cmc.stats.Sets.Add(1)
}

func (cmc *CacheMetricsCollector) RecordDelete() {
	cmc.stats.Deletes.Add(1)
}

func (cmc *CacheMetricsCollector) RecordError() {
	cmc.stats.Errors.Add(1)
}

func (cmc *CacheMetricsCollector) RecordLatency(duration time.Duration) {
	cmc.stats.TotalLatency.Add(duration.Nanoseconds())
	cmc.queryLatency.Record(duration)
}

func (cmc *CacheMetricsCollector) RecordKeyAccess(key string) {
	val, _ := cmc.keyAccessCount.LoadOrStore(key, int64(0))
	newVal := val.(int64) + 1
	cmc.keyAccessCount.Store(key, newVal)

	cmc.hotKeys.Store(key, &HotKeyInfo{
		Key:         key,
		AccessCount: newVal,
		LastAccess:  time.Now(),
	})
}

func (cmc *CacheMetricsCollector) RecordCompressed() {
	cmc.stats.Compressed.Add(1)
}

func (cmc *CacheMetricsCollector) RecordDecompressed() {
	cmc.stats.Decompressed.Add(1)
}

func (cmc *CacheMetricsCollector) UpdateMemoryUsage(l1Size, l2Keys, l2Memory int64) {
	cmc.mu.Lock()
	defer cmc.mu.Unlock()
	cmc.memoryUsage.L1Size = l1Size
	cmc.memoryUsage.L2Keys = l2Keys
	cmc.memoryUsage.L2MemoryUsed = l2Memory
}

func (cmc *CacheMetricsCollector) GetDetailedMetrics() *DetailedMetrics {
	cmc.mu.RLock()
	defer cmc.mu.RUnlock()

	totalRequests := cmc.stats.RequestCount.Load()
	hits := cmc.stats.Hits.Load()
	misses := cmc.stats.Misses.Load()
	l1Hits := cmc.stats.L1Hits.Load()
	l1Misses := cmc.stats.L1Misses.Load()
	l2Hits := cmc.stats.L2Hits.Load()
	l2Misses := cmc.stats.L2Misses.Load()
	totalLatency := cmc.stats.TotalLatency.Load()

	var hitRate, l1HitRate, l2HitRate float64
	var avgLatency time.Duration

	if totalRequests > 0 {
		hitRate = float64(hits) / float64(totalRequests) * 100
		avgLatency = time.Duration(totalLatency / totalRequests)
	}

	if l1Hits+l1Misses > 0 {
		l1HitRate = float64(l1Hits) / float64(l1Hits+l1Misses) * 100
	}

	if l2Hits+l2Misses > 0 {
		l2HitRate = float64(l2Hits) / float64(l2Hits+l2Misses) * 100
	}

	return &DetailedMetrics{
		Uptime:            time.Since(cmc.started),
		HitRate:           hitRate,
		TotalRequests:     totalRequests,
		Hits:              hits,
		Misses:            misses,
		L1Hits:            l1Hits,
		L1Misses:          l1Misses,
		L2Hits:            l2Hits,
		L2Misses:          l2Misses,
		Sets:              cmc.stats.Sets.Load(),
		Deletes:           cmc.stats.Deletes.Load(),
		Errors:            cmc.stats.Errors.Load(),
		AverageLatency:    avgLatency,
		P95Latency:        cmc.queryLatency.Percentile(0.95),
		P99Latency:        cmc.queryLatency.Percentile(0.99),
		L1HitRate:         l1HitRate,
		L2HitRate:         l2HitRate,
		CompressedCount:   cmc.stats.Compressed.Load(),
		DecompressedCount: cmc.stats.Decompressed.Load(),
		HotKeys:           cmc.getHotKeys(),
		MemoryUsage:       *cmc.memoryUsage,
		LatencyDistribution: cmc.queryLatency.GetDistribution(),
	}
}

func (cmc *CacheMetricsCollector) getHotKeys() []HotKeyMetric {
	var hotKeys []HotKeyMetric
	cmc.hotKeys.Range(func(key, value interface{}) bool {
		info := value.(*HotKeyInfo)
		hotKeys = append(hotKeys, HotKeyMetric{
			Key:         info.Key,
			AccessCount: info.AccessCount,
			LastAccess:  info.LastAccess,
		})
		return true
	})
	return hotKeys
}

func (cmc *CacheMetricsCollector) GetHotKeys(limit int) []HotKeyMetric {
	hotKeys := cmc.getHotKeys()
	if len(hotKeys) > limit {
		hotKeys = hotKeys[:limit]
	}
	return hotKeys
}

func (cmc *CacheMetricsCollector) Reset() {
	cmc.mu.Lock()
	defer cmc.mu.Unlock()

	cmc.stats.Hits.Store(0)
	cmc.stats.Misses.Store(0)
	cmc.stats.Sets.Store(0)
	cmc.stats.Deletes.Store(0)
	cmc.stats.Errors.Store(0)
	cmc.stats.L1Hits.Store(0)
	cmc.stats.L1Misses.Store(0)
	cmc.stats.L2Hits.Store(0)
	cmc.stats.L2Misses.Store(0)
	cmc.stats.TotalLatency.Store(0)
	cmc.stats.RequestCount.Store(0)
	cmc.stats.Compressed.Store(0)
	cmc.stats.Decompressed.Store(0)

	cmc.started = time.Now()
	cmc.keyAccessCount = &sync.Map{}
	cmc.hotKeys = &sync.Map{}
	cmc.queryLatency = NewLatencyHistogram()
}

func (cmc *CacheMetricsCollector) ToJSON() ([]byte, error) {
	return json.Marshal(cmc.GetDetailedMetrics())
}

type MetricsExporter struct {
	collector *CacheMetricsCollector
	interval  time.Duration
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

func NewMetricsExporter(collector *CacheMetricsCollector, interval time.Duration) *MetricsExporter {
	if interval <= 0 {
		interval = 10 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &MetricsExporter{
		collector: collector,
		interval:  interval,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (me *MetricsExporter) Start() {
	me.wg.Add(1)
	go me.run()
}

func (me *MetricsExporter) Stop() {
	me.cancel()
	me.wg.Wait()
}

func (me *MetricsExporter) run() {
	defer me.wg.Done()

	ticker := time.NewTicker(me.interval)
	defer ticker.Stop()

	for {
		select {
		case <-me.ctx.Done():
			return
		case <-ticker.C:
			me.export()
		}
	}
}

func (me *MetricsExporter) export() {
}

type CacheAlert struct {
	Type      string
	Message   string
	Timestamp time.Time
	Severity  string
}

type AlertManager struct {
	alerts    []CacheAlert
	mu        sync.RWMutex
	maxAlerts int
}

func NewAlertManager(maxAlerts int) *AlertManager {
	if maxAlerts <= 0 {
		maxAlerts = 100
	}
	return &AlertManager{
		maxAlerts: maxAlerts,
	}
}

func (am *AlertManager) AddAlert(alertType, message, severity string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert := CacheAlert{
		Type:      alertType,
		Message:   message,
		Timestamp: time.Now(),
		Severity:  severity,
	}

	am.alerts = append(am.alerts, alert)

	if len(am.alerts) > am.maxAlerts {
		am.alerts = am.alerts[len(am.alerts)-am.maxAlerts:]
	}
}

func (am *AlertManager) GetAlerts() []CacheAlert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]CacheAlert, len(am.alerts))
	copy(alerts, am.alerts)
	return alerts
}

func (am *AlertManager) CheckHitRate(metrics *DetailedMetrics, threshold float64) {
	if metrics.HitRate < threshold {
		am.AddAlert(
			"low_hit_rate",
			"Cache hit rate is below threshold",
			"warning",
		)
	}
}

func (am *AlertManager) CheckErrorRate(metrics *DetailedMetrics, threshold float64) {
	total := metrics.TotalRequests
	if total == 0 {
		return
	}

	errorRate := float64(metrics.Errors) / float64(total) * 100
	if errorRate > threshold {
		am.AddAlert(
			"high_error_rate",
			"Cache error rate is above threshold",
			"error",
		)
	}
}

var (
	globalMetricsCollector *CacheMetricsCollector
	globalMetricsCollectorOnce sync.Once
	globalAlertManager *AlertManager
	globalAlertManagerOnce sync.Once
)

func GetGlobalMetricsCollector() *CacheMetricsCollector {
	globalMetricsCollectorOnce.Do(func() {
		globalMetricsCollector = NewCacheMetricsCollector()
	})
	return globalMetricsCollector
}

func GetGlobalAlertManager() *AlertManager {
	globalAlertManagerOnce.Do(func() {
		globalAlertManager = NewAlertManager(100)
	})
	return globalAlertManager
}
