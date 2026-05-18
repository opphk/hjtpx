package redis

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ExpirationStrategy int

const (
	ExpirationFixed ExpirationStrategy = iota
	ExpirationSliding
	ExpirationRandom
	ExpirationAdaptive
)

type InvalidationMode int

const (
	InvalidationImmediate InvalidationMode = iota
	InvalidationDelayed
	InvalidationBatch
	InvalidationLazy
)

type CacheExpirationConfig struct {
	DefaultTTL         time.Duration
	MinTTL            time.Duration
	MaxTTL            time.Duration
	Strategy          ExpirationStrategy
	SlidingWindow     time.Duration
	RandomVariance    float64
	EnableAutoRefresh bool
	RefreshThreshold  float64
}

var DefaultExpirationConfig = &CacheExpirationConfig{
	DefaultTTL:        10 * time.Minute,
	MinTTL:            1 * time.Minute,
	MaxTTL:            1 * time.Hour,
	Strategy:          ExpirationSliding,
	SlidingWindow:     1 * time.Minute,
	RandomVariance:    0.1,
	EnableAutoRefresh: true,
	RefreshThreshold:  0.8,
}

type CacheInvalidationConfig struct {
	Mode              InvalidationMode
	BatchSize         int
	BatchInterval     time.Duration
	DelayDuration     time.Duration
	EnableVersioning  bool
	EnableTags        bool
	ConsistencyLevel  string
}

var DefaultInvalidationConfig = &CacheInvalidationConfig{
	Mode:             InvalidationImmediate,
	BatchSize:        100,
	BatchInterval:    1 * time.Second,
	DelayDuration:    100 * time.Millisecond,
	EnableVersioning: true,
	EnableTags:       true,
	ConsistencyLevel: "strong",
}

type CacheExpirationManager struct {
	config     *CacheExpirationConfig
	versioning map[string]int64
	mu         sync.RWMutex
	refreshCh  chan string
}

