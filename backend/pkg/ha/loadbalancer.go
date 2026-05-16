package ha

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/api/middleware"
)

type LoadBalancerConfig struct {
	Strategy           middleware.LoadBalancerStrategy
	HealthCheckEnabled bool
	HealthCheckInterval time.Duration
	MaxRetries        int
	Timeout           time.Duration
	Backoff           *BackoffConfig
	CircuitBreaker    *CircuitBreakerConfig
	RetryPolicy       *RetryPolicyConfig
}

type BackoffConfig struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	MaxElapsedTime  time.Duration
}

type CircuitBreakerConfig struct {
	FailureThreshold int
	SuccessThreshold int
	Timeout         time.Duration
}

type RetryPolicyConfig struct {
	MaxAttempts    int
	RetryableCodes []int
	Backoff        *BackoffConfig
}

func DefaultLoadBalancerConfig() *LoadBalancerConfig {
	return &LoadBalancerConfig{
		Strategy:           middleware.StrategyRoundRobin,
		HealthCheckEnabled: true,
		HealthCheckInterval: 10 * time.Second,
		MaxRetries:        3,
		Timeout:           30 * time.Second,
		Backoff: &BackoffConfig{
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
			MaxElapsedTime:  5 * time.Minute,
		},
		CircuitBreaker: &CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          60 * time.Second,
		},
		RetryPolicy: &RetryPolicyConfig{
			MaxAttempts: 3,
			RetryableCodes: []int{500, 502, 503, 504},
			Backoff: &BackoffConfig{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     10 * time.Second,
				Multiplier:      2.0,
			},
		},
	}
}

type HAProxy struct {
	config        *LoadBalancerConfig
	loadBalancer  *middleware.LoadBalancer
	healthChecker *HealthChecker
	failover      *FailoverController
	circuitBreakers map[string]*CircuitBreaker
	retryPolicies   map[string]*RetryPolicy
	metrics        *HAMetrics
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	stopChan       chan struct{}
}

type CircuitBreaker struct {
	state       CircuitBreakerState
	failures    int
	successes   int
	threshold   int
	timeout     time.Duration
	lastFailure time.Time
	mu          sync.RWMutex
}

type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateHalfOpen
	StateOpen
)

type RetryPolicy struct {
	attempts    int
	maxAttempts int
	backoff     *BackoffConfig
	mu          sync.RWMutex
}

type HAMetrics struct {
	TotalRequests   int64
	FailedRequests  int64
	SuccessfulRequests int64
	RetriedRequests int64
	CircuitTripped  int64
	AvgLatency      int64
	mu              sync.RWMutex
	latencies       []time.Duration
}

func NewHAMetrics() *HAMetrics {
	return &HAMetrics{
		latencies: make([]time.Duration, 0, 1000),
	}
}

func (m *HAMetrics) RecordRequest(success bool, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++
	if success {
		m.SuccessfulRequests++
	} else {
		m.FailedRequests++
	}

	m.latencies = append(m.latencies, latency)
	if len(m.latencies) > 1000 {
		m.latencies = m.latencies[1:]
	}

	var total int64
	for _, l := range m.latencies {
		total += l.Nanoseconds()
	}
	m.AvgLatency = total / int64(len(m.latencies))
}

func (m *HAMetrics) RecordRetry() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RetriedRequests++
}

func (m *HAMetrics) RecordCircuitTrip() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CircuitTripped++
}

