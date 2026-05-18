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
	ErrMultiLevelCacheNotInitialized = errors.New("multi-level cache not initialized")
)

type MultiLevelCacheService struct {
	cache              *redis.EnhancedCache
	l3Cache            *redis.L3Cache
	ttlEngine          *redis.SmartTTLEngine
	consistencyManager *redis.DistributedCacheConsistency
	defaultTTL         time.Duration
}

type MultiLevelCacheOption func(*MultiLevelCacheService)

func WithMultiLevelDefaultTTL(ttl time.Duration) MultiLevelCacheOption {
	return func(mcs *MultiLevelCacheService) {
		mcs.defaultTTL = ttl
	}
}

func WithL3Cache(dsn string) MultiLevelCacheOption {
	return func(mcs *MultiLevelCacheService) {
		if dsn != "" {
			redis.InitL3Cache(&redis.L3CacheConfig{
				Enabled:      true,
				DSN:          dsn,
				TTL:          24 * time.Hour,
				MaxOpenConns: 10,
				MaxIdleConns: 5,
			})
			mcs.l3Cache = redis.GetL3Cache()
			mcs.l3Cache.Initialize(context.Background())
		}
	}
}

func NewMultiLevelCacheService(opts ...MultiLevelCacheOption) *MultiLevelCacheService {
	mcs := &MultiLevelCacheService{
		cache:              redis.GetEnhancedCache(),
		ttlEngine:          redis.GetSmartTTLEngine(),
		consistencyManager: redis.GetDistributedCacheConsistency(),
		defaultTTL:         5 * time.Minute,
	}

	for _, opt := range opts {
		opt(mcs)
	}

	return mcs
}

func (mcs *MultiLevelCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	if mcs.cache == nil {
		return nil, ErrMultiLevelCacheNotInitialized
	}

	mcs.ttlEngine.RecordAccess(key)
	return mcs.cache.Get(ctx, key, nil)
}

func (mcs *MultiLevelCacheService) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := mcs.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (mcs *MultiLevelCacheService) GetString(ctx context.Context, key string) (string, error) {
	data, err := mcs.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (mcs *MultiLevelCacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if mcs.cache == nil {
		return ErrMultiLevelCacheNotInitialized
	}

	var data []byte
	var err error

	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		data, err = json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
	}

	if ttl == 0 {
		ttl = mcs.ttlEngine.CalculateTTL(key)
	}

	mcs.ttlEngine.RecordAccess(key)

	return mcs.cache.Set(ctx, key, data, &redis.SetOptions{
		TTL:   ttl,
		Level: redis.CacheLevelAll,
	})
}

func (mcs *MultiLevelCacheService) SetWithConsistency(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if mcs.consistencyManager == nil {
		return mcs.Set(ctx, key, value, ttl)
	}

	var data []byte
	var err error

	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		data, err = json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
	}

	if ttl == 0 {
		ttl = mcs.defaultTTL
	}

	return mcs.consistencyManager.SetWithConsistency(ctx, key, data, ttl)
}

func (mcs *MultiLevelCacheService) Delete(ctx context.Context, key string) error {
	if mcs.cache == nil {
		return ErrMultiLevelCacheNotInitialized
	}

	return mcs.cache.Delete(ctx, key, &redis.DeleteOptions{Level: redis.CacheLevelAll})
}

func (mcs *MultiLevelCacheService) DeleteWithConsistency(ctx context.Context, key string) error {
	if mcs.consistencyManager == nil {
		return mcs.Delete(ctx, key)
	}

	return mcs.consistencyManager.DeleteWithConsistency(ctx, key)
}

func (mcs *MultiLevelCacheService) GetWithConsistency(ctx context.Context, key string) ([]byte, error) {
	if mcs.consistencyManager == nil {
		return mcs.Get(ctx, key)
	}

	return mcs.consistencyManager.GetWithConsistency(ctx, key)
}

func (mcs *MultiLevelCacheService) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	var result interface{}
	err := mcs.GetJSON(ctx, key, &result)
	if err == nil {
		return result, nil
	}

	if errors.Is(err, redis.ErrCacheMiss) {
		data, err := fn()
		if err != nil {
			return nil, err
		}

		if err := mcs.Set(ctx, key, data, ttl); err != nil {
		}

		return data, nil
	}

	return nil, err
}

