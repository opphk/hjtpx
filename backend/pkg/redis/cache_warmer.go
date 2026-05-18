package redis

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type CacheWarmer struct {
	client      *goredis.Client
	tasks       map[string]*WarmupTask
	intervals   map[string]time.Duration
	stopCh      chan struct{}
	stoppedCh   chan struct{}
	running     bool
	mu          sync.RWMutex
	stats       *WarmupStats
}

type WarmupTask struct {
	Name         string
	KeyPrefix    string
	Loader       func(context.Context, *goredis.Client) (map[string]string, error)
	Interval     time.Duration
	Priority     int
	Enabled      bool
	ParallelLoad bool
}

type WarmupStats struct {
	TotalLoads      int64
	TotalKeys       int64
	TotalErrors     int64
	LastLoadTime    time.Time
	LastLoadCount   int64
	LastErrorCount  int64
	LoadDurations   []time.Duration
	mu              sync.RWMutex
}

type WarmupOption func(*CacheWarmer)

func WithWarmupInterval(keyPrefix string, interval time.Duration) WarmupOption {
	return func(cw *CacheWarmer) {
		cw.intervals[keyPrefix] = interval
	}
}

func NewCacheWarmer(client *goredis.Client) *CacheWarmer {
	return &CacheWarmer{
		client:   client,
		tasks:    make(map[string]*WarmupTask),
		intervals: make(map[string]time.Duration),
		stopCh:   make(chan struct{}),
		stoppedCh: make(chan struct{}),
		stats: &WarmupStats{
			LoadDurations: make([]time.Duration, 0),
		},
	}
}

func (cw *CacheWarmer) RegisterTask(task *WarmupTask) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if task.Interval == 0 {
		task.Interval = 5 * time.Minute
	}
	if task.Loader == nil {
		log.Printf("[CACHE_WARMER] Task %s has no loader, skipping", task.Name)
		return
	}

	cw.tasks[task.KeyPrefix] = task
	if interval, ok := cw.intervals[task.KeyPrefix]; ok {
		task.Interval = interval
	}

	log.Printf("[CACHE_WARMER] Registered task: %s (prefix: %s, interval: %s)",
		task.Name, task.KeyPrefix, task.Interval)
}

func (cw *CacheWarmer) RegisterDefaultTasks() {
	cw.RegisterTask(&WarmupTask{
		Name:      "config_cache",
		KeyPrefix: "config:",
		Loader: func(ctx context.Context, client *goredis.Client) (map[string]string, error) {
			return cw.loadConfigCache(ctx, client)
		},
		Interval:     10 * time.Minute,
		Priority:     1,
		Enabled:      true,
		ParallelLoad: true,
	})

	cw.RegisterTask(&WarmupTask{
		Name:      "captcha_templates",
		KeyPrefix: "captcha:template:",
		Loader: func(ctx context.Context, client *goredis.Client) (map[string]string, error) {
			return cw.loadCaptchaTemplates(ctx, client)
		},
		Interval:     30 * time.Minute,
		Priority:     2,
		Enabled:      true,
		ParallelLoad: true,
	})

	cw.RegisterTask(&WarmupTask{
		Name:      "application_configs",
		KeyPrefix: "app:",
		Loader: func(ctx context.Context, client *goredis.Client) (map[string]string, error) {
			return cw.loadApplicationConfigs(ctx, client)
		},
		Interval:     15 * time.Minute,
		Priority:     3,
		Enabled:      true,
		ParallelLoad: false,
	})

	cw.RegisterTask(&WarmupTask{
		Name:      "whitelist",
		KeyPrefix: "whitelist:",
		Loader: func(ctx context.Context, client *goredis.Client) (map[string]string, error) {
			return cw.loadWhitelist(ctx, client)
		},
		Interval:     20 * time.Minute,
		Priority:     4,
		Enabled:      true,
		ParallelLoad: true,
	})

	cw.RegisterTask(&WarmupTask{
		Name:      "blacklist",
		KeyPrefix: "blacklist:",
		Loader: func(ctx context.Context, client *goredis.Client) (map[string]string, error) {
			return cw.loadBlacklist(ctx, client)
		},
		Interval:     10 * time.Minute,
		Priority:     5,
		Enabled:      true,
		ParallelLoad: true,
	})

	cw.RegisterTask(&WarmupTask{
		Name:      "rate_limit_rules",
		KeyPrefix: "ratelimit:rule:",
		Loader: func(ctx context.Context, client *goredis.Client) (map[string]string, error) {
			return cw.loadRateLimitRules(ctx, client)
		},
		Interval:     5 * time.Minute,
		Priority:     1,
		Enabled:      true,
		ParallelLoad: false,
	})
}

