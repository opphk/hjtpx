package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type WarmupService struct {
	cache           *AdvancedCache
	warmupStrategies map[string]WarmupStrategy
	progress        *WarmupProgress
	status          int32
	mu              sync.RWMutex
	stopChan        chan struct{}
}

type WarmupStrategy interface {
	GetPriorityKeys(ctx context.Context, limit int) ([]string, error)
	WarmupItem(ctx context.Context, key string) error
	GetStrategyName() string
}

type WarmupProgress struct {
	TotalItems     int64
	CompletedItems int64
	FailedItems    int64
	StartTime      time.Time
	EndTime        time.Time
	CurrentPhase   string
	Strategies     map[string]*StrategyProgress
	mu             sync.RWMutex
}

type StrategyProgress struct {
	Name          string
	TotalItems    int64
	CompletedItems int64
	FailedItems   int64
	LastError     string
}

func NewWarmupProgress() *WarmupProgress {
	return &WarmupProgress{
		Strategies: make(map[string]*StrategyProgress),
	}
}

func (wp *WarmupProgress) Start(phase string) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	wp.StartTime = time.Now()
	wp.CurrentPhase = phase
	wp.TotalItems = 0
	wp.CompletedItems = 0
	wp.FailedItems = 0
}

func (wp *WarmupProgress) End() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	wp.EndTime = time.Now()
}

func (wp *WarmupProgress) SetTotalItems(total int64) {
	atomic.StoreInt64(&wp.TotalItems, total)
}

func (wp *WarmupProgress) IncrementCompleted() {
	atomic.AddInt64(&wp.CompletedItems, 1)
}

func (wp *WarmupProgress) IncrementFailed() {
	atomic.AddInt64(&wp.FailedItems, 1)
}

func (wp *WarmupProgress) SetPhase(phase string) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	wp.CurrentPhase = phase
}

func (wp *WarmupProgress) GetProgress() (total, completed, failed int64, phase string, percent float64) {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	total = atomic.LoadInt64(&wp.TotalItems)
	completed = atomic.LoadInt64(&wp.CompletedItems)
	failed = atomic.LoadInt64(&wp.FailedItems)
	phase = wp.CurrentPhase

	if total > 0 {
		percent = float64(completed) / float64(total) * 100
	}

	return
}

func (wp *WarmupProgress) GetElapsedTime() time.Duration {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	if wp.StartTime.IsZero() {
		return 0
	}
	if !wp.EndTime.IsZero() {
		return wp.EndTime.Sub(wp.StartTime)
	}
	return time.Since(wp.StartTime)
}

func (wp *WarmupProgress) GetEstimatedTimeRemaining() time.Duration {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	if wp.StartTime.IsZero() {
		return 0
	}

	completed := atomic.LoadInt64(&wp.CompletedItems)
	if completed == 0 {
		return 0
	}

	total := atomic.LoadInt64(&wp.TotalItems)
	elapsed := time.Since(wp.StartTime)
	avgTimePerItem := elapsed / time.Duration(completed)
	remaining := total - completed

	return avgTimePerItem * time.Duration(remaining)
}

type PopularCaptchaWarmupStrategy struct {
	cache     *AdvancedCache
	redisKey  string
	batchSize int
}

func NewPopularCaptchaWarmupStrategy(cache *AdvancedCache) *PopularCaptchaWarmupStrategy {
	return &PopularCaptchaWarmupStrategy{
		cache:     cache,
		redisKey:  "captcha:popular",
		batchSize: 100,
	}
}

func (s *PopularCaptchaWarmupStrategy) GetStrategyName() string {
	return "popular_captcha"
}

func (s *PopularCaptchaWarmupStrategy) GetPriorityKeys(ctx context.Context, limit int) ([]string, error) {
	keys := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		keys = append(keys, fmt.Sprintf("captcha:popular:%d", i))
	}
	return keys, nil
}

func (s *PopularCaptchaWarmupStrategy) WarmupItem(ctx context.Context, key string) error {
	if s.cache == nil {
		return nil
	}
	_, err := s.cache.Get(ctx, key)
	return err
}

type StartupWarmupStrategy struct {
	cache       *AdvancedCache
	essentialKeys []string
}

func NewStartupWarmupStrategy(cache *AdvancedCache) *StartupWarmupStrategy {
	keys := []string{
		"captcha:config:default",
		"captcha:config:security",
		"captcha:config:rate_limits",
		"captcha:assets:fonts",
		"captcha:assets:templates",
		"captcha:i18n:en",
		"captcha:i18n:zh",
	}
	return &StartupWarmupStrategy{
		cache:         cache,
		essentialKeys: keys,
	}
}

func (s *StartupWarmupStrategy) GetStrategyName() string {
	return "startup"
}

