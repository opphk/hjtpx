package captcha

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type CaptchaType string

const (
	CaptchaTypeSlider  CaptchaType = "slider"
	CaptchaTypeClick   CaptchaType = "click"
	CaptchaTypeImage   CaptchaType = "image"
	CaptchaTypeRotation CaptchaType = "rotation"
	CaptchaTypeGesture CaptchaType = "gesture"
	CaptchaTypeJigsaw  CaptchaType = "jigsaw"
)

const (
	CaptchaTypeNumber CaptchaType = "number"
	CaptchaTypeLetter CaptchaType = "letter"
	CaptchaTypeMixed  CaptchaType = "mixed"
	CaptchaTypeChinese CaptchaType = "chinese"
)

type Config struct {
	BaseURL        string
	MaxIdleConns   int
	MaxOpenConns   int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	HTTPTimeout     time.Duration
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
	DebugMode       bool
	APIKey          string
}

type Stats struct {
	TotalRequests        int64
	SuccessfulRequests   int64
	FailedRequests       int64
	RetriedRequests      int64
	SuccessRate          float64
	ActiveConnections    int
	IdleConnections      int
	LastError            error
	LastErrorTime        time.Time
	mu                   sync.RWMutex
}

type CaptchaClient struct {
	appID      string
	appSecret  string
	config     *Config
	httpClient *http.Client
	baseURL    string
	stats      *Stats
	mu         sync.RWMutex
}

func NewCaptchaClient(appID, appSecret string, cfg *Config) *CaptchaClient {
	if cfg == nil {
		cfg = &Config{}
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:8080"
	}

	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 100 * time.Millisecond
	}

	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = 10 * time.Second
	}

	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConns,
		IdleConnTimeout:     cfg.ConnMaxIdleTime,
	}

	if cfg.ConnMaxLifetime > 0 {
		transport.MaxConnsPerHost = cfg.MaxOpenConns
	}

	client := &CaptchaClient{
		appID:     appID,
		appSecret: appSecret,
		config:    cfg,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   cfg.HTTPTimeout,
		},
		baseURL: strings.TrimSuffix(cfg.BaseURL, "/"),
		stats:   &Stats{},
	}

	return client
}

func (c *CaptchaClient) Close() error {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
	return nil
}

func (c *CaptchaClient) GetStats() *Stats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	stats := &Stats{
		TotalRequests:      c.stats.TotalRequests,
		SuccessfulRequests:  c.stats.SuccessfulRequests,
		FailedRequests:      c.stats.FailedRequests,
		RetriedRequests:     c.stats.RetriedRequests,
		SuccessRate:         c.stats.SuccessRate,
		ActiveConnections:   c.stats.ActiveConnections,
		IdleConnections:     c.stats.IdleConnections,
		LastError:           c.stats.LastError,
		LastErrorTime:       c.stats.LastErrorTime,
	}

	if c.stats.TotalRequests > 0 {
		stats.SuccessRate = float64(c.stats.SuccessfulRequests) / float64(c.stats.TotalRequests) * 100
	}

	return stats
}

func (c *CaptchaClient) SetPoolConfig(cfg *Config) error {
	if cfg.MaxIdleConns > 0 {
		c.config.MaxIdleConns = cfg.MaxIdleConns
	}
	if cfg.MaxOpenConns > 0 {
		c.config.MaxOpenConns = cfg.MaxOpenConns
	}
	if cfg.MaxRetries > 0 {
		c.config.MaxRetries = cfg.MaxRetries
	}
	if cfg.RetryDelay > 0 {
		c.config.RetryDelay = cfg.RetryDelay
	}

	if c.httpClient != nil && c.httpClient.Transport != nil {
		if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
			transport.MaxIdleConns = c.config.MaxIdleConns
			transport.MaxIdleConnsPerHost = c.config.MaxIdleConns
		}
	}

	return nil
}

