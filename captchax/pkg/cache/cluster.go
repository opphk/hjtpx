package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type ClusterConfig struct {
	Addrs        []string
	Password     string
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	TLSCert      *tls.Config
}

type RedisCluster struct {
	client *redis.ClusterClient
	pool   *clusterPool
}

type clusterPool struct {
	mu      sync.RWMutex
	nodes   map[string]*nodePool
	config  *ClusterConfig
}

type nodePool struct {
	client   *redis.Client
	addr     string
	mu       sync.Mutex
	conns    int
	maxConns int
}

type ClusterAwareCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	MGet(ctx context.Context, keys ...string) ([][]byte, []error)
	MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error)
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error)
	ScriptLoad(ctx context.Context, script string) (string, error)
	ScriptExists(ctx context.Context, hashes ...string) ([]bool, error)
	Close() error
}

func NewRedisCluster(cfg *ClusterConfig) (*RedisCluster, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cluster config is required")
	}

	if len(cfg.Addrs) == 0 {
		return nil, fmt.Errorf("at least one cluster address is required")
	}

	opt := &redis.ClusterOptions{
		Addrs:        cfg.Addrs,
		Password:     cfg.Password,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		TLSConfig:    cfg.TLSCert,
	}

	client := redis.NewClusterClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis cluster: %w", err)
	}

	rc := &RedisCluster{
		client: client,
		pool: &clusterPool{
			nodes:  make(map[string]*nodePool),
			config: cfg,
		},
	}

	return rc, nil
}

func (rc *RedisCluster) Get(ctx context.Context, key string) ([]byte, error) {
	return rc.client.Get(ctx, key).Bytes()
}

func (rc *RedisCluster) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return rc.client.Set(ctx, key, value, ttl).Err()
}

func (rc *RedisCluster) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return rc.client.Del(ctx, keys...).Err()
}

func (rc *RedisCluster) MGet(ctx context.Context, keys ...string) ([][]byte, []error) {
	if len(keys) == 0 {
		return nil, nil
	}

	cmds, err := rc.client.MGet(ctx, keys...).Result()
	if err != nil {
		errors := make([]error, len(keys))
		for i := range errors {
			errors[i] = err
		}
		return nil, errors
	}

	results := make([][]byte, len(cmds))
	errors := make([]error, len(cmds))

	for i, cmd := range cmds {
		if cmd == nil {
			errors[i] = redis.Nil
			continue
		}

		switch v := cmd.(type) {
		case string:
			results[i] = []byte(v)
		case []byte:
			results[i] = v
		default:
			errors[i] = fmt.Errorf("unexpected type: %T", cmd)
		}
	}

	return results, errors
}

func (rc *RedisCluster) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	pipe := rc.client.Pipeline()
	for key, value := range items {
		pipe.Set(ctx, key, value, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (rc *RedisCluster) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return rc.client.Eval(ctx, script, keys, args...).Result()
}

func (rc *RedisCluster) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error) {
	return rc.client.EvalSha(ctx, sha1, keys, args...).Result()
}

func (rc *RedisCluster) ScriptLoad(ctx context.Context, script string) (string, error) {
	return rc.client.ScriptLoad(ctx, script).Result()
}

func (rc *RedisCluster) ScriptExists(ctx context.Context, hashes ...string) ([]bool, error) {
	return rc.client.ScriptExists(ctx, hashes...).Result()
}

func (rc *RedisCluster) Close() error {
	return rc.client.Close()
}

func (rc *RedisCluster) Client() *redis.ClusterClient {
	return rc.client
}

type ClusterRouter struct {
	cluster *RedisCluster
	local   *LocalCache
	config  *RouterConfig
}

type RouterConfig struct {
	LocalCacheSize int
	LocalCacheTTL   time.Duration
	ReadFromReplica bool
	WriteConsistency ClusterConsistency
}

type ClusterConsistency int

const (
	StrongConsistency ClusterConsistency = iota
	EventualConsistency
)

func NewClusterRouter(cluster *RedisCluster, cfg *RouterConfig) *ClusterRouter {
	if cfg == nil {
		cfg = &RouterConfig{
			LocalCacheSize: 1000,
			LocalCacheTTL:  30 * time.Second,
		}
	}

	return &ClusterRouter{
		cluster: cluster,
		local:  NewLocalCache(cfg.LocalCacheSize, cfg.LocalCacheTTL),
		config: cfg,
	}
}

func (cr *ClusterRouter) Get(ctx context.Context, key string) ([]byte, error) {
	if data, ok := cr.local.Get(key); ok {
		return data, nil
	}

	data, err := cr.cluster.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	cr.local.Set(key, data)
	return data, nil
}

