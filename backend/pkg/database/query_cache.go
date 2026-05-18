package database

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

type QueryCache struct {
	mu        sync.RWMutex
	cache     map[string]cacheEntry
	maxSize   int
	ttl       time.Duration
	enabled   bool
	stats     *QueryCacheStats
	strategy  QueryCacheStrategy
	hits      atomic.Int64
	misses    atomic.Int64
	evictions atomic.Int64
}

type cacheEntry struct {
	value       interface{}
	expiration  time.Time
	accessCount int64
	lastAccess  time.Time
	frequency   float64
}

type QueryCacheStats struct {
	TotalHits     atomic.Int64
	TotalMisses   atomic.Int64
	TotalSets     atomic.Int64
	TotalEvictions atomic.Int64
	AvgLatency    atomic.Int64
	PeakSize      atomic.Int64
}

type QueryCacheStrategy int

const (
	StrategyLRU QueryCacheStrategy = iota
	StrategyLFU
	StrategyAdaptive
	StrategyTTL
)

type IntelligentQueryCache struct {
	baseCache     *QueryCache
	queryPatterns map[string]*QueryPattern
	mu            sync.RWMutex
}

type QueryPattern struct {
	Pattern       string
	Frequency     float64
	AvgLatency    time.Duration
	HitRate       float64
	OptimalTTL    time.Duration
	TotalQueries  int64
	LastQuery     time.Time
	Complexity    string
	Priority      int
}

var queryCache *QueryCache
var intelligentCache *IntelligentQueryCache

func InitQueryCache(cfg *config.Config) {
	queryCache = &QueryCache{
		cache:   make(map[string]cacheEntry),
		maxSize: cfg.Database.QueryOptimization.MaxQueryCacheSize,
		ttl:     time.Duration(cfg.Database.QueryOptimization.QueryCacheTTLSecs) * time.Second,
		enabled: cfg.Database.QueryOptimization.EnableQueryCache,
		stats:  &QueryCacheStats{},
		strategy: StrategyAdaptive,
	}

	intelligentCache = &IntelligentQueryCache{
		baseCache:     queryCache,
		queryPatterns: make(map[string]*QueryPattern),
	}

	go queryCache.startCleanup()

	if queryCache.enabled {
		log.Println("Query cache initialized with intelligent strategy")
	}
}

func GetIntelligentCache() *IntelligentQueryCache {
	if intelligentCache == nil {
		intelligentCache = NewIntelligentQueryCache()
	}
	return intelligentCache
}

func GetQueryCache() *QueryCache {
	return queryCache
}

func CachedQuery(keySuffix string, dest interface{}, queryFunc func() error, ttl ...time.Duration) error {
	if !queryCache.enabled {
		return queryFunc()
	}

	key := queryCache.generateKey(keySuffix)
	if cached, ok := queryCache.Get(key); ok {
		if cachedData, err := json.Marshal(cached); err == nil {
			json.Unmarshal(cachedData, dest)
			return nil
		}
	}

	if err := queryFunc(); err != nil {
		return err
	}

	queryCache.Set(key, dest)
	return nil
}

func InvalidateQueryCache(tableName string) {
	if queryCache == nil {
		return
	}
	queryCache.ClearPattern(fmt.Sprintf("table:%s:", tableName))
}

func (c *QueryCache) generateKey(query string, args ...interface{}) string {
	data := query
	for _, arg := range args {
		argBytes, _ := json.Marshal(arg)
		data += string(argBytes)
	}
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (c *QueryCache) Get(key string) (interface{}, bool) {
	if !c.enabled {
		return nil, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		c.misses.Add(1)
		return nil, false
	}

	if time.Now().After(entry.expiration) {
		delete(c.cache, key)
		c.misses.Add(1)
		return nil, false
	}

	entry.accessCount++
	entry.lastAccess = time.Now()
	c.hits.Add(1)
	return entry.value, true
}

func (c *QueryCache) Set(key string, value interface{}) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cache) >= c.maxSize {
		c.evictByStrategy()
	}

	entry := cacheEntry{
		value:       value,
		expiration:  time.Now().Add(c.ttl),
		accessCount: 1,
		lastAccess:  time.Now(),
		frequency:   1.0,
	}

	c.cache[key] = entry
	c.stats.TotalSets.Add(1)

	if int64(len(c.cache)) > c.stats.PeakSize.Load() {
		c.stats.PeakSize.Store(int64(len(c.cache)))
	}
}

