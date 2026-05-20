package api_performance

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type MiddlewareChainOptimizer struct {
	config      *ChainOptimizerConfig
	chainStats  *ChainStats
	perfTracker *PerformanceTracker
	executors   map[string]*MiddlewareExecutor
	mu          sync.RWMutex
}

type ChainOptimizerConfig struct {
	EnableBatching     bool
	EnableCaching      bool
	BatchWindow        time.Duration
	CacheEnabled       bool
	CacheTTL           time.Duration
	MaxConcurrency     int
	EnableEarlyExit    bool
	EarlyExitThreshold time.Duration
	MonitorEnabled     bool
}

var DefaultChainOptimizerConfig = &ChainOptimizerConfig{
	EnableBatching:     true,
	EnableCaching:      true,
	BatchWindow:        5 * time.Millisecond,
	CacheEnabled:       true,
	CacheTTL:           30 * time.Second,
	MaxConcurrency:     1000,
	EnableEarlyExit:    true,
	EarlyExitThreshold: 10 * time.Millisecond,
	MonitorEnabled:     true,
}

type ChainStats struct {
	TotalRequests      atomic.Int64
	CacheHits          atomic.Int64
	CacheMisses        atomic.Int64
	BatchHits          atomic.Int64
	EarlyExits         atomic.Int64
	AvgLatency         atomic.Int64
	P95Latency         atomic.Int64
	P99Latency         atomic.Int64
	TotalMiddlewareTime atomic.Int64
	Errors             atomic.Int64
}

type PerformanceTracker struct {
	mu        sync.RWMutex
	records   []MiddlewareRecord
	maxRecords int
}

type MiddlewareRecord struct {
	Timestamp    time.Time
	Name        string
	Duration    time.Duration
	Order       int
	RequestID   string
}

type MiddlewareExecutor struct {
	name       string
	handler    gin.HandlerFunc
	order      int
	cache      *MiddlewareCache
	stats      *ExecutorStats
}

type MiddlewareCache struct {
	cache   *sync.Map
	maxSize int
	mu      sync.RWMutex
	hits    atomic.Int64
	misses  atomic.Int64
}

type ExecutorStats struct {
	Executions    atomic.Int64
	CacheHits     atomic.Int64
	AvgDuration   atomic.Int64
	MaxDuration   atomic.Int64
	Errors        atomic.Int64
}

type MiddlewareChain struct {
	executors []*MiddlewareExecutor
	optimizer *MiddlewareChainOptimizer
	mu        sync.RWMutex
}

type RequestContext struct {
	ID        string
	StartTime time.Time
	Data      map[string]interface{}
	MiddlewareTimes map[string]time.Duration
	mu        sync.RWMutex
}

type MiddlewareChainConfig struct {
	EnableCache   bool
	EnableMetrics bool
	EnableTrace   bool
}

func NewMiddlewareChainOptimizer(config *ChainOptimizerConfig) *MiddlewareChainOptimizer {
	if config == nil {
		config = DefaultChainOptimizerConfig
	}

	return &MiddlewareChainOptimizer{
		config:      config,
		chainStats:  &ChainStats{},
		perfTracker: NewPerformanceTracker(10000),
		executors:   make(map[string]*MiddlewareExecutor),
	}
}

func NewPerformanceTracker(maxRecords int) *PerformanceTracker {
	return &PerformanceTracker{
		records:   make([]MiddlewareRecord, 0, maxRecords),
		maxRecords: maxRecords,
	}
}

func (pt *PerformanceTracker) Record(name string, duration time.Duration, order int, requestID string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.records = append(pt.records, MiddlewareRecord{
		Timestamp:  time.Now(),
		Name:       name,
		Duration:   duration,
		Order:      order,
		RequestID:  requestID,
	})

	if len(pt.records) > pt.maxRecords {
		pt.records = pt.records[1:]
	}
}

