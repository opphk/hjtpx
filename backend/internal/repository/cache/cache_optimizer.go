package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type CacheOptimizer struct {
	config      *OptimizerConfig
	keyManager  *OptimizedKeyManager
	expiration  *ExpirationOptimizer
	preheat     *PreheatManager
	monitor     *CacheMonitor
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	mu          sync.RWMutex
	started     bool
}

type OptimizerConfig struct {
	EnableKeyOptimization    bool
	EnableExpirationOptimize bool
	EnablePreheat            bool
	EnableMonitoring         bool
	DefaultTTL               time.Duration
	MinTTL                   time.Duration
	MaxTTL                   time.Duration
	PreheatBatchSize         int
	PreheatConcurrency       int
	MonitorInterval          time.Duration
	HotKeyThreshold          int64
	MaxMemoryPercent         float64
}

var DefaultOptimizerConfig = &OptimizerConfig{
	EnableKeyOptimization:    true,
	EnableExpirationOptimize: true,
	EnablePreheat:           true,
	EnableMonitoring:        true,
	DefaultTTL:              10 * time.Minute,
	MinTTL:                  1 * time.Minute,
	MaxTTL:                  1 * time.Hour,
	PreheatBatchSize:        50,
	PreheatConcurrency:      10,
	MonitorInterval:         30 * time.Second,
	HotKeyThreshold:         100,
	MaxMemoryPercent:        0.8,
}

type OptimizedKeyManager struct {
	mu        sync.RWMutex
	prefixes  map[string]*KeyPrefix
	version   int
	maxKeys   int
	keyStats  map[string]*KeyStats
}

type KeyPrefix struct {
	Name        string
	Separator   string
	Version     int
	Segments    []string
	TTL         time.Duration
	Description string
}

type KeyStats struct {
	AccessCount   int64
	LastAccess    time.Time
	CreationTime time.Time
	HitCount     int64
	MissCount    int64
	EvictionCount int64
	AvgLatency   time.Duration
	TotalLatency int64
	AccessCount2 int64
}

type ExpirationOptimizer struct {
	mu           sync.RWMutex
	config       *OptimizerConfig
	policies     map[string]*ExpirationPolicy
	accessCounts map[string]*AccessCount
	ttlHistory   map[string][]TTLRecord
	maxHistory   int
}

type ExpirationPolicy struct {
	Name           string
	BaseTTL        time.Duration
	MinTTL         time.Duration
	MaxTTL         time.Duration
	Strategy       ExpirationStrategy
	RefreshRatio   float64
	SlidingWindow  time.Duration
	RandomVariance float64
}

type ExpirationStrategy int

const (
	StrategyFixed ExpirationStrategy = iota
	StrategySliding
	StrategyRandom
	StrategyAdaptive
	StrategyPredictive
)

type AccessCount struct {
	Count     int64
	Timestamp time.Time
	Pattern   string
}

type TTLRecord struct {
	TTL       time.Duration
	Timestamp time.Time
	HitRate   float64
}

type PreheatManager struct {
	mu          sync.RWMutex
	config      *OptimizerConfig
	profiles    map[string]*PreheatProfile
	accessLog   map[string]*AccessLog
	hotKeys     map[string]*HotKeyInfo
	history     map[string][]time.Time
	maxHistory  int
	enabled     bool
	executor    *PreheatExecutor
	scheduler   *PreheatScheduler
}

type PreheatProfile struct {
	Name           string
	Priority       int
	Keys           []string
	DataLoader     func(ctx context.Context, key string) (interface{}, error)
	TTL            time.Duration
	Concurrency    int
	BatchSize      int
	Timeout        time.Duration
	RetryPolicy    *PreheatRetryPolicy
	SuccessRate    float64
	WarmmedCount   int64
	FailedCount    int64
	AvgDuration    time.Duration
	LastRun        time.Time
	Enabled        bool
	TotalRuns      int64
	SuccessRuns    int64
	FailedRuns     int64
	TotalKeys      int64
}

type PreheatRetryPolicy struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
}

type AccessLog struct {
	Key       string
	Count     int64
	LastTime  time.Time
	FirstTime time.Time
	Pattern   string
}

type HotKeyInfo struct {
	Key        string
	Count     int64
	Score     float64
	FirstSeen time.Time
	LastSeen  time.Time
	HitRate   float64
	AvgLatency time.Duration
	IsHot     bool
}

type PreheatExecutor struct {
	mu         sync.RWMutex
	config     *OptimizerConfig
	stats      *PreheatStats
	currentRun *PreheatRun
	abortCh    chan struct{}
}

type PreheatStats struct {
	TotalRuns   int64
	SuccessRuns int64
	FailedRuns  int64
	TotalKeys   int64
	WarmmedKeys int64
	FailedKeys  int64
	AvgDuration time.Duration
	LastRunTime time.Time
}

type PreheatRun struct {
	ProfileName string
	StartTime   time.Time
	Keys        []string
	Progress    float64
	Status      string
}

type PreheatScheduler struct {
	mu       sync.RWMutex
	profiles map[string]*PreheatProfile
	interval time.Duration
	enabled  bool
	ctx      context.Context
	cancel   context.CancelFunc
}

type CacheMonitor struct {
	mu              sync.RWMutex
	config          *OptimizerConfig
	metrics         *CacheMetrics
	alerts          []*CacheAlert
	maxAlerts       int
	hotKeyTracker   *HotKeyTracker
	latencyTracker  *LatencyTracker
	healthChecker   *HealthChecker
	started         bool
	stopCh          chan struct{}
}

