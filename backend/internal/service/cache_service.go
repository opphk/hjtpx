package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/redis"
	goredis "github.com/redis/go-redis/v9"
)

var (
	ErrCacheMiss       = errors.New("cache miss")
	ErrLockNotAcquired = errors.New("lock not acquired")
)

const (
	CaptchaSessionTTL = 5 * time.Minute
	ConfigCacheTTL    = 10 * time.Minute
	UserSessionTTL    = 24 * time.Hour
	StatsCacheTTL     = 5 * time.Minute
	BlacklistTTL      = 1 * time.Hour
	BehaviorCacheTTL  = 30 * time.Minute
	RateLimitWindow   = 1 * time.Minute
)

type CacheService struct {
	defaultTTL          time.Duration
	luaScripts          map[string]*goredis.Script
	keyManager          *redis.CacheKeyManager
	monitoringCollector *redis.CacheMonitoringCollector
	expirationManager  *redis.CacheExpirationManager
	invalidationManager *redis.CacheInvalidationManager
}

type CacheServiceOption func(*CacheService)

func WithDefaultTTL(ttl time.Duration) CacheServiceOption {
	return func(cs *CacheService) {
		cs.defaultTTL = ttl
	}
}

func NewCacheService(opts ...CacheServiceOption) *CacheService {
	cs := &CacheService{
		defaultTTL:          5 * time.Minute,
		luaScripts:          make(map[string]*goredis.Script),
		keyManager:          redis.NewCacheKeyManager(redis.NamespaceGlobal),
		monitoringCollector: redis.GetCacheMonitoringCollector(),
		expirationManager:  redis.GetExpirationManager(),
		invalidationManager: redis.GetInvalidationManager(),
	}

	for _, opt := range opts {
		opt(cs)
	}

	return cs
}

func (cs *CacheService) Get(ctx context.Context, key string) (string, error) {
	if redis.Client == nil {
		return "", ErrCacheMiss
	}

	start := time.Now()
	val, err := redis.Client.Get(ctx, key).Result()
	cs.monitoringCollector.RecordLatency(time.Since(start))

	if err == goredis.Nil {
		cs.monitoringCollector.RecordMiss()
		return "", ErrCacheMiss
	}

	if err != nil {
		cs.monitoringCollector.RecordError()
		return "", err
	}

	cs.monitoringCollector.RecordHit()
	cs.monitoringCollector.RecordKeyAccess(key)
	return val, nil
}

func (cs *CacheService) GetBytes(ctx context.Context, key string) ([]byte, error) {
	if redis.Client == nil {
		return nil, ErrCacheMiss
	}

	start := time.Now()
	val, err := redis.Client.Get(ctx, key).Bytes()
	cs.monitoringCollector.RecordLatency(time.Since(start))

	if err == goredis.Nil {
		cs.monitoringCollector.RecordMiss()
		return nil, ErrCacheMiss
	}

	if err != nil {
		cs.monitoringCollector.RecordError()
		return nil, err
	}

	cs.monitoringCollector.RecordHit()
	cs.monitoringCollector.RecordKeyAccess(key)
	return val, nil
}

func (cs *CacheService) GetJSON(ctx context.Context, key string, dest interface{}) error {
	if redis.Client == nil {
		return ErrCacheMiss
	}

	start := time.Now()
	val, err := redis.Client.Get(ctx, key).Bytes()
	cs.monitoringCollector.RecordLatency(time.Since(start))

	if err == goredis.Nil {
		cs.monitoringCollector.RecordMiss()
		return ErrCacheMiss
	}

	if err != nil {
		cs.monitoringCollector.RecordError()
		return err
	}

	cs.monitoringCollector.RecordHit()
	cs.monitoringCollector.RecordKeyAccess(key)
	return json.Unmarshal(val, dest)
}

func (cs *CacheService) Set(ctx context.Context, key string, value interface{}) error {
	return cs.SetWithTTL(ctx, key, value, cs.defaultTTL)
}

func (cs *CacheService) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if redis.Client == nil {
		return nil
	}

	adaptiveTTL := cs.expirationManager.CalculateTTL(key, ttl)

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

	err = redis.Client.Set(ctx, key, data, adaptiveTTL).Err()
	if err != nil {
		cs.monitoringCollector.RecordError()
		return err
	}

	cs.monitoringCollector.RecordSet()
	cs.monitoringCollector.RecordKeyAccess(key)
	return nil
}

