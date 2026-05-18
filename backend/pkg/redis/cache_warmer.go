package redis

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type WarmupTask struct {
	Name      string
	Key       string
	TTL       time.Duration
	Frequency time.Duration
	Loader    func(ctx context.Context) ([]byte, error)
	Enabled   bool
}

type CacheWarmer struct {
	tasks   map[string]*WarmupTask
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	cache   *EnhancedCache
	running bool
	status  WarmerStatus
}

type WarmerStatus struct {
	Running          bool          `json:"running"`
	TaskCount        int           `json:"task_count"`
	LastWarmupTime   time.Time     `json:"last_warmup_time"`
	TotalWarmups     int64         `json:"total_warmups"`
	FailedWarmups    int64         `json:"failed_warmups"`
	CurrentWarmups   int32         `json:"current_warmups"`
}

func NewCacheWarmer(cache *EnhancedCache) *CacheWarmer {
	if cache == nil {
		cache = GetEnhancedCache()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &CacheWarmer{
		tasks:  make(map[string]*WarmupTask),
		ctx:    ctx,
		cancel: cancel,
		cache:  cache,
		status: WarmerStatus{
			Running: false,
			TaskCount: 0,
		},
	}
}

func (cw *CacheWarmer) AddTask(task *WarmupTask) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	cw.tasks[task.Name] = task

	if cw.running && task.Enabled {
		cw.wg.Add(1)
		go cw.runTask(task)
	}
}

func (cw *CacheWarmer) RemoveTask(name string) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	delete(cw.tasks, name)
}

func (cw *CacheWarmer) EnableTask(name string) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if task, ok := cw.tasks[name]; ok {
		task.Enabled = true
		if cw.running {
			cw.wg.Add(1)
			go cw.runTask(task)
		}
	}
}

func (cw *CacheWarmer) DisableTask(name string) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if task, ok := cw.tasks[name]; ok {
		task.Enabled = false
	}
}

func (cw *CacheWarmer) Start() {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if cw.running {
		return
	}

	cw.running = true

	for _, task := range cw.tasks {
		if task.Enabled {
			cw.wg.Add(1)
			go cw.runTask(task)
		}
	}
}

func (cw *CacheWarmer) Stop() {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if !cw.running {
		return
	}

	cw.cancel()
	cw.wg.Wait()

	cw.ctx, cw.cancel = context.WithCancel(context.Background())
	cw.running = false
}

func (cw *CacheWarmer) WarmupAll() error {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	for _, task := range cw.tasks {
		if task.Enabled {
			if err := cw.executeTask(task); err != nil {
				continue
			}
		}
	}

	return nil
}

func (cw *CacheWarmer) WarmupTask(name string) error {
	cw.mu.RLock()
	task, ok := cw.tasks[name]
	cw.mu.RUnlock()

	if !ok {
		return ErrKeyNotFound
	}

	return cw.executeTask(task)
}

func (cw *CacheWarmer) runTask(task *WarmupTask) {
	defer cw.wg.Done()

	ticker := time.NewTicker(task.Frequency)
	defer ticker.Stop()

	if err := cw.executeTask(task); err != nil {
	}

	for {
		select {
		case <-cw.ctx.Done():
			return
		case <-ticker.C:
			cw.mu.RLock()
			enabled := task.Enabled
			cw.mu.RUnlock()

			if !enabled {
				return
			}

			if err := cw.executeTask(task); err != nil {
			}
		}
	}
}

func (cw *CacheWarmer) executeTask(task *WarmupTask) error {
	ctx, cancel := context.WithTimeout(cw.ctx, 30*time.Second)
	defer cancel()

	atomic.AddInt32(&cw.status.CurrentWarmups, 1)
	defer atomic.AddInt32(&cw.status.CurrentWarmups, -1)

	data, err := task.Loader(ctx)
	if err != nil {
		cw.mu.Lock()
		cw.status.FailedWarmups++
		cw.mu.Unlock()
		return err
	}

	err = cw.cache.Set(ctx, task.Key, data, &SetOptions{
		TTL:   task.TTL,
		Level: CacheLevelBoth,
	})

	cw.mu.Lock()
	cw.status.TotalWarmups++
	cw.status.LastWarmupTime = time.Now()
	cw.mu.Unlock()

	return err
}

func (cw *CacheWarmer) GetStatus() WarmerStatus {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	
	status := cw.status
	status.TaskCount = len(cw.tasks)
	status.Running = cw.running
	return status
}

func (cw *CacheWarmer) Warmup() error {
	return cw.WarmupAll()
}

func (cw *CacheWarmer) GetTasks() []*WarmupTask {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	tasks := make([]*WarmupTask, 0, len(cw.tasks))
	for _, task := range cw.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

type SmartWarmer struct {
	*CacheWarmer
	accessCounts map[string]int64
	threshold    int64
	mu           sync.RWMutex
}

func NewSmartWarmer(cache *EnhancedCache, threshold int64) *SmartWarmer {
	return &SmartWarmer{
		CacheWarmer:  NewCacheWarmer(cache),
		accessCounts: make(map[string]int64),
		threshold:    threshold,
	}
}

func (sw *SmartWarmer) RecordAccess(key string) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.accessCounts[key]++
}

func (sw *SmartWarmer) GetHotKeys() []string {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	var hotKeys []string
	for key, count := range sw.accessCounts {
		if count >= sw.threshold {
			hotKeys = append(hotKeys, key)
		}
	}
	return hotKeys
}

func (sw *SmartWarmer) ResetCounts() {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.accessCounts = make(map[string]int64)
}

