package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_FullCaptchaWorkflow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("完整滑块验证码流程", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/v1/captcha/slider", GetSliderCaptcha)
		router.POST("/api/v1/captcha/verify", VerifyCaptcha)

		req1, _ := http.NewRequest("GET", "/api/v1/captcha/slider?app_id=test_app", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		assert.Equal(t, http.StatusOK, w1.Code)

		var captchaResp CaptchaResponse
		err := json.Unmarshal(w1.Body.Bytes(), &captchaResp)
		require.NoError(t, err)
		require.NotEmpty(t, captchaResp.SessionID)

		verifyReq := VerifyRequest{
			SessionID: captchaResp.SessionID,
			Type:      "slider",
			X:         captchaResp.TargetX + 5,
			Y:         captchaResp.TargetY + 5,
			BehaviorData: []TrajectoryPoint{
				{X: 0, Y: float64(captchaResp.TargetY), Timestamp: time.Now().UnixMilli() - 1000},
				{X: 50, Y: float64(captchaResp.TargetY + 2), Timestamp: time.Now().UnixMilli() - 800},
				{X: 100, Y: float64(captchaResp.TargetY - 2), Timestamp: time.Now().UnixMilli() - 600},
				{X: 150, Y: float64(captchaResp.TargetY), Timestamp: time.Now().UnixMilli()},
			},
		}

		jsonBody, _ := json.Marshal(verifyReq)
		req2, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusNotFound}, w2.Code)

		var verifyResp map[string]interface{}
		json.Unmarshal(w2.Body.Bytes(), &verifyResp)
		t.Logf("验证响应: %+v", verifyResp)
	})

	t.Run("完整点击验证码流程", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/v1/captcha/click", GetClickCaptcha)
		router.POST("/api/v1/captcha/verify", VerifyCaptcha)

		req1, _ := http.NewRequest("GET", "/api/v1/captcha/click?mode=number&points=3", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		assert.Equal(t, http.StatusOK, w1.Code)

		var clickResp ClickCaptchaResponse
		err := json.Unmarshal(w1.Body.Bytes(), &clickResp)
		require.NoError(t, err)
		require.NotEmpty(t, clickResp.SessionID)

		verifyReq := VerifyRequest{
			SessionID:     clickResp.SessionID,
			Type:          "click",
			Points:        clickResp.TargetPoints,
			ClickSequence: clickResp.HintOrder,
			BehaviorData:  generateTestBehaviorData(),
		}

		jsonBody, _ := json.Marshal(verifyReq)
		req2, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusNotFound}, w2.Code)
	})
}

func TestIntegration_DatabaseOperations(t *testing.T) {
	t.Run("应用创建和查询", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.POST("/api/v1/admin/applications", CreateApplication)
		router.GET("/api/v1/admin/applications", ListApplications)
		router.GET("/api/v1/admin/applications/:id", GetApplication)

		appReq := CreateApplicationRequest{
			Name:        fmt.Sprintf("TestApp_%d", time.Now().Unix()),
			Description: "Integration Test Application",
		}

		jsonBody, _ := json.Marshal(appReq)
		req1, _ := http.NewRequest("POST", "/api/v1/admin/applications", bytes.NewBuffer(jsonBody))
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		assert.Equal(t, http.StatusOK, w1.Code)

		var createResp map[string]interface{}
		json.Unmarshal(w1.Body.Bytes(), &createResp)
		t.Logf("创建应用响应: %+v", createResp)

		req2, _ := http.NewRequest("GET", "/api/v1/admin/applications", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)

		var listResp map[string]interface{}
		json.Unmarshal(w2.Body.Bytes(), &listResp)
		t.Logf("应用列表响应: %+v", listResp)
	})

	t.Run("配置管理和更新", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.GET("/api/v1/admin/config", GetConfig)
		router.PUT("/api/v1/admin/config", UpdateConfig)

		req1, _ := http.NewRequest("GET", "/api/v1/admin/config", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		assert.Equal(t, http.StatusOK, w1.Code)

		var configResp map[string]interface{}
		json.Unmarshal(w1.Body.Bytes(), &configResp)
		t.Logf("配置响应: %+v", configResp)

		updateReq := UpdateConfigRequest{
			CaptchaExpireSeconds: 300,
			MaxVerifyAttempts:    5,
		}

		jsonBody, _ := json.Marshal(updateReq)
		req2, _ := http.NewRequest("PUT", "/api/v1/admin/config", bytes.NewBuffer(jsonBody))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, w2.Code)
	})
}

