package redis

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type MultiLevelCache struct {
	l1        *L1MemoryCache
	l2        *goredis.Client
	l2Enabled bool
	metrics   *MultiLevelCacheMetrics
	mu        sync.RWMutex
}

type L1MemoryCache struct {
	items      map[string]*L1CacheItem
	mu         sync.RWMutex
	maxSize    int
	ttl        time.Duration
	hitCount   int64
	missCount  int64
	evictCount int64
}

type L1CacheItem struct {
	Value      interface{}
	ExpireAt   time.Time
	AccessTime time.Time
	AccessCnt  int64
}

type MultiLevelCacheMetrics struct {
	l1Hits       int64
	l1Misses     int64
	l2Hits       int64
	l2Misses     int64
	l2Errors     int64
	totalGets    int64
	hitRate      float64
	mu           sync.RWMutex
}

type MultiLevelCacheOption func(*MultiLevelCache)

func WithL1CacheSize(maxSize int) MultiLevelCacheOption {
	return func(m *MultiLevelCache) {
		m.l1.maxSize = maxSize
	}
}

func WithL1CacheTTL(ttl time.Duration) MultiLevelCacheOption {
	return func(m *MultiLevelCache) {
		m.l1.ttl = ttl
	}
}

func NewMultiLevelCache(l2 *goredis.Client, opts ...MultiLevelCacheOption) *MultiLevelCache {
	cache := &MultiLevelCache{
		l1: &L1MemoryCache{
			items:   make(map[string]*L1CacheItem),
			maxSize: 10000,
			ttl:     30 * time.Second,
		},
		l2:        l2,
		l2Enabled: l2 != nil,
		metrics: &MultiLevelCacheMetrics{},
	}

	for _, opt := range opts {
		opt(cache)
	}

	go cache.startCleanupRoutine()
	go cache.startMetricsCalculation()

	return cache
}

func (m *MultiLevelCache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	atomic.AddInt64(&m.metrics.totalGets, 1)

	m.l1.mu.RLock()
	item, exists := m.l1.items[key]
	m.l1.mu.RUnlock()

	if exists {
		if time.Now().Before(item.ExpireAt) {
			atomic.AddInt64(&m.metrics.l1Hits, 1)
			m.l1.mu.Lock()
			item.AccessTime = time.Now()
			item.AccessCnt++
			m.l1.mu.Unlock()
			return item.Value, true, nil
		}
		m.l1.mu.Lock()
		delete(m.l1.items, key)
		m.l1.mu.Unlock()
		atomic.AddInt64(&m.l1.evictCount, 1)
	}

	atomic.AddInt64(&m.metrics.l1Misses, 1)

	if !m.l2Enabled {
		return nil, false, nil
	}

	val, err := m.l2.Get(ctx, key).Result()
	if err == goredis.Nil {
		atomic.AddInt64(&m.metrics.l2Misses, 1)
		return nil, false, nil
	}
	if err != nil {
		atomic.AddInt64(&m.metrics.l2Errors, 1)
		return nil, false, err
	}

	atomic.AddInt64(&m.metrics.l2Hits, 1)

	m.l1.mu.Lock()
	if len(m.l1.items) < m.l1.maxSize {
		m.l1.items[key] = &L1CacheItem{
			Value:      val,
			ExpireAt:   time.Now().Add(m.l1.ttl),
			AccessTime: time.Now(),
			AccessCnt:  1,
		}
	}
	m.l1.mu.Unlock()

	return val, true, nil
}

func (m *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.l1.mu.Lock()
	evicted := false
	if len(m.l1.items) >= m.l1.maxSize {
		m.evictL1Oldest()
		evicted = true
	}

	m.l1.items[key] = &L1CacheItem{
		Value:      value,
		ExpireAt:   time.Now().Add(m.l1.ttl),
		AccessTime: time.Now(),
		AccessCnt:  1,
	}
	m.l1.mu.Unlock()

	if evicted {
		atomic.AddInt64(&m.l1.evictCount, 1)
	}

	if !m.l2Enabled {
		return nil
	}

	return m.l2.Set(ctx, key, value, ttl).Err()
}

