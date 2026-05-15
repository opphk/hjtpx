package cache

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"captchax/pkg/cache"
)

type MockRedisClient struct {
	data    map[string]interface{}
	dataBytes map[string][]byte
	mu      sync.RWMutex
	ttl     map[string]time.Time
	setNX   map[string]bool
	sets    map[string]map[string]bool
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data:      make(map[string]interface{}),
		dataBytes: make(map[string][]byte),
		ttl:       make(map[string]time.Time),
		setNX:     make(map[string]bool),
		sets:      make(map[string]map[string]bool),
	}
}

func (m *MockRedisClient) GetBytes(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if exp, ok := m.ttl[key]; ok && time.Now().After(exp) {
		return nil, cache.ErrCacheMiss
	}

	if v, ok := m.dataBytes[key]; ok {
		return v, nil
	}
	return nil, cache.ErrCacheMiss
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch v := value.(type) {
	case []byte:
		m.dataBytes[key] = v
	case string:
		m.dataBytes[key] = []byte(v)
	default:
		m.data[key] = value
	}

	if ttl > 0 {
		m.ttl[key] = time.Now().Add(ttl)
	}
	return nil
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range keys {
		delete(m.data, key)
		delete(m.dataBytes, key)
		delete(m.ttl, key)
		delete(m.setNX, key)
	}
	return nil
}

func (m *MockRedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []string
	for k := range m.data {
		result = append(result, k)
	}
	return result, nil
}

func (m *MockRedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var count int64
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			count++
		}
	}
	return count, nil
}

func (m *MockRedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if exp, ok := m.ttl[key]; ok {
		return time.Until(exp), nil
	}
	return 0, nil
}

func (m *MockRedisClient) Close() error {
	return nil
}

func (m *MockRedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.setNX[key]; ok {
		return false, nil
	}
	m.setNX[key] = true
	return true, nil
}

func (m *MockRedisClient) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sets == nil {
		m.sets = make(map[string]map[string]bool)
	}
	if m.sets[key] == nil {
		m.sets[key] = make(map[string]bool)
	}

	var added int64
	for _, member := range members {
		str := fmt.Sprintf("%v", member)
		if !m.sets[key][str] {
			m.sets[key][str] = true
			added++
		}
	}
	return added, nil
}

func (m *MockRedisClient) SRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sets[key] == nil {
		return 0, nil
	}

	var removed int64
	for _, member := range members {
		str := fmt.Sprintf("%v", member)
		if m.sets[key][str] {
			delete(m.sets[key], str)
			removed++
		}
	}
	return removed, nil
}

func (m *MockRedisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.sets[key] == nil {
		return nil, nil
	}

	result := make([]string, 0, len(m.sets[key]))
	for k := range m.sets[key] {
		result = append(result, k)
	}
	return result, nil
}

func (m *MockRedisClient) Incr(ctx context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int64
	if v, ok := m.dataBytes[key]; ok {
		fmt.Sscanf(string(v), "%d", &count)
	} else if v, ok := m.data[key]; ok {
		fmt.Sscanf(fmt.Sprintf("%v", v), "%d", &count)
	}
	count++
	m.dataBytes[key] = []byte(fmt.Sprintf("%d", count))
	return count, nil
}

func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.dataBytes[key]; ok {
		m.ttl[key] = time.Now().Add(expiration)
	} else if _, ok := m.data[key]; ok {
		m.ttl[key] = time.Now().Add(expiration)
	}
	return nil
}

func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if exp, ok := m.ttl[key]; ok && time.Now().After(exp) {
		return "", cache.ErrCacheMiss
	}

	if v, ok := m.dataBytes[key]; ok {
		return string(v), nil
	}
	if v, ok := m.data[key]; ok {
		return fmt.Sprintf("%v", v), nil
	}
	return "", cache.ErrCacheMiss
}

func (m *MockRedisClient) HSet(ctx context.Context, key string, values ...interface{}) error {
	return nil
}

func (m *MockRedisClient) HGet(ctx context.Context, key, field string) (string, error) {
	return "", nil
}

func (m *MockRedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return nil, nil
}

func (m *MockRedisClient) HDel(ctx context.Context, key string, fields ...string) error {
	return nil
}

func (m *MockRedisClient) ExistsByFunc(ctx context.Context, keys ...string) (int64, error) {
	return m.Exists(ctx, keys...)
}

type simpleLocalCache struct {
	mu    sync.RWMutex
	items map[string]*simpleCacheItem
	maxSize int
	defaultTTL time.Duration
}

type simpleCacheItem struct {
	Value     []byte
	ExpiresAt time.Time
}

func NewLocalCacheSimple(maxSize int, defaultTTL time.Duration) *simpleLocalCache {
	return &simpleLocalCache{
		items:      make(map[string]*simpleCacheItem),
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
	}
}

func (lc *simpleLocalCache) Get(key string) ([]byte, bool) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	item, ok := lc.items[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(item.ExpiresAt) {
		return nil, false
	}
	return item.Value, true
}

func (lc *simpleLocalCache) Set(key string, value []byte) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.items[key] = &simpleCacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(lc.defaultTTL),
	}
	if len(lc.items) > lc.maxSize {
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
}

