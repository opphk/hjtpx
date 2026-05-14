package circuitbreaker

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half_open"
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

type Config struct {
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
	HalfOpenMaxReqs  int
}

type DefaultConfig struct{}

func (c DefaultConfig) FailureThreshold() int {
	return 5
}

func (c DefaultConfig) SuccessThreshold() int {
	return 2
}

func (c DefaultConfig) Timeout() time.Duration {
	return 30 * time.Second
}

func (c DefaultConfig) HalfOpenMaxReqs() int {
	return 3
}

type CircuitBreaker interface {
	Allow() (bool, error)
	RecordSuccess()
	RecordFailure()
	State() State
	String() string
	Stats() Stats
}

type Stats struct {
	Requests         int64
	Successes        int64
	Failures         int64
	Rejects          int64
	State            State
	FailuresSnapshot int64
}

type circuitBreaker struct {
	name             string
	failureThreshold int
	successThreshold int
	timeout          time.Duration
	halfOpenMaxReqs  int

	state            atomic.Value
	failureCount     atomic.Int64
	successCount     atomic.Int64
	totalRequests    atomic.Int64
	totalSuccesses   atomic.Int64
	totalFailures    atomic.Int64
	totalRejects     atomic.Int64
	lastFailureTime  atomic.Int64
	halfOpenRequests atomic.Int64

	expiry     atomic.Int64
	mu         sync.RWMutex
	onStateChange func(name, from, to string)
}

func New(name string, config Config) CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 2
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.HalfOpenMaxReqs <= 0 {
		config.HalfOpenMaxReqs = 3
	}

	cb := &circuitBreaker{
		name:             name,
		failureThreshold: config.FailureThreshold,
		successThreshold: config.SuccessThreshold,
		timeout:          config.Timeout,
		halfOpenMaxReqs:  config.HalfOpenMaxReqs,
	}

	cb.state.Store(StateClosed)
	cb.expiry.Store(0)

	return cb
}

func (cb *circuitBreaker) Allow() (bool, error) {
	return cb.allowRequest()
}

func (cb *circuitBreaker) allowRequest() (bool, error) {
	state := cb.currentState()

	switch state {
	case StateClosed:
		return true, nil

	case StateOpen:
		if cb.isTimeoutElapsed() {
			cb.moveToHalfOpen()
			return cb.allowHalfOpenRequest()
		}
		cb.totalRejects.Add(1)
		return false, ErrCircuitOpen

	case StateHalfOpen:
		return cb.allowHalfOpenRequest()

	default:
		return false, fmt.Errorf("unknown state: %s", state)
	}
}

func (cb *circuitBreaker) allowHalfOpenRequest() (bool, error) {
	reqs := cb.halfOpenRequests.Add(1)
	if reqs > int64(cb.halfOpenMaxReqs) {
		return false, ErrCircuitOpen
	}
	return true, nil
}

func (cb *circuitBreaker) currentState() State {
	return cb.state.Load().(State)
}

func (cb *circuitBreaker) isTimeoutElapsed() bool {
	expiry := cb.expiry.Load()
	if expiry == 0 {
		return false
	}
	return time.Now().UnixNano() > expiry
}

func (cb *circuitBreaker) moveToHalfOpen() {
	cb.state.Store(StateHalfOpen)
	cb.successCount.Store(0)
	cb.halfOpenRequests.Store(0)

	expiry := time.Now().Add(cb.timeout).UnixNano()
	cb.expiry.Store(expiry)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, string(StateOpen), string(StateHalfOpen))
	}
}

func (cb *circuitBreaker) moveToClosed() {
	cb.state.Store(StateClosed)
	cb.failureCount.Store(0)
	cb.expiry.Store(0)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, string(StateHalfOpen), string(StateClosed))
	}
}

func (cb *circuitBreaker) moveToOpen() {
	cb.state.Store(StateOpen)
	cb.failureCount.Store(0)
	cb.lastFailureTime.Store(time.Now().UnixNano())

	expiry := time.Now().Add(cb.timeout).UnixNano()
	cb.expiry.Store(expiry)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, string(StateClosed), string(StateOpen))
	}
}

func (cb *circuitBreaker) RecordSuccess() {
	cb.totalRequests.Add(1)
	cb.totalSuccesses.Add(1)

	state := cb.currentState()

	switch state {
	case StateClosed:
		cb.failureCount.Store(0)

	case StateHalfOpen:
		successCount := cb.successCount.Add(1)
		if successCount >= int64(cb.successThreshold) {
			cb.moveToClosed()
		}

	case StateOpen:
	}
}

func (cb *circuitBreaker) RecordFailure() {
	cb.totalRequests.Add(1)
	cb.totalFailures.Add(1)

	state := cb.currentState()

	switch state {
	case StateClosed:
		failureCount := cb.failureCount.Add(1)
		if failureCount >= int64(cb.failureThreshold) {
			cb.moveToOpen()
		}

	case StateHalfOpen:
		cb.moveToOpen()

	case StateOpen:
	}
}

