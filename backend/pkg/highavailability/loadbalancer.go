package highavailability

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type LoadBalancingStrategy string

const (
	StrategyRoundRobin         LoadBalancingStrategy = "round_robin"
	StrategyWeightedRoundRobin LoadBalancingStrategy = "weighted_round_robin"
	StrategyLeastConnection    LoadBalancingStrategy = "least_connection"
	StrategyIPHash            LoadBalancingStrategy = "ip_hash"
	StrategyRandom            LoadBalancingStrategy = "random"
	StrategyConsistentHash    LoadBalancingStrategy = "consistent_hash"
	StrategyHealthBased       LoadBalancingStrategy = "health_based"
)

type LBBackend struct {
	URL          string
	Weight       int
	Healthy      bool
	ActiveConns  int64
	TotalConns   uint64
	Failures     int64
	Latency      time.Duration
	LastCheck    time.Time
	Metadata     map[string]string
	Priority     int
	Region       string
	Zone         string
	Tags         []string
}

type LBBackendStats struct {
	URL          string            `json:"url"`
	Weight       int               `json:"weight"`
	Healthy      bool              `json:"healthy"`
	ActiveConns  int64             `json:"active_connections"`
	TotalConns   uint64            `json:"total_connections"`
	Failures     int64             `json:"failures"`
	Latency      time.Duration     `json:"latency"`
	LastCheck    time.Time         `json:"last_check"`
	Metadata     map[string]string `json:"metadata"`
	SuccessRate  float64           `json:"success_rate"`
	HealthScore  float64           `json:"health_score"`
}

type LoadBalancer struct {
	backends     map[string]*LBBackend
	strategy     LoadBalancingStrategy
	mu           sync.RWMutex
	currentIdx   uint32
	ipMap        map[string]int
	ipMapMu      sync.RWMutex
	hashRing     []uint32
	hashRingMu   sync.RWMutex
	healthCheck  *LBHealthChecker
	failover     *FailoverManager
	config       *LoadBalancerConfig
}

type LoadBalancerConfig struct {
	MaxFailures       int
	HealthCheckPeriod time.Duration
	HealthCheckTimeout time.Duration
	HealthCheckPath   string
	FailoverEnabled   bool
	RecoveryTimeout   time.Duration
	Quorum            int
	Weights           map[string]int
}

var defaultLBConfig = &LoadBalancerConfig{
	MaxFailures:        3,
	HealthCheckPeriod:  10 * time.Second,
	HealthCheckTimeout: 5 * time.Second,
	HealthCheckPath:    "/health",
	FailoverEnabled:    true,
	RecoveryTimeout:    30 * time.Second,
	Quorum:             1,
}

func NewLoadBalancer(config *LoadBalancerConfig) *LoadBalancer {
	if config == nil {
		config = defaultLBConfig
	}

	return &LoadBalancer{
		backends:    make(map[string]*LBBackend),
		strategy:    StrategyRoundRobin,
		ipMap:       make(map[string]int),
		hashRing:    make([]uint32, 0),
		healthCheck: NewLBHealthChecker(config),
		failover:    NewFailoverManager(config),
		config:      config,
	}
}

func (lb *LoadBalancer) AddBackend(backend *LBBackend) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if backend == nil {
		return fmt.Errorf("backend is nil")
	}

	if backend.URL == "" {
		return fmt.Errorf("backend URL is required")
	}

	if backend.Weight <= 0 {
		backend.Weight = 1
	}

	if backend.Metadata == nil {
		backend.Metadata = make(map[string]string)
	}

	backend.Healthy = true
	backend.LastCheck = time.Now()

	lb.backends[backend.URL] = backend

	lb.rebuildHashRing()

	return nil
}

func (lb *LoadBalancer) RemoveBackend(url string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if _, ok := lb.backends[url]; !ok {
		return fmt.Errorf("backend not found: %s", url)
	}

	delete(lb.backends, url)
	lb.rebuildHashRing()

	return nil
}

