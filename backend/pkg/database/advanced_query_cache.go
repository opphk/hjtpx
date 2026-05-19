package database

import (
	"crypto/md5"
	"encoding/hex"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

type IntelligentCacheStrategy struct {
	mu             sync.RWMutex
	strategies      map[string]*CacheStrategyConfig
	globalConfig    *GlobalCacheConfig
	performanceData *PerformanceData
	adaptiveEnabled bool
}

type CacheStrategyConfig struct {
	TableName       string
	StrategyType    string
	TTL             time.Duration
	MaxSize         int
	Priority        int
	PreloadEnabled  bool
	InvalidateOnWrite bool
	CompressionEnabled bool
	LastAccessTime time.Time
	AccessCount     int64
	HitRate         float64
}

type GlobalCacheConfig struct {
	DefaultTTL       time.Duration
	MaxCacheSize     int
	EvictionPolicy   string
	CompressionEnabled bool
	StatsEnabled     bool
	WarmupEnabled    bool
}

type PerformanceData struct {
	TotalHits    int64
	TotalMisses  int64
	TotalSets    int64
	TotalEvicts  int64
	AvgLatency   time.Duration
	PeakLatency  time.Duration
	MemoryUsage  int64
	StartTime    time.Time
}

type CacheOptimizationReport struct {
	Timestamp       time.Time
	StrategiesCount int
	GlobalConfig    *GlobalCacheConfig
	Performance     *PerformanceData
	Recommendations []string
	TopTables       []TableCacheStats
}

type TableCacheStats struct {
	TableName      string
	HitRate        float64
	AccessCount    int64
	MemoryUsage    int64
	RecommendedTTL time.Duration
}

var intelligentStrategy *IntelligentCacheStrategy

func InitIntelligentCacheStrategy(cfg *GlobalCacheConfig) {
	intelligentStrategy = &IntelligentCacheStrategy{
		strategies:      make(map[string]*CacheStrategyConfig),
		globalConfig:     cfg,
		performanceData:  &PerformanceData{StartTime: time.Now()},
		adaptiveEnabled:  true,
	}

	go intelligentStrategy.runOptimizationLoop()
	log.Println("Intelligent cache strategy initialized with adaptive optimization")
}

func GetIntelligentCacheStrategy() *IntelligentCacheStrategy {
	if intelligentStrategy == nil {
		intelligentStrategy = &IntelligentCacheStrategy{
			strategies:      make(map[string]*CacheStrategyConfig),
			globalConfig:     &GlobalCacheConfig{DefaultTTL: 5 * time.Minute},
			performanceData:  &PerformanceData{StartTime: time.Now()},
			adaptiveEnabled:  true,
		}
	}
	return intelligentStrategy
}

func (s *IntelligentCacheStrategy) RegisterStrategy(tableName string, config *CacheStrategyConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	config.LastAccessTime = time.Now()
	s.strategies[tableName] = config
	log.Printf("[CACHE_STRATEGY] Registered strategy for table %s: %s (TTL: %v)",
		tableName, config.StrategyType, config.TTL)
}

func (s *IntelligentCacheStrategy) GetStrategy(tableName string) *CacheStrategyConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if strategy, ok := s.strategies[tableName]; ok {
		strategy.LastAccessTime = time.Now()
		atomic.AddInt64(&strategy.AccessCount, 1)
		return strategy
	}

	return &CacheStrategyConfig{
		TableName:    tableName,
		StrategyType: "default",
		TTL:          s.globalConfig.DefaultTTL,
		MaxSize:      s.globalConfig.MaxCacheSize,
		Priority:     1,
	}
}

func (s *IntelligentCacheStrategy) CalculateOptimalTTL(tableName string, accessFrequency int64, avgLatency time.Duration) time.Duration {
	baseTTL := s.globalConfig.DefaultTTL

	if accessFrequency > 10000 {
		baseTTL = baseTTL * 2
	} else if accessFrequency < 100 {
		baseTTL = baseTTL / 2
	}

	if avgLatency > 100*time.Millisecond {
		baseTTL = time.Duration(float64(baseTTL) * 1.5)
	} else if avgLatency < 10*time.Millisecond {
		baseTTL = time.Duration(float64(baseTTL) * 0.8)
	}

	if baseTTL < 1*time.Minute {
		baseTTL = 1 * time.Minute
	}
	if baseTTL > 30*time.Minute {
		baseTTL = 30 * time.Minute
	}

	return baseTTL
}

