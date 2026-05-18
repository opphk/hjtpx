package redis

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

type CacheMonitoringConfig struct {
	EnableDetailedMetrics bool
	EnableHotKeyTracking  bool
	EnableLatencyTracking bool
	EnableMemoryTracking  bool
	MonitoringInterval    time.Duration
	HotKeyThreshold      int64
	LatencyBuckets       []time.Duration
}

var DefaultMonitoringConfig = &CacheMonitoringConfig{
	EnableDetailedMetrics: true,
	EnableHotKeyTracking:  true,
	EnableLatencyTracking: true,
	EnableMemoryTracking: true,
	MonitoringInterval:    10 * time.Second,
	HotKeyThreshold:      100,
	LatencyBuckets: []time.Duration{
		100 * time.Microsecond,
		500 * time.Microsecond,
		1 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	},
}

type CacheMonitoringCollector struct {
	config              *CacheMonitoringConfig
	ctx                 context.Context
	cancel              context.CancelFunc
	wg                  sync.WaitGroup
	mu                  sync.RWMutex
	started             bool
	
	hitCounter          atomic.Int64
	missCounter         atomic.Int64
	setCounter          atomic.Int64
	deleteCounter       atomic.Int64
	errorCounter        atomic.Int64
	l1HitCounter        atomic.Int64
	l1MissCounter       atomic.Int64
	l2HitCounter        atomic.Int64
	l2MissCounter       atomic.Int64
	totalLatency        atomic.Int64
	compressedCounter   atomic.Int64
	decompressedCounter atomic.Int64
	
	hotKeys             sync.Map
	latencyDistribution map[string]*atomic.Int64
	keyAccessHistory    map[string]*AccessRecord
	memorySnapshots     []MemorySnapshot
	
	lastHitCount        int64
	lastMissCount       int64
	lastHitRate         float64
	peakHitRate         float64
	lowHitRateCount     int64
}

type AccessRecord struct {
	Key         string
	AccessCount int64
	LastAccess  time.Time
	FirstAccess time.Time
}

type MemorySnapshot struct {
	Timestamp time.Time
	L1Size    int64
	L2Keys    int64
	L2Memory  int64
}

type CacheHealthStatus struct {
	Status           string    `json:"status"`
	HitRate          float64   `json:"hit_rate"`
	ErrorRate        float64   `json:"error_rate"`
	L1HitRate        float64   `json:"l1_hit_rate"`
	L2HitRate        float64   `json:"l2_hit_rate"`
	AvgLatency       float64   `json:"avg_latency_ms"`
	P95Latency       float64   `json:"p95_latency_ms"`
	P99Latency       float64   `json:"p99_latency_ms"`
	TotalRequests    int64     `json:"total_requests"`
	TotalErrors      int64     `json:"total_errors"`
	HotKeyCount      int       `json:"hot_key_count"`
	LowHitRateAlerts int64     `json:"low_hit_rate_alerts"`
	PeakHitRate      float64   `json:"peak_hit_rate"`
	LastChecked      time.Time `json:"last_checked"`
}

func NewCacheMonitoringCollector(config *CacheMonitoringConfig) *CacheMonitoringCollector {
	if config == nil {
		config = DefaultMonitoringConfig
	}

	ctx, cancel := context.WithCancel(context.Background())

	collector := &CacheMonitoringCollector{
		config:              config,
		ctx:                 ctx,
		cancel:              cancel,
		latencyDistribution: make(map[string]*atomic.Int64),
		keyAccessHistory:    make(map[string]*AccessRecord),
		memorySnapshots:     make([]MemorySnapshot, 0, 100),
	}

	for _, bucket := range config.LatencyBuckets {
		collector.latencyDistribution[bucket.String()] = &atomic.Int64{}
	}

	return collector
}

