package cache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type WarmerStatus string

const (
	WarmerStatusIdle    WarmerStatus = "idle"
	WarmerStatusRunning WarmerStatus = "running"
	WarmerStatusFailed  WarmerStatus = "failed"
	WarmerStatusSuccess WarmerStatus = "success"
)

type WarmerInfo struct {
	Name        string
	Status      WarmerStatus
	LastRun     time.Time
	LastSuccess time.Time
	LastError   string
	RunCount    int64
	SuccessCount int64
	FailureCount int64
	AvgDuration time.Duration
}

type WarmingResult struct {
	Name     string
	Duration time.Duration
	Error    error
}

type WarmingFunc func(ctx context.Context) error

type CacheWarmingService struct {
	cache       *AdvancedCache
	warmers     map[string]WarmingFunc
	warmerInfo  map[string]*WarmerInfo
	mu          sync.RWMutex
	isWarming   bool
	results     []WarmingResult
	logger      func(format string, args ...interface{})
}

func NewCacheWarmingService(cache *AdvancedCache) *CacheWarmingService {
	return &CacheWarmingService{
		cache:       cache,
		warmers:     make(map[string]WarmingFunc),
		warmerInfo:  make(map[string]*WarmerInfo),
		logger:      func(format string, args ...interface{}) {},
	}
}

func (w *CacheWarmingService) SetLogger(logger func(format string, args ...interface{})) {
	w.logger = logger
}

func (w *CacheWarmingService) Register(name string, fn WarmingFunc) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.warmers[name] = fn
	if _, exists := w.warmerInfo[name]; !exists {
		w.warmerInfo[name] = &WarmerInfo{
			Name:   name,
			Status: WarmerStatusIdle,
		}
	}
	w.logger("registered cache warmer: %s", name)
}

func (w *CacheWarmingService) Warm(ctx context.Context) ([]WarmingResult, error) {
	w.mu.Lock()
	if w.isWarming {
		w.mu.Unlock()
		return nil, fmt.Errorf("cache warming already in progress")
	}
	w.isWarming = true
	w.results = nil
	warmers := make(map[string]WarmingFunc)
	for name, fn := range w.warmers {
		warmers[name] = fn
	}
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.isWarming = false
		w.mu.Unlock()
	}()

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]WarmingResult, 0, len(warmers))
	errs := make([]error, 0)

	semaphore := make(chan struct{}, 5)

	for name, fn := range warmers {
		wg.Add(1)
		go func(n string, f WarmingFunc) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			start := time.Now()
			w.updateWarmerStatus(n, WarmerStatusRunning)
			
			w.logger("starting cache warmer: %s", n)
			err := f(ctx)
			duration := time.Since(start)
			
			result := WarmingResult{
				Name:     n,
				Duration: duration,
				Error:    err,
			}

			mu.Lock()
			results = append(results, result)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", n, err))
				w.updateWarmerFailure(n, err, duration)
				w.logger("cache warmer failed: %s, error: %v, duration: %v", n, err, duration)
			} else {
				w.updateWarmerSuccess(n, duration)
				w.logger("cache warmer completed: %s, duration: %v", n, duration)
			}
			mu.Unlock()
		}(name, fn)
	}

	wg.Wait()

	w.mu.Lock()
	w.results = results
	w.mu.Unlock()

	if len(errs) > 0 {
		return results, fmt.Errorf("cache warming completed with %d errors: %v", len(errs), errs)
	}

	return results, nil
}

func (w *CacheWarmingService) WarmWithTimeout(ctx context.Context, timeout time.Duration) ([]WarmingResult, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return w.Warm(ctxWithTimeout)
}

func (w *CacheWarmingService) WarmSingle(ctx context.Context, name string) (WarmingResult, error) {
	w.mu.RLock()
	fn, exists := w.warmers[name]
	w.mu.RUnlock()

	if !exists {
		return WarmingResult{Name: name}, fmt.Errorf("warmer not found: %s", name)
	}

	start := time.Now()
	w.updateWarmerStatus(name, WarmerStatusRunning)
	w.logger("starting single cache warmer: %s", name)
	
	err := fn(ctx)
	duration := time.Since(start)
	
	result := WarmingResult{
		Name:     name,
		Duration: duration,
		Error:    err,
	}

	if err != nil {
		w.updateWarmerFailure(name, err, duration)
		w.logger("single cache warmer failed: %s, error: %v", name, err)
	} else {
		w.updateWarmerSuccess(name, duration)
		w.logger("single cache warmer completed: %s, duration: %v", name, duration)
	}

	return result, err
}

func (w *CacheWarmingService) WarmPeriodically(ctx context.Context, interval time.Duration) {
	w.logger("starting periodic cache warming with interval: %v", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger("periodic cache warming stopped")
			return
		case <-ticker.C:
			warmCtx, cancel := context.WithTimeout(ctx, interval)
			if _, err := w.Warm(warmCtx); err != nil {
				w.logger("periodic cache warming error: %v", err)
			}
			cancel()
		}
	}
}

func (w *CacheWarmingService) Unregister(name string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.warmers, name)
	delete(w.warmerInfo, name)
	w.logger("unregistered cache warmer: %s", name)
}

func (w *CacheWarmingService) ListWarmers() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	names := make([]string, 0, len(w.warmers))
	for name := range w.warmers {
		names = append(names, name)
	}
	return names
}

func (w *CacheWarmingService) GetWarmerInfo(name string) (*WarmerInfo, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	info, exists := w.warmerInfo[name]
	return info, exists
}

func (w *CacheWarmingService) GetAllWarmerInfo() []WarmerInfo {
	w.mu.RLock()
	defer w.mu.RUnlock()
	infos := make([]WarmerInfo, 0, len(w.warmerInfo))
	for _, info := range w.warmerInfo {
		infos = append(infos, *info)
	}
	return infos
}

func (w *CacheWarmingService) IsWarming() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.isWarming
}

func (w *CacheWarmingService) GetLastResults() []WarmingResult {
	w.mu.RLock()
	defer w.mu.RUnlock()
	results := make([]WarmingResult, len(w.results))
	copy(results, w.results)
	return results
}

func (w *CacheWarmingService) updateWarmerStatus(name string, status WarmerStatus) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if info, exists := w.warmerInfo[name]; exists {
		info.Status = status
	}
}

func (w *CacheWarmingService) updateWarmerSuccess(name string, duration time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if info, exists := w.warmerInfo[name]; exists {
		info.Status = WarmerStatusSuccess
		info.LastRun = time.Now()
		info.LastSuccess = time.Now()
		info.LastError = ""
		info.RunCount++
		info.SuccessCount++
		totalDuration := info.AvgDuration * time.Duration(info.SuccessCount-1)
		info.AvgDuration = (totalDuration + duration) / time.Duration(info.SuccessCount)
	}
}

func (w *CacheWarmingService) updateWarmerFailure(name string, err error, duration time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if info, exists := w.warmerInfo[name]; exists {
		info.Status = WarmerStatusFailed
		info.LastRun = time.Now()
		info.LastError = err.Error()
		info.RunCount++
		info.FailureCount++
	}
}
