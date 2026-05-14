package cache

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type RemoteCache interface {
	GetBytes(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Keys(ctx context.Context, pattern string) ([]string, error)
}

type MultiLevelCacheConfig struct {
	LocalCacheItems   int
	LocalCacheTTL     time.Duration
	RemoteCacheTTL    time.Duration
	StatsEnabled      bool
	AsyncWriteEnabled bool
}

type MultiLevelCache struct {
	mu          sync.RWMutex
	local       *localCache
	remote      RemoteCache
	config      *MultiLevelCacheConfig
	metrics     *CacheMetrics
	asyncWriter *asyncWriter
}

type localCache struct {
	mu         sync.RWMutex
	items      map[string]*localItem
	maxItems   int
	defaultTTL time.Duration
}

type localItem struct {
	Value     []byte
	ExpiresAt time.Time
}

type CacheMetrics struct {
	hits       atomic.Int64
	misses     atomic.Int64
	evictions  atomic.Int64
	latencies  atomic.Int64
	writeQueue atomic.Int64
	mu         sync.RWMutex
}

type asyncWriter struct {
	ch   chan writeRequest
	done chan struct{}
}

type writeRequest struct {
	key    string
	value  []byte
	ttl    time.Duration
	opType writeOpType
}

type writeOpType int

const (
	writeOpSet writeOpType = iota
	writeOpDelete
)

func NewMultiLevelCache(remote RemoteCache, cfg *MultiLevelCacheConfig) *MultiLevelCache {
	if cfg == nil {
		cfg = &MultiLevelCacheConfig{
			LocalCacheItems:   5000,
			LocalCacheTTL:     5 * time.Minute,
			RemoteCacheTTL:    30 * time.Minute,
			StatsEnabled:      true,
			AsyncWriteEnabled: true,
		}
	}

	mlc := &MultiLevelCache{
		local: &localCache{
			items:      make(map[string]*localItem),
			maxItems:   cfg.LocalCacheItems,
			defaultTTL: cfg.LocalCacheTTL,
		},
		remote:  remote,
		config: cfg,
		metrics: &CacheMetrics{},
	}

	if cfg.AsyncWriteEnabled {
		mlc.asyncWriter = newAsyncWriter(remote)
	}

	return mlc
}

func newAsyncWriter(remote RemoteCache) *asyncWriter {
	aw := &asyncWriter{
		ch:   make(chan writeRequest, 10000),
		done: make(chan struct{}),
	}
	go aw.process(remote)
	return aw
}

func (aw *asyncWriter) process(remote RemoteCache) {
	for {
		select {
		case req := <-aw.ch:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			switch req.opType {
			case writeOpSet:
				remote.Set(ctx, req.key, req.value, req.ttl)
			case writeOpDelete:
				remote.Del(ctx, req.key)
			}
			cancel()
		case <-aw.done:
			for len(aw.ch) > 0 {
				req := <-aw.ch
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				switch req.opType {
				case writeOpSet:
					remote.Set(ctx, req.key, req.value, req.ttl)
				case writeOpDelete:
					remote.Del(ctx, req.key)
				}
				cancel()
			}
			return
		}
	}
}

func (aw *asyncWriter) stop() {
	close(aw.done)
}

func (m *MultiLevelCache) Get(ctx context.Context, key string) ([]byte, bool) {
	start := time.Now()
	defer func() {
		if m.metrics != nil {
			m.metrics.RecordLatency(time.Since(start))
		}
	}()

	m.local.mu.RLock()
	if item, ok := m.local.items[key]; ok {
		if time.Now().Before(item.ExpiresAt) {
			value := make([]byte, len(item.Value))
			copy(value, item.Value)
			m.local.mu.RUnlock()
			if m.metrics != nil {
				m.metrics.RecordHit()
			}
			return value, true
		}
		delete(m.local.items, key)
	}
	m.local.mu.RUnlock()

	if m.remote != nil {
		data, err := m.remote.GetBytes(ctx, m.prefixKey(key))
		if err == nil {
			m.local.Set(ctx, key, data, 0)
			if m.metrics != nil {
				m.metrics.RecordHit()
			}
			return data, true
		}
	}

	if m.metrics != nil {
		m.metrics.RecordMiss()
	}
	return nil, false
}

func (m *MultiLevelCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	prefixedKey := m.prefixKey(key)

	if ttl == 0 {
		ttl = m.config.RemoteCacheTTL
	}

	m.local.Set(ctx, key, value, ttl)

	if m.config.AsyncWriteEnabled && m.asyncWriter != nil {
		m.asyncWriter.ch <- writeRequest{
			key:    prefixedKey,
			value:  value,
			ttl:    ttl,
			opType: writeOpSet,
		}
		return nil
	}

	if m.remote != nil {
		return m.remote.Set(ctx, prefixedKey, value, ttl)
	}
	return nil
}

func (m *MultiLevelCache) Delete(ctx context.Context, key string) error {
	prefixedKey := m.prefixKey(key)

	m.local.Delete(ctx, key)

	if m.config.AsyncWriteEnabled && m.asyncWriter != nil {
		m.asyncWriter.ch <- writeRequest{
			key:    prefixedKey,
			opType: writeOpDelete,
		}
		return nil
	}

	if m.remote != nil {
		return m.remote.Del(ctx, prefixedKey)
	}
	return nil
}

func (m *MultiLevelCache) Clear(ctx context.Context) error {
	m.local.Clear()

	if m.remote != nil {
		keys, err := m.remote.Keys(ctx, m.prefixKey("*"))
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			return m.remote.Del(ctx, keys...)
		}
	}
	return nil
}