func (cs *CacheService) Delete(ctx context.Context, keys ...string) error {
	if redis.Client == nil {
		return nil
	}

	if len(keys) == 0 {
		return nil
	}

	err := cs.invalidationManager.Invalidate(ctx, keys[0])
	if err != nil {
		return redis.Client.Del(ctx, keys...).Err()
	}

	cs.monitoringCollector.RecordDelete()
	return nil
}

func (cs *CacheService) Exists(ctx context.Context, key string) (bool, error) {
	if redis.Client == nil {
		return false, nil
	}

	result, err := redis.Client.Exists(ctx, key).Result()
	return result > 0, err
}

func (cs *CacheService) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if redis.Client == nil {
		return nil
	}

	adaptiveTTL := cs.expirationManager.CalculateTTL(key, ttl)
	return redis.Client.Expire(ctx, key, adaptiveTTL).Err()
}

func (cs *CacheService) TTL(ctx context.Context, key string) (time.Duration, error) {
	if redis.Client == nil {
		return 0, ErrCacheMiss
	}

	return redis.Client.TTL(ctx, key).Result()
}

func (cs *CacheService) Increment(ctx context.Context, key string) (int64, error) {
	if redis.Client == nil {
		return 0, nil
	}

	result, err := redis.Client.Incr(ctx, key).Result()
	if err == nil {
		cs.monitoringCollector.RecordKeyAccess(key)
	}
	return result, err
}

func (cs *CacheService) Decrement(ctx context.Context, key string) (int64, error) {
	if redis.Client == nil {
		return 0, nil
	}

	return redis.Client.Decr(ctx, key).Result()
}

func (cs *CacheService) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	if redis.Client == nil {
		return 0, nil
	}

	result, err := redis.Client.IncrBy(ctx, key, value).Result()
	if err == nil {
		cs.monitoringCollector.RecordKeyAccess(key)
	}
	return result, err
}

type DistributedLock struct {
	key      string
	value    string
	ttl      time.Duration
	acquired bool
	client   *goredis.Client
}

type LockOptions struct {
	RetryCount int
	RetryDelay time.Duration
	TTL        time.Duration
}

var defaultLockOptions = &LockOptions{
	RetryCount: 3,
	RetryDelay: 100 * time.Millisecond,
	TTL:        10 * time.Second,
}

func (cs *CacheService) AcquireLock(ctx context.Context, key string, opts *LockOptions) (*DistributedLock, error) {
	if redis.Client == nil {
		return nil, ErrLockNotAcquired
	}

	if opts == nil {
		opts = defaultLockOptions
	}

	lockKey := cs.keyManager.BuildLockKey(key)
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())

	lock := &DistributedLock{
		key:      lockKey,
		value:    lockValue,
		ttl:      opts.TTL,
		acquired: false,
		client:   redis.Client,
	}

	for i := 0; i < opts.RetryCount; i++ {
		ok, err := redis.Client.SetNX(ctx, lockKey, lockValue, opts.TTL).Result()
		if err == nil && ok {
			lock.acquired = true
			return lock, nil
		}

		if i < opts.RetryCount-1 {
			select {
			case <-ctx.Done():
				return lock, ctx.Err()
			case <-time.After(opts.RetryDelay):
			}
		}
	}

	return lock, ErrLockNotAcquired
}

func (l *DistributedLock) Release(ctx context.Context) error {
	if !l.acquired || l.client == nil {
		return nil
	}

	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	_, err := l.client.Eval(ctx, script, []string{l.key}, l.value).Result()
	l.acquired = false
	return err
}

func (l *DistributedLock) Extend(ctx context.Context, ttl time.Duration) error {
	if !l.acquired || l.client == nil {
		return errors.New("lock not acquired")
	}

	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script, []string{l.key}, l.value, ttl.Milliseconds()).Int64()
	if err != nil {
		return err
	}

	if result == 0 {
		l.acquired = false
		return errors.New("lock expired")
	}

	l.ttl = ttl
	return nil
}

func (l *DistributedLock) IsAcquired() bool {
	return l.acquired
}

type CacheStats struct {
	Hits       int64
	Misses     int64
	Keys       int64
	MemoryUsed int64
}

func (cs *CacheService) GetStats(ctx context.Context) (*CacheStats, error) {
	if redis.Client == nil {
		return &CacheStats{}, nil
	}

	enhancedStats := cs.monitoringCollector.GetDetailedMetrics()

	var keys int64
	iter := redis.Client.Scan(ctx, 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		keys++
	}
	if scanErr := iter.Err(); scanErr != nil {
		return nil, scanErr
	}

	return &CacheStats{
		Hits: int64(enhancedStats["hits"].(int64)),
		Misses: int64(enhancedStats["misses"].(int64)),
		Keys: keys,
	}, nil
}

