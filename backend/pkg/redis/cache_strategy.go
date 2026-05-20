package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheStrategy string

const (
	StrategyLRU    CacheStrategy = "lru"
	StrategyLFU    CacheStrategy = "lfu"
	StrategyFIFO   CacheStrategy = "fifo"
	StrategyTTL    CacheStrategy = "ttl"
	StrategyAdaptive CacheStrategy = "adaptive"
)

type CacheConfig struct {
	DefaultTTL     time.Duration
	MaxMemory      int64
	EvictionPolicy CacheStrategy
	EnableCompression bool
	SerializationType string
}

type LegacySerializer interface {
	Serialize(interface{}) ([]byte, error)
	Deserialize([]byte, interface{}) error
}

type legacyJSONSerializer struct{}

func (s *legacyJSONSerializer) Serialize(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (s *legacyJSONSerializer) Deserialize(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func newLegacyJSONSerializer() LegacySerializer {
	return &legacyJSONSerializer{}
}

type SmartCacheManager struct {
	client      *redis.Client
	config      *CacheConfig
	serializers map[string]LegacySerializer
	strategies  map[CacheStrategy]EvictionStrategy
	mu          sync.RWMutex
}

type EvictionStrategy interface {
	ShouldEvict(key string, stats *CacheStats) bool
	RecordAccess(key string)
}

type CacheStats struct {
	Hits       int64
	Misses     int64
	Keys       int64
	MemoryUsed int64
	Evictions  int64
	HitRate    float64
}

func NewSmartCacheManager(client *redis.Client, config *CacheConfig) *SmartCacheManager {
	manager := &SmartCacheManager{
		client:      client,
		config:      config,
		serializers: make(map[string]LegacySerializer),
		strategies:  make(map[CacheStrategy]EvictionStrategy),
	}

	manager.serializers["json"] = newLegacyJSONSerializer()

	manager.strategies[StrategyLRU] = &LRUStrategy{}
	manager.strategies[StrategyLFU] = &LFUStrategy{}
	manager.strategies[StrategyFIFO] = &FIFOStrategy{}
	manager.strategies[StrategyTTL] = &TTLStrategy{}
	manager.strategies[StrategyAdaptive] = &AdaptiveStrategy{}

	return manager
}

func (m *SmartCacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := m.serialize(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	if m.config.EnableCompression && len(data) > 1024 {
		data, err = m.compress(data)
		if err != nil {
			return fmt.Errorf("failed to compress value: %w", err)
		}
	}

	if ttl == 0 {
		ttl = m.config.DefaultTTL
	}

	return m.client.Set(ctx, key, data, ttl).Err()
}

func (m *SmartCacheManager) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := m.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	if m.config.EnableCompression {
		data, err = m.decompress(data)
		if err != nil {
			return fmt.Errorf("failed to decompress value: %w", err)
		}
	}

	return m.deserialize(data, dest)
}

func (m *SmartCacheManager) GetOrSet(ctx context.Context, key string, dest interface{}, ttl time.Duration, loader func() (interface{}, error)) error {
	err := m.Get(ctx, key, dest)
	if err == nil {
		return nil
	}

	if err != redis.Nil {
		return err
	}

	value, err := loader()
	if err != nil {
		return err
	}

	if err := m.Set(ctx, key, value, ttl); err != nil {
		return err
	}

	data, err := m.serialize(value)
	if err != nil {
		return err
	}

	return m.deserialize(data, dest)
}

func (m *SmartCacheManager) Delete(ctx context.Context, keys ...string) error {
	return m.client.Del(ctx, keys...).Err()
}

func (m *SmartCacheManager) Exists(ctx context.Context, keys ...string) (int64, error) {
	return m.client.Exists(ctx, keys...).Result()
}

func (m *SmartCacheManager) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return m.client.Expire(ctx, key, ttl).Err()
}

func (m *SmartCacheManager) TTL(ctx context.Context, key string) (time.Duration, error) {
	return m.client.TTL(ctx, key).Result()
}

func (m *SmartCacheManager) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	pipe := m.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for i, cmd := range cmds {
		data, err := cmd.Bytes()
		if err == nil {
			result[keys[i]] = data
		}
	}

	return result, nil
}