func (c *QueryCache) evictByStrategy() {
	switch c.strategy {
	case StrategyLRU:
		c.evictLRU()
	case StrategyLFU:
		c.evictLFU()
	case StrategyAdaptive:
		c.evictAdaptive()
	default:
		c.evictLRU()
	}
}

func (c *QueryCache) evictLRU() {
	oldestKey := ""
	oldestTime := time.Now().Add(24 * time.Hour)

	for k, v := range c.cache {
		if v.lastAccess.Before(oldestTime) {
			oldestTime = v.lastAccess
			oldestKey = k
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
		c.evictions.Add(1)
		c.stats.TotalEvictions.Add(1)
	}
}

func (c *QueryCache) evictLFU() {
	lowestFreqKey := ""
	lowestFreq := float64(0)

	for k, v := range c.cache {
		if lowestFreqKey == "" || v.frequency < lowestFreq {
			lowestFreq = v.frequency
			lowestFreqKey = k
		}
	}

	if lowestFreqKey != "" {
		delete(c.cache, lowestFreqKey)
		c.evictions.Add(1)
		c.stats.TotalEvictions.Add(1)
	}
}

func (c *QueryCache) evictAdaptive() {
	now := time.Now()
	score := make(map[string]float64)

	for k, v := range c.cache {
		age := now.Sub(v.lastAccess).Seconds()
		freq := v.frequency
		ttlRemaining := v.expiration.Sub(now).Seconds()

		score[k] = freq * 10 / (age + 1) * (ttlRemaining / c.ttl.Seconds() + 0.1)
	}

	var lowestScoreKey string
	var lowestScore float64 = 0

	for k, s := range score {
		if lowestScoreKey == "" || s < lowestScore {
			lowestScore = s
			lowestScoreKey = k
		}
	}

	if lowestScoreKey != "" {
		delete(c.cache, lowestScoreKey)
		c.evictions.Add(1)
		c.stats.TotalEvictions.Add(1)
	}
}

func (c *QueryCache) startCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanupExpired()
		c.updateFrequencies()
	}
}

func (c *QueryCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, v := range c.cache {
		if now.After(v.expiration) {
			delete(c.cache, k)
			c.evictions.Add(1)
			c.stats.TotalEvictions.Add(1)
		}
	}
}

func (c *QueryCache) updateFrequencies() {
	c.mu.Lock()
	defer c.mu.Unlock()

	decayFactor := 0.95
	for k, v := range c.cache {
		v.frequency *= decayFactor
		c.cache[k] = v
	}
}

func (c *QueryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]cacheEntry)
}

func (c *QueryCache) ClearPattern(pattern string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k := range c.cache {
		if len(k) >= len(pattern) && k[:len(pattern)] == pattern {
			delete(c.cache, k)
		}
	}
}

func (c *QueryCache) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"size":            len(c.cache),
		"max_size":        c.maxSize,
		"enabled":         c.enabled,
		"hits":            c.hits.Load(),
		"misses":          c.misses.Load(),
		"evictions":        c.evictions.Load(),
		"hit_rate":        c.calculateHitRate(),
		"peak_size":       c.stats.PeakSize.Load(),
		"strategy":        c.getStrategyName(),
	}
}

