package redis

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var (
	ErrNoNodesAvailable     = errors.New("no nodes available")
	ErrPrefetchTimeout      = errors.New("prefetch timeout")
	ErrRecoveryFailed       = errors.New("recovery failed")
	ErrInvalidNodeCount     = errors.New("invalid node count for virtual nodes")
)

const (
	DefaultVirtualNodes    = 150
	DefaultRecoveryTimeout = 30 * time.Second
	DefaultPrefetchBatch   = 100
	DefaultEvictionBatch  = 50
)

type EvictionPolicy int

const (
	EvictionPolicyLRU EvictionPolicy = iota
	EvictionPolicyLFU
	EvictionPolicyHybrid
)

type NodeStatus int

const (
	NodeStatusHealthy NodeStatus = iota
	NodeStatusDegraded
	NodeStatusFailed
	NodeStatusRecovering
)

type ConsistentHashRing struct {
	mu           sync.RWMutex
	virtualNodes int
	hashFunc     hash.Hash64
	ring         map[uint64]string
	sortedKeys   []uint64
	nodes        map[string]*NodeInfo
	version      int64
}

type NodeInfo struct {
	Address      string
	Status       NodeStatus
	FailCount    int32
	LastFailTime time.Time
	Latency      int64
	Weight       int
	Replica      int
}

type PrefetchConfig struct {
	Enabled         bool
	BatchSize       int
	Concurrency     int
	LookAheadWindow time.Duration
	PredictionAlgo string
}

type HybridEvictionConfig struct {
	LRUWeight       float64
	LFUWeight       float64
	WindowSize      time.Duration
	DecayFactor     float64
	MaxMemoryPercent float64
}

type FailoverConfig struct {
	Enabled           bool
	MaxRetries        int
	RetryInterval     time.Duration
	HealthCheckPeriod time.Duration
	DegradedThreshold int
	FailureThreshold  int
	RecoveryTimeout   time.Duration
}

type EvictionEntry struct {
	Key           string
	Score         float64
	LRUScore      float64
	LFUScore      float64
	AccessTime    time.Time
	AccessCount   int64
	LastValue     []byte
	TTL           time.Duration
	Size          int
	Frequency     int64
	HitsInWindow  int64
}

type CacheNode interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) (int64, error)
	Ping(ctx context.Context) error
	Addr() string
}

type EnhancedCacheV2 struct {
	mu              sync.RWMutex
	config          *EnhancedCacheV2Config
	consistentHash *ConsistentHashRing
	cluster         *goredis.ClusterClient
	clients         map[string]CacheNode
	evictionHeap    *EvictionHeap
	prefetcher      *Prefetcher
	failoverMgr     *FailoverManager
	hotDataTracker  *HotDataTracker
	metrics         *CacheV2Metrics
	localCache      *LocalCacheLayer
	version         int64
}

type EnhancedCacheV2Config struct {
	VirtualNodes      int
	Prefetch          *PrefetchConfig
	Eviction          *HybridEvictionConfig
	Failover          *FailoverConfig
	LocalCacheEnabled bool
	LocalCacheSize    int
	LocalCacheTTL     time.Duration
	MetricsEnabled    bool
}

type CacheV2Metrics struct {
	Hits               atomic.Int64
	Misses             atomic.Int64
	PrefetchHits       atomic.Int64
	PrefetchMisses     atomic.Int64
	Evictions          atomic.Int64
	EvictionLRU        atomic.Int64
	EvictionLFU        atomic.Int64
	EvictionHybrid     atomic.Int64
	FailoverTriggered  atomic.Int64
	FailoverRecovered  atomic.Int64
	NodeFails          atomic.Int64
	NodeRecoveries     atomic.Int64
	LatencySum         atomic.Int64
	LatencyCount       atomic.Int64
	ActiveNodes        atomic.Int64
	ConsistentHashOps  atomic.Int64
	TotalMemory        atomic.Int64
	UsedMemory         atomic.Int64
}

type LocalCacheLayer struct {
	mu        sync.RWMutex
	cache     *sync.Map
	maxSize   int
	ttl       time.Duration
	hits      atomic.Int64
	misses    atomic.Int64
	evictions atomic.Int64
}

type CacheAccessRecord struct {
	Key       string
	Timestamp time.Time
}

type HotDataTracker struct {
	mu           sync.RWMutex
	accessLog    []CacheAccessRecord
	maxLogSize   int
	windowSize   time.Duration
	hotThreshold int64
}