func (cs *CacheService) GetMonitoringStats() map[string]interface{} {
	return cs.monitoringCollector.GetDetailedMetrics()
}

func (cs *CacheService) GetHealthStatus() *redis.CacheHealthStatus {
	return cs.monitoringCollector.GetHealthStatus()
}

func (cs *CacheService) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if redis.Client == nil {
		return nil, nil
	}

	if len(keys) == 0 {
		return make(map[string]string), nil
	}

	results := make(map[string]string)
	values, err := redis.Client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	for i, val := range values {
		if val != nil {
			results[keys[i]] = val.(string)
		}
	}

	return results, nil
}

func (cs *CacheService) SetMultiple(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if redis.Client == nil {
		return nil
	}

	if len(items) == 0 {
		return nil
	}

	pipe := redis.Client.Pipeline()
	for key, value := range items {
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
				continue
			}
		}

		adaptiveTTL := cs.expirationManager.CalculateTTL(key, ttl)
		pipe.Set(ctx, key, data, adaptiveTTL)
	}

	_, err := pipe.Exec(ctx)
	if err == nil {
		cs.monitoringCollector.RecordSet()
	}
	return err
}

func (cs *CacheService) DeleteByPattern(ctx context.Context, pattern string) (int64, error) {
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

type CacheEntry struct {
	Key       string
	Value     interface{}
	TTL       time.Duration
	ExpiresAt time.Time
}

func (cs *CacheService) GetEntry(ctx context.Context, key string) (*CacheEntry, error) {
	if redis.Client == nil {
		return nil, ErrCacheMiss
	}

	pipe := redis.Client.Pipeline()
	getCmd := pipe.Get(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)

	_, err := pipe.Exec(ctx)
	if err != nil && err != goredis.Nil {
		return nil, err
	}

	val, err := getCmd.Result()
	if err == goredis.Nil {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}

	ttl, _ := ttlCmd.Result()

	return &CacheEntry{
		Key:       key,
		Value:     val,
		TTL:       ttl,
		ExpiresAt: time.Now().Add(ttl),
	}, nil
}

type CircuitBreaker struct {
	mu           sync.RWMutex
	failures     int
	lastFailure  time.Time
	threshold    int
	resetTimeout time.Duration
	state        string
}

func NewCircuitBreaker(threshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold:    threshold,
		resetTimeout: resetTimeout,
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

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = "closed"
}

type cachedFunction struct {
	service *CacheService
	breaker *CircuitBreaker
}

func NewCachedFunction() *cachedFunction {
	return &cachedFunction{
		service: NewCacheService(),
		breaker: NewCircuitBreaker(5, 30*time.Second),
	}
}

func (cf *cachedFunction) Execute(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	if cf.breaker.Allow() {
		val, err := cf.service.Get(ctx, key)
		if err == nil {
			return val, nil
		}
	}

	result, err := fn()
	if err != nil {
		cf.breaker.RecordFailure()
		return nil, err
	}

	cf.breaker.RecordSuccess()
	cf.service.Set(ctx, key, result)
	return result, nil
}

func (cs *CacheService) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	if redis.Client == nil {
		return fn()
	}

	val, err := cs.Get(ctx, key)
	if err == nil {
		return val, nil
	}

	result, err := fn()
	if err != nil {
		return nil, err
	}

	cs.SetWithTTL(ctx, key, result, ttl)
	return result, nil
}

func (cs *CacheService) GetJSONOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	if redis.Client == nil {
		return fn()
	}

	var result interface{}
	err := cs.GetJSON(ctx, key, &result)
	if err == nil {
		return result, nil
	}

	data, err := fn()
	if err != nil {
		return nil, err
	}

	cs.SetWithTTL(ctx, key, data, ttl)
	return data, nil
}

type CaptchaCache struct {
	CaptchaID  string    `json:"captcha_id"`
	Answer     string    `json:"answer"`
	ExpiresAt  time.Time `json:"expires_at"`
	Difficulty int       `json:"difficulty"`
}

func (cs *CacheService) SetCaptchaCache(ctx context.Context, captchaID string, data *CaptchaCache) error {
	if redis.Client == nil {
		return nil
	}

	key := cs.keyManager.BuildCaptchaKey(captchaID)
	return cs.SetWithTTL(ctx, key, data, CaptchaSessionTTL)
}

