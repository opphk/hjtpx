package performance

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type ServerlessArchitecture struct {
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	isRunning     bool
	functions     map[string]*ServerlessFunction
	invoker       *FunctionInvoker
	eventQueue    *EventQueue
	coldStartMgr  *ColdStartOptimizer
	metrics       *ServerlessMetrics
}

type ServerlessMetrics struct {
	TotalInvocations atomic.Int64
	ColdStarts      atomic.Int64
	WarmInvocations atomic.Int64
	AvgLatencyMs   atomic.Int64
	P99LatencyMs   atomic.Int64
	ErrorRate      atomic.Int64
	ActiveFunctions atomic.Int64
}

type ServerlessFunction struct {
	ID           string
	Name         string
	MemoryMB     int
	TimeoutSec   int
	Code         []byte
	Compiled     bool
	LastInvoked  time.Time
	ColdStartMs  int64
	Warm         bool
}

type FunctionInvoker struct {
	mu        sync.RWMutex
	workers   int
	queue     chan *Invocation
	pool      *FunctionPool
}

type FunctionPool struct {
	mu      sync.RWMutex
	functions map[string]*FunctionInstance
	maxSize  int
}

type FunctionInstance struct {
	FunctionID string
	Ready      bool
	Busy       atomic.Bool
}

type EventQueue struct {
	mu       sync.RWMutex
	events   []*Event
	maxSize  int
}

type Event struct {
	ID        string
	Type      string
	Payload   []byte
	Timestamp time.Time
}

func NewServerlessArchitecture() *ServerlessArchitecture {
	ctx, cancel := context.WithCancel(context.Background())

	return &ServerlessArchitecture{
		ctx:          ctx,
		cancel:       cancel,
		functions:    make(map[string]*ServerlessFunction),
		invoker:      NewFunctionInvoker(),
		eventQueue:   NewEventQueue(1000),
		coldStartMgr: NewColdStartOptimizer(),
		metrics:      &ServerlessMetrics{},
	}
}

func NewFunctionInvoker() *FunctionInvoker {
	return &FunctionInvoker{
		workers: 10,
		queue:   make(chan *Invocation, 1000),
		pool:    NewFunctionPool(100),
	}
}

func NewFunctionPool(maxSize int) *FunctionPool {
	return &FunctionPool{
		functions: make(map[string]*FunctionInstance),
		maxSize:  maxSize,
	}
}

func NewEventQueue(maxSize int) *EventQueue {
	return &EventQueue{
		events:  make([]*Event, 0, maxSize),
		maxSize: maxSize,
	}
}

func NewColdStartOptimizer() *ColdStartOptimizer {
	return &ColdStartOptimizer{
		enabled:       true,
		prewarmEnabled: true,
	}
}

func (s *ServerlessArchitecture) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return nil
	}

	s.isRunning = true

	go s.invoker.run(s.ctx)
	go s.eventQueue.process(s.ctx)

	log.Println("[ServerlessArchitecture] Started successfully")
	return nil
}

func (s *ServerlessArchitecture) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	s.cancel()
	s.isRunning = false
	log.Println("[ServerlessArchitecture] Stopped")
}

func (s *ServerlessArchitecture) DeployFunction(ctx context.Context, fn *ServerlessFunction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fn.ID = fmt.Sprintf("fn_%d", time.Now().UnixNano())
	fn.Compiled = false
	fn.ColdStartMs = 0

	s.functions[fn.ID] = fn
	s.metrics.ActiveFunctions.Add(1)

	log.Printf("[ServerlessArchitecture] Deployed function: %s", fn.Name)
	return nil
}

func (s *ServerlessArchitecture) InvokeFunction(ctx context.Context, fnID string, payload []byte) (*InvocationResult, error) {
	s.mu.RLock()
	fn, exists := s.functions[fnID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("function %s not found", fnID)
	}

	s.metrics.TotalInvocations.Add(1)

	start := time.Now()

	isColdStart := !fn.Warm
	if isColdStart {
		s.metrics.ColdStarts.Add(1)
		fn.ColdStartMs = time.Since(start).Milliseconds()
	} else {
		s.metrics.WarmInvocations.Add(1)
	}

	result := s.invoker.invoke(ctx, fn, payload)
	result.LatencyMs = time.Since(start).Milliseconds()
	result.IsColdStart = isColdStart

	fn.LastInvoked = time.Now()
	fn.Warm = true

	return result, nil
}

func (s *ServerlessArchitecture) PublishEvent(ctx context.Context, event *Event) error {
	return s.eventQueue.push(event)
}

func (s *ServerlessArchitecture) GetMetrics() map[string]interface{} {
	total := s.metrics.TotalInvocations.Load()
	coldStarts := s.metrics.ColdStarts.Load()

	var coldStartRate float64
	if total > 0 {
		coldStartRate = float64(coldStarts) / float64(total) * 100
	}

	return map[string]interface{}{
		"total_invocations": s.metrics.TotalInvocations.Load(),
		"cold_starts":      s.metrics.ColdStarts.Load(),
		"warm_invocations": s.metrics.WarmInvocations.Load(),
		"cold_start_rate":  coldStartRate,
		"avg_latency_ms":  s.metrics.AvgLatencyMs.Load(),
		"p99_latency_ms":  s.metrics.P99LatencyMs.Load(),
		"active_functions": s.metrics.ActiveFunctions.Load(),
	}
}

type ColdStartOptimizer struct {
	mu             sync.RWMutex
	enabled        bool
	prewarmEnabled bool
	prewarmDelay   time.Duration
}

type Invocation struct {
	FunctionID string
	Payload    []byte
	Result     chan *InvocationResult
}

type InvocationResult struct {
	FunctionID   string
	Payload      []byte
	LatencyMs    int64
	Error        error
	IsColdStart  bool
}

func (i *FunctionInvoker) invoke(ctx context.Context, fn *ServerlessFunction, payload []byte) *InvocationResult {
	start := time.Now()

	result := &InvocationResult{
		FunctionID: fn.ID,
		Payload:    []byte(fmt.Sprintf("executed:%s", string(payload))),
		LatencyMs:  time.Since(start).Milliseconds(),
	}

	return result
}

func (i *FunctionInvoker) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case inv := <-i.queue:
			go i.processInvocation(ctx, inv)
		}
	}
}

func (i *FunctionInvoker) processInvocation(ctx context.Context, inv *Invocation) {
	result := &InvocationResult{
		FunctionID: inv.FunctionID,
		Payload:    inv.Payload,
	}

	inv.Result <- result
}

func (q *EventQueue) push(event *Event) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.events) >= q.maxSize {
		return fmt.Errorf("event queue full")
	}

	q.events = append(q.events, event)
	return nil
}

func (q *EventQueue) process(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.mu.Lock()
			if len(q.events) > 0 {
				q.events = q.events[1:]
			}
			q.mu.Unlock()
		}
	}
}
