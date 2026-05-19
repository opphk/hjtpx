package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type ConsistencyMode int

const (
	ConsistencyModeStrong ConsistencyMode = iota
	ConsistencyModeEventual
	ConsistencyModeCausal
	ConsistencyModeReadYourWrites
)

type CacheConsistencyConfig struct {
	Mode                ConsistencyMode
	SyncWrites          bool
	InvalidationDelay   time.Duration
	StalenessWindow     time.Duration
	WriteBehindEnabled  bool
	WriteBehindInterval time.Duration
}

var defaultConsistencyConfig = &CacheConsistencyConfig{
	Mode:                ConsistencyModeEventual,
	SyncWrites:          false,
	InvalidationDelay:   100 * time.Millisecond,
	StalenessWindow:     5 * time.Second,
	WriteBehindEnabled:  true,
	WriteBehindInterval: 1 * time.Second,
}

type CacheConsistencyManager struct {
	mu          sync.RWMutex
	config      *CacheConsistencyConfig
	versions    map[string]int64
	dependencies map[string][]string
	pendingOps  chan *ConsistencyOperation
	stats       *ConsistencyStats
}

type ConsistencyOperation struct {
	Type      string
	Key       string
	Value     interface{}
	Version   int64
	Timestamp time.Time
	Callback  chan error
}

type ConsistencyStats struct {
	TotalOperations     atomic.Int64
	SuccessfulOps       atomic.Int64
	FailedOps           atomic.Int64
	Invalidations       atomic.Int64
	SyncDelays          atomic.Int64
	VersionMismatches    atomic.Int64
}

var (
	globalConsistencyManager *CacheConsistencyManager
	consistencyManagerOnce   sync.Once
)

func InitConsistencyManager(config *CacheConsistencyConfig) {
	consistencyManagerOnce.Do(func() {
		if config == nil {
			config = defaultConsistencyConfig
		}
		globalConsistencyManager = &CacheConsistencyManager{
			config:       config,
			versions:     make(map[string]int64),
			dependencies: make(map[string][]string),
			pendingOps:   make(chan *ConsistencyOperation, 10000),
			stats:        &ConsistencyStats{},
		}
		go globalConsistencyManager.processOperations()
	})
}

func GetConsistencyManager() *CacheConsistencyManager {
	if globalConsistencyManager == nil {
		InitConsistencyManager(nil)
	}
	return globalConsistencyManager
}