func (cb *circuitBreaker) State() State {
	return cb.currentState()
}

func (cb *circuitBreaker) String() string {
	return fmt.Sprintf("circuit_breaker{name=%s, state=%s}", cb.name, cb.currentState())
}

func (cb *circuitBreaker) Stats() Stats {
	return Stats{
		Requests:         cb.totalRequests.Load(),
		Successes:        cb.totalSuccesses.Load(),
		Failures:         cb.totalFailures.Load(),
		Rejects:          cb.totalRejects.Load(),
		State:            cb.currentState(),
		FailuresSnapshot: cb.failureCount.Load(),
	}
}

func (cb *circuitBreaker) OnStateChange(fn func(name, from, to string)) {
	cb.onStateChange = fn
}

type RateLimiter interface {
	Allow() bool
	AllowN(n int) bool
	Stats() RateLimiterStats
}

type RateLimiterStats struct {
	Allowed  int64
	Rejected int64
	Capacity int64
}

type tokenBucketLimiter struct {
	capacity      int64
	tokens        atomic.Int64
	refillRate    int64
	lastRefill    atomic.Int64
	mu            sync.Mutex
	allowedCount  atomic.Int64
	rejectedCount atomic.Int64
}

func NewTokenBucketLimiter(capacity int64, refillRate int64) RateLimiter {
	if capacity <= 0 {
		capacity = 100
	}
	if refillRate <= 0 {
		refillRate = 10
	}

	tb := &tokenBucketLimiter{
		capacity:   capacity,
		tokens:     atomic.Int64{},
		refillRate: refillRate,
		lastRefill: atomic.Int64{},
	}

	tb.tokens.Store(capacity)
	tb.lastRefill.Store(time.Now().UnixNano())

	return tb
}

func (tb *tokenBucketLimiter) Allow() bool {
	return tb.AllowN(1)
}

func (tb *tokenBucketLimiter) AllowN(n int) bool {
	tb.refill()

	for {
		currentTokens := tb.tokens.Load()
		if currentTokens < int64(n) {
			tb.rejectedCount.Add(1)
			return false
		}

		newTokens := currentTokens - int64(n)
		if tb.tokens.CompareAndSwap(currentTokens, newTokens) {
			tb.allowedCount.Add(1)
			return true
		}
	}
}

func (tb *tokenBucketLimiter) refill() {
	now := time.Now().UnixNano()
	lastRefill := tb.lastRefill.Load()

	elapsed := now - lastRefill
	if elapsed < 0 {
		return
	}

	elapsedSeconds := float64(elapsed) / float64(time.Second)
	tokensToAdd := int64(elapsedSeconds * float64(tb.refillRate))

	if tokensToAdd > 0 {
		for {
			currentTokens := tb.tokens.Load()
			newTokens := math.Min(float64(currentTokens)+float64(tokensToAdd), float64(tb.capacity))

			if tb.tokens.CompareAndSwap(currentTokens, int64(newTokens)) {
				tb.lastRefill.Store(now)
				break
			}
		}
	}
}

func (tb *tokenBucketLimiter) Stats() RateLimiterStats {
	return RateLimiterStats{
		Allowed:  tb.allowedCount.Load(),
		Rejected: tb.rejectedCount.Load(),
		Capacity: tb.capacity,
	}
}

type slidingWindowLimiter struct {
	windowSize  time.Duration
	buckets     int
	bucketData  []atomic.Int64
	bucketIndex atomic.Int64
	lastUpdate  atomic.Int64
	mu          sync.RWMutex
	allowed     atomic.Int64
	rejected    atomic.Int64
}

func NewSlidingWindowLimiter(windowSize time.Duration, maxRequests int) RateLimiter {
	if windowSize <= 0 {
		windowSize = time.Minute
	}
	if maxRequests <= 0 {
		maxRequests = 100
	}

	buckets := 10
	if windowSize < 10*time.Second {
		buckets = int(windowSize / time.Second)
		if buckets < 1 {
			buckets = 1
		}
	}

	sw := &slidingWindowLimiter{
		windowSize:  windowSize,
		buckets:     buckets,
		bucketData:  make([]atomic.Int64, buckets),
		bucketIndex: atomic.Int64{},
		lastUpdate:  atomic.Int64{},
	}

	sw.lastUpdate.Store(time.Now().UnixNano())

	return sw
}

func (sw *slidingWindowLimiter) Allow() bool {
	return sw.AllowN(1)
}

