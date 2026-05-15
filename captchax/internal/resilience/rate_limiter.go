package resilience

import (
	"sync"
	"time"
)

type TokenBucket struct {
	capacity   int64
	tokens     int64
	refillRate int64
	lastRefill time.Time
	mu         sync.Mutex
}

func NewTokenBucket(capacity, refillRate int64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (b *TokenBucket) Allow() bool {
	return b.AllowN(1)
}

func (b *TokenBucket) AllowN(n int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill()

	if b.tokens >= n {
		b.tokens -= n
		return true
	}

	return false
}

func (b *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)

	tokensToAdd := int64(elapsed.Seconds()) * b.refillRate
	if tokensToAdd > 0 {
		b.tokens = min(b.capacity, b.tokens+tokensToAdd)
		b.lastRefill = now
	}
}

func (b *TokenBucket) GetTokens() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refill()
	return b.tokens
}

type SlidingWindowLimiter struct {
	windowSize  time.Duration
	maxRequests  int64
	requests    []time.Time
	mu          sync.Mutex
}

func NewSlidingWindowLimiter(windowSize time.Duration, maxRequests int64) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		requests:    make([]time.Time, 0),
	}
}

func (l *SlidingWindowLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.windowSize)

	var validRequests []time.Time
	for _, t := range l.requests {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}
	l.requests = validRequests

	if int64(len(l.requests)) >= l.maxRequests {
		return false
	}

	l.requests = append(l.requests, now)
	return true
}

func (l *SlidingWindowLimiter) GetCount() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.windowSize)

	count := int64(0)
	for _, t := range l.requests {
		if t.After(cutoff) {
			count++
		}
	}
	return count
}

type AdaptiveRateLimiter struct {
	tokenBucket     *TokenBucket
	slidingWindow   *SlidingWindowLimiter
	enabled         bool
	mu              sync.RWMutex
}

func NewAdaptiveRateLimiter(requestsPerSec, burstSize int64) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		tokenBucket:   NewTokenBucket(burstSize, requestsPerSec),
		slidingWindow: NewSlidingWindowLimiter(1*time.Second, requestsPerSec),
		enabled:       true,
	}
}

func (a *AdaptiveRateLimiter) Allow() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.enabled {
		return true
	}

	return a.tokenBucket.Allow() && a.slidingWindow.Allow()
}

func (a *AdaptiveRateLimiter) Enable() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enabled = true
}

func (a *AdaptiveRateLimiter) Disable() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enabled = false
}

func (a *AdaptiveRateLimiter) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}