type CacheInvalidationManager struct {
	config         *CacheInvalidationConfig
	pendingDeletes map[string]*PendingDelete
	batchQueue     chan string
	versionCache   map[string]int64
	tagIndex       map[string]map[string]bool
	mu             sync.RWMutex
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

type PendingDelete struct {
	Key       string
	Tags      []string
	Timestamp time.Time
	Delay     time.Duration
}

type CacheConsistencyManager struct {
	mu          sync.RWMutex
	lockCache   map[string]*CacheLock
	versionMap  map[string]int64
	watchers    map[string][]chan struct{}
	maxVersions int
}

type CacheLock struct {
	Key        string
	Version    int64
	AcquiredAt time.Time
	TTL        time.Duration
}

func NewCacheExpirationManager(config *CacheExpirationConfig) *CacheExpirationManager {
	if config == nil {
		config = DefaultExpirationConfig
	}

	return &CacheExpirationManager{
		config:     config,
		versioning: make(map[string]int64),
		refreshCh:  make(chan string, 1000),
	}
}

func (cem *CacheExpirationManager) CalculateTTL(key string, baseTTL time.Duration) time.Duration {
	switch cem.config.Strategy {
	case ExpirationFixed:
		return baseTTL

	case ExpirationSliding:
		return baseTTL + cem.config.SlidingWindow

	case ExpirationRandom:
		variance := cem.config.RandomVariance
		randomFactor := 1.0 + (float64(time.Now().UnixNano()%1000)/1000.0-0.5)*2*variance
		ttl := time.Duration(float64(baseTTL) * randomFactor)
		
		if ttl < cem.config.MinTTL {
			return cem.config.MinTTL
		}
		if ttl > cem.config.MaxTTL {
			return cem.config.MaxTTL
		}
		return ttl

	case ExpirationAdaptive:
		version := cem.GetVersion(key)
		multiplier := 1.0 + float64(version)*0.1
		ttl := time.Duration(float64(baseTTL) * multiplier)
		
		if ttl > cem.config.MaxTTL {
			return cem.config.MaxTTL
		}
		return ttl

	default:
		return baseTTL
	}
}

func (cem *CacheExpirationManager) GetVersion(key string) int64 {
	cem.mu.RLock()
	defer cem.mu.RUnlock()
	return cem.versioning[key]
}

func (cem *CacheExpirationManager) IncrementVersion(key string) int64 {
	cem.mu.Lock()
	defer cem.mu.Unlock()
	version := cem.versioning[key] + 1
	cem.versioning[key] = version
	return version
}

func (cem *CacheExpirationManager) SetVersion(key string, version int64) {
	cem.mu.Lock()
	defer cem.mu.Unlock()
	cem.versioning[key] = version
}

func (cem *CacheExpirationManager) ShouldRefresh(key string, remainingTTL time.Duration) bool {
	if !cem.config.EnableAutoRefresh {
		return false
	}

	threshold := time.Duration(float64(cem.config.DefaultTTL) * cem.config.RefreshThreshold)
	return remainingTTL < threshold
}

func (cem *CacheExpirationManager) GetRefreshChannel() <-chan string {
	return cem.refreshCh
}

func (cem *CacheExpirationManager) SignalRefresh(key string) {
	select {
	case cem.refreshCh <- key:
	default:
	}
}

func (cem *CacheExpirationManager) CleanupVersions(maxAge time.Duration) {
	cem.mu.Lock()
	defer cem.mu.Unlock()

	for key := range cem.versioning {
		delete(cem.versioning, key)
		if len(cem.versioning) <= 100 {
			break
		}
	}
}

func NewCacheInvalidationManager(config *CacheInvalidationConfig) *CacheInvalidationManager {
	if config == nil {
		config = DefaultInvalidationConfig
	}

	manager := &CacheInvalidationManager{
		config:         config,
		pendingDeletes: make(map[string]*PendingDelete),
		batchQueue:     make(chan string, 10000),
		versionCache:   make(map[string]int64),
		tagIndex:       make(map[string]map[string]bool),
		stopCh:         make(chan struct{}),
	}

	if config.Mode == InvalidationBatch {
		manager.startBatchProcessor()
	}

	return manager
}

func (cim *CacheInvalidationManager) Invalidate(ctx context.Context, key string) error {
	return cim.InvalidateWithTags(ctx, key, nil)
}

func (cim *CacheInvalidationManager) InvalidateWithTags(ctx context.Context, key string, tags []string) error {
	if cim.config.Mode == InvalidationImmediate {
		return cim.invalidateImmediately(ctx, key, tags)
	} else if cim.config.Mode == InvalidationDelayed {
		return cim.invalidateDelayed(ctx, key, tags)
	} else if cim.config.Mode == InvalidationBatch {
		return cim.invalidateBatch(ctx, key, tags)
	}
	return nil
}

func (cim *CacheInvalidationManager) invalidateImmediately(ctx context.Context, key string, tags []string) error {
	if Client == nil {
		return nil
	}

	pipe := Client.Pipeline()

	if cim.config.EnableVersioning {
		cim.IncrementVersion(key)
	}

	pipe.Del(ctx, key)

	if cim.config.EnableTags && len(tags) > 0 {
		for _, tag := range tags {
			tagKey := fmt.Sprintf("%s:%s", PrefixTag, tag)
			pipe.SRem(ctx, tagKey, key)
		}
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (cim *CacheInvalidationManager) invalidateDelayed(ctx context.Context, key string, tags []string) error {
	cim.mu.Lock()
	cim.pendingDeletes[key] = &PendingDelete{
		Key:       key,
		Tags:      tags,
		Timestamp: time.Now(),
		Delay:     cim.config.DelayDuration,
	}
	cim.mu.Unlock()

	go func() {
		time.Sleep(cim.config.DelayDuration)
		cim.processPendingDelete(context.Background(), key)
	}()

	return nil
}

func (cim *CacheInvalidationManager) invalidateBatch(ctx context.Context, key string, tags []string) error {
	select {
	case cim.batchQueue <- key:
		return nil
	default:
		return fmt.Errorf("batch queue is full")
	}
}

func (cim *CacheInvalidationManager) processPendingDelete(ctx context.Context, key string) {
	cim.mu.Lock()
	pending, exists := cim.pendingDeletes[key]
	if !exists {
		cim.mu.Unlock()
		return
	}
	delete(cim.pendingDeletes, key)
	cim.mu.Unlock()

	if pending != nil {
		cim.invalidateImmediately(ctx, pending.Key, pending.Tags)
	}
}

func (cim *CacheInvalidationManager) startBatchProcessor() {
	cim.wg.Add(1)
	go func() {
		defer cim.wg.Done()

		batch := make([]string, 0, cim.config.BatchSize)
		ticker := time.NewTicker(cim.config.BatchInterval)
		defer ticker.Stop()

		for {
			select {
			case <-cim.stopCh:
				if len(batch) > 0 {
					cim.processBatch(context.Background(), batch)
				}
				return

			case key := <-cim.batchQueue:
				batch = append(batch, key)
				if len(batch) >= cim.config.BatchSize {
					cim.processBatch(context.Background(), batch)
					batch = make([]string, 0, cim.config.BatchSize)
				}

			case <-ticker.C:
				if len(batch) > 0 {
					cim.processBatch(context.Background(), batch)
					batch = make([]string, 0, cim.config.BatchSize)
				}
			}
		}
	}()
}

func (cim *CacheInvalidationManager) processBatch(ctx context.Context, keys []string) {
	if Client == nil || len(keys) == 0 {
		return
	}

	pipe := Client.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}

	pipe.Exec(ctx)
}

func (cim *CacheInvalidationManager) InvalidateByTag(ctx context.Context, tag string) error {
	if !cim.config.EnableTags || Client == nil {
		return nil
	}

	tagKey := fmt.Sprintf("%s:%s", PrefixTag, tag)
	keys, err := Client.SMembers(ctx, tagKey).Result()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	pipe := Client.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}
	pipe.Del(ctx, tagKey)

	_, err = pipe.Exec(ctx)
	return err
}

func (cim *CacheInvalidationManager) InvalidateByPattern(ctx context.Context, pattern string) (int, error) {
	if Client == nil {
		return 0, nil
	}

	var deleted int
	iter := Client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := Client.Del(ctx, iter.Val()).Err(); err == nil {
			deleted++
		}
	}

	return deleted, iter.Err()
}

