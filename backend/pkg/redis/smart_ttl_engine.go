package redis

import (
	"sync"
	"sync/atomic"
	"time"
)

type AccessPattern string

const (
	AccessPatternUnknown   AccessPattern = "unknown"
	AccessPatternFrequent  AccessPattern = "frequent"
	AccessPatternModerate  AccessPattern = "moderate"
	AccessPatternRare      AccessPattern = "rare"
	AccessPatternBursty    AccessPattern = "bursty"
)

type TTLStrategyType string

const (
	TTLStrategyFixed          TTLStrategyType = "fixed"
	TTLStrategySliding        TTLStrategyType = "sliding"
	TTLStrategyAdaptive       TTLStrategyType = "adaptive"
	TTLStrategyAccessBased    TTLStrategyType = "access_based"
	TTLStrategyHotKey         TTLStrategyType = "hot_key"
)

type SmartTTLEngineConfig struct {
	BaseTTL               time.Duration
	MinTTL                time.Duration
	MaxTTL                time.Duration
	Strategy              TTLStrategyType
	SlidingWindow         time.Duration
	FrequentThreshold     int64
	ModerateThreshold     int64
	RareThreshold         int64
	HotKeyMultiplier      float64
	NormalMultiplier      float64
	RareMultiplier        float64
	DecayInterval         time.Duration
	AccessTrackingEnabled bool
	EnableJitter          bool
	JitterFactor          float64
}

var DefaultSmartTTLEngineConfig = &SmartTTLEngineConfig{
	BaseTTL:               10 * time.Minute,
	MinTTL:                1 * time.Minute,
	MaxTTL:                1 * time.Hour,
	Strategy:              TTLStrategyAdaptive,
	SlidingWindow:         1 * time.Minute,
	FrequentThreshold:     100,
	ModerateThreshold:     20,
	RareThreshold:         5,
	HotKeyMultiplier:      2.0,
	NormalMultiplier:      1.0,
	RareMultiplier:        0.5,
	DecayInterval:         5 * time.Minute,
	AccessTrackingEnabled: true,
	EnableJitter:          true,
	JitterFactor:          0.2,
}

type KeyAccessTracker struct {
	accessCount    atomic.Int64
	lastAccessTime atomic.Value
	firstAccess    atomic.Value
}

type SmartTTLEngine struct {
	config         *SmartTTLEngineConfig
	keyTrackers    *sync.Map
	decayTicker    *time.Ticker
	stopCh         chan struct{}
	running        atomic.Bool
	mu             sync.Mutex
}

func NewSmartTTLEngine(config *SmartTTLEngineConfig) *SmartTTLEngine {
	if config == nil {
		config = DefaultSmartTTLEngineConfig
	}

	engine := &SmartTTLEngine{
		config:      config,
		keyTrackers: &sync.Map{},
		stopCh:      make(chan struct{}),
	}

	if config.AccessTrackingEnabled {
		engine.startDecay()
	}

	return engine
}

func (engine *SmartTTLEngine) RecordAccess(key string) {
	if !engine.config.AccessTrackingEnabled {
		return
	}

	tracker, _ := engine.keyTrackers.LoadOrStore(key, &KeyAccessTracker{})
	t := tracker.(*KeyAccessTracker)
	t.accessCount.Add(1)
	t.lastAccessTime.Store(time.Now())
	if t.firstAccess.Load() == nil {
		t.firstAccess.Store(time.Now())
	}
}

func (engine *SmartTTLEngine) CalculateTTL(key string) time.Duration {
	switch engine.config.Strategy {
	case TTLStrategyFixed:
		return engine.config.BaseTTL

	case TTLStrategySliding:
		return engine.config.BaseTTL + engine.config.SlidingWindow

	case TTLStrategyAdaptive:
		return engine.calculateAdaptiveTTL(key)

	case TTLStrategyAccessBased:
		return engine.calculateAccessBasedTTL(key)

	case TTLStrategyHotKey:
		return engine.calculateHotKeyTTL(key)

	default:
		return engine.config.BaseTTL
	}
}

func (engine *SmartTTLEngine) calculateAdaptiveTTL(key string) time.Duration {
	tracker, exists := engine.keyTrackers.Load(key)
	if !exists {
		return engine.config.BaseTTL
	}

	t := tracker.(*KeyAccessTracker)
	accessCount := t.accessCount.Load()
	lastAccess := t.lastAccessTime.Load()

	var age time.Duration
	if lastAccess != nil {
		age = time.Since(lastAccess.(time.Time))
	}

	var multiplier float64

	switch {
	case accessCount >= engine.config.FrequentThreshold:
		multiplier = engine.config.HotKeyMultiplier
	case accessCount >= engine.config.ModerateThreshold:
		multiplier = engine.config.NormalMultiplier
	case accessCount >= engine.config.RareThreshold:
		multiplier = engine.config.RareMultiplier
	default:
		multiplier = engine.config.RareMultiplier * 0.5
	}

	if age > engine.config.BaseTTL {
		multiplier *= 0.5
	}

	ttl := time.Duration(float64(engine.config.BaseTTL) * multiplier)
	return engine.applyBoundsAndJitter(ttl)
}