func (lb *LoadBalancer) GetBackend(clientIP string) (*LBBackend, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	backends := lb.getHealthyBackends()
	if len(backends) == 0 {
		return nil, fmt.Errorf("no healthy backends available")
	}

	switch lb.strategy {
	case StrategyRoundRobin:
		return lb.roundRobinSelect(backends), nil
	case StrategyWeightedRoundRobin:
		return lb.weightedRoundRobinSelect(backends), nil
	case StrategyLeastConnection:
		return lb.leastConnectionSelect(backends), nil
	case StrategyIPHash:
		return lb.ipHashSelect(clientIP, backends), nil
	case StrategyRandom:
		return lb.randomSelect(backends), nil
	case StrategyConsistentHash:
		return lb.consistentHashSelect(clientIP, backends), nil
	case StrategyHealthBased:
		return lb.healthBasedSelect(backends), nil
	default:
		return lb.roundRobinSelect(backends), nil
	}
}

func (lb *LoadBalancer) getHealthyBackends() []*LBBackend {
	var healthy []*LBBackend
	for _, backend := range lb.backends {
		if backend.Healthy {
			healthy = append(healthy, backend)
		}
	}

	if len(healthy) == 0 {
		healthy = make([]*LBBackend, 0, len(lb.backends))
		for _, backend := range lb.backends {
			healthy = append(healthy, backend)
		}
	}

	return healthy
}

func (lb *LoadBalancer) roundRobinSelect(backends []*LBBackend) *LBBackend {
	idx := atomic.AddUint32(&lb.currentIdx, 1)
	backend := backends[int(idx-1)%len(backends)]
	lb.incrementConnStats(backend)
	return backend
}

func (lb *LoadBalancer) weightedRoundRobinSelect(backends []*LBBackend) *LBBackend {
	type weightedBackend struct {
		backend *LBBackend
		count   int
	}

	var weightedBackends []weightedBackend
	for _, b := range backends {
		for i := 0; i < b.Weight; i++ {
			weightedBackends = append(weightedBackends, weightedBackend{
				backend: b,
				count:   b.Weight,
			})
		}
	}

	if len(weightedBackends) == 0 {
		return backends[0]
	}

	idx := atomic.AddUint32(&lb.currentIdx, 1)
	selected := weightedBackends[int(idx-1)%len(weightedBackends)]
	lb.incrementConnStats(selected.backend)
	return selected.backend
}

func (lb *LoadBalancer) leastConnectionSelect(backends []*LBBackend) *LBBackend {
	var selected *LBBackend
	minConns := int64(math.MaxInt64)

	for _, b := range backends {
		active := atomic.LoadInt64(&b.ActiveConns)
		if active < minConns {
			minConns = active
			selected = b
		}
	}

	if selected == nil {
		selected = backends[0]
	}

	lb.incrementConnStats(selected)
	return selected
}

func (lb *LoadBalancer) ipHashSelect(clientIP string, backends []*LBBackend) *LBBackend {
	lb.ipMapMu.RLock()
	idx, exists := lb.ipMap[clientIP]
	lb.ipMapMu.RUnlock()

	if exists && idx < len(backends) {
		backend := backends[idx]
		lb.incrementConnStats(backend)
		return backend
	}

	hash := 0
	for _, c := range clientIP {
		hash = hash*31 + int(c)
	}

	idx = hash % len(backends)

	lb.ipMapMu.Lock()
	lb.ipMap[clientIP] = idx
	lb.ipMapMu.Unlock()

	backend := backends[idx]
	lb.incrementConnStats(backend)
	return backend
}

func (lb *LoadBalancer) randomSelect(backends []*LBBackend) *LBBackend {
	idx := time.Now().UnixNano() % int64(len(backends))
	if idx < 0 {
		idx = -idx
	}
	backend := backends[idx]
	lb.incrementConnStats(backend)
	return backend
}

func (lb *LoadBalancer) consistentHashSelect(key string, backends []*LBBackend) *LBBackend {
	lb.hashRingMu.RLock()
	defer lb.hashRingMu.RUnlock()

	if len(lb.hashRing) == 0 {
		return backends[0]
	}

	hash := hashString(key)
	idx := binarySearchHash(lb.hashRing, hash)

	backendURL := lb.hashRing[idx]
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	for url, backend := range lb.backends {
		if hashString(url) == backendURL {
			lb.incrementConnStats(backend)
			return backend
		}
	}

	return backends[0]
}