func (c *CacheConsistencyManager) processOperations() {
	ticker := time.NewTicker(c.config.WriteBehindInterval)
	defer ticker.Stop()

	var batch []ConsistencyOperation

	for {
		select {
		case op := <-c.pendingOps:
			batch = append(batch, *op)
			if len(batch) >= 100 {
				c.executeBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				c.executeBatch(batch)
				batch = batch[:0]
			}

		case <-time.After(c.config.WriteBehindInterval):
			if len(batch) > 0 {
				c.executeBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

func (c *CacheConsistencyManager) executeBatch(ops []ConsistencyOperation) {
	if c.config.SyncWrites {
		for i := range ops {
			c.executeOperation(&ops[i])
		}
	}
}

func (c *CacheConsistencyManager) executeOperation(op *ConsistencyOperation) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch op.Type {
	case "set":
		c.stats.SuccessfulOps.Add(1)
	case "delete":
		c.stats.Invalidations.Add(1)
	case "invalidate":
		c.invalidateKeys(op.Key)
	}

	c.stats.TotalOperations.Add(1)
}

func (c *CacheConsistencyManager) invalidateKeys(pattern string) {
	client := GetClient()
	if client == nil {
		return
	}

	ctx := context.Background()

	var cursor uint64
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern+"*", 100).Result()
		if err != nil {
			break
		}

		if len(keys) > 0 {
			client.Del(ctx, keys...)
			c.stats.Invalidations.Add(int64(len(keys)))
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}

func (c *CacheConsistencyManager) GetVersion(key string) int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.versions[key]
}

func (c *CacheConsistencyManager) IncrementVersion(key string) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	version := c.versions[key] + 1
	c.versions[key] = version
	return version
}

func (c *CacheConsistencyManager) SetVersion(key string, version int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.versions[key] = version
}

func (c *CacheConsistencyManager) AddDependency(key, dependency string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	deps := c.dependencies[key]
	c.dependencies[key] = append(deps, dependency)
}

func (c *CacheConsistencyManager) RemoveDependency(key, dependency string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	deps := c.dependencies[key]
	for i, d := range deps {
		if d == dependency {
			c.dependencies[key] = append(deps[:i], deps[i+1:]...)
			break
		}
	}
}

func (c *CacheConsistencyManager) InvalidateDependencies(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for depKey, deps := range c.dependencies {
		for _, dep := range deps {
			if dep == key {
				delete(c.versions, depKey)
				c.invalidateKeys(depKey)
			}
		}
	}
}

func (c *CacheConsistencyManager) SetWithVersion(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	version := c.IncrementVersion(key)

	if c.config.Mode == ConsistencyModeStrong || c.config.SyncWrites {
		return c.setSync(ctx, key, value, ttl, version)
	}

	c.pendingOps <- &ConsistencyOperation{
		Type:      "set",
		Key:       key,
		Value:     value,
		Version:   version,
		Timestamp: time.Now(),
	}

	return nil
}

func (c *CacheConsistencyManager) setSync(ctx context.Context, key string, value interface{}, ttl time.Duration, version int64) error {
	client := GetClient()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	data := map[string]interface{}{
		"value":   value,
		"version": version,
		"updated": time.Now().Unix(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return client.Set(ctx, key, jsonData, ttl).Err()
}

func (c *CacheConsistencyManager) GetWithVersion(ctx context.Context, key string) (interface{}, int64, error) {
	client := GetClient()
	if client == nil {
		return nil, 0, fmt.Errorf("redis client not initialized")
	}

	result, err := client.Get(ctx, key).Result()
	if err != nil {
		return nil, 0, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		return result, 0, nil
	}

	version := int64(0)
	if v, ok := data["version"].(float64); ok {
		version = int64(v)
	}

	value := data["value"]

	c.mu.RLock()
	expectedVersion := c.versions[key]
	c.mu.RUnlock()

	if version < expectedVersion {
		c.stats.VersionMismatches.Add(1)
	}

	return value, version, nil
}

func (c *CacheConsistencyManager) DeleteWithDependencies(ctx context.Context, key string) error {
	c.mu.Lock()
	c.IncrementVersion(key)
	c.InvalidateDependencies(key)
	c.mu.Unlock()

	if c.config.Mode == ConsistencyModeStrong || c.config.SyncWrites {
		return c.deleteSync(ctx, key)
	}

	c.pendingOps <- &ConsistencyOperation{
		Type:      "delete",
		Key:       key,
		Timestamp: time.Now(),
	}

	return nil
}

func (c *CacheConsistencyManager) deleteSync(ctx context.Context, key string) error {
	client := GetClient()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	return client.Del(ctx, key).Err()
}

func (c *CacheConsistencyManager) InvalidatePattern(ctx context.Context, pattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.invalidateKeys(pattern)

	if c.config.Mode == ConsistencyModeStrong {
		client := GetClient()
		if client == nil {
			return fmt.Errorf("redis client not initialized")
		}

		var cursor uint64
		for {
			keys, nextCursor, err := client.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				return err
			}

			if len(keys) > 0 {
				if err := client.Del(ctx, keys...).Err(); err != nil {
					return err
				}
			}

			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}
	}

	c.stats.Invalidations.Add(1)
	return nil
}

func (c *CacheConsistencyManager) CheckConsistency(ctx context.Context, keys []string) (map[string]bool, error) {
	client := GetClient()
	if client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	results := make(map[string]bool)

	for _, key := range keys {
		result, err := client.Get(ctx, key).Result()
		if err != nil {
			results[key] = false
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result), &data); err != nil {
			results[key] = true
			continue
		}

		version := int64(0)
		if v, ok := data["version"].(float64); ok {
			version = int64(v)
		}

		c.mu.RLock()
		expectedVersion := c.versions[key]
		c.mu.RUnlock()

		results[key] = version >= expectedVersion
	}

	return results, nil
}

func (c *CacheConsistencyManager) GetStats() *ConsistencyStats {
	return &ConsistencyStats{
		TotalOperations:    c.stats.TotalOperations,
		SuccessfulOps:     c.stats.SuccessfulOps,
		FailedOps:        c.stats.FailedOps,
		Invalidations:     c.stats.Invalidations,
		SyncDelays:       c.stats.SyncDelays,
		VersionMismatches: c.stats.VersionMismatches,
	}
}

func (c *CacheConsistencyManager) WaitForConsistency(ctx context.Context, key string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for consistency")
			}

			c.mu.RLock()
			version := c.versions[key]
			c.mu.RUnlock()

			client := GetClient()
			if client == nil {
				continue
			}

			result, err := client.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			var data map[string]interface{}
			if err := json.Unmarshal([]byte(result), &data); err != nil {
				continue
			}

			if v, ok := data["version"].(float64); ok && int64(v) >= version {
				return nil
			}
		}
	}
}

