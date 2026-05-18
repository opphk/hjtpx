package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type LRUCache struct {
	capacity int
	mu       sync.RWMutex
	items    map[string]*LRUItem
	head     *LRUItem
	tail     *LRUItem
}

type LRUItem struct {
	Key        string
	Value      []byte
	ExpiresAt  time.Time
	Version    int64
	Prev       *LRUItem
	Next       *LRUItem
	accessTime time.Time
	size       int
}

func NewLRUCache(capacity int) *LRUCache {
	c := &LRUCache{
		capacity: capacity,
		items:    make(map[string]*LRUItem),
	}
	c.head = &LRUItem{}
	c.tail = &LRUItem{}
	c.head.Next = c.tail
	c.tail.Prev = c.head
	return c
}

func (c *LRUCache) Get(key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if !exists {
		return nil, ErrCacheMiss
	}

	if time.Now().After(item.ExpiresAt) {
		c.remove(item)
		delete(c.items, key)
		return nil, ErrCacheMiss
	}

	c.moveToFront(item)
	item.accessTime = time.Now()

	return item.Value, nil
}

func (c *LRUCache) Set(key string, value []byte, ttl time.Duration, version int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		c.remove(item)
	}

	for len(c.items) >= c.capacity {
		c.removeBack()
	}

	item := &LRUItem{
		Key:        key,
		Value:      value,
		ExpiresAt:  time.Now().Add(ttl),
		Version:    version,
		accessTime: time.Now(),
		size:       len(value),
	}

	c.addToFront(item)
	c.items[key] = item
}

func (c *LRUCache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if !exists {
		return false
	}

	c.remove(item)
	delete(c.items, key)
	return true
}

func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*LRUItem)
	c.head.Next = c.tail
	c.tail.Prev = c.head
}

func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

func (c *LRUCache) addToFront(item *LRUItem) {
	item.Prev = c.head
	item.Next = c.head.Next
	c.head.Next.Prev = item
	c.head.Next = item
}

func (c *LRUCache) remove(item *LRUItem) {
	item.Prev.Next = item.Next
	item.Next.Prev = item.Prev
	item.Prev = nil
	item.Next = nil
}

func (c *LRUCache) removeBack() {
	item := c.tail.Prev
	if item != c.head {
		c.remove(item)
		delete(c.items, item.Key)
	}
}

func (c *LRUCache) moveToFront(item *LRUItem) {
	c.remove(item)
	c.addToFront(item)
}

type PipelineBatcher struct {
	client        *goredis.Client
	batchSize     int
	flushInterval time.Duration
	mu            sync.Mutex
	pendingSets   map[string]*pendingSet
	pendingGets   []string
	pendingDel    []string
	done          chan struct{}
	wg            sync.WaitGroup
}

type pendingSet struct {
	Value interface{}
	TTL   time.Duration
	Tags  []string
}

func NewPipelineBatcher(batchSize int, flushInterval time.Duration) *PipelineBatcher {
	return &PipelineBatcher{
		batchSize:     batchSize,
		flushInterval: flushInterval,
		pendingSets:   make(map[string]*pendingSet),
		pendingGets:   make([]string, 0, batchSize),
		pendingDel:    make([]string, 0, batchSize),
		done:          make(chan struct{}),
	}
}

func (pb *PipelineBatcher) Start() {
	pb.wg.Add(1)
	go pb.flushLoop()
}

func (pb *PipelineBatcher) Stop() {
	close(pb.done)
	pb.wg.Wait()
}

func (pb *PipelineBatcher) flushLoop() {
	defer pb.wg.Done()
	ticker := time.NewTicker(pb.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pb.done:
			pb.flush()
			return
		case <-ticker.C:
			pb.flush()
		}
	}
}

func (pb *PipelineBatcher) AddSet(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.pendingSets[key] = &pendingSet{
		Value: value,
		TTL:   ttl,
	}

	if len(pb.pendingSets) >= pb.batchSize {
		go pb.flush()
	}
}

func (pb *PipelineBatcher) AddGet(ctx context.Context, key string) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.pendingGets = append(pb.pendingGets, key)

	if len(pb.pendingGets) >= pb.batchSize {
		go pb.flush()
	}
}

