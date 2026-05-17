package highavailability

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type CircuitState string

const (
	CircuitStateClosed   CircuitState = "closed"
	CircuitStateOpen     CircuitState = "open"
	CircuitStateHalfOpen CircuitState = "half_open"
)

type CircuitBreakerConfig struct {
	FailureThreshold  int
	SuccessThreshold  int
	Timeout          time.Duration
	RequestTimeout   time.Duration
	HalfOpenMaxReqs  int
}

var defaultCircuitBreakerConfig = &CircuitBreakerConfig{
	FailureThreshold: 5,
	SuccessThreshold: 2,
	Timeout:         60 * time.Second,
	RequestTimeout:  10 * time.Second,
	HalfOpenMaxReqs: 3,
}

type CircuitBreaker struct {
	name          string
	state         CircuitState
	config        *CircuitBreakerConfig
	mu            sync.RWMutex
	failures      int32
	successes     int32
	lastFailure   time.Time
	lastStateChange time.Time
	halfOpenReqs  int32
	totalRequests uint64
	totalFailures uint64
	totalSuccesses uint64
}

type CircuitBreakerStats struct {
	Name             string        `json:"name"`
	State            CircuitState  `json:"state"`
	Failures         int32         `json:"failures"`
	Successes        int32         `json:"successes"`
	TotalRequests    uint64        `json:"total_requests"`
	TotalFailures    uint64        `json:"total_failures"`
	TotalSuccesses   uint64        `json:"total_successes"`
	LastFailure      time.Time     `json:"last_failure"`
	LastStateChange  time.Time     `json:"last_state_change"`
	FailureRate      float64       `json:"failure_rate"`
	SuccessRate      float64       `json:"success_rate"`
}

func NewCircuitBreaker(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = defaultCircuitBreakerConfig
	}

	return &CircuitBreaker{
		name:           name,
		state:          CircuitStateClosed,
		config:         config,
		lastStateChange: time.Now(),
	}
}

func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.AddUint64(&cb.totalRequests, 1)

	switch cb.state {
	case CircuitStateClosed:
		return true

	case CircuitStateOpen:
		if time.Since(cb.lastStateChange) >= cb.config.Timeout {
			cb.state = CircuitStateHalfOpen
			cb.lastStateChange = time.Now()
			cb.halfOpenReqs = 0
			return true
		}
		return false

	case CircuitStateHalfOpen:
		if atomic.LoadInt32(&cb.halfOpenReqs) < int32(cb.config.HalfOpenMaxReqs) {
			atomic.AddInt32(&cb.halfOpenReqs, 1)
			return true
		}
		return false

	default:
		return false
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.AddUint64(&cb.totalSuccesses, 1)
	atomic.AddInt32(&cb.successes, 1)
	atomic.StoreInt32(&cb.failures, 0)

	switch cb.state {
	case CircuitStateHalfOpen:
		if atomic.LoadInt32(&cb.successes) >= int32(cb.config.SuccessThreshold) {
			cb.state = CircuitStateClosed
			cb.lastStateChange = time.Now()
			atomic.StoreInt32(&cb.successes, 0)
		}

	case CircuitStateClosed:
		break
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.AddUint64(&cb.totalFailures, 1)
	failures := atomic.AddInt32(&cb.failures, 1)
	cb.lastFailure = time.Now()

	switch cb.state {
	case CircuitStateHalfOpen:
		cb.state = CircuitStateOpen
		cb.lastStateChange = time.Now()
		atomic.StoreInt32(&cb.halfOpenReqs, 0)

	case CircuitStateClosed:
		if failures >= int32(cb.config.FailureThreshold) {
			cb.state = CircuitStateOpen
			cb.lastStateChange = time.Now()
		}
	}
}

func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) GetStats() *CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	total := atomic.LoadUint64(&cb.totalRequests)
	failures := atomic.LoadUint64(&cb.totalFailures)
	successes := atomic.LoadUint64(&cb.totalSuccesses)

	var failureRate, successRate float64
	if total > 0 {
		failureRate = float64(failures) / float64(total) * 100
		successRate = float64(successes) / float64(total) * 100
	}

	return &CircuitBreakerStats{
		Name:            cb.name,
		State:           cb.state,
		Failures:        atomic.LoadInt32(&cb.failures),
		Successes:       atomic.LoadInt32(&cb.successes),
		TotalRequests:   total,
		TotalFailures:   failures,
		TotalSuccesses:  successes,
		LastFailure:     cb.lastFailure,
		LastStateChange: cb.lastStateChange,
		FailureRate:     failureRate,
		SuccessRate:     successRate,
	}
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitStateClosed
	cb.lastStateChange = time.Now()
	atomic.StoreInt32(&cb.failures, 0)
	atomic.StoreInt32(&cb.successes, 0)
	atomic.StoreInt32(&cb.halfOpenReqs, 0)
}

