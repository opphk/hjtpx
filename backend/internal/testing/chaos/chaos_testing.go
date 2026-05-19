package chaos

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"
)

var ErrNetDown = errors.New("network is down")

type ChaosExperiment struct {
	Name        string
	Description string
	Setup       func() error
	Run         func(context.Context) error
	Teardown    func() error
}

type ChaosEngine struct {
	experiments []*ChaosExperiment
	results     map[string]*ExperimentResult
	mu          sync.RWMutex
}

type ExperimentResult struct {
	Success bool
	Error   error
	Metrics map[string]interface{}
	Start   time.Time
	End     time.Time
}

func NewChaosEngine() *ChaosEngine {
	return &ChaosEngine{
		experiments: make([]*ChaosExperiment, 0),
		results:     make(map[string]*ExperimentResult),
	}
}

func (ce *ChaosEngine) AddExperiment(exp *ChaosExperiment) {
	ce.experiments = append(ce.experiments, exp)
}

func (ce *ChaosEngine) RunExperiment(ctx context.Context, name string) (*ExperimentResult, error) {
	ce.mu.Lock()
	var exp *ChaosExperiment
	for _, e := range ce.experiments {
		if e.Name == name {
			exp = e
			break
		}
	}
	ce.mu.Unlock()

	if exp == nil {
		return nil, fmt.Errorf("experiment %q not found", name)
	}

	result := &ExperimentResult{
		Start:   time.Now(),
		Metrics: make(map[string]interface{}),
	}

	defer func() {
		result.End = time.Now()
		ce.mu.Lock()
		ce.results[name] = result
		ce.mu.Unlock()
	}()

	if exp.Setup != nil {
		if err := exp.Setup(); err != nil {
			result.Error = fmt.Errorf("setup failed: %w", err)
			return result, result.Error
		}
	}

	if exp.Teardown != nil {
		defer exp.Teardown()
	}

	if err := exp.Run(ctx); err != nil {
		result.Error = err
		return result, err
	}

	result.Success = true
	return result, nil
}

func (ce *ChaosEngine) RunAll(ctx context.Context) map[string]*ExperimentResult {
	for _, exp := range ce.experiments {
		select {
		case <-ctx.Done():
			return ce.results
		default:
			ce.RunExperiment(ctx, exp.Name)
		}
	}
	return ce.results
}

type LatencyInjector struct {
	Delay   time.Duration
	Jitter  time.Duration
	Enabled bool
}

func NewLatencyInjector(delay, jitter time.Duration) *LatencyInjector {
	return &LatencyInjector{
		Delay:   delay,
		Jitter:  jitter,
		Enabled: true,
	}
}

func (li *LatencyInjector) Inject() {
	if !li.Enabled {
		return
	}
	delay := li.Delay
	if li.Jitter > 0 {
		jitter := time.Duration(rand.Int63n(int64(li.Jitter)))
		if rand.Float32() < 0.5 {
			delay += jitter
		} else {
			delay -= jitter
		}
		if delay < 0 {
			delay = 0
		}
	}
	time.Sleep(delay)
}

type FaultInjector struct {
	ErrorRate float64
	Errors    []error
}

func NewFaultInjector(errorRate float64, errs ...error) *FaultInjector {
	return &FaultInjector{
		ErrorRate: errorRate,
		Errors:    errs,
	}
}

func (fi *FaultInjector) Inject() error {
	if rand.Float64() < fi.ErrorRate && len(fi.Errors) > 0 {
		return fi.Errors[rand.Intn(len(fi.Errors))]
	}
	return nil
}

type NetworkPartition struct {
	targets []string
	blocked bool
}

func NewNetworkPartition(targets []string) *NetworkPartition {
	return &NetworkPartition{
		targets: targets,
		blocked: false,
	}
}

func (np *NetworkPartition) Block() {
	np.blocked = true
}

func (np *NetworkPartition) Unblock() {
	np.blocked = false
}

func (np *NetworkPartition) IsBlocked(addr string) bool {
	if !np.blocked {
		return false
	}
	for _, target := range np.targets {
		if target == addr {
			return true
		}
	}
	return false
}

type ResourceExhaustion struct {
	memoryUsed int64
	maxMemory  int64
}

func NewResourceExhaustion(maxMemory int64) *ResourceExhaustion {
	return &ResourceExhaustion{
		maxMemory: maxMemory,
	}
}

func (re *ResourceExhaustion) ConsumeMemory(amount int64) error {
	if re.memoryUsed+amount > re.maxMemory {
		return errors.New("out of memory")
	}
	re.memoryUsed += amount
	return nil
}

func (re *ResourceExhaustion) ReleaseMemory(amount int64) {
	re.memoryUsed -= amount
	if re.memoryUsed < 0 {
		re.memoryUsed = 0
	}
}

type DiskFailure struct {
	ioErrors bool
}

func NewDiskFailure() *DiskFailure {
	return &DiskFailure{
		ioErrors: false,
	}
}

func (df *DiskFailure) EnableIOErrors() {
	df.ioErrors = true
}

func (df *DiskFailure) DisableIOErrors() {
	df.ioErrors = false
}

func (df *DiskFailure) Read() error {
	if df.ioErrors {
		return &net.OpError{Op: "read", Err: errors.New("disk I/O error")}
	}
	return nil
}

func RunChaosExperiments(t *testing.T) {
	t.Run("LatencyInjection", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		engine := NewChaosEngine()
		engine.AddExperiment(&ChaosExperiment{
			Name:        "API-Latency",
			Description: "Inject latency into API calls",
			Setup: func() error {
				return nil
			},
			Run: func(ctx context.Context) error {
				injector := NewLatencyInjector(100*time.Millisecond, 50*time.Millisecond)
				start := time.Now()
				injector.Inject()
				duration := time.Since(start)
				if duration < 50*time.Millisecond {
					return fmt.Errorf("expected latency, got %v", duration)
				}
				return nil
			},
			Teardown: func() error {
				return nil
			},
		})

		result, err := engine.RunExperiment(ctx, "API-Latency")
		assertNoError(t, err, "Chaos experiment should not fail")
		assertTrue(t, result.Success, "Experiment should succeed")
	})

	t.Run("FaultInjection", func(t *testing.T) {
		injector := NewFaultInjector(0.5, errors.New("database connection failed"), errors.New("timeout"))
		successCount := 0
		errorCount := 0

		for i := 0; i < 100; i++ {
			err := injector.Inject()
			if err != nil {
				errorCount++
			} else {
				successCount++
			}
		}

		assertGreater(t, errorCount, 0, "Should have some errors")
		assertGreater(t, successCount, 0, "Should have some successes")
	})
}

func assertNoError(t *testing.T, err error, msg string, args ...interface{}) {
	t.Helper()
	if err != nil {
		t.Errorf(msg+": %v", append(args, err)...)
	}
}

func assertTrue(t *testing.T, condition bool, msg string, args ...interface{}) {
	t.Helper()
	if !condition {
		t.Errorf(msg, args...)
	}
}

func assertGreater(t *testing.T, a, b int, msg string, args ...interface{}) {
	t.Helper()
	if a <= b {
		t.Errorf(msg, args...)
	}
}
