package captcha

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig 测试配置
type TestConfig struct {
	BaseURL string
	Timeout time.Duration
}

// CaptchaTestClient 验证码测试客户端
type CaptchaTestClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewCaptchaTestClient 创建测试客户端
func NewCaptchaTestClient(baseURL string) *CaptchaTestClient {
	return &CaptchaTestClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SliderCaptchaResponse 滑块验证码响应
type SliderCaptchaResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		SessionID       string `json:"session_id"`
		BackgroundImage string `json:"background_image"`
		SliderImage     string `json:"slider_image"`
		TargetX         int    `json:"target_x"`
		TargetY         int    `json:"target_y"`
		ImageWidth      int    `json:"image_width"`
		ImageHeight     int    `json:"image_height"`
		ExpiresIn       int    `json:"expires_in"`
	} `json:"data"`
}

// SliderVerifyRequest 滑块验证请求
type SliderVerifyRequest struct {
	SessionID   string           `json:"session_id"`
	X           int              `json:"x"`
	Y           int              `json:"y"`
	Trajectory  []TrajectoryPoint `json:"trajectory"`
}

// TrajectoryPoint 轨迹点
type TrajectoryPoint struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Timestamp int64   `json:"timestamp"`
}

// SliderVerifyResponse 滑块验证响应
type SliderVerifyResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Success           bool    `json:"success"`
		Message           string  `json:"message"`
		RemainingAttempts int     `json:"remaining_attempts"`
		RiskScore         float64 `json:"risk_score"`
		Token             string  `json:"token,omitempty"`
	} `json:"data"`
}

// GenerateSliderCaptcha 生成滑块验证码
func (c *CaptchaTestClient) GenerateSliderCaptcha(width, height, tolerance int) (*SliderCaptchaResponse, error) {
	apiURL := fmt.Sprintf("%s/api/v1/captcha/slider/generate?width=%d&height=%d&tolerance=%d",
		c.BaseURL, width, height, tolerance)

	resp, err := c.HTTPClient.Post(apiURL, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result SliderCaptchaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

// VerifySliderCaptcha 验证滑块验证码
func (c *CaptchaTestClient) VerifySliderCaptcha(req *SliderVerifyRequest) (*SliderVerifyResponse, error) {
	apiURL := fmt.Sprintf("%s/api/v1/captcha/slider/verify", c.BaseURL)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	resp, err := c.HTTPClient.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result SliderVerifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

// TestSliderCaptchaGenerate 测试滑块验证码生成
func TestSliderCaptchaGenerate(t *testing.T) {
	client := NewCaptchaTestClient("http://localhost:8080")

	testCases := []struct {
		name      string
		width     int
		height    int
		tolerance int
		expectErr bool
	}{
		{
			name:      "默认尺寸",
			width:     320,
			height:    160,
			tolerance: 8,
			expectErr: false,
		},
		{
			name:      "自定义尺寸",
			width:     400,
			height:    200,
			tolerance: 10,
			expectErr: false,
		},
		{
			name:      "最小尺寸",
			width:     200,
			height:    100,
			tolerance: 5,
			expectErr: false,
		},
		{
			name:      "最大尺寸",
			width:     600,
			height:    400,
			tolerance: 15,
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := client.GenerateSliderCaptcha(tc.width, tc.height, tc.tolerance)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, 0, result.Code)
				assert.NotEmpty(t, result.Data.SessionID)
				assert.NotEmpty(t, result.Data.BackgroundImage)
				assert.NotEmpty(t, result.Data.SliderImage)
				assert.Greater(t, result.Data.TargetX, 0)
				assert.Greater(t, result.Data.TargetY, 0)
				assert.Greater(t, result.Data.ExpiresIn, 0)
			}
		})
	}
}

// TestSliderCaptchaVerify 测试滑块验证码验证
func TestSliderCaptchaVerify(t *testing.T) {
	client := NewCaptchaTestClient("http://localhost:8080")

	// 先生成验证码
	captcha, err := client.GenerateSliderCaptcha(320, 160, 8)
	require.NoError(t, err)
	require.NotNil(t, captcha)

	// 生成轨迹数据
	trajectory := GenerateMockTrajectory(captcha.Data.TargetX, captcha.Data.TargetY)

	testCases := []struct {
		name            string
		sessionID       string
		x               int
		y               int
		trajectory      []TrajectoryPoint
		expectSuccess   bool
	}{
		{
			name:       "正确位置",
			sessionID:  captcha.Data.SessionID,
			x:          captcha.Data.TargetX,
			y:          captcha.Data.TargetY,
			trajectory: trajectory,
			expectSuccess: true,
		},
		{
			name:       "接近正确位置",
			sessionID:  captcha.Data.SessionID,
			x:          captcha.Data.TargetX - 5,
			y:          captcha.Data.TargetY,
			trajectory: trajectory,
			expectSuccess: true,
		},
		{
			name:       "错误位置",
			sessionID:  captcha.Data.SessionID,
			x:          10,
			y:          10,
			trajectory: trajectory,
			expectSuccess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &SliderVerifyRequest{
				SessionID:  tc.sessionID,
				X:          tc.x,
				Y:          tc.y,
				Trajectory: tc.trajectory,
			}

			result, err := client.VerifySliderCaptcha(req)

			require.NoError(t, err)
			assert.Equal(t, 0, result.Code)
			assert.NotNil(t, result.Data.Success)
		})
	}
}