func (lc *simpleLocalCache) Delete(key string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	delete(lc.items, key)
}

type simpleHashRing struct {
	mu       sync.RWMutex
	nodes    []string
	nodeMap  map[string]int
	positions []uint32
	count    int
}

func NewHashRingCache(nodes []string) *simpleHashRing {
	if len(nodes) == 0 {
		return &simpleHashRing{
			nodes:    []string{},
			nodeMap:  make(map[string]int),
			positions: []uint32{},
			count:    0,
		}
	}
	positions := make([]uint32, len(nodes)*100)
	for i := 0; i < len(nodes)*100; i++ {
		nodeIdx := i / 100
		if nodeIdx < len(nodes) {
			positions[i] = simpleHashCRC32(fmt.Sprintf("%s-%d", nodes[nodeIdx], i))
		}
	}
	return &simpleHashRing{
		nodes:    nodes,
		nodeMap:  make(map[string]int),
		positions: positions,
		count:    len(nodes),
	}
}

func (hr *simpleHashRing) GetNode(key string) string {
	if hr.count == 0 {
		return ""
	}
	hash := simpleHashCRC32(key)
	idx := sort.Search(len(hr.positions), func(i int) bool {
		return hr.positions[i] >= hash
	})
	if idx >= len(hr.positions) {
		idx = 0
	}
	nodeIdx := idx / 100
	if nodeIdx >= hr.count {
		nodeIdx = hr.count - 1
	}
	return hr.nodes[nodeIdx]
}

func (hr *simpleHashRing) AddNode(node string) {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	for _, n := range hr.nodes {
		if n == node {
			return
		}
	}
	hr.nodes = append(hr.nodes, node)
	newPositions := make([]uint32, 100)
	for i := 0; i < 100; i++ {
		newPositions[i] = simpleHashCRC32(fmt.Sprintf("%s-%d", node, i))
	}
	hr.positions = append(hr.positions, newPositions...)
	hr.nodeMap[node] = len(hr.nodes) - 1
	hr.count = len(hr.nodes)
}

func (hr *simpleHashRing) RemoveNode(node string) {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	nodeIdx := -1
	for n, idx := range hr.nodeMap {
		if n == node {
			nodeIdx = idx
			break
		}
	}

	if nodeIdx == -1 {
		return
	}

	hr.nodes = append(hr.nodes[:nodeIdx], hr.nodes[nodeIdx+1:]...)

	newPos := make([]uint32, 0, len(hr.positions)-100)
	for i := 0; i < len(hr.positions); i++ {
		pi := i / 100
		if pi != nodeIdx {
			newPos = append(newPos, hr.positions[i])
		}
	}
	hr.positions = newPos

	hr.count = len(hr.nodes)
	hr.nodeMap = make(map[string]int)
	for i, n := range hr.nodes {
		hr.nodeMap[n] = i
	}
}

func simpleHashCRC32(key string) uint32 {
	hash := uint32(5381)
	for i := 0; i < len(key); i++ {
		hash = ((hash << 5) + hash) + uint32(key[i])
	}
	return hash
}

var _ = sort.Search

func TestMultiLevelCache_Get(t *testing.T) {
	t.Run("LocalCacheHit", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		cfg := &MultiLevelCacheConfig{
			LocalCacheItems:   100,
			LocalCacheTTL:     5 * time.Minute,
			RemoteCacheTTL:    10 * time.Minute,
			StatsEnabled:      true,
			AsyncWriteEnabled: false,
		}

		cache := NewMultiLevelCache(mockRedis, cfg)
		ctx := context.Background()

		cache.local.Set(ctx, "key1", []byte("value1"), 5*time.Minute)

		value, ok := cache.Get(ctx, "key1")
		if !ok {
			t.Fatal("expected to get value from local cache")
		}
		if string(value) != "value1" {
			t.Errorf("expected 'value1', got '%s'", string(value))
		}
	})

	t.Run("RedisCacheHit", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		cfg := &MultiLevelCacheConfig{
			LocalCacheItems:   100,
			LocalCacheTTL:     5 * time.Minute,
			RemoteCacheTTL:    10 * time.Minute,
			StatsEnabled:      true,
			AsyncWriteEnabled: false,
		}

		cache := NewMultiLevelCache(mockRedis, cfg)
		ctx := context.Background()

		mockRedis.Set(ctx, "captchax:mlc:key2", []byte("redis_value"), 10*time.Minute)

		value, ok := cache.Get(ctx, "key2")
		if !ok {
			t.Fatal("expected to get value from redis cache")
		}
		if string(value) != "redis_value" {
			t.Errorf("expected 'redis_value', got '%s'", string(value))
		}
	})

	t.Run("CacheMiss", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		cfg := &MultiLevelCacheConfig{
			LocalCacheItems:   100,
			LocalCacheTTL:     5 * time.Minute,
			RemoteCacheTTL:    10 * time.Minute,
			StatsEnabled:      true,
			AsyncWriteEnabled: false,
		}

		cache := NewMultiLevelCache(mockRedis, cfg)
		ctx := context.Background()

		_, ok := cache.Get(ctx, "nonexistent")
		if ok {
			t.Error("expected cache miss for nonexistent key")
		}
	})
}

