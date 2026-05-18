package redis

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"sync"
	"sync/atomic"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var (
	ErrCacheMiss     = errors.New("cache miss")
	ErrCacheDisabled = errors.New("cache disabled")
	ErrCircuitOpen   = errors.New("circuit breaker open")
	ErrLockTimeout   = errors.New("lock timeout")
	ErrKeyNotFound   = errors.New("key not found")
)

const (
	DefaultMaxCacheSize      = 10000
	DefaultL1TTL             = 5 * time.Minute
	DefaultL2TTL             = 30 * time.Minute
	DefaultLockTimeout       = 10 * time.Second
	DefaultCompressThreshold = 1024 // 1KB
)

type CacheLevel int

const (
	CacheLevelL1 CacheLevel = iota
	CacheLevelL2
	CacheLevelBoth
)

type CacheConfig struct {
	Enabled           bool
	L1Enabled         bool
	L2Enabled         bool
	L1Size            int
	L1TTL             time.Duration
	L2TTL             time.Duration
	CompressEnabled   bool
	CompressThreshold int
	StatsEnabled      bool
	HotKeyThreshold   int64
	BreakerThreshold  int
	BreakerTimeout    time.Duration
}

var DefaultCacheConfig = &CacheConfig{
	Enabled:           true,
	L1Enabled:         true,
	L2Enabled:         true,
	L1Size:            DefaultMaxCacheSize,
	L1TTL:             DefaultL1TTL,
	L2TTL:             DefaultL2TTL,
	CompressEnabled:   true,
	CompressThreshold: DefaultCompressThreshold,
	StatsEnabled:      true,
	HotKeyThreshold:   100,
	BreakerThreshold:  5,
	BreakerTimeout:    30 * time.Second,
}

type l1Entry struct {
	value      []byte
	expiresAt  time.Time
	version    int64
	accessTime time.Time
}

type EnhancedCache struct {
	config         *CacheConfig
	l1Cache        *sync.Map
	l1Metrics      *l1Metrics
	stats          *CacheStats
	breaker        *CircuitBreaker
	hotKeys        *sync.Map
	bloomFilter    *BloomFilter
	versionManager *VersionManager
	metrics        *Metrics
	mu             sync.RWMutex
}

type l1Metrics struct {
	hits      atomic.Int64
	misses    atomic.Int64
	evictions atomic.Int64
	size      atomic.Int64
}

type CacheStats struct {
	Hits         atomic.Int64
	Misses       atomic.Int64
	Sets         atomic.Int64
	Deletes      atomic.Int64
	Compressed   atomic.Int64
	Decompressed atomic.Int64
	L1Hits       atomic.Int64
	L1Misses     atomic.Int64
	L2Hits       atomic.Int64
	L2Misses     atomic.Int64
	Errors       atomic.Int64
	TotalLatency atomic.Int64
	RequestCount atomic.Int64
	Expired      atomic.Int64
}

type CircuitBreaker struct {
	mu           sync.RWMutex
	failures     int
	lastFailure  time.Time
	threshold    int
	resetTimeout time.Duration
	state        string
}

type VersionManager struct {
	versions *sync.Map
}

type HotKeyInfo struct {
	Key         string
	AccessCount int64
	LastAccess  time.Time
}

type CacheEntry struct {
	Key        string
	Value      []byte
	TTL        time.Duration
	Version    int64
	Tags       []string
	Compressed bool
	Level      CacheLevel
}

type GetOptions struct {
	Level      CacheLevel
	SkipLocal  bool
	SkipRemote bool
}

type SetOptions struct {
	Level    CacheLevel
	TTL      time.Duration
	Version  int64
	Tags     []string
	Compress bool
}

type DeleteOptions struct {
	Level CacheLevel
	ByTag bool
}