func (cim *CacheInvalidationManager) IncrementVersion(key string) int64 {
	cim.mu.Lock()
	defer cim.mu.Unlock()

	version := cim.versionCache[key] + 1
	cim.versionCache[key] = version
	return version
}

func (cim *CacheInvalidationManager) GetVersion(key string) int64 {
	cim.mu.RLock()
	defer cim.mu.RUnlock()
	return cim.versionCache[key]
}

func (cim *CacheInvalidationManager) AddTag(key, tag string) {
	cim.mu.Lock()
	defer cim.mu.Unlock()

	if cim.tagIndex[tag] == nil {
		cim.tagIndex[tag] = make(map[string]bool)
	}
	cim.tagIndex[tag][key] = true
}

func (cim *CacheInvalidationManager) GetKeysByTag(tag string) []string {
	cim.mu.RLock()
	defer cim.mu.RUnlock()

	if keys, ok := cim.tagIndex[tag]; ok {
		result := make([]string, 0, len(keys))
		for key := range keys {
			result = append(result, key)
		}
		return result
	}
	return nil
}

func (cim *CacheInvalidationManager) Stop() {
	close(cim.stopCh)
	cim.wg.Wait()
}

func NewCacheConsistencyManager(maxVersions int) *CacheConsistencyManager {
	if maxVersions <= 0 {
		maxVersions = 100
	}

	return &CacheConsistencyManager{
		lockCache:   make(map[string]*CacheLock),
		versionMap:  make(map[string]int64),
		watchers:    make(map[string][]chan struct{}),
		maxVersions: maxVersions,
	}
}

func (ccm *CacheConsistencyManager) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	if lock, exists := ccm.lockCache[key]; exists {
		if time.Since(lock.AcquiredAt) < lock.TTL {
			return false, nil
		}
	}

	version := ccm.IncrementVersion(key)
	ccm.lockCache[key] = &CacheLock{
		Key:        key,
		Version:    version,
		AcquiredAt: time.Now(),
		TTL:        ttl,
	}

	go ccm.notifyWatchers(key)

	return true, nil
}

func (ccm *CacheConsistencyManager) ReleaseLock(key string) {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()
	delete(ccm.lockCache, key)
}

