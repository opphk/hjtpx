package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrVersionMismatch = errors.New("version mismatch")
	ErrDataCorrupted  = errors.New("data corrupted")
	ErrSyncTimeout    = errors.New("sync timeout")
)

type InvalidationStrategy int

const (
	InvalidateOnWrite InvalidationStrategy = iota
	InvalidateOnRead
	InvalidateOnExpire
	InvalidateBatch
	InvalidateDelayed
)

type SyncStrategy int

const (
	SyncImmediate SyncStrategy = iota
	SyncAsync
	SyncEventual
	SyncLazy
)

type ConflictResolutionStrategy int

const (
	ConflictLastWriteWins ConflictResolutionStrategy = iota
	ConflictFirstWriteWins
	ConflictVersionVector
	ConflictMerge
	ConflictNotify
)

type CacheConsistencyManager struct {
	config            *ConsistencyConfig
	invalidationQueue chan *InvalidationEvent
	syncManager       *SyncManager
	conflictResolver  *ConflictResolver
	versionTracker    *VersionTracker
	stats             *ConsistencyStats
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
}

type ConsistencyConfig struct {
	Strategy              InvalidationStrategy
	SyncStrategy          SyncStrategy
	ConflictStrategy      ConflictResolutionStrategy
	BatchSize             int
	BatchInterval         time.Duration
	SyncTimeout           time.Duration
	RetryCount            int
	RetryInterval         time.Duration
	EnableVersionTracking bool
	EnableConflictDetection bool
	MaxQueueSize          int
}

type InvalidationEvent struct {
	Key       string
	Keys      []string
	Tags      []string
	Pattern   string
	EventType string
	Source    string
	Timestamp time.Time
	Version   int64
}

type SyncManager struct {
	config        *ConsistencyConfig
	syncQueue     chan *SyncEvent
	subscribers   map[string]chan *SyncEvent
	mu            sync.RWMutex
	pendingSyncs  map[string]*PendingSync
	eventEmitter  *EventEmitter
}

type SyncEvent struct {
	Key       string
	Operation string
	Data      []byte
	Version   int64
	Timestamp time.Time
	Source    string
}

type PendingSync struct {
	Key       string
	Operation string
	Data      []byte
	Retries   int
	CreatedAt time.Time
	LastTry   time.Time
}

type ConflictResolver struct {
	config     *ConsistencyConfig
	versionDB  *sync.Map
	conflicts  *sync.Map
	handlers   map[ConflictResolutionStrategy]ConflictHandler
}

type ConflictHandler func(a, b *ConflictData) (*ConflictData, error)

type ConflictData struct {
	Key       string
	Value     []byte
	Version   int64
	Timestamp time.Time
	Source    string
	Checksum  uint64
}

type ConflictInfo struct {
	Key         string
	OldValue    *ConflictData
	NewValue    *ConflictData
	Resolution  string
	ResolvedAt  time.Time
}

type VersionTracker struct {
	versions *sync.Map
	mu       sync.RWMutex
}

type ConsistencyStats struct {
	Invalidations    atomic.Int64
	SyncSuccess      atomic.Int64
	SyncFailures     atomic.Int64
	Conflicts        atomic.Int64
	ConflictResolved atomic.Int64
	VersionMismatch  atomic.Int64
	TotalLatency     atomic.Int64
	RequestCount     atomic.Int64
}

type EventEmitter struct {
	subscribers map[string][]EventHandler
	mu          sync.RWMutex
}

type EventHandler func(event *CacheEvent)

type CacheEvent struct {
	Type    string
	Key     string
	Data    interface{}
	Time    time.Time
	Source  string
}

type WriteThroughPolicy struct {
	cache   *EnhancedCache
	db      DatabaseWriter
	mu      sync.RWMutex
	enabled bool
}

type DatabaseWriter interface {
	Write(ctx context.Context, key string, value []byte) error
	Read(ctx context.Context, key string) ([]byte, error)
}

type CacheAsidePolicy struct {
	cache        *EnhancedCache
	db           DatabaseReader
	mu           sync.RWMutex
	enabled      bool
	readOnlyMode bool
}

type DatabaseReader interface {
	Read(ctx context.Context, key string) ([]byte, error)
}