func (cb *CircuitBreaker) ForceOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitStateOpen
	cb.lastStateChange = time.Now()
}

func (cb *CircuitBreaker) ForceClosed() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitStateClosed
	cb.lastStateChange = time.Now()
	atomic.StoreInt32(&cb.failures, 0)
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if !cb.AllowRequest() {
		return fmt.Errorf("circuit breaker is open for: %s", cb.name)
	}

	type result struct {
		err error
	}

	done := make(chan result, 1)

	go func() {
		err := fn()
		done <- result{err: err}
	}()

	select {
	case <-ctx.Done():
		cb.RecordFailure()
		return ctx.Err()
	case r := <-done:
		if r.err != nil {
			cb.RecordFailure()
			return r.err
		}
		cb.RecordSuccess()
		return nil
	}
}

type CircuitBreakerGroup struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
	config   *CircuitBreakerConfig
}

func NewCircuitBreakerGroup(config *CircuitBreakerConfig) *CircuitBreakerGroup {
	if config == nil {
		config = defaultCircuitBreakerConfig
	}

	return &CircuitBreakerGroup{
		breakers: make(map[string]*CircuitBreaker),
		config:   config,
	}
}

func (g *CircuitBreakerGroup) GetOrCreate(name string) *CircuitBreaker {
	g.mu.RLock()
	cb, exists := g.breakers[name]
	g.mu.RUnlock()

	if exists {
		return cb
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if cb, exists = g.breakers[name]; exists {
		return cb
	}

	cb = NewCircuitBreaker(name, g.config)
	g.breakers[name] = cb
	return cb
}

func (g *CircuitBreakerGroup) Get(name string) (*CircuitBreaker, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	cb, ok := g.breakers[name]
	return cb, ok
}

func (g *CircuitBreakerGroup) GetAllStats() map[string]*CircuitBreakerStats {
	g.mu.RLock()
	defer g.mu.RUnlock()

	stats := make(map[string]*CircuitBreakerStats, len(g.breakers))
	for name, cb := range g.breakers {
		stats[name] = cb.GetStats()
	}
	return stats
}

func (g *CircuitBreakerGroup) ResetAll() {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, cb := range g.breakers {
		cb.Reset()
	}
}

type HeartbeatManager struct {
	heartbeats map[string]*Heartbeat
	mu         sync.RWMutex
	interval   time.Duration
	timeout    time.Duration
	stopCh     chan struct{}
	listeners  []HeartbeatListener
	listenerMu sync.RWMutex
}

type Heartbeat struct {
	ID        string
	Name      string
	Interval  time.Duration
	Timeout   time.Duration
	LastBeat  time.Time
	NextBeat  time.Time
	Active    bool
	Metadata  map[string]string
	FailureCount int
}

type HeartbeatListener interface {
	OnHeartbeatTimeout(id string, name string)
	OnHeartbeatFailure(id string, name string, count int)
	OnHeartbeatRecovery(id string, name string)
}

type HeartbeatCallback func(id string, name string)

func (c HeartbeatCallback) OnHeartbeatTimeout(id string, name string) { c(id, name) }
func (c HeartbeatCallback) OnHeartbeatFailure(id string, name string, count int) {}
func (c HeartbeatCallback) OnHeartbeatRecovery(id string, name string) {}

type HeartbeatManagerConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}

var defaultHeartbeatConfig = &HeartbeatManagerConfig{
	Interval: 5 * time.Second,
	Timeout:  30 * time.Second,
}

