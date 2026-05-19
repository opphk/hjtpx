package service

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type FunctionState int

const (
	FunctionStatePending FunctionState = iota
	FunctionStateDeploying
	FunctionStateRunning
	FunctionStateScaling
	FunctionStateError
	FunctionStateStopped
	FunctionStateUpdating
)

func (s FunctionState) String() string {
	switch s {
	case FunctionStatePending:
		return "pending"
	case FunctionStateDeploying:
		return "deploying"
	case FunctionStateRunning:
		return "running"
	case FunctionStateScaling:
		return "scaling"
	case FunctionStateError:
		return "error"
	case FunctionStateStopped:
		return "stopped"
	case FunctionStateUpdating:
		return "updating"
	default:
		return "unknown"
	}
}

type RuntimeType string

const (
	RuntimeGo116       RuntimeType = "go1.16"
	RuntimeGo118       RuntimeType = "go1.18"
	RuntimeGo120       RuntimeType = "go1.20"
	RuntimeGo122       RuntimeType = "go1.22"
	RuntimeGo124       RuntimeType = "go1.24"
	NodeJS16           RuntimeType = "nodejs16.x"
	NodeJS18           RuntimeType = "nodejs18.x"
	NodeJS20           RuntimeType = "nodejs20.x"
	Python39           RuntimeType = "python3.9"
	Python310          RuntimeType = "python3.10"
	Python311          RuntimeType = "python3.11"
	Python312          RuntimeType = "python3.12"
)

type MemorySize int

const (
	Memory128MB MemorySize = 128
	Memory256MB MemorySize = 256
	Memory512MB MemorySize = 512
	Memory1024MB MemorySize = 1024
	Memory2048MB MemorySize = 2048
	Memory4096MB MemorySize = 4096
)

func (m MemorySize) ToMiB() int {
	return int(m) * 1024 * 1024
}

type TimeoutDuration int

const (
	Timeout3s   TimeoutDuration = 3
	Timeout10s  TimeoutDuration = 10
	Timeout30s  TimeoutDuration = 30
	Timeout60s  TimeoutDuration = 60
	Timeout300s  TimeoutDuration = 300
	Timeout600s  TimeoutDuration = 600
)

func (t TimeoutDuration) Seconds() int {
	return int(t)
}

type TriggerType string

const (
	TriggerHTTP     TriggerType = "http"
	TriggerTimer    TriggerType = "timer"
	TriggerQueue    TriggerType = "queue"
	TriggerEvent    TriggerType = "event"
	TriggerS3       TriggerType = "s3"
	TriggerKafka     TriggerType = "kafka"
	TriggerCron      TriggerType = "cron"
	TriggerWebSocket TriggerType = "websocket"
)

type FunctionConfig struct {
	FunctionName    string        `json:"function_name"`
	Runtime         RuntimeType   `json:"runtime"`
	Memory          MemorySize    `json:"memory"`
	Timeout         TimeoutDuration `json:"timeout"`
	Handler         string        `json:"handler"`
	SourceCode      string        `json:"source_code,omitempty"`
	EnvironmentVars map[string]string `json:"environment_vars"`
	Triggers        []TriggerConfig `json:"triggers"`
	Dependencies    []string      `json:"dependencies"`
	MaxInstances    int           `json:"max_instances"`
	MinInstances    int           `json:"min_instances"`
	Concurrency     int           `json:"concurrency"`
	VPCConfig       *VPCConfig    `json:"vpc_config,omitempty"`
	Layers          []string      `json:"layers"`
	ARM64           bool          `json:"arm64"`
}

type VPCConfig struct {
	SubnetIDs        []string `json:"subnet_ids"`
	SecurityGroupIDs []string `json:"security_group_ids"`
	EnablePublicIP   bool     `json:"enable_public_ip"`
}

type TriggerConfig struct {
	TriggerType   TriggerType `json:"trigger_type"`
	TriggerName   string      `json:"trigger_name"`
	Enabled       bool        `json:"enabled"`
	Configuration interface{} `json:"configuration"`
	FilterPattern string      `json:"filter_pattern,omitempty"`
	BatchSize     int         `json:"batch_size,omitempty"`
	Parallelism   int         `json:"parallelism,omitempty"`
}