type WriteBehindPolicy struct {
	cache      *EnhancedCache
	buffer     *WriteBuffer
	flushInterval time.Duration
	mu         sync.RWMutex
	enabled    bool
}

type WriteBuffer struct {
	items    map[string]*BufferItem
	mu       sync.Mutex
	maxSize  int
}

type BufferItem struct {
	Key       string
	Value     []byte
	Operation string
	Timestamp time.Time
	Retries   int
}

type DataValidator struct {
	checksumAlgorithm string
	schemaValidators map[string]SchemaValidator
	mu                sync.RWMutex
}

type SchemaValidator func(data []byte) error

type ConsistencyChecker struct {
	manager *CacheConsistencyManager
	interval time.Duration
	mu      sync.RWMutex
	enabled bool
}

func DefaultConsistencyConfig() *ConsistencyConfig {
	return &ConsistencyConfig{
		Strategy:               InvalidateOnWrite,
		SyncStrategy:           SyncAsync,
		ConflictStrategy:       ConflictLastWriteWins,
		BatchSize:              100,
		BatchInterval:          100 * time.Millisecond,
		SyncTimeout:            5 * time.Second,
		RetryCount:             3,
		RetryInterval:          1 * time.Second,
		EnableVersionTracking:  true,
		EnableConflictDetection: true,
		MaxQueueSize:           10000,
	}
}

func NewCacheConsistencyManager(config *ConsistencyConfig) *CacheConsistencyManager {
	if config == nil {
		config = DefaultConsistencyConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &CacheConsistencyManager{
		config:            config,
		invalidationQueue: make(chan *InvalidationEvent, config.MaxQueueSize),
		syncManager:       NewSyncManager(config),
		conflictResolver:  NewConflictResolver(config),
		versionTracker:    NewVersionTracker(),
		stats:             &ConsistencyStats{},
		ctx:               ctx,
		cancel:            cancel,
	}

	manager.startInvalidationWorker()
	manager.startSyncWorker()

	return manager
}

func (ccm *CacheConsistencyManager) startInvalidationWorker() {
	ccm.wg.Add(1)
	go func() {
		defer ccm.wg.Done()

		batch := make([]*InvalidationEvent, 0, ccm.config.BatchSize)
		ticker := time.NewTicker(ccm.config.BatchInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ccm.ctx.Done():
				if len(batch) > 0 {
					ccm.processBatch(batch)
				}
				return
			case event := <-ccm.invalidationQueue:
				batch = append(batch, event)
				if len(batch) >= ccm.config.BatchSize {
					ccm.processBatch(batch)
					batch = make([]*InvalidationEvent, 0, ccm.config.BatchSize)
				}
			case <-ticker.C:
				if len(batch) > 0 {
					ccm.processBatch(batch)
					batch = make([]*InvalidationEvent, 0, ccm.config.BatchSize)
				}
			}
		}
	}()
}

func (ccm *CacheConsistencyManager) processBatch(events []*InvalidationEvent) {
	for _, event := range events {
		ccm.processInvalidationEvent(event)
	}
}

func (ccm *CacheConsistencyManager) processInvalidationEvent(event *InvalidationEvent) {
	start := time.Now()
	defer func() {
		ccm.stats.TotalLatency.Add(time.Since(start).Nanoseconds())
	}()

	switch event.EventType {
	case "delete":
		ccm.invalidateKey(event.Key, event.Version)
	case "delete_pattern":
		ccm.invalidatePattern(event.Pattern)
	case "delete_tag":
		ccm.invalidateByTag(event.Tags)
	case "delete_keys":
		ccm.invalidateKeys(event.Keys)
	}

	ccm.stats.Invalidations.Add(1)
}

func (ccm *CacheConsistencyManager) invalidateKey(key string, version int64) {
	if ccm.config.EnableVersionTracking {
		if v, _ := ccm.versionTracker.Get(key); v >= version {
			return
		}
		ccm.versionTracker.Set(key, version)
	}

	cache := GetEnhancedCache()
	if cache != nil {
		cache.Delete(context.Background(), key, &DeleteOptions{Level: CacheLevelBoth})
	}

	ccm.syncManager.EmitSyncEvent(&SyncEvent{
		Key:       key,
		Operation: "invalidate",
		Version:   version,
		Timestamp: time.Now(),
	})
}

