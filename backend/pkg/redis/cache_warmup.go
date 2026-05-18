package redis

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type WarmupPolicy int

const (
	WarmupPolicyEager WarmupPolicy = iota
	WarmupPolicyLazy
	WarmupPolicyScheduled
	WarmupPolicyAdaptive
	WarmupPolicyOnDemand
)

type WarmupPriority int

const (
	WarmupPriorityCritical WarmupPriority = iota
	WarmupPriorityHigh
	WarmupPriorityNormal
	WarmupPriorityLow
)

type WarmupItem struct {
	Key    string
	Value  []byte
	TTL    time.Duration
	Loader func(ctx context.Context) ([]byte, error)
}

type CacheWarmupTask struct {
	Name        string
	Key         string
	Priority    WarmupPriority
	Policy      WarmupPolicy
	TTL         time.Duration
	Frequency   time.Duration
	Loader      func(ctx context.Context) ([]byte, error)
	Condition   func(ctx context.Context) bool
	Enabled     bool
	MaxRetries  int
	RetryCount  int
	LastRun     time.Time
	LastSuccess time.Time
	Stats       *WarmupStats
	PreloadKeys []string
}

type WarmupStats struct {
	TotalRuns     atomic.Int64
	SuccessCount  atomic.Int64
	FailureCount  atomic.Int64
	TotalDuration atomic.Int64
	AvgDuration   atomic.Int64
	LastError     string
}

func NewWarmupStats() *WarmupStats {
	return &WarmupStats{}
}

type CacheWarmupManager struct {
	mu          sync.RWMutex
	tasks       map[string]*CacheWarmupTask
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	policy      WarmupPolicy
	concurrency int
	batchSize   int
}

type WarmupConfig struct {
	Policy      WarmupPolicy
	Concurrency int
	BatchSize   int
	Enabled     bool
}

var DefaultWarmupConfig = &WarmupConfig{
	Policy:      WarmupPolicyAdaptive,
	Concurrency: 5,
	BatchSize:   100,
	Enabled:     true,
}

type WarmupMetrics struct {
	TotalWarmupTasks   int32
	ActiveWarmupTasks  int32
	CompletedWarmupTasks int32
	FailedWarmupTasks  int32
	LastWarmupTime     time.Time
	AverageWarmupTime  time.Duration
}

type WarmupStatistics struct {
	TotalRuns      atomic.Int64
	SuccessCount   atomic.Int64
	FailureCount   atomic.Int64
	TotalDuration  atomic.Int64
	AvgDuration    atomic.Int64
	LastError      atomic.Value
	PeakMemory     atomic.Int64
	PeakKeys       atomic.Int64
}

func NewWarmupStatistics() *WarmupStatistics {
	return &WarmupStatistics{}
}

func NewCacheWarmupManager(config *WarmupConfig) *CacheWarmupManager {
	if config == nil {
		config = DefaultWarmupConfig
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &CacheWarmupManager{
		tasks:       make(map[string]*CacheWarmupTask),
		ctx:         ctx,
		cancel:      cancel,
		policy:      config.Policy,
		concurrency: config.Concurrency,
		batchSize:   config.BatchSize,
	}
}

func (cwm *CacheWarmupManager) AddTask(task *CacheWarmupTask) {
	cwm.mu.Lock()
	defer cwm.mu.Unlock()

	task.Stats = NewWarmupStats()
	cwm.tasks[task.Name] = task

	if cwm.running && task.Enabled {
		cwm.wg.Add(1)
		go cwm.runTask(task)
	}
}

func (cwm *CacheWarmupManager) RemoveTask(name string) {
	cwm.mu.Lock()
	defer cwm.mu.Unlock()
	delete(cwm.tasks, name)
}

func (cwm *CacheWarmupManager) EnableTask(name string) {
	cwm.mu.Lock()
	defer cwm.mu.Unlock()

	if task, ok := cwm.tasks[name]; ok {
		task.Enabled = true
		if cwm.running {
			cwm.wg.Add(1)
			go cwm.runTask(task)
		}
	}
}

func (cwm *CacheWarmupManager) DisableTask(name string) {
	cwm.mu.Lock()
	defer cwm.mu.Unlock()

	if task, ok := cwm.tasks[name]; ok {
		task.Enabled = false
	}
}

func (cwm *CacheWarmupManager) Start() {
	cwm.mu.Lock()
	defer cwm.mu.Unlock()

	if cwm.running {
		return
	}
	cwm.running = true

	for _, task := range cwm.tasks {
		if task.Enabled {
			cwm.wg.Add(1)
			go cwm.runTask(task)
		}
	}
}

func (cwm *CacheWarmupManager) Stop() {
	cwm.mu.Lock()
	if !cwm.running {
		cwm.mu.Unlock()
		return
	}
	cwm.running = false
	cwm.mu.Unlock()

	cwm.cancel()
	cwm.wg.Wait()
}

func (cwm *CacheWarmupManager) WarmupAll(ctx context.Context) error {
	cwm.mu.RLock()
	tasks := make([]*CacheWarmupTask, 0, len(cwm.tasks))
	for _, task := range cwm.tasks {
		if task.Enabled {
			tasks = append(tasks, task)
		}
	}
	cwm.mu.RUnlock()

	tasks = cwm.sortByPriority(tasks)

	if cwm.policy == WarmupPolicyEager {
		return cwm.warmupSequentially(ctx, tasks)
	}

	return cwm.warmupWithConcurrency(ctx, tasks)
}

func (cwm *CacheWarmupManager) WarmupTask(name string) error {
	cwm.mu.RLock()
	task, ok := cwm.tasks[name]
	cwm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("task not found: %s", name)
	}

	return cwm.executeTask(task)
}

