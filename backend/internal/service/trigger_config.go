package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type TriggerState int

const (
	TriggerStateInactive TriggerState = iota
	TriggerStateActive
	TriggerStateError
	TriggerStatePaused
)

func (s TriggerState) String() string {
	switch s {
	case TriggerStateInactive:
		return "inactive"
	case TriggerStateActive:
		return "active"
	case TriggerStateError:
		return "error"
	case TriggerStatePaused:
		return "paused"
	default:
		return "unknown"
	}
}

type HTTPTriggerConfig struct {
	Path          string            `json:"path"`
	Method        []string          `json:"method"`
	AuthType      string            `json:"auth_type"`
	CORSConfig    *CORSConfig       `json:"cors_config"`
	RateLimit     *RateLimitConfig  `json:"rate_limit"`
	CustomDomain  string            `json:"custom_domain"`
	TLSCert       string            `json:"tls_cert"`
	TLSKey        string            `json:"tls_key"`
	APIKeyEnabled bool              `json:"api_key_enabled"`
	Headers       map[string]string `json:"headers"`
}

type CORSConfig struct {
	AllowOrigins     []string `json:"allow_origins"`
	AllowMethods    []string `json:"allow_methods"`
	AllowHeaders    []string `json:"allow_headers"`
	ExposeHeaders   []string `json:"expose_headers"`
	AllowCredentials bool    `json:"allow_credentials"`
	MaxAge           int     `json:"max_age"`
}

type RateLimitConfig struct {
	RequestsPerSecond int `json:"requests_per_second"`
	BurstSize         int `json:"burst_size"`
	Quota             int `json:"quota"`
	QuotaPeriod       int `json:"quota_period_seconds"`
}

type TimerTriggerConfig struct {
	Expression   string `json:"expression"`
	CronEnabled  bool   `json:"cron_enabled"`
	IntervalMs   int64  `json:"interval_ms"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	Timezone     string `json:"timezone"`
	Payload      string `json:"payload"`
}

type QueueTriggerConfig struct {
	QueueName       string            `json:"queue_name"`
	BatchSize       int               `json:"batch_size"`
	VisibilityTimeout int             `json:"visibility_timeout_seconds"`
	MaxRetries      int               `json:"max_retries"`
	DeadLetterQueue string            `json:"dead_letter_queue"`
	MessageFilter   string            `json:"message_filter"`
}

type EventTriggerConfig struct {
	EventSource   string                 `json:"event_source"`
	EventType     string                 `json:"event_type"`
	FilterRules   []EventFilterRule      `json:"filter_rules"`
	RetryPolicy   *EventRetryPolicy      `json:"retry_policy"`
}

type EventFilterRule struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type EventRetryPolicy struct {
	MaxRetries      int `json:"max_retries"`
	BackoffBase     int `json:"backoff_base_seconds"`
	BackoffMax      int `json:"backoff_max_seconds"`
}

type S3TriggerConfig struct {
	Bucket         string   `json:"bucket"`
	Events         []string `json:"events"`
	Prefix         string   `json:"prefix"`
	Suffix         string   `json:"suffix"`
	DestinationARN string   `json:"destination_arn"`
}

type KafkaTriggerConfig struct {
	Brokers        []string          `json:"brokers"`
	Topic          string            `json:"topic"`
	ConsumerGroup  string            `json:"consumer_group"`
	AutoOffsetReset string          `json:"auto_offset_reset"`
	MaxPollRecords int              `json:"max_poll_records"`
	SessionTimeout int              `json:"session_timeout_ms"`
}

type CronTriggerConfig struct {
	Expression    string            `json:"expression"`
	Input         map[string]interface{} `json:"input"`
	OutputDestination string        `json:"output_destination"`
	RetryPolicy   *EventRetryPolicy `json:"retry_policy"`
}

type WebSocketTriggerConfig struct {
	ConnectionTTL    int    `json:"connection_ttl_seconds"`
	RouteExpression  string `json:"route_expression"`
	MessageTimeout   int    `json:"message_timeout_ms"`
}

type TriggerMetadata struct {
	TriggerID     string                 `json:"trigger_id"`
	FunctionName  string                 `json:"function_name"`
	TriggerType   TriggerType            `json:"trigger_type"`
	Name          string                 `json:"name"`
	State         TriggerState           `json:"state"`
	Config        interface{}            `json:"config"`
	Invocations   atomic.Int64           `json:"invocations"`
	Errors        atomic.Int64           `json:"errors"`
	LastTriggered time.Time              `json:"last_triggered"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type TriggerManager struct {
	triggers  map[string]*TriggerMetadata
	configs  map[string]interface{}
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	handlers map[TriggerType]TriggerHandler
}

type TriggerHandler interface {
	Initialize(ctx context.Context, config interface{}) error
	Start() error
	Stop() error
	Trigger(ctx context.Context, data interface{}) error
	GetStatus() (TriggerState, error)
}

type BaseTriggerHandler struct {
	metadata *TriggerMetadata
	ctx      context.Context
	cancel   context.CancelFunc
}

func (h *BaseTriggerHandler) Initialize(ctx context.Context, config interface{}) error {
	return nil
}

func (h *BaseTriggerHandler) Start() error {
	return nil
}

func (h *BaseTriggerHandler) Stop() error {
	h.cancel()
	return nil
}

func (h *BaseTriggerHandler) Trigger(ctx context.Context, data interface{}) error {
	return nil
}

func (h *BaseTriggerHandler) GetStatus() (TriggerState, error) {
	return TriggerStateActive, nil
}

type HTTPTriggerHandler struct {
	*BaseTriggerHandler
	config     *HTTPTriggerConfig
	server     interface{}
}

type TimerTriggerHandler struct {
	*BaseTriggerHandler
	config     *TimerTriggerConfig
	ticker     *time.Ticker
	stopChan   chan struct{}
}

type QueueTriggerHandler struct {
	*BaseTriggerHandler
	config     *QueueTriggerConfig
	processor  QueueProcessor
}

type QueueProcessor interface {
	Process(ctx context.Context, messages []interface{}) error
}

type EventTriggerHandler struct {
	*BaseTriggerHandler
	config     *EventTriggerConfig
	subscriber EventSubscriber
}

type EventSubscriber interface {
	Subscribe(ctx context.Context, handler func(interface{})) error
	Unsubscribe() error
}

func NewTriggerManager() *TriggerManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &TriggerManager{
		triggers:  make(map[string]*TriggerMetadata),
		configs:   make(map[string]interface{}),
		ctx:       ctx,
		cancel:    cancel,
		handlers:  make(map[TriggerType]TriggerHandler),
	}
	
	manager.registerDefaultHandlers()
	
	return manager
}

