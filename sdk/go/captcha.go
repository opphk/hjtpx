package captcha

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	DefaultAPIEndpoint = "http://localhost:8080"
	ImageCaptchaPath   = "/api/v1/captcha/image"
	ImageVerifyPath    = "/api/v1/captcha/image/verify"
	SliderCaptchaPath  = "/api/v1/captcha/slider"
	ClickCaptchaPath   = "/api/v1/captcha/click"
	VerifyCaptchaPath  = "/api/v1/captcha/verify"
)

var (
	ErrNetworkError       = errors.New("network error")
	ErrTimeout           = errors.New("request timeout")
	ErrInvalidResponse   = errors.New("invalid response")
	ErrServerError       = errors.New("server error")
	ErrInvalidParams     = errors.New("invalid parameters")
	ErrVerificationFailed = errors.New("verification failed")
	ErrRateLimited       = errors.New("rate limited")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInternalError     = errors.New("internal error")
)

type SDKError struct {
	Code    int
	Message string
	Err     error
}

func (e *SDKError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("SDKError(code=%d, message=%s): %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("SDKError(code=%d, message=%s)", e.Code, e.Message)
}

func (e *SDKError) Unwrap() error {
	return e.Err
}

func IsSDKError(err error) bool {
	var sdkErr *SDKError
	return errors.As(err, &sdkErr)
}

func GetSDKErrorCode(err error) int {
	var sdkErr *SDKError
	if errors.As(err, &sdkErr) {
		return sdkErr.Code
	}
	return 0
}

func wrapSDKError(code int, message string, err error) *SDKError {
	return &SDKError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func NewSDKError(code int, message string) *SDKError {
	return &SDKError{
		Code:    code,
		Message: message,
	}
}

func (e *SDKError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

type Config struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	HTTPTimeout  time.Duration
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	MaxRetries int
	RetryDelay time.Duration

	BaseURL   string
	AppID     string
	AppSecret string
	DebugMode bool
}

func (c *Config) setDefaults() {
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 10
	}
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = 100
	}
	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = 30 * time.Minute
	}
	if c.ConnMaxIdleTime == 0 {
		c.ConnMaxIdleTime = 5 * time.Minute
	}
	if c.HTTPTimeout == 0 {
		c.HTTPTimeout = 30 * time.Second
	}
	if c.DialTimeout == 0 {
		c.DialTimeout = 10 * time.Second
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = 15 * time.Second
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = 15 * time.Second
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
	if c.RetryDelay == 0 {
		c.RetryDelay = 100 * time.Millisecond
	}
	if c.BaseURL == "" {
		c.BaseURL = DefaultAPIEndpoint
	}
}

type poolStats struct {
	mu                   sync.RWMutex
	activeConns          int
	idleConns            int
	totalRequests        int64
	failedRequests       int64
	successfulRequests    int64
	retriedRequests       int64
	lastError            error
	lastErrorTime        time.Time
}

type CaptchaClient struct {
	httpClient  *http.Client
	baseURL     string
	appID       string
	appSecret   string
	config      *Config
	pool        *poolStats
	transport   *http.Transport
	mu          sync.RWMutex
	closed      bool
}

func NewCaptchaClient(appID, appSecret string, cfg *Config) *CaptchaClient {
	if cfg == nil {
		cfg = &Config{}
	}
	cfg.setDefaults()

	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConns,
		MaxConnsPerHost:     cfg.MaxOpenConns,
		IdleConnTimeout:     cfg.ConnMaxIdleTime,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   cfg.HTTPTimeout,
	}

	return &CaptchaClient{
		httpClient: httpClient,
		baseURL:    cfg.BaseURL,
		appID:      appID,
		appSecret:  appSecret,
		config:     cfg,
		pool: &poolStats{
			activeConns: 0,
			idleConns:   cfg.MaxIdleConns,
		},
		transport: transport,
	}
}

func (c *CaptchaClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.httpClient.CloseIdleConnections()
	c.closed = true
	return nil
}

