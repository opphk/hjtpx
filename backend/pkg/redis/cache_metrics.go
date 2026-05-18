package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type CacheMetricsCollector struct {
	stats          *CacheStats
	hotKeys        *sync.Map
	keyAccessCount *sync.Map
	queryLatency   *LatencyHistogram
	memoryUsage    *MemoryStats
	mu             sync.RWMutex
	started        time.Time
	hitsHistory    *RingBuffer
	missesHistory  *RingBuffer
	evictionHistory *RingBuffer
}

type RingBuffer struct {
	buffer []int64
	size   int
	index  int
	mu     sync.Mutex
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		buffer: make([]int64, size),
		size:   size,
		index:  0,
	}
}

func (rb *RingBuffer) Add(value int64) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.buffer[rb.index] = value
	rb.index = (rb.index + 1) % rb.size
}

func (rb *RingBuffer) Average() float64 {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	var sum int64
	var count int
	for i := 0; i < rb.size; i++ {
		if i < rb.index || rb.index == 0 {
			sum += rb.buffer[i]
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return float64(sum) / float64(count)
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
	PeakUsage    int64
}

type DetailedMetrics struct {
	Uptime              time.Duration    `json:"uptime"`
	HitRate             float64          `json:"hit_rate"`
	TotalRequests       int64            `json:"total_requests"`
	Hits                int64            `json:"hits"`
	Misses              int64            `json:"misses"`
	Sets                int64            `json:"sets"`
	Deletes             int64            `json:"deletes"`
	Errors              int64            `json:"errors"`
	AverageLatency      time.Duration    `json:"average_latency"`
	P95Latency          time.Duration    `json:"p95_latency"`
	P99Latency          time.Duration    `json:"p99_latency"`
	L1HitRate           float64          `json:"l1_hit_rate"`
	L2HitRate           float64          `json:"l2_hit_rate"`
	CompressedCount     int64            `json:"compressed_count"`
	DecompressedCount   int64            `json:"decompressed_count"`
	HotKeys             []HotKeyMetric   `json:"hot_keys"`
	MemoryUsage         MemoryStats      `json:"memory_usage"`
	LatencyDistribution map[string]int64 `json:"latency_distribution"`
	ExpirationRate      float64          `json:"expiration_rate"`
	EvictionRate        float64          `json:"eviction_rate"`
	HitRateTrend        float64          `json:"hit_rate_trend"`
	AverageHitRate      float64          `json:"average_hit_rate"`
}

type HotKeyMetric struct {
	Key         string    `json:"key"`
	AccessCount int64     `json:"access_count"`
	LastAccess  time.Time `json:"last_access"`
	TTL         time.Duration `json:"ttl"`
	HitRate     float64   `json:"hit_rate"`
}

type LRUMetrics struct {
	TotalEvictions int64
	EvictionReasons map[string]int64
	LastEvictedKey string
	PeakEvictions  int64
}

func NewCacheMetricsCollector() *CacheMetricsCollector {
	cmc := &CacheMetricsCollector{
		stats:           &CacheStats{},
		hotKeys:         &sync.Map{},
		keyAccessCount:  &sync.Map{},
		queryLatency:    NewLatencyHistogram(),
		memoryUsage:     &MemoryStats{},
		started:         time.Now(),
		hitsHistory:     NewRingBuffer(100),
		missesHistory:   NewRingBuffer(100),
		evictionHistory: NewRingBuffer(100),
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
	if cmc.hitsHistory != nil {
		cmc.hitsHistory.Add(1)
	}
}

func (cmc *CacheMetricsCollector) RecordMiss() {
	cmc.stats.Misses.Add(1)
	cmc.stats.RequestCount.Add(1)
	if cmc.missesHistory != nil {
		cmc.missesHistory.Add(1)
	}
}

func (cmc *CacheMetricsCollector) RecordEviction() {
	if cmc.evictionHistory != nil {
		cmc.evictionHistory.Add(1)
	}
}

func (cmc *CacheMetricsCollector) RecordExpiration() {
	cmc.stats.Expired.Add(1)
}

func (cmc *CacheMetricsCollector) GetExpirationRate() float64 {
	totalSets := cmc.stats.Sets.Load()
	expired := cmc.stats.Expired.Load()
	if totalSets == 0 {
		return 0
	}
	return float64(expired) / float64(totalSets) * 100
}

func (cmc *CacheMetricsCollector) GetEvictionRate() float64 {
	if cmc.evictionHistory == nil {
		return 0
	}
	evictions := int64(cmc.evictionHistory.Average())
	sets := cmc.stats.Sets.Load()
	if sets == 0 {
		return 0
	}
	return (float64(evictions) / float64(sets)) * 100
}

func (cmc *CacheMetricsCollector) GetHitRateTrend() float64 {
	if cmc.hitsHistory == nil || cmc.missesHistory == nil {
		return 0
	}
	avgHits := cmc.hitsHistory.Average()
	avgMisses := cmc.missesHistory.Average()
	total := avgHits + avgMisses
	if total == 0 {
		return 0
	}
	return (avgHits / total) * 100
}

func (cmc *CacheMetricsCollector) GetAverageHitRate() float64 {
	if cmc.hitsHistory == nil || cmc.missesHistory == nil {
		return 0
	}
	avgHits := cmc.hitsHistory.Average()
	avgMisses := cmc.missesHistory.Average()
	total := avgHits + avgMisses
	if total == 0 {
		return 0
	}
	return (avgHits / total) * 100
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
		Uptime:               time.Since(cmc.started),
		HitRate:              hitRate,
		TotalRequests:        totalRequests,
		Hits:                 hits,
		Misses:               misses,
		Sets:                 cmc.stats.Sets.Load(),
		Deletes:              cmc.stats.Deletes.Load(),
		Errors:               cmc.stats.Errors.Load(),
		AverageLatency:       avgLatency,
		P95Latency:           cmc.queryLatency.Percentile(0.95),
		P99Latency:           cmc.queryLatency.Percentile(0.99),
		L1HitRate:            l1HitRate,
		L2HitRate:            l2HitRate,
		CompressedCount:      cmc.stats.Compressed.Load(),
		DecompressedCount:    cmc.stats.Decompressed.Load(),
		HotKeys:              cmc.getHotKeys(),
		MemoryUsage:          *cmc.memoryUsage,
		LatencyDistribution:  cmc.queryLatency.GetDistribution(),
		ExpirationRate:       cmc.GetExpirationRate(),
		EvictionRate:        cmc.GetEvictionRate(),
		HitRateTrend:        cmc.GetHitRateTrend(),
		AverageHitRate:      cmc.GetAverageHitRate(),
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

func (me *MetricsExporter) ExportToPrometheus() string {
	metrics := me.collector.GetDetailedMetrics()

	return fmt.Sprintf(`# HELP cache_hits_total Total number of cache hits
# TYPE cache_hits_total counter
cache_hits_total %d

# HELP cache_misses_total Total number of cache misses
# TYPE cache_misses_total counter
cache_misses_total %d

# HELP cache_hit_rate Current cache hit rate
# TYPE cache_hit_rate gauge
cache_hit_rate %.2f

# HELP cache_requests_total Total number of cache requests
# TYPE cache_requests_total counter
cache_requests_total %d

# HELP cache_latency_seconds Cache operation latency
# TYPE cache_latency_seconds histogram
cache_latency_seconds_bucket{quantile="0.95"} %.6f
cache_latency_seconds_bucket{quantile="0.99"} %.6f

# HELP cache_errors_total Total number of cache errors
# TYPE cache_errors_total counter
cache_errors_total %d

# HELP cache_sets_total Total number of cache sets
# TYPE cache_sets_total counter
cache_sets_total %d

# HELP cache_deletes_total Total number of cache deletes
# TYPE cache_deletes_total counter
cache_deletes_total %d

# HELP cache_l1_hit_rate L1 cache hit rate
# TYPE cache_l1_hit_rate gauge
cache_l1_hit_rate %.2f

# HELP cache_l2_hit_rate L2 cache hit rate
# TYPE cache_l2_hit_rate gauge
cache_l2_hit_rate %.2f

# HELP cache_compressed_total Total number of compressions
# TYPE cache_compressed_total counter
cache_compressed_total %d

# HELP cache_decompressed_total Total number of decompressions
# TYPE cache_decompressed_total counter
cache_decompressed_total %d`,
		metrics.Hits,
		metrics.Misses,
		metrics.HitRate/100,
		metrics.TotalRequests,
		metrics.P95Latency.Seconds(),
		metrics.P99Latency.Seconds(),
		metrics.Errors,
		metrics.Sets,
		metrics.Deletes,
		metrics.L1HitRate/100,
		metrics.L2HitRate/100,
		metrics.CompressedCount,
		metrics.DecompressedCount,
	)
}

func (me *MetricsExporter) ExportToJSON() (string, error) {
	metrics := me.collector.GetDetailedMetrics()
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (me *MetricsExporter) GetAlertSummary() map[string]interface{} {
	metrics := me.collector.GetDetailedMetrics()
	alerts := GetCacheMonitoringCollector().GetAlerts()

	summary := make(map[string]interface{})
	summary["total_alerts"] = len(alerts)
	summary["recent_alerts"] = alerts

	alertCounts := make(map[string]int)
	for _, alert := range alerts {
		alertCounts[alert.Type]++
	}
	summary["alert_counts"] = alertCounts

	summary["hit_rate_status"] = "healthy"
	if metrics.HitRate < 80 {
		summary["hit_rate_status"] = "warning"
	}
	if metrics.HitRate < 50 {
		summary["hit_rate_status"] = "critical"
	}

	summary["error_rate"] = 0.0
	if metrics.TotalRequests > 0 {
		summary["error_rate"] = float64(metrics.Errors) / float64(metrics.TotalRequests) * 100
	}

	return summary
}

type CacheMetricsSnapshot struct {
	Timestamp time.Time
	Metrics   *DetailedMetrics
	Alerts    []CacheAlert
}

func (cmc *CacheMetricsCollector) TakeSnapshot() *CacheMetricsSnapshot {
	return &CacheMetricsSnapshot{
		Timestamp: time.Now(),
		Metrics:   cmc.GetDetailedMetrics(),
		Alerts:   GetCacheMonitoringCollector().GetAlerts(),
	}
}

type MetricsAggregator struct {
	snapshots    []*CacheMetricsSnapshot
	maxSnapshots int
	mu           sync.Mutex
}

func NewMetricsAggregator(maxSnapshots int) *MetricsAggregator {
	if maxSnapshots <= 0 {
		maxSnapshots = 100
	}
	return &MetricsAggregator{
		snapshots:    make([]*CacheMetricsSnapshot, 0, maxSnapshots),
		maxSnapshots: maxSnapshots,
	}
}

func (ma *MetricsAggregator) AddSnapshot(snapshot *CacheMetricsSnapshot) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	ma.snapshots = append(ma.snapshots, snapshot)
	if len(ma.snapshots) > ma.maxSnapshots {
		ma.snapshots = ma.snapshots[1:]
	}
}

func (ma *MetricsAggregator) GetAverageHitRate() float64 {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	if len(ma.snapshots) == 0 {
		return 0
	}

	var sum float64
	for _, snapshot := range ma.snapshots {
		sum += snapshot.Metrics.HitRate
	}
	return sum / float64(len(ma.snapshots))
}

func (ma *MetricsAggregator) GetAverageLatency() time.Duration {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	if len(ma.snapshots) == 0 {
		return 0
	}

	var sum int64
	for _, snapshot := range ma.snapshots {
		sum += snapshot.Metrics.AverageLatency.Nanoseconds()
	}
	return time.Duration(sum / int64(len(ma.snapshots)))
}

var (
	globalMetricsCollector     *CacheMetricsCollector
	globalMetricsCollectorOnce sync.Once
)

func GetGlobalMetricsCollector() *CacheMetricsCollector {
	globalMetricsCollectorOnce.Do(func() {
		globalMetricsCollector = NewCacheMetricsCollector()
	})
	return globalMetricsCollector
}