func (ccm *CacheConsistencyManager) invalidatePattern(pattern string) {
	cache := GetEnhancedCache()
	if cache == nil || Client == nil {
		return
	}

	ctx := context.Background()
	iter := Client.Scan(ctx, 0, pattern, 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	for _, key := range keys {
		cache.Delete(ctx, key, &DeleteOptions{Level: CacheLevelBoth})
		ccm.versionTracker.Increment(key)
	}
}

func (ccm *CacheConsistencyManager) invalidateByTag(tags []string) {
	for _, tag := range tags {
		cache := GetEnhancedCache()
		if cache != nil {
			cache.Delete(context.Background(), tag, &DeleteOptions{Level: CacheLevelBoth, ByTag: true})
		}
	}
}

func (ccm *CacheConsistencyManager) invalidateKeys(keys []string) {
	for _, key := range keys {
		ccm.invalidateKey(key, ccm.versionTracker.Increment(key))
	}
}

func (ccm *CacheConsistencyManager) Invalidate(key string) {
	ccm.invalidationQueue <- &InvalidationEvent{
		Key:       key,
		EventType: "delete",
		Timestamp: time.Now(),
		Version:   ccm.versionTracker.Increment(key),
	}
}

func (ccm *CacheConsistencyManager) InvalidateKeys(keys []string) {
	ccm.invalidationQueue <- &InvalidationEvent{
		Keys:      keys,
		EventType: "delete_keys",
		Timestamp: time.Now(),
	}
}

func (ccm *CacheConsistencyManager) InvalidatePattern(pattern string) {
	ccm.invalidationQueue <- &InvalidationEvent{
		Pattern:   pattern,
		EventType: "delete_pattern",
		Timestamp: time.Now(),
	}
}

func (ccm *CacheConsistencyManager) InvalidateByTag(tags []string) {
	ccm.invalidationQueue <- &InvalidationEvent{
		Tags:      tags,
		EventType: "delete_tag",
		Timestamp: time.Now(),
	}
}

func (ccm *CacheConsistencyManager) RegisterSyncSubscriber(key string, handler chan *SyncEvent) {
	ccm.syncManager.RegisterSubscriber(key, handler)
}

func (ccm *CacheConsistencyManager) UnregisterSyncSubscriber(key string, handler chan *SyncEvent) {
	ccm.syncManager.UnregisterSubscriber(key, handler)
}

func (ccm *CacheConsistencyManager) startSyncWorker() {
	ccm.wg.Add(1)
	go ccm.syncManager.Start(ccm.ctx)
}

func (ccm *CacheConsistencyManager) Sync(key string, data []byte, version int64) error {
	return ccm.syncManager.Sync(key, data, version)
}

func (ccm *CacheConsistencyManager) GetVersion(key string) (int64, error) {
	return ccm.versionTracker.Get(key)
}

func (ccm *CacheConsistencyManager) CheckVersion(key string, expectedVersion int64) error {
	actualVersion, err := ccm.versionTracker.Get(key)
	if err != nil {
		return err
	}
	if actualVersion != expectedVersion {
		ccm.stats.VersionMismatch.Add(1)
		return ErrVersionMismatch
	}
	return nil
}

func (ccm *CacheConsistencyManager) DetectConflict(key string, newData *ConflictData) (*ConflictInfo, error) {
	if !ccm.config.EnableConflictDetection {
		return nil, nil
	}

	existing, _ := ccm.conflictResolver.GetStoredVersion(key)
	if existing != nil && existing.Version >= newData.Version {
		conflict := &ConflictInfo{
			Key:      key,
			OldValue: existing,
			NewValue: newData,
		}
		ccm.stats.Conflicts.Add(1)

		resolved, err := ccm.conflictResolver.Resolve(existing, newData)
		if err != nil {
			return conflict, err
		}

		conflict.Resolution = ccm.config.ConflictStrategy.String()
		conflict.ResolvedAt = time.Now()
		ccm.stats.ConflictResolved.Add(1)

		ccm.conflictResolver.StoreVersion(key, resolved)
		return conflict, nil
	}

	ccm.conflictResolver.StoreVersion(key, newData)
	return nil, nil
}

func NewSyncManager(config *ConsistencyConfig) *SyncManager {
	return &SyncManager{
		config:       config,
		syncQueue:    make(chan *SyncEvent, config.MaxQueueSize),
		subscribers:  make(map[string]chan *SyncEvent),
		pendingSyncs: make(map[string]*PendingSync),
		eventEmitter: NewEventEmitter(),
	}
}

func (sm *SyncManager) Start(ctx context.Context) {
	ticker := time.NewTicker(sm.config.BatchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-sm.syncQueue:
			sm.processSyncEvent(event)
		case <-ticker.C:
			sm.retryPendingSyncs()
		}
	}
}