func TestMultiLevelCache_Set(t *testing.T) {
	t.Run("SetAndGet", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		cfg := &MultiLevelCacheConfig{
			LocalCacheItems:   100,
			LocalCacheTTL:     5 * time.Minute,
			RemoteCacheTTL:    10 * time.Minute,
			StatsEnabled:      true,
			AsyncWriteEnabled: false,
		}

		cache := NewMultiLevelCache(mockRedis, cfg)
		ctx := context.Background()

		err := cache.Set(ctx, "test_key", []byte("test_value"), 0)
		if err != nil {
			t.Fatalf("failed to set value: %v", err)
		}

		value, ok := cache.Get(ctx, "test_key")
		if !ok {
			t.Fatal("expected to get value after setting")
		}
		if string(value) != "test_value" {
			t.Errorf("expected 'test_value', got '%s'", string(value))
		}
	})
}

func TestMultiLevelCache_Delete(t *testing.T) {
	t.Run("DeleteFromBothCaches", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		cfg := &MultiLevelCacheConfig{
			LocalCacheItems:   100,
			LocalCacheTTL:     5 * time.Minute,
			RemoteCacheTTL:    10 * time.Minute,
			StatsEnabled:      true,
			AsyncWriteEnabled: false,
		}

		cache := NewMultiLevelCache(mockRedis, cfg)
		ctx := context.Background()

		cache.Set(ctx, "delete_key", []byte("delete_value"), 0)

		_, ok := cache.Get(ctx, "delete_key")
		if !ok {
			t.Fatal("expected value to exist before delete")
		}

		cache.Delete(ctx, "delete_key")

		_, ok = cache.Get(ctx, "delete_key")
		if ok {
			t.Error("expected value to be deleted")
		}
	})
}

func TestMultiLevelCache_Stats(t *testing.T) {
	mockRedis := NewMockRedisClient()
	cfg := &MultiLevelCacheConfig{
		LocalCacheItems:   100,
		LocalCacheTTL:     5 * time.Minute,
		RemoteCacheTTL:    10 * time.Minute,
		StatsEnabled:      true,
		AsyncWriteEnabled: false,
	}

	cache := NewMultiLevelCache(mockRedis, cfg)
	ctx := context.Background()

	cache.Set(ctx, "key1", []byte("value1"), 0)
	cache.Get(ctx, "key1")
	cache.Get(ctx, "nonexistent")

	stats := cache.GetStats()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}
	if stats.HitRate == 0 {
		t.Error("expected non-zero hit rate after hit and miss")
	}
}

func TestTokenBucketLimiter_Allow(t *testing.T) {
	t.Run("AllowWithinLimit", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(10, 5)
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			result, err := limiter.Allow(ctx, "test_user")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.Allowed {
				t.Errorf("request %d should be allowed", i+1)
			}
		}
	})

	t.Run("DenyWhenExhausted", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(10, 3)
		ctx := context.Background()

		for i := 0; i < 3; i++ {
			limiter.Allow(ctx, "test_user")
		}

		result, err := limiter.Allow(ctx, "test_user")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("request should be denied after exhausting tokens")
		}
		if result.Remaining != 0 {
			t.Errorf("expected 0 remaining, got %d", result.Remaining)
		}
	})

	t.Run("RefillTokens", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(100, 1)
		ctx := context.Background()

		limiter.Allow(ctx, "test_user")

		time.Sleep(20 * time.Millisecond)

		result, err := limiter.Allow(ctx, "test_user")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Error("request should be allowed after token refill")
		}
	})
}

func TestSlidingWindowLimiter_Allow(t *testing.T) {
	t.Run("AllowWithinWindow", func(t *testing.T) {
		cfg := &RateLimitConfig{
			Strategy: SlidingWindowStrategy,
			Rate:     60,
			Burst:    5,
			Window:   1 * time.Minute,
		}

		limiter := NewSlidingWindowLimiter(nil, cfg)
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			result, err := limiter.Allow(ctx, "test_user")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.Allowed {
				t.Errorf("request %d should be allowed", i+1)
			}
		}
	})

	t.Run("DenyWhenWindowFull", func(t *testing.T) {
		cfg := &RateLimitConfig{
			Strategy: SlidingWindowStrategy,
			Rate:     60,
			Burst:    2,
			Window:   1 * time.Minute,
		}

		limiter := NewSlidingWindowLimiter(nil, cfg)
		ctx := context.Background()

		limiter.Allow(ctx, "test_user")
		limiter.Allow(ctx, "test_user")

		result, err := limiter.Allow(ctx, "test_user")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("request should be denied when window is full")
		}
	})
}

func TestLeakyBucketLimiter_Allow(t *testing.T) {
	cfg := &RateLimitConfig{
		Strategy: LeakyBucketStrategy,
		Rate:     10,
		Burst:    5,
	}

	limiter := NewLeakyBucketLimiter(nil, cfg)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		result, err := limiter.Allow(ctx, "test_user")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	time.Sleep(100 * time.Millisecond)

	result, _ := limiter.Allow(ctx, "test_user")
	if !result.Allowed {
		t.Error("request should be allowed after leak")
	}
}

