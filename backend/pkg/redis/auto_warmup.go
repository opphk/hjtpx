package redis

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type WarmupStrategy int

const (
	WarmupStrategyEager WarmupStrategy = iota
	WarmupStrategyLazy
	WarmupStrategyPredictive
	WarmupStrategyHybrid
)

type WarmupTrigger int

const (
	WarmupTriggerStartup WarmupTrigger = iota
	WarmupTriggerScheduled
	WarmupTriggerOnDemand
	WarmupTriggerAdaptive
	WarmupTriggerThreshold
)

type CacheWarmupProfile struct {
	Name             string
	Priority         int
	Keys             []string
	Loader           func(ctx context.Context, key string) ([]byte, error)
	TTL              time.Duration
	Concurrency      int
	BatchSize        int
	Timeout          time.Duration
	RetryPolicy      RetryPolicy
	SuccessCriteria  SuccessCriteria
	Enabled          bool
}

type RetryPolicy struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
}

type SuccessCriteria struct {
	MinSuccessRate float64
	MinWarmupRate  float64
	MaxFailureRate float64
}

type WarmupScheduler struct {
	mu              sync.RWMutex
	config          *AutoWarmupConfig
	profiles        map[string]*CacheWarmupProfile
	schedules       map[string]*WarmupSchedule
	triggers        map[WarmupTrigger]TriggerHandler
	executor        *WarmupExecutor
	accessPredictor *AccessPredictor
	performanceTracker *PerformanceTracker
	enabled         bool
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

type WarmupSchedule struct {
	ProfileName string
	CronExpr    string
	Interval    time.Duration
	LastRun     time.Time
	NextRun     time.Time
	Enabled     bool
}

type TriggerHandler interface {
	ShouldTrigger() bool
	GetWarmupKeys() []string
}

type AccessPredictor struct {
	mu           sync.RWMutex
	model        *PredictionModel
	accessHistory map[string][]AccessRecord
}

type AccessRecord struct {
	Timestamp  time.Time
	Key        string
	Count      int64
	Duration   time.Duration
}

type PredictionModel struct {
	mu              sync.RWMutex
	patterns        map[string]*AccessPattern
	windowSize      time.Duration
	minConfidence   float64
}

type AccessPattern struct {
	Key              string
	BaseFrequency    float64
	PeriodicScore    float64
	RecencyScore     float64
	Confidence       float64
	LastPredictedAt  time.Time
}

type PerformanceTracker struct {
	mu           sync.RWMutex
	metrics      map[string]*WarmupMetrics
	currentRun   *RunningWarmup
}

type WarmupMetrics struct {
	TotalRuns      atomic.Int64
	SuccessRuns    atomic.Int64
	FailedRuns     atomic.Int64
	TotalKeys      atomic.Int64
	WarmmedKeys    atomic.Int64
	FailedKeys     atomic.Int64
	AvgDuration    atomic.Int64
	PeakMemory     atomic.Int64
	SuccessRate    float64
	LastRunTime    atomic.Value
}

type RunningWarmup struct {
	ProfileName string
	StartTime   time.Time
	Keys        []string
	Progress    float64
	Status      string
}

type AutoWarmupConfig struct {
	Enabled           bool
	DefaultStrategy   WarmupStrategy
	DefaultPriority   int
	Concurrency       int
	BatchSize         int
	Timeout           time.Duration
	EnablePrediction  bool
	PredictionWindow  time.Duration
	ThresholdUtil     float64
	EnableMetrics     bool
	AutoTuning        bool
	SchedulerEnabled  bool
}

var DefaultAutoWarmupConfig = &AutoWarmupConfig{
	Enabled:          true,
	DefaultStrategy:  WarmupStrategyPredictive,
	DefaultPriority:  5,
	Concurrency:      10,
	BatchSize:        100,
	Timeout:          30 * time.Second,
	EnablePrediction: true,
	PredictionWindow: 1 * time.Hour,
	ThresholdUtil:    0.8,
	EnableMetrics:    true,
	AutoTuning:       true,
	SchedulerEnabled: true,
}

type WarmupExecutor struct {
	mu           sync.RWMutex
	config       *AutoWarmupConfig
	stats        *WarmupMetrics
	currentRun   *RunningWarmup
	abortCh      chan struct{}
}

func NewWarmupScheduler(config *AutoWarmupConfig) *WarmupScheduler {
	if config == nil {
		config = DefaultAutoWarmupConfig
	}

	ctx, cancel := context.WithCancel(context.Background())

	ws := &WarmupScheduler{
		config:             config,
		profiles:           make(map[string]*CacheWarmupProfile),
		schedules:          make(map[string]*WarmupSchedule),
		triggers:           make(map[WarmupTrigger]TriggerHandler),
		executor:           NewWarmupExecutor(config),
		accessPredictor:    NewAccessPredictor(config.PredictionWindow),
		performanceTracker: NewPerformanceTracker(),
		enabled:            config.Enabled,
		ctx:                ctx,
		cancel:             cancel,
	}

	ws.initTriggers()

	if config.SchedulerEnabled {
		ws.startScheduler()
	}

	return ws
}

func (ws *WarmupScheduler) initTriggers() {
	ws.triggers[WarmupTriggerStartup] = &StartupTrigger{ws: ws}
	ws.triggers[WarmupTriggerScheduled] = &ScheduledTrigger{ws: ws}
	ws.triggers[WarmupTriggerAdaptive] = &AdaptiveTrigger{ws: ws, predictor: ws.accessPredictor}
	ws.triggers[WarmupTriggerThreshold] = &ThresholdTrigger{ws: ws, threshold: ws.config.ThresholdUtil}
}

func (ws *WarmupScheduler) AddProfile(profile *CacheWarmupProfile) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if profile.Concurrency == 0 {
		profile.Concurrency = ws.config.Concurrency
	}
	if profile.BatchSize == 0 {
		profile.BatchSize = ws.config.BatchSize
	}
	if profile.Timeout == 0 {
		profile.Timeout = ws.config.Timeout
	}

	ws.profiles[profile.Name] = profile
}