func NewEnhancedCache(config *CacheConfig) *EnhancedCache {
	if config == nil {
		config = DefaultCacheConfig
	}

	cache := &EnhancedCache{
		config:         config,
		l1Cache:        &sync.Map{},
		l1Metrics:      &l1Metrics{},
		stats:          &CacheStats{},
		breaker:        NewCircuitBreaker(config.BreakerThreshold, config.BreakerTimeout),
		hotKeys:        &sync.Map{},
		bloomFilter:    NewBloomFilter(100000, 0.01),
		versionManager: NewVersionManager(),
		metrics:        NewMetrics(),
	}

	if config.L1Enabled {
		go cache.startL1Eviction()
	}

	if config.StatsEnabled {
		go cache.startHotKeyTracking()
	}

	return cache
}

func (ec *EnhancedCache) Get(ctx context.Context, key string, opts *GetOptions) ([]byte, error) {
	if !ec.config.Enabled {
		return nil, ErrCacheDisabled
	}

	if opts == nil {
		opts = &GetOptions{Level: CacheLevelBoth}
	}

	start := time.Now()
	defer ec.recordLatency(start)
	ec.stats.RequestCount.Add(1)

	ec.trackHotKey(key)

	if !ec.bloomFilter.MayContain(key) {
		ec.stats.Misses.Add(1)
		return nil, ErrCacheMiss
	}

	if ec.config.L1Enabled && opts.Level != CacheLevelL2 && !opts.SkipLocal {
		if val, err := ec.getFromL1(key); err == nil {
			ec.stats.Hits.Add(1)
			ec.stats.L1Hits.Add(1)
			return val, nil
		}
		ec.stats.L1Misses.Add(1)
	}

	if ec.config.L2Enabled && opts.Level != CacheLevelL1 && !opts.SkipRemote {
		if val, err := ec.getFromL2(ctx, key); err == nil {
			ec.stats.Hits.Add(1)
			ec.stats.L2Hits.Add(1)
			if ec.config.L1Enabled {
				ec.setToL1(key, val, ec.config.L1TTL, 0)
			}
			return val, nil
		}
		ec.stats.L2Misses.Add(1)
	}

	ec.stats.Misses.Add(1)
	return nil, ErrCacheMiss
}

func (ec *EnhancedCache) Set(ctx context.Context, key string, value []byte, opts *SetOptions) error {
	if !ec.config.Enabled {
		return ErrCacheDisabled
	}

	if opts == nil {
		opts = &SetOptions{Level: CacheLevelBoth}
	}

	ec.stats.Sets.Add(1)

	var compressed []byte
	var useCompression bool
	if ec.config.CompressEnabled && len(value) >= ec.config.CompressThreshold {
		var err error
		compressed, err = compress(value)
		if err == nil {
			value = compressed
			useCompression = true
			ec.stats.Compressed.Add(1)
		}
	}

	if opts.TTL == 0 {
		if opts.Level == CacheLevelL1 {
			opts.TTL = ec.config.L1TTL
		} else {
			opts.TTL = ec.config.L2TTL
		}
	}

	if ec.config.L1Enabled && (opts.Level == CacheLevelL1 || opts.Level == CacheLevelBoth) {
		ec.setToL1(key, value, opts.TTL, opts.Version)
	}

	if ec.config.L2Enabled && (opts.Level == CacheLevelL2 || opts.Level == CacheLevelBoth) {
		if err := ec.setToL2(ctx, key, value, opts.TTL, opts.Version, opts.Tags, useCompression); err != nil {
			ec.stats.Errors.Add(1)
			return err
		}
	}

	ec.bloomFilter.Add(key)
	return nil
}

func (ec *EnhancedCache) Delete(ctx context.Context, key string, opts *DeleteOptions) error {
	if !ec.config.Enabled {
		return ErrCacheDisabled
	}

	if opts == nil {
		opts = &DeleteOptions{Level: CacheLevelBoth}
	}

	ec.stats.Deletes.Add(1)

	if opts.ByTag {
		return ec.deleteByTag(ctx, key)
	}

	if ec.config.L1Enabled && (opts.Level == CacheLevelL1 || opts.Level == CacheLevelBoth) {
		ec.deleteFromL1(key)
	}

	if ec.config.L2Enabled && (opts.Level == CacheLevelL2 || opts.Level == CacheLevelBoth) {
		if err := ec.deleteFromL2(ctx, key); err != nil {
			ec.stats.Errors.Add(1)
			return err
		}
	}

	return nil
}

