package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	SDKVersion   = "2.0.0"
	APIv2BaseURL = "https://api.hjtpx.com/v2"
)

type HjtpxSDK struct {
	APIKey        string
	APISecret     string
	BaseURL       string
	Timeout       time.Duration
	HTTPClient    *http.Client
	RetryAttempts int
	RetryDelay    time.Duration
	Plugins       []Plugin
	Middleware    []MiddlewareFunc
}

type Config struct {
	APIKey        string
	APISecret     string
	BaseURL       string
	Timeout       time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
	EnableDebug   bool
}

type MiddlewareFunc func(*http.Request) error

type Plugin interface {
	Name() string
	Version() string
	Execute(ctx context.Context, req *CaptchaRequest) (*CaptchaResponse, error)
}

type CaptchaRequest struct {
	AppID        string                 `json:"app_id"`
	CaptchaType  string                 `json:"captcha_type"`
	Action       string                 `json:"action"`
	UserID       string                 `json:"user_id,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	Metadata     map[string]string      `json:"metadata,omitempty"`
}

type CaptchaResponse struct {
	CaptchaID    string                 `json:"captcha_id"`
	Status       string                 `json:"status"`
	Type         string                 `json:"type"`
	Data         map[string]interface{} `json:"data"`
	ExpiresAt    time.Time              `json:"expires_at,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	Metadata     map[string]string      `json:"metadata,omitempty"`
}

type VerificationRequest struct {
	CaptchaID    string                 `json:"captcha_id"`
	Token        string                 `json:"token"`
	Solution     interface{}            `json:"solution,omitempty"`
	UserID       string                 `json:"user_id,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
}

type VerificationResponse struct {
	Valid          bool                   `json:"valid"`
	Score          float64                `json:"score,omitempty"`
	RiskLevel      string                 `json:"risk_level,omitempty"`
	Reasons        []string               `json:"reasons,omitempty"`
	SessionID      string                 `json:"session_id,omitempty"`
	RemainingTries int                    `json:"remaining_tries,omitempty"`
	Metadata       map[string]string      `json:"metadata,omitempty"`
}

type AnalyticsRequest struct {
	AppID      string                 `json:"app_id"`
	StartDate  time.Time             `json:"start_date"`
	EndDate    time.Time             `json:"end_date"`
	Metrics    []string              `json:"metrics"`
	Dimensions []string             `json:"dimensions,omitempty"`
	Filters    map[string]interface{} `json:"filters,omitempty"`
}

type AnalyticsResponse struct {
	Results []MetricResult `json:"results"`
	Summary *Summary       `json:"summary,omitempty"`
}

type MetricResult struct {
	Metric   string                 `json:"metric"`
	Value    interface{}            `json:"value"`
	Breakdown map[string]interface{} `json:"breakdown,omitempty"`
}

type Summary struct {
	TotalCount   int64   `json:"total_count"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

type AppConfig struct {
	AppID          string            `json:"app_id"`
	Name           string            `json:"name"`
	AppKey         string            `json:"app_key"`
	EnabledTypes   []string          `json:"enabled_types"`
	SecurityLevel  string            `json:"security_level"`
	CustomSettings map[string]interface{} `json:"custom_settings"`
}

type SDKError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *SDKError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
}

func New(config *Config) *HjtpxSDK {
	if config.BaseURL == "" {
		config.BaseURL = APIv2BaseURL
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}

	return &HjtpxSDK{
		APIKey:        config.APIKey,
		APISecret:     config.APISecret,
		BaseURL:       config.BaseURL,
		Timeout:       config.Timeout,
		HTTPClient:    &http.Client{Timeout: config.Timeout},
		RetryAttempts: config.RetryAttempts,
		RetryDelay:    config.RetryDelay,
		Plugins:       []Plugin{},
		Middleware:    []MiddlewareFunc{},
	}
}

func (s *HjtpxSDK) UsePlugin(plugin Plugin) {
	s.Plugins = append(s.Plugins, plugin)
}

func (s *HjtpxSDK) UseMiddleware(middleware MiddlewareFunc) {
	s.Middleware = append(s.Middleware, middleware)
}

func (s *HjtpxSDK) CreateCaptcha(ctx context.Context, req *CaptchaRequest) (*CaptchaResponse, error) {
	for _, plugin := range s.Plugins {
		if plugin.Name() == "preprocessor" {
			resp, err := plugin.Execute(ctx, req)
			if err == nil && resp != nil {
				return resp, nil
			}
		}
	}

	for _, middleware := range s.Middleware {
		if err := middleware(s.prepareRequest("POST", "/captcha/create", req)); err != nil {
			return nil, err
		}
	}

	var resp CaptchaResponse
	err := s.doRequest(ctx, "POST", "/captcha/create", req, &resp)
	if err != nil {
		return nil, err
	}

	for _, plugin := range s.Plugins {
		if plugin.Name() == "postprocessor" {
			_, err := plugin.Execute(ctx, &CaptchaRequest{
				CaptchaID: resp.CaptchaID,
				Action:    "created",
			})
			if err != nil {
				return nil, err
			}
		}
	}

	return &resp, nil
}