type Prefetcher struct {
	mu           sync.RWMutex
	patterns     []string
	batchSize    int
	concurrency  int
	active       atomic.Bool
	stopChan     chan struct{}
	prefetched   atomic.Int64
	prefetchHit  atomic.Int64
	prefetchMiss atomic.Int64
}

type FailoverManager struct {
	mu               sync.RWMutex
	nodes            map[string]*FailoverNode
	maxRetries       int
	retryInterval    time.Duration
	healthCheckPeriod time.Duration
	activeFailover   atomic.Bool
}

type FailoverNode struct {
	Name             string
	Addr             string
	Status           NodeStatus
	FailCount        int32
	LastFailTime     time.Time
	HealthCheckCount int32
	RecoveryStart    time.Time
}

type EvictionHeap struct {
	mu    sync.RWMutex
	items []*EvictionEntry
}

func DefaultEnhancedCacheV2Config() *EnhancedCacheV2Config {
	return &EnhancedCacheV2Config{
		VirtualNodes:      DefaultVirtualNodes,
		LocalCacheEnabled: true,
		LocalCacheSize:    5000,
		LocalCacheTTL:     30 * time.Second,
		MetricsEnabled:    true,
		Prefetch: &PrefetchConfig{
			Enabled:         true,
			BatchSize:       DefaultPrefetchBatch,
			Concurrency:     5,
			LookAheadWindow: 5 * time.Minute,
			PredictionAlgo:  "frequency",
		},
		Eviction: &HybridEvictionConfig{
			LRUWeight:       0.3,
			LFUWeight:       0.7,
			WindowSize:      10 * time.Minute,
			DecayFactor:     0.95,
			MaxMemoryPercent: 80,
		},
		Failover: &FailoverConfig{
			Enabled:           true,
			MaxRetries:        3,
			RetryInterval:     5 * time.Second,
			HealthCheckPeriod: 10 * time.Second,
			DegradedThreshold: 3,
			FailureThreshold:  5,
			RecoveryTimeout:   DefaultRecoveryTimeout,
		},
	}
}

func NewEnhancedCacheV2(config *EnhancedCacheV2Config) *EnhancedCacheV2 {
	if config == nil {
		config = DefaultEnhancedCacheV2Config()
	}

	ec := &EnhancedCacheV2{
		config:          config,
		consistentHash: NewConsistentHashRing(config.VirtualNodes),
		clients:         make(map[string]CacheNode),
		evictionHeap:    NewEvictionHeap(),
		prefetcher:      NewPrefetcher(config.Prefetch),
		failoverMgr:     NewFailoverManager(config.Failover),
		hotDataTracker:  NewHotDataTracker(),
		metrics:         &CacheV2Metrics{},
		version:         time.Now().UnixNano(),
	}

	if config.LocalCacheEnabled {
		ec.localCache = NewLocalCacheLayer(config.LocalCacheSize, config.LocalCacheTTL)
	}

	return ec
}

func NewConsistentHashRing(virtualNodes int) *ConsistentHashRing {
	if virtualNodes <= 0 {
		virtualNodes = DefaultVirtualNodes
	}
	return &ConsistentHashRing{
		virtualNodes: virtualNodes,
		hashFunc:     fnv.New64(),
		ring:         make(map[uint64]string),
		sortedKeys:   make([]uint64, 0),
		nodes:        make(map[string]*NodeInfo),
	}
}

func (ch *ConsistentHashRing) AddNode(addr string, weight int) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if weight <= 0 {
		weight = 1
	}

	info := &NodeInfo{
		Address: addr,
		Status:  NodeStatusHealthy,
		Weight:  weight,
		Replica: ch.virtualNodes * weight,
	}
	ch.nodes[addr] = info

	for i := 0; i < info.Replica; i++ {
		key := ch.computeKey(fmt.Sprintf("%s#%d", addr, i))
		ch.ring[key] = addr
		ch.sortedKeys = append(ch.sortedKeys, key)
	}

	sort.Slice(ch.sortedKeys, func(i, j int) bool {
		return ch.sortedKeys[i] < ch.sortedKeys[j]
	})

	atomic.AddInt64(&ch.version, 1)
}

