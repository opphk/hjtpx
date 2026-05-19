package circuitbreaker

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/go-kratos/aegis/circuitbreaker"
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

type CircuitBreaker struct {
	cb          circuitbreaker.CircuitBreaker
	name        string
	fallback    func(error) error
	mu          sync.RWMutex
	lastTrip    time.Time
	tripCount   int64
	successCount int64
	failureCount int64
}

type Config struct {
	Name          string
	MaxRequests   int           // 熔断器半开状态下允许的最大请求数
	Interval      time.Duration // 统计间隔
	Timeout       time.Duration // 熔断器打开时间
	ReadyToTrip   func(counts circuitbreaker.Counts) bool
	Fallback      func(error) error
}

func NewCircuitBreaker(config *Config) *CircuitBreaker {
	if config.MaxRequests == 0 {
		config.MaxRequests = 10
	}
	if config.Interval == 0 {
		config.Interval = 5 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.ReadyToTrip == nil {
		config.ReadyToTrip = func(counts circuitbreaker.Counts) bool {
			failureRatio := float64(counts.Failures) / float64(counts.Requests)
			return counts.Requests >= 10 && failureRatio >= 0.5
		}
	}

	cb := circuitbreaker.NewCircuitBreaker(
		circuitbreaker.WithMaxRequests(config.MaxRequests),
		circuitbreaker.WithInterval(config.Interval),
		circuitbreaker.WithTimeout(config.Timeout),
		circuitbreaker.WithReadyToTrip(config.ReadyToTrip),
	)

	return &CircuitBreaker{
		cb:       cb,
		name:     config.Name,
		fallback: config.Fallback,
	}
}

func (c *CircuitBreaker) Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	c.mu.Lock()
	state := c.cb.State()
	c.mu.Unlock()

	if state == circuitbreaker.StateOpen {
		c.mu.Lock()
		c.tripCount++
		c.mu.Unlock()
		
		log.Printf("[CircuitBreaker] %s is OPEN, rejecting request", c.name)
		
		if c.fallback != nil {
			return nil, c.fallback(ErrCircuitOpen)
		}
		return nil, ErrCircuitOpen
	}

	result, err := c.cb.Allow()
	if err != nil {
		c.mu.Lock()
		c.tripCount++
		c.mu.Unlock()
		
		log.Printf("[CircuitBreaker] %s rejected request: %v", c.name, err)
		
		if c.fallback != nil {
			return nil, c.fallback(err)
		}
		return nil, err
	}

	defer func() {
		if err != nil {
			result.Fail()
			c.mu.Lock()
			c.failureCount++
			c.mu.Unlock()
			log.Printf("[CircuitBreaker] %s request failed: %v", c.name, err)
		} else {
			result.Success()
			c.mu.Lock()
			c.successCount++
			c.mu.Unlock()
		}
	}()

	return fn()
}

func (c *CircuitBreaker) State() circuitbreaker.State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cb.State()
}

func (c *CircuitBreaker) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cb.Reset()
	c.tripCount = 0
	c.successCount = 0
	c.failureCount = 0
	c.lastTrip = time.Time{}
	log.Printf("[CircuitBreaker] %s reset", c.name)
}

func (c *CircuitBreaker) GetStats() *Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return &Stats{
		Name:          c.name,
		State:         c.cb.State().String(),
		LastTrip:      c.lastTrip,
		TripCount:     c.tripCount,
		SuccessCount:  c.successCount,
		FailureCount:  c.failureCount,
	}
}

type Stats struct {
	Name         string
	State        string
	LastTrip     time.Time
	TripCount    int64
	SuccessCount int64
	FailureCount int64
}

type CircuitBreakerManager struct {
	breakers sync.Map
}

func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{}
}

func (m *CircuitBreakerManager) GetOrCreate(name string, config *Config) *CircuitBreaker {
	if existing, ok := m.breakers.Load(name); ok {
		return existing.(*CircuitBreaker)
	}

	if config == nil {
		config = &Config{
			Name: name,
		}
	} else {
		config.Name = name
	}

	cb := NewCircuitBreaker(config)
	if existing, loaded := m.breakers.LoadOrStore(name, cb); loaded {
		return existing.(*CircuitBreaker)
	}
	return cb
}

func (m *CircuitBreakerManager) Get(name string) (*CircuitBreaker, bool) {
	breaker, ok := m.breakers.Load(name)
	if !ok {
		return nil, false
	}
	return breaker.(*CircuitBreaker), true
}

func (m *CircuitBreakerManager) Remove(name string) {
	m.breakers.Delete(name)
}

func (m *CircuitBreakerManager) List() []*Stats {
	var stats []*Stats
	m.breakers.Range(func(key, value interface{}) bool {
		cb := value.(*CircuitBreaker)
		stats = append(stats, cb.GetStats())
		return true
	})
	return stats
}

func (m *CircuitBreakerManager) ResetAll() {
	m.breakers.Range(func(key, value interface{}) bool {
		cb := value.(*CircuitBreaker)
		cb.Reset()
		return true
	})
}