func (cw *CacheWarmer) Start(ctx context.Context) {
	cw.mu.Lock()
	if cw.running {
		cw.mu.Unlock()
		return
	}
	cw.running = true
	cw.mu.Unlock()

	log.Println("[CACHE_WARMER] Starting cache warmer...")

	go cw.runWarmupLoop(ctx)

	go func() {
		for _, task := range cw.tasks {
			if task.Enabled {
				go cw.warmupTask(ctx, task)
			}
		}
	}()

	log.Printf("[CACHE_WARMER] Started with %d registered tasks", len(cw.tasks))
}

func (cw *CacheWarmer) Stop() {
	cw.mu.Lock()
	if !cw.running {
		cw.mu.Unlock()
		return
	}
	cw.running = false
	cw.mu.Unlock()

	close(cw.stopCh)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	select {
	case <-cw.stoppedCh:
	case <-ticker.C:
	}

	log.Println("[CACHE_WARMER] Stopped")
}

func (cw *CacheWarmer) runWarmupLoop(ctx context.Context) {
	defer close(cw.stoppedCh)

	initialDelay := time.NewTimer(5 * time.Second)
	defer initialDelay.Stop()

	<-initialDelay.C

	log.Println("[CACHE_WARMER] Running initial warmup...")

	var wg sync.WaitGroup
	for _, task := range cw.tasks {
		if task.Enabled {
			wg.Add(1)
			go func(t *WarmupTask) {
				defer wg.Done()
				cw.warmupTask(ctx, t)
			}(task)
		}
	}
	wg.Wait()

	log.Println("[CACHE_WARMER] Initial warmup completed")

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cw.stopCh:
			return
		case <-ticker.C:
			cw.runScheduledWarmup(ctx)
		}
	}
}

func (cw *CacheWarmer) runScheduledWarmup(ctx context.Context) {
	var wg sync.WaitGroup

	for _, task := range cw.tasks {
		if !task.Enabled {
			continue
		}

		wg.Add(1)
		go func(t *WarmupTask) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case <-cw.stopCh:
				return
			default:
				cw.warmupTask(ctx, t)
			}
		}(task)
	}

	wg.Wait()
}

func (cw *CacheWarmer) warmupTask(ctx context.Context, task *WarmupTask) {
	start := time.Now()

	data, err := task.Loader(ctx, cw.client)
	if err != nil {
		log.Printf("[CACHE_WARMER] Error warming up %s: %v", task.Name, err)
		cw.stats.mu.Lock()
		cw.stats.TotalErrors++
		cw.stats.mu.Unlock()
		return
	}

	if len(data) == 0 {
		log.Printf("[CACHE_WARMER] No data to warm for %s", task.Name)
		return
	}

	pipe := cw.client.Pipeline()
	for key, value := range data {
		pipe.Set(ctx, key, value, task.Interval)
	}

	_, err = pipe.Exec(ctx)
	duration := time.Since(start)

	cw.stats.mu.Lock()
	cw.stats.TotalLoads++
	cw.stats.TotalKeys += int64(len(data))
	cw.stats.LastLoadTime = time.Now()
	cw.stats.LastLoadCount = int64(len(data))
	cw.stats.LoadDurations = append(cw.stats.LoadDurations, duration)
	if len(cw.stats.LoadDurations) > 100 {
		cw.stats.LoadDurations = cw.stats.LoadDurations[1:]
	}
	cw.stats.mu.Unlock()

	log.Printf("[CACHE_WARMER] Warmed up %s: %d keys in %v", task.Name, len(data), duration)
}

func (cw *CacheWarmer) GetStats() *WarmupStats {
	cw.stats.mu.RLock()
	defer cw.stats.mu.RUnlock()

	statsCopy := &WarmupStats{
		TotalLoads:     cw.stats.TotalLoads,
		TotalKeys:      cw.stats.TotalKeys,
		TotalErrors:    cw.stats.TotalErrors,
		LastLoadTime:   cw.stats.LastLoadTime,
		LastLoadCount:  cw.stats.LastLoadCount,
		LastErrorCount: cw.stats.LastErrorCount,
	}

	if len(cw.stats.LoadDurations) > 0 {
		var total time.Duration
		for _, d := range cw.stats.LoadDurations {
			total += d
		}
		avgDuration := total / time.Duration(len(cw.stats.LoadDurations))
		statsCopy.LoadDurations = []time.Duration{avgDuration}
	}

	return statsCopy
}

func (cw *CacheWarmer) GetTasksStatus() map[string]interface{} {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	status := make(map[string]interface{})
	for prefix, task := range cw.tasks {
		status[prefix] = map[string]interface{}{
			"name":      task.Name,
			"enabled":   task.Enabled,
			"interval":  task.Interval.String(),
			"priority":  task.Priority,
			"parallel":  task.ParallelLoad,
		}
	}

	return status
}

func (cw *CacheWarmer) EnableTask(prefix string, enabled bool) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if task, ok := cw.tasks[prefix]; ok {
		task.Enabled = enabled
		log.Printf("[CACHE_WARMER] Task %s enabled=%v", prefix, enabled)
	}
}