func (pt *PerformanceTracker) GetStats() map[string]interface{} {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if len(pt.records) == 0 {
		return map[string]interface{}{
			"total_records": 0,
		}
	}

	middlewareStats := make(map[string]*MiddlewareStats)
	for _, record := range pt.records {
		if _, exists := middlewareStats[record.Name]; !exists {
			middlewareStats[record.Name] = &MiddlewareStats{
				Name: record.Name,
			}
		}

		stats := middlewareStats[record.Name]
		stats.Count++
		stats.TotalDuration += record.Duration
		if record.Duration > stats.MaxDuration {
			stats.MaxDuration = record.Duration
		}
		if stats.MinDuration == 0 || record.Duration < stats.MinDuration {
			stats.MinDuration = record.Duration
		}
	}

	result := map[string]interface{}{
		"total_records": len(pt.records),
	}

	for name, stats := range middlewareStats {
		stats.AvgDuration = stats.TotalDuration / time.Duration(stats.Count)
		result[name] = stats
	}

	return result
}

type MiddlewareStats struct {
	Name          string
	Count         int
	TotalDuration time.Duration
	AvgDuration   time.Duration
	MaxDuration   time.Duration
	MinDuration   time.Duration
}

func (o *MiddlewareChainOptimizer) AddMiddleware(name string, handler gin.HandlerFunc, order int) {
	o.mu.Lock()
	defer o.mu.Unlock()

	executor := &MiddlewareExecutor{
		name:    name,
		handler: handler,
		order:   order,
		cache:   NewMiddlewareCache(1000),
		stats:   &ExecutorStats{},
	}

	o.executors[name] = executor
}

func (o *MiddlewareChainOptimizer) CreateChain() gin.HandlerFunc {
	o.mu.RLock()
	executors := make([]*MiddlewareExecutor, 0, len(o.executors))
	for _, exec := range o.executors {
		executors = append(executors, exec)
	}
	o.mu.RUnlock()

	sortExecutors(executors)

	return func(c *gin.Context) {
		start := time.Now()
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("X-Request-ID", requestID)

		ctx := &RequestContext{
			ID:              requestID,
			StartTime:       start,
			Data:            make(map[string]interface{}),
			MiddlewareTimes: make(map[string]time.Duration),
		}

		c.Set("RequestContext", ctx)

		o.chainStats.TotalRequests.Add(1)

		if o.config.EnableEarlyExit {
			if o.shouldEarlyExit(c) {
				o.chainStats.EarlyExits.Add(1)
				c.Next()
				return
			}
		}

		o.executeChain(c, executors, ctx)

		latency := time.Since(start)
		o.recordLatency(latency)

		c.Header("X-Response-Time", latency.String())
		c.Header("X-Request-ID", requestID)
	}
}

func (o *MiddlewareChainOptimizer) shouldEarlyExit(c *gin.Context) bool {
	path := c.Request.URL.Path

	if path == "/health" || path == "/health/v2" || path == "/health/ready" || path == "/health/live" {
		return true
	}

	return false
}

func (o *MiddlewareChainOptimizer) executeChain(c *gin.Context, executors []*MiddlewareExecutor, ctx *RequestContext) {
	for _, exec := range executors {
		start := time.Now()

		exec.handler(c)

		duration := time.Since(start)
		ctx.MiddlewareTimes[exec.name] = duration
		exec.stats.Executions.Add(1)

		if o.config.MonitorEnabled {
			o.perfTracker.Record(exec.name, duration, exec.order, ctx.ID)
		}

		durationNs := int64(duration)
		if durationNs > exec.stats.MaxDuration.Load() {
			exec.stats.MaxDuration.Store(durationNs)
		}

		old := exec.stats.AvgDuration.Load()
		count := exec.stats.Executions.Load()
		if count > 0 {
			newAvg := (old*(count-1) + durationNs) / count
			exec.stats.AvgDuration.Store(newAvg)
		}
	}
}

func (o *MiddlewareChainOptimizer) recordLatency(latency time.Duration) {
	o.chainStats.AvgLatency.Store(int64(latency))
}

func sortExecutors(executors []*MiddlewareExecutor) {
	for i := 0; i < len(executors); i++ {
		for j := i + 1; j < len(executors); j++ {
			if executors[i].order > executors[j].order {
				executors[i], executors[j] = executors[j], executors[i]
			}
		}
	}
}