func (m *SmartCacheManager) SetMulti(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	pipe := m.client.Pipeline()

	for key, value := range items {
		data, err := m.serialize(value)
		if err != nil {
			continue
		}

		if m.config.EnableCompression && len(data) > 1024 {
			data, err = m.compress(data)
			if err != nil {
				continue
			}
		}

		if ttl == 0 {
			ttl = m.config.DefaultTTL
		}

		pipe.Set(ctx, key, data, ttl)
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (m *SmartCacheManager) DeleteMulti(ctx context.Context, pattern string) (int64, error) {
	iter := m.client.Scan(ctx, 0, pattern, 100).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return 0, err
	}

	if len(keys) == 0 {
		return 0, nil
	}

	return m.client.Del(ctx, keys...).Result()
}

func (m *SmartCacheManager) GetStats(ctx context.Context) (*CacheStats, error) {
	info, err := m.client.Info(ctx, "stats", "memory", "keyspace").Result()
	if err != nil {
		return nil, err
	}

	stats := &CacheStats{}

	for _, line := range splitLines(info) {
		if line == "" {
			continue
		}

		if !contains(line, ":") {
			continue
		}

		parts := splitKeyValue(line)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]

		switch key {
		case "keyspace_hits":
			fmt.Sscanf(value, "%d", &stats.Hits)
		case "keyspace_misses":
			fmt.Sscanf(value, "%d", &stats.Misses)
		case "evicted_keys":
			fmt.Sscanf(value, "%d", &stats.Evictions)
		}
	}

	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total) * 100
	}

	dbSize, err := m.client.DBSize(ctx).Result()
	if err == nil {
		stats.Keys = dbSize
	}

	return stats, nil
}

func (m *SmartCacheManager) serialize(value interface{}) ([]byte, error) {
	serializer, ok := m.serializers[m.config.SerializationType]
	if !ok {
		serializer = newLegacyJSONSerializer()
	}
	return serializer.Serialize(value)
}

func (m *SmartCacheManager) deserialize(data []byte, dest interface{}) error {
	serializer, ok := m.serializers[m.config.SerializationType]
	if !ok {
		serializer = newLegacyJSONSerializer()
	}
	return serializer.Deserialize(data, dest)
}

func (m *SmartCacheManager) compress(data []byte) ([]byte, error) {
	return data, nil
}

func (m *SmartCacheManager) decompress(data []byte) ([]byte, error) {
	return data, nil
}

type LRUStrategy struct {
	accessTime map[string]time.Time
	mu         sync.Mutex
}

func (s *LRUStrategy) ShouldEvict(key string, stats *CacheStats) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if stats.MemoryUsed > 0 && stats.Keys > 0 {
		return float64(stats.Keys)/float64(stats.MemoryUsed) < 0.8
	}
	return false
}

func (s *LRUStrategy) RecordAccess(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accessTime[key] = time.Now()
}

type LFUStrategy struct {
	frequency map[string]int64
	mu        sync.Mutex
}

func (s *LFUStrategy) ShouldEvict(key string, stats *CacheStats) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	freq, exists := s.frequency[key]
	if !exists {
		return false
	}

	avgFreq := float64(stats.Hits) / float64(stats.Keys)
	return float64(freq) < avgFreq*0.5
}

func (s *LFUStrategy) RecordAccess(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.frequency[key]++
}

type FIFOStrategy struct{}

func (s *FIFOStrategy) ShouldEvict(key string, stats *CacheStats) bool {
	return stats.Keys > 10000
}

func (s *FIFOStrategy) RecordAccess(key string) {}

type TTLStrategy struct{}

func (s *TTLStrategy) ShouldEvict(key string, stats *CacheStats) bool {
	return false
}

func (s *TTLStrategy) RecordAccess(key string) {}

type AdaptiveStrategy struct {
	*LRUStrategy
	*LFUStrategy
}

func (s *AdaptiveStrategy) ShouldEvict(key string, stats *CacheStats) bool {
	if stats.HitRate < 50 {
		return s.LRUStrategy.ShouldEvict(key, stats)
	}
	return s.LFUStrategy.ShouldEvict(key, stats)
}

func (s *AdaptiveStrategy) RecordAccess(key string) {
	s.LRUStrategy.RecordAccess(key)
	s.LFUStrategy.RecordAccess(key)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitKeyValue(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
