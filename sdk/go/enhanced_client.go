package captcha

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// 新增验证码类型常量
const (
	CaptchaTypeRotation CaptchaType = "rotation"
	CaptchaTypeGesture  CaptchaType = "gesture"
	CaptchaTypeJigsaw   CaptchaType = "jigsaw"
	CaptchaTypeVoice    CaptchaType = "voice"
	CaptchaTypeConnect  CaptchaType = "connect"
	CaptchaType3D       CaptchaType = "3d"
)

// RotationCaptchaResult 旋转验证码结果
type RotationCaptchaResult struct {
	ChallengeID string `json:"challenge_id"`
	Image       string `json:"image"`
}

// GestureCaptchaResult 手势验证码结果
type GestureCaptchaResult struct {
	SessionID string `json:"session_id"`
	Pattern   string `json:"pattern,omitempty"`
	GridSize  int    `json:"grid_size,omitempty"`
	Hint      string `json:"hint,omitempty"`
}

// JigsawPiece 拼图碎片
type JigsawPiece struct {
	Index      int `json:"index"`
	OriginalX  int `json:"original_x"`
	OriginalY  int `json:"original_y"`
	CurrentX   int `json:"current_x"`
	CurrentY   int `json:"current_y"`
	Width      int `json:"width"`
	Height     int `json:"height"`
	Rotation   int `json:"rotation,omitempty"`
}

// JigsawCaptchaResult 拼图验证码结果
type JigsawCaptchaResult struct {
	SessionID   string       `json:"session_id"`
	ImageURL    string       `json:"image_url"`
	Pieces      []JigsawPiece `json:"pieces"`
	PieceImages []string     `json:"piece_images"`
	GridSize    int          `json:"grid_size"`
	PieceWidth  int          `json:"piece_width"`
	PieceHeight int          `json:"piece_height"`
	ImageWidth  int          `json:"image_width"`
	ImageHeight int          `json:"image_height"`
}

// VoiceCaptchaResult 语音验证码结果
type VoiceCaptchaResult struct {
	SessionID string `json:"session_id"`
	AudioURL  string `json:"audio_url"`
	Length    int    `json:"length"`
	Hint      string `json:"hint,omitempty"`
}

// ConnectCaptchaResult 连线验证码结果
type ConnectCaptchaResult struct {
	SessionID string     `json:"session_id"`
	ImageURL  string     `json:"image_url"`
	Pairs     [][]int    `json:"pairs"`
	Lines     [][]int    `json:"lines"`
}

// ThreeDCaptchaResult 3D验证码结果
type ThreeDCaptchaResult struct {
	SessionID       string    `json:"session_id"`
	ModelURL        string    `json:"model_url"`
	TargetPosition  []float64 `json:"target_position"`
}

// GenerateRotationCaptcha 获取旋转验证码
func (c *CaptchaClient) GenerateRotationCaptcha() (*RotationCaptchaResult, error) {
	return c.GenerateRotationCaptchaWithContext(context.Background())
}

func (c *CaptchaClient) GenerateRotationCaptchaWithContext(ctx context.Context) (*RotationCaptchaResult, error) {
	body, _, err := c.doRequest(ctx, "GET", "/api/v1/captcha/rotation", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code    int                      `json:"code"`
		Message string                   `json:"message"`
		Data    RotationCaptchaResult    `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, NewSDKError(resp.Code, resp.Message)
	}

	return &resp.Data, nil
}

// VerifyRotationCaptcha 验证旋转验证码
func (c *CaptchaClient) VerifyRotationCaptcha(challengeID string, angle int) (*VerifyResult, error) {
	return c.VerifyRotationCaptchaWithContext(context.Background(), challengeID, angle)
}

func (c *CaptchaClient) VerifyRotationCaptchaWithContext(ctx context.Context, challengeID string, angle int) (*VerifyResult, error) {
	if challengeID == "" {
		return nil, NewSDKError(400, "challenge ID is required")
	}

	body, _, err := c.doRequest(ctx, "POST", "/api/v1/captcha/rotation/verify", map[string]interface{}{
		"challenge_id": challengeID,
		"angle":        angle,
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

// GenerateGestureCaptcha 获取手势验证码
func (c *CaptchaClient) GenerateGestureCaptcha() (*GestureCaptchaResult, error) {
	return c.GenerateGestureCaptchaWithContext(context.Background())
}

func (c *CaptchaClient) GenerateGestureCaptchaWithContext(ctx context.Context) (*GestureCaptchaResult, error) {
	body, _, err := c.doRequest(ctx, "GET", "/api/v1/captcha/gesture", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code    int                      `json:"code"`
		Message string                   `json:"message"`
		Data    GestureCaptchaResult     `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, NewSDKError(resp.Code, resp.Message)
	}

	return &resp.Data, nil
}