func (c *CaptchaClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	c.stats.mu.Lock()
	c.stats.TotalRequests++
	c.stats.mu.Unlock()

	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
	}

	url := c.baseURL + path
	var respBody []byte
	var statusCode int
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			c.stats.mu.Lock()
			c.stats.RetriedRequests++
			c.stats.mu.Unlock()

			delay := c.config.RetryDelay * time.Duration(1<<uint(attempt-1))
			time.Sleep(delay)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, 0, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Captcha-Go-SDK/1.0")

		if c.appID != "" {
			req.Header.Set("X-App-ID", c.appID)
		}
		if c.config.APIKey != "" {
			req.Header.Set("X-API-Key", c.config.APIKey)
		}
		if c.appSecret != "" {
			timestamp := time.Now().Unix()
			signStr := fmt.Sprintf("%s:%d:%s", c.appID, timestamp, c.appSecret)
			hash := sha256.Sum256([]byte(signStr))
			signature := hex.EncodeToString(hash[:])
			req.Header.Set("X-Timestamp", fmt.Sprintf("%d", timestamp))
			req.Header.Set("X-Signature", signature)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if !isRetryableError(err) {
				break
			}
			continue
		}

		statusCode = resp.StatusCode
		respBody, err = io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = err
			resp.Body.Close()
			continue
		}

		if !isRetryableStatus(statusCode) {
			break
		}

		lastErr = fmt.Errorf("HTTP %d", statusCode)
	}

	if lastErr != nil {
		c.stats.mu.Lock()
		c.stats.FailedRequests++
		c.stats.LastError = lastErr
		c.stats.LastErrorTime = time.Now()
		c.stats.mu.Unlock()
		return nil, statusCode, lastErr
	}

	c.stats.mu.Lock()
	c.stats.SuccessfulRequests++
	c.stats.mu.Unlock()

	return respBody, statusCode, nil
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "reset by peer")
}

func isRetryableStatus(statusCode int) bool {
	return statusCode == 429 || (statusCode >= 500 && statusCode < 600)
}

type SliderCaptchaResult struct {
	ChallengeID      string `json:"challenge_id"`
	BackgroundImage  string `json:"background_image"`
	SliderImage      string `json:"slider_image"`
	SliderWidth      int    `json:"slider_width"`
	SliderHeight     int    `json:"slider_height"`
	BackgroundWidth  int    `json:"background_width"`
	BackgroundHeight int    `json:"background_height"`
	SecretX          int    `json:"secret_x"`
	SecretY          int    `json:"secret_y"`
	ImageWidth       int    `json:"image_width"`
	ImageHeight      int    `json:"image_height"`
}

type ClickCaptchaResult struct {
	ChallengeID     string       `json:"challenge_id"`
	ImageURL        string       `json:"image_url"`
	HintOrder       []int        `json:"hint_order"`
	TargetIndex     int          `json:"target_index"`
	IconPositions   [][]int      `json:"icon_positions"`
	TargetPosition  []int        `json:"target_position"`
	TotalIcons      int          `json:"total_icons"`
}

type ImageCaptchaResult struct {
	ChallengeID string `json:"challenge_id"`
	Image       string `json:"image"`
}

type ImageCaptchaRequest struct {
	Type       CaptchaType `json:"type,omitempty"`
	Count      int         `json:"count,omitempty"`
	NoiseMode  int         `json:"noise_mode,omitempty"`
	LineMode   int         `json:"line_mode,omitempty"`
	CustomSet  string      `json:"custom_set,omitempty"`
}

type VerifyResult struct {
	Success      bool          `json:"success"`
	Message      string        `json:"message"`
	Score        float64       `json:"score"`
	RiskLevel    string        `json:"risk_level"`
	CaptchaPass  bool          `json:"captcha_pass"`
	FailReason   string        `json:"fail_reason,omitempty"`
	RetryCount   int           `json:"retry_count,omitempty"`
	TrajectoryScore float64   `json:"trajectory_score,omitempty"`
}

type ClickData struct {
	X         int   `json:"x"`
	Y         int   `json:"y"`
	Duration  int64 `json:"duration,omitempty"`
	Timestamp int64 `json:"timestamp,omitempty"`
}

func (c *CaptchaClient) GenerateSliderCaptcha() (*SliderCaptchaResult, error) {
	return c.GenerateSliderCaptchaWithContext(context.Background())
}

func (c *CaptchaClient) GenerateSliderCaptchaWithContext(ctx context.Context) (*SliderCaptchaResult, error) {
	body, _, err := c.doRequest(ctx, "GET", "/api/v1/captcha/slider", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code    int                  `json:"code"`
		Message string               `json:"message"`
		Data    SliderCaptchaResult  `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("API error: %s (code: %d)", resp.Message, resp.Code)
	}

	return &resp.Data, nil
}

func (c *CaptchaClient) VerifySliderCaptcha(challengeID, answer string) (*VerifyResult, error) {
	return c.VerifySliderCaptchaWithContext(context.Background(), challengeID, answer)
}

func (c *CaptchaClient) VerifySliderCaptchaWithContext(ctx context.Context, challengeID, answer string) (*VerifyResult, error) {
	if challengeID == "" {
		return nil, NewSDKError(400, "challenge ID is required")
	}
	if answer == "" {
		return nil, NewSDKError(400, "answer is required")
	}

	body, _, err := c.doRequest(ctx, "POST", "/api/v1/captcha/verify", map[string]interface{}{
		"type":         "slider",
		"challenge_id": challengeID,
		"answer":       answer,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    VerifyResult `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, NewSDKError(resp.Code, resp.Message)
	}

	return &resp.Data, nil
}