func (pb *PipelineBatcher) AddDelete(ctx context.Context, key string) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.pendingDel = append(pb.pendingDel, key)

	if len(pb.pendingDel) >= pb.batchSize {
		go pb.flush()
	}
}

func (pb *PipelineBatcher) flush() {
	pb.mu.Lock()
	if len(pb.pendingSets) == 0 && len(pb.pendingGets) == 0 && len(pb.pendingDel) == 0 {
		pb.mu.Unlock()
		return
	}

	sets := pb.pendingSets
	gets := pb.pendingGets
	dels := pb.pendingDel

	pb.pendingSets = make(map[string]*pendingSet)
	pb.pendingGets = make([]string, 0, pb.batchSize)
	pb.pendingDel = make([]string, 0, pb.batchSize)
	pb.mu.Unlock()

	if Client == nil {
		return
	}

	ctx := context.Background()
	pipe := Client.Pipeline()

	for key, set := range sets {
		pipe.Set(ctx, key, set.Value, set.TTL)
	}

	for _, key := range gets {
		pipe.Get(ctx, key)
	}

	for _, key := range dels {
		pipe.Del(ctx, key)
	}

	pipe.Exec(ctx)
}

type OptimizedRedisConfig struct {
	PoolSize            int
	MinIdleConns        int
	MaxIdleConns        int
	MaxRetries          int
	DialTimeout         time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	PoolTimeout         time.Duration
	ConnMaxLifetime     time.Duration
	ConnMaxIdleTime     time.Duration
	EnableHealthCheck   bool
	HealthCheckInterval time.Duration
	PreConnect          bool
}

var defaultOptimizedConfig = &OptimizedRedisConfig{
	PoolSize:            100,
	MinIdleConns:        20,
	MaxIdleConns:        50,
	MaxRetries:          3,
	DialTimeout:         5 * time.Second,
	ReadTimeout:         3 * time.Second,
	WriteTimeout:        3 * time.Second,
	PoolTimeout:         4 * time.Second,
	ConnMaxLifetime:     30 * time.Minute,
	ConnMaxIdleTime:    10 * time.Minute,
	EnableHealthCheck:  true,
	HealthCheckInterval: 30 * time.Second,
	PreConnect:          true,
}

func ConnectOptimizedRedis(cfg *OptimizedRedisConfig) error {
	if Client != nil {
		return nil
	}

	optCfg := defaultOptimizedConfig
	if cfg != nil {
		optCfg = cfg
	}

	Client = goredis.NewClient(&goredis.Options{
		Addr:            "localhost:6379",
		Password:        "",
		DB:              0,
		MaxRetries:      optCfg.MaxRetries,
		DialTimeout:     optCfg.DialTimeout,
		ReadTimeout:     optCfg.ReadTimeout,
		WriteTimeout:    optCfg.WriteTimeout,
		PoolSize:        optCfg.PoolSize,
		MinIdleConns:    optCfg.MinIdleConns,
		PoolTimeout:     optCfg.PoolTimeout,
		ConnMaxLifetime: optCfg.ConnMaxLifetime,
		ConnMaxIdleTime: optCfg.ConnMaxIdleTime,
	})

	if err := Client.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	go func() {
		ticker := time.NewTicker(optCfg.HealthCheckInterval)
		defer ticker.Stop()

		for range ticker.C {
			stats := Client.PoolStats()
			if stats.Timeouts > 10 {
				fmt.Printf("[REDIS_WARNING] High timeout count: %d\n", stats.Timeouts)
			}
			if stats.StaleConns > 5 {
				fmt.Printf("[REDIS_WARNING] High stale connection count: %d\n", stats.StaleConns)
			}
		}
	}()

	if optCfg.PreConnect {
		go func() {
			var wg sync.WaitGroup
			sem := make(chan struct{}, 5)

			for i := 0; i < optCfg.MinIdleConns; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					conn := Client.Conn()
					conn.Close()
				}()
			}

			wg.Wait()
			fmt.Printf("[REDIS] Pre-connected %d pool connections\n", optCfg.MinIdleConns)
		}()
	}

	return nil
}