// VerifyGestureCaptcha 验证手势验证码
func (c *CaptchaClient) VerifyGestureCaptcha(sessionID string, pattern []int) (*VerifyResult, error) {
	return c.VerifyGestureCaptchaWithContext(context.Background(), sessionID, pattern)
}

func (c *CaptchaClient) VerifyGestureCaptchaWithContext(ctx context.Context, sessionID string, pattern []int) (*VerifyResult, error) {
	if sessionID == "" {
		return nil, NewSDKError(400, "session ID is required")
	}
	if len(pattern) == 0 {
		return nil, NewSDKError(400, "pattern is required")
	}

	body, _, err := c.doRequest(ctx, "POST", "/api/v1/captcha/gesture/verify", map[string]interface{}{
		"session_id": sessionID,
		"pattern":    pattern,
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

// GenerateJigsawCaptcha 获取拼图验证码
func (c *CaptchaClient) GenerateJigsawCaptcha(width, height, gridSize int) (*JigsawCaptchaResult, error) {
	return c.GenerateJigsawCaptchaWithContext(context.Background(), width, height, gridSize)
}

func (c *CaptchaClient) GenerateJigsawCaptchaWithContext(ctx context.Context, width, height, gridSize int) (*JigsawCaptchaResult, error) {
	params := map[string]interface{}{}
	if width > 0 {
		params["width"] = width
	}
	if height > 0 {
		params["height"] = height
	}
	if gridSize > 0 {
		params["grid_size"] = gridSize
	}

	body, _, err := c.doRequest(ctx, "GET", "/api/v1/captcha/jigsaw", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    JigsawCaptchaResult    `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, NewSDKError(resp.Code, resp.Message)
	}

	return &resp.Data, nil
}

// VerifyJigsawCaptcha 验证拼图验证码
func (c *CaptchaClient) VerifyJigsawCaptcha(sessionID string, pieces []JigsawPiece) (*VerifyResult, error) {
	return c.VerifyJigsawCaptchaWithContext(context.Background(), sessionID, pieces)
}

func (c *CaptchaClient) VerifyJigsawCaptchaWithContext(ctx context.Context, sessionID string, pieces []JigsawPiece) (*VerifyResult, error) {
	if sessionID == "" {
		return nil, NewSDKError(400, "session ID is required")
	}
	if len(pieces) == 0 {
		return nil, NewSDKError(400, "pieces is required")
	}

	body, _, err := c.doRequest(ctx, "POST", "/api/v1/captcha/jigsaw/verify", map[string]interface{}{
		"session_id": sessionID,
		"pieces":     pieces,
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

// GenerateVoiceCaptcha 获取语音验证码
func (c *CaptchaClient) GenerateVoiceCaptcha(language string) (*VoiceCaptchaResult, error) {
	return c.GenerateVoiceCaptchaWithContext(context.Background(), language)
}

func (c *CaptchaClient) GenerateVoiceCaptchaWithContext(ctx context.Context, language string) (*VoiceCaptchaResult, error) {
	params := map[string]interface{}{}
	if language != "" {
		params["language"] = language
	}

	body, _, err := c.doRequest(ctx, "GET", "/api/v1/captcha/voice", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code    int                   `json:"code"`
		Message string                `json:"message"`
		Data    VoiceCaptchaResult    `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, NewSDKError(resp.Code, resp.Message)
	}

	return &resp.Data, nil
}

// VerifyVoiceCaptcha 验证语音验证码
func (c *CaptchaClient) VerifyVoiceCaptcha(sessionID, answer string) (*VerifyResult, error) {
	return c.VerifyVoiceCaptchaWithContext(context.Background(), sessionID, answer)
}

func (c *CaptchaClient) VerifyVoiceCaptchaWithContext(ctx context.Context, sessionID, answer string) (*VerifyResult, error) {
	if sessionID == "" {
		return nil, NewSDKError(400, "session ID is required")
	}
	if answer == "" {
		return nil, NewSDKError(400, "answer is required")
	}

	body, _, err := c.doRequest(ctx, "POST", "/api/v1/captcha/voice/verify", map[string]interface{}{
		"session_id": sessionID,
		"answer":     answer,
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

// GenerateConnectCaptcha 获取连线验证码
func (c *CaptchaClient) GenerateConnectCaptcha() (*ConnectCaptchaResult, error) {
	return c.GenerateConnectCaptchaWithContext(context.Background())
}

func (c *CaptchaClient) GenerateConnectCaptchaWithContext(ctx context.Context) (*ConnectCaptchaResult, error) {
	body, _, err := c.doRequest(ctx, "GET", "/api/v1/captcha/connect", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code    int                     `json:"code"`
		Message string                  `json:"message"`
		Data    ConnectCaptchaResult    `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, NewSDKError(resp.Code, resp.Message)
	}

	return &resp.Data, nil
}

// VerifyConnectCaptcha 验证连线验证码
func (c *CaptchaClient) VerifyConnectCaptcha(sessionID string, connections [][]int) (*VerifyResult, error) {
	return c.VerifyConnectCaptchaWithContext(context.Background(), sessionID, connections)
}

func (c *CaptchaClient) VerifyConnectCaptchaWithContext(ctx context.Context, sessionID string, connections [][]int) (*VerifyResult, error) {
	if sessionID == "" {
		return nil, NewSDKError(400, "session ID is required")
	}
	if len(connections) == 0 {
		return nil, NewSDKError(400, "connections is required")
	}

	body, _, err := c.doRequest(ctx, "POST", "/api/v1/captcha/connect/verify", map[string]interface{}{
		"session_id":  sessionID,
		"connections": connections,
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

// GenerateThreeDCaptcha 获取3D验证码
func (c *CaptchaClient) GenerateThreeDCaptcha() (*ThreeDCaptchaResult, error) {
	return c.GenerateThreeDCaptchaWithContext(context.Background())
}

func (c *CaptchaClient) GenerateThreeDCaptchaWithContext(ctx context.Context) (*ThreeDCaptchaResult, error) {
	body, _, err := c.doRequest(ctx, "GET", "/api/v1/captcha/3d", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    ThreeDCaptchaResult    `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, NewSDKError(resp.Code, resp.Message)
	}

	return &resp.Data, nil
}

// VerifyThreeDCaptcha 验证3D验证码
func (c *CaptchaClient) VerifyThreeDCaptcha(sessionID string, targetPosition []float64) (*VerifyResult, error) {
	return c.VerifyThreeDCaptchaWithContext(context.Background(), sessionID, targetPosition)
}

func (c *CaptchaClient) VerifyThreeDCaptchaWithContext(ctx context.Context, sessionID string, targetPosition []float64) (*VerifyResult, error) {
	if sessionID == "" {
		return nil, NewSDKError(400, "session ID is required")
	}
	if len(targetPosition) == 0 {
		return nil, NewSDKError(400, "target position is required")
	}

	body, _, err := c.doRequest(ctx, "POST", "/api/v1/captcha/3d/verify", map[string]interface{}{
		"session_id":      sessionID,
		"target_position": targetPosition,
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

// BatchGetCaptchas 批量获取验证码
func (c *CaptchaClient) BatchGetCaptchas(ctx context.Context, requests []CaptchaRequest) ([]CaptchaResponse, error) {
	responses := make([]CaptchaResponse, len(requests))
	var wg sync.WaitGroup
	var mu sync.Mutex

	concurrency := 10
	if len(requests) < concurrency {
		concurrency = len(requests)
	}

	sem := make(chan struct{}, concurrency)

	for i, req := range requests {
		wg.Add(1)
		sem <- struct{}{}

		go func(index int, r CaptchaRequest) {
			defer wg.Done()
			defer func() { <-sem }()

			select {
			case <-ctx.Done():
				mu.Lock()
				responses[index] = CaptchaResponse{
					Index: index,
					Error: ctx.Err(),
				}
				mu.Unlock()
				return
			default:
			}

			var err error
			var result interface{}

			switch r.Type {
			case CaptchaTypeSlider:
				result, err = c.GenerateSliderCaptchaWithContext(ctx)
			case CaptchaTypeClick:
				result, err = c.GenerateClickCaptchaWithContext(ctx)
			case CaptchaTypeImage:
				result, err = c.GenerateImageCaptchaWithContext(ctx, r.ImageRequest)
			case CaptchaTypeRotation:
				result, err = c.GenerateRotationCaptchaWithContext(ctx)
			case CaptchaTypeGesture:
				result, err = c.GenerateGestureCaptchaWithContext(ctx)
			case CaptchaTypeJigsaw:
				result, err = c.GenerateJigsawCaptchaWithContext(ctx, r.Width, r.Height, r.GridSize)
			case CaptchaTypeVoice:
				result, err = c.GenerateVoiceCaptchaWithContext(ctx, r.Language)
			case CaptchaTypeConnect:
				result, err = c.GenerateConnectCaptchaWithContext(ctx)
			case CaptchaType3D:
				result, err = c.GenerateThreeDCaptchaWithContext(ctx)
			default:
				err = NewSDKError(400, fmt.Sprintf("unsupported captcha type: %s", r.Type))
			}

			mu.Lock()
			if err != nil {
				responses[index] = CaptchaResponse{
					Index: index,
					Error: err,
				}
			} else {
				responses[index] = CaptchaResponse{
					Index:      index,
					SessionID:  getSessionID(result),
					Result:     result,
				}
			}
			mu.Unlock()
		}(i, req)
	}

	wg.Wait()
	return responses, nil
}

// CaptchaRequest 验证码请求
type CaptchaRequest struct {
	Type        CaptchaType
	Width       int
	Height      int
	GridSize    int
	Language    string
	ImageRequest *ImageCaptchaRequest
}

// CaptchaResponse 验证码响应
type CaptchaResponse struct {
	Index     int
	SessionID string
	Result    interface{}
	Error     error
}

func getSessionID(result interface{}) string {
	switch r := result.(type) {
	case *SliderCaptchaResult:
		return r.ChallengeID
	case *ClickCaptchaResult:
		return r.ChallengeID
	case *ImageCaptchaResult:
		return r.ChallengeID
	case *RotationCaptchaResult:
		return r.ChallengeID
	case *GestureCaptchaResult:
		return r.SessionID
	case *JigsawCaptchaResult:
		return r.SessionID
	case *VoiceCaptchaResult:
		return r.SessionID
	case *ConnectCaptchaResult:
		return r.SessionID
	case *ThreeDCaptchaResult:
		return r.SessionID
	default:
		return ""
	}
}

// EnvironmentService 环境检测服务
type EnvironmentService struct {
	client *CaptchaClient
}

func (c *CaptchaClient) Environment() *EnvironmentService {
	return &EnvironmentService{client: c}
}

func (e *EnvironmentService) GetDetectionScript(ctx context.Context, callback string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/detect/script", e.client.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	if callback != "" {
		q := req.URL.Query()
		q.Add("callback", callback)
		req.URL.RawQuery = q.Encode()
	}

	if e.client.config.APIKey != "" {
		req.Header.Set("X-API-Key", e.client.config.APIKey)
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

	if resp.StatusCode != http.StatusOK {
		return "", NewSDKError(resp.StatusCode, string(body))
	}

	return string(body), nil
}

func (e *EnvironmentService) SubmitDetection(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	body, _, err := e.client.doRequest(ctx, "POST", "/api/v1/detect/submit", data)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (e *EnvironmentService) CheckEnvironment(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	body, _, err := e.client.doRequest(ctx, "POST", "/api/v1/detect/check", data)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}