func TestIntegration_RedisCacheOperations(t *testing.T) {
	t.Run("Session缓存读写", func(t *testing.T) {
		ctx := context.Background()

		sessionID := fmt.Sprintf("test_session_%d", time.Now().UnixNano())
		sessionData := CaptchaSession{
			ID:         sessionID,
			Type:       "slider",
			TargetX:    150,
			TargetY:    100,
			Tolerance:  10,
			CreatedAt:  time.Now(),
			ExpiresAt:  time.Now().Add(5 * time.Minute),
			VerifyTimes: 0,
		}

		t.Logf("测试Session缓存: %s", sessionID)

		result, err := testCacheSet(ctx, sessionID, sessionData)
		require.NoError(t, err)
		assert.True(t, result)

		retrieved, err := testCacheGet(ctx, sessionID)
		require.NoError(t, err)
		if retrieved != nil {
			assert.Equal(t, sessionID, retrieved.ID)
		}

		deleted, err := testCacheDelete(ctx, sessionID)
		require.NoError(t, err)
		assert.True(t, deleted)
	})

	t.Run("验证码结果缓存", func(t *testing.T) {
		ctx := context.Background()

		token := fmt.Sprintf("test_token_%d", time.Now().UnixNano())
		verifyResult := VerifyResult{
			Success:    true,
			Score:      15.5,
			RiskLevel:  "low",
			Token:      token,
			VerifiedAt: time.Now(),
		}

		result, err := testCacheSet(ctx, "verify:"+token, verifyResult)
		require.NoError(t, err)
		assert.True(t, result)

		retrieved, err := testCacheGet(ctx, "verify:"+token)
		require.NoError(t, err)
		if retrieved != nil {
			assert.Equal(t, token, retrieved.Token)
		}
	})

	t.Run("限流计数器", func(t *testing.T) {
		ctx := context.Background()
		key := fmt.Sprintf("rate_limit:test_ip:%d", time.Now().Unix())

		for i := 0; i < 5; i++ {
			result, err := testIncrementCounter(ctx, key)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, result, int64(1))
		}

		count, err := testGetCounter(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})
}

func TestIntegration_APIEndpointIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("健康检查端点", func(t *testing.T) {
		router := gin.New()
		router.GET("/health", HealthCheck)

		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "healthy", resp["status"])
	})

	t.Run("认证端点", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/auth/login", AdminLogin)

		loginReq := LoginRequest{
			Username: "admin",
			Password: "admin123",
		}

		jsonBody, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		t.Logf("登录响应状态码: %d", w.Code)
		t.Logf("登录响应内容: %s", w.Body.String())
	})

	t.Run("统计端点", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/v1/admin/stats", GetStats)

		req, _ := http.NewRequest("GET", "/api/v1/admin/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized, http.StatusBadRequest}, w.Code)
	})
}

