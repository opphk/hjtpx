package service

import (
	"fmt"
	"sync"
	"time"
)

type AnalysisCache struct {
	mu          sync.RWMutex
	cache       map[string]*CacheItem
	maxSize     int
	ttl         time.Duration
	hitCount    int64
	missCount   int64
	evictCount  int64
	enableStats bool
}

type CacheItem struct {
	Key         string
	Value       interface{}
	CreatedAt   time.Time
	AccessedAt  time.Time
	AccessCount int64
	Cost        int64
}

func NewAnalysisCache(maxSize int, ttl time.Duration) *AnalysisCache {
	cache := &AnalysisCache{
		cache:       make(map[string]*CacheItem),
		maxSize:     maxSize,
		ttl:         ttl,
		enableStats: true,
	}

	go cache.startCleanupRoutine()

	return cache
}

func (c *AnalysisCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, exists := c.cache[key]
	c.mu.RUnlock()

	if !exists {
		if c.enableStats {
			c.mu.Lock()
			c.missCount++
			c.mu.Unlock()
		}
		return nil, false
	}

	if time.Since(item.CreatedAt) > c.ttl {
		c.Delete(key)
		if c.enableStats {
			c.mu.Lock()
			c.missCount++
			c.mu.Unlock()
		}
		return nil, false
	}

	if c.enableStats {
		c.mu.Lock()
		c.hitCount++
		item.AccessedAt = time.Now()
		item.AccessCount++
		c.mu.Unlock()
	}

	return item.Value, true
}

func (c *AnalysisCache) Set(key string, value interface{}) {
	c.SetWithCost(key, value, 1)
}

func (c *AnalysisCache) SetWithCost(key string, value interface{}, cost int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.cache[key]; exists {
		item.Value = value
		item.CreatedAt = time.Now()
		item.AccessedAt = time.Now()
		item.Cost = cost
		return
	}

	if len(c.cache) >= c.maxSize {
		c.evict()
	}

	c.cache[key] = &CacheItem{
		Key:         key,
		Value:       value,
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 1,
		Cost:        cost,
	}
}

func (c *AnalysisCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.cache[key]; exists {
		delete(c.cache, key)
		c.evictCount++
	}
}

func (c *AnalysisCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CacheItem)
}

func (c *AnalysisCache) evict() {
	if len(c.cache) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time

	for key, item := range c.cache {
		if oldestTime.IsZero() || item.AccessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.AccessedAt
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
		c.evictCount++
	}
}

func (c *AnalysisCache) startCleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

func (c *AnalysisCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	keysToDelete := make([]string, 0)

	for key, item := range c.cache {
		if now.Sub(item.CreatedAt) > c.ttl {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(c.cache, key)
		c.evictCount++
	}
}

func (c *AnalysisCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["size"] = len(c.cache)
	stats["max_size"] = c.maxSize
	stats["hit_count"] = c.hitCount
	stats["miss_count"] = c.missCount
	stats["evict_count"] = c.evictCount

	if c.hitCount+c.missCount > 0 {
		stats["hit_rate"] = float64(c.hitCount) / float64(c.hitCount+c.missCount)
	} else {
		stats["hit_rate"] = 0.0
	}

	stats["ttl_seconds"] = c.ttl.Seconds()

	return stats
}

func (c *AnalysisCache) EnableStats(enable bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enableStats = enable
}

type PerformanceMonitor struct {
	mu              sync.RWMutex
	measurements    map[string][]time.Duration
	maxMeasurements int
}

func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		measurements:    make(map[string][]time.Duration),
		maxMeasurements: 1000,
	}
}

func (pm *PerformanceMonitor) Record(name string, duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.measurements[name]; !exists {
		pm.measurements[name] = make([]time.Duration, 0)
	}

	pm.measurements[name] = append(pm.measurements[name], duration)

	if len(pm.measurements[name]) > pm.maxMeasurements {
		pm.measurements[name] = pm.measurements[name][1:]
	}
}

func (pm *PerformanceMonitor) GetStats(name string) map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	measurements, exists := pm.measurements[name]
	if !exists || len(measurements) == 0 {
		return nil
	}

	total := time.Duration(0)
	min := measurements[0]
	max := measurements[0]

	for _, m := range measurements {
		total += m
		if m < min {
			min = m
		}
		if m > max {
			max = m
		}
	}

	avg := total / time.Duration(len(measurements))

	stats := make(map[string]interface{})
	stats["count"] = len(measurements)
	stats["avg_ms"] = float64(avg.Nanoseconds()) / 1e6
	stats["min_ms"] = float64(min.Nanoseconds()) / 1e6
	stats["max_ms"] = float64(max.Nanoseconds()) / 1e6

	return stats
}

