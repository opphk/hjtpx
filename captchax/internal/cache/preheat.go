package cache

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type PreheatStrategy string

const (
	PreheatStrategyParallel PreheatStrategy = "parallel"
	PreheatStrategySerial   PreheatStrategy = "serial"
	PreheatStrategyGradual  PreheatStrategy = "gradual"
)

type PreheatConfig struct {
	Enabled           bool
	Concurrency       int
	BatchSize         int
	RetryCount        int
	RetryInterval     time.Duration
	MaxRetryInterval  time.Duration
	Strategy          PreheatStrategy
	GradualInterval   time.Duration
	KeyTimeout        time.Duration
	Logger            func(format string, args ...interface{})
}

type PreheatLoader func(ctx context.Context, keys []string) (map[string][]byte, error)

type KeyLoadResult struct {
	Key    string
	Value  []byte
	Error  error
	Loaded bool
}

type PreheatManager struct {
	config       *PreheatConfig
	preheatFunc  PreheatLoader
	cache        *MultiLevelCache
	mu           sync.RWMutex
	isPreheating bool
	preheatKeys  []string
	stats        *PreheatStats
	failedKeys   []string
	stopCh       chan struct{}
}

type PreheatStats struct {
	TotalKeys     int
	LoadedKeys    int32
	FailedKeys    int32
	SkippedKeys   int32
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	RetryCount    int32
}

type CacheWarmer interface {
	Preheat(ctx context.Context, keys []string) error
	PreheatWithPriority(ctx context.Context, keys [][]string) error
	GetStats() *PreheatStats
}

func NewPreheatManager(cache *MultiLevelCache, loader PreheatLoader, cfg *PreheatConfig) *PreheatManager {
	if cfg == nil {
		cfg = &PreheatConfig{
			Enabled:           true,
			Concurrency:       4,
			BatchSize:         100,
			RetryCount:        3,
			RetryInterval:     100 * time.Millisecond,
			MaxRetryInterval:  5 * time.Second,
			Strategy:          PreheatStrategyParallel,
			GradualInterval:   500 * time.Millisecond,
			KeyTimeout:        10 * time.Second,
			Logger:            func(format string, args ...interface{}) {},
		}
	}

	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 4
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.RetryCount <= 0 {
		cfg.RetryCount = 3
	}
	if cfg.RetryInterval <= 0 {
		cfg.RetryInterval = 100 * time.Millisecond
	}
	if cfg.MaxRetryInterval <= 0 {
		cfg.MaxRetryInterval = 5 * time.Second
	}
	if cfg.GradualInterval <= 0 {
		cfg.GradualInterval = 500 * time.Millisecond
	}
	if cfg.KeyTimeout <= 0 {
		cfg.KeyTimeout = 10 * time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger = func(format string, args ...interface{}) {}
	}

	return &PreheatManager{
		config:     cfg,
		preheatFunc: loader,
		cache:      cache,
		stats:      &PreheatStats{},
		failedKeys: make([]string, 0),
		stopCh:     make(chan struct{}),
	}
}

func (pm *PreheatManager) Preheat(ctx context.Context, keys []string) error {
	if !pm.config.Enabled {
		pm.config.Logger("preheat disabled, skipping")
		return nil
	}

	pm.mu.Lock()
	if pm.isPreheating {
		pm.mu.Unlock()
		return fmt.Errorf("preheat already in progress")
	}
	pm.isPreheating = true
	pm.stats = &PreheatStats{
		TotalKeys: len(keys),
		StartTime: time.Now(),
	}
	pm.preheatKeys = keys
	pm.failedKeys = make([]string, 0)
	pm.stopCh = make(chan struct{})
	pm.mu.Unlock()

	pm.config.Logger("starting preheat for %d keys", len(keys))

	defer func() {
		pm.mu.Lock()
		pm.isPreheating = false
		pm.stats.EndTime = time.Now()
		pm.stats.Duration = pm.stats.EndTime.Sub(pm.stats.StartTime)
		pm.mu.Unlock()
		pm.config.Logger("preheat completed: total=%d, loaded=%d, failed=%d, duration=%v", 
			pm.stats.TotalKeys, pm.stats.LoadedKeys, pm.stats.FailedKeys, pm.stats.Duration)
	}()

	switch pm.config.Strategy {
	case PreheatStrategySerial:
		return pm.preheatSerial(ctx, keys)
	case PreheatStrategyGradual:
		return pm.preheatGradual(ctx, keys)
	case PreheatStrategyParallel:
		fallthrough
	default:
		return pm.preheatBatch(ctx, keys)
	}
}