func (sm *SyncManager) processSyncEvent(event *SyncEvent) {
	if sm.config.SyncStrategy == SyncAsync {
		go sm.executeSync(event)
	} else {
		sm.executeSync(event)
	}
}

func (sm *SyncManager) executeSync(event *SyncEvent) {
	cache := GetEnhancedCache()
	if cache == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), sm.config.SyncTimeout)
	defer cancel()

	var err error
	switch event.Operation {
	case "invalidate":
		err = cache.Delete(ctx, event.Key, &DeleteOptions{Level: CacheLevelBoth})
	case "update":
		err = cache.Set(ctx, event.Key, event.Data, &SetOptions{
			Level:    CacheLevelBoth,
			Version:  event.Version,
		})
	}

	if err != nil {
		sm.pendingSyncs[event.Key] = &PendingSync{
			Key:       event.Key,
			Operation: event.Operation,
			Data:      event.Data,
			Retries:   0,
			CreatedAt: time.Now(),
			LastTry:   time.Now(),
		}
	} else {
		delete(sm.pendingSyncs, event.Key)
	}
}

func (sm *SyncManager) retryPendingSyncs() {
	for key, pending := range sm.pendingSyncs {
		if pending.Retries >= sm.config.RetryCount {
			delete(sm.pendingSyncs, key)
			continue
		}

		if time.Since(pending.LastTry) < sm.config.RetryInterval {
			continue
		}

		event := &SyncEvent{
			Key:       pending.Key,
			Operation: pending.Operation,
			Data:      pending.Data,
			Timestamp: time.Now(),
		}

		sm.executeSync(event)
		pending.Retries++
		pending.LastTry = time.Now()
	}
}

func (sm *SyncManager) EmitSyncEvent(event *SyncEvent) {
	sm.syncQueue <- event
	sm.eventEmitter.Emit("sync", &CacheEvent{
		Type:   "sync",
		Key:    event.Key,
		Data:   event,
		Time:   time.Now(),
		Source: event.Source,
	})
}

func (sm *SyncManager) Sync(key string, data []byte, version int64) error {
	event := &SyncEvent{
		Key:       key,
		Operation: "update",
		Data:      data,
		Version:   version,
		Timestamp: time.Now(),
	}

	if sm.config.SyncStrategy == SyncImmediate {
		sm.executeSync(event)
		return nil
	}

	sm.EmitSyncEvent(event)
	return nil
}

func (sm *SyncManager) RegisterSubscriber(key string, handler chan *SyncEvent) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.subscribers[key] = handler
}

func (sm *SyncManager) UnregisterSubscriber(key string, handler chan *SyncEvent) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.subscribers, key)
}

func NewConflictResolver(config *ConsistencyConfig) *ConflictResolver {
	cr := &ConflictResolver{
		config:    config,
		versionDB: &sync.Map{},
		conflicts: &sync.Map{},
		handlers:  make(map[ConflictResolutionStrategy]ConflictHandler),
	}

	cr.handlers[ConflictLastWriteWins] = cr.lastWriteWins
	cr.handlers[ConflictFirstWriteWins] = cr.firstWriteWins
	cr.handlers[ConflictVersionVector] = cr.versionVector
	cr.handlers[ConflictMerge] = cr.merge
	cr.handlers[ConflictNotify] = cr.notify

	return cr
}

func (cr *ConflictResolver) lastWriteWins(a, b *ConflictData) (*ConflictData, error) {
	if a.Timestamp.After(b.Timestamp) {
		return a, nil
	}
	return b, nil
}