func (m *MultiLevelCache) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		m.l1.mu.Lock()
		delete(m.l1.items, key)
		m.l1.mu.Unlock()
	}

	if !m.l2Enabled {
		return nil
	}

	return m.l2.Del(ctx, keys...).Err()
}

func (m *MultiLevelCache) evictL1Oldest() {
	if len(m.l1.items) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time

	for key, item := range m.l1.items {
		if oldestTime.IsZero() || item.AccessTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.AccessTime
		}
	}

	if oldestKey != "" {
		delete(m.l1.items, oldestKey)
	}
}

func (m *MultiLevelCache) startCleanupRoutine() {
	ticker := time.NewTicker(m.l1.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		m.l1.mu.Lock()
		now := time.Now()
		keysToDelete := make([]string, 0, 100)

		for key, item := range m.l1.items {
			if now.After(item.ExpireAt) {
				keysToDelete = append(keysToDelete, key)
			}
		}

		for _, key := range keysToDelete {
			delete(m.l1.items, key)
		}
		m.l1.mu.Unlock()

		if len(keysToDelete) > 0 {
			atomic.AddInt64(&m.l1.evictCount, int64(len(keysToDelete)))
		}
	}
}

func (m *MultiLevelCache) startMetricsCalculation() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.metrics.mu.Lock()
		total := m.metrics.l1Hits + m.metrics.l1Misses + m.metrics.l2Hits + m.metrics.l2Misses
		if total > 0 {
			hits := m.metrics.l1Hits + m.metrics.l2Hits
			m.metrics.hitRate = float64(hits) / float64(total) * 100
		}
		m.metrics.mu.Unlock()
	}
}

func (m *MultiLevelCache) GetMetrics() map[string]interface{} {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	m.l1.mu.RLock()
	l1Size := len(m.l1.items)
	m.l1.mu.RUnlock()

	return map[string]interface{}{
		"l1_size":        l1Size,
		"l1_max_size":    m.l1.maxSize,
		"l1_hits":        m.metrics.l1Hits,
		"l1_misses":      m.metrics.l1Misses,
		"l1_evicts":      m.l1.evictCount,
		"l2_hits":        m.metrics.l2Hits,
		"l2_misses":      m.metrics.l2Misses,
		"l2_errors":      m.metrics.l2Errors,
		"total_gets":     m.metrics.totalGets,
		"hit_rate":       m.metrics.hitRate,
		"l1_hit_rate":    m.calculateL1HitRate(),
	}
}

func (m *MultiLevelCache) calculateL1HitRate() float64 {
	hits := atomic.LoadInt64(&m.metrics.l1Hits)
	misses := atomic.LoadInt64(&m.metrics.l1Misses)
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

func (m *MultiLevelCache) ClearL1() {
	m.l1.mu.Lock()
	m.l1.items = make(map[string]*L1CacheItem)
	m.l1.mu.Unlock()
}

func (m *MultiLevelCache) ClearAll(ctx context.Context) error {
	m.ClearL1()

	if !m.l2Enabled {
		return nil
	}

	pattern := "*"
	iter := m.l2.Scan(ctx, 0, pattern, 1000).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		return m.l2.Del(ctx, keys...).Err()
	}
	return nil
}

type PipelineCache struct {
	client     *goredis.Client
	batchSize  int
	maxWorkers int
	semaphore  chan struct{}
}

func NewPipelineCache(client *goredis.Client, batchSize, maxWorkers int) *PipelineCache {
	if batchSize <= 0 {
		batchSize = 100
	}
	if maxWorkers <= 0 {
		maxWorkers = 10
	}

	return &PipelineCache{
		client:     client,
		batchSize:  batchSize,
		maxWorkers: maxWorkers,
		semaphore:  make(chan struct{}, maxWorkers),
	}
}