func (cmc *CacheMonitoringCollector) Start() {
	cmc.mu.Lock()
	if cmc.started {
		cmc.mu.Unlock()
		return
	}
	cmc.started = true
	cmc.mu.Unlock()

	cmc.wg.Add(1)
	go cmc.monitorLoop()

	if cmc.config.EnableDetailedMetrics {
		cmc.wg.Add(1)
		go cmc.collectDetailedMetrics()
	}
}

func (cmc *CacheMonitoringCollector) Stop() {
	cmc.mu.Lock()
	if !cmc.started {
		cmc.mu.Unlock()
		return
	}
	cmc.started = false
	cmc.mu.Unlock()

	cmc.cancel()
	cmc.wg.Wait()
}

func (cmc *CacheMonitoringCollector) monitorLoop() {
	defer cmc.wg.Done()

	ticker := time.NewTicker(cmc.config.MonitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cmc.ctx.Done():
			return
		case <-ticker.C:
			cmc.updateHealthMetrics()
		}
	}
}

func (cmc *CacheMonitoringCollector) collectDetailedMetrics() {
	defer cmc.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-cmc.ctx.Done():
			return
		case <-ticker.C:
			cmc.cleanupOldData()
			cmc.snapshotMemory()
		}
	}
}

func (cmc *CacheMonitoringCollector) RecordHit() {
	cmc.hitCounter.Add(1)
}

func (cmc *CacheMonitoringCollector) RecordMiss() {
	cmc.missCounter.Add(1)
}

func (cmc *CacheMonitoringCollector) RecordSet() {
	cmc.setCounter.Add(1)
}

func (cmc *CacheMonitoringCollector) RecordDelete() {
	cmc.deleteCounter.Add(1)
}

func (cmc *CacheMonitoringCollector) RecordError() {
	cmc.errorCounter.Add(1)
}

func (cmc *CacheMonitoringCollector) RecordL1Hit() {
	cmc.l1HitCounter.Add(1)
}

func (cmc *CacheMonitoringCollector) RecordL1Miss() {
	cmc.l1MissCounter.Add(1)
}

func (cmc *CacheMonitoringCollector) RecordL2Hit() {
	cmc.l2HitCounter.Add(1)
}

func (cmc *CacheMonitoringCollector) RecordL2Miss() {
	cmc.l2MissCounter.Add(1)
}

func (cmc *CacheMonitoringCollector) RecordLatency(duration time.Duration) {
	cmc.totalLatency.Add(duration.Nanoseconds())
	
	if cmc.config.EnableLatencyTracking {
		for _, bucket := range cmc.config.LatencyBuckets {
			if duration <= bucket {
				cmc.latencyDistribution[bucket.String()].Add(1)
				return
			}
		}
		cmc.latencyDistribution[cmc.config.LatencyBuckets[len(cmc.config.LatencyBuckets)-1].String()].Add(1)
	}
}

func (cmc *CacheMonitoringCollector) RecordKeyAccess(key string) {
	if !cmc.config.EnableHotKeyTracking {
		return
	}

	now := time.Now()
	cmc.mu.Lock()
	defer cmc.mu.Unlock()

	if record, exists := cmc.keyAccessHistory[key]; exists {
		record.AccessCount++
		record.LastAccess = now
	} else {
		cmc.keyAccessHistory[key] = &AccessRecord{
			Key:         key,
			AccessCount: 1,
			LastAccess:  now,
			FirstAccess: now,
		}
	}

	accessCount := cmc.keyAccessHistory[key].AccessCount
	if accessCount >= cmc.config.HotKeyThreshold {
		cmc.hotKeys.Store(key, accessCount)
	}
}

func (cmc *CacheMonitoringCollector) RecordCompressed() {
	cmc.compressedCounter.Add(1)
}

func (cmc *CacheMonitoringCollector) RecordDecompressed() {
	cmc.decompressedCounter.Add(1)
}

