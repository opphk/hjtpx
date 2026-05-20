package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type ChaosType string

const (
	ChaosNetwork    ChaosType = "network"
	ChaosLatency    ChaosType = "latency"
	ChaosPacketLoss ChaosType = "packet_loss"
	ChaosTimeout    ChaosType = "timeout"
	ChaosMemory     ChaosType = "memory"
	ChaosCPU        ChaosType = "cpu"
	ChaosKill       ChaosType = "kill"
)

type ChaosConfig struct {
	Type           ChaosType
	Intensity      float64
	Duration       time.Duration
	TargetServices []string
	Enabled        bool
}

type ChaosMetrics struct {
	mu                sync.RWMutex
	ExperimentsRun    uint64
	ExperimentsPassed uint64
	ExperimentsFailed uint64
	TotalDowntime    time.Duration
	Errors           []ChaosError
}

type ChaosError struct {
	Time       time.Time
	Experiment string
	Step       string
	Error      string
	Severity   string
}

type ExperimentResult struct {
	Name     string
	StartTime time.Time
	EndTime   time.Time
	Status    string
	Duration  time.Duration
	Errors    []string
}

func NewChaosMetrics() *ChaosMetrics {
	return &ChaosMetrics{
		Errors: make([]ChaosError, 0),
	}
}

type NetworkChaosEngine struct {
	config  *ChaosConfig
	metrics *ChaosMetrics
	active  bool
	stopCh  chan struct{}
	mu      sync.RWMutex
}

func NewNetworkChaosEngine(config *ChaosConfig) *NetworkChaosEngine {
	return &NetworkChaosEngine{
		config:  config,
		metrics: NewChaosMetrics(),
		active:  false,
		stopCh:  make(chan struct{}),
	}
}

func (e *NetworkChaosEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.active {
		return fmt.Errorf("engine already running")
	}

	e.active = true
	e.stopCh = make(chan struct{})

	go e.runNetworkChaos(ctx)

	return nil
}

func (e *NetworkChaosEngine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.active {
		return fmt.Errorf("engine not running")
	}

	close(e.stopCh)
	e.active = false

	return nil
}

func (e *NetworkChaosEngine) runNetworkChaos(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			if e.shouldInjectChaos() {
				e.injectNetworkIssue()
			}
		}
	}
}

func (e *NetworkChaosEngine) shouldInjectChaos() bool {
	if e.config == nil {
		return false
	}

	r := rand.Float64()
	return r < e.config.Intensity
}

func (e *NetworkChaosEngine) injectNetworkIssue() {
	if e.config == nil {
		return
	}

	atomic.AddUint64(&e.metrics.ExperimentsRun, 1)

	switch e.config.Type {
	case ChaosNetwork:
		e.injectNetworkPartition()
	case ChaosLatency:
		e.injectLatency()
	case ChaosPacketLoss:
		e.injectPacketLoss()
	case ChaosTimeout:
		e.injectTimeout()
	}
}

func (e *NetworkChaosEngine) injectNetworkPartition() {
	time.Sleep(time.Duration(rand.Intn(100)+50) * time.Millisecond)
}

func (e *NetworkChaosEngine) injectLatency() {
	latency := time.Duration(e.config.Intensity*100) * time.Millisecond
	time.Sleep(latency)
}

func (e *NetworkChaosEngine) injectPacketLoss() {
	time.Sleep(time.Duration(rand.Intn(50)+10) * time.Millisecond)
}

func (e *NetworkChaosEngine) injectTimeout() {
	time.Sleep(5 * time.Second)
}

func (e *NetworkChaosEngine) GetMetrics() *ChaosMetrics {
	return e.metrics
}

func RunChaosTests() map[string]*ExperimentResult {
	results := make(map[string]*ExperimentResult)

	experiments := []struct {
		name string
		fn   func() error
	}{
		{"network_partition", func() error {
			config := &ChaosConfig{Type: ChaosLatency, Intensity: 0.8, Duration: 30 * time.Second}
			engine := NewNetworkChaosEngine(config)
			return engine.Start(context.Background())
		}},
	}

	ctx := context.Background()

	for _, exp := range experiments {
		start := time.Now()
		err := exp.fn()
		duration := time.Since(start)

		result := &ExperimentResult{
			Name:      exp.name,
			StartTime: start,
			EndTime:   time.Now(),
			Duration:  duration,
		}

		if err != nil {
			result.Status = "error"
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.Status = "passed"
		}

		results[exp.name] = result
	}

	return results
}