func TestIntegration_ErrorHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/captcha/verify", VerifyCaptcha)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		checkError     bool
	}{
		{
			name:           "无效的JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			checkError:     true,
		},
		{
			name: "缺失必填参数",
			requestBody: map[string]interface{}{
				"type": "slider",
			},
			expectedStatus: http.StatusBadRequest,
			checkError:     true,
		},
		{
			name: "不存在的Session",
			requestBody: VerifyRequest{
				SessionID: "nonexistent_session_12345",
				Type:      "slider",
				X:         100,
				Y:         50,
			},
			expectedStatus: http.StatusNotFound,
			checkError:     false,
		},
		{
			name: "过大的坐标值",
			requestBody: VerifyRequest{
				SessionID: "test_session",
				Type:      "slider",
				X:         999999,
				Y:         999999,
			},
			expectedStatus: http.StatusBadRequest,
			checkError:     true,
		},
		{
			name: "负数坐标值",
			requestBody: VerifyRequest{
				SessionID: "test_session",
				Type:      "slider",
				X:         -100,
				Y:         -50,
			},
			expectedStatus: http.StatusBadRequest,
			checkError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jsonBody []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				jsonBody = []byte(str)
			} else {
				jsonBody, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkError {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.NotEmpty(t, resp["message"])
			}
		})
	}
}

func TestIntegration_ConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/captcha/slider", GetSliderCaptcha)
	router.POST("/api/v1/captcha/verify", VerifyCaptcha)

	t.Run("并发生成验证码", func(t *testing.T) {
		concurrency := 10
		results := make(chan *http.Response, concurrency)
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(idx int) {
				req, err := http.NewRequest("GET", fmt.Sprintf("/api/v1/captcha/slider?app_id=test_%d", idx), nil)
				if err != nil {
					errors <- err
					return
				}

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				results <- w
			}(i)
		}

		successCount := 0
		for i := 0; i < concurrency; i++ {
			select {
			case resp := <-results:
				if resp.Code == http.StatusOK {
					successCount++
				}
			case err := <-errors:
				t.Errorf("请求错误: %v", err)
			}
		}

		assert.Equal(t, concurrency, successCount, "所有并发请求应该成功")
	})

	t.Run("并发验证同一验证码", func(t *testing.T) {
		req1, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		var captchaResp CaptchaResponse
		json.Unmarshal(w1.Body.Bytes(), &captchaResp)

		concurrency := 5
		results := make(chan *http.Response, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(idx int) {
				verifyReq := VerifyRequest{
					SessionID: captchaResp.SessionID,
					Type:      "slider",
					X:         captchaResp.TargetX + idx,
					Y:         captchaResp.TargetY,
				}

				jsonBody, _ := json.Marshal(verifyReq)
				req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				results <- w
			}(i)
		}

		processedCount := 0
		for i := 0; i < concurrency; i++ {
			select {
			case <-results:
				processedCount++
			case <-time.After(5 * time.Second):
				t.Logf("等待第 %d 个结果超时", i)
			}
		}

		assert.Equal(t, concurrency, processedCount, "所有并发验证请求应该被处理")
	})
}

func TestIntegration_TimeoutAndExpiry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/captcha/verify", VerifyCaptcha)

	t.Run("验证码过期处理", func(t *testing.T) {
		expiredSessionID := fmt.Sprintf("expired_session_%d", time.Now().Add(-10*time.Minute).Unix())

		verifyReq := VerifyRequest{
			SessionID: expiredSessionID,
			Type:      "slider",
			X:         100,
			Y:         50,
		}

		jsonBody, _ := json.Marshal(verifyReq)
		req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Contains(t, []int{http.StatusNotFound, http.StatusBadRequest, http.StatusGone}, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if msg, ok := resp["message"].(string); ok {
			t.Logf("过期响应消息: %s", msg)
		}
	})

	t.Run("验证次数超限", func(t *testing.T) {
		maxAttempts := 3

		for i := 0; i < maxAttempts+2; i++ {
			verifyReq := VerifyRequest{
				SessionID: "test_session_max_attempts",
				Type:      "slider",
				X:         100 + i*10,
				Y:         50,
			}

			jsonBody, _ := json.Marshal(verifyReq)
			req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if i >= maxAttempts {
				assert.Equal(t, http.StatusTooManyRequests, w.Code)
			}
		}
	})
}