func TestDistributedRateLimiter_Allow(t *testing.T) {
	t.Run("AllowWithinLimit", func(t *testing.T) {
		cfg := &RateLimitConfig{
			Strategy:    SlidingWindowStrategy,
			Rate:        60,
			Burst:       10,
			Window:      1 * time.Minute,
			RedisEnabled: false,
		}

		limiter := NewDistributedRateLimiter(nil, cfg)
		ctx := context.Background()

		for i := 0; i < 10; i++ {
			result, err := limiter.Allow(ctx, "test_user")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.Allowed {
				t.Errorf("request %d should be allowed", i+1)
			}
		}
	})

	t.Run("DenyWhenLimitExceeded", func(t *testing.T) {
		cfg := &RateLimitConfig{
			Strategy:    SlidingWindowStrategy,
			Rate:        60,
			Burst:       3,
			Window:      1 * time.Minute,
			RedisEnabled: false,
		}

		limiter := NewDistributedRateLimiter(nil, cfg)
		ctx := context.Background()

		for i := 0; i < 3; i++ {
			limiter.Allow(ctx, "test_user")
		}

		result, err := limiter.Allow(ctx, "test_user")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("request should be denied when limit exceeded")
		}
	})
}

func TestSessionStore_Create(t *testing.T) {
	t.Run("CreateAndGetSession", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		cfg := &SessionConfig{
			TTL:            24 * time.Hour,
			CookieName:     "test_session",
			MaxSessions:    10,
			RefreshEnabled: true,
		}

		store := NewRedisSessionStore(mockRedis, cfg)
		ctx := context.Background()

		data := map[string]interface{}{
			"user_id": "user123",
			"role":    "admin",
		}

		session, err := store.Create(ctx, data)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		if session.ID == "" {
			t.Error("expected non-empty session ID")
		}
		if session.UserID != "user123" {
			t.Errorf("expected user_id 'user123', got '%s'", session.UserID)
		}

		retrieved, exists, err := store.Get(ctx, session.ID)
		if err != nil {
			t.Fatalf("failed to get session: %v", err)
		}
		if !exists {
			t.Error("expected session to exist")
		}
		if retrieved.UserID != "user123" {
			t.Errorf("expected user_id 'user123', got '%s'", retrieved.UserID)
		}
	})
}

func TestSessionStore_Update(t *testing.T) {
	mockRedis := NewMockRedisClient()
	cfg := &SessionConfig{
		TTL:            24 * time.Hour,
		CookieName:     "test_session",
		MaxSessions:    10,
		RefreshEnabled: true,
	}

	store := NewRedisSessionStore(mockRedis, cfg)
	ctx := context.Background()

	data := map[string]interface{}{
		"user_id": "user123",
		"level":   1,
	}

	session, _ := store.Create(ctx, data)

	updateData := map[string]interface{}{
		"level": 2,
		"score": 100,
	}

	err := store.Update(ctx, session.ID, updateData)
	if err != nil {
		t.Fatalf("failed to update session: %v", err)
	}

	updated, _, _ := store.Get(ctx, session.ID)
	if updated.Data["level"] != float64(2) {
		t.Errorf("expected level 2, got %v", updated.Data["level"])
	}
	if updated.Data["score"] != float64(100) {
		t.Errorf("expected score 100, got %v", updated.Data["score"])
	}
}

func TestSessionStore_Delete(t *testing.T) {
	mockRedis := NewMockRedisClient()
	cfg := &SessionConfig{
		TTL:            24 * time.Hour,
		CookieName:     "test_session",
		MaxSessions:    10,
		RefreshEnabled: true,
	}

	store := NewRedisSessionStore(mockRedis, cfg)
	ctx := context.Background()

	data := map[string]interface{}{
		"user_id": "user123",
	}

	session, _ := store.Create(ctx, data)

	_, exists, _ := store.Get(ctx, session.ID)
	if !exists {
		t.Fatal("session should exist before delete")
	}

	err := store.Delete(ctx, session.ID)
	if err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	_, exists, _ = store.Get(ctx, session.ID)
	if exists {
		t.Error("session should not exist after delete")
	}
}

func TestLocalSessionStore(t *testing.T) {
	cfg := &SessionConfig{
		TTL:       30 * time.Minute,
		MaxSessions: 100,
	}

	store := NewLocalSessionStore(cfg)
	ctx := context.Background()

	t.Run("CreateAndGet", func(t *testing.T) {
		data := map[string]interface{}{
			"user_id": "local_user",
		}

		session, err := store.Create(ctx, data)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		retrieved, exists, err := store.Get(ctx, session.ID)
		if err != nil {
			t.Fatalf("failed to get session: %v", err)
		}
		if !exists {
			t.Error("session should exist")
		}
		if retrieved.UserID != "local_user" {
			t.Errorf("expected 'local_user', got '%s'", retrieved.UserID)
		}
	})

	t.Run("Expiration", func(t *testing.T) {
		shortTTL := &SessionConfig{
			TTL:       1 * time.Millisecond,
			MaxSessions: 100,
		}
		store := NewLocalSessionStore(shortTTL)

		data := map[string]interface{}{
			"user_id": "expire_user",
		}

		session, _ := store.Create(ctx, data)

		time.Sleep(10 * time.Millisecond)

		_, exists, _ := store.Get(ctx, session.ID)
		if exists {
			t.Error("session should have expired")
		}
	})
}

