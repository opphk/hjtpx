package redis

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type TTLStrategy int

const (
	TTLStrategyFixed TTLStrategy = iota
	TTLStrategySliding
	TTLStrategyAdaptive
	TTLStrategyPredictive
	TTLStrategyTiered
)

type TTLConfig struct {
	BaseTTL         time.Duration
	MinTTL          time.Duration
	MaxTTL          time.Duration
	Strategy        TTLStrategy
	RefreshInterval time.Duration
	LoadMultiplier  float64
	StalenessCutoff time.Duration
}

var defaultTTLConfig = &TTLConfig{
	BaseTTL:         5 * time.Minute,
	MinTTL:          1 * time.Minute,
	MaxTTL:          60 * time.Minute,
	Strategy:        TTLStrategyAdaptive,
	RefreshInterval: 30 * time.Second,
	LoadMultiplier:  1.5,
	StalenessCutoff: 30 * time.Second,
}

type TTLManager struct {
	mu           sync.RWMutex
	config       *TTLConfig
	accessCounts map[string]*accessCount
	ttlCache     map[string]time.Duration
	stats        *TTLStats
}

type accessCount struct {
	count    int64
	lastTime time.Time
}

type TTLStats struct {
	TotalRequests     atomic.Int64
	CacheHits         atomic.Int64
	CacheMisses       atomic.Int64
	TTLAdjustments    atomic.Int64
	AdaptiveRefreshes atomic.Int64
	AverageTTL        atomic.Int64
}

var (
	globalTTLManager *TTLManager
	ttlManagerOnce   sync.Once
)

func InitTTLManager(config *TTLConfig) {
	ttlManagerOnce.Do(func() {
		if config == nil {
			config = defaultTTLConfig
		}
		globalTTLManager = &TTLManager{
			config:       config,
			accessCounts: make(map[string]*accessCount),
			ttlCache:     make(map[string]time.Duration),
			stats:        &TTLStats{},
		}
		go globalTTLManager.startAdaptiveRefresh()
	})
}

func GetTTLManager() *TTLManager {
	if globalTTLManager == nil {
		InitTTLManager(nil)
	}
	return globalTTLManager
}

func (t *TTLManager) startAdaptiveRefresh() {
	ticker := time.NewTicker(t.config.RefreshInterval)
	defer ticker.Stop()

	for range ticker.C {
		t.adaptiveRefresh()
	}
}

func (t *TTLManager) adaptiveRefresh() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for key, ac := range t.accessCounts {
		if time.Since(ac.lastTime) > t.config.StalenessCutoff {
			delete(t.accessCounts, key)
			delete(t.ttlCache, key)
		}
	}
}

func (t *TTLManager) GetTTL(key string) time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if ttl, ok := t.ttlCache[key]; ok {
		return ttl
	}

	return t.calculateTTL(key, 0)
}

func (t *TTLManager) CalculateTTL(key string, accessCount int64) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.calculateTTL(key, accessCount)
}

func (t *TTLManager) calculateTTL(key string, accessCount int64) time.Duration {
	baseTTL := t.config.BaseTTL

	switch t.config.Strategy {
	case TTLStrategyFixed:
		return baseTTL

	case TTLStrategySliding:
		ac := t.accessCounts[key]
		if ac != nil {
			elapsed := time.Since(ac.lastTime)
			remaining := baseTTL - elapsed
			if remaining > 0 {
				return remaining + t.config.RefreshInterval
			}
		}
		return baseTTL

	case TTLStrategyAdaptive:
		if accessCount > 1000 {
			return time.Duration(float64(baseTTL) * t.config.LoadMultiplier)
		} else if accessCount < 10 {
			return time.Duration(float64(baseTTL) * 0.5)
		}
		return baseTTL

	case TTLStrategyPredictive:
		ac := t.accessCounts[key]
		if ac != nil && ac.count > 0 {
			rate := float64(ac.count) / time.Since(ac.lastTime).Seconds()
			if rate > 10 {
				return time.Duration(float64(baseTTL) * 2)
			} else if rate < 0.1 {
				return time.Duration(float64(baseTTL) * 0.3)
			}
		}
		return baseTTL

	case TTLStrategyTiered:
		if accessCount > 10000 {
			return t.config.MaxTTL
		} else if accessCount > 1000 {
			return 30 * time.Minute
		} else if accessCount > 100 {
			return 15 * time.Minute
		} else if accessCount > 10 {
			return 5 * time.Minute
		}
		return t.config.MinTTL

	default:
		return baseTTL
	}
}