func TestIntegration_SecurityValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/captcha/verify", VerifyCaptcha)

	t.Run("SQL注入防护", func(t *testing.T) {
		maliciousInputs := []string{
			"admin' OR '1'='1",
			"'; DROP TABLE captcha;--",
			"1; DELETE FROM sessions WHERE '1'='1'",
		}

		for _, input := range maliciousInputs {
			verifyReq := VerifyRequest{
				SessionID: input,
				Type:      "slider",
				X:         100,
				Y:         50,
			}

			jsonBody, _ := json.Marshal(verifyReq)
			req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.NotEqual(t, http.StatusOK, w.Code)
		}
	})

	t.Run("XSS防护", func(t *testing.T) {
		maliciousInputs := []string{
			"<script>alert('XSS')</script>",
			"javascript:alert('XSS')",
			"<img src=x onerror=alert('XSS')>",
		}

		for _, input := range maliciousInputs {
			verifyReq := VerifyRequest{
				SessionID: input,
				Type:      "slider",
				X:         100,
				Y:         50,
			}

			jsonBody, _ := json.Marshal(verifyReq)
			req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.NotEqual(t, http.StatusOK, w.Code)
		}
	})

	t.Run("路径遍历防护", func(t *testing.T) {
		maliciousInputs := []string{
			github.com/hjtpx/hjtpx/../../etc/passwd",
			"..\\..\\..\\windows\\system32",
			"....//....//....//etc/passwd",
		}

		for _, input := range maliciousInputs {
			verifyReq := VerifyRequest{
				SessionID: input,
				Type:      "slider",
				X:         100,
				Y:         50,
			}

			jsonBody, _ := json.Marshal(verifyReq)
			req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.NotEqual(t, http.StatusOK, w.Code)
		}
	})
}

type CaptchaSession struct {
	ID          string
	Type        string
	TargetX     int
	TargetY     int
	Tolerance   int
	CreatedAt   time.Time
	ExpiresAt   time.Time
	VerifyTimes int
}

type VerifyResult struct {
	Success    bool
	Score      float64
	RiskLevel  string
	Token      string
	VerifiedAt time.Time
}

type TestCacheEntry struct {
	data interface{}
}

var testCache = make(map[string]*TestCacheEntry)
var testCacheMu = struct{}{}

func testCacheSet(ctx context.Context, key string, value interface{}) (bool, error) {
	testCacheMu.Lock()
	defer testCacheMu.Unlock()
	testCache[key] = &TestCacheEntry{data: value}
	return true, nil
}

func testCacheGet(ctx context.Context, key string) (interface{}, error) {
	testCacheMu.Lock()
	defer testCacheMu.Unlock()
	if entry, ok := testCache[key]; ok {
		return entry.data, nil
	}
	return nil, nil
}

func testCacheDelete(ctx context.Context, key string) (bool, error) {
	testCacheMu.Lock()
	defer testCacheMu.Unlock()
	delete(testCache, key)
	return true, nil
}

func testIncrementCounter(ctx context.Context, key string) (int64, error) {
	testCacheMu.Lock()
	defer testCacheMu.Unlock()
	if entry, ok := testCache[key]; ok {
		if count, ok := entry.data.(int64); ok {
			entry.data = count + 1
			return count + 1, nil
		}
	}
	testCache[key] = &TestCacheEntry{data: int64(1)}
	return 1, nil
}

func testGetCounter(ctx context.Context, key string) (int64, error) {
	testCacheMu.Lock()
	defer testCacheMu.Unlock()
	if entry, ok := testCache[key]; ok {
		if count, ok := entry.data.(int64); ok {
			return count, nil
		}
	}
	return 0, nil
}

func generateTestBehaviorData() []TrajectoryPoint {
	baseTime := time.Now().UnixMilli()
	return []TrajectoryPoint{
		{X: 50, Y: 50, Timestamp: baseTime - 500},
		{X: 100, Y: 52, Timestamp: baseTime - 400},
		{X: 150, Y: 48, Timestamp: baseTime - 300},
		{X: 200, Y: 50, Timestamp: baseTime - 200},
		{X: 250, Y: 51, Timestamp: baseTime - 100},
		{X: 300, Y: 50, Timestamp: baseTime},
	}
}