func (cw *CacheWarmer) RunTaskNow(ctx context.Context, prefix string) error {
	cw.mu.RLock()
	task, ok := cw.tasks[prefix]
	cw.mu.RUnlock()

	if !ok {
		return fmt.Errorf("task not found: %s", prefix)
	}

	cw.warmupTask(ctx, task)
	return nil
}

func (cw *CacheWarmer) loadConfigCache(ctx context.Context, client *goredis.Client) (map[string]string, error) {
	result := make(map[string]string)

	iter := client.Scan(ctx, 0, "config:*", 100).Iterator()
	for iter.Next(ctx) {
		val, err := client.Get(ctx, iter.Val()).Result()
		if err == nil {
			result[iter.Val()] = val
		}
	}

	return result, iter.Err()
}

func (cw *CacheWarmer) loadCaptchaTemplates(ctx context.Context, client *goredis.Client) (map[string]string, error) {
	result := make(map[string]string)

	iter := client.Scan(ctx, 0, "captcha:template:*", 100).Iterator()
	for iter.Next(ctx) {
		val, err := client.Get(ctx, iter.Val()).Result()
		if err == nil {
			result[iter.Val()] = val
		}
	}

	return result, iter.Err()
}

func (cw *CacheWarmer) loadApplicationConfigs(ctx context.Context, client *goredis.Client) (map[string]string, error) {
	result := make(map[string]string)

	iter := client.Scan(ctx, 0, "app:*", 100).Iterator()
	for iter.Next(ctx) {
		val, err := client.Get(ctx, iter.Val()).Result()
		if err == nil {
			result[iter.Val()] = val
		}
	}

	return result, iter.Err()
}

func (cw *CacheWarmer) loadWhitelist(ctx context.Context, client *goredis.Client) (map[string]string, error) {
	result := make(map[string]string)

	iter := client.Scan(ctx, 0, "whitelist:*", 100).Iterator()
	for iter.Next(ctx) {
		val, err := client.Get(ctx, iter.Val()).Result()
		if err == nil {
			result[iter.Val()] = val
		}
	}

	return result, iter.Err()
}

func (cw *CacheWarmer) loadBlacklist(ctx context.Context, client *goredis.Client) (map[string]string, error) {
	result := make(map[string]string)

	iter := client.Scan(ctx, 0, "blacklist:*", 100).Iterator()
	for iter.Next(ctx) {
		val, err := client.Get(ctx, iter.Val()).Result()
		if err == nil {
			result[iter.Val()] = val
		}
	}

	return result, iter.Err()
}

func (cw *CacheWarmer) loadRateLimitRules(ctx context.Context, client *goredis.Client) (map[string]string, error) {
	result := make(map[string]string)

	iter := client.Scan(ctx, 0, "ratelimit:rule:*", 100).Iterator()
	for iter.Next(ctx) {
		val, err := client.Get(ctx, iter.Val()).Result()
		if err == nil {
			result[iter.Val()] = val
		}
	}

	return result, iter.Err()
}

type AdaptiveCacheWarmer struct {
	*CacheWarmer
	peakHours      []int
	peakLoadFactor float64
	offPeakInterval time.Duration
	peakInterval   time.Duration
}

func NewAdaptiveCacheWarmer(client *goredis.Client) *AdaptiveCacheWarmer {
	acw := &AdaptiveCacheWarmer{
		CacheWarmer:    NewCacheWarmer(client),
		peakHours:      []int{9, 10, 11, 14, 15, 16, 19, 20, 21},
		peakLoadFactor: 0.5,
		offPeakInterval: 15 * time.Minute,
		peakInterval:   5 * time.Minute,
	}

	return acw
}

func (acw *AdaptiveCacheWarmer) Start(ctx context.Context) {
	acw.CacheWarmer.Start(ctx)
	go acw.adaptiveInterval(ctx)
}

func (acw *AdaptiveCacheWarmer) adaptiveInterval(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-acw.stopCh:
			return
		case <-ticker.C:
			acw.adjustIntervals()
		}
	}
}

func (acw *AdaptiveCacheWarmer) adjustIntervals() {
	hour := time.Now().Hour()
	isPeak := false

	for _, peakHour := range acw.peakHours {
		if hour == peakHour {
			isPeak = true
			break
		}
	}

	acw.mu.Lock()
	defer acw.mu.Unlock()

	for _, task := range acw.tasks {
		if isPeak {
			task.Interval = time.Duration(float64(task.Interval) * acw.peakLoadFactor)
		} else {
			task.Interval = acw.offPeakInterval
		}
	}
}

func (acw *AdaptiveCacheWarmer) SetPeakHours(hours []int) {
	acw.mu.Lock()
	defer acw.mu.Unlock()
	acw.peakHours = hours
}

func (acw *AdaptiveCacheWarmer) SetLoadFactor(factor float64) {
	acw.mu.Lock()
	defer acw.mu.Unlock()
	acw.peakLoadFactor = factor
}