func (m *MultiLevelCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	var allKeys []string

	m.local.mu.RLock()
	for k := range m.local.items {
		if m.matchPattern(k, pattern) {
			allKeys = append(allKeys, k)
		}
	}
	m.local.mu.RUnlock()

	if m.remote != nil {
		prefixedPattern := m.prefixKey(pattern)
		remoteKeys, err := m.remote.Keys(ctx, prefixedPattern)
		if err == nil {
			for _, k := range remoteKeys {
				unprefixed := m.unprefixKey(k)
				if !m.keyExists(unprefixed, allKeys) && m.matchPattern(unprefixed, pattern) {
					allKeys = append(allKeys, unprefixed)
				}
			}
		}
	}

	return allKeys, nil
}

func (m *MultiLevelCache) keyExists(key string, keys []string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

func (m *MultiLevelCache) matchPattern(key, pattern string) bool {
	if pattern == "*" || pattern == "" {
		return true
	}

	pattern = strings.TrimPrefix(pattern, "*")
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(key, prefix)
	}
	return strings.Contains(key, pattern)
}

func (m *MultiLevelCache) prefixKey(key string) string {
	return "captchax:mlc:" + key
}

func (m *MultiLevelCache) unprefixKey(key string) string {
	return strings.TrimPrefix(key, "captchax:mlc:")
}

func (m *MultiLevelCache) Stop() {
	if m.asyncWriter != nil {
		m.asyncWriter.stop()
	}
}

func (m *MultiLevelCache) GetStats() *CacheStats {
	if m.metrics == nil {
		return nil
	}

	hits := m.metrics.hits.Load()
	misses := m.metrics.misses.Load()
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	m.local.mu.RLock()
	itemCount := int64(len(m.local.items))
	m.local.mu.RUnlock()

	return &CacheStats{
		Hits:      hits,
		Misses:    misses,
		Evictions: m.metrics.evictions.Load(),
		ItemCount: itemCount,
		HitRate:   hitRate,
	}
}

func (m *CacheMetrics) RecordHit() {
	m.hits.Add(1)
}

func (m *CacheMetrics) RecordMiss() {
	m.misses.Add(1)
}

func (m *CacheMetrics) RecordLatency(d time.Duration) {
	m.latencies.Add(d.Milliseconds())
}

func (m *CacheMetrics) RecordEviction() {
	m.evictions.Add(1)
}

func (lc *localCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if ttl == 0 {
		ttl = lc.defaultTTL
	}

	expiresAt := time.Now().Add(ttl)

	if existing, ok := lc.items[key]; ok {
		existing.Value = value
		existing.ExpiresAt = expiresAt
		return
	}

	if len(lc.items) >= lc.maxItems {
		lc.evictOldest()
	}

	lc.items[key] = &localItem{
		Value:     value,
		ExpiresAt: expiresAt,
	}
}

func (lc *localCache) Delete(ctx context.Context, key string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	delete(lc.items, key)
}

func (lc *localCache) Clear() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.items = make(map[string]*localItem)
}

func (lc *localCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for k, item := range lc.items {
		if oldestTime.IsZero() || item.ExpiresAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = item.ExpiresAt
		}
	}

	if oldestKey != "" {
		delete(lc.items, oldestKey)
	}
}

func (lc *localCache) cleanup() {
	now := time.Now()
	for k, item := range lc.items {
		if now.After(item.ExpiresAt) {
			delete(lc.items, k)
		}
	}
}