func NewHAProxy(config *LoadBalancerConfig) *HAProxy {
	if config == nil {
		config = DefaultLoadBalancerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &HAProxy{
		config:         config,
		loadBalancer:   middleware.NewLoadBalancer(config.Strategy),
		healthChecker:  NewHealthChecker(config.HealthCheckInterval, 5*time.Second),
		circuitBreakers: make(map[string]*CircuitBreaker),
		retryPolicies:  make(map[string]*RetryPolicy),
		metrics:        NewHAMetrics(),
		ctx:            ctx,
		cancel:         cancel,
		stopChan:       make(chan struct{}),
	}
}

func (hap *HAProxy) Start(ctx context.Context) {
	if hap.config.HealthCheckEnabled {
		hap.healthChecker.Start(ctx)
	}
}

func (hap *HAProxy) Stop() {
	hap.cancel()
	close(hap.stopChan)
}

func (hap *HAProxy) AddBackend(url string, weight int) {
	hap.loadBalancer.AddBackend(url, weight)
	hap.healthChecker.AddNode(url, url)

	hap.mu.Lock()
	hap.circuitBreakers[url] = NewCircuitBreaker(
		hap.config.CircuitBreaker.FailureThreshold,
		hap.config.CircuitBreaker.SuccessThreshold,
		hap.config.CircuitBreaker.Timeout,
	)
	hap.retryPolicies[url] = NewRetryPolicy(hap.config.RetryPolicy.MaxAttempts, hap.config.RetryPolicy.Backoff)
	hap.mu.Unlock()
}

func (hap *HAProxy) RemoveBackend(url string) {
	hap.loadBalancer.RemoveBackend(url)
	hap.healthChecker.RemoveNode(url)

	hap.mu.Lock()
	delete(hap.circuitBreakers, url)
	delete(hap.retryPolicies, url)
	hap.mu.Unlock()
}

func (hap *HAProxy) GetBackend(clientIP string) (*middleware.Backend, error) {
	return hap.loadBalancer.GetBackend(clientIP)
}

func (hap *HAProxy) IsCircuitOpen(url string) bool {
	hap.mu.RLock()
	cb, exists := hap.circuitBreakers[url]
	hap.mu.RUnlock()

	if !exists {
		return false
	}

	return cb.IsOpen()
}

func (hap *HAProxy) RecordSuccess(url string) {
	hap.mu.RLock()
	cb, exists := hap.circuitBreakers[url]
	hap.mu.RUnlock()

	if exists {
		cb.RecordSuccess()
	}

	hap.mu.RLock()
	lb := hap.loadBalancer
	hap.mu.RUnlock()

	backend, _ := lb.GetBackendByURL(url)
	if backend != nil {
		lb.RecordSuccess(backend, 0)
	}
}

func (hap *HAProxy) RecordFailure(url string) {
	hap.mu.RLock()
	cb, exists := hap.circuitBreakers[url]
	hap.mu.RUnlock()

	if exists {
		tripped := cb.RecordFailure()
		if tripped {
			hap.metrics.RecordCircuitTrip()
		}
	}

	hap.mu.RLock()
	lb := hap.loadBalancer
	hap.mu.RUnlock()

	backend, _ := lb.GetBackendByURL(url)
	if backend != nil {
		lb.RecordFailure(backend)
	}
}

func (hap *HAProxy) ExecuteWithRetry(url string, fn func() error) error {
	hap.mu.RLock()
	policy, exists := hap.retryPolicies[url]
	hap.mu.RUnlock()

	if !exists {
		return fn()
	}

	var lastErr error
	for attempt := 0; attempt <= policy.maxAttempts; attempt++ {
		if attempt > 0 {
			delay := policy.GetBackoff(attempt)
			time.Sleep(delay)
			hap.metrics.RecordRetry()
		}

		if err := fn(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}

	return lastErr
}

func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:           StateClosed,
		failures:        0,
		successes:       0,
		threshold:       failureThreshold,
		timeout:         timeout,
	}
}

func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.failures = 0
			cb.mu.Unlock()
			cb.mu.RLock()
			return false
		}
		return true
	}

	return false
}

func (cb *CircuitBreaker) RecordFailure() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.state == StateHalfOpen {
		cb.state = StateOpen
		return true
	}

	if cb.failures >= cb.threshold {
		cb.state = StateOpen
		return true
	}

	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.successes++
		if cb.successes >= 2 {
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
		}
	} else {
		cb.failures = 0
	}
}

func (cb *CircuitBreaker) GetState() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

func NewRetryPolicy(maxAttempts int, backoff *BackoffConfig) *RetryPolicy {
	if backoff == nil {
		backoff = &BackoffConfig{
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     10 * time.Second,
			Multiplier:      2.0,
		}
	}

	return &RetryPolicy{
		maxAttempts: maxAttempts,
		backoff:     backoff,
	}
}

func (rp *RetryPolicy) GetBackoff(attempt int) time.Duration {
	interval := float64(rp.backoff.InitialInterval)
	for i := 0; i < attempt; i++ {
		interval *= rp.backoff.Multiplier
		if interval > float64(rp.backoff.MaxInterval) {
			interval = float64(rp.backoff.MaxInterval)
		}
	}
	return time.Duration(interval)
}

func (rp *RetryPolicy) ShouldRetry(statusCode int, retryableCodes []int) bool {
	for _, code := range retryableCodes {
		if code == statusCode {
			return true
		}
	}
	return false
}

