package redis

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type AccessPattern struct {
	Key           string
	AccessCount   int64
	LastAccess    time.Time
	AvgAccessTime time.Duration
}

type SmartTTLOptimizer struct {
	mu               sync.RWMutex
	accessPatterns   map[string]*AccessPattern
	baseTTL          time.Duration
	minTTL           time.Duration
	maxTTL           time.Duration
	ttlAdjustments   map[string]time.Duration
	stats            *SmartTTLStats
}

type SmartTTLStats struct {
	TotalAdjustments atomic.Int64
	CacheHits        atomic.Int64
	CacheMisses      atomic.Int64
	MemorySaved      atomic.Int64
}

type SmartTTLConfig struct {
	BaseTTL          time.Duration
	MinTTL           time.Duration
	MaxTTL           time.Duration
	HotKeyThreshold  int64
	ColdKeyThreshold int64
}

var DefaultSmartTTLConfig = &SmartTTLConfig{
	BaseTTL:          10 * time.Minute,
	MinTTL:           1 * time.Minute,
	MaxTTL:           2 * time.Hour,
	HotKeyThreshold:  100,
	ColdKeyThreshold: 5,
}

func NewSmartTTLOptimizer(config *SmartTTLConfig) *SmartTTLOptimizer {
	if config == nil {
		config = DefaultSmartTTLConfig
	}

	return &SmartTTLOptimizer{
		accessPatterns: make(map[string]*AccessPattern),
		baseTTL:        config.BaseTTL,
		minTTL:         config.MinTTL,
		maxTTL:         config.MaxTTL,
		ttlAdjustments: make(map[string]time.Duration),
		stats:          &SmartTTLStats{},
	}
}

func (st *SmartTTLOptimizer) RecordAccess(key string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	pattern, exists := st.accessPatterns[key]
	if !exists {
		pattern = &AccessPattern{
			Key:        key,
			LastAccess: time.Now(),
		}
		st.accessPatterns[key] = pattern
	}

	now := time.Now()
	if pattern.AccessCount > 0 {
		timeSinceLast := now.Sub(pattern.LastAccess)
		pattern.AvgAccessTime = (pattern.AvgAccessTime*time.Duration(pattern.AccessCount) + timeSinceLast) / time.Duration(pattern.AccessCount+1)
	}

	pattern.AccessCount++
	pattern.LastAccess = now
}

func (st *SmartTTLOptimizer) CalculateTTL(key string) time.Duration {
	st.mu.RLock()
	pattern, exists := st.accessPatterns[key]
	st.mu.RUnlock()

	if !exists {
		return st.baseTTL
	}

	// 根据访问频率和平均访问间隔计算调整因子
	factor := st.calculateAdjustmentFactor(pattern)
	adjustedTTL := time.Duration(float64(st.baseTTL) * factor)

	// 确保 TTL 在合理范围内
	if adjustedTTL < st.minTTL {
		adjustedTTL = st.minTTL
	}
	if adjustedTTL > st.maxTTL {
		adjustedTTL = st.maxTTL
	}

	st.stats.TotalAdjustments.Add(1)
	return adjustedTTL
}

func (st *SmartTTLOptimizer) calculateAdjustmentFactor(pattern *AccessPattern) float64 {
	// 访问频率因子
	frequencyFactor := math.Min(1.0+float64(pattern.AccessCount)/50.0, 3.0)

	// 访问模式因子 - 如果平均访问间隔短，说明是热数据
	intervalFactor := 1.0
	if pattern.AvgAccessTime > 0 {
		normalizedInterval := pattern.AvgAccessTime.Minutes() / 30.0 // 30分钟为基准
		if normalizedInterval < 1.0 {
			intervalFactor = 2.0 - normalizedInterval // 短间隔增加 TTL
		} else if normalizedInterval > 5.0 {
			intervalFactor = math.Max(0.3, 1.0 - (normalizedInterval-5.0)/10.0) // 长间隔减少 TTL
		}
	}

	// 综合因子
	return frequencyFactor * intervalFactor
}

func (st *SmartTTLOptimizer) GetHotKeys(threshold int64) []string {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var hotKeys []string
	for key, pattern := range st.accessPatterns {
		if pattern.AccessCount >= threshold {
			hotKeys = append(hotKeys, key)
		}
	}
	return hotKeys
}