func (s *IntelligentCacheStrategy) RecordHit(tableName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	atomic.AddInt64(&s.performanceData.TotalHits, 1)

	if strategy, ok := s.strategies[tableName]; ok {
		total := atomic.LoadInt64(&s.performanceData.TotalHits) + atomic.LoadInt64(&s.performanceData.TotalMisses)
		if total > 0 {
			strategy.HitRate = (strategy.HitRate*float64(total-1) + 100) / float64(total)
		}
	}
}

func (s *IntelligentCacheStrategy) RecordMiss(tableName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	atomic.AddInt64(&s.performanceData.TotalMisses, 1)

	if strategy, ok := s.strategies[tableName]; ok {
		total := atomic.LoadInt64(&s.performanceData.TotalHits) + atomic.LoadInt64(&s.performanceData.TotalMisses)
		if total > 0 {
			hits := float64(atomic.LoadInt64(&s.performanceData.TotalHits))
			strategy.HitRate = hits / float64(total) * 100
		}
	}
}

func (s *IntelligentCacheStrategy) runOptimizationLoop() {
	if !s.adaptiveEnabled {
		return
	}

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.analyzeAndOptimize()
	}
}

func (s *IntelligentCacheStrategy) analyzeAndOptimize() {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Println("[CACHE_OPTIMIZATION] Running adaptive optimization analysis...")

	for tableName, strategy := range s.strategies {
		if strategy.HitRate < 30 && strategy.AccessCount > 100 {
			log.Printf("[CACHE_OPTIMIZATION] Table %s has low hit rate (%.2f%%), reducing priority",
				tableName, strategy.HitRate)
			strategy.Priority = 1
			strategy.TTL = strategy.TTL / 2
		} else if strategy.HitRate > 80 && strategy.AccessCount > 1000 {
			log.Printf("[CACHE_OPTIMIZATION] Table %s has high hit rate (%.2f%%), increasing TTL",
				tableName, strategy.HitRate)
			strategy.Priority = 3
			strategy.TTL = time.Duration(float64(strategy.TTL) * 1.5)
		}

		if strategy.TTL > 30*time.Minute {
			strategy.TTL = 30 * time.Minute
		}
		if strategy.TTL < 1*time.Minute {
			strategy.TTL = 1 * time.Minute
		}
	}
}

func (s *IntelligentCacheStrategy) GenerateReport() *CacheOptimizationReport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	report := &CacheOptimizationReport{
		Timestamp:       time.Now(),
		StrategiesCount: len(s.strategies),
		GlobalConfig:    s.globalConfig,
		Performance:     s.performanceData,
		Recommendations:  s.generateRecommendations(),
		TopTables:       s.getTopTables(),
	}

	return report
}

func (s *IntelligentCacheStrategy) generateRecommendations() []string {
	var recommendations []string

	total := atomic.LoadInt64(&s.performanceData.TotalHits) + atomic.LoadInt64(&s.performanceData.TotalMisses)
	if total == 0 {
		recommendations = append(recommendations, "Cache not yet warmed up, monitor after some usage")
		return recommendations
	}

	hitRate := float64(atomic.LoadInt64(&s.performanceData.TotalHits)) / float64(total) * 100

	if hitRate < 50 {
		recommendations = append(recommendations, "Overall hit rate is low (<50%), consider reviewing cache strategies")
		recommendations = append(recommendations, "Check for cache invalidation issues or insufficient TTL")
	}

	if atomic.LoadInt64(&s.performanceData.TotalMisses) > atomic.LoadInt64(&s.performanceData.TotalHits)*2 {
		recommendations = append(recommendations, "Miss rate is very high, consider enabling cache warmup")
	}

	if s.performanceData.AvgLatency > 50*time.Millisecond {
		recommendations = append(recommendations, "Average cache latency is high, consider enabling compression")
	}

	if len(s.strategies) < 5 {
		recommendations = append(recommendations, "Few table-specific strategies registered, add more for better optimization")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Cache performance is within acceptable ranges")
	}

	return recommendations
}

func (s *IntelligentCacheStrategy) getTopTables() []TableCacheStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tables []TableCacheStats
	for tableName, strategy := range s.strategies {
		tables = append(tables, TableCacheStats{
			TableName:      tableName,
			HitRate:        strategy.HitRate,
			AccessCount:    atomic.LoadInt64(&strategy.AccessCount),
			RecommendedTTL: s.CalculateOptimalTTL(tableName, atomic.LoadInt64(&strategy.AccessCount), s.performanceData.AvgLatency),
		})
	}

	return tables
}

