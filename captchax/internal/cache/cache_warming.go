package cache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type CacheWarmingService struct {
	cache    *AdvancedCache
	warmers  map[string]WarmingFunc
	mu       sync.RWMutex
}

type WarmingFunc func(ctx context.Context) error

func NewCacheWarmingService(cache *AdvancedCache) *CacheWarmingService {
	return &CacheWarmingService{
		cache:   cache,
		warmers: make(map[string]WarmingFunc),
	}
}

func (w *CacheWarmingService) Register(name string, fn WarmingFunc) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.warmers[name] = fn
}

func (w *CacheWarmingService) Warm(ctx context.Context) error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var wg sync.WaitGroup
	var errs []error
	var mu sync.Mutex

	for name, fn := range w.warmers {
		wg.Add(1)
		go func(n string, f WarmingFunc) {
			defer wg.Done()
			if err := f(ctx); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", n, err))
				mu.Unlock()
			}
		}(name, fn)
	}

	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("cache warming errors: %v", errs)
	}

	return nil
}

func (w *CacheWarmingService) WarmPeriodically(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Warm(ctx); err != nil {
			}
		}
	}
}

func (w *CacheWarmingService) Unregister(name string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.warmers, name)
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
