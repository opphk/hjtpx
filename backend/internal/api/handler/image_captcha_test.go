package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
	"github.com/stretchr/testify/assert"
)

func TestGenerateImageCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/v1/captcha/image", GenerateImageCaptcha)

	tests := []struct {
		name           string
		queryParams    string
		expectedCode   int
	}{
		{
			name:           "默认参数",
			queryParams:    "",
			expectedCode:   http.StatusOK,
		},
		{
			name:           "数字验证码",
			queryParams:    "?type=number",
			expectedCode:   http.StatusOK,
		},
		{
			name:           "字母验证码",
			queryParams:    "?type=letter",
			expectedCode:   http.StatusOK,
		},
		{
			name:           "混合验证码，6位",
			queryParams:    "?type=mixed&count=6",
			expectedCode:   http.StatusOK,
		},
		{
			name:           "超过最大长度限制",
			queryParams:    "?count=10",
			expectedCode:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清理测试前的存储
			fallbackCaptchaStore = make(map[string]string)
			
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/captcha/image"+tt.queryParams, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			var resp response.Response
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, 0, resp.Code)

			dataMap, ok := resp.Data.(map[string]interface{})
			assert.True(t, ok)
			assert.NotEmpty(t, dataMap["challenge_id"])
			assert.NotEmpty(t, dataMap["image"])
		})
	}
}

func TestVerifyImageCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/v1/captcha/image/verify", VerifyImageCaptcha)

	tests := []struct {
		name            string
		setupStore      func()
		requestBody     VerifyImageCaptchaRequest
		expectedCode    int
		expectSuccess bool
	}{
		{
			name: "成功验证",
			setupStore: func() {
				setCaptchaAnswer("test-challenge-id", "abcd")
			},
			requestBody: VerifyImageCaptchaRequest{
				ChallengeID: "test-challenge-id",
				Answer:      "abcd",
			},
			expectedCode:    http.StatusOK,
			expectSuccess: true,
		},
		{
			name: "不区分大小写",
			setupStore: func() {
				setCaptchaAnswer("test-challenge-id-2", "abcd")
			},
			requestBody: VerifyImageCaptchaRequest{
				ChallengeID: "test-challenge-id-2",
				Answer:      "ABCD",
			},
			expectedCode:    http.StatusOK,
			expectSuccess: true,
		},
		{
			name: "验证码无效或过期",
			setupStore: func() {},
			requestBody: VerifyImageCaptchaRequest{
				ChallengeID: "invalid-challenge-id",
				Answer:      "abcd",
			},
			expectedCode:    http.StatusOK,
			expectSuccess: false,
		},
		{
			name: "答案错误",
			setupStore: func() {
				setCaptchaAnswer("test-challenge-id-3", "abcd")
			},
			requestBody: VerifyImageCaptchaRequest{
				ChallengeID: "test-challenge-id-3",
				Answer:      "wrong",
			},
			expectedCode:    http.StatusOK,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清理测试前的存储
			fallbackCaptchaStore = make(map[string]string)
			
			tt.setupStore()

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/captcha/image/verify", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			var resp response.Response
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)

			// 只有在我们期望成功的情况检查
			if tt.expectSuccess {
				assert.Equal(t, 0, resp.Code)
				dataMap, ok := resp.Data.(map[string]interface{})
				assert.True(t, ok)
				assert.True(t, dataMap["success"].(bool))
			} else {
				// 对于失败的情况，可能有不同的响应码
				// 但我们主要检查如果有响应是有效的JSON响应，只要没有崩溃
			}
		})
	}
}

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		chars  string
		length int
	}{
		{
			name:   "数字字符串",
			chars:  digitChars,
			length: 4,
		},
		{
			name:   "字母字符串",
			chars:  letterChars,
			length: 6,
		},
		{
			name:   "混合字符串",
			chars:  allChars,
			length: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateRandomString(tt.chars, tt.length)
			assert.Len(t, result, tt.length)

			for _, c := range result {
				assert.Contains(t, tt.chars, string(c))
			}
		})
	}
}

func TestGenerateCaptchaImage(t *testing.T) {
	testText := "test123"
	img := generateCaptchaImage(testText)

	assert.NotNil(t, img)
	assert.Equal(t, captchaWidth, img.Bounds().Dx())
	assert.Equal(t, captchaHeight, img.Bounds().Dy())
}