func TestTagManager(t *testing.T) {
	tm := NewTagManager()

	t.Run("RegisterAndGetTags", func(t *testing.T) {
		tm.RegisterTag("user_tag", "user:1", "user:2", "user:3")

		keys := tm.GetKeysByTag("user_tag")
		if len(keys) != 3 {
			t.Errorf("expected 3 keys, got %d", len(keys))
		}
	})

	t.Run("GetNonExistentTag", func(t *testing.T) {
		keys := tm.GetKeysByTag("nonexistent")
		if keys != nil {
			t.Error("expected nil for non-existent tag")
		}
	})

	t.Run("RemoveTag", func(t *testing.T) {
		tm.RegisterTag("temp", "temp:1")
		tm.RemoveTag("temp")

		keys := tm.GetKeysByTag("temp")
		if keys != nil {
			t.Error("expected nil after tag removal")
		}
	})
}

func TestCacheInvalidationManager(t *testing.T) {
	mockRedis := NewMockRedisClient()
	cfg := &InvalidationConfig{
		Strategy:            HybridInvalidation,
		TTL:                 5 * time.Minute,
		MaxTTL:              1 * time.Hour,
		CleanupInterval:     1 * time.Minute,
		StaleThresholdRatio: 0.2,
	}

	mgr := NewCacheInvalidationManager(mockRedis, cfg)
	ctx := context.Background()

	t.Run("SetAndGet", func(t *testing.T) {
		err := mgr.Set(ctx, "inv_key1", []byte("value1"), 0, "tag1")
		if err != nil {
			t.Fatalf("failed to set: %v", err)
		}

		value, ok := mgr.Get(ctx, "inv_key1")
		if !ok {
			t.Fatal("expected to get value")
		}
		if string(value) != "value1" {
			t.Errorf("expected 'value1', got '%s'", string(value))
		}
	})

	t.Run("Invalidate", func(t *testing.T) {
		mgr.Set(ctx, "inv_key2", []byte("value2"), 0)

		mgr.Invalidate(ctx, "inv_key2")

		_, ok := mgr.Get(ctx, "inv_key2")
		if ok {
			t.Error("expected value to be invalidated")
		}
	})

	t.Run("InvalidateWithTag", func(t *testing.T) {
		mgr.Set(ctx, "tagged_key1", []byte("val1"), 0, "test_tag")
		mgr.Set(ctx, "tagged_key2", []byte("val2"), 0, "test_tag")

		mgr.InvalidateWithTag(ctx, "test_tag")

		_, ok1 := mgr.Get(ctx, "tagged_key1")
		_, ok2 := mgr.Get(ctx, "tagged_key2")
		if ok1 || ok2 {
			t.Error("expected all tagged keys to be invalidated")
		}
	})

	mgr.Stop()
}

func TestOptimizedImageCache(t *testing.T) {
	t.Run("MemoryLimit", func(t *testing.T) {
		cfg := &MemoryLimitConfig{
			MaxMemoryBytes:    1000,
			MaxItems:          10,
			EvictionBatchSize: 5,
			EvictionPercent:   0.1,
			StatsEnabled:      true,
		}

		cache := NewOptimizedImageCache(nil, cfg)
		ctx := context.Background()

		for i := 0; i < 20; i++ {
			cache.Set(ctx, "key", []byte("large_value_1234567890"), 0)
		}

		stats := cache.GetStats()
		if stats == nil {
			t.Fatal("expected non-nil stats")
		}

		cache.Stop()
	})

	t.Run("LRUEviction", func(t *testing.T) {
		cfg := &MemoryLimitConfig{
			MaxMemoryBytes:    500,
			MaxItems:          5,
			EvictionBatchSize: 2,
			EvictionPercent:   0.2,
			StatsEnabled:      true,
		}

		cache := NewOptimizedImageCache(nil, cfg)
		ctx := context.Background()

		for i := 0; i < 10; i++ {
			cache.Set(ctx, "key", []byte("item"), 0)
		}

		_, ok := cache.Get(ctx, "key")
		if !ok {
			t.Error("most recent item should exist")
		}

		cache.Stop()
	})
}

func TestImageDeduplicator(t *testing.T) {
	dedup := NewImageDeduplicator(5*time.Minute, 100)

	t.Run("CheckAndMark", func(t *testing.T) {
		hash := dedup.ComputeHash([]byte("test_data"))

		first := dedup.CheckAndMark(hash)
		if !first {
			t.Error("first check should return true")
		}

		second := dedup.CheckAndMark(hash)
		if second {
			t.Error("second check should return false")
		}
	})

	t.Run("Remove", func(t *testing.T) {
		hash := dedup.ComputeHash([]byte("removable"))

		dedup.CheckAndMark(hash)
		dedup.Remove(hash)

		third := dedup.CheckAndMark(hash)
		if !third {
			t.Error("should be able to check again after remove")
		}
	})
}