func (ch *ConsistentHashRing) RemoveNode(addr string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if _, ok := ch.nodes[addr]; !ok {
		return
	}

	info := ch.nodes[addr]
	for i := 0; i < info.Replica; i++ {
		key := ch.computeKey(fmt.Sprintf("%s#%d", addr, i))
		delete(ch.ring, key)
	}

	newKeys := make([]uint64, 0, len(ch.sortedKeys))
	for _, k := range ch.sortedKeys {
		if ch.ring[k] != addr {
			newKeys = append(newKeys, k)
		}
	}
	ch.sortedKeys = newKeys

	delete(ch.nodes, addr)
	atomic.AddInt64(&ch.version, 1)
}

func (ch *ConsistentHashRing) GetNode(key string) (string, error) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if len(ch.ring) == 0 {
		return "", ErrNoNodesAvailable
	}

	hashVal := ch.computeKey(key)
	idx := sort.Search(len(ch.sortedKeys), func(i int) bool {
		return ch.sortedKeys[i] >= hashVal
	})

	if idx >= len(ch.sortedKeys) {
		idx = 0
	}

	nodeAddr := ch.ring[ch.sortedKeys[idx]]
	if node, ok := ch.nodes[nodeAddr]; ok && node.Status != NodeStatusFailed {
		return nodeAddr, nil
	}

	for i := idx + 1; i < len(ch.sortedKeys)+idx; i++ {
		actualIdx := i % len(ch.sortedKeys)
		nodeAddr = ch.ring[ch.sortedKeys[actualIdx]]
		if node, ok := ch.nodes[nodeAddr]; ok && node.Status != NodeStatusFailed {
			return nodeAddr, nil
		}
	}

	return "", ErrNoNodesAvailable
}

func (ch *ConsistentHashRing) computeKey(key string) uint64 {
	h := sha256.New()
	h.Write([]byte(key))
	sum := h.Sum(nil)
	return binary.BigEndian.Uint64(sum[:8])
}

func (ch *ConsistentHashRing) GetAllNodes() []string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	nodes := make([]string, 0, len(ch.nodes))
	for addr := range ch.nodes {
		nodes = append(nodes, addr)
	}
	return nodes
}

func (ch *ConsistentHashRing) UpdateNodeStatus(addr string, status NodeStatus) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if node, ok := ch.nodes[addr]; ok {
		node.Status = status
		if status == NodeStatusFailed {
			atomic.AddInt32(&node.FailCount, 1)
			node.LastFailTime = time.Now()
		} else if status == NodeStatusHealthy {
			atomic.StoreInt32(&node.FailCount, 0)
		}
	}
}

func ConsistentHashDistribution(key string, nodes []string) string {
	if len(nodes) == 0 {
		return ""
	}

	ring := NewConsistentHashRing(DefaultVirtualNodes)
	for _, node := range nodes {
		ring.AddNode(node, 1)
	}

	node, err := ring.GetNode(key)
	if err != nil {
		return nodes[0]
	}
	return node
}

func NewPrefetcher(config *PrefetchConfig) *Prefetcher {
	if config == nil {
		config = &PrefetchConfig{
			Enabled:         true,
			BatchSize:       DefaultPrefetchBatch,
			Concurrency:     5,
			LookAheadWindow: 5 * time.Minute,
			PredictionAlgo:  "frequency",
		}
	}

	return &Prefetcher{
		patterns:    make([]string, 0),
		batchSize:  config.BatchSize,
		concurrency: config.Concurrency,
		stopChan:   make(chan struct{}),
	}
}

func (p *Prefetcher) AddPattern(pattern string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.patterns = append(p.patterns, pattern)
}

func (p *Prefetcher) Start(ctx context.Context, getFunc func(ctx context.Context, key string) (string, error)) {
	if !p.active.CompareAndSwap(false, true) {
		return
	}

	go p.prefetchLoop(ctx, getFunc)
}

func (p *Prefetcher) Stop() {
	if p.active.CompareAndSwap(true, false) {
		close(p.stopChan)
	}
}

func (p *Prefetcher) prefetchLoop(ctx context.Context, getFunc func(ctx context.Context, key string) (string, error)) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.performPrefetch(ctx, getFunc)
		}
	}
}

