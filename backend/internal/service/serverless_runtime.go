package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type RuntimeState int

const (
	RuntimeStateInitialized RuntimeState = iota
	RuntimeStateReady
	RuntimeStateRunning
	RuntimeStateBusy
	RuntimeStateError
	RuntimeStateStopping
	RuntimeStateStopped
)

func (s RuntimeState) String() string {
	switch s {
	case RuntimeStateInitialized:
		return "initialized"
	case RuntimeStateReady:
		return "ready"
	case RuntimeStateRunning:
		return "running"
	case RuntimeStateBusy:
		return "busy"
	case RuntimeStateError:
		return "error"
	case RuntimeStateStopping:
		return "stopping"
	case RuntimeStateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

type InvocationRequest struct {
	RequestID    string                 `json:"request_id"`
	FunctionName string                 `json:"function_name"`
	Payload      []byte                 `json:"payload"`
	Headers      map[string]string      `json:"headers"`
	Context      map[string]interface{} `json:"context"`
	Deadline     time.Time              `json:"deadline"`
}

type InvocationResponse struct {
	RequestID     string        `json:"request_id"`
	StatusCode    int           `json:"status_code"`
	Payload       []byte        `json:"payload"`
	Headers       map[string]string `json:"headers"`
	Latency       time.Duration `json:"latency"`
	BilledDuration time.Duration `json:"billed_duration"`
	MemoryUsed    int64         `json:"memory_used"`
	Error         string        `json:"error,omitempty"`
}

type RuntimeMetrics struct {
	TotalInvocations    atomic.Int64
	SuccessInvocations  atomic.Int64
	ErrorInvocations    atomic.Int64
	TotalLatency        atomic.Int64
	AvgLatency          atomic.Int64
	MaxLatency          atomic.Int64
	MinLatency          atomic.Int64
	TotalBilledDuration atomic.Int64
	AvgBilledDuration   atomic.Int64
	MaxMemoryUsed       atomic.Int64
	AvgMemoryUsed       atomic.Int64
	InitDuration        atomic.Int64
}

type FunctionHandler interface {
	Handle(ctx context.Context, req *InvocationRequest) (*InvocationResponse, error)
}

type ServerlessRuntime struct {
	functionName   string
	runtime        RuntimeType
	handler        FunctionHandler
	manager        *ServerlessManager
	metrics        *RuntimeMetrics
	state          atomic.Value
	envVars        map[string]string
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	initialized    atomic.Bool
	httpServer     *http.Server
	maxConcurrency int
}

type RuntimeConfig struct {
	FunctionName    string            `json:"function_name"`
	Runtime         RuntimeType       `json:"runtime"`
	Handler         string            `json:"handler"`
	Memory          MemorySize        `json:"memory"`
	Timeout         TimeoutDuration   `json:"timeout"`
	EnvironmentVars map[string]string `json:"environment_vars"`
	MaxConcurrency  int               `json:"max_concurrency"`
	PreloadEnabled  bool              `json:"preload_enabled"`
	KeepAlive       bool              `json:"keep_alive"`
}

type WarmInstance struct {
	InstanceID    string
	InitializedAt time.Time
	LastUsed      time.Time
	State         RuntimeState
	RequestCount   atomic.Int64
}

type InstancePool struct {
	functionName  string
	instances     map[string]*WarmInstance
	available     chan *WarmInstance
	mu            sync.RWMutex
	maxInstances  int
	minInstances  int
	currentCount  atomic.Int32
}

func NewServerlessRuntime(functionName string, runtimeType RuntimeType) *ServerlessRuntime {
	ctx, cancel := context.WithCancel(context.Background())
	
	rt := &ServerlessRuntime{
		functionName:   functionName,
		runtime:        runtimeType,
		manager:        NewServerlessManager(),
		metrics:        &RuntimeMetrics{},
		envVars:        make(map[string]string),
		ctx:            ctx,
		cancel:         cancel,
		maxConcurrency: 100,
	}
	
	rt.state.Store(RuntimeStateInitialized)
	
	return rt
}

func (rt *ServerlessRuntime) Initialize(config *RuntimeConfig) error {
	if !rt.initialized.CompareAndSwap(false, true) {
		return fmt.Errorf("runtime already initialized")
	}
	
	rt.functionName = config.FunctionName
	rt.runtime = config.Runtime
	
	for key, value := range config.EnvironmentVars {
		rt.envVars[key] = value
	}
	
	rt.state.Store(RuntimeStateReady)
	
	rt.metrics.InitDuration.Store(100 * 1e6)
	
	return nil
}

func (rt *ServerlessRuntime) Start() error {
	if rt.state.Load().(RuntimeState) != RuntimeStateReady {
		return fmt.Errorf("runtime not in ready state")
	}
	
	rt.state.Store(RuntimeStateRunning)
	
	go rt.runHealthCheck()
	
	go rt.collectMetrics()
	
	return nil
}

func (rt *ServerlessRuntime) Stop() error {
	if !rt.state.CompareAndSwap(RuntimeStateRunning, RuntimeStateStopping) {
		return fmt.Errorf("runtime not in running state")
	}
	
	rt.cancel()
	
	if rt.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		rt.httpServer.Shutdown(shutdownCtx)
	}
	
	rt.state.Store(RuntimeStateStopped)
	
	return nil
}

func (rt *ServerlessRuntime) Invoke(ctx context.Context, req *InvocationRequest) (*InvocationResponse, error) {
	if rt.state.Load().(RuntimeState) != RuntimeStateRunning && rt.state.Load().(RuntimeState) != RuntimeStateReady {
		return nil, fmt.Errorf("runtime not running")
	}
	
	rt.state.Store(RuntimeStateBusy)
	defer rt.state.Store(RuntimeStateRunning)
	
	start := time.Now()
	
	resp := &InvocationResponse{
		RequestID:  req.RequestID,
		StatusCode: 200,
		Headers:    make(map[string]string),
	}
	
	defer func() {
		resp.Latency = time.Since(start)
		resp.BilledDuration = calculateBilledDuration(resp.Latency)
		
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		resp.MemoryUsed = int64(m.Alloc)
		
		rt.metrics.TotalInvocations.Add(1)
		rt.metrics.TotalLatency.Add(resp.Latency.Nanoseconds())
		rt.metrics.TotalBilledDuration.Add(resp.BilledDuration.Nanoseconds())
		
		count := rt.metrics.SuccessInvocations.Load()
		rt.metrics.AvgLatency.Store((rt.metrics.TotalLatency.Load()) / (count + 1))
		
		if resp.StatusCode >= 400 {
			rt.metrics.ErrorInvocations.Add(1)
		} else {
			rt.metrics.SuccessInvocations.Add(1)
		}
		
		if resp.Latency.Nanoseconds() > rt.metrics.MaxLatency.Load() {
			rt.metrics.MaxLatency.Store(resp.Latency.Nanoseconds())
		}
		
		if rt.metrics.MinLatency.Load() == 0 || resp.Latency.Nanoseconds() < rt.metrics.MinLatency.Load() {
			rt.metrics.MinLatency.Store(resp.Latency.Nanoseconds())
		}
	}()
	
	if rt.handler != nil {
		result, err := rt.handler.Handle(ctx, req)
		if err != nil {
			resp.StatusCode = 500
			resp.Error = err.Error()
			return resp, err
		}
		
		resp.Payload = result.Payload
		resp.StatusCode = result.StatusCode
	}
	
	return resp, nil
}

func (rt *ServerlessRuntime) SetHandler(handler FunctionHandler) {
	rt.handler = handler
}

func (rt *ServerlessRuntime) GetState() RuntimeState {
	return rt.state.Load().(RuntimeState)
}

func (rt *ServerlessRuntime) GetMetrics() map[string]interface{} {
	successCount := rt.metrics.SuccessInvocations.Load()
	
	var avgLatency int64
	if successCount > 0 {
		avgLatency = rt.metrics.TotalLatency.Load() / successCount
	}
	
	var avgBilledDuration int64
	if successCount > 0 {
		avgBilledDuration = rt.metrics.TotalBilledDuration.Load() / successCount
	}
	
	return map[string]interface{}{
		"total_invocations":    rt.metrics.TotalInvocations.Load(),
		"success_invocations":  rt.metrics.SuccessInvocations.Load(),
		"error_invocations":    rt.metrics.ErrorInvocations.Load(),
		"avg_latency_ms":       float64(avgLatency) / 1e6,
		"max_latency_ms":       float64(rt.metrics.MaxLatency.Load()) / 1e6,
		"min_latency_ms":       float64(rt.metrics.MinLatency.Load()) / 1e6,
		"avg_billed_ms":        float64(avgBilledDuration) / 1e6,
		"init_duration_ms":    float64(rt.metrics.InitDuration.Load()) / 1e6,
		"state":                rt.state.Load().(RuntimeState).String(),
	}
}

func (rt *ServerlessRuntime) SetEnvironmentVariable(key, value string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.envVars[key] = value
}

func (rt *ServerlessRuntime) GetEnvironmentVariable(key string) (string, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	value, exists := rt.envVars[key]
	return value, exists
}

func (rt *ServerlessRuntime) runHealthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-rt.ctx.Done():
			return
		case <-ticker.C:
			if rt.state.Load().(RuntimeState) == RuntimeStateError {
				if rt.initialized.Load() {
					rt.state.Store(RuntimeStateReady)
				}
			}
		}
	}
}