func (c *QueryCache) calculateHitRate() float64 {
	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

func (c *QueryCache) getStrategyName() string {
	switch c.strategy {
	case StrategyLRU:
		return "LRU"
	case StrategyLFU:
		return "LFU"
	case StrategyAdaptive:
		return "Adaptive"
	case StrategyTTL:
		return "TTL"
	default:
		return "Unknown"
	}
}

func (c *QueryCache) SetStrategy(strategy QueryCacheStrategy) {
	c.strategy = strategy
}

func NewIntelligentQueryCache() *IntelligentQueryCache {
	return &IntelligentQueryCache{
		baseCache:     queryCache,
		queryPatterns: make(map[string]*QueryPattern),
	}
}

func (iqc *IntelligentQueryCache) RecordQuery(query string, latency time.Duration, cached bool) {
	iqc.mu.Lock()
	defer iqc.mu.Unlock()

	pattern, exists := iqc.queryPatterns[query]
	if !exists {
		pattern = &QueryPattern{
			Pattern:      query,
			OptimalTTL:   5 * time.Minute,
			Priority:     1,
			Complexity:   "normal",
		}
		iqc.queryPatterns[query] = pattern
	}

	pattern.TotalQueries++
	pattern.LastQuery = time.Now()

	if cached {
		pattern.HitRate = (pattern.HitRate*float64(pattern.TotalQueries-1) + 100) / float64(pattern.TotalQueries)
	} else {
		pattern.HitRate = (pattern.HitRate * float64(pattern.TotalQueries-1)) / float64(pattern.TotalQueries)
	}

	pattern.AvgLatency = (pattern.AvgLatency*time.Duration(pattern.TotalQueries-1) + latency) / time.Duration(pattern.TotalQueries)

	if latency > 100*time.Millisecond {
		pattern.Complexity = "high"
		pattern.OptimalTTL = 15 * time.Minute
		pattern.Priority = 3
	} else if latency > 10*time.Millisecond {
		pattern.Complexity = "medium"
		pattern.OptimalTTL = 10 * time.Minute
		pattern.Priority = 2
	} else {
		pattern.Complexity = "low"
		pattern.OptimalTTL = 5 * time.Minute
		pattern.Priority = 1
	}
}

func (iqc *IntelligentQueryCache) GetOptimalTTL(query string) time.Duration {
	iqc.mu.RLock()
	defer iqc.mu.RUnlock()

	if pattern, exists := iqc.queryPatterns[query]; exists {
		return pattern.OptimalTTL
	}
	return 5 * time.Minute
}

func (iqc *IntelligentQueryCache) GetPatternStats() map[string]interface{} {
	iqc.mu.RLock()
	defer iqc.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_patterns"] = len(iqc.queryPatterns)

	var totalHits float64
	var totalQueries int64
	for _, p := range iqc.queryPatterns {
		totalHits += p.HitRate
		totalQueries += p.TotalQueries
	}

	if len(iqc.queryPatterns) > 0 {
		stats["avg_hit_rate"] = totalHits / float64(len(iqc.queryPatterns))
	}
	stats["total_queries"] = totalQueries

	return stats
}

func (c *QueryCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cache) >= c.maxSize {
		c.evictByStrategy()
	}

	entry := cacheEntry{
		value:       value,
		expiration:  time.Now().Add(ttl),
		accessCount: 1,
		lastAccess:  time.Now(),
		frequency:   1.0,
	}

	c.cache[key] = entry
	c.stats.TotalSets.Add(1)

	if int64(len(c.cache)) > c.stats.PeakSize.Load() {
		c.stats.PeakSize.Store(int64(len(c.cache)))
	}
}

func (c *QueryCache) InvalidateKeysByPattern(pattern string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k := range c.cache {
		if strings.Contains(k, pattern) {
			delete(c.cache, k)
			c.stats.TotalEvictions.Add(1)
		}
	}
}

func (c *QueryCache) GetKeysByPattern(pattern string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var keys []string
	for k := range c.cache {
		if strings.Contains(k, pattern) {
			keys = append(keys, k)
		}
	}
	return keys
}

func (c *QueryCache) GetEntryInfo(key string) (*CacheEntryInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	return &CacheEntryInfo{
		Key:          key,
		AccessCount:  entry.accessCount,
		LastAccess:   entry.lastAccess,
		Expiration:   entry.expiration,
		Frequency:    entry.frequency,
		TTLRemaining: time.Until(entry.expiration),
	}, true
}

