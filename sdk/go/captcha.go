package captcha

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client 验证码客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewClient 创建新的验证码客户端
func NewClient(baseURL string, options ...Option) *Client {
	c := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range options {
		opt(c)
	}

	return c
}

// Option 客户端配置选项
type Option func(*Client)

// WithHTTPClient 设置自定义HTTP客户端
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithAPIKey 设置API密钥
func WithAPIKey(apiKey string) Option {
	return func(c *Client) {
		c.apiKey = apiKey
	}
}

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// SliderCaptchaResponse 滑块验证码响应
type SliderCaptchaResponse struct {
	SessionID    string `json:"session_id"`
	ImageURL    string `json:"image_url"`
	PuzzleURL   string `json:"puzzle_url"`
	HintURL     string `json:"hint_url"`
	Shape       int    `json:"shape"`
	SecretY     int    `json:"secret_y"`
	ImageWidth  int    `json:"image_width"`
	ImageHeight int    `json:"image_height"`
}

// TrajectoryPoint 轨迹点
type TrajectoryPoint struct {
	X int   `json:"x"`
	Y int   `json:"y"`
	T int64 `json:"t"`
}

// VerifyCaptchaRequest 验证码验证请求
type VerifyCaptchaRequest struct {
	SessionID string             `json:"session_id"`
	X         int                `json:"x"`
	Y         int                `json:"y,omitempty"`
	Trajectory []TrajectoryPoint `json:"trajectory,omitempty"`
}

// VerifyCaptchaResponse 验证码验证响应
type VerifyCaptchaResponse struct {
	Success        bool           `json:"success"`
	Message      string         `json:"message"`
	Remaining    int              `json:"remaining_attempts"`
	TrajectoryResult *TrajectoryResult `json:"trajectory_result,omitempty"`
}

// TrajectoryResult 轨迹分析结果
type TrajectoryResult struct {
	Score   float64  `json:"score"`
	Passed  bool     `json:"passed"`
	Reasons []string `json:"reasons,omitempty"`
}

// GetSliderCaptcha 获取滑块验证码
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
		Code    int                  `json:"code"`
		Message string               `json:"message"`
		Data    SliderCaptchaResponse `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return &result.Data, nil
}

// VerifyCaptcha 验证验证码
func (c *Client) VerifyCaptcha(req *VerifyCaptchaRequest) (*VerifyCaptchaResponse, error) {
	url := fmt.Sprintf("%s/api/v1/captcha/verify", c.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
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
		Code    int                  `json:"code"`
		Message string               `json:"message"`
		Data    VerifyCaptchaResponse `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return &result.Data, nil
}

// UserAuth 用户认证相关
type UserAuth struct {
	client *Client
}

// Auth 获取用户认证
func (c *Client) Auth() *UserAuth {
	return &UserAuth{client: c}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	CaptchaToken string `json:"captcha_token,omitempty"`
}

// LoginResponse 登录响应
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

// Login 用户登录
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
		Code    int          `json:"code"`
		Message string       `json:"message"`
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

// Environment 环境检测相关
type Environment struct {
	client *Client
}

// Env 获取环境检测
func (c *Client) Env() *Environment {
	return &Environment{client: c}
}

// DetectionScriptResponse 环境检测响应
type DetectionScriptResponse struct {
	Script string
}

// GetDetectionScript 获取检测脚本
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

// Usage example
func ExampleClient() {
	// 创建客户端
	client := NewClient("http://localhost:8080")

	// 获取滑块验证码
	captcha, err := client.GetSliderCaptcha(320, 160, 8)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Session ID: %s\n", captcha.SessionID)

	// 验证验证码（这里假设用户滑动到了x=185的位置
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
