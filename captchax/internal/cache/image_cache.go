package cache

import (
	"container/list"
	"context"
	"sync"
	"time"

	"captchax/pkg/cache"
)

type ImageCacheConfig struct {
	MemoryMaxItems int
	MemoryTTL      time.Duration
	RedisTTL       time.Duration
	PreheatCount   int
}

type LRUItem struct {
	Key       string
	Value     []byte
	ExpiresAt time.Time
}

type ImageCache struct {
	mu         sync.RWMutex
	memory     *list.List
	items      map[string]*list.Element
	redis      *cache.RedisClient
	config     *ImageCacheConfig
	maxItems   int
	memoryTTL  time.Duration
	redisTTL   time.Duration
}

type lruEntry struct {
	key   string
	value []byte
}

func NewImageCache(redisClient *cache.RedisClient, cfg *ImageCacheConfig) *ImageCache {
	if cfg == nil {
		cfg = &ImageCacheConfig{
			MemoryMaxItems: 1000,
			MemoryTTL:      5 * time.Minute,
			RedisTTL:       10 * time.Minute,
			PreheatCount:   100,
		}
	}

	return &ImageCache{
		memory:    list.New(),
		items:     make(map[string]*list.Element),
		redis:     redisClient,
		config:    cfg,
		maxItems:  cfg.MemoryMaxItems,
		memoryTTL: cfg.MemoryTTL,
		redisTTL:  cfg.RedisTTL,
	}
}

func (c *ImageCache) redisKey(key string) string {
	return "captchax:image:" + key
}

func (c *ImageCache) Get(ctx context.Context, key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*lruEntry)
		c.memory.MoveToFront(elem)
		return entry.value, true
	}

	c.mu.Unlock()
	if c.redis != nil {
		data, err := c.redis.GetBytes(ctx, c.redisKey(key))
		if err == nil {
			c.mu.Lock()
			c.Set(key, data)
			c.mu.Unlock()
			return data, true
		}
	}
	c.mu.Lock()
	return nil, false
}

func (c *ImageCache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*lruEntry)
		entry.value = value
		c.memory.MoveToFront(elem)
		return
	}

	entry := &lruEntry{key: key, value: value}
	elem := c.memory.PushFront(entry)
	c.items[key] = elem

	if c.memory.Len() > c.maxItems {
		c.evictOldest()
	}
}

func (c *ImageCache) SetWithTTL(ctx context.Context, key string, value []byte) error {
	c.Set(key, value)

	if c.redis != nil {
		return c.redis.Set(ctx, c.redisKey(key), value, c.redisTTL)
	}
	return nil
}

func (c *ImageCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.memory.Remove(elem)
		delete(c.items, key)
	}
}

func (c *ImageCache) DeleteContext(ctx context.Context, key string) error {
	c.Delete(key)

	if c.redis != nil {
		return c.redis.Del(ctx, c.redisKey(key))
	}
	return nil
}

func (c *ImageCache) evictOldest() {
	elem := c.memory.Back()
	if elem != nil {
		entry := elem.Value.(*lruEntry)
		delete(c.items, entry.key)
		c.memory.Remove(elem)
	}
}

func (c *ImageCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.memory.Init()
	c.items = make(map[string]*list.Element)
}

func (c *ImageCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.memory.Len()
}

func (c *ImageCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return map[string]interface{}{
		"memory_items": c.memory.Len(),
		"max_items":    c.maxItems,
		"memory_ttl":   c.memoryTTL.String(),
		"redis_ttl":    c.redisTTL.String(),
	}
}

func (c *ImageCache) Preheat(ctx context.Context, loader func(keys []string) map[string][]byte) error {
	if c.redis == nil {
		return nil
	}

	pattern := c.redisKey("*")
	keys, err := c.redis.Keys(ctx, pattern)
	if err != nil {
		return err
	}

	count := c.config.PreheatCount
	if count > len(keys) {
		count = len(keys)
	}

	if count == 0 {
		return nil
	}

	preheatKeys := make([]string, count)
	for i := 0; i < count; i++ {
		preheatKeys[i] = keys[i]
	}

	data := loader(preheatKeys)

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, k := range preheatKeys {
		if val, ok := data[k]; ok {
			entry := &lruEntry{key: k, value: val}
			elem := c.memory.PushFront(entry)
			c.items[k] = elem
		}
	}

	for c.memory.Len() > c.maxItems {
		c.evictOldest()
	}

	return nil
}

type ImageCachePool struct {
	pool sync.Pool
}

func NewImageCachePool(factory func() *ImageCache) *ImageCachePool {
	return &ImageCachePool{
		pool: sync.Pool{
			New: func() interface{} {
				return factory()
			},
		},
	}
}

func (p *ImageCachePool) Get() *ImageCache {
	return p.pool.Get().(*ImageCache)
}

func (p *ImageCachePool) Put(c *ImageCache) {
	c.Clear()
	p.pool.Put(c)
}

type ByteSlicePool struct {
	pool sync.Pool
}

func NewByteSlicePool(size int) *ByteSlicePool {
	return &ByteSlicePool{
		pool: sync.Pool{
			New: func() interface{} {
				b := make([]byte, size)
				return &b
			},
		},
	}
}

func (p *ByteSlicePool) Get() []byte {
	return *(p.pool.Get().(*[]byte))
}

func (p *ByteSlicePool) Put(b []byte) {
	p.pool.Put(&b)
}