func (pm *PreheatManager) preheatSerial(ctx context.Context, keys []string) error {
	batchSize := pm.config.BatchSize
	var lastErr error

	for i := 0; i < len(keys); i += batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pm.stopCh:
			return fmt.Errorf("preheat stopped")
		default:
		}

		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]

		if err := pm.preheatKeysInternal(ctx, batch); err != nil {
			lastErr = err
			pm.config.Logger("batch preheat failed (keys %d-%d): %v", i, end-1, err)
		}
	}

	return lastErr
}

func (pm *PreheatManager) preheatGradual(ctx context.Context, keys []string) error {
	batchSize := pm.config.BatchSize
	var lastErr error

	for i := 0; i < len(keys); i += batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pm.stopCh:
			return fmt.Errorf("preheat stopped")
		default:
		}

		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]

		if err := pm.preheatKeysInternal(ctx, batch); err != nil {
			lastErr = err
			pm.config.Logger("batch preheat failed (keys %d-%d): %v", i, end-1, err)
		}

		if i+batchSize < len(keys) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-pm.stopCh:
				return fmt.Errorf("preheat stopped")
			case <-time.After(pm.config.GradualInterval):
			}
		}
	}

	return lastErr
}

func (pm *PreheatManager) preheatBatch(ctx context.Context, keys []string) error {
	batchSize := pm.config.BatchSize
	concurrency := pm.config.Concurrency

	if len(keys) <= batchSize {
		return pm.preheatKeysInternal(ctx, keys)
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var lastErr error

	for i := 0; i < len(keys); i += batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pm.stopCh:
			return fmt.Errorf("preheat stopped")
		default:
		}

		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]

		wg.Add(1)
		go func(b []string, batchIndex int) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			case <-pm.stopCh:
				return
			}
			defer func() { <-sem }()

			batchCtx, cancel := context.WithTimeout(ctx, pm.config.KeyTimeout)
			defer cancel()

			if err := pm.preheatKeysInternal(batchCtx, b); err != nil {
				errMu.Lock()
				lastErr = err
				errMu.Unlock()
				pm.config.Logger("batch preheat failed (batch %d): %v", batchIndex, err)
			}
		}(batch, i/batchSize)
	}

	wg.Wait()
	return lastErr
}

func (pm *PreheatManager) preheatKeysInternal(ctx context.Context, keys []string) error {
	if pm.preheatFunc == nil {
		return nil
	}

	var lastErr error
	
	for attempt := 0; attempt < pm.config.RetryCount; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pm.stopCh:
			return fmt.Errorf("preheat stopped")
		default:
		}

		attemptCtx, cancel := context.WithTimeout(ctx, pm.config.KeyTimeout)
		data, err := pm.preheatFunc(attemptCtx, keys)
		cancel()

		if err == nil {
			loadedCount := 0
			for key, value := range data {
				if err := pm.cache.Set(ctx, key, value, 0); err != nil {
					pm.config.Logger("failed to set key %s in cache: %v", key, err)
					pm.addFailedKey(key)
					continue
				}
				atomic.AddInt32(&pm.stats.LoadedKeys, 1)
				loadedCount++
			}
			if loadedCount > 0 {
				pm.config.Logger("loaded %d keys in batch", loadedCount)
			}
			return nil
		}

		lastErr = err
		atomic.AddInt32(&pm.stats.RetryCount, 1)
		
		if attempt < pm.config.RetryCount-1 {
			retryDelay := pm.calculateRetryDelay(attempt)
			pm.config.Logger("batch load failed (attempt %d/%d), retrying in %v: %v", 
				attempt+1, pm.config.RetryCount, retryDelay, err)
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-pm.stopCh:
				return fmt.Errorf("preheat stopped")
			case <-time.After(retryDelay):
			}
		}
	}

	for _, key := range keys {
		pm.addFailedKey(key)
	}

	return lastErr
}

func (pm *PreheatManager) calculateRetryDelay(attempt int) time.Duration {
	delay := time.Duration(math.Pow(2, float64(attempt))) * pm.config.RetryInterval
	if delay > pm.config.MaxRetryInterval {
		delay = pm.config.MaxRetryInterval
	}
	return delay
}

