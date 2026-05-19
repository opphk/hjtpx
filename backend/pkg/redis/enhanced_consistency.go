package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type UpdateStrategy int

const (
	UpdateStrategySync UpdateStrategy = iota
	UpdateStrategyAsync
	UpdateStrategyDeferred
)

type VersionedCacheEntry struct {
	Key       string
	Value     []byte
	Version   int64
	Timestamp time.Time
	Checksum  string
	Source    string
}

type CacheVersionVector struct {
	mu       sync.RWMutex
	versions map[string]int64
}

func NewCacheVersionVector() *CacheVersionVector {
	return &CacheVersionVector{
		versions: make(map[string]int64),
	}
}

func (cvv *CacheVersionVector) Increment(nodeID string) int64 {
	cvv.mu.Lock()
	defer cvv.mu.Unlock()

	version := cvv.versions[nodeID] + 1
	cvv.versions[nodeID] = version
	return version
}

func (cvv *CacheVersionVector) Get(nodeID string) int64 {
	cvv.mu.RLock()
	defer cvv.mu.RUnlock()
	return cvv.versions[nodeID]
}

func (cvv *CacheVersionVector) Merge(other *CacheVersionVector) {
	cvv.mu.Lock()
	defer cvv.mu.Unlock()

	other.mu.RLock()
	defer other.mu.RUnlock()

	for nodeID, version := range other.versions {
		if version > cvv.versions[nodeID] {
			cvv.versions[nodeID] = version
		}
	}
}

func (cvv *CacheVersionVector) IsConcurrent(other *CacheVersionVector) bool {
	cvv.mu.RLock()
	ownVersions := make(map[string]int64)
	for k, v := range cvv.versions {
		ownVersions[k] = v
	}
	cvv.mu.RUnlock()

	other.mu.RLock()
	defer other.mu.RUnlock()

	for nodeID, otherVersion := range other.versions {
		if ownVersion, exists := ownVersions[nodeID]; exists && ownVersion > otherVersion {
			return false
		}
	}

	return true
}

type EnhancedCacheConsistency struct {
	mu                sync.RWMutex
	config            *EnhancedConsistencyConfig
	mode              ConsistencyMode
	updateStrategy    UpdateStrategy
	versionVector     *CacheVersionVector
	localCache        *sync.Map
	pendingUpdates    chan *VersionedCacheEntry
	consistencyQueue  chan *ConsistencyMessage
	conflictResolver  ConflictResolver
	versionManager    *EnhancedVersionManager
	metrics           *EnhancedConsistencyMetrics
	pubSubChannels    map[string]*goredis.PubSub
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
}

type EnhancedConsistencyConfig struct {
	Mode                ConsistencyMode
	UpdateStrategy      UpdateStrategy
	VersionVectorEnabled bool
	CheckInterval       time.Duration
	MaxPendingUpdates   int
	ConflictResolution  string
	EnablePubSub        bool
	PubSubChannels      []string
	SyncTimeout         time.Duration
	RetryAttempts       int
}

type ConsistencyMessage struct {
	Type      string
	Key       string
	Version   int64
	Timestamp time.Time
	Source    string
	Checksum  string
}

type ConflictResolver interface {
	Resolve(conflicts [][]byte) []byte
}

type DefaultConflictResolver struct{}

func (dcr *DefaultConflictResolver) Resolve(conflicts [][]byte) []byte {
	if len(conflicts) == 0 {
		return nil
	}

	if len(conflicts) == 1 {
		return conflicts[0]
	}

	var latest []byte
	var latestTime time.Time

	for _, data := range conflicts {
		var entry VersionedCacheEntry
		if err := json.Unmarshal(data, &entry); err == nil {
			if latestTime.IsZero() || entry.Timestamp.After(latestTime) {
				latest = data
				latestTime = entry.Timestamp
			}
		}
	}

	return latest
}

type EnhancedVersionManager struct {
	mu       sync.RWMutex
	versions map[string]*VersionInfo
}

type VersionInfo struct {
	Version   int64
	UpdatedAt time.Time
	Checksum  string
	NodeID    string
}

func NewEnhancedVersionManager() *EnhancedVersionManager {
	return &EnhancedVersionManager{
		versions: make(map[string]*VersionInfo),
	}
}

