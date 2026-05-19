
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

type EdgeComputingService struct {
	mu            sync.RWMutex
	cache         *EdgeCache
	nodeManager   *EdgeNodeManager
	cacheStrategy *CacheStrategy
	stats         *EdgeComputingStats
	initialized   bool
}

type EdgeCache struct {
	mu             sync.RWMutex
	items          map[string]*CacheItem
	maxSize        int
	evictionPolicy string
}

type CacheItem struct {
	Key         string
	Value       []byte
	Expiration  time.Time
	AccessCount int64
	LastAccess  time.Time
	Size        int
}

type CacheStrategy struct {
	mu         sync.RWMutex
	defaultTTL time.Duration
	maxSize    int
	strategies map[string]string
}

type EdgeComputingStats struct {
	TotalRequests  int64         `json:"total_requests"`
	CacheHits      int64         `json:"cache_hits"`
	CacheMisses    int64         `json:"cache_misses"`
	CacheHitRate   float64       `json:"cache_hit_rate"`
	ActiveTasks    int           `json:"active_tasks"`
	CompletedTasks int64         `json:"completed_tasks"`
	AvgLatency     time.Duration `json:"avg_latency"`
	LastUpdate     time.Time     `json:"last_update"`
}

type EdgeComputeRequest struct {
	TaskID    string                 `json:"task_id"`
	TaskType  string                 `json:"task_type"`
	InputData map[string]interface{} `json:"input_data"`
	CacheKey  string                 `json:"cache_key,omitempty"`
	UseCache  bool                   `json:"use_cache"`
	TTL       time.Duration          `json:"ttl,omitempty"`
}

type EdgeComputeResponse struct {
	Success   bool                   `json:"success"`
	TaskID    string                 `json:"task_id"`
	Result    map[string]interface{} `json:"result,omitempty"`
	FromCache bool                   `json:"from_cache"`
	Latency   time.Duration          `json:"latency"`
	Error     string                 `json:"error,omitempty"`
}

type CacheInvalidationRequest struct {
	Keys          []string `json:"keys"`
	InvalidateAll bool     `json:"invalidate_all"`
}

type CacheStatsResponse struct {
	CacheSize      int           `json:"cache_size"`
	MaxSize        int           `json:"max_size"`
	HitRate        float64       `json:"hit_rate"`
	TotalItems     int           `json:"total_items"`
	EvictionPolicy string        `json:"eviction_policy"`
}

func NewEdgeComputingService(nodeManager *EdgeNodeManager) *EdgeComputingService {
	return &EdgeComputingService{
		cache:         NewEdgeCache(10000, "lru"),
		nodeManager:   nodeManager,
		cacheStrategy: NewCacheStrategy(),
		stats:         &EdgeComputingStats{},
	}
}

func NewEdgeCache(maxSize int, evictionPolicy string) *EdgeCache {
	return &EdgeCache{
		items:          make(map[string]*CacheItem),
		maxSize:        maxSize,
		evictionPolicy: evictionPolicy,
	}
}

func NewCacheStrategy() *CacheStrategy {
	return &CacheStrategy{
		defaultTTL: 1 * time.Hour,
		maxSize:    100 * 1024 * 1024,
		strategies: make(map[string]string),
	}
}

func (s *EdgeComputingService) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	go s.cleanCache(ctx)
	s.initialized = true
	log.Println("[EdgeComputingService] Initialized successfully")
	return nil
}

func (s *EdgeComputingService) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return nil
	}

	s.initialized = false
	log.Println("[EdgeComputingService] Shutdown complete")
	return nil
}

func (s *EdgeComputingService) ExecuteTask(ctx context.Context, req *EdgeComputeRequest) (*EdgeComputeResponse, error) {
	s.stats.TotalRequests++
	start := time.Now()

	if req.UseCache && req.CacheKey != "" {
		if cached, found := s.cache.Get(req.CacheKey); found {
			s.stats.CacheHits++
			s.updateStats()

			var result map[string]interface{}
			json.Unmarshal(cached.Value, &result)

			return &EdgeComputeResponse{
				Success:   true,
				TaskID:    req.TaskID,
				Result:    result,
				FromCache: true,
				Latency:   time.Since(start),
			}, nil
		}
		s.stats.CacheMisses++
	}

	result, err := s.processTask(ctx, req)
	if err != nil {
		s.updateStats()
		return &EdgeComputeResponse{
			Success: false,
			TaskID:  req.TaskID,
			Error:   err.Error(),
			Latency: time.Since(start),
		}, err
	}

	if req.UseCache && req.CacheKey != "" {
		value, _ := json.Marshal(result)
		ttl := req.TTL
		if ttl == 0 {
			ttl = s.cacheStrategy.defaultTTL
		}
		s.cache.Set(req.CacheKey, value, ttl)
	}

	s.stats.CompletedTasks++
	s.updateStats()

	return &EdgeComputeResponse{
		Success:   true,
		TaskID:    req.TaskID,
		Result:    result,
		FromCache: false,
		Latency:   time.Since(start),
	}, nil
}