func (ec *EnhancedCache) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error) {
	val, err := ec.Get(ctx, key, nil)
	if err == nil {
		return val, nil
	}

	if errors.Is(err, ErrCacheMiss) {
		data, err := fn()
		if err != nil {
			return nil, err
		}

		if err := ec.Set(ctx, key, data, &SetOptions{TTL: ttl}); err != nil {
		}

		return data, nil
	}

	return nil, err
}

func (ec *EnhancedCache) GetJSON(ctx context.Context, key string, dest interface{}, opts *GetOptions) error {
	data, err := ec.Get(ctx, key, opts)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (ec *EnhancedCache) SetJSON(ctx context.Context, key string, value interface{}, opts *SetOptions) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return ec.Set(ctx, key, data, opts)
}

func (ec *EnhancedCache) IncrementVersion(ctx context.Context, key string) (int64, error) {
	return ec.versionManager.Increment(key)
}

func (ec *EnhancedCache) GetVersion(ctx context.Context, key string) (int64, error) {
	return ec.versionManager.Get(key)
}

func (ec *EnhancedCache) getFromL1(key string) ([]byte, error) {
	val, ok := ec.l1Cache.Load(key)
	if !ok {
		ec.l1Metrics.misses.Add(1)
		return nil, ErrCacheMiss
	}

	entry := val.(*l1Entry)
	if time.Now().After(entry.expiresAt) {
		ec.l1Cache.Delete(key)
		ec.l1Metrics.evictions.Add(1)
		return nil, ErrCacheMiss
	}

	entry.accessTime = time.Now()
	ec.l1Metrics.hits.Add(1)

	var value []byte
	var err error
	if ec.config.CompressEnabled {
		value, err = decompress(entry.value)
		if err != nil {
			value = entry.value
		} else {
			ec.stats.Decompressed.Add(1)
		}
	} else {
		value = entry.value
	}

	return value, nil
}

func (ec *EnhancedCache) setToL1(key string, value []byte, ttl time.Duration, version int64) {
	entry := &l1Entry{
		value:      value,
		expiresAt:  time.Now().Add(ttl),
		version:    version,
		accessTime: time.Now(),
	}

	ec.l1Cache.Store(key, entry)
	ec.l1Metrics.size.Add(1)

	if ec.l1Metrics.size.Load() > int64(ec.config.L1Size) {
		ec.evictLRU()
	}
}

func (ec *EnhancedCache) deleteFromL1(key string) {
	ec.l1Cache.Delete(key)
}

func (ec *EnhancedCache) evictLRU() {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	var oldestKey string
	var oldestTime time.Time

	ec.l1Cache.Range(func(key, value interface{}) bool {
		entry := value.(*l1Entry)
		if oldestKey == "" || entry.accessTime.Before(oldestTime) {
			oldestKey = key.(string)
			oldestTime = entry.accessTime
		}
		return true
	})

	if oldestKey != "" {
		ec.l1Cache.Delete(oldestKey)
		ec.l1Metrics.size.Add(-1)
		ec.l1Metrics.evictions.Add(1)
	}
}

func (ec *EnhancedCache) evictLRUWithCount(count int) int {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	evicted := 0
	for i := 0; i < count; i++ {
		var oldestKey string
		var oldestTime time.Time

		ec.l1Cache.Range(func(key, value interface{}) bool {
			entry := value.(*l1Entry)
			if oldestKey == "" || entry.accessTime.Before(oldestTime) {
				oldestKey = key.(string)
				oldestTime = entry.accessTime
			}
			return true
		})

		if oldestKey == "" {
			break
		}

		ec.l1Cache.Delete(oldestKey)
		ec.l1Metrics.size.Add(-1)
		ec.l1Metrics.evictions.Add(1)
		evicted++
	}

	return evicted
}

func (ec *EnhancedCache) evictByTTL() int {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	now := time.Now()
	evicted := 0

	ec.l1Cache.Range(func(key, value interface{}) bool {
		entry := value.(*l1Entry)
		if now.After(entry.expiresAt) {
			ec.l1Cache.Delete(key)
			ec.l1Metrics.size.Add(-1)
			ec.l1Metrics.evictions.Add(1)
			evicted++
		}
		return true
	})

	return evicted
}