func TestPreheatManager(t *testing.T) {
	mockRedis := NewMockRedisClient()
	cfg := &MultiLevelCacheConfig{
		LocalCacheItems:   100,
		LocalCacheTTL:     5 * time.Minute,
		RemoteCacheTTL:    10 * time.Minute,
		StatsEnabled:      true,
		AsyncWriteEnabled: false,
	}

	cache := NewMultiLevelCache(mockRedis, cfg)

	preheatCfg := &PreheatConfig{
		Enabled:       true,
		Concurrency:   2,
		BatchSize:     5,
		RetryCount:    3,
		RetryInterval: 10 * time.Millisecond,
	}

	loader := func(ctx context.Context, keys []string) (map[string][]byte, error) {
		result := make(map[string][]byte)
		for _, k := range keys {
			result[k] = []byte("preheated_" + k)
		}
		return result, nil
	}

	manager := NewPreheatManager(cache, loader, preheatCfg)
	ctx := context.Background()

	keys := []string{"key1", "key2", "key3", "key4", "key5"}

	err := manager.Preheat(ctx, keys)
	if err != nil {
		t.Fatalf("preheat failed: %v", err)
	}

	for _, key := range keys {
		val, ok := cache.Get(ctx, key)
		if !ok {
			t.Errorf("key %s should be preheated", key)
		}
		if string(val) != "preheated_"+key {
			t.Errorf("unexpected value for key %s", key)
		}
	}

	stats := manager.GetStats()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.LoadedKeys != int32(len(keys)) {
		t.Errorf("expected %d loaded keys, got %d", len(keys), stats.LoadedKeys)
	}

	cache.Stop()
}

func TestPreheatManager_Strategies(t *testing.T) {
	mockRedis := NewMockRedisClient()
	cfg := &MultiLevelCacheConfig{
		LocalCacheItems:   100,
		LocalCacheTTL:     5 * time.Minute,
		RemoteCacheTTL:    10 * time.Minute,
		StatsEnabled:      true,
		AsyncWriteEnabled: false,
	}

	loader := func(ctx context.Context, keys []string) (map[string][]byte, error) {
		result := make(map[string][]byte)
		for _, k := range keys {
			result[k] = []byte("value_" + k)
		}
		return result, nil
	}

	testCases := []struct {
		name     string
		strategy PreheatStrategy
	}{
		{"Parallel", PreheatStrategyParallel},
		{"Serial", PreheatStrategySerial},
		{"Gradual", PreheatStrategyGradual},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache := NewMultiLevelCache(mockRedis, cfg)
			preheatCfg := &PreheatConfig{
				Enabled:         true,
				Concurrency:     2,
				BatchSize:       3,
				RetryCount:      3,
				RetryInterval:   10 * time.Millisecond,
				Strategy:        tc.strategy,
				GradualInterval: 10 * time.Millisecond,
			}

			manager := NewPreheatManager(cache, loader, preheatCfg)
			ctx := context.Background()
			keys := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}

			err := manager.Preheat(ctx, keys)
			if err != nil {
				t.Fatalf("%s preheat failed: %v", tc.name, err)
			}

			for _, key := range keys {
				_, ok := cache.Get(ctx, key)
				if !ok {
					t.Errorf("key %s should be preheated in %s strategy", key, tc.name)
				}
			}

			cache.Stop()
		})
	}
}

func TestPreheatManager_Stop(t *testing.T) {
	mockRedis := NewMockRedisClient()
	cfg := &MultiLevelCacheConfig{
		LocalCacheItems:   100,
		LocalCacheTTL:     5 * time.Minute,
		RemoteCacheTTL:    10 * time.Minute,
		StatsEnabled:      true,
		AsyncWriteEnabled: false,
	}

	cache := NewMultiLevelCache(mockRedis, cfg)

	loader := func(ctx context.Context, keys []string) (map[string][]byte, error) {
		time.Sleep(50 * time.Millisecond)
		result := make(map[string][]byte)
		for _, k := range keys {
			result[k] = []byte("value_" + k)
		}
		return result, nil
	}

	preheatCfg := &PreheatConfig{
		Enabled:       true,
		Concurrency:   1,
		BatchSize:     1,
		RetryCount:    1,
		RetryInterval: 10 * time.Millisecond,
	}

	manager := NewPreheatManager(cache, loader, preheatCfg)
	ctx := context.Background()

	keys := make([]string, 100)
	for i := 0; i < 100; i++ {
		keys[i] = fmt.Sprintf("key%d", i)
	}

	done := make(chan error)
	go func() {
		done <- manager.Preheat(ctx, keys)
	}()

	time.Sleep(10 * time.Millisecond)
	manager.Stop()

	select {
	case err := <-done:
		t.Logf("Preheat stopped with error: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("Preheat should have stopped")
	}

	cache.Stop()
}

func TestPreheatManager_RetryFailed(t *testing.T) {
	mockRedis := NewMockRedisClient()
	cfg := &MultiLevelCacheConfig{
		LocalCacheItems:   100,
		LocalCacheTTL:     5 * time.Minute,
		RemoteCacheTTL:    10 * time.Minute,
		StatsEnabled:      true,
		AsyncWriteEnabled: false,
	}

	cache := NewMultiLevelCache(mockRedis, cfg)

	shouldFail := true
	loader := func(ctx context.Context, keys []string) (map[string][]byte, error) {
		if shouldFail {
			return nil, fmt.Errorf("simulated failure")
		}
		result := make(map[string][]byte)
		for _, k := range keys {
			result[k] = []byte("value_" + k)
		}
		return result, nil
	}

	preheatCfg := &PreheatConfig{
		Enabled:         true,
		Concurrency:     2,
		BatchSize:       5,
		RetryCount:      2,
		RetryInterval:   10 * time.Millisecond,
		MaxRetryInterval: 100 * time.Millisecond,
	}

	manager := NewPreheatManager(cache, loader, preheatCfg)
	ctx := context.Background()

	keys := []string{"k1", "k2", "k3"}
	err := manager.Preheat(ctx, keys)
	if err == nil {
		t.Fatal("expected error from first preheat")
	}

	shouldFail = false
	err = manager.RetryFailedKeys(ctx)
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}

	stats := manager.GetStats()
	if stats == nil {
		t.Fatal("expected stats after retry")
	}

	for _, key := range keys {
		val, ok := cache.Get(ctx, key)
		if !ok {
			t.Errorf("key %s should be loaded after retry", key)
		}
		if string(val) != "value_"+key {
			t.Errorf("unexpected value for key %s", key)
		}
	}

	cache.Stop()
}