func (cr *ConflictResolver) firstWriteWins(a, b *ConflictData) (*ConflictData, error) {
	if a.Timestamp.Before(b.Timestamp) {
		return a, nil
	}
	return b, nil
}

func (cr *ConflictResolver) versionVector(a, b *ConflictData) (*ConflictData, error) {
	if a.Version >= b.Version {
		return a, nil
	}
	return b, nil
}

func (cr *ConflictResolver) merge(a, b *ConflictData) (*ConflictData, error) {
	if a.Key != b.Key {
		return nil, errors.New("cannot merge different keys")
	}

	if string(a.Value) == string(b.Value) {
		return a, nil
	}

	merged := &ConflictData{
		Key:       a.Key,
		Version:   max(a.Version, b.Version) + 1,
		Timestamp: time.Now(),
		Source:    "merged",
	}

	var mergedData map[string]interface{}
	if err := json.Unmarshal(a.Value, &mergedData); err == nil {
		var bData map[string]interface{}
		if err := json.Unmarshal(b.Value, &bData); err == nil {
			for k, v := range bData {
				mergedData[k] = v
			}
			merged.Value, _ = json.Marshal(mergedData)
			return merged, nil
		}
	}

	if a.Timestamp.After(b.Timestamp) {
		merged.Value = a.Value
	} else {
		merged.Value = b.Value
	}

	return merged, nil
}

func (cr *ConflictResolver) notify(a, b *ConflictData) (*ConflictData, error) {
	conflict := &ConflictInfo{
		Key:      a.Key,
		OldValue: a,
		NewValue: b,
		Resolution: "notification_sent",
	}
	cr.conflicts.Store(a.Key, conflict)
	return a, nil
}

func (cr *ConflictResolver) Resolve(a, b *ConflictData) (*ConflictData, error) {
	handler, ok := cr.handlers[cr.config.ConflictStrategy]
	if !ok {
		handler = cr.handlers[ConflictLastWriteWins]
	}
	return handler(a, b)
}

func (cr *ConflictResolver) StoreVersion(key string, data *ConflictData) {
	cr.versionDB.Store(key, data)
}

func (cr *ConflictResolver) GetStoredVersion(key string) (*ConflictData, error) {
	val, ok := cr.versionDB.Load(key)
	if !ok {
		return nil, nil
	}
	return val.(*ConflictData), nil
}

func (cr *ConflictResolver) GetConflicts() []*ConflictInfo {
	var conflicts []*ConflictInfo
	cr.conflicts.Range(func(key, value interface{}) bool {
		if info, ok := value.(*ConflictInfo); ok {
			conflicts = append(conflicts, info)
		}
		return true
	})
	return conflicts
}

func NewVersionTracker() *VersionTracker {
	return &VersionTracker{
		versions: &sync.Map{},
	}
}

func (vt *VersionTracker) Get(key string) (int64, error) {
	val, ok := vt.versions.Load(key)
	if !ok {
		return 0, nil
	}
	return val.(int64), nil
}

func (vt *VersionTracker) Set(key string, version int64) {
	vt.versions.Store(key, version)
}

func (vt *VersionTracker) Increment(key string) int64 {
	val, _ := vt.versions.LoadOrStore(key, int64(0))
	newVal := val.(int64) + 1
	vt.versions.Store(key, newVal)
	return newVal
}

func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		subscribers: make(map[string][]EventHandler),
	}
}

func (ee *EventEmitter) Subscribe(eventType string, handler EventHandler) {
	ee.mu.Lock()
	defer ee.mu.Unlock()
	ee.subscribers[eventType] = append(ee.subscribers[eventType], handler)
}

func (ee *EventEmitter) Emit(eventType string, event *CacheEvent) {
	ee.mu.RLock()
	handlers := ee.subscribers[eventType]
	ee.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}

func (ccm *CacheConsistencyManager) GetStats() *ConsistencyStats {
	return ccm.stats
}