type CacheMetrics struct {
	TotalHits     int64
	TotalMisses   int64
	TotalErrors   int64
	HitRate       float64
	AvgLatency    time.Duration
	P50Latency    time.Duration
	P95Latency    time.Duration
	P99Latency    time.Duration
	L1Hits        int64
	L1Misses      int64
	L2Hits        int64
	L2Misses      int64
	CurrentMemory int64
	PeakMemory    int64
	Evictions     int64
	Compressions  int64
}

type CacheAlert struct {
	ID        string
	Type      AlertType
	Severity  AlertSeverity
	Message   string
	Timestamp time.Time
	Key       string
	Metrics   map[string]interface{}
	Resolved  bool
}

type AlertType string

const (
	AlertLowHitRate    AlertType = "low_hit_rate"
	AlertHighLatency   AlertType = "high_latency"
	AlertHighErrorRate AlertType = "high_error_rate"
	AlertMemoryFull    AlertType = "memory_full"
	AlertHotKey        AlertType = "hot_key"
	AlertEviction      AlertType = "eviction"
)

type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

type HotKeyTracker struct {
	mu         sync.RWMutex
	keys       map[string]*HotKeyInfo
	maxKeys    int
	window     time.Duration
	accessLog  []AccessEvent
	maxLogSize int
}

type AccessEvent struct {
	Key       string
	Timestamp time.Time
	Latency   time.Duration
}

type LatencyTracker struct {
	mu      sync.RWMutex
	samples []time.Duration
	maxSize int
	buckets map[time.Duration]int64
}

type HealthChecker struct {
	mu            sync.RWMutex
	status        HealthStatus
	checks        map[string]HealthCheck
	lastCheckTime time.Time
}

type HealthStatus struct {
	Overall   string
	Score     float64
	Checks    map[string]HealthCheck
	Timestamp time.Time
}

type HealthCheck struct {
	Name        string
	Status     string
	LastRun    time.Time
	LastSuccess time.Time
	LastFailure time.Time
	Message     string
}

func NewCacheOptimizer(config *OptimizerConfig) *CacheOptimizer {
	if config == nil {
		config = DefaultOptimizerConfig
	}

	ctx, cancel := context.WithCancel(context.Background())

	optimizer := &CacheOptimizer{
		config:     config,
		keyManager: NewOptimizedKeyManager(),
		expiration: NewExpirationOptimizer(config),
		preheat:    NewPreheatManager(config),
		monitor:    NewCacheMonitor(config),
		ctx:        ctx,
		cancel:     cancel,
	}

	return optimizer
}

func NewOptimizedKeyManager() *OptimizedKeyManager {
	return &OptimizedKeyManager{
		prefixes: make(map[string]*KeyPrefix),
		version:  1,
		maxKeys:  10000,
		keyStats: make(map[string]*KeyStats),
	}
}

func (okm *OptimizedKeyManager) RegisterPrefix(prefix *KeyPrefix) {
	okm.mu.Lock()
	defer okm.mu.Unlock()

	if prefix.Separator == "" {
		prefix.Separator = ":"
	}
	okm.prefixes[prefix.Name] = prefix
}

func (okm *OptimizedKeyManager) BuildKey(prefixName string, segments ...string) string {
	okm.mu.RLock()
	prefix, exists := okm.prefixes[prefixName]
	okm.mu.RUnlock()

	if !exists {
		return strings.Join(append([]string{prefixName}, segments...), ":")
	}

	parts := []string{prefix.Name}
	parts = append(parts, segments...)

	if prefix.Version > 1 {
		parts = append(parts, fmt.Sprintf("v%d", prefix.Version))
	}

	return strings.Join(parts, prefix.Separator)
}

func (okm *OptimizedKeyManager) BuildCaptchaKey(captchaID string) string {
	return okm.BuildKey("captcha", captchaID)
}

func (okm *OptimizedKeyManager) BuildSessionKey(sessionID string) string {
	return okm.BuildKey("session", sessionID)
}

func (okm *OptimizedKeyManager) BuildUserKey(userID string) string {
	return okm.BuildKey("user", userID)
}

func (okm *OptimizedKeyManager) BuildStatsKey(metric string) string {
	return okm.BuildKey("stats", metric)
}

func (okm *OptimizedKeyManager) BuildRateLimitKey(identifier string, window int) string {
	return okm.BuildKey("ratelimit", identifier, fmt.Sprintf("w%d", window))
}

func (okm *OptimizedKeyManager) BuildBehaviorKey(sessionID string) string {
	return okm.BuildKey("behavior", sessionID)
}

func (okm *OptimizedKeyManager) BuildConfigKey(configType, configID string) string {
	return okm.BuildKey("config", configType, configID)
}

func (okm *OptimizedKeyManager) BuildLockKey(resource string) string {
	return okm.BuildKey("lock", resource)
}

func (okm *OptimizedKeyManager) BuildPattern(prefixName string) string {
	return okm.BuildKey(prefixName) + ":*"
}

func (okm *OptimizedKeyManager) RecordAccess(key string) {
	okm.mu.Lock()
	defer okm.mu.Unlock()

	stats, exists := okm.keyStats[key]
	if !exists {
		stats = &KeyStats{
			CreationTime: time.Now(),
		}
		okm.keyStats[key] = stats
	}

	atomic.AddInt64(&stats.AccessCount, 1)
	atomic.AddInt64(&stats.AccessCount2, 1)
	stats.LastAccess = time.Now()

	if len(okm.keyStats) > okm.maxKeys {
		okm.cleanup()
	}
}

func (okm *OptimizedKeyManager) RecordHit(key string) {
	okm.mu.Lock()
	defer okm.mu.Unlock()

	stats, exists := okm.keyStats[key]
	if !exists {
		stats = &KeyStats{
			CreationTime: time.Now(),
		}
		okm.keyStats[key] = stats
	}

	atomic.AddInt64(&stats.HitCount, 1)
}