func (st *SmartTTLOptimizer) CleanupStalePatterns(olderThan time.Duration) {
	st.mu.Lock()
	defer st.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	for key, pattern := range st.accessPatterns {
		if pattern.LastAccess.Before(cutoff) {
			delete(st.accessPatterns, key)
			delete(st.ttlAdjustments, key)
		}
	}
}

func (st *SmartTTLOptimizer) GetStats() *SmartTTLStats {
	return st.stats
}

type ImprovedBloomFilter struct {
	mu          sync.RWMutex
	bits        []uint64
	m           uint64
	k           uint64
	expectedN   uint64
	fpRate      float64
	insertCount uint64
}

func NewImprovedBloomFilter(expectedItems uint64, falsePositiveRate float64) *ImprovedBloomFilter {
	m := optimalM(expectedItems, falsePositiveRate)
	k := optimalK(m, expectedItems)

	return &ImprovedBloomFilter{
		bits:      make([]uint64, (m+63)/64),
		m:         m,
		k:         k,
		expectedN: expectedItems,
		fpRate:    falsePositiveRate,
	}
}

func (ibf *ImprovedBloomFilter) Add(item string) {
	ibf.mu.Lock()
	defer ibf.mu.Unlock()

	h1 := hashItem(item, 0)
	h2 := hashItem(item, 1)

	for i := uint64(0); i < ibf.k; i++ {
		hash := h1 + i*h2
		bitPos := hash % ibf.m
		wordPos := bitPos / 64
		bitIdx := bitPos % 64
		ibf.bits[wordPos] |= 1 << bitIdx
	}

	ibf.insertCount++
}

func (ibf *ImprovedBloomFilter) MayContain(item string) bool {
	ibf.mu.RLock()
	defer ibf.mu.RUnlock()

	h1 := hashItem(item, 0)
	h2 := hashItem(item, 1)

	for i := uint64(0); i < ibf.k; i++ {
		hash := h1 + i*h2
		bitPos := hash % ibf.m
		wordPos := bitPos / 64
		bitIdx := bitPos % 64
		if (ibf.bits[wordPos] & (1 << bitIdx)) == 0 {
			return false
		}
	}

	return true
}

func (ibf *ImprovedBloomFilter) Clear() {
	ibf.mu.Lock()
	defer ibf.mu.Unlock()

	ibf.bits = make([]uint64, (ibf.m+63)/64)
	ibf.insertCount = 0
}

func (ibf *ImprovedBloomFilter) Count() uint64 {
	ibf.mu.RLock()
	defer ibf.mu.RUnlock()
	return ibf.insertCount
}

func (ibf *ImprovedBloomFilter) EstimatedFalsePositiveRate() float64 {
	ibf.mu.RLock()
	defer ibf.mu.RUnlock()

	if ibf.insertCount == 0 {
		return 0
	}

	// 公式: (1 - e^(-k*n/m))^k
	n := float64(ibf.insertCount)
	m := float64(ibf.m)
	k := float64(ibf.k)

	term := 1 - math.Exp(-k*n/m)
	return math.Pow(term, k)
}

func hashItem(item string, seed uint64) uint64 {
	var h uint64 = seed
	for i := 0; i < len(item); i++ {
		h = h*31 + uint64(item[i])
	}
	return h
}

func optimalM(n uint64, p float64) uint64 {
	return uint64(math.Ceil(-float64(n) * math.Log(p) / (math.Ln2 * math.Ln2)))
}

func optimalK(m, n uint64) uint64 {
	return uint64(math.Max(1, math.Min(30, math.Ceil(math.Ln2*float64(m)/float64(n)))))
}

var (
	globalSmartTTLOptimizer *SmartTTLOptimizer
	smartTTLOnce            sync.Once
)

func InitSmartTTLOptimizer(config *SmartTTLConfig) {
	smartTTLOnce.Do(func() {
		globalSmartTTLOptimizer = NewSmartTTLOptimizer(config)
	})
}

func GetSmartTTLOptimizer() *SmartTTLOptimizer {
	if globalSmartTTLOptimizer == nil {
		InitSmartTTLOptimizer(nil)
	}
	return globalSmartTTLOptimizer
}
