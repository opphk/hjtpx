package captcha

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNilRequest       = errors.New("request cannot be nil")
	ErrMissingChallenge = errors.New("challenge_id is required")
	ErrMissingAnswer    = errors.New("answer is required")
	ErrEmptyDataURI     = errors.New("data URI is empty")
	ErrUnsupportedFormat = errors.New("unsupported image format")
	ErrAPIError         = errors.New("API error")
	ErrNetworkError     = errors.New("network error")
	ErrTimeoutError     = errors.New("timeout error")
	ErrRetryExhausted   = errors.New("retry exhausted")
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

type RetryConfig struct {
	MaxRetries     int
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	BackoffFactor  float64
	RetryableCodes []int
}

func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:     3,
		InitialDelay:   100 * time.Millisecond,
		MaxDelay:       5 * time.Second,
		BackoffFactor:  2.0,
		RetryableCodes: []int{429, 500, 502, 503, 504},
	}
}

func (r *RetryConfig) ShouldRetry(statusCode int) bool {
	for _, code := range r.RetryableCodes {
		if code == statusCode {
			return true
		}
	}
	return false
}

func (r *RetryConfig) NextDelay(attempt int) time.Duration {
	delay := float64(r.InitialDelay) * pow(r.BackoffFactor, float64(attempt))
	if delay > float64(r.MaxDelay) {
		return r.MaxDelay
	}
	jitter := time.Duration(rand.Float64() * 0.3 * delay)
	return time.Duration(delay) + jitter
}

func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

const (
	DefaultAPIEndpoint = "http://localhost:8080"
	ImageCaptchaPath   = "/api/v1/captcha/image"
	ImageVerifyPath    = "/api/v1/captcha/image/verify"
	SliderCaptchaPath  = "/api/v1/captcha/slider"
	ClickCaptchaPath   = "/api/v1/captcha/click"
	VerifyCaptchaPath  = "/api/v1/captcha/verify"
)

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
	appID        string
	appSecret    string
	signatureKey string
	endpoint     string
	httpClient   *http.Client
	timeout      time.Duration
	debugMode    bool
	retryConfig  *RetryConfig
	useSignature bool
	signMutex    sync.Mutex
	lastSignTime int64
	signCache    string
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

func WithAppID(appID string) ClientOption {
	return func(c *Client) {
		c.appID = appID
	}
}

func WithAppSecret(appSecret string) ClientOption {
	return func(c *Client) {
		c.appSecret = appSecret
	}
}

func WithSignatureKey(signatureKey string) ClientOption {
	return func(c *Client) {
		c.signatureKey = signatureKey
		c.useSignature = true
	}
}

func WithRetryConfig(config *RetryConfig) ClientOption {
	return func(c *Client) {
		c.retryConfig = config
	}
}