func (s *IntelligentCacheStrategy) EnableAdaptiveOptimization(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.adaptiveEnabled = enabled
	log.Printf("[CACHE_STRATEGY] Adaptive optimization %s", map[bool]string{true: "enabled", false: "disabled"}[enabled])
}

type QueryCacheOptimizer struct {
	mu        sync.RWMutex
	cache     map[string]*QueryCacheEntry
	maxSize   int
	evictionPolicy string
	hitRate   float64
	enabled   bool
}

type QueryCacheEntry struct {
	Key         string
	Query       string
	Result      interface{}
	CreatedAt   time.Time
	AccessedAt  time.Time
	AccessCount int64
	TTL         time.Duration
	Compressed  bool
	Tags        []string
}

func NewQueryCacheOptimizer(maxSize int, policy string) *QueryCacheOptimizer {
	return &QueryCacheOptimizer{
		cache:          make(map[string]*QueryCacheEntry),
		maxSize:        maxSize,
		evictionPolicy: policy,
		hitRate:        0,
		enabled:        true,
	}
}

func (o *QueryCacheOptimizer) Set(key string, query string, result interface{}, ttl time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if len(o.cache) >= o.maxSize {
		o.evict()
	}

	entry := &QueryCacheEntry{
		Key:         key,
		Query:       query,
		Result:      result,
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 1,
		TTL:         ttl,
		Tags:        o.extractTags(query),
	}

	o.cache[key] = entry
}

func (o *QueryCacheOptimizer) Get(key string) (interface{}, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()

	entry, exists := o.cache[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.CreatedAt.Add(entry.TTL)) {
		delete(o.cache, key)
		return nil, false
	}

	entry.AccessedAt = time.Now()
	entry.AccessCount++
	o.updateHitRate(true)

	return entry.Result, true
}

func (o *QueryCacheOptimizer) evict() {
	if len(o.cache) == 0 {
		return
	}

	var keyToEvict string

	switch o.evictionPolicy {
	case "lru":
		keyToEvict = o.evictLRU()
	case "lfu":
		keyToEvict = o.evictLFU()
	case "fifo":
		keyToEvict = o.evictFIFO()
	default:
		keyToEvict = o.evictLRU()
	}

	if keyToEvict != "" {
		delete(o.cache, keyToEvict)
	}
}

func (o *QueryCacheOptimizer) evictLRU() string {
	var oldestKey string
	var oldestTime time.Time

	for k, v := range o.cache {
		if oldestKey == "" || v.AccessedAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.AccessedAt
		}
	}

	return oldestKey
}

func (o *QueryCacheOptimizer) evictLFU() string {
	var lowestFreqKey string
	var lowestFreq int64 = 0

	for k, v := range o.cache {
		if lowestFreqKey == "" || v.AccessCount < lowestFreq {
			lowestFreqKey = k
			lowestFreq = v.AccessCount
		}
	}

	return lowestFreqKey
}

func (o *QueryCacheOptimizer) evictFIFO() string {
	var oldestKey string
	var oldestTime time.Time

	for k, v := range o.cache {
		if oldestKey == "" || v.CreatedAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.CreatedAt
		}
	}

	return oldestKey
}

func (o *QueryCacheOptimizer) updateHitRate(hit bool) {
	total := int64(len(o.cache))
	if total == 0 {
		o.hitRate = 0
		return
	}

	hits := float64(0)
	for _, v := range o.cache {
		hits += float64(v.AccessCount)
	}

	o.hitRate = hits / float64(total) * 100
}

func (o *QueryCacheOptimizer) InvalidateByTag(tag string) int {
	o.mu.Lock()
	defer o.mu.Unlock()

	count := 0
	for key, entry := range o.cache {
		for _, t := range entry.Tags {
			if t == tag {
				delete(o.cache, key)
				count++
				break
			}
		}
	}

	return count
}

func (o *QueryCacheOptimizer) InvalidateByPattern(pattern string) int {
	o.mu.Lock()
	defer o.mu.Unlock()

	count := 0
	for key := range o.cache {
		if containsPattern(key, pattern) {
			delete(o.cache, key)
			count++
		}
	}

	return count
}

func containsPattern(s, pattern string) bool {
	for i := 0; i <= len(s)-len(pattern); i++ {
		if s[i:i+len(pattern)] == pattern {
			return true
		}
	}
	return false
}