func (okm *OptimizedKeyManager) RecordMiss(key string) {
	okm.mu.Lock()
	defer okm.mu.Unlock()

	stats, exists := okm.keyStats[key]
	if !exists {
		stats = &KeyStats{
			CreationTime: time.Now(),
		}
		okm.keyStats[key] = stats
	}

	atomic.AddInt64(&stats.MissCount, 1)
}

func (okm *OptimizedKeyManager) RecordLatency(key string, latency time.Duration) {
	okm.mu.Lock()
	defer okm.mu.Unlock()

	stats, exists := okm.keyStats[key]
	if !exists {
		stats = &KeyStats{
			CreationTime: time.Now(),
		}
		okm.keyStats[key] = stats
	}

	atomic.AddInt64(&stats.TotalLatency, latency.Nanoseconds())
	atomic.AddInt64(&stats.AccessCount2, 1)
	count := atomic.LoadInt64(&stats.AccessCount2)
	if count > 0 {
		stats.AvgLatency = time.Duration(atomic.LoadInt64(&stats.TotalLatency) / count)
	}
}

func (okm *OptimizedKeyManager) GetKeyStats(key string) *KeyStats {
	okm.mu.RLock()
	defer okm.mu.RUnlock()

	return okm.keyStats[key]
}

func (okm *OptimizedKeyManager) GetHotKeys(n int) []*KeyStats {
	okm.mu.RLock()
	defer okm.mu.RUnlock()

	type keyStatPair struct {
		Key   string
		Stats *KeyStats
	}

	var pairs []keyStatPair
	for key, stats := range okm.keyStats {
		if atomic.LoadInt64(&stats.AccessCount) > 10 {
			pairs = append(pairs, keyStatPair{Key: key, Stats: stats})
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Stats.AccessCount > pairs[j].Stats.AccessCount
	})

	result := make([]*KeyStats, 0, n)
	for i := 0; i < n && i < len(pairs); i++ {
		result = append(result, pairs[i].Stats)
	}

	return result
}

func (okm *OptimizedKeyManager) cleanup() {
	type keyStatPair struct {
		Key   string
		Stats *KeyStats
	}

	var pairs []keyStatPair
	for key, stats := range okm.keyStats {
		pairs = append(pairs, keyStatPair{Key: key, Stats: stats})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Stats.LastAccess.Before(pairs[j].Stats.LastAccess)
	})

	deleteCount := len(okm.keyStats) / 4
	for i := 0; i < deleteCount && i < len(pairs); i++ {
		delete(okm.keyStats, pairs[i].Key)
	}
}

func (okm *OptimizedKeyManager) HashKey(key string) string {
	h := fnv.New32a()
	h.Write([]byte(key))
	return fmt.Sprintf("%x", h.Sum32())
}

func NewExpirationOptimizer(config *OptimizerConfig) *ExpirationOptimizer {
	if config == nil {
		config = DefaultOptimizerConfig
	}

	return &ExpirationOptimizer{
		config:      config,
		policies:    make(map[string]*ExpirationPolicy),
		accessCounts: make(map[string]*AccessCount),
		ttlHistory: make(map[string][]TTLRecord),
		maxHistory: 1000,
	}
}

func (eo *ExpirationOptimizer) RegisterPolicy(policy *ExpirationPolicy) {
	eo.mu.Lock()
	defer eo.mu.Unlock()

	if policy.BaseTTL == 0 {
		policy.BaseTTL = eo.config.DefaultTTL
	}
	if policy.MinTTL == 0 {
		policy.MinTTL = eo.config.MinTTL
	}
	if policy.MaxTTL == 0 {
		policy.MaxTTL = eo.config.MaxTTL
	}

	eo.policies[policy.Name] = policy
}

func (eo *ExpirationOptimizer) CalculateTTL(key string, policyName string) time.Duration {
	eo.mu.RLock()
	policy, exists := eo.policies[policyName]
	eo.mu.RUnlock()

	if !exists {
		return eo.config.DefaultTTL
	}

	accessCount := eo.getAccessCount(key)

	switch policy.Strategy {
	case StrategyFixed:
		return policy.BaseTTL

	case StrategySliding:
		ttl := policy.BaseTTL + policy.SlidingWindow
		if ttl > policy.MaxTTL {
			return policy.MaxTTL
		}
		return ttl

	case StrategyRandom:
		variance := policy.RandomVariance
		randomFactor := 1.0 + (float64(time.Now().UnixNano()%1000)/1000.0-0.5)*2*variance
		ttl := time.Duration(float64(policy.BaseTTL) * randomFactor)

		if ttl < policy.MinTTL {
			return policy.MinTTL
		}
		if ttl > policy.MaxTTL {
			return policy.MaxTTL
		}
		return ttl

	case StrategyAdaptive:
		multiplier := 1.0 + float64(accessCount)*0.01
		ttl := time.Duration(float64(policy.BaseTTL) * multiplier)

		if ttl < policy.MinTTL {
			return policy.MinTTL
		}
		if ttl > policy.MaxTTL {
			return policy.MaxTTL
		}
		return ttl

	case StrategyPredictive:
		return eo.calculatePredictiveTTL(key, policy)

	default:
		return policy.BaseTTL
	}
}

func (eo *ExpirationOptimizer) getAccessCount(key string) int64 {
	eo.mu.RLock()
	defer eo.mu.RUnlock()

	if ac, exists := eo.accessCounts[key]; exists {
		return ac.Count
	}
	return 0
}

