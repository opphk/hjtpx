package captcha

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