type FunctionMetadata struct {
	FunctionName     string            `json:"function_name"`
	FunctionARN      string            `json:"function_arn"`
	Runtime          RuntimeType       `json:"runtime"`
	Memory           MemorySize        `json:"memory"`
	Timeout          TimeoutDuration   `json:"timeout"`
	Handler          string            `json:"handler"`
	State            FunctionState     `json:"state"`
	LastModified     time.Time         `json:"last_modified"`
	CreatedAt        time.Time         `json:"created_at"`
	Version          string            `json:"version"`
	InvokeCount      atomic.Int64      `json:"invoke_count"`
	ErrorCount       atomic.Int64      `json:"error_count"`
	TotalLatency     atomic.Int64      `json:"total_latency_ns"`
	AvgLatency       atomic.Int64      `json:"avg_latency_ns"`
	MaxLatency       atomic.Int64      `json:"max_latency_ns"`
	MinLatency       atomic.Int64      `json:"min_latency_ns"`
	Instances        atomic.Int32      `json:"instances"`
	MaxInstances     int               `json:"max_instances"`
	MinInstances     int               `json:"min_instances"`
	ARM64            bool              `json:"arm64"`
	CostPerGBSecond  float64           `json:"cost_per_gb_second"`
	Invocations      []InvocationRecord `json:"invocations,omitempty"`
}

type InvocationRecord struct {
	Timestamp     time.Time `json:"timestamp"`
	Duration      int64     `json:"duration_ns"`
	MemoryUsed    int64     `json:"memory_used"`
	BilledDuration int64    `json:"billed_duration"`
	InitDuration  int64     `json:"init_duration"`
	StatusCode    int       `json:"status_code"`
	RequestID     string    `json:"request_id"`
}

type FunctionEvent struct {
	Type         string                 `json:"type"`
	FunctionName string                 `json:"function_name"`
	Timestamp    time.Time              `json:"timestamp"`
	Data         map[string]interface{} `json:"data"`
}

type ServerlessManager struct {
	functions      map[string]*FunctionMetadata
	configs       map[string]*FunctionConfig
	states        map[string]FunctionState
	deployments   map[string]*DeploymentInfo
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	metrics       *serverlessMetrics
	eventHandlers []func(*FunctionEvent)
}

type serverlessMetrics struct {
	TotalFunctions      atomic.Int64
	RunningFunctions    atomic.Int64
	TotalInvocations    atomic.Int64
	TotalErrors         atomic.Int64
	AvgColdStartTime    atomic.Int64
	AvgWarmLatency      atomic.Int64
	TotalCost           float64
	TotalCostMu         sync.Mutex
	ActiveConnections   atomic.Int64
}

