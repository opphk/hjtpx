package circuitbreaker

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type State int32

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half_open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

type CircuitBreaker interface {
	Execute(func() error) error
	State() State
	Counts() Counts
	CanAttempt() bool
	RecordSuccess()
	RecordFailure()
	Reset()
	String() string
	OnSuccess(func())
	OnFailure(func())
	OnStateChange(func(name, from, to string))
}

type Counts struct {
	Requests             int64
	TotalSuccesses       int64
	TotalFailures        int64
	ConsecutiveSuccesses int64
	ConsecutiveFailures  int64
}

type Options struct {
	Name                string
	MaxRequests         int
	Interval            int
	Timeout             int
	FailureThreshold    float64
	SuccessThreshold    float64
	HalfOpenMaxRequests int
}

type circuitBreaker struct {
	name                string
	maxRequests         int32
	interval            int32
	timeout             int32
	failureThreshold    float64
	successThreshold    float64
	halfOpenMaxRequests int32

	mu sync.RWMutex

	state           State
	generation      int64
	counts          Counts
	expiry          int64

	onSuccess func()
	onFailure func()
	onStateChange func(name, from, to string)
}

func NewCircuitBreaker(opts Options) CircuitBreaker {
	cb := &circuitBreaker{
		name:                opts.Name,
		maxRequests:        int32(opts.MaxRequests),
		interval:           int32(opts.Interval),
		timeout:            int32(opts.Timeout),
		failureThreshold:  opts.FailureThreshold,
		successThreshold:   opts.SuccessThreshold,
		halfOpenMaxRequests: int32(opts.HalfOpenMaxRequests),
		state:              StateClosed,
	}

	if cb.maxRequests == 0 {
		cb.maxRequests = 5
	}
	if cb.interval == 0 {
		cb.interval = 10
	}
	if cb.timeout == 0 {
		cb.timeout = 60
	}
	if cb.failureThreshold == 0 {
		cb.failureThreshold = 0.5
	}
	if cb.successThreshold == 0 {
		cb.successThreshold = 2
	}
	if cb.halfOpenMaxRequests == 0 {
		cb.halfOpenMaxRequests = 3
	}

	return cb
}

func (cb *circuitBreaker) Execute(fn func() error) error {
	if !cb.CanAttempt() {
		return ErrOpenState
	}

	cb.incrementRequests()

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	if err != nil {
		cb.recordFailure(duration)
		return err
	}

	cb.recordSuccess(duration)
	return nil
}

func (cb *circuitBreaker) RecordSuccess() {
	cb.recordSuccess(0)
}

func (cb *circuitBreaker) RecordFailure() {
	cb.recordFailure(0)
}

func (cb *circuitBreaker) recordSuccess(duration time.Duration) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.counts.ConsecutiveFailures = 0
		cb.counts.ConsecutiveSuccesses++
		if cb.onSuccess != nil {
			cb.onSuccess()
		}

	case StateHalfOpen:
		cb.counts.ConsecutiveSuccesses++
		if cb.counts.ConsecutiveSuccesses >= int64(cb.successThreshold) {
			cb.transitionTo(StateClosed)
		}
	}

	cb.counts.TotalSuccesses++
}

func (cb *circuitBreaker) recordFailure(duration time.Duration) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.counts.ConsecutiveSuccesses = 0
		cb.counts.ConsecutiveFailures++
		if cb.onFailure != nil {
			cb.onFailure()
		}

		if cb.shouldTrip() {
			cb.transitionTo(StateOpen)
		}

	case StateHalfOpen:
		cb.transitionTo(StateOpen)
	}

	cb.counts.TotalFailures++
}

func (cb *circuitBreaker) shouldTrip() bool {
	if cb.failureThreshold <= 0 {
		return false
	}

	if cb.failureThreshold >= 1.0 {
		return cb.counts.ConsecutiveFailures >= 1
	}

	if cb.counts.Requests < int64(cb.maxRequests) {
		return false
	}

	failureRatio := float64(cb.counts.ConsecutiveFailures) / float64(cb.counts.Requests)
	return failureRatio >= cb.failureThreshold
}

func (cb *circuitBreaker) transitionTo(state State) {
	if cb.state == state {
		return
	}

	oldState := cb.state
	cb.state = state
	cb.generation++

	switch state {
	case StateClosed:
		cb.counts = Counts{}
		cb.expiry = 0

	case StateHalfOpen:
		cb.counts = Counts{}

	case StateOpen:
		cb.expiry = time.Now().Add(time.Duration(cb.timeout) * time.Second).Unix()
	}

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, oldState.String(), state.String())
	}
}