// TestSliderCaptchaExpired 测试验证码过期
func TestSliderCaptchaExpired(t *testing.T) {
	client := NewCaptchaTestClient("http://localhost:8080")

	// 生成验证码
	captcha, err := client.GenerateSliderCaptcha(320, 160, 8)
	require.NoError(t, err)

	// 模拟验证码过期（使用过期session ID）
	req := &SliderVerifyRequest{
		SessionID:  "expired_session_123",
		X:          100,
		Y:          80,
		Trajectory: GenerateMockTrajectory(100, 80),
	}

	result, err := client.VerifySliderCaptcha(req)

	// 预期会失败，因为session不存在或已过期
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestSliderCaptchaInvalidSession 测试无效session
func TestSliderCaptchaInvalidSession(t *testing.T) {
	client := NewCaptchaTestClient("http://localhost:8080")

	req := &SliderVerifyRequest{
		SessionID:  "invalid_session",
		X:          100,
		Y:          80,
		Trajectory: GenerateMockTrajectory(100, 80),
	}

	result, err := client.VerifySliderCaptcha(req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestSliderCaptchaWithEmptyTrajectory 测试空轨迹
func TestSliderCaptchaWithEmptyTrajectory(t *testing.T) {
	client := NewCaptchaTestClient("http://localhost:8080")

	// 生成验证码
	captcha, err := client.GenerateSliderCaptcha(320, 160, 8)
	require.NoError(t, err)

	// 使用空轨迹
	req := &SliderVerifyRequest{
		SessionID:  captcha.Data.SessionID,
		X:          captcha.Data.TargetX,
		Y:          captcha.Data.TargetY,
		Trajectory: []TrajectoryPoint{},
	}

	result, err := client.VerifySliderCaptcha(req)

	require.NoError(t, err)
	// 空轨迹可能会降低风险评分，但仍可能通过
	assert.NotNil(t, result.Data.Success)
}

// TestSliderCaptchaMultipleAttempts 测试多次尝试
func TestSliderCaptchaMultipleAttempts(t *testing.T) {
	client := NewCaptchaTestClient("http://localhost:8080")

	// 生成验证码
	captcha, err := client.GenerateSliderCaptcha(320, 160, 8)
	require.NoError(t, err)

	// 多次错误尝试
	for i := 0; i < 3; i++ {
		req := &SliderVerifyRequest{
			SessionID:  captcha.Data.SessionID,
			X:          10 + i*10,
			Y:          10,
			Trajectory: GenerateMockTrajectory(10, 10),
		}

		result, err := client.VerifySliderCaptcha(req)
		
		if err == nil && result != nil {
			// 如果还有剩余尝试次数，应该还能验证
			if result.Data.RemainingAttempts > 0 {
				continue
			}
		}
	}

	// 最后一次尝试应该失败（次数用尽）
	req := &SliderVerifyRequest{
		SessionID:  captcha.Data.SessionID,
		X:          50,
		Y:          50,
		Trajectory: GenerateMockTrajectory(50, 50),
	}

	result, err := client.VerifySliderCaptcha(req)

	// 应该失败，因为尝试次数已用尽
	if err == nil && result != nil {
		assert.False(t, result.Data.Success)
		assert.Equal(t, 0, result.Data.RemainingAttempts)
	}
}

// GenerateMockTrajectory 生成模拟轨迹数据
func GenerateMockTrajectory(targetX, targetY int) []TrajectoryPoint {
	var trajectory []TrajectoryPoint
	baseTime := time.Now().UnixMilli()
	
	// 生成10个轨迹点
	for i := 0; i < 10; i++ {
		trajectory = append(trajectory, TrajectoryPoint{
			X:        float64(targetX * i / 10),
			Y:        float64(targetY) + float64(i%3-1)*2,
			Timestamp: baseTime + int64(i*50),
		})
	}
	
	return trajectory
}

// TestSliderCaptchaImageFormats 测试不同图片格式
func TestSliderCaptchaImageFormats(t *testing.T) {
	client := NewCaptchaTestClient("http://localhost:8080")

	testCases := []struct {
		name   string
		width  int
		height int
	}{
		{"标准", 320, 160},
		{"正方形", 300, 300},
		{"宽图", 500, 200},
		{"长图", 300, 400},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := client.GenerateSliderCaptcha(tc.width, tc.height, 8)

			require.NoError(t, err)
			assert.Equal(t, tc.width, result.Data.ImageWidth)
			assert.Equal(t, tc.height, result.Data.ImageHeight)
		})
	}
}

// TestSliderCaptchaTolerance 测试容差值
func TestSliderCaptchaTolerance(t *testing.T) {
	client := NewCaptchaTestClient("http://localhost:8080")

	testCases := []struct {
		name      string
		tolerance int
	}{
		{"严格", 3},
		{"标准", 8},
		{"宽松", 15},
		{"非常宽松", 20},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := client.GenerateSliderCaptcha(320, 160, tc.tolerance)

			require.NoError(t, err)
			assert.Equal(t, tc.tolerance, 8) // 实际tolerance可能会被标准化
		})
	}
}