type CacheEntryInfo struct {
	Key          string
	AccessCount  int64
	LastAccess   time.Time
	Expiration   time.Time
	Frequency    float64
	TTLRemaining time.Duration
}

func (c *QueryCache) Warmup(entries map[string]interface{}, ttl time.Duration) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for key, value := range entries {
		if len(c.cache) >= c.maxSize {
			c.evictByStrategy()
		}

		entry := cacheEntry{
			value:       value,
			expiration:  time.Now().Add(ttl),
			accessCount: 1,
			lastAccess:  time.Now(),
			frequency:   1.0,
		}

		c.cache[key] = entry
		c.stats.TotalSets.Add(1)
	}
}

func (c *QueryCache) SmartWarmup(ctx context.Context, warmupFunc func() (map[string]interface{}, error), ttl time.Duration) error {
	if !c.enabled {
		return nil
	}

	data, err := warmupFunc()
	if err != nil {
		return err
	}

	c.Warmup(data, ttl)
	return nil
}

func (c *QueryCache) GetWarmupStatus() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	expiringSoon := 0
	expired := 0

	for _, entry := range c.cache {
		if now.After(entry.expiration) {
			expired++
		} else if time.Until(entry.expiration) < 5*time.Minute {
			expiringSoon++
		}
	}

	return map[string]interface{}{
		"total_entries":   len(c.cache),
		"expired_entries": expired,
		"expiring_soon":   expiringSoon,
		"max_size":        c.maxSize,
	}
}

func (c *QueryCache) AutoAdjustStrategy() {
	hitRate := c.calculateHitRate()

	if hitRate < 50 {
		c.strategy = StrategyLFU
	} else if hitRate < 70 {
		c.strategy = StrategyAdaptive
	} else {
		c.strategy = StrategyLRU
	}
}

func (c *QueryCache) GetStrategyPerformance() map[string]interface{} {
	return map[string]interface{}{
		"strategy":       c.getStrategyName(),
		"hit_rate":       c.calculateHitRate(),
		"evictions":      c.evictions.Load(),
		"current_size":   len(c.cache),
		"max_size":       c.maxSize,
	}
}

func (c *QueryCache) SetMaxSize(maxSize int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxSize = maxSize

	for len(c.cache) > c.maxSize {
		c.evictByStrategy()
	}
}

func (c *QueryCache) TTLBasedEviction() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, v := range c.cache {
		if now.After(v.expiration) {
			delete(c.cache, k)
			c.evictions.Add(1)
			c.stats.TotalEvictions.Add(1)
		}
	}
}

func (iqc *IntelligentQueryCache) AutoAdjustTTL() {
	iqc.mu.Lock()
	defer iqc.mu.Unlock()

	for _, pattern := range iqc.queryPatterns {
		if pattern.HitRate > 90 {
			pattern.OptimalTTL *= 2
			if pattern.OptimalTTL > 1*time.Hour {
				pattern.OptimalTTL = 1 * time.Hour
			}
		} else if pattern.HitRate < 30 {
			pattern.OptimalTTL /= 2
			if pattern.OptimalTTL < 30*time.Second {
				pattern.OptimalTTL = 30 * time.Second
			}
		}
	}
}

func (iqc *IntelligentQueryCache) GetTopPatterns(count int) []*QueryPattern {
	iqc.mu.RLock()
	defer iqc.mu.RUnlock()

	patterns := make([]*QueryPattern, 0, len(iqc.queryPatterns))
	for _, p := range iqc.queryPatterns {
		patterns = append(patterns, p)
	}

	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].TotalQueries > patterns[j].TotalQueries
	})

	if count > len(patterns) {
		count = len(patterns)
	}

	return patterns[:count]
}

func (iqc *IntelligentQueryCache) EvictLowPriority() {
	iqc.mu.Lock()
	defer iqc.mu.Unlock()

	thresholdTime := time.Now().Add(-10 * time.Minute)
	for query, pattern := range iqc.queryPatterns {
		if pattern.LastQuery.Before(thresholdTime) && pattern.Priority == 1 {
			delete(iqc.queryPatterns, query)
		}
	}
}