func (o *QueryCacheOptimizer) extractTags(query string) []string {
	var tags []string

	tables := extractTableNames(query)
	tags = append(tags, tables...)

	if strings.Contains(query, "WHERE") {
		tags = append(tags, "filtered")
	}
	if strings.Contains(query, "JOIN") {
		tags = append(tags, "joined")
	}
	if strings.Contains(query, "ORDER BY") {
		tags = append(tags, "sorted")
	}

	return tags
}

func extractTableNames(query string) []string {
	var tables []string

	patterns := []string{"FROM", "JOIN", "INTO", "UPDATE", "TABLE"}

	for _, pattern := range patterns {
		if idx := indexOf(query, pattern); idx >= 0 {
			start := idx + len(pattern)
			end := start
			for end < len(query) && (query[end] == ' ' || query[end] == '\t') {
				end++
			}
			if end < len(query) {
				name := extractNextWord(query[end:])
				tables = append(tables, name)
			}
		}
	}

	return tables
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func extractNextWord(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' || s[i] == ',' || s[i] == ';' || s[i] == ')' {
			break
		}
		result = append(result, s[i])
	}
	return string(result)
}

func (o *QueryCacheOptimizer) GetStats() map[string]interface{} {
	o.mu.RLock()
	defer o.mu.RUnlock()

	stats := map[string]interface{}{
		"size":           len(o.cache),
		"max_size":       o.maxSize,
		"eviction_policy": o.evictionPolicy,
		"hit_rate":       o.hitRate,
		"enabled":        o.enabled,
	}

	var totalAccess int64
	for _, entry := range o.cache {
		totalAccess += entry.AccessCount
	}
	stats["total_accesses"] = totalAccess

	return stats
}

func (o *QueryCacheOptimizer) Clear() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.cache = make(map[string]*QueryCacheEntry)
}

func (o *QueryCacheOptimizer) Enable() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.enabled = true
}

func (o *QueryCacheOptimizer) Disable() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.enabled = false
}

type PreparedQueryCache struct {
	mu        sync.RWMutex
	queries   map[string]*PreparedQuery
	maxSize   int
	hitCount  int64
	missCount int64
	enabled   bool
}

type PreparedQuery struct {
	Query       string
	Plan        string
	ExecCount   int64
	AvgDuration time.Duration
	CreatedAt   time.Time
	LastUsed    time.Time
}

func NewPreparedQueryCache(maxSize int) *PreparedQueryCache {
	return &PreparedQueryCache{
		queries: make(map[string]*PreparedQuery),
		maxSize: maxSize,
		enabled: true,
	}
}

func (c *PreparedQueryCache) Register(query string, plan string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.queries) >= c.maxSize {
		c.evictLeastUsed()
	}

	c.queries[query] = &PreparedQuery{
		Query:       query,
		Plan:        plan,
		ExecCount:   0,
		AvgDuration: 0,
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
	}

	return nil
}

func (c *PreparedQueryCache) Get(query string) (*PreparedQuery, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prepared, exists := c.queries[query]
	if !exists {
		c.missCount++
		return nil, false
	}

	prepared.LastUsed = time.Now()
	prepared.ExecCount++
	c.hitCount++

	return prepared, true
}

func (c *PreparedQueryCache) UpdateStats(query string, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if prepared, exists := c.queries[query]; exists {
		total := prepared.ExecCount * prepared.AvgDuration.Nanoseconds()
		prepared.AvgDuration = time.Duration(total/duration.Nanoseconds() + duration.Nanoseconds()/prepared.ExecCount)
	}
}

func (c *PreparedQueryCache) evictLeastUsed() {
	var oldestKey string
	var oldestTime time.Time

	for k, v := range c.queries {
		if oldestKey == "" || v.LastUsed.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.LastUsed
		}
	}

	if oldestKey != "" {
		delete(c.queries, oldestKey)
	}
}

func (c *PreparedQueryCache) GetTopQueries(limit int) []*PreparedQuery {
	c.mu.RLock()
	defer c.mu.RUnlock()

	queries := make([]*PreparedQuery, 0, len(c.queries))
	for _, q := range c.queries {
		queries = append(queries, q)
	}

	if len(queries) <= limit {
		return queries
	}

	for i := 0; i < len(queries)-1; i++ {
		for j := i + 1; j < len(queries); j++ {
			if queries[j].ExecCount > queries[i].ExecCount {
				queries[i], queries[j] = queries[j], queries[i]
			}
		}
	}

	return queries[:limit]
}

