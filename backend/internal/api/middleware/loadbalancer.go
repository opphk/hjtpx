package middleware

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type LoadBalancerStrategy string

const (
	StrategyRoundRobin   LoadBalancerStrategy = "round_robin"
	StrategyWeighted    LoadBalancerStrategy = "weighted"
	StrategyLeastConn   LoadBalancerStrategy = "least_conn"
	StrategyIPHash      LoadBalancerStrategy = "ip_hash"
	StrategyRandom      LoadBalancerStrategy = "random"
	StrategyWeightedRR  LoadBalancerStrategy = "weighted_round_robin"
)

type LoadBalancer struct {
	backends    []*Backend
	strategy    LoadBalancerStrategy
	mu          sync.RWMutex
	currentIdx  uint32
	ipMap       map[string]int
	ipMapMu     sync.RWMutex
	healthCheck *HealthChecker
}

type Backend struct {
	URL        string
	Weight     int
	Healthy    bool
	ActiveConn int64
	TotalConn  uint64
	Failures   int64
	Latency    time.Duration
	LastCheck  time.Time
	Metadata   map[string]interface{}
}

type BackendStats struct {
	URL         string                 `json:"url"`
	Weight      int                    `json:"weight"`
	Healthy     bool                   `json:"healthy"`
	ActiveConn  int64                  `json:"active_connections"`
	TotalConn   uint64                 `json:"total_connections"`
	Failures    int                    `json:"failures"`
	Latency     string                 `json:"latency"`
	LastCheck   time.Time              `json:"last_check"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func NewLoadBalancer(strategy LoadBalancerStrategy) *LoadBalancer {
	return &LoadBalancer{
		backends:   make([]*Backend, 0),
		strategy:   strategy,
		ipMap:      make(map[string]int),
		healthCheck: NewHealthChecker(10*time.Second, 5*time.Second),
	}
}

func (lb *LoadBalancer) AddBackend(url string, weight int) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	backend := &Backend{
		URL:      url,
		Weight:   weight,
		Healthy:  true,
		Metadata: make(map[string]interface{}),
	}

	lb.backends = append(lb.backends, backend)
	lb.healthCheck.AddBackend(url, weight)
}

func (lb *LoadBalancer) RemoveBackend(url string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, backend := range lb.backends {
		if backend.URL == url {
			lb.backends = append(lb.backends[:i], lb.backends[i+1:]...)
			return
		}
	}
}

func (lb *LoadBalancer) GetBackend(clientIP string) (*Backend, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	healthyBackends := lb.getHealthyBackends()
	if len(healthyBackends) == 0 {
		return nil, fmt.Errorf("no healthy backends available")
	}

	switch lb.strategy {
	case StrategyRoundRobin:
		return lb.roundRobin(healthyBackends)
	case StrategyWeighted, StrategyWeightedRR:
		return lb.weightedRoundRobin(healthyBackends)
	case StrategyLeastConn:
		return lb.leastConn(healthyBackends)
	case StrategyIPHash:
		return lb.ipHash(clientIP, healthyBackends)
	case StrategyRandom:
		return lb.random(healthyBackends)
	default:
		return lb.roundRobin(healthyBackends)
	}
}

func (lb *LoadBalancer) getHealthyBackends() []*Backend {
	var healthy []*Backend
	for _, b := range lb.backends {
		if b.Healthy {
			healthy = append(healthy, b)
		}
	}

	if len(healthy) == 0 {
		return lb.backends
	}

	return healthy
}

func (lb *LoadBalancer) roundRobin(backends []*Backend) (*Backend, error) {
	idx := atomic.AddUint32(&lb.currentIdx, 1)
	backend := backends[int(idx-1)%len(backends)]
	atomic.AddUint64(&backend.TotalConn, 1)
	atomic.AddInt64(&backend.ActiveConn, 1)
	return backend, nil
}

func (lb *LoadBalancer) weightedRoundRobin(backends []*Backend) (*Backend, error) {
	type weightedBackend struct {
		backend *Backend
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
		return backends[0], nil
	}

	idx := atomic.AddUint32(&lb.currentIdx, 1)
	selected := weightedBackends[int(idx-1)%len(weightedBackends)]
	atomic.AddUint64(&selected.backend.TotalConn, 1)
	atomic.AddInt64(&selected.backend.ActiveConn, 1)
	return selected.backend, nil
}

func (lb *LoadBalancer) leastConn(backends []*Backend) (*Backend, error) {
	var minConn *Backend
	minActive := int64(1 << 63 - 1)

	for _, b := range backends {
		active := atomic.LoadInt64(&b.ActiveConn)
		if active < minActive {
			minActive = active
			minConn = b
		}
	}

	if minConn == nil {
		return backends[0], nil
	}

	atomic.AddUint64(&minConn.TotalConn, 1)
	atomic.AddInt64(&minConn.ActiveConn, 1)
	return minConn, nil
}

func (lb *LoadBalancer) ipHash(clientIP string, backends []*Backend) (*Backend, error) {
	lb.ipMapMu.RLock()
	idx, exists := lb.ipMap[clientIP]
	lb.ipMapMu.RUnlock()

	if exists && idx < len(backends) {
		backend := backends[idx]
		atomic.AddUint64(&backend.TotalConn, 1)
		atomic.AddInt64(&backend.ActiveConn, 1)
		return backend, nil
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
	atomic.AddUint64(&backend.TotalConn, 1)
	atomic.AddInt64(&backend.ActiveConn, 1)
	return backend, nil
}

func (lb *LoadBalancer) random(backends []*Backend) (*Backend, error) {
	idx := rand.Intn(len(backends))
	backend := backends[idx]
	atomic.AddUint64(&backend.TotalConn, 1)
	atomic.AddInt64(&backend.ActiveConn, 1)
	return backend, nil
}

func (lb *LoadBalancer) ReleaseBackend(backend *Backend) {
	atomic.AddInt64(&backend.ActiveConn, -1)
}

func (lb *LoadBalancer) RecordFailure(backend *Backend) {
	atomic.AddInt64(&backend.Failures, 1)
	lb.mu.Lock()
	if backend.Failures >= 3 {
		backend.Healthy = false
	}
	lb.mu.Unlock()
}

func (lb *LoadBalancer) RecordSuccess(backend *Backend, latency time.Duration) {
	lb.mu.Lock()
	backend.Failures = 0
	backend.Healthy = true
	backend.Latency = latency
	backend.LastCheck = time.Now()
	lb.mu.Unlock()
}

func (lb *LoadBalancer) GetStats() []*BackendStats {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	stats := make([]*BackendStats, len(lb.backends))
	for i, b := range lb.backends {
		stats[i] = &BackendStats{
			URL:        b.URL,
			Weight:     b.Weight,
			Healthy:    b.Healthy,
			ActiveConn: atomic.LoadInt64(&b.ActiveConn),
			TotalConn:  atomic.LoadUint64(&b.TotalConn),
			Failures:   int(atomic.LoadInt64(&b.Failures)),
			Latency:    b.Latency.String(),
			LastCheck:  b.LastCheck,
			Metadata:   b.Metadata,
		}
	}

	return stats
}

func (lb *LoadBalancer) StartHealthCheck(ctx context.Context) {
	lb.healthCheck.Start(ctx)
}

func (lb *LoadBalancer) StopHealthCheck() {
	lb.healthCheck.Stop()
}

type ProxyMiddleware struct {
	loadBalancer *LoadBalancer
	timeout      time.Duration
	maxRetries   int
}

func NewProxyMiddleware(lb *LoadBalancer) *ProxyMiddleware {
	return &ProxyMiddleware{
		loadBalancer: lb,
		timeout:      30 * time.Second,
		maxRetries:   3,
	}
}

func (pm *ProxyMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		backend, err := pm.loadBalancer.GetBackend(clientIP)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "no backend available",
			})
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
			pm.loadBalancer.RecordFailure(backend)
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "failed to create proxy request",
			})
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
			Timeout: pm.timeout,
		}

		resp, err := client.Do(req)
		latency := time.Since(start)

		if err != nil {
			pm.loadBalancer.RecordFailure(backend)
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "failed to proxy request",
			})
			c.Abort()
			return
		}
		defer resp.Body.Close()

		pm.loadBalancer.RecordSuccess(backend, latency)

		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		c.Header("X-Proxy-Backend", backend.URL)
	c.Header("X-Proxy-Latency", latency.String())

	c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)

	pm.loadBalancer.ReleaseBackend(backend)
	}
}

type ServerPool struct {
	lb    *LoadBalancer
	srvs  []*HTTPServer
	mu    sync.RWMutex
}

type HTTPServer struct {
	URL     string
	healthy bool
	weight  int
}

func NewServerPool(strategy LoadBalancerStrategy) *ServerPool {
	return &ServerPool{
		lb: NewLoadBalancer(strategy),
	}
}

func (sp *ServerPool) AddServer(url string, weight int) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.lb.AddBackend(url, weight)
	sp.srvs = append(sp.srvs, &HTTPServer{
		URL:    url,
		weight: weight,
	})
}

func (sp *ServerPool) RemoveServer(url string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.lb.RemoveBackend(url)

	for i, srv := range sp.srvs {
		if srv.URL == url {
			sp.srvs = append(sp.srvs[:i], sp.srvs[i+1:]...)
			return
		}
	}
}

func (sp *ServerPool) GetServer(clientIP string) (*Backend, error) {
	return sp.lb.GetBackend(clientIP)
}

func (sp *ServerPool) GetStats() []*BackendStats {
	return sp.lb.GetStats()
}

func (sp *ServerPool) StartHealthCheck(ctx context.Context) {
	sp.lb.StartHealthCheck(ctx)
}

func (sp *ServerPool) StopHealthCheck() {
	sp.lb.StopHealthCheck()
}

func (sp *ServerPool) GetHealthyCount() int {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	count := 0
	for _, srv := range sp.srvs {
		if srv.healthy {
			count++
		}
	}
	return count
}

func (sp *ServerPool) GetTotalCount() int {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return len(sp.srvs)
}

type CircuitBreakerLB struct {
	mu            sync.RWMutex
	backends      map[string]*CircuitState
	threshold     int
	resetTimeout  time.Duration
	currentState  string
}

type CircuitState struct {
	Failures    int
	LastFailure time.Time
	State       string
	Backend     *Backend
}

func NewCircuitBreakerLB(threshold int, resetTimeout time.Duration) *CircuitBreakerLB {
	return &CircuitBreakerLB{
		backends:     make(map[string]*CircuitState),
		threshold:    threshold,
		resetTimeout: resetTimeout,
		currentState: "closed",
	}
}

func (cb *CircuitBreakerLB) AddBackend(backend *Backend) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.backends[backend.URL] = &CircuitState{
		Backend: backend,
		State:   "closed",
	}
}

func (cb *CircuitBreakerLB) IsAvailable(url string) bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	state, ok := cb.backends[url]
	if !ok {
		return true
	}

	switch state.State {
	case "closed":
		return true
	case "half-open":
		return true
	case "open":
		if time.Since(state.LastFailure) > cb.resetTimeout {
			return true
		}
		return false
	}

	return true
}

func (cb *CircuitBreakerLB) RecordSuccess(url string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, ok := cb.backends[url]
	if !ok {
		return
	}

	state.Failures = 0
	state.State = "closed"
}

func (cb *CircuitBreakerLB) RecordFailure(url string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, ok := cb.backends[url]
	if !ok {
		return
	}

	state.Failures++
	state.LastFailure = time.Now()

	if state.Failures >= cb.threshold {
		state.State = "open"
	}
}

func (cb *CircuitBreakerLB) GetState(url string) string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	state, ok := cb.backends[url]
	if !ok {
		return "unknown"
	}

	return state.State
}

func (cb *CircuitBreakerLB) GetAllStates() map[string]string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	states := make(map[string]string)
	for url, state := range cb.backends {
		states[url] = state.State
	}
	return states
}

type ConsistentHashLB struct {
	mu         sync.RWMutex
	ring       []uint32
	nodes      map[uint32]*Backend
	virtualNodes int
	sorted      bool
}

func NewConsistentHashLB(virtualNodes int) *ConsistentHashLB {
	return &ConsistentHashLB{
		nodes:        make(map[uint32]*Backend),
		virtualNodes: virtualNodes,
		sorted:      true,
	}
}

func (ch *ConsistentHashLB) AddNode(backend *Backend) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	hash := hashKey(backend.URL)
	ch.nodes[hash] = backend
	ch.ring = append(ch.ring, hash)

	for i := 0; i < ch.virtualNodes; i++ {
		vHash := hashKey(fmt.Sprintf("%s#%d", backend.URL, i))
		ch.nodes[vHash] = backend
		ch.ring = append(ch.ring, vHash)
	}

	ch.sorted = false
	ch.sortRing()
}

func (ch *ConsistentHashLB) RemoveNode(url string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	hash := hashKey(url)
	delete(ch.nodes, hash)
	ch.ring = removeFromSlice(ch.ring, hash)

	for i := 0; i < ch.virtualNodes; i++ {
		vHash := hashKey(fmt.Sprintf("%s#%d", url, i))
		delete(ch.nodes, vHash)
		ch.ring = removeFromSlice(ch.ring, vHash)
	}
}

func (ch *ConsistentHashLB) GetNode(key string) *Backend {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if len(ch.ring) == 0 {
		return nil
	}

	hash := hashKey(key)
	idx := ch.binarySearch(hash)

	return ch.nodes[ch.ring[idx]]
}

func (ch *ConsistentHashLB) sortRing() {
	if ch.sorted {
		return
	}

	for i := 0; i < len(ch.ring); i++ {
		for j := i + 1; j < len(ch.ring); j++ {
			if ch.ring[i] > ch.ring[j] {
				ch.ring[i], ch.ring[j] = ch.ring[j], ch.ring[i]
			}
		}
	}

	ch.sorted = true
}

func (ch *ConsistentHashLB) binarySearch(hash uint32) int {
	low := 0
	high := len(ch.ring) - 1

	for low < high {
		mid := (low + high + 1) / 2
		if ch.ring[mid] <= hash {
			low = mid
		} else {
			high = mid - 1
		}
	}

	if ch.ring[low] >= hash {
		return low
	}

	return 0
}

func hashKey(key string) uint32 {
	hash := uint32(5381)
	for _, c := range key {
		hash = ((hash << 5) + hash) + uint32(c)
	}
	return hash
}

func removeFromSlice(slice []uint32, val uint32) []uint32 {
	for i, v := range slice {
		if v == val {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func (lb *LoadBalancer) SetStrategy(strategy LoadBalancerStrategy) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.strategy = strategy
}

func (lb *LoadBalancer) GetStrategy() LoadBalancerStrategy {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.strategy
}

func (lb *LoadBalancer) UpdateBackendWeight(url string, weight int) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, backend := range lb.backends {
		if backend.URL == url {
			backend.Weight = weight
			return nil
		}
	}

	return fmt.Errorf("backend not found: %s", url)
}

func (lb *LoadBalancer) SetBackendHealthy(url string, healthy bool) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, backend := range lb.backends {
		if backend.URL == url {
			backend.Healthy = healthy
			return
		}
	}
}

func (lb *LoadBalancer) GetBackendByURL(url string) (*Backend, bool) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	for _, backend := range lb.backends {
		if backend.URL == url {
			return backend, true
		}
	}

	return nil, false
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