func (mcs *MultiLevelCacheService) GetOrSetWithConsistency(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	data, err := mcs.GetWithConsistency(ctx, key)
	if err == nil {
		var result interface{}
		if err := json.Unmarshal(data, &result); err == nil {
			return result, nil
		}
		return string(data), nil
	}

	if errors.Is(err, redis.ErrCacheMiss) {
		result, err := fn()
		if err != nil {
			return nil, err
		}

		if err := mcs.SetWithConsistency(ctx, key, result, ttl); err != nil {
		}

		return result, nil
	}

	return nil, err
}

func (mcs *MultiLevelCacheService) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	if mcs.cache == nil {
		return nil, ErrMultiLevelCacheNotInitialized
	}

	for _, key := range keys {
		mcs.ttlEngine.RecordAccess(key)
	}

	return mcs.cache.MGet(ctx, keys, nil)
}

func (mcs *MultiLevelCacheService) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if mcs.cache == nil {
		return ErrMultiLevelCacheNotInitialized
	}

	cacheItems := make(map[string][]byte)
	for key, value := range items {
		var data []byte
		var err error

		switch v := value.(type) {
		case string:
			data = []byte(v)
		case []byte:
			data = v
		default:
			data, err = json.Marshal(value)
			if err != nil {
				continue
			}
		}
		cacheItems[key] = data
		mcs.ttlEngine.RecordAccess(key)
	}

	return mcs.cache.MSet(ctx, cacheItems, &redis.SetOptions{TTL: ttl})
}

func (mcs *MultiLevelCacheService) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if mcs.cache == nil {
		return false, ErrMultiLevelCacheNotInitialized
	}

	return mcs.cache.Lock(ctx, key, ttl)
}

func (mcs *MultiLevelCacheService) ReleaseLock(ctx context.Context, key string) error {
	if mcs.cache == nil {
		return ErrMultiLevelCacheNotInitialized
	}

	return mcs.cache.Unlock(ctx, key)
}

func (mcs *MultiLevelCacheService) GetStats() *redis.CacheStatsSnapshot {
	if mcs.cache == nil {
		return nil
	}

	return mcs.cache.GetStats()
}

func (mcs *MultiLevelCacheService) GetHotKeys() []*redis.HotKeyInfo {
	if mcs.cache == nil {
		return nil
	}

	return mcs.cache.GetHotKeys()
}

func (mcs *MultiLevelCacheService) GetAccessPattern(key string) redis.AccessPattern {
	return mcs.ttlEngine.GetAccessPattern(key)
}

func (mcs *MultiLevelCacheService) GetConsistencyStatus() *redis.CacheConsistencyStatus {
	if mcs.consistencyManager == nil {
		return nil
	}

	return mcs.consistencyManager.GetStatus()
}

func (mcs *MultiLevelCacheService) SetTTLStrategy(strategy redis.TTLStrategyType) {
	mcs.ttlEngine.SetStrategy(strategy)
}

func (mcs *MultiLevelCacheService) GetTTLStrategy() redis.TTLStrategyType {
	return mcs.ttlEngine.GetStrategy()
}

func (mcs *MultiLevelCacheService) ClearCache(ctx context.Context) error {
	if mcs.cache == nil {
		return ErrMultiLevelCacheNotInitialized
	}

	return mcs.cache.Clear(ctx, redis.CacheLevelAll)
}

func (mcs *MultiLevelCacheService) WarmupCache(ctx context.Context, tasks []*redis.CacheWarmupTask) error {
	manager := redis.GetCacheWarmupManager()
	for _, task := range tasks {
		manager.AddTask(task)
	}
	return manager.WarmupAll(ctx)
}

func (mcs *MultiLevelCacheService) StartWarmupScheduler() {
	manager := redis.GetCacheWarmupManager()
	manager.Start()
}

func (mcs *MultiLevelCacheService) StopWarmupScheduler() {
	manager := redis.GetCacheWarmupManager()
	manager.Stop()
}

var (
	globalMultiLevelCacheService     *MultiLevelCacheService
	globalMultiLevelCacheServiceOnce sync.Once
)

func GetMultiLevelCacheService() *MultiLevelCacheService {
	globalMultiLevelCacheServiceOnce.Do(func() {
		globalMultiLevelCacheService = NewMultiLevelCacheService()
	})
	return globalMultiLevelCacheService
}

func InitMultiLevelCacheService(opts ...MultiLevelCacheOption) {
	globalMultiLevelCacheServiceOnce.Do(func() {
		globalMultiLevelCacheService = NewMultiLevelCacheService(opts...)
	})
}