func (cr *ClusterRouter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	cr.local.Set(key, value)
	return cr.cluster.Set(ctx, key, value, ttl)
}

func (cr *ClusterRouter) Delete(ctx context.Context, key string) error {
	cr.local.Delete(key)
	return cr.cluster.Del(ctx, key)
}

func (cr *ClusterRouter) MGet(ctx context.Context, keys ...string) ([][]byte, []error) {
	localResults := make([][]byte, len(keys))
	localHits := make([]bool, len(keys))

	for i, key := range keys {
		if data, ok := cr.local.Get(key); ok {
			localResults[i] = data
			localHits[i] = true
		}
	}

	missedKeys := make([]string, 0)
	missedIndices := make([]int, 0)
	for i, hit := range localHits {
		if !hit {
			missedKeys = append(missedKeys, keys[i])
			missedIndices = append(missedIndices, i)
		}
	}

	clusterResults := make([][]byte, len(keys))
	var clusterErrors []error

	if len(missedKeys) > 0 {
		results, errors := cr.cluster.MGet(ctx, missedKeys...)

		for i, idx := range missedIndices {
			if results != nil && idx < len(results) && results[i] != nil {
				clusterResults[idx] = results[i]
				cr.local.Set(keys[idx], results[i])
			}
			if errors != nil && idx < len(errors) && errors[i] != nil {
				clusterErrors = append(clusterErrors, errors[i])
			}
		}
	}

	for i, hit := range localHits {
		if hit {
			clusterResults[i] = localResults[i]
		}
	}

	return clusterResults, clusterErrors
}

type LocalCache struct {
	mu       sync.RWMutex
	items    map[string]*localCacheItem
	maxSize  int
	defaultTTL time.Duration
}

type localCacheItem struct {
	Value     []byte
	ExpiresAt time.Time
}

func NewLocalCache(maxSize int, defaultTTL time.Duration) *LocalCache {
	return &LocalCache{
		items:      make(map[string]*localCacheItem),
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
	}
}

func (lc *LocalCache) Get(key string) ([]byte, bool) {
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

func (lc *LocalCache) Set(key string, value []byte) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.items[key] = &localCacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(lc.defaultTTL),
	}

	if len(lc.items) > lc.maxSize {
		lc.evictOldest()
	}
}

func (lc *LocalCache) Delete(key string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	delete(lc.items, key)
}

func (lc *LocalCache) evictOldest() {
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

type HashRing struct {
	mu       sync.RWMutex
	nodes    []string
	nodeMap   map[string]int
	positions []uint32
	count     int
}

func NewHashRing(nodes []string) *HashRing {
	positions := make([]uint32, len(nodes)*100)
	for i := 0; i < len(nodes)*100; i++ {
		positions[i] = hashCRC32(fmt.Sprintf("%s-%d", nodes[i], i))
	}

	return &HashRing{
		nodes:    nodes,
		nodeMap:  make(map[string]int),
		positions: positions,
		count:    len(nodes),
	}
}

func (hr *HashRing) GetNode(key string) string {
	if hr.count == 0 {
		return ""
	}

	hash := hashCRC32(key)
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

func (hr *HashRing) AddNode(node string) {
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
		newPositions[i] = hashCRC32(fmt.Sprintf("%s-%d", node, i))
	}
	hr.positions = append(hr.positions, newPositions...)
	hr.nodeMap[node] = len(hr.nodes) - 1
	hr.count = len(hr.nodes)
}

func (hr *HashRing) RemoveNode(node string) {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	nodeIdx, ok := hr.nodeMap[node]
	if !ok {
		return
	}

	newNodes := make([]string, 0, len(hr.nodes)-1)
	newPositions := make([]uint32, 0, len(hr.positions)-100)

	for i, n := range hr.nodes {
		if i != nodeIdx {
			newNodes = append(newNodes, n)
		}
	}

	for i := 0; i < len(hr.positions); i++ {
		posNodeIdx := i / 100
		if posNodeIdx != nodeIdx {
			newPositions = append(newPositions, hr.positions[i])
		}
	}

	hr.nodes = newNodes
	hr.positions = newPositions
	hr.count = len(hr.nodes)

	delete(hr.nodeMap, node)
	for n, idx := range hr.nodeMap {
		if idx > nodeIdx {
			hr.nodeMap[n] = idx - 1
		}
	}
}

func hashCRC32(key string) uint32 {
	hash := uint32(5381)
	for i := 0; i < len(key); i++ {
		hash = ((hash << 5) + hash) + uint32(key[i])
	}
	return hash
}

var _ = strings.Compare
