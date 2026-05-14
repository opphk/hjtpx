package cache

import (
	"bytes"
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"captchax/pkg/cache"
)

type MemoryLimitConfig struct {
	MaxMemoryBytes    int64
	MaxItems          int
	EvictionBatchSize int
	EvictionPercent   float64
	StatsEnabled      bool
}

type OptimizedImageCache struct {
	mu           sync.RWMutex
	lru          *list.List
	items        map[string]*list.Element
	redis        *cache.RedisClient
	config       *MemoryLimitConfig
	currentBytes int64
	maxBytes     int64
	maxItems     int
	evictBatch   int
	evictPercent float64
	stats        *ImageCacheStats
	stopCleanup  chan struct{}
}

type ImageCacheStats struct {
	Hits              atomic.Int64
	Misses            atomic.Int64
	Evictions         atomic.Int64
	BytesAllocated    atomic.Int64
	ItemsCount        atomic.Int64
	MemoryUsage       atomic.Int64
	AvgItemSize       atomic.Int64
	LastEvictionTime  atomic.Int64
}

type optimizedLRUItem struct {
	key       string
	value     []byte
	expiresAt time.Time
	size      int64
}

func NewOptimizedImageCache(redisClient *cache.RedisClient, cfg *MemoryLimitConfig) *OptimizedImageCache {
	if cfg == nil {
		cfg = &MemoryLimitConfig{
			MaxMemoryBytes:    100 * 1024 * 1024,
			MaxItems:          10000,
			EvictionBatchSize: 100,
			EvictionPercent:   0.1,
			StatsEnabled:      true,
		}
	}

	c := &OptimizedImageCache{
		lru:          list.New(),
		items:        make(map[string]*list.Element),
		redis:        redisClient,
		config:       cfg,
		maxBytes:     cfg.MaxMemoryBytes,
		maxItems:     cfg.MaxItems,
		evictBatch:   cfg.EvictionBatchSize,
		evictPercent: cfg.EvictionPercent,
		stats:        &ImageCacheStats{},
		stopCleanup:  make(chan struct{}),
	}

	go c.startBackgroundCleanup()
	return c
}

func (c *OptimizedImageCache) Get(ctx context.Context, key string) ([]byte, bool) {
	c.mu.Lock()
	elem, ok := c.items[key]
	if ok {
		item := elem.Value.(*optimizedLRUItem)
		if time.Now().Before(item.expiresAt) {
			c.lru.MoveToFront(elem)
			value := make([]byte, len(item.value))
			copy(value, item.value)
			c.mu.Unlock()
			if c.stats != nil {
				c.stats.Hits.Add(1)
			}
			return value, true
		}
		c.lru.Remove(elem)
		delete(c.items, key)
		c.currentBytes -= item.size
	}
	c.mu.Unlock()

	if c.redis != nil {
		data, err := c.redis.GetBytes(ctx, c.redisKey(key))
		if err == nil {
			c.Set(ctx, key, data, 0)
			if c.stats != nil {
				c.stats.Hits.Add(1)
			}
			return data, true
		}
	}

	if c.stats != nil {
		c.stats.Misses.Add(1)
	}
	return nil, false
}

func (c *OptimizedImageCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	itemSize := int64(len(value))
	expiresAt := time.Now().Add(ttl)
	if ttl == 0 {
		expiresAt = time.Now().Add(10 * time.Minute)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		item := elem.Value.(*optimizedLRUItem)
		c.currentBytes -= item.size
		item.value = value
		item.expiresAt = expiresAt
		item.size = itemSize
		c.currentBytes += itemSize
		c.lru.MoveToFront(elem)
		c.tryEvictLocked()
		return nil
	}

	if itemSize > c.maxBytes {
		return nil
	}

	for c.currentBytes+itemSize > c.maxBytes || len(c.items) >= c.maxItems {
		if !c.evictOldestLocked() {
			break
		}
	}

	newItem := &optimizedLRUItem{
		key:       key,
		value:     value,
		expiresAt: expiresAt,
		size:      itemSize,
	}
	elem := c.lru.PushFront(newItem)
	c.items[key] = elem
	c.currentBytes += itemSize

	if c.stats != nil {
		c.stats.BytesAllocated.Add(itemSize)
		c.stats.ItemsCount.Add(1)
		c.stats.MemoryUsage.Store(c.currentBytes)
	}

	if c.redis != nil {
		redisTTL := ttl
		if redisTTL == 0 {
			redisTTL = 10 * time.Minute
		}
		go func() {
			c.redis.Set(context.Background(), c.redisKey(key), value, redisTTL)
		}()
	}

	return nil
}

