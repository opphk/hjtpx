package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/redis"
)

var (
	ErrEnhancedCacheNotInitialized = errors.New("enhanced cache not initialized")
)

type EnhancedCacheService struct {
	cache      *redis.EnhancedCache
	defaultTTL time.Duration
}

func NewEnhancedCacheService(opts ...CacheServiceOption) *EnhancedCacheService {
	cs := &EnhancedCacheService{
		defaultTTL: 5 * time.Minute,
	}

	for _, opt := range opts {
		opt(&CacheService{defaultTTL: cs.defaultTTL})
	}

	cs.cache = redis.GetEnhancedCache()
	return cs
}

func (ecs *EnhancedCacheService) Get(ctx context.Context, key string) (string, error) {
	data, err := ecs.cache.Get(ctx, key, nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (ecs *EnhancedCacheService) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return ecs.cache.Get(ctx, key, nil)
}

func (ecs *EnhancedCacheService) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := ecs.cache.Get(ctx, key, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (ecs *EnhancedCacheService) Set(ctx context.Context, key string, value interface{}) error {
	return ecs.SetWithTTL(ctx, key, value, ecs.defaultTTL)
}

func (ecs *EnhancedCacheService) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	var data []byte
	var err error

	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		data, err = json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
	}

	return ecs.cache.Set(ctx, key, data, &redis.SetOptions{
		TTL: ttl,
	})
}

func (ecs *EnhancedCacheService) SetWithTags(ctx context.Context, key string, value interface{}, ttl time.Duration, tags []string) error {
	var data []byte
	var err error

	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		data, err = json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
	}

	return ecs.cache.Set(ctx, key, data, &redis.SetOptions{
		TTL:   ttl,
		Tags:  tags,
		Level: redis.CacheLevelBoth,
	})
}

func (ecs *EnhancedCacheService) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		if err := ecs.cache.Delete(ctx, key, &redis.DeleteOptions{
			Level: redis.CacheLevelBoth,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (ecs *EnhancedCacheService) DeleteByTag(ctx context.Context, tag string) error {
	return ecs.cache.Delete(ctx, tag, &redis.DeleteOptions{
		Level: redis.CacheLevelBoth,
		ByTag: true,
	})
}

func (ecs *EnhancedCacheService) Exists(ctx context.Context, key string) (bool, error) {
	_, err := ecs.cache.Get(ctx, key, nil)
	if err == redis.ErrCacheMiss {
		return false, nil
	}
	return err == nil, err
}

func (ecs *EnhancedCacheService) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if redis.Client == nil {
		return nil
	}
	return redis.Client.Expire(ctx, key, ttl).Err()
}

func (ecs *EnhancedCacheService) TTL(ctx context.Context, key string) (time.Duration, error) {
	if redis.Client == nil {
		return 0, ErrCacheMiss
	}
	return redis.Client.TTL(ctx, key).Result()
}

func (ecs *EnhancedCacheService) Increment(ctx context.Context, key string) (int64, error) {
	if redis.Client == nil {
		return 0, nil
	}
	return redis.Client.Incr(ctx, key).Result()
}

func (ecs *EnhancedCacheService) Decrement(ctx context.Context, key string) (int64, error) {
	if redis.Client == nil {
		return 0, nil
	}
	return redis.Client.Decr(ctx, key).Result()
}

func (ecs *EnhancedCacheService) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	if redis.Client == nil {
		return 0, nil
	}
	return redis.Client.IncrBy(ctx, key, value).Result()
}

func (ecs *EnhancedCacheService) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	var result interface{}
	err := ecs.GetJSON(ctx, key, &result)
	if err == nil {
		return result, nil
	}

	if errors.Is(err, redis.ErrCacheMiss) {
		data, err := fn()
		if err != nil {
			return nil, err
		}

		if err := ecs.SetWithTTL(ctx, key, data, ttl); err != nil {
		}

		return data, nil
	}

	return nil, err
}