func TestCacheWarmingService(t *testing.T) {
	localCache := NewLocalLRUCache(100, 5*time.Minute)
	cache := NewAdvancedCache(nil, localCache)

	service := NewCacheWarmingService(cache)

	warmedData := make(map[string]string)
	service.Register("warmer1", func(ctx context.Context) error {
		warmedData["w1"] = "value1"
		return nil
	})

	service.Register("warmer2", func(ctx context.Context) error {
		warmedData["w2"] = "value2"
		return nil
	})

	ctx := context.Background()
	results, err := service.Warm(ctx)
	if err != nil {
		t.Fatalf("warm failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	if warmedData["w1"] != "value1" {
		t.Error("warmer1 did not run")
	}
	if warmedData["w2"] != "value2" {
		t.Error("warmer2 did not run")
	}

	warmerInfos := service.GetAllWarmerInfo()
	if len(warmerInfos) != 2 {
		t.Errorf("expected 2 warmer infos, got %d", len(warmerInfos))
	}

	service.Unregister("warmer1")
	warmers := service.ListWarmers()
	if len(warmers) != 1 {
		t.Errorf("expected 1 warmer after unregister, got %d", len(warmers))
	}
}

func TestCacheWarmingService_Single(t *testing.T) {
	localCache := NewLocalLRUCache(100, 5*time.Minute)
	cache := NewAdvancedCache(nil, localCache)

	service := NewCacheWarmingService(cache)

	executed := false
	service.Register("test_warmer", func(ctx context.Context) error {
		executed = true
		return nil
	})

	ctx := context.Background()
	result, err := service.WarmSingle(ctx, "test_warmer")
	if err != nil {
		t.Fatalf("warm single failed: %v", err)
	}

	if result.Name != "test_warmer" {
		t.Errorf("expected result name 'test_warmer', got '%s'", result.Name)
	}

	if !executed {
		t.Error("warmer was not executed")
	}

	_, err = service.WarmSingle(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent warmer")
	}
}

func TestCacheWarmingService_WithTimeout(t *testing.T) {
	localCache := NewLocalLRUCache(100, 5*time.Minute)
	cache := NewAdvancedCache(nil, localCache)

	service := NewCacheWarmingService(cache)

	service.Register("slow_warmer", func(ctx context.Context) error {
		time.Sleep(1 * time.Second)
		return nil
	})

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	results, err := service.Warm(ctx)
	t.Logf("Warm completed with %d results, err: %v", len(results), err)
}

func TestLazyCacheLoader(t *testing.T) {
	mockRedis := NewMockRedisClient()
	cfg := &MultiLevelCacheConfig{
		LocalCacheItems:   100,
		LocalCacheTTL:     5 * time.Minute,
		RemoteCacheTTL:    10 * time.Minute,
		StatsEnabled:      true,
		AsyncWriteEnabled: false,
	}

	cache := NewMultiLevelCache(mockRedis, cfg)

	loadCount := 0
	loader := func(ctx context.Context, key string) ([]byte, error) {
		loadCount++
		return []byte("loaded_" + key), nil
	}

	lazyLoader := NewLazyCacheLoader(cache, loader, 5*time.Minute)
	ctx := context.Background()

	val, ok, err := lazyLoader.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("first get failed: %v", err)
	}
	if !ok {
		t.Error("expected ok to be true")
	}
	if string(val) != "loaded_test_key" {
		t.Errorf("unexpected value: %s", string(val))
	}
	if loadCount != 1 {
		t.Errorf("expected load count 1, got %d", loadCount)
	}

	val, ok, err = lazyLoader.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("second get failed: %v", err)
	}
	if !ok {
		t.Error("expected ok to be true on second get")
	}
	if loadCount != 1 {
		t.Errorf("expected load count still 1, got %d", loadCount)
	}

	cache.Stop()
}