func (cmc *CacheMonitoringCollector) updateHealthMetrics() {
	hits := cmc.hitCounter.Load()
	misses := cmc.missCounter.Load()
	total := hits + misses

	currentHitRate := 0.0
	if total > 0 {
		currentHitRate = float64(hits) / float64(total) * 100
	}

	cmc.mu.Lock()
	
	if total > 0 {
		cmc.lastHitRate = currentHitRate
	}
	
	if currentHitRate > cmc.peakHitRate {
		cmc.peakHitRate = currentHitRate
	}

	if currentHitRate < 50.0 && total > 100 {
		cmc.lowHitRateCount++
	}

	cmc.mu.Unlock()

	cmc.lastHitCount = hits
	cmc.lastMissCount = misses
}

func (cmc *CacheMonitoringCollector) snapshotMemory() {
	cmc.mu.Lock()
	defer cmc.mu.Unlock()

	snapshot := MemorySnapshot{
		Timestamp: time.Now(),
		L1Size:    int64(0),
		L2Keys:    int64(0),
		L2Memory:  int64(0),
	}

	if enhancedCache := GetEnhancedCache(); enhancedCache != nil {
		stats := enhancedCache.GetStats()
		snapshot.L1Size = stats.L1Hits + stats.L1Misses
		snapshot.L2Keys = stats.Hits + stats.Misses
	}

	cmc.memorySnapshots = append(cmc.memorySnapshots, snapshot)

	if len(cmc.memorySnapshots) > 100 {
		cmc.memorySnapshots = cmc.memorySnapshots[len(cmc.memorySnapshots)-100:]
	}
}

func (cmc *CacheMonitoringCollector) cleanupOldData() {
	cmc.mu.Lock()
	defer cmc.mu.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour)
	for key, record := range cmc.keyAccessHistory {
		if record.LastAccess.Before(cutoff) {
			delete(cmc.keyAccessHistory, key)
			cmc.hotKeys.Delete(key)
		}
	}
}

func (cmc *CacheMonitoringCollector) GetHealthStatus() *CacheHealthStatus {
	hits := cmc.hitCounter.Load()
	misses := cmc.missCounter.Load()
	total := hits + misses
	errors := cmc.errorCounter.Load()

	l1Hits := cmc.l1HitCounter.Load()
	l1Misses := cmc.l1MissCounter.Load()
	l2Hits := cmc.l2HitCounter.Load()
	l2Misses := cmc.l2MissCounter.Load()

	totalLatency := cmc.totalLatency.Load()
	avgLatency := 0.0
	if total > 0 {
		avgLatency = float64(totalLatency) / float64(total) / 1e6
	}

	hitRate := 0.0
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	errorRate := 0.0
	if total > 0 {
		errorRate = float64(errors) / float64(total) * 100
	}

	l1HitRate := 0.0
	l1Total := l1Hits + l1Misses
	if l1Total > 0 {
		l1HitRate = float64(l1Hits) / float64(l1Total) * 100
	}

	l2HitRate := 0.0
	l2Total := l2Hits + l2Misses
	if l2Total > 0 {
		l2HitRate = float64(l2Hits) / float64(l2Total) * 100
	}

	cmc.mu.RLock()
	lowHitRateCount := cmc.lowHitRateCount
	peakHitRate := cmc.peakHitRate
	cmc.mu.RUnlock()

	status := "healthy"
	if errorRate > 5.0 {
		status = "degraded"
	}
	if errorRate > 10.0 {
		status = "unhealthy"
	}

	hotKeyCount := 0
	cmc.hotKeys.Range(func(_, _ interface{}) bool {
		hotKeyCount++
		return true
	})

	return &CacheHealthStatus{
		Status:           status,
		HitRate:          hitRate,
		ErrorRate:        errorRate,
		L1HitRate:        l1HitRate,
		L2HitRate:        l2HitRate,
		AvgLatency:       avgLatency,
		P95Latency:       cmc.calculatePercentileLatency(0.95),
		P99Latency:       cmc.calculatePercentileLatency(0.99),
		TotalRequests:    total,
		TotalErrors:     errors,
		HotKeyCount:     hotKeyCount,
		LowHitRateAlerts: lowHitRateCount,
		PeakHitRate:     peakHitRate,
		LastChecked:     time.Now(),
	}
}