func (m *TriggerManager) registerDefaultHandlers() {
	m.handlers[TriggerHTTP] = &HTTPTriggerHandler{
		BaseTriggerHandler: &BaseTriggerHandler{},
	}
	m.handlers[TriggerTimer] = &TimerTriggerHandler{
		BaseTriggerHandler: &BaseTriggerHandler{},
	}
	m.handlers[TriggerQueue] = &QueueTriggerHandler{
		BaseTriggerHandler: &BaseTriggerHandler{},
	}
	m.handlers[TriggerEvent] = &EventTriggerHandler{
		BaseTriggerHandler: &BaseTriggerHandler{},
	}
}

func (m *TriggerManager) CreateTrigger(functionName, name string, triggerType TriggerType, config interface{}) (*TriggerMetadata, error) {
	if functionName == "" {
		return nil, fmt.Errorf("function name is required")
	}
	
	if name == "" {
		return nil, fmt.Errorf("trigger name is required")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	triggerID := fmt.Sprintf("%s-%s-%d", functionName, name, time.Now().UnixNano())
	
	metadata := &TriggerMetadata{
		TriggerID:    triggerID,
		FunctionName: functionName,
		TriggerType:  triggerType,
		Name:         name,
		State:        TriggerStateInactive,
		Config:       config,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	m.triggers[triggerID] = metadata
	m.configs[triggerID] = config
	
	return metadata, nil
}

func (m *TriggerManager) GetTrigger(triggerID string) (*TriggerMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	trigger, exists := m.triggers[triggerID]
	if !exists {
		return nil, fmt.Errorf("trigger %s not found", triggerID)
	}
	
	return trigger, nil
}

func (m *TriggerManager) ListTriggers(functionName string) []*TriggerMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var triggers []*TriggerMetadata
	for _, trigger := range m.triggers {
		if functionName == "" || trigger.FunctionName == functionName {
			triggers = append(triggers, trigger)
		}
	}
	
	return triggers
}

func (m *TriggerManager) UpdateTrigger(triggerID string, config interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	trigger, exists := m.triggers[triggerID]
	if !exists {
		return fmt.Errorf("trigger %s not found", triggerID)
	}
	
	trigger.Config = config
	trigger.UpdatedAt = time.Now()
	m.configs[triggerID] = config
	
	return nil
}

func (m *TriggerManager) DeleteTrigger(triggerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.triggers[triggerID]; !exists {
		return fmt.Errorf("trigger %s not found", triggerID)
	}
	
	delete(m.triggers, triggerID)
	delete(m.configs, triggerID)
	
	return nil
}

func (m *TriggerManager) EnableTrigger(triggerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	trigger, exists := m.triggers[triggerID]
	if !exists {
		return fmt.Errorf("trigger %s not found", triggerID)
	}
	
	handler, exists := m.handlers[trigger.TriggerType]
	if !exists {
		return fmt.Errorf("no handler for trigger type %s", trigger.TriggerType)
	}
	
	config := m.configs[triggerID]
	if err := handler.Initialize(m.ctx, config); err != nil {
		trigger.State = TriggerStateError
		return fmt.Errorf("failed to initialize trigger: %w", err)
	}
	
	if err := handler.Start(); err != nil {
		trigger.State = TriggerStateError
		return fmt.Errorf("failed to start trigger: %w", err)
	}
	
	trigger.State = TriggerStateActive
	trigger.UpdatedAt = time.Now()
	
	return nil
}

func (m *TriggerManager) DisableTrigger(triggerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	trigger, exists := m.triggers[triggerID]
	if !exists {
		return fmt.Errorf("trigger %s not found", triggerID)
	}
	
	handler, exists := m.handlers[trigger.TriggerType]
	if !exists {
		return fmt.Errorf("no handler for trigger type %s", trigger.TriggerType)
	}
	
	if err := handler.Stop(); err != nil {
		return fmt.Errorf("failed to stop trigger: %w", err)
	}
	
	trigger.State = TriggerStateInactive
	trigger.UpdatedAt = time.Now()
	
	return nil
}

func (m *TriggerManager) RecordInvocation(triggerID string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if trigger, exists := m.triggers[triggerID]; exists {
		trigger.Invocations.Add(1)
		trigger.LastTriggered = time.Now()
	}
}

func (m *TriggerManager) RecordError(triggerID string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if trigger, exists := m.triggers[triggerID]; exists {
		trigger.Errors.Add(1)
	}
}

func (m *TriggerManager) GetMetrics(functionName string) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var totalInvocations, totalErrors int64
	activeCount := 0
	
	for _, trigger := range m.triggers {
		if functionName == "" || trigger.FunctionName == functionName {
			totalInvocations += trigger.Invocations.Load()
			totalErrors += trigger.Errors.Load()
			if trigger.State == TriggerStateActive {
				activeCount++
			}
		}
	}
	
	return map[string]interface{}{
		"total_triggers":    len(m.triggers),
		"active_triggers":   activeCount,
		"total_invocations": totalInvocations,
		"total_errors":      totalErrors,
	}
}