func (eo *ExpirationOptimizer) RecordAccess(key string, pattern string) {
	eo.mu.Lock()
	defer eo.mu.Unlock()

	ac, exists := eo.accessCounts[key]
	if !exists {
		ac = &AccessCount{
			Timestamp: time.Now(),
		}
		eo.accessCounts[key] = ac
	}

	atomic.AddInt64(&ac.Count, 1)
	ac.Timestamp = time.Now()
	ac.Pattern = pattern

	if len(eo.accessCounts) > 10000 {
		eo.cleanupAccessCounts()
	}
}

func (eo *ExpirationOptimizer) cleanupAccessCounts() {
	type acPair struct {
		Key string
		AC  *AccessCount
	}

	var pairs []acPair
	for key, ac := range eo.accessCounts {
		pairs = append(pairs, acPair{Key: key, AC: ac})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].AC.Timestamp.Before(pairs[j].AC.Timestamp)
	})

	deleteCount := len(eo.accessCounts) / 4
	for i := 0; i < deleteCount && i < len(pairs); i++ {
		delete(eo.accessCounts, pairs[i].Key)
	}
}

func (eo *ExpirationOptimizer) calculatePredictiveTTL(key string, policy *ExpirationPolicy) time.Duration {
	eo.mu.RLock()
	history, exists := eo.ttlHistory[key]
	eo.mu.RUnlock()

	if !exists || len(history) == 0 {
		return eo.CalculateTTL(key, "adaptive")
	}

	var totalHitRate float64
	for _, record := range history {
		totalHitRate += record.HitRate
	}
	avgHitRate := totalHitRate / float64(len(history))

	accessCount := float64(eo.getAccessCount(key))
	baseMultiplier := 1.0 + accessCount*0.005

	hitRateMultiplier := 1.0 + (avgHitRate-50.0)/100.0
	if hitRateMultiplier < 0.5 {
		hitRateMultiplier = 0.5
	}
	if hitRateMultiplier > 2.0 {
		hitRateMultiplier = 2.0
	}

	ttl := time.Duration(float64(policy.BaseTTL) * baseMultiplier * hitRateMultiplier)

	if ttl < policy.MinTTL {
		return policy.MinTTL
	}
	if ttl > policy.MaxTTL {
		return policy.MaxTTL
	}

	return ttl
}

func (eo *ExpirationOptimizer) RecordTTLUsage(key string, ttl time.Duration, hitRate float64) {
	eo.mu.Lock()
	defer eo.mu.Unlock()

	record := TTLRecord{
		TTL:       ttl,
		Timestamp: time.Now(),
		HitRate:   hitRate,
	}

	eo.ttlHistory[key] = append(eo.ttlHistory[key], record)

	if len(eo.ttlHistory[key]) > eo.maxHistory {
		eo.ttlHistory[key] = eo.ttlHistory[key][1:]
	}
}

func (eo *ExpirationOptimizer) ShouldRefresh(key string, remainingTTL time.Duration) bool {
	eo.mu.RLock()
	defer eo.mu.RUnlock()

	ac, exists := eo.accessCounts[key]
	if !exists || ac.Count == 0 {
		return false
	}

	for _, policy := range eo.policies {
		threshold := time.Duration(float64(policy.BaseTTL) * policy.RefreshRatio)
		return remainingTTL < threshold
	}

	threshold := time.Duration(float64(eo.config.DefaultTTL) * 0.8)
	return remainingTTL < threshold
}

func (eo *ExpirationOptimizer) GetOptimalPolicy(key string) *ExpirationPolicy {
	eo.mu.RLock()
	defer eo.mu.RUnlock()

	ac, exists := eo.accessCounts[key]
	if !exists {
		return eo.getDefaultPolicy()
	}

	pattern := ac.Pattern
	if pattern == "" {
		pattern = eo.classifyAccessPattern(ac)
	}

	for _, policy := range eo.policies {
		if policy.Name == pattern {
			return policy
		}
	}

	return eo.getDefaultPolicy()
}

func (eo *ExpirationOptimizer) classifyAccessPattern(ac *AccessCount) string {
	frequency := float64(ac.Count) / time.Since(ac.Timestamp).Minutes()

	if frequency > 10 {
		return "frequent"
	} else if frequency > 1 {
		return "moderate"
	}
	return "rare"
}

func (eo *ExpirationOptimizer) getDefaultPolicy() *ExpirationPolicy {
	return &ExpirationPolicy{
		Name:         "default",
		BaseTTL:      eo.config.DefaultTTL,
		MinTTL:       eo.config.MinTTL,
		MaxTTL:       eo.config.MaxTTL,
		Strategy:     StrategyAdaptive,
		RefreshRatio: 0.8,
	}
}

func NewPreheatManager(config *OptimizerConfig) *PreheatManager {
	if config == nil {
		config = DefaultOptimizerConfig
	}

	pm := &PreheatManager{
		config:     config,
		profiles:   make(map[string]*PreheatProfile),
		accessLog:  make(map[string]*AccessLog),
		hotKeys:    make(map[string]*HotKeyInfo),
		history:    make(map[string][]time.Time),
		maxHistory: 1000,
		enabled:    config.EnablePreheat,
		executor:   NewPreheatExecutor(config),
	}

	pm.scheduler = &PreheatScheduler{
		profiles: pm.profiles,
		interval: 5 * time.Minute,
		enabled:  true,
	}

	return pm
}

func (pm *PreheatManager) RegisterProfile(profile *PreheatProfile) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if profile.Concurrency == 0 {
		profile.Concurrency = pm.config.PreheatConcurrency
	}
	if profile.BatchSize == 0 {
		profile.BatchSize = pm.config.PreheatBatchSize
	}
	if profile.Timeout == 0 {
		profile.Timeout = 30 * time.Second
	}

	pm.profiles[profile.Name] = profile
}

