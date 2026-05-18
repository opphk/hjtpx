package database

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

type QueryCacheV2 struct {
	mu        sync.RWMutex
	cache     map[string]cacheEntryV2
	maxSize   int
	ttl       time.Duration
	enabled   bool
	stats     *QueryCacheStatsV2
	strategy  QueryCacheStrategyV2
	hits      atomic.Int64
	misses    atomic.Int64
	evictions atomic.Int64
}

type cacheEntryV2 struct {
	value       interface{}
	expiration  time.Time
	accessCount int64
	lastAccess  time.Time
	frequency   float64
}

type QueryCacheStatsV2 struct {
	TotalHits     atomic.Int64
	TotalMisses   atomic.Int64
	TotalSets     atomic.Int64
	TotalEvictions atomic.Int64
	AvgLatency    atomic.Int64
	PeakSize      atomic.Int64
}

type QueryCacheStrategyV2 int

const (
	StrategyLRUV2 QueryCacheStrategyV2 = iota
	StrategyLFUV2
	StrategyAdaptiveV2
	StrategyTTLV2
)

type IntelligentQueryCacheV2 struct {
	baseCache     *QueryCacheV2
	queryPatterns map[string]*QueryPatternV2
	mu            sync.RWMutex
}

type QueryPatternV2 struct {
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

var queryCacheV2 *QueryCacheV2
var intelligentCacheV2 *IntelligentQueryCacheV2

func InitQueryCache(cfg *config.Config) {
	queryCacheV2 = &QueryCacheV2{
		cache:    make(map[string]cacheEntryV2),
		maxSize:  cfg.Database.QueryOptimization.MaxQueryCacheSize,
		ttl:      time.Duration(cfg.Database.QueryOptimization.QueryCacheTTLSecs) * time.Second,
		enabled:  cfg.Database.QueryOptimization.EnableQueryCache,
		stats:    &QueryCacheStatsV2{},
		strategy: StrategyAdaptiveV2,
	}

	intelligentCacheV2 = &IntelligentQueryCacheV2{
		baseCache:     queryCacheV2,
		queryPatterns: make(map[string]*QueryPatternV2),
	}

	go queryCacheV2.startCleanup()

	if queryCacheV2.enabled {
		log.Println("Query cache initialized with intelligent strategy")
	}
}

func GetIntelligentCache() *IntelligentQueryCacheV2 {
	if intelligentCacheV2 == nil {
		intelligentCacheV2 = NewIntelligentQueryCache()
	}
	return intelligentCacheV2
}

func GetQueryCache() *QueryCacheV2 {
	return queryCacheV2
}

func (qc *QueryCacheV2) startCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		qc.cleanup()
	}
}

func (qc *QueryCacheV2) cleanup() {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	
	now := time.Now()
	for key, entry := range qc.cache {
		if now.After(entry.expiration) {
			delete(qc.cache, key)
			qc.evictions.Add(1)
		}
	}
	
	if len(qc.cache) > qc.maxSize {
		qc.evictLRU()
	}
}

func (qc *QueryCacheV2) evictLRU() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range qc.cache {
		if oldestKey == "" || entry.lastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.lastAccess
		}
	}
	
	if oldestKey != "" {
		delete(qc.cache, oldestKey)
		qc.evictions.Add(1)
	}
}

func (qc *QueryCacheV2) Get(key string) (interface{}, bool) {
	if !qc.enabled {
		return nil, false
	}
	
	qc.mu.RLock()
	defer qc.mu.RUnlock()
	
	entry, exists := qc.cache[key]
	if !exists {
		qc.misses.Add(1)
		return nil, false
	}
	
	if time.Now().After(entry.expiration) {
		qc.mu.RUnlock()
		qc.mu.Lock()
		delete(qc.cache, key)
		qc.mu.Unlock()
		qc.mu.RLock()
		qc.misses.Add(1)
		return nil, false
	}
	
	entry.accessCount++
	entry.lastAccess = time.Now()
	qc.hits.Add(1)
	return entry.value, true
}