func NewClient(opts ...ClientOption) *Client {
	client := &Client{
		endpoint:    DefaultAPIEndpoint,
		timeout:     30 * time.Second,
		debugMode:   false,
		httpClient:  &http.Client{},
		retryConfig: DefaultRetryConfig(),
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

func (c *Client) SetRetryConfig(config *RetryConfig) {
	c.retryConfig = config
}

func (c *Client) EnableSignature(use bool) {
	c.useSignature = use
}

func (c *Client) GenerateSignature(method, path string, params map[string]string, body []byte) string {
	if c.signatureKey == "" {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(method)
	sb.WriteString(path)

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString(params[k])
	}

	if len(body) > 0 {
		hash := md5.Sum(body)
		sb.WriteString(hex.EncodeToString(hash[:]))
	}

	sb.WriteString(c.signatureKey)

	hash := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(hash[:])
}

func (c *Client) GenerateHMACSignature(data string) string {
	if c.appSecret == "" {
		return ""
	}

	h := hmac.New(sha256.New, []byte(c.appSecret))
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (c *Client) GenerateToken(nonce string, timestamp int64) string {
	c.signMutex.Lock()
	defer c.signMutex.Unlock()

	if c.lastSignTime == 0 || time.Now().Unix()-c.lastSignTime > 300 {
		data := fmt.Sprintf("%s:%d:%s", c.appID, timestamp, nonce)
		c.signCache = c.GenerateHMACSignature(data)
		c.lastSignTime = timestamp
	}

	return c.signCache
}

func (c *Client) VerifySignature(method, path string, params map[string]string, body []byte, signature string) bool {
	expected := c.GenerateSignature(method, path, params, body)
	return hmac.Equal([]byte(expected), []byte(signature))
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
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, &SDKError{
				Code:    -1,
				Message: "failed to marshal request body",
				Err:     err,
			}
		}
		c.debug("Request body: %s", string(reqBody))
	}

	params := make(map[string]string)
	parsedURL, _ := url.Parse(path)
	if parsedURL.RawQuery != "" {
		queryParams, _ := url.ParseQuery(parsedURL.RawQuery)
		for k, v := range queryParams {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	}

	if c.useSignature && c.signatureKey != "" {
		signature := c.GenerateSignature(method, path, params, reqBody)
		c.debug("Generated signature: %s", signature)
	}

	return c.doRequestWithRetry(method, path, bytes.NewReader(reqBody), "application/json")
}

func (c *Client) doRequestWithRetry(method, path string, body io.Reader, contentType string) (*SDKResponse, error) {
	var lastErr error
	retryConfig := c.retryConfig

	if retryConfig == nil {
		retryConfig = DefaultRetryConfig()
	}

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := retryConfig.NextDelay(attempt - 1)
			c.debug("Retrying request (attempt %d/%d) after %v", attempt, retryConfig.MaxRetries, delay)
			time.Sleep(delay)
		}

		req, err := http.NewRequest(method, c.buildURL(path), body)
		if err != nil {
			return nil, &SDKError{
				Code:    -1,
				Message: "failed to create request",
				Err:     err,
			}
		}

		req.Header.Set("Content-Type", contentType)
		if c.apiKey != "" {
			req.Header.Set("X-API-Key", c.apiKey)
		}
		if c.apiSecret != "" {
			req.Header.Set("X-API-Secret", c.apiSecret)
		}
		if c.appID != "" {
			req.Header.Set("X-App-ID", c.appID)
		}
		if c.useSignature && c.signatureKey != "" {
			signature := c.GenerateSignature(method, path, nil, nil)
			req.Header.Set("X-Signature", signature)
		}

		c.debug("Request: %s %s", method, req.URL.String())

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = &SDKError{
				Code:    -2,
				Message: "failed to send request",
				Err:     err,
			}

			if attempt < retryConfig.MaxRetries {
				c.debug("Network error, will retry: %v", err)
				continue
			}
			return nil, lastErr
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, &SDKError{
				Code:    resp.StatusCode,
				Message: "failed to read response body",
				Err:     err,
			}
		}

		c.debug("Response status: %d, body: %s", resp.StatusCode, string(respBody))

		if resp.StatusCode == http.StatusRequestTimeout {
			lastErr = &SDKError{
				Code:    resp.StatusCode,
				Message: "request timeout",
				Err:     ErrTimeoutError,
			}
			if attempt < retryConfig.MaxRetries {
				c.debug("Request timeout, will retry")
				continue
			}
			return nil, lastErr
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = &SDKError{
				Code:    resp.StatusCode,
				Message: fmt.Sprintf("unexpected status code: %d", resp.StatusCode),
			}

			if retryConfig.ShouldRetry(resp.StatusCode) && attempt < retryConfig.MaxRetries {
				c.debug("Server error %d, will retry", resp.StatusCode)
				continue
			}
			return nil, lastErr
		}

		var sdkResp SDKResponse
		if err := json.Unmarshal(respBody, &sdkResp); err != nil {
			return nil, &SDKError{
				Code:    -3,
				Message: "failed to unmarshal response",
				Err:     err,
			}
		}

		if sdkResp.Code != 0 {
			return nil, &SDKError{
				Code:    sdkResp.Code,
				Message: sdkResp.Message,
				Err:     ErrAPIError,
			}
		}

		return &sdkResp, nil
	}

	return nil, &SDKError{
		Code:    -4,
		Message: "retry exhausted",
		Err:     ErrRetryExhausted,
	}
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
		return nil, &SDKError{
			Code:    -5,
			Message: "request cannot be nil",
			Err:     ErrNilRequest,
		}
	}
	if req.ChallengeID == "" {
		return nil, &SDKError{
			Code:    -5,
			Message: "challenge_id is required",
			Err:     ErrMissingChallenge,
		}
	}
	if req.Answer == "" {
		return nil, &SDKError{
			Code:    -5,
			Message: "answer is required",
			Err:     ErrMissingAnswer,
		}
	}

	sdkResp, err := c.doRequest("POST", ImageVerifyPath, req)
	if err != nil {
		return nil, err
	}

	var verifyResp VerifyImageCaptchaResponse
	if err := json.Unmarshal(sdkResp.Data, &verifyResp); err != nil {
		return nil, &SDKError{
			Code:    -3,
			Message: "failed to unmarshal verify response",
			Err:     err,
		}
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
		return nil, &SDKError{
			Code:    -5,
			Message: "request cannot be nil",
			Err:     ErrNilRequest,
		}
	}
	if req.ChallengeID == "" {
		return nil, &SDKError{
			Code:    -5,
			Message: "challenge_id is required",
			Err:     ErrMissingChallenge,
		}
	}

	sdkResp, err := c.doRequest("POST", VerifyCaptchaPath, req)
	if err != nil {
		return nil, err
	}

	var verifyResp VerifyCaptchaResponse
	if err := json.Unmarshal(sdkResp.Data, &verifyResp); err != nil {
		return nil, &SDKError{
			Code:    -3,
			Message: "failed to unmarshal verify response",
			Err:     err,
		}
	}

	return &verifyResp, nil
}

func (c *Client) ExtractBase64Image(dataURI string) ([]byte, error) {
	if dataURI == "" {
		return nil, &SDKError{
			Code:    -5,
			Message: "data URI is empty",
			Err:     ErrEmptyDataURI,
		}
	}

	prefix := "data:image/png;base64,"
	if len(dataURI) > len(prefix) && dataURI[:len(prefix)] == prefix {
		return base64.StdEncoding.DecodeString(dataURI[len(prefix):])
	}

	prefixJPEG := "data:image/jpeg;base64,"
	if len(dataURI) > len(prefixJPEG) && dataURI[:len(prefixJPEG)] == prefixJPEG {
		return base64.StdEncoding.DecodeString(dataURI[len(prefixJPEG):])
	}

	return nil, &SDKError{
		Code:    -5,
		Message: "unsupported image format",
		Err:     ErrUnsupportedFormat,
	}
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

	mux.HandleFunc(SliderCaptchaPath, func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(fmt.Sprintf(`{
				"challenge_id":"%s",
				"background_image":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
				"slider_image":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
				"slider_width":50,
				"slider_height":50
			}`, m.ChallengeID)),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc(ClickCaptchaPath, func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(fmt.Sprintf(`{
				"challenge_id":"%s",
				"background_image":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
				"target_position":[100,100],
				"target_index":3,
				"icon_positions":[[50,50],[100,100],[150,150],[200,200],[250,250],[300,300],[350,350],[400,400],[450,450]]
			}`, m.ChallengeID)),
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