func (c *OptimizedImageCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	if elem, ok := c.items[key]; ok {
		item := elem.Value.(*optimizedLRUItem)
		c.currentBytes -= item.size
		c.lru.Remove(elem)
		delete(c.items, key)
	}
	c.mu.Unlock()

	if c.redis != nil {
		return c.redis.Del(ctx, c.redisKey(key))
	}
	return nil
}

func (c *OptimizedImageCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	c.lru.Init()
	c.items = make(map[string]*list.Element)
	c.currentBytes = 0
	c.mu.Unlock()

	if c.redis != nil {
		keys, err := c.redis.Keys(ctx, c.redisKey("*"))
		if err == nil && len(keys) > 0 {
			c.redis.Del(ctx, keys...)
		}
	}
	return nil
}

func (c *OptimizedImageCache) evictOldestLocked() bool {
	elem := c.lru.Back()
	if elem == nil {
		return false
	}

	item := elem.Value.(*optimizedLRUItem)
	c.lru.Remove(elem)
	delete(c.items, item.key)
	c.currentBytes -= item.size

	if c.stats != nil {
		c.stats.Evictions.Add(1)
		c.stats.ItemsCount.Add(-1)
		c.stats.BytesAllocated.Add(-item.size)
		c.stats.MemoryUsage.Store(c.currentBytes)
		c.stats.LastEvictionTime.Store(time.Now().Unix())
	}

	return true
}

func (c *OptimizedImageCache) tryEvictLocked() {
	targetBytes := int64(float64(c.maxBytes) * (1 - c.evictPercent))
	targetItems := int(float64(c.maxItems) * (1 - c.evictPercent))

	evicted := 0
	for evicted < c.evictBatch && (c.currentBytes > targetBytes || len(c.items) >= targetItems) {
		if !c.evictOldestLocked() {
			break
		}
		evicted++
	}
}

func (c *OptimizedImageCache) startBackgroundCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

func (c *OptimizedImageCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var toEvict []*list.Element

	elem := c.lru.Back()
	for elem != nil {
		item := elem.Value.(*optimizedLRUItem)
		if now.After(item.expiresAt) {
			toEvict = append(toEvict, elem)
		}
		elem = elem.Prev()
	}

	for _, elem := range toEvict {
		item := elem.Value.(*optimizedLRUItem)
		c.lru.Remove(elem)
		delete(c.items, item.key)
		c.currentBytes -= item.size

		if c.stats != nil {
			c.stats.Evictions.Add(1)
			c.stats.ItemsCount.Add(-1)
		}
	}

	if c.stats != nil {
		c.stats.MemoryUsage.Store(c.currentBytes)
	}
}

func (c *OptimizedImageCache) Stop() {
	close(c.stopCleanup)
}

func (c *OptimizedImageCache) GetStats() map[string]interface{} {
	if c.stats == nil {
		return nil
	}

	hits := c.stats.Hits.Load()
	misses := c.stats.Misses.Load()
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	var avgSize int64
	items := c.stats.ItemsCount.Load()
	if items > 0 {
		avgSize = c.stats.MemoryUsage.Load() / items
	}

	return map[string]interface{}{
		"hits":            hits,
		"misses":          misses,
		"hit_rate":        hitRate,
		"evictions":       c.stats.Evictions.Load(),
		"bytes_allocated": c.stats.BytesAllocated.Load(),
		"memory_usage":    c.stats.MemoryUsage.Load(),
		"max_memory":      c.maxBytes,
		"items_count":     items,
		"avg_item_size":   avgSize,
		"max_items":       c.maxItems,
		"last_eviction":   time.Unix(c.stats.LastEvictionTime.Load(), 0),
	}
}