func (p *Prefetcher) performPrefetch(ctx context.Context, getFunc func(ctx context.Context, key string) (string, error)) {
	p.mu.Lock()
	patterns := make([]string, len(p.patterns))
	copy(patterns, p.patterns)
	p.mu.Unlock()

	if len(patterns) == 0 {
		return
	}

	semaphore := make(chan struct{}, p.concurrency)
	var wg sync.WaitGroup

	for _, pattern := range patterns {
		keys := p.generateKeysFromPattern(pattern)
		batches := p.splitIntoBatches(keys, p.batchSize)

		for _, batch := range batches {
			wg.Add(1)
			go func(keys []string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				for _, key := range keys {
					select {
					case <-ctx.Done():
						return
					default:
						if _, err := getFunc(ctx, key); err == nil {
							p.prefetchHit.Add(1)
						} else {
							p.prefetchMiss.Add(1)
						}
						p.prefetched.Add(1)
					}
				}
			}(batch)
		}
	}

	wg.Wait()
}

func (p *Prefetcher) generateKeysFromPattern(pattern string) []string {
	keys := make([]string, 0, 100)
	baseKey := pattern

	baseKey = pattern
	if len(baseKey) > 0 && baseKey[len(baseKey)-1] == '*' {
		baseKey = baseKey[:len(baseKey)-1]
	}

	for i := 0; i < 50; i++ {
		keys = append(keys, fmt.Sprintf("%s%d", baseKey, i))
	}

	return keys
}

func (p *Prefetcher) splitIntoBatches(keys []string, batchSize int) [][]string {
	if batchSize <= 0 {
		batchSize = DefaultPrefetchBatch
	}

	batches := make([][]string, 0)
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batches = append(batches, keys[i:end])
	}

	return batches
}

func PrefetchHotData(ctx context.Context, patterns []string) error {
	if len(patterns) == 0 {
		return nil
	}

	prefetcher := NewPrefetcher(nil)
	for _, pattern := range patterns {
		prefetcher.AddPattern(pattern)
	}

	getFunc := func(ctx context.Context, key string) (string, error) {
		if Client == nil {
			return "", ErrCacheMiss
		}
		return Client.Get(ctx, key).Result()
	}

	prefetcher.Start(ctx, getFunc)

	timeout := 5 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}

	select {
	case <-ctx.Done():
		prefetcher.Stop()
		return ctx.Err()
	case <-time.After(timeout):
		prefetcher.Stop()
		return nil
	}
}

func NewEvictionHeap() *EvictionHeap {
	return &EvictionHeap{
		items: make([]*EvictionEntry, 0),
	}
}

func (h *EvictionHeap) Push(entry *EvictionEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.items = append(h.items, entry)
	h.bubbleUp(len(h.items) - 1)
}

func (h *EvictionHeap) Pop() *EvictionEntry {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.items) == 0 {
		return nil
	}

	top := h.items[0]
	last := h.items[len(h.items)-1]
	h.items[0] = last
	h.items = h.items[:len(h.items)-1]

	if len(h.items) > 0 {
		h.bubbleDown(0)
	}

	return top
}

func (h *EvictionHeap) Remove(key string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i, entry := range h.items {
		if entry.Key == key {
			last := h.items[len(h.items)-1]
			h.items[i] = last
			h.items = h.items[:len(h.items)-1]

			if i < len(h.items) {
				h.bubbleDown(i)
				h.bubbleUp(i)
			}
			return true
		}
	}

	return false
}

func (h *EvictionHeap) bubbleUp(idx int) {
	for idx > 0 {
		parent := (idx - 1) / 2
		if h.items[idx].Score >= h.items[parent].Score {
			break
		}
		h.items[idx], h.items[parent] = h.items[parent], h.items[idx]
		idx = parent
	}
}

func (h *EvictionHeap) bubbleDown(idx int) {
	length := len(h.items)
	for {
		left := 2*idx + 1
		right := 2*idx + 2
		smallest := idx

		if left < length && h.items[left].Score < h.items[smallest].Score {
			smallest = left
		}
		if right < length && h.items[right].Score < h.items[smallest].Score {
			smallest = right
		}

		if smallest == idx {
			break
		}

		h.items[idx], h.items[smallest] = h.items[smallest], h.items[idx]
		idx = smallest
	}
}

func (h *EvictionHeap) Size() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.items)
}