func (cmc *CacheMonitoringCollector) calculatePercentileLatency(percentile float64) float64 {
	cmc.mu.RLock()
	defer cmc.mu.RUnlock()

	total := int64(0)
	for _, bucket := range cmc.config.LatencyBuckets {
		total += cmc.latencyDistribution[bucket.String()].Load()
	}

	if total == 0 {
		return 0.0
	}

	targetCount := int64(float64(total) * percentile)
	currentCount := int64(0)

	for i, bucket := range cmc.config.LatencyBuckets {
		currentCount += cmc.latencyDistribution[bucket.String()].Load()
		if currentCount >= targetCount {
			return float64(bucket) / 1e6
		}
		if i == len(cmc.config.LatencyBuckets)-1 {
			return float64(bucket) / 1e6
		}
	}

	return 0.0
}

func (cmc *CacheMonitoringCollector) GetDetailedMetrics() map[string]interface{} {
	hits := cmc.hitCounter.Load()
	misses := cmc.missCounter.Load()
	total := hits + misses

	metrics := map[string]interface{}{
		"hits":                  hits,
		"misses":                misses,
		"sets":                  cmc.setCounter.Load(),
		"deletes":               cmc.deleteCounter.Load(),
		"errors":                cmc.errorCounter.Load(),
		"total_requests":        total,
		"hit_rate":               0.0,
		"l1_hits":               cmc.l1HitCounter.Load(),
		"l1_misses":             cmc.l1MissCounter.Load(),
		"l2_hits":               cmc.l2HitCounter.Load(),
		"l2_misses":             cmc.l2MissCounter.Load(),
		"l1_hit_rate":           0.0,
		"l2_hit_rate":           0.0,
		"compressed":            cmc.compressedCounter.Load(),
		"decompressed":          cmc.decompressedCounter.Load(),
		"avg_latency_ms":         0.0,
		"p95_latency_ms":         cmc.calculatePercentileLatency(0.95),
		"p99_latency_ms":         cmc.calculatePercentileLatency(0.99),
		"hot_key_count":          0,
		"memory_snapshots_count": len(cmc.memorySnapshots),
	}

	if total > 0 {
		metrics["hit_rate"] = float64(hits) / float64(total) * 100
	}

	l1Total := cmc.l1HitCounter.Load() + cmc.l1MissCounter.Load()
	if l1Total > 0 {
		metrics["l1_hit_rate"] = float64(cmc.l1HitCounter.Load()) / float64(l1Total) * 100
	}

	l2Total := cmc.l2HitCounter.Load() + cmc.l2MissCounter.Load()
	if l2Total > 0 {
		metrics["l2_hit_rate"] = float64(cmc.l2HitCounter.Load()) / float64(l2Total) * 100
	}

	if total > 0 {
		metrics["avg_latency_ms"] = float64(cmc.totalLatency.Load()) / float64(total) / 1e6
	}

	cmc.mu.RLock()
	hotKeyCount := 0
	cmc.hotKeys.Range(func(_, _ interface{}) bool {
		hotKeyCount++
		return true
	})
	metrics["hot_key_count"] = hotKeyCount
	cmc.mu.RUnlock()

	return metrics
}

func (cmc *CacheMonitoringCollector) GetHotKeys(limit int) []AccessRecord {
	cmc.mu.RLock()
	defer cmc.mu.RUnlock()

	records := make([]AccessRecord, 0, len(cmc.keyAccessHistory))
	for _, record := range cmc.keyAccessHistory {
		if record.AccessCount >= cmc.config.HotKeyThreshold {
			records = append(records, *record)
		}
	}

	if len(records) > limit {
		return records[:limit]
	}

	return records
}