func (m *TriggerManager) ValidateConfig(triggerType TriggerType, config interface{}) error {
	switch triggerType {
	case TriggerHTTP:
		httpConfig, ok := config.(*HTTPTriggerConfig)
		if !ok {
			return fmt.Errorf("invalid HTTP trigger config type")
		}
		if httpConfig.Path == "" {
			return fmt.Errorf("HTTP path is required")
		}
		if len(httpConfig.Method) == 0 {
			return fmt.Errorf("at least one HTTP method is required")
		}
		
	case TriggerTimer:
		timerConfig, ok := config.(*TimerTriggerConfig)
		if !ok {
			return fmt.Errorf("invalid timer trigger config type")
		}
		if timerConfig.Expression == "" && timerConfig.IntervalMs == 0 {
			return fmt.Errorf("either cron expression or interval is required")
		}
		
	case TriggerQueue:
		queueConfig, ok := config.(*QueueTriggerConfig)
		if !ok {
			return fmt.Errorf("invalid queue trigger config type")
		}
		if queueConfig.QueueName == "" {
			return fmt.Errorf("queue name is required")
		}
		if queueConfig.BatchSize <= 0 {
			return fmt.Errorf("batch size must be positive")
		}
		
	case TriggerEvent:
		eventConfig, ok := config.(*EventTriggerConfig)
		if !ok {
			return fmt.Errorf("invalid event trigger config type")
		}
		if eventConfig.EventSource == "" {
			return fmt.Errorf("event source is required")
		}
		
	default:
		return fmt.Errorf("unsupported trigger type: %s", triggerType)
	}
	
	return nil
}

func (m *TriggerManager) ExportConfig(triggerID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	config, exists := m.configs[triggerID]
	if !exists {
		return "", fmt.Errorf("trigger %s not found", triggerID)
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}
	
	return string(data), nil
}