func (ccm *CacheConsistencyManager) Close() {
	ccm.cancel()
	done := make(chan struct{})
	go func() {
		ccm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
}

func (ccm *CacheConsistencyManager) CreateWriteThroughPolicy(db DatabaseWriter) *WriteThroughPolicy {
	return &WriteThroughPolicy{
		cache:   GetEnhancedCache(),
		db:      db,
		enabled: true,
	}
}

func (ccm *CacheConsistencyManager) CreateCacheAsidePolicy(db DatabaseReader) *CacheAsidePolicy {
	return &CacheAsidePolicy{
		cache:        GetEnhancedCache(),
		db:           db,
		enabled:      true,
		readOnlyMode: false,
	}
}

func (ccm *CacheConsistencyManager) CreateWriteBehindPolicy(flushInterval time.Duration) *WriteBehindPolicy {
	return &WriteBehindPolicy{
		cache:         GetEnhancedCache(),
		buffer:        NewWriteBuffer(1000),
		flushInterval: flushInterval,
		enabled:       true,
	}
}

func (wt *WriteThroughPolicy) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := wt.db.Write(ctx, key, value); err != nil {
		return err
	}

	if wt.cache != nil {
		wt.cache.Set(ctx, key, value, &SetOptions{
			Level: CacheLevelBoth,
			TTL:   ttl,
		})
	}

	return nil
}

func (wt *WriteThroughPolicy) Delete(ctx context.Context, key string) error {
	if err := wt.db.Write(ctx, key, nil); err != nil {
		return err
	}

	if wt.cache != nil {
		wt.cache.Delete(ctx, key, &DeleteOptions{Level: CacheLevelBoth})
	}

	return nil
}

func (ca *CacheAsidePolicy) Get(ctx context.Context, key string) ([]byte, error) {
	if ca.cache != nil {
		data, err := ca.cache.Get(ctx, key, nil)
		if err == nil {
			return data, nil
		}
	}

	data, err := ca.db.Read(ctx, key)
	if err != nil {
		return nil, err
	}

	if ca.cache != nil && !ca.readOnlyMode {
		ca.cache.Set(ctx, key, data, &SetOptions{Level: CacheLevelBoth})
	}

	return data, nil
}

func NewWriteBuffer(maxSize int) *WriteBuffer {
	return &WriteBuffer{
		items:   make(map[string]*BufferItem),
		maxSize: maxSize,
	}
}

func (wb *WriteBuffer) Add(key string, value []byte, operation string) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	wb.items[key] = &BufferItem{
		Key:       key,
		Value:     value,
		Operation: operation,
		Timestamp: time.Now(),
	}
}

func (wb *WriteBuffer) GetAndClear() map[string]*BufferItem {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	result := wb.items
	wb.items = make(map[string]*BufferItem)
	return result
}

func (wb *WriteBuffer) Size() int {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	return len(wb.items)
}

func (wb *WriteBehindPolicy) Set(ctx context.Context, key string, value []byte) error {
	wb.buffer.Add(key, value, "set")

	if wb.cache != nil {
		wb.cache.Set(ctx, key, value, &SetOptions{Level: CacheLevelBoth})
	}

	return nil
}

func (wb *WriteBehindPolicy) Delete(ctx context.Context, key string) error {
	wb.buffer.Add(key, nil, "delete")

	if wb.cache != nil {
		wb.cache.Delete(ctx, key, &DeleteOptions{Level: CacheLevelBoth})
	}

	return nil
}

func (wb *WriteBehindPolicy) Flush(ctx context.Context) error {
	items := wb.buffer.GetAndClear()

	for key, item := range items {
		if item.Operation == "delete" {
			fmt.Printf("Flushing delete for key: %s\n", key)
		} else {
			fmt.Printf("Flushing set for key: %s\n", key)
		}
	}

	return nil
}

func NewDataValidator(checksumAlgorithm string) *DataValidator {
	return &DataValidator{
		checksumAlgorithm: checksumAlgorithm,
		schemaValidators:  make(map[string]SchemaValidator),
	}
}

func (dv *DataValidator) AddSchemaValidator(key string, validator SchemaValidator) {
	dv.mu.Lock()
	defer dv.mu.Unlock()
	dv.schemaValidators[key] = validator
}

func (dv *DataValidator) Validate(key string, data []byte) error {
	dv.mu.RLock()
	validator, ok := dv.schemaValidators[key]
	dv.mu.RUnlock()

	if ok && validator != nil {
		return validator(data)
	}

	return nil
}