func NewHeartbeatManager(config *HeartbeatManagerConfig) *HeartbeatManager {
	if config == nil {
		config = defaultHeartbeatConfig
	}

	return &HeartbeatManager{
		heartbeats: make(map[string]*Heartbeat),
		interval:   config.Interval,
		timeout:    config.Timeout,
		stopCh:     make(chan struct{}),
		listeners:  make([]HeartbeatListener, 0),
	}
}

func (hm *HeartbeatManager) Register(id, name string, metadata map[string]string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.heartbeats[id]; exists {
		return fmt.Errorf("heartbeat already exists: %s", id)
	}

	hb := &Heartbeat{
		ID:        id,
		Name:      name,
		Interval:  hm.interval,
		Timeout:   hm.timeout,
		LastBeat:  time.Now(),
		NextBeat:  time.Now().Add(hm.interval),
		Active:    true,
		Metadata:  metadata,
	}

	hm.heartbeats[id] = hb
	return nil
}

func (hm *HeartbeatManager) Unregister(id string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.heartbeats[id]; !exists {
		return fmt.Errorf("heartbeat not found: %s", id)
	}

	delete(hm.heartbeats, id)
	return nil
}

func (hm *HeartbeatManager) Beat(id string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hb, exists := hm.heartbeats[id]
	if !exists {
		return fmt.Errorf("heartbeat not found: %s", id)
	}

	now := time.Now()
	hb.LastBeat = now
	hb.NextBeat = now.Add(hb.Interval)

	if !hb.Active {
		hb.Active = true
		hb.FailureCount = 0
		hm.notifyRecovery(id, hb.Name)
	}

	return nil
}

func (hm *HeartbeatManager) Start(ctx context.Context) {
	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hm.stopCh:
			return
		case <-ticker.C:
			hm.checkHeartbeats()
		}
	}
}

func (hm *HeartbeatManager) Stop() {
	close(hm.stopCh)
}

func (hm *HeartbeatManager) checkHeartbeats() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	now := time.Now()

	for id, hb := range hm.heartbeats {
		if now.After(hb.NextBeat) {
			hb.FailureCount++
			hb.Active = false
			hm.notifyFailure(id, hb.Name, hb.FailureCount)

			if now.Sub(hb.LastBeat) > hb.Timeout {
				hm.notifyTimeout(id, hb.Name)
			}
		}
	}
}

func (hm *HeartbeatManager) AddListener(listener HeartbeatListener) {
	hm.listenerMu.Lock()
	defer hm.listenerMu.Unlock()
	hm.listeners = append(hm.listeners, listener)
}

func (hm *HeartbeatManager) notifyTimeout(id, name string) {
	hm.listenerMu.RLock()
	defer hm.listenerMu.RUnlock()

	for _, listener := range hm.listeners {
		listener.OnHeartbeatTimeout(id, name)
	}
}

func (hm *HeartbeatManager) notifyFailure(id, name string, count int) {
	hm.listenerMu.RLock()
	defer hm.listenerMu.RUnlock()

	for _, listener := range hm.listeners {
		listener.OnHeartbeatFailure(id, name, count)
	}
}

func (hm *HeartbeatManager) notifyRecovery(id, name string) {
	hm.listenerMu.RLock()
	defer hm.listenerMu.RUnlock()

	for _, listener := range hm.listeners {
		listener.OnHeartbeatRecovery(id, name)
	}
}

func (hm *HeartbeatManager) GetHeartbeat(id string) (*Heartbeat, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	hb, ok := hm.heartbeats[id]
	return hb, ok
}

func (hm *HeartbeatManager) GetAllHeartbeats() []*Heartbeat {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	heartbeats := make([]*Heartbeat, 0, len(hm.heartbeats))
	for _, hb := range hm.heartbeats {
		heartbeats = append(heartbeats, hb)
	}
	return heartbeats
}

func (hm *HeartbeatManager) GetActiveCount() int {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	count := 0
	for _, hb := range hm.heartbeats {
		if hb.Active {
			count++
		}
	}
	return count
}

type AutoRestartManager struct {
	processes map[string]*ProcessInfo
	mu         sync.RWMutex
	config     *AutoRestartConfig
	stopCh     chan struct{}
	listeners  []RestartListener
	listenerMu sync.RWMutex
}

