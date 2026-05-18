package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/backend/pkg/redis"
)

type MultiLevelCacheService struct {
	l1Cache         *LocalCache
	l2Cache         *redis.EnhancedCache
	config          *MultiLevelConfig
	stats           *MultiLevelStats
	evictionPolicy  *redis.EnhancedCacheEvictor
	consistency     *redis.EnhancedCacheConsistency
	promotionPolicy *PromotionPolicy
	enabled         bool
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

type LocalCache struct {
	mu         sync.RWMutex
	data       map[string]*CacheItem
	maxSize    int
	evictor    *EvictionHelper
	hitCount   atomic.Int64
	missCount  atomic.Int64
}

type CacheItem struct {
	Key        string
	Value      []byte
	ExpiresAt  time.Time
	AccessedAt time.Time
	Frequency  int64
	Size       int
}

type MultiLevelConfig struct {
	L1Enabled        bool
	L2Enabled        bool
	L1MaxSize        int
	L1TTL            time.Duration
	L2TTL            time.Duration
	PromotionEnabled bool
	DemotionEnabled  bool
	PromoteOnHit     bool
	DemoteOnMiss     bool
	WriteThrough     bool
	WriteBehind      bool
	ConsistencyMode  string
}

var DefaultMultiLevelConfig = &MultiLevelConfig{
	L1Enabled:        true,
	L2Enabled:        true,
	L1MaxSize:        10000,
	L1TTL:            5 * time.Minute,
	L2TTL:            30 * time.Minute,
	PromotionEnabled: true,
	DemotionEnabled:  true,
	PromoteOnHit:     true,
	DemoteOnMiss:     false,
	WriteThrough:     true,
	WriteBehind:      false,
	ConsistencyMode:  "eventual",
}

type MultiLevelStats struct {
	L1Hits          atomic.Int64
	L1Misses        atomic.Int64
	L2Hits          atomic.Int64
	L2Misses        atomic.Int64
	Promotions      atomic.Int64
	Demotions       atomic.Int64
	TotalGets       atomic.Int64
	TotalSets       atomic.Int64
	TotalDeletes    atomic.Int64
	L1HitRate       atomic.Value
	L2HitRate       atomic.Value
	OverallHitRate  atomic.Value
	AvgLatency      atomic.Value
	LastUpdateTime  atomic.Value
}

type PromotionPolicy struct {
	mu               sync.RWMutex
	hotKeys          map[string]*HotKeyInfo
	promotionCount   int64
	demotionCount    int64
	threshold        int64
	windowSize       time.Duration
}

type HotKeyInfo struct {
	Key           string
	AccessCount   int64
	LastAccess    time.Time
	AvgLatency    time.Duration
	PromotionScore float64
}

func NewMultiLevelCacheService(config *MultiLevelConfig) *MultiLevelCacheService {
	if config == nil {
		config = DefaultMultiLevelConfig
	}

	ctx, cancel := context.WithCancel(context.Background())

	mlcs := &MultiLevelCacheService{
		l1Cache:        NewLocalCache(config.L1MaxSize, config.L1TTL),
		l2Cache:        redis.GetEnhancedCache(),
		config:         config,
		stats:          &MultiLevelStats{},
		evictionPolicy: redis.GetCacheEvictor(),
		consistency:    redis.GetEnhancedCacheConsistency(),
		promotionPolicy: NewPromotionPolicy(),
		enabled:        true,
		ctx:            ctx,
		cancel:         cancel,
	}

	mlcs.startBackgroundTasks()

	return mlcs
}

func NewLocalCache(maxSize int, ttl time.Duration) *LocalCache {
	return &LocalCache{
		data:     make(map[string]*CacheItem),
		maxSize:  maxSize,
		evictor:  NewEvictionHelper(),
	}
}

func (lc *LocalCache) Get(key string) ([]byte, bool) {
	lc.mu.RLock()
	item, exists := lc.data[key]
	lc.mu.RUnlock()

	if !exists {
		lc.missCount.Add(1)
		return nil, false
	}

	if !item.ExpiresAt.IsZero() && time.Now().After(item.ExpiresAt) {
		lc.Delete(key)
		lc.missCount.Add(1)
		return nil, false
	}

	lc.mu.Lock()
	item.AccessedAt = time.Now()
	item.Frequency++
	lc.mu.Unlock()

	lc.hitCount.Add(1)
	return item.Value, true
}

func (lc *LocalCache) Set(key string, value []byte, ttl time.Duration) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	item := &CacheItem{
		Key:        key,
		Value:      value,
		AccessedAt: time.Now(),
		Frequency:  1,
		Size:       len(value),
	}

	if ttl > 0 {
		item.ExpiresAt = time.Now().Add(ttl)
	}

	lc.data[key] = item

	if len(lc.data) > lc.maxSize {
		lc.evict()
	}
}

