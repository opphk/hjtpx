package performance

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type SubMillisecondOptimizer struct {
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	isRunning   bool

	// 连接池
	connectionPool *ConnectionPool

	// 缓存预热
	cacheWarmer *CacheWarmer

	// 零拷贝缓冲区
	zeroCopyBuffer *ZeroCopyBuffer

	// 统计信息
	stats *SubMillisecondStats
}

type ConnectionPool struct {
	mu           sync.RWMutex
	pool         map[string]*PooledConnection
	maxSize      int
	idleTimeout  time.Duration
	stats        *PoolStats
}

type PooledConnection struct {
	conn       interface{}
	createdAt  time.Time
	lastUsedAt time.Time
	inUse      atomic.Bool
}

type CacheWarmer struct {
	mu           sync.RWMutex
	preloadKeys  []string
	warmed       map[string]bool
	warmerChan   chan string
}

type ZeroCopyBuffer struct {
	buffers    [][]byte
	bufferSize int
	mu         sync.Mutex
	available  chan int
}

type SubMillisecondStats struct {
	PoolHits        atomic.Int64
	PoolMisses      atomic.Int64
	CachePreloadHits atomic.Int64
	ZeroCopyAllocations atomic.Int64
	AvgLatency      atomic.Int64
	LastUpdate      atomic.Value
}

type PoolStats struct {
	TotalConns     atomic.Int64
	ActiveConns    atomic.Int64
	IdleConns      atomic.Int64
}

func NewSubMillisecondOptimizer() *SubMillisecondOptimizer {
	ctx, cancel := context.WithCancel(context.Background())
	return &SubMillisecondOptimizer{
		ctx:             ctx,
		cancel:          cancel,
		connectionPool:  NewConnectionPool(1000, 5*time.Minute),
		cacheWarmer:     NewCacheWarmer(),
		zeroCopyBuffer:  NewZeroCopyBuffer(4096, 10000),
		stats:           &SubMillisecondStats{},
	}
}

func NewConnectionPool(maxSize int, idleTimeout time.Duration) *ConnectionPool {
	return &ConnectionPool{
		pool:         make(map[string]*PooledConnection, maxSize),
		maxSize:      maxSize,
		idleTimeout:  idleTimeout,
		stats:        &PoolStats{},
	}
}

func NewCacheWarmer() *CacheWarmer {
	return &CacheWarmer{
		warmed:     make(map[string]bool),
		warmerChan: make(chan string, 1000),
	}
}

func NewZeroCopyBuffer(bufferSize, count int) *ZeroCopyBuffer {
	buffers := make([][]byte, count)
	available := make(chan int, count)
	for i := 0; i < count; i++ {
		buffers[i] = make([]byte, 0, bufferSize)
		available <- i
	}
	return &ZeroCopyBuffer{
		buffers:    buffers,
		bufferSize: bufferSize,
		available:  available,
	}
}

func (s *SubMillisecondOptimizer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return nil
	}
	s.isRunning = true

	go s.cleanupIdleConnections()
	go s.preloadCacheEntries()
	go s.updateStats()

	log.Println("[SubMillisecondOptimizer] Started successfully")
	return nil
}

func (s *SubMillisecondOptimizer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}
	s.isRunning = false
	s.cancel()

	log.Println("[SubMillisecondOptimizer] Stopped")
}

func (s *SubMillisecondOptimizer) GetConnection(key string, factory func() (interface{}, error)) (interface{}, error) {
	start := time.Now()
	defer s.recordLatency(start)

	if conn := s.connectionPool.Get(key); conn != nil {
		s.stats.PoolHits.Add(1)
		return conn, nil
	}

	s.stats.PoolMisses.Add(1)
	conn, err := factory()
	if err != nil {
		return nil, err
	}
	s.connectionPool.Put(key, conn)
	return conn, nil
}

func (s *SubMillisecondOptimizer) AddPreloadKey(key string) {
	s.cacheWarmer.AddKey(key)
}