type ProcessInfo struct {
	ID           string
	Name         string
	Command      string
	Args         []string
	RestartCount int
	LastRestart  time.Time
	Status       string
	Metadata     map[string]string
}

type RestartListener interface {
	OnProcessRestart(id, name string, count int)
	OnProcessCrash(id, name string, err error)
}

type RestartConfig struct {
	MaxRetries       int
	RetryInterval    time.Duration
	BackoffMultiplier float64
	MaxBackoff       time.Duration
}

var defaultRestartConfig = &RestartConfig{
	MaxRetries:        3,
	RetryInterval:     5 * time.Second,
	BackoffMultiplier: 2.0,
	MaxBackoff:        60 * time.Second,
}

type AutoRestartConfig struct {
	Restart *RestartConfig
}

func NewAutoRestartManager(config *AutoRestartConfig) *AutoRestartManager {
	if config == nil || config.Restart == nil {
		config = &AutoRestartConfig{
			Restart: defaultRestartConfig,
		}
	}

	return &AutoRestartManager{
		processes: make(map[string]*ProcessInfo),
		config:    config,
		stopCh:    make(chan struct{}),
		listeners: make([]RestartListener, 0),
	}
}

func (arm *AutoRestartManager) RegisterProcess(info *ProcessInfo) error {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	if _, exists := arm.processes[info.ID]; exists {
		return fmt.Errorf("process already registered: %s", info.ID)
	}

	info.Status = "running"
	info.RestartCount = 0
	arm.processes[info.ID] = info

	return nil
}

func (arm *AutoRestartManager) UnregisterProcess(id string) error {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	if _, exists := arm.processes[id]; !exists {
		return fmt.Errorf("process not found: %s", id)
	}

	delete(arm.processes, id)
	return nil
}

func (arm *AutoRestartManager) MarkRunning(id string) error {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	process, exists := arm.processes[id]
	if !exists {
		return fmt.Errorf("process not found: %s", id)
	}

	process.Status = "running"
	return nil
}

func (arm *AutoRestartManager) MarkStopped(id string) error {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	process, exists := arm.processes[id]
	if !exists {
		return fmt.Errorf("process not found: %s", id)
	}

	process.Status = "stopped"
	process.LastRestart = time.Now()
	process.RestartCount++

	arm.notifyRestart(id, process.Name, process.RestartCount)

	if process.RestartCount >= arm.config.Restart.MaxRetries {
		process.Status = "failed"
	}

	return nil
}

func (arm *AutoRestartManager) AddListener(listener RestartListener) {
	arm.listenerMu.Lock()
	defer arm.listenerMu.Unlock()
	arm.listeners = append(arm.listeners, listener)
}

func (arm *AutoRestartManager) notifyRestart(id, name string, count int) {
	arm.listenerMu.RLock()
	defer arm.listenerMu.RUnlock()

	for _, listener := range arm.listeners {
		listener.OnProcessRestart(id, name, count)
	}
}

func (arm *AutoRestartManager) notifyCrash(id, name string, err error) {
	arm.listenerMu.RLock()
	defer arm.listenerMu.RUnlock()

	for _, listener := range arm.listeners {
		listener.OnProcessCrash(id, name, err)
	}
}

func (arm *AutoRestartManager) GetProcess(id string) (*ProcessInfo, bool) {
	arm.mu.RLock()
	defer arm.mu.RUnlock()

	process, ok := arm.processes[id]
	return process, ok
}

func (arm *AutoRestartManager) GetAllProcesses() []*ProcessInfo {
	arm.mu.RLock()
	defer arm.mu.RUnlock()

	processes := make([]*ProcessInfo, 0, len(arm.processes))
	for _, p := range arm.processes {
		processes = append(processes, p)
	}
	return processes
}

func (arm *AutoRestartManager) GetRestartDelay(count int) time.Duration {
	delay := arm.config.Restart.RetryInterval
	for i := 1; i < count; i++ {
		delay = time.Duration(float64(delay) * arm.config.Restart.BackoffMultiplier)
		if delay > arm.config.Restart.MaxBackoff {
			delay = arm.config.Restart.MaxBackoff
		}
	}
	return delay
}