func (cs *CacheService) GetCaptchaCache(ctx context.Context, captchaID string) (*CaptchaCache, error) {
	if redis.Client == nil {
		return nil, ErrCacheMiss
	}

	key := cs.keyManager.BuildCaptchaKey(captchaID)
	var cache CaptchaCache
	if err := cs.GetJSON(ctx, key, &cache); err != nil {
		return nil, err
	}

	if time.Now().After(cache.ExpiresAt) {
		cs.Delete(ctx, key)
		return nil, ErrCacheMiss
	}

	return &cache, nil
}

func (cs *CacheService) DeleteCaptchaCache(ctx context.Context, captchaID string) error {
	if redis.Client == nil {
		return nil
	}

	key := cs.keyManager.BuildCaptchaKey(captchaID)
	return cs.Delete(ctx, key)
}

type BehaviorCache struct {
	UserID     string    `json:"user_id"`
	SessionID  string    `json:"session_id"`
	Trajectory string    `json:"trajectory"`
	Timestamp  time.Time `json:"timestamp"`
}

func (cs *CacheService) SetBehaviorCache(ctx context.Context, sessionID string, data *BehaviorCache) error {
	if redis.Client == nil {
		return nil
	}

	key := cs.keyManager.BuildBehaviorKey(sessionID)
	return cs.SetWithTTL(ctx, key, data, BehaviorCacheTTL)
}

func (cs *CacheService) GetBehaviorCache(ctx context.Context, sessionID string) (*BehaviorCache, error) {
	if redis.Client == nil {
		return nil, ErrCacheMiss
	}

	key := cs.keyManager.BuildBehaviorKey(sessionID)
	var cache BehaviorCache
	if err := cs.GetJSON(ctx, key, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

func (cs *CacheService) DeleteBehaviorCache(ctx context.Context, sessionID string) error {
	if redis.Client == nil {
		return nil
	}

	key := cs.keyManager.BuildBehaviorKey(sessionID)
	return cs.Delete(ctx, key)
}

type SessionCache struct {
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
}

func (cs *CacheService) SetSessionCache(ctx context.Context, token string, data *SessionCache) error {
	if redis.Client == nil {
		return nil
	}

	key := cs.keyManager.BuildSessionKey(token)
	return cs.SetWithTTL(ctx, key, data, UserSessionTTL)
}

func (cs *CacheService) GetSessionCache(ctx context.Context, token string) (*SessionCache, error) {
	if redis.Client == nil {
		return nil, ErrCacheMiss
	}

	key := cs.keyManager.BuildSessionKey(token)
	var cache SessionCache
	if err := cs.GetJSON(ctx, key, &cache); err != nil {
		return nil, err
	}

	if time.Now().After(cache.ExpiresAt) {
		cs.Delete(ctx, key)
		return nil, ErrCacheMiss
	}

	return &cache, nil
}

func (cs *CacheService) DeleteSessionCache(ctx context.Context, token string) error {
	if redis.Client == nil {
		return nil
	}

	key := cs.keyManager.BuildSessionKey(token)
	return cs.Delete(ctx, key)
}

func (cs *CacheService) RefreshSession(ctx context.Context, token string) error {
	if redis.Client == nil {
		return nil
	}

	cache, err := cs.GetSessionCache(ctx, token)
	if err != nil {
		return err
	}

	cache.ExpiresAt = time.Now().Add(24 * time.Hour)
	return cs.SetSessionCache(ctx, token, cache)
}

type RateLimitCache struct {
	Identifier   string    `json:"identifier"`
	RequestCount int       `json:"request_count"`
	WindowStart  time.Time `json:"window_start"`
}

func (cs *CacheService) IncrementRateLimit(ctx context.Context, identifier string, window time.Duration) (int, error) {
	if redis.Client == nil {
		return 0, nil
	}

	key := cs.keyManager.BuildRateLimitKey(identifier, int(window.Seconds()))

	pipe := redis.Client.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}

	count, err := incrCmd.Result()
	if err != nil {
		return 0, err
	}

	ttl, err := ttlCmd.Result()
	if err != nil || ttl == -1 {
		redis.Client.Expire(ctx, key, window)
	}

	return int(count), nil
}

func (cs *CacheService) GetRateLimitCount(ctx context.Context, identifier string, window time.Duration) (int, error) {
	if redis.Client == nil {
		return 0, nil
	}

	key := cs.keyManager.BuildRateLimitKey(identifier, int(window.Seconds()))
	val, err := redis.Client.Get(ctx, key).Int()
	if err == goredis.Nil {
		return 0, nil
	}
	return val, err
}

func (cs *CacheService) ResetRateLimit(ctx context.Context, identifier string) error {
	if redis.Client == nil {
		return nil
	}

	key := cs.keyManager.BuildRateLimitKey(identifier, 0)
	return cs.Delete(ctx, key)
}

type CacheWarmer struct {
	cacheService *redis.CacheWarmupManager
	stopCh       chan struct{}
	running      bool
	mu           sync.Mutex
}

type WarmupTask struct {
	Name     string
	Handler  func() error
	Interval time.Duration
}

func NewCacheWarmer() *CacheWarmer {
	return &CacheWarmer{
		cacheService: redis.GetCacheWarmupManager(),
		stopCh:       make(chan struct{}),
		running:      false,
	}
}

func (cw *CacheWarmer) AddTask(task *redis.CacheWarmupTask) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.cacheService.AddTask(task)
}