func (ec *EnhancedCache) getEvictionStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["total_evictions"] = ec.l1Metrics.evictions.Load()
	stats["current_size"] = ec.l1Metrics.size.Load()
	stats["max_size"] = ec.config.L1Size
	stats["hit_rate"] = ec.l1Metrics.hits.Load()
	stats["miss_rate"] = ec.l1Metrics.misses.Load()

	return stats
}

func (ec *EnhancedCache) startL1Eviction() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		ec.l1Cache.Range(func(key, value interface{}) bool {
			entry := value.(*l1Entry)
			if now.After(entry.expiresAt) {
				ec.l1Cache.Delete(key)
				ec.l1Metrics.size.Add(-1)
				ec.l1Metrics.evictions.Add(1)
			}
			return true
		})
	}
}

func (ec *EnhancedCache) getFromL2(ctx context.Context, key string) ([]byte, error) {
	if !ec.breaker.Allow() {
		return nil, ErrCircuitOpen
	}

	if Client == nil {
		return nil, ErrCacheMiss
	}

	val, err := Client.Get(ctx, key).Bytes()
	if err != nil {
		if err == goredis.Nil {
			return nil, ErrCacheMiss
		}
		ec.breaker.RecordFailure()
		ec.stats.Errors.Add(1)
		return nil, err
	}

	ec.breaker.RecordSuccess()

	var decompressed []byte
	if ec.config.CompressEnabled {
		decompressed, err = decompress(val)
		if err == nil {
			val = decompressed
			ec.stats.Decompressed.Add(1)
		}
	}

	return val, nil
}

func (ec *EnhancedCache) setToL2(ctx context.Context, key string, value []byte, ttl time.Duration, version int64, tags []string, compressed bool) error {
	if Client == nil {
		return nil
	}

	if !ec.breaker.Allow() {
		return ErrCircuitOpen
	}

	pipe := Client.Pipeline()

	pipe.Set(ctx, key, value, ttl)

	if version > 0 {
		pipe.HSet(ctx, fmt.Sprintf("cache:version:%s", key), "version", version)
	}

	if len(tags) > 0 {
		for _, tag := range tags {
			pipe.SAdd(ctx, fmt.Sprintf("cache:tag:%s", tag), key)
		}
	}

	if compressed {
		pipe.HSet(ctx, fmt.Sprintf("cache:meta:%s", key), "compressed", "1")
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		ec.breaker.RecordFailure()
		ec.stats.Errors.Add(1)
		return err
	}

	ec.breaker.RecordSuccess()
	return nil
}

func (ec *EnhancedCache) deleteFromL2(ctx context.Context, key string) error {
	if Client == nil {
		return nil
	}

	return Client.Del(ctx, key).Err()
}

func (ec *EnhancedCache) deleteByTag(ctx context.Context, tag string) error {
	if Client == nil {
		return nil
	}

	tagKey := fmt.Sprintf("cache:tag:%s", tag)
	keys, err := Client.SMembers(ctx, tagKey).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		pipe := Client.Pipeline()
		for _, key := range keys {
			pipe.Del(ctx, key)
			ec.deleteFromL1(key)
		}
		pipe.Del(ctx, tagKey)
		_, err = pipe.Exec(ctx)
	}

	return err
}

func (ec *EnhancedCache) trackHotKey(key string) {
	if !ec.config.StatsEnabled {
		return
	}

	val, _ := ec.hotKeys.LoadOrStore(key, &HotKeyInfo{
		Key:         key,
		AccessCount: 0,
		LastAccess:  time.Now(),
	})

	info := val.(*HotKeyInfo)
	atomic.AddInt64(&info.AccessCount, 1)
	info.LastAccess = time.Now()
}

func (ec *EnhancedCache) startHotKeyTracking() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ec.hotKeys.Range(func(key, value interface{}) bool {
			info := value.(*HotKeyInfo)
			if time.Since(info.LastAccess) > 30*time.Minute {
				ec.hotKeys.Delete(key)
			}
			return true
		})
	}
}