func (rt *ServerlessRuntime) collectMetrics() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-rt.ctx.Done():
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			
			if int64(m.Alloc) > rt.metrics.MaxMemoryUsed.Load() {
				rt.metrics.MaxMemoryUsed.Store(int64(m.Alloc))
			}
			
			count := rt.metrics.TotalInvocations.Load()
			if count > 0 {
				rt.metrics.AvgMemoryUsed.Store(int64(m.Alloc))
			}
		}
	}
}

func calculateBilledDuration(latency time.Duration) time.Duration {
	billedMs := ((latency.Milliseconds() + 99) / 100) * 100
	
	if billedMs < 100 {
		billedMs = 100
	}
	
	return time.Duration(billedMs) * time.Millisecond
}

func NewInstancePool(functionName string, minInstances, maxInstances int) *InstancePool {
	pool := &InstancePool{
		functionName: functionName,
		instances:    make(map[string]*WarmInstance),
		available:    make(chan *WarmInstance, maxInstances),
		maxInstances: maxInstances,
		minInstances: minInstances,
	}
	
	for i := 0; i < minInstances; i++ {
		instance := &WarmInstance{
			InstanceID:    fmt.Sprintf("%s-%d", functionName, i),
			InitializedAt: time.Now(),
			LastUsed:      time.Now(),
			State:         RuntimeStateReady,
		}
		instance.State = RuntimeStateReady
		pool.instances[instance.InstanceID] = instance
		pool.available <- instance
		pool.currentCount.Add(1)
	}
	
	return pool
}

