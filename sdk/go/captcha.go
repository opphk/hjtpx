package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	ImageCaptchaPath  = "/captcha/image"
	ImageVerifyPath   = "/captcha/image/verify"
	SliderCaptchaPath = "/captcha/slider"
	ClickCaptchaPath = "/captcha/click"
	GestureCaptchaPath = "/captcha/gesture"
	VerifyCaptchaPath = "/captcha/verify"
	GestureVerifyPath = "/captcha/gesture/verify"
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

type SliderCaptchaResponse struct {
	ChallengeID     string `json:"challenge_id"`
	BackgroundImage string `json:"background_image"`
	SliderImage    string `json:"slider_image"`
	SliderWidth    int    `json:"slider_width"`
	SliderHeight   int    `json:"slider_height"`
	TargetX        int    `json:"target_x,omitempty"`
	TargetY        int    `json:"target_y,omitempty"`
	PuzzleY        int    `json:"puzzle_y,omitempty"`
	PuzzleStyle    int    `json:"puzzle_style,omitempty"`
	Tolerance       int    `json:"tolerance,omitempty"`
}

type SliderCaptchaRequest struct {
	Width     int `json:"width,omitempty"`
	Height    int `json:"height,omitempty"`
	Tolerance int `json:"tolerance,omitempty"`
}

type ClickCaptchaRequest struct {
	Width     int `json:"width,omitempty"`
	Height    int `json:"height,omitempty"`
	IconCount int `json:"icon_count,omitempty"`
	Mode      string `json:"mode,omitempty"`
}

type ClickCaptchaResponse struct {
	ChallengeID   string     `json:"challenge_id"`
	BackgroundImage string   `json:"background_image"`
	ImageURL     string     `json:"image_url,omitempty"`
	Hint         string     `json:"hint,omitempty"`
	HintOrder    []int      `json:"hint_order,omitempty"`
	MaxPoints    int        `json:"max_points"`
	Mode         string     `json:"mode,omitempty"`
	Points       [][2]int   `json:"points,omitempty"`
	TargetIndex  int        `json:"target_index,omitempty"`
	IconPositions [][2]int  `json:"icon_positions,omitempty"`
	TargetPosition [2]int    `json:"target_position,omitempty"`
}

type ClickData struct {
	X        int   `json:"x"`
	Y        int   `json:"y"`
	Duration int64 `json:"duration,omitempty"`
}

type VerifyCaptchaRequest struct {
	ChallengeID     string            `json:"challenge_id"`
	Action         string            `json:"action,omitempty"`
	Type           string            `json:"type,omitempty"`
	X              int               `json:"x,omitempty"`
	Y              int               `json:"y,omitempty"`
	Points         [][2]int         `json:"points,omitempty"`
	ClickSequence  []int            `json:"click_sequence,omitempty"`
	Data           map[string]interface{} `json:"data,omitempty"`
}

type VerifyCaptchaResponse struct {
	Success      bool     `json:"success"`
	Score        float64  `json:"score,omitempty"`
	RiskLevel    string   `json:"risk_level,omitempty"`
	Message      string   `json:"message,omitempty"`
	CaptchaPass  bool     `json:"captcha_pass,omitempty"`
	FailReason   string   `json:"fail_reason,omitempty"`
	RiskScore    float64  `json:"risk_score,omitempty"`
	TrajectoryResult *TrajectoryResult `json:"trajectory_result,omitempty"`
}

type TrajectoryResult struct {
	Score   float64  `json:"score"`
	Passed  bool     `json:"passed"`
	Reasons []string `json:"reasons,omitempty"`
}

type GestureCaptchaResponse struct {
	ChallengeID string `json:"challenge_id"`
	Pattern     string `json:"pattern"`
	GridSize    int    `json:"grid_size"`
	ImageURL    string `json:"image_url,omitempty"`
}

type VerifyGestureRequest struct {
	ChallengeID string  `json:"challenge_id"`
	Pattern    []int   `json:"pattern"`
}