func (ecs *EnhancedCacheService) GetJSONOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	return ecs.GetOrSet(ctx, key, ttl, fn)
}

func (ecs *EnhancedCacheService) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	results := make(map[string]string)
	data, err := ecs.cache.MGet(ctx, keys, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range data {
		results[k] = string(v)
	}
	return results, nil
}

func (ecs *EnhancedCacheService) SetMultiple(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	cacheItems := make(map[string][]byte)
	for k, v := range items {
		var data []byte
		var err error

		switch val := v.(type) {
		case string:
			data = []byte(val)
		case []byte:
			data = val
		default:
			data, err = json.Marshal(val)
			if err != nil {
				continue
			}
		}
		cacheItems[k] = data
	}

	return ecs.cache.MSet(ctx, cacheItems, &redis.SetOptions{TTL: ttl})
}

func (ecs *EnhancedCacheService) DeleteByPattern(ctx context.Context, pattern string) (int64, error) {
	if redis.Client == nil {
		return 0, nil
	}

	var deleted int64
	iter := redis.Client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		n, err := redis.Client.Del(ctx, iter.Val()).Result()
		if err == nil {
			deleted += n
		}
	}

	return deleted, iter.Err()
}

func (ecs *EnhancedCacheService) GetVersion(ctx context.Context, key string) (int64, error) {
	return ecs.cache.GetVersion(ctx, key)
}

func (ecs *EnhancedCacheService) IncrementVersion(ctx context.Context, key string) (int64, error) {
	return ecs.cache.IncrementVersion(ctx, key)
}

func (ecs *EnhancedCacheService) GetStats() *redis.CacheStatsSnapshot {
	return ecs.cache.GetStats()
}

func (ecs *EnhancedCacheService) GetHotKeys() []*redis.HotKeyInfo {
	return ecs.cache.GetHotKeys()
}

func (ecs *EnhancedCacheService) ClearCache(ctx context.Context, level redis.CacheLevel) error {
	return ecs.cache.Clear(ctx, level)
}

func (ecs *EnhancedCacheService) AcquireLock(ctx context.Context, key string, opts *LockOptions) (*DistributedLock, error) {
	if redis.Client == nil {
		return nil, ErrLockNotAcquired
	}

	if opts == nil {
		opts = defaultLockOptions
	}

	lockKey := fmt.Sprintf("lock:%s", key)
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())

	ok, err := ecs.cache.Lock(ctx, key, opts.TTL)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrLockNotAcquired
	}

	return &DistributedLock{
		key:      lockKey,
		value:    lockValue,
		ttl:      opts.TTL,
		acquired: true,
		client:   redis.Client,
	}, nil
}

func (ecs *EnhancedCacheService) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	return ecs.cache.MGet(ctx, keys, nil)
}

func (ecs *EnhancedCacheService) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	return ecs.cache.MSet(ctx, items, &redis.SetOptions{TTL: ttl})
}

type ProgressiveInvalidator struct {
	cache         *redis.EnhancedCache
	invalidateCh  chan string
	batchSize     int
	flushInterval time.Duration
	mu            struct {
		sync.RWMutex
		pending map[string]struct{}
	}
	ctx    context.Context
	cancel context.CancelFunc
}

func NewProgressiveInvalidator(cache *redis.EnhancedCache, batchSize int, flushInterval time.Duration) *ProgressiveInvalidator {
	if cache == nil {
		cache = redis.GetEnhancedCache()
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())
	pi := &ProgressiveInvalidator{
		cache:         cache,
		invalidateCh:  make(chan string, 1000),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		ctx:           ctx,
		cancel:        cancel,
	}
	pi.mu.pending = make(map[string]struct{})

	go pi.run()
	return pi
}

func (pi *ProgressiveInvalidator) Invalidate(key string) {
	select {
	case pi.invalidateCh <- key:
	default:
	}
}