func (ec *EnhancedCache) GetHotKeys() []*HotKeyInfo {
	var hotKeys []*HotKeyInfo
	ec.hotKeys.Range(func(key, value interface{}) bool {
		info := value.(*HotKeyInfo)
		if info.AccessCount >= ec.config.HotKeyThreshold {
			hotKeys = append(hotKeys, info)
		}
		return true
	})
	return hotKeys
}

func (ec *EnhancedCache) recordLatency(start time.Time) {
	latency := time.Since(start).Nanoseconds()
	ec.stats.TotalLatency.Add(latency)
}

func (ec *EnhancedCache) GetStats() *CacheStatsSnapshot {
	return &CacheStatsSnapshot{
		Hits:         ec.stats.Hits.Load(),
		Misses:       ec.stats.Misses.Load(),
		Sets:         ec.stats.Sets.Load(),
		Deletes:      ec.stats.Deletes.Load(),
		Compressed:   ec.stats.Compressed.Load(),
		Decompressed: ec.stats.Decompressed.Load(),
		L1Hits:       ec.stats.L1Hits.Load(),
		L1Misses:     ec.stats.L1Misses.Load(),
		L2Hits:       ec.stats.L2Hits.Load(),
		L2Misses:     ec.stats.L2Misses.Load(),
		Errors:       ec.stats.Errors.Load(),
		HitRate:      ec.calculateHitRate(),
		AvgLatency:   ec.calculateAvgLatency(),
	}
}

func (ec *EnhancedCache) calculateHitRate() float64 {
	hits := ec.stats.Hits.Load()
	misses := ec.stats.Misses.Load()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

func (ec *EnhancedCache) calculateAvgLatency() time.Duration {
	total := ec.stats.RequestCount.Load()
	if total == 0 {
		return 0
	}
	return time.Duration(ec.stats.TotalLatency.Load() / total)
}

func (ec *EnhancedCache) Clear(ctx context.Context, level CacheLevel) error {
	if level == CacheLevelL1 || level == CacheLevelBoth {
		ec.l1Cache = &sync.Map{}
		ec.l1Metrics = &l1Metrics{}
	}

	if level == CacheLevelL2 || level == CacheLevelBoth {
		if Client != nil {
			iter := Client.Scan(ctx, 0, "*", 0).Iterator()
			var keys []string
			for iter.Next(ctx) {
				keys = append(keys, iter.Val())
			}
			if len(keys) > 0 {
				Client.Del(ctx, keys...)
			}
		}
	}

	return nil
}

type CacheStatsSnapshot struct {
	Hits         int64
	Misses       int64
	Sets         int64
	Deletes      int64
	Compressed   int64
	Decompressed int64
	L1Hits       int64
	L1Misses     int64
	L2Hits       int64
	L2Misses     int64
	Errors       int64
	HitRate      float64
	AvgLatency   time.Duration
}

func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold:    threshold,
		resetTimeout: timeout,
		state:        "closed",
	}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case "closed":
		return true
	case "open":
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = "half-open"
			return true
		}
		return false
	case "half-open":
		return true
	}
	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = "closed"
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.threshold {
		cb.state = "open"
	}
}

func (cb *CircuitBreaker) State() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func NewVersionManager() *VersionManager {
	return &VersionManager{
		versions: &sync.Map{},
	}
}

func (vm *VersionManager) Increment(key string) (int64, error) {
	val, _ := vm.versions.LoadOrStore(key, int64(0))
	newVal := val.(int64) + 1
	vm.versions.Store(key, newVal)
	return newVal, nil
}

func (vm *VersionManager) Get(key string) (int64, error) {
	val, ok := vm.versions.Load(key)
	if !ok {
		return 0, nil
	}
	return val.(int64), nil
}

func (vm *VersionManager) Set(key string, version int64) {
	vm.versions.Store(key, version)
}

type BloomFilter struct {
	m    uint64
	k    uint64
	bits []uint64
	mu   sync.RWMutex
}