func (ws *WarmupScheduler) RemoveProfile(name string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	delete(ws.profiles, name)
}

func (ws *WarmupScheduler) AddSchedule(schedule *WarmupSchedule) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if schedule.Interval == 0 {
		schedule.Interval = 1 * time.Hour
	}
	schedule.NextRun = time.Now()

	ws.schedules[schedule.ProfileName] = schedule
}

func (ws *WarmupScheduler) Start() {
	ws.mu.Lock()
	ws.enabled = true
	ws.mu.Unlock()

	if trigger, ok := ws.triggers[WarmupTriggerStartup]; ok && trigger.ShouldTrigger() {
		go ws.executeWarmup()
	}
}

func (ws *WarmupScheduler) Stop() {
	ws.mu.Lock()
	ws.enabled = false
	ws.mu.Unlock()

	ws.cancel()
	ws.wg.Wait()
}

func (ws *WarmupScheduler) startScheduler() {
	ws.wg.Add(1)
	go ws.runScheduler()
}

func (ws *WarmupScheduler) runScheduler() {
	defer ws.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ws.ctx.Done():
			return
		case <-ticker.C:
			ws.checkSchedules()
			ws.checkAdaptiveTriggers()
		}
	}
}

func (ws *WarmupScheduler) checkSchedules() {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	now := time.Now()
	for _, schedule := range ws.schedules {
		if !schedule.Enabled {
			continue
		}

		if now.After(schedule.NextRun) {
			go ws.executeProfileWarmup(schedule.ProfileName)
			schedule.LastRun = now
			schedule.NextRun = now.Add(schedule.Interval)
		}
	}
}

func (ws *WarmupScheduler) checkAdaptiveTriggers() {
	if trigger, ok := ws.triggers[WarmupTriggerAdaptive]; ok {
		if trigger.ShouldTrigger() {
			keys := trigger.GetWarmupKeys()
			if len(keys) > 0 {
				go ws.executeAdaptiveWarmup(keys)
			}
		}
	}

	if trigger, ok := ws.triggers[WarmupTriggerThreshold]; ok {
		if trigger.ShouldTrigger() {
			go ws.executeWarmup()
		}
	}
}

func (ws *WarmupScheduler) executeWarmup() {
	ws.mu.RLock()
	profiles := make([]*CacheWarmupProfile, 0, len(ws.profiles))
	for _, profile := range ws.profiles {
		if profile.Enabled {
			profiles = append(profiles, profile)
		}
	}
	ws.mu.RUnlock()

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Priority < profiles[j].Priority
	})

	for _, profile := range profiles {
		ws.executeProfileWarmup(profile.Name)
	}
}

func (ws *WarmupScheduler) executeProfileWarmup(profileName string) {
	ws.mu.RLock()
	profile, exists := ws.profiles[profileName]
	ws.mu.RUnlock()

	if !exists || !profile.Enabled {
		return
	}

	ws.executor.Execute(profile)
}

func (ws *WarmupScheduler) executeAdaptiveWarmup(keys []string) {
	ws.mu.RLock()
	config := ws.config
	ws.mu.RUnlock()

	profile := &CacheWarmupProfile{
		Name:        "adaptive",
		Priority:    1,
		Keys:        keys,
		Concurrency: config.Concurrency,
		BatchSize:   config.BatchSize,
		Timeout:     config.Timeout,
		Enabled:     true,
	}

	ws.executor.Execute(profile)
}