func (lb *LoadBalancer) healthBasedSelect(backends []*LBBackend) *LBBackend {
	type scoredBackend struct {
		backend *LBBackend
		score   float64
	}

	scored := make([]scoredBackend, len(backends))
	for i, b := range backends {
		score := lb.calculateHealthScore(b)
		scored[i] = scoredBackend{
			backend: b,
			score:   score,
		}
	}

	maxScore := scoredBackend{}
	for _, s := range scored {
		if s.score > maxScore.score {
			maxScore = s
		}
	}

	lb.incrementConnStats(maxScore.backend)
	return maxScore.backend
}

func (lb *LoadBalancer) calculateHealthScore(backend *LBBackend) float64 {
	score := 100.0

	failures := atomic.LoadInt64(&backend.Failures)
	if failures > 0 {
		score -= float64(failures) * 10
	}

	latency := backend.Latency
	if latency > 0 {
		if latency > 1000*time.Millisecond {
			score -= 50
		} else if latency > 500*time.Millisecond {
			score -= 25
		} else if latency > 200*time.Millisecond {
			score -= 10
		}
	}

	activeConns := atomic.LoadInt64(&backend.ActiveConns)
	maxConns := 100.0
	connRatio := float64(activeConns) / maxConns
	if connRatio > 0.8 {
		score -= 30
	} else if connRatio > 0.5 {
		score -= 15
	}

	if !backend.Healthy {
		score = 0
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (lb *LoadBalancer) incrementConnStats(backend *LBBackend) {
	atomic.AddUint64(&backend.TotalConns, 1)
	atomic.AddInt64(&backend.ActiveConns, 1)
}

func (lb *LoadBalancer) ReleaseBackend(backend *LBBackend) {
	atomic.AddInt64(&backend.ActiveConns, -1)
}

func (lb *LoadBalancer) RecordSuccess(backend *LBBackend, latency time.Duration) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	atomic.StoreInt64(&backend.Failures, 0)
	backend.Healthy = true
	backend.Latency = latency
	backend.LastCheck = time.Now()
}

func (lb *LoadBalancer) RecordFailure(backend *LBBackend) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	failures := atomic.AddInt64(&backend.Failures, 1)
	backend.LastCheck = time.Now()

	if failures >= int64(lb.config.MaxFailures) {
		backend.Healthy = false
	}
}

func (lb *LoadBalancer) GetStats() []*LBBackendStats {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	stats := make([]*LBBackendStats, 0, len(lb.backends))
	for _, backend := range lb.backends {
		stats = append(stats, &LBBackendStats{
			URL:         backend.URL,
			Weight:      backend.Weight,
			Healthy:     backend.Healthy,
			ActiveConns: atomic.LoadInt64(&backend.ActiveConns),
			TotalConns:  atomic.LoadUint64(&backend.TotalConns),
			Failures:    atomic.LoadInt64(&backend.Failures),
			Latency:     backend.Latency,
			LastCheck:   backend.LastCheck,
			Metadata:    backend.Metadata,
			SuccessRate: lb.calculateSuccessRate(backend),
			HealthScore: lb.calculateHealthScore(backend),
		})
	}

	return stats
}

func (lb *LoadBalancer) calculateSuccessRate(backend *LBBackend) float64 {
	totalConns := atomic.LoadUint64(&backend.TotalConns)
	if totalConns == 0 {
		return 100.0
	}

	failures := atomic.LoadInt64(&backend.Failures)
	totalFailures := failures
	successes := totalConns - uint64(totalFailures)

	return float64(successes) / float64(totalConns) * 100
}

func (lb *LoadBalancer) SetStrategy(strategy LoadBalancingStrategy) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.strategy = strategy
}

func (lb *LoadBalancer) GetStrategy() LoadBalancingStrategy {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.strategy
}

func (lb *LoadBalancer) UpdateBackendWeight(url string, weight int) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	backend, ok := lb.backends[url]
	if !ok {
		return fmt.Errorf("backend not found: %s", url)
	}

	backend.Weight = weight
	lb.rebuildHashRing()

	return nil
}

func (lb *LoadBalancer) SetBackendHealthy(url string, healthy bool) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	backend, ok := lb.backends[url]
	if !ok {
		return
	}

	backend.Healthy = healthy
	if healthy {
		atomic.StoreInt64(&backend.Failures, 0)
	}
}