func (t *TTLManager) RecordAccess(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ac := t.accessCounts[key]
	if ac == nil {
		ac = &accessCount{}
		t.accessCounts[key] = ac
	}

	atomic.AddInt64(&ac.count, 1)
	ac.lastTime = time.Now()

	ttl := t.calculateTTL(key, atomic.LoadInt64(&ac.count))
	t.ttlCache[key] = ttl

	t.stats.TotalRequests.Add(1)
}

func (t *TTLManager) GetAndRefreshTTL(key string) (time.Duration, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ttl := t.GetTTL(key)
	ac := t.accessCounts[key]

	if ac == nil {
		return ttl, false
	}

	currentTTL := t.ttlCache[key]
	newTTL := t.calculateTTL(key, atomic.LoadInt64(&ac.count))

	if newTTL != currentTTL {
		t.ttlCache[key] = newTTL
		t.stats.TTLAdjustments.Add(1)
		return newTTL, true
	}

	return ttl, false
}

func (t *TTLManager) SetTTL(key string, ttl time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if ttl < t.config.MinTTL {
		ttl = t.config.MinTTL
	}
	if ttl > t.config.MaxTTL {
		ttl = t.config.MaxTTL
	}

	t.ttlCache[key] = ttl
}

func (t *TTLManager) InvalidateTTL(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.accessCounts, key)
	delete(t.ttlCache, key)
}

func (t *TTLManager) GetStats() *TTLStats {
	return &TTLStats{
		TotalRequests:     t.stats.TotalRequests,
		CacheHits:         t.stats.CacheHits,
		CacheMisses:       t.stats.CacheMisses,
		TTLAdjustments:    t.stats.TTLAdjustments,
		AdaptiveRefreshes: t.stats.AdaptiveRefreshes,
		AverageTTL:        t.stats.AverageTTL,
	}
}

func (t *TTLManager) GetCacheHitRate() float64 {
	total := t.stats.TotalRequests.Load()
	hits := t.stats.CacheHits.Load()

	if total == 0 {
		return 0
	}

	return float64(hits) / float64(total) * 100
}

func (t *TTLManager) Optimize() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.adaptiveRefresh()
	t.stats.AdaptiveRefreshes.Add(1)
}

type TTLObserver struct {
	mu           sync.RWMutex
	observations map[string][]time.Duration
	maxObs       int
}

func NewTTLObserver(maxObs int) *TTLObserver {
	if maxObs <= 0 {
		maxObs = 100
	}
	return &TTLObserver{
		observations: make(map[string][]time.Duration),
		maxObs:       maxObs,
	}
}

func (o *TTLObserver) Record(key string, ttl time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()

	obs := o.observations[key]
	obs = append(obs, ttl)

	if len(obs) > o.maxObs {
		obs = obs[len(obs)-o.maxObs:]
	}

	o.observations[key] = obs
}

func (o *TTLObserver) GetAverageTTL(key string) time.Duration {
	o.mu.RLock()
	defer o.mu.RUnlock()

	obs := o.observations[key]
	if len(obs) == 0 {
		return 0
	}

	var total int64
	for _, ttl := range obs {
		total += ttl.Nanoseconds()
	}

	return time.Duration(total / int64(len(obs)))
}

func (o *TTLObserver) GetTTLDistribution(key string) map[string]int {
	o.mu.RLock()
	defer o.mu.RUnlock()

	obs := o.observations[key]
	dist := map[string]int{
		"<1m":   0,
		"1-5m":  0,
		"5-15m": 0,
		"15-30m": 0,
		">30m":  0,
	}

	for _, ttl := range obs {
		switch {
		case ttl < time.Minute:
			dist["<1m"]++
		case ttl < 5*time.Minute:
			dist["1-5m"]++
		case ttl < 15*time.Minute:
			dist["5-15m"]++
		case ttl < 30*time.Minute:
			dist["15-30m"]++
		default:
			dist[">30m"]++
		}
	}

	return dist
}

func FormatTTL(ttl time.Duration) string {
	if ttl < time.Minute {
		return fmt.Sprintf("%ds", int(ttl.Seconds()))
	} else if ttl < time.Hour {
		return fmt.Sprintf("%dm", int(ttl.Minutes()))
	} else {
		return fmt.Sprintf("%dh", int(ttl.Hours()))
	}
}