func (cmc *CacheMonitoringCollector) GetLatencyDistribution() map[string]int64 {
	cmc.mu.RLock()
	defer cmc.mu.RUnlock()

	distribution := make(map[string]int64)
	for bucket, counter := range cmc.latencyDistribution {
		distribution[bucket] = counter.Load()
	}

	return distribution
}

func (cmc *CacheMonitoringCollector) GetMemoryTrend() []MemorySnapshot {
	cmc.mu.RLock()
	defer cmc.mu.RUnlock()

	snapshots := make([]MemorySnapshot, len(cmc.memorySnapshots))
	copy(snapshots, cmc.memorySnapshots)

	return snapshots
}

func (cmc *CacheMonitoringCollector) ExportMetricsJSON() ([]byte, error) {
	metrics := cmc.GetDetailedMetrics()
	health := cmc.GetHealthStatus()

	exportData := map[string]interface{}{
		"metrics":      metrics,
		"health":       health,
		"latency_dist": cmc.GetLatencyDistribution(),
		"hot_keys":     cmc.GetHotKeys(10),
		"memory_trend": cmc.GetMemoryTrend(),
		"timestamp":    time.Now(),
	}

	return json.MarshalIndent(exportData, "", "  ")
}

func (cmc *CacheMonitoringCollector) Reset() {
	cmc.hitCounter.Store(0)
	cmc.missCounter.Store(0)
	cmc.setCounter.Store(0)
	cmc.deleteCounter.Store(0)
	cmc.errorCounter.Store(0)
	cmc.l1HitCounter.Store(0)
	cmc.l1MissCounter.Store(0)
	cmc.l2HitCounter.Store(0)
	cmc.l2MissCounter.Store(0)
	cmc.totalLatency.Store(0)
	cmc.compressedCounter.Store(0)
	cmc.decompressedCounter.Store(0)

	cmc.mu.Lock()
	cmc.hotKeys = sync.Map{}
	cmc.keyAccessHistory = make(map[string]*AccessRecord)
	cmc.memorySnapshots = make([]MemorySnapshot, 0, 100)
	cmc.lastHitRate = 0
	cmc.peakHitRate = 0
	cmc.lowHitRateCount = 0
	cmc.mu.Unlock()

	for _, bucket := range cmc.config.LatencyBuckets {
		cmc.latencyDistribution[bucket.String()].Store(0)
	}
}

var (
	globalMonitoringCollector *CacheMonitoringCollector
	globalMonitoringOnce      sync.Once
)

func InitCacheMonitoring(config *CacheMonitoringConfig) {
	globalMonitoringOnce.Do(func() {
		globalMonitoringCollector = NewCacheMonitoringCollector(config)
	})
}

func GetCacheMonitoringCollector() *CacheMonitoringCollector {
	if globalMonitoringCollector == nil {
		InitCacheMonitoring(nil)
	}
	return globalMonitoringCollector
}

func StartCacheMonitoring() {
	GetCacheMonitoringCollector().Start()
}

func StopCacheMonitoring() {
	GetCacheMonitoringCollector().Stop()
}

func RecordCacheHit() {
	GetCacheMonitoringCollector().RecordHit()
}

func RecordCacheMiss() {
	GetCacheMonitoringCollector().RecordMiss()
}

func RecordCacheSet() {
	GetCacheMonitoringCollector().RecordSet()
}

func RecordCacheDelete() {
	GetCacheMonitoringCollector().RecordDelete()
}

func RecordCacheError() {
	GetCacheMonitoringCollector().RecordError()
}

func RecordCacheLatency(duration time.Duration) {
	GetCacheMonitoringCollector().RecordLatency(duration)
}

func RecordCacheKeyAccess(key string) {
	GetCacheMonitoringCollector().RecordKeyAccess(key)
}