func (pm *PreheatManager) addFailedKey(key string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.failedKeys = append(pm.failedKeys, key)
	atomic.AddInt32(&pm.stats.FailedKeys, 1)
}

func (pm *PreheatManager) PreheatWithPriority(ctx context.Context, priorityGroups [][]string) error {
	for i, group := range priorityGroups {
		pm.config.Logger("preheating priority group %d with %d keys", i+1, len(group))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pm.stopCh:
			return fmt.Errorf("preheat stopped")
		default:
		}

		if err := pm.Preheat(ctx, group); err != nil {
			pm.config.Logger("priority group %d preheat failed: %v", i+1, err)
			return err
		}
	}
	return nil
}

func (pm *PreheatManager) RetryFailedKeys(ctx context.Context) error {
	pm.mu.RLock()
	failedKeys := make([]string, len(pm.failedKeys))
	copy(failedKeys, pm.failedKeys)
	pm.mu.RUnlock()

	if len(failedKeys) == 0 {
		pm.config.Logger("no failed keys to retry")
		return nil
	}

	pm.config.Logger("retrying %d failed keys", len(failedKeys))
	return pm.Preheat(ctx, failedKeys)
}

func (pm *PreheatManager) Stop() {
	pm.mu.Lock()
	select {
	case <-pm.stopCh:
	default:
		close(pm.stopCh)
	}
	pm.mu.Unlock()
	pm.config.Logger("preheat stop requested")
}

func (pm *PreheatManager) GetStats() *PreheatStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	stats := *pm.stats
	return &stats
}

func (pm *PreheatManager) GetFailedKeys() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	keys := make([]string, len(pm.failedKeys))
	copy(keys, pm.failedKeys)
	return keys
}

func (pm *PreheatManager) IsPreheating() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.isPreheating
}

func (pm *PreheatManager) SetLoader(loader PreheatLoader) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.preheatFunc = loader
}

func (pm *PreheatManager) UpdateConfig(cfg *PreheatConfig) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if cfg != nil {
		pm.config = cfg
	}
}

type LazyCacheLoader struct {
	loader     func(ctx context.Context, key string) ([]byte, error)
	cache      *MultiLevelCache
	ttl        time.Duration
	loading    map[string]*loadState
	loadingMu  sync.Mutex
	logger     func(format string, args ...interface{})
}

type loadState struct {
	done chan struct{}
	err  error
}

func NewLazyCacheLoader(cache *MultiLevelCache, loader func(ctx context.Context, key string) ([]byte, error), ttl time.Duration) *LazyCacheLoader {
	return &LazyCacheLoader{
		loader:  loader,
		cache:   cache,
		ttl:     ttl,
		loading: make(map[string]*loadState),
		logger:  func(format string, args ...interface{}) {},
	}
}

func (lcl *LazyCacheLoader) SetLogger(logger func(format string, args ...interface{})) {
	lcl.logger = logger
}

func (lcl *LazyCacheLoader) Get(ctx context.Context, key string) ([]byte, bool, error) {
	data, ok := lcl.cache.Get(ctx, key)
	if ok {
		return data, true, nil
	}

	lcl.loadingMu.Lock()
	if state, exists := lcl.loading[key]; exists {
		lcl.loadingMu.Unlock()
		lcl.logger("waiting for concurrent load of key: %s", key)
		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		case <-state.done:
			if state.err != nil {
				return nil, false, state.err
			}
			data, ok = lcl.cache.Get(ctx, key)
			return data, ok, nil
		}
	}
	
	state := &loadState{done: make(chan struct{})}
	lcl.loading[key] = state
	lcl.loadingMu.Unlock()

	defer func() {
		lcl.loadingMu.Lock()
		delete(lcl.loading, key)
		close(state.done)
		lcl.loadingMu.Unlock()
	}()

	lcl.logger("loading key: %s", key)
	data, err := lcl.loader(ctx, key)
	if err != nil {
		state.err = err
		lcl.logger("failed to load key %s: %v", key, err)
		return nil, false, err
	}

	lcl.cache.Set(ctx, key, data, lcl.ttl)
	lcl.logger("loaded and cached key: %s", key)
	return data, true, nil
}