type TTLRecommendation struct {
	Key         string
	CurrentTTL  time.Duration
	RecommendedTTL time.Duration
	Reason      string
	Confidence  float64
}

func (t *TTLManager) GetRecommendations() []TTLRecommendation {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var recommendations []TTLRecommendation

	for key, ac := range t.accessCounts {
		currentTTL := t.ttlCache[key]
		accessCount := atomic.LoadInt64(&ac.count)

		recommendedTTL := t.calculateTTL(key, accessCount)

		if recommendedTTL != currentTTL {
			reason := "访问频率变化"
			if recommendedTTL > currentTTL {
				reason = "访问频率增加，建议延长TTL"
			} else {
				reason = "访问频率降低，建议缩短TTL"
			}

			recommendations = append(recommendations, TTLRecommendation{
				Key:            key,
				CurrentTTL:     currentTTL,
				RecommendedTTL: recommendedTTL,
				Reason:         reason,
				Confidence:     0.8,
			})
		}
	}

	return recommendations
}

func (t *TTLManager) ApplyRecommendations(recs []TTLRecommendation) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, rec := range recs {
		if rec.Confidence > 0.7 {
			t.ttlCache[rec.Key] = rec.RecommendedTTL
			t.stats.TTLAdjustments.Add(1)
		}
	}
}

type TTLPolicyRule struct {
	KeyPattern   string
	MinTTL       time.Duration
	MaxTTL       time.Duration
	Strategy     TTLStrategy
	Priority     int
}

type TTLPolicyEngine struct {
	mu    sync.RWMutex
	rules []TTLPolicyRule
}

func NewTTLPolicyEngine() *TTLPolicyEngine {
	return &TTLPolicyEngine{
		rules: make([]TTLPolicyRule, 0),
	}
}

func (e *TTLPolicyEngine) AddRule(rule TTLPolicyRule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rules = append(e.rules, rule)
}

func (e *TTLPolicyEngine) GetRule(key string) *TTLPolicyRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for i := len(e.rules) - 1; i >= 0; i-- {
		rule := &e.rules[i]
		if matchKeyPattern(key, rule.KeyPattern) {
			return rule
		}
	}

	return nil
}

func (e *TTLPolicyEngine) CalculateTTL(key string, accessCount int64) time.Duration {
	rule := e.GetRule(key)

	if rule == nil {
		return defaultTTLConfig.BaseTTL
	}

	ttlManager := GetTTLManager()
	ttl := ttlManager.CalculateTTL(key, accessCount)

	if ttl < rule.MinTTL {
		return rule.MinTTL
	}
	if ttl > rule.MaxTTL {
		return rule.MaxTTL
	}

	return ttl
}

func matchKeyPattern(key, pattern string) bool {
	if pattern == "*" {
		return true
	}

	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}

	return key == pattern
}

func InitializeTTLWithDefaultRules() {
	engine := NewTTLPolicyEngine()

	engine.AddRule(TTLPolicyRule{
		KeyPattern: "captcha:*",
		MinTTL:     5 * time.Minute,
		MaxTTL:     30 * time.Minute,
		Strategy:   TTLStrategyTiered,
		Priority:   10,
	})

	engine.AddRule(TTLPolicyRule{
		KeyPattern: "session:*",
		MinTTL:     15 * time.Minute,
		MaxTTL:     60 * time.Minute,
		Strategy:   TTLStrategySliding,
		Priority:   10,
	})

	engine.AddRule(TTLPolicyRule{
		KeyPattern: "stats:*",
		MinTTL:     1 * time.Minute,
		MaxTTL:     10 * time.Minute,
		Strategy:   TTLStrategyFixed,
		Priority:   5,
	})

	engine.AddRule(TTLPolicyRule{
		KeyPattern: "config:*",
		MinTTL:     30 * time.Minute,
		MaxTTL:     24 * time.Hour,
		Strategy:   TTLStrategyPredictive,
		Priority:   10,
	})

	engine.AddRule(TTLPolicyRule{
		KeyPattern: "user:*",
		MinTTL:     10 * time.Minute,
		MaxTTL:     60 * time.Minute,
		Strategy:   TTLStrategyTiered,
		Priority:   8,
	})
}

func SetKeyTTLWithPolicy(ctx context.Context, key string, value interface{}, accessCount int64) error {
	engine := NewTTLPolicyEngine()
	ttl := engine.CalculateTTL(key, accessCount)

	client := GetClient()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	return client.Set(ctx, key, value, ttl).Err()
}