func (s *EdgeComputingService) processTask(ctx context.Context, req *EdgeComputeRequest) (map[string]interface{}, error) {
	s.stats.ActiveTasks++
	defer func() {
		s.stats.ActiveTasks--
	}()

	node, err := s.nodeManager.SelectNode(ctx, req.TaskType)
	if err != nil {
		return s.processLocally(ctx, req)
	}

	result, err := s.processOnNode(ctx, node, req)
	if err != nil {
		return s.processLocally(ctx, req)
	}

	return result, nil
}

func (s *EdgeComputingService) processLocally(ctx context.Context, req *EdgeComputeRequest) (map[string]interface{}, error) {
	time.Sleep(10 * time.Millisecond)
	return map[string]interface{}{
		"processed": true,
		"method":    "local",
		"task_type": req.TaskType,
		"timestamp": time.Now(),
		"data":      req.InputData,
	}, nil
}

func (s *EdgeComputingService) processOnNode(ctx context.Context, node *EdgeNode, req *EdgeComputeRequest) (map[string]interface{}, error) {
	return map[string]interface{}{
		"processed": true,
		"method":    "edge",
		"node_id":   node.ID,
		"node_name": node.Name,
		"region":    node.Region,
		"task_type": req.TaskType,
		"timestamp": time.Now(),
		"data":      req.InputData,
	}, nil
}

func (s *EdgeComputingService) GetCache(key string) ([]byte, bool) {
	item, found := s.cache.Get(key)
	if !found {
		return nil, false
	}
	return item.Value, true
}

func (s *EdgeComputingService) SetCache(key string, value []byte, ttl time.Duration) {
	s.cache.Set(key, value, ttl)
}

func (s *EdgeComputingService) InvalidateCache(req *CacheInvalidationRequest) error {
	if req.InvalidateAll {
		s.cache.Clear()
	} else {
		for _, key := range req.Keys {
			s.cache.Delete(key)
		}
	}
	return nil
}

func (s *EdgeComputingService) GetCacheStats() *CacheStatsResponse {
	s.cache.mu.RLock()
	defer s.cache.mu.RUnlock()

	s.mu.RLock()
	defer s.mu.RUnlock()

	hitRate := 0.0
	if s.stats.TotalRequests > 0 {
		hitRate = float64(s.stats.CacheHits) / float64(s.stats.TotalRequests)
	}

	return &CacheStatsResponse{
		CacheSize:      len(s.cache.items),
		MaxSize:        s.cache.maxSize,
		HitRate:        hitRate,
		TotalItems:     len(s.cache.items),
		EvictionPolicy: s.cache.evictionPolicy,
	}
}

func (s *EdgeComputingService) GetStats() *EdgeComputingStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

func (s *EdgeComputingService) updateStats() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats.TotalRequests > 0 {
		s.stats.CacheHitRate = float64(s.stats.CacheHits) / float64(s.stats.TotalRequests)
	}
	s.stats.LastUpdate = time.Now()
}

func (s *EdgeComputingService) cleanCache(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cache.cleanExpired()
		}
	}
}

func (c *EdgeCache) Get(key string) (*CacheItem, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	if item.Expiration.Before(time.Now()) {
		return nil, false
	}

	item.AccessCount++
	item.LastAccess = time.Now()
	return item, true
}

func (c *EdgeCache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for len(c.items) >= c.maxSize {
		c.evict()
	}

	c.items[key] = &CacheItem{
		Key:         key,
		Value:       value,
		Expiration:  time.Now().Add(ttl),
		AccessCount: 1,
		LastAccess:  time.Now(),
		Size:        len(value),
	}
}

func (c *EdgeCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func (c *EdgeCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*CacheItem)
}

func (c *EdgeCache) cleanExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if item.Expiration.Before(now) {
			delete(c.items, key)
		}
	}
}

func (c *EdgeCache) evict() {
	switch c.evictionPolicy {
	case "lru":
		c.evictLRU()
	case "lfu":
		c.evictLFU()
	case "fifo":
		c.evictFIFO()
	default:
		c.evictLRU()
	}
}

func (c *EdgeCache) evictLRU() {
	var oldestKey string
	oldestTime := time.Now()

	for key, item := range c.items {
		if item.LastAccess.Before(oldestTime) {
			oldestTime = item.LastAccess
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

func (c *EdgeCache) evictLFU() {
	var leastKey string
	leastCount := int64(^uint64(0) >> 1)

	for key, item := range c.items {
		if item.AccessCount < leastCount {
			leastCount = item.AccessCount
			leastKey = key
		}
	}

	if leastKey != "" {
		delete(c.items, leastKey)
	}
}

func (c *EdgeCache) evictFIFO() {
	var oldestKey string
	oldestExp := time.Now()

	for key, item := range c.items {
		if item.Expiration.Before(oldestExp) {
			oldestExp = item.Expiration
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

func (cs *CacheStrategy) SetTTL(taskType string, ttl time.Duration) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.strategies[taskType] = fmt.Sprintf("%v", ttl)
}

func (cs *CacheStrategy) GetTTL(taskType string) time.Duration {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if ttlStr, exists := cs.strategies[taskType]; exists {
		ttl, _ := time.ParseDuration(ttlStr)
		return ttl
	}
	return cs.defaultTTL
}

func (cs *CacheStrategy) SetDefaultTTL(ttl time.Duration) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.defaultTTL = ttl
}