func NewBloomFilter(expectedItems uint64, falsePositiveRate float64) *BloomFilter {
	m := optimalM(expectedItems, falsePositiveRate)
	k := optimalK(m, expectedItems)

	return &BloomFilter{
		m:    m,
		k:    k,
		bits: make([]uint64, (m+63)/64),
	}
}

func optimalM(n uint64, p float64) uint64 {
	m := -float64(n) * math.Log(p) / (math.Ln2 * math.Ln2)
	return uint64(m)
}

func optimalK(m, n uint64) uint64 {
	k := math.Ln2 * float64(m) / float64(n)
	return uint64(math.Max(1, math.Min(k, 30)))
}

func (bf *BloomFilter) Add(item string) {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	h := fnv.New64a()
	h.Write([]byte(item))
	hash1 := h.Sum64()

	h.Reset()
	h.Write([]byte("salt"))
	h.Write([]byte(item))
	hash2 := h.Sum64()

	for i := uint64(0); i < bf.k; i++ {
		hash := hash1 + i*hash2
		bitPos := hash % bf.m
		wordPos := bitPos / 64
		bitIdx := bitPos % 64
		bf.bits[wordPos] |= 1 << bitIdx
	}
}

func (bf *BloomFilter) MayContain(item string) bool {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	h := fnv.New64a()
	h.Write([]byte(item))
	hash1 := h.Sum64()

	h.Reset()
	h.Write([]byte("salt"))
	h.Write([]byte(item))
	hash2 := h.Sum64()

	for i := uint64(0); i < bf.k; i++ {
		hash := hash1 + i*hash2
		bitPos := hash % bf.m
		wordPos := bitPos / 64
		bitIdx := bitPos % 64
		if (bf.bits[wordPos] & (1 << bitIdx)) == 0 {
			return false
		}
	}
	return true
}

func (bf *BloomFilter) Clear() {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	bf.bits = make([]uint64, (bf.m+63)/64)
}

type Metrics struct {
	gauges   map[string]*atomic.Int64
	counters map[string]*atomic.Int64
	mu       sync.RWMutex
}

func NewMetrics() *Metrics {
	return &Metrics{
		gauges:   make(map[string]*atomic.Int64),
		counters: make(map[string]*atomic.Int64),
	}
}

func (m *Metrics) IncCounter(name string) {
	m.mu.RLock()
	counter, ok := m.counters[name]
	m.mu.RUnlock()

	if !ok {
		m.mu.Lock()
		if counter, ok = m.counters[name]; !ok {
			counter = &atomic.Int64{}
			m.counters[name] = counter
		}
		m.mu.Unlock()
	}

	counter.Add(1)
}

func (m *Metrics) SetGauge(name string, value int64) {
	m.mu.RLock()
	gauge, ok := m.gauges[name]
	m.mu.RUnlock()

	if !ok {
		m.mu.Lock()
		if gauge, ok = m.gauges[name]; !ok {
			gauge = &atomic.Int64{}
			m.gauges[name] = gauge
		}
		m.mu.Unlock()
	}

	gauge.Store(value)
}

func compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decompress(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, errors.New("invalid compressed data")
	}

	if data[0] != 0x1f || data[1] != 0x8b {
		return nil, errors.New("not gzip data")
	}

	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(gz); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (ec *EnhancedCache) MGet(ctx context.Context, keys []string, opts *GetOptions) (map[string][]byte, error) {
	if !ec.config.Enabled {
		return nil, ErrCacheDisabled
	}

	if opts == nil {
		opts = &GetOptions{Level: CacheLevelBoth}
	}

	result := make(map[string][]byte)
	missingKeys := make([]string, 0)

	if ec.config.L1Enabled && !opts.SkipLocal {
		for _, key := range keys {
			if val, err := ec.getFromL1(key); err == nil {
				result[key] = val
			} else {
				missingKeys = append(missingKeys, key)
			}
		}
	} else {
		missingKeys = keys
	}

	if len(missingKeys) > 0 && ec.config.L2Enabled && !opts.SkipRemote {
		if Client != nil {
			vals, err := Client.MGet(ctx, missingKeys...).Result()
			if err == nil {
				for i, key := range missingKeys {
					if vals[i] != nil {
						valStr, ok := vals[i].(string)
						if ok {
							val := []byte(valStr)
							result[key] = val
							if ec.config.L1Enabled {
								ec.setToL1(key, val, ec.config.L1TTL, 0)
							}
						}
					}
				}
			}
		}
	}

	return result, nil
}