func (lc *LocalCache) Delete(key string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	delete(lc.data, key)
}

func (lc *LocalCache) Clear() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.data = make(map[string]*CacheItem)
}

func (lc *LocalCache) evict() {
	if len(lc.data) <= lc.maxSize {
		return
	}

	evictCount := len(lc.data) - lc.maxSize

	var candidates []string
	for key, item := range lc.data {
		candidates = append(candidates, key)
		if len(candidates) >= evictCount {
			break
		}
	}

	for _, key := range candidates {
		delete(lc.data, key)
	}
}

func (lc *LocalCache) EvictExpired() int {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	now := time.Now()
	evicted := 0

	for key, item := range lc.data {
		if !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
			delete(lc.data, key)
			evicted++
		}
	}

	return evicted
}

func (lc *LocalCache) GetStats() (hits, misses int64, hitRate float64) {
	hits = lc.hitCount.Load()
	misses = lc.missCount.Load()
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}
	return
}

func NewPromotionPolicy() *PromotionPolicy {
	return &PromotionPolicy{
		hotKeys:    make(map[string]*HotKeyInfo),
		threshold:  100,
		windowSize: 5 * time.Minute,
	}
}

func (pp *PromotionPolicy) RecordAccess(key string, latency time.Duration) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	info, exists := pp.hotKeys[key]
	if !exists {
		info = &HotKeyInfo{
			Key:        key,
			LastAccess: time.Now(),
		}
		pp.hotKeys[key] = info
	}

	info.AccessCount++
	info.LastAccess = time.Now()
	info.AvgLatency = (info.AvgLatency*time.Duration(info.AccessCount-1) + latency) / time.Duration(info.AccessCount)
	info.PromotionScore = pp.calculateScore(info)
}

func (pp *PromotionPolicy) calculateScore(info *HotKeyInfo) float64 {
	recencyWeight := 0.4
	frequencyWeight := 0.4
	latencyWeight := 0.2

	recencyScore := 1.0 - float64(time.Since(info.LastAccess).Seconds())/pp.windowSize.Seconds()
	if recencyScore < 0 {
		recencyScore = 0
	}

	normalizedFreq := float64(info.AccessCount) / 100.0
	if normalizedFreq > 1.0 {
		normalizedFreq = 1.0
	}

	latencyScore := 1.0 - float64(info.AvgLatency.Milliseconds())/1000.0
	if latencyScore < 0 {
		latencyScore = 0
	}

	return recencyWeight*recencyScore + frequencyWeight*normalizedFreq + latencyWeight*latencyScore
}

func (pp *PromotionPolicy) ShouldPromote(key string) bool {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	if info, exists := pp.hotKeys[key]; exists {
		return info.PromotionScore > 0.7 && info.AccessCount > pp.threshold
	}
	return false
}

func (pp *PromotionPolicy) GetHotKeys(count int) []string {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	type keyScore struct {
		Key   string
		Score float64
	}

	var scores []keyScore
	for key, info := range pp.hotKeys {
		scores = append(scores, keyScore{Key: key, Score: info.PromotionScore})
	}

	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[i].Score < scores[j].Score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	if count > len(scores) {
		count = len(scores)
	}

	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = scores[i].Key
	}

	return result
}