func (ws *WarmupScheduler) TriggerWarmup(profileName string) error {
	ws.mu.RLock()
	_, exists := ws.profiles[profileName]
	ws.mu.RUnlock()

	if !exists {
		return nil
	}

	go ws.executeProfileWarmup(profileName)
	return nil
}

func (ws *WarmupScheduler) GetProfile(name string) *CacheWarmupProfile {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.profiles[name]
}

func (ws *WarmupScheduler) GetStats() map[string]*WarmupMetrics {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	stats := make(map[string]*WarmupMetrics)
	for name := range ws.profiles {
		stats[name] = ws.performanceTracker.GetMetrics(name)
	}
	return stats
}

func NewWarmupExecutor(config *AutoWarmupConfig) *WarmupExecutor {
	return &WarmupExecutor{
		config: config,
		stats:  &WarmupMetrics{},
		abortCh: make(chan struct{}),
	}
}

func (we *WarmupExecutor) Execute(profile *CacheWarmupProfile) {
	if profile == nil || len(profile.Keys) == 0 {
		return
	}

	we.mu.Lock()
	we.currentRun = &RunningWarmup{
		ProfileName: profile.Name,
		StartTime:   time.Now(),
		Keys:        profile.Keys,
		Progress:    0,
		Status:      "running",
	}
	we.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), profile.Timeout)
	defer cancel()

	semaphore := make(chan struct{}, profile.Concurrency)
	var wg sync.WaitGroup
	var statsMutex sync.Mutex
	localStats := &WarmupMetrics{}

	keys := profile.Keys
	batchSize := profile.BatchSize

	for i := 0; i < len(keys); i += batchSize {
		select {
		case <-ctx.Done():
			goto done
		default:
		}

		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		wg.Add(1)

		go func(batch []string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			we.processBatch(ctx, batch, profile, localStats, &statsMutex)
		}(batch)
	}

	wg.Wait()

done:
	we.mu.Lock()
	if we.currentRun != nil {
		we.currentRun.Status = "completed"
		we.currentRun.Progress = 1.0
	}
	we.mu.Unlock()

	if we.stats.TotalRuns.Load() > 0 {
		we.stats.SuccessRuns.Add(localStats.WarmmedKeys.Load())
		we.stats.TotalKeys.Add(localStats.TotalKeys.Load())
	}
}

func (we *WarmupExecutor) processBatch(ctx context.Context, keys []string, profile *CacheWarmupProfile, stats *WarmupMetrics, mu *sync.Mutex) {
	for _, key := range keys {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if profile.Loader == nil {
			continue
		}

		data, err := profile.Loader(ctx, key)
		if err != nil {
			stats.FailedKeys.Add(1)
			continue
		}

		if enhancedCache := GetEnhancedCache(); enhancedCache != nil {
			enhancedCache.Set(ctx, key, data, &SetOptions{
				TTL:   profile.TTL,
				Level: CacheLevelBoth,
			})
		} else if Client != nil {
			Client.Set(ctx, key, data, profile.TTL)
		}

		stats.WarmmedKeys.Add(1)
		stats.TotalKeys.Add(1)
	}
}

func (we *WarmupExecutor) GetCurrentRun() *RunningWarmup {
	we.mu.RLock()
	defer we.mu.RUnlock()
	return we.currentRun
}

func NewAccessPredictor(windowSize time.Duration) *AccessPredictor {
	return &AccessPredictor{
		accessHistory: make(map[string][]AccessRecord),
		model: &PredictionModel{
			windowSize:    windowSize,
			minConfidence: 0.7,
			patterns:      make(map[string]*AccessPattern),
		},
	}
}

func (ap *AccessPredictor) RecordAccess(key string, count int64, duration time.Duration) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	record := AccessRecord{
		Timestamp: time.Now(),
		Key:       key,
		Count:     count,
		Duration: duration,
	}

	ap.accessHistory[key] = append(ap.accessHistory[key], record)

	ap.pruneHistory(key)

	ap.updatePattern(key)
}

func (ap *AccessPredictor) pruneHistory(key string) {
	cutoff := time.Now().Add(-ap.model.windowSize)

	records := ap.accessHistory[key]
	var pruned []AccessRecord
	for _, record := range records {
		if record.Timestamp.After(cutoff) {
			pruned = append(pruned, record)
		}
	}
	ap.accessHistory[key] = pruned
}

