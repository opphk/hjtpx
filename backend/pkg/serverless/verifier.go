package serverless

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type ServerlessVerifier struct {
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	isRunning    bool

	functions   map[string]*Function
	instances   *InstancePool
	scheduler   *EventDrivenScheduler
	autoScaler  *AutoScaler
	coldStart   *ColdStartOptimizer
	stats       *ServerlessStats
}

type Function struct {
	ID          string
	Name        string
	Runtime     string
	MemoryMB    int
	Timeout     time.Duration
	Concurrency int
	Code        []byte
	Version     string
	EnvVars     map[string]string
	CreatedAt   time.Time
	InvokeCount int64
	ErrorCount  int64
}

type InstancePool struct {
	mu       sync.RWMutex
	instances map[string]*FunctionInstance
	poolSize int
	maxSize  int
}

type FunctionInstance struct {
	ID         string
	FunctionID string
	Status     string
	MemoryMB   int
	CPUWeight  int
	ActiveReqs int32
	Ready      bool
	StartedAt  time.Time
	LastUsed   time.Time
	InvokeTime time.Duration
}

type EventDrivenScheduler struct {
	mu          sync.RWMutex
	eventQueue  chan *FunctionEvent
	rules       map[string]*RoutingRule
}

type FunctionEvent struct {
	EventID   string
	FunctionID string
	Payload   []byte
	Timestamp time.Time
	Priority  int
	Source    string
}

type RoutingRule struct {
	FunctionID string
	Condition string
	Weight    int
	Priority  int
}

type AutoScaler struct {
	mu          sync.RWMutex
	enabled     bool
	minInstances int
	maxInstances int
	targetCPU   int
	scaleUpCount int32
	scaleDownCount int32
	metrics     *ScalingMetrics
}

type ScalingMetrics struct {
	CPUUsage     float64
	MemoryUsage  float64
	RequestRate  float64
	QueueLength  int
	ActiveConns  int
}

type ColdStartOptimizer struct {
	mu            sync.RWMutex
	preWarmed     map[string]bool
	predictionModel *PredictionModel
	prewarmCount  int32
	optimizations []string
}

type PredictionModel struct {
	enabled bool
	model   interface{}
}

type ServerlessStats struct {
	TotalInvocations   atomic.Int64
	SuccessfulInvocations atomic.Int64
	FailedInvocations  atomic.Int64
	ColdStarts         atomic.Int64
	WarmInvocations    atomic.Int64
	AvgLatencyNanos    atomic.Int64
	P99LatencyNanos   atomic.Int64
	ActiveInstances   atomic.Int64
	TotalInstances    atomic.Int64
	ColdStartMs       atomic.Int64
	InstanceReuse     atomic.Int64
	ScaleUpEvents     atomic.Int64
	ScaleDownEvents   atomic.Int64
	LastUpdate        atomic.Value
}

type VerificationRequest struct {
	RequestID  string
	FunctionID string
	Payload    []byte
	Context    map[string]interface{}
	Headers    map[string]string
}

type VerificationResponse struct {
	Success        bool
	RequestID      string
	FunctionID     string
	Result         []byte
	InstanceID     string
	Latency        time.Duration
	IsWarm         bool
	Error          string
}

type FunctionConfig struct {
	Name         string
	Runtime      string
	MemoryMB     int
	Timeout      time.Duration
	Concurrency  int
	MinInstances int
	MaxInstances int
}

type ScalingPolicy struct {
	MetricType    string
	TargetValue   float64
	ScaleUpCooldown time.Duration
	ScaleDownCooldown time.Duration
}

type ColdStartConfig struct {
	PrewarmEnabled  bool
	MinPreWarmed    int
	PredictivePrewarm bool
}

const (
	InstanceStatusCold = "cold"
	InstanceStatusWarming = "warming"
	InstanceStatusReady = "ready"
	InstanceStatusBusy = "busy"
	InstanceStatusIdle = "idle"
	InstanceStatusStopping = "stopping"

	RuntimeGo = "go"
	RuntimeNodeJS = "nodejs"
	RuntimePython = "python"
)