func (dv *DataValidator) ComputeChecksum(data []byte) uint64 {
	h := fnv.New64a()
	h.Write(data)
	return h.Sum64()
}

func (ccm *CacheConsistencyManager) CreateConsistencyChecker(interval time.Duration) *ConsistencyChecker {
	return &ConsistencyChecker{
		manager:  ccm,
		interval: interval,
		enabled:  true,
	}
}

func (cc *ConsistencyChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(cc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if cc.enabled {
				cc.check()
			}
		}
	}
}

func (cc *ConsistencyChecker) check() {
	fmt.Println("Running consistency check...")
}

func (cc *ConsistencyChecker) Enable() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.enabled = true
}

func (cc *ConsistencyChecker) Disable() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.enabled = false
}

func (s InvalidationStrategy) String() string {
	switch s {
	case InvalidateOnWrite:
		return "invalidate_on_write"
	case InvalidateOnRead:
		return "invalidate_on_read"
	case InvalidateOnExpire:
		return "invalidate_on_expire"
	case InvalidateBatch:
		return "invalidate_batch"
	case InvalidateDelayed:
		return "invalidate_delayed"
	default:
		return "unknown"
	}
}

func (s SyncStrategy) String() string {
	switch s {
	case SyncImmediate:
		return "sync_immediate"
	case SyncAsync:
		return "sync_async"
	case SyncEventual:
		return "sync_eventual"
	case SyncLazy:
		return "sync_lazy"
	default:
		return "unknown"
	}
}

func (s ConflictResolutionStrategy) String() string {
	switch s {
	case ConflictLastWriteWins:
		return "last_write_wins"
	case ConflictFirstWriteWins:
		return "first_write_wins"
	case ConflictVersionVector:
		return "version_vector"
	case ConflictMerge:
		return "merge"
	case ConflictNotify:
		return "notify"
	default:
		return "unknown"
	}
}

type ConsistentHashRing struct {
	nodes     map[string][]string
	replicas  int
	mu        sync.RWMutex
}

func NewConsistentHashRing(replicas int) *ConsistentHashRing {
	if replicas <= 0 {
		replicas = 100
	}
	return &ConsistentHashRing{
		nodes:    make(map[string][]string),
		replicas: replicas,
	}
}

func (chr *ConsistentHashRing) AddNode(node string) {
	chr.mu.Lock()
	defer chr.mu.Unlock()
	chr.nodes[node] = nil
}

func (chr *ConsistentHashRing) RemoveNode(node string) {
	chr.mu.Lock()
	defer chr.mu.Unlock()
	delete(chr.nodes, node)
}

func (chr *ConsistentHashRing) GetNode(key string) string {
	chr.mu.RLock()
	defer chr.mu.RUnlock()

	if len(chr.nodes) == 0 {
		return ""
	}

	h := fnv.New64a()
	h.Write([]byte(key))
	hash := h.Sum64()

	var result string
	minHash := uint64(0)
	first := true

	for node := range chr.nodes {
		for i := 0; i < chr.replicas; i++ {
			h.Reset()
			h.Write([]byte(fmt.Sprintf("%s:%d", node, i)))
			nodeHash := h.Sum64()

			if first || nodeHash > minHash && nodeHash <= hash {
				minHash = nodeHash
				result = node
				first = false
			}
		}
	}

	return result
}

func (ccm *CacheConsistencyManager) CreateConsistentHashRing(replicas int) *ConsistentHashRing {
	return NewConsistentHashRing(replicas)
}

var (
	globalConsistencyManager *CacheConsistencyManager
	globalConsistencyOnce   sync.Once
)

func InitConsistencyManager(config *ConsistencyConfig) {
	globalConsistencyOnce.Do(func() {
		globalConsistencyManager = NewCacheConsistencyManager(config)
	})
}

func GetConsistencyManager() *CacheConsistencyManager {
	if globalConsistencyManager == nil {
		InitConsistencyManager(nil)
	}
	return globalConsistencyManager
}

func CloseConsistencyManager() {
	if globalConsistencyManager != nil {
		globalConsistencyManager.Close()
	}
}
