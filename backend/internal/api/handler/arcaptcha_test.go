package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/stretchr/testify/assert"
)

func init() {
	arGeneratorService = captcha.NewARGeneratorServiceSimple()
	arVerifierService = captcha.NewARVerifierService(nil, nil)
}

func TestCreateARCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectSession  bool
	}{
		{
			name:           "创建AR验证码-默认难度",
			requestBody:    `{}`,
			expectedStatus: http.StatusOK,
			expectSession:  true,
		},
		{
			name:           "创建AR验证码-简单难度",
			requestBody:    `{"difficulty": "easy"}`,
			expectedStatus: http.StatusOK,
			expectSession:  true,
		},
		{
			name:           "创建AR验证码-中等难度",
			requestBody:    `{"difficulty": "medium"}`,
			expectedStatus: http.StatusOK,
			expectSession:  true,
		},
		{
			name:           "创建AR验证码-困难难度",
			requestBody:    `{"difficulty": "hard"}`,
			expectedStatus: http.StatusOK,
			expectSession:  true,
		},
		{
			name:           "创建AR验证码-专家难度",
			requestBody:    `{"difficulty": "expert"}`,
			expectedStatus: http.StatusOK,
			expectSession:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/captcha/ar/create", CreateARCaptcha)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/captcha/ar/create", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectSession {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, float64(0), response["code"])
				assert.NotNil(t, response["data"])

				data := response["data"].(map[string]interface{})
				assert.NotEmpty(t, data["sessionID"])
				assert.NotNil(t, data["scene"])
				assert.NotNil(t, data["expiresIn"])
				assert.NotNil(t, data["expiresAt"])

				scene := data["scene"].(map[string]interface{})
				assert.NotEmpty(t, scene["targetShape"])
				assert.NotEmpty(t, scene["targetColor"])
				assert.NotEmpty(t, scene["objects"])
				assert.NotEmpty(t, scene["targetPosition"])
				assert.NotEmpty(t, scene["gridSize"])
				assert.NotEmpty(t, scene["difficulty"])
				assert.NotEmpty(t, scene["timeLimit"])
				assert.NotEmpty(t, scene["requiredGesture"])
			}
		})
	}
}

func TestVerifyARCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("验证AR验证码-缺少sessionID", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/v1/captcha/ar/verify", VerifyARCaptcha)

		requestBody := `{"scene": {}, "userGesture": "tap", "placedObjectID": 0}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/captcha/ar/verify", bytes.NewBufferString(requestBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEqual(t, float64(0), response["code"])
	})

	t.Run("验证AR验证码-无效sessionID", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/v1/captcha/ar/verify", VerifyARCaptcha)

		requestBody := `{"sessionID": "invalid-session", "scene": {}, "userGesture": "tap", "placedObjectID": 0}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/captcha/ar/verify", bytes.NewBufferString(requestBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(500), response["code"])
	})
}

func TestGetARCaptchaStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("获取AR验证码状态-缺少sessionID", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/captcha/ar/status/:sessionID", GetARCaptchaStatus)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/captcha/ar/status/", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEqual(t, float64(0), response["code"])
	})

	t.Run("获取AR验证码状态-无效sessionID", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/captcha/ar/status/:sessionID", GetARCaptchaStatus)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/captcha/ar/status/invalid-session", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(404), response["code"])
	})
}

func TestCheckARCaptchaValid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("检查AR验证码有效性-缺少sessionID", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/captcha/ar/check/:sessionID", CheckARCaptchaValid)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/captcha/ar/check/", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEqual(t, float64(0), response["code"])
	})

	t.Run("检查AR验证码有效性-无效sessionID", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/captcha/ar/check/:sessionID", CheckARCaptchaValid)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/captcha/ar/check/invalid-session", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(0), response["code"])

		data := response["data"].(map[string]interface{})
		assert.False(t, data["valid"].(bool))
		assert.Equal(t, "会话不存在", data["message"])
	})
}

func TestGetWebXRSupport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("获取WebXR支持信息", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/captcha/ar/webxr-support", GetWebXRSupport)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/captcha/ar/webxr-support", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(0), response["code"])

		data := response["data"].(map[string]interface{})
		assert.True(t, data["supportsWebXR"].(bool))
		assert.True(t, data["supportsAR"].(bool))
		assert.True(t, data["supportsVR"].(bool))
		assert.NotNil(t, data["requiredFeatures"])
		assert.NotNil(t, data["recommendedFeatures"])
		assert.NotNil(t, data["capabilities"])
	})
}

func TestARGeneratorService(t *testing.T) {
	tests := []struct {
		name       string
		difficulty string
	}{
		{"生成简单难度场景", "easy"},
		{"生成中等难度场景", "medium"},
		{"生成困难难度场景", "hard"},
		{"生成专家难度场景", "expert"},
		{"生成默认难度场景", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := captcha.NewARGeneratorServiceSimple()
			req := &captcha.CreateARRequest{
				Difficulty: tt.difficulty,
			}

			result, err := generator.Create(nil, req)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.NotEmpty(t, result.SessionID)
			assert.NotNil(t, result.Scene)
			assert.Greater(t, result.ExpiresIn, int64(0))
			assert.Greater(t, result.ExpiresAt, int64(0))

			scene := result.Scene
			assert.NotEmpty(t, scene.TargetShape)
			assert.NotEmpty(t, scene.TargetColor)
			assert.NotNil(t, scene.Objects)
			assert.NotNil(t, scene.TargetPosition)
			assert.Greater(t, scene.GridSize, 0)
			assert.NotEmpty(t, scene.Difficulty)
			assert.Greater(t, scene.TimeLimit, 0)
			assert.NotEmpty(t, scene.RequiredGesture)
			assert.NotNil(t, scene.Environment)

			var targetCount int
			for _, obj := range scene.Objects {
				if obj.IsTarget {
					targetCount++
					assert.Equal(t, scene.TargetColor, obj.Color)
				}
			}
			assert.Equal(t, 1, targetCount)
		})
	}
}

func TestARVerifierService(t *testing.T) {
	t.Run("验证器验证-无效session", func(t *testing.T) {
		verifier := captcha.NewARVerifierService(nil, nil)
		req := &captcha.ARVerifyRequest{
			SessionID: "invalid",
		}

		result, err := verifier.Verify(nil, req)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("检查会话有效性-无效session", func(t *testing.T) {
		verifier := captcha.NewARVerifierService(nil, nil)
		valid, message := verifier.CheckSessionValid(nil, "invalid")

		assert.False(t, valid)
		assert.Equal(t, "会话不存在", message)
	})
}