func (c *CaptchaClient) GenerateClickCaptcha() (*ClickCaptchaResult, error) {
	return c.GenerateClickCaptchaWithContext(context.Background())
}

func (c *CaptchaClient) GenerateClickCaptchaWithContext(ctx context.Context) (*ClickCaptchaResult, error) {
	body, _, err := c.doRequest(ctx, "GET", "/api/v1/captcha/click", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code    int               `json:"code"`
		Message string            `json:"message"`
		Data    ClickCaptchaResult `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("API error: %s (code: %d)", resp.Message, resp.Code)
	}

	return &resp.Data, nil
}

func (c *CaptchaClient) VerifyClickCaptcha(challengeID string, clicks []ClickData) (*VerifyResult, error) {
	return c.VerifyClickCaptchaWithContext(context.Background(), challengeID, clicks)
}

func (c *CaptchaClient) VerifyClickCaptchaWithContext(ctx context.Context, challengeID string, clicks []ClickData) (*VerifyResult, error) {
	if challengeID == "" {
		return nil, NewSDKError(400, "challenge ID is required")
	}
	if len(clicks) == 0 {
		return nil, NewSDKError(400, "clicks data is required")
	}

	body, _, err := c.doRequest(ctx, "POST", "/api/v1/captcha/verify", map[string]interface{}{
		"type":         "click",
		"challenge_id": challengeID,
		"clicks":       clicks,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    VerifyResult `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, NewSDKError(resp.Code, resp.Message)
	}

	return &resp.Data, nil
}

func (c *CaptchaClient) GenerateImageCaptcha(req *ImageCaptchaRequest) (*ImageCaptchaResult, error) {
	return c.GenerateImageCaptchaWithContext(context.Background(), req)
}

func (c *CaptchaClient) GenerateImageCaptchaWithContext(ctx context.Context, req *ImageCaptchaRequest) (*ImageCaptchaResult, error) {
	var body []byte
	var err error

	if req != nil {
		body, _, err = c.doRequest(ctx, "GET", "/api/v1/captcha/image", req)
	} else {
		body, _, err = c.doRequest(ctx, "GET", "/api/v1/captcha/image", nil)
	}
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code    int                `json:"code"`
		Message string             `json:"message"`
		Data    ImageCaptchaResult `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("API error: %s (code: %d)", resp.Message, resp.Code)
	}

	return &resp.Data, nil
}

type VerifyImageCaptchaRequest struct {
	ChallengeID string `json:"challenge_id"`
	Answer      string `json:"answer"`
}

func (c *CaptchaClient) VerifyImageCaptcha(challengeID, answer string) (*VerifyResult, error) {
	return c.VerifyImageCaptchaWithContext(context.Background(), challengeID, answer)
}

func (c *CaptchaClient) VerifyImageCaptchaWithContext(ctx context.Context, challengeID, answer string) (*VerifyResult, error) {
	if challengeID == "" {
		return nil, NewSDKError(400, "challenge ID is required")
	}
	if answer == "" {
		return nil, NewSDKError(400, "answer is required")
	}

	body, _, err := c.doRequest(ctx, "POST", "/api/v1/captcha/image/verify", map[string]interface{}{
		"challenge_id": challengeID,
		"answer":       answer,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    VerifyResult `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, NewSDKError(resp.Code, resp.Message)
	}

	return &resp.Data, nil
}

func (c *CaptchaClient) ExtractBase64Image(dataURI string) ([]byte, error) {
	if dataURI == "" {
		return nil, NewSDKError(400, "data URI is required")
	}

	parts := strings.SplitN(dataURI, ",", 2)
	if len(parts) != 2 {
		return nil, NewSDKError(400, "invalid data URI format")
	}

	prefix := parts[0]
	if !strings.HasPrefix(prefix, "data:image/") {
		return nil, NewSDKError(400, "invalid image data URI")
	}

	encoded := parts[1]
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, NewSDKError(400, fmt.Sprintf("failed to decode base64: %v", err))
	}

	return decoded, nil
}

type AuthService struct {
	client *CaptchaClient
}

type LoginRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	CaptchaToken string `json:"captcha_token,omitempty"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"user"`
}

func (c *CaptchaClient) Auth() *AuthService {
	return &AuthService{client: c}
}

func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	body, _, err := s.client.doRequest(ctx, "POST", "/api/v1/auth/login", req)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code    int           `json:"code"`
		Message string        `json:"message"`
		Data    LoginResponse `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("API error: %s", resp.Message)
	}

	return &resp.Data, nil
}

func generateHMAC(secret, message string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