func (pm *PreheatManager) RecordAccess(key string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	log, exists := pm.accessLog[key]
	if !exists {
		log = &AccessLog{
			Key:       key,
			FirstTime: time.Now(),
		}
		pm.accessLog[key] = log
	}

	atomic.AddInt64(&log.Count, 1)
	log.LastTime = time.Now()

	pm.history[key] = append(pm.history[key], time.Now())
	if len(pm.history[key]) > pm.maxHistory {
		pm.history[key] = pm.history[key][1:]
	}

	pm.updateHotKeys(key)
}

func (pm *PreheatManager) updateHotKeys(key string) {
	log := pm.accessLog[key]
	if log == nil {
		return
	}

	now := time.Now()
	hki, exists := pm.hotKeys[key]
	if !exists {
		hki = &HotKeyInfo{
			Key:       key,
			FirstSeen: now,
		}
		pm.hotKeys[key] = hki
	}

	hki.Count = atomic.LoadInt64(&log.Count)
	hki.LastSeen = now
	hki.Score = pm.calculateHotScore(key)

	threshold := pm.config.HotKeyThreshold
	if hki.Count > threshold {
		hki.IsHot = true
	}
}

func (pm *PreheatManager) calculateHotScore(key string) float64 {
	log := pm.accessLog[key]
	if log == nil {
		return 0
	}

	count := float64(atomic.LoadInt64(&log.Count))
	recency := time.Since(log.LastTime).Seconds()
	frequency := count / math.Max(recency/60.0, 1.0)

	return count*0.6 + frequency*0.4
}

func (pm *PreheatManager) PredictHotKeys(n int) []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	type hotKeyPair struct {
		Key  string
		Info *HotKeyInfo
	}

	var pairs []hotKeyPair
	for key, info := range pm.hotKeys {
		pairs = append(pairs, hotKeyPair{Key: key, Info: info})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Info.Score > pairs[j].Info.Score
	})

	result := make([]string, 0, n)
	for i := 0; i < n && i < len(pairs); i++ {
		result = append(result, pairs[i].Key)
	}

	return result
}

func (pm *PreheatManager) ExecutePreheat(profileName string) error {
	pm.mu.RLock()
	profile, exists := pm.profiles[profileName]
	pm.mu.RUnlock()

	if !exists || !profile.Enabled {
		return fmt.Errorf("profile not found or disabled")
	}

	if len(profile.Keys) == 0 {
		profile.Keys = pm.PredictHotKeys(profile.BatchSize * 2)
	}

	go pm.executor.Execute(profile)
	return nil
}

func (pm *PreheatManager) GetPreheatStats(profileName string) *PreheatStats {
	pm.mu.RLock()
	profile, exists := pm.profiles[profileName]
	pm.mu.RUnlock()

	if !exists {
		return nil
	}

	return &PreheatStats{
		TotalRuns:   profile.TotalRuns,
		SuccessRuns: profile.SuccessRuns,
		FailedRuns:  profile.FailedRuns,
		TotalKeys:   profile.TotalKeys,
		WarmmedKeys: profile.WarmmedCount,
		FailedKeys:  profile.FailedCount,
		AvgDuration: profile.AvgDuration,
		LastRunTime: profile.LastRun,
	}
}

func (pm *PreheatManager) Start() {
	if !pm.enabled {
		return
	}

	go pm.scheduler.Run()
}

func (pm *PreheatManager) Stop() {
	pm.enabled = false
	if pm.scheduler != nil {
		pm.scheduler.Stop()
	}
}

func (pm *PreheatScheduler) Run() {
	ticker := time.NewTicker(pm.interval)
	defer ticker.Stop()

	for pm.enabled {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.checkAndPreheat()
		}
	}
}

func (pm *PreheatScheduler) checkAndPreheat() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, profile := range pm.profiles {
		if !profile.Enabled {
			continue
		}

		if time.Since(profile.LastRun) < pm.interval {
			continue
		}
	}
}

func (pm *PreheatScheduler) Stop() {
	pm.mu.Lock()
	pm.enabled = false
	pm.mu.Unlock()
}

func NewPreheatExecutor(config *OptimizerConfig) *PreheatExecutor {
	return &PreheatExecutor{
		config:  config,
		stats:   &PreheatStats{},
		abortCh: make(chan struct{}),
	}
}

func (pe *PreheatExecutor) Execute(profile *PreheatProfile) {
	if profile == nil || len(profile.Keys) == 0 {
		return
	}

	pe.mu.Lock()
	pe.currentRun = &PreheatRun{
		ProfileName: profile.Name,
		StartTime:   time.Now(),
		Keys:        profile.Keys,
		Progress:    0,
		Status:      "running",
	}
	pe.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), profile.Timeout)
	defer cancel()

	semaphore := make(chan struct{}, profile.Concurrency)
	var wg sync.WaitGroup
	var statsMutex sync.Mutex
	localStats := &PreheatStats{}

	keys := profile.Keys
	batchSize := profile.BatchSize

	for i := 0; i < len(keys); i += batchSize {
		select {
		case <-ctx.Done():
			goto done
		case <-pe.abortCh:
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

			pe.processBatch(ctx, batch, profile, localStats, &statsMutex)
		}(batch)
	}

	wg.Wait()

done:
	pe.mu.Lock()
	if pe.currentRun != nil {
		pe.currentRun.Status = "completed"
		pe.currentRun.Progress = 1.0
	}
	pe.mu.Unlock()

	profile.LastRun = time.Now()
	profile.TotalRuns++
	profile.SuccessRuns += localStats.WarmmedKeys
	profile.FailedCount += localStats.FailedKeys
	profile.WarmmedCount += localStats.WarmmedKeys

	pe.stats.TotalRuns++
	pe.stats.WarmmedKeys += localStats.WarmmedKeys
	pe.stats.FailedKeys += localStats.FailedKeys
}

