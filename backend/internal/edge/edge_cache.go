package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

type SyncStrategy string

const (
	SyncStrategyEager     SyncStrategy = "eager"
	SyncStrategyLazy      SyncStrategy = "lazy"
	SyncStrategyWriteThrough SyncStrategy = "write_through"
	SyncStrategyWriteBehind  SyncStrategy = "write_behind"
	SyncStrategyEventual    SyncStrategy = "eventual"
)

type CacheSyncEntry struct {
	Key         string        `json:"key"`
	Value       interface{}   `json:"value"`
	TTL         time.Duration `json:"ttl"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Version     int64         `json:"version"`
	Checksum    string        `json:"checksum"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
	NodeID      string        `json:"node_id,omitempty"`
	Region      Region        `json:"region,omitempty"`
	Compressed  bool          `json:"compressed"`
	Expired     bool          `json:"expired"`
}

type CacheSyncMessage struct {
	Type      string      `json:"type"`
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	TTL       int64       `json:"ttl"`
	Version   int64       `json:"version"`
	Checksum  string      `json:"checksum"`
	Timestamp time.Time   `json:"timestamp"`
	NodeID    string      `json:"node_id"`
	Region    Region      `json:"region"`
	Tags      []string    `json:"tags"`
}

type CacheSyncConfig struct {
	Strategy          SyncStrategy   `json:"strategy"`
	BatchSize         int             `json:"batch_size"`
	SyncInterval      time.Duration   `json:"sync_interval"`
	MaxQueueSize      int             `json:"max_queue_size"`
	CompressionEnabled bool            `json:"compression_enabled"`
	EncryptionEnabled bool            `json:"encryption_enabled"`
	ConflictResolution string         `json:"conflict_resolution"`
	RetryAttempts     int             `json:"retry_attempts"`
	RetryDelay        time.Duration   `json:"retry_delay"`
}

type EdgeCacheSync struct {
	cache         map[string]*CacheSyncEntry
	nodeManager   *EdgeNodeManager
	redisClient   *redis.Client
	config        *CacheSyncConfig
	syncQueue     chan *CacheSyncMessage
	mu            sync.RWMutex
	nodeID        string
	region        Region
	version       int64
	metrics       *CacheSyncMetrics
	eventHandlers map[string][]func(*CacheSyncMessage)
	handlerMu     sync.RWMutex
}

type CacheSyncMetrics struct {
	TotalSyncs       int64                  `json:"total_syncs"`
	SuccessfulSyncs  int64                   `json:"successful_syncs"`
	FailedSyncs      int64                   `json:"failed_syncs"`
	BytesTransferred int64                   `json:"bytes_transferred"`
	SyncLatencyMs    float64                 `json:"sync_latency_ms"`
	QueueSize        int                     `json:"queue_size"`
	RegionMetrics    map[Region]*RegionSyncMetric `json:"region_metrics"`
	mu               sync.RWMutex
}