func (p *PipelineCache) PipelineGet(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return make(map[string]string), nil
	}

	results := make(map[string]string)
	mu := sync.Mutex{}
	errCh := make(chan error, (len(keys)+p.batchSize-1)/p.batchSize)
	wg := sync.WaitGroup{}

	for i := 0; i < len(keys); i += p.batchSize {
		end := i + p.batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]

		p.semaphore <- struct{}{}
		wg.Add(1)

		go func(batch []string) {
			defer wg.Done()
			defer func() { <-p.semaphore }()

			pipe := p.client.Pipeline()
			cmds := make([]*goredis.StringCmd, len(batch))
			for j, key := range batch {
				cmds[j] = pipe.Get(ctx, key)
			}

			_, err := pipe.Exec(ctx)
			if err != nil && err != goredis.Nil {
				errCh <- err
				return
			}

			localResults := make(map[string]string)
			for j, cmd := range cmds {
				val, err := cmd.Result()
				if err == nil {
					localResults[batch[j]] = val
				}
			}

			mu.Lock()
			for k, v := range localResults {
				results[k] = v
			}
			mu.Unlock()
		}(batch)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return results, err
		}
	}

	return results, nil
}

func (p *PipelineCache) PipelineSet(ctx context.Context, items map[string]string, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	errCh := make(chan error, (len(items)+p.batchSize-1)/p.batchSize)
	wg := sync.WaitGroup{}

	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}

	for i := 0; i < len(keys); i += p.batchSize {
		end := i + p.batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batchKeys := keys[i:end]

		p.semaphore <- struct{}{}
		wg.Add(1)

		go func(batch []string) {
			defer wg.Done()
			defer func() { <-p.semaphore }()

			pipe := p.client.Pipeline()
			for _, key := range batch {
				pipe.Set(ctx, key, items[key], ttl)
			}

			_, err := pipe.Exec(ctx)
			if err != nil {
				errCh <- err
				return
			}
		}(batchKeys)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *PipelineCache) PipelineDelete(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	pipe := p.client.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	return err
}

type OptimizedRedisClient struct {
	client        *goredis.Client
	poolStats     *PoolStatsCollector
	commandStats  *CommandStatsCollector
	slowLog       *SlowLogCollector
	healthChecker *HealthChecker
}

type PoolStatsCollector struct {
	mu           sync.RWMutex
	statsHistory []PoolStatsType
	maxHistory   int
}

type PoolStatsType struct {
	Timestamp    time.Time
	TotalConns   int
	IdleConns    int
	InUseConns   int
	Timeouts     int
	Hits         int64
	Misses       int64
}

type CommandStatsCollector struct {
	mu          sync.RWMutex
	commandCount map[string]int64
	commandTime  map[string]time.Duration
}

type SlowLogCollector struct {
	mu         sync.RWMutex
	slowLogs   []SlowLogEntry
	maxEntries int
	threshold  time.Duration
}

type SlowLogEntry struct {
	Timestamp   time.Time
	Command     string
	Duration    time.Duration
	Args        []string
}

type HealthChecker struct {
	client      *goredis.Client
	interval    time.Duration
	lastCheck   time.Time
	isHealthy   bool
	latency     time.Duration
	failures    int
	maxFailures int
	stopCh      chan struct{}
	mu          sync.RWMutex
}

func NewOptimizedRedisClient(client *goredis.Client) *OptimizedRedisClient {
	orc := &OptimizedRedisClient{
		client:       client,
		poolStats:    &PoolStatsCollector{maxHistory: 1000},
		commandStats: &CommandStatsCollector{
			commandCount: make(map[string]int64),
			commandTime:  make(map[string]time.Duration),
		},
		slowLog: &SlowLogCollector{
			maxEntries: 100,
			threshold:  100 * time.Millisecond,
		},
		healthChecker: &HealthChecker{
			client:      client,
			interval:    10 * time.Second,
			maxFailures:  5,
			stopCh:      make(chan struct{}),
		},
	}

	go orc.collectPoolStats()
	go orc.healthChecker.Start()

	return orc
}

func (p *PoolStatsCollector) AddSnapshot(stats PoolStatsType) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.statsHistory = append(p.statsHistory, stats)
	if len(p.statsHistory) > p.maxHistory {
		p.statsHistory = p.statsHistory[1:]
	}
}