func (ec *EnhancedCacheV2) HybridEviction(ctx context.Context) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if ec.evictionHeap == nil {
		ec.evictionHeap = NewEvictionHeap()
	}

	entries := ec.collectEvictionCandidates(ctx)

	for _, entry := range entries {
		lruScore := ec.calculateLRUScore(entry)
		lfuScore := ec.calculateLFUScore(entry)
		hybridScore := ec.calculateHybridScore(entry, lruScore, lfuScore)

		entry.LRUScore = lruScore
		entry.LFUScore = lfuScore
		entry.Score = hybridScore

		ec.evictionHeap.Push(entry)
	}

	evicted := 0
	targetCount := len(entries) / 10
	if targetCount < 1 {
		targetCount = 1
	}

	for evicted < targetCount {
		entry := ec.evictionHeap.Pop()
		if entry == nil {
			break
		}

		if err := ec.evictEntry(ctx, entry); err == nil {
			evicted++
			ec.metrics.Evictions.Add(1)
			ec.metrics.EvictionHybrid.Add(1)
		}
	}

	ec.applyDecayFactor()

	return nil
}

func (ec *EnhancedCacheV2) collectEvictionCandidates(ctx context.Context) []*EvictionEntry {
	entries := make([]*EvictionEntry, 0, DefaultEvictionBatch)

	if Client != nil {
		var cursor uint64
		for {
			keys, nextCursor, err := Client.Scan(ctx, cursor, "*", 100).Result()
			if err != nil {
				break
			}

			for _, key := range keys {
				entry := &EvictionEntry{
					Key:        key,
					AccessTime: time.Now(),
					AccessCount: 1,
				}
				entries = append(entries, entry)

				if len(entries) >= DefaultEvictionBatch {
					return entries
				}
			}

			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}
	}

	return entries
}

func (ec *EnhancedCacheV2) calculateLRUScore(entry *EvictionEntry) float64 {
	if ec.config == nil || ec.config.Eviction == nil {
		return float64(time.Since(entry.AccessTime).Milliseconds())
	}

	age := time.Since(entry.AccessTime).Seconds()
	return math.Max(0, 1000-age)
}

func (ec *EnhancedCacheV2) calculateLFUScore(entry *EvictionEntry) float64 {
	if ec.config == nil || ec.config.Eviction == nil {
		return float64(entry.AccessCount)
	}

	return float64(entry.AccessCount) * ec.config.Eviction.DecayFactor
}

func (ec *EnhancedCacheV2) calculateHybridScore(entry *EvictionEntry, lruScore, lfuScore float64) float64 {
	if ec.config == nil || ec.config.Eviction == nil {
		return lfuScore * 0.7 + lruScore * 0.3
	}

	config := ec.config.Eviction
	return lfuScore*config.LFUWeight + lruScore*config.LRUWeight
}

func (ec *EnhancedCacheV2) evictEntry(ctx context.Context, entry *EvictionEntry) error {
	if Client != nil {
		if err := Client.Del(ctx, entry.Key).Err(); err != nil {
			return err
		}
	}

	if ec.localCache != nil {
		ec.localCache.Delete(entry.Key)
	}

	return nil
}

func (ec *EnhancedCacheV2) applyDecayFactor() {
	if ec.config == nil || ec.config.Eviction == nil {
		return
	}

	decay := ec.config.Eviction.DecayFactor

	ec.evictionHeap.mu.Lock()
	defer ec.evictionHeap.mu.Unlock()

	for _, entry := range ec.evictionHeap.items {
		entry.AccessCount = int64(float64(entry.AccessCount) * decay)
		entry.Score = ec.calculateHybridScore(entry, entry.LRUScore, entry.LFUScore)
	}
}

func NewFailoverManager(config *FailoverConfig) *FailoverManager {
	if config == nil {
		config = &FailoverConfig{
			Enabled:           true,
			MaxRetries:        3,
			RetryInterval:     5 * time.Second,
			HealthCheckPeriod: 10 * time.Second,
			DegradedThreshold: 3,
			FailureThreshold:  5,
			RecoveryTimeout:   DefaultRecoveryTimeout,
		}
	}

	return &FailoverManager{
		nodes:            make(map[string]*FailoverNode),
		maxRetries:       config.MaxRetries,
		retryInterval:    config.RetryInterval,
		healthCheckPeriod: config.HealthCheckPeriod,
	}
}

func (fm *FailoverManager) RegisterNode(name, addr string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fm.nodes[name] = &FailoverNode{
		Name:   name,
		Addr:   addr,
		Status: NodeStatusHealthy,
	}
}

func (fm *FailoverManager) UnregisterNode(name string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	delete(fm.nodes, name)
}

