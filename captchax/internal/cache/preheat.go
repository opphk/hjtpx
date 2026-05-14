package cache

import (
	"context"
	"log"
	"sync"
	"time"
)

type PreheatConfig struct {
	Enabled       bool
	Concurrency   int
	BatchSize     int
	RetryCount    int
	RetryInterval time.Duration
}

type PreheatLoader func(ctx context.Context, keys []string) (map[string][]byte, error)

type PreheatManager struct {
	config      *PreheatConfig
	preheatFunc PreheatLoader
	cache       *MultiLevelCache
	mu          sync.RWMutex
	isPreheating bool
	preheatKeys []string
	stats       *PreheatStats
}

type PreheatStats struct {
	TotalKeys     int
	LoadedKeys    int32
	FailedKeys    int32
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
}

type CacheWarmer interface {
	Preheat(ctx context.Context, keys []string) error
	PreheatWithPriority(ctx context.Context, keys [][]string) error
	GetStats() *PreheatStats
}

func NewPreheatManager(cache *MultiLevelCache, loader PreheatLoader, cfg *PreheatConfig) *PreheatManager {
	if cfg == nil {
		cfg = &PreheatConfig{
			Enabled:       true,
			Concurrency:   4,
			BatchSize:     100,
			RetryCount:    3,
			RetryInterval: 100 * time.Millisecond,
		}
	}

	return &PreheatManager{
		config:      cfg,
		preheatFunc: loader,
		cache:       cache,
		stats:       &PreheatStats{},
	}
}

func (pm *PreheatManager) Preheat(ctx context.Context, keys []string) error {
	if !pm.config.Enabled {
		return nil
	}

	pm.mu.Lock()
	if pm.isPreheating {
		pm.mu.Unlock()
		return nil
	}
	pm.isPreheating = true
	pm.stats = &PreheatStats{
		TotalKeys: len(keys),
		StartTime: time.Now(),
	}
	pm.preheatKeys = keys
	pm.mu.Unlock()

	defer func() {
		pm.mu.Lock()
		pm.isPreheating = false
		pm.stats.EndTime = time.Now()
		pm.stats.Duration = pm.stats.EndTime.Sub(pm.stats.StartTime)
		pm.mu.Unlock()
	}()

	return pm.preheatBatch(ctx, keys)
}

func (pm *PreheatManager) preheatBatch(ctx context.Context, keys []string) error {
	batchSize := pm.config.BatchSize
	concurrency := pm.config.Concurrency

	if len(keys) <= batchSize {
		return pm.preheatKeysInternal(ctx, keys)
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	errCh := make(chan error, len(keys)/batchSize+1)

	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]

		wg.Add(1)
		go func(b []string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := pm.preheatKeysInternal(ctx, b); err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}(batch)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

func (pm *PreheatManager) preheatKeysInternal(ctx context.Context, keys []string) error {
	if pm.preheatFunc == nil {
		return nil
	}

	var lastErr error
	for attempt := 0; attempt < pm.config.RetryCount; attempt++ {
		data, err := pm.preheatFunc(ctx, keys)
		if err == nil {
			for key, value := range data {
				if err := pm.cache.Set(ctx, key, value, 0); err != nil {
					lastErr = err
					continue
				}
				pm.stats.LoadedKeys++
			}
			return nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pm.config.RetryInterval):
		}
	}

	return lastErr
}

func (pm *PreheatManager) PreheatWithPriority(ctx context.Context, priorityGroups [][]string) error {
	for i, group := range priorityGroups {
		log.Printf("Preheating priority group %d with %d keys", i+1, len(group))
		if err := pm.Preheat(ctx, group); err != nil {
			return err
		}
	}
	return nil
}

func (pm *PreheatManager) GetStats() *PreheatStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.stats
}

func (pm *PreheatManager) IsPreheating() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.isPreheating
}

type LazyCacheLoader struct {
	loader     func(ctx context.Context, key string) ([]byte, error)
	cache      *MultiLevelCache
	ttl        time.Duration
	loading    map[string]chan struct{}
	loadingMu  sync.Mutex
}

func NewLazyCacheLoader(cache *MultiLevelCache, loader func(ctx context.Context, key string) ([]byte, error), ttl time.Duration) *LazyCacheLoader {
	return &LazyCacheLoader{
		loader:   loader,
		cache:    cache,
		ttl:      ttl,
		loading:  make(map[string]chan struct{}),
	}
}

func (lcl *LazyCacheLoader) Get(ctx context.Context, key string) ([]byte, bool, error) {
	data, ok := lcl.cache.Get(ctx, key)
	if ok {
		return data, true, nil
	}

	lcl.loadingMu.Lock()
	if _, exists := lcl.loading[key]; exists {
		lcl.loadingMu.Unlock()
		waitCh := make(chan struct{})
		defer close(waitCh)
		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		case <-waitCh:
		}
		data, ok = lcl.cache.Get(ctx, key)
		return data, ok, nil
	}
	lcl.loading[key] = make(chan struct{})
	lcl.loadingMu.Unlock()

	defer func() {
		lcl.loadingMu.Lock()
		delete(lcl.loading, key)
		close(lcl.loading[key])
		lcl.loadingMu.Unlock()
	}()

	data, err := lcl.loader(ctx, key)
	if err != nil {
		return nil, false, err
	}

	lcl.cache.Set(ctx, key, data, lcl.ttl)
	return data, true, nil
}

type BackgroundRefresher struct {
	cache       *MultiLevelCache
	interval    time.Duration
	keys        []string
	loader      func(ctx context.Context, key string) ([]byte, error)
	stopCh      chan struct{}
	mu          sync.RWMutex
}

func NewBackgroundRefresher(cache *MultiLevelCache, keys []string, loader func(ctx context.Context, key string) ([]byte, error), interval time.Duration) *BackgroundRefresher {
	return &BackgroundRefresher{
		cache:    cache,
		keys:     keys,
		loader:   loader,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (br *BackgroundRefresher) Start() {
	go br.run()
}

func (br *BackgroundRefresher) run() {
	ticker := time.NewTicker(br.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			br.refresh()
		case <-br.stopCh:
			return
		}
	}
}

func (br *BackgroundRefresher) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, key := range br.keys {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if data, err := br.loader(ctx, key); err == nil {
			br.cache.Set(ctx, key, data, 0)
		}
	}
}

func (br *BackgroundRefresher) Stop() {
	close(br.stopCh)
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
}