func (cwm *CacheWarmupManager) runTask(task *CacheWarmupTask) {
	defer cwm.wg.Done()

	if task.Policy == WarmupPolicyLazy {
		return
	}

	ticker := time.NewTicker(task.Frequency)
	defer ticker.Stop()

	if err := cwm.executeTask(task); err != nil {
	}

	for {
		select {
		case <-cwm.ctx.Done():
			return
		case <-ticker.C:
			cwm.mu.RLock()
			enabled := task.Enabled
			cwm.mu.RUnlock()

			if !enabled {
				return
			}

			if task.Condition != nil && !task.Condition(cwm.ctx) {
				continue
			}

			if err := cwm.executeTask(task); err != nil {
			}
		}
	}
}

func (cwm *CacheWarmupManager) executeTask(task *CacheWarmupTask) error {
	start := time.Now()
	task.LastRun = start

	ctx, cancel := context.WithTimeout(cwm.ctx, 30*time.Second)
	defer cancel()

	data, err := task.Loader(ctx)
	if err != nil {
		task.Stats.FailureCount.Add(1)
		task.Stats.LastError = err.Error()
		task.RetryCount++

		if task.RetryCount < task.MaxRetries {
			go func() {
				time.Sleep(time.Duration(task.RetryCount) * time.Second)
				cwm.executeTask(task)
			}()
		}

		return err
	}

	if enhancedCache := GetEnhancedCache(); enhancedCache != nil {
		enhancedCache.Set(ctx, task.Key, data, &SetOptions{
			TTL:   task.TTL,
			Level: CacheLevelBoth,
		})
	} else if Client != nil {
		Client.Set(ctx, task.Key, data, task.TTL)
	}

	task.Stats.SuccessCount.Add(1)
	task.Stats.TotalRuns.Add(1)
	task.LastSuccess = time.Now()

	duration := time.Since(start)
	task.Stats.TotalDuration.Add(duration.Nanoseconds())
	task.Stats.AvgDuration.Store(task.Stats.TotalDuration.Load() / task.Stats.TotalRuns.Load())

	task.RetryCount = 0

	return nil
}