func (c *Client) GenerateImageCaptcha(req *ImageCaptchaRequest) (*ImageCaptchaResponse, error) {
	var queryParams string
	if req != nil {
		params := make(map[string]interface{})
		if req.Type != "" {
			params["type"] = string(req.Type)
		}
		if req.Count > 0 {
			params["count"] = req.Count
		}
		if req.CustomSet != "" {
			params["custom_set"] = req.CustomSet
		}
		if req.NoiseMode > 0 {
			params["noise_mode"] = req.NoiseMode
		}
		if req.LineMode > 0 {
			params["line_mode"] = req.LineMode
		}
		queryParams = ParseQueryParams(params)
	}

	path := ImageCaptchaPath
	if queryParams != "" {
		path = path + "?" + queryParams
	}

	resp, err := c.doRequestWithRetry("GET", path, nil)
	if err != nil {
		return nil, err
	}

	return DecodeImageCaptchaResponse(resp.Data)
}

func (c *Client) VerifyImageCaptcha(req *VerifyImageCaptchaRequest) (*VerifyImageCaptchaResponse, error) {
	if req == nil {
		return nil, NewSDKError(400, "request cannot be nil")
	}
	if req.ChallengeID == "" {
		return nil, NewSDKError(400, "challenge_id is required")
	}
	if req.Answer == "" {
		return nil, NewSDKError(400, "answer is required")
	}

	resp, err := c.doRequestWithRetry("POST", ImageVerifyPath, req)
	if err != nil {
		return nil, err
	}

	var result VerifyImageCaptchaResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetSliderCaptcha(req *SliderCaptchaRequest) (*SliderCaptchaResponse, error) {
	path := SliderCaptchaPath
	if req != nil && (req.Width > 0 || req.Height > 0 || req.Tolerance > 0) {
		params := make(map[string]interface{})
		if req.Width > 0 {
			params["width"] = req.Width
		}
		if req.Height > 0 {
			params["height"] = req.Height
		}
		if req.Tolerance > 0 {
			params["tolerance"] = req.Tolerance
		}
		queryParams := ParseQueryParams(params)
		if queryParams != "" {
			path = path + "?" + queryParams
		}
	}

	resp, err := c.doRequestWithRetry("GET", path, nil)
	if err != nil {
		return nil, err
	}

	return DecodeSliderResponse(resp.Data)
}

func (c *Client) VerifySliderCaptcha(challengeID, answer string) (*VerifyCaptchaResponse, error) {
	if challengeID == "" {
		return nil, NewSDKError(400, "challenge_id is required")
	}
	if answer == "" {
		return nil, NewSDKError(400, "answer is required")
	}

	req := &VerifyCaptchaRequest{
		ChallengeID: challengeID,
		Action:     "slide",
		X:          parseSliderAnswer(answer),
	}

	resp, err := c.doRequestWithRetry("POST", VerifyCaptchaPath, req)
	if err != nil {
		return nil, err
	}

	return DecodeVerifyResponse(resp.Data)
}

func parseSliderAnswer(answer string) int {
	var x int
	fmt.Sscanf(answer, "%d", &x)
	return x
}

func (c *Client) GetClickCaptcha(req *ClickCaptchaRequest) (*ClickCaptchaResponse, error) {
	path := ClickCaptchaPath
	if req != nil {
		params := make(map[string]interface{})
		if req.Width > 0 {
			params["width"] = req.Width
		}
		if req.Height > 0 {
			params["height"] = req.Height
		}
		if req.IconCount > 0 {
			params["icon_count"] = req.IconCount
		}
		if req.Mode != "" {
			params["mode"] = req.Mode
		}
		queryParams := ParseQueryParams(params)
		if queryParams != "" {
			path = path + "?" + queryParams
		}
	}

	resp, err := c.doRequestWithRetry("GET", path, nil)
	if err != nil {
		return nil, err
	}

	return DecodeClickResponse(resp.Data)
}

func (c *Client) VerifyClickCaptcha(challengeID string, clicks []ClickData) (*VerifyCaptchaResponse, error) {
	if challengeID == "" {
		return nil, NewSDKError(400, "challenge_id is required")
	}
	if len(clicks) == 0 {
		return nil, NewSDKError(400, "clicks is required and cannot be empty")
	}

	points := make([][2]int, len(clicks))
	for i, click := range clicks {
		points[i] = [2]int{click.X, click.Y}
	}

	req := &VerifyCaptchaRequest{
		ChallengeID: challengeID,
		Action:     "click",
		Points:     points,
	}

	resp, err := c.doRequestWithRetry("POST", VerifyCaptchaPath, req)
	if err != nil {
		return nil, err
	}

	return DecodeVerifyResponse(resp.Data)
}

func (c *Client) VerifyCaptcha(req *VerifyCaptchaRequest) (*VerifyCaptchaResponse, error) {
	if req == nil {
		return nil, NewSDKError(400, "request cannot be nil")
	}
	if req.ChallengeID == "" {
		return nil, NewSDKError(400, "challenge_id is required")
	}

	resp, err := c.doRequestWithRetry("POST", VerifyCaptchaPath, req)
	if err != nil {
		return nil, err
	}

	return DecodeVerifyResponse(resp.Data)
}

func (c *Client) GetGestureCaptcha() (*GestureCaptchaResponse, error) {
	resp, err := c.doRequestWithRetry("GET", GestureCaptchaPath, nil)
	if err != nil {
		return nil, err
	}

	var result GestureCaptchaResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) VerifyGestureCaptcha(req *VerifyGestureRequest) (*VerifyCaptchaResponse, error) {
	if req == nil {
		return nil, NewSDKError(400, "request cannot be nil")
	}
	if req.ChallengeID == "" {
		return nil, NewSDKError(400, "challenge_id is required")
	}
	if len(req.Pattern) == 0 {
		return nil, NewSDKError(400, "pattern is required")
	}

	resp, err := c.doRequestWithRetry("POST", GestureVerifyPath, req)
	if err != nil {
		return nil, err
	}

	return DecodeVerifyResponse(resp.Data)
}

type CaptchaClient struct {
	*Client
}

func NewCaptchaClient(apiKey, apiSecret string, cfg *Config) *CaptchaClient {
	client := NewClient(
		WithAPIKey(apiKey),
		WithAPISecret(apiSecret),
	)
	if cfg != nil {
		if cfg.BaseURL != "" {
			client.SetEndpoint(cfg.BaseURL)
		}
		if cfg.HTTPTimeout > 0 {
			client.SetHTTPClient(&http.Client{Timeout: cfg.HTTPTimeout})
		}
		if cfg.DebugMode {
			client.SetDebugMode(true)
		}
	}
	return &CaptchaClient{Client: client}
}

func (cc *CaptchaClient) GenerateSliderCaptcha() (*SliderCaptchaResponse, error) {
	return cc.GetSliderCaptcha(nil)
}

func (cc *CaptchaClient) GenerateClickCaptcha() (*ClickCaptchaResponse, error) {
	return cc.GetClickCaptcha(nil)
}

func (cc *CaptchaClient) GenerateImageCaptchaWithOptions(captchaType CaptchaType, count int) (*ImageCaptchaResponse, error) {
	return cc.Client.GenerateImageCaptcha(&ImageCaptchaRequest{
		Type:  captchaType,
		Count: count,
	})
}

func (cc *CaptchaClient) GetStats() *PoolStats {
	return &PoolStats{
		TotalRequests: 0,
	}
}

type PoolStats struct {
	ActiveConnections   int
	IdleConnections    int
	TotalRequests      int64
	FailedRequests     int64
	SuccessfulRequests int64
	RetriedRequests   int64
	SuccessRate       float64
	LastError         error
	LastErrorTime     time.Time
}

func (cc *CaptchaClient) Close() error {
	return nil
}

func (cc *CaptchaClient) VerifyImageCaptcha(challengeID, answer string) (*VerifyImageCaptchaResponse, error) {
	return cc.Client.VerifyImageCaptcha(&VerifyImageCaptchaRequest{
		ChallengeID: challengeID,
		Answer:     answer,
	})
}

func (cc *CaptchaClient) VerifySliderCaptcha(challengeID string, answer string) (*VerifyCaptchaResponse, error) {
	return cc.Client.VerifySliderCaptcha(challengeID, answer)
}

func (cc *CaptchaClient) VerifyClickCaptcha(challengeID string, clicks []ClickData) (*VerifyCaptchaResponse, error) {
	return cc.Client.VerifyClickCaptcha(challengeID, clicks)
}

func (cc *CaptchaClient) SetPoolConfig(cfg *Config) {
	if cfg == nil {
		return
	}
	if cfg.MaxIdleConns > 0 {
		cc.Client.config.MaxIdleConns = cfg.MaxIdleConns
	}
	if cfg.MaxOpenConns > 0 {
		cc.Client.config.MaxOpenConns = cfg.MaxOpenConns
	}
	if cfg.MaxRetries > 0 {
		cc.Client.config.MaxRetries = cfg.MaxRetries
	}
}

func calculateSuccessRate(total, success int64) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(success) / float64(total) * 100
}
