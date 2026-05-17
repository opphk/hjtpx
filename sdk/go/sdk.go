package sdk

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
	"time"
)

const (
	DefaultAPIEndpoint = "http://localhost:8080/api/v1"
	SDKVersion        = "2.0.0"
)

var (
	ErrNetworkError        = errors.New("network error")
	ErrTimeout            = errors.New("request timeout")
	ErrInvalidResponse    = errors.New("invalid response")
	ErrServerError        = errors.New("server error")
	ErrInvalidParams      = errors.New("invalid parameters")
	ErrVerificationFailed = errors.New("verification failed")
	ErrRateLimited        = errors.New("rate limited")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInternalError      = errors.New("internal error")
)

type Config struct {
	MaxIdleConns     int
	MaxOpenConns     int
	ConnMaxLifetime  time.Duration
	ConnMaxIdleTime  time.Duration
	HTTPTimeout      time.Duration
	DialTimeout      time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	MaxRetries       int
	RetryDelay       time.Duration
	BaseURL          string
	APIKey           string
	APISecret        string
	DebugMode        bool
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

func (e *SDKError) Is(target error) bool {
	return e.Err == target
}

func NewSDKError(code int, message string) *SDKError {
	return &SDKError{Code: code, Message: message}
}

func wrapSDKError(code int, message string, err error) *SDKError {
	return &SDKError{Code: code, Message: message, Err: err}
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

type SDKResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	apiSecret  string
	debugMode  bool
	config     *Config
}

func NewClient(options ...Option) *Client {
	cfg := &Config{}
	cfg.setDefaults()

	client := &Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
		apiKey:    cfg.APIKey,
		apiSecret: cfg.APISecret,
		debugMode: cfg.DebugMode,
		config:    cfg,
	}

	for _, opt := range options {
		opt(client)
	}

	return client
}

type Option func(*Client)

func WithAPIKey(apiKey string) Option {
	return func(c *Client) {
		c.apiKey = apiKey
	}
}

func WithAPISecret(apiSecret string) Option {
	return func(c *Client) {
		c.apiSecret = apiSecret
	}
}

func WithEndpoint(endpoint string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimSuffix(endpoint, "/")
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

func WithDebugMode(debug bool) Option {
	return func(c *Client) {
		c.debugMode = debug
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func (c *Client) SetEndpoint(endpoint string) {
	c.baseURL = strings.TrimSuffix(endpoint, "/")
}

func (c *Client) SetDebugMode(debug bool) {
	c.debugMode = debug
}

func (c *Client) SetHTTPClient(httpClient *http.Client) {
	c.httpClient = httpClient
}

func (c *Client) buildURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return c.baseURL + path
}

func (c *Client) debug(format string, args ...interface{}) {
	if c.debugMode {
		fmt.Printf("[SDK DEBUG] "+format+"\n", args...)
	}
}

func (c *Client) doRequest(method, path string, body interface{}) (*SDKResponse, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, wrapSDKError(400, "failed to marshal request body", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, c.buildURL(path), reqBody)
	if err != nil {
		return nil, wrapSDKError(0, "failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	if c.apiSecret != "" {
		req.Header.Set("X-API-Secret", c.apiSecret)
	}

	c.debug("Request: %s %s", method, path)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			return nil, wrapSDKError(0, "request timeout", ErrTimeout)
		}
		return nil, wrapSDKError(0, "failed to send request", ErrNetworkError)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, wrapSDKError(429, "rate limited", ErrRateLimited)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, wrapSDKError(401, "unauthorized", ErrUnauthorized)
	}
	if resp.StatusCode >= 500 {
		return nil, wrapSDKError(resp.StatusCode, "server error", ErrServerError)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, wrapSDKError(0, "failed to read response", err)
	}

	c.debug("Response: %s", string(respBody))

	var sdkResp SDKResponse
	if err := json.Unmarshal(respBody, &sdkResp); err != nil {
		return nil, wrapSDKError(0, "failed to unmarshal response", err)
	}

	if sdkResp.Code != 0 {
		return nil, wrapSDKError(sdkResp.Code, sdkResp.Message, ErrInvalidResponse)
	}

	return &sdkResp, nil
}

func (c *Client) doRequestWithRetry(method, path string, body interface{}) (*SDKResponse, error) {
	var lastErr error
	maxRetries := 3
	retryDelay := 100 * time.Millisecond

	if c.config != nil {
		maxRetries = c.config.MaxRetries
		retryDelay = c.config.RetryDelay
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			c.debug("Retry attempt %d after %v", attempt, retryDelay)
			time.Sleep(retryDelay * time.Duration(attempt))
		}

		resp, err := c.doRequest(method, path, body)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		sdkErr, ok := err.(*SDKError)
		if !ok {
			return nil, err
		}

		if sdkErr.Code == 0 || sdkErr.Code >= 400 && sdkErr.Code < 500 && sdkErr.Code != 429 {
			return nil, err
		}

		if sdkErr.Code == 429 {
			time.Sleep(time.Second)
		}
	}

	return nil, lastErr
}

func (c *Client) ExtractBase64Image(dataURI string) ([]byte, error) {
	if dataURI == "" {
		return nil, fmt.Errorf("empty data URI")
	}

	parts := strings.SplitN(dataURI, ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid data URI format")
	}

	prefix := parts[0]
	if !strings.HasPrefix(prefix, "data:image/") {
		return nil, fmt.Errorf("not a valid image data URI")
	}

	return base64.StdEncoding.DecodeString(parts[1])
}

func DecodeImageCaptchaResponse(data json.RawMessage) (*ImageCaptchaResponse, error) {
	var resp ImageCaptchaResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func DecodeVerifyResponse(data json.RawMessage) (*VerifyCaptchaResponse, error) {
	var resp VerifyCaptchaResponse
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

func ParseQueryParams(params map[string]interface{}) string {
	if len(params) == 0 {
		return ""
	}
	query := url.Values{}
	for key, value := range params {
		switch v := value.(type) {
		case string:
			if v != "" {
				query.Set(key, v)
			}
		case int:
			if v > 0 {
				query.Set(key, fmt.Sprintf("%d", v))
			}
		case bool:
			query.Set(key, fmt.Sprintf("%v", v))
		}
	}
	return query.Encode()
}