func (sw *SmartWarmer) SmartWarmup(ctx context.Context, loader func(ctx context.Context, key string) ([]byte, error)) error {
	hotKeys := sw.GetHotKeys()

	for _, key := range hotKeys {
		data, err := loader(ctx, key)
		if err != nil {
			continue
		}

		if err := sw.cache.Set(ctx, key, data, &SetOptions{
			TTL:   sw.cache.config.L2TTL,
			Level: CacheLevelBoth,
		}); err != nil {
			continue
		}
	}

	return nil
}

type CacheRefreshStrategy int

const (
	RefreshStrategyFixed CacheRefreshStrategy = iota
	RefreshStrategyAdaptive
	RefreshStrategyNever
)

type AdaptiveRefresher struct {
	cache           *EnhancedCache
	strategy        CacheRefreshStrategy
	refreshWindow   time.Duration
	accessThreshold int
	mu              sync.RWMutex
	keyStats        map[string]*keyStat
}

type keyStat struct {
	accessCount    int
	lastAccess     time.Time
	refreshCount   int
	avgRefreshTime time.Duration
}

func NewAdaptiveRefresher(cache *EnhancedCache) *AdaptiveRefresher {
	if cache == nil {
		cache = GetEnhancedCache()
	}

	return &AdaptiveRefresher{
		cache:           cache,
		strategy:        RefreshStrategyAdaptive,
		refreshWindow:   10 * time.Minute,
		accessThreshold: 5,
		keyStats:        make(map[string]*keyStat),
	}
}

func (ar *AdaptiveRefresher) RecordAccess(key string) {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	stat, ok := ar.keyStats[key]
	if !ok {
		stat = &keyStat{
			lastAccess: time.Now(),
		}
		ar.keyStats[key] = stat
	}

	stat.accessCount++
	stat.lastAccess = time.Now()
}

func (ar *AdaptiveRefresher) ShouldRefresh(key string, ttl time.Duration) bool {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	stat, ok := ar.keyStats[key]
	if !ok {
		return false
	}

	switch ar.strategy {
	case RefreshStrategyFixed:
		return true
	case RefreshStrategyAdaptive:
		remaining := ttl - time.Since(stat.lastAccess)
		return remaining < ar.refreshWindow && stat.accessCount >= ar.accessThreshold
	case RefreshStrategyNever:
		return false
	default:
		return false
	}
}

func (ar *AdaptiveRefresher) CalculateTTL(key string, baseTTL time.Duration) time.Duration {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	stat, ok := ar.keyStats[key]
	if !ok {
		return baseTTL
	}

	if stat.accessCount > ar.accessThreshold*2 {
		return baseTTL * 2
	}

	return baseTTL
}

func (ar *AdaptiveRefresher) Cleanup() {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	now := time.Now()
	for key, stat := range ar.keyStats {
		if now.Sub(stat.lastAccess) > 24*time.Hour {
			delete(ar.keyStats, key)
		}
	}
}

type BatchWarmer struct {
	cache       *EnhancedCache
	batchSize   int
	concurrency int
}

func NewBatchWarmer(cache *EnhancedCache, batchSize, concurrency int) *BatchWarmer {
	if cache == nil {
		cache = GetEnhancedCache()
	}

	if batchSize <= 0 {
		batchSize = 100
	}
	if concurrency <= 0 {
		concurrency = 5
	}

	return &BatchWarmer{
		cache:       cache,
		batchSize:   batchSize,
		concurrency: concurrency,
	}
}

func (bw *BatchWarmer) Warmup(ctx context.Context, items []*WarmupItem) error {
	semaphore := make(chan struct{}, bw.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for i := 0; i < len(items); i += bw.batchSize {
		end := i + bw.batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]

		wg.Add(1)
		go func(batch []*WarmupItem) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			batchErr := bw.warmupBatch(ctx, batch)
			if batchErr != nil {
				mu.Lock()
				errs = append(errs, batchErr)
				mu.Unlock()
			}
		}(batch)
	}

	wg.Wait()

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (bw *BatchWarmer) warmupBatch(ctx context.Context, batch []*WarmupItem) error {
	cacheItems := make(map[string][]byte)
	ttls := make(map[string]time.Duration)

	for _, item := range batch {
		var value []byte
		var err error

		if item.Loader != nil {
			value, err = item.Loader(ctx)
			if err != nil {
				continue
			}
		} else {
			value = item.Value
		}

		cacheItems[item.Key] = value
		ttls[item.Key] = item.TTL
	}

	if len(cacheItems) > 0 {
		for key, value := range cacheItems {
			ttl := ttls[key]
			if ttl == 0 {
				ttl = bw.cache.config.L2TTL
			}
			if err := bw.cache.Set(ctx, key, value, &SetOptions{TTL: ttl}); err != nil {
				continue
			}
		}
	}

	return nil
}

var (
	globalWarmer     *CacheWarmer
	globalWarmerOnce sync.Once
)

func GetGlobalWarmer() *CacheWarmer {
	globalWarmerOnce.Do(func() {
		globalWarmer = NewCacheWarmer(nil)
	})
	return globalWarmer
}

func StartCacheWarmer() {
	GetGlobalWarmer().Start()
}

func StopCacheWarmer() {
	GetGlobalWarmer().Stop()
}

func AddWarmupTask(task *WarmupTask) {
	GetGlobalWarmer().AddTask(task)
}

func GetCacheWarmer() *CacheWarmer {
	return GetGlobalWarmer()
}