func NewServerlessVerifier() *ServerlessVerifier {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ServerlessVerifier{
		ctx:          ctx,
		cancel:       cancel,
		functions:    make(map[string]*Function),
		instances:   NewInstancePool(100, 1000),
		scheduler:   NewEventDrivenScheduler(),
		autoScaler:  NewAutoScaler(1, 100),
		coldStart:   NewColdStartOptimizer(),
		stats:       &ServerlessStats{},
	}
}

func NewInstancePool(initialSize, maxSize int) *InstancePool {
	return &InstancePool{
		instances: make(map[string]*FunctionInstance),
		poolSize: initialSize,
		maxSize:  maxSize,
	}
}

func NewEventDrivenScheduler() *EventDrivenScheduler {
	return &EventDrivenScheduler{
		eventQueue: make(chan *FunctionEvent, 10000),
		rules:     make(map[string]*RoutingRule),
	}
}

func NewAutoScaler(minInstances, maxInstances int) *AutoScaler {
	return &AutoScaler{
		enabled:       true,
		minInstances:  minInstances,
		maxInstances:  maxInstances,
		targetCPU:     70,
		metrics:       &ScalingMetrics{},
	}
}

func NewColdStartOptimizer() *ColdStartOptimizer {
	return &ColdStartOptimizer{
		preWarmed:     make(map[string]bool),
		optimizations: []string{"container_reuse", "lazy_loading", "code_caching"},
	}
}

func (s *ServerlessVerifier) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return nil
	}

	s.isRunning = true

	go s.scheduler.runEventProcessor(s.ctx, s)
	go s.autoScaler.runScaler(s.ctx, s)
	go s.coldStart.runPrewarmer(s.ctx, s)
	go s.statsCollector()

	log.Println("[ServerlessVerifier] Initialized successfully")
	return nil
}

func (s *ServerlessVerifier) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	s.isRunning = false
	s.cancel()

	log.Println("[ServerlessVerifier] Shutdown complete")
	return nil
}

func (s *ServerlessVerifier) RegisterFunction(ctx context.Context, config *FunctionConfig) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	functionID := generateFunctionID()

	fn := &Function{
		ID:          functionID,
		Name:        config.Name,
		Runtime:     config.Runtime,
		MemoryMB:    config.MemoryMB,
		Timeout:     config.Timeout,
		Concurrency: config.Concurrency,
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
	}

	s.functions[functionID] = fn

	if s.autoScaler.minInstances < config.MinInstances {
		s.autoScaler.minInstances = config.MinInstances
	}
	if s.autoScaler.maxInstances > config.MaxInstances {
		s.autoScaler.maxInstances = config.MaxInstances
	}

	log.Printf("[ServerlessVerifier] Registered function: %s", functionID)
	return functionID, nil
}

func (s *ServerlessVerifier) InvokeFunction(ctx context.Context, req *VerificationRequest) (*VerificationResponse, error) {
	s.stats.TotalInvocations.Add(1)
	start := time.Now()

	fn := s.getFunction(req.FunctionID)
	if fn == nil {
		s.stats.FailedInvocations.Add(1)
		return nil, fmt.Errorf("function %s not found", req.FunctionID)
	}

	instance, isWarm, err := s.getAvailableInstance(ctx, fn)
	if err != nil {
		s.stats.FailedInvocations.Add(1)
		return &VerificationResponse{
			Success:    false,
			RequestID:  req.RequestID,
			FunctionID: req.FunctionID,
			Error:     err.Error(),
			Latency:   time.Since(start),
		}, err
	}

	atomic.AddInt32(&instance.ActiveReqs, 1)
	defer atomic.AddInt32(&instance.ActiveReqs, -1)

	result, err := s.executeFunction(ctx, instance, req)
	
	instance.LastUsed = time.Now()

	if err != nil {
		fn.ErrorCount++
		s.stats.FailedInvocations.Add(1)
	} else {
		fn.InvokeCount++
		s.stats.SuccessfulInvocations.Add(1)
	}

	latency := time.Since(start)

	if !isWarm {
		s.stats.ColdStarts.Add(1)
		s.stats.ColdStartMs.Store(latency.Milliseconds())
	} else {
		s.stats.WarmInvocations.Add(1)
		s.stats.InstanceReuse.Add(1)
	}

	avgLatency := atomic.LoadInt64(&s.stats.AvgLatencyNanos)
	newAvg := (avgLatency + latency.Nanoseconds()) / 2
	atomic.StoreInt64(&s.stats.AvgLatencyNanos, newAvg)

	return &VerificationResponse{
		Success:    true,
		RequestID:  req.RequestID,
		FunctionID:  req.FunctionID,
		InstanceID: instance.ID,
		Result:     result,
		Latency:    latency,
		IsWarm:     isWarm,
	}, nil
}