func (pi *ProgressiveInvalidator) InvalidateMany(keys []string) {
	for _, key := range keys {
		pi.Invalidate(key)
	}
}

func (pi *ProgressiveInvalidator) run() {
	ticker := time.NewTicker(pi.flushInterval)
	defer ticker.Stop()

	batch := make([]string, 0, pi.batchSize)

	for {
		select {
		case <-pi.ctx.Done():
			pi.flush(batch)
			return
		case key := <-pi.invalidateCh:
			pi.mu.Lock()
			if _, exists := pi.mu.pending[key]; !exists {
				pi.mu.pending[key] = struct{}{}
				batch = append(batch, key)
			}
			pi.mu.Unlock()

			if len(batch) >= pi.batchSize {
				pi.flush(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				pi.flush(batch)
				batch = batch[:0]
			}
		}
	}
}

func (pi *ProgressiveInvalidator) flush(keys []string) {
	if len(keys) == 0 {
		return
	}

	pi.mu.Lock()
	for _, key := range keys {
		delete(pi.mu.pending, key)
	}
	pi.mu.Unlock()

	for _, key := range keys {
		pi.cache.Delete(context.Background(), key, &redis.DeleteOptions{
			Level: redis.CacheLevelBoth,
		})
	}
}

func (pi *ProgressiveInvalidator) Stop() {
	pi.cancel()
}

type VersionedCache struct {
	cache      *redis.EnhancedCache
	versionKey string
}

func NewVersionedCache(cache *redis.EnhancedCache, versionKey string) *VersionedCache {
	if cache == nil {
		cache = redis.GetEnhancedCache()
	}
	return &VersionedCache{
		cache:      cache,
		versionKey: versionKey,
	}
}

func (vc *VersionedCache) Get(ctx context.Context, key string) ([]byte, int64, error) {
	version, _ := vc.cache.GetVersion(ctx, vc.versionKey)
	data, err := vc.cache.Get(ctx, key, nil)
	return data, version, err
}

func (vc *VersionedCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	version, _ := vc.cache.IncrementVersion(ctx, vc.versionKey)
	return vc.cache.Set(ctx, key, value, &redis.SetOptions{
		TTL:     ttl,
		Version: version,
	})
}

func (vc *VersionedCache) InvalidateAll(ctx context.Context) error {
	_, err := vc.cache.IncrementVersion(ctx, vc.versionKey)
	return err
}

func (vc *VersionedCache) GetCurrentVersion(ctx context.Context) (int64, error) {
	return vc.cache.GetVersion(ctx, vc.versionKey)
}

type CachePenetrationProtector struct {
	bloomFilter *redis.BloomFilter
	cache       *redis.EnhancedCache
	nullTTL     time.Duration
}

func NewCachePenetrationProtector(cache *redis.EnhancedCache, expectedItems uint64, falsePositiveRate float64, nullTTL time.Duration) *CachePenetrationProtector {
	if cache == nil {
		cache = redis.GetEnhancedCache()
	}
	if nullTTL <= 0 {
		nullTTL = 5 * time.Minute
	}

	return &CachePenetrationProtector{
		bloomFilter: redis.NewBloomFilter(expectedItems, falsePositiveRate),
		cache:       cache,
		nullTTL:     nullTTL,
	}
}

func (cpp *CachePenetrationProtector) AddToBloomFilter(key string) {
	cpp.bloomFilter.Add(key)
}

func (cpp *CachePenetrationProtector) MayExist(key string) bool {
	return cpp.bloomFilter.MayContain(key)
}

func (cpp *CachePenetrationProtector) Get(ctx context.Context, key string) ([]byte, error) {
	if !cpp.bloomFilter.MayContain(key) {
		return nil, redis.ErrCacheMiss
	}

	data, err := cpp.cache.Get(ctx, key, nil)
	if err == redis.ErrCacheMiss {
		nullKey := fmt.Sprintf("null:%s", key)
		_, nullErr := cpp.cache.Get(ctx, nullKey, nil)
		if nullErr == nil {
			return nil, redis.ErrCacheMiss
		}
	}

	return data, err
}

func (cpp *CachePenetrationProtector) SetNull(ctx context.Context, key string) error {
	nullKey := fmt.Sprintf("null:%s", key)
	return cpp.cache.Set(ctx, nullKey, []byte("null"), &redis.SetOptions{
		TTL: cpp.nullTTL,
	})
}

type CacheBreakdownProtector struct {
	cache        *redis.EnhancedCache
	lockTimeout  time.Duration
	singleflight *singleflightGroup
}

type singleflightGroup struct {
	mu sync.Mutex
	m  map[string]*call
}

type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

func NewCacheBreakdownProtector(cache *redis.EnhancedCache, lockTimeout time.Duration) *CacheBreakdownProtector {
	if cache == nil {
		cache = redis.GetEnhancedCache()
	}
	if lockTimeout <= 0 {
		lockTimeout = 10 * time.Second
	}

	return &CacheBreakdownProtector{
		cache:        cache,
		lockTimeout:  lockTimeout,
		singleflight: &singleflightGroup{m: make(map[string]*call)},
	}
}

func (sf *singleflightGroup) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	sf.mu.Lock()
	if c, ok := sf.m[key]; ok {
		sf.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	c := new(call)
	c.wg.Add(1)
	sf.m[key] = c
	sf.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	sf.mu.Lock()
	delete(sf.m, key)
	sf.mu.Unlock()

	return c.val, c.err
}

func (cbp *CacheBreakdownProtector) GetOrLoad(ctx context.Context, key string, ttl time.Duration, loadFn func() ([]byte, error)) ([]byte, error) {
	data, err := cbp.cache.Get(ctx, key, nil)
	if err == nil {
		return data, nil
	}

	if err != redis.ErrCacheMiss {
		return nil, err
	}

	val, err := cbp.singleflight.Do(key, func() (interface{}, error) {
		data, err := cbp.cache.Get(ctx, key, nil)
		if err == nil {
			return data, nil
		}

		loadedData, err := loadFn()
		if err != nil {
			return nil, err
		}

		if err := cbp.cache.Set(ctx, key, loadedData, &redis.SetOptions{TTL: ttl}); err != nil {
		}

		return loadedData, nil
	})

	if err != nil {
		return nil, err
	}

	return val.([]byte), nil
}

type CacheAvalancheProtector struct {
	cache        *redis.EnhancedCache
	baseTTL      time.Duration
	jitterFactor float64
}

func NewCacheAvalancheProtector(cache *redis.EnhancedCache, baseTTL time.Duration, jitterFactor float64) *CacheAvalancheProtector {
	if cache == nil {
		cache = redis.GetEnhancedCache()
	}
	if jitterFactor <= 0 || jitterFactor >= 1 {
		jitterFactor = 0.2
	}

	return &CacheAvalancheProtector{
		cache:        cache,
		baseTTL:      baseTTL,
		jitterFactor: jitterFactor,
	}
}

func (cap *CacheAvalancheProtector) calculateJitteredTTL() time.Duration {
	jitter := time.Duration(float64(cap.baseTTL) * cap.jitterFactor * (0.5 + 0.5))
	return cap.baseTTL + jitter
}

func (cap *CacheAvalancheProtector) Set(ctx context.Context, key string, value []byte) error {
	ttl := cap.calculateJitteredTTL()
	return cap.cache.Set(ctx, key, value, &redis.SetOptions{TTL: ttl})
}

var (
	globalEnhancedCacheService     *EnhancedCacheService
	globalEnhancedCacheServiceOnce sync.Once
)

func GetEnhancedCacheService() *EnhancedCacheService {
	globalEnhancedCacheServiceOnce.Do(func() {
		globalEnhancedCacheService = NewEnhancedCacheService()
	})
	return globalEnhancedCacheService
}