func NewMiddlewareCache(maxSize int) *MiddlewareCache {
	return &MiddlewareCache{
		cache:   &sync.Map{},
		maxSize: maxSize,
	}
}

func (mc *MiddlewareCache) Get(key string) (gin.HandlerFunc, bool) {
	val, ok := mc.cache.Load(key)
	if !ok {
		mc.misses.Add(1)
		return nil, false
	}

	mc.hits.Add(1)
	return val.(gin.HandlerFunc), true
}

func (mc *MiddlewareCache) Set(key string, handler gin.HandlerFunc) {
	if mc.getSize() >= mc.maxSize {
		mc.evictOldest()
	}

	mc.cache.Store(key, handler)
}

func (mc *MiddlewareCache) Delete(key string) {
	mc.cache.Delete(key)
}

func (mc *MiddlewareCache) Clear() {
	mc.cache = &sync.Map{}
}

func (mc *MiddlewareCache) getSize() int {
	size := 0
	mc.cache.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}

func (mc *MiddlewareCache) evictOldest() {
	var oldestKey string

	mc.cache.Range(func(key, value interface{}) bool {
		if oldestKey == "" {
			oldestKey = key.(string)
		}
		return true
	})

	if oldestKey != "" {
		mc.cache.Delete(oldestKey)
	}
}

func (mc *MiddlewareCache) GetStats() map[string]interface{} {
	hits := mc.hits.Load()
	misses := mc.misses.Load()
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"hits":     hits,
		"misses":   misses,
		"hit_rate": hitRate,
		"size":     mc.getSize(),
		"max_size": mc.maxSize,
	}
}

func (o *MiddlewareChainOptimizer) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_requests":      o.chainStats.TotalRequests.Load(),
		"cache_hits":          o.chainStats.CacheHits.Load(),
		"cache_misses":        o.chainStats.CacheMisses.Load(),
		"batch_hits":          o.chainStats.BatchHits.Load(),
		"early_exits":         o.chainStats.EarlyExits.Load(),
		"avg_latency":         time.Duration(o.chainStats.AvgLatency.Load()).String(),
		"p95_latency":         time.Duration(o.chainStats.P95Latency.Load()).String(),
		"p99_latency":         time.Duration(o.chainStats.P99Latency.Load()).String(),
		"total_middleware_time": time.Duration(o.chainStats.TotalMiddlewareTime.Load()).String(),
		"errors":              o.chainStats.Errors.Load(),
	}

	for k, v := range o.perfTracker.GetStats() {
		stats["perf_"+k] = v
	}

	o.mu.RLock()
	defer o.mu.RUnlock()

	executorStats := make(map[string]interface{})
	for name, exec := range o.executors {
		executorStats[name] = map[string]interface{}{
			"executions":   exec.stats.Executions.Load(),
			"cache_hits":   exec.stats.CacheHits.Load(),
			"avg_duration": time.Duration(exec.stats.AvgDuration.Load()).String(),
			"max_duration": time.Duration(exec.stats.MaxDuration.Load()).String(),
			"errors":       exec.stats.Errors.Load(),
		}
	}
	stats["executors"] = executorStats

	return stats
}

func (o *MiddlewareChainOptimizer) ClearCaches() {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, exec := range o.executors {
		exec.cache.Clear()
	}
}

type ResponseCache struct {
	cache    *sync.Map
	maxSize  int
	mu       sync.RWMutex
	hits     atomic.Int64
	misses   atomic.Int64
	evictions atomic.Int64
}

type CacheEntry struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	ExpiresAt  time.Time
}

func NewResponseCache(maxSize int) *ResponseCache {
	return &ResponseCache{
		cache:   &sync.Map{},
		maxSize: maxSize,
	}
}

func (rc *ResponseCache) Get(key string) (*CacheEntry, bool) {
	val, ok := rc.cache.Load(key)
	if !ok {
		rc.misses.Add(1)
		return nil, false
	}

	entry := val.(*CacheEntry)
	if time.Now().After(entry.ExpiresAt) {
		rc.cache.Delete(key)
		rc.evictions.Add(1)
		rc.misses.Add(1)
		return nil, false
	}

	rc.hits.Add(1)
	return entry, true
}