func (fm *FailoverManager) MarkNodeFailed(name string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if node, ok := fm.nodes[name]; ok {
		atomic.AddInt32(&node.FailCount, 1)
		node.LastFailTime = time.Now()
		node.HealthCheckCount = 0

		if atomic.LoadInt32(&node.FailCount) >= 5 {
			node.Status = NodeStatusFailed
		} else {
			node.Status = NodeStatusDegraded
		}
	}
}

func (fm *FailoverManager) MarkNodeRecovering(name string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if node, ok := fm.nodes[name]; ok {
		node.Status = NodeStatusRecovering
		node.RecoveryStart = time.Now()
	}
}

func (fm *FailoverManager) MarkNodeHealthy(name string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if node, ok := fm.nodes[name]; ok {
		atomic.StoreInt32(&node.FailCount, 0)
		node.Status = NodeStatusHealthy
		node.HealthCheckCount = 0
	}
}

func (fm *FailoverManager) StartHealthChecks(ctx context.Context, checkFunc func(addr string) error) {
	if !fm.activeFailover.CompareAndSwap(false, true) {
		return
	}

	go fm.healthCheckLoop(ctx, checkFunc)
}

func (fm *FailoverManager) StopHealthChecks() {
	fm.activeFailover.Store(false)
}

func (fm *FailoverManager) healthCheckLoop(ctx context.Context, checkFunc func(addr string) error) {
	ticker := time.NewTicker(fm.healthCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fm.performHealthChecks(ctx, checkFunc)
		}
	}
}

func (fm *FailoverManager) performHealthChecks(ctx context.Context, checkFunc func(addr string) error) {
	fm.mu.Lock()
	nodes := make([]*FailoverNode, 0, len(fm.nodes))
	for _, node := range fm.nodes {
		nodes = append(nodes, node)
	}
	fm.mu.Unlock()

	for _, node := range nodes {
		err := checkFunc(node.Addr)

		fm.mu.Lock()
		currentNode, exists := fm.nodes[node.Name]
		fm.mu.Unlock()

		if !exists {
			continue
		}

		if err != nil {
			atomic.AddInt32(&currentNode.HealthCheckCount, 1)

			if atomic.LoadInt32(&currentNode.HealthCheckCount) >= int32(fm.maxRetries) {
				fm.MarkNodeFailed(node.Name)
			}
		} else {
			if currentNode.Status != NodeStatusHealthy {
				fm.MarkNodeHealthy(node.Name)
			}
		}
	}
}

func (fm *FailoverManager) GetAvailableNode(excludeFailed bool) string {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	for _, node := range fm.nodes {
		if excludeFailed && node.Status == NodeStatusFailed {
			continue
		}
		return node.Addr
	}

	return ""
}

func FailoverRecovery(ctx context.Context) error {
	if Client == nil {
		return ErrNoNodesAvailable
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := Client.Ping(pingCtx).Err(); err == nil {
		return nil
	}

	if Cluster != nil {
		pingCtx2, cancel2 := context.WithTimeout(ctx, 5*time.Second)
		defer cancel2()

		if err := Cluster.Ping(pingCtx2).Err(); err == nil {
			return nil
		}
	}

	sentinelAddrs := getSentinelAddrs()
	for _, addr := range sentinelAddrs {
		client := goredis.NewClient(&goredis.Options{
			Addr:         addr,
			DialTimeout:  3 * time.Second,
			ReadTimeout:  2 * time.Second,
			WriteTimeout: 2 * time.Second,
		})

		if err := client.Ping(ctx).Err(); err == nil {
			info, err := client.Info(ctx, "sentinel").Result()
			if err == nil && containsMasterInfo(info) {
				client.Close()
				return nil
			}
			client.Close()
		}
	}

	return ErrRecoveryFailed
}

func getSentinelAddrs() []string {
	return []string{}
}

func containsMasterInfo(info string) bool {
	return len(info) > 0
}

func NewLocalCacheLayer(maxSize int, ttl time.Duration) *LocalCacheLayer {
	if maxSize <= 0 {
		maxSize = 5000
	}
	if ttl <= 0 {
		ttl = 30 * time.Second
	}

	return &LocalCacheLayer{
		cache:    &sync.Map{},
		maxSize:  maxSize,
		ttl:      ttl,
	}
}

func (lc *LocalCacheLayer) Get(key string) ([]byte, bool) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	val, ok := lc.cache.Load(key)
	if !ok {
		lc.misses.Add(1)
		return nil, false
	}

	entry := val.(*localCacheEntry)
	if time.Now().After(entry.expiresAt) {
		lc.cache.Delete(key)
		lc.misses.Add(1)
		return nil, false
	}

	lc.hits.Add(1)
	return entry.value, true
}