type EvictionHelper struct{}

func NewEvictionHelper() *EvictionHelper {
	return &EvictionHelper{}
}

func (eh *EvictionHelper) SelectEvictionCandidate(items map[string]*CacheItem) string {
	var bestKey string
	var bestScore float64

	for key, item := range items {
		score := eh.calculateEvictionScore(item)
		if bestKey == "" || score < bestScore {
			bestKey = key
			bestScore = score
		}
	}

	return bestKey
}

func (eh *EvictionHelper) calculateEvictionScore(item *CacheItem) float64 {
	recencyWeight := 0.5
	frequencyWeight := 0.3
	sizeWeight := 0.2

	recencyScore := float64(time.Since(item.AccessedAt).Seconds()) / 3600.0
	if recencyScore > 1.0 {
		recencyScore = 1.0
	}

	frequencyScore := 1.0 - float64(item.Frequency)/1000.0
	if frequencyScore < 0 {
		frequencyScore = 0
	}

	sizeScore := float64(item.Size) / (1024 * 1024)

	return recencyWeight*recencyScore + frequencyWeight*frequencyScore + sizeWeight*sizeScore
}

func (mlcs *MultiLevelCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	if !mlcs.enabled {
		return nil, redis.ErrCacheDisabled
	}

	mlcs.stats.TotalGets.Add(1)
	start := time.Now()

	if mlcs.config.L1Enabled {
		if val, found := mlcs.l1Cache.Get(key); found {
			mlcs.stats.L1Hits.Add(1)
			mlcs.promotionPolicy.RecordAccess(key, time.Since(start))

			if mlcs.config.PromoteOnHit {
				go mlcs.maybePromote(key)
			}

			mlcs.updateHitRates()
			return val, nil
		}
		mlcs.stats.L1Misses.Add(1)
	}

	if mlcs.config.L2Enabled {
		val, err := mlcs.l2Cache.Get(ctx, key, nil)
		if err == nil {
			mlcs.stats.L2Hits.Add(1)

			if mlcs.config.L1Enabled {
				mlcs.l1Cache.Set(key, val, mlcs.config.L1TTL)
			}

			mlcs.updateHitRates()
			return val, nil
		}
		mlcs.stats.L2Misses.Add(1)
	}

	mlcs.updateHitRates()
	return nil, redis.ErrCacheMiss
}

func (mlcs *MultiLevelCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if !mlcs.enabled {
		return redis.ErrCacheDisabled
	}

	mlcs.stats.TotalSets.Add(1)

	if mlcs.config.L1Enabled {
		mlcs.l1Cache.Set(key, value, mlcs.config.L1TTL)
	}

	if mlcs.config.WriteThrough || !mlcs.config.L2Enabled {
		if mlcs.config.L2Enabled {
			return mlcs.l2Cache.Set(ctx, key, value, &redis.SetOptions{
				TTL:   ttl,
				Level: redis.CacheLevelL2,
			})
		}
	}

	if mlcs.config.WriteBehind {
		go func() {
			mlcs.l2Cache.Set(ctx, key, value, &redis.SetOptions{
				TTL:   ttl,
				Level: redis.CacheLevelL2,
			})
		}()
	}

	return nil
}

func (mlcs *MultiLevelCacheService) Delete(ctx context.Context, keys ...string) error {
	if !mlcs.enabled {
		return nil
	}

	mlcs.stats.TotalDeletes.Add(int64(len(keys)))

	if mlcs.config.L1Enabled {
		for _, key := range keys {
			mlcs.l1Cache.Delete(key)
		}
	}

	if mlcs.config.L2Enabled {
		for _, key := range keys {
			mlcs.l2Cache.Delete(ctx, key, &redis.DeleteOptions{
				Level: redis.CacheLevelL2,
			})
		}
	}

	return nil
}

