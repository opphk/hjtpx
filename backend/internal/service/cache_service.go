package service

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

var (
	ErrCacheMiss       = errors.New("cache miss")
	ErrLockNotAcquired = errors.New("lock not acquired")
)

type CacheService struct {
	defaultTTL time.Duration
	luaScripts map[string]*goredis.Script
	enablePipeline bool
	enableCompression bool
	compressionThreshold int
}

type CacheServiceOption func(*CacheService)

func WithDefaultTTL(ttl time.Duration) CacheServiceOption {
	return func(cs *CacheService) {
		cs.defaultTTL = ttl
	}
}

func WithPipeline(enable bool) CacheServiceOption {
	return func(cs *CacheService) {
		cs.enablePipeline = enable
	}
}

func WithCompression(enable bool) CacheServiceOption {
	return func(cs *CacheService) {
		cs.enableCompression = enable
	}
}

func NewCacheService(opts ...CacheServiceOption) *CacheService {
	cfg := config.GetCacheConfig()
	
	cs := &CacheService{
		defaultTTL: 5 * time.Minute,
		luaScripts: make(map[string]*goredis.Script),
		enablePipeline: cfg.EnablePipeline,
		enableCompression: cfg.EnableCompression,
		compressionThreshold: 1024,
	}

	for _, opt := range opts {
		opt(cs)
	}

	cs.registerLuaScripts()

	return cs
}

func (cs *CacheService) registerLuaScripts() {
	cs.luaScripts["decr_if_positive"] = goredis.NewScript(`
		local current = redis.call("get", KEYS[1])
		if current and tonumber(current) > 0 then
			return redis.call("decr", KEYS[1])
		else
			return 0
		end
	`)

	cs.luaScripts["set_nx_with_expiry"] = goredis.NewScript(`
		if redis.call("setnx", KEYS[1], ARGV[1]) == 1 then
			redis.call("expire", KEYS[1], ARGV[2])
			return 1
		else
			return 0
		end
	`)

	cs.luaScripts["zrem_by_score"] = goredis.NewScript(`
		local removed = 0
		local members = redis.call("zrangebyscore", KEYS[1], '-inf', ARGV[1])
		for i, member in ipairs(members) do
			redis.call("zrem", KEYS[1], member)
			removed = removed + 1
		end
		return removed
	`)
}

func (cs *CacheService) Get(ctx context.Context, key string) (string, error) {
	if redis.Client == nil {
		return "", ErrCacheMiss
	}

	val, err := redis.Client.Get(ctx, key).Result()
	if err == goredis.Nil {
		return "", ErrCacheMiss
	}
	return val, err
}

func (cs *CacheService) GetBytes(ctx context.Context, key string) ([]byte, error) {
	if redis.Client == nil {
		return nil, ErrCacheMiss
	}

	val, err := redis.Client.Get(ctx, key).Bytes()
	if err == goredis.Nil {
		return nil, ErrCacheMiss
	}
	return val, err
}

func (cs *CacheService) GetJSON(ctx context.Context, key string, dest interface{}) error {
	if redis.Client == nil {
		return ErrCacheMiss
	}

	val, err := redis.Client.Get(ctx, key).Bytes()
	if err == goredis.Nil {
		return ErrCacheMiss
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(val, dest)
}

func (cs *CacheService) Set(ctx context.Context, key string, value interface{}) error {
	return cs.SetWithTTL(ctx, key, value, cs.defaultTTL)
}

func (cs *CacheService) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if redis.Client == nil {
		return nil
	}

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

	if cs.enableCompression && len(data) > cs.compressionThreshold {
		data, err = cs.compress(data)
		if err != nil {
			return fmt.Errorf("failed to compress value: %w", err)
		}
		key = key + ":gz"
	}

	return redis.Client.Set(ctx, key, data, ttl).Err()
}

func (cs *CacheService) compress(data []byte) ([]byte, error) {
	writer := &compressWriter{data: make([]byte, 0, len(data))}
	gzipWriter := gzip.NewWriter(writer)
	_, err := gzipWriter.Write(data)
	if err != nil {
		return nil, err
	}
	gzipWriter.Close()
	return writer.data, nil
}

func (cs *CacheService) decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(&compressReader{data: data})
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

type compressWriter struct {
	data []byte
}