func (s *HjtpxSDK) Verify(ctx context.Context, req *VerificationRequest) (*VerificationResponse, error) {
	for _, middleware := range s.Middleware {
		if err := middleware(s.prepareRequest("POST", "/captcha/verify", req)); err != nil {
			return nil, err
		}
	}

	var resp VerificationResponse
	err := s.doRequest(ctx, "POST", "/captcha/verify", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *HjtpxSDK) GetAnalytics(ctx context.Context, req *AnalyticsRequest) (*AnalyticsResponse, error) {
	var resp AnalyticsResponse
	err := s.doRequest(ctx, "POST", "/analytics/query", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *HjtpxSDK) GetAppConfig(ctx context.Context, appID string) (*AppConfig, error) {
	var resp AppConfig
	err := s.doRequest(ctx, "GET", fmt.Sprintf("/app/%s/config", appID), nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *HjtpxSDK) UpdateAppConfig(ctx context.Context, appID string, config *AppConfig) (*AppConfig, error) {
	var resp AppConfig
	err := s.doRequest(ctx, "PUT", fmt.Sprintf("/app/%s/config", appID), config, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *HjtpxSDK) RegisterWebhook(ctx context.Context, appID, eventType, url string) error {
	req := map[string]string{
		"app_id":    appID,
		"event":     eventType,
		"webhook_url": url,
	}

	return s.doRequest(ctx, "POST", "/webhooks/register", req, nil)
}

func (s *HjtpxSDK) ListWebhooks(ctx context.Context, appID string) ([]WebhookInfo, error) {
	var resp struct {
		Webhooks []WebhookInfo `json:"webhooks"`
	}

	err := s.doRequest(ctx, "GET", fmt.Sprintf("/app/%s/webhooks", appID), nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Webhooks, nil
}

type WebhookInfo struct {
	ID        string    `json:"id"`
	AppID     string    `json:"app_id"`
	EventType string    `json:"event_type"`
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *HjtpxSDK) doRequest(ctx context.Context, method, endpoint string, body, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := s.BaseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", s.APIKey)
	req.Header.Set("X-API-Secret", s.APISecret)
	req.Header.Set("X-SDK-Version", SDKVersion)

	var lastErr error
	for attempt := 0; attempt <= s.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(s.RetryDelay * time.Duration(attempt))
		}

		resp, err := s.HTTPClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if result != nil && resp.StatusCode != http.StatusNoContent {
				if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
					return fmt.Errorf("failed to decode response: %w", err)
				}
			}
			return nil
		}

		if resp.StatusCode >= 500 && attempt < s.RetryAttempts {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		var sdkErr SDKError
		if err := json.NewDecoder(resp.Body).Decode(&sdkErr); err != nil {
			return fmt.Errorf("request failed with status %d", resp.StatusCode)
		}
		return &sdkErr
	}

	return lastErr
}

func (s *HjtpxSDK) prepareRequest(method, endpoint string, body interface{}) error {
	return nil
}

type Builder struct {
	sdk      *HjtpxSDK
	appID    string
	captchaType string
	userID   string
	sessionID string
	ipAddress string
	userAgent string
	parameters map[string]interface{}
	metadata   map[string]string
}

func (s *HjtpxSDK) NewBuilder() *Builder {
	return &Builder{
		sdk:        s,
		parameters: make(map[string]interface{}),
		metadata:   make(map[string]string),
	}
}

func (b *Builder) AppID(appID string) *Builder {
	b.appID = appID
	return b
}

func (b *Builder) Type(captchaType string) *Builder {
	b.captchaType = captchaType
	return b
}

func (b *Builder) UserID(userID string) *Builder {
	b.userID = userID
	return b
}

func (b *Builder) SessionID(sessionID string) *Builder {
	b.sessionID = sessionID
	return b
}

func (b *Builder) IPAddress(ip string) *Builder {
	b.ipAddress = ip
	return b
}

func (b *Builder) UserAgent(ua string) *Builder {
	b.userAgent = ua
	return b
}

func (b *Builder) Parameter(key string, value interface{}) *Builder {
	b.parameters[key] = value
	return b
}

func (b *Builder) Metadata(key, value string) *Builder {
	b.metadata[key] = value
	return b
}

func (b *Builder) Build(ctx context.Context) (*CaptchaResponse, error) {
	req := &CaptchaRequest{
		AppID:       b.appID,
		CaptchaType: b.captchaType,
		Action:      "create",
		UserID:      b.userID,
		SessionID:   b.sessionID,
		IPAddress:   b.ipAddress,
		UserAgent:   b.userAgent,
		Parameters:  b.parameters,
		Metadata:    b.metadata,
	}

	return b.sdk.CreateCaptcha(ctx, req)
}

type RetryPlugin struct {
	maxRetries int
	delay      time.Duration
}

func NewRetryPlugin(maxRetries int, delay time.Duration) *RetryPlugin {
	return &RetryPlugin{
		maxRetries: maxRetries,
		delay:      delay,
	}
}

func (p *RetryPlugin) Name() string {
	return "retry"
}

func (p *RetryPlugin) Version() string {
	return "1.0.0"
}

func (p *RetryPlugin) Execute(ctx context.Context, req *CaptchaRequest) (*CaptchaResponse, error) {
	return nil, nil
}

type CachePlugin struct {
	cache map[string]*CaptchaResponse
}

func NewCachePlugin() *CachePlugin {
	return &CachePlugin{
		cache: make(map[string]*CaptchaResponse),
	}
}

func (p *CachePlugin) Name() string {
	return "cache"
}

func (p *CachePlugin) Version() string {
	return "1.0.0"
}

func (p *CachePlugin) Execute(ctx context.Context, req *CaptchaRequest) (*CaptchaResponse, error) {
	if cached, ok := p.cache[req.SessionID]; ok {
		if time.Now().Before(cached.ExpiresAt) {
			return cached, nil
		}
	}
	return nil, nil
}

func (p *CachePlugin) Store(id string, resp *CaptchaResponse) {
	p.cache[id] = resp
}

type LoggingMiddleware struct {
	enabled bool
}

func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{enabled: true}
}

func (m *LoggingMiddleware) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, nil
}