func (hap *HAProxy) GetMetrics() map[string]interface{} {
	stats := hap.loadBalancer.GetStats()
	healthStatus := hap.healthChecker.GetClusterHealth()
	lbMetrics := hap.metrics

	return map[string]interface{}{
		"total_requests":       lbMetrics.TotalRequests,
		"successful_requests":   lbMetrics.SuccessfulRequests,
		"failed_requests":      lbMetrics.FailedRequests,
		"retried_requests":     lbMetrics.RetriedRequests,
		"circuit_tripped":      lbMetrics.CircuitTripped,
		"avg_latency_ms":       lbMetrics.AvgLatency / 1e6,
		"backends":             stats,
		"cluster_health":       healthStatus,
	}
}

func (hap *HAProxy) GetCircuitBreakerStatus(url string) string {
	hap.mu.RLock()
	cb, exists := hap.circuitBreakers[url]
	hap.mu.RUnlock()

	if !exists {
		return "not_configured"
	}

	return cb.GetState()
}

func (hap *HAProxy) GetAllCircuitBreakerStatuses() map[string]string {
	hap.mu.RLock()
	defer hap.mu.RUnlock()

	statuses := make(map[string]string)
	for url, cb := range hap.circuitBreakers {
		statuses[url] = cb.GetState()
	}
	return statuses
}

func (hap *HAProxy) ResetCircuitBreaker(url string) {
	hap.mu.Lock()
	defer hap.mu.Unlock()

	if cb, exists := hap.circuitBreakers[url]; exists {
		cb.mu.Lock()
		cb.state = StateClosed
		cb.failures = 0
		cb.successes = 0
		cb.mu.Unlock()
	}
}

type HighAvailabilityLoadBalancer struct {
	*HAProxy
	observer *LoadBalancerObserver
}

type LoadBalancerObserver struct {
	callbacks []func(event *LBEvent)
	mu        sync.RWMutex
}

type LBEvent struct {
	Type        string
	BackendURL  string
	Timestamp   time.Time
	Metadata    map[string]interface{}
}

func NewHighAvailabilityLoadBalancer(config *LoadBalancerConfig) *HighAvailabilityLoadBalancer {
	hap := NewHAProxy(config)

	observer := &LoadBalancerObserver{
		callbacks: make([]func(event *LBEvent), 0),
	}

	return &HighAvailabilityLoadBalancer{
		HAProxy:   hap,
		observer:  observer,
	}
}

func (lb *HighAvailabilityLoadBalancer) AddEventCallback(callback func(event *LBEvent)) {
	lb.observer.mu.Lock()
	defer lb.observer.mu.Unlock()
	lb.observer.callbacks = append(lb.observer.callbacks, callback)
}

func (lb *HighAvailabilityLoadBalancer) notify(event *LBEvent) {
	lb.observer.mu.RLock()
	callbacks := lb.observer.callbacks
	lb.observer.mu.RUnlock()

	for _, callback := range callbacks {
		go callback(event)
	}
}

func (lb *HighAvailabilityLoadBalancer) SelectBackend(clientIP string) (*middleware.Backend, error) {
	backend, err := lb.HAProxy.GetBackend(clientIP)
	if err != nil {
		return nil, err
	}

	if lb.HAProxy.IsCircuitOpen(backend.URL) {
		lb.notify(&LBEvent{
			Type:       "circuit_open",
			BackendURL: backend.URL,
			Timestamp:  time.Now(),
		})
		return nil, fmt.Errorf("circuit breaker open for %s", backend.URL)
	}

	return backend, nil
}

func (lb *HighAvailabilityLoadBalancer) RecordRequest(url string, success bool, latency time.Duration) {
	lb.HAProxy.metrics.RecordRequest(success, latency)

	lb.notify(&LBEvent{
		Type:       "request_complete",
		BackendURL: url,
		Timestamp:  time.Now(),
		Metadata: map[string]interface{}{
			"success": success,
			"latency": latency.String(),
		},
	})
}

func (lb *HighAvailabilityLoadBalancer) GetHealthSummary() map[string]interface{} {
	health := lb.HAProxy.healthChecker.GetClusterHealth()
	circuitBreakers := lb.HAProxy.GetAllCircuitBreakerStatuses()
	backends := lb.HAProxy.loadBalancer.GetStats()

	return map[string]interface{}{
		"cluster_status":   health.ClusterStatus,
		"healthy_nodes":    health.HealthyNodes,
		"total_nodes":      health.TotalNodes,
		"circuit_breakers": circuitBreakers,
		"backends":         backends,
	}
}