type WriteThroughCache struct {
	*CacheConsistencyManager
	backendDB interface{}
}

func NewWriteThroughCache(config *CacheConsistencyConfig, db interface{}) *WriteThroughCache {
	InitConsistencyManager(config)
	return &WriteThroughCache{
		CacheConsistencyManager: GetConsistencyManager(),
		backendDB:              db,
	}
}

func (w *WriteThroughCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if err := w.CacheConsistencyManager.SetWithVersion(ctx, key, value, ttl); err != nil {
		return err
	}

	return nil
}

func (w *WriteThroughCache) Get(ctx context.Context, key string) (interface{}, error) {
	value, _, err := w.CacheConsistencyManager.GetWithVersion(ctx, key)
	return value, err
}

func (w *WriteThroughCache) Delete(ctx context.Context, key string) error {
	return w.CacheConsistencyManager.DeleteWithDependencies(ctx, key)
}

type WriteBehindCache struct {
	*CacheConsistencyManager
	queue chan *CacheEntry
}

type CacheEntry struct {
	Key    string
	Value  interface{}
	TTL    time.Duration
	Time   time.Time
}

func NewWriteBehindCache(config *CacheConsistencyConfig) *WriteBehindCache {
	InitConsistencyManager(config)
	wb := &WriteBehindCache{
		CacheConsistencyManager: GetConsistencyManager(),
		queue: make(chan *CacheEntry, 10000),
	}
	go wb.processQueue()
	return wb
}

func (w *WriteBehindCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	w.queue <- &CacheEntry{
		Key:   key,
		Value: value,
		TTL:   ttl,
		Time:  time.Now(),
	}

	return nil
}

func (w *WriteBehindCache) processQueue() {
	batch := make([]*CacheEntry, 0, 100)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case entry := <-w.queue:
			batch = append(batch, entry)
			if len(batch) >= 100 {
				w.flushBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				w.flushBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

func (w *WriteBehindCache) flushBatch(entries []*CacheEntry) {
	ctx := context.Background()
	client := GetClient()
	if client == nil {
		return
	}

	for _, entry := range entries {
		w.CacheConsistencyManager.SetWithVersion(ctx, entry.Key, entry.Value, entry.TTL)
	}
}

func (w *WriteBehindCache) ForceFlush() {
	ctx := context.Background()
	client := GetClient()
	if client == nil {
		return
	}

	for {
		select {
		case entry := <-w.queue:
			w.CacheConsistencyManager.SetWithVersion(ctx, entry.Key, entry.Value, entry.TTL)
		default:
			return
		}
	}
}