func (s *StartupWarmupStrategy) GetPriorityKeys(ctx context.Context, limit int) ([]string, error) {
	if limit > len(s.essentialKeys) {
		limit = len(s.essentialKeys)
	}
	keys := make([]string, limit)
	copy(keys, s.essentialKeys[:limit])
	return keys, nil
}

func (s *StartupWarmupStrategy) WarmupItem(ctx context.Context, key string) error {
	if s.cache == nil {
		return nil
	}
	_, err := s.cache.Get(ctx, key)
	return err
}

type PeriodicWarmupStrategy struct {
	cache      *AdvancedCache
	interval   time.Duration
	maxItems   int
	keyPattern string
}

func NewPeriodicWarmupStrategy(cache *AdvancedCache, interval time.Duration, maxItems int) *PeriodicWarmupStrategy {
	return &PeriodicWarmupStrategy{
		cache:      cache,
		interval:   interval,
		maxItems:   maxItems,
		keyPattern: "captcha:session:*",
	}
}

func (s *PeriodicWarmupStrategy) GetStrategyName() string {
	return "periodic"
}

func (s *PeriodicWarmupStrategy) GetPriorityKeys(ctx context.Context, limit int) ([]string, error) {
	keys := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		keys = append(keys, fmt.Sprintf("captcha:session:%d", i))
	}
	return keys, nil
}

func (s *PeriodicWarmupStrategy) WarmupItem(ctx context.Context, key string) error {
	if s.cache == nil {
		return nil
	}
	_, err := s.cache.Get(ctx, key)
	return err
}

func NewWarmupService(cache *AdvancedCache) *WarmupService {
	return &WarmupService{
		cache:            cache,
		warmupStrategies: make(map[string]WarmupStrategy),
		progress:         NewWarmupProgress(),
		status:           0,
		stopChan:         make(chan struct{}),
	}
}

func (s *WarmupService) RegisterStrategy(strategy WarmupStrategy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.warmupStrategies[strategy.GetStrategyName()] = strategy
}

func (s *WarmupService) UnregisterStrategy(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.warmupStrategies, name)
}

func (s *WarmupService) GetRegisteredStrategies() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.warmupStrategies))
	for name := range s.warmupStrategies {
		names = append(names, name)
	}
	return names
}

func (s *WarmupService) Warmup(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&s.status, 0, 1) {
		return fmt.Errorf("warmup already in progress")
	}
	defer atomic.StoreInt32(&s.status, 0)

	s.mu.RLock()
	strategies := make([]WarmupStrategy, 0, len(s.warmupStrategies))
	for _, strategy := range s.warmupStrategies {
		strategies = append(strategies, strategy)
	}
	s.mu.RUnlock()

	totalKeys := 0
	for _, strategy := range strategies {
		keys, err := strategy.GetPriorityKeys(ctx, 1000)
		if err != nil {
			continue
		}
		totalKeys += len(keys)
		s.progress.Strategies[strategy.GetStrategyName()] = &StrategyProgress{
			Name:       strategy.GetStrategyName(),
			TotalItems: int64(len(keys)),
		}
	}

	s.progress.SetTotalItems(int64(totalKeys))
	s.progress.Start("warming")

	var wg sync.WaitGroup
	var warmupErrors []error
	var errorsMu sync.Mutex

	for _, strategy := range strategies {
		wg.Add(1)
		go func(str WarmupStrategy) {
			defer wg.Done()

			keys, err := str.GetPriorityKeys(ctx, 1000)
			if err != nil {
				errorsMu.Lock()
				warmupErrors = append(warmupErrors, fmt.Errorf("%s: %w", str.GetStrategyName(), err))
				errorsMu.Unlock()
				return
			}

			for _, key := range keys {
				select {
				case <-ctx.Done():
					return
				default:
					if err := str.WarmupItem(ctx, key); err != nil {
						errorsMu.Lock()
						warmupErrors = append(warmupErrors, fmt.Errorf("%s:%s: %w", str.GetStrategyName(), key, err))
						s.progress.IncrementFailed()
						if s.progress.Strategies[str.GetStrategyName()] != nil {
							s.progress.Strategies[str.GetStrategyName()].FailedItems++
							s.progress.Strategies[str.GetStrategyName()].LastError = err.Error()
						}
						errorsMu.Unlock()
					} else {
						s.progress.IncrementCompleted()
						if s.progress.Strategies[str.GetStrategyName()] != nil {
							s.progress.Strategies[str.GetStrategyName()].CompletedItems++
						}
					}
				}
			}
		}(strategy)
	}

	wg.Wait()
	s.progress.End()

	if len(warmupErrors) > 0 {
		return fmt.Errorf("warmup completed with %d errors: %v", len(warmupErrors), warmupErrors)
	}

	return nil
}

func (s *WarmupService) WarmupAsync(ctx context.Context) {
	go func() {
		warmupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		s.Warmup(warmupCtx)
	}()
}

func (s *WarmupService) StartPeriodicWarmup(ctx context.Context, interval time.Duration) {
	s.mu.Lock()
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			warmupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			s.Warmup(warmupCtx)
			cancel()
		}
	}
}