func (cb *circuitBreaker) CanAttempt() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true

	case StateHalfOpen:
		return cb.counts.Requests < int64(cb.halfOpenMaxRequests)

	case StateOpen:
		if cb.expiry == 0 {
			return true
		}
		if time.Now().Unix() >= cb.expiry {
			cb.mu.RUnlock()
			cb.mu.Lock()
			if cb.state == StateOpen {
				cb.transitionTo(StateHalfOpen)
			}
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	}

	return false
}

func (cb *circuitBreaker) incrementRequests() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateClosed {
		cb.counts.Requests++
	} else if cb.state == StateHalfOpen {
		cb.counts.Requests++
	}
}

func (cb *circuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *circuitBreaker) Counts() Counts {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.counts
}

func (cb *circuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transitionTo(StateClosed)
}

func (cb *circuitBreaker) String() string {
	return cb.name
}

func (cb *circuitBreaker) OnSuccess(fn func()) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onSuccess = fn
}

func (cb *circuitBreaker) OnFailure(fn func()) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onFailure = fn
}

func (cb *circuitBreaker) OnStateChange(fn func(name, from, to string)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onStateChange = fn
}

var (
	ErrOpenState = errors.New("circuit breaker is open")
)

type CircuitBreakerManager struct {
	breakers map[string]CircuitBreaker
	defaultOpts Options
	mu sync.RWMutex
}

func NewManager(defaultOpts Options) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]CircuitBreaker),
		defaultOpts: defaultOpts,
	}
}

func (m *CircuitBreakerManager) Get(name string) CircuitBreaker {
	m.mu.RLock()
	cb, exists := m.breakers[name]
	m.mu.RUnlock()

	if exists {
		return cb
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if cb, exists = m.breakers[name]; exists {
		return cb
	}

	opts := m.defaultOpts
	opts.Name = name
	cb = NewCircuitBreaker(opts)
	m.breakers[name] = cb
	return cb
}

func (m *CircuitBreakerManager) Register(name string, opts Options) CircuitBreaker {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.breakers[name]; exists {
		return m.breakers[name]
	}

	if opts.MaxRequests == 0 {
		opts.MaxRequests = m.defaultOpts.MaxRequests
	}
	if opts.Interval == 0 {
		opts.Interval = m.defaultOpts.Interval
	}
	if opts.Timeout == 0 {
		opts.Timeout = m.defaultOpts.Timeout
	}
	if opts.FailureThreshold == 0 {
		opts.FailureThreshold = m.defaultOpts.FailureThreshold
	}
	if opts.SuccessThreshold == 0 {
		opts.SuccessThreshold = m.defaultOpts.SuccessThreshold
	}
	if opts.HalfOpenMaxRequests == 0 {
		opts.HalfOpenMaxRequests = m.defaultOpts.HalfOpenMaxRequests
	}

	opts.Name = name
	cb := NewCircuitBreaker(opts)
	m.breakers[name] = cb
	return cb
}

func (m *CircuitBreakerManager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.breakers, name)
}

func (m *CircuitBreakerManager) ResetAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, cb := range m.breakers {
		cb.Reset()
	}
}

func (m *CircuitBreakerManager) GetAllStates() map[string]State {
	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make(map[string]State, len(m.breakers))
	for name, cb := range m.breakers {
		states[name] = cb.State()
	}
	return states
}

func (m *CircuitBreakerManager) GetAllCounts() map[string]Counts {
	m.mu.RLock()
	defer m.mu.RUnlock()

	counts := make(map[string]Counts, len(m.breakers))
	for name, cb := range m.breakers {
		counts[name] = cb.Counts()
	}
	return counts
}

type MetricsCollector struct {
	successCount  int64
	failureCount  int64
	openCount     int64
	halfOpenCount int64
	closedCount   int64
	mu            sync.Mutex
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

func (mc *MetricsCollector) RecordSuccess() {
	atomic.AddInt64(&mc.successCount, 1)
}

func (mc *MetricsCollector) RecordFailure() {
	atomic.AddInt64(&mc.failureCount, 1)
}

func (mc *MetricsCollector) RecordStateChange(from, to string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if to == "open" {
		atomic.AddInt64(&mc.openCount, 1)
	} else if to == "half_open" {
		atomic.AddInt64(&mc.halfOpenCount, 1)
	} else if to == "closed" {
		atomic.AddInt64(&mc.closedCount, 1)
	}
}

func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	return map[string]interface{}{
		"total_successes":  atomic.LoadInt64(&mc.successCount),
		"total_failures":   atomic.LoadInt64(&mc.failureCount),
		"total_opens":      atomic.LoadInt64(&mc.openCount),
		"total_half_opens": atomic.LoadInt64(&mc.halfOpenCount),
		"total_closes":     atomic.LoadInt64(&mc.closedCount),
	}
}