func (m *LoggingMiddleware) Middleware(next http.RoundTripper) http.RoundTripper {
	return next
}

func (s *HjtpxSDK) doRequestWithMiddleware(ctx context.Context, method, endpoint string, body, result interface{}) error {
	return s.doRequest(ctx, method, endpoint, body, result)
}

type RateLimitPlugin struct {
	maxRequests int
	window      time.Duration
	requests    []time.Time
}

func NewRateLimitPlugin(maxRequests int, window time.Duration) *RateLimitPlugin {
	return &RateLimitPlugin{
		maxRequests: maxRequests,
		window:      window,
		requests:    []time.Time{},
	}
}

func (p *RateLimitPlugin) Name() string {
	return "rate_limiter"
}

func (p *RateLimitPlugin) Version() string {
	return "1.0.0"
}

func (p *RateLimitPlugin) Execute(ctx context.Context, req *CaptchaRequest) (*CaptchaResponse, error) {
	now := time.Now()

	var validRequests []time.Time
	for _, t := range p.requests {
		if now.Sub(t) < p.window {
			validRequests = append(validRequests, t)
		}
	}
	p.requests = validRequests

	if len(p.requests) >= p.maxRequests {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	p.requests = append(p.requests, now)
	return nil, nil
}

type MetricsPlugin struct {
	TotalRequests   int64
	SuccessCount   int64
	FailureCount   int64
	TotalLatency   time.Duration
}

func NewMetricsPlugin() *MetricsPlugin {
	return &MetricsPlugin{}
}

func (p *MetricsPlugin) Name() string {
	return "metrics"
}

func (p *MetricsPlugin) Version() string {
	return "1.0.0"
}

func (p *MetricsPlugin) Execute(ctx context.Context, req *CaptchaRequest) (*CaptchaResponse, error) {
	return nil, nil
}

func (p *MetricsPlugin) RecordSuccess(latency time.Duration) {
	p.TotalRequests++
	p.SuccessCount++
	p.TotalLatency += latency
}

func (p *MetricsPlugin) RecordFailure() {
	p.TotalRequests++
	p.FailureCount++
}

func (p *MetricsPlugin) GetMetrics() map[string]interface{} {
	var avgLatency time.Duration
	if p.TotalRequests > 0 {
		avgLatency = p.TotalLatency / time.Duration(p.TotalRequests)
	}

	return map[string]interface{}{
		"total_requests":  p.TotalRequests,
		"success_count":   p.SuccessCount,
		"failure_count":   p.FailureCount,
		"success_rate":    float64(p.SuccessCount) / float64(p.TotalRequests),
		"avg_latency_ms":  avgLatency.Milliseconds(),
	}
}

type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	failures        int
	state            string
	lastFailure     time.Time
}

func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            "closed",
	}
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if cb.state == "open" {
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = "half-open"
			cb.failures = 0
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	err := fn()
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.failureThreshold {
			cb.state = "open"
		}
		return err
	}

	if cb.state == "half-open" {
		cb.state = "closed"
		cb.failures = 0
	}

	return nil
}

func (cb *CircuitBreaker) GetState() string {
	return cb.state
}
