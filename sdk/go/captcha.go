package captcha

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type SignatureAlgorithm string

const (
	AlgorithmHMACSHA256 SignatureAlgorithm = "HMAC-SHA256"
	AlgorithmHMACSHA512 SignatureAlgorithm = "HMAC-SHA512"
	AlgorithmBlake2b256 SignatureAlgorithm = "BLAKE2B-256"
	AlgorithmBlake2b512 SignatureAlgorithm = "BLAKE2B-512"
)

type Client struct {
	baseURL             string
	httpClient          *http.Client
	apiKey              string
	signatureKey        string
	signatureAlgorithm  SignatureAlgorithm
	enableKeyRotation   bool
}

func NewClient(baseURL string, options ...Option) *Client {
	c := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		signatureAlgorithm: AlgorithmHMACSHA256,
	}

	for _, opt := range options {
		opt(c)
	}

	return c
}

type Option func(*Client)

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

func WithSignatureKey(key string) Option {
	return func(c *Client) {
		c.signatureKey = key
	}
}

func WithSignatureAlgorithm(algorithm SignatureAlgorithm) Option {
	return func(c *Client) {
		c.signatureAlgorithm = algorithm
	}
}

func WithKeyRotation(enable bool) Option {
	return func(c *Client) {
		c.enableKeyRotation = enable
	}
}

func (c *Client) generateNonce(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := io.ReadFull(c.randReader(), bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

func (c *Client) randReader() io.Reader {
	return bytes.NewReader(make([]byte, 32))
}

func (c *Client) sortQueryString(query string) string {
	if query == "" {
		return ""
	}

	values := make(map[string][]string)
	parts := strings.Split(query, "&")

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		key := kv[0]
		value := ""
		if len(kv) > 1 {
			value = kv[1]
		}
		values[key] = append(values[key], value)
	}

	var keys []string
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var result []string
	for _, key := range keys {
		for _, value := range values[key] {
			result = append(result, key+"="+value)
		}
	}

	return strings.Join(result, "&")
}

func (c *Client) hashBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	h := sha256.Sum256(body)
	return hex.EncodeToString(h[:])
}

func (c *Client) buildStringToSign(method, path, query string, timestamp int64, nonce, bodyHash string) string {
	var parts []string
	parts = append(parts, strings.ToUpper(method))
	parts = append(parts, path)

	if query != "" {
		sortedQuery := c.sortQueryString(query)
		parts = append(parts, sortedQuery)
	}

	parts = append(parts, strconv.FormatInt(timestamp, 10))

	if nonce != "" {
		parts = append(parts, nonce)
	}

	if bodyHash != "" {
		parts = append(parts, bodyHash)
	}

	return strings.Join(parts, "\n")
}

func (c *Client) computeSignature(key, data string) string {
	switch c.signatureAlgorithm {
	case AlgorithmHMACSHA256:
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write([]byte(data))
		return hex.EncodeToString(mac.Sum(nil))

	case AlgorithmHMACSHA512:
		mac := hmac.New(sha512.New, []byte(key))
		mac.Write([]byte(data))
		return hex.EncodeToString(mac.Sum(nil))

	case AlgorithmBlake2b256, AlgorithmBlake2b512:
		h := sha512.New()
		h.Write([]byte(key))
		h.Write([]byte(data))
		hash := h.Sum(nil)
		if c.signatureAlgorithm == AlgorithmBlake2b256 {
			return hex.EncodeToString(hash[:32])
		}
		return hex.EncodeToString(hash)

	default:
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write([]byte(data))
		return hex.EncodeToString(mac.Sum(nil))
	}
}

func (c *Client) generateSignature(method, path, query string, timestamp int64, nonce string, body []byte) (string, error) {
	if c.signatureKey == "" {
		return "", fmt.Errorf("signature key not configured")
	}

	bodyHash := c.hashBody(body)
	stringToSign := c.buildStringToSign(method, path, query, timestamp, nonce, bodyHash)
	return c.computeSignature(c.signatureKey, stringToSign), nil
}

func (c *Client) addSignatureHeaders(req *http.Request, body []byte) error {
	if c.signatureKey == "" {
		return nil
	}

	timestamp := time.Now().Unix()
	nonce, err := c.generateNonce(16)
	if err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	path := req.URL.Path
	query := req.URL.RawQuery

	signature, err := c.generateSignature(req.Method, path, query, timestamp, nonce, body)
	if err != nil {
		return fmt.Errorf("failed to generate signature: %w", err)
	}

	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Signature-Algorithm", string(c.signatureAlgorithm))

	return nil
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

	if err := c.addSignatureHeaders(req, nil); err != nil {
		return nil, err
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

	if err := c.addSignatureHeaders(httpReq, reqBody); err != nil {
		return nil, err
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

	if err := u.client.addSignatureHeaders(httpReq, reqBody); err != nil {
		return nil, err
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