func (p *InstancePool) Acquire(ctx context.Context) (*WarmInstance, error) {
	select {
	case instance := <-p.available:
		instance.LastUsed = time.Now()
		instance.RequestCount.Add(1)
		return instance, nil
	default:
		if p.currentCount.Load() < int32(p.maxInstances) {
			return p.createInstance()
		}
		
		select {
		case instance := <-p.available:
			instance.LastUsed = time.Now()
			instance.RequestCount.Add(1)
			return instance, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			return nil, fmt.Errorf("timeout acquiring instance")
		}
	}
}

func (p *InstancePool) Release(instance *WarmInstance) {
	instance.State = RuntimeStateReady
	p.available <- instance
}

func (p *InstancePool) createInstance() (*WarmInstance, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.currentCount.Load() >= int32(p.maxInstances) {
		return nil, fmt.Errorf("max instances reached")
	}
	
	instance := &WarmInstance{
		InstanceID:    fmt.Sprintf("%s-%d", p.functionName, time.Now().UnixNano()),
		InitializedAt: time.Now(),
		LastUsed:      time.Now(),
	}
	instance.State = RuntimeStateReady
	
	p.instances[instance.InstanceID] = instance
	p.currentCount.Add(1)
	instance.RequestCount.Add(1)
	
	return instance, nil
}

func (p *InstancePool) ScaleUp(count int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	for i := 0; i < count && int(p.currentCount.Load()) < p.maxInstances; i++ {
		instance := &WarmInstance{
			InstanceID:    fmt.Sprintf("%s-%d", p.functionName, time.Now().UnixNano()),
			InitializedAt: time.Now(),
			LastUsed:      time.Now(),
		}
		instance.State = RuntimeStateReady
		p.instances[instance.InstanceID] = instance
		p.available <- instance
		p.currentCount.Add(1)
	}
}

func (p *InstancePool) ScaleDown(count int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	for i := 0; i < count && int(p.currentCount.Load()) > p.minInstances; i++ {
		select {
		case instance := <-p.available:
			delete(p.instances, instance.InstanceID)
			p.currentCount.Add(-1)
		default:
			break
		}
	}
}

func (p *InstancePool) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"current_instances": p.currentCount.Load(),
		"available_instances": len(p.available),
		"max_instances": p.maxInstances,
		"min_instances": p.minInstances,
	}
}

type DefaultHandler struct {
	functionName string
}

func (h *DefaultHandler) Handle(ctx context.Context, req *InvocationRequest) (*InvocationResponse, error) {
	return &InvocationResponse{
		RequestID:  req.RequestID,
		StatusCode: 200,
		Payload:    []byte(fmt.Sprintf("Hello from %s", h.functionName)),
	}, nil
}

func (rt *ServerlessRuntime) StartHTTP(addr string) error {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/invoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		
		req := &InvocationRequest{
			RequestID:    fmt.Sprintf("req-%d", time.Now().UnixNano()),
			FunctionName: rt.functionName,
			Payload:      body,
			Headers:      make(map[string]string),
			Deadline:     time.Now().Add(30 * time.Second),
		}
		
		for key, values := range r.Header {
			if len(values) > 0 {
				req.Headers[key] = values[0]
			}
		}
		
		resp, err := rt.Invoke(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		for key, value := range resp.Headers {
			w.Header().Set(key, value)
		}
		
		w.WriteHeader(resp.StatusCode)
		w.Write(resp.Payload)
	})
	
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"healthy","state":"%s"}`, rt.GetState())
	})
	
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics := rt.GetMetrics()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%v", metrics)
	})
	
	rt.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	
	return rt.httpServer.ListenAndServe()
}

func (rt *ServerlessRuntime) RegisterMiddleware(middleware func(http.Handler) http.Handler) {
	// Middleware registration placeholder
}

func CreateRuntimeHandler(functionName string) *ServerlessRuntime {
	runtimeType := GetDefaultRuntime()
	
	rt := NewServerlessRuntime(functionName, runtimeType)
	
	handler := &DefaultHandler{functionName: functionName}
	rt.SetHandler(handler)
	
	return rt
}