func (pm *PerformanceMonitor) GetAllStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	allStats := make(map[string]interface{})

	for name, measurements := range pm.measurements {
		if len(measurements) > 0 {
			total := time.Duration(0)
			min := measurements[0]
			max := measurements[0]

			for _, m := range measurements {
				total += m
				if m < min {
					min = m
				}
				if m > max {
					max = m
				}
			}

			avg := total / time.Duration(len(measurements))

			stats := map[string]interface{}{
				"count":  len(measurements),
				"avg_ms": float64(avg.Nanoseconds()) / 1e6,
				"min_ms": float64(min.Nanoseconds()) / 1e6,
				"max_ms": float64(max.Nanoseconds()) / 1e6,
			}

			allStats[name] = stats
		}
	}

	return allStats
}

func (pm *PerformanceMonitor) Clear() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.measurements = make(map[string][]time.Duration)
}

type PerformanceTimer struct {
	name      string
	startTime time.Time
	monitor   *PerformanceMonitor
}

func (pm *PerformanceMonitor) StartTimer(name string) *PerformanceTimer {
	return &PerformanceTimer{
		name:      name,
		startTime: time.Now(),
		monitor:   pm,
	}
}

func (pt *PerformanceTimer) Stop() {
	duration := time.Since(pt.startTime)
	pt.monitor.Record(pt.name, duration)
}

type OptimizedAnalyzer struct {
	cache              *AnalysisCache
	performanceMonitor *PerformanceMonitor
	enableCache        bool
}

func NewOptimizedAnalyzer() *OptimizedAnalyzer {
	return &OptimizedAnalyzer{
		cache:              NewAnalysisCache(10000, 5*time.Minute),
		performanceMonitor: NewPerformanceMonitor(),
		enableCache:        true,
	}
}

func (oa *OptimizedAnalyzer) AnalyzeSliderWithCache(trajectory []SliderPoint, targetPosition int) (*SliderAnalysisResult, error) {
	timer := oa.performanceMonitor.StartTimer("slider_analysis")

	defer timer.Stop()

	if oa.enableCache {
		cacheKey := oa.generateSliderCacheKey(trajectory, targetPosition)
		if cached, exists := oa.cache.Get(cacheKey); exists {
			return cached.(*SliderAnalysisResult), nil
		}
	}

	analyzer := NewSliderAnalyzer()
	result, err := analyzer.AnalyzeSliderTrajectory(trajectory, targetPosition)
	if err != nil {
		return nil, err
	}

	if oa.enableCache {
		cacheKey := oa.generateSliderCacheKey(trajectory, targetPosition)
		oa.cache.Set(cacheKey, result)
	}

	return result, nil
}

func (oa *OptimizedAnalyzer) generateSliderCacheKey(trajectory []SliderPoint, targetPosition int) string {
	if len(trajectory) == 0 {
		return ""
	}

	first := trajectory[0]
	last := trajectory[len(trajectory)-1]

	return fmt.Sprintf("slider_%d_%d_%d_%d_%d",
		targetPosition,
		first.Timestamp,
		last.Timestamp,
		len(trajectory),
		(last.X - first.X))
}

func (oa *OptimizedAnalyzer) AnalyzeClickWithCache(verification *ClickVerification) *ClickAnalysisResult {
	timer := oa.performanceMonitor.StartTimer("click_analysis")

	defer timer.Stop()

	if oa.enableCache {
		cacheKey := oa.generateClickCacheKey(verification)
		if cached, exists := oa.cache.Get(cacheKey); exists {
			return cached.(*ClickAnalysisResult)
		}
	}

	analyzer := NewClickAnalyzer()
	result := analyzer.AnalyzeClickVerification(verification)

	if oa.enableCache {
		cacheKey := oa.generateClickCacheKey(verification)
		oa.cache.Set(cacheKey, result)
	}

	return result
}

func (oa *OptimizedAnalyzer) generateClickCacheKey(verification *ClickVerification) string {
	if verification == nil || len(verification.Clicks) == 0 {
		return "click_empty"
	}

	first := verification.Clicks[0]
	last := verification.Clicks[len(verification.Clicks)-1]

	return fmt.Sprintf("click_%d_%d_%d_%d",
		first.Timestamp,
		last.Timestamp,
		len(verification.Clicks),
		first.X)
}

func (oa *OptimizedAnalyzer) GetPerformanceStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["cache"] = oa.cache.GetStats()
	stats["performance"] = oa.performanceMonitor.GetAllStats()
	return stats
}

func (oa *OptimizedAnalyzer) EnableCache(enable bool) {
	oa.enableCache = enable
}

func (oa *OptimizedAnalyzer) ClearCache() {
	oa.cache.Clear()
}

func (oa *OptimizedAnalyzer) ClearPerformanceStats() {
	oa.performanceMonitor.Clear()
}