func (pe *PreheatExecutor) processBatch(ctx context.Context, keys []string, profile *PreheatProfile, stats *PreheatStats, mu *sync.Mutex) {
	for _, key := range keys {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if profile.DataLoader == nil {
			continue
		}

		data, err := profile.DataLoader(ctx, key)
		if err != nil {
			atomic.AddInt64(&stats.FailedKeys, 1)
			continue
		}

		_, err = json.Marshal(data)
		if err != nil {
			atomic.AddInt64(&stats.FailedKeys, 1)
			continue
		}

		atomic.AddInt64(&stats.WarmmedKeys, 1)
		atomic.AddInt64(&stats.TotalKeys, 1)
	}
}

func NewCacheMonitor(config *OptimizerConfig) *CacheMonitor {
	if config == nil {
		config = DefaultOptimizerConfig
	}

	return &CacheMonitor{
		config:         config,
		metrics:        &CacheMetrics{},
		maxAlerts:      100,
		hotKeyTracker:  NewHotKeyTracker(1000, 30*time.Minute),
		latencyTracker: NewLatencyTracker(10000),
		healthChecker:  NewHealthChecker(),
		stopCh:        make(chan struct{}),
	}
}

func (cm *CacheMonitor) RecordHit() {
	atomic.AddInt64(&cm.metrics.TotalHits, 1)
	cm.updateHitRate()
}

func (cm *CacheMonitor) RecordMiss() {
	atomic.AddInt64(&cm.metrics.TotalMisses, 1)
	cm.updateHitRate()
}

func (cm *CacheMonitor) RecordError() {
	atomic.AddInt64(&cm.metrics.TotalErrors, 1)
}

func (cm *CacheMonitor) RecordLatency(latency time.Duration) {
	cm.latencyTracker.Record(latency)

	cm.mu.Lock()
	cm.metrics.AvgLatency = cm.latencyTracker.GetAverage()
	total := atomic.LoadInt64(&cm.metrics.TotalHits) + atomic.LoadInt64(&cm.metrics.TotalMisses)
	if total%100 == 0 {
		cm.metrics.P50Latency = cm.latencyTracker.GetPercentile(0.50)
		cm.metrics.P95Latency = cm.latencyTracker.GetPercentile(0.95)
		cm.metrics.P99Latency = cm.latencyTracker.GetPercentile(0.99)
	}
	cm.mu.Unlock()
}

func (cm *CacheMonitor) RecordAccess(key string, latency time.Duration, hit bool) {
	cm.hotKeyTracker.Record(key, latency)

	if hit {
		cm.RecordHit()
	} else {
		cm.RecordMiss()
	}

	cm.RecordLatency(latency)
}

func (cm *CacheMonitor) RecordL1Hit() {
	atomic.AddInt64(&cm.metrics.L1Hits, 1)
}

func (cm *CacheMonitor) RecordL1Miss() {
	atomic.AddInt64(&cm.metrics.L1Misses, 1)
}

func (cm *CacheMonitor) RecordL2Hit() {
	atomic.AddInt64(&cm.metrics.L2Hits, 1)
}

func (cm *CacheMonitor) RecordL2Miss() {
	atomic.AddInt64(&cm.metrics.L2Misses, 1)
}

func (cm *CacheMonitor) RecordEviction() {
	atomic.AddInt64(&cm.metrics.Evictions, 1)
}

func (cm *CacheMonitor) RecordCompression() {
	atomic.AddInt64(&cm.metrics.Compressions, 1)
}

func (cm *CacheMonitor) updateHitRate() {
	hits := atomic.LoadInt64(&cm.metrics.TotalHits)
	misses := atomic.LoadInt64(&cm.metrics.TotalMisses)
	total := hits + misses

	if total > 0 {
		hitRate := float64(hits) / float64(total) * 100
		cm.mu.Lock()
		cm.metrics.HitRate = hitRate
		cm.mu.Unlock()
	}
}

func (cm *CacheMonitor) GetMetrics() *CacheMetrics {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return &CacheMetrics{
		TotalHits:     cm.metrics.TotalHits,
		TotalMisses:   cm.metrics.TotalMisses,
		TotalErrors:   cm.metrics.TotalErrors,
		HitRate:       cm.metrics.HitRate,
		AvgLatency:    cm.metrics.AvgLatency,
		P50Latency:    cm.metrics.P50Latency,
		P95Latency:    cm.metrics.P95Latency,
		P99Latency:    cm.metrics.P99Latency,
		L1Hits:        cm.metrics.L1Hits,
		L1Misses:      cm.metrics.L1Misses,
		L2Hits:        cm.metrics.L2Hits,
		L2Misses:      cm.metrics.L2Misses,
		Evictions:     cm.metrics.Evictions,
		Compressions:  cm.metrics.Compressions,
	}
}

func (cm *CacheMonitor) GetHotKeys(n int) []*HotKeyInfo {
	return cm.hotKeyTracker.GetTopKeys(n)
}