func (lc *LocalCacheLayer) Set(key string, value []byte) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if _, exists := lc.cache.Load(key); !exists {
		if lc.currentSize() >= lc.maxSize {
			lc.evictOldest()
		}
	}

	entry := &localCacheEntry{
		value:     value,
		expiresAt: time.Now().Add(lc.ttl),
	}
	lc.cache.Store(key, entry)
}

func (lc *LocalCacheLayer) Delete(key string) {
	lc.cache.Delete(key)
}

func (lc *LocalCacheLayer) Clear() {
	lc.cache = &sync.Map{}
}

func (lc *LocalCacheLayer) currentSize() int {
	count := 0
	lc.cache.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

func (lc *LocalCacheLayer) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	lc.cache.Range(func(key, value interface{}) bool {
		entry := value.(*localCacheEntry)
		if oldestKey == "" || entry.expiresAt.Before(oldestTime) {
			oldestKey = key.(string)
			oldestTime = entry.expiresAt
		}
		return true
	})

	if oldestKey != "" {
		lc.cache.Delete(oldestKey)
		lc.evictions.Add(1)
	}
}

type localCacheEntry struct {
	value     []byte
	expiresAt time.Time
}

func NewHotDataTracker() *HotDataTracker {
	return &HotDataTracker{
		accessLog:    make([]CacheAccessRecord, 0),
		maxLogSize:   10000,
		windowSize:   10 * time.Minute,
		hotThreshold: 10,
	}
}

func (h *HotDataTracker) RecordAccess(key string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	record := CacheAccessRecord{
		Key:       key,
		Timestamp: time.Now(),
	}

	h.accessLog = append(h.accessLog, record)

	if len(h.accessLog) > h.maxLogSize {
		h.accessLog = h.accessLog[len(h.accessLog)-h.maxLogSize:]
	}

	h.cleanupOldRecords()
}

func (h *HotDataTracker) GetHotKeys() []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.cleanupOldRecords()

	frequency := make(map[string]int)
	for _, record := range h.accessLog {
		frequency[record.Key]++
	}

	hotKeys := make([]string, 0)
	for key, count := range frequency {
		if int64(count) >= h.hotThreshold {
			hotKeys = append(hotKeys, key)
		}
	}

	return hotKeys
}

func (h *HotDataTracker) cleanupOldRecords() {
	cutoff := time.Now().Add(-h.windowSize)
	newLog := make([]CacheAccessRecord, 0, len(h.accessLog))

	for _, record := range h.accessLog {
		if record.Timestamp.After(cutoff) {
			newLog = append(newLog, record)
		}
	}

	h.accessLog = newLog
}

func (ec *EnhancedCacheV2) Get(ctx context.Context, key string) (string, error) {
	if ec.localCache != nil {
		if val, ok := ec.localCache.Get(key); ok {
			ec.metrics.Hits.Add(1)
			return string(val), nil
		}
	}

	ec.mu.RLock()
	hash := ec.consistentHash
	ec.mu.RUnlock()

	nodeAddr, err := hash.GetNode(key)
	if err != nil {
		ec.metrics.Misses.Add(1)
		return "", err
	}

	ec.mu.RLock()
	client, ok := ec.clients[nodeAddr]
	ec.mu.RUnlock()

	if !ok {
		ec.metrics.Misses.Add(1)
		return "", ErrNoNodesAvailable
	}

	val, err := client.Get(ctx, key)
	if err != nil {
		ec.metrics.Misses.Add(1)
		return "", err
	}

	ec.metrics.Hits.Add(1)

	if ec.localCache != nil {
		ec.localCache.Set(key, []byte(val))
	}

	ec.hotDataTracker.RecordAccess(key)

	return val, nil
}

func (ec *EnhancedCacheV2) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	ec.mu.RLock()
	hash := ec.consistentHash
	ec.mu.RUnlock()

	nodeAddr, err := hash.GetNode(key)
	if err != nil {
		return err
	}

	ec.mu.RLock()
	client, ok := ec.clients[nodeAddr]
	ec.mu.RUnlock()

	if !ok {
		return ErrNoNodesAvailable
	}

	if err := client.Set(ctx, key, value, expiration); err != nil {
		ec.failoverMgr.MarkNodeFailed(nodeAddr)
		return err
	}

	if ec.localCache != nil {
		valStr, ok := value.(string)
		if ok {
			ec.localCache.Set(key, []byte(valStr))
		}
	}

	return nil
}