type BackgroundRefresher struct {
	cache        *MultiLevelCache
	interval     time.Duration
	keys         []string
	loader       func(ctx context.Context, key string) ([]byte, error)
	stopCh       chan struct{}
	mu           sync.RWMutex
	running      bool
	refreshStats *RefreshStats
	logger       func(format string, args ...interface{})
}

type RefreshStats struct {
	TotalRefreshes int64
	SuccessCount   int64
	FailureCount   int64
	LastRefresh    time.Time
}

func NewBackgroundRefresher(cache *MultiLevelCache, keys []string, loader func(ctx context.Context, key string) ([]byte, error), interval time.Duration) *BackgroundRefresher {
	return &BackgroundRefresher{
		cache:        cache,
		keys:         keys,
		loader:       loader,
		interval:     interval,
		stopCh:       make(chan struct{}),
		refreshStats: &RefreshStats{},
		logger:       func(format string, args ...interface{}) {},
	}
}

func (br *BackgroundRefresher) SetLogger(logger func(format string, args ...interface{})) {
	br.logger = logger
}

func (br *BackgroundRefresher) Start() {
	br.mu.Lock()
	if br.running {
		br.mu.Unlock()
		return
	}
	br.running = true
	br.stopCh = make(chan struct{})
	br.mu.Unlock()

	br.logger("starting background refresher with interval: %v", br.interval)
	go br.run()
}

func (br *BackgroundRefresher) run() {
	ticker := time.NewTicker(br.interval)
	defer ticker.Stop()

	br.refresh()

	for {
		select {
		case <-ticker.C:
			br.refresh()
		case <-br.stopCh:
			br.mu.Lock()
			br.running = false
			br.mu.Unlock()
			br.logger("background refresher stopped")
			return
		}
	}
}

func (br *BackgroundRefresher) refresh() {
	br.mu.RLock()
	keys := make([]string, len(br.keys))
	copy(keys, br.keys)
	br.mu.RUnlock()

	if len(keys) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	br.logger("starting refresh of %d keys", len(keys))
	
	successCount := 0
	failureCount := 0

	for _, key := range keys {
		select {
		case <-ctx.Done():
			br.logger("refresh cancelled due to context timeout")
			return
		default:
		}

		if data, err := br.loader(ctx, key); err == nil {
			br.cache.Set(ctx, key, data, 0)
			successCount++
		} else {
			failureCount++
			br.logger("failed to refresh key %s: %v", key, err)
		}
	}

	br.mu.Lock()
	br.refreshStats.TotalRefreshes++
	br.refreshStats.SuccessCount += int64(successCount)
	br.refreshStats.FailureCount += int64(failureCount)
	br.refreshStats.LastRefresh = time.Now()
	br.mu.Unlock()

	br.logger("refresh completed: success=%d, failure=%d", successCount, failureCount)
}

func (br *BackgroundRefresher) Stop() {
	br.mu.Lock()
	if !br.running {
		br.mu.Unlock()
		return
	}
	select {
	case <-br.stopCh:
	default:
		close(br.stopCh)
	}
	br.mu.Unlock()
}

func (br *BackgroundRefresher) AddKey(key string) {
	br.mu.Lock()
	defer br.mu.Unlock()

	for _, k := range br.keys {
		if k == key {
			return
		}
	}
	br.keys = append(br.keys, key)
	br.logger("added key to refresher: %s", key)
}

func (br *BackgroundRefresher) RemoveKey(key string) {
	br.mu.Lock()
	defer br.mu.Unlock()

	newKeys := make([]string, 0, len(br.keys))
	for _, k := range br.keys {
		if k != key {
			newKeys = append(newKeys, k)
		}
	}
	br.keys = newKeys
	br.logger("removed key from refresher: %s", key)
}

func (br *BackgroundRefresher) GetStats() *RefreshStats {
	br.mu.RLock()
	defer br.mu.RUnlock()
	stats := *br.refreshStats
	return &stats
}

func (br *BackgroundRefresher) IsRunning() bool {
	br.mu.RLock()
	defer br.mu.RUnlock()
	return br.running
}

func (br *BackgroundRefresher) UpdateInterval(interval time.Duration) {
	br.mu.Lock()
	defer br.mu.Unlock()
	br.interval = interval
	br.logger("updated refresh interval to: %v", interval)
}