func TestLazyCacheLoader_Concurrent(t *testing.T) {
	mockRedis := NewMockRedisClient()
	cfg := &MultiLevelCacheConfig{
		LocalCacheItems:   100,
		LocalCacheTTL:     5 * time.Minute,
		RemoteCacheTTL:    10 * time.Minute,
		StatsEnabled:      true,
		AsyncWriteEnabled: false,
	}

	cache := NewMultiLevelCache(mockRedis, cfg)

	loadCount := 0
	loader := func(ctx context.Context, key string) ([]byte, error) {
		time.Sleep(10 * time.Millisecond)
		loadCount++
		return []byte("value"), nil
	}

	lazyLoader := NewLazyCacheLoader(cache, loader, 5*time.Minute)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lazyLoader.Get(ctx, "concurrent_key")
		}()
	}

	wg.Wait()

	if loadCount > 1 {
		t.Errorf("expected load count <= 1, got %d", loadCount)
	}

	cache.Stop()
}

func TestBackgroundRefresher(t *testing.T) {
	mockRedis := NewMockRedisClient()
	cfg := &MultiLevelCacheConfig{
		LocalCacheItems:   100,
		LocalCacheTTL:     5 * time.Minute,
		RemoteCacheTTL:    10 * time.Minute,
		StatsEnabled:      true,
		AsyncWriteEnabled: false,
	}

	cache := NewMultiLevelCache(mockRedis, cfg)

	refreshCount := 0
	loader := func(ctx context.Context, key string) ([]byte, error) {
		refreshCount++
		return []byte(fmt.Sprintf("refreshed_%d", refreshCount)), nil
	}

	refresher := NewBackgroundRefresher(cache, []string{"key1", "key2"}, loader, 50*time.Millisecond)

	refresher.Start()

	time.Sleep(150 * time.Millisecond)

	refresher.Stop()

	// Wait a bit for goroutine to stop
	time.Sleep(50 * time.Millisecond)

	if refresher.IsRunning() {
		t.Error("refresher should not be running after stop")
	}

	stats := refresher.GetStats()
	if stats.TotalRefreshes < 1 {
		t.Errorf("expected at least 1 refresh, got %d", stats.TotalRefreshes)
	}

	cache.Stop()
}

func TestBackgroundRefresher_AddRemove(t *testing.T) {
	mockRedis := NewMockRedisClient()
	cfg := &MultiLevelCacheConfig{
		LocalCacheItems:   100,
		LocalCacheTTL:     5 * time.Minute,
		RemoteCacheTTL:    10 * time.Minute,
		StatsEnabled:      true,
		AsyncWriteEnabled: false,
	}

	cache := NewMultiLevelCache(mockRedis, cfg)

	loader := func(ctx context.Context, key string) ([]byte, error) {
		return []byte("value"), nil
	}

	refresher := NewBackgroundRefresher(cache, []string{"initial_key"}, loader, 100*time.Millisecond)

	refresher.AddKey("new_key1")
	refresher.AddKey("new_key2")
	refresher.AddKey("new_key1")

	refresher.RemoveKey("initial_key")
	refresher.RemoveKey("nonexistent")

	refresher.Start()
	time.Sleep(50 * time.Millisecond)
	refresher.Stop()

	cache.Stop()
}

func TestLocalCache(t *testing.T) {
	cache := NewLocalCacheSimple(100, 30*time.Second)

	t.Run("SetAndGet", func(t *testing.T) {
		cache.Set("key1", []byte("value1"))

		val, ok := cache.Get("key1")
		if !ok {
			t.Fatal("expected to get value")
		}
		if string(val) != "value1" {
			t.Errorf("expected 'value1', got '%s'", string(val))
		}
	})

	t.Run("Delete", func(t *testing.T) {
		cache.Set("key2", []byte("value2"))
		cache.Delete("key2")

		_, ok := cache.Get("key2")
		if ok {
			t.Error("expected value to be deleted")
		}
	})

	t.Run("Eviction", func(t *testing.T) {
		smallCache := NewLocalCacheSimple(3, 30*time.Second)

		smallCache.Set("k1", []byte("v1"))
		smallCache.Set("k2", []byte("v2"))
		smallCache.Set("k3", []byte("v3"))
		smallCache.Set("k4", []byte("v4"))

		_, ok := smallCache.Get("k1")
		if ok {
			t.Error("k1 should have been evicted")
		}
	})
}

func BenchmarkMultiLevelCache_Get(b *testing.B) {
	mockRedis := NewMockRedisClient()
	cfg := &MultiLevelCacheConfig{
		LocalCacheItems:   1000,
		LocalCacheTTL:     5 * time.Minute,
		RemoteCacheTTL:    10 * time.Minute,
		StatsEnabled:      false,
		AsyncWriteEnabled: false,
	}

	cache := NewMultiLevelCache(mockRedis, cfg)
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		cache.Set(ctx, "key", []byte("value"), 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(ctx, "key")
	}
}

func BenchmarkTokenBucketLimiter_Allow(b *testing.B) {
	limiter := NewTokenBucketLimiter(1000, 10000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, "bench_user")
	}
}

func BenchmarkSlidingWindowLimiter_Allow(b *testing.B) {
	cfg := &RateLimitConfig{
		Strategy: SlidingWindowStrategy,
		Rate:     60,
		Burst:    1000,
		Window:   1 * time.Minute,
	}

	limiter := NewSlidingWindowLimiter(nil, cfg)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, "bench_user")
	}
}