func (lb *LoadBalancer) GetBackendCount() int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return len(lb.backends)
}

func (lb *LoadBalancer) GetHealthyBackendCount() int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	count := 0
	for _, backend := range lb.backends {
		if backend.Healthy {
			count++
		}
	}
	return count
}

func (lb *LoadBalancer) rebuildHashRing() {
	lb.hashRing = make([]uint32, 0)

	for url, backend := range lb.backends {
		hash := hashString(url)
		lb.hashRing = append(lb.hashRing, hash)

		for i := 0; i < backend.Weight*10; i++ {
			vHash := hashString(fmt.Sprintf("%s-%d", url, i))
			lb.hashRing = append(lb.hashRing, vHash)
		}
	}

	sortUint32Slice(lb.hashRing)
}

func (lb *LoadBalancer) StartHealthCheck(ctx context.Context) {
	lb.mu.RLock()
	backends := make([]*LBBackend, 0, len(lb.backends))
	for _, b := range lb.backends {
		backends = append(backends, b)
	}
	lb.mu.RUnlock()

	lb.healthCheck.Start(ctx, backends)
}

func (lb *LoadBalancer) StopHealthCheck() {
	lb.healthCheck.Stop()
}

type LBHealthChecker struct {
	config     *LoadBalancerConfig
	stopCh     chan struct{}
	resultsCh  chan *HealthCheckResult
	mu         sync.RWMutex
	results    map[string]*HealthCheckResult
}

type HealthCheckResult struct {
	URL       string
	Healthy   bool
	Latency   time.Duration
	Error     error
	Timestamp time.Time
}

func NewLBHealthChecker(config *LoadBalancerConfig) *LBHealthChecker {
	if config == nil {
		config = defaultLBConfig
	}

	return &LBHealthChecker{
		config:    config,
		stopCh:    make(chan struct{}),
		resultsCh: make(chan *HealthCheckResult, 100),
		results:   make(map[string]*HealthCheckResult),
	}
}

func (hc *LBHealthChecker) Start(ctx context.Context, backends []*LBBackend) {
	ticker := time.NewTicker(hc.config.HealthCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.checkAll(backends)
		}
	}
}

func (hc *LBHealthChecker) Stop() {
	close(hc.stopCh)
}

func (hc *LBHealthChecker) checkAll(backends []*LBBackend) {
	var wg sync.WaitGroup
	for _, backend := range backends {
		wg.Add(1)
		go func(b *LBBackend) {
			defer wg.Done()
			hc.checkBackend(b)
		}(backend)
	}
	wg.Wait()
}

func (hc *LBHealthChecker) checkBackend(backend *LBBackend) {
	start := time.Now()

	url := backend.URL + hc.config.HealthCheckPath

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		hc.recordResult(backend.URL, false, 0, err)
		return
	}

	client := &http.Client{
		Timeout: hc.config.HealthCheckTimeout,
	}

	resp, err := client.Do(req)
	latency := time.Since(start)

	if err != nil {
		hc.recordResult(backend.URL, false, latency, err)
		return
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode >= 200 && resp.StatusCode < 400
	hc.recordResult(backend.URL, healthy, latency, nil)
}

func (hc *LBHealthChecker) recordResult(url string, healthy bool, latency time.Duration, err error) {
	result := &HealthCheckResult{
		URL:       url,
		Healthy:   healthy,
		Latency:   latency,
		Error:     err,
		Timestamp: time.Now(),
	}

	hc.mu.Lock()
	hc.results[url] = result
	hc.mu.Unlock()

	select {
	case hc.resultsCh <- result:
	default:
	}
}

func (hc *LBHealthChecker) GetResult(url string) (*HealthCheckResult, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result, ok := hc.results[url]
	return result, ok
}

func (hc *LBHealthChecker) GetAllResults() map[string]*HealthCheckResult {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	results := make(map[string]*HealthCheckResult, len(hc.results))
	for k, v := range hc.results {
		results[k] = v
	}
	return results
}

type FailoverManager struct {
	config      *LoadBalancerConfig
	failedBackends map[string]*FailedBackend
	mu          sync.RWMutex
}

type FailedBackend struct {
	URL         string
	FailedAt    time.Time
	FailCount   int
	NextRetry   time.Time
}