func (sw *slidingWindowLimiter) AllowN(n int) bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	bucketDuration := sw.windowSize / time.Duration(sw.buckets)
	if bucketDuration <= 0 {
		bucketDuration = time.Second
	}
	currentBucket := int(now.UnixNano() / int64(bucketDuration))
	currentBucket = currentBucket % sw.buckets

	sw.bucketIndex.Store(int64(currentBucket))

	elapsed := now.UnixNano() - sw.lastUpdate.Load()
	if elapsed > int64(sw.windowSize) {
		for i := range sw.bucketData {
			sw.bucketData[i].Store(0)
		}
		sw.lastUpdate.Store(now.UnixNano())
	}

	bucketToClear := (currentBucket + 1) % sw.buckets
	if int(sw.bucketIndex.Load()) != bucketToClear {
		sw.bucketData[bucketToClear].Store(0)
	}

	var total int64
	for i := range sw.bucketData {
		total += sw.bucketData[i].Load()
	}

	if total >= int64(n) {
		sw.rejected.Add(1)
		return false
	}

	sw.bucketData[currentBucket].Add(int64(n))
	sw.allowed.Add(1)

	return true
}

func (sw *slidingWindowLimiter) Stats() RateLimiterStats {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	var total int64
	for i := range sw.bucketData {
		total += sw.bucketData[i].Load()
	}

	return RateLimiterStats{
		Allowed:  sw.allowed.Load(),
		Rejected: sw.rejected.Load(),
		Capacity: total,
	}
}

type limiterRegistry struct {
	limiters map[string]RateLimiter
	mu       sync.RWMutex
}

var defaultRegistry = &limiterRegistry{
	limiters: make(map[string]RateLimiter),
}

func RegisterLimiter(name string, limiter RateLimiter) {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.limiters[name] = limiter
}

func GetLimiter(name string) (RateLimiter, bool) {
	defaultRegistry.mu.RLock()
	defer defaultRegistry.mu.RUnlock()
	limiter, ok := defaultRegistry.limiters[name]
	return limiter, ok
}

func GetOrCreateLimiter(name string, capacity int64, refillRate int64) RateLimiter {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()

	if limiter, ok := defaultRegistry.limiters[name]; ok {
		return limiter
	}

	limiter := NewTokenBucketLimiter(capacity, refillRate)
	defaultRegistry.limiters[name] = limiter
	return limiter
}

type AdaptiveRateLimiter struct {
	baseLimiter   RateLimiter
	maxCapacity   int64
	minCapacity   int64
	currentCap    atomic.Int64
	increaseRate  float64
	decreaseRate  float64
	mu            sync.RWMutex
	stats         AdaptiveStats
}

type AdaptiveStats struct {
	TotalAllowed  int64
	TotalRejected int64
	CurrentLimit  int64
}

func NewAdaptiveRateLimiter(initialCapacity int64, minCapacity int64, maxCapacity int64) *AdaptiveRateLimiter {
	if minCapacity <= 0 {
		minCapacity = 10
	}
	if maxCapacity <= initialCapacity {
		maxCapacity = initialCapacity * 10
	}

	ar := &AdaptiveRateLimiter{
		baseLimiter:  NewTokenBucketLimiter(initialCapacity, initialCapacity/10),
		maxCapacity:  maxCapacity,
		minCapacity:  minCapacity,
		currentCap:   atomic.Int64{},
		increaseRate: 1.1,
		decreaseRate: 0.9,
	}
	ar.currentCap.Store(initialCapacity)
	return ar
}

func (ar *AdaptiveRateLimiter) Allow() bool {
	allowed := ar.baseLimiter.Allow()

	ar.mu.Lock()
	if allowed {
		ar.stats.TotalAllowed++
	} else {
		ar.stats.TotalRejected++
	}
	ar.mu.Unlock()

	if allowed {
		ar.tryIncrease()
	} else {
		ar.tryDecrease()
	}

	return allowed
}

func (ar *AdaptiveRateLimiter) AllowN(n int) bool {
	allowed := ar.baseLimiter.AllowN(n)

	ar.mu.Lock()
	if allowed {
		ar.stats.TotalAllowed++
	} else {
		ar.stats.TotalRejected++
	}
	ar.mu.Unlock()

	return allowed
}

func (ar *AdaptiveRateLimiter) tryIncrease() {
	current := ar.currentCap.Load()
	if current >= ar.maxCapacity {
		return
	}

	if ar.stats.TotalAllowed > 100 && ar.stats.TotalRejected < 10 {
		newCapacity := int64(float64(current) * ar.increaseRate)
		if newCapacity > ar.maxCapacity {
			newCapacity = ar.maxCapacity
		}

		ar.currentCap.Store(newCapacity)
		ar.stats.CurrentLimit = newCapacity
	}
}

func (ar *AdaptiveRateLimiter) tryDecrease() {
	current := ar.currentCap.Load()
	if current <= ar.minCapacity {
		return
	}

	if ar.stats.TotalRejected > 5 {
		newCapacity := int64(float64(current) * ar.decreaseRate)
		if newCapacity < ar.minCapacity {
			newCapacity = ar.minCapacity
		}

		ar.currentCap.Store(newCapacity)
		ar.stats.CurrentLimit = newCapacity
	}
}

func (ar *AdaptiveRateLimiter) Stats() RateLimiterStats {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	return RateLimiterStats{
		Allowed:  ar.stats.TotalAllowed,
		Rejected: ar.stats.TotalRejected,
		Capacity: ar.currentCap.Load(),
	}
}