func (s *WarmupService) StopPeriodicWarmup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopChan != nil {
		close(s.stopChan)
		s.stopChan = make(chan struct{})
	}
}

func (s *WarmupService) GetProgress() (total, completed, failed int64, phase string, percent float64) {
	return s.progress.GetProgress()
}

func (s *WarmupService) GetDetailedProgress() *WarmupProgress {
	s.progress.mu.RLock()
	defer s.progress.mu.RUnlock()

	detailed := &WarmupProgress{
		TotalItems:     atomic.LoadInt64(&s.progress.TotalItems),
		CompletedItems: atomic.LoadInt64(&s.progress.CompletedItems),
		FailedItems:    atomic.LoadInt64(&s.progress.FailedItems),
		StartTime:      s.progress.StartTime,
		EndTime:        s.progress.EndTime,
		CurrentPhase:   s.progress.CurrentPhase,
		Strategies:     make(map[string]*StrategyProgress),
	}

	for name, sp := range s.progress.Strategies {
		detailed.Strategies[name] = &StrategyProgress{
			Name:          sp.Name,
			TotalItems:    atomic.LoadInt64(&sp.TotalItems),
			CompletedItems: atomic.LoadInt64(&sp.CompletedItems),
			FailedItems:   atomic.LoadInt64(&sp.FailedItems),
			LastError:     sp.LastError,
		}
	}

	return detailed
}

func (s *WarmupService) IsWarmingUp() bool {
	return atomic.LoadInt32(&s.status) == 1
}

func (s *WarmupService) TriggerManualWarmup(ctx context.Context, keys []string) error {
	if s.cache == nil {
		return fmt.Errorf("cache not initialized")
	}

	for _, key := range keys {
		_, err := s.cache.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("failed to warmup key %s: %w", key, err)
		}
	}

	return nil
}

type WarmupMetrics struct {
	totalWarmups      int64
	successfulWarmups int64
	failedWarmups     int64
	totalItemsWarmed  int64
	totalWarmupTime   time.Duration
	mu                sync.RWMutex
}

func NewWarmupMetrics() *WarmupMetrics {
	return &WarmupMetrics{}
}

func (m *WarmupMetrics) RecordWarmup(success bool, items int64, duration time.Duration) {
	atomic.AddInt64(&m.totalWarmups, 1)

	m.mu.Lock()
	defer m.mu.Unlock()

	if success {
		atomic.AddInt64(&m.successfulWarmups, 1)
	} else {
		atomic.AddInt64(&m.failedWarmups, 1)
	}

	atomic.AddInt64(&m.totalItemsWarmed, items)
	m.totalWarmupTime += duration
}

func (m *WarmupMetrics) GetSuccessRate() float64 {
	total := atomic.LoadInt64(&m.totalWarmups)
	if total == 0 {
		return 0
	}
	success := atomic.LoadInt64(&m.successfulWarmups)
	return float64(success) / float64(total) * 100
}

func (m *WarmupMetrics) GetAverageWarmupTime() time.Duration {
	total := atomic.LoadInt64(&m.totalWarmups)
	if total == 0 {
		return 0
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.totalWarmupTime / time.Duration(total)
}

type WarmupAPI struct {
	service *WarmupService
}

func NewWarmupAPI(service *WarmupService) *WarmupAPI {
	return &WarmupAPI{service: service}
}

type WarmupStatusResponse struct {
	Status       string    `json:"status"`
	IsWarmingUp  bool      `json:"is_warming_up"`
	Progress     float64   `json:"progress_percent"`
	TotalItems   int64     `json:"total_items"`
	Completed    int64     `json:"completed_items"`
	Failed       int64     `json:"failed_items"`
	ElapsedTime  float64   `json:"elapsed_seconds"`
	RemainingTime float64  `json:"remaining_seconds"`
	Strategies   []string  `json:"strategies"`
}

func (api *WarmupAPI) GetStatus() WarmupStatusResponse {
	total, completed, failed, phase, percent := api.service.GetProgress()
	elapsed := api.service.progress.GetElapsedTime()
	remaining := api.service.progress.GetEstimatedTimeRemaining()

	return WarmupStatusResponse{
		Status:        phase,
		IsWarmingUp:   api.service.IsWarmingUp(),
		Progress:      percent,
		TotalItems:    total,
		Completed:     completed,
		Failed:        failed,
		ElapsedTime:   elapsed.Seconds(),
		RemainingTime: remaining.Seconds(),
		Strategies:    api.service.GetRegisteredStrategies(),
	}
}

type ManualWarmupRequest struct {
	Keys []string `json:"keys" binding:"required"`
}

func (api *WarmupAPI) TriggerWarmup(ctx context.Context, req ManualWarmupRequest) error {
	return api.service.TriggerManualWarmup(ctx, req.Keys)
}