func (s *ServerlessVerifier) getFunction(functionID string) *Function {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.functions[functionID]
}

func (s *ServerlessVerifier) getAvailableInstance(ctx context.Context, fn *Function) (*FunctionInstance, bool, error) {
	s.instances.mu.RLock()
	
	var warmInstance *FunctionInstance
	var coldInstance *FunctionInstance

	for _, inst := range s.instances.instances {
		if inst.FunctionID != fn.ID {
			continue
		}

		if inst.Status == InstanceStatusReady && atomic.LoadInt32(&inst.ActiveReqs) < int32(fn.Concurrency) {
			warmInstance = inst
			break
		}

		if inst.Status == InstanceStatusIdle {
			coldInstance = inst
		}
	}
	s.instances.mu.RUnlock()

	if warmInstance != nil {
		warmInstance.Status = InstanceStatusBusy
		return warmInstance, true, nil
	}

	if coldInstance != nil {
		coldInstance.Status = InstanceStatusBusy
		return coldInstance, false, nil
	}

	instance := s.createInstance(ctx, fn)
	return instance, false, nil
}

func (s *ServerlessVerifier) createInstance(ctx context.Context, fn *Function) *FunctionInstance {
	instanceID := generateInstanceID()

	instance := &FunctionInstance{
		ID:         instanceID,
		FunctionID: fn.ID,
		Status:     InstanceStatusWarming,
		MemoryMB:   fn.MemoryMB,
		CPUWeight:  fn.MemoryMB / 128,
		Ready:      false,
		StartedAt:  time.Now(),
		LastUsed:   time.Now(),
	}

	s.instances.mu.Lock()
	if s.instances.poolSize < s.instances.maxSize {
		s.instances.instances[instanceID] = instance
		s.instances.poolSize++
		s.stats.TotalInstances.Add(1)
	}
	s.instances.mu.Unlock()

	go func() {
		time.Sleep(50 * time.Millisecond)
		instance.Status = InstanceStatusReady
		instance.Ready = true
	}()

	return instance
}

func (s *ServerlessVerifier) executeFunction(ctx context.Context, instance *FunctionInstance, req *VerificationRequest) ([]byte, error) {
	invokeStart := time.Now()
	
	time.Sleep(5 * time.Millisecond)
	
	instance.InvokeTime = time.Since(invokeStart)

	instance.Status = InstanceStatusReady

	return req.Payload, nil
}

func (s *ServerlessVerifier) PublishEvent(ctx context.Context, event *FunctionEvent) error {
	select {
	case s.scheduler.eventQueue <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("event queue full")
	}
}

func (s *ServerlessVerifier) SetScalingPolicy(ctx context.Context, policy *ScalingPolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.autoScaler.targetCPU = int(policy.TargetValue)
}

func (s *ServerlessVerifier) SetColdStartConfig(ctx context.Context, config *ColdStartConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.coldStart.preWarmed = make(map[string]bool)
	if config.PrewarmEnabled {
		for i := 0; i < config.MinPreWarmed; i++ {
			s.coldStart.prewarmCount++
		}
	}
}