func (evm *EnhancedVersionManager) GetVersion(key string) int64 {
	evm.mu.RLock()
	defer evm.mu.RUnlock()

	if info, exists := evm.versions[key]; exists {
		return info.Version
	}
	return 0
}

func (evm *EnhancedVersionManager) IncrementVersion(key string, nodeID string) int64 {
	evm.mu.Lock()
	defer evm.mu.Unlock()

	info := evm.versions[key]
	if info == nil {
		info = &VersionInfo{
			Version:   1,
			UpdatedAt: time.Now(),
			NodeID:    nodeID,
		}
	} else {
		info.Version++
		info.UpdatedAt = time.Now()
		info.NodeID = nodeID
	}

	evm.versions[key] = info
	return info.Version
}

func (evm *EnhancedVersionManager) SetVersion(key string, version int64, checksum string, nodeID string) {
	evm.mu.Lock()
	defer evm.mu.Unlock()

	info := &VersionInfo{
		Version:   version,
		UpdatedAt: time.Now(),
		Checksum:  checksum,
		NodeID:    nodeID,
	}

	evm.versions[key] = info
}

func (evm *EnhancedVersionManager) GetInfo(key string) *VersionInfo {
	evm.mu.RLock()
	defer evm.mu.RUnlock()
	return evm.versions[key]
}

type EnhancedConsistencyMetrics struct {
	TotalOperations    atomic.Int64
	SuccessfulOps      atomic.Int64
	FailedOps          atomic.Int64
	Conflicts          atomic.Int64
	VersionMismatches  atomic.Int64
	PendingUpdates     atomic.Int64
	CompletedUpdates   atomic.Int64
	LastConsistencyTime atomic.Value
}

var DefaultEnhancedConsistencyConfig = &EnhancedConsistencyConfig{
	Mode:                ConsistencyModeEventual,
	UpdateStrategy:      UpdateStrategySync,
	VersionVectorEnabled: true,
	CheckInterval:       1 * time.Second,
	MaxPendingUpdates:   1000,
	ConflictResolution:  "timestamp",
	EnablePubSub:        true,
	PubSubChannels:      []string{"cache:consistency", "cache:invalidation"},
	SyncTimeout:         5 * time.Second,
	RetryAttempts:       3,
}

func NewEnhancedCacheConsistency(config *EnhancedConsistencyConfig) *EnhancedCacheConsistency {
	if config == nil {
		config = DefaultEnhancedConsistencyConfig
	}

	ctx, cancel := context.WithCancel(context.Background())

	ecc := &EnhancedCacheConsistency{
		config:           config,
		mode:              config.Mode,
		updateStrategy:    config.UpdateStrategy,
		versionVector:     NewCacheVersionVector(),
		localCache:        &sync.Map{},
		pendingUpdates:    make(chan *VersionedCacheEntry, config.MaxPendingUpdates),
		consistencyQueue:  make(chan *ConsistencyMessage, config.MaxPendingUpdates),
		conflictResolver:  &DefaultConflictResolver{},
		versionManager:    NewEnhancedVersionManager(),
		metrics:           &EnhancedConsistencyMetrics{},
		pubSubChannels:    make(map[string]*goredis.PubSub),
		ctx:               ctx,
		cancel:            cancel,
	}

	ecc.startWorkers()

	if config.EnablePubSub {
		ecc.startPubSubListeners()
	}

	return ecc
}

func (ecc *EnhancedCacheConsistency) startWorkers() {
	ecc.wg.Add(1)
	go ecc.processPendingUpdates()

	ecc.wg.Add(1)
	go ecc.processConsistencyQueue()

	ecc.wg.Add(1)
	go ecc.versionVectorGC()
}

func (ecc *EnhancedCacheConsistency) processPendingUpdates() {
	defer ecc.wg.Done()

	for {
		select {
		case <-ecc.ctx.Done():
			return
		case update := <-ecc.pendingUpdates:
			ecc.applyUpdate(update)
		}
	}
}

func (ecc *EnhancedCacheConsistency) processConsistencyQueue() {
	defer ecc.wg.Done()

	ticker := time.NewTicker(ecc.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ecc.ctx.Done():
			return
		case <-ticker.C:
			ecc.checkConsistency()
		case msg := <-ecc.consistencyQueue:
			ecc.handleConsistencyMessage(msg)
		}
	}
}

