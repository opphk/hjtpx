package captcha

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	retryConfig *RetryConfig
	rateLimiter *RateLimiter
}

type Option func(*Client)

func NewClient(baseURL string, options ...Option) *Client {
	c := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		retryConfig: &RetryConfig{
			MaxRetries: 3,
			BaseDelay:  100 * time.Millisecond,
			MaxDelay:   5 * time.Second,
		},
		rateLimiter: NewRateLimiter(100, time.Second),
	}

	for _, opt := range options {
		opt(c)
	}

	return c
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

func WithAPIKey(apiKey string) Option {
	return func(c *Client) {
		c.apiKey = apiKey
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

func WithRetryConfig(config *RetryConfig) Option {
	return func(c *Client) {
		c.retryConfig = config
	}
}

func WithRateLimiter(requestsPerSecond int, window time.Duration) Option {
	return func(c *Client) {
		c.rateLimiter = NewRateLimiter(requestsPerSecond, window)
	}
}

type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

type RateLimiter struct {
	mu           sync.Mutex
	tokens       int
	maxTokens    int
	refillRate   int
	lastRefill   time.Time
	window       time.Duration
}

func NewRateLimiter(requestsPerSecond int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     requestsPerSecond,
		maxTokens:  requestsPerSecond,
		refillRate: requestsPerSecond,
		lastRefill: time.Now(),
		window:     window,
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(float64(rl.refillRate) * elapsed.Seconds() / rl.window.Seconds())

	if tokensToAdd > 0 {
		rl.tokens = min(rl.maxTokens, rl.tokens+tokensToAdd)
		rl.lastRefill = now
	}
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if rl.Allow() {
				return nil
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

type SliderCaptchaResponse struct {
	SessionID    string `json:"session_id"`
	ImageURL     string `json:"image_url"`
	PuzzleURL    string `json:"puzzle_url"`
	HintURL      string `json:"hint_url"`
	Shape        int    `json:"shape"`
	SecretY      int    `json:"secret_y"`
	ImageWidth   int    `json:"image_width"`
	ImageHeight  int    `json:"image_height"`
}

type TrajectoryPoint struct {
	X int   `json:"x"`
	Y int   `json:"y"`
	T int64 `json:"t"`
}

type VerifyCaptchaRequest struct {
	SessionID   string           `json:"session_id"`
	X           int              `json:"x"`
	Y           int              `json:"y,omitempty"`
	Trajectory  []TrajectoryPoint `json:"trajectory,omitempty"`
}

type VerifyCaptchaResponse struct {
	Success          bool               `json:"success"`
	Message          string             `json:"message"`
	Remaining        int                `json:"remaining_attempts"`
	TrajectoryResult *TrajectoryResult   `json:"trajectory_result,omitempty"`
}

type TrajectoryResult struct {
	Score   float64  `json:"score"`
	Passed  bool     `json:"passed"`
	Reasons []string `json:"reasons,omitempty"`
}

type BatchVerifyRequest struct {
	Requests []VerifyCaptchaRequest `json:"requests"`
}

type BatchVerifyResponse struct {
	Results   []VerifyResult `json:"results"`
	Success   int            `json:"success_count"`
	Failed    int            `json:"failed_count"`
	TotalTime int64          `json:"total_time_ms"`
}

type VerifyResult struct {
	SessionID string                  `json:"session_id"`
	Success   bool                    `json:"success"`
	Message   string                  `json:"message"`
	Remaining int                     `json:"remaining_attempts,omitempty"`
}

type AsyncVerifyRequest struct {
	SessionID   string           `json:"session_id"`
	X           int              `json:"x"`
	Y           int              `json:"y,omitempty"`
	Trajectory  []TrajectoryPoint `json:"trajectory,omitempty"`
	CallbackURL string           `json:"callback_url,omitempty"`
}

type AsyncVerifyResponse struct {
	TaskID     string `json:"task_id"`
	Status     string `json:"status"`
	ResultURL  string `json:"result_url,omitempty"`
	CreatedAt  int64  `json:"created_at"`
}

type AsyncResultResponse struct {
	TaskID     string                 `json:"task_id"`
	Status     string                 `json:"status"`
	Result     *VerifyCaptchaResponse `json:"result,omitempty"`
	Error      string                 `json:"error,omitempty"`
	CompletedAt int64                 `json:"completed_at,omitempty"`
}

func (c *Client) GetSliderCaptcha(width, height, tolerance int) (*SliderCaptchaResponse, error) {
	url := fmt.Sprintf("%s/api/v1/captcha/slider", c.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	if width > 0 {
		q.Add("width", fmt.Sprintf("%d", width))
	}
	if height > 0 {
		q.Add("height", fmt.Sprintf("%d", height))
	}
	if tolerance > 0 {
		q.Add("tolerance", fmt.Sprintf("%d", tolerance))
	}
	req.URL.RawQuery = q.Encode()

	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    SliderCaptchaResponse  `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return &result.Data, nil
}

func (c *Client) VerifyCaptcha(req *VerifyCaptchaRequest) (*VerifyCaptchaResponse, error) {
	return c.VerifyCaptchaWithContext(context.Background(), req)
}

func (c *Client) VerifyCaptchaWithContext(ctx context.Context, req *VerifyCaptchaRequest) (*VerifyCaptchaResponse, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/captcha/verify", c.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("X-API-Key", c.apiKey)
	}

	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := c.retryConfig.BaseDelay * time.Duration(1<<uint(attempt-1))
			if delay > c.retryConfig.MaxDelay {
				delay = c.retryConfig.MaxDelay
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, lastErr = c.httpClient.Do(httpReq)
		if lastErr == nil {
			break
		}

		if !isRetryableError(lastErr) {
			return nil, lastErr
		}
	}

	if resp != nil {
		defer resp.Body.Close()
	} else {
		return nil, lastErr
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code    int                     `json:"code"`
		Message string                  `json:"message"`
		Data    VerifyCaptchaResponse  `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return &result.Data, nil
}

func (c *Client) BatchVerify(requests []VerifyCaptchaRequest) (*BatchVerifyResponse, error) {
	return c.BatchVerifyWithContext(context.Background(), requests)
}

func (c *Client) BatchVerifyWithContext(ctx context.Context, requests []VerifyCaptchaRequest) (*BatchVerifyResponse, error) {
	if len(requests) == 0 {
		return &BatchVerifyResponse{
			Results: []VerifyResult{},
			Success: 0,
			Failed:  0,
		}, nil
	}

	startTime := time.Now()

	type result struct {
		sessionID string
		response *VerifyCaptchaResponse
		err      error
	}

	results := make([]result, len(requests))
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, 10)

	for i, req := range requests {
		wg.Add(1)
		go func(index int, r VerifyCaptchaRequest) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			resp, err := c.VerifyCaptchaWithContext(ctx, &r)
			results[index] = result{
				sessionID: r.SessionID,
				response:  resp,
				err:       err,
			}
		}(i, req)
	}

	wg.Wait()

	verifyResults := make([]VerifyResult, len(results))
	successCount := 0
	failedCount := 0

	for i, r := range results {
		if r.err != nil {
			verifyResults[i] = VerifyResult{
				SessionID: r.sessionID,
				Success:   false,
				Message:   r.err.Error(),
			}
			failedCount++
		} else {
			verifyResults[i] = VerifyResult{
				SessionID: r.sessionID,
				Success:   r.response.Success,
				Message:   r.response.Message,
				Remaining: r.response.Remaining,
			}
			if r.response.Success {
				successCount++
			} else {
				failedCount++
			}
		}
	}

	return &BatchVerifyResponse{
		Results:   verifyResults,
		Success:   successCount,
		Failed:    failedCount,
		TotalTime: time.Since(startTime).Milliseconds(),
	}, nil
}

func (c *Client) AsyncVerify(req *AsyncVerifyRequest) (*AsyncVerifyResponse, error) {
	return c.AsyncVerifyWithContext(context.Background(), req)
}

func (c *Client) AsyncVerifyWithContext(ctx context.Context, req *AsyncVerifyRequest) (*AsyncVerifyResponse, error) {
	url := fmt.Sprintf("%s/api/v1/captcha/async/verify", c.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code    int                   `json:"code"`
		Message string                `json:"message"`
		Data    AsyncVerifyResponse   `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return &result.Data, nil
}

func (c *Client) GetAsyncResult(taskID string) (*AsyncResultResponse, error) {
	return c.GetAsyncResultWithContext(context.Background(), taskID)
}

func (c *Client) GetAsyncResultWithContext(ctx context.Context, taskID string) (*AsyncResultResponse, error) {
	url := fmt.Sprintf("%s/api/v1/captcha/async/result/%s", c.baseURL, taskID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    AsyncResultResponse    `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return &result.Data, nil
}

func (c *Client) WaitAsyncResult(taskID string, timeout time.Duration) (*AsyncResultResponse, error) {
	return c.WaitAsyncResultWithContext(context.Background(), taskID, timeout)
}

func (c *Client) WaitAsyncResultWithContext(ctx context.Context, taskID string, timeout time.Duration) (*AsyncResultResponse, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(timeout)
	}

	pollInterval := 500 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for async result")
		}

		asyncResult, err := c.GetAsyncResultWithContext(ctx, taskID)
		if err != nil {
			return nil, err
		}

		if asyncResult.Status == "completed" || asyncResult.Status == "failed" {
			return asyncResult, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func (c *Client) Auth() *UserAuth {
	return &UserAuth{client: c}
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

type UserAuth struct {
	client *Client
}

func (u *UserAuth) Login(req *LoginRequest) (*LoginResponse, error) {
	url := fmt.Sprintf("%s/api/v1/auth/login", u.client.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if u.client.apiKey != "" {
		httpReq.Header.Set("X-API-Key", u.client.apiKey)
	}

	resp, err := u.client.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code    int           `json:"code"`
		Message string        `json:"message"`
		Data    LoginResponse `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return &result.Data, nil
}

func (c *Client) Env() *Environment {
	return &Environment{client: c}
}

type Environment struct {
	client *Client
}

type DetectionScriptResponse struct {
	Script string
}

func (e *Environment) GetDetectionScript(callback string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/detect/script", e.client.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	if callback != "" {
		q := req.URL.Query()
		q.Add("callback", callback)
		req.URL.RawQuery = q.Encode()
	}

	if e.client.apiKey != "" {
		req.Header.Set("X-API-Key", e.client.apiKey)
	}

	resp, err := e.client.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if urlErr, ok := err.(*url.Error); ok {
		if urlErr.Temporary() {
			return true
		}
		switch urlErr.Err {
		case io.EOF:
			return true
		}
	}
	return false
}

func ExampleClient() {
	client := NewClient("http://localhost:8080")

	captcha, err := client.GetSliderCaptcha(320, 160, 8)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Session ID: %s\n", captcha.SessionID)

	verifyReq := &VerifyCaptchaRequest{
		SessionID: captcha.SessionID,
		X:         185,
		Y:         captcha.SecretY,
		Trajectory: []TrajectoryPoint{
			{X: 0, Y: captcha.SecretY, T: time.Now().UnixMilli() - 1000},
			{X: 50, Y: captcha.SecretY + 5, T: time.Now().UnixMilli() - 800},
			{X: 100, Y: captcha.SecretY - 3, T: time.Now().UnixMilli() - 500},
			{X: 150, Y: captcha.SecretY + 2, T: time.Now().UnixMilli() - 200},
			{X: 185, Y: captcha.SecretY, T: time.Now().UnixMilli()},
		},
	}

	result, err := client.VerifyCaptcha(verifyReq)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Success: %v, Message: %s\n", result.Success, result.Message)
}

func ExampleBatchVerify() {
	client := NewClient("http://localhost:8080")

	requests := []VerifyCaptchaRequest{
		{SessionID: "session-1", X: 100},
		{SessionID: "session-2", X: 150},
		{SessionID: "session-3", X: 200},
	}

	batchResult, err := client.BatchVerify(requests)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Success: %d, Failed: %d, Time: %dms\n",
		batchResult.Success, batchResult.Failed, batchResult.TotalTime)

	for _, r := range batchResult.Results {
		fmt.Printf("Session %s: Success=%v, Message=%s\n",
			r.SessionID, r.Success, r.Message)
	}
}

func ExampleAsyncVerify() {
	client := NewClient("http://localhost:8080")

	asyncReq := &AsyncVerifyRequest{
		SessionID:  "session-async-1",
		X:          150,
		CallbackURL: "https://example.com/callback",
	}

	asyncResp, err := client.AsyncVerify(asyncReq)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Task ID: %s, Status: %s\n", asyncResp.TaskID, asyncResp.Status)

	result, err := client.WaitAsyncResult(asyncResp.TaskID, 30*time.Second)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Result: Success=%v, Message=%s\n", result.Result.Success, result.Result.Message)
}