func (s *ServerlessVerifier) GetActiveInstances() int {
	s.instances.mu.RLock()
	defer s.instances.mu.RUnlock()

	count := 0
	for _, inst := range s.instances.instances {
		if inst.Status == InstanceStatusReady || inst.Status == InstanceStatusBusy {
			count++
		}
	}
	return count
}

func (s *ServerlessVerifier) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_invocations":   s.stats.TotalInvocations.Load(),
		"successful_invocations": s.stats.SuccessfulInvocations.Load(),
		"failed_invocations":   s.stats.FailedInvocations.Load(),
		"cold_starts":          s.stats.ColdStarts.Load(),
		"warm_invocations":     s.stats.WarmInvocations.Load(),
		"avg_latency_ns":      s.stats.AvgLatencyNanos.Load(),
		"p99_latency_ns":     s.stats.P99LatencyNanos.Load(),
		"active_instances":    s.stats.ActiveInstances.Load(),
		"total_instances":     s.stats.TotalInstances.Load(),
		"cold_start_ms":       s.stats.ColdStartMs.Load(),
		"instance_reuse":      s.stats.InstanceReuse.Load(),
		"scale_up_events":     s.stats.ScaleUpEvents.Load(),
		"scale_down_events":   s.stats.ScaleDownEvents.Load(),
		"last_update":         s.stats.LastUpdate.Load(),
	}
}

func (s *ServerlessVerifier) statsCollector() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.collectStats()
		}
	}
}

func (s *ServerlessVerifier) collectStats() {
	activeCount := int64(0)
	s.instances.mu.RLock()
	for _, inst := range s.instances.instances {
		if inst.Status == InstanceStatusReady || inst.Status == InstanceStatusBusy {
			activeCount++
		}
	}
	s.instances.mu.RUnlock()

	s.stats.ActiveInstances.Store(activeCount)
	s.stats.LastUpdate.Store(time.Now())
}

func (sched *EventDrivenScheduler) runEventProcessor(ctx context.Context, s *ServerlessVerifier) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-sched.eventQueue:
			go s.processEvent(ctx, event)
		}
	}
}

func (s *ServerlessVerifier) processEvent(ctx context.Context, event *FunctionEvent) {
	req := &VerificationRequest{
		RequestID:  event.EventID,
		FunctionID: event.FunctionID,
		Payload:    event.Payload,
	}

	s.InvokeFunction(ctx, req)
}

func (as *AutoScaler) runScaler(ctx context.Context, s *ServerlessVerifier) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if as.enabled {
				as.evaluateScaling(s)
			}
		}
	}
}

func (as *AutoScaler) evaluateScaling(s *ServerlessVerifier) {
	as.mu.Lock()
	defer as.mu.Unlock()

	activeInstances := int(s.stats.ActiveInstances.Load())
	
	if as.metrics.CPUUsage > float64(as.targetCPU) && activeInstances < as.maxInstances {
		atomic.AddInt32(&as.scaleUpCount, 1)
		s.stats.ScaleUpEvents.Add(1)
	}

	if as.metrics.CPUUsage < float64(as.targetCPU)/2 && activeInstances > as.minInstances {
		atomic.AddInt32(&as.scaleDownCount, 1)
		s.stats.ScaleDownEvents.Add(1)
	}
}

func (cso *ColdStartOptimizer) runPrewarmer(ctx context.Context, s *ServerlessVerifier) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if cso.predictionModel != nil && cso.predictionModel.enabled {
				cso.predictivePrewarm(ctx, s)
			}
		}
	}
}

func (cso *ColdStartOptimizer) predictivePrewarm(ctx context.Context, s *ServerlessVerifier) {
	cso.mu.Lock()
	defer cso.mu.Unlock()

	prewarmCount := atomic.LoadInt32(&cso.prewarmCount)
	if prewarmCount < 5 {
		atomic.AddInt32(&cso.prewarmCount, 1)
	}
}

func generateFunctionID() string {
	return fmt.Sprintf("func_%d", time.Now().UnixNano())
}

func generateInstanceID() string {
	return fmt.Sprintf("inst_%d", time.Now().UnixNano())
}