func (c *PreparedQueryCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hitCount + c.missCount
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(c.hitCount) / float64(total) * 100
	}

	return map[string]interface{}{
		"total_queries": len(c.queries),
		"max_size":      c.maxSize,
		"hit_count":     c.hitCount,
		"miss_count":    c.missCount,
		"hit_rate":      hitRate,
		"enabled":       c.enabled,
	}
}

var globalQueryCacheOptimizer *QueryCacheOptimizer
var globalPreparedQueryCache *PreparedQueryCache

func init() {
	globalQueryCacheOptimizer = NewQueryCacheOptimizer(1000, "lru")
	globalPreparedQueryCache = NewPreparedQueryCache(500)
}

func GetQueryCacheOptimizer() *QueryCacheOptimizer {
	return globalQueryCacheOptimizer
}

func GetPreparedQueryCache() *PreparedQueryCache {
	return globalPreparedQueryCache
}

type AdvancedQueryOptimizer struct {
	db                  *gorm.DB
	preparedStmts       *PreparedQueryCache
	slowQueryThreshold  time.Duration
	enableQueryAnalysis bool
	mu                  sync.RWMutex
	queryPatterns       map[string]*QueryPatternInfo
}

type QueryPatternInfo struct {
	Query         string
	ExecutionCount int64
	TotalDuration time.Duration
	AvgDuration   time.Duration
	LastExecuted  time.Time
	SuggestedIndex string
}

func NewAdvancedQueryOptimizer(db *gorm.DB, threshold time.Duration) *AdvancedQueryOptimizer {
	return &AdvancedQueryOptimizer{
		db:                  db,
		preparedStmts:      NewPreparedQueryCache(100),
		slowQueryThreshold:  threshold,
		enableQueryAnalysis: true,
		queryPatterns:      make(map[string]*QueryPatternInfo),
	}
}

func (qo *AdvancedQueryOptimizer) OptimizeAll() error {
	return nil
}

type QueryCacheManager struct {
	optimizer      *QueryCacheOptimizer
	preparedCache  *PreparedQueryCache
	strategy       *IntelligentCacheStrategy
	mu             sync.RWMutex
	enabled        bool
}

var globalCacheManager *QueryCacheManager

func InitQueryCacheManager() {
	globalCacheManager = &QueryCacheManager{
		optimizer:     globalQueryCacheOptimizer,
		preparedCache: globalPreparedQueryCache,
		strategy:      GetIntelligentCacheStrategy(),
		enabled:       true,
	}
}

func GetQueryCacheManager() *QueryCacheManager {
	if globalCacheManager == nil {
		InitQueryCacheManager()
	}
	return globalCacheManager
}

func (m *QueryCacheManager) RecordQuery(query string, result interface{}, duration time.Duration) {
	if !m.enabled {
		return
	}

	key := generateCacheKey(query)
	m.optimizer.Set(key, query, result, 5*time.Minute)
	m.strategy.RecordHit(extractPrimaryTable(query))
}

func (m *QueryCacheManager) GetCachedResult(query string) (interface{}, bool) {
	if !m.enabled {
		return nil, false
	}

	key := generateCacheKey(query)
	result, found := m.optimizer.Get(key)
	if found {
		m.strategy.RecordHit(extractPrimaryTable(query))
		return result, true
	}

	m.strategy.RecordMiss(extractPrimaryTable(query))
	return nil, false
}

func (m *QueryCacheManager) InvalidateTable(tableName string) {
	if !m.enabled {
		return
	}

	m.optimizer.InvalidateByTag(tableName)
	log.Printf("[CACHE_MANAGER] Invalidated cache for table: %s", tableName)
}

func (m *QueryCacheManager) InvalidateAll() {
	m.optimizer.Clear()
	log.Println("[CACHE_MANAGER] Cleared all query cache")
}

func (m *QueryCacheManager) GetReport() map[string]interface{} {
	return map[string]interface{}{
		"optimizer":  m.optimizer.GetStats(),
		"prepared":   m.preparedCache.GetStats(),
		"strategy":   m.strategy.GenerateReport(),
		"enabled":    m.enabled,
	}
}

func generateCacheKey(query string) string {
	data := []byte(query)
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

func extractPrimaryTable(query string) string {
	upperQuery := strings.ToUpper(query)

	patterns := []string{"FROM ", "UPDATE ", "INTO ", "TABLE "}
	for _, pattern := range patterns {
		if idx := indexOf(upperQuery, pattern); idx >= 0 {
			start := idx + len(pattern)
			tableName := extractNextWord(query[start:])
			if tableName != "" {
				return tableName
			}
		}
	}

	return "unknown"
}