type DeploymentInfo struct {
	DeploymentID    string            `json:"deployment_id"`
	FunctionName    string            `json:"function_name"`
	Status          string            `json:"status"`
	StartedAt       time.Time         `json:"started_at"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
	ErrorMessage    string            `json:"error_message,omitempty"`
	Logs            []string          `json:"logs"`
	Artifacts       map[string]string `json:"artifacts"`
}

func NewServerlessManager() *ServerlessManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &ServerlessManager{
		functions:    make(map[string]*FunctionMetadata),
		configs:      make(map[string]*FunctionConfig),
		states:       make(map[string]FunctionState),
		deployments:  make(map[string]*DeploymentInfo),
		ctx:          ctx,
		cancel:       cancel,
		metrics:      &serverlessMetrics{},
	}
	
	return manager
}

func (m *ServerlessManager) RegisterFunction(config *FunctionConfig) error {
	if config == nil {
		return fmt.Errorf("function config cannot be nil")
	}
	
	if config.FunctionName == "" {
		return fmt.Errorf("function name is required")
	}
	
	if config.Runtime == "" {
		return fmt.Errorf("runtime is required")
	}
	
	if config.Handler == "" {
		return fmt.Errorf("handler is required")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.functions[config.FunctionName]; exists {
		return fmt.Errorf("function %s already registered", config.FunctionName)
	}
	
	defaults := m.applyDefaults(config)
	
	metadata := &FunctionMetadata{
		FunctionName:    config.FunctionName,
		FunctionARN:      fmt.Sprintf("arn:serverless:default:%s:%s", config.FunctionName, generateARN()),
		Runtime:          defaults.Runtime,
		Memory:           defaults.Memory,
		Timeout:          defaults.Timeout,
		Handler:          defaults.Handler,
		State:            FunctionStatePending,
		CreatedAt:        time.Now(),
		LastModified:     time.Now(),
		Version:          "1.0.0",
		MaxInstances:     defaults.MaxInstances,
		MinInstances:     defaults.MinInstances,
		ARM64:            defaults.ARM64,
		Invocations:      make([]InvocationRecord, 0),
	}
	
	m.functions[config.FunctionName] = metadata
	m.configs[config.FunctionName] = defaults
	m.states[config.FunctionName] = FunctionStatePending
	
	m.metrics.TotalFunctions.Add(1)
	
	m.emitEvent(&FunctionEvent{
		Type:         "FunctionRegistered",
		FunctionName: config.FunctionName,
		Timestamp:    time.Now(),
		Data:         map[string]interface{}{"runtime": config.Runtime},
	})
	
	return nil
}

func (m *ServerlessManager) applyDefaults(config *FunctionConfig) *FunctionConfig {
	if config.Memory == 0 {
		config.Memory = Memory256MB
	}
	
	if config.Timeout == 0 {
		config.Timeout = Timeout30s
	}
	
	if config.MaxInstances == 0 {
		config.MaxInstances = 100
	}
	
	if config.MinInstances == 0 {
		config.MinInstances = 0
	}
	
	if config.Concurrency == 0 {
		config.Concurrency = 1
	}
	
	if config.EnvironmentVars == nil {
		config.EnvironmentVars = make(map[string]string)
	}
	
	if config.Triggers == nil {
		config.Triggers = []TriggerConfig{}
	}
	
	if config.Dependencies == nil {
		config.Dependencies = []string{}
	}
	
	if config.Layers == nil {
		config.Layers = []string{}
	}
	
	return config
}

func (m *ServerlessManager) GetFunction(name string) (*FunctionMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	metadata, exists := m.functions[name]
	if !exists {
		return nil, fmt.Errorf("function %s not found", name)
	}
	
	return metadata, nil
}

func (m *ServerlessManager) ListFunctions() []*FunctionMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make([]*FunctionMetadata, 0, len(m.functions))
	for _, fn := range m.functions {
		result = append(result, fn)
	}
	
	return result
}

func (m *ServerlessManager) UpdateFunction(name string, config *FunctionConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	metadata, exists := m.functions[name]
	if !exists {
		return fmt.Errorf("function %s not found", name)
	}
	
	m.states[name] = FunctionStateUpdating
	
	metadata.Runtime = config.Runtime
	metadata.Memory = config.Memory
	metadata.Timeout = config.Timeout
	metadata.Handler = config.Handler
	metadata.MaxInstances = config.MaxInstances
	metadata.MinInstances = config.MinInstances
	metadata.ARM64 = config.ARM64
	metadata.LastModified = time.Now()
	
	m.configs[name] = config
	
	m.states[name] = FunctionStateRunning
	
	m.emitEvent(&FunctionEvent{
		Type:         "FunctionUpdated",
		FunctionName: name,
		Timestamp:    time.Now(),
		Data:         map[string]interface{}{"version": metadata.Version},
	})
	
	return nil
}

func (m *ServerlessManager) DeleteFunction(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.functions[name]; !exists {
		return fmt.Errorf("function %s not found", name)
	}
	
	delete(m.functions, name)
	delete(m.configs, name)
	delete(m.states, name)
	
	m.metrics.TotalFunctions.Add(-1)
	
	m.emitEvent(&FunctionEvent{
		Type:         "FunctionDeleted",
		FunctionName: name,
		Timestamp:    time.Now(),
		Data:         map[string]interface{}{},
	})
	
	return nil
}

func (m *ServerlessManager) SetFunctionState(name string, state FunctionState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.functions[name]; !exists {
		return fmt.Errorf("function %s not found", name)
	}
	
	m.states[name] = state
	m.functions[name].State = state
	m.functions[name].LastModified = time.Now()
	
	m.emitEvent(&FunctionEvent{
		Type:         "StateChanged",
		FunctionName: name,
		Timestamp:    time.Now(),
		Data:         map[string]interface{}{"state": state.String()},
	})
	
	return nil
}

func (m *ServerlessManager) GetFunctionState(name string) (FunctionState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	state, exists := m.states[name]
	if !exists {
		return FunctionStatePending, fmt.Errorf("function %s not found", name)
	}
	
	return state, nil
}

func (m *ServerlessManager) RecordInvocation(name string, record *InvocationRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	metadata, exists := m.functions[name]
	if !exists {
		return fmt.Errorf("function %s not found", name)
	}
	
	metadata.InvokeCount.Add(1)
	metadata.TotalLatency.Add(record.Duration)
	
	invokeCount := metadata.InvokeCount.Load()
	if invokeCount > 0 {
		metadata.AvgLatency.Store(metadata.TotalLatency.Load() / invokeCount)
	}
	
	if currentMax := metadata.MaxLatency.Load(); record.Duration > currentMax {
		metadata.MaxLatency.Store(record.Duration)
	}
	
	if currentMin := metadata.MinLatency.Load(); currentMin == 0 || record.Duration < currentMin {
		metadata.MinLatency.Store(record.Duration)
	}
	
	if record.StatusCode >= 400 {
		metadata.ErrorCount.Add(1)
		m.metrics.TotalErrors.Add(1)
	}
	
	if len(metadata.Invocations) < 1000 {
		metadata.Invocations = append(metadata.Invocations, *record)
	}
	
	m.metrics.TotalInvocations.Add(1)
	
	return nil
}

func (m *ServerlessManager) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_functions":      m.metrics.TotalFunctions.Load(),
		"running_functions":   m.metrics.RunningFunctions.Load(),
		"total_invocations":    m.metrics.TotalInvocations.Load(),
		"total_errors":         m.metrics.TotalErrors.Load(),
		"avg_cold_start_ms":    m.metrics.AvgColdStartTime.Load() / 1e6,
		"avg_warm_latency_ms":  m.metrics.AvgWarmLatency.Load() / 1e6,
		"total_cost_usd":       m.metrics.TotalCost,
		"active_connections":   m.metrics.ActiveConnections.Load(),
	}
}

func (m *ServerlessManager) GetFunctionMetrics(name string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	metadata, exists := m.functions[name]
	if !exists {
		return nil, fmt.Errorf("function %s not found", name)
	}
	
	invokeCount := metadata.InvokeCount.Load()
	var avgLatency int64
	if invokeCount > 0 {
		avgLatency = metadata.AvgLatency.Load()
	}
	
	return map[string]interface{}{
		"function_name":       metadata.FunctionName,
		"state":               metadata.State.String(),
		"invoke_count":        invokeCount,
		"error_count":         metadata.ErrorCount.Load(),
		"avg_latency_ns":      avgLatency,
		"max_latency_ns":      metadata.MaxLatency.Load(),
		"min_latency_ns":      metadata.MinLatency.Load(),
		"instances":           metadata.Instances.Load(),
		"runtime":             metadata.Runtime,
		"memory_mb":           metadata.Memory,
		"version":             metadata.Version,
	}, nil
}

func (m *ServerlessManager) CreateDeployment(name string) (*DeploymentInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.functions[name]; !exists {
		return nil, fmt.Errorf("function %s not found", name)
	}
	
	deploymentID := generateARN()
	
	deployment := &DeploymentInfo{
		DeploymentID: deploymentID,
		FunctionName: name,
		Status:       "in_progress",
		StartedAt:    time.Now(),
		Logs:         []string{},
		Artifacts:    make(map[string]string),
	}
	
	m.deployments[deploymentID] = deployment
	m.states[name] = FunctionStateDeploying
	
	return deployment, nil
}

func (m *ServerlessManager) GetDeployment(deploymentID string) (*DeploymentInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	deployment, exists := m.deployments[deploymentID]
	if !exists {
		return nil, fmt.Errorf("deployment %s not found", deploymentID)
	}
	
	return deployment, nil
}

func (m *ServerlessManager) UpdateDeployment(deploymentID string, status string, logs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	deployment, exists := m.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment %s not found", deploymentID)
	}
	
	deployment.Status = status
	if logs != nil {
		deployment.Logs = append(deployment.Logs, logs...)
	}
	
	if status == "completed" || status == "failed" {
		now := time.Now()
		deployment.CompletedAt = &now
		
		if status == "completed" {
			m.states[deployment.FunctionName] = FunctionStateRunning
			m.metrics.RunningFunctions.Add(1)
		}
	}
	
	return nil
}

func (m *ServerlessManager) AddEventHandler(handler func(*FunctionEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.eventHandlers = append(m.eventHandlers, handler)
}

func (m *ServerlessManager) emitEvent(event *FunctionEvent) {
	for _, handler := range m.eventHandlers {
		go handler(event)
	}
}

func (m *ServerlessManager) GetConfig(name string) (*FunctionConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	config, exists := m.configs[name]
	if !exists {
		return nil, fmt.Errorf("function %s not found", name)
	}
	
	return config, nil
}

func (m *ServerlessManager) SetEnvironmentVariable(name, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	config, exists := m.configs[name]
	if !exists {
		return fmt.Errorf("function %s not found", name)
	}
	
	if config.EnvironmentVars == nil {
		config.EnvironmentVars = make(map[string]string)
	}
	
	config.EnvironmentVars[key] = value
	
	return nil
}

func (m *ServerlessManager) GetEnvironmentVariable(name, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	config, exists := m.configs[name]
	if !exists {
		return "", fmt.Errorf("function %s not found", name)
	}
	
	value, exists := config.EnvironmentVars[key]
	if !exists {
		return "", fmt.Errorf("environment variable %s not found", key)
	}
	
	return value, nil
}

func (m *ServerlessManager) CalculateCost(name string) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	metadata, exists := m.functions[name]
	if !exists {
		return 0, fmt.Errorf("function %s not found", name)
	}
	
	config, exists := m.configs[name]
	if !exists {
		return 0, fmt.Errorf("function %s config not found", name)
	}
	
	invokeCount := metadata.InvokeCount.Load()
	if invokeCount == 0 {
		return 0, nil
	}
	
	avgLatencyMs := float64(metadata.AvgLatency.Load()) / 1e6
	memoryMb := float64(config.Memory) / 1024.0
	
	gbSeconds := (avgLatencyMs / 1000.0) * memoryMb * float64(invokeCount)
	
	costPerGBSecond := 0.00001667
	
	cost := gbSeconds * costPerGBSecond
	
	return cost, nil
}

func (m *ServerlessManager) SetARM64(name string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	metadata, exists := m.functions[name]
	if !exists {
		return fmt.Errorf("function %s not found", name)
	}
	
	metadata.ARM64 = enabled
	metadata.LastModified = time.Now()
	
	return nil
}

func (m *ServerlessManager) GetLogs(name string, limit int) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	metadata, exists := m.functions[name]
	if !exists {
		return nil, fmt.Errorf("function %s not found", name)
	}
	
	logs := make([]string, 0)
	for _, inv := range metadata.Invocations {
		log := fmt.Sprintf("[%s] RequestID: %s, Duration: %dms, Status: %d",
			inv.Timestamp.Format(time.RFC3339),
			inv.RequestID,
			inv.Duration/1e6,
			inv.StatusCode,
		)
		logs = append(logs, log)
		
		if limit > 0 && len(logs) >= limit {
			break
		}
	}
	
	return logs, nil
}

func (m *ServerlessManager) ExportConfig(name string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	config, exists := m.configs[name]
	if !exists {
		return "", fmt.Errorf("function %s not found", name)
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}
	
	return string(data), nil
}

func (m *ServerlessManager) ImportConfig(configJSON string) error {
	var config FunctionConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return m.RegisterFunction(&config)
}

func (m *ServerlessManager) Stop() {
	m.cancel()
}

func generateARN() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomString(16))
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

func GetDefaultRuntime() RuntimeType {
	version := runtime.Version()
	switch {
	case version >= "go1.24":
		return RuntimeGo124
	case version >= "go1.22":
		return RuntimeGo122
	case version >= "go1.20":
		return RuntimeGo120
	case version >= "go1.18":
		return RuntimeGo118
	default:
		return RuntimeGo116
	}
}