func (w *compressWriter) Write(p []byte) (int, error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

type compressReader struct {
	data []byte
	pos  int
}

func (r *compressReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (cs *CacheService) Delete(ctx context.Context, keys ...string) error {
	if redis.Client == nil {
		return nil
	}

	if len(keys) == 0 {
		return nil
	}

	return redis.Client.Del(ctx, keys...).Err()
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

	return redis.Client.Expire(ctx, key, ttl).Err()
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

	return redis.Client.Incr(ctx, key).Result()
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

	return redis.Client.IncrBy(ctx, key, value).Result()
}

func (cs *CacheService) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	if redis.Client == nil {
		return false, ErrLockNotAcquired
	}

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
			return false, err
		}
	}

	return redis.Client.SetNX(ctx, key, data, ttl).Result()
}

type DistributedLock struct {
	key        string
	value      string
	ttl        time.Duration
	acquired   bool
	client     *goredis.Client
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

	lockKey := fmt.Sprintf("lock:%s", key)
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
	Hits        int64
	Misses      int64
	Keys        int64
	MemoryUsed  int64
	HitRate     float64
}

func (cs *CacheService) GetStats(ctx context.Context) (*CacheStats, error) {
	stats := &CacheStats{}

	if redis.Client == nil {
		return stats, nil
	}

	iter := redis.Client.Scan(ctx, 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		stats.Keys++
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

func (cs *CacheService) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if redis.Client == nil {
		return nil, nil
	}

	if len(keys) == 0 {
		return make(map[string]string), nil
	}

	if cs.enablePipeline {
		return cs.getMultiplePipeline(ctx, keys)
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

func (cs *CacheService) getMultiplePipeline(ctx context.Context, keys []string) (map[string]string, error) {
	pipe := redis.Client.Pipeline()
	cmds := make([]*goredis.StringCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != goredis.Nil {
		return nil, err
	}

	results := make(map[string]string)
	for i, cmd := range cmds {
		val, err := cmd.Result()
		if err == nil {
			results[keys[i]] = val
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

	if cs.enablePipeline {
		return cs.setMultiplePipeline(ctx, items, ttl)
	}

	for key, value := range items {
		if err := cs.SetWithTTL(ctx, key, value, ttl); err != nil {
			continue
		}
	}

	return nil
}

func (cs *CacheService) setMultiplePipeline(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
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

		pipe.Set(ctx, key, data, ttl)
	}

	_, err := pipe.Exec(ctx)
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

func (cs *CacheService) GetMultipleJSON(ctx context.Context, keys []string, dest interface{}) error {
	if redis.Client == nil {
		return ErrCacheMiss
	}

	data, err := redis.Client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, dest)
}

func (cs *CacheService) SetWithnx(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	if redis.Client == nil {
		return false, nil
	}

	script := cs.luaScripts["set_nx_with_expiry"]
	if script == nil {
		return cs.SetNX(ctx, key, value, ttl)
	}

	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		var err error
		data, err = json.Marshal(v)
		if err != nil {
			return false, err
		}
	}

	result, err := script.Run(ctx, redis.Client, []string{key}, string(data), int(ttl.Seconds())).Int()
	if err != nil {
		return false, err
	}

	return result == 1, nil
}

func (cs *CacheService) IncrementIfPositive(ctx context.Context, key string) (int64, error) {
	if redis.Client == nil {
		return 0, nil
	}

	script := cs.luaScripts["decr_if_positive"]
	if script == nil {
		current, err := redis.Client.Get(ctx, key).Int64()
		if err != nil {
			return 0, err
		}
		if current > 0 {
			return redis.Client.Decr(ctx, key).Result()
		}
		return 0, nil
	}

	return script.Run(ctx, redis.Client, []string{key}).Int64()
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

func (cs *CacheService) MGetJSON(ctx context.Context, keys []string) (map[string]interface{}, error) {
	if redis.Client == nil {
		return nil, ErrCacheMiss
	}

	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}

	pipe := redis.Client.Pipeline()
	cmds := make([]*goredis.StringCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != goredis.Nil {
		return nil, err
	}

	results := make(map[string]interface{})
	for i, cmd := range cmds {
		val, err := cmd.Bytes()
		if err != nil {
			continue
		}

		var data interface{}
		if err := json.Unmarshal(val, &data); err == nil {
			results[keys[i]] = data
		}
	}

	return results, nil
}

func (cs *CacheService) ZAdd(ctx context.Context, key string, members map[string]float64) error {
	if redis.Client == nil {
		return nil
	}

	pipe := redis.Client.Pipeline()
	for member, score := range members {
		pipe.ZAdd(ctx, key, goredis.Z{Score: score, Member: member})
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (cs *CacheService) ZRangeByScore(ctx context.Context, key string, min, max float64, offset, count int64) ([]string, error) {
	if redis.Client == nil {
		return nil, nil
	}

	return redis.Client.ZRangeByScore(ctx, key, &goredis.ZRangeBy{
		Min:   fmt.Sprintf("%f", min),
		Max:   fmt.Sprintf("%f", max),
		Offset: offset,
		Count: count,
	}).Result()
}

func (cs *CacheService) ZRemRangeByScore(ctx context.Context, key string, minScore string) (int64, error) {
	if redis.Client == nil {
		return 0, nil
	}

	script := cs.luaScripts["zrem_by_score"]
	if script == nil {
		members, err := redis.Client.ZRangeByScore(ctx, key, &goredis.ZRangeBy{
			Min: "-inf",
			Max: minScore,
		}).Result()
		if err != nil {
			return 0, err
		}

		var removed int64
		for _, member := range members {
			n, _ := redis.Client.ZRem(ctx, key, member).Result()
			removed += n
		}
		return removed, nil
	}

	return script.Run(ctx, redis.Client, []string{key}, minScore).Int64()
}

func (cs *CacheService) HSet(ctx context.Context, key string, values map[string]interface{}) error {
	if redis.Client == nil {
		return nil
	}

	pipe := redis.Client.Pipeline()
	for field, value := range values {
		pipe.HSet(ctx, key, field, value)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (cs *CacheService) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if redis.Client == nil {
		return nil, nil
	}

	return redis.Client.HGetAll(ctx, key).Result()
}

func (cs *CacheService) HGet(ctx context.Context, key, field string) (string, error) {
	if redis.Client == nil {
		return "", ErrCacheMiss
	}

	return redis.Client.HGet(ctx, key, field).Result()
}

func (cs *CacheService) HDel(ctx context.Context, key string, fields ...string) error {
	if redis.Client == nil {
		return nil
	}

	return redis.Client.HDel(ctx, key, fields...).Err()
}

func (cs *CacheService) IncrByFloat(ctx context.Context, key string, value float64) (float64, error) {
	if redis.Client == nil {
		return 0, nil
	}

	return redis.Client.IncrByFloat(ctx, key, value).Result()
}

func (cs *CacheService) ExpireAt(ctx context.Context, key string, t time.Time) error {
	if redis.Client == nil {
		return nil
	}

	return redis.Client.ExpireAt(ctx, key, t).Err()
}

func (cs *CacheService) Append(ctx context.Context, key, value string) (int64, error) {
	if redis.Client == nil {
		return 0, nil
	}

	return redis.Client.Append(ctx, key, value).Result()
}

func (cs *CacheService) GetSet(ctx context.Context, key string, value interface{}) (string, error) {
	if redis.Client == nil {
		return "", ErrCacheMiss
	}

	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	oldValue, err := redis.Client.Get(ctx, key).Result()
	if err != nil && err.Error() != "redis: nil" {
		return "", err
	}

	pipe := redis.Client.Pipeline()
	pipe.Set(ctx, key, data, cs.defaultTTL)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return "", err
	}

	return oldValue, nil
}

func (cs *CacheService) FlushDB(ctx context.Context) error {
	if redis.Client == nil {
		return nil
	}

	return redis.Client.FlushDB(ctx).Err()
}

func (cs *CacheService) DBSize(ctx context.Context) (int64, error) {
	if redis.Client == nil {
		return 0, nil
	}

	return redis.Client.DBSize(ctx).Result()
}

func (cs *CacheService) Ping(ctx context.Context) error {
	if redis.Client == nil {
		return errors.New("redis client not available")
	}

	return redis.Client.Ping(ctx).Err()
}

func (cs *CacheService) GetKeys(ctx context.Context, pattern string) ([]string, error) {
	if redis.Client == nil {
		return nil, nil
	}

	var keys []string
	iter := redis.Client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	return keys, iter.Err()
}

func (cs *CacheService) GetKeyType(ctx context.Context, key string) (string, error) {
	if redis.Client == nil {
		return "", nil
	}

	return redis.Client.Type(ctx, key).Result()
}

func (cs *CacheService) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	if redis.Client == nil {
		return 0, ErrCacheMiss
	}

	return redis.Client.TTL(ctx, key).Result()
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