func (ap *AccessPredictor) updatePattern(key string) {
	records := ap.accessHistory[key]
	if len(records) == 0 {
		return
	}

	var totalCount int64
	var totalDuration time.Duration
	for _, record := range records {
		totalCount += record.Count
		totalDuration += record.Duration
	}

	avgFrequency := float64(totalCount) / float64(len(records))

	pattern := &AccessPattern{
		Key:             key,
		BaseFrequency:   avgFrequency,
		RecencyScore:    1.0,
		LastPredictedAt: time.Now(),
	}

	ap.model.mu.Lock()
	ap.model.patterns[key] = pattern
	ap.model.mu.Unlock()
}

func (ap *AccessPredictor) PredictHotKeys(count int) []string {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	type keyScore struct {
		Key   string
		Score float64
	}

	var scores []keyScore

	ap.model.mu.RLock()
	for key, pattern := range ap.model.patterns {
		score := pattern.BaseFrequency * pattern.RecencyScore
		scores = append(scores, keyScore{Key: key, Score: score})
	}
	ap.model.mu.RUnlock()

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	if count > len(scores) {
		count = len(scores)
	}

	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = scores[i].Key
	}

	return result
}

func NewPerformanceTracker() *PerformanceTracker {
	return &PerformanceTracker{
		metrics: make(map[string]*WarmupMetrics),
	}
}

func (pt *PerformanceTracker) GetMetrics(name string) *WarmupMetrics {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if metrics, exists := pt.metrics[name]; exists {
		return metrics
	}

	return &WarmupMetrics{}
}

func (pt *PerformanceTracker) RecordMetrics(name string, m *WarmupMetrics) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.metrics[name] = m
}

type StartupTrigger struct {
	ws *WarmupScheduler
}

func (st *StartupTrigger) ShouldTrigger() bool {
	return st.ws.config.Enabled
}

func (st *StartupTrigger) GetWarmupKeys() []string {
	return nil
}

type ScheduledTrigger struct {
	ws *WarmupScheduler
}

func (st *ScheduledTrigger) ShouldTrigger() bool {
	return false
}

func (st *ScheduledTrigger) GetWarmupKeys() []string {
	return nil
}

type AdaptiveTrigger struct {
	ws        *WarmupScheduler
	predictor *AccessPredictor
}

func (at *AdaptiveTrigger) ShouldTrigger() bool {
	return at.predictor != nil && len(at.predictor.PredictHotKeys(100)) > 0
}

func (at *AdaptiveTrigger) GetWarmupKeys() []string {
	if at.predictor == nil {
		return nil
	}
	return at.predictor.PredictHotKeys(500)
}

type ThresholdTrigger struct {
	ws        *WarmupScheduler
	threshold float64
}

func (tt *ThresholdTrigger) ShouldTrigger() bool {
	if Client == nil {
		return false
	}

	info := Client.Info(tt.ws.ctx, "memory")
	if info == nil || info.Err() != nil {
		return false
	}

	var usedMemory float64
	var maxMemory float64

	infoStr := info.Val()
	lines := strings.Split(infoStr, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "used_memory:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &usedMemory)
			}
		} else if strings.HasPrefix(line, "maxmemory:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &maxMemory)
			}
		}
	}

	if maxMemory == 0 {
		return false
	}

	utilization := usedMemory / maxMemory
	return utilization > tt.threshold
}

func (tt *ThresholdTrigger) GetWarmupKeys() []string {
	return nil
}

func CalculateWarmupPriority(profile *CacheWarmupProfile) int {
	priority := profile.Priority

	if profile.SuccessCriteria.MinSuccessRate > 0 {
		priority += int((1.0 - profile.SuccessCriteria.MinSuccessRate) * 10)
	}

	return int(math.Max(float64(priority), 1))
}

var (
	globalWarmupScheduler    *WarmupScheduler
	globalWarmupSchedulerOnce sync.Once
)

func InitAutoWarmupScheduler(config *AutoWarmupConfig) {
	globalWarmupSchedulerOnce.Do(func() {
		globalWarmupScheduler = NewWarmupScheduler(config)
	})
}

func GetAutoWarmupScheduler() *WarmupScheduler {
	if globalWarmupScheduler == nil {
		InitAutoWarmupScheduler(nil)
	}
	return globalWarmupScheduler
}

func StartAutoWarmup() {
	GetAutoWarmupScheduler().Start()
}

func StopAutoWarmup() {
	GetAutoWarmupScheduler().Stop()
}

func AddWarmupProfile(profile *CacheWarmupProfile) {
	GetAutoWarmupScheduler().AddProfile(profile)
}

func TriggerProfileWarmup(profileName string) error {
	return GetAutoWarmupScheduler().TriggerWarmup(profileName)
}