func NewFailoverManager(config *LoadBalancerConfig) *FailoverManager {
	if config == nil {
		config = defaultLBConfig
	}

	return &FailoverManager{
		config:         config,
		failedBackends: make(map[string]*FailedBackend),
	}
}

func (fm *FailoverManager) MarkFailed(url string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	failed, exists := fm.failedBackends[url]
	if !exists {
		failed = &FailedBackend{
			URL:       url,
			FailedAt:  time.Now(),
			FailCount: 0,
		}
		fm.failedBackends[url] = failed
	}

	failed.FailCount++
	failed.NextRetry = time.Now().Add(fm.config.RecoveryTimeout)
}

func (fm *FailoverManager) MarkRecovered(url string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	delete(fm.failedBackends, url)
}

func (fm *FailoverManager) IsFailed(url string) bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	failed, exists := fm.failedBackends[url]
	if !exists {
		return false
	}

	if time.Now().After(failed.NextRetry) {
		return false
	}

	return true
}

func (fm *FailoverManager) GetFailedBackends() []*FailedBackend {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	failed := make([]*FailedBackend, 0, len(fm.failedBackends))
	for _, f := range fm.failedBackends {
		failed = append(failed, f)
	}
	return failed
}

func (fm *FailoverManager) ShouldFailover(healthyCount int) bool {
	totalCount := len(fm.failedBackends) + healthyCount
	if totalCount == 0 {
		return false
	}

	failedRatio := float64(len(fm.failedBackends)) / float64(totalCount)
	return failedRatio >= 0.5
}

type ProxyHandler struct {
	loadBalancer *LoadBalancer
	timeout      time.Duration
	maxRetries   int
}

func NewProxyHandler(lb *LoadBalancer, timeout time.Duration, maxRetries int) *ProxyHandler {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	if maxRetries == 0 {
		maxRetries = 3
	}

	return &ProxyHandler{
		loadBalancer: lb,
		timeout:      timeout,
		maxRetries:   maxRetries,
	}
}

func (ph *ProxyHandler) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		backend, err := ph.loadBalancer.GetBackend(clientIP)
		if err != nil {
			response.InternalServerError(c, "no backend available")
			c.Abort()
			return
		}

		start := time.Now()

		proxyURL := backend.URL + c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			proxyURL += "?" + c.Request.URL.RawQuery
		}

		req, err := http.NewRequest(c.Request.Method, proxyURL, c.Request.Body)
		if err != nil {
			ph.loadBalancer.RecordFailure(backend)
			response.InternalServerError(c, "failed to create proxy request")
			c.Abort()
			return
		}

		for key, values := range c.Request.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		req.Header.Set("X-Forwarded-For", clientIP)
		req.Header.Set("X-Real-IP", clientIP)
		req.Header.Set("X-Original-URL", c.Request.URL.String())

		client := &http.Client{
			Timeout: ph.timeout,
		}

		resp, err := client.Do(req)
		latency := time.Since(start)

		if err != nil {
			ph.loadBalancer.RecordFailure(backend)
			response.InternalServerError(c, "failed to proxy request")
			c.Abort()
			return
		}
		defer resp.Body.Close()

		ph.loadBalancer.RecordSuccess(backend, latency)

		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		c.Header("X-Proxy-Backend", backend.URL)
		c.Header("X-Proxy-Latency", latency.String())

		c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)

		ph.loadBalancer.ReleaseBackend(backend)
	}
}

func hashString(s string) uint32 {
	hash := uint32(5381)
	for _, c := range s {
		hash = ((hash << 5) + hash) + uint32(c)
	}
	return hash
}

func binarySearchHash(arr []uint32, target uint32) int {
	low := 0
	high := len(arr) - 1

	if high < 0 {
		return 0
	}

	for low < high {
		mid := (low + high + 1) / 2
		if arr[mid] <= target {
			low = mid
		} else {
			high = mid - 1
		}
	}

	if arr[low] >= target {
		return low
	}

	return 0
}

func sortUint32Slice(arr []uint32) {
	for i := 0; i < len(arr); i++ {
		for j := i + 1; j < len(arr); j++ {
			if arr[i] > arr[j] {
				arr[i], arr[j] = arr[j], arr[i]
			}
		}
	}
}