func (m *TriggerManager) ImportConfig(functionName, name string, triggerType TriggerType, configJSON string) (*TriggerMetadata, error) {
	var config interface{}
	
	switch triggerType {
	case TriggerHTTP:
		var httpConfig HTTPTriggerConfig
		if err := json.Unmarshal([]byte(configJSON), &httpConfig); err != nil {
			return nil, fmt.Errorf("invalid HTTP trigger config: %w", err)
		}
		config = &httpConfig
		
	case TriggerTimer:
		var timerConfig TimerTriggerConfig
		if err := json.Unmarshal([]byte(configJSON), &timerConfig); err != nil {
			return nil, fmt.Errorf("invalid timer trigger config: %w", err)
		}
		config = &timerConfig
		
	case TriggerQueue:
		var queueConfig QueueTriggerConfig
		if err := json.Unmarshal([]byte(configJSON), &queueConfig); err != nil {
			return nil, fmt.Errorf("invalid queue trigger config: %w", err)
		}
		config = &queueConfig
		
	case TriggerEvent:
		var eventConfig EventTriggerConfig
		if err := json.Unmarshal([]byte(configJSON), &eventConfig); err != nil {
			return nil, fmt.Errorf("invalid event trigger config: %w", err)
		}
		config = &eventConfig
		
	default:
		return nil, fmt.Errorf("unsupported trigger type: %s", triggerType)
	}
	
	if err := m.ValidateConfig(triggerType, config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	return m.CreateTrigger(functionName, name, triggerType, config)
}

func (h *HTTPTriggerHandler) Start() error {
	return nil
}

func (h *TimerTriggerHandler) Start() error {
	h.stopChan = make(chan struct{})
	
	if h.config.CronEnabled {
		go h.runCron()
	} else if h.config.IntervalMs > 0 {
		go h.runInterval()
	}
	
	return nil
}

func (h *TimerTriggerHandler) runCron() {
	h.ticker = time.NewTicker(1 * time.Minute)
	defer h.ticker.Stop()
	
	for {
		select {
		case <-h.stopChan:
			return
		case <-h.ticker.C:
			// 模拟触发
		}
	}
}

func (h *TimerTriggerHandler) runInterval() {
	interval := time.Duration(h.config.IntervalMs) * time.Millisecond
	h.ticker = time.NewTicker(interval)
	defer h.ticker.Stop()
	
	for {
		select {
		case <-h.stopChan:
			return
		case <-h.ticker.C:
			// 模拟触发
		}
	}
}

func (h *TimerTriggerHandler) Stop() error {
	close(h.stopChan)
	return nil
}

func CreateHTTPTrigger(functionName, name, path string, methods []string) (*TriggerMetadata, error) {
	manager := NewTriggerManager()
	
	config := &HTTPTriggerConfig{
		Path:   path,
		Method: methods,
	}
	
	return manager.CreateTrigger(functionName, name, TriggerHTTP, config)
}

func CreateTimerTrigger(functionName, name, expression string) (*TriggerMetadata, error) {
	manager := NewTriggerManager()
	
	config := &TimerTriggerConfig{
		Expression:   expression,
		CronEnabled:  true,
		Timezone:     "UTC",
	}
	
	return manager.CreateTrigger(functionName, name, TriggerTimer, config)
}

func CreateQueueTrigger(functionName, name, queueName string, batchSize int) (*TriggerMetadata, error) {
	manager := NewTriggerManager()
	
	config := &QueueTriggerConfig{
		QueueName:     queueName,
		BatchSize:     batchSize,
		MaxRetries:    3,
	}
	
	return manager.CreateTrigger(functionName, name, TriggerQueue, config)
}

func CreateEventTrigger(functionName, name, eventSource, eventType string) (*TriggerMetadata, error) {
	manager := NewTriggerManager()
	
	config := &EventTriggerConfig{
		EventSource: eventSource,
		EventType:  eventType,
	}
	
	return manager.CreateTrigger(functionName, name, TriggerEvent, config)
}

func CreateCronTrigger(functionName, name, expression string, input map[string]interface{}) (*TriggerMetadata, error) {
	manager := NewTriggerManager()
	
	config := &CronTriggerConfig{
		Expression: expression,
		Input:     input,
	}
	
	return manager.CreateTrigger(functionName, name, TriggerCron, config)
}

func ParseCronExpression(expression string) (time.Duration, error) {
	return 1 * time.Minute, nil
}

func ValidateHTTPPath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	
	if path[0] != '/' {
		return fmt.Errorf("path must start with /")
	}
	
	return nil
}

func ValidateCronExpression(expression string) error {
	if expression == "" {
		return fmt.Errorf("cron expression cannot be empty")
	}
	
	// 简单的 cron 表达式验证
	parts := splitAndTrim(expression, " ")
	if len(parts) != 5 && len(parts) != 6 {
		return fmt.Errorf("invalid cron expression format")
	}
	
	return nil
}

func splitAndTrim(s, sep string) []string {
	result := []string{}
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	if sep == "" {
		return []string{s}
	}
	
	result := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