func (mlcs *MultiLevelCacheService) maybePromote(key string) {
	if !mlcs.promotionPolicy.ShouldPromote(key) {
		return
	}

	mlcs.promotionPolicy.mu.Lock()
	mlcs.promotionPolicy.promotionCount++
	mlcs.promotionPolicy.mu.Unlock()

	mlcs.stats.Promotions.Add(1)

	mlcs.l1Cache.mu.Lock()
	defer mlcs.l1Cache.mu.Unlock()

	if _, exists := mlcs.l1Cache.data[key]; !exists {
		mlcs.l1Cache.Set(key, nil, mlcs.config.L1TTL)
	}
}

func (mlcs *MultiLevelCacheService) updateHitRates() {
	l1Hits := mlcs.stats.L1Hits.Load()
	l1Misses := mlcs.stats.L1Misses.Load()
	l1Total := l1Hits + l1Misses

	if l1Total > 0 {
		mlcs.stats.L1HitRate.Store(float64(l1Hits) / float64(l1Total) * 100)
	}

	l2Hits := mlcs.stats.L2Hits.Load()
	l2Misses := mlcs.stats.L2Misses.Load()
	l2Total := l2Hits + l2Misses

	if l2Total > 0 {
		mlcs.stats.L2HitRate.Store(float64(l2Hits) / float64(l2Total) * 100)
	}

	totalHits := l1Hits + l2Hits
	totalMisses := l1Misses + l2Misses
	total := totalHits + totalMisses

	if total > 0 {
		mlcs.stats.OverallHitRate.Store(float64(totalHits) / float64(total) * 100)
	}

	mlcs.stats.LastUpdateTime.Store(time.Now())
}

func (mlcs *MultiLevelCacheService) startBackgroundTasks() {
	mlcs.wg.Add(1)
	go mlcs.evictionWorker()

	mlcs.wg.Add(1)
	go mlcs.consistencyWorker()
}

func (mlcs *MultiLevelCacheService) evictionWorker() {
	defer mlcs.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-mlcs.ctx.Done():
			return
		case <-ticker.C:
			if mlcs.config.L1Enabled {
				mlcs.l1Cache.EvictExpired()
			}
		}
	}
}

func (mlcs *MultiLevelCacheService) consistencyWorker() {
	defer mlcs.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-mlcs.ctx.Done():
			return
		case <-ticker.C:
			if mlcs.config.ConsistencyMode != "" && mlcs.consistency != nil {
			}
		}
	}
}

func (mlcs *MultiLevelCacheService) GetStats() *MultiLevelStats {
	return mlcs.stats
}

func (mlcs *MultiLevelCacheService) Clear() error {
	if mlcs.config.L1Enabled {
		mlcs.l1Cache.Clear()
	}

	if mlcs.config.L2Enabled {
		mlcs.l2Cache.Clear(mlcs.ctx, redis.CacheLevelBoth)
	}

	return nil
}

func (mlcs *MultiLevelCacheService) Close() {
	mlcs.mu.Lock()
	mlcs.enabled = false
	mlcs.mu.Unlock()

	mlcs.cancel()
	mlcs.wg.Wait()
}

func (mlcs *MultiLevelCacheService) SetEnabled(enabled bool) {
	mlcs.mu.Lock()
	defer mlcs.mu.Unlock()
	mlcs.enabled = enabled
}

func (mlcs *MultiLevelCacheService) IsEnabled() bool {
	mlcs.mu.RLock()
	defer mlcs.mu.RUnlock()
	return mlcs.enabled
}

var (
	globalMultiLevelCache *MultiLevelCacheService
	globalMLCOnce         sync.Once
)

func InitMultiLevelCache(config *MultiLevelConfig) {
	globalMLCOnce.Do(func() {
		globalMultiLevelCache = NewMultiLevelCacheService(config)
	})
}

func GetMultiLevelCache() *MultiLevelCacheService {
	if globalMultiLevelCache == nil {
		InitMultiLevelCache(nil)
	}
	return globalMultiLevelCache
}