type MetricsCollector struct {
	mu              sync.RWMutex
	metrics         map[string]*Metric
	aggregators     map[string]*MetricAggregator
	windowSize      time.Duration
	cleanupInterval time.Duration
	stopCh          chan struct{}
}

type Metric struct {
	Name      string
	Value     float64
	Timestamp time.Time
	Labels    map[string]string
	Type      MetricType
}

type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

type MetricAggregator struct {
	Name      string
	Type      MetricType
	Values    []float64
	Sum       float64
	Count     int
	Min       float64
	Max       float64
	LastValue float64
}

func NewMetricsCollector(windowSize, cleanupInterval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		metrics:         make(map[string]*Metric),
		aggregators:     make(map[string]*MetricAggregator),
		windowSize:      windowSize,
		cleanupInterval: cleanupInterval,
		stopCh:          make(chan struct{}),
	}
}

func (mc *MetricsCollector) RecordCounter(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.makeKey(name, labels)

	metric := &Metric{
		Name:      name,
		Value:     value,
		Timestamp: time.Now(),
		Labels:    labels,
		Type:      MetricTypeCounter,
	}

	mc.metrics[key] = metric

	if _, exists := mc.aggregators[key]; !exists {
		mc.aggregators[key] = &MetricAggregator{
			Name:  name,
			Type:  MetricTypeCounter,
			Min:   math.MaxFloat64,
			Values: make([]float64, 0),
		}
	}

	agg := mc.aggregators[key]
	agg.Values = append(agg.Values, value)
	agg.Sum += value
	agg.Count++
	agg.LastValue = value
	if value < agg.Min {
		agg.Min = value
	}
	if value > agg.Max {
		agg.Max = value
	}
}

func (mc *MetricsCollector) RecordGauge(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.makeKey(name, labels)

	metric := &Metric{
		Name:      name,
		Value:     value,
		Timestamp: time.Now(),
		Labels:    labels,
		Type:      MetricTypeGauge,
	}

	mc.metrics[key] = metric

	if _, exists := mc.aggregators[key]; !exists {
		mc.aggregators[key] = &MetricAggregator{
			Name:  name,
			Type:  MetricTypeGauge,
			Min:   math.MaxFloat64,
			Values: make([]float64, 0),
		}
	}

	agg := mc.aggregators[key]
	agg.Values = append(agg.Values, value)
	agg.Sum += value
	agg.Count++
	agg.LastValue = value
	if value < agg.Min {
		agg.Min = value
	}
	if value > agg.Max {
		agg.Max = value
	}
}

func (mc *MetricsCollector) RecordHistogram(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.makeKey(name, labels)

	metric := &Metric{
		Name:      name,
		Value:     value,
		Timestamp: time.Now(),
		Labels:    labels,
		Type:      MetricTypeHistogram,
	}

	mc.metrics[key] = metric

	if _, exists := mc.aggregators[key]; !exists {
		mc.aggregators[key] = &MetricAggregator{
			Name:  name,
			Type:  MetricTypeHistogram,
			Min:   math.MaxFloat64,
			Values: make([]float64, 0),
		}
	}

	agg := mc.aggregators[key]
	agg.Values = append(agg.Values, value)
	agg.Sum += value
	agg.Count++
	agg.LastValue = value
	if value < agg.Min {
		agg.Min = value
	}
	if value > agg.Max {
		agg.Max = value
	}
}

func (mc *MetricsCollector) GetMetric(name string, labels map[string]string) (*Metric, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	key := mc.makeKey(name, labels)
	metric, ok := mc.metrics[key]
	return metric, ok
}

func (mc *MetricsCollector) GetAggregator(name string, labels map[string]string) (*MetricAggregator, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	key := mc.makeKey(name, labels)
	agg, ok := mc.aggregators[key]
	return agg, ok
}

func (mc *MetricsCollector) GetAllAggregators() map[string]*MetricAggregator {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]*MetricAggregator, len(mc.aggregators))
	for k, v := range mc.aggregators {
		result[k] = v
	}
	return result
}

func (mc *MetricsCollector) Start(ctx context.Context) {
	ticker := time.NewTicker(mc.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-mc.stopCh:
			return
		case <-ticker.C:
			mc.cleanup()
		}
	}
}