func (qc *QueryCacheV2) Set(key string, value interface{}) {
	if !qc.enabled {
		return
	}
	
	qc.mu.Lock()
	defer qc.mu.Unlock()
	
	if len(qc.cache) >= qc.maxSize {
		qc.evictLRU()
	}
	
	qc.cache[key] = cacheEntryV2{
		value:       value,
		expiration:  time.Now().Add(qc.ttl),
		accessCount: 1,
		lastAccess:  time.Now(),
	}
}

func (qc *QueryCacheV2) generateKey(suffix string) string {
	h := md5.New()
	h.Write([]byte(suffix))
	return hex.EncodeToString(h.Sum(nil))
}

func CachedQuery(keySuffix string, dest interface{}, queryFunc func() error, ttl ...time.Duration) error {
	if !queryCacheV2.enabled {
		return queryFunc()
	}

	key := queryCacheV2.generateKey(keySuffix)
	if cached, ok := queryCacheV2.Get(key); ok {
		if cachedData, err := json.Marshal(cached); err == nil {
			json.Unmarshal(cachedData, dest)
			return nil
		}
	}

	if err := queryFunc(); err != nil {
		return err
	}

	queryCacheV2.Set(key, dest)
	return nil
}

func InvalidateQueryCache(tableName string) {
	if queryCacheV2 != nil {
		queryCacheV2.mu.Lock()
		defer queryCacheV2.mu.Unlock()
		
		for key := range queryCacheV2.cache {
			if len(tableName) == 0 || len(key) >= len(tableName) && key[:len(tableName)] == tableName {
				delete(queryCacheV2.cache, key)
			}
		}
	}
}

func NewIntelligentQueryCache() *IntelligentQueryCacheV2 {
	return &IntelligentQueryCacheV2{
		baseCache:     queryCacheV2,
		queryPatterns: make(map[string]*QueryPatternV2),
	}
}

func (ic *IntelligentQueryCacheV2) Get(key string) (interface{}, bool) {
	if ic.baseCache == nil {
		return nil, false
	}
	return ic.baseCache.Get(key)
}

func (ic *IntelligentQueryCacheV2) Set(key string, value interface{}) {
	if ic.baseCache == nil {
		return
	}
	ic.baseCache.Set(key, value)
}

func (ic *IntelligentQueryCacheV2) RecordQuery(pattern string, latency time.Duration, cached bool) {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	
	p, exists := ic.queryPatterns[pattern]
	if !exists {
		p = &QueryPatternV2{
			Pattern: pattern,
		}
		ic.queryPatterns[pattern] = p
	}
	
	p.TotalQueries++
	p.LastQuery = time.Now()
	
	if cached {
		p.HitRate = (p.HitRate*float64(p.TotalQueries-1) + 100) / float64(p.TotalQueries)
	} else {
		p.HitRate = (p.HitRate * float64(p.TotalQueries-1)) / float64(p.TotalQueries)
	}
	
	p.AvgLatency = (p.AvgLatency*time.Duration(p.TotalQueries-1) + latency) / time.Duration(p.TotalQueries)
}

func GetCacheStats() map[string]interface{} {
	if queryCacheV2 == nil {
		return nil
	}
	
	return map[string]interface{}{
		"hits":      queryCacheV2.hits.Load(),
		"misses":    queryCacheV2.misses.Load(),
		"evictions": queryCacheV2.evictions.Load(),
		"size":      func() int {
			queryCacheV2.mu.RLock()
			defer queryCacheV2.mu.RUnlock()
			return len(queryCacheV2.cache)
		}(),
	}
}

func GetCacheHitRate() float64 {
	if queryCacheV2 == nil {
		return 0
	}
	hits := queryCacheV2.hits.Load()
	misses := queryCacheV2.misses.Load()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

func ClearQueryCache() {
	if queryCacheV2 != nil {
		queryCacheV2.mu.Lock()
		defer queryCacheV2.mu.Unlock()
		queryCacheV2.cache = make(map[string]cacheEntryV2)
	}
}