func (engine *SmartTTLEngine) calculateAccessBasedTTL(key string) time.Duration {
	tracker, exists := engine.keyTrackers.Load(key)
	if !exists {
		return engine.config.BaseTTL
	}

	t := tracker.(*KeyAccessTracker)
	accessCount := t.accessCount.Load()

	pattern := engine.detectAccessPattern(key, accessCount)

	switch pattern {
	case AccessPatternFrequent:
		ttl := engine.config.BaseTTL * 2
		return engine.applyBoundsAndJitter(ttl)

	case AccessPatternModerate:
		return engine.applyBoundsAndJitter(engine.config.BaseTTL)

	case AccessPatternRare:
		ttl := engine.config.BaseTTL / 2
		return engine.applyBoundsAndJitter(ttl)

	case AccessPatternBursty:
		ttl := engine.config.BaseTTL * 3
		return engine.applyBoundsAndJitter(ttl)

	default:
		return engine.config.BaseTTL
	}
}

func (engine *SmartTTLEngine) calculateHotKeyTTL(key string) time.Duration {
	tracker, exists := engine.keyTrackers.Load(key)
	if !exists {
		return engine.config.BaseTTL
	}

	t := tracker.(*KeyAccessTracker)
	accessCount := t.accessCount.Load()

	if accessCount >= engine.config.FrequentThreshold {
		ttl := engine.config.BaseTTL * time.Duration(engine.config.HotKeyMultiplier)
		return engine.applyBoundsAndJitter(ttl)
	}

	return engine.config.BaseTTL
}

func (engine *SmartTTLEngine) detectAccessPattern(key string, accessCount int64) AccessPattern {
	if accessCount >= engine.config.FrequentThreshold {
		return AccessPatternFrequent
	}

	if accessCount >= engine.config.ModerateThreshold {
		return AccessPatternModerate
	}

	if accessCount >= engine.config.RareThreshold {
		return AccessPatternRare
	}

	return AccessPatternUnknown
}

func (engine *SmartTTLEngine) applyBoundsAndJitter(ttl time.Duration) time.Duration {
	if ttl < engine.config.MinTTL {
		ttl = engine.config.MinTTL
	}
	if ttl > engine.config.MaxTTL {
		ttl = engine.config.MaxTTL
	}

	if engine.config.EnableJitter && engine.config.JitterFactor > 0 {
		jitter := time.Duration(float64(ttl) * engine.config.JitterFactor * (float64(time.Now().UnixNano()%1000)/1000.0 - 0.5) * 2)
		ttl += jitter
	}

	return ttl
}

func (engine *SmartTTLEngine) startDecay() {
	engine.running.Store(true)
	engine.decayTicker = time.NewTicker(engine.config.DecayInterval)

	go func() {
		for {
			select {
			case <-engine.stopCh:
				return
			case <-engine.decayTicker.C:
				engine.decayAccessCounts()
			}
		}
	}()
}

func (engine *SmartTTLEngine) decayAccessCounts() {
	engine.keyTrackers.Range(func(key, value interface{}) bool {
		tracker := value.(*KeyAccessTracker)
		current := tracker.accessCount.Load()
		if current > 0 {
			newCount := int64(float64(current) * 0.8)
			tracker.accessCount.Store(newCount)
		}
		return true
	})
}

func (engine *SmartTTLEngine) GetAccessPattern(key string) AccessPattern {
	tracker, exists := engine.keyTrackers.Load(key)
	if !exists {
		return AccessPatternUnknown
	}

	t := tracker.(*KeyAccessTracker)
	accessCount := t.accessCount.Load()
	return engine.detectAccessPattern(key, accessCount)
}

func (engine *SmartTTLEngine) GetAccessStats(key string) (int64, time.Time) {
	tracker, exists := engine.keyTrackers.Load(key)
	if !exists {
		return 0, time.Time{}
	}

	t := tracker.(*KeyAccessTracker)
	lastAccess := t.lastAccessTime.Load()
	var lastAccessTime time.Time
	if lastAccess != nil {
		lastAccessTime = lastAccess.(time.Time)
	}

	return t.accessCount.Load(), lastAccessTime
}

func (engine *SmartTTLEngine) SetStrategy(strategy TTLStrategyType) {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	engine.config.Strategy = strategy
}

func (engine *SmartTTLEngine) GetStrategy() TTLStrategyType {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	return engine.config.Strategy
}

func (engine *SmartTTLEngine) SetCustomTTL(keyPattern string, ttl time.Duration) {
}

func (engine *SmartTTLEngine) Stop() {
	if engine.decayTicker != nil {
		engine.decayTicker.Stop()
	}
	close(engine.stopCh)
	engine.running.Store(false)
}

func (engine *SmartTTLEngine) IsRunning() bool {
	return engine.running.Load()
}

func (engine *SmartTTLEngine) GetConfig() *SmartTTLEngineConfig {
	return engine.config
}

var globalSmartTTLEngine *SmartTTLEngine
var globalSmartTTLEngineOnce sync.Once

func InitSmartTTLEngine(config *SmartTTLEngineConfig) {
	globalSmartTTLEngineOnce.Do(func() {
		globalSmartTTLEngine = NewSmartTTLEngine(config)
	})
}

func GetSmartTTLEngine() *SmartTTLEngine {
	if globalSmartTTLEngine == nil {
		InitSmartTTLEngine(nil)
	}
	return globalSmartTTLEngine
}