func (ec *EnhancedCacheV2) Delete(ctx context.Context, keys ...string) error {
	deleted := 0
	for _, key := range keys {
		ec.mu.RLock()
		hash := ec.consistentHash
		ec.mu.RUnlock()

		nodeAddr, err := hash.GetNode(key)
		if err != nil {
			continue
		}

		ec.mu.RLock()
		client, ok := ec.clients[nodeAddr]
		ec.mu.RUnlock()

		if !ok {
			continue
		}

		if _, err := client.Del(ctx, key); err == nil {
			deleted++
		}

		if ec.localCache != nil {
			ec.localCache.Delete(key)
		}
	}

	if deleted == 0 && len(keys) > 0 {
		return errors.New("failed to delete any keys")
	}

	return nil
}

func (ec *EnhancedCacheV2) RegisterClient(addr string, client CacheNode) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.clients[addr] = client
	ec.consistentHash.AddNode(addr, 1)
	ec.failoverMgr.RegisterNode(addr, addr)
}

func (ec *EnhancedCacheV2) UnregisterClient(addr string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	delete(ec.clients, addr)
	ec.consistentHash.RemoveNode(addr)
	ec.failoverMgr.UnregisterNode(addr)
}

func (ec *EnhancedCacheV2) GetMetrics() *CacheV2Metrics {
	return ec.metrics
}

func (ec *EnhancedCacheV2) GetHotKeys() []string {
	return ec.hotDataTracker.GetHotKeys()
}

func (ec *EnhancedCacheV2) StartPrefetching(ctx context.Context) {
	ec.prefetcher.Start(ctx, ec.Get)
}

func (ec *EnhancedCacheV2) StopPrefetching() {
	ec.prefetcher.Stop()
}

func (ec *EnhancedCacheV2) StartFailoverRecovery(ctx context.Context) {
	checkFunc := func(addr string) error {
		ec.mu.RLock()
		client, ok := ec.clients[addr]
		ec.mu.RUnlock()

		if !ok {
			return errors.New("client not found")
		}

		return client.Ping(ctx)
	}

	ec.failoverMgr.StartHealthChecks(ctx, checkFunc)
}

func (ec *EnhancedCacheV2) StopFailoverRecovery() {
	ec.failoverMgr.StopHealthChecks()
}

func (ec *EnhancedCacheV2) TriggerEviction(ctx context.Context) error {
	return ec.HybridEviction(ctx)
}

func (ec *EnhancedCacheV2) RecoverFromFailure(ctx context.Context) error {
	return FailoverRecovery(ctx)
}

func (ec *EnhancedCacheV2) ClearLocalCache() {
	if ec.localCache != nil {
		ec.localCache.Clear()
	}
}

func (ec *EnhancedCacheV2) AddPrefetchPattern(pattern string) {
	ec.prefetcher.AddPattern(pattern)
}

func (ec *EnhancedCacheV2) GetNodeDistribution() map[string]int {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	distribution := make(map[string]int)

	keys := []string{
		"key1", "key2", "key3", "key4", "key5",
		"user:1", "user:2", "user:3",
		"product:100", "product:200",
	}

	for _, key := range keys {
		node, err := ec.consistentHash.GetNode(key)
		if err == nil {
			distribution[node]++
		}
	}

	return distribution
}

func (ec *EnhancedCacheV2) GetNodeStatus() map[string]NodeStatus {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	status := make(map[string]NodeStatus)
	for addr, node := range ec.consistentHash.nodes {
		status[addr] = node.Status
	}

	return status
}

var globalEnhancedCacheV2 *EnhancedCacheV2
var globalEnhancedCacheV2Once sync.Once

func InitEnhancedCacheV2(config *EnhancedCacheV2Config) {
	globalEnhancedCacheV2Once.Do(func() {
		globalEnhancedCacheV2 = NewEnhancedCacheV2(config)
	})
}

func GetEnhancedCacheV2() *EnhancedCacheV2 {
	if globalEnhancedCacheV2 == nil {
		InitEnhancedCacheV2(nil)
	}
	return globalEnhancedCacheV2
}