func (p *PoolStatsCollector) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.statsHistory) == 0 {
		return nil
	}

	latest := p.statsHistory[len(p.statsHistory)-1]

	var avgIdle float64
	var avgInUse float64
	var totalHits, totalMisses int64

	for _, s := range p.statsHistory {
		avgIdle += float64(s.IdleConns)
		avgInUse += float64(s.InUseConns)
		totalHits += s.Hits
		totalMisses += s.Misses
	}

	count := float64(len(p.statsHistory))
	avgIdle /= count
	avgInUse /= count

	hitRate := float64(0)
	if totalHits+totalMisses > 0 {
		hitRate = float64(totalHits) / float64(totalHits+totalMisses) * 100
	}

	return map[string]interface{}{
		"current_total":    latest.TotalConns,
		"current_idle":     latest.IdleConns,
		"current_in_use":   latest.InUseConns,
		"avg_idle":         avgIdle,
		"avg_in_use":       avgInUse,
		"hit_rate":         hitRate,
		"total_timeouts":   latest.Timeouts,
		"history_length":   len(p.statsHistory),
	}
}

func (c *CommandStatsCollector) Record(command string, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.commandCount[command]++
	c.commandTime[command] += duration
}

func (c *CommandStatsCollector) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := make(map[string]interface{})
	for cmd, count := range c.commandCount {
		avgTime := time.Duration(0)
		if count > 0 {
			avgTime = c.commandTime[cmd] / time.Duration(count)
		}
		stats[cmd] = map[string]interface{}{
			"count":     count,
			"total_time": c.commandTime[cmd].String(),
			"avg_time":  avgTime.String(),
		}
	}

	return stats
}

func (s *SlowLogCollector) Add(entry SlowLogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.slowLogs = append(s.slowLogs, entry)
	if len(s.slowLogs) > s.maxEntries {
		s.slowLogs = s.slowLogs[1:]
	}
}

func (s *SlowLogCollector) GetLogs() []SlowLogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs := make([]SlowLogEntry, len(s.slowLogs))
	copy(logs, s.slowLogs)
	return logs
}

func (h *HealthChecker) Start() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.Check()
		}
	}
}

func (h *HealthChecker) Stop() {
	close(h.stopCh)
}

func (h *HealthChecker) Check() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	err := h.client.Ping(ctx).Err()
	h.latency = time.Since(start)

	h.mu.Lock()
	defer h.mu.Unlock()

	h.lastCheck = time.Now()

	if err != nil {
		h.isHealthy = false
		h.failures++
	} else {
		h.isHealthy = true
		h.failures = 0
	}
}

func (h *HealthChecker) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isHealthy
}

func (h *HealthChecker) GetLatency() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.latency
}

func (orc *OptimizedRedisClient) collectPoolStats() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if orc.client != nil {
			stats := orc.client.PoolStats()
			orc.poolStats.AddSnapshot(PoolStatsFromRedis(stats))
		}
	}
}

func PoolStatsFromRedis(stats *goredis.PoolStats) PoolStatsType {
	return PoolStatsType{
		Timestamp:  time.Now(),
		TotalConns: int(stats.TotalConns),
		IdleConns:  int(stats.IdleConns),
		Timeouts:   int(stats.Timeouts),
		Hits:       int64(stats.Hits),
		Misses:     int64(stats.Misses),
		InUseConns: int(stats.TotalConns) - int(stats.IdleConns),
	}
}

func (orc *OptimizedRedisClient) GetAllMetrics() map[string]interface{} {
	return map[string]interface{}{
		"pool":    orc.poolStats.GetStats(),
		"command": orc.commandStats.GetStats(),
		"slowlog": len(orc.slowLog.GetLogs()),
		"health": map[string]interface{}{
			"is_healthy": orc.healthChecker.IsHealthy(),
			"latency":    orc.healthChecker.GetLatency().String(),
		},
	}
}

func (orc *OptimizedRedisClient) Close() error {
	orc.healthChecker.Stop()
	return orc.client.Close()
}

func GetOptimizedRedisClient(client *goredis.Client) *OptimizedRedisClient {
	return NewOptimizedRedisClient(client)
}