func (cm *CacheMonitor) CheckAlerts() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.metrics.HitRate < 50.0 && cm.metrics.TotalHits+cm.metrics.TotalMisses > 100 {
		cm.addAlert(AlertLowHitRate, SeverityWarning,
			fmt.Sprintf("Cache hit rate is %.2f%%, below optimal threshold", cm.metrics.HitRate),
			"", map[string]interface{}{"hit_rate": cm.metrics.HitRate})
	}

	if cm.metrics.P99Latency > 100*time.Millisecond {
		cm.addAlert(AlertHighLatency, SeverityWarning,
			fmt.Sprintf("P99 latency is %v, above threshold", cm.metrics.P99Latency),
			"", map[string]interface{}{"p99_latency": cm.metrics.P99Latency})
	}

	total := cm.metrics.TotalHits + cm.metrics.TotalMisses
	if total > 0 {
		errorRate := float64(cm.metrics.TotalErrors) / float64(total) * 100
		if errorRate > 5.0 {
			cm.addAlert(AlertHighErrorRate, SeverityCritical,
				fmt.Sprintf("Cache error rate is %.2f%%, above threshold", errorRate),
				"", map[string]interface{}{"error_rate": errorRate, "errors": cm.metrics.TotalErrors})
		}
	}
}

func (cm *CacheMonitor) addAlert(alertType AlertType, severity AlertSeverity, message, key string, metrics map[string]interface{}) {
	alert := &CacheAlert{
		ID:        fmt.Sprintf("%s-%d", alertType, time.Now().Unix()),
		Type:      alertType,
		Severity:  severity,
		Message:   message,
		Timestamp: time.Now(),
		Key:       key,
		Metrics:   metrics,
		Resolved:  false,
	}

	cm.alerts = append(cm.alerts, alert)

	if len(cm.alerts) > cm.maxAlerts {
		cm.alerts = cm.alerts[1:]
	}
}

func (cm *CacheMonitor) GetAlerts(severity string, limit int) []*CacheAlert {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var filtered []*CacheAlert
	for _, alert := range cm.alerts {
		if severity == "" || string(alert.Severity) == severity {
			filtered = append(filtered, alert)
		}
	}

	if limit > 0 && len(filtered) > limit {
		return filtered[len(filtered)-limit:]
	}

	return filtered
}

func (cm *CacheMonitor) GetHealthStatus() *HealthStatus {
	return cm.healthChecker.GetStatus()
}

func (cm *CacheMonitor) Start() {
	if !cm.config.EnableMonitoring {
		return
	}

	cm.started = true
	go cm.runMonitoring()
}

func (cm *CacheMonitor) Stop() {
	cm.started = false
	close(cm.stopCh)
}

func (cm *CacheMonitor) runMonitoring() {
	ticker := time.NewTicker(cm.config.MonitorInterval)
	defer ticker.Stop()

	for cm.started {
		select {
		case <-cm.stopCh:
			return
		case <-ticker.C:
			cm.CheckAlerts()
		}
	}
}

func NewHotKeyTracker(maxKeys int, window time.Duration) *HotKeyTracker {
	return &HotKeyTracker{
		keys:       make(map[string]*HotKeyInfo),
		maxKeys:    maxKeys,
		window:     window,
		accessLog:  make([]AccessEvent, 0, 10000),
		maxLogSize: 10000,
	}
}

func (hkt *HotKeyTracker) Record(key string, latency time.Duration) {
	hkt.mu.Lock()
	defer hkt.mu.Unlock()

	info, exists := hkt.keys[key]
	if !exists {
		info = &HotKeyInfo{
			Key:       key,
			FirstSeen: time.Now(),
		}
		hkt.keys[key] = info
	}

	atomic.AddInt64(&info.Count, 1)
	info.LastSeen = time.Now()
	info.AvgLatency = time.Duration((int64(info.AvgLatency) + latency.Nanoseconds()) / 2)

	hkt.accessLog = append(hkt.accessLog, AccessEvent{
		Key:       key,
		Timestamp: time.Now(),
		Latency:   latency,
	})

	if len(hkt.accessLog) > hkt.maxLogSize {
		hkt.accessLog = hkt.accessLog[1:]
	}

	if len(hkt.keys)%100 == 0 {
		hkt.cleanupLocked()
	}
}

func (hkt *HotKeyTracker) cleanupLocked() {
	if len(hkt.keys) <= hkt.maxKeys {
		return
	}

	var pairs []struct {
		Key  string
		Info *HotKeyInfo
	}

	for key, info := range hkt.keys {
		pairs = append(pairs, struct {
			Key  string
			Info *HotKeyInfo
		}{Key: key, Info: info})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Info.Count > pairs[j].Info.Count
	})

	for i := hkt.maxKeys; i < len(pairs); i++ {
		delete(hkt.keys, pairs[i].Key)
	}
}

func (hkt *HotKeyTracker) GetTopKeys(n int) []*HotKeyInfo {
	hkt.mu.RLock()
	defer hkt.mu.RUnlock()

	type pair struct {
		Key  string
		Info *HotKeyInfo
	}

	var pairs []pair
	for key, info := range hkt.keys {
		pairs = append(pairs, pair{Key: key, Info: info})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Info.Count > pairs[j].Info.Count
	})

	result := make([]*HotKeyInfo, 0, n)
	for i := 0; i < n && i < len(pairs); i++ {
		result = append(result, pairs[i].Info)
	}

	return result
}

func NewLatencyTracker(maxSize int) *LatencyTracker {
	return &LatencyTracker{
		samples: make([]time.Duration, 0, maxSize),
		maxSize: maxSize,
		buckets: make(map[time.Duration]int64),
	}
}

func (lt *LatencyTracker) Record(latency time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	lt.samples = append(lt.samples, latency)
	if len(lt.samples) > lt.maxSize {
		lt.samples = lt.samples[1:]
	}

	bucket := lt.getBucket(latency)
	lt.buckets[bucket]++
}