func (mc *MetricsCollector) Stop() {
	close(mc.stopCh)
}

func (mc *MetricsCollector) cleanup() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	cutoff := time.Now().Add(-mc.windowSize)

	for key, metric := range mc.metrics {
		if metric.Timestamp.Before(cutoff) {
			delete(mc.metrics, key)
		}
	}
}

func (mc *MetricsCollector) makeKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	key := name
	for k, v := range labels {
		key += fmt.Sprintf("|%s=%s", k, v)
	}
	return key
}

func (mc *MetricsCollector) GetAverage(name string, labels map[string]string) float64 {
	agg, ok := mc.GetAggregator(name, labels)
	if !ok || agg.Count == 0 {
		return 0
	}
	return agg.Sum / float64(agg.Count)
}

func (mc *MetricsCollector) GetPercentile(name string, labels map[string]string, percentile float64) float64 {
	agg, ok := mc.GetAggregator(name, labels)
	if !ok || len(agg.Values) == 0 {
		return 0
	}

	sorted := make([]float64, len(agg.Values))
	copy(sorted, agg.Values)

	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	idx := int(float64(len(sorted)-1) * percentile / 100.0)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}

type AvailabilityTracker struct {
	mu          sync.RWMutex
	uptime      time.Duration
	downtime    time.Duration
	startTime   time.Time
	lastUpTime  time.Time
	isUp        bool
	totalReqs   uint64
	totalErrors uint64
	sloWindow   time.Duration
	sloTarget   float64
}

func NewAvailabilityTracker(sloWindow time.Duration, sloTarget float64) *AvailabilityTracker {
	return &AvailabilityTracker{
		startTime: time.Now(),
		lastUpTime: time.Now(),
		isUp:       true,
		sloWindow:  sloWindow,
		sloTarget:  sloTarget,
	}
}

func (at *AvailabilityTracker) RecordUp() {
	at.mu.Lock()
	defer at.mu.Unlock()

	if !at.isUp {
		at.isUp = true
		at.lastUpTime = time.Now()
	}
}

func (at *AvailabilityTracker) RecordDown() {
	at.mu.Lock()
	defer at.mu.Unlock()

	if at.isUp {
		at.isUp = false
		at.downtime += time.Since(at.lastUpTime)
	}
}

func (at *AvailabilityTracker) RecordRequest() {
	atomic.AddUint64(&at.totalReqs, 1)
}

func (at *AvailabilityTracker) RecordError() {
	atomic.AddUint64(&at.totalErrors, 1)
}

func (at *AvailabilityTracker) GetAvailability() float64 {
	at.mu.RLock()
	defer at.mu.RUnlock()

	totalTime := time.Since(at.startTime)
	if totalTime == 0 {
		return 100.0
	}

	upTime := totalTime - at.downtime
	return float64(upTime) / float64(totalTime) * 100
}

func (at *AvailabilityTracker) GetErrorRate() float64 {
	total := atomic.LoadUint64(&at.totalReqs)
	if total == 0 {
		return 0
	}

	errors := atomic.LoadUint64(&at.totalErrors)
	return float64(errors) / float64(total) * 100
}

func (at *AvailabilityTracker) GetSLOStatus() bool {
	at.mu.RLock()
	defer at.mu.RUnlock()

	totalTime := time.Since(at.startTime)
	if totalTime == 0 {
		return true
	}

	upTime := totalTime - at.downtime
	availability := float64(upTime) / float64(totalTime) * 100
	return availability >= at.sloTarget
}

func (at *AvailabilityTracker) GetStats() map[string]interface{} {
	at.mu.RLock()
	defer at.mu.RUnlock()

	return map[string]interface{}{
		"uptime":        time.Since(at.startTime).String(),
		"availability":  at.GetAvailability(),
		"error_rate":    at.GetErrorRate(),
		"total_requests": atomic.LoadUint64(&at.totalReqs),
		"total_errors":  atomic.LoadUint64(&at.totalErrors),
		"is_up":         at.isUp,
		"slo_target":    at.sloTarget,
		"slo_met":       at.GetSLOStatus(),
	}
}