func (ccm *CacheConsistencyManager) IncrementVersion(key string) int64 {
	version := ccm.versionMap[key] + 1
	ccm.versionMap[key] = version

	if len(ccm.versionMap) > ccm.maxVersions {
		ccm.cleanupOldVersions()
	}

	return version
}

func (ccm *CacheConsistencyManager) GetVersion(key string) int64 {
	ccm.mu.RLock()
	defer ccm.mu.RUnlock()
	return ccm.versionMap[key]
}

func (ccm *CacheConsistencyManager) Watch(key string, ch chan struct{}) {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()
	ccm.watchers[key] = append(ccm.watchers[key], ch)
}

func (ccm *CacheConsistencyManager) Unwatch(key string, ch chan struct{}) {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	if watchers, ok := ccm.watchers[key]; ok {
		for i, watcher := range watchers {
			if watcher == ch {
				ccm.watchers[key] = append(watchers[:i], watchers[i+1:]...)
				break
			}
		}
	}
}

func (ccm *CacheConsistencyManager) notifyWatchers(key string) {
	ccm.mu.RLock()
	watchers := ccm.watchers[key]
	ccm.mu.RUnlock()

	for _, ch := range watchers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (ccm *CacheConsistencyManager) cleanupOldVersions() {
	for key := range ccm.versionMap {
		delete(ccm.versionMap, key)
		if len(ccm.versionMap) <= ccm.maxVersions/2 {
			break
		}
	}
}

type AdaptiveExpirationPolicy struct {
	mu            sync.RWMutex
	accessCounts  map[string]int64
	baseTTL      time.Duration
	minTTL       time.Duration
	maxTTL       time.Duration
	windowSize   time.Duration
	refreshRatio float64
	ttlStrategy  map[string]time.Duration
}

func NewAdaptiveExpirationPolicy(baseTTL, minTTL, maxTTL time.Duration) *AdaptiveExpirationPolicy {
	return &AdaptiveExpirationPolicy{
		accessCounts: make(map[string]int64),
		baseTTL:      baseTTL,
		minTTL:       minTTL,
		maxTTL:       maxTTL,
		windowSize:   5 * time.Minute,
		refreshRatio: 0.8,
		ttlStrategy:  make(map[string]time.Duration),
	}
}

func (aep *AdaptiveExpirationPolicy) RecordAccess(key string) {
	aep.mu.Lock()
	defer aep.mu.Unlock()

	count := aep.accessCounts[key] + 1
	aep.accessCounts[key] = count

	if len(aep.accessCounts) > 10000 {
		aep.cleanup()
	}
}

func (aep *AdaptiveExpirationPolicy) CalculateTTL(key string) time.Duration {
	aep.mu.RLock()
	defer aep.mu.RUnlock()

	count := aep.accessCounts[key]

	multiplier := 1.0 + float64(count)*0.01
	ttl := time.Duration(float64(aep.baseTTL) * multiplier)

	if ttl < aep.minTTL {
		return aep.minTTL
	}
	if ttl > aep.maxTTL {
		return aep.maxTTL
	}

	return ttl
}

func (aep *AdaptiveExpirationPolicy) GetOptimizedTTL(key string, accessPattern string) time.Duration {
	baseTTL := aep.CalculateTTL(key)

	switch accessPattern {
	case "frequent":
		return baseTTL * 2
	case "moderate":
		return baseTTL
	case "rare":
		return baseTTL / 2
	default:
		return baseTTL
	}
}

func (aep *AdaptiveExpirationPolicy) SetTTLStrategy(keyPattern string, ttl time.Duration) {
	aep.mu.Lock()
	defer aep.mu.Unlock()
	aep.ttlStrategy[keyPattern] = ttl
}

func (aep *AdaptiveExpirationPolicy) GetTTLStrategy(keyPattern string) time.Duration {
	aep.mu.RLock()
	defer aep.mu.RUnlock()

	if ttl, ok := aep.ttlStrategy[keyPattern]; ok {
		return ttl
	}
	return aep.baseTTL
}

func (aep *AdaptiveExpirationPolicy) ShouldRefresh(key string, remainingTTL time.Duration) bool {
	if remainingTTL <= 0 {
		return false
	}

	aep.mu.RLock()
	accessCount := aep.accessCounts[key]
	aep.mu.RUnlock()

	if accessCount == 0 {
		return false
	}

	threshold := time.Duration(float64(aep.baseTTL) * aep.refreshRatio)
	return remainingTTL < threshold
}

func (aep *AdaptiveExpirationPolicy) cleanup() {
	for key, count := range aep.accessCounts {
		if count == 0 {
			delete(aep.accessCounts, key)
		}
		if len(aep.accessCounts) <= 1000 {
			break
		}
	}
}

func (aep *AdaptiveExpirationPolicy) Cleanup() {
	aep.mu.Lock()
	defer aep.mu.Unlock()
	aep.accessCounts = make(map[string]int64)
}

type TTLOption struct {
	BaseTTL      time.Duration
	MinTTL       time.Duration
	MaxTTL       time.Duration
	RefreshRatio float64
	Strategy     string
}

var DefaultTTLOption = &TTLOption{
	BaseTTL:      10 * time.Minute,
	MinTTL:       1 * time.Minute,
	MaxTTL:       1 * time.Hour,
	RefreshRatio: 0.8,
	Strategy:     "sliding",
}

func (aep *AdaptiveExpirationPolicy) ApplyOption(opt *TTLOption) {
	aep.mu.Lock()
	defer aep.mu.Unlock()

	if opt.BaseTTL > 0 {
		aep.baseTTL = opt.BaseTTL
	}
	if opt.MinTTL > 0 {
		aep.minTTL = opt.MinTTL
	}
	if opt.MaxTTL > 0 {
		aep.maxTTL = opt.MaxTTL
	}
	if opt.RefreshRatio > 0 {
		aep.refreshRatio = opt.RefreshRatio
	}
}

type CacheExpirationOptimizer struct {
	policies map[string]*AdaptiveExpirationPolicy
	mu       sync.RWMutex
}

func NewCacheExpirationOptimizer() *CacheExpirationOptimizer {
	return &CacheExpirationOptimizer{
		policies: make(map[string]*AdaptiveExpirationPolicy),
	}
}

func (ceo *CacheExpirationOptimizer) AddPolicy(name string, baseTTL, minTTL, maxTTL time.Duration) {
	ceo.mu.Lock()
	defer ceo.mu.Unlock()

	ceo.policies[name] = NewAdaptiveExpirationPolicy(baseTTL, minTTL, maxTTL)
}

func (ceo *CacheExpirationOptimizer) GetPolicy(name string) *AdaptiveExpirationPolicy {
	ceo.mu.RLock()
	defer ceo.mu.RUnlock()

	return ceo.policies[name]
}

func (ceo *CacheExpirationOptimizer) OptimizeTTL(key string, policyName string) time.Duration {
	policy := ceo.GetPolicy(policyName)
	if policy == nil {
		policy = NewAdaptiveExpirationPolicy(DefaultTTLOption.BaseTTL, DefaultTTLOption.MinTTL, DefaultTTLOption.MaxTTL)
	}

	return policy.CalculateTTL(key)
}

var (
	globalExpirationManager   *CacheExpirationManager
	globalInvalidationManager *CacheInvalidationManager
	globalConsistencyManager *CacheConsistencyManager
	managersOnce             sync.Once
)

func InitCacheManagers(expirationConfig *CacheExpirationConfig, invalidationConfig *CacheInvalidationConfig) {
	managersOnce.Do(func() {
		globalExpirationManager = NewCacheExpirationManager(expirationConfig)
		globalInvalidationManager = NewCacheInvalidationManager(invalidationConfig)
		globalConsistencyManager = NewCacheConsistencyManager(1000)
	})
}

func GetExpirationManager() *CacheExpirationManager {
	if globalExpirationManager == nil {
		InitCacheManagers(nil, nil)
	}
	return globalExpirationManager
}

func GetInvalidationManager() *CacheInvalidationManager {
	if globalInvalidationManager == nil {
		InitCacheManagers(nil, nil)
	}
	return globalInvalidationManager
}

func GetConsistencyManager() *CacheConsistencyManager {
	if globalConsistencyManager == nil {
		InitCacheManagers(nil, nil)
	}
	return globalConsistencyManager
}