func (cwm *CacheWarmupManager) sortByPriority(tasks []*CacheWarmupTask) []*CacheWarmupTask {
	priorityMap := map[WarmupPriority]int{
		WarmupPriorityCritical: 0,
		WarmupPriorityHigh:     1,
		WarmupPriorityNormal:   2,
		WarmupPriorityLow:      3,
	}

	for i := 0; i < len(tasks)-1; i++ {
		for j := i + 1; j < len(tasks); j++ {
			if priorityMap[tasks[i].Priority] > priorityMap[tasks[j].Priority] {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}

	return tasks
}

func (cwm *CacheWarmupManager) warmupSequentially(ctx context.Context, tasks []*CacheWarmupTask) error {
	for _, task := range tasks {
		if err := cwm.executeTask(task); err != nil {
			continue
		}
	}
	return nil
}

func (cwm *CacheWarmupManager) warmupWithConcurrency(ctx context.Context, tasks []*CacheWarmupTask) error {
	semaphore := make(chan struct{}, cwm.concurrency)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var lastErr error

	for _, task := range tasks {
		wg.Add(1)
		go func(t *CacheWarmupTask) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := cwm.executeTask(t); err != nil {
				errMu.Lock()
				lastErr = err
				errMu.Unlock()
			}
		}(task)
	}

	wg.Wait()
	return lastErr
}

func (cwm *CacheWarmupManager) GetTasks() []*CacheWarmupTask {
	cwm.mu.RLock()
	defer cwm.mu.RUnlock()

	tasks := make([]*CacheWarmupTask, 0, len(cwm.tasks))
	for _, task := range cwm.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

func (cwm *CacheWarmupManager) GetTaskStats() map[string]*WarmupStats {
	cwm.mu.RLock()
	defer cwm.mu.RUnlock()

	stats := make(map[string]*WarmupStats)
	for name, task := range cwm.tasks {
		stats[name] = task.Stats
	}
	return stats
}

func (cwm *CacheWarmupManager) ResetTaskStats(name string) {
	cwm.mu.RLock()
	task, ok := cwm.tasks[name]
	cwm.mu.RUnlock()

	if ok {
		task.Stats = NewWarmupStats()
	}
}

func (cwm *CacheWarmupManager) WarmupByPriority(ctx context.Context, priority WarmupPriority) error {
	cwm.mu.RLock()
	tasks := make([]*CacheWarmupTask, 0, len(cwm.tasks))
	for _, task := range cwm.tasks {
		if task.Enabled && task.Priority == priority {
			tasks = append(tasks, task)
		}
	}
	cwm.mu.RUnlock()

	return cwm.warmupWithConcurrency(ctx, tasks)
}

func (cwm *CacheWarmupManager) WarmupByPattern(ctx context.Context, keys []string, loader func(ctx context.Context, key string) ([]byte, error), ttl time.Duration) error {
	if len(keys) == 0 {
		return nil
	}

	items := make([]*WarmupItem, 0, len(keys))
	for _, key := range keys {
		k := key
		items = append(items, &WarmupItem{
			Key:    k,
			Loader: func(ctx context.Context) ([]byte, error) {
				return loader(ctx, k)
			},
			TTL:    ttl,
		})
	}

	processor := NewBatchWarmupProcessor(cwm.batchSize, cwm.concurrency)
	return processor.Warmup(ctx, items)
}

func (cwm *CacheWarmupManager) GetWarmupStatus() map[string]interface{} {
	cwm.mu.RLock()
	defer cwm.mu.RUnlock()

	status := make(map[string]interface{})
	status["running"] = cwm.running
	status["total_tasks"] = len(cwm.tasks)

	enabledTasks := 0
	for _, task := range cwm.tasks {
		if task.Enabled {
			enabledTasks++
		}
	}
	status["enabled_tasks"] = enabledTasks

	return status
}

func (cwm *CacheWarmupManager) Pause() {
	cwm.mu.Lock()
	defer cwm.mu.Unlock()
	cwm.running = false
}

func (cwm *CacheWarmupManager) Resume() {
	cwm.mu.Lock()
	defer cwm.mu.Unlock()

	if !cwm.running {
		cwm.running = true
		for _, task := range cwm.tasks {
			if task.Enabled {
				cwm.wg.Add(1)
				go cwm.runTask(task)
			}
		}
	}
}

type WarmupPolicyHandler interface {
	ShouldWarmup(task *CacheWarmupTask) bool
	GetLoadPriority() []string
}

type AdaptiveWarmupPolicy struct {
	threshold int64
	tracker  *AccessTracker
}

func NewAdaptiveWarmupPolicy(threshold int64) *AdaptiveWarmupPolicy {
	return &AdaptiveWarmupPolicy{
		threshold: threshold,
		tracker:  NewAccessTracker(),
	}
}

func (awp *AdaptiveWarmupPolicy) RecordAccess(key string) {
	awp.tracker.RecordAccess(key)
}

func (awp *AdaptiveWarmupPolicy) ShouldWarmup(task *CacheWarmupTask) bool {
	for _, key := range task.PreloadKeys {
		if awp.tracker.GetAccessCount(key) >= awp.threshold {
			return true
		}
	}
	return false
}

func (awp *AdaptiveWarmupPolicy) GetLoadPriority() []string {
	return awp.tracker.GetHotKeys(awp.threshold)
}

type SmartWarmupStrategy struct {
	*CacheWarmupManager
	accessTracker *AccessTracker
	threshold     int64
}

type AccessTracker struct {
	mu          sync.RWMutex
	accessCount map[string]int64
	lastAccess  map[string]time.Time
}

func NewAccessTracker() *AccessTracker {
	return &AccessTracker{
		accessCount: make(map[string]int64),
		lastAccess:  make(map[string]time.Time),
	}
}

func (at *AccessTracker) RecordAccess(key string) {
	at.mu.Lock()
	defer at.mu.Unlock()

	at.accessCount[key]++
	at.lastAccess[key] = time.Now()

	if len(at.accessCount) > 10000 {
		at.cleanup()
	}
}

func (at *AccessTracker) GetAccessCount(key string) int64 {
	at.mu.RLock()
	defer at.mu.RUnlock()
	return at.accessCount[key]
}

func (at *AccessTracker) GetHotKeys(threshold int64) []string {
	at.mu.RLock()
	defer at.mu.RUnlock()

	var hotKeys []string
	for key, count := range at.accessCount {
		if count >= threshold {
			hotKeys = append(hotKeys, key)
		}
	}
	return hotKeys
}

func (at *AccessTracker) cleanup() {
	cutoff := time.Now().Add(-30 * time.Minute)
	for key, lastAccess := range at.lastAccess {
		if lastAccess.Before(cutoff) {
			delete(at.accessCount, key)
			delete(at.lastAccess, key)
		}
	}
}

func (at *AccessTracker) Reset() {
	at.mu.Lock()
	defer at.mu.Unlock()
	at.accessCount = make(map[string]int64)
	at.lastAccess = make(map[string]time.Time)
}

func NewSmartWarmupStrategy(config *WarmupConfig, threshold int64) *SmartWarmupStrategy {
	return &SmartWarmupStrategy{
		CacheWarmupManager: NewCacheWarmupManager(config),
		accessTracker:      NewAccessTracker(),
		threshold:          threshold,
	}
}

func (sw *SmartWarmupStrategy) RecordAccess(key string) {
	sw.accessTracker.RecordAccess(key)
}

func (sw *SmartWarmupStrategy) GetWarmupRecommendations() []string {
	return sw.accessTracker.GetHotKeys(sw.threshold)
}

func (sw *SmartWarmupStrategy) AdaptiveWarmup(ctx context.Context, loader func(ctx context.Context, key string) ([]byte, error)) error {
	recommendations := sw.GetWarmupRecommendations()

	for _, key := range recommendations {
		data, err := loader(ctx, key)
		if err != nil {
			continue
		}

		if enhancedCache := GetEnhancedCache(); enhancedCache != nil {
			enhancedCache.Set(ctx, key, data, &SetOptions{
				TTL:   30 * time.Minute,
				Level: CacheLevelBoth,
			})
		}
	}

	return nil
}

type BatchWarmupProcessor struct {
	cache      *EnhancedCache
	batchSize  int
	workers    int
}

func NewBatchWarmupProcessor(batchSize, workers int) *BatchWarmupProcessor {
	if batchSize <= 0 {
		batchSize = 100
	}
	if workers <= 0 {
		workers = 5
	}

	return &BatchWarmupProcessor{
		batchSize: batchSize,
		workers:   workers,
	}
}

func (bwp *BatchWarmupProcessor) Warmup(ctx context.Context, items []*WarmupItem) error {
	if len(items) == 0 {
		return nil
	}

	semaphore := make(chan struct{}, bwp.workers)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var lastErr error

	for i := 0; i < len(items); i += bwp.batchSize {
		end := i + bwp.batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		wg.Add(1)

		go func(batch []*WarmupItem) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := bwp.processBatch(ctx, batch); err != nil {
				errMu.Lock()
				lastErr = err
				errMu.Unlock()
			}
		}(batch)
	}

	wg.Wait()
	return lastErr
}

func (bwp *BatchWarmupProcessor) processBatch(ctx context.Context, batch []*WarmupItem) error {
	if bwp.cache == nil {
		bwp.cache = GetEnhancedCache()
	}

	for _, item := range batch {
		var data []byte
		var err error

		if item.Loader != nil {
			data, err = item.Loader(ctx)
			if err != nil {
				continue
			}
		} else {
			data = item.Value
		}

		ttl := item.TTL
		if ttl == 0 {
			ttl = 30 * time.Minute
		}

		if bwp.cache != nil {
			bwp.cache.Set(ctx, item.Key, data, &SetOptions{
				TTL:   ttl,
				Level: CacheLevelBoth,
			})
		}
	}

	return nil
}

var (
	globalWarmupManager *CacheWarmupManager
	globalWarmupOnce   sync.Once
)

func InitCacheWarmupManager(config *WarmupConfig) {
	globalWarmupOnce.Do(func() {
		globalWarmupManager = NewCacheWarmupManager(config)
	})
}

func GetCacheWarmupManager() *CacheWarmupManager {
	if globalWarmupManager == nil {
		InitCacheWarmupManager(nil)
	}
	return globalWarmupManager
}

func StartCacheWarmup() {
	GetCacheWarmupManager().Start()
}

func StopCacheWarmup() {
	GetCacheWarmupManager().Stop()
}

func AddCacheWarmupTask(task *CacheWarmupTask) {
	GetCacheWarmupManager().AddTask(task)
}

func WarmupAllCache() error {
	return GetCacheWarmupManager().WarmupAll(context.Background())
}