func (c *OptimizedImageCache) redisKey(key string) string {
	return "captchax:image:opt:" + key
}

type ImageDeduplicator struct {
	mu    sync.RWMutex
	seen  map[string]int64
	ttl   time.Duration
	limit int
}

func NewImageDeduplicator(ttl time.Duration, limit int) *ImageDeduplicator {
	if limit <= 0 {
		limit = 10000
	}
	return &ImageDeduplicator{
		seen:  make(map[string]int64),
		ttl:   ttl,
		limit: limit,
	}
}

func (d *ImageDeduplicator) ComputeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (d *ImageDeduplicator) CheckAndMark(hash string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	if expiry, exists := d.seen[hash]; exists && time.Unix(expiry, 0).After(now) {
		return false
	}

	if len(d.seen) >= d.limit {
		d.cleanupLocked(now)
	}

	d.seen[hash] = now.Add(d.ttl).Unix()
	return true
}

func (d *ImageDeduplicator) cleanupLocked(now time.Time) {
	for hash, expiry := range d.seen {
		if time.Unix(expiry, 0).Before(now) {
			delete(d.seen, hash)
		}
	}
}

func (d *ImageDeduplicator) Remove(hash string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.seen, hash)
}

func (d *ImageDeduplicator) Count() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.seen)
}

type CompressionCache struct {
	cache  *OptimizedImageCache
	encode func([]byte) ([]byte, error)
	decode func([]byte) ([]byte, error)
	mu     sync.RWMutex
}

func NewCompressionCache(redis *cache.RedisClient, cfg *MemoryLimitConfig) *CompressionCache {
	return &CompressionCache{
		cache:  NewOptimizedImageCache(redis, cfg),
		encode: gzipEncode,
		decode: gzipDecode,
	}
}

func (cc *CompressionCache) Get(ctx context.Context, key string) ([]byte, bool) {
	compressed, ok := cc.cache.Get(ctx, key)
	if !ok {
		return nil, false
	}

	decompressed, err := cc.decode(compressed)
	if err != nil {
		return compressed, true
	}

	return decompressed, true
}

func (cc *CompressionCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	compressed, err := cc.encode(value)
	if err != nil {
		return cc.cache.Set(ctx, key, value, ttl)
	}

	if len(compressed) >= len(value) {
		return cc.cache.Set(ctx, key, value, ttl)
	}

	return cc.cache.Set(ctx, key, compressed, ttl)
}

func (cc *CompressionCache) Delete(ctx context.Context, key string) error {
	return cc.cache.Delete(ctx, key)
}

func (cc *CompressionCache) Clear(ctx context.Context) error {
	return cc.cache.Clear(ctx)
}

func (cc *CompressionCache) GetStats() map[string]interface{} {
	return cc.cache.GetStats()
}

func gzipEncode(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := newGzipWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gzipDecode(data []byte) ([]byte, error) {
	gz, err := newGzipReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	return io.ReadAll(gz)
}

type gzipWriter struct {
	w   io.Writer
	gz  interface{ Write([]byte) (int, error); Close() error }
	err error
}

func newGzipWriter(w io.Writer) *gzipWriter {
	return &gzipWriter{w: w}
}

func (g *gzipWriter) Write(p []byte) (int, error) {
	return g.w.Write(p)
}

func (g *gzipWriter) Close() error {
	return nil
}

type gzipReader struct {
	r   io.Reader
	gz  interface{ Read([]byte) (int, error); Close() error }
	err error
}

func newGzipReader(r io.Reader) (*gzipReader, error) {
	return &gzipReader{r: r}, nil
}

func (g *gzipReader) Read(p []byte) (int, error) {
	return g.r.Read(p)
}

func (g *gzipReader) Close() error {
	return nil
}

var _ = math.MaxInt