func (ecc *EnhancedCacheConsistency) applyUpdate(update *VersionedCacheEntry) {
	ecc.mu.Lock()
	defer ecc.mu.Unlock()

	oldInfo := ecc.versionManager.GetInfo(update.Key)

	if oldInfo != nil && oldInfo.Version >= update.Version {
		return
	}

	ecc.versionManager.SetVersion(update.Key, update.Version, update.Checksum, update.Source)
	ecc.localCache.Store(update.Key, update)

	ecc.metrics.CompletedUpdates.Add(1)
	ecc.metrics.LastConsistencyTime.Store(time.Now())
}

func (ecc *EnhancedCacheConsistency) checkConsistency() {
	if Client == nil {
		return
	}

	ecc.mu.RLock()
	keys := make([]string, 0)
	ecc.localCache.Range(func(key, value interface{}) bool {
		keys = append(keys, key.(string))
		return true
	})
	ecc.mu.RUnlock()

	for _, key := range keys {
		if err := ecc.verifyKeyConsistency(key); err != nil {
		}
	}
}

func (ecc *EnhancedCacheConsistency) verifyKeyConsistency(key string) error {
	if Client == nil {
		return nil
	}

	versionKey := fmt.Sprintf("cache:version:%s", key)
	remoteVersion, err := Client.Get(ecc.ctx, versionKey).Int64()
	if err != nil && err != goredis.Nil {
		return err
	}

	localVersion := ecc.versionManager.GetVersion(key)

	if remoteVersion > localVersion {
		ecc.metrics.VersionMismatches.Add(1)

		val, err := Client.Get(ecc.ctx, key).Bytes()
		if err != nil {
			return err
		}

		update := &VersionedCacheEntry{
			Key:       key,
			Value:     val,
			Version:   remoteVersion,
			Timestamp: time.Now(),
			Source:    "remote",
		}

		ecc.pendingUpdates <- update
	}

	return nil
}

func (ecc *EnhancedCacheConsistency) handleConsistencyMessage(msg *ConsistencyMessage) {
	switch msg.Type {
	case "invalidate":
		ecc.invalidateKey(msg.Key, msg.Version)
	case "update":
		ecc.handleUpdateMessage(msg)
	case "sync":
		ecc.requestSync(msg.Key)
	}
}

func (ecc *EnhancedCacheConsistency) invalidateKey(key string, version int64) {
	ecc.mu.Lock()
	defer ecc.mu.Unlock()

	currentVersion := ecc.versionManager.GetVersion(key)
	if version > currentVersion {
		ecc.versionManager.SetVersion(key, version, "", "invalidation")
	}

	ecc.localCache.Delete(key)

	if enhancedCache := GetEnhancedCache(); enhancedCache != nil {
		enhancedCache.Delete(context.Background(), key, &DeleteOptions{Level: CacheLevelL1})
	}
}

func (ecc *EnhancedCacheConsistency) handleUpdateMessage(msg *ConsistencyMessage) {
	ecc.metrics.TotalOperations.Add(1)

	localVersion := ecc.versionManager.GetVersion(msg.Key)

	if msg.Version > localVersion {
		ecc.invalidateKey(msg.Key, msg.Version)
	}
}

func (ecc *EnhancedCacheConsistency) requestSync(key string) {
	if Client == nil {
		return
	}

	msg := &ConsistencyMessage{
		Type:      "sync",
		Key:       key,
		Version:   ecc.versionManager.GetVersion(key),
		Timestamp: time.Now(),
		Source:    "local",
	}

	payload, _ := json.Marshal(msg)
	for _, channel := range ecc.config.PubSubChannels {
		Client.Publish(ecc.ctx, channel, payload)
	}
}

func (ecc *EnhancedCacheConsistency) startPubSubListeners() {
	if Client == nil {
		return
	}

	for _, channel := range ecc.config.PubSubChannels {
		pubsub := Client.Subscribe(ecc.ctx, channel)
		ecc.pubSubChannels[channel] = pubsub

		ecc.wg.Add(1)
		go ecc.handlePubSubMessages(pubsub, channel)
	}
}