type RegionSyncMetric struct {
	Region          Region `json:"region"`
	SyncCount       int64  `json:"sync_count"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
	TotalBytes      int64   `json:"total_bytes"`
	FailureCount    int64   `json:"failure_count"`
}

type CacheConsistency struct {
	TotalChecks    int64   `json:"total_checks"`
	ConsistentKeys int64   `json:"consistent_keys"`
	InconsistentKeys int64 `json:"inconsistent_keys"`
	AvgCheckLatency float64 `json:"avg_check_latency"`
}

func NewEdgeCacheSync(nodeManager *EdgeNodeManager, redisClient *redis.Client, config *CacheSyncConfig) *EdgeCacheSync {
	if config == nil {
		config = &CacheSyncConfig{
			Strategy:          SyncStrategyEventual,
			BatchSize:         100,
			SyncInterval:      5 * time.Second,
			MaxQueueSize:      10000,
			CompressionEnabled: true,
			RetryAttempts:     3,
			RetryDelay:        1 * time.Second,
		}
	}

	sync := &EdgeCacheSync{
		cache:         make(map[string]*CacheSyncEntry),
		nodeManager:   nodeManager,
		redisClient:   redisClient,
		config:        config,
		syncQueue:     make(chan *CacheSyncMessage, config.MaxQueueSize),
		metrics: &CacheSyncMetrics{
			RegionMetrics: make(map[Region]*RegionSyncMetric),
		},
		eventHandlers: make(map[string][]func(*CacheSyncMessage)),
	}

	sync.startSyncProcessor()
	sync.startExpiryChecker()
	sync.startMetricsCollector()

	return sync
}

func (s *EdgeCacheSync) Set(ctx context.Context, key string, value interface{}, ttl time.Duration, options ...CacheOption) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := &CacheSyncEntry{
		Key:       key,
		Value:     value,
		TTL:       ttl,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
		Checksum:  s.calculateChecksum(value),
		NodeID:    s.nodeID,
		Region:    s.region,
		Expired:   false,
	}

	for _, opt := range options {
		opt(entry)
	}

	existing, exists := s.cache[key]
	if exists {
		entry.CreatedAt = existing.CreatedAt
		entry.Version = existing.Version + 1
	}

	s.cache[key] = entry

	if s.redisClient != nil {
		data, _ := json.Marshal(entry)
		redisKey := fmt.Sprintf("edge:cache:%s", key)
		if ttl > 0 {
			s.redisClient.Set(ctx, redisKey, data, ttl)
		} else {
			s.redisClient.Set(ctx, redisKey, data, 0)
		}
	}

	s.enqueueSyncMessage(&CacheSyncMessage{
		Type:      "set",
		Key:       key,
		Value:     value,
		TTL:       int64(ttl.Seconds()),
		Version:   entry.Version,
		Checksum:  entry.Checksum,
		Timestamp: time.Now(),
		NodeID:    s.nodeID,
		Region:    s.region,
		Tags:      entry.Tags,
	})

	return nil
}

func (s *EdgeCacheSync) Get(ctx context.Context, key string) (interface{}, error) {
	s.mu.RLock()
	entry, exists := s.cache[key]
	s.mu.RUnlock()

	if !exists || entry.Expired {
		if s.redisClient != nil {
			redisKey := fmt.Sprintf("edge:cache:%s", key)
			data, err := s.redisClient.Get(ctx, redisKey).Bytes()
			if err == nil {
				var cachedEntry CacheSyncEntry
				if err := json.Unmarshal(data, &cachedEntry); err == nil {
					if cachedEntry.TTL > 0 && time.Since(cachedEntry.UpdatedAt) > cachedEntry.TTL {
						return nil, fmt.Errorf("cache entry expired")
					}
					return cachedEntry.Value, nil
				}
			}
		}
		return nil, fmt.Errorf("cache miss: %s", key)
	}

	if entry.TTL > 0 && time.Since(entry.UpdatedAt) > entry.TTL {
		s.Delete(ctx, key)
		return nil, fmt.Errorf("cache entry expired")
	}

	return entry.Value, nil
}

func (s *EdgeCacheSync) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.cache[key]; !exists {
		return nil
	}

	delete(s.cache, key)

	if s.redisClient != nil {
		s.redisClient.Del(ctx, fmt.Sprintf("edge:cache:%s", key))
	}

	s.enqueueSyncMessage(&CacheSyncMessage{
		Type:      "delete",
		Key:       key,
		Timestamp: time.Now(),
		NodeID:    s.nodeID,
		Region:    s.region,
	})

	return nil
}

func (s *EdgeCacheSync) GetOrSet(ctx context.Context, key string, factory func() (interface{}, error), ttl time.Duration) (interface{}, error) {
	if value, err := s.Get(ctx, key); err == nil {
		return value, nil
	}

	value, err := factory()
	if err != nil {
		return nil, err
	}

	if err := s.Set(ctx, key, value, ttl); err != nil {
		return nil, err
	}

	return value, nil
}

func (s *EdgeCacheSync) InvalidateByTag(ctx context.Context, tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var keysToDelete []string
	for key, entry := range s.cache {
		for _, t := range entry.Tags {
			if t == tag {
				keysToDelete = append(keysToDelete, key)
				break
			}
		}
	}

	for _, key := range keysToDelete {
		delete(s.cache, key)
		if s.redisClient != nil {
			s.redisClient.Del(ctx, fmt.Sprintf("edge:cache:%s", key))
		}
	}

	return nil
}

func (s *EdgeCacheSync) GetByTag(ctx context.Context, tag string) ([]*CacheSyncEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var entries []*CacheSyncEntry
	for _, entry := range s.cache {
		for _, t := range entry.Tags {
			if t == tag {
				entryCopy := *entry
				entries = append(entries, &entryCopy)
				break
			}
		}
	}

	return entries, nil
}

func (s *EdgeCacheSync) Exists(ctx context.Context, key string) bool {
	s.mu.RLock()
	entry, exists := s.cache[key]
	s.mu.RUnlock()

	if !exists {
		return false
	}

	if entry.TTL > 0 && time.Since(entry.UpdatedAt) > entry.TTL {
		return false
	}

	return true
}

func (s *EdgeCacheSync) Flush(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache = make(map[string]*CacheSyncEntry)

	if s.redisClient != nil {
		iter := s.redisClient.Scan(ctx, 0, "edge:cache:*", 0).Iterator()
		for iter.Next(ctx) {
			s.redisClient.Del(ctx, iter.Val())
		}
	}

	s.enqueueSyncMessage(&CacheSyncMessage{
		Type:      "flush",
		Timestamp: time.Now(),
		NodeID:    s.nodeID,
		Region:    s.region,
	})

	return nil
}

func (s *EdgeCacheSync) enqueueSyncMessage(msg *CacheSyncMessage) {
	if s.config.Strategy == SyncStrategyLazy || s.config.Strategy == SyncStrategyEventual {
		select {
		case s.syncQueue <- msg:
		default:
			atomic.AddInt64(&s.metrics.FailedSyncs, 1)
		}
	} else {
		go s.processSyncMessage(msg)
	}
}

func (s *EdgeCacheSync) processSyncMessage(msg *CacheSyncMessage) {
	atomic.AddInt64(&s.metrics.TotalSyncs, 1)
	startTime := time.Now()

	var err error
	switch msg.Type {
	case "set":
		err = s.syncSetToNodes(msg)
	case "delete":
		err = s.syncDeleteToNodes(msg)
	case "flush":
		err = s.syncFlushToNodes(msg)
	}

	latency := time.Since(startTime).Seconds() * 1000
	s.metrics.SyncLatencyMs = latency

	if err != nil {
		atomic.AddInt64(&s.metrics.FailedSyncs, 1)
	} else {
		atomic.AddInt64(&s.metrics.SuccessfulSyncs, 1)
		s.updateRegionMetrics(msg.Region, latency, len(fmt.Sprintf("%v", msg.Value)))
	}
}

func (s *EdgeCacheSync) syncSetToNodes(msg *CacheSyncMessage) error {
	if s.nodeManager == nil {
		return nil
	}

	nodes, err := s.nodeManager.GetNodesByRegion(msg.Region)
	if err != nil || len(nodes) == 0 {
		nodes, err = s.nodeManager.ListNodes(&NodeSelectorOptions{Status: NodeStatusActive})
		if err != nil {
			return err
		}
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(nodes))

	for _, node := range nodes {
		if node.ID == s.nodeID {
			continue
		}
		wg.Add(1)
		go func(n *EdgeNode) {
			defer wg.Done()
			if err := s.syncToNode(n, msg); err != nil {
				errChan <- err
			}
		}(node)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *EdgeCacheSync) syncToNode(node *EdgeNode, msg *CacheSyncMessage) error {
	return nil
}

func (s *EdgeCacheSync) syncDeleteToNodes(msg *CacheSyncMessage) error {
	return s.syncSetToNodes(msg)
}

func (s *EdgeCacheSync) syncFlushToNodes(msg *CacheSyncMessage) error {
	return s.syncSetToNodes(msg)
}

func (s *EdgeCacheSync) updateRegionMetrics(region Region, latencyMs float64, valueSize int) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	if _, exists := s.metrics.RegionMetrics[region]; !exists {
		s.metrics.RegionMetrics[region] = &RegionSyncMetric{Region: region}
	}

	metric := s.metrics.RegionMetrics[region]
	metric.SyncCount++
	metric.TotalBytes += int64(valueSize)
	metric.AvgLatencyMs = (metric.AvgLatencyMs*float64(metric.SyncCount-1) + latencyMs) / float64(metric.SyncCount)
}

func (s *EdgeCacheSync) startSyncProcessor() {
	if s.config.Strategy != SyncStrategyLazy && s.config.Strategy != SyncStrategyEventual {
		return
	}

	go func() {
		batch := make([]*CacheSyncMessage, 0, s.config.BatchSize)
		ticker := time.NewTicker(s.config.SyncInterval)
		defer ticker.Stop()

		for {
			select {
			case msg := <-s.syncQueue:
				batch = append(batch, msg)
				if len(batch) >= s.config.BatchSize {
					s.processBatch(batch)
					batch = batch[:0]
				}
			case <-ticker.C:
				if len(batch) > 0 {
					s.processBatch(batch)
					batch = batch[:0]
				}
			}
		}
	}()
}

func (s *EdgeCacheSync) processBatch(messages []*CacheSyncMessage) {
	for _, msg := range messages {
		s.processSyncMessage(msg)
	}
}

func (s *EdgeCacheSync) startExpiryChecker() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			s.checkExpiry()
		}
	}()
}

func (s *EdgeCacheSync) checkExpiry() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, entry := range s.cache {
		if entry.TTL > 0 && now.Sub(entry.UpdatedAt) > entry.TTL {
			entry.Expired = true
		}
	}
}

func (s *EdgeCacheSync) startMetricsCollector() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			s.metrics.mu.Lock()
			s.metrics.QueueSize = len(s.syncQueue)
			s.metrics.mu.Unlock()
		}
	}()
}

func (s *EdgeCacheSync) calculateChecksum(value interface{}) string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

func (s *EdgeCacheSync) GetMetrics() *CacheSyncMetrics {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	metricsCopy := &CacheSyncMetrics{
		TotalSyncs:       atomic.LoadInt64(&s.metrics.TotalSyncs),
		SuccessfulSyncs:  atomic.LoadInt64(&s.metrics.SuccessfulSyncs),
		FailedSyncs:      atomic.LoadInt64(&s.metrics.FailedSyncs),
		BytesTransferred: atomic.LoadInt64(&s.metrics.BytesTransferred),
		SyncLatencyMs:    s.metrics.SyncLatencyMs,
		QueueSize:        len(s.syncQueue),
		RegionMetrics:    make(map[Region]*RegionSyncMetric),
	}

	for k, v := range s.metrics.RegionMetrics {
		metricCopy := *v
		metricsCopy.RegionMetrics[k] = &metricCopy
	}

	return metricsCopy
}

func (s *EdgeCacheSync) SetNodeID(nodeID string) {
	s.nodeID = nodeID
}

func (s *EdgeCacheSync) SetRegion(region Region) {
	s.region = region
}

func (s *EdgeCacheSync) OnEvent(eventType string, handler func(*CacheSyncMessage)) {
	s.handlerMu.Lock()
	defer s.handlerMu.Unlock()

	s.eventHandlers[eventType] = append(s.eventHandlers[eventType], handler)
}

func (s *EdgeCacheSync) EmitEvent(msg *CacheSyncMessage) {
	s.handlerMu.RLock()
	handlers := s.eventHandlers[msg.Type]
	s.handlerMu.RUnlock()

	for _, handler := range handlers {
		go handler(msg)
	}
}

func (s *EdgeCacheSync) CheckConsistency(ctx context.Context) (*CacheConsistency, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	consistency := &CacheConsistency{}
	now := time.Now()

	for key, entry := range s.cache {
		atomic.AddInt64(&consistency.TotalChecks, 1)

		if entry.TTL > 0 && now.Sub(entry.UpdatedAt) > entry.TTL {
			atomic.AddInt64(&consistency.InconsistentKeys, 1)
			continue
		}

		if s.redisClient != nil {
			redisKey := fmt.Sprintf("edge:cache:%s", key)
			data, err := s.redisClient.Get(ctx, redisKey).Bytes()
			if err != nil {
				atomic.AddInt64(&consistency.InconsistentKeys, 1)
				continue
			}

			var cachedEntry CacheSyncEntry
			if err := json.Unmarshal(data, &cachedEntry); err != nil {
				atomic.AddInt64(&consistency.InconsistentKeys, 1)
				continue
			}

			if entry.Version != cachedEntry.Version {
				atomic.AddInt64(&consistency.InconsistentKeys, 1)
				continue
			}
		}

		atomic.AddInt64(&consistency.ConsistentKeys, 1)
	}

	return consistency, nil
}

func (s *EdgeCacheSync) GetKeys(pattern string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys []string
	for key := range s.cache {
		if pattern == "" || pattern == "*" {
			keys = append(keys, key)
		} else if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
			prefix := pattern[:len(pattern)-1]
			if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
				keys = append(keys, key)
			}
		} else if key == pattern {
			keys = append(keys, key)
		}
	}

	return keys
}

func (s *EdgeCacheSync) GetEntry(key string) (*CacheSyncEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.cache[key]
	if !exists {
		return nil, fmt.Errorf("cache entry not found: %s", key)
	}

	entryCopy := *entry
	return &entryCopy, nil
}

func (s *EdgeCacheSync) UpdateConfig(config *CacheSyncConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = config
}

func (s *EdgeCacheSync) GetConfig() *CacheSyncConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configCopy := *s.config
	return &configCopy
}

func (s *EdgeCacheSync) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.cache)
}

type CacheOption func(*CacheSyncEntry)

func WithTags(tags []string) CacheOption {
	return func(entry *CacheSyncEntry) {
		entry.Tags = tags
	}
}

func WithMetadata(metadata map[string]interface{}) CacheOption {
	return func(entry *CacheSyncEntry) {
		entry.Metadata = metadata
	}
}

func WithCompression(compressed bool) CacheOption {
	return func(entry *CacheSyncEntry) {
		entry.Compressed = compressed
	}
}

func (s *EdgeCacheSync) SyncFromRedis(ctx context.Context) error {
	if s.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	iter := s.redisClient.Scan(ctx, 0, "edge:cache:*", 0).Iterator()
	s.mu.Lock()
	defer s.mu.Unlock()

	for iter.Next(ctx) {
		data, err := s.redisClient.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}

		var entry CacheSyncEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}

		key := iter.Val()[len("edge:cache:"):]
		entry.Key = key
		s.cache[key] = &entry
	}

	return iter.Err()
}