func (s *SubMillisecondOptimizer) WarmCache(key string, loader func() interface{}) {
	s.cacheWarmer.Warm(key, loader)
	s.stats.CachePreloadHits.Add(1)
}

func (s *SubMillisecondOptimizer) GetBuffer() []byte {
	s.stats.ZeroCopyAllocations.Add(1)
	return s.zeroCopyBuffer.Get()
}

func (s *SubMillisecondOptimizer) ReleaseBuffer(buf []byte) {
	s.zeroCopyBuffer.Put(buf)
}

func (s *SubMillisecondOptimizer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"pool_hits":              s.stats.PoolHits.Load(),
		"pool_misses":            s.stats.PoolMisses.Load(),
		"cache_preload_hits":     s.stats.CachePreloadHits.Load(),
		"zero_copy_allocations":  s.stats.ZeroCopyAllocations.Load(),
		"avg_latency_ns":         s.stats.AvgLatency.Load(),
		"last_update":            s.stats.LastUpdate.Load(),
	}
}

func (s *SubMillisecondOptimizer) cleanupIdleConnections() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.connectionPool.CleanupIdle()
		}
	}
}

func (s *SubMillisecondOptimizer) preloadCacheEntries() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case key := <-s.cacheWarmer.warmerChan:
			s.cacheWarmer.markWarmed(key)
		}
	}
}

func (s *SubMillisecondOptimizer) updateStats() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.stats.LastUpdate.Store(time.Now())
		}
	}
}

func (s *SubMillisecondOptimizer) recordLatency(start time.Time) {
	latency := time.Since(start).Nanoseconds()
	old := s.stats.AvgLatency.Load()
	if old == 0 {
		s.stats.AvgLatency.Store(latency)
	} else {
		s.stats.AvgLatency.Store((old*9 + latency) / 10)
	}
}

func (p *ConnectionPool) Get(key string) interface{} {
	p.mu.RLock()
	conn, exists := p.pool[key]
	p.mu.RUnlock()

	if exists && conn.inUse.CompareAndSwap(false, true) {
		conn.lastUsedAt = time.Now()
		p.stats.ActiveConns.Add(1)
		return conn.conn
	}
	return nil
}

func (p *ConnectionPool) Put(key string, conn interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.pool) >= p.maxSize {
		return
	}

	pooledConn := &PooledConnection{
		conn:       conn,
		createdAt:  time.Now(),
		lastUsedAt: time.Now(),
	}
	p.pool[key] = pooledConn
	p.stats.TotalConns.Add(1)
	p.stats.IdleConns.Add(1)
}

func (p *ConnectionPool) CleanupIdle() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for key, conn := range p.pool {
		if !conn.inUse.Load() && now.Sub(conn.lastUsedAt) > p.idleTimeout {
			delete(p.pool, key)
			p.stats.IdleConns.Add(-1)
			p.stats.TotalConns.Add(-1)
		}
	}
}

func (cw *CacheWarmer) AddKey(key string) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.preloadKeys = append(cw.preloadKeys, key)
}

func (cw *CacheWarmer) Warm(key string, loader func() interface{}) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.warmed[key] = true
	loader()
}

func (cw *CacheWarmer) markWarmed(key string) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.warmed[key] = true
}

func (z *ZeroCopyBuffer) Get() []byte {
	select {
	case idx := <-z.available:
		return z.buffers[idx]
	default:
		return make([]byte, z.bufferSize)
	}
}

func (z *ZeroCopyBuffer) Put(buf []byte) {
	if len(buf) != z.bufferSize {
		return
	}

	z.mu.Lock()
	defer z.mu.Unlock()

	idx := -1
	for i, b := range z.buffers {
		if &b[0] == &buf[0] {
			idx = i
			break
		}
	}

	if idx != -1 {
		select {
		case z.available <- idx:
		default:
		}
	}
}