func (lt *LatencyTracker) getBucket(latency time.Duration) time.Duration {
	buckets := []time.Duration{
		100 * time.Microsecond,
		500 * time.Microsecond,
		1 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}

	for _, bucket := range buckets {
		if latency <= bucket {
			return bucket
		}
	}

	return 10 * time.Second
}

func (lt *LatencyTracker) GetAverage() time.Duration {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if len(lt.samples) == 0 {
		return 0
	}

	var total int64
	for _, s := range lt.samples {
		total += s.Nanoseconds()
	}

	return time.Duration(total / int64(len(lt.samples)))
}

func (lt *LatencyTracker) GetPercentile(p float64) time.Duration {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if len(lt.samples) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(lt.samples))
	copy(sorted, lt.samples)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	index := int(float64(len(sorted)) * p)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	if index < 0 {
		index = 0
	}

	return sorted[index]
}

func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]HealthCheck),
		status: HealthStatus{
			Checks: make(map[string]HealthCheck),
		},
	}
}

func (hc *HealthChecker) RegisterCheck(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.checks[name] = HealthCheck{
		Name: name,
		Status: "unknown",
	}
}

func (hc *HealthChecker) RunCheck(name string, checkFunc func() error) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	check, exists := hc.checks[name]
	if !exists {
		return
	}

	check.LastRun = time.Now()
	err := checkFunc()

	if err != nil {
		check.Status = "failed"
		check.LastFailure = time.Now()
		check.Message = err.Error()
	} else {
		check.Status = "healthy"
		check.LastSuccess = time.Now()
		check.Message = "OK"
	}

	hc.checks[name] = check
	hc.updateStatus()
}

func (hc *HealthChecker) updateStatus() {
	healthyCount := 0
	totalCount := len(hc.checks)

	for _, check := range hc.checks {
		if check.Status == "healthy" {
			healthyCount++
		}
	}

	if totalCount == 0 {
		hc.status.Overall = "unknown"
		hc.status.Score = 0
	} else if healthyCount == totalCount {
		hc.status.Overall = "healthy"
		hc.status.Score = 100
	} else if healthyCount > 0 {
		hc.status.Overall = "degraded"
		hc.status.Score = float64(healthyCount) / float64(totalCount) * 100
	} else {
		hc.status.Overall = "unhealthy"
		hc.status.Score = 0
	}

	hc.status.Timestamp = time.Now()
	hc.status.Checks = hc.checks
}

func (hc *HealthChecker) GetStatus() *HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	return &HealthStatus{
		Overall:   hc.status.Overall,
		Score:     hc.status.Score,
		Checks:    hc.status.Checks,
		Timestamp: hc.status.Timestamp,
	}
}

func (co *CacheOptimizer) Start() {
	co.mu.Lock()
	defer co.mu.Unlock()

	if co.started {
		return
	}

	co.started = true

	if co.config.EnablePreheat {
		co.preheat.Start()
	}

	if co.config.EnableMonitoring {
		co.monitor.Start()
	}
}

func (co *CacheOptimizer) Stop() {
	co.mu.Lock()
	defer co.mu.Unlock()

	if !co.started {
		return
	}

	co.started = false
	co.cancel()

	if co.config.EnablePreheat {
		co.preheat.Stop()
	}

	if co.config.EnableMonitoring {
		co.monitor.Stop()
	}
}

func (co *CacheOptimizer) GetKeyManager() *OptimizedKeyManager {
	return co.keyManager
}

func (co *CacheOptimizer) GetExpirationOptimizer() *ExpirationOptimizer {
	return co.expiration
}

func (co *CacheOptimizer) GetPreheatManager() *PreheatManager {
	return co.preheat
}

func (co *CacheOptimizer) GetMonitor() *CacheMonitor {
	return co.monitor
}

func (co *CacheOptimizer) RecordCacheAccess(key string, latency time.Duration, hit bool) {
	co.keyManager.RecordAccess(key)
	co.keyManager.RecordLatency(key, latency)

	if hit {
		co.keyManager.RecordHit(key)
	} else {
		co.keyManager.RecordMiss(key)
	}

	co.monitor.RecordAccess(key, latency, hit)
	co.expiration.RecordAccess(key, "auto")
}

func (co *CacheOptimizer) OptimizeKey(key string) string {
	return co.keyManager.HashKey(key)
}

func (co *CacheOptimizer) GetOptimalTTL(key string) time.Duration {
	policy := co.expiration.GetOptimalPolicy(key)
	return co.expiration.CalculateTTL(key, policy.Name)
}

func (co *CacheOptimizer) ShouldPreheat(key string) bool {
	return co.preheat != nil && len(co.preheat.hotKeys) > 0
}

func (co *CacheOptimizer) GetCacheStats() map[string]interface{} {
	metrics := co.monitor.GetMetrics()
	hotKeys := co.monitor.GetHotKeys(10)
	alerts := co.monitor.GetAlerts("", 10)

	return map[string]interface{}{
		"metrics":  metrics,
		"hot_keys": hotKeys,
		"alerts":   alerts,
		"health":   co.monitor.GetHealthStatus(),
	}
}

var (
	globalCacheOptimizer *CacheOptimizer
	optimizerOnce       sync.Once
)

func GetGlobalCacheOptimizer() *CacheOptimizer {
	optimizerOnce.Do(func() {
		globalCacheOptimizer = NewCacheOptimizer(nil)
	})
	return globalCacheOptimizer
}

func InitCacheOptimizer(config *OptimizerConfig) *CacheOptimizer {
	optimizer := NewCacheOptimizer(config)
	globalCacheOptimizer = optimizer
	return optimizer
}

func StartCacheOptimizer() {
	GetGlobalCacheOptimizer().Start()
}

func StopCacheOptimizer() {
	GetGlobalCacheOptimizer().Stop()
}