func (rc *ResponseCache) Set(key string, entry *CacheEntry) {
	if rc.getSize() >= rc.maxSize {
		rc.evictOldest()
	}

	rc.cache.Store(key, entry)
}

func (rc *ResponseCache) Delete(key string) {
	rc.cache.Delete(key)
}

func (rc *ResponseCache) Clear() {
	rc.cache = &sync.Map{}
}

func (rc *ResponseCache) getSize() int {
	size := 0
	rc.cache.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}

func (rc *ResponseCache) evictOldest() {
	var oldestKey string

	rc.cache.Range(func(key, value interface{}) bool {
		if oldestKey == "" {
			oldestKey = key.(string)
		}
		return true
	})

	if oldestKey != "" {
		rc.cache.Delete(oldestKey)
		rc.evictions.Add(1)
	}
}

func (rc *ResponseCache) GetStats() map[string]interface{} {
	hits := rc.hits.Load()
	misses := rc.misses.Load()
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"hits":      hits,
		"misses":    misses,
		"hit_rate":  hitRate,
		"evictions": rc.evictions.Load(),
		"size":      rc.getSize(),
		"max_size":  rc.maxSize,
	}
}

func (o *MiddlewareChainOptimizer) CreateResponseCacheMiddleware(ttl time.Duration) gin.HandlerFunc {
	cache := NewResponseCache(10000)

	return func(c *gin.Context) {
		if c.Request.Method != "GET" {
			c.Next()
			return
		}

		key := fmt.Sprintf("%s:%s", c.Request.URL.Path, c.Request.URL.RawQuery)

		if entry, ok := cache.Get(key); ok {
			for k, v := range entry.Headers {
				c.Header(k, v)
			}
			c.Header("X-Cache", "HIT")
			c.Data(entry.StatusCode, "application/json", entry.Body)
			c.Abort()
			return
		}

		c.Header("X-Cache", "MISS")

		c.Next()

		if c.Writer.Status() == 200 {
			entry := &CacheEntry{
				StatusCode: c.Writer.Status(),
				Headers:    make(map[string]string),
				Body:       []byte{},
				ExpiresAt:  time.Now().Add(ttl),
			}

			for k, vals := range c.Writer.Header() {
				if len(vals) > 0 {
					entry.Headers[k] = vals[0]
				}
			}

			cache.Set(key, entry)
		}
	}
}

type AdaptiveMiddleware struct {
	enabled    bool
	strategies map[string]*MiddlewareStrategy
	mu         sync.RWMutex
	current    *MiddlewareStrategy
}

type MiddlewareStrategy struct {
	Name        string
	Priority    int
	Middleware  gin.HandlerFunc
	Condition   func(*gin.Context) bool
	Weight      float64
}

func NewAdaptiveMiddleware() *AdaptiveMiddleware {
	return &AdaptiveMiddleware{
		enabled:    true,
		strategies: make(map[string]*MiddlewareStrategy),
		current:    nil,
	}
}

func (am *AdaptiveMiddleware) AddStrategy(strategy *MiddlewareStrategy) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.strategies[strategy.Name] = strategy

	if am.current == nil || strategy.Priority > am.current.Priority {
		am.current = strategy
	}
}

func (am *AdaptiveMiddleware) SelectStrategy(c *gin.Context) *MiddlewareStrategy {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if am.current == nil {
		return nil
	}

	for _, strategy := range am.strategies {
		if strategy.Condition != nil && strategy.Condition(c) {
			return strategy
		}
	}

	return am.current
}

func (am *AdaptiveMiddleware) CreateHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !am.enabled {
			c.Next()
			return
		}

		strategy := am.SelectStrategy(c)
		if strategy != nil && strategy.Middleware != nil {
			strategy.Middleware(c)
		} else {
			c.Next()
		}
	}
}

func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Nanosecond()%10000)
}