func (ec *EnhancedCache) MSet(ctx context.Context, items map[string][]byte, opts *SetOptions) error {
	if !ec.config.Enabled {
		return ErrCacheDisabled
	}

	if opts == nil {
		opts = &SetOptions{Level: CacheLevelBoth}
	}

	for key, value := range items {
		if err := ec.Set(ctx, key, value, opts); err != nil {
			return err
		}
	}

	return nil
}

func (ec *EnhancedCache) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if Client == nil {
		return false, nil
	}

	lockKey := fmt.Sprintf("lock:%s", key)
	return Client.SetNX(ctx, lockKey, "1", ttl).Result()
}

func (ec *EnhancedCache) Unlock(ctx context.Context, key string) error {
	if Client == nil {
		return nil
	}

	lockKey := fmt.Sprintf("lock:%s", key)
	return Client.Del(ctx, lockKey).Err()
}

func (ec *EnhancedCache) ExtendLock(ctx context.Context, key string, ttl time.Duration) error {
	if Client == nil {
		return nil
	}

	lockKey := fmt.Sprintf("lock:%s", key)
	return Client.Expire(ctx, lockKey, ttl).Err()
}

func (ec *EnhancedCache) AcquireLock(ctx context.Context, key string, ttl time.Duration, retry int) (bool, error) {
	for i := 0; i < retry; i++ {
		ok, err := ec.Lock(ctx, key, ttl)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false, ErrLockTimeout
}

type MultiLevelCacheConfig struct {
	L1MaxMemory      int64
	L1MaxItems       int
	L1EvictionPolicy string
	L2MaxMemory      int64
	L2EvictionPolicy string
	EnableTiered     bool
	TierThreshold    int64
	PromoteOnHit     bool
	DemoteOnMiss     bool
}

var DefaultMultiLevelConfig = &MultiLevelCacheConfig{
	L1MaxMemory:      100 * 1024 * 1024,
	L1MaxItems:       10000,
	L1EvictionPolicy: "lru",
	L2MaxMemory:      1024 * 1024 * 1024,
	L2EvictionPolicy: "lru",
	EnableTiered:     true,
	TierThreshold:    100,
	PromoteOnHit:     true,
	DemoteOnMiss:     false,
}

type TieredCache struct {
	config   *MultiLevelCacheConfig
	l1Cache  *EnhancedCache
	l2Cache  *EnhancedCache
	l1Stats  *TierStats
	l2Stats  *TierStats
	mu       sync.RWMutex
}

type TierStats struct {
	Hits              atomic.Int64
	Misses            atomic.Int64
	Promotions        atomic.Int64
	Demotions         atomic.Int64
	CurrentMemory     atomic.Int64
	CurrentItems      atomic.Int64
	Evictions         atomic.Int64
	LastPromotionTime atomic.Value
	LastDemotionTime  atomic.Value
}

func NewTieredCache(config *MultiLevelCacheConfig) *TieredCache {
	if config == nil {
		config = DefaultMultiLevelConfig
	}

	return &TieredCache{
		config:  config,
		l1Cache: NewEnhancedCache(&CacheConfig{L1Enabled: true, L2Enabled: false}),
		l2Cache: NewEnhancedCache(&CacheConfig{L1Enabled: false, L2Enabled: true}),
		l1Stats: &TierStats{},
		l2Stats: &TierStats{},
	}
}

func (tc *TieredCache) Get(ctx context.Context, key string) ([]byte, error) {
	if val, err := tc.l1Cache.Get(ctx, key, nil); err == nil {
		tc.l1Stats.Hits.Add(1)
		if tc.config.PromoteOnHit && tc.config.EnableTiered {
			tc.promoteToL1(ctx, key, val)
		}
		return val, nil
	}

	tc.l1Stats.Misses.Add(1)

	if val, err := tc.l2Cache.Get(ctx, key, nil); err == nil {
		tc.l2Stats.Hits.Add(1)
		if tc.config.EnableTiered {
			tc.promoteToL1(ctx, key, val)
		}
		return val, nil
	}

	tc.l2Stats.Misses.Add(1)
	return nil, ErrCacheMiss
}

func (tc *TieredCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := tc.l1Cache.Set(ctx, key, value, &SetOptions{TTL: ttl, Level: CacheLevelL1}); err != nil {
		return err
	}

	tc.l1Stats.CurrentMemory.Add(int64(len(value)))
	tc.l1Stats.CurrentItems.Add(1)

	if tc.config.EnableTiered {
		if err := tc.l2Cache.Set(ctx, key, value, &SetOptions{TTL: ttl, Level: CacheLevelL2}); err != nil {
			return err
		}
		tc.l2Stats.CurrentMemory.Add(int64(len(value)))
	}

	if tc.l1Stats.CurrentItems.Load() > int64(tc.config.L1MaxItems) {
		tc.evictFromL1()
	}

	return nil
}

func (tc *TieredCache) Delete(ctx context.Context, key string) error {
	tc.l1Cache.Delete(ctx, key, &DeleteOptions{Level: CacheLevelL1})
	tc.l2Cache.Delete(ctx, key, &DeleteOptions{Level: CacheLevelL2})
	return nil
}

func (tc *TieredCache) promoteToL1(ctx context.Context, key string, value []byte) {
	tc.l1Stats.Promotions.Add(1)
	tc.l1Stats.LastPromotionTime.Store(time.Now())

	if tc.l1Stats.CurrentItems.Load() >= int64(tc.config.L1MaxItems) {
		tc.evictFromL1()
	}

	tc.l1Cache.Set(ctx, key, value, &SetOptions{Level: CacheLevelL1})
	tc.l1Stats.CurrentItems.Add(1)
	tc.l1Stats.CurrentMemory.Add(int64(len(value)))
}

func (tc *TieredCache) evictFromL1() {
	tc.l1Stats.Evictions.Add(1)
}

func (tc *TieredCache) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"l1_hits":        tc.l1Stats.Hits.Load(),
		"l1_misses":      tc.l1Stats.Misses.Load(),
		"l1_promotions":  tc.l1Stats.Promotions.Load(),
		"l1_items":       tc.l1Stats.CurrentItems.Load(),
		"l1_memory":       tc.l1Stats.CurrentMemory.Load(),
		"l2_hits":        tc.l2Stats.Hits.Load(),
		"l2_misses":      tc.l2Stats.Misses.Load(),
		"l2_demotions":   tc.l2Stats.Demotions.Load(),
		"l2_items":       tc.l2Stats.CurrentItems.Load(),
		"l2_memory":      tc.l2Stats.CurrentMemory.Load(),
		"l1_hit_rate":    tc.calculateHitRate(tc.l1Stats),
		"l2_hit_rate":    tc.calculateHitRate(tc.l2Stats),
	}
}

func (tc *TieredCache) calculateHitRate(stats *TierStats) float64 {
	hits := stats.Hits.Load()
	misses := stats.Misses.Load()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

var globalTieredCache *TieredCache
var globalTieredCacheOnce sync.Once

func InitTieredCache(config *MultiLevelCacheConfig) {
	globalTieredCacheOnce.Do(func() {
		globalTieredCache = NewTieredCache(config)
	})
}

func GetTieredCache() *TieredCache {
	if globalTieredCache == nil {
		InitTieredCache(nil)
	}
	return globalTieredCache
}

var globalEnhancedCache *EnhancedCache
var globalEnhancedCacheOnce sync.Once

func InitEnhancedCache(config *CacheConfig) {
	globalEnhancedCacheOnce.Do(func() {
		globalEnhancedCache = NewEnhancedCache(config)
	})
}

func GetEnhancedCache() *EnhancedCache {
	if globalEnhancedCache == nil {
		InitEnhancedCache(nil)
	}
	return globalEnhancedCache
}