func (ecc *EnhancedCacheConsistency) handlePubSubMessages(pubsub *goredis.PubSub, channel string) {
	defer ecc.wg.Done()

	ch := pubsub.Channel()
	for {
		select {
		case <-ecc.ctx.Done():
			return
		case msg := <-ch:
			var consistencyMsg ConsistencyMessage
			if err := json.Unmarshal([]byte(msg.Payload), &consistencyMsg); err == nil {
				ecc.consistencyQueue <- &consistencyMsg
			}
		}
	}
}

func (ecc *EnhancedCacheConsistency) versionVectorGC() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ecc.ctx.Done():
			return
		case <-ticker.C:
			ecc.cleanupVersionVector()
		}
	}
}

func (ecc *EnhancedCacheConsistency) cleanupVersionVector() {
	ecc.mu.Lock()
	defer ecc.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)

	ecc.localCache.Range(func(key, value interface{}) bool {
		entry := value.(*VersionedCacheEntry)
		if entry.Timestamp.Before(cutoff) {
			ecc.localCache.Delete(key)
		}
		return true
	})
}

func (ecc *EnhancedCacheConsistency) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ecc.metrics.TotalOperations.Add(1)

	newVersion := ecc.versionManager.IncrementVersion(key, "local")

	checksum := computeChecksum(value)
	entry := &VersionedCacheEntry{
		Key:       key,
		Value:     value,
		Version:   newVersion,
		Timestamp: time.Now(),
		Checksum:  checksum,
		Source:    "local",
	}

	ecc.mu.Lock()
	ecc.localCache.Store(key, entry)
	ecc.mu.Unlock()

	ecc.versionVector.Increment("local")

	if ecc.updateStrategy == UpdateStrategySync {
		return ecc.writeThrough(ctx, key, value, ttl, newVersion, checksum)
	}

	go ecc.writeBehind(ctx, key, value, ttl, newVersion, checksum)

	return nil
}

func (ecc *EnhancedCacheConsistency) writeThrough(ctx context.Context, key string, value []byte, ttl time.Duration, version int64, checksum string) error {
	if Client == nil {
		return nil
	}

	pipe := Client.Pipeline()

	pipe.Set(ctx, key, value, ttl)

	versionKey := fmt.Sprintf("cache:version:%s", key)
	pipe.Set(ctx, versionKey, version, ttl)

	checksumKey := fmt.Sprintf("cache:checksum:%s", key)
	pipe.Set(ctx, checksumKey, checksum, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		ecc.metrics.FailedOps.Add(1)
		return err
	}

	ecc.metrics.SuccessfulOps.Add(1)
	return nil
}

func (ecc *EnhancedCacheConsistency) writeBehind(ctx context.Context, key string, value []byte, ttl time.Duration, version int64, checksum string) error {
	select {
	case ecc.pendingUpdates <- &VersionedCacheEntry{
		Key:       key,
		Value:     value,
		Version:   version,
		Timestamp: time.Now(),
		Checksum:  checksum,
		Source:    "local",
	}:
		ecc.metrics.PendingUpdates.Add(1)
	default:
		ecc.metrics.FailedOps.Add(1)
		return fmt.Errorf("pending updates queue is full")
	}

	ecc.metrics.SuccessfulOps.Add(1)
	return nil
}

func (ecc *EnhancedCacheConsistency) Get(ctx context.Context, key string) ([]byte, error) {
	ecc.mu.RLock()
	if entry, exists := ecc.localCache.Load(key); exists {
		ecc.mu.RUnlock()
		ecc.metrics.SuccessfulOps.Add(1)
		return entry.(*VersionedCacheEntry).Value, nil
	}
	ecc.mu.RUnlock()

	ecc.metrics.TotalOperations.Add(1)

	if err := ecc.verifyKeyConsistency(key); err != nil {
		ecc.metrics.FailedOps.Add(1)
		return nil, err
	}

	if entry, exists := ecc.localCache.Load(key); exists {
		return entry.(*VersionedCacheEntry).Value, nil
	}

	return nil, ErrCacheMiss
}

func (ecc *EnhancedCacheConsistency) Delete(ctx context.Context, key string) error {
	ecc.mu.Lock()
	defer ecc.mu.Unlock()

	newVersion := ecc.versionManager.IncrementVersion(key, "local")
	ecc.localCache.Delete(key)

	if err := ecc.publishInvalidation(ctx, key, newVersion); err != nil {
		return err
	}

	ecc.metrics.TotalOperations.Add(1)
	return nil
}