func (c *CaptchaClient) SetPoolConfig(cfg *Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return errors.New("client is closed")
	}

	if cfg.MaxIdleConns > 0 {
		c.transport.MaxIdleConns = cfg.MaxIdleConns
	}
	if cfg.MaxOpenConns > 0 {
		c.transport.MaxConnsPerHost = cfg.MaxOpenConns
	}
	if cfg.ConnMaxLifetime > 0 {
		c.transport.ResponseHeaderTimeout = cfg.ConnMaxLifetime
	}
	if cfg.ConnMaxIdleTime > 0 {
		c.transport.IdleConnTimeout = cfg.ConnMaxIdleTime
	}
	if cfg.HTTPTimeout > 0 {
		c.httpClient.Timeout = cfg.HTTPTimeout
	}
	if cfg.MaxRetries > 0 {
		c.config.MaxRetries = cfg.MaxRetries
	}
	if cfg.RetryDelay > 0 {
		c.config.RetryDelay = cfg.RetryDelay
	}

	return nil
}

func (c *CaptchaClient) doRequestWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error
	maxRetries := c.config.MaxRetries

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			delay := c.config.RetryDelay * time.Duration(i)
			time.Sleep(delay)
			c.pool.mu.Lock()
			c.pool.retriedRequests++
			c.pool.mu.Unlock()
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				lastErr = fmt.Errorf("%w: %v", ErrTimeout, err)
				continue
			}
			if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "no such host") {
				lastErr = fmt.Errorf("%w: %v", ErrNetworkError, err)
				continue
			}
			lastErr = fmt.Errorf("%w: %v", ErrNetworkError, err)
			continue
		}

		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("%w: status code %d", ErrServerError, resp.StatusCode)
			continue
		}

		if resp.StatusCode == 429 {
			resp.Body.Close()
			lastErr = ErrRateLimited
			retryAfter := resp.Header.Get("Retry-After")
			if retryAfter != "" {
				if delay, parseErr := time.ParseDuration(retryAfter + "s"); parseErr == nil {
					time.Sleep(delay)
				}
			}
			continue
		}

		if resp.StatusCode == 401 {
			resp.Body.Close()
			lastErr = ErrUnauthorized
			continue
		}

		c.pool.mu.Lock()
		c.pool.successfulRequests++
		c.pool.mu.Unlock()

		return resp, nil
	}

	c.pool.mu.Lock()
	c.pool.failedRequests++
	c.pool.lastError = lastErr
	c.pool.lastErrorTime = time.Now()
	c.pool.mu.Unlock()

	return nil, lastErr
}

func (c *CaptchaClient) buildURL(path string) string {
	return c.baseURL + path
}