func (cw *CacheWarmer) Start(ctx context.Context) {
	cw.mu.Lock()
	if cw.running {
		cw.mu.Unlock()
		return
	}
	cw.running = true
	cw.mu.Unlock()

	cw.cacheService.Start()
}

func (cw *CacheWarmer) Stop() {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if cw.running {
		cw.cacheService.Stop()
		cw.running = false
	}
}

func (cw *CacheWarmer) WarmupAll() error {
	return cw.cacheService.WarmupAll(context.Background())
}

func (cs *CacheService) StartCleanupTask(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				patterns := cs.keyManager.BuildAllPatterns()

				for _, pattern := range patterns {
					cs.CleanupExpiredKeys(ctx, pattern)
				}
			}
		}
	}()
}

func (cs *CacheService) CleanupExpiredKeys(ctx context.Context, pattern string) (int, error) {
	if redis.Client == nil {
		return 0, nil
	}

	var cleaned int64
	iter := redis.Client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		ttl, err := redis.Client.TTL(ctx, iter.Val()).Result()
		if err == nil && ttl <= 0 {
			n, err := redis.Client.Del(ctx, iter.Val()).Result()
			if err == nil {
				cleaned += n
			}
		}
	}

	return int(cleaned), iter.Err()
}

type CacheMetrics struct {
	Hits       int64
	Misses     int64
	Sets       int64
	Deletes    int64
	Expired    int64
	Evicted    int64
	HitRate    float64
	mu         sync.RWMutex
}

var globalMetrics = &CacheMetrics{}

func (cs *CacheService) GetMetrics() *CacheMetrics {
	globalMetrics.mu.RLock()
	defer globalMetrics.mu.RUnlock()
	return &CacheMetrics{
		Hits:    globalMetrics.Hits,
		Misses:  globalMetrics.Misses,
		Sets:    globalMetrics.Sets,
		Deletes: globalMetrics.Deletes,
		Expired: globalMetrics.Expired,
		Evicted: globalMetrics.Evicted,
		HitRate: globalMetrics.HitRate,
	}
}

func (cs *CacheService) RecordHit() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.Hits++
	cs.updateHitRate()
}

func (cs *CacheService) RecordMiss() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.Misses++
	cs.updateHitRate()
}

func (cs *CacheService) RecordSet() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.Sets++
}

func (cs *CacheService) RecordDelete() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.Deletes++
}

func (cs *CacheService) RecordExpired() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.Expired++
}

func (cs *CacheService) RecordEvicted() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.Evicted++
}

func (cs *CacheService) updateHitRate() {
	total := globalMetrics.Hits + globalMetrics.Misses
	if total > 0 {
		globalMetrics.HitRate = float64(globalMetrics.Hits) / float64(total) * 100
	}
}

func ResetMetrics() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.Hits = 0
	globalMetrics.Misses = 0
	globalMetrics.Sets = 0
	globalMetrics.Deletes = 0
	globalMetrics.Expired = 0
	globalMetrics.Evicted = 0
	globalMetrics.HitRate = 0
}

func (cs *CacheService) InvalidateByTag(ctx context.Context, tag string) error {
	return cs.invalidationManager.InvalidateByTag(ctx, tag)
}

func (cs *CacheService) InvalidateByKeyPrefix(ctx context.Context, prefix string) (int, error) {
	pattern := cs.keyManager.BuildKey(redis.CacheKeyPrefix(prefix)) + ":*"
	return cs.invalidationManager.InvalidateByPattern(ctx, pattern)
}

func (cs *CacheService) GetVersion(key string) int64 {
	return cs.expirationManager.GetVersion(key)
}

func (cs *CacheService) IncrementVersion(key string) int64 {
	return cs.expirationManager.IncrementVersion(key)
}