func (ecc *EnhancedCacheConsistency) publishInvalidation(ctx context.Context, key string, version int64) error {
	if Client == nil || !ecc.config.EnablePubSub {
		return nil
	}

	msg := &ConsistencyMessage{
		Type:      "invalidate",
		Key:       key,
		Version:   version,
		Timestamp: time.Now(),
		Source:    "local",
	}

	payload, _ := json.Marshal(msg)

	for _, channel := range ecc.config.PubSubChannels {
		if err := Client.Publish(ctx, channel, payload).Err(); err != nil {
			ecc.metrics.FailedOps.Add(1)
			return err
		}
	}

	return nil
}

func (ecc *EnhancedCacheConsistency) InvalidateByTag(ctx context.Context, tag string) error {
	if Client == nil {
		return nil
	}

	tagKey := fmt.Sprintf("cache:tag:%s", tag)
	keys, err := Client.SMembers(ctx, tagKey).Result()
	if err != nil {
		return err
	}

	for _, key := range keys {
		if err := ecc.Delete(ctx, key); err != nil {
			continue
		}
	}

	if len(keys) > 0 {
		Client.Del(ctx, tagKey)
	}

	return nil
}

func (ecc *EnhancedCacheConsistency) InvalidateByPattern(ctx context.Context, pattern string) error {
	if Client == nil {
		return nil
	}

	var deleted int64
	iter := Client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := ecc.Delete(ctx, iter.Val()); err == nil {
			deleted++
		}
	}

	return iter.Err()
}

func (ecc *EnhancedCacheConsistency) SyncKey(ctx context.Context, key string) error {
	if Client == nil {
		return nil
	}

	val, err := Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	versionKey := fmt.Sprintf("cache:version:%s", key)
	version, _ := Client.Get(ctx, versionKey).Int64()

	checksumKey := fmt.Sprintf("cache:checksum:%s", key)
	checksum, _ := Client.Get(ctx, checksumKey).Result()

	entry := &VersionedCacheEntry{
		Key:       key,
		Value:     val,
		Version:   version,
		Timestamp: time.Now(),
		Checksum:  checksum,
		Source:    "sync",
	}

	ecc.applyUpdate(entry)

	return nil
}

func (ecc *EnhancedCacheConsistency) GetVersion(key string) int64 {
	return ecc.versionManager.GetVersion(key)
}

func (ecc *EnhancedCacheConsistency) SetMode(mode ConsistencyMode) {
	ecc.mu.Lock()
	defer ecc.mu.Unlock()
	ecc.mode = mode
}

func (ecc *EnhancedCacheConsistency) GetMode() ConsistencyMode {
	ecc.mu.RLock()
	defer ecc.mu.RUnlock()
	return ecc.mode
}

func (ecc *EnhancedCacheConsistency) SetUpdateStrategy(strategy UpdateStrategy) {
	ecc.mu.Lock()
	defer ecc.mu.Unlock()
	ecc.updateStrategy = strategy
}

func (ecc *EnhancedCacheConsistency) GetMetrics() *EnhancedConsistencyMetrics {
	return ecc.metrics
}

func (ecc *EnhancedCacheConsistency) Close() {
	ecc.cancel()

	for _, pubsub := range ecc.pubSubChannels {
		pubsub.Close()
	}

	ecc.wg.Wait()
}

func computeChecksum(data []byte) string {
	sum := uint64(0)
	for i, b := range data {
		sum += uint64(b) * uint64(i+1)
	}
	return fmt.Sprintf("%x", sum)
}

var (
	globalEnhancedConsistency *EnhancedCacheConsistency
	globalConsistencyOnce    sync.Once
)

func InitEnhancedCacheConsistency(config *EnhancedConsistencyConfig) {
	globalConsistencyOnce.Do(func() {
		globalEnhancedConsistency = NewEnhancedCacheConsistency(config)
	})
}

func GetEnhancedCacheConsistency() *EnhancedCacheConsistency {
	if globalEnhancedConsistency == nil {
		InitEnhancedCacheConsistency(nil)
	}
	return globalEnhancedConsistency
}