func (c *CaptchaClient) doRequest(method, path string, body interface{}) (*SDKResponse, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, wrapSDKError(400, "failed to marshal request body", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.buildURL(path), reqBody)
	if err != nil {
		return nil, wrapSDKError(400, "failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.appID != "" {
		req.Header.Set("X-App-ID", c.appID)
	}
	if c.appSecret != "" {
		req.Header.Set("X-App-Secret", c.appSecret)
	}

	c.pool.mu.Lock()
	c.pool.totalRequests++
	c.pool.activeConns++
	c.pool.mu.Unlock()

	defer func() {
		c.pool.mu.Lock()
		c.pool.activeConns--
		c.pool.mu.Unlock()
	}()

	resp, err := c.doRequestWithRetry(req)
	if err != nil {
		if errors.Is(err, ErrTimeout) {
			return nil, wrapSDKError(408, "request timeout", err)
		}
		if errors.Is(err, ErrRateLimited) {
			return nil, wrapSDKError(429, "rate limited", err)
		}
		if errors.Is(err, ErrUnauthorized) {
			return nil, wrapSDKError(401, "unauthorized", err)
		}
		if errors.Is(err, ErrServerError) {
			return nil, wrapSDKError(500, "server error", err)
		}
		return nil, wrapSDKError(500, "request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, wrapSDKError(500, "failed to read response body", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, wrapSDKError(resp.StatusCode, fmt.Sprintf("unexpected status code: %d", resp.StatusCode), nil)
	}

	var sdkResp SDKResponse
	if err := json.Unmarshal(respBody, &sdkResp); err != nil {
		return nil, wrapSDKError(500, "failed to unmarshal response", err)
	}

	if sdkResp.Code != 0 {
		return nil, wrapSDKError(sdkResp.Code, sdkResp.Message, nil)
	}

	return &sdkResp, nil
}

func (c *CaptchaClient) SetDebugMode(debug bool) {
	c.config.DebugMode = debug
}

func (c *CaptchaClient) debug(format string, args ...interface{}) {
	if c.config.DebugMode {
		fmt.Printf(format+"\n", args...)
	}
}

func (c *CaptchaClient) GenerateSliderCaptcha() (*SliderCaptchaResponse, error) {
	sdkResp, err := c.doRequest("GET", SliderCaptchaPath, nil)
	if err != nil {
		return nil, err
	}

	var sliderResp SliderCaptchaResponse
	if err := json.Unmarshal(sdkResp.Data, &sliderResp); err != nil {
		return nil, wrapSDKError(500, "failed to unmarshal slider response", err)
	}

	return &sliderResp, nil
}

func (c *CaptchaClient) VerifySliderCaptcha(captchaID, answer string) (*VerifyCaptchaResponse, error) {
	if captchaID == "" {
		return nil, wrapSDKError(400, "captcha_id is required", ErrInvalidParams)
	}
	if answer == "" {
		return nil, wrapSDKError(400, "answer is required", ErrInvalidParams)
	}

	req := &VerifyCaptchaRequest{
		ChallengeID: captchaID,
		Action:     "slide",
		Data: map[string]interface{}{
			"offset": answer,
		},
	}

	sdkResp, err := c.doRequest("POST", VerifyCaptchaPath, req)
	if err != nil {
		return nil, err
	}

	var verifyResp VerifyCaptchaResponse
	if err := json.Unmarshal(sdkResp.Data, &verifyResp); err != nil {
		return nil, wrapSDKError(500, "failed to unmarshal verify response", err)
	}

	return &verifyResp, nil
}

func (c *CaptchaClient) GenerateClickCaptcha() (*ClickCaptchaResponse, error) {
	sdkResp, err := c.doRequest("GET", ClickCaptchaPath, nil)
	if err != nil {
		return nil, err
	}

	var clickResp ClickCaptchaResponse
	if err := json.Unmarshal(sdkResp.Data, &clickResp); err != nil {
		return nil, wrapSDKError(500, "failed to unmarshal click response", err)
	}

	return &clickResp, nil
}

func (c *CaptchaClient) VerifyClickCaptcha(captchaID string, clicks []ClickData) (*VerifyCaptchaResponse, error) {
	if captchaID == "" {
		return nil, wrapSDKError(400, "captcha_id is required", ErrInvalidParams)
	}
	if len(clicks) == 0 {
		return nil, wrapSDKError(400, "clicks data is required", ErrInvalidParams)
	}

	req := &VerifyCaptchaRequest{
		ChallengeID: captchaID,
		Action:     "click",
		Data: map[string]interface{}{
			"clicks": clicks,
		},
	}

	sdkResp, err := c.doRequest("POST", VerifyCaptchaPath, req)
	if err != nil {
		return nil, err
	}

	var verifyResp VerifyCaptchaResponse
	if err := json.Unmarshal(sdkResp.Data, &verifyResp); err != nil {
		return nil, wrapSDKError(500, "failed to unmarshal verify response", err)
	}

	return &verifyResp, nil
}

func (c *CaptchaClient) GenerateImageCaptcha(req *ImageCaptchaRequest) (*ImageCaptchaResponse, error) {
	if req == nil {
		req = &ImageCaptchaRequest{}
	}

	queryParams := url.Values{}
	if req.Type != "" {
		queryParams.Set("type", string(req.Type))
	}
	if req.Count > 0 {
		queryParams.Set("count", fmt.Sprintf("%d", req.Count))
	}
	if req.CustomSet != "" {
		queryParams.Set("custom_set", req.CustomSet)
	}
	if req.NoiseMode > 0 {
		queryParams.Set("noise_mode", fmt.Sprintf("%d", req.NoiseMode))
	}
	if req.LineMode > 0 {
		queryParams.Set("line_mode", fmt.Sprintf("%d", req.LineMode))
	}

	path := ImageCaptchaPath
	if len(queryParams) > 0 {
		path = path + "?" + queryParams.Encode()
	}

	sdkResp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var captchaResp ImageCaptchaResponse
	if err := json.Unmarshal(sdkResp.Data, &captchaResp); err != nil {
		return nil, wrapSDKError(500, "failed to unmarshal captcha response", err)
	}

	return &captchaResp, nil
}

func (c *CaptchaClient) VerifyImageCaptcha(captchaID, answer string) (*VerifyImageCaptchaResponse, error) {
	if captchaID == "" {
		return nil, wrapSDKError(400, "captcha_id is required", ErrInvalidParams)
	}
	if answer == "" {
		return nil, wrapSDKError(400, "answer is required", ErrInvalidParams)
	}

	req := &VerifyImageCaptchaRequest{
		ChallengeID: captchaID,
		Answer:     answer,
	}

	sdkResp, err := c.doRequest("POST", ImageVerifyPath, req)
	if err != nil {
		return nil, err
	}

	var verifyResp VerifyImageCaptchaResponse
	if err := json.Unmarshal(sdkResp.Data, &verifyResp); err != nil {
		return nil, wrapSDKError(500, "failed to unmarshal verify response", err)
	}

	return &verifyResp, nil
}

func (c *CaptchaClient) ExtractBase64Image(dataURI string) ([]byte, error) {
	if dataURI == "" {
		return nil, errors.New("data URI is empty")
	}

	prefix := "data:image/png;base64,"
	if len(dataURI) > len(prefix) && dataURI[:len(prefix)] == prefix {
		return base64.StdEncoding.DecodeString(dataURI[len(prefix):])
	}

	prefixJPEG := "data:image/jpeg;base64,"
	if len(dataURI) > len(prefixJPEG) && dataURI[:len(prefixJPEG)] == prefixJPEG {
		return base64.StdEncoding.DecodeString(dataURI[len(prefixJPEG):])
	}

	return nil, errors.New("unsupported image format")
}

func (c *CaptchaClient) GetStats() PoolStats {
	c.pool.mu.RLock()
	defer c.pool.mu.RUnlock()

	return PoolStats{
		ActiveConnections:   c.pool.activeConns,
		IdleConnections:     c.pool.idleConns,
		TotalRequests:       c.pool.totalRequests,
		FailedRequests:      c.pool.failedRequests,
		SuccessfulRequests:  c.pool.successfulRequests,
		RetriedRequests:     c.pool.retriedRequests,
		SuccessRate:        calculateSuccessRate(c.pool.totalRequests, c.pool.successfulRequests),
		LastError:          c.pool.lastError,
		LastErrorTime:      c.pool.lastErrorTime,
	}
}

func calculateSuccessRate(total, success int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(success) / float64(total) * 100
}

type PoolStats struct {
	ActiveConnections   int
	IdleConnections     int
	TotalRequests       int64
	FailedRequests      int64
	SuccessfulRequests  int64
	RetriedRequests     int64
	SuccessRate         float64
	LastError           error
	LastErrorTime       time.Time
}

type ClickData struct {
	X        int   `json:"x"`
	Y        int   `json:"y"`
	Duration int64 `json:"duration"`
}

type CaptchaType string

const (
	CaptchaTypeNumber CaptchaType = "number"
	CaptchaTypeLetter CaptchaType = "letter"
	CaptchaTypeMixed  CaptchaType = "mixed"
)

type ImageCaptchaRequest struct {
	Type      CaptchaType `json:"type,omitempty"`
	Count     int         `json:"count,omitempty"`
	CustomSet string      `json:"custom_set,omitempty"`
	NoiseMode int         `json:"noise_mode,omitempty"`
	LineMode  int         `json:"line_mode,omitempty"`
}

type ImageCaptchaResponse struct {
	ChallengeID string `json:"challenge_id"`
	Image       string `json:"image"`
}

type VerifyImageCaptchaRequest struct {
	ChallengeID string `json:"challenge_id"`
	Answer     string `json:"answer"`
}

type VerifyImageCaptchaResponse struct {
	Success bool `json:"success"`
}

type SliderCaptchaRequest struct {
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
}

type SliderCaptchaResponse struct {
	ChallengeID     string `json:"challenge_id"`
	BackgroundImage string `json:"background_image"`
	SliderImage     string `json:"slider_image"`
	SliderWidth     int    `json:"slider_width"`
	SliderHeight    int    `json:"slider_height"`
}

type ClickCaptchaRequest struct {
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
	IconCount int `json:"icon_count,omitempty"`
}

type ClickCaptchaResponse struct {
	ChallengeID     string   `json:"challenge_id"`
	BackgroundImage string   `json:"background_image"`
	TargetPosition  []int    `json:"target_position"`
	TargetIndex     int      `json:"target_index"`
	IconPositions   [][]int  `json:"icon_positions"`
}

type VerifyCaptchaRequest struct {
	ChallengeID string `json:"challenge_id"`
	Action     string `json:"action,omitempty"`
	Data       interface{} `json:"data,omitempty"`
}

type VerifyCaptchaResponse struct {
	Success   bool    `json:"success"`
	Score     float64 `json:"score,omitempty"`
	Message   string  `json:"message,omitempty"`
	RiskLevel string  `json:"risk_level,omitempty"`
}

type BehaviorData struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Timestamp int64  `json:"timestamp"`
	Event     string `json:"event"`
}

type MouseTrajectory struct {
	Points           []BehaviorData `json:"points"`
	TotalDistance   float64        `json:"total_distance"`
	AverageSpeed    float64        `json:"average_speed"`
	MaxSpeed        float64        `json:"max_speed"`
	MinSpeed        float64        `json:"min_speed"`
	PathEfficiency  float64        `json:"path_efficiency"`
	DirectionChanges int           `json:"direction_changes"`
}

type ClickPattern struct {
	Clicks          []BehaviorData `json:"clicks"`
	ClickCount      int            `json:"click_count"`
	AverageInterval float64        `json:"average_interval"`
	ClickSpeed      float64        `json:"click_speed"`
	Regularity      float64        `json:"regularity"`
}

type SpeedAnalysis struct {
	Speeds              []float64 `json:"speeds"`
	AverageSpeed        float64   `json:"average_speed"`
	MedianSpeed         float64   `json:"median_speed"`
	MaxSpeed            float64   `json:"max_speed"`
	MinSpeed            float64   `json:"min_speed"`
	SpeedVariance       float64   `json:"speed_variance"`
	SpeedStdDev         float64   `json:"speed_std_dev"`
	SpeedSkewness       float64   `json:"speed_skewness"`
	Accelerations       []float64 `json:"accelerations"`
	AverageAcceleration float64   `json:"average_acceleration"`
	MaxAcceleration     float64   `json:"max_acceleration"`
	IsSpeedConsistent   bool      `json:"is_speed_consistent"`
	SpeedOutliers       int       `json:"speed_outliers"`
}

type AnalysisResult struct {
	Trajectory      MouseTrajectory `json:"trajectory"`
	ClickPattern   ClickPattern   `json:"click_pattern"`
	SpeedAnalysis  SpeedAnalysis `json:"speed_analysis"`
	RiskScore      float64       `json:"risk_score"`
	RiskIndicators []string      `json:"risk_indicators"`
	IsBotLikely    bool          `json:"is_bot_likely"`
	Confidence     float64       `json:"confidence"`
	RiskFactors    map[string]float64 `json:"risk_factors"`
}

type SDKResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type ClientOption func(*Client)

type Client struct {
	apiKey       string
	apiSecret    string
	endpoint     string
	httpClient   *http.Client
	timeout      time.Duration
	debugMode    bool
}

func WithAPIKey(apiKey string) ClientOption {
	return func(c *Client) {
		c.apiKey = apiKey
	}
}

func WithAPISecret(apiSecret string) ClientOption {
	return func(c *Client) {
		c.apiSecret = apiSecret
	}
}

func WithEndpoint(endpoint string) ClientOption {
	return func(c *Client) {
		c.endpoint = endpoint
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
	}
}

func WithDebugMode(debug bool) ClientOption {
	return func(c *Client) {
		c.debugMode = debug
	}
}

func NewClient(opts ...ClientOption) *Client {
	client := &Client{
		endpoint:   DefaultAPIEndpoint,
		timeout:    30 * time.Second,
		debugMode:   false,
		httpClient: &http.Client{},
	}

	for _, opt := range opts {
		opt(client)
	}

	client.httpClient.Timeout = client.timeout

	return client
}

func (c *Client) SetHTTPClient(httpClient *http.Client) {
	c.httpClient = httpClient
}

func (c *Client) SetEndpoint(endpoint string) {
	c.endpoint = endpoint
}

func (c *Client) SetDebugMode(debug bool) {
	c.debugMode = debug
}

func (c *Client) debug(format string, args ...interface{}) {
	if c.debugMode {
		fmt.Printf(format+"\n", args...)
	}
}

func (c *Client) buildURL(path string) string {
	return c.endpoint + path
}

func (c *Client) doRequest(method, path string, body interface{}) (*SDKResponse, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
		c.debug("Request body: %s", string(jsonData))
	}

	req, err := http.NewRequest(method, c.buildURL(path), reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	if c.apiSecret != "" {
		req.Header.Set("X-API-Secret", c.apiSecret)
	}

	c.debug("Request: %s %s", method, req.URL.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	c.debug("Response status: %d, body: %s", resp.StatusCode, string(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sdkResp SDKResponse
	if err := json.Unmarshal(respBody, &sdkResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if sdkResp.Code != 0 {
		return nil, fmt.Errorf("API error: code=%d, message=%s", sdkResp.Code, sdkResp.Message)
	}

	return &sdkResp, nil
}

func (c *Client) GenerateImageCaptcha(req *ImageCaptchaRequest) (*ImageCaptchaResponse, error) {
	if req == nil {
		req = &ImageCaptchaRequest{}
	}

	queryParams := url.Values{}
	if req.Type != "" {
		queryParams.Set("type", string(req.Type))
	}
	if req.Count > 0 {
		queryParams.Set("count", fmt.Sprintf("%d", req.Count))
	}
	if req.CustomSet != "" {
		queryParams.Set("custom_set", req.CustomSet)
	}
	if req.NoiseMode > 0 {
		queryParams.Set("noise_mode", fmt.Sprintf("%d", req.NoiseMode))
	}
	if req.LineMode > 0 {
		queryParams.Set("line_mode", fmt.Sprintf("%d", req.LineMode))
	}

	path := ImageCaptchaPath
	if len(queryParams) > 0 {
		path = path + "?" + queryParams.Encode()
	}

	sdkResp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var captchaResp ImageCaptchaResponse
	if err := json.Unmarshal(sdkResp.Data, &captchaResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal captcha response: %w", err)
	}

	return &captchaResp, nil
}

func (c *Client) VerifyImageCaptcha(req *VerifyImageCaptchaRequest) (*VerifyImageCaptchaResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.ChallengeID == "" {
		return nil, fmt.Errorf("challenge_id is required")
	}
	if req.Answer == "" {
		return nil, fmt.Errorf("answer is required")
	}

	sdkResp, err := c.doRequest("POST", ImageVerifyPath, req)
	if err != nil {
		return nil, err
	}

	var verifyResp VerifyImageCaptchaResponse
	if err := json.Unmarshal(sdkResp.Data, &verifyResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal verify response: %w", err)
	}

	return &verifyResp, nil
}

func (c *Client) GetSliderCaptcha(req *SliderCaptchaRequest) (*SliderCaptchaResponse, error) {
	if req == nil {
		req = &SliderCaptchaRequest{}
	}

	queryParams := url.Values{}
	if req.Width > 0 {
		queryParams.Set("width", fmt.Sprintf("%d", req.Width))
	}
	if req.Height > 0 {
		queryParams.Set("height", fmt.Sprintf("%d", req.Height))
	}

	path := SliderCaptchaPath
	if len(queryParams) > 0 {
		path = path + "?" + queryParams.Encode()
	}

	sdkResp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var sliderResp SliderCaptchaResponse
	if err := json.Unmarshal(sdkResp.Data, &sliderResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal slider response: %w", err)
	}

	return &sliderResp, nil
}

func (c *Client) GetClickCaptcha(req *ClickCaptchaRequest) (*ClickCaptchaResponse, error) {
	if req == nil {
		req = &ClickCaptchaRequest{}
	}

	queryParams := url.Values{}
	if req.Width > 0 {
		queryParams.Set("width", fmt.Sprintf("%d", req.Width))
	}
	if req.Height > 0 {
		queryParams.Set("height", fmt.Sprintf("%d", req.Height))
	}
	if req.IconCount > 0 {
		queryParams.Set("icon_count", fmt.Sprintf("%d", req.IconCount))
	}

	path := ClickCaptchaPath
	if len(queryParams) > 0 {
		path = path + "?" + queryParams.Encode()
	}

	sdkResp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var clickResp ClickCaptchaResponse
	if err := json.Unmarshal(sdkResp.Data, &clickResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal click response: %w", err)
	}

	return &clickResp, nil
}

func (c *Client) VerifyCaptcha(req *VerifyCaptchaRequest) (*VerifyCaptchaResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.ChallengeID == "" {
		return nil, fmt.Errorf("challenge_id is required")
	}

	sdkResp, err := c.doRequest("POST", VerifyCaptchaPath, req)
	if err != nil {
		return nil, err
	}

	var verifyResp VerifyCaptchaResponse
	if err := json.Unmarshal(sdkResp.Data, &verifyResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal verify response: %w", err)
	}

	return &verifyResp, nil
}

func (c *Client) ExtractBase64Image(dataURI string) ([]byte, error) {
	if dataURI == "" {
		return nil, fmt.Errorf("data URI is empty")
	}

	prefix := "data:image/png;base64,"
	if len(dataURI) > len(prefix) && dataURI[:len(prefix)] == prefix {
		return base64.StdEncoding.DecodeString(dataURI[len(prefix):])
	}

	prefixJPEG := "data:image/jpeg;base64,"
	if len(dataURI) > len(prefixJPEG) && dataURI[:len(prefixJPEG)] == prefixJPEG {
		return base64.StdEncoding.DecodeString(dataURI[len(prefixJPEG):])
	}

	return nil, fmt.Errorf("unsupported image format")
}

func DecodeImageCaptchaResponse(data json.RawMessage) (*ImageCaptchaResponse, error) {
	var resp ImageCaptchaResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func DecodeVerifyResponse(data json.RawMessage) (*VerifyImageCaptchaResponse, error) {
	var resp VerifyImageCaptchaResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func DecodeSliderResponse(data json.RawMessage) (*SliderCaptchaResponse, error) {
	var resp SliderCaptchaResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func DecodeClickResponse(data json.RawMessage) (*ClickCaptchaResponse, error) {
	var resp ClickCaptchaResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

type MockServer struct {
	Port         int
	ChallengeID  string
	CurrentAnswer string
	Server       *http.Server
	VerifyCalls  int
}

func NewMockServer(port int) *MockServer {
	return &MockServer{
		Port:        port,
		ChallengeID: "test-challenge-id",
	}
}

func (m *MockServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc(ImageCaptchaPath, func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(fmt.Sprintf(`{"challenge_id":"%s","image":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="}`, m.ChallengeID)),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc(ImageVerifyPath, func(w http.ResponseWriter, r *http.Request) {
		m.VerifyCalls++
		var req VerifyImageCaptchaRequest
		json.NewDecoder(r.Body).Decode(&req)

		success := req.Answer == m.CurrentAnswer
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(fmt.Sprintf(`{"success":%v}`, success)),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	m.Server = &http.Server{
		Addr:    fmt.Sprintf(":%d", m.Port),
		Handler: mux,
	}

	go m.Server.ListenAndServe()
	return nil
}

func (m *MockServer) Stop() error {
	if m.Server != nil {
		return m.Server.Close()
	}
	return nil
}

func (m *MockServer) SetCorrectAnswer(answer string) {
	m.CurrentAnswer = answer
}